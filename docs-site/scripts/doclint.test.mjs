/*
 * doclint tests — the fixture-corpus acceptance gate (SPEC §6) plus a few engine
 * checks that prove prose extraction hides every SPEC §3 non-prose region.
 *
 * Run with:  node --test scripts/doclint.test.mjs   (wired as `npm run test:doclint`).
 */
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { RULES } from "./doclint/rules/index.mjs";
import { checkRuleCorpus } from "./doclint/fixtures.mjs";
import { parse, extractProse, lintMarkdown } from "./doclint/engine.mjs";
import { tagSentence } from "./doclint/rules/simple-tenses.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));

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

/** The no-phrasal-verbs violations of a snippet (drops the other rules' noise). */
function phrasalViolations(markdown) {
  return lintMarkdown(markdown, RULES).filter((v) => v.rule === "no-phrasal-verbs");
}

test("no-phrasal-verbs flags a seeded phrasal verb as an error", () => {
  const violations = phrasalViolations("We spin up a worker for each job.");
  assert.equal(violations.length, 1);
  assert.equal(violations[0].severity, "error");
  assert.equal(violations[0].rule, "no-phrasal-verbs");
  assert.match(violations[0].message, /"spin up"/);
});

test("no-phrasal-verbs catches an inflected form the wordlist does not list literally", () => {
  // "kicked off" is the -ed form of the "kick off" entry. The rule must catch the
  // inflection, not only the base lemma (acceptance: flags every seeded phrasal verb).
  assert.equal(phrasalViolations("The scheduler kicked off three runs.").length, 1);
});

test("no-phrasal-verbs catches an irregular past form", () => {
  // "spun up" is the irregular past of "spin up", listed in the wordlist's irregular set.
  assert.equal(phrasalViolations("The operator spun up the cluster.").length, 1);
});

test("no-phrasal-verbs does not match a particle inside the next word (word boundary)", () => {
  // "spin upward" must not read as "spin up": "upward" has no word boundary after "up".
  assert.equal(phrasalViolations("The rotor can spin upward of ten times.").length, 0);
});

test("no-phrasal-verbs reports the source line of the phrasal verb", () => {
  const doc = "# Title\n\nA short intro line.\n\nWe spin up a fresh worker here.\n";
  const violations = phrasalViolations(doc);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].line, 5);
});

test("no-phrasal-verbs does not read a blockquote (frozen source) or a code span", () => {
  // Both regions are non-prose (SPEC §3), so a phrasal verb inside them never flags
  // (acceptance: no code or blockquote text is matched).
  assert.equal(phrasalViolations("> The vendor said to spin up a worker.").length, 0);
  assert.equal(phrasalViolations("Run `spin up` in the shell.").length, 0);
});

/** The simple-tenses violations of a snippet (drops the other rules' noise). */
function tenseViolations(markdown) {
  return lintMarkdown(markdown, RULES).filter((v) => v.rule === "simple-tenses");
}

test("simple-tenses flags a compound form as a warning, not an error", () => {
  const violations = tenseViolations("The worker has completed the job.");
  assert.equal(violations.length, 1);
  assert.equal(violations[0].severity, "warning");
  assert.equal(violations[0].rule, "simple-tenses");
  assert.match(violations[0].message, /"has completed"/);
});

test("simple-tenses flags each of the have, has, and had anchors", () => {
  // The anchor set is have/has/had. Each one before a past participle flags (SPEC §2.4).
  assert.equal(tenseViolations("The tokens have changed since the release.").length, 1);
  assert.equal(tenseViolations("The worker has completed the job.").length, 1);
  assert.equal(tenseViolations("The values had changed before the reset.").length, 1);
});

test("simple-tenses leaves an anchor before a non-participle alone", () => {
  // "has" before a number or a determiner is possession, not a compound verb form. It must
  // not flag, or the rule would fire on ordinary "has" sentences (SPEC §2.4 precision).
  assert.equal(tenseViolations("The tool has three states.").length, 0);
  assert.equal(tenseViolations("The report has a clear structure.").length, 0);
});

test("simple-tenses reports the source line of the compound form", () => {
  const doc = "# Title\n\nA short intro line.\n\nThe worker has completed the job.\n";
  const violations = tenseViolations(doc);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].line, 5);
});

test("simple-tenses does not read a blockquote (frozen source) or a code span", () => {
  // Both regions are non-prose (SPEC §3), so a compound form inside them never flags.
  assert.equal(tenseViolations("> The vendor has completed the work.").length, 0);
  assert.equal(tenseViolations("Run `has completed` in the shell.").length, 0);
});

test("the pos tokenizer does not duplicate a word (SPEC §4.3 mitigation)", () => {
  // The chosen mitigation is the `pos` lexer with a direct tag pass (SPEC §4.3 option 1).
  // The retext stack this rule avoids turned "a b c" into "a b b c c". The `pos` lexer
  // returns one token per word, so the token stream holds no duplicate (acceptance #3).
  const words = tagSentence("alpha beta gamma delta epsilon").map(([w]) => w);
  assert.deepEqual(words, ["alpha", "beta", "gamma", "delta", "epsilon"]);
});

test("a warning-only run does not change the exit code (SPEC §5.1)", () => {
  // The CLI exits non-zero only on an error-level violation. A file that holds a single
  // simple-tenses warning must exit zero. execFileSync throws on a non-zero exit, so a clean
  // return is the proof (acceptance #2). The output still prints the warning line.
  const cli = join(SCRIPT_DIR, "doclint.mjs");
  const file = join(tmpdir(), `doclint-warn-${process.pid}.md`);
  writeFileSync(file, "The worker has completed the job.\n");
  try {
    const out = execFileSync("node", [cli, file], { encoding: "utf8" });
    assert.match(out, /simple-tenses/);
    assert.match(out, /0 error\(s\), 1 warning\(s\)/);
  } finally {
    rmSync(file, { force: true });
  }
});
