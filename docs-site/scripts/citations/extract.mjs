import { visitParents } from "unist-util-visit-parents";
import { parse } from "../doclint/engine.mjs";

// A fenced block, front matter and a raw html block are sample text, not a citation (#1450).
const OPAQUE = new Set(["code", "yaml", "html"]);

const BLOCK = new Set([
  "paragraph",
  "heading",
  "tableCell",
  "listItem",
  "blockquote",
  "definition",
  "root",
]);

// A path inside an ADR-0058 withdrawal marker is a historical record, not a live claim (#1450).
const WITHDRAWAL = /\bWITHDRAWN\b|\bWITHDRAWAL\b|\bSUPERSEDED\b/;

const REF_BRANCH = /^[a-z][a-z0-9]*\/[A-Za-z0-9._-]+$/;
const REF_SHA = /^[0-9a-f]{7,40}$/;

// The digit keeps this off `defaced`, seven hex characters and also a word (#2257).
const COMMIT_HASH = /^(?=[0-9a-f]*\d)[0-9a-f]{7,40}$/;

// One spelling, so the corroborator and Arm A's pin cannot read one span two ways (#2284).
export function spellsCommitHash(value) {
  return COMMIT_HASH.test(value);
}

export function textOf(node) {
  if (node.value != null && typeof node.value === "string") return node.value;
  if (!node.children) return "";
  return node.children.map(textOf).join(" ");
}

function lineOf(node) {
  return node.position?.start?.line ?? 1;
}

export function nearestBlock(ancestors) {
  for (let i = ancestors.length - 1; i >= 0; i--) {
    if (BLOCK.has(ancestors[i].type)) return ancestors[i];
  }
  return ancestors[0] ?? null;
}

export function inOpaque(ancestors) {
  return ancestors.some((a) => OPAQUE.has(a.type));
}

function withdrawn(node, ancestors) {
  if (ancestors.some((a) => a.type === "delete")) return true;
  for (const a of ancestors) {
    if (a.type === "blockquote" && WITHDRAWAL.test(textOf(a))) return true;
  }
  return WITHDRAWAL.test(textOf(node)) && node.type !== "inlineCode";
}

// Only the leading word separates a ref from a branch-shaped path such as apache/kafka (#1436).
const REF_LEAD = /\b(branch|commit|ref|revision|tag)e?[sd]?\s*[:—-]?\s*$/i;

// A branch or commit named beside a path moves the claim off the working tree (#1436).
export function refTokensOf(tree) {
  const byBlock = new Map();
  const prose = new Map();
  visitParents(tree, (node, ancestors) => {
    if (inOpaque(ancestors) || OPAQUE.has(node.type)) return;
    if (node.type !== "text" && node.type !== "inlineCode") return;
    const block = nearestBlock(ancestors);
    if (node.type === "text") {
      prose.set(block, (prose.get(block) ?? "") + node.value);
      return;
    }
    const value = node.value.trim();
    if (!REF_BRANCH.test(value) && !REF_SHA.test(value)) return;
    if (!REF_LEAD.test(prose.get(block) ?? "")) return;
    if (!byBlock.has(block)) byBlock.set(block, []);
    byBlock.get(block).push({ ref: value, line: lineOf(node) });
  });
  return { refsByBlock: byBlock, proseByBlock: prose };
}

function fromLink(url) {
  if (typeof url !== "string") return null;
  let raw = url.trim();
  if (raw === "") return null;
  if (/^[a-z][a-z0-9+.-]*:/i.test(raw)) return null;
  if (raw.startsWith("#") || raw.startsWith("//")) return null;
  raw = raw.replace(/^<|>$/g, "");
  return raw;
}

const ANCHOR_CHARS = "[A-Za-z0-9_.@+/-]";

// One class, so the code span and the link fragment can never spell an anchor apart (#1968).
export const ANCHOR = new RegExp(`^${ANCHOR_CHARS}+$`);

// Bare prose spells a path far less exactly, and #1450 measured that population 60% wrong.
const CODE_PATH = new RegExp(
  `^[A-Za-z0-9_.@][A-Za-z0-9_.@+-]*(?:/[A-Za-z0-9_.@+-]+)+(?:/|#${ANCHOR_CHARS}+)?$`,
);

// One class for the extractor and the guard, so a sibling citation span is not a name (#2007).
export function spellsPath(value) {
  return fromCode(value) !== null;
}

function fromCode(value) {
  const raw = value.trim();
  if (raw === "" || /\s/.test(raw)) return null;
  // The anchor spends this gate's one safe character on `#`, and a colon stays out (#1968, #1969).
  if (!CODE_PATH.test(raw)) return null;
  return raw;
}

export function extractCitations(markdown) {
  return extractCitationsFromTree(parse(markdown));
}

// A snippet is the code span immediately after an anchor span (SPEC §3.4).
export function snippetAfter(node, ancestors) {
  const siblings = ancestors[ancestors.length - 1]?.children ?? [];
  const at = siblings.indexOf(node);
  if (at < 0) return null;
  for (const next of siblings.slice(at + 1)) {
    if (next.type === "text" && next.value.trim() === "") continue;
    if (next.type !== "inlineCode") return null;
    // Two citations in one sentence are a common shape, and neither disambiguates the other.
    return fromCode(next.value) === null ? next.value : null;
  }
  return null;
}

export function extractCitationsFromTree(tree) {
  const { refsByBlock, proseByBlock } = refTokensOf(tree);
  const out = [];

  const push = (raw, node, ancestors, kind) => {
    if (raw == null) return;
    const block = nearestBlock(ancestors);
    out.push({
      raw,
      kind,
      line: lineOf(node),
      withdrawn: withdrawn(node, ancestors),
      prose: proseByBlock.get(block) ?? "",
      refs: refsByBlock.get(block) ?? [],
      snippet: snippetAfter(node, ancestors),
    });
  };

  visitParents(tree, (node, ancestors) => {
    if (inOpaque(ancestors) || OPAQUE.has(node.type)) return;
    if (node.type === "link" || node.type === "definition") {
      push(fromLink(node.url), node, ancestors, "link");
    } else if (node.type === "image") {
      push(fromLink(node.url), node, ancestors, "image");
    } else if (node.type === "inlineCode") {
      push(fromCode(node.value), node, ancestors, "code");
    }
  });

  return out;
}
