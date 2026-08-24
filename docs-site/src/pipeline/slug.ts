/*
 * Heading-slug algorithm + on-page TOC extraction.
 *
 * PIPELINE: shared helper used by BOTH the render+transform stage (heading `id`s,
 * see render.jsx) and the build-time TOC extraction here. Keeping one algorithm in
 * one file is the seam that lets T2 (#352) rewrite `foo.md#anchor` cross-links and
 * be certain its computed anchor matches the rendered heading `id` exactly.
 *
 * ALGORITHM: GitHub-flavoured slugs via the `github-slugger` package (the same
 * library `rehype-slug` uses). Lowercase, strip punctuation, spaces -> "-",
 * de-duplicate collisions with a numeric suffix (`-1`, `-2`, ...). The de-dup
 * counter is stateful and order-sensitive, so a slugger instance MUST visit every
 * heading in source order (all levels, h1..h6) for its counter to line up with the
 * renderer's. `extractToc` therefore slugs every heading it sees and only *emits*
 * the h2 rows — advancing the counter over h1/h3-h6 without listing them. The
 * on-page "On this page" TOC lists h2 headings only, per DocsPage.jsx (D6).
 */
import GithubSlugger from "github-slugger";

export interface TocItem {
  label: string;
  href: string;
  /** heading depth (2 or 3) — retained so callers can indent; DocsLayout ignores it */
  level?: number;
  active?: boolean;
}

/** Strip the small set of inline markdown that appears in guide headings to plain text. */
function stripInline(s: string): string {
  return s
    .replace(/`([^`]+)`/g, "$1") // `code`
    .replace(/\*\*([^*]+)\*\*/g, "$1") // **bold**
    .replace(/\*([^*]+)\*/g, "$1") // *italic*
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1") // [text](url)
    .trim();
}

/**
 * Extract the on-page table of contents from raw markdown.
 *
 * Returns one row per `##` (h2) heading, in document order, each `href` a
 * `#<slug>` fragment matching the rendered heading `id`. Fenced code blocks are
 * skipped so a `# comment` line inside ```sh is never mistaken for a heading.
 */
export function extractToc(markdown: string): TocItem[] {
  const slugger = new GithubSlugger();
  const toc: TocItem[] = [];
  let inFence = false;

  for (const line of markdown.split(/\r?\n/)) {
    if (/^\s*(```|~~~)/.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;

    const m = /^(#{1,6})\s+(.*?)\s*#*\s*$/.exec(line);
    if (!m) continue;

    const level = m[1].length;
    const label = stripInline(m[2]);
    // Slug EVERY heading to keep the de-dup counter in step with the renderer,
    // but only surface h2 in the on-page TOC (DocsPage.jsx lists h2 only).
    const id = slugger.slug(label);
    if (level === 2) {
      toc.push({ label, href: `#${id}`, level });
    }
  }

  return toc;
}
