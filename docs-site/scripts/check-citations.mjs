#!/usr/bin/env node
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, relative, resolve } from "node:path";
import { inScopeFiles, isInScope } from "./citations/scope.mjs";
import { extractCitationsFromTree } from "./citations/extract.mjs";
import { parse } from "./doclint/engine.mjs";
import {
  classify,
  trackedPaths,
  trackedExtensions,
  topLevelEntries,
} from "./citations/classify.mjs";
import { loadExemptions, exemptionMatcher } from "./citations/exempt.mjs";
import { scanLineAnchorsFromTree } from "./citations/lineanchor.mjs";
import { armB, formatBroken, formatFatal } from "./citations/armb.mjs";
import {
  loadBurndown,
  burndownKey,
  BURNDOWN_FILE,
  LIST_COMMENT,
} from "./citations/burndown.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");

export function environment(repoRoot, exemptionsFile) {
  const tracked = trackedPaths(repoRoot);
  return {
    repoRoot,
    tracked,
    extensions: trackedExtensions(tracked),
    roots: topLevelEntries(tracked),
    exempt: exemptionMatcher(loadExemptions(exemptionsFile)),
  };
}

export function run(repoRoot, files) {
  const env = environment(repoRoot);
  const results = [];
  const lineAnchors = [];
  const unreadable = [];
  for (const abs of files) {
    const docFile = relative(repoRoot, abs).replace(/\\/g, "/");
    let markdown;
    try {
      markdown = readFileSync(abs, "utf8");
    } catch (err) {
      // A changed-file list names a deleted document, so a missing file is operator error (#1436).
      unreadable.push({ file: docFile, code: err.code ?? err.message });
      continue;
    }
    // One parse for both arms, because the whole-tree run gates every merge on `main`.
    const tree = parse(markdown);
    for (const r of classify(env, docFile, extractCitationsFromTree(tree))) {
      results.push({ ...r, file: docFile });
    }
    // Arm A reads its own scanner, because the extractor's path class holds no colon (SPEC §7.5).
    for (const a of scanLineAnchorsFromTree(tree)) {
      lineAnchors.push({ ...a, file: docFile });
    }
  }
  return { results, lineAnchors, unreadable };
}

export function armA(lineAnchors, entries) {
  const listed = new Set(entries.map((e) => burndownKey(e.file, e.token)));
  const found = new Set(lineAnchors.map((a) => burndownKey(a.file, a.token)));
  return {
    refused: lineAnchors.filter((a) => !listed.has(burndownKey(a.file, a.token))),
    // An entry no scan finds silently re-licenses that exact token (SPEC §8.2 rule 4).
    stale: entries.filter((e) => !found.has(burndownKey(e.file, e.token))),
  };
}

export function formatLineAnchor(a) {
  const where = a.kind === "link" ? "link target" : "code span";
  return `${a.file}:${a.line}  ->  ${a.token}  (a citation names no line: this ${where} must name the enclosing declaration)`;
}

export function formatStale(e) {
  return `${e.file}  ->  ${e.token}  (stale entry: no scan finds this token, so delete the entry)`;
}

export function formatDead(r) {
  const why = r.detail ?? "no such path in the tree";
  return `${r.file}:${r.line}  ->  ${r.value}  (dead: ${why})`;
}

export function formatRefUnknown(r) {
  return `${r.file}:${r.line}  ->  ${r.value}  (ref gone: \`${r.ref}\` is not a ref this clone holds)`;
}

function formatNote(r, note) {
  return `  ${r.file}:${r.line}  ->  ${r.value}  (${note})`;
}

function resolveFiles(paths, inScopeOnly) {
  if (paths.length === 0) return inScopeOnly ? [] : inScopeFiles(REPO_ROOT);
  const abs = paths.map((p) => resolve(process.cwd(), p));
  return inScopeOnly ? abs.filter((a) => isInScope(REPO_ROOT, a)) : abs;
}

function pruneList(lineAnchors, entries) {
  const found = new Set(lineAnchors.map((a) => burndownKey(a.file, a.token)));
  // Shrink only. A command that could add an entry would re-admit the token the ratchet refuses.
  const kept = entries
    .filter((e) => found.has(burndownKey(e.file, e.token)))
    .map((e) => ({ file: e.file, token: e.token }))
    .sort((x, y) => x.file.localeCompare(y.file) || x.token.localeCompare(y.token));
  const body = { comment: LIST_COMMENT, entries: kept };
  writeFileSync(BURNDOWN_FILE, `${JSON.stringify(body, null, 2)}\n`);
  return entries.length - kept.length;
}

