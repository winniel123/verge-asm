/*
 * GitHub Actions output for the CI job (SPEC §5.2, #822).
 *
 * The `doclint` CI job runs the tool on a pull request and reports in two channels:
 *
 *   1. An inline annotation per violation, so a reader sees each flag on the exact line of
 *      the pull-request diff. GitHub reads a `::error`/`::warning` workflow command on stdout
 *      and renders it inline.
 *   2. A job-log summary of counts by rule and by severity, written to the step summary.
 *
 * Both builders are pure, so the tests read them directly. doclint.mjs wires them behind the
 * `--github` flag. There is no SARIF in v1 (SPEC §5.2).
 */

/**
 * Escape the data segment of a workflow command. GitHub reads a literal `%`, CR, or LF in the
 * message as command syntax, so each one is percent-encoded. The message reaches the reader
 * after the `::`, so only these three need escaping there.
 * @param {string} value
 * @returns {string}
 */
function escapeData(value) {
  return String(value).replace(/%/g, "%25").replace(/\r/g, "%0D").replace(/\n/g, "%0A");
}

/**
 * Escape a property value of a workflow command (the `file=` and `title=` values). A property
 * value additionally must not carry a `,` (the property separator) or a `:` (which ends the
 * property list), so both are percent-encoded on top of the data escapes.
 * @param {string} value
 * @returns {string}
 */
function escapeProperty(value) {
  return escapeData(value).replace(/:/g, "%3A").replace(/,/g, "%2C");
}

/**
 * One workflow-command annotation for a violation. An error-severity violation becomes an
 * `::error` command and a warning becomes a `::warning` command, so GitHub renders the two at
 * the matching level inline on the pull request.
 * @param {import("../engine.mjs").Violation & {file: string}} v
 * @returns {string}
 */
export function annotationLine(v) {
  const level = v.severity === "error" ? "error" : "warning";
  const file = escapeProperty(v.file);
  const title = escapeProperty(`doclint (${v.rule})`);
  return `::${level} file=${file},line=${v.line},title=${title}::${escapeData(v.message)}`;
}

/**
 * The job-log summary (SPEC §5.2): counts by rule and by severity. The summary is markdown,
 * so it renders in the GitHub Actions step-summary panel and reads plainly in a raw log.
 * @param {number} fileCount how many files the job linted.
 * @param {(import("../engine.mjs").Violation & {file: string})[]} violations
 * @returns {string}
 */
export function summaryMarkdown(fileCount, violations) {
  const errors = violations.filter((v) => v.severity === "error").length;
  const warnings = violations.length - errors;

  const lines = [];
  lines.push("## doclint");
  lines.push("");
  lines.push("Advisory documentation lint (SPEC §5.2). This check never blocks a merge.");
  lines.push("");

  if (violations.length === 0) {
    lines.push(`**${fileCount} file(s) linted, no violations.**`);
    lines.push("");
    return lines.join("\n");
  }

  lines.push(
    `**${fileCount} file(s) linted, ${violations.length} violation(s): ` +
      `${errors} error(s), ${warnings} warning(s).**`,
  );
  lines.push("");

  // Counts by rule, one row per rule that fired, most-frequent first, then by name. A rule
  // carries one fixed severity (the Rule.severity field), so the map keys by rule name alone
  // and keeps the severity of the first hit — no composite key to split back apart.
  const byRule = new Map();
  for (const v of violations) {
    const row = byRule.get(v.rule);
    if (row) row.count++;
    else byRule.set(v.rule, { rule: v.rule, severity: v.severity, count: 1 });
  }
  const ruleRows = [...byRule.values()].sort(
    (a, b) => b.count - a.count || a.rule.localeCompare(b.rule),
  );

  lines.push("### By rule");
  lines.push("");
  lines.push("| Rule | Severity | Count |");
  lines.push("| --- | --- | --- |");
  for (const r of ruleRows) lines.push(`| ${r.rule} | ${r.severity} | ${r.count} |`);
  lines.push("");

  lines.push("### By severity");
  lines.push("");
  lines.push("| Severity | Count |");
  lines.push("| --- | --- |");
  lines.push(`| error | ${errors} |`);
  lines.push(`| warning | ${warnings} |`);
  lines.push("");

  return lines.join("\n");
}
