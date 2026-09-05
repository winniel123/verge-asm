import { startLineOf } from "../engine.mjs";

const MESSAGE = "a semicolon in prose is not allowed — write two sentences";

export const noSemicolons = {
  name: "no-semicolons",
  severity: "error",
  check(proseNodes) {
    const violations = [];
    // A regex over raw markdown would flag a semicolon in a code span (doc-lint-tool.md §2.1).
    for (const node of proseNodes) {
      const value = node.value;
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
    mustFlag: [
      "This clause is wrong; it joins two sentences with a semicolon.",
      "The tool has three states: ready; running; done.",
      "A trailing case;",
      "Line one has no problem.\n\nLine three does; here it is.",
    ],
    mustNotFlag: [
      "This sentence is clean and has no semicolon.",
      "Run the `for x in y; do z; done` loop to iterate.",
      "```sh\nfor x in y; do z; done\n```",
      "| Command | Effect |\n| --- | --- |\n| `a; b` | runs a then b |",
      "> A quoted source keeps its semicolon; the tool leaves it frozen.",
      "---\ntitle: One; two; three\n---\n\nThe body prose is clean.",
    ],
  },
};
