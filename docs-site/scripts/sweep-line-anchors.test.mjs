import test from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";
import { parse } from "./doclint/engine.mjs";
import { scanLineAnchorsFromTree } from "./citations/lineanchor.mjs";
import { ROWS } from "./citations/rows.mjs";
import { derive, splitToken } from "./sweep/derive.mjs";
import { rewriteDocument, replacementFor, trailingGlue, namesAnotherSite } from "./sweep/rewrite.mjs";
import { selectEntries, planFor, scanDocuments, reportRecord } from "./sweep-line-anchors.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(SCRIPT_DIR, "..", "..");

// The Go row shells out to `go run ./cmd/godecls` from the repository root, so a fixture
// target must sit under that root. A dot prefix keeps it out of `go vet ./...`.
const FIXTURE = `.sweep-fixture-${process.pid}`;

function fixture(files) {
  const root = join(REPO_ROOT, FIXTURE);
  rmSync(root, { recursive: true, force: true });
  for (const [path, body] of Object.entries(files)) {
    const abs = join(root, path);
    mkdirSync(dirname(abs), { recursive: true });
    writeFileSync(abs, body);
  }
  return Object.keys(files).map((p) => `${FIXTURE}/${p}`);
}

// classify resolves against the tracked tree, and a fixture is untracked by design.
function envFor(paths) {
  const files = new Set(paths);
  const dirs = new Set();
  for (const f of files) {
    const parts = f.split("/");
    for (let i = 1; i < parts.length; i++) dirs.add(parts.slice(0, i).join("/"));
  }
  const extensions = new Set([...files].map((f) => f.slice(f.lastIndexOf("."))));
  return {
    repoRoot: REPO_ROOT,
    tracked: { files, dirs },
    extensions,
    roots: new Set([...files].map((f) => f.split("/")[0])),
    exempt: () => null,
  };
}

// The fixture mirrors a real prefix, so each row predicate stays its own (#1971, #1972).
const AT_FIXTURE = ROWS.map((row) => ({
  ...row,
  matches: (p) => p.startsWith(`${FIXTURE}/`) && row.matches(p.slice(FIXTURE.length + 1)),
}));

function hit(token, { file = "docs/spec/fixture.md", line = 1 } = {}) {
  return { token, file, line, kind: "code", start: 0, end: token.length + 2 };
}

function run(files, tokens) {
  const paths = fixture(files);
  const env = envFor(paths);
  return derive(REPO_ROOT, env, tokens.map((t) => hit(t)), AT_FIXTURE);
}

test.after(() => rmSync(join(REPO_ROOT, FIXTURE), { recursive: true, force: true }));

test("splitToken reads both retired spellings and both ends", () => {
  assert.deepEqual(splitToken("a/b.go:12"), { value: "a/b.go", fromLine: 12, toLine: 12 });
  assert.deepEqual(splitToken("a/b.go:12-20"), { value: "a/b.go", fromLine: 12, toLine: 20 });
  assert.deepEqual(splitToken("a/b.go#L12"), { value: "a/b.go", fromLine: 12, toLine: 12 });
  assert.deepEqual(splitToken("a/b.go#L12-L20"), { value: "a/b.go", fromLine: 12, toLine: 20 });
  assert.equal(splitToken("a/b.go"), null);
});

const GO_LINES = [
  "package fixture",
  "",
  "// Doc names the function.",
  "func Alpha() int {",
  "\treturn 1",
  "}",
  "",
  "const (",
  "\tBeta = 2",
  "\tGamma = 3",
  ")",
  "",
  "type Delta struct{}",
  "",
  "func (d *Delta) Epsilon() {}",
  "",
  "var Zeta, Eta = 1, 2",
  "",
];
const GO_SOURCE = GO_LINES.join("\n");

// A test names the line it cites, so an edit to the fixture never silently re-points a token.
function goLine(text) {
  const exact = GO_LINES.indexOf(text);
  const at = exact >= 0 ? exact : GO_LINES.findIndex((l) => l.includes(text));
  if (at < 0) throw new Error(`no fixture line holds ${text}`);
  return at + 1;
}

const goRange = (from, to) => `${FIXTURE}/go/decls.go:${goLine(from)}-${goLine(to)}`;

const goToken = (text) => `${FIXTURE}/go/decls.go:${goLine(text)}`;

