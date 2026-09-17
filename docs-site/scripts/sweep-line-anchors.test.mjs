import test from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";
import { parse } from "./doclint/engine.mjs";
import { scanLineAnchorsFromTree } from "./citations/lineanchor.mjs";
import { ROWS } from "./citations/rows.mjs";
import { derive, resolveBasename, splitToken } from "./sweep/derive.mjs";
import { rewriteDocument, replacementFor, trailingGlue, namesAnotherSite } from "./sweep/rewrite.mjs";
import { auditAnchors, countByFamily } from "./sweep/audit.mjs";
import { familyOf } from "./citations/scope.mjs";
import { trackedBasenames, trackedExtensions } from "./citations/classify.mjs";
import { rivalName } from "./sweep/corroborate.mjs";
import {
  selectFiles,
  planFor,
  scanDocuments,
  pathsNamedIn,
  foreignBasenamesIn,
  visibleText,
  reportRecord,
  auditRecord,
  planWrites,
} from "./sweep-line-anchors.mjs";

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
  const extensions = trackedExtensions({ files });
  const basenames = trackedBasenames({ files });
  return {
    repoRoot: REPO_ROOT,
    tracked: { files, dirs },
    basenames,
    knownFile: (base) => basenames.has(base),
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

function hit(token, { file = "docs/spec/fixture.md", line = 1, lineText, namedInDocument } = {}) {
  return { token, file, line, kind: "code", start: 0, end: token.length + 2, lineText, namedInDocument };
}

function run(files, tokens) {
  const paths = fixture(files);
  const env = envFor(paths);
  return derive(REPO_ROOT, env, tokens.map((t) => hit(t)), AT_FIXTURE);
}

// The citing line is an argument here, so one fixture serves every corroboration case.
function runCiting(files, token, lineText) {
  const paths = fixture(files);
  return derive(REPO_ROOT, envFor(paths), [hit(token, { lineText })], AT_FIXTURE);
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

test("the citing line names another real declaration, so the drifted line degrades", () => {
  const citing = "| `Delta.Epsilon` | [`x.go:1`](../x.go) |";
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), citing);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /the citing line names `Delta\.Epsilon`/);
});

test("a package-qualified name on the citing line still names the declaration", () => {
  const citing = "| `decls.Delta.Epsilon` | `x.go:1` |";
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), citing);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /Delta\.Epsilon/);
});

test("a name behind another name on the citing line is still read", () => {
  // One pass over an exhausted iterator read only `Unknown`, and `Delta.Epsilon` went unseen.
  const citing = "| `Unknown` | `Delta.Epsilon` | `x.go:1` |";
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), citing);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /Delta\.Epsilon/);
});

test("the citing line names the derived region itself, so the conversion stands", () => {
  const citing = "`Alpha` returns one (`x.go:1`)";
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), citing);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
});

test("a name the target does not declare corroborates nothing, so the conversion stands", () => {
  const citing = "`SomeOtherThing` calls it (`x.go:1`)";
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), citing);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
});

test("the citation's own span names the path, and a path corroborates nothing", () => {
  const token = goToken("return 1");
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, `see \`${token}\``);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
});

test("a name outside a code span is prose, so it corroborates nothing", () => {
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), "Delta.Epsilon is near");
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
});

