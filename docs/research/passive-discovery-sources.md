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
| RIPEstat Data API (`network-info`, `announced-prefixes`, `searchcomplete`) | Keyless, no registration under 1000 req/day, fast, fills ARIN's gaps (RIPE/APNIC/LACNIC/AFRINIC + BGP reality). **Caveat in §4.5 — non-commercial only.** |
| Common Crawl index (CDX) | Keyless, generous, open data licence. Weak signal but free. |

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
   Query 3–5 long random labels under **each name in the control-probe population** (e.g.
   `<random32>.dev.example.com`) for the declared qtype set.
2. If they answer, record the wildcard answer set as a **poison signature**.
3. ~~Repeat one level down for each discovered sub-zone — wildcards can exist at any label depth.~~
   No repetition and no depth walk: the parent population already covers every held name at every
   depth.
4. Suppress (or flag as unverifiable) any candidate whose answer matches the poison signature.
   **The match predicate is unspecified and is not set equality** — **[measured]** 2026-08-14,
   `herokuapp.com` returned three distinct synthesised address sets across five control labels and
   `vercel.com` returned five, seconds apart from one vantage, while `github.io` and `localtest.me`
   held still. Open as [#111](https://github.com/winniel123/verge-asm/issues/111).
5. Note the RFC 4592 escape hatch: a name that resolves to something *different* from the wildcard
   signature is genuinely present even under a wildcard, because an exact match blocks synthesis.
6. Where the control probe under a name's parent **did not complete**, that name records a **`Gap`**
   and never a value — *an undiscriminated answer is never a value* (ADR-0066). A probe that
   completed and found no wildcard licenses everything beneath it.

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

That chain — **org name → ASNs → announced prefixes → per-IP reverse lookup** — is the complete
"from a company name to the IP ranges it actually announces" path, keyless, in three requests. It covers all
five RIRs and, crucially, reflects **BGP reality** rather than registry paperwork, so it catches space the
org announces but has not tidily registered.

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
RIPEstat data — so the re-packaging clause is not triggered by the software itself. **Default-on is
defensible for the self-hosted, non-commercial case**, with a low concurrency cap (≤8, per the docs), a
per-run request budget aimed at staying under 1000/day, and an identifying User-Agent. **But:** anyone
operating verge-asm as a paid service needs RIPE NCC's written permission, and the README must say so.
This is the one Tier-1 default with a genuine ToS asterisk — if the project wants zero contractual
ambiguity in defaults, demote RIPEstat to Tier 2 and accept weaker RIR coverage out of the box.

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
| Wildcard detection (RFC 4592) | Poison signature for ~~a zone~~ **a name's child space** — the population is *the parents of the names in scope*, per §3.2 / [ADR-0066](../adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md) | — | Yes | n/a | Excellent | None | **Tier 0 (mandatory)** |
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
- Should verge-asm ship a bundled, periodically-refreshed IP→ASN table (from RIPE RIS / RouteViews open
  data) to remove the RIPEstat dependency from the default path entirely?
- What is the correct default posture for NSEC3 zones — attempt a small dictionary, or skip silently?
- Does the "free-fall risk" feature need registrar-expiry monitoring for domains outside the operator's
  supplied list, and if so how are those domains discovered without CZDS or reverse-WHOIS?
