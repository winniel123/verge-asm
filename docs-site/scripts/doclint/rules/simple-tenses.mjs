/*
 * Rule: simple-tenses (SPEC §2.4, warning). The first warning-severity rule.
 *
 * The style standard §4.1 rule 4 flags a compound verb form built on `have`, `has`, or
 * `had` plus a past participle (for example "has completed", "have changed"). The standard
 * allows the form to stay when it carries current relevance or a hedge. That keep-decision
 * is a judgment residue, so the tool cannot fail a build on it.
 *
 * This rule is a WARNING, not an error. The tool flags the form. A human decides whether to
 * keep it. The warning prints a line and a CI annotation (SPEC §5), but it never changes the
 * exit code. This is the first rule that exercises the skeleton warning path.
 *
 * Method (SPEC §2.4). POS tagging. Flag `have`, `has`, or `had` directly before a past
 * participle (Penn tag VBN). #792 ran this rule with the `pos` tagger over all 176 in-scope
 * docs. Precision was about 100% (634 flags, zero scored false positives). The grammatical
 * position after the anchor forces the participle tag, so the tagger does not confuse it.
 * Volume is high on clean docs, and many flags are legitimate current-relevance uses (for
 * example "nobody has measured"). A hard error would be wrong. A warning is right.
 *
 * The tokenizer mitigation (SPEC §4.3, option 1). #792 found the retext prose stack
 * duplicates words at these package versions ("a b c" tokenizes to "a b b c c"), which
 * breaks every word-count and POS rule. So this rule does NOT use the retext bridge or a
 * retext-english pipeline. It calls the `pos` Lexer and Tagger directly, the exact path the
 * #792 prototype took, so the measured precision holds. The `pos` lexer does not duplicate a
 * word. tagSentence() is exported so a test can prove that directly.
 *
 * The rule reads a block (engine.extractProseBlocks), not the loose text nodes, because the
 * anchor and the participle can sit on either side of an inline span (emphasis, a link, an
 * inline code span). The block joins those fragments, so the pair still reaches one tag run.
 * The engine already dropped every SPEC §3 non-prose region, so a compound form inside a code
 * block, a table cell, or a blockquote never becomes a block here.
 *
 * A naming note. The SPEC §5.1 output example prints the rule id as `simple-tense`. Every
 * registration signpost (this file, rules/index.mjs, the #820 issue title) uses the plural
 * `simple-tenses`, matching the `no-semicolons` plural style. The plural is the id.
 */
import pkg from "pos";
import { escapeRegex, lineAtValueOffset } from "../engine.mjs";

const { Lexer, Tagger } = pkg;

/*
 * One lexer and one tagger for every block. Neither holds state between calls, so a single
 * shared instance is safe and skips the per-block construction cost. This mirrors the shared
 * parser in sentence-length-cap.
 */
const lexer = new Lexer();
const tagger = new Tagger();

/** The three anchor lemmas. A compound form starts on one of these. */
const ANCHORS = new Set(["have", "has", "had"]);

/** The Penn Treebank tag for a past participle. The anchor plus this tag is the flag. */
const PARTICIPLE = "VBN";

/**
 * Split a block into sentences with a naive boundary rule: a `.`, `!`, or `?` followed by
 * whitespace and a capital letter or an open paren. This is the exact split the #792
 * prototype used, so the tagger sees the same sentence shapes its precision was measured on.
 * A mis-split only changes the tag context a little, never whether a `;`-free block is read.
 *
 * The name is `splitSentences`, not `sentencesOf`, because sentence-length-cap already owns a
 * `sentencesOf` that walks an nlcst tree. These are two different operations on two different
 * inputs, so they carry two different names.
 * @param {string} block
 * @returns {string[]}
 */
