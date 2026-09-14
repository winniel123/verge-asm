import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const HERE = dirname(fileURLToPath(import.meta.url));

export const BURNDOWN_FILE = join(HERE, "line-anchors.json");

export const LIST_COMMENT =
  "The line anchors the sweep has not reached, one entry per document and token. " +
  "A citation names no line, and the check refuses every line anchor this list does not hold " +
  "(SPEC docs/spec/citation-anchors.md §5, §8.2). It also refuses an entry no scan finds, so a " +
  "conversion pull request edits the document and this list together. An entry carries no reason: " +
  "a reason makes the list permanent, and an empty list is what proves the sweep complete. " +
  "`npm run check:citations -- --prune-list` drops every converted entry. It only removes, " +
  "because a command that could add an entry would re-admit the token the ratchet refuses (#1969).";

export function burndownKey(file, token) {
  return JSON.stringify([file, token]);
}

export function loadBurndown(file = BURNDOWN_FILE) {
  const parsed = JSON.parse(readFileSync(file, "utf8"));
  const entries = parsed.entries ?? [];
  const seen = new Set();
  for (const e of entries) {
    if (!e.file || !e.token) {
      throw new Error(`citations: a burn-down entry needs file and token: ${JSON.stringify(e)}`);
    }
    // A reason makes the list permanent, and an empty list proves the sweep complete (§8.2 rule 3).
    if ("reason" in e) {
      throw new Error(`citations: a burn-down entry carries no reason: ${JSON.stringify(e)}`);
    }
    const key = burndownKey(e.file, e.token);
    if (seen.has(key)) {
      throw new Error(`citations: duplicate burn-down entry: ${e.file} -> ${e.token}`);
    }
    seen.add(key);
  }
  return entries;
}
