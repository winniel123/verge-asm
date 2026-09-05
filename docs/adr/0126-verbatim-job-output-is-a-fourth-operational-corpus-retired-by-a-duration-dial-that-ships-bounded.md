# ADR-0126: Verbatim job output is a fourth Operational corpus — the `Transcript` — retired by a duration dial that ships bounded, and it is the one corpus Postgres holds a secret for

- **Status:** Accepted
- **Date:** 2026-08-29 (drafted, [#839](https://github.com/winniel123/verge-asm/issues/839)) · finalised 2026-08-31 ([#871](https://github.com/winniel123/verge-asm/issues/871))
- **Ticket:** [#839 Raw-output corpus + retention](https://github.com/winniel123/verge-asm/issues/839), finalised from the [#844](https://github.com/winniel123/verge-asm/issues/844) handoff spec ([`docs/spec/raw-job-output.md`](../spec/raw-job-output.md)) by [#871](https://github.com/winniel123/verge-asm/issues/871)
- **Map:** [#838 Verbatim raw job output for operator debugging](https://github.com/winniel123/verge-asm/issues/838)
- **Amends/reverses:** [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) (the unbounded-default and the single-clock-corpus rulings, at the sites that state them) and [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) (its *"Postgres holds no secret"* clause, for this one corpus)

## Context

Today `/runs/{id}?job={n}` shows only `kind · state · vantage`, built by `runLog` in `cmd/web/scans.go:920` from the `queue_job` operational record. That is the record of what the system *did*, not the output of the job. A probe job emits only structured NDJSON observations on stdout. There is no raw-stdout/stderr channel, no store, and no wire type for raw output today (fact-find, 2026-08-29).

An operator debugging a job needs the genuinely-raw output: **stdout + stderr + exec-meta** (exit code or signal, duration, the `JobSpec` sent). Surfacing the observations verbatim (alternative D1) was rejected — observations do not carry stderr, the exit code, or the spec. This ADR pursues D2: a durable, verbatim store, captured on **local and remote** vantages, surfaced on a dedicated admin-gated view reached from `?job={id}`.

Four of ADR-0041's rulings meet this new data, and this ADR consciously overrides three of them. A fourth deliberate mechanism, the redaction posture, is overridden too, and a fifth — ADR-0053's *"Postgres holds no secret"* — is reversed for this one corpus:

- *"A corpus is retained by what may still read it, never by its age."* — **Kept.** The new corpus obeys it.
- *"`Dispatch` [is] the only corpus a wall clock may retire."* — **Amended.** A second dial-carrying Operational corpus is added.
- *"v1's shipped defaults: unbounded on both retirable corpora."* — **Overridden** for this corpus: it ships bounded.
- The redaction posture (the progress/log vocabulary is closed and redacted, so verbatim bytes never reach rest). — **Overridden**, consciously and priced, with an operator-visible exception (§ *The redaction posture*).
- ADR-0053's *"Postgres holds no secret at all."* — **Reversed for the `Transcript` corpus alone.** It is the first *corpus* Postgres holds a secret for — `account.totp_secret` was already ciphertext there ([ADR-0172](./0172-a-bearer-authenticator-seed-is-admitted-to-postgres-as-aead-ciphertext-and-the-sealing-key-stays-on-the-volume.md)) — and it is stored encrypted (§ *The at-rest reversal*).

The Operational layer is already **three** corpora, not one: `Dispatch`, and — since [#119](https://github.com/winniel123/verge-asm/issues/119) — `Message` and `Delivery`. Only `Dispatch` carries a dial. This ADR adds a fourth, `Transcript`, and a second dial.

## Decision

| Concern | Decision |
| --- | --- |
| Corpus placement | A **fourth Operational corpus**, **`Transcript`**, keyed **per job** (`queue_job` grain), standing beside `Dispatch` / `Message` / `Delivery` |
| Why not an extension of `Dispatch` | Wrong grain. A `Dispatch` is one Scan firing over a fan-out of many jobs; a `Transcript` is one job's output |
| Why not columns on the job record | The lean record (`kind · state · vantage`) is negligible and kept cheaply. The verbatim bytes are the volume. They must be **retirable independently** of the record |
| The fence | No derivation may read a `Transcript`. That fence — the same one that fences `Dispatch` — is what makes a clock legal on it. Only the worker writes it; only the read handler reads it |
| Keying grain | **One `Transcript` per `queue_job` row, which is one per attempt.** A retry enqueues a new `queue_job` row, so a superseded attempt and its successor each keep their own transcript, retired independently |
| The value shape | A **closed union** over a common frame `{ queue_job id, kind, duration, captured-at }` — `ProberTranscript \| CTTranscript \| ZoneTranscript` — each variant naming the one exchange it made and carrying its **own typed outcome**. Not a struct with optional fields |
| Retention instrument | A **duration** dial, operator-tunable. Never a byte or row budget (ADR-0041's instrument, kept) |
| Retention default | **Ships bounded.** The one corpus that does. ADR-0041's unbounded-default ruling is reversed here because the byte-volume arithmetic does not transfer |
| The default number | **14 days** (`retention_settings.transcript_currency_days`, migration `23700`). Floor **1 day** for any positive value (`retention.TranscriptFloorDays`); **`0` = unbounded**, the explicit operator opt-out matching the existing sentinel. The non-zero default *is* the "ships bounded" reversal |
| Floor | **No coverage-style floor.** No derivation reads a `Transcript`, so no minimum-retention window applies. The 1-day floor is a fixed tightest-honoured window, not a derived one |
| Per-stream size | A capture/transport **byte cap** bounds each stream — stdout **4 MiB**, stderr **256 KiB**, sent-scope **64 KiB** — by **head+tail truncate-and-mark**, distinct from the unchanged 64 MiB memory guard (~4.3 MiB/transcript worst case) |
| Access control | **Admin-only.** Reuses `requireAdmin` (`cmd/web/auth.go`). `viewer` accounts **lose** the raw-output visibility they have for the state-derived log today — an intentional escalation, because a `Transcript` can carry secrets the redacted log cannot |
| At-rest handling | **Encrypted at rest with AEAD** (XChaCha20-Poly1305). One **instance-wide symmetric key** on a service volume both `web` and `worker` mount, per-value random nonce, **never in Postgres**. No envelope encryption, no per-record data key. **Excluded from backups entirely**. **Scoped** ([#1321](https://github.com/winniel123/verge-asm/issues/1321) §3): the obligation binds a **credential**, not every value the corpus stores — see *The at-rest reversal* below |
| The redaction reversal | Verbatim `stdout`, `stderr`, and the sent `JobSpec` persist at rest. The redacted LogViewer surface (#771/#780) is **untouched**; the verbatim bytes reach a **separate**, admin-gated view. The exception is priced and operator-visible |
| Scope | Local **and** remote vantages. Capture scope = stdout + stderr + exec-meta. The remote transport reverses the remote-byte discard in `internal/remoteexec/probe.go` |

## Rationale

### A fourth corpus, per-job, because the grain and the volume both demand it

Raw output is about a *job*, not a *firing*. A `Dispatch` groups many jobs, so attaching raw output to `Dispatch` would key it at the wrong level. Attaching it as columns on the `queue_job` record keys it correctly but couples two things with opposite volume profiles: the lean record is negligible and worth keeping cheaply, while the verbatim bytes are the whole volume problem — the address-scope 1,024 → 134,144-subject pressure that birthed ADR-0041. A separate corpus lets the fat bytes be retired on a tighter schedule than the record we keep. That decoupling is the reason it is a corpus and not a column.

It is Operational by construction. No comparison path or derivation may read it, exactly as none may read a `Dispatch`. ADR-0041's own reasoning applies unchanged: the property that makes a record safe to delete on a schedule is the property that makes it safe to keep out of the comparison path.

### The instrument stays a duration; only the default flips, and the number is now set

ADR-0041 chose a duration dial over a byte budget, because a budget makes the retained *window* a function of how much happened. That reasoning is untouched, so the `Transcript` window is a **duration** dial too.

What does not transfer is the *unbounded default*. ADR-0041 shipped `Dispatch` and observations unbounded because the arithmetic made it affordable — a `Dispatch` is ~420 rows/year, negligible. Verbatim job output is bytes, not rows, and its volume is large on exactly the address-scope installs that motivated retention in the first place: a single ceiling job's stdout is on the order of 25 MB, mostly a verbatim re-statement of the Observation corpus, bounded by the 4 MiB store cap. Shipping it unbounded-by-default would put a disk hazard on the installs that can least afford one. So this corpus ships **bounded**.

The draft deferred the number to the volume work. It is now set: **14 days**, floored at **1 day** for a positive value, with **`0`** the explicit operator opt-out (unbounded). Fourteen days is not derived from a measurement — ADR-0038 forbids shipping a constant that measures the world on the day it was written — it is a forensic-window default chosen to bound the byte hazard while a real debugging window stays reachable. It re-states ADR-0041's honest position in the other direction: where nothing bounds a row-cheap corpus the honest default is unbounded, and where the bytes are the hazard the honest default is a bounded window the operator can widen or switch off.

A byte *cap per stream* is a different instrument from a byte *budget as the window*, and it is adopted. It bounds a single `Transcript` at capture and on the wire. It is not the retention window and does not reopen ADR-0041's rejected budget.

### The at-rest reversal: Postgres holds this one corpus's secrets, encrypted

The redacted log vocabulary was closed precisely so that verbatim bytes never reached rest. This corpus reverses that: three surfaces are stored verbatim and each may carry a secret.

1. **stderr** — may carry credentials or tokens on a crash.
2. **the sent scope (stdin)** — carries the exact `JobSpec`, which can hold credentials for credentialed sources.
3. **pre-gate stdout** — captured before the #773 scope re-gate, so it can hold lines for subjects the Observation corpus dropped, including out-of-scope bytes a compromised prober injects. It is kept verbatim, because it is the **most valuable evidence** for debugging a misbehaving or compromised prober; dropping it would defeat the corpus.

**The obligation binds a credential, and not every value the corpus stores** ([#1321](https://github.com/winniel123/verge-asm/issues/1321) §3, scoped 2026-09-05). The three surfaces above are sealed because each may carry a credential. A stored value that cannot carry one is outside the obligation, and this corpus already holds such a value: the **CT request URL**, written plaintext on the outcome object by `encodeCTOutcome` (`internal/queue/transcript.go`). It is a public query against a public log, reconstructible by anyone who knows the subject, and both CT sources carry their credential in an HTTP header and never in the URL — `NewCertSpotterFetcher` sets `Authorization: Bearer` (`internal/queue/crtsh.go:73`), and crt.sh carries no credential at all. Sealing it would protect nothing, and it would cost two things that are real: a failed CT job stops being reproducible, and an operator cannot see what we asked for.

This clause **does not reopen the ADR-0053 reversal it sits inside**. That reversal is what lets Postgres hold a sealed secret at all, and it stands exactly as written: the three surfaces are still sealed, the key still lives on a service volume and never enters Postgres, and the corpus is still excluded from every backup. The clause narrows only the *reach* of the obligation, never the posture that discharges it. The standing condition, so a later change can be tested against it: **a CT source that ever put a credential in a query string would fall back inside the obligation**, and its URL would then have to be sealed in a role column or not captured at all. The outcome object is not sealed, so any value placed there is stored in the clear by construction.

Two controls answer the new risk class, and they land the fifth conscious override. First, access is **admin-only**: the raw view reuses `requireAdmin`, above today's run/job log that any authenticated account may read. A `viewer` loses raw-output visibility it has today — an intentional escalation, no new capability and no new role. Second, the bytes are **encrypted at rest**, which **reverses ADR-0053's *"Postgres holds no secret"*** for this one corpus. ADR-0053 rejected sealing a secret in the database under a KEK; the `Transcript` corpus does exactly that, consciously, because unlike the SSH key or the session key these bytes are *evidence the operator asked to keep*, not a credential the system could have generated in place.

The reversal is narrow, and ADR-0053's deeper rule survives inside it. The sealing key is **one instance-wide symmetric key on a service volume both `web` and `worker` mount, never in Postgres** — the same volume-secret pattern ADR-0053 chose for the session and prober keys. So a read-only database leak (a backup, a replica, an export) discloses ciphertext and no key, and the rule *a secret is held only where the act it authorises is performed* still holds for the key itself. Each `bytea` value is sealed with AEAD (XChaCha20-Poly1305) under a **random per-value nonce**. There is **no envelope encryption and no per-record data key**, because the 14-day bounded retention already expires the lifecycle in bulk, so per-record crypto-shredding buys nothing.

The `Transcript` corpus is **excluded from backups entirely**. It is transient debugging data with bounded retention, not the durable estate ADR-0124 preserves. Exclusion keeps ADR-0124's *"a backup carries data and no credential"* invariant intact — no secret-bearing bytes leave through a backup — and avoids shipping ciphertext a fresh-instance restore cannot decrypt, because the key stays on the volume and never rides in a database dump.

### The redaction posture gains an operator-visible, priced exception

The verbatim bytes never touch the redacted surface. The frozen LogViewer (#771) and its #780 live-append client are untouched, and raw output reaches its **own dedicated admin-gated view**, reached by a small *"Raw output (admin)"* affordance and rendered post-hoc only — there is no raw stream to tail, because capture is written in the job's terminal transaction and the live hub persists nothing. The exception is therefore visible where an operator meets it (the affordance names itself *admin*), and it is priced here rather than left implicit: the redaction posture is a property of the redacted surface, and the raw corpus is a second surface with a higher gate, not a hole in the first.

## Consequences

- **A new `docs/adr/0126` is minted** — this document — and it is the single vehicle the shipped code cites for both the ADR-0041 retention amendment and the ADR-0053 at-rest reversal (`internal/transcript/key.go`, `internal/retention/transcript.go`, `cmd/worker/main.go`, `cmd/web/handlers.go`). The spec §10 open edge *"the ADR-0053-reversal vehicle is unchosen"* is closed here: **one ADR, not two**, because the code already names ADR-0126 for the reversal and a separate ADR would strand that citation.
- **[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) is marked at every site that states the two amended claims**, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as [#106](https://github.com/winniel123/verge-asm/issues/106) read it — the unit is the sentence, so *"the only corpus a wall clock may retire"* and *"unbounded on both retirable corpora"* are marked in the Decision table, in the *v1 ships no expiry* section, and in the Consequences bullet that restates the first. ADR-0041's instrument (a duration dial), its fence rationale, and the unbounded default *for `Dispatch` and observations* are untouched and confirmed.
- **[ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) is marked at the two sites that state *"the database holds no secret"***, per ADR-0058. Its rule (*a secret is held only where the act it authorises is performed*), its volume-secret pattern, and its backup invariant are **kept** — the `Transcript` key lives on a volume and the corpus is out of every backup.
- **[`CONTEXT.md`](../../CONTEXT.md) gains one term, `Transcript`, and amends two entries.** The `Dispatch` entry moves the operational record from *three* corpora to *four* and marks that `Dispatch` is no longer the only clock-retirable corpus. `Observation` is cross-referenced: the verbatim NDJSON stream now also persists in the `Transcript`, as-emitted.
- **The redaction posture gains a documented, priced exception** for at-rest verbatim capture, operator-visible through the admin-gated raw view.
- **The build wave landed the code** across #862–#870, plus #911 (the operator-facing retention dial), surfaced from #868.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Columns on the `queue_job` record | Couples the negligible record to the unbounded bytes. Cannot retire the bytes sooner than the record, which was the point |
| An extension of `Dispatch` | Wrong grain — `Dispatch` is per-firing, raw output is per-job |
| Unbounded default, like `Dispatch` | The byte-volume arithmetic does not transfer. A disk hazard on address-scope installs |
| A byte/row budget as the window dial | ADR-0041's reasoning stands — the window would be a function of drift. A per-stream byte *cap* is a different instrument and is adopted |
| Redact the verbatim bytes at rest | Defeats the operator's stated need for genuinely-raw output. The reversal is conscious and priced, and its bytes reach a separate admin-gated surface |
| Store the secrets in plaintext | The three surfaces carry credentials; a read-only database leak would then disclose them. Encryption under a volume key is the ADR-0053 pattern applied to evidence rather than to a credential |
| Envelope encryption / a per-record data key | A KEK is a new operator-held secret invented to protect an old one, and per-record crypto-shredding is unneeded — the 14-day bounded retention expires the lifecycle in bulk |
| Encrypt the whole database at rest | Answers media theft, not the read-only-leak threat, at the price of an operator-held key this ADR would then have to rule on. Volume/filesystem encryption stays the operator's line of deployment guidance |
| A dedicated capability or third role for the raw view | Over-engineering for a two-role app with no capability system. `requireAdmin` is the existing gate |
| Audit every read of a `Transcript` | The audit facility is a repo-wide stub; a read-audit is deferred with the accepted residual risk stated below, not invented here |

## Where this is thin, stated rather than smoothed

- **The bounded-default number is a chosen figure, not a measured one.** 14 days bounds the byte hazard and reserves a debugging window; it is the operator's dial to widen or switch off (`0`), and it is not derived from any measurement of the world (ADR-0038).
- **Reads are unaudited in v1.** Any admin can read any `Transcript` with no trail, because the audit facility is a repo-wide stub (`fillAuditSection` returns nil). Accepted residual risk.
- **The instance key is co-located with the data.** A full host compromise yields both key and ciphertext. Encryption defends against a database-only or backup exfiltration, not host compromise. This is the load-bearing security gap, accepted for v1.
- **Zone capture is completed-path only.** Zone has no failure transaction today, so `ZoneTranscript` is captured on the completed path only unless a zone failure tx is added later. `RestateZone` was changed to surface its skips; the failure-tx edge is out of scope for this effort and flagged for a later ticket.
- **Retried-attempt capture is per-attempt, and both attempts are kept.** A superseded attempt keeps its transcript on its retired row and a dead-lettered job keeps the failing attempt's, each retired independently by the dial. This is a size statement rather than a modelling gap: a logical prober job holds up to five transcripts.
