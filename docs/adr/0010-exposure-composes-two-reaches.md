# ADR-0010: Exposure composes two Reaches, and a rule reads a leg rather than a state

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#32 Does sensitive-port-exposed fire on edge-only as well as exposed?](https://github.com/winniel123/verge-asm/issues/32)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#14](https://github.com/winniel123/verge-asm/issues/14) produced a derived per-`Service`
reachability state with five values — `exposed`, `firewalled`, `internal-only`, `edge-only`,
`unreachable` — and [ADR-0004](./0004-signals-are-release-coupled-rules.md) settled that
`sensitive-port-exposed` reads the derived `Exposure` rather than re-deriving reachability
from vantage-stamped observations, because a second implementation of the flagship derivation
would surface its divergence from the first as false drift. Neither said **which values
satisfy the rule**, and [#21](https://github.com/winniel123/verge-asm/issues/21) flagged the
gap rather than closing it.

Read #14's table literally and the states are not a partition:

- `exposed` — "reachable from an internet vantage"
- `edge-only` — "reachable from the internet but not internally"

`edge-only` is a **subset** of `exposed`'s written definition. They overlap, and an
implementation that assigns `edge-only` preferentially while a rule tests `state == exposed`
produces no error, no `not-evaluable` and no evidence that anything was skipped. It simply
never fires on the more alarming case — a sensitive port published to the internet that the
operator's own network does not even route to.

The overlap is a symptom. The five states **flatten two orthogonal legs**: what an
internet-class vantage found, and what an internal-class one found. Five names were laid over
that grid, so some names span several cells and some cells have no name at all. A rule that
enumerates state labels is therefore a list somebody maintains, and the failure mode when the
list falls behind is silence rather than an error — which is the same shape as the
hand-maintained union that [ADR-0009](./0009-verge-core-is-a-union.md) replaced with a
definition.

## Decision

**`Exposure` is defined as the composition of two `Reach`es, one per vantage class, and a
rule reads a leg rather than a state.**

**1. `Reach` is a named `Derivation` leaf.** One per `(Service, vantage class)`, recording
what vantages of that class found. **Two values only — `reached` and `not-reached`.** There is
no third value for *we did not look*, because a `Batch` whose recorded scope excludes a port
does not touch that timeline at all, so the absence is already a `Gap`
([ADR-0007](./0007-drift-is-a-timeline-of-spans.md)). Adding a `not-checked` value would
re-invent `Gap` — the move [ADR-0006](./0006-subjects-leave-by-measurement.md) warns against,
where a thing that already has a representation is given a second one and a transition
between them has to be invented too.

> **CONDITIONAL, added 2026-08-15 by [#173](https://github.com/winniel123/verge-asm/issues/173) ·
> [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md).
> Nothing here is struck — every sentence above is true of v1 — but read alone in the present tense
> this paragraph specifies that an absent `Reach` is always explained by the recorded scope, and a
> competent session would build that identity into `Coverage`.** It holds **while every probed
> transport's outcome union projects totally onto `Reach`**. TCP's does:
> `connected │ refused │ no-response` all project.
> [ADR-0083](./0083-silence-decides-only-on-a-connection-oriented-transport.md)'s connectionless
> union does not — `unanswered` projects onto **neither** value — so a **UDP** pair *inside* the
> recorded scope, probed at cadence and writing an observation every run, can hold no `Reach` value
> at all. **v1 probes TCP alone, so the identity holds today and would stop holding the day a UDP
> tier opened.** Two values only, and no third: that half is unaffected and ADR-0083 refuses a third
> value again on its own account.

**2. The five states are a projection, not the thing.** `Exposure` remains the operator-facing
enumeration and the board's axis; it is *derived from* the two legs rather than being what the
model stores as primary:

| internal ↓ · internet → | `reached` | `not-reached` | *(`Gap`)* |
| --- | --- | --- | --- |
| **`reached`** | `exposed` | `firewalled` | `internal-only` |
| **`not-reached`** | `edge-only` | `unreachable` | *unnamed* |
| ***(`Gap`)*** | *unnamed* | *unnamed* | no `Exposure` |

**3. `exposed` is tightened to mean both legs `reached`**, which is what makes the five states
mutually exclusive. #14's wording named the whole internet-`reached` column and that is the
defect this ADR opened on.

**4. `sensitive-port-exposed` is renamed `sensitive-port-reached-from-internet`**, and its
definition names the leg it reads: it fires where the internet `Reach` is `reached`, does not
fire where it is `not-reached`, and is `not-evaluable` where it is a `Gap`. No state label
appears in the rule. The name encodes the predicate, because a rule whose name does not say
what it reads is how `state == exposed` became plausible in the first place.

> **Read *`not-evaluable` where the leg holds no value*, of which *`Gap`* is one case —
> [#173](https://github.com/winniel123/verge-asm/issues/173) ·
> [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md).**
> A `Gap` is a **span**, so a leg that never opened has none, and §4 read literally would leave the
> rule with no answer to return there. It returns `not-evaluable` in both cases and the warrant is
> the same one #44 gave: `not-evaluable` **needs a subject**, and a `Service` exists for every pair
> in the recorded scope, open or closed. What #44 refuses is a `not-evaluable` with no subject —
> the aperture half — and that case is untouched.

**5. `Exposure` exists where at least one leg holds a value, and the two ways a leg can be
absent are different.** Where no internet-class vantage was ever in play, the internet `Reach`
has no timeline and `internal-only` is an honest value — *we never looked*. Where the timeline
existed and went silent, `Exposure` opens a `Gap` — *we stopped looking*.

> **A THIRD way, conditional on transport — [#173](https://github.com/winniel123/verge-asm/issues/173)
> · [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md),
> on [ADR-0083](./0083-silence-decides-only-on-a-connection-oriented-transport.md).** *"The two ways"*
> is a closed enumeration read alone, and on a connectionless exchange there are three: **we looked
> and the exchange did not decide.** It is not *we never looked* — the pair is in the recorded scope
> and an observation is written every cadence — and it is not *we stopped looking*. It splits by
> history: where the leg had **already opened** it is a `Gap`, and where it never opened it is
> **nothing at all**, per `CONTEXT.md`'s `Reach` entry. The second is the dangerous half, having no
> span, no recorded cause, no closing edge and no message; ADR-0095 routes it to the aperture
> statement because nothing else can carry it. **`internal-only` is separately withdrawn** by
> [ADR-0017](./0017-exposure-needs-both-legs.md) and this note revives nothing.

## Consequences

- **`firewalled` versus `internal-only` was a `Gap` all along.** #14's most careful
  distinction — evidence of absence versus absence of evidence — is exactly *value* versus
  *`Gap`* on the internet leg. It was never a pair of states; it was one state and a hole,
  named twice, two tickets before `Gap` existed. Nothing about #14's intent is lost; the
  mechanism it was reaching for now has a name and applies uniformly.
- **The flagship transition is a column move, not a cell move.** What the product exists to
  catch is the internet `Reach` going `not-reached` → `reached`, which spans **both**
  `firewalled` → `exposed` *and* `firewalled` → `edge-only`. The second is the same escalation
  and is currently not the hero cell on [#10](https://github.com/winniel123/verge-asm/issues/10)'s
  board. Any rendering that promotes one and not the other buries half of its own headline.
- **Four cells have no name, and one of them is dangerous.** Internal `not-reached` with the
  internet leg in a `Gap` currently reads `unreachable` — "no vantage reaches it" — while we
  never looked from outside. That is #14's own false-reassurance failure occurring in the one
  row where #14 did not guard against it. Naming the cells is board vocabulary and is carried
  as its own ticket; this ADR fixes only the composition they are cells of.
- **A rule composes the leaf, not the whole value.** Under
  [ADR-0008](./0008-derivation-versions-move-on-content.md) the version is a vector with one
  leaf per named `Derivation`, so `sensitive-port-reached-from-internet` composes the internet
  `Reach` leaf alone. Re-labelling which cell counts as `exposed` versus `edge-only` is then a
  presentation change that does **not** break the signal estate-wide. Under the enumerating
  design it would have.
- **This is the only v1 signal that reads `Exposure`.** Every other rule in
  [ADR-0004](./0004-signals-are-release-coupled-rules.md)'s set fires on an `Endpoint` and its
  evidence — a `certificate` or `http-identity` observation — presupposes that something
  reached it, so no leg gate is needed. The temptation is to gate *plaintext HTTP with no
  HTTPS* on internet reach because it feels less urgent internally, but ADR-0004 settled that
  signals carry no severity, and gating would smuggle severity back in disguised as
  evaluability.
- **`fired` → `not-evaluable` notifies in the coverage class**, worded as *we stopped looking*
  and never as a clear. Routed to drift it would read as the signal **clearing** — false
  relief on the sharpest signal in the product, caused by our own outage. This is the fifth
  precedent for [#8](https://github.com/winniel123/verge-asm/issues/8)'s finding that the three
  notification classes partition **messages** rather than events. The way back out — a value
  appearing where a `Gap` was — is [#42](https://github.com/winniel123/verge-asm/issues/42)
  and is deliberately not answered here.

  > **The edge and the class are unchanged — coverage class, member 5
  > ([ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md)) — and only the WORDING takes a
  > conditional**, [#173](https://github.com/winniel123/verge-asm/issues/173) ·
  > [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md).
  > On a **UDP** leg *we stopped looking* is **false**: we looked, at cadence, and the exchange
  > decided nothing. **No member is minted and the coverage class stands at ten.** *"Never as a
  > clear"* is untouched and binds harder here than anywhere, the whole hazard being a clean bill on
  > the pairs the sensitive list exists for. The copy belongs to
  > [#120](https://github.com/winniel123/verge-asm/issues/120) / ADR-0064's grammar and is not
  > written here; v1 probes TCP alone, so this wording is correct on every shipped configuration.
- **[#28](https://github.com/winniel123/verge-asm/issues/28)'s rider is narrowed.** Its "a
  subject whose vantage is unavailable holds no `Exposure` at all and cannot be given a
  column" is true of the *went-silent* case and false of the *never-configured* one, where
  #14 deliberately built the no-prober deployment to be "a complete, honest internal
  reachability inventory". Both survive because `Gap` already does the work: a silent timeline
  produces no span, so there is no adjacency and therefore **no transition to `internal-only`
  to render** — which is what stops a dying prober from de-escalating every `exposed` service
  in the estate overnight.
- **The rename is recorded, not retrofitted.** `sensitive-port-exposed` appears in three
  closed resolutions ([#16](https://github.com/winniel123/verge-asm/issues/16),
  [#21](https://github.com/winniel123/verge-asm/issues/21),
  [#29](https://github.com/winniel123/verge-asm/issues/29)) and in
  [ADR-0004](./0004-signals-are-release-coupled-rules.md). Those stand as written; the map
  carries the rename. Rewriting closed tickets is not this map's habit.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Enumerate the satisfying states** — the rule tests `state ∈ {exposed, edge-only}` | Answers today's question and leaves the mechanism intact. A sixth state added later means every rule silently under-fires, which is the bug this ticket opened on, one iteration further along. A list somebody maintains, where a definition was available |
| **Give `Reach` a third value, `not-checked`** | Re-invents `Gap`. ADR-0007 already has a word for a period we could not say, and inventing a state means inventing a transition into and out of it — a threshold of our own sitting inside the comparison path |
| **Let `exposed` name the whole internet-`reached` column, demoting `edge-only` to a sub-distinction** | Internally consistent, and it would have made the original rule correct. But #10's board is a matrix over the five states as peers, and collapsing two of them into a parent breaks the axis the landing view is built on |
| **Keep the name `sensitive-port-exposed`** | Puts `exposed` on two meanings in one vocabulary — the precise defect that got `Host` rejected from `CONTEXT.md`. Nothing has shipped, so the rename costs only legibility, and the ambiguity is what this ADR exists to remove |
| **Fire louder on `edge-only`** | It *is* the more alarming shape, but ADR-0004 settled that signals carry no severity. The differential belongs to the transition that surfaced it, which is where the model already puts urgency |
| **Gate the cert, TLS and HTTP signals on internet reach too** | Their evidence already presupposes a reach, so the gate would change nothing except to make internally-observed defects `not-evaluable` — reporting less than we measured, in order to express a severity the model refuses to carry |
