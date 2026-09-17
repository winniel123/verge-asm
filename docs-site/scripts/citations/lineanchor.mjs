import { visitParents } from "unist-util-visit-parents";
import { parse } from "../doclint/engine.mjs";
import { inOpaque, nearestBlock, refTokensOf, snippetAfter, textOf } from "./extract.mjs";

const SEGMENT = "[A-Za-z0-9_.@+-]";
const DIR_PATH = `[A-Za-z0-9_.@]${SEGMENT}*(?:/${SEGMENT}+)+`;

// A dot-extension opening with a letter separates a slashless file from a ratio (#2120).
const FILE_NAME = `[A-Za-z0-9_@]${SEGMENT}*\\.[A-Za-z][A-Za-z0-9]*`;
const PATH = `(?:${DIR_PATH}|${FILE_NAME})`;

// Either case of the second L, so a malformed range records no truncated token (SPEC §5).
const LINE = String.raw`(?::\d+(?:-\d+)?|#L\d+(?:-[Ll]?\d+)?)`;

// A path-less `:NNN` leans on the path its prose already named, and it drifts the same way (#2120).
const BARE_LINE = String.raw`:\d+(?:-\d+)?`;

// A search, never a whole-span anchor: the retired form also sits inside a longer span.
const LINE_ANCHOR = new RegExp(
  `(?<![A-Za-z0-9_.@/+:\\]-])(${PATH}${LINE}|${BARE_LINE})(?![0-9A-Za-z_.])`,
  "g",
);

// A listen address spells `:8080` as a bare anchor spells `:45`, so only prose parts them (#2185).
const LISTENER = /\blisten(?:ing|ers?|ed|s)?\b/i;

// classify.mjs reads these two the same way, so both arms judge one set of paths (#1450).
const HOSTNAME = /^[a-z0-9-]+(?:\.[a-z0-9-]+)+$/i;
const TLD = /\.[a-z]{2,}$/i;

// A token spells a whole path, a slashless file name, or no path at all (#2120).
export function formOf(token) {
  if (token.startsWith(":")) return "bare";
  return token.includes("/") ? "path" : "file";
}