test("a hit carrying no citing line derives exactly as it did before the guard", () => {
  const [r] = run({ "go/decls.go": GO_SOURCE }, [goToken("return 1")]);
  assert.equal(r.outcome, "anchor");
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

test("a document a later edit deleted leaves the dry run alive", () => {
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

// scanDocuments reads the tree, and these cases need a document the tree does not hold.
function inMemory(documents, env) {
  const found = new Map();
  const hits = [];
  for (const [file, markdown] of Object.entries(documents)) {
    const lines = markdown.split("\n");
    const scanned = scanLineAnchorsFromTree(parse(markdown), { knownFile: env.knownFile }).map((h) => ({
      ...h,
      file,
      glue: trailingGlue(markdown, h),
      lineText: lines[h.line - 1] ?? "",
      namedInDocument: new Set(),
    }));
    found.set(file, { markdown, hits: scanned });
    hits.push(...scanned);
  }
  return { found, hits };
}

test("a conversion whose glue swallows a held bare line is held back with it (#2158)", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE });
  const env = envFor(paths);
  const token = goToken("return 1");
  const { hits } = inMemory({ "docs/spec/glued.md": `See \`${token}, :160\` here.\n` }, env);
  assert.deepEqual(hits.map((h) => h.token), [token, ":160"]);
  const results = derive(REPO_ROOT, env, hits, AT_FIXTURE);
  const bare = results.find((r) => r.token === ":160");
  const path = results.find((r) => r.token === token);
  assert.equal(bare.outcome, "held");
  assert.equal(path.outcome, "held");
  assert.match(path.reason, /converting it may swallow `:160`/);
});

test("a token the span repeats beside a held bare line is held back too (#2158)", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE });
  const env = envFor(paths);
  const token = goToken("return 1");
  const { found, hits } = inMemory(
    { "docs/spec/twice.md": `See \`${token} and ${token}, :160\` here.\n` },
    env,
  );
  // trailingGlue reads the first occurrence, so neither copy records the glue that is really there.
  assert.deepEqual(hits.map((h) => h.glue), ["", "", ""]);
  const results = derive(REPO_ROOT, env, hits, AT_FIXTURE);
  assert.deepEqual(results.map((r) => r.outcome), ["held", "held", "held"]);
  assert.deepEqual(planWrites(results.filter((r) => r.outcome !== "held"), found, env), []);
});

test("a glued document no longer aborts the write plan for a sound one (#2158)", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE });
  const env = envFor(paths);
  const { found, hits } = inMemory(
    {
      "docs/spec/glued.md": `See \`${goToken("return 1")}, :160\` here.\n`,
      "docs/spec/sound.md": `See \`${goToken("Beta")}\` here.\n`,
    },
    env,
  );
  const results = derive(REPO_ROOT, env, hits, AT_FIXTURE);
  const writes = planWrites(results.filter((r) => r.outcome !== "held"), found, env);
  assert.deepEqual(writes.map((w) => w.file), ["docs/spec/sound.md"]);
  assert.equal(writes[0].next, `See \`${FIXTURE}/go/decls.go#Beta\` here.\n`);
});

test("a mismatch the sweep cannot explain still refuses to write (#2158)", () => {
  const file = "docs/spec/fixture.md";
  const markdown = "See `a/b.go:4` here.\n";
  const hits = scanLineAnchorsFromTree(parse(markdown)).map((h) => ({ ...h, file }));
  const found = new Map([[file, { markdown, hits }]]);
  assert.throws(
    () =>
      planWrites(
        [
          { ...hits[0], outcome: "degraded", value: "a/b.go" },
          { ...hits[0], token: "c/d.go:9", outcome: "degraded", value: "c/d.go" },
        ],
        found,
        null,
      ),
    /converted 1 token\(s\), and 2 were planned/,
  );
});

// The list retired, so the conversion arm derives over every token the scan found (#1980).
test("planFor derives over every scanned token, and no list narrows it", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE });
  const file = "docs/spec/fixture.md";
  const markdown = "`" + goToken("return 1") + "` and `" + goToken("Beta") + "`\n";
  const hits = scanLineAnchorsFromTree(parse(markdown)).map((h) => ({ ...h, file }));
  const found = new Map([[file, { markdown, hits }]]);
  const results = planFor(REPO_ROOT, envFor(paths), found, AT_FIXTURE);
  assert.deepEqual(results.map((r) => r.anchor), ["Alpha", "Beta"]);
});

// SPEC §4.3 class A: the rival belongs to a second claim, and the site cell comes first.
test("a rival spelled only after the citation converts and enters the review queue", () => {
  const token = goToken("return 1");
  const citing = `\`${token}\` holds \`Delta.Epsilon\`, the shipped precedent`;
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, citing);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
  assert.deepEqual(r.review, { rival: "Delta.Epsilon", position: "after" });
});

test("the review queue names both candidates, so a human reads the pair", () => {
  const token = goToken("return 1");
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, `\`${token}\` holds \`Delta.Epsilon\``);
  const [record] = reportRecord([r]);
  assert.equal(record.anchor, `${FIXTURE}/go/decls.go#Alpha`);
  assert.equal(record.review.rival, "Delta.Epsilon");
});

