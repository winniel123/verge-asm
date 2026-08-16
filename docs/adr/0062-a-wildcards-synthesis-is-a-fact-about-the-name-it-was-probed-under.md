# ADR-0062: A wildcard's synthesis is a fact about the `Name` it was probed under, and it is a facet of its own

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#103 Is a zone's measured wildcard poison signature a value the model holds, and on what subject?](https://github.com/winniel123/verge-asm/issues/103)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Discharges:** [ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md)'s ticketed residue

## Context

The measurement already happens, every batch, on every wildcarded zone.
[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §3.2 calls it *mandatory,
not optional*: query 3–5 long random labels under a domain, and if they answer, **record the wildcard
answer set as a poison signature**. *(Both halves of that sentence have since been superseded and
are quoted here as they stood: the **signature** by [ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md) — there is no*
the *answer set, only a per-component union of three — and the **labels** by [ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md) — ~~five~~ **nine** random plus one structured, each exactly one label, the count raised by [#115](https://github.com/winniel123/verge-asm/issues/115).)* `wildcard-discrimination` is one of
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s five named prober leaves, and
[ADR-0011](./0011-a-facet-is-six-parts.md) rules that deciding `Shadowed` takes **two measurements
inside one batch** — the name's own answer, and that signature.

So the signature is measured on every cadence and the model **holds it nowhere**. It is an input to
a decision and never a value on a timeline.

The consequence is precise. A wildcard being **repointed** is completely invisible. Every name
beneath it stays `Shadowed` across the move, which is deliberate and correct and is exactly the drift
suppression [`CONTEXT.md`](../../CONTEXT.md)'s `Shadowed` was built for: *"repoint one wildcard and
every fictional name beneath it reports a resolution change the same night."* But *your wildcard now
points somewhere else* is one fact about one thing, and today no message and no row carries it. The
suppression was aimed at the fictional names; it swallowed the real one alongside them.

[ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md) is what makes
this a question rather than a shrug, because it removed the obvious answer. *Hold it on the
`*.example.com` `Name`* is unavailable and will stay unavailable: a wildcard denotes a set of names
rather than a name, so it is a `Subject` from no source. ADR-0060 flagged the residue in its own
thinness section — *"the content is not lost … but nothing holds it"* — and ticketed it rather than
absorbing it.

## Decision

**A wildcard's measured synthesis is a fact about the `Name` the control labels were generated under.
It is a facet of its own and never a value on `dns-record`, it admits no subject, and it does not
ship in v1.**

| Concern | Decision |
| --- | --- |
| Is the signature a value the model **holds**? | **Yes in the model; no in v1.** The fact has a well-formed carrier, and the carrier is deferred rather than refused |
| Which subject does it hang on? | **The `Name` the control labels were generated under** — the name whose immediate child space the wildcard synthesises into |
| Is that *the zone apex*? | **No.** *Apex* is what that name happens to be today, under §3.2's procedure; the rule is *the name we probed under*, which is a measured fact the `Batch` records |
| Why the rule rather than the word | [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) already called *apex* **"a heuristic standing in for a rule"** once. The rule stated here survives any change to which names get probed; *apex* would not |
| A new `Facet`, or a value on `dns-record`? | **A new facet.** `dns-record` is *what an authority served for a qtype* **for this subject**; the signature is what it serves for names that are not subjects at all |
| The decisive objection to `dns-record` | **The `Span` key collides.** `(subject, facet, discriminator, vantage, source)` with discriminator = qtype is already occupied by the A RRset the authority served for this very name |
| Its discriminator, if it ships | **The qtype** — which is a reason it is **not** `dns-record`, not a reason it is: two facets with one discriminator each is the model's shape, one facet with a two-part discriminator is not |
| Its name, if it ships | **`wildcard-synthesis`.** Never `poison-signature`: that names **our instrument** and borrows a security taxonomy's word, and the project has chosen the measured word over the taxonomy's four times running (`Lame`, `no-response`, `tls-acceptance`, `Shadowed`) |
| Its value space | ~~A closed union — at minimum *no synthesis* and *synthesised RRset*. **Not settled here**, and deliberately: see the sixth part below~~ — **SETTLED** by [#111](https://github.com/winniel123/verge-asm/issues/111) / [ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md), and it is **three** members rather than the two guessed here, drawn **per component** (`(qtype, RR type)`): **`NoSynthesis`** │ **`Determinate(RRset)`** │ **`Indeterminate`**. The guess was short because *the* synthesised RRset is not a well-formed object — **[measured]** an authority returns a different one for the **same** control label on consecutive queries |
| Does a synthesised RRset admit subjects? | **No, and never.** It cites no `Address`, opens no `Endpoint`, and writes no `Citation`. It is a value and only a value |
| What a repoint is | A **`Transition` on one timeline**, and nothing else |
| Is that `Transition` a message? | **No.** `dns-record`'s status verbatim under [ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md): recorded, and reaching nobody until a rule reads it |
| Does it re-manufacture the refused burst? | **No**, and by count: **one** subject, `qtypes × vantages` spans, and **zero** subjects admitted — against N fictional `Name`s each closing a `resolution` span and firing membership messages beneath them |
| `Shadowed` | **Untouched and not reopened.** Names beneath a wildcard report nothing when it moves, before this ruling and after it |
| ADR-0060 | **Untouched and not reopened.** `*.example.com` is a `Subject` from no source, and this ruling does not reach for it |
| Does it ship in v1? | **No.** Out of scope, priced at [ADR-0015](./0015-the-value-space-is-the-commitment.md)'s `revealed` plus one message with **no `Break`**, so there is no deadline and nothing to buy now |
| What blocks it, beyond price | ~~**Its sixth part cannot be written.** ADR-0011 requires a **batch-scope obligation naming what its silence covers**, and **nothing in the repository declares which `Name`s are control-probed**~~ — **DISCHARGED** by [#108](https://github.com/winniel123/verge-asm/issues/108) / [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md): the population is *the parents of the `Name`s in the batch's resolution scope*, recorded on the `Batch` by content, so the sixth part **is writable**. One new open item takes its place — see the amendment below |
| That gap in v1 | A **live defect independent of this facet**, since a wildcard nobody probed under makes every synthesised name beneath it read `Resolved` with a fictional address set. Ticketed as [#108](https://github.com/winniel123/verge-asm/issues/108), **blocking [#12](https://github.com/winniel123/verge-asm/issues/12)** |
| The `Derivation` vector | **Gains nothing.** No leaf, no version, no `Break` cause. `wildcard-discrimination` stays as ADR-0021 drew it and this ruling adds no leaf — the total is ~~five~~ **six** since [ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md) |
| Cost of the ruling | **Zero.** Nothing is held today, so nothing is withdrawn, no timeline closes and no re-baseline is owed |

## Rationale

### The subject is the name we probed under, and that is a rule where *apex* is a description

The ticket is right to be suspicious of *apex*, and right about why: ADR-0013's rejected-alternatives
table already called it *"a heuristic standing in for a rule"*, because Route 53 aliases and
Cloudflare flattening both apply **below** the apex, so the word missed the motivating case while
reading as coverage.

The word is avoidable here, and avoiding it is what makes the answer robust rather than lucky.

RFC 4592 §2.1.1 defines a wildcard as an owner name `*.X` — an ordinary owner name in a zone, whose
leftmost label is the octet pair `0x01 0x2a`. What it synthesises is answers for query names that
**fall off the tree below X**. So the thing measured is a property of **X's own name space**: *the
authority for X answers, for names beneath X that do not exist, with this*. X is the subject that
statement is about.

And X is not inferred. §3.2's procedure **names it in the act of measuring**: the control labels are
generated *under X*, so X is the one input the measurement takes, and the `Batch` records it as the
scope the probe covered. That is the difference between a rule and a heuristic. *The apex* is a claim
about where wildcards sit, and it is a claim §3.2 itself already qualifies — step 3 says *"repeat one
level down for each discovered sub-zone — wildcards can exist at any label depth."* **The name we
probed under** is a reading of what happened, and it stays correct whichever names
[#108](https://github.com/winniel123/verge-asm/issues/108) decides to probe.

~~X is also, by construction, a `Name` already in the estate. There is no admission step here and no
subject invented: we do not probe under names we do not hold, so the carrier exists before the
measurement is made.~~ **WITHDRAWN by [#108](https://github.com/winniel123/verge-asm/issues/108) /
[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md),
and falsified by measurement.** A probe site is a `Name`'s **parent**, and a parent need not be held
or even exist: **[measured]** `ns.iana.org` is NXDOMAIN and in no `Citation`, while `www.ns.iana.org`
is a held `Name` whose closest encloser is `iana.org` — so probing under `ns.iana.org` is both
correct and necessary. **The subject rule below is untouched** — the carrier is the name the control
labels were generated under — but it is no longer true that the carrier always exists, which is this
facet's one new open item. Nothing is admitted either way: a probe site is a label sequence we
construct, not a subject we cite. That is what separates this from ADR-0060's refused *fold it: strip the `*.` and
treat it as evidence about the parent* — that move **admitted** a subject from a pattern, and it lost
because pruning proves control rather than existence. Nothing is admitted here. A subject that is
already ours acquires a value.

The model has done exactly this before, and the precedent is `Lame`. `Lame` is a measured fact whose
immediate object is *the nameservers the parent zone delegates to*, and it is held as a value on the
`Name`. Partial lameness is held as an RRset of `(nameserver, serves │ does-not-serve)` pairs, again
on the `Name`. So a `Name` already carries facts about the machinery that answers for it, and not
only facts about its own records.

### It is not a value on `dns-record`, and the `Span` key is the proof

This is the closest call in the ruling, and it looks cheap from a distance. `dns-record` already
carries a **qtype discriminator**, a wildcard synthesises for **any** qtype, and that is the same
reason `Shadowed` lives on both `resolution` and `dns-record`. So the discriminator question is live,
as the ticket says, rather than incidental.

It resolves against `dns-record`, on two grounds that do not need each other.

**The key collides.** ADR-0011 fixed the timeline key as
`(subject, facet, discriminator, vantage, source)`, with `dns-record`'s discriminator the qtype. Put
the synthesis on `example.com`'s `dns-record`/A timeline and it lands on the key already holding the
A RRset the authority actually served for `example.com` — two different measured values, one key, and
neither one wrong. Repairing that means giving `dns-record` a **second discriminator dimension**,
which edits one of the six parts of a facet that already holds values across the estate, for the
benefit of a value that is not a reading of that subject's records at all. A new facet gets its own
discriminator for nothing.

**And the meaning is wrong before the key is.** `dns-record` is *what an authority served for a
qtype* — for **this** subject. The synthesised answer was served for `<random32>.example.com`, which
is a name we invented, which does not exist, and which is a subject nowhere. Recording it as
`example.com`'s `dns-record` asserts something about `example.com`'s records that is false. That is
ADR-0011's own discipline — *a decoder that helpfully normalises is an unversioned canonicaliser
wearing a parser's clothes* — one term over: **a value filed under a facet whose definition it does
not satisfy is a second facet wearing the first one's name.**

So it is a facet, and the pricing is ADR-0015's, unchanged: a wholly new facet is **strictly
additive**, costing `revealed` plus one coverage-class message with **no `Break`**. That is the same
shelf `http-acceptance` ([#54](https://github.com/winniel123/verge-asm/issues/54)), the
`listener-negotiation` facet ([#41](https://github.com/winniel123/verge-asm/issues/41)) and the
CT-fed `Name` facet ([#56](https://github.com/winniel123/verge-asm/issues/56)) already sit on, and it
is the shelf ADR-0060 nominated when it ticketed this.

### It admits nothing, and that is what keeps the refused burst refused

The ticket sets the burden correctly: *one value moving on one subject is not the estate-wide burst
that was refused — but the burden is to show that in terms, not to assert it.* In terms, then.

The burst `Shadowed` exists to prevent is **N fictional `Name` subjects each closing a `resolution`
span in one night**. N is unbounded in principle — it is however many names beneath the wildcard some
`Citation` admitted, a SAN list or a zone file — and it is not merely N rows. Each of those names
resolves to the wildcard's synthesised address set, so a repoint admits new `Address` subjects, which
yield `Service`s, which yield `Endpoint`s; membership fires at the root of each entering sub-tree
([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)) and the internet
`Reach` leg fires the flagship. The burst is a burst of **messages**, not of storage.

Against that, a `wildcard-synthesis` timeline on X:

- **One subject.** X, and no other. A repoint closes `|qtypes probed| × |vantages|` spans and touches
  no other timeline in the estate.
- **Zero subjects admitted.** This is the load-bearing half. A synthesised RRset is what an authority
  serves for names that **do not exist**, so nothing in it may write a `Citation`, admit an
  `Address`, or open an `Endpoint`. That is ADR-0060's rule read from the DNS side rather than the
  PKIX side: a construct that denotes a set of names admits none of them, whichever artefact carries
  it. The one difference is that here the model was never tempted, because the value hangs on a
  subject it already had.
- **Zero messages.** See the next section.

So the two shapes do not resemble each other arithmetically or in kind, and the constraint the ticket
set is met without argument by analogy: names beneath a wildcard report nothing when it moves, before
this ruling and after it.

### The channel question does not arise, and inventing one would be a `Signal`

If the facet ships, is its `Transition` a message on its own account?

**No**, and ADR-0026 answers it without a new judgement. *The facet layer is evidence and not a
channel*: a `Transition` is a message only where it is the **sole carrier of a fact the operator asked
for**, and in v1 exactly one facet transition qualifies — a `resolution` move that opens an `Endpoint`
no membership message covers. `dns-record` has **no channel at all**, so an MX or TXT change is
recorded and reaches nobody until a rule reads it. A repoint on `wildcard-synthesis` opens no
`Endpoint` — it opens nothing, by the section above — so it inherits `dns-record`'s status exactly.

That is worth stating plainly because it is the half of the ticket's motivation the facet does **not**
serve. Holding the signature gives the operator a **row**: a durable, queryable timeline on which
*your wildcard used to point here and now points there* is answerable, which is the map's Destination
in its own words — drift as a first-class, queryable object with a lifecycle. It does **not** give
them a message. Wanting one is a `Signal` question under
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s cadence test and a rule named for the fact
it reads, and it is a separate cost on a separate ticket. It is emphatically not a facet-level change
channel: [#64](https://github.com/winniel123/verge-asm/issues/64) already refused one for
`dns-record` as ADR-0007's named burst shape verbatim, and a second facet does not re-open it.

### It cannot ship, because its sixth part cannot be written — and that is a live defect anyway

This is the finding that decides *not in v1*, and it is stronger than the pricing argument, because
price alone would only say *no deadline*.

ADR-0011 fixed what a facet **is**: six parts, and the sixth is a **batch-scope obligation naming what
its silence covers**. It named the failure mode in terms, and named it as recurrent: *"a session adds
a seventh facet, writes a canonicaliser, and silently skips the batch-scope obligation — which is the
`{161}` defect arriving through a facet instead of a port list, for the third time."*

> **This whole subsection's premise is DISCHARGED by
> [#108](https://github.com/winniel123/verge-asm/issues/108) /
> [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md).**
> The population is declared — *the immediate parents of the `Name`s in the batch's resolution
> scope*, the **seventh aperture input**, recorded on the `Batch` by content — so the sixth part is
> writable and *not in v1* now rests on price alone, which ADR-0015 makes free. The survey of what
> was missing below is **dated, not permanent**, and reads as history.

That obligation ~~is unwritable today~~ **was unwritable when this was written**. **Nothing declares which `Name`s are control-probed.**
ADR-0021's parameter table — which
[`project-authored-constants.md`](../research/project-authored-constants.md) calls *"the authoritative
population"* — gives `wildcard-discrimination` exactly *control-label count and construction* and *the
match predicate*. [`measurement-offers.md`](../spec/measurement-offers.md) never mentions the control
probe.
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) declares EDNS options a
parameter of this leaf and says nothing about its population. The only statement anywhere is §3.2's
research prose — *the apex, then one level down for each discovered sub-zone* — which is a procedure
in a note, not a scope any `Batch` can record.

So the facet would ship with a hole in exactly the place ADR-0011 says facets get shipped broken, and
minting it now would be that failure with an ADR's authority behind it.

**The gap is not this facet's problem, though; it is v1's.** `Shadowed` is the value a `resolution`
observation takes *when the answer matches a wildcard's measured poison signature*. Where no signature
was measured under X, no name beneath X can be `Shadowed`, so every synthesised answer is recorded as
`Resolved` with a fictional address set — which admits fictional `Address`es, fictional `Service`s and
fictional `Endpoint`s. That is §3.2's stated catastrophe in full: *"any pipeline that treats 'resolves'
as 'exists' reports an unbounded, entirely fictional asset inventory."* RFC 4592 puts a wildcard at any
depth in a zone, and §3.2's procedure probes at two, so the shortfall is structural rather than
hypothetical.

That is filed as [#108](https://github.com/winniel123/verge-asm/issues/108) and priced as **blocking
[#12](https://github.com/winniel123/verge-asm/issues/12)**: handing implementation the machinery §3.2
calls mandatory, without saying which names it runs against, is handing it the aperture question that
decides whether the whole name pipeline is honest. It is by-catch of this ticket and it is worth more
than the facet was.

### Where this is thin, stated rather than smoothed

- **The value space is named and not settled, and that is a deliberate half-answer.** *A closed union
  of no-synthesis and synthesised-RRset* is the obvious shape and it is probably right, but ADR-0011
  is explicit that the value space is the expensive part and
  [ADR-0015](./0015-the-value-space-is-the-commitment.md) that widening one later costs a `Break` on
  every timeline of the facet. Settling it before the population is settled would be deciding the
  costly part first, so it is left open on purpose. A session that finds this ADR expecting a value
  space should read that as the ruling and not as an omission.
- **Whether the signature churns on its own is not measured, and it decides how much this is worth.**
  A wildcard pointing at a rotating CDN edge set would close a span every batch, making the timeline
  noisy for reasons that are the CDN's rather than the operator's. The map already names *the re-point
  on a churning CDN edge set* as one of **three unmeasured volumes**, so this is a known class and not
  a new one, and it is bounded here by the facet having no channel — noisy rows, never noisy messages.
  But nobody has counted it, and if the count is bad the facet is worth less than this ADR implies.
- **The `Shadowed` currency question is not reached.** If the facet ever ships, X's synthesis value and
  a name beneath X's `resolution` value are two timelines that must agree, and the model's answer for
  *how stale may one be when a rule reads the other* is `k` cadences plus, at the boundary,
  [ADR-0043](./0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md)'s clock
  class. Nothing here needs it, because `Shadowed` stays decided **inside one batch** by
  `wildcard-discrimination` and never assembled from two timelines — but a session that later builds
  the facet and reaches for the stored value to decide `Shadowed` would be undoing ADR-0011's rule,
  and it is flagged here because that is exactly the shape a stored value invites.
- **The zone file is a second potential source and v1 gives it nothing.** A zone export literally
  contains the `*` owner name and its RRsets, so this facet would be the model's **second** two-source
  facet and would give [ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md)'s conflict
  machinery a second `enumerable` pair. It is unavailable in v1 for a reason already ruled rather than
  a new one — [#48](https://github.com/winniel123/verge-asm/issues/48)'s line is that **a rule may read
  which names a zone contains, never what records it holds for them** — and it is noted so that
  whoever reopens that line knows a second thing arrives with it.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) changes in two entries and adds no term.** `Shadowed` records
  which subject the signature is a fact about, that v1 holds it nowhere, and that a repoint therefore
  reaches nobody; `Name` gains one clause pointing the existing sentence at the subject rule. **No
  term is added, no term changes meaning, no facet is added, and the facet list stays six.**
- **[ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md)'s ticketed
  residue is discharged, and ADR-0060 is confirmed rather than amended.** Its thinness entry — *"the
  wildcard's own content becomes invisible as a row, deliberately, and it is not measured how much
  that costs"* — is answered in both halves: the row has a named carrier, and the cost is priced and
  deferred. Its refusal of `*.example.com` is untouched and was not reached for.
- **[ADR-0011](./0011-a-facet-is-six-parts.md) is confirmed and used as a gate for the first time.**
  Its sixth part is what refuses this facet entry in v1, which is the first occasion the batch-scope
  obligation has stopped something rather than described it.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) is untouched.** Five leaves stay
  five, `wildcard-discrimination`'s declared parameters are unchanged, and no `Break` cause is added.
  A session will look here; there is nothing.
- **[ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md)'s enumeration is unchanged.**
  Exactly one facet `Transition` is a message in v1 and this ruling adds none — it adds no facet.
- **One new ticket, and it is the larger half of this one's output.**
  [#108](https://github.com/winniel123/verge-asm/issues/108): which `Name`s are control-probed, is
  that population a declared parameter or an aperture input, and what does a `Batch` record about it.
  **Blocking [#12](https://github.com/winniel123/verge-asm/issues/12)**, on the ground that
  `wildcard-discrimination` cannot be handed to implementation without it.
- **The `wildcard-synthesis` facet goes on the map's out-of-scope shelf with a stated reopening
  condition**, alongside `http-acceptance` and the `listener-negotiation` facet: **it reopens once
  [#108](https://github.com/winniel123/verge-asm/issues/108) fixes the control-probe population**,
  which is what makes its sixth part writable, and it costs `revealed` plus one message with no
  `Break` whenever it is taken up. Nothing is bought now and nothing needs rework later.
  *(**The condition is met** —
  [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md).
  The shelf entry stands, on price alone, and now carries ~~two open items: a probe site with no
  carrier, and the value space, which waits on
  [#111](https://github.com/winniel123/verge-asm/issues/111)~~ **one open item — a probe site with
  no carrier. The value space is settled by
  [ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md)** as a
  three-member closed union drawn per component, so **both parts this facet was blocked on are now
  writable** and the refusal rests on price alone. See the amendment below.)*
- **The cost is zero and there is no re-baseline.** No signature has ever been held, so nothing is
  withdrawn, no timeline closes, no `Break` fires and no message is owed. This is ADR-0027's cheap
  direction for the second time: the ruling names something that could not yet have produced a value.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) can state v1's position in one sentence:**
  the wildcard's effects are visible as `Shadowed` on everything beneath it, its content is measured
  every batch and held nowhere, and a repoint is invisible by design until the facet is taken up.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Hold it as a value on `dns-record`, under a second discriminator dimension** — the *losing option*, and the cheap-looking one the ticket flags as live | The `Span` key `(subject, facet, discriminator, vantage, source)` is already occupied: `example.com`'s `dns-record`/A holds the A RRset the authority served **for `example.com`**, and the synthesis would land on the same key with a different, equally correct value. Repairing it edits one of the six parts of a facet that already holds values estate-wide, to carry a value that is not a reading of that subject's records at all. And the meaning fails before the key does — `dns-record` is *what an authority served for a qtype* for **this** subject, while the synthesised answer was served for a random label that is a subject nowhere |
| **Hold it nowhere, ever — it stays an input to `wildcard-discrimination` and nothing more** | The runner-up, and it is right about v1 and wrong as a modelling claim. *Nothing* is only honest where the fact has no well-formed carrier, and this one has: a real subject already in the estate, a facet-shaped value, a discriminator, and a `Transition` that is one move on one timeline. Refusing it permanently would make the model unable to say what *your wildcard moved* is **about**, which is the hole ADR-0060 declined to leave one step over. What is true is the narrower thing this ADR rules: it does not ship in v1 |
| **Hold it on the `*.example.com` `Name`** | Barred by the ticket and by [ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md): a wildcard denotes a set of names rather than a name, so it is a `Subject` from no source. Not reached for, and not reopened |
| **Say *the zone apex* and be done** | [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) already called *apex* *"a heuristic standing in for a rule"*, and here it would be one twice over: §3.2 itself probes below the apex for each discovered sub-zone, and RFC 4592 permits a wildcard at any depth. *The name the control labels were generated under* is a reading of what the measurement did, is recorded by the `Batch`, and survives whatever [#108](https://github.com/winniel123/verge-asm/issues/108) decides |
| **Hold it on the `Batch` — it is measurement context, so record it as scope** | A `Batch` records **what its silence covers**, not what it measured, and it holds no timeline. Under that filing a repoint could never be a `Transition`, which is the entire fact this ticket is about. It also puts a measurement inside the scope record, which is the category error ADR-0005 and ADR-0009 keep the scope mechanism clean of |
| **Ship the facet in v1** | ~~Its **sixth part cannot be written**: ADR-0011 requires a batch-scope obligation naming what its silence covers, and nothing declares which `Name`s are control-probed.~~ **This ground is discharged by [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)** and the refusal now rests on price alone, plus the open carrier item in the amendment below. Shipping it anyway was ADR-0011's own named failure — *a session adds a facet, writes a canonicaliser, and silently skips the batch-scope obligation* — with an ADR behind it. ADR-0015 prices it at `revealed` plus one message with no `Break`, so waiting costs nothing and buys the missing part |
| **Settle the value space now, so the expensive part is decided while the context is loaded** | Inverted: the value space is expensive **because** widening one later costs a `Break` on every timeline of the facet, and it cannot be drawn honestly before the population it summarises is known. Deciding the costly part first, on the incomplete half, is the trade in the wrong direction |
| **Make the repoint a message on its own account** | [ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md): the facet layer is evidence and not a channel, and a `Transition` is a message only where it is the sole carrier of a fact the operator asked for. This one opens no `Endpoint` and admits nothing, so it inherits `dns-record`'s silence. A message here is a **`Signal`** under ADR-0004, on its own ticket — and a facet-level change channel was already refused for `dns-record` by [#64](https://github.com/winniel123/verge-asm/issues/64) |
| **Let the stored signature decide `Shadowed`, now that there is somewhere to store it** | Undoes ADR-0011's rule that any value needing more than one measurement is decided by the **measurement binary inside one batch**, and reintroduces the cross-observation dependency with its own currency and staleness problem inside the comparison path. The facet is a record of what was measured, never an input to the decision that was made alongside it |
| **Admit `Address` subjects from the synthesised RRset, since we measured those addresses** | They are the answer to a query for a name that does not exist. Admitting them is the fictional inventory §3.2 exists to prevent, arriving through a facet instead of through a resolution — and it is ADR-0060's *a construct denoting a set of names admits none of them* read from the DNS side |
| **Fold the population question into this ticket rather than filing it** | It is a larger question with its own evidence — RFC 4592's depth rule against §3.2's two-level procedure — it bears on `Shadowed` in v1 whether or not this facet ever ships, and it is **blocking [#12](https://github.com/winniel123/verge-asm/issues/12)** where this ticket expressly is not. Folding a blocking question into a non-blocking one hides it. *(Confirmed by outcome: [ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md) added an aperture input, rewrote §3.2's procedure, struck five stale counts and opened [#111](https://github.com/winniel123/verge-asm/issues/111) — none of which fits inside a facet-carrier ruling.)* |

## Amendment — [#108](https://github.com/winniel123/verge-asm/issues/108): the blocker is discharged, and the carrier is not guaranteed to exist

[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)
declares the control-probe population, so this ADR's stated reopening condition is **met**: the
`wildcard-synthesis` facet's **sixth part is writable**. It stays out of scope, now on price alone —
`revealed` plus one message with no `Break`, and ADR-0015 gives that no deadline.

**One open item arrives with the discharge, and it is new rather than inherited.** The population is
*the parents of the `Name`s in scope*, and a parent need not be a `Name` we hold: **[measured]**
`ns.iana.org` is NXDOMAIN and in no `Citation` while `www.ns.iana.org` is a held `Name` whose
closest encloser is `iana.org`, so it is a probe site with **no carrier**. This ADR's subject rule —
*the `Name` the control labels were generated under* — survives unchanged and is still the right
rule; what it loses is the guarantee, asserted in the Rationale above and now withdrawn, that the
subject always already exists. Inventing one is barred by ADR-0060 and ADR-0027 alike: nothing cites
it, and the prober is not an admitting source. Whoever takes the facet up owns that question, and it
is recorded here rather than ticketed because the facet has no deadline and buying the answer early
buys nothing.

**One thinness entry gains a measurement.** *"Whether the signature churns on its own is not
measured"* is answered in the label-to-label direction and not the batch-to-batch one it named:
**[measured]** 2026-08-14, five control labels per zone seconds apart from one vantage,
`herokuapp.com` returned **three** distinct address sets and `vercel.com` **five**, while
`github.io` and `localtest.me` held still. That is a defect in the **match predicate** rather than in
this facet, and it is [#111](https://github.com/winniel123/verge-asm/issues/111), **blocking
[#12](https://github.com/winniel123/verge-asm/issues/12)** — but it also means the value space this
ADR left open (*a closed union of no-synthesis and synthesised RRset*) cannot be drawn until #111
lands, because on the churning half *the* RRset is not a well-formed object. The deferral was right
for a second reason it did not know about.

> **#111 landed, and the churn is worse than this entry recorded.**
> [ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md).
> **[measured]** 2026-08-14 the answer is not a function of the label either: **six repeats of one
> label** against `vercel.com` returned five distinct address pairs, and **four repeats of one
> label direct to `ns01.herokudns.net`** returned four different ingress nodes with four disjoint
> address sets — so the rotation is the **authority's own** rather than an anycast resolver's. The
> batch-to-batch direction this entry named is therefore answered *a fortiori* and remains
> unmeasured only as a separate number. The value space is settled at **three** members per
> component, and the guess of two was short for exactly this reason.
