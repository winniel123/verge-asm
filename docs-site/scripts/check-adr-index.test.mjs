import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import {
  INDEX_FILE,
  buildIndex,
  decisionBlock,
  derivedStatus,
  loadAdrs,
  run,
  splitFrontMatter,
  validate,
} from "./check-adr-index.mjs";

const SLUG_223 = "a-bar-is-authored-in-the-release";
const SLUG_227 = "caida-publishes-an-org-name-search";
const SLUG_1700 = "a-proposer-answers-a-search-with-at-most-one-organisation";

const FRONT_223 = [
  "number: 223",
  'title: "a bar is authored in the release"',
  `slug: ${SLUG_223}`,
  "date: 2026-09-07",
  "status: accepted",
  "source: fix",
  "ticket: [1519, 1583, 1604]",
  'proof: {none: "predates the governance SPEC"}',
  "relations:",
  "  - {kind: rests-on, adr: 3}",
];

const FRONT_227 = [
  "number: 227",
  'title: "CAIDA publishes an org-name search"',
  `slug: ${SLUG_227}`,
  "date: 2026-09-07",
  "status: accepted",
  "source: fix",
  "ticket: [1616, 1519]",
  'proof: {none: "predates the governance SPEC"}',
  "relations:",
  '  - {kind: retires, adr: 223, clause: "3"}',
  "  - {kind: rests-on, adr: 3}",
];

const FRONT_1700 = [
  "number: 1700",
  'title: "A proposer answers a search with at most one organisation"',
  `slug: ${SLUG_1700}`,
  "date: 2026-09-10",
  "status: accepted",
  "source: grilling",
  "ticket: 1700",
  "map: 1638",
  'proof: {test: "internal/exposure/caida_test.go::TestSearchReturnsOneOrg"}',
  "relations:",
  '  - {kind: amends, adr: 227, clause: "2"}',
];

const NUMBERED_BODY = [
  "## Context",
  "",
  "Three tickets ask one question.",
  "",
  "## Decision",
  "",
  "A reachability finding is Declared. An install's last failed attempt is Operational.",
  "",
  "Rejected: a periodic probe.",
  "",
  "### 1. The first rule",
  "",
  "Text.",
  "",
  "### 2. The second rule",
  "",
  "Text.",
  "",
  "### 3. The third rule",
  "",
  "Text.",
  "",
  "### 4. The fourth rule",
  "",
  "Text.",
  "",
  "## Consequences",
  "",
  "Text.",
  "",
];

const UNNUMBERED_BODY = [
  "## Context",
  "",
  "Text.",
  "",
  "## Decision",
  "",
  "The rule.",
  "",
  "### An unnumbered rule",
  "",
  "Text.",
  "",
];

const NEW_BODY = [
  "## Decision",
  "",
  "A proposer answers a search with at most one organisation. Rejected: a list.",
  "",
  "## 1. Rationale",
  "",
  "Text.",
  "",
];

function adr({ front, h1, body }) {
  const head = front ? ["---", ...front, "---", ""] : [];
  return [...head, h1, "", ...body].join("\n");
}

const GO_TEST = "package exposure\n\nfunc TestSearchReturnsOneOrg(t *testing.T) {}\n";

function corpus(overrides = {}) {
  return {
    [`docs/adr/0003-third-party.md`]: adr({ h1: "# ADR-0003: Third party", body: UNNUMBERED_BODY }),
    [`docs/adr/0223-${SLUG_223}.md`]: adr({
      front: FRONT_223,
      h1: "# ADR-0223: a bar is authored in the release",
      body: NUMBERED_BODY,
    }),
    [`docs/adr/0227-${SLUG_227}.md`]: adr({
      front: FRONT_227,
      h1: "# ADR-0227: CAIDA publishes an org-name search",
      body: NUMBERED_BODY,
    }),
    [`docs/adr/1700-${SLUG_1700}.md`]: adr({
      front: FRONT_1700,
      h1: "# ADR-1700: A proposer answers a search with at most one organisation",
      body: NEW_BODY,
    }),
    "internal/exposure/caida_test.go": GO_TEST,
    ...overrides,
  };
}

