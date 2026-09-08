import { test } from "node:test";
import assert from "node:assert/strict";
import { addedAdrFiles, bodyDecision, evaluate, gather, parseMarker } from "./check-adr-review.mjs";

const HEAD = "0123456789abcdef0123456789abcdef01234567";
const OLD = "fedcba9876543210fedcba9876543210fedcba98";

const SLUG = "a-proposer-answers-a-search-with-at-most-one-organisation";
const FILE_1700 = `docs/adr/1700-${SLUG}.md`;
const FILE_1701 = "docs/adr/1701-a-second-rule.md";

const DECISION = [
  "A proposer answers a search with at most one organisation. The join keeps its second leg.",
  "",
  "Rejected: a list. A list needs a ranking rule nobody wrote.",
];

const ADR_1700 = [
  "---",
  "number: 1700",
  'title: "A proposer answers a search with at most one organisation"',
  `slug: ${SLUG}`,
  "date: 2026-09-10",
  "status: accepted",
  "source: grilling",
  "ticket: 1700",
  'proof: {test: "internal/exposure/caida_test.go::TestSearchReturnsOneOrg"}',
  "---",
  "",
  "# ADR-1700: A proposer answers a search with at most one organisation",
  "",
  "## Decision",
  "",
  ...DECISION,
  "",
  "## 1. Rationale",
  "",
  "Text.",
  "",
].join("\n");

const BODY = ["Closes #1700", "", "## Decision", "", ...DECISION, "", "## Test plan", "", "- ran it"].join(
  "\n",
);

const marker = (sha, verdict) => `<!-- adr-review sha=${sha} verdict=${verdict} -->`;

function comment(sha, verdict, extra = "") {
  return {
    body: [
      marker(sha, verdict),
      "",
      "| Row | Verdict | Evidence |",
      "| --- | --- | --- |",
      `| 1. Three-part test | ${verdict} | evidence |`,
      "",
      "Fresh-context subagent. Inputs: PR diff, `index.json`, the proof test.",
      extra,
    ].join("\n"),
  };
}

const added = (filename) => ({ filename, status: "added" });
const modified = (filename) => ({ filename, status: "modified" });

function run(overrides = {}) {
  const contents = { [FILE_1700]: ADR_1700, [FILE_1701]: ADR_1700.replace("1700", "1701") };
  return evaluate({
    headSha: HEAD,
    files: [added(FILE_1700), modified("docs/adr/index.json")],
    comments: [comment(HEAD, "pass")],
    prBody: BODY,
    readAdr: (path) => contents[path] ?? null,
    ...overrides,
  });
}

test("a PR that adds no ADR file passes", () => {
  const r = run({
    files: [modified("docs/adr/0001-stack-and-runtime.md"), added("docs/spec/x.md"), added("docs/adr/index.json")],
    comments: [],
    prBody: "",
  });
  assert.equal(r.code, 0);
  assert.deepEqual(r.added, []);
  assert.deepEqual(r.problems, []);
});

test("one added ADR, one pass marker on the head SHA, and a matching body pass", () => {
  const r = run();
  assert.deepEqual(r.problems, []);
  assert.equal(r.code, 0);
  assert.deepEqual(r.added, [FILE_1700]);
});

test("two markers for one SHA fail, whatever their verdicts", () => {
  const r = run({ comments: [comment(HEAD, "pass"), comment(HEAD, "pass")] });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /2 adr-review markers for head 0123456/);
});

test("two added ADR files fail", () => {
  const r = run({ files: [added(FILE_1700), added(FILE_1701)] });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /adds 2 ADR files/);
});

