import { execFileSync } from "node:child_process";
import { getCollection } from "astro:content";

export interface Frontmatter {
  title?: string;
  section?: string;
  order?: number;
  description?: string;
}

export interface Source {
  version: string;
  slug: string;
  rawMarkdown: string;
  frontmatter: Frontmatter;
}

// The shape mirrors the design system's VersionSelect.d.ts, so the manifest needs no reshaping.
export interface VersionOption {
  value: string;
  tag?: string;
}

export const DEFAULT_VERSION = "main";
export const LATEST_VERSION = "latest";

const GUIDES_DIR = "docs/guides";
const SEMVER_TAG = /^v(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/;

let repoRootCache: string | null = null;
function repoRoot(): string {
  if (repoRootCache) return repoRootCache;
  repoRootCache = git(["rev-parse", "--show-toplevel"]);
  return repoRootCache;
}

// Reading the object store, never checking out, is what makes a dirty tree safe to build from.
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

function allSemverTags(): ParsedTag[] {
  let out: string;
  try {
    out = git(["tag", "-l", "v*"], repoRoot());
  } catch {
    return [];
  }
  const tags: ParsedTag[] = [];
  for (const line of out.split("\n")) {
    const raw = line.trim();
    if (!raw) continue;
    const m = SEMVER_TAG.exec(raw);
    if (!m) continue;
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

function compareTagsDesc(a: ParsedTag, b: ParsedTag): number {
  if (a.major !== b.major) return b.major - a.major;
  if (a.minor !== b.minor) return b.minor - a.minor;
  if (a.patch !== b.patch) return b.patch - a.patch;
  if (a.prerelease === null && b.prerelease !== null) return -1;
  if (a.prerelease !== null && b.prerelease === null) return 1;
  if (a.prerelease === null && b.prerelease === null) return 0;
  return (b.prerelease as string).localeCompare(a.prerelease as string);
}

function hasGuides(ref: string): boolean {
  try {
    const listing = git(["ls-tree", "-r", "--name-only", ref, GUIDES_DIR], repoRoot());
    return listing.split("\n").some((f) => f.trim().endsWith(".md"));
  } catch {
    return false;
  }
}

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

export function refForVersion(version: string): string {
  if (version === DEFAULT_VERSION) return DEFAULT_VERSION;
  if (version === LATEST_VERSION) {
    const stable = publishableTags().find((t) => t.prerelease === null);
    return stable ? stable.raw : DEFAULT_VERSION;
  }
  return version;
}

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

function slugOf(path: string): string {
  return path.replace(/^.*\//, "").replace(/\.md$/, "");
}

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

export async function resolveSources(
  version: string = DEFAULT_VERSION,
): Promise<Source[]> {
  if (version === DEFAULT_VERSION) return sourcesFromCollection(DEFAULT_VERSION);
  const ref = refForVersion(version);
  // Stamping "latest" rather than "main" is what emits the alias tree while no stable tag exists.
  if (ref === DEFAULT_VERSION) return sourcesFromCollection(version);
  return sourcesFromRef(ref, version);
}

export function listVersions(): VersionOption[] {
  const tags = publishableTags();
  // A prerelease is browsable, but "current" never names one, so a candidate is never the default (ADR-0155 §3).
  const newestStable = tags.find((t) => t.prerelease === null) ?? null;
  const options: VersionOption[] = [
    newestStable ? { value: LATEST_VERSION } : { value: LATEST_VERSION, tag: "current" },
  ];
  for (const t of tags) {
    options.push(t.raw === newestStable?.raw ? { value: t.raw, tag: "current" } : { value: t.raw });
  }
  options.push({ value: DEFAULT_VERSION, tag: "dev" });
  return options;
}

export function canonicalPath(slug: string): string {
  return `/${LATEST_VERSION}/${slug}`;
}