// SPEC §5.3: rule 1 stands, so a rival on both sides still degrades.
test("a rival spelled before the citation outranks one spelled after", () => {
  const token = goToken("return 1");
  const citing = `\`Delta.Epsilon\` and \`${token}\` and \`Delta.Epsilon\` again`;
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, citing);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /names `Delta\.Epsilon` before the citation/);
});

// SPEC §4.3 class B: `POST /onboarding` collides with the declaration `server.onboarding`.
test("a route span spells no declaration name, so the conversion stands", () => {
  const citing = "| `POST /Delta.Epsilon` | Wizard step (`x.go:1`) |";
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), citing);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
  assert.equal(r.review, undefined);
});

test("a leading slash spells a route too, and a bare segment corroborates nothing", () => {
  const citing = "see `/Delta.Epsilon` and `x.go:1`";
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), citing);
  assert.equal(r.outcome, "anchor");
});

// A converted sibling on one line named its own target, and each read the other as a rival.
test("a second citation's own span names a target, not a declaration", () => {
  const citing = `\`${FIXTURE}/go/decls.go#Delta.Epsilon\` and \`${goToken("return 1")}\``;
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, goToken("return 1"), citing);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
});

test("familyOf reads the boundary's own entries, so no caller copies the list", () => {
  assert.equal(familyOf("docs/spec/audit-act.md"), "docs/spec");
  assert.equal(familyOf("docs/adr/0001-a.md"), "docs/adr");
  assert.equal(familyOf("CONTEXT.md"), ".");
  assert.equal(familyOf("docs/research/note.md"), null);
});

test("selectFiles reads a path as a document or a directory prefix", () => {
  const files = [`${REPO_ROOT}/docs/spec/a.md`, `${REPO_ROOT}/docs/adr/0001-b.md`];
  assert.equal(selectFiles(REPO_ROOT, files, []).length, 2);
  assert.deepEqual(selectFiles(REPO_ROOT, files, ["docs/spec"]), [files[0]]);
  assert.deepEqual(selectFiles(REPO_ROOT, files, ["docs/adr/0001-b.md"]), [files[1]]);
  assert.deepEqual(selectFiles(REPO_ROOT, files, ["docs/none"]), []);
});

// The audit reads a written region anchor, which the conversion arm's line scan never sees.
const AUDIT_DOC = [
  "# Page",
  "",
  `\`Alpha\` returns one (\`${FIXTURE}/go/decls.go#Alpha\`)`,
  "",
  `\`Delta.Epsilon\` (\`${FIXTURE}/go/decls.go#Alpha\`) caps it`,
  "",
  `\`${FIXTURE}/go/decls.go#Alpha\` holds \`Delta.Epsilon\``,
  "",
  `a bare mention of \`${FIXTURE}/go/decls.go#Alpha\``,
  "",
  `| \`POST /Delta.Epsilon\` | writes nothing (\`${FIXTURE}/go/decls.go#Alpha\`) |`,
  "",
  `\`${FIXTURE}/go/decls.go#Delta.Epsilon\` and \`${FIXTURE}/go/decls.go#Alpha\``,
  "",
].join("\n");

function auditFixture() {
  const paths = fixture({ "go/decls.go": GO_SOURCE, "docs/spec/page.md": AUDIT_DOC });
  const abs = join(REPO_ROOT, FIXTURE, "docs/spec/page.md");
  return auditAnchors(REPO_ROOT, envFor(paths), [abs], AT_FIXTURE);
}

test("the audit judges every written region anchor the tree holds", () => {
  const judged = auditFixture();
  assert.deepEqual(
    judged.map((a) => `${a.line}:${a.verdict}`),
    ["3:corroborated", "5:suspect", "7:suspect", "9:unproven", "11:unproven", "13:unproven", "13:unproven"],
  );
});

test("the audit reports the rival and its position, and never a new target", () => {
  const judged = auditFixture();
  const before = judged.find((a) => a.line === 5);
  const after = judged.find((a) => a.line === 7);
  assert.deepEqual([before.anchor, before.rival, before.position], ["Alpha", "Delta.Epsilon", "before"]);
  assert.deepEqual([after.anchor, after.rival, after.position], ["Alpha", "Delta.Epsilon", "after"]);
});

