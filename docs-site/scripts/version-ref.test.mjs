import test from "node:test";
import assert from "node:assert/strict";
import {
  DEFAULT_VERSION,
  LATEST_VERSION,
  parseSemverTags,
  newestStableTag,
  refForTagVersion,
  versionManifest,
  refFromManifest,
  landingVersion,
} from "../src/version-ref.mjs";

// The repo has no `v*` tag, so a fixture list is the only way to reach a released state (#1402).
const FIXTURES = {
  none: [],
  oneStable: ["v1.0.0"],
  stableAndOlder: ["v0.9.2", "v1.0.0", "v0.10.0"],
  prereleaseAhead: ["v1.0.0", "v1.0.1-rc1", "v1.0.1-rc2"],
  onlyPrerelease: ["v1.0.0-rc1"],
  noise: ["v1.0.0", "nightly", "v1", "release-2", ""],
};

test("parseSemverTags orders newest first and drops a non-semver name", () => {
  assert.deepEqual(
    parseSemverTags(FIXTURES.stableAndOlder).map((t) => t.raw),
    ["v1.0.0", "v0.10.0", "v0.9.2"],
  );
  assert.deepEqual(parseSemverTags(FIXTURES.noise).map((t) => t.raw), ["v1.0.0"]);
});

test("a prerelease sorts above the release it follows but is never the stable pick", () => {
  const tags = parseSemverTags(FIXTURES.prereleaseAhead);
  assert.deepEqual(tags.map((t) => t.raw), ["v1.0.1-rc2", "v1.0.1-rc1", "v1.0.0"]);
  assert.equal(newestStableTag(tags)?.raw, "v1.0.0");
});

test("refForTagVersion resolves latest to the highest stable tag (ADR-0115 §2, ADR-0155)", () => {
  assert.equal(refForTagVersion(LATEST_VERSION, parseSemverTags(FIXTURES.oneStable)), "v1.0.0");
  assert.equal(refForTagVersion(LATEST_VERSION, parseSemverTags(FIXTURES.stableAndOlder)), "v1.0.0");
  assert.equal(refForTagVersion(LATEST_VERSION, parseSemverTags(FIXTURES.prereleaseAhead)), "v1.0.0");
});

test("latest falls back to main only while no stable tag exists", () => {
  assert.equal(refForTagVersion(LATEST_VERSION, parseSemverTags(FIXTURES.none)), DEFAULT_VERSION);
  assert.equal(refForTagVersion(LATEST_VERSION, parseSemverTags(FIXTURES.onlyPrerelease)), DEFAULT_VERSION);
});

test("main and a concrete tag are their own ref", () => {
  const tags = parseSemverTags(FIXTURES.stableAndOlder);
  assert.equal(refForTagVersion(DEFAULT_VERSION, tags), DEFAULT_VERSION);
  assert.equal(refForTagVersion("v0.9.2", tags), "v0.9.2");
});

test("the manifest carries the resolved ref, so latest never reads as main once a tag exists", () => {
  const manifest = versionManifest(parseSemverTags(FIXTURES.oneStable));
  const latest = manifest.find((v) => v.value === LATEST_VERSION);
  assert.equal(latest.ref, "v1.0.0");
  assert.notEqual(latest.ref, DEFAULT_VERSION);
});

test("the manifest keeps its order and its badges", () => {
  assert.deepEqual(versionManifest(parseSemverTags(FIXTURES.none)), [
    { value: "latest", ref: "main", tag: "current" },
    { value: "main", ref: "main", tag: "dev" },
  ]);
  assert.deepEqual(versionManifest(parseSemverTags(FIXTURES.prereleaseAhead)), [
    { value: "latest", ref: "v1.0.0" },
    { value: "v1.0.1-rc2", ref: "v1.0.1-rc2" },
    { value: "v1.0.1-rc1", ref: "v1.0.1-rc1" },
    { value: "v1.0.0", ref: "v1.0.0", tag: "current" },
    { value: "main", ref: "main", tag: "dev" },
  ]);
});

