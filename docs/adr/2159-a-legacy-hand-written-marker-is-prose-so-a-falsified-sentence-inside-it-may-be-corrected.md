---
number: 2159
title: "A legacy hand-written marker is prose, so a falsified sentence inside it may be corrected"
slug: a-legacy-hand-written-marker-is-prose-so-a-falsified-sentence-inside-it-may-be-corrected
date: 2026-09-16
status: accepted
source: grilling
ticket: 2159
proof: {ticket: 2102}
relations:
  - {kind: amends, adr: 1644, clause: "4"}
  - {kind: rests-on, adr: 58}
---

# ADR-2159: A legacy hand-written marker is prose, so a falsified sentence inside it may be corrected

## Decision

> **A legacy hand-written marker is prose, so a falsified sentence inside it may be corrected.**
> ADR-1644 §4 keeps such a marker in place as prose. *In place* exempts it from the tool. It does not
> freeze its text. A human may remove a sentence an already-recorded withdrawal falsified. Two bounds
> hold together: the blockquote carries no `adr-marker` sentinel, and the correction never changes
> what the ADR decided.
>
> A reader asks how ADR-0110 says no clause of its Decision stands and, three times beneath, that the
> rest stands.
>
> Rejected: a later ADR carrying `supersedes` on ADR-0110. That is whole-ADR scope, so it would
> retire a live ADR to fix three sentences.
>
> Reversal leaves a landed ADR asserting both a claim and its withdrawal.

## 1. Context

ADR-0110's Decision opens: *"No clause of this Decision now stands. The Context and Consequences
below remain a record of what was decided in 2026-08."*

Three legacy hand-written withdrawal markers sit beneath that line, at `:77`, `:88` and `:136`. Each
closes with *"The rest of this Decision stands."* The third spells it out: *"the console screens are
still the IA spec, ported verbatim."* ADR-1988 has since retired the verbatim-port obligation
entirely.

So one block says no clause stands and, three times, that the rest stands.

[ADR-1644](./1644-for-an-adr-target-a-withdrawal-is-a-tool-written-marker-not-a-hand-edited-sentence.md)
rules that for an ADR target a withdrawal is a tool-written marker, never a hand-edited sentence. An
agent reading that stops, because dating or striking those sentences looks like the hand edit
ADR-1644 retired.

## 2. What §4's sentence does and does not say

ADR-1644 §4 reads: *"A legacy hand-written marker inside an ADR keeps its place as prose."*

It is silent on the text. *Keeps its place* answers a question about the tool: CI regenerates every
marker and fails on any difference, so §4 says the tool neither rewrites nor removes a legacy line.
Without that sentence, a regeneration would delete 38 blockquotes.

§5 makes the same reading plain from the other side: *"A blockquote without the sentinel is prose."*
And §4's own next clause preserves ADR-0058's hand-written form for a prose target.

What ADR-1644 retired is the practice of hand-writing a **new** marker to record a **new**
withdrawal. The 32 unmarked ADRs of 2026-09-07 are what that practice cost. Removing a sentence that
a later recorded withdrawal already falsified is not that act. It records nothing new.

## 3. The two bounds, and why both are needed

| Bound | Why |
| --- | --- |
| The blockquote carries no `<!-- adr-marker -->` sentinel | A tool-written marker is regenerated from a relation. Editing one desynchronises the file from its front matter, and CI fails. §5 already forbids it. |
| The correction removes a statement an already-recorded withdrawal falsified | This is the whole warrant. The correction adds no decision and withdraws nothing. It deletes a sentence whose contradiction is written in the same file. |

Either bound alone is too wide. The first alone would license rewriting any legacy prose, including
the reasoning a withdrawal recorded. The second alone would license editing a tool-written marker.

A correction that changed what the ADR decided is outside both, and runs through
`docs/spec/adr-governance.md` §4 as any other change does.

## 4. The rejected alternatives

**A later ADR carrying `supersedes` on ADR-0110.** Fully within ADR-1644, and the tool writes the
marker. Rejected because the instrument is far larger than the defect. `supersedes` is whole-ADR
scope with `clause` forbidden, and it derives status `superseded`. ADR-0110 reads `status: accepted`
and carries two tool-written `amends` markers, from ADR-0145 and ADR-0147. Retiring a live ADR to
delete three trailing sentences prices the cure above the disease.

**Leave all three sentences.** Cheapest. Rejected because a reader who lands on `:136` from a
citation meets a live-sounding claim that ADR-1988 retired, and the opening line that corrects it is
sixty lines above.

**Number ADR-0110's Decision headings so a clause-scoped `amends` can reach them.** Rejected: it
hand-edits a landed ADR to make a tool accept a relation, and renumbering moves every `§` citation
into the file.

## 5. The other legacy markers

This is not one file's problem, and a rule that answered one file would not be a rule.

`docs/adr` holds **38 legacy marker-like blockquotes without a sentinel, across 21 files**, against
80 tool-written markers. The oldest ADRs carry most of them: ADR-0001 holds five, ADR-0005 five,
ADR-0002 three.

The rule reaches all 38 and licenses an edit to almost none. A legacy marker is only correctable
where a later **recorded** withdrawal has falsified a sentence inside it. Most are simply old, and
being old is not being wrong. ADR-0110 qualifies because its own Decision line contradicts its own
trailing sentences.

No sweep follows from this ADR. Each case is a human's reading, as `docs/spec/citation-anchors.md`
§1.3 already gives the act to a human.

## 6. What this ADR does not rule

**Tool-written markers.** They stay untouchable. §5 of ADR-1644 governs them and this ADR does not
reach it.

**Whether a legacy marker should be migrated to a relation.** Converting 38 blockquotes into front
matter relations is a separate question with its own cost, and nothing here asks for it.

**ADR-0110's Consequences.** ADR-2157 rules a Consequences section a dated record. That reading
applies here unchanged, and this ADR reaches the Decision block alone.

## 7. Consequences

#2102 becomes a prose repair. A human deletes the three falsified trailing sentences in ADR-0110 and
leaves the withdrawals themselves standing, because those record real acts.

A reader of ADR-0110 meets one answer rather than two.

Reversal leaves a landed ADR asserting both a claim and its withdrawal, and leaves the next session
that finds one with no instrument short of retiring the file.

## 8. How ADR-1644 §4 is amended

ADR-1644 numbers its headings, and §4 is *What ADR-0058 keeps*. So this ADR declares
`{kind: amends, adr: 1644, clause: "4"}`, and the tool writes the marker under that heading.
ADR-1644's own prose is not hand-edited — which is the rule it wrote.

Nothing else in ADR-1644 moves. Its tool-written marker rule, its clause unit, its rejection of a
sentence-level marker and its check all stand. This ADR adds one limit to §4: *keeps its place*
binds the tool, not the text.

## 9. Proof

`{ticket: 2102}`. #2102 is open and names the case this rule settles: ADR-0110's Decision re-asserts
the verbatim-port rule three times beneath its own withdrawal.
