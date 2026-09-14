import GithubSlugger, { slug } from "github-slugger";

// Slugging a subset diverges the de-duplication counter from the renderer (docs-site/PIPELINE.md).
export function headingInventory(markdown) {
  const slugger = new GithubSlugger();
  const ids = new Set();
  const deduplicated = new Set();
  let inFence = false;
  for (const line of markdown.split(/\r?\n/)) {
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
  }
  return { ids, deduplicated };
}

export function collectAnchors(markdown) {
  return headingInventory(markdown).ids;
}
