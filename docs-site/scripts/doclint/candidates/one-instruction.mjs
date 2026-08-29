/*
 * CANDIDATE: one-instruction-per-sentence (SPEC §2.5 — prove before enable, #824).
 *
 * Status: a candidate detector, NOT an enabled rule. It is not in the RULES set, so the
 * doclint tool never runs it. See the measured result and the enable-or-defer decision in
 * the #824 ticket and SPEC §7.
 *
 * The style standard §4.2 asks a writer to keep one instruction per sentence. A sentence with
 * two instructions ("Run the build and deploy the result") should split into two. Ticket Q2
 * classified this as a warning, not an error, because the standard §4.2 says some second
 * instructions need a human to see them. So the tool would flag the form, and a human would
 * decide. That judgment residue is why the candidate targets a warning, if precision holds.
 *
 * No known detector (SPEC §2.5). This is the best-effort attempt. A "second instruction" is a
 * second imperative clause. English marks an imperative only by position (a bare base-form verb
 * heading the clause), so there is no tag to key on directly. The detector uses two signals:
 *
 *   1. The sentence is imperative. The first content word (past a leading adverb like "First,"
 *      or "Then") is a base-form verb. The `pos` Brill tagger mistags a sentence-initial
 *      capitalized verb as a noun or an adjective (it tags "Run" as NNP, "Click" as NN, "Open"
 *      as JJ — see the #824 probe), so a raw VB test misses most imperatives. The detector adds
 *      a curated imperative-verb list (IMPERATIVE_VERBS) to recover the mistagged leading verb,
 *      the same curated-list route no-phrasal-verbs took when the tagger failed a class.
 *   2. A coordinator ("and", "or", or "then") joins a second imperative verb. The token straight
 *      after the coordinator (past a coordinator chain like "and then") is a base-form verb (VB)
 *      or a listed imperative verb. Mid-sentence verbs tag VB reliably (the tagger's noun bias is
 *      sentence-initial only), so the second verb needs no list, but the list still helps.
 *
 * The residue this cannot remove: a coordinator also joins two objects ("Read the spec and the
 * standard") or a verb and a non-instruction clause ("Save the file and you are done"). The
 * detector requires a verb straight after the coordinator to cut the object case, but the clause
 * case and every tagger mis-tag stay. The measurement (candidates/measure.mjs) scores whether
 * the residue is small enough to ship as a warning.
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

/** The coordinators that can join a second instruction. "then" tags RB, "and"/"or" tag CC. */
const COORDINATORS = new Set(["and", "or", "then"]);

/** The Penn Treebank tag for a base-form verb. A mid-sentence second verb tags this reliably. */
const BASE_VERB = "VB";

/**
 * Whether a tagged token reads as an imperative verb. A base-form VB tag qualifies. A listed
 * imperative verb qualifies whatever its tag, because the tagger mistags the sentence-initial
 * one. This is the shared test for both the leading verb and the verb after a coordinator.
 * @param {[string, string]} token  a [word, Penn tag] pair.
 * @returns {boolean}
 */
function isImperativeVerb([word, tag]) {
  return tag === BASE_VERB || IMPERATIVE_VERBS.has(word.toLowerCase());
}

/**
 * The index of the first content token, past any leading adverb and any punctuation. "First, run
 * the build" starts its instruction at "run", not "First" and not the comma the `pos` lexer emits
 * as its own token.
 * @param {[string, string][]} tagged
 * @returns {number}
 */
function firstContentIndex(tagged) {
  let i = 0;
  while (
    i < tagged.length &&
    (LEADING_ADVERBS.has(tagged[i][0].toLowerCase()) || !/[\w]/.test(tagged[i][0]))
  ) {
    i++;
  }
  return i;
}

/**
 * Whether a tagged sentence carries two coordinated instructions. The sentence must open with an
 * imperative verb (the first content token), and after a coordinator a second imperative verb
 * must appear within a short window. Returns true on the first such pair.
 * @param {[string, string][]} tagged
 * @returns {boolean}
 */
function hasTwoInstructions(tagged) {
  const start = firstContentIndex(tagged);
  if (start >= tagged.length || !isImperativeVerb(tagged[start])) return false;

  for (let i = start + 1; i < tagged.length; i++) {
    if (!COORDINATORS.has(tagged[i][0].toLowerCase())) continue;
    // Step past a coordinator chain like "and then", then test the first token after it. A verb
    // there marks a second imperative clause. A determiner or a noun there marks a coordinated
    // object instead ("Read the spec and the standard"), so the sentence is one instruction.
    let j = i + 1;
    while (j < tagged.length && COORDINATORS.has(tagged[j][0].toLowerCase())) j++;
    if (j < tagged.length && isImperativeVerb(tagged[j])) return true;
  }
  return false;
}

/** @type {import("../engine.mjs").Rule} */
export const oneInstruction = {
  name: "one-instruction-per-sentence",
  severity: "warning",
  check(_proseNodes, ctx) {
    const violations = [];
    for (const block of ctx.blocks) {
      for (const sentence of splitSentences(block.value)) {
        if (!hasTwoInstructions(tagSentence(sentence))) continue;
        violations.push({
          line: lineOfSentence(block, sentence),
          message:
            "a sentence with two instructions — keep one instruction per sentence, or confirm the split is not needed",
        });
      }
    }
    return violations;
  },
  fixtures: {
    // Must-flag: a real two-instruction sentence the detector must catch (SPEC §6). The set
    // covers each coordinator and the leading-adverb opener.
    mustFlag: [
      "Run the build and deploy the result.", // and, leading verb mistagged NNP
      "Open the file then save it.", // then, leading verb mistagged JJ
      "Stop the server and restart it.", // and, both verbs tag VB
      "Save the changes and close the editor.", // and, two full clauses
      "First, install the plugin then enable it.", // leading adverb opener
    ],
    // Must-not-flag: valid one-instruction prose the detector must leave alone (SPEC §6). The
    // set covers a coordinated object (not a second verb), a single imperative with a purpose
    // clause, a declarative sentence (a subject, not an imperative), and non-prose regions.
    mustNotFlag: [
      "Read the spec and the standard before you start.", // "and" joins two objects
      "Click the button to submit the form.", // one instruction, a purpose clause
      "The tool parses the file and prints the result.", // declarative, a subject heads it
      "Run `build && deploy` in the shell.", // inline code span (non-prose)
      "> Run the build and deploy the result.", // blockquote (frozen source, non-prose)
    ],
  },
};
