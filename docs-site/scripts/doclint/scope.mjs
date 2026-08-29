/*
 * In-scope file discovery (SPEC §1.3).
 *
 * The tool lints the same file set the documentation style standard §1.1 governs: five
 * doc families plus four root files. Everything else is out of scope.
 *
 * The `.md` filter on the family directories does the out-of-scope work for free: it
 * drops `docs/guides/embed.go`, the token files, and any generated non-markdown file
 * the SPEC §1.3 skip list names. The out-of-scope `docs/correspondence/` and
 * `docs/wayfinder/` directories are not family directories, so they never appear.
 */
import { readdirSync, existsSync } from "node:fs";
import { join, relative } from "node:path";

/** The five doc families, as repo-relative directory paths (SPEC §1.3). */
const FAMILY_DIRS = [
  "docs/adr",
  "docs/spec",
  "docs/agents",
  "docs/guides",
  "docs/research",
];

/** The four in-scope root files (SPEC §1.3). */
const ROOT_FILES = ["CONTEXT.md", "CLAUDE.md", "README.md", "SECURITY.md"];

/**
 * Every `.md` file under a directory, at any depth. The recursion closes a silent
 * scope hole: a doc nested in a subdirectory of a family (for example
 * `docs/research/2026/report.md`) is still in the family, so the tool must lint it.
 * @param {string} dir absolute path.
 * @returns {string[]}
 */
function markdownFilesUnder(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const abs = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...markdownFilesUnder(abs));
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      files.push(abs); // the .md filter drops embed.go, token files, generated files
    }
  }
  return files;
}

/**
 * Every in-scope file, as an absolute path. Used when the writer runs the tool with
 * no path argument and it lints the whole tree.
 * @param {string} repoRoot absolute path to the repo root.
 * @returns {string[]}
 */
export function inScopeFiles(repoRoot) {
  const files = [];
  for (const dir of FAMILY_DIRS) {
    const abs = join(repoRoot, dir);
    if (existsSync(abs)) files.push(...markdownFilesUnder(abs));
  }
  for (const name of ROOT_FILES) {
    const abs = join(repoRoot, name);
    if (existsSync(abs)) files.push(abs);
  }
  return files;
}

/**
 * Whether a file is in scope (SPEC §1.3), by path only. The CI job (#822) hands the tool the
 * changed files on a pull request. It filters that diff through this predicate, so an
 * out-of-scope doc never reaches a rule. The predicate uses the same FAMILY_DIRS and ROOT_FILES
 * constants as inScopeFiles(), so the whole-tree walk and the diff filter agree on one scope.
 *
 * A file is in scope when it is a `.md` file under one of the five family directories (at any
 * depth), or when it is one of the four in-scope root files (at the repo root exactly). A root
 * file name in a subdirectory (`docs/CONTEXT.md`) is not the root file, so it is out of scope.
 * @param {string} repoRoot absolute path to the repo root.
 * @param {string} absPath absolute path to the file.
 * @returns {boolean}
 */
export function isInScope(repoRoot, absPath) {
  const rel = relative(repoRoot, absPath).replace(/\\/g, "/");
  // A path outside the repo root resolves to a `../` prefix. It is never in scope.
  if (rel === "" || rel.startsWith("../")) return false;
  // A root file matches by exact repo-relative name, so a same-named file in a subdirectory
  // (docs/CONTEXT.md) does not match.
  if (ROOT_FILES.includes(rel)) return true;
  // Every family file is markdown. The `.md` filter drops embed.go, a token file, and any
  // generated non-markdown file in a family directory, the same drop inScopeFiles() makes.
  if (!rel.endsWith(".md")) return false;
  return FAMILY_DIRS.some((dir) => rel === dir || rel.startsWith(`${dir}/`));
}
