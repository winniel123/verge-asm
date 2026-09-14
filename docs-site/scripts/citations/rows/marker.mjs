import { readSource, splitLines, spansFromMarkers } from "./source.mjs";

// A column-0 marker is the real declaration syntax, so neither row parses (SPEC §7.3, #1971).
export function markerInventory(repoRoot, paths, pattern) {
  // A sticky or global pattern carries lastIndex between lines, and drops every other marker.
  const perLine = new RegExp(pattern.source, pattern.flags.replace(/[gy]/g, ""));
  const inventory = new Map();
  for (const path of paths) {
    const { source, error } = readSource(repoRoot, path);
    if (error) {
      inventory.set(path, { error });
      continue;
    }
    const lines = splitLines(source);
    const marks = [];
    for (const [i, line] of lines.entries()) {
      const m = perLine.exec(line);
      if (m) marks.push({ name: m[1], line: i + 1 });
    }
    const names = new Set(marks.map((m) => m.name));
    inventory.set(path, { names, spans: spansFromMarkers(marks, lines.length), lines });
  }
  return inventory;
}
