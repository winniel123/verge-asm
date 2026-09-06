// A client island bundles this file, so nothing here may import `node:` or `astro:content`.

import { refFromManifest } from "./version-ref.mjs";

export const REPO_URL = "https://github.com/winniel123/verge-asm";

export interface DocsVersion {
  value: string;
  ref: string;
  tag?: string;
}

export function refForDocsVersion(version: string, versions: DocsVersion[]): string {
  return refFromManifest(version, versions);
}

export function repoBlobUrl(ref: string, path: string): string {
  return `${REPO_URL}/blob/${ref}/${path}`;
}