test("a line inside a top-level declaration yields that declaration's name", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goToken("return 1")]);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
  assert.equal(r.row.name, "go");
});

test("a doc comment reads as part of the declaration", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goToken("Doc names")]);
  assert.equal(r.anchor, "Alpha");
});

test("a line inside a parenthesised const group yields the member's name", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goToken("Gamma")]);
  assert.equal(r.anchor, "Gamma");
});

test("a method is spelled Receiver.Method, with no star", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goToken("Epsilon")]);
  assert.equal(r.anchor, "Delta.Epsilon");
});

test("a range enclosed by one declaration yields that declaration", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goRange("func Alpha", "}")]);
  assert.equal(r.anchor, "Alpha");
});

test("a range that spans two declarations degrades", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goRange("return 1", "type Delta")]);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /encloses the cited line/);
});

test("two names on one declaration line degrade, because the sweep picks neither", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goToken("Zeta")]);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /two declarations share/);
});

test("a blank line degrades, and is not guessed", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [`${FIXTURE}/go/decls.go:${goLine("const (") - 1}`]);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /is blank/);
});

test("a line past the end of the file degrades", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [`${FIXTURE}/go/decls.go:400`]);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /past the end/);
});

test("a line the language declares nothing at degrades", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goToken("package fixture")]);
  assert.equal(r.outcome, "degraded");
});

test("a sqlc query name comes off its `-- name:` marker", () => {
  const source = ["-- name: First :one", "SELECT 1;", "", "-- name: Second :many", "SELECT 2;"].join("\n");
  const [r] = run({ "db/queries/q.sql": source }, [`${FIXTURE}/db/queries/q.sql:5`]);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Second");
  assert.equal(r.row.name, "sqlc");
});

test("a template define name comes off its `{{define}}`", () => {
  const source = ['{{define "shell"}}', "<main></main>", "{{end}}"].join("\n");
  const [r] = run({ "design-system/templates/t.tmpl": source }, [
    `${FIXTURE}/design-system/templates/t.tmpl:2`,
  ]);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "shell");
  assert.equal(r.row.name, "template");
});

test("a Markdown line yields the slug of the heading above it, and the nearest one wins", () => {
  const source = ["# Top", "", "text", "", "## Inner Part", "", "more", ""].join("\n");
  const [outer, inner] = run({ "docs/page.md": source }, [
    `${FIXTURE}/docs/page.md:3`,
    `${FIXTURE}/docs/page.md:7`,
  ]).sort((a, b) => a.fromLine - b.fromLine);
  assert.equal(outer.anchor, "top");
  assert.equal(inner.anchor, "inner-part");
  assert.equal(inner.row.name, "markdown");
});

test("a line above the first heading degrades", () => {
  const source = ["text", "", "# Top", ""].join("\n");
  const [r] = run({ "docs/page.md": source }, [`${FIXTURE}/docs/page.md:1`]);
  assert.equal(r.outcome, "degraded");
});

test("a de-duplicated slug degrades, because the gate refuses that spelling", () => {
  const source = ["# Same", "", "a", "", "# Same", "", "b", ""].join("\n");
  const [r] = run({ "docs/page.md": source }, [`${FIXTURE}/docs/page.md:7`]);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /de-duplicated slug/);
});

test("a slug outside the anchor character class degrades", () => {
  const source = ["# Café", "", "text", ""].join("\n");
  const [r] = run({ "docs/page.md": source }, [`${FIXTURE}/docs/page.md:3`]);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /anchor character class/);
});

test("a target with no anchor vocabulary degrades", () => {
  const source = ["-- +goose Up", "ALTER TABLE t ADD COLUMN c text;", ""].join("\n");
  const [r] = run({ "db/migrations/1.sql": source }, [`${FIXTURE}/db/migrations/1.sql:2`]);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /no anchor vocabulary/);
});

test("a path that left the tree is held back, because a bare path would red the gate", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE });
  const results = derive(REPO_ROOT, envFor(paths), [hit(`${FIXTURE}/go/gone.go:4`)], AT_FIXTURE);
  assert.equal(results[0].outcome, "held");
  assert.match(results[0].reason, /no longer in the tree/);
});

