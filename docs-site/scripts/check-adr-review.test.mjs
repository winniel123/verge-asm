import { test } from "node:test";
import assert from "node:assert/strict";
import {
  addedAdrFiles,
  bodyDecision,
  evaluate,
  gather,
  headContents,
  legacyQuoteRuns,
  modifiedAdrFiles,
  parseMarker,
  patchHunks,
} from "./check-adr-review.mjs";

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

const added = (filename, patch) => ({ filename, status: "added", patch });
const modified = (filename, patch) => ({ filename, status: "modified", patch });

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
  assert.deepEqual(modifiedAdrFiles(files), [FILE_1701]);
});

test("the added and modified sets stay distinct", () => {
  const files = [added(FILE_1700), modified(FILE_1701)];
  assert.deepEqual(addedAdrFiles(files), [FILE_1700]);
  assert.deepEqual(modifiedAdrFiles(files), [FILE_1701]);
  assert.deepEqual(addedAdrFiles([modified(FILE_1700)]), []);
  assert.deepEqual(modifiedAdrFiles([added(FILE_1700)]), []);
});

test("bodyDecision runs from the first ## Decision to the next heading of any level", () => {
  assert.equal(bodyDecision(BODY), DECISION.join("\n"));
  assert.equal(bodyDecision("## Decision\r\n\r\nRule.  \r\n\r\n### Sub\r\nmore"), "Rule.");
  assert.equal(bodyDecision("## Decisions\n\ntext"), null);
  assert.equal(bodyDecision(null), null);
});

const TRAILER = "\u{1F916} Generated with [Claude Code](https://claude.com/claude-code)";
const SESSION = "https://claude.ai/code/session_01UMirUT7YQoTqdEhW1Rtm1N";

const LAST_SECTION_BODY = ["Closes #1700", "", "## Decision", "", ...DECISION, "", TRAILER].join("\n");

const LAST_SECTION_BODY_WITH_SESSION = [
  "Closes #1700",
  "",
  "## Decision",
  "",
  ...DECISION,
  "",
  TRAILER,
  "",
  SESSION,
  "",
].join("\n");

test("a last-position ## Decision does not absorb the attribution trailer", () => {
  assert.equal(bodyDecision(LAST_SECTION_BODY), DECISION.join("\n"));
  assert.equal(bodyDecision(`## Decision\n\nRule.\n\n\n${TRAILER}\n`), "Rule.");
  assert.equal(bodyDecision(`## Decision\r\n\r\nRule.\r\n\r\n${TRAILER}\r\n`), "Rule.");
});

test("the trailer is a block: the session link below the attribution line goes with it", () => {
  assert.equal(bodyDecision(LAST_SECTION_BODY_WITH_SESSION), DECISION.join("\n"));
  assert.equal(bodyDecision(`## Decision\r\n\r\nRule.\r\n\r\n${TRAILER}\r\n\r\n${SESSION}\r\n`), "Rule.");
});

test("a PR body whose ## Decision is the last section passes", () => {
  assert.equal(run({ prBody: LAST_SECTION_BODY }).code, 0);
  const r = run({ prBody: LAST_SECTION_BODY_WITH_SESSION });
  assert.equal(r.code, 0);
  assert.deepEqual(r.problems, []);
});

