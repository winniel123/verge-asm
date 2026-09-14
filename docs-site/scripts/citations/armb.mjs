import { ROWS, rowFor } from "./rows.mjs";

// An anchor resolves against its row, or the required check goes red (SPEC §7.1, #1970).
export function armB(repoRoot, results, rows = ROWS) {
  // An `ignored` candidate was never a path citation, so it writes no citation anchor (#1450).
  const anchored = results.filter((r) => r.anchor && r.status !== "ignored");
  // The path failure is the real fault, and a second error on one token is noise (SPEC §7.6).
  const unresolved = anchored.filter((r) => r.status !== "ok");
  const noRow = [];
  const broken = [];
  const fatal = [];

  const byRow = new Map();
  for (const r of anchored) {
    if (r.status !== "ok") continue;
    const row = rowFor(r.path, rows);
    if (row === null) {
      noRow.push(r);
      continue;
    }
    if (!byRow.has(row)) byRow.set(row, []);
    byRow.get(row).push(r);
  }

  for (const [row, cited] of byRow) {
    const paths = [...new Set(cited.map((r) => r.path))].sort();
    let inventory;
    try {
      inventory = row.inventory(repoRoot, paths);
    } catch (err) {
      // An absent toolchain is a claim this gate should judge and could not (SPEC §7.7).
      fatal.push({ row: row.name, detail: err.message.split("\n")[0] });
      continue;
    }
    for (const r of cited) {
      const entry = inventory.get(r.path);
      if (entry === undefined) {
        fatal.push({ ...r, row: row.name, detail: "the inventory names no such target" });
        continue;
      }
      if (entry.error) {
        fatal.push({ ...r, row: row.name, detail: entry.error });
        continue;
      }
      if (!entry.names.has(r.anchor)) {
        broken.push({ ...r, row: row.name, vocabulary: row.vocabulary });
      }
    }
  }

  return { anchored, broken, noRow, unresolved, fatal };
}

export function formatBroken(r) {
  const why = `no such anchor: ${r.path} declares no ${r.vocabulary} named ${r.anchor}`;
  return `${r.file}:${r.line}  ->  ${r.value}#${r.anchor}  (${why})`;
}

export function formatFatal(f) {
  if (f.file === undefined) return `check:citations: the ${f.row} row could not run (${f.detail})`;
  const where = `${f.file}:${f.line} -> ${f.value}#${f.anchor}`;
  return `check:citations: cannot judge ${where} (${f.detail})`;
}