function splitSentences(block) {
  return block
    .split(/(?<=[.!?])\s+(?=[A-Z(])/)
    .map((s) => s.trim())
    .filter(Boolean);
}

/**
 * Tag one sentence with the `pos` lexer and tagger.
 *
 * Exported for the no-duplication test (SPEC §4.3, acceptance criterion 3). The `pos` lexer
 * returns one token per word, so "a b c" never becomes "a b b c c". The retext stack this
 * rule avoids does duplicate. A test asserts the clean stream here.
 * @param {string} sentence
 * @returns {[string, string][]}  [word, Penn tag] pairs, in order.
 */
export function tagSentence(sentence) {
  return tagger.tag(lexer.lex(sentence));
}

/**
 * The 1-based source line of the anchor-plus-participle pair inside a block. The block value
 * joins its fragments with a space (never a newline), and a soft-wrapped paragraph keeps its
 * own newlines, so lineAtValueOffset maps the pair's offset back to the source line. This is
 * the same offset-to-line rule the other rules use.
 *
 * The search starts at `from` and returns the new cursor, so two identical pairs on different
 * lines report their own lines instead of both pointing at the first.
 * @param {import("../engine.mjs").ProseBlock} block
 * @param {string} anchor  the anchor word as lexed (have/has/had, original case).
 * @param {string} participle  the participle word as lexed (original case).
 * @param {number} from  the search cursor into block.value.
 * @returns {{line: number, next: number}}
 */
function locate(block, anchor, participle, from) {
  const value = block.value;
  const re = new RegExp(`\\b${escapeRegex(anchor)}\\s+${escapeRegex(participle)}\\b`, "i");
  const m = re.exec(value.slice(from));
  if (!m) return { line: block.startLine, next: from };
  const index = from + m.index;
  return { line: lineAtValueOffset(value, block.startLine, index), next: index + m[0].length };
}

/** @type {import("../engine.mjs").Rule} */
export const simpleTenses = {
  name: "simple-tenses",
  severity: "warning",
  check(_proseNodes, ctx) {
    const violations = [];
    for (const block of ctx.blocks) {
      let cursor = 0; // a per-block search cursor, so repeated pairs map to distinct lines
      for (const sentence of splitSentences(block.value)) {
        const tagged = tagSentence(sentence);
        for (let i = 0; i < tagged.length - 1; i++) {
          const anchor = tagged[i][0];
          const [participle, tag] = tagged[i + 1];
          if (!ANCHORS.has(anchor.toLowerCase()) || tag !== PARTICIPLE) continue;
          const hit = `${anchor} ${participle}`;
          const { line, next } = locate(block, anchor, participle, cursor);
          cursor = next;
          violations.push({
            line,
            message: `a compound verb form "${hit}" — confirm current relevance or a hedge, else use a simple tense`,
          });
        }
      }
    }
    return violations;
  },
  fixtures: {
    // Each must-flag snippet holds a real compound form the rule must catch (SPEC §6). The
    // set covers all three anchors, the auxiliary "have been" chain, and a legitimate
    // current-relevance use — the rule still flags it, because a human, not the tool, keeps
    // it. Every participle here tags VBN with the `pos` tagger (#792's path).
    mustFlag: [
      "The tokens have changed since the last release.", // have anchor
      "The worker has completed the job.", // has anchor
      "The values had changed before the reset.", // had anchor
      "The docs have been rewritten this week.", // auxiliary "have been" chain
      "Nobody has measured the cost yet.", // a current-relevance use — flag, human keeps
    ],
    // Each must-not-flag snippet holds valid prose the rule must leave alone (SPEC §6). The
    // set proves a simple past without an anchor, an anchor before a noun or a number, and
    // every SPEC §3 non-prose region (inline code, a code block, a blockquote, a table cell).
    mustNotFlag: [
      "The worker completed the job.", // simple past, no anchor before the participle
      "The tool has three states.", // has + a number, not a participle
      "The report has a clear structure.", // has + a determiner and noun, not a participle
      "Run `has completed` in the shell to check it.", // inline code span (non-prose)
      "```sh\necho has completed the job\n```", // fenced code block (non-prose)
      "> The vendor has completed the work for us.", // blockquote (frozen source, non-prose)
      "| Step | Note |\n| --- | --- |\n| ship | has completed |", // table cell (non-prose)
    ],
  },
};
