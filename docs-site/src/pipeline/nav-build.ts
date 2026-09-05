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

function titleFromSlug(slug: string): string {
  const s = slug.replace(/[-_]+/g, " ");
  return s.charAt(0).toUpperCase() + s.slice(1);
}

const FALLBACK_SECTION = "Guides";

const UNORDERED = Number.POSITIVE_INFINITY;

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
    // Ranking a section by its smallest order avoids adding a section-order field to the schema.
    .sort((a, b) => a.minOrder - b.minOrder || a.firstSeen - b.firstSeen)
    .map((group) => ({
      title: group.title,
      items: group.rows
        .sort((a, b) => a.order - b.order || a.seq - b.seq)
        .map((row) => row.item),
    }));
}
