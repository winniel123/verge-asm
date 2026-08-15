# A control probe is generated under a `Name`'s parent, and that population is aperture rather than a parameter

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#108 Which `Name`s does `wildcard-discrimination` generate control labels under, and is that population a declared parameter or an aperture input?](https://github.com/winniel123/verge-asm/issues/108)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Discharges:** [ADR-0062](./0062-a-wildcards-synthesis-is-a-fact-about-the-name-it-was-probed-under.md)'s ticketed gap — its sixth part is now writable

## Context

`Shadowed` is the value a `resolution` observation takes **when the answer matches a wildcard's
measured poison signature** ([`CONTEXT.md`](../../CONTEXT.md)). Where no signature was measured
under a name, no name beneath it can be `Shadowed`, so every synthesised answer is recorded as
`Resolved` with a fictional address set — which admits fictional `Address`es, `Service`s and
`Endpoint`s. That is
[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §3.2's stated catastrophe
in full — *"any pipeline that treats 'resolves' as 'exists' reports an unbounded, entirely fictional
asset inventory"* — and §3.2 is the section that calls the machinery **mandatory, not optional**.

And the population was nowhere declared.
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s parameter table — which
[`project-authored-constants.md`](../research/project-authored-constants.md) §6 calls *"the
authoritative population"* — gives `wildcard-discrimination` exactly *control-label count and
construction* and *the match predicate*. [`measurement-offers.md`](../spec/measurement-offers.md)
never mentions the control probe.
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) makes EDNS options a
declared parameter of this leaf and is silent on its population. The only statement anywhere is
§3.2's prose — *query 3–5 long random labels under the apex*, then *"repeat one level down for each
discovered sub-zone"* — which is a procedure in a research note rather than a scope any `Batch` can
record, and which is under-inclusive by its own admission, RFC 4592 §2.1.1 putting a wildcard owner
name at any depth in a zone while §3.2 probes at two.

## Decision

| Concern | Decision |
| --- | --- |
| Which `Name`s get control labels | **The immediate parent of every `Name` in the batch's resolution scope**, deduplicated, and intersected with the `Seed` name scopes those names sit inside |
| The rule in one sentence | **A name is discriminated at its parent**, because a control label generated under P falls off the tree at exactly the encloser the names under P fall off at |
| Why that answers *"wildcards can exist at any label depth"* | It **dissolves** the depth question rather than covering it. You never enumerate depths; you enumerate parents, and the probe finds the synthesising wildcard wherever it sits — **including where the parent itself does not exist** |
| Must the probe site be a `Name` we hold? | **No**, and this is the load-bearing widening. A probe site is a **label sequence we construct**, not a subject we admit — nothing is cited, nothing enters the estate |
| Is the population a declared parameter of `wildcard-discrimination`? | **No.** It is a **function of the batch's own scope**, and a parameter is authored data |
| Is it an aperture input? | **Yes — the seventh.** The enumerated list moves from six to seven |
| What the `Batch` records | The **set of label sequences control labels were generated under**, by content, deduplicated — never the rule that produced it and never a count |
| Which qtypes the control probe runs | **The batch's declared qtype set**, all seven, and no second list. `Shadowed` is already committed on `dns-record` *for any qtype* |
| Is a sixth `Offer` minted? | **No.** The population is not an offer — nothing about it is a candidate on the wire — and `measurement-offers.md` stays at five |
| What a name holds where the control probe under its parent **did not complete** | A **`Gap`**, stating its cause. **An undiscriminated answer is never a value** |
| What it holds where the probe **completed and found no wildcard** | An ordinary value. The modal case is unaffected |
| What an aperture widening costs here | A **`Break`** on the timelines beneath the newly-probed parent — and it is **structurally vacuous** in the ordinary case, by the rule's own shape |
| `Shadowed`'s decision site | **Untouched.** Both measurements still happen inside one batch and the leaf still decides ([ADR-0011](./0011-a-facet-is-six-parts.md)) |
| `wildcard-discrimination` | **Still one leaf**, still five leaves total, and its declared parameters are **unchanged** |
| Cost of the ruling | **Zero.** Nothing has shipped, no timeline exists, no `Break` fires |

