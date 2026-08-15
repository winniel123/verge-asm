# Passive discovery sources for a keyless first run

**Ticket:** wayfinder research #3 — "What can be discovered about an organisation's internet-exposed assets
using only free, keyless data sources?"
**Date of research:** 2026-07-30 / 2026-07-31 (UTC)
**Status:** research complete; feeds the v1 spec. No code exists yet.

## 0. Scope and method

verge-asm must produce useful discovery on **first run, with no API keys and no account registration**.
This document evaluates every candidate passive source against four questions:

1. **What does it yield** (concretely, as fields)?
2. **What does it miss?**
3. **Rate limits and reliability** — measured live where possible, not quoted from blogs.
4. **Terms-of-service constraint on programmatic use from a self-hosted deployment.**

Point 4 is the hard filter. verge-asm will be run by strangers we have no relationship with, on their own
infrastructure, under AGPL-3.0. A source whose ToS forbids automated/bulk querying, restricts use to
"personal or evaluation purposes", or prohibits commercial use **cannot be a shipped default**, no matter how
good the data is. Such sources may still ship as *opt-in, operator-configured* integrations, where the
operator accepts the terms themselves.

All live measurements below were taken from a single residential/commercial IP on 2026-07-30/31 and are
reproducible with `curl`. Measurements are labelled **[measured]**.

---

## 1. Recommended default set for a keyless first run

**Tier 0 — always on, no ToS risk, no third party:**

| Source | Why |
|---|---|
| Operator's own DNS resolver (A/AAAA/CNAME/MX/NS/TXT/SOA/CAA, PTR in owned ranges) | No third party, no ToS, no rate limit beyond the operator's own resolver. Highest-value single source. |
| DNS wildcard detection (random-label probing per RFC 4592) | Prevents the whole pipeline from poisoning itself with synthetic answers. Mandatory prerequisite, not optional. |
| Operator-supplied zone data (uploaded zone file, or AXFR against a nameserver the operator authorises) | Ground truth. Beats every guess-based technique. This is the defensive tool's structural advantage. |
| IANA RDAP bootstrap → RIR RDAP (`ip`, `autnum`, `entity`, ARIN `entities?fn=`) | IETF standard, keyless, fast, no per-query terms accepted at query time. Org name → all ARIN-registered prefixes works today. |

**Tier 1 — on by default, third-party but clean:**

| Source | Why |
|---|---|
| Certificate Transparency **logs read directly** (RFC 6962 `get-entries` + static-ct-api tiles) | Chrome's CT log policy forbids operators from imposing conditions on retrieving log data. Zero ToS risk. See §2.1 for why this is a *targeted*, not full-firehose, capability in v1. |
| **crt.sh** | The only keyless way to get "all certs for `%.example.com`" in one request. Ships as default with hard 5 req/min throttle, aggressive caching, and graceful degradation — it *will* return 503. |
| ~~RIPEstat Data API (`network-info`, `announced-prefixes`, `searchcomplete`)~~ | ~~Keyless, no registration under 1000 req/day, fast, fills ARIN's gaps (RIPE/APNIC/LACNIC/AFRINIC + BGP reality). **Caveat in §4.5 — non-commercial only.**~~ **Withdrawn in two halves — see below the table.** |
| Common Crawl index (CDX) | Keyless, generous, open data licence. Weak signal but free. |

