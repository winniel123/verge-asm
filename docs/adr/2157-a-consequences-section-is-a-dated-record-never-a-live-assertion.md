---
number: 2157
title: "A Consequences section is a dated record, never a live assertion"
slug: a-consequences-section-is-a-dated-record-never-a-live-assertion
date: 2026-09-16
status: accepted
source: grilling
ticket: 2157
proof: {ticket: 2127}
relations:
  - {kind: bounds, adr: 58}
---

# ADR-2157: A Consequences section is a dated record, never a live assertion

## Decision

> **A Consequences section is a dated record, never a live assertion.** It states what the session
> saw when it decided. Production then moves, and the record does not become false — it becomes
> historical. So a Consequences section is never amended, and a reference inside it that no longer
> resolves degrades rather than being repointed.
>
> A reader asks what to do when a landed ADR describes a predicate that ships differently, as
> ADR-0186 and ADR-0185 do.
>
> Rejected: amending both through a later ADR. `docs/spec/adr-governance.md` §4 cannot express it —
> a clause is required wherever the target numbers any heading, and no numbered clause carries the
> claim.
>
> Reversal makes every Consequences section editable in place, so the record of what was decided
> stops being a record.

## 1. Context

ADR-0186 and ADR-0185 each carry a Consequences bullet saying `certificate-expiring`'s predicate
ships a flat thirty-day window through `certExpiryWindow`.

Both were accurate when written. `certExpiryWindow` is now declared nowhere, and
`internal/signal/endpoint.go#CertHorizon` returns `validity/3`, or `validity/2` at ten days or less —
the proportional horizon ADR-0186's own bullet says is *"not in the code"*.

The question #2157 asked was how the two ADRs should record that the code moved. The answer is that
they should not, because recording what the code did on a date is already what they were doing.

## 2. A Decision clause and a Consequences bullet are different kinds of sentence

A Decision clause is a rule. It holds until a later ADR changes it, so a reader may act on it today,
and `docs/spec/adr-governance.md` §4 exists to keep that promise.

A Consequences bullet is an observation. It says what the session found while deciding, and it is
bounded by the date in the front matter. Nothing promises it still holds.

Collapsing the two is what makes a false problem here. Read as a live assertion, ADR-0186's bullet is
wrong and needs repair. Read as a dated observation, it is correct and needs nothing.

[ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md) already
states this for itself: *"The Context and Consequences below remain a record of what was decided in
2026-08."* This ADR generalises that sentence rather than inventing it.

## 3. What still happens to a stale reference

The record stands. The reference inside it may not resolve, and that is a separate act.

`docs/spec/citation-anchor-repair.md` §6.3 rules that an anchor whose declaration does not survive
degrades and is never repointed. That applies inside a Consequences section exactly as it does
anywhere else. Degrading a reference does not touch the sentence around it, so the record is
undisturbed.

So the repair for ADR-0186 and ADR-0185 is: leave both bullets, degrade `certExpiryWindow`, and add
no marker and no relation. #2127 carries that work.

## 4. Why the amendment could not be written

This ADR was first drafted as the amendment #2157 asked for, and the draft could not pass.

`docs/spec/adr-governance.md` §4 requires a `clause` wherever the target numbers any heading, and the
check tests the whole file. Both targets number their Decision subsections and do not number
`## Consequences`. The claims sit only in Consequences, so a clause was demanded and no honest clause
existed. Amending a numbered Decision clause would have asserted a change that did not happen —
ADR-0186's own bullet says *"The band is unaffected"*.

That gap is real and is filed as #2195. This ADR does not close it. It removes the case that exposed
it, and #2195 stays open for a target whose **Decision** genuinely needs amending at whole-file
scope.

## 5. The rejected alternatives

**Amend both targets through a later ADR.** The reading #2157 assumed. Rejected because it cannot be
expressed, and because it treats an observation as a rule.

**Number the Consequences headings of both targets, then amend by clause.** Rejected: it hand-edits
two landed ADRs to make a tool accept a relation, and renumbering headings moves every `§` citation
into them.

**Leave the bullets and the stale reference alone.** Rejected in part. The record stands, but a
reference that resolves to nothing sends a reader looking for a constant no file declares, and §6.3
already rules what to do with one.

## 6. What this ADR does not rule

**Context sections.** ADR-0110's sentence pairs Context with Consequences, and the same reasoning
appears to reach both. This ADR rules Consequences alone, because that is the section #2127 exposed,
and a rule written past its evidence is the habit this repository keeps finding.

**Whether a Consequences bullet may state a rule.** Some do, in practice. This ADR rules how a
Consequences section is read, not how it should have been written.

**The wording in `docs/spec/adr-governance.md`.** The SPEC gains a line saying a Consequences section
is a dated record. That edit belongs to the ticket that implements this, not to the ADR.

## 7. Consequences

A session that finds a landed ADR describing production as it no longer is has a cheap answer:
nothing, unless a reference fails to resolve.

A reader who wants to know what production does today reads the code, and a reader who wants to know
what was believed on a date reads the Consequences section. Neither is served by editing the second
into the first.

Reversal makes every Consequences section editable in place, so the record of what was decided stops
being a record.

## 8. How this bounds ADR-0058

[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) rules that a
superseded mechanism is withdrawn at the site that specifies it.

A Consequences bullet does not specify a mechanism. It reports one. So ADR-0058's obligation does not
reach it, and this ADR declares `{kind: bounds, adr: 58}` to record where that boundary lies.
`bounds` derives no status and writes no marker, so ADR-0058 is untouched.

## 9. Proof

`{ticket: 2127}`. #2127 is open and names the case this rule settles: ADR-0186's Consequences bullet
cites a removed `certExpiryWindow`.
