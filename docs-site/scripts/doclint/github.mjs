// GitHub reads a bare %, CR or LF in a workflow command as syntax, so each one is encoded.
function escapeData(value) {
  return String(value).replace(/%/g, "%25").replace(/\r/g, "%0D").replace(/\n/g, "%0A");
}

// A property value also carries GitHub's own separators, so "," and ":" escape on top of these.
function escapeProperty(value) {
  return escapeData(value).replace(/:/g, "%3A").replace(/,/g, "%2C");
}

export function annotationLine(v) {
  const level = v.severity === "error" ? "error" : "warning";
  const file = escapeProperty(v.file);
  const title = escapeProperty(`doclint (${v.rule})`);
  return `::${level} file=${file},line=${v.line},title=${title}::${escapeData(v.message)}`;
}

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
