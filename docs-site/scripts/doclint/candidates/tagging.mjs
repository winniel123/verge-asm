import { lineAtValueOffset } from "../engine.mjs";

export { tagSentence } from "../rules/simple-tenses.mjs";

// A sentence-initial verb often tags NNP, NN or JJ, so a VB test alone misses it (#824).
export const IMPERATIVE_VERBS = new Set([
  "add", "build", "call", "check", "click", "close", "copy", "create", "delete", "deploy",
  "disable", "edit", "enable", "enter", "export", "find", "fix", "follow", "give", "import",
  "install", "keep", "leave", "list", "load", "make", "merge", "move", "open", "pass", "pick",
  "pin", "print", "pull", "push", "read", "rebase", "remove", "rename", "replace", "reset",
  "restart", "run", "save", "scope", "seed", "set", "ship", "skip", "split", "start", "stop",
  "tag", "treat", "turn", "update", "use", "verify", "walk", "write",
]);

export const LEADING_ADVERBS = new Set(["first", "then", "now", "next", "finally", "also"]);

// A verbatim copy of simple-tenses' unexported split, kept in sync by hand alone.
export function splitSentences(block) {
  return block
    .split(/(?<=[.!?])\s+(?=[A-Z(])/)
    .map((s) => s.trim())
    .filter(Boolean);
}

export function lineOfSentence(block, sentence) {
  const idx = block.value.indexOf(sentence.slice(0, 24));
  if (idx < 0) return block.startLine;
  return lineAtValueOffset(block.value, block.startLine, idx);
}
