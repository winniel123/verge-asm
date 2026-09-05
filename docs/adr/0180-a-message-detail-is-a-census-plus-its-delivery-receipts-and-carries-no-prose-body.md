# ADR-0180: a message detail is a census plus its delivery receipts, and carries no prose body

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1333 ADR gaps: cmd/web/devfixtures.go](https://github.com/winniel123/verge-asm/issues/1333), gap 3
- **Sweep PR that deleted the comment:** [#1335](https://github.com/winniel123/verge-asm/pull/1335)
- **Rests on:** [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md), which fixes a
  `Message` as **one sentence** under one grammar — *what moved · what it now is · what we counted* —
  computed once at the cause, and [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md),
  which makes the message the surface a failed `Delivery` is seen on
- **Narrows:** [ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md),
  whose delivery limb (`:121-124`) already rules the failed receipt — surfaced on *"the **Message it
  failed to carry**"*, reading *"could not deliver, not … nothing fired"*, *"with the reason as a
  drill-down"*. That is one of the four parts below. This ADR fixes the other three and the closure
- **Not bound by:** [ADR-0114](./0114-the-report-pdf-is-rendered-in-process-from-the-artifact-not-from-html.md),
  which rules the render form of the **report `Artifact`** and reaches no `Message`; and
  [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md), which rules a rule's predicate
  domain and touches messages only to say a domain move needs none (`:369-371`)

## Context

`cmd/web/devfixtures.go:2237` carries the surviving statement of this rule, uncited:

> `// The message detail carries no prose body: the form is the census plus the delivery receipts.`

Its pre-sweep citation was `SPEC-CHANGE #24`. `design-system/SPEC-CHANGE.md` is not on disk, and
ADR-0116's status line withdrew the collision protocol that minted those numbers. The retired
protocol left one other trace on disk: `design-system/fixtures/fixtures.json:3422` still carries
`"$note": "SPEC-CHANGE #24: no .Body hole — the detail form is census + deliveries (ADR-0064 stays
intact; no prose producer)."` The rule is therefore recorded twice and ruled nowhere — comment-policy
§8.3 shape 2.

**The rule is true of the code today, on both surfaces, and nothing on the path could make it false.**

`messageRow` (`cmd/web/messages.go:33-45`) holds `ID`, `Cause`, `Class`, `Headline`, `Href`,
`LinkText`, `Instant`, `Read`, `Census`, `Deliveries`, `AnyUndelivered`. `inboxView`
(`:137-142`) adds `Rel`, `JumpLabel`, `Selected`. There is **no body field, and no field it could
hide in**: `Census` is `[]censusRowView{Kind, Key, Href}` (`:47-51`) and `Deliveries` is
`[]deliveryView{ChannelHost, Class, When, State, Failed, LastError}` (`:21-31`).

The model behind it is the same shape. `message.Message` (`internal/message/message.go:73-91`)
carries `Cause`, `Class`, `SubjectKind`, `FiredAt`, `Instant`, `Census`, `Headline`, `Read`. The
table behind **that** is `db/migrations/20500_message.sql:15-51`: eight columns, of which `headline`
(`:43`) is the only text the message says, described there as *"the rendered sentence, computed once
at the cause"*. A `CensusEntry` is `{Kind, Key}` and nothing else (`internal/message/census.go:14-17`).
`CONTEXT.md:1791-1793` enumerates what a `Message` carries — class, key, instant, census, read-state,
delivery outcomes — and names no body.

**The one place the corpus produces message-adjacent prose, it drops it before the message.**
`narrowingLoss` (`internal/message/render.go:90-94`) names what a narrowing loses; it rides the
receipt (`internal/message/narrowing.go:28`) onto the **preview** screens
(`design-system/templates/scope.tmpl:182`, `:421`), and `Narrowing` then builds the `Message` from
the headline alone (`:57-64`). The prose exists at the act, where the operator can still choose. It
does not travel to the record.

## Decision

> **A message detail renders the census — the kind and the linked mono key of each thing that moved
> — and the delivery receipts, and nothing else that is prose. There is no body field on the model,
> the row, the template or the wire, and none may be added. A failed receipt is flagged
> `undelivered`, names its channel host, and carries its `last_error` as a drill-down.**

### 1. The four parts of the detail, and they are closed

`design-system/templates/inbox.tmpl:99-142` is the whole detail pane:

| Part | Where | What it is |
| --- | --- | --- |
| **Identity** | `:103-104`, `:106`, `:110` | class, the headline sentence, relative and absolute instant |
| **Census** | `:111-120` | one row per entry: `Kind` as a micro label, `Key` in mono, linked to the subject where `subjectHref` resolves one (`cmd/web/messages.go:333-344`) |
| **Receipts** | `:121-134` | one row per `Delivery` |
| **Egress** | `:135-139` | the jump button to what fired, and mark-unread |

The headline is not a body. It is ADR-0064's **one sentence**, computed once at the cause and stored
verbatim, and the store never recomputes it. A body would be a second rendering of the same fold,
authored at a different instant, versioned by nothing.

### 2. The census is the message's content, and a body would duplicate it

ADR-0064 rules that a message **names what moved**, read from the fold. The census **is** that
reading, enumerated and never sampled, ranked or truncated (`internal/message/census.go:19`). A body
could say only one of two things. It could restate the census — a second, unversioned rendering of a
fact the census already carries exactly, and the two can disagree, which is the failure mode
`20500_message.sql:6-9` exists to forbid one object across. Or it could say something the census does
not, and then a fact about the estate lives outside the timelines and the comparison path's fence
(`CONTEXT.md:1798-1799`) is a fence around nothing.

**Fixed copy that is a property of the form is not a body.** `inbox.tmpl:127` carries the clause
*"this message could not be delivered, not that nothing fired"*. It is identical on every message,
it restates ADR-0039's *a dead-lettered `Delivery` licenses no silence* at the site where the
operator would otherwise misread the flag, and it is not computed from the fold. Template copy is
allowed. Per-message text is not.

### 3. A failed receipt, exactly

`toDeliveryView` (`cmd/web/messages.go:302-316`) sets `Failed` from `State == "undelivered"`, copies
`LastError` **only** when `Failed` so a delivered receipt cannot leak an error string, and derives
`ChannelHost` from the URL's host alone — never the raw URL, which may carry the operator's embedded
token.

The template renders a danger pill reading `undelivered` (`inbox.tmpl:126`), then `to <host>`, then
the fixed clause, then — where `LastError` is present — the word `(reason)` carrying the error in a
`title` attribute (`:127`) under a dashed underline (`:47`). **That is what "drill-down" resolves to
here: a hover disclosure on one word, not a panel and not a log line.** `AnyUndelivered`
(`cmd/web/messages.go:82-86`) rolls the fact up for the list.

### 4. The rule binds both message surfaces, and the wire

The `/messages` fold in `design-system/templates/settings.tmpl:1430-1458` renders the identical
form — cause, class, instant, headline (`:1437`), census rows, receipts, link — with the same
undelivered markup at `:1447`. The outbound `Body` (`internal/delivery/delivery.go:41-49`) carries `message`,
`class`, `cause`, `subject`, `instant`, `headline`, a census **count** and a link. Three renderings
of one record, and no body in any of them.

### 5. Nothing in `internal/message` renders a `Message` at all

`internal/message/pdf.go` and `render.go:221-230` render the report `Artifact`, and every caller of
`RenderArtifact` / `RenderArtifactPDF` is a report path (`cmd/web/reports.go:766`,
`reports_export.go:84`, `reports_schedule.go:399`, `internal/report/dispatcher.go:106`). A `Message`
reaches neither, and the tree ships no email sender. **The PDF cannot disagree with this rule
because it is not about the same object.** The package name is the whole of the resemblance.

## Consequences

- **An operator who wants context follows the mono key.** The detail says a service was reached and
  which facets opened beneath it; *why that matters* lives on the subject page the census row links
  to. That is one click, and it is the cost.
- **There is no place to explain an unusual change inside the message.** A first expired certificate
  after a bad deploy reads with the same furniture as the hundredth routine one. **This is
  acceptable, and for the reason ADR-0064 §5 already gave**: a rendering that changes with the
  circumstances makes the message's content a function of how notable we judged it, and the judging
  is what this model refuses. A message records that the operator was told; the estate's own screens
  are where the estate is explained.
- **The detail cannot grow a field by accident.** Adding one costs a column on `message`, a field on
  `message.Message`, a field on `messageRow`, and markup — four deliberate edits, one a migration.
- **`docs/guides/notification-channels.md:239` and `:242-244` are wrong and are not repaired here.**
  They say the last error appears *"only as drill-down on the channel surface"*. ADR-0108 `:121-124`
  ruled the opposite and `inbox.tmpl:127` implements it. Under
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) the
  correction belongs at that site.
- **The two off-model records of this rule stay put.** `cmd/web/devfixtures.go:2237` and
  `design-system/fixtures/fixtures.json:3422` now have a document to cite.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A short generated prose summary** — expand the census into a sentence at render time | It is a second rendering of the fold, recomputed on every page load where the headline is computed once and stored (`20500_message.sql:40-42`). The two can disagree and the reader cannot tell which is the record. It also needs a vocabulary, and every word it could reach for that is not already in the census is one ADR-0064 §3 bans |
| **An optional operator-authored note on the message** | Operator-authored content in the alerting path — ADR-0039 §4's refusal, restated in ADR-0064's table. Worse here: a `Message` records *that the operator was told*, so an editable field makes the record of what we said mutable by the party we said it to. It is also new operator state on an object `CONTEXT.md:1792-1793` closes at read-state and delivery outcomes |
| **A prose body in the channel POST only**, leaving the console form alone | v1-spec §4.5 requires the body to carry *"exactly what the in-app message carries"*, read *"from the same computation that renders the in-app message"*. A body only on the wire is a second computation by construction, and it lands where nobody can check it against the store — the receiver sees a fuller message than the instance holds, and a support conversation then turns on which copy is the record |
| **Keep the rule as the comment it was**, uncited, and file nothing | It passes comment-policy §8.2's three gates squarely: the rule binds `cmd/web/messages.go`, both templates, `internal/message` and `internal/delivery`, and it survived only in a dev-fixtures helper. The next reader of `inbox.tmpl` finds a detail pane with no body and no reason for it, which is how a body gets added |
| **Fold the receipts onto the channel surface and leave the detail as census alone** | Reverses ADR-0039 `:179-180` and ADR-0108's delivery limb. An undelivered POST is a fact about *this* message; on the channel it becomes a configuration note about a host, and the operator reading the message has no way to know they are the only one who saw it |
| **Render the `last_error` inline rather than as a hover drill-down** | ADR-0108 named the drill-down specifically, on #22's *reason is a slot, not a design*. An inline TLS trace on every failed row makes the receipt unreadable at the moment the operator is scanning for which message did not get out |
