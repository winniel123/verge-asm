// The `Site` field of a Decision proposal block (SPEC docs/spec/citation-anchors.md §6, #1974).
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { environment, siteArm } from "./check-citations.mjs";
import {
  judgeSite,
  siteItems,
  formatSiteLineAnchor,
  fetchPullBody,
  pullContext,
  apiBase,
} from "./citations/site.mjs";
import { parse } from "./doclint/engine.mjs";
import { loadBurndown } from "./citations/burndown.mjs";

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = join(SCRIPT_DIR, "..", "..");
const ENV = environment(REPO_ROOT);

const GO_TARGET = "internal/commentlint/surface/golang.go";
const WHERE = "PR #1974 body";

function judge(body, env = ENV, root = REPO_ROOT) {
  return judgeSite(env, root, body, WHERE);
}

function block(site) {
  return [
    "## Decision proposal",
    "",
    "- **Thesis:** A citation names a region, never a line.",
    `- **Site:** ${site}`,
    "- **Alternative:** Keep the line anchor.",
    "- **Reversal:** cheap.",
    "- **Proof:** the suite below.",
    "- [ ] Ratified",
  ].join("\n");
}

const key = (r) => `${r.value}#${r.anchor}`;

test("a Site naming a real Go declaration passes", () => {
  const r = judge(block(`\`${GO_TARGET}#goDocSpans\``));
  assert.equal(r.fields, 1);
  assert.deepEqual(r.verified.map(key), [`${GO_TARGET}#goDocSpans`]);
  assert.deepEqual(r.broken, []);
  assert.deepEqual(r.dead, []);
});

test("a Site naming a declaration the target does not hold fails", () => {
  const r = judge(block(`\`${GO_TARGET}#noSuchDeclaration\``));
  assert.deepEqual(r.broken.map(key), [`${GO_TARGET}#noSuchDeclaration`]);
  assert.equal(r.broken[0].file, WHERE);
});

test("a Site naming no path in the tree is dead", () => {
  const r = judge(block("`internal/queue/no-such-file.go`"));
  assert.equal(r.dead.length, 1);
  assert.equal(r.dead[0].value, "internal/queue/no-such-file.go");
});

test("a Site carrying a bare path passes, because the gate asserts correctness", () => {
  // The checker asserts correctness, never presence (SPEC §7.2).
  const r = judge(block(`\`${GO_TARGET}\``));
  assert.deepEqual(r.anchored, []);
  assert.deepEqual(r.broken, []);
  assert.deepEqual(r.dead, []);
  assert.deepEqual(r.refused, []);
});

test("a Site carrying a line anchor is refused", () => {
  // SPEC §5. The refusal is unconditional here: the burn-down list never gains a Site entry.
  const token = "docs/adr/0144-the-verge-core-body-is-compiled-in-and-an-operator-edit-layers-over-it.md:34";
  const r = judge(block(`\`${token}\``));
  assert.deepEqual(r.refused.map((a) => a.token), [token]);
  assert.match(formatSiteLineAnchor(r.refused[0]), /a Site field names no line/);
});

test("a Site carrying an #L line anchor is refused", () => {
  const r = judge(block(`\`${GO_TARGET}#L12-L20\``));
  assert.equal(r.refused.length, 1);
});

test("prose in the field does not withdraw its own claim", () => {
  // The author writes the whole field, so a withdrawal word here is self-certification (§6).
  for (const site of [
    "`internal/queue/nope.go#Foo` (the deleted reader)",
    "~~`internal/queue/nope.go`~~",
    "`internal/queue/nope.go`, which is gone",
  ]) {
    assert.equal(judge(block(site)).dead.length, 1, site);
  }
});

test("a named ref does not rescue a line anchor in a Site field", () => {
  // A Site resolves against the merge ref alone, so a ref token pins nothing (SPEC §6 rule 2).
  const pinned = `on branch \`feat/x-y\`, \`${GO_TARGET}:34\``;
  assert.equal(judge(block(pinned)).refused.length, 1);
  assert.equal(judge(block(`\`${GO_TARGET}:34\``)).refused.length, 1);
});

test("a nested Site field is judged once, not twice", () => {
  const body = ["- **Site:** `internal/queue/nope.go#A`", "  - **Site:** `internal/queue/nope.go#B`"];
  const r = judge(body.join("\n"));
  assert.equal(r.fields, 1);
  assert.equal(r.dead.length, 2);
});

test("no burn-down entry ever names a pull-request body", () => {
  // The rule is new-blocks-only, and the list stays the sweep's own (SPEC §6 rule 4).
  assert.deepEqual(
    loadBurndown().filter((e) => e.file.startsWith("PR #")),
    [],
  );
});

test("a body with no Decision proposal block is judged and passes", () => {
  const r = judge("## Summary\n\nThis PR renames one function.\n");
  assert.equal(r.fields, 0);
  assert.deepEqual(r.broken, []);
  assert.deepEqual(r.dead, []);
  assert.deepEqual(r.refused, []);
});

