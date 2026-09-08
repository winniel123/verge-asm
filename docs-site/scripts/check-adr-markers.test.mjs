import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { incomingEdges, loadAdrs } from "./check-adr-index.mjs";
import { classify, regenerate, renderMarker, report, run } from "./check-adr-markers.mjs";

const SLUG_223 = "a-bar-is-authored-in-the-release-and-a-health-record-is-per-install-so-the-two-never-share-a-badge";
const SLUG_227 = "caida-publishes-an-org-name-search-so-the-join-replaces-its-first-leg-and-keeps-its-second";
const SLUG_1700 = "a-proposer-answers-a-search-with-at-most-one-organisation";
const SLUG_145 = "design-system-lives-in-the-repo";

const TITLE_223 =
  "a bar is authored in the release and a health record is per-install, so the two never share a badge";
const TITLE_227 =
  "CAIDA publishes an org-name search, so the join replaces its first leg and keeps its second";
const TITLE_1700 = "A proposer answers a search with at most one organisation";

const FILE_223 = `docs/adr/0223-${SLUG_223}.md`;
const FILE_227 = `docs/adr/0227-${SLUG_227}.md`;
const FILE_1700 = `docs/adr/1700-${SLUG_1700}.md`;
const FILE_145 = `docs/adr/0145-${SLUG_145}.md`;
const FILE_3 = "docs/adr/0003-third-party.md";

const RETIRE_223_3 =
  "> **Retired**, with no replacement, by " +
  `[ADR-0227: ${TITLE_227}](./0227-${SLUG_227}.md), 2026-09-07. <!-- adr-marker retires 227 -->`;
const AMEND_227_2 =
  `> **Amended** by [ADR-1700: ${TITLE_1700}](./1700-${SLUG_1700}.md), 2026-09-10. <!-- adr-marker amends 1700 -->`;

const FRONT_223 = [
  "number: 223",
  `title: "${TITLE_223}"`,
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
  `title: "${TITLE_227}"`,
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
  `title: "${TITLE_1700}"`,
  `slug: ${SLUG_1700}`,
  "date: 2026-09-10",
  "status: accepted",
  "source: grilling",
  "ticket: 1700",
  "map: 1638",
  'proof: {none: "a fixture"}',
  "relations:",
  '  - {kind: amends, adr: 227, clause: "2"}',
];

const FRONT_145 = [
  "number: 145",
  'title: "Design system lives in the repo"',
  `slug: ${SLUG_145}`,
  "date: 2026-08-28",
  "status: accepted",
  "source: fix",
  'proof: {none: "a fixture"}',
];

const NUMBERED_BODY = [
  "## Context",
  "",
  "Three tickets ask one question.",
  "",
  "## Decision",
  "",
  "The rule.",
  "",
  "### 1. The first rule",
  "",
  "Text.",
  "",
  "### 2. The second rule",
  "",
  "Text.",
  "",
  "### 3. The two CAIDA proposers are barred, not repaired",
  "",
  "They are barred on reason 2.",
  "",
  "_Retired 2026-09-07 by ADR-0227, hand-written._",
  "",
  "### 4. The fourth rule",
  "",
  "Text.",
  "",
];

const UNNUMBERED_BODY = ["## Context", "", "Text.", "", "## Decision", "", "The rule.", ""];

const NEW_BODY = ["## Decision", "", "One organisation. Rejected: a list.", "", "## 1. Rationale", "", "Text.", ""];

function adr({ front, h1, body }) {
  const head = front ? ["---", ...front, "---", ""] : [];
  return [...head, h1, "", ...body].join("\n");
}

const H1_223 = `# ADR-0223: ${TITLE_223}`;
const H1_227 = `# ADR-0227: ${TITLE_227}`;
const H1_1700 = `# ADR-1700: ${TITLE_1700}`;

