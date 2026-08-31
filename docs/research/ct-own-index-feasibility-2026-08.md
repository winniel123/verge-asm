# Is a self-owned full CT name-index feasible for verge-asm?

- **Ticket:** wayfinder research [#926](https://github.com/winniel123/verge-asm/issues/926) — feeds wayfinder
  map [#925](https://github.com/winniel123/verge-asm/issues/925).
- **Question:** Can verge-asm build and maintain its **own** full Certificate-Transparency name-index — every
  CT log ingested to genesis, SANs parsed and indexed, answering bulk-by-name queries like `%.example.com`
  from that store, with **no third-party index** (no crt.sh, no Cert Spotter) and **no operator API key** —
  and under what **deployment model**?
- **Date of research:** 2026-08-31 (UTC).
- **Status:** research complete. Re-tests the 2026-07 conclusion in
  [`passive-discovery-sources.md`](./passive-discovery-sources.md) §2.1 against live 2026-08 facts.
- **Prior conclusion under test:** §2.1 held that tailing the CT firehose "is not a thing a self-hosted
  small-org tool does" and that bulk-by-name "must come from an index (crt.sh / Cert Spotter)". This note
  re-measures the corpus and asks specifically whether verge could *be* that index.

## 0. Method and honesty note

Every corpus figure below is taken live from primary sources on 2026-08-31 and is labelled `[measured]`:
the Google authoritative log list, and each log's own `get-sth` (RFC 6962) or `checkpoint`
(static-ct-api) endpoint. Sizing that depends on per-entry byte costs is labelled `[estimated]` and states
the measured inputs it is built from. Operator-reported figures (crt.sh, Sunlight) are labelled
`[operator-reported]` with the source and date, because they are somebody else's measurement, not ours.

**The single most important honesty point up front:** a CT log's *tree size* counts **log entries**, not
distinct certificates and not distinct names. A certificate is logged to **two or more** logs (Chrome
requires ≥2 SCTs), a pre-certificate and its final certificate are **separate** entries, and certificates
are **re-issued** every 90 days carrying the same names. So the 30.36 billion entries below over-count
distinct certificates several-fold and over-count distinct *names* by a large factor. This cuts **both**
ways and both are stated plainly: the **index** you end up storing is far smaller than 30B rows, but the
**bytes you must download to build it** are the full 30.36B entries — you cannot deduplicate what you have
not yet fetched.

---

## 1. Corpus size now `[measured]`

Live from `https://www.gstatic.com/ct/log_list/v3/log_list.json`, **version 89.34**, timestamped
**2026-08-29T13:39:10Z**. Of the 48 catalogued logs, **39** are `usable` or `readonly` (the readable set the
own-index would ingest); the rest are `retired` (2 bogus placeholders + Oak2026h2) or `qualified` (Google
ParcelYard/PlumbersArms, 6 logs, live but not yet Chrome-counting — excluded here).

Summing each readable log's own `get-sth` / `checkpoint` tree size on 2026-08-31 (39 of 39 endpoints
answered):

| Split | Logs | Entries `[measured]` |
| --- | ---: | ---: |
| RFC 6962 (`get-entries`) | 23 | **25,757,662,208** |
| static-ct-api (tiled) | 16 | **4,602,485,056** |
| **Total (usable + readonly)** | **39** | **30,360,147,264** |

Largest single logs: Cloudflare Nimbus2026 **6.15 B**, Google Argon2026h2 **2.81 B**, TrustAsia log2026b
**2.66 B**, TrustAsia log2026a **2.64 B**, Google Xenon2026h2 **2.39 B**, Sectigo Tiger2026h2 **2.16 B**.
(Full per-log table in the appendix.)

**This is up ~5× from how §2.1 framed the problem in 2026-07.** That note anchored on the single largest
log then visible — Nimbus2026 at 5.71 B — and on "~5.7 B entries/log, ~40 logs". The *aggregate* readable
corpus is now **30.36 B entries**. The problem has grown, not shrunk.

### Index storage (name-only) `[estimated]`

A name-only index stores `name → cert id, issuer id, not_before, not_after` — no certificate DER. Estimating
from the corpus and public dedup ratios:

- Distinct certificates: entries ÷ (≥2 logs/cert × pre-cert+cert doubling) ≈ 30.36 B ÷ ~4 ≈ **~7–8 B**
  distinct certs `[estimated]`.
- Distinct `(name, cert)` associations after collapsing exact-duplicate rows: order **3–5 B rows**
  `[estimated]` (avg ~2–4 SAN dNSNames per cert, heavily repeating across 90-day re-issuance).
- At ~60–80 bytes/row of stored fields plus B-tree/index overhead (a trigram or reverse-domain index on the
  name is what makes `%.example.com` fast, and it is not cheap), a name-only index lands at roughly
  **1–4 TB** `[estimated]`.

**Real-world anchor for that estimate:** crt.sh's `certwatch_db` — a full name-indexed CT database, the
closest thing that exists to "the own index" — was **7,472 GB logical / 3.9 TB on disk after filesystem
compression** as reported by Rob Stradling (Sectigo) on **2020-07-02** `[operator-reported]`. That figure
*includes* the full certificate DER, so a name-only index is smaller than it — but it is a 2020 number
against a corpus that has since grown ~5×, so today's crt.sh is materially larger. A few TB for a name-only
index is the right order of magnitude and is a floor, not a ceiling.

### Raw download to build it `[estimated from measured bytes/entry]`

Measured per-entry wire cost on 2026-08-31 (one 256-entry `get-entries` batch / one data tile each):

| Interface | Measured bytes/entry | Why |
| --- | ---: | --- |
| RFC 6962 `get-entries` | **5,519 – 9,075** (Nimbus 5.5 K, Argon 6.5 K, Tiger 9.1 K) | base64-inflated JSON carrying `leaf_input` **and** the full issuance chain in `extra_data` |
| static-ct-api data tile | **1,856** (Sycamore) | binary `TileLeaf`, chain carried as SHA-256 fingerprints, no base64 |

Applying ~6.5 KB/entry to the RFC 6962 corpus and ~1.9 KB/entry to the tiled corpus:

- RFC 6962: 25.76 B × 6.5 KB ≈ **167 TB**
- Tiled: 4.60 B × 1.9 KB ≈ **9 TB**
- **Total raw download to genesis ≈ 176 TB** `[estimated]`.

The RFC 6962 half dominates because `get-entries` forces you to download the entire issuance chain, base64,
for every entry, even though a name index needs only the leaf certificate's SANs. There is no leaner
RFC 6962 read. (The tiled interface is ~3.5× leaner per entry, and this matters in §4.)

---

## 2. Ingest cost to genesis `[measured caps → estimated totals]`

**Batch caps are per-operator and measured live 2026-08-31:**

- Google logs cap `get-entries` at **32 entries/request** `[measured]` (requested 256 from Argon2026h2,
  got 32). This is unchanged from §2.1's 2026-07 measurement.
- Cloudflare returns **256/request** `[measured]`; Sectigo Tiger returned **192** `[measured]`.
- static-ct-api data tiles are fixed at **256 entries/tile** by the spec ("Full tiles MUST be exactly 256
  hashes wide"; the code's `CTTileWidth = 256`).

**HTTP fetches to read the whole corpus to genesis `[estimated]`:** ~**176 M** requests for Google's logs
alone (their 32-cap × ~5.6 B Google entries), ~**79 M** for the other RFC 6962 logs at 256, ~**18 M** tile
fetches — **≈ 273 M fetches total**.

**Wall-clock, three ways, each a different binding constraint `[estimated]`:**

| Constraint | Assumption | Time to genesis |
| --- | --- | --- |
| Bandwidth | 176 TB moved at 1 Gbps saturated | **~16 days** (163 days at 100 Mbps; 1.6 days at 10 Gbps) |
| Request rate | 273 M fetches at 200 req/s aggregate across all logs | **~16 days** (63 days at 50 req/s) |
| Parse compute | 30.36 B certs × X.509 parse at 10 k/s/core × 8 cores | **~4–15 days** |

None of these is the *home/office box* case — they assume a saturated datacentre link and a machine that
can hold and index multiple TB. On a "normal box" (a handful of cores, a 1–4 TB disk, an office uplink shared
with real work) the binding constraint is **all three at once, plus disk**, and the honest answer is
**weeks-to-months of continuous, dedicated operation** to catch up once — before any query is answered.

**There is no rate-limit or terms obstacle** — and that is the subtle trap. The Chrome CT Log Policy
*requires* operators to "Not impose conditions on retrieving or sharing data from the logs" and to set "all
rate limits … to ensure that all well-behaved clients can reliably retrieve log entries at a rate greater
than the growth rate of the log," with a worked `ceil(X/Y)` requests/sec floor and a mandate to let clients
"catch up." So the logs are *contractually obliged to let you do this*. The constraint is **not**
permission; it is **physics and per-install duplication** (§4).

**Ongoing forward-maintenance rate once caught up `[estimated from measured shard rate]`:** the
CT-source spec measured one usable current shard growing **~9,500 entries/min** (`#858`, cited in
[`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.4). With roughly ~20 actively-growing
shards across the big operators (each ~half-year 2026h2/2027h1 shard), the aggregate forward firehose is on
the order of **~190,000 entries/min ≈ ~3,200 entries/sec**, downloaded and mostly discarded, **forever**.
This is exactly why the existing `ct-tail` Scan ships **opt-in / `DefaultOn: false`** — it is heavy even as a
*forward* tail. Extending it *backward to genesis* multiplies the one-time cost by the 30.36 B corpus.

---

## 3. Prior art / datasets — is there a shortcut? `[measured / operator-reported]`

The question that decides deployment model (c): does **anyone** publish a bulk CT dump or a syncable
prebuilt name-index a fresh install could pull once, instead of ingesting 176 TB from scratch?

| Project | What it is | Self-host / sync a prebuilt index? |
| --- | --- | --- |
| **crt.sh `certwatch_db`** (Sectigo) | The reference full name-indexed CT DB. Ingests logs directly via `ct_monitor` (libcurl `get-entries` → libpq). Schema open-source at `crtsh/certwatch_db`. | **No bulk dump.** Offers only *query* access (public read-only Postgres `crt.sh:5432`, throttled to 5 concurrent connections per IP — [`passive-discovery-sources.md`](./passive-discovery-sources.md) §2.2). No pg_dump, no replication to third parties. Operator reports the replicas "freezing on an almost daily basis" under load. |
| **Axeman** (CaliDog) | Python CT harvester → **CSV on local filesystem**. | Builds from scratch via `get-entries`. No published dataset. A scraper, not a dataset. |
| **scrape-ct-log / certslurp** | CLI / distributed bulk `get-entries` downloaders "for billions of entries". | Tools to *scrape from scratch*. Neither publishes a synced index. |
| **certstream** (CaliDog) | Real-time **forward** firehose over websockets. | Forward-only stream; no history, no index, no bulk. |
| **Sunlight** (Filippo Valsorda / Let's Encrypt) | A static-ct-api **log** implementation. Its tiles *are* natively mirrorable static assets. | Not an index and not a monitor — it is a *log*. But see §4(c): a tiled log's tiles are the one thing in this whole space you *can* rsync. Sunlight/Tuscolo reports **314 GiB compressed** for its live shards (June 2026), ~2.75 TB provisioned for full retention `[operator-reported]` — that is **one** log's own data, not the aggregate. |
| **ct-woodpecker** (Let's Encrypt) | A log *monitor* (liveness/consistency), not a name index. | No. |
| **Google CT monitor / search** | Google's own indexed search. | No dump; no API to pull the index. |

**Finding:** **no party publishes a bulk CT name-index or a full-corpus dump that a fresh install could pull
once.** Google's CT team has explicitly said there is no easily-accessible dump "except if you use the
`get-entries` endpoint … repeatedly." Every self-hostable tool builds from scratch off `get-entries`. The
**only** natively syncable substrate is the static-ct-api tiles themselves — and those cover only the
**tiled 4.6 B / 30.36 B ≈ 15%** of the corpus. The RFC 6962 **85%** is not mirrorable as static assets; it
can only be scraped entry-by-entry.

---

## 4. Deployment models — cost and verdict per model

### (a) Per-operator full ingest — is it physically possible on a normal box? **No.**

Every verge-asm install would, on first run, download **~176 TB**, issue **~273 M** HTTP fetches, parse
**30.36 B** certificates, and land a **multi-TB** index — a process measured in **weeks-to-months** on a
normal box, before answering a single query — and then hold a **~3,200 entry/sec** forward firehose forever
to stay current. A "normal box" for a small-org self-hosted tool does not have the disk, the sustained
bandwidth, or the tolerance for a multi-week cold start. And because verge-asm is "run by strangers on their
own infrastructure," model (a) means **every install re-downloads the same 176 TB from the same 39 logs** —
individually permitted by the Chrome policy, but in aggregate an abusive, pointless re-scrape of the entire
ecosystem, N times over. **Verdict: infeasible, and more so than in 2026-07.**

### (b) A shared / central verge-run index every install queries. **Feasible — but it is a crt.sh clone.**

Verge runs the ingest **once** centrally (176 TB, multi-TB index, the ~3,200/sec forward tail) and exposes a
bulk-by-name API every install calls. This is *technically* the same system crt.sh already is — including
crt.sh's operational reality (a 3.9 TB+ DB in 2020, replicas "freezing on an almost daily basis," constant
overload). It **reopens the project's foundational "no central deployment" assumption**: verge would run,
fund, and SLA a central service, and every install would depend on it — the exact single-point-of-failure
and third-party-index posture the keyless/no-third-party requirement exists to avoid. **Verdict: feasible,
but it *is* re-building and operating a crt.sh, and it contradicts the requirement that produced the
question.**

### (c) A downloadable / syncable prebuilt index an install pulls, then forward-tails. **Feasible only if verge publishes it — which reduces to (b) plus a large client download.**

No third party publishes such a dataset (§3), so for (c) to exist **verge must be the publisher** — i.e.
verge pays (b)'s full ingest and maintenance cost, then additionally distributes a multi-TB snapshot that
each install pulls (multi-TB per install) and then forward-tails with the existing `ct-tail` engine. The one
genuinely lighter variant: **mirror only the static-ct-api tiled corpus**, which *is* natively syncable
(rsync/HTTP the tiles) — but that is **only ~15%** of names; the RFC 6962 85% (all of Google, Cloudflare,
DigiCert, Sectigo, TrustAsia's RFC logs) cannot be mirrored this way and would be missing. **Verdict:
feasible only as verge-published; full-coverage (c) = (b)'s cost + a heavy client sync; partial (c) via
tiled-only mirror is cheaper but silently drops the majority of issuance.** Also reopens "no central
deployment," because someone still has to run the ingest and host the snapshot.

---

## 5. Verdict and recommendation

**Headline: infeasible as (a) per-operator; feasible only as (b) a verge-run central index or (c) a
verge-published syncable dataset — and both (b) and (c) reopen the project's "no central deployment"
assumption.** This **reaffirms and strengthens** the 2026-07 conclusion: the aggregate corpus is now
**30.36 B entries `[measured]`**, the one-time build is **~176 TB / ~273 M fetches / weeks-to-months
`[estimated]`**, the forward cost is **~3,200 entries/sec forever `[estimated]`**, and **no prebuilt index
or dump exists to short-cut it** `[measured]`. The blocker is not terms or rate limits — the Chrome policy
guarantees retrieval faster than growth — it is **physics and per-install duplication**.

**Recommendation: do NOT build a self-owned full CT name-index as a per-operator capability.** Keep the
[`ct-source-replacement.md`](../spec/ct-source-replacement.md) split exactly as specced:

1. **Bulk-by-name stays an index query** — crt.sh (keyless fallback) / Cert Spotter (operator-keyed
   primary). Owning the index does not beat this at self-hosted scale; it loses to it by ~176 TB.
2. **Forward drift stays the existing `ct-tail` engine** (`internal/scan/cttail.go`,
   `internal/queue/cttail.go`) — opt-in, forward-only, never backfilled to genesis. This note is the
   measured justification for that engine's `DefaultOn: false` and its forward-only invariant.
3. **If a keyless, self-owned bulk capability is genuinely wanted**, it is a **central-deployment decision
   for a wayfinder, not a shippable default** — and the only two shapes are (b) run a crt.sh-equivalent
   central index, or (c) publish a syncable dataset (full = (b)'s cost; or tiled-only ≈ 15% coverage). Both
   require the project to first decide to operate central infrastructure, which is out of scope for a keyless
   self-hosted tool as currently scoped.

The own-index question is answered: **the logs will let you, and it still cannot be done per-operator.**

---

## Appendix — per-log tree sizes `[measured 2026-08-31]`

Source: each log's `get-sth` (RFC 6962) or `checkpoint` (static-ct-api), log_list.json v89.34.

| Log | State | Interface | Entries |
| --- | --- | --- | ---: |
| Cloudflare Nimbus2026 | usable | RFC 6962 | 6,151,018,180 |
| Google Argon2026h2 | usable | RFC 6962 | 2,810,794,117 |
| TrustAsia log2026b | usable | RFC 6962 | 2,660,166,011 |
| TrustAsia log2026a | usable | RFC 6962 | 2,639,218,172 |
| Google Xenon2026h2 | usable | RFC 6962 | 2,389,549,546 |
| Sectigo Tiger2026h2 | usable | RFC 6962 | 2,162,891,295 |
| DigiCert Wyvern2026h2 | usable | RFC 6962 | 1,819,684,098 |
| Sectigo Elephant2026h2 | usable | RFC 6962 | 1,799,258,809 |
| DigiCert Sphinx2026h2 | usable | RFC 6962 | 1,794,664,122 |
| IPng Halloumi2026h2a | usable | tiled | 992,828,052 |
| IPng Gouda2026h2 | usable | tiled | 984,905,970 |
| LE Sycamore2026h2 | usable | tiled | 719,322,817 |
| LE Willow2026h2 | usable | tiled | 693,780,114 |
| Geomys Tuscolo2026h2 | usable | tiled | 563,486,824 |
| Cloudflare Nimbus2027 | usable | RFC 6962 | 298,763,578 |
| Google Argon2027h1 | usable | RFC 6962 | 260,635,167 |
| Sectigo Tiger2027h1 | usable | RFC 6962 | 204,169,184 |
| LE Sycamore2027h1 | usable | tiled | 199,278,727 |
| LE Willow2027h1 | usable | tiled | 189,202,721 |
| Sectigo Elephant2027h1 | usable | RFC 6962 | 187,529,373 |
| Google Xenon2027h1 | usable | RFC 6962 | 169,531,414 |
| DigiCert Wyvern2027h1 | usable | RFC 6962 | 102,328,363 |
| TrustAsia HETU2027 | usable | RFC 6962 | 96,993,674 |
| TrustAsia Luoshu2027 | usable | tiled | 92,271,960 |
| DigiCert sphinx2027h1 | usable | RFC 6962 | 83,987,049 |
| Sectigo Sabre2026h2 | readonly | RFC 6962 | 68,732,793 |
| IPng Halloumi2027h1 | usable | tiled | 60,996,901 |
| IPng Gouda2027h1 | usable | tiled | 59,680,278 |
| Sectigo Mammoth2026h2 | readonly | RFC 6962 | 57,634,084 |
| Geomys Tuscolo2027h1 | usable | tiled | 46,639,356 |
| DigiCert Wyvern2027h2 | usable | RFC 6962 | 27,988 |
| DigiCert sphinx2027h2 | usable | RFC 6962 | 27,915 |
| Sectigo Elephant2027h2 | usable | RFC 6962 | 28,958 |
| Sectigo Tiger2027h2 | usable | RFC 6962 | 28,318 |
| Geomys Tuscolo2027h2 | usable | tiled | 23,207 |
| LE Sycamore2027h2 | usable | tiled | 17,043 |
| LE Willow2027h2 | usable | tiled | 17,065 |
| IPng Halloumi2027h2 | usable | tiled | 16,889 |
| IPng Gouda2027h2 | usable | tiled | 17,132 |
| **Total** | | | **30,360,147,264** |

## Sources

- Google authoritative CT log list (v89.34, 2026-08-29):
  `https://www.gstatic.com/ct/log_list/v3/log_list.json` `[measured]`
- Per-log `get-sth` / `checkpoint` endpoints, queried 2026-08-31 `[measured]`
- RFC 6962 (get-entries, MerkleTreeLeaf): `https://www.rfc-editor.org/rfc/rfc6962.txt`
- static-ct-api (tiles, 256-wide): `https://c2sp.org/static-ct-api`
- Chrome CT Log Policy (no conditions on retrieval; rate ≥ growth; `ceil(X/Y)`; availability):
  `https://googlechrome.github.io/CertificateTransparency/log_policy.html`
- crt.sh DB size — Rob Stradling (Sectigo), crtsh Google Group "Certificate Data ingestion", 2020-07-02:
  `https://groups.google.com/g/crtsh/c/snAFqYtRN3w` `[operator-reported]`
- crt.sh architecture (ct_monitor ingest via get-entries): `https://www.lukeshu.com/blog/crt-sh-architecture.html`
- crt.sh schema: `https://github.com/crtsh/certwatch_db`
- Sunlight (storage sizing; Tuscolo 314 GiB / ~2.75 TB): `https://github.com/FiloSottile/sunlight`,
  `https://sunlight.dev/`, `https://words.filippo.io/run-sunlight/` `[operator-reported]`
- Axeman (CSV harvester): `https://github.com/CaliDog/Axeman`
- scrape-ct-log / certslurp (bulk scrapers): `https://github.com/mpalmer/scrape-ct-log`,
  `https://github.com/chtzvt/certslurp`
- verge-asm CT spec & engine: [`docs/spec/ct-source-replacement.md`](../spec/ct-source-replacement.md),
  `internal/scan/cttail.go`, `internal/queue/cttail.go`