test("free prose in a body is not judged, and the Site field is", () => {
  // This SPEC reaches the Site field and nothing else in a body (SPEC §6 rule 1).
  const body = [
    `The old reader was \`${GO_TARGET}#neverDeclared\`, and it is gone.`,
    "",
    block(`\`${GO_TARGET}#goDocSpans\``),
  ].join("\n");
  const r = judge(body);
  assert.deepEqual(r.broken, []);
  assert.deepEqual(r.verified.map(key), [`${GO_TARGET}#goDocSpans`]);
});

test("a Site field inside a fenced block is sample text, not a claim", () => {
  const body = ["```markdown", `- **Site:** \`${GO_TARGET}#neverDeclared\``, "```"].join("\n");
  assert.equal(judge(body).fields, 0);
});

test("the label binds on four spellings and on no other field", () => {
  const spellings = [
    "- **Site:** `x/y.go`",
    "- **Site**: `x/y.go`",
    "- Site: `x/y.go`",
    // A body may write the field as its own paragraph, and the label is still the reach.
    "**Site:** `x/y.go`",
  ];
  for (const line of spellings) {
    assert.equal(siteItems(parse(line)).length, 1, line);
  }
  for (const line of ["- **Thesis:** `x/y.go`", "- The site is `x/y.go`", "- **Proof:** `x/y.go`"]) {
    assert.equal(siteItems(parse(line)).length, 0, line);
  }
});

test("a paragraph-form Site field is judged like a list-item one", () => {
  const r = judge(`## Decision proposal\n\n**Site:** \`${GO_TARGET}#noSuchDeclaration\`\n`);
  assert.equal(r.fields, 1);
  assert.deepEqual(r.broken.map(key), [`${GO_TARGET}#noSuchDeclaration`]);
});

test("one list item is one field, however many paragraphs it holds", () => {
  const body = [
    "- **Site:**",
    "",
    `  \`${GO_TARGET}#goDocSpans\``,
    "- **Proof:** the suite below.",
  ].join("\n");
  const r = judge(body);
  assert.equal(r.fields, 1);
  assert.deepEqual(r.verified.map(key), [`${GO_TARGET}#goDocSpans`]);
});

