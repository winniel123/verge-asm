# The IANA Special-Purpose Address Registry, as `non-globally-reachable-address-resolved-from-internet` reads it

The curated table read by the v1 rule
**`non-globally-reachable-address-resolved-from-internet`**, admitted by
[#128](https://github.com/winniel123/verge-asm/issues/128) and recorded in
[ADR-0071](../adr/0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md).

It is the **fourth** curated table in the product, after the 38 sensitive `(port, transport)` pairs
([`sensitive-ports.md`](./sensitive-ports.md)), `certificate-expiring`'s fraction
([`acme-renewal-timing.md`](./acme-renewal-timing.md)) and
`certificate-weak-key-or-signature`'s five rows
([`weak-key-and-signature.md`](./weak-key-and-signature.md)). Like the other three it sits inside
[ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)'s **gate 2** — a
table asserting about the **world**. Unlike the other three it is a **transcription** of its owner's
own artefact rather than a selection from one, which is what makes it pass gate 2 by construction and
is the general finding ADR-0071 records.

**This document is the transcription and the reading rule. It authors no row.**

---

## 1. What the table is, and who owns it

The **IANA IPv4 Special-Purpose Address Registry** and the **IANA IPv6 Special-Purpose Address
Registry**, both established by [RFC 6890](https://www.rfc-editor.org/rfc/rfc6890.html) (the IPv4 one
succeeding the RFC 5736 registry), each carrying a column headed **Globally Reachable**.

- [iana-ipv4-special-registry](https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml)
- [iana-ipv6-special-registry](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml)

**The owner is IANA, and the artefact is the allocation itself.** This is the cleanest owner in the
product's four tables and it is clean for a structural reason rather than a lucky one: the other
three assert something *about* an artefact somebody else designed — *this exposure is never correct*,
*this primitive is weak*, *this is the last point the operator can act*. This one asserts that IANA
allocated a block for a purpose that is not global reachability, and **IANA's allocation is what
makes that true**. There is no gap between the claim and the claimant for an attestation to have to
cross.

Per [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) an
attestation is retrieved over the **artefact**, not over the row. Here the artefact **is** the table,
so one retrieval attests every row at once, and a block the registry names that this file lacks is a
transcription defect rather than an admission question.

**The closed claim set, derived per [ADR-0034](../adr/0034-derive-the-claim-before-looking-for-the-owner.md)
before looking for the owner.** ADR-0032 §3 obliges a second curated table to derive its own closed
claim set from **what its rule reads**, and never to inherit #21's three. The rule reads an address
and the registry's classification of the block containing it. A registry of allocations, asked about
an address, can assert exactly one thing: **that the allocating authority designated the block
containing this address for a purpose other than global reachability.** One member, and the
derivation closes there because a registry of allocations has no second kind of sentence to utter
about an address.

## 2. The reading rule, and every judgement in it is the registry's own

> **Take the most specific registered block containing the address. The address is in the firing set
> if and only if that block's `Globally Reachable` cell reads `False`.**

Four things follow, and none of them is a choice of ours.

**Longest match is the owner's instruction, not our construction.** The registries nest — `192.0.0.9/32`
(Port Control Protocol Anycast, **True**) sits inside `192.0.0.0/24` (IETF Protocol Assignments,
**False**), and `2001:1::1/128`, `2001:1::2/128` and `2001:1::3/128` (all **True**) sit inside
`2001::/23` (**False**). The IPv6 registry states the rule in its own footnote on that very row:

> `[1]` **"Unless allowed by a more specific allocation."**
> — footnote to `2001::/23`, [iana-ipv6-special-registry](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml)

Reading *any containing block* instead of *the most specific* reports `192.0.0.9` — an address the
registry marks globally reachable — as not globally reachable. That is a **false verdict**, and it is
the failure mode #31 named a signature database for.

**`True` is not a firing value and is not merely an absence.** Fourteen blocks across the two
registries carry `True`. They are inside the registry and outside the firing set, which is what makes
the set a **partition of the registry** rather than a list we drew.

**A cell that is not `True` or `False` is not `False`.** Four blocks carry `N/A` or a termination
date. Reading them as `False` would be **us supplying a value the owner declined to supply**, which
is authorship, and it is the exact line ADR-0071 draws. So they do not fire.

That reading is also right on the merits, and the glossary already agrees with it: `CONTEXT.md`'s
`Address` entry records that *"NAT64, 6to4 and Teredo addresses are real IPv6 addresses reachable
only by their own paths"* — the transition mechanisms whose blocks carry `N/A` embed a globally
routable IPv4 address, so reachability is a property of the embedded address rather than of the
prefix, which is presumably why the owner left the cell empty. **The model's refusal to fold them and
the table's refusal to classify them are the same refusal.**

**An address that is not inside any registered block is outside the firing set**, with no default and
no inference. Ordinary global unicast space is simply unregistered here.

## 3. The transcription

**[measured]** Retrieved 2026-08-15 from the two registry pages named in §1. Cells are reproduced as
printed, footnote markers included.

### 3.1 IPv4 — 25 blocks

| Block | Name | `Globally Reachable` | Fires |
| --- | --- | --- | --- |
| `0.0.0.0/8` | "This network" | False | **yes** |
| `0.0.0.0/32` | "This host on this network" | False | **yes** |
| `10.0.0.0/8` | Private-Use | False | **yes** |
| `100.64.0.0/10` | Shared Address Space | False | **yes** |
| `127.0.0.0/8` | Loopback | False `[1]` | **yes** |
| `169.254.0.0/16` | Link Local | False | **yes** |
| `172.16.0.0/12` | Private-Use | False | **yes** |
| `192.0.0.0/24` | IETF Protocol Assignments | False | **yes** |
| `192.0.0.0/29` | IPv4 Service Continuity Prefix | False | **yes** |
| `192.0.0.8/32` | IPv4 dummy address | False | **yes** |
| `192.0.0.9/32` | Port Control Protocol Anycast | **True** | no |
| `192.0.0.10/32` | Traversal Using Relays around NAT Anycast | **True** | no |
| `192.0.0.170/32`, `192.0.0.171/32` | NAT64/DNS64 Discovery | False | **yes** |
| `192.0.2.0/24` | Documentation (TEST-NET-1) | False | **yes** |
| `192.31.196.0/24` | AS112-v4 | **True** | no |
| `192.52.193.0/24` | AMT | **True** | no |
| `192.88.99.0/24` | Deprecated (6to4 Relay Anycast) | **N/A** | no |
| `192.88.99.2/32` | 6a44-relay anycast address | False | **yes** |
| `192.168.0.0/16` | Private-Use | False | **yes** |
| `192.175.48.0/24` | Direct Delegation AS112 Service | **True** | no |
| `198.18.0.0/15` | Benchmarking | False | **yes** |
| `198.51.100.0/24` | Documentation (TEST-NET-2) | False | **yes** |
| `203.0.113.0/24` | Documentation (TEST-NET-3) | False | **yes** |
| `240.0.0.0/4` | Reserved | False | **yes** |
| `255.255.255.255/32` | Limited Broadcast | False | **yes** |

`[1]` on Loopback: *"Several protocols have been granted exceptions to this rule. For examples, see
RFC 8029 and RFC 5884."* The **cell still reads `False`** and the footnote is about protocol-internal
uses of `127/8` (MPLS OAM, BFD) rather than about what a DNS answer means, so it does not move the
read. Recorded because it is the one `False` cell carrying a qualifier.

### 3.2 IPv6 — 25 blocks

| Block | Name | `Globally Reachable` | Fires |
| --- | --- | --- | --- |
| `::1/128` | Loopback Address | False | **yes** |
| `::/128` | Unspecified Address | False | **yes** |
| `::ffff:0:0/96` | IPv4-mapped Address | False | **unreachable — §4.1** |
| `64:ff9b::/96` | IPv4-IPv6 Translat. | **True** | no |
| `64:ff9b:1::/48` | IPv4-IPv6 Translat. | False | **yes** |
| `100::/64` | Discard-Only Address Block | False | **yes** |
| `100:0:0:1::/64` | Dummy IPv6 Prefix | False | **yes** |
| `2001::/23` | IETF Protocol Assignments | False `[1]` | **yes** |
| `2001::/32` | TEREDO | **N/A** `[2]` | no |
| `2001:1::1/128` | Port Control Protocol Anycast | **True** | no |
| `2001:1::2/128` | Traversal Using Relays around NAT Anycast | **True** | no |
| `2001:1::3/128` | DNS-SD Service Registration Protocol Anycast | **True** | no |
| `2001:2::/48` | Benchmarking | False | **yes** |
| `2001:3::/32` | AMT | **True** | no |
| `2001:4:112::/48` | AS112-v6 | **True** | no |
| `2001:10::/28` | Deprecated (previously ORCHID) | **(terminated 2014-03)** | no |
| `2001:20::/28` | ORCHIDv2 | **True** | no |
| `2001:30::/28` | Drone Remote ID Protocol Entity Tags (DETs) Prefix | **True** | no |
| `2001:db8::/32` | Documentation | False | **yes** |
| `2002::/16` `[3]` | 6to4 | **N/A** `[3]` | no |
| `2620:4f:8000::/48` | Direct Delegation AS112 Service | **True** | no |
| `3fff::/20` | Documentation | False | **yes** |
| `5f00::/16` | Segment Routing (SRv6) SIDs | False | **yes** |
| `fc00::/7` | Unique-Local | False `[4]` | **yes** |
| `fe80::/10` | Link-Local Unicast | False | **yes** |

Footnotes as printed: `[1]` *"Unless allowed by a more specific allocation."* · `[2]` *"See Section 5
of [RFC4380] for details."* · `[3]` *"See [RFC3056] for details."* · `[4]` *"The Unique-Local prefix
is drawn from the IPv6 Global Unicast Address range, but is specified as not globally routed."*

### 3.3 The arithmetic

| | IPv4 | IPv6 | Total |
| --- | --- | --- | --- |
| Blocks registered | 25 | 25 | **50** |
| `Globally Reachable` = `False` | 19 | 13 | **32** |
| `Globally Reachable` = `True` | 5 | 9 | **14** |
| Neither (`N/A`, terminated) | 1 | 3 | **4** |
| **Blocks that can ever fire** | 19 | 12 | **31** |

The firing set is **32 blocks, of which 31 can ever be reached** — `::ffff:0:0/96` is excluded by the
model rather than by the registry (§4.1). The `True` and neither columns are the load-bearing half:
**18 of the 50 blocks in the registry are inside the table and outside the firing set**, and a
hand-drawn private-range list would have had to invent all eighteen answers.

## 4. Two model interactions that were checked rather than assumed

### 4.1 `::ffff:0:0/96` is unreachable by construction, and that is ADR-0051 working

The IPv4-mapped block carries `False`, so on a naive read it fires. It cannot, because
[ADR-0051](../adr/0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)
keys an IPv4-mapped address **as the IPv4 address it represents** — *"one subject, since that block
is defined as a way of writing an IPv4 address rather than as an address"*. No `Address` subject in
the model ever holds a value inside `::ffff:0:0/96`, so the row is transcribed and can never be
consulted. It is retained rather than dropped because **dropping it would be a selection**, which is
the one thing this file may not do.

The interaction runs the right way round: `::ffff:0:0/96` is read out as the IPv4 address, and the
IPv4 address is then classified by §3.1. `::ffff:10.0.0.1` fires as `10.0.0.1`, which is correct.

### 4.2 The registry names three blocks the model has already reasoned about

`CONTEXT.md`'s `Address` entry names **NAT64, 6to4 and Teredo** as *"real IPv6 addresses reachable
only by their own paths"* and refuses to fold them. Of those, the registry gives 6to4 and Teredo
`N/A` and gives NAT64 a **split** verdict — `64:ff9b::/96` is `True` and `64:ff9b:1::/48` is `False`.
The split is precisely the longest-match rule doing work, and it is a case no hand-drawn list would
have produced.

## 5. Cadence, per [ADR-0004](../adr/0004-signals-are-release-coupled-rules.md)

The test is *would we ever want to push updates to this list out of band?* **No**, and the reason is
structural rather than a promise.

- The reference set is **closed and enumerable** in ADR-0004's own sense: 50 rows over a finite
  address space, each row an allocation rather than an observation of anybody's behaviour.
- It **does move** — `3fff::/20`, `5f00::/16`, `100:0:0:1::/64` and `2001:1::3/128` are all recent
  additions — but it moves at **RFC publication cadence**, on the order of a row a year across both
  registries.
- The failure mode of staleness is **a blind spot, never a false verdict**. A block allocated after
  our release is unregistered as far as our copy is concerned, so an address inside it is outside the
  firing set and the rule stays quiet. Nothing already in the table changes meaning underneath a
  comparison, which is the harm ADR-0004 was written for.

**That asymmetry is the general test and it is stated as derived rather than measured**: a verdict
table is a signature database where staleness produces a **false verdict**, and it is reference data
where staleness produces a **blind spot**. `nmap-service-probes`' `match` half is the first kind (a
stale row emits a wrong product name); this registry is the second.

