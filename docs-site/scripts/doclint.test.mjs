import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { writeFileSync, rmSync, mkdirSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { RULES } from "./doclint/rules/index.mjs";
import { checkRuleCorpus } from "./doclint/fixtures.mjs";
import { oneInstruction } from "./doclint/candidates/one-instruction.mjs";
import { noEllipsis } from "./doclint/candidates/no-ellipsis.mjs";
import { parse, extractProse, lintMarkdown } from "./doclint/engine.mjs";
import { tagSentence } from "./doclint/rules/simple-tenses.mjs";
import { isInScope } from "./doclint/scope.mjs";
import { annotationLine, summaryMarkdown } from "./doclint/github.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = join(SCRIPT_DIR, "..", "..");

for (const rule of RULES) {
  test(`fixture corpus: ${rule.name}`, () => {
    const failures = checkRuleCorpus(rule);
    const detail = failures
      .map((f) => `  [${f.kind}] got ${f.got} for ${JSON.stringify(f.snippet)}`)
      .join("\n");
    assert.equal(failures.length, 0, `\n${detail}`);
  });
}

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

function lengthViolations(markdown) {
  return lintMarkdown(markdown, RULES).filter((v) => v.rule === "sentence-length-cap");
}

// the pair mis-splits at "cap." in parse-english, so the cap over-flags (doc-lint-tool.md §2.2)
const MERGED_OVER_CAP =
  "The tool measures each sentence against the universal word cap. A second sentence adds more clauses and words and clauses until the pair clearly runs over.";

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
  const sentence =
    "You should run the `doclint` command against every single documentation file in the repository before you open a pull request or ask anyone to review the change.";
  assert.equal(lengthViolations(sentence).length, 1);
});

test("sentence-length-cap: documented boundary residue — an abbreviation merges two sentences", () => {
  const violations = lengthViolations(MERGED_OVER_CAP);
  assert.equal(violations.length, 1);
  assert.match(violations[0].message, /runs 26 words/);
});

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
  assert.equal(phrasalViolations("The scheduler kicked off three runs.").length, 1);
});

test("no-phrasal-verbs catches an irregular past form", () => {
  assert.equal(phrasalViolations("The operator spun up the cluster.").length, 1);
});

test("no-phrasal-verbs does not match a particle inside the next word (word boundary)", () => {
  assert.equal(phrasalViolations("The rotor can spin upward of ten times.").length, 0);
});

test("no-phrasal-verbs reports the source line of the phrasal verb", () => {
  const doc = "# Title\n\nA short intro line.\n\nWe spin up a fresh worker here.\n";
  const violations = phrasalViolations(doc);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].line, 5);
});

test("no-phrasal-verbs does not read a blockquote (frozen source) or a code span", () => {
  assert.equal(phrasalViolations("> The vendor said to spin up a worker.").length, 0);
  assert.equal(phrasalViolations("Run `spin up` in the shell.").length, 0);
});

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
  assert.equal(tenseViolations("The tokens have changed since the release.").length, 1);
  assert.equal(tenseViolations("The worker has completed the job.").length, 1);
  assert.equal(tenseViolations("The values had changed before the reset.").length, 1);
});

test("simple-tenses leaves an anchor before a non-participle alone", () => {
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
  assert.equal(tenseViolations("> The vendor has completed the work.").length, 0);
  assert.equal(tenseViolations("Run `has completed` in the shell.").length, 0);
});

test("the pos tokenizer does not duplicate a word (SPEC §4.3 mitigation)", () => {
  // the retext stack duplicates words at these package versions (doc-lint-tool.md §4.3)
  const words = tagSentence("alpha beta gamma delta epsilon").map(([w]) => w);
  assert.deepEqual(words, ["alpha", "beta", "gamma", "delta", "epsilon"]);
});

function passiveViolations(markdown) {
  return lintMarkdown(markdown, RULES).filter((v) => v.rule === "passive-voice");
}

test("passive-voice flags a passive form as a warning, not an error", () => {
  const violations = passiveViolations("The job was completed by the worker.");
  assert.equal(violations.length, 1);
  assert.equal(violations[0].severity, "warning");
  assert.equal(violations[0].rule, "passive-voice");
  assert.match(violations[0].message, /"was completed"/);
});

