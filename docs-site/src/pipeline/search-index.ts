import GithubSlugger from "github-slugger";
import { resolveSources } from "./source-resolution";

export interface SearchDoc {
  slug: string;
  guideTitle: string;
  heading: string;
  level: number;
  anchor: string;
  href: string;
  text: string;
}

// This mirrors nav-build.ts's fallback, so a rail label and a search result never disagree.
function titleFromSlug(slug: string): string {
  const s = slug.replace(/[-_]+/g, " ");
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function plain(s: string): string {
  return s
    .replace(/`([^`]+)`/g, "$1")
    .replace(/!\[[^\]]*\]\([^)]*\)/g, "")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/[*_]{1,3}([^*_]+)[*_]{1,3}/g, "$1")
    .replace(/^\s*>\s?/, "")
    .replace(/^\s*[-*+]\s+/, "")
    .replace(/^\s*\d+\.\s+/, "")
    .replace(/\|/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

const TEXT_CAP = 1500;

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
  let current = root;
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
      const id = slugger.slug(headingText);
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
      continue;
    }

    const p = plain(line);
    if (p) current.text = current.text ? `${current.text} ${p}` : p;
  }

  for (const d of docs) d.text = d.text.slice(0, TEXT_CAP);
  return docs;
}

// One index per version, so a query can never surface another version's headings.
export async function buildSearchDocs(version: string): Promise<SearchDoc[]> {
  const sources = await resolveSources(version);
  return sources.flatMap((s) =>
    docsForSource(version, s.slug, s.frontmatter.title, s.rawMarkdown),
  );
}
