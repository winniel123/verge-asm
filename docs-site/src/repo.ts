/*
 * Repository coordinates for the docs-site — the single source of truth for the
 * canonical GitHub repo URL and the docs-version → git-ref mapping used whenever a
 * guide links out to a repo file (e.g. an ADR blob) or the header links home.
 *
 * Kept dependency-free (no `node:*`, no `astro:content`) ON PURPOSE: it is imported
 * from BOTH server code (DocsLayout.astro) and client islands (render.jsx), so it
 * must be safe to bundle for the browser. source-resolution.ts is the node-only
 * pipeline module and cannot be reached from an island — this is its client mirror
 * for the two facts an island needs.
 */

/** Canonical repository URL (header link + repo cross-ref blob URLs). */
export const REPO_URL = "https://github.com/winniel123/verge-asm";

/**
 * Map a docs manifest version to the git ref its repo cross-refs resolve against.
 * Client-safe mirror of source-resolution.ts's `refForVersion`: `main` and the
 * `latest` alias resolve to `main` (today there are zero release tags, so `latest`
 * tracks `main`); any other value is a release tag used verbatim as the ref.
 */
export function refForDocsVersion(version: string): string {
  return version === "main" || version === "latest" ? "main" : version;
}

/** GitHub blob URL for a repo-relative `path` at `ref` (no leading slash on path). */
export function repoBlobUrl(ref: string, path: string): string {
  return `${REPO_URL}/blob/${ref}/${path}`;
}
