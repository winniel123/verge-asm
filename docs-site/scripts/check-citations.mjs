#!/usr/bin/env node
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, relative, resolve } from "node:path";
import { inScopeFiles, isInScope } from "./citations/scope.mjs";
import { extractCitationsFromTree } from "./citations/extract.mjs";
import { parse } from "./doclint/engine.mjs";
import {
  classify,
  trackedPaths,
  trackedBasenames,
  trackedExtensions,
  topLevelEntries,
} from "./citations/classify.mjs";
import { loadExemptions, exemptionMatcher } from "./citations/exempt.mjs";
import { scanLineAnchorsFromTree } from "./citations/lineanchor.mjs";
import { armB, formatBroken, formatFatal } from "./citations/armb.mjs";
import {
  judgeSite,
  formatSiteLineAnchor,
  fetchPullBody,
  pullContext,
  apiBase,
} from "./citations/site.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");

export function environment(repoRoot, exemptionsFile) {
  const tracked = trackedPaths(repoRoot);
  const basenames = trackedBasenames(tracked);
  return {
    repoRoot,
    tracked,
    basenames,
    knownFile: (base) => basenames.has(base),
    extensions: trackedExtensions(tracked),
    roots: topLevelEntries(tracked),
    exempt: exemptionMatcher(loadExemptions(exemptionsFile)),
  };
}

export function run(repoRoot, files, env = environment(repoRoot)) {
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
    for (const a of scanLineAnchorsFromTree(tree, { knownFile: env.knownFile })) {
      lineAnchors.push({ ...a, file: docFile });
    }
  }
  return { results, lineAnchors, unreadable };
}

export function formatLineAnchor(a) {
  const where = a.kind === "link" ? "link target" : "code span";
  return `${a.file}:${a.line}  ->  ${a.token}  (a citation names no line: this ${where} must name the enclosing declaration)`;
}

// The refusal lands behind the conversion, so a form the scanner has just reached is sized
// rather than refused (SPEC docs/spec/citation-anchors.md §8.2, #2120).
export function stageOf(anchor) {
  return anchor.form === "path" ? "refused" : "staged";
}

