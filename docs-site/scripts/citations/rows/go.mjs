import { execFileSync } from "node:child_process";
import { readSource, splitLines } from "./source.mjs";

// A column-0 regex fails in both directions here, so the row parses instead (SPEC §7.3).
export function goInventory(root, paths, { cwd = root } = {}) {
  // `go run` needs the module, and the history arm parses a tree that is not it (#2015).
  const out = execFileSync("go", ["run", "./cmd/godecls", "--root", root], {
    cwd,
    input: `${paths.join("\n")}\n`,
    encoding: "utf8",
    maxBuffer: 256 * 1024 * 1024,
    // A cold module cache behind a slow proxy would hang a required check (SPEC §7.7).
    timeout: 5 * 60 * 1000,
  });
  const parsed = JSON.parse(out);
  const inventory = new Map();
  for (const [path, entry] of Object.entries(parsed)) {
    if (entry.error) {
      inventory.set(path, { error: entry.error });
      continue;
    }
    // godecls reports the region, and the source a snippet matches is read here (SPEC §3.4).
    const { source, error } = readSource(root, path);
    if (error) {
      inventory.set(path, { error });
      continue;
    }
    inventory.set(path, {
      names: new Set(entry.names ?? []),
      spans: new Map(Object.entries(entry.spans ?? {})),
      lines: splitLines(source),
    });
  }
  return inventory;
}

export const GO_ROW = {
  name: "go",
  vocabulary: "top-level func, type, var or const",
  matches: (path) => path.endsWith(".go"),
  inventory: goInventory,
};
