// Arm B: an anchor resolves against its row (SPEC docs/spec/citation-anchors.md §3.2, §7.1, #1970).
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { writeFileSync, rmSync, mkdtempSync, symlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { extractCitations } from "./citations/extract.mjs";
import { classify } from "./citations/classify.mjs";
import { armB, formatBroken, formatFatal } from "./citations/armb.mjs";
import { goInventory } from "./citations/rows/go.mjs";
import { rowFor } from "./citations/rows.mjs";
import { SQLC_ROW } from "./citations/rows/sqlc.mjs";
import { TEMPLATE_ROW } from "./citations/rows/template.mjs";
import { MARKDOWN_ROW } from "./citations/rows/markdown.mjs";
import { CONTAINMENT_ROW } from "./citations/rows/containment.mjs";
import { markerInventory } from "./citations/rows/marker.mjs";
import { readdirSync, readFileSync } from "node:fs";
import { environment } from "./check-citations.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = join(SCRIPT_DIR, "..", "..");
const ENV = environment(REPO_ROOT);

const DOC = "docs/adr/0001-stack-and-runtime.md";
const GO_TARGET = "internal/commentlint/surface/golang.go";
const GROUPED = "internal/queue/membership.go";

function anchorsOf(markdown, docFile = DOC) {
  const results = classify(ENV, docFile, extractCitations(markdown)).map((r) => ({
    ...r,
    file: docFile,
  }));
  return armB(REPO_ROOT, results);
}

const key = (r) => `${r.value}#${r.anchor}`;

function brokenAnchors(markdown, docFile = DOC) {
  return anchorsOf(markdown, docFile).broken.map(key);
}

function verifiedAnchors(markdown, docFile = DOC) {
  return anchorsOf(markdown, docFile).verified.map(key);
}

function stubbed(markdown, row) {
  const results = classify(ENV, DOC, extractCitations(markdown)).map((r) => ({ ...r, file: DOC }));
  return armB(REPO_ROOT, results, [row]);
}

function cliStatus(args, env = process.env) {
  try {
    execFileSync(process.execPath, [join(SCRIPT_DIR, "check-citations.mjs"), ...args], {
      encoding: "utf8",
      stdio: "pipe",
      env,
    });
    return 0;
  } catch (err) {
    return err.status;
  }
}

function cliOutput(args) {
  try {
    return execFileSync(process.execPath, [join(SCRIPT_DIR, "check-citations.mjs"), ...args], {
      encoding: "utf8",
      stdio: "pipe",
    });
  } catch (err) {
    return err.stdout ?? "";
  }
}

test("a Go anchor naming a top-level declaration passes", () => {
  assert.deepEqual(verifiedAnchors(`The lexer is \`${GO_TARGET}#goDocSpans\`.`), [
    `${GO_TARGET}#goDocSpans`,
  ]);
});

test("a Go anchor naming no declaration is broken", () => {
  const token = `${GO_TARGET}#noSuchDeclaration`;
  const { broken } = anchorsOf(`The lexer is \`${token}\`.`);
  assert.deepEqual(broken.map(key), [token]);
  assert.match(formatBroken(broken[0]), /no such anchor: .*noSuchDeclaration/);
});

test("a declaration inside a parenthesised const group passes", () => {
  // A column-0 regex reds 9.57% of the cited Go surface on this shape (SPEC §7.3).
  assert.deepEqual(verifiedAnchors(`The kind is \`${GROUPED}#subjectKindEndpoint\`.`), [
    `${GROUPED}#subjectKindEndpoint`,
  ]);
});

test("a name that occurs only inside a raw-string literal is broken", () => {
  // cmd/godecls/main_test.go embeds `func fabricated` as a test fixture, at column 0.
  const token = "cmd/godecls/main_test.go#fabricated";
  assert.deepEqual(brokenAnchors(`The fixture declares \`${token}\`.`), [token]);
});

test("a method anchor is spelled Receiver.Method, and the bare method name is broken", () => {
  assert.deepEqual(verifiedAnchors(`The lexer is \`${GO_TARGET}#Go.Lex\`.`), [
    `${GO_TARGET}#Go.Lex`,
  ]);
  assert.deepEqual(brokenAnchors(`The lexer is \`${GO_TARGET}#Lex\`.`), [`${GO_TARGET}#Lex`]);
});

test("a star or a parenthesis spells no anchor, so no anchor claim reaches Arm B", () => {
  // SPEC §3.3 rule 4 bounds the anchor to the extractor's own character class (#1968).
  for (const spelling of ["(*Go).Lex", "Go.Lex()", "*Go.Lex"]) {
    const token = `${GO_TARGET}#${spelling}`;
    assert.deepEqual(anchorsOf(`The lexer is \`${token}\`.`).anchored, [], spelling);
    assert.deepEqual(anchorsOf(`The lexer is [it](${token}).`).anchored, [], spelling);
  }
});

test("a Go citation carrying no anchor passes, because the gate asserts correctness", () => {
  // The checker asserts correctness, never presence (SPEC §7.2).
  const { anchored, broken } = anchorsOf(`The lexer is \`${GO_TARGET}\`.`);
  assert.deepEqual(anchored, []);
  assert.deepEqual(broken, []);
});

test("a link fragment carries the same anchor vocabulary as a code span", () => {
  // Both spellings bind, for every row (SPEC §3.5).
  assert.deepEqual(verifiedAnchors(`See [the lexer](../../${GO_TARGET}#Go.Lex).`), [
    `../../${GO_TARGET}#Go.Lex`,
  ]);
  assert.deepEqual(brokenAnchors(`See [the lexer](../../${GO_TARGET}#gone).`), [
    `../../${GO_TARGET}#gone`,
  ]);
});

test("an anchor on a path this gate does not resolve is passed over, and listed", () => {
  // Arm B judges an anchor only where the path resolves (SPEC §7.6).
  const markdown = [
    "The tree is `apache/kafka/core/src/Main.go#Run`.",
    "",
    "It is `docs-site/node_modules/astro/dist/core.go#Run`.",
    "",
    "So `internal/queue/gone-forever.go#Run` names nothing.",
  ].join("\n");
  const { anchored, unresolved, broken } = anchorsOf(markdown);
  assert.equal(anchored.length, 3);
  assert.deepEqual(broken, []);
  assert.deepEqual(unresolved.map((r) => r.status).sort(), ["dead", "foreign", "untracked"]);
});

test("an anchor on a withdrawn path is passed over, and --verbose names the bucket", () => {
  // Every bucket the gate passes over is listable, so no suppression is silent (SPEC §7.6).
  const fixture = join(SCRIPT_DIR, "citations", `withdrawn-${process.pid}.md`);
  const token = "internal/queue/gone-forever.go#Run";
  try {
    writeFileSync(fixture, `The file \`${token}\` no longer exists.\n`);
    assert.deepEqual(anchorsOf(`The file \`${token}\` no longer exists.`).unresolved.length, 1);
    assert.equal(cliStatus([fixture]), 0);
    const out = cliOutput(["--verbose", fixture]);
    assert.match(out, /Anchored, and this gate does not resolve the path:/);
    assert.match(out, /#Run on a withdrawn path/);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("a target kind with no declaring row falls to containment", () => {
  // A target that declares no name MAY carry a containment anchor (SPEC §3.2 rule 2).
  assert.equal(rowFor("db/migrations/00100_init.sql").name, "containment");
  assert.deepEqual(verifiedAnchors("The scripts are `docs-site/package.json#scripts`."), [
    "docs-site/package.json#scripts",
  ]);
});

test("a row whose inventory reports an unparseable target is fatal, never a violation", () => {
  // An unparseable target is operator error, and it takes exit 2 (SPEC §7.7).
  const { fatal, broken, verified } = stubbed(`It is \`${GO_TARGET}#Go.Lex\`.`, {
    name: "stub",
    matches: (path) => path.endsWith(".go"),
    inventory: (_root, paths) => new Map(paths.map((p) => [p, { error: "parse: 1:1: bad" }])),
  });
  assert.deepEqual(broken, []);
  assert.deepEqual(verified, []);
  assert.equal(fatal.length, 1);
  assert.match(formatFatal(fatal[0]), /cannot judge .*parse: 1:1: bad/);
});

test("a row whose inventory cannot run at all is fatal, never a violation", () => {
  const { fatal, broken, verified } = stubbed(`It is \`${GO_TARGET}#Go.Lex\`.`, {
    name: "stub",
    matches: (path) => path.endsWith(".go"),
    inventory: () => {
      throw new Error("spawn go ENOENT");
    },
  });
  assert.deepEqual(broken, []);
  // A count by subtraction would report this anchor as one the gate resolved.
  assert.deepEqual(verified, []);
  assert.deepEqual(
    fatal.map((f) => f.row),
    ["stub"],
  );
  assert.match(formatFatal(fatal[0]), /the stub row could not run \(spawn go ENOENT\)/);
});

test("the Go inventory reports an unparseable file rather than throwing", () => {
  const rel = `docs-site/scripts/citations/broken-${process.pid}.go.txt`;
  const fixture = join(REPO_ROOT, rel);
  try {
    writeFileSync(fixture, "package x\n\nfunc (\n");
    assert.ok(goInventory(REPO_ROOT, [rel]).get(rel).error);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("the Go inventory reads no file outside the repository", () => {
  // os.Root refuses a path that leaves the root, so no citation can name /etc/passwd.
  for (const escape of ["../outside.go", "/etc/hostname"]) {
    assert.ok(goInventory(REPO_ROOT, [escape]).get(escape).error, escape);
  }
});

test("the CLI exits 1 on a broken Go anchor, and 0 on one that resolves", () => {
  const fixture = join(SCRIPT_DIR, "citations", `goanchor-${process.pid}.md`);
  try {
    writeFileSync(fixture, `The lexer is \`${GO_TARGET}#noSuchDeclaration\`.\n`);
    assert.equal(cliStatus([fixture]), 1);
    writeFileSync(fixture, `The lexer is \`${GO_TARGET}#Go.Lex\`.\n`);
    assert.equal(cliStatus([fixture]), 0);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("the CLI exits 2 when no go binary is on PATH", () => {
  // The gate shells out to `git` too, so PATH keeps git alone. An absent git is another fault.
  const shim = mkdtempSync(join(tmpdir(), "nogo-"));
  const git = execFileSync("sh", ["-c", "command -v git"], { encoding: "utf8" }).trim();
  symlinkSync(git, join(shim, "git"));
  const fixture = join(SCRIPT_DIR, "citations", `nogo-${process.pid}.md`);
  try {
    writeFileSync(fixture, `The lexer is \`${GO_TARGET}#Go.Lex\`.\n`);
    assert.equal(cliStatus([fixture], { ...process.env, PATH: shim }), 2);
  } finally {
    rmSync(fixture, { force: true });
    rmSync(shim, { force: true, recursive: true });
  }
});

const SQL_TARGET = "db/queries/vantages.sql";
const TMPL_TARGET = "design-system/templates/coverage.tmpl";
const MIGRATION = "db/migrations/00001_heartbeat.sql";

test("a sqlc anchor naming a `-- name:` query passes, and an absent one is broken", () => {
  assert.deepEqual(verifiedAnchors(`The read is \`${SQL_TARGET}#ListVantages\`.`), [
    `${SQL_TARGET}#ListVantages`,
  ]);
  const token = `${SQL_TARGET}#ListVantage`;
  const { broken } = anchorsOf(`The read is \`${token}\`.`);
  assert.deepEqual(broken.map(key), [token]);
  assert.match(formatBroken(broken[0]), /declares no `-- name:` query named ListVantage/);
});

test("a template anchor naming a {{define}} passes, and an absent one is broken", () => {
  assert.deepEqual(verifiedAnchors(`The card is \`${TMPL_TARGET}#cv-dial\`.`), [
    `${TMPL_TARGET}#cv-dial`,
  ]);
  const token = `${TMPL_TARGET}#cv-dialog`;
  const { broken } = anchorsOf(`The card is \`${token}\`.`);
  assert.deepEqual(broken.map(key), [token]);
  assert.match(formatBroken(broken[0]), /declares no `\{\{define\}\}` named cv-dialog/);
});

test("a goose migration is not judged by the sqlc row, because the key is a path predicate", () => {
  // A migration declares no names, and containment covers it instead (SPEC §3.2 rule 2).
  assert.equal(rowFor(MIGRATION).name, "containment");
  assert.deepEqual(verifiedAnchors(`The table is \`${MIGRATION}#heartbeat\`.`), [
    `${MIGRATION}#heartbeat`,
  ]);
});

test("every tracked .sql outside db/queries/ falls outside the sqlc row", () => {
  const tracked = execFileSync("git", ["ls-files", "--", "*.sql"], {
    cwd: REPO_ROOT,
    encoding: "utf8",
  })
    .split("\n")
    .filter(Boolean);
  const outside = tracked.filter((p) => !p.startsWith("db/queries/"));
  assert.ok(outside.length > 0);
  assert.deepEqual(
    outside.filter((p) => rowFor(p)?.name === "sqlc"),
    [],
  );
  assert.ok(tracked.some((p) => rowFor(p)?.name === "sqlc"));
});

test("a sqlc or template citation carrying no anchor passes", () => {
  // The checker asserts correctness, never presence (SPEC §7.2).
  for (const target of [SQL_TARGET, TMPL_TARGET]) {
    const { anchored, broken } = anchorsOf(`The file is \`${target}\`.`);
    assert.deepEqual(anchored, [], target);
    assert.deepEqual(broken, [], target);
  }
});

test("a link fragment carries the sqlc and template vocabularies too", () => {
  // The rule binds both spellings, for every row (SPEC §3.5).
  assert.deepEqual(verifiedAnchors(`See [the read](../../${SQL_TARGET}#GetVantage).`), [
    `../../${SQL_TARGET}#GetVantage`,
  ]);
  assert.deepEqual(brokenAnchors(`See [the card](../../${TMPL_TARGET}#gone).`), [
    `../../${TMPL_TARGET}#gone`,
  ]);
});

test("neither row shells out to a parser, so an empty PATH resolves both", () => {
  // A column-0 marker is the real declaration syntax for both rows (SPEC §7.3).
  const shim = mkdtempSync(join(tmpdir(), "norun-"));
  const restore = process.env.PATH;
  try {
    process.env.PATH = shim;
    const queries = SQLC_ROW.inventory(REPO_ROOT, [SQL_TARGET]).get(SQL_TARGET);
    const defines = TEMPLATE_ROW.inventory(REPO_ROOT, [TMPL_TARGET]).get(TMPL_TARGET);
    assert.ok(queries.names.has("GetVantage"));
    assert.ok(defines.names.has("coverage"));
  } finally {
    process.env.PATH = restore;
    rmSync(shim, { force: true, recursive: true });
  }
});

test("an indented marker is a declaration, and each row reads it", () => {
  // A missed marker reds a correct citation, which is the fault SPEC §7.3 rejects.
  const cases = [
    [
      SQLC_ROW,
      `db/queries/indented-${process.pid}.sql`,
      "  -- name: IndentedQuery :one\n",
      "IndentedQuery",
    ],
    [
      TEMPLATE_ROW,
      `design-system/templates/indented-${process.pid}.tmpl`,
      '  {{- define "indented"}}\n',
      "indented",
    ],
  ];
  for (const [row, rel, source, name] of cases) {
    const fixture = join(REPO_ROOT, rel);
    try {
      writeFileSync(fixture, source);
      assert.ok(row.inventory(REPO_ROOT, [rel]).get(rel).names.has(name), rel);
    } finally {
      rmSync(fixture, { force: true });
    }
  }
});

test("a marker inventory reports an unreadable target rather than throwing", () => {
  // An unreadable target is operator error, and it takes exit 2 (SPEC §7.7).
  for (const escape of ["../outside.sql", "/etc/hostname", "db/queries/no-such-file.sql"]) {
    assert.ok(SQLC_ROW.inventory(REPO_ROOT, [escape]).get(escape).error, escape);
  }
});

test("the CLI exits 1 on a broken template anchor, and 0 on one that resolves", () => {
  const fixture = join(SCRIPT_DIR, "citations", `tmplanchor-${process.pid}.md`);
  try {
    writeFileSync(fixture, `The card is \`${TMPL_TARGET}#noSuchDefine\`.\n`);
    assert.equal(cliStatus([fixture]), 1);
    writeFileSync(fixture, `The card is \`${TMPL_TARGET}#coverage\`.\n`);
    assert.equal(cliStatus([fixture]), 0);
  } finally {
    rmSync(fixture, { force: true });
  }
});

const MD_TARGET = "docs/spec/citation-anchors.md";

// The row reads a file, so a fixture proves a heading shape the tree does not happen to hold.
function inventoryOf(markdown) {
  const rel = `docs-site/scripts/citations/heading-${process.pid}.md`;
  const fixture = join(REPO_ROOT, rel);
  try {
    writeFileSync(fixture, markdown);
    return MARKDOWN_ROW.inventory(REPO_ROOT, [rel]).get(rel);
  } finally {
    rmSync(fixture, { force: true });
  }
}

test("a Markdown anchor naming a real heading slug passes, and an absent one is broken", () => {
  assert.deepEqual(verifiedAnchors(`The form is \`${MD_TARGET}#3-the-form\`.`), [
    `${MD_TARGET}#3-the-form`,
  ]);
  const token = `${MD_TARGET}#3-the-shape`;
  const { broken } = anchorsOf(`The form is \`${token}\`.`);
  assert.deepEqual(broken.map(key), [token]);
  assert.match(formatBroken(broken[0]), /declares no `github-slugger` heading slug named/);
});

test("every heading level is citable, H1 to H6", () => {
  // A numbered heading earns no privilege (SPEC §4 rule 1).
  const levels = [1, 2, 3, 4, 5, 6].map((n) => `${"#".repeat(n)} Level ${n}\n`).join("\n");
  const { ids } = { ids: inventoryOf(levels).names };
  for (const n of [1, 2, 3, 4, 5, 6]) assert.ok(ids.has(`level-${n}`), `H${n}`);
});

test("the slug is derived from the inline-markup-stripped label", () => {
  // Stripping matches what both renderers slug (SPEC §4 rule 6).
  const entry = inventoryOf(
    [
      "## The `citations` gate",
      "",
      "## A **bold** claim",
      "",
      "## An *italic* claim",
      "",
      "## See [the SPEC](docs/spec/citation-anchors.md)",
      "",
    ].join("\n"),
  );
  for (const slug of ["the-citations-gate", "a-bold-claim", "an-italic-claim", "see-the-spec"]) {
    assert.ok(entry.names.has(slug), slug);
  }
  // The same rule holds on a tracked file, so the row and the fixture agree.
  assert.deepEqual(
    verifiedAnchors(`The field is \`${MD_TARGET}#6-the-adr-decision-proposal-blocks-site-field\`.`),
    [`${MD_TARGET}#6-the-adr-decision-proposal-blocks-site-field`],
  );
});

test("a de-duplicated slug is refused, and the reason names the drift", () => {
  // The suffix is positional, so a copy inserted above re-points it in silence (SPEC §4 rule 4).
  const entry = inventoryOf("## Consequences\n\n## Consequences\n");
  assert.deepEqual([...entry.names].sort(), ["consequences", "consequences-1"]);
  assert.match(entry.refused.get("consequences-1"), /a de-duplicated slug/);
  assert.match(entry.refused.get("consequences-1"), /re-points the anchor in silence/);
  // SPEC §4 rule 4 refuses the suffix. The unsuffixed first copy is outside the rule as written.
  assert.equal(entry.refused.has("consequences"), false);
});

test("a refused spelling reaches the broken bucket carrying the row's own reason", () => {
  // A row may hold a name and still refuse it, so the reason replaces the absence message.
  const { broken, verified } = stubbed(`It is \`${MD_TARGET}#consequences-1\`.`, {
    name: "stub",
    vocabulary: "heading slug",
    matches: (path) => path.endsWith(".md"),
    inventory: (_root, paths) =>
      new Map(
        paths.map((p) => [
          p,
          {
            names: new Set(["consequences-1"]),
            refused: new Map([["consequences-1", "a de-duplicated slug: it drifts in silence"]]),
          },
        ]),
      ),
  });
  assert.deepEqual(verified, []);
  assert.equal(broken.length, 1);
  assert.match(formatBroken(broken[0]), /a de-duplicated slug: it drifts in silence/);
});

test("a heading inside a fence is not a heading, so it slugs no anchor", () => {
  const entry = inventoryOf(["```sh", "# not a heading", "```", "", "## A real heading", ""].join("\n"));
  assert.deepEqual([...entry.names], ["a-real-heading"]);
});

test("a target carrying no heading takes a bare path, and a citation with no anchor passes", () => {
  // A citation never re-heads its target (SPEC §4 rule 3).
  assert.equal(inventoryOf("Body text only, and no heading at all.\n").names.size, 0);
  const { anchored, broken } = anchorsOf(`The SPEC is \`${MD_TARGET}\`.`);
  assert.deepEqual(anchored, []);
  assert.deepEqual(broken, []);
});

test("a link fragment carries the Markdown vocabulary too", () => {
  // The rule binds both spellings, for every row (SPEC §3.5).
  assert.deepEqual(verifiedAnchors(`See [the form](../../${MD_TARGET}#3-the-form).`), [
    `../../${MD_TARGET}#3-the-form`,
  ]);
  assert.deepEqual(brokenAnchors(`See [the form](../../${MD_TARGET}#gone).`), [
    `../../${MD_TARGET}#gone`,
  ]);
});

test("one heading inventory feeds both checks, so no script builds a second", () => {
  // Do not add a fourth copy: two renderers already slug this way (SPEC §4 rule 6).
  const builders = readdirSync(SCRIPT_DIR, { recursive: true })
    .map(String)
    .filter((f) => f.endsWith(".mjs") && !f.endsWith(".test.mjs"))
    .filter((f) => readFileSync(join(SCRIPT_DIR, f), "utf8").includes("new GithubSlugger"));
  assert.deepEqual(builders, ["headings.mjs"]);
});

test("the CLI exits 1 on a broken Markdown anchor, and 0 on one that resolves", () => {
  const fixture = join(SCRIPT_DIR, "citations", `mdanchor-${process.pid}.md`);
  try {
    writeFileSync(fixture, `The form is \`${MD_TARGET}#no-such-heading\`.\n`);
    assert.equal(cliStatus([fixture]), 1);
    writeFileSync(fixture, `The form is \`${MD_TARGET}#3-the-form\`.\n`);
    assert.equal(cliStatus([fixture]), 0);
  } finally {
    rmSync(fixture, { force: true });
  }
});

// Containment and the disambiguating snippet (SPEC §3.2 rules 2 and 3, §3.4, #1973).

test("a containment anchor passes when the target holds the token, and fails when it does not", () => {
  assert.deepEqual(verifiedAnchors(`The table is \`${MIGRATION}#heartbeat\`.`), [
    `${MIGRATION}#heartbeat`,
  ]);
  const token = `${MIGRATION}#no_such_token`;
  const { broken } = anchorsOf(`The table is \`${token}\`.`);
  assert.deepEqual(broken.map(key), [token]);
  assert.match(formatBroken(broken[0]), /declares no token named no_such_token/);
});

test("a containment anchor that occurs more than once passes", () => {
  // Containment proves existence, and it need not be unique (SPEC §3.2 rule 3).
  const rel = `db/migrations/99999_repeat-${process.pid}.sql`;
  const fixture = join(REPO_ROOT, rel);
  try {
    writeFileSync(fixture, "-- +goose Up\nCREATE TABLE widget (id int);\nDROP TABLE widget;\n");
    const entry = CONTAINMENT_ROW.inventory(REPO_ROOT, [rel]).get(rel);
    assert.equal(entry.holds("widget"), true);
    assert.equal(entry.holds("gadget"), false);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("a containment anchor matches a token, never a substring of a longer one", () => {
  // A rename to `heartbeat_v2` is the drift this row exists to catch (SPEC §3.2 rule 3).
  const rel = `db/migrations/99998_rename-${process.pid}.sql`;
  const fixture = join(REPO_ROOT, rel);
  try {
    writeFileSync(fixture, "-- +goose Up\nALTER TABLE heartbeat_v2 ADD COLUMN identity int;\n");
    const entry = CONTAINMENT_ROW.inventory(REPO_ROOT, [rel]).get(rel);
    assert.equal(entry.holds("heartbeat_v2"), true);
    assert.equal(entry.holds("heartbeat"), false);
    assert.equal(entry.holds("id"), false);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("containment never rescues a target that has a declaring row", () => {
  // `spans` is a local golang.go holds on many lines, and it declares nothing (SPEC §3.2 rule 4).
  assert.deepEqual(brokenAnchors(`The lexer is \`${GO_TARGET}#spans\`.`), [`${GO_TARGET}#spans`]);
  for (const target of [GO_TARGET, SQL_TARGET, TMPL_TARGET, MD_TARGET]) {
    assert.notEqual(rowFor(target).name, "containment", target);
  }
});

test("a citation on a no-vocabulary target carrying no anchor passes", () => {
  // The checker asserts correctness, never presence (SPEC §7.2).
  const { anchored, broken } = anchorsOf(`The migration is \`${MIGRATION}\`.`);
  assert.deepEqual(anchored, []);
  assert.deepEqual(broken, []);
});

test("an anchor span followed by a code span matches one line inside the declaration", () => {
  const token = `${GO_TARGET}#goDocSpans`;
  assert.deepEqual(verifiedAnchors(`The parse is \`${token}\` \`fset := token.NewFileSet()\`.`), [
    token,
  ]);
});

test("a snippet that matches no line inside the declaration is broken", () => {
  const token = `${GO_TARGET}#goDocSpans`;
  const { broken } = anchorsOf(`The parse is \`${token}\` \`no such line anywhere\`.`);
  assert.deepEqual(broken.map(key), [token]);
  assert.match(formatBroken(broken[0]), /no such snippet: no line inside goDocSpans/);
});

test("a snippet that sits in the file but outside the declaration is broken", () => {
  // The snippet sits inside the declaration the anchor names, not merely in the file.
  const token = `${GO_TARGET}#goDocSpans`;
  assert.deepEqual(brokenAnchors(`The parse is \`${token}\` \`packageDoc bool\`.`), [token]);
});

test("a snippet that differs only by whitespace passes", () => {
  const token = `${GO_TARGET}#goDocSpans`;
  assert.deepEqual(verifiedAnchors(`The parse is \`${token}\` \`fset  :=   token.NewFileSet()\`.`), [
    token,
  ]);
});

test("a bare path followed by a code span is not read as a snippet", () => {
  // The pairing keys on the `#`, and a bare path beside a code span is unrelated (SPEC §3.4).
  const { anchored, broken } = anchorsOf(`The lexer is \`${GO_TARGET}\` \`no such line\`.`);
  assert.deepEqual(anchored, []);
  assert.deepEqual(broken, []);
});

test("only whitespace may separate the anchor span from its snippet", () => {
  // Two anchored citations in one sentence must not read each other as snippets.
  const markdown = `It is \`${GO_TARGET}#Go.Lex\` and \`${GO_TARGET}#goDocSpans\`.`;
  assert.deepEqual(verifiedAnchors(markdown).sort(), [
    `${GO_TARGET}#Go.Lex`,
    `${GO_TARGET}#goDocSpans`,
  ]);
});

test("a second citation is never read as the first one's snippet", () => {
  // A path span disambiguates nothing, and reading one as a snippet is a false red.
  for (const second of [`${GO_TARGET}#Go.Lex`, SQL_TARGET, `${MIGRATION}`]) {
    const markdown = `Two legs: \`${GO_TARGET}#goDocSpans\` \`${second}\`.`;
    assert.deepEqual(anchorsOf(markdown).broken, [], second);
    assert.ok(verifiedAnchors(markdown).includes(`${GO_TARGET}#goDocSpans`), second);
  }
});

test("a soft line break between two citations is not a snippet pairing", () => {
  // remark makes the break a whitespace text node, so adjacency alone cannot settle it.
  const markdown = `Two legs: \`${GO_TARGET}#goDocSpans\`\n\`${GO_TARGET}#Go.Lex\`.`;
  assert.deepEqual(anchorsOf(markdown).broken, []);
});

test("a Go span covers the declaration's doc comment", () => {
  // A doc comment reads as part of the declaration, and a false red is the fatal direction.
  const rel = `docs-site/scripts/citations/doccomment-${process.pid}.go.txt`;
  const fixture = join(REPO_ROOT, rel);
  const source = [
    "package x",
    "",
    "// Wide exists because ADR-0001 says so.",
    "func Wide() {}",
    "",
    "// Narrow exists too.",
    "var Narrow = 1",
    "",
    "const (",
    "\t// Grouped carries its own doc.",
    "\tGrouped = 2",
    ")",
    "",
  ].join("\n");
  try {
    writeFileSync(fixture, source);
    const spans = goInventory(REPO_ROOT, [rel]).get(rel).spans;
    assert.deepEqual(spans.get("Wide"), [[3, 4]]);
    assert.deepEqual(spans.get("Narrow"), [[6, 7]]);
    assert.deepEqual(spans.get("Grouped"), [[10, 11]]);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("a marker row reads every marker even when the caller passes a global pattern", () => {
  // A sticky or global pattern carries lastIndex between lines, and drops every other marker.
  const rel = `db/queries/global-${process.pid}.sql`;
  const fixture = join(REPO_ROOT, rel);
  try {
    writeFileSync(fixture, "-- name: One :one\nSELECT 1;\n-- name: Two :one\nSELECT 2;\n");
    const row = {
      ...SQLC_ROW,
      inventory: (root, paths) => markerInventory(root, paths, /^[ \t]*-- name: (\S+)/gm),
    };
    assert.deepEqual([...row.inventory(REPO_ROOT, [rel]).get(rel).names], ["One", "Two"]);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("the snippet rule is kind-blind, so it reaches a Markdown anchor too", () => {
  const token = `${MD_TARGET}#34-disambiguation`;
  assert.deepEqual(verifiedAnchors(`The rule is \`${token}\` \`The rule is kind-blind\`.`), [token]);
  assert.deepEqual(brokenAnchors(`The rule is \`${token}\` \`no such line at all\`.`), [token]);
});

test("a snippet on a containment anchor matches any line in the file", () => {
  // A containment anchor proves existence only, so its region is the whole file (§3.2 rule 3).
  const token = `${MIGRATION}#heartbeat`;
  assert.deepEqual(verifiedAnchors(`The table is \`${token}\` \`+goose Up\`.`), [token]);
  assert.deepEqual(brokenAnchors(`The table is \`${token}\` \`no such line at all\`.`), [token]);
});

test("a link fragment carries the snippet rule too", () => {
  // The rule binds both spellings, for every row (SPEC §3.5).
  const token = `../../${GO_TARGET}#goDocSpans`;
  assert.deepEqual(verifiedAnchors(`See [it](${token}) \`fset := token.NewFileSet()\`.`), [token]);
  assert.deepEqual(brokenAnchors(`See [it](${token}) \`no such line at all\`.`), [token]);
});

test("a row that reports no region for a verified anchor is fatal, never a violation", () => {
  // A snippet this gate should have judged and could not takes exit 2 (SPEC §7.7).
  const { fatal, broken, verified } = stubbed(`It is \`${GO_TARGET}#Go.Lex\` \`some line\`.`, {
    name: "stub",
    vocabulary: "name",
    matches: (path) => path.endsWith(".go"),
    inventory: (_root, paths) => new Map(paths.map((p) => [p, { names: new Set(["Go.Lex"]) }])),
  });
  assert.deepEqual(broken, []);
  assert.deepEqual(verified, []);
  assert.match(formatFatal(fatal[0]), /the row reports no region for this anchor/);
});

test("the CLI exits 1 on a broken containment anchor, and 0 on one that resolves", () => {
  const fixture = join(SCRIPT_DIR, "citations", `contain-${process.pid}.md`);
  try {
    writeFileSync(fixture, `The table is \`${MIGRATION}#no_such_token\`.\n`);
    assert.equal(cliStatus([fixture]), 1);
    writeFileSync(fixture, `The table is \`${MIGRATION}#heartbeat\`.\n`);
    assert.equal(cliStatus([fixture]), 0);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("the CLI exits 1 on a broken snippet, and 0 on one that resolves", () => {
  const fixture = join(SCRIPT_DIR, "citations", `snippet-${process.pid}.md`);
  const token = `${GO_TARGET}#goDocSpans`;
  try {
    writeFileSync(fixture, `The parse is \`${token}\` \`no such line at all\`.\n`);
    assert.equal(cliStatus([fixture]), 1);
    writeFileSync(fixture, `The parse is \`${token}\` \`fset := token.NewFileSet()\`.\n`);
    assert.equal(cliStatus([fixture]), 0);
  } finally {
    rmSync(fixture, { force: true });
  }
});
