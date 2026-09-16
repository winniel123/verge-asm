#!/usr/bin/env node
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";
import { parse } from "./doclint/engine.mjs";
import { scanLineAnchorsFromTree } from "./citations/lineanchor.mjs";
import { environment } from "./check-citations.mjs";
import { derive } from "./sweep/derive.mjs";
import { rewriteDocument, trailingGlue } from "./sweep/rewrite.mjs";
import { auditAnchors, reportAudit, writtenAnchors } from "./sweep/audit.mjs";
import { judgeHistory, reportHistory } from "./sweep/history.mjs";
import { familyOf, inScopeFiles } from "./citations/scope.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");

const USAGE = `usage:
  sweep-line-anchors [<path>...]            # a dry run, and it writes nothing
  sweep-line-anchors --write <path>...      # convert the line anchors in those documents
  sweep-line-anchors --report <file> ...    # also write the plan as JSON
  sweep-line-anchors --audit [<path>...]    # judge the anchors already written, and write nothing
  sweep-line-anchors --history [<path>...]  # judge those anchors against the target's history

A <path> is a document or a directory prefix, spelled from the repository root.
Every arm reads the citations boundary, and a run with no path reads all of it.
`;

function repoPath(repoRoot, abs) {
  return abs.slice(repoRoot.length + 1).replace(/\\/g, "/");
}

// A prefix reads as a directory, so `docs/spec` takes every document under it.
export function selectFiles(repoRoot, files, prefixes) {
  if (prefixes.length === 0) return files;
  return files.filter((abs) => {
    const rel = repoPath(repoRoot, abs);
    return prefixes.some((p) => rel === p || rel.startsWith(`${p}/`));
  });
}

// The guard reads the citing line, and this arm reads the target's history (SPEC §7 consequence 2).
export function historyCitations(repoRoot, env, files, rows) {
  const cited = writtenAnchors(repoRoot, env, files).map((a) => ({
    file: a.file,
    line: a.line,
    family: a.family,
    path: a.path,
    value: a.value,
    anchor: a.anchor,
    form: "region",
  }));
  // A retired line anchor names the declaration the conversion arm would write, so it judges too.
  const relPaths = files.map((abs) => repoPath(repoRoot, abs));
  for (const r of planFor(repoRoot, env, scanDocuments(repoRoot, relPaths, env), rows)) {
    if (r.outcome !== "anchor") continue;
    cited.push({
      file: r.file,
      line: r.line,
      family: familyOf(r.file) ?? ".",
      path: r.path,
      value: r.value,
      anchor: r.anchor,
      form: "line",
    });
  }
  return cited;
}

// A verdict rests on a commit and a line number, so the record carries both for a hand check.
export function historyRecord(judged) {
  return judged.map((c) => ({
    file: c.file,
    line: c.line,
    family: c.family,
    form: c.form,
    target: `${c.value}#${c.anchor}`,
    verdict: c.verdict,
    ...(c.witness ? { witness: c.witness.commit, cited: `${c.citedValue}:${c.fromLine}` } : {}),
    ...(c.cause ? { cause: c.cause } : {}),
    ...(c.then !== undefined ? { enclosedThen: c.then, declaredThen: c.declaredThen } : {}),
    ...(c.detail ? { detail: c.detail } : {}),
  }));
}

function history(prefixes, reportFile) {
  const env = environment(REPO_ROOT);
  const files = selectFiles(REPO_ROOT, inScopeFiles(REPO_ROOT), prefixes);
  const judged = judgeHistory(REPO_ROOT, env, historyCitations(REPO_ROOT, env, files));
  const { drifted } = reportHistory(judged);
  if (reportFile) {
    writeFileSync(reportFile, `${JSON.stringify(historyRecord(judged), null, 2)}\n`);
    console.log("");
    console.log(`  wrote the history report to ${reportFile}`);
  }
  console.log("");
  console.log(
    `  the history arm writes nothing. ${drifted.length} anchor(s) are drift candidates, and ` +
      "a human reads each one before it degrades (SPEC §6.3).",
  );
}

function audit(prefixes, reportFile) {
  const env = environment(REPO_ROOT);
  const files = selectFiles(REPO_ROOT, inScopeFiles(REPO_ROOT), prefixes);
  const judged = auditAnchors(REPO_ROOT, env, files);
  const { suspects, review } = reportAudit(judged);
  if (reportFile) {
    writeFileSync(reportFile, `${JSON.stringify(auditRecord(judged), null, 2)}\n`);
    console.log("");
    console.log(`  wrote the audit to ${reportFile}`);
  }
  console.log("");
  console.log(
    `  the audit writes nothing. Of ${suspects.length + review.length} suspect anchor(s), ` +
      `${suspects.length} degrade by hand (SPEC §6.3) and ${review.length} enter the review queue.`,
  );
}

