#!/usr/bin/env node
/*
 * Screenshot / design-drift gate for the docs pipeline (T5 / #355).
 *
 * WHY: the guides render through the design system (tokens + DS components). When a
 * token or component changes shape — spacing, type scale, hierarchy, rail widths —
 * the prose pages silently re-lay-out. This gate renders a built docs page in a real
 * browser and pixel-diffs it against a COMMITTED baseline, so a design change that
 * moves the layout fails CI until the baseline is intentionally refreshed.
 *
 * It is NOT a diff against design-system/screenshots/docs.jpg — that capture is a
 * downscaled (~917px) JPG, useful as the human design target but useless as a
 * byte-baseline. We generate our own full-resolution PNG baseline here.
 *
 * USAGE
 *   node scripts/screenshot-check.mjs            # compare against baseline, exit 1 on drift
 *   node scripts/screenshot-check.mjs --update    # (re)write the baseline PNG, exit 0
 *
 * PREREQ: `npm run build` has produced ./dist. Chromium via `npx playwright install
 * chromium`. Deterministic by construction: fixed viewport, forced light theme,
 * animations/transitions/carets disabled, web-fonts awaited before capture.
 */
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

// The page under test: the Quick start guide — richest layout (rails, TOC, tables,
// callouts, code blocks, accordion), so it exercises the most DS surface.
const TARGET_PATH = "/main/using/";
const VIEWPORT = { width: 1280, height: 900 };
// Fail when more than this fraction of pixels differ. A spacing/hierarchy regression
// shifts thousands of pixels (well above this); sub-pixel AA noise stays under it.
const FAIL_RATIO = 0.005; // 0.5%
// pixelmatch per-pixel colour tolerance (0 strict … 1 loose).
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

/** Minimal static server over dist/ with Astro directory-index resolution. */
function serveDist() {
  return new Promise((resolveServer) => {
    const server = createServer(async (req, res) => {
      try {
        let urlPath = decodeURIComponent((req.url || "/").split("?")[0]);
        if (urlPath.endsWith("/")) urlPath += "index.html";
        let filePath = join(DIST_DIR, urlPath);
        if (!extname(filePath)) filePath = join(filePath, "index.html");
        // Path-injection guard: a decoded request path can contain ".." segments
        // that escape dist/. Resolve and require the result to stay within DIST_DIR
        // before touching the filesystem.
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
  // --no-sandbox: the gate runs inside the pinned Playwright container (CI and local
  // baseline generation alike), where chromium launches as root and the sandbox is
  // unavailable. It does not affect rendering, so the baseline stays reproducible.
  const browser = await chromium.launch({ headless: true, args: ["--no-sandbox"] });
  try {
    const context = await browser.newContext({
      viewport: VIEWPORT,
      deviceScaleFactor: 1,
      colorScheme: "light", // deterministic theme regardless of the CI runner
    });
    const page = await context.newPage();
    await page.goto(url, { waitUntil: "networkidle" });
    // Kill anything that would make two captures differ: transitions, animations,
    // the text caret, and smooth-scroll. Then wait for web-fonts to load so glyph
    // metrics are final before we measure layout.
    await page.addStyleTag({
      content: `*,*::before,*::after{transition:none!important;animation:none!important;caret-color:transparent!important;scroll-behavior:auto!important}`,
    });
    await page.evaluate(async () => {
      if (document.fonts && document.fonts.ready) await document.fonts.ready;
    });
    await page.waitForTimeout(300); // final settle after hydration
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

  // A changed page HEIGHT or WIDTH is itself a layout regression — report it clearly
  // instead of letting pixelmatch throw on mismatched dimensions.
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
