# The crt.sh replacement — certificate-transparency sources

- **Status:** Accepted — spec content for the CT-source wayfinding effort, [map #854](https://github.com/winniel123/verge-asm/issues/854)
- **Ticket:** [#860 Assemble the crt.sh-replacement spec](https://github.com/winniel123/verge-asm/issues/860)
- **Rulings that bind this spec:** [ADR-0027](../adr/0027-a-source-may-admit-without-observing.md) (admit without observing), [ADR-0020](../adr/0020-a-conflict-needs-two-enumerable-sources.md) (a conflict needs two enumerable sources), [ADR-0106](../adr/0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md) (the CT poll is a Scan), [ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) (a secret is held only where its act runs)

This document is the **handoff artifact** of the CT-source wayfinding map. It assembles the five
settled decisions into one buildable spec. **No code lands from the map.** This document hands off to
implementation; it does not itself implement.

It replaces the position where crt.sh is the **sole** certificate-transparency (CT) source today. It
does **not** remove crt.sh. crt.sh stays as the keyless fallback (§2).

## Provenance

Every statement below traces to a closed map ticket. The mark in each section names its source. A
statement without a mark is assembly glue, derived from the marked decisions.

| Mark | Source ticket |
| --- | --- |
| `[R-primary]` | [#855 Research: validate Cert Spotter as the bulk CT primary](https://github.com/winniel123/verge-asm/issues/855) |
| `[R-logs]` | [#856 Research: CT-logs-direct protocol and ecosystem facts](https://github.com/winniel123/verge-asm/issues/856) |
| `[D-bulk]` | [#857 Decide the bulk CT source model and reliability bar](https://github.com/winniel123/verge-asm/issues/857) |
| `[D-logs]` | [#858 Design the CT-logs-direct tail and verification capability](https://github.com/winniel123/verge-asm/issues/858) |
| `[P-ui]` | [#859 Prototype the focused Sources-UI for the CT source experience](https://github.com/winniel123/verge-asm/issues/859) |

Two research findings live on unpushed branches, cited but not landed:
`docs/research/ct-bulk-primary-2026-08.md` (branch `research/cert-spotter-primary`) and
`docs/research/ct-logs-direct-2026-08.md` (branch `research/ct-logs-direct-facts`). The UI prototype
lives on branch `prototype/ct-source-859` at `prototypes/ct-source/index.html`.

---

## 1. Three capabilities under one theme

Today crt.sh is one flat catalogue row and one `ct` Scan. This spec makes CT **one theme with three
capabilities**. `[D-bulk]` `[D-logs]` `[P-ui]`

| Capability | What it does | Query shape | Keyed? | Ships as |
| --- | --- | --- | --- | --- |
| **Bulk-by-name** | Enumerate names under a domain from a CT index | `%.example.com` in one query | Primary needs a key; fallback keyless | The `ct` Scan — one source active per config (§2) |
| **Drift tail** | Watch new issuance for names already known | Forward delta of the logs, filtered against known names | Keyless | A new `ct-tail` Scan, opt-in (§4) |
| **Verification** | Confirm one specific certificate is logged in CT | Point-check from an SCT or the cert bytes | Keyless | A non-Scan on-demand operation (§5) |

**The bulk-by-name / logs-direct split is a protocol fact, not a preference.** `[R-logs]` RFC 6962 and
static-ct-api have **no query-by-domain**. The logs are append-ordered Merkle trees read by position
(`get-entries?start=&end=`), about 5.7 billion entries per log. crt.sh and Cert Spotter are *indexes*
built over the logs; the logs themselves carry no name index. So the drift tail and verification can
**never** do bulk-by-name. This invariant guards the whole design against bulk-mode drift.

---

## 2. Bulk-by-name — crt.sh demotion and the Cert Spotter primary

### 2.1 Two sources, one Scan, one active per config `[D-bulk]`

Two CT sources live under the single `ct` Scan. Exactly **one** is active per configuration.

| Slug | Role | Consent | `DefaultOn` | Reliability bar |
| --- | --- | --- | --- | --- |
| `certspotter` | Operator-keyed bulk-by-name **primary** | `operator-credentialed` | `false` | Must clear all three limbs (§3) |
| `crtsh` | Keyless bulk-by-name **fallback** | `unencumbered` | `true` | **Exempt** (§3) |

- **`certspotter` is a repurpose, not a new slug.** `[D-bulk]` The existing **barred** `certspotter`
  catalogue entry becomes the keyed primary: set `Barred: false`, `Consent: operator-credentialed`,
  `DefaultOn: false`, and add a runner. `consent` names the instrument, so the operator-keyed door is
  a different state from the barred unauthenticated one. No new slug, no collision.
- **`crtsh` is unchanged** as a catalogue entry: `DefaultOn: true`, `unencumbered`, keyless.

### 2.2 Why Cert Spotter, and the honest gap `[R-primary]`

Cert Spotter qualifies as the bulk-by-name primary in its **keyed** form. Its authenticated tier
clears the terms limit: the "personal or evaluation purposes" restriction is scoped to
*unauthenticated* queries, and SSLMate's own docs direct production users to authenticate.

**Honest gap, preserved:** no separate master API terms page was locatable for the authenticated tier.
The implementer must **confirm the production terms at sign-up** before shipping the key. If the
authenticated terms do not clear the consent bar ([ADR-0003](../adr/0003-third-party-source-consent-bar.md)),
this primary does not ship and crt.sh remains the sole bulk source.

No newer keyless, terms-clean, name-indexed CT index beats crt.sh as of 2026-08. MerkleMap needs a key
and bars automated or commercial use; CertIndex, Apify, and ctlogs.dev are keyed, restricted, or crt.sh
wrappers. So crt.sh stays the keyless default.

**Query shape (confirmed):** `domain=`, `include_subdomains=true`, `expand=dns_names`,
`expand=issuer`, `after=` cursor. Free authenticated tier is 10 full-domain queries per hour.

### 2.3 Config-time selection — by key presence `[D-bulk]`

Selection happens **at worker wire-time**, by the presence of the operator key. There is no runtime
choice.

| `VERGE_CERTSPOTTER_TOKEN` | Active `ct` source |
| --- | --- |
| Set | `certspotter` |
| Absent | `crtsh` |

- Exactly **one** CT source is active per `ct` Batch cycle. There is no simultaneous dual fetch.
- **Corroborate-only is a property across runs, never a double-fetch.** A name may hold citations from
  both sources over time; a single Batch cycle fetches from one source only.
- If the operator toggles the **selected** source off in `source_state`, the `ct` Scan fires over an
  **empty scope** — a legible state, not an error, and **no auto-fallback**. Runtime failover stays
  deferred (§7).
- While a key is set, `crtsh` is **standby**: catalogued and `DefaultOn`, but active only when no key
  is configured. The Sources UI labels it as the fallback so its toggle is not a silent no-op (§6).

### 2.4 Operator key location — worker-only `[D-bulk]`

Per [ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md),
the key is held only where its act runs.

- The key is **worker-only**, provisioned as the environment variable `VERGE_CERTSPOTTER_TOKEN` (the
  shipped `VERGE_*` convention). The `web` process **never** reads it.
- The Sources UI renders "operator key required — set on the worker" from catalogue metadata. It never
  reads the value.
- **In-UI key entry** (web writes the token to the worker volume, ADR-0053's intended-but-unbuilt
  pattern) is **deferred** (§7).

### 2.5 Throttle — per source `[D-bulk]`

Replace the single-source throttle row `crtsh_throttle` with a per-source reservation table.

```
ct_throttle(source TEXT PRIMARY KEY, next_free_at TIMESTAMPTZ)
```

- One reservation row per source slug. The interval is supplied per-source from Go config.
- `crtsh` keeps its 12 s interval. `certspotter` takes a conservative interval from its documented caps,
  **re-measured against the reliability bar** (§3).
- The `Reserve` query is unchanged. The interval is already an argument to it.

### 2.6 Admission mapping — a decoder translates shape, never fact `[D-bulk]`

Per [ADR-0027](../adr/0027-a-source-may-admit-without-observing.md), a completed Batch produces
`admitted_name` rows. It admits without observing: no facet, no observation.

- A **per-source decoder** translates Cert Spotter JSON (`after=` cursor pagination, `dns_names`
  arrays) into the same candidate-name list. It consumes the full cursor into **one `Batch` per name
  scope**.
- The **shared pure filter** is reused unchanged: wildcard refusal
  ([ADR-0060](../adr/0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md)), scope
  filter ([ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md)), and the 100k cap.
- `admitCT` stamps `Source: "certspotter"` or `Source: "crtsh"`. The read path (`ListAdmittedNames`,
  `SELECT DISTINCT name`) is already source-agnostic and is **unchanged**.
- Cert Spotter has **no 999-result cap**, so it is more complete than crt.sh.
- **Per-slug enablement gate.** Enablement stays `source_state(slug, enabled)` overriding catalogue
  `DefaultOn`. Replace the hardcoded single-slug `crtshEnabled` gate with a **per-slug** gate that the
  `ct` fan-out consults for the selected source.

### 2.7 Corroborate-only — zero citation migration `[D-bulk]`

A `certspotter` admission is a **new** `admitted_name` row citing its own `Batch`.

- Existing `crtsh` rows are **untouched**: no backfill, no re-pointing, no cross-source dedup.
- A name admitted by both sources simply holds two citations. Per
  [ADR-0020](../adr/0020-a-conflict-needs-two-enumerable-sources.md), a conflict needs two *enumerable*
  sources; CT is corroborative, so two CT citations never conflict.
- A name leaves only by measured Name Error
  ([ADR-0006](../adr/0006-subjects-leave-by-measurement.md)).
- The code map confirms the write path stamps the source (`admitCT` → `InsertAdmittedName.Source`) and
  the read path is source-agnostic. **No migration is required.**

---

## 3. The reliability bar — measured, not asserted `[D-bulk]`

A **primary** must clear all three limbs. crt.sh is **exempt** as the keyless fallback.

| Limb | Bar | crt.sh measured (for contrast) |
| --- | --- | --- |
| Success rate | **≥ 99%** over a rolling sample of bulk-by-name queries | ~50% (4 of 8 identical requests failed) |
| p95 end-to-end latency | **≤ 5 s** per bulk query | 11.9–59.6 s |
| False-empty | **None** — never an empty result for a name that has certs | Spurious 404s observed |

**The bar is measured, not asserted once.** The spec records the measurement **method** — sample size
and re-measurement cadence — so the bar stays a measured bar. The implementer states these in the
runbook when the primary ships.

- If a configured primary is **below** its bar at run time, the Scan keeps running the primary. There
  is **no silent swap** to crt.sh (runtime failover is deferred, §7). The UI surfaces the degraded
  state (§6).

---

## 4. CT-logs-direct — the drift tail `[D-logs]` `[R-logs]`

The tail watches for **new issuance on names the operator already knows**. It admits in-scope names it
sees, exactly like crt.sh, **and** emits an ephemeral issuance event when a new certificate appears for
a known name — that is the drift signal.

**Design invariant:** the tail reads only **forward deltas** and never backfills history.

### 4.1 Model fit — admission plus ephemeral event `[D-logs]`

- The tail admits like crt.sh: an `admitted_name` row, `authority: inferred`, citing a Batch
  ([ADR-0027](../adr/0027-a-source-may-admit-without-observing.md)).
- It emits an **ephemeral issuance event** when a new certificate appears for a known name.
- It creates **no new facet and no Signal.** CT-logs-direct dissolves all three of ADR-0027's grounds
  for deferring issuance detection: it produces real observations with fingerprints; the forward-delta
  read never touches history, so it cannot conflate; and the log yields the fingerprint join key. A
  **durable, alertable** signal would still force a new facet, so v1 keeps the signal **ephemeral**
  (§7).

### 4.2 Scan shape and cursor `[D-logs]`

- **New Scan kind `ct-tail`**, separate from the bulk `ct` Scan. It fans out **per-log**, not
  per-Seed, and carries cursor state. It shares nothing with bulk `ct` except the CT theme (idiomatic
  under [ADR-0084](../adr/0084-a-scan-is-a-cadence-over-an-exchange-and-an-uncovered-facet-has-no-currency-bound.md)
  and [ADR-0106](../adr/0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md),
  which name Scans by exchange).
- **First durable per-target cursor in the system.** No Scan keeps one today; only the global crtsh
  throttle row exists. A new table keyed by `log_id` stores the tree size (`S_last`) and the last
  signed head (STH / checkpoint):

  ```
  ct_log_cursor(log_id TEXT PRIMARY KEY, tree_size BIGINT, signed_head BYTEA, ...)
  ```

  The signed head lets the tail prove append-only continuity. Running the proof is **opportunistic**,
  not mandatory (§4.4).

### 4.3 Log-set `[D-logs]` `[R-logs]`

~~Follow logs from the **live** `log_list.json` (v89.34 as of 2026-08-29) where:~~

> **"live" WITHDRAWN 2026-09-05 by [#1308](https://github.com/winniel123/verge-asm/issues/1308) /
> [ADR-0190](../adr/0190-the-ct-log-list-is-a-build-time-artefact-pinned-in-the-image-refreshed-only-by-a-release-and-carrying-no-log-public-keys.md)
> ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
> The log list is a **build-time artefact**. A snapshot is committed at
> `internal/scan/log_list.json`, embedded by `//go:embed`, and pinned in the image the release
> signs. **No code path fetches it at run time, and none is added.** Refreshing the snapshot is a
> release act, and a live-refresh path is **refused rather than deferred** (ADR-0190 §7), so §8's
> deferred list gains no entry for it. The version and date name **which snapshot is embedded**,
> not a document to go and get. The selection rule below is untouched and confirmed; it is applied
> to the pinned snapshot. The pinned snapshot also carries **no log public keys** (ADR-0190 §5).

Follow logs from the embedded `log_list.json` snapshot (v89.34, 2026-08-29) where:

- `state` is `usable` **or** `readonly` (both readable), **and**
- `temporal_interval` covers now or the near future (current shard plus the next shard).

Skip `retired` (may 404) and the placeholder `logs[]` entries. `qualified` is optional (live, but not
yet Chrome-counting).

**Two client implementations remain mandatory:** RFC 6962 and static-ct-api (tiled). Some operators
(Let's Encrypt, Geomys, IPng) are tiled-only; Google runs both. Drive the client choice off the
`url` versus `monitoring_url` discriminator, filtered by `state`.

### 4.4 Cadence — a measured bar, and opt-in `[D-logs]`

The firehose is heavy for a self-hosted tool. One usable current shard grows about **9,500
entries/min**, and the tail downloads every new entry to discard the non-matching majority. So:

- The tail ships **opt-in**: a new catalogue source `ct-tail`, `Consent: unencumbered` (keyless),
  **`DefaultOn: false`**, gated by `source_state` exactly like `crtsh`. This is a **third** thing
  beside §2's keyed bulk selection, not part of the `certspotter`-vs-`crtsh` fork.
- **Cadence is a measured bar:** poll each log often enough that the P95 delta stays inside one bounded
  fetch window. Per-operator batch caps apply (Argon 32/request, Nimbus and tiled 256/request).
- Run the **consistency proof opportunistically** — near-free on tiled logs, since the tiles are
  already fetched — never mandatory per poll.

---

## 5. CT-logs-direct — verification `[D-logs]` `[R-logs]`

Verification confirms that **one specific certificate is logged in CT**. It complements the tail: the
**tail** catches certificates in CT we did not expect; **verification** catches certificates we
observed that are not in CT.

**Design invariant:** verification only **point-checks** and never enumerates. It can never start from
a bare name.

### 5.1 Model fit — stateless point-check `[D-logs]`

- An on-demand point-check. It mints **no subject** (ADR-0027 refused a "certificate that exists"
  subject) and stores **no durable result**.
- Given a certificate it computes the `MerkleTreeLeaf` hash and asks the correct log or shard:
  `get-proof-by-hash` (RFC 6962), or the hash tile that covers the leaf's index (static-ct-api).
  ~~or a self-recomputed tile inclusion proof (static-ct-api)~~ **Withdrawn by
  [ADR-0214](../adr/0214-ct-verification-checks-presence-not-log-integrity-so-it-authenticates-no-log-and-its-tiled-arm-compares-one-slot.md)
  §3.** The tiled arm compares the leaf hash against one slot in that tile, and recomputes no
  path to the checkpoint root. The RFC arm does recompute the audit path, against the root the
  same `get-sth` response served. **No log signature is verified on either arm** (ADR-0214 §2),
  so a `logged` verdict states presence and never log integrity.
- It must start from an **SCT or the cert bytes**. There is no query-by-name anywhere (RFC 6962 §4).

### 5.2 Not a Scan `[D-logs]`

Verification is **not a Scan.** A Scan is the declared scheduled dispatch; verification has no schedule
and no scope fan-out. It is an operator-triggered or observation-triggered worker operation.

### 5.3 Capture at handshake — the larger commitment `[D-logs]`

Today we store only chain **fingerprints** — the wrong hash for CT (a plain `sha256(DER)`, not the
leaf hash) — and no cert bytes and no SCTs (`connectoutcome/tls.go`; a grep for `SCT` / `OCSP` across
Go returns nothing). So:

- **Capture SCTs and leaf bytes at handshake time.** Capture **both**: the SCTs (embedded in the cert,
  via the TLS extension, and via the OCSP staple) **and** the leaf certificate bytes. The SCT names the
  log; the bytes give the leaf hash. This is always-on enrichment — the SCTs already sit in the TLS
  state we currently discard.
- **Store them in a fingerprint-keyed side store**, one immutable row per distinct certificate,
  deduped:

  ```
  certificate_material(fingerprint TEXT PRIMARY KEY, der BYTEA, scts BYTEA)
  ```

- **The `certificate` facet value stays untouched.** The observation still records only the
  fingerprint. No CT input feeds the facet value, so ADR-0027's fence stays closed. This matches
  ADR-0027's "Certificate held as an immutable value, shared by fingerprint."

### 5.4 Trigger and result `[D-logs]`

- **Auto-verify** each new `certificate` observation once, plus an **on-demand** re-check.
- The result is an **ephemeral event**: ~~"logged / NOT logged in CT."~~ **"logged / NOT logged /
  unverifiable."** **NOT logged** is the notable signal — an internal CA, or evasion.

  > **WIDENED to three values 2026-09-05 by [#1308](https://github.com/winniel123/verge-asm/issues/1308) /
  > [ADR-0193](../adr/0193-a-stapled-ocsp-response-only-narrows-the-sct-set-and-no-usable-sct-is-unverifiable-never-not-logged.md)
  > ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
  > A verification that finds **no usable SCT** — none presented, none parseable, no log matched, or a
  > log unreachable — is **`unverifiable`**, never `NOT logged`. `NOT logged` is warranted only where a
  > log the certificate's own SCT names answered the inclusion query and did not hold it. The
  > *"NOT logged is the notable signal"* reading above is untouched and confirmed; the third value is
  > what protects it. `unverifiable` emits no operator event today (ADR-0193 §4).

---

## 6. The Sources-UI redesign — focused `[P-ui]`

The redesign is **focused on the CT-source experience only**. The full Sources-page overhaul is out of
scope (§8). The CT section renders as **one "Certificate transparency" theme with three capabilities**,
not the flat catalogue row crt.sh occupies today.

### 6.1 Chosen layout — active-source hero

The section **leads with which bulk source is live and why**, its reliability against the §3 bar, then
the drift tail and verification as a secondary "More CT capabilities" card. This foregrounds
primary-vs-fallback plus reliability — the section's headline job. (Grouped-rows and tabs were the
rejected forks.)

### 6.2 The four asks the section answers

1. **Primary-vs-fallback status** — a status badge on the card header (`primary · Cert Spotter` /
   `fallback · crt.sh`) plus a "dormant" tile naming the source that would take over.
2. **Operator-key field** — **read-only presence**: `detected` / `not set`, sourced from
   `VERGE_CERTSPOTTER_TOKEN` on the worker. The console reads presence only. It **never** shows or
   stores the token. This is §2.4 as-decided. The editable-input fork was shown and **rejected**;
   in-UI key entry stays deferred (§7).
3. **Which source ran** — a run readout line: `last ct scan · <source> · <relative time> · <n> names
   admitted`. The tail and verification carry their own readouts.
4. **Reliability** — three KPI tiles measured against the bar. The primary shows pass/fail per metric.
   crt.sh-as-fallback is shown **bar-exempt** (muted, not failed), matching §3.

### 6.3 Honest edges drawn, not smoothed over

- **Below-bar primary** (degraded run state): success renders in danger, and a callout states the
  primary is under its bar and that **runtime failover is not built (deferred)**. The Scan keeps
  running the primary until the worker is reconfigured. **No silent swap.**
- **Keyless default** (fallback run state): with no key, crt.sh runs and is marked bar-exempt, with a
  hint on how to promote Cert Spotter.

### 6.4 Constraints honoured `[P-ui]`

Current design system only (not the legacy look); CONTEXT.md vocabulary (source / Scan / admit /
Name); no severity ramp on this surface; terse sentence case; mono technical values; light and dark
verified; zero console errors.

The prototype is a throwaway (`prototypes/ct-source/index.html`, branch `prototype/ct-source-859`).
Any UI work uses the `verge-asm-design` skill and re-derives from the design system; the prototype is a
reference, not source markup to lift.

---

## 7. New durable state and schema summary

The implementer adds this durable state. All of it is new; none migrates existing rows.

| Object | Shape | Purpose | Source |
| --- | --- | --- | --- |
| `ct_throttle` | `(source TEXT PK, next_free_at TIMESTAMPTZ)` | Per-source bulk throttle; **replaces** `crtsh_throttle` | §2.5 |
| `ct_log_cursor` | `(log_id TEXT PK, tree_size BIGINT, signed_head BYTEA, ...)` | The tail's per-log forward cursor | §4.2 |
| `certificate_material` | `(fingerprint TEXT PK, der BYTEA, scts BYTEA)` | Fingerprint-keyed side store for verification | §5.3 |

Catalogue and Scan changes:

- `certspotter` catalogue entry repurposed (§2.1); per-slug enablement gate replaces `crtshEnabled`
  (§2.6).
- New Scan kind `ct-tail` and new catalogue source `ct-tail` (`DefaultOn: false`, keyless) (§4).
- Handshake path captures SCTs plus leaf bytes as always-on enrichment (§5.3).

---

## 8. What ships in v1, and what is deferred

### Ships (v1)

- crt.sh demoted to keyless fallback; **never removed** (§2).
- Cert Spotter operator-keyed bulk primary, config-time selection by key presence (§2).
- The measured reliability bar for the primary (§3).
- The keyless opt-in `ct-tail` drift Scan (§4).
- Keyless on-demand verification, with SCT + leaf-byte capture at handshake (§5).
- The focused CT Sources-UI redesign (§6).

### Deferred to fog (out of v1)

These are **in scope for the effort but not sharp enough or not wanted for v1**. Each graduates only if
a later effort takes it up.

- **Runtime failover** to crt.sh when the primary hard-fails at run time. §2.3 confirmed selection is
  config-time only. This graduates only if a later spec decides to retain runtime failover.
- **In-UI operator-key entry** — web writes the Cert Spotter token to the worker volume (ADR-0053's
  intended-but-unbuilt pattern). §2.4 ships the key as a worker env var; §6.2 confirmed the field is
  read-only presence.
- **Durable issuance findings** — a queryable, alertable **issuance facet on `Name`** (ADR-0027's
  "revealed + one message" facet) giving both the tail's drift signal and verification's "not-logged"
  result a durable home. §4 and §5 keep both **ephemeral** in v1. This graduates only if a later effort
  wants history or alerting.

### Out of scope (not this effort)

- **Non-CT sources** (passive DNS, RDAP, brute-force, and so on) — separate efforts.
- **Full Sources-page overhaul** — only the CT-source section is in scope.