test("countByFamily reports suspect, corroborated and unproven per family", () => {
  const [row] = countByFamily(auditFixture());
  assert.deepEqual(row, {
    family: ".",
    anchors: 7,
    suspect: 2,
    corroborated: 1,
    unproven: 4,
    unreadable: 0,
  });
});

test("auditRecord carries the pair a human reads, and proposes no anchor", () => {
  const record = auditRecord(auditFixture());
  const suspect = record.find((a) => a.line === 5);
  assert.equal(suspect.verdict, "suspect");
  assert.equal(suspect.rival, "Delta.Epsilon");
  assert.equal(suspect.target, `${FIXTURE}/go/decls.go#Alpha`);
  assert.equal(record.find((a) => a.line === 9).rival, undefined);
});

// A signature span carries a space and no slash, and rule 3 must not read it as a route.
test("a span that spells a signature still corroborates the region", () => {
  const token = goToken("return 1");
  const citing = `\`func Alpha() int\` and \`Delta.Epsilon\` hold \`${token}\``;
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, citing);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
  assert.equal(r.review, undefined);
});

// A retired token hides its path from the path class, so the class reads it without the line.
test("a sibling line-anchor token names a target, not a declaration", () => {
  const token = goToken("return 1");
  const citing = `\`${FIXTURE}/go/other.go:12\` then \`${token}\``;
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, citing);
  assert.equal(r.outcome, "anchor");
  assert.equal(r.anchor, "Alpha");
});

// `pkg/dir.Name` parses as a path, and rule 1 must still reach it (SPEC §5.2, #2013).
test("a package-qualified name spelled before the citation degrades the token", () => {
  const token = goToken("return 1");
  const citing = `\`${FIXTURE}/go.Delta.Epsilon\` (\`${token}\`) caps it`;
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, citing);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /names `Delta\.Epsilon` before the citation/);
});

test("a package-qualified name spelled after the citation enters the review queue", () => {
  const token = goToken("return 1");
  const citing = `\`${token}\` holds \`${FIXTURE}/go.Delta.Epsilon\``;
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, citing);
  assert.equal(r.outcome, "anchor");
  assert.deepEqual(r.review, { rival: "Delta.Epsilon", position: "after" });
});

// An unused extension is not enough on its own: a real path keeps its spans out (#2013).
test("a sibling path corroborates nothing with an anchor, a line, or neither", () => {
  const token = goToken("return 1");
  for (const sibling of [
    `${FIXTURE}/go/decls.go#Delta.Epsilon`,
    `${FIXTURE}/go/decls.go:${goLine("func (d *Delta) Epsilon() {}")}`,
    `${FIXTURE}/Delta/Epsilon`,
    `${FIXTURE}/go/decls.key#Delta.Epsilon`,
    `web-state/Delta.Epsilon`,
  ]) {
    const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, `\`${sibling}\` and \`${token}\``);
    assert.equal(r.outcome, "anchor", sibling);
    assert.equal(r.anchor, "Alpha", sibling);
    assert.equal(r.review, undefined, sibling);
  }
});

test("a bare-prose copy of the token does not reorder the rival", () => {
  const token = goToken("return 1");
  const citing = `see ${token} and \`Delta.Epsilon\` then \`${token}\``;
  const [r] = runCiting({ "go/decls.go": GO_SOURCE }, token, citing);
  assert.equal(r.outcome, "degraded");
  assert.match(r.reason, /names `Delta\.Epsilon` before the citation/);
});

test("the audit counts a row that could not run apart from a missing vocabulary", () => {
  const paths = fixture({ "go/decls.go": GO_SOURCE, "docs/spec/page.md": AUDIT_DOC });
  const abs = join(REPO_ROOT, FIXTURE, "docs/spec/page.md");
  const broken = AT_FIXTURE.map((row) =>
    row.name === "go"
      ? { ...row, inventory: () => { throw new Error("godecls did not finish"); } }
      : row,
  );
  const judged = auditAnchors(REPO_ROOT, envFor(paths), [abs], broken);
  assert.equal(judged.length, 7);
  assert.ok(judged.every((a) => a.verdict === "unreadable"));
  assert.match(judged[0].detail, /the go row could not run: godecls did not finish/);
});

