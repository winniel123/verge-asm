import { readFileSync } from "node:fs";
import { resolve, sep } from "node:path";
import { headingInventory } from "../../headings.mjs";

// The slug is derived, so no heading holds it verbatim and containment cannot verify it (SPEC §4).
function markdownInventory(repoRoot, paths) {
  const root = resolve(repoRoot);
  const inventory = new Map();
  for (const path of paths) {
    const abs = resolve(root, path);
    if (abs !== root && !abs.startsWith(root + sep)) {
      inventory.set(path, { error: "the path leaves the repository root" });
      continue;
    }
    let markdown;
    try {
      markdown = readFileSync(abs, "utf8");
    } catch (err) {
      // An unreadable target is a claim this gate should judge and could not (SPEC §7.7).
      inventory.set(path, { error: err.code ?? err.message });
      continue;
    }
    const { ids, deduplicated } = headingInventory(markdown);
    const refused = new Map();
    for (const id of deduplicated) {
      refused.set(
        id,
        `a de-duplicated slug: ${path} repeats this heading, and the numeric suffix is positional, ` +
          "so a copy inserted above re-points the anchor in silence. Retitle the heading instead",
      );
    }
    inventory.set(path, { names: ids, refused });
  }
  return inventory;
}

export const MARKDOWN_ROW = {
  name: "markdown",
  vocabulary: "`github-slugger` heading slug",
  matches: (path) => path.endsWith(".md"),
  inventory: markdownInventory,
};
