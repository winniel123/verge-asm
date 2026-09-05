import { ParseEnglish } from "parse-english";

// A 20-word warning would flag a 21-word sentence, which #818's acceptance line forbids.
const MAX_WORDS = 25;

// No state survives a parse() call, so one shared parser instance is safe for every block.
const parser = new ParseEnglish();

// A unist-util-visit walk double-counts a WordNode on this nlcst tree, so its count is wrong.
function collect(node, type, out = []) {
  if (node.type === type) {
    out.push(node);
    return out;
  }
  for (const child of node.children ?? []) collect(child, type, out);
  return out;
}

function wordCount(sentence) {
  return collect(sentence, "WordNode").length;
}

function sentencesOf(tree) {
  return collect(tree, "SentenceNode");
}

export const sentenceLengthCap = {
  name: "sentence-length-cap",
  severity: "error",
  check(_proseNodes, ctx) {
    const violations = [];
    // A sentence can run across an inline span, which splits it into several loose text nodes.
    for (const block of ctx.blocks) {
      const tree = parser.parse(block.value);
      for (const sentence of sentencesOf(tree)) {
        const words = wordCount(sentence);
        if (words <= MAX_WORDS) continue;
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
    mustFlag: [
      "This single sentence keeps going and going and it adds another clause and then one more clause until it clearly runs well over the universal cap value here.",
      "You should run the `doclint` command against every single documentation file in the whole repository before you open a pull request or ask anyone to review the change.",
      "The tool ships version 1.5 today and it still counts every single ordinary word in this deliberately long run-on sentence that clearly goes over the cap.",
      "Pass one flag, e.g. the verbose one, whenever you debug a genuinely tricky failure that spans a great many words and clearly runs over the universal cap.",
    ],
    mustNotFlag: [
      "The tool reports one line per violation.",
      "This sentence holds exactly twenty five plain words in total and it sits right at the cap value so the rule must leave it alone.",
      "This first sentence carries fifteen words and it stays well under the universal cap value here. This second sentence also carries a comfortable fifteen words under the cap.",
      "```text\nthis code comment runs on and on with far more than twenty five words in a single long line that a prose rule must never ever read or count\n```",
      "| Field | Note |\n| --- | --- |\n| cap | this cell holds far more than twenty five words in one long run that the rule must never read because a table is non prose |",
      "> This quoted source sentence runs on with well over twenty five words in a single long line and the tool leaves every frozen word of it alone.",
    ],
  },
};
