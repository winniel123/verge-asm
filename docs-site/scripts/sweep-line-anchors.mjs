#!/usr/bin/env node
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";
import { parse } from "./doclint/engine.mjs";
import { scanLineAnchorsFromTree } from "./citations/lineanchor.mjs";
import { environment } from "./check-citations.mjs";
import {
  loadBurndown,
  burndownKey,
  BURNDOWN_FILE,
  LIST_COMMENT,
} from "./citations/burndown.mjs";
import { derive } from "./sweep/derive.mjs";
import { rewriteDocument, trailingGlue } from "./sweep/rewrite.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");

const USAGE = `usage:
  sweep-line-anchors [<path>...]            # a dry run, and it writes nothing
  sweep-line-anchors --write <path>...      # convert those documents and delete their entries
  sweep-line-anchors --report <file> ...    # also write the plan as JSON

A <path> is a document or a directory prefix, spelled from the repository root.
A dry run with no path reads the whole burn-down list.
`;

// A --write run leaves no trace of a degradation, and SPEC §8.4 feeds a later repair effort.
export function reportRecord(results) {
  return results.map((r) => ({
    file: r.file,
    line: r.line,
    token: r.token,
    outcome: r.outcome,
    row: r.row?.name,
    ...(r.outcome === "anchor" ? { anchor: `${r.value}#${r.anchor}` } : { reason: r.reason }),
  }));
}

// A prefix reads as a directory, so `docs/spec` takes every document under it.
export function selectEntries(entries, prefixes) {
  if (prefixes.length === 0) return entries;
  return entries.filter((e) => prefixes.some((p) => e.file === p || e.file.startsWith(`${p}/`)));
}

export function scanDocuments(repoRoot, files) {
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
    const hits = scanLineAnchorsFromTree(parse(markdown)).map((h) => ({
      ...h,
      file,
      glue: trailingGlue(markdown, h),
      // The derivation reads the citing line for a name, so the scan carries it (#1977).
      lineText: lines[h.line - 1] ?? "",
    }));
    found.set(file, { markdown, hits });
  }
  return found;
}

export function planFor(repoRoot, env, entries, found, rows) {
  const wanted = new Set(entries.map((e) => burndownKey(e.file, e.token)));
  const hits = [];
  for (const [, { hits: scanned }] of found) {
    for (const hit of scanned) {
      if (wanted.has(burndownKey(hit.file, hit.token))) hits.push(hit);
    }
  }
  // An entry no scan finds already reds the gate, and the sweep converts no token it cannot see.
  const seen = new Set(hits.map((h) => burndownKey(h.file, h.token)));
  const missing = entries.filter((e) => !seen.has(burndownKey(e.file, e.token)));
  return { results: derive(repoRoot, env, hits, rows), missing };
}

function report(results, missing) {
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
    console.log("Held back — the path itself is gone, so the entry stays and a human repairs it:");
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

  if (missing.length > 0) {
    console.log("");
    console.log("Listed and unseen — no scan finds these, so the gate already refuses them:");
    for (const e of missing) console.log(`  ${e.file}  ->  ${e.token}`);
  }

  console.log("");
  const n = (rows) => String(rows.length).padStart(5);
  console.log(`sweep — ${results.length} token(s) read from the burn-down list.`);
  console.log(`  ${n(anchors)}  derive an anchor`);
  console.log(`  ${n(degradations)}  degrade to a bare path`);
  console.log(`  ${n(holds)}  held back: the path itself is gone`);
  console.log(`  ${n(missing)}  listed, and no scan finds them`);
  return { anchors, degradations, holds };
}

// A pair that converts at one site and holds at another keeps its entry (SPEC §8.2 rule 4).
function writeBurndown(entries, results) {
  const held = new Set(
    results.filter((r) => r.outcome === "held").map((r) => burndownKey(r.file, r.token)),
  );
  const gone = new Set(
    results
      .filter((r) => r.outcome !== "held")
      .map((r) => burndownKey(r.file, r.token))
      .filter((k) => !held.has(k)),
  );
  const kept = entries
    .filter((e) => !gone.has(burndownKey(e.file, e.token)))
    .map((e) => ({ file: e.file, token: e.token }))
    .sort((x, y) => x.file.localeCompare(y.file) || x.token.localeCompare(y.token));
  writeFileSync(BURNDOWN_FILE, `${JSON.stringify({ comment: LIST_COMMENT, entries: kept }, null, 2)}\n`);
  return entries.length - kept.length;
}

// A half-written tree beside an untouched list is the split SPEC §8.2 forbids.
function planWrites(results, found) {
  const byFile = new Map();
  for (const r of results) {
    if (!byFile.has(r.file)) byFile.set(r.file, []);
    byFile.get(r.file).push(r);
  }
  const writes = [];
  for (const [file, conversions] of byFile) {
    const { markdown, hits } = found.get(file);
    const next = rewriteDocument(markdown, conversions);
    const converted = hits.length - scanLineAnchorsFromTree(parse(next)).length;
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
  const args = argv.filter((a, i) => !a.startsWith("--") && i !== at + 1);
  const prefixes = args.map((p) => p.replace(/\/+$/, ""));
  if (write && prefixes.length === 0) {
    console.error("sweep: --write names the documents it rewrites, so it takes a path");
    process.exit(2);
  }

  const entries = selectEntries(loadBurndown(), prefixes);
  if (entries.length === 0) {
    console.log("sweep — the burn-down list holds no entry for that path.");
    return;
  }
  const env = environment(REPO_ROOT);
  const files = [...new Set(entries.map((e) => e.file))].sort();
  const found = scanDocuments(REPO_ROOT, files);
  const { results, missing } = planFor(REPO_ROOT, env, entries, found);
  const { anchors, degradations, holds } = report(results, missing);

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
  const writes = planWrites(converted, found);
  const documents = writeDocuments(REPO_ROOT, writes);
  const deleted = writeBurndown(loadBurndown(), results);
  console.log("");
  console.log(
    `sweep — wrote ${documents} document(s): ${anchors.length} anchor(s) and ` +
      `${degradations.length} degradation(s). Deleted ${deleted} burn-down entr(ies), ` +
      `and held ${holds.length} back.`,
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
