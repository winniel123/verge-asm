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