export function formatStagedAnchor(a) {
  const what = a.form === "bare" ? "names no path" : "names no directory";
  return `  ${a.file}:${a.line}  ->  ${a.token}  (${what}, and the conversion has not reached it)`;
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

// This gate also judges the Site of a Decision block (SPEC docs/spec/citation-anchors.md §6).
export async function siteArm(env, repoRoot, verbose, deps = {}) {
  const {
    processEnv = process.env,
    readEvent = (p) => readFileSync(p, "utf8"),
    fetchImpl = fetch,
  } = deps;
  let context;
  try {
    context = pullContext(processEnv, readEvent);
  } catch (err) {
    // A payload named and unreadable is not the same claim as no payload at all (SPEC §7.7).
    console.error(`check:citations: cannot read the event payload (${err.message})`);
    return { violations: 0, fatal: 1 };
  }
  if (context === null) {
    console.log("");
    console.log("  this run has no pull-request context, so it judged no Site field");
    return { violations: 0, fatal: 0 };
  }
  let body;
  try {
    body = await fetchPullBody({
      ...context,
      token: processEnv.GITHUB_TOKEN,
      api: apiBase(processEnv),
      fetchImpl,
    });
  } catch (err) {
    // An unreadable body is a claim this gate should judge and could not (SPEC §7.7).
    console.error(`check:citations: ${err.message}`);
    return { violations: 0, fatal: 1 };
  }

  const where = `PR #${context.number} body`;
  const site = judgeSite(env, repoRoot, body, where);
  // One stage for both arms, or a Site field reds on a form the document arm reports (#2120).
  const refused = site.refused.filter((a) => stageOf(a) === "refused");
  const staged = site.refused.filter((a) => stageOf(a) === "staged");
  for (const f of site.fatal) console.error(formatFatal(f));
  for (const a of refused) console.log(formatSiteLineAnchor(a));
  for (const a of staged) console.log(formatStagedAnchor(a));
  for (const r of site.broken) console.log(formatBroken(r));
  for (const r of site.dead) console.log(formatDead(r));
  if (verbose) {
    for (const r of site.unresolved) {
      console.log(`  ${where}:${r.line}  ->  ${r.value}  (#${r.anchor} on a ${r.status} path)`);
    }
  }

  console.log("");
  const n = (rows) => String(rows.length).padStart(5);
  console.log(`  ${site.fields} Site field(s) in the body of PR #${context.number}`);
  console.log(`  ${n(site.anchored)}  write an anchor`);
  console.log(`  ${n(site.verified)}  resolve against their row`);
  console.log(`  ${n(site.broken)}  broken: the target declares no such name`);
  console.log(`  ${n(site.dead)}  dead`);
  console.log(`  ${n(refused)}  refused: a Site field names no line`);
  console.log(`  ${n(staged)}  line anchor(s) the conversion of #2120 has not reached`);
  return {
    violations: refused.length + site.broken.length + site.dead.length,
    fatal: site.fatal.length,
  };
}

async function main() {
  const argv = process.argv.slice(2);
  const inScopeOnly = argv.includes("--in-scope-only");
  const verbose = argv.includes("--verbose");
  const paths = argv.filter((a) => !a.startsWith("--"));
  const files = resolveFiles(paths, inScopeOnly);

  const env = environment(REPO_ROOT);
  const { results, lineAnchors, unreadable } = run(REPO_ROOT, files, env);

  const { anchored, verified, broken, unresolved, fatal } = armB(REPO_ROOT, results);
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

  const refusedAnchors = lineAnchors.filter((a) => stageOf(a) === "refused");
  const stagedAnchors = lineAnchors.filter((a) => stageOf(a) === "staged");

  for (const r of unreadable) console.error(`check:citations: cannot read ${r.file} (${r.code})`);
  for (const f of fatal) console.error(formatFatal(f));
  for (const a of refusedAnchors) console.log(formatLineAnchor(a));
  for (const r of broken) console.log(formatBroken(r));
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
  ];
  if (stagedAnchors.length > 0) {
    console.log("");
    console.log("A line anchor the conversion of #2120 has not reached (reported, not a violation):");
    if (verbose) for (const a of stagedAnchors) console.log(formatStagedAnchor(a));
    else console.log("  re-run with --verbose to list every one of them.");
  }
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
      `${refusedAnchors.length} line anchor(s) across ${files.length} file(s).`,
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
  console.log(`  ${n(broken)}  broken: the target declares no such name, or refuses the spelling`);
  if (fatal.length > 0) console.log(`  ${n(fatal)}  this gate should have judged and could not`);

  const passedOverAnchors = unresolved.length;
  if (!verbose && judged - ok.length - dead.length + passedOverAnchors > 0) {
    console.log("Re-run with --verbose to list every citation this gate passed over.");
  }

  console.log("");
  // The sweep is complete for the path form, and an empty count is what proves it (SPEC §8.2).
  console.log(`  ${n(refusedAnchors)}  refused: a citation names no line`);
  // What sizes the staged work is this count, and an empty one is what retires the stage (§8.2).
  console.log(`  ${n(stagedAnchors)}  line anchor(s) the conversion of #2120 has not reached`);

  const site = await siteArm(env, REPO_ROOT, verbose);

  if (unreadable.length + fatal.length + site.fatal > 0) process.exit(2);
  if (dead.length + broken.length + refusedAnchors.length + site.violations > 0) {
    process.exit(1);
  }
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(fileURLToPath(import.meta.url))) {
  main().catch((err) => {
    console.error(`check:citations: ${err.stack ?? err.message}`);
    process.exit(2);
  });
}
