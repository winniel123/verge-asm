# ADR-0075: A prototype is a dated record of a reading, never of a rule

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#131 What does a session owe a closed ticket's prototype when a later ruling invalidates a figure in it?](https://github.com/winniel123/verge-asm/issues/131)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

Two agents in one batch met the same defect independently, and both declined to act for the same
stated reason.

- **[#123](https://github.com/winniel123/verge-asm/issues/123)**, on
  [`prototypes/proposal-confirm/`](../../prototypes/proposal-confirm/index.html): the screen offers
  **`Confirm 8,388,608 addresses`**, an act [ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)'s
  `1,024`-address cap now refuses at declaration per
  [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md).
  [#81](https://github.com/winniel123/verge-asm/issues/81) flagged it — *"it collides with one drawn
  artefact … the example wants restating at an admissible size"* — and nobody restated it. #123 left
  it: *"a prototype is a dated record of what that session drew, and rewriting one to match a later
  ruling makes the record lie about when the ruling arrived."*
- **[#118](https://github.com/winniel123/verge-asm/issues/118)**, on
  [#44](https://github.com/winniel123/verge-asm/issues/44)'s
  [`prototypes/signal-evaluability/`](../../prototypes/signal-evaluability/index.html) `?fill=modal`:
  it draws **616 `certificate-expiring`** and **409 `plaintext-http-no-https`** subjects on the modal
  cloud-resident install, on the copy *"only the ports your names imply are probed there — 80 and
  443"*. [ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md) makes the probing gate
  **total** over an `Address`, so on that install both populations are **hollow** and that sentence is
  the clause ADR-0019 withdrew. #118 left it: *"a closed ticket's throwaway is not another ticket's to
  re-fill."*

**The tension, as both stated it:** rewriting makes the record lie about when the ruling arrived;
leaving it makes a wrong example reachable from `main`.

### The third sighting, and it is the one that decides the shape of the rule

**[measured]** [`prototypes/coverage/index.html`](../../prototypes/coverage/index.html) gives
`203.0.113.0/24` a `Coverage` denominator of **`254`**, at six sites in its fixture. #81's answer 1
ruled the denominator is **`256`** — *"network and broadcast addresses are **not** exempted, because
that infers a subnetting we never measure"* — and wrote the correction in its own resolution: *"(the
`coverage` prototype's `254` should read `256`.)"* Nobody restated it, and it is live on `main` today.

Nobody counted this as a sighting, and it is not one, and **that is the finding.** A denominator that
moved by two leaves a drawing that still answers its own question — *what does the `Coverage` screen
look like when a vantage dies?* — with a number that is now a reading of a superseded corpus. That is
a **figure**. `Confirm 8,388,608 addresses` is not: it is a control the product will not offer at any
number. The two sightings the ticket carries and the one it does not fall on opposite sides of a line
no ticket has drawn.

### What the corpus already binds

- **[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s test is
  met by demonstration rather than by judgement**, and #118 says so outright: *"the superseded
  sentence, read alone and in the present tense, **caused a competent session to build the thing**."*
  #44's prototype did not record the withdrawn clause; it **implemented** it.
- **ADR-0058's Scope row expressly excludes figures** — *"Not **figures**, which already have `FIGURE
  DELTA`, and not claims about the world, which have the amendment convention."* So the sightings
  cannot be admitted wholesale as ADR-0058 instances; something has to separate the figure from the
  rule.
- **[#106](https://github.com/winniel123/verge-asm/issues/106) put the unit on the *sentence* and
  the line on *voice, not section*** — the rule reaches a passage asserting, in the **operative
  voice**, that the mechanism holds. Its practical guard: *if the sentence needs "as it then stood"
  added to make it read historically, it was in the operative voice and owes the mark.*
- **[ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) §5: a withdrawal that
  supplies no replacement does not hold.** A sentence naming no successor is re-derived by the next
  session that needs one; the identity ADR-0038 struck regenerated at **19 sites**.
- **ADR-0057 §6: a standing duty has no artefact**, so its absence is indistinguishable from its
  discharge. Any answer here that reads *and then check the prototypes* is that duty.
- **The map's own convention is stated over figures**: *"every **figure** inside a Decisions entry or
  a rule below is a **dated record**, not a current value."* It has never been stated over a rule.
- **The map's batch note records a fourth flavour of this**, among four cross-ticket desyncs the
  ten-agent run cost: *"a brand-new prototype copying a figure a closed ticket had already
  superseded."*

### Nothing marks a prototype as dated, and where the marking would have to go is measurable

**[measured]** All eleven prototypes on `main` open with the same line —
`PROTOTYPE — throwaway. Issue #N: "<question>"` — and in **every one of them it is inside an HTML
comment**, invisible in a browser. Ten put it at line 3, before `<html>`; `seeds/` puts it at line 9,
inside `<head>`. **Not one carries a date**, a corpus state, or anything at all that a reader sees on
the rendered page. The `?fill` and `?variant` fixtures do carry dates — `coverage/` alone renders six
— but they are dates **inside the drawn scenario**, and a reader takes them for the story rather than
for the artefact's own provenance, which is precisely the confusion a dateline resolves.

**Both sightings were made on the rendered page, not in the source.** #118 quotes what the screen
*draws* and what its copy *reads*; #81 quotes the button label. The place the belief forms is the
place with no marker on it.

## Decision

**A prototype is a dated record of a *reading*, and never a dated record of a *rule*.** Three limbs
and one payer.

### Limb 1 — a figure is dated, and a later ruling that moves it reaches the prototype not at all

Every quantity a prototype renders is that session's reading of the corpus on that session's date, on
exactly the footing the map already gives a figure inside a Decisions entry. A ruling that **moves the
quantity** owes the prototype nothing: no rewrite, no mark, no ticket. `254 → 256` is the case, and it
is closed here as **no action**, recorded so a later sweep does not re-find it as a miss.

### Limb 2 — a drawn state the product can no longer reach is a withdrawn rule drawn, and it is owed the mark

The test is on the **drawing**, not on the number:

> **Could the product, as currently ruled, put this surface in front of an operator?** Where a session
> could take the drawn state for a state the product *has* — an **act** it offers, a **population** it
> shows as non-empty, a **sentence** it puts in the product's mouth — and the product now refuses that
> act, hollows that population, or has withdrawn that sentence, the surface is a **rule drawn after
> its withdrawal**, and it is owed the mark at the prototype.

This is ADR-0058's own test with the word *sentence* read as *drawing*, and #106's voice test carried
across unchanged: markup rendering a control, a populated census or a line of product copy is the
**operative voice**. A fixture number is not.

Applied to the three sightings:

| Site | What the ruling did | Verdict |
| --- | --- | --- |
| `proposal-confirm` `Confirm 8,388,608 addresses` | ADR-0049 / ADR-0047 **refuse the act** at declaration, at any size over `1,024` | **Owed the mark** — a control, not a count |
| #44's `?fill=modal` **616** / **409** | ADR-0019 **hollows both populations** on that install; there is no admissible smaller number | **Owed the mark** |
| #44's `?fill=modal` *"only the ports your names imply are probed there — 80 and 443"* | ADR-0019 **withdrew the sentence** | **Owed the mark**, and this half is an ADR-0058 site with or without this ADR |
| `coverage`'s `254` | #81 **moved the denominator** to `256` | **Not owed.** Limb 1 |

### Limb 3 — the mark is a replacement, it is on the rendered surface, and the drawing is left standing

- **Left standing and marked**, per this repo's name-and-withdraw convention as ADR-0058 restates it.
  The drawing is **not** redrawn. The evidential value of the wrong screen is the whole reason #118's
  argument could be made at all.
- **A replacement, not a strike** (ADR-0057 §5). The mark names the ruling *and states what the
  surface would draw now* — *"under ADR-0019 both populations are empty on this install"*, not merely
  *"superseded"*. A struck drawing with no successor is re-derived by the next session that needs one.
- **On the rendered surface**, not only in the source comment. A header comment is the **pointer**
  ADR-0058's Rationale already rejected: it reaches the reader who opened the source, who is the
  reader who did not need it.
- **Form**: the dashed accent annotation box `prototypes/seeds/` already ships for reviewer
  commentary (`.anno`, `1px dashed var(--accent)` on `var(--accent-soft)`). No new token, no
  design-system addition, and it is already declared *"not part of the design"*.
- **Granularity is the drawing, never the file.** A prototype is many drawings behind one URL — `seeds/`
  is 22 combinations, `signal-evaluability/` is two screens by four fills. The mark attaches to the
  variant or fill it condemns and leaves the siblings alone.

### Who pays, and why it is bounded

**The pass that supersedes — and only where it already holds the prototype**: it opened it, cited it,
or its own ticket names it. **No pass ever acquires an obligation to go and find prototypes.**

This is #106's *grep the document you are writing in* at one further hop: **mark the artefact you
already opened.** Both sightings arose exactly that way — #123 and #118 each had the file in hand and
each stopped one edit short, which is #106's diagnosis of its own population verbatim.

### The residue is handled by dating, not by sweeping

A prototype nobody held is left to its reader, and the reader is given the one thing they lack today.
**Every prototype carries a dateline on its rendered surface** — the ticket, the date, and one clause
pointing at the map's `THE CURRENT COMPOSED STATE` line as the only live absolutes.

This is not the failed pointer. A pointer is conditioned on the reader **noticing an absence**, which
is the reader who has already solved the problem. A dateline is **unconditional and at the site**: it
does not ask the reader to detect anything, it tells them what kind of object they are looking at
before they read a number off it. It is the same move the map made on itself when it wrote *"every
figure inside a Decisions entry is a dated record"* at the top rather than beside each figure.

## Rationale

### The dated-record convention was minted over figures, and it does not stretch to rules

The map's sentence is *"every **figure** inside a Decisions entry or a rule below is a **dated
record**, not a current value."* Both declining agents reached for it and both generalised it from
*figure* to *artefact* in one step, without noticing the step. It is right about figures for the
reason it was written: a figure is a **reading**, its correctness is indexed to a date, and a reader
who takes a dated reading for a current value has misread the object rather than been misled by it.

A rule is not indexed to a date in that way. A control the product offers, a population it shows
populated, a sentence it says — these are the product's commitments, and a drawing of one is a
**claim that the product does this**, in the present tense, in the strongest form the corpus contains.
There is no reading of *"dated record"* on which a drawing of an act the product refuses is merely a
stale reading.

### A drawing is the strongest form of ADR-0058's failure mode, not a weaker one

ADR-0058 diagnoses a reader who *"**believes** the mechanism exists and never looks"*, believing it
because the sentence says so in the present tense. A prototype is that sentence with the evidence
attached. #118's session did not have to infer the withdrawn sentence's consequences — it read them
off a screen that had already drawn them, at 616 and 409, and then had to argue itself back out.

ADR-0058's *Restrict it to ADRs* alternative lost because the load-bearing instance ran the wrong
direction, and #106's extension won because the failure mode is indifferent to file boundaries.
Neither reason consults the file's **format**. A rule stopping at prose catches neither sighting.

### The cut is what keeps it bounded, and the `254` case is what proves the cut does work

A rule reaching every wrong quantity in a prototype is the unbounded sweep the ticket bars, because
every prototype is nothing *but* quantities — fixture counts, timestamps, address lists, census
totals, all of them readings of a corpus that moves weekly. Limb 1 discharges all of it in one
sentence and leaves a population that is small by construction: a ruling that makes a **drawn state
unreachable** is rare, it is exactly the kind of ruling that gets an ADR, and the pass that makes it
already knows it has made it.

`254` is the discriminating instance. Under *rewrite everything wrong*, it is owed an edit; under
*rewrite nothing*, so are 616 and 409 owed nothing. Under this rule it is owed nothing and they are
owed a mark, and the reason is stateable in one clause: **the `Coverage` screen still works.**

### The obligation lands where ADR-0058 already put it, and nowhere new

*"The pass that supersedes. It is the last party who knows both states."* The narrowing — *only where
it already holds the prototype* — costs the rule the prototypes nobody opened, and buys the guarantee
the ticket demands. It is also not much of a cost: the reason a pass holds a prototype is that the
prototype is about the thing it is ruling on.

## Consequences

- **[#132](https://github.com/winniel123/verge-asm/issues/132) is executable, and it does not
  re-fill.** Under limb 2 there is no admissible number for 616 and 409 — ADR-0019 hollows the
  populations rather than shrinking them — so #132 leaves the drawing standing and writes **one
  `.anno` block on `?fill=modal`** naming ADR-0019, stating that both rules have empty populations on
  the modal cloud-resident install, and stating that the *"80 and 443"* copy is withdrawn. Its title's
  *re-fill* is the losing action; the mark is the owed one. Small, mechanical, and it closes.
- **[`prototypes/proposal-confirm/`](../../prototypes/proposal-confirm/index.html) `?fill=cloud` is
  owed the same mark** for `Confirm 8,388,608 addresses`, against ADR-0049 and ADR-0047. Not written
  here — #131 rules the convention and does not apply it.
- **[`prototypes/coverage/`](../../prototypes/coverage/index.html)'s `254` is owed nothing**, and is
  recorded here so a later pass does not re-find it as a miss. #81's parenthetical is superseded by
  this ADR: the correction was optional and remains optional.
- **The dateline is owed on all eleven existing prototypes**, and on every prototype written after
  this. Ticketed rather than written — three prototypes were in flight in other worktrees when this
  ruled.
- **Detectable, not watched.** No ninth curation trigger. This fires when **we** move, which is
  [#102](https://github.com/winniel123/verge-asm/issues/102)'s and #106's ruling on shape, and it
  joins the map's *detectable defects* group as the **fifth** member.
- **[`docs/agents/design-system.md`](../agents/design-system.md) gains the rider** — it is the
  document a session reads before writing prototype markup, and it is the prototype-shaped analogue of
  [`docs/agents/domain.md`](../agents/domain.md)'s ADR-0058 rider.
- **[`CONTEXT.md`](../../CONTEXT.md) needs no change.** No term is added and none is amended. A
  prototype is a repo artefact, not a domain object. Precedent: ADR-0058 and #106 both closed the same
  way.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) gains nothing and loses nothing.** No
  prototype is a spec input; this reaches the artefacts, not the specification.

## Alternatives rejected

**A prototype is a dated record, full stop — nothing is ever owed. THIS IS THE OPTION THAT LOST**, and
it is the one the ticket floats and the one both #123 and #118 acted on, in good faith and
independently. It loses on #118's own measurement, which is a demonstration rather than a judgement:
the prototype **implemented** a withdrawn sentence, and the session that met it read `616` off the
screen before it read ADR-0019. Under this option the artefact surviving on `main` is a drawing of an
install the product cannot produce, behind a URL a later session opens first. Its supporting
authority — the map's dated-record convention — is stated over **figures**, and this option needs it
over **rules**, a generalisation neither agent noticed making. What survives of it is limb 1, which is
the half it was right about.

**Rewrite the prototype to match the ruling.** Rejected on three grounds. It makes the record lie
about when the ruling arrived, which is both prior agents' stated reason and correct. It is unbounded
in exactly the way this ticket bars. And it **destroys evidence** — #118's argument is only makeable
because the wrong screen still exists to be pointed at, and #81's collision finding likewise.

**Make it a curation trigger, or a standing duty on the reader to check prototypes against the
corpus.** Rejected on ADR-0057 §6's ground before its cost: a standing duty has no artefact, so its
absence is indistinguishable from its discharge. The map's triggers are watches on the **world**
moving; this fires when **we** move, which is the wrong list on shape.

**Mark the whole file rather than the drawing.** Rejected on measurement: `signal-evaluability/` is
eight drawings and one is wrong; `seeds/` is 22. Condemning a file buys one correction and loses
twenty-one good answers, and it re-introduces the *"is this whole thing stale?"* question the dateline
exists to answer.

**Mark it in the header comment only.** The cheapest option, and the one the corpus already almost
implements — all eleven prototypes carry their provenance there. Rejected on where the reading
happens: both sightings were made on the rendered page, and an HTML comment reaches only a reader who
opened the source. That is ADR-0058's rejected pointer with a new address.

**Delete the stale prototype.** Rejected on the ground the name-and-withdraw convention was adopted:
a deletion takes the reasoning with it, breaks inbound citations from a closed ticket's resolution,
and leaves a later reader unable to tell a withdrawal from an omission.

## Thin ground

**The population is three sightings and its *rate* is unmeasured.** Two are the ticket's own; the
third I found in the course of reading #81 rather than by sweeping, and **I did not sweep the other
eight prototypes — deliberately, because a sweep is the obligation this ADR declines.** So this rests
on the reach of ADR-0058's own words and on three instances, not on a measured failure rate. That is
the same concession #106's amendment had to make for its own extension, and it is ruled anyway on the
same ground: the correction is one annotation per condemned drawing, paid by a pass already holding
the file, against surfaces that were readable forward — one of which drew an act the product refuses
and one of which drew a populated census that must be empty.

**The cut between limb 1 and limb 2 is a line this ADR draws rather than finds.** No prior ticket
separated *a number that moved* from *a state that became unreachable*, and `254` is the only measured
instance on the near side of it. A ruling that both shrinks a population and hollows it in different
fills would sit on the line; none exists today, and the tie-break is stated in advance: **if the
drawing still answers the question its ticket asked, it is a figure.**

**One case is left open rather than decided.** A prototype whose *load-bearing question* is invalidated
outright — not a fill but the thing the file was built to answer — is not covered here, because it has
never happened. The mark would be inadequate and deletion is barred; the likely answer is that the
prototype is superseded by a new one and cross-linked, which is what #106 ruled for documents. It is
not ruled here, because ruling it would be inventing the instance.
