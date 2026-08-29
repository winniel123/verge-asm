/*
 * doclint tests — the fixture-corpus acceptance gate (SPEC §6) plus a few engine
 * checks that prove prose extraction hides every SPEC §3 non-prose region.
 *
 * Run with:  node --test scripts/doclint.test.mjs   (wired as `npm run test:doclint`).
 */
import { test } from "node:test";
import assert from "node:assert/strict";
import { RULES } from "./doclint/rules/index.mjs";
import { checkRuleCorpus } from "./doclint/fixtures.mjs";
import { parse, extractProse, lintMarkdown } from "./doclint/engine.mjs";

// The acceptance gate: every rule flags its must-flag set and no must-not-flag set.
for (const rule of RULES) {
  test(`fixture corpus: ${rule.name}`, () => {
    const failures = checkRuleCorpus(rule);
    const detail = failures
      .map((f) => `  [${f.kind}] got ${f.got} for ${JSON.stringify(f.snippet)}`)
      .join("\n");
    assert.equal(failures.length, 0, `\n${detail}`);
  });
}

/** The prose text of a snippet, joined — what the rules see. */
function proseText(markdown) {
  return extractProse(parse(markdown))
    .map((n) => n.value)
    .join("|");
}

test("prose extraction keeps ordinary paragraph text", () => {
  assert.equal(proseText("A plain sentence."), "A plain sentence.");
});

test("prose extraction drops a fenced code block", () => {
  assert.equal(proseText("Before.\n\n```sh\nrm -rf /; echo done\n```\n\nAfter."), "Before.|After.");
});

test("prose extraction drops an inline code span", () => {
  assert.equal(proseText("Run `a; b` now."), "Run | now.");
});

test("prose extraction drops table cells", () => {
  assert.equal(proseText("| A | B |\n| --- | --- |\n| `x; y` | z |"), "");
});

test("prose extraction drops a blockquote subtree", () => {
  assert.equal(proseText("Lead in.\n\n> Quoted; frozen source.\n\nTail."), "Lead in.|Tail.");
});

test("prose extraction drops a YAML front-matter block", () => {
  assert.equal(proseText("---\ntitle: One; two\n---\n\nBody prose."), "Body prose.");
});

test("no-semicolons reports the exact line of the semicolon", () => {
  const violations = lintMarkdown("Line one.\n\nLine three; here.", RULES);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].line, 3);
  assert.equal(violations[0].severity, "error");
  assert.equal(violations[0].rule, "no-semicolons");
});

test("no-semicolons reports a line with two semicolons only once", () => {
  const violations = lintMarkdown("Three states: ready; running; done.", RULES);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].line, 1);
});

/** The sentence-length-cap violations of a snippet (drops the no-semicolons noise). */
function lengthViolations(markdown) {
  return lintMarkdown(markdown, RULES).filter((v) => v.rule === "sentence-length-cap");
}

test("sentence-length-cap flags a sentence over 25 words as an error", () => {
  const sentence =
    "This single sentence keeps going and going and it adds one clause and then one more clause until it clearly runs over the universal cap value here.";
  const violations = lengthViolations(sentence);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].severity, "error");
  assert.equal(violations[0].rule, "sentence-length-cap");
});

test("sentence-length-cap leaves a 25-word sentence alone (inclusive cap)", () => {
  const sentence =
    "This sentence holds exactly twenty five plain words in total and it sits right at the cap value so the rule must leave it alone.";
  assert.equal(lengthViolations(sentence).length, 0);
});

test("sentence-length-cap counts a sentence, not a paragraph", () => {
  // Two 15-word sentences: the paragraph is 30 words, but neither sentence is over the cap.
  const paragraph =
    "This first sentence carries fifteen words and it stays well under the universal cap value here. This second sentence also carries a comfortable fifteen words under the cap.";
  assert.equal(lengthViolations(paragraph).length, 0);
});

test("sentence-length-cap reports the source line of the long sentence", () => {
  const long =
    "This single sentence keeps going and going and it adds one clause and then one more clause until it clearly runs over the universal cap value here.";
  const doc = `# Title\n\nA short intro line.\n\n${long}\n`;
  const violations = lengthViolations(doc);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].line, 5);
});

test("sentence-length-cap counts one sentence across an inline code span", () => {
  // The prose on both sides of the span joins into one sentence, so the words still total
  // over the cap even though the span drops out.
  const sentence =
    "You should run the `doclint` command against every single documentation file in the repository before you open a pull request or ask anyone to review the change.";
  assert.equal(lengthViolations(sentence).length, 1);
});

test("sentence-length-cap: documented boundary residue — an abbreviation merges two sentences", () => {
  // SPEC §2.2 worst case: "an abbreviation or a decimal number can mis-split a sentence".
  // parse-english reads "cap." as an abbreviation, so it does not break the sentence there.
  // Two sentences, each under the cap on its own (10 words then 16 words), merge into one
  // 26-word sentence and the rule over-flags the pair. This is a known false positive, not
  // a defect. The deferred inline disable directive (SPEC §6, #821) is the escape hatch.
  // This test pins the residue. A parser upgrade that splits on "cap." will change it.
  const merged =
    "The tool measures each sentence against the universal word cap. A second sentence adds more clauses and words and clauses until the pair clearly runs over.";
  const violations = lengthViolations(merged);
  assert.equal(violations.length, 1);
  assert.match(violations[0].message, /runs 26 words/);
});
