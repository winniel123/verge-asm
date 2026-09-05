import { readdirSync, existsSync } from "node:fs";
import { join, relative } from "node:path";

const FAMILY_DIRS = [
  "docs/adr",
  "docs/spec",
  "docs/agents",
  "docs/guides",
  "docs/research",
];

const ROOT_FILES = ["CONTEXT.md", "CLAUDE.md", "README.md", "SECURITY.md"];

function markdownFilesUnder(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const abs = join(dir, entry.name);
    if (entry.isDirectory()) {
      // The scope table lists families, not depths, so a doc nested inside one still counts (ADR-0156 §1).
      files.push(...markdownFilesUnder(abs));
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      files.push(abs);
    }
  }
  return files;
}

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

export function isInScope(repoRoot, absPath) {
  const rel = relative(repoRoot, absPath).replace(/\\/g, "/");
  if (rel === "" || rel.startsWith("../")) return false;
  if (ROOT_FILES.includes(rel)) return true;
  if (!rel.endsWith(".md")) return false;
  return FAMILY_DIRS.some((dir) => rel === dir || rel.startsWith(`${dir}/`));
}