## Rationale

### The measurement: a control label under P falls off the tree where P's children do

RFC 4592's synthesis rule is about the **closest encloser**. `*.E` answers a query name QNAME when
E is QNAME's closest existing ancestor and no closer match exists. So the wildcard that can poison a
held `Name` N is `*.E` where E is N's closest existing ancestor — which is a fact about the *world*,
not about our estate, and which is exactly why *"probe at depth 1, then depth 2"* is a guess.

The escape is that a control label constructed under N's parent P has the **same** closest encloser
as N. N and `<random>.P` differ only in their leftmost label, and N's own existence is precisely
what is in question, so both fall off the tree at the same E and both are answered by the same
`*.E` — whatever depth E sits at, and **whether or not P itself exists**.

**[measured]** — RFC 4592 §2.2 read against live authorities, from one vantage over Google Public
DNS, 2026-08-14:

| Zone | `<r>.Z` | `<r>.<r>.Z` (parent absent) | Verdict |
| --- | --- | --- | --- |
| `localtest.me` | `127.0.0.1` | `127.0.0.1` | identical at three depths |
| `traefik.me` | `127.0.0.1` | `127.0.0.1` | identical at three depths |
| `vcap.me` | `103.224.182.214` | `103.224.182.214` | identical at three depths |
| `github.io` | four `185.199.10x.153` | same four | identical |
| `railway.app` | `34.107.141.139` | `34.107.141.139` | identical |
| `netlify.com` | `18.208.88.157`, `98.84.224.111` | same pair under `dev.netlify.com`, whose SOA owner is `netlify.com.` — **not a zone cut** | identical |
| `render.com` | **NXDOMAIN** | `10.4.3.1` under `staging.render.com`, and the same at two labels below it | apex-only sees nothing |

A 170-query sweep over ten common labels across seventeen domains returned **31 depth-2 hits**, and
every one of them was the apex wildcard reaching down through an intermediate label that does not
exist. The mechanism is not marginal; it is how wildcards behave.

`render.com` is the whole ticket in one row. Its apex control label is NXDOMAIN, so §3.2 step 1
reports *no wildcard* — while `*.staging.render.com` answers `10.4.3.1` for anything beneath it.
Every synthesised name under that sub-tree would be recorded `Resolved([10.4.3.1])`, and RFC 1918
space published in public DNS is the kind of thing this product exists to notice, not to invent.

### The estate the repo already sampled decides between the three options

§7 of `passive-discovery-sources.md` sampled `%.iana.org` through crt.sh. The same query answers
this ticket.

