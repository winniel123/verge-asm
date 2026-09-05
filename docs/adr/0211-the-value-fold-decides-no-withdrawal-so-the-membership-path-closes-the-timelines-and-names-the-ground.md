# ADR-0211: the value fold decides no withdrawal, so the membership path both closes a departing subject's timelines and names the ground

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1321 ADR gaps: internal/queue (#1199)](https://github.com/winniel123/verge-asm/issues/1321), gap 4
- **PR that deleted the comment:** [#1327](https://github.com/winniel123/verge-asm/pull/1327)
- **Rests on:** [ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md), which rules that a withdrawn subject's timelines close, at every key the subject held. It rules **that** they close. It does not rule which fold closes them
- **Rests on:** [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md), which closes the reason vocabulary at three and rules that **a withdrawal's closure carries a reason, and no other** — an ordinary value move needs none, because the next span is the fact. It rules the reason. It does not rule which fold records it
- **Rests on:** [ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md), which rules what a cross-class composition is. `resolutionWitnesses` (`internal/queue/membership.go:141`) cites it for the collapse this ADR relies on
- **Rests on:** [ADR-0007](./0007-drift-is-a-timeline-of-spans.md), whose incremental fold is what runs first in §4's ordering
- **Sibling of, and not ruled by:** [#1315](https://github.com/winniel123/verge-asm/issues/1315) gap 4, an open record about **when** membership is re-decided. This ADR rules **which fold** decides it. The two touch the same two functions and state different rules

## Context

`internal/queue/spanfold.go:31` carried this in declaration position, until #1327 deleted it:

```go
// A withdrawal is NOT decided here: whether a subject left the estate is a
// subject-level cross-class composition (internal/estate), and closing its
// timelines with a `measured-absent`/`uncited`/`descoped` ground is that path's
// job. This fold only tracks per-timeline value movement.
```

The same block's first paragraph also stated that *"an ordinary value move, so the closure records no
reason (the next span is the fact)"*. **That half is already ruled**, verbatim, by ADR-0087's
Decision table — *"Which closures carry a reason: a withdrawal's, and no other. An ordinary value
move needs none — the next span is the fact."* So the gap is narrower than #1321 records it, and the
uncovered part is the division of labour: which fold decides a departure, and where the ground is
named.

**Five folds run in one transaction, in this order** (`internal/queue/worker.go:429-445`):

| Order | Fold | Site | What it closes | Reason it records |
| --- | --- | --- | --- | --- |
| 1 | `foldObservationsIntoSpans` | `spanfold.go:23` | one span at one timeline key, and only to open its successor | **none** — the field is left unset |
| 2 | `foldEstateTransitions` | `membership.go:38` | **every** open span the subject holds | `measured-absent` or `descoped` |
| 3 | `foldNameSeedWithdrawals` | `nameseedwithdrawal.go:12` | the spans the withdrawn name seed covered | `descoped` |
| 4 | `foldAddressExclusionWithdrawals` | `withdrawal.go:14` | the spans the exclusion covers | `descoped` |
| 5 | `foldSeedWithdrawals` | `seedwithdrawal.go:14` | the spans the withdrawn seed covered | `descoped` |

Fold 1 calls `qtx.CloseSpan` at `spanfold.go:84` with `ClosedAt`, `ClosedBatchID` and `ID`. It omits
`ClosureReason`, so the column is written NULL. Folds 2 to 5 all reach `closeSpansByID`
(`membership.go:186`), which sets it on every row.

**The division is forced by what each function can see, and that is the measurement.** `foldOne`
(`spanfold.go:36`) is called once per observation and holds exactly one `drift.TimelineKey` —
`(subject kind, subject key, facet, discriminator, source)` against one vantage. A departure is not a
fact at that grain. `decideNameDeparture` (`membership.go:119`) reads
`estate.DecidedAbsentCrossClass` or `estate.WithdrawnCrossClass` over the witnesses that
`resolutionWitnesses` (`membership.go:141`) collects from **all** of the subject's open resolution
spans, one class per vantage. `foldOne` never loads that set, and cannot: it is inside a loop over
one batch's observations, with one key in hand.

**A session reading `foldOne` alone would get this wrong in a specific way.** It sees a `CloseSpan`
call and a `db.CloseSpanParams` struct with a `ClosureReason` field it does not fill. The natural
repair is to fill it. Filling it is exactly the error: it would make an ordinary value move look like
a withdrawal to every reader of the span corpus, and ADR-0087 prices a closure reason at under 2% of
the corpus **because only withdrawal closures carry one**.

**The ordering is not incidental either.** `membership.go:41` records that the value fold runs first,
so a name's current resolution span is already open when `foldEstateTransitions` reads
`ListOpenSpansForSubject`. The membership decision reads the post-fold state. If the value fold
closed a departing subject's timelines, the membership path would find nothing open and could not
decide the departure at all.

## Decision

> **`foldObservationsIntoSpans` decides no withdrawal. It moves one timeline at a time, and it closes
> a span only in order to open that timeline's successor, with no reason recorded. Whether a subject
> left the estate is a cross-class composition over every timeline the subject holds, so the
> membership path decides it, closes all of those timelines, and names the ground. A `CloseSpan` call
> that carries a reason belongs on the membership path and nowhere else.**

### 1. The value fold's closure is always paired with an opening

`foldOne` closes at `spanfold.go:84` and opens at `:88`, in that order, in one transaction. There is
no path through the function that closes a span and opens none. That pairing is what makes the
closure an ordinary value move, and it is why ADR-0087 needs no reason on it: the next span is the
fact.

A bare close — a closure with no successor — is a withdrawal, and it is not this fold's act.

### 2. The membership path owns the departure, because it is the only path that can see it

The composition needs every open span of the subject, across every vantage class. Only
`foldEstateTransitions` loads that set. This is not a preference about where to put the code. A
departure decided from one timeline key would be a claim about the estate made from one vantage, and
ADR-0080 rules that a cross-class composition is the thing that answers this question.

### 3. The ground is named there, and only there

Four folds record a reason and all four reach `closeSpansByID`. Three of them close on our own
aperture and record `descoped`. `foldEstateTransitions` records `measured-absent` where the estate
composition decides absence, and `descoped` where an exclusion covers the name
(`membership.go:123-137`). Which of ADR-0087's three grounds applies is ADR-0087's question, not
this one.

### 4. The value fold runs first, and it must

The membership path reads the state the value fold produced. Reversing the two would make the
departure decision read the pre-fold estate, so a name whose last resolution went `NameError` in this
very batch would not be seen to have departed until the next one.

### 5. What this rule does not reach

- **When membership is re-decided.** `membership.go:39` records that it is re-decided only where
  fresh evidence arrived, never as a background sweep. That is a separate rule and
  [#1315](https://github.com/winniel123/verge-asm/issues/1315) holds its record.
- **Which of the three grounds applies.** [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md)
  rules the vocabulary and the sorting.
- **Whether a closure carries a reason at all.** ADR-0087 rules that too, and §Context shows the
  deleted comment was restating it.
- **The three declaration-driven withdrawal folds.** They are on the membership side of this line by
  construction — they read the declaration and not an observation — and this ADR confirms rather than
  moves them.

## Consequences

- **No production behaviour changes.** All five folds already have this shape.
- **The record is narrower than #1321 states, in one limb.** The deleted comment's *"closes with no
  reason"* clause is already ruled by ADR-0087's Decision table. This ADR rules only the division of
  labour, and the manifest's comment citation names this ADR for that half alone.
- **`internal/queue/spanfold.go` gains this ADR's citation** on the survivor comment beside the
  `CloseSpan` call. It is recorded in this issue's manifest rather than edited here.
- **A defect: nothing pins the value fold's closure as reason-free.** `internal/queue/membership_test.go`
  asserts the reason each departure records. No test asserts that a span closed by an ordinary value
  move has a NULL `closure_reason`, so the repair §Context describes would pass CI. One assertion in
  the span-fold tests would close it, and it ships as its own ticket.
- **`CONTEXT.md` gains nothing.** It already carries `Closure` and its three grounds, added by
  ADR-0087. Which function performs the close is not a domain term.
- **A sixth fold is admitted by this rule.** A fold that closes a span must either open its successor
  in the same act, or decide a departure over the subject's whole open set and name the ground.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Decide the withdrawal in the value fold** | `foldOne` holds one `drift.TimelineKey` against one vantage. The composition needs every open span of the subject across every vantage class, which `resolutionWitnesses` loads and `foldOne` never reads. The decision is not merely inconvenient there. It is not available there |
| **Let the value fold close the span and have the membership path back-fill the reason** | Two writes to one row from two paths in one transaction, whose intermediate state is a closed span with no reason — which is precisely how an ordinary value move reads. Any reader between the two writes, and any future fold that runs between them, sees a withdrawal disguised as a value move |
| **Give the value fold's closure a fourth reason, such as `value-moved`** | ADR-0087 closes the vocabulary at three, sorted on the **ground the closure rests on**, and rules that an ordinary move needs none because the next span is the fact. A fourth member would name a distinction the three grounds do not make, and would put a reason on every closure in the corpus rather than on the withdrawals ADR-0087 priced |
| **Merge the value fold and the membership fold into one pass** | The value fold is per observation and the membership fold is per subject. Merging runs the cross-class composition once for every observation in the batch, and it also destroys §4's ordering, which the membership read depends on |
| **Move the reason onto the departure record rather than the span** | ADR-0087 already refused a second representation: a closure *"is not an object"* and adds one field to a row that exists. The `departure` value the fold collects feeds the message producer and is not the durable record |
| **State the rule in `CONTEXT.md`'s `Closure` entry** | The glossary carries terms, and `Closure` already has its entry. This rule is a division of labour between two functions in one package, and filing it in the glossary would put it where readers look for a vocabulary |
| **Leave it to the surviving comment** | The comment is one sentence in one of the two files it binds. A session editing `foldOne` and filling the `ClosureReason` field it can plainly see would break the corpus, and the comment sat in `spanfold.go` only |
