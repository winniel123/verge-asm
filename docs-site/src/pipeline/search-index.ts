/*
 * PER-VERSION SEARCH INDEX — build-time generator.  OWNER: T4 (#354).
 *
 * Turns the pipeline's resolved sources for ONE version into a flat, client-side
 * search index (JSON). One record per guide root + one per `##`/`###` heading, so a
 * ⌘K query returns section-level hits that deep-link to the exact anchor.
 *
 * SEAM: reads ONLY `resolveSources(version)` (stage 1) — it does not touch the
 * renderer, the link gate, or nav-build. Anchor ids are produced by the SAME
 * `github-slugger` algorithm the renderer (render.jsx) and `extractToc` (slug.ts)
 * use, and every heading (h1..h6) is visited in source order so the de-dup counter
 * stays in step — a record's `#anchor` therefore resolves to the rendered heading id
 * exactly. See docs-site/PIPELINE.md.
 *
 * The index is emitted per version by the static endpoint
 * `src/pages/search/[version].json.ts` as `/search/<version>.json`; TopNav loads the
 * ACTIVE version's file, so searching in version X only ever sees X's records.
 * (No new ADR — ADR-0115 covers the multi-version model; decision recorded here.)
 */
import GithubSlugger from "github-slugger";
import { resolveSources } from "./source-resolution";

/** One searchable unit: a guide root (level 0, no anchor) or a heading section. */
export interface SearchDoc {
  /** Guide slug — the `/<version>/<slug>` path segment. */
  slug: string;
  /** Guide display title (frontmatter.title or a humane slug fallback). */
  guideTitle: string;
  /** Heading text; "" for the guide-root record. */
  heading: string;
  /** Heading depth (2 or 3); 0 for the guide root. */
  level: number;
  /** github-slugger anchor id; "" for the guide root. */
  anchor: string;
  /** In-site destination: `/<version>/<slug>` or `/<version>/<slug>#<anchor>`. */
  href: string;
  /** Plain-text prose of this section — index body carried for context/ranking. */
  text: string;
}

/** "zone-files" -> "Zone files". Mirrors nav-build's fallback so labels agree. */
function titleFromSlug(slug: string): string {
  const s = slug.replace(/[-_]+/g, " ");
  return s.charAt(0).toUpperCase() + s.slice(1);
}

/** Strip the small set of inline/line markdown a guide uses down to plain text. */
function plain(s: string): string {
  return s
    .replace(/`([^`]+)`/g, "$1") // `code`
    .replace(/!\[[^\]]*\]\([^)]*\)/g, "") // images
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1") // [text](url)
    .replace(/[*_]{1,3}([^*_]+)[*_]{1,3}/g, "$1") // **bold** / *em* / _em_
    .replace(/^\s*>\s?/, "") // blockquote marker
    .replace(/^\s*[-*+]\s+/, "") // bullet marker
    .replace(/^\s*\d+\.\s+/, "") // ordered marker
    .replace(/\|/g, " ") // table pipes
    .replace(/\s+/g, " ")
    .trim();
}

const TEXT_CAP = 1500; // keep each record's body small; enough for context matching.

/** Build the search records for one guide at one version. */
function docsForSource(
  version: string,
  slug: string,
  guideTitleRaw: string | undefined,
  markdown: string,
): SearchDoc[] {
  const guideTitle = guideTitleRaw ?? titleFromSlug(slug);
  const base = `/${version}/${slug}`;
  const root: SearchDoc = {
    slug,
    guideTitle,
    heading: "",
    level: 0,
    anchor: "",
    href: base,
    text: "",
  };
  const docs: SearchDoc[] = [root];
  // Prose flows into the most recent h2/h3 section (or the root before the first one).
  let current = root;
  // ONE slugger per guide, visiting EVERY heading in order — the de-dup counter must
  // line up with render.jsx / extractToc or anchors would drift.
  const slugger = new GithubSlugger();
  let inFence = false;

  for (const line of markdown.split(/\r?\n/)) {
    if (/^\s*(```|~~~)/.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;

    const h = /^(#{1,6})\s+(.*?)\s*#*\s*$/.exec(line);
    if (h) {
      const level = h[1].length;
      const headingText = plain(h[2]);
      const id = slugger.slug(headingText); // slug every heading (counter alignment)
      if (level === 2 || level === 3) {
        current = {
          slug,
          guideTitle,
          heading: headingText,
          level,
          anchor: id,
          href: `${base}#${id}`,
          text: headingText,
        };
        docs.push(current);
      }
      // h1 / h4..h6: counter already advanced; prose keeps flowing into `current`.
      continue;
    }

    const p = plain(line);
    if (p) current.text = current.text ? `${current.text} ${p}` : p;
  }

  for (const d of docs) d.text = d.text.slice(0, TEXT_CAP);
  return docs;
}

/**
 * The full search index for ONE version: every guide's root + heading records,
 * flattened in source (slug, then document) order.
 */
export async function buildSearchDocs(version: string): Promise<SearchDoc[]> {
  const sources = await resolveSources(version);
  return sources.flatMap((s) =>
    docsForSource(version, s.slug, s.frontmatter.title, s.rawMarkdown),
  );
}
