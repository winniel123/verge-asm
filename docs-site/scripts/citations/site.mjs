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
  const out = [];
  visitParents(tree, (node, ancestors) => {
    if (node.type !== "listItem") return;
    if (inOpaque([...ancestors, node])) return;
    const paragraph = (node.children ?? []).find((c) => c.type === "paragraph");
    if (!paragraph) return;
    if (!LABEL.test(leadingLabel(paragraph))) return;
    out.push(node);
  });
  return out;
}

// One list item is one root, so no neighbouring field lends a Site its prose or its ref.
export function judgeSite(env, repoRoot, body, where) {
  const items = siteItems(parse(body));
  const refused = [];
  const results = [];
  for (const item of items) {
    for (const a of scanLineAnchorsFromTree(item)) refused.push({ ...a, file: where });
    for (const r of classify(env, PSEUDO_DOC, extractCitationsFromTree(item))) {
      results.push({ ...r, file: where });
    }
  }
  const { anchored, verified, broken, unresolved, fatal } = armB(repoRoot, results);
  return {
    fields: items.length,
    refused,
    dead: results.filter((r) => r.status === "dead"),
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

// A re-run replays a stale payload, so a body edit could never clear the gate (#1974).
export async function fetchPullBody({ repo, number, token, fetchImpl = fetch }) {
  const headers = { Accept: "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28" };
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetchImpl(`${API}/repos/${repo}/pulls/${number}`, { headers });
  if (!res.ok) throw new Error(`GET /repos/${repo}/pulls/${number}: HTTP ${res.status}`);
  const pr = await res.json();
  return pr.body ?? "";
}

export function pullContext(env = process.env, readEvent) {
  const { GITHUB_EVENT_PATH: eventPath, GITHUB_REPOSITORY: repo } = env;
  if (!eventPath || !repo) return null;
  let event;
  try {
    event = JSON.parse(readEvent(eventPath));
  } catch {
    return null;
  }
  const number = event.pull_request?.number;
  return number ? { repo, number } : null;
}
