#!/usr/bin/env node
import { readFileSync, appendFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, relative, resolve } from "node:path";
import { RULES } from "./doclint/rules/index.mjs";
import { lintMarkdown, formatViolation } from "./doclint/engine.mjs";
import { inScopeFiles, isInScope } from "./doclint/scope.mjs";
import { annotationLine, summaryMarkdown } from "./doclint/github.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");

function resolveFiles(paths, inScopeOnly) {
  if (paths.length === 0) {
    // CI hands this flag the raw diff, so an empty diff lints nothing (doc-lint-tool.md §5.2)
    return inScopeOnly ? [] : inScopeFiles(REPO_ROOT);
  }
  const abs = paths.map((p) => resolve(process.cwd(), p));
  return inScopeOnly ? abs.filter((a) => isInScope(REPO_ROOT, a)) : abs;
}

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

  // a warning is advisory, so it never changes the exit code (doc-lint-tool.md §5.1)
  if (errors > 0 || readErrors > 0) process.exit(1);
}

// the CI check never blocks a merge, so this mode exits zero on an error (doc-lint-tool.md §5.2)
function reportGithub(files, violations) {
  if (files.length === 0) {
    console.log("doclint: no in-scope docs changed — nothing to lint.");
  }
  for (const v of violations) console.log(annotationLine(v));

  const summary = summaryMarkdown(files.length, violations);
  const summaryFile = process.env.GITHUB_STEP_SUMMARY;
  if (summaryFile) appendFileSync(summaryFile, `${summary}\n`);
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
