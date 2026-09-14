// Arm B: an anchor resolves against its row (SPEC docs/spec/citation-anchors.md §3.2, §7.1, #1970).
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { writeFileSync, rmSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { delimiter, dirname, join } from "node:path";
import { extractCitations } from "./citations/extract.mjs";
import { classify } from "./citations/classify.mjs";
import { armB, formatBroken, formatFatal } from "./citations/armb.mjs";
import { goInventory } from "./citations/rows/go.mjs";
import { rowFor } from "./citations/rows.mjs";
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

// An anchor no bucket holds is one Arm B resolved against its row.
function verifiedAnchors(markdown, docFile = DOC) {
  const { anchored, broken, noRow, unresolved } = anchorsOf(markdown, docFile);
  const held = new Set([...broken, ...noRow, ...unresolved].map(key));
  return anchored.map(key).filter((k) => !held.has(k));
}

function stubbed(markdown, row) {
  const results = classify(ENV, DOC, extractCitations(markdown)).map((r) => ({ ...r, file: DOC }));
  return armB(REPO_ROOT, results, [row]);
}

function cliStatus(args, env = process.env) {
  try {
    execFileSync("node", [join(SCRIPT_DIR, "check-citations.mjs"), ...args], {
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
    return execFileSync("node", [join(SCRIPT_DIR, "check-citations.mjs"), ...args], {
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
  const { fatal, broken } = stubbed(`It is \`${GO_TARGET}#Go.Lex\`.`, {
    name: "stub",
    matches: (path) => path.endsWith(".go"),
    inventory: (_root, paths) => new Map(paths.map((p) => [p, { error: "parse: 1:1: bad" }])),
  });
  assert.deepEqual(broken, []);
  assert.equal(fatal.length, 1);
  assert.match(formatFatal(fatal[0]), /cannot judge .*parse: 1:1: bad/);
});

test("a row whose inventory cannot run at all is fatal, never a violation", () => {
  const { fatal, broken } = stubbed(`It is \`${GO_TARGET}#Go.Lex\`.`, {
    name: "stub",
    matches: (path) => path.endsWith(".go"),
    inventory: () => {
      throw new Error("spawn go ENOENT");
    },
  });
  assert.deepEqual(broken, []);
  assert.deepEqual(
    fatal.map((f) => f.row),
    ["stub"],
  );
  assert.match(formatFatal(fatal[0]), /the stub row could not run \(spawn go ENOENT\)/);
});

test("the Go inventory reports an unparseable file rather than throwing", () => {
  const fixture = join(SCRIPT_DIR, "citations", `broken-${process.pid}.go.txt`);
  try {
    writeFileSync(fixture, "package x\n\nfunc (\n");
    assert.ok(goInventory(REPO_ROOT, [fixture]).get(fixture).error);
  } finally {
    rmSync(fixture, { force: true });
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
  // Only `go` leaves PATH: the gate shells out to `git` too, and that absence is another fault.
  const withoutGo = (process.env.PATH ?? "")
    .split(delimiter)
    .filter((dir) => dir !== "" && !existsSync(join(dir, "go")))
    .join(delimiter);
  const fixture = join(SCRIPT_DIR, "citations", `nogo-${process.pid}.md`);
  try {
    writeFileSync(fixture, `The lexer is \`${GO_TARGET}#Go.Lex\`.\n`);
    assert.equal(cliStatus([fixture], { ...process.env, PATH: withoutGo }), 2);
  } finally {
    rmSync(fixture, { force: true });
  }
});
