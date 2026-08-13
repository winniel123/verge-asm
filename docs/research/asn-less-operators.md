# Is the modal operator ASN-less?

Research ticket #26 — wayfinder research for the verge-asm v1 spec.

**Question.** Is the modal operator ASN-less — and if so, how much of
[#20](https://github.com/winniel123/verge-asm/issues/20)'s keyless org→prefix win survives?

**Framing.** [#20](https://github.com/winniel123/verge-asm/issues/20) closed the ARIN-only gap for
two regions with an `unencumbered` path: CAIDA `as-org2info` joined to
`delegated-<rir>-extended-latest` on the NRO opaque-id. The ticket's worry is that this entire path
is **AS-keyed**, so an organisation holding provider-aggregatable space with no ASN of its own gets
nothing from it — and that if the modal operator is that organisation, #20's win is mostly
illusory, the encumbered registry paths carry the value, and
[#19](https://github.com/winniel123/verge-asm/issues/19) plus the three registry asks become
urgent.

The mechanical half of that worry is **correct and is confirmed below in the dataset's own format
specification**. The strategic half — that this makes the encumbered paths more valuable — is
**falsified**. The measurements point somewhere the ticket did not anticipate: the modal operator is
not merely ASN-less, they are **registry-less**, and the entire org→prefix family (AS-keyed and
name-keyed, `unencumbered` and `operator-accepted` alike) is invisible to them in the same way and
for the same reason. Making the encumbered paths available does not reach this operator either.

Two constraints from decisions already made shape the answer before any evidence is gathered:

1. **[ADR-0002](../adr/0002-ownership-gates-probing.md) derives `Ownership` per `Address` and gates
   probing on it.** So "which addresses are ours" is a safety property, not a display nicety, and
   the question of what an ASN-less operator's addresses *are* has teeth (§6).
2. **[ADR-0003](../adr/0003-third-party-source-consent-bar.md) defines the modal operator** as *"a
   small commercial organisation inventorying its own estate"*. That definition is the denominator
   for every ratio in this note, and §4 takes it literally rather than substituting "organisation
   that shows up in registry data", which is the substitution that makes this question look closer
   than it is.

Throughout, findings are labelled **[measured]** (numbers run here, commands in §2), **[cited]** (a
published figure from the body that owns it), or **[inferred]**.

---

## 1. Summary

| Decision | Answer |
|---|---|
| Is the modal operator ASN-less | **Yes — and more than that, registry-less.** Fewer than 1 % of small commercial organisations hold *any* RIR number resource, ASN or address — §4 |
| Is #20's path AS-keyed | **Yes, provably.** CAIDA publishes `opaque_id` in the *AS* record format and not in the *organization* record format. 0 of 98,597 organisation records carry one — §5 |
| How much of #20's win is lost to it | **Materially less than feared.** The AS-keying costs 66.5 % of APNIC's registered address holders and 16.4 % of AFRINIC's — but only APNIC has no name-keyed alternative — §5.3 |
| Does the ASN-less case break the *other* registry paths | **No.** ARIN, RIPE, AFRINIC and LACNIC all key on organisation name/handle, not ASN. Four ASN-less organisations resolved end to end — §6 |
| Does an org→prefix lookup reach a PA-space renter | **In ARIN, yes** — SWIP customer objects are name-searchable and embed the reassigned CIDR. This falsifies the ticket's premise for one region — §7 |
| What an ASN-less cloud estate looks like in address terms | All 52,680 published AWS/Azure/GCP IPv4 prefixes belong, in registry terms, to **48 organisations**, none of them the operator — §8 |
| Convenience or capability | **Convenience for 97.5 % of registered address holders** (those holding ≤10 blocks); 79.1 % of ASN-less holders hold exactly one. Of the organisations for whom it is a real capability, **85.1 % hold an ASN** — §9 |
| Does #19 become more urgent | **No — less.** It buys depth in the same sub-1 % population, and it does not reach the modal operator either — §10 |
| Onboarding message for an ASN-less operator | **Not a missing-capability annotation.** "Declare your CIDRs directly" is the true message — §10 |

The headline is the one that would not have come out of reasoning about ASNs at all:

> **The ASN is the wrong axis.** The org→prefix path serves the **128,233 organisations worldwide
> that hold an RIR delegation** — against 6,395,635 US employer firms and 32,255,105 EU27
> enterprises. Whether those 128,233 hold an ASN changes which *sources* reach them; it does not
> change that they are **under 1 % of the persona ADR-0003 describes**. #20's win is small, and it
> was always going to be small — but it is not made smaller by AS-keying in the way the ticket
> supposed, and the encumbered paths it was competing with are small in exactly the same way.

---

## 2. What was measured, and from what

Everything below was retrieved with `curl` on **2026-08-13 UTC** from a single commercial IP and
counted locally with `awk`. No summarising fetch layer was used, and no secondary write-up is relied
on for any number.

| Dataset | URL | Retrieved | Content date / size |
|---|---|---|---|
| ARIN delegated extended | [`ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest`](https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest) | 2026-08-13 | header `2.3\|arin\|…\|201935\|…\|20260812`, 12,749,270 B |
| RIPE NCC delegated extended | [`ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest`](https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest) | 2026-08-13 | `2\|ripencc\|…\|260194\|…\|20260812`, 18,020,508 B |
| APNIC delegated extended | [`ftp.apnic.net/apnic/stats/apnic/delegated-apnic-extended-latest`](https://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-extended-latest) | 2026-08-13 | `2.3\|apnic\|20260813\|189082\|…`, 9,194,052 B |
| LACNIC delegated extended | [`ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest`](https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest) | 2026-08-13 | `2.3\|lacnic\|20260812\|96857\|…`, 4,546,193 B |
| AFRINIC delegated extended | [`ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest`](https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest) | 2026-08-13 | `2\|afrinic\|20260813\|19562\|…`, 987,806 B |
| CAIDA `as-org2info` | [`publicdata.caida.org/datasets/as-organizations/latest.as-org2info.jsonl.gz`](https://publicdata.caida.org/datasets/as-organizations/latest.as-org2info.jsonl.gz) | 2026-08-13 | byte-identical to `20260801.as-org2info.jsonl.gz`; 220,585 records |
| CAIDA `routeviews-prefix2as` | [`publicdata.caida.org/datasets/routing/routeviews-prefix2as/2026/08/routeviews-rv2-20260811-1000.pfx2as.gz`](https://publicdata.caida.org/datasets/routing/routeviews-prefix2as/2026/08/) | 2026-08-13 | 1,116,792 prefixes |
| AWS IP ranges | [`ip-ranges.amazonaws.com/ip-ranges.json`](https://ip-ranges.amazonaws.com/ip-ranges.json) | 2026-08-13 | `createDate` `2026-08-13-08-57-05` |
| Azure service tags | [`download.microsoft.com/…/ServiceTags_Public_20260810.json`](https://www.microsoft.com/en-us/download/details.aspx?id=56519) | 2026-08-13 | `changeNumber` 413 |
| GCP cloud ranges | [`gstatic.com/ipranges/cloud.json`](https://www.gstatic.com/ipranges/cloud.json) | 2026-08-13 | `creationTime` `2026-08-13T01:04:12` |
| US Census SUSB 2022 | [`www2.census.gov/programs-surveys/susb/tables/2022/us_state_naics_detailedsizes_2022.txt`](https://www2.census.gov/programs-surveys/susb/tables/2022/us_state_naics_detailedsizes_2022.txt) | 2026-08-13 | `Last-Modified: 2025-04-10`, 2022 reference year |
| Eurostat `sbs_sc_ovw` | [`ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/sbs_sc_ovw`](https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/sbs_sc_ovw?format=JSON&lang=EN&geo=EU27_2020&indic_sbs=ENT_NR&nace_r2=B-S_X_O_S94&time=2022) | 2026-08-13 | 2022 reference year, `updated` 2026-03-10 |

### 2.1 What a delegated-stats file does and does not contain

This bounds every count in §3 and §4, so it is stated before the numbers rather than after.

The [NRO extended-stats format specification](https://www.nro.net/wp-content/uploads/nro-extended-stats-readme5.txt)
(retrieved 2026-08-13) defines the join key this note leans on:

> "opaque-id  This is an in-series identifier which uniquely identifies a single organisation, an
> Internet number resource holder.
>
> All records in the file with the same opaque-id are registered to the same resource holder.
>
> The opaque-id is not guaranteed to be constant between versions of the file."

So counting distinct opaque-ids within **one** file counts **resource holders**, which is exactly
what is wanted. The instability warning bites across versions, not within one.

**The file records RIR→holder delegations only.** Sub-allocations an ISP makes to its own customers
are not in it. **[measured]** the smallest IPv4 record across all five files is 8 addresses (a /29),
and there are only 9 of those, 5 of size 16 and 31 of size 32 out of ~260,000 records; ARIN's
smallest is 256 (a /24) with no exceptions. If customer reassignments were present, ARIN alone would
carry tens of thousands of /29-to-/27 records, because its own policy compels their registration
(§7.2). They are absent. This is the single most important scoping fact in the note: **the delegated
files describe who holds space *from a registry*, not who uses space.**

```sh
# distinct resource holders, and how many hold an ASN, per RIR
awk -v RIRNAME=ARIN -f orgcount.awk delegated-arin-extended-latest
# (orgcount.awk: skip '#' and summary lines; key on $8 opaque-id;
#  status in {allocated, assigned}; flag holders by $3 in {asn, ipv4, ipv6})

# smallest IPv4 record across all five files
awk -F'|' '!/^#/ && NF>=8 && $2!="*" && $3=="ipv4" && ($7=="allocated"||$7=="assigned") {print $5}' \
  all-delegated.txt | sort -n | uniq -c | head -4
```

---

## 3. The registered population is 128,233 organisations worldwide

**[measured]**, from the five delegated files, counting distinct opaque-ids with at least one
`allocated` or `assigned` record:

| RIR | Holders of any resource | Hold ≥1 ASN | Hold ≥1 IPv4 block | IPv4 **and** ASN | IPv4 but **no** ASN |
|---|---:|---:|---:|---:|---:|
| ARIN | 39,212 | 24,338 (62.1 %) | 30,570 | 16,085 (52.6 %) | **14,485 (47.4 %)** |
| RIPE NCC | 44,644 | 32,365 (72.5 %) | 32,678 | 21,280 (65.1 %) | **11,398 (34.9 %)** |
| APNIC | 26,687 | 8,842 (33.1 %) | 25,778 | 8,629 (33.5 %) | **17,149 (66.5 %)** |
| LACNIC | 14,754 | 13,617 (92.3 %) | 11,474 | 10,556 (92.0 %) | **918 (8.0 %)** |
| AFRINIC | 2,936 | 2,459 (83.8 %) | 2,874 | 2,402 (83.6 %) | **472 (16.4 %)** |
| **All five** | **128,233** | **81,621 (63.7 %)** | **103,374** | **58,952 (57.0 %)** | **44,422 (43.0 %)** |

Two independent corroborations that this population is the right order of magnitude:

- **[measured]** the global BGP table on 2026-08-11 carries **1,116,792 prefixes originated by
  78,836 distinct origin ASes** (CAIDA `routeviews-rv2-20260811-1000.pfx2as`). That is the hard
  ceiling on organisations reachable by *any* AS-keyed instrument, and it is smaller than the
  registered population because not every allocated ASN is announced.
- **[measured]** CAIDA's `as-org2info` 2026-08-01 snapshot contains **98,597 organisation records
  and 121,988 ASN records** — the same order, derived independently from RIR whois rather than from
  delegated stats.

### 3.1 The denominator ADR-0003 actually names

ADR-0003's modal operator is *"a small commercial organisation inventorying its own estate"*. The
official statistics for that population:

- **[cited]** **6,395,635 US employer firms** in 2022 — US Census Bureau, Statistics of U.S.
  Businesses, `us_state_naics_detailedsizes_2022.txt`, row `STATE=00, NAICS=--, ENTRSIZE=01`
  ("01: Total"), field `FIRM`. Of those, **5,720,093 have fewer than 20 employees** (`ENTRSIZE=33`,
  "06: <20") and **6,374,594 fewer than 500** (`ENTRSIZE=37`, "19: <500") — that is 99.7 %.
- **[cited]** **32,255,105 enterprises in the EU27** in 2022 — Eurostat `sbs_sc_ovw`, `geo=EU27_2020`,
  `indic_sbs=ENT_NR` ("Enterprises - number"), `nace_r2=B-S_X_O_S94`, `size_emp=TOTAL`. Of those,
  **30,394,396 employ 0–9 persons** (94.2 %); only **1,860,707 employ 10 or more**.

The ratios, **[measured]** against **[cited]**:

| Comparison | Ratio |
|---|---|
| ARIN-region resource holders ÷ US employer firms | 39,212 ÷ 6,395,635 = **0.61 %** |
| ARIN-region **ASN** holders ÷ US employer firms | 24,338 ÷ 6,395,635 = **0.38 %** |
| RIPE-region resource holders ÷ EU27 enterprises | 44,644 ÷ 32,255,105 = **≤ 0.14 %** |
| RIPE-region **ASN** holders ÷ EU27 enterprises | 32,365 ÷ 32,255,105 = **≤ 0.10 %** |

Both RIPE rows are upper bounds, because the RIPE service region is substantially larger than the
EU27 while the denominator is not.

Push the denominator as far toward the numerator as honesty allows — restrict to firms with 20+
employees (675,542 in the US; 1,860,707 with 10+ in the EU27) and pretend every single
registry-holding organisation is such a firm rather than an ISP, university, hospital, government
department or bank:

- ARIN ASN holders ÷ US firms with 20+ employees = 24,338 ÷ 675,542 = **≤ 3.6 %**
- RIPE ASN holders ÷ EU27 enterprises with 10+ employees = 32,365 ÷ 1,860,707 = **≤ 1.7 %**

**[inferred]** the true figure is well below both, because a large share of registry holders are
network operators by trade — the population that needs address space *because* selling connectivity
is the business. Nothing here measures that share, and §12 records it as unestablished.

**Verdict.** The modal operator is ASN-less. That is not the interesting part. The modal operator
holds **no RIR resource of any kind**, which means the org→prefix path returns the empty set for
them — correctly, and no matter which registry, which key, or which consent tier it runs under.

---

## 4. Within the registered population, 43.0 % hold no ASN

**[measured]**, IPv4 holders bucketed by total addresses held, all five RIRs combined:

| Total IPv4 held | Orgs | Of which hold an ASN | % with ASN |
|---|---:|---:|---:|
| ≤ /24 (≤256) | 29,107 | 12,075 | **41.5 %** |
| /23–/22 (512–1,024) | 41,287 | 24,142 | 58.5 % |
| /21–/20 (2,048–4,096) | 16,627 | 11,345 | 68.2 % |
| /19–/16 (8k–65k) | 13,029 | 9,098 | 69.8 % |
| > /16 (>65k) | 3,324 | 2,292 | 69.0 % |

The gradient is the expected one and it confirms the ticket's intuition **within** the registered
population: the smaller the address holding, the less likely an ASN. At the small end — the shape
closest to ADR-0003's modal operator — **fewer than half** hold an ASN.

**One caveat on the APNIC figure, which is the most extreme in the table.** APNIC's 66.5 % ASN-less
rate is partly structural rather than behavioural. **[measured]** 16,678 APNIC IPv4 records are
exactly 1,024 addresses (a /22) and **9,502 of them (57.0 %) fall inside 103.0.0.0/8** — the block
APNIC issued under its final-/8 policy, one /22 per member. APNIC's file also carries NIR
sub-delegations (JPNIC, KRNIC and others appear as separate holders), so it reaches one level deeper
into end organisations than ARIN's does. Both effects inflate the count of small ASN-less holders
relative to the other regions. The cleanest single-registry readings are therefore **ARIN 47.4 %**
and **RIPE 34.9 %**, and those are the ones to quote.

---

## 5. #20's join key is published per-ASN only — proven from the dataset's own README

The ticket asserts that the CAIDA ⋈ delegated-stats path is AS-keyed. It is, and the proof is
stronger than a coverage statistic — it is in the format specification.

### 5.1 The format specification

[CAIDA `as-organizations` README](https://publicdata.caida.org/datasets/as-organizations/README.txt),
retrieved 2026-08-13, defines exactly two record formats:

> ```
> # format: aut|changed|aut_name|org_id|opaque_id|source
> # format: org_id|changed|org_name|country|source
> ```
>
> "opaque_id   : opaque identifier used by RIR extended delegation format"

`opaque_id` appears in the **AS** record format. It does not appear in the **organization** record
format. Since the opaque-id is the only key into `delegated-<rir>-extended-latest`, an organisation
with no ASN has no key, and cannot be joined to its prefixes at all — not imperfectly, but
structurally.

**[measured]** confirmation in the shipped data: of 98,597 `"type":"Organization"` records,
**0 carry an `opaqueId`**; of 121,988 `"type":"ASN"` records, 63,245 do.

```sh
grep '"type":"Organization"' latest.as-org2info.jsonl | grep -c 'opaqueId'   # -> 0
grep '"type":"ASN"'          latest.as-org2info.jsonl | grep -c 'opaqueId'   # -> 63245
```

### 5.2 Per-RIR opaque-id coverage, independently reproduced

**[measured]** — and identical to the figures #20 reported, which is a useful cross-check that both
passes read the data the same way:

| Source | ASN records | With `opaqueId` | Coverage |
|---|---:|---:|---:|
| AFRINIC | 2,763 | 2,763 | 100.00 % |
| APNIC | 30,735 | 25,927 | 84.36 % |
| ARIN | 34,123 | 34,107 | 99.95 % |
| JPNIC | 448 | 448 | 100.00 % |
| LACNIC | 14,279 | 0 | **0 %** |
| RIPE | 39,640 | 0 | **0 %** |

### 5.3 What the AS-keying actually costs, per region

This is where the ticket's framing needs splitting. #20's `unencumbered` win was the CAIDA join for
**AFRINIC and APNIC**. Applying §3's ASN-less rates to exactly those two regions:

| Region | Registered IPv4 holders | Reachable by the AS-keyed join | Unreachable by it |
|---|---:|---:|---:|
| APNIC | 25,778 | 8,629 (33.5 %) | **17,149 (66.5 %)** |
| AFRINIC | 2,874 | 2,402 (83.6 %) | **472 (16.4 %)** |
| **Both** | **28,652** | **11,031 (38.5 %)** | **17,621 (61.5 %)** |

**[measured]** So of the organisations #20's win was supposed to serve, **61.5 % get nothing from
it.** The ticket's suspicion is confirmed at that level.

But the loss is not evenly distributed, and for one of the two regions it is nearly harmless — which
§6 establishes.

---

## 6. The name-keyed paths are not AS-keyed, and they work ASN-less

The ticket generalises from "#20's path is AS-keyed" to a worry about the org→prefix capability as a
whole. That generalisation does not hold. Every registry path except the CAIDA join keys on
**organisation name or handle**, and the ASN is never on the critical path. Four organisations were
resolved end to end to demonstrate it — each one selected mechanically from the delegated files as a
holder of IPv4 with **no** ASN record under the same opaque-id, then followed through live RDAP.

| Region | Organisation (ASN-less, from delegated stats) | Chain | Result |
|---|---|---|---|
| **ARIN** | Medical Electronic Data Exchange, Inc. (`MEDE-1`) | `entities?fn=Medical Electronic Data Exchange*` → `entity/MEDE-1` | HTTP 200; entity embeds `199.164.228.0/24`, **0 `autnums`** |
| **RIPE** | Givaudan International SA (`ORG-GIS15-RIPE`) | `rdap.db.ripe.net/entity/ORG-GIS15-RIPE` | HTTP 200, 11,271 B; embeds `193.16.241.0`, **no `autnums` key at all** |
| **AFRINIC** | Columbus Stainless (Proprietary) Limited (`ORG-ZZ8-AFRINIC`) | `entities?fn=Columbus Stainless` — **one request** | HTTP 200 in 1.14 s; embeds `196.13.165.0` inline with `"autnums":[]` |
| **LACNIC** | `MX-ITME-LACNIC` | `rdap.lacnic.net/rdap/entity/MX-ITME-LACNIC` | HTTP 200, 6,290 B; embeds **two** networks (`200.23.57.0`, `200.34.128.0`) |
| **APNIC** | — | `rdap.apnic.net/entities?fn=Telstra*` | **HTTP 200 with `"entitySearchResults":[]`** |

Three findings fall out of that table.

**6.1 AFRINIC's ASN-less operators lose nothing in capability terms.** The AFRINIC organisation
`ORG-ZZ8-AFRINIC` holds a /24 and zero ASNs, and AFRINIC's `entities?fn=` returns its network inline
in a single keyless request. So the 472 AFRINIC organisations §5.3 counts as "unreachable" are
unreachable **only by the `unencumbered` tier**, because AFRINIC RDAP ships `operator-accepted`
pending the terms email #20 raised. What they lose is first-run depth, one click deep — precisely
the disposition ADR-0003 already defines and already requires onboarding to state.

**6.2 APNIC is the real casualty, and it is the only one.** #20 recorded that APNIC's RDAP entity
search is a no-op stub; **[measured]** this reproduces exactly — HTTP 200, empty
`entitySearchResults`, where RFC 9082 requires 501 for an unsupported query type. So APNIC has **no
name-keyed org→prefix path at any consent tier**, and the AS-keyed CAIDA join is the only instrument
that exists. The 17,149 ASN-less APNIC address holders are unreachable by *any* path, encumbered or
not. That is the one place where AS-keying removes a capability rather than deferring it.

**6.3 ARIN needs two requests where AFRINIC needs one.** `entities?fn=` returns the entity but does
**not** embed networks; `entity/MEDE-1` does. #20 reported the same asymmetry. Also **[measured]**:
a name search with no match returns **HTTP 404** (`fn=Arcanvs*`, 1,116 B), not an empty result
array — the opposite convention to APNIC's, and a client written to one will misread the other. A
404 that means "no such organisation" and a 200 that means "search unimplemented" must both be
mapped to `unknown`, never to an empty estate, or a drift product reports previously-known space as
removed.

---

## 7. ARIN reaches provider-aggregatable renters — which the ticket assumed nobody could

The ticket's premise is that *"an organisation holding provider-aggregatable space from an upstream,
with no ASN of its own, gets nothing."* For the CAIDA join, true. For ARIN's baseline path — the one
ADR-0003 already ships as the keyless default — **false**, and this is the most consequential
correction in the note.

### 7.1 Measured

**[measured]** `https://rdap.arin.net/registry/entities?fn=Acme*` returns HTTP 200, 263,638 B, and
**257 entities — of which 227 carry `C…` handles**. Those are ARIN **customer** objects: SWIP
reassignment records for downstream customers of an ISP, not RIR delegations. They do not appear in
`delegated-arin-extended-latest` at all.

Fetching one confirms it yields addresses:

```sh
curl -H 'Accept: application/rdap+json' https://rdap.arin.net/registry/entity/C00012855
# HTTP 200, 3,246 B
# "networks": [ { "cidr0_cidrs": [ { "v4prefix": "207.208.112.64", "length": 27 } ], ... } ]
```

A /27 reassignment, reachable by organisation name, keylessly, with no ASN anywhere in the chain.

### 7.2 Why the coverage exists, and where it stops

[ARIN NRPM §4.2.3.7.1](https://www.arin.net/participate/policy/nrpm/) (retrieved 2026-08-13):

> "Each IPv4 reassignment or reallocation containing a /29 or more addresses shall be registered via
> SWIP or a directory services system which meets the standards set forth in section 3.2.
> Reassignment registrations must include each customer name, except where specifically exempted by
> this policy."

The exemption is residential, not commercial — §4.2.3.7.3.2 lets an organisation substitute its own
name *"to maintain the privacy of their residential customers"*. A small **commercial** organisation's
reassignment therefore carries its real name by policy.

Two hard boundaries follow, and they matter as much as the capability:

- **The floor is a /29.** A customer holding a /30, or a single static address on a business
  broadband line, is below the registration threshold and appears nowhere. **[inferred]** that is
  the majority of very small commercial organisations with any static addressing at all.
- **The record is written by the upstream, not the operator.** Its accuracy is the ISP's, its name
  string is whatever the ISP typed, and ARIN's own RDAP marks stale contacts (the `MEDE-1` chain
  carried *"ARIN has attempted to validate the data for this POC, but has received no response from
  the POC since 2010-07-20"*). Under ADR-0002 a wrong expansion either probes a stranger or misses
  the operator's own space, so this is a safety input, not a convenience input.

### 7.3 What was not established

Whether the other four registries expose an equivalent name-searchable path to PA assignments was
**not** established here and is the sharpest follow-on question this note raises. RIPE registers PA
assignments as `inetnum` objects with `status: ASSIGNED PA`, but whether those carry an `org:`
attribute naming the end customer often enough to be searchable — and whether they are reachable
given that #20 measured `type-filter=inetnum` returning HTTP 400 on multi-word queries and RIPE's
RDAP entity search returning 500 — is open. §12 records it; §11 turns it into a ticket.

> **Answered in [§13](#13-are-provider-aggregatable-assignments-name-searchable-outside-arin)**
> ([#38](https://github.com/winniel123/verge-asm/issues/38), 2026-08-13). Short version: the *policy*
> generalises to all five RIRs — RIPE and AFRINIC compel it with **no** prefix floor, stricter than
> ARIN's /29 — and the *name search* generalises to RIPE and LACNIC but not to APNIC or AFRINIC. The
> `org:` attribute this paragraph guessed at turns out to be the wrong thing to have measured: it is
> populated on **6.98 %** of RIPE `ASSIGNED PA` objects and names a genuine End User on only 4.24 %.
> The route that works reads `descr:`, which is populated on 89.48 %. **Every non-ARIN path found sits
> behind `operator-accepted`, so the keyless default set does not move** (§13.8).

---

## 8. What an ASN-less operator's estate looks like in address terms

### 8.1 Cloud address space belongs, in registry terms, to 48 organisations

**[measured]** the three hyperscalers' published IPv4 ranges, deduplicated and merged:

| Provider | Unique IPv4 prefixes | Addresses |
|---|---:|---:|
| AWS | 7,883 | 102,427,110 |
| Azure | 43,800 | 53,026,405 |
| GCP | 997 | 19,091,840 |
| **Merged union** | **52,680** | **172,161,276** (10.26 /8-equivalents) |

Against a BGP-announced IPv4 space of **3,119,764,724 addresses** (**[measured]**, CAIDA prefix2as
2026-08-11, overlapping prefixes merged rather than summed), that is **5.52 %**.

Joining every one of those 52,680 prefixes against the merged delegated files by containment:

```
cloud IPv4 prefixes examined                        : 52680
  matched to a covering RIR delegation              : 52680 (100.00%)
  no covering RIR delegation record                 : 0
distinct (registry, opaque-id) holders covering them: 48
top holders by cloud-prefix count:
   32362  arin|b004b3ec…  -> MSFT-ARIN      (Microsoft)
    9502  ripencc|0a863480-…
    5119  arin|4a8a91b5…  -> Amazon (second ARIN handle; 3.0.0.0/8)
    2258  arin|20c786e8…  -> AMAZO-4-ARIN   (Amazon)
     990  arin|5cb965e9…  -> GOOGL-2-ARIN   (Google)
```

**Every published hyperscaler prefix has a covering RIR delegation, and the whole set belongs to 48
resource holders.** None of them is the operator. An org→prefix lookup keyed on the operator's name
returns nothing — not through a gap, but because the registry answer is *correct*: they do not hold
that space.

A detail worth keeping: **[measured]** the second-largest ARIN holder in that list, `4a8a91b5…`
(5,119 prefixes, including 3.0.0.0/8), carries **no ASN record in CAIDA under that opaque-id**. Even
Amazon splits address holding from ASN holding across handles. The ASN-less-address-holder shape is
not a small-business anomaly; it is how the registry system routinely works.

### 8.2 What ADR-0002 does with those addresses

ADR-0002 derives `Ownership` per `Address` from *"the operator's address-scope seeds plus RDAP
expansion of their organisation's ranges"*, and gates probing on it. For an operator whose estate
sits in cloud ranges, the RDAP-expansion limb contributes **nothing** — there are no ranges under
their organisation to expand. So:

- With **no** address-scope `Seed`, every cloud `Address` reached by following a name-scope `Seed`
  derives `third-party` (or `unknown`, which ADR-0002 treats as `third-party`). Probing is confined
  to *"only the ports the `Name` implies (443, 80)"*. Exposure attribution lands on the `Name` —
  *"this name is served from infrastructure you do not control"* — which ADR-0002 already calls a
  finding worth surfacing in its own right, available without scanning a stranger.
- With an address-scope `Seed` covering their allocated addresses, those become `owned` and the full
  tiered port sets apply.

**The escape hatch ADR-0002 names is the entire mechanism this operator needs**, and it already
exists. Nothing in the org→prefix family was ever going to add to it.

### 8.3 The awkward part, which is a `Seed` problem and not a registry problem

The addresses an operator holds in a cloud are **not stable the way a delegation is**. An elastic
address is a /32 that can be released and reallocated; managed load balancers and serverless
front-ends change address without notice; CDN-fronted names resolve into anycast ranges shared with
every other tenant. So the address-scope `Seed` an ASN-less cloud operator would declare is both
small and perishable, and declaring a whole provider range would be flatly wrong — it would assert
ownership over other tenants and, under ADR-0002, make their infrastructure eligible for the full
port sets.

**[inferred]** the honest primitive for this operator is the **name-scope `Seed`**, with the
address side left to derive as `third-party`, and with the handful of genuinely-held addresses
declared individually. That is a different first-run story from "we could not look up your prefixes",
and §10 takes it up.

---

## 9. Convenience or capability, per install shape

#20 asked this and answered it for the ASN-holding case. Answering it for the ASN-less case turns
out to answer it for everyone, because the discriminator is not the ASN — it is how many blocks the
organisation holds.

**[measured]** IPv4 records held per organisation, all five RIRs, split by whether the organisation
also holds an ASN:

| IPv4 records held | ASN-less holders (44,422) | ASN-holding holders (58,952) |
|---|---:|---:|
| exactly 1 | 35,129 (**79.1 %**) | 37,070 (62.9 %) |
| 2–3 | 7,191 (16.2 %) | 13,512 (22.9 %) |
| 4–10 | 1,709 (3.8 %) | 6,131 (10.4 %) |
| 11–50 | 336 (0.8 %) | 1,929 (3.3 %) |
| > 50 | 57 (0.1 %) | 310 (0.5 %) |

**95.3 % of ASN-less address holders hold three blocks or fewer**, and 79.1 % hold exactly one. An
operator can type one CIDR. For them the org→prefix path is a **convenience** — it saves a copy-paste,
and it cannot save them from forgetting a block they do not have.

Now invert it. #20's argument for the path was Mythic Beasts: 19 network objects under one handle,
more than anyone types from memory. Take that as the definition of a genuine **capability** — more
than ten blocks:

- Organisations holding **≥ 11** IPv4 records: **2,632 worldwide**, of which **2,239 hold an ASN**.
- **[measured] 85.1 % of the population for whom org→prefix is a real capability holds an ASN.**
- Widen to ≥ 4 records: 10,472 organisations, **79.9 %** hold an ASN.
- Read the other way: **97.5 % of all 103,374 registered address holders hold ten blocks or fewer**,
  and 89.9 % hold three or fewer. The capability case is 2.5 % of an already sub-1 % population.

This is the finding that most changes the ticket's conclusion. The AS-keying of #20's path looks
severe when counted in organisations (61.5 % of two regions' holders lost, §5.3) and is nearly
harmless when counted in **delivered capability**, because the organisations it misses are
overwhelmingly the ones holding a single block, who never needed a lookup.

### 9.1 Install shapes

| Install shape | Holds an ASN | What org→prefix gives them | Verdict |
|---|---|---|---|
| Estate entirely on AWS/Azure/GCP/SaaS | No | Nothing, correctly — the space is registered to 48 other organisations (§8.1) | **Not applicable.** Name-scope `Seed`s; ADR-0002 derives `third-party` |
| PA space rented from an upstream, ARIN region, ≥ /29 | No | The reassigned CIDR, via SWIP customer objects, keylessly (§7) | **Convenience**, and a real one |
| PA space, non-ARIN region | No | **Unestablished** (§7.3); nothing measured here reaches them | **Open** |
| One direct RIR block, no ASN — 35,129 organisations | No | One CIDR they already know | **Convenience** |
| 2–10 blocks, no ASN — 8,900 organisations | No | Modest; reachable by every name-keyed path, and by none of #20's | **Convenience** |
| > 10 blocks — 2,632 organisations, 85.1 % with an ASN | Mostly yes | The genuine "exceeds memory" case #20 was built for | **Capability** |
| Multi-subsidiary footprint | Usually yes | #20's finding stands: subsidiaries are where forgotten exposure lives | **Capability** |
| APNIC region, no ASN — 17,149 organisations | No | **Nothing, at any consent tier** (§6.2) | **Capability gap, region-specific** |

---

## 10. What this means for the onboarding message

[#15](https://github.com/winniel123/verge-asm/issues/15) / ADR-0003 made first-run depth
registry-dependent and required onboarding to say so. The ticket asks whether depth is also
**ASN-dependent**, giving the checklist a second axis.

**It is not, and adding that axis would be a mistake.** The measurements support a different shape:

1. **The axis that exists is "do you hold registry resources at all", not "do you hold an ASN".**
   For 99 %+ of the persona the answer is no (§3.1), and for them the whole org→prefix family is
   inapplicable rather than degraded. Annotating it as a missing capability would tell the modal
   operator that something is broken when nothing is — and ADR-0003's own rejected alternative,
   the *declared-status bar*, was rejected partly for asking a permanent onboarding question to buy
   very little. A permanent "do you have an ASN?" axis buys less.
2. **"Declare your CIDRs directly, this path is not for you" is the true message** for the ASN-less
   operator with a block or three — 95.3 % of ASN-less holders (§9). It is not a consolation prize.
   It is the correct instrument, and §8.2 shows the mechanism (the address-scope `Seed`) is already
   the escape hatch ADR-0002 names.
3. **There is exactly one honest missing-capability annotation, and it is regional, not
   ASN-shaped:** an **APNIC-region operator with no ASN** genuinely cannot have their registered
   prefixes looked up by any path this project has found (§6.2). That is 17,149 organisations, and it
   is a statement about APNIC's broken entity search, which no email fixes.
4. **AFRINIC-region ASN-less operators get the existing #15 annotation unchanged** — the capability
   exists at `operator-accepted`, one click deep, pending the AFRINIC terms email (§6.1). No new
   axis needed; the pattern ADR-0003 already established covers it exactly.

---

## 11. What this falsifies, and what it leaves standing

**Falsified — the ticket's strategic hypothesis.** *"#19 plus the three registry asks become
considerably more urgent than they currently look."* They do not. The encumbered paths (RIPE,
LACNIC) are name-keyed and therefore *do* serve ASN-less organisations (§6) — that half of the
hypothesis holds. But they serve them inside the same 128,233-organisation population, which is
**under 1 % of the persona** (§3.1). Unlocking RIPE and LACNIC converts a sub-1 % population from
`operator-accepted` to `unencumbered` depth. That is worth doing on its own merits and it is not
made more urgent by ASN-lessness. **[inferred]** if anything the priority falls, because §9 shows
the marginal organisation unlocked holds one CIDR.

**Falsified — the ticket's premise for one region.** *"An organisation holding provider-aggregatable
space from an upstream, with no ASN of its own, gets nothing from it."* True of the CAIDA join;
false of ARIN's shipped keyless path, which reaches SWIP customer objects down to a /29 and returns
their CIDR (§7).

**Confirmed.** #20's path is AS-keyed, provably and by construction (§5.1). The modal operator is
ASN-less (§4), and more so than the ticket supposed.

**Standing, and unchanged.** #20's per-region consent verdicts; its four "needs an email" tickets;
its client-correctness hazards, one of which (APNIC's 200-with-empty-array) is independently
reproduced here and turns out to be the single most consequential defect for this question (§6.2).

**Made stale.** Nothing in `CONTEXT.md` or the ADRs. ADR-0003's consequence that *"first-run
discovery depth is registry-dependent, and says so"* survives intact — this note argues against
*adding* an ASN axis to it, not against the consequence. §10.3 does add one regional annotation
(APNIC + no ASN) that ADR-0003's text does not currently anticipate.

---

## 12. Caveats — what could not be established

1. **Self-selection is unmeasured, and it is the biggest hole.** Every ratio in §3.1 uses the
   general business population as the denominator. Someone who installs a self-hosted AGPL attack
   surface tool is not a random SME. If verge-asm's actual installs skew heavily toward
   registry-holding organisations, the org→prefix path matters more than these ratios suggest.
   Nothing here measures that skew, and no public dataset obviously would. The direction of the
   conclusion is robust to a large skew — a 100× over-representation still leaves the path
   inapplicable to most installs — but the magnitude is not.
2. **Industry composition of registry holders is not measured.** §3.1 says "many holders are ISPs,
   universities and governments rather than small commercial organisations" and labels it
   **[inferred]**. CAIDA's AS-classification dataset is not present under
   `publicdata.caida.org/datasets/` (a request for `as-classification/` returned **404** on
   2026-08-13), and the obvious alternative — PeeringDB's `info_type` — is both self-selected toward
   peering networks and, per #20, ambiguous on terms. So the claim stands as inference only, and it
   makes the true small-business ASN rate *lower* than §3.1's upper bounds, never higher.
3. ~~**PA-assignment reachability outside ARIN is open** (§7.3). This is the one gap that could
   materially move the answer, because it governs whether the §7 correction is ARIN-only or
   general.~~ **Closed by [§13](#13-are-provider-aggregatable-assignments-name-searchable-outside-arin)**
   ([#38](https://github.com/winniel123/verge-asm/issues/38)). It did move the answer: §7's correction
   is general as a *capability* (three of five regions) and ARIN-only as a *default*, and §11's
   pricing of the RIPE and LACNIC asks is withdrawn as a result (§13.8.2). §13.9 records what §13
   in turn left open.
4. **Record counts are not CIDR counts.** §9 counts delegated-stats *records* per organisation. The
   NRO specification notes that *"records of the same type for the same opaque-id for the same date
   can be held to be a single assignment or allocation"*, and adjacent records may aggregate into
   one CIDR. So §9 slightly **over**-states how many distinct things an operator would type, which
   strengthens the "convenience" conclusion rather than weakening it.
5. **Cloud address share is not cloud tenancy share.** §8.1's 5.52 % is a measured share of
   *addresses*. It says nothing about the share of *organisations* hosted there, which is far higher
   because cloud and CDN addresses are densely multi-tenanted — one address serves many `Name`s.
   Quantifying tenancy would need a large passive-DNS corpus and was not attempted.
6. **Opaque-ids are per-file, not global.** Counting distinct opaque-ids within one file is exactly
   what the NRO specification licenses (§2.1). But an organisation holding resources in two regions
   counts twice in the "all five" row, and an APNIC organisation holding IPv4 from an NIR and an ASN
   from APNIC directly could carry two ids and be counted as ASN-less. §4 flags this as the reason
   to quote ARIN's 47.4 % and RIPE's 34.9 % rather than APNIC's 66.5 %.
7. **The four worked organisations in §6 are illustrations, not a sample.** Each was drawn
   mechanically from the delegated files (first ASN-less IPv4 holder encountered, small block
   preferred), but four is four. They establish that the name-keyed paths *can* resolve an ASN-less
   organisation; they do not measure how often the name string in the registry matches what an
   operator would type.
8. **RIPE was probed via RDAP only.** The name→handle step needs RIPE DB REST, whose terms #20
   found ambiguous and #19 is asking about. To avoid leaning on an unresolved source, §6 measured
   only `rdap.db.ripe.net/entity/<handle>`; the name→handle leg is #20's measurement, not this
   note's.

---

## 13. Are provider-aggregatable assignments name-searchable outside ARIN?

Research ticket [#38](https://github.com/winniel123/verge-asm/issues/38), answering the question §7.3
left open and §12.4 named as *"the one gap that could materially move the answer"*. Everything in this
section was retrieved with `curl` on **2026-08-13 UTC** from a single commercial IP, throttled to one
request every 3–6 s against live registry endpoints, and counted locally with `awk`. No summarising
fetch layer was used for any quoted text; every policy quotation below was verified against the
registry's own published document, not a secondary write-up — a discipline that earned its keep here
(§13.7).

### 13.1 Summary

The ticket offered *"ARIN is unique because NRPM is unique"* as an acceptable negative result. **It is
the wrong answer, and it is wrong in the more useful direction.** The two layers separate cleanly:

| | Answer |
|---|---|
| **Does the policy generalise?** | **Yes, completely.** All five RIRs compel registration of PA assignments naming the End User. NRPM is not unusual — it is the *median*. RIPE and AFRINIC compel it with **no prefix floor at all**, which is stricter than ARIN's /29 — §13.2 |
| **Does the name search generalise?** | **No — it splits 3–2.** ARIN, RIPE and LACNIC reach PA renters by name. APNIC and AFRINIC do not, at any consent tier — §13.3–§13.6 |
| **Does the *default set* change?** | **No, not at all.** Every non-ARIN path found here sits behind `operator-accepted`. The keyless default still reaches PA renters in ARIN only — §13.8 |
| **Is §7 a correction to one region or to the finding?** | **To the finding, as a capability; to one region, as a default.** The distinction is the whole of §13.8 and it is what the onboarding message turns on |

The headline is a population, and it is the largest number in this note:

> **[measured]** RIPE's `ASSIGNED PA` layer alone holds **1,127,002** assignments whose customer name
> string is unique in the entire database — against the **128,233** organisations worldwide that hold
> any RIR delegation (§3). The PA layer is roughly **8.8×** the entire registered population this note
> spent §3–§9 measuring. §26's "the registered population is tiny" stands; **"and therefore nobody is
> reachable" does not.** The reachable population is an order of magnitude larger than the delegation
> layer, and it is made of exactly the small commercial organisations ADR-0003 describes.

### 13.2 The policy layer — every RIR compels it, and two compel more than ARIN

The ticket asked for *"the equivalent of ARIN's NRPM §4.2.3.7.1"* per RIR, and treated its absence as
the likely finding. It is present everywhere. **[cited]**, each retrieved 2026-08-13 and quoted from
the registry's own document:

| RIR | Document | Floor | Compels the End User's *name*? |
|---|---|---|---|
| **ARIN** | [NRPM §4.2.3.7.1](https://www.arin.net/participate/policy/nrpm/) | **/29** | Yes — *"must include each customer name"* |
| **RIPE** | [ripe-733 §4.0, §6.2](https://www.ripe.net/publications/docs/ripe-733/) | **none** | Yes — *"registered separately with the contact details of the End User"* |
| **APNIC** | [APNIC-127 §3.1.2, §5.3.1](https://www.apnic.net/community/policy/resources/) | **> /30** | Yes, but **publication is opt-in and defaults to hidden** |
| **LACNIC** | [Policy Manual v2.21 §2.3.2.13](https://www.lacnic.net/innovaportal/file/680/1/manual-politicas-en-2-21.pdf) | **/29** | Yes — *"organization name"* required |
| **AFRINIC** | [CPM v1.6 §5.2.1.2, §5.3.2, §5.5.1.8](https://afrinic.net/consolidated-policy-manual.html) | **none** | Yes — *"registered with the contacts of the end-user"* |

ARIN's rule, re-verified verbatim today rather than carried over from §7.2:

> "Each IPv4 reassignment or reallocation containing a /29 or more addresses shall be registered via
> SWIP or a directory services system which meets the standards set forth in section 3.2.
> Reassignment registrations must include each customer name, except where specifically exempted by
> this policy."

**RIPE is stricter than ARIN, and this is the finding that reframes the question.** ripe-733 §4.0:

> "All assignments and allocations must be registered in the RIPE Database. This is necessary to
> ensure uniqueness and to support network operations.
>
> Only allocations and assignments registered in the RIPE Database are considered valid."

and §6.2, which is the direct analogue of NRPM §4.2.3.7.1 and carries **no prefix floor**:

> "When an End User has a network using public address space this must be registered separately with
> the contact details of the End User. Where the End User is an individual rather than an
> organisation, the contact information of the service provider may be substituted for the End
> Users."

Note the exemption is *individuals*, exactly ARIN's residential carve-out (§7.2) and LACNIC's
(§2.3.2.13.1.2). **A small commercial organisation is inside the mandate in all five regions.**

**LACNIC's §2.3.2.13 is a structural twin of NRPM §4.2.3.7.1** — same floor, same name requirement,
same residential exemption, and a tighter deadline than ARIN's seven days:

> "All IPv4 address block assignments of a /29 or larger block made by an ISP to customers connected
> to their network and users of services provided must be registered on LACNIC's WHOIS database no
> more than 7 days after the assignment."

> "Assignments registered on LACNIC's WHOIS database must include the following information regarding
> the assignee: organization name; address; administrative contact, technical contact, and contact in
> case of abuse […]"

**APNIC compels registration and then makes publication optional — this is the whole APNIC story**,
and it is a policy fact, not a protocol defect. APNIC-127 §3.1.2:

> "Account holders that receive an allocation from APNIC can choose whether their customer assignment
> registrations should be publicly available are not.
>
> If the account holder does not indicate a choice, or chooses to hide its customer assignment
> registrations, then those records will not be visible in the public Whois database. Whois queries on
> these records will return details of the allocation."

with the floor at §5.3.1: *"Delegations made to networks greater than a /30 for IPv4 and /48 for IPv6
must be registered."*

**AFRINIC compels it with no floor**, CPM §5.3.2: *"All allocations, PI assignments, PA assignments,
sub-allocations and other types of resource assignments will be registered in the AFRINIC database.
Any unregistered resources will be considered invalid."*

So the ticket's proposed negative result is falsified at the policy layer. **ARIN's coverage is not a
policy artefact peculiar to ARIN.** Whatever makes ARIN special is downstream of policy.

### 13.3 RIPE — `org:` coverage is 7 %, and it is the wrong attribute to have measured

The ticket asked to *"measure `org:` coverage across `ASSIGNED PA` rather than assuming it"*. Measured,
from the RIPE Database split file (`ripe.db.inetnum.gz`, 222,676,573 B, `Last-Modified: Wed, 12 Aug
2026 22:24:10 GMT`, retrieved 2026-08-13):

| RIPE `inetnum` status | Objects | with `org:` | with `descr:` |
|---|---:|---:|---:|
| **ASSIGNED PA** | **3,906,074** | **272,665 (6.98 %)** | **3,495,226 (89.48 %)** |
| ALLOCATED PA | 66,388 | 66,388 (100 %) | 3,803 |
| SUB-ALLOCATED PA | 17,710 | 10,284 | 5,740 |
| ASSIGNED PI | 19,469 | 19,462 | 3,080 |
| LEGACY | 105,373 | 33,498 | 93,843 |
| *(all statuses)* | 4,133,936 | — | — |

**`org:` is worse than 6.98 % makes it look, because most of it points at the LIR rather than the
customer.** Comparing the org-handle sets on `ASSIGNED PA` against those on `ALLOCATED PA` (which are
by definition LIR handles):

- **[measured]** 41,570 distinct org handles appear on `ASSIGNED PA`; 2,577 of them are also LIR
  handles.
- **[measured]** of the 272,665 `ASSIGNED PA` records carrying `org:`, **107,082 point at an LIR** and
  only **165,583 point at a non-LIR organisation** — i.e. a genuine End User is named by `org:` on
  **4.24 %** of RIPE's PA assignments.

**The registrations are there; they are just in `descr:`.** 89.48 % of `ASSIGNED PA` objects carry a
free-text `descr:`, and RIPE registers *far* below ARIN's floor, as ripe-733 §6.2's lack of a floor
predicts:

- **[measured]** **3,036,888** `ASSIGNED PA` objects (**77.75 %**) are a **/29 or smaller** — below or
  at the prefix where ARIN's SWIP obligation stops.
- **[measured]** **1,519,525** (38.90 %) are a **/30 or smaller**, which ARIN does not compel at all.

Quality of `descr:`, which matters because free text is not a name field — **[measured]** across the
3,487,319 `ASSIGNED PA` objects with a non-empty first `descr:` there are **1,418,684 distinct**
strings, of which **1,127,002 occur exactly once**. The head is placeholder-heavy and the tail is
real:

| first `descr:` | records |
|---|---:|
| `STATIC IP` | 232,890 |
| `OBS CUSTOMER` | 82,029 |
| `BT-ADSL` | 67,287 |
| `A1 TELEKOM AUSTRIA AG - BUSINESS CUSTOMER` | 50,940 |
| `NONE SPECIFIED` | 14,334 |

So **[inferred]** the 1,127,002 once-only strings are the honest estimate of distinct named End Users;
the 1,418,684 figure is an upper bound and the 291,682 repeated strings are mostly ISP bulk labels.

### 13.4 RIPE — the route is open, and #20's blocked route has two alternatives

#20 measured `type-filter=inetnum` returning HTTP 400 for multi-word org names and RIPE RDAP entity
search 500ing, and #38 asked *"whether another exists"*. **Two do.** All are `operator-accepted`
(RIPE Database, #19/#23 unanswered).

Worked end to end against **Sundstrom Safety AB**, selected mechanically from the dump as an
`ASSIGNED PA` /28 (`212.181.110.48 - 212.181.110.63`, `mnt-by: TELIANET-LIR`) — a small commercial
organisation renting PA space from Telia with no ASN, i.e. precisely the case #26 assumed unreachable:

```sh
# A. name -> org handle. HTTP 200, 2,445 B. #20 tried this with type-filter=inetnum.
curl -A "$UA" 'https://rest.db.ripe.net/search?query-string=Sundstrom%20Safety%20AB&type-filter=organisation&flags=r'
#    -> <object type="organisation"> ORG-SSA73-RIPE

# B. org handle -> prefixes, inverse lookup. HTTP 200, 2,360 B.
curl -A "$UA" 'https://rest.db.ripe.net/search?query-string=ORG-SSA73-RIPE&inverse-attribute=org&type-filter=inetnum&flags=r'
#    -> inetnum 212.181.110.48 - 212.181.110.63 | netname SE-SUNDSTROMSAFETY | status ASSIGNED PA

# C. #20's route, reproduced exactly. HTTP 400, 416 B, "invalid search key".
curl -A "$UA" 'https://rest.db.ripe.net/search?query-string=Sundstrom%20Safety%20AB&type-filter=inetnum&flags=r'
```

**The one-request route is full-text search, and it is the one that matters**, because it reads
`descr:` and therefore reaches the 89 % rather than the 4 %. Worked against
**Dierenartspraktyk Eiland Tholen W Brons** — a Dutch veterinary practice on a **/29**
(`193.172.56.56 - 193.172.56.63`), `ASSIGNED PA`, **carrying no `org:` attribute at all**:

```sh
# F. HTTP 200, 1,953 B, numFound=2.
curl -A "$UA" 'https://rest.db.ripe.net/fulltextsearch/select?q=%22Dierenartspraktyk+Eiland+Tholen%22&wt=xml&rows=5'
#    -> object-type inetnum | 193.172.56.56 - 193.172.56.63 | descr "Dierenartspraktyk Eiland Tholen W Brons"
#       | status ASSIGNED PA
```

That is the exact RIPE analogue of §7.1's ARIN result — an organisation with no registry resources of
its own, no ASN, and no `org:` record, found by name and yielding its CIDR.

**#20's RDAP 500 is real but narrower than recorded, and the difference is the wildcard.**
**[measured]**, same endpoint, same organisation, one variable changed:

| Request | Result |
|---|---|
| `rdap.db.ripe.net/entities?fn=Sundstrom%20Safety%20AB` (exact) | **HTTP 200**, 4,213 B, returns `ORG-SSA73-RIPE` |
| `rdap.db.ripe.net/entities?fn=Sundstrom%20Safety*` (wildcard) | **HTTP 500**, 703 B |
| `rdap.db.ripe.net/entities?fn=Hauraton%20GmbH%26CoKG` (exact, control) | **HTTP 200**, 4,908 B |
| `rdap.db.ripe.net/entities?fn=Dierenartspraktyk*` (wildcard, control) | **HTTP 500**, 703 B |

So RIPE RDAP entity search **works on exact names and 500s on wildcards** — the opposite convention to
ARIN, where §7.1's whole method is `fn=Acme*`. A client written against ARIN's wildcard idiom gets a
500 from RIPE and, per §6.3's rule, must map it to `unknown` rather than to an empty estate.

### 13.5 APNIC — the data exists, is named, and is unreachable by name at any tier

This is the region #26 §6.2 called "the real casualty", and #38 asked whether *any other* APNIC
interface reaches sub-allocations by name. **[measured]** from `apnic.db.inetnum.gz` (53,489,652 B,
`Last-Modified: Thu, 13 Aug 2026 15:02:41 GMT`, retrieved 2026-08-13):

| APNIC `inetnum` status | Objects | with `org:` | with `descr:` |
|---|---:|---:|---:|
| **ASSIGNED NON-PORTABLE** (the PA analogue) | **1,102,994** | **140 (0.013 %)** | **1,102,994 (100 %)** |
| ALLOCATED NON-PORTABLE | 87,014 | 78 | 87,014 |
| ALLOCATED PORTABLE | 47,195 | 20,659 | 47,195 |
| ASSIGNED PORTABLE | 14,513 | 7,289 | 14,513 |
| *(all statuses)* | 1,251,716 | — | — |

The `descr:` values are real customer names, at ARIN-like and finer granularity — sampled
mechanically, they include `Ebisu Servers Pte Ltd` on a /27, `jiangyin yongshenguanda metal Co.,Ltd`
on a /30, `DELHI PUBLIC SCHOOL FARIDABAD`, and `SMECDO2013091269817_DOUGHBAKE, INC.` on a **single
address**. **[measured]** 746,325 distinct first-`descr:` strings, **657,044 occurring exactly once**.

**Every name-search interface APNIC exposes fails on it.** All `operator-accepted`:

```sh
# APNIC RDAP entity search, wildcard AND exact. Both HTTP 200 with "entitySearchResults":[]
curl -H 'Accept: application/rdap+json' 'https://rdap.apnic.net/entities?fn=Ebisu%20Servers*'      # 200, 851 B, []
curl -H 'Accept: application/rdap+json' 'https://rdap.apnic.net/entities?fn=Ebisu%20Servers%20Pte%20Ltd'  # 200, 873 B, []

# APNIC whois, port 43, free-text name
printf 'Ebisu Servers Pte Ltd\r\n' >&3   # -> %ERROR:101: no entries found

# control, same object by address -- proves the record is present and named
printf '203.211.144.16\r\n' >&3
#   inetnum: 203.211.144.16 - 203.211.144.47
#   netname: EBISUSERVERS-SG
#   descr:   Ebisu Servers Pte Ltd # 225
#   status:  ASSIGNED NON-PORTABLE
```

This reproduces #20's and #26's 200-with-empty-array exactly, and extends it: the stub is not a
wildcard artefact (exact match is equally empty), and port 43 is no better. **[measured]** 657,044
uniquely-named APNIC PA assignments exist in a file we can download and cannot query by name through
any APNIC interface.

**The bulk file is the only instrument that reaches them, and its own header may forbid the use.**
`apnic.db.inetnum.gz` opens with:

> "Restricted rights.
>
> Except for agreed Internet operational purposes, no part of this publication may be reproduced,
> **stored in a retrieval system**, or transmitted, in any form or by any means, electronic,
> mechanical, recording, or otherwise, without prior permission of APNIC on behalf of the copyright
> holders."

Indexing a registry dump so it can be searched by name **is** storing it in a retrieval system. This
is map note *"Redistribution is a separate permission from use"* ([#27](https://github.com/winniel123/verge-asm/issues/27))
arriving on a path we would previously have classified as a plain query, and it is not resolved here
— see §13.9.

> **Amendment, [#46](https://github.com/winniel123/verge-asm/issues/46), 2026-08-13 — the quote above
> is truncated, and the omission changes the answer.** Two corrections, and the second is the
> substantive one.
>
> **The header has a third sentence.** **[re-measured]** against the live file (53,489,652 B,
> `Last-Modified: Thu, 13 Aug 2026 15:02:41 GMT`, unauthenticated HTTP `206` range request), the
> quoted two sentences are verbatim-accurate and the block ends early. In full, after *"…copyright
> holders."*: *"Any use of this material to target advertising or similar activities are explicitly
> forbidden and will be prosecuted. APNIC requests to be notified of any such activities or
> suspicions thereof."* That is the header's only **named** prohibited purpose, and it is not ours.
>
> **The header does not define *"agreed"*, and the page it is reproduced on does.**
> <https://www.apnic.net/manage-ip/using-whois/bulk-access/> carries this header verbatim under
> *"Restricted rights to the APNIC Whois Database"*, directly beneath: *"Bulk access to whois data,
> including domain objects, is available under an Acceptable Use Policy (AUP). … **Requestors must
> sign the AUP agreement and lodge it with APNIC.**"* and *"To request access to the Whois data for
> bulk download, please complete and return the APNIC whois data acceptable use agreement."* So the
> carve-out is a **conditional prohibition with a published, per-requestor route through it**, not an
> ambiguity — which is what [#46](https://github.com/winniel123/verge-asm/issues/46) ruled on, and it
> closes §13.9 item 3.
>
> Recorded as the **fifth** instance of §13.7's habit and the first that is a *truncation* rather than
> a misattribution: **verifying a quote against its source is not the same as verifying it against
> its context.** A clause quoted out of the document that qualifies it can be word-perfect and still
> wrong about what it permits.

### 13.6 LACNIC and AFRINIC — one works, one does not, and neither for the reason expected

**LACNIC reaches PA renters by name. It is ARIN's pattern exactly**, and it is the second region where
§7's correction holds. Ground truth first, then the search — `operator-accepted` throughout (#25
unanswered):

```sh
# whois.lacnic.net, port 43, by address: a compelled /29 reassignment with the customer's own handle
printf '190.196.20.1\r\n' >&3
#   inetnum:    190.196.20.0/29
#   status:     reallocated
#   owner:      Typack S.A.
#   ownerid:    CL-TYSA-LACNIC
#   inetnum-up: 190.196.20.0/25

# now by NAME, keylessly.  HTTP 200, 2,038 B -- exactly one entity
curl -H 'Accept: application/rdap+json' 'https://rdap.lacnic.net/rdap/entities?fn=Typack*'
#   -> "handle":"CL-TYSA-LACNIC", "fn":"Typack S.A."

# second request yields the prefix, same asymmetry as ARIN (#26 §6.3).  HTTP 200, 5,132 B
curl -H 'Accept: application/rdap+json' 'https://rdap.lacnic.net/rdap/entity/CL-TYSA-LACNIC'
#   -> "handle":"190.196.20.0/29", startAddress 190.196.20.0, endAddress 190.196.20.7
```

**Two LACNIC-specific defects bound the path, and both fail silently.** **[measured]**:

1. **The query matches a single token only. Any space returns an empty 200.** Consistent across three
   independent names: `fn=Typack*` → 1 hit; `fn=Typack%20S*` → **0**, 796 B; `fn=Typack%20S.A.` → **0**;
   `fn=Universidad*` → 100 hits; `fn=Universidad%20del%20Pacifico` → **0**; `fn=SEP*` → 80 hits;
   `fn=SEP%20Instituto*` → **0** (against `SEP Instituto Tecnológico de Mérida`, confirmed present via
   `entity/MX-ITME-LACNIC`). A client that sends the operator's full organisation name gets nothing,
   with no error.
2. **Results are hard-capped at 100**, stated in the response's own notices: *"Search results are
   limited to 100 per query."* `fn=Instituto*` and `fn=Universidad*` both returned exactly 100.

Both are the #26 §6.3 hazard in a new dress: **a 200 that means "your query shape was wrong" and a 200
that means "truncated" must both map to `unknown`, never to an empty estate.**

**AFRINIC does not reach PA renters, and the reason is that `org:` names the LIR.** **[measured]** from
`afrinic.db.gz` (10,399,463 B, retrieved 2026-08-13):

| AFRINIC `inetnum` status | Objects | with `org:` | with `descr:` |
|---|---:|---:|---:|
| **ASSIGNED PA** | **172,392** | **10,594 (6.15 %)** | **172,392 (100 %)** |
| ALLOCATED PA | 3,826 | 3,826 | 3,826 |
| SUB-ALLOCATED PA | 365 | 364 | 365 |
| ASSIGNED PI | 1,640 | 1,640 | 1,640 |

- **[measured]** only **105 distinct org handles** appear across all 172,392 `ASSIGNED PA` objects, and
  **102 of them are themselves allocation holders**. Of the 10,594 records carrying `org:`, **10,237
  point at an allocation holder** and only **357 (0.21 % of all PA assignments)** name a non-LIR
  organisation. AFRINIC's `org:` on a PA assignment is the ISP, not the customer.
- `descr:` is 100 % but frequently an internal code rather than a name — sampled mechanically:
  `ZOL V6 SK 564008`, `core-223`, `gbssat00077`, `Assignment made to SEACOM Customers.` The four most
  repeated are `XNEELO-TSCOLO` (8,782), `CLOUD INNOVATION LTD` (7,671), an abuse-contact sentence
  (6,808) and `MTN BUSINESS SA CUSTOMER` (2,980). **[measured]** 70,221 distinct strings, 59,978
  once-only.
- AFRINIC exposes **no full-text search**, so `descr:` is unreachable by name. `whois.afrinic.net`
  port 43 on `Cloud Innovation Ltd` returns the `organisation:` object alone (`ORG-CIL1-AFRINIC`) with
  no `inetnum`; `rdap.afrinic.net/rdap/entities?fn=` returns organisation entities with their **own
  delegations** (`154.192.0.0/11`, `156.224.0.0/11`, …), never the PA assignments beneath them.

So #26 §6.1's "AFRINIC's ASN-less operators lose nothing in capability terms" is correct **for
delegation holders** and does not extend to PA renters. AFRINIC is APNIC's situation reached by a
different route: the policy compels the registration (§13.2, and with *no* floor), the data is
published, and no interface indexes the name.

### 13.7 One quotation that did not survive checking

Worth recording because it is the fourth time this project has caught it. A secondary summary of
AFRINIC policy asserts that *"All IP block assignments equal or larger than a /29 … must be registered
and documented in AFRINIC Whois database no later than seven (7) business days after the assignment"*
— a clean ARIN/LACNIC-shaped rule, and the obvious thing to quote.

**[measured]** the current CPM (v1.6, retrieved 2026-08-13) contains **zero** occurrences of `/29` and
**zero** of `business days`. The sentence comes from a pre-CPM archived policy, not from the manual in
force. Had it been taken at face value, this section would have reported AFRINIC as having ARIN's
floor when it in fact has **none** — an error in the *lenient* direction that would have understated
AFRINIC's mandate while overstating its similarity to ARIN. Map note *"verify quotes against the
source, not a summarising layer"* ([#21](https://github.com/winniel123/verge-asm/issues/21)) paid for
itself again.

### 13.8 What this does to the default set — nothing, and that is the point

The ticket's stake was that *"if it generalises, the keyless default set reaches materially more
operators than [#20] or [#26] credited it with."* **The capability generalises; the keyless default
set does not move at all.** Consent tier per finding, which is the constraint #38 asked to have
recorded explicitly:

| Region | PA renters reachable by name? | Instrument | Consent tier | In the default set? |
|---|---|---|---|---|
| **ARIN** | **Yes** | `rdap.arin.net` `entities?fn=` → `entity/C…` | **`unencumbered`** | **Yes** (§7) |
| **RIPE** | **Yes** | `rest.db.ripe.net/fulltextsearch` (or organisation→inverse `org:`) | `operator-accepted` — #19/#23 *asked, no answer* | **No** |
| **LACNIC** | **Yes** | `rdap.lacnic.net` `entities?fn=` → `entity/<handle>` | `operator-accepted` — #25 *asked, no answer* | **No** |
| **APNIC** | **No** | none at any tier; bulk file only, and its header restricts storage | `operator-accepted` + an unresolved redistribution question | **No** |
| **AFRINIC** | **No** | none — `org:` names the LIR, `descr:` is not indexed | `operator-accepted` — #24 *asked, no answer* | **No** |

Three consequences bind:

1. **§7 is a correction to the finding, not to one region — but only as a *capability*.** Three of five
   regions reach PA renters by name. "ARIN is unique because NRPM is unique" is false twice over: NRPM
   is not unique (§13.2), and ARIN is not the only region that reaches them (§13.6). What ARIN is
   uniquely is **keyless**.
2. **The four registry non-answers cost more than §11 priced them at, and the increase is large.**
   §11 concluded that unlocking RIPE and LACNIC *"converts a sub-1 % population from
   `operator-accepted` to `unencumbered` depth"*. That is now measurably wrong about the population.
   The RIPE and LACNIC asks do not only buy depth inside the 128,233 delegation holders — they buy the
   **PA-renter layer beneath them**, which for RIPE alone is 1,127,002 uniquely-named assignments
   (§13.3). §11's "if anything the priority falls" should be read as **withdrawn for RIPE and LACNIC**
   and left standing for AFRINIC and APNIC, where the terms buy nothing because no name path exists to
   unlock.
3. **§10.3's "exactly one honest missing-capability annotation" is now two, and both are regional.**
   §10.3 named APNIC. **AFRINIC joins it** for the PA-renter case specifically: unlike §10.4's reading,
   the AFRINIC terms email does not buy PA-renter reachability, because AFRINIC's `org:` names the ISP
   (§13.6). §10.4 remains correct for AFRINIC *delegation holders*; it does not cover renters.

**The onboarding message is unchanged in substance and sharpened in honesty.** §10.2's *"declare your
CIDRs directly, this path is not for you"* survives, because for the modal cloud-resident operator it
is still true. What changes is what an install may claim once the operator *has* accepted terms: in
RIPE and LACNIC an org→prefix lookup now honestly reaches a renter with no registry resources of their
own, and in APNIC and AFRINIC it honestly does not.

### 13.9 What was not established

1. **`descr:` strings are not organisation names, and the gap is unquantified.** §13.3's 1,127,002 and
   §13.5's 657,044 count *strings unique in the database*, which is a proxy. Some are addresses, some
   are internal codes (§13.6 shows AFRINIC's are heavily so), and some organisations appear under two
   spellings. The direction is knowable — the once-only filter removes the bulk ISP labels that
   dominate the head — but no measurement here establishes what fraction of a once-only string set is
   a name an operator would type. **[inferred]** the true count is materially below 1,127,002 for RIPE
   and far below 59,978 for AFRINIC.
2. **LACNIC was not counted, only demonstrated.** LACNIC publishes no bulk database dump, so §13.6's
   LACNIC row has a worked chain and no population figure. Its 100-result cap makes enumeration by
   repeated name search impractical and, given #25's unanswered terms, inappropriate to attempt.
3. ~~**Whether an indexed local copy of a registry dump is "stored in a retrieval system"** (§13.5) is a
   licence question this note cannot answer, and it gates the *only* instrument that reaches APNIC's
   657,044 named PA assignments. It is precisely the shape of question [#27](https://github.com/winniel123/verge-asm/issues/27)
   settled for CAIDA — a build-time or storage-time act by the project rather than a runtime query by
   the operator — and it deserves its own ticket rather than an inference here. Note the RIPE and
   AFRINIC dumps carry their own terms which were likewise not assessed.~~
   **Closed by [#46](https://github.com/winniel123/verge-asm/issues/46)
   ([ADR-0018](../adr/0018-a-clear-conditional-is-not-an-ambiguity.md)), 2026-08-13.** It *is* storing
   it in a retrieval system, and the header permits that on a condition whose route is published: sign
   APNIC's bulk-access AUP and lodge it. The header is therefore **clear, not ambiguous** — no email
   was sent — and the tier would be `operator-credentialed`, a grant rather than a reading. The
   capability nonetheless **does not ship in v1**: `ftp.apnic.net` gates nothing, so the toggle would
   be a self-certification in front of an open door, which is ADR-0003's rejected declared-status bar
   rebuilt by us, and [#43](https://github.com/winniel123/verge-asm/issues/43) prices deferring a
   proposer at zero. **No third-party bulk registry dump enters v1**, which disposes of the unassessed
   RIPE and AFRINIC dump terms as a class rather than one at a time.
4. **The `org:`-vs-LIR test is structural, not semantic.** §13.3 and §13.6 classify an org handle as
   "an LIR" if it also appears on an `ALLOCATED` object in the same file. That is exact for AFRINIC's
   purposes and slightly *under*-counts LIRs in RIPE, where an LIR may hold allocations under a
   different handle than the one it stamps on assignments. The direction favours the conclusion being
   argued against, so RIPE's 4.24 % is an **upper** bound on genuine End-User `org:` coverage.
5. **Only IPv4 was measured.** All five policies quoted in §13.2 carry IPv6 clauses with their own
   floors (APNIC's `/48`, for instance). Nothing here measures IPv6 assignment registration or its
   searchability.
6. **The worked chains are three, not a sample.** Sundstrom Safety AB, Dierenartspraktyk Eiland Tholen
   W Brons and Typack S.A. were each drawn mechanically from a bulk file or a whois walk, and each
   demonstrates that the path *can* resolve. As §12.7 already says of §6's four, they do not measure
   how often the registered string matches what an operator would type — and §13.6's LACNIC
   single-token defect makes that gap operationally worse, not better.

### 13.10 Commands

```sh
UA='verge-asm-research/1.0 (spec research; contact logan.m.winnie@gmail.com)'

# --- policy layer (primary sources, all 2026-08-13) ---
curl -sSL -o ripe-ipv4-policy.html   'https://www.ripe.net/publications/docs/ripe-733/'
curl -sSL -o apnic-127.html          'https://www.apnic.net/community/policy/resources/'
curl -sSL -o afrinic-cpm.html -A "$UA" 'https://afrinic.net/consolidated-policy-manual.html'
curl -sSL -o arin-nrpm.html   -A "$UA" 'https://www.arin.net/participate/policy/nrpm/'
curl -sSL -o lacnic-221.pdf   -A "$UA" \
  'https://www.lacnic.net/innovaportal/file/680/1/manual-politicas-en-2-21.pdf'
pdftotext -layout lacnic-221.pdf lacnic-221.txt

# --- bulk registry data ---
curl -sS -o ripe.db.inetnum.gz  'https://ftp.ripe.net/ripe/dbase/split/ripe.db.inetnum.gz'
curl -sS -o ripe.db.organisation.gz 'https://ftp.ripe.net/ripe/dbase/split/ripe.db.organisation.gz'
curl -sS -o apnic.db.inetnum.gz 'https://ftp.apnic.net/apnic/whois/apnic.db.inetnum.gz'
curl -sS -o afrinic.db.gz       'https://ftp.afrinic.net/dbase/afrinic.db.gz'

# status tally + org:/descr: coverage + size distribution
# (ripecount.awk: objects are blank-line-delimited; key on status:, flag org:/descr:,
#  size from the inetnum: range; ASSIGNED PA bucketed at <=8 and <=4 addresses)
gzip -dc ripe.db.inetnum.gz    | awk -f ripecount.awk
gzip -dc apnic.db.inetnum.gz   | awk -f ripecount.awk
gzip -dc afrinic.db.gz         | awk -f ripecount.awk

# is an ASSIGNED PA org: handle an End User, or the LIR?
# (ripeorg.awk emits "<status>\t<org-handle>" per inetnum carrying org:)
gzip -dc ripe.db.inetnum.gz | awk -f ripeorg.awk > ripe-org-pairs.txt
awk -F'\t' '$1=="ASSIGNED_PA"{print $2}'  ripe-org-pairs.txt | sort -u > org-assigned-pa.txt
awk -F'\t' '$1=="ALLOCATED_PA"{print $2}' ripe-org-pairs.txt | sort -u > org-allocated-pa.txt
comm -12 org-assigned-pa.txt org-allocated-pa.txt | wc -l

# distinct / once-only customer-name strings
# (descr1.awk emits the first descr: of every inetnum whose status == WANT, upper-cased)
gzip -dc ripe.db.inetnum.gz  | awk -v WANT="ASSIGNED PA"            -f descr1.awk \
  | sort | uniq -c | sort -rn > ripe-descr-freq.txt
awk '$1==1' ripe-descr-freq.txt | wc -l
gzip -dc apnic.db.inetnum.gz | awk -v WANT="ASSIGNED NON-PORTABLE" -f descr1.awk | sort -u | wc -l
gzip -dc afrinic.db.gz       | awk -v WANT="ASSIGNED PA"            -f descr1.awk | sort -u | wc -l

# --- live paths, throttled 3-6 s between requests ---
curl -sS -A "$UA" 'https://rest.db.ripe.net/search?query-string=Sundstrom%20Safety%20AB&type-filter=organisation&flags=r'
curl -sS -A "$UA" 'https://rest.db.ripe.net/search?query-string=ORG-SSA73-RIPE&inverse-attribute=org&type-filter=inetnum&flags=r'
curl -sS -A "$UA" 'https://rest.db.ripe.net/search?query-string=Sundstrom%20Safety%20AB&type-filter=inetnum&flags=r'
curl -sS -A "$UA" 'https://rest.db.ripe.net/fulltextsearch/select?q=%22Dierenartspraktyk+Eiland+Tholen%22&wt=xml&rows=5'
curl -sS -A "$UA" -H 'Accept: application/rdap+json' 'https://rdap.db.ripe.net/entities?fn=Sundstrom%20Safety%20AB'
curl -sS -A "$UA" -H 'Accept: application/rdap+json' 'https://rdap.db.ripe.net/entities?fn=Sundstrom%20Safety*'
curl -sS -A "$UA" -H 'Accept: application/rdap+json' 'https://rdap.apnic.net/entities?fn=Ebisu%20Servers*'
curl -sS -A "$UA" -H 'Accept: application/rdap+json' 'https://rdap.apnic.net/entities?fn=Ebisu%20Servers%20Pte%20Ltd'
curl -sS -A "$UA" -H 'Accept: application/rdap+json' 'https://rdap.lacnic.net/rdap/entities?fn=Typack*'
curl -sS -A "$UA" -H 'Accept: application/rdap+json' 'https://rdap.lacnic.net/rdap/entity/CL-TYSA-LACNIC'
curl -sS -A "$UA" -H 'Accept: application/rdap+json' 'https://rdap.afrinic.net/rdap/entities?fn=Cloud%20Innovation%20Ltd'

# whois port 43 (bash /dev/tcp), one query per connection
exec 3<>/dev/tcp/whois.apnic.net/43   ; printf 'Ebisu Servers Pte Ltd\r\n' >&3 ; cat <&3
exec 3<>/dev/tcp/whois.apnic.net/43   ; printf '203.211.144.16\r\n'        >&3 ; cat <&3
exec 3<>/dev/tcp/whois.lacnic.net/43  ; printf '190.196.20.1\r\n'          >&3 ; cat <&3
exec 3<>/dev/tcp/whois.afrinic.net/43 ; printf 'Cloud Innovation Ltd\r\n'  >&3 ; cat <&3
```
