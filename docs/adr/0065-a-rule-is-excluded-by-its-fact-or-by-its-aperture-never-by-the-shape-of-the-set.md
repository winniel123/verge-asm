# ADR-0065: A rule is excluded by its fact or by its aperture, never by the shape of the set

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#104 Does `smb-signing-not-required` become a v1 signal, now that the principle it was excluded on has been withdrawn?](https://github.com/winniel123/verge-asm/issues/104)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`insecure-listener-rules.md`](../research/insecure-listener-rules.md) §9.2 excluded
`smb-signing-not-required` from v1 on **two** grounds, and called the row *the closest call in the
note*:

> it is **integrity rather than confidentiality, so it fits neither rule**, and ~~a rule built for it
> would cover exactly one protocol — which is §9.1's per-protocol signal wearing a general-sounding
> name~~

[ADR-0015](./0015-the-value-space-is-the-commitment.md) withdrew the second — *"The doubt is correct
and **the principle as stated is wrong**"* — and
[#102](https://github.com/winniel123/verge-asm/issues/102) struck it in place under
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), leaving the
verdict standing on one ground and ticketing it here.

**The surviving ground does not survive either, and it is worth being precise about why.** *It fits
neither rule* is not a claim about SMB. It is a claim about **the two rules §9 happened to admit**.
Read in the present tense it says: the fact is real, spec-defined and credential-free, and we are
dropping it because our set has no box shaped like it. That is a reason to cut a third box.

ADR-0015 saw this and said so in the same breath it withdrew the other ground — *"it would be a
third signal in any case — signing is **integrity**, which is neither of the two facts the other
rules read"* — treating *third* as the **shape** of the answer rather than as an objection to it.
So §9.2's two grounds are **both** gone, and the note's own §12 q2 flagged the exposure before either
of them fell: *"a single-protocol rule may be legitimate, and this note has not established that it
is not."*

**This confusion has a population, and this is its second measured instance.**
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s #35 amendment named the first and named it
generally:

> So a candidate rule is admitted on the cadence test alone. … **A bar erected to protect the list
> rather than the property would have excluded #35 on exactly that confusion.**

