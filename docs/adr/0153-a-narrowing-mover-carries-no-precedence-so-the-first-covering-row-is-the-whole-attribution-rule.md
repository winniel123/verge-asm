# ADR-0153: A narrowing mover carries no precedence, so the first covering row is the whole attribution rule

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1326 ADR gaps: internal/queue (seed withdrawal limbs)](https://github.com/winniel123/verge-asm/issues/1326), gap 1
- **PR that deleted the comment:** [#1325](https://github.com/winniel123/verge-asm/pull/1325)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md) (a narrowing fires at the scope and carries a count), [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md) §3 (an address `Seed`'s scope **is** its CIDR), [ADR-0135](./0135-a-name-seed-withdrawal-states-one-act-and-its-tombstone-carries-the-domain-alone.md) §1 (a name `Seed`'s scope **is** its domain)
- **Sibling of, and not ruled by:** [ADR-0154](./0154-a-narrowing-fold-closes-only-what-it-can-attribute-to-a-mover-and-drops-every-other-candidate.md). That ADR rules what the fold does when **no** mover covers the candidate. This ADR rules which mover it takes when **several** do

## Context

Four functions in `internal/queue` answer one question. Each takes a subject and a corpus of
narrowing movers, and each returns the mover that covers the subject. Each walks the corpus in
order and returns on the first hit. None of the four compares two covering rows.

| Function | Site | Corpus |
| --- | --- | --- |
| `coveringAddressExclusion` | [`internal/queue/membership.go:255`](../../internal/queue/membership.go) | live `exclusion` rows |
| `coveringExclusionKey` | [`internal/queue/membership.go:69`](../../internal/queue/membership.go) | live `exclusion` rows |
| `coveringSeedWithdrawal` | [`internal/queue/seedwithdrawal.go:177`](../../internal/queue/seedwithdrawal.go) | pending `seed_withdrawal` rows, kind `address` |
| `coveringNameSeedWithdrawal` | [`internal/queue/nameseedwithdrawal.go:160`](../../internal/queue/nameseedwithdrawal.go) | pending `seed_withdrawal` rows, kind `name` |

#1325 deleted the only statement of the rule from `nameseedwithdrawal.go` and compressed the one in
`seedwithdrawal.go`. The deleted text read:

> Containment is the family-matched prefix test the coverage predicate already applies, so an IPv4
> address is never read as inside an IPv6 scope. Withdrawals do not nest into a precedence the way
> `Seed`s do — each one is the same "I stopped declaring this" — so first match is the whole rule.

`coveringAddressExclusion` states the same rule and cites
[#1032](https://github.com/winniel123/verge-asm/issues/1032). That issue is live and on topic, and
its body never states it. It rules the closure ground, the disjunctive membership limb and the
preview's wording, and it lists two open questions. Neither is this one. Per
[`comment-policy.md`](../spec/comment-policy.md) §8.3, a source suppresses only where it rules the
thing, so the citation does not stand for the rule.

**The declaration side does compare, and that is the contrast the rule lives against.** Three reads
take the **most specific** covering `Seed`.

- `FindCoveringAddressSeed` (`db/queries/subjects.sql:311`) orders by `masklen(s.address_cidr) DESC`.
- `FindCoveringNameSeed` (`db/queries/subjects.sql:320`) orders by `length(s.name_domain) DESC`.
- `narrowingScope` (`internal/queue/withdrawal.go:104`) keeps the largest `Bits()`, and its own
  comment says it must mirror `FindCoveringAddressSeed`.

**#1326 named a fourth site for that contrast and it is wrong.** The ticket cites `coveringSeedKey`
(`internal/queue/produce.go:370`) as a longest match over the live `Seed` corpus. It is not. It
returns the first covering `Seed` in `ListSeeds` order, which is `created_at DESC, id DESC`, so it
answers with the newest covering `Seed` rather than the most specific one. The contrast this ADR
rests on is the three reads above. `coveringSeedKey` is an unruled disagreement on the declaration
side, and §5 below excludes it.

**Both narrowing corpora are totally ordered, so "first" is deterministic.**
`ListPendingSeedWithdrawals` and `ListPendingNameSeedWithdrawals` order by `w.id`, so the oldest
tombstone wins. `ListExclusions` orders by `e.created_at DESC, e.id DESC`, so the newest exclusion
wins. The two corpora therefore run in opposite directions, and neither read is unstable.

**What the choice decides is narrow.** The returned scope becomes the receipt's scope string and its
count bucket, so it names the act the operator reads in the coverage message. It does not decide
whether the span closes, because any covering mover reaches the same closure. It does not decide
whether a tombstone is spent, because `SpendSeedWithdrawals` runs its own per-row exhaustion test in
SQL and never reads the covering function.

## Decision

> **A narrowing mover carries no precedence over another mover of the same kind. A fold that must
> name the mover takes the FIRST covering row in the corpus's own read order, and compares no
> further. There is no longest-prefix match on an address limb and no most-specific-apex match on a
> name limb. The most-specific read belongs to the DECLARATION side, where it answers a different
> question.**

Five limbs.

### 1. The four covering functions state one rule

`coveringAddressExclusion`, `coveringExclusionKey`, `coveringSeedWithdrawal` and
`coveringNameSeedWithdrawal` return on first containment. A change to any one of them is a change to
this rule. It needs a new decision rather than a local fix.

### 2. The ground is that every narrowing act makes the same claim

A `Seed` declaration says *measure this ground*. A second declaration over a smaller scope says the
same thing about less ground. The two nest, so the operator recognises the more specific one as the
site the estate hangs on.

A withdrawal says *I stopped declaring this*, and an exclusion says *forbid this*. Two withdrawals
over overlapping ground make the identical claim about the ground they share. Neither is more
specific about the act, and the fold holds no fact that ranks them. A comparison would invent one.

### 3. The order is the corpus's own, and it must stay total

The rule forbids a comparison inside the fold. It does not fix one direction across both corpora,
and today they differ. A tombstone read is oldest first. An exclusion read is newest first.

What the rule does require is a **total** order in the query. An unordered read would attribute the
same span to a different act on different folds, and a `Message` is written once and never
recomputed (ADR-0074, ADR-0134 §6). Any query that feeds one of the four functions carries an
`ORDER BY` that breaks every tie.

### 4. Containment is family-matched, and a NULL scope column is skipped

The address limbs test containment with `netip.Prefix.Contains`, which is false across families, so
an IPv4 address is never read as inside an IPv6 scope. The name limbs test with `nameWithinDomain`.

`seed_withdrawal` holds one scope column per kind under the `seed_withdrawal_shape` CHECK
(ADR-0135 §2), so the other column is NULL. Each covering function skips a NULL rather than reading
it as a match.

### 5. Two adjacent questions are named and excluded

Both are real and neither is ruled here, so that a later session does not read the silence as a
ruling.

- **The declaration side disagrees with itself.** `coveringSeedKey` takes the newest covering `Seed`
  and `FindCoveringAddressSeed` takes the most specific one. Both answer *which declared scope holds
  this subject*, and they can answer differently. This ADR rules the narrowing side alone.
- **An overlapping pair of withdrawals states one receipt.** Withdraw `10.0.0.0/8`, then withdraw
  `10.0.0.0/24`, and every span under the second is attributed to the first. The second tombstone
  still spends once its ground is exhausted, and it writes no message of its own. That cost is
  accepted below rather than ruled away.

## Consequences

- **[`internal/queue/seedwithdrawal.go`](../../internal/queue/seedwithdrawal.go) gains this ADR's
  citation** on the surviving line that states the rule.
- **[`internal/queue/nameseedwithdrawal.go`](../../internal/queue/nameseedwithdrawal.go) gains
  nothing.** #1325 deleted its copy of the rule under §4.1's cross-block test, and one rule stated at
  four sites is four copies to keep in agreement. One site speaks, and that site carries the
  citation.
- **[`internal/queue/membership.go`](../../internal/queue/membership.go)'s `#1032` citation names an
  issue that does not state the rule.** That file belongs to another sweep ticket, so the repair is
  not made here. This ADR is the source that ticket should cite.
- **An overlapping withdrawal is silent about ground an earlier withdrawal already claimed.** The
  operator reads one receipt, naming the older scope. The count is still correct, because each span
  is attributed once, and ADR-0074's rule of one act and one receipt is unhurt on the earlier act.
- **No production behaviour changes.** All four functions already have this shape.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Exclusion, `Seed` withdrawal and tombstone all
  have entries already, and this is a rule about a fold rather than a term.
- **Nothing enforces this.** No test pins the absence of a comparison, so review carries the rule.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Longest-prefix match over the pending tombstones**, mirroring `FindCoveringAddressSeed` | The fold holds no fact that ranks two withdrawals. Both say "I stopped declaring this" about the shared ground, and the more specific one is not the truer one. It changes which act the coverage message names, and a `Message` is written once and never recomputed |
| **Most-specific-apex match on the name limb** | The same ground as the row above, one limb over. `a.example.com` withdrawn under `example.com` withdrawn is two identical claims about the Names beneath the first |
| **Attribute a span to every covering mover** | One span closes once. Counting it under two scopes would state the same subjects twice across two receipts, which is the census ADR-0074 exists to replace |
| **Sort the pending set by mask length before the loop** | A longest match with the comparison moved one line up. It buys the same wrong answer and hides the decision from the reader of the covering function |
| **Make both corpora read in the same direction** | A real tidiness argument that rules nothing this ADR needs. The requirement is a total order, and both queries already carry one. Reordering `ListExclusions` would move the mover named on existing overlapping exclusions, for no stated gain |
| **Leave the rule uncited at all four sites** | The state #1326 recorded. #1325 could delete the last statement of it, and a later session reading `coveringSeedWithdrawal` beside `narrowingScope` would read the missing comparison as an omission rather than a decision |
