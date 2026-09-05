# ADR-0174: the backup allowlist bounds every identifier the restore interpolates, because the restore's input is operator-supplied

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1363 ADR gaps: cmd/web messages.go, restore.go and drift.go](https://github.com/winniel123/verge-asm/issues/1363), gap 2
- **Sweep PR that deleted the comment:** [#1365](https://github.com/winniel123/verge-asm/pull/1365)
- **Rests on:** [ADR-0161](./0161-the-backup-allowlist-and-the-exclusion-list-partition-the-business-schema-so-a-new-table-is-classified-by-a-human-or-the-test-fails.md), which rules who is on `backupTables` and in what order. This ADR adds no name and moves none
- **Rests on:** [ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md) §1, which makes reading business tables only *a rule of the export*. That is why a list exists at all
- **Not bound by:** [ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md):79, whose injection row rules *"Any command the **UI** assembles… is a command-injection surface over the **host**"* — a different interpreter and a different input
- **Not bound by:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md), which rules secret custody. The hazard here is a statement, not a leak

## Context

`backupTables` (`cmd/web/backup.go:19-53`) is a hand-written list of 32 business tables. On the
**export** side it is a read filter: `streamBackup` ranges it (`:155`) and `dumpBackupTable`
interpolates each name into `SELECT to_jsonb(t) FROM "…" t` (`:168`). That string is a package-level
Go literal; no request reaches it.

**The restore inverts the direction of trust, and nothing on disk says so.** Its input is a file the
operator uploads (`cmd/web/restore.go:147-158`), staged in memory (`:184-190`) and replayed later.
Every table name in it — the manifest's `Tables` array and every row line's `table` field — is
attacker-authorable in the sense that matters: a `.ndjson` file is trivially hand-edited, and
restoring an archive obtained from elsewhere is the workflow the Restore card offers.

Three of those names reach SQL by interpolation, because **a table name cannot be a bound
parameter.** `$1` names a value, and PostgreSQL resolves a relation before any parameter binds. The
restore has no protocol-level defence available, so the bound must be a list.

The reason was written once, in a block comment on `applyRestore`, and the §8 sweep deleted it
(#1365), leaving two lines that state the mechanism without the rule (`cmd/web/restore.go:303`,
`cmd/web/backup.go:167`) and no citation, because there was nothing to cite. Nothing under
`docs/adr/`, `docs/spec/`, `docs/guides/`, `docs/research/` or `CONTEXT.md` rules SQL identifier
interpolation. `docs/guides/backup-and-restore.md:74-76` states the allowlist as an **export**
invariant with leakage as its reason. ADR-0161 rules the list's membership and order. Neither says
the list is load-bearing for the restore's SQL, which is the claim that makes it unrelaxable.

`gosec` does not hold the property either: the job runs `-severity high -confidence high`
(`.github/workflows/ci.yml:494`) and the tree is green with both concatenations in place and no
`#nosec` on either.

## Decision

> **Every table identifier the restore interpolates into SQL is checked for exact membership in
> `backupTables` immediately before the interpolation that consumes it. The check happens at every
> such site, with no exception for a statement that only truncates, and it is membership — never a
> pattern, never a catalogue lookup, never quoting alone. `backupTables` is therefore a security
> bound as well as a content list, and it may not be relaxed, widened by pattern, or generated at
> run time without re-examining this surface.**

### 1. Three interpolations, three gates

`preflightArchive` **interpolates nothing.** It parses, validates and counts
(`cmd/web/restore.go:59-117`); its `backupAllowed` loop at `:74-78` refuses a bad archive before the
operator is offered a confirm dialog. It guards no statement.

`applyRestore` interpolates at three places, and each is preceded by a check:

| Interpolation | Gate |
| --- | --- |
| `"`+t+`"` into the `TRUNCATE … RESTART IDENTITY CASCADE` list (`restore.go:269`, executed `:271`) | `backupAllowed(t)` over every `man.Tables` entry, `restore.go:248-252` |
| `INSERT INTO "` + row.Table + `"` (`restore.go:304`) | `backupAllowed(row.Table)`, `restore.go:289-291` |
| `NULL::"` + row.Table + `"` inside `jsonb_populate_record` (`restore.go:305`) | the same check, same iteration |

**The `TRUNCATE` gate is not a courtesy.** It takes a list, honours `CASCADE`, and is the most
destructive statement the binary issues — an unbounded identifier there destroys a relation outside
the estate rather than reading one. It gets no shorter argument than the insert path.

### 2. The gate is membership, and a pattern would not do

`backupAllowed` (`cmd/web/restore.go:119-126`) is a linear scan for `t == table` over
`backupTables`. Exact equality: no case folding, no trimming, no unquoting, no prefix.

A pattern — `^[a-z_][a-z0-9_]*$`, say — admits `pg_shadow`, `goose_db_version`, `session` and
`transcript`. Each is well formed and none is in the estate the archive describes; under
`TRUNCATE … CASCADE` a forged manifest could then destroy the migration ledger or the encrypted
transcript corpus. **The set the restore may touch is finite, enumerated and already written down**,
so a predicate admitting a superset of it gives up the bound and buys nothing.

### 3. The apply pass re-validates and never inherits preflight's verdict

Preflight and apply are separate requests. Between them the archive sits in `restoreStage`
(`cmd/web/restore.go:184-190`, read at `:203`), and `applyRestore` re-parses the manifest from those
bytes (`:236-252`) rather than trusting `restorePreflight.Tables`. It also gates **every row line**
at `:289`, which preflight never inspects — preflight reads row lines only to count open `span`
subjects and `continue`s past every other table (`:94-96`).

That duplication is the ruling, not redundancy to tidy away. `applyRestore` composes the SQL, so
`applyRestore` is where the check lives. Hoisting the gate into preflight alone re-opens the surface.

### 4. `jsonb_populate_record` forces the interpolation, and quoting is not the bound

The issue describes this correctly. `INSERT INTO "t" [OVERRIDING SYSTEM VALUE] SELECT * FROM
jsonb_populate_record(NULL::"t", $1::jsonb)` (`cmd/web/restore.go:304-305`) passes the whole row as
one `jsonb` **value** in `$1`, and `NULL::"t"` names the live rowtype that decodes it. The restore
therefore writes no per-column decoder: timestamps, arrays, `bytea` and nested `jsonb` round-trip
through the type Postgres already holds, and `redactBackupRow`'s key reordering
(`cmd/web/backup.go:96-97`) is immaterial. The price is that the rowtype is named by identifier, and
an identifier cannot be a parameter. **What removes 32 hand-written decoders is what forces the
interpolation.**

The identifier is double-quoted at `:269`, `:304` and `:305`, and an embedded `"` is doubled at none
of the three. **That is safe only because §1's gate ran first.** Quoting is defence in depth over a
value already proved to be one of 32 literals, never the bound.

### 5. A neighbouring clause in ADR-0124 is unbounded, and this ADR does not rule it

`docs/adr/0124…:38` requires the archive framing to be settled *"against the requirement that it be
**forward-restorable across a migration bump**"*, repeated at `:66`. Code and guide do the opposite:
`restorePreflight` refuses on `pf.SchemaVersion != running` (`cmd/web/restore.go:172-176`), and
`docs/guides/backup-and-restore.md:149-151` states the refusal as the operator contract — *"a
restore **across a migration bump** is caught at preflight."*

**PR #1420 did not bound this clause.** The note it added to ADR-0124 sits at `:28-35` and bounds
*"No secret"* under ADR-0160; `:38` carries none. Under
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) a superseded
requirement is withdrawn at the site that specifies it, so `:38` and `:66` each need a dated note.
**This ADR writes neither and does not rule on forward-restorability. That is a follow-up.**

