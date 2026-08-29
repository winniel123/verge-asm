# A control label is one label, and the set must be able to falsify label-independence

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#113 Is a random control label the right construction, given that a synthesis can be a function of the label?](https://github.com/winniel123/verge-asm/issues/113)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Discharges:** [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s *control-label count and construction*, the last declared parameter of `wildcard-discrimination` with no value. It also discharges [ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md)'s **two** stated residues — the label-function synthesis and the third door
- **Amended in place by:** [#115 Is a wildcard authority's answer instability per-label, per-query or per-time?](https://github.com/winniel123/verge-asm/issues/115) — **the random count moves from `5` to `9`**, so the set is **9 random + 1 structured**, ten labels per site. The **construction** is untouched: one label, hyphenated quad, RFC 5737 space, the declared qtype set. #115 amends here rather than minting an ADR beside this one, because this is the site that specifies the parameter and a rule stated in two places is what [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) exists to prevent. Its measured basis is [`passive-discovery-sources.md`](../research/passive-discovery-sources.md) **§13**

## Context

[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)
ruled **which `Name`s** the control probe runs under.
[ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md) ruled the
**match predicate** that reads the answers. Neither ruled what the label *is*, and both assumed
[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §3.2 step 1's *3–5 long
random labels*. ADR-0068 then falsified that assumption twice, in its own words, and ticketed it:

| Zone | Long random label | A label with content | What §3.2 concludes |
| --- | --- | --- | --- |
| `nip.io`, `sslip.io` | **NODATA** | `10.0.0.1.nip.io` → `10.0.0.1` | *no wildcard* — the probe **completes**, and ADR-0066 licenses everything beneath it |
| `traefik.me` | `127.0.0.1`, every time | `10.0.0.1.traefik.me` → `10.0.0.1` | **`Determinate`** — so set equality certifies the component and reports a fictional RFC 1918 address as `Resolved` |

Both are §3.2's stated catastrophe — an unbounded fictional inventory — arriving through the one
door ADR-0066 and ADR-0068 each declined to close. The population door is shut and the predicate
door is shut. **A random label is the one shape that cannot see either failure**, because both
authorities are computing an answer from the label and a random label carries nothing to compute
from.

The architecture underneath all three rulings is RFC 4592's guarantee that synthesis is a function
of the **wildcard RRset** and not of the query name. Where that guarantee holds, one label speaks
for every label. `nip.io` and `traefik.me` are not RFC 4592 wildcards at all — they are authorities
that parse the query name — so the guarantee is absent and nothing in the probe notices.

## Decision

**A control label is exactly one label, and a control-label set that cannot falsify
label-independence is not a measurement of it. The set is ~~5~~ **9** random labels plus 1 structured label
whose content an address-parsing authority will decode; every label runs the batch's declared qtype
set; and the verdict is delivered by ADR-0068's existing `Indeterminate` limb, which needs no
amendment to reach it.**

> **The count in that sentence was `5` and is `9`
> ([#115](https://github.com/winniel123/verge-asm/issues/115)).** Read alone and in the present
> tense the original would build the set the *Raise the random count above 5* row below refuses, and
> that refusal is itself withdrawn at its own row. Everything else in this ruling stands verbatim.

| Concern | Decision |
| --- | --- |
| The rule in one sentence | **A control label is one label, and the set must be able to falsify label-independence** |
| **Label depth** | **Exactly one label** under the probe site, always. This is not new — it is what ADR-0066's own warrant already requires, stated |
| Why one label is forced | ADR-0066 admits the probe *because* "a control label constructed under P falls off the tree at the same closest encloser the names under P do". Every candidate is **one** label under its parent. A multi-label control label has ancestors between it and P, and where one exists the name falls off at a **different, deeper** encloser — so it measures a wildcard that is not the one the candidates meet |
| The set | ~~**5 random labels + 1 structured label**, six per site~~ **9 random labels + 1 structured label, ten per site** ([#115](https://github.com/winniel123/verge-asm/issues/115)) |
| The random half | ~~**5**~~ **9**, a **value** where §3.2 had a range `3–5`. A declared parameter may not be a range: ADR-0021's gate diffs parameter values, and a range has no diff. The move from 5 to 9 is a value moving, which the same gate handles: it bumps the leaf and `Break`s `resolution` and `dns-record`, free while nothing has shipped |
| Why **9** and not 5 | **[measured]** the mechanism the count is bought against is now identified — **per-label sharding** — and on the one zone with a two-member pool (`surge.sh`, `188/172` over 360 labels) the false-`Determinate` rate is **6.7% at six draws and 0% in 30 trials at eight and above**. Ten draws puts the modelled miss at **0.21%**, under a 1% bar across the whole confidence interval of the measured split. §13.7, §13.8 |
| Why the count and not **spacing** or **repeats** | **[measured]** no authority in the population is per-time over 35 minutes, so spacing buys nothing; and repeats are annihilated by the resolver the probe sits behind — `vercel.com` gives **6** distinct answers to eight repeats at its authority and **1** over a resolver honouring its **1800 s** TTL. Distinct labels are distinct cache entries, so each is a fresh draw. §13.4, §13.6 |
| The structured label | One label, of the form `<a>-<b>-<c>-<d>`, the four octets of an address drawn at random from **RFC 5737** documentation space (`192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`) |
| Why hyphens and not dots | A dotted quad is **four labels**, which the depth rule refuses. **[measured]** it is also strictly dominated: it detects no authority the hyphenated form misses, and manufactures a wildcard on **3 of 5** genuinely un-wildcarded zones |
| Why RFC 5737 | Same job as *long random*: make accidental existence negligible. It also keeps our own traffic legible in an operator's DNS log, which a random routable quad does not |
| Which qtypes the structured label runs | **The batch's declared qtype set**, all seven, exactly as the random labels do. A component is defined over *all n* control labels, so a label covering fewer qtypes leaves that definition ill-formed |
| **What reads the result** | **Nothing new.** ADR-0068's three-member union already closes over it: labels that disagree at a component make it `Indeterminate`, whether they disagree in an RRset's contents or in whether an RRset was carried at all |
| So what happens at `traefik.me` | A varies across the set → **`Indeterminate`** → never consulted → every name beneath is `Shadowed`. The certified-and-wrong verdict is **removed rather than graded** |
| So what happens at `nip.io` | The structured label answers where the random labels did not, so the probe is in the *wildcard measured* branch and the **licence is not issued** |
| **The "no wildcard" licence** | Qualified at ADR-0066's row and §3.2 step 6: a probe finds *no wildcard* only where **no control label of any shape carried an RR at any qtype**. Under a heterogeneous set that is a stronger test than it was, and it is the whole fix for the third door |
| Does `Determinate` get a footing of its own? | **No** — see the rationale. The ticket's own instance stops being determinate under this ruling, and a tier with no second grade of evidence to award is not a tier |
| Declared parameter, aperture, or value space? | **Declared parameter**, and ADR-0021's existing row is where it lives. It is **authored data** — a written enumeration of shapes — not a function of the batch's scope (which is what made ADR-0066's population aperture) and not a fact read off the probe's answers (which is what made ADR-0068's determinacy verdict neither) |
| Does the aperture list move? | **No. It stays at seven.** The construction adds and removes no probe site and no candidate; it changes what a probe at an already-chosen site can see, which is the predicate's side of ADR-0066's line |
| Does any value space move? | **No.** `resolution` does not move, and `wildcard-synthesis`'s three-member per-component union is untouched — it already covers this |
| `wildcard-discrimination` | **Still one leaf**, this ruling adds none (total ~~five~~ **six** since [ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md)), and the parameter set is unchanged in shape — a named parameter gains a value |
| Query cost | ~~**6 labels × 7 qtypes per parent**, against 5 × 7 today — **+20%**. On the `%.iana.org` estate §3.2 measures, 6 parents: **252 resolver queries per batch against 210**~~ **10 labels × 7 qtypes per parent — 70 per site, and 420 resolver queries per batch on that estate** ([#115](https://github.com/winniel123/verge-asm/issues/115)). Still resolver queries rather than packets at a target host, so `safe-active-probing.md` §6.3's per-target ceilings do not bind, and ADR-0066's arithmetic against the 200 pkt/s global ceiling is untroubled. **That *resolver* is now ruled rather than assumed** — [#116](https://github.com/winniel123/verge-asm/issues/116) / [ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md) makes the query path a declared parameter shared with `resolution-walk`, one value per `Batch`, valued at the `Vantage`'s configured recursive resolver, so this cost row's unit is correct by ruling and not by luck |
| Cost of the ruling | **Zero.** Nothing has shipped, no control probe has run, no `resolution` timeline exists |

## Rationale

**[measured]** 2026-08-15, Google Public DNS DoH JSON from one vantage. Twenty-two zones.

### One label is ADR-0066's own warrant, and the dotted quad breaks it — measured

ADR-0066 does not admit the control probe because probing is cheap. It admits it because of an
equivalence: a label constructed under P falls off the tree at exactly the encloser the names under
P fall off at, so what the probe measures is what the candidates will meet. Every candidate is
**one** label under its parent — that is what *immediate parent* means.

A dotted quad is four labels, and the equivalence does not survive it:

| Zone | `<random>` | `10-0-0-1` (one label) | `10.0.0.1` (four labels) |
| --- | --- | --- | --- |
| `pages.dev` | NXDOMAIN | NXDOMAIN | **`172.66.44.200`, `172.66.47.56`** |
| `workers.dev` | NXDOMAIN | NXDOMAIN | **`104.21.31.174`, `172.67.178.236`** |
| `fly.dev` | NXDOMAIN | NXDOMAIN | **NODATA** |
| `azurewebsites.net`, `repl.co` | NXDOMAIN | NXDOMAIN | NXDOMAIN |

**3 of 5** un-wildcarded zones are made to look wildcarded, and 2 of them hard. The mechanism is not
inferred — it was retrieved:

| Probe | Answer |
| --- | --- |
| `1.pages.dev` | `172.66.44.200`, `172.66.47.56` |
| `zz9q7x.1.pages.dev` | `172.66.44.200`, `172.66.47.56` |
| `1.workers.dev` | NODATA |
| `zz9q7x.0.1.workers.dev` | `104.21.31.174`, `172.67.178.236` |

`1.pages.dev` is a **third party's Cloudflare Pages project**, and it carries a wildcard beneath it.
`10.0.0.1.pages.dev` is therefore synthesised by `*.1.pages.dev`, two labels below the probe site,
by a party with no relation to the operator. A dotted-quad control probe under `pages.dev` measures
that stranger's wildcard and files it as `pages.dev`'s — and under ADR-0068 the disagreement with
the random labels reads `Indeterminate`, shadowing **every** name the operator holds under
`pages.dev` on evidence about somebody else's subtree.

So the dotted quad is refused on ADR-0066's warrant rather than on taste, and the measurement is
what that refusal costs to ignore. It is also **strictly dominated**: all three parsing authorities
decode the hyphenated form too, so the dotted quad detects nothing extra and manufactures three
false wildcards.

### The structured label separates a parser from a wildcard, and it does it in one label

| Zone | `<random32>` ×2 | `10-0-0-1` | `192-168-5-5` | `203-0-113-7` | Verdict |
| --- | --- | --- | --- | --- | --- |
| `nip.io` | NODATA | `10.0.0.1` | `192.168.5.5` | `203.0.113.7` | **parser** |
| `sslip.io` | NODATA | `10.0.0.1` | `192.168.5.5` | `203.0.113.7` | **parser** |
| `traefik.me` | `127.0.0.1` | `10.0.0.1` | `192.168.5.5` | `203.0.113.7` | **parser** |
| `localtest.me` | `127.0.0.1` | `127.0.0.1` | `127.0.0.1` | `127.0.0.1` | constant |
| `github.io` | the four `185.199.10x.153` | identical | identical | identical | constant |

RFC 5737 space is decoded exactly as RFC 1918 space is, on all three, so the structured label can be
**randomised over a reserved range** — it keeps *long random*'s defence against accidental existence
instead of trading it away for detection.

The `nip.io` row is the third door closing. Under a random-only set the A component is
`NoSynthesis`, the probe reports *no wildcard*, and ADR-0066 licenses everything beneath. Under this
set one label carries an A RR and five do not, which is ADR-0068's *they disagreed* — so the
component is `Indeterminate`, the licence is withheld, and **no amendment to the three-member union
was needed to get there**. The `traefik.me` row is the second residue closing, by the same limb.

And it reaches the awkward case, which was worth checking rather than assuming. A quad-shaped
candidate has a quad-shaped **parent**, and the probe still fires there:

| Probe | Answer |
| --- | --- |
| `<random32>.0.0.1.nip.io` | NODATA |
| `10-0-0-1.0.0.1.nip.io` | **`10.0.0.1`** |

### The affordability measurement: it costs nothing on genuine wildcards

The strict direction is only affordable if adding a structured label does not convert honest
`Determinate` verdicts into `Indeterminate` ones. Across 22 zones it converts **none**:

| Class | Zones | Verdict change |
| --- | --- | --- |
| Address-parsing authorities | `nip.io`, `sslip.io`, `traefik.me` | **3 changed, all correctly** |
| Constant wildcards | `github.io`, `netlify.com`, `railway.app`, `onrender.com`, `glitch.me`, `staging.render.com`, `localtest.me`, `vcap.me`, `lvh.me`, `local.gd`, `localho.st`, `fbi.com` | 0 — byte-identical RRsets across every shape |
| Un-wildcarded | `pages.dev`, `workers.dev`, `azurewebsites.net`, `fly.dev`, `repl.co` | 0 — NXDOMAIN on every single-label shape |
| No synthesis at A | `1u.ms` | 0 |
| Already unstable | `surge.sh` | 0 — it moves under random labels alone |

**0 false `Indeterminate` in 22 zones**, against 3 catastrophes prevented. That is the whole
affordability case, and it is why the structured label is admitted rather than the escape hatch
abolished.

### The count moves off `3–5` to `5`, and it is ~~refused a larger value on the measurement~~ **RAISED to `9` by #115**

> **The refusal below is WITHDRAWN at this site by
> [#115](https://github.com/winniel123/verge-asm/issues/115)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).** Read
> alone and in the present tense, *the number stays at 5* and *more labels is not monotonically more
> sensitive against this process* would build the wrong set. The random count is **9**. What holds
> verbatim is the *ground* of the refusal — **a count bought against an unidentified process cannot
> be priced in band** — which is exactly why #115 isolated the process first and moved the number
> second. The process is **per-label sharding**, and against it **[measured]** the count is
> monotone: 13.3% false `Determinate` at 4 draws, 6.7% at 6, **0 of 30 at 8 and above**. The single
> ten-label run tabulated below is a draw from a 0.2% tail that **30 fresh trials did not
> reproduce**. See [`passive-discovery-sources.md`](../research/passive-discovery-sources.md)
> §13.3, §13.7 and §13.8.

The count has to become a **value** whatever else happens: ADR-0021's gate is bidirectional on a
changed declared parameter, and *3–5* cannot be diffed. It goes to the top of the range because
ADR-0068 gave it a second job — sampling a varying answer — and nothing argues for fewer.

~~Raising it beyond 5 was the live question, and it is **refused on evidence that points the other
way**.~~ **[measured]** `surge.sh`:

| Run | Labels | Distinct A answers |
| --- | --- | --- |
| Five distinct random labels | 5 | **2** — `159.203.50.177`, `159.203.159.100`, alternating |
| Ten distinct random labels, minutes later | 10 | **1** — `159.203.50.177` |
| One label, six repeats | 1 | 1 |

~~The larger run saw **less** variation than the smaller one, so *more labels* is not monotonically
more sensitive against this process~~, and the mechanism — per-label sharding, per-query rotation, a
time window, or resolver caching in front of any of them — ~~**was not isolated**~~ **is per-label
sharding, isolated by #115**. A count bought
against an unidentified process is a purchase whose value cannot be stated in band, which is exactly
the objection ADR-0068 used to kill intersection-with-the-union. ~~The number stays at 5 and the
question is ticketed instead.~~ **The question was ticketed as
[#115](https://github.com/winniel123/verge-asm/issues/115), which identified the process and moved
the number to 9 at a stated price.**

Two things follow that are worth writing down rather than leaving implied. ADR-0068's base-rate
table files `surge.sh` under *determinate at A across five labels*. **Today it does not reproduce**,
so *10 of 14* has at least one member that is a coin-flip rather than a fact. That does not weaken
ADR-0068 — it is one more zone needing the gate, and the gate is what ADR-0068 ruled. And the
structured label contributes to this defence for free: it is a ~~sixth~~ **tenth** draw
([#115](https://github.com/winniel123/verge-asm/issues/115) counts it as one for exactly the reason
this sentence gives), so the set's power
against a varying answer rises even though the random count does not.

### `Determinate` gets no footing of its own

The ticket asks, on good grounds: `traefik.me` was the case where a determinate verdict is *correct
about the probe and wrong about the world*, and grading evidence is what a footing tier
([ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md)) is
for. It is refused, four times over.

- **The instance is gone.** Under this ruling `traefik.me` is `Indeterminate`. The defect is
  **removed at the measurement** rather than annotated, which is the better repair whenever it is
  available.
- **There is no second grade to award.** A footing tier grades *evidential distance* — how many
  premises a reader supplies between a source's sentence and the proposition. Determinacy is
  computed from one body of evidence, the control probe's own answers, and there is no weaker or
  stronger version of it to distinguish. A tier that cannot vary is not a tier.
- **A tier grades an attestation, and this is a measurement.** ADR-0059's instrument exists for a
  curated table of what owners say about the world. Determinacy is a measured in-batch fact
  (ADR-0068) with no owner, no curator and no artefact to re-retrieve.
- **[#33](https://github.com/winniel123/verge-asm/issues/33) already refused this shape** on the
  interface side: a per-row evidence tier is severity arriving labelled as honesty, it names no act
  the operator can take, and its real consumer is a curator who does not exist here.

What survives the refusal is a **residue**, and it is disclosed as a searched corpus rather than as
a caveat ([ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md)) — see below.

### Where this is thin, stated rather than smoothed

- **The corpus of encodings is four, and one ships.** Probed: dotted-decimal (**refused** on depth),
  hyphenated-decimal (**ships**), hex (`cb007107` → `203.0.113.7` on `nip.io` and `sslip.io`.
  `traefik.me` returns its constant and does **not** decode it), and hyphenated IPv6 (`2001-db8--1`
  → `2001:db8::1` on all three). Both extras are **measured redundant**: hyphenated-decimal is
  decoded by **3 of 3** parsers and **0 of 3** decode any other family without also decoding it, so
  it is the base form and the rest are optional extras. An authority whose *only* decodable form is
  one we do not send remains invisible. **The boundary is falsifiable by naming one such
  authority**, and it is a bounded residue rather than a permanent one.
- **The corpus is three parsers, all public developer conveniences.** ADR-0066 intersects the
  population with the operator's `Seed`, so the modal operator will never probe `nip.io`. Whether a
  small org runs an address-parsing authority inside its own zone is **unmeasured**, and it is the
  same hole ADR-0066 and ADR-0068 each flagged one question over.
- **A structured label that happens to exist produces a false `Indeterminate`.** The reserved-range
  space is 3 × 254 forms, far smaller than a 32-character random label's. The consequence is
  **total suppression of that parent** — the safe direction, and ADR-0068's own posture — never a
  fabricated address, so the risk is priced as legibility lost rather than inventory invented.
- **One vantage, one day, over a resolver we do not control.** ~~`surge.sh`'s two answers were not
  separated from cache behaviour, which is precisely why the count is not moved on them.~~
  **They have since been separated direct-to-authority by
  [#115](https://github.com/winniel123/verge-asm/issues/115), the cache is not in it, and the count
  moved to 9.** What remains thin is narrower and is stated at
  [`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §13.10: the count buys
  against *balanced* pools and cannot reach a **rare** second answer — `(1-f)^n` misses a
  5%-share member 60% of the time at ten draws — so a determinate verdict is still evidence and
  never proof, exactly as ADR-0068 has it.
- **No control probe has ever run inside a batch**, verbatim from ADR-0066 and ADR-0068. What is
  measured is DNS behaviour against live authorities.

## Consequences

- **New rule, one sentence:** a control label is one label, and the control-label set must be able
  to falsify label-independence.
- **`wildcard-discrimination` has no parameter without a value.** ADR-0021's row and
  [`project-authored-constants.md`](../research/project-authored-constants.md) §6.8's row both
  close. **The open-parameter count does not move** — it stays at **two**, the availability window
  (§6.4) and the capped body read (§6.8), because §6.8 filed this row under *Product?* = **No**
  rather than *Open*. That is worth recording rather than passing over: a parameter carrying a
  **range** instead of a value, and a construction nobody had specified, sat inside a *No* cell
  through two rulings, because the column answers *is a world quantity in here* and not *does this
  have a value*. §6.8 is annotated so the next audit of that table does not repeat the reading.
- **ADR-0068's two stated residues are both discharged**, by one instrument and with no amendment
  to its ruling: its three-member union, its per-component determinacy and its `Indeterminate` limb
  all reach these cases unchanged. Its *structured control label* ticket closes. The DNSSEC one was
  ruled out of scope separately.
- **ADR-0066's licence clause is qualified at the site that specifies it**
  ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)): *a
  probe that completed and found no wildcard licenses everything beneath it* now requires that **no
  control label of any shape carried an RR at any qtype**. Read alone and in the present tense the
  unqualified sentence would license `nip.io`, so it is amended rather than pointed at.
- **`passive-discovery-sources.md` §3.2 step 1 and step 6 amended in place, §11.7's second residue
  struck, and a new §12 added** carrying the measurements.
- **`CONTEXT.md` needs no new term and no amended definition.** `Shadowed` already reads *"when the
  answer was not discriminated from the parent's synthesis"* (ADR-0068), which covers an
  indeterminate component without change. One stale clause is corrected — see below.
- **The aperture list stays at seven** and no `Batch` scope dimension is added.
- **Nothing `Break`s in the model's shape.** The parameter's value moving bumps
  `wildcard-discrimination` under ADR-0021, which is free while nothing has shipped.
- **One ticket, not folded in:** is a wildcard authority's answer instability per-label, per-query
  or per-time — and does the control-label count buy anything against it? It does not block
  [#12](https://github.com/winniel123/verge-asm/issues/12): the parameter has a value and the
  question is whether that value can be improved. ***Now closed as
  [#115](https://github.com/winniel123/verge-asm/issues/115)***, which **amended this ADR in place**
  rather than ruling beside it: the instability is **per-query** at `herokuapp.com` and
  `vercel.com`, **per-label** at `appspot.com` and `surge.sh`, and **per-time at none of them**. The
  count **does** buy, it is the only lever of the three that survives a recursive resolver, and the
  random half moves **5 → 9**. The construction this ADR rules is untouched. Measured basis:
  [`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §13.
- **Cost: zero.** Nothing has shipped, no control probe has run, no `resolution` timeline exists.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A dotted-quad structured label** — the ticket's first candidate, and the form both measured failures were originally found with | It is **four labels**, which breaks the equivalence ADR-0066 admits the probe on: a multi-label control name falls off the tree at a deeper encloser than the candidates do. **[measured]** it manufactures a wildcard on **3 of 5** un-wildcarded zones, and the retrieved mechanism is a **third party's** project subtree — `1.pages.dev` exists and wildcards beneath it, so the probe files a stranger's wildcard as the operator's parent's and shadows every name under it. It detects **no** authority the hyphenated form misses |
| **A label drawn from the candidate's own shape** — the ticket's second candidate | It is the strongest instrument on paper and it is unaffordable and circular. Unaffordable: the probe count becomes a function of the **candidate** population rather than the parent population — on §3.2's own `%.iana.org` sample, 25 candidates against 6 parents — and ADR-0066 records the population on the `Batch` by content, so it multiplies what every `Batch` carries. Circular: the one label that would settle a candidate is the candidate itself, and its answer is what we are trying to interpret. A near-miss of `api` says nothing about `api`, and where it does say something — an address-shaped candidate — the fixed structured label already fires, **[measured]** including at the quad-shaped parent (`10-0-0-1.0.0.1.nip.io` → `10.0.0.1`) |
| **Refuse to generalise from any zone whose answers vary with label content** — the ticket's third candidate | **This is what the ruling does**, and it needed no new rule to do it. ADR-0068's `Indeterminate` limb already refuses to generalise from a component whose control labels disagreed; the only defect was that random labels never made them disagree. Writing a second refusal rule would restate ADR-0068 one term over, and a rule restated in two places is what ADR-0058 exists to prevent |
| **Ship the hex and IPv6 shapes as well**, at one extra label each | **[measured]** redundant on the corpus. Hyphenated-decimal is decoded by **3 of 3** parsers; hex by 2 of 3 (`traefik.me` does not decode it); hyphenated-v6 by 3 of 3, and **no** parser decodes any family without also decoding hyphenated-decimal. So the extras reach no finding and prevent no measured falsity, which is [ADR-0030](./0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md)'s bar unmet, and each is a 7-query multiplier per parent. The residue is disclosed as a searched corpus instead |
| ~~**Raise the random count above 5**~~ **RAISED — this row is withdrawn** | ~~**[measured]** refused on evidence pointing the other way: ten random labels under `surge.sh` saw **one** answer where five had seen **two**. The mechanism was not isolated, and against a time-varying or cache-mediated process a larger *n* buys nothing at all. Buying a count against an unidentified process is a purchase whose value cannot be stated in band — ADR-0068's own objection to intersection-with-the-union. **Ticketed**~~ **[#115](https://github.com/winniel123/verge-asm/issues/115) isolated the mechanism and reversed this.** It is **per-label sharding** over a near-fair binary hash, not a time window and not the cache: **[measured]** direct to `ns1.surge.world`, one label repeated eight times returns **one** answer and eight distinct labels return **two**, the label→answer map reproduces byte-for-byte 35 minutes later, and all four authorities agree. Against that process the count is monotone — **[measured]** 13.3% false `Determinate` at 4 draws, 6.7% at 6, **0 of 30** at 8 and above. The count is **9 random + 1 structured**. The one-answer ten-label run this row rested on is a 0.2% tail event that 30 fresh trials did not reproduce |
| **Buy the sensitivity with *spacing* instead** — the second of the ticket's three levers | **[measured]** refused: no authority in the population is per-time. Run C re-probed each zone's own labels at +5, +10, +20 and +35 minutes; the per-label maps did not move a single answer, and the per-query zones redraw whether spaced or not, so spacing buys precisely what one more back-to-back query buys at minutes of wall clock instead of milliseconds. It would also serialise the control probe at the head of every batch, and — fatally for a **declared parameter** — no spacing is long enough against a deploy-driven window, so the value would be unprincipled in exactly the way the range `3–5` was |
| **Buy it by *repeating one label* rather than drawing distinct ones** — the third lever, and the **losing option** | It is the only lever aimed at the mechanism that is most common in the sample (per-query, 2 of 5 zones), and it still loses, on a measurement. The probe stands **behind a recursive resolver** — this ADR's own cost row counts *resolver* queries — and *n* repeats of one label are **one** cache entry while *n* distinct labels are *n*. **[measured]** `vercel.com` answers **6 distinct** address pairs to eight repeats at its authority and **1** over Google DoH, at an authority TTL of **1800 s**: a repeat sees one answer for the whole batch and every batch for the next half hour. Distinct labels take one fresh draw each from a per-query rotation *and* are the only lever that reaches per-label sharding, so they **weakly dominate repeats at every measured mechanism and strictly dominate at three** |
| **Give `Determinate` a footing tier** | The instance that motivates it — `traefik.me` — **stops being determinate** under this ruling, so the defect is repaired at the measurement rather than annotated. Beyond that a tier has nothing to grade: determinacy comes from one body of evidence with no weaker or stronger form, ADR-0059's instrument grades **attestations** rather than measurements, and [#33](https://github.com/winniel123/verge-asm/issues/33) already refused a per-row evidence tier as severity arriving labelled as honesty |
| **Read the answer for the address encoded in the label** — an *the authority echoed our label* test | It is sound and it is **inert**: the verdict it produces is the verdict `Indeterminate` already produces, total suppression beneath that parent, so it buys no different outcome for a new mechanism. It would also be a rule reading the **content** of an answer to decide what the answer means, which is [#31](https://github.com/winniel123/verge-asm/issues/31)'s line, and it is unnecessary to cross it |
| **Make the construction an aperture input**, following ADR-0066 on the population | ADR-0066's line is that aperture decides **which subjects were covered** and the predicate decides what a **covered** subject's answer means. The construction adds no probe site and no candidate — the same parents are probed and the same names read — so it sits on the predicate's side, exactly where ADR-0068 put the match predicate. It is also **authored data**, which is the test ADR-0066 used to push the population *out* of the parameter table, applied in the other direction |
