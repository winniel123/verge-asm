import { visitParents } from "unist-util-visit-parents";
import { parse } from "../doclint/engine.mjs";
import { inOpaque, nearestBlock, refTokensOf } from "./extract.mjs";

const SEGMENT = "[A-Za-z0-9_.@+-]";
const PATH = `[A-Za-z0-9_.@]${SEGMENT}*(?:/${SEGMENT}+)+`;

// Either case of the second L, so a malformed range records no truncated token (SPEC §5).
const LINE = String.raw`(?::\d+(?:-\d+)?|#L\d+(?:-[Ll]?\d+)?)`;

// A search, never a whole-span anchor: the retired form also sits inside a longer span.
const LINE_ANCHOR = new RegExp(
  `(?<![A-Za-z0-9_.@/+-])(${PATH}${LINE})(?![0-9A-Za-z_.])`,
  "g",
);

// classify.mjs reads these two the same way, so both arms judge one set of paths (#1450).
const HOSTNAME = /^[a-z0-9-]+(?:\.[a-z0-9-]+)+$/i;
const TLD = /\.[a-z]{2,}$/i;

function hosted(token) {
  const first = token.slice(0, token.indexOf("/"));
  return HOSTNAME.test(first) && TLD.test(first);
}

function tokensIn(value) {
  // A URL names no path in this repository, and a registry tag is no line either.
  return [...value.matchAll(LINE_ANCHOR)].map((m) => m[1]).filter((t) => !hosted(t));
}

function linkTarget(url) {
  if (typeof url !== "string") return null;
  const raw = url.trim().replace(/^<|>$/g, "");
  // A scheme addresses another host, and this gate judges no tree but ours (#1436).
  if (raw === "" || /^[a-z][a-z0-9+.-]*:/i.test(raw)) return null;
  if (raw.startsWith("//")) return null;
  return raw;
}

export function scanLineAnchors(markdown) {
  return scanLineAnchorsFromTree(parse(markdown));
}

// The carve-out is off where the base tree is fixed and a ref token can pin nothing (SPEC §6).
export function scanLineAnchorsFromTree(tree, { refPin = true } = {}) {
  // Arm A reads one on-ref implementation rather than a second copy (SPEC §5).
  const { refsByBlock } = refTokensOf(tree);
  const out = [];

  visitParents(tree, (node, ancestors) => {
    if (inOpaque([...ancestors, node])) return;
    let value = null;
    let kind = null;
    if (node.type === "inlineCode") {
      value = node.value;
      kind = "code";
    } else if (node.type === "link" || node.type === "definition" || node.type === "image") {
      // An image spells its target exactly as a link does, so one arm reads both.
      value = linkTarget(node.url);
      kind = "link";
    }
    if (value == null) return;
    const tokens = tokensIn(value);
    if (tokens.length === 0) return;
    // A line pinned to a named ref cannot drift, so it is the one carve-out (SPEC §5).
    if (refPin && refsByBlock.has(nearestBlock(ancestors))) return;
    for (const token of tokens) {
      out.push({ token, kind, line: node.position?.start?.line ?? 1 });
    }
  });

  return out;
}
