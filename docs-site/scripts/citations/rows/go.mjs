import { execFileSync } from "node:child_process";

// A column-0 regex fails in both directions here, so the row parses instead (SPEC §7.3).
export function goInventory(repoRoot, paths) {
  const out = execFileSync("go", ["run", "./cmd/godecls", "--root", repoRoot], {
    cwd: repoRoot,
    input: `${paths.join("\n")}\n`,
    encoding: "utf8",
    maxBuffer: 256 * 1024 * 1024,
    // A cold module cache behind a slow proxy would hang a required check (SPEC §7.7).
    timeout: 5 * 60 * 1000,
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
