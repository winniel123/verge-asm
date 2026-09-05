# ADR-0185: a severity is the operator-facing grade, so it composes into no version vector and is not a fifth part of a rule

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1301 ADR gaps: internal/signal](https://github.com/winniel123/verge-asm/issues/1301), gap 1
- **PR that deleted the comment:** [#1302](https://github.com/winniel123/verge-asm/pull/1302)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Withdraws a clause of:** [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md). Its
  second guard reads *"a domain may cite only evidence the rule declares, and everything a rule
  declares composes into its version vector"*. The second half is withdrawn here and replaced. The
  first half stands, and so does the four-part count in the same ADR's Decision table
- **Rests on:** [ADR-0008](./0008-derivation-versions-move-on-content.md), whose Decision table fixes
  what moves a version — *"an **output-affecting change only** — never because a release shipped"* —
  and [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md), which states the same test
  from the other side: a part is a leaf *"exactly where its output can move while the world does
  not"*, and *"the vector is what decides comparability"*
- **Bounded by:** [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md).
  Its live remnant rules that a signal carries a five-level severity assigned per rule, and
  [`CONTEXT.md`](../../CONTEXT.md)'s `Signal` entry restates it and adds that the transition owns
  timing and not grade. That half is written. This ADR restates none of it and rules only the
  question ADR-0116 left open
- **Sibling of, and not ruled by:**
  [ADR-0151](./0151-a-field-no-emitter-renders-is-cross-leaf-plumbing-and-plumbing-moves-no-leaf-version.md).
  It applies the same output-affecting test one level down, to a field on a measurement leaf's
  result type. Its object is a leaf and a params digest. This ADR's object is a rule and a census.
  Neither contains the other
- **Sibling of, and not ruled by:** [ADR-0184](./0184-an-unknown-severity-token-folds-to-info-on-every-surface-and-no-surface-folds-it-differently.md). That ADR folds an unknown or stale severity token to `info` on every surface. This ADR is why such a token exists at all: a re-rating moves a rule's grade without moving its version vector, so a stored token can name a level the rule no longer carries

## Context

`internal/signal/severity.go:3` carried this, in declaration position on `type Severity string`,
until #1302 compressed it to two lines:

```go
// Severity is the five-level urgency ramp a `Signal` rule carries (P0.1;
// design-system/examples/console/SignalData.jsx `sev` / `SEV_ORDER`). It is a
// property of the RULE — assigned per rule and identical for every instance the
// rule raises — never a property of the subject or of the transition that
// surfaces it. A rule's severity moves with a deliberate re-rating, so it is not
// folded into the version vector (which tracks output-affecting evidence, not the
// operator-facing ramp): two censuses of the same rule at different severities are
// still comparable.
//
// This supersedes the old "a signal carries no severity" reading (the ruling in
// PARITY-CHART.md §"The ruling", collision #1 in SPEC-CHANGE.md): the design is
// normative for look AND functionality, and where the domain lacked the datum the
// fix is to build it, per rule, rather than drop the ramp.
```

**Most of that block is already written down, and this ADR does not restate it.** ADR-0116's
surviving bullet rules that a signal carries a five-level severity assigned per rule. `CONTEXT.md`'s
`Signal` entry carries the same rule, cites ADR-0116 for it, and adds that what the transition owns
is timing and not grade. The supersession the block's second paragraph describes is recorded at
ADR-0116, at `CONTEXT.md`, and — since [#1410](https://github.com/winniel123/verge-asm/issues/1410) —
at ADR-0024's own site.

**One clause is written nowhere:** the severity is not folded into the version vector, so two
censuses of one rule at different severities stay comparable. `version vector` appears in nine files
under `docs/` and in `CONTEXT.md` once. No occurrence names a severity.

**The two documents the block cited for its own authority do not exist.** `PARITY-CHART.md` and
`SPEC-CHANGE.md` are in no path in the tree. Both retired with the design-system handoff workflow on
2026-08-28. The comment was the surviving statement of its own second paragraph, and #1302 deleted
it.

### The naive reading of ADR-0024 folds the grade in, and it is the reading on the page

ADR-0024 states the vector's reach in one sentence:

> The second guard is narrower and stops the first being evaded: **a domain may cite only evidence
> the rule declares**, and everything a rule declares composes into its version vector.

A severity is something a rule declares. `Severity()` sits on the `Rule` interface at
`internal/signal/signal.go:79`, between `Version()` and `Eval()`, and every one of the seventeen
rules answers it. Read literally, ADR-0024 puts the grade in the vector and makes a re-rating a
`Break` on every timeline the rule feeds.

### The code already answers the other way, in three places

| Where | What it shows |
| --- | --- |
| `internal/signal/endpoint.go:113` | `certDetailRule` holds `name`, `sev` and `pick` in one struct. `Version()` returns `certVersion()` and reads no field of the receiver. Four of the five certificate-detail rules differ in `sev` and share one vector |
| `internal/signal/severity.go:30` | `SeverityFor` resolves a grade from a rule name alone. It never reads a `Version`, and no caller passes one |
| `db/migrations/22300_signal_instance.sql:11` | *"the severity is the rule's (assigned per rule in `internal/signal`, not stored)"*. Nothing anywhere persists a severity. `signal_instance` holds an id, a name, a subject and a first-seen instant, and no column carries a grade |

So the exclusion is already the behaviour. What was missing is the ruling, and with it the reason a
future author may not fold the grade in for symmetry.

### What folding it in would cost, and what it would buy

The seventeen shipped grades, read from the three registries:

| Grade | Rules |
| --- | --- |
| `critical` | `sensitive-port-reached-from-internet`, `certificate-expired` |
| `high` | `cname-target-name-error`, `certificate-not-yet-valid`, `certificate-weak-key-or-signature`, `certificate-hostname-san-mismatch`, `unauthenticated-request-answered` |
| `medium` | `lame-delegation`, `zone-declared-name-returns-name-error`, `non-globally-reachable-address-resolved-from-internet`, `tls-1.0-accepted`, `certificate-expiring`, `certificate-self-signed`, `plaintext-http-no-https`, `redirect-to-host-outside-estate` |
| `low` | `resolved-name-absent-from-zone`, `redirect-does-not-upgrade-to-tls` |

**The cost is a `Break` per re-rating and it is charged to the whole ramp.** A grade is the one part
of a rule an operator argues with. Two of seventeen rules are `critical`, and any pressure to move a
rule up or down the ramp lands on a boundary between two adjacent bands. Under the folded reading a
single band move on one rule makes every census that rule produced before the move incomparable with
every census it produces after, under ADR-0008's `Break`, for one cadence.

**The gain is nothing.** No member changes register. No subject enters or leaves the domain. No
predicate answers differently. `Evaluate` at `internal/signal/signal.go:108` partitions on
`Eval`'s `Outcome` and reads no grade, so the two censuses are the same three lists over the same
population, byte for byte. A reader comparing them would be told they may not be compared, and would
find nothing that differs when they looked.

## Decision

> **A `Signal` rule's severity does not compose into that rule's version vector, and it is not a
> fifth part of a rule. A rule is still four parts. What composes into the vector is everything the
> rule declares that changes **which subjects it matches** or **what it asserts about them**. An
> operator-facing grade changes neither, so a deliberate re-rating leaves every census the rule has
> already produced comparable with every census it produces after.**

Five limbs.

### 1. The vector's test is output-affecting, and a grade fails it

ADR-0008 fixed the test at an *output-affecting change only*. ADR-0021 restated it as *its output
can move while the world does not*. A rule's output is its census: three member lists over one
population, and the rule's own identity beside them.

A severity moves none of the three lists and moves no subject between them. It is read after the
census exists, by the web layer, to rank and badge rows the engine already partitioned —
`cmd/web/reports.go:283`, `cmd/web/graph.go:464`, `cmd/web/subjects.go:550`, `cmd/web/auth.go:599`
and six more sites all call `SeverityFor(rule)` on a census that is already built.

So the grade is downstream of the output the vector versions. Composing it would move a version for
a reason the version's own definition excludes.

### 2. The withdrawn clause of ADR-0024, and the wording that replaces it

ADR-0024's *everything a rule declares composes into its version vector* is withdrawn at ADR-0024's
own site, per
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). Read alone
and in the present tense it instructs a session to fold the grade in, which is the failure mode
ADR-0058 exists to close.

It is replaced by: **everything a rule declares that changes which subjects it matches, or what it
asserts about them, composes into its version vector. The operator-facing grade does not.**

The clause's own job is untouched. ADR-0024 wrote it to stop a domain reaching for a fact for free,
and the replacement still stops that: a domain is exactly a statement about which subjects the rule
matches, so no domain edit escapes the vector under the new wording. The clause was doing narrow
work and was written wide.

### 3. A severity is not a fifth part, and this is the reading taken

**The reading taken: a rule declares five things, and four of them are *parts*.** A part is a term
of the rule's verdict — the domain says of whom the question is asked, the predicate answers it, the
`not-evaluable` case says when we may not answer, and the version vector says under what evidence
the answer was reached. A grade is not a term of the verdict. It is how a console ranks a verdict
already reached, for one operator, in one inbox.

**The reading refused: number it a fifth part.** Refused because the part list is the only place
ADR-0024 says what a rule declares, so numbering the grade a part is what puts it back inside the
vector by ADR-0024's own guard, wide or narrow. The two questions are one question, and answering
*fifth part* answers *in the vector* whether the author meant to or not.

**This follows ADR-0024's own precedent, which points the same way.** Its
[#138](https://github.com/winniel123/verge-asm/issues/138) amendment box handled the last thing
proposed as a fifth part, the `Vantage composition`, and kept the count at four: *"the composition
is **not a fifth part of a rule**: a rule is still four parts, and what was missing was the
requirement to state both halves rather than a place to put them."* That case resolved to four
because both halves of the composition landed inside existing parts. This case resolves to four for
the opposite reason: the grade lands inside none of them, and belongs to a different layer.

The
[v1 SPEC §5.2](../spec/v1-spec.md) already writes it this way and has since P0.1. It states the
five-level ramp in one sentence and then says *"Four parts"*, listing the domain, the predicate, the
`not-evaluable` case and the version vector. This ADR makes that ordering a ruling.

### 4. What a re-rating costs, and what it does not

A re-rating is deliberate. It is a release-authored change to a shipped rule, and it is argued at
the rule, not per install and not per subject.

It costs: a code change in `internal/signal`, a review of the argument for the new band, and a
changed badge and sort position on every console surface that ranks by grade.

It does not cost: a version move, a `Break`, a comparability cycle, a corpus row, a migration, or a
rewrite of anything stored. Nothing persists a grade, so no stored row disagrees with the new one
and no backfill exists to run.

**Two censuses of one rule taken at different severities are comparable, and the comparison is
sound.** That is the clause this ADR exists to write down: comparability is a claim about evidence,
and no evidence moved.

### 5. What this does not reach

- **Whether a signal carries a severity at all, and at what granularity.** ADR-0116 rules it, per
  rule and five levels, and `CONTEXT.md`'s `Signal` entry restates it. This ADR takes both as given.
- **The ramp itself.** The five bands, their order, and `SevOrder`'s ranking are P0.1's and are not
  reopened.
- **What rates any individual rule.** For the certificate family that is
  [ADR-0186](./0186-a-certificate-rules-severity-is-rated-by-the-client-failure-it-can-show-today-never-by-certificate-quality.md).
  The other twelve rules' grades are still unwritten.
- **A `Message`.** A message carries no severity
  ([ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md)), so there is
  nothing here to exclude from anything.
- **Everything else a rule declares.** The vector still moves on a domain edit, a predicate edit, a
  `not-evaluable` edit, and on every leaf version the rule composes. This ADR subtracts one item and
  adds none.

## Consequences

- **This ADR changes no Go code.** `Version()` already ignores `sev` at every one of the seventeen
  rules, and nothing stores a grade. The ruling records the behaviour and supplies the reason a
  future author may not reverse it for symmetry.
- **[ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md) gains a withdrawal at its own
  site**, per ADR-0058, with the replacement wording from §2. It is recorded in this batch's
  manifest and applied in the same change, not by this ADR's author.
- **[`CONTEXT.md`](../../CONTEXT.md)'s `Signal` entry gains one clause**, beside the four-part
  sentence it already carries: the grade composes into no vector and is not a fifth part. Recorded
  in the manifest.
- **`internal/signal/severity.go`'s surviving comment gains a citation.** It states this rule
  uncited today. Recorded in the manifest.
- **This ADR is now the only site for the clause.** `PARITY-CHART.md` and `SPEC-CHANGE.md`, the two
  documents the deleted comment cited, are not in the tree.
- **Nothing enforces the exclusion.** No test asserts that `Version()` is independent of `sev`, and
  a rule author who appended a grade token to `Version.Composes` would pass `go vet`, the golden
  corpus and every gating check. Review carries this rule, as it carries ADR-0149's. A test is
  cheap here and is worth its own ticket: assert that no rule's `Version().String()` changes when
  its `Severity()` is varied.
- **The ruling's own test exposes a live violation, and it is not this ADR's to fix.**
  `certificate-expiring`'s predicate reads a 30-day window, `certExpiryWindow` at
  `cmd/web/deltas.go:18`, applied in the fold at `cmd/web/signals.go:1054`. That constant decides
  **which subjects the rule matches**, so under §2's replacement wording it composes into the rule's
  vector. It does not: `certVersion()` composes `co.CertVersion` alone. The repo already knows the
  shape and applies it one rule away — `weakKeyRule.Version()` at `internal/signal/endpoint.go:103`
  composes a read-side floor token, `weak-key-floor/v1`, precisely because moving the floor moves
  what fires. `certificate-expiring` is missing the same treatment. **This ships as its own
  ticket**, and it is entangled with the second defect below.
- **A second, larger defect sits under the same constant, and it is a spec drift rather than a
  version-vector gap.** [ADR-0043](./0043-a-clock-reading-rule-bounds-its-evidence-in-the-subjects-own-units.md)
  rules that `certificate-expiring`'s horizon is `N = ⅓ × (not_after − not_before)`, and `½ ×` below
  a 10-day validity, and that all three clock-reading certificate rules gain an evaluability guard
  on the observation's age against that horizon. Its Consequences price the change at *"three rules
  change their predicate, so three rules `Break`"*. The shipped code implements neither: it reads a
  flat 30 days and applies no age guard. A flat 30 is the exact value ADR-0043 §7.3 names as the
  failure it repaired. **This ships as its own ticket**, ahead of the version-vector fix above,
  because the vector cannot be corrected before the predicate it versions is the ruled one. This ADR
  states the defect and rates nothing by it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Fold the grade in — ADR-0024's clause read literally** | Charges a `Break` on every timeline of a rule for a change that moves no member between the three lists, moves no subject into or out of the domain, and leaves the census identical byte for byte. The reader is told two censuses may not be compared and finds nothing that differs. It also prices the one part of a rule an operator argues with at the highest price in the model, so the ramp would be defended against correction by its cost rather than by its argument |
| **Number the severity a fifth part of a rule, but keep it out of the vector** | Splits ADR-0024's guard from its own part list. The guard is stated over *everything a rule declares*, and the part list is the only place that ADR says what a rule declares. The split leaves two readings of *declares* inside one document, and the next author to meet the guard alone folds the grade in correctly by its wording. §3 resolves the two questions together for exactly this reason |
| **Mint a second, severity-only version axis** | Creates a comparability question nobody has: two censuses would be comparable on evidence and not on grade, and the console would have to render which. It also re-imports the cost the exclusion exists to avoid, one axis over, and every consumer of `Version.String()` — the drawer at `cmd/web/signals.go:385`, the subject page at `cmd/web/subjects.go:553`, the cold page at `cmd/web/cold.go:399` — would have to choose one |
| **Store a severity on `signal_instance` so a re-rating is dated** | `db/migrations/22300_signal_instance.sql` refuses it in the table's own header: only identity and first-seen persist, and *"the severity is the rule's … not stored"*. Storing it would make a grade a fact about history, so a re-rating would need a backfill and a rule would carry two grades at once. It also puts an operator-facing ramp inside the record, which is the collapse ADR-0064 refused for the message store |
| **Rule it on [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md)** | That ADR is `Superseded`, and what survives it is one bullet whose scope was fixed by [#1410](https://github.com/winniel123/verge-asm/issues/1410) at *a signal carries a five-level severity assigned per rule*. Adding a new rule to a superseded document puts a live decision behind a status word every reader is told to distrust |
| **Rule it on [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)** | Its subject is the measurement binary, its five decision procedures and the authored hermetic corpus. A `Signal` rule is not a measurement leaf and has no corpus rows of that kind. The ruling would state a signal-layer rule inside a document scoped to the prober |
| **Leave it to the code comment** | The comment was deleted by #1302, and the two documents it cited for its own authority — `PARITY-CHART.md` and `SPEC-CHANGE.md` — are in no path in the tree. There was nothing left to leave it to |
| **Say nothing, because the code already behaves this way** | The behaviour is invisible at the site that would break it. A rule author reads ADR-0024's guard, reads `Severity()` sitting on the `Rule` interface beside `Version()`, and folds the grade in as the consistent move. Nothing in the tree contradicts them, and the golden corpus passes either way |
