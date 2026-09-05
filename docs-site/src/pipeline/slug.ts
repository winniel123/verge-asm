import GithubSlugger from "github-slugger";

export interface TocItem {
  label: string;
  href: string;
  level?: number;
  active?: boolean;
}

function stripInline(s: string): string {
  return s
    .replace(/`([^`]+)`/g, "$1")
    .replace(/\*\*([^*]+)\*\*/g, "$1")
    .replace(/\*([^*]+)\*/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .trim();
}

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
    const id = slugger.slug(label);
    if (level === 2) {
      toc.push({ label, href: `#${id}`, level });
    }
  }

  return toc;
}
