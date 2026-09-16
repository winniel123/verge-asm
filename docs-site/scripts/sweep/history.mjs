import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { ROWS, rowFor } from "../citations/rows.mjs";
import { CONTAINMENT_ROW } from "../citations/rows/containment.mjs";
import { basenameOf, formOf, lineAnchorPattern } from "../citations/lineanchor.mjs";
import { enclosing, resolveBasename, resolvePath, splitToken } from "./derive.mjs";

// A byte no Markdown source holds, so a diff body can never open a block.
export const BLOCK = "\u0001";

export function gitLogLine(repoRoot, file, line) {
  return execFileSync("git", ["log", `-L${line},${line}:${file}`, `--format=${BLOCK}%H %s`], {
    cwd: repoRoot,
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
}

export function gitShow(repoRoot, commit, path) {
  return execFileSync("git", ["show", `${commit}:${path}`], {
    cwd: repoRoot,
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
}

// `git log -L` walks the citing line through every rename and reflow, and a blame stops at one.
export function revisionsFromLog(text) {
  const revisions = [];
  let current = null;
  for (const raw of text.split("\n")) {
    if (raw.startsWith(BLOCK)) {
      const rest = raw.slice(BLOCK.length);
      const space = rest.indexOf(" ");
      current = {
        commit: space < 0 ? rest : rest.slice(0, space),
        subject: space < 0 ? "" : rest.slice(space + 1),
        added: [],
      };
      revisions.push(current);
      continue;
    }
    if (current === null) continue;
    if (raw.startsWith("+++")) continue;
    if (raw.startsWith("+")) current.added.push(raw.slice(1));
  }
  return revisions;
}

// One scanner reads the retired form here and in the conversion arm, so neither drifts (#1975).
export function citedLinesFor(env, file, added, path) {
  const found = [];
  for (const text of added) {
    for (const match of text.matchAll(lineAnchorPattern())) {
      const parts = splitToken(match[1]);
      if (parts === null || parts.value === "") continue;
      // The conversion arm resolves a slashless name against the tree, so this arm reads it too.
      const value =
        formOf(match[1]) === "file"
          ? resolveBasename(env, basenameOf(parts.value), new Set([path])).path
          : parts.value;
      if (value === undefined) continue;
      const resolved = resolvePath(env, file, value);
      if (resolved.status !== "ok" || resolved.path !== path) continue;
      found.push(parts);
    }
  }
  return found;
}

// The newest revision that still spelled a number is the last time a human asserted one.
export function witnessFor(env, file, revisions, path) {
  for (const revision of revisions) {
    const cited = citedLinesFor(env, file, revision.added, path);
    if (cited.length > 0) return { commit: revision.commit, subject: revision.subject, cited };
  }
  return null;
}

// Two causes leave an anchor unwitnessed, and a report that merged them would misstate the bound.
export const NO_NUMBER = "no revision of the citing line spells a line number";
export const UNPAIRED = "the witness spells a different count of numbers for the path";

function unwitnessed(citation, cause, detail) {
  return { ...citation, verdict: "unwitnessed", cause, detail: detail ?? cause };
}

// A doc line carries several citations, and only an equal count pairs them without a guess.
function pairOnLine(env, file, revisions, citations) {
  const out = [];
  const byPath = new Map();
  for (const c of citations) {
    if (!byPath.has(c.path)) byPath.set(c.path, []);
    byPath.get(c.path).push(c);
  }
  for (const [path, group] of byPath) {
    const witness = witnessFor(env, file, revisions, path);
    if (witness === null) {
      for (const c of group) out.push(unwitnessed(c, NO_NUMBER));
      continue;
    }
    if (witness.cited.length !== group.length) {
      const detail =
        `the citing line carries ${group.length} citation(s) of ${path} and ` +
        `${witness.commit.slice(0, 8)} spells ${witness.cited.length} line number(s) for it`;
      for (const c of group) out.push(unwitnessed(c, UNPAIRED, detail));
      continue;
    }
    for (const [i, c] of group.entries()) {
      const { value: citedValue, fromLine, toLine } = witness.cited[i];
      // `c.value` is the spelling the document carries today, and a report a human checks reads it.
      out.push({
        ...c,
        witness: { commit: witness.commit, subject: witness.subject },
        citedValue,
        fromLine,
        toLine,
      });
    }
  }
  return out;
}

function materialise(repoRoot, root, witnessed, show) {
  const failed = new Map();
  const wanted = new Map();
  for (const c of witnessed) {
    const key = `${c.witness.commit}/${c.path}`;
    if (wanted.has(key) || failed.has(key)) continue;
    let source;
    try {
      source = show(repoRoot, c.witness.commit, c.path);
    } catch {
      failed.set(key, `${c.path} was not in the tree at ${c.witness.commit.slice(0, 8)}`);
      continue;
    }
    const abs = join(root, key);
    mkdirSync(dirname(abs), { recursive: true });
    writeFileSync(abs, source);
    wanted.set(key, c.path);
  }
  return { wanted, failed };
}

function inventories(repoRoot, root, wanted, rows) {
  const byRow = new Map();
  for (const [key, path] of wanted) {
    const row = rowFor(path, rows);
    if (!byRow.has(row)) byRow.set(row, []);
    byRow.get(row).push(key);
  }
  const out = new Map();
  for (const [row, keys] of byRow) {
    if (row === CONTAINMENT_ROW) {
      for (const key of keys) out.set(key, { error: "the target declares no anchor vocabulary" });
      continue;
    }
    let inventory;
    try {
      // `go run` needs the module root, and the historical tree is a scratch directory.
      inventory = row.inventory(root, keys.sort(), { cwd: repoRoot });
    } catch (err) {
      const detail = (err.message ?? "").split("\n")[0] || err.code || "it did not finish";
      for (const key of keys) out.set(key, { error: `the ${row.name} row could not run: ${detail}` });
      continue;
    }
    for (const key of keys) out.set(key, inventory.get(key) ?? { error: "the inventory names no such target" });
  }
  return out;
}

function judgeOne(citation, entry) {
  const when = citation.witness.commit.slice(0, 8);
  if (entry === undefined || entry.error) {
    return { ...citation, verdict: "unreadable", detail: entry?.error ?? "the inventory names no such target" };
  }
  if (!entry.spans) {
    return { ...citation, verdict: "unreadable", detail: "the target declares no anchor vocabulary" };
  }
  // A reversed or zero range satisfies the containment test against a region holding neither line.
  if (citation.fromLine < 1 || citation.fromLine > citation.toLine) {
    const detail = `the range ${citation.fromLine}-${citation.toLine} runs backwards or starts before line 1`;
    return { ...citation, verdict: "unreadable", detail };
  }
  const lines = entry.lines ?? [];
  if (citation.toLine > lines.length) {
    const detail = `line ${citation.toLine} was past the end of ${citation.path} at ${when} (${lines.length} lines)`;
    return { ...citation, verdict: "unreadable", detail };
  }
  const region = enclosing(entry.spans, citation.fromLine, citation.toLine);
  if (region?.ambiguous) {
    const detail = `two declarations shared the enclosing region of line ${citation.fromLine} at ${when}`;
    return { ...citation, verdict: "unreadable", detail };
  }
  const declared = entry.spans.has(citation.anchor);
  const then = region === null ? null : region.name;
  const evidence = { then, declaredThen: declared };
  if (then === citation.anchor) return { ...citation, verdict: "consistent", ...evidence };
  const detail =
    then === null
      ? `no declaration enclosed line ${citation.fromLine} of ${citation.path} at ${when}`
      : `line ${citation.fromLine} of ${citation.path} sat in \`${then}\` at ${when}`;
  return { ...citation, verdict: "drifted", ...evidence, detail };
}

export function judgeHistory(repoRoot, env, citations, options = {}) {
  const { rows = ROWS, readLog = gitLogLine, show = gitShow } = options;
  const byLine = new Map();
  for (const c of citations) {
    const key = `${c.line}:${c.file}`;
    if (!byLine.has(key)) byLine.set(key, []);
    byLine.get(key).push(c);
  }

  const judged = [];
  const witnessed = [];
  for (const group of byLine.values()) {
    const { file, line } = group[0];
    let revisions;
    try {
      revisions = revisionsFromLog(readLog(repoRoot, file, line));
    } catch (err) {
      const detail = (err.message ?? "").split("\n")[0] || err.code || "it did not finish";
      for (const c of group) judged.push({ ...c, verdict: "unreadable", detail: `git log -L failed: ${detail}` });
      continue;
    }
    for (const paired of pairOnLine(env, file, revisions, group)) {
      if (paired.verdict === "unwitnessed") judged.push(paired);
      else witnessed.push(paired);
    }
  }

  const root = options.root ?? mkdtempSync(join(tmpdir(), "sweep-history-"));
  try {
    const { wanted, failed } = materialise(repoRoot, root, witnessed, show);
    const found = inventories(repoRoot, root, wanted, rows);
    for (const c of witnessed) {
      const key = `${c.witness.commit}/${c.path}`;
      if (failed.has(key)) {
        judged.push({ ...c, verdict: "unreadable", detail: failed.get(key) });
        continue;
      }
      judged.push(judgeOne(c, found.get(key)));
    }
  } finally {
    if (options.root === undefined) rmSync(root, { recursive: true, force: true });
  }

  return judged.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line || a.anchor.localeCompare(b.anchor));
}

export function countByFamily(judged) {
  const families = new Map();
  for (const c of judged) {
    if (!families.has(c.family)) {
      families.set(c.family, { family: c.family, judged: 0, drifted: 0, consistent: 0, unwitnessed: 0, unreadable: 0 });
    }
    const row = families.get(c.family);
    row[c.verdict] += 1;
    if (c.verdict === "drifted" || c.verdict === "consistent") row.judged += 1;
  }
  return [...families.values()].sort((a, b) => a.family.localeCompare(b.family));
}

// A human checks one row by hand, so the row carries every input the verdict rests on.
export function formatDrift(c) {
  const target = `${c.value}#${c.anchor}`;
  const lost = c.declaredThen ? "" : ", which the target did not declare then";
  return (
    `${c.file}:${c.line}  ->  ${target}${lost}\n` +
    `      ${c.witness.commit.slice(0, 8)} wrote \`${c.citedValue}:${c.fromLine}\` — ${c.witness.subject}\n` +
    `      ${c.detail}`
  );
}

export function reportHistory(judged, log = console.log) {
  const counts = countByFamily(judged);
  const drifted = judged.filter((c) => c.verdict === "drifted");
  const unreadable = judged.filter((c) => c.verdict === "unreadable");

  if (drifted.length > 0) {
    log("");
    log("Drift candidates — the cited line sat in another declaration when the number was written:");
    for (const c of drifted) log(`  ${formatDrift(c)}`);
  }
  if (unreadable.length > 0) {
    const why = new Map();
    for (const c of unreadable) why.set(c.detail, (why.get(c.detail) ?? 0) + 1);
    log("");
    log(`Unjudged — ${unreadable.length} anchor(s) the history arm could not read:`);
    for (const [detail, count] of [...why].sort()) log(`  ${String(count).padStart(4)}  ${detail}`);
  }
  const unwitnessedBy = new Map();
  for (const c of judged) {
    if (c.verdict !== "unwitnessed") continue;
    unwitnessedBy.set(c.cause, (unwitnessedBy.get(c.cause) ?? 0) + 1);
  }
  if (unwitnessedBy.size > 0) {
    log("");
    log("Unwitnessed — no line number the arm can resolve stands behind these anchors:");
    for (const [cause, count] of [...unwitnessedBy].sort()) log(`  ${String(count).padStart(4)}  ${cause}`);
  }

  log("");
  log("history — anchors judged against the target's own history (SPEC §7 consequence 2):");
  const n = (v) => String(v).padStart(4);
  for (const c of counts) {
    log(
      `  ${c.family.padEnd(14)}${n(c.judged)} judged: ${n(c.drifted)} drift candidate(s), ` +
        `${n(c.consistent)} consistent, ${n(c.unwitnessed)} unwitnessed`,
    );
  }
  return { counts, drifted, unreadable };
}
