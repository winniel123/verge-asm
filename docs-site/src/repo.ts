// A client island bundles this file, so nothing here may import `node:` or `astro:content`.

export const REPO_URL = "https://github.com/winniel123/verge-asm";

// A `v*` tag makes source-resolution.ts's refForVersion diverge from this mirror (ADR-0115).
export function refForDocsVersion(version: string): string {
  return version === "main" || version === "latest" ? "main" : version;
}

export function repoBlobUrl(ref: string, path: string): string {
  return `${REPO_URL}/blob/${ref}/${path}`;
}
