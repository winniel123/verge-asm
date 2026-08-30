# ADR-0127: The address-scope range cap has no ceiling — a large scope is priced at policy time, not gated

- **Status:** Accepted
- **Date:** 2026-08-30
- **Map:** [#845 Map: raise the fixed 1024 address-scope cap (spec)](https://github.com/winniel123/verge-asm/issues/845)
- **Amends:** [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md), [ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)
- **Decisions:** [#846](https://github.com/winniel123/verge-asm/issues/846), [#847](https://github.com/winniel123/verge-asm/issues/847), [#848](https://github.com/winniel123/verge-asm/issues/848), [#849](https://github.com/winniel123/verge-asm/issues/849), [#850](https://github.com/winniel123/verge-asm/issues/850), [#851](https://github.com/winniel123/verge-asm/issues/851), [#882](https://github.com/winniel123/verge-asm/issues/882), [#883](https://github.com/winniel123/verge-asm/issues/883), [#884](https://github.com/winniel123/verge-asm/issues/884), [#885](https://github.com/winniel123/verge-asm/issues/885)

## Context

[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) ruled that an address-scope `Seed`
**enumerates** — every address inside a declared CIDR is a subject from the declaration, walked every
cadence — and that a `1,024`-address per-scope range size cap is what makes that affordable.
[ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md) restated the
cap in its true unit — addresses, not prefix lengths — and made a CIDR family-agnostic.

The cap is a compiled constant. `seed.DefaultAddressCap = 1024` (`internal/seed/seed.go`) is flowed
once into `seedAddressCap` (`cmd/web/handlers.go`); the Settings **#206** control that would let an
operator raise it is planned and **not wired**. So today the operator who genuinely holds more than
1,024 addresses in one range has **no route to sweep that range blind** — the knob that ADR-0047 and
ADR-0049 both describe as raisable does not exist yet, and both ADRs leave the edges of a raised cap
open rather than closed.

This ADR is the spec that closes those edges. It rests on a decision the two load-bearing ADRs
**already anticipated** — ADR-0047 states in terms that *"a `/8` declared with the knob raised is
accepted"*, and ADR-0049 states *"No ceiling is invented … ADR-0047 already accepted that a
knob-raised declaration may be uncompletable and left the consequence with the operator."* The cap
mechanism is not touched. What is settled here is everything a raised cap forces: whether there is an
upper bound on the knob, how the raise and the large declaration are confirmed, what the surfaces say
about a scope that cannot finish in its cadence, and what none of it needs by way of migration.

**Read ADR-0047 and ADR-0049 first.** This ADR strikes their cap-*upper-bound* clauses in place, per
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), and reads as
their amendment rather than their replacement.

## Decision

**The operator cap on an address scope has no upper bound. A scope larger than the default cap is not
refused outright — it is admitted once the operator raises the cap, priced at policy time, and
reported honestly after declaration. The only friction above the default is a flat confirm and the
deliberate act of raising the cap. Nothing branches on the number.**

The cap mechanism ADR-0047/0049 specify — a validation at declaration, per scope, never on a sum,
family-agnostic, counting addresses — survives verbatim. This ADR removes one thing: the ceiling
above the operator-set value.

| Concern | Decision |
| --- | --- |
| Is there a hard ceiling above the operator knob? | **No.** The cap is operator-configurable with **no upper bound**. A ceiling would be [#27](https://github.com/winniel123/verge-asm/issues/27)'s invented threshold sitting in the safety path — the one place this model keeps clear. Ratifies ADR-0047/0049, which already treated the knob as raisable |
| What refuses a large scope, then | **Nothing refuses it.** Above the *default* cap it is refused **until the operator raises the cap** — a deliberate policy act, not a gate on the declaration. Below the operator's own cap it is admitted with a flat confirm |
| The declaration confirm shape | ADR-0047 [#50](https://github.com/winniel123/verge-asm/issues/50)'s **flat confirm** carrying the count — `N addresses · within your cap of M` + one `Confirm`. No interstitial, no second gesture, **nothing branches on the number**. There is **no pre-declaration completion warning** — a warning that branched on the count would violate #50 and drift toward the out-of-scope completion-guarantee |
| Where the cost of a large scope is shown | **At policy time, on the Settings #206 cap control** (Variant C, the *policy-forward dial*). The control shows the largest scope the current cap admits and the sweep cadences a cap-sized scope needs. The operator chooses how large a lag they accept **when they set the cap**, not when they declare. The cadence consequence is a **policy-time readout, not a declaration-time gate** |
| A scope that cannot finish inside its cadence | **Reported, never hidden or refused.** ADR-0005's overlap rule turns an uncompletable scope into a permanent skip; `Coverage` and `Scans` state the lag. Completion is **accepted-and-reported**, not guaranteed — the guarantee path is out of scope |
| What `Coverage` says | **`declared / current`** plus an oldest-current as-of. `current < declared` is the honest lag. Currency stays **nominal** (`k × declared cadence`, never stretched to the effective cadence — ADR-0005's fence: execution never redraws the Declared `Scan`). There is **no reached-ever count** — that is inventory, which [#28](https://github.com/winniel123/verge-asm/issues/28) refuses |
| What `Scans` says | The **predicted + effective cadence** — arithmetic, no new domain term — alongside ADR-0005's existing skip events. Division of labour: `Coverage` is evidential, `Scans` is operational; operational lag never leaks into a comparison figure |
| Does the enumeration stream? | The model **guarantees memory is never a ceiling bound** — a property, not a mechanism. It rests on ADR-0005's one-address-per-batch plus a streamed fan-out, so no record holds the whole scope. Single-probing is required (dedup against the small resolved set). Enumeration stays **whole**; load-completion is not a bound either |
| The atomicity of the fan-out | **An execution concern, not a ceiling.** ADR-0005's atomic dispatch relaxes into chunked commits under the existing `(scan, scheduled_time)` idempotency key, mirroring the memory fix. This is an ADR-0005 amendment for implementation, not a new bound |
| Family | **Family-blind.** Cost is a pure address count and completion cost tracks the count, so an IPv4 `/8` and an IPv6 `/104` (both 2^24) cost and confirm identically. No family guard, no family-shaped ceiling |
| The seductive 2^32 ceiling | **Rejected.** No IPv4 scope exceeds 2^32, so a ceiling there refuses **only** IPv6 — the family guard wearing a family-blind number — and it reopens the no-ceiling ruling. Any cap ≤ 2^32 admits only *completable* IPv6 scopes; only a cap **above** 2^32 admits an **inert** IPv6 scope, which is an ordinary over-large scope: flat confirm, honest reporting, no family treatment |
| A refused over-cap IPv6 declaration | **Unchanged — ADR-0052 already owns it.** The refusal names the `custody extension` route and names the knob **only to shut it**. A raisable knob was already assumed |
| Trailing-edge staleness `Gap`s | **No message at all.** A trailing-edge `Gap` is the ordinary currency-lapse `Gap`, which mints **no operator message** anywhere in the model. Per-address messages are refused (they would mint messages the model never fires); a single folded scope message is refused too (that would import ADR-0047's Declared-act aperture parity onto a **measurement-caused** `Gap`). The honest fold already exists — `Coverage`'s `declared / current` **is** the scope-level count of these `Gap`s |
| Retention sizing at a raised cap | **Documented, not re-opened.** ADR-0041's retention model is scale-invariant (retained-by-reader, spans-never-compacted, sizing-as-projection). **Ship-unbounded holds.** The one clause the cap touches — the ship-unbounded default's `~13 GB/year at the 1024 ceiling` grounding — moves onto #850's policy-forward dial, priced at policy time beside the cadence lag |
| Proposal-confirm behaviour | **No new Proposal behaviour, no second cap.** ADR-0047 already checks the cap on confirming a `Proposal`; unbounding the cap **restores ADR-0022's original regime** (safety from **singular** confirm, not the cap). A large `Proposal` is refused until the operator's policy cap admits it; **decline-all** stays the expected response. Confirm shape unchanged; bulk-confirm stays forbidden and is *strengthened* |
| Migration | **None.** One *"no migration required"* clause (below). The default stays `1,024` — only the upper bound is removed — and the cap is applied at declaration and read by no rule, so nothing stored is invalidated and nothing is seeded on upgrade |

## Rationale

### There is no ceiling because a ceiling is [#27](https://github.com/winniel123/verge-asm/issues/27)'s refused threshold, and the ADRs already said so

This is the load-bearing ruling and it is a ratification rather than a new position. The whole map
was charted to raise the cap; the question it had to settle was whether *raising* it should hit a
second, higher wall. It should not, on the ground ADR-0047 and ADR-0049 both already reached from the
other side.

ADR-0049 rejected *"Invent a hard ceiling the knob may not pass"* in its own Alternatives table:
*"A threshold inside the safety path with no owner and no derivation behind its value — #27's shape
exactly."* ADR-0047 admitted the cap **only** by distinguishing it from #27's refused registry-size
cap: this cap *"adjudicates cost, not truth"* and *"cannot fail silently at all, because its only
failure mode is a declaration that does not take."* A **ceiling on the knob** would have neither
property. It would sit in the safety path with no derivation behind its number, and it would refuse a
declaration the operator deliberately sized and paid for. The cap survives #27; a ceiling would be
#27.

So the knob's only backstop is friction, and there are two kinds, both deliberate and neither
branching on the number:

- **The raise itself.** Above the default, a large scope is refused *until the operator raises the
  cap*. That refusal is ADR-0052's route-naming refusal, unchanged. Raising the cap is a policy act
  the operator performs once, with the cost in front of them (below).
- **The declaration confirm.** ADR-0047 #50's flat confirm, carrying the count and nothing else.

### Atomicity was the last candidate forcer of a ceiling, and it is an execution concern

The map interrogated every reason a hard ceiling might be *forced* rather than chosen, and disposed
of each. Memory was deleted first ([#846](https://github.com/winniel123/verge-asm/issues/846)): the
model guarantees memory is never a ceiling bound, resting on ADR-0005's one-address-per-batch and a
streamed fan-out so that no record ever holds the whole scope. `scan/hot.go`'s one-job-per-`Vantage`
build contradicts ADR-0005's one-address-per-batch and is the execution gap that must close — flagged
there, not decided here.

The last candidate forcer was **transaction duration**. #846 fixed memory, not the atomic dispatch's
duration. The ruling: this is *real* once per-address batching lands, but it is an **execution
concern, not a ceiling**. ADR-0005's atomic fan-out relaxes into **chunked commits** under the
existing `(scan, scheduled_time)` idempotency key — the same shape as the memory fix. The trade-off
is honest: relaxation re-opens ADR-0005's partial-dispatch under-coverage, mitigated by the
idempotency key and already inside the honest-lag tolerance the surfaces report. **This is an
ADR-0005 amendment for the eventual implementation, not a new ticket and not a bound on the knob.**

### The cost is shown at policy time, so the declaration can stay a flat confirm

ADR-0047 #50 fixed one rule the whole map runs under: *"nothing branches on the number."* A large
declaration cannot be met with a warning that reads the count and decides, because that is exactly
the branch #50 forbids, and it drifts toward the completion-guarantee the map ruled out of scope.

But the cost is real and must be shown somewhere. **Variant C — the policy-forward dial** — puts it
at policy time, on the Settings #206 cap control, rather than at the moment of the declaration. The
control front-loads the consequence: the largest scope the cap admits, and the sweep cadences a
cap-sized scope needs. The operator decides how large a lag they will tolerate **when they set the
cap**. The declaration that follows is then a bare within-policy confirm — `N addresses · within your
cap of M` — carrying the count and nothing that branches on it.

Variant A (count only) was rejected for showing too little of the consequence the control exists to
surface. Variant B (a cadence line at the declaration) was rejected for reading as the
pre-declaration completion warning #848 ruled out. C puts the cost where the decision actually is.

### Honest lag: the surfaces report a scope that cannot finish, and never hide it

A scope larger than its cadence can complete is a permanent skip under ADR-0005's overlap rule. The
map ruled that this is **reported, never hidden and never refused** — the destination is a
*non-completing cadence reported honestly*, not resumable scans.

The reporting divides by surface, and the division is load-bearing:

- **`Coverage` is evidential.** It renders `declared / current` — where `current < declared` **is**
  the honest lag — plus an oldest-current as-of. Currency stays **nominal** (`k × declared cadence`),
  never stretched to the effective cadence, because ADR-0005's fence is that execution never redraws
  the Declared `Scan`. There is **no reached-ever count**: a count of addresses ever reached is
  inventory, which #28 refuses.
- **`Scans` is operational.** It states the predicted and effective cadence — plain arithmetic, no
  new domain term — beside ADR-0005's existing skip events.

Operational lag lives on `Scans` and never leaks into a `Coverage` comparison figure. The
trailing-edge `Gap` that a lapsing scope opens mints **no message at all**: it is the ordinary
currency-lapse `Gap`, which is silent everywhere in the model, and `Coverage`'s `declared / current`
is already its honest, scope-level fold.

### Family-blind, and the 2^32 ceiling is a family guard in disguise

The cap is a pure address count, and completion cost tracks the count, so **cost is family-blind at a
given cap**: an IPv6 `/104` (2^24 addresses) costs exactly what an IPv4 `/8` (2^24) costs. This is the
new argument the IPv6 edge turned on. Any cap **≤ 2^32** admits only *completable* IPv6 scopes, so a
raised cap adds **no new family hazard**; only a cap **above 2^32** — which has no IPv4 purpose, since
IPv4 holds 2^32 — admits an inert IPv6 scope.

The tempting move is a ceiling at 2^32. It is **rejected**: no IPv4 scope exceeds 2^32, so a ceiling
there refuses **only** IPv6. It is the family guard ADR-0049 already refused, wearing a family-blind
number, and it reopens the no-ceiling ruling. An IPv6 scope an operator raises the cap past 2^32 to
declare is an **ordinary over-large scope**: it enumerates, it never completes (a permanent skip), it
gets a flat confirm and honest reporting, and it gets **no family treatment**. A **refused** over-cap
IPv6 declaration is unchanged — ADR-0052 names the extension and names the knob only to shut it, on a
knob it already assumed raisable.

### Retention, Proposals, and migration each ratify rather than reopen

Three surfaces the map checked and found unmoved:

- **Retention ([#851](https://github.com/winniel123/verge-asm/issues/851)).** ADR-0041's model is
  scale-invariant: it retires a corpus **by reader**, so volume is the axis it was built to ignore.
  Retained-by-reader, spans-never-compacted and sizing-as-projection all survive a raised cap
  verbatim. **Ship-unbounded holds** — its principle, *"the honest default for a quantity nothing
  bounds is not to bound it"*, is scale-invariant too. The one clause the cap touches is that
  default's `~13 GB/year at the 1024 ceiling` grounding, and the disk-growth projection moves onto
  #850's policy-forward dial, priced beside the cadence lag. A default expiry was rejected: it re-adds
  the wall clock ADR-0041 refused, on the >99% of installs with nothing to retire.
- **Proposals ([#884](https://github.com/winniel123/verge-asm/issues/884)).** The "bypass" premise is
  corrected: ADR-0047 **already** checks the cap on confirming a `Proposal`, and ADR-0022's
  large-confirm examples (the AWS 76M) predate the cap. Unbounding the cap **restores ADR-0022's
  original regime** — large single confirms admissible, safety from **singular** confirm, not the
  cap. A confirmed `Proposal` is a declaration the operator did not type, so a large `Proposal` is
  refused until the operator's policy cap admits it, and **decline-all** stays the expected response.
  There is **no second cap and no Proposal special-case**; reporting parity is by construction,
  because a confirmed `Proposal` is a `Seed`.
- **Migration ([#885](https://github.com/winniel123/verge-asm/issues/885)).** See the clause in
  Consequences. Nothing is stored, nothing is invalidated, nothing is seeded.

## Consequences

### The cap-upper-bound clauses are struck at their sites, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)

This ADR removes the upper bound on the knob, which withdraws the clauses in ADR-0047 and ADR-0049
that read that bound as fixed. Per ADR-0058 (+ its #106 amendment) the withdrawal is written **at the
superseded clause**, in this same change, because *"an address scope wider than the cap is not
declarable"* is a **gate**, not a figure. The mark takes the struck-clause-plus-pointer form. Both
directions are recorded, so the *one-sibling-got-the-mark* failure ADR-0058 measured does not recur:

- **[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md)** — the Decision-table row *"An
  address scope wider than the cap → Not declarable at the shipped default"*, and the #85 amendment's
  *"`/117` and shorter are refused by the cap, which is every prefix an operator is assigned"* clause,
  are struck and pointed here.
- **[ADR-0049](./0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)** — the
  Decision-table rows *"Which IPv6 prefixes are therefore declarable"*, *"Which are not"*, and
  *"Sweeping IPv6 space"* are marked where they read the upper bound as fixed.

The **polarity** is important and is stated at every mark: the withdrawal removes the **upper bound**
on the knob, never the cap. The cap-at-declaration, per-scope, never-on-a-sum, family-agnostic
mechanism survives verbatim. `Alternatives rejected` entries in both ADRs — including ADR-0049's
*"Invent a hard ceiling the knob may not pass"* — are **out of scope** for the mark by ADR-0058
#106's voice test, and ADR-0049's Rationale *"No ceiling is invented"* already agrees with this ADR
and is left standing.

### No behavioural code change lands with this ADR

- **No migration.** Every declaration that predates this change is `≤` the old 1,024 cap and stays
  valid. The cap is applied at declaration and read by no rule (`internal/seed/seed.go`), and the
  shipped default is unchanged, so an upgraded install with an untouched knob behaves identically.
  Persistence for the operator-set cap (#850 / Settings #206) is future work and seeds its empty
  state from the same compiled default. The code already tolerates a since-changed cap
  (`maxEnumCapHint`, `internal/seed/seed.go`).
- **The knob's persistence and the Settings #206 policy-forward control are implementation for
  #850/#206**, not this ADR. This ADR carries no init rule and no store schema.
- **`scan/hot.go`'s one-job-per-`Vantage` build contradicts ADR-0005's one-address-per-batch** and is
  the execution gap #846 named. It is recorded here as owed work, not decided here.
- **ADR-0005 gains an amendment when implementation lands** — the atomic fan-out relaxes into chunked
  commits under the existing `(scan, scheduled_time)` idempotency key. Recorded as the trade-off it
  is (re-opened partial-dispatch under-coverage, inside the honest-lag tolerance), for the pass that
  writes the fan-out.

### The glossary records the removed ceiling

- **[`CONTEXT.md`](../../CONTEXT.md)'s `Seed` entry** reads, in the present tense, that *"the cap
  refuses every prefix an operator is actually assigned"* for IPv6. That is true at the **default**
  cap and is the fixed-upper-bound reading this ADR removes. The entry gains that the operator cap has
  no ceiling: a larger IPv6 scope becomes **declarable-but-inert** once the operator raises the cap
  past 2^32, never **swept** (it never completes) — so *"IPv6 space is not swept and no configuration
  makes it sweepable"* survives verbatim, while the declarability absolute is qualified. No term is
  added and none changes meaning.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Invent a hard ceiling above the operator knob** | A threshold in the safety path with no owner and no derivation behind its value — [#27](https://github.com/winniel123/verge-asm/issues/27)'s shape exactly, and the one ADR-0047 worked to stay clear of. It also refuses a declaration the operator deliberately sized and paid for. ADR-0049 already rejected it in the same words; this ADR ratifies that |
| **A pre-declaration completion warning that branches on the count** | Violates ADR-0047 #50's *"nothing branches on the number"*, and drifts toward the completion-guarantee the map ruled out of scope. The cadence cost belongs at policy time (Variant C), reported after declaration on `Coverage`/`Scans`, never as a gate on the act |
| **Show the cadence cost as a line on the declaration confirm (Variant B)** | Reads as the pre-declaration warning above. The declaration stays a bare within-policy confirm; the cost lives on the Settings #206 dial |
| **Show only the count on the cap control (Variant A)** | Too little of the consequence the control exists to surface. The operator sets a cap without seeing the lag a cap-sized scope incurs |
| **Cap at 2^32** | No IPv4 scope exceeds 2^32, so the ceiling refuses **only** IPv6 — a family guard wearing a family-blind number. It reopens the no-ceiling ruling and re-imports the family branch ADR-0049 refused. A cap ≤ 2^32 already admits only completable IPv6 scopes |
| **Guard or refuse a large IPv6 scope by family** | Cost is family-blind at a given cap, so a large IPv6 scope past 2^32 is an ordinary over-large scope: enumerated, never completing, flat-confirmed, honestly reported. A family guard is the model's only family-aware rule and makes a claim about IPv6 where the cap makes a claim about us — the distinction ADR-0047/0049 used to admit the cap |
| **Fold trailing-edge staleness into one scope message** | Imports ADR-0047's Declared-act aperture parity onto a `Gap` that is **measurement-caused**, not a Declared act. A currency `Gap` mints no message anywhere in the model; `Coverage`'s `declared / current` is already its honest, scope-level fold. Per-address messages are refused for the same reason |
| **Stretch currency to the effective cadence for a lagging scope** | Crosses ADR-0005's fence: execution never redraws the Declared `Scan`. Currency stays nominal (`k × declared cadence`); the lag is the honest `current < declared`, not a redrawn bound |
| **Add a reached-ever count so the operator sees progress** | A count of addresses ever reached is inventory, which #28 refuses. `Coverage` reports `declared / current` and an oldest-current as-of, both evidential |
| **A default expiry to bound disk growth at a raised cap** | Re-adds the wall clock ADR-0041 refused, on the >99% of installs with nothing to retire, and takes pricing from the one operator with the forensic context. ADR-0041's retire-by-reader model is scale-invariant; ship-unbounded holds and the disk projection moves onto the policy dial |
| **A second cap, or a Proposal-confirm special case, for large confirms** | ADR-0047 already checks the cap on a `Proposal` confirm, and unbounding it restores ADR-0022's original singular-confirm regime. Safety is the singular confirm, not the cap; decline-all stays the expected response; a confirmed `Proposal` is a `Seed`, so reporting parity is by construction |
| **A migration note / init rule for the raised default** | Nothing to migrate: the default is unchanged, every stored declaration is `≤ 1,024`, the cap is read by no rule, and nothing is persisted today. Recorded as a clause, matching the repo's inverse convention (ADR-0104/0105/0122/0125 all state *no migration* as a short clause) |
| **Supersede ADR-0047/0049 (`Status: Superseded`)** | Overstates a non-reversal — the cap mechanism stays and only its upper bound is removed — and invents a form this corpus never uses (ADR-0058: no ADR here carries `Status: Superseded`). A new ADR that **amends** matches ADR-0049 and ADR-0125, the two closest precedents |
