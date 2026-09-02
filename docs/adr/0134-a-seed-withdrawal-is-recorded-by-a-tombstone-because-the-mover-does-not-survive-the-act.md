# ADR-0134: A `Seed` withdrawal is recorded by a tombstone, because the mover does not survive the act

- **Status:** Accepted
- **Date:** 2026-09-01
- **Ticket:** [#1040 Deleting a `Seed` writes no withdrawal](https://github.com/winniel123/verge-asm/issues/1040)
- **Follows:** [ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §8.1, which discharged the other narrowing act
- **Constrained by:** [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md) (a closure records no actor), [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md) (a narrowing fires at the scope), [ADR-0111](./0111-a-span-cites-the-batch-that-folded-it.md) (a closure cites its batch), [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) (retention is by readership)

## Context

`CONTEXT.md` names two acts that withdraw an `Address` from the estate. An exclusion is one.
A `Seed` that stops covering the address is the other. `Seed` and exclusion are the two halves
of one rule, and [#1032](https://github.com/winniel123/verge-asm/issues/1032) fixed the first
half alone.

The second half does nothing. `deleteSeed` resolves the scope's display string, runs
`DELETE FROM seed`, and redirects with a toast. It closes no span and writes no message. The
toast states the broken behaviour as though it were the design:

> nothing new is admitted under it; existing subjects keep their citations.

So an address the estate held **only** because that `Seed` covered it keeps its open timelines
for ever. No resolution cites it. No declaration covers it. Nothing closes it. The custody
extension does not reach it. It is in the estate on the strength of a row that no longer exists.

**The exclusion fix does not generalise.** ADR-0133 §8.1 ruled that the withdrawal is driven
from the declaration side: `ListAddressExclusionWithdrawals` reads the live `exclusion` corpus,
so the mover is a row the fold can still read, and the preview and the act read the same shape.
A `Seed` withdrawal destroys the mover in the same statement that performs the act. The delete
is a hard `DELETE FROM seed` and its foreign keys cascade. After it commits, nothing in the
database names what moved.

That matters because the fold must name the mover. `composeAddressWithdrawals` already refuses
to close a span it cannot attribute:

> A row it cannot attribute to a declared Exclusion is DROPPED, not closed. A closure with no
> mover to name is a withdrawal the operator cannot trace back to their own act.

The same rule binds here. It rules out the obvious cheap fix.

**One term needs sharpening first.** `CONTEXT.md` listed three bearers of the `descoped` ground
— an exclusion, a narrower `Seed`, and a release narrowing a composed population — and a
withdrawn `Seed` is literally none of them. It is not narrower. It is gone.

## Decision

### 1. Removal is the limiting case of narrowing, and the act is a withdrawal

A `Seed` withdrawn entirely and a `Seed` narrowed to a smaller scope differ in degree, not in
kind. Both stop our aperture covering ground it covered before, and the ground a closure records
is unchanged: `descoped`. The union of three closure grounds stays closed, and no fourth
ground-bearer is added.

`CONTEXT.md` now reads "a `Seed` that narrows or is withdrawn" in both places. The act is a
**`Seed` withdrawal**, which is the word the code already reached for on its own — both
`deleteSeed` and its query say a delete *withdraws* a declared `Seed`.

Narrowing a `Seed` in place is out of scope because it does not exist. `seed` supports create
and delete. Its one `UPDATE` sets `custody_extension`, which is a different act (§7).

### 2. A tombstone records the mover

Withdrawing a `Seed` writes a `seed_withdrawal` row in the same transaction as the delete. That
row is the mover the fold reads, standing in for the `exclusion` row the other act reads live.
The `seed` delete stays a hard `DELETE`, so every existing reader of `seed` is untouched and the
cascade the R4-R2 guard protects is unchanged.

This is the one place in the model where a declared input is read from a record of a past act
rather than from live truth. That asymmetry is the whole reason this ADR exists, and §4 is what
makes it safe.

### 3. The tombstone records the CIDR, the actor and the batch — and the actor stops there

An address `Seed`'s display scope **is** its CIDR, so the CIDR alone carries both the mover's
identity and the message's firing site. No display string is stored.

The row carries `created_by` and `created_at`, matching `seed` and `exclusion`. A withdrawal is
the counterpart act to a declaration, so the two are recorded the same way.

**ADR-0087's refusal is not weakened.** That ADR rejects recording an actor on a **closure**, and
its rejected-alternatives table names the case explicitly. A `seed_withdrawal` row is a declared
input, not a closure. The boundary is a rule, not an accident: `created_by` never reaches the
span and never reaches the message.

**Amended in implementation (#1040).** `created_by` is **nullable**, `ON DELETE SET NULL`, which
is where the mirror of `seed` and `exclusion` stops. Those two carry a `NOT NULL` FK that refuses
to remove an account which authored a live declaration, and the operator lifts that refusal by
removing the declaration. A tombstone is removable by no operator act, so the same FK would make
the admin who withdrew an address scope permanently undeletable, with no cleanup path even once
the row is spent. The attribution is worth keeping while the account exists. It is not worth
making a member undeletable.

### 4. Three survivors keep an address open

An address does not leave while any limb still holds it. The fold applies all three:

- **A current resolution cites it.** The disjunctive membership rule, applied in SQL as the
  `NOT EXISTS` clause the exclusion query already carries.
- **A live `Seed` covers it.** Read from the `Seed` corpus at fold time, never from the
  tombstone.
- **`custody.Estate.Derive` still calls it `operator`.** The custody extension lives on a
  **name** `Seed` and an address `Seed` can never carry one, so withdrawing an address `Seed`
  leaves any extension standing. Such an address is still enumerated, still probed and still
  measured.

The third survivor is forced by ADR-0133 §8.1 for exactly the reason it gave there. Without it
the address closes, the next batch reopens it, and the batch after closes it again — a `descoped`
departure and a coverage message every cadence, for ever, over an address the gate never stopped
probing. The survivor test must be `Derive` itself so the membership decision and the probe gate
read one derivation and cannot disagree.

**The live-`Seed` survivor is load-bearing twice**, and the second time is specific to the
tombstone. It settles an address that a second declared `Seed` still covers. It also settles
**re-declaration**: withdraw `10.0.0.0/24`, declare it again later, and the addresses re-enter
through the `Seed` limb while a stale tombstone still names the CIDR. Reading the live corpus is
what stops the tombstone closing ground that is declared again.

A re-citation needs no special rule. The resolution survivor already covers it.

### 5. It runs on the next membership fold, last, and its tombstone is then consumed

ADR-0133 §8.1's ruling holds unchanged and for the same two reasons. A closure cites the folding
batch (ADR-0111) and a web handler holds none. And a withdrawn scope stops being enumerated, so
a fold scoped to the subjects a batch observed could never reach the address.

The accepted bound is the same: the withdrawal lands on the next completed job, so an estate
running no jobs holds its spans open until one completes.

The fold runs **last** in the batch transaction, after the value fold, the Name estate fold and
the exclusion withdrawal. An address the exclusion fold already closed is not open, so it is
never counted or attributed twice.

The fold then stamps the tombstone with the batch that spent it, and the listing query filters
consumed rows out. Idempotency does not depend on the stamp — the three survivors give the same
by-construction idempotency the exclusion act has — but the stamp earns its place three times
over. It bounds the read. It satisfies ADR-0041, because a spent row is a row nothing may still
read. And it records which batch performed the withdrawal, mirroring ADR-0111 at the declared
input.

### 5.1 A tombstone is spent late, and the read is claimed

Amended in implementation (#1040). Three rulings the section above left implicit, each found by
review of the first commit.

**A tombstone is spent only once its withdrawal is EXHAUSTED** — once no open timeline is left
under its CIDR — never on the first fold that reads it. Idempotency does not depend on the stamp,
but **liveness does**, and §4's survivors are not all permanent. A citing resolution goes away. A
custody extension is turned off (§7). The exclusion twin can afford to act late because its live
`exclusion` row is still there to re-read on the next batch. A tombstone is the only mover its act
will ever have. Spending it while its ground is still held would strand exactly the addresses this
ADR exists to release — open for ever, uncited and undeclared — and nothing else would reach them:
`foldEstateTransitions` decides departures for **Names**, and `estate.AddressClosure` has no
production caller. A row that is not spent costs the next fold two reads and closes nothing,
because the same survivor drops every candidate again.

**An IPv6 withdrawal is never spent.** The candidate query inherits the IPv4-only subject-key gate
from `PreviewExclusionWithdrawal`, so it cannot see an IPv6 span and reports the ground empty when
it is not. For the exclusion twin that limit only bounds the act smaller, because the declared row
survives and a later widening still acts. Here the mover is destroyed, so a spent IPv6 tombstone
loses its ground for good. It stays pending until the gate widens.

**The pending read is claimed with `FOR UPDATE SKIP LOCKED`**, the idiom `ClaimJob` already uses.
Workers are multi-instance. Without the lock two folds completing at once both compose the same
receipt, and the loser writes a coverage message stating subjects its own batch withdrew none of —
the closure and the stamp are guarded by `IS NULL` predicates, but the receipt is collected before
either runs. A `Message` is written once and never recomputed, so that duplicate would be
permanent.

### 6. The message keeps ADR-0074's form and gains its own sentence

One coverage message per withdrawn `Seed`, not one per subject. ADR-0074's rule is unchanged: a
scope and a count, no comparison and no rows. `Census` stays nil.

**The dangling scope key is already licensed.** `message.Narrowing` documents it: "a scope later
narrowed out of existence is a dated fact, not a broken join, because a Message is written once
and never recomputed." A withdrawn `Seed` is that case, arriving for the first time.

**The rendered copy is not.** `NarrowingReceipt.Scope` is documented as "the Seed scope the
message fires at — the only object that survives the act", and this act is the one that breaks
that assumption. `Scope` and `Removed` are the same string here, so the existing headline renders

> 10.0.0.0/24 narrowed · 10.0.0.0/24 excluded · 3 subjects withdrawn · 7 timelines taken out of
> the estate

It says *excluded* for an act that declared no exclusion, and *narrowed* for a scope that is
gone. `message` therefore gains a withdrawal headline beside the narrowing one, on the same
receipt type and the same coverage class:

> 10.0.0.0/24 withdrawn · 3 subjects withdrawn · 7 timelines taken out of the estate

A `Message` is written once and never recomputed, so a false sentence written here is false for
as long as the corpus is retained. That is why the copy is a model decision and not a detail.

### 7. Two adjacent acts are named and excluded

Both are real gaps and both are out of this ADR's scope, so that a later session does not read
their absence as a ruling.

- **Withdrawing a custody extension.** `SetCustodyExtension(false)` narrows the aperture, and §4
  above deliberately holds addresses open on the strength of that extension. Turning it off
  should release exactly those addresses, and nothing closes them.
- **The name limb.** `foldEstateTransitions` runs over the Names a batch observed. Withdraw a
  name `Seed` and its Names stop being enumerated, so the fold never revisits them. The Name side
  keeps a per-subject `declared-input` message rather than an aggregate, so it is a different
  message contract and a different review.

## Consequences

- Withdrawing an address `Seed` becomes enforcing for the first time. The spans it alone held
  close on the next completed job, and the operator gets a coverage message naming the scope and
  the count.
- The removal toast must stop promising that existing subjects keep their citations. It states
  the opposite of the fix.
- **A migration.** `seed_withdrawal` is a new table, so the change ships a goose migration
  numbered above `origin/main`'s current maximum, plus a new `sqlc` query and a regenerated
  `internal/db`.
- The delete handler needs a transaction. It currently issues one statement and holds none.
- `message` gains a second headline on an existing receipt type. No new message class, no new
  cause, no census.
- The model now has one declared input read from a record of a past act. §4's live-corpus
  survivors are what keep that from drifting out of agreement with live truth, and they are the
  first thing to check if this act ever misbehaves.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Sweep the open address spans each fold** — close any address no `Seed` covers and no resolution cites | No mover. `composeAddressWithdrawals` already refuses to close a span it cannot attribute to a declared act, because the operator cannot trace it back. A sweep also cannot tell a `descoped` departure from an `uncited` one, so it would blur two of the three closure grounds ADR-0087 keeps distinct. It needs no migration, which is its only merit |
| **Convert the delete into a declared exclusion** over the withdrawn CIDR, and let #1032's machinery act | It records a claim the operator never made. An exclusion says *forbid this*, not *I stopped declaring this*. It outlives the `Seed`, holds a unique index on the CIDR, and would block re-declaring the same scope later. The receipt would name a mover that does not exist |
| **Soft-delete the `Seed`** with a `deleted_at` column | Every reader of `seed` would have to filter, and a missed filter re-admits a withdrawn scope silently. It also changes the FK cascade that `TestSeedForeignKeysCascadeOnDelete` guards. A separate table buys the same mover with no change to any existing read |
| **Close the spans in the delete handler** | Ruled out by ADR-0111 and already ruled out by ADR-0133 §8.1 for the exclusion act. A closure cites the folding batch and a web handler holds none |
| **Close the spans and fire no message** | The operator loses the only trace of an act that removed subjects from the estate. ADR-0074 exists to make exactly this act legible, and `message.Narrowing` was written for it |
| **A fourth `descoped` ground-bearer** for removal, beside narrowing | It would split one ground into two names and buy nothing. The ground is what the closure records, and the ground does not move |
| **Spend every tombstone the fold reads** (#1040 review) | §5.1. Two of §4's survivors are transient, and the tombstone is the only mover the act will ever have. Spending it early strands the addresses this ADR exists to release |
