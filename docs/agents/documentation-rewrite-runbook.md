# Documentation rewrite runbook

This runbook is the fixed procedure for the documentation corpus rewrite. It applies the [Documentation Style Standard](../spec/documentation-style-standard.md) to one doc at a time. A reviewer finds the doc family, then follows the same steps for every doc.

A rewrite changes form only. Every technical claim, number, and hedge stays identical. A meaning change is not a rewrite. A meaning change becomes a new ADR.

**This runbook family is `docs/agents/*`. Its mode is Strict. Its ADHD verdict is Yes.**

## Family verdict lookup

Find the doc in this table. Read its STE mode, its ADHD verdict, and its frozen-content rule. This table encodes standard §3 and §6 for all nine families.

| Family | Path | STE mode | ADHD subset | Frozen content |
| --- | --- | --- | --- | --- |
| guides | `docs/guides/*` | STE-flavored | Yes — whole doc is procedural | None special |
| specs | `docs/spec/*` | STE-flavored | Procedural sections only | None special |
| ADRs | `docs/adr/*` | STE-flavored | No | The decision, the rationale, and the title. Escalate a thesis STE cannot express |
| research | `docs/research/*` | STE-flavored | No | Quoted primary-source text and data tables stay verbatim |
| `CONTEXT.md` | `CONTEXT.md` | Strict | No | A technical claim, number, or scope qualifier |
| `CLAUDE.md` | `CLAUDE.md` | Strict | No | A technical claim, number, or scope qualifier |
| agent docs | `docs/agents/*` | Strict | Yes | A technical claim, number, or scope qualifier |
| `README.md` | `README.md` | STE-flavored | Setup and quickstart steps only | A technical claim, number, or scope qualifier |
| `SECURITY.md` | `SECURITY.md` | STE-flavored | Report-a-vuln procedure only | Disclosure commitments (timelines, scope) stay verbatim |

## Procedure

### 1. Set the mode

1. Find the doc family in the lookup table above.
2. Read the STE mode for that family.
3. Read the ADHD verdict for that family.
4. Read the frozen-content rule for that family.

Strict mode enforces every gate, every prompt, and the one-word-one-meaning discipline. STE-flavored mode enforces every gate and every prompt. It treats the lexical rules as advisory. Both modes keep "keep modality" as a hard rule.

### 2. Apply the five mechanical gates

Apply each gate with a fixed rule and no judgment. Any hit fails.

1. **No semicolons.** Search the text for `;`. Any hit fails.
2. **Sentence length.** Count words per sentence. Fail above 20 words for an instruction. Fail above 25 words for a description.
3. **Noun-cluster cap.** Count nouns stacked as one modifier. Fail at 4 or more.
4. **Simple tenses.** Flag a compound or auxiliary verb form. Keep it only for current relevance or a hedge.
5. **No phrasal verbs.** Flag a verb plus a particle, such as "take off" or "spin up".

### 3. Apply the judgment prompts

Read each sentence. Apply these five prompts by reading. Each keeps a judgment residue.

1. **Active voice.** Find a passive form. Rewrite to active unless the actor is unknown or irrelevant.
2. **One instruction per sentence.** Split a sentence that holds two instructions.
3. **No ellipsis.** Keep the subject, the verb, and the article in each sentence.
4. **Lists for sequences.** Convert three or more prose steps to a numbered or bulleted list.
5. **Paragraph limit.** Split a paragraph above six sentences. Keep one topic per paragraph.

### 4. Keep every hedge

This rule is pure judgment. It is never a mechanical gate. It is the most common failure the skill flags.

- Compare the rewrite against the source hedge.
- "may have failed" stays "may have failed".
- Never upgrade a hedge to a fact.

In Strict mode, also keep the one-word-one-meaning discipline. Use one word for one meaning across the doc.

### 5. Apply the ADHD subset

Apply this subset to a procedural section only. First decide the section type.

1. Ask the boundary test: does the reader do these actions in order?
2. If yes, the section is procedural. Apply the eight rules below.
3. If no, the section is reference. Do not apply the subset.

| # | Rule | Action |
| --- | --- | --- |
| 1 | Lead with the next action | Start the section with the command, path, or snippet |
| 2 | Number multi-step tasks | Make each step one bounded action |
| 3 | End with one concrete next action | Close the section with one named next step |
| 4 | Suppress tangents | Finish the first topic. Move a second topic to a note |
| 5 | Matter-of-fact tone for errors | State the cause and the fix |
| 6 | Cap lists at 5 items | Apply this limit to any list |
| 7 | No preamble or closer | Start with the answer. Stop when the answer is done |
| 8 | Pre-send edit pass | Delete the announcing sentence, the closing question, and empty hedges |

Four ADHD rules are session-runtime only. Do not apply them to a static doc:

- Restate state every turn.
- Give a live time estimate.
- Make completed work visible.
- The persistence and mode-toggle mechanics.

### 6. Protect frozen content and escalate

1. Find the frozen content for the family in the lookup table.
2. Confirm each frozen claim is identical after the rewrite.
3. If STE and a frozen claim cannot co-exist, stop. Do not reword the claim.
4. Escalate the conflict. A meaning change becomes a new ADR.

An ADR title is not exempt from STE. If a title encodes a thesis STE cannot express, escalate the thesis. Never reword it silently.

### 7. Run the acceptance checklist

Copy the acceptance template below. Complete one copy per doc. The doc passes only when every check passes.

## Acceptance template

Copy this block into the rewrite ticket or the PR. Complete one per doc.

```markdown
### Acceptance: <doc path>

- Family: <family>
- STE mode: <Strict | STE-flavored>
- ADHD verdict: <Yes | No | procedural sections only>

- [ ] 1. Family and mode confirmed against the §3 table.
- [ ] 2. Mechanical gates pass (no semicolons, sentence cap, noun-cluster cap, simple tenses, no phrasal verbs).
- [ ] 3. Review prompts applied (active voice, one instruction, no ellipsis, lists, paragraph limit).
- [ ] 4. Frozen content intact. No hedge became a fact.
- [ ] 5. Structure correct. The ADHD subset is applied to procedural sections and not to reference sections.
```

After the checklist passes, open the rewrite PR. Link this runbook and the doc's ticket in the PR body.
