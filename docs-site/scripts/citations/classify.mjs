import { execFileSync } from "node:child_process";
import { posix } from "node:path";

// A template stands for a path a reader supplies, so no tree can hold it (#1450).
const PLACEHOLDER = /[<>{}*?]|N{3,}|n{4,}|X\.Y\.Z|\.\.\./;

const DIR_CITATION = /\/$/;

// dev.mysql.com/doc/... and kafka.apache.org/... are URLs a document writes without a scheme.
const HOSTNAME = /^[a-z0-9-]+(?:\.[a-z0-9-]+)+$/i;
const TLD = /\.[a-z]{2,}$/i;

// @ds/tokens/ and @astrojs/react are npm scopes, not directories.
const NPM_SCOPE = /^@/;

// #1450 repaired dead paths by stating absence at the site, and a bare "removed" is too loose.
const WITHDRAWN_IN_PROSE = new RegExp(
  [
    "not on disk",
    "not on `?main`?",
    "not in the tree",
    "no such (?:file|path|directory)",
    "does not (?:yet )?exist",
    "never existed",
    "no longer exists?",
    "(?:is|are) gone",
    "not reachable",
    "unreachable",
    "no successor",
    "dead (?:reference|pointer|token)",
    "deleted",
    "retired",
    "withdrawn",
    "superseded",
    "not yet (?:written|built|created|landed)",
  ].join("|"),
  "i",
);

export function trackedPaths(repoRoot) {
  const out = execFileSync("git", ["ls-files", "-z"], {
    cwd: repoRoot,
    encoding: "utf8",
    maxBuffer: 256 * 1024 * 1024,
  });
  const files = new Set();
  const dirs = new Set();
  for (const f of out.split("\0")) {
    if (f === "") continue;
    files.add(f);
    const parts = f.split("/");
    for (let i = 1; i < parts.length; i++) dirs.add(parts.slice(0, i).join("/"));
  }
  return { files, dirs };
}

export function topLevelEntries(tracked) {
  const roots = new Set();
  for (const f of tracked.files) roots.add(f.split("/")[0]);
  return roots;
}

export function trackedExtensions(tracked) {
  const exts = new Set();
  for (const f of tracked.files) {
    const base = f.slice(f.lastIndexOf("/") + 1);
    const dot = base.lastIndexOf(".");
    if (dot > 0) exts.add(base.slice(dot).toLowerCase());
  }
  return exts;
}

function stripTarget(raw) {
  let value = raw;
  const hash = value.indexOf("#");
  if (hash >= 0) value = value.slice(0, hash);
  const query = value.indexOf("?");
  if (query >= 0) value = value.slice(0, query);
  return value.replace(/[.,;:)\]]+$/, "");
}

// A ./ or ../ prefix fixes the reading; anything else is read both ways, because the tree
// writes a sibling link and a repo-rooted citation in the same bare form.
function normalize(docFile, value) {
  const docRelative = posix.normalize(posix.join(posix.dirname(docFile), value));
  if (value.startsWith("./") || value.startsWith("../")) {
    // A target that climbs past the repo root names nothing this gate can ever read (#1436).
    if (docRelative.startsWith("../")) return { candidates: [], rooted: null, escapes: true };
    return { candidates: [docRelative], rooted: docRelative };
  }
  // A document may write a path against its own package root, not the repo root (#1407).
  const candidates = [];
  let base = posix.dirname(docFile);
  while (base !== "." && base !== "/") {
    candidates.push(posix.normalize(posix.join(base, value)));
    base = posix.dirname(base);
  }
  const rootRelative = posix.normalize(value);
  candidates.push(rootRelative);
  return {
    candidates: candidates.filter((c) => !c.startsWith("../")),
    rooted: rootRelative.startsWith("../") ? null : rootRelative,
  };
}

// Tracked content only: an untracked or gitignored file resolves here and not on a clean checkout.
function present(tracked, path) {
  const clean = path.replace(/\/$/, "");
  if (clean === "") return false;
  return tracked.files.has(clean) || tracked.dirs.has(clean);
}

// node_modules/ is installed, never tracked, so a citation to it is about an artefact the
// gate cannot read from the tree at all. .gitignore is the repo's own statement of that.
function gitIgnored(repoRoot, path) {
  const parts = path.replace(/\/$/, "").split("/");
  // Shortest prefix first: an ignored ancestor settles it, and git refuses to look past one.
  for (let i = 1; i <= parts.length; i++) {
    const prefix = parts.slice(0, i).join("/");
    // A `dir/` pattern matches only a directory, and git reads that from disk unless we say so.
    for (const probe of [prefix, `${prefix}/`]) {
      try {
        execFileSync("git", ["check-ignore", "-q", "--", probe], {
          cwd: repoRoot,
          stdio: "ignore",
        });
        return true;
      } catch {
        continue;
      }
    }
  }
  return false;
}

