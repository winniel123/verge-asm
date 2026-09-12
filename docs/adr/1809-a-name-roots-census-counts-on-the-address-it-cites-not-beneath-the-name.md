---
number: 1809
title: "A `Name` root's census counts on the address it cites, not beneath the `Name`"
slug: a-name-roots-census-counts-on-the-address-it-cites-not-beneath-the-name
date: 2026-09-12
source: fix
status: accepted
ticket: 1809
proof: {test: "internal/message/censusownership_test.go::TestASecondNameOnOneAddressClaimsNoMoreThanTheFirst"}
relations:
  - {kind: amends, adr: 31}
  - {kind: rests-on, adr: 1806, clause: "4"}
---

# ADR-1809: A `Name` root's census counts on the address it cites, not beneath the `Name`

## Decision

**A membership headline names the ground its count was taken over. A `Name` root reads
`timelines opened on an address it cites`. An `Address` root keeps `opened beneath it`.**

One address carries many `Name`s. Every `Name` citing it censuses every `Service` and `Endpoint`
on it, so `k` roots each report the same `n` subjects. "Beneath it" reads as a claim about the
`Name`, and no `Name` owns that ground.

The census itself keeps every member, so the citation axis still decides membership.

Rejected: splitting the census into an owned group and a shared one. `internal/scan/hot.go` sets
no `Scope.Names`, so every `Endpoint` the hot tier opens is nameless. That owned group is empty in
every production census. Rejected: deduplicating across roots, which silences `k-1` of them.

Reversal cost: a sent headline is immutable, so a third wording leaves three shapes in one inbox.

## 1. Context

[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) fires one membership
message at the root of an entering sub-tree and carries "the census of what entered beneath it".
Its Decision defines the root as the entering subject whose own `Citation` reaches the estate, and
it walks a `Citation` chain rather than a name hierarchy.

[ADR-1806](./1806-a-census-is-computed-once-from-a-cause-frozen-basis-when-the-admitting-tier-has-drained.md)
§4 states what that chain admits. A `Name` root's census is the set of subjects whose address the
root's own resolution cites at that batch. It adds the `Endpoint`s that name owns.

The address is the only anchor, and one address may carry many `Name`s. `releaseHeldMessage`
computes one held row at a time, per root. So `k` `Name`s citing one address each release a census
naming all `n` subjects on it. On a CDN or a shared-hosting address both numbers are large.

The payload is right and the sentence is wrong. "Beneath it" is a claim about the `Name`, and the
count is true only of the address.

## 2. The headline names its ground

`membershipCensusClause` takes the root kind and renders the ground with it. A `Name` root reads
`opened on an address it cites`. An `Address` root reads `opened beneath it`, because the address
is the ground its sub-tree sits on rather than a thing it points at.

The count, its factors and the census payload are untouched.
[ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md) §4 still gets
its factors before the product, and
[ADR-0102](./0102-a-subjects-row-is-the-base-a-census-member-row-is-its-explicit-modifier.md) still
gets one count over one population.

The release path reads the root kind from the frozen basis. That basis already carries `RootKind`,
for ADR-1806 §4's own reason: a `revealed` firing fires at the `Seed` and names no root on the row.

## 3. Why the census is not split instead

\#1809 proposed a second remedy: count only what the root's `Name` owns, and render the rest as a
named second group. That group is empty in production, permanently.

`HotJob.JobSpec` (`internal/scan/hot.go`) marshals a `connectoutcome.Scope` with no `Names`.
`Scope.endpointNames` therefore returns one empty string, and `EmitCertificate` writes
`EndpointKey("", target, "tcp")`. Every `Endpoint` the hot tier opens is nameless.
ADR-1806 §5 admits only `reachability` and `certificate` to a census, and `http-identity`, which
does carry a `Name`, runs behind §3's bound. So no census member is ever owned by any `Name`.

[#1808](https://github.com/winniel123/verge-asm/issues/1808) and
[#1810](https://github.com/winniel123/verge-asm/issues/1810) removed the name-leg test for that
reason. [#1774](https://github.com/winniel123/verge-asm/issues/1774) then closed on the rule that
a later ticket must not restore the name axis. A split restores it as the counting axis.

## 4. Proof, and what it does not cover

`TestASecondNameOnOneAddressClaimsNoMoreThanTheFirst`
(`internal/message/censusownership_test.go`) renders two `Name` roots over one census and asserts
that neither headline says "beneath it". `TestAnAddressRootKeepsBeneathIt` holds the other arm.
`TestTwoNamesOnOneAddressCensusTheSameSubjects` (`internal/queue/censusownership_test.go`) is
\#1809's stated failing input. It records the fan-out this ADR declines to remove.

Two things stay open. The `k × n` volume is untouched. `k` messages still carry `n` members each,
and `ListSubjectsOpenedSinceBatch` still reads every subject opened at or after the root's batch
with no upper bound. `rePointHeadline` (`internal/message/repoint.go`) renders "opened beneath it"
for a `Name` subject, and it over-claims on the same ground. ADR-0026 §2 owns that message, so it
is a decision of its own.

## 5. Rejected alternatives

| Alternative | Why not |
| --- | --- |
| Split the census into owned and shared groups | The owned group is empty in every production census, per §3. It also counts on the axis #1774 told a later ticket not to restore. |
| Deduplicate across roots, so one root carries what no root owns | A release computes one row at a time, when that root's own tier drains, so there is no moment at which the sibling roots are all known. It also silences `k-1` roots about real openings. |
| Cap the census | [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)'s damping constraint puts every count in the payload, and [#27](https://github.com/winniel123/verge-asm/issues/27) refuses an invented number in a safety path. |
| Leave the wording and document the reading | The headline is the whole message in a channel body, which carries a count and no rows. A reader who must consult an ADR to parse it has no remedy at all. |