function withRepo(files, fn) {
  const root = mkdtempSync(join(tmpdir(), "adr-index-"));
  try {
    for (const [rel, body] of Object.entries(files)) {
      mkdirSync(dirname(join(root, rel)), { recursive: true });
      writeFileSync(join(root, rel), body);
    }
    return fn(root);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

function problemsOf(files) {
  return withRepo(files, (root) => validate(loadAdrs(root), root).map((p) => p.message));
}

function assertProblem(files, pattern) {
  const problems = problemsOf(files);
  assert.ok(problems.some((m) => pattern.test(m)), `${pattern} not in:\n  ${problems.join("\n  ")}`);
}

function replacing(front, key, line) {
  const out = [];
  let dropping = false;
  for (const l of front) {
    if (l.startsWith(`${key}:`)) dropping = true;
    else if (dropping && /^\s/.test(l)) continue;
    else dropping = false;
    if (!dropping) out.push(l);
  }
  return line === undefined ? out : [...out, line];
}

function withFront(number, front, extra = {}) {
  const [slug, h1, body] = {
    223: [SLUG_223, "# ADR-0223: a bar is authored in the release", NUMBERED_BODY],
    227: [SLUG_227, "# ADR-0227: CAIDA publishes an org-name search", NUMBERED_BODY],
    1700: [SLUG_1700, "# ADR-1700: A proposer answers a search with at most one organisation", NEW_BODY],
  }[number];
  const name = `${String(number).padStart(4, "0")}-${slug}.md`;
  return corpus({ [`docs/adr/${name}`]: adr({ front, h1: extra.h1 ?? h1, body: extra.body ?? body }), ...extra.files });
}

test("the clean corpus validates with no problem", () => {
  assert.deepEqual(problemsOf(corpus()), []);
});

test("an unquoted date stays a string under the core schema", () => {
  const { front } = splitFrontMatter("---\ndate: 2026-09-07\n---\n\n# T\n");
  assert.equal(front.date, "2026-09-07");
});

test("a file with no front matter has no front and an unchanged body", () => {
  const { front, body, frontLines } = splitFrontMatter("# T\n\nText.\n");
  assert.equal(front, null);
  assert.equal(frontLines, 0);
  assert.equal(body, "# T\n\nText.\n");
});

test("the body starts after the closing fence, and the front matter's line count is known", () => {
  const { body, frontLines } = splitFrontMatter("---\na: 1\nb: 2\n---\n\n# T\n");
  assert.equal(body, "\n# T\n");
  assert.equal(frontLines, 4);
});

test("a legacy ADR without front matter is loaded but gets no row", () => {
  withRepo(corpus(), (root) => {
    const adrs = loadAdrs(root);
    assert.equal(adrs.get(3).front, null);
    assert.deepEqual(
      buildIndex(adrs).adrs.map((r) => r.number),
      [223, 227, 1700],
    );
  });
});

test("front matter must be followed by one blank line, then the H1", () => {
  const raw = ["---", ...FRONT_223, "---", "# ADR-0223: a bar is authored in the release", "", ...NUMBERED_BODY].join("\n");
  assertProblem(corpus({ [`docs/adr/0223-${SLUG_223}.md`]: raw }), /one blank line, then the H1/);
});

test("invalid YAML is a schema problem, not a crash", () => {
  const raw = ["---", "number: [", "---", "", "# ADR-0223: a bar is authored in the release", "", ...NUMBERED_BODY].join("\n");
  assertProblem(corpus({ [`docs/adr/0223-${SLUG_223}.md`]: raw }), /not valid YAML/);
});

for (const key of ["number", "title", "slug", "date", "status", "source", "proof"]) {
  test(`a missing \`${key}\` fails`, () => {
    assertProblem(withFront(223, replacing(FRONT_223, key)), new RegExp(`lacks \`${key}\``));
  });
}

test("number must equal the filename prefix", () => {
  assertProblem(withFront(223, replacing(FRONT_223, "number", "number: 224")), /number 224 != filename 223/);
});

test("slug must equal the filename stem", () => {
  assertProblem(withFront(223, replacing(FRONT_223, "slug", "slug: other")), /slug `other` != filename stem/);
});

test("title must equal the H1 less an optional ADR-NNNN: prefix", () => {
  assert.deepEqual(problemsOf(withFront(223, FRONT_223, { h1: "# a bar is authored in the release" })), []);
  assertProblem(withFront(223, replacing(FRONT_223, "title", 'title: "other"')), /title != H1/);
});

test("an H1 prefix that names another number fails", () => {
  assertProblem(withFront(223, FRONT_223, { h1: "# ADR-0224: a bar is authored in the release" }), /H1 prefix ADR-0224/);
});

test("status is accepted or withdrawn, and source is grilling, fix, or sweep", () => {
  assertProblem(withFront(223, replacing(FRONT_223, "status", "status: amended")), /status `amended`/);
  assertProblem(withFront(223, replacing(FRONT_223, "source", "source: audit")), /source `audit`/);
});

test("date must be YYYY-MM-DD", () => {
  assertProblem(withFront(223, replacing(FRONT_223, "date", "date: 7 Sep 2026")), /date `7 Sep 2026`/);
  assertProblem(withFront(223, replacing(FRONT_223, "date", "date: 2026-13-45")), /date `2026-13-45`/);
});

test("ticket is an integer or a list of integers, and map and pr are integers", () => {
  assertProblem(withFront(223, replacing(FRONT_223, "ticket", 'ticket: "#1519"')), /ticket must be an integer or a list/);
  assertProblem(withFront(223, [...FRONT_223, "map: none"]), /map must be an integer/);
  assertProblem(withFront(223, [...FRONT_223, 'pr: "1705"']), /pr must be an integer/);
});

test("above 227, ticket is required and number must equal it", () => {
  assertProblem(withFront(1700, replacing(FRONT_1700, "ticket")), /above 227, `ticket` is required/);
  assertProblem(withFront(1700, replacing(FRONT_1700, "ticket", "ticket: 1701")), /number 1700 must equal ticket 1701/);
  assertProblem(withFront(1700, replacing(FRONT_1700, "ticket", "ticket: [1700, 1638]")), /number 1700 must equal ticket/);
});

test("above 227, the H1 carries the prefix and the title is 16 words or fewer", () => {
  assertProblem(
    withFront(1700, FRONT_1700, { h1: "# A proposer answers a search with at most one organisation" }),
    /H1 must carry the ADR-1700: prefix/,
  );
  const long = "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen";
  assertProblem(
    withFront(1700, replacing(FRONT_1700, "title", `title: "${long}"`), { h1: `# ADR-1700: ${long}` }),
    /title is 17 words, cap is 16/,
  );
});

test("withdrawn needs all three withdrawal fields", () => {
  const front = replacing(FRONT_223, "status", "status: withdrawn");
  assertProblem(withFront(223, front), /withdrawn without withdrawal\.date/);
  assertProblem(withFront(223, [...front, "withdrawal: {date: 2026-09-08, reason: r}"]), /withdrawn without withdrawal\.moved-to/);
  assert.deepEqual(
    problemsOf(withFront(223, [...front, 'withdrawal: {date: 2026-09-08, reason: "false", moved-to: none}'])),
    [],
  );
});

test("moved-to is a path on disk, an ADR on disk, or none", () => {
  const front = replacing(FRONT_223, "status", "status: withdrawn");
  assertProblem(withFront(223, [...front, "withdrawal: {date: 2026-09-08, reason: r, moved-to: 999}"]), /moved-to ADR-0999 is not on disk/);
  assertProblem(withFront(223, [...front, "withdrawal: {date: 2026-09-08, reason: r, moved-to: docs/spec/x.md}"]), /moved-to docs\/spec\/x\.md does not exist/);
  assert.deepEqual(
    problemsOf(withFront(223, [...front, "withdrawal: {date: 2026-09-08, reason: r, moved-to: 3}"])),
    [],
  );
});

test("proof holds exactly one of test, ticket, none", () => {
  assertProblem(withFront(223, replacing(FRONT_223, "proof", "proof: {none: r, ticket: 1}")), /exactly one of test\|ticket\|none/);
  assertProblem(withFront(223, replacing(FRONT_223, "proof", "proof: {tests: r}")), /exactly one of test\|ticket\|none/);
});

test("a none proof needs a reason string", () => {
  assertProblem(withFront(223, replacing(FRONT_223, "proof", "proof: {none: null}")), /none proof needs a reason/);
  assertProblem(withFront(223, replacing(FRONT_223, "proof", 'proof: {none: ""}')), /none proof needs a reason/);
});

test("a ticket proof is an integer and is not checked", () => {
  assert.deepEqual(problemsOf(withFront(223, replacing(FRONT_223, "proof", "proof: {ticket: 99999}"))), []);
  assertProblem(withFront(223, replacing(FRONT_223, "proof", 'proof: {ticket: "#1"}')), /ticket proof must be an integer/);
});

test("a .go test proof needs the file and that Test function", () => {
  const proof = (v) => replacing(FRONT_1700, "proof", `proof: {test: "${v}"}`);
  assertProblem(withFront(1700, proof("internal/exposure/nope_test.go::TestX")), /proof test path internal\/exposure\/nope_test\.go does not exist/);
  assertProblem(withFront(1700, proof("internal/exposure/caida_test.go::TestOther")), /does not declare func TestOther/);
  assertProblem(withFront(1700, proof("internal/exposure/caida_test.go::SearchReturnsOneOrg")), /is not a Test function/);
  assertProblem(withFront(1700, proof("internal/exposure/caida_test.go")), /proof test must be <path>::<name>/);
});

test("a .mjs test proof needs a test() with that title", () => {
  const files = { "docs-site/scripts/x.test.mjs": 'test("a relation without a marker fails", () => {});\n' };
  const proof = (v) => replacing(FRONT_1700, "proof", `proof: {test: "${v}"}`);
  assert.deepEqual(
    problemsOf(withFront(1700, proof("docs-site/scripts/x.test.mjs::a relation without a marker fails"), { files })),
    [],
  );
  assertProblem(withFront(1700, proof("docs-site/scripts/x.test.mjs::another title"), { files }), /holds no test\("another title"\)/);
  assertProblem(withFront(1700, proof("docs-site/scripts/x.test.mjs::a relation without a marker")), /does not exist/);
});

test("a test proof in another language does not resolve", () => {
  const files = { "web/x.test.ts": 'test("t", () => {});\n' };
  assertProblem(withFront(1700, replacing(FRONT_1700, "proof", 'proof: {test: "web/x.test.ts::t"}'), { files }), /must be a \.go or \.mjs path/);
});

function withRelations(number, front, rows) {
  return withFront(number, [...replacing(front, "relations"), ...(rows.length ? ["relations:", ...rows] : [])]);
}

test("a relation kind is one of the six", () => {
  assertProblem(withRelations(227, FRONT_227, ["  - {kind: replaces, adr: 223}"]), /kind `replaces` is not one of the six/);
});

test("a relation may target only an ADR file on disk", () => {
  assertProblem(withRelations(227, FRONT_227, ["  - {kind: rests-on, adr: 999}"]), /targets ADR-0999, which is not on disk/);
  assertProblem(withRelations(227, FRONT_227, ['  - {kind: rests-on, adr: "docs/spec/x.md"}']), /adr must be an integer/);
  assertProblem(withRelations(227, FRONT_227, ["  - {kind: rests-on, adr: 227}"]), /targets itself/);
});

test("supersedes and sibling refuse a clause", () => {
  assertProblem(withRelations(227, FRONT_227, ['  - {kind: supersedes, adr: 223, clause: "3"}']), /supersedes forbids a clause/);
  assertProblem(withRelations(227, FRONT_227, ['  - {kind: sibling, adr: 223, clause: "3"}']), /sibling forbids a clause/);
});

test("amends and retires need a clause when the target numbers a heading", () => {
  assertProblem(withRelations(227, FRONT_227, ["  - {kind: retires, adr: 223}"]), /retires ADR-0223 needs a clause/);
  assertProblem(withRelations(227, FRONT_227, ["  - {kind: amends, adr: 223}"]), /amends ADR-0223 needs a clause/);
});

test("amends and retires on an unnumbered target act on the whole file", () => {
  assert.deepEqual(problemsOf(withRelations(227, FRONT_227, ["  - {kind: amends, adr: 3}"])), []);
  assertProblem(withRelations(227, FRONT_227, ['  - {kind: amends, adr: 3, clause: "1"}']), /ADR-0003 numbers no heading/);
});

test("a clause must resolve to a numbered heading of the target", () => {
  assertProblem(withRelations(227, FRONT_227, ['  - {kind: retires, adr: 223, clause: "9"}']), /target numbers 1, 2, 3, 4/);
  assertProblem(withRelations(227, FRONT_227, ['  - {kind: rests-on, adr: 223, clause: "9"}']), /target numbers 1, 2, 3, 4/);
  assertProblem(withRelations(227, FRONT_227, ["  - {kind: retires, adr: 223, clause: 3}"]), /clause 3 must be a string/);
  assertProblem(withRelations(227, FRONT_227, ['  - {kind: retires, adr: 223, clause: "§3"}']), /clause `§3` is not a dotted heading number/);
});

function bothSides() {
  const files = withRelations(223, FRONT_223, ["  - {kind: rests-on, adr: 227}"]);
  return {
    ...files,
    ...withRelations(227, FRONT_227, ['  - {kind: retires, adr: 223, clause: "3"}', "  - {kind: rests-on, adr: 223}"]),
    [`docs/adr/0223-${SLUG_223}.md`]: files[`docs/adr/0223-${SLUG_223}.md`],
  };
}

test("a relation written on both sides fails on both files", () => {
  const files = bothSides();
  const problems = problemsOf(files).filter((m) => /an edge lives once/.test(m));
  assert.equal(problems.length, 2);
  assert.ok(problems.some((m) => m.startsWith("docs/adr/0223-")));
  assert.ok(problems.some((m) => m.startsWith("docs/adr/0227-")));
});

test("a duplicate relation fails", () => {
  assertProblem(withRelations(227, FRONT_227, ["  - {kind: rests-on, adr: 3}", "  - {kind: rests-on, adr: 3}"]), /duplicate relation/);
});

test("relations must be a list of objects", () => {
  assertProblem(withFront(227, [...replacing(FRONT_227, "relations"), "relations: 223"]), /relations must be a list/);
});

function statusOf(files, number) {
  return withRepo(files, (root) => {
    const index = buildIndex(loadAdrs(root));
    return index.adrs.find((r) => r.number === number).status;
  });
}

test("retires and amends derive amended on their targets, and an actor with no incoming edge stays accepted", () => {
  assert.deepEqual(statusOf(corpus(), 223), { set: "accepted", derived: "amended" });
  assert.deepEqual(statusOf(corpus(), 227), { set: "accepted", derived: "amended" });
  assert.deepEqual(statusOf(corpus(), 1700), { set: "accepted", derived: "accepted" });
});

test("precedence is withdrawn, superseded, amended, accepted", () => {
  assert.equal(derivedStatus("accepted", [{ kind: "amends" }, { kind: "supersedes" }]), "superseded");
  assert.equal(derivedStatus("accepted", [{ kind: "retires" }]), "amended");
  assert.equal(derivedStatus("accepted", [{ kind: "rests-on" }, { kind: "bounds" }, { kind: "sibling" }]), "accepted");
  assert.equal(derivedStatus("withdrawn", [{ kind: "supersedes" }]), "withdrawn");
});

test("every edge counts for the life of the file", () => {
  const files = withRelations(1700, FRONT_1700, ["  - {kind: supersedes, adr: 227}"]);
  assert.equal(statusOf(files, 227).derived, "superseded");
  assert.equal(statusOf(files, 223).derived, "amended");
});

test("sibling shows on both sides of the index", () => {
  const files = withRelations(1700, FRONT_1700, ["  - {kind: sibling, adr: 227}"]);
  withRepo(files, (root) => {
    const rows = buildIndex(loadAdrs(root)).adrs;
    const row = (n) => rows.find((r) => r.number === n);
    assert.deepEqual(row(227).incoming.filter((e) => e.kind === "sibling"), [{ kind: "sibling", from: 1700, clause: null }]);
    assert.deepEqual(row(1700).incoming, [{ kind: "sibling", from: 227, clause: null }]);
    assert.deepEqual(row(1700).relations, [{ kind: "sibling", adr: 227 }]);
  });
});

test("the Decision block runs from the first ## Decision to the next heading of any level", () => {
  const d = decisionBlock(NUMBERED_BODY.join("\n"));
  assert.equal(d.text, "A reachability finding is Declared. An install's last failed attempt is Operational.\n\nRejected: a periodic probe.");
  assert.equal(d.words, 16);
});

test("a marker line under the Decision heading is not part of the block", () => {
  const body = ["## Decision", "", "> **Amended** by x. <!-- adr-marker amends 1700 -->", "", "The rule.", "", "### 1. A", ""].join("\n");
  assert.deepEqual(decisionBlock(body), { words: 2, text: "The rule." });
});

test("an ADR with no Decision heading quotes its first paragraph", () => {
  const body = ["# Subjects leave by measurement", "", "First line of the paragraph.", "Second line.", "", "Next paragraph.", ""].join("\n");
  assert.deepEqual(decisionBlock(body), { words: 7, text: "First line of the paragraph.\nSecond line." });
});

test("a legacy Decision block is quoted uncapped", () => {
  const words = Array.from({ length: 200 }, (_, i) => `w${i}`).join(" ");
  const body = ["## Context", "", "## Decision", "", words, "", "### 1. A", "", "### 3. C", ""];
  const files = withFront(223, FRONT_223, { body });
  assert.deepEqual(problemsOf(files), []);
  withRepo(files, (root) => {
    assert.equal(buildIndex(loadAdrs(root)).adrs.find((r) => r.number === 223).decision.words, 200);
  });
});

test("above 227 the Decision block is the first ## and holds 150 words or fewer", () => {
  const words = Array.from({ length: 151 }, (_, i) => `w${i}`).join(" ");
  assertProblem(
    withFront(1700, FRONT_1700, { body: ["## Decision", "", words, "", "## 1. A", ""] }),
    /Decision block is 151 words, cap is 150/,
  );
  assertProblem(
    withFront(1700, FRONT_1700, { body: ["## Context", "", "## Decision", "", "x", "", "## 1. A", ""] }),
    /first ## must be Decision/,
  );
  assertProblem(withFront(1700, FRONT_1700, { body: ["## 1. A", "", "x", ""] }), /first ## must be Decision/);
});

test("each row carries the #1645 fields, and a marker has no line", () => {
  withRepo(corpus(), (root) => {
    const index = buildIndex(loadAdrs(root));
    const row = index.adrs.find((r) => r.number === 223);
    assert.deepEqual(Object.keys(row), [
      "number",
      "file",
      "title",
      "slug",
      "date",
      "status",
      "source",
      "proof",
      "ticket",
      "map",
      "pr",
      "withdrawal",
      "relations",
      "incoming",
      "sections",
      "markers",
      "decision",
    ]);
    assert.equal(row.file, `docs/adr/0223-${SLUG_223}.md`);
    assert.equal(row.date, "2026-09-07");
    assert.deepEqual(row.ticket, [1519, 1583, 1604]);
    assert.equal(row.map, null);
    assert.deepEqual(row.sections, ["1", "2", "3", "4"]);
    assert.deepEqual(row.incoming, [{ kind: "retires", from: 227, clause: "3" }]);
    assert.deepEqual(row.markers, [{ kind: "retires", from: 227, clause: "3" }]);
    assert.deepEqual(Object.keys(row.decision), ["words", "text"]);
  });
});

test("a withdrawn ADR plans a withdrawn marker first, then supersedes, then ascending actor", () => {
  const front = [
    ...replacing(FRONT_223, "status", "status: withdrawn"),
    'withdrawal: {date: 2026-09-08, reason: "false", moved-to: none}',
  ];
  const files = withFront(223, front, {
    files: {
      [`docs/adr/1700-${SLUG_1700}.md`]: adr({
        front: [...replacing(FRONT_1700, "relations"), "relations:", "  - {kind: supersedes, adr: 223}"],
        h1: "# ADR-1700: A proposer answers a search with at most one organisation",
        body: NEW_BODY,
      }),
    },
  });
  withRepo(files, (root) => {
    const row = buildIndex(loadAdrs(root)).adrs.find((r) => r.number === 223);
    assert.deepEqual(row.status, { set: "withdrawn", derived: "withdrawn" });
    assert.deepEqual(row.markers, [
      { kind: "withdrawn", from: null, clause: null },
      { kind: "supersedes", from: 1700, clause: null },
      { kind: "retires", from: 227, clause: "3" },
    ]);
  });
});

test("rows are ordered by number", () => {
  withRepo(corpus(), (root) => {
    assert.deepEqual(buildIndex(loadAdrs(root)).adrs.map((r) => r.number), [223, 227, 1700]);
  });
});

test("--write writes the index, and --check is then clean", () => {
  withRepo(corpus(), (root) => {
    const wrote = run(root, { write: true });
    assert.equal(wrote.code, 0);
    assert.ok(existsSync(join(root, INDEX_FILE)));
    const parsed = JSON.parse(readFileSync(join(root, INDEX_FILE), "utf8"));
    assert.equal(parsed.adrs.length, 3);
    const checked = run(root, { write: false });
    assert.equal(checked.code, 0);
    assert.deepEqual(checked.diffs, []);
  });
});

test("--check fails with exit 1 on a missing or stale index", () => {
  withRepo(corpus(), (root) => {
    const missing = run(root, { write: false });
    assert.equal(missing.code, 1);
    assert.deepEqual(missing.diffs, ["docs/adr/index.json is stale"]);
    run(root, { write: true });
    writeFileSync(join(root, INDEX_FILE), readFileSync(join(root, INDEX_FILE), "utf8").replace("amended", "accepted"));
    const stale = run(root, { write: false });
    assert.equal(stale.code, 1);
    assert.deepEqual(stale.diffs, ["docs/adr/index.json is stale"]);
  });
});

test("--check fails with exit 2 on a schema or relation problem, and writes nothing", () => {
  const files = withRelations(227, FRONT_227, ['  - {kind: supersedes, adr: 223, clause: "3"}']);
  withRepo(files, (root) => {
    const r = run(root, { write: true });
    assert.equal(r.code, 2);
    assert.match(r.problems[0].message, /supersedes forbids a clause/);
    assert.equal(existsSync(join(root, INDEX_FILE)), false);
  });
});

test("the four schema cases of #1645 §5 each fail with the stated message, exit 2", () => {
  const cases = [
    [bothSides(), /an edge lives once/],
    [withRelations(227, FRONT_227, ['  - {kind: retires, adr: 223, clause: "9"}']), /target numbers 1, 2, 3, 4/],
    [withRelations(227, FRONT_227, ['  - {kind: supersedes, adr: 223, clause: "3"}']), /supersedes forbids a clause/],
    [withRelations(227, FRONT_227, ["  - {kind: retires, adr: 223}"]), /needs a clause/],
  ];
  for (const [files, pattern] of cases) {
    withRepo(files, (root) => {
      const r = run(root, { write: false });
      assert.equal(r.code, 2, String(pattern));
      assert.ok(r.problems.some((p) => pattern.test(p.message)), String(pattern));
    });
  }
});

test("a corpus with no front matter yields an empty index", () => {
  withRepo({ "docs/adr/0003-third-party.md": adr({ h1: "# ADR-0003: Third party", body: UNNUMBERED_BODY }) }, (root) => {
    const r = run(root, { write: true });
    assert.equal(r.code, 0);
    assert.deepEqual(JSON.parse(readFileSync(join(root, INDEX_FILE), "utf8")).adrs, []);
  });
});
