import { markerInventory } from "./marker.mjs";

// Indentation is legal template source, and a missed marker reds a correct citation (SPEC §7.3).
const DEFINE_NAME = /^[ \t]*\{\{-?[ \t]*define[ \t]+"([^"]*)"/;

export const TEMPLATE_ROW = {
  name: "template",
  vocabulary: "`{{define}}`",
  matches: (path) => path.startsWith("design-system/templates/") && path.endsWith(".tmpl"),
  inventory: (repoRoot, paths) => markerInventory(repoRoot, paths, DEFINE_NAME),
};
