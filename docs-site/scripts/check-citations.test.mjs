import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { writeFileSync, rmSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { extractCitations } from "./citations/extract.mjs";
import { classify } from "./citations/classify.mjs";
import { loadExemptions } from "./citations/exempt.mjs";
import { isInScope, inScopeFiles } from "./citations/scope.mjs";
import { environment, run } from "./check-citations.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = join(SCRIPT_DIR, "..", "..");
const ENV = environment(REPO_ROOT);

const DOC = "docs/adr/0001-stack-and-runtime.md";

function results(markdown, docFile = DOC) {
  return classify(ENV, docFile, extractCitations(markdown));
}

function dead(markdown, docFile = DOC) {
  return results(markdown, docFile)
    .filter((r) => r.status === "dead")
    .map((r) => r.value);
}

function statusOf(markdown, value, docFile = DOC) {
  const hit = results(markdown, docFile).find((r) => r.value === value);
  return hit ? hit.status : "absent";
}

test("a code span naming a path the tree does not hold is dead", () => {
  assert.deepEqual(dead("The ruling sits in `docs/spec/no-such-spec.md` today."), [
    "docs/spec/no-such-spec.md",
  ]);
});

test("a markdown link to a deleted file is dead", () => {
  assert.deepEqual(dead("See [the chart](../../design-system/PARITY-CHART.md)."), [
    "../../design-system/PARITY-CHART.md",
  ]);
});

test("a path the tree holds passes", () => {
  assert.deepEqual(dead("The engine is `docs-site/scripts/doclint/engine.mjs`."), []);
});

test("a sibling link inside a family passes", () => {
  assert.deepEqual(dead("See [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)."), []);
});

test("a citation relative to the document's package root passes", () => {
  const markdown = "Start at `docs/AGENT-GUIDE.md`.";
  assert.deepEqual(dead(markdown, "design-system/docs/DESIGN-NOTES.md"), []);
});

test("a struck-through path is a historical record, not a violation", () => {
  assert.deepEqual(dead("The capture ~~`design-system/screenshots/docs.jpg`~~ went with it."), []);
});

test("a path inside a WITHDRAWN blockquote is a historical record", () => {
  const markdown = "> **WITHDRAWN 2026-08-28.** The ruling lived in `design-system/SPEC-CHANGE.md` #19c.\n";
  assert.deepEqual(dead(markdown), []);
});

test("a path whose absence the site states in prose is a historical record", () => {
  const markdown =
    "The view arrived as `design-system/verify/WORK-ORDER-390-391.md`. That file is not on `main`, and no successor carries it.";
  assert.equal(statusOf(markdown, "design-system/verify/WORK-ORDER-390-391.md"), "withdrawn");
});

test("another project's tree is not judged against ours, and gets its own status", () => {
  const markdown = "Cassandra ships `conf/cassandra.yaml` and Kubernetes ships `pkg/cluster/ports/ports.go`.";
  assert.deepEqual(dead(markdown), []);
  for (const r of results(markdown)) assert.equal(r.status, "foreign");
});

test("a template placeholder is not a path", () => {
  const markdown = "Write `docs/release-notes/<tag>.md`, then `docs/release-notes/vX.Y.Z.md`, then `db/migrations/NNNNN_thing.sql`.";
  assert.deepEqual(dead(markdown), []);
});

test("a Go package.Symbol token is not a path", () => {
  const markdown = "`internal/queue.Worker` calls `internal/delivery.Runner` and `internal/release.Store`.";
  assert.deepEqual(dead(markdown), []);
});

test("a URL route and a markdown template are not paths", () => {
  const markdown = "Open `/settings?tab=api` and file [#NNN](url).";
  assert.deepEqual(dead(markdown), []);
});

test("a URL written without its scheme is not a path", () => {
  assert.deepEqual(dead("See `dev.mysql.com/doc/refman/8.4/en/security-guidelines.html`."), []);
});

test("a fenced code block is sample text", () => {
  const markdown = "```sh\ncat docs/spec/no-such-spec.md\n```\n";
  assert.deepEqual(dead(markdown), []);
});

test("an npm scope is not a directory", () => {
  assert.deepEqual(dead("Import from `@ds/tokens/` and `@astrojs/react`."), []);
});

test("docs/research is out of scope", () => {
  assert.equal(isInScope(REPO_ROOT, join(REPO_ROOT, "docs/research/anything.md")), false);
  assert.equal(isInScope(REPO_ROOT, join(REPO_ROOT, "docs/adr/0001-stack-and-runtime.md")), true);
  assert.equal(isInScope(REPO_ROOT, join(REPO_ROOT, "design-system/docs/DESIGN-NOTES.md")), true);
});

test("a path cited against a branch resolves on that branch, never dead", () => {
  const markdown =
    "Prototype source: branch `research/843-rawoutput-prototype`, commit `6142247`, `design-system/templates/rundetail-rawoutput.prototype.html`.";
  const status = statusOf(markdown, "design-system/templates/rundetail-rawoutput.prototype.html");
  // A clone without the branch must say the ref is gone, never that the path rotted.
  assert.ok(status === "on-ref" || status === "ref-unknown", `got ${status}`);
});

test("a path cited against a ref no clone holds reports the ref, not the path", () => {
  const markdown =
    "The UI prototype lives on branch `prototype/ct-source-859` at `prototypes/ct-source/index.html`.";
  const hit = results(markdown).find((r) => r.value === "prototypes/ct-source/index.html");
  assert.equal(hit.status, "ref-unknown");
  assert.equal(hit.ref, "prototype/ct-source-859");
});

test("a repo slug beside a path is not a ref", () => {
  const markdown = "Measured on `apache/kafka`: `docs/security/no-such-file.md` says so.";
  assert.deepEqual(dead(markdown), ["docs/security/no-such-file.md"]);
});

test("every exemption names a document that still exists", () => {
  for (const e of loadExemptions()) {
    assert.ok(existsSync(join(REPO_ROOT, e.file)), `exemption names a missing document: ${e.file}`);
    assert.ok(e.reason.length > 20, `exemption needs a reason: ${e.file}`);
  }
});

test("every exemption still suppresses something", () => {
  const byFile = new Map();
  for (const r of run(REPO_ROOT, inScopeFiles(REPO_ROOT)).results) {
    if (r.status !== "exempt") continue;
    byFile.set(`${r.file}\u0000${r.path}`, true);
    byFile.set(`${r.file}\u0000${r.value}`, true);
  }
  for (const e of loadExemptions()) {
    assert.ok(byFile.has(`${e.file}\u0000${e.path}`), `stale exemption: ${e.file} -> ${e.path}`);
  }
});

test("a bare removed or reachable no longer suppresses a dead path", () => {
  assert.deepEqual(dead("The old handler was removed. See `docs/spec/no-such-file.md` for the rule."), [
    "docs/spec/no-such-file.md",
  ]);
  assert.deepEqual(dead("Nothing here is reachable from the console. See `docs/spec/no-such-file.md`."), [
    "docs/spec/no-such-file.md",
  ]);
});

test("an untracked file does not satisfy the gate", () => {
  const fixture = join(SCRIPT_DIR, "citations", `untracked-${process.pid}.md`);
  writeFileSync(fixture, "placeholder\n");
  try {
    const value = `docs-site/scripts/citations/untracked-${process.pid}.md`;
    assert.ok(existsSync(fixture));
    assert.deepEqual(dead(`The note is \`${value}\`.`), [value]);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("a path .gitignore covers is installed, not authored, and gets its own status", () => {
  const markdown = "`esbuild` lives in `docs-site/node_modules/.bin/`, and a diff lands in `docs-site/tests/diff/`.";
  assert.deepEqual(dead(markdown), []);
  for (const r of results(markdown)) assert.equal(r.status, "untracked");
});

test("a target that climbs past the repo root is dead, and is never probed", () => {
  const hit = results("See [it](../../../outside.md).").find((r) => r.value === "../../../outside.md");
  assert.equal(hit.status, "dead");
  assert.equal(hit.detail, "climbs past the repo root");
  assert.equal(hit.path, undefined);
});

test("the tree cites no dead path", () => {
  const rot = run(REPO_ROOT, inScopeFiles(REPO_ROOT)).results.filter((r) => r.status === "dead");
  const detail = rot.map((r) => `  ${r.file}:${r.line} -> ${r.value}`).join("\n");
  assert.equal(rot.length, 0, `\n${detail}`);
});

test("every citation the gate passes over is counted, and --verbose lists it", () => {
  const cli = join(SCRIPT_DIR, "check-citations.mjs");
  const out = execFileSync("node", [cli], { encoding: "utf8" });
  for (const line of [
    /\d+ path citation\(s\) judged/,
    /\d+ {2}resolve in the tracked tree/,
    /\d+ {2}absent, and the site withdraws the claim there/,
    /\d+ {2}absent, and exemptions\.json says why/,
    /\d+ {2}address another project's tree/,
    /\d+ {2}a path \.gitignore covers, so the tree never tracks it/,
    /\d+ {2}resolve on a ref the document names/,
    /\d+ {2}name a ref this clone cannot see/,
    /\d+ {2}dead/,
    /\d+ candidate\(s\) were not path citations, and are not judged/,
    /--verbose/,
  ]) {
    assert.match(out, line);
  }

  const verbose = execFileSync("node", [cli, "--verbose"], { encoding: "utf8" });
  for (const heading of [
    /Absent, and the site withdraws the claim there:/,
    /Absent, and exemptions\.json says why:/,
    /Addresses another project's tree, so this gate cannot judge it:/,
    /A path \.gitignore covers, so the tree never tracks it:/,
    /Resolved against a ref the document names:/,
  ]) {
    assert.match(verbose, heading);
  }
});

test("the judged total is the sum of its buckets, so no citation hides in a rounding", () => {
  const { results: all } = run(REPO_ROOT, inScopeFiles(REPO_ROOT));
  const count = (status) => all.filter((r) => r.status === status).length;
  const judged = all.length - count("ignored");
  const buckets = ["ok", "withdrawn", "exempt", "foreign", "untracked", "on-ref", "ref-unknown", "dead"];
  assert.equal(buckets.reduce((sum, s) => sum + count(s), 0), judged);
});

function cliStatus(args) {
  try {
    execFileSync("node", [join(SCRIPT_DIR, "check-citations.mjs"), ...args], {
      encoding: "utf8",
      stdio: "pipe",
    });
    return 0;
  } catch (err) {
    return err.status;
  }
}

test("the CLI exits non-zero on a dead path", () => {
  const fixture = join(SCRIPT_DIR, "citations", `fixture-${process.pid}.md`);
  writeFileSync(fixture, "The ruling sits in `docs/spec/no-such-spec.md`.\n");
  try {
    assert.equal(cliStatus([fixture]), 1);
  } finally {
    rmSync(fixture, { force: true });
  }
});

test("the CLI exits 2 on an argument it cannot read", () => {
  assert.equal(cliStatus([join(REPO_ROOT, "docs", `gone-${process.pid}.md`)]), 2);
  assert.equal(cliStatus([join(REPO_ROOT, "docs", "adr")]), 2);
});