test("a trailing line that is neither the attribution nor its session link stays", () => {
  assert.equal(bodyDecision(`## Decision\n\n${TRAILER}\n\nRule.`), `${TRAILER}\n\nRule.`);
  assert.equal(bodyDecision("## Decision\n\nRule.\n\nSee the log.\n"), "Rule.\n\nSee the log.");
  assert.equal(bodyDecision(`## Decision\n\nRule.\n\n${SESSION}\n`), `Rule.\n\n${SESSION}`);
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

const FILE_2300 = "docs/adr/2300-a-rule.md";
const FILE_0110 = "docs/adr/0110-a-legacy-rule.md";
const MARKER_2400 =
  "> **Amended** by [ADR-2400: A later rule](./2400-a-later-rule.md), 2026-09-18. <!-- adr-marker amends 2400 -->";

const adr2300 = (lines) =>
  [
    "---",
    "number: 2300",
    'title: "A rule"',
    "slug: a-rule",
    "date: 2026-09-17",
    "status: accepted",
    "source: fix",
    "ticket: 2300",
    'proof: {none: "a rule with no test"}',
    "---",
    "",
    "# ADR-2300: A rule",
    "",
    ...lines,
  ].join("\n");

const BODY_2300 = [
  "## Decision",
  "",
  "A rule.",
  "",
  "## 1. Context",
  "",
  "> **Withdrawn** 2026-09-01: the older rule is gone.",
  "> The rest of this Decision stands.",
  "",
  "Prose line.",
  "",
];

const ADR_2300 = adr2300(BODY_2300);
const ADR_2300_CORRECTED = adr2300(BODY_2300.filter((l) => l !== "> The rest of this Decision stands."));
const ADR_2300_PROSE = adr2300(BODY_2300.map((l) => (l === "Prose line." ? "Prose line, corrected." : l)));
const ADR_2300_MARKED = adr2300(["", MARKER_2400, "", ...BODY_2300.slice(1)]);

const PATCH_CORRECTION = [
  "@@ -18,6 +18,5 @@",
  " ## 1. Context",
  " ",
  " > **Withdrawn** 2026-09-01: the older rule is gone.",
  "-> The rest of this Decision stands.",
  " ",
  " Prose line.",
].join("\n");

const PATCH_PROSE = [
  "@@ -20,4 +20,4 @@",
  " > **Withdrawn** 2026-09-01: the older rule is gone.",
  " > The rest of this Decision stands.",
  " ",
  "-Prose line.",
  "+Prose line, corrected.",
].join("\n");

const PATCH_MARKER = [
  "@@ -12,3 +12,5 @@",
  " # ADR-2300: A rule",
  " ",
  `+${MARKER_2400}`,
  "+",
  " ## Decision",
].join("\n");

function runModified(file, patch, contents, overrides = {}) {
  return evaluate({
    headSha: HEAD,
    files: [modified(file, patch), modified("docs/adr/index.json")],
    comments: [comment(HEAD, "pass")],
    prBody: "Closes #2300",
    readAdr: (path) => contents[path] ?? null,
    ...overrides,
  });
}

test("legacyQuoteRuns keeps a sentinel-free blockquote run and drops one that carries a sentinel", () => {
  const text = ["a", "> one", "> two", "", "> **Amended** <!-- adr-marker amends 9 -->", "> tail"].join("\n");
  assert.deepEqual(legacyQuoteRuns(text), [[2, 3]]);
  assert.deepEqual(legacyQuoteRuns(["```", "> fenced", "```"].join("\n")), []);
});

test("patchHunks numbers a changed line by its position in the after file", () => {
  const [h] = patchHunks(PATCH_CORRECTION);
  assert.deepEqual(h.changes, [{ kind: "-", text: "> The rest of this Decision stands.", at: 21 }]);
  assert.deepEqual(patchHunks(PATCH_PROSE)[0].changes, [
    { kind: "-", text: "Prose line.", at: 23 },
    { kind: "+", text: "Prose line, corrected.", at: 23 },
  ]);
});

test("a modified ADR above 227 reaches the review rather than being filtered out", () => {
  const r = runModified(FILE_2300, PATCH_CORRECTION, { [FILE_2300]: ADR_2300_CORRECTED });
  assert.deepEqual(r.problems, []);
  assert.equal(r.code, 0);
  assert.deepEqual(r.added, []);
  assert.deepEqual(r.modified, [FILE_2300]);
  assert.deepEqual(r.reviewed, [FILE_2300]);
});

test("an ADR-2159-shaped marker correction above 227 needs a pass verdict on the head SHA", () => {
  const contents = { [FILE_2300]: ADR_2300_CORRECTED };
  const none = runModified(FILE_2300, PATCH_CORRECTION, contents, { comments: [] });
  assert.equal(none.code, 1);
  assert.match(none.problems.join("\n"), /no adr-review marker for head 0123456/);

  const fail = runModified(FILE_2300, PATCH_CORRECTION, contents, { comments: [comment(HEAD, "fail")] });
  assert.equal(fail.code, 1);
  assert.match(fail.problems.join("\n"), /verdict for head 0123456.* is fail/);
});

test("a prose edit outside a legacy marker blockquote above 227 fails and names the amends route", () => {
  const r = runModified(FILE_2300, PATCH_PROSE, { [FILE_2300]: ADR_2300_PROSE });
  assert.equal(r.code, 1);
  const text = r.problems.join("\n");
  assert.match(text, /docs\/adr\/2300-a-rule\.md:23 deletes a line outside a standing legacy marker blockquote/);
  assert.match(text, /row M1 refuses above 227/);
  assert.match(text, /\{kind: amends, adr: 2300\}/);
  assert.deepEqual(r.reviewed, []);
});

test("a tool-written marker hunk above 227 is check-adr-markers' business, not a modification to review", () => {
  const r = runModified(FILE_2300, PATCH_MARKER, { [FILE_2300]: ADR_2300_MARKED }, { comments: [] });
  assert.deepEqual(r.problems, []);
  assert.equal(r.code, 0);
  assert.deepEqual(r.reviewed, []);
});

test("at or below 227 a modified ADR keeps the legacy route and passes without the M rows", () => {
  const r = runModified(FILE_0110, PATCH_PROSE, {}, { comments: [] });
  assert.deepEqual(r.problems, []);
  assert.equal(r.code, 0);
  assert.deepEqual(r.modified, [FILE_0110]);
  assert.deepEqual(r.reviewed, []);
});

test("a modified ADR above 227 with no diff fails closed", () => {
  const r = runModified(FILE_2300, undefined, { [FILE_2300]: ADR_2300 });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /carries no diff, so row M1 cannot be judged/);
});

test("a modified ADR above 227 the checkout cannot read fails", () => {
  const r = runModified(FILE_2300, PATCH_PROSE, {});
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /cannot read docs\/adr\/2300-a-rule\.md at the head SHA/);
});

