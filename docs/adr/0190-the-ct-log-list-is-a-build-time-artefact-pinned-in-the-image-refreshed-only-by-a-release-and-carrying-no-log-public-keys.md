# ADR-0190: the CT log list is a build-time artefact pinned in the image, refreshing it is a release act, and the pinned snapshot carries no log public keys

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1308 ADR gaps: internal/scan (CT and zone Scans)](https://github.com/winniel123/verge-asm/issues/1308), gaps 2 and 3
- **PR that deleted the comments:** [#1307](https://github.com/winniel123/verge-asm/pull/1307)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Withdraws a clause of:** [`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.3, whose first sentence says to *"follow logs from the live `log_list.json`"*. The clause is withdrawn at its own site under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). The rest of §4.3 — the `state` filter, the `temporal_interval` filter, the two mandatory client implementations, the `url` versus `monitoring_url` discriminator — is untouched and confirmed
- **Rests on:** [ADR-0139](./0139-the-probers-origin-is-the-image-that-carries-it-and-a-host-bounds-the-binary-rather-than-verifies-it.md), whose §1 rules that a shipped artefact's origin is the image that carries it, because *"every read path hits the worker image's read-only filesystem. No path fetches a release asset, and no path reaches the network."* It rules the prober binary. This ADR applies the same shape to a data artefact
- **Rests on:** [ADR-0138](./0138-a-release-pins-every-byte-it-builds-so-it-delegates-no-build-step-and-anchors-identity-in-its-own-workflow.md), whose §1 rules that the release pins every byte it builds and that *"this project cannot enforce pin-every-byte inside a third party."* It rules the build. This ADR rules a byte the build pins
- **Bounded by:** [ADR-0003](./0003-third-party-source-consent-bar.md), which rules which third parties may be queried without the operator saying so, and makes that a property of the release rather than of a deployment. It rules the **source catalogue**. It does not rule which endpoints a source's own runner contacts, which is what the log list decides
- **Sibling of, and not ruled by:** [ADR-0144](./0144-the-verge-core-body-is-compiled-in-and-an-operator-edit-layers-over-it.md). It rules that the `verge-core` body is compiled in, over a spec clause that said the opposite, and it changed no Go code. The shape is the same and the artefact is different. Neither contains the other

## Context

`internal/scan/cttail.go:45` and `:52` carried these two, until #1307 rewrote them:

```go
// embeddedLogList is the pinned snapshot of Google's CT log_list.json (v89.34,
// 2026-08-29 — the version §4.3 names). The tail selects which logs it follows from
// this list. It is embedded rather than fetched live so the log-set is deterministic
// and testable and the fan-out reads it with no network step inside its transaction;
// a live-refresh path is deferred to fog. Refreshing it is a snapshot bump, not a
// schema change.
```

```go
// Each log's `key` (its public key) is stripped from the snapshot: this file selects
// and reads logs by id, url, state and temporal_interval, and never verifies a
// signature (the consistency proof is opportunistic and deferred — §4.4), so the key
// is unused weight. Stripping it also keeps the file out of a secret scanner's
// generic-key heuristic. A signature-verifying successor re-embeds the keys.
```

The sweep kept three lines at `internal/scan/cttail.go:25`:

```go
// Embedded rather than fetched live, so the log set is deterministic and needs no network.
// Each log's public key is stripped: no signature is verified here, so it is unused weight.
// Stripping the keys also keeps this file out of a secret scanner's generic-key heuristic.
```

### The artefact, measured

`internal/scan/log_list.json`, 27,989 bytes. `"version": "89.34"`,
`"log_list_timestamp": "2026-08-29T13:39:10Z"`. Eight operators — Google, Cloudflare, DigiCert,
Sectigo, Let's Encrypt, TrustAsia, Geomys, IPng Networks. **48 log entries**: 26 under `logs[]`
(RFC 6962) and 22 under `tiled_logs[]` (static-ct-api). By `state`: 37 `usable`, 6 `qualified`,
3 `retired`, 2 `readonly`.

**Zero of the 48 entries carry a `key` field.** Upstream publishes one on every entry.

### Two readers, and neither fetches

`internal/scan/cttail.go:29` is `//go:embed log_list.json`. Line 30 is `var embeddedLogList []byte`.
Exactly two functions unmarshal it, and there is no third:

- `SelectTailLogs(now)` (`internal/scan/cttail.go:78`) — the **tail's** reader. It applies
  `tailReadableState` (`usable` or `readonly`) and `tailCoversNow` (`temporal_interval` covering now,
  plus `nextShardHorizon` of 366 days ahead).
- `AllLogs()` (`internal/scan/ctverify.go:405`) — the **verification** reader. It applies **no state
  filter and no temporal filter**, and returns every entry that has an id and a URL.

**No path in the tree fetches a log list.** `ctTailFetcher` reaches `get-sth`, `get-entries`,
`checkpoint` and `tile/data`. `ctVerifyFetcher` reaches `get-proof-by-hash` and the tile paths.
Neither reaches a log-list URL, and no code constructs one.

### The tail's reader runs inside the dispatch transaction

`internal/queue/queue.go:120` opens the transaction, `:126` takes
`pg_advisory_xact_lock`, `:150` calls `fanOutCTTail`, and `:163` commits.
`internal/queue/cttail.go:292` calls `scan.SelectTailLogs(d.now())` inside that call.

So the log set is chosen between `BEGIN` and `COMMIT`, under an advisory lock. A fetch there holds a
Postgres transaction and a lock open across a third-party round trip.

### The SPEC says the opposite, in one sentence

[`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.3 opens:

> Follow logs from the live `log_list.json` (v89.34 as of 2026-08-29) where:

The parenthetical names a version and a date. The word before it is *live*. Both readings are
genuinely defensible on the sentence alone, and §1 below rules between them.

Nothing else in the corpus states either rule. §8's *"Deferred to fog"* list carries three entries —
runtime failover, in-UI operator-key entry, durable issuance findings — and **a live-refresh path is
not among them**. The deleted comment asserted a deferral no document records.

## Decision

> **The CT log list is a build-time artefact. It is committed to this repository, embedded in the
> binary by `//go:embed`, and pinned in the image the release signs. No code path fetches it at run
> time and none is added. Refreshing the list is a release act — a commit, a review and a new image —
> never a runtime fetch. The pinned snapshot carries no log public key, because no code path verifies
> a CT log signature; a signature-verifying successor re-embeds the keys and is a change to this
> Decision.**

### 1. The shipped code is correct, and §4.3 overstates

**The ruling, plainly: the code stays as it is and the SPEC sentence is withdrawn.**

Two readings of *"the live `log_list.json`"* were open, and both are defensible on the sentence
alone.

| Reading | What it says | Verdict |
| --- | --- | --- |
| **A — fetch it** | *live* modifies the act. Go and get the document at run time, so the log set follows upstream | **Withdrawn.** It is not what the code does and it is not what this ADR rules |
| **B — the current published one** | *live* modifies the document. Use the real published list rather than a hand-written or stale one, and the parenthetical names which | **The intended reading, and it still does not save the sentence** |

**Reading B loses as a defence of the sentence, and this is the part worth stating.** B is almost
certainly what the author meant: the parenthetical *(v89.34 as of 2026-08-29)* is exactly how a
person names a pinned version, and a sentence that meant *fetch it* would not name one. So B is the
better reading of the author.

It is still not the better reading of the **sentence**, because ADR-0058's test is not about
authorial intent. Its test is: *"if the superseded sentence, read alone and out of context, would
cause a competent session to build or specify the thing — it is not withdrawn."* `log_list.json` is
the filename of a document published over HTTP. A session holding §4.3 and nothing else reads *follow
logs from the live `log_list.json`* and writes a fetch. That is a build instruction, in the operative
voice, and it owes the mark whichever reading its author held.

So the withdrawal is narrow. **What is withdrawn is the word *live*, and nothing else in §4.3.** The
`state` filter, the `temporal_interval` filter, the two mandatory client implementations and the
`url`/`monitoring_url` discriminator are all shipped, correct and untouched. §4.3's version and date
survive with their meaning made explicit: they name **which snapshot is embedded**, not a document to
go and get.

### 2. The log list is a trust input, so it is pinned rather than fetched

This is the ground, and the two properties the deleted comment led with are consequences of it
rather than reasons for it.

**The list decides which parties we believe.** On the tail it decides which endpoints the worker
contacts, per log, on a 300-second cadence. On the verification path `FindLogByLogID` decides which
log may satisfy an inclusion proof — so an entry in this file is a party whose answer converts a
certificate to `logged in CT`. Adding a row is adding a trust anchor. That is not configuration.

**A fetch would move that decision out of the release and hand it to a third party, between two
ticks.** The list would change without a commit, without a review, without a version and without
anything an operator could point at afterwards. Nothing in the tree would record which list a given
scan ran against.

**A fetch would also make every CT scan depend on a third party being reachable at scan time.** The
tail's fan-out would fail, or fall back to a stale copy, whenever that host was unreachable — a
dependency the tail does not otherwise have, since the tail already talks only to the logs it
follows.

**This is the posture the repo already holds one layer up and one layer down.**
[ADR-0138](./0138-a-release-pins-every-byte-it-builds-so-it-delegates-no-build-step-and-anchors-identity-in-its-own-workflow.md)
§1 refuses a delegation that *"does not freeze what the workflow fetches at run time"*, and states
the consequence exactly: adopting it *"would not satisfy the rule. It would **replace** the rule with
a trust-Docker rule for the build half."* A live log list is that swap, one layer down: it replaces
a pin rule with a trust-the-log-list-publisher rule for the log set.
[ADR-0139](./0139-the-probers-origin-is-the-image-that-carries-it-and-a-host-bounds-the-binary-rather-than-verifies-it.md)
§1 states the property that survives when the artefact is pinned instead: *"an operator who verifies
the worker image has verified the prober."* Substituting the log list is exact — an operator who
verifies the image has verified the log set, and the image's signature, provenance attestation and
SBOM already cover these 27,989 bytes.

[ADR-0003](./0003-third-party-source-consent-bar.md) bounds this rather than deciding it. It rules
that **which third parties may be queried without the operator saying so** is authored by the
release, which is why `db/migrations/18500_source_state.sql` puts the catalogue *"in the binary
rather than the database"* and keeps only the operator's override in Postgres. The log list is the
same kind of authored fact one level finer, and this ADR is what states it for the log list, because
ADR-0003's own subject is the catalogue.

### 3. Determinism and the transaction are real, and they are consequences

The deleted comment named both, and both hold.

- **Deterministic and testable.** `SelectTailLogs` is a pure function of the embedded bytes and the
  clock, so a test can assert exactly which logs are followed at a given instant with no network and
  no fixture server.
- **No network step inside the transaction.** §Context measures where the call sits. A fetch there
  would hold `pg_advisory_xact_lock` across a third-party round trip, on the dispatcher's one-minute
  tick.

**They are not the ground.** Both could be bought another way — a cached fetch outside the
transaction, refreshed on its own schedule, is deterministic within a tick and holds no lock. That
design is refused by §2, not by these two. Stating them as the reason would leave the rule open to
the first person who solves them.

### 4. Refreshing the list is a release act, and this is what it is

Four steps, in order:

1. Replace `internal/scan/log_list.json` with the upstream document.
2. **Strip every `key` field**, per §5.
3. Review the diff for which logs entered and which left. That diff is the trust change, and it is
   the reason this passes through review at all.
4. Ship a release. The image carries it and the release's provenance covers it.

**It moves no schema and no derivation version.** It is not a migration, it does not touch
`db/migrations/`, and it `Break`s nothing: the tail admits `Name`s and observes nothing (ADR-0027),
so no timeline and no golden-corpus row is a function of these bytes.

**A refresh is not on a cadence and this ADR does not put it on one.** §6 states what it costs to
leave it, in measured terms.

### 5. The snapshot carries no log public keys

Three grounds. The first is sufficient on its own and the other two are why the answer is comfortable
rather than grudging.

**Nothing in the tree verifies a CT log signature.** `SelectTailLogs` reads `description`, `log_id`,
`url` or `monitoring_url`, `state` and `temporal_interval`. `AllLogs` reads `log_id`, `url` or
`monitoring_url` and `description`. `ParseSTH` reads `tree_size` and keeps the raw body.
`ParseCheckpoint` splits the note and reads the size. `internal/queue/cttail.go:130` says so at the
one place a reader might assume otherwise:

```go
// An append-only tree cannot shrink, so a tree below the cursor is a fork or a rollback.
// This shrink check is not the consistency proof, and no signature is verified here.
```

§4.2 already rules that the stored signed head exists so a later poll **can** prove continuity, and
§4.4 rules that running the consistency proof is *"opportunistic, not mandatory."* Neither states
that the signed head's **signature** is unchecked, and neither is what this limb rests on. This limb
rests on the code: there is no verifier, so there is nothing a key would feed.

**A bare public key in a committed JSON file trips `gitleaks`, which is a blocking required check.**
This is measured in this repository rather than assumed. `.gitleaks.toml`'s allowlist already carries
a second entry for exactly this heuristic, and describes it: *"a domain identifier gitleaks'
generic-api-key rule flags on the `key = "..."` keyword+entropy heuristic."* A document with 48
`"key": "<base64 SPKI>"` fields is that shape 48 times. The alternative is a path-scoped suppression,
and `.gitleaks.toml`'s own header states why the repo refuses those: the allowlist is *"scoped to the
exact literal, not to a file or path, so any OTHER secret … still trips the scan."*

**A stripped file is honest about its own reach.** A reader who opens the snapshot and finds keys
concludes the tail verifies something. It does not.

**The obligation on a successor, stated as a bound rather than a hope: a change that verifies a CT
log signature re-embeds the keys, and it is a change to this Decision.** It is not a small change,
because §Context's measurement — no verifier anywhere — is what §5 rests on, and adding one moves it.

### 6. Two gaps, one document

Gap 2 rules where the artefact comes from. Gap 3 rules what it contains. They are one decision about
one file, and splitting them breaks the procedure.

**§4 step 1 is a wrong instruction without §5 step 2.** A maintainer holding only the provenance rule
refreshes by copying the upstream document wholesale, re-embeds 48 public keys, and fails the
`gitleaks` gate — or worse, allowlists a path to get past it. The refresh procedure is not statable
without the content rule, so the two belong in one document.

### 7. A live-refresh path is refused, not deferred

The deleted comment said *"a live-refresh path is deferred to fog."* §8's deferred list does not
carry it and never did.

**This ADR refuses it.** A deferral says *not sharp enough yet, and a later effort may take it up*.
That is the wrong shape here, because the objection in §2 is not about sharpness or cost. It is that
the list is a trust input, and no amount of later engineering makes a runtime fetch stop moving the
trust decision out of the release.

**What would reopen it, named rather than left implicit:** a signed log list whose signature this
project verifies against a key pinned in the image. That changes the question, because the fetched
bytes would then be covered by an anchor the release still holds. Nothing of that shape is built or
specified today, and a refresh under it would still be an explicit act rather than a silent follow.

### 8. What this rule does not reach

- **Which logs are selected.** §4.3's `state` and `temporal_interval` rules decide that and are
  untouched.
- **The `nextShardHorizon` constant** (366 days, `internal/scan/cttail.go:34`). It is a selection
  parameter, not a provenance rule.
- **Whether the tail runs at all.** That is the `ct-tail` source toggle, ADR-0003 and ADR-0189.
- **The `ct_log_cursor` rows.** They are durable per-log state, keyed by `log_id`, and they survive a
  snapshot refresh untouched. A log that leaves the snapshot leaves its cursor row behind, and a log
  that returns resumes from it.

## Consequences

- **This ADR changes no Go code.** `internal/scan/cttail.go`, `internal/scan/ctverify.go` and
  `internal/scan/log_list.json` are correct as they stand.
- **[`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.3 gains a withdrawal** at its
  own first sentence, under ADR-0058, with replacement wording. Recorded in this issue's manifest.
- **§8's *"Deferred to fog"* list gains nothing.** §7 refuses the live-refresh path rather than
  deferring it, so an entry there would record a status this ADR does not hold. The deleted comment's
  *"deferred to fog"* clause is withdrawn as unfounded: no document ever carried that deferral.
- **`internal/scan/cttail.go:25` gains this ADR's citation** on the surviving three-line block.
  Recorded in this issue's manifest.

### The cost of the pin, measured

**The tail's followed-log set decays to zero, and nothing reports it.** `SelectTailLogs` applied to
the shipped snapshot, by date:

| Date | Logs the tail follows |
| --- | --- |
| 2026-09-05 | 39 |
| 2027-01-05 | 25 |
| 2027-07-05 | 12 |
| 2028-01-05 | 2 |
| 2028-02-05 | **0** |

The snapshot's latest `end_exclusive` is `2028-01-08`. Every shard in it expires, and no shard minted
after 2026-08-29 is in it to take over. From early 2028 the `ct-tail` Scan fans out **zero jobs**
every 300 seconds, forever, on an install that never takes a release.

**What the operator sees in each case:**

- **A newly qualified log:** nothing at all. No error, no warning, no count. Certificates logged only
  to that log are never seen, and the drift signal for them never fires. Under ADR-0027, ADR-0096 §7
  and ADR-0106 the `ct-tail` Scan carries **no currency bound and no withdrawal power**, so no `Gap`
  opens and `Coverage` reports nothing. The silence is the model working as ruled, and it is why this
  case is invisible.
- **A log retired upstream:** the snapshot still reads its `state` as `usable`, so the tail keeps
  polling it. §4.3 already notes a retired log may 404. The fetch fails, `ctHTTPCause` builds
  `CT log get-sth returned HTTP 404`, and `retryOrDeadLetterCT` retries and then dead-letters a
  Batch. **The operator sees a recurring failed `ct-tail` job on that log, every cadence**, worded as
  a log-side HTTP failure rather than as a stale pin.
- **The decay to zero:** the one surface that moves is the Sources page's CT capabilities card.
  `CTTailLastBatch` reads the newest `ct-tail` Batch, and zero jobs write zero Batches, so
  `TailLastRel` stops advancing and reads as an ever-older *last run*. It says the tail has not run.
  It does not say the log list expired.

- **No surface reports the snapshot's version or age. That is a defect this ruling exposes, and it
  ships as its own ticket.** The pin is correct and its staleness must be legible: the Sources page's
  CT card is the place, and `logList.Version` and `log_list_timestamp` are already parsed off the
  embedded bytes. A stale snapshot must be visible before it is empty, not after.
- **Verification does not decay, and its exposure is different.** `AllLogs` applies no state or
  temporal filter, so all 48 entries stay available to the verifier for the life of the image. A
  certificate whose SCT names a log minted after the snapshot misses `FindLogByLogID`, and the
  verification returns `unverifiable`, never `not-logged`. That is ADR-0193's rule holding, and it is
  why the pin's staleness is safe on this path in a way it is not on the tail's.
- **`CONTEXT.md` gains nothing.** The log list is an implementation input to one Scan. It carries no
  timeline, mints no subject and is not a domain term.
- **A refresh is a small, reviewable change.** One file, one diff, four steps in §4. Its cost is a
  release, and the table above is what an operator pays for not taking one.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Fetch the log list at run time, as §4.3's reading A says** | It hands a third party the power to change which logs we trust between two ticks, with no commit, no review, no version and no record of which list a scan ran against. On the verification path an added entry becomes a party whose answer converts a certificate to `logged in CT`. It also makes the tail's fan-out depend on a host it otherwise never contacts, inside a Postgres transaction holding an advisory lock |
| **Fetch it on a slow cadence and cache it in `source_state` or a new table** | Solves determinism and the lock, and neither is the objection. §3 states this: both properties can be bought this way, and the trust move in §2 is untouched. It also invents durable state whose provenance nothing records, so an operator could not answer *which list did last night's scan use* |
| **Ship the list as an operator-editable file on a volume** | It puts the trust anchor in the deployment, where ADR-0139 §4 already measured what that costs: the operator's controls bound what a thing may do and *"do not prove what it is."* [ADR-0144](./0144-the-verge-core-body-is-compiled-in-and-an-operator-edit-layers-over-it.md) §2 refused the same shape for `verge-core` on the sharper version of this ground — a replaceable body lets the operator move the half the release is supposed to author |
| **Refresh the snapshot on a cadence, enforced by a scheduled CI job** | A cadence check on a third-party document is a watch on the world moving, and [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s #102 ruling keeps that population bounded and deliberate. It also does not fix what is actually broken, which is that no operator can see how old their snapshot is. The Consequences ticket fixes that |
| **Keep the log public keys, so a future verifier needs no refresh** | 48 `"key": "<base64 SPKI>"` fields is the exact keyword-plus-entropy shape `.gitleaks.toml`'s own allowlist documents, against a blocking required check, and the only way past it is the path-scoped suppression that file's header refuses. The keys also feed nothing: there is no verifier anywhere in the tree, so they would be committed weight that misdescribes the code's reach |
| **Verify the signed head's signature now, so the keys earn their place** | It is a real feature with a real design, and it is not this document's subject. §5 states the bound instead: a change that verifies a signature re-embeds the keys and is a change to this Decision. Doing it here would mix a CT-verification feature into a provenance ruling |
| **Two ADRs, one per gap** | §6 measures the failure: the refresh procedure in §4 is a wrong instruction without §5's stripping step, so a maintainer holding one document does the wrong thing. Both rule the same 27,989 bytes |
| **State it in [`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.3 alone, with no ADR** | §4.3 is the site that is wrong. Correcting a spec sentence records what to do and not why, and ADR-0058's own Rationale is that the reasoning lives at the superseding site. Both edits happen; only one of them is a decision |
| **Record it on [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)** | ADR-0106 rules the bulk `ct` Scan over crt.sh. It predates the tail, names no log list, and reads no logs directly. §4.2 of the spec states the tail *"shares nothing with bulk `ct` except the CT theme"* |
