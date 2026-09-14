import { readFileSync } from "node:fs";
import { relative } from "node:path";
import { parse } from "../doclint/engine.mjs";
import { extractCitationsFromTree } from "../citations/extract.mjs";
import { classify } from "../citations/classify.mjs";
import { familyOf } from "../citations/scope.mjs";
import { ROWS, rowFor } from "../citations/rows.mjs";
import { CONTAINMENT_ROW } from "../citations/rows/containment.mjs";
import { rivalName } from "./corroborate.mjs";

// The burn-down entries are deleted, so the audit reads the tree and not the list (SPEC §6.1).
export function writtenAnchors(repoRoot, env, files) {
  const out = [];
  for (const abs of files) {
    const file = relative(repoRoot, abs).replace(/\\/g, "/");
    let markdown;
    try {
      markdown = readFileSync(abs, "utf8");
    } catch {
      continue;
    }
    const lines = markdown.split("\n");
    for (const r of classify(env, file, extractCitationsFromTree(parse(markdown)))) {
      // Arm B judges an anchor only where the path resolves, and so does the guard (SPEC §7.6).
      if (r.anchor === undefined || r.status !== "ok") continue;
      out.push({
        ...r,
        file,
        family: familyOf(file) ?? ".",
        lineText: lines[r.line - 1] ?? "",
      });
    }
  }
  return out.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line);
}

// A containment anchor names no declaration, so no rival can contradict one (SPEC §3.2 rule 2).
function judgeRow(repoRoot, env, row, anchors) {
  let inventory;
  try {
    inventory = row.inventory(repoRoot, [...new Set(anchors.map((a) => a.path))].sort());
  } catch (err) {
    const detail = (err.message ?? "").split("\n")[0] || err.code || "it did not finish";
    return anchors.map((a) => ({ ...a, verdict: "unreadable", detail: `the ${row.name} row could not run: ${detail}` }));
  }
  return anchors.map((a) => {
    const entry = inventory.get(a.path);
    if (entry === undefined) return { ...a, verdict: "unreadable", detail: "the inventory names no such target" };
    if (entry.error) return { ...a, verdict: "unreadable", detail: entry.error };
    if (!entry.spans) return { ...a, verdict: "unreadable", detail: "the target declares no anchor vocabulary" };
    const judged = rivalName(a.lineText, a.raw, entry.spans.keys(), a.anchor, env.extensions);
    return { ...a, row: row.name, ...judged };
  });
}

export function auditAnchors(repoRoot, env, files, rows = ROWS) {
  const anchors = writtenAnchors(repoRoot, env, files);
  const byRow = new Map();
  const judged = [];
  for (const a of anchors) {
    const row = rowFor(a.path, rows);
    if (row === CONTAINMENT_ROW) {
      judged.push({ ...a, row: row.name, verdict: "unreadable", detail: "the target declares no anchor vocabulary" });
      continue;
    }
    if (!byRow.has(row)) byRow.set(row, []);
    byRow.get(row).push(a);
  }
  for (const [row, cited] of byRow) judged.push(...judgeRow(repoRoot, env, row, cited));
  return judged.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line);
}

export function countByFamily(judged) {
  const families = new Map();
  for (const a of judged) {
    if (!families.has(a.family)) {
      families.set(a.family, { family: a.family, anchors: 0, suspect: 0, corroborated: 0, unproven: 0, unreadable: 0 });
    }
    const row = families.get(a.family);
    row[a.verdict] += 1;
    if (a.verdict !== "unreadable") row.anchors += 1;
  }
  return [...families.values()].sort((a, b) => a.family.localeCompare(b.family));
}

export function formatSuspect(a) {
  return `${a.file}:${a.line}  ->  ${a.value}#${a.anchor}  (the citing line names \`${a.rival}\` ${a.position} the citation)`;
}

export function reportAudit(judged, log = console.log) {
  const counts = countByFamily(judged);
  const suspects = judged.filter((a) => a.verdict === "suspect");
  const unreadable = judged.filter((a) => a.verdict === "unreadable");

  const degrade = suspects.filter((a) => a.position === "before");
  const review = suspects.filter((a) => a.position === "after");

  if (degrade.length > 0) {
    log("");
    log("Suspect — the anchor drops and the bare path stays (SPEC §6.3):");
    for (const a of degrade) log(`  ${formatSuspect(a)}`);
  }
  if (review.length > 0) {
    log("");
    log("Review queue — a rival spelled only afterwards, so a human reads the pair (SPEC §5.2 rule 2):");
    for (const a of review) log(`  ${formatSuspect(a)}`);
  }
  // A row that could not run and a target with no vocabulary are not one outcome.
  if (unreadable.length > 0) {
    const why = new Map();
    for (const a of unreadable) why.set(a.detail, (why.get(a.detail) ?? 0) + 1);
    log("");
    log(`Unjudged — ${unreadable.length} anchor(s) the audit could not read:`);
    for (const [detail, count] of [...why].sort()) log(`  ${String(count).padStart(4)}  ${detail}`);
  }

  log("");
  log("audit — written region anchors inside the citations boundary (SPEC §1.2):");
  const n = (v) => String(v).padStart(4);
  for (const c of counts) {
    log(
      `  ${c.family.padEnd(14)}${n(c.anchors)} anchor(s): ${n(c.suspect)} suspect, ` +
        `${n(c.corroborated)} corroborated, ${n(c.unproven)} unproven`,
    );
  }
  return { counts, suspects: degrade, review, unreadable };
}
