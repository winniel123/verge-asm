/*
 * Candidate measurement harness (#824, the #792/#823 method).
 *
 * The prove-before-enable process (SPEC §2.5) is: build the detector, prove it on the fixture
 * corpus (SPEC §6), then measure it on the real in-scope docs to score precision, then enable
 * or defer. This harness runs the two measurements for each #824 candidate, so a person can
 * read the fixture-corpus result and a sample of real flags and make the enable-or-defer call.
 *
 * Run from the docs-site directory:  node scripts/doclint/candidates/measure.mjs
 *
 * It is a throwaway measurement aid, like the #792 prototype. It ships no rule and changes no
 * doc. It only reports numbers.
 */
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, relative, resolve } from "node:path";
import { lintMarkdown } from "../engine.mjs";
import { checkRuleCorpus } from "../fixtures.mjs";
import { inScopeFiles } from "../scope.mjs";
import { oneInstruction } from "./one-instruction.mjs";
import { noEllipsis } from "./no-ellipsis.mjs";

const HERE = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(HERE, "..", "..", "..", "..");
const CANDIDATES = [oneInstruction, noEllipsis];
const SAMPLE = 25; // flags to print per candidate, for a manual precision score

/**
 * The fixture-corpus result for one candidate (SPEC §6): does it flag every must-flag snippet
 * and no must-not-flag snippet?
 * @param {import("../engine.mjs").Rule} rule
 */
function reportCorpus(rule) {
  const failures = checkRuleCorpus(rule);
  if (failures.length === 0) {
    console.log(`  fixture corpus: PASS (flags every must-flag, no must-not-flag)`);
    return;
  }
  console.log(`  fixture corpus: FAIL (${failures.length} case(s))`);
  for (const f of failures) {
    console.log(`    [${f.kind}] got ${f.got} for ${JSON.stringify(f.snippet)}`);
  }
}

/**
 * The real-doc sweep for one candidate (the #792 method): run it over every in-scope doc, count
 * the flags, and print a sample of flagged lines for a manual precision score.
 * @param {import("../engine.mjs").Rule} rule
 * @param {string[]} files  absolute in-scope file paths.
 */
function reportRealDocs(rule, files) {
  const hits = [];
  let fileCount = 0;
  for (const abs of files) {
    const rel = relative(REPO_ROOT, abs).replace(/\\/g, "/");
    let markdown;
    try {
      markdown = readFileSync(abs, "utf8");
    } catch {
      continue;
    }
    const lines = markdown.split("\n");
    const violations = lintMarkdown(markdown, [rule]);
    if (violations.length > 0) fileCount++;
    for (const v of violations) {
      const text = (lines[v.line - 1] ?? "").trim().slice(0, 120);
      hits.push({ file: rel, line: v.line, text });
    }
  }
  console.log(`  real docs: ${hits.length} flag(s) across ${fileCount} file(s) of ${files.length}`);
  const step = Math.max(1, Math.floor(hits.length / SAMPLE));
  const sample = hits.filter((_, i) => i % step === 0).slice(0, SAMPLE);
  console.log(`  sample (${sample.length} of ${hits.length}, evenly spaced):`);
  for (const h of sample) console.log(`    ${h.file}:${h.line}\n      | ${h.text}`);
}

function main() {
  const files = inScopeFiles(REPO_ROOT);
  for (const rule of CANDIDATES) {
    console.log(`\n=== ${rule.name} (${rule.severity}) ===`);
    reportCorpus(rule);
    reportRealDocs(rule, files);
  }
  console.log("");
}

main();
