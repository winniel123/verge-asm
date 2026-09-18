import test from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";
import { ROWS } from "./citations/rows.mjs";
import { trackedExtensions } from "./citations/classify.mjs";
import {
  BLOCK,
  NO_NUMBER,
  UNPAIRED,
  citedLinesFor,
  judgeHistory,
  countByFamily,
  reportHistory,
  revisionsFromLog,
  witnessFor,
} from "./sweep/history.mjs";
import { historyRecord } from "./sweep-line-anchors.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");
const SWEEP = join(SCRIPT_DIR, "sweep-line-anchors.mjs");

// The fixture target never lands on disk: the arm reads it out of a faked `git show`.
const FIXTURE = `.history-fixture-${process.pid}`;
const TARGET = `${FIXTURE}/go/target.go`;
const DOC = "docs/spec/fixture.md";

const AT_FIXTURE = ROWS.map((row) => ({
  ...row,
  matches: (p) => p.startsWith(`${FIXTURE}/`) && row.matches(p.slice(FIXTURE.length + 1)),
}));

// classify resolves against the tracked tree, and a fixture is untracked by design.
function envFor(paths) {
  const files = new Set(paths);
  const dirs = new Set();
  for (const f of files) {
    const parts = f.split("/");
    for (let i = 1; i < parts.length; i++) dirs.add(parts.slice(0, i).join("/"));
  }
  return {
    repoRoot: REPO_ROOT,
    tracked: { files, dirs },
    extensions: trackedExtensions({ files }),
    roots: new Set([...files].map((f) => f.split("/")[0])),
    exempt: () => null,
  };
}

const ENV = envFor([TARGET, DOC]);

const THEN = [
  "package fixture",
  "",
  "func Alpha() int {",
  "\treturn 1",
  "}",
  "",
  "func Gamma() int {",
  "\treturn 3",
  "}",
  "",
  "var Zeta, Eta = 1, 2",
  "",
].join("\n");

const MOVED = [
  "package fixture",
  "",
  "func Gamma() int {",
  "\treturn 3",
  "}",
  "",
  "func Alpha() int {",
  "\treturn 1",
  "}",
  "",
  "var Zeta, Eta = 1, 2",
  "",
].join("\n");

function log(revisions) {
  return revisions
    .map(({ commit, subject, added }) => [`${BLOCK}${commit} ${subject}`, ...added.map((a) => `+${a}`)].join("\n"))
    .join("\n");
}

function citation(anchor, overrides = {}) {
  return {
    file: DOC,
    line: 10,
    family: "docs/spec",
    path: TARGET,
    value: TARGET,
    anchor,
    form: "region",
    ...overrides,
  };
}