test("passive-voice flags several be forms and the have-been chain", () => {
  assert.equal(passiveViolations("The tokens are generated at build time.").length, 1);
  assert.equal(passiveViolations("Each rule is measured against the corpus.").length, 1);
  assert.equal(passiveViolations("The docs have been rewritten this week.").length, 1);
});

test("passive-voice skips an adverb or a negation between the be form and the participle", () => {
  const adverb = passiveViolations("The report was recently generated by the job.");
  assert.equal(adverb.length, 1);
  assert.match(adverb[0].message, /"was recently generated"/);
  assert.equal(passiveViolations("The stale file is not removed until the next run.").length, 1);
});

test("passive-voice leaves an active clause and a plain adjective alone", () => {
  assert.equal(passiveViolations("The worker completed the job on time.").length, 0);
  assert.equal(passiveViolations("The build is ready for a review.").length, 0);
});

test("passive-voice does not flag a be form and an article before a participle", () => {
  assert.equal(passiveViolations("The report is a completed draft.").length, 0);
});

test("passive-voice reports the source line of the passive form", () => {
  const doc = "# Title\n\nA short intro line.\n\nThe job was completed by the worker.\n";
  const violations = passiveViolations(doc);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].line, 5);
});

test("passive-voice does not read a blockquote (frozen source) or a code span", () => {
  assert.equal(passiveViolations("> The vendor said the work was completed.").length, 0);
  assert.equal(passiveViolations("Run `is completed` in the shell.").length, 0);
});

const DISABLE = "<!-- doclint-disable-line -->";

test("doclint-disable-line silences an error violation on the next line", () => {
  const flagged = lintMarkdown("This bad line; has a semicolon.", RULES);
  assert.equal(flagged.length, 1);
  const silenced = lintMarkdown(`${DISABLE}\nThis bad line; has a semicolon.`, RULES);
  assert.equal(silenced.length, 0);
});

test("doclint-disable-line silences a warning as well as an error (both severities)", () => {
  const both = "The worker has completed the job; then it stopped.";
  assert.equal(lintMarkdown(both, RULES).length, 2);
  assert.equal(lintMarkdown(`${DISABLE}\n${both}`, RULES).length, 0);
});

test("doclint-disable-line silences only the next line, not the lines below it", () => {
  const doc = `${DISABLE}\nFirst bad line; here.\n\nSecond bad line; here.`;
  const violations = lintMarkdown(doc, RULES);
  assert.equal(violations.length, 1);
  assert.equal(violations[0].line, 4);
});

test("doclint-disable-line silences a real documented false positive (SPEC §6)", () => {
  assert.equal(lengthViolations(MERGED_OVER_CAP).length, 1);
  assert.equal(lengthViolations(`${DISABLE}\n${MERGED_OVER_CAP}`).length, 0);
});

test("doclint-disable-line silences the immediate next line only, not across a blank line", () => {
  // the next source line is silenced literally, so a blank line absorbs it (doc-lint-tool.md §6)
  const doc = `${DISABLE}\n\nThis bad line; still flags.`;
  assert.equal(lintMarkdown(doc, RULES).length, 1);
});

test("doclint-disable-line shown inside a code fence does not silence anything", () => {
  const doc = "```\n<!-- doclint-disable-line -->\n```\n\nThis bad line; still flags.";
  const violations = lintMarkdown(doc, RULES).filter((v) => v.rule === "no-semicolons");
  assert.equal(violations.length, 1);
});

test("doclint-disable-line: a silenced error does not change the exit code (SPEC §6)", () => {
  const cli = join(SCRIPT_DIR, "doclint.mjs");
  const file = join(tmpdir(), `doclint-disable-${process.pid}.md`);
  writeFileSync(file, `A clean intro line.\n\n${DISABLE}\nThis bad line; has a semicolon.\n`);
  try {
    const out = execFileSync("node", [cli, file], { encoding: "utf8" });
    assert.match(out, /check:doclint OK/);
  } finally {
    rmSync(file, { force: true });
  }
});