**[measured]** — crt.sh, `%.iana.org`, 2026-08-14 (two attempts; the first returned a spurious 404,
which is §7's own documented failure mode): **25 distinct non-wildcard names**, whose immediate
parents are **six** names inside the seed —

`iana.org` · `int.iana.org` · `itar.iana.org` · `ns.iana.org` · `rzm.iana.org` · `rzm-epp.iana.org`

— plus `org`, which the probing gate excludes. Resolving each parent's SOA owner:

| Parent | Zone it sits in | A zone cut? |
| --- | --- | --- |
| `iana.org` | `iana.org.` | **yes** |
| `itar.iana.org` | `iana.org.` | no |
| `rzm-epp.iana.org` | `iana.org.` | no |
| `ns.iana.org` | `iana.org.` — and the name is **NXDOMAIN** | no |
| `int.iana.org` | `vip.icann.org.` (a CNAME) | no |
| `rzm.iana.org` | `vip.icann.org.` (a CNAME) | no |

That table sorts all three candidate populations at once.

- **The apex alone** probes one of the six sites and covers the 14 names whose parent is `iana.org`.
  **11 of 25 names — 44% — are left undiscriminated**, every one of them under `int.`, `itar.`,
  `ns.`, `rzm.` or `rzm-epp.`
- **The apex plus each measured zone cut** — the ticket's own middle, and the *losing option* —
  probes **exactly the same one site**, because **one of the six parents is a zone cut**. On this
  estate option B *is* option A. It is not a middle at all.
- **Every `Name` in the estate** probes 25 sites to cover 25 names.
- **The parent set** probes **6** sites and covers **25 of 25**. A 4.2× reduction against the closed
  direction, with **complete** coverage rather than partial.

`ns.iana.org` is the row that also settles the *must the site be a `Name` we hold* question, and it
settles it by existing: it is NXDOMAIN, it is in no `Citation`, and `www.ns.iana.org` is a held
`Name` whose closest encloser is `iana.org`. Refuse to probe under names we do not hold and that
name goes undiscriminated for the sake of a purity nothing needed.

### Zone cuts are the wrong index, and they are wrong in both directions

The ticket's middle looked principled because SOA is already in the qtype set
([ADR-0030](./0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md)) *precisely
because it establishes whether a name is its own zone*. But that is what makes it seductive rather
than right: it indexes **delegation**, and a wildcard is not a delegation. RFC 4592 §2.1.1 defines
it as an ordinary owner name in a zone whose leftmost label is the octet pair `0x01 0x2a`. Nothing
about it requires a cut at its parent, and nothing about a cut implies one.

**Under-selects `[spec]` and `[measured]`.** Five of IANA's six parents are not cuts. And the
motivating persona makes it worse than the numbers: a small org has **one zone**, so *the apex plus
each measured zone cut* degenerates to *the apex* for the modal operator, and `*.dev.example.com`
inside a single flat zone — the commonest wildcard shape there is — is invisible to it.

**Over-selects `[measured]`.** Of fifteen zone cuts probed — `amazonaws.com`, `core.windows.net`,
`blob.core.windows.net`, `akamaiedge.net`, `dscx.akamaiedge.net`, `akadns.net`, `cloudfront.net`,
`fastly.net`, `ssl.fastly.net`, `map.fastly.net`, `edgesuite.net`, `edgekey.net`,
`elb.amazonaws.com`, `us-east-1.elb.amazonaws.com`, `compute-1.amazonaws.com`,
`execute-api.us-east-1.amazonaws.com`, `pages.dev`, `workers.dev` — **thirteen carry no wildcard at
all**. Paying a full control probe per cut buys nothing on most of them.

So the index is not merely imprecise; it tracks a different property of the zone. That is why it
loses, and it loses to a rule that indexes the thing the RFC indexes.

### Every `Name` in the estate loses on buying nothing, not on cost

The closed direction is the one to beat honestly, because the tempting objection to it — *it is
expensive* — is weaker than it looks. Run the arithmetic against
[`safe-active-probing.md`](../research/safe-active-probing.md) §6.3's **200 pkt/s global ceiling**,
with §3.2's 3–5 labels and the seven-qtype offer: 21–35 queries per site.

| Population | Sites, IANA estate | Queries | Wall clock at 200 pkt/s |
| --- | --- | --- | --- |
| Apex alone | 1 | 21–35 | ~0.2 s |
| Parent set | 6 | 126–210 | ~1 s |
| Every `Name` | 25 | 525–875 | ~4 s |

None of that threatens the ceiling, and pretending otherwise would be the kind of safety argument
[#21](https://github.com/winniel123/verge-asm/issues/21) refuses. These are queries to the
operator's own recursive resolver rather than packets at a target host, so §6.3's per-host caps do
not bind and only the global ceiling does.

The closed direction loses on a stronger ground: **it buys nothing.** A control label under N
discriminates exactly the names whose closest encloser is N — which is to say, names *beneath* N.
For a `Name` with no held descendant that population is empty, and leaves dominate every name tree
a discovery pipeline produces. On IANA's estate, 19 of the 25 sites option C pays for are leaves,
and the probe under each one answers a question about a sub-tree the estate does not hold.

There is a residual cost worth stating rather than leaning on. Control labels generate NXDOMAIN
traffic at the operator's own authority — 210 a night on IANA's estate, against §3.5's *"a million
queries"* objection to brute-forcing, so the shape is right and the magnitude is nowhere near it.
And they pollute the recursive resolver's negative cache with junk names, which is a real if minor
cost that scales with the site count and is another reason not to pay for leaves.

### It is aperture, and the decisive argument is that a parameter cannot hold it

The ticket frames the choice well and then mis-frames the class. *"The model's five offers are all
wire content, and this is a set of subjects"* — true of offers, and irrelevant, because **aperture
inputs were never coextensive with offers.** Three of the six settled before this ruling are
populations of subjects, not wire content, and ADR-0025's own table states their negatives:

> Port and transport tiers — *"A pair never probed would be an absent `Service`"*. The custody gate
> — *"An address never probed would be an absent everything"*. Enabled sources — *"A source never
> asked would be an absent per-source timeline"*.

Read the control-probe population the same way and it is worse than an absence: **a `Name` never
control-probed is a sub-tree that can never be `Shadowed`**, and the value it takes instead is
`Resolved` with a fictional address set that goes on to admit `Address`, `Service` and `Endpoint`.
[ADR-0009](./0009-verge-core-is-a-union.md)'s `{161}` defect is a record asserting an **absence**
over something it never touched; this is a record asserting a **presence** it never discriminated,
which is the same defect in the direction that costs more.

That places it. What settles it is that the alternative is unavailable.

**A declared parameter is authored data, and this population is measured.**
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) makes a leaf version move on *a
changed declared parameter*, and makes that mechanically checkable *"because the parameter set is
declared data"*. The control-probe population is a **function of the batch's own resolution scope**,
which grows every cadence as CT and the zone file admit names. Filed as a declared parameter, it
would bump `wildcard-discrimination` **every time the estate discovered a subdomain with a parent it
had not seen** — and that leaf feeds `resolution` and `dns-record`, so each bump `Break`s the name
half of the estate. Discovering a subdomain is the single most ordinary event this product has.
That is ADR-0021's *unsurvivable fallback* rebuilt inside one leaf, arriving through the parameter
table instead of through a release.

ADR-0025's *"a default is not a declaration"* is the same rule read from the other end: a parameter
whose value is *whatever the batch found* is not declared data either.

**And a `Batch` is the only object that can hold it.** *A `Batch` records the scope its silence
covers* — and `Shadowed`'s unavailability beneath an unprobed name **is** a silence.
[ADR-0014](./0014-only-revealed-generalises.md) makes a widening detectable by diffing named
dimensions across batches, and diffing the control-probed name set across batches is exactly how
*we started probing under `dev.example.com`* becomes visible. A rule recorded in place of the set
would be ADR-0025's rejected library-version dimension in different clothes: you cannot diff a rule
to tell a widening from a narrowing. Hence **by content**, as `Batch` already requires of every
dimension.

The counter-case deserves its own paragraph because ADR-0025's table appears to have decided this
already: **Control-label count | `Shadowed` is one value | Parameter.** If the count is a parameter,
why is the population not? Because they are two facts, and ADR-0025's own reasoning separates them.
The count *conditions* a value — `Shadowed` is one member of a closed union and looking harder moves
which member you get. The population decides **whether the value is available at all**, across a
whole sub-tree, which is the shape every subject-population aperture input has. The count stays a
declared parameter, untouched, in ADR-0021's table where it already sits. This is not one fact in
two homes, which is the trap ADR-0025's [#62](https://github.com/winniel123/verge-asm/issues/62)
amendment closed; it is two facts in the two homes each belongs in.

### The seventh input is free, and that is a theorem rather than a promise

ADR-0025 fixed the price: *an aperture widening `Break`s the timelines it touches and `revealed`s
the timelines it opens; a dimension in the key opens, a dimension inside the value breaks.* The
population sits in neither key — the timeline key is `(subject, facet, discriminator, vantage,
source)` — so probing under a name we did not before moves `Resolved` → `Shadowed` on running
timelines beneath it, and that is a **`Break`**.

Except that it cannot happen in the ordinary case, and the reason is the rule's own shape. The
population is *the parents of the names in scope*. A parent P enters the population only when a
name under P enters the estate — because any other name under P would have put P there already. So
at the moment P is first probed, **the only timelines beneath P are the ones opening that cadence**,
and a `Break` on timelines that do not yet exist is what
[ADR-0009](./0009-verge-core-is-a-union.md) calls **vacuous**: not waived, but a property of
timelines that exist, and there are none.

Two cases survive, correctly.

- A batch that could not complete the control probe under P leaves a `Gap`; the next batch closes
  it. `Gap` → value on a running timeline is **`revealed`**, which ADR-0014 already ruled and
  ADR-0025 already quoted. Not a `Break`.
- A **release** that changes the population rule — a later session widening it to every `Name`, say
  — is a genuine, non-vacuous widening and `Break`s what it touches. That is the right price for
  the right event, and it is the price this ADR is buying insurance against by settling the rule
  now.

Narrowing is symmetric and equally vacuous: a parent leaves the population only when the last name
under it leaves, so there is nothing beneath to stop covering.

### An undiscriminated answer is never a value

ADR-0025 ruled the sibling case in one sentence: *"A truncated answer is never a value.
`resolution-walk` retries over the fallback transport, or it records no value — a `Gap`, we could
not say — and it never folds a partial RRset."* The shape here is identical. An answer we cannot
tell from a synthesis is an answer we cannot read, and

> **an undiscriminated answer is never a value.** Where the control probe under a `Name`'s parent
> did not complete, that `Name`'s `resolution` and `dns-record` observations record a **`Gap`**
> stating its cause, and never a value.

Three things follow, and the third is the one to say out loud.

`Gap` is the right object and not a stretched one. ADR-0025 reserves it for *we did not look* and
refuses it for *we looked and got an answer* — and here we **did not look** at the parent, which is
the measurement `Shadowed` needs. `CONTEXT.md`'s `Gap` already opens on *a dead-lettered `Batch`'s
empty scope* and already **records its cause**; this is one more cause on an existing object, and a
`Gap` never withdraws a subject, so nothing cascades.

Nothing widens. `resolution`'s value space is untouched, which matters because
[ADR-0015](./0015-the-value-space-is-the-commitment.md) makes the value space the commitment and
`resolution` feeds `Reach` and `Exposure`. An `Undiscriminated` variant was the available
alternative and it is refused below.

And the fragility is real. If the control probe under the apex fails, every name in that sub-tree
holds a `Gap` for the cadence, and on a single-zone estate that is the whole estate going dark for
a night. That is the correct trade against a fictional inventory, it is the same shape as the
dead-lettered batch `Gap` already accounts for, and the failure it takes is a handful of queries
against the operator's own resolver — which, if it is failing, has already taken everything else
with it. It is flagged rather than smoothed.

The modal case needs stating so nobody reads this backwards: **a control probe that completes and
finds no wildcard licenses everything beneath it.** The `Gap` is for *incomplete*, never for
*wildcarded* and never for *not wildcarded*.

### The control probe runs the qtype set, because `Shadowed` was already promised on all of it

By-catch, ruled rather than ticketed, because leaving it open would leave a committed value
unemittable.

`CONTEXT.md` says `Shadowed` *"is a value on `dns-record` as well as `resolution`, since a wildcard
synthesises answers for **any** qtype"*, and `measurement-offers.md` §2 fixes the v1 qtype set at
**seven**. §3.2's procedure probes **three** — A, AAAA and CNAME. So under today's text a wildcarded
zone's synthesised MX, TXT, NS and SOA answers are recorded as that `Name`'s own `dns-record` RRsets
with no `Shadowed` available: `{161}` again, four qtypes wide.

The repair is to make no new list. **The control probe runs the batch's declared qtype set**, which
is an existing aperture input recorded on the `Batch` already, so the population is the only new
dimension and `measurement-offers.md` stays at five offers. The cost is the ×7/3 already priced into
the table above — about a second — and for the four qtypes a wildcard rarely carries the measured
result will be *no synthesis*, which is a negative that licenses `dns-record`'s absence rather than
a wasted query.

### The bound below, and the one wildcard we will never see

The parent of an apex is a TLD. `iana.org`'s parent is `org`, and a `*.org` would synthesise for
`iana.org` itself if `iana.org` did not exist — which is not hypothetical: VeriSign's Site Finder
wildcarded `*.com` and `*.net` in 2003.

We do not probe there, and the reason is already on the books rather than new.
[ADR-0002](./0002-ownership-gates-probing.md) gates probing on the estate, and `CONTEXT.md`'s
**subtree containment is label-wise suffix comparison** is the test: a probe site must sit inside a
`Seed` name scope. A TLD does not, so the population is bounded below by the seed and the exclusion
falls out of an existing rule instead of a carve-out. The residue is stated: **a wildcard at or
above the operator's own apex is undetectable by verge-asm and is out of scope by the probing
gate.**

### Where this is thin, stated rather than smoothed

- **The zone-cut index is killed with a measurement in one direction and a specification in the
  other.** That it *over*-selects is measured — thirteen of fifteen cuts carry no wildcard. That it
  *under*-selects is RFC 4592 §2.1.1's definition plus one measured estate, and the modal
  single-zone small org is an argument from the persona rather than from a count. Nobody has
  measured how often a wildcard sits at a non-cut name in the population verge-asm actually serves,
  because that population is small orgs' private zones and there is no public sample of it. Read the
  under-selection limb as `[spec]`, in `measurement-offers.md`'s sense.
- **The `[measured]` estate is one estate, and it is a registry rather than a small org.** IANA has
  25 names and six parents; a 300-name flat estate has 300 names and **one** parent, and the
  reduction the parent set buys is far larger there and untested here. The direction is safe — the
  parent set is never larger than the name set and is usually much smaller — but the 4.2× figure is
  one sample and should not be quoted as a rate.
- **Nothing has run a control probe inside a batch.** Every claim here about *what the leaf does*
  is a specification claim; what is measured is DNS behaviour against live authorities, from one
  vantage, on one day, over a resolver we do not control. §3.2 step 5's escape hatch — a name whose
  answer differs from the signature is genuinely present — is untested, and its converse residue
  is untouched by this ADR: **a name that genuinely exists and happens to carry the wildcard's own
  RRset reads as `Shadowed`**, and that is a match-predicate limit rather than a population one.
- **The match predicate is not this ticket's and it is measurably broken.** See the consequence
  below: two of four wildcarded zones return a **different** synthesised address set per control
  label, seconds apart, from one vantage, so a set-equality predicate suppresses nothing on them.
  This ADR names the defect and does not fix it, because the predicate is `wildcard-discrimination`'s
  declared parameter and moving one is ADR-0021's business. If that ticket lands badly, the
  population being right buys less than this ADR implies.

## Consequences

- **The enumerated aperture input list moves from six to seven**, and every site asserting an older
  count is struck at the clause under
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) —
  ADR-0007's #36 amendment, ADR-0009, ADR-0011, ADR-0013 and ADR-0025 in three places. Three of the
  five were **already stale**, carrying *five* or *three* after
  [ADR-0017](./0017-exposure-needs-both-legs.md) made it six; the correction names both moves.
