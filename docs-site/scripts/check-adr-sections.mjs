#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { appendFileSync, readFileSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { basename, dirname, join, relative, resolve } from "node:path";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = resolve(SCRIPT_DIR, "..", "..");
const ADR_DIR = "docs/adr";

const ADR_FILE = /^(\d{4})-.+\.md$/;
const FENCE = /^\s{0,3}(?:```|~~~)/;
const NUMBERED_HEADING = /^#{2,6}\s+(\d+(?:\.\d+)*)[.)]?\s+\S/;

// #1455's repairs wrote a comma, ADR-0038 a semicolon, and reports.md:253 wraps before its §3
const SEPARATOR = "[,;]?(?:[ \\t\\u00a0]|\\n[ \\t]*)?";
const SECTION = "\\u00a7(\\d+(?:\\.\\d+)*)";

// A bare citation, the form CLAUDE.md fixes for a surviving comment.
const BARE_CITATION = new RegExp(`ADR-(\\d{4})${SEPARATOR}${SECTION}`, "g");
// The same citation written as a Markdown link, the form an ADR cross-reference uses.
const LINKED_CITATION = new RegExp(
  `\\[ADR-(\\d{4})\\]\\([^)\\s]*\\)${SEPARATOR}${SECTION}`,
  "g",
);
// §4.4 puts the issue number after the section, so a section on the issue inverts it (#1455)
const ISSUE_SECTION_CITATION = new RegExp(
  `ADR-(\\d{4})${SEPARATOR}#(\\d+)[ \\t\\u00a0]?${SECTION}`,
  "g",
);

const TEXT_EXTENSIONS = new Set([
  "md",
  "go",
  "sql",
  "mjs",
  "js",
  "jsx",
  "ts",
  "tsx",
  "tmpl",
  "html",
  "css",
  "yml",
  "yaml",
  "toml",
  "json",
  "sh",
  "txt",
]);

// deploy/prober/Dockerfile carries a citation in a comment and has no extension to match on.
const TEXT_BASENAMES = new Set(["Dockerfile", "Containerfile", "Makefile"]);

// This one file's citations are all fixtures, and .mjs has no code span to quote them with (#1437).
const SELF_TEST = "docs-site/scripts/check-adr-sections.test.mjs";

export function numberedSections(markdown) {
  const sections = new Set();
  let fenced = false;
  for (const line of markdown.split(/\r?\n/)) {
    if (FENCE.test(line)) {
      fenced = !fenced;
      continue;
    }
    if (fenced) continue;
    const m = NUMBERED_HEADING.exec(line);
    if (m) sections.add(m[1]);
  }
  return sections;
}

export function buildAdrIndex(repoRoot) {
  const index = new Map();
  let names;
  try {
    names = readdirSync(join(repoRoot, ADR_DIR));
  } catch {
    return index;
  }
  for (const name of names.sort()) {
    const m = ADR_FILE.exec(name);
    if (!m) continue;
    const file = `${ADR_DIR}/${name}`;
    const text = readFileSync(join(repoRoot, file), "utf8");
    index.set(m[1], { file, sections: numberedSections(text) });
  }
  return index;
}

// A code span quotes a specimen, and "~" stops a section reaching across it (comment-policy §4.7)
function maskCodeSpans(line) {
  return line.replace(/(`+)[^`]*?\1/g, (span) => "~".repeat(span.length));
}

// Masking keeps every line its own length, so a match offset still maps to the line it came from
function maskLines(lines, markdown) {
  if (!markdown) return lines;
  let fenced = false;
  return lines.map((line) => {
    if (FENCE.test(line)) {
      fenced = !fenced;
      return "~".repeat(line.length);
    }
    return fenced ? "~".repeat(line.length) : maskCodeSpans(line);
  });
}

function lineOf(starts, offset) {
  let lo = 0;
  let hi = starts.length - 1;
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1;
    if (starts[mid] <= offset) lo = mid;
    else hi = mid - 1;
  }
  return lo + 1;
}

export function findCitations(text, { markdown }) {
  const lines = maskLines(text.split(/\r?\n/), markdown);
  const starts = [];
  let at = 0;
  for (const line of lines) {
    starts.push(at);
    at += line.length + 1;
  }
  // The whole text is one subject, so a citation the author wrapped still reads as one citation
  const joined = lines.join("\n");

  const found = [];
  ISSUE_SECTION_CITATION.lastIndex = 0;
  let issue;
  while ((issue = ISSUE_SECTION_CITATION.exec(joined)) !== null) {
    found.push({
      line: lineOf(starts, issue.index),
      adr: issue[1],
      issue: issue[2],
      section: issue[3],
      text: `ADR-${issue[1]}, #${issue[2]} §${issue[3]}`,
    });
  }
  // An intervening token takes the section: in "ADR-0108, ADR-0180 §3" it is ADR-0180's (§4.7)
  for (const pattern of [LINKED_CITATION, BARE_CITATION]) {
    pattern.lastIndex = 0;
    let m;
    while ((m = pattern.exec(joined)) !== null) {
      found.push({
        line: lineOf(starts, m.index),
        adr: m[1],
        section: m[2],
        text: `ADR-${m[1]} §${m[2]}`,
      });
    }
  }
  return found.sort((a, b) => a.line - b.line);
}

