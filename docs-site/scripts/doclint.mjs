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
 * The skeleton (#817) shipped one rule, no-semicolons. #818 added the sentence-length
 * rule, #819 the phrasal-verb rule, and #820 the simple-tenses rule (the first warning).
 * Later tickets add the doclint-disable-line directive (#821) and the CI job (#822).
 */
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, relative, resolve } from "node:path";
import { RULES } from "./doclint/rules/index.mjs";
import { lintMarkdown, formatViolation } from "./doclint/engine.mjs";
import { inScopeFiles } from "./doclint/scope.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");

function main() {
  const args = process.argv.slice(2);
  const files =
    args.length > 0
      ? args.map((a) => resolve(process.cwd(), a))
      : inScopeFiles(REPO_ROOT);

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

  // Exit non-zero only on an error-level violation (or a file the tool could not read).
  // A warning is advisory and never changes the exit code.
  if (errors > 0 || readErrors > 0) process.exit(1);
}

main();
