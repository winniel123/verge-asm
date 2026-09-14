import { readSource, splitLines } from "./source.mjs";

const IDENTIFIER = /[A-Za-z0-9_]/;

// A rename to `heartbeat_v2` is the drift this row exists to catch, so a substring is no hit.
function holdsToken(source, anchor) {
  for (let at = source.indexOf(anchor); at >= 0; at = source.indexOf(anchor, at + 1)) {
    const before = source[at - 1] ?? "";
    const after = source[at + anchor.length] ?? "";
    if (!IDENTIFIER.test(before) && !IDENTIFIER.test(after)) return true;
  }
  return false;
}

// A target that declares no name MAY carry one token it contains (SPEC §3.2 rule 2).
function containmentInventory(repoRoot, paths) {
  const inventory = new Map();
  for (const path of paths) {
    const { source, error } = readSource(repoRoot, path);
    if (error) {
      inventory.set(path, { error });
      continue;
    }
    // No row can enumerate every token a file holds, so this row answers a question instead.
    inventory.set(path, { holds: (anchor) => holdsToken(source, anchor), lines: splitLines(source) });
  }
  return inventory;
}

export const CONTAINMENT_ROW = {
  name: "containment",
  vocabulary: "token",
  matches: () => true,
  inventory: containmentInventory,
};