test("an added ADR file gets the four rows and not the M rows", () => {
  const patch = ["@@ -0,0 +1,3 @@", "+# ADR-1700: A rule", "+", "+Prose, and no blockquote in sight."].join("\n");
  const r = run({ files: [added(FILE_1700, patch), modified("docs/adr/index.json")] });
  assert.deepEqual(r.problems, []);
  assert.equal(r.code, 0);
  assert.deepEqual(r.reviewed, [FILE_1700]);
});

test("one added ADR and one reviewable modification above 227 is two ADRs in one PR", () => {
  const contents = { [FILE_1700]: ADR_1700, [FILE_2300]: ADR_2300_CORRECTED };
  const r = run({
    files: [added(FILE_1700), modified(FILE_2300, PATCH_CORRECTION)],
    readAdr: (path) => contents[path] ?? null,
  });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /reviews 2 ADR files .*; one ADR per PR/);
});

const CONTENTS = "https://api.github.com/repos/o/r/contents";
const RAW_MEDIA = "application/vnd.github.raw";

const FRESH_LINE = "> Actually this rule is void and the opposite holds.";
const ADR_2300_FRESH = adr2300(["## Decision", "", "A rule.", "", FRESH_LINE, "", ...BODY_2300.slice(4)]);

const PATCH_FRESH = ["@@ -16,3 +16,5 @@", " A rule.", " ", `+${FRESH_LINE}`, "+", " ## 1. Context"].join("\n");

const ADR_2300_BLANK = adr2300(BODY_2300.filter((l) => l !== "> The rest of this Decision stands.").slice(0, -1));

test("a blockquote this patch wrote whole is not a standing legacy marker, so M1 refuses it", () => {
  const r = runModified(FILE_2300, PATCH_FRESH, { [FILE_2300]: ADR_2300_FRESH });
  assert.equal(r.code, 1);
  const text = r.problems.join("\n");
  assert.match(text, /docs\/adr\/2300-a-rule\.md:18 adds a line outside a standing legacy marker blockquote/);
  assert.deepEqual(r.reviewed, []);
});

test("a blank line beside a named correction is no offence of its own", () => {
  const patch = [
    "@@ -18,7 +18,5 @@",
    " ## 1. Context",
    " ",
    " > **Withdrawn** 2026-09-01: the older rule is gone.",
    "-> The rest of this Decision stands.",
    "-",
    " Prose line.",
  ].join("\n");
  const r = runModified(FILE_2300, patch, { [FILE_2300]: ADR_2300_BLANK });
  assert.deepEqual(r.problems, []);
  assert.equal(r.code, 0);
  assert.deepEqual(r.reviewed, [FILE_2300]);
});

test("a hunk that changes blank lines alone has no warrant and fails, naming the blank", () => {
  const patch = ["@@ -21,3 +21,2 @@", " > The rest of this Decision stands.", "-", " Prose line."].join("\n");
  const r = runModified(FILE_2300, patch, { [FILE_2300]: ADR_2300 });
  assert.equal(r.code, 1);
  assert.match(r.problems.join("\n"), /deletes a line outside a standing legacy marker blockquote/);
  assert.match(r.problems.join("\n"), /\n {2}-<blank>/);
});

test("legacyQuoteRuns groups a run and drops the one the sentinel sits in, wherever in it", () => {
  const text = ["a", "> one", "> two", "", "> three", "> **x** <!-- adr-marker amends 9 -->"].join("\n");
  assert.deepEqual(legacyQuoteRuns(text), [[2, 3]]);
  assert.deepEqual(legacyQuoteRuns(["> a", "", "> b", "> c"].join("\n")), [[1], [3, 4]]);
});

test("headContents reads each path at the head SHA and maps a 404 to null", async () => {
  const seen = [];
  const fetchImpl = async (url, init) => {
    seen.push([url, init.headers.Accept, init.headers.Authorization]);
    if (url.includes("gone")) return { ok: false, status: 404, text: async () => "" };
    return { ok: true, status: 200, text: async () => "body" };
  };
  const out = await headContents({
    repo: "o/r",
    sha: HEAD,
    paths: ["docs/adr/2300-a-rule.md", "docs/adr/gone.md"],
    token: "tok",
    fetchImpl,
  });
  assert.equal(out.get("docs/adr/2300-a-rule.md"), "body");
  assert.equal(out.get("docs/adr/gone.md"), null);
  assert.equal(seen[0][0], `${CONTENTS}/docs/adr/2300-a-rule.md?ref=${HEAD}`);
  assert.equal(seen[0][1], RAW_MEDIA);
  assert.equal(seen[0][2], "Bearer tok");
});

test("headContents throws on a non-2xx that is not a 404", async () => {
  const fetchImpl = async () => ({ ok: false, status: 500, text: async () => "" });
  await assert.rejects(headContents({ repo: "o/r", sha: HEAD, paths: ["docs/adr/x.md"], fetchImpl }), /500/);
});