test("a warning-only run does not change the exit code (SPEC §5.1)", () => {
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

test("isInScope accepts a file in each of the five doc families (SPEC §1.3)", () => {
  for (const rel of [
    "docs/adr/0001-example.md",
    "docs/spec/doc-lint-tool.md",
    "docs/agents/domain.md",
    "docs/guides/using.md",
    "docs/research/r1.md",
  ]) {
    assert.equal(isInScope(REPO_ROOT, join(REPO_ROOT, rel)), true, rel);
  }
});

test("isInScope accepts each of the four in-scope root files (SPEC §1.3)", () => {
  for (const rel of ["CONTEXT.md", "CLAUDE.md", "README.md", "SECURITY.md"]) {
    assert.equal(isInScope(REPO_ROOT, join(REPO_ROOT, rel)), true, rel);
  }
});

test("isInScope accepts a doc nested in a family subdirectory", () => {
  assert.equal(isInScope(REPO_ROOT, join(REPO_ROOT, "docs/research/2026/report.md")), true);
});

test("isInScope rejects the SPEC §1.3 out-of-scope paths", () => {
  for (const rel of [
    "docs/correspondence/note.md",
    "docs/wayfinder/map.md",
    "docs/guides/embed.go",
    "docs/adr/diagram.png",
    "docs/CONTEXT.md",
    "package.json",
    "CHANGELOG.md",
  ]) {
    assert.equal(isInScope(REPO_ROOT, join(REPO_ROOT, rel)), false, rel);
  }
});

test("isInScope rejects a path outside the repo root", () => {
  assert.equal(isInScope(REPO_ROOT, join(REPO_ROOT, "..", "elsewhere", "README.md")), false);
});

// deferral is on real-doc precision, not on a broken detector (doc-lint-tool.md §7.1, #824)
test("the #824 candidates are not enabled (SPEC §7.1 deferral)", () => {
  const names = new Set(RULES.map((r) => r.name));
  assert.equal(names.has("one-instruction-per-sentence"), false);
  assert.equal(names.has("no-ellipsis"), false);
});

test("the one-instruction candidate is sound on its own fixture corpus", () => {
  assert.deepEqual(checkRuleCorpus(oneInstruction), []);
});

test("the no-ellipsis candidate is sound on its own fixture corpus", () => {
  assert.deepEqual(checkRuleCorpus(noEllipsis), []);
});

test("annotationLine emits an ::error command for an error violation", () => {
  const line = annotationLine({
    file: "docs/adr/0001-x.md",
    line: 12,
    rule: "no-semicolons",
    severity: "error",
    message: "a semicolon in prose is not allowed — write two sentences",
  });
  assert.equal(
    line,
    "::error file=docs/adr/0001-x.md,line=12,title=doclint (no-semicolons)::a semicolon in prose is not allowed — write two sentences",
  );
});

test("annotationLine emits a ::warning command for a warning violation", () => {
  const line = annotationLine({
    file: "README.md",
    line: 3,
    rule: "simple-tenses",
    severity: "warning",
    message: "a compound verb form",
  });
  assert.match(line, /^::warning file=README\.md,line=3,title=doclint \(simple-tenses\)::/);
});

test("annotationLine escapes the workflow-command reserved characters", () => {
  const line = annotationLine({
    file: "docs/spec/a,b:c.md",
    line: 1,
    rule: "no-semicolons",
    severity: "error",
    message: "100% done\nnext",
  });
  assert.match(line, /file=docs\/spec\/a%2Cb%3Ac\.md/);
  assert.match(line, /::100%25 done%0Anext$/);
});

test("summaryMarkdown lists counts by rule and by severity", () => {
  const violations = [
    { file: "a.md", line: 1, rule: "no-semicolons", severity: "error", message: "m" },
    { file: "a.md", line: 2, rule: "no-semicolons", severity: "error", message: "m" },
    { file: "b.md", line: 5, rule: "simple-tenses", severity: "warning", message: "m" },
  ];
  const md = summaryMarkdown(4, violations);
  assert.match(md, /4 file\(s\) linted/);
  assert.match(md, /2 error\(s\)/);
  assert.match(md, /1 warning\(s\)/);
  assert.match(md, /no-semicolons/);
  assert.match(md, /simple-tenses/);
  assert.match(md, /no-semicolons \| error \| 2/);
});

test("summaryMarkdown reports a clean run with no violations", () => {
  const md = summaryMarkdown(3, []);
  assert.match(md, /no violations/i);
  assert.match(md, /3 file\(s\) linted/);
});

test("--github prints an annotation per violation and a summary", () => {
  const cli = join(SCRIPT_DIR, "doclint.mjs");
  const file = join(tmpdir(), `doclint-gh-${process.pid}.md`);
  writeFileSync(file, "This bad line; has a semicolon.\n");
  try {
    const out = execFileSync("node", [cli, "--github", file], { encoding: "utf8" });
    assert.match(out, /^::error file=/m);
    assert.match(out, /no-semicolons/);
    assert.match(out, /1 error\(s\)/);
  } finally {
    rmSync(file, { force: true });
  }
});

test("--github does not exit non-zero on an error (advisory, SPEC §5.2)", () => {
  // a clean execFileSync return is the exit-code assertion, because it throws on a non-zero exit
  const cli = join(SCRIPT_DIR, "doclint.mjs");
  const file = join(tmpdir(), `doclint-gh-exit-${process.pid}.md`);
  writeFileSync(file, "This bad line; has a semicolon.\n");
  try {
    const out = execFileSync("node", [cli, "--github", file], { encoding: "utf8" });
    assert.match(out, /::error file=/);
  } finally {
    rmSync(file, { force: true });
  }
});

test("--github writes the summary to GITHUB_STEP_SUMMARY when the env var is set", () => {
  const cli = join(SCRIPT_DIR, "doclint.mjs");
  const file = join(tmpdir(), `doclint-gh-sum-${process.pid}.md`);
  const summaryFile = join(tmpdir(), `doclint-gh-step-${process.pid}.md`);
  writeFileSync(file, "This bad line; has a semicolon.\n");
  writeFileSync(summaryFile, "");
  try {
    execFileSync("node", [cli, "--github", file], {
      encoding: "utf8",
      env: { ...process.env, GITHUB_STEP_SUMMARY: summaryFile },
    });
    const summary = readFileSync(summaryFile, "utf8");
    assert.match(summary, /no-semicolons/);
    assert.match(summary, /1 error\(s\)/);
  } finally {
    rmSync(file, { force: true });
    rmSync(summaryFile, { force: true });
  }
});

test("--in-scope-only drops an out-of-scope file and lints only the in-scope one", () => {
  const cli = join(SCRIPT_DIR, "doclint.mjs");
  const inScope = join(REPO_ROOT, "docs", "adr", `doclint-scope-${process.pid}.md`);
  const outOfScope = join(REPO_ROOT, "docs", "correspondence", `doclint-scope-${process.pid}.md`);
  const bad = "This bad line; has a semicolon.\n";
  writeFileSync(inScope, bad);
  mkdirSync(join(REPO_ROOT, "docs", "correspondence"), { recursive: true });
  writeFileSync(outOfScope, bad);
  try {
    const out = execFileSync(
      "node",
      [cli, "--github", "--in-scope-only", inScope, outOfScope],
      { encoding: "utf8" },
    );
    assert.match(out, /1 error\(s\)/);
    assert.doesNotMatch(out, /correspondence/);
  } finally {
    rmSync(inScope, { force: true });
    rmSync(outOfScope, { force: true });
  }
});

test("--in-scope-only with no in-scope file lints nothing and stays advisory", () => {
  const cli = join(SCRIPT_DIR, "doclint.mjs");
  mkdirSync(join(REPO_ROOT, "docs", "correspondence"), { recursive: true });
  const outOfScope = join(REPO_ROOT, "docs", "correspondence", `doclint-empty-${process.pid}.md`);
  writeFileSync(outOfScope, "This bad line; has a semicolon.\n");
  try {
    const out = execFileSync("node", [cli, "--github", "--in-scope-only", outOfScope], {
      encoding: "utf8",
    });
    assert.doesNotMatch(out, /::error/);
    assert.match(out, /no in-scope/i);
  } finally {
    rmSync(outOfScope, { force: true });
  }
});
