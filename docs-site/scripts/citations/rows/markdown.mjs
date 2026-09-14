import { headingInventory } from "../../headings.mjs";
import { readSource, splitLines } from "./source.mjs";

// The slug is derived, so no heading holds it verbatim and containment cannot verify it (SPEC §4).
function markdownInventory(repoRoot, paths) {
  const inventory = new Map();
  for (const path of paths) {
    const { source, error } = readSource(repoRoot, path);
    if (error) {
      inventory.set(path, { error });
      continue;
    }
    const { ids, deduplicated, sections } = headingInventory(source);
    const refused = new Map();
    for (const id of deduplicated) {
      refused.set(
        id,
        `a de-duplicated slug: ${path} repeats this heading, and the numeric suffix is positional, ` +
          "so a copy inserted above re-points the anchor in silence. Retitle the heading instead",
      );
    }
    inventory.set(path, { names: ids, refused, spans: sections, lines: splitLines(source) });
  }
  return inventory;
}

export const MARKDOWN_ROW = {
  name: "markdown",
  vocabulary: "`github-slugger` heading slug",
  matches: (path) => path.endsWith(".md"),
  inventory: markdownInventory,
};
