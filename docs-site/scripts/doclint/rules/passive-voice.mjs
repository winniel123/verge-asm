/*
 * Rule: passive-voice (SPEC §2.5, candidate warning — proven before enabled).
 *
 * The style standard §4.2 lists passive voice as a review prompt. Ticket Q2 classified it
 * as a warning, not an error, because a human keeps a passive form that reads better. So the
 * tool flags the form, and a human decides. The tool never fails a build on it.
 *
 * A candidate, not a born-enabled rule (SPEC §2.5). #791 and #792 tested the five §4.1
 * mechanical rules, not the §4.2 review prompts, so no measured precision figure existed for
 * this one. This ticket (#823) built the detector, measured it on the fixture corpus (SPEC §6)
 * and the real in-scope docs, then enabled it once the precision held. The fixture corpus test
 * that runs on every RULES member is the standing gate: the rule cannot ship if it fails it.
 *
 * Method (SPEC §2.5). The curated "be" plus participle wordlist route the retext-passive plugin
 * uses. The participle half is the maintained repo asset passive-participles.json. The rule
 * anchors on a participle from that list, then looks back a short window for a "be" form
 * (am/is/are/was/were/be/been/being). So "was completed" and "is generated" flag, but the
 * active "the worker completed the job" does not, because no "be" form precedes the participle.
 *
 * The tokenizer caveat (SPEC §4.3) does not bite here. That caveat is about the retext prose
 * stack duplicating words ("a b c" -> "a b b c c"). This rule never uses retext or a
 * parse-latin/parse-english tokenizer. It runs a regex over the prose the engine already
 * extracted, never over a fresh token stream, so there is no tokenizer to duplicate a word.
 * no-phrasal-verbs matches its wordlist the same regex-over-extracted-prose way. So the #820
 * mitigation is not needed here.
 *
 * The rule reads a block (ctx.blocks), not the loose text nodes, because the "be" form and the
 * participle can sit on either side of an inline span (emphasis, a link, an inline code span).
 * The block joins those fragments, so the pair still reaches one scan. The engine already
 * dropped every SPEC §3 non-prose region, so a passive form inside a code block, a table cell,
 * or a blockquote never becomes a block here.
 *
 * Precision (SPEC §2.5). Moderate by design. The list catches its entries. It misses a
 * participle not on the list. It may over-flag a stative use of a listed word ("the value is
 * defined" as a state, not an action). A warning tolerates that residue, and the inline disable
 * directive (SPEC §6, #821) is the escape hatch for an unavoidable false positive.
 *
 * Measured (this ticket #823, the #792 method). The detector ran over the in-scope docs and
 * flagged 1412 forms across 177 files. A ~98-flag two-sample scoring found about 99% precision:
 * one false positive, an attributive participle ("are measured facts", where "measured" modifies
 * the noun). That one class needs a POS tag on the next word, which this route does not carry, so
 * it is the documented residue. Volume is high, the same shape #792 recorded for simple-tenses
 * (634 flags, ~100%), and many flags are legitimate passives a human may keep. A hard error would
 * be wrong. So the precision held, and the rule ships as a warning.
 */
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { escapeRegex, lineAtValueOffset } from "../engine.mjs";

const LIST_PATH = join(dirname(fileURLToPath(import.meta.url)), "..", "passive-participles.json");

/**
 * The "be" forms that mark a passive construction. A participle from the list becomes a flag
 * only when one of these sits a short window before it. This is the "be" half of the SPEC §2.5
 * "a form of 'be' plus a participle list" route.
 */
const BE_FORMS = new Set(["am", "is", "are", "was", "were", "be", "been", "being"]);

/**
 * The words allowed between the "be" form and the participle. A passive form takes an adverb or
 * a negation in the middle ("was recently changed", "is not measured"). The back-walk skips
 * these, up to WINDOW of them, then expects the "be" form. Any `-ly` adverb counts too (see
 * isFiller), so the set holds only the common non-`-ly` fillers.
 */
const FILLERS = new Set([
  "not",
  "never",
  "also",
  "already",
  "then",
  "now",
  "just",
  "still",
  "yet",
  "once",
  "only",
  "so",
  "thus",
  "again",
  "even",
]);

/** The most fillers allowed between the "be" form and the participle. */
const WINDOW = 3;

/**
 * Whether a word may sit between the "be" form and the participle without breaking the passive
 * form. A curated filler or any `-ly` adverb qualifies. A plain word ("a", "the", a noun) does
 * not, so "is a completed draft" and "is the changed file" never flag.
 * @param {string} word  the word, lowercase.
 * @returns {boolean}
 */
