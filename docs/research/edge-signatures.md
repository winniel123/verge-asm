# Measuring a shared foreign proxy edge

**Ticket:** wayfinder research #941 — "Can the system *measure* that an address is a *shared* foreign
proxy edge, rather than reading a provider list?"
**Date of research:** 2026-08-31 (UTC)
**Status:** research complete; no code exists yet. Feeds the boundary-measurement spec.

## 0. Scope, stance, and the discriminator

verge-asm has a load-bearing principle: it **measures** its estate boundary and **never reads it from a
list of providers**. A shipped list of "known CDN / proxy ASNs or cert hostnames" is exactly the failure
mode this ticket exists to avoid — it rots, it is incomplete for self-hosted estates run by strangers, and
it encodes someone else's opinion of the boundary instead of an observation of it.

The property we want to detect is **shared-ness**, defined precisely:

> One edge IP address that fronts **many unrelated registrable domains** — i.e. many distinct
> eTLD+1 names whose registrations do not belong to a single estate.

Providers describe this multi-tenant-IP model in their own words: Cloudflare says its edge IPs are
"shared by all proxied hostnames" (<https://developers.cloudflare.com/fundamentals/concepts/cloudflare-ip-addresses/>);
CloudFront documents "CloudFront can use SNI to host multiple distributions on the same IP"
(<https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cnames-https-dedicated-ip-or-sni.html>);
Fastly installs certificates "at a shared set of IP addresses … Because multiple certificates are served
off the same IP address pool, SNI is required" (<https://docs.fastly.com/products/platform-tls>).

The discriminator is **shared-ness, not ownership.** A cloud-resident estate is titled to third parties
*by design*: the address is owned by AWS / Cloudflare / Google, the TLS certificate may be issued to the
provider, and the reverse DNS may name the provider. None of that distinguishes "a shared proxy edge in
front of thousands of strangers" from "a dedicated host the operator legitimately rents in that cloud."
Ownership answers *who holds the IP*; it does not answer *is this edge shared among unrelated domains*.
Those are orthogonal. This document rates every candidate signal by that test.

**"Unrelated registrable domain" requires a reduction step.** A hostname seen on an edge
(`a.b.example.co.uk`) must be reduced to its registrable domain (`example.co.uk`) before it can be
counted, because the registrable boundary sits below the public suffix, which is *not* computable from
dot-counting. The only correct, keyless way to do this reduction is the **Public Suffix List**
(Mozilla, algorithm in the PSL spec). <https://publicsuffix.org/> and the matching algorithm at
<https://publicsuffix.org/list/>. The PSL is data we consume to compute a boundary, not a provider list;
it enumerates registry suffixes, not proxies.

### Reliability scale used below

Each signal is rated on **how well it measures shared-ness** (not how easy it is to observe):

- **High** — directly observes the shared-ness property; hard to fake in the direction that matters.
- **Medium** — correlates with shared-ness but also fires on non-shared edges, or misses shared edges;
  usable only as corroboration.
- **Low** — measures a different property (usually ownership or provider identity); using it as the
  discriminator reintroduces a provider list.

Summary verdict up front:

| Signal | Measures shared-ness? | Rating |
|---|---|---|
| 1. Fan-out: one IP, many unrelated registrable domains (TLS SAN + HTTP Host) | Directly | **High** |
| 2. Anycast (one IP answers from disparate vantages) | Correlate only | **Medium** |
| 3. Generic edge cert issuer / SAN pattern (`sni.cloudflaressl.com` …) | Provider identity | **Medium→Low** |
| 4. RDAP / whois ownership and ASN | Ownership, not shared-ness | **Low** |
| 5. Reverse DNS / PTR naming of edge pools | Operator-controlled label | **Low** |

Measurement of shared-ness without a provider list is **viable**, anchored on Signal 1, with Signal 2 as
corroboration. Signals 3–5 are enrichment/context and must not be the discriminator.

---

## 1. Fan-out — one IP presenting identities for many unrelated registrable domains

**Rating: High. This is the signal that actually measures shared-ness.**

### What it measures

A shared proxy edge, by construction, answers TLS and HTTP for many tenants. Two observable identity
channels expose those tenants:

- **TLS Server Name Indication (SNI) and certificate SANs.** SNI (RFC 6066 §3) lets a client name the
  host it wants during the handshake, which is precisely what lets one IP serve many certificates.
  <https://www.rfc-editor.org/rfc/rfc6066#section-3>. The certificate returned carries
  `subjectAltName` dNSName entries (X.509, RFC 5280 §4.2.1.6) — the set of hostnames that certificate is
  valid for. <https://www.rfc-editor.org/rfc/rfc5280#section-4.2.1.6>.
- **HTTP `Host` / `:authority`.** HTTP/1.1 requires a `Host` header (RFC 9110 §7.2) and HTTP/2/3 the
  `:authority` pseudo-header, which name-based virtual hosts key on to route a single IP to many origins.
  <https://www.rfc-editor.org/rfc/rfc9110#section-7.2>.

The measurement: for a candidate IP, collect the set of hostnames it presents, reduce each to its
registrable domain via the PSL, and count the **distinct, unrelated** registrable domains. A high count
(dozens to thousands of eTLD+1s that are not one estate) is a direct observation of shared-ness.

### How to collect the hostname set (cheap → expensive)

- **Passive, via Certificate Transparency.** CAs must publish every issued certificate to public,
  append-only CT logs, and each entry lists all hostnames it protects (CN + SANs). CT v1 is RFC 6962
  (<https://www.rfc-editor.org/rfc/rfc6962>), superseded by CT v2 RFC 9162
  (<https://www.rfc-editor.org/rfc/rfc9162.html>); Let's Encrypt documents the CA-side obligation and SAN
  listing (<https://letsencrypt.org/docs/ct-logs/>). A hostname→cert index such as crt.sh lets you
  enumerate certs (and their SANs) for a name; the reverse (certs seen on an IP) requires correlating
  resolved A/AAAA records. **CT records that a cert with names X,Y,Z was issued — it does not record which
  IP serves it**, so a shared-cert SAN bundle is strong evidence but must be bound to the edge IP by an
  active probe. CT is the cheapest channel and needs no traffic to the target.
- **Active TLS, no/varying SNI.** One TLS handshake with no SNI (or the IP as SNI) returns the edge's
  **default certificate**, whose SAN set is itself a fan-out sample on shared edges. Cost: one handshake
  per IP.
- **Active HTTP Host probing.** Confirms which known names actually resolve *and answer* on the IP.
  Cost: one request per (IP, host) pair — only for hosts you already have.

### Cost

Low-to-moderate. CT ingestion is passive and already in scope for verge-asm's discovery (see the passive
discovery research note). One extra TLS handshake and one HTTP request per candidate IP is cheap and
within a safe active-probing profile.

### False positives (says "shared" when it is not)

- **A single estate with many brands on one IP.** A company legitimately fronting `brand-a.com`,
  `brand-b.net`, … on one reverse proxy will show many registrable domains. The PSL cannot tell you those
  eTLD+1s are one owner. *Mitigation:* set the "unrelated" threshold high — a genuine single estate rarely
  fronts hundreds of unrelated registrable domains on one IP — and treat the count as a graded signal, not
  a boolean. Do **not** resolve the ambiguity with ownership lookups; that reintroduces Signal 4's flaw.
- **Stale CT SANs.** A certificate's SAN list records names valid at issuance; some may no longer resolve
  to that IP. Confirm with an active probe before counting when precision matters.

### False negatives (misses a genuinely shared edge)

- **SNI-required edges.** Many edges return no useful certificate (or a handshake failure) unless you
  already send the right SNI, so you only ever see the tenants you *already know* resolve there — the fan
  looks small.
- **Dedicated per-customer certificates.** An edge that issues each tenant its own single-name cert emits
  no shared SAN bundle; the fan-out is only visible by aggregating many IPs↔names observations over time.
  This is a deliberate product option on shared infra — Cloudflare Advanced Certificate Manager and
  CloudFront dedicated-IP custom certs both put one tenant's names on their own cert while the IP stays
  physically shared. <https://developers.cloudflare.com/ssl/edge-certificates/advanced-certificate-manager/> ;
  <https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cnames-https-dedicated-ip-or-sni.html>.
- **Encrypted ClientHello (ECH), TLS 1.3.** ECH encrypts the SNI, removing the plaintext hostname from
  the handshake and blunting active SNI enumeration; the underlying SNI-privacy problem is stated in
  RFC 8744 (<https://www.rfc-editor.org/rfc/rfc8744.html>), and ECH is defined over TLS 1.3,
  RFC 8446 (<https://www.rfc-editor.org/rfc/rfc8446>). Passive CT is unaffected.

### Fit with "measure the boundary"

**Best fit of any signal.** It observes the shared-ness property itself and needs no provider list — only
the PSL (a registry-suffix dataset) to compute registrable domains. This is the anchor signal.

---

## 2. Anycast — one IP answers from disparate geographic vantages

**Rating: Medium. Correlates with shared edges but measures the wrong property.**

### What it measures

Anycast advertises the same IP prefix from multiple locations so routing delivers a client to the nearest
instance. It is defined and operationally described in RFC 1546 (host anycasting),
<https://www.rfc-editor.org/rfc/rfc1546>; RFC 4786 / BCP 126 (operation of anycast services),
<https://www.rfc-editor.org/rfc/rfc4786>; and RFC 7094 (architectural considerations of IP anycast),
<https://www.rfc-editor.org/rfc/rfc7094>. Large shared edges (Cloudflare, Fastly) use anycast, so
"one IP appears to be in many places at once" correlates with a shared edge.

### How to measure

Probe the IP from several geographically dispersed vantage points and look for inconsistency that a single
unicast host cannot produce: RTT lower than the speed of light to a single location would allow from
multiple continents, divergent traceroute penultimate hops, or geolocation disagreement. RIPE Atlas is the
standard distributed measurement platform for this — thousands of globally distributed probes running
ping/traceroute (<https://atlas.ripe.net/docs/apis/rest-api-reference/measurements/>). RIPE's IPmap
formalises the detection: cluster observed RTTs into disjoint "speed-of-light bubbles" and set an anycast
flag from the cluster count (`numClusters`) (<https://ipmap.ripe.net/docs/01.manual/>;
method write-up <https://labs.ripe.net/author/kenneth_finnegan/measuring-anycast-dns-services-using-ripe-atlas/>).
Cloudflare and Fastly document that they announce the same IP from many locations
(<https://www.cloudflare.com/learning/cdn/glossary/anycast-network/>).

### Cost

**High relative to the others.** It requires distributed vantage points — either RIPE Atlas credits and
scheduling, or your own multi-region probes. A self-hosted verge-asm has one vantage by default, so this
signal is not available on first run without an external measurement network.

### False positives / negatives

- **FP: anycast ≠ shared.** The DNS root servers and many single-purpose services are anycast yet front a
  single logical service, not thousands of unrelated domains. A *single tenant* can even anycast its own
  prefix on provider infrastructure (Cloudflare "bring your own IP")
  (<https://developers.cloudflare.com/reference-architecture/diagrams/network/bring-your-own-ip-space-to-cloudflare/>).
  Anycast is *not sufficient* for shared-ness.
- **FN: shared ≠ anycast.** Not all shared edges are anycast. DNS-mapping CDNs historically steer clients
  by handing out **different** IPs per region (GeoDNS) rather than one anycast IP, so a genuinely shared
  edge can look perfectly unicast from every vantage. Anycast is *not necessary* either.

### Fit with "measure the boundary"

It is a real measurement, but of a property (routing topology) that only loosely tracks shared-ness. Use it
to **corroborate** a high Signal-1 fan-out, never as the discriminator.

---

## 3. Generic edge certificate issuer / SAN pattern

**Rating: Medium as a hint, Low as a discriminator — it slides into a provider list.**

### What it measures

Shared edges often serve a **default/shared certificate** with a recognisable identity. Cloudflare's
Universal SSL places `sni.cloudflaressl.com` in the certificate CN (with the customer hostname in the SAN)
under defined conditions <https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/>; the
CloudFront default certificate is the distribution's `*.cloudfront.net` name
<https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cnames-https-dedicated-ip-or-sni.html>;
Fastly serves off a shared IP pool selected by SNI <https://docs.fastly.com/products/platform-tls>. These
strings are observable in the returned certificate at zero extra cost once you already have the cert from
Signal 1.

### Cost

**Negligible** — the certificate is already in hand from Signal 1. No extra probe.

### False positives / negatives

- **FP / stance violation:** matching a *known string* like `cloudflaressl.com` is **reading a provider
  signature**, not measuring shared-ness. A hardcoded set of "generic issuer" patterns is a provider list
  wearing a disguise; it rots exactly as an ASN list does when providers rotate hostnames or issuers.
- **FN:** providers change default cert hostnames and issuers over time; edges that issue dedicated
  per-tenant certs emit no generic marker at all.

### The salvageable, stance-compatible reading

There is a **structural** fact inside the certificate that *is* a shared-ness measurement and does not
name any provider: a single default certificate whose SAN bundle spans **many unrelated registrable
domains** is itself fan-out (this is just Signal 1 read off one cert). Prefer that structural observation.
Treat the brand string (`sni.cloudflaressl.com` etc.) only as low-weight corroboration or human-readable
labelling — never as the boundary decision.

### Fit with "measure the boundary"

Structural SAN fan-out: fits (it *is* Signal 1). Brand-string matching: does **not** fit; it is a provider
list.

---

## 4. RDAP / whois ownership and ASN

**Rating: Low for shared-ness. Reliable and cheap as an *ownership* signal — but ownership is the wrong
question.**

### What it measures

RDAP maps an IP to its registered network object and holder. The query format is RFC 9082
(obsoletes RFC 7482) — `/ip/{addr}` for the covering network, `/autnum/{asn}` for an AS
(<https://www.rfc-editor.org/rfc/rfc9082>); the JSON response is RFC 9083 (obsoletes RFC 7483)
(<https://www.rfc-editor.org/rfc/rfc9083>); the IANA bootstrap that routes an IP to the correct RIR
server is RFC 9224 (<https://www.rfc-editor.org/rfc/rfc9224>), served from IANA's registries at
<https://data.iana.org/rdap/>. IP→ASN→org is likewise measurable keyless via RIR RDAP autnum objects
(origin ASN itself comes from BGP/route-collector data) or a service such as Team Cymru's IP-to-ASN. This
all works and is cheap.

### Why ownership alone is insufficient (the core caveat)

- **Cloud estates are titled to third parties by design.** Every address in a cloud-resident estate is
  *owned* by the cloud provider — Cloudflare's own IPs "are shared by all proxied hostnames," registered
  to Cloudflare, not to any fronted domain
  (<https://developers.cloudflare.com/fundamentals/concepts/cloudflare-ip-addresses/>). So "RDAP holder is
  a big provider" is true for the operator's own legitimate dedicated cloud host **and** for a shared proxy
  edge in front of thousands of strangers. Ownership cannot separate the two — it fires on the entire
  cloud, which is the whole estate.
- **Registration can even run the other way.** With "bring your own IP," an IP registered to a *customer
  org* is announced from and served by the provider's shared infrastructure
  (<https://developers.cloudflare.com/reference-architecture/diagrams/network/bring-your-own-ip-space-to-cloudflare/>) —
  the RDAP holder is the tenant, yet the edge is shared. Ownership and shared-ness are decoupled in both
  directions.
- **Shared-ness is orthogonal to who holds the IP.** A shared reverse proxy can run inside an operator's
  *own* ASN and prefix (self-hosted multi-tenant), and a *dedicated* single-tenant host can sit in a big
  provider's ASN. Neither the org name nor the ASN moves with the property we care about.
- **ASN→"known CDN" is the banned list.** The moment you map an ASN to a curated set of "these ASNs are
  proxies," you are reading a provider list. That is precisely the design smell this ticket rejects.

### Cost

Low. Keyless, standardised, fast.

### False positives / negatives (as a shared-ness signal)

- **FP:** every cloud-hosted asset looks "foreign-owned," so ownership over-selects massively.
- **FN:** self-hosted shared proxies in the operator's own space look "owned by us" and are missed.

### Fit with "measure the boundary"

**Poor as the discriminator.** Keep RDAP/ASN as *enrichment and context* attached to a finding (who to
contact, what network it sits in), never as the shared-ness decision.

---

## 5. Reverse DNS / PTR naming conventions of edge pools

**Rating: Low. An operator-controlled label, optional and pattern-matchy.**

### What it measures

A PTR record (type 12) in the `in-addr.arpa` (IPv4) or `ip6.arpa` (IPv6) reverse tree maps an IP back to a
name (RFC 1035 §3.5, <https://www.rfc-editor.org/rfc/rfc1035#section-3.5>; DNS tree structure RFC 1034,
<https://www.rfc-editor.org/rfc/rfc1034>; the `.arpa` infrastructure domain, RFC 3172,
<https://www.rfc-editor.org/rfc/rfc3172>; terminology in RFC 8499,
<https://www.rfc-editor.org/rfc/rfc8499>). RFC 1912 §2.1 *recommends* every IP have a matching PTR, but it
is Informational best practice, not a requirement (<https://www.rfc-editor.org/rfc/rfc1912>). Some CDN
pools use systematic PTR names; the idea is that a recognisable PTR pattern flags an edge pool.

### Cost

Low — one reverse lookup per IP.

### False positives / negatives

- **Operator-controlled and optional.** PTR is set by whoever controls the reverse zone. Cloudflare edge
  IPs frequently have **no** PTR at all; other pools set arbitrary or generic names. Absence is common and
  non-informative.
- **Spoofable / must be forward-confirmed.** A PTR name proves nothing unless the name resolves back to
  the same IP (Forward-Confirmed reverse DNS). FCrDNS is harder to fake because it needs control of both
  the forward and reverse zones — but "FCrDNS is not authentication": passing it proves control of both
  zones, not that an edge is shared. Even then it only reflects a label the network owner chose.
- **One name per IP, structurally.** PTR returns at most one name, so it can *never* enumerate the many
  unrelated domains a shared edge fronts — the opposite of Signal 1's SAN enumeration.
- **Pattern-matching PTR strings is a provider list.** Keying on `*.fastly.net` / `*.1e100.net` /
  provider-specific suffixes is the same rot as an ASN list.

### The salvageable reading

The *structural* observation — many **unrelated forward names** resolving to one IP that carries a generic
or absent PTR — is again Signal 1, not a PTR-pattern match. Prefer that.

### Fit with "measure the boundary"

Weak. Corroboration/labelling only; never the discriminator.

---

## 6. Verdict

- **Shared-ness is measurable without a provider list.** The anchor is **Signal 1**: enumerate the
  identities an edge IP presents (TLS SANs via CT and active handshake; HTTP `Host`), reduce each to its
  registrable domain with the **Public Suffix List**, and count distinct **unrelated** registrable
  domains. That count is a direct observation of shared-ness and needs no list of providers — only the
  registry-suffix dataset.
- **Corroborate with Signal 2 (anycast)** where a distributed vantage is available, understanding it is
  neither necessary nor sufficient on its own.
- **Signals 3–5 (generic cert brand strings, RDAP/ASN ownership, PTR patterns) are enrichment, not the
  discriminator.** Each, when used as a decision, degenerates into "read the boundary from a provider
  list" — the exact stance violation. Their only stance-compatible use is the *structural* fan-out already
  captured by Signal 1, or as human-readable context on a finding.
- **Ownership can never be the test.** Cloud estates are titled to third parties by design, so RDAP/ASN
  answers "who holds the IP," which is orthogonal to "is this edge shared among unrelated domains."

### Recommended measurement (keyless, first-run capable)

1. For each estate IP, gather presented hostnames: passively from CT SANs correlated by resolved address,
   plus one no-SNI TLS handshake for the default cert, plus HTTP `Host` confirmation for known names.
2. Reduce every hostname to eTLD+1 via the PSL.
3. Count distinct registrable domains; flag as a *shared proxy edge candidate* when the count of unrelated
   eTLD+1s crosses a graded threshold.
4. Attach RDAP/ASN and PTR as **context only**. Optionally schedule a RIPE Atlas anycast check to raise
   confidence. Never let steps in (4) override the step-3 measurement.

---

## Sources

Primary sources, grouped by signal. All URLs are the standard/first-party documents that own the claim.

- **Registrable-domain reduction:** Public Suffix List — <https://publicsuffix.org/> ;
  list + matching algorithm — <https://publicsuffix.org/list/>.
- **Signal 1 (fan-out):** RFC 6066 §3 SNI — <https://www.rfc-editor.org/rfc/rfc6066#section-3> ;
  RFC 5280 §4.2.1.6 subjectAltName — <https://www.rfc-editor.org/rfc/rfc5280#section-4.2.1.6> ;
  RFC 9110 §7.2 Host — <https://www.rfc-editor.org/rfc/rfc9110#section-7.2> ;
  RFC 6962 Certificate Transparency — <https://www.rfc-editor.org/rfc/rfc6962> ,
  superseded by RFC 9162 CT v2 — <https://www.rfc-editor.org/rfc/rfc9162.html> ;
  Let's Encrypt on CT / SAN listing — <https://letsencrypt.org/docs/ct-logs/> ;
  RFC 8446 TLS 1.3 — <https://www.rfc-editor.org/rfc/rfc8446> ;
  RFC 8744 SNI-encryption problem (ECH) — <https://www.rfc-editor.org/rfc/rfc8744.html> ;
  Cloudflare "IPs shared by all proxied hostnames" — <https://developers.cloudflare.com/fundamentals/concepts/cloudflare-ip-addresses/> ;
  CloudFront SNI multi-tenancy / dedicated-IP — <https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cnames-https-dedicated-ip-or-sni.html> ;
  Fastly Platform TLS (shared IP pool + SNI) — <https://docs.fastly.com/products/platform-tls> ;
  Cloudflare Advanced Certificate Manager (dedicated certs) — <https://developers.cloudflare.com/ssl/edge-certificates/advanced-certificate-manager/>.
- **Signal 2 (anycast):** RFC 1546 — <https://www.rfc-editor.org/rfc/rfc1546> ;
  RFC 4786 / BCP 126 — <https://www.rfc-editor.org/rfc/rfc4786> ;
  RFC 7094 — <https://www.rfc-editor.org/rfc/rfc7094> ;
  RIPE Atlas measurements API — <https://atlas.ripe.net/docs/apis/rest-api-reference/measurements/> ;
  RIPE IPmap anycast clustering — <https://ipmap.ripe.net/docs/01.manual/> ;
  RIPE Labs anycast-with-Atlas method — <https://labs.ripe.net/author/kenneth_finnegan/measuring-anycast-dns-services-using-ripe-atlas/> ;
  Cloudflare anycast glossary — <https://www.cloudflare.com/learning/cdn/glossary/anycast-network/> ;
  Cloudflare bring-your-own-IP (single-tenant anycast) — <https://developers.cloudflare.com/reference-architecture/diagrams/network/bring-your-own-ip-space-to-cloudflare/>.
- **Signal 3 (generic cert):** Cloudflare Universal SSL (`sni.cloudflaressl.com` CN) — <https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/> ;
  CloudFront default `*.cloudfront.net` cert — <https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cnames-https-dedicated-ip-or-sni.html> ;
  Fastly Platform TLS — <https://docs.fastly.com/products/platform-tls> ; certificate structure per RFC 5280 above.
- **Signal 4 (RDAP/ASN):** RFC 9082 RDAP query (obsoletes RFC 7482) — <https://www.rfc-editor.org/rfc/rfc9082> ;
  RFC 9083 RDAP JSON response (obsoletes RFC 7483) — <https://www.rfc-editor.org/rfc/rfc9083> ;
  RFC 9224 RDAP bootstrap — <https://www.rfc-editor.org/rfc/rfc9224> ;
  IANA RDAP bootstrap registries — <https://data.iana.org/rdap/> ;
  RIPE Database RDAP query docs — <https://docs.db.ripe.net/How-to-Query-the-RIPE-Database/Registration-Data-Access-Protocol/>.
- **Signal 5 (PTR):** RFC 1035 §3.5 in-addr.arpa — <https://www.rfc-editor.org/rfc/rfc1035#section-3.5> ;
  RFC 1034 DNS structure — <https://www.rfc-editor.org/rfc/rfc1034> ;
  RFC 3172 `.arpa` — <https://www.rfc-editor.org/rfc/rfc3172> ;
  RFC 1912 §2.1 PTR recommendation — <https://www.rfc-editor.org/rfc/rfc1912> ;
  RFC 8499 DNS terminology (FCrDNS) — <https://www.rfc-editor.org/rfc/rfc8499>.
