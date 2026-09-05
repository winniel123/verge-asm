import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const HERE = dirname(fileURLToPath(import.meta.url));

export function loadExemptions(file = join(HERE, "exemptions.json")) {
  const parsed = JSON.parse(readFileSync(file, "utf8"));
  const entries = parsed.entries ?? [];
  for (const e of entries) {
    if (!e.file || !e.path || !e.reason) {
      throw new Error(`citations: exemption needs file, path and reason: ${JSON.stringify(e)}`);
    }
  }
  return entries;
}

// A trailing slash makes an entry cover a family, which is what a whole-directory exemption needs.
function fileMatches(pattern, docFile) {
  return pattern.endsWith("/") ? docFile.startsWith(pattern) : pattern === docFile;
}

export function exemptionMatcher(entries) {
  return (docFile, path, value) => {
    for (const e of entries) {
      if (!fileMatches(e.file, docFile)) continue;
      if (e.path !== path && e.path !== value) continue;
      return e.reason;
    }
    return null;
  };
}
