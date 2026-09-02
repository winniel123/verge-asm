# ADR-0135: A name `Seed` withdrawal states one act, and its tombstone carries the domain alone

- **Status:** Accepted
- **Date:** 2026-09-01
- **Ticket:** [#1045 Withdrawing a name `Seed` writes no withdrawal](https://github.com/winniel123/verge-asm/issues/1045)
- **Follows:** [ADR-0134](./0134-a-seed-withdrawal-is-recorded-by-a-tombstone-because-the-mover-does-not-survive-the-act.md) §7, which named this limb and excluded it
- **Constrained by:** [ADR-0133](./0133-an-address-exclusion-is-a-limb-of-the-custody-derivation.md) §8.1 (a withdrawal is driven from the declaration side), [ADR-0074](./0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md) (a narrowing fires at the scope and carries a count, not a census), [ADR-0111](./0111-a-span-cites-the-batch-that-folded-it.md) (a closure cites its batch), [ADR-0087](./0087-a-closure-records-the-ground-it-rests-on-and-there-are-three-grounds.md) (a closure records no actor)

## Context

ADR-0134 made an address `Seed` withdrawal enforcing and named the name limb as a
gap it was deliberately leaving open. This ADR closes it.

`foldEstateTransitions` iterates the Names **this batch observed**. The dns Scan's
resolution set is the live name-`Seed` domains unioned with the distinct
CT-admitted names (`mergeResolutionNames`). Withdraw a name `Seed` and the domain
leaves the first limb, while the foreign-key cascade takes its `admitted_name` rows
out of the second. Those Names stop being enumerated, so no observation about them
ever arrives again, so the fold never revisits them and never closes them. Their
timelines stay open for ever, with no `Seed` covering them and no declaration to
point at.

This is ADR-0133 §8.1's shape a third time: **a withdrawal scoped to observed
subjects can never reach a subject the act stopped enumerating.**

The decision function is not the gap. `decideNameDeparture` already takes a
seed-covered flag and already returns `descoped`. The gap is the iteration set of
the fold that calls it.

**Three questions had to be ruled before any code, and two of them are questions
ADR-0133 §8.1 and ADR-0134 §7 left open on purpose.**

## Decision

### 1. The act states itself once, as an aggregate `Narrowing`

Withdrawing a name `Seed` writes **one** coverage-class `message.Narrowing` per
withdrawn domain, carrying the subjects and the timelines as two factors. It does
not write a `declared-input` message per withdrawn Name.

ADR-0133 §8.1 kept a per-subject `declared-input` on the Name side and an aggregate
on the address side, and that looks at first like a rule about the subject kind. It
is not. The reason it gave was arity: **a name exclusion withdraws one Name**, so
there is no aggregate for it to state, while an address exclusion withdraws a
population. A name `Seed` withdrawal is the first Name act that removes many
subjects in one operator act, so it falls on the other side of the same rule.

ADR-0074 then settles it. A row per withdrawn subject *is* the census the receipt
exists to replace. Four hundred Names under one withdrawn domain would produce four
hundred permanent messages for one click.

The copy needs no new constructor. `message.PreviewSeedWithdrawal` already renders
the withdrawal sentence from a scope and two counts, and a name `Seed`'s display
scope **is** its domain exactly as an address `Seed`'s is its CIDR:

> example.com withdrawn · 3 subjects withdrawn · 7 timelines taken out of the estate

`coveringExclusionKey` and the per-subject `declared-input` path are untouched. A
name **exclusion** still withdraws its one Name and still says so per subject.

### 2. The tombstone grows a name limb, taking `seed`'s own shape

`seed_withdrawal` gains `kind` and `name_domain`, `address_cidr` relaxes to
nullable, and a `seed_withdrawal_shape` CHECK requires exactly one scope column
populated for the kind. That is `seed`'s own shape (`seed_shape`, 00003), and a
tombstone is the counterpart act to a declaration, so the two are recorded the same
way.

`WithdrawSeed` writes the row for both kinds in the same statement as the delete,
so no path can leave a withdrawn scope of either kind with no mover.

The existing address reads gain `kind = 'address'` and the new name reads carry
`kind = 'name'`, so each fold claims only its own rows and the two never lock each
other out through `FOR UPDATE SKIP LOCKED`.

A separate `name_seed_withdrawal` table was the alternative. It would have kept
24700's `NOT NULL` intact and cost a duplicate `consumed_at` machinery, a second
spend query and a second set of rules to keep in agreement — for a row that records
the same act about the other limb.

### 3. Two survivors, and both are decided in Go

A Name does not leave while any limb still holds it.

- **A live name `Seed` still covers it.** Read from the `Seed` corpus at fold time,
  never from the tombstone. As on the address side this is load-bearing twice: it
  settles a Name a second declared `Seed` also covers, and it settles
  **re-declaration** — withdraw `example.com`, declare it again, and the Names
  re-enter through the `Seed` limb while a stale tombstone still names the domain.
- **A surviving `Seed`'s admission still enumerates it.** `admitted_name` rows
  carry their own `seed_id`, so the cascade takes only the withdrawn `Seed`'s
  admissions. A Name a surviving name `Seed` also admitted keeps its row, stays in
  the resolution set, and is still walked every batch. #1045's brief observed that
  this survivor works today by the shape of `ListAdmittedNames`, which is global
  and not filtered by `Seed`, and asked for it to be **stated**. It is stated here,
  and now it is also enforced.

Without the second survivor the Name closes `descoped`, the next batch reopens it
and the batch after closes it again — a departure and a coverage message every
cadence, over a Name the estate never stopped measuring. That is ADR-0134 §4's
third-survivor argument, arriving on the Name side through a different limb.

**Both tests run in Go, and both must.** Each has to use the same key function the
resolution set uses — `nameSeedCovered` over the `Seed` corpus, `resolutionNameKey`
over the admitted names. A SQL survivor test keying names its own way would drop a
Name the estate still walks, or hold one it stopped walking. This is ADR-0134 §4's
reason for deciding the custody survivor through `Derive` itself: the membership
decision and the enumeration must read one function and cannot be allowed to
disagree.

**The address limb's third survivor has no Name counterpart, and neither does
`CONTEXT.md`'s resolution-citation limb.** `custody.Estate` derives *addresses*,
and its extension lives on a name `Seed`, so withdrawing that `Seed` takes the
extension with it and leaves nothing standing to test. "Unless a current resolution
still cites it" is likewise an `Address` rule — an address is in the estate while a
resolution cites it, and a Name is the thing that resolves rather than a thing
resolutions cite.

The fold closes **the Name's own open spans**, with no fan-out to a subordinate
subject. That is exactly what `foldEstateTransitions` closes for a departing Name,
and the two are one closure reached by two routes. The address limb fans out to
`service` and `endpoint` because an `Address`'s subordinates are keyed by the
address itself; a Name's are not.

### 4. The cascade costs the fold nothing, so the tombstone carries the domain alone

Deleting the `Seed` cascades away its `admitted_name` and its `zone_file` rows, so
the evidence that **admitted** the Names is gone in the same transaction as the
declaration. It was open whether the tombstone therefore had to carry more than the
domain — a snapshot of the Names the `Seed` had admitted.

It does not. The cascade touches no `span`, and `span` is where the fold reads its
candidates. **The evidence that admitted a Name is not the record of its
timelines.** The candidate query walks open name spans under the withdrawn domain,
which is a set the delete cannot reach.

The cascade is not merely harmless, it is what makes survivor two correct. After
the delete, `admitted_name` holds exactly the admissions of the `Seed`s that
survive, so "is this Name still admitted" and "does a surviving `Seed` still hold
it" are the same question.

A snapshot would also have written an unbounded row set inside a click handler —
the admitted-name cap is ten-thousand-scale — and given the fold a second corpus to
reconcile against live spans.

### 5. When it runs, and when the tombstone is spent

ADR-0134 §5 unchanged, for the reasons it gave. A closure cites the folding batch
and a web handler holds none, and a withdrawn scope stops being enumerated. The
withdrawal lands on the **next completed job**, so an estate running no jobs holds
its spans open until one completes.

The name fold runs **after** `foldEstateTransitions`, so a Name that fold already
closed is not open and neither counts it twice.

A name tombstone is spent **late**, once no open timeline is left under its domain,
and not on the first fold that reads it. Both name survivors are transient in the
same way ADR-0134 §5's are: a withdrawn domain can be declared again, and a
surviving `Seed`'s next CT poll can re-admit a Name whose admission the cascade
removed. A tombstone is the only mover its act will ever have, so spending it while
its ground is still held would strand those Names open for ever.

**Exhausted is not enough on its own, and here the name limb parts from the address
one.** A dns Scan fans out one job **per vantage**, and every job freezes the whole
resolution set into its own scope gate — `authorizedScope.admits` reads the job's
own name list, never the live corpus. So a job enqueued before the withdrawal still
admits observations about the withdrawn domain when it completes after it, and the
value fold opens a fresh resolution span for a Name this act just closed.

Spending on the first exhausted fold would strand exactly those spans. The batch
that closed vantage 1's timeline would consume the mover, and vantage 2's job would
then re-open its own with nothing left to close it — this table's own leak,
reintroduced by the fan-out.

The address twin is safe from this by accident, not by rule. An address re-opens
only through a resolution citing it, and an address a current resolution cites is
dropped from the candidate set, so its span stays open and its tombstone stays
pending.

So a name tombstone additionally waits for the **dns queue to drain**. A job fanned
out after the withdrawal cannot carry the domain — `fanOutDNS` reads the live seed
domains and the live admitted names, and the cascade removed the admissions — so
once no dns job is outstanding, none can re-open the ground. Waiting on every dns
job rather than only the older ones is deliberately conservative: a retry enqueues a
**fresh row carrying the old frozen spec**, so neither its id nor its `created_at`
can tell a stale job from a current one.

**A re-declared domain spends immediately**, whatever is in flight. Survivor one
drops every candidate once a live `Seed` covers the ground again, so the tombstone
can never close anything and its exhaustion test could never come true — it would
stay pending for ever and cost every completed job the candidate read. Live truth
settles re-declaration at the spend exactly as it does in the fold, and a later
withdrawal of the re-declared `Seed` writes its own row.

There is no `family()` guard to carry across. That one exists because the address
candidate query reads IPv4 subject keys alone and would call an IPv6 tombstone's
ground empty when it is not. The name subtree test matches every name the candidate
query matches, so the read and the spend agree on what is left.

## Consequences

- Withdrawing a name `Seed` becomes enforcing. The timelines it alone held close on
  the next completed job with the `descoped` ground, citing the folding batch.
- The removal toast's two limbs collapse into one sentence. Both now perform the
  withdrawal they describe, and the name limb stops promising that existing
  subjects keep their citations — which was a plain statement of the bug.
- **A migration.** `seed_withdrawal` grows two columns, relaxes one, and swaps its
  partial index for a `(kind, id)` one. `address_cidr` becomes `*netip.Prefix` in
  the generated model, so `coveringSeedWithdrawal` skips a nil as an exclusion's
  CIDR does.
- `CONTEXT.md`'s `Name` entry gains the declared limb of its departure rule. It
  stated only the measured one, and an exclusion has closed a Name `descoped` since
  #722.
- No new message class, no new cause, no census, and no new constructor.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **One `declared-input` message per withdrawn Name**, reading ADR-0133 §8.1 literally | Its stated reason was arity, not subject kind: a name exclusion withdraws ONE Name. ADR-0074 forbids a row per subject for one act, and the volume here is the case it was written against. The message would also have to name a Source that the act destroyed |
| **A separate `name_seed_withdrawal` table** | Duplicates the `consumed_at` machinery, the spend rule and the FK shape for a row recording the same act about the other limb. `seed` itself is one table with two limbs, and the tombstone is its counterpart |
| **Snapshot the admitted Names onto the tombstone** so the act records exactly what it withdrew | The candidates come from `span`, which no cascade touches, so the snapshot answers a question nobody asks. It writes an unbounded row set inside a click handler and gives the fold a second corpus to reconcile |
| **Test the survivors in SQL**, in the candidate query | The tests must use the same key functions the resolution set uses, or "does it survive" and "does it stay enumerated" can disagree — dropping a Name the estate still walks, or holding one it stopped walking. ADR-0134 §4 made the same call for the custody survivor |
| **Fan the closure out to the Endpoints beneath the withdrawn Names** | `foldEstateTransitions` closes a departing Name's own spans and nothing else. Two routes to one closure must remove one shape of ground, and widening it here would make the same departure mean two different things |
| **Keep the name limb enumerated and let the ordinary fold decide** | It cannot. The act stops the enumeration in the same transaction, which is the whole bug. The fold would need a background sweep of the estate, which ADR-0133 §8.1 and ADR-0134 both rejected for want of a mover |
