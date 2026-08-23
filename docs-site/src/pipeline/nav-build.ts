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

/** Guides without a `section` land here — this is also the whole rail for a
 *  frontmatter-LESS version, where it degrades to one flat, filename-titled list. */
const FALLBACK_SECTION = "Guides";

/** Undefined `order` sorts last, in the source's incoming (slug) order. */
const UNORDERED = Number.POSITIVE_INFINITY;

/**
 * Build the left-rail section model for one version, grouped and ordered from that
 * version's own guide frontmatter.
 *
 * - Guides group by `frontmatter.section`; any guide missing one falls into a
 *   `"Guides"` catch-all. A version whose guides carry NO frontmatter therefore
 *   collapses to a single flat `"Guides"` section (filename-derived titles) — a sane
 *   fallback, never a blank rail.
 * - Within a section, guides sort by `frontmatter.order` ascending; ties and
 *   order-less guides keep the incoming slug order (resolveSources sorts by slug).
 * - Sections themselves order by their smallest `order`, then by first appearance —
 *   `order` is a within-section field (per the schema), so this is the only stable,
 *   frontmatter-derived way to rank the groups without a new required field.
 *
 * Consumes only `Source`; returns only `NavSection[]`. No edits to stages 1 or 2.
 *
 * @param sources    every guide at the active version (from resolveSources)
 * @param activeSlug the slug of the page currently being rendered (gets active:true)
 */
export function buildNav(
  sources: Source[],
  activeSlug?: string,
): NavSection[] {
  interface Group {
    title: string;
    firstSeen: number;
    minOrder: number;
    rows: { item: NavItem; order: number; seq: number }[];
  }

  const groups = new Map<string, Group>();

  sources.forEach((s, seq) => {
    const sectionTitle = s.frontmatter.section ?? FALLBACK_SECTION;
    const order = s.frontmatter.order ?? UNORDERED;

    let group = groups.get(sectionTitle);
    if (!group) {
      group = { title: sectionTitle, firstSeen: seq, minOrder: order, rows: [] };
      groups.set(sectionTitle, group);
    }
    group.minOrder = Math.min(group.minOrder, order);

    group.rows.push({
      item: {
        label: s.frontmatter.title ?? titleFromSlug(s.slug),
        href: `/${s.version}/${s.slug}`,
        active: s.slug === activeSlug,
      },
      order,
      seq,
    });
  });

  return [...groups.values()]
    // Rank sections by smallest order, then by first appearance (stable).
    .sort((a, b) => a.minOrder - b.minOrder || a.firstSeen - b.firstSeen)
    .map((group) => ({
      title: group.title,
      // Rank items within a section by order, then by incoming (slug) sequence.
      items: group.rows
        .sort((a, b) => a.order - b.order || a.seq - b.seq)
        .map((row) => row.item),
    }));
}
