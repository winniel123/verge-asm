import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  annotationLine,
  buildAdrIndex,
  checkFile,
  findCitations,
  isTextFile,
  numberedSections,
  readErrorLine,
  summaryMarkdown,
} from "./check-adr-sections.mjs";

const ADR_WITH_SECTIONS = [
  "# ADR-0129: A title",
  "",
  "## Context",
  "",
  "## Decision",
  "",
  "### 1. The first rule",
  "",
  "### 2. The second rule",
  "",
  "#### 2.1 A subsection",
  "",
  "## Consequences",
  "",
].join("\n");

const ADR_WITHOUT_SECTIONS = [
  "# ADR-0029: A title",
  "",
  "## Context",
  "",
  "## Decision",
  "",
  "### An unnumbered rule",
  "",
  "## Consequences",
  "",
].join("\n");

function withRepo(files, run) {
  const root = mkdtempSync(join(tmpdir(), "adr-sections-"));
  try {
    mkdirSync(join(root, "docs/adr"), { recursive: true });
    for (const [rel, body] of Object.entries(files)) {
      writeFileSync(join(root, rel), body);
    }
    run(root);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

test("a numbered heading is indexed by its number path", () => {
  const sections = numberedSections(ADR_WITH_SECTIONS);
  assert.deepEqual([...sections].sort(), ["1", "2", "2.1"]);
});

test("an ADR that numbers no heading indexes an empty set", () => {
  assert.equal(numberedSections(ADR_WITHOUT_SECTIONS).size, 0);
});

test("a numbered heading inside a fence is not a section", () => {
  const md = ["## Decision", "", "```md", "### 1. Not a heading here", "```", ""].join("\n");
  assert.equal(numberedSections(md).size, 0);
});

test("a heading number needs a title after it", () => {
  assert.equal(numberedSections("### 1.\n").size, 0);
});

test("a bare citation is found", () => {
  const found = findCitations("no gate reads it (ADR-0129 §2.1).", { markdown: false });
  assert.deepEqual(found, [{ line: 1, adr: "0129", section: "2.1", text: "ADR-0129 §2.1" }]);
});

test("a Markdown link citation is found", () => {
  const found = findCitations("see [ADR-0129](./0129-a.md) §2 for the rule", { markdown: true });
  assert.deepEqual(found, [{ line: 1, adr: "0129", section: "2", text: "ADR-0129 §2" }]);
});

test("a comma between the ADR and the section is a separator, not a break", () => {
  const found = findCitations("(ADR-0083, §3.5)", { markdown: false });
  assert.deepEqual(found, [{ line: 1, adr: "0083", section: "3.5", text: "ADR-0083 §3.5" }]);
});

test("a semicolon separates too", () => {
  const found = findCitations("ADR-0038; §39.4 item 8", { markdown: false });
  assert.deepEqual(found, [{ line: 1, adr: "0038", section: "39.4", text: "ADR-0038 §39.4" }]);
});

test("a word between the ADR and the section binds the section to that word", () => {
  assert.deepEqual(findCitations("(ADR-0053, spec §2.4)", { markdown: false }), []);
});

test("a second ADR takes the section, and the first is left alone", () => {
  const found = findCitations("(ADR-0108, ADR-0180 §3)", { markdown: false });
  assert.deepEqual(found, [{ line: 1, adr: "0180", section: "3", text: "ADR-0180 §3" }]);
});

test("a section on an issue number inside an ADR citation is found", () => {
  const found = findCitations("(ADR-0126, #1321 §3)", { markdown: false });
  assert.deepEqual(found, [
    { line: 1, adr: "0126", issue: "1321", section: "3", text: "ADR-0126, #1321 §3" },
  ]);
});

test("a section on an issue number outside an ADR citation is left alone", () => {
  assert.deepEqual(findCitations("// a read-side floor (#715 §6)", { markdown: false }), []);
});

test("Dockerfile carries a citation, and its name has no extension", () => {
  assert.equal(isTextFile("deploy/prober/Dockerfile"), true);
  assert.equal(isTextFile("internal/x/y.go"), true);
  assert.equal(isTextFile("web/logo.png"), false);
});

test("this suite's own fixtures are out of scope, and no other .mjs is", () => {
  assert.equal(isTextFile("docs-site/scripts/check-adr-sections.test.mjs"), false);
  assert.equal(isTextFile("docs-site/scripts/check-adr-sections.mjs"), true);
  assert.equal(isTextFile("docs-site/scripts/doclint.test.mjs"), true);
});

test("a section that wraps to the next line still belongs to its ADR", () => {
  const md = [
    "**ADR-0180 is not authority for this rule.**",
    "[ADR-0180](../adr/0180-a-message-detail.md)",
    "§3 states the same sentence about a **message** detail.",
  ].join("\n");
  const found = findCitations(md, { markdown: true });
  assert.deepEqual(found, [{ line: 2, adr: "0180", section: "3", text: "ADR-0180 §3" }]);
});

test("a bare citation wraps too, and the line is where the ADR sits", () => {
  const found = findCitations("a reason (ADR-0064\n§2)", { markdown: false });
  assert.deepEqual(found, [{ line: 1, adr: "0064", section: "2", text: "ADR-0064 §2" }]);
});

test("a blank line between them is not a wrap", () => {
  const md = "[ADR-0180](./x.md)\n\n§3 opens a new thought";
  assert.deepEqual(findCitations(md, { markdown: true }), []);
});

test("a fenced block stays skipped once the whole text is one subject", () => {
  const md = ["before", "```go", "// r (ADR-0129 §9)", "```", "after"].join("\n");
  assert.deepEqual(findCitations(md, { markdown: true }), []);
});

test("a citation inside a Markdown code span is a specimen, not a citation", () => {
  assert.deepEqual(findCitations("`(ADR-0129 §9)` was repaired", { markdown: true }), []);
});

test("a citation inside a Markdown fence is a specimen, not a citation", () => {
  const md = ["```go", "// a reason (ADR-0129 §9)", "```", ""].join("\n");
  assert.deepEqual(findCitations(md, { markdown: true }), []);
});

test("a code span does not hide a citation off Markdown", () => {
  const found = findCitations("// `x` is set (ADR-0129 §9)", { markdown: false });
  assert.equal(found.length, 1);
});

test("a citation to a section the ADR numbers is clean", () => {
  withRepo(
    {
      "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS,
      "docs/adr/0029-a-title.md": ADR_WITHOUT_SECTIONS,
    },
    (root) => {
      const index = buildAdrIndex(root);
      assert.deepEqual(checkFile("internal/x/y.go", "// r (ADR-0129 §2.1)", index), []);
    },
  );
});

test("a citation to an ADR that numbers no heading is wrong by construction", () => {
  withRepo({ "docs/adr/0029-a-title.md": ADR_WITHOUT_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const found = checkFile("internal/x/y.go", "// r (ADR-0029 §5)", index);
    assert.equal(found.length, 1);
    assert.equal(found[0].rule, "unnumbered-adr");
    assert.equal(found[0].line, 1);
    assert.match(found[0].message, /numbers no heading/);
  });
});

test("a citation past the last numbered section is out of range", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const found = checkFile("docs/spec/x.md", "The rule (ADR-0129 §5) holds.", index);
    assert.equal(found.length, 1);
    assert.equal(found[0].rule, "section-out-of-range");
    assert.match(found[0].message, /numbers 1, 2, 2\.1/);
  });
});

