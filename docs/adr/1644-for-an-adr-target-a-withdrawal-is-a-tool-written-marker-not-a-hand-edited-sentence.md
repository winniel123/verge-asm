---
number: 1644
title: "For an ADR target, a withdrawal is a tool-written marker, not a hand-edited sentence"
slug: for-an-adr-target-a-withdrawal-is-a-tool-written-marker-not-a-hand-edited-sentence
date: 2026-09-08
status: accepted
source: grilling
ticket: 1644
map: 1638
proof: {test: "docs-site/scripts/check-adr-markers.test.mjs::a relation without a marker fails"}
relations:
  - {kind: amends, adr: 58}
---

# ADR-1644: For an ADR target, a withdrawal is a tool-written marker, not a hand-edited sentence

## Decision

For an ADR target, a withdrawal is a tool-written marker, not a hand-edited sentence. The acting ADR declares one relation in its front matter. The tool renders one blockquote line under the affected heading, signed `<!-- adr-marker -->`. CI regenerates every marker and fails on any difference. A relation without a marker, a marker without a relation, and a hand-written marker all fail.

The unit is the numbered clause, or the whole file when the target numbers no heading. This coarsens the #106 sentence rule for an ADR target only. A spec or research target keeps the hand-written sentence rule.

The pass that supersedes still pays. It writes the relation. The marker names the acting ADR by thesis title and date.

Rejected: a sentence-level marker, which needs a prose parser and a hand edit. On 2026-09-07, 32 ADRs read Accepted while a later ADR amended a clause.

Proof: `docs-site/scripts/check-adr-markers.test.mjs`.

## 1. Context

[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) rules that
the pass that supersedes a mechanism marks the superseded site in the same change. Its amendment
from [#106](https://github.com/winniel123/verge-asm/issues/106) fixes the unit at the sentence, so a
clause read alone never reads as live. That rule asks a hand to find the sentence and edit it.

The rule held for a spec or research target. It failed for an ADR target. On 2026-09-07 the count
stood at 32: ADRs whose status line read *Accepted* while a later ADR amended one of their clauses.
No parser reads prose for the sentence that a later decision withdraws, and no check caught the
gap. The wayfinder map [#1638](https://github.com/winniel123/verge-asm/issues/1638) grilled the
repair in [#1644](https://github.com/winniel123/verge-asm/issues/1644), and
`docs/spec/adr-governance.md` §5 carries the marker form this ADR decides.

## 2. The unit for an ADR target

The unit is the numbered clause of `docs/spec/comment-policy.md` §4.7, named as a dotted string in
the relation's `clause` field. When the target numbers no heading, the relation omits `clause` and
acts on the whole file. ADR-0058 is one such target, so this amendment carries no clause and its
marker sits under ADR-0058's H1.

The reader test of ADR-0058 stands. A clause read alone must not read as live. The marker meets it
because it is the first line under the affected heading, so no reader reaches the clause without
passing it.

## 3. The marker and the check

The acting ADR declares one `amends`, `retires`, or `supersedes` relation in its front matter. The
tool renders one blockquote line under the affected heading. The line names the acting ADR by thesis
title and date, links `./<number>-<slug>.md`, and ends in `<!-- adr-marker <kind> <n> -->`. A
`withdrawn` status renders its own line from the `withdrawal` field.

The check regenerates every marker from the front matter and diffs the tree. Three failures collapse
into that one diff.

| Failure | What the diff shows |
| --- | --- |
| A relation without a marker | The render adds a line the file lacks |
| A marker without a relation | The file holds a sentinel line the render does not produce |
| A hand-written marker | A sentinel line that does not byte-equal the render |

A blockquote without the sentinel is prose. The check ignores it, so the legacy hand-written lines
in ADR-0001, ADR-0002, and ADR-0004 stand.

## 4. What ADR-0058 keeps

A spec or research target keeps the sentence rule and the hand-written form. A legacy hand-written
marker inside an ADR keeps its place as prose. The pass that supersedes still pays: it writes the
relation, and the tool does the rest. The obligation moves from a prose edit to a front matter line.
It does not go away.

## 5. Rejected: a sentence-level marker

A sentence-level marker for an ADR target needs a prose parser to find the sentence and a hand edit
to mark it. The 32 unmarked ADRs of 2026-09-07 are the cost of that pair. A clause is a heading the
existing checker already parses, so the tool can write and verify the marker with no hand in the
loop.

## 6. Proof

`docs-site/scripts/check-adr-markers.test.mjs` holds `a relation without a marker fails`,
`a marker without a relation fails`, and `a hand-edited marker fails`. The `adr-sections` job runs
the suite and then `--check` on the tree, so every marker on `main` byte-equals its regeneration.
