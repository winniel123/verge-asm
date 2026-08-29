/*
 * Rule: no-semicolons (SPEC §2.1, error).
 *
 * The style standard §4.1 rule 1 makes any semicolon in prose a failure. The detection
 * is a deterministic tree-walk: read each prose text node the engine hands over and
 * match a literal `;`. The rule never runs a regular expression over the raw markdown,
 * because that would flag a semicolon inside a code span or a table cell. The engine
 * already dropped every non-prose region (SPEC §3) before this rule runs, so every `;`
 * it sees is prose.
 */
import { startLineOf } from "../engine.mjs";

const MESSAGE = "a semicolon in prose is not allowed — write two sentences";

/** @type {import("../engine.mjs").Rule} */
export const noSemicolons = {
  name: "no-semicolons",
  severity: "error",
  check(proseNodes) {
    const violations = [];
    for (const node of proseNodes) {
      const value = node.value;
      // One scan per node. Track the line with a running counter, one increment per
      // newline, so a match points at its own line, not the node's first line. The
      // engine de-dups a line reported twice, so this rule reports every match.
      let line = startLineOf(node);
      for (let i = 0; i < value.length; i++) {
        const ch = value[i];
        if (ch === "\n") line++;
        else if (ch === ";") violations.push({ line, message: MESSAGE });
      }
    }
    return violations;
  },
  fixtures: {
    // Each must-flag snippet holds a real violation the rule must catch (SPEC §6).
    mustFlag: [
      "This clause is wrong; it joins two sentences with a semicolon.",
      "The tool has three states: ready; running; done.",
      "A trailing case;",
      "Line one has no problem.\n\nLine three does; here it is.",
    ],
    // Each must-not-flag snippet holds valid prose the rule must leave alone (SPEC §6).
    mustNotFlag: [
      "This sentence is clean and has no semicolon.",
      "Run the `for x in y; do z; done` loop to iterate.", // inline code span
      "```sh\nfor x in y; do z; done\n```", // fenced code block
      "| Command | Effect |\n| --- | --- |\n| `a; b` | runs a then b |", // table cell
      "> A quoted source keeps its semicolon; the tool leaves it frozen.", // blockquote
      "---\ntitle: One; two; three\n---\n\nThe body prose is clean.", // front-matter
    ],
  },
};