function main() {
  const argv = process.argv.slice(2);
  const inScopeOnly = argv.includes("--in-scope-only");
  const verbose = argv.includes("--verbose");
  const prune = argv.includes("--prune-list");
  const paths = argv.filter((a) => !a.startsWith("--"));
  const files = resolveFiles(paths, inScopeOnly);
  // A partial file set cannot tell a stale entry from one this run never opened.
  const wholeTree = paths.length === 0 && !inScopeOnly;

  let burndown;
  try {
    burndown = loadBurndown();
  } catch (err) {
    // An unreadable list is a claim this gate should judge and could not (SPEC §7.7).
    console.error(`check:citations: ${err.message}`);
    process.exit(2);
  }

  const { results, lineAnchors, unreadable } = run(REPO_ROOT, files);

  if (prune) {
    if (!wholeTree) {
      console.error("check:citations: --prune-list reads the whole tree, so it takes no path");
      process.exit(2);
    }
    console.log(`check:citations — pruned ${pruneList(lineAnchors, burndown)} burn-down entr(ies).`);
    return;
  }

  const { refused, stale } = armA(lineAnchors, burndown);
  const { anchored, verified, broken, noRow, unresolved, fatal } = armB(REPO_ROOT, results);
  const staleCount = wholeTree ? stale.length : 0;
  const of = (status) => results.filter((r) => r.status === status);
  const ok = of("ok");
  const dead = of("dead");
  const refUnknown = of("ref-unknown");
  const onRef = of("on-ref");
  const withdrawn = of("withdrawn");
  const exempt = of("exempt");
  const foreign = of("foreign");
  const untracked = of("untracked");
  const skipped = of("ignored");
  const judged = results.length - skipped.length;

  for (const r of unreadable) console.error(`check:citations: cannot read ${r.file} (${r.code})`);
  for (const f of fatal) console.error(formatFatal(f));
  for (const a of refused) console.log(formatLineAnchor(a));
  for (const r of broken) console.log(formatBroken(r));
  if (wholeTree) for (const e of stale) console.log(formatStale(e));
  for (const r of dead) console.log(formatDead(r));

  if (refUnknown.length > 0) {
    console.log("");
    console.log("Cited against a ref this clone cannot see (reported, not a violation):");
    for (const r of refUnknown) console.log(formatRefUnknown(r));
  }

  // Every bucket this gate passes over is listable, so no suppression is silent (#1436).
  const passedOver = [
    ["Absent, and the site withdraws the claim there:", withdrawn, () => "withdrawn at the site"],
    ["Absent, and exemptions.json says why:", exempt, (r) => r.reason],
    ["Addresses another project's tree, so this gate cannot judge it:", foreign, () => "not our tree"],
    ["A path .gitignore covers, so the tree never tracks it:", untracked, () => "never tracked"],
    ["Resolved against a ref the document names:", onRef, (r) => `resolves on ${r.ref}`],
    [
      "Anchored, and this gate does not resolve the path:",
      unresolved,
      (r) => `#${r.anchor} on a ${r.status} path`,
    ],
    [
      "Anchored, and the table gives this target kind no vocabulary:",
      noRow,
      (r) => `#${r.anchor} on a target kind with no row`,
    ],
  ];
  if (verbose) {
    for (const [heading, rows, note] of passedOver) {
      if (rows.length === 0) continue;
      console.log("");
      console.log(heading);
      for (const r of rows) console.log(formatNote(r, note(r)));
    }
  }

  console.log("");
  console.log(
    `check:citations — ${dead.length} dead path(s), ${broken.length} broken anchor(s) and ` +
      `${refused.length} new line anchor(s) across ${files.length} file(s).`,
  );
  const n = (rows) => String(rows.length).padStart(5);
  console.log(`  ${judged} path citation(s) judged`);
  console.log(`  ${n(ok)}  resolve in the tracked tree`);
  console.log(`  ${n(withdrawn)}  absent, and the site withdraws the claim there`);
  console.log(`  ${n(exempt)}  absent, and exemptions.json says why`);
  console.log(`  ${n(foreign)}  address another project's tree`);
  console.log(`  ${n(untracked)}  a path .gitignore covers, so the tree never tracks it`);
  console.log(`  ${n(onRef)}  resolve on a ref the document names`);
  console.log(`  ${n(refUnknown)}  name a ref this clone cannot see`);
  console.log(`  ${n(dead)}  dead`);
  console.log(`  ${skipped.length} candidate(s) were not path citations, and are not judged`);
  console.log("");
  console.log(`  ${anchored.length} anchor(s) written by a citation`);
  console.log(`  ${n(verified)}  resolve against their row`);
  console.log(`  ${n(unresolved)}  sit on a path this gate does not resolve`);
  console.log(`  ${n(noRow)}  sit on a target kind the table gives no vocabulary`);
  console.log(`  ${n(broken)}  broken: the target declares no such name`);
  if (fatal.length > 0) console.log(`  ${n(fatal)}  this gate should have judged and could not`);

  const passedOverAnchors = unresolved.length + noRow.length;
  if (!verbose && judged - ok.length - dead.length + passedOverAnchors > 0) {
    console.log("Re-run with --verbose to list every citation this gate passed over.");
  }

  console.log("");
  // The list's length sizes the remaining sweep, so a reader never opens the file (#1969).
  console.log(`  ${String(burndown.length).padStart(5)}  line anchor(s) the sweep has not reached`);
  console.log(`  ${n(lineAnchors)}  line anchor(s) found in ${files.length} file(s)`);
  console.log(`  ${n(refused)}  refused: a citation names no line`);
  if (wholeTree) console.log(`  ${n(stale)}  stale: an entry no scan finds`);
  else console.log("  the stale-entry rule needs the whole tree, and this run named paths");

  if (unreadable.length + fatal.length > 0) process.exit(2);
  if (dead.length + broken.length + refused.length + staleCount > 0) process.exit(1);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(fileURLToPath(import.meta.url))) {
  main();
}
