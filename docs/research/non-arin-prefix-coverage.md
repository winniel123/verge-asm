# Keyless org-name to prefix coverage outside ARIN

**Ticket:** wayfinder research #20 — "Which keyless, no-account data sources take an organisation name to the
IP prefixes that organisation holds or announces across the RIPE, APNIC, LACNIC and AFRINIC regions, and which
of them clear the modal-operator consent bar?"
**Extends:** [#3 / `docs/research/passive-discovery-sources.md`](https://github.com/winniel123/verge-asm/blob/research/passive-discovery-sources/docs/research/passive-discovery-sources.md)
**Consent bar from:** [#15](https://github.com/winniel123/verge-asm/issues/15) / [ADR-0003](../adr/0003-third-party-source-consent-bar.md)
**Date of research:** 2026-08-13 (UTC)
**Status:** research complete. Four of the candidates need an email before a verdict exists; those are listed
in §12 as follow-on `task` tickets, not resolved here.

---

## 0. Scope and method

[#3](https://github.com/winniel123/verge-asm/issues/3) established that ARIN's RDAP `entities?fn=` →
`entity/<handle>` → `networks` path turns a free-text organisation name into that organisation's complete
registered footprint — **202 CIDRs in two keyless requests** — and that this is North America only.
[#15](https://github.com/winniel123/verge-asm/issues/15) has moved RIPEstat, which filled the other four
regions, off by default pending [#19](https://github.com/winniel123/verge-asm/issues/19). This document asks
whether the other four regions can be covered without it.

Every candidate is evaluated against three questions, in this order:

1. **Does the path exist at all, measured?** RDAP search extensions are optional (RFC 9082: *"Server
   implementations are free to support only a subset of these features … Servers MUST return an HTTP 501
   (Not Implemented) response to inform clients of unsupported query types"*
   <https://www.rfc-editor.org/rfc/rfc9082.html>), so support is measured per registry, never inferred from
   the standard.
2. **Limb 1 — the software's inherent behaviour.** Per
   [ADR-0003](../adr/0003-third-party-source-consent-bar.md): *"automated querying, storing results in a
   database, retaining them across runs"* must be permitted, and *"a clause forbidding storage in another
   retrieval system, or copying a significant part of a database, fails here regardless of who the operator
   is."*
3. **Limb 2 — the modal operator.** *"A small commercial organisation inventorying its own estate, doing
   exactly what verge-asm does"* must be inside the terms. *"'Personal', 'non-commercial' or 'evaluation' use
   only fails. A clause barring resale or redistribution of the source's data does not fail, because the
   operator is not doing that."*

**Verdicts** use the `Source` **consent** property defined in [`CONTEXT.md`](../../CONTEXT.md) —
`unencumbered`, `operator-accepted`, `operator-credentialed` — plus `ambiguous — needs an email` and
`unusable` for candidates that are not yet, or never will be, a shipped `Source` at all.

ADR-0003's two corollaries do real work in §8 and are quoted where they bite:

> *"**Ambiguity is asked about, not read.** Where terms are genuinely in tension, the project writes to the
> source operator and ships the source **off by default until an answer arrives** — indefinitely if none does.
> Silence is not consent…"*
>
> *"**Absence of terms clears the bar.** A source that publishes no terms presents nothing to breach. Its
> risks are operational and are governed as operational risks, not laundered through a compliance rule."*

So a verdict of `ambiguous — needs an email` here means concretely: **ships off by default, available as
`operator-accepted`, until the source operator answers.** It is not a rejection and it is not a deferral of
the work — the capability is one click away for any operator willing to accept the terms themselves.

All live measurements were taken from a single commercial IP on 2026-08-13 UTC and are reproducible with
`curl` (8.21.0). Port-43 whois was driven with `printf '<query>\r\n' | curl telnet://<host>:43`. Measurements
are labelled **[measured]**. Elapsed times are `%{time_total}`; sizes are `%{size_download}`. Every quoted
clause below was retrieved and read at its own URL; none is quoted from a secondary write-up.

---

## 1. The answer, first

**Two of the four non-ARIN regions have a keyless org-name → prefix path that clears both limbs today. Two do
not, and closing them requires an email, not more research.**

| Region | Best keyless org-name → prefix path | Requests | Consent verdict |
|---|---|---|---|
| **AFRINIC** | CAIDA `as-org2info` ⋈ `delegated-afrinic-extended-latest` (opaque-id join) | 2 file fetches, cached | **`unencumbered`** |
| **APNIC** | CAIDA `as-org2info` ⋈ `delegated-apnic-extended-latest` (opaque-id join) | 2 file fetches, cached | **`unencumbered`** |
| **RIPE** | RIPE DB REST `type-filter=organisation` → RDAP `entity/<ORG-…>` → `networks` | 2 | **`ambiguous — needs an email`** |
| **LACNIC** | LACNIC RDAP `entities?fn=<name>*` → `entity/<handle>` → `networks` | 2 | **`ambiguous — needs an email`** |

Each region also has a *live registry* path that is faster, fresher and more complete than the offline join —
AFRINIC's is the best org→prefix endpoint of any RIR, better than ARIN's — but every one of those live paths
sits under registry terms that cannot be cleared by reading. See §8. Per ADR-0003 all four ship **off by
default, as `operator-accepted`**, until their registry answers; what is lost is first-run depth, not the
capability.

This is a direct improvement on ADR-0003's stated consequence that *"ARIN's `entities?fn=` org-name path
carries the keyless default set and covers North America only."* On these measurements the keyless default
set can carry **three** regions — North America, Africa and Asia-Pacific — with nobody's permission, once the
CAIDA + delegated-stats path ships.

~~**BGP reality** — space an organisation announces but has not tidily registered — is covered in all four
regions by **RouteViews** under CC BY 4.0, `unencumbered`. But RouteViews is ASN-keyed, so it only helps once
you already have the ASN, and the ASN comes from the same org-name lookup above.~~

> **Withdrawn 2026-08-15 by [#126](https://github.com/winniel123/verge-asm/issues/126) /
> [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md), at the site that
> specifies it.** The consent verdict on RouteViews is untouched and correct — CC BY 4.0,
> `unencumbered`. What is withdrawn is *"BGP reality … is covered"*, which reads the announced set as
> a set of the operator's addresses. **A routing announcement names who carries packets toward a
> prefix, never who controls what listens in it.** The BGP leg does not ship in v1; see §14. (The
> second sentence is also incomplete on its own terms: [#27](https://github.com/winniel123/verge-asm/issues/27)
> later found the ASN is recoverable from the delegated file's own opaque-id grouping, not only from
> the org-name lookup. That correction is now moot here.)

---

## 2. RDAP entity search, registry by registry

The ARIN-equivalent question is whether `GET /entities?fn=<org name>` works. **[measured]** it behaves
differently at all four registries, and three of the four differences are undocumented.

| Registry | `entities?fn=` | Prefix-match syntax | `networks` in search result? |
|---|---|---|---|
| RIPE | **HTTP 500** | — | — |
| APNIC | HTTP 200, **always empty** | — | — |
| LACNIC | HTTP 200 | **requires trailing `*`** | No — second fetch needed |
| AFRINIC | HTTP 200 | **substring; `*` is literal** | **Yes — inline** |

### 2.1 RIPE — entity search returns 500

**[measured]**

| Query | Result |
|---|---|
| `https://rdap.db.ripe.net/entities?fn=Cloudflare*` | **500** in 0.42 s, 703 B |
| `https://rdap.db.ripe.net/entities?handle=ORG-MBL5-RIPE` | 200 in 0.40 s, 3,643 B |

The 500 body is a well-formed RDAP error object, which is worth noting because a client that only checks for
JSON-parseability will not notice the failure:

```json
{"rdapConformance":["cidr0","rdap_level_0","nro_rdap_profile_0","redacted"],
 "notices":[{"title":"Terms and Conditions", …}],
 "port43":"whois.ripe.net","errorCode":500,"title":"Internal Server Error"}
```

`entities?handle=` *does* work, but it is useless for this question: it takes the handle you were trying to
find, and **[measured]** the single result it returns carries no `networks` and no `autnums`. This is a handle
echo, not a resource search.

**The RIPE entity *fetch* is a different story and does work.** **[measured]**
`https://rdap.db.ripe.net/entity/ORG-MBL5-RIPE` → **200 in 0.81 s, 150,259 B**, carrying **19 networks and
4 autnums**. So RIPE's gap is precisely and only *name → handle*, and §4 closes it.

### 2.2 APNIC — search is implemented as a no-op stub, which is worse than a 501

**[measured]** Every query form returns **HTTP 200** in 0.13–0.19 s with `"entitySearchResults":[]` and an
835–853 B body that is entirely `notices` boilerplate — including `fn=Telstra*`, `fn=Telstra Limited`,
`fn=Cloudflare*`, `fn=Google*`, `fn=APNIC*`, `fn=*Telstra*`, `fn=a`, and by handle `handle=TA166-AP`.

The same handle fetched directly works: **[measured]** `https://rdap.apnic.net/entity/TA166-AP` → 200 in
0.15 s, 1,500 B. So the objects exist; the search index does not.

**This is the single most dangerous finding in this document for client implementation.** RFC 9082 requires a
501 for unsupported query types. APNIC returns 200 with an empty result set, which is indistinguishable from
"this organisation holds no APNIC resources". A client that trusts it will silently report an empty APNIC
footprint for every organisation on Earth — and in a drift product, on the run after an organisation's APNIC
space is first discovered by some other means, it will report that space as **removed**.

`https://rdap.apnic.net/help` (**[measured]** 200, 802 B) advertises no search capability and no `501`
behaviour; it is `notices` only. There is no way to detect this from the protocol.

### 2.3 LACNIC — works, non-standard response key, requires an explicit wildcard

**[measured]**

| Query | Result |
|---|---|
| `https://rdap.lacnic.net/rdap/entities?fn=Cloudflare*` | 200 in 0.84 s, 10,697 B — **8 entities** |
| `https://rdap.lacnic.net/rdap/entities?fn=Cloudflare` | 200 in 1.19 s, 2,037 B — **1 entity** (exact match only) |

Two client-visible deviations:

1. **The results come back under `"entities"`, not `"entitySearchResults"`.** RFC 9083 §8 names the member
   `entitySearchResults`; LACNIC uses `entities`. A conformant client reads zero results.
   <https://www.rfc-editor.org/rfc/rfc9083.html>
2. **No implicit prefix matching.** `fn=Cloudflare` returns only the entity whose `fn` is exactly
   `CLOUDFLARE`. The trailing `*` is required, and case is insensitive.

The search result carries `links`, `objectClassName`, `handle` and `vcardArray` — **no `networks`**. A second
fetch is needed, and it delivers. **[measured]**
`https://rdap.lacnic.net/rdap/entity/CR-CLAS1-LACNIC` → 200 in 0.46 s, 7,125 B, **3 networks**:
`131.0.72.0/22`, `190.93.240.0/20`, `2803:f800::/32`.

So LACNIC is a working ARIN-shaped two-request path. It is the only registry of the four whose *shape* matches
ARIN's exactly.

### 2.4 AFRINIC — the best org→prefix endpoint of any RIR, in one request

**[measured]** AFRINIC does substring matching on `fn` and treats `*` as a literal character, which is the
inverse of LACNIC and will silently return nothing for a client that assumes the ARIN convention:

| Query | Result |
|---|---|
| `https://rdap.afrinic.net/rdap/entities?fn=AFRINIC*` | 200 in 0.85 s, **1,998 B — 0 results** |
| `https://rdap.afrinic.net/rdap/entities?fn=AFRINIC` | 200 in 3.32 s, **379,787 B — 106 results** |
| `https://rdap.afrinic.net/rdap/entities?fn=Safaricom` | 200 in 2.75 s, **195,856 B — 19 results** |
| `https://rdap.afrinic.net/rdap/entities?fn=Internet Solutions` | 200 in 5.66 s, 291,178 B — 51 results |
| `https://rdap.afrinic.net/rdap/entities?fn=MTN` | 200 in 16.05 s, **1,418,557 B** |

**And the search results embed `networks` and `autnums` inline.** **[measured]** the `fn=Safaricom` response
alone yields, with no second request:

```
ORG-SL70-AFRINIC   "Safaricom Limited"                          13 networks, 2 autnums
                     105.160.0.0 - 105.167.255.255, 105.48.0.0 - 105.63.255.255,
                     196.201.208.0 - 196.201.223.255, …
ORG-STEP1-AFRINIC  "SAFARICOM TELECOMMUNICATIONS ETHIOPIA PLC"   5 networks, 1 autnum
                     102.203.224.0 - 102.203.227.255, 102.208.96.0 - 102.208.99.255, …
```

Multi-word queries work (`fn=Internet Solutions` → 51 entities). Person entities in the same result set carry
no populated `networks` array, so filtering `ORG-*` handles with a non-empty `networks` array is a one-line
client rule.

**[measured]** the entity fetch is equally rich: `https://rdap.afrinic.net/rdap/entity/ORG-AFNC1-AFRINIC` →
200 in 1.66 s, 110,762 B, with **27 networks** (`102.0.0.0/8`, `105.0.0.0/8`, `196.1.0.0/24`, …) and
**5 autnums** (AS33764, AS37177, AS37181, AS37301, AS37708).

Functionally this beats ARIN's two-request path. The cost is latency and payload: 0.9–18.5 s per query, and
`fn=MTN` returns 1.4 MB because the substring match is greedy and unbounded — there is no `count`, no
pagination and no truncation notice observed.

**[measured] Rate limiting: none observed.** 12 consecutive `fn=Safaricom` requests with no delay all returned
200 at 1.91–4.15 s, byte-identical 195,856 B responses.

---

## 3. Port-43 whois — the path APNIC's RDAP does not have

RIPE, APNIC and AFRINIC all run RIPE-derived whois software supporting `-T <type>` and `-i <attribute>`
inverse queries. LACNIC does not.

**[measured] APNIC — a complete keyless org-name → prefix path exists on port 43:**

```
-r -T organisation Telstra        → 2,986 B in 0.38 s
    organisation: ORG-TC6-AP    org-name: Telstra Limited               org-type: LIR    country: AU
    organisation: ORG-TCL23-AP  org-name: Telstra Corporation Limited   org-type: OTHER  country: AU
    organisation: ORG-TIL3-AP   org-name: Telstra International Limited org-type: LIR    country: HK
    organisation: ORG-TIPL8-AP  org-name: TELSTRA IVISION PTY LTD                        country: AU
    organisation: ORG-TNT2-AP   …

-r -i org ORG-TC6-AP              → 88,479 B in 0.37 s
    95 inetnum, 3 inet6num, 14 aut-num objects
```

**[measured]** 10 consecutive `-r -T organisation Telstra` queries: all succeeded, 0.27–0.38 s each,
byte-identical. No throttling.

Every response is prefixed with the copyright pointer, which is where §8.1's terms come from:

```
% [whois.apnic.net]
% Whois data copyright terms    http://www.apnic.net/db/dbcopyright.html
```

**[measured] AFRINIC** — same shape: `-i org ORG-SL70-AFRINIC` → **874,628 B in 3.47 s**, returning
`inetnum:` objects with `netname: safaricom-2012` / `descr: Safaricom Limited`. Free-text `Safaricom` also
matches directly (16,425 B in 1.34 s). Banner: *"The AFRINIC whois database is subject to the following terms
of Use. See https://afrinic.net/whois/terms"* — a URL which **[measured]** returns **HTTP 404** (§8.2).

**[measured] RIPE** — `-r -i org ORG-CI40-RIPE` → 4,376 B in 0.45 s, returning `inetnum: 141.101.64.0 -
141.101.127.255` etc. Banner points at `https://docs.db.ripe.net/terms-conditions.html`.

**[measured] LACNIC — no org search on port 43, and it says so:**

```
% Joint Whois - whois.lacnic.net
%  This server accepts single ASN, IPv4 or IPv6 queries
…
% No match for "CLOUDFLARE"
% whois.lacnic.net accepts only direct match queries.
% Types of queries are: POCs, ownerid, CIDR blocks, IP and AS numbers.
```

For LACNIC the RDAP path in §2.3 is the only one.

---

## 4. RIPE Database REST search API

**[measured]** `type-filter=organisation` is the working name→handle step, and it is the one #3 did not test:

| Query | Result |
|---|---|
| `rest.db.ripe.net/search?query-string=Mythic%20Beasts&type-filter=organisation&flags=no-filtering` | 200 in 0.38 s, 3,477 B — **1 object, `ORG-MBL5-RIPE` / `Mythic Beasts Ltd`** |
| `…query-string=cloudflare&type-filter=organisation…` | 200 in 0.55 s, 16,796 B — 8 objects incl. `ORG-CI40-RIPE` / `Cloudflare Inc` |
| `…query-string=cloudflare&type-filter=inetnum…` | 200 in 0.74 s, 112,137 B (reproduces #3's result) |
| `…query-string=Mythic%20Beasts&type-filter=inetnum…` | **HTTP 400** in 0.39 s |
| `…query-string=Mythic&type-filter=inetnum…` | **HTTP 404** in 0.39 s |
| `…query-string=Mythic%20Beasts&type-filter=aut-num…` | **HTTP 400** in 0.39 s |

**The `type-filter=inetnum` path that #3 measured only works for single-token organisation names.** A
multi-word name returns:

```json
{"errormessages":{"errormessage":[{"severity":"Error",
  "text":"ERROR:115: invalid search key\n\nSearch key entered is not valid for the specified object type(s)\n"}]},
 "terms-and-conditions":{"type":"locator","href":"https://docs.db.ripe.net/terms-conditions.html"}}
```

That is most organisations. `type-filter=organisation` has no such restriction, so **the correct RIPE chain is
`type-filter=organisation` → RDAP `entity/<ORG-…>`**, not the inetnum search.

**[measured] end to end for a genuinely small operator, Mythic Beasts Ltd (UK hosting company):**

1. `rest.db.ripe.net/search?query-string=Mythic%20Beasts&type-filter=organisation&flags=no-filtering` →
   `ORG-MBL5-RIPE` (0.38 s)
2. `rdap.db.ripe.net/entity/ORG-MBL5-RIPE` → 200 in 0.81 s, 150,259 B, **19 networks (20 CIDRs) and 4
   autnums**: `2001:678:2e0::/48`, `2001:678:634::/48`, `2a00:1098::/32`, `2a04:ad80::/29`, `2a06:1c80::/29`,
   `2a07:1500::/29`, `36.255.148.0/22`, `45.139.80.0/22`, `46.235.224.0/21`, `93.93.128.0/21`,
   `103.105.48.0/22`, `103.209.164.0/22`, `176.126.240.0/21`, `185.47.60.0/22`, `185.101.96.0/22`,
   `185.139.144.0/22`, `193.227.244.0/23`, `195.10.223.0/24`, `195.191.54.0/23`, `195.191.56.0/23`;
   AS60011, AS44684, AS198557, AS204345.

**Two client hazards, both [measured]:**

- **`Accept: application/rdap+json` gets HTTP 406 with a zero-byte body.** No error object, no message, 0.42 s.
  A client sharing a header set between the REST and RDAP endpoints fails silently.
- **Omitting `Accept` entirely returns 200 with 12,891 B of XML**, not JSON. `Accept: application/json` must be
  sent explicitly.

**[measured] Rate limiting: none.** 12 consecutive organisation searches, all 200, 0.38–0.55 s. The AUP's only
hard structural limit is *"Number of simultaneous connections to the RIPE Database server – 3"* (§8.4).

---

## 5. Delegated extended statistics — and the join that makes them answer the question

### 5.1 What the files are

**[measured]** all four fetch keylessly, HTTP 200:

| File | Size | Elapsed |
|---|---|---|
| `https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest` | 987,806 B | 4.12 s |
| `https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest` | 4,546,193 B | 1.39 s |
| `https://ftp.apnic.net/stats/apnic/delegated-apnic-extended-latest` | 9,194,052 B | 3.52 s |
| `https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest` | 18,020,508 B | 2.52 s |

The 8th field is the one that matters. Per the NRO format specification
<https://ftp.apnic.net/stats/apnic/README-EXTENDED.TXT>:

> `opaque-id  This is an in-series identifier which uniquely identifies a single organisation, an Internet
> number resource holder.`
>
> `All records in the file with the same opaque-id are registered to the same resource holder.`
>
> `The opaque-id is not guaranteed to be constant between versions of the file.`

**[measured]** the four registries use four incompatible opaque-id formats:

```
ripencc|PS|ipv4|1.178.112.0|4096|20071126|allocated|6c170944-abac-4543-8948-6130b9a05faf   (UUID)
apnic  |AU|ipv4|1.120.0.0|524288|20100518|allocated|A916A983                              (8 hex)
lacnic |GT|ipv4|2.152.0.0|1024|20260714|allocated|71316                                   (integer)
afrinic|ZA|asn |1228|1|19910301|allocated|F36B9F4B                                        (8 hex)
```

**The organisation name is not in these files and is not recoverable from them alone.** The ticket asked
whether the handle resolves to a name keylessly; it does not — the ids are opaque by design and, per the spec
above, not even stable across versions. On their own the delegated files answer "which prefixes share a
holder with this prefix", never "which prefixes belong to Acme Ltd".

### 5.2 The join that does answer it — CAIDA `as-org2info` ⋈ delegated stats

CAIDA's AS Organizations dataset publishes exactly the missing column. **[measured]**
`https://publicdata.caida.org/datasets/as-organizations/latest.as-org2info.jsonl.gz` → 200 in 1.22 s,
**4,367,041 B**, 220,585 JSONL records: **98,597 `Organization` records and 121,988 `ASN` records**. Each
`ASN` record carries `organizationId`, `name`, `source`, and — crucially — **`opaqueId`**, described in the
dataset README as *"opaque identifier used by RIR extended delegation format"*
<https://publicdata.caida.org/datasets/as-organizations/README.txt>.

**[measured] the join works.** Two worked examples, org name in, prefixes out, no key and no account:

```
"Safaricom"  → CAIDA orgs ORG-SL70-AFRINIC ("Safaricom Limited", KE), ORG-STEP1-AFRINIC (ET)
             → ASNs 33771, 37061 (opaqueId F3682104_AFRINIC), 328988 (F36CA351_AFRINIC)
             → grep F3682104 delegated-afrinic-extended-latest = 14 records:
               2 ASNs + 10 IPv4 blocks (41.80.0.0/15, 41.90.0.0/16, 41.139.128.0/17,
               41.203.208.0/20, 105.48.0.0/12, 105.160.0.0/13, 196.96.0.0/12,
               196.201.208.0/20, 197.176.0.0/13, 197.248.0.0/16) + 2 IPv6 (2001:43d0::/32, 2c0f:fe38::/32)

"Telstra Limited" → CAIDA org ORG-TC6-AP-APNIC → opaqueId A916A983_APNIC (16 ASNs share it)
             → grep A916A983 delegated-apnic-extended-latest = 143 records
```

Cross-checked against §2.4: the AFRINIC RDAP live query returned 13 networks for `ORG-SL70-AFRINIC`; the
offline join returned 12 (10 IPv4 + 2 IPv6) for the same holder. The join is faithful.

### 5.3 The join's coverage is exactly two regions, and this is the crux

**[measured]** `opaqueId` presence in `latest.as-org2info.jsonl.gz`, by source registry:

| Source | ASN records | With `opaqueId` | Coverage |
|---|---|---|---|
| ARIN | 34,123 | 34,107 | 99.95 % |
| **AFRINIC** | 2,763 | **2,763** | **100 %** |
| **APNIC** | 30,735 | **25,927** | **84.36 %** |
| JPNIC | 448 | 448 | 100 % |
| **RIPE** | 39,640 | **0** | **0 %** |
| **LACNIC** | 14,279 | **0** | **0 %** |

And LACNIC is worse than the zero suggests. **[measured]** all **14,279 of 14,279** LACNIC `Organization`
records in the CAIDA file have `name: "undefined"` — there are no LACNIC organisation names in the dataset at
all — and there are exactly as many LACNIC "organisations" as LACNIC ASN records, i.e. CAIDA is manufacturing
one synthetic organisation per ASN because LACNIC's whois does not expose organisation objects publicly. RIPE
organisations *do* have real names (32,021 records, **0** with `name: "undefined"`), they simply carry no
`opaqueId`, so they cannot be joined to `delegated-ripencc-extended-latest`.

**So the clean offline path covers AFRINIC and APNIC, and does not exist for RIPE or LACNIC.** That is the
plain negative result this ticket asked for, and no workaround was found that does not route back through
registry terms.

One further limitation, structural: CAIDA's dataset is **AS-keyed**. An organisation that holds address space
but has never been assigned an ASN has no row, and is invisible to this path in every region. For a small
commercial operator with PA space from an upstream and no ASN of their own — a common case for the modal
operator — the join returns nothing.

---

## 6. BGP reality: route collectors other than RIPEstat

### 6.1 RouteViews — the clean win

**[measured]** `https://api.routeviews.org/asn/<asn>` returns a bare JSON array of prefix strings, keylessly:

| Query | Result |
|---|---|
| `/asn/13335` (Cloudflare) | 200, 106,795 B — **5,336 prefixes** (3,018 IPv6) |
| `/asn/33771` (Safaricom) | 200 in 0.43 s, 2,141 B — **119 prefixes** (19 IPv6) |
| `/asn/33764` (AFRINIC) | 200, 147 B — 8 prefixes |
| `/asn/44684` (Mythic Beasts) | 200, 160 B — 9 prefixes |

**[measured] client hazard: the endpoint 302-redirects and a client must follow it.**
`GET https://api.routeviews.org/asn/33771` returns **HTTP 302** with
`Location: https://api.routeviews.org/guest/asn/33771` and a 154-byte HTML body. A client that does not follow
redirects gets a 302 and an HTML payload where it expected a JSON array — silent failure, or a JSON parse
error, depending on how carelessly it is written. `curl -L` (or the equivalent) is mandatory.

**[measured] the rate limit is a pacer, not a rejecter.** Five requests fired back to back with no delay all
returned **200**, each taking **0.891–0.971 s** — the server holds the connection to enforce ~1 req/s rather
than returning 429. This is the friendliest throttle measured in either this document or #3: a naive client
cannot generate an error, only slow down. The documented behaviour matches
<https://api.routeviews.org/docs/>:

> *"The API can be accessed without authentication as a guest user, with severe rate limits to the usage
> (currently 1 API call per second). In order to relax these limits, the API users must authenticate
> themselves via the API key mechanism. … Rate limit for valid keys is 10 API calls per second."*

Note that the key upgrade is gated on PeeringDB: *"The API key management is available to any verified user
authenticated through PeeringDB."* Keyless guest access needs no account.

~~This replaces RIPEstat's `announced-prefixes` call one for one, for all four regions, with no ToS asterisk
(§8.6).~~

> **Amended 2026-08-15 by [#126](https://github.com/winniel123/verge-asm/issues/126) /
> [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md).** Every
> measurement above stands and none is re-derived: the endpoint works, the licence is clean, the
> redirect hazard is real and the pacer is the friendliest throttle in this corpus. What is withdrawn
> is the **conclusion**, and precisely: *replaces* is true of the **capability** and was read three
> times afterwards as a statement about the **shipped set**. RIPEstat's `announced-prefixes` was never
> in the shipped set — [#19](https://github.com/winniel123/verge-asm/issues/19) shipped RIPEstat off
> and [#47](https://github.com/winniel123/verge-asm/issues/47) scoped its toggle to org→prefix — so
> there was nothing to replace, and after ADR-0063 there is nothing for RouteViews to be the
> instrument of. **This section describes a source verge-asm does not query.**

### 6.2 RIPE RIS raw data and `riswhois` — works, but the terms bar us

**[measured]** `riswhois.ripe.net:43` with `-k -F -M 13335` returned **151,685 B** of
`origin<TAB>prefix<TAB>peer-count` lines. Keyless, no account.

But this is the RIPE Data Repository, and its Terms and Conditions
<https://www.ripe.net/analyse/raw-data-sets/terms-conditions/> (which redirect to
<https://www.ripe.net/about-us/legal/ripe-data-repository-terms-and-conditions/>) say:

> *"A User may only access the RIPE Data Repository and the Data for scientific research or research in order
> to support the operation of a network. Access to and use of the RIPE Data Repository and the Data for any
> commercial purposes, for example selling the Data or services based on the Data, is not allowed."*
>
> *"The User must provide a valid email address to the RIPE NCC for the purposes of establishing a channel of
> communication between the User and the RIPE NCC regarding the RIPE Data Repository."*

The commercial prohibition fails limb 2 outright for the modal operator, and clause 7 is a registration
requirement. It is genuinely ambiguous whether these T&C bind the port-43 `riswhois` gateway as opposed to the
MRT dumps at `data.ris.ripe.net`, **but that ambiguity does not need an email**, because RouteViews answers
exactly the same question under CC BY 4.0. Recorded as `unusable` and dropped.

### 6.3 IRR / RADB — ASN-first only, no org path

**[measured]** `whois.radb.net:43`:

- `!gAS13335` → 68,739 B in 0.41 s, a space-separated prefix list. Works, keyless.
- `Mythic Beasts` → `%  No entries found for the selected source(s).`
- `-i descr Mythic` → `%% ERROR: Inverse attribute search not supported for descr, only supported for
  attributes: role, zone-c, person, origin, mbrs-by-ref, admin-c, tech-c, mp-members, mnt-by, member-of,
  members`

The IRR holds organisation names in `descr:` attributes but **the inverse index does not cover `descr`**, so
there is no org-name entry point. IRR is a useful ASN → prefix corroborator and nothing more for this
question.

### 6.4 bgpview.io — dead

**[measured]** `https://api.bgpview.io/search?query_term=cloudflare` → `curl` exit code 6, *"Could not resolve
host: api.bgpview.io"*. The host does not resolve. Any design or tutorial that reaches for bgpview.io as the
keyless org→ASN→prefix service is out of date. **Unusable.**

---

## 7. PeeringDB — organisation and ASN metadata, not prefixes

**[measured]** the keyless API works and is fast:

| Query | Result |
|---|---|
| `https://www.peeringdb.com/api/org?name__contains=Cloudflare` | 200 in 0.42 s, 750 B — `{"id":4715,"name":"Cloudflare, Inc.", …}` |
| `https://www.peeringdb.com/api/net?name__contains=Cloudflare` | 200 in 0.45 s, 2,295 B — `"asn":13335`, `"irr_as_set":"AS13335:AS-CLOUDFLARE"`, `"info_prefixes4":80000`, `"info_prefixes6":30000` |
| `https://www.peeringdb.com/api/netixlan?asn=13335` | 200 in 0.63 s, 163,124 B |

**It does not yield prefixes.** `info_prefixes4` / `info_prefixes6` are the operator's self-declared *maximum
prefix limits* for peering sessions — a count used to size peer filters, not a list of address space. The
`netixlan` records are IXP peering-LAN addresses assigned by the exchange, which are **not** the
organisation's own prefixes and must never be treated as owned `Address` subjects.

What PeeringDB genuinely gives is **org name → ASN**, plus `irr_as_set`, which is the correct key for an IRR
expansion. As a link in a chain it is real; as an answer to this ticket it is not. Its terms are dealt with in
§8.7 and they are the same clause as APNIC's.

---

## 8. Terms of service, verbatim

### 8.1 APNIC Whois Database — `ambiguous — needs an email`

The `rel: "terms-of-service"` link emitted in every APNIC RDAP and whois response is
`http://www.apnic.net/db/dbcopyright.html`, which **[measured]** redirects to
<https://www.apnic.net/manage-ip/using-whois/bulk-access/copyright/>. It states:

> *"Except for Internet operational purposes approved by APNIC, no part of the APNIC whois data may be
> reproduced, stored in a retrieval system, or transmitted, in any form or by any means, electronic,
> mechanical, recording, or otherwise, without prior permission of APNIC on behalf of the copyright holders.
> Any use of this material to target advertising or similar activities is explicitly forbidden and will be
> prosecuted. APNIC requests to be notified of any such activities or suspicions thereof. The APNIC whois data
> may not be passed on in bulk to any other person or organization unless approved by APNIC."*

and glosses the carve-out:

> *"Users will not be able to download the full contents of the database unless the intended use is for
> 'Internet operational issues'. These words are tightly defined and would include network trouble-shooting,
> abuse reporting, and Internet research and analysis. It would not include compiling marketing lists,
> demographic mapping, or any other commercial application."*
>
> *"Each request would be carefully considered in light of the APNIC Whois Database acceptable use agreement."*

**Boundary case against ADR-0003 limb 1.** *"stored in a retrieval system"* is a literal description of what
verge-asm's `Observation` store is, and ADR-0003 limb 1 says *"a clause forbidding storage in another
retrieval system … fails here regardless of who the operator is."* Read flatly, that disqualifies APNIC
outright and no email is needed.

**It should not be read flatly, and ADR-0003's own reasoning is why.** The ADR analyses HackerTarget's clause
as the archetype, and HackerTarget's is **unqualified** — *"Nor may you transmit it or store it in any other
website or other form of electronic retrieval system"*, full stop. APNIC's carries a carve-out on its face:
*"Except for Internet operational purposes approved by APNIC"*. A qualified prohibition whose qualification
may well cover us is the definition of terms *"genuinely in tension"*, which is the trigger for the ambiguity
corollary, not for exclusion. The tension is real in both directions: the carve-out exists, but APNIC's own
gloss says approval is a per-request act (*"Each request would be carefully considered"*), not a standing
permission a stranger running an AGPL tool inherits.

**Ask APNIC.** Specifically: is an operator storing their own organisation's whois results across runs inside
*"Internet operational purposes approved by APNIC"*, or must each deployment seek approval individually? **If
the answer is no, APNIC joins HackerTarget as a clean limb-1 failure** and this section becomes a `unusable`
verdict rather than a pending one.

Limb 2 is not the problem: the marketing and bulk-redistribution prohibitions do not bite the modal operator.

### 8.2 AFRINIC Whois Database — `ambiguous — needs an email`

**[measured] the terms URL AFRINIC's own RDAP server advertises is broken.** Every AFRINIC RDAP response
carries `{"title":"Terms and Conditions", …, "rel":"terms-of-service","href":"https://afrinic.net/whois/terms"}`
and the port-43 banner says *"See https://afrinic.net/whois/terms"*. That URL returns **HTTP 404, 146 B**. The
live text is at <https://afrinic.net/whois-terms.html>. This matters operationally: a client that surfaces the
source's terms at enable-time, as ADR-0003 contemplates, would show its operator a 404.

The live terms are a **closed list**:

> *"AFRINIC Whois Database is to be used for the following purposes, including: Evaluating routing policies or
> ensuring compliance with routing policies. Facilitating operational coordination between network operators
> (e.g., network problem resolution, etc.). Providing reverse DNS (rDNS) delegations. For use in conjunction
> with providing network and Internet services so long as such use does not include reselling, available data
> contained within the AFRINIC Whois Database. Ensuring the uniqueness of Internet number resource usage.
> Conducting scientific research into network operations. Researching and tracking abuse issues in connection
> with the maintenance of AFRINIC Whois Data. Identifying resources used or suspected of being used for
> unlawful or harassing purposes. …"*
>
> *"Notwithstanding the above, AFRINIC Whois Database is specifically prohibited for the following: As part of
> a commercial service or product, including the solicitation and servicing of any customer, even if
> additional data not derived from the AFRINIC Whois Database is incorporated therein. For advertising, direct
> marketing, marketing research or similar purposes. For criminal and/or illicit purposes."*
>
> *"Where a legitimate need for high-volume queries or bulk access to the AFRINIC Whois Database exists, such
> need has to be requested from AFRINIC by emailing [address] with a request for bulk whois data."*
>
> *"Unless duly authorised in writing, no other use of the AFRINIC Whois Database other than those expressly
> described herein shall be implied or permitted. In the event that there is the need to discuss possible uses
> of the AFRINIC Whois database that are not described in the present Terms of Use, the AFRINIC Hostmaster may
> be contacted at [address]."*
>
> <https://afrinic.net/whois-terms.html>

**Why this cannot be resolved by reading.** ARIN's Terms of Use — which #3 relied on to clear ARIN — contain
the phrase *"Internet operational or technical research purposes"*, and #3 read "inventory my own attack
surface" as squarely inside it. **AFRINIC's list contains no such phrase.** The nearest fits are "Facilitating
operational coordination between network operators" and "Conducting scientific research into network
operations", and neither describes an operator cataloguing their own exposed services. The closing clause
*"no other use … shall be implied or permitted"* converts that stretch from a defensible reading into a
breach. **Ask AFRINIC** — at the Hostmaster address the terms themselves nominate for exactly this.

Limb 2's *"As part of a commercial service or product, including the solicitation and servicing of any
customer"* is a resale clause of exactly ARIN's shape and does **not** disqualify the modal operator per
ADR-0003.

### 8.3 LACNIC RDAP — `ambiguous — needs an email` (the terms cannot be read at all)

Every LACNIC RDAP response carries:

```json
{"title":"Terms and Conditions",
 "description":["This is the LACNIC RDAP service. Objects are in RDAP format."],
 "links":[{"value":"https://www.lacnic.net/registration-data-access-protocol",
           "rel":"terms-of-service",
           "href":"https://www.lacnic.net/registration-data-access-protocol","type":"text/html"}]}
```

**[measured] that URL, and every other documented LACNIC policy URL, returns a JavaScript application shell
containing no terms text.** Five distinct URLs, all HTTP 200, all returning a byte-identical **7,014 B** React
bootstrap whose only human-readable content is a `<title>` and Spanish-language `<meta description>` about
LACNIC as a registry:

- `https://www.lacnic.net/registration-data-access-protocol` (the ToS link RDAP emits)
- `https://www.lacnic.net/1024/2/lacnic/terms-of-use`
- `https://www.lacnic.net/1039/2/lacnic/terms-of-use-and-privacy-policy`
- `https://www.lacnic.net/2472/2/lacnic/accessing-bulk-whois`
- `https://www.lacnic.net/688/2/lacnic/8-request-for-bulk-whois-…`

The only terms LACNIC actually *delivers alongside the data* are the port-43 banner, **[measured]**:

> *"% Copyright LACNIC lacnic.net*
> *%  The data below is provided for information purposes*
> *%  and to assist persons in obtaining information about or*
> *%  related to AS and IP numbers registrations*
> *%  By submitting a whois query, you agree to use this data*
> *%  only for lawful purposes."*

**Boundary case against ADR-0003's "absence of terms clears the bar".** The ADR says *"a source that
publishes no terms presents nothing to breach"*, and on that corollary LACNIC would ship **`unencumbered`** —
"only for lawful purposes" imposes no limb-1 or limb-2 constraint on its face, and this would be the
convenient answer.

**LACNIC is not that case, and the distinction is load-bearing.** The corollary's worked example in ADR-0003
is crt.sh, which *asserts nothing*: it has no terms page, points at no terms page, and claims no agreement.
LACNIC's RDAP does the opposite — **every response carries a `"title":"Terms and Conditions"` notice and a
`rel: "terms-of-service"` link**, which is an affirmative assertion that a binding document exists and governs
the query being answered. A document that is asserted, nominated as binding, and then not served is not an
absence of terms; it is terms we cannot read. Treating "unreadable" as "absent" would let any source acquire
`unencumbered` status by breaking its own link, and it inverts the burden the ADR spends its Decision section
establishing (*"a legal reading performed on strangers' behalf is the same kind of assumption"* — reading a
document one has not seen is the limit case of that).

There is a second, independent reason not to resolve this by reading: **ADR-0003's opt-in pattern presupposes
that a source's terms can be surfaced to the operator at enable-time**, and LACNIC's cannot be. Whatever the
verdict, that has to be fixed before LACNIC can ship in any state.

**Ask LACNIC** for a retrievable statement of the terms governing automated RDAP querying, local storage and
cross-run retention. This is a cheap question with a decisive answer either way: if LACNIC confirms there are
no terms beyond the banner, the absence corollary applies cleanly and LACNIC becomes `unencumbered`.

### 8.4 RIPE Database — `ambiguous — needs an email` (fold into #19)

Terms and Conditions <https://docs.db.ripe.net/terms-conditions.html> (redirects to
<https://docs.db.ripe.net/HTML-Terms-And-Conditions>):

> **Article 3 — Purpose of the RIPE Database.** *"The RIPE Database contains information for the following
> purposes: Ensuring the uniqueness of Internet number resource usage through registration of information
> related to the resources and Registrants; Publishing routing policies by network operators (IRR);
> Facilitating coordination between network operators (network problem resolution, outage notification etc.);
> Facilitating network operations by publishing geolocation information …; Provisioning of Reverse Domain Name
> System (DNS) and ENUM delegations; Providing information about the Registrant and Maintainer of Internet
> number resources when the resources are suspected of being used for unlawful activities …; Scientific
> research into network operations and topology; Providing information to parties involved in disputes over
> Internet number resource registrations …"*
>
> **Article 4.1.** *"A User may only Access the RIPE Database and the data contained therein for any of the
> purposes as mentioned in Article 3 hereof and provided these Terms and Conditions are followed."*
>
> **Article 4.2.** *"A User may only conduct queries or submit updates of a nature, rate or volume permitted
> by the Acceptable Use Policy."*
>
> **Article 4.5.** *"A User may not re-package, download, compile, re-distribute or re-use any or all of the
> RIPE Database or the data contained therein unless they do so only with an insubstantial part of the RIPE
> Database or the data contained therein or when permission to do so is granted by the RIPE NCC."*

Acceptable Use Policy
<https://www.ripe.net/manage-ips-and-asns/db/support/documentation/ripe-database-acceptable-use-policy/>:

> *"No copy of a significant part of the RIPE Database is made without the consent of the RIPE NCC."*
> *"Number of queries from an IP address – Unlimited"*
> *"Number of personal data sets returned in queries from an IP address – 1,000 per 24 hours"*
> *"Number of personal data sets returned in queries from a proxy IP address – 20,000 per 24 hours"*
> *"Number of simultaneous connections to the RIPE Database server – 3"*
> *"Where no limits are set, or a limit is set as Unlimited, we work on the basis of reasonable use. This means
> that we expect Users not to do anything that could abuse or damage the RIPE Database service…"*

**The ticket's three RIPE sub-questions, answered:**

- ✅ **Do network objects fall outside the personal-data cap?** **Yes.** The AUP sets *queries themselves* to
  "Unlimited" and caps only *"personal data sets returned"*. #3's reading is confirmed by the current text.
  `inetnum` and `organisation` objects are not personal data sets. The binding numeric limit for a client is
  instead the **3 simultaneous connections** cap, which is a concurrency constraint, not a daily budget.
- ⚠️ **Is storing one org's `inetnum` set "a significant part"?** **Unresolved, and both wordings are live.**
  The ticket quoted the AUP's *"a copy of a significant part of the RIPE Database"* — **[measured]** that
  phrase is still present verbatim in the AUP's Introduction today. The T&C Article 4.5 states the same line
  from the other side, permitting re-use of *"an insubstantial part"*. Neither document defines the threshold.
  Whether one organisation's own object set (19 networks for Mythic Beasts) is insubstantial is a lawyer's
  question with an obvious common-sense answer and no textual one. **This cannot be settled by reading.**
- ❌ **Is ASM inventory an Article 3 purpose?** Article 4.1 confines Access to the Article 3 purposes, and
  **none of them is "inventory your own internet-exposed estate"**. This is the binding constraint, it is
  stricter than #3 recognised, and it was not on the ticket's list. It is the same class of question already
  open with RIPE NCC in [#19](https://github.com/winniel123/verge-asm/issues/19) about RIPEstat.

**Do not open a second conversation with RIPE NCC.** Extend #19's email to cover the RIPE Database REST search
API and `rdap.db.ripe.net` alongside RIPEstat, and ask the Article 3 / Article 4.5 / "significant part"
questions in the same message.

### 8.5 Delegated extended statistics files — `unencumbered`

<https://ftp.apnic.net/stats/apnic/README.TXT>, §2 CONDITIONS OF USE:

> *"The files are freely available for download and use on the condition that APNIC will not be held
> responsible for any loss or damage arising from the use of the information contained in these reports."*
>
> *"APNIC endeavours to the best of its ability to ensure the accuracy of these reports; however, APNIC makes
> no guarantee in this regard."*
>
> *"In particular, it should be noted that these reports seek to indicate where resources were first allocated
> or assigned. It is not intended that these reports be considered as an authoritative statement of the
> location in which any specific resource may currently be in use."*

A liability disclaimer and an accuracy caveat, and nothing else. No storage restriction, no commercial
restriction, no automation restriction, no attribution requirement. **Both limbs clear.** These files are
published as a joint RIR project and are the single cleanest registry data source found in this research or in
#3.

The accuracy caveat has a direct modelling consequence: a delegated-stats record is `inferred` authority for
current holding, not `declared`. It says where a range was *allocated*, not who uses it now.

### 8.6 RouteViews — `unencumbered`

<https://www.routeviews.org/routeviews/> and <https://www.routeviews.org/routeviews/licenses/>:

> *"Use of the data created and owned by RouteViews ("RouteViews Data") is licensed under a Creative Commons
> Attribution 4.0 International License (CC BY 4.0)."*
>
> *"…generally free to use RouteViews data for network operations and research, provided that you give
> attribution to RouteViews for use of the data."*
>
> *"When selling services, products, reports, or other derivative works based on RouteViews Data to third
> parties, you agree to provide attribution to RouteViews as follows: 'This [Product/Report] utilizes data
> provided by RouteViews (www.routeviews.org). Use of this data is subject to the CC BY 4.0 license.'"*
>
> *"RouteViews and University of Oregon may revoke your license to use RouteViews Data at any time if you do
> not provide the attribution called for above, or if you abuse this permission in any way."*

CC BY 4.0 permits commercial use, storage, and derivative works, subject to attribution. The page titled
"Commercial Terms of Use" is not a commercial *prohibition*: its obligations are attribution obligations
triggered *"When selling … to third parties"*, which the modal operator does not do. **Both limbs clear.** The
only shipping obligation is a visible attribution, the RouteViews logo, and the boilerplate description the
page specifies — which belongs in the UI and README.

This is the direct, clean replacement for RIPEstat's `announced-prefixes` capability.

### 8.7 PeeringDB — `ambiguous — needs an email`

<https://www.peeringdb.com/aup>:

> *"Except for Internet operational purposes approved by PeeringDB, no part of the PeeringDB data may be
> reproduced, stored in a retrieval system, or transmitted, in any form or by any means, electronic,
> mechanical, recording, or otherwise, without prior permission of PeeringDB on behalf of the copyright
> holders. Any use of this material to target advertising or similar activities is explicitly forbidden and
> will be prosecuted. PeeringDB requests to be notified of any such activities or suspicions thereof. The
> PeeringDB data may not be passed on in bulk to any other person or organization unless approved by
> PeeringDB."*
>
> *"Users will not be able to download the full contents of the database unless the intended use is for
> 'Internet operational issues'. These words are tightly defined and would include network trouble-shooting,
> abuse reporting, and Internet research and analysis. It would not include compiling marketing lists,
> demographic mapping, or any other commercial application."*

**This is word-for-word APNIC's clause with the name substituted** (§8.1) — **[measured]**, both pages
retrieved and compared. The identical limb-1 problem applies for the identical reason, and it needs the
identical question asked. Given that §7 establishes PeeringDB does not answer this ticket's question anyway —
it yields ASNs, not prefixes — **this email is low priority**; ask it only if PeeringDB is wanted for the
`irr_as_set` link.

### 8.8 CAIDA AS Organizations — `unencumbered`

The dataset README <https://publicdata.caida.org/datasets/as-organizations/README.txt> names its agreement as
the Public-AUA. <https://www.caida.org/about/legal/aua/public_aua/>:

> *"CAIDA's authorization to access the data grants You a limited, non-exclusive, non-transferable,
> non-assignable, and terminable license to copy, modify, and use the data in accordance with this Public
> Agreement. No license is granted for any other purpose and there are no implied licenses in this Agreement.
> Nothing in this License is intended to limit any rights You may have arising from fair use or due to other
> limitations on CAIDA's exclusive rights under copyright law or other applicable laws."*
>
> *"CAIDA has the authority and reserves the right, in its sole discretion, to discontinue further access and
> use to anyone who violates this AUA."*
>
> *"If You create a publication (including web pages, papers published by a third party, and publicly
> available presentations) using data from this dataset, You must cite the data as follows: The CAIDA UCSD
> [Dataset Name] - [dates used], https://catalog.caida.org/dataset/[dataset-URL]"*

An express licence to *"copy, modify, and use"*, with no non-commercial clause and no storage clause. **Both
limbs clear.** The citation obligation attaches to *publication*, not to internal use, so a self-hosted
inventory triggers nothing; the README's requested citation should nonetheless appear in verge-asm's
attributions.

**One caveat that is worth stating plainly and not resolving here.** CAIDA's dataset is *derived from RIR
whois*. CAIDA is the redistributor and carries whatever obligations that entails; a downstream user takes the
data under CAIDA's Public-AUA. This is the ordinary structure of licensed redistribution and is not a hidden
encumbrance — but it does mean the AFRINIC and APNIC "clean path" in §1 depends on CAIDA's own standing with
those registries, which is CAIDA's to maintain and not something verge-asm can verify.

### 8.9 RIPE RIS raw data / `riswhois` — `unusable`

Quoted at §6.2. *"Access to and use of the RIPE Data Repository and the Data for any commercial purposes …
is not allowed"* fails limb 2 for a small commercial organisation, and clause 7's email-registration
requirement makes it credentialed even for permitted users. Not worth an email because RouteViews covers the
same ground under CC BY 4.0.

### 8.10 `ftp.ripe.net/ripe/asnames/asn.txt` — `ambiguous`, and not needed

**[measured]** 200 in 1.52 s, **6,190,143 B**, lines of the form `13335 CLOUDFLARENET, US` — a global ASN →
name mapping covering all registries, not just RIPE. It would work as a CAIDA substitute for the org-name
step. It is served from RIPE NCC's FTP under no stated licence, and it is unclear whether the RIPE Database
T&C (§8.4) reach it. Since CAIDA provides the same mapping under a clean express licence, **do not use this
file** and do not spend a question on it.

---

## 9. Summary comparison table

| Source | Org name → ? | Keyless | Rate limit (measured) | Reliability | ToS constraint | **Verdict** |
|---|---|---|---|---|---|---|
| **AFRINIC RDAP `entities?fn=`** | **→ prefixes + ASNs, 1 request, inline** | Yes | None in 12/12; 0.9–18.5 s | Good; unbounded payload (1.4 MB for `fn=MTN`) | Closed permitted-use list, no "operational research" entry; *"no other use … shall be implied or permitted"* | **`ambiguous — needs an email`** |
| **LACNIC RDAP `entities?fn=<n>*` → `entity/`** | **→ prefixes + ASNs, 2 requests** | Yes | None; 0.5–1.2 s | Good; non-standard `entities` key; `*` mandatory | **No retrievable terms served at any of 5 documented URLs** | **`ambiguous — needs an email`** |
| **RIPE DB REST `type-filter=organisation` → RDAP `entity/`** | **→ prefixes + ASNs, 2 requests** | Yes | None in 12/12; 0.4 s + 0.8 s; 3 concurrent conns | Good; 406 + 0-byte body on wrong `Accept`; XML if `Accept` omitted | T&C Art. 4.1 confines use to Art. 3 purposes, none of them ASM; Art. 4.5 "insubstantial part"; AUP "significant part" | **`ambiguous — needs an email`** (fold into #19) |
| **APNIC whois `-T organisation` → `-i org`** | **→ prefixes + ASNs, 2 queries** | Yes | None in 10/10; 0.27–0.38 s | Excellent | *"stored in a retrieval system"* barred except for *"Internet operational purposes approved by APNIC"* | **`ambiguous — needs an email`** |
| APNIC RDAP `entities?fn=` | **→ nothing** | Yes | n/a | **Returns 200 + empty for every query — never 501** | — | **`unusable`** (and a client hazard) |
| RIPE RDAP `entities?fn=` | → nothing | Yes | n/a | **HTTP 500, well-formed RDAP error body** | — | **`unusable`** |
| LACNIC whois (port 43) | → nothing | Yes | n/a | *"accepts only direct match queries"* | — | **`unusable`** for this question |
| **Delegated extended stats (all 4 RIRs)** | → **opaque id only**; name not recoverable alone | Yes | None; 1.4–4.1 s for 1–18 MB | Excellent | *"freely available for download and use"* + liability disclaimer | **`unencumbered`** |
| **CAIDA `as-org2info`** | **→ ASN + `opaqueId`** (ARIN/APNIC/AFRINIC/JPNIC only) | Yes | None; 1.2 s for 4.4 MB | Excellent; **0 % opaqueId for RIPE, 0 % for LACNIC, no LACNIC org names at all** | Express licence to *"copy, modify, and use"*; cite on publication | **`unencumbered`** |
| **RouteViews `api.routeviews.org/asn/`** | → announced prefixes (ASN-keyed) | Yes | **Paces to ~1 req/s, never 429** (5/5 at 0.89–0.97 s) | Excellent, but **302-redirects to `/guest/…` — must follow** | **CC BY 4.0**; attribution; commercial terms only on resale | **`unencumbered`** |
| RIPE RIS `riswhois` / raw MRT | → announced prefixes (ASN-keyed) | Yes | Not probed | Good | *"for any commercial purposes … is not allowed"*; email registration required | **`unusable`** |
| IRR / RADB | → nothing from a name; `!gAS<n>` works | Yes | None; 0.4 s | Good | — | **`unusable`** for org-name entry |
| **PeeringDB API** | → **ASN + `irr_as_set`, not prefixes** | Yes | None; 0.4–0.6 s | Excellent | Verbatim identical to APNIC's *"stored in a retrieval system"* clause | **`ambiguous — needs an email`** (low priority) |
| bgpview.io | — | — | — | **Host does not resolve (curl exit 6)** | — | **`unusable`** |
| `ftp.ripe.net/ripe/asnames/asn.txt` | → ASN name, global, 6.2 MB | Yes | None; 1.5 s | Good | No stated licence; unclear whether RIPE DB T&C reach it | **`ambiguous`** — superseded by CAIDA, do not use |

---

## 10. Does ASN-first work where org-first does not, and how much is actually lost?

### 10.1 The chain is assemblable, and it is the answer for two regions

Yes. For **AFRINIC and APNIC** the chain composes from two sources neither of which does the whole job, and
both of which are `unencumbered`:

```
org name ──CAIDA as-org2info──▶ opaqueId ──delegated-<rir>-extended-latest──▶ registered prefixes
         └──CAIDA as-org2info──▶ ASN ──api.routeviews.org/asn/<n>──▶ announced prefixes
```

Both legs are bulk files or a paced API, both cache well, and neither needs a key or an account. **[measured]**
end to end for Safaricom: 4.4 MB + 1.0 MB of cached files plus one 0.43 s API call yields 10 registered IPv4
blocks, 2 registered IPv6 blocks, 3 ASNs and 119 announced prefixes.

For **RIPE and LACNIC** the chain does not compose, because the org-name step has no clean source: CAIDA
carries no `opaqueId` for RIPE (0 of 39,640) and no organisation names at all for LACNIC (0 of 14,279 usable).
There is no keyless, unencumbered org-name entry point into those two regions. **That is the negative result,
and no workaround was found that does not route back through registry terms.**

### 10.2 How much the BGP leg actually adds — measured, and it is less than the folklore

The case for BGP data is that it catches space an organisation announces but has not tidily registered. Two
worked comparisons, registered set from the registry path versus announced set from RouteViews:

**Safaricom (AFRINIC, AS33771)** — 10 registered IPv4 blocks, 100 announced IPv4 prefixes.
**[measured] 0 of 100 announced prefixes fall outside a registered block.** Every announcement is a
more-specific of space already in the delegated file. The BGP leg added decomposition detail, not territory.

**Mythic Beasts (RIPE, AS44684 + AS60011 + AS198557 + AS204345)** — 19 registered network objects carrying 20
CIDRs (14 IPv4, 6 IPv6); 17 announced prefixes (11 IPv4, 6 IPv6). **[measured] 2 of 11 announced IPv4
prefixes fall outside every registered IPv4 CIDR**: `198.199.155.0/24` and `185.33.27.0/24`. Roughly 18 % of
announced IPv4 prefixes are invisible to the registry path alone — space announced from a transit or customer
relationship rather than held under the org handle.

~~So the BGP leg is worth having and is not worth blocking on. **The registry leg carries most of the value**,
and for a small operator the BGP leg contributes a low-single-digit number of extra prefixes. Since RouteViews
is `unencumbered` and cheap, ship it — but do not let its absence hold up the registry path, and do not
present a registry-only result as incomplete.~~

> **Withdrawn 2026-08-15 by [#126](https://github.com/winniel123/verge-asm/issues/126) /
> [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md), at the site that
> specifies it — this is the sentence that would have caused a session to build it.** The two
> measurements above are correct and are the whole ruling's evidence; what is withdrawn is *ship it*.
> Read the 2-of-11 row again: Mythic Beasts is **a hosting company**, and this section's own gloss of
> those two prefixes — *"space announced from a transit or customer relationship rather than held
> under the org handle"* — names the instrument's characteristic **false positive**, not its yield. A
> transit provider announces its customers' prefixes and controls nothing inside them, so those rows
> are somebody else's estate. The full decomposition, and the one residue that could reopen this, is
> at §14.

### 10.3 Who genuinely needs this path

The ticket asks whether this changes v1 at all. On the measurements, three groups genuinely need it and one
does not:

- **Does not need it:** an operator whose estate sits in a handful of CIDRs they can name. They declare
  address-scope `Seed`s and the whole path is a convenience. This is probably the majority of the target
  persona.
- **Needs it — footprint larger than memory:** **[measured]** Mythic Beasts, a small UK hosting company, holds
  **19 network objects (20 CIDRs: 14 IPv4, 6 IPv6) and 4 ASNs** under one org handle, including three separate
  `/29` IPv6 allocations. Nobody types that list from memory correctly.
- **Needs it — space under subsidiary names:** **[measured]** "Safaricom" resolves to two distinct AFRINIC org
  handles in two countries; "Telstra" to several APNIC organisation objects (`ORG-TC6-AP`, `ORG-TCL23-AP`,
  `ORG-TIL3-AP`, `ORG-TIPL8-AP`, `ORG-TNT2-AP`, …) across two countries; "Cloudflare" to eight LACNIC entities
  across five countries. An operator declaring seeds by hand finds the parent and misses the subsidiaries —
  and the subsidiaries are exactly where forgotten exposure lives.
- **Needs it — announced-but-unregistered space:** the ~18 % measured above for Mythic Beasts.

**Recommendation for v1:** first-run depth stays registry-dependent and first-run keeps saying so, per
ADR-0003. But the shape of that message should change from "we cover North America properly" to **"we cover
North America, Africa and Asia-Pacific properly"** once the CAIDA + delegated-stats + RouteViews path ships,
because that path needs no permission from anybody. Europe and Latin America stay shallow until #19 and the
two new emails come back.

---

## 11. Findings

1. **AFRINIC has the best org-name → prefix endpoint of any RIR, including ARIN.**
   `rdap.afrinic.net/rdap/entities?fn=<name>` returns matching `ORG-*` entities with `networks` and `autnums`
   **embedded in the search result** — org name to complete registered footprint in **one keyless request**,
   where ARIN needs two. It uses substring matching and treats `*` literally, the exact inverse of LACNIC's
   convention, so a client written against ARIN's convention gets zero results and no error.

2. **APNIC's RDAP entity search is a no-op stub that returns HTTP 200 with an empty result set, and this is a
   correctness hazard, not just a gap.** Every query form returns 200 with `"entitySearchResults":[]`, while
   the same handles fetch fine at `entity/<handle>`. RFC 9082 requires 501 for unsupported query types. A
   client that trusts APNIC's 200 will report an empty APNIC footprint for every organisation, and in a drift
   product will report previously-known APNIC space as **removed** on the first run after discovery. Any RDAP
   client must treat an empty `entitySearchResults` from APNIC as `unknown`, never as `empty`.

3. **RIPE's entity search returns HTTP 500 — but the entity *fetch* works, so RIPE's only gap is name →
   handle, and `type-filter=organisation` closes it.**
   `rest.db.ripe.net/search?…&type-filter=organisation` → `rdap.db.ripe.net/entity/<ORG-…>` → `networks` is
   the working two-request RIPE chain, measured end to end on a small operator.

4. **#3's RIPE recipe is broken for most organisations and should not be implemented as written.**
   `type-filter=inetnum` with a multi-word query string returns **HTTP 400 `ERROR:115: invalid search key`**;
   it only worked in #3 because "cloudflare" is one token. Use `type-filter=organisation`.

5. **LACNIC is a working ARIN-shaped two-request path with two undocumented deviations:** results arrive under
   the non-standard key `"entities"` rather than RFC 9083's `entitySearchResults`, and a trailing `*` is
   mandatory — `fn=Cloudflare` returns 1 exact match where `fn=Cloudflare*` returns 8.

6. **The delegated extended statistics files cannot answer the question alone — the org name is genuinely not
   recoverable from an opaque-id — but joined to CAIDA's `as-org2info` they can, for AFRINIC and APNIC.**
   Both datasets are `unencumbered`, the join is measured working on two worked examples, and it is the only
   org-name → prefix path found that clears both limbs with no email outstanding.

7. **That join covers exactly two of the four regions, and the failure is total for the other two.** CAIDA
   carries `opaqueId` for 100 % of AFRINIC and 84.36 % of APNIC ASNs, and for **0 % of RIPE (0/39,640) and
   0 % of LACNIC (0/14,279)**. Worse, all 14,279 CAIDA LACNIC organisation records carry `name: "undefined"`,
   one per ASN, zero grouping — there are no LACNIC organisation names in the dataset at all. **There is no
   keyless, unencumbered org-name entry point into the RIPE or LACNIC regions.**

8. **The join is also blind to ASN-less organisations in every region**, because CAIDA's dataset is AS-keyed.
   A small operator with PA space from an upstream and no ASN of their own gets nothing from it — which is a
   meaningful slice of the modal operator population.

9. **RouteViews replaces RIPEstat's BGP capability outright, under CC BY 4.0.**
   `api.routeviews.org/asn/<n>` is keyless, returns a plain prefix array (5,336 prefixes for AS13335), and
   **paces to ~1 req/s rather than returning 429** — measured 5/5 at 0.89–0.97 s with no delay between
   requests. It is the only source in this research or #3 whose rate limit a naive client cannot turn into an
   error. It does, however, **302-redirect to `/guest/asn/<n>`**, so a client that does not follow redirects
   gets HTML where it expected JSON. Attribution is the whole ToS obligation.
   **Amended by [#126](https://github.com/winniel123/verge-asm/issues/126): every measurement and the
   `unencumbered` verdict stand, and *replaces* is true of the capability — but RIPEstat's BGP capability was
   never in the shipped set, so nothing was displaced, and after
   [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md) verge-asm queries this
   endpoint nowhere. §14.**

10. **The BGP leg adds less than its reputation suggests.** Measured: **0 of 100** announced IPv4 prefixes
    outside registered blocks for Safaricom; **2 of 11** for Mythic Beasts. ~~Ship it because it is clean and
    cheap, not because the registry path is incomplete without it.~~ **The measurements stand; the
    recommendation is withdrawn by [#126](https://github.com/winniel123/verge-asm/issues/126) /
    [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md) — *clean and cheap*
    prices the request rather than the singular confirmations it queues, and the 2-of-11 residue is the
    instrument's error mode rather than its yield. §14.**

11. **Four candidates are ambiguous and stop here, and three of them fail on the *same clause pattern*.**
    APNIC and PeeringDB publish word-for-word identical AUPs barring data being *"stored in a retrieval
    system"* except for *"Internet operational purposes approved by"* the operator — a carve-out their own
    gloss says is granted per request. AFRINIC publishes a **closed** permitted-use list containing no
    equivalent of the *"Internet operational or technical research purposes"* phrase that #3 relied on to
    clear ARIN, closed by *"no other use … shall be implied or permitted"*. RIPE's Article 4.1 confines Access
    to the enumerated Article 3 purposes, none of which is estate inventory. **None of these can be resolved
    by reading; all four are §12 tickets.**

12. **LACNIC serves no retrievable terms at all.** The `rel: "terms-of-service"` URL its own RDAP server
    emits, and four other documented policy URLs, all return a byte-identical 7,014 B JavaScript shell
    containing no terms text. This is not permissiveness and must not be read as such; it also independently
    breaks ADR-0003's opt-in pattern, which assumes a source's terms can be surfaced to the operator at
    enable-time.

13. **Two registries advertise terms URLs that are broken.** AFRINIC's RDAP and whois both point at
    `https://afrinic.net/whois/terms`, which returns **HTTP 404**; the live text is at
    `https://afrinic.net/whois-terms.html`. LACNIC's points at a JS shell. A consent-surfacing UI needs a
    curated terms-URL table, not the `rel: "terms-of-service"` link the protocol hands it.

14. **Of #20's three RIPE sub-questions, one is answered, one is not resolvable by reading, and the binding
    one was not on the list.** Network objects **do** fall outside the personal-data cap — the AUP sets
    queries to "Unlimited" and caps only *"personal data sets returned"*, confirming #3; the real numeric
    constraint is the **3 simultaneous connections** limit. The *"significant part"* question is **live in
    both documents** — the AUP still says *"No copy of a significant part of the RIPE Database is made without
    the consent of the RIPE NCC"* and T&C Art. 4.5 permits only *"an insubstantial part"* — and neither
    defines the threshold, so it cannot be settled by reading. But **Article 4.1's closed purpose list is the
    tighter constraint**, and it was not on the ticket's list at all.

15. **`bgpview.io` no longer resolves.** A keyless org→ASN→prefix service that appears in most recon tutorials
    is simply gone; `curl` cannot resolve `api.bgpview.io` (exit code 6).

16. **PeeringDB does not yield prefixes and should not be modelled as if it does.** `info_prefixes4: 80000`
    is a self-declared peering *max-prefix limit*, and `netixlan` records are IXP peering-LAN addresses
    assigned by the exchange. Treating either as owned `Address` subjects would put third-party IXP
    infrastructure inside an operator's estate and, under
    [ADR-0002](../adr/0002-ownership-gates-probing.md), make it eligible for probing. What PeeringDB does give
    is org name → ASN and `irr_as_set`.

17. **Two verdicts here are boundary cases that ADR-0003 does not squarely decide, and both are resolved
    toward asking.** APNIC's *"stored in a retrieval system"* clause has the shape ADR-0003 limb 1 says fails
    *"regardless of who the operator is"* — but unlike HackerTarget's archetype it carries a carve-out
    (*"Except for Internet operational purposes approved by APNIC"*), which makes it terms in tension rather
    than terms that fail. LACNIC serves no readable terms, which looks like ADR-0003's *"absence of terms
    clears the bar"* — but LACNIC's RDAP affirmatively **asserts** that a binding terms document exists and
    then fails to serve it, which is not the crt.sh case the corollary was written for. Treating unreadable
    as absent would let any source earn `unencumbered` by breaking its own link.

18. **ADR-0003's stated coverage consequence is now beatable.** The ADR records that *"ARIN's `entities?fn=`
    org-name path carries the keyless default set and covers North America only."* On these measurements the
    keyless default set can carry **three of five regions** — North America, Africa and Asia-Pacific — with
    nobody's permission, via CAIDA ⋈ delegated stats. Europe and Latin America are the residue, and they are
    the two the emails are for.

---

## 12. What needs an email — proposed follow-on `task` tickets

Per [ADR-0003](../adr/0003-third-party-source-consent-bar.md), the output of an ambiguous clause is a `task`
ticket in the shape of [#19](https://github.com/winniel123/verge-asm/issues/19), not a verdict — and each of
these sources meanwhile *"ships off by default until an answer arrives — indefinitely if none does."* Four are
proposed, in priority order. **All four block the same thing**: whether verge-asm's default address-scope
discovery reaches Europe and Latin America.

1. **Extend #19's RIPE NCC email to cover the RIPE Database** (do not open a second thread). Ask: (a) is an
   organisation running verge-asm to inventory its own internet-exposed estate acting within one of the
   Article 3 purposes; (b) is one organisation's own `inetnum` / `organisation` object set an *"insubstantial
   part"* under Article 4.5 — equivalently, not *"a significant part"* under the AUP's Introduction;
   (c) does storing those objects in a local database across runs count as *"re-use"*. Quote §8.4.

2. **Email LACNIC** asking for a retrievable statement of the terms governing automated RDAP querying, local
   storage and cross-run retention — and reporting that the `rel: "terms-of-service"` URL their RDAP server
   emits serves no terms text. Quote §8.3. This one is unusual: we are asking for the document's *existence*
   in readable form before we can ask anything about its content. **Cheapest of the four to resolve** — if
   LACNIC confirms there is nothing beyond the whois banner, ADR-0003's absence corollary applies cleanly and
   LACNIC becomes `unencumbered`, taking Latin America to the same footing as Africa.

3. **Email AFRINIC hostmaster** (the address their terms nominate for exactly this) asking whether an
   operator inventorying their own estate falls within the permitted-use list, given that the list is closed
   by *"no other use … shall be implied or permitted"* and contains no equivalent of ARIN's *"Internet
   operational or technical research purposes"*. Also report the 404 at `https://afrinic.net/whois/terms`.
   Quote §8.2.

4. **Email APNIC** asking whether an ASM tool storing an organisation's own whois results across runs is an
   *"Internet operational purpose approved by APNIC"*, or whether each deployment must seek approval
   individually. Quote §8.1. **A "no" here makes APNIC a clean ADR-0003 limb-1 failure**, alongside
   HackerTarget — the only outcome among the four that removes a capability rather than unlocking one.
   Note that APNIC's *offline* path (CAIDA ⋈ `delegated-apnic-extended-latest`, §5.2) is unaffected either
   way: the delegated files carry their own permissive conditions of use (§8.5) and are not APNIC whois data.
   **The same question, verbatim, goes to PeeringDB** if PeeringDB is wanted for `irr_as_set` — but PeeringDB
   does not answer this ticket, so that is a separate, lower-priority ticket.

---

## 13. Open questions for the spec

- Should the CAIDA + delegated-stats join ship as a **bundled, periodically-refreshed table** (**[measured]**
  4.4 MB for `latest.as-org2info.jsonl.gz` as served, plus 1.0 MB AFRINIC and 9.2 MB APNIC delegated files
  uncompressed) rather than a live lookup? It caches perfectly, works offline, and removes a third-party
  dependency from the first-run path entirely — which is what #3's §10 already asked about RIPEstat. The cost
  is staleness between refreshes and shipping a dataset in the container image.
- ~~Given finding 8, is an **ASN-less organisation** the common case for the modal operator? If so the whole
  ASN-keyed leg is worth less than it looks and the registry paths in §2–§4 — the encumbered ones — carry
  nearly all the value, which raises the stakes on §12's emails.~~
  **Answered, twice, and the second answer went further than this question asked.**
  [#26](https://github.com/winniel123/verge-asm/issues/26): yes — **43.0%** of the registered population holds
  no ASN, and the registered population is itself under 1% of the persona, so the operator is not merely
  ASN-less but **registry-less**. [#126](https://github.com/winniel123/verge-asm/issues/126) then found the
  ASN-keyed leg is worth less than it looks for a reason that is not about the axis at all
  ([ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md)); §14.
- Does first-run's "registry-dependent depth" message need to name the *reason* per region — "we cannot query
  the RIPE Database until RIPE NCC answers" reads very differently from "RIPE has no keyless path", and only
  one of them is true.
- What is the correct `authority` for a delegated-stats record, given the file's own caveat that it is *"not
  intended … as an authoritative statement of the location in which any specific resource may currently be in
  use"*? `inferred`, on the reading in §8.5 — but that needs to be settled in `CONTEXT.md` rather than assumed
  by whoever writes the loader.

---

## 14. The BGP leg does not ship — [#126](https://github.com/winniel123/verge-asm/issues/126)

Added 2026-08-15. Sections 1, 6.1, 10.2, 11 (findings 9 and 10) and 13 are amended in place above; this
section carries the reasoning so that none of it is re-derived. **No measurement in this note is
re-taken, contradicted or re-interpreted.** Every figure §6.1 and §10.2 record is correct, and two of
them are what decide the question against the recommendation the same sections made.

The ruling is [ADR-0063](../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md): **a
routing announcement names who carries packets toward a prefix, never who controls what listens in it.**

### 14.1 The difference set decomposes exhaustively, and none of it is undiscovered estate

Difference an AS's announced set against the registered rows the holder's opaque-id groups. The residue
is three things and no fourth:

| Limb | The rows | Already reached by | Measured here |
|---|---|---|---|
| **1** | More-specifics of registered space | The registry leg. And once a proposal is confirmed, an address scope **enumerates** ([ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md)) — every address in the more-specific is a subject already. Decomposition, never territory | **100 of 100** Safaricom prefixes (§10.2) |
| **2** | Space registered to **somebody else** that this AS carries — customer or transit | Nothing, and correctly so. `Custody` is control of what listens ([ADR-0013](../adr/0013-custody-is-control-and-extends-by-declaration.md)); the carrying AS controls nothing inside a customer's prefix | The **2 of 11** Mythic Beasts residue, on this note's own description of it (§10.2) |
| **3** | Space registered to the operator under a **different handle** — a subsidiary | The org-name box, structurally: a subsidiary holds a different opaque-id, which is exactly the gap [#43](https://github.com/winniel123/verge-asm/issues/43) shipped the CAIDA join to close | §10.3's two-handle Safaricom and eight-entity Cloudflare cases |

The leg's only **unique** limb is 2, and limb 2 is not the operator's estate.

### 14.2 The one non-zero yield is the error mode wearing the finding's clothes

§10.2's *"roughly 18% of announced IPv4 prefixes are invisible to the registry path alone"* is the whole
positive case ever made for this leg, and it is measured on **a hosting company**. The section's own
gloss of the two prefixes names the mechanism: *"space announced from a transit or customer relationship
rather than held under the org handle."* That is limb 2, described accurately and then counted as yield.

**A yield figure measured on the population where the instrument's error mode lives is not a yield
figure.** This note refused the same shape once already, at §7, without naming it: PeeringDB's `netixlan`
records were declined because *"treating [them] as owned `Address` subjects would put third-party IXP
infrastructure inside an operator's estate."* IXP peering-LAN addresses and carried customer prefixes
fail identically — both are routing facts about the path, offered as facts about the holder.

### 14.3 The price is confirmations, not requests

*"Clean and cheap"* (§10.2, finding 10) prices the HTTP request. The request is not the cost.
[ADR-0022](../adr/0022-confirmation-is-singular.md) makes confirmation **singular** by design, so a
proposer's real price is the length of the queue it puts in front of the operator at the one act where
the probing gate opens. From this note's own numbers:

| Organisation | Registry leg proposes | BGP leg adds | New territory |
|---|---|---|---|
| Safaricom, AS33771 | 10 IPv4 blocks (§5) | **100** announced prefixes, each a subset of those 10 | **0** |
| Cloudflare, AS13335 | — | **5,336** announced prefixes (§6.1) | not measured |
| Mythic Beasts, 4 ASNs | 19 network objects | 17 announced prefixes | 2, of limb-2 shape |

An order-of-magnitude multiplication of a singular-confirmation queue for zero territory is the most
expensive thing a proposer can do, because what it spends is the operator's attention.

### 14.4 An announcement is a configuration the operator authored

Every argument this note makes for a proposer is §10.3's: the estate exceeds memory, *"nobody types that
list from memory correctly."* Registry rows earn that — they are paperwork filed by people who have left.
**A BGP announcement is not paperwork.** It is a live router configuration somebody in the operator's own
organisation maintains this week. Reconstructing it from a third party's route collector — with limb 2
mixed inseparably in — is a guess-based technique standing in for a declaration the operator's own
organisation holds, which is the inversion the map's spine principle refuses.

### 14.5 The consent verdict is untouched, and it is not an argument

§8.6's `unencumbered` verdict on RouteViews stands in full and is re-confirmed against
[ADR-0003](../adr/0003-third-party-source-consent-bar.md) at that ADR's sixth amendment: limb 1 clears
(CC BY 4.0 permits automated querying, storage and retention, with no retrieval-system clause), limb 2
clears (the commercial condition triggers only on **selling derivative works to third parties**, the
reseller shape ADR-0003 says does not fail), and the fifth amendment does not reach it because a public
licence travels to AGPL-3.0 recipients intact. **The bar decides whether a source may run and is silent
on whether it is worth running**; this is the corpus's first candidate to clear it cleanly and be refused
anyway.

Nor was there anything to displace. RIPEstat's `announced-prefixes` was never in the shipped set —
[#19](https://github.com/winniel123/verge-asm/issues/19) shipped RIPEstat off and
[#47](https://github.com/winniel123/verge-asm/issues/47) scoped its toggle to org→prefix — so §6.1's
*replaces* was true of the **capability** and has been read since as a statement about the **shipped
set**.

### 14.6 Where this is thin, and what reopens it — a rerun, not an argument

One shape the decomposition does not obviously exhaust: **a provider-aggregatable assignment announced by
the organisation that uses it and registered under its upstream LIR's handle.** It is the operator's
custody, it sits outside every row their own opaque-id groups, and outside ARIN — where
[#39](https://github.com/winniel123/verge-asm/issues/39)'s SWIP `C…` customer objects reach it down to a
/29 — no shipped path may reach it.

**Nobody has measured how often that happens.** Both comparisons in §10.2 are holders with their own PI
space; neither is that case, and n=2 with a 0 and an 18% is not a rate. Two things bound it: reaching it
requires the operator to hold **their own ASN**, which [#26](https://github.com/winniel123/verge-asm/issues/26)
prices at ≤57% of a population under 1% of the persona; and an organisation running its own AS over
borrowed address space is the operator most certain to know which prefixes it originates.

**Reopening condition, stated as a measurement so it can be run rather than argued:** for a sample of
small, single-ASN, **non-transit** organisations, count announced IPv4 prefixes that fall outside every
delegated row sharing the holder's opaque-id **and** are not registered to any third party. If that count
is materially non-zero, limb 2 is not exhaustive of the residue and ADR-0063's application to this leg —
though not its rule — is reopened. The measurement is the whole ruling on this point, so it is a **rerun**
and not a new argument.

### 14.7 By-catch, not decided here

- **A `Proposal` larger than the range cap has no stated route to becoming a `Seed`.** ADR-0047 caps an
  address scope at **1,024 addresses**, checked at declaration; the registry leg routinely proposes far
  larger blocks — Safaricom's 10 AFRINIC IPv4 blocks among them. Whether such a proposal is refused,
  offered for narrowing, or routes the operator to a `custody extension` is unspecified. It was raised in
  #126 as a possible rescue for the BGP leg (announced `/24`s fit where a registered `/16` does not) and
  refused: a routing decomposition is the network engineer's and not the estate's, and a hole in the
  `Proposal`-to-`Seed` path is not an argument for a source.
- **Overlapping address scopes.** Confirming a proposal and a more-specific of it yields two `Seed`s
  covering the same addresses. `Custody` is unaffected — *covered by a `Seed`?* is total — but ADR-0047
  gives an address scope a `Coverage` denominator, and what two overlapping denominators do is not
  written down.
