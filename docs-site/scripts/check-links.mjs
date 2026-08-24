#!/usr/bin/env node
/*
 * Broken-link gate for the docs pipeline (T2 / #352).
 *
 * Standalone CLI — NOT an inline build assertion — so T5/#355 can invoke it from
 * CI:  `node scripts/check-links.mjs`  (wired as `npm run check:links`). Exits
 * non-zero and prints `file:line  ->  target  (reason)` for every dead internal
 * link, so a renamed or deleted guide (or a heading whose slug moved) fails the
 * build instead of silently 404ing on the live site.
 *
 * WHAT IT CHECKS (matches the render.jsx `a`-renderer rewrite exactly):
 *   - Intra-guide relative `.md` links (`using.md`, `./running.md`, optionally with
 *     a `#fragment`) must point at a guide that EXISTS in that version, and the
 *     fragment must match a real heading id in the target guide.
 *   - Anchor-only links (`#frag`) must match a heading id in their own guide.
 *   - External `http(s)`/`mailto:` links are left alone.
 *   - `../adr/<file>.md` cross-references (which render.jsx rewrites to a GitHub
 *     blob URL) must name an ADR file that EXISTS in docs/adr/, so a renamed or
 *     deleted ADR fails the build instead of shipping a dead blob link (#428).
 *     The `#fragment` is NOT checked: ADRs render on GitHub, whose heading-anchor
 *     algorithm differs from ours, so only file reachability is meaningful here.
 *   - Other relative links that escape the guides dir (`../../deploy/...`, `*.go`,
 *     bare directories) are repo cross-references with no home on the docs site —
 *     out of scope for this gate, so they are not flagged.
 *
 * PER-VERSION: the checker loops over a version list and validates each version's
 * link graph independently against ONLY that version's guide + anchor inventory —
 * a link valid in v0.9.2 may be dead in v0.9.1. Today source-resolution exposes a
 * single version ("main") read from the working-tree guides; when Tv/#351 lands
 * git-ref versions, `loadVersions()` is the one function to extend (read each ref's
 * `docs/guides/*.md`), and the per-version loop below is unchanged.
 *
 * Anchor ids come from `github-slugger` (the shared algorithm — see slug.ts and
 * render.jsx): lowercase, strip punctuation, spaces -> "-", de-dup collisions with
 * a numeric suffix. Every heading (h1..h6) is slugged in source order so the de-dup
 * counter matches the renderer; fenced code blocks are skipped so a `# comment`
 * inside a ```sh block is never read as a heading.
 */
import GithubSlugger from "github-slugger";
import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, relative, resolve } from "node:path";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");
const GUIDES_DIR = join(REPO_ROOT, "docs", "guides");
const ADR_DIR = join(REPO_ROOT, "docs", "adr");

/**
 * ADR filenames present in docs/adr/, for validating `../adr/<file>.md` cross-refs.
 * Read once from the working tree — correct while there is a single working-tree
 * version (like the guide inventory in loadVersions). When Tv's git-ref versions
 * land, this moves into the per-version seam so each ref checks its own ADR set.
 */
const ADR_FILES = new Set(
  readdirSync(ADR_DIR).filter((name) => name.endsWith(".md")),
);

/** A guide resolved at one version: slug + its raw markdown + repo-relative path. */
/**
 * Load every version and its guide set.
 *
 * TODAY: one version, "main", read from the working-tree guides — mirroring
 * source-resolution.ts (`resolveSources` / `listVersions`), which this gate cannot
 * import directly because that module reads Astro's `astro:content` collection
 * (only available inside the Astro build). This is the SEAM to extend when Tv/#351
 * makes versions real: emit one entry per git ref, each `guides` map built from
 * `git show <ref>:docs/guides/<file>`. The per-version checking below needs no edits.
 *
 * @returns {{ version: string, guides: Map<string, { markdown: string, file: string }> }[]}
 */
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

/**
 * Every heading id in a guide, via the shared github-slugger algorithm. Slugs every
 * heading (all levels) in source order so the de-dup counter matches render.jsx;
 * fenced code blocks are skipped. Returns a Set of anchor ids (no leading `#`).
 */
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

/**
 * Every markdown link target in a doc, with its 1-based line number. Skips fenced
 * code blocks so a `](x.md)` inside a code sample is not treated as a link.
 * @returns {{ line: number, target: string }[]}
 */
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
    // [text](target) — target stops at whitespace (drops any `"title"`) or `)`.
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
// intra-guide relative link: optional `./`, a bare slug, `.md`, optional `#frag`.
const INTRA_GUIDE = /^\.?\/?([a-z0-9][a-z0-9-]*)\.md(?:#(.+))?$/i;
// ADR cross-ref: `../adr/<file>.md` (flat filename, no nested path), optional `#frag`.
// Mirrors render.jsx's ADR_XREF exactly so gate and renderer classify links identically.
const ADR_XREF = /^\.\.\/adr\/([^#?/]+\.md)(?:#(.+))?$/i;

/**
 * Check one link target against a version's guide + anchor inventory.
 * @returns {string|null} a failure reason, or null if the link is fine / out of scope.
 */
function checkTarget(target, currentSlug, guides, anchorsBySlug) {
  if (EXTERNAL.test(target) || MAILTO.test(target)) return null; // external — untouched

  if (target.startsWith("#")) {
    const frag = target.slice(1);
    if (!frag) return null;
    return anchorsBySlug.get(currentSlug).has(frag)
      ? null
      : `anchor "#${frag}" not found in ${currentSlug}`;
  }

  const adr = ADR_XREF.exec(target);
  if (adr) {
    // render.jsx rewrites this to a GitHub blob URL; guard the file still exists.
    return ADR_FILES.has(adr[1])
      ? null
      : `ADR "${adr[1]}" does not exist in docs/adr/`;
  }

  const m = INTRA_GUIDE.exec(target);
  if (!m) return null; // ../../deploy/, *.go, directories — out of scope

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
    // Per-version anchor inventory, built once from THIS version's guides only.
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
