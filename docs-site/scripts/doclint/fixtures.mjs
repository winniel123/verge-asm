/*
 * The fixture-corpus harness (SPEC §6).
 *
 * Each rule owns two snippet sets: a must-flag set and a must-not-flag set. A rule
 * passes when it flags every must-flag snippet and no must-not-flag snippet. The
 * harness runs the whole engine on each snippet, not the rule alone, so a must-not-flag
 * snippet also proves that prose extraction (SPEC §3) hides the non-prose regions.
 *
 * The candidate warnings (SPEC §2.5) will use this same gate to prove precision before
 * a later ticket enables them.
 */
import { lintMarkdown } from "./engine.mjs";

/**
 * @typedef {Object} CorpusFailure
 * @property {"must-flag"|"must-not-flag"} kind
 * @property {string} snippet
 * @property {number} got  the violation count the rule produced.
 */

/**
 * Run one rule against its own must-flag and must-not-flag sets.
 * @param {import("./engine.mjs").Rule} rule
 * @returns {CorpusFailure[]} empty when the rule passes.
 */
export function checkRuleCorpus(rule) {
  const failures = [];
  const fixtures = rule.fixtures ?? { mustFlag: [], mustNotFlag: [] };

  for (const snippet of fixtures.mustFlag) {
    const got = lintMarkdown(snippet, [rule]).length;
    if (got === 0) failures.push({ kind: "must-flag", snippet, got });
  }
  for (const snippet of fixtures.mustNotFlag) {
    const got = lintMarkdown(snippet, [rule]).length;
    if (got > 0) failures.push({ kind: "must-not-flag", snippet, got });
  }
  return failures;
}
