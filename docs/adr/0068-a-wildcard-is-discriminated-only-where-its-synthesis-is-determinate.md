# A wildcard is discriminated only where its synthesis is determinate, and determinacy is measured per component

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#111 What is `wildcard-discrimination`'s match predicate, given that a synthesised answer set is not stable across control labels?](https://github.com/winniel123/verge-asm/issues/111)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Discharges:** [ADR-0062](./0062-a-wildcards-synthesis-is-a-fact-about-the-name-it-was-probed-under.md)'s deferred **value space**, and [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s *the match predicate has no value anywhere*

## Context

[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)
settled **which `Name`s** the control probe runs under and closed the population door. It left the
other door open and said so: `wildcard-discrimination`'s **match predicate** is one of
[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s declared parameters, ADR-0021 names
it without giving it a value, and
[`project-authored-constants.md`](../research/project-authored-constants.md) §6.8 files it **Open —
no value has been chosen**.

[`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §3.2 supplies the only
prose there is: *record the wildcard answer set as a poison signature*, then *suppress any candidate
whose answer matches the poison signature*. Read as **set equality** — the only reading the text
supports — that fails against live authorities, and the failure is the same catastrophe §3.2 exists
to prevent, arriving through the predicate instead of through the population: the recorded signature
matches nothing, `Shadowed` is never emitted, and every fictional name beneath is recorded
`Resolved` with a fictional address set, admitting fictional `Address`es, `Service`s and
`Endpoint`s.

The measurement underneath this ADR is worse than the ticket's, and the difference decides the
ruling. The ticket found that *a synthesised answer set is not stable across control labels*. It is
not stable across **queries for one label**, either — including direct to a single delegated
authority — so the object §3.2 calls *the* answer set is not merely unstable, it does not exist.

## Decision

**A wildcard's synthesis is discriminated only at the components a control probe measured to be
determinate. A component that varied across the control labels is never consulted: it can neither
shadow a name nor exempt one. Where no component is determinate, every name beneath that parent is
`Shadowed` and §3.2's escape hatch is withdrawn there.**

| Concern | Decision |
| --- | --- |
| What the predicate compares | **Set equality on the RDATA set**, per **component** — a component being one `(qtype asked, RR type in the answer)` pair — and **only at components measured determinate** |
| What a *component* is, and why not *the answer* | The answer to one qtype is a **chain**, and its parts have different stabilities. **[measured]** `s3.amazonaws.com` rotates eight A addresses on every label while its CNAME component holds `s3-1-w.amazonaws.com.` still |
| What the signature **is**, given n control labels | Per component, a **closed union of three**: `NoSynthesis` · `Determinate(RRset)` · `Indeterminate`. Never *the* answer set, and **never a union of the observed sets** |
| Why never a union | The union is the object that licenses an intersection predicate, and recording it invites one back. It is not recorded, so it cannot be read |
| When a candidate is **discriminated** | When it **differs at some determinate component** — a different RRset where the control had one, or an RRset where the control determinately had none |
| When it is `Shadowed` | Whenever it is not discriminated. **The default beneath a measured wildcard is `Shadowed`**, and the predicate's job is to find the exemption, never the match |
| Is discrimination per qtype or per name? | **Per name.** RFC 4592 blocks synthesis for **every** type once the name exists, so a candidate discriminated at **any** component exists, and **none** of its answers is synthesised — including the ones that coincide with the signature |
| Which way it errs | **Toward `Shadowed`, by construction** rather than by preference. An unmeasurable component is refused the power to exempt |
| Is that stated? | **Yes, and it is required to be.** ADR-0011's *the leaf may not helpfully normalise* means the direction must be a decision |
| Is the match predicate a declared parameter? | **Yes, still**, and it now has a value. ADR-0021's table row and §6.8's **Open** row both close |
| Is the **determinacy verdict** a parameter? | **No, and not aperture either.** It is a **measured in-batch fact**, computed from the control probe's own answers — a parameter is authored data, and this is a function of what the authority said this batch |
| Does the aperture list move? | **No. It stays at seven.** Unlike ADR-0066's population, determinacy does not decide *which subjects were covered*; it decides what a covered subject's answer **means**, so no `Batch` scope dimension is added |
| Does `resolution`'s value space move? | **No.** `Shadowed` already exists and no member is added, so there is **no `Break`** |
| Which value space *does* move | **`wildcard-synthesis`'s** — ADR-0062's deferred closed union is **three** members, not the two it guessed. Its remaining blocker is discharged |
| Control-label count | **Untouched**, and still a parameter (ADR-0025). Determinacy reads the answers a count already fixed |
| `wildcard-discrimination` | **Still one leaf**, still five leaves total |
| Cost of the ruling | **Zero.** Nothing has shipped and no `resolution` timeline exists |

## Rationale

### The measurement: *the* answer set is not an object

**[measured]** 2026-08-14, Google Public DNS DoH JSON from one vantage, and — where stated — direct
to a delegated authority with `nslookup`.

Thirty long random labels under `herokuapp.com` returned **eight** distinct A answer sets, one per
`vaNN`/`ieNN` ingress node, **pairwise disjoint**, thirty addresses in total. Thirty under
`vercel.com` drew two addresses each from a closed pool of **eight**.

That is the ticket's finding. This is the one that decides the ruling:

| Probe | Result |
| --- | --- |
| One label, six repeats, via the public resolver, `vercel.com` | **five** distinct address pairs |
| One label, four repeats, **direct to `ns01.herokudns.net`** | **four** different ingress nodes — `ie01`, `va06`, `va02`, `va04` — with four disjoint address sets |
| One label, one moment, **four different authorities of `herokuapp.com`** | four different answers |

So the synthesised answer is not a function of the label, and the rotation is the **authority's own**
rather than an artefact of an anycast resolver in front of it. A "poison signature" recorded from a
control label is one draw from a process, and the next draw — for the same label, at the same
authority, seconds later — is a different one. Any predicate phrased as *the candidate's answer
versus the recorded answer* is comparing two samples and calling the difference evidence.

### Components, not answers — and this is what rescues the ticket's CNAME intuition

The answer to one qtype is a chain, and §3.2's *the answer set* flattens it. Split it by RR type and
the same zone reads completely differently:

**[measured]** five control labels per zone, all seven qtypes:

| Zone | Determinate components | Indeterminate components |
| --- | --- | --- |
| `github.io` | A (four `185.199.10x.153`), AAAA; NODATA at the other five | — |
| `localtest.me` | A (`127.0.0.1`), AAAA (`::1`); NODATA at the other five | — |
| `s3.amazonaws.com` | **CNAME (`s3-1-w.amazonaws.com.`)**, TXT, NS, SOA | A — eight fresh addresses on every label |
| `appspot.com` | **MX** (the five `gmr-smtp-in.l.google.com.` hosts, identical every label) | A, AAAA |
| `vercel.com` | NODATA at six qtypes | A |
| `herokuapp.com` | **none, at any qtype** | CNAME, at all seven — the target rotates over eight nodes |

The ticket proposes *the CNAME target rather than the address set*, on the strength of
`herokuapp.com` and `s3.amazonaws.com`. It is half right and its half is the wrong half. `s3` does
hold its CNAME target still; `herokuapp.com`'s rotates over **eight** targets, and over **four**
across four repeats of one label at one authority. Privileging CNAME **by name** gets `s3` right by
luck and `herokuapp` wrong.

Determinacy gets both right for a reason, and it does it without naming a single RR type: on `s3`
the A component is simply not consulted and the CNAME component decides, and on `herokuapp` nothing
decides. That is the ticket's own third candidate — *a per-qtype predicate, since a rotating A set
says nothing about a stable MX* — generalised one level down, from the qtype to the component,
because **[measured]** `s3` is the case where the rotating and the stable parts sit inside **one
qtype's answer**.

### ADR-0066's seven qtypes turn out to be load-bearing here, and that is measured

§3.2 step 1 originally probed **A, AAAA and CNAME**, and ADR-0066 replaced that with the batch's
declared qtype set on a `{161}` argument about MX, TXT, NS and SOA being unemittable. That widening
now pays a second time, for a reason nobody argued: **[measured]** `appspot.com`'s only determinate
**positive** component in the seven is **MX** — A and AAAA both rotate, and CNAME, TXT, NS and SOA
are NODATA. Under the withdrawn three-qtype clause that zone has no positive determinate component
at all and every name beneath it is suppressed. Under seven it discriminates.

### The base rate, which is what makes the strict rule affordable

**[measured]** nineteen zones, five long random labels each, A qtype:

| | Count |
| --- | --- |
| Wildcarded (a random label answered) | **14** |
| Not wildcarded (NXDOMAIN — `pages.dev`, `workers.dev`, `azurewebsites.net`, `fly.dev`, `repl.co`) | 5 |
| Of the 14: **determinate at A** across five labels | **10** — `github.io`, `localtest.me`, `traefik.me`, `vcap.me`, `netlify.com`, `staging.render.com`, `railway.app`, `onrender.com`, `surge.sh`, `glitch.me` |
| Of the 14: indeterminate at A | 4 — `herokuapp.com`, `vercel.com`, `appspot.com`, `s3.amazonaws.com` |
| Of those 4: **still carry a determinate component** at another qtype or RR type | 3 — `s3` (CNAME/TXT/NS/SOA), `appspot` (MX), `vercel` (a determinate NODATA at six qtypes) |
| **No determinate component anywhere** | **1 of 14 — `herokuapp.com`** |

So set equality is not wrong. It is **unscoped**, and it is right for ten of fourteen outright. The
ticket's *two of four* sampled four zones of which two were the pathological ones. Total suppression
— the expensive limb of this ruling — reaches **one measured parent in fourteen**, not half of them.

### The losing option is measured to buy nothing exactly where it is needed

The ticket's leading candidate is **non-empty intersection with the union of the observed
signatures**. It loses on arithmetic over the same thirty-label runs. Signature = the first five
labels; test = the next twenty-five.

| Zone | Set equality catches | Intersection-with-union catches |
| --- | --- | --- |
| `herokuapp.com` | **7 of 25** | **7 of 25 — identical** |
| `vercel.com` | 5 of 25 | **25 of 25** |

On `herokuapp.com` the synthesised sets are **disjoint blocks**, one per ingress node, so
*intersects the union* and *equals a recorded set* are the **same predicate**, and both leave **18
of 25 (72%)** fresh fictional labels recorded `Resolved` with a fictional address set. On
`vercel.com` the pool is eight addresses and each answer draws two, so a five-label union covers
seven of eight and nothing can miss it.

Intersection is therefore neither sound nor unsound in general. It is a **total fix or literally no
fix**, and which one it is depends on the shape of the provider's load balancing — a fact the
control probe cannot observe and the `Batch` cannot record. A predicate whose error rate is
unknowable in band is worse than one that refuses, because the refusal is legible and the error is
not.

### Which way it errs, and why the two errors are not comparable

The direction is not a preference and is not chosen for safety's sake. It falls out of refusing to
let an unmeasured thing carry weight — the same move as ADR-0025's *a truncated answer is never a
value* and ADR-0066's *an undiscriminated answer is never a value*. A component that varied is a
component we have no reading of, so it decides nothing in either direction.

What the direction costs is real and must be stated rather than absorbed:

- A **false `Shadowed`** withholds one `resolution` value. It is confined to one facet, it is
  named, the name **stays in the estate** — admission turns on its `Citation`, never on this answer
  — and it *"stays visibly unconfirmed until the operator supplies coverage or excludes it"*
  ([`CONTEXT.md`](../../CONTEXT.md)). §3.3's zone upload is the stated remedy and it is already a
  first-class onboarding step.
- A **false `Resolved`** fabricates an address set, and that set **cites `Address`es, opens
  `Service`s and `Endpoint`s, feeds `Reach` and `Exposure` and reaches the board**. Downstream it is
  indistinguishable from a true one.

One error is loud and confined to a facet. The other is silent and propagates across three subject
types. `CONTEXT.md` already accepted the collateral in as many words — the suppression *"working as
intended on the fictional names and swallowing the real one alongside them"* — so this ruling does
not introduce that trade. It measures it: **[measured]** five real GitHub Pages sites
(`github.github.io`, `mozilla.github.io`, `twbs.github.io`, `git-lfs.github.io`, `d3.github.io`)
return **exactly** the wildcard's four addresses, and `www.vercel.com` draws **both** its addresses
from the wildcard's own pool. Those names are not distinguishable from synthesis by any content
predicate whatever, loose or tight, and calling them `Shadowed` is not an error the model could
have avoided — it is the honest reading of what the authority said.

### Discrimination is a fact about the name, and that is RFC 4592 rather than a convenience

RFC 4592 §2.2.1 synthesises only where the query name has **no exact match**, and a name that exists
for any type has one. So a candidate discriminated at a single component **exists**, and *none* of
its RRsets is synthesised — including the ones that happen to equal the signature.

Two consequences, both stated so nobody re-derives them backwards. `Shadowed` is **all-or-nothing
across a name's qtypes**: a name is `Shadowed` on `resolution` and on every one of its `dns-record`
discriminators, or on none of them. And ADR-0066's stated converse residue — *"a name that genuinely
exists and happens to carry the wildcard's own RRset still reads as `Shadowed`"* — is **narrowed,
not closed**: with seven qtypes the name gets seven chances to differ somewhere, and it reads
`Shadowed` only when it coincides at every one. The GitHub Pages measurement above is what that
residue looks like when it survives.

### Where this is thin, stated rather than smoothed

- **Determinacy is evidence, not proof, and it is falsifiable — I falsified it.** **[measured]**
  `traefik.me` answers `127.0.0.1` for three random control labels and is determinate by this
  ruling's own test, while `10.0.0.1.traefik.me` → `10.0.0.1`, `192.168.5.5.traefik.me` →
  `192.168.5.5`, `8.8.4.4.traefik.me` → `8.8.4.4`. A synthesis that is a **function of the label**
  looks constant to random labels, and set equality then reports a fictional RFC 1918 address as
  `Resolved`. *n* random labels evidence a constant synthesiser; they cannot prove one, and no
  finite *n* can. This is `render.com`'s lesson from ADR-0066 arriving through the predicate.
- **And there is a third door that neither ADR-0066 nor this ruling closes.** **[measured]**
  `nip.io` and `sslip.io` return **NODATA** for random control labels while `10.0.0.1.nip.io` →
  `10.0.0.1`. §3.2 step 1 reports *no wildcard at all*, the probe completes, and ADR-0066's *a probe
  that completed and found no wildcard licenses everything beneath it* licenses a fictional
  inventory. Neither the population nor the predicate is the defect; the **control label's own
  shape** is, and it is `wildcard-discrimination`'s other declared parameter — *control-label count
  and construction* — which this ticket was not asked about and does not touch.
- **Every indeterminate zone measured is a third-party hosting provider's.** ADR-0066 intersects the
  population with the operator's `Seed`, which excludes most of what is measured here. The modal
  operator's `*.dev.example.com` in front of an ingress load balancer may or may not rotate, and
  **nobody has measured it**, because there is no public sample of small-org private zones — the
  same hole ADR-0066 flagged one question over. Read *1 of 14* as a rate over **provider** zones.
- **One vantage, one day.** `herokuapp.com`'s `ie0x` and `va0x` nodes are Ireland and Virginia, so
  part of the rotation is geographic, and ADR-0025 already calls EDNS Client Subnet *"a `Vantage` in
  an option's clothes"*. A multi-vantage run would see **more** rotation, never less: the direction
  is safe, and *10 of 14 determinate* is an **upper bound** on determinacy rather than an estimate.
- **No control probe has ever run inside a batch**, verbatim from ADR-0066. What is measured is DNS
  behaviour against live authorities.

## Consequences

- **New rule, one sentence:** a wildcard is discriminated only where its synthesis is determinate,
  measured per component, and an indeterminate component decides nothing.
- **[ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s leaf table is amended**: *the
  match predicate has no value anywhere* is discharged and the value is stated at the row.
- **[`project-authored-constants.md`](../research/project-authored-constants.md) §6.8's
  match-predicate row closes**, and the open-parameter count returns to the **two** its own §9 and
  §10 prose already claim — it had been three since ADR-0066 added the row.
- **[ADR-0062](./0062-a-wildcards-synthesis-is-a-fact-about-the-name-it-was-probed-under.md)'s value
  space is drawn and its last blocker discharged.** `wildcard-synthesis` is a closed union of
  **three**, per component: `NoSynthesis` · `Determinate(RRset)` · `Indeterminate`. The facet stays
  out of scope on price, with both its parts now writable.
- **`passive-discovery-sources.md` §3.2 amended in place and a new §11 added.** Step 2's *the
  wildcard answer set* and step 4's *whose answer matches the poison signature* are struck **at the
  clause** per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md),
  and **step 5's escape hatch is withdrawn as written** — it is sound only at determinate
  components, and `traefik.me` falsifies it as stated.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Shadowed` is amended at the site that specifies it.** *"when
  the answer matches a wildcard's measured poison signature"* is superseded by *"when the answer was
  not discriminated from the parent's synthesis"*. The entry's own justification — *we cannot see
  here is a fact the operator needs* — was already epistemic; the *matches* clause was the
  instrument standing in for the fact, which is the substitution
  [ADR-0013](./0013-custody-is-control-and-extends-by-declaration.md) and ADR-0062 have each caught
  once.
- **`resolution` does not move and nothing `Break`s.** No value-space member is added.
- **ADR-0066 confirmed and repaid.** Its population is untouched, its seven-qtype by-catch is
  measured load-bearing for this ruling, and its converse residue is narrowed.
- **Two new tickets, neither folded in.** The **DNSSEC wildcard proof** as a second discriminator,
  and the **structured control label** — see the rejected table for why each is a ticket rather than
  a clause here.
- **Cost: zero.** Nothing has shipped, no control probe has run, no `resolution` timeline exists.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Non-empty intersection with the union of the observed signatures** — the *losing option*, and the ticket's own leading candidate | **[measured]** it buys **nothing** where it is needed. `herokuapp.com`'s synthesised sets are pairwise disjoint blocks, so intersection and set equality are the **same predicate** there — both catch **7 of 25** and leave **18 of 25 (72%)** fictional labels `Resolved` — while on `vercel.com` it catches **25 of 25**. Its correctness is a property of the provider's load balancer that the probe cannot observe and the `Batch` cannot record, so the predicate's error rate is unknowable in band. It also requires **recording the union**, which is the one object that makes this predicate expressible |
| **The CNAME target rather than the address set** — the ticket's second candidate | **[measured]** half right, wrong half. `s3.amazonaws.com` holds `s3-1-w.amazonaws.com.` still; `herokuapp.com`'s CNAME target rotates over **eight** ingress nodes and over **four** across four repeats of one label at one authority. Privileging an RR type **by name** gets one zone right by luck. Determinacy subsumes the intuition and picks the right component per zone |
| **Set equality unscoped**, §3.2 step 4 read literally | **[measured]** it is right for **10 of 14** wildcarded zones and catastrophic on the rest — 5 of 25 on `vercel.com`. The defect is that it is unscoped, not that it is wrong, which is why the ruling keeps it and gates it |
| **A convergence stopping rule** — draw control labels until the union stops growing, then treat disjointness as discrimination | It recovers `api.vercel.com` (76.76.21.112, plainly outside the pool), which total suppression loses, so it is a genuine middle. It loses three times over: it makes the **control-label count a function of what the batch found**, which is exactly the *"a parameter whose value is whatever the batch found is no more declared than one whose value is whatever the library defaults to"* that ADR-0066 used to push the population out of the parameter table — so it needs an **eighth aperture input** and a second `Batch` scope dimension for a leaf that has not shipped; it has **no termination guarantee** against an authority whose synthesis is a function of the label (`traefik.me`, measured); and **[measured]** it does not buy the false positive it exists to buy, since `www.vercel.com` draws both its addresses from the wildcard's own pool and is suppressed anyway |
| **Abolish the escape hatch entirely** — shadow everything beneath any measured wildcard | The maximally safe reading, and it over-pays. **[measured]** **10 of 14** wildcarded zones are determinate at A across five labels, so wholesale abolition discards RFC 4592 §2.2.1's own licence on 71% of the population to fix the 7% where it does not hold. Determinacy is the smallest gate that separates them |
| **The DNSSEC wildcard proof** — RFC 4035 §3.1.3's short RRSIG `Labels` field plus NSEC/NSEC3 proving no closer match | The only **sound** discriminator that exists, and it is **[measured]** unavailable. Exactly **1 of 15** zones probed carries a DS (`herokuapp.com`) — and that one **online-signs its synthesised answers**: its RRSIG reads `cname 13 3 …`, `Labels` = **3** against a 3-label owner, with no NSEC3, so the proof that would discriminate is never served. It is missing precisely where content discrimination is also missing. It also changes the leaf's **query mode** (DO bit, RRSIG parsing, NSEC3 handling), which is a parameter change of its own. **Ticketed, not folded** |
| **A new `Undiscriminated` member in `resolution`'s value space**, distinct from `Shadowed` | `Shadowed` **is** that value once the *matches the signature* clause is read as the instrument it was. Adding a member to `resolution` is ADR-0015's expensive move and would `Break` every timeline of the facet, to name a distinction the operator does not act on differently — in both cases we cannot see here, and in both the remedy is the zone file |
| **Make the predicate an aperture input**, following ADR-0066's move on the population | The population decides **which subjects were covered**, which is what an aperture input is for and why the `Batch` records it by content. The predicate decides what a **covered** subject's answer means. Filing it as aperture would make every `Batch` record a rule rather than a scope, which is the *record the population as a rule* alternative ADR-0066 already refused, inverted |
| **Fold the structured-control-label hole in**, since `nip.io` was measured in the same sweep | It is the other declared parameter — *control-label count and **construction*** — and the failure is that a random label sees nothing, which no match predicate can repair. **[measured]** `nip.io`/`sslip.io` answer NODATA for random labels and `10.0.0.1.nip.io` → `10.0.0.1`. Folding a second parameter's defect into this one hides it, which is ADR-0062's objection to folding #108 into #103 and ADR-0066's to folding this ticket into #108. **Ticketed** |
