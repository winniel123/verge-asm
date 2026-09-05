import {
  splitSentences,
  tagSentence,
  lineOfSentence,
  IMPERATIVE_VERBS,
  LEADING_ADVERBS,
} from "./tagging.mjs";

const COORDINATORS = new Set(["and", "or", "then"]);

const BASE_VERB = "VB";

function isImperativeVerb([word, tag]) {
  return tag === BASE_VERB || IMPERATIVE_VERBS.has(word.toLowerCase());
}

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

function hasTwoInstructions(tagged) {
  const start = firstContentIndex(tagged);
  if (start >= tagged.length || !isImperativeVerb(tagged[start])) return false;

  for (let i = start + 1; i < tagged.length; i++) {
    if (!COORDINATORS.has(tagged[i][0].toLowerCase())) continue;
    let j = i + 1;
    while (j < tagged.length && COORDINATORS.has(tagged[j][0].toLowerCase())) j++;
    if (j < tagged.length && isImperativeVerb(tagged[j])) return true;
  }
  return false;
}

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
    mustFlag: [
      "Run the build and deploy the result.",
      "Open the file then save it.",
      "Stop the server and restart it.",
      "Save the changes and close the editor.",
      "First, install the plugin then enable it.",
    ],
    mustNotFlag: [
      "Read the spec and the standard before you start.",
      "Click the button to submit the form.",
      "The tool parses the file and prints the result.",
      "Run `build && deploy` in the shell.",
      "> Run the build and deploy the result.",
    ],
  },
};
