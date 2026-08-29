# ADR-0052: A declaration refusal names a route and never takes it

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#123 The `Seeds` screen: six jobs beyond listing, and three refusals with three different right answers](https://github.com/winniel123/verge-asm/issues/123)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Discharges:** [ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md), [ADR-0055](./0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md), [ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md)
- **Constrains:** [ADR-0002](./0002-ownership-gates-probing.md), [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md)

## Context

The `Seed` box is the one operator-typed input sitting in front of the probing gate.
[ADR-0002](./0002-ownership-gates-probing.md) as amended reads `Seed`s alone, so every character
typed there is a boundary claim and nothing else in the product can correct one.

Three closed decisions each ended by handing this screen a debt, in almost the same words, and none
of them said what a refusal is *allowed to do*:

- [ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md) — an
  over-cap **IPv6** declaration *"must name the `custody extension` as the route"*, and *"naming the
  knob is correct for IPv4 and a trap for IPv6"*. It calls this *"a surface obligation carried by an
  unfinished screen, which is a real dependency and is stated as one."*
- [ADR-0055](./0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md) —
  a typed **U-label** is refused, and the remedy is *"refusal copy naming the A-label as the route,
  in the shape #85 established"*.
- [ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md) — a typed
  **`*.example.com`** is refused, naming the **subtree exclusion** as the route, with the rider that
  *"the two are not the same object and must not be silently translated"*.

Each ADR reached for the one before it as a template — *in the shape #85 established* — which is
three sessions asserting that one shape exists without anybody writing it down. Drawing the screen
([`prototypes/seeds/`](../../prototypes/seeds/index.html)) forced the question the template was
hiding, because the **fourth** refusal on the same box is the over-cap **IPv4** declaration, and it
is correct there to offer the operator the knob. Two refusals of the same rule, on the same control,
with opposite right answers.

## Decision

**A declaration refusal names a route and never takes it. A route may move this project's
configuration; it may never move the operator's claim.**

The test is one question, and it is not about families, formats or tidiness:

> **Does the route reach the same set as the thing typed?**
> If it does, the refusal may offer to take it. If it does not, the refusal names it and stops —
> nothing is pre-filled, nothing is converted, and the box keeps what the operator wrote.