> **The RIPEstat row is withdrawn here, at the site that specifies it
> ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)), and
> the two halves fell to different decisions.**
>
> - ***On by default* fell to [#15](https://github.com/winniel123/verge-asm/issues/15) /
>   [ADR-0003](../adr/0003-third-party-source-consent-bar.md)** on 2026-08-13: RIPEstat ships **off**
>   under `operator-accepted`, and [#19](https://github.com/winniel123/verge-asm/issues/19) made that
>   indefinite. Nothing in this note was marked at the time.
> - **`announced-prefixes` and *BGP reality* fell to
>   [#126](https://github.com/winniel123/verge-asm/issues/126) /
>   [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md)** on 2026-08-15:
>   **a routing announcement names who carries packets toward a prefix, never who controls what listens
>   in it.** The BGP leg ships nowhere in v1, under RIPEstat or any other instrument, so
>   [#47](https://github.com/winniel123/verge-asm/issues/47)'s scoping of RIPEstat's toggle to the
>   org→prefix leg is now the whole of it. See
>   [`non-arin-prefix-coverage.md`](./non-arin-prefix-coverage.md) §14.
>
> What survives of the row is `searchcomplete` and `network-info` as **org→prefix** and IP→ASN calls,
> behind an `operator-accepted` toggle. §4.5's measurements are all correct and none is re-taken.

**Tier 2 — shipped but off by default, operator opts in:**

| Source | Why off |
|---|---|
| SSLMate Cert Spotter (unauthenticated) | ToS scopes unauthenticated use to "personal or evaluation purposes". Fine if the operator supplies their own key. |
| mnemonic Passive DNS | 1000 req/day and no subdomain enumeration; useful for reverse IP→name, but the quota is a shared-fate resource. |
| Team Cymru IP→ASN (bulk netcat / DNS) | Null-routes IPs that hammer the whois server with individual queries. Safe only with the bulk interface implemented correctly. |
| Subdomain brute-forcing | Active, noisy, resolver-abusive, and largely redundant given zone-file input + CT. See §3.5. |

**Explicitly NOT default (ToS-blocked):** HackerTarget, bgp.tools, ICANN CZDS, CIRCL Passive DNS, Rapid7
Sonar, Wayback CDX. Reasons in §6.

---

## 2. Certificate Transparency

### 2.1 CT logs read directly (RFC 6962 / static-ct-api)

**Standard.** RFC 6962 §4 defines the log client HTTP API: `get-sth`, `get-sth-consistency`,
`get-proof-by-hash`, `get-entries`, `get-entry-and-proof`, `get-roots`.
<https://www.rfc-editor.org/rfc/rfc6962.txt>

`get-entries?start=&end=` returns an array of `{leaf_input, extra_data}`, base64-encoded — `leaf_input` is a
`MerkleTreeLeaf` containing a `TimestampedEntry` (`timestamp`, `entry_type` ∈ {`x509_entry`,
`precert_entry`}, `signed_entry`), `extra_data` carries the issuance chain (RFC 6962 §3.4, §4.6). Parsing
the certificate out gives you Subject CN and **subjectAltName dNSNames** — the actual asset signal —
plus issuer, validity window, and key material.

**Batch size is per-operator, not specified.** RFC 6962 §4.6: *"Logs MAY restrict the number of entries that
can be retrieved per 'get-entries' request. If a client requests more than the permitted number of entries,
the log SHALL return the maximum number of entries permissible."*

**[measured]** Requesting `start=0&end=1023` from each log returned:

| Log | Entries returned | Latency |
|---|---|---|
| Google Argon2026h2 | **32** | 0.92 s |
| Cloudflare Nimbus2026 | **256** | 1.27 s |
| DigiCert Sphinx2026h2 | **256** | 1.15 s |
| Sectigo Tiger2026h2 | **256** | 0.85 s |
| TrustAsia log2026a | **32** | 1.57 s |

**[measured]** Current tree sizes from `get-sth`: Google Argon2026h2 = **2,248,107,238** entries;
Cloudflare Nimbus2026 = **5,712,661,045** entries.

**This is the decisive number.** Reading Nimbus2026 alone at 256 entries/request is ~22.3 million HTTP
requests. There are 34 usable + 6 qualified logs. **Tailing the CT firehose is not a thing a self-hosted
small-org tool does.** v1 must not attempt it. CT-log-direct reading is only viable in verge-asm as:
(a) a *tail* — poll `get-sth`, read only entries appended since the last run, for near-real-time drift
detection on an already-known asset set; or (b) a verification path for a specific certificate.
Bulk historical "what certs exist for example.com" must come from an index (crt.sh / Cert Spotter), because
CT logs are **append-ordered, not name-indexed** — there is no query-by-domain in RFC 6962 at all.

**The log ecosystem is mid-migration and RFC 6962 is not the whole picture any more.**
**[measured]** From Google's authoritative log list `https://www.gstatic.com/ct/log_list/v3/log_list.json`
(version 89.3, timestamped 2026-07-30T13:36:39Z), the split is:

- **RFC 6962 (`logs`, with `url` + `/ct/v1/get-entries`):** Google Argon/Xenon, Cloudflare Nimbus,
  DigiCert Wyvern/Sphinx, Sectigo Elephant/Tiger/Mammoth/Sabre, TrustAsia.
- **Tiled / static-ct-api (`tiled_logs`, with `submission_url` + `monitoring_url`, NO `get-entries`):**
  all of Let's Encrypt (Sycamore, Willow), Google ParcelYard/PlumbersArms, Geomys Tuscolo,
  IPng Halloumi/Gouda, TrustAsia Luoshu.
- **Let's Encrypt `Oak2026h2` is `retired`.** Let's Encrypt now runs *only* tiled logs. Any implementation
  that hardcodes `https://oak.ct.letsencrypt.org/…/ct/v1/get-entries` is already broken — verified live,
  the Oak hostname no longer resolves.

Tiled logs use the C2SP static-ct-api: `<monitoring prefix>/checkpoint` (the STH as a signed note),
`<monitoring prefix>/tile/<L>/<N>` (Merkle tiles, 256 hashes / 8192 bytes when full),
`<monitoring prefix>/tile/data/<N>` (the `TileLeaf` entries), and `<monitoring prefix>/issuer/<fingerprint>`.
There is **no `get-entries` equivalent**; a monitor fetches static assets and reassembles.
<https://c2sp.org/static-ct-api>

**Implication for the spec:** any "read CT logs directly" feature needs *two* client implementations
(RFC 6962 + static-ct-api) and must drive both off the live log list JSON rather than a hardcoded list.
Budget accordingly, or defer CT-direct out of v1.

**Terms of service — this is the good news.** The Chrome CT Log Policy requires log operators to
*"Not impose conditions on retrieving or sharing data from the logs"*, to *"Maintain log availability of 99%
or above"* (90-day rolling average, measured as the minimum across endpoints), and states that
*"Log data availability is of paramount importance. All rate limits must be set to ensure that all
well-behaved clients can reliably retrieve log entries at a rate greater than the growth rate of the log."*
<https://googlechrome.github.io/CertificateTransparency/log_policy.html>

**There is no ToS obstacle to reading CT logs directly.** This is the only major source in this document
that is contractually clean by design.

**Misses:** certificates never logged to CT (internal CAs, self-signed, pre-2018 legacy certs); hosts that
have never had a public certificate; wildcard certs hide the specific hostnames behind `*.example.com`.

### 2.2 crt.sh

**What it is.** A free web + JSON interface over an indexed copy of the CT logs, operated by Sectigo
Limited (© footer on <https://crt.sh/>). It is the only keyless service that answers the question
verge-asm actually asks: *"give me every certificate whose identity matches `%.example.com`"*.

**Query patterns [measured, all HTTP 200]:**

- `https://crt.sh/?q=%25.example.com&output=json` — wildcard identity match, includes subdomains.
- `https://crt.sh/?identity=example.com&output=json` — exact identity match.
- `https://crt.sh/?q=example.com&output=json` — general match.
- Also accepts `&exclude=expired`, `&deduplicate=Y`, and CA/serial/fingerprint lookups.

Returned JSON fields (observed): `issuer_ca_id`, `issuer_name`, `common_name`, `name_value` (newline-
separated SAN list — the field you actually want), `id`, `entry_timestamp`, `not_before`, `not_after`,
`serial_number`, `result_count`.

**Rate limits — primary source, from the operator.** In the public `crtsh` Google Group, Rob Stradling
(Sectigo) announced in Jan 2020 that *"Requests to the crt.sh web service are now being throttled at
60 requests per IP address per minute."*
<https://groups.google.com/g/crtsh/c/NZJntKrBdmg>
That was **superseded**. On 2023-06-09 a Sectigo address replied to a bulk-querying question stating the
current limits are **5 requests per minute per IP on crt.sh:443** and **5 concurrent connections per IP on
crt.sh:5432** (the public PostgreSQL interface), explaining that *"both crt.sh:443 and crt.sh:5432 are
heavily (ab)used"* with frequent overload causing *"slow responses and HTTP 502 errors"*, and warning that
limits may be reduced further. <https://groups.google.com/g/crtsh/c/QXQFoy331pE>

There is a separate documented cap of **999 results** on `output=json` responses.
<https://github.com/crtsh/certwatch_db/issues/70>

**Reliability — verified, the reputation is earned.** **[measured]** Consecutive requests for
`?q=%25.iana.org&output=json`:

- attempt 1: **HTTP 503 after 16.9 s**
- attempt 2: HTTP 200 after **16.3 s**
- attempt 3 (`identity=`): HTTP 200 after 2.6 s
- attempt 4: HTTP 200 after 2.3 s

A fuller timed sample series is in **§7**, and it is worse than the reputation suggests: **4 of 8 identical
requests failed (2× 502, 2× 404), and successful responses took 11.9 s – 59.6 s** — all while staying under
the documented 5 req/min limit. Critically, crt.sh returns **spurious 404s for domains that demonstrably
have certificates**, which a naive client will misread as "no assets found". See §7 for the resulting
hard requirements on the client implementation.

**Terms of service.** **[measured]** crt.sh publishes no Terms of Use page; the homepage carries only a
Sectigo copyright notice, and neither `/` nor the JSON endpoint returns terms. The only stated constraint
anywhere is the operator's rate-limit announcement above. **Assessment:** there is no ToS *prohibiting*
programmatic use — crt.sh is explicitly an API — but the operator has publicly asked for restraint and has
cut limits twice due to abuse. Shipping crt.sh as a default is defensible **only** with a hard client-side
throttle at or below 5 req/min, a distinctive `User-Agent` identifying verge-asm and its version, and
per-domain result caching. Shipping it with a naive retry loop would make verge-asm an abuse vector at scale
and is the single most likely way this project ends up rate-limited into uselessness for everyone.

**Misses:** the 999-result cap truncates large estates; wildcard certs; unlogged certificates; and crt.sh's
index lag behind the logs.

### 2.3 SSLMate Cert Spotter CT Search API

**Endpoint:** `GET https://api.certspotter.com/v1/issuances?domain=DOMAIN&include_subdomains=true&expand=dns_names&expand=issuer`
<https://sslmate.com/help/reference/ct_search_api_v1>

**[measured]** Works with no API key, returns HTTP 200 in ~1.3 s with fields `id`, `tbs_sha256`,
`cert_sha256`, `dns_names[]`, `pubkey_sha256`, `issuer{}`, `not_before`, `not_after`, `revoked`.
Pagination is cursor-based via `after=<last id>`. It is faster and far more reliable than crt.sh.

**Rate limits.** Documented free-tier limits are **100 single-hostname queries/hour** and **10 full-domain
queries/hour** (`include_subdomains=true`). <https://sslmate.com/pricing/ct_search_api>
**[measured]** confirmed exactly: 9 consecutive `include_subdomains=true` requests returned 200, the 10th
onward returned **HTTP 429** with `Retry-After: 306` and body
`{"code":"rate_limited","message":"You have exceeded the domain search rate limit for the SSLMate CT Search API. …"}`.

**Terms of service — this is why it is not a default.** The API reference states: *"You can make a limited
number of unauthenticated 'List Issuances for a Domain' queries per hour, **for personal or evaluation
purposes**."* <https://sslmate.com/help/reference/ct_search_api_v1> The docs further note SSLMate *"may
discontinue, alter, or restrict functionality for free tier users at any time without notice."*

A self-hosted ASM tool run by an organisation to monitor its own production estate is **not** "personal or
evaluation" use. Shipping unauthenticated Cert Spotter as a default would put every verge-asm operator in
technical breach of SSLMate's terms without their knowledge. **Ship it as an opt-in integration with an
operator-supplied API key**, and say so in the UI.

---

## 3. DNS

### 3.1 Which record types to collect

Resolution should run through the **operator's own recursive resolver** (or a resolver they configure).
This is the cleanest possible source: no third party, no terms to accept, no shared quota, and it reflects
what the operator's own network actually sees.

> **RULED, and *should* is now *does*** — [#116](https://github.com/winniel123/verge-asm/issues/116) /
> [ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md).
> The **query path** is a declared parameter shared by `resolution-walk` and
> `wildcard-discrimination`, taking **one value per `Batch`**, and its value is **the `Vantage`'s
> configured recursive resolver**. Three riders this recommendation did not carry. **`Resolved`,
> `NoData` and `NameError` are read on that path** — `resolution-walk` also walks the delegation, and
> nothing here said which of the two answers it holds becomes the value. **The delegation walk is not
> governed by the parameter**: it decides `Lame` and the per-nameserver `serves │ does-not-serve`
> RRset direct to the delegated authorities, as `Lame`'s own definition requires, and it **supplies no
> address set**. And **which resolver stands at that path is part of the `Vantage`**, not of the
> parameter — a declared parameter may never be operator-configurable, while §3.6's resolver choice
> plainly is one. See §14.

| Type | What it yields for ASM |
|---|---|
| **A / AAAA** | The exposure itself. Every discovered name must resolve to something before it is an asset. AAAA is routinely forgotten by operators — v6-only exposure is a real finding. |
| **CNAME** | Third-party hosting and the **dangling-CNAME / subdomain-takeover** signal: a CNAME to a deprovisioned SaaS/cloud target is the highest-value free-fall risk finding available from passive data alone. Requires resolving the target and classifying NXDOMAIN/unclaimed-bucket responses. |
| **MX** | Mail provider identification; MX hosts are themselves exposed assets. |
| **NS** | Delegation map; reveals sub-zones the operator forgot, and **lame delegation** (NS pointing at a nameserver that no longer serves the zone) is a takeover primitive. NS also enumerates AXFR candidates (§3.3). |
| **TXT** | SPF (`v=spf1`, RFC 7208 <https://www.rfc-editor.org/rfc/rfc7208.html>) — the `include:`, `a:`, `mx:`, and `ip4:`/`ip6:` mechanisms enumerate *every third party authorised to send as the org*, which is a free, accurate list of vendor relationships and often of owned IP space. DMARC at `_dmarc.<domain>` (RFC 7489 <https://www.rfc-editor.org/rfc/rfc7489.html>) yields policy posture and `rua`/`ruf` reporting addresses. Other TXT records carry domain-verification tokens (`google-site-verification`, `MS=`, etc.) that fingerprint which SaaS platforms the org uses. |
| **SOA** | Zone boundary detection — tells you whether a name is its own zone (and therefore has its own delegation and its own AXFR surface). |
| **CAA** | Which CAs are authorised; a mismatch between CAA and observed CT issuance is a drift signal. |
| **PTR** | Only meaningful inside ranges the org owns (§4). Reverse-walking an owned /24 is cheap and finds hosts that no forward source knows about. |

**Do not use ANY queries.** RFC 8482 makes it explicit that *"returning a subset of available RRsets when
processing an ANY query is legitimate"* and that servers *"MAY decline to provide a conventional ANY
response or MAY instead send a response with a single RRset"*. Google Public DNS repeats the warning:
`type=255` *"is not a replacement for sending queries for both A and AAAA or MX records."*
<https://www.rfc-editor.org/rfc/rfc8482.html>,
<https://developers.google.com/speed/public-dns/docs/doh/json>
verge-asm must issue explicit per-type queries.

### 3.2 Wildcard detection — mandatory, not optional

Per RFC 4592, a wildcard `*.example.com` causes the server to *"create RRs with an owner name equal to the
query name and contents taken from the wildcard RRs"* whenever the query name "falls off the tree" — i.e.
no closer match exists. Synthesis is blocked if the query name is known to exist (including as an empty
non-terminal), or if a name between the wildcard and the query name exists.
<https://www.rfc-editor.org/rfc/rfc4592.html>

Consequence: on a wildcarded zone, **every** guessed or third-party-sourced name resolves, and any pipeline
that treats "resolves" as "exists" reports an unbounded, entirely fictional asset inventory. Detection is
cheap and must run before any name-based expansion:

> **Steps 1 and 3 below are WITHDRAWN by [#108](https://github.com/winniel123/verge-asm/issues/108) /
> [ADR-0066](../adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md),
> which rules the population.** *Under the apex* and *repeat one level down for each discovered
> sub-zone* are both replaced: **a name is discriminated at its parent.** The control-probe
> population of a `Batch` is the set of **immediate parents of the `Name`s in that batch's resolution
> scope**, deduplicated and intersected with the `Seed` name scopes they sit inside — which needs no
> depth rule, because a control label constructed under P falls off the tree at the same closest
> encloser the names under P do, whatever depth the wildcard sits at and **whether or not P itself
> exists**. There is no in-batch sub-zone discovery loop: the population is computable at batch
> start. The **A, AAAA and CNAME** qtype clause in step 1 is withdrawn by the same ruling — the
> control probe runs the batch's **declared qtype set**, all seven
> ([`measurement-offers.md`](../spec/measurement-offers.md) §2), because `Shadowed` is committed on
> `dns-record` for *any* qtype and three qtypes leave MX, TXT, NS and SOA synthesised answers
> recorded as the name's own records. The population is the **seventh aperture input**, recorded on
> the `Batch` by content, and never a declared parameter of `wildcard-discrimination`.

1. ~~Query 3–5 long random labels under the apex (e.g. `<random32>.example.com`) for A, AAAA, and CNAME.~~
   ~~Query 3–5 long random labels under **each name in the control-probe population** (e.g.
   `<random32>.dev.example.com`) for the declared qtype set.~~
   **AMENDED by [#113](https://github.com/winniel123/verge-asm/issues/113) /
   [ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md):
   a random label alone cannot falsify label-independence, and `3–5` is a range where a declared
   parameter needs a value.** ~~Query **six** control labels~~ **RAISED to **ten** by
   [#115](https://github.com/winniel123/verge-asm/issues/115) — see the bullet below.** Query
   **ten** control labels under **each name in the control-probe
   population**, each **exactly one label** and each run over the declared qtype set:

   - ~~**5 long random labels** (e.g. `<random32>.dev.example.com`) — the count is now **5**, a value
     rather than a range;~~
     **RAISED by [#115](https://github.com/winniel123/verge-asm/issues/115) / §13: **9 long random
     labels**.** The count is still a **value** and not a range; the value is now **9**. Read alone
     and in the present tense, *the count is now 5* builds a set **[measured]** to misread a
     two-member per-label pool **2 times in 30** — `surge.sh` splits its labels `188/172` over a
     near-fair binary hash, and six draws miss the second member 3.2% of the time against 0.21% at
     ten. The mechanism the raise is bought against is **per-label sharding**, isolated in §13;
   - **1 structured label** of the form `<a>-<b>-<c>-<d>`, the four octets of an address drawn at
     random from **RFC 5737** documentation space (`192.0.2.0/24`, `198.51.100.0/24`,
     `203.0.113.0/24`) — e.g. `203-0-113-7.dev.example.com`. Its job is to be **decodable**, so an
     authority that computes its answer from the query name answers it differently from a random
     label and is caught by step 2's `Indeterminate`.

   **Exactly one label is load-bearing and is not a style choice.** ADR-0066 admits the control
   probe because a label constructed under P falls off the tree at the same closest encloser the
   names under P do, and every candidate is one label under its parent. A multi-label control label
   (a *dotted* quad, `10.0.0.1.example.com`) has ancestors between it and P, and where one exists it
   falls off at a **deeper** encloser — measuring somebody else's wildcard and filing it as P's.
   **[measured]** 2026-08-15 that manufactures a wildcard on **3 of 5** un-wildcarded zones. See
   §12.

   **And the ten labels are asked on the batch's declared query path**
   ([#116](https://github.com/winniel123/verge-asm/issues/116) /
   [ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md))
   — **the same path the candidate's own answer is read on**, which is the `Vantage`'s configured
   recursive resolver, and there is **no second path** exactly as there is no second qtype list. A
   control probe asked somewhere the candidate was not is not a measurement of the candidate's
   synthesis: **[measured]** direct to `s3.amazonaws.com`'s own authority the A component is
   `NoSynthesis` — a *determinate* reading — while a resolver answers every name beneath it with
   eight addresses, so a skewed pair discriminates **every** fictional label and records it
   `Resolved`. See §14.
2. ~~If they answer, record the wildcard answer set as a **poison signature**.~~
   **WITHDRAWN by [#111](https://github.com/winniel123/verge-asm/issues/111) /
   [ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md):
   there is no *the* answer set.** If they answer, record the signature **per component** — a
   component being one `(qtype asked, RR type in the answer)` pair — as one of three:
   **`NoSynthesis`** (no control label carried an RR of that type), **`Determinate(RRset)`** (all
   *n* carried the identical RRset), **`Indeterminate`** (they disagreed). **Never a union of the
   observed sets**, which is the object an intersection predicate needs and which is therefore
   deliberately not recorded. See §11.
3. ~~Repeat one level down for each discovered sub-zone — wildcards can exist at any label depth.~~
   No repetition and no depth walk: the parent population already covers every held name at every
   depth.
4. ~~Suppress (or flag as unverifiable) any candidate whose answer matches the poison signature.~~
   ~~**The match predicate is unspecified and is not set equality**~~ — **RULED by
   [#111](https://github.com/winniel123/verge-asm/issues/111) /
   [ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md).**
   The predicate is **set equality on the RDATA set, per component, at determinate components
   only**. An `Indeterminate` component is **never consulted**: it can neither shadow a name nor
   exempt one. **Suppression is the default** beneath a measured wildcard — a candidate is
   `Shadowed` unless it **differs at some determinate component**, so the predicate looks for the
   exemption and never for the match. See §11.
5. ~~Note the RFC 4592 escape hatch: a name that resolves to something *different* from the wildcard
   signature is genuinely present even under a wildcard, because an exact match blocks synthesis.~~
   **WITHDRAWN AS WRITTEN by the same ruling.** The escape hatch is sound only at a **determinate**
   component: where the synthesis varied, *different from the signature* is a second draw from the
   same process rather than evidence of anything. It is unsound even at a determinate component
   against a synthesis that is a **function of the label** — **[measured]** 2026-08-14,
   `traefik.me` answers `127.0.0.1` for every random control label while `10.0.0.1.traefik.me`
   answers `10.0.0.1`. What survives is the RFC 4592 §2.2.1 half: a candidate discriminated at
   **any** component **exists**, and therefore **none** of its RRsets is synthesised — including
   ones that coincide with the signature. `Shadowed` is all-or-nothing across a name's qtypes.
6. Where the control probe under a name's parent **did not complete**, that name records a **`Gap`**
   and never a value — *an undiscriminated answer is never a value* (ADR-0066). A probe that
   completed and found no wildcard licenses everything beneath it — ~~**with one measured residue,
   which is the control label's construction rather than this rule: [measured]** `nip.io` and
   `sslip.io` answer **NODATA** to a random label while `10.0.0.1.nip.io` answers `10.0.0.1`. See
   §11.7.~~ **and *found no wildcard* is QUALIFIED by
   [#113](https://github.com/winniel123/verge-asm/issues/113) /
   [ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md):
   the licence is issued only where **no control label of any shape carried an RR at any qtype**.
   Under step 1's heterogeneous set that is a strictly stronger test than it was, and it is the
   whole fix for the residue this bullet used to carry — **[measured]** `nip.io`'s and `sslip.io`'s
   random labels answer NODATA while `203-0-113-7.nip.io` answers `203.0.113.7`, so one label
   carries an RR, the set disagrees, the component is `Indeterminate` and **the licence is
   withheld**. Read alone and in the present tense the unqualified sentence still licenses a
   fictional inventory there, which is why it is amended here rather than pointed at from §12.

**[measured]** 2026-08-14, against live authorities over Google Public DNS, and it is what decides
the population. `render.com`'s apex control label is **NXDOMAIN** while `*.staging.render.com`
answers `10.4.3.1` for anything beneath it, at every depth — so apex-only probing reports *no
wildcard* over a sub-tree that is entirely synthesised. On the `%.iana.org` estate §7 already
sampled, crt.sh returns **25 distinct non-wildcard names** whose immediate parents are **six** names
inside the seed, and **exactly one of the six — the apex — is a zone cut**: `itar.`, `rzm-epp.` and
`ns.` sit inside `iana.org` and `int.`/`rzm.` CNAME into `vip.icann.org`. So *the apex plus each
measured zone cut* probes the same single site as the apex alone and leaves **11 of those 25 names
(44%) undiscriminated**, while the parent set covers **25 of 25** from **6** sites — against 25 for
*every `Name`*, of which 19 are leaves whose probes discriminate nothing the estate holds.
`ns.iana.org` is itself **NXDOMAIN** and in no `Citation`, and probing under it is still correct:
`www.ns.iana.org` is a held name whose closest encloser is `iana.org`.

### 3.3 AXFR and operator-supplied zone data — the defensive advantage

RFC 5936: AXFR is TCP-only, returns the entire zone bracketed by its SOA, and access is policy-controlled —
*"A general-purpose implementation SHOULD NOT have a default policy for AXFR requests to be 'open to all'"*,
with TSIG/SIG(0) recommended for access control.
<https://www.rfc-editor.org/rfc/rfc5936.html>

For an **offensive** tool, AXFR is a lottery ticket you almost never win. For a **defensive** tool it is the
opposite: verge-asm's operator *owns the zone*. They can authorise an AXFR from the verge-asm host, or
simply upload the zone file, or export it from their DNS provider. That single input supersedes CT, passive
DNS, brute force, and web archives combined — it is complete, authoritative, and instant.

**Spec implication:** verge-asm should make "give me your zone" a **first-class, prominently-surfaced
onboarding step**, supporting (a) zone file upload / on-disk path, (b) AXFR with optional TSIG key, and
(c) a re-check that flags names present in the zone but not reachable, and names reachable but not in the
zone (shadow records, third-party-managed sub-delegations). Everything else in this document exists to
cover the cases where the operator *cannot* supply the zone — forgotten domains, shadow IT registered on a
personal card, sub-zones delegated to a vendor, and acquisitions.

AXFR against nameservers the operator does *not* control must be **off by default and gated behind an
explicit scope confirmation** — it is an unauthorised access attempt against a third party.

### 3.4 DNSSEC zone walking (NSEC / NSEC3)

RFC 5155 states the problem plainly: with NSEC, *"the complete set of NSEC records lists all the names in a
zone"*, enabling full enumeration. NSEC3 replaces owner names with salted, iterated hashes, but the RFC
concedes *"NSEC3 RRs are still susceptible to dictionary attacks … the attacker retrieves all the NSEC3 RRs,
then calculates the hashes of all likely domain names."*
<https://www.rfc-editor.org/rfc/rfc5155.html>

For verge-asm: **NSEC walking of the operator's own signed zones is free, complete, and legitimate** — it is
the operator's own data, and it costs a handful of queries rather than a brute-force wordlist. It should be
attempted automatically whenever the apex has DNSSEC and returns NSEC (not NSEC3). NSEC3 cracking is
explicitly out of scope: it is expensive, incomplete, and (see §3.5) redundant against a zone file.

### 3.5 Does subdomain brute-forcing earn its place? — **No, not as a default**

The case against, for this product specifically:

1. **It is active, not passive.** It sends traffic that lands in the target's authoritative-DNS logs, and it
   hammers whatever recursive resolver it is pointed at. A 100k-word list against 10 domains is a million
   queries. Pointing that at a public resolver (§3.6) is abuse; pointing it at the operator's own resolver
   is a self-inflicted incident.
2. **It is redundant against the primary input.** Anything brute force could find in a zone the operator
   controls is already in the zone file / AXFR / NSEC walk — completely, instantly, and without a single
   wasted query.
3. **It is wildcard-fragile.** On a wildcarded zone (§3.2) it produces pure noise, and wildcards are common.
4. **Its recall is bad where it matters.** The assets a defensive ASM tool exists to find are the *forgotten*
   ones — `jenkins-old-2019`, `acme-corp-uat-eu-west-2` — which are precisely the names a generic wordlist
   does not contain. CT, by contrast, finds them exactly because someone issued them a certificate.
5. **It produces the wrong risk profile for a small-org security owner.** The persona has no appetite for
   "your tool generated a million DNS queries and our provider throttled us."

The case for keeping it available: it covers names that (a) never had a public certificate, (b) are not in
the zone the operator supplied (shadow IT under a domain they forgot they own), and (c) are absent from all
passive indices. That is a real but narrow gap.

**Recommendation:** ship brute-forcing as an **explicitly opt-in, rate-limited, small-wordlist** mode with a
mandatory wildcard check, a visible query-count estimate before it runs, and a required resolver choice.
Never on by default. Never with a 100k list. Frame it in the UI as an active technique, alongside probing,
not as passive discovery.

### 3.6 Public DoH JSON resolvers

Google Public DNS (`https://dns.google/resolve?name=&type=&cd=&do=`)
<https://developers.google.com/speed/public-dns/docs/doh/json> and Cloudflare
(`https://cloudflare-dns.com/dns-query` with `Accept: application/dns-json`)
<https://developers.cloudflare.com/1.1.1.1/encryption/dns-over-https/make-api-requests/dns-json/>
both offer keyless JSON DNS. Neither publishes a numeric rate limit in its API documentation.
Cloudflare notes the JSON format *"does not have a formal RFC, which means behavior might be different
between providers"* and recommends wireformat DoH for critical applications.

**Recommendation:** these are a **fallback**, not the default. verge-asm should resolve via the host's own
resolver by default (zero third parties, zero terms, no shared quota), and offer DoH only for operators
whose environment blocks outbound 53 or who distrust their local resolver. Using a public resolver as the
default for bulk ASM resolution would mean shipping software that free-rides on Google's and Cloudflare's
infrastructure at every install.

> **Where that choice lives** — [#116](https://github.com/winniel123/verge-asm/issues/116) / §14. It
> is a **`Vantage` declaration**, never a declared parameter: `CONTEXT.md`'s `Derivation` says *none
> is ever operator-configurable*, and this is an operator choice. So an operator moving to DoH
> declares a different `Vantage`, and because `vantage` is in the timeline key that **opens**
> timelines (`revealed`) rather than `Break`ing the estate. What the parameter fixes is the **kind**
> of path — recursive rather than direct-to-authority — which is authored data and ships in the
> release. **[measured]** the distinction is not academic: `vercel.com`'s wildcard draws from
> `64.239.109.0/24` and `64.239.123.0/24` at one vantage against the `76.76.21.x` family §11
> recorded at another, two disjoint pools for one name.

---

## 4. RDAP / WHOIS and ASN-to-IP-range mapping

### 4.1 Bootstrap mechanics (this is the part that makes RDAP work keylessly)

RDAP is the IETF replacement for WHOIS. Lookups are defined in RFC 9082 as path segments: `ip/`, `autnum/`,
`domain/`, `nameserver/`, `entity/`, `help/`; searches as `domains?name=`, `nameservers?name=`,
`entities?fn=`, `entities?handle=`. Search support is **optional** — *"Server implementations are free to
support only a subset of these features … Servers MUST return an HTTP 501 (Not Implemented) response to
inform clients of unsupported query types"*, and HTTP 422 when a particular partial-match style is
unsupported. <https://www.rfc-editor.org/rfc/rfc9082.html>

Response shapes are RFC 9083: `objectClassName` on every object; IP networks carry `startAddress`,
`endAddress`, `cidr0_cidrs`, `name`, `type`, `country`, `parentHandle`, `status`; autnums carry
`startAutnum`/`endAutnum`; entities carry `handle`, `roles[]`, `vcardArray` (jCard); `links[]` with
`rel: "self"`/`"up"` enable navigation; `notices[]` is where servers attach a `"Terms of Service"` title.
<https://www.rfc-editor.org/rfc/rfc9083.html>

**Bootstrap** removes the need for any hardcoded registry map. IANA publishes machine-readable bootstrap
files — **[measured]**, all HTTP 200:

| File | Purpose | Publication seen |
|---|---|---|
| `https://data.iana.org/rdap/dns.json` | TLD → RDAP base URL | 2026-07-23 |
| `https://data.iana.org/rdap/ipv4.json` | IPv4 block → RIR RDAP base URL | 2019-06-07 |
| `https://data.iana.org/rdap/ipv6.json` | IPv6 block → RIR RDAP base URL | — |
| `https://data.iana.org/rdap/asn.json` | ASN range → RIR RDAP base URL | 2026-06-01 |

Format is `{"description", "publication", "services": [[[keys…],[urls…]], …]}`. A client caches these,
matches the query resource against the key list, and dispatches to the URL. That is the whole mechanism —
**no key, no account, no registration, and it is the IETF-blessed way to do it.**
Registry index: <https://www.iana.org/assignments/rdap-dns/rdap-dns.xhtml>

### 4.2 RIR RDAP: org name / single IP → owned ranges

**[measured]** All five RIRs respond keylessly:

| Query | Result |
|---|---|
| `https://rdap.arin.net/registry/ip/8.8.8.8` | 200 in 0.81 s — `cidr0_cidrs: [{v4prefix 8.8.8.0, length 24}]`, `entities[]` with abuse/tech contacts |
| `https://rdap.db.ripe.net/ip/193.0.6.139` | 200 in 1.20 s — `handle`, `startAddress`/`endAddress`, `name: "RIPE-NCC"`, `type: "ASSIGNED PA"`, `country`, `parentHandle` |
| `https://rdap.lacnic.net/rdap/ip/200.3.13.10` | 200 |
| `https://rdap.afrinic.net/rdap/ip/196.216.2.1` | 200 |
| `https://rdap.arin.net/registry/autnum/13335` | 200 — AS object with entities |

**The org-name → ranges path works on ARIN, and it is the single most valuable RDAP capability for ASM.**
**[measured]** `https://rdap.arin.net/registry/entities?fn=Cloudflare*` returns an `entitySearchResults`
array of org handles; fetching `https://rdap.arin.net/registry/entity/CLOUD14` then returns a `networks`
array containing **202 CIDR prefixes** — the org's complete ARIN-registered IPv4/IPv6 footprint, from a
free-text company name, with two keyless HTTP requests.

Coverage is uneven, and the spec must not assume otherwise. **[measured]**
`https://rdap.db.ripe.net/entities?fn=RIPE*` returned **HTTP 500**; `https://rdap.apnic.net/entities?fn=Google*`
returned 200. Entity search is optional per RFC 9082 and RIPE's implementation is not usable. For RIPE the
working keyless substitute is the **RIPE Database REST search API** — **[measured]**
`https://rest.db.ripe.net/search?query-string=cloudflare&type-filter=inetnum&flags=no-filtering` with
`Accept: application/json` returned 200 and 122 KB of matching `inetnum` objects.

**Rate limits.** **[measured]** 25 rapid consecutive ARIN RDAP `ip/` lookups all returned 200 — no
throttling observed, and ARIN's Whois-RWS API documentation states no numeric limit
(<https://www.arin.net/resources/registry/whois/rws/api/>). RDAP.org's public bootstrap proxy documents
10 requests / 10 seconds with 429 (<https://about.rdap.org/>) — a reasonable client-side default to adopt
regardless.

**Terms of service — ARIN.** The ARIN Whois Terms of Use (effective 2014-04-09) explicitly *permit*
*"Internet operational or technical research purposes"*, *"evaluating routing policies or assuring
compliance with routing policies"*, *"facilitating operational coordination between network operators"*, and
*"ensuring the uniqueness of Internet number resource usage"*, plus abuse research and tracking.
Prohibited: use *"as part of a commercial service or product, including the solicitation and servicing of
your, or your employer's, customers"*, and *"for advertising, direct marketing, marketing research or
similar purposes"*. Redistribution requires imposing *"terms that are at least as restrictive"*.
<https://www.arin.net/resources/registry/whois/tou/>

**Assessment:** an organisation running verge-asm to inventory **its own** attack surface is squarely inside
"Internet operational or technical research" and "assuring compliance with routing policies", and is not
soliciting or servicing customers. This is a **default-safe** use. The clause that *would* bite is someone
building a paid ASM SaaS on top of verge-asm — that is a commercial service, and the AGPL does not exempt
them from ARIN's terms. Document this distinction in the README rather than blocking the feature.

**Terms of service — RIPE.** The RIPE Database Acceptable Use Policy limits **personal data sets returned**
to 1,000 per 24 h per IP (20,000 for a proxy IP, 1,000 per RIPE NCC Access Account), with temporary blocking
on breach escalating to permanent blocking for repeat offenders, and prohibits creating *"a copy of a
significant part of the RIPE Database … without the consent of the RIPE NCC"*.
<https://www.ripe.net/manage-ips-and-asns/db/support/documentation/ripe-database-acceptable-use-policy/>
The limit counts *personal data* specifically, so network-object queries are much less constrained, but a
1,000/day ceiling is the number to design against. Well within reach for a single org's estate.

### 4.3 gTLD domain RDAP (registration data)

As of **28 January 2025**, RDAP is the required RDDS mechanism for gTLDs and the WHOIS obligation has
sunset (except .com/.name/.post); ICANN's Registration Data Policy took effect 21 August 2025.
<https://www.icann.org/en/announcements/details/icann-update-launching-rdap-sunsetting-whois-27-01-2025-en>,
<https://www.icann.org/en/contracted-parties/registry-operators/resources/registration-data-access-protocol>

**[measured]** `https://rdap.verisign.com/com/v1/domain/example.com` → 200, returning `handle`, `ldhName`,
`status[]` (client*Prohibited flags), `events[]` (registration/expiration/last-changed dates), `nameservers`,
and an `entities[]` array containing **only the registrar** plus an abuse contact with empty `fn`/`tel`/
`email` values. Registrant identity is redacted.

**Verdict for ASM:** gTLD RDAP is useful for **expiry monitoring** (a domain about to lapse is a genuine
free-fall risk — expired domain → takeover of everything CNAME'd to it), **nameserver drift**, and
**registrar-lock status** (`clientTransferProhibited` absent is a finding). It is **not** useful for
"find other domains this org owns" — post-GDPR redaction killed reverse-WHOIS from free sources.
`https://rdap.verisign.com/com/v1/domain/iana.org` correctly 404s (wrong TLD server) — the bootstrap file
must be consulted, not guessed.

### 4.4 Team Cymru IP→ASN mapping

Free, keyless, three interfaces: WHOIS (`v4.whois.cymru.com`, `v4-peer.whois.cymru.com` on port 43), DNS
(`origin.asn.cymru.com`, `origin6.asn.cymru.com`, `peer.asn.cymru.com`, `asn.cymru.com`), and a bulk netcat
mode using `begin`/`end` markers. Data is refreshed every ~4 hours from 50+ BGP peers and yields BGP origin
ASN, peer ASN, BGP prefix, prefix country code, and AS description.
<https://team-cymru.com/community-services/ip-asn-mapping/>

**Terms / operational constraint — explicit and severe.** The service states:
*"IPs that are seen abusing the whois server with large numbers of individual queries instead of using the
bulk netcat interface will be null routed."* It also warns *"IP to ASN Mapping is Not a GeoIP Service"* —
the country code is registry-derived, not geolocation, and must not be presented as asset location.

**Recommendation:** genuinely free and unrestricted in licence terms, but the null-routing warning makes a
naive per-IP loop a self-destruct button. Use only via the bulk interface, batching all IPs in one session
and aggregating to /24 (v4) and /64 (v6) as the service recommends. Given RIPEstat's `network-info` already
answers IP→ASN over plain HTTPS with no such warning, **Team Cymru is a Tier-2 alternate**, not the default.

### 4.5 RIPEstat Data API

**[measured]** All keyless, all HTTP 200:

| Call | Result |
|---|---|
| `https://stat.ripe.net/data/network-info/data.json?resource=8.8.8.8` | `{"asns":["15169"],"prefix":"8.8.8.0/24"}` |
| `https://stat.ripe.net/data/announced-prefixes/data.json?resource=AS13335` | 1.93 s, **5,310 prefixes**, with the note *"Results exclude routes with very low visibility (less than 10 RIS full-feed peers seeing)"* |
| `https://stat.ripe.net/data/searchcomplete/data.json?resource=cloudflare` | org name → `AS13335 "CLOUDFLARENET - Cloudflare, Inc."`, AS14789, AS132892, … |

~~That chain — **org name → ASNs → announced prefixes → per-IP reverse lookup** — is the complete
"from a company name to the IP ranges it actually announces" path, keyless, in three requests. It covers all
five RIRs and, crucially, reflects **BGP reality** rather than registry paperwork, so it catches space the
org announces but has not tidily registered.~~

> **Withdrawn 2026-08-15 by [#126](https://github.com/winniel123/verge-asm/issues/126) /
> [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md), at the site that
> specifies it.** This is the sentence the BGP leg has been carried on since 2026-07-30, restated twice
> since, and it contains the defect: *"the IP ranges it **actually announces**"* is offered as a better
> answer to *which addresses are the operator's*, and it is an answer to a different question. **A routing
> announcement names who carries packets toward a prefix, never who controls what listens in it** — so
> *space the org announces but has not tidily registered* is, measured, either a more-specific of space
> already proposed, or a subsidiary's registration the org-name box reaches, or **a customer's network the
> org merely carries**. The full decomposition and the two measurements are at
> [`non-arin-prefix-coverage.md`](./non-arin-prefix-coverage.md) §14 (**[measured] 0 of 100** and **2 of
> 11**, the second on a hosting company). The `announced-prefixes` call is in no shipped path.

**Limits.** The docs state no request cap but *"the system limits the usage to 8 concurrent requests coming
from one IP address"*, and ask that you *"please register if you plan to regularly do more than 1000
requests/day"* (registration is an email to stat@ripe.net, not a key).
<https://stat.ripe.net/docs/02.data-api/>

**Terms of service — read this before making it a default.** The RIPEstat Service Terms and Conditions
scope permitted use to *"network analysis, network monitoring and debugging and research"*, require
attribution when publishing results, state that a user *"may not re-package, compile, re-distribute the
RIPEstat Service and any (or all) of the RIPEstat Data"* without written permission, prohibit commercial
purposes including *"providing paid services, products or any other derivatives"* without written
permission, and reserve the right to *"restrict any use of the RIPEstat Service"*.
<https://www.ripe.net/about-us/legal/ripestat-service-terms-and-conditions/>

**Assessment:** "network analysis, network monitoring" is exactly what a self-hosted ASM tool does for its
own operator, and each verge-asm install queries RIPEstat directly rather than verge-asm redistributing
RIPEstat data — so the re-packaging clause is not triggered by the software itself. ~~**Default-on is
defensible for the self-hosted, non-commercial case**, with a low concurrency cap (≤8, per the docs), a
per-run request budget aimed at staying under 1000/day, and an identifying User-Agent.~~ **But:** anyone
operating verge-asm as a paid service needs RIPE NCC's written permission, and the README must say so.
~~This is the one Tier-1 default with a genuine ToS asterisk — if the project wants zero contractual
ambiguity in defaults, demote RIPEstat to Tier 2 and accept weaker RIR coverage out of the box.~~

> **Both struck sentences are withdrawn at the site that specifies them
> ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)), by two
> different decisions.** *Default-on is defensible* and the Tier-2 fork were settled by
> [#15](https://github.com/winniel123/verge-asm/issues/15) /
> [ADR-0003](../adr/0003-third-party-source-consent-bar.md) and
> [#19](https://github.com/winniel123/verge-asm/issues/19) — RIPEstat ships **off**, indefinitely, under
> `operator-accepted`, and the reading is the operator's rather than the project's. The
> `announced-prefixes` call inside the same section is separately out under
> [#126](https://github.com/winniel123/verge-asm/issues/126) /
> [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md). The **measurements**
> and the **verbatim terms quotes** in this section stand untouched and are still the record for
> RIPEstat's clauses.

### 4.6 bgp.tools — **not usable as a default**

**[measured]** `https://bgp.tools/kb/api` with an identifying User-Agent returned **HTTP 307 → a login
page** stating *"Due to persistent issues from the IP or network you are accessing from, you will need to
log in to use bgp.tools."* Their scripting policy states that systems using default or generic user agents
*"may be blocked"* and that users who scrape HTML *"may be banned with no notice"*, because
*"bgp.tools's HTML output is a site for humans and their respective usage patterns, not robots."*
<https://bgp.tools/kb/api>

Excellent data, but it requires an account for a meaningful fraction of client IPs and explicitly polices
automated use. **Cannot ship as a keyless default.** Optional operator-configured integration at most.

---

## 5. Passive DNS

Short version: **there is essentially one genuinely free, genuinely keyless, genuinely usable passive DNS
source, and it cannot do subdomain enumeration.** Passive DNS is where the keyless constraint bites hardest.

### 5.1 mnemonic Passive DNS — the only keyless option

**[measured]** `https://api.mnemonic.no/pdns/v3/iana.org` returns HTTP 200 with no authentication:
`{"query","answer","rrtype","rrclass","minTtl","maxTtl","times","tlp","firstSeenTimestamp","lastSeenTimestamp"}`.
Also supports `?rrType=a&rrType=mx`, `?limit=0`, a POST JSON form at `/pdns/v3/search`, a `/seen` existence
endpoint, and PassiveDNS Common Output Format at `/pdns/v3/cof/`.

**Limits, from mnemonic's own documentation:** *"Unauthenticated users are limited to 10 requests per
minute, and 1000 requests per day."* Exceeding returns **HTTP 402**; the per-minute limit rejects briefly,
the per-day limit *"will typically be rejected for a period up to 24 hours."* Unauthenticated queries see
**only TLP:WHITE** records, and the public result `limit` ceiling is 1000 (10,000 authenticated).
<https://docs.mnemonic.no/api/services/pdns/01-public_api.html>

**The critical limitation.** mnemonic's own examples state that querying `cnn.com` *"Does not contain
subdomains of cnn.com."* **[measured]** confirmed: `GET /pdns/v3/*.iana.org` and a POST search with
`{"query":"*.iana.org"}` both return `count: 0` / `object.not.found`. **You cannot enumerate subdomains
with the public mnemonic API.**

What it *is* good for, and what verge-asm should use it for: **reverse resolution inside ranges the operator
owns.** Querying an IP returns every name observed pointing at it, with first/last-seen timestamps — which
is exactly how you find a host in your own /24 that no forward source knows about, and how you get historical
A-record evidence for drift ("this name pointed here until 2026-03, now it points at a cloud provider").

**Recommendation:** Tier 2, opt-in. The 1000/day quota is a shared community resource; a tool that ships it
on-by-default to an unknown number of installs is a good way to get the public endpoint closed. Enable it
for targeted IP→name questions, not bulk sweeps.

### 5.2 CIRCL Passive DNS — **not available**

*"Access to CIRCL Passive DNS is restricted to trusted partners both in Luxembourg and abroad"*; prospective
users must *"contact us and provide details about your affiliation and the intended use."*
<https://www.circl.lu/services/passive-dns/> Vetted, discretionary access. **Unusable for a tool run by
strangers.**

### 5.3 HackerTarget — **ToS-blocked as a default**

Offers keyless `hostsearch`, `dnslookup`, `reverseiplookup`, `aslookup` endpoints, free tier capped at
50 API calls/day at 2 req/s with HTTP 429 beyond. <https://hackertarget.com/ip-tools/>

But the Terms of Use restrict content to *"your personal and non-commercial use only"*, state
*"You may not, except with our express written permission, distribute or commercially exploit the content"*
and *"Nor may you transmit it or store it in any other website or other form of electronic retrieval
system"* — which is a direct description of what an ASM tool does with results — and explicitly exclude
those *"creating similar documents, goods or services for the purpose of providing them for a fee."*
<https://hackertarget.com/terms/>

**Do not ship as a default.** Storing HackerTarget results in verge-asm's asset database is precisely the
"other form of electronic retrieval system" the terms prohibit, and a security-owner using it for their
employer is not "personal and non-commercial".

### 5.4 Rapid7 Project Sonar / Open Data — **gone as a free bulk source**

The AWS Open Data Registry entry for Rapid7 FDNS ANY carries the notice *"The provider of this dataset will
no longer maintain this dataset"*, referencing a policy change of **10 February 2022**.
<https://registry.opendata.aws/rapid7-fdns-any/> **[measured]** `https://opendata.rapid7.com/` now 301s to
`https://sonardata.rapid7.com/`, whose API help states access requires a Sonar Data account and a user API
key sent in an `X-API-Key` header, with a default quota of *"30 operations in a 24 hour window"*, and
confirms *"The policies for accessing this data changed on Feb 10, 2022."*
<https://sonardata.rapid7.com/apihelp/>

Any design or tutorial that assumes free anonymous Sonar FDNS bulk downloads is out of date. **Not usable.**

### 5.5 Key-required passive DNS (documented for completeness, all out of scope for a keyless first run)

- **VirusTotal** — API key, free tier heavily rate-limited. <https://docs.virustotal.com/reference/overview>
- **SecurityTrails** — API key from a registered account. <https://docs.securitytrails.com/docs>
- **AlienVault OTX** — free but requires account registration for an API key. <https://otx.alienvault.com/api>
- **Farsight/DomainTools DNSDB** — commercial. <https://www.domaintools.com/products/farsight-dnsdb/>
- **ProjectDiscovery Chaos** — API key, and its dataset is scoped to public bug-bounty programmes, so it has
  no coverage for a typical small org. <https://chaos.projectdiscovery.io/>

These are the right shape for **operator-supplied-key opt-in integrations** in v1.1+, not for the default path.

### 5.6 ICANN CZDS (TLD zone files) — **registration-gated**

CZDS is *"an online portal where any interested party can request access to the Zone Files provided by
participating generic Top-Level Domains"*, requiring an account, per-TLD requests, and acceptance of the CZDS
Terms and Conditions. Zone files are **not** available anonymously. <https://czds.icann.org/>
Powerful (it is how you find *other* domains an org registered), but structurally incompatible with
"no account registration on first run." Correct treatment: an operator who already has CZDS access can point
verge-asm at their downloaded zone files.

---

## 6. Web archives and crawl indices

### 6.1 Common Crawl index — usable

**[measured]** `https://index.commoncrawl.org/collinfo.json` → 200, listing monthly indices
(latest `CC-MAIN-2026-30`, July 2026). A CDX query
`https://index.commoncrawl.org/CC-MAIN-2026-30-index?url=*.iana.org&output=json&limit=5` → 200 in 0.62 s,
returning `{urlkey, timestamp, url, mime, status, digest, length, offset, filename, languages}`.
Keyless, fast, open-data licensed. <https://commoncrawl.org/terms-of-use>

**Value:** low-to-moderate. It finds hostnames that were *linked from the crawled web*, which skews toward
public-facing marketing surface — the assets the operator already knows about. It occasionally surfaces a
forgotten host. Cheap enough to include as a Tier-1 supplement; not worth engineering effort beyond a single
CDX call per apex domain per index.

### 6.2 Wayback Machine CDX — **not usable as a default**

**[measured]** `https://web.archive.org/cdx/search/cdx?url=*.iana.org&…` returned **HTTP 429 Too Many
Requests on the very first request**, and again on retry with an identifying User-Agent. The public CDX
endpoint is now aggressively rate-limited/gated for unauthenticated clients. Whatever the nominal policy,
**empirically it does not work keylessly**, so it cannot be a default.

---

## 7. crt.sh reliability sample series

**[measured]** Eight requests for the identical URL `https://crt.sh/?q=%25.iana.org&output=json`, spaced
~13 s apart (i.e. deliberately *under* the documented 5 req/min limit, from a single IP, 2026-07-31 ~01:50 UTC):

| # | Result | Elapsed |
|---|---|---|
| 1 | 200 | 26.9 s |
| 2 | **502** | 0.2 s |
| 3 | 200 | **59.6 s** |
| 4 | **404** | 5.6 s |
| 5 | 200 | 28.4 s |
| 6 | 200 | 11.9 s |
| 7 | **404** | 4.6 s |
| 8 | **502** | 0.1 s |

**Success rate: 4/8 (50%).** Successful-response latency ranged **11.9 s – 59.6 s**. Combined with the
earlier series (one 503 at 16.9 s, then 200s at 16.3 s / 2.6 s / 2.3 s), the overall measured picture is
roughly **60% success with latency spanning 2 s to 60 s**.

**The 404s are the dangerous finding.** The same URL that returned a valid 95 KB JSON body seconds earlier
returned **HTTP 404** twice. A naive client that maps 404 → "no certificates exist for this domain" will
silently report **zero discovered assets** and, worse, in a drift-detection product will report every
previously-known certificate as **removed** — a fabricated all-clear or a flood of false "asset
disappeared" alerts.

**Hard requirements this imposes on the verge-asm crt.sh client:**

1. Treat `404`, `502`, `503`, and any non-200 as **transient failure**, never as an empty result set. Only a
   well-formed `200` with a parseable JSON array is evidence of anything.
2. Never compute drift (added/removed assets) from a run in which the CT index call did not succeed —
   mark the run's CT dimension as `unknown`, not `empty`.
3. Timeouts must be ≥ 60 s, because legitimate successful responses take that long.
4. Retry with jittered backoff while respecting the 5 req/min ceiling; cache successful per-domain results
   for hours (certificate data is immutable).
5. Surface "CT index degraded/unavailable" prominently in the UI for that run.

---

## 8. Source-by-source summary table

| Source | Yields | Misses | Keyless? | Rate limit (source) | Reliability | ToS constraint | Default? |
|---|---|---|---|---|---|---|---|
| Own recursive resolver | A/AAAA/CNAME/MX/NS/TXT/SOA/CAA/PTR | Names you don't know yet | Yes | Operator's own | Excellent | None | **Tier 0** |
| Wildcard detection (RFC 4592) | Poison signature for ~~a zone~~ **a name's child space** — the population is *the parents of the names in scope*, per §3.2 / [ADR-0066](../adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md); the signature is **per component and three-valued**, per §11 / [ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md) | — | Yes | n/a | ~~Excellent~~ **Excellent where the synthesis is determinate, absent where it is not** — **[measured]** §11 | None | **Tier 0 (mandatory)** |
| Zone file / AXFR (RFC 5936) | Complete authoritative zone | Zones the operator doesn't control | Yes | n/a | Excellent | Must be operator-authorised | **Tier 0** |
| NSEC walk (RFC 5155) | Full zone for NSEC-signed zones | NSEC3 zones, unsigned zones | Yes | n/a | Good | None (own zone) | **Tier 0 when applicable** |
| IANA RDAP bootstrap | TLD/IP/ASN → RDAP base URL | — | Yes | None observed | Excellent | None | **Tier 0** |
| RIR RDAP (ARIN/RIPE/APNIC/LACNIC/AFRINIC) | Prefixes, autnums, org→networks (ARIN, 202 CIDRs measured), abuse contacts | RIPE entity search 500s; registry ≠ BGP reality | Yes | ARIN: none documented, 25/25 OK measured. RIPE AUP: 1000 personal-data sets/24 h/IP | Excellent | ARIN ToU: no commercial-service use, no marketing. RIPE AUP: no bulk copy | **Tier 0** |
| gTLD domain RDAP | Expiry, status flags, nameservers, registrar | Registrant identity (redacted) | Yes | Per-registry | Good | Per-registry notices | **Tier 0 (expiry/drift only)** |
| RIPE DB REST search | inetnums matching an org string | RIPE region only | Yes | RIPE AUP | Good | RIPE AUP | Tier 1 |
| CT logs direct (RFC 6962 + static-ct-api) | SAN dNSNames, issuer, validity, chain | Unlogged/internal certs; no query-by-domain; 2.2–5.7 B entries per log | Yes | Chrome policy forbids limits that block monitors | Excellent | **None — operators may not impose conditions** | Tier 1 (tail only, not firehose) |
| **crt.sh** | Query-by-domain over CT: `name_value` SAN list, issuer, dates | 999-result cap; unlogged certs; wildcards hide names | Yes | **5 req/min per IP** (operator, 2023) | **Poor: 50% failure over 8 samples; 11.9–59.6 s when it works; spurious 404s** | No published ToS; operator has publicly asked for restraint | **Tier 1 (throttled + cached + failure-safe)** |
| SSLMate Cert Spotter | Same as crt.sh but fast and reliable; cursor pagination | Same CT-inherent misses | Yes (limited) | **10 full-domain queries/hour** (measured: 9 OK, then 429 + `Retry-After: 306`) | Excellent | **"personal or evaluation purposes"** | Tier 2 (opt-in w/ key) |
| RIPEstat Data API | IP→ASN+prefix, ASN→5310 prefixes, org name→ASNs | Low-visibility routes excluded | Yes | 8 concurrent; register above 1000/day | Excellent | **Non-commercial; no re-packaging/redistribution** | Tier 1 (with asterisk) |
| Team Cymru IP→ASN | Origin ASN, peer ASN, prefix, AS name | Not geolocation | Yes | **Null-routes abusive per-IP querying** | Good | Must use bulk interface | Tier 2 |
| bgp.tools | Rich BGP/ASN data | — | **No — login required** | Blocks generic UAs; bans HTML scrapers | Good | Explicitly polices automation | **Never default** |
| mnemonic Passive DNS | Historical rrtype/rrdata + first/last seen; **IP→names** | **No subdomain enumeration**; TLP:WHITE only | Yes | **10/min, 1000/day**, HTTP 402 | Good | Public quota is a shared resource | Tier 2 |
| CIRCL Passive DNS | — | — | **No — vetted partners only** | — | — | Restricted access | **Never default** |
| HackerTarget | hostsearch, reverse IP, AS lookup | — | Yes | 50 calls/day, 2 req/s | Good | **Personal, non-commercial only; no storing in another retrieval system** | **Never default** |
| Rapid7 Sonar | FDNS/RDNS bulk | — | **No — API key, 30 ops/24 h** | 30 ops/24 h | — | Policy changed 2022-02-10 | **Never default** |
| ICANN CZDS | gTLD zone files (find other owned domains) | — | **No — account + per-TLD approval** | — | Good | CZDS T&Cs | **Never default** (accept operator's own files) |
| Common Crawl index | Hostnames linked from the crawled web | Skews to known marketing surface | Yes | Generous | Good | Open data terms | Tier 1 (cheap supplement) |
| Wayback CDX | Historical URLs/hostnames | — | **Empirically no — 429 on first request** | — | **Unusable** | — | **Never default** |
| Public DoH JSON (Google/Cloudflare) | DNS answers over HTTPS | — | Yes | Not published | Excellent | Free-riding at scale | Fallback only |

---

## 9. What this means for the v1 spec

1. **The zone file is the product's spine.** The single biggest differentiator of a *defensive* ASM tool over
   an offensive recon script is that the operator can hand it ground truth. Onboarding should ask for it
   first, support upload and AXFR-with-TSIG, and every guess-based technique should be positioned as
   gap-filling for domains the operator *doesn't* control.

2. **CT is the best keyless discovery source, and crt.sh is its only keyless query-by-domain front door —
   which is a real single point of failure that fails ~50% of the time.** Design for it: 5 req/min ceiling,
   ≥60 s timeouts, aggressive per-domain caching (cert data is immutable), jittered backoff, a visible
   "CT index degraded" state, a strict rule that **non-200 never means "no certificates"** (crt.sh emits
   spurious 404s — see §7), and a pluggable CT-index interface so an operator can drop in a Cert Spotter key
   without a code change.

3. **CT direct-read needs two protocol clients now, not one.** RFC 6962 `get-entries` and C2SP static-ct-api
   tiles. Let's Encrypt is tiled-only and Oak is retired. Drive everything from the live log list.

4. **RDAP replaces WHOIS wholesale and needs no keys.** Implement bootstrap properly (cache the four IANA
   JSON files), and exploit ARIN's `entities?fn=` → `entity` → `networks` path — org name to 202 prefixes in
   two requests is genuinely the highest-leverage keyless capability found in this research. Fall back to
   RIPE DB REST for the RIPE region, where RDAP entity search returns 500.

5. **RIPEstat closes the ASN gap but carries a non-commercial clause.** Decide deliberately: default-on with
   a README note distinguishing self-hosted-own-estate use (fine) from paid-service use (needs RIPE NCC
   permission), or default-off for zero ambiguity. Recommendation: default-on, documented.

6. **Passive DNS is effectively unavailable keylessly.** Do not architect around it. Treat it as an
   opt-in enrichment, and make sure the discovery pipeline produces good results with it entirely absent.

7. **Do not brute-force by default.** It is active, noisy, wildcard-fragile, redundant against zone input,
   and has poor recall on exactly the forgotten assets the product exists to find.

8. **The ToS-blocked list is short but firm:** HackerTarget, bgp.tools, CIRCL, Rapid7 Sonar, CZDS, Wayback
   CDX, and unauthenticated Cert Spotter. Every one of these fails on a term that a stranger running
   verge-asm cannot satisfy. None should be reachable without the operator explicitly opting in and
   supplying their own credentials.

---

## 10. Open questions for the spec

- Is a CT *tail* (poll `get-sth` / `checkpoint`, read only the delta) worth the two-client implementation
  cost in v1, or is polling crt.sh per known apex domain sufficient for drift detection?
- ~~Should verge-asm ship a bundled, periodically-refreshed IP→ASN table (from RIPE RIS / RouteViews open
  data) to remove the RIPEstat dependency from the default path entirely?~~
  **Its premise is gone and the door it opens is now closed.** RIPEstat is not on the default path
  ([#15](https://github.com/winniel123/verge-asm/issues/15),
  [#19](https://github.com/winniel123/verge-asm/issues/19)); `Custody` reads `Seed`s alone and no ASN
  enters the model at all ([#27](https://github.com/winniel123/verge-asm/issues/27),
  [ADR-0012](../adr/0012-a-proposer-is-not-a-source.md)); and sibling expansion is keyed on the
  delegated files' opaque-id rather than on an ASN. A session reviving this must read
  [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md) first: a table built
  from route collectors may label **which AS announces an address** and may never be read as saying
  **whose address it is**, which is the only job the question was reaching for.
- What is the correct default posture for NSEC3 zones — attempt a small dictionary, or skip silently?
- Does the "free-fall risk" feature need registrar-expiry monitoring for domains outside the operator's
  supplied list, and if so how are those domains discovered without CZDS or reverse-WHOIS?

---

## 11. The wildcard match predicate — determinacy, measured per component

Settled by [#111](https://github.com/winniel123/verge-asm/issues/111) /
[ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md). This
section is the measured basis for §3.2 steps 2, 4 and 5 as they now read. §3.2 settles **where** the
control probe runs ([ADR-0066](../adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md));
this settles **how its answers are read**.

### 11.1 The rule

> **A wildcard is discriminated only at the components a control probe measured to be determinate.**
> A **component** is one `(qtype asked, RR type in the answer)` pair. Per component the signature is
> `NoSynthesis` │ `Determinate(RRset)` │ `Indeterminate`. A candidate is **discriminated** iff it
> differs at some **determinate** component — a different RRset where the control had one, or an
> RRset where the control determinately had none — and is **`Shadowed`** otherwise. An
> `Indeterminate` component is never consulted. Where **no** component is determinate, every name
> beneath that parent is `Shadowed`.

Two riders, both RFC 4592 §2.2.1 rather than policy. Discrimination is a fact about **the name**:
synthesis is blocked for *every* type once the name exists, so a candidate discriminated at any one
component exists and **none** of its RRsets is synthesised — including ones that coincide with the
signature. And `Shadowed` is therefore **all-or-nothing across a name's qtypes**: it holds on
`resolution` and on every `dns-record` discriminator, or on none.

### 11.2 *The* answer set is not an object

**[measured]** 2026-08-14, Google Public DNS DoH JSON from one vantage, and where stated direct to a
delegated authority via `nslookup`. Thirty long random labels per zone.

| Zone | Distinct A answer sets across 30 control labels |
| --- | --- |
| `github.io` · `localtest.me` | **1** |
| `vercel.com` | 2 addresses per label drawn from a closed pool of **8** |
| `herokuapp.com` | **8**, one per `vaNN`/`ieNN` ingress node, **pairwise disjoint**, 30 addresses total |

That is the ticket's finding. This is the one that decides the rule — the answer is not a function
of the label at all:

| Probe | Result |
| --- | --- |
| One label, six repeats, public resolver, `vercel.com` | **5** distinct address pairs |
| One label, four repeats, **direct to `ns01.herokudns.net`** | **4** different ingress nodes — `ie01`, `va06`, `va02`, `va04` — four disjoint address sets |
| One label, one moment, **four authorities of `herokuapp.com`** | four different answers |

The rotation is the **authority's own**, not an anycast-resolver artefact. A recorded signature is
one draw from a process; the next draw for the same label at the same authority seconds later is a
different one.

### 11.3 Components, because the stable and rotating parts share one answer

**[measured]** five control labels per zone, all seven declared qtypes:

| Zone | Determinate components | Indeterminate |
| --- | --- | --- |
| `github.io` | A (four `185.199.10x.153`), AAAA; NODATA at the other five | — |
| `localtest.me` | A (`127.0.0.1`), AAAA (`::1`); NODATA at the other five | — |
| `s3.amazonaws.com` | **CNAME (`s3-1-w.amazonaws.com.`)**, TXT, NS, SOA | A — ~~eight fresh addresses every label~~ **eight fresh addresses per *lookup*, and they are not this authority's** — see below |
| `appspot.com` | **MX** (the five `gmr-smtp-in.l.google.com.` hosts, identical every label) | A, AAAA |
| `vercel.com` | NODATA at six qtypes | A |
| `herokuapp.com` | **none, at any qtype** | CNAME, at all seven — target rotates over eight nodes |

`s3.amazonaws.com` is why the unit is the component and not the qtype: the rotating and the stable
parts sit inside **one qtype's answer chain**.

> **The attribution in that row is corrected by
> [#115](https://github.com/winniel123/verge-asm/issues/115) / §13.5, and the ruling is untouched.**
> **[measured]** direct to `ns-63.awsdns-07.com`, `s3.amazonaws.com` returns `CNAME
> s3-1-w.amazonaws.com.` and **no A record at all** — identical across 8 repeats, 8 distinct labels
> and all four authorities. `s3-1-w.amazonaws.com` is a **separate delegation** that this authority
> answers **REFUSED** for, and the eight rotating addresses are *its* zone's, spliced into our
> answer by the recursive resolver following the chain. The component is still `Indeterminate` from
> where the probe stands and it is still never consulted — but *this* authority does not rotate, and
> a session reading the row as *S3 rotates* would be wrong about which operator to ask.

**ADR-0066's seven-qtype widening pays a second time here.** **[measured]** `appspot.com`'s only
determinate *positive* component in the seven is **MX**; under §3.2's withdrawn A/AAAA/CNAME clause
that zone has no positive determinate component at all.

### 11.4 The base rate, which is what makes the strict rule affordable

**[measured]** nineteen zones, five long random labels each, A qtype — **five *distinct* labels, and
the distinctness is load-bearing rather than incidental** ([#116](https://github.com/winniel123/verge-asm/issues/116)):
it is the shipped probe's shape at a smaller *n*, so no reading here is a cache artefact of the
resolver these runs went through. §13.6's `Determinate`-over-a-resolver row is **one label repeated
eight times**, a shape the ruled ten-label set never takes, which is why this table stands and
`vercel.com` reads `Indeterminate` at A in it. See §14.2:

| | Count |
| --- | --- |
| Wildcarded (a random label answered) | **14** |
| Not wildcarded — NXDOMAIN: `pages.dev`, `workers.dev`, `azurewebsites.net`, `fly.dev`, `repl.co` | 5 |
| **Determinate at A** across five labels | ~~**10 of 14**~~ **8 of 14** — `github.io`, `localtest.me`, ~~`traefik.me`~~, `vcap.me`, `netlify.com`, `staging.render.com`, `railway.app`, `onrender.com`, ~~`surge.sh`~~, `glitch.me` (the two struck members are corrected below) |
| Indeterminate at A | 4 — `herokuapp.com`, `vercel.com`, `appspot.com`, `s3.amazonaws.com` |
| …of those, still carrying a determinate component elsewhere | 3 — `s3` (CNAME/TXT/NS/SOA), `appspot` (MX), `vercel` (determinate NODATA at six qtypes) |
| **No determinate component anywhere** | **1 of 14 — `herokuapp.com`** |

Set equality is not wrong; it is **unscoped**, and right for ~~ten~~ **eight** of fourteen outright.
Total suppression reaches **one measured parent in fourteen**.

> **Two rows of this table are corrected by §12, measured 2026-08-15.** `traefik.me` is filed under
> *determinate at A*; it is an **address-parsing authority** and reads `Indeterminate` once a
> structured label is in the set (§12.4). And `surge.sh` **does not reproduce** — a fresh five-label
> run returned two distinct answers, a ten-label run minutes later returned one (§12.6). So *10 of
> 14* is really **8 of 14** on today's evidence, with one member unstable. Both corrections point
> the same way: they are more zones needing §11.1's gate, which is the rule §11 already states.
>
> **§13 closes the *unstable* member.** `surge.sh` is not unstable — it is a **two-member hash of
> the query label**, `188/172` over 360 labels, identical on all four of its authorities and
> unchanged over 35 minutes. At five random labels it therefore reads `Determinate` **[measured]**
> 3 times in 30 and `Indeterminate` the other 27, which is a **sampling rate** rather than a
> coin-flip authority; at the raised count (§13.8) it reads `Indeterminate` 30 times in 30.

### 11.5 Why intersection-with-the-union lost

Signature = the first five labels of the thirty-label runs; test = the next twenty-five.

| Zone | Set equality catches | Intersection with the 5-label union catches |
| --- | --- | --- |
| `herokuapp.com` | **7 of 25** | **7 of 25 — identical** |
| `vercel.com` | 5 of 25 | **25 of 25** |

`herokuapp.com`'s synthesised sets are **disjoint blocks**, so *intersects the union* and *equals a
recorded set* are the **same predicate** there, and both leave **18 of 25 (72%)** fresh fictional
labels recorded `Resolved` with a fictional address set. On `vercel.com` a five-label union covers
seven of the pool's eight addresses and nothing can miss it. Intersection is a **total fix or
literally no fix**, and which one depends on the provider's load-balancing shape — a fact the probe
cannot observe and the `Batch` cannot record.

### 11.6 The cost of erring toward `Shadowed`, measured

**[measured]** five real GitHub Pages sites — `github.github.io`, `mozilla.github.io`,
`twbs.github.io`, `git-lfs.github.io`, `d3.github.io` — return **exactly** the wildcard's four
addresses, and `www.vercel.com` draws **both** its addresses from the wildcard's own pool. These are
genuinely present names that **no** content predicate can distinguish from synthesis. `Shadowed` on
them is the honest reading, not an avoidable error, and `CONTEXT.md` already accepted the collateral
— the suppression *"working as intended on the fictional names and swallowing the real one alongside
them."*

The asymmetry that decides the direction: a false `Shadowed` withholds one `resolution` value, is
confined to one facet, leaves the name in the estate (admission turns on its `Citation`) and is
visibly unconfirmed until §3.3's zone upload fixes it. A false `Resolved` fabricates an address set
that **cites `Address`es, opens `Service`s and `Endpoint`s, and feeds `Reach` and `Exposure`**, where
it is indistinguishable from a true one.

### 11.7 Residue — two holes this ruling does not close

> **BOTH DISCHARGED by [#113](https://github.com/winniel123/verge-asm/issues/113) /
> [ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md),
> which rules the control label's construction.** Neither needed an amendment to this section's
> ruling: §3.2 step 1 now sends a **structured** label alongside the random ones, both authorities
> below answer it differently from a random label, and §11.1's existing `Indeterminate` limb does
> the rest. `traefik.me`'s A component is no longer determinate, and `nip.io`'s licence is
> withheld. The two bullets are kept as the measurement that forced the ruling. See §12.

- ~~**A synthesis that is a function of the label looks determinate to random labels.**~~
  **[measured]** `traefik.me` answers `127.0.0.1` for every random control label — determinate by
  this section's own test — while `10.0.0.1.traefik.me` → `10.0.0.1`, `192.168.5.5.traefik.me` →
  `192.168.5.5`, `8.8.4.4.traefik.me` → `8.8.4.4`. Set equality then reports a fictional RFC 1918
  address as `Resolved`. *n* random labels **evidence** a constant synthesiser and cannot prove one.
  **Closed at the measurement**: `203-0-113-7.traefik.me` → `203.0.113.7`, so the A component reads
  `Indeterminate` and every name beneath is `Shadowed`.
- ~~**And a third door neither §3.2 nor this section closes.**~~ **[measured]** `nip.io` and
  `sslip.io` return **NODATA** for random control labels while `10.0.0.1.nip.io` → `10.0.0.1`. §3.2
  step 1 reports *no wildcard at all*, the probe completes, and ADR-0066's *a probe that completed
  and found no wildcard licenses everything beneath it* then licenses a fictional inventory. ~~The
  defect is in the **control label's construction** — `wildcard-discrimination`'s *other* declared
  parameter — and is ticketed rather than ruled here.~~ **Closed at step 6**: the licence now
  requires that no control label **of any shape** carried an RR at any qtype, and the structured
  label carries one.

### 11.8 The DNSSEC discriminator, and why it is not the answer

RFC 4035 §3.1.3 gives the only **sound** test that exists: a wildcard-synthesised answer's RRSIG
`Labels` field is shorter than the owner's label count, and the response carries NSEC/NSEC3 proving
no closer match. It is measured unavailable.

**[measured]** exactly **1 of 15** zones probed carries a DS — `herokuapp.com`, the one zone with no
determinate component at all. And it **online-signs its synthesised answers**: with `do=1` a random
label returns `RRSIG cname 13 3 300 …`, `Labels` = **3** against a 3-label owner, and no NSEC3. The
proof that would discriminate is never served. For contrast, `<random>.iana.org` — signed and not
wildcarded — returns three NSEC3 records and their RRSIGs, so the machinery exists and works where
there is no wildcard in front of it.

It is therefore missing exactly where content discrimination is also missing, and adopting it would
change the leaf's query mode (DO bit, RRSIG parsing, NSEC3 handling). Ticketed, not folded.

### 11.9 Where this is thin

- **Every indeterminate zone measured is a third-party hosting provider's.** ADR-0066 intersects the
  control-probe population with the operator's `Seed`, which excludes most of what is measured here.
  Whether a small org's `*.dev.example.com` in front of an ingress load balancer rotates is
  **unmeasured**, there being no public sample of private zones. Read *1 of 14* as a rate over
  **provider** zones.
- **One vantage, one day.** `herokuapp.com`'s `ie0x` and `va0x` nodes are Ireland and Virginia, so
  part of the rotation is geographic. A multi-vantage run would see **more** rotation, never less:
  ~~*10 of 14 determinate*~~ ***8 of 14 determinate*** (corrected at §11.4's table) is an **upper
  bound**, and the direction of the error is safe.
- **No control probe has ever run inside a batch.** What is measured is DNS behaviour against live
  authorities over a resolver we do not control, plus four direct-to-authority runs.

---

## 12. The control label's construction — one label, and a set that can be falsified

Ruled by [#113](https://github.com/winniel123/verge-asm/issues/113) /
[ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md).
This closes `wildcard-discrimination`'s **last** declared parameter without a value, and it
discharges **both** residues §11.7 recorded. §3.2 step 1 and step 6 are amended in place.

### 12.1 The rule

**A control label is exactly one label, and a control-label set that cannot falsify
label-independence is not a measurement of it.** The set is ~~**5 random labels + 1 structured
label**, six per site~~ **9 random labels + 1 structured label, ten per site
([#115](https://github.com/winniel123/verge-asm/issues/115) / §13 — the construction below is
untouched and only the count moves)**, every one of them run over the declared qtype set.

The structured label is `<a>-<b>-<c>-<d>` — the four octets of an address drawn at random from
**RFC 5737** documentation space (`192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`). It is
randomised for the same reason *long random* is: to make accidental existence negligible.

**Nothing reads the result that did not already exist.** §11.1's three-member union closes over
these cases unchanged — labels that disagree at a component make it `Indeterminate`, whether they
disagree in an RRset's contents or in whether an RRset was carried at all.

### 12.2 Why RFC 4592 makes this necessary rather than prudent

§3.2's whole architecture rests on RFC 4592's guarantee that synthesis is a function of the
**wildcard RRset** and not of the query name — which is what lets one label speak for every label.
An authority that **parses** the query name is not an RFC 4592 wildcard, the guarantee is absent,
and a random label carries nothing for it to parse. So the failure is invisible by construction, not
by bad luck.

### 12.3 One label — measured, and it is ADR-0066's own warrant

ADR-0066 admits the probe on an equivalence: a label constructed under P falls off the tree at the
encloser the names under P fall off at. Every candidate is **one** label under its parent. A dotted
quad is four, and the equivalence does not survive it.

**[measured]** 2026-08-15, Google Public DNS DoH JSON, one vantage:

| Zone | `<random32>` | `10-0-0-1` (one label) | `10.0.0.1` (four labels) |
| --- | --- | --- | --- |
| `pages.dev` | NXDOMAIN | NXDOMAIN | **`172.66.44.200`, `172.66.47.56`** |
| `workers.dev` | NXDOMAIN | NXDOMAIN | **`104.21.31.174`, `172.67.178.236`** |
| `fly.dev` | NXDOMAIN | NXDOMAIN | **NODATA** |
| `azurewebsites.net` | NXDOMAIN | NXDOMAIN | NXDOMAIN |
| `repl.co` | NXDOMAIN | NXDOMAIN | NXDOMAIN |

**3 of 5** un-wildcarded zones are made to look wildcarded. The mechanism was retrieved rather than
inferred:

| Probe | Answer |
| --- | --- |
| `1.pages.dev` | `172.66.44.200`, `172.66.47.56` |
| `zz9q7x.1.pages.dev` | `172.66.44.200`, `172.66.47.56` |
| `1.workers.dev` | NODATA |
| `zz9q7x.0.1.workers.dev` | `104.21.31.174`, `172.67.178.236` |

`1.pages.dev` is a **third party's** Cloudflare Pages project carrying a wildcard beneath it, so
`10.0.0.1.pages.dev` is synthesised by `*.1.pages.dev` — two labels below the probe site, by a party
with no relation to the operator. The probe would file that stranger's wildcard as `pages.dev`'s,
and under §11.1 the disagreement with the random labels reads `Indeterminate`, shadowing **every**
name the operator holds under `pages.dev`.

The dotted form is also **strictly dominated**: all three parsing authorities decode the hyphenated
form too, so it detects nothing extra for its three false wildcards.

### 12.4 The structured label separates a parser from a wildcard

| Zone | `<random32>` ×2 | `10-0-0-1` | `192-168-5-5` | `203-0-113-7` | Verdict |
| --- | --- | --- | --- | --- | --- |
| `nip.io` | NODATA | `10.0.0.1` | `192.168.5.5` | `203.0.113.7` | **parser** |
| `sslip.io` | NODATA | `10.0.0.1` | `192.168.5.5` | `203.0.113.7` | **parser** |
| `traefik.me` | `127.0.0.1` | `10.0.0.1` | `192.168.5.5` | `203.0.113.7` | **parser** |
| `localtest.me` | `127.0.0.1` | `127.0.0.1` | `127.0.0.1` | `127.0.0.1` | constant |
| `github.io` | the four `185.199.10x.153` | identical | identical | identical | constant |

RFC 5737 space is decoded exactly as RFC 1918 space is, on all three — so the structured label keeps
*long random*'s defence against accidental existence instead of trading it for detection.

It also reaches the awkward case, which was checked rather than assumed: a quad-shaped candidate has
a quad-shaped **parent**, and the probe still fires there.

| Probe | Answer |
| --- | --- |
| `<random32>.0.0.1.nip.io` | NODATA |
| `10-0-0-1.0.0.1.nip.io` | **`10.0.0.1`** |

### 12.5 Affordability — 0 false `Indeterminate` in 22 zones

The strict direction is affordable only if a structured label does not turn honest `Determinate`
verdicts into `Indeterminate` ones. **[measured]** across 22 zones it turns over none:

| Class | Zones | Verdict change |
| --- | --- | --- |
| Address-parsing authorities | `nip.io`, `sslip.io`, `traefik.me` | **3 changed, all correctly** |
| Constant wildcards | `github.io`, `netlify.com`, `railway.app`, `onrender.com`, `glitch.me`, `staging.render.com`, `localtest.me`, `vcap.me`, `lvh.me`, `local.gd`, `localho.st`, `fbi.com` | 0 — byte-identical RRsets across every shape |
| Un-wildcarded | `pages.dev`, `workers.dev`, `azurewebsites.net`, `fly.dev`, `repl.co` | 0 — NXDOMAIN on every single-label shape |
| No synthesis at A | `1u.ms` | 0 |
| Already unstable | `surge.sh` | 0 — it moves under random labels alone |

**Cost:** ~~6 labels × 7 qtypes per parent against 5 × 7 — **+20%**. On the `%.iana.org` estate §7
sampled, 6 parents: **252 resolver queries per batch against 210**.~~ **SUPERSEDED by
[#115](https://github.com/winniel123/verge-asm/issues/115) / §13.8, which raises the count: it is
**10 labels × 7 qtypes per parent**, and on the same estate **420 resolver queries per batch**. The
+20% this line prices was the structured label's; the current figure includes the raised count.**
These go to a resolver, not to
the operator's hosts, so §6.3's per-target ceilings in
[`safe-active-probing.md`](./safe-active-probing.md) do not bind them — and no document prices
resolver load, which is stated rather than assumed away.

### 12.6 ~~The count becomes `5`, and is refused a larger value on the measurement~~ — the refusal is REVERSED by §13

> **This subsection's ruling is WITHDRAWN by [#115](https://github.com/winniel123/verge-asm/issues/115)
> / §13, at the site that states it
> ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
> Read alone and in the present tense, *the number stays at 5* and *more labels is not monotonically
> more sensitive here* would build the set this section refused to improve. **[measured]** in 30
> independent trials the false-`Determinate` rate on `surge.sh` **is** monotone in the count —
> 13.3% at 4 draws, 6.7% at 6, 0% at 8 and above — and the mechanism this subsection could not name
> is **per-label sharding over a near-fair binary hash**, isolated in §13.3. The single ten-label
> run below is a draw from a 0.2% tail that **30 fresh trials did not reproduce**. The random count
> is now **9**. What survives verbatim is the *reason* stated here — a count bought against an
> unidentified process cannot be priced in band — which is why §13 identified the process before
> moving the number.

It has to become a **value** whatever else happens: ADR-0021's gate is bidirectional on a changed
declared parameter, and `3–5` cannot be diffed. It goes to the top of the range because §11 gave it
a second job — sampling a varying answer — and nothing argues for fewer.

Raising it further was the live question. **[measured]** `surge.sh`, 2026-08-15:

| Run | Labels | Distinct A answers |
| --- | --- | --- |
| Five distinct random labels | 5 | **2** — `159.203.50.177`, `159.203.159.100`, alternating |
| Ten distinct random labels, minutes later | 10 | **1** — `159.203.50.177` |
| One label, six repeats | 1 | 1 |

~~The larger run saw **less** variation than the smaller one, so more labels is not monotonically more
sensitive here~~, and the mechanism — per-label sharding, per-query rotation, a time window, or
resolver caching in front of any of them — ~~**was not isolated**~~ **is per-label sharding, and it
IS isolated — §13.3**. A count bought against an
unidentified process is a purchase whose value cannot be stated in band, which is §11.5's own
objection to intersection-with-the-union. ~~The number stays at **5** and the question is ticketed.~~
**The question was ticketed as [#115](https://github.com/winniel123/verge-asm/issues/115) and is now
answered: the random count is **9**, bought against a measured process at a stated price.**

Two riders. §11.4's base-rate table files `surge.sh` under *determinate at A across five labels*;
**today that does not reproduce**, so *10 of 14* has at least one member that is a coin-flip rather
than a fact — which does not weaken §11's ruling, it is one more zone needing the gate §11 ruled.
**§13.3 supplies the missing half: it is not a coin-flip, it is a two-member hash *of the label*,
and each of five labels lands in one of the two shards independently.**
And the structured label is a **sixth draw** ~~, so the set's power against a varying answer rises even
though the random count does not~~ **— it is now the **tenth**, and §13.8 counts it as a draw for
exactly the reason this sentence gives.**

### 12.7 Residue — the searched corpus of encodings

Four encodings were probed and **one ships**. This is a bounded residue disclosed as a searched
corpus, not a caveat.

| Encoding | Example | `nip.io` | `sslip.io` | `traefik.me` | Disposition |
| --- | --- | --- | --- | --- | --- |
| Dotted decimal | `10.0.0.1` | decodes | decodes | decodes | **Refused** — four labels, §12.3 |
| Hyphenated decimal | `10-0-0-1` | decodes | decodes | decodes | **Ships** |
| Hex | `cb007107` | decodes | decodes | **no** — returns its constant | Refused, redundant |
| Hyphenated IPv6 | `2001-db8--1` | decodes | decodes | decodes | Refused, redundant |

Hyphenated decimal is decoded by **3 of 3** parsers, and **0 of 3** decode any other family without
also decoding it — so it is the base form and the rest are optional extras that reach no finding and
prevent no measured falsity ([ADR-0030](../adr/0030-an-offer-is-admitted-on-a-finding-or-on-a-falsity-it-prevents.md)'s
bar), at a 7-query multiplier each.

**What stays invisible:** an authority whose *only* decodable form is one we do not send. The
boundary is **falsifiable by naming one such authority**, which is what makes it bounded rather than
permanent.

### 12.8 Where this is thin

- **The corpus is three parsers, all public developer conveniences.** ADR-0066 intersects the
  population with the operator's `Seed`, so the modal operator never probes `nip.io`. Whether a
  small org runs an address-parsing authority inside its own zone is **unmeasured** — the same hole
  §11.9 and ADR-0066 each flagged one question over.
- **A structured label that happens to exist produces a false `Indeterminate`.** The reserved-range
  space is 3 × 254 forms, far smaller than a 32-character random label's. The consequence is
  **total suppression of that parent** — the safe direction, and §11.6's own posture — never a
  fabricated address.
- **One vantage, one day, over a resolver we do not control.** ~~`surge.sh`'s two answers were not
  separated from cache behaviour, which is exactly why the count was not moved on them.~~
  **They are now separated ([#115](https://github.com/winniel123/verge-asm/issues/115) / §13.6):
  direct to `ns1.surge.world`, one label repeated eight times returns one answer and eight distinct
  labels return two, so the process is per-label and the cache is not in it. The count moved.**
- **No control probe has ever run inside a batch.** What is measured is DNS behaviour against live
  authorities.

---

## 13. The instability's mechanism, and what the control-label count buys against it

Ruled by [#115](https://github.com/winniel123/verge-asm/issues/115), which **amends
[ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md)
in place** rather than minting a rule beside it. §3.2 settles **where** the control probe runs, §11
**how its answers are read**, §12 **what a control label is**; this settles **how many**, by
identifying the process the count is bought against — which is the thing §12.6 refused to guess at.

### 13.1 The rule

> **A wildcard authority's answer instability is per-query at one authority and per-label at
> another; it is not per-time at any authority measured; and the only lever that survives the
> vantage the probe actually stands at is the count of *distinct* labels. The random count moves
> from `5` to `9`, so the set is 9 random + 1 structured — ten labels per site, each exactly one
> label, each run over the declared qtype set.**

Two riders that are the rest of the ticket's question.

- **`Determinate` acquires no shelf life, because it already has one.** ADR-0011 decides the value
  inside the batch that measured it and never assembles it across batches; that confinement is what
  prices per-query rotation, and it turns out to be load-bearing rather than housekeeping.
- **Nothing else moves.** ADR-0068's per-component gate, its three-member union and its
  `Indeterminate` limb are untouched, and ADR-0069's construction — one label, hyphenated quad, RFC
  5737 space — is untouched. A **number** moves, which is the only thing §12.6 left open.

### 13.2 The design, stated before the numbers

The four mechanisms are indistinguishable from a single vantage over a caching resolver, so every
run below goes **direct to the delegated authority**, which is the move that made ADR-0068's four
confirming runs sound. Each run holds one thing still.

| Run | Probe | What it isolates |
| --- | --- | --- |
| **A** | one label, **8 back-to-back repeats**, one authority | **per-query** — the label is held still, so any variation is the query's |
| **B** | **8 distinct labels**, back-to-back, same authority, same window as A | **per-label** — more distinct answers than A means the label is an input |
| **C** | **B's own labels re-probed** at +5, +10, +20 and +35 min | **per-time** — a moved label→answer map under an unchanged label set is per-time and nothing else |
| **E** | one label, one instant, **every authority** of the zone | **per-authority** — a shard of the server rather than of the name |
| **D** | one label ×8 at the authority **against** one label ×8 over Google DoH | **resolver caching** in front of any of the above |
| **J** | **30 trials × 12 distinct labels**, the verdict read at every draw count | whether *n* draws are **independent samples**, which is the assumption the count rests on |

A control on the instrument comes first, because the design collapses without it: a *constant*
result has to be the authority's and not the box's. **[measured]** 2026-08-15,
`Resolve-DnsName -Server <auth-ip>` leaves **0** entries in the Windows DNS client cache for the
name it probed and hits none — so the constant rows below are constants at the authority.

The population is §12.6's: ADR-0068's four indeterminate zones plus `surge.sh`, the five zones on
which the question exists at all.

### 13.3 It varies by authority, and that is the finding

**[measured]** 2026-08-15, direct to a delegated authority of each zone, one vantage.

| Zone | **A** — 1 label ×8 | **B** — 8 labels | **E** — across authorities | Mechanism |
| --- | --- | --- | --- | --- |
| `herokuapp.com` | **4 distinct** | 5 distinct | **3 distinct** of 4 servers | **per-query**, and per-authority besides |
| `vercel.com` | **6 distinct** | 7 distinct | **2 distinct** of 2 servers | **per-query**, and per-authority besides |
| `appspot.com` | **1** | **7 distinct** | 1 — all four agree | **per-label** |
| `surge.sh` | **1** | **2 distinct** | 1 — all four agree | **per-label** |
| `s3.amazonaws.com` | 1 | 1 | 1 — all four agree | **none of the four** — see §13.5 |

Run A is what separates the two live mechanisms and it separates them cleanly. `herokuapp.com` and
`vercel.com` vary while the label is held **fixed**, so the label is not an input and distinctness
is irrelevant to them; `appspot.com` and `surge.sh` do not vary at all under repetition and vary
freely across labels, so on those the answer is a **function of the query name**.

The per-label function is deterministic, not merely repeatable within a burst: **[measured]** every
one of run C's five passes reproduced run B's label→answer map on `appspot.com` and `surge.sh`
**byte for byte**, eight labels each, out to **+35 minutes** — §13.4.

Two per-label pools, sized:

| Zone | Distinct answers over 40 labels | Shape |
| --- | --- | --- |
| `appspot.com` | **11** | a wide, weighted spread of Google front-end `…153` addresses |
| `surge.sh` | **2** | `159.203.50.177` / `159.203.159.100`, **188 / 172 over 360 labels** — a near-fair binary hash |

`surge.sh` is the whole count question in one row — a **two-member** pool is the smallest that can
produce a false `Determinate`, and it is the only one in the corpus. §13.7 and §13.8 are about it.

### 13.4 No authority measured is per-time, and that is what decides the lever

Run C re-probed run B's **own eight labels** at each site at +5, +10, +20 and +35 minutes, at the
same authority.

**[measured]** 2026-08-15, 02:37–03:13 UTC, five passes per zone. The unit is the **whole
label→answer map** — all eight labels as one string — so *1 of 5* means the authority answered every
one of the eight identically at every pass.

| Zone | Mechanism | Distinct **maps** over the five passes | The one fixed label, ×4 per pass |
| --- | --- | --- | --- |
| `appspot.com` | per-label | **1 of 5** — byte-identical at +0, +5, +10, +20, +35 | 1 distinct every pass, and the **same** address `64.233.176.153` at all five |
| `surge.sh` | per-label | **1 of 5** — byte-identical | 1 distinct every pass, the same `159.203.50.177` at all five |
| `s3.amazonaws.com` | — (§13.5) | **1 of 5** | 1 distinct, the same CNAME |
| `vercel.com` | per-query | **5 of 5** — a fresh map every pass | **3–4 distinct in four repeats**, every pass |
| `herokuapp.com` | per-query | **5 of 5** | **3–4 distinct in four repeats**, every pass |

The two columns say the same thing twice. Where the answer is a function of the label it is the
**same** function 36 minutes later; where it is not, it was already redrawing between two
back-to-back queries and spacing adds nothing to that.

That is the negative result the ruling turns on. On the two per-label zones the map does not move
inside a window many times a batch's length, so **spacing buys nothing there**. On the two
per-query zones every pass is a fresh draw whether it is spaced or not, so **spacing buys nothing
there either** — it buys exactly what one more back-to-back query buys, at minutes of wall clock
instead of milliseconds.

*Per-time* is not disproved as a thing that can happen — a deploy or a health check plainly can
move an answer, and §13.10 says so. What is measured is that **it is not the mechanism on any zone
where the instability actually exists**, and a parameter is bought against measured processes.

### 13.5 `s3.amazonaws.com` is not unstable, and ADR-0068 attributes it to the wrong zone

ADR-0068 files `s3.amazonaws.com` as *rotating eight A addresses on every label*. **[measured]** the
authority does no such thing: over 8 repeats, 8 distinct labels and all four `awsdns` servers it
returns exactly `CNAME s3-1-w.amazonaws.com.` and **no A record at all**, at a CNAME TTL of
**42 821 s**.

The rotation is real and it belongs to a **different zone**. `s3-1-w.amazonaws.com` is a separate
delegation — the `s3.amazonaws.com` authority answers **REFUSED** for it — and **[measured]** over
a recursive resolver it serves eight fresh addresses per lookup, which the resolver then splices
into the answer chain the probe reads.

So the `Indeterminate` verdict at that component is correct **about what the instrument sees** and
wrong about the authority it names. Nothing normative moves: the probe reads what its vantage
returns, the CNAME component is determinate, and names beneath are discriminable there. The
attribution is corrected at §11.3 and at ADR-0068's own row, because a claim about which authority
rotates is a claim a later session will act on.

This is the ticket's fourth mechanism arriving in an unexpected shape. It is not caching — it is
the vantage **following a chain across a zone cut** and returning another operator's instability
inside our answer.

### 13.6 Caching is why *repeats* lose and *distinct labels* win

The instrument does not stand at the authority. ADR-0069 prices the control probe in **resolver
queries** — §12.5's cost line, now 420 per batch on §7's estate — so the mechanism that matters is
the one that survives a recursive resolver in front of it.

**[measured]** one label, eight repeats, at the authority and over Google DoH, with the authority's
own TTL:

| Zone | TTL of the synthesised answer | Distinct at the **authority** | Distinct over the **resolver** |
| --- | --- | --- | --- |
| `vercel.com` | **A 1800** | **6** | **1** |
| `herokuapp.com` | CNAME 300, A 60 | **7** | 2 |
| `appspot.com` | A 300 | 1 | 1 |
| `surge.sh` | A 301 | 1 | 1 |
| `s3.amazonaws.com` | CNAME 42 821 | 1 | 2 — the §13.5 chain |

`vercel.com` is the decisive row and it decides against the ticket's third lever. The authority is
the most unstable thing in the sample, and behind a resolver with a **30-minute** TTL a repeat of
one label sees **one** answer for the whole batch and every batch that follows within the half
hour. **Repeating one label is annihilated by the cache the probe sits behind.**

> **Two riders from [#116](https://github.com/winniel123/verge-asm/issues/116) / §14, neither
> disturbing the conclusion.** First, **that `Determinate` is this table's *repeat* column and not
> the shipped probe's**: the ruled set is **ten distinct** labels, which by this section's own
> structural sentence are ten distinct cache entries and ten fresh draws — **[measured]** ten
> distinct labels under `vercel.com` through a recursive resolver return **9 distinct A sets**,
> `Indeterminate`. Nothing that reads *`vercel.com` is `Determinate` over a resolver* may be carried
> to the probe as built. Second, **the annihilation is a property of that resolver rather than of
> resolvers**: **[measured]** at a second vantage the same one-label ×8 repeat returns **7 distinct**
> answers at the same reported TTL of 1800. The lever ranking is unchanged — distinct labels weakly
> dominate at every mechanism and are unaffected by either finding — but the *degree* of
> annihilation is a fact about the vantage, which is why §14 makes the vantage declared.

Distinct labels are not, and the reason is structural rather than lucky: *n* distinct labels are *n*
distinct cache entries, so each one is a fresh miss and a fresh draw at the authority. That gives
distinct labels the property repeats do not have:

| Mechanism | More **distinct labels** | More **repeats of one label** | More **spacing** |
| --- | --- | --- | --- |
| per-label (`appspot`, `surge`) | **the only lever that works** | nothing — the answer is a function of the label | nothing measured (§13.4) |
| per-query (`herokuapp`, `vercel`) | **one fresh draw each** | one fresh draw each **at the authority**, and **zero** through a cache | same as a repeat, minutes later |
| per-authority | one draw each, resolver's choice | same | same |
| per-time | nothing | nothing | the lever — and no zone needs it |

Distinct labels **weakly dominate** the other two levers at every measured mechanism and strictly
dominate them at three. That is why the count is the parameter that moves and the other two are
refused.

### 13.7 The draws are independent, so the count buys what §12.6 doubted — measured

This is the assumption the whole instrument rests on and it had never been tested. **[measured]**
30 independent trials of 12 distinct labels under `surge.sh`, direct to `ns1.surge.world`, the
verdict read from the first *n* of each trial:

| Draws in the set | False `Determinate` at A | Rate | Balanced-pool model `k^(1-n)` |
| --- | --- | --- | --- |
| 4 | 4 / 30 | 13.3 % | 12.5 % |
| 5 | 3 / 30 | 10.0 % | 6.3 % |
| **6 — the set today** | **2 / 30** | **6.7 %** | **3.1 %** |
| 7 | 1 / 30 | 3.3 % | 1.6 % |
| 8 | 0 / 30 | 0 % | 0.8 % |
| 9 | 0 / 30 | 0 % | 0.4 % |
| **10 — the set ruled** | **0 / 30** | **0 %** | **0.2 %** |
| 11 · 12 | 0 / 30 | 0 % | 0.1 % · 0.05 % |

The rate is **monotone decreasing** and tracks the independent-sampling model. A second confirming
run: **[measured]** 20 trials of 10 labels each gave **2/20** false `Determinate` at 5 draws and
**0/20** at 10, on `surge.sh`, and **0/20 at both** on `appspot.com`, `herokuapp.com` and
`vercel.com` — the three zones whose pools are too wide or too fast to be missed at any count.

**So §12.6's observation was a sample, not a law.** Ten labels seeing *one* answer where five saw
*two* is a draw from a distribution whose ten-label tail is **0.2 %**, and 30 fresh trials did not
reproduce it once. §12.6 read a single pair of runs as evidence that *more labels is not
monotonically more sensitive*; it is, and the count buys exactly the sensitivity `k^(1-n)` says it
buys, on the one zone in the corpus where the purchase matters.

### 13.8 Why `9 + 1`, and not `7 + 1` or `19 + 1`

The count is bought against the **measured** failure and nothing else: a two-member per-label pool,
which is the smallest pool that can produce a false `Determinate` and is the shape of the one zone
that produces one. `surge.sh`'s split is **188 / 172 over 360 labels** — p̂ = 0.522, a 95 % interval
of roughly [0.47, 0.57].

**What a false `Determinate` costs is why the bar is where it is**, and it is the expensive
direction rather than the safe one. If all ten labels land in one shard, §11.1 records
`Determinate(159.203.50.177)` — and then every fictional candidate beneath that parent that hashes
to the *other* shard **differs at a determinate component**, so it is discriminated, recorded
`Resolved`, and cites a fabricated `Address`. On this zone that is roughly **half** the fictional
names beneath the parent. §11.6's asymmetry applies at full strength: a false `Shadowed` withholds
one value, a false `Resolved` opens `Service`s and `Endpoint`s.

The bar is **a false `Determinate` below 1 % on that pool across the whole interval**, and the value
is the smallest count that clears it:

| Draws | Miss at p̂ = 0.52 | Miss at the interval's worst end, p = 0.57 |
| --- | --- | --- |
| 6 — today | 3.2 % | 4.0 % |
| 8 | 0.8 % | 1.2 % — **fails the bar** |
| 9 | 0.4 % | 0.7 % |
| **10 — ruled** | **0.21 %** | **0.39 %** |

Ten draws, and the set already carries one structured label which is a draw like any other against a
label hash — §12.6 said so of the sixth and it is equally true of the tenth. So **the random count
becomes 9** and the construction is untouched, which is what ADR-0069 requires of a larger count:
*a larger count adds random labels; it does not revisit the construction*.

**`9` is still authored data**, which is the bar ADR-0066 set when it pushed the population out of
the parameter table and which §11.5 used again to kill a convergence stopping rule. The number is
written down once, from a published measurement, and every batch runs the same ten labels; it is
**not** a function of what this batch found, so no eighth aperture input and no second `Batch` scope
dimension is needed. A count that grew until the observed pool stopped growing would be the barred
shape, and it stays barred.

**Why not more.** The purchase has sharply diminishing returns against a linear query cost — 11
draws buys 0.1 %, 12 buys 0.05 % — and, more decisively, the residual it would be chasing is a shape
the count **cannot** reach. A second answer served to a fraction *f* of labels is missed with
probability `(1-f)^n`; at ten draws a 50 % member is missed 0.1 % of the time, a 20 % member **11 %**
of the time and a 5 % member **60 %** of the time. Buying past the balanced-pool bar buys against
rare members, and no affordable *n* reaches those. The honest boundary is stated at §13.10 rather
than papered over with a bigger number.

**Cost, and the affordability check the raise is owed.** Query cost per site goes from **42** to
**70** — 10 labels × 7 qtypes against 6 × 7, **+67 %** — and on the `%.iana.org` estate §7 sampled,
6 parents, from **252** resolver queries per batch to **420**. These go to a resolver rather than to
the operator's hosts, so §6.3's per-target ceilings in
[`safe-active-probing.md`](./safe-active-probing.md) do not bind them, and ADR-0066's §6.3 arithmetic
against the 200 pkt/s global ceiling is untroubled at ~2 s of wall clock.

The raise is affordable in the other direction too — it must not turn honest `Determinate` verdicts
into `Indeterminate` ones. **[measured]** 2026-08-15, 10 random labels + 1 structured, **direct to
each zone's own authority**, across the classes §12.5 used:

| Class | Zones | Verdict change at 10 draws |
| --- | --- | --- |
| Constant wildcards | `github.io`, `netlify.com`, `railway.app`, `onrender.com`, `glitch.me`, `localtest.me`, `vcap.me`, `staging.render.com`, `lvh.me`, `local.gd`, `localho.st` | **0 of 11** — one answer at 10 labels, unchanged by the structured one |
| No synthesis at A | `1u.ms` | 0 |
| Un-wildcarded | `pages.dev`, `workers.dev`, `azurewebsites.net`, `fly.dev`, `repl.co` | **0 of 5** — NXDOMAIN on every label |

**0 of 17.** Doubling the count costs nothing in false `Indeterminate`, which is the same result
§12.5 got for the structured label and for the same reason: more labels can only find variation that
is there.

### 13.9 `Determinate` is a claim about a vantage at a moment — and it already expires

The ticket asks whether a determinate verdict has a shelf life. It does, the shelf is **the batch**,
and the model already builds it — so the answer is that nothing is added and one thing is stated.

**[measured]** the verdict is a claim about the **vantage**, not about the authority, three times
over in this section alone. `vercel.com`'s authority rotates six answers in eight repeats and reads
**`Determinate` for thirty minutes** behind a resolver honouring its 1800 s TTL. `herokuapp.com`
answers a different ingress node on **3 of 4** of its own servers at one instant. `s3.amazonaws.com`
reads `Indeterminate` at A only because the vantage crossed a zone cut (§13.5).

ADR-0011 already forbids assembling this value across observations, and ADR-0068 already files the
determinacy verdict as *a measured in-batch fact* rather than a parameter or an aperture input.
Together those two make the verdict expire at the batch boundary by construction: the control
probe's answers and the candidate's answer are drawn in one batch, compared in one batch, and the
verdict is never carried into the next. **What this ticket adds is that the confinement is
load-bearing** — it is what makes a per-query authority safe to reason about at all — and a session
tempted to cache a determinacy verdict between batches to save 420 queries would be reintroducing
exactly the cross-observation currency problem ADR-0011 refused, on a value measured here to move
within **eight consecutive queries**.

No new member, no new tier, no expiry field. `wildcard-synthesis`'s three-member per-component union
is untouched, `resolution` does not move, and nothing `Break`s in the model's shape.

> **The vantage in that sentence is now declared** ([#116](https://github.com/winniel123/verge-asm/issues/116)
> / §14). *A claim about a vantage at a moment* was, when this section was written, a claim about an
> **undeclared** vantage — which is what made it a caveat. It is now a claim about the vantage the
> candidate's own answer was drawn from, on one query path, in one batch, and that is the only
> vantage the predicate ever compares across. The sentence is unchanged and its status is not: it
> stops being a warning and becomes the ruling's premise.

### 13.10 Where this is thin

- **The count cannot reach a rare second answer, and that is a permanent boundary rather than a
  gap.** `(1-f)^n` at ten draws misses a 20 %-share member **11 %** of the time and a 5 %-share
  member **60 %** of the time. The count buys against balanced pools, which is what was measured;
  an authority that serves a second answer to one label in twenty is invisible to any affordable
  *n*, and the safe reading of a determinate verdict remains ADR-0068's — evidence, never proof.
- **Per-time is unfalsified rather than disproved.** The window measured is **35 minutes on one
  day**, and a deploy, a certificate rotation or a health-check flap plainly can move an answer over
  hours. What is ruled is that spacing is not worth a parameter *for the mechanisms that exist in
  the sample*, not that answers never move with time. Naming one zone whose label→answer map moves
  inside a batch's span falsifies this, which is what makes it bounded.
- **`surge.sh`'s binary hash carries the whole count argument.** It is **1 of 14** wildcarded zones
  and the only measured instance of the shape the raise is bought against; `appspot.com`'s pool is
  wide enough that no count is needed and the per-query zones are caught at any count. So the ruling
  moves a parameter for the estate on the strength of **one** zone's shape, argued from a model the
  other four are consistent with rather than from four confirmations.
- **§12.6's un-reproduced run.** Ten labels seeing one answer is a 0.2 % event under the measured
  hash and **30 fresh trials did not reproduce it**. It is filed as a sample from the tail, but a
  cause that was not ruled out is that #113's ten labels were not independent of one another — the
  generator was not recorded. That is a defect in the earlier evidence rather than in this one, and
  it is why this section measured trials rather than runs.
- **One vantage, one day, and no control probe has ever run inside a batch** — verbatim from
  ADR-0066, ADR-0068 and ADR-0069. What is measured is DNS behaviour against live authorities, now
  mostly direct rather than through a resolver, which is a different exposure and not a smaller one:
  the shipped probe stands where §13.6's resolver column stands, and every mechanism verdict here is
  read from the authority column.
- ~~**The probe's vantage is a parameter nobody has declared, and this section is the first evidence
  that it matters.**~~ **RULED by [#116](https://github.com/winniel123/verge-asm/issues/116) /
  [ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)
  — see §14.** What stands verbatim is the evidence: §13.5 and §13.6 both turn a determinacy verdict
  on whether a recursive resolver is in the path. What is withdrawn is the disposal. ~~Whether
  `wildcard-discrimination` should query **direct-to-authority** is a **query-mode** change — the
  class ADR-0068 refused the DNSSEC discriminator on — and it is not a number. **Not ruled here, and
  not folded in.**~~ Read alone and in the present tense, *not ruled here* would leave a session
  believing the path is still open; it is not. It is a **declared parameter shared by
  `resolution-walk` and `wildcard-discrimination`, one value per `Batch`**, valued at **the
  `Vantage`'s configured recursive resolver** — and *query-mode change* was never a third home,
  ADR-0068 itself calling a query-mode change *"a parameter change of its own"*. §14.
- **Every zone in the population is a third-party hosting provider's**, so *per-query at two of
  five* is a rate over **provider** zones and not over the operator estates ADR-0066's population
  actually intersects. Unchanged and unfixable from public data, and flagged for the fourth time.

---

## 14. The probe's vantage — one declared query path, and the resolver is part of the position

Ruled by [#116](https://github.com/winniel123/verge-asm/issues/116) /
[ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md).
§3.2 settles **where** the control probe runs, §11 **how its answers are read**, §12 **what a
control label is**, §13 **how many**; this settles **where it is asked from** — the fourth axis, and
the one §13.10 recorded as a parameter nobody had declared.

**[measured]** 2026-08-15, from a **second vantage** — a host resolver on a different network from
the one §11, §12 and §13 were measured at — with direct-to-authority runs against each zone's own
delegated servers. Every figure below is new to this document.

### 14.1 The rule

> **A control probe is asked from where the answer it discriminates was asked from.** The **query
> path** — direct to the delegated authorities, or through the `Vantage`'s configured recursive
> resolver — is a **declared parameter shared by `resolution-walk` and `wildcard-discrimination`**,
> taking **exactly one value per `Batch`**. Its value is **the `Vantage`'s configured recursive
> resolver**. **Which resolver stands there is part of the `Vantage`'s identity**, in the timeline
> key, never in the scope record and never inside a leaf.

Three riders, each answering a limb the ticket asked separately.

- **`resolution-walk` has the same hole and it is one sentence wider.** The leaf already makes
  **both** queries — a delegation walk for `Lame`, a resolution for the address set — and nothing
  said which answer is the value. It is now stated: **`Resolved`, `NoData` and `NameError` are read
  on the declared path**; the delegation walk decides `Lame` and the per-nameserver
  `serves │ does-not-serve` RRset and **supplies no address set**; and **the parameter does not
  govern the walk**, because `Lame`'s availability is its own definition's rather than a setting's.
- **A determinacy verdict measured over a resolver is admissible**, and §11.4's base rate is **not**
  re-opened. §14.2.
- **The operator's own resolver is a `Vantage` fact, not a parameter fact.** §14.5.

### 14.2 The base rate is not a cache artefact, because it was never measured with a repeat

This is the limb that could have cost something, and it is closed by reading the two measurements
apart rather than by re-measuring.

| Reading | Probe shape | Verdict at `vercel.com`'s A component |
| --- | --- | --- |
| §13.6, over Google Public DNS | **one label, 8 repeats** | `Determinate` — a 30-minute cache artefact at a 1800 s TTL |
| §11.3 / §11.4, over a resolver | **5 distinct** random labels | **`Indeterminate`** |
| **This section**, over a resolver, second vantage | **10 distinct** random labels | **`Indeterminate` — 9 distinct A sets in 10 labels** |

§13.6 supplies its own reason: *"n distinct labels are n distinct cache entries, so each one is a
fresh miss and a fresh draw at the authority."* The shipped set is **ten distinct** labels and
repeats none of them, and a candidate name is asked once per qtype — each qtype its own cache entry.
So the shape that produces the artefact is **unreachable inside a batch**, and across batches
ADR-0011 already forbids carrying the verdict.

§11.4 therefore stands at **8 of 14** and owes nothing. The two readings the ticket put in tension
are readings of two different probes, and the base rate is the one the product will run.

### 14.3 The skew fabricates, and that is why the parameter is one parameter

The failure a per-leaf setting would license, measured on the zone the corpus knows best.

**[measured]** direct to `ns-63.awsdns-07.com` (205.251.192.63), `s3.amazonaws.com`'s own delegated
authority, qtype A:

| Probe | Answer |
| --- | --- |
| 4 distinct random labels | `CNAME s3-1-w.amazonaws.com.` and **no A record**, plus a delegation referral — byte-identical across all four |
| 1 label, 4 repeats | the same, byte-identical |

**[measured]** the same zone, the same instant, through this vantage's recursive resolver:

| Probe | Answer |
| --- | --- |
| 3 distinct random labels | `CNAME s3-1-w.amazonaws.com.` → `CNAME s3-w.us-east-1.amazonaws.com.` → **eight A records, a different eight on every label** |

Read the components. Direct to the authority the `(A asked, A answered)` component is
**`NoSynthesis`**, which is a *determinate* reading and not an absent one; through the resolver it is
**`Indeterminate`**. §11.1 discriminates a candidate carrying *"an RRset where the control
determinately had none"* — so under a skewed pair (control at the authority, candidate at the
resolver) **every** fictional label beneath that parent differs at a determinate component, is
discriminated, and is recorded `Resolved` with eight fabricated addresses that cite `Address`es and
open `Service`s and `Endpoint`s.

The reverse skew is the safe direction and still wrong: `Indeterminate` at A, never consulted,
everything beneath `Shadowed`. §11.6's asymmetry decides which of the two would have shipped as the
disaster, and neither is acceptable — hence **one parameter, one value, one `Batch`**, and the rule
stated as an equivalence rather than as a pair of settings.

This also reproduces §13.5 at a second vantage and on a second day, with the attribution intact: the
rotation is a **separate delegation's**, spliced into the chain by the resolver, and the authority
for `s3.amazonaws.com` returns no address at all.

### 14.4 The path is invisible on ordinary zones — the affordability check

The strict direction is only affordable if declaring a path does not move honest verdicts.
**[measured]** four distinct random labels per zone, each run twice — through the resolver and direct
to the zone's own first delegated authority:

| Zone | Authority probed | Resolver | Direct | Agreement |
| --- | --- | --- | --- | --- |
| `github.io` | `ns-692.awsdns-22.net` | 1 set — the four `185.199.10x.153` | 1 set — the same four | **byte-identical** |
| `netlify.com` | `ns01.netlifydns.com` | 1 set — `18.208.88.157`, `98.84.224.111` | 1 set — the same pair | **byte-identical** |
| `railway.app` | `blue.foundationdns.org` | 1 set — `34.107.141.139` | 1 set — the same | **byte-identical** |

**3 of 3.** The path decides only on the pathological zones, which is the same result §12.5 got for
the structured label and for the same reason: a measurement that moves only where something is
already wrong is cheap to make strict.

### 14.5 The resolver is the position, and that is measured rather than analogous

Two sentences in the corpus look incompatible until the fact is placed. `CONTEXT.md`'s `Derivation`:
*a declared parameter … **none is ever operator-configurable***. §3.6: verge-asm *offers DoH only
for operators whose environment blocks outbound 53 or who distrust their local resolver* — an
operator choice.

Both stand, because they are two facts. The **kind of path** is authored data in the leaf. **Which
resolver stands at it** is a property of the `Vantage` — *a network position observations are made
from, declared as intent* — which sits outside every derivation, is already in the timeline key, and
is already what a `Batch` is *from one of*.

ADR-0025 reached this for the neighbouring case and did not generalise it: EDNS Client Subnet is *"a
`Vantage` in an option's clothes"*, and if v1 ever sends one *"ECS belongs in the **key**, never in
the scope record"*. A recursive resolver is ECS with the subnet implicit. The measurement makes it
concrete:

| `vercel.com` wildcard, A qtype | Address pool observed |
| --- | --- |
| §11.2, first vantage, 2026-08-14 | a closed pool of **8**, the `76.76.21.x` family — `api.vercel.com` at `76.76.21.112`, `www.vercel.com` drawing both its addresses from it |
| **This section**, second vantage, 2026-08-15 | `64.239.109.0/24` and `64.239.123.0/24` — **disjoint from the above** |

Two vantages, one name, one week, **two disjoint pools**. Not more rotation or less — different
content. A determinacy verdict, an address set and every `Address` it cites are all functions of
where the query appeared to come from, which is what the `vantage` component of the timeline key
exists for.

And the instability itself moves with the vantage, not only the content:

| `vercel.com`, one label × 8 repeats | Distinct answers | TTL reported |
| --- | --- | --- |
| Google Public DNS (§13.6) | **1** | 1800 |
| This vantage's host resolver | **7** | 1800 |

So *"repeating one label is annihilated by the cache the probe sits behind"* is true of that cache.
The lever ranking in §13.6 is unaffected — distinct labels weakly dominate at every mechanism and
are untouched by either resolver's behaviour — but the **degree** is a fact about the vantage, which
is the whole argument for declaring it.

### 14.6 Why the resolver and not the authority

The competing value has the better story on paper: no cache, no middlebox, the authority's own
words, and `resolution-walk` is already there for `Lame`. It loses four ways.

- **It cannot produce an address set across a zone cut without becoming a resolver.** **[measured]**
  above: at `s3.amazonaws.com`'s own authority a synthesised name is a CNAME into a zone that
  authority does not serve, returned with a referral. Following the chain means implementing
  recursion inside the leaf, with its own cache and its own answers —
  [#5](https://github.com/winniel123/verge-asm/issues/5)'s *every seam is a place drift can be
  manufactured*, and §3.6's rebuilt resolver.
- **Its errors are the silent ones.** `NoData` where the resolver has eight addresses is an absent
  `Address`, `Service`, `Endpoint` and `Reach` leg — ADR-0009's `{161}` in the direction that
  **under-reports** exposure.
- **It answers a question about a zone where the product's question is about an estate.** §3.1's own
  ground: the operator's resolver *"reflects what the operator's own network actually sees"*, and an
  address that network would never resolve to is not their exposure.
- **The sensitivity it buys is unnecessary**, per §14.2 — ten distinct labels through a resolver
  already read `Indeterminate` where it matters.

### 14.7 Where this is thin

- **One day, one second vantage, and that vantage's resolver is itself part of the finding.** It
  does not honour `vercel.com`'s 1800 s TTL across eight repeats where Google Public DNS did.
  Whether that is forwarding, an anycast pool or a shortened cache is **not isolated** — §13.2's
  design would isolate it and was not run, because the ruling turns on the two vantages disagreeing
  rather than on why.
- **The value is chosen on `{161}` and on §3.1, not on a counted comparison of estates.** Nobody has
  run a whole batch on each path and counted the `Address`es each admits. The under-report is argued
  from one zone's chain plus the structure of CNAME-across-a-cut, and how often that shape occurs in
  a small org's estate is **unmeasured** — the same hole §11.9, §12.8 and §13.10 each flagged.
- **A hijacking or filtering resolver shadows the whole estate**, and that is left as the loud
  failure rather than ticketed. A resolver that answers every name makes every parent read as
  wildcarded; the result is total suppression, which is the **honest** reading of an instrument that
  answers everything. §3.3's zone upload and §3.6's DoH fallback are the remedies and both are
  already first-class.
- **The resolver is recorded as declared intent**, so an upstream that moves under a stable
  declaration — *the host's own resolver*, changed by DHCP — moves answers with no key change and is
  invisible. Same class as `Vantage`'s existing *declared as intent* residue, not repaired here,
  because re-verifying a resolver's identity needs a probe whose answer is itself path-dependent.
- **No control probe has ever run inside a batch**, verbatim from ADR-0066, ADR-0068 and ADR-0069.