test("a subsection the ADR does not number is out of range even when its parent exists", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const found = checkFile("docs/spec/x.md", "The rule (ADR-0129 §2.4) holds.", index);
    assert.equal(found.length, 1);
    assert.equal(found[0].rule, "section-out-of-range");
  });
});

test("a citation to an ADR that is not on disk is unresolvable", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const found = checkFile("docs/spec/x.md", "The rule (ADR-0999 §1) holds.", index);
    assert.equal(found.length, 1);
    assert.equal(found[0].rule, "unresolvable-adr");
  });
});

test("a named section carries no number, so it is out of this check", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    assert.deepEqual(checkFile("docs/spec/x.md", "ADR-0129 §Context says so.", index), []);
  });
});

// The pre-repair text of the sites #1455 repaired, so a pass proves the check would have caught it
const REPAIRED_BY_1455 = [
  {
    name: "internal/vergecore/vergecore.go carried (ADR-0083, §3.5)",
    file: "internal/vergecore/vergecore.go",
    text: "// the ceiling is the product's, never the operator's (ADR-0083, §3.5)",
    adr: "0083",
    rule: "unnumbered-adr",
  },
  {
    name: "internal/queue/transcript.go carried (ADR-0126, #1321 §3)",
    file: "internal/queue/transcript.go",
    text: "// the transcript is written once and never amended (ADR-0126, #1321 §3)",
    adr: "0126",
    rule: "section-on-issue-number",
  },
];

