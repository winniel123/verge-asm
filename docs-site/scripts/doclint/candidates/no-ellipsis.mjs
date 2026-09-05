import {
  splitSentences,
  tagSentence,
  lineOfSentence,
  IMPERATIVE_VERBS,
  LEADING_ADVERBS,
} from "./tagging.mjs";

const VERB_TAGS = new Set(["VB", "VBD", "VBG", "VBN", "VBP", "VBZ", "MD"]);

function opensWithImperative(tagged) {
  let i = 0;
  while (i < tagged.length && LEADING_ADVERBS.has(tagged[i][0].toLowerCase())) i++;
  return i < tagged.length && IMPERATIVE_VERBS.has(tagged[i][0].toLowerCase());
}

// A heading arrives as an ordinary prose block, so the closing-period test is what excludes it.
function looksLikeSentence(sentence) {
  if (!/[.!?]$/.test(sentence)) return false;
  return (sentence.match(/[\w'-]+/g) ?? []).length >= 3;
}

function looksElided(sentence, tagged) {
  if (!looksLikeSentence(sentence)) return false;
  if (opensWithImperative(tagged)) return false;
  return !tagged.some(([, tag]) => VERB_TAGS.has(tag));
}

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
    mustFlag: [
      "Configuration of the build tokens.",
      "Then the same result for the tokens.",
      "A short note about the run cost.",
    ],
    mustNotFlag: [
      "The tool parses the file.",
      "The worker completed the job.",
      "Run the build before you deploy.",
      "The docs are rewritten this week.",
      "> Configuration of the build tokens.",
    ],
  },
};
