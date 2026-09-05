# ADR-0182: a progress event reaches a live viewer as a new appended line, and the stream cursor is composite

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1374 ADR gaps: cmd/web production 16/17 (#1217)](https://github.com/winniel123/verge-asm/issues/1374), gap 2
- **Sweep PR that deleted the comment:** [#1375](https://github.com/winniel123/verge-asm/pull/1375)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0131](./0131-the-console-is-vanilla-server-rendered-prg-and-the-htmx-stack-is-withdrawn.md) §1, which ratifies hand-rolled inline vanilla JavaScript and refuses a fetch/swap layer, and so licenses the poll client this ADR must live with; [ADR-0005](./0005-scan-execution-model.md), which commits a job's outcome with its observations and so puts the terminal write on the `queue_job` row
- **Read with:** [`raw-job-output.md`](../spec/raw-job-output.md) §6.2, which rules the raw admin view post-hoc only and states that this stream persists nothing at rest
- **Not bound by:** [ADR-0165](./0165-a-recorded-dispatch-disposition-overrides-the-live-status-derivation-and-the-run-pages-status-word-is-one-token-that-styles-and-labels-the-badge.md), which rules the run page's status **word**. This ADR rules the run page's log **stream**; the badge is a page-render value and never travels on the stream

## Context

`cmd/web/progress.go` carried an eight-line block above `eventStreamLines` stating why a progress
event is appended rather than merged and why the cursor packs two counters. #1375 deleted it as
uncited. One sentence survives at [`progress.go:94`](../../cmd/web/progress.go), uncited under
[`comment-policy.md`](../spec/comment-policy.md) §4.7 route 3 because no document states the rule.

**The client is append-only, and that is a constraint, not a decision.**
[`design-system/templates/rundetail.tmpl:213`](../../design-system/templates/rundetail.tmpl) sets
`cursor = body.querySelectorAll(".rd-line").length` — the count of lines the server already
rendered inside `.rd-logbody` (`:144-148`). `:217-223` builds each incoming line and calls
`body.appendChild(row)`. There is no code path that reaches an existing `.rd-line`, and no line
carries an identifier a later poll could address. The client only grows the log at its tail.

**Gate A is a judgement call here, and the gap is real.** §8.2 gate A says *"a hazard, a cost note,
or an external constraint survives silently and opens nothing"*, and the append-only client is
exactly such a constraint. The **response** to it is not: a single monotonic counter was the obvious
alternative and works for a run that never retries, so packing two counters into one integer is a
chosen design with a chosen cost. Gates B and C both hold — the contract binds `internal/queue`'s
emitter, `cmd/web`'s stream endpoint and the template's client, and none of the three is a
`_test.go` file.

**A terminal write mutates the `queue_job` row in place.** `markDone`, `markDead` and `markRetried`
are `UPDATE queue_job SET state = …` at [`db/queries/measurement.sql:114`](../../db/queries/measurement.sql),
`:117` and `:120`, called from [`internal/queue/worker.go:454`](../../internal/queue/worker.go),
`:486` and `:513`. The state log is one line per row
([`cmd/web/scans.go:722-742`](../../cmd/web/scans.go)), so a dead-letter changes the *text* of a
line the viewer already holds and can never be told about. The reason — `dead-lettered after 5
attempts · crt.sh returned HTTP 502` (`internal/queue/progress.go:44-46`) — arrives as its own line
or not at all.

**Where the rule used to be stated, and is not now.** #1374 recorded that `rundetail.tmpl:7` said
*"appends live, auto-follows"*. That site is gone — the D3 sweep #1396 deleted it, and line 7 today
is a CSS rule. The five-place search over `docs/spec/`, `docs/adr/`, `docs/research/`,
`docs/guides/` and `CONTEXT.md` returns no hit for `streamCursorBase` or for the packing.

**What `raw-job-output.md` §6.2 settles, and what it leaves open.** It settles the *corpus*
question for the raw admin view: capture is written in the job's terminal transaction, this stream
is an in-memory hub, and so raw bytes are post-hoc only and the two surfaces never share a panel. It
says nothing about how a line reaches this surface — delivery shape, merge-versus-append and cursor
encoding are all outside it. #1420's bounding note at `:171-175`, which scopes a mid-flight cancel's
rollback to the attempt's staged work, does not reach here either.

## Decision

> **A per-job progress event reaches a live viewer as a NEW appended line, never as a reason merged
> onto a line already rendered. The stream cursor is composite: two independent counters packed
> into one integer, with the state-line count as the low part, so the state lines and the event
> lines advance separately and a retry that grows the state log cannot shift an event's position.**

### 1. An event is a line, and the merge is forbidden

`eventStreamLines` ([`progress.go:93-107`](../../cmd/web/progress.go)) turns each `jobProgress` into
its own `runStreamLine`, tagged `#<job id>` and carrying its own level. It never consults the state
log and never rewrites one of its entries.

The prohibition is not stylistic. The client has no addressing scheme for a rendered line, so a
merged reason reaches a viewer only on the next full page render — the one case the stream exists
to avoid. Work that wants to enrich a state line emits a new event instead.

### 2. The counters are independent because the state log is not monotonic

The state log is **derived on every poll** from the current `queue_job` rows (`deriveRunStream`,
`scans.go:505-553`). A retry enqueues a fresh job (`worker.go:494-505`), so the row count grows
mid-run. The event log is genuinely append-only: `progressHub.record` appends and never evicts the
front, precisely so an index stays valid (`progress.go:55-59`).

One counter over a concatenation of the two would move an event's index whenever a retry lands, and
the client would either re-append lines it holds or skip lines it does not. Two counters make each
index an index into its own stable sequence.

### 3. State is the low part, and that ordering is forced

`encodeStreamCursor(events, state)` returns `events*streamCursorBase + state` with
`streamCursorBase = 1_000_000` (`progress.go:109-117`). The order is not free: the client's *first*
cursor is a bare rendered-line count (`rundetail.tmpl:213`), a state count with no event part, and
only with state in the low part does that integer decode to `(0, n)`. `scans.go:566` states this at
the decode site. It is also why `encodeStreamCursor(0, 7) == 7` — a run with no events returns
exactly the state-line count, the transport contract that existed before events did.

Within one response, state lines precede event lines (`scans.go:584-590`) and the client appends in
array order, so ordering is stable across polls.

### 4. The ceilings are asymmetric, and only one of them is reachable

The event counter is bounded at `maxEventsPerRun = 1024` (`progress.go:31`, enforced at `:56-58`).
The state counter is clamped at `streamCursorBase - 1 = 999,999` (`progress.go:113-115`). The
largest cursor the wire can carry is therefore `1024 * 1_000_000 + 999_999 = 1,024,999,999` — well
inside `int32` and far inside JavaScript's safe-integer range, so the echo at `rundetail.tmpl:233`
and the `strconv.Atoi` at `scans.go:563` are both safe.

**The state ceiling is reachable and the clamp is not a fix.** One state line is one `queue_job`
row, a `hot` `Scan` fans out one job per `(Vantage, Address)` pair (`CONTEXT.md:453-454`,
`internal/scan/hot.go:50-55`), and
[ADR-0127](./0127-the-address-scope-range-cap-has-no-ceiling-a-large-scope-is-priced-not-gated.md)
rules the address-scope cap has **no upper bound**. At a million rows the clamp makes
`len(state) > stateCur` (`scans.go:583`) permanently true and the endpoint re-serves the tail of the
state log once a second forever. **The ruling is that the served slice and the cursor must agree**:
where the state log exceeds what the cursor can address, the stream serves the addressable prefix
and stops, rather than reporting a position it cannot reach.

## Consequences

- **The reason and the state are two lines.** A dead-lettered job shows `#900 ct · dead` from the
  state log and `#900 dead-lettered after 5 attempts · …` from the hub. The duplication is the
  price of the append-only client, paid once per terminal event.
- **A run with no jobs never streams.** `.rd-logbody` renders only under `{{if .Log}}`
  (`rundetail.tmpl:143`), so the script's `body` is null and it returns at `:212`. That follows from
  §3: with no rendered lines there is no cursor origin.
- **The state log stays bare.** `runLog` may not be enriched from the hub; `TestPageLogStaysBareState`
  (`cmd/web/progress_test.go:170-195`) pins that, and this ADR states why.
- **Three survivors gain a citation** and the deleted block does not come back.
- **A defect is open against §4.** `deriveRunStream` does not truncate the state log to
  `streamCursorBase - 1`, so the ceiling case livelocks instead of stopping. It is reported to the
  ticket, not fixed here.
- **A test is owed.** No test grows the state log while events are present:
  `TestRunStreamAdvancesAndConcludes` grows state with no hub attached, and
  `TestRunStreamEnrichedByHub` attaches a hub and never grows state, so §2 is unpinned.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A single monotonic counter** over the concatenation of state and event lines | The state log is re-derived per poll and grows on a retry (`worker.go:494-505`), so an event's index in the concatenation moves under the viewer. The client, whose cursor is a plain integer, then re-appends lines it holds or skips lines it never received. The bug appears only on a run that retries — the run an operator is most likely to be watching |
| **Merge the reason onto the existing line and have the client re-render it** | Unimplementable against the shipped client and expensive to make implementable. Nothing gives a rendered `.rd-line` an identity — `rundetail.tmpl:146` emits no key — so the client would need a line index, a lookup and an in-place mutation path. ADR-0131 §1 refuses a fetch/swap layer for exactly this cost, and the merge re-enters the frozen LogViewer surface `raw-job-output.md` §6.1 requires be left untouched |
| **An opaque string cursor** (`"e3:s17"`, or a base64 token) | It breaks the pre-existing transport contract the client depends on: the first cursor is a rendered-line count the browser computes from the DOM (`rundetail.tmpl:213`), and a string cursor has no origin the client can construct without a second server round-trip. It also adds a parser and a validation path on an untrusted query parameter, where the integer form fails closed at `strconv.Atoi` (`scans.go:563`) |
| **Swap the packing — events low, state high** | It removes the reachable clamp, since events are hard-capped at 1024 (`progress.go:31`). It also destroys the origin: an initial cursor of `n` rendered state lines decodes as `(state 0, events n)`, so the client re-receives the whole state log and skips the first `n` events. The §3 compatibility limb is worth more than the headroom |
| **Persist the events and read them back beside the state log** | It turns one derived read of `queue_job` into two reads that must agree, and it contradicts `raw-job-output.md` §6.2's *"persists nothing at rest"*, which is what keeps this surface out of the retention model ADR-0126 prices for `Transcript` |
