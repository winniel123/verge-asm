# ADR-0179: an absent age sorts as a sentinel beyond every real age, never as zero

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1363 ADR gaps: cmd/web messages.go, restore.go and drift.go](https://github.com/winniel123/verge-asm/issues/1363), gap 3
- **Sweep PR that rewrote the comment:** [#1365](https://github.com/winniel123/verge-asm/pull/1365)
- **Rests on:** [ADR-0158](./0158-a-read-only-console-screen-may-scope-its-rendered-rows-in-the-client-and-a-screen-that-submits-a-form-carries-its-scope-in-the-query-string.md), which settles **where** a console sort may run. This ADR rules the **value** the server hands that sort, and it binds identically whether the comparator is JavaScript or Go
- **Not bound by:** [ADR-0081](./0081-a-floor-is-territory-and-an-unbounded-default-is-a-position.md), whose refusal of a sentinel — *"a sentinel is what a program writes when it has no answer, and the operator correctly infers that nobody decided"* (`0081:82-83`) — is about a number the operator **authors and reads** in a retention dial. A sort key is authored by nobody and rendered to nobody
- **Not bound by:** [ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md) decision 4 and [ADR-0158](./0158-a-read-only-console-screen-may-scope-its-rendered-rows-in-the-client-and-a-screen-that-submits-a-form-carries-its-scope-in-the-query-string.md) §3, which forbid a hidden per-row datum from becoming a **filter's** carrier. A sort hides no row, so a hidden datum is a legal sort key

## Context

The console has two age-ordered lists, and their sort keys are built the same way and read in two
different languages.

**Reports.** `reportScheduleRow.LastMins` (`cmd/web/messages.go:356`) is whole minutes since a
schedule's last delivery. `reportScheduleRows` fills it. A schedule with no delivery row gets
`reportScheduleNeverRunMins`, declared `1 << 30` at `cmd/web/messages.go:389` and assigned at
`:392` beside the em dash the Last-sent cell renders (`:391`). A real delivery overwrites it at
`:399`. The template renders it into a hidden attribute — `data-last="{{.LastMins}}"`
(`design-system/templates/reports.tmpl:256`) — and the header button at `:251` drives a comparator
at `:346` that subtracts one `data-last` from another.

**Signals.** `signalRow.seenAge` (`cmd/web/signals.go:72`) is whole minutes since a signal was last
seen. `seenAgeMinutes` (`:685-698`) returns `1 << 62` for an empty timestamp (`:687`) and for an
unparseable one (`:691`). `sortSignalRows` compares it at `:621-628`, server-side, because Signals
submits forms and therefore carries its sort in the query string (`cmd/web/signals.go:198`, under
ADR-0158 §1).

So the same decision is made twice, in two files, in two languages, on two sides of the
client/server line ADR-0158 draws — and it was written down nowhere. #1365 swept the two
declaration comments that carried it into one uncited line at `cmd/web/messages.go:390`:

```go
// A never-run schedule sorts last under the client-side sort, so the sentinel exceeds any age.
```

That line is true, but it is a comment about one field. It cannot bind `signals.go`, it cannot bind
the comparator in `reports.tmpl`, and neither side can be read alone: `messages.go` cannot show that
a larger number sorts later, and `reports.tmpl` cannot show that `1073741824` means *never*.

**The line numbers in #1363 are stale.** It cites `messages.go:492-494` and `:554-559`, read
against `f9bc284`; the true sites are `:356` and `:389-392`. It cites `reports.tmpl:279` and `:369`;
the true sites are `:256` and `:346`.

## Decision

> **Where a numeric sort key stands for an age and the event it measures has never happened, the key
> carries a sentinel larger than every representable real age, so the row sorts last. It never
> carries zero. The sentinel is one named constant, and no other site spells its value.**

### 1. Zero is the wrong end, and it is the wrong end for the rows that matter most

Zero minutes means *the event happened now*. Ascending by age puts it first. A never-run schedule
would therefore land at the top of the same list a recently-delivered one heads, and the operator
who sorted by *Last sent* to find the freshest delivery would find a schedule that has never
delivered instead.

*Never delivered* and *delivered a moment ago* are the two extremes of the column, and zero
collapses one onto the other. It is silently wrong: the cell reads `—`, so nothing on screen
contradicts the position the row took.

The sentinel instead reaches absence deliberately. One click sorts ascending, a second flips
`state.dir` (`reports.tmpl:343`), and the never-run rows head the list.

### 2. The comparator learns nothing, and that is the point

`reports.tmpl:346` is `((+a.data-last) - (+b.data-last)) * state.dir`. It knows nothing about
schedules, deliveries or absence. `sortSignalRows` at `signals.go:621-628` is the same three-way
compare on an `int64`.

A comparator that special-cases absence must be taught the encoding, so the encoding is spelled at
two sites and can drift at one. Keeping absence inside the ordinary numeric domain keeps the
comparator total and the rule in one file. **The sentinel is a fact about the producer, never about
the comparator.**

That is why the ruling is on the value and not on the sort. ADR-0158 lets Reports hold its sort in
the client and requires Signals to hold its in the query string; under this ADR both build the key
the same way, and moving a screen across ADR-0158's line changes nothing here.

### 3. Direction is confirmed against what the operator sees, not against the array

`rows.sort` orders ascending on a negative return, and `reports.tmpl:349` appends the sorted array
into the `tbody` in order, so array index 0 is the visible top row. With `state.dir === 1` — the
first click on a fresh key (`:343`) — a larger `data-last` returns positive and lands later.
**Larger sorts lower.** The sentinel row is last. Server-side, `sortSignalRows` reverses its `less`
for `dir == "desc"` (`signals.go:645-650`) and is otherwise ascending, so the same holds. Two sign
flips sit between the number and the row, so *sorts last* is checked at the rendered order.

### 4. The sentinel is unreachable, and it is unreachable by construction

`1 << 30` is 1,073,741,824 minutes, about **2,041 years**. That alone is an argument from
plausibility. The hard bound is `time.Duration`: `now.Sub(inst)` saturates at `math.MaxInt64`
nanoseconds, which is 292.5 years, or **153,722,867 minutes**. No value `messages.go:398` can
compute reaches within a factor of six of the sentinel, whatever a row's timestamp holds, so a real
age can never collide with it. The same argument covers `1 << 62` in `signals.go` with far more
room. `1 << 30` also fits in an `int32`, so the constant is safe in the `int` field at `:356` on any
platform, and it survives the decimal rendering into a JavaScript `Number` exactly.

**A sentinel chosen this way is a claim that needs the arithmetic written down.** A later session
moving `LastMins` to seconds divides the headroom by sixty and must redo it.

### 5. The key and the rendered cell state the same thing

`messages.go:391-392` sets the em dash and the sentinel in one statement pair, because they are one
fact. A row whose cell says a delivery happened and whose key says none did cannot be reconciled,
and the disagreement is invisible until the operator sorts.

Any age the producer cannot express as a real age is clamped to zero, not folded into the sentinel.
`seenAgeMinutes` does this at `signals.go:694-696`, and so does `relTime` at
`cmd/web/messages.go:250-252`. `reportScheduleRows` does not: its `m >= 0` guard at `:398` leaves a
future-dated delivery on the sentinel. **The contradiction is two lines apart in one function.**
`:397` renders the cell through `relTime`, which clamps the same negative duration and returns
`"now"` (`:250-255`); `:398` refuses it and leaves `lastMins` at `reportScheduleNeverRunMins`. The
row then reads *now* and sorts *never*. That is a defect against this limb.

## Consequences

- **`cmd/web/messages.go:390` gains a citation** and keeps its clause. The rule now has a document,
  so the comment is repairable under `comment-policy.md` §4.7 rather than uncited.
- **`cmd/web/signals.go:687` and `:691` become a defect** against the naming half: one value, two
  bare literals, no name. The fix is a hoisted constant; the ordering is already right.
- **`reportScheduleNeverRunMins` is declared inside the `for` body** (`messages.go:389`), so it is
  scoped to one iteration and no test and no other function can name it. A named constant nothing
  can reference does half the job this ADR asks of it. It moves to package scope.
- **`cmd/web/messages.go:398-400` becomes a defect** against §5. Its negative-age fall-through
  disagrees with the `relTime` call one line above it, and with `seenAgeMinutes`.
- **A new age-ordered column states its sentinel and its headroom.** The arithmetic in §4 is the
  work a reviewer checks, and it is cheap once.
- **No production behaviour changes on the Reports path.** The ordering shipped is the ordering this
  ADR rules. Only the representation defects above move.
- **`CONTEXT.md` gains nothing.** *Sort key* and *sentinel* are console-shell terms, not
  product-domain terms.
- **`parseSigNum` (`cmd/web/signals.go:677-682`) is out of scope.** It returns `0` for an
  unparseable id, but an id is an identity, not an age, and there is no ordering claim that a
  missing id is older or newer than a present one.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Zero for absent** | It inverts the column for exactly the rows an operator is hunting. Zero already means *just now*; reusing it for *never* makes one number carry two facts that sit at opposite ends of the axis, and no cell on screen exposes the collision. It also fails quietly in the other direction: sorted descending, the never-run rows fall to the bottom where they read as the stalest — accidentally right, for the wrong reason, so the next reader keeps it |
| **A nullable field the comparator special-cases** — `*int` in Go, an omitted `data-last` in the DOM | It spells the encoding twice, and one of the two sites fails open. `reports.tmpl:346` coerces with unary `+`, and both `+null` and `+""` are `0`, so a comparator that forgets the case silently implements the zero row above. It also makes the comparator partial: `signals.go:621-628` gains two branches, and a third caller may invent a third convention. One constant in one file is checkable; a convention spread over every comparator is not |
| **A separate boolean column plus a two-key sort** — `HasDelivery` first, `LastMins` second | Two fields that must be read together to know one thing, with an unrepresentable state (`HasDelivery` false beside a real `LastMins`) the type cannot forbid. Both comparators become composite, and the DOM carries a second attribute whose only job is to say the first is meaningless. `HasDelivery` (`messages.go:358`) already serves the disabled *View last delivery* menu item, so this also overloads a live field with a sort obligation |
| **Sort server-side and drop the client comparator** | ADR-0158 §1 permits it — Reports submits no form for this table — but it costs a round trip per sort click on a table the server already rendered whole, and it does not answer this ADR's question. The server would still choose a value for a never-run schedule, and choosing zero there is the same defect one layer down |
| **Sort the rendered string lexically and drop `LastMins`** | `"—"` has no defensible neighbour among `"now"`, `"4m"`, `"3d"`, and those strings do not order by age in any case. It also makes `relTime` (`messages.go:248-262`) load-bearing for ordering. `LastMins` exists beside `LastSent` precisely because the string is for reading and the number is for ordering |