function isFiller(word) {
  return FILLERS.has(word) || word.endsWith("ly");
}

/** The participle set, lowercase, loaded once from the maintained wordlist asset. */
const PARTICIPLES = loadParticiples();

/** @returns {Set<string>} */
function loadParticiples() {
  const { participles } = JSON.parse(readFileSync(LIST_PATH, "utf8"));
  return new Set(participles.map((p) => p.toLowerCase()));
}

/**
 * One global, case-insensitive matcher for every participle, as whole words. Longest first, so
 * the alternation is predictable; order never changes whether a match exists. The `\b` anchors
 * stop a partial-word match ("recompleted" does not match "completed").
 */
const PARTICIPLE_RE = new RegExp(
  `\\b(?:${[...PARTICIPLES]
    .sort((a, b) => b.length - a.length)
    .map(escapeRegex)
    .join("|")})\\b`,
  "gi",
);

/** A word plus its offset inside a string. */
/** @typedef {{ word: string, index: number }} Token */

/**
 * The words of a string with their offsets, in order. A word is a run of word characters (a
 * letter, a digit, or an underscore), an apostrophe, or a hyphen. Used to walk back from a
 * participle over the words before it.
 * @param {string} text
 * @returns {Token[]}
 */
function words(text) {
  const tokens = [];
  for (const m of text.matchAll(/[\w'-]+/g)) tokens.push({ word: m[0], index: m.index });
  return tokens;
}

/**
 * Whether the words before a participle form a passive construction, and where the "be" form
 * sits. Walk back from the nearest word: skip up to WINDOW fillers, then expect a "be" form.
 * The first non-filler, non-"be" word ends the walk with no passive (an active clause, a noun,
 * or an article before the participle).
 * @param {Token[]} before  the words before the participle, in document order.
 * @returns {Token | null}  the "be" token when the form is passive, else null.
 */
function beBefore(before) {
  let fillers = 0;
  for (let k = before.length - 1; k >= 0; k--) {
    const w = before[k].word.toLowerCase();
    if (BE_FORMS.has(w)) return before[k];
    if (fillers < WINDOW && isFiller(w)) {
      fillers++;
      continue;
    }
    return null; // a non-filler, non-"be" word: not a passive form
  }
  return null;
}

/** @type {import("../engine.mjs").Rule} */
export const passiveVoice = {
  name: "passive-voice",
  severity: "warning",
  check(_proseNodes, ctx) {
    const violations = [];
    for (const block of ctx.blocks) {
      const value = block.value;
      PARTICIPLE_RE.lastIndex = 0;
      let m;
      while ((m = PARTICIPLE_RE.exec(value)) !== null) {
        const before = words(value.slice(0, m.index));
        const be = beBefore(before);
        if (!be) continue;
        const span = value.slice(be.index, m.index + m[0].length).replace(/\s+/g, " ").trim();
        violations.push({
          line: lineAtValueOffset(value, block.startLine, be.index),
          message: `a passive verb form "${span}" — prefer the active voice, or confirm the passive is intentional`,
        });
      }
    }
    return violations;
  },
  fixtures: {
    // Each must-flag snippet holds a real passive form the rule must catch (SPEC §6). The set
    // covers the plain "be + participle" pair, three "be" forms, the "have been" auxiliary
    // chain, an adverb filler, and a negation filler.
    mustFlag: [
      "The job was completed by the worker.", // was + participle
      "The tokens are generated at build time.", // are + participle
      "Each rule is measured against the fixture corpus.", // is + participle
      "The docs have been rewritten this week.", // been + participle (auxiliary chain)
      "The report was recently generated by the nightly job.", // adverb (-ly) filler
      "The stale file is not removed until the next run.", // negation filler
    ],
    // Each must-not-flag snippet holds valid prose the rule must leave alone (SPEC §6). The set
    // proves an active clause, a "be" plus a plain adjective (not on the list), a "be" plus an
    // article before a listed word (not adjacent), and every SPEC §3 non-prose region.
    mustNotFlag: [
      "The worker completed the job on time.", // active voice, no "be" before the participle
      "The build is ready for a review.", // "be" + adjective, not a listed participle
      "The report is a completed draft.", // "be" + article + participle: "a" is not a filler
      "The team measured the cost last week.", // active use of a listed participle stem
      "Run `is completed` in the shell to check it.", // inline code span (non-prose)
      "```sh\necho was completed the job\n```", // fenced code block (non-prose)
      "> The vendor said the work was completed for us.", // blockquote (frozen source, non-prose)
      "| Step | Note |\n| --- | --- |\n| ship | was completed |", // table cell (non-prose)
    ],
  },
};
