/*
 * Rule: sentence-length-cap (SPEC §2.2, error).
 *
 * The style standard §4.1 rule 2 sets two caps: 20 words for an instruction, 25 words
 * for a description. The tool cannot tell an instruction from a description, because that
 * split needs section context the tool does not have. So the tool uses one universal
 * error line: a sentence above 25 words fails.
 *
 * The 20-word warning stays deferred. SPEC §2.2 marks it optional ("the implement effort
 * MAY add it"). The #818 acceptance criteria state that a sentence at or under 25 words is
 * "not flagged". A warning still flags: it prints a line and a CI annotation (SPEC §5), even
 * though it never changes the exit code. So a 20-word warning would flag a 21-to-25-word
 * sentence and contradict that criterion. This is an issue-versus-SPEC tension, because
 * SPEC §2.2 does frame the 20-word cap as a warning. A later effort can add it with its own
 * acceptance line that resolves the tension.
 *
 * Method (SPEC §2.2). Parse each prose block into an nlcst prose tree with parse-english.
 * Count the WordNode children of each SentenceNode. This is a plain word count, not the
 * retext-readability formula the SPEC §2.2 rules out. #791 confirmed it needs no plugin.
 *
 * The rule reads a block (engine.extractProseBlocks), not the loose text nodes, because a
 * sentence can run across an inline span (emphasis, a link, an inline code span). The block
 * joins those fragments, so the whole sentence reaches one word count. The engine already
 * dropped every SPEC §3 non-prose region, so a long line in a code block, a table cell, or
 * a blockquote never becomes a sentence here.
 *
 * Boundary hedge (SPEC §2.2). The sentence split is deterministic, but it is not perfect.
 * An abbreviation or a decimal can mis-split a sentence. parse-english reads a lowercase
 * word plus a period as a possible abbreviation, so it does not always break there. For
 * example "cap." (as in "the word cap.") reads as an abbreviation, so parse-english keeps
 * the next sentence joined to it. Two short sentences then merge into one long one, and the
 * rule can over-flag it. This residue is expected, not a defect. The fixture corpus holds a
 * decimal and an "e.g." abbreviation inside a genuinely long sentence, to prove the count
 * still holds in those cases. The deferred inline directive (SPEC §6, #821) is the escape
 * hatch for the unavoidable over-flag: a writer marks the line and the tool skips it.
 *
 * The tokenizer caveat (SPEC §4.3). #792 reported that the retext prose stack could
 * duplicate words at these versions ("a b c" reads as "a b b c c"). This code reproduced
 * the duplication, and it comes from unist-util-visit over this nlcst tree, not from
 * parse-english. The parser tree is correct: one WordNode per word. So this rule walks the
 * tree by hand (see collect() below), which counts correctly. The package pins the exact
 * parse-english version anyway (SPEC §4.3 option 2), so a later install cannot drift into a
 * parser that regresses. The fixture corpus (SPEC §6) is the guard: a duplicated word would
 * push a must-not-flag sentence over the cap and fail the acceptance gate.
 */
import { ParseEnglish } from "parse-english";

/** The universal error cap. A sentence strictly above this many words fails. */
const MAX_WORDS = 25;

/*
 * One parser for every block. ParseEnglish holds no state between parse() calls, so a
 * single shared instance is safe and skips the per-block construction cost.
 */
const parser = new ParseEnglish();

/**
 * Collect every nlcst node of one type, in document order, without descending into a
 * match. The two callers below share this one walk, so there is one traversal, not two.
 *
 * The rule walks the tree by hand and does NOT use unist-util-visit. The visitor
 * double-counts a WordNode on this nlcst tree: it turns "a b c" into "a b b c c", so a
 * word count would roughly double. The bug is in the visitor over this tree shape, not in
 * parse-english, whose tree holds one WordNode per word. #792 saw the same duplication and
 * read it as a parser bug (SPEC §4.3). A hand walk is the reliable route.
 * @param {object} node   an nlcst node.
 * @param {string} type   the node type to collect.
 * @param {object[]} [out]
 * @returns {object[]}
 */
