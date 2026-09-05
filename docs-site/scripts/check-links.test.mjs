import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { loadVersions, findFailures } from "./check-links.mjs";

function git(root, ...args) {
  execFileSync("git", args, { cwd: root, encoding: "utf8", stdio: "pipe" });
}

function makeRepo() {
  const root = mkdtempSync(join(tmpdir(), "check-links-"));
  mkdirSync(join(root, "docs", "guides"), { recursive: true });
  mkdirSync(join(root, "docs", "adr"), { recursive: true });
  git(root, "init", "-b", "main");
  git(root, "config", "user.email", "test@example.com");
  git(root, "config", "user.name", "test");
  git(root, "config", "commit.gpgsign", "false");
  return root;
}

function write(root, path, body) {
  writeFileSync(join(root, path), body, "utf8");
}

function commit(root, message) {
  git(root, "add", "-A");
  git(root, "commit", "-m", message);
}

function reasonsFor(failures, version) {
  return failures.filter((f) => f.version === version).map((f) => f.reason);
}

test("a dead link in a tagged version fails the gate while the working tree is clean", () => {
  const root = makeRepo();
  try {
    write(root, "docs/guides/alpha.md", "# Alpha\n\nSee [beta](./gone.md).\n");
    write(root, "docs/guides/beta.md", "# Beta\n");
    commit(root, "tagged state");
    git(root, "tag", "v1.0.0");

    write(root, "docs/guides/alpha.md", "# Alpha\n\nSee [beta](./beta.md).\n");
    commit(root, "fix the link on main");

    const versions = loadVersions(root);
    assert.deepEqual(
      versions.map((v) => v.version),
      ["latest", "v1.0.0", "main"],
    );

    const failures = findFailures(versions);
    assert.deepEqual(reasonsFor(failures, "main"), []);
    assert.deepEqual(reasonsFor(failures, "v1.0.0"), [
      'guide "gone" does not exist in this version',
    ]);
    assert.deepEqual(reasonsFor(failures, "latest"), [
      'guide "gone" does not exist in this version',
    ]);
    assert.equal(
      failures.find((f) => f.version === "v1.0.0").file,
      "docs/guides/alpha.md",
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("the main version reads the working tree, not the committed ref", () => {
  const root = makeRepo();
  try {
    write(root, "docs/guides/alpha.md", "# Alpha\n");
    commit(root, "clean state");
    write(root, "docs/guides/alpha.md", "# Alpha\n\nSee [gone](./gone.md).\n");

    const failures = findFailures(loadVersions(root));
    assert.deepEqual(reasonsFor(failures, "main"), [
      'guide "gone" does not exist in this version',
    ]);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("latest resolves to main while no stable tag exists", () => {
  const root = makeRepo();
  try {
    write(root, "docs/guides/alpha.md", "# Alpha\n");
    commit(root, "no tags");
    write(root, "docs/guides/alpha.md", "# Alpha\n\nSee [gone](./gone.md).\n");

    const versions = loadVersions(root);
    assert.deepEqual(
      versions.map((v) => v.version),
      ["latest", "main"],
    );
    assert.deepEqual(reasonsFor(findFailures(versions), "latest"), [
      'guide "gone" does not exist in this version',
    ]);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("a prerelease tag never becomes latest", () => {
  const root = makeRepo();
  try {
    write(root, "docs/guides/alpha.md", "# Alpha\n");
    commit(root, "base");
    git(root, "tag", "v2.0.0-rc.1");

    const versions = loadVersions(root);
    assert.deepEqual(
      versions.map((v) => v.version),
      ["latest", "v2.0.0-rc.1", "main"],
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("each version validates ADR cross-references against its own docs/adr", () => {
  const root = makeRepo();
  try {
    write(root, "docs/guides/alpha.md", "# Alpha\n\nSee [ADR](../adr/ADR-0002.md).\n");
    write(root, "docs/adr/ADR-0001.md", "# One\n");
    commit(root, "tagged state without ADR-0002");
    git(root, "tag", "v1.0.0");

    write(root, "docs/adr/ADR-0002.md", "# Two\n");
    commit(root, "add ADR-0002 on main");

    const failures = findFailures(loadVersions(root));
    assert.deepEqual(reasonsFor(failures, "main"), []);
    assert.deepEqual(reasonsFor(failures, "v1.0.0"), [
      'ADR "ADR-0002.md" does not exist in docs/adr/',
    ]);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("a tag without docs/guides is skipped, as the site skips it", () => {
  const root = makeRepo();
  try {
    write(root, "README.md", "# Repo\n");
    commit(root, "no guides yet");
    git(root, "tag", "v0.1.0");

    write(root, "docs/guides/alpha.md", "# Alpha\n");
    commit(root, "add guides");

    const versions = loadVersions(root);
    assert.deepEqual(
      versions.map((v) => v.version),
      ["latest", "main"],
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