// The zero-tag repo hid this: every assertion about the pair passed by coincidence (#1402).
test("the client-side lookup returns the same ref the git-side rule resolves", () => {
  for (const [name, names] of Object.entries(FIXTURES)) {
    const tags = parseSemverTags(names);
    const manifest = versionManifest(tags);
    for (const option of manifest) {
      assert.equal(
        refFromManifest(option.value, manifest),
        refForTagVersion(option.value, tags),
        `${name}: ${option.value} resolves to two refs`,
      );
    }
  }
});

test("refFromManifest reads the manifest rather than re-deriving from the version string", () => {
  const manifest = versionManifest(parseSemverTags(FIXTURES.oneStable));
  assert.equal(refFromManifest(LATEST_VERSION, manifest), "v1.0.0");
  assert.equal(refFromManifest(DEFAULT_VERSION, manifest), DEFAULT_VERSION);
  assert.equal(refFromManifest("v1.0.0", manifest), "v1.0.0");
});

test("an unlisted version resolves to itself, never to main", () => {
  const manifest = versionManifest(parseSemverTags(FIXTURES.oneStable));
  assert.equal(refFromManifest("v0.1.0", manifest), "v0.1.0");
  assert.equal(refFromManifest(LATEST_VERSION, []), LATEST_VERSION);
});

// listVersions() declared `ref` required and never set it, so this returned undefined (#1445).
test("every manifest row carries a ref, so the client lookup never reads undefined", () => {
  for (const [name, names] of Object.entries(FIXTURES)) {
    for (const option of versionManifest(parseSemverTags(names))) {
      assert.equal(typeof option.ref, "string", `${name}: ${option.value} carries no ref`);
      assert.notEqual(option.ref, "", `${name}: ${option.value} carries an empty ref`);
    }
  }
});

test("the current badge names the newest stable release, and latest only where none exists", () => {
  for (const [name, names] of Object.entries(FIXTURES)) {
    const tags = parseSemverTags(names);
    const badged = versionManifest(tags).filter((v) => v.tag === "current");
    assert.equal(badged.length, 1, `${name}: expected exactly one current row`);
    const stable = newestStableTag(tags);
    assert.equal(badged[0].value, stable ? stable.raw : LATEST_VERSION, name);
    assert.notEqual(badged[0].value, DEFAULT_VERSION, `${name}: current named the dev row`);
  }
});

// This is the state the repo is in today: `git tag -l "v*"` returns nothing (#1445).
test("with no tag at all, latest takes the badge and resolves to main", () => {
  const manifest = versionManifest(parseSemverTags(FIXTURES.none));
  assert.deepEqual(manifest, [
    { value: LATEST_VERSION, ref: DEFAULT_VERSION, tag: "current" },
    { value: DEFAULT_VERSION, ref: DEFAULT_VERSION, tag: "dev" },
  ]);
  assert.equal(refFromManifest(LATEST_VERSION, manifest), DEFAULT_VERSION);
});

// A prerelease is browsable and never current, so the only tag here takes no badge (ADR-0155 §3).
test("a prerelease-only tag list leaves the badge on latest", () => {
  const manifest = versionManifest(parseSemverTags(FIXTURES.onlyPrerelease));
  assert.deepEqual(manifest, [
    { value: LATEST_VERSION, ref: DEFAULT_VERSION, tag: "current" },
    { value: "v1.0.0-rc1", ref: "v1.0.0-rc1" },
    { value: DEFAULT_VERSION, ref: DEFAULT_VERSION, tag: "dev" },
  ]);
});

// TopNav's fallback runs only where a caller omits `version`, and no route omits it (#1570).
test("the landing version is the latest alias under every tag list", () => {
  for (const [name, names] of Object.entries(FIXTURES)) {
    assert.equal(landingVersion(versionManifest(parseSemverTags(names))), LATEST_VERSION, name);
  }
});

test("the badge and the landing version name different rows once a stable tag exists", () => {
  const manifest = versionManifest(parseSemverTags(FIXTURES.oneStable));
  assert.equal(manifest.find((v) => v.tag === "current").value, "v1.0.0");
  assert.equal(landingVersion(manifest), LATEST_VERSION);
});

test("a list with no latest row falls back to its first, and an empty list to undefined", () => {
  assert.equal(landingVersion([{ value: "v2.0.0", ref: "v2.0.0", tag: "current" }]), "v2.0.0");
  assert.equal(landingVersion([]), undefined);
  assert.equal(landingVersion(undefined), undefined);
});
