/*
 * Rule: no-phrasal-verbs (SPEC §2.3, error).
 *
 * The style standard §4.1 rule 5 makes a phrasal verb (a verb plus a particle, such as
 * "spin up" or "kick off") in prose a failure. #791 found no POS route: the `pos` tagger
 * does not tag the particle class well. So the detector is a curated wordlist, the same
 * pattern the retext-passive plugin uses. The wordlist is a maintained repo asset in
 * phrasal-verbs.json, seeded from the phrasal verbs the style standard and the ASD-STE100
 * skill name (SPEC §2.3).
 *
 * Method. For each wordlist entry, match a verb form directly followed by its particle,
 * as whole words, case-insensitive. The rule reads the prose text nodes the engine hands
 * over (SPEC §2.1), the same nodes no-semicolons reads, so a phrasal verb inside a code
 * span, a table cell, or a blockquote never reaches it. The engine already dropped every
 * SPEC §3 non-prose region before this rule runs.
 *
 * Precision (SPEC §2.3). Moderate by design. The list catches its entries reliably. It
 * misses a phrasal verb not on the list. It misses a separated form ("spin the job up"),
 * because the matcher reads the adjacent form only. It may over-flag a literal use ("set
 * up the ladder" against "set up the server"), because the two read the same. The fixture
 * corpus (SPEC §6) sets the acceptance bar for the seeded list. The deferred inline
 * disable directive (SPEC §6, #821) is the escape hatch for an unavoidable over-flag.
 */
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { lineAtOffset, escapeRegex } from "../engine.mjs";

const WORDLIST_PATH = join(dirname(fileURLToPath(import.meta.url)), "..", "phrasal-verbs.json");

/**
 * The regular inflected forms of a verb lemma: the base, the third-person `-s`, the `-ing`
 * form, and the `-ed` form. The `-s` and `-ed` rules follow ordinary English spelling. The
 * `-ing` rule does not double a final consonant, so a doubling verb ("spin" -> "spinning")
 * lists that form in `irregular` instead. A naive form the generator produces for such a
 * verb ("spining", "spinned") is a dead non-word. It never appears in prose, so it can
 * never cause a false match. The correct irregular form does the real work.
 * @param {string} verb  the verb lemma, lowercase.
 * @returns {string[]}
 */
function regularForms(verb) {
  const forms = [verb];

  // Third-person singular. A sibilant ending takes -es. A consonant + y takes -ies.
  if (/(s|x|z|ch|sh)$/.test(verb)) forms.push(verb + "es");
  else if (/[^aeiou]y$/.test(verb)) forms.push(verb.slice(0, -1) + "ies");
  else forms.push(verb + "s");

  // Present participle. No consonant doubling here (see the doc comment above).
  forms.push(verb + "ing");

  // Past / past participle. A final e takes -d. A consonant + y takes -ied. Else -ed.
  if (verb.endsWith("e")) forms.push(verb + "d");
  else if (/[^aeiou]y$/.test(verb)) forms.push(verb.slice(0, -1) + "ied");
  else forms.push(verb + "ed");

  return forms;
}

/**
 * One compiled matcher per wordlist entry: a global, case-insensitive regex that matches
 * any verb form, then whitespace, then the particle, all as whole words. The leading and
 * trailing `\b` stop a partial-word match: "respin up" does not match "spin up", and "set
 * upstream" does not match "set up", because "upstream" has no word boundary after "up".
 * @typedef {Object} Matcher
 * @property {RegExp} regex
 * @property {string} label  the phrasal verb as `<lemma> <particle>`, for the message.
 */

/** @returns {Matcher[]} */
function buildMatchers() {
  const raw = readFileSync(WORDLIST_PATH, "utf8");
  const { entries } = JSON.parse(raw);
  return entries.map((entry) => {
    const forms = [...regularForms(entry.verb), ...(entry.irregular ?? [])];
    // Longest first, so the alternation prefers the longest matching form. Order does not
    // change whether a match exists, but it keeps the engine's work predictable.
    const alternation = [...new Set(forms)].sort((a, b) => b.length - a.length).map(escapeRegex);
    const particle = escapeRegex(entry.particle);
    return {
      regex: new RegExp(`\\b(?:${alternation.join("|")})\\s+${particle}\\b`, "gi"),
      label: `${entry.verb} ${entry.particle}`,
    };
  });
}

const MATCHERS = buildMatchers();

/** @type {import("../engine.mjs").Rule} */
export const noPhrasalVerbs = {
  name: "no-phrasal-verbs",
  severity: "error",
  check(proseNodes) {
    const violations = [];
    for (const node of proseNodes) {
      const value = node.value;
      for (const { regex, label } of MATCHERS) {
        regex.lastIndex = 0;
        let m;
        while ((m = regex.exec(value)) !== null) {
          violations.push({
            line: lineAtOffset(node, value, m.index),
            message: `a phrasal verb "${label}" is not allowed — use a single plain verb`,
          });
        }
      }
    }
    return violations;
  },
  fixtures: {
    // Each must-flag snippet holds a real phrasal verb the rule must catch (SPEC §6). The
    // set proves the base form, an inflected form (-s, -ed, -ing, and an irregular past),
    // and a sentence-initial capital all match.
    mustFlag: [
      "We spin up a new worker for each job.", // base form
      "Kick off the build before lunch.", // sentence-initial capital
      "The scheduler kicked off three runs today.", // regular -ed
      "The operator spun up the cluster overnight.", // irregular past
      "The system falls back to the cache on error.", // -s form
      "Please reach out to the team for access.", // reach out
      "We should follow up on that ticket.", // follow up
      "Take off the access panel first.", // take off
      "The guide dives into the internals.", // -s form, dive into
    ],
    // Each must-not-flag snippet holds valid prose the rule must leave alone (SPEC §6).
    mustNotFlag: [
      "The build starts before lunch.", // a plain verb, no particle
      "Remove the access panel first.", // the STE rewrite of "take off"
      "Contact the team for access.", // the STE rewrite of "reach out"
      "The plane will take the northern route.", // "take" without its particle
      "The kickoff meeting covered the rollout plan.", // one-word nouns, no space
      "The rotor can spin upward of ten times.", // "spin up" boundary: "upward" is not "up"
      "Run `spin up` in the shell to start it.", // inline code span (non-prose)
      "```sh\nspin up the job now\n```", // fenced code block (non-prose)
      "> The vendor told us to spin up a worker.", // blockquote (frozen source, non-prose)
      "| Step | Note |\n| --- | --- |\n| init | spin up the worker |", // table cell (non-prose)
    ],
  },
};