| Concern | Decision |
| --- | --- |
| Over-cap **IPv4** address scope | **Offer the route.** Raising the cap is a change to *our* configuration, after which the declaration is exactly the range typed. Nothing is substituted, so an affordance is legitimate — a link to the setting, and the arithmetic for splitting |
| Over-cap **IPv6** address scope | **Name the route, offer nothing.** The route is a name scope with a `custody extension`, which covers *what resolves* rather than *the prefix* — a different set. No pre-filled domain, no link to the cap setting, no button |
| The knob, in the IPv6 refusal | **Named in order to shut it**, not omitted. See the rationale: the setting is reachable from Settings and from the IPv4 refusal one screen away, so silence is not silence — it is leaving the trap unlabelled |
| A typed **U-label** | **Name the route, and do not compute the A-label.** Printing *did you mean `xn--…`* would run the UTS-46 conversion ADR-0055 bars a key normalisation from making, and then render its output as advice. The route named is **where to obtain** the A-label, never the A-label |
| A typed **`*.example.com`** | **Name the subtree exclusion, and state the difference in the refusal itself.** A subtree exclusion runs on label-wise suffix equality and therefore **includes** `example.com`; the wildcard expressly excludes it. The operator makes the substitution or does not |
| What a refusal may never do | Pre-fill a corrected value, offer an *accept the fix* control, or write anything. **A value this project refuses to hold may not be rendered as a suggestion** |
| What a refusal must always do | Say plainly that nothing was declared, and leave the typed text in the box |
| Where the arithmetic lives | Inline and live, as the operator types. An address count over a CIDR is arithmetic over a declaration, never a measurement of an estate ([ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md), #50), so it is not [#28](https://github.com/winniel123/verge-asm/issues/28)'s refused figure |
| Confirming a `Proposal` | **Is a declaration**, so the cap is checked there too. An over-cap `Proposal` row carries no live confirm control, and the refusal renders in the row rather than after the click |

## Rationale

### The cut is the set, and every other cut fails on the fourth case

Three cuts were available and two of them survive only until the IPv4 refusal is put beside the
IPv6 one.

**By family** — *IPv4 refusals offer the knob, IPv6 refusals do not.* It reproduces the right answer
for these two and is the reading ADR-0049 spent its whole ruling refusing:
*"a family check would be a claim about IPv6; the cap is a claim about us."* Re-introducing the
family branch in the copy throws away the only thing that made the cap admissible, and it says
nothing at all about the U-label or the wildcard.

**By magnitude** — *offer the knob below some size, refuse above it.* This is
[#27](https://github.com/winniel123/verge-asm/issues/27)'s invented threshold inside the safety path,
arriving in the one place this model has kept clear, and ADR-0049 already refused a hard ceiling on
exactly that ground.

**By whether the route reaches the same set** — it produces all four answers with no branch, and it
explains *why* each is right rather than recording that it is. Raising the cap declares
`198.51.100.0/16`, the set typed. A `custody extension` covers the addresses names resolve to, which
is not the prefix. `xn--bcher-kva` is one of two readings of `bücher`, so choosing it is choosing a
name. A subtree exclusion on `example.com` includes `example.com`, which `*.example.com` excludes.
One test, four answers, and the two-word statement of it is that **a route may move us and may not
move them**.

### Why the knob is named in order to shut it

ADR-0049 says *must not offer the knob*, and the cautious reading of that is silence: do not mention
a setting you do not want used.

**Silence loses, and it loses on a fact about the product rather than on tone.** The cap is
operator-configurable and lives in Settings. The IPv4 refusal on the same control points straight at
it. An operator who has just been told, correctly, that raising the cap is what one does about an
over-cap range will go and do it. Saying nothing does not remove the knob — it removes the only
sentence that would have stopped it being used, and leaves an operator who follows the obvious path
with a permanently uncompletable scope and a `Coverage` figure pinned near zero.

So the IPv6 refusal states *do not raise the cap for this* and gives the reason in the operator's own
units. What is refused is every **affordance**: no link to Settings on that panel, no pre-filled
domain, no *declare `example.com` instead* button. Naming a route is not offering it, and the
distinction is exactly the one this ADR is about.

This is the least-supported paragraph in the ruling and is marked as such: ADR-0049's words admit
either reading, and nobody has watched an operator meet either version.

### Why the A-label is not computed

The tempting refusal prints the conversion as help: *`bücher.example` is not held; did you mean
`xn--bcher-kva.example`?*

It is barred, and by ADR-0055's own prohibition rather than by a new one. Computing that string means
running UTS-46 — Unicode mapping, normalisation, Bidi and ContextJ data, which move on somebody
else's schedule and disagree across three live profiles today. ADR-0055 refuses that inside the key
function. Running it in a text box and rendering the output as advice is the same computation with
the enforcement removed: the operator copies our suggestion back into the box, and a value this
project would not derive enters the model wearing a declaration.

**The general form is worth stating, because it is not about names:** *a value this project refuses
to hold may not be rendered as a suggestion.* So the route is a place — the operator's own DNS
provider, which shows the ASCII form beside the name — and never an answer.

### Why the wildcard is not translated, and the set v1 cannot express

`*.example.com` and the subtree `example.com` are one label apart on screen and a different set in
the model, and the difference falls on the **apex**. Silently reading the first as the second would
put `example.com` outside the estate on the strength of a pattern that expressly excludes it — a
boundary claim the operator did not make, made by a text transformation, in front of the probing
gate.

Drawing the refusal surfaced something the three earlier ADRs did not have to face. If the operator
means *everything beneath `example.com` but not `example.com` itself*, **v1 has no object for it**:
`Seed` exclusions are exact names, subtrees or address scopes, a subtree takes the apex with it, and
the remaining set is infinite. The refusal says so, in the operator's terms, and routes them to
excluding the names they know individually. It is a real gap and it is stated rather than papered
over.

### The accepted case earns copy too

A refusal is only legible against an acceptance, so the accepted declaration renders what pressing
the button does, and it is one sentence the model already owns: **one coverage-class message at the
scope**, `revealed`, carrying a count of timelines opened — never one message per address, and never
`appeared` ([ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md),
[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)). Beside it, the
invariant the declaration-time cap exists to protect: **every address a `Seed` covers is measured**,
because the cap is checked here rather than truncating a target list later, so there is no such thing
as a partial scope.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **No general rule — leave each refusal to its own ADR** | Three ADRs already wrote *in the shape #85 established* without anybody establishing a shape, and the fourth case (over-cap IPv4) contradicts the template they were copying. A fourth session would have copied it again |
| **Cut on family: IPv4 offers the knob, IPv6 does not** | Reproduces the right answer for two cases and re-introduces the family branch ADR-0049 refused, on the ground that it makes a claim about IPv6 where the cap makes a claim about us. Silent on the other two refusals |
| **Cut on magnitude: offer the knob below a threshold** | [#27](https://github.com/winniel123/verge-asm/issues/27)'s invented threshold inside the safety path. ADR-0049 refused a hard ceiling on precisely this ground |
| **Say nothing about the cap setting in the IPv6 refusal** | The literal reading of *must not offer the knob*, and it removes the sentence rather than the setting. The knob is in Settings and is named by the IPv4 refusal one control away, so silence leaves the trap unlabelled for the operator most likely to walk into it |
| **Pre-fill the corrected value and let the operator confirm it** | This is the legal *future* shape ADR-0055 named for a U-label — the box offers, the operator declares — and it is a `Proposal`-shaped act needing a `Proposal`-shaped object. Not v1, and named so nobody builds the version without the confirmation |
| **Offer *did you mean* for the U-label alone, since the conversion is well-defined** | It is not: three live profiles disagree on ß, ς, ZWJ and ZWNJ, and computing it runs the reference-data lookup ADR-0055 bars. A value refused by the key function may not be rendered as advice |
| **Translate `*.example.com` to the subtree silently, since it is what the operator meant** | It is not what they typed, and the two sets differ at the apex — the one name the wildcard is defined to exclude. A text transformation would make a boundary claim in front of the probing gate |
| **Let an over-cap `Proposal` be confirmed and refuse afterwards** | Confirming *is* the declaration, and ADR-0047 placed the cap at the declaration precisely so that no scope is ever half-accepted. Refusing after the click would put a partial act between two Declared states |
| **Put the refusals behind a disclosure, as [#28](https://github.com/winniel123/verge-asm/issues/28)'s `▸ why` idiom does for errors** | [#51](https://github.com/winniel123/verge-asm/issues/51)'s rule: a hazard that fails silently may never sit behind a disclosure. A refusal is the whole message, not its detail |

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Seed` entry** gains two clauses: that a subtree exclusion
  covers the subtree's own name as well as everything beneath it and that v1 has no object for the
  names beneath a name without the name itself, and that a refused declaration names a route and
  never takes it.
- **[ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md),
  [ADR-0055](./0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md)
  and [ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md)** each
  carry an *owes refusal copy* clause in the present tense. All three are marked discharged in place,
  per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) — a
  session reading any of them alone would otherwise write the copy a second time.
- **No new term, no term changed meaning, and no figure moves.** The cap stays at 1,024 addresses per
  scope. Nothing here is a threshold.
- **No `Derivation` leaf is touched.** A refusal is a surface, it writes nothing, and its content is
  the arithmetic of a prefix plus two rules already in the model — so no version moves and nothing
  `Break`s.
- **One drawn artefact is stale.** [`prototypes/proposal-confirm/`](../../prototypes/proposal-confirm/index.html)
  offers `Confirm 8,388,608 addresses`, which the shipped default now refuses at the declaration. The
  control's shape and its *nothing branches on the number* rule both survive. The example wants
  restating at an admissible size. Left in place as that session's dated record.
- **The cost, stated:** an operator with an internationalised estate, and an operator who means
  *everything beneath a name but not the name*, both do more work than a helpful tool would ask of
  them. That is the price of the probing gate reading only what the operator actually claimed.
