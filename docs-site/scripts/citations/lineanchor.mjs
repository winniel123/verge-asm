import { visitParents } from "unist-util-visit-parents";
import { parse } from "../doclint/engine.mjs";
import { inOpaque, nearestBlock, refTokensOf } from "./extract.mjs";

const SEGMENT = "[A-Za-z0-9_.@+-]";
const PATH = `[A-Za-z0-9_.@]${SEGMENT}*(?:/${SEGMENT}+)+`;

// SPEC docs/spec/citation-anchors.md §5 lists a colon form and GitHub's own `#L` permalink.
const LINE = String.raw`(?::\d+(?:-\d+)?|#L\d+(?:-L?\d+)?)`;

// A search, never a whole-span anchor: the retired form also sits inside a longer span.
// The trailing dot is refused, so an image tag such as ghcr.io/owner/app:1.26 is no anchor.
const LINE_ANCHOR = new RegExp(
  `(?<![A-Za-z0-9_.@/+-])(${PATH}${LINE})(?![0-9A-Za-z_.])`,
  "g",
);

function tokensIn(value) {
  return [...value.matchAll(LINE_ANCHOR)].map((m) => m[1]);
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
  const tree = parse(markdown);
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
    } else if (node.type === "link" || node.type === "definition") {
      value = linkTarget(node.url);
      kind = "link";
    }
    if (value == null) return;
    const tokens = tokensIn(value);
    if (tokens.length === 0) return;
    // A line pinned to a named ref cannot drift, so it is the one carve-out (SPEC §5).
    if (refsByBlock.has(nearestBlock(ancestors))) return;
    for (const token of tokens) {
      out.push({ token, kind, line: node.position?.start?.line ?? 1 });
    }
  });

  return out;
}
