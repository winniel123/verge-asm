#!/usr/bin/env node
import GithubSlugger from "github-slugger";
import { execFileSync } from "node:child_process";
import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = resolve(SCRIPT_DIR, "..", "..");
const GUIDES_DIR = "docs/guides";
const ADR_DIR = "docs/adr";
const DEFAULT_VERSION = "main";
const LATEST_VERSION = "latest";
const SEMVER_TAG = /^v(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/;

function git(root, args) {
  return execFileSync("git", args, {
    cwd: root,
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  }).replace(/\r?\n$/, "");
}

function lsTree(root, ref, dir) {
  let listing;
  try {
    listing = git(root, ["ls-tree", "-r", "--name-only", ref, dir]);
  } catch {
    return [];
  }
  return listing
    .split("\n")
    .map((f) => f.trim())
    .filter((f) => f.endsWith(".md"));
}

function compareTagsDesc(a, b) {
  if (a.major !== b.major) return b.major - a.major;
  if (a.minor !== b.minor) return b.minor - a.minor;
  if (a.patch !== b.patch) return b.patch - a.patch;
  if (a.prerelease === null && b.prerelease !== null) return -1;
  if (a.prerelease !== null && b.prerelease === null) return 1;
  if (a.prerelease === null && b.prerelease === null) return 0;
  return b.prerelease.localeCompare(a.prerelease);
}

function publishableTags(root) {
  let out;
  try {
    out = git(root, ["tag", "-l", "v*"]);
  } catch {
    return [];
  }
  const tags = [];
  for (const line of out.split("\n")) {
    const raw = line.trim();
    const m = SEMVER_TAG.exec(raw);
    if (!m) continue;
    if (lsTree(root, raw, GUIDES_DIR).length === 0) continue;
    tags.push({
      raw,
      major: Number(m[1]),
      minor: Number(m[2]),
      patch: Number(m[3]),
      prerelease: m[4] ?? null,
    });
  }
  return tags.sort(compareTagsDesc);
}

// mirrors source-resolution.ts, unimportable here: it reads Astro's astro:content (#1395)
function versionPlan(root) {
  const tags = publishableTags(root);
  const newestStable = tags.find((t) => t.prerelease === null) ?? null;
  const plan = [
    { version: LATEST_VERSION, ref: newestStable ? newestStable.raw : DEFAULT_VERSION },
  ];
  for (const t of tags) plan.push({ version: t.raw, ref: t.raw });
  plan.push({ version: DEFAULT_VERSION, ref: DEFAULT_VERSION });
  return plan;
}

function markdownFilesInDir(root, dir) {
  try {
    return readdirSync(join(root, dir))
      .filter((name) => name.endsWith(".md"))
      .map((name) => `${dir}/${name}`);
  } catch {
    return [];
  }
}

function guidesOf(files, read) {
  const guides = new Map();
  for (const file of files) {
    const slug = file.replace(/^.*\//, "").replace(/\.md$/, "");
    guides.set(slug, { markdown: read(file), file });
  }
  return guides;
}

function adrNamesOf(files) {
  return new Set(files.map((f) => f.replace(/^.*\//, "")));
}

export function loadVersions(root = DEFAULT_ROOT) {
  return versionPlan(root).map(({ version, ref }) => {
    // CI builds a PR as a merge commit, so `git show main:` misses the PR's own edits (#1395)
    if (ref === DEFAULT_VERSION) {
      return {
        version,
        guides: guidesOf(markdownFilesInDir(root, GUIDES_DIR), (f) =>
          readFileSync(join(root, f), "utf8"),
        ),
        adrFiles: adrNamesOf(markdownFilesInDir(root, ADR_DIR)),
      };
    }
    return {
      version,
      guides: guidesOf(lsTree(root, ref, GUIDES_DIR), (f) => git(root, ["show", `${ref}:${f}`])),
      adrFiles: adrNamesOf(lsTree(root, ref, ADR_DIR)),
    };
  });
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

function checkTarget(target, currentSlug, guides, anchorsBySlug, adrFiles) {
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
    return adrFiles.has(adr[1]) ? null : `ADR "${adr[1]}" does not exist in docs/adr/`;
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

export function findFailures(versions) {
  const failures = [];
  for (const { version, guides, adrFiles } of versions) {
    const anchorsBySlug = new Map();
    for (const [slug, { markdown }] of guides) {
      anchorsBySlug.set(slug, collectAnchors(markdown));
    }

    for (const [slug, { markdown, file }] of guides) {
      for (const { line, target } of scanLinks(markdown)) {
        const reason = checkTarget(target, slug, guides, anchorsBySlug, adrFiles);
        if (reason) failures.push({ version, file, line, target, reason });
      }
    }
  }
  return failures;
}

function main() {
  const versions = loadVersions();
  const failures = findFailures(versions);

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

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
