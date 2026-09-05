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
