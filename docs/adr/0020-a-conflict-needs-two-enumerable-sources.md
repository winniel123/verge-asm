# A conflict needs two enumerable sources, and the zone is readable for names and not for records

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#48 Does zone-declared-name-does-not-resolve join the signal set, and what else does per-source keying expose?](https://github.com/winniel123/verge-asm/issues/48)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) keys timelines per source and **refuses to
arbitrate between them**, on a worked example. The operator's zone file lists
`old.example.com` and our own resolver returns a Name Error. That refusal is what stops a zone
file keeping a dead name alive, and it is where `authority` finally did work. Its by-product is
that the estate now *knows* when two sources covering one subject disagree. And
[#8](https://github.com/winniel123/verge-asm/issues/8) called telling the operator arguably the
most valuable thing the zone file buys beyond removal detection. ADR-0007 recorded that as an
opportunity rather than acting on it, because [#16](https://github.com/winniel123/verge-asm/issues/16)
appeared to have closed the signal set.

[#35](https://github.com/winniel123/verge-asm/issues/35) removed that obstacle —
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s cadence test is the gate, #16's list is
its output — so the question is now the ordinary one. And
[ADR-0015](./0015-the-value-space-is-the-commitment.md) removes its urgency: the thing with a
deadline is never *does this rule ship* but *does this facet record this field*.

Two things had never been enumerated. Which disagreement, exactly, out of the five `resolution`
outcomes [ADR-0011](./0011-a-facet-is-six-parts.md) fixed. And what *else* per-source keying
exposes, which had been left as three unchecked candidates.

## Decision

| Concern | Decision |
| --- | --- |
| What can conflict at all | **Two `enumerable` sources** holding current values on one `(subject, facet, discriminator, vantage)` |
| How many such pairs exist in v1 | **Exactly one** — the operator's zone file against our own resolver, on `dns-record` |
| The flagship rule | **Ships** as **`zone-declared-name-returns-name-error`** |
| The reverse direction | **Ships** as **`resolved-name-absent-from-zone`** |
| Which `resolution` outcome fires | **`NameError` only.** `NoData` is a non-firing evaluation; `Lame` and `Shadowed` are `not-evaluable` |
| Composition across vantages | **Unanimity across available vantages** — [ADR-0006](./0006-subjects-leave-by-measurement.md)'s membership rule reused, not a second one |
| Reading the zone's **records** | **Refused for v1.** The zone-file decoder's corpus is unbounded |
| A general `source-disagreement` signal | **Refused** |
| Two vantages differing | **Not a conflict** — a composition, and ADR-0006 already met this case |
| A `Seed` differing from a source | **Not a conflict** — a `Seed` declares a boundary and observes nothing |
| Certificate transparency against our handshake | **Not a conflict** — `corroborative`, and it has no subject to key on |
| A signal firing on a **withdrawn** subject | **Legal** — a signal's lifecycle is its evidence's, not its subject's membership |
| Alerting | The flagship alerts on **both** edges; `resolved-name-absent-from-zone`'s clear is recorded, not alerted |
| New fields on any facet | **None**, so neither rule carries an ADR-0015 deadline |

## Rationale

### The conflict lands on `dns-record`, and the rule reads `resolution`

ADR-0007's worked example names two sources and one name and does not say which **facet** they
collide on. It matters, because a signal must name the timelines it reads.

The zone file cannot produce a `resolution` value. ADR-0011 divided the two DNS facets on **walk
versus reading**: `resolution` follows CNAMEs and, since #35, queries the delegated authorities
directly. `dns-record` records what an authority served for a qtype. A file is a reading. It also
cannot reach the two values that need a second measurement — `Lame` needs the delegation walked
and `Shadowed` needs the ~~parent zone's~~ **parent name's** poison signature — the control probe runs
under a name's **parent**, which is not a zone boundary
([ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)) — which
[ADR-0011](./0011-a-facet-is-six-parts.md) puts inside the measurement binary in one batch.

So the collision is on **`dns-record`**, which is where ADR-0011 already put this pair when it
named *"our resolver's wire answer against the operator's zone file"* as a decoder pair. ADR-0007's
example is right in substance and imprecise about the facet. This ADR fixes the imprecision and
disturbs nothing.

But `dns-record` is keyed **per qtype**, and a name declared with A, MX and TXT would fire three
times for one fact — the failure ADR-0015 named when it forbade one fact wearing three names. The
fact here is about the **name**, not about a record. So both rules read the measured side from
**`resolution`**, the facet keyed per `Name` with an empty discriminator, and read the declared
side only for *does any zone-file `dns-record` timeline for this `Name` hold a record*. One firing
per `Name`.

Reading two facets and two sources in one rule is precedented rather than new:
`cname-target-name-error` already reads this `Name`'s CNAME record and the **target** `Name`'s
`resolution`.

### Only two enumerable sources exist, so the enumeration is short

A conflict in ADR-0007's sense requires two sources that both hold a **current value** on one
`(subject, facet, discriminator, vantage)`. A `corroborative` source cannot supply one in the
sense that matters: its silence asserts nothing, so under per-source keying *"such a source's
timeline simply has no closing events"*, and a difference between it and a `measured` source is
that source's staleness rather than a fact about the estate. So **a conflict needs two
`enumerable` sources.**

Walk the v1 matrix on that test and it collapses to one pair:

| Facet | Enumerable sources over it | Conflict? |
| --- | --- | --- |
| `resolution` | our resolver only | No |
| `dns-record` | our resolver, the operator's zone file | **Yes** |
| `reachability` | our prober only | No |
| `certificate` | our prober; `crt.sh` is `corroborative` | No |
| `tls-acceptance` | our prober only | No |
| `http-identity` | our prober only | No |

This is the useful half of the ticket, and it is a *negative* result: **per-source keying exposes
exactly one conflict pair in v1**, and every other candidate anyone can name is a different thing
wearing a conflict's clothes. Recording the test is worth more than recording the list, because
the list grows and the test does not.

### Vantage is a composition, and a `Seed` is a boundary

Two of the three candidates the ticket named dissolve on inspection, in the way
[#39](https://github.com/winniel123/verge-asm/issues/39)'s question was *malformed rather than
hard*.

**Two vantages disagreeing on resolution is not a conflict.** `vantage` sits in the timeline key
beside `source`, but the two are there for opposite reasons: two sources report the same fact and
may differ, while two vantages report **different facts** — what DNS answers *at that network
position* — and both are true. Reading ADR-0007's *report the conflict* as covering vantage would
make `Exposure` illegal, since [ADR-0010](./0010-exposure-composes-two-reaches.md) is precisely a
disagreement between two vantage classes composed into a value. And ADR-0006 already met the split
horizon case and answered it the same way: **withdrawal needs every available vantage to agree**,
*"or every split-DNS name flaps on every run"*. There is nothing left to decide.

A rule for the undeclared-split-horizon case would additionally need a Declared *"this scope uses
split DNS"* flag whose only consumer is a rule that wants to be quiet — a new Declared field bought
to damp a signal, which is [ADR-0009](./0009-verge-core-is-a-union.md)'s *a port the operator can
hide is a signal the operator can silence* arriving through DNS.

**An address-scope `Seed` covering an address no resolution cites is not a conflict either.** A
`Seed` produces no observations, so it is not a source in the ADR-0012 sense one layer over. It
declares where the estate ends and asserts nothing about what is inside. `CONTEXT.md` already
makes the two grounds for an `Address`'s membership **disjunctive** — *cited by a current
resolution* **or** *covered by a `Seed`* — so there is no shared timeline and nothing to
arbitrate. The operator's real question underneath it — *I declared a /24 and you found nothing in
it* — is a statement about a **declared scope we completed**, which is
[#28](https://github.com/winniel123/verge-asm/issues/28)'s `Coverage` screen by definition, and
routing it to a signal would put a coverage figure inside the comparison path.

### Certificate transparency cannot conflict, and cannot even be keyed

The third named candidate — a certificate SAN naming a name the zone does not — fails twice, and
the second failure is a defect rather than a decision.

It fails the enumerability test: CT is append-only and `corroborative`, so a certificate for a
long-dead name sits in the log forever and a rule over it would fire on every historical
certificate in the estate. That is not ADR-0015's licensed *commonness*. It is the **conflation**
ADR-0015 distinguished from it, since *a record was added out of band* and *a certificate outlived
its name* are opposite facts under one predicate. The half of it worth having — a name that
**resolves** and is absent from the zone — is `resolved-name-absent-from-zone` below, and it reads
`resolution` rather than the certificate.

The second failure is worth a ticket. ADR-0011 lists `crt.sh` as a **decoder for `certificate`**,
but `certificate` is *Presented(chain)* — a fact about what an `Endpoint` or `Service` served on
the wire — and **a CT log entry names neither**. There is no subject to key its timeline on, so
either CT is not a `certificate` source at all and only introduces `Name`s, or the facet needs a
subject it does not have. Under ADR-0015 that is a value-space question and therefore the only
part of this ticket's territory with a v1 deadline, so it is opened as its own ticket rather than
guessed at here.

### The zone file is readable for its name set and not for its records

This is the constraint that decides everything the two shipping rules do not do.

A rule comparing the zone file's **RRsets** against what the authority serves is the obvious next
step and it is refused, for a reason that is ADR-0004's own test one level down. A zone export
expresses things the wire does not — provider `ALIAS`/`ANAME` pseudo-records, apex CNAME
flattening, `$GENERATE`, `$INCLUDE`, provider-side DNSSEC signing that never appears in the file,
SOA serials, apex NS sets the provider rewrites. Decoding those into the wire's value space means
a stripper per provider convention, forever, which is exactly the unbounded corpus ADR-0004 calls
the out-of-band tell and ADR-0011 refused for the body hash. It arrives through the **decoder**
rather than through the rule, but a signal's effective version composes the decoder's, so a rule
resting on it is a signature database one level down.

This is not speculative. [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md)
already measured the boundary and already refused to stand on it: requiring the zone file to catch
a flattened `ALIAS` at a zone apex was declined because Route 53's API distinguishes `ALIAS` from
`A` and **Cloudflare's is unmeasured**, and it fails *silently*.

So the line is drawn where the evidence supports it:

> **A rule may read which names a zone contains. It may not read what records it holds for them.**

Parsing a file for its set of owner names needs no interpretation of any record's meaning.
Comparing an RRset does. Both shipping rules stay on the readable side, and both would be legal on
the other side of a future ticket that actually measures the export formats.

### `NameError` only, and the ticket's own working name was the collapse it warned against

The map recorded this candidate as *zone-declared-name-does-not-resolve*, and *does not resolve* is
true of four of `resolution`'s five values. Firing on all of them collapses distinctions
[#36](https://github.com/winniel123/verge-asm/issues/36) exists to keep apart, so each is taken
separately:

- **`NameError`** — the authority we walked to says the name does not exist while the operator's
  own file says it does. This is the fact, and it is the only one that **fires**.
- **`NoData`** — the name exists at the authority and holds no record of the qtype asked. The
  predicate is false and the evidence is present, so this is an ordinary **non-firing
  evaluation**, not `not-evaluable`.
- **`Lame`** — the delegated authorities were reached and none serves the zone, so *no* name in
  that zone can return a Name Error. Firing here would restate `lame-delegation` once per name
  beneath it, which is ADR-0007's *alert at the cause, never per consequence* violated at estate
  scale. **`not-evaluable`.**
- **`Shadowed`** — a wildcard answered, so the name has no answer of its own to read. **#35's
  precedent verbatim**, and it is the case where a false clear is worst: a shadowed name is where
  the operator has already lost visibility.

Note what the last two are not. They are `not-evaluable` because the evidence for *this* predicate
is absent, which ADR-0004 settled two tickets before #35 — not because a special case was carved
for them.

### Composition across vantages is ADR-0006's, reused

`resolution` is keyed per vantage, so both rules need a rule for disagreement, and ADR-0006 wrote
one: **withdrawal needs every available vantage to agree**, composing `Availability` exactly as
`Exposure` does. Both rules read that composed value, and where the available vantages do not
agree the composed value is `not-evaluable`, exactly as membership is *not-comparable* there.

Reusing it rather than writing a second composition is what makes split DNS a **non-event by
construction** on both rules, which is the largest false-positive class either of them has. An
asymmetric alternative was considered — unanimity to assert an absence, one vantage to assert a
presence, mirroring `completeness`'s own asymmetry — and refused: it is defensible in the
abstract and it fires `resolved-name-absent-from-zone` on every internal-only name in a
split-horizon estate, which is the failure ADR-0006 wrote its rule to prevent.

The declared side needs no composition. A `Batch` is *from one vantage* and the zone-file batch is
read by the worker, so that timeline has one vantage and no disagreement is possible on it.

> **NAMED by [#138](https://github.com/winniel123/verge-asm/issues/138) ·
> [ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md).**
> Both rules take the **cross-class** `Vantage composition`, as does membership; the *"every
> available vantage"* phrasing above is repaired at ADR-0006's own bullet and means **every
> `Vantage class` holds a current value and they agree**. Three consequences for this section.
>
> **The asymmetric alternative below is no longer *rejected* — it is inexpressible.** *"Unanimity to
> assert an absence, one vantage to assert a presence"* tries to put a quantifier on a cross-class
> composition, and the cross-class kind has no quantifier place. What this section correctly saw as a
> split-horizon hazard is now structural: across classes a difference is a fact about our aperture,
> not about the estate.
>
> **`lame-delegation` and `cname-target-name-error` are placed here too** — this ADR's neighbours from
> [#35](https://github.com/winniel123/verge-asm/issues/35) stated no composition at all. Both are
> cross-class: a lame delegation and a CNAME target's Name Error are facts about the delegation rather
> than about where the question was asked.
>
> **A cost this section did not state.** A cross-class composition over **one** class is that class's
> value agreed with nobody, so *"split DNS is a non-event by construction"* holds only on an install
> running two classes. On [#14](https://github.com/winniel123/verge-asm/issues/14)'s modal internal
> install — the one most likely to hold a split zone and to have supplied a file —
> `resolved-name-absent-from-zone` fires on every internal-only name in the zone. Disclosed rather
> than fixed: making the rule class-scoped to `internet` would make it `not-evaluable` there and leave
> the zone file's `completeness` with no consumer on the only install that has one, which is the
> argument this ADR shipped the rule on.
>
> **RULED by [#169](https://github.com/winniel123/verge-asm/issues/169): the noise floor is accepted,
> and the rule does not change.** Class-scoping to `internet` was the only fix on offer and it was
> re-priced, not re-argued: it does not lower the false-positive rate on the modal install, it removes
> the rule from it, which converts *fires too often on the install that matters most* into *is silent
> on the install that matters most* — a worse failure against #48's own reason for shipping. Two
> things this section under-stated are worth naming precisely. First, a single-class disagreement is
> not a new failure mode; it is the **same** disagreement ADR-0007 already licensed this rule to
> report without arbitration, now with a second contributing cause — split horizon joins staleness
> under one honest signal, rather than a corruption of it. Second, `Vantage class` is a fact about
> **our deployment**, not about the estate, so the fix for an under-vantaged install is an install
> decision — [#14](https://github.com/winniel123/verge-asm/issues/14)'s own open question about a
> second prober — and not a per-rule patch on `resolved-name-absent-from-zone`. A Declared "no split
> horizon here" flag to license firing on one class was considered and rejected on the same ground
> this ADR already used to reject the opposite flag (*"this scope uses split DNS"*, above): a new
> Declared field bought to move one rule's needle is the ADR-0009 pattern this project keeps
> refusing. No model change follows from this ruling.

### Two rules, and not a `source-disagreement`

A single rule over the whole class is tempting and fails twice.

It fails the naming test. A signal is **named for the fact it reads**, and the fact
`source-disagreement` reads is *two things differ* — which is a fact about our source
configuration, not about the estate. Its evidence and its remedy differ per pair, so one name
would mean *one of several unrelated things*, which is #35's stated reason for refusing to merge
`lame-delegation` with `cname-target-name-error` and applies here unchanged.

It also fails on evaluability. Evaluated over every `(subject, facet, discriminator, vantage)`, a
general rule is `not-evaluable` on almost every timeline in the estate, because almost every one
has a single source. A signal whose `not-evaluable` population is a fact about which sources the
operator enabled is a coverage statement wearing a signal's clothes.

Merging the two shipping rules into one fails for the same reason at smaller scale. They read
opposite values of `resolution`, they leave the operator holding opposite problems, and their
clearing edges are alertable in opposite directions.

### The reverse direction is the only consumer of the zone file's `enumerable`

`resolved-name-absent-from-zone` looked like the weaker of the two and is the one with the
structural argument behind it.

The operator-supplied zone file carries `authority: declared` and `completeness: enumerable`, and
the second of those currently buys **nothing**.
[#17](https://github.com/winniel123/verge-asm/issues/17) established that *our own resolver is
`enumerable` over a scope of one name*, so removal detection never needed the file — which means
two older documents overstate it. ADR-0005 says splitting the zone file per name *"would destroy
the only mechanism by which removal detection works at all"* and
[#7](https://github.com/winniel123/verge-asm/issues/7) made a `Seed`'s scope completeness decide
*"whether removal detection works inside that scope at all"*. Both were written before #17 and
both are stale in the same place. Everything the file still does — introducing names nobody would
guess — is `authority`, not `completeness`.

`resolved-name-absent-from-zone` is the **only rule in the product that reads an operator-declared
source's silence**. Without it the zone file is a name list and the map's *the zone file is the
product's spine* is decorative. That inverts the usual scope weighing: the cost is not two new
rules, it is finally spending something per-source keying and the zone file have both already paid
for.

The predicate is the mirror image: `resolution` composes to `Resolved` and no zone-file
`dns-record` timeline for that `Name` holds a record, within a scope the zone-file batch recorded
as completed. The scope clause carries most of the honesty. A name beneath a **delegated subzone**
is outside the file's zone, so the batch never touched its timeline and the rule is
`not-evaluable`. Likewise a name under a registrable domain for which no file was supplied. What is
left is tight: an A or AAAA-bearing name, inside a zone the operator exported, that the export does
not contain. `NoData` names — a lone `_acme-challenge` TXT — never reach the predicate, because
`resolution` did not compose to `Resolved`.

Its noise floor is real and is not a defect. An operator who exports once at onboarding will
accumulate firings as their live zone moves, and the honest reading is that **the file's staleness
is the thing being reported**. Per-source keying handles it correctly by construction: we report
the disagreement and never say which side is wrong, which is ADR-0007's whole position. And the
operator adding the name to their export is not [ADR-0009](./0009-verge-core-is-a-union.md)'s
silenceable signal — they are re-supplying ground truth, not removing evidence, and both timelines
still hold values afterwards.

### A signal may fire on a withdrawn subject

The flagship has a property nothing in the model has needed before, and it is worth stating
because it looks like a contradiction.

The condition that makes `zone-declared-name-returns-name-error` true — our resolver measuring a
Name Error from every available vantage — is exactly the condition under which ADR-0006 **withdraws
the `Name`**. So the rule fires on subjects that are no longer members of the estate.

That is legal and already written down: `CONTEXT.md` says a `Signal` *"has no lifecycle of its own;
its lifecycle is its evidence's"*, and the evidence here is two open, current spans. Membership is
a third thing and the glossary never made it a precondition. What was missing is the consequence:
**this is the only v1 signal whose firing population is the withdrawn one**, so a surface that
lists only living subjects cannot render it. The
[`Subjects` screen](https://github.com/winniel123/verge-asm/issues/1) patch already owns
*withdrawn* as its first population and
[#44](https://github.com/winniel123/verge-asm/issues/44) owns where a signal is seen. This is where
the two meet.

It is also what the rule is *for*. The question it answers — *which of the names that left are
still in your zone file?* — is one the model could not previously ask, and it is #8's claim about
what the zone file buys, made concrete.

### Both edges, and which side moved

#35 established that **a clear may not be good news** and both rules are checked in both
directions.

**`zone-declared-name-returns-name-error` alerts on both edges.** It clears when the authority
starts serving the name again — the operator restored it, **or somebody with write access to their
DNS created it**, which is the same sentence #35 wrote about an orphaned name being claimed. So the
clear reads *this changed, look at it* and never *resolved*.

There is a second clearing cause with no analogue in #35, and it is the reason for the obligation
below: the rule also clears when the **declared** side moves — the operator re-exports and the name
is gone from the file. Nothing in the world changed. So this is the first signal in the set whose
firing and clearing messages must **name which of the two timelines moved**, because *your zone
and your authority now agree* means two entirely different things depending on which one gave way.
That is a wording obligation on #44 and the notification patch, not a model change: both spans are
recorded and the difference is already in the data.

**`resolved-name-absent-from-zone`'s clear is recorded and not alerted.** It clears when the name
stops resolving, or when the operator adds it to their export — and the second is the operator
**answering the question the signal asked**, which is the commonest intentional resolution and the
case ADR-0006 used to refuse alerting on `withdrawn`. Alerting on it trains the operator to ignore
the channel.

The flagship does **not** inherit that refusal, and the distinction is #35's, unmodified: a name
withdrawing is a decommission, and a name withdrawing *while the operator's own file still declares
it* is the decommission done **untidily**, which groups by what the operator is left holding rather
than by how the change came about.

### Neither rule needs a field, so neither has a deadline

ADR-0015 requires this check and the answer is clean. Both rules read `resolution`'s existing
closed union and the existence of an existing `dns-record` span. No facet gains a field, no value
space widens, and no timeline breaks. **The land-grab argument is dead here** — under ADR-0011's
strictly-additive rule and ADR-0014's vacuous break on an opening, both rules could ship in v1.1
for the price of one re-baseline message.

They ship in v1 anyway, and the reason is the one above rather than a deadline: without
`resolved-name-absent-from-zone` the zone file's `completeness` has no consumer at all, and a spine
nothing reads is not a spine.

The one thing in this ticket's territory that *does* carry an ADR-0015 deadline is the CT keying
question, which is why it is a ticket and not a paragraph.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in two entries.** `Authority` records what can
  conflict at all — two `enumerable` sources, never a `corroborative` one, never two vantages, and
  never a `Seed`. `Signal` records that its lifecycle is its evidence's and **not its subject's
  membership**, so it may fire on a withdrawn subject.
- **[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s v1 set gains two members**, both
  carrying zero rows of reference data, no new measurement and no new field — a cleaner pass than
  the two #35 admitted, which needed a delegation walk.
- **[ADR-0007](./0007-drift-is-a-timeline-of-spans.md)'s final consequence is discharged.** *Zone-
  declared names that do not resolve become visible* is now two named rules, and the facet the
  conflict lands on is written down.
- **Two older statements about the zone file are stale and are not repaired here.**
  [ADR-0005](./0005-scan-execution-model.md)'s *"the only mechanism by which removal detection
  works at all"* and #7's *"whether removal detection works inside that scope at all"* were both
  withdrawn in substance by #17. They are recorded as stale rather than rewritten, because the
  partitioning rule ADR-0005 states is unaffected: the zone file is still one batch per zone, now
  because `resolved-name-absent-from-zone` reads its silence.
- **The zone-file source's batch-scope obligation is load-bearing and must record the *zone*, not
  the registrable domain.** A zone stops at a delegation, and a scope recorded one level too wide
  makes `resolved-name-absent-from-zone` fire on every name in every delegated subzone — ADR-0009's
  `{161}` defect for the fourth time, arriving through a DNS scope.
- **A fourth register joins #44's absence vocabulary, and it is the first the operator causes.**
  Beside *we never looked*, *we stopped looking* and #35's *you stopped answering* sits **you
  stopped telling us** — the declared timeline ageing into a `Gap`, which takes both rules to
  `not-evaluable`.
- **The first two-source signal, so a message must name which side moved.** Every prior signal
  reads one source and a transition needs no attribution. These two do.
- **The flagship's population is the withdrawn one.** #44 and the `Subjects` screen patch own the
  rendering. The model change is none.
- **One ticket opens** — which subject a certificate-transparency observation attaches to, given
  ADR-0011 lists `crt.sh` as a `certificate` decoder and a log entry names no `Endpoint` and no
  `Service`.
- **Reading the zone's records is deferred, not discarded.** It becomes legal the moment somebody
  measures what the major providers' exports actually emit — the measurement ADR-0013 declined to
  rest a safety property on.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| One `source-disagreement` signal | Its fact is *two things differ*, which is a fact about our source configuration; and it is `not-evaluable` on almost every timeline, so its evaluability is a coverage statement in a signal's clothes |
| `zone-declared-name-does-not-resolve`, the map's own working name | *Does not resolve* is true of four of `resolution`'s five values, three of which must not fire; the name is the collapse the ticket warned against |
| Merge the two shipping rules into one | Opposite values, opposite remedies, opposite clearing edges — #35's reason for refusing to merge its own pair |
| Compare the zone's RRsets against the wire, per qtype | The decoder needs a stripper per provider convention forever — ADR-0004's out-of-band tell arriving one level down, and ADR-0013 already measured that Cloudflare's export is unmeasured |
| Read the disagreement on `dns-record` on both sides | Fires once per qtype for one fact about a name — three names for one fact, which ADR-0015 forbids |
| A split-DNS-disagreement signal | Two vantages measure different facts, not one fact differently; ADR-0006 already met this case and composed it, and ADR-0010 would be illegal under the other reading |
| A Declared *this scope uses split horizon* flag | A new Declared field whose only consumer is a rule that wants to be quiet — ADR-0009's silenceable signal through DNS |
| A signal on an address-scope `Seed` covering an uncited address | A `Seed` observes nothing and the two membership grounds are disjunctive; the real question is *we completed your declared scope and found nothing*, which is #28's `Coverage` |
| A certificate-SAN-not-in-zone signal | CT is append-only, so it fires on every historical certificate for every dead name — conflated rather than merely common, and the half worth having reads `resolution` instead |
| A CT-versus-handshake certificate conflict rule in v1 | CT is `corroborative`, so it can hold no conflicting current value; and it has no `Endpoint` or `Service` to key on, which is a defect to fix before a rule is written |
| Asymmetric vantage composition — unanimity for absence, one vantage for presence | Defensible in the abstract; fires `resolved-name-absent-from-zone` on every internal-only name in a split-horizon estate, which is what ADR-0006's rule exists to prevent |
| `authority` as a tiebreak so the declared side wins | The arbitration ADR-0007 refused, arriving as a signal; it would let a zone file keep a dead name alive, which ADR-0006 forbids |
| Record the flagship without alerting, alongside `withdrawn` | Groups by how the change came about rather than by what the operator is left holding — the door ADR-0007 closed when it made `Break` structural |
| Alert on `resolved-name-absent-from-zone` clearing | The commonest clearing cause is the operator adopting the name into their export, which is them answering the question; alerting trains the operator to ignore the channel |
| A threshold — *declared and unresolved for N runs* | Persistence is the span's duration once the failure is a value; an `N` destroys the flap count and costs a `Break` per adjustment (#35, unchanged) |
