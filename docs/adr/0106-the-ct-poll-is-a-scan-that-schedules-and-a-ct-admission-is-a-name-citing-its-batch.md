# ADR-0106: The CT poll is a `Scan` that schedules and bounds nothing, and a CT admission is a `Name` citing its `Batch`

- **Status:** Accepted
- **Date:** 2026-08-16
- **Ticket:** [#250 Build the crt.sh CT runner (#241 framing 1)](https://github.com/winniel123/verge-asm/issues/250)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

Certificate transparency is v1's flagship keyless discovery source and it has never run. [ADR-0003](./0003-third-party-source-consent-bar.md) cleared `crt.sh` on the absence of terms and shipped it enabled; [ADR-0027](./0027-a-source-may-admit-without-observing.md) ruled what it does — it **admits `Name`s** on `authority: inferred`, observes no facet, holds no timeline, and its `Citation` hop is the `Batch`. [#241](https://github.com/winniel123/verge-asm/issues/241) then found the honesty defect that opened this ticket: `crt.sh` was catalogued and shipped *on*, but **nothing queries CT**, so it was presented as *catalogued — not yet executing* (`NoRunner`) rather than counted as an active source. #241 fixed the false reassurance; this ticket adds the capability behind it, and it is `ready-for-human` because it carries design that wanted an ADR before code.

The ticket named three unresolved questions. Two are already settled, and the useful first move is to record that rather than re-litigate them:

- **Citation aging.** #250 records #176 as *"closed without deciding."* That is stale. [#176](https://github.com/winniel123/verge-asm/issues/176) · [ADR-0096](./0096-a-citation-never-ages-it-is-contradicted-and-only-an-enumerable-sources-silence-can-do-it.md) **decided it**: a `Citation` never ages on any hop, and CT's `corroborative` `completeness` means its silence retires nothing — the withdrawal route is refused on **capability**, because CT is append-only and a citation clock over it fires only on our own instrument's defects. So this runner implements **no** aging and **no** withdrawal. A CT-admitted `Name` leaves the way every `Name` leaves — our own resolver measures a Name Error (ADR-0096 §1).
- **Admission shape.** ADR-0027 fixed it and [ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md) bounded it: CT admits the `Name`s a SAN carries **except** any `dNSName` or `common_name` value with an asterisk label (`0x2A`), which denotes a set and admits none of them. The `Citation` hop is the `Batch`; the chain terminates at the name-scope `Seed` the query was built from.

What was genuinely open is the **CT poll's cadence and the mechanism that schedules it**. ADR-0096 §7 and [ADR-0084](./0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md) both stated this as a live hole and **declined to mint the `Scan`**, on one ground: a *generic* admitting-source `Scan` (the shape ADR-0084 refused the name `discovery` for) cannot be sized, because `passive-discovery-sources.md` §16.2's residue is one row short — Common Crawl is mis-filed there — so the covered set of "sources that admit without observing" is undecided. This ADR is that successor, and it mints the `Scan` by drawing its scope the way ADR-0084 drew `dns`'s.

## Decision

| Concern | Decision |
| --- | --- |
| Citation aging | **Not reopened.** ADR-0096 decided it — a `Citation` never ages, CT's silence retires nothing. This runner has no withdrawal path |
| The scheduling mechanism | **A sixth `Scan`, `ct`** — worker-read, one `Batch` per name-scope `Seed`, over the CT exchange (`crt.sh`) |
| Its name | **`ct`** — the exchange (certificate transparency), the ADR-0084 move. Not `crtsh` (the instrument), not `discovery` (the refused generic) |
| Why §7's blocker does not bite | §7 could not size a *generic* admitting-source `Scan`; naming the **exchange** draws the scope truthfully regardless of Common Crawl, which is a web-crawl index and a **different** exchange that would take its own `Scan` |
| Currency bound | **None.** Per ADR-0096 §7's pre-armed rule: a `Scan` over a source that admits without observing carries no currency bound and no withdrawal power — the currency rule quantifies over observations, and there are none |
| Withdrawal power | **None.** It schedules and gives `Coverage` a row; it bounds nothing |
| Port list | **None** — the third `Scan` with none, after `zone` and `dns`. A CT query is not a connect |
| Vantage | **None** — worker-read, like `zone`. A logged certificate is not a function of where we read the log from |
| Cadence | The operator's, **shipped at daily** — the discovery cadence, matching `dns`. An operator dial; moving it moves no version and `Break`s nothing |
| Aperture inputs | **Seven, unchanged.** A cadence is not aperture (ADR-0084); CT is already the first input, *enabled sources* |
| What a CT admission is, in the store | A durable **`admitted_name`** row: the `Name`, `source = crtsh`, the covering `Seed`, and the **`Batch`** that admitted it — ADR-0027's *admits without observing* materialised. **No observation, no span, no facet** |
| Membership of an admitted `Name` | **Measured, not admitted** (ADR-0096 §5). The `admitted_name` row records *how it entered*; it becomes a current member only when our resolver resolves it. A `Citation` is necessary for membership and not sufficient |
| Feeding admitted names into resolution (wave-1) | **NOT in this ticket. Stated, not absorbed** — the #124/#142/#176 move. The seam is `admitted_name`; a successor unions it into the `dns` `Scan`'s resolution set |
| Wildcard / partial-wildcard values | **Admit nothing** (ADR-0060) — no value with `0x2A` in any label, in `name_value` or `common_name` |
| Names outside the queried name scope | **Filtered** (ADR-0047: the `Seed` decides which names are inside). A cert bearing a foreign SAN admits only the names under the scope we asked about |
| A non-200 / failed fetch | **No admission, and never an absence.** It retries, then dead-letters a `Batch` with an **empty** scope (ADR-0005). Only a well-formed 200 admits anything |
| The 999-row cap | **Stated, not modelled around** (ADR-0027) — a silent truncation under a 200. It admits fewer, never asserts more are gone, because `corroborative` |
| The 5 req/min throttle | **Honoured instance-wide, in Postgres** (ADR-0005) — a reservation row, not per-worker memory |
| Enablement | The `ct` `Scan` ships **enabled** (the schedule); `fanOutCT` gates on the **`crtsh` source** being enabled. Two controls: the `Scan` is the Declared cadence, the source toggle is ADR-0003 consent |
| `NoRunner` | **Reversed.** `crtsh` ships on, toggleable, and executing — the state ADR-0003 always intended and #241 held open pending this runner |

## Rationale

### It is a `Scan`, because a recurring measurement takes one and CT had none

ADR-0005's rule, applied four times since (`tls-acceptance`, `zone`, `dns`): *a measurement needing a cadence of its own takes a `Scan` of its own.* The CT poll needs a cadence — a name issued a certificate today should enter on some stated clock, and `Coverage`'s whole job is *did it run, and when* — and it has had none anywhere in the corpus. Nothing else in the model schedules recurring work, so hanging the poll off anything else is the hidden-field failure ADR-0005, #124 and ADR-0084 have each refused; ADR-0084 refused the very *name* `discovery` because it would tell the next session to hang the crt.sh poll off `dns`. The poll is a `Scan`.

What makes this `Scan` unlike the other five is that its source **admits without observing**, and ADR-0096 §7 already wrote down what that means for it before it existed:

> A `Scan` covering a source that admits without observing carries no currency bound and no withdrawal power. ADR-0007's currency rule quantifies over observations, and such a source produces none. The bound has an empty domain. It schedules, and it gives `Coverage` a row; it bounds nothing.

So `ct` is a scheduler and nothing more. It opens no `Gap` (CT holds no timeline for one to sit on — ADR-0027), it retires no subject (ADR-0096), and it moves no aperture input (a cadence is not aperture — ADR-0084). It is the first `Scan` whose batches carry no observations at all.

### It is named `ct`, and that is how §7's blocker is answered rather than waited out

ADR-0096 §7 and ADR-0084 declined to mint this `Scan` on a specific, correct ground: a `Scan` whose scope is *"sources that admit without observing"* cannot be sized, because §16.2's enumeration of that residue is one row short — Common Crawl yields hostnames and observes no facet, which is ADR-0027's shape a second time, and its shipping status is undecided in the corpus. A `Scan` cannot be named for a scope whose membership is undecided.

That blocker is real and it is **only about the generic name**. It dissolves the moment the `Scan` is named for its **exchange**, which is the move ADR-0084 made to choose `dns` over `discovery`: *a `Scan` name must describe its scope truthfully.* Certificate transparency is a single, well-drawn exchange — `crt.sh`'s query-by-domain over the CT logs — and its scope is the name-scope registrable domains, exactly `dns`'s and `zone`'s. Common Crawl is not certificate transparency; it is a web-crawl index, a **different** exchange, and if it ever ships it takes its own `Scan` for the same reason `zone` and `dns` are two. So `ct` describes its scope truthfully whether or not Common Crawl is ever decided, and the undecided-residue objection — which was an objection to *one `Scan` for the whole category* — never reaches it.

`crtsh` as the name loses for the reason `authority` lost for `dns`: it names the **instrument**, not the exchange, and a better CT front door (a log directly, Cert Spotter with a key) would make the name a lie about a scope it still covers. The instrument's name lives on the **source** (`crtsh`), where ADR-0003 put it; the exchange's name is the `Scan`.

### A CT admission is a row, because the estate is observation-derived and CT produces no observation

This is the ticket's real design work, and it is forced by an architecture fact ADR-0027 did not have in front of it. In the shipped model a subject is in the estate **only** through observations: membership reads the `resolution` timeline, and a `Citation` is the earliest live `resolution` observation and the `Batch` it rode in on. CT produces no observation. So ADR-0027's *admits a `Name`, `Citation` hop is the `Batch`* has, today, **nowhere to land** — a CT `Batch` with zero observations records that we ran and admits nothing anything can read.

The faithful materialisation of ADR-0027 is a durable **`admitted_name`** row: the `Name`, the `source` (`crtsh`), the covering `Seed` (the chain's terminus), and the `Batch` that admitted it (the `Citation` hop, exactly as ADR-0027 names it). This is not a new `Subject` kind and not a timeline — ADR-0027 refused both, and this refuses them again. It is the provenance record ADR-0027 described in prose, written down: *why is this name here* answered by *this CT batch, under this seed*.

What it deliberately is **not** is membership. ADR-0096 §5 is decisive and it keeps this ADR consistent with itself:

> A `Citation` is necessary for membership and not sufficient... a `Name`'s membership is measured. Holding a citation asserts nothing.

So an `admitted_name` row does not put the `Name` in `ListCurrentNameSubjects`. The `Name` becomes a current member when our own resolver resolves it and writes a `resolution` observation — *"acquires a `resolution` timeline from our own resolver within one cadence, and leaves by Name Error like any other"* (ADR-0027, ADR-0096 §1). That is the correct behaviour and it is the whole reason CT needs no timeline of its own.

### The wave-1 seam is stated, not built — the move that produced ADR-0084 and ADR-0096

Here is the honest residue, written down rather than smoothed: **the resolver does not yet resolve discovered names.** `fanOutDNS` resolves the name-scope `Seed` domains and only those (`ListNameSeedDomains`); the `dns` `Scan` is wave-0. So an `admitted_name` row is, today, inert until a successor unions it into the resolution set — at which point ADR-0027's "acquires a resolution timeline within one cadence" becomes true in code as well as in prose.

Building that union in this ticket was considered and refused as premature, on the model rather than on effort. Feeding CT-discovered names into the `dns` `Scan` widens the resolved population, which moves the **control-probe population** (an aperture input), changes what `wildcard-discrimination` reads, and — once those names resolve to addresses — reaches the probing gate and `Custody`. That is the discovery→measurement wave the project has deliberately not built, and it is its own decision with its own golden-corpus obligations (ADR-0041's membership-vector corpora). Minting it inside a runner ticket would smear an estate-widening change through a fetch-and-parse change. The seam is `admitted_name`; the successor reads it. This is the #124/#142/#176 discipline: *state the wider hole, ticket it, do not guess it here.*

### Non-200 admits nothing, and this is the standing rule at a new door

The map's oldest rule: *a source that errors must produce no observation, never an observation of absence.* CT is where it bites hardest, because the instrument is measured unreliable — **[measured 2026-07-31, quoted from `passive-discovery-sources.md` §7, not re-run]** 4 successes in 8 identical requests, including two spurious **404**s against a URL that served a 95 KB body seconds earlier. A naive client maps 404 → *no certificates*, which in a drift product is a fabricated all-clear.

So the runner treats **every** non-200 as transient failure: it retries on the ordinary queue backoff (attempts, unlike `zone`'s single read, because an HTTP fetch has transient failure to back off from), and past the attempt budget it **dead-letters a `Batch` with an empty scope** — the ADR-0005 machinery that records *we tried and covered nothing* without asserting a single absence. Only a well-formed 200 with a parseable JSON array admits anything. The 999-row cap is a well-formed 200 that truncates silently; it admits fewer names and, because `corroborative`, its shorter answer asserts nothing about the names it dropped. Both are the same discipline: CT's silence, however it arises, is not evidence.

### Wildcards and foreign names are filtered at admission, per rulings already made

ADR-0060 is applied verbatim: no `dNSName` or `common_name` value containing an asterisk label admits a `Name` — a wildcard denotes a set, a partial wildcard has two denotations, and both are refused rather than interpreted. ADR-0047 is applied at the other edge: a certificate legitimately carries SANs for several estates, and the `Seed` decides which names are inside — so a `crt.sh` answer for `example.com` admits only the names under the scope we queried, never a co-tenant's name that shared the certificate. Neither is new policy; both are existing rulings arriving at the admission step.

### The throttle is instance-wide and lives in Postgres, because ADR-0005 said so and the source is abused

ADR-0005 fenced this off in advance: the crt.sh limit *"is per-source across the whole instance — so it lives in Postgres alongside the queue, not in worker memory."* The operator asked for 5 req/min per IP and has cut limits twice for abuse; shipping a naive retry loop is *"the single most likely way this project ends up rate-limited into uselessness for everyone"* (`passive-discovery-sources.md` §2.2). So the throttle is a Postgres reservation: each fetch atomically claims the next 12-second slot instance-wide before it goes on the wire, so `--scale worker=N` cannot exceed the ceiling. It sits outside every derivation — it changes only timing, and can neither manufacture nor suppress an admission.

### Enablement is two controls, and that is the model, not an accident

The `ct` `Scan` is the Declared recurring intent — the cadence — and it ships enabled, because a disabled `Scan` never fires. The `crtsh` **source** is ADR-0003 consent — *may this source run at all* — and it is the operator's toggle. These are genuinely two things: `Scan` is Declared and carries no consent, and source enablement carries no cadence. So `fanOutCT` fans out on the `Scan`'s cadence and produces jobs only while the `crtsh` source is enabled; toggling the source off leaves the `Scan` firing over an empty scope, which is the same legible zero-job state `zone` has when no file is supplied. This is the first time the dispatcher reads `source_state` — until now no source had a runner for it to gate — and the gate lands in `fanOutCT` where the scope is drawn, not in the generic dispatch loop.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) is amended in three entries and gains one term.** `Source` records that CT's admission is now materialised as an `admitted_name` row citing its `Batch`, and that admission is not membership; `Scan` records a sixth kind, `ct`, that schedules and bounds nothing; `Citation` records that a CT-admitted `Name`'s hop is its `Batch` and that the row holding it is `admitted_name`. `admitted_name` enters under Observed-adjacent provenance as the store of an admission without an observation.
- **There are six `Scan`s and still two port tiers.** Every site stating *five* is correct at its date; the live absolute is the map's to move. `ct` is the third `Scan` with no port list and the second worker-read one.
- **`crt.sh` presents as executing again — #241 is reversed.** `NoRunner` comes off the catalogue row; `crtsh` ships on, toggleable, and counted in the *enabled sources* aperture line. The *catalogued — not yet executing* bucket loses its only member.
- **The dispatcher reads `source_state` for the first time.** The read is confined to `fanOutCT`; no other `Scan` gains a source gate, because no other `Scan`'s source has an on/off the operator sets against a running instrument.
- **A new corpus, `admitted_name`, is retained under ADR-0041's `Batch` limb.** A CT `Batch` is retained while it is the current admission `Citation` of a name in the estate, released only when a later CT `Batch` re-admits the same `Name` and takes the role over — the normal case on every poll of an append-only source (ADR-0096's stated residue, now with a table under it).
- **No `Gap`, no withdrawal, no facet, no timeline, no span, no observation from CT.** The runner writes `batch` and `admitted_name` rows and nothing in the drift engine. `foldObservationsIntoSpans` is never called on a `ct` batch.
- **The wave-1 successor is owed and drawn.** Feeding `admitted_name` into the `dns` `Scan`'s resolution set is the ticket that makes an admitted `Name` a measured member; it inherits an estate-widening decision (control-probe population, `Custody`) and its golden-corpus obligations, and it is not a fetch-and-parse change.
- **The mis-issuance / CT-fed-facet line is not reopened.** ADR-0027's out-of-scope entry stands; this ADR mints no facet, no value and no observation from CT.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Reopen citation aging and give `ct` a currency bound that withdraws CT names** — the ticket's own framing of #176 as undecided | It **is** decided — ADR-0096, on capability: CT is append-only, so a citation clock's complete firing set is instrument defects (the source errored, the 999-cap truncated, a wildcard-only SAN was never admitted), none a fact about the estate. A bound over an admitting source has an empty domain. This runner withdraws nothing |
| **Do not mint a `Scan`; hang the CT poll off `dns`** since CT names are resolved there anyway | The hidden-field failure ADR-0005, #124 and ADR-0084 each refused — and ADR-0084 refused the name `discovery` precisely because it would tell the next session to do this. Disabling `dns` would silently stop CT, under a name that does not claim to cover it |
| **Name the `Scan` `discovery` or `passive`, to cover every admitting source at once** | ADR-0096 §7's live blocker: the residue of admit-without-observe sources is undecided (Common Crawl mis-filed), so the scope cannot be drawn. Naming the exchange (`ct`) sidesteps it — a different exchange takes a different `Scan`, exactly as `zone` and `dns` are two |
| **Name it `crtsh`** | Names the instrument, not the exchange. A better CT front door would make the name a lie about a scope it still covers. The instrument's name is the source's; the exchange's is the `Scan`'s |
| **Write a CT `Batch` with zero observations and no admission row** — the literal minimum, "a batch records we ran" | Admits nothing anything can read: the estate is observation-derived and CT produces no observation, so ADR-0027's admitted `Name`s would vanish. The `admitted_name` row is what materialises ADR-0027's *admits without observing* at all |
| **Admit CT names as `resolution` observations so they appear in the estate immediately** | Manufactures an observation nobody made — ADR-0027's decoder rule, *a source's shape is translated, never its facts*. A CT row is not a resolution; asserting one fabricates a measured value and pairs a `corroborative` timeline against our `enumerable` resolver's, which ADR-0027 already refused |
| **Feed admitted names into the `dns` resolution set in this ticket** | Correct in the end and premature here. It widens the resolved population, moving the control-probe aperture input, `wildcard-discrimination`'s reads, and — via resolved addresses — the probing gate and `Custody`. That is the discovery→measurement wave, its own decision with its own golden-corpus obligations. Smearing it through a runner ticket is exactly what ADR-0084's *state-it-rather-than-absorb-it* forbids |
| **Throttle in worker memory** | ADR-0005: the limit is per-source across the whole instance, so `--scale worker=N` needs cross-worker state in Postgres. In-memory would let N workers each hit 5/min and get the project rate-limited |
| **Map a 404 to "no certificates for this domain"** | The measured-dangerous failure (§7): the same URL 404s and 200s seconds apart. It is the standing rule's exact violation — an observation of absence from an error — and in a drift product it fabricates an all-clear. Non-200 is transient, always |
| **Gate the source with the `Scan`'s `enabled` flag instead of `source_state`** | Conflates two model layers. `Scan` is Declared cadence and carries no consent; the source toggle is ADR-0003 consent and carries no cadence. Folding them would make the operator's consent control disappear behind a schedule |
