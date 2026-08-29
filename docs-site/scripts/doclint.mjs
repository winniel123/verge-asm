#!/usr/bin/env node
/*
 * doclint — the documentation lint tool (SPEC docs/spec/doc-lint-tool.md).
 *
 * An advisory linter for the mechanical STE rules from the documentation style
 * standard. It runs in two modes (SPEC §5.1):
 *
 *   node scripts/doclint.mjs               lint every in-scope file (SPEC §1.3)
 *   node scripts/doclint.mjs a.md b.md     lint the named files
 *
 * It prints one line per violation in the `file:line  ->  rule  (severity: reason)`
 * style, modeled on check-links.mjs. It exits non-zero when it finds an error-level
 * violation. A warning alone never changes the exit code, because a warning is advisory.
 *
 * Two flags drive the CI job (SPEC §5.2, #822):
 *
 *   --github          report GitHub Actions annotations plus a job-log summary, instead
 *                     of the human lines. The mode is advisory: a violation never changes
 *                     the exit code, because the CI check never blocks a merge.
 *   --in-scope-only   filter the named files to the SPEC §1.3 set. The job hands the tool
 *                     the raw pull-request diff, so an out-of-scope doc never reaches a rule.
 *
 * The skeleton (#817) shipped one rule, no-semicolons. #818 added the sentence-length
 * rule, #819 the phrasal-verb rule, and #820 the simple-tenses rule (the first warning).
 * #821 added the doclint-disable-line directive (an engine-level filter, SPEC §6). #822
 * added the CI job and these two flags.
 */
import { readFileSync, appendFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, relative, resolve } from "node:path";
import { RULES } from "./doclint/rules/index.mjs";
import { lintMarkdown, formatViolation } from "./doclint/engine.mjs";
import { inScopeFiles, isInScope } from "./doclint/scope.mjs";
import { annotationLine, summaryMarkdown } from "./doclint/github.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");

/**
 * Resolve the file list from the CLI arguments. With one or more named paths, lint those.
 * With `--in-scope-only`, drop any path the SPEC §1.3 set does not cover. With no path and
 * no `--in-scope-only`, lint the whole in-scope tree (the writer's no-argument path). The
 * `--in-scope-only` flag never triggers the whole-tree fallback: an empty diff lints nothing.
 * @param {string[]} paths the positional path arguments.
 * @param {boolean} inScopeOnly whether the --in-scope-only flag is set.
 * @returns {string[]} absolute file paths.
 */
function resolveFiles(paths, inScopeOnly) {
  if (paths.length === 0) {
    // The whole-tree fallback is the no-argument writer path only. A diff-scoped CI run passes
    // --in-scope-only, so an empty changed set lints nothing rather than the whole tree.
    return inScopeOnly ? [] : inScopeFiles(REPO_ROOT);
  }
  const abs = paths.map((p) => resolve(process.cwd(), p));
  return inScopeOnly ? abs.filter((a) => isInScope(REPO_ROOT, a)) : abs;
}

/**
 * Lint every file and collect the violations, each tagged with its repo-relative path.
 * @param {string[]} files absolute file paths.
 * @returns {{violations: object[], readErrors: number}}
 */
function collect(files) {
  const violations = [];
  let readErrors = 0;
  for (const abs of files) {
    const file = relative(REPO_ROOT, abs).replace(/\\/g, "/");
    let markdown;
    try {
      markdown = readFileSync(abs, "utf8");
    } catch (err) {
      console.error(`doclint: cannot read ${file} (${err.code ?? err.message})`);
      readErrors++;
      continue;
    }
    for (const v of lintMarkdown(markdown, RULES)) {
      violations.push({ ...v, file });
    }
  }
  return { violations, readErrors };
}

/**
 * The human reporter (SPEC §5.1): one line per violation, then a count summary. Exits non-zero
 * on an error-level violation or a read error, so a writer's shell gate fails on a real error.
 * A warning alone never changes the exit code.
 */
function reportHuman(files, violations, readErrors) {
  const errors = violations.filter((v) => v.severity === "error").length;
  const warnings = violations.length - errors;

  if (violations.length === 0 && readErrors === 0) {
    console.log(`check:doclint OK — ${files.length} file(s), no violations.`);
    return;
  }

  for (const v of violations) console.log(formatViolation(v));
  console.log("");
  console.log(
    `check:doclint — ${errors} error(s), ${warnings} warning(s) across ${files.length} file(s).`,
  );

  if (errors > 0 || readErrors > 0) process.exit(1);
}

/**
 * The GitHub Actions reporter (SPEC §5.2): one annotation per violation on stdout, and a
 * summary of counts by rule and by severity. The summary goes to GITHUB_STEP_SUMMARY when the
 * env var is set (the CI step-summary panel), and to stdout otherwise (a local run or a test).
 *
 * The mode is fully advisory: it always exits zero, whatever it reports. The CI check never
 * blocks a merge (SPEC §5.2), so the exit code carries no signal a merge gate could read. A
 * read error is not silent, though: collect() already prints it to stderr, so it shows in the
 * job log. The `readErrors` count is unused here for that reason.
 */
function reportGithub(files, violations) {
  if (files.length === 0) {
    console.log("doclint: no in-scope docs changed — nothing to lint.");
  }
  for (const v of violations) console.log(annotationLine(v));

  const summary = summaryMarkdown(files.length, violations);
  const summaryFile = process.env.GITHUB_STEP_SUMMARY;
  if (summaryFile) appendFileSync(summaryFile, `${summary}\n`);
  // Always echo the summary to stdout too, so the job log carries the counts even when the
  // step-summary panel is not available (SPEC §5.2: the job log prints the summary).
  console.log(summary);
}

function main() {
  const argv = process.argv.slice(2);
  const github = argv.includes("--github");
  const inScopeOnly = argv.includes("--in-scope-only");
  const paths = argv.filter((a) => !a.startsWith("--"));

  const files = resolveFiles(paths, inScopeOnly);
  const { violations, readErrors } = collect(files);

  if (github) reportGithub(files, violations);
  else reportHuman(files, violations, readErrors);
}

main();
