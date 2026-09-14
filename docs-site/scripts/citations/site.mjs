import { visitParents } from "unist-util-visit-parents";
import { parse } from "../doclint/engine.mjs";
import { extractCitationsFromTree, inOpaque } from "./extract.mjs";
import { scanLineAnchorsFromTree } from "./lineanchor.mjs";
import { classify } from "./classify.mjs";
import { armB } from "./armb.mjs";

const API = "https://api.github.com";

const LABEL = /^site\s*:/i;

// A body sits in no directory: a Site path reads against the repo root, and no exemption names it.
const PSEUDO_DOC = "";

function flatten(node) {
  if (typeof node.value === "string") return node.value;
  if (!node.children) return "";
  return node.children.map(flatten).join("");
}

// The label ends at the first code span, because the anchor itself is the first one.
function leadingLabel(paragraph) {
  let text = "";
  for (const child of paragraph.children ?? []) {
    if (child.type === "inlineCode") break;
    text += flatten(child);
  }
  return text.trim();
}

// The reach is the label: a body's Site field alone (SPEC docs/spec/citation-anchors.md §6).
export function siteItems(tree) {
  const out = new Set();
  visitParents(tree, (node, ancestors) => {
    if (node.type !== "paragraph") return;
    if (inOpaque([...ancestors, node])) return;
    if (!LABEL.test(leadingLabel(node))) return;
    // A nested field sits inside an outer root already, and one token earns one error.
    if (ancestors.some((a) => out.has(a))) return;
    // The list item is the root where there is one, so a wrapped value stays inside the field.
    const parent = ancestors[ancestors.length - 1];
    out.add(parent?.type === "listItem" ? parent : node);
  });
  return [...out];
}

// The author writes the whole field, so a prose withdrawal here is self-certification (SPEC §6).
const ABSENT = new Set(["dead", "withdrawn"]);

// One list item is one root, so no neighbouring field lends a Site its prose or its ref.
export function judgeSite(env, repoRoot, body, where) {
  const items = siteItems(parse(body));
  const refused = [];
  const results = [];
  for (const item of items) {
    // A Site resolves against the merge ref alone, so a ref token pins nothing (SPEC §6 rule 2).
    for (const a of scanLineAnchorsFromTree(item, { refPin: false })) {
      refused.push({ ...a, file: where });
    }
    for (const r of classify(env, PSEUDO_DOC, extractCitationsFromTree(item))) {
      results.push({ ...r, file: where });
    }
  }
  const { anchored, verified, broken, unresolved, fatal } = armB(repoRoot, results);
  return {
    fields: items.length,
    refused,
    dead: results.filter((r) => ABSENT.has(r.status)),
    anchored,
    verified,
    broken,
    unresolved,
    fatal,
  };
}

export function formatSiteLineAnchor(a) {
  return `${a.file}:${a.line}  ->  ${a.token}  (a Site field names no line: name the enclosing declaration)`;
}

// A blip on a required check costs a whole cycle, and the Go row caps its own call the same way.
const TIMEOUT_MS = 30_000;
const RETRY_ON = new Set([429, 500, 502, 503, 504]);

// A re-run replays a stale payload, so a body edit could never clear the gate (#1974).
export async function fetchPullBody({ repo, number, token, api = API, fetchImpl = fetch }) {
  const headers = { Accept: "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28" };
  if (token) headers.Authorization = `Bearer ${token}`;
  const where = `/repos/${repo}/pulls/${number}`;
  let last;
  for (let attempt = 0; attempt < 2; attempt++) {
    const res = await fetchImpl(`${api}${where}`, {
      headers,
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
    if (res.ok) return (await res.json()).body ?? "";
    last = res.status;
    if (!RETRY_ON.has(res.status)) break;
  }
  throw new Error(`GET ${where}: HTTP ${last}`);
}

export function apiBase(env = process.env) {
  return env.GITHUB_API_URL || API;
}

// An unreadable payload is a claim this gate should judge and could not (SPEC §7.7).
export function pullContext(env = process.env, readEvent) {
  const { GITHUB_EVENT_PATH: eventPath, GITHUB_REPOSITORY: repo } = env;
  if (!eventPath || !repo) return null;
  const event = JSON.parse(readEvent(eventPath));
  const number = event.pull_request?.number;
  return number ? { repo, number } : null;
}
