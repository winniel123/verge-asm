/*
 * Per-version search index endpoint — emits `/search/<version>.json` at build time.
 * OWNER: T4 (#354).
 *
 * One static JSON file per manifest version (`listVersions()`), each holding that
 * version's `SearchDoc[]` (see search-index.ts). TopNav fetches the ACTIVE version's
 * file, so the ⌘K palette is always scoped to the version the reader is viewing;
 * switching version switches the file it loads. Static output — plain `fetch()` from
 * the client island, no server runtime.
 */
import type { APIRoute } from "astro";
import { listVersions } from "../../pipeline/source-resolution";
import { buildSearchDocs } from "../../pipeline/search-index";

export function getStaticPaths() {
  return listVersions().map((v) => ({ params: { version: v.value } }));
}

export const GET: APIRoute = async ({ params }) => {
  const docs = await buildSearchDocs(params.version as string);
  return new Response(JSON.stringify(docs), {
    headers: { "content-type": "application/json; charset=utf-8" },
  });
};
