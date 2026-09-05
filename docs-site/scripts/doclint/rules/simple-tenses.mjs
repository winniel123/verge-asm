import pkg from "pos";
import { escapeRegex, lineAtValueOffset } from "../engine.mjs";

const { Lexer, Tagger } = pkg;

// Neither the lexer nor the tagger holds state between calls, so one shared instance is safe.
const lexer = new Lexer();
const tagger = new Tagger();

const ANCHORS = new Set(["have", "has", "had"]);

const PARTICIPLE = "VBN";

// A different split gives the tagger different context, so the measured precision would not carry.
function splitSentences(block) {
  return block
    .split(/(?<=[.!?])\s+(?=[A-Z(])/)
    .map((s) => s.trim())
    .filter(Boolean);
}

export function tagSentence(sentence) {
  return tagger.tag(lexer.lex(sentence));
}

function locate(block, anchor, participle, from) {
  const value = block.value;
  const re = new RegExp(`\\b${escapeRegex(anchor)}\\s+${escapeRegex(participle)}\\b`, "i");
  const m = re.exec(value.slice(from));
  if (!m) return { line: block.startLine, next: from };
  const index = from + m.index;
  return { line: lineAtValueOffset(value, block.startLine, index), next: index + m[0].length };
}

export const simpleTenses = {
  name: "simple-tenses",
  severity: "warning",
  check(_proseNodes, ctx) {
    const violations = [];
    for (const block of ctx.blocks) {
      let cursor = 0;
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
    mustFlag: [
      "The tokens have changed since the last release.",
      "The worker has completed the job.",
      "The values had changed before the reset.",
      "The docs have been rewritten this week.",
      "Nobody has measured the cost yet.",
    ],
    mustNotFlag: [
      "The worker completed the job.",
      "The tool has three states.",
      "The report has a clear structure.",
      "Run `has completed` in the shell to check it.",
      "```sh\necho has completed the job\n```",
      "> The vendor has completed the work for us.",
      "| Step | Note |\n| --- | --- |\n| ship | has completed |",
    ],
  },
};