// The two forms #2120 reached: the sweep resolves one against the tree and holds the other.

test("a path-less token is held, because degrading it would write an empty citation", () => {
  const paths = fixture({ "go/one.go": "package one\n\nfunc One() {}\n" });
  const [result] = derive(REPO_ROOT, envFor(paths), [hit(":3")], AT_FIXTURE);
  assert.equal(result.outcome, "held");
  assert.match(result.reason, /spells no path/);
  assert.equal(replacementFor(result), "");
});

test("a slashless token resolves against the one file of that name in the tree", () => {
  const paths = fixture({ "go/one.go": "package one\n\nfunc One() {}\n" });
  const [result] = derive(REPO_ROOT, envFor(paths), [hit("one.go:3")], AT_FIXTURE);
  assert.equal(result.outcome, "anchor");
  assert.equal(replacementFor(result), `${FIXTURE}/go/one.go#One`);
});

test("a basename several directories carry resolves only against the document's own paths", () => {
  const env = envFor(["a/x/one.go", "a/y/one.go", "a/z/two.go"]);
  assert.deepEqual(resolveBasename(env, "two.go", new Set()), { path: "a/z/two.go" });
  assert.match(resolveBasename(env, "gone.go", new Set()).reason, /holds no file named gone\.go/);
  assert.match(resolveBasename(env, "one.go", new Set()).reason, /names none of them/);
  assert.deepEqual(resolveBasename(env, "one.go", new Set(["a/y/one.go"])), { path: "a/y/one.go" });
  const both = new Set(["a/x/one.go", "a/y/one.go"]);
  assert.match(resolveBasename(env, "one.go", both).reason, /names several/);
});

test("a slashless token the document cannot place is held, and never guessed", () => {
  const paths = fixture({
    "go/one.go": "package one\n\nfunc One() {}\n",
    "go2/one.go": "package one\n",
  });
  const [result] = derive(REPO_ROOT, envFor(paths), [hit("one.go:3")], AT_FIXTURE);
  assert.equal(result.outcome, "held");
  assert.match(result.reason, /2 files named one\.go/);

  const named = new Set([`${FIXTURE}/go/one.go`]);
  const placed = derive(
    REPO_ROOT,
    envFor(paths),
    [hit("one.go:3", { namedInDocument: named })],
    AT_FIXTURE,
  );
  assert.equal(placed[0].outcome, "anchor");
  assert.equal(replacementFor(placed[0]), `${FIXTURE}/go/one.go#One`);
});

test("a document places a basename by the full paths it writes, and the tree confirms each", () => {
  const env = envFor(["cmd/web/cold.go", "internal/queue/cold.go"]);
  const prose = "Read `cmd/web/cold.go` and `docs/gone/none.go`, then `cold.go:12`.";
  assert.deepEqual([...pathsNamedIn(prose, env)], ["cmd/web/cold.go"]);
  assert.deepEqual([...pathsNamedIn("no path here", env)], []);
  assert.deepEqual([...pathsNamedIn("anything", null)], []);
  // An ADR cross-links relatively, so the commonest spelling resolves against the citing document.
  const rel = "See [web](../../cmd/web/cold.go) and [queue](./cold.go).";
  assert.deepEqual([...pathsNamedIn(rel, env, "internal/queue/notes.md")], [
    "cmd/web/cold.go",
    "internal/queue/cold.go",
  ]);
  assert.deepEqual([...pathsNamedIn(rel, env)], []);
});

// #2160: the tree answers every basename, so the document's own foreign paths are read first.

test("#2160 a basename the document spells against another tree is held, never repointed", () => {
  const env = envFor(["a/x/two.go"]);
  assert.deepEqual(resolveBasename(env, "two.go", new Set()), { path: "a/x/two.go" });
  assert.match(
    resolveBasename(env, "two.go", new Set(), new Set(["two.go"])).reason,
    /spells two\.go against a path outside this tree/,
  );
  // A basename the tree never held is answered by the tree, so the reason names the real cause.
  assert.match(
    resolveBasename(env, "gone.go", new Set(), new Set(["gone.go"])).reason,
    /holds no file named gone\.go/,
  );
  // The document's own full paths cannot outvote the foreign signal either.
  const both = envFor(["a/x/one.go", "a/y/one.go"]);
  const named = new Set(["a/y/one.go"]);
  assert.deepEqual(resolveBasename(both, "one.go", named), { path: "a/y/one.go" });
  assert.match(
    resolveBasename(both, "one.go", named, new Set(["one.go"])).reason,
    /outside this tree/,
  );
});