function collect(node, type, out = []) {
  if (node.type === type) {
    out.push(node); // do not descend: the target type never nests itself here
    return out;
  }
  for (const child of node.children ?? []) collect(child, type, out);
  return out;
}

/**
 * The WordNode count of one sentence. A SentenceNode holds WordNode, WhiteSpaceNode,
 * PunctuationNode, and SymbolNode children. Only a WordNode counts toward the cap.
 * @param {import("nlcst").Sentence} sentence
 * @returns {number}
 */
function wordCount(sentence) {
  return collect(sentence, "WordNode").length;
}

/**
 * Every SentenceNode of an nlcst tree, in document order.
 * @param {import("nlcst").Root} tree
 * @returns {import("nlcst").Sentence[]}
 */
function sentencesOf(tree) {
  return collect(tree, "SentenceNode");
}

/** @type {import("../engine.mjs").Rule} */
export const sentenceLengthCap = {
  name: "sentence-length-cap",
  severity: "error",
  check(_proseNodes, ctx) {
    const violations = [];
    for (const block of ctx.blocks) {
      const tree = parser.parse(block.value);
      for (const sentence of sentencesOf(tree)) {
        const words = wordCount(sentence);
        if (words <= MAX_WORDS) continue;
        // The sentence position is 1-based inside the block string. The block joins its
        // fragments with a space (never a newline), so the within-block line offset still
        // maps back to the source line.
        const offset = (sentence.position?.start?.line ?? 1) - 1;
        violations.push({
          line: block.startLine + offset,
          message: `a sentence runs ${words} words. The cap is ${MAX_WORDS} words. Split it.`,
        });
      }
    }
    return violations;
  },
  fixtures: {
    // Each must-flag snippet holds a sentence above the 25-word cap (SPEC §6). The set
    // captures the SPEC §2.2 boundary "worst cases": a decimal and an abbreviation that
    // must not stop the tokenizer from counting the whole long sentence.
    mustFlag: [
      // 28 plain words, one flat sentence.
      "This single sentence keeps going and going and it adds another clause and then one more clause until it clearly runs well over the universal cap value here.",
      // A long sentence that wraps an inline code span. The prose on both sides joins into
      // one sentence, so the words still total over the cap even though the span drops out.
      "You should run the `doclint` command against every single documentation file in the whole repository before you open a pull request or ask anyone to review the change.",
      // Boundary hedge: a decimal number mid-sentence. It stays one word, and the sentence
      // still runs over the cap, so the decimal must not mask the length.
      "The tool ships version 1.5 today and it still counts every single ordinary word in this deliberately long run-on sentence that clearly goes over the cap.",
      // Boundary hedge: an abbreviation mid-sentence. "e.g." must not split the sentence in
      // a way that hides its true length.
      "Pass one flag, e.g. the verbose one, whenever you debug a genuinely tricky failure that spans a great many words and clearly runs over the universal cap.",
    ],
    // Each must-not-flag snippet holds only sentences at or under the cap (SPEC §6).
    mustNotFlag: [
      // A short clean sentence.
      "The tool reports one line per violation.",
      // Exactly 25 words. The cap is inclusive, so this must not flag. It is the boundary.
      "This sentence holds exactly twenty five plain words in total and it sits right at the cap value so the rule must leave it alone.",
      // Two sentences in one paragraph, 30 words together but neither over the cap. This
      // proves the rule counts a sentence, not a paragraph (acceptance criterion 3).
      "This first sentence carries fifteen words and it stays well under the universal cap value here. This second sentence also carries a comfortable fifteen words under the cap.",
      // A long line inside a fenced code block (over 25 words). Code is non-prose, so the
      // rule never counts it (acceptance criterion 4).
      "```text\nthis code comment runs on and on with far more than twenty five words in a single long line that a prose rule must never ever read or count\n```",
      // A long cell inside a table. Table text is non-prose, so it never counts.
      "| Field | Note |\n| --- | --- |\n| cap | this cell holds far more than twenty five words in one long run that the rule must never read because a table is non prose |",
      // A long blockquote sentence. A blockquote is frozen quoted source, so it never counts.
      "> This quoted source sentence runs on with well over twenty five words in a single long line and the tool leaves every frozen word of it alone.",
    ],
  },
};