test("a Site resolves against the tree the gate is handed, so an added target passes", () => {
  // A Site anchor resolves against the pull request's merge ref (SPEC §6 rule 2).
  const root = mkdtempSync(join(tmpdir(), "site-merge-ref-"));
  try {
    mkdirSync(join(root, "docs"));
    writeFileSync(join(root, "docs", "new.md"), "# Added by this pull request\n\nBody.\n");
    const env = {
      repoRoot: root,
      tracked: { files: new Set(["docs/new.md"]), dirs: new Set(["docs"]) },
      extensions: new Set([".md"]),
      roots: new Set(["docs"]),
      exempt: () => null,
    };
    const r = judgeSite(env, root, block("`docs/new.md#added-by-this-pull-request`"), WHERE);
    assert.deepEqual(r.verified.map(key), ["docs/new.md#added-by-this-pull-request"]);

    // The same anchor against a tree that never held the heading is broken.
    writeFileSync(join(root, "docs", "new.md"), "# Another title\n\nBody.\n");
    const gone = judgeSite(env, root, block("`docs/new.md#added-by-this-pull-request`"), WHERE);
    assert.deepEqual(gone.broken.map(key), ["docs/new.md#added-by-this-pull-request"]);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("the body comes from the REST API, never from the event payload", () => {
  // A `gh run rerun` replays a stale payload, so a body edit could never clear the gate.
  const seen = [];
  const fetchImpl = async (url, init) => {
    seen.push({ url, auth: init.headers.Authorization });
    return { ok: true, json: async () => ({ body: "the live body" }) };
  };
  return fetchPullBody({ repo: "o/r", number: 7, token: "t", fetchImpl }).then((body) => {
    assert.equal(body, "the live body");
    assert.deepEqual(seen, [
      { url: "https://api.github.com/repos/o/r/pulls/7", auth: "Bearer t" },
    ]);
  });
});

test("an HTTP failure is a claim the gate should judge and could not", async () => {
  const fetchImpl = async () => ({ ok: false, status: 404 });
  await assert.rejects(() => fetchPullBody({ repo: "o/r", number: 7, fetchImpl }), /HTTP 404/);
});

test("a transient status is retried once, and a client error is not", async () => {
  let calls = 0;
  const flaky = async () => {
    calls += 1;
    return calls === 1
      ? { ok: false, status: 503 }
      : { ok: true, json: async () => ({ body: "second try" }) };
  };
  assert.equal(await fetchPullBody({ repo: "o/r", number: 7, fetchImpl: flaky }), "second try");
  assert.equal(calls, 2);

  calls = 0;
  const gone = async () => {
    calls += 1;
    return { ok: false, status: 404 };
  };
  await assert.rejects(() => fetchPullBody({ repo: "o/r", number: 7, fetchImpl: gone }));
  assert.equal(calls, 1);
});

test("a run on GitHub Enterprise reads its own API host", () => {
  assert.equal(apiBase({}), "https://api.github.com");
  assert.equal(apiBase({ GITHUB_API_URL: "https://ghe.example/api/v3" }), "https://ghe.example/api/v3");
  const seen = [];
  const fetchImpl = async (url) => {
    seen.push(url);
    return { ok: true, json: async () => ({ body: "" }) };
  };
  return fetchPullBody({ repo: "o/r", number: 7, api: "https://ghe.example/api/v3", fetchImpl }).then(
    () => assert.deepEqual(seen, ["https://ghe.example/api/v3/repos/o/r/pulls/7"]),
  );
});

test("a named but unreadable payload is fatal, and no payload at all is not", async () => {
  // A payload named and unreadable is not the same claim as no payload (SPEC §7.7).
  const processEnv = { GITHUB_EVENT_PATH: "/e", GITHUB_REPOSITORY: "o/r" };
  const broke = await siteArm(ENV, REPO_ROOT, false, {
    processEnv,
    readEvent: () => "{ not json",
    fetchImpl: async () => assert.fail("no body is fetched when the payload cannot be read"),
  });
  assert.deepEqual(broke, { violations: 0, fatal: 1 });
  assert.deepEqual(await arm("irrelevant", {}), { violations: 0, fatal: 0 });
});

test("a null body reads as an empty body, not as a crash", async () => {
  const fetchImpl = async () => ({ ok: true, json: async () => ({ body: null }) });
  assert.equal(await fetchPullBody({ repo: "o/r", number: 7, fetchImpl }), "");
});

test("a run with no pull-request context has none to judge", () => {
  const read = () => JSON.stringify({ pull_request: { number: 7 } });
  assert.equal(pullContext({}, read), null);
  assert.equal(pullContext({ GITHUB_EVENT_PATH: "/e" }, read), null);
  assert.equal(pullContext({ GITHUB_REPOSITORY: "o/r" }, read), null);
  assert.deepEqual(pullContext({ GITHUB_EVENT_PATH: "/e", GITHUB_REPOSITORY: "o/r" }, read), {
    repo: "o/r",
    number: 7,
  });
});

test("a push event carries no pull request, so the gate judges no body", () => {
  const env = { GITHUB_EVENT_PATH: "/e", GITHUB_REPOSITORY: "o/r" };
  assert.equal(pullContext(env, () => JSON.stringify({ after: "abc" })), null);
});

test("an unreadable payload throws rather than reading as no pull request", () => {
  const env = { GITHUB_EVENT_PATH: "/e", GITHUB_REPOSITORY: "o/r" };
  assert.throws(() => pullContext(env, () => "{ not json"));
  assert.throws(() =>
    pullContext(env, () => {
      throw new Error("ENOENT");
    }),
  );
});

function arm(body, processEnv = { GITHUB_EVENT_PATH: "/e", GITHUB_REPOSITORY: "o/r" }) {
  return siteArm(ENV, REPO_ROOT, false, {
    processEnv,
    readEvent: () => JSON.stringify({ pull_request: { number: 1974, body: "the stale payload" } }),
    fetchImpl: async () => ({ ok: true, json: async () => ({ body }) }),
  });
}

test("the arm judges the body the API returns, not the one the payload carries", async () => {
  // A `gh run rerun` replays a stale payload, so the two paths must agree (#1974).
  assert.deepEqual(await arm(block(`\`${GO_TARGET}#goDocSpans\``)), { violations: 0, fatal: 0 });
  assert.deepEqual(await arm(block(`\`${GO_TARGET}#noSuchDeclaration\``)), {
    violations: 1,
    fatal: 0,
  });
});

test("the arm judges nothing without a pull-request context", async () => {
  assert.deepEqual(await arm("irrelevant", {}), { violations: 0, fatal: 0 });
});

test("the arm reports an unreachable API as fatal, never as a pass", async () => {
  const r = await siteArm(ENV, REPO_ROOT, false, {
    processEnv: { GITHUB_EVENT_PATH: "/e", GITHUB_REPOSITORY: "o/r" },
    readEvent: () => JSON.stringify({ pull_request: { number: 1974 } }),
    fetchImpl: async () => ({ ok: false, status: 403 }),
  });
  assert.deepEqual(r, { violations: 0, fatal: 1 });
});

test("the CLI passes with no pull-request context, and says it judged no body", () => {
  // A runner exports both keys, so a local shape has to be built rather than inherited.
  const { GITHUB_EVENT_PATH, GITHUB_REPOSITORY, ...local } = process.env;
  const out = execFileSync(
    process.execPath,
    [join(SCRIPT_DIR, "check-citations.mjs"), "--in-scope-only", join(REPO_ROOT, "CLAUDE.md")],
    { encoding: "utf8", stdio: "pipe", env: local },
  );
  assert.match(out, /no pull-request context, so it judged no Site field/);
});
