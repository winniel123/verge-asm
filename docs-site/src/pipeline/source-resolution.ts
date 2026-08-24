/*
 * PIPELINE STAGE 1 of 3 — source-resolution.  OWNER: Tv (#351).
 *
 * Contract (do not change the shape — T2/T3 build against it):
 *   resolveSources(version?) -> Promise<Source[]>
 *   listVersions()           -> VersionOption[]
 *
 * A `Source` is one guide at one version: { version, slug, rawMarkdown, frontmatter }.
 * This stage now iterates git refs and emits one Source per (ref x guide), WITHOUT
 * touching render.jsx (T2) or nav-build.ts (T3) — they consume only the Source /
 * VersionOption types below. That is the seam. See docs-site/PIPELINE.md.
 *
 * ─────────────────────────────────────────────────────────────────────────────
 * VERSION / TAG POLICY (Tv decision — recorded here per PIPELINE.md's "no new ADR"
 * rule; ADR-0115 already covers the model, and parallel ADRs would collide):
 *
 *   • Release glob ............ tags matching `v*` that are strict semver:
 *                               /^v\d+\.\d+\.\d+(?:-<prerelease>)?$/. A `v*` tag
 *                               that is not semver-shaped (e.g. `vintage`) is
 *                               ignored, not published.
 *   • Pre-release tags ........ a semver tag WITH a prerelease segment
 *                               (`-rc`, `-alpha`, `-beta`, `-fixture`, …) IS
 *                               rendered as its own /<tag>/* version tree (so RC
 *                               docs are readable) but is EXCLUDED from `latest`.
 *                               Rationale: readers may want pre-release docs, but
 *                               "current" must always point at a shipped, stable
 *                               release.
 *   • `main` .................. always published as the `dev` version (moving branch).
 *   • `latest` ................ alias tree that tracks the highest STABLE semver
 *                               tag; falls back to `main` when there is no stable
 *                               release (true today — zero tags). Carries the
 *                               "current" badge in the manifest.
 *   • Canonical URL ........... every page's canonical points at the `latest`
 *                               alias (see canonicalPath()); wired in the route.
 *
 * The `latest` alias is BUILT as a real /latest/* tree today. T5 (#355) owns the
 * deploy/redirect mechanism and MAY swap this for host-level redirects from
 * /latest/* to the resolved release; nothing else in this stage need change.
 * ─────────────────────────────────────────────────────────────────────────────
 */
import { execFileSync } from "node:child_process";
import { getCollection } from "astro:content";

export interface Frontmatter {
  title?: string;
  section?: string;
  order?: number;
  description?: string;
}

/** One guide, resolved at one version. The unit every downstream stage consumes. */
export interface Source {
  version: string;
  slug: string;
  rawMarkdown: string;
  frontmatter: Frontmatter;
}

/**
 * Version-picker option. Mirrors `VersionOption` from
 * design-system/components/navigation/VersionSelect.d.ts so the manifest T4 (#354)
 * consumes needs no adaptation: { value, tag? } with tag "current" on the latest
 * release and "dev" on the moving `main` branch.
 */
export interface VersionOption {
  value: string;
  tag?: string;
}

/** The moving branch, published as the `dev` version. */
export const DEFAULT_VERSION = "main";
/** The alias that tracks the highest stable release (or `main` when none exists). */
export const LATEST_VERSION = "latest";