function refShapes(ref) {
  if (/^[0-9a-f]{7,40}$/.test(ref)) return [ref];
  return [ref, `origin/${ref}`, `refs/remotes/origin/${ref}`];
}

function pathOnRef(repoRoot, ref, path) {
  const clean = path.replace(/\/$/, "");
  for (const shape of refShapes(ref)) {
    try {
      execFileSync("git", ["cat-file", "-e", `${shape}:${clean}`], {
        cwd: repoRoot,
        stdio: "ignore",
      });
      return shape;
    } catch {
      continue;
    }
  }
  return null;
}

function refKnown(repoRoot, ref) {
  for (const shape of refShapes(ref)) {
    try {
      execFileSync("git", ["rev-parse", "--verify", "--quiet", `${shape}^{commit}`], {
        cwd: repoRoot,
        stdio: "ignore",
      });
      return true;
    } catch {
      continue;
    }
  }
  return false;
}

// Ordered because the first reason that applies is the one a reader needs.
function ignoreReason(citation, value, tracked, extensions) {
  if (PLACEHOLDER.test(value)) return "template placeholder";
  if (!value.includes("/")) return "not a path";
  if (NPM_SCOPE.test(value)) return "npm scope";
  const first = value.slice(0, value.indexOf("/"));
  if (HOSTNAME.test(first) && TLD.test(first)) return "a URL written without its scheme";
  if (DIR_CITATION.test(value)) return null;
  const base = value.slice(value.lastIndexOf("/") + 1);
  const dot = base.lastIndexOf(".");
  if (dot <= 0) return "no file extension";
  // An unused extension is not a file name, which excludes a Go package.Symbol token (#1450).
  if (!extensions.has(base.slice(dot).toLowerCase())) return "not a file extension the tree uses";
  return null;
}

export function classify(env, docFile, citations) {
  const { repoRoot, tracked, extensions, roots, exempt } = env;
  const results = [];
  for (const citation of citations) {
    const value = stripTarget(citation.raw);
    if (value === "" || value === "." || value === "./") continue;

    const skip = ignoreReason(citation, value, tracked, extensions);
    if (skip) {
      results.push({ ...citation, value, status: "ignored", reason: skip });
      continue;
    }

    const { candidates, rooted, escapes } = normalize(docFile, value);
    if (escapes) {
      results.push({ ...citation, value, status: "dead", detail: "climbs past the repo root" });
      continue;
    }
    if (candidates.length === 0) continue;
    const hit = candidates.find((c) => present(tracked, c));
    if (hit) {
      results.push({ ...citation, value, path: hit, status: "ok" });
      continue;
    }

    // A path rooted at no top-level entry of this repo addresses another project's tree (#1450).
    if (rooted === null || !roots.has(rooted.split("/")[0])) {
      results.push({ ...citation, value, status: "foreign" });
      continue;
    }

    const path = rooted;
    if (gitIgnored(repoRoot, path)) {
      results.push({ ...citation, value, path, status: "untracked" });
      continue;
    }

    const exemption = exempt(docFile, path, value);
    if (exemption) {
      results.push({ ...citation, value, path, status: "exempt", reason: exemption });
      continue;
    }

    // A ref beside the path moves the claim to that ref, and the nearest ref wins (#1436).
    const refs = [...citation.refs].sort(
      (a, b) => Math.abs(a.line - citation.line) - Math.abs(b.line - citation.line),
    );
    let unknownRef = null;
    let resolved = null;
    for (const { ref } of refs) {
      const on = candidates.map((c) => pathOnRef(repoRoot, ref, c)).find(Boolean);
      if (on) {
        resolved = { ref, shape: on };
        break;
      }
      if (!refKnown(repoRoot, ref) && unknownRef === null) unknownRef = ref;
    }
    if (resolved) {
      results.push({ ...citation, value, path, status: "on-ref", ref: resolved.shape });
      continue;
    }
    if (unknownRef) {
      results.push({ ...citation, value, path, status: "ref-unknown", ref: unknownRef });
      continue;
    }
    if (citation.withdrawn || WITHDRAWN_IN_PROSE.test(citation.prose)) {
      results.push({ ...citation, value, path, status: "withdrawn" });
      continue;
    }

    results.push({ ...citation, value, path, status: "dead" });
  }
  return results;
}
