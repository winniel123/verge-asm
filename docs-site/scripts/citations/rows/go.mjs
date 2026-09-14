import { execFileSync } from "node:child_process";

// A column-0 regex fails in both directions here, so the row parses instead (SPEC §7.3).
export function goInventory(repoRoot, paths) {
  const out = execFileSync("go", ["run", "./cmd/godecls"], {
    cwd: repoRoot,
    input: `${paths.join("\n")}\n`,
    encoding: "utf8",
    maxBuffer: 256 * 1024 * 1024,
  });
  const parsed = JSON.parse(out);
  const inventory = new Map();
  for (const [path, entry] of Object.entries(parsed)) {
    if (entry.error) inventory.set(path, { error: entry.error });
    else inventory.set(path, { names: new Set(entry.names ?? []) });
  }
  return inventory;
}

export const GO_ROW = {
  name: "go",
  vocabulary: "top-level func, type, var or const",
  matches: (path) => path.endsWith(".go"),
  inventory: goInventory,
};
