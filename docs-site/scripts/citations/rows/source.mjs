import { readFileSync } from "node:fs";
import { resolve, sep } from "node:path";

// One read for every row, so a target that leaves the root fails the same way in each (#1973).
export function readSource(repoRoot, path) {
  const root = resolve(repoRoot);
  const abs = resolve(root, path);
  if (abs !== root && !abs.startsWith(root + sep)) {
    return { error: "the path leaves the repository root" };
  }
  try {
    return { source: readFileSync(abs, "utf8") };
  } catch (err) {
    // An unreadable target is a claim this gate should judge and could not (SPEC §7.7).
    return { error: err.code ?? err.message };
  }
}

export function splitLines(source) {
  return source.split(/\r?\n/);
}

// The region runs to the next marker, because a false red is the fatal direction (SPEC §3.4).
export function spansFromMarkers(marks, lineCount) {
  const spans = new Map();
  for (let i = 0; i < marks.length; i++) {
    const { name, line } = marks[i];
    const end = i + 1 < marks.length ? marks[i + 1].line - 1 : lineCount;
    if (!spans.has(name)) spans.set(name, []);
    spans.get(name).push([line, end]);
  }
  return spans;
}