[#8](https://github.com/winniel123/verge-asm/issues/8) and
[#17](https://github.com/winniel123/verge-asm/issues/17) each declined a rule *because the v1 set was
closed*, and #35 ruled both wrong — the closed set was a fact about the session that produced it, not
a gate. §9.2 is the same move one turn further out: not *the set is closed* but *the set is this
shape*. ADR-0015's own Alternatives table records the third instance in one line — *"Keep §9.2's rule
that a single-protocol signal is illegitimate: **excludes rules for the wrong reason**."*

What is missing is the positive rule. ADR-0004 says what admits a rule; nothing says what may
**exclude** one, so an exclusion has been free to rest on whatever the excluding session found
persuasive — and §9.2's ground rotted in eleven tickets while the verdict it carried stayed correct
for a reason nobody had written down.

## Decision

**An exclusion of a candidate `Signal` names which of exactly two grounds it rests on, and the shape
of the existing set is neither.**

| Ground | What it says | Half-life |
| --- | --- | --- |
| **The fact** | The rule is **inadmissible**. Its reference data fails ADR-0004's cadence test; or it is named for a conclusion its evidence cannot carry ([ADR-0010](./0010-exposure-composes-two-reaches.md)); or it is named for a **protocol** rather than the fact it reads ([ADR-0015](./0015-the-value-space-is-the-commitment.md)); or no owner attests the claim ([ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)) | **Permanent** until the world, an owner or a name moves. This is the *wrong* kind of out-of-scope entry |
| **The aperture** | The rule is **admissible** and the observation does not exist in this release — no facet holds the value, or no exchange produces it | **Dissolves** exactly when the aperture lands. Priced at `revealed` plus one message and **no `Break`** ([ADR-0014](./0014-only-revealed-generalises.md), ADR-0015), so it is *deferrable with no rework* |
| ~~The shape of the set~~ | *It fits none of the rules we have* · *the set is closed* · *it would be the only one of its kind* | **Not a ground.** It is a fact about the set, and the set is our own artefact |

Three riders bind.

**An admissible rule with no aperture is not `not-evaluable` — it renders nothing at all.** ADR-0004's
#44 amendment settled this: where no subject exists because we never looked, *"no row is possible,
ever"*, and the honesty lands on `Coverage` as a standing statement rather than on the rule. So
admitting a rule ahead of its aperture does not buy a visible gap the operator can act on; it buys a
name in the spec with an empty census behind it.

**The aperture cost is weighed and is never a correctness objection**, unchanged from ADR-0004: *"the
new measurement it requires … is a **scope** cost to be weighed, never a correctness objection."*
This ADR adds the converse — a scope cost that has been weighed and declined is a **complete**
exclusion and needs no principle propping it up.

**The exclusion is recorded with its ground on the map's out-of-scope line**, which is where the two
half-lives are already distinguished in practice — *"Unlike most entries here this is **wrong**
rather than deferrable"* — and where the condition that reopens it belongs.

## Rationale

### The two grounds fail differently, which is the whole reason to separate them

An aperture exclusion is **self-repairing**. It states a fact about this release's measurement
surface, and the day the surface widens the exclusion is discharged by construction — nobody has to
notice that the reasoning went stale, because the reasoning never was about the candidate.

A principled exclusion is **load-bearing prose**, and prose rots. §9.2 is the measurement: its
principle was withdrawn by ADR-0015 at [#41](https://github.com/winniel123/verge-asm/issues/41), the
note carrying it cites ADR-0015 **zero times** (#102, measured), and the row went on reading as
settled all the way to #104 on a ground that had been dead since the day it was written down.
ADR-0058 exists because a superseded **mechanism** read forward; this is the same failure with a
*reason* in the load-bearing position, and it is worse in one respect — a mechanism's absence is
eventually noticed when someone looks for it, while a reason's absence is never looked for at all.

The asymmetry is the argument: an exclusion that could have been recorded on either ground should be
recorded on the aperture one, because that is the ground that cannot go stale unnoticed.

### "It fits none of the rules we have" inverts what a signal set is

ADR-0004 makes the signal set an **output** — of the cadence test, applied to candidates as they
arrive. ADR-0024 makes each rule's population the extension of its own name. Neither gives the set a
shape that a candidate could fail to match, and nothing in the model reads the set as a whole: rules
are versioned **per rule** precisely so that no rule's admission is a fact about any other.

So a candidate that fits none of the existing rules has told us something about our coverage and
nothing about itself. Treating that as an exclusion is `Finding`'s instinct — a backlog to be kept
tidy — applied to the rule namespace instead of to the rows.

### It does not license admitting everything

The bar is unchanged and it is ADR-0004's, plus the naming discipline
`CONTEXT.md` already carries. A candidate still has to read a fact rather than a conclusion, be named
for the fact rather than for a protocol or a table's contents, and have reference data that changes at
release cadence. What this ADR removes is a bar **nobody ever adopted** — and which #8, #17 and §9.2
nonetheless applied, the first two having already been corrected for it by #35.

### Applied to `smb-signing-not-required`

The fact clears every admissibility test: `SMB2_NEGOTIATE_SIGNING_REQUIRED` is a mandatory field of a
pre-authentication response, its reference data is two bits in a Microsoft specification, and the
integrity fact it carries is genuinely single-protocol in v1's aperture — which ADR-0015 made
legitimate. The name in the ticket is not the name it would ship under: *smb-* names a protocol, which
ADR-0015 forbids, so the rule would be named for the integrity fact.

It is excluded on the **aperture** and on that alone. Reading the field needs an SMB2 `NEGOTIATE`
exchange — application bytes v1 does not send, under an `Offer` v1 has not declared
([ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md)) — and a facet to hold the
result, which is `listener-negotiation`, ruled out of scope for this map by ADR-0015. `445/tcp` being
a `verge-core` pair buys the **connect**, not the exchange.

**One thing does not defer with it, and it is ADR-0015's own finding one level down.** The integrity
fact is neither of `listener-negotiation`'s two proposed fields, so it needs a third — and a facet's
value space is decided **once**, widened afterwards at the cost of a `Break` on every timeline it
holds. While the facet does not exist the third field is free either way; the day it ships with two,
adding the third stops being free. So the **field** question is owed at that facet's specification
time, ahead of and independently of whether any rule reads it. The **rule** remains free to defer
forever. That asymmetry is the general shape ADR-0015 found for `http-identity`'s status class, and it
is why an aperture exclusion may still leave an obligation behind — the exclusion is discharged when
the aperture lands, and the obligation is owed to whoever lands it.

## Consequences

- **The v1 rule set stays at sixteen.** No count in the corpus moves, and
  [#12](https://github.com/winniel123/verge-asm/issues/12) carries sixteen unchanged.
- **[`insecure-listener-rules.md`](../research/insecure-listener-rules.md) §1, §9.2 and §12 q2 are
  amended in place** by [#104](https://github.com/winniel123/verge-asm/issues/104): the *fits neither
  rule* ground is withdrawn, the verdict is restated on the aperture ground, and §12 q2 is answered
  rather than re-parked.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Signal` entry gains one clause** — a fact fitting none of the
  rules we have is a candidate for a new rule, never a reason to drop the fact.
- **An obligation travels with the deferred aperture, and it is not #12's.**
  [`insecure-listener-rules.md`](../research/insecure-listener-rules.md) §8.2 now records that
  whoever specifies `listener-negotiation` owes a decision on a **third field, integrity**, before
  that facet ships — because a value space is decided once. The rule defers freely; the field does
  not defer twice.
- **The map's out-of-scope lines carry their ground.** Existing entries already do this informally;
  new ones state it.
- **This is a *detectable* defect and not a curation trigger**, on ADR-0058's own reasoning: both
  sides are written down — an exclusion's stated ground, and the decision that withdrew it — and it
  fires when *we* move rather than when the world does.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Admit `smb-signing-not-required` as a seventeenth v1 rule, on the strength of the internal-leg gap | The gap is real — `sensitive-port-reached-from-internet` is silent on the internal leg by design (#58) — but per ADR-0004's #44 amendment the rule would render **no row at all**, not even `not-evaluable`, because no subject exists. It buys a name and an empty census, and costs every count in the corpus a re-reconciliation off sixteen |
| Keep the exclusion on *integrity rather than confidentiality, so it fits neither rule* | It is a claim about our two rules, not about SMB. ADR-0015 already treated *third rule* as the shape of the answer rather than an objection to it |
| Rule the fact inadmissible because SMB is one protocol | The principle ADR-0015 withdrew, re-argued. Explicitly out of this ticket's scope |
| Mint the rule now and gate it behind the prober | A rule whose evaluation is conditioned on a facet that does not exist is a rule with an unbuilt leaf in its version vector; the vector could not be written down, so no `Span` could carry it |
| Leave the verdict unstated and let #12 decide | #12 assembles; it does not rule. A closest call resting on a withdrawn ground is exactly what ADR-0058 was minted to stop reaching code |
| State no ground at all, only the verdict | The failure this ADR is about. An unreasoned exclusion cannot be discharged, because nothing tells a later session what would change it |
