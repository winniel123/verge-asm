# ADR-0009: `verge-core` is the union of a frequency set and a normative list

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#29 Hot-set transport defects and the sensitive-subset-of-hot invariant](https://github.com/winniel123/verge-asm/issues/29)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

Two port lists exist, built for different questions on incompatible evidence standards.

[#4](https://github.com/winniel123/verge-asm/issues/4) built the ~~~140-port~~ **`verge-core` hot
set** on frequency — *how likely is this port to be found open in a small org's estate?* — from
`nmap-services` open-frequency data with a modern-services supplement, keyed on **bare port
numbers**, and specified that it ship "as an editable list file, not compiled in".

> **`~140` was never this set's size.** **[measured]** by
> [#97](https://github.com/winniel123/verge-asm/issues/97): the frequency half is **123, all TCP**, and
> `verge-core` — this ADR's union — is **136 pairs** as composed after
> [#95](https://github.com/winniel123/verge-asm/issues/95). The label is left standing above per the
> name-and-withdraw convention; nothing in this ADR's Decision reads it.

[#21](https://github.com/winniel123/verge-asm/issues/21) built the **38-pair sensitive list** on a
normative standard — *given the port is open to the internet, is that ever correct?* — keyed on
**`(port, transport)` pairs**, and settled the relationship between the two in its §6: the lists
stay **independent**, coupled by a one-directional build-time invariant

```
every (port, transport) on the sensitive list MUST be a member of the hot set
```

with a test that fails the build otherwise. The coupling points that way because the two lists have
asymmetric constraints — the hot set is bounded by probe cost, the sensitive list by correctness —
and probing one more port per host per day is cheap where being unable to evaluate the product's
best signal is not. **Cost yields to correctness**, and the invariant encodes which is which.

The invariant is violated today, in two ways at once.

**Two transport mismatches.** `verge-core`'s Management/OOB group is specified as
`161 (TCP), 623`. Per IANA, SNMP is **161/udp**; IPMI/ASF-RMCP is **623/udp**; and **623/tcp** is
`oob-ws-http`, the DMTF out-of-band web services protocol — a different protocol on the same
number. As specified, the hot set probes the TCP siblings of two UDP services and evaluates
neither.

**Four omissions.** 512/tcp (rexec), 4369/tcp (epmd), 25672/tcp (RabbitMQ inter-node) and
27019/tcp (MongoDB config server) are on the sensitive list and absent from the hot set. A
sensitive pair absent from the hot set can never fire `sensitive-port-exposed` **at all**, and
under a naive implementation presents as *did not fire* — the clean bill of health
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s `not-evaluable` rule exists to prevent.

A containment check over bare integers passes today and lets both defects survive.

Two things found while resolving the ticket, each of which changed an answer:

- **The conflation is in the evidence, not only in the list.** [#4](https://github.com/winniel123/verge-asm/issues/4)
  §2.2 — the measured argument for the whole modern-services supplement — carries the row
  `623 | IPMI/BMC | 1,267 | no`. That rank is computed over `nmap-services` **TCP** entries and
  labelled with a **UDP** service. 623 entered `verge-core` under a frequency figure for a
  protocol it was not measuring.
- **[`CONTEXT.md`](../../CONTEXT.md) was already right.** `Service` is an
  `(Address, port, transport)` triple. The domain model has been transport-keyed since
  [#7](https://github.com/winniel123/verge-asm/issues/7); it is the two lists and the wire contract
  that disagree with the model they feed, not the model that needs changing.

## Decision

| Concern | Decision |
| --- | --- |
| Keying | Both lists on **`(port, transport)` pairs** |
| How far the re-key reaches | The list files, the [#5](https://github.com/winniel123/verge-asm/issues/5) job spec / `internal/wire` contract, **and the `Batch` scope record** |
| What `verge-core` is | **Derived: `frequency-set ∪ sensitive-list`** — a definition, never a maintained list |
| The invariant | **Dissolved.** `sensitive ⊆ verge-core` holds by construction |
| Enforcement | **None, anywhere** — no build-time test, no config-load check, no runtime check |
| Operator editing | The **frequency half only**; the sensitive half is a shipped signal's reference data |
| The four TCP additions | An ordinary aperture widening → [ADR-0007](https://github.com/winniel123/verge-asm/blob/adr-0007-drift-model/docs/adr/0007-drift-is-a-timeline-of-spans.md)'s **`revealed`** |
| Pre-release exemption | **None.** The aperture rule is *vacuous* before the first install, not waived |
| A "correction" | **Not a kind of change** — a removal plus an addition, each priced separately |
| 161/tcp, 623/tcp | **Removed.** Re-adding 623/tcp later is a new widening, not a reversal |
| UDP list | [#4](https://github.com/winniel123/verge-asm/issues/4) §2.5's enumerated opt-in list is **superseded** |
| UDP capability | **Unchanged** — off by default; a batch measuring no UDP records no UDP pairs in scope |

## Rationale

### The re-key is a no-false-absence fix; the invariant is the small half

Re-keying is usually argued from the invariant: a containment check over bare integers cannot see
the difference between 161/tcp and 161/udp, so the check passes while the bug survives. True, and
not the reason that matters.

[ADR-0005](./0005-scan-execution-model.md) makes a `Batch`'s recorded scope the thing that
**licenses silence to count as evidence of absence**. A batch that records its scope as `{161}`
asserts an absence over a transport it never touched. That is the no-false-absence rule broken at
the record — the same failure as the dead-lettered batch recording "attempted 140 ports", arriving
through the key rather than through the failure path. The invariant is a build convenience; the
scope record is a correctness property, and it is why the re-key may not stop at the list file.

Nothing in the domain model changes. `Service` was already the triple.

### `verge-core` was already a union; it was implicit and hand-maintained

Look at what the four omissions have in common. 512/tcp ranks 239th by open-frequency; 4369, 25672
and 27019 rank nowhere at all. **None of them belongs in the hot set on the hot set's own evidence
standard.** They belong there only because the sensitive list forces them.

So `verge-core` is not, and never was, a frequency-derived set. It is a union of a frequency set
and a normative one, with the union left implicit and reconstructed by hand every time either side
moves — which is precisely how the four went missing, and the 25672 row shows the mechanism
exactly: the frequency half selected RabbitMQ's *popular* ports (5672, 15672), the normative half
selects its *indefensible* one (25672), and they are disjoint. Somebody had to notice. Nobody did.

Writing the union down changes nothing about what ships and everything about who maintains it.

### A definition beats a test, for the reason this project prefers structure everywhere

[#21](https://github.com/winniel123/verge-asm/issues/21) §6 specified a build-time test. A test is
a good mechanism and it is strictly weaker than the alternative available here: a test can be
skipped, deleted or never written, and it protects the invariant only where CI runs. A **definition
cannot fail**, because the violating state is not expressible.

This is the same move made twice already —
[ADR-0007](https://github.com/winniel123/verge-asm/blob/adr-0007-drift-model/docs/adr/0007-drift-is-a-timeline-of-spans.md)'s
`Break` enforcing comparison legality structurally rather than by discipline, and
[ADR-0008](https://github.com/winniel123/verge-asm/blob/adr-0008-derivation-versions/docs/adr/0008-derivation-versions-move-on-content.md)'s
golden corpus making a version bump mechanical while leaving the judgement human. A union is the
same shape one level down.

**It does not re-open the laundering [#21](https://github.com/winniel123/verge-asm/issues/21)
closed.** That objection was to deriving the sensitive list *from* the hot set, which would make
frequency a precondition of normativity. The union runs the other way: normativity forces
probe-scope, which is what "cost yields to correctness" already licensed. The two source lists keep
their separate evidence standards, their separate governance and their separate revision triggers —
only the derived set is new.

### The pre-release exemption has no work to do

The ticket offered one: v1 has not shipped, so the hot set is not yet frozen and the additions cost
nothing. It also named the hazard — that argument stops being available exactly once, and nobody
ever writes down when.

The exemption is unnecessary. `revealed` is a property of a **timeline that already exists**, and
before the first install there are none. So the aperture rule is not *waived* pre-release, it is
**vacuous** — and vacuity needs no policy, no freeze date, and cannot be quietly extended, because
the moment an install has a timeline the rule binds by its own terms with nothing to repeal. Taking
the exemption would have meant authoring a freeze policy with exactly one user, in order to avoid
using a mechanism [ADR-0007](https://github.com/winniel123/verge-asm/blob/adr-0007-drift-model/docs/adr/0007-drift-is-a-timeline-of-spans.md)
already built and already prices at zero: port tiers are one of its three named aperture inputs,
alongside enabled sources and the ownership gate, and all three yield `revealed`.

State it as vacuity, not as a grace period. The distinction is the whole safety of it.

### "Correcting a mis-aimed port" is a category defined by intent

The ticket proposed a third kind of change — *161/tcp was never measuring SNMP, so fixing it does
not widen what we can see of the world; it fixes a measurement that was mis-aimed* — and called it
the first change that alters the port set without altering the aperture.

The category is refused. Once the aperture is keyed on pairs, 161/tcp and 161/udp are two different
keys over two different timelines, and 623/tcp is a genuinely different protocol from 623/udp on
IANA's own registry. The fix decomposes without residue:

- **A removal.** 161/tcp and 623/tcp leave the set. Their timelines stop being fed and open a `Gap`
  under ADR-0007's currency rule — which is the honest rendering: *we stopped looking*. Nothing is
  withdrawn, because ceasing to measure is not measuring absence, and nothing is re-derived, because
  history never is.
- **An addition.** 161/udp and 623/udp enter. Plain `revealed`.

Nothing is left over for a third category to hold. The intuition behind it is about **our intent**;
the model keys on **subjects**. Admitting a change-kind defined by intent is the door ADR-0007
closed when it made `Break` structural — and "it's just a fix" is a permanently available feeling,
which is why the rejection is recorded here rather than left to be re-derived.

### The two TCP siblings leave on their own merits

Because the fix is a removal plus an addition, the removal needs its own justification rather than
riding along on the addition's.

161/tcp is registered but is not where SNMP agents listen. 623/tcp is `oob-ws-http`, a real DMTF
protocol and a legitimate candidate on the frequency half's terms. But **neither was ever selected
on those terms**: both entered through the Management/OOB supplement, whose stated rule is that
each member "maps to a named v1 risk signal" — and after the transport fix, neither does. 623/tcp
is re-addable later on a fresh frequency or signal argument; nobody has made one, and making one
would be a new widening rather than a reversal of this.

### What the operator loses, and why that is the right loss

`verge-core` is computed, so it is not a file. The operator edits the **frequency half**; the
sensitive half is a shipped signal's reference data, release-coupled under
[ADR-0004](./0004-signals-are-release-coupled-rules.md), and not a file they reach.

The cost is real: the operator can no longer say *"don't probe 4369"*. That is deliberate, and it
has a precedent. [ADR-0006](https://github.com/winniel123/verge-asm/blob/adr-0006-subject-lifecycle/docs/adr/0006-subjects-leave-by-measurement.md)
ruled that removing a subject from the estate belongs to `Seed`, never to `Annotation`, because
operator opinion may not do the job measurement does; and
[#22](https://github.com/winniel123/verge-asm/issues/22) refused suppression of coverage gaps for
the same reason. **A port the operator can hide is a signal the operator can silence** — #22's
refused suppression arriving through the port list. If the operator genuinely needs a host left
alone, the instrument is the scope declaration, not the port set.

The same construction removes the last enforcement point that would otherwise have been needed. An
operator-editable `verge-core` could break the invariant after release, which would have wanted a
config-load check; a computed one cannot.

### UDP is a transport capability, not a list

[#4](https://github.com/winniel123/verge-asm/issues/4) §2.5 put UDP off by default on measured
signal-to-cost grounds — even a 1,000-port UDP scan misses half of what is open, and
`open|filtered` resolves only by timeout. That decision stands and is not revisited here.

But it was expressed as a **hand-picked list** — `53, 123, 161, 500, 623, 1900, 5353` — and
measured against the sensitive list that list is wrong in the same way `verge-core` was: it covers
161 and 623 and misses **69/udp, 137/udp, 138/udp and 11211/udp**, four sensitive pairs. Under the
union it is superseded rather than amended: the UDP leg is `verge-core`'s UDP pairs, of which the
sensitive half contributes six and the frequency half contributes the residue
`53, 123, 500, 1900, 5353`. Nobody maintains a UDP list by hand again.

Separating the two ideas is what makes this work. Membership of `verge-core` is a *definition*;
whether a given `Batch` measures a pair is the *prober's* business. A batch that measures no UDP
records a scope containing no UDP pairs, and `not-evaluable` falls out of ADR-0004's existing rule
with no new mechanism — which is exactly what
[#21](https://github.com/winniel123/verge-asm/issues/21) §6.1 asked for, obtained for free rather
than by policy. Conflating capability with membership is what produced the 161/tcp bug in the first
place.

## Consequences

- The `sensitive ⊆ hot` test [#21](https://github.com/winniel123/verge-asm/issues/21) §6 specified
  is **not written**. A reader finding §6 and no test in the codebase should read this ADR: the
  invariant was not dropped, it was made unfalsifiable.
- Adding a pair to the sensitive list now does two things at once — bumps
  `sensitive-port-exposed`'s rule version (ADR-0004/ADR-0008) **and** widens the aperture
  (`revealed`). They compose cleanly, landing on different objects: the signal's version governs
  comparability, the new `Service` timeline starts fresh. Neither produces a false transition.
- Revising the **frequency half** still manufactures no drift, per ADR-0007 — a batch whose
  recorded scope excludes a port does not touch that timeline.
- The six UDP pairs remain `not-evaluable` on default settings, by design and visibly. **The
  product currently has nowhere to show that**: `Coverage` as prototyped in
  [#28](https://github.com/winniel123/verge-asm/issues/28) renders source coverage and vantage
  state per `Seed` scope and nothing below it, so
  [#21](https://github.com/winniel123/verge-asm/issues/21) §6.1's stated justification for keeping
  the rows — that it makes the gap visible in the product rather than invisible in a list file — is
  unbacked. This ADR does not repair it; it is open as a separate ticket, and it is not confined to
  UDP.
- The **governance** question is untouched and stays fog on the map: who revises either source
  list, on what trigger. The union changes who maintains the *derived* set (nobody) and nothing
  about who maintains the inputs.
- [`CONTEXT.md`](../../CONTEXT.md) needs no change. No term is added and none is amended.

### Amendment — [#66](https://github.com/winniel123/verge-asm/issues/66): the sensitive half loses `161/udp`, so the union narrows

The Decision is unchanged: `verge-core` is still `frequency-set ∪ sensitive-list` and still a
definition rather than a list. What moves is an **input**, and with it three figures written above.

[#66](https://github.com/winniel123/verge-asm/issues/66) retrieved the SNMP standards family
(RFC 3411, 3412, 3413, 3414, 3417, 3584, 3410, 2570, 1157 and 6353) and net-snmp's own documentation,
found **no first-party sentence placing SNMP inside a network boundary**, and removed `161/udp` from
the sensitive list — [`sensitive-ports.md`](../research/sensitive-ports.md) §11. The list is **37**
pairs.

- **The UDP arithmetic above is superseded.** *"the sensitive half contributes six"* UDP pairs is now
  **five** — 69/udp, 137/udp, 138/udp, 623/udp, 11211/udp — and *"The six UDP pairs remain
  `not-evaluable` on default settings"* reads **five**. (That sentence is separately corrected by
  [#44](https://github.com/winniel123/verge-asm/issues/44), which found they hold no subject at all;
  the count moves either way.)
- **The worked example in *"Correcting a mis-aimed port"* is superseded in one half.** *"An addition.
  161/udp and 623/udp enter. Plain `revealed`"* — **`161/udp` does not enter.** The frequency half is
  TCP-only ([#4](https://github.com/winniel123/verge-asm/issues/4) §2.5) and this ADR attributes 161
  to the sensitive half, so `161/udp` is not in `verge-core` on either leg. `623/udp` is unaffected.
  The removal half — 161/tcp and 623/tcp leaving — is unaffected, and the decomposition the example
  exists to demonstrate is unaffected.
- **The price is the one this ADR already named, and it is vacuous.** Removing a pair from the
  sensitive list bumps `sensitive-port-reached-from-internet`'s rule version
  ([ADR-0008](./0008-derivation-versions-move-on-content.md)) and **narrows** the aperture. Before the
  first install both are vacuous — *not waived*, per the pre-release section above — which is the whole
  reason #66 was worth running before v1 rather than after.

### Amendment — [#91](https://github.com/winniel123/verge-asm/issues/91): the sensitive half gains `10259/tcp` and `10257/tcp`, so the union widens

The Decision is unchanged again, and this is the **first widening**. Every prior move to the sensitive
half has been a removal ([#66](https://github.com/winniel123/verge-asm/issues/66)) or no move at all.

[#91](https://github.com/winniel123/verge-asm/issues/91) disposed of the three candidates
[ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 2 obliged
[`sensitive-ports.md`](../research/sensitive-ports.md) §19.8 to ticket rather than admit. `10259/tcp`
kube-scheduler and `10257/tcp` kube-controller-manager are **admitted** on Claim 3; `10256/tcp`
kube-proxy is **refused** — §24. The list is **39** pairs.

- **The union widens by two members**, both in the sensitive half, both **TCP**. The UDP arithmetic
  above is **untouched** and still reads five.
- **§6's one-directional invariant fires for the first time in earnest.** Neither new pair is in the
  frequency half, so *every pair on the sensitive list MUST be a member of the hot set* forces **two
  hot-set additions**. This is the coupling direction this ADR chose, spending itself exactly as
  designed: probe cost yields to correctness. The addition must be checked against
  [#4](https://github.com/winniel123/verge-asm/issues/4) §2.3's modern-services supplement at merge —
  §24.10 flags the membership claim as asserted rather than measured.
- **The price is real this time, and it is the pre-release price.** Adding a pair bumps
  `sensitive-port-reached-from-internet`'s rule version
  ([ADR-0008](./0008-derivation-versions-move-on-content.md)), `Break`s every evaluation and
  **widens** the aperture. Before the first install all three are vacuous — *not waived* — which is
  again the whole reason the ticket was run before v1 rather than after.
- **The *"Correcting a mis-aimed port"* worked example is unaffected**: it turns on a transport
  mismatch and neither new pair has a UDP sibling in either half.

### Amendment — [#95](https://github.com/winniel123/verge-asm/issues/95): the sensitive half gains `10249/tcp` and `10248/tcp`, so the union widens a second time

> **`verge-core = frequency-set ∪ sensitive-list` is unchanged. The sensitive half gains two more TCP
> pairs — `10249/tcp` kube-proxy metrics (Class A) and `10248/tcp` kubelet healthz (Class C) — and
> `10258/tcp` cloud-controller-manager is refused and adds nothing.** The list is **41 pairs**, class
> totals `12 / 7 / 22`. [`sensitive-ports.md`](../research/sensitive-ports.md) §27.

- **The definition does not move and neither does the arithmetic's shape.** This is the second
  widening in two sections, and it is the same act: the sensitive half is edited, the union follows
  by construction, and nothing has to be remembered.
- ~~**§6's one-directional invariant forces two more hot-set additions**~~, taking the missing-TCP group
  from six to eight and §6.1's containment arithmetic to `28 + 8 + 5 = 41`. The UDP half is untouched
  at five, both new pairs being TCP.
  > **The invariant clause is WITHDRAWN, on the same ground the #97 amendment below withdraws #91's.**
  > #95 was resolved concurrently with #97 and inherited the mechanism from #91's amendment above.
  > **This ADR dissolved it**: nothing fires and nothing is forced, and `10249/tcp` and `10248/tcp` were
  > inside `verge-core` from the instant §27 admitted them.
  > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) is why the
  > withdrawal is made here rather than only at #97's amendment. **The arithmetic is untouched** — the
  > missing-TCP group really is eight and `28 + 8 + 5 = 41`; what is withdrawn is the mechanism that
  > sentence names.
- **§24's flagged membership check is discharged rather than passed on again.** The amendment above
  left *"the addition must be checked against [#4](https://github.com/winniel123/verge-asm/issues/4)
  §2.3's modern-services supplement at merge"*. **[measured]**
  [`safe-active-probing.md`](../research/safe-active-probing.md) §2.3's supplement reads, for this
  family, *"Orchestration/control planes: 2375, 2376, 2379, 2380, 6443, 10250, 10255"*, and none of
  `10248`, `10249`, `10257`, `10258` or `10259` appears in it or in nmap's top-100. §24's *four → six*
  is confirmed and this pass's own additions take it to eight.
- **The price is spent a second time**, and knowingly: `sensitive-port-reached-from-internet`'s
  content moves, its rule version bumps under
  [ADR-0008](./0008-derivation-versions-move-on-content.md), every evaluation `Break`s and the
  aperture widens. Still pre-install, so still vacuous and still not waived.
- **Four pairs in two sections, all Kubernetes component ports, all found by reading one file whole.**
  Recorded here as well as at §27.13 because the union is where a curator would notice a single
  owner's artefact being mined harder than any other's, and the concentration is real even though the
  four rows carry four different footings.

### Amendment — [#97](https://github.com/winniel123/verge-asm/issues/97): the membership claim is measured, and the mechanism above it is withdrawn from this ADR's own body

The Decision is unchanged for the third time. What moves is the **evidence** under #91's amendment and
one **sentence inside it** — written in this ADR, and contradicting the Decision table two screens up.

[#97](https://github.com/winniel123/verge-asm/issues/97) enumerated the frequency half from
[#4](https://github.com/winniel123/verge-asm/issues/4) §2.3 and tested all 39 sensitive pairs against
it — [`sensitive-ports.md`](../research/sensitive-ports.md) §29. **The list is 41 as composed**, and
#95's two are measured against the same limb with the same answer (§27.12): neither is in the
frequency half.

- **The membership claim is confirmed and is now measured.** **[measured]** The frequency half is
  **123 ports, all TCP** — 81 retained from nmap's top-100 after 19 deletions, plus 44 net-new from
  the supplement, less this ADR's own removal of `161/tcp` and `623/tcp`. §2.3's orchestration limb is
  `2375, 2376, 2379, 2380, 6443, 10250, 10255` and stops there, so **neither `10259/tcp` nor
  `10257/tcp` is in the frequency half**. #91's *"neither new pair is in the frequency half"* holds,
  and *"the addition must be checked against #4 §2.3's modern-services supplement at merge"* is
  **discharged**. Independently re-derived, and it reproduces
  [`nmap-services-licence.md`](../research/nmap-services-licence.md) §6.2's count exactly.
- **`verge-core` is 134 pairs** — 123 from the frequency half plus the 11 the sensitive half
  contributes alone (6 TCP: `512`, `4369`, `25672`, `27019`, `10259`, `10257`; 5 UDP: `69`, `137`,
  `138`, `623`, `11211`). 129 of those are probed on default settings, UDP being off.
  > **Composed with the #95 amendment above, which landed in the same merge: `verge-core` is 136
  > pairs** — 123 from the frequency half plus the **13** the sensitive half contributes alone (**8
  > TCP**: the six named here, `+ 10249`, `+ 10248`; 5 UDP: unchanged). **131** are probed on default
  > settings. **The frequency half does not move** — neither new pair is in it, measured against the
  > same orchestration limb. This is the figure to quote; `134` is #97's against the pre-#95 list.
- **The clause *"§6's one-directional invariant fires for the first time in earnest … forces two
  hot-set additions"* is WITHDRAWN.** It describes the mechanism **this ADR dissolved**: the Decision
  table reads *"The invariant: **Dissolved**"* and *"Enforcement: **None, anywhere**"*, and the
  Consequences say the test *"is **not written**"*. Nothing fires and nothing is forced. **The union
  widens by two because the sensitive half gained two members the frequency half does not carry** —
  which is what the Decision's *"Derived: `frequency-set ∪ sensitive-list`"* row already says, and it
  needs no invariant to say it. `10259/tcp` and `10257/tcp` were inside `verge-core` from the instant
  §24 admitted them; there was never a separate addition to perform or a gate to pass.
- **The widening, the price and the coupling direction are all unaffected.** *Cost yields to
  correctness* still names why the union runs this way. **[measured]** the yield is **+1.6 %** of the
  daily tier's per-host probe budget — 127 → 129 TCP pairs, 19.0 s → 19.3 s per pass on a dropping
  host under §6.3's caps; **composed with #95's two, 127 → 131 TCP pairs, 19.0 s → 19.6 s, +3.1 %** —
  with no change to technique, to the zero-added-capabilities budget, or to
  the concurrency bound that governs the middlebox hazard. [ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)
  does not bite: both pairs join a configured `Scan` at a daily cadence, so they have currency.
- **This ADR is the second site where the dissolved mechanism was written forward**, the first being
  [`sensitive-ports.md`](../research/sensitive-ports.md) §6, which was never amended and which #91
  read. The Consequences section anticipated a confused reader and answered with a **pointer** —
  *"A reader finding §6 and no test in the codebase should read this ADR"* — which reaches only a
  reader who arrives holding the ADR. Two passes did not, and the second carried the mechanism into
  [#97](https://github.com/winniel123/verge-asm/issues/97)'s own stated stakes, one hop from the spec.
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) is the rule
  that forces the withdrawal to happen at the superseded site instead. §6, §6.1, §6.3, §7.1 and §1's
  summary row are amended in place.

## Alternatives rejected

**Keep the lists independent with a build-time test** — [#21](https://github.com/winniel123/verge-asm/issues/21)
§6's answer. Rejected on failure mode, not on cost: a test is a mechanism that can be absent, and
the state it guards against is one a definition simply cannot express. The union preserves
everything §6 argued for — separate evidence standards, separate governance, coupling that runs
from correctness to cost — and removes only the part that had to be remembered.

**Shrink the sensitive list instead**, dropping the four unprobed pairs rather than widening the
hot set. This inverts §6's coupling direction and makes probe cost a precondition of normativity.
Settled by [#21](https://github.com/winniel123/verge-asm/issues/21) and not re-opened.

**Treat the hot set as unfrozen until v1.0.0.** Rejected above: it authors a policy with one user
to avoid a mechanism that already exists and already costs nothing, and it converts a structural
fact into a promise that has to be revoked on a date nobody will write down.

**A `not-probed` state distinct from `not-evaluable`.** Considered for the six UDP pairs and
rejected — it is a new lifecycle state for a condition that already has an observed account
(ADR-0006's habit: check whether the thing already has a value before inventing a state). The
`Batch` scope record already says the pair was not measured, and ADR-0004 already names the
outcome.
