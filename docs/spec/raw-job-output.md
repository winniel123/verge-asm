# Verbatim raw job output for operator debugging

- **Status:** Accepted — handoff spec for map [#838](https://github.com/winniel123/verge-asm/issues/838), terminal ticket [#844](https://github.com/winniel123/verge-asm/issues/844)
- **Ruling:** [ADR-0126](../adr/0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md), **Accepted**. [#839](https://github.com/winniel123/verge-asm/issues/839) posted the draft, and [#871](https://github.com/winniel123/verge-asm/issues/871) finalised it from this spec on 2026-08-31.
- **Build status:** **Built.** Map [#838](https://github.com/winniel123/verge-asm/issues/838) shipped this spec through tickets [#862](https://github.com/winniel123/verge-asm/issues/862) to [#871](https://github.com/winniel123/verge-asm/issues/871). The map closed on 2026-08-31.
- **Decisions folded:** [#839](https://github.com/winniel123/verge-asm/issues/839) corpus + retention · [#840](https://github.com/winniel123/verge-asm/issues/840) producer · [#841](https://github.com/winniel123/verge-asm/issues/841) transport · [#842](https://github.com/winniel123/verge-asm/issues/842) access + secrets · [#843](https://github.com/winniel123/verge-asm/issues/843) UI

This document folds the map's five locked decisions into one buildable spec. It makes **no new
decision**. Where a decision was deferred or left thin, this document says so and does not smooth it
over (§10). The build session worked from here and did not re-open the tickets.

**Reading this after the build.** A sentence about the pre-build code names the 2026-08-29
fact-find and takes the past tense. Every other statement describes the code as it stands, and it
names the site. §1.3 and §10 mark the parts the build did not build ([#1457](https://github.com/winniel123/verge-asm/issues/1457)).

## What this builds

An operator debugging a job needs the genuinely-raw output of that job: **stdout + stderr +
exec-meta** (exit code or signal, duration, the `JobSpec` sent). At the 2026-08-29 fact-find,
`/runs/{id}?job={n}` showed only `kind · state · vantage`. `runLog` (`cmd/web/scans.go`) built that
line from the lean `queue_job` operational record. That is the record of what the system *did*, not
the job's output. A probe job emitted only structured NDJSON observations on stdout. No
raw-stdout/stderr channel, no store, and no wire type for raw output existed.

This effort added a durable, verbatim store — the `Transcript` corpus. The worker captures it at
the prober boundary on **local and remote** vantages. A dedicated admin-gated view surfaces it, and
the `?job={id}` chip links to it. All three parts stand: the `wire.Transcript` union
(`internal/wire/transcript.go`), the `transcript` table (`db/migrations/23700_transcript.sql`), and
`GET /runs/{id}/raw` behind `requireAdmin` (`cmd/web/handlers.go`).

### Three conscious overrides

This change consciously reversed three deliberate mechanisms. Each reversal is priced and recorded,
not accidental.

| Mechanism | Before the build | What shipped |
| --- | --- | --- |
| **ADR-0041** — a corpus is retained by what may still read it; only `Dispatch` carries a clock, and v1 ships both retirable corpora **unbounded** | Nothing verbatim was retained at rest, and the run log was state-derived | A fourth Operational corpus carries a clock and **ships bounded** (§1, §4) |
| **Redaction posture** — the progress/log vocabulary is closed and redacted, so verbatim bytes never reach rest | No verbatim bytes at rest | The `transcript` table stores verbatim `stdout`, `stderr`, and the sent `JobSpec` (§1, §5) |
| **ADR-0053** — Postgres holds no secret | Secrets lived on the service volume, never in the DB | `Transcript` is the first secret-bearing corpus at rest, **encrypted** with a volume key (§5) |

Two marks are used, on `docs/spec/measurement-offers.md`'s convention:

| Mark | Means |
| --- | --- |
| `[derived]` | Follows from a locked decision named in the row |
| `[thin]` | Chosen with a stated price, or deferred; revisable |

---

## 1. Data model — the `Transcript` corpus

**Ruling: [#839](https://github.com/winniel123/verge-asm/issues/839).**

`Transcript` is a **new, fourth Operational corpus**, standing beside `Dispatch`, `Message`, and
`Delivery`. It is keyed **per job** (`queue_job` grain).

- **Not** an extension of `Dispatch`. `[derived]` Wrong grain — a `Dispatch` is one Scan firing over
  a fan-out of many jobs; a `Transcript` is one job's output.
- **Not** columns on the lean `queue_job` record. `[derived]` The record (`kind · state · vantage`)
  is negligible and kept cheaply. The verbatim bytes are the whole volume. They must be **retirable
  independently** of the record.
- **The fence.** No derivation may read a `Transcript`, exactly as none may read a `Dispatch`. That
  fence is what makes a clock legal on it. Only the worker writes it; only the §6 handler reads it.

### 1.1 Keying grain — one `Transcript` per attempt

**Ruling: [#840 §7](https://github.com/winniel123/verge-asm/issues/840).** One `Transcript` per
`queue_job` row, which is **one per attempt**. Retry enqueues a new `queue_job` row (`retry`'s
`EnqueueJob` call, `worker.go`), so a retried attempt keeps its own transcript on its **retired**
row while the fresh attempt gets a new one on its new row. Both superseded and live attempts
retain their transcripts, each retired independently by the §4 dial. `?job={id}` is already keyed
on this id, so §6 addresses a transcript directly. Retry fan-out is 5 for every prober kind, so a
logical prober job holds up to **5** transcripts.

### 1.2 The `Transcript` value — a closed union

**Ruling: [#840 §3](https://github.com/winniel123/verge-asm/issues/840).** Model `wire.Transcript` as
a **closed union**, one variant per producer kind — not a struct with optional fields (CONTEXT.md:
"every value space is a closed union, never a record with optional fields").

A **common frame** carries `{ queue_job id, kind, duration, captured-at }`. `captured-at` is stamped
`w.now()` so the §4 duration dial can retire by age. Each variant names the one exchange it made and
carries its **own typed outcome** (there is no shared outcome union):

| Variant | Vantage | Streams captured | Typed outcome |
| --- | --- | --- | --- |
| **`ProberTranscript`** | local and remote, both shipped ([#865](https://github.com/winniel123/verge-asm/issues/865), [#867](https://github.com/winniel123/verge-asm/issues/867)) | stdout bytes, stderr bytes, and the **exact bytes sent** to prober stdin | `exited(code) \| signalled(sig) \| context-cancelled` |
| **`CTTranscript`** | crt.sh HTTP producer | request URL (`scan.CrtshURL`), HTTP status, response body (verbatim), transport-error text (the stderr analog) | `http(status) \| transport-error(text) \| context-cancelled` |
| **`ZoneTranscript`** | zone restate | the restate result — see §1.3 | `parsed \| decode-error` |

A `ctx`-killed prober (job timeout or mid-flight cancel) reads as `context-cancelled`, **never** a
fake `exited(0)`. The build takes the local exit code from `ProcessState.ExitCode()`, in
`classifyProberOutcome` (`internal/queue/worker.go`). It takes the remote one from
`*ssh.ExitError.ExitStatus()`, `.Signal()` and `*ssh.ExitMissingError`, in `classifyExit`
(`internal/remoteexec/conn.go`). Both test the cancel first (§3).

### 1.3 The Zone variant

**Ruling: [#840 §3c](https://github.com/winniel123/verge-asm/issues/840).** Zone sends nothing to a
prober, so its debug artifact is the **restate result**: the count restated and, above all, the
records `RestateZone` **skipped** because it could not marshal them ("why is this DNS record missing
from the estate?" → "we skipped it"). `ZoneTranscript` does **not** store the zone-file bytes — the
file already sits in the operator's supplied zone-file row.

Two producer-implementation notes, each reconciled against the build:
1. **Built** ([#869](https://github.com/winniel123/verge-asm/issues/869)). `RestateZone` surfaces
   its skips. It returns `(records, skipped)`, and `completeZone` (`internal/queue/zone.go`)
   carries the skips into `wire.ZoneTranscript`. At the fact-find it did not.
2. **Unbuilt** — Zone still has **no failure tx**. Zone capture stays on the **completed path
   only**, unless a zone failure tx arrives later. `completeZone` is the sole zone terminal path
   (`internal/queue/worker.go`). `[thin]`

### 1.4 The `transcript` table

**Ruling: [#840 §6](https://github.com/winniel123/verge-asm/issues/840).** A `transcript` table keyed
on the `queue_job` id, holding:

| Column group | Contents | Notes |
| --- | --- | --- |
| Common frame | `queue_job` id (key), kind, duration, `captured-at` | `captured-at` stamped `w.now()` |
| Verbatim streams | `bytea` columns, one per stream | Each can reach its §3.2 cap. **Not** JSON-embedded. Encrypted at rest — §5.2 |
| Truncation marker | per-stream `{ kept, dropped }` (or memory-guard-tripped marker) | §3.1 |
| Variant | the variant tag with its typed outcome | §1.2 |

The producer inserts a row via an sqlc `InsertTranscript` query, mirroring `InsertObservation`. **A
job with no capture is a legible absence** — no row — distinct from a captured-but-empty stream.

---

## 2. Producer — capture at the prober boundary (local)

**Ruling: [#840](https://github.com/winniel123/verge-asm/issues/840).**

### 2.1 The capture seam

Change the `Prober` interface to return a result value, not a bare slice:

```
ProbeResult{ Observations []wire.Observation; Transcript wire.Transcript }
```

Change `VantageRouter.ProbeVantage` (`worker.go`) to the **same shape now**, so the worker's
`probe()` fan-in (`Worker.probe`, `worker.go`) has one return shape. The remote path returned an
**absent** transcript, which is a legible state. §3 then filled the bytes across the wire
([#841](https://github.com/winniel123/verge-asm/issues/841) ruled it,
[#867](https://github.com/winniel123/verge-asm/issues/867) built it), and the remote path carries
them now. The type and seam were this ticket's, and the remote *content* was §3's — a pure
fill-in, not a signature change.

Grounding, 2026-08-29: `ExecProber.Probe` (`internal/queue/worker.go`) buffered raw stdout and
captured stderr, then **discarded both**, because the `Prober` interface returned only
`[]wire.Observation`. Nothing took the exit code or the duration, because `cmd.Run()` dropped
`ProcessState`.

What shipped: `ExecProber.Probe` returns `wire.ProbeResult` on every path, and
`buildProberTranscript` reads `cmd.ProcessState` for the outcome. The duration comes from a
`time.Now`/`time.Since` pair around `cmd.Run()`, **not** from the `Start`→`Wait` bracket this
section names. `cmd.Run()` calls `Start` and then `Wait`, so the measured span is the same.

### 2.2 Capture on every outcome that ran a producer

The `Transcript` rides the **error** return too. Capture and persist on **completed, retried,
dead-lettered, and decode-failure** — not success only. `[derived]` The raw output is highest-value
exactly when the job failed or the observation decode failed (`ExecProber.Probe`'s `sc.Err()`
return). This is the whole reason the seam holds the transcript on the error path. `complete`,
`deadLetter` and `retry` each call `persistTranscript` inside their terminal tx
(`internal/queue/worker.go`).

### 2.3 The sent payload is verbatim

For the prober variant, capture the **exact stdin bytes** (`wire.EncodeJobSpec` output,
`worker.go`), not a re-encoded struct. `[derived]` #839 locked a verbatim-at-rest posture and
"the JobSpec sent" means the literal payload.

### 2.4 Transaction placement and mid-flight cancel

**Ruling: [#840 §6b](https://github.com/winniel123/verge-asm/issues/840).** Write the `Transcript`
**inside the same tx as the terminal state** on each path:

| Terminal path | Site | Transcript attaches to |
| --- | --- | --- |
| `complete` | `complete`'s tx, beside `markDone` (`worker.go`) | the completed row |
| `deadLetter` | `deadLetter`'s tx, beside `markDead` (`worker.go`) | the dead-lettered row |
| `retry` | `retry`'s tx, beside `markRetried` (`worker.go`) | the **failed attempt's** row, **not** the freshly-enqueued one |

A mid-flight cancel (`errJobCanceled`, `worker.go`) **rolls the transcript back** with all other
staged work — a terminated job discards everything, no exception. This is why the raw view is
post-hoc only (§6.2).

**"Everything" is bounded to the attempt's staged work.** The cancel marks the `queue_job` row and
deletes no committed `batch` and no committed `observation`. A running job whose transaction already
committed is `done` or `dead` before the cancel reaches it, and its work stands. See
[ADR-0164](../adr/0164-an-operator-ends-a-dispatch-by-recording-a-disposition-once-and-stop-keeps-the-running-jobs-while-terminate-rolls-their-staged-work-back.md)
§3, which rules the operator act that causes the cancel.

### 2.5 Optional seam — `WithTranscripts`

**Ruling: [#840 §8](https://github.com/winniel123/verge-asm/issues/840).** Transcript capture is a
`WithTranscripts` seam, off when unwired and under `devMode` — the same pattern as
`WithMessages` / `WithCT` / `WithRouter`, with a no-op guard like `changeCollector`. Measurement-only
and fixture construction write no transcript, so this effort moves **no golden fixture**.

---

## 3. Transport — carry raw bytes across the vantage wire (remote)

**Ruling: [#841](https://github.com/winniel123/verge-asm/issues/841).**

### 3.1 Wire shape — no new `internal/wire` type; grow the `Conn` seam

There is **no new serialized wire type**. The prober's stdout *is* the verbatim NDJSON the
`ProberTranscript` stores; the worker is the center and assembles the transcript in-process before
writing it to the `transcript` table (§1.4). Nothing new crosses a wire as a decoded type. Instead:

- **Widen the narrow `Conn.Run` seam** to surface the two channels it dropped — a **stderr sink**
  and a **typed exit result**. The seam took the first form named here,
  `Run(ctx, cmd, stdin, stdout, stderr) (ExitResult, error)`, and not the `RunResult` struct
  alternative (the `Conn` interface, `internal/remoteexec/conn.go`). The fake-testability property
  survives, and the in-memory fake fills two more fields.
- `ExitResult` maps the prober's typed outcome (§1.2): `exited(code)` from `*ssh.ExitError.ExitStatus()`,
  `signalled(sig)` from `*ssh.ExitError.Signal()` / `*ssh.ExitMissingError`, `context-cancelled` from
  a `ctx`-killed session. `classifyExit` (`internal/remoteexec/conn.go`) holds that mapping.
- `remoteexec.Probe` returns `ProbeResult` (§2.1) **populated even on the error path**, so failed and
  decode-failed jobs still capture. At the fact-find `Probe` returned `nil, err` and discarded the
  drained stdout. It now returns `wire.ProbeResult{Transcript: t}` on the run-error path and on the
  decode-error path alike (`Probe`, `internal/remoteexec/probe.go`). One exception shipped. A
  failure before the measured exec, such as the binary push, carries no transcript. This fill-in of
  the remote `ProberTranscript` content is §3's, and it changed no signature in the worker fan-in.

Grounding, 2026-08-29: SSH delivers three native channels — stdout, stderr, and an exit-status
message. At the fact-find `Conn.Run(ctx,cmd,stdin,stdout)` had **no stderr sink**, and `runSession`
returned the `sess.Wait()` error raw. That error carries `*ssh.ExitError` (exit code, signal), and
nothing read it. Stderr and exec-meta already rode the wire, and the remote path simply dropped two
of the three channels. Prober stderr is normally **empty**. The prober writes NDJSON to stdout
only, and stderr holds content solely on a `log.Fatalf` crash (`main`, `cmd/prober/main.go`).

What shipped: `Conn.Run` takes a `stderr io.Writer` and returns `ExitResult`, and `classifyExit`
reads both `*ssh.ExitError` and `*ssh.ExitMissingError` (`internal/remoteexec/conn.go`).

### 3.2 Truncation and per-stream store caps

**Head+tail truncate-and-mark**, per stream, with a `{ kept, dropped }` marker. `[derived]` Head+tail
(not head-only) because when a job overflows the cap the **tail** holds the crash/exit context an
operator most wants; a head-only cut usually loses exactly the failure. Truncation never fails a
job — a job that overflows is exactly one you want the transcript for.

The **64 MiB fail-closed `LimitedBuffer` memory guard stays unchanged** (`wire.MaxProberStdout`,
`wire.go`) as the memory ceiling and the observation-decode source (the decoder still needs the
whole stream). stdout is already fully buffered, so the transcript's stdout is
`head+tail(buffer, cap)` taken **post-drain** — no streaming tee. stderr gets its own `head+tail`
sink (no decoder reads it). On a 64 MiB overflow (the memory guard trips and errors the job),
capture `head(...)` of what the guard retained plus a distinct **memory-guard-tripped** marker, so
the overflow job still carries a transcript.

Per-stream **store** caps (distinct from the 64 MiB memory guard):

| Stream | Store cap | Rationale |
| --- | --- | --- |
| **stdout** | **4 MiB** | Holds a small/typical job whole; a ceiling job (~25 MB) keeps ~2 MiB head + ~2 MiB tail + marker. The cap is honest, not lossy — transcript stdout mostly re-states the Observation corpus; its unique value is the small pre-#773-gate delta (§2.3, §5.1). |
| **stderr** | **256 KiB** | Normally empty; a panic/stack trace is tiny. Generous headroom at near-zero disk cost. |
| **sent-scope (stdin)** | **64 KiB** | Comfortably fits the 1024-address job spec (~15 KB); truncate-marks a pathological scope. |

Worst-case per transcript ≈ **4.3 MiB**.

The build holds the three caps as `capTranscriptStdout`, `capTranscriptStderr` and
`capTranscriptSentScope` (`internal/queue/transcript.go`). `headTail` applies them, and
`buildProberParams` writes the `{ kept, dropped }` marker. A guard trip adds a
`memory_guard_tripped` flag to the stdout marker. `wire.MaxProberStdout` is still 64 MiB.

### 3.3 Volume posture

A `Transcript` is **per-batch, not per-subject**. One hot-Scan connect-outcome job fans out one job
per Vantage over the whole Custody-admitted address set (`fanOutHot` and `enqueueHotJob`,
`internal/queue/hot.go`). At the `DefaultAddressCap = 1024` ceiling (`internal/seed/seed.go`) that
is up to ~134,144 observation lines in one job's stdout; at ~185 bytes/line a ceiling job's stdout
is ~25 MB — large, and mostly a verbatim re-statement of the Observation corpus, which the 4 MiB
cap bounds.

Transcript disk scales with `remote-jobs/day × attempts(≤5) × per-stream cap × retention-days`,
bounded by the caps (§3.2) and the shipped retention window (§4). `egressguard` needs **no change** —
it governs outbound dials; inbound return data is an orthogonal axis.

---

## 4. Retention — the `transcript_currency_days` dial

**Ruling: [#839 §2](https://github.com/winniel123/verge-asm/issues/839) (posture) +
[#841 §5](https://github.com/winniel123/verge-asm/issues/841) (number).**

Add a **new `transcript_currency_days` column** to the `retention_settings` singleton, unit **days**,
mirroring `observation_currency_days` (a forensic debug window reads naturally as "keep raw output for
N days") — **not** the cadence-multiple unit.

| Property | Value | Note |
| --- | --- | --- |
| Unit | days | Same duration instrument and sweep engine as the Dispatch dial (`internal/retention/retention.go`) |
| **Default** | **14** | Ships **bounded**. This non-zero default **is** #839's "ships bounded" reversal of ADR-0041 (which shipped `0`=unbounded on both retirable corpora) |
| Floor | 1 day for any positive value | |
| `0` | unbounded | Explicit operator opt-out; matches the existing sentinel |
| Coverage-style floor | **None** | No derivation reads a `Transcript`, so no minimum-retention floor applies |

The instrument stays a **duration** dial (ADR-0041's choice, kept); only the **default** flips from
unbounded to bounded, because verbatim bytes are the volume problem on exactly the address-scope
installs that motivated retention. Existing dials for reference: `observation_currency_days` and
`dispatch_cadence_multiple`, both `DEFAULT 0` where `0 == unbounded`
(the `retention_settings` table, `db/migrations/20600_channels_and_retention.sql`).

What shipped: `db/migrations/23700_transcript.sql` adds the column with `DEFAULT 14` and a
non-negative check. `TranscriptFloorDays`, `TranscriptWindowDays` and `TranscriptCutoff` hold the
floor and the `0` sentinel, and `TranscriptRetirer.Sweep` retires by age
(`internal/retention/transcript.go`). The settings page carries the dial ([#911](https://github.com/winniel123/verge-asm/issues/911)).

---

## 5. Access and at-rest posture

**Ruling: [#842](https://github.com/winniel123/verge-asm/issues/842).**

### 5.1 The three secret surfaces

All three are stored verbatim, sit behind the same admin gate (§5.2), and are encrypted alike (§5.3):

1. **stderr** — may carry credentials or tokens on a crash.
2. **the sent scope (stdin)** — carries the exact `JobSpec`, which can hold credentials for
   credentialed sources.
3. **pre-gate stdout** — the prober transcript captures stdout **before** the #773 scope re-gate
   (`complete`'s `parseAuthorizedScope` gate, `worker.go`), so it can hold lines for subjects the
   Observation corpus dropped, including out-of-scope bytes a compromised prober injects
   (`internal/queue/scopegate.go`). Kept verbatim, because it is the **most valuable evidence**
   for debugging a misbehaving or compromised prober; dropping it would defeat the corpus.

**Scoped** ([#1321](https://github.com/winniel123/verge-asm/issues/1321) §3, 2026-09-05): three
is the count of **credential-bearing** surfaces, not of everything the corpus stores. The sealing
obligation binds a credential. The CT request URL rides the outcome object in plaintext, because
it is a public query against a public log and both CT sources carry their credential in a header.
See the scope clause in
[ADR-0126](../adr/0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md),
*The at-rest reversal*.

### 5.2 Access model — admin-only

Raw `Transcript` output is readable by **`admin` accounts only**. The view reuses the existing
`requireAdmin` gate (`cmd/web/auth.go`), which the shipped `GET /runs/{id}/raw` route wraps
(`cmd/web/handlers.go`). This raises the gate above the run/job log, which any authenticated
account — `admin` or `viewer` — can still read (`requireLogin` on the `GET /runs/{id}` route).

`viewer` accounts **lose** the raw-output visibility they have for the state-derived log today. This
is an **intentional escalation** — the `Transcript` can carry secrets the state-derived log cannot.
No new capability and no new role; the two-role model (`roleAdmin`/`roleViewer`, `auth.go`) is
unchanged. `[derived]` A dedicated capability was rejected as over-engineering for a two-role app with
no capability system.

### 5.3 At-rest handling — encrypted

The `Transcript` `bytea` columns are **encrypted at rest with AEAD**. This consciously reverses
ADR-0053's "Postgres holds no secret" for this one corpus — the same way #839 reversed ADR-0041.
`Transcript` is the first secret-bearing data Postgres holds at rest.

**Key management:**
- **One instance-wide symmetric key**, generated into the **service volume**. This follows the
  existing ADR-0053 volume-secret pattern (session-signing key, SSH prober key). The key **never
  enters Postgres**.
- Each `bytea` column value is sealed with AEAD and a **random per-value nonce**.
- **No** envelope encryption and **no** per-record data key. `[derived]` #839's bounded retention
  (14-day default) already expires the lifecycle in bulk, so per-record crypto-shredding is not
  needed.

**Backups:** the `Transcript` corpus is **excluded from backups entirely**. `[derived]` It is
transient debugging data with bounded retention, not the durable estate ADR-0124 preserves. Exclusion
keeps ADR-0124's "a backup carries data and no credential" invariant intact — no secret-bearing bytes
leave through a backup — and avoids shipping ciphertext a fresh-instance restore cannot decrypt,
because the key stays on the volume and never rides in a DB backup.

What shipped: `LoadOrCreateKey` writes a 32-byte key to the service volume, and `Seal` uses
XChaCha20-Poly1305 with a random per-value nonce (`internal/transcript`). A nil plaintext stays SQL
NULL, so a legible absence survives the seal. `cmd/web/backup.go` lists `transcript` among the
excluded tables and states the reason.

### 5.4 Blast-radius note — residual risk accepted

Controls: admin-only read (`requireAdmin`), encrypted at rest (instance key on service volume, key
never in DB or backup), excluded from backups. Accepted gaps carried forward:

1. **Reads are unaudited.** `[thin]` Any admin can read any `Transcript` with no trail. The audit
   facility is a repo-wide stub (`fillAuditSection` returns nil, `cmd/web/settings.go`) and
   stays deferred. **No audit-of-reads in v1.**
2. **Pre-gate stdout is stored verbatim** and is attacker-influenceable. Safe rendering
   (escape-on-render) is §6.4.
3. **The instance key sits on the same volume as the data.** `[thin]` A full host compromise yields
   both key and ciphertext. Encryption defends against DB-only or backup exfiltration, not host
   compromise. This is the load-bearing gap.
4. **`viewer` accounts lose** the raw-output visibility they have today. Intentional (§5.2).

---

## 6. UI — surface raw output on `?job={id}`

**Ruling: [#843](https://github.com/winniel123/verge-asm/issues/843).** Prototype primary source:
branch [`research/843-rawoutput-prototype`](https://github.com/winniel123/verge-asm/tree/research/843-rawoutput-prototype),
commit `6142247`, `design-system/templates/rundetail-rawoutput.prototype.html`. That path lives
on that branch alone. The commit never merged to `main`, and its own message calls the prototype
throwaway, so a checkout of `main` holds no such file. The shipped template below replaced it.

**Shipped** ([#866](https://github.com/winniel123/verge-asm/issues/866)). `GET /runs/{id}/raw` is live behind `requireAdmin`
(`cmd/web/handlers.go`), and `rawOutputPage` (`cmd/web/rawoutput.go`) renders
`design-system/templates/rundetail-raw.tmpl`. The job-filter chip carries the `rd-rawlink`
affordance, and only an admin sees it (`design-system/templates/rundetail.tmpl`).

### 6.1 Render surface — a dedicated admin-gated view

Raw output gets its **own dedicated, admin-gated view** — **not** a panel inside the redacted log
region. It is opened by a small **"Raw output (admin)"** affordance on the job-filter chip. The
frozen redacted LogViewer (`rundetail.tmpl`, #771) and its #780 live-append client stay
**untouched**. Verbatim bytes never land on the redacted surface.

`[derived]` **Variant B ruled out** — an in-place `Redacted ↔ Raw` tab swap in one panel. Mixing the
closed/redacted vocabulary with verbatim bytes on the same surface is the highest-risk option against
the frozen template and re-entangles with the #780 append client.

### 6.2 Live-tail vs post-hoc — post-hoc only

**Post-hoc only.** `[derived]` Locked by the producer/storage shape, not by taste: §2.4 writes the
`Transcript` in the job's terminal tx (rolls back on mid-flight cancel), and the #780 live stream is
an in-memory hub that persists nothing at rest. There is **no raw byte stream to tail** during a run.
The redacted live stream stays the sole during-run view; raw appears once the job reaches a terminal
state. This **dissolves collision #40** — the two surfaces never share a panel.

### 6.3 Multi-stream layout

`stdout` is the primary reading surface — a large console panel with room for the 4 MiB cap and the
head+tail truncation marker. `exec-meta` and `stderr` sit in a **340px side rail**:

| Region | Content |
| --- | --- |
| Primary panel | `stdout` — large console panel; 4 MiB cap; head+tail truncation marker |
| Side rail — `.rd-kv` card | exec-meta: exit code, signal, ctx-cancelled, duration, captured-at, and the **JobSpec sent** |
| Side rail — panel | `stderr` — a shorter console panel |

Reuse the design-system `rd-log` / `rd-logbody` / `rd-line` vocabulary, the `--console-surface`
tokens, and `.rd-card` / `.rd-kv`. Invoke the `verge-asm-design` skill before writing the markup.

### 6.4 Escape-on-render — mandatory

The view **must escape on render**. `[derived]` The inbound bytes are untrusted-at-rest (§3, §5.1),
including attacker-influenceable pre-gate stdout. **Build lines with `textContent` span-by-span,
exactly as the #780 client does; never `innerHTML`.** This is the escape-on-render item deferred here
from #841 and #842.

---

## 7. ADR and CONTEXT.md changes

The build session finalised the documentation, not just the code. **All five bullets below
landed** ([#871](https://github.com/winniel123/verge-asm/issues/871)). §10 item 1 names the ADR-0053-reversal vehicle the build chose.

- **Finalise draft ADR-0126** (posted in #839): "Verbatim job output is a fourth Operational corpus —
  the `Transcript` — retired by a duration dial that ships bounded." Move it from Proposed (draft) to
  Accepted with the final number (§4).
- **Amend ADR-0041 at two sites** (per ADR-0058, at the sites that state them): *"the only corpus a
  wall clock may retire"* gains a second member (`Transcript`), and *"unbounded on both retirable
  corpora"* is no longer total — `Transcript` ships bounded.
- **Record the ADR-0053 reversal** (§5.3) — encryption-at-rest for the `Transcript` corpus. `[thin]`
  #842 left the vehicle open: a new ADR, **or** an extension of ADR-0126. The build session picks one
  (§10).
- **CONTEXT.md** gains one term, `Transcript`, and amends three entries: the Operational-layer intro
  moves from *three* corpora to *four*; `Observation` is cross-referenced (the verbatim NDJSON stream
  now also persists in the `Transcript`, as-emitted); `Dispatch` gains that it is no longer the *only*
  clock-retirable corpus.
- **The redaction posture gains a documented, priced exception** for at-rest verbatim capture,
  operator-visible.

---

## 8. Build order

A suggested dependency order for the build session (not a new decision — the seams dictate it).
The build followed it, one ticket per step, from [#862](https://github.com/winniel123/verge-asm/issues/862) to
[#871](https://github.com/winniel123/verge-asm/issues/871):

1. **Data model + migration** (§1.4, §4): the `transcript` table, the `transcript_currency_days`
   column, sqlc `InsertTranscript`.
2. **`wire.Transcript` closed union + `ProbeResult`** (§1.2, §2.1): the type and the seam change,
   with the remote path returning an absent transcript.
3. **Local producer** (§2): capture at `ExecProber.Probe`, wire the terminal-tx inserts, the
   `WithTranscripts` seam, `RestateZone` skip surfacing.
4. **Remote transport** (§3): widen `Conn.Run`, `ExitResult`, `Probe` on the error path, head+tail
   caps.
5. **Encryption + access** (§5): AEAD seal/open on the `bytea` columns with the volume key, the
   `requireAdmin` gate, backup exclusion.
6. **Retention sweep** (§4): extend the sweep engine to the new dial.
7. **UI** (§6): the dedicated admin view, escape-on-render.
8. **Docs** (§7): finalise ADR-0126, amend ADR-0041, the ADR-0053 reversal, CONTEXT.md.

---

## 9. Traceability

| Concern | Locked by | Spec section |
| --- | --- | --- |
| Corpus placement, keying, retention posture | #839 | §1, §4 |
| Capture seam, outcomes, tx placement, zone skips, optional seam | #840 | §1.2, §1.3, §2 |
| `Conn.Run` widening, `ExitResult`, error-path `Probe`, caps, retention number | #841 | §1.2, §3, §4 |
| Admin-only read, AEAD at rest, key on volume, backups excluded | #842 | §5 |
| Dedicated admin view, post-hoc, layout, escape-on-render | #843 | §6 |

---

## 10. Where this is thin, stated rather than smoothed

These were the open edges the map did **not** close. They were noted, not decided. The build closed
two of them. Three stand, and each line below says which.

1. **Closed.** The ADR-0053-reversal vehicle (§7) was unchosen. #842 left it as "a new ADR, or an
   extension of ADR-0126". [#871](https://github.com/winniel123/verge-asm/issues/871) extended ADR-0126, whose title now
   carries the secret-at-rest clause.
2. **Open, and unbuilt.** Zone has no failure tx (§1.3). Zone capture stays completed-path only,
   unless a zone failure tx arrives later. `completeZone` (`internal/queue/zone.go`) is the sole
   zone terminal path. Out of scope for this effort, and still out of scope.
3. **Closed** ([#869](https://github.com/winniel123/verge-asm/issues/869)). `RestateZone` surfaces its skips. It returns
   `(records, skipped)`, and `completeZone` carries them into `wire.ZoneTranscript`.
4. **Open, accepted.** Reads are unaudited (§5.4). The audit facility is still a repo-wide stub
   (`fillAuditSection` returns nil, `cmd/web/settings.go`), and v1 ships no audit-of-reads.
5. **Open, accepted.** The instance key still sits on the same volume as the data (§5.4).
   Encryption does not defend against host compromise. This is the load-bearing security gap,
   accepted for v1.

#844 closed on 2026-08-30, and the map reached its destination. The build ran from this document
through [#862](https://github.com/winniel123/verge-asm/issues/862) to [#871](https://github.com/winniel123/verge-asm/issues/871), and it did not
re-open the tickets.
