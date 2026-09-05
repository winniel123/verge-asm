# ADR-0210: a variant's own scalar rides its typed-outcome object and earns no column on the shared table

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1321 ADR gaps: internal/queue (#1199)](https://github.com/winniel123/verge-asm/issues/1321), gap 2
- **PR that deleted the comment:** [#1327](https://github.com/winniel123/verge-asm/pull/1327)
- **Rests on:** [ADR-0126](./0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md), which rules that each variant carries **its own typed outcome** and that the verbatim streams are `bytea` columns. It rules the value shape. It does not rule where a scalar that only one variant carries lands
- **Bounded by:** [ADR-0126](./0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md)'s at-rest sealing clause, as scoped by this issue's manifest. The outcome object is **not** sealed, so §4 makes that clause the gate a new scalar passes
- **Sibling of, and not ruled by:** [ADR-0209](./0209-a-closed-union-we-author-refuses-an-unknown-member-and-writes-no-row.md). That ADR rules the default branch of a reader of this same union. This ADR rules where a member's own field is stored. Neither implies the other
- **Rests on:** [`raw-job-output.md`](../spec/raw-job-output.md) §1.2 and §1.4, which show the variant table and the column groups. §1.2's table names the three typed outcomes and rules nothing about why a column was refused

## Context

`internal/queue/transcript.go:93` carried this in declaration position, until #1327 deleted it:

```go
// encodeZoneOutcome encodes a zone restate's typed outcome as the JSONB object the
// transcript row stores: {"kind":"parsed","restated":N} / {"kind":"decode-error",
// "restated":N,"text":T} (spec §1.2, §1.3). The restated count rides the object so
// the §6 read handler reads it without a separate column, mirroring how the prober
// outcome carries its exit code. The union is closed, so an unknown member is a bug.
```

`encodeCTOutcome` at `internal/queue/transcript.go:156` stated the same rule a second time, for the
request URL — *"so the §6 read handler reads it without a separate column, mirroring how the zone
outcome carries its restated count"*. Two comments, each pointing at the other as its precedent, and
no source. That is #1321's gap 2.

**Three variants, three scalars of their own, and zero columns.**

| Variant | Its own scalar | Where it lands | Read at |
| --- | --- | --- | --- |
| `prober` | exit code, or signal name | `{"kind":"exited","code":N}`, `{"kind":"signalled","signal":S}` | `cmd/web/rawoutput.go:180` |
| `zone` | the restated record count | `{"kind":"parsed","restated":N}` | `cmd/web/rawoutput.go:206` |
| `ct` | the request URL | on **every** arm of the CT outcome | `cmd/web/rawoutput.go:230` |

**The columns that do exist are role columns, and each is reused across variants.** Migration
`23700_transcript.sql` gives the table eleven columns. Three are the common frame (`kind`,
`duration_ns`, `captured_at`), one is the key, one is the union tag, one is the outcome object, one
is the truncation marker, one is `created_at`, and three are stream roles:

| Role column | `prober` holds | `ct` holds | `zone` holds |
| --- | --- | --- | --- |
| `stdout` | the prober's stdout | the verbatim HTTP response body | the joined list of skipped records |
| `stderr` | the prober's stderr | NULL | NULL |
| `sent_scope` | the exact `JobSpec` bytes sent to stdin | NULL | NULL |

So the schema already answers *what is common* with a column and *what differs* with reuse plus the
outcome object. A `restated INT` column would be NULL on two variants of three on the day it landed,
and each further variant would add another such column to a table every variant shares.

**The union tag and the outcome object are one unit, and the migration says so.** `variant` is
`CHECK`-constrained to the three names and `outcome` is `NOT NULL` JSONB. The tag decides which
decoder reads the object. Putting a variant's scalar in a column would split the variant's shape
across two places and break that: the tag would no longer determine which fields of the row are
meaningful, and a reader would have to know, out of band, that `restated` is populated only when
`variant = 'zone'`.

**One consequence of this shape is not visible from either comment, and it is load-bearing.**
`stdout`, `stderr` and `sent_scope` are sealed at rest (`transcript.Seal`, ADR-0126). **`outcome` is
not sealed at all.** Every scalar on the outcome object — the exit code, the restated count, the
request URL — is stored in the clear beside three sealed columns. Placing a scalar on the object is
therefore also a decision about secrecy, and nothing on either deleted comment said so.

## Decision

> **A scalar that only one producer variant carries rides that variant's typed-outcome JSONB object,
> under that variant's own `kind` tag. It earns no column on the shared `transcript` table, and the
> read surface reads it out of the object. Because the outcome object is not sealed, putting a scalar
> there is also a decision to store it in the clear, so a scalar that is a credential may not go
> there at all.**

### 1. The union tag plus the outcome object is the whole discriminated part of the row

Everything else on the row is either the common frame that every variant carries, or a role column
that every variant may reuse. A reader who holds `variant` and `outcome` holds everything specific to
that variant. That property is what makes one table serve three producers, and a per-variant column
is the only thing that can break it.

### 2. A column on this table is a role, never a variant

`stdout` is not *the prober's stdout*. It is **the primary reading surface of whatever exchange this
variant made** — bytes for the prober, a response body for CT, a skip list for zone. That is the test
for a new column: it earns one only if every variant can say what it would put there, or leave it
NULL for a reason the role itself explains.

A scalar that only one variant can produce fails that test by construction.

### 3. The read surface reads it out of the object, and a scalar with no reader earns nothing

The three decoders at `cmd/web/rawoutput.go:180`, `:206` and `:230` are the only readers of the
corpus, because ADR-0126's fence puts no derivation on it. A scalar that no decoder renders is not
stored anywhere at all. It is not put on the object "for later".

### 4. The object is plaintext, so this rule carries a secrecy gate

A scalar placed here is readable to anyone who reads the database, without the instance key. That is
correct for the three scalars in the table above and it is not a general licence.

The gate is ADR-0126's at-rest sealing clause, as this issue's manifest scopes it: **the sealing
obligation binds a credential.** A scalar that is a credential is not admitted to the outcome object.
It goes into a sealed role column, or it is not captured. A reviewer adding a fourth scalar answers
that question first, and only then §1's question about the column.

### 5. What this rule does not reach

- **The common frame.** `kind`, `duration_ns` and `captured_at` are columns, correctly. Every variant
  carries all three, and the retention dial sweeps on `captured_at`, which a JSONB field could not
  serve cheaply.
- **The truncation marker.** It is keyed by stream name and describes the role columns, so it is a
  property of the streams rather than of the variant.
- **The closed union's default branch.** [ADR-0209](./0209-a-closed-union-we-author-refuses-an-unknown-member-and-writes-no-row.md)
  rules that.

## Consequences

- **No production behaviour changes.** The migration, both encoders and all three decoders already
  have this shape.
- **`internal/queue/transcript.go` gains this ADR's citation** on the survivor comment at
  `encodeZoneOutcome`. It is recorded in this issue's manifest rather than edited here.
- **A fourth variant costs no column.** It adds a `kind` arm to its encoder and a decoder arm to
  `cmd/web/rawoutput.go`. The migration it needs is the `CHECK` widening ADR-0209 §3 names, which is
  the cost of the union rather than of this rule.
- **A cost, stated: a scalar in JSONB cannot be indexed or constrained by the schema.** Nothing
  queries a transcript by its restated count or its request URL today. Every read is by
  `queue_job_id`, which is the primary key. If a query ever needs to filter or sort on one of these
  scalars, this ruling is the thing to re-open, and the re-opening is a migration.
- **A cost, stated: the three decoders parse defensively.** Each declares pointer fields and falls
  back to `—`, because the schema does not enforce the object's shape. That is the price of the
  JSONB, and it is paid three times.
- **A defect: nothing pins the encoder and decoder key names together.** `encodeZoneOutcome` writes
  `"restated"` and `rawDecodeZoneOutcome` reads `"restated"`, in two packages, with no shared
  constant and no round-trip test. A typo in either renders `—` in the admin view and fails no test.
  A round-trip test over the three variants would close it, and it ships as its own ticket.
- **`CONTEXT.md` gains nothing.** Its `Transcript` entry already states that each variant carries its
  own typed outcome. Which storage the scalar lands in is a schema rule, not a domain term.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A column per variant scalar — `restated INT`, `request_url TEXT`, `exit_code INT`** | Each is NULL on two variants of three the day it lands, and every new variant adds another. It also splits a variant's shape across the tag and a column set, so `variant` stops determining which fields of the row are meaningful and a reader needs out-of-band knowledge to read the row |
| **One shared `extra JSONB` column beside `outcome`** | Two JSONB objects where one does the work, with no rule to sort a field between them. `restated` is as much part of the zone outcome as its `text` is, and any line drawn between the two objects would be arbitrary at the first field that sits near it |
| **Put the scalar in the `truncation` marker** | That object is keyed by stream name and describes the `bytea` role columns. A restated count is not a property of a stream, and filing it there would make the marker's key space mean two things |
| **Seal the `outcome` object with the streams** | The reader needs the `kind` tag to choose a decoder, so sealing the object would force a decrypt on every render of the corpus, including the rows whose streams the operator did not open. It would also seal the CT request URL, which the manifest's ADR-0126 scope clause rules is not a credential and is worth keeping reproducible |
| **A table per variant** | Three tables and a three-way join to answer one `?job={id}` read, for a corpus with a single key and a single reader. It also multiplies the retention sweep and the backup exclusion by three |
| **Widen the shared table so every variant fills every column** | ADR-0126 and `CONTEXT.md` refuse it at the level above: the value is a closed union, **never a record with optional fields**. A table of per-variant columns is that record, written in SQL |
| **State the rule in [`raw-job-output.md`](../spec/raw-job-output.md) §1.4** | That document makes no new decision by its own preamble. Its §1.4 shows the column groups the ruling produced and gives no reason a column was refused |
