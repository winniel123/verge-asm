# ADR-0044: A one-off measurement has no currency, so there is no one-off aperture — and the widest tier is the one that is asked for

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#80 Is the onboarding full-range sweep opt-in, or does it run unconditionally?](https://github.com/winniel123/verge-asm/issues/80)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`safe-active-probing.md`](../research/safe-active-probing.md) §2.4's tier table marks the
full-range tier **opt-in**, and the prose beneath it says the full-range sweep *"should also run
**once at target onboarding**, so the operator gets a complete baseline immediately"* — without
repeating the qualifier. The two sentences have disagreed since #4 closed.

It was harmless while a **warm tier** sat between the hot set and the full range.
[#78](https://github.com/winniel123/verge-asm/issues/78) retired that tier on a licence ground,
and the disagreement became the whole of the answer to *what does a default install look at?* —
`verge-core` plus the ~65,000-port tail **once**, or `verge-core` **forever**. (`verge-core` is a
union and [ADR-0009](./0009-verge-core-is-a-union.md) owns its membership; it stands at roughly 140
pairs today, and every ratio below is stated against that figure rather than depending on it.)

[#44](https://github.com/winniel123/verge-asm/issues/44) put a standing **aperture statement** on
`Coverage`, one line per aperture input, and port tiers are one of the five. A default install whose
aperture is `verge-core` and does not say so is the exact failure that statement exists to prevent,
so whichever way this goes, the aperture statement is where it has to be legible.

The two cases are both real and were stated fairly by the ticket. For **unconditional**: onboarding
is *"the one moment the operator is present and watching — the right time to spend the load"*, and a
baseline the daily scan can diff against is what stops the first month of drift being an artefact of
never having looked. For **opt-in**: #4 §1's safety posture, and an operator who mistyped a scope and
has not yet had a chance to notice.

## Decision

**The full-range tier stays opt-in, there is no unconditional onboarding sweep, and the onboarding
baseline is an act the operator performs rather than a default that runs.**

| Concern | Decision |
| --- | --- |
| Does a full-range measurement run unasked? | **No** |
| What the onboarding baseline *is* | [#51](https://github.com/winniel123/verge-asm/issues/51)'s first-run step 4, *Run the first batch* — a manual dispatch of the `Scan`s that exist |
| A one-off measurement with a port set and no cadence | **Not expressible.** No cadence, no currency bound; no `Scan`, no configured object |
| Opting in — per-`Seed` or per-estate | **Per `Seed` scope.** The cold `Scan` ships configured and **disabled**, with an empty scope list |
| The cold `Scan`'s existence | It exists disabled, so *"the operator has not enabled the full-range tier"* is a legible state rather than an absence |
| The aperture statement's port-tier line | States the tier, its cadence and its **off** state; counts **our** lists and rules; carries **no** count of what is unmeasured |
| A middle tier to cover the gap | **Refused**, and not on [#78](https://github.com/winniel123/verge-asm/issues/78)'s licence ground — see below |
| §6.3's *"~22 minutes"* | **Superseded.** It is the best case; the measured range is 22 min – 8 h per host |
| Enabling the tier later | An ordinary aperture widening — `revealed`, no `Break`, one coverage-class message ([ADR-0014](./0014-only-revealed-generalises.md)) |

## Rationale

### The unconditional limb was unbuildable as stated, and that is the finding

The ticket offered a binary. Its *unconditional* limb — a one-off full-range sweep sitting above an
opt-in recurring tier — is not an object this model can hold, and it fails on three independent
mechanisms that were all already in place.

**It has no configured object.** [ADR-0005](./0005-scan-execution-model.md) refuses ad-hoc runs
twice, in its Decision table and again in its rejected alternatives: *"Manual runs dispatch an
existing `Scan`, never an ad-hoc one-off: an ad-hoc run with a hand-picked port set produces a batch
whose scope no configured object accounts for, which is an aperture change with nothing to point
at."* An onboarding sweep is precisely that batch.

**It has no currency bound, and this is the decisive half.** [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)
sets currency at `k` cadences of the covering Declared `Scan` and publishes the full-range row as
**two months** (`k`=2 × monthly); [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md)
carries the same figure into its generations table as `monthly (full-range only) | 60 days |
≈ 10 generations`. A measurement that runs **once** has no cadence, so `k × cadence` has no value,
and every timeline it opens either renders a day-one observation as current forever or opens a `Gap`
at a time no rule can compute. ADR-0028 says in terms that narrowing that window *"would mean probing
the full range more often, which is the cost #4 declined"* — the arithmetic it publishes
**presupposes a recurring tier**. A one-off sweep is not a cheaper full-range tier; it is the same
tier with the denominator removed from a formula two ADRs already publish.

**It breaks the constancy [#44](https://github.com/winniel123/verge-asm/issues/44) rested on.**
#44 decision 10 discharged the three-densities obligation for the aperture statement on the ground
that *"the aperture statement is **constant**, so it can never escalate"*. A once-only full-range
sweep makes the port-tier line non-constant: it reads *1–65535* for a few hours on day one and
`verge-core` ever after. That falsifies decision 10's premise on the one screen whose whole job is
honesty about what is missing.

Making the tier a fifth `Scan` with a cadence of *once* fails the same way — `Scan` is *"the
operator's configured recurring intent"* and ADR-0005 gives each `Scan` one cadence. A cadence of
`null` is a hidden field wearing a configured object's clothes.

So the real choice was never *opt-in vs unconditional*. It was **opt-in** against **shipping the cold
`Scan` enabled at monthly cadence**, which is the only expressible form of *unasked*. That is the
option that lost, and it is named below.

### #4's own scheduling rules make the unconditional case's premise false

The whole force of *unconditional* is that onboarding is the one moment the operator is present and
watching. **#4 §6.4 forbids the mechanism that would make that true**: *"Never scan on config save.
Adding a target should queue a scan, not fire one."* A queued sweep runs at the next tick, under
±20 % jitter, inside operator-set quiet hours — that is, at 03:00 the following morning, unattended.

What was actually on offer, then, was not *spend the load while the operator watches*. It was **an
8-hour full-range sweep against production the operator has just declared, alone, overnight, on
their first night, which they did not ask for and will not see happen.** #4 §2.4 and #4 §6.4 have
been arguing with each other; §6.4 is right, and it takes the unconditional case's premise with it.

`Seed`'s own definition says the same thing one layer up: *"It declares a boundary, not a starting
point."* Reading a declaration as a starting gun is what the sentence was written to refuse.

### The blast radius, measured against #4 §6's own caps

The ticket asked for the arithmetic rather than an impression of it. Working from §6.3's shipped
defaults — ≤ 50 connection attempts/s per target host, ≤ 20 concurrent per host, 3 s connect timeout,
2 retries, global ceiling 200 pkt/s — against 65,535 TCP ports on one host:

| Host behaviour | Binding cap | On-wire rate | One pass | With 2 retries |
| --- | --- | --- | --- | --- |
| Answers closed ports with RST | rate, 50/s | 50 pkt/s | **21 min 51 s** | n/a — an RST is an answer |
| Drops (filtered) — the default cloud posture | **concurrency**, 20 ÷ 3 s | **6.67 pkt/s** | **2 h 44 min** | **8 h 11 min** |

**§6.3's "~22 minutes" is the best case stated as if it were the figure.** The dominant term on a
dropping host is timeout × concurrency, which §6.3 never multiplied out: 20 concurrent connections
each waiting 3 s yields 6.67 attempts/s, one seventh of the rate cap it thought was binding.

Three things fall out, and they do not all point the same way.

**Peak load is unchanged, and that is the strongest fact the unconditional case had.** Both tiers run
under identical caps, so a full-range sweep's peak rate and peak concurrency are exactly the daily
hot scan's. #4 §1's named hazard is the middlebox state table — *"Many NAT/firewall devices keep a
state entry for every port probe … occasional (pathetic) implementations crash instead"* — and that
is bounded by **concurrency**, which is 20 either way. **The full-range tier adds zero peak
state-table pressure over the daily scan the operator has already accepted.** The load is not
prohibitive and this ADR does not pretend it is.

**What it costs is duration, and duration scales past the caps the project already ships.**
`verge-core` costs 2.8 s (answering) to 63 s (dropping, with retries) per host per day. The full
range costs 22 min to 8 h. At the 200 pkt/s global ceiling the estate figure is 5.5 minutes per
address answering and **16.4 minutes per address dropping-with-retries** — so at §9's own shipped
`/22` per-target range cap (1,024 addresses) a single full-range pass is **3.9 days answering and
11.6 days dropping**. That cap exists to *"prevent a typo'd `/8` from becoming a multi-day scan"*.
It does not prevent this one; it permits an eleven-day one, monthly. A default that cannot complete
inside its own cadence on an estate size the project explicitly allows is not a default.

**The cost nobody had priced is subjects, not packets.** `Service` *"exists for every
`(port, transport)` in the recorded scope, **open or closed**"* — that is what gives `unreachable` a
subject to be a verdict about. So a full-range batch creates **65,535 `Service` subjects per
address** where the hot tier creates one per `verge-core` pair, each carrying a two-legged `Reach`
and an `Exposure`; and [ADR-0006](./0006-subjects-leave-by-measurement.md) means nothing leaves
because time passed, so they persist. The multiplier is `65,535 ÷ |verge-core|` — a shade under
**470×** at today's membership, and insensitive to the two rows
[#75](https://github.com/winniel123/verge-asm/issues/75) may move. Against the estate
[ADR-0001](./0001-stack-and-runtime.md) sized — ~70k reachability observations per hot run — that is
about **33 million** subjects and observations from one dispatch. The retention patch on the map is
sized against the hot tier. *(Closed by [#121](https://github.com/winniel123/verge-asm/issues/121) ·
[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md), which is also
sized against the hot tier and says why that is the right sizing: `Service` subjects and `Reach`
timelines are **spans**, one each and flat, so the cold tier's 470× multiplier lands on the corpus
that is never compacted; the observations it generates monthly age out at the currency bound like
any other.)*

### Narrow is not the failure; narrow and silent is

The ticket's discomfort is that *opt-in* means a default install sees the tail **never**. That reads
as the embarrassing answer, and #44 already ruled how the embarrassment is discharged.

**No rule loses anything.** [ADR-0009](./0009-verge-core-is-a-union.md) makes `verge-core` the union
`frequency-set ∪ sensitive-list`, so every pair `sensitive-port-reached-from-internet` reads is
inside the daily tier **by construction**. Walking [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)'s
domain table: one of the ~~sixteen~~ **seventeen** rules names a port and its domain is fully covered; four read
`Name`s and are untouched by any port tier; the remaining ~~eleven~~ **twelve** read a facet on a `Service` or an
`Endpoint`, so the tier bounds **which subjects exist** and never **which rules can speak**. There is
no `not-evaluable` hiding here and no unearned clean bill of health — ADR-0004's rule has nothing to
catch, because the population is honestly reported over every subject that exists.

> **DENOMINATOR struck at the clause by [#133](https://github.com/winniel123/verge-asm/issues/133)'s
> gate run (G4).** The v1 set is **seventeen** since
> [#128](https://github.com/winniel123/verge-asm/issues/128)
> ([ADR-0071](./0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)), whose
> `non-globally-reachable-address-resolved-from-internet` reads `resolution` — a facet on a subject —
> and names no port, so it joins the **third** bucket and the split is `1 / 4 / 12`. **The conclusion
> is untouched**: the tier still bounds which subjects exist and never which rules can speak, and the
> new rule is bounded by it no more than the other eleven were.
>
> **This is the site that *specifies* the split**, and
> [`safe-active-probing.md`](../research/safe-active-probing.md) §"cold tier" is the site that
> *restates* it. The 2026-08-15 reconciliation corrected the restating site and did not reach this
> one — [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
> defect in its purest form, found by the gate's first full run.

**What the tier buys is drift breadth, not signal correctness.** A listener on 31337 is real value
and it is exactly the Destination's *unexpected open port* — but it arrives as a `Service` census
entry and a `Reach` timeline, not as one of the ~~sixteen~~ **seventeen**. That is a coverage question, and #44
decision 7 already settled how a coverage question is stated: **counts of our own rules and lists,
never a count or proportion of the operator's estate.** *How many listeners are out there* has no
denominator, and inventing one is [#28](https://github.com/winniel123/verge-asm/issues/28)'s refused
estate-completeness score arriving through the port axis.

So the aperture statement's port-tier line states the tier, its cadence and its off state; it carries
~~`0 of 37 sensitive pairs unread`~~ ~~`0 of 41 sensitive pairs unread`~~ ~~`0 of 40 sensitive pairs unread`~~ ~~`0 of 38 sensitive pairs unread`~~ **`5 of 38 sensitive pairs unread`** (#109 removed `1433/tcp`; [#114](https://github.com/winniel123/verge-asm/issues/114) removed `9200/tcp` and `9300/tcp`; **the numerator is corrected from `0` by [#124](https://github.com/winniel123/verge-asm/issues/124) — five sensitive pairs are UDP and UDP is off on default settings, so ADR-0009's union puts them in `verge-core` without any of them being read. See the correction under the Consequences below**) and `0 of 16 rules
unevaluable`, ~~both of which are ours, closed and
true~~ **both of which are ours and closed, the second of which is true and the first of which is
now stated over the intersection of `verge-core` with the transports we probe**; and it says in prose that ports outside `verge-core` produce no `Service`, so nothing on them
is measured, evaluated **or counted**. Unlike #44's custody row it may carry a pointer to the `Scan`
configuration, because here an action genuinely exists and recommending it tells the operator nothing
false about themselves.

> **Denominator corrected in the five-pass merge reconciliation.** #80 measured this at 37 pairs; the
> sensitive list is **41**. **The numerator is `0` for every `|S|`** — ADR-0009's union puts every
> sensitive pair inside `verge-core` by construction — so the correction is a denominator and nothing
> in this ADR's reasoning reads it. `0 of 16 rules unevaluable` is unchanged.
> [`safe-active-probing.md`](../research/safe-active-probing.md) §2.4 and
> [`sensitive-ports.md`](../research/sensitive-ports.md) §29.9.

### Opting in is per `Seed` scope, because the widening is the largest the product can perform

The cold `Scan` ships **configured and disabled**, with a monthly cadence and an **empty scope list**.
That is ADR-0005's own construction — *"'The operator disabled the weekly deep scan' becomes a
legible state"* — and it is what keeps the aperture a configured object rather than an absence.

Enabling it is **per `Seed` scope, one scope at a time**, rather than an estate-wide switch. Two
reasons, and the first is measured. Enabling over the whole estate is a single click that opens
~~65,395~~ **65,404** pairs × every address in custody × two `Reach` legs of new timelines — for a ten-address
estate, ~1.3 million — carried by one coverage-class message whose payload under ADR-0014 is *"a
count of timelines opened and no comparison at all"*. The model absorbs it correctly and only
because [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) ruled a `Service`
entering is never a message; but a switch whose consequence is a seven-figure count is not a switch
to offer estate-wide. Second, [ADR-0022](./0022-confirmation-is-singular.md)'s asymmetry is the right
shape by **analogy** rather than by direct application — it governs `Proposal` confirmation, and this
is not a `Proposal` — and the analogy is stated rather than leaned on: an act that opens the widest
aperture the product has, over a named scope, belongs on the singular side.

> **The pair count is corrected here, at the site that computes it — `65,404`, not `65,395`.**
> The subtrahend is `verge-core`'s **probed** size, and `~140` was retired: it was never
> `verge-core`'s size, struck at ADR-0001, ADR-0005 and ADR-0009, and the union is **136 pairs, 131
> of them TCP** with the five UDP pairs outside v1's aperture
> ([#141](https://github.com/winniel123/verge-asm/issues/141) ·
> [ADR-0083](./0083-silence-decides-only-on-a-connection-oriented-transport.md) confirms UDP stays
> off, so `131` is the live multiplier). `65,535 − 131 = 65,404`.
> **The conclusion does not move** — a ten-address estate is still ~1.3 million and still
> seven-figure, which is why five passes walked past the arithmetic: the strike landed at every site
> that *defines* `~140` and at none that *multiplies* it.
> Found by [#120](https://github.com/winniel123/verge-asm/issues/120), which reported it rather than
> chasing it into two other ADRs mid-batch; verified and recorded by the merging session.

Turning it off again is a narrowing, which opens a `Gap` on every timeline it fed once currency
lapses. That asymmetry is a second reason the toggle is scoped rather than global.

### A middle tier is refused on the project's own union, not on the licence

#78 retired the warm tier because it was defined by reference to nmap's ranking. The obvious repair
for *opt-in* is a project-authored middle tier that avoids that trap. It is refused, and on a ground
that does not depend on the licence at all.

ADR-0009 already says what the project's own selection is: a port belongs to `verge-core` if the
frequency half's signal-mapping rule reaches it or the sensitive list forces it. A middle tier
authored on the project's own rule is therefore **either a subset of `verge-core`** — already
measured daily, buying nothing — **or a set of ports no shipped rule names**, which is the cold
tier's job at the cold tier's cadence. There is no middle to occupy. Building one anyway costs a
third `Scan`, a third aperture-statement line, a third cadence to explain, and a fifth curated input
for the governance patch that #78 had just shrunk to hand-authored-only. Priced and refused; and
because it is refused on the union rather than on the licence, it stays refused even if the licence
position ever changes.

## Consequences

- **[`safe-active-probing.md`](../research/safe-active-probing.md) §2.4's onboarding paragraph is
  replaced** by the ruling, and the tier table's cold row states the disabled-and-empty-scope shipped
  form. §1's summary row follows. §6.3's *"~22 minutes"* gains the measured range and the reason the
  concurrency cap binds. §9's `Port set (per tier)` default loses the retired `top-1000`.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Scan` entry is corrected** — with the warm tier retired,
  **two** `Scan`s are port tiers and there are **three** `Scan`s, not four. No term is added and none
  is amended in substance; the count was stranded by #78.
- **[ADR-0005](./0005-scan-execution-model.md) is amended** for the same count, and its ad-hoc
  prohibition gains the currency limb it did not have.
- **[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) and
  [ADR-0038](./0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md) are confirmed,
  not amended.** Their 60-day full-range currency figure survives exactly because the tier stayed
  recurring; ADR-0028's *"across all three reachability `Scan`s"* is stranded by #78 in the same way
  and reads **two**.
- **The honest statement of v1's default aperture is now available**, which is what
  [#12](https://github.com/winniel123/verge-asm/issues/12) was waiting on: *v1 measures `verge-core`
  daily on every address in custody, and nothing else, until the operator says otherwise.*
- **#78 §11.2's self-correction stands and is now priced.** Its retirement moved the tail ports to
  *never* on default settings rather than to 30-day latency. That does not reopen #78 — the licence
  ground is independent of the cadence — but the retirement's true price is recorded here rather than
  left as a half-corrected sentence.
- **[#63](https://github.com/winniel123/verge-asm/issues/63)'s open question narrows on the default
  path.** *Is the entry census emitted incrementally per completed `Batch`, or does it wait for a
  defined set of tiers?* — on a default install there is one tier that runs, so there is no set to
  wait for. The question survives only for an install that has enabled the cold tier, where it is the
  ordinary widening message.
- **An unresolved seam is surfaced rather than assumed.** §3.4's `-Pn` posture (*"against every target
  IP address specified"*) and §9's `/22` range cap read as though a declared address scope is
  enumerated, while `Seed` is *"a boundary, not a starting point"* and
  [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) has addresses existing only by
  citation. Every load figure in this ADR is stated per address for that reason. Ticketed
  separately — it is not this ADR's to settle.

  > **Settled 2026-08-14 by [#81](https://github.com/winniel123/verge-asm/issues/81) —
  > [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md): it enumerates.** Nothing in this
  > ADR is amended and its per-address figures stand. Two things resolve. The estate limb stops
  > being given twice: for an address-scope `Seed` the multiplier is the **declared size**, at most
  > **1,024** at §9's cap, so the *3.9-to-11.6-day* pass in the rationale and in the rejected
  > alternatives is the shipped worst case rather than a hypothesis — the losing option's arithmetic
  > becomes unconditional. And the hot tier at that same cap costs **12 to 36 minutes per day** at
  > the 200 pkt/s ceiling, which is the figure this ADR could not state and the reason the surviving
  > default fits inside its own cadence. ~~`0 of 37 sensitive pairs unread`~~ ~~`0 of 41 sensitive
  > pairs unread`~~ ~~`0 of 40 sensitive pairs unread`~~ **`0 of 38 sensitive pairs unread`** is untouched **by the enumeration**: it is a
  > count over our own list, evaluated per `Service`, and enumeration only widens the population it
  > is true over. The denominator moved because the **list** moved — 37 → 39 → 41 — and the numerator
  > is `0` for every value of it. **And it moved DOWN for the first time — 41 → 40 — when [#109](https://github.com/winniel123/verge-asm/issues/109) removed `1433/tcp` from the sensitive list; the numerator is still `0`, and **[measured]** the probed set did not move at all, `verge-core` staying at 136 pairs / 131 probed because the pair is in the frequency half. It moved down a second time and by two — 40 → 38 — when [#114](https://github.com/winniel123/verge-asm/issues/114) removed `9200/tcp` and `9300/tcp`, with the same disposal in every respect: numerator `0`, probed set unmoved, `verge-core` at 136 pairs / 131 probed, both pairs being in the frequency half.**
  >
  > > **The numerator is CORRECTED at this clause, 2026-08-15 by
  > > [#124](https://github.com/winniel123/verge-asm/issues/124). It reads `5 of 38 sensitive pairs
  > > unread`, not `0`, and it always did.**
  > >
  > > The reading that produced `0` is this ADR's own and is stated in the rationale: *ADR-0009's
  > > union puts every sensitive pair inside the hot set, so `sensitive-port-reached-from-internet`
  > > reads all of them daily.* The first clause is true and the second does not follow.
  > > [ADR-0009](./0009-verge-core-is-a-union.md) says in the same breath that `verge-core` is
  > > **136 pairs** of which **131 are probed on default settings, UDP being off** — and the five it
  > > leaves out (`69/udp`, `137/udp`, `138/udp`, `623/udp`, `11211/udp`) reach `verge-core` from the
  > > **sensitive list alone**. Every one of them is therefore a sensitive pair that a default install
  > > does not read. The figure sat inside the same sentence as its own refutation for four
  > > amendments, and every one of them re-checked the **denominator**.
  > >
  > > **Membership of `verge-core` is not measurement.** The aperture is `verge-core` ∩ the transports
  > > the shipped configuration probes, and the count is over the intersection.
  > >
  > > **`0 of 16 rules unevaluable` is untouched.** `sensitive-port-reached-from-internet` reads a leg
  > > on a `Service` and its domain is populated by the 131 TCP pairs, so the rule speaks. What the
  > > five cost is **subjects**, not evaluability: there is no `Service` for a pair outside the
  > > recorded scope. And this ADR's *"no unearned clean bill of health"* now holds for the reason it
  > > claimed rather than in spite of a rounded figure — the line names the transport and, on this
  > > ADR's own allowance, points at the tier config, because here an action genuinely exists. See
  > > [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §6.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Ship the cold `Scan` enabled at monthly cadence** — the only expressible form of *unasked*, and the reading under which #78's own *"7-day to 30-day latency"* pricing is simply true | **The losing option, and it lost on arithmetic rather than on posture.** It is affordable per host (22 min – 8 h, at zero added peak state-table pressure) and unaffordable per estate: at §9's shipped `/22` cap it is a 3.9-to-11.6-day pass, monthly, so the default configuration cannot complete inside its own cadence and ADR-0005's overlap rule turns it into a permanent skip the operator never asked for. It also creates ~32.8 million `Service` subjects on ADR-0001's sized estate, permanently, and spends 8 hours a month against production on a default nobody chose |
| A one-off unconditional sweep at onboarding, cold tier still opt-in — **the ticket's own *unconditional* limb** | Not expressible: no configured object (ADR-0005), no cadence and therefore no currency bound (ADR-0028/ADR-0038 publish a figure that presupposes one), and it makes #44's aperture statement non-constant, falsifying the premise decision 10 used to discharge three densities |
| A fifth `Scan` named *onboarding baseline*, cadence *once* | A `Scan` is recurring intent with one cadence; `null` is a hidden field wearing a configured object's clothes, and it would need a permanent aperture-statement row stating a historical fact rather than a standing state |
| Fire the sweep at `Seed` declaration, so the operator really is watching | §6.4's *"Never scan on config save"* by name, and `Seed`'s *"a boundary, not a starting point"* one layer up |
| A project-authored middle tier to cover the gap left by #78 | Refused on ADR-0009's union rather than on the licence: any set authored on the project's own signal-mapping rule is already inside `verge-core` or is the cold tier's population at the cold tier's cadence. There is no middle to occupy |
| Put a count of unmeasured ports on the aperture statement — *65,395 of 65,535 unread* | #44 decision 7. The number is knowable, which is what makes it tempting, and it is an estate-completeness score in port clothing: it invites the operator to read a deliberate aperture as a 99.8 % failure |
| Enable the cold tier estate-wide in one act rather than per scope | One click opening ~1.3 million timelines on a ten-address estate, and a symmetrical `Gap` storm on the way back out |
| Leave §2.4 ambiguous and let the implementation decide | The ambiguity **is** the answer to *what does v1 cover by default*, which #12 has to state |