test("a path this gate passes over degrades, because the bare path stays unjudged", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE });
  const results = derive(REPO_ROOT, envFor(paths), [hit("kafka.apache.org/docs/thing.go:4")], AT_FIXTURE);
  assert.equal(results[0].outcome, "degraded");
  assert.match(results[0].reason, /does not resolve/);
});

test("a following code span would read as a snippet, so a token it does not hold degrades", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE });
  const token = goToken("return 1");
  const [holds, misses] = derive(
    REPO_ROOT,
    envFor(paths),
    [
      { ...hit(token), snippet: "return 1" },
      { ...hit(token, { line: 2 }), snippet: "// nothing like this" },
    ],
    AT_FIXTURE,
  );
  assert.equal(holds.outcome, "anchor");
  assert.equal(misses.outcome, "degraded");
  assert.match(misses.reason, /would read .* as a snippet/);
});

test("replacementFor keeps the path and drops the line when the token degrades", () => {
  assert.equal(replacementFor({ outcome: "anchor", value: "a/b.go", anchor: "Alpha" }), "a/b.go#Alpha");
  assert.equal(replacementFor({ outcome: "degraded", value: "a/b.go" }), "a/b.go");
});

// The CLI hands rewriteDocument one derived result per occurrence, so a test builds the same.
function rewrite(markdown, anchors) {
  const conversions = scanLineAnchorsFromTree(parse(markdown))
    .filter((h) => h.token in anchors)
    .map((h) => {
      const [value, anchor] = [h.token.replace(/[:#].*$/, ""), anchors[h.token]];
      return anchor === null
        ? { ...h, outcome: "degraded", value }
        : { ...h, outcome: "anchor", value, anchor };
    });
  return rewriteDocument(markdown, conversions);
}

test("a code span and a link target both convert", () => {
  const md = "See `a/b.go:4` and [it](a/b.go:4).\n";
  assert.equal(rewrite(md, { "a/b.go:4": "Alpha" }), "See `a/b.go#Alpha` and [it](a/b.go#Alpha).\n");
});

test("a longer token beside a shorter one is untouched", () => {
  const md = "`a/b.go:4` and `a/b.go:42`\n";
  assert.equal(rewrite(md, { "a/b.go:4": "Alpha" }), "`a/b.go#Alpha` and `a/b.go:42`\n");
});

test("a fenced block is sample text, so a rewrite never reaches it", () => {
  const md = ["`a/b.go:4`", "", "```", "a/b.go:4", "```", ""].join("\n");
  const out = rewrite(md, { "a/b.go:4": "Alpha" });
  assert.match(out, /`a\/b\.go#Alpha`/);
  assert.match(out, /```\na\/b\.go:4\n```/);
});

test("a token this run does not convert keeps its line", () => {
  const md = "`a/b.go:4` and `c/d.go:9`\n";
  assert.equal(rewrite(md, { "a/b.go:4": "Alpha" }), "`a/b.go#Alpha` and `c/d.go:9`\n");
});

test("a suffix the anchor class would swallow is consumed by the rewrite", () => {
  const md = "See `a/b.go:265+` here.\n";
  const hits = scanLineAnchorsFromTree(parse(md));
  assert.equal(trailingGlue(md, hits[0]), "+");
  const out = rewriteDocument(md, [
    { ...hits[0], outcome: "anchor", value: "a/b.go", anchor: "Apply" },
  ]);
  assert.equal(out, "See `a/b.go#Apply` here.\n");
});

test("a second line reference goes with the token it qualifies", () => {
  const md = "See `a/b.go:978,997` and `c/d.md:37,:41` here.\n";
  const hits = scanLineAnchorsFromTree(parse(md));
  assert.equal(trailingGlue(md, hits[0]), ",997");
  assert.equal(trailingGlue(md, hits[1]), ",:41");
  assert.ok(namesAnotherSite(trailingGlue(md, hits[0])));
  assert.ok(!namesAnotherSite("+"));
  const out = rewriteDocument(md, [
    { ...hits[0], outcome: "degraded", value: "a/b.go" },
    { ...hits[1], outcome: "degraded", value: "c/d.md" },
  ]);
  // A residue left behind keeps a line number no arm can see (#1976).
  assert.equal(out, "See `a/b.go` and `c/d.md` here.\n");
});

test("a token naming a second place degrades, because one anchor represents neither", () => {
  // The verdict lands before the path resolves, so no row and no tree are read.
  const [result] = derive(REPO_ROOT, envFor([]), [{ ...hit("a/b.go:978"), glue: ",997" }], []);
  assert.equal(result.outcome, "degraded");
  assert.match(result.reason, /names another place at `,997`/);
});

test("one token converts at one site and holds at another", () => {
  const md = "`a/b.go:4` once, and `a/b.go:4` twice.\n";
  const hits = scanLineAnchorsFromTree(parse(md));
  const out = rewriteDocument(md, [
    { ...hits[0], outcome: "anchor", value: "a/b.go", anchor: "Alpha" },
    { ...hits[1], outcome: "degraded", value: "a/b.go" },
  ]);
  assert.equal(out, "`a/b.go#Alpha` once, and `a/b.go` twice.\n");
});

test("a link label holding a code span converts both, and the outer node wins", () => {
  const md = "See [the queue `a/b.go:4`](a/b.go:4).\n";
  const hits = scanLineAnchorsFromTree(parse(md));
  assert.equal(hits.length, 2);
  const out = rewriteDocument(
    md,
    hits.map((h) => ({ ...h, outcome: "anchor", value: "a/b.go", anchor: "Alpha" })),
  );
  assert.equal(out, "See [the queue `a/b.go#Alpha`](a/b.go#Alpha).\n");
  assert.equal(scanLineAnchorsFromTree(parse(out)).length, 0);
});

test("a nested span the run spells two ways rewrites nothing", () => {
  const md = "See [the queue `a/b.go:4`](a/b.go:4).\n";
  const hits = scanLineAnchorsFromTree(parse(md));
  assert.throws(
    () =>
      rewriteDocument(md, [
        { ...hits[0], outcome: "anchor", value: "a/b.go", anchor: "Alpha" },
        { ...hits[1], outcome: "degraded", value: "a/b.go" },
      ]),
    /two ways/,
  );
});

test("a reversed range degrades, and mints no anchor from a region holding neither line", () => {
  const token = `${FIXTURE}/go/decls.go:${goLine("var Zeta")}-${goLine("return 1")}`;
  const [r] = run({ "go/decls.go": GO_SOURCE }, [token]);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /runs backwards/);
});

test("a burn-down entry naming a deleted document leaves the dry run alive", () => {
  const found = scanDocuments(REPO_ROOT, [`docs/gone-${process.pid}.md`]);
  assert.equal(found.size, 0);
});

test("reportRecord carries the anchor or the reason, so the record outlives the run", () => {
  const record = reportRecord([
    { file: "d.md", line: 3, token: "a/b.go:4", outcome: "anchor", value: "a/b.go", anchor: "Alpha", row: { name: "go" } },
    { file: "d.md", line: 9, token: "a/c.go:7", outcome: "degraded", reason: "line 7 of a/c.go is blank" },
  ]);
  assert.equal(record[0].anchor, "a/b.go#Alpha");
  assert.equal(record[0].row, "go");
  assert.equal(record[1].reason, "line 7 of a/c.go is blank");
  assert.equal(record[1].anchor, undefined);
});

test("selectEntries reads a path as a document or a directory prefix", () => {
  const entries = [
    { file: "docs/spec/a.md", token: "x/y.go:1" },
    { file: "docs/adr/0001-b.md", token: "x/y.go:2" },
  ];
  assert.equal(selectEntries(entries, []).length, 2);
  assert.deepEqual(selectEntries(entries, ["docs/spec"]), [entries[0]]);
  assert.deepEqual(selectEntries(entries, ["docs/adr/0001-b.md"]), [entries[1]]);
  assert.deepEqual(selectEntries(entries, ["docs/none"]), []);
});

test("planFor reports a listed token no scan finds", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE });
  const file = "docs/spec/fixture.md";
  const markdown = "`" + goToken("return 1") + "`\n";
  const hits = scanLineAnchorsFromTree(parse(markdown)).map((h) => ({ ...h, file }));
  const found = new Map([[file, { markdown, hits }]]);
  const entries = [
    { file, token: goToken("return 1") },
    { file, token: goToken("Beta") },
  ];
  const { results, missing } = planFor(REPO_ROOT, envFor(paths), entries, found, AT_FIXTURE);
  assert.equal(results.length, 1);
  assert.equal(results[0].anchor, "Alpha");
  assert.deepEqual(missing, [entries[1]]);
});