for (const site of REPAIRED_BY_1455) {
  test(`#1455: ${site.name}`, () => {
    withRepo({ [`docs/adr/${site.adr}-a-title.md`]: ADR_WITHOUT_SECTIONS }, (root) => {
      const index = buildAdrIndex(root);
      const found = checkFile(site.file, site.text, index);
      assert.equal(found.length, 1);
      assert.equal(found[0].rule, site.rule);
    });
  });
}

test("#1455: cmd/worker/main.go carried (ADR-0053, spec §2.4), which is out of scope", () => {
  withRepo({ "docs/adr/0053-a-title.md": ADR_WITHOUT_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const text = "// the operator key lives on the worker alone (ADR-0053, spec §2.4)";
    assert.deepEqual(checkFile("cmd/worker/main.go", text, index), []);
  });
});

test("an annotation escapes the separators GitHub reads as syntax", () => {
  const line = annotationLine({
    file: "docs/spec/x.md",
    line: 7,
    rule: "unnumbered-adr",
    message: "ADR-0029 §5: 100% wrong",
  });
  assert.equal(
    line,
    "::error file=docs/spec/x.md,line=7,title=check%3Aadr-sections (unnumbered-adr)" +
      "::ADR-0029 §5: 100%25 wrong",
  );
});

test("the clean summary reports the file count and no rule table", () => {
  const summary = summaryMarkdown(12, []);
  assert.match(summary, /\*\*12 file\(s\) scanned, 0 violation\(s\)\.\*\*/);
  assert.equal(summary.includes("### By rule"), false);
  assert.equal(summary.includes("could not be read"), false);
});

test("a read failure reaches the summary, so no red check reports a clean run", () => {
  const summary = summaryMarkdown(12, [], [{ file: "docs/adr/0001-x.md", reason: "EACCES" }]);
  assert.match(summary, /\*\*1 file\(s\) could not be read, and none of them was checked\.\*\*/);
  assert.match(summary, /- `docs\/adr\/0001-x\.md` \(EACCES\)/);
});

test("a read failure also gets its own annotation", () => {
  const line = readErrorLine("docs/adr/0001-x.md", "EACCES");
  assert.match(line, /^::error file=docs\/adr\/0001-x\.md,line=1,/);
  assert.match(line, /cannot read docs\/adr\/0001-x\.md \(EACCES\)/);
  assert.match(line, /so no citation in it was checked$/);
});

test("the summary counts by rule, most frequent first", () => {
  const summary = summaryMarkdown(2, [
    { rule: "unnumbered-adr" },
    { rule: "unnumbered-adr" },
    { rule: "section-out-of-range" },
  ]);
  const rows = summary.split("\n").filter((l) => l.startsWith("| ") && !l.startsWith("| Rule"));
  assert.deepEqual(rows.slice(1), ["| unnumbered-adr | 2 |", "| section-out-of-range | 1 |"]);
});