/** Guides live at the repo root, one level up from this Astro project. */
const GUIDES_DIR = "docs/guides";
/** A `v*` tag is a release iff it is strict semver, with an optional prerelease. */
const SEMVER_TAG = /^v(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/;

// ── git plumbing ────────────────────────────────────────────────────────────
// Read-only. We never check anything out; `git show`/`git ls-tree` read straight
// from the object store, so parallel builds and a dirty working tree are safe.

let repoRootCache: string | null = null;
function repoRoot(): string {
  if (repoRootCache) return repoRootCache;
  repoRootCache = git(["rev-parse", "--show-toplevel"]);
  return repoRootCache;
}

function git(args: string[], cwd?: string): string {
  return execFileSync("git", args, {
    cwd: cwd ?? process.cwd(),
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  }).replace(/\r?\n$/, "");
}

interface ParsedTag {
  raw: string;
  major: number;
  minor: number;
  patch: number;
  prerelease: string | null;
}

/** All strict-semver `v*` tags, unfiltered, newest-first. */
function allSemverTags(): ParsedTag[] {
  let out: string;
  try {
    out = git(["tag", "-l", "v*"], repoRoot());
  } catch {
    return []; // no git / no tags: `latest` falls back to `main`.
  }
  const tags: ParsedTag[] = [];
  for (const line of out.split("\n")) {
    const raw = line.trim();
    if (!raw) continue;
    const m = SEMVER_TAG.exec(raw);
    if (!m) continue; // `v*` but not semver (e.g. `vintage`) — not a release.
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

/** Descending semver order; a stable release outranks its own prereleases. */
function compareTagsDesc(a: ParsedTag, b: ParsedTag): number {
  if (a.major !== b.major) return b.major - a.major;
  if (a.minor !== b.minor) return b.minor - a.minor;
  if (a.patch !== b.patch) return b.patch - a.patch;
  // Same x.y.z: a release (no prerelease) sorts above any prerelease of it.
  if (a.prerelease === null && b.prerelease !== null) return -1;
  if (a.prerelease !== null && b.prerelease === null) return 1;
  if (a.prerelease === null && b.prerelease === null) return 0;
  return (b.prerelease as string).localeCompare(a.prerelease as string);
}

/** Does this ref carry a `docs/guides/` tree with at least one `.md`? */
function hasGuides(ref: string): boolean {
  try {
    const listing = git(["ls-tree", "-r", "--name-only", ref, GUIDES_DIR], repoRoot());
    return listing.split("\n").some((f) => f.trim().endsWith(".md"));
  } catch {
    return false;
  }
}

/**
 * Publishable release tags: strict-semver `v*` tags that actually carry guides.
 * Memoised for the whole build (tags don't move mid-build) so we shell out once
 * and log each guide-less skip exactly once — not per page. A `v*` tag without
 * `docs/guides/` is dropped from BOTH the manifest and the routes, logged, never
 * fatal (acceptance: missing-docs/guides refs are skipped, not silent).
 */
let publishableCache: ParsedTag[] | null = null;
function publishableTags(): ParsedTag[] {
  if (publishableCache) return publishableCache;
  const kept: ParsedTag[] = [];
  for (const t of allSemverTags()) {
    if (hasGuides(t.raw)) {
      kept.push(t);
    } else {
      console.warn(
        `[source-resolution] release tag "${t.raw}" has no ${GUIDES_DIR}/ — skipped from manifest + routes, not fatal.`,
      );
    }
  }
  publishableCache = kept;
  return kept;
}

/**
 * The git ref a manifest `value` resolves to. `latest` → highest stable tag (or
 * `main` when there are no stable tags yet). Exported so the guide route can hand
 * render.jsx the exact ref its ADR cross-refs must blob-link against — the island
 * cannot run git itself, so the server resolves the ref for it.
 */
export function refForVersion(version: string): string {
  if (version === DEFAULT_VERSION) return DEFAULT_VERSION;
  if (version === LATEST_VERSION) {
    const stable = publishableTags().find((t) => t.prerelease === null);
    return stable ? stable.raw : DEFAULT_VERSION;
  }
  return version; // a tag name, used verbatim as the ref.
}

// ── frontmatter ─────────────────────────────────────────────────────────────
// Dependency-free splitter. The four schema fields (ADR-0115) are all optional
// scalars, and today's / older guides carry NO frontmatter at all, so a
// leading `---` block with `key: value` lines is all we ever need to parse.
// (Astro's content collection handles `main` via gray-matter; this path only
// runs for the git-extracted refs.)

function splitFrontmatter(raw: string): { body: string; frontmatter: Frontmatter } {
  const m = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/.exec(raw);
  if (!m) return { body: raw, frontmatter: {} };
  const body = raw.slice(m[0].length);
  const fm: Frontmatter = {};
  for (const line of m[1].split(/\r?\n/)) {
    const kv = /^([A-Za-z_][\w-]*):\s*(.*)$/.exec(line);
    if (!kv) continue;
    const key = kv[1];
    let val = kv[2].trim();
    if (
      (val.startsWith('"') && val.endsWith('"')) ||
      (val.startsWith("'") && val.endsWith("'"))
    ) {
      val = val.slice(1, -1);
    }
    if (key === "title") fm.title = val;
    else if (key === "section") fm.section = val;
    else if (key === "description") fm.description = val;
    else if (key === "order" && val !== "") fm.order = Number(val);
  }
  return { body, frontmatter: fm };
}

// ── resolution ──────────────────────────────────────────────────────────────

/** `entry.id`-style slug: filename without the `.md` extension. */
function slugOf(path: string): string {
  return path.replace(/^.*\//, "").replace(/\.md$/, "");
}

/** Read `main`/working-tree guides from the Astro content collection. */
async function sourcesFromCollection(versionLabel: string): Promise<Source[]> {
  const entries = await getCollection("guides");
  return entries
    .map((entry) => ({
      version: versionLabel,
      slug: entry.id,
      rawMarkdown: entry.body ?? "",
      frontmatter: (entry.data ?? {}) as Frontmatter,
    }))
    .sort((a, b) => a.slug.localeCompare(b.slug));
}

/** Extract a ref's guides read-only via `git ls-tree` + `git show`. */
function sourcesFromRef(ref: string, versionLabel: string): Source[] {
  let listing: string;
  try {
    listing = git(["ls-tree", "-r", "--name-only", ref, GUIDES_DIR], repoRoot());
  } catch {
    console.warn(
      `[source-resolution] ref "${ref}" (version "${versionLabel}"): git ls-tree failed — skipped, not fatal.`,
    );
    return [];
  }
  const files = listing
    .split("\n")
    .map((f) => f.trim())
    .filter((f) => f.endsWith(".md"));
  if (files.length === 0) {
    console.warn(
      `[source-resolution] ref "${ref}" (version "${versionLabel}") has no ${GUIDES_DIR}/ — skipped, not fatal.`,
    );
    return [];
  }
  const sources: Source[] = [];
  for (const file of files) {
    let raw: string;
    try {
      raw = git(["show", `${ref}:${file}`], repoRoot());
    } catch {
      console.warn(
        `[source-resolution] ref "${ref}": could not read ${file} — skipped.`,
      );
      continue;
    }
    const { body, frontmatter } = splitFrontmatter(raw);
    sources.push({ version: versionLabel, slug: slugOf(file), rawMarkdown: body, frontmatter });
  }
  return sources.sort((a, b) => a.slug.localeCompare(b.slug));
}

/**
 * Resolve every guide at `version`.
 *
 * `main` and the `latest`-alias-when-there-are-no-tags read the working tree via
 * the content collection; every real tag (and `latest` once a stable release
 * exists) is extracted read-only from git. Refs lacking `docs/guides/` yield an
 * empty list and a logged skip — never fatal.
 */
export async function resolveSources(
  version: string = DEFAULT_VERSION,
): Promise<Source[]> {
  if (version === DEFAULT_VERSION) return sourcesFromCollection(DEFAULT_VERSION);
  const ref = refForVersion(version);
  // `latest` with no stable release resolves to `main` → read the working tree,
  // but stamp the label "latest" so the alias tree is emitted.
  if (ref === DEFAULT_VERSION) return sourcesFromCollection(version);
  return sourcesFromRef(ref, version);
}

/**
 * The version manifest, in the exact shape T4 (#354) feeds to the DS VersionSelect
 * (`VersionOption[]`). Order: latest → release tags (newest first) → main.
 *
 *   • `latest` .... always present, tag "current" (tracks the highest stable tag,
 *                   or `main` today). This is the "current" badge.
 *   • each `v*` .... a plain entry (no badge); prereleases included so their docs
 *                   are selectable, but they never carry "current".
 *   • `main` ...... tag "dev".
 *
 * Today (zero tags): [{value:"latest",tag:"current"}, {value:"main",tag:"dev"}].
 * T4 imports this function directly and passes the result to the TopNav island —
 * no reshaping (see docs-site/PIPELINE.md).
 */
export function listVersions(): VersionOption[] {
  const options: VersionOption[] = [{ value: LATEST_VERSION, tag: "current" }];
  for (const t of publishableTags()) options.push({ value: t.raw });
  options.push({ value: DEFAULT_VERSION, tag: "dev" });
  return options;
}

/**
 * Canonical path for a slug — always the `latest` alias, regardless of which
 * version the page was rendered for (task #351 point 4). Root-relative because
 * astro.config sets no `site`; T5 may prepend the deploy origin.
 */
export function canonicalPath(slug: string): string {
  return `/${LATEST_VERSION}/${slug}`;
}