// A suspect anchor is repaired by a human, so the record carries the pair and never a new target.
export function auditRecord(judged) {
  return judged.map((a) => ({
    file: a.file,
    line: a.line,
    family: a.family,
    target: `${a.value}#${a.anchor}`,
    verdict: a.verdict,
    ...(a.rival ? { rival: a.rival, position: a.position } : {}),
    ...(a.detail ? { detail: a.detail } : {}),
  }));
}

// A --write run leaves no trace of a degradation, and SPEC §8.4 feeds a later repair effort.
export function reportRecord(results) {
  return results.map((r) => ({
    file: r.file,
    line: r.line,
    token: r.token,
    outcome: r.outcome,
    row: r.row?.name,
    ...(r.outcome === "anchor" ? { anchor: `${r.value}#${r.anchor}` } : { reason: r.reason }),
    ...(r.review ? { review: r.review } : {}),
  }));
}

const TREE_PATH = /[A-Za-z0-9_.@][A-Za-z0-9_.@+-]*(?:\/[A-Za-z0-9_.@+-]+)+/g;

// An ADR cross-links with `./` and `../`, so the commonest spelling of a path is a relative one.
function absolutePath(match, docFile) {
  if (!/(?:^|\/)\.\.?\//.test(match)) return match;
  if (docFile === null) return match;
  const parts = docFile.split("/").slice(0, -1);
  for (const seg of match.split("/")) {
    if (seg === ".") continue;
    if (seg === "..") parts.pop();
    else parts.push(seg);
  }
  return parts.join("/");
}

// The document's own full paths disambiguate a basename several directories carry (#2120).
export function pathsNamedIn(markdown, env, docFile = null) {
  const out = new Set();
  if (!env?.tracked?.files) return out;
  for (const [match] of markdown.matchAll(TREE_PATH)) {
    const path = absolutePath(match, docFile);
    if (env.tracked.files.has(path)) out.add(path);
  }
  return out;
}

export function scanDocuments(repoRoot, files, env = null) {
  const found = new Map();
  for (const file of files) {
    let markdown;
    try {
      markdown = readFileSync(join(repoRoot, file), "utf8");
    } catch {
      // A listed document a later edit deleted must not kill the dry run (#1436 reads it so).
      continue;
    }
    const lines = markdown.split("\n");
    const namedInDocument = pathsNamedIn(markdown, env, file);
    const scan = scanLineAnchorsFromTree(parse(markdown), { knownFile: env?.knownFile ?? null });
    const hits = scan.map((h) => ({
      ...h,
      file,
      glue: trailingGlue(markdown, h),
      // The derivation reads the citing line for a name, so the scan carries it (#1977).
      lineText: lines[h.line - 1] ?? "",
      namedInDocument,
    }));
    found.set(file, { markdown, hits });
  }
  return found;
}

export function planFor(repoRoot, env, found, rows) {
  const hits = [];
  for (const [, { hits: scanned }] of found) hits.push(...scanned);
  return derive(repoRoot, env, hits, rows);
}

function report(results) {
  const anchors = results.filter((r) => r.outcome === "anchor");
  const degradations = results.filter((r) => r.outcome === "degraded");
  const holds = results.filter((r) => r.outcome === "held");

  let document = null;
  for (const r of anchors) {
    if (r.file !== document) {
      document = r.file;
      console.log("");
      console.log(document);
    }
    console.log(`  ${String(r.line).padStart(5)}  ${r.token}  ->  ${r.value}#${r.anchor}  [${r.row.name}]`);
  }

  const review = anchors.filter((r) => r.review);
  if (review.length > 0) {
    console.log("");
    console.log("Review queue — a rival spelled only afterwards, so a human reads the pair (SPEC §5.2 rule 2):");
    for (const r of review) {
      console.log(`  ${r.file}:${r.line}  ${r.value}#${r.anchor}  vs  \`${r.review.rival}\``);
    }
  }

  if (degradations.length > 0) {
    console.log("");
    console.log("Degraded — the anchor drops and the bare path stays (SPEC §8.4):");
    document = null;
    for (const r of degradations) {
      if (r.file !== document) {
        document = r.file;
        console.log("");
        console.log(document);
      }
      console.log(`  ${String(r.line).padStart(5)}  ${r.token}  ->  ${r.value}  (${r.reason})`);
    }
  }

  if (holds.length > 0) {
    console.log("");
    console.log("Held back — the path itself is gone, so the token stays and a human repairs it:");
    for (const r of holds) console.log(`  ${r.file}:${r.line}  ${r.token}  (${r.reason})`);
  }

  // A held token is never rewritten, so no suffix of its is ever dropped.
  const glued = results.filter((r) => r.glue && r.outcome !== "held");
  if (glued.length > 0) {
    console.log("");
    console.log("Consumed — a suffix the anchor class would swallow, and a region subsumes:");
    for (const r of glued) {
      console.log(`  ${r.file}:${r.line}  ${r.token}${r.glue}  drops \`${r.glue}\``);
    }
  }

  console.log("");
  const n = (rows) => String(rows.length).padStart(5);
  console.log(`sweep — ${results.length} token(s) read from the tree.`);
  console.log(`  ${n(anchors)}  derive an anchor`);
  console.log(`  ${n(review)}  of those enter the review queue`);
  console.log(`  ${n(degradations)}  degrade to a bare path`);
  console.log(`  ${n(holds)}  held back: the path itself is gone`);
  return { anchors, degradations, holds };
}

function planWrites(results, found, env) {
  const byFile = new Map();
  for (const r of results) {
    if (!byFile.has(r.file)) byFile.set(r.file, []);
    byFile.get(r.file).push(r);
  }
  const writes = [];
  for (const [file, conversions] of byFile) {
    const { markdown, hits } = found.get(file);
    const next = rewriteDocument(markdown, conversions);
    const after = scanLineAnchorsFromTree(parse(next), { knownFile: env?.knownFile ?? null });
    const converted = hits.length - after.length;
    if (converted !== conversions.length) {
      throw new Error(
        `sweep: ${file} converted ${converted} token(s), and ${conversions.length} were planned`,
      );
    }
    writes.push({ file, next });
  }
  return writes;
}

function writeDocuments(repoRoot, writes) {
  for (const { file, next } of writes) writeFileSync(join(repoRoot, file), next);
  return writes.length;
}

function main() {
  const argv = process.argv.slice(2);
  if (argv.includes("--help") || argv.includes("-h")) {
    process.stdout.write(USAGE);
    return;
  }
  const write = argv.includes("--write");
  const at = argv.indexOf("--report");
  const reportFile = at >= 0 ? argv[at + 1] : null;
  if (at >= 0 && (reportFile === undefined || reportFile.startsWith("--"))) {
    console.error("sweep: --report names the file it writes the plan to");
    process.exit(2);
  }
  // An absent --report puts `at` at -1, and the old guard then dropped the first path (#2007).
  const args = argv.filter((a, i) => !a.startsWith("--") && (at < 0 || i !== at + 1));
  const prefixes = args.map((p) => p.replace(/\/+$/, ""));
  if (argv.includes("--audit") && argv.includes("--history")) {
    console.error("sweep: --audit and --history are two instruments, and one run reports one of them");
    process.exit(2);
  }
  if (argv.includes("--audit") || argv.includes("--history")) {
    if (write) {
      console.error("sweep: a reporting arm decides nothing, and a human repairs an anchor (SPEC §6.3)");
      process.exit(2);
    }
    if (argv.includes("--history")) history(prefixes, reportFile);
    else audit(prefixes, reportFile);
    return;
  }
  if (write && prefixes.length === 0) {
    console.error("sweep: --write names the documents it rewrites, so it takes a path");
    process.exit(2);
  }

  const env = environment(REPO_ROOT);
  const files = selectFiles(REPO_ROOT, inScopeFiles(REPO_ROOT), prefixes).map((abs) =>
    repoPath(REPO_ROOT, abs),
  );
  const found = scanDocuments(REPO_ROOT, files, env);
  const results = planFor(REPO_ROOT, env, found);
  if (results.length === 0) {
    console.log("sweep — no line anchor under that path.");
    return;
  }
  const { anchors, degradations, holds } = report(results);

  if (reportFile) {
    writeFileSync(reportFile, `${JSON.stringify(reportRecord(results), null, 2)}\n`);
    console.log("");
    console.log(`  wrote the plan to ${reportFile}`);
  }

  if (!write) {
    console.log("");
    console.log("  a dry run wrote nothing. Re-run with --write <path>... to convert.");
    return;
  }
  const converted = results.filter((r) => r.outcome !== "held");
  // Build and check every rewrite before one byte lands, so no throw can split the act.
  const writes = planWrites(converted, found, env);
  const documents = writeDocuments(REPO_ROOT, writes);
  console.log("");
  console.log(
    `sweep — wrote ${documents} document(s): ${anchors.length} anchor(s) and ` +
      `${degradations.length} degradation(s), and held ${holds.length} back.`,
  );
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(fileURLToPath(import.meta.url))) {
  try {
    main();
  } catch (err) {
    console.error(`sweep: ${err.stack ?? err.message}`);
    process.exit(2);
  }
}
