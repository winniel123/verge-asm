/*
 * PIPELINE STAGE 3 of 3 — nav-build.  OWNER: T3 (#353).
 *
 * Contract: buildNav(sources, activeSlug?) -> NavSection[], the exact shape
 * SectionNav.astro renders: [{ title, items: [{ label, href, active? }] }].
 *
 * TODAY frontmatter is empty, so there is no section/order to group by: this stage
 * emits a single flat "Guides" section listing every slug (label falls back to a
 * title-cased slug). T3 rewrites the body to group by `frontmatter.section` and sort
 * within a group by `frontmatter.order` — WITHOUT touching source-resolution.ts or
 * render.jsx, because it consumes only `Source` and returns only `NavSection[]`.
 *
 * See docs-site/PIPELINE.md.
 */
import type { Source } from "./source-resolution";

export interface NavItem {
  label: string;
  href: string;
  active?: boolean;
}

export interface NavSection {
  title: string;
  items: NavItem[];
}

/** "zone-files" -> "Zone files". A humane fallback label until T3 supplies `title`. */
function titleFromSlug(slug: string): string {
  const s = slug.replace(/[-_]+/g, " ");
  return s.charAt(0).toUpperCase() + s.slice(1);
}

/**
 * Build the left-rail section model for one version.
 *
 * @param sources    every guide at the active version (from resolveSources)
 * @param activeSlug the slug of the page currently being rendered (gets active:true)
 */
export function buildNav(
  sources: Source[],
  activeSlug?: string,
): NavSection[] {
  const items: NavItem[] = sources.map((s) => ({
    label: s.frontmatter.title ?? titleFromSlug(s.slug),
    href: `/${s.version}/${s.slug}`,
    active: s.slug === activeSlug,
  }));

  // Flat, single-section fallback. T3 replaces this with frontmatter-driven grouping.
  return [{ title: "Guides", items }];
}