### 6. What is not yet held by a test

`TestPreflightArchiveRejectsUnknownTable` (`cmd/web/restore_test.go:64-70`) covers the preflight gate
with a forged manifest naming `pg_catalog_pg_proc`. **`applyRestore` has no test at all** — no test
in `cmd/web` assigns `s.pool` — so both gates that guard real SQL are unexercised.

One is testable today with no code change, because the manifest gate at `:248-252` returns before
`s.identityTables` touches the pool at `:254`: **`TestApplyRestoreRefusesForgedManifestTable`** — an
archive whose manifest `tables` array holds `"goose_db_version"`; call
`(&server{}).applyRestore(ctx, archive)` on a nil pool and assert `errRestoreUnknownTbl`. A
regression that dropped the gate panics on the nil pool rather than returning, so the assertion is
sharp in both directions.

The row gate at `:289` sits inside the transaction and needs a database. The smallest enabling
change is to lift the row-table scan into a pure helper called before `s.pool.Begin`, then add
**`TestApplyRestoreRefusesForgedRowTable`** — a manifest listing only allowlisted tables plus one row
line with `"table":"session"`, asserting `errRestoreUnknownTbl`. Not applied here.

## Consequences

- **`backupTables` gains a second job.** ADR-0161 makes it a classification; this makes it a bound.
  Sorting it, filtering it, or building it from `pg_catalog` at boot must re-argue this ADR, not
  only ADR-0161 §4's ordering. The list stays a Go literal a reviewer can read.
