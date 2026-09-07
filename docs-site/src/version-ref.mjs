// A client island bundles this file, so nothing here may import `node:` or `astro:content`.

export const DEFAULT_VERSION = "main";
export const LATEST_VERSION = "latest";

const SEMVER_TAG = /^v(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/;

export function parseSemverTags(names) {
  const tags = [];
  for (const name of names) {
    const raw = String(name).trim();
    if (!raw) continue;
    const m = SEMVER_TAG.exec(raw);
    if (!m) continue;
    tags.push({
      raw,
      major: Number(m[1]),
      minor: Number(m[2]),
      patch: Number(m[3]),
      prerelease: m[4] ?? null,
    });
  }
  return tags.sort(compareTagsDesc);
}

function compareTagsDesc(a, b) {
  if (a.major !== b.major) return b.major - a.major;
  if (a.minor !== b.minor) return b.minor - a.minor;
  if (a.patch !== b.patch) return b.patch - a.patch;
  if (a.prerelease === null && b.prerelease !== null) return -1;
  if (a.prerelease !== null && b.prerelease === null) return 1;
  if (a.prerelease === null && b.prerelease === null) return 0;
  return b.prerelease.localeCompare(a.prerelease);
}

// "current" never names a prerelease, so a candidate is never the default (ADR-0155 §3).
export function newestStableTag(tags) {
  return tags.find((t) => t.prerelease === null) ?? null;
}

export function refForTagVersion(version, tags) {
  if (version === DEFAULT_VERSION) return DEFAULT_VERSION;
  if (version === LATEST_VERSION) {
    const stable = newestStableTag(tags);
    return stable ? stable.raw : DEFAULT_VERSION;
  }
  return version;
}

export function versionManifest(tags) {
  const stable = newestStableTag(tags);
  const latestRef = refForTagVersion(LATEST_VERSION, tags);
  // "current" names the newest stable release, or `latest` where none exists (ADR-0155 §3, #1445).
  const options = [
    stable
      ? { value: LATEST_VERSION, ref: latestRef }
      : { value: LATEST_VERSION, ref: latestRef, tag: "current" },
  ];
  for (const t of tags) {
    options.push(
      t.raw === stable?.raw
        ? { value: t.raw, ref: t.raw, tag: "current" }
        : { value: t.raw, ref: t.raw },
    );
  }
  options.push({ value: DEFAULT_VERSION, ref: DEFAULT_VERSION, tag: "dev" });
  return options;
}

// An unlisted version is its own ref; guessing one is what made `latest` resolve twice (#1402).
export function refFromManifest(version, versions) {
  const match = (versions ?? []).find((v) => v.value === version);
  return match ? match.ref : version;
}
