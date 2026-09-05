# ADR-0218: the address exclusion withdrawal is idempotent by construction, and its receipt twins the preview's firing site and its counts

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1323 ADR gaps: internal/queue (queue, cttail, withdrawal, hot, ctverify, scopegate)](https://github.com/winniel123/verge-asm/issues/1323), gap 9
- **PR that deleted the comment:** [#1322](https://github.com/winniel123/verge-asm/pull/1322)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rules what [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md) §5 assumed.** That section rules the tombstone's idempotency and reaches for this fold as its comparison: *"the three survivors give the same by-construction idempotency the exclusion act has"*. It states the property of this fold as a premise and rules it for the tombstone alone. §1 below rules it here
- **Rests on:** [ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §8.1, which rules **when** the withdrawal runs and that the fold reads the live `exclusion` corpus. §2 below is why that corpus makes a marker unnecessary
- **Rests on:** [ADR-0154](./0154-a-narrowing-fold-closes-only-what-it-can-attribute-to-a-mover-and-drops-every-other-candidate.md), which rules that the fold closes only an attributable candidate and drops the rest. Those drops are what keep the candidate set stable across folds
- **Rests on:** [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md), which fixes a narrowing as one receipt at the scope, carrying a count
- **Sibling of, and not ruled by:** [ADR-0153](./0153-a-narrowing-mover-carries-no-precedence-so-the-first-covering-row-is-the-whole-attribution-rule.md). It rules which **mover** the fold names when several cover a candidate, and its §5 names the declaration side as an open question it does not rule: *"`coveringSeedKey` takes the newest covering `Seed` and `FindCoveringAddressSeed` takes the most specific one … This ADR rules the narrowing side alone."* §3 below rules one half of that pair

## Context

[`internal/queue/withdrawal.go:76`](../../internal/queue/withdrawal.go), `:170` and `:192` carried
this text, until #1322 deleted it:

```go
// It is idempotent by construction rather than by a marker. The closure is what
// removes the row from the query's answer, so the next batch reads no row, closes
// nothing and fires no second message.
```

```go
// It mirrors the preview's
// FindCoveringAddressSeed, most-specific-covering-scope-first, so the act and the
// receipt name the same firing site.
```

The sweep left three compressed lines, at `withdrawal.go:19`, `:105` and `:87`. Nothing on disk
states any of the three rules. That is #1323's gap 9.

**The idempotency is stated as a premise in one ADR and ruled in none.** ADR-0134 §5 rules the
tombstone's idempotency and borrows this fold's as a known quantity. A reader who follows the
reference arrives at ADR-0133 §8.1, which rules when the withdrawal runs and whether its counts must
match the preview's, and never rules how a second fold is safe.

**The closure is the marker, and the query is where that is visible.**
`ListAddressExclusionWithdrawals` (`internal/db/messages.sql.go:72`) filters `s.closed_at IS NULL`
three times: in the `withdrawn_addr` CTE, in the resolution-survivor `NOT EXISTS`, and in the outer
`SELECT`. `closeSpansByID` calls `CloseSpan`, which writes `closed_at`, `closure_reason` and
`closed_batch_id` (`membership.go:186`). The rows the fold just closed are gone from the next fold's
answer. The `exclusion` table carries no consumed marker for the address limb, and the fold writes
none.

**The two counts are twinned at two different places, and the twinning is easy to break at either.**

| Fact | Preview side | Act side |
| --- | --- | --- |
| Which declared scope the receipt names | `FindCoveringAddressSeed` (`internal/db/subjects.sql.go:15`), `address_cidr >>= $1::inet`, `ORDER BY masklen(address_cidr) DESC LIMIT 1` | `narrowingScope` (`withdrawal.go:104`), `Contains(excluded.Addr())`, keep the largest `Bits()` |
| Fallback where no scope covers | `scope = p.String()` (`cmd/web/exclusions.go:79`) | `return excluded.String()` (`withdrawal.go:121`) |
| How the counts become a sentence | `message.PreviewNarrowing(scope, p.String(), …)` (`exclusions.go:97`) | `message.PreviewNarrowing(c.scope, key, …)` (`withdrawal.go:81`) |

## Decision

> **The address exclusion withdrawal carries no idempotency marker. The closure is what removes the
> row from the fold's own query, so a second fold reads no row, closes nothing and composes no
> second receipt. Its receipt must name the firing site the preview named, by the same
> most-specific-covering-scope rule, and it must render through the preview's own constructor, so
> the two can never state one act in two ways.**

Five limbs.

### 1. The closure is the marker, and the survivor rules are what make that hold

Idempotency here is a property of the query and the write together. The query returns open spans.
The write closes them. Nothing else in the fold changes state.

That holds only because the candidate set is stable between folds, and three rules keep it stable.

- A row with no covering exclusion is dropped and stays open (ADR-0154 §1). The next fold drops it
  again, for the same reason.
- An address the custody extension still reaches is dropped and stays open (ADR-0133 §8.1). The next
  fold reads the same `Estate` and drops it again.
- A row that is closed is not open, so it is outside the query.

So a second fold over an unchanged estate is a read that returns nothing new and a write of nothing.
A **third** state does not exist: there is no row that was closed and is a candidate again, because
`CloseSpan` is not reversed anywhere.

This is ruled here rather than borrowed. ADR-0134 §5 may go on citing it.

### 2. This act needs no marker because its mover survives, and the tombstone act needs one because its mover does not

The two halves of the narrowing family look symmetric and are not.

The exclusion act's mover is a live `exclusion` row. It is written at declaration time and it stays.
The fold re-reads it on every batch (ADR-0133 §8.1), so a candidate the fold could not close today is
reachable again tomorrow with no state of the fold's own.

The `Seed` withdrawal's mover is destroyed by the act, so a tombstone stands in for it
(ADR-0134). A tombstone is consumable, so it needs a stamp, and ADR-0134 §5.1 rules that
**liveness** depends on that stamp while idempotency does not. That is the sentence a reader must
not generalise backwards. A marker on this fold would buy nothing, because there is nothing here
that can be spent.

### 3. The receipt names the most-specific covering scope, and `narrowingScope` mirrors the preview's query

The operator sees the same sentence twice: once when they preview an exclusion, and once when the
fold performs it. A narrowing fires **at the scope** (ADR-0074), so the sentence names a declared
scope rather than the excluded CIDR. If the two sides picked that scope differently, the receipt
would name a firing site the preview did not, for an act the operator already approved.

The rule is **most-specific covering scope first**, and the two implementations must agree on every
part of it.

- **Corpus.** Both read the `seed` table. The preview queries it live; the fold reads `in.seeds`
  from `ListSeeds`, which is unfiltered and ordered.
- **Predicate.** `address_cidr >>= $1::inet` against the exclusion's base address, and
  `AddressCidr.Contains(excluded.Addr())` against the same address.
- **Choice.** `ORDER BY masklen(address_cidr) DESC LIMIT 1`, and keep the largest `Bits()`.
- **Fallback.** The excluded CIDR's own string, where no `Seed` covers it.

Ties cannot diverge. Two address `Seed`s of equal mask length that both contain one address are the
same prefix, so both sides render the same text.

**This is the declaration side, and ADR-0153 rules the other one.** ADR-0153 forbids a
most-specific comparison when the fold picks the **mover**, because two withdrawals over shared
ground make the identical claim and nothing ranks them. Here the question is which declared scope
the operator recognises as the site their estate hangs on, and a more specific declaration is the
more specific statement about that ground. The two rules point in opposite directions on purpose.

### 4. Both sides render through `message.PreviewNarrowing`, and neither formats a sentence of its own

`withdrawalCount` (`withdrawal.go:89`) holds the scope, a subject set and a timeline count, and it
formats nothing. `composeAddressWithdrawals` passes those four values to
`message.PreviewNarrowing`, which is the same function `cmd/web/exclusions.go` calls.

The constructor derives everything a reader compares: `Fires` from `subjectsWithdrawn > 0`,
`Headline` from all four values, and `Loss` from the removed CIDR. None of them is a parameter. So
the preview and the act cannot state one act in two wordings, and they cannot disagree about whether
it fires at all.

`TestComposeAddressWithdrawalsRendersThroughPreviewNarrowing` (`withdrawal_test.go:221`) locks the
act side by constructing the expected receipt with the same call.

### 5. The twinning is of the rule, never of the instant

§3 and §4 promise that the two sides apply one rule to one input. They promise nothing about the two
sides seeing the same estate.

ADR-0133 §8.1 already rules that half: the preview and the fold each measure at their own instant,
and a disagreement between their counts is correct behaviour. A `Seed` declared between the preview
and the fold legitimately moves the firing site as well as the counts. This ADR does not reopen it.

## Consequences

- **This ADR changes no Go code.** All three rules already ship.
- **[ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md)
  §5's premise is now a citation rather than an assumption.** Its sentence stands as written and
  needs no edit, because it asserts the property rather than specifying a mechanism. ADR-0058 does
  not reach it.
- **`narrowingScope` and `FindCoveringAddressSeed` are twins that no test compares.**
  `withdrawal_test.go` exercises the Go side and `cmd/web/handlers_test.go:2055` hand-writes a fake
  that reimplements the SQL side. A change to either passes both suites. **A test that drives one
  input through both and asserts one answer ships as its own ticket.** It needs a database, so it
  belongs beside the other query-level tests rather than in `internal/queue`.
- **[ADR-0153](./0153-a-narrowing-mover-carries-no-precedence-so-the-first-covering-row-is-the-whole-attribution-rule.md)
  §5's first open question is now half ruled.** It names two disagreeing declaration-side reads,
  `coveringSeedKey` and `FindCoveringAddressSeed`. §3 rules the pair `FindCoveringAddressSeed` and
  `narrowingScope`. The `coveringSeedKey` disagreement is untouched and still needs its own record.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** No domain term moves. The receipt's wording and
  its class are ADR-0074's.
- **Three comments carry a rule and no citation.** `withdrawal.go:19`, `:105` and `:87` each state
  one limb. Citations are recorded in this issue's manifest.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Add a consumed marker to the `exclusion` row, mirroring the tombstone** | Marks a mover that is not consumed. ADR-0133 §8.1 requires the fold to re-read the live `exclusion` corpus on every batch, so a marked row would have to be unmarked whenever a dropped candidate became closable, and nothing knows when that happens. It also makes an operator's live declaration carry a fold's bookkeeping |
| **Stamp the closing batch on the exclusion row and filter on it** | The closure already records `closed_batch_id` on each span (ADR-0111), which is the durable answer to *which batch withdrew this*. A second copy on the mover would be a second source of truth for one fact, and it would be wrong the moment a later batch closes a further span under the same exclusion |
| **Let the fold format its own headline from the counts** | Two renderers for one sentence. The preview and the act would drift on wording, on the `Fires` threshold, and on whether a zero-subject act says anything at all. A `Message` is written once and never recomputed (ADR-0074), so the drift would be permanent in the corpus |
| **Take the newest covering `Seed`, matching `coveringSeedKey`** | Names a scope the preview did not name, for an act the operator approved against the preview's wording. It also picks the less specific statement about the ground more often, so a `/24` declared inside a `/8` would be invisible in the receipt |
| **Apply [ADR-0153](./0153-a-narrowing-mover-carries-no-precedence-so-the-first-covering-row-is-the-whole-attribution-rule.md)'s first-covering-row rule here too** | ADR-0153's ground is that two movers make the identical claim and nothing ranks them. Two declared scopes do not make the identical claim: the more specific one is the more specific statement about the same ground, and the operator recognises it as the site. Applying the mover rule here would make the receipt depend on `ListSeeds`'s `created_at DESC` order |
| **Require the preview and the act to agree on the counts as well as the rule** | ADR-0133 §8.1 ruled that they need not, because each measures at its own instant. Requiring agreement means either a lock held across an operator's page view or a stored preview the fold must honour, and both make the act state a measurement it did not take |
