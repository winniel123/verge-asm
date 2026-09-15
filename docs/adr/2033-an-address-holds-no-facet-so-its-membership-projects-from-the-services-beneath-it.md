---
number: 2033
title: "An Address holds no facet, so its membership projects from the Services beneath it"
slug: an-address-holds-no-facet-so-its-membership-projects-from-the-services-beneath-it
date: 2026-09-15
status: accepted
source: fix
ticket: 2033
pr: 2049
proof: {test: "cmd/web/inventory_test.go::TestBuildInventoryProjectsAddressesFromServicesAndEndpoints"}
relations:
  - {kind: amends, adr: 105, clause: "1"}
  - {kind: rests-on, adr: 47}
  - {kind: rests-on, adr: 104}
  - {kind: rests-on, adr: 2027}
---

# ADR-2033: An Address holds no facet, so its membership projects from the Services beneath it

## Decision

**An Address holds no facet of its own. No producer writes an address-kind span, and its
membership projects from the Services and Endpoints beneath it.**

Every read that needs the estate's Addresses derives them from those subject keys. The
inventory renders one facet-less Address row per distinct host beneath a Service or an
Endpoint. A withdrawal closes the timelines beneath an Address and counts the Address as a
subject that left, holding none of its own.

A reader asks why a listed subject carries no row. Nothing at the connect layer separates
an Address from the Services on it, which ADR-0104 settled when it made a blanket
responder a gap cause on the Service rather than a value about the Address.

Rejected: an address-scoped facet, which needs a measurement that does not exist.

Reversal returns five reads that silently answer nothing.

## 1. Context

`internal/queue/spanfold.go#foldOne` is the only writer of a span, and it takes the subject kind
from `internal/queue/pure.go#subjectKindFor`. That function maps `reachability` and `tls-acceptance`
to `service`, `certificate` and `http-identity` to `endpoint`, `resolution` and `dns-record` to
`name`, and defaults to `name`. No branch returns `address`.

So an address-kind span cannot exist in a live estate.
[#2033](https://github.com/winniel123/verge-asm/issues/2033) found four surfaces resting on one, and
the implementation found five reads:

| Read | What it returned |
| --- | --- |
| `db/queries/span.sql` `ListCitedAddressSpansForNames` | Nothing, so `internal/queue/membership.go#closeUncitedAddresses` closed no Service or Endpoint beneath an uncited Address. |
| `db/queries/messages.sql` `ListSeedWithdrawalCandidates` | Nothing. Withdrawing an address Seed closed nothing at all. |
| `db/queries/messages.sql` `ListAddressExclusionWithdrawals` | Nothing. An address exclusion withdrew nothing. |
| `db/queries/messages.sql` `PreviewExclusionWithdrawal` | Zero subjects and zero timelines. |
| `db/queries/messages.sql` `SpendSeedWithdrawals` | A guard that was vacuously true, so the tombstone spent by accident. |

`cmd/web/inventory.go` carried the absence in a comment — *"No address-kind reach span exists, so
without the lift no Address flags"* — beside the proxy-edge lift that works around it. What no
document stated is whether any facet was ever meant to produce the kind.

## 2. No measurement is about an Address

The candidate is `blanket-discrimination`. `internal/measure/blanketdiscrim/leaf.go` opens by saying
it *"decides one fact about an `Address`"*, and its operator reason reads *"this address answers on
all ports"*. It is the only leaf whose subject is the Address rather than a Service on it.

It is already ruled, and ruled the other way.
[ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md)
makes an undiscriminated reach the sixth **gap cause** on `reachability`, never a third value, and
`Kind` is *"composed by connect-outcome, not dispatched"*. The verdict reaches the corpus as a
property of the Service's reach, on the Service's subject.

[ADR-0125](./0125-a-port-selective-silent-drop-edge-is-undiscriminated-and-stays-a-gap.md) then
refused to add a positive signal for a port-selective edge, because the connect layer cannot tell one
from a default-drop origin. Its reasoning generalises: what is measurable about an address is
measured by connecting to a port on it, which is a Service. An address-scoped facet would need an
observation that is about the address and not about any port, and no leaf produces one.

The four remaining facets settle it by their own subjects. `resolution` and `dns-record` are about a
Name. `certificate` and `http-identity` are single-valued only per `(Name, Service)`, which is what
the Endpoint kind exists for.

## 3. What the projection reads

[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) already holds the Address as a subject
with Services beneath it, and `internal/estate/address.go#AddressPresent` already encodes the rule
this ADR completes: presence is citation or Seed cover, because an Address has no existence to
observe.

The projection is the reading side of that. A Service key is `<addr>:<port>/<transport>` and an
Endpoint key prefixes it with `<name>@`, so every open Service and Endpoint span names the Address it
sits on. The set of Addresses in the estate is the set of hosts those keys carry.

`cmd/web/inventory.go#projectAddressSubjects` derives the inventory group that way, and each SQL read
derives its own set in a `projected_addr` CTE. The Address row carries no facet, because it holds no
reading. That is the honest render, not a missing one.

The withdrawal counts follow. An Address that leaves takes the Services and Endpoints beneath it, so
it counts as a subject withdrawn and contributes no timeline. The preview and the fold derive the
same set, which is what keeps them from disagreeing under
[ADR-0218](./0218-the-address-exclusion-withdrawal-is-idempotent-by-construction-and-its-receipt-twins-the-previews-site-and-counts.md)
§4.

## 4. Why this amends ADR-0105 §1

[ADR-0105](./0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md) §1 says
inventory is *"the same `span` rows, projected differently"*, and that there is no new table, no new
observation, and no new value. The first clause is now too narrow, and the rest still holds.

An Address row corresponds to no span. It is derived from the **keys** of the spans beneath it, which
is still a read over the open-span corpus and still invents no datum — but it is no longer a
one-to-one render of rows. ADR-0105 could not have said otherwise: it was written when the address
rows in the design corpus made the group look like every other one.

The alternative reading is to drop the group, and it fails on the queue. The Address is load-bearing
outside the inventory: `internal/queue/repoint.go` roots an `appeared` message on it, `message.CensusBasis`
carries it as a root kind, and `seed` and `exclusion` both declare an address kind. A subject the
estate withdraws, cites, and roots messages on is not a render detail.

[ADR-2027](./2027-an-inventory-row-counts-a-reading-and-a-per-vantage-facet-names-its-derived-class.md)
is untouched. It rules that a facet row counts a reading and collapses nothing. A projected Address
renders zero facet rows because it holds zero readings, which is that rule applied rather than an
exception to it.

## 5. What reversal costs

Reverting is mechanical: five reads go back to filtering on `subject_kind = 'address'`, and the
inventory group empties.

What comes back is a console that silently answers nothing. An operator withdraws an address Seed and
the estate does not move. An operator previews an address exclusion and reads zero subjects over
ground that holds Services. A departed Name leaves its Addresses' Services open forever. None of
those fail loudly, and none of them appeared in a test, because every fixture that exercised them
carried the address-kind span no producer writes.

That is the reversal cost this ADR is recorded against. The code is small. What is expensive is a
corpus that looks correct while five reads return nothing, which is the state #2033 found.