function corpus(overrides = {}) {
  return {
    [FILE_3]: adr({ h1: "# ADR-0003: Third party", body: UNNUMBERED_BODY }),
    [FILE_223]: adr({ front: FRONT_223, h1: H1_223, body: NUMBERED_BODY }),
    [FILE_227]: adr({ front: FRONT_227, h1: H1_227, body: NUMBERED_BODY }),
    [FILE_1700]: adr({ front: FRONT_1700, h1: H1_1700, body: NEW_BODY }),
    ...overrides,
  };
}

function withRepo(files, fn) {
  const root = mkdtempSync(join(tmpdir(), "adr-markers-"));
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

function withRelations(front, rows) {
  return [...replacing(front, "relations"), ...(rows.length ? ["relations:", ...rows] : [])];
}

function regenerated(files, file) {
  return withRepo(files, (root) => {
    const adrs = loadAdrs(root);
    const incoming = incomingEdges(adrs);
    const a = [...adrs.values()].find((x) => x.file === file);
    return regenerate(a, incoming.get(a.number) ?? [], adrs).slice(a.frontRaw.length).split("\n");
  });
}

function renders(files, file) {
  return regenerated(files, file).filter((l) => /<!-- adr-marker /.test(l));
}

function withdrawn223(movedTo, reason = "the code no longer exhibits the rule") {
  return [
    ...replacing(FRONT_223, "status", "status: withdrawn"),
    `withdrawal: {date: 2026-09-10, reason: "${reason}", moved-to: ${movedTo}}`,
  ];
}

test("the retires render matches the #1644 resolution byte for byte", () => {
  assert.deepEqual(renders(corpus(), FILE_223), [RETIRE_223_3]);
});

test("the amends render", () => {
  assert.deepEqual(renders(corpus(), FILE_227), [AMEND_227_2]);
});

test("the supersedes render", () => {
  const files = corpus({
    [FILE_1700]: adr({ front: withRelations(FRONT_1700, ["  - {kind: supersedes, adr: 223}"]), h1: H1_1700, body: NEW_BODY }),
  });
  assert.deepEqual(renders(files, FILE_223), [
    `> **Superseded** by [ADR-1700: ${TITLE_1700}](./1700-${SLUG_1700}.md), 2026-09-10. <!-- adr-marker supersedes 1700 -->`,
    RETIRE_223_3,
  ]);
});

test("the withdrawn render: moved-to none is Nothing replaces it", () => {
  const files = corpus({ [FILE_223]: adr({ front: withdrawn223("none"), h1: H1_223, body: NUMBERED_BODY }) });
  assert.equal(
    renders(files, FILE_223)[0],
    "> **Withdrawn** 2026-09-10: the code no longer exhibits the rule. Nothing replaces it. <!-- adr-marker withdrawn -->",
  );
});

test("the withdrawn render: an ADR number is an ADR link, and a legacy target takes its title from the H1", () => {
  const files = corpus({ [FILE_223]: adr({ front: withdrawn223("3"), h1: H1_223, body: NUMBERED_BODY }) });
  assert.equal(
    renders(files, FILE_223)[0],
    "> **Withdrawn** 2026-09-10: the code no longer exhibits the rule. " +
      "Moved to [ADR-0003: Third party](./0003-third-party.md). <!-- adr-marker withdrawn -->",
  );
});

test("the withdrawn render: a path is a relative link, and a trailing period on the reason is not doubled", () => {
  const files = corpus({
    [FILE_223]: adr({ front: withdrawn223("docs/spec/x.md", "duplicates a live document."), h1: H1_223, body: NUMBERED_BODY }),
    "docs/spec/x.md": "# x\n",
  });
  assert.equal(
    renders(files, FILE_223)[0],
    "> **Withdrawn** 2026-09-10: duplicates a live document. Moved to [docs/spec/x.md](../spec/x.md). <!-- adr-marker withdrawn -->",
  );
});

test("rests-on, bounds, and sibling write nothing", () => {
  const files = corpus({
    [FILE_1700]: adr({
      front: withRelations(FRONT_1700, [
        "  - {kind: rests-on, adr: 223}",
        '  - {kind: bounds, adr: 223, clause: "1"}',
        "  - {kind: sibling, adr: 227}",
      ]),
      h1: H1_1700,
      body: NEW_BODY,
    }),
  });
  assert.deepEqual(renders(files, FILE_223), [RETIRE_223_3]);
  assert.deepEqual(renders(files, FILE_227), []);
  assert.deepEqual(renders(files, FILE_1700), []);
});

test("a clause marker is the first line under its numbered heading, one blank line above and below", () => {
  const lines = regenerated(corpus(), FILE_223);
  const at = lines.indexOf("### 3. The two CAIDA proposers are barred, not repaired");
  assert.deepEqual(lines.slice(at, at + 5), [
    "### 3. The two CAIDA proposers are barred, not repaired",
    "",
    RETIRE_223_3,
    "",
    "They are barred on reason 2.",
  ]);
  assert.equal(lines.filter((l) => /adr-marker/.test(l)).length, 1);
});

test("supersedes, withdrawn, and a clause-less amends sit under the H1", () => {
  const files = corpus({
    [FILE_223]: adr({ front: withdrawn223("none"), h1: H1_223, body: NUMBERED_BODY }),
    [FILE_1700]: adr({
      front: withRelations(FRONT_1700, ["  - {kind: supersedes, adr: 223}", "  - {kind: amends, adr: 3}"]),
      h1: H1_1700,
      body: NEW_BODY,
    }),
  });
  const lines = regenerated(files, FILE_223);
  assert.equal(lines[1], H1_223);
  assert.equal(lines[2], "");
  assert.match(lines[3], /^> \*\*Withdrawn\*\*/);
  assert.equal(lines[4], "");
  assert.match(lines[5], /^> \*\*Superseded\*\*/);
  assert.equal(lines[6], "");
  assert.equal(lines[7], "## Context");
  const legacy = regenerated(files, FILE_3);
  assert.deepEqual(legacy.slice(0, 5), [
    "# ADR-0003: Third party",
    "",
    `> **Amended** by [ADR-1700: ${TITLE_1700}](./1700-${SLUG_1700}.md), 2026-09-10. <!-- adr-marker amends 1700 -->`,
    "",
    "## Context",
  ]);
});

test("several markers at one heading are separate blockquotes ordered withdrawn, supersedes, then ascending actor", () => {
  const files = corpus({
    [FILE_223]: adr({ front: withdrawn223("none"), h1: H1_223, body: NUMBERED_BODY }),
    [FILE_227]: adr({ front: withRelations(FRONT_227, ["  - {kind: amends, adr: 223}"]), h1: H1_227, body: NUMBERED_BODY }),
    [FILE_1700]: adr({ front: withRelations(FRONT_1700, ["  - {kind: supersedes, adr: 223}"]), h1: H1_1700, body: NEW_BODY }),
    [FILE_145]: adr({ front: [...FRONT_145, "relations:", "  - {kind: amends, adr: 223}"], h1: "# ADR-0145: Design system lives in the repo", body: UNNUMBERED_BODY }),
  });
  const lines = regenerated(files, FILE_223).slice(1, 10);
  assert.deepEqual(
    lines.map((l) => (/adr-marker/.test(l) ? l.match(/<!-- adr-marker ([^>]+) -->/)[1] : l)),
    [H1_223, "", "withdrawn", "", "supersedes 1700", "", "amends 145", "", "amends 227"],
  );
});

test("a blockquote without the sentinel is prose and is left alone", () => {
  const body = ["> Superseded by ADR-0002 on 2025-01-01.", "", ...NUMBERED_BODY];
  const files = corpus({ [FILE_223]: adr({ front: FRONT_223, h1: H1_223, body }) });
  const lines = regenerated(files, FILE_223);
  assert.equal(lines[3], "> Superseded by ADR-0002 on 2025-01-01.");
  assert.ok(lines.includes("_Retired 2026-09-07 by ADR-0227, hand-written._"));
});

test("a sentinel inside a fenced block is an example, not a marker", () => {
  const body = ["## Decision", "", "```markdown", RETIRE_223_3, "```", "", "## 1. Rationale", "", "Text.", ""];
  const files = corpus({ [FILE_1700]: adr({ front: FRONT_1700, h1: H1_1700, body }) });
  const lines = regenerated(files, FILE_1700);
  assert.deepEqual(lines.filter((l) => l === RETIRE_223_3).length, 1);
  assert.equal(lines[lines.indexOf("```markdown") + 1], RETIRE_223_3);
});

test("a heading with no blank line after it still gets one blank line below the marker", () => {
  const body = ["## Decision", "", "### 3. Tight", "Text.", ""];
  const files = corpus({ [FILE_223]: adr({ front: FRONT_223, h1: H1_223, body }) });
  const lines = regenerated(files, FILE_223);
  const at = lines.indexOf("### 3. Tight");
  assert.deepEqual(lines.slice(at, at + 4), ["### 3. Tight", "", RETIRE_223_3, ""]);
  assert.equal(lines[at + 4], "Text.");
});

test("regeneration is idempotent", () => {
  withRepo(corpus(), (root) => {
    const first = run(root, { write: true });
    assert.equal(first.code, 0);
    assert.deepEqual(first.wrote.sort(), [FILE_223, FILE_227]);
    const before = readFileSync(join(root, FILE_223), "utf8");
    const second = run(root, { write: true });
    assert.equal(second.code, 0);
    assert.deepEqual(second.wrote, []);
    assert.equal(readFileSync(join(root, FILE_223), "utf8"), before);
  });
});

test("--write writes the markers, and --check is then clean with exit 0", () => {
  withRepo(corpus(), (root) => {
    run(root, { write: true });
    const text = readFileSync(join(root, FILE_223), "utf8");
    assert.ok(text.includes(`\n### 3. The two CAIDA proposers are barred, not repaired\n\n${RETIRE_223_3}\n\nThey are barred`));
    assert.ok(text.startsWith("---\nnumber: 223\n"));
    const checked = run(root, { write: false });
    assert.equal(checked.code, 0);
    assert.deepEqual(checked.diffs, []);
  });
});

test("a relation without a marker fails", () => {
  withRepo(corpus(), (root) => {
    const r = run(root, { write: false });
    assert.equal(r.code, 1);
    const d = r.diffs.find((x) => x.file === FILE_223);
    assert.ok(d, "ADR-0223 is reported");
    assert.deepEqual(d.hunks, [{ line: 34, kind: "missing", want: RETIRE_223_3, have: null }]);
    assert.equal(readFileSync(join(root, FILE_223), "utf8"), corpus()[FILE_223]);
  });
});

test("a marker without a relation fails", () => {
  withRepo(corpus(), (root) => {
    run(root, { write: true });
    const stray = "> **Amended** by [ADR-0227: x](./0227-x.md), 2026-09-07. <!-- adr-marker amends 227 -->";
    const path = join(root, FILE_223);
    writeFileSync(path, readFileSync(path, "utf8").replace("### 1. The first rule\n", `### 1. The first rule\n\n${stray}\n`));
    const r = run(root, { write: false });
    assert.equal(r.code, 1);
    assert.deepEqual(r.diffs.find((x) => x.file === FILE_223).hunks, [{ line: 26, kind: "stray", want: null, have: stray }]);
  });
});

test("a hand-edited marker fails", () => {
  withRepo(corpus(), (root) => {
    run(root, { write: true });
    const path = join(root, FILE_223);
    const edited = RETIRE_223_3.replace("with no replacement", "with a replacement");
    writeFileSync(path, readFileSync(path, "utf8").replace(RETIRE_223_3, edited));
    const r = run(root, { write: false });
    assert.equal(r.code, 1);
    assert.deepEqual(r.diffs.find((x) => x.file === FILE_223).hunks, [
      { line: 34, kind: "edited", want: RETIRE_223_3, have: edited },
    ]);
  });
});

test("a stray marker on a legacy file with no front matter fails", () => {
  const stray = "> **Superseded** by [ADR-0227: x](./0227-x.md), 2026-09-07. <!-- adr-marker supersedes 227 -->";
  const files = corpus({ [FILE_3]: adr({ h1: "# ADR-0003: Third party", body: [stray, "", ...UNNUMBERED_BODY] }) });
  withRepo(files, (root) => {
    const r = run(root, { write: false });
    assert.equal(r.code, 1);
    assert.deepEqual(r.diffs.find((x) => x.file === FILE_3).hunks, [{ line: 3, kind: "stray", want: null, have: stray }]);
  });
});

test("the report prints the want and have lines", () => {
  const lines = report({
    file: FILE_223,
    hunks: [
      { line: 33, kind: "missing", want: RETIRE_223_3, have: null },
      { line: 40, kind: "stray", want: null, have: "> x <!-- adr-marker amends 1 -->" },
      { line: 41, kind: "other", want: "", have: null },
    ],
  });
  assert.deepEqual(lines, [
    `${FILE_223}:33: a relation without a marker`,
    `  want: ${RETIRE_223_3}`,
    "  have: <none>",
    `${FILE_223}:40: a marker without a relation`,
    "  want: <none>",
    "  have: > x <!-- adr-marker amends 1 -->",
    `${FILE_223}:41: differs from the regeneration`,
    "  want: <blank>",
    "  have: <none>",
  ]);
});

test("--check fails with exit 2 on a schema or relation problem, and writes nothing", () => {
  const files = corpus({
    [FILE_227]: adr({ front: withRelations(FRONT_227, ['  - {kind: retires, adr: 223, clause: "9"}']), h1: H1_227, body: NUMBERED_BODY }),
  });
  withRepo(files, (root) => {
    const r = run(root, { write: true });
    assert.equal(r.code, 2);
    assert.match(r.problems[0].message, /target numbers 1, 2, 3, 4/);
    assert.equal(readFileSync(join(root, FILE_223), "utf8"), files[FILE_223]);
  });
});

test("a corpus with no front matter is clean", () => {
  withRepo({ [FILE_3]: corpus()[FILE_3] }, (root) => {
    const r = run(root, { write: false });
    assert.equal(r.code, 0);
    assert.deepEqual(r.diffs, []);
  });
});

test("renderMarker refuses a kind that writes no marker", () => {
  withRepo(corpus(), (root) => {
    const adrs = loadAdrs(root);
    assert.throws(() => renderMarker({ kind: "rests-on", from: 227 }, adrs, adrs.get(223)), /no marker for rests-on/);
  });
});

test("a target with no heading to anchor under fails with exit 2, and --write then touches no file", () => {
  const files = corpus({ [FILE_3]: ["Text with no H1.", "", ...UNNUMBERED_BODY].join("\n") });
  files[FILE_1700] = adr({ front: withRelations(FRONT_1700, ["  - {kind: amends, adr: 3}"]), h1: H1_1700, body: NEW_BODY });
  withRepo(files, (root) => {
    const r = run(root, { write: true });
    assert.equal(r.code, 2);
    assert.deepEqual(r.wrote, []);
    assert.match(r.problems[0].message, /0003-third-party\.md: no heading to anchor the amends marker under/);
    assert.equal(readFileSync(join(root, FILE_223), "utf8"), files[FILE_223]);
  });
});

test("classify names the three failure cases", () => {
  const sentinel = "> x <!-- adr-marker amends 1 -->";
  assert.equal(classify(sentinel, null), "missing");
  assert.equal(classify(null, sentinel), "stray");
  assert.equal(classify(sentinel, `${sentinel} `), "edited");
  assert.equal(classify("", "Text."), "other");
});