export function basenameOf(token) {
  const at = token.lastIndexOf("/");
  const name = at < 0 ? token : token.slice(at + 1);
  return name.replace(/(?::\d+(?:-\d+)?|#L\d+(?:-[Ll]?\d+)?)$/, "");
}

function hosted(token) {
  const at = token.indexOf("/");
  // `cold.go` passes this pair too, so a slashless token is judged against the tree instead.
  if (at < 0) return false;
  const first = token.slice(0, at);
  return HOSTNAME.test(first) && TLD.test(first);
}

function tokensIn(value, knownFile) {
  // A URL names no path in this repository, and a registry tag is no line either.
  const found = [...value.matchAll(LINE_ANCHOR)]
    .map((m) => ({ token: m[1], at: m.index }))
    .filter((o) => !hosted(o.token));
  if (knownFile === null) return found;
  // `admin.example.com:443` is a host and a port, and the tree holds no file of that name (#2120).
  return found.filter((o) => formOf(o.token) !== "file" || knownFile(basenameOf(o.token)));
}

function linkTarget(url) {
  if (typeof url !== "string") return null;
  const raw = url.trim().replace(/^<|>$/g, "");
  // A scheme addresses another host, and this gate judges no tree but ours (#1436).
  if (raw === "" || /^[a-z][a-z0-9+.-]*:/i.test(raw)) return null;
  if (raw.startsWith("//")) return null;
  return raw;
}

const SENTENCE_END = /[.!?](?:\s|$)/;

function lastSentence(text) {
  let cut = 0;
  for (const m of text.matchAll(new RegExp(SENTENCE_END.source, "g"))) cut = m.index + m[0].length;
  return text.slice(cut);
}

function firstSentence(text) {
  const m = SENTENCE_END.exec(text);
  return m === null ? text : text.slice(0, m.index + 1);
}

function sentenceParts(block, node) {
  let before = "";
  let after = "";
  let passed = false;
  visitParents(block, (n, ancestors) => {
    if (n === node || ancestors.includes(node)) {
      passed = true;
      return;
    }
    if (n.type !== "text" && n.type !== "inlineCode") return;
    if (passed) after += n.value;
    else before += n.value;
  });
  return { before, after };
}

// A word qualifies the sentence that holds it, so a second sentence may not borrow it (#2185).
function sentenceAround(block, node) {
  const { before, after } = sentenceParts(block, node);
  return lastSentence(before) + textOf(node) + firstSentence(after);
}

// Letters only, so `x.go:4.` still closes the sentence a citation ends.
const ABBREVIATION = /^(?:[A-Za-z]{1,2}(?:\.[A-Za-z]{1,2})*|etc|fig|no|pp|ch|eq|vol|sec|al)$/i;

// A wrong split drops a hold, so this arm splits less often than the carve-out (#2286).
function endsSentence(before, at) {
  const run = before.slice(0, at + 1);
  return !ABBREVIATION.test(run.slice(run.lastIndexOf(" ") + 1, -1));
}

function sentenceIndex(block, node) {
  const { before } = sentenceParts(block, node);
  let count = 0;
  for (const m of before.matchAll(new RegExp(SENTENCE_END.source, "g"))) {
    if (endsSentence(before, m.index)) count++;
  }
  return count;
}

function holdsProse(node) {
  if (node.type === "text") return /[A-Za-z]/.test(node.value);
  return (node.children ?? []).some(holdsProse);
}

// A code-only table cell writes no sentence of its own, so its row is the smallest one (#2185).
function addressNode(chain) {
  let cell = null;
  let row = null;
  for (const a of chain) {
    if (a.type === "tableCell") cell = a;
    if (a.type === "tableRow") row = a;
  }
  return cell === null ? null : holdsProse(cell) ? cell : (row ?? cell);
}

// The span a reader reads the token in, named so two tokens sharing one can be compared (#2286).
export function addressScope(node, ancestors) {
  const cell = addressNode([...ancestors, node]);
  // remark opens a row and its first cell at one offset, so the type and the end part them.
  if (cell !== null) {
    return `${cell.type}@${cell.position?.start?.offset ?? 0}:${cell.position?.end?.offset ?? 0}`;
  }
  const block = nearestBlock(ancestors);
  if (block === null || block.type === "root") return null;
  return `block@${block.position?.start?.offset ?? 0}#${sentenceIndex(block, node)}`;
}

function addressText(node, ancestors) {
  const cell = addressNode([...ancestors, node]);
  if (cell !== null) return textOf(cell);
  const block = nearestBlock(ancestors);
  // The root is the whole document, and a word anywhere in it qualifies nothing here.
  if (block === null || block.type === "root") return "";
  return sentenceAround(block, node);
}

// One pattern, so a rewrite can never touch a token this scan did not find (#1975).
export function lineAnchorPattern() {
  return new RegExp(LINE_ANCHOR.source, LINE_ANCHOR.flags);
}

export function scanLineAnchors(markdown, options) {
  return scanLineAnchorsFromTree(parse(markdown), options);
}

// The carve-out is off where the base tree is fixed and a ref token can pin nothing (SPEC §6).
export function scanLineAnchorsFromTree(tree, { refPin = true, knownFile = null } = {}) {
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
    let tokens = tokensIn(value, knownFile);
    if (tokens.some((o) => formOf(o.token) === "bare") && LISTENER.test(addressText(node, ancestors))) {
      tokens = tokens.filter((o) => formOf(o.token) !== "bare");
    }
    if (tokens.length === 0) return;
    // A line pinned to a named ref cannot drift, so it is the one carve-out (SPEC §5).
    if (refPin && refsByBlock.has(nearestBlock(ancestors))) return;
    for (const { token, at } of tokens) {
      // The sweep rewrites inside the node this scan read, so no second scanner drifts (#1975).
      out.push({
        token,
        form: formOf(token),
        kind,
        line: node.position?.start?.line ?? 1,
        start: node.position?.start?.offset,
        end: node.position?.end?.offset,
        // The value, not the source span: a link label spells a token the target never did (#2249).
        text: value,
        at,
        scope: addressScope(node, ancestors),
        // A converted token turns the next code span into a snippet claim (SPEC §3.4, #1975).
        snippet: snippetAfter(node, ancestors),
      });
    }
  });

  return out;
}
