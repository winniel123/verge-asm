---
number: 2086
title: "An anchor that moved keeps its meaning only while it names the same declaration"
slug: an-anchor-that-moved-keeps-its-meaning-only-while-it-names-the-same-declaration
date: 2026-09-16
status: accepted
source: grilling
ticket: 2086
proof: {ticket: 2080}
relations:
  - {kind: rests-on, adr: 58}
---

# ADR-2086: An anchor that moved keeps its meaning only while it names the same declaration

## Decision

> **An anchor that moved keeps its meaning only while it names the same declaration.** Repointing it
> to the same declaration is a factual repair: it needs no relation, and a tool may do it. Repointing
> it to a different declaration changes what the document asserts, so it runs through a later ADR
> under `docs/spec/adr-governance.md` §4, and a human does it.
>
> A reader asks which of two SPECs governs. §3 called every repoint a repair, and
> `docs/spec/citation-anchor-repair.md` §6.3 says a suspect anchor degrades and is never repointed.
> Neither narrowed the other.
>
> Rejected: letting §3 win outright. It authorises 803 anchors inside landed ADRs with no relation
> and no marker, and a history-arm run finds 71 that cross declarations.
>
> Reversal lets a tool move what a landed ADR asserts, leaving no record that it moved.

## 1. Context

Two SPECs overlapped on `docs/adr` and disagreed.

`docs/spec/adr-governance.md` §3 says repointing a stale anchor is a factual repair that needs no
relation. `docs/spec/citation-anchor-repair.md` §6.3 says a suspect anchor degrades and is never
repointed, because putting one declaration where another stood can change an assertion.

Two partial reconciliations existed, and neither closed the gap. adr-governance §1 excludes
`docs/spec/*`, which settles a spec anchor but not `docs/adr`, where both SPECs apply.
citation-anchor-repair §1.3 gives the act to a human, which narrows the actor but not the
classification.

## 2. Why neither sentence was wrong

Both are true of the case each was written for, and each overreached by leaving its case unstated.

A line number drifts constantly. A declaration is renamed, split or deleted rarely. §3 was written
for the common case, where the cited thing is still there and has merely moved down the file.
§6.3 was written for the rare one, where the nearest match is a *different* thing that happens to
sit where the old anchor pointed.

So the disagreement is not about repair versus change. It is about which of two different acts the
word *repointing* names.

## 3. The line is declaration identity

| The anchor now names | The act | Relation | Who |
| --- | --- | --- | --- |
| The same declaration, at a new position | A factual repair | None | A tool may do it |
| A different declaration | A change to what the document asserts | Required, under adr-governance §4 | A human |

Position is not meaning. A citation names a declaration, and the line number is only how a reader
finds it. When the declaration survives, nothing the document asserts has moved, and the repair
restores the citation to what it always meant. When the declaration does not survive, the citation
now points at a claim the author never made.

`docs/spec/citation-anchors.md` §3.3 already carries the instrument for the second row: an anchor
the sweep cannot derive degrades rather than guessing.

## 4. What a tool may do, and what it may not

A tool may repoint an anchor whose declaration it can prove is the same one. It may always degrade.

It may not decide that two declarations are the same declaration. That judgement is what the second
row reserves to a human, and no check can make it: the `citations` gate proves a declaration name
exists, not that it is the one the document meant.

## 5. The rejected alternatives

**§3 wins outright, and every repoint is a repair.** Cheapest, and it keeps the sweep working
unchanged. Rejected because a landed ADR's meaning could then move with no marker and no relation,
which is the one thing `docs/spec/adr-governance.md` §4 exists to prevent.

**§6.3 wins outright, and every repoint inside a landed ADR needs a relation.** Rejected on cost with
no matching benefit: it invalidates the authorisation `docs/spec/citation-anchors.md` §8.1 rests on,
so all 803 anchors need rework, including the large majority whose declaration never changed. It
would also force an ADR for a line number that drifted because a comment above it grew.

## 6. Blast radius

`docs/spec/citation-anchors.md` §8.1 cites §3 to authorise converting 803 anchors inside landed ADRs.
Under this ADR that authorisation covers the first row alone.

A history-arm run over `docs/adr` reports 413 judged: 71 drift candidates, 342 consistent, and 396
unwitnessed. The 71 are the second row and need a human. The 396 are unproven rather than wrong.
#2140 carries the triage.

## 7. What this ADR does not rule

**The staged conversion.** #2156 carries the 986 anchors the widened gate now reports and does not
refuse. This ADR rules which of them a tool may convert, not when the stage retires.

**Non-ADR targets.** adr-governance §1 excludes `docs/spec/*`. This ADR rules the overlap on
`docs/adr` and leaves that exclusion standing.

**The gate.** Nothing here asks the `citations` check to detect a crossed declaration. It cannot, and
§4 of this file says why.

## 8. Consequences

The sweep keeps its common case and loses its dangerous one. A repoint that crosses declarations now
stops and degrades, which is visible, rather than succeeding quietly.

Both SPECs keep their sentence and gain a limit: adr-governance §3 says *to the same declaration*,
and citation-anchor-repair §6.3 says *a different declaration*. Neither is withdrawn.

Reversal lets a tool move what a landed ADR asserts, leaving no record that it moved.

## 9. Proof

`{ticket: 2080}`. #2080 is open and names this rule: it asks for the line between a factual repair
and a change to an assertion, where the two SPECs overlap.
