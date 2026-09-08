import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  ADR_FILE,
  FENCE,
  annotationLine,
  buildAdrIndex,
  checkFile,
  findCitations,
  headings,
  isTextFile,
  numberedHeadings,
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

test("a section named by word is a violation, since only a number resolves", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const found = checkFile("docs/spec/x.md", "ADR-0129 §Context says so.", index);
    assert.equal(found.length, 1);
    assert.equal(found[0].rule, "section-by-name");
    assert.equal(found[0].text, "ADR-0129 §Context");
    assert.match(found[0].message, /names a heading by word/);
  });
});

test("a linked citation names a section by word too", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const md = "see [ADR-0129](./0129-a.md) §Rationale for why";
    const found = checkFile("docs/adr/0194-x.md", md, index);
    assert.equal(found.length, 1);
    assert.equal(found[0].rule, "section-by-name");
  });
});

test("a section by word inside a code span is a specimen, not a citation", () => {
  assert.deepEqual(findCitations("the 52 `ADR-0129 §n` pointers", { markdown: true }), []);
});

test("a lone § after an ADR is not a citation", () => {
  assert.deepEqual(findCitations("ADR-0129 § below", { markdown: false }), []);
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

test("a numbered heading reports its number, line, and level", () => {
  assert.deepEqual(numberedHeadings(ADR_WITH_SECTIONS), [
    { number: "1", line: 7, level: 3 },
    { number: "2", line: 9, level: 3 },
    { number: "2.1", line: 11, level: 4 },
  ]);
});

test("a numbered heading inside a fence has no line", () => {
  const md = ["## Decision", "", "```md", "### 1. Not a heading here", "```", ""].join("\n");
  assert.deepEqual(numberedHeadings(md), []);
});

test("the Set wrapper keeps the numbers of the heading list", () => {
  const md = "## 3 Rationale\n### 3.1) sub\n";
  assert.deepEqual(numberedHeadings(md).map((h) => h.number), ["3", "3.1"]);
  assert.deepEqual([...numberedSections(md)], ["3", "3.1"]);
});

test("the ADR-file pattern and the fence rule are the exported ones", () => {
  assert.ok(ADR_FILE.test("0129-a-title.md"));
  assert.ok(ADR_FILE.test("12345-a-title.md"));
  assert.equal(ADR_FILE.test("129-a-title.md"), false);
  assert.equal(ADR_FILE.test("0129-a-title.txt"), false);
  assert.ok(FENCE.test("```go"));
  assert.ok(FENCE.test("   ~~~"));
  assert.equal(FENCE.test("    ```"), false);
});

test("the index carries each heading's line and level beside the section set", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const entry = buildAdrIndex(root).get("0129");
    assert.equal(entry.file, "docs/adr/0129-a-title.md");
    assert.deepEqual([...entry.sections], ["1", "2", "2.1"]);
    assert.deepEqual(entry.headings[2], { number: "2.1", line: 11, level: 4 });
  });
});

test("a five-digit ADR is indexed under its filename prefix", () => {
  withRepo({ "docs/adr/12345-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    assert.ok(index.has("12345"));
    assert.deepEqual(checkFile("internal/x/y.go", "// r (ADR-12345 §2.1)", index), []);
  });
});

test("a five-digit citation to a missing file is unresolvable, not silent", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const found = checkFile("internal/x/y.go", "// r (ADR-12345 §1)", index);
    assert.equal(found.length, 1);
    assert.equal(found[0].rule, "unresolvable-adr");
    assert.equal(found[0].adr, "12345");
  });
});

test("a legacy amendment citation is found with its issue and no section", () => {
  const found = findCitations("// a veto (ADR-0129 #944).", { markdown: false });
  assert.deepEqual(found, [{ line: 1, adr: "0129", issue: "944", text: "ADR-0129 #944" }]);
});

test("a legacy amendment citation wraps like any other", () => {
  const found = findCitations("(ADR-0129\n#944)", { markdown: false });
  assert.deepEqual(found, [{ line: 1, adr: "0129", issue: "944", text: "ADR-0129 #944" }]);
});

test("an issue beside the ADR, after a comma, is not an amendment citation", () => {
  assert.deepEqual(findCitations("(ADR-0227, #1634)", { markdown: false }), []);
  assert.deepEqual(findCitations("(ADR-0227 §2, #1634)", { markdown: false }), [
    { line: 1, adr: "0227", section: "2", text: "ADR-0227 §2" },
  ]);
});

