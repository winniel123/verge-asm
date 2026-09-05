#!/usr/bin/env node
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, relative, resolve } from "node:path";
import { inScopeFiles, isInScope } from "./citations/scope.mjs";
import { extractCitations } from "./citations/extract.mjs";
import {
  classify,
  trackedPaths,
  trackedExtensions,
  topLevelEntries,
} from "./citations/classify.mjs";
import { loadExemptions, exemptionMatcher } from "./citations/exempt.mjs";

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
  const unreadable = [];
  for (const abs of files) {
    const docFile = relative(repoRoot, abs).replace(/\\/g, "/");
    let markdown;
    try {
      markdown = readFileSync(abs, "utf8");
    } catch (err) {
      // A changed-file list names a deleted document, so a missing file is an operator error (#1436).
      unreadable.push({ file: docFile, code: err.code ?? err.message });
      continue;
    }
    for (const r of classify(env, docFile, extractCitations(markdown))) {
      results.push({ ...r, file: docFile });
    }
  }
  return { results, unreadable };
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

function main() {
  const argv = process.argv.slice(2);
  const inScopeOnly = argv.includes("--in-scope-only");
  const verbose = argv.includes("--verbose");
  const paths = argv.filter((a) => !a.startsWith("--"));
  const files = resolveFiles(paths, inScopeOnly);

  const { results, unreadable } = run(REPO_ROOT, files);
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
  console.log(`check:citations — ${dead.length} dead path(s) across ${files.length} file(s).`);
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
  if (!verbose && judged - ok.length - dead.length > 0) {
    console.log("Re-run with --verbose to list every citation this gate passed over.");
  }

  if (unreadable.length > 0) process.exit(2);
  if (dead.length > 0) process.exit(1);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(fileURLToPath(import.meta.url))) {
  main();
}
