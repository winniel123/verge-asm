# ADR-0220: A build-time input that ages out empties a population without opening a `Gap`, and the release answers for it

- **Status:** Accepted
- **Date:** 2026-09-07
- **Ticket:** [#1513 Decide whether an expired CT log list opens a `Gap`, and amend ADR-0106 if it does](https://github.com/winniel123/verge-asm/issues/1513)
- **Split out of:** [#1434](https://github.com/winniel123/verge-asm/issues/1434), closed by [PR #1517](https://github.com/winniel123/verge-asm/pull/1517). #1434 answered *not there* and named this question the larger change
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md). It rules absence a property of a **cell**, and a `Gap` a `Span` on a live subject's timeline. This ADR asks what holds the absence when there is no cell
- **Rests on:** [ADR-0206](./0206-a-fan-out-over-an-empty-population-is-a-legible-zero-job-dispatch-and-it-fabricates-no-batch.md) §1, which rules a fan-out over an empty population a legible zero-job dispatch. Its §5 adds that the rule does not call the population **healthy**. This ADR rules the case §5 left open
- **Rests on:** [ADR-0190](./0190-the-ct-log-list-is-a-build-time-artefact-pinned-in-the-image-refreshed-only-by-a-release-and-carrying-no-log-public-keys.md) §2, which rules the CT log list a trust input pinned in the image. Its §4 rules the refresh a release act. This ADR rules who answers when the pin ages out
- **Bounded by:** [ADR-0096](./0096-a-citation-never-ages-it-is-contradicted-and-only-an-enumerable-sources-silence-can-do-it.md), whose §7 pre-armed the rule that a `Scan` over a source which admits without observing carries **no currency bound and no withdrawal power**. That rule is untouched, and §2 below states why it is also sufficient
- **Amends nothing.** [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md) keeps its reach exactly as written. §6 states why it was not even the site the amendment would have landed on

## Context

[#1434](https://github.com/winniel123/verge-asm/issues/1434) measured a real decay. The pinned CT
log list expires, `SelectTailLogs` then selects zero logs, and the `ct-tail` `Scan` fans out zero
jobs on every 300-second tick forever. #1434 shipped three repairs and refused a fourth. The fourth
was a `Gap`, and #1513 is that question stated on its own.

**The proposal, in one sentence: when `scan.SelectTailLogs` selects zero logs, the `ct-tail` `Scan`
should open a `Gap` rather than dispatch a legible zero-job fan-out.**

### Two rulings stand in the way, and neither is wrong today

- [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)
  gives the CT `Scan` no currency bound and no withdrawal power. Nothing opens a `Gap` when the tail
  stops producing. ADR-0190's Consequences states the result plainly: *"the silence is the model
  working as ruled, and it is why this case is invisible."*
- [ADR-0206](./0206-a-fan-out-over-an-empty-population-is-a-legible-zero-job-dispatch-and-it-fabricates-no-batch.md)
  §1 rules a fan-out over an empty population a legible zero-job dispatch. A zero-log tick has that
  exact shape.

### The artefact, re-measured at `c068bb9`

`internal/scan/log_list.json` is **27,970 bytes**. `"version": "90.6"`, `"log_list_timestamp":
"2026-09-06T13:35:59Z"`. Eight operators, **48 entries** — 26 under `logs[]` and 22 under
`tiled_logs[]`. By `state`: 43 `usable`, 2 `readonly`, 3 `retired`. **Zero entries carry a `key`
field**, per ADR-0190 §5.

`SelectTailLogs` over those bytes, by date:

| Date | Logs selected |
| --- | --- |
| 2026-09-07 | 45 |
| 2027-01-05 | 29 |
| 2027-07-05 | 14 |
| 2028-01-05 | 2 |
| 2028-01-07 | 2 |
| **2028-01-08** | **0** |

The newest `end_exclusive` is `2028-01-08T00:00:00Z`. The count reaches zero on that instant and
never recovers without a release. **#1513's table measured v89.34 and is superseded by this one.**
The shape it recorded is unchanged, and the cliff date is the same.

### What the `ct-tail` `Scan` produces

`internal/queue/cttail.go` calls `qtx.InsertAdmittedName` and nothing else that reaches the drift
engine. [`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.1 states the shape: the
tail admits like crt.sh, on `authority: inferred`, citing a `Batch`, and it *"creates no new facet
and no Signal."* Its issuance event is **ephemeral**. So the tail writes **no observation, no span
and no timeline**, which is
[ADR-0027](./0027-a-source-may-admit-without-observing.md)'s *admits without observing* holding at a
second door.

### What already ships

[PR #1517](https://github.com/winniel123/verge-asm/pull/1517) closed #1434 with three things, and
this ADR redoes none of them.

- The Sources page's CT capabilities card states the pinned version, the cut date, the count of logs
  selectable today, and the date every log expires. A zero count earns an `expired` badge and a warn
  callout.
- `TestShippedLogListOutlivesTheExpiryHorizon` (`internal/scan/cttail_test.go`) fails when the newest
  `end_exclusive` falls within **180 days** of the build date. It runs inside the required `test`
  check, so an ageing snapshot blocks a merge.
- [`release-pipeline.md`](../spec/release-pipeline.md) §10.7 sets the cadence: every release, and at
  minimum every 90 days. §10.4's checklist carries the line.

The shipped callout in
[`design-system/templates/settings.tmpl`](../../design-system/templates/settings.tmpl) already tells
the operator *"no gap opens for it."* **That sentence asserts this ruling and nothing on disk carried
it.** This ADR is what puts it there.

### The question, stated exactly as #1513 states it

An empty population is legible under ADR-0206 because the population is a true measurement of the
world. A pinned list that has aged out is not a measurement of anything. It is an input that stopped
describing the world. **Is that difference load-bearing enough to carry a `Gap`?** Or does it belong
to the class of staleness the **release**, not the runtime, answers for?

## Decision

> **No `Gap` opens. A population emptied by a build-time input that has aged out is still a legible
> zero-job dispatch under ADR-0206 §1. The difference between an empty population the world produced
> and an empty population a stale pin produced is real, and it is a difference of **cause**, not of
> coverage. A `Gap` cannot carry it, because a `Gap` is a `Span` on a timeline and no admitting-source
> `Scan` holds one. The staleness of a pinned input is the release's to answer, and the release
> answers it with a blocking build gate and an operator surface.**

### 1. The distinction is real, and it is a distinction of cause

#1513's framing is correct and this ADR does not soften it. An empty population the world produced
and an empty population a stale pin produced are **not the same fact**. The first says *we looked and
there was nothing*. The second says *we stopped being able to look, and nobody was told*.

ADR-0206 §5 already reserved this ground rather than settling it:

> **Whether a population *should* be empty.** The rule says an empty population is legible. It does
> not say it is healthy.

So the question is not whether the difference exists. It is **which artefact carries it**. The rest
of this Decision answers that, and the answer is not a `Gap`.

The repository already holds a precedent for the shape of that answer.
[#1120](https://github.com/winniel123/verge-asm/issues/1120) added a `'skipped'` `dispatch.status`
token in [`25000_dispatch_skipped_status.sql`](../../db/migrations/25000_dispatch_skipped_status.sql),
because a cadence-lag skip was *"indistinguishable from a dispatch that genuinely found nothing."*
A `LEFT JOIN` derives how many jobs a tick enqueued. No join derives **why**. So a cause needs its own
carrier, and the carrier the model reaches for is a status token or an operator surface. It never
reaches for an absence record on a timeline.

### 2. There is no denominator, and that is decisive on its own

ADR-0072 fixes what a `Gap` is:

> A `Gap` is a `Span` and it records its cause.

and

> A `Gap` sits on one timeline of a subject that is otherwise entirely alive, and only withdrawal
> changes whether the subject is there to have timelines.

**The `ct-tail` `Scan` holds no timeline.** §Context measures it: the tail writes `admitted_name`
rows, creates no facet and no Signal, and its issuance event is ephemeral. There is no span for a
`Gap` to be, and no cell for it to sit in.

The obvious repair fails too. One might put the `Gap` on the names the expired list **would have**
admitted. Those names are not in the estate. ADR-0072 rules that absence is a property of a cell, and
an unadmitted name has no row and no column. It is not even ADR-0072's *we never looked* state, which
is a cell on a subject already listed. It is nothing at all.

**#1513 pre-armed this test and this ADR reaches it.** The ticket says: *"If you cannot name the
denominator, that is itself an argument against opening a `Gap`."* The denominator cannot be named,
and it cannot be named for a structural reason rather than a temporary one. **The set of names a CT
log would have shown us is unenumerable.** That is the same fact as CT's `corroborative`
`completeness`, and §3 is why it also settles the question a second time.

### 3. A `corroborative` source's silence asserts nothing, whatever caused the silence

[ADR-0096](./0096-a-citation-never-ages-it-is-contradicted-and-only-an-enumerable-sources-silence-can-do-it.md)
rules that only an **enumerable** source's silence can contradict a citation. CT is `corroborative`.
Its silence retires nothing and asserts nothing about the world.

A `Gap` is an assertion. It says *there is a value here we could not read*. Opening one on a
zero-log tick would make the **cause** of a corroborative source's silence change what that silence
asserts. It does not. A silence we caused and a silence the world caused are equally empty of
evidence about the estate.

ADR-0096 §2 already worked this out for the neighbouring question. A citation clock over CT could
fire only on our own instrument's defects, never on the world moving. **An expired log list is
precisely such a defect**, and ADR-0096's answer to that class is that it is not a fact about the
estate. This ADR applies the same answer one door along, and mints no new rule to do it.

### 4. The one CT path that feeds a value does not decay, so the last candidate site is closed

A reader may reasonably ask whether the `Gap` belongs on the verification path instead, where CT does
reach a value.

It does not, and the reason is already ruled. ADR-0190 §5 measures that `AllLogs` applies **no state
filter and no temporal filter**. All 48 entries stay available to the verifier for the life of the
image. The verification reader does not decay with the tail's.
[ADR-0193](./0193-a-stapled-ocsp-response-only-narrows-the-sct-set-and-no-usable-sct-is-unverifiable-never-not-logged.md)
§2 then rules what happens when a log is missing anyway: the result is `unverifiable`, **never**
`not-logged`.

So the one place a CT staleness could have reached a value already answers with a conservative
**value**, not with a `Gap`. Both candidate sites are closed, and they are closed on different
grounds.

### 5. The release is answerable, and an absence a release alone can close is the wrong instrument

ADR-0190 §2 rules the log list a **trust input** and §4 rules that refreshing it is a **release act**.
The list is a byte the image pins, covered by the release's signature and provenance. So a stale list
is a defect of the **image**, not of a measurement.

**This is the load-bearing half of the ruling, and it is a claim about what a `Gap` is for.** A `Gap`
closes when a measurement succeeds. Nothing the runtime can do closes this one. No cadence, no retry,
no operator toggle and no re-probe puts a log back in the snapshot. Only a new image does. An absence
record that only a release can close is a version nag wearing a measurement's clothes. It would sit
in the corpus every currency, coverage and retention read runs over.

The release already answers, in two places and at two audiences:

- **Before the image ships.** `TestShippedLogListOutlivesTheExpiryHorizon` fails the build 180 days
  before the newest `end_exclusive`. It runs in the required `test` check. The horizon doubles
  `release-pipeline.md` §10.7's 90-day cadence floor, so a missed refresh keeps a cycle of slack.
- **After the image ships.** The Sources page's CT card states the version, the cut date, the
  selectable count and the expiry date. It raises a warn callout at zero. That is what an operator
  running an old image reads.

The build gate is what the project sees. The card is what the operator sees. **Between them the
staleness is legible before it empties.** That is what ADR-0190's Consequences bullet asked for, and
it reaches that reading without asserting a measurement nobody made.

### 6. ADR-0106 keeps its reach, and it was never the site an amendment would land on

#1513's title offers to amend ADR-0106. **No amendment is owed, and two separate facts say so.**

First, the answer is no, so ADR-0106's reach does not move.

Second, and worth stating because a later reader will otherwise repeat the search: **ADR-0106 does
not rule `ct-tail` at all.** It mints the bulk `ct` `Scan` over `crt.sh`. ADR-0190's own
Alternatives table says it in those words — *"ADR-0106 rules the bulk `ct` Scan over crt.sh. It
predates the tail, names no log list, and reads no logs directly."*
[`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.2 agrees: the tail *"shares nothing
with bulk `ct` except the CT theme."*

The rule that actually reaches `ct-tail` is **ADR-0096 §7's**, which binds any `Scan` over a source
that admits without observing. ADR-0106 restates it for `ct`, and ADR-0190's Consequences applies it
to `ct-tail` by name. So an amendment to ADR-0106 would have written the ruling on a document whose
subject is a different `Scan`. This ADR is the site instead, and §7 states the scope it claims.

### 7. The rule binds the class, not the CT log list alone

**A build-time input that ages out and empties a population opens no `Gap`, on any `Scan`.** §2's
ground is that the `Scan` holds no timeline for one. §5's ground is that the release answers for a
pinned byte. Neither ground mentions certificate transparency.

So a later pinned artefact inherits this without arguing it again. It also inherits the
**obligation** the rule rests on. A pinned input whose staleness can empty a population owes two
things. It owes a build gate that fires before the emptying. It owes an operator surface that states
the pin's age. #1434 built both for the log list. A future pin that ships neither falls outside this ADR — it
is a defect this ADR names in advance.

### 8. What this rule does not reach

- **A `Scan` that holds a timeline.** The ruling turns on §2. A `Scan` whose source observes has
  cells, and a currency bound over them is ADR-0007's subject and not this document's.
- **ADR-0190 §7's refusal of the live-refresh path.** Untouched. This ruling licenses no runtime
  fetch, and nothing in it depends on one.
- **The `'skipped'` `dispatch.status` token.** #1120 minted it for the ADR-0137 §4 cadence-lag gate
  alone. §1 cites its ground and claims none of its machinery.
- **Whether the `ct-tail` source should ship on.** It ships off. That is ADR-0003 consent and
  [ADR-0189](./0189-a-ct-sources-go-constant-is-the-one-slug-literal-and-the-catalogue-names-the-constant-rather-than-repeating-it.md)'s
  slug, and neither moves here.
- **The 180-day horizon and the 90-day floor.** They are #1434's numbers, recorded in
  `release-pipeline.md` §10.7 and in #1521's amendment to ADR-0190. This ADR cites them and sets
  neither.

### 9. What would reopen this

Two changes would, and naming them is cheaper than leaving them implicit.

- **A durable CT-fed facet or an alertable issuance Signal.** §4.1 of the spec keeps the issuance
  event ephemeral precisely to avoid minting one, and ADR-0027 refused a CT-fed facet for v1 with no
  deadline. Build one and the tail acquires a timeline. §2's ground falls, a denominator exists, and
  the currency question becomes answerable rather than empty.
- **An admitting source whose `completeness` is `enumerable`.** §3's ground falls there, because such
  a source's silence can contradict. Nothing of that shape ships, and CT cannot become one.

Neither is a deferral of this ruling. Each is a different question that this ruling would no longer
cover.

## Consequences

- **This ADR changes no Go code, no SQL and no template.** The shipped behaviour is already the ruled
  behaviour. The gap was that the ruling lived only in a warn callout's prose.
- **`design-system/templates/settings.tmpl`'s callout is true by ruling.** Its sentence *"It is not
  idle, and no gap opens for it"* now cites a document. A later session that reads the callout and
  goes looking for the rule finds this file.
- **#1521's amendment to ADR-0190 is discharged at its fourth item.** That item states *"a zero
  selectable count still opens no `Gap`, and the decision is #1513."* This is that decision. The
  amendment needs no edit, because it stated the pending question correctly.
- **ADR-0106, ADR-0206 and ADR-0190 are confirmed and none is narrowed.** ADR-0206 §1 covers the
  zero-log tick as it always did, and §5's reserved question is now answered rather than reopened.
- **`CONTEXT.md` gains nothing.** `Gap`, `Scan` and `Batch` carry the terms this rule uses and none of
  them changes meaning. `Gap`'s entry already rules it a `Span` on a live subject's timeline, which is
  the whole of §2.
- **A stale-pin tick and a source-off tick still render identically in the `dispatch` table.** Both
  read `fanned-out` with zero jobs through the `LEFT JOIN`. The Sources card is where the two part,
  and §1 rules that correct: the cause is a release-side fact, and the card carries it. **This ADR
  opens no ticket to move that distinction into the `dispatch` row.**
- **The rule is held by review, not by a gate.** No test asserts that a zero-log `ct-tail` tick writes
  no `Gap`, because no code path could write one. ADR-0206's parallel consequence says the same of
  its nine fan-outs.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Open a `Gap` on the `ct-tail` `Scan` when `SelectTailLogs` returns zero** — #1513's proposal, taken literally | §2. A `Gap` is a `Span` on a timeline (ADR-0072) and the tail holds none. There is no cell to put it in and no denominator to name. The proposal cannot be built without first building something else |
| **Mint a CT-fed facet or a per-`Name` observability timeline, so the `Gap` has somewhere to sit** | ADR-0027 refused a CT-fed facet for v1, and the spec keeps the issuance event ephemeral for the same reason. It fabricates a measurement in order to record that a measurement is missing, which is ADR-0206 §3's fabrication with an extra step |
| **Give `ct-tail` a currency bound, so the silence eventually rots to a `Gap`** | ADR-0096 §7: a `Scan` over a source that admits without observing carries no currency bound, because the currency rule quantifies over observations and there are none. The bound has an empty domain. Nothing about a stale pin fills that domain |
| **Dead-letter a `Batch` with an empty scope on every zero-log tick, reusing ADR-0005's machinery** | ADR-0206 §4 keeps the failure path and the success path apart, and its Alternatives table measures the cost: at `ct-tail`'s 300-second cadence this manufactures 288 rows a day asserting a completed measurement nothing performed. `dead-lettered` also names an instrument or target fault, and a stale pin is neither |
| **Return an error from `fanOutCTTail` when the selection is empty** | ADR-0206 §1 draws the line between *we could not look* and *we looked and there was nothing*, and its Alternatives table measures what inverting it costs. It also mis-files the fault: the read succeeded, and the defect is in the image the read ran against |
| **Add a `dispatch.status` token for a population emptied by a stale pin** | ADR-0206's Alternatives table refuses a status for the empty population, on the ground that the `LEFT JOIN` already derives the count. A stale pin's cause is a **release-side** fact, legible on the Sources card and in a blocking test, and it is not a dispatch-side one. A token would put a build-time property on a runtime row |
| **Amend ADR-0106's reach, as #1513's title offers** | §6. The answer is no, so the reach does not move — and ADR-0106 rules the bulk `ct` `Scan`, not `ct-tail`. ADR-0190's own Alternatives table and `ct-source-replacement.md` §4.2 both say the two share nothing but the theme |
| **Rule it in ADR-0190 instead, beside the decay table that measured the problem** | ADR-0190's subject is the artefact's **provenance** — where the bytes come from and what they contain. Whether an absence record is owed is a question about the drift model, and #1521 already owns ADR-0190's amendment. Two tickets editing one file in one week is the collision ADR-0058's dated-record discipline exists to avoid |
| **Close #1513 with a comment and write no document** | The warn callout already asserts *"no gap opens for it"* with nothing on disk behind it. A comment on a closed issue is not where a later session looks, and #1521's amendment explicitly forwards the question here. The question would arrive again at the next pinned artefact, which is what §7 exists to prevent |