export function checkFile(file, text, index) {
  const markdown = file.endsWith(".md");
  const violations = [];
  for (const c of findCitations(text, { markdown })) {
    const target = index.get(c.adr);
    if (!target) {
      violations.push({
        ...c,
        file,
        rule: "unresolvable-adr",
        message: `${c.text} cites no ADR on disk`,
      });
      continue;
    }
    if (c.issue !== undefined) {
      violations.push({
        ...c,
        file,
        rule: "section-on-issue-number",
        message:
          `${c.text} hangs a section on issue #${c.issue}, ` +
          `which §4.4's grammar does not allow`,
      });
      continue;
    }
    if (target.sections.size === 0) {
      violations.push({
        ...c,
        file,
        rule: "unnumbered-adr",
        message: `${c.text} is wrong by construction: ${target.file} numbers no heading`,
      });
      continue;
    }
    if (!target.sections.has(c.section)) {
      const have = [...target.sections].join(", ");
      violations.push({
        ...c,
        file,
        rule: "section-out-of-range",
        message: `${c.text} names no section of ${target.file}, which numbers ${have}`,
      });
    }
  }
  return violations;
}

export function isTextFile(path) {
  if (path === SELF_TEST) return false;
  if (TEXT_BASENAMES.has(basename(path))) return true;
  const parts = basename(path).split(".");
  return parts.length > 1 && TEXT_EXTENSIONS.has(parts.pop().toLowerCase());
}

export function trackedFiles(repoRoot) {
  const out = execFileSync("git", ["ls-files", "-z"], {
    cwd: repoRoot,
    encoding: "utf8",
    maxBuffer: 256 * 1024 * 1024,
  });
  return out.split("\0").filter(Boolean).filter(isTextFile);
}

// GitHub reads a bare %, CR or LF in a workflow command as syntax, so each one is encoded.
function escapeData(value) {
  return String(value).replace(/%/g, "%25").replace(/\r/g, "%0D").replace(/\n/g, "%0A");
}

function escapeProperty(value) {
  return escapeData(value).replace(/:/g, "%3A").replace(/,/g, "%2C");
}

export function annotationLine(v) {
  const title = escapeProperty(`check:adr-sections (${v.rule})`);
  const file = escapeProperty(v.file);
  return `::error file=${file},line=${v.line},title=${title}::${escapeData(v.message)}`;
}

export function readErrorLine(file, reason) {
  const title = escapeProperty("check:adr-sections (unreadable-file)");
  return `::error file=${escapeProperty(file)},line=1,title=${title}::${escapeData(
    `cannot read ${file} (${reason}), so no citation in it was checked`,
  )}`;
}

export function summaryMarkdown(fileCount, violations, readErrors = []) {
  const lines = ["## check:adr-sections", ""];
  lines.push(
    "A `§n` citation resolves to a numbered heading (SPEC docs/spec/comment-policy.md §4.7).",
  );
  lines.push("");
  lines.push(`**${fileCount} file(s) scanned, ${violations.length} violation(s).**`);
  lines.push("");
  // A read failure is not a clean run, so the summary says it rather than leaving the red X bare
  if (readErrors.length > 0) {
    lines.push(`**${readErrors.length} file(s) could not be read, and none of them was checked.**`);
    lines.push("");
    for (const e of readErrors) lines.push(`- \`${e.file}\` (${e.reason})`);
    lines.push("");
  }
  if (violations.length === 0) return lines.join("\n");

  const byRule = new Map();
  for (const v of violations) byRule.set(v.rule, (byRule.get(v.rule) ?? 0) + 1);
  const rows = [...byRule.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
  lines.push("### By rule", "", "| Rule | Count |", "| --- | --- |");
  for (const [rule, count] of rows) lines.push(`| ${rule} | ${count} |`);
  lines.push("");
  return lines.join("\n");
}

function main() {
  const argv = process.argv.slice(2);
  const github = argv.includes("--github");
  const paths = argv.filter((a) => !a.startsWith("--"));
  const repoRoot = DEFAULT_ROOT;

  const index = buildAdrIndex(repoRoot);
  if (index.size === 0) {
    console.error(`check:adr-sections: no ADR found under ${ADR_DIR}`);
    process.exit(2);
  }

  const files = (
    paths.length > 0
      ? paths.map((p) => relative(repoRoot, resolve(process.cwd(), p)).replace(/\\/g, "/"))
      : trackedFiles(repoRoot)
  ).filter((f) => f !== SELF_TEST);

  const violations = [];
  const readErrors = [];
  for (const file of files) {
    let text;
    try {
      text = readFileSync(join(repoRoot, file), "utf8");
    } catch (err) {
      const reason = err.code ?? err.message;
      console.error(`check:adr-sections: cannot read ${file} (${reason})`);
      readErrors.push({ file, reason });
      continue;
    }
    violations.push(...checkFile(file, text, index));
  }

  if (github) {
    for (const v of violations) console.log(annotationLine(v));
    for (const e of readErrors) console.log(readErrorLine(e.file, e.reason));
    const summary = summaryMarkdown(files.length, violations, readErrors);
    if (process.env.GITHUB_STEP_SUMMARY) {
      appendFileSync(process.env.GITHUB_STEP_SUMMARY, `${summary}\n`);
    }
    console.log(summary);
  } else if (violations.length === 0 && readErrors.length === 0) {
    console.log(
      `check:adr-sections OK — ${files.length} file(s), ${index.size} ADR(s), no violations.`,
    );
  } else {
    for (const v of violations) console.log(`${v.file}:${v.line}  ->  ${v.rule}  (${v.message})`);
    console.log("");
    console.log(
      `check:adr-sections — ${violations.length} violation(s) across ${files.length} file(s), ` +
        `${index.size} ADR(s), ${readErrors.length} unreadable.`,
    );
  }

  if (violations.length > 0 || readErrors.length > 0) process.exit(1);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
