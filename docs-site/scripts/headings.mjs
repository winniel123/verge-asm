import GithubSlugger, { slug } from "github-slugger";

// Slugging a subset diverges the de-duplication counter from the renderer (docs-site/PIPELINE.md).
export function headingInventory(markdown) {
  const slugger = new GithubSlugger();
  const ids = new Set();
  const deduplicated = new Set();
  const headings = [];
  const lines = markdown.split(/\r?\n/);
  let inFence = false;
  for (const [i, line] of lines.entries()) {
    if (/^\s*(```|~~~)/.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    const m = /^(#{1,6})\s+(.*?)\s*#*\s*$/.exec(line);
    if (!m) continue;
    // Stripping matches what both renderers slug, and it changes 83 of 3,878 headings (SPEC §4).
    const label = m[2]
      .replace(/`([^`]+)`/g, "$1")
      .replace(/\*\*([^*]+)\*\*/g, "$1")
      .replace(/\*([^*]+)\*/g, "$1")
      .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
      .trim();
    const id = slugger.slug(label);
    // The suffix is positional, so a heading inserted above silently re-points it (SPEC §4 rule 4).
    if (id !== slug(label)) deduplicated.add(id);
    ids.add(id);
    headings.push({ id, level: m[1].length, line: i + 1 });
  }
  return { ids, deduplicated, sections: sectionsOf(headings, lines.length) };
}

// A heading declares its section, and a snippet must sit inside it (SPEC §3.4, #1973).
function sectionsOf(headings, lineCount) {
  const sections = new Map();
  for (const [i, heading] of headings.entries()) {
    const next = headings.slice(i + 1).find((h) => h.level <= heading.level);
    const end = next ? next.line - 1 : lineCount;
    if (!sections.has(heading.id)) sections.set(heading.id, []);
    sections.get(heading.id).push([heading.line, end]);
  }
  return sections;
}

export function collectAnchors(markdown) {
  return headingInventory(markdown).ids;
}