- **[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) is amended and not
  overturned.** Its offer test is unreached: the control-probe population is not an offer, so
  *"this ticket adds none"* was true of #54 and is not a claim about the list's closure. Its
  Control-label-count row stands.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) is untouched.** Five leaves stay
  five, `wildcard-discrimination`'s declared parameters are unchanged, and no `Break` cause is
  added. Its table gains an inline pointer, because a session will look there.
- **[ADR-0011](./0011-a-facet-is-six-parts.md) is confirmed.** `Shadowed` is still decided by the
  measurement binary inside one batch: the control probe and the name's own answer are both in it,
  and the population is computable at batch start from the entering scope, so there is no in-batch
  discovery loop and no cross-observation dependency.
- **[ADR-0062](./0062-a-wildcards-synthesis-is-a-fact-about-the-name-it-was-probed-under.md) is
  amended in two places and its blocker is discharged.** *"Nothing declares which `Name`s are
  control-probed"* is now false, so the `wildcard-synthesis` facet's **sixth part is writable** and
  its stated reopening condition is met. And *"X is, by construction, a `Name` already in the
  estate — we do not probe under names we do not hold"* is **falsified by measurement**:
  `ns.iana.org` is a probe site, is NXDOMAIN, and is in no `Citation`. Its subject rule survives —
  *the `Name` the control labels were generated under* — and it acquires one open item, below.
