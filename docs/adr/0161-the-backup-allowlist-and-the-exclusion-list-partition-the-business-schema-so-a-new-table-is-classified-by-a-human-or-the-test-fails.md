# ADR-0161: the backup allowlist and the exclusion list partition the business schema, so a new table is classified by a human or the test fails

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1367 ADR gaps: cmd/web backup.go](https://github.com/winniel123/verge-asm/issues/1367), gap 2
- **PR that deleted the comment:** [#1366](https://github.com/winniel123/verge-asm/pull/1366)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Extends, and withdraws nothing in:** [ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md) §1, which rules that the backup path reads business tables only and calls that a rule of the export. It rules the allowlist half and says nothing about a denylist
- **Sibling of:** [ADR-0160](./0160-a-backup-redacts-a-reversible-cleartext-credential-and-carries-a-hash-or-an-externally-keyed-ciphertext-and-restore-re-applies-the-same-redaction.md), which rules the columns of a table this rule admits

## Context

Three hand-written lists decide what the in-app archive carries.

| List | Site | Entries |
| --- | --- | --- |
| `backupTables` | [`cmd/web/backup.go:19`](../../cmd/web/backup.go) | 32 |
| `backupExcluded` | [`cmd/web/backup.go:59`](../../cmd/web/backup.go) | 8 |
| `knownBusinessTables` | [`cmd/web/backup_test.go:13`](../../cmd/web/backup_test.go) | 40 |

`backupTables` is ordered FK-parents-first, because the restore replays it in order.
`backupExcluded` is a map from table name to the reason that table is left out. Every value is a
sentence a reader can weigh, not a marker.

`TestBackupTablesPartitionSchema` ([`cmd/web/backup_test.go:24`](../../cmd/web/backup_test.go))
asserts both directions. Every name in `knownBusinessTables` appears in exactly one of the two
lists. Every name in either list appears in `knownBusinessTables`. So 32 plus 8 equals 40, and the
test fails on any other arithmetic.

**The rule the deleted comment carried was the reason for the second list, not the fact of it.**
A denylist buys nothing on the export path. `dumpBackupTable` reads `backupTables` and never reads
`backupExcluded`. Deleting `backupExcluded` would change no byte of any archive. It exists to make
the two lists a partition, and the partition exists to stop a new table from being carried or
dropped by default.

**Nothing on disk states it.** ADR-0124 §1 rules the allowlist: *"the backup path reads business
tables only"*, and *"a rule of the export, not an accident"*. It never mentions a denylist and never
rules that an unclassified table is an error.
[`docs/guides/backup-and-restore.md`](../guides/backup-and-restore.md) describes the exclusions to
an operator and gives the same allowlist ground.

The comment cited `backup_test.go` alone. Under [`comment-policy.md`](../spec/comment-policy.md)
§8.3 a test file is not in the suppressing set. The test enforces the rule and does not record the
decision behind it.

## Decision

> **`backupTables` and `backupExcluded` partition the business schema. A table is in exactly one of
> them, an exclusion carries its reason as prose beside the name, and
> `TestBackupTablesPartitionSchema` fails when the partition breaks. There is no default. A session
> that adds a business table classifies it, and a session that cannot classify it does not add it.**

Four limbs.

### 1. Classification is a human act with no default in either direction

Two defaults were available and both are refused.

*Dump anything the schema holds* carries a new table off-host on the day it is created, before
anyone has asked what is in it. That is how a secret, a raw byte corpus or a transient queue leaves
the instance without a decision.

*Carry only what the allowlist names, and ignore the rest silently* loses estate data. A table
someone forgot to list is absent from every archive taken after it was created, and the operator
learns this at the restore.

The failure modes point opposite ways, so no default is safe for both. The partition removes the
default and asks the question instead.

### 2. The exclusion carries its reason, and the reason is what makes the list reviewable

`backupExcluded` is a map to prose, not a set of names. Each value states why the table is dropped
and what the operator loses. `session` lapses on restore. `queue_job` would restore stale `running`
rows. `transcript` is ciphertext under a volume key the archive must not carry.

A bare set would record the same eight decisions and none of the eight grounds, and a later session
could not tell a deliberate exclusion from an oversight.

### 3. The test is the check, and it checks the three lists against each other

`TestBackupTablesPartitionSchema` reads the three literals in the package. It runs in `test`, which
is a required check on `main`, so a broken partition blocks the merge.

**The check does not read the live schema.** `knownBusinessTables` is a hand-written literal, and no
code compares it with `db/migrations/`. So the test catches a table that reaches any one of the
three lists and not the other two. It does not catch a table that reaches the database and none of
the three.

This is stated rather than hidden, because the mechanism is weaker than the rule. **The migration
author carries the step the test cannot.** A migration that creates a business table adds the name
to `knownBusinessTables` and to exactly one of `backupTables` and `backupExcluded`, in the same
change.

### 4. A place in `backupTables` is a place in the restore's replay order

`backupTables` is ordered, and `applyRestore` replays the archive in the order the file
carries ([`cmd/web/restore.go:276`](../../cmd/web/restore.go)). A table added to the allowlist goes after
the tables its foreign keys point at. Classification therefore answers two questions and not one.

## Consequences

- **`cmd/web/backup.go` gains this ADR's citation** on the comment that already states the rule. No
  behaviour changes.
- **The migration author gains a named step.** Limb 3 states it, because the test cannot enforce it.
- **A table that nobody can classify is a signal, not a blocker.** A table that fits neither *the
  operator would want this back* nor *this is transient* has usually not settled what it is. The
  rule sends that question back to the schema change.
- **`TestBackupTablesPartitionSchema` gains no new assertion.** It already checks what it can check
  without a database.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Allowlist and denylist are code-structure
  terms, not domain terms.
- **[ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md)
  is unaltered.** Its §1 rules the allowlist and this ADR adds the partition around it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep the allowlist and delete `backupExcluded`** | The export path never reads it, so the deletion is invisible in every archive and free at review. It is also the whole mechanism. Without the second list the partition cannot be asserted, and a new table falls through to whichever silent default the reader assumes |
| **Dump every table the schema holds, minus a small exclusion list** | Carries a new table off-host on the day it is created. `transcript` is the measured case: verbatim job output, ciphertext under a volume key, and the archive must not carry it ([ADR-0126](./0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md)). A default that ships it and waits for someone to notice inverts ADR-0124 §1 |
| **Keep the allowlist alone and let an unlisted table be skipped silently** | Loses estate data with no signal. The archive is complete by the operator's reading of it, and the gap appears at the restore, which is the worst moment to find one |
| **Derive `knownBusinessTables` from the live schema in the test** | The `cmd/web` tests run against a hand-written fake store and no database. Reading `information_schema` would put a Postgres instance behind a unit test that has never needed one, for a list that changes a few times a year |
| **Parse `db/migrations/` in the test to build the third list** | Writes a SQL parser to check a list of forty names. `sqlc` already parses the schema for a different purpose, and its output names Go types rather than the table set the archive reasons about. The cost lands on every future migration syntax the parser does not know |
| **Enforce it with a lint rule or a `commentlint` check instead of a test** | The fact is a set relation between three Go literals in one package, which is what a test states directly. A lint rule would re-implement the same comparison outside the compiler and outside the required `test` check |
| **Record the rule in [`docs/guides/backup-and-restore.md`](../guides/backup-and-restore.md)** | The guide is written for an operator taking a backup. This rule binds a session writing a migration, who has no reason to open an operator guide |
| **Leave the rule in the test and file nothing** | [`comment-policy.md`](../spec/comment-policy.md) §8.3 does not admit a test file as a suppressing source, and the reason is exactly this case. `TestBackupTablesPartitionSchema` states that the partition holds. It does not state that no default is safe, which is the decision |