test("#2160 derive holds a slashless token the document places outside this tree", () => {
  const paths = fixture({ "go/one.go": "package one\n\nfunc One() {}\n" });
  const env = envFor(paths);
  const [converted] = derive(REPO_ROOT, env, [hit("one.go:3")], AT_FIXTURE);
  assert.equal(converted.outcome, "anchor");

  const flagged = { ...hit("one.go:3"), foreignInDocument: new Set(["one.go"]) };
  const [result] = derive(REPO_ROOT, env, [flagged], AT_FIXTURE);
  assert.equal(result.outcome, "held");
  assert.match(result.reason, /outside this tree/);
});

test("#2160 a foreign path denies its basename, and a shorthand of the document's own path does not", () => {
  const env = envFor(["proj/internal/dnsx/message.go", "proj/internal/measure/tls.go"]);
  const wanted = new Set(["message.go", "tls.go"]);

  // ADR-0143's shape: the library path roots at no entry of this tree, and nothing explains it.
  const upstream = "`dns/dnsmessage/message.go` holds the table, so `message.go:493` reads it.";
  assert.deepEqual([...foreignBasenamesIn(upstream, env, "docs/adr/x.md", wanted)], ["message.go"]);

  // ADR-0152's shape: `measure/tls.go` is the tail of a path the document also writes in full.
  const named = new Set(["proj/internal/measure/tls.go"]);
  const short = "See `proj/internal/measure/tls.go`, and `measure/tls.go` at `tls.go:3`.";
  assert.deepEqual([...foreignBasenamesIn(short, env, "docs/adr/x.md", wanted)], ["tls.go"]);
  assert.deepEqual([...foreignBasenamesIn(short, env, "docs/adr/x.md", wanted, named)], []);

  // A basename no slashless token spells costs no classification at all.
  assert.deepEqual([...foreignBasenamesIn(upstream, env, "docs/adr/x.md", new Set())], []);
  assert.deepEqual([...foreignBasenamesIn(upstream, null, "docs/adr/x.md", wanted)], []);
});

test("#2160 the foreign signal survives a sentence's punctuation and reads a URL too", () => {
  const env = envFor(["proj/internal/dnsx/message.go"]);
  const wanted = new Set(["message.go"]);
  const at = (text) => [...foreignBasenamesIn(text, env, "docs/adr/x.md", wanted)];

  // TREE_PATH takes the full stop with the path, and classify trims it off its target.
  assert.deepEqual(at("The table lives in dns/dnsmessage/message.go. See `message.go:493`."), [
    "message.go",
  ]);
  assert.deepEqual(at("Compare (dns/dnsmessage/message.go), then `message.go:493`."), ["message.go"]);

  // classify calls a schemeless URL `ignored`, and it names another tree as plainly as a root does.
  assert.deepEqual(at("https://github.com/golang/net/blob/main/dns/dnsmessage/message.go"), [
    "message.go",
  ]);
});

test("#2160 a path inside a fence or an HTML comment denies nothing", () => {
  const env = envFor(["proj/internal/dnsx/message.go"]);
  const wanted = new Set(["message.go"]);
  const denied = (markdown) =>
    [...foreignBasenamesIn(visibleText(parse(markdown)), env, "docs/adr/x.md", wanted)];

  assert.deepEqual(denied("Read `dns/dnsmessage/message.go` for the table.\n"), ["message.go"]);
  assert.deepEqual(denied("```sh\ncp dns/dnsmessage/message.go .\n```\n"), []);
  assert.deepEqual(denied("<!-- dns/dnsmessage/message.go -->\n\nProse.\n"), []);
  // A link target is live text, and the scanner reads it, so this scan reads it too.
  assert.deepEqual(denied("See [the table](dns/dnsmessage/message.go).\n"), ["message.go"]);
});

