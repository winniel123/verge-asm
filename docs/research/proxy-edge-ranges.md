# Published proxy-edge IP-range sources: what exists, how fresh

**Ticket:** wayfinder research [#940](https://github.com/winniel123/verge-asm/issues/940) — "Published
proxy-edge ranges: what exists, how fresh."
**Part of:** [#936](https://github.com/winniel123/verge-asm/issues/936).
**Feeds:** [#943](https://github.com/winniel123/verge-asm/issues/943) — the flagship list-vs-measurement
decision.
**Date of research:** 2026-08-31 (UTC). All endpoints below were live-fetched on that date.
**Status:** research complete. Every published list was fetched and its field names inspected at source.
Akamai is the one provider with no anonymous endpoint; its facts come from Akamai's own techdocs, not a live
fetch, and are labelled as such.

> **Provenance note.** While fetching the AWS documentation page, the research pass encountered embedded text
> urging it to run an `aws agent-toolkit search-skills` CLI command. That is injected third-party content, not
> part of this ticket, and it was disregarded. No such command was run. Every fact below was read at the
> provider's own URL.

---

## 0. Scope and method

The question, from [#940](https://github.com/winniel123/verge-asm/issues/940): for the major shared-edge
providers, what published IP-range sources exist, and — per provider — the authoritative endpoint, the format
and IPv4/IPv6 coverage, the update cadence and any change signal, the licensing for programmatic consumption,
and whether the list separates *proxy edge* ranges from other provider infrastructure (origin, storage,
control plane).

Eight providers are covered: Cloudflare, Fastly, AWS CloudFront, Akamai, Google Cloud, Azure Front Door, and
two high-value extras (Imperva/Incapsula, Bunny.net). Each was read at its first-party source. Where an
endpoint is public, it was fetched and its top-level JSON fields or plaintext shape were confirmed against the
live body; those facts are labelled **[measured]**. Facts taken from a provider's own documentation without a
matching live fetch (the whole of Akamai, and the SNS/notification mechanics) are labelled **[documented]**.

On **licensing**, the relevant bar is the `Source` **consent** property in
[`CONTEXT.md`](../../CONTEXT.md) and [ADR-0003](../adr/0003-third-party-source-consent-bar.md):
`unencumbered`, `operator-accepted`, `operator-credentialed`. ADR-0003's corollary — *"Absence of terms
clears the bar. A source that publishes no terms presents nothing to breach"* — bears directly on these
lists, which almost all publish anonymously with **no** stated redistribution license. Verdicts below use that
vocabulary.

---

## 1. Cloudflare

1. **Authoritative endpoints.** Landing page <https://www.cloudflare.com/ips/>; IPv4 plaintext
   <https://www.cloudflare.com/ips-v4/>; IPv6 plaintext <https://www.cloudflare.com/ips-v6/>; public JSON API
   (no auth) <https://api.cloudflare.com/client/v4/ips>, documented at
   <https://developers.cloudflare.com/api/resources/ips/methods/list/>.
2. **Format / coverage.** `ips-v4`/`ips-v6` are plaintext, one CIDR per line, `Content-Type:
   text/plain;charset=UTF-8` (first lines `173.245.48.0/20`, `103.21.244.0/22`…) **[measured]**. IPv4 and
   IPv6 are in **separate files**. The API returns JSON
   `{"result":{"ipv4_cidrs":[…],"ipv6_cidrs":[…],"etag":"…"},"success":true,…}` — both families in one
   response **[measured]**. `?networks=jdcloud` adds a `jdcloud_cidrs` field (JD Cloud partner network in
   China).
3. **Cadence / change signal.** No fixed schedule; changes are infrequent. The documented versioning signal is
   the API `etag` field — *"A digest of the IP data. Useful for determining if the data has changed"*
   (<https://developers.cloudflare.com/api/resources/ips/methods/list/>). No SNS/webhook notification.
4. **Licensing.** No license or terms stated on <https://www.cloudflare.com/ips/> or the API method page. The
   page calls itself *"the definitive source of Cloudflare's current IP ranges."* Treat as **no explicit
   programmatic-use license** (general Cloudflare site terms apply) → `unencumbered` under ADR-0003's
   absence-of-terms corollary.
5. **Edge separation.** Cloudflare is a pure reverse-proxy edge; the entire published set **is** the shared
   proxy edge — the IPs that front customer sites and connect to origins. There is no origin/storage/control-
   plane list to exclude. **Clean edge-only subset: the whole list.**

---

## 2. Fastly

1. **Authoritative endpoint.** <https://api.fastly.com/public-ip-list> (public, no token — **[measured]**),
   documented at <https://www.fastly.com/documentation/reference/api/utils/public-ip-list> (the
   `developer.fastly.com/reference/api/utils/public-ip-list/` path 308-redirects here).
2. **Format / coverage.** JSON, `Content-Type: application/json` **[measured]**. Two fields: **`addresses`**
   (IPv4 CIDR array) and **`ipv6_addresses`** (IPv6 CIDR array) — both families. Live body: ~19 IPv4 blocks
   (e.g. `23.235.32.0/20`, `151.101.0.0/16`) and 2 IPv6 blocks (`2a04:4e40::/32`, `2a04:4e42::/32`)
   **[measured]**.
3. **Cadence / change signal.** No fixed cadence. Docs state changes *"will be announced in advance as an 'IP
   address announcement'"* via the Fastly status page **[documented]**. **No ETag / Last-Modified caching
   signal** — the response header is `Cache-Control: no-store` **[measured]** — and no versioning token in the
   body, so a consumer must diff the content itself.
4. **Licensing.** No explicit programmatic-use license. Docs describe the list as *"exhaustive and includes
   all Fastly-owned IP ranges"* for *"client connections, log streaming reports, and origin connections"* →
   `unencumbered`.
5. **Edge separation.** **None.** Fastly states the list is all Fastly-owned ranges covering client-facing,
   log-streaming, and origin-facing traffic together. **No clean edge-only subset is extractable** from this
   endpoint.

---

## 3. AWS CloudFront

1. **Authoritative endpoint.** <https://ip-ranges.amazonaws.com/ip-ranges.json> (~2.5 MB, **[measured]**);
   docs <https://docs.aws.amazon.com/vpc/latest/userguide/aws-ip-ranges.html>. Supplementary RFC-8805
   geolocation feed <https://ip-ranges.amazonaws.com/geo-ip-feed.csv>.
2. **Format / coverage.** JSON. Top-level fields `syncToken`, `createDate`, `prefixes`, `ipv6_prefixes`
   **[measured]**. IPv4 entries under `prefixes` carry `ip_prefix`, `region`, `service`,
   `network_border_group`; IPv6 entries under `ipv6_prefixes` carry `ipv6_prefix`, `region`, `service`,
   `network_border_group`. Both families, in **separate arrays**. Note: BYOIP ranges are **not** in the file.
3. **Cadence / change signal.** Updated frequently (often several times a day). Live `syncToken:
   "1788175025"`, `createDate: "2026-08-31-11-17-05"` (syncToken is the Unix epoch of publication)
   **[measured]**. Response headers `ETag: "d735bf0cc6f91457081d7eacaf95b5bf"` and `Last-Modified` both
   present **[measured]**. Official change-notification: the Amazon SNS topic **`AmazonIpSpaceChanged`**,
   documented at <https://docs.aws.amazon.com/vpc/latest/userguide/subscribe-notifications.html> — subscribe
   (email/Lambda) for a push on every publish **[documented]**.
4. **Licensing.** No explicit license text. The one stated obligation for programmatic access:
   *"it is your responsibility to ensure that the application downloads the file only after successfully
   verifying the TLS certificate presented by the server"*
   (<https://docs.aws.amazon.com/vpc/latest/userguide/aws-ip-ranges.html>). General AWS Site Terms apply →
   `unencumbered`, with the TLS-verification obligation noted as an operational, not licensing, condition.
5. **Edge separation — AWS's strength, cleanly separable.** The **`service`** field partitions everything.
   Observed values include `AMAZON` (catch-all superset), `EC2`, `S3`, `CLOUDFRONT`,
   **`CLOUDFRONT_ORIGIN_FACING`**, `GLOBALACCELERATOR`, `ROUTE53*`, `API_GATEWAY`, `DYNAMODB` **[measured]**.
   - **`service=CLOUDFRONT`** — the full client-facing CloudFront POP edge (~292 tagged prefix references;
     IPv4 breakdown ~150 `region=GLOBAL` plus regional POP blocks: `us-east-1`×12, `us-west-2`×11,
     `eu-west-2`×8, `sa-east-1`×7, `eu-central-1`×6, …) **[measured]**.
   - **`service=CLOUDFRONT_ORIGIN_FACING`** — the subset CloudFront uses to pull **from your origin**; the set
     you allowlist on an origin security group.
   - `region` and `network_border_group` further scope each prefix (`GLOBAL`, `ap-northeast-1`,
     `eu-central-1`, …) **[measured]**.
   **Clean edge-only subset is fully extractable:** filter `prefixes`/`ipv6_prefixes` where
   `service == "CLOUDFRONT"` (client-facing) or `"CLOUDFRONT_ORIGIN_FACING"` (origin-pull). `GLOBALACCELERATOR`
   is the separate anycast-edge product, also isolable.

---

## 4. Akamai

1. **Authoritative source.** Akamai publishes **no** single anonymous JSON. Edge CIDR data comes through
   authenticated products/APIs **[documented]**:
   - **Firewall Rules Notification (CIDR blocks)** — <https://techdocs.akamai.com/firewall-rules/docs/about-cidr-blocks>
     — edge-server CIDR blocks per service, in Akamai Control Center / via API, with activation dates.
   - **Origin IP ACL** — <https://techdocs.akamai.com/origin-ip-acl/docs/welcome> — *"a small and stable
     list"* of Akamai edge IPs (CIDR) that connect **to your origin**.
   - **Site Shield** — a per-customer edge map retrieved via the authenticated Site Shield API.
2. **Format / coverage.** CIDR notation; columns include service, CIDR block, port, activation date, and
   status (Active / Future / Deleted). IPv4/IPv6 split is not stated on the overview pages (Akamai edge is
   predominantly IPv4, IPv6 available per service); the authoritative per-account answer is the authenticated
   CIDR list. No public first-party JSON URL exists to inspect field names **[documented]**.
3. **Cadence / change signal.** Change-driven, not scheduled. The notification model is by **activation
   date**: new/changed CIDR blocks are published in advance with a date by which firewall rules must be
   updated; the UI surfaces the most-recent blocks at the top with Future/Deleted status flags. Subscribers
   receive advance notice (email / Control Center) **[documented]**.
4. **Licensing — the one hard blocker.** Access is gated behind an Akamai contract and authentication and is
   governed by the customer's Akamai agreement, not a public open-data license. There is no public terms
   document for an anonymous consumer because there is no anonymous endpoint → `operator-credentialed` (an
   Akamai account is required before any range can be read).
5. **Edge separation.** Yes — organized per Akamai service/map. Origin IP ACL isolates the origin-facing edge
   subset specifically; Firewall Rules groups CIDRs by service. An edge / origin-facing subset **is**
   available, but only to authenticated customers.

---

## 5. Google Cloud (Cloud CDN / global forwarding / GCP)

1. **Authoritative endpoints.** Customer-usable external ranges
   <https://www.gstatic.com/ipranges/cloud.json>; all Google-owned ranges (superset)
   <https://www.gstatic.com/ipranges/goog.json>; canonical doc
   <https://support.google.com/a/answer/10026322>; Media CDN client-connectivity doc
   <https://docs.cloud.google.com/media-cdn/docs/client-connectivity>.
2. **Format / coverage.** JSON, `Content-Type: application/json` **[measured]**. Top-level `syncToken`,
   `creationTime`, `prefixes`; each `prefixes` entry has **`ipv4Prefix`** *or* **`ipv6Prefix`** (one per
   entry). `cloud.json` adds `service` (observed `Google Cloud`) and **`scope`** (region or `global`; 68
   distinct scopes incl. `us-central1`, `europe-west1`, `global`) **[measured]**. `goog.json` entries carry
   just the prefix field. Both families covered.
3. **Cadence / change signal.** No fixed public schedule. Body signal is `syncToken` + `creationTime`; HTTP
   `Last-Modified` is present, `Cache-Control: public, max-age=0`, **no ETag** **[measured]**. No first-party
   notification topic.
4. **Licensing.** No explicit programmatic-use license on the JSON. Google's guidance is simply to read
   `cloud.json` instead of the deprecated `_cloud-netblocks` DNS TXT record
   (<https://support.google.com/a/answer/10026322>). Google *documentation* is generally CC BY 4.0, but the IP
   *data* carries no stated license → `unencumbered`.
5. **Edge separation — partial.** `goog.json` minus `cloud.json` yields Google's own services
   (Search/Gmail/YouTube/global APIs). `cloud.json` is customer-facing GCP ranges scoped by `scope`
   (region/global) but is **not** tagged by product, so a "Cloud CDN edge" or "global-forwarding anycast
   front-end" subset **cannot** be isolated from general GCP compute ranges. Cloud CDN rides the shared Google
   front-end anycast ranges; Media CDN Edge Cache gets **dedicated per-service anycast IPs** read from your
   own resource config, not from a shared list. **A Google-vs-GCP split is extractable; a CDN-edge-only subset
   is not.**

---

## 6. Microsoft Azure Front Door / CDN

1. **Authoritative sources.** Downloadable JSON (Public cloud): confirmation page
   <https://www.microsoft.com/download/details.aspx?id=56519> → a dated file
   `download.microsoft.com/…/ServiceTags_Public_<YYYYMMDD>.json` (other clouds: US Gov `id=57063`,
   China/21Vianet `id=57062`, Germany `id=57064`). **Service Tag Discovery API** (authenticated,
   always-current): REST <https://learn.microsoft.com/rest/api/virtualnetwork/servicetags/list>,
   `Get-AzNetworkServiceTag` (PowerShell), `az network list-service-tags` (CLI). Overview
   <https://learn.microsoft.com/azure/virtual-network/service-tags-overview>.
2. **Format / coverage.** JSON: top-level `changeNumber`, `cloud`, `values[]`; each value has `name` (the
   service tag), `id`, and `properties` with `changeNumber`, `region`, `platform`, `systemService`, and
   **`addressPrefixes`** (CIDR array). Both IPv4 and IPv6 appear in `addressPrefixes`, but the **`AzureCloud`**
   tag *"doesn't include IPv6"* per the doc; IPv6 coverage varies by tag **[documented]**.
3. **Cadence / change signal.** **Published weekly** (typically Monday). Change signal is the **`changeNumber`**
   field: each subsection (e.g. `Storage.WestUS`) has its own `changeNumber`, and the top-level increments
   when any subsection changes. Operationally, *newly added IPs are not used by Azure for at least one week*
   after appearing (a grace window). The Discovery API can lag the JSON file by up to ~4 weeks for brand-new
   tags and needs auth + a subscription read role **[documented]**.
4. **Licensing.** Distributed via Microsoft Download Center under its terms; no separate open-data license is
   stated on the service-tags doc, and no programmatic-consumption restriction. The doc warns service tags
   alone are not sufficient for security → `unencumbered` (Download-Center / general site terms apply).
5. **Edge separation — cleanly separable via the tag `name`.** Front Door / CDN edge tags:
   - **`AzureFrontDoor.Frontend`** — *"the IP addresses that clients use to reach Front Door"* — the
     client-facing edge set.
   - **`AzureFrontDoor.Backend`** — the IPs Front Door uses to reach **your origins**; allowlist on the
     origin (origin-facing subset).
   - **`AzureFrontDoor.FirstParty`** / **`AzureFrontDoor.MicrosoftSecurity`** — reserved for select Microsoft
     services on Front Door.
   The broad `AzureCloud` tag (all datacenter public IPs, IPv4 only) is the superset to avoid for edge-only
   use. **Clean edge-only subset is fully extractable:** filter `values[]` where `name` starts with
   `AzureFrontDoor.` (`.Frontend` client-facing, `.Backend` origin-facing). Regional variants
   `AzureFrontDoor.Frontend.<region>` exist where supported.

---

## 7. High-value extras

### 7a. Imperva (Incapsula) — WAF / reverse-proxy edge

1. Endpoint: <https://my.imperva.com/api/integration/v1/ips> (HTTP **POST**, public, no auth — **[measured]**).
2. JSON. Fields **`ipRanges`** (IPv4 CIDR array) and **`ipv6Ranges`** (IPv6 CIDR array), plus `res` /
   `res_message` — both families **[measured]**. Live sample: `199.83.128.0/21`, `198.143.32.0/19`,
   `107.154.0.0/16`, …, `2a02:e980::/29`. Context doc <https://docs.imperva.com/> (Cloud WAF → allowlisting
   Imperva IPs on origin).
3. Cadence: not fixed; poll periodically. No documented ETag / notification.
4. Licensing: no explicit license stated on the API → `unencumbered`.
5. Edge separation: the whole list **is** the Imperva edge/proxy set (the IPs that reach your origin) — clean
   edge-only by construction; no origin/control-plane mixed in.

### 7b. Bunny.net — CDN edge

1. Endpoints (public, no auth — **[measured]**): JSON <https://bunnycdn.com/api/system/edgeserverlist> and
   <https://bunnycdn.com/api/system/edgeserverlist/IPv6>; plaintext `…/edgeserverlist/plain` and
   `…/edgeserverlist/IPv6/plain`. Docs <https://support.bunny.net/hc/en-us/articles/24155254055964>.
2. JSON array (or newline plaintext). IPv4 and IPv6 in **separate** endpoints — both live (IPv4 e.g.
   `89.187.188.227`; IPv6 e.g. `2400:52e0:1500::714:1`) **[measured]**.
3. Cadence: dynamic list — *"request it periodically to avoid missing newly added IPs."* No versioning token
   or notification documented.
4. Licensing: none stated; general Bunny.net terms → `unencumbered`.
5. Edge separation: the list **is** the shared CDN edge (PoP) set — clean edge-only by construction.

---

## 8. Summary matrix

| Provider | Primary URL | Format | IPv4 / IPv6 | Change signal | Clean edge-only subset? |
|---|---|---|---|---|---|
| Cloudflare | `cloudflare.com/ips-v4`,`-v6`; `api.cloudflare.com/client/v4/ips` | plaintext / JSON | both (separate files / one JSON) | `etag` field | **Yes** — whole list is edge |
| Fastly | `api.fastly.com/public-ip-list` | JSON (`addresses`,`ipv6_addresses`) | both | none (`Cache-Control: no-store`); status-page announcements | **No** — all Fastly IPs mixed |
| AWS CloudFront | `ip-ranges.amazonaws.com/ip-ranges.json` | JSON (`service`,`region`,`network_border_group`) | both (separate arrays) | `syncToken`/`createDate` + ETag/Last-Modified + SNS `AmazonIpSpaceChanged` | **Yes** — `service=CLOUDFRONT` / `CLOUDFRONT_ORIGIN_FACING` |
| Akamai | Firewall Rules / Origin IP ACL / Site Shield — **authenticated** | CIDR (Control Center / API) | per-service | activation dates + Future/Deleted status | **Yes** (Origin IP ACL) but auth-only |
| Google | `gstatic.com/ipranges/cloud.json` & `goog.json` | JSON (`ipv4Prefix`/`ipv6Prefix`,`scope`) | both | `syncToken`/`creationTime` + Last-Modified | **Partial** — Google-vs-GCP only; no CDN-edge tag |
| Azure Front Door | download `id=56519` JSON; Service Tag Discovery API | JSON (service tags, `addressPrefixes`) | both (`AzureCloud` is IPv4-only) | `changeNumber` (top + per-tag); weekly (~Mon) | **Yes** — `AzureFrontDoor.Frontend`/`.Backend` |
| Imperva | `my.imperva.com/api/integration/v1/ips` (POST) | JSON (`ipRanges`,`ipv6Ranges`) | both | none documented | **Yes** — whole list is edge |
| Bunny.net | `bunnycdn.com/api/system/edgeserverlist(/IPv6)(/plain)` | JSON / plaintext | both (separate endpoints) | none documented | **Yes** — whole list is edge |

---

## 9. Cross-cutting findings

**Licensing.** Only **Akamai** carries a hard access blocker: no public/anonymous endpoint — edge CIDRs
require an Akamai account and authenticated API, governed by the customer's contract (`operator-credentialed`).
Every other provider publishes anonymously, but **none states an explicit open-data / redistribution
license.** AWS's only stated programmatic obligation is TLS-certificate verification; Azure and Google ship
under Download-Center / general site terms; Cloudflare, Fastly, Imperva, and Bunny state no terms on the
endpoints themselves. Under ADR-0003's absence-of-terms corollary these clear the consent bar as
`unencumbered` for *consuming* the list — but if verge-asm intends to **redistribute or relicense** the range
data, the absence of an explicit grant is the risk to note, and is a question to ask the operator, not read.

**Change-tracking maturity, best to worst.**
- **Strongest:** AWS (SNS `AmazonIpSpaceChanged` push + `syncToken`/`createDate` + ETag/Last-Modified) and
  Azure (`changeNumber` per-tag and top-level, weekly cadence, one-week activation grace).
- **Middle:** Cloudflare (`etag` in the API body), Google (`syncToken`/`creationTime` + Last-Modified).
- **Weakest:** Fastly (`Cache-Control: no-store`, no token — announced only on the status page), Imperva, and
  Bunny (no versioning token or notification at all) — a consumer must diff the content itself.

**Edge-vs-infrastructure separation — the decision-relevant axis for [#943](https://github.com/winniel123/verge-asm/issues/943).**
- **Pure-edge providers** (Cloudflare, Imperva, Bunny) publish a list that *is* the proxy edge — no filtering
  needed.
- **Tagged multi-service providers** (AWS, Azure, Akamai) let you extract a clean client-facing edge subset
  *and* a distinct origin-facing subset: AWS `service=CLOUDFRONT` / `CLOUDFRONT_ORIGIN_FACING`, Azure
  `AzureFrontDoor.Frontend` / `.Backend`, Akamai Origin IP ACL.
- **Fastly and Google are the gap:** Fastly deliberately mixes client-, log-, and origin-facing IPs with no
  tag; Google's `cloud.json` is scoped by region but not by product, so CDN-edge IPs cannot be separated from
  general GCP compute. For these two, a published list cannot substitute for measurement if edge-only
  precision is required — which is exactly the list-vs-measurement question [#943](https://github.com/winniel123/verge-asm/issues/943)
  must weigh.