- **The two surviving mechanism comments gain citations**, and the reason moves here.
- **Adding a table costs nothing extra.** Append to the literal, in FK-parent order (ADR-0161 §4).
- **A forged manifest is refused, not partially applied.** The replay runs in one transaction
  (`cmd/web/restore.go:259-263`, committed `:330`), so a foreign row line rolls back everything.
- **The restore stays deliberately schema-specific.** Letting the archive describe its own
  destination re-opens this surface.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A regexp identifier check** — accept any `^[a-z_][a-z0-9_]*$` and quote it | It admits every well-formed name in the database, including `goose_db_version`, `session`, `transcript` and every `pg_catalog` relation. Under `TRUNCATE … CASCADE` that is destruction of state outside the estate; under `INSERT` it is a write into a table the archive has no business describing. The legal set is 32 known strings, so a predicate matching millions is strictly weaker for no gain, and it drifts silently — a rename in `backupTables` still matches |
| **A `pg_catalog` lookup at run time** — accept any relation in `public` | It answers *"does this exist?"* when the question is *"is this ours to overwrite?"* It is self-defeating on the partition ADR-0161 rules: `session`, `queue_job` and `transcript` all sit in `public` and are all deliberately excluded, so the catalogue readmits exactly what `backupExcluded` exists to keep out. It also adds a query and a failure mode inside the restore transaction, and makes the bound depend on live database state rather than a reviewed literal |
| **Quote with `pgx.Identifier{table}.Sanitize()` and drop the allowlist** | Sanitising answers a different question. It makes the identifier *syntactically* safe — the statement parses and names one relation — and says nothing about *which*. `pgx.Identifier{"pg_shadow"}.Sanitize()` is a perfectly sanitised identifier for a table the restore must never touch. Escaping bounds the parse; only membership bounds the target |
| **Validate once in `preflightArchive` and trust the staged archive at apply** | They are separate requests over a slice in `s.restoreStage`, and preflight never inspects a non-`span` row line (`cmd/web/restore.go:94-96`), so the row gate has nothing to inherit. A check anywhere but beside the interpolation makes its correctness depend on a call graph rather than on adjacency |
| **Skip the gate on the `TRUNCATE`, since it names no columns** | `CASCADE` reaches referencing tables the archive never named, so an unbounded identifier there is worse than an unbounded `INSERT`. "It only truncates" is the argument that deletes the check that matters most |
