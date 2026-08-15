# ADR-0026: The facet layer is evidence, not a channel — a `Transition` is a message only where it is the sole carrier

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#64 Which facet `Transition`s besides the internet `Reach` leg are messages?](https://github.com/winniel123/verge-asm/issues/64)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

The drift class — *the world moved* — has been named since
[ADR-0007](./0007-drift-is-a-timeline-of-spans.md) and its membership has never been enumerated.
Exactly two things were known to be in it: the internet `Reach` leg going `not-reached` → `reached`
([ADR-0029](./0029-an-alert-fires-on-a-leg.md)) and a membership message on a `Name` or an
`Address` ([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)). Nothing
had ruled on any other facet `Transition`.

[#63](https://github.com/winniel123/verge-asm/issues/63) needed *does a `resolution` move notify?*
twice and could not answer it inside its own question — once for a `Name` re-pointing at an
`Address` already in the estate, which mints `Endpoint`s and fires no membership message, and once
for [ADR-0029](./0029-an-alert-fires-on-a-leg.md) §4's narrowed third carrier, where the sole
record of a service moving address is a `resolution` `Transition` nobody had ruled alertable.

### The question was posed per facet and the answer is not

The ticket asked which of the six facets — `resolution`, `dns-record`, `reachability`,
`certificate`, `http-identity`, `tls-acceptance` — are messages. The facet is the wrong unit, and
the proof is already in the accepted law rather than in anything this ruling adds.

ADR-0029 did not rule that the `reachability` facet notifies. It ruled that **one projection**
(`Reach`, not the three-value facet space), on **one key** (the internet `Vantage class`, not the
internal one), in **one direction** (`not-reached` → `reached`, not its reverse) is a message. That
is three narrowings below the facet, in one accepted ADR, and it leaves a facet transition —
`refused` ↔ `no-response` — that moves the recorded value and moves no message at all, because it
does not move the projection the predicate reads.

So *which facets notify* is the wrong question in the same way
[#61](https://github.com/winniel123/verge-asm/issues/61) found *the tier* was the wrong question:
the answer does not live at that granularity, and asking there forces a coarser ruling than the
model already supports.

## Decision

**No `Transition` is a message for the layer it sits in. A `Transition` is a message exactly where
it is the sole carrier of a fact the operator asked for; everything else is recorded, queryable,
and silent. Concretely: the facet layer is evidence, not a channel, and the drift class's real
membership is at the `Reach`, `Signal` and membership layers above it.**

### 1. The facet enumeration, all six, including the ones that are not

| Facet | Transitions that are messages | Everything else, and why |
| --- | --- | --- |
| `resolution` | → `Shadowed` (coverage member 1, existing law) · → `Lame` (coverage member 6, existing law, carried by `lame-delegation`) · `Shadowed`/`Lame`/`Gap` → a value, which is coverage member 7 fired **at the cause** with the census of what it restored sight to · **and one new message, §2** | `Resolved` → `NameError` is `withdrawn` and silent ([ADR-0006](./0006-subjects-leave-by-measurement.md)); `NameError` → `Resolved` is `returned` and belongs to membership (ADR-0031); `Resolved` → `NoData`, and any move that only removes addresses, is the shrinking direction ADR-0029 §4 and ADR-0006 both silence |
| `dns-record` | ~~**none, in any direction**~~ **none, except where the move opens a rule at `fired`** — amended by the [#65](https://github.com/winniel123/verge-asm/issues/65) amendment below | No v1 rule reads `dns-record` directly — [#48](https://github.com/winniel123/verge-asm/issues/48)'s two signals read the *composed* `resolution` — and a message per RRset move is ADR-0007's named burst shape verbatim: *a `dns-record` rule that fires per qtype per name*. Growth reached through a record — an MX or CNAME target entering — is a `Name` `appeared` and membership carries it |
| `reachability` | the internet `Reach` leg `not-reached` → `reached` (ADR-0029 §2), **payload widened by §3** | `refused` ↔ `no-response` never moves `Reach`, so it is recorded and silent — the model's own proof that the cut is below the facet; internet `reached` → `not-reached` silent (ADR-0029 §4); the internal leg silent in both directions (ADR-0029 §3) |
| `certificate` | ~~**none**~~ **none, except where the move opens a rule at `fired`** — amended by the [#65](https://github.com/winniel123/verge-asm/issues/65) amendment below, this reason never addressing `NoTLS` → `Presented` | `Presented(c1)` → `Presented(c2)` is renewal, the modal event on an ACME estate, and what matters about it is read by the clock rules; `Presented` → `NoTLS`/`TLSRefused` is the shrinking direction and a domain exit, which §4 rules is not a transition |
| `http-identity` | ~~**none**~~ **none, except where the move opens a rule at `fired`** — amended by the [#65](https://github.com/winniel123/verge-asm/issues/65) amendment below | `Responded(…)` → `Responded(…)` moves on every deploy — status, `Location`, `Server`, title — and is the burst shape one facet across; `Responded` ↔ `NoHTTPResponse` is a domain entry or exit, §4 |
| `tls-acceptance` | **none** | The accepted version and cipher sets moving is exactly what `tls-1.0-accepted` is named for, and §5 makes that rule's own edge the message |

### 2. The one new facet message: a `resolution` transition that admits ground nothing else covers

**A `resolution` `Transition` is a message where it opens an `Endpoint` that no membership message
in the same fold covers, and its census is exactly those `Endpoint`s and what opened beneath
them.** Drift class, *the world moved*. One message per transition, no count and no threshold in
the predicate — the test is inhabitance of a set the fold already computed, read from the two
adjacent spans alone and never reaching back across a `Break`.

This is [#63](https://github.com/winniel123/verge-asm/issues/63)'s own argument one layer across,
and it is textual rather than a base rate. ADR-0031 refused *`Name` only* because it is "silent on
a new name landing on an address already in the estate, which mints `Endpoint`s and therefore new
`certificate` and `http-identity` timelines" — and then closed only the half where the **name** is
new. Where the name is old and re-points, the `Endpoint`s open beneath two subjects both already in
the estate, so §1's root walk terminates above them and no membership message fires. Everything
that opens there reaches nobody, because no alerting predicate in this product is opening-shaped.

Three consequences fall out of the wording rather than being stipulated. Where every new address is
new to the estate, the `Address` `appeared` message already covers the whole residue, so the
residue is empty and **no second message fires** — no doubling. Where a name only drops addresses,
nothing opens and it is silent, which is the shrinking direction again. And `NoData` → `Resolved` is
the same predicate with an empty previous set, so it needs no case of its own.

### 3. The flagship message carries a census

ADR-0029 §2's flagship fires on a `Service` already in the estate whose internet leg opens. Beneath
it, `certificate`, `http-identity` and `tls-acceptance` timelines **open** —
[ADR-0011](./0011-a-facet-is-six-parts.md) gives a `certificate` timeline to every *open* `Service`
— and so do the rules over them. Under §4 those openings emit no `Transition` and reach nobody.

So the flagship message carries the census of what opened beneath the newly-reached `Service`, in
ADR-0029 §7's and ADR-0031 §3's shape: computed once at the cause, a description and never a
`Transition`, no difference set, nothing alerted individually. ADR-0031 wrote that a third producer
of that shape "would be a signal that the shape is right"; this is the third and §2 is the fourth.

### 4. Openings, and domain entry and exit, are not transitions and are not messages

This is [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)'s plain text — *a subject
outside the domain is not rendered as a member, a row, a state or a **transition*** — and
ADR-0014's, and it is confirmed rather than extended. ~~A rule that **opens at `fired`** is carried
by the census of a message above it where one exists (§2, §3, ADR-0031 §3) and by nothing where
none does.~~ ADR-0024's stated **reason** is withdrawn; see §7.

> **NARROWED by the [#65](https://github.com/winniel123/verge-asm/issues/65) amendment below.** A rule
> that opens at `fired` is carried **by the census where one exists, by the move beneath it where
> there is one, and by nothing where the timeline merely opened.** This section's ruling that domain
> entry and exit are **not** transitions is untouched. Marked here rather than only at the amendment,
> per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as
> widened by [#106](https://github.com/winniel123/verge-asm/issues/106).

### 5. The signal layer, per rule — where the drift class actually lives

A `Signal`'s edges are messages. ~~Sixteen~~ **Seventeen** rules are named in ADR-0024's v1 domain
table, [#128](https://github.com/winniel123/verge-asm/issues/128) having added
`non-globally-reachable-address-resolved-from-internet`. **Every count in this section is marked at
its sentence and only the denominators move** — the membership of the four below is unchanged, #128's
clearing edge being an ordinary silent one
([ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)).

- **`not-fired` → `fired` is a message, drift class, for all ~~sixteen~~ seventeen.** On a subject already in the
  estate this edge is by construction *something got worse*, so it needs no base rate to place it
  on [#17](https://github.com/winniel123/verge-asm/issues/17)'s alerting side.
- **`fired` → `not-fired` is silent, except where a third party could have caused the clearing.**
  Exactly four of ~~sixteen~~ seventeen qualify, and they are exactly the four rules whose clearing condition is a
  name somebody else can claim: `lame-delegation`, `cname-target-name-error`,
  `zone-declared-name-returns-name-error` and `resolved-name-absent-from-zone`. Those four carry
  [#35](https://github.com/winniel123/verge-asm/issues/35)'s *this improved and you should still
  look* register and [#48](https://github.com/winniel123/verge-asm/issues/48)'s attribution
  obligation — the message names which timeline gave way. This makes #35's *a clear is not always
  good news* structural: it was always a claim that clears are **sometimes** good news, which is a
  per-rule claim, and this is the rule that recovers it.
- **`fired` → `not-evaluable` is coverage class**, member 5, unchanged
  ([ADR-0010](./0010-exposure-composes-two-reaches.md), [#32](https://github.com/winniel123/verge-asm/issues/32)).
- **`not-evaluable` → any value is coverage class**, member 7 — `not-evaluable` is a `Gap` on the
  signal's own timeline ([#44](https://github.com/winniel123/verge-asm/issues/44)) — fired **at the
  cause**, stating the value it closed to. Where the cause is one `resolution` transition lifting a
  wildcard, that is **one** message with a census, not one per signal.

### 6. Subsumption: one cause, one message

Where two layers would report one fact, the message fires at the transition that caused it and the
other rides its census. ADR-0007's *alert on the cause, record the consequence* and its refusal of a
second representation of one fact, together. The worked example is
`sensitive-port-reached-from-internet`, whose firing edge happens exactly when the internet leg
opens on a sensitive port: it **never fires a message of its own**, and the flagship names it in the
payload — which is what ADR-0031 §3's census example was already doing without a rule behind it.

### 7. What is withdrawn

**ADR-0024's reason for refusing *outside the domain* as a third rendering case is withdrawn; its
ruling stands.** The reason given was that a subject "left the domain by a `Transition` on the facet
timeline underneath, which is already stored and **already alerted**". Under §1 most facet
transitions are stored and not alerted, so the reason is false. The ruling survives on ADR-0024's
own fourth binding: a rule's census is current state and **may never be rendered as a delta**, so
the operator is never shown a domain difference and there is nothing for a message to explain.
[#53](https://github.com/winniel123/verge-asm/issues/53)'s *the denominator has exactly two ways to
move and neither is silent* is withdrawn in the same clause and for the same reason.

**Nothing is minted.** No fifth cause, no fourth class, no new coverage-class member — the class
stays at nine. §2's message is *the world moved*, drift class, which is the class it was always in.
The only additions are payload: two more producers of a census shape that already had two.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in five entries** — `Transition` gains the
  sole-carrier rule and the statement that the facet layer is not a channel; `Facet` states that a
  facet transition is evidence and never a message on its own account; `Signal` gains §5's four
  edges; `Predicate domain` loses the withdrawn clause; `Reach` records that the flagship carries a
  census.
- **ADR-0029's §4 third carrier is repaired.** The direction cut on the internet leg rested on three
  carriers, one of which ADR-0031 made conditional. §2 rules that carrier alertable, so all three
  stand and the cut is no longer resting on an unruled residue.
- **ADR-0031's *`Name` only* / *`Address` only* hole is closed.** Its rejected-alternatives table
  names both halves of the re-point case; membership closed one and §2 closes the other.
- **ADR-0029's stated cost is wider than it was stated.** *An internal port opening tells nobody* is
  true of more than the leg move: `certificate`, `http-identity` and `tls-acceptance` open beneath
  it, the rules over them open, and §4 makes every one of those openings silent. The reason ADR-0029
  gave still holds — the install has not measured the question the product asks — but the price is
  larger than that ADR priced it at, and it is recorded here rather than left implicit.
- **The v1 rule set is sixteen, not ten.** ADR-0024's own domain table enumerates sixteen and says
  so (*"writing all sixteen out is what tested the rule"*), while
  [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md), ADR-0029 and ADR-0031 each say
  *ten*. This is [#47](https://github.com/winniel123/verge-asm/issues/47)'s hazard again — a count
  written once and copied three times after the thing it counted grew — and
  [#12](https://github.com/winniel123/verge-asm/issues/12) must use ~~sixteen~~ **seventeen**.
  > **The set grew again at [#128](https://github.com/winniel123/verge-asm/issues/128)**
  > ([ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)), which is
  > this bullet's own hazard arriving on schedule. The ruling — *a count written once and copied is the
  > defect* — is why the live figure belongs on the map's composed-state line and every ADR figure is a
  > dated record. Marked at the sentence per
  > [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened
  > by [#106](https://github.com/winniel123/verge-asm/issues/106).
- **`dns-record` is a recording facet in v1 and has no channel at all.** An MX, TXT or NS change
  reaches nobody. This is a scope statement rather than a hole: the route to alerting on one is a
  rule under ADR-0004's cadence test, which [#35](https://github.com/winniel123/verge-asm/issues/35)
  established #16's list does not bar, priced at nothing since adding a rule opens a timeline
  ([ADR-0015](./0015-the-value-space-is-the-commitment.md)).
- **Two payload obligations land on the notification patch.** The flagship's census and §2's census
  are a third and fourth producer of one shape, so the patch's wording work now covers four
  messages that all render a census at a cause. And §2's message can arrive the same night as an
  `Address` `appeared` message on a partly-new re-point — a **fourth wording pair**, after
  #28/[#22](https://github.com/winniel123/verge-asm/issues/22)'s,
  [#55](https://github.com/winniel123/verge-asm/issues/55)/[#51](https://github.com/winniel123/verge-asm/issues/51)'s
  and ADR-0031's.
- **Decided on thin ground, in two places, and neither is dressed as a derivation.** The **volume**
  of §2's message on a CDN-fronted or DNS-load-balanced estate is unmeasured, exactly as ADR-0031
  left `Name` `appeared`'s volume unmeasured; a `Resolved` value is an unordered address set so
  rotation within a fixed pool does not move it, but an edge set that genuinely churns fires every
  cadence, and if that drowns the channel the remedy is coalescing in the notification patch and
  never a predicate change, because the transition is recorded either way. And §5's **twelve silent
  clears** rest on the claim that clearing them requires control of the endpoint, and that control
  of the endpoint is a worse fact carried elsewhere or not at all — an argument, not a measurement.
  Neither this ADR nor ADR-0029 nor #17 has measured the base rates any of this stands beside, and
  the correction in every case is cheap: it moves one predicate in the notification layer and
  touches no timeline.
- **The residue this ruling does not discharge is named and ticketed.** A rule opening at `fired`
  with no message above it — an existing `Name` gaining a dangling CNAME, an existing `Endpoint`
  starting to present an expired certificate, a `tls-acceptance` timeline opening at `fired` on its
  own weekly `Scan` — reaches nobody. It is refused here rather than left open, because a carrier
  needs the three-way case analysis on *why the timeline opened* that ADR-0031 rejected in terms,
  and the version that avoids it fires on every internal deploy. It is
  [#65](https://github.com/winniel123/verge-asm/issues/65), and it does not block
  [#12](https://github.com/winniel123/verge-asm/issues/12): v1's answer is silence with the cost
  stated.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Answer per facet — name the facets that notify** | The question as asked, and the model refuses it. ADR-0029 already cut `reachability` three times below the facet, and `refused` ↔ `no-response` is a facet transition that moves the value and no message. A per-facet answer would have to make that edge loud or make the internal leg silent by hand, and either way it is a list where a predicate was available — ADR-0009's move, refused at the notification layer |
| **Every facet `Transition` is a message** | `http-identity` moves on every deploy and `certificate` on every renewal. It is ADR-0007's named burst shape and #17's common-and-intentional side, and it would train the operator off the channel before the flagship ever fires |
| **No facet `Transition` is ever a message** | Tidy, and it drops the one case with no other carrier. A `Name` re-pointing inside the estate mints `Endpoint`s that open, and openings reach nobody — #63's own argument, which this option answers by ignoring |
| **§2 fires on every `Resolved` → `Resolved`, including pure removals** | Fires on the shrinking direction that ADR-0006 and ADR-0029 §4 both silence, and on a CDN dropping an edge, for no fact the operator asked for. The `Endpoint`-opening test is what the message is named for |
| **§2 tests whether the name has *ever* pointed at the address** | Reaches back across a `Break`, so the same event notifies or does not depending on whether a derivation upgraded that cadence — ADR-0008's licence to reach back spent inside the notification predicate |
| **Make a rule opening at `fired` a message** | Genuinely attractive, and it would catch the dangling CNAME created on an existing `Name`. It loses on the burst: an internal deploy opens `plaintext-http-no-https` at *fired* on most estates, so the general form fires on every deploy — the thing ADR-0029 §3 refused — and the narrow form needs the three-way case analysis on why the timeline opened that ADR-0031 rejected by name. Refused for v1, cost stated, ticketed |
| **Per-rule enumeration of which openings notify** | Sixteen judgements each resting on an unmeasured base rate, in a map that has flagged #17's unmeasured base rates three times. Refused as the shape the map keeps warning about, not on the merits of any one row |
| **Route §2's message to the coverage class** | The world moved: the operator's name now reaches ground it did not reach. Our looking did not change. Filing it as coverage would be ADR-0031's *rooting an appearance at the `Seed`* defect one facet across — a world event under an observer cause |
| **Give the clearing edge one uniform answer, loud or silent** | Loud fires on every remediation in the estate. Silent contradicts #35, which established in terms that `cname-target-name-error` clears when somebody else may have claimed the orphaned name. #35's own wording — *not always* — is a per-rule claim, and pretending otherwise is manufacturing a consensus neither reading supports |
| **Mint a fifth cause for *a name now reaches new ground*** | It is *the world moved*, which is the cause it has always been. The map's constraint is that a fifth cause needs a reason and not a slot, and the four causes carry this one without strain |

## Amendment — [#65](https://github.com/winniel123/verge-asm/issues/65): the growing direction was under-enumerated, and the refused carrier needed no case analysis

[ADR-0033](./0033-a-move-carries-the-rule-that-opens-at-fired.md) took the residue this ADR
ticketed. **The Decision stands and is applied rather than overturned** — a `Transition` is a
message exactly where it is the sole carrier of a fact the operator asked for, and ADR-0033 finds
one more place where it is. Three rows of §1 and one sentence of §4 move.

**§1's table is amended in three rows, each of which is *none* by a reason that does not cover the
growing direction.** `certificate`'s reason addresses renewal and `Presented` → `NoTLS` and never
addresses `NoTLS` → `Presented`. `http-identity`'s routes `Responded` ↔ `NoHTTPResponse` to §4 in
both directions at once. And `dns-record`'s — *growth reached through a record is a `Name`
`appeared` and membership carries it* — is sound for a CNAME target that exists and false for one
that does not, which is the only case that fires a rule. Each of the three now reads **none, except
where the move opens a rule at `fired`**.

**§4 is narrowed.** *A rule that opens at `fired` is carried by the census of a message above it
where one exists and by nothing where none does* becomes: by the census where one exists, by the
move beneath it where there is one, and by nothing where the timeline merely opened. §4's ruling
that domain entry and exit are **not transitions** is untouched and is what ADR-0033 rests on: the
message is the facet move's, not a `Signal`-layer edge.

**This ADR's stated reason for refusing the carrier does not survive, and its instinct does.** The
refusal rested on the narrow form needing *the three-way case analysis on why the timeline opened
that ADR-0031 rejected by name*. It needs no analysis: a `Transition` either exists beneath the
opening or does not, and because none crosses a `Gap` or a `Break` and an opening emits none, that
one test excludes membership, aperture, a closing `Gap` and a slower tier at once. The general
form's burst objection was right and survives — ADR-0033 fires on neither shape of a deploy,
because a deploy either mints subjects whose timelines *open* or leaves `http-identity` at
`Responded` with no domain to enter.

**The census payload has five producers, not four** — the fifth being a move that opens a rule at
`fired`.
