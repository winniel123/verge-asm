/*
 * CANDIDATE: no-ellipsis (SPEC §2.5 — prove before enable, #824).
 *
 * Status: a candidate detector, NOT an enabled rule. It is not in the RULES set, so the
 * doclint tool never runs it. See the measured result and the enable-or-defer decision in
 * the #824 ticket and SPEC §7.
 *
 * The style standard §4.2 asks a writer to avoid ellipsis: do not drop a subject, a verb, or an
 * article the reader needs ("Click button" for "Click the button"; "Then returns an error" for
 * "Then it returns an error"). Ticket Q2 classified this as a warning. So the tool would flag a
 * suspected drop, and a human would decide.
 *
 * Higher risk (SPEC §2.5): "detecting a dropped subject, verb, or article needs a parse the
 * `pos` tagger does not support reliably". This file is the best-effort attempt at the most
 * defensible of the three drops, the dropped verb. A grammatical sentence needs one finite verb.
 * So the detector flags a sentence-shaped block that carries no finite-verb tag.
 *
 * The wall this hits (measured in candidates/measure.mjs): the `pos` Brill tagger mistags a
 * sentence-initial imperative verb as a noun ("Run" -> NNP, "Click" -> NN; see the #824 probe).
 * A legitimate one-verb imperative ("Run the build.") then carries no finite-verb tag and reads
 * exactly like a true dropped-verb fragment. An imperative is the most common instruction form
 * in these docs, so a raw no-verb test flags almost every instruction. The detector excludes a
 * sentence that opens with a curated imperative verb (IMPERATIVE_VERBS) to cut that class, but
 * the exclusion is the same tagger-blind guess, and it cannot separate "Click the button." (a
 * good imperative) from "Click button." (a dropped article). The dropped-article and
 * dropped-subject drops are worse still: English drops an article legitimately everywhere (a
 * plural, a mass noun, "at build time", "in scope"), so a dropped-article detector floods. This
 * file does not attempt them; the measurement records why.
 *
 * The tokenizer caveat (SPEC §4.3) does not bite: this reuses the direct `pos` path through
 * candidates/tagging.mjs (simple-tenses' tagSentence), never the retext stack.
 */
import {
  splitSentences,
  tagSentence,
  lineOfSentence,
  IMPERATIVE_VERBS,
  LEADING_ADVERBS,
} from "./tagging.mjs";

/** The finite and non-finite verb tags. Any one of these means the sentence carries a verb. */
const VERB_TAGS = new Set(["VB", "VBD", "VBG", "VBN", "VBP", "VBZ", "MD"]);

/**
 * Whether a sentence opens with a curated imperative verb (past a leading adverb). A true
 * imperative keeps its verb, so it is not an ellipsis even when the tagger mistags that verb.
 * @param {[string, string][]} tagged
 * @returns {boolean}
 */
function opensWithImperative(tagged) {
  let i = 0;
  while (i < tagged.length && LEADING_ADVERBS.has(tagged[i][0].toLowerCase())) i++;
  return i < tagged.length && IMPERATIVE_VERBS.has(tagged[i][0].toLowerCase());
}

/**
 * Whether a sentence reads as a full sentence, not a heading or a caption. It must end with `.`,
 * `!`, or `?` and hold at least three words. A heading has no trailing period, so this drops the
 * heading and caption blocks that would otherwise flood the no-verb test.
 * @param {string} sentence
 * @returns {boolean}
 */
function looksLikeSentence(sentence) {
  if (!/[.!?]$/.test(sentence)) return false;
  return (sentence.match(/[\w'-]+/g) ?? []).length >= 3;
}

/**
 * Whether a tagged sentence carries a suspected dropped verb: it looks like a sentence, it does
 * not open with an imperative verb, and no token carries a verb tag.
 * @param {string} sentence  the raw sentence text.
 * @param {[string, string][]} tagged
 * @returns {boolean}
 */
function looksElided(sentence, tagged) {
  if (!looksLikeSentence(sentence)) return false;
  if (opensWithImperative(tagged)) return false;
  return !tagged.some(([, tag]) => VERB_TAGS.has(tag));
}

/** @type {import("../engine.mjs").Rule} */
export const noEllipsis = {
  name: "no-ellipsis",
  severity: "warning",
  check(_proseNodes, ctx) {
    const violations = [];
    for (const block of ctx.blocks) {
      for (const sentence of splitSentences(block.value)) {
        if (!looksElided(sentence, tagSentence(sentence))) continue;
        violations.push({
          line: lineOfSentence(block, sentence),
          message:
            "a sentence with no verb — a subject or a verb may be dropped, or confirm it is a full sentence",
        });
      }
    }
    return violations;
  },
  fixtures: {
    // Must-flag: a real fragment presented as a sentence, with a dropped verb (SPEC §6). Each one
    // tags with no verb and does not open with an imperative verb.
    mustFlag: [
      "Configuration of the build tokens.", // dropped verb ("is")
      "Then the same result for the tokens.", // dropped verb
      "A short note about the run cost.", // fragment, dropped verb
    ],
    // Must-not-flag: valid prose the detector must leave alone (SPEC §6). A declarative sentence
    // carries a verb; an imperative opens with a listed verb; non-prose never reaches a rule.
    mustNotFlag: [
      "The tool parses the file.", // declarative, carries a verb (VBZ)
      "The worker completed the job.", // declarative, carries a verb (VBN)
      "Run the build before you deploy.", // imperative, opens with a listed verb
      "The docs are rewritten this week.", // declarative, carries a verb
      "> Configuration of the build tokens.", // blockquote (frozen source, non-prose)
    ],
  },
};
