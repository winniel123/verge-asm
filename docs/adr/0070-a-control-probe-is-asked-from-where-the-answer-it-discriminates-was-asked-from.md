# A control probe is asked from where the answer it discriminates was asked from, and the query path is one declared parameter

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#116 Is the control probe's vantage a declared parameter, an aperture input, or a query-mode change?](https://github.com/winniel123/verge-asm/issues/116)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Discharges:** [`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §13.10's last bullet — *"the probe's vantage is a parameter nobody has declared … not ruled here, and not folded in"* — and the same hole one leaf over, in `resolution-walk`

## Context

Three rulings have closed three doors on `wildcard-discrimination`.
[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)
ruled the **population** — which `Name`s the control probe runs under.
[ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md) ruled the
**match predicate** that reads the answers.
[ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md), as
raised by [#115](https://github.com/winniel123/verge-asm/issues/115), ruled the **count and
construction** of the labels. Between them they say what to ask, how many times, under what name,
and what the answers mean.

**Nothing anywhere says where the probe is asked from**, and #115 measured that the answer changes
the verdict in both directions.

**[measured]** 2026-08-15,
[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §13.5 and §13.6:

- **`s3.amazonaws.com`** — the delegated authority returns the CNAME `s3-1-w.amazonaws.com.` and
  **no A record at all**, identical across repeats, labels and every one of its servers. The eight
  rotating addresses belong to a **separate delegation** the probe's own vantage followed across a
  zone cut. So the A component is `Indeterminate` over a resolver and **does not exist**
  direct-to-authority.
- **`vercel.com`** — reads `Determinate` over one resolver and `Indeterminate` at its authority,
  because that resolver's `Determinate` is a **30-minute cache artefact** at a 1800 s TTL, over an
  authority that gives **6** distinct answers to 8 repeats of one label.

So `Determinate` is not a property of the authority. It is a property of **the authority, the
vantage, and the cache in between** — and only one of those three is written down.

The model has the vocabulary and has not applied it here. ADR-0025 already calls EDNS Client Subnet
*"a `Vantage` in an option's clothes"* and rules that if v1 ever sends one it belongs **in the key,
never in the scope record**. A recursive resolver is the same phenomenon with the subnet left
unstated: **[measured]** below, one probe shape run at two vantages on one day draws
`vercel.com`'s synthesised addresses from two **entirely disjoint** pools.

And the hole is not confined to the leaf the ticket names. `resolution-walk` already queries the
delegated authorities directly — [`CONTEXT.md`](../../CONTEXT.md)'s `Lame` says so and makes it
load-bearing, *"a recursive resolver's SERVFAIL cannot tell a dead delegation from a bad
upstream"* — while §3.1 says resolution *"should run through the operator's own recursive
resolver"*. Both are true of one leaf, so the leaf holds **two answers to the same question**, and
nothing says which one becomes `Resolved(address set)`.

## Decision

**A control probe is asked from where the answer it discriminates was asked from. The query path —
direct to the delegated authorities, or through the `Vantage`'s configured recursive resolver — is
a declared parameter shared by `resolution-walk` and `wildcard-discrimination`, taking exactly one
value per `Batch`. Its value is the `Vantage`'s configured recursive resolver. Which resolver
stands there is part of the `Vantage`'s identity, in the timeline key, never in the scope record
and never in a leaf.**

| Concern | Decision |
| --- | --- |
| Which of the three the vantage is | **A declared parameter** — and a **shared** one, of `resolution-walk` **and** `wildcard-discrimination`. That shape already exists in [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s table: the EDNS option set and the DNS library are each a parameter of both leaves |
| Is it an **aperture input**? | **No. The list stays at seven.** ADR-0066's test is *does it decide which subjects were covered* — and this adds no probe site, removes no candidate and moves no name. The same parents are probed and the same names read; what moves is what a covered subject's answer **means**. Third application of that line, after ADR-0068 refused it to the predicate and ADR-0069 to the construction |
| Is it a **query-mode change `wildcard-discrimination` may not make**? | **Half right, and the right half is the load-bearing one.** *Query-mode change* is not a third home: ADR-0068 says of the DNSSEC discriminator that changing the leaf's query mode *"is a parameter change of its own"*. What is true is that `wildcard-discrimination` may not make it **on its own** — a path it chooses independently of `resolution-walk`'s is a **skew**, and the skew fabricates an inventory |
| The anti-skew clause, which is the whole ruling | **One parameter, one value, one `Batch`.** The control probe's answers and the candidate's own answer are drawn on the **same** path, or the predicate is comparing two vantages. This is ADR-0066's *the control probe runs the batch's declared qtype set, and no second list*, one dimension over |
| The **value** | **The `Vantage`'s configured recursive resolver.** Not direct-to-authority — see the rationale; it is the losing value and it loses on `{161}` rather than on cost |
| Does **`resolution-walk`** have the same hole? | **Yes, and worse.** It is not that nobody chose a resolver; it is that the leaf demonstrably makes **both** queries — a delegation walk for `Lame`, a resolution for the address set — and nothing said which answer is the value. **[measured]** the two differ: at `s3.amazonaws.com`, `NoData` at the authority against eight addresses over a resolver, at one instant |
| What the **delegation walk** keeps | `Lame` and per-nameserver `serves │ does-not-serve` are decided **at the delegated authorities**, and **this parameter does not govern them**. That is `Lame`'s own definition and it may not move — a parameter that could switch it off would delete a value |
| What the delegation walk may **never** supply | **An address set.** `Resolved(unordered address set)`, `NoData` and `NameError` are read on the **declared path** and never off the walk. A leaf holding two answers must be told which one is the value, and this is it |
| Is a determinacy verdict **measured over a resolver** admissible? | **Yes.** The verdict's job was never to describe the authority — [ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md) already files it as *a measured in-batch fact* and §13.9 already rules it *a claim about a vantage at a moment*. Under this ruling that vantage is the one the candidate's own answer is drawn from, which is the only vantage the predicate compares across |
| Is **§11.4's base rate** re-opened? | **No, and the reason is a measurement rather than a judgement.** §11.4 was measured with **five distinct** random labels per zone. The `Determinate` that is a cache artefact is §13.6's **one label, eight repeats**. *n* distinct labels are *n* distinct cache entries (§13.6's own structural reason), so the shipped ten-label set never takes the shape the artefact needs. **[measured]** 2026-08-15, ten distinct labels under `vercel.com` **through a recursive resolver** return **9 distinct A sets** — `Indeterminate`, exactly as §11.3 and §11.4 have it |
| The **operator's own resolver** | **A `Vantage` fact, not a parameter fact.** `CONTEXT.md`'s `Derivation` forbids an operator-configurable declared parameter outright; §3.6 nonetheless offers the operator a resolver choice. Both stand, because they are two facts: the **kind of path** is authored data in the leaf, and **which resolver stands at it** is part of the `Vantage` — declared intent, in the timeline key, outside every derivation |
| What switching resolvers therefore costs | A **new `Vantage`**, so the timelines it feeds **open** — `revealed` — under ADR-0025's *a dimension in the key opens*. It is **not** a `Break`, and it may not be one: an operator changing their upstream DNS must not clamp the estate |
| Does any **value space** move? | **No.** `resolution`, `dns-record` and `wildcard-synthesis`'s three-member per-component union are all untouched. Nothing is added and nothing is renamed |
| Does the **aperture list** move? | **No. Seven.** No `Batch` scope dimension is added; the path is a leaf parameter and the resolver is a key component, and neither is scope |
| `wildcard-discrimination` | **Still one leaf**, and this ruling adds none — total ~~five~~ **six** since [ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md). Both it and `resolution-walk` gain one named parameter, which arrives **with a value** |
| Cost of the ruling | **Zero.** Nothing has shipped, no control probe has run, no `resolution` timeline exists |

## Rationale

**[measured]** 2026-08-15, from a **second** vantage — a host resolver on a different network from
the one §11, §12 and §13 were measured at — with direct-to-authority runs against each zone's own
delegated servers. Every number in this section is new to the corpus and none of it was taken from
§13.

### The skew is the whole ticket, and it fabricates in one direction

Suppose the two paths could differ: the control probe direct to the authority, the candidate's own
lookup through the resolver. Take the zone the corpus already has the most evidence about.

**[measured]** direct to `ns-63.awsdns-07.com` (205.251.192.63), `s3.amazonaws.com`'s own delegated
authority, qtype A:

| Probe | Answer |
| --- | --- |
| 4 distinct random labels | `CNAME s3-1-w.amazonaws.com.` and **no A record**, plus a delegation referral — byte-identical across all four |
| 1 label, 4 repeats | the same, byte-identical |

**[measured]** the same zone, the same instant, through this vantage's recursive resolver:

| Probe | Answer |
| --- | --- |
| 3 distinct random labels | `CNAME s3-1-w.amazonaws.com.` → `CNAME s3-w.us-east-1.amazonaws.com.` → **eight A records**, a **different eight on every label** |

Now read the components. Direct to the authority, the `(A asked, A answered)` component is
**`NoSynthesis`** — no control label carried an A RR — and `NoSynthesis` is a **determinate**
reading, not an absent one. Through the resolver it is **`Indeterminate`**.

ADR-0068 discriminates a candidate that carries *"an RRset where the control determinately had
none"*. So under a skewed pair — control at the authority, candidate at the resolver — **every**
fictional label beneath `s3.amazonaws.com` differs at a determinate component, is discriminated, is
recorded `Resolved`, and cites eight fabricated `Address`es that go on to open `Service`s and
`Endpoint`s. That is §3.2's stated catastrophe in full, arriving through neither the population nor
the predicate nor the construction, but through a **disagreement between two settings**.

The reverse skew — control at the resolver, candidate at the authority — is the safe direction and
still wrong: `Indeterminate` at A, never consulted, everything beneath `Shadowed`. One error is
loud and confined; the other is silent and propagates across three subject types. ADR-0068's
asymmetry, unchanged.

**This is what kills the runner-up.** The ticket's cheapest answer — *a declared parameter of
`wildcard-discrimination`*, a row and a value in ADR-0021's table — is a defect the moment the
table has a row and `resolution-walk` does not, or has one with a different value. A parameter that
can be set to two things is a parameter that will be, and the failure is invisible in band: both
leaves are behaving exactly as declared. So the parameter is **one parameter**, held jointly, and
the rule that reads it is stated as an equivalence rather than as a pair of settings.

### It is a parameter, on both of the tests already on the books, and they agree

**ADR-0025's offer test.** *An offer is scope where any value it feeds carries a per-candidate
negative — we asked about X and X was not there — and then it is scope for every batch that makes
it; otherwise it is a declared parameter of the leaf that made it.* `resolution`'s value is not a
subset of the paths asked. There is no *we asked the authority and the authority was not there*
sitting inside a value; a query path that fails produces a `Gap`, which is ADR-0014's ordinary
adjacency and not an opening. That is the same shape ADR-0025 used to make the **DNS transport and
fallback policy** a parameter of `resolution-walk`, and the query path is that policy's sibling:
the transport is *how* we ask, the path is *whom* we ask.

**ADR-0066's aperture test.** *Does it decide which subjects were covered?* No. The population is
the parents of the names in scope, computed at batch start, unmoved by the path; the candidate set
is the names in scope, unmoved by the path. What moves is what a covered subject's answer means,
which is the predicate's side of ADR-0066's line — exactly where ADR-0068 put the match predicate
and ADR-0069 put the construction. Three questions in a row have landed on that line and it has not
had to be re-argued once.

Both tests give the same answer, which is the tell that it is the cut rather than a rationalisation.

### Why *query-mode change* is not a third home

The ticket's third limb is drawn from ADR-0068's refusal of the DNSSEC discriminator, and it is
worth quoting the whole clause rather than the half that reads as a category: *"It also changes the
leaf's **query mode** (DO bit, RRSIG parsing, NSEC3 handling), which is **a parameter change of its
own**."* ADR-0068 did not refuse the DNSSEC discriminator for being a query-mode change; it refused
it on **[measured]** availability — 1 of 15 zones carries a DS, and that one online-signs its
synthesised answers — and named the query-mode cost as a second, smaller reason, **classified as a
parameter change**.

So *query-mode change* is a description of what a parameter change costs, not a home a fact can
live in. Treating it as a third option would leave the fact homeless: a thing that is neither
authored data, nor a function of the batch's scope, nor read off the probe's answers, is a thing no
object in this model can record — and a fact with no home is exactly how this one went undeclared
through three rulings in the first place.

What the limb gets right, and what is kept, is the **may not make it** half. `wildcard-discrimination`
may not choose a path of its own. The clause is written into the Decision above rather than into a
category.

### The value: the operator's configured recursive resolver, and it loses nothing that matters

The competing value is direct-to-authority, and it has the better story on paper — no cache, no
middlebox, the authority's own words, and `resolution-walk` is already there for `Lame`. It loses
four times.

**It cannot produce an address set across a zone cut without becoming a resolver.** Measured above:
at `s3.amazonaws.com`'s own authority a synthesised name is a CNAME into a zone that authority does
not serve, returned with a referral. To reach an address from there the leaf must follow the chain
itself — which is to say, implement recursion, with its own cache, its own retry policy and its own
answers. [#5](https://github.com/winniel123/verge-asm/issues/5)'s *every seam is a place drift can
be manufactured* bites directly, and §3.6 has already ruled that the host's resolver is the thing to
use rather than the thing to rebuild.

**Its errors are the silent ones.** A direct-to-authority `Resolved` would be **empty** where the
resolver has eight addresses — `NoData` on a name that plainly resolves. That is an absent
`Address`, an absent `Service`, an absent `Endpoint` and a `Reach` leg that never opens:
[ADR-0009](./0009-verge-core-is-a-union.md)'s `{161}` in the direction that under-reports exposure,
which is the failure this product exists not to have.

**It is a claim about a zone rather than about an estate.** `Address` is in the estate *"exactly
while a current resolution cites it"*, and those addresses are then probed for listeners. An address
the operator's own network would never resolve to is not their exposure, and §3.1 already says why
the resolver is *"the cleanest possible source … it reflects what the operator's own network
actually sees"*. Overturning that silently is precisely the shape
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) exists to
prevent.

**And the sensitivity it buys is measured small.** The one thing direct-to-authority genuinely buys
is a determinacy verdict no cache can freeze. **[measured]** the shipped set does not need it:

| Probe, this vantage, 2026-08-15 | Result |
| --- | --- |
| `vercel.com`, **10 distinct** random labels, through the recursive resolver | **9 distinct A sets** → `Indeterminate` |
| `vercel.com`, **1 label × 8 repeats**, through the same resolver | **7 distinct** — this resolver does not freeze at all, at a reported TTL of 1800 |

The second row is worth more than the first. §13.6 measured **1** distinct answer for that same
probe over Google Public DNS at the same 1800 s TTL. So the cache annihilation §13.6 found is a
property of **that resolver**, not of resolvers — two vantages, one probe shape, opposite verdicts —
which is the ticket's own thesis arriving as a measurement rather than as an argument, and it is why
the resolver has to be recorded rather than assumed.

**The affordability check the value is owed**, in ADR-0069's form — the path must not turn honest
verdicts into different ones on the ordinary zone:

| Zone | Resolver, 4 distinct labels | Authority, 4 distinct labels | Agreement |
| --- | --- | --- | --- |
| `github.io` | 1 set — the four `185.199.10x.153` | 1 set — the same four | **byte-identical** |
| `netlify.com` | 1 set — `18.208.88.157`, `98.84.224.111` | 1 set — the same pair | **byte-identical** |
| `railway.app` | 1 set — `34.107.141.139` | 1 set — the same | **byte-identical** |

**3 of 3.** The path choice is invisible on constant wildcards and decides only on the pathological
ones — which is the same result ADR-0069's structured label got, for the same reason: a measurement
that only moves where something is already wrong is cheap to make strict.

### The operator's resolver is a `Vantage`, and that is what resolves the apparent contradiction

Two rules in the glossary look incompatible until this fact is placed.

> A **declared parameter** is authored by the project and ships in the release, and **none is ever
> operator-configurable** … An operator's dial may sit anywhere **outside** every derivation and
> nowhere inside one. — `CONTEXT.md`, `Derivation`

> verge-asm should resolve via the host's own resolver by default … and offer DoH only for
> operators whose environment blocks outbound 53 or who distrust their local resolver. — §3.6

If the resolver were the parameter, §3.6's offer would be an operator dial inside a leaf — the one
actor `Derivation` says could break the estate without a release and without a corpus row moving,
and it would leave two installs on one release comparing as comparable while holding different
content behind one leaf. That is not a wrinkle; it is the exact failure the sentence was written
against.

The split that dissolves it is the one the model already draws everywhere else between *what the
project decided* and *where we were standing*:

- **The kind of path** — recursive, or direct to the delegated authorities — is **authored data**,
  one value, in the leaf, not operator-configurable. It is the thing a corpus row can be written
  against and a version bump can be justified by.
- **Which resolver stands at that path** is a property of the **`Vantage`**: *a network position
  observations are made from, declared as intent*. It sits outside every derivation, it is already
  in the timeline key `(subject, facet, discriminator, vantage, source)`, and a `Batch` is already
  *from one vantage*.

ADR-0025 reached this conclusion for the neighbouring case and did not generalise it: EDNS Client
Subnet is *"a `Vantage` in an option's clothes … which is the job the `vantage` component of the
timeline key already does"*, and if v1 ever sends one **"ECS belongs in the key, never in the scope
record"**. A recursive resolver is ECS with the subnet implicit — it makes a geo-aware authority's
answer a function of where the query appeared to come from, which is §11.9's own reading of
`herokuapp.com`'s `ie0x`/`va0x` split. The measurement makes it concrete rather than analogous:
**[measured]** `vercel.com`'s wildcard draws its addresses from `64.239.109.0/24` and
`64.239.123.0/24` at this vantage, against the `76.76.21.x`-family pool §11 recorded at another —
**two disjoint pools for one name on one day**. If that is not a vantage difference, nothing is.

Three consequences follow, and all three are things the model already knows how to do.

- **A self-hosted install probing from wherever the operator put it is one `Vantage` with one
  resolver**, so everything above is free on the modal deployment. The measurements in §11–§13 and
  in this section are *different* vantages, which is why they disagree, and the model has always
  said two vantages hold two true facts rather than forcing an arbitration.
- **Switching resolvers opens rather than breaks.** Vantage is in the key, and ADR-0025's rule is
  *a dimension in the key opens; a dimension inside the value breaks*. So the new resolver's
  timelines are `revealed` and the old vantage's are handled by the existing
  `Vantage`-becomes-`unavailable` machinery `Gap` already opens on. An operator changing their
  upstream DNS gets a coverage message, never an estate-wide `Break` and never a night of false
  `Transition`s.
- **The external prober is already a second vantage** ([#14](https://github.com/winniel123/verge-asm/issues/14)),
  and it resolves through whatever its host resolves through. Under this ruling that is stated
  rather than accidental, and its answers are separated from the instance's by the key rather than
  averaged with them.

### Why a resolver-measured determinacy verdict is admissible, and §11.4 is not re-opened

This is the limb that could have cost something, so it is argued rather than asserted.

**The verdict was never a claim about the authority.** ADR-0068 files determinacy as *a measured
in-batch fact, computed from the control probe's own answers*, and §13.9 states it in as many words:
*`Determinate` is a claim about a vantage at a moment*. What the predicate does with it is compare
the control probe's answers against **the candidate's answer** — and under the anti-skew clause
those are drawn on one path, in one batch, from one vantage. A verdict that describes the authority
would be describing something the predicate never touches. §13.5's own conclusion says the same
thing from the other end: the `Indeterminate` at `s3`'s A component is *"correct about what the
instrument sees and wrong about the authority it names"*, and correct-about-the-instrument is the
property the predicate needs.

**The cache artefact belongs to a probe shape the ruled set does not have.** §13.6's `Determinate`
over a resolver is **one label, eight repeats**. §13.6's own structural sentence says why that does
not generalise: *"n distinct labels are n distinct cache entries, so each one is a fresh miss and a
fresh draw at the authority."* The shipped set is **ten distinct labels** and repeats none of them,
and a candidate name is asked once per qtype, each qtype being its own cache entry. So the shape
that produces the artefact is unreachable inside a batch — and across batches ADR-0011 already
forbids carrying the verdict, which #115 made load-bearing rather than tidy.

**And §11.4's base rate was measured in the right shape.** Its header is *nineteen zones, **five
long random labels** each, A qtype* — distinct labels, over a resolver. That is the shipped shape at
a smaller *n*, not the repeat shape, and §11.3's and §11.4's own tables read `vercel.com` as
**`Indeterminate` at A** rather than `Determinate`. The two readings the ticket puts in tension are
readings of two different probes, and the base rate is the one the product will run.

Confirmed rather than argued from: **[measured]** ten distinct labels under `vercel.com` through a
recursive resolver, this vantage, 2026-08-15 → **9 distinct A sets**. `Indeterminate`. Nothing
re-opens, no ticket is owed, and §11.4 stands at **8 of 14**.

**The residual direction is the safe one.** A resolver adds variation rather than removing it: it
follows chains across zone cuts and splices other zones' rotation into the answer (§13.5, and
measured again above at `s3`), and it may sit at a different geography from the authority. §11.9
already priced that direction — *"a multi-vantage run would see more rotation, never less … 8 of 14
determinate is an upper bound"* — and it is the direction that yields more `Indeterminate`, hence
more `Shadowed`, which is ADR-0068's chosen error.

### `resolution-walk`'s hole is the same hole and it is one sentence wider

The ticket asks whether the sibling leaf has the same defect. It does, and stating it precisely
matters because the obvious statement is wrong.

The wrong statement is *nobody said which resolver `resolution-walk` uses*. The true one is that
**the leaf already makes both queries and nothing said which answer is the value.** `Lame` is only
available *"because the measurement binary queries the delegated authorities directly"*, and
ADR-0025 wrote a prober obligation about that walk — *no `BADCOOKIE` or cookie-driven `REFUSED` may
produce `Lame`*. Meanwhile §3.1 puts resolution through the operator's resolver. So the walk hands
the leaf an authoritative answer to the very name it is resolving, and the resolver hands it
another, and `NameError │ NoData │ Lame │ Resolved(unordered address set)` is one closed union with
two candidate inputs.

**[measured]** the two are not interchangeable: at `s3.amazonaws.com`, at one instant, the
delegated authority answers a synthesised name with a CNAME and no address while the resolver
answers it with eight. Read off the walk, the value is `NoData`; read off the resolver, it is
`Resolved` with eight addresses. Nothing in the corpus chose.

One member of that union was already committed and nobody noticed it decided the others.
`CONTEXT.md`'s `Name` says a name *"leaves when **our own resolver** measures a Name Error from every
available vantage"* — so `NameError` has always been read on the recursive path. A union whose
members are read on two different paths is not a closed union of one measurement; it is two
measurements wearing one value space, and a name could hold `NameError` on one path while the other
had an answer for it. Fixing `Resolved` to the same path is therefore not a new commitment so much as
the completion of one the glossary made and left partial.

The ruling therefore has two halves and both are needed:

> **`Resolved`, `NoData` and `NameError` are read on the declared path.** The delegation walk
> decides `Lame` and the per-nameserver `serves │ does-not-serve` RRset on `dns-record`'s NS
> discriminator, and **supplies no address set**.

> **This parameter does not govern the delegation walk.** Setting the path to *recursive* does not
> and may not stop the walk happening — `Lame`'s availability is its own definition's, not a
> setting's, and a parameter able to delete a value would be a settings field that silences a
> finding.

That second half is stated because the first, read alone and in the present tense, invites it: *the
value is read on the recursive path* sounds like *the leaf resolves recursively*, and a competent
session building from that sentence alone would drop the walk and lose `Lame`.

### Where this is thin, stated rather than smoothed

- **The measurements here are a second vantage, one day, and one host resolver whose behaviour is
  itself the finding.** This vantage's resolver does not honour `vercel.com`'s 1800 s TTL across
  eight back-to-back repeats; Google Public DNS did. Whether that is forwarding, an anycast pool, or
  a shortened cache is **not isolated** — the design that would isolate it is §13.2's, and it was not
  run here because the ruling does not turn on the mechanism. It turns on the fact that two vantages
  disagree, which is what is measured.
- **The value is chosen on `{161}` and on §3.1, not on a measured comparison of estates.** Nobody has
  run a whole batch either way and counted the `Address`es each path admits. The direct-to-authority
  under-report is argued from one zone's chain (`s3`) plus the structure of CNAME-across-a-cut; the
  frequency of that shape in a small org's estate is **unmeasured**, and it is the same hole
  ADR-0066, ADR-0068 and ADR-0069 each flagged one question over — there is no public sample of
  small-org private zones.
- **A hijacking or filtering resolver shadows the whole estate, and that is left as the loud
  failure.** A resolver that answers every name — captive portals, NXDOMAIN-monetising ISPs — makes
  every control probe find a wildcard, so every parent is wildcarded, most components come back
  `Indeterminate`, and every name beneath is `Shadowed`. It is legible, confined to one facet, and
  the remedies already exist and are already first-class (§3.3's zone upload, §3.6's DoH fallback).
  It is **not ticketed** and the reason is stated rather than assumed: the honest reading of a
  resolver that lies about everything is that it can tell us nothing, which is what total
  suppression says.
- **The resolver is recorded as declared intent, so an upstream that moves under a stable
  declaration is invisible.** An operator who declares *the host's own resolver* and whose host
  changes upstream by DHCP gets moved answers with no key change, and the model cannot see it. That
  is the same class as `Vantage`'s existing *declared as intent and re-verified every batch* residue
  and it is not repaired here; re-verifying a resolver's identity would need a probe whose answer is
  itself path-dependent.
- **No control probe has ever run inside a batch**, verbatim from ADR-0066, ADR-0068 and ADR-0069.
  What is measured is DNS behaviour against live authorities and live resolvers.

## Consequences

- **New rule, one sentence:** a control probe is asked from where the answer it discriminates was
  asked from, and the query path is one declared parameter with one value per batch.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s leaf table gains one
  parameter, shared by two leaves, arriving with a value.** `resolution-walk` and
  `wildcard-discrimination` each name the **query path**; the value is the `Vantage`'s configured
  recursive resolver. The parameter set changing bumps both leaves under ADR-0021's gate and
  `Break`s `resolution` and `dns-record`, which is **free while nothing has shipped** — the same
  price ADR-0069 paid for the count.
- **[`project-authored-constants.md`](../research/project-authored-constants.md) §6.8 gains one row,
  filed *Product?* = **No**, and the open-parameter count does not move.** It stays at **two** — the
  availability window (§6.4) and the capped body read (§6.8) — because this parameter arrives
  valued. §6.8's own caution about the *Product?* column answering *is a world quantity in here*
  rather than *does this have a value* applies to the new row and is why it is filed with its value
  written into the cell.
- **[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md) is amended and not
  overturned.** Its offer test is applied and passed — the query path carries no per-candidate
  negative — so it joins the DNS transport policy as a parameter of `resolution-walk`. Its ECS
  clause is **generalised at the clause**: a recursive resolver is a `Vantage` in a nameserver's
  clothes on the same ground ECS is one in an option's, and the *belongs in the key* half now has a
  second instance and a measurement.
- **[ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md)'s
  determinacy row is qualified at the site that specifies it**
  ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)). *"reads
  `Determinate` for the 1800 s its TTL is cached over a resolver"* is true of **one label repeated**
  and false of the ruled ten-distinct-label set; read alone and in the present tense it would have a
  session believe the shipped probe certifies `vercel.com`. Its ruling is otherwise **untouched**
  and its `Indeterminate` limb reaches every case here unchanged.
- **[ADR-0011](./0011-a-facet-is-six-parts.md) is confirmed and used rather than amended.** Its
  confinement of `Shadowed` to two measurements inside one batch is the premise the anti-skew clause
  is derived from: two paths would be two vantages, a `Batch` is *from one vantage*, and the two
  measurements would land in two batches. The confinement is **strengthened** — it now forbids a
  skew inside the batch as well as an assembly across batches — and nothing in it is weakened.
- **[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)
  and [ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md)
  are untouched.** Population, count and construction do not move; the aperture list stays at
  **seven**; `measurement-offers.md` stays at five offers and gains nothing, the query path being no
  more a candidate on the wire than the population was.
- **[`CONTEXT.md`](../../CONTEXT.md) changes in three entries and adds no term.** `Vantage` states
  that the resolver it resolves through is part of the declared position and therefore of the key;
  `Shadowed` states that the control probe and the candidate's answer are drawn on one path in one
  batch; `Lame` states that the delegation walk is not governed by the query path and supplies no
  address set.
- **[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) gains §14** carrying
  the measurements, **§13.10's last bullet is struck at the clause** — it says *not ruled here* and
  it is ruled now — and **§3.1's and §3.2's procedures state the path**, §3.1's *should run through*
  becoming a rule rather than a recommendation and §3.2 step 1 naming the path the ten labels are
  asked on. §11.4 gains a one-line rider recording that its five labels were **distinct**, which is
  what keeps it standing.
- **`resolution-walk` acquires a stated answer to a question it was already holding two answers
  to.** That is the largest thing this ticket found and it was not in its title.
- **Nothing is ticketed.** The three residues above — the unmeasured estate comparison, the lying
  resolver, and the moving upstream under a stable declaration — are disclosed rather than bought,
  and none of them blocks [#12](https://github.com/winniel123/verge-asm/issues/12).
- **[#12](https://github.com/winniel123/verge-asm/issues/12) is unblocked on this axis.** ADR-0021's
  table is complete for both DNS leaves, and implementation can be handed a probe that knows where
  to ask from.
- **Cost: zero.** Nothing has shipped, no control probe has run, no `resolution` timeline exists.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A declared parameter of `wildcard-discrimination` alone** — the *runner-up*, the ticket's own cheapest limb, *"a row and a value"* | It is the answer that would have shipped the defect. A path settable on one leaf and not the other, or settable to two values, is a **skew**, and **[measured]** the skew fabricates: direct to `ns-63.awsdns-07.com` the `(A, A)` component under `s3.amazonaws.com` is **`NoSynthesis`** — a *determinate* reading — while through a resolver every candidate beneath carries eight A records, so every fictional name *"differs at a determinate component"* under ADR-0068, is discriminated, and is recorded `Resolved` with a fabricated address set. Both leaves would be behaving exactly as declared, so nothing in band would show it. The parameter is one parameter, held jointly, with one value per `Batch` |
| **An aperture input, following ADR-0066 on the population** — the *losing option* | ADR-0066's test is *does it decide which subjects were covered*, and the path decides no such thing: the same parents are probed, the same names read, no probe site is added and no candidate removed. It decides what a **covered** subject's answer means, which is the predicate's side of that line — the third time that line has sorted a question, after ADR-0068 refused aperture to the match predicate and ADR-0069 to the construction. Filing it as aperture would also put it in the `Batch` scope record, where **[measured]** it does not belong: a widening is detected by diffing named dimensions, and *resolver* against *authority* is not a widening or a narrowing of anything |
| **A query-mode change `wildcard-discrimination` may not make**, as a category | Not a home. ADR-0068's own sentence classifies a query-mode change as *"a parameter change of its own"*, and it refused the DNSSEC discriminator on **[measured]** availability rather than on query mode. A fact that is neither authored data, nor a function of the batch's scope, nor read off the probe's answers has no object in this model to live in — which is how this one stayed undeclared through three rulings. The limb's *may not make it* half is **kept**, as the anti-skew clause, and its category half is refused |
| **Direct to the delegated authorities, as the value** | The better story on paper and it loses four ways. It **cannot produce an address set across a zone cut without implementing recursion** — **[measured]** `<random>.s3.amazonaws.com` at its own authority is a CNAME into a zone that authority does not serve, returned with a referral — which is #5's *every seam is a place drift can be manufactured* and §3.6's rebuilt resolver. Its errors are the **silent** ones: `NoData` where the resolver has eight addresses is an absent `Address`, `Service`, `Endpoint` and `Reach` leg, `{161}` in the under-reporting direction. It answers a question about a **zone** where the product's question is about an **estate** — §3.1's *what the operator's own network actually sees*. And the sensitivity it buys is **[measured]** unnecessary: ten distinct labels through a resolver read `Indeterminate` on `vercel.com`, because *n* distinct labels are *n* cache entries |
| **The resolver's identity as the declared parameter**, rather than the path's kind | `CONTEXT.md`'s `Derivation` forbids it in one sentence — *none is ever operator-configurable* — and §3.6 offers the operator exactly this choice. A settings field inside a leaf is the one actor that can `Break` the estate without a release and without a corpus row moving, and it would leave two installs on one release comparing as comparable while holding different content behind one leaf |
| **The resolver as a `Batch` scope dimension**, recorded by content beside the control-probe population | Scope licenses absences — *we covered X, so silence about X means X was not there*. A resolver licenses no absence; it conditions every answer equally. And ADR-0025 already ruled the sibling case in the other direction: ECS *"belongs in the **key**, never in the scope record"*, and a recursive resolver is ECS with the subnet implicit |
| **A new `Vantage class`, or a second class for *resolved-through*** | `Vantage class` is which side of the operator's **boundary** a prober sits on, verified against declared address scopes, and it is what `Reach` is measured per and what makes `Exposure` a conclusion across two legs. A resolver choice moves neither the boundary nor a leg. Adding a class would put a third leg into a composition ADR-0017 built on exactly two |
| **Let the delegation walk supply the address set, since the leaf already has one** | **[measured]** it is `NoData` where the resolver has eight addresses, so it would systematically under-report exposure — and it is not even complete, the authority returning a referral rather than a chain. The walk's job is `Lame` and the per-nameserver RRset, and it is kept at exactly that |
| **Make the query path govern the delegation walk too**, for symmetry | Setting it to *recursive* would delete `Lame`, whose own definition makes direct-to-authority load-bearing: *a recursive resolver's SERVFAIL cannot tell a dead delegation from a bad upstream*. A parameter that can silence a value is the offer-the-operator-can-narrow objection arriving inside the project's own release |
| **Rule resolver-measured determinacy inadmissible**, and re-measure §11.4 direct-to-authority | The expensive limb, and it rests on a misreading the corpus itself supplies the correction for. §11.4 was measured with **five distinct** labels; the cache artefact is §13.6's **one label, eight repeats**. Distinct labels are distinct cache entries, so the shipped ten-label set cannot take the artefact's shape — **[measured]** confirmed at ten labels through a resolver on `vercel.com`, 9 distinct sets, `Indeterminate`. Ruling it inadmissible would re-open a base rate that was never measured in the shape it was accused of |
| **Give the determinacy verdict a shelf life or a vantage field**, now that the vantage is named | #115 answered this and the answer is unchanged: the shelf **is** the batch, ADR-0011 already supplies it, and the verdict is never carried across. Naming the vantage adds no field because the vantage is already in the key of everything the verdict feeds |
| **Ticket the lying-resolver case** | The failure is loud, confined to one facet, and already remedied by two first-class things (§3.3's zone upload, §3.6's DoH fallback). Total suppression is the **honest** reading of a resolver that answers everything, not a defect in the reading. Disclosed as a residue rather than bought, on ADR-0040's *disclose the searched corpus* posture |
