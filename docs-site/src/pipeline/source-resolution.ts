/*
 * PIPELINE STAGE 1 of 3 — source-resolution.  OWNER: Tv (#351).
 *
 * Contract (do not change the shape — T2/T3 build against it):
 *   resolveSources(version?) -> Promise<Source[]>
 *   listVersions()           -> VersionOption[]
 *
 * A `Source` is one guide at one version: { version, slug, rawMarkdown, frontmatter }.
 * TODAY this stage returns a single version, "main", read from the Astro content
 * collection (src/content.config.ts). Tv swaps the internals to iterate git refs and
 * emit one Source per (ref x guide) — WITHOUT touching render.jsx or nav-build.ts,
 * because those consume only the Source/VersionOption types below. That is the seam.
 *
 * See docs-site/PIPELINE.md.
 */
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

/** The one version that exists today. Tv replaces this with a discovered ref set. */
export const DEFAULT_VERSION = "main";

/**
 * Resolve every guide at `version`.
 *
 * Today `version` is ignored beyond being stamped onto each Source — there is only
 * `main`, read from the content collection. Tv makes this fetch the guides as they
 * existed at the given git ref.
 */
export async function resolveSources(
  version: string = DEFAULT_VERSION,
): Promise<Source[]> {
  const entries = await getCollection("guides");
  return entries
    .map((entry) => ({
      version,
      slug: entry.id, // glob-loader id === filename without extension
      rawMarkdown: entry.body ?? "",
      frontmatter: (entry.data ?? {}) as Frontmatter,
    }))
    .sort((a, b) => a.slug.localeCompare(b.slug));
}

/**
 * The version manifest, in the exact shape T4 feeds to the DS VersionSelect.
 * Today: just `main`, tagged "dev". Tv returns the discovered refs, tagging the
 * newest release "current" and `main` "dev".
 */
export function listVersions(): VersionOption[] {
  return [{ value: DEFAULT_VERSION, tag: "dev" }];
}
