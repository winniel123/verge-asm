import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { lineAtOffset, escapeRegex } from "../engine.mjs";

const WORDLIST_PATH = join(dirname(fileURLToPath(import.meta.url)), "..", "phrasal-verbs.json");

function regularForms(verb) {
  const forms = [verb];

  if (/(s|x|z|ch|sh)$/.test(verb)) forms.push(verb + "es");
  else if (/[^aeiou]y$/.test(verb)) forms.push(verb.slice(0, -1) + "ies");
  else forms.push(verb + "s");

  forms.push(verb + "ing");

  if (verb.endsWith("e")) forms.push(verb + "d");
  else if (/[^aeiou]y$/.test(verb)) forms.push(verb.slice(0, -1) + "ied");
  else forms.push(verb + "ed");

  return forms;
}

function buildMatchers() {
  const raw = readFileSync(WORDLIST_PATH, "utf8");
  const { entries } = JSON.parse(raw);
  return entries.map((entry) => {
    const forms = [...regularForms(entry.verb), ...(entry.irregular ?? [])];
    const alternation = [...new Set(forms)].sort((a, b) => b.length - a.length).map(escapeRegex);
    const particle = escapeRegex(entry.particle);
    return {
      regex: new RegExp(`\\b(?:${alternation.join("|")})\\s+${particle}\\b`, "gi"),
      label: `${entry.verb} ${entry.particle}`,
    };
  });
}

const MATCHERS = buildMatchers();

export const noPhrasalVerbs = {
  name: "no-phrasal-verbs",
  severity: "error",
  check(proseNodes) {
    const violations = [];
    for (const node of proseNodes) {
      const value = node.value;
      for (const { regex, label } of MATCHERS) {
        regex.lastIndex = 0;
        let m;
        while ((m = regex.exec(value)) !== null) {
          violations.push({
            line: lineAtOffset(node, value, m.index),
            message: `a phrasal verb "${label}" is not allowed — use a single plain verb`,
          });
        }
      }
    }
    return violations;
  },
  fixtures: {
    mustFlag: [
      "We spin up a new worker for each job.",
      "Kick off the build before lunch.",
      "The scheduler kicked off three runs today.",
      "The operator spun up the cluster overnight.",
      "The system falls back to the cache on error.",
      "Please reach out to the team for access.",
      "We should follow up on that ticket.",
      "Take off the access panel first.",
      "The guide dives into the internals.",
    ],
    mustNotFlag: [
      "The build starts before lunch.",
      "Remove the access panel first.",
      "Contact the team for access.",
      "The plane will take the northern route.",
      "The kickoff meeting covered the rollout plan.",
      "The rotor can spin upward of ten times.",
      "Run `spin up` in the shell to start it.",
      "```sh\nspin up the job now\n```",
      "> The vendor told us to spin up a worker.",
      "| Step | Note |\n| --- | --- |\n| init | spin up the worker |",
    ],
  },
};
