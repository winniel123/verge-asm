import { lintMarkdown } from "./engine.mjs";

// The whole engine runs per snippet, so a must-not-flag case also proves doc-lint-tool.md §3.
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
