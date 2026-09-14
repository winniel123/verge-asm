import { ROWS, rowFor } from "./rows.mjs";

// Containment is a predicate, because no row can enumerate every token a file holds (#1973).
function declares(entry, anchor) {
  return entry.holds ? entry.holds(anchor) : entry.names.has(anchor);
}

// A containment anchor proves existence only, so its region is the whole file (SPEC §3.2 rule 3).
function regionsOf(entry, anchor) {
  if (!entry.lines) return undefined;
  return entry.spans ? entry.spans.get(anchor) : [[1, entry.lines.length]];
}

function collapse(text) {
  return text.replace(/\s+/g, " ").trim();
}

// The snippet holds a verbatim substring of one line inside the declaration (SPEC §3.4).
function holdsSnippet(lines, regions, snippet) {
  const want = collapse(snippet);
  if (want === "") return false;
  for (const [start, end] of regions) {
    for (let i = start; i <= end; i++) {
      if (collapse(lines[i - 1] ?? "").includes(want)) return true;
    }
  }
  return false;
}

// An anchor resolves against its row, or the required check goes red (SPEC §7.1, #1970).
export function armB(repoRoot, results, rows = ROWS) {
  // An `ignored` candidate was never a path citation, so it writes no citation anchor (#1450).
  const anchored = results.filter((r) => r.anchor && r.status !== "ignored");
  // The path failure is the real fault, and a second error on one token is noise (SPEC §7.6).
  const unresolved = anchored.filter((r) => r.status !== "ok");
  const broken = [];
  const verified = [];
  const fatal = [];

  const byRow = new Map();
  for (const r of anchored) {
    if (r.status !== "ok") continue;
    const row = rowFor(r.path, rows);
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
      const detail = (err.message ?? "").split("\n")[0] || err.code || "it did not finish";
      fatal.push({ row: row.name, detail });
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
      // A row may hold a name and still refuse the spelling, and the reason is its own (SPEC §4).
      const refusal = entry.refused?.get(r.anchor);
      if (refusal) {
        broken.push({ ...r, row: row.name, vocabulary: row.vocabulary, why: refusal });
        continue;
      }
      if (!declares(entry, r.anchor)) {
        broken.push({ ...r, row: row.name, vocabulary: row.vocabulary });
        continue;
      }
      // One error per token: a broken anchor names no region for a snippet to sit inside.
      if (r.snippet !== undefined) {
        const regions = regionsOf(entry, r.anchor);
        if (regions === undefined) {
          fatal.push({ ...r, row: row.name, detail: "the row reports no region for this anchor" });
          continue;
        }
        if (!holdsSnippet(entry.lines, regions, r.snippet)) {
          broken.push({
            ...r,
            row: row.name,
            why:
              `no such snippet: no line inside ${r.anchor} in ${r.path} holds ` +
              `\`${r.snippet}\``,
          });
          continue;
        }
      }
      verified.push(r);
    }
  }

  // `verified` counts a resolution that happened. A count derived by subtraction would
  // report an anchor a failed row never read as one the gate resolved.
  return { anchored, verified, broken, unresolved, fatal };
}

export function formatBroken(r) {
  const why = r.why ?? `no such anchor: ${r.path} declares no ${r.vocabulary} named ${r.anchor}`;
  return `${r.file}:${r.line}  ->  ${r.value}#${r.anchor}  (${why})`;
}

export function formatFatal(f) {
  if (f.file === undefined) return `check:citations: the ${f.row} row could not run (${f.detail})`;
  const where = `${f.file}:${f.line} -> ${f.value}#${f.anchor}`;
  return `check:citations: cannot judge ${where} (${f.detail})`;
}
