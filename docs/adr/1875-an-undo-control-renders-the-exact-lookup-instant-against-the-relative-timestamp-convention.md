---
number: 1875
title: "An undo control renders the exact lookup instant, against the relative-timestamp convention"
slug: an-undo-control-renders-the-exact-lookup-instant-against-the-relative-timestamp-convention
date: 2026-09-12
source: fix
status: accepted
ticket: 1875
proof: {test: "cmd/web/proposals_test.go::TestUndoControlsDateEveryDeclineOfOneScope"}
relations:
  - {kind: rests-on, adr: 22}
---

# ADR-1875: An undo control renders the exact lookup instant, against the relative-timestamp convention

## Decision

**The undo control on an exclusion row renders its lookup instant in full, to the second, as
`2026-08-15 09:00:00 UTC`. Every other console timestamp keeps the design system's terse relative
form.**

The label is a disambiguator, not a timestamp. Two declines of one scope sit side by side, each
reversing a different proposal. `relTime` buckets hours under a day and days under a week, so
repeats read alike. A label that cannot part two controls has failed at its only job. Two lookups
can land in one minute, which is what sets the precision.

The ISO string already sat in the button's `title`, which needs a mouse. The label reaches touch
and keyboard.

Rejected: the relative form, with the `title` as the only disambiguator. It leaves anyone who
cannot hover guessing.

Reversal cost: the convention is repo-wide, so a later sweep restores the terse form without
reading this row.

## 1. Context

[ADR-0022](./0022-confirmation-is-singular.md) makes declining the expected act and confirming the
rare one. Its rider holds that undo is per row and per proposal, and that no surface may reverse a
whole scope in one gesture. So an exclusion row draws one undo control per declined proposal, and
the count is unbounded: `ListDeclinedProposalScopes` has no limit, and a repeated org lookup files a
fresh proposal for the same scope every time.