function judge(citations, revisions, { show = () => THEN } = {}) {
  const root = mkdtempSync(join(tmpdir(), "history-test-"));
  try {
    return judgeHistory(REPO_ROOT, ENV, citations, {
      rows: AT_FIXTURE,
      readLog: () => log(revisions),
      show,
      root,
    });
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

const WROTE_ALPHA = { commit: "a".repeat(40), subject: "docs: cite Alpha", added: [`\`${TARGET}:4\` holds it`] };

test("revisionsFromLog reads one block per revision and keeps only the added lines", () => {
  const text = [
    `${BLOCK}1111111111111111111111111111111111111111 second subject`,
    "diff --git a/x b/x",
    "--- a/x",
    "+++ b/x",
    "@@ -5,1 +5,1 @@",
    "-gone",
    "+stayed",
    `${BLOCK}2222222222222222222222222222222222222222 first subject`,
    "+born",
  ].join("\n");
  assert.deepEqual(revisionsFromLog(text), [
    { commit: "1".repeat(40), subject: "second subject", added: ["stayed"] },
    { commit: "2".repeat(40), subject: "first subject", added: ["born"] },
  ]);
});

test("revisionsFromLog reads a subject holding a space, and an empty one", () => {
  const [first, second] = revisionsFromLog(`${BLOCK}abc a b c\n${BLOCK}def`);
  assert.equal(first.subject, "a b c");
  assert.equal(second.subject, "");
});

test("citedLinesFor takes the line numbers of one path and leaves the others", () => {
  const added = [`\`${TARGET}:4\` and \`${FIXTURE}/go/other.go:9\` and \`${TARGET}:8\``];
  const env = envFor([TARGET, `${FIXTURE}/go/other.go`, DOC]);
  assert.deepEqual(
    citedLinesFor(env, DOC, added, TARGET).map((c) => c.fromLine),
    [4, 8],
  );
});

test("citedLinesFor ignores a token that spells no line", () => {
  assert.deepEqual(citedLinesFor(ENV, DOC, [`\`${TARGET}#Alpha\``], TARGET), []);
});

test("witnessFor takes the newest revision that still spells a number", () => {
  const revisions = [
    { commit: "c".repeat(40), subject: "the sweep", added: [`\`${TARGET}#Alpha\``] },
    { commit: "b".repeat(40), subject: "a correction", added: [`\`${TARGET}:8\``] },
    WROTE_ALPHA,
  ];
  const witness = witnessFor(ENV, DOC, revisions, TARGET);
  assert.equal(witness.commit, "b".repeat(40));
  assert.deepEqual(
    witness.cited.map((c) => c.fromLine),
    [8],
  );
});

test("SPEC §7 shape 6 — an unrelated edit to the citing line hides a drift before it", () => {
  const wrote = { commit: "a".repeat(40), subject: "docs: cite the fan-out", added: [`\`${TARGET}:4\` holds it`] };
  const touched = {
    commit: "d".repeat(40),
    subject: "docs: convert a second token on the line",
    added: [`\`${TARGET}:4\` holds it, and \`${TARGET}#Gamma\``],
  };
  const trees = new Map([
    [wrote.commit, THEN],
    [touched.commit, MOVED],
  ]);
  const show = (_repoRoot, commit) => trees.get(commit);

  const [atWrite] = judge([citation("Gamma")], [wrote], { show });
  assert.equal(atWrite.witness.commit, wrote.commit);
  assert.equal(atWrite.verdict, "drifted");
  assert.equal(atWrite.then, "Alpha");

  const [afterEdit] = judge([citation("Gamma")], [touched, wrote], { show });
  assert.equal(afterEdit.witness.commit, touched.commit);
  assert.equal(afterEdit.verdict, "consistent");
  assert.equal(afterEdit.then, "Gamma");
});

test("an anchor the cited line did not sit in is a drift candidate", () => {
  const [judged] = judge([citation("Gamma")], [WROTE_ALPHA]);
  assert.equal(judged.verdict, "drifted");
  assert.equal(judged.then, "Alpha");
  assert.equal(judged.declaredThen, true);
  assert.equal(judged.fromLine, 4);
  assert.match(judged.detail, /sat in `Alpha`/);
});

test("an anchor the cited line did sit in is consistent", () => {
  const [judged] = judge([citation("Alpha")], [WROTE_ALPHA]);
  assert.equal(judged.verdict, "consistent");
  assert.equal(judged.then, "Alpha");
});

test("a name the target did not declare then is recorded beside the verdict", () => {
  const [judged] = judge([citation("Epsilon")], [WROTE_ALPHA]);
  assert.equal(judged.verdict, "drifted");
  assert.equal(judged.declaredThen, false);
});

test("a citing line no revision ever wrote with a number is unwitnessed", () => {
  const revisions = [{ commit: "c".repeat(40), subject: "the sweep", added: [`\`${TARGET}#Alpha\``] }];
  const [judged] = judge([citation("Alpha")], revisions);
  assert.equal(judged.verdict, "unwitnessed");
  assert.equal(judged.cause, NO_NUMBER);
});

test("two citations of one path and one witnessed number pair by no guess", () => {
  const judged = judge([citation("Alpha"), citation("Gamma")], [WROTE_ALPHA]);
  assert.deepEqual(
    judged.map((c) => c.verdict),
    ["unwitnessed", "unwitnessed"],
  );
  assert.match(judged[0].detail, /2 citation\(s\).*1 line number\(s\)/);
  assert.equal(judged[0].cause, UNPAIRED);
});

test("a cited line two declarations share is unjudged, never a verdict", () => {
  const revision = { commit: "9".repeat(40), subject: "one line, two names", added: [`\`${TARGET}:11\``] };
  const [judged] = judge([citation("Alpha")], [revision]);
  assert.equal(judged.verdict, "unreadable");
  assert.match(judged.detail, /two declarations shared/);
});

test("a reversed range is unjudged, because it encloses neither cited line", () => {
  const revision = { commit: "8".repeat(40), subject: "backwards", added: [`\`${TARGET}:8-4\``] };
  const [judged] = judge([citation("Alpha")], [revision]);
  assert.equal(judged.verdict, "unreadable");
  assert.match(judged.detail, /runs backwards/);
});

test("two citations of one path and two witnessed numbers pair in order", () => {
  const revision = { commit: "d".repeat(40), subject: "both", added: [`\`${TARGET}:4\` then \`${TARGET}:8\``] };
  const judged = judge([citation("Alpha"), citation("Gamma")], [revision]);
  assert.deepEqual(
    judged.map((c) => [c.anchor, c.fromLine, c.verdict]),
    [
      ["Alpha", 4, "consistent"],
      ["Gamma", 8, "consistent"],
    ],
  );
});

test("a target absent from the witness commit is unjudged, never a verdict", () => {
  const [judged] = judge([citation("Alpha")], [WROTE_ALPHA], {
    show: () => {
      throw new Error("fatal: path does not exist");
    },
  });
  assert.equal(judged.verdict, "unreadable");
  assert.match(judged.detail, /was not in the tree/);
});

test("a number past the end of the witnessed target is unjudged", () => {
  const revision = { commit: "e".repeat(40), subject: "far", added: [`\`${TARGET}:400\``] };
  const [judged] = judge([citation("Alpha")], [revision]);
  assert.equal(judged.verdict, "unreadable");
  assert.match(judged.detail, /past the end/);
});

test("a cited line inside no declaration is a drift candidate the report names", () => {
  const revision = { commit: "f".repeat(40), subject: "import block", added: [`\`${TARGET}:1\``] };
  const [judged] = judge([citation("Alpha")], [revision]);
  assert.equal(judged.verdict, "drifted");
  assert.equal(judged.then, null);
  assert.match(judged.detail, /no declaration enclosed line 1/);
});

test("countByFamily counts a judged verdict and leaves the unjudged out of the total", () => {
  const judged = [
    { family: "docs/adr", verdict: "drifted" },
    { family: "docs/adr", verdict: "consistent" },
    { family: "docs/adr", verdict: "unwitnessed" },
    { family: "docs/spec", verdict: "unreadable" },
  ];
  assert.deepEqual(countByFamily(judged), [
    { family: "docs/adr", judged: 2, drifted: 1, consistent: 1, unwitnessed: 1, unreadable: 0 },
    { family: "docs/spec", judged: 0, drifted: 0, consistent: 0, unwitnessed: 0, unreadable: 1 },
  ]);
});

test("the report names the witness commit and the cited line, so a human checks one row", () => {
  const [judged] = judge([citation("Gamma")], [WROTE_ALPHA]);
  const lines = [];
  reportHistory([judged], (l) => lines.push(l));
  const text = lines.join("\n");
  assert.match(text, /aaaaaaaa wrote `.*:4`/);
  assert.match(text, /docs: cite Alpha/);
  assert.match(text, /1 drift candidate\(s\)/);
});

test("historyRecord carries the evidence, and never a repointed target", () => {
  const [judged] = judge([citation("Gamma")], [WROTE_ALPHA]);
  const [record] = historyRecord([judged]);
  assert.deepEqual(record, {
    file: DOC,
    line: 10,
    family: "docs/spec",
    form: "region",
    target: `${TARGET}#Gamma`,
    verdict: "drifted",
    witness: "a".repeat(40),
    cited: `${TARGET}:4`,
    enclosedThen: "Alpha",
    declaredThen: true,
    detail: record.detail,
  });
});

test("the target keeps the spelling the document carries, and the witness keeps its own", () => {
  const relative = `../../${TARGET}`;
  const [judged] = judge([citation("Gamma", { value: relative })], [WROTE_ALPHA]);
  const [record] = historyRecord([judged]);
  assert.equal(record.target, `${relative}#Gamma`);
  assert.equal(record.cited, `${TARGET}:4`);
});

function refuses(args) {
  try {
    execFileSync(process.execPath, [SWEEP, ...args], { cwd: REPO_ROOT, encoding: "utf8", stdio: "pipe" });
  } catch (err) {
    return { status: err.status, stderr: err.stderr };
  }
  return { status: 0, stderr: "" };
}

test("one run reports one instrument", () => {
  const { status, stderr } = refuses(["--audit", "--history"]);
  assert.equal(status, 2);
  assert.match(stderr, /two instruments/);
});

test("the history arm writes nothing, so it refuses --write", () => {
  const { status, stderr } = refuses(["--history", "--write", "docs/spec"]);
  assert.equal(status, 2);
  assert.match(stderr, /decides nothing/);
});
