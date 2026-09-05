#!/usr/bin/env node
import GithubSlugger from "github-slugger";
import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, relative, resolve } from "node:path";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");
const GUIDES_DIR = join(REPO_ROOT, "docs", "guides");
const ADR_DIR = join(REPO_ROOT, "docs", "adr");

const ADR_FILES = new Set(
  readdirSync(ADR_DIR).filter((name) => name.endsWith(".md")),
);

function loadVersions() {
  const guides = new Map();
  for (const name of readdirSync(GUIDES_DIR)) {
    if (!name.endsWith(".md")) continue;
    const slug = name.slice(0, -3);
    const abs = join(GUIDES_DIR, name);
    guides.set(slug, {
      markdown: readFileSync(abs, "utf8"),
      file: relative(REPO_ROOT, abs).replace(/\\/g, "/"),
    });
  }
  return [{ version: "main", guides }];
}

// slugging a subset diverges the de-dup counter from the renderer (docs-site/PIPELINE.md)
function collectAnchors(markdown) {
  const slugger = new GithubSlugger();
  const ids = new Set();
  let inFence = false;
  for (const line of markdown.split(/\r?\n/)) {
    if (/^\s*(```|~~~)/.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    const m = /^(#{1,6})\s+(.*?)\s*#*\s*$/.exec(line);
    if (!m) continue;
    const label = m[2]
      .replace(/`([^`]+)`/g, "$1")
      .replace(/\*\*([^*]+)\*\*/g, "$1")
      .replace(/\*([^*]+)\*/g, "$1")
      .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
      .trim();
    ids.add(slugger.slug(label));
  }
  return ids;
}

function scanLinks(markdown) {
  const out = [];
  let inFence = false;
  const lines = markdown.split(/\r?\n/);
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (/^\s*(```|~~~)/.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    const re = /\]\(([^)\s]+)/g;
    let m;
    while ((m = re.exec(line)) !== null) {
      out.push({ line: i + 1, target: m[1] });
    }
  }
  return out;
}

const EXTERNAL = /^([a-z][a-z0-9+.-]*:)?\/\//i;
const MAILTO = /^mailto:/i;
const INTRA_GUIDE = /^\.?\/?([a-z0-9][a-z0-9-]*)\.md(?:#(.+))?$/i;
// mirrors render.jsx's ADR_XREF, so gate and renderer cannot classify a link differently
const ADR_XREF = /^\.\.\/adr\/([^#?/]+\.md)(?:#(.+))?$/i;

function checkTarget(target, currentSlug, guides, anchorsBySlug) {
  if (EXTERNAL.test(target) || MAILTO.test(target)) return null;

  if (target.startsWith("#")) {
    const frag = target.slice(1);
    if (!frag) return null;
    return anchorsBySlug.get(currentSlug).has(frag)
      ? null
      : `anchor "#${frag}" not found in ${currentSlug}`;
  }

  const adr = ADR_XREF.exec(target);
  if (adr) {
    // an ADR is not a docs-site page, so this gate holds no heading inventory for its fragment
    return ADR_FILES.has(adr[1])
      ? null
      : `ADR "${adr[1]}" does not exist in docs/adr/`;
  }

  const m = INTRA_GUIDE.exec(target);
  // only docs/guides/*.md becomes a page, so any other relative target is out of this gate
  if (!m) return null;

  const slug = m[1];
  const frag = m[2];
  if (!guides.has(slug)) return `guide "${slug}" does not exist in this version`;
  if (frag && !anchorsBySlug.get(slug).has(frag)) {
    return `anchor "#${frag}" not found in ${slug}`;
  }
  return null;
}

function main() {
  const versions = loadVersions();
  const failures = [];

  for (const { version, guides } of versions) {
    const anchorsBySlug = new Map();
    for (const [slug, { markdown }] of guides) {
      anchorsBySlug.set(slug, collectAnchors(markdown));
    }

    for (const [slug, { markdown, file }] of guides) {
      for (const { line, target } of scanLinks(markdown)) {
        const reason = checkTarget(target, slug, guides, anchorsBySlug);
        if (reason) failures.push({ version, file, line, target, reason });
      }
    }
  }

  if (failures.length === 0) {
    const n = versions.reduce((a, v) => a + v.guides.size, 0);
    console.log(
      `check:links OK — ${n} guide(s) across ${versions.length} version(s), no broken internal links.`,
    );
    return;
  }

  console.error(`check:links FAILED — ${failures.length} broken internal link(s):\n`);
  for (const f of failures) {
    console.error(`  [${f.version}] ${f.file}:${f.line}  ->  ${f.target}  (${f.reason})`);
  }
  console.error("");
  process.exit(1);
}

main();
