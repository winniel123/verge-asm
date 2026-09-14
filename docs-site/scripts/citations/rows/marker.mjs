import { readFileSync } from "node:fs";
import { resolve, sep } from "node:path";

// A column-0 marker is the real declaration syntax, so neither row parses (SPEC §7.3, #1971).
export function markerInventory(repoRoot, paths, pattern) {
  const root = resolve(repoRoot);
  const inventory = new Map();
  for (const path of paths) {
    const abs = resolve(root, path);
    if (abs !== root && !abs.startsWith(root + sep)) {
      inventory.set(path, { error: "the path leaves the repository root" });
      continue;
    }
    let source;
    try {
      source = readFileSync(abs, "utf8");
    } catch (err) {
      // An unreadable target is a claim this gate should judge and could not (SPEC §7.7).
      inventory.set(path, { error: err.code ?? err.message });
      continue;
    }
    const names = new Set();
    for (const match of source.matchAll(pattern)) names.add(match[1]);
    inventory.set(path, { names });
  }
  return inventory;
}