test("an amendment citation followed by a section is the issue-section form alone", () => {
  const found = findCitations("(ADR-0126 #1321 §3)", { markdown: false });
  assert.equal(found.length, 1);
  assert.equal(found[0].section, "3");
});

test("a legacy amendment citation is accepted at 227 and below", () => {
  withRepo(
    {
      "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS,
      "docs/adr/0227-a-title.md": ADR_WITHOUT_SECTIONS,
    },
    (root) => {
      const index = buildAdrIndex(root);
      assert.deepEqual(checkFile("cmd/web/x.go", "// r (ADR-0129 #944)", index), []);
      assert.deepEqual(checkFile("cmd/web/x.go", "// r (ADR-0227 #1634)", index), []);
    },
  );
});

test("a legacy amendment citation fails above 227", () => {
  withRepo(
    {
      "docs/adr/0228-a-title.md": ADR_WITH_SECTIONS,
      "docs/adr/1646-a-title.md": ADR_WITH_SECTIONS,
    },
    (root) => {
      const index = buildAdrIndex(root);
      for (const text of ["// r (ADR-0228 #1700)", "// r (ADR-1646 #1646)"]) {
        const found = checkFile("cmd/web/x.go", text, index);
        assert.equal(found.length, 1, text);
        assert.equal(found[0].rule, "amendment-above-legacy");
        assert.match(found[0].message, /in-file amendment/);
      }
    },
  );
});

test("a legacy amendment citation to a missing ADR is unresolvable first", () => {
  withRepo({ "docs/adr/0129-a-title.md": ADR_WITH_SECTIONS }, (root) => {
    const index = buildAdrIndex(root);
    const found = checkFile("cmd/web/x.go", "// r (ADR-0999 #12)", index);
    assert.equal(found.length, 1);
    assert.equal(found[0].rule, "unresolvable-adr");
  });
});

// The pre-repair text of the four sites #1735 repaired, so a pass proves the check catches the form
const REPAIRED_BY_1735 = [
  ["docs/adr/0194-x.md", "ADR-0011 §Rationale says why the two values must stay", "0011"],
  ["docs/adr/0196-x.md", "ADR-0148 §Context states this for `http-exchange`", "0148"],
  ["docs/adr/0209-x.md", "ADR-0143 §Consequences found the same", "0143"],
  ["docs/guides/reports.md", "map of your attack surface (ADR-0039 §Context), and", "0039"],
];

for (const [file, text, adr] of REPAIRED_BY_1735) {
  test(`#1735: ${file} carried a §Name citation of ADR-${adr}`, () => {
    withRepo({ [`docs/adr/${adr}-a-title.md`]: ADR_WITHOUT_SECTIONS }, (root) => {
      const found = checkFile(file, text, buildAdrIndex(root));
      assert.equal(found.length, 1);
      assert.equal(found[0].rule, "section-by-name");
    });
  });
}

test("every ATX heading is listed with its level, title, and line", () => {
  assert.deepEqual(headings(ADR_WITH_SECTIONS).slice(0, 3), [
    { level: 1, number: null, title: "ADR-0129: A title", line: 1 },
    { level: 2, number: null, title: "Context", line: 3 },
    { level: 2, number: null, title: "Decision", line: 5 },
  ]);
  assert.deepEqual(headings(ADR_WITH_SECTIONS)[5], { level: 4, number: "2.1", title: "A subsection", line: 11 });
});

test("a heading inside a fence is not listed, and an H1 never carries a number", () => {
  const md = ["# 3 things", "", "```", "## 1. fenced", "```", "## 4.", ""].join("\n");
  assert.deepEqual(headings(md), [
    { level: 1, number: null, title: "3 things", line: 1 },
    { level: 2, number: null, title: "4.", line: 6 },
  ]);
});

test("the numbered list is the heading list filtered to numbered ones", () => {
  assert.deepEqual(
    numberedHeadings(ADR_WITH_SECTIONS),
    headings(ADR_WITH_SECTIONS)
      .filter((h) => h.number !== null)
      .map(({ number, line, level }) => ({ number, line, level })),
  );
});
