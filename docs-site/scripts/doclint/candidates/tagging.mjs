/*
 * Shared sentence tagging for the candidate detectors (#824).
 *
 * The two #824 candidates (one-instruction-per-sentence, no-ellipsis) both reason about a
 * tagged sentence, the same way the shipped simple-tenses rule does. This helper holds the
 * two operations they share, so a candidate file carries only its own detection logic:
 *
 *   1. splitSentences() — a block value -> its sentences, the naive boundary #792 used.
 *   2. tagSentence()    — re-exported from simple-tenses, the direct `pos` lexer/tagger path
 *                         (SPEC §4.3 option 1, the tokenizer mitigation). The candidates reuse
 *                         it so they never touch the retext stack that duplicates words.
 *
 * These files are CANDIDATES, not enabled rules (SPEC §2.5). They are the #824 attempt: the
 * effort builds each detector, measures it on the real corpus (candidates/measure.mjs, the
 * #792 method), then enables or defers it. Nothing here is in the RULES set, so the doclint
 * tool never runs a candidate. See the deferral record in SPEC §7 and the #824 ticket.
 */
import { lineAtValueOffset } from "../engine.mjs";

export { tagSentence } from "../rules/simple-tenses.mjs";

/*
 * The curated imperative-verb list, shared by both candidates. Each entry is a base-form verb
 * that commonly heads an instruction in these docs. The `pos` Brill tagger mistags a
 * sentence-initial capitalized verb as a noun or an adjective ("Run" -> NNP, "Click" -> NN, "Open"
 * -> JJ; see the #824 probe), so neither candidate can key on a VB tag for the leading verb. This
 * list recovers it, the same curated-wordlist route no-phrasal-verbs took when the tagger failed a
 * class. It lives here once because both candidates read it. Lowercase, base form only.
 */
export const IMPERATIVE_VERBS = new Set([
  "add", "build", "call", "check", "click", "close", "copy", "create", "delete", "deploy",
  "disable", "edit", "enable", "enter", "export", "find", "fix", "follow", "give", "import",
  "install", "keep", "leave", "list", "load", "make", "merge", "move", "open", "pass", "pick",
  "pin", "print", "pull", "push", "read", "rebase", "remove", "rename", "replace", "reset",
  "restart", "run", "save", "scope", "seed", "set", "ship", "skip", "split", "start", "stop",
  "tag", "treat", "turn", "update", "use", "verify", "walk", "write",
]);

/** Adverbs a sentence can open with before the imperative verb ("First,"/"Then"/"Now"/"Next"). */
export const LEADING_ADVERBS = new Set(["first", "then", "now", "next", "finally", "also"]);

/**
 * Split a block into sentences with the naive boundary #792 used: a `.`, `!`, or `?` then
 * whitespace then a capital letter or an open paren. It is the exact split simple-tenses runs,
 * copied here so a candidate does not import a private helper. A mis-split changes only the tag
 * context a little, never whether a sentence is read.
 * @param {string} block
 * @returns {string[]}
 */
export function splitSentences(block) {
  return block
    .split(/(?<=[.!?])\s+(?=[A-Z(])/)
    .map((s) => s.trim())
    .filter(Boolean);
}

/**
 * The 1-based source line a substring falls on inside a block, or the block start line when the
 * substring is not found. A candidate points at the sentence it flagged, not the block's first
 * line, because one block can span several soft-wrapped source lines.
 * @param {import("../engine.mjs").ProseBlock} block
 * @param {string} sentence  the sentence text to locate inside block.value.
 * @returns {number}
 */
export function lineOfSentence(block, sentence) {
  const idx = block.value.indexOf(sentence.slice(0, 24));
  if (idx < 0) return block.startLine;
  // The engine owns the offset->line count ("lives here once", engine.mjs), so route through it.
  return lineAtValueOffset(block.value, block.startLine, idx);
}