- **The deferred facet reopens with one new open item, and no ticket.** A probe site that is not a
  held `Name` has **no carrier**, and inventing one would admit a subject from our own probe with no
  `Citation`, which ADR-0060 and ADR-0027 both bar. The facet is out of scope, has no deadline and
  costs `revealed` plus one message whenever it is taken up, so the item is recorded on the
  out-of-scope shelf rather than bought.
- **[`CONTEXT.md`](../../CONTEXT.md) changes in three entries and adds no term.** `Shadowed` gains
  the population rule and the `Gap`, and loses *"which names are control-probed is a live
  question"*; `Batch` names the control-probe population as a recorded scope dimension; `Name`'s
  wildcard clause points at the parent rule. The facet list stays six and the `Offer` list stays
  five.
- **[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §3.2's procedure is
  rewritten in place**, its *under the apex* and *repeat one level down for each discovered
  sub-zone* steps struck at the clause, and its A/AAAA/CNAME qtype clause replaced by the declared
  qtype set. §1's and §8's summary rows lose *for a zone*.
- **[`measurement-offers.md`](../spec/measurement-offers.md) gains one cross-reference in §2 and no
  section.** The population is not an offer and this document enumerates offers.
- **`{161}` closes on four qtypes.** MX, TXT, NS and SOA become `Shadowed`-capable on `dns-record`,
  which they were promised and could not be.
- **One new ticket, and it is the other half of making §3.2 implementable.** The **match
  predicate** is unspecified and measurably harder than set equality: **[measured]** `herokuapp.com`
  returned three distinct address sets across five control labels and `vercel.com` returned five,
  seconds apart, from one vantage, while `github.io` and `localtest.me` were stable. A predicate
  reading *the answer equals the recorded signature* suppresses nothing on the churning half, and
  every fictional name beneath reads `Resolved`. **Blocking
  [#12](https://github.com/winniel123/verge-asm/issues/12)** on the same ground this ticket was.
- **[ADR-0062](./0062-a-wildcards-synthesis-is-a-fact-about-the-name-it-was-probed-under.md)'s
  churn thinness gains its first measurement** — *"whether the signature churns on its own is not
  measured"* — in the label-to-label direction rather than the batch-to-batch one it named. The
  batch-to-batch rate is still unmeasured.
- **The cost is zero and there is no re-baseline.** Nothing has shipped, no control probe has run,
  no `resolution` timeline exists, so the seventh aperture input arrives free and the `Gap` rule
  withdraws nothing.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **The apex plus each measured zone cut** — the *losing option*, the ticket's own middle, and the one SOA-in-the-qtype-set makes look principled | It indexes **delegation** and a wildcard is not a delegation — RFC 4592 §2.1.1 makes it an ordinary owner name needing no cut at its parent. **[measured]** on IANA's estate exactly **one of six** parents is a zone cut, so option B probes the same single site as apex-only and leaves the same 11 of 25 names undiscriminated; and the modal verge-asm operator has **one zone**, so it degenerates to *the apex* for the persona the product is for. It over-selects too: **thirteen of fifteen** probed zone cuts carry no wildcard |
| **The apex alone**, §3.2 step 1 read literally | **[measured]** `render.com`'s apex control label is NXDOMAIN while `*.staging.render.com` answers `10.4.3.1` at every depth beneath it; and on IANA's estate it covers 14 of 25 names. §3.2 already qualifies itself in step 3, so this is not even the note's position |
| **Every `Name` in the estate** — the closed direction | It buys nothing where it costs most. A control label under N discriminates only names **beneath** N, and a `Name` with no held descendant has none — 19 of IANA's 25 sites are leaves. The cost is survivable (≈4 s at the 200 pkt/s ceiling) so this loses on **shape**, not on price, and saying otherwise would be a safety argument dressed over a tidiness one |
| **A declared parameter of `wildcard-discrimination`** — the runner-up, and the reading ADR-0025's Control-label-count row invites | A parameter is **authored data** (ADR-0021: *"the parameter set is declared data"*), and this population is a function of the batch's own scope. Filed there, discovering one subdomain with an unseen parent bumps the leaf and `Break`s `resolution` and `dns-record` across the name estate — the unsurvivable fallback rebuilt inside one leaf, on the most ordinary event the product has. The count and the population are two facts, so this is not the one-fact-two-homes trap #62 closed |
| **Restrict probe sites to `Name`s we hold**, preserving ADR-0062's *we do not probe under names we do not hold* | **[measured]** `ns.iana.org` is NXDOMAIN and in no `Citation`, yet `www.ns.iana.org` is a held `Name` whose closest encloser is `iana.org`. The restriction leaves it undiscriminated to protect a claim nothing depended on. A probe site is a **label sequence we construct**, not a subject we admit: no `Citation` is written and nothing enters the estate |
| **A new `Undiscriminated` variant in `resolution`'s value space** | ADR-0015 makes the value space the commitment and `resolution` feeds `Reach` and `Exposure`; a `Gap` says the same thing using an object that already exists, already records its cause, and already never withdraws a subject. ADR-0025's *a truncated answer is never a value* is the precedent on all fours |
| **A `Gap` on every name beneath an unprobed wildcard, as the standing v1 position** | Under this ruling there is no such name: the population is defined so that every held `Name` has its parent probed. The `Gap` is for a probe that **did not complete**, and reading it as *wildcards produce gaps* inverts the ruling |
| **Record the population as a rule on the `Batch`** — *the apex plus each parent*, rather than the set | `Batch` requires every scope dimension **by content**, because a widening is detected by diffing named dimensions and a rule cannot be diffed. It is ADR-0025's rejected library-version dimension one object across |
| **Mint a sixth `Offer` for the control probe** | Nothing about the population is a candidate on the wire, and the probe's only wire content — the qtype set — is an offer the batch already declares. A sixth offer would put a subject population in a document whose subject is what the binary puts on the wire |
| **A separate control-probe qtype list, smaller than the batch's** | It re-opens the `{161}` hole it was meant to close: `Shadowed` is committed on `dns-record` *for any qtype*, so any qtype outside the control list records a synthesised RRset as the name's own. And a second list is a sixth offer under another name |
| **Fold the match predicate in, since the measurement is in hand** | It is `wildcard-discrimination`'s **declared parameter** and ADR-0021's table is its home; moving one is a leaf bump with its own price and its own corpus row. Folding a parameter change into an aperture ruling hides it, which is the objection ADR-0062 made to folding this ticket into that one |
| **Probe under the apex's own parent, so a TLD wildcard is caught** | ADR-0002 gates probing on the estate and containment is label-wise suffix comparison over a `Seed`'s scope; a TLD is outside it. Site Finder is real and the residue is stated rather than closed |
