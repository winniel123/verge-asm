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

test("a target kind with no row is passed over, and neither passes nor fails", () => {
  // The table is open, and a missing row never blocks a citation (SPEC §3.2 rule 1).
  assert.equal(rowFor("db/migrations/00100_init.sql"), null);
  const { noRow, broken } = anchorsOf("The SPEC is `docs/spec/v1-spec.md#no-such-heading`.");
  assert.deepEqual(broken, []);
  assert.deepEqual(noRow.map(key), ["docs/spec/v1-spec.md#no-such-heading"]);
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
  assert.equal(rowFor(MIGRATION), null);
  const { broken, noRow } = anchorsOf(`The table is \`${MIGRATION}#heartbeat\`.`);
  assert.deepEqual(broken, []);
  assert.deepEqual(noRow.map(key), [`${MIGRATION}#heartbeat`]);
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