## 6. The disclosure

Per ADR-0032 §7 and its #67 and #73 amendments, the weak part is published here with the retrieval
that resolves it named.

**One cell class is read on our reading of what an empty cell means.** The four `N/A` and terminated
rows are read as *not firing* (§2). The owner declined to state a value and we decline to conclude
one — which is authorship refused, not a gap papered over — and it is the only place in this table
where a cell is not simply copied into a verdict. **The direction of the residual error is toward
silence**, not toward a false firing: a `2002::` or `2001::/32` address published in an operator's
public DNS would not fire. That is the honest cost and it is small, because both blocks embed a
globally routable IPv4 address and so disclose something other than internal topology.

**What resolves it:** an owner statement giving those blocks a `Globally Reachable` value, or a
registry revision that does. **What does not resolve it:** any amount of reasoning by us about what
6to4 and Teredo addresses imply — that is the reader inference §2 refuses, and it would convert this
file from a transcription into a selection.

**One `False` cell carries a qualifier** — `127.0.0.0/8`'s footnote `[1]` — and is read as `False`
(§3.1). This is disclosed rather than argued: the footnote's examples are MPLS OAM and BFD, neither
of which is a DNS answer, so the cell governs. **What would resolve it:** an owner statement that any
of those exceptions makes a `127/8` address globally reachable.

**Nothing here is a shipping gate.** ADR-0032's Consequences are explicit that gate 2 never was one:
a table with an unattested row ships, disclosed. Both disclosures above are narrower than that — the
rows are attested; what is disclosed is a reading.

## Sources

- [IANA IPv4 Special-Purpose Address Registry](https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml) — retrieved 2026-08-15
- [IANA IPv6 Special-Purpose Address Registry](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml) — retrieved 2026-08-15
- [RFC 6890 — Special-Purpose IP Address Registries](https://www.rfc-editor.org/rfc/rfc6890.html), which establishes both registries and the `Globally Reachable` attribute