test("#2160 scanDocuments carries the foreign basenames the derivation reads", () => {
  const paths = fixture({
    "go/one.go": "package one\n\nfunc One() {}\n",
    "docs/note.md": "Upstream `vendored/pkg/one.go` is the subject, so `one.go:3` names it.\n",
  });
  const env = envFor(paths);
  const doc = `${FIXTURE}/docs/note.md`;
  const [scanned] = scanDocuments(REPO_ROOT, [doc], env).get(doc).hits;
  assert.deepEqual([...scanned.foreignInDocument], ["one.go"]);
  const [result] = derive(REPO_ROOT, env, [scanned], AT_FIXTURE);
  assert.equal(result.outcome, "held");
});

// The identifier pattern cuts `fa97` out of `860fa97`, and F12 counts that site (#2257).
test("a bare hex commit hash spells no candidate name", () => {
  const env = envFor(["cmd/web/signals.go"]);
  const line = "`860fa97`: the commit that moved `x.go:1`";
  assert.equal(rivalName(line, "x.go:1", ["fa97"], "Alpha", env).verdict, "unproven");
});

test("a hash of every git width spells no candidate name", () => {
  const env = envFor(["cmd/web/signals.go"]);
  const full = "da39a3ee5e6b4b0d3255bfef95601890afd80709";
  for (const hash of ["860fa97", "860fa9712c", full, "deadb33f"]) {
    const line = `\`${hash}\` moved \`x.go:1\``;
    const names = [...hash.matchAll(/[A-Za-z_][A-Za-z0-9_.]*/g)].map(([m]) => m);
    assert.equal(names.length > 0, true, `${hash} yields no identifier to suppress`);
    assert.equal(rivalName(line, "x.go:1", names, "Alpha", env).verdict, "unproven", hash);
  }
});

// A word that is hex by accident is still prose, and a name is still a name (#2257).
test("a hex-shaped word and a real name are untouched by the hash guard", () => {
  const env = envFor(["cmd/web/signals.go"]);
  const bad = rivalName("`defaced` moved `x.go:1`", "x.go:1", ["defaced"], "Alpha", env);
  assert.equal(bad.verdict, "suspect");
  assert.equal(bad.rival, "defaced");
  const good = rivalName("`Beta` moved `x.go:1`", "x.go:1", ["Beta"], "Alpha", env);
  assert.equal(good.verdict, "suspect");
  assert.equal(good.rival, "Beta");
});

// `sameName` serves both arms, so a prefix arm moves both at once (ADR-2277 §3).
test("a field access on the anchor's own type corroborates nothing", () => {
  const env = envFor(["cmd/web/signals.go"]);
  const field = "`signalRow.seenAge` is stale (`x.go:1`)";
  const own = rivalName(field, "x.go:1", ["signalRow"], "signalRow", env);
  assert.equal(own.verdict, "unproven");
  const other = rivalName(field, "x.go:1", ["signalRow", "Other"], "signalRow", env);
  assert.equal(other.verdict, "unproven");
  assert.equal(other.rival, null);
  const tail = rivalName(field, "x.go:1", ["signalRow", "seenAge"], "signalRow", env);
  assert.equal(tail.verdict, "suspect");
  assert.equal(tail.rival, "seenAge");
  const qualified = "`web.signalRow` is it (`x.go:1`)";
  assert.equal(rivalName(qualified, "x.go:1", ["signalRow"], "signalRow", env).verdict, "corroborated");
});

// ADR-2155 refuses to repair this inversion, because §9 fact F12 finds nothing true behind it.
test("a rival the target no longer declares stays unproven", () => {
  const env = envFor(["cmd/web/deltas.go"]);
  const line = "`certExpiryWindow` bounds it (`x.go:1`)";
  const withdrawn = rivalName(line, "x.go:1", ["Alpha"], "Alpha", env);
  assert.equal(withdrawn.verdict, "unproven");
  assert.equal(withdrawn.rival, null);
  const present = rivalName(line, "x.go:1", ["Alpha", "certExpiryWindow"], "Alpha", env);
  assert.equal(present.verdict, "suspect");
  assert.equal(present.rival, "certExpiryWindow");
  assert.equal(present.position, "before");
});
