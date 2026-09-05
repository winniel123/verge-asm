import { readdirSync, existsSync, statSync } from "node:fs";
import { join, relative } from "node:path";

// #1407's worst case is in design-system/docs/, and #1450 ruled docs/research out.
const FAMILY_DIRS = [
  "docs/adr",
  "docs/spec",
  "docs/agents",
  "docs/guides",
  "design-system",
  "docs-site",
];

// A build output is not authored text, and node_modules would swamp the walk.
const EXCLUDED_SEGMENTS = new Set(["node_modules", "dist", ".astro"]);

const ROOT_FILES = [
  "CONTEXT.md",
  "CLAUDE.md",
  "README.md",
  "SECURITY.md",
  "CONTRIBUTING.md",
  "CHANGELOG.md",
];

function excluded(repoRoot, abs) {
  const rel = relative(repoRoot, abs).replace(/\\/g, "/");
  return rel.split("/").some((seg) => EXCLUDED_SEGMENTS.has(seg));
}

function markdownFilesUnder(repoRoot, dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const abs = join(dir, entry.name);
    if (excluded(repoRoot, abs)) continue;
    if (entry.isDirectory()) files.push(...markdownFilesUnder(repoRoot, abs));
    else if (entry.isFile() && entry.name.endsWith(".md")) files.push(abs);
  }
  return files;
}

export function inScopeFiles(repoRoot) {
  const files = [];
  for (const dir of FAMILY_DIRS) {
    const abs = join(repoRoot, dir);
    if (existsSync(abs) && statSync(abs).isDirectory()) {
      files.push(...markdownFilesUnder(repoRoot, abs));
    }
  }
  for (const name of ROOT_FILES) {
    const abs = join(repoRoot, name);
    if (existsSync(abs)) files.push(abs);
  }
  return files.sort();
}

export function isInScope(repoRoot, absPath) {
  const rel = relative(repoRoot, absPath).replace(/\\/g, "/");
  if (rel === "" || rel.startsWith("../")) return false;
  if (ROOT_FILES.includes(rel)) return true;
  if (!rel.endsWith(".md")) return false;
  if (excluded(repoRoot, absPath)) return false;
  return FAMILY_DIRS.some((dir) => rel.startsWith(`${dir}/`));
}
