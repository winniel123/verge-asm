# ADR-0058: A superseded mechanism is withdrawn at the site that specifies it, not only at the site that supersedes it

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#97 Is the hot set actually a superset? `10259` and `10257` were asserted into it, never measured](https://github.com/winniel123/verge-asm/issues/97)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`sensitive-ports.md`](../research/sensitive-ports.md) §6 specified the relationship between the two
port lists as *"a one-directional invariant enforced at build time … with a test that fails the build
otherwise"*.

[ADR-0009](./0009-verge-core-is-a-union.md) dissolved that mechanism eleven months of tickets ago. Its
Decision table carries two adjacent rows — *"The invariant: **Dissolved.** `sensitive ⊆ verge-core`
holds by construction"* and *"Enforcement: **None, anywhere** — no build-time test, no config-load
check, no runtime check"* — and it made `verge-core` the union `frequency-set ∪ sensitive-list`, so
containment became **analytic**: `S ⊆ F ∪ S` is true of every `S` and every `F`, and the violating
state is not expressible. The ADR chose a definition over a test **on failure mode**, not on cost:
*"a test is a mechanism that can be absent, and the state it guards against is one a definition simply
cannot express."*

**ADR-0009 saw the reader this would strand, and answered with a pointer.** Its Consequences open:

> The `sensitive ⊆ hot` test [#21](https://github.com/winniel123/verge-asm/issues/21) §6 specified is
> **not written**. A reader finding §6 and no test in the codebase should read this ADR: the invariant
> was not dropped, it was made unfalsifiable.

§6 itself was left standing, unamended, in the present tense.

**Two passes then read §6 and wrote its mechanism forward.** This is the forcing measurement, and it
is a chain rather than an instance:

1. **[#91](https://github.com/winniel123/verge-asm/issues/91)** admitted `10259/tcp` and `10257/tcp`
   and wrote, **into ADR-0009's own body** as an amendment: *"**§6's one-directional invariant fires
   for the first time in earnest.** Neither new pair is in the frequency half, so *every pair on the
   sensitive list MUST be a member of the hot set* forces **two hot-set additions**."*
   [`sensitive-ports.md`](../research/sensitive-ports.md) §24.12 restated it. The mechanism was
   re-asserted inside the document that dissolved it, two screens below the row dissolving it.
2. **[#97](https://github.com/winniel123/verge-asm/issues/97)** inherited it at one further hop and
   stated the stakes as *"an addition to the sensitive list that is not in the hot set does not
   silently degrade, it **fails the build**"* — pricing itself as blocking
   [#12](https://github.com/winniel123/verge-asm/issues/12) on *"handing implementation a build-time
   invariant with an unverified term is handing it a build that may not go green."*

The third hop reaches code. #12 is the spec; carried forward, it instructs implementation to write the
build-time containment test ADR-0009 refused — a skippable guard over an unfalsifiable property, which
would also invite a later reader to conclude the union is maintained by the test rather than by the
definition.

The pointer did its job for readers who arrived holding the ADR. Both of these arrived holding §6.

**A second instance surfaced in the same pass, on a different pair of documents.**
[`safe-active-probing.md`](../research/safe-active-probing.md) §2.3's Management/OOB limb still reads
`161 (TCP), 623` — the exact two members ADR-0009's Decision table removed. A session enumerating the
frequency half from §2.3's own text gets **125** and never learns otherwise;
[#78](https://github.com/winniel123/verge-asm/issues/78) caught it only because it happened to be
counting for a licence question, and recorded the corrected **123** in a note nothing points to from
here. Same shape, different documents, no ticket in between.

## Decision

**When a decision supersedes a mechanism, the document that *specifies* the mechanism is amended in
the same change.** A note in the superseding document does not discharge the obligation.

| Concern | Decision |
| --- | --- |
| Where the withdrawal is written | At the **superseded** site, in the same change — and at the superseding site too, which is where the reasoning lives |
| What the superseded site gets | A **withdrawal**, naming what no longer holds and pointing at the decision — never a silent deletion, per the note's existing name-and-withdraw convention |
| Whether a pointer from the superseding document suffices | **No.** It reaches only a reader who already holds it, which is the reader who did not need it |
| Scope | Any **mechanism** — a test, a gate, a check, an enforcement point, a procedure. Not figures, which already have `FIGURE DELTA`, and not claims about the world, which have the amendment convention |
| What counts as discharge | The superseded sentence can no longer be read, in isolation and in the present tense, as specifying something that exists |
| Who does it | The pass that supersedes. It is the last party who knows both states |

**The test for a reader:** if the superseded sentence, read alone and out of context, would cause a
competent session to build or specify the thing — it is not withdrawn.

## Rationale

### A pointer is addressed to the reader who does not need it

ADR-0009's *"a reader finding §6 and no test in the codebase should read this ADR"* is conditioned on
noticing an absence. But the failure mode is not a reader who looks for the test and cannot find it;
it is a reader who **believes the test exists and never looks**, because §6 says so in the present
tense and there is no reason to doubt it. #91 did not go looking for a test — it wrote *"the invariant
fires"* and moved on, correctly by §6's lights.

This is the same asymmetry [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)
already relies on one level down: an obligation discharged by *the reader will check* is discharged
only for readers with a reason to check.

### It is the *structural* move this project makes everywhere, applied to prose

ADR-0009 preferred a definition to a test because *"a definition cannot fail, because the violating
state is not expressible"*, and cited
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s `Break` and
[ADR-0008](./0008-derivation-versions-move-on-content.md)'s golden corpus as the same move. This ADR
applies it to the documents: a superseded sentence that has been **struck** cannot be read forward,
while one that has merely been **contradicted elsewhere** can be, and was, twice.

Amending the superseded site is not a documentation-hygiene preference. It is the only version of the
withdrawal whose violating state is not expressible.

### The cost is small and lands on the party who can pay it

The pass that supersedes holds both states in hand — it has just read the old mechanism in order to
replace it. Every later reader holds only one. Deferring the edit does not avoid it; it relocates it
to a session that must first discover the discrepancy, which is what
[#97](https://github.com/winniel123/verge-asm/issues/97) spent most of its budget on and what
[#78](https://github.com/winniel123/verge-asm/issues/78) did by accident.

### It does not license rewriting history

The withdrawal **names** what no longer holds; it does not delete the record. That is the convention
[`sensitive-ports.md`](../research/sensitive-ports.md) already uses everywhere — §3's `161/udp` row is
*"left standing … marked here rather than deleted, per the name-and-withdraw convention"* — and the
reason is unchanged: a deleted sentence takes its reasoning with it, and a reader who arrives via an
old citation finds nothing rather than a redirection. What this ADR adds is that the marking is
**owed**, and owed at the superseded site, rather than being available to whoever notices.

## Consequences

- **[`sensitive-ports.md`](../research/sensitive-ports.md) §6, §6.1, §6.3, §7.1 and §1's summary row
  are amended in place** by §29, and ADR-0009's #91 amendment has its *"forces two hot-set
  additions"* clause withdrawn. [`safe-active-probing.md`](../research/safe-active-probing.md) §2.3
  is annotated for ADR-0009's two removals.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) must carry the union definition and
  §29.3's decomposition, and must not carry a build-time containment test.** This is the consequence
  that reaches code, and it is a subtraction from what #97 was priced to deliver.
- **The map's curation triggers gain nothing.** This is not a watch — it creates no obligation to
  re-read anything on a cadence, which is the shape
  [ADR-0046](./0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
  refused on cost. It is a **step in a change that is already happening**, discharged when that
  change lands.
- **It is detectable rather than watched, and cheaply**, because both sides are written down: a
  document specifying a mechanism, and an ADR dissolving it. It joins the map's *detectable defects*
  group alongside *a row whose footing tier disagrees with its class*
  ([#83](https://github.com/winniel123/verge-asm/issues/83)) rather than its trigger list.
- **The existing corpus is not swept.** Two instances are known and both are repaired here. Whether
  others exist is unmeasured, and a sweep is not opened — it would be a read of every superseding
  decision in the repository against every document it names, which is exactly the obligation the
  bullet above declines. It is ticketed as a bounded question instead.
- [`CONTEXT.md`](../../CONTEXT.md) needs no change. No term is added and none is amended.

## Alternatives rejected

**Leave the pointer and rely on the reader** — ADR-0009's own choice, made explicitly and in good
faith. Rejected on its measured failure rate: two of the passes that met it wrote the mechanism
forward, one of them into the superseding ADR's own body. The pointer is not wrong; it is addressed
to a reader who has already noticed the problem, and the failing readers had not.

**Delete the superseded sentence.** Rejected on the same ground the note's name-and-withdraw
convention was adopted: a deletion takes the reasoning with it, breaks inbound citations, and leaves a
later reader unable to tell a withdrawal from an omission. §6's argument for *why the coupling points
this way* is still live and still correct; only its final clause about enforcement is not.

**Make it a curation trigger on the map.** Rejected because a trigger implies a watch, and the watch
here would be *re-read every superseded document forever* — the standing obligation ADR-0046 exists to
deny, and the ground on which #93's ninth trigger lost. The discharge point is a change that is
already in flight, so it costs nothing to attach it there and everything to attach it to a clock.

**Restrict it to ADRs.** Rejected because the load-bearing instance runs the wrong way: the superseded
site here is a **research note** (§6) and the superseding site is an ADR, and the second instance is
note-to-ADR as well ([`safe-active-probing.md`](../research/safe-active-probing.md) §2.3 against
ADR-0009). A rule that only reaches ADRs would have caught neither.