[#1721](https://github.com/winniel123/verge-asm/issues/1721) and
[#1777](https://github.com/winniel123/verge-asm/issues/1777) exist because an earlier render keyed
on the CIDR, which left every decline but the highest-id unreachable.
[#1803](https://github.com/winniel123/verge-asm/issues/1803) then recorded that the reachable
controls were unlabelled, so the admin's choice among them was a guess.

[#1871](https://github.com/winniel123/verge-asm/issues/1871) labelled each control on three axes —
source slug, record kind, and the lookup instant in relative form:

```
Undo decline  arin · RIR delegation · 3h
```

[#1873](https://github.com/winniel123/verge-asm/issues/1873) records what that leaves open. Two
declines that tie on all three still render identically, and the only thing left to part them is
the button's `title`.

## 2. The two ties, and why neither is exotic

The first tie is inside one lookup. One ARIN response may list two registered holders of the same
range. `ARIN.Propose` (`internal/proposer/arin.go`) walks each matched entity and emits a candidate
per CIDR, so those two proposals share a source, a record kind and an instant by construction.
Nothing in the three-axis label parts them.

The second tie is across lookups. `relTime` (`cmd/web/messages.go`) buckets minutes under an hour,
then whole hours under a day, then whole days under a week, then whole weeks. Three lookups of one
org in one afternoon, read the next day, all render `3h` or `1d`. #1803's own motivating case
produces this one: a repeated org lookup files fresh proposals for the same scope, and those
repeats share a source and a record kind by construction.

So the axis that was meant to carry the difference is the axis that discards it.

## 3. Why the `title` does not close it

A `title` appears on mouse hover alone. It is unreachable by touch and by keyboard, and screen
readers do not announce it reliably. In exactly the case where the visible label has run out of
axes, the remaining disambiguator is unavailable to some admins entirely.

The cost of the wrong choice is not cosmetic. Each control reverses one proposal and returns one
scope to pending. Undoing the wrong one leaves the intended exclusion standing, which the toast does
report, but the admin still cannot tell which control they have not yet used.

## 4. What the label carries, and where the rule is enforced

`undoControlView.Label` (`cmd/web/proposals.go`) joins the parts that are present:

```
Acme Corp · arin · RIR delegation · 2026-08-15 09:00:00 UTC
```

`org_name` is the holder name the proposal was filed under. It is the only stored column that parts
two holders inside one response, and it needs no migration —
`db/migrations/21000_proposals.sql` already declares it, `NOT NULL DEFAULT ''`. An empty holder
drops its part, as a lookup with no instant already drops the date.

The column means different things per source, and the label does not say which. `ARIN.Propose`
derives it from the RDAP entity, so for `arin` it is the registered holder.
`CAIDA.delegations` (`internal/proposer/caida.go`) stamps every candidate with the operator's own
query string, so for `afrinic-caida` and `apnic-caida` it repeats what the operator typed. That
costs this decision nothing, because the tie it parts is the one inside a single response, and
`CAIDA.orgIDs` resolves each matched org to its own opaque id — one delegated-stats prefix arrives
under one id. Making the CAIDA sources record the holder they matched is
[#1878](https://github.com/winniel123/verge-asm/issues/1878).

The instant carries seconds. `runLookup` applies no dedupe and no rate limit, so a double-submitted
form files two lookups in one minute, and a minute-precision label would tie them.

`scope.tmpl` renders `{{.Label}}` and composes nothing itself. That is the enforcement point: the
test asserts uniqueness over the same string the template prints, so the invariant cannot drift
away from the markup.

The `title` keeps its sentence and gains the holder, so it states every axis the label does. It now
restates what the label already shows, which costs nothing and keeps the hover affordance the rest
of the console teaches. A `title` that carried less than the label would part two controls for
nobody.

The row's ghost buttons drop `white-space: nowrap`. The rule has to sit on the button, because
`.sc-btn-ghost` declares `nowrap` and `white-space` inherits, so an override on the inner span does
nothing. Without it a long label is one unbreakable line and the row overflows at phone width.

## 5. What this does not decide

The row still grows one control per decline, and the label is now longer. #1803 weighed a disclosure
for the tail and deferred it rather than rejecting it. That is a layout decision and this is a
content decision; a disclosure would still need these axes inside it, because a collapsed entry
that reads `arin · RIR delegation · 3h` ties exactly as the row does.

There is still no decline instant. `proposal.status` moves in place and no column dates the move, so
the label states when the *lookup* ran. A `declined_at` column would need a migration, and it would
not close this ticket on its own: a decline instant buckets exactly as a lookup instant does.

## 6. Rejected alternatives

| Alternative | Why not |
| --- | --- |
| Keep the relative form, and rely on the `title` | The `title` needs a mouse, per §3. The case where the label ties is the case where the fallback is unreachable. |
| Add `org_name` alone, and keep the relative form | Closes the same-response tie and leaves the same-bucket tie open. #1873's stated failing input is the second one, so the fix would not hold its own test. |
| Render the exact instant alone, and skip `org_name` | Closes the same-bucket tie and leaves two holders in one response reading alike. Both ties are ordinary, so closing one is not a fix. |
| Hold this change until the CAIDA sources record a real holder ([#1878](https://github.com/winniel123/verge-asm/issues/1878)) | The tie `org_name` parts is the ARIN one, where the column is already the registered holder. Holding the fix leaves every admin guessing to buy a label improvement for two sources that do not produce the tie. |
| Render `org_name` only for `arin`, until #1878 lands | A label whose parts change per source teaches nothing, and the branch would have to be removed again. The column is the proposal's own record either way. |
| Move the tail behind a disclosure instead | A layout change that leaves the content decision unmade, per §5. It is also the larger change, and it must keep every decline reachable, which is the failure [#1721](https://github.com/winniel123/verge-asm/issues/1721) and [#1777](https://github.com/winniel123/verge-asm/issues/1777) exist to prevent. |
| Number the controls, so the label reads `1 of 3` | An ordinal names no fact about the decline. It parts the controls and tells the admin nothing about which one they want. |
| Change `relTime` so every console timestamp gains precision | The convention is right everywhere it is used. An inbox row is read for recency, and `4m` is the better answer there. This row is read for identity. |
