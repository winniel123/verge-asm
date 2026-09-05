#!/usr/bin/env node
// a design-system token or component change can silently re-lay-out a prose page, unseen elsewhere
import { createServer } from "node:http";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve, extname, sep } from "node:path";
import { chromium } from "playwright";
import { PNG } from "pngjs";
import pixelmatch from "pixelmatch";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const SITE_ROOT = resolve(SCRIPT_DIR, "..");
const DIST_DIR = join(SITE_ROOT, "dist");
const BASELINE = join(SITE_ROOT, "tests", "baseline", "docs.png");
const DIFF_OUT = join(SITE_ROOT, "tests", "diff", "docs.diff.png");
const ACTUAL_OUT = join(SITE_ROOT, "tests", "diff", "docs.actual.png");

const TARGET_PATH = "/main/using/";
const VIEWPORT = { width: 1280, height: 900 };
// antialiasing differs between two otherwise identical captures, so the floor is not zero
const FAIL_RATIO = 0.005;
const PIXEL_THRESHOLD = 0.1;

const UPDATE = process.argv.includes("--update");

const MIME = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".mjs": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".woff2": "font/woff2",
  ".woff": "font/woff",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".ico": "image/x-icon",
};

function serveDist() {
  return new Promise((resolveServer) => {
    const server = createServer(async (req, res) => {
      try {
        let urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
        if (urlPath.endsWith("/")) urlPath += "index.html";
        let filePath = join(DIST_DIR, urlPath);
        if (!extname(filePath)) filePath = join(filePath, "index.html");
        const resolved = resolve(filePath);
        if (resolved !== DIST_DIR && !resolved.startsWith(DIST_DIR + sep)) {
          res.writeHead(403, { "content-type": "text/plain" });
          res.end("forbidden");
          return;
        }
        const body = await readFile(resolved);
        res.writeHead(200, { "content-type": MIME[extname(filePath)] || "application/octet-stream" });
        res.end(body);
      } catch {
        res.writeHead(404, { "content-type": "text/plain" });
        res.end("not found");
      }
    });
    server.listen(0, "127.0.0.1", () => resolveServer(server));
  });
}

async function capture(url) {
  // the gate runs inside the pinned Playwright container, where chromium's sandbox is unavailable
  const browser = await chromium.launch({ headless: true, args: ["--no-sandbox"] });
  try {
    const context = await browser.newContext({
      viewport: VIEWPORT,
      deviceScaleFactor: 1,
      colorScheme: "light",
    });
    const page = await context.newPage();
    await page.goto(url, { waitUntil: "networkidle" });
    // an animation or a blinking caret puts a different frame in each run
    await page.addStyleTag({
      content: `*,*::before,*::after{transition:none!important;animation:none!important;caret-color:transparent!important;scroll-behavior:auto!important}`,
    });
    // a web font arriving after networkidle re-flows the page and moves the capture
    await page.evaluate(async () => {
      if (document.fonts && document.fonts.ready) await document.fonts.ready;
    });
    await page.waitForTimeout(300);
    const buf = await page.screenshot({ fullPage: true, animations: "disabled" });
    return PNG.sync.read(buf);
  } finally {
    await browser.close();
  }
}

async function main() {
  if (!existsSync(DIST_DIR)) {
    console.error("check:screenshot FAILED — ./dist not found. Run `npm run build` first.");
    process.exit(1);
  }

  const server = await serveDist();
  const { port } = server.address();
  const url = `http://127.0.0.1:${port}${TARGET_PATH}`;

  let actual;
  try {
    actual = await capture(url);
  } finally {
    server.close();
  }

  if (UPDATE) {
    await mkdir(dirname(BASELINE), { recursive: true });
    await writeFile(BASELINE, PNG.sync.write(actual));
    console.log(
      `check:screenshot — baseline written (${actual.width}x${actual.height}) to ${BASELINE}`,
    );
    return;
  }

  if (!existsSync(BASELINE)) {
    console.error(
      "check:screenshot FAILED — no baseline. Generate one with `npm run screenshot:update` and commit it.",
    );
    process.exit(1);
  }

  const baseline = PNG.sync.read(await readFile(BASELINE));

  // a dimension mismatch makes pixelmatch throw, which reads as a crash not a layout regression
  if (baseline.width !== actual.width || baseline.height !== actual.height) {
    await mkdir(dirname(ACTUAL_OUT), { recursive: true });
    await writeFile(ACTUAL_OUT, PNG.sync.write(actual));
    console.error(
      `check:screenshot FAILED — dimensions drifted: baseline ${baseline.width}x${baseline.height}, ` +
        `now ${actual.width}x${actual.height}. Layout changed height/width.\n` +
        `  actual capture written to ${ACTUAL_OUT}\n` +
        `  If this change is intended, refresh with \`npm run screenshot:update\` and commit.`,
    );
    process.exit(1);
  }

  const { width, height } = baseline;
  const diff = new PNG({ width, height });
  const changed = pixelmatch(baseline.data, actual.data, diff.data, width, height, {
    threshold: PIXEL_THRESHOLD,
  });
  const ratio = changed / (width * height);

  if (ratio > FAIL_RATIO) {
    await mkdir(dirname(DIFF_OUT), { recursive: true });
    await writeFile(DIFF_OUT, PNG.sync.write(diff));
    await writeFile(ACTUAL_OUT, PNG.sync.write(actual));
    console.error(
      `check:screenshot FAILED — ${changed} px differ (${(ratio * 100).toFixed(3)}% > ` +
        `${(FAIL_RATIO * 100).toFixed(3)}% threshold). Design drift on ${TARGET_PATH}.\n` +
        `  diff written to   ${DIFF_OUT}\n` +
        `  actual written to ${ACTUAL_OUT}\n` +
        `  If this change is intended, refresh with \`npm run screenshot:update\` and commit.`,
    );
    process.exit(1);
  }

  console.log(
    `check:screenshot OK — ${changed} px differ (${(ratio * 100).toFixed(3)}% ≤ ` +
      `${(FAIL_RATIO * 100).toFixed(3)}% threshold) on ${TARGET_PATH} (${width}x${height}).`,
  );
}

main().catch((err) => {
  console.error("check:screenshot FAILED —", err?.stack || err);
  process.exit(1);
});