test("a PR body whose ## Decision differs from the file's Decision block fails", () => {
  const r = run({ prBody: BODY.replace("at most one organisation", "at most two organisations") });
  assert.equal(r.code, 1);
  const text = r.problems.join("\n");
  assert.match(text, /## Decision differs from docs\/adr\/1700-/);
  assert.match(text, /file: A proposer answers a search with at most one organisation/);
  assert.match(text, /body: A proposer answers a search with at most two organisations/);
});

test("a PR body with no ## Decision section fails", () => {
  const r = run({ prBody: "Closes #1700\n\nSome prose." });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /no ## Decision section/);
});

test("a marker for an older SHA does not count for the head", () => {
  const r = run({ comments: [comment(OLD, "pass")] });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /no adr-review marker for head 0123456/);
});

test("a fail verdict on the head SHA fails", () => {
  const r = run({ comments: [comment(OLD, "fail"), comment(HEAD, "fail")] });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /verdict for head 0123456.* is fail/);
});

test("the marker must open the comment", () => {
  assert.deepEqual(parseMarker(`\n  ${marker(HEAD, "pass")}\nprose`), { sha: HEAD, verdict: "pass" });
  assert.equal(parseMarker(`prose\n${marker(HEAD, "pass")}`), null);
  assert.equal(parseMarker(`<!-- adr-review sha=abc verdict=pass -->`), null);
  assert.equal(parseMarker(`${marker(HEAD, "pass")}junk`), null);
  assert.deepEqual(parseMarker(`${marker(HEAD, "fail")} \r\nprose`), { sha: HEAD, verdict: "fail" });
  assert.equal(parseMarker(null), null);
  const r = run({ comments: [{ body: `See below.\n${marker(HEAD, "pass")}` }] });
  assert.match(r.problems.join("\n"), /no adr-review marker for head/);
});

test("an unreadable added ADR file fails", () => {
  const r = run({ readAdr: () => null });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /cannot read docs\/adr\/1700-/);
});

test("addedAdrFiles keeps only added ADR files under docs/adr/", () => {
  const files = [
    added(FILE_1700),
    modified(FILE_1701),
    { filename: "docs/adr/0227-x.md", status: "removed" },
    { filename: "docs/adr/0226-y.md", status: "renamed" },
    added("docs/adr/index.json"),
    added("docs/spec/1702-not-an-adr.md"),
    added("docs/adr/README.md"),
  ];
  assert.deepEqual(addedAdrFiles(files), [FILE_1700]);
});

test("bodyDecision runs from the first ## Decision to the next heading of any level", () => {
  assert.equal(bodyDecision(BODY), DECISION.join("\n"));
  assert.equal(bodyDecision("## Decision\r\n\r\nRule.  \r\n\r\n### Sub\r\nmore"), "Rule.");
  assert.equal(bodyDecision("## Decisions\n\ntext"), null);
  assert.equal(bodyDecision(null), null);
});

test("gather pages through the PR files and comments", async () => {
  const page = (n, prefix) => Array.from({ length: n }, (_, i) => ({ filename: `${prefix}${i}`, body: `${prefix}${i}` }));
  const pages = {
    "/repos/o/r/pulls/7/files?per_page=100&page=1": page(100, "f"),
    "/repos/o/r/pulls/7/files?per_page=100&page=2": page(3, "g"),
    "/repos/o/r/issues/7/comments?per_page=100&page=1": page(2, "c"),
  };
  const seen = [];
  const fetchImpl = async (url, init) => {
    seen.push([url, init.headers.Authorization]);
    const path = url.replace("https://api.github.com", "");
    const body = pages[path];
    return { ok: body !== undefined, status: body ? 200 : 404, json: async () => body ?? {} };
  };
  const r = await gather({ repo: "o/r", number: 7, token: "tok", fetchImpl });
  assert.equal(r.files.length, 103);
  assert.equal(r.comments.length, 2);
  assert.equal(seen.length, 3);
  assert.ok(seen.every(([, auth]) => auth === "Bearer tok"));
});

test("gather throws on a non-2xx response", async () => {
  const fetchImpl = async () => ({ ok: false, status: 403, json: async () => ({ message: "rate limited" }) });
  await assert.rejects(gather({ repo: "o/r", number: 7, token: "", fetchImpl }), /403/);
});
