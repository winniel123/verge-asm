---
number: 227
title: "CAIDA publishes an org-name search, so the join replaces its first leg and keeps its second"
slug: caida-publishes-an-org-name-search-so-the-join-replaces-its-first-leg-and-keeps-its-second
date: 2026-09-07
status: accepted
source: fix
ticket: [1616, 1519]
proof: {none: "predates the governance SPEC"}
relations:
  - {kind: retires, adr: 223, clause: "3"}
  - {kind: rests-on, adr: 3}
  - {kind: rests-on, adr: 12}
---

# ADR-0227: CAIDA publishes an org-name search, so the join replaces its first leg and keeps its second


## Context

[ADR-0223](./0223-a-bar-is-authored-in-the-release-and-a-health-record-is-per-install-so-the-two-never-share-a-badge.md) §3
barred the `afrinic` and `apnic-caida` proposers on a measured finding: no published CAIDA endpoint
serves `/as2org/v1/org2ids`, and `api.caida.org` is NXDOMAIN. That ADR wrote its own reopening
condition, and it named the candidate that would meet it.

The candidate was measured on 2026-09-07, from the dev machine, against
`api.data.caida.org`. Every figure below is a live response, not a schema.

### The search path answers, keyless

```
GET https://api.data.caida.org/as2org/v1/search/?name=Safaricom
→ 200, application/json, 85,166 B, 0 redirects, no credential sent
```

`data.caida.org/as2org/v1/search/` answers `401` and stays refused. A move to it would leave
`unencumbered` for `operator-credentialed`, which ADR-0003 governs and which no ticket authorised.
`api.data.caida.org` sends no `WWW-Authenticate` and needs no key, so the tier does not move.

### The envelope and the record

```json
{"totalCount":316,"pageInfo":{"first":500,"offset":0,"hasNextPage":false},"errors":null,
 "data":[{"score":23,"asn":"37061","asnName":"Safaricom","country":"KE","source":"AFRINIC",
          "orgId":"33771","orgName":"Safaricom Limited","opaqueId":"F3682104_AFRINIC", …}]}
```

The identifier is `opaqueId` — camelCase, singular, one per record, and suffixed with the RIR.
`caidaOrgIDs` decoded `{"opaque_ids": []string}` at the top level. The path, the envelope and the
field name all differ.

The `_<RIR>` suffix strips to field 8 of the extended delegated-stats file. The join reproduces the
worked examples in `docs/research/non-arin-prefix-coverage.md` §5.2 exactly:

```
name=Safaricom → AFRINIC opaqueIds F3682104, F36CA351
               → delegated-afrinic-extended-latest: F3682104 gives 10 IPv4 and 2 IPv6 rows
name=Seacom    → AFRINIC opaqueIds F365C741, F3670C40, F36F76E3 → 16 IPv4 and IPv6 rows
```

### The search is scored, so it answers wider than the query

`name=Seacom` returns 338 records: 284 AFRINIC, 28 RIPE and 26 ARIN. One ARIN record names
`seacom LLC`, which is a different company on another continent. One AFRINIC record carries the
`opaqueId` of `JBJ Internet (The SA Internet)`, whose name holds no `Seacom` at all.
`name=google` returns 2,541 records across five RIRs, and 146 of them carry an `orgName` that does
not hold `google`.

So the response is a scored full-text result and never an org-name match. A rule has to be chosen,
and #1553 refused to choose one inside a bug fix. §2 chooses it.

### Not every record carries a join key

Measured on the same responses: 133 of 316 `Safaricom` records carry no `opaqueId` key, and 62 more
carry it empty. `name=Telkom Kenya` returns 71 records and **none** carries an `opaqueId` — every
one of the 71 is an organisation record, and the file's organisation rows have never carried the
field. This is the same coverage limit ADR-0003 already prices at 84.36% of APNIC opaque ids.

### The published page size does not slice the response

`first` and `offset` are echoed into `pageInfo` and are **not applied** to `data`.
`first=2&offset=0` and `first=2&offset=2` each returned all 316 `Safaricom` records, and each
reported `hasNextPage: true`. `first=6000` clamped to `5000` in `pageInfo`, which fixes the cap.
A broad query fails rather than paginating: `name=Telecom` answered `500` at `first=100`, `500` at
`first=500` and `500` at `first=5000`.

### The Public-AUA

Re-read on 2026-09-07 and unchanged. It governs redistribution, so it bars bundling and not a
runtime fetch by the operator's own install. It applies unchanged to this endpoint and bars nothing
here.

## Decision

The two CAIDA proposers ship on again, against `https://api.data.caida.org/as2org/v1/search/`.
Both halves stay keyless, and the consent tier does not move. ADR-0223 §3's bar is retired, and the
`endpoint does not answer` reason leaves both entries.

A record contributes an `opaqueId` only when its `source` equals this proposer's RIR and its
`orgName` holds the query, case-folded. The rule errs wide, because a lost holder reads as absence.

Three failures stay loud. A transport failure, a malformed envelope, a server-stated error, a short
page, and a matched record set with no key each return an error. Only an empty answer is quiet.

The join keeps its second leg. The search returns no prefix, so the delegated-stats file is still
the only artefact that carries one.

Rejected: a URL swap. It trades a loud failure for a silent one.

### 1. The two CAIDA proposers ship on again, against `/as2org/v1/search/`

`caidaBase` moves to `https://api.data.caida.org/as2org/v1`, `orgIDs` reads `/search/?name=`, and
the decoder reads `opaqueId` per record. ADR-0223 §3's bar is retired, the `Barred` flag and the
`endpoint does not answer` reason come off both entries, and both return to `DefaultOn: true`
alongside ARIN. They are `unencumbered`, both halves are keyless, and the consent tier does not
move.

The delegated-stats leg is untouched. `ftp.afrinic.net` and `ftp.apnic.net` both answer, and they
are still the only artefact that carries a prefix.

### 2. The join key rule: this RIR, and a name that holds the query

A record contributes an `opaqueId` only when **both** hold.

- **Its `source` equals this proposer's RIR.** The search answers with every RIR. Without this, an
  ARIN company that shares a name would put its id into an AFRINIC file's join. The measured
  `seacom LLC` case is exactly that.
- **Its `orgName` holds the query, case-folded.** Substring and not exact, because the research's
  own example needs it: `Safaricom` is the query and `Safaricom Limited` is the org. Exact match
  returns nothing for it. Substring is the rule that reproduces the research, and it is the widest
  rule that still refuses a scored near miss.

The `_<RIR>` suffix is then stripped, because field 8 of the delegated-stats file does not carry it.

**The rule is deliberately loose rather than tight.** A tight rule loses holders, and a lost holder
reads as *no holder matched*, which is [#50](https://github.com/winniel123/verge-asm/issues/50)'s
hazard. A loose rule offers a wrong proposal, and ADR-0012 already rules that a wrong proposal is
declined and costs coverage rather than safety. The two failures are not symmetric, so the rule
errs wide.

### 3. Three failures stay loud, and only an empty result is quiet

This is the property ADR-0223 refused a constant swap to protect, and it is now the code's own
invariant. `Propose` returns `nil, nil` for *no holder matched* and it must reach that branch on
one ground only: CAIDA answered and named nobody.

Each of these returns an **error** instead:

| Condition | Why it is not an absence |
| --- | --- |
| A transport failure or a non-200 | The request never produced an answer |
| An envelope without `totalCount`, `pageInfo` or `data` | `encoding/json` drops an unknown key, so the old `opaque_ids` decoder read a live body as `nil` and returned *no holder matched* |
| A non-null `errors` member | The server states its own failure inside a `200` |
| Fewer rows than `totalCount`, with no next page | A partial join reads as a narrower estate |
| Records match the RIR and the name, and none carries an `opaqueId` | CAIDA holds the organisation and publishes no join key. That is a gap, and a gap is not an absence |

The last row is the one that costs. `name=Telkom Kenya` matches 71 AFRINIC records and yields no
key, so that lookup now reports an error from the AFRINIC proposer on every attempt.
`Registry.Propose` joins a source's error and keeps every other source's answer, so the cost is
this source's coverage and never the whole search. The operator reads *this source could not
answer*, which is true, in place of *nobody holds this name*, which is not.

### 4. `endpoint does not answer` stays defined and is claimed by nobody

ADR-0223 §2 made the bar reason data. That machinery is right and stays. No catalogue entry carries
this particular reason any more, and a test asserts that, so the next entry that earns it inherits
a reason nobody has quietly reused.

### 5. `/as2org/v1/search/` replaces the first leg. It retires no ADR, because ADR-0012 never made the claim

`internal/proposer/caida.go` carried the declaration comment *"no org-name search is published, so
two keyless sets are joined (ADR-0012)"*. #1616 asked whether a published org-name search falsifies
that premise and retires the join.

**ADR-0012 states no such premise.** It is *A proposer is not a `Source`*. It rules that a proposer
carries `consent` alone, and it names no join, no CAIDA, no `org2ids` and no org-name search. The
premise was the comment's own, and the citation attached it to an ADR that never argued it. The
comment is deleted, and it is replaced by the constraint that is actually true and actually
external.

**The join stands, and only its first leg moves.** The ground is not a premise about what is
published. It is what each artefact carries. `/as2org/v1/search/` returns `orgName`, `orgId`,
`asn`, `country`, `source` and `opaqueId`. It returns **no prefix**, on any record, at any page
size. The extended delegated-stats file is still the only artefact that carries one. So the second
keyless set is not replaceable by this endpoint, and two sets are still joined.

## Consequences

- **#1519's first box is met.** Both proposers now reach a host that resolves and answers, and the
  `opaque_ids` decode is confirmed against a real body — confirmed **absent**, and replaced by
  `opaqueId`. The ticket's other three boxes were already closed by #1553 and ADR-0223.
- **A scored search reaches the operator as a proposal, never as a fact.** ADR-0012 already holds
  the line: a proposal enters the estate only when the operator confirms it into a `Seed`. §2's
  loose rule is priced against that ruling and against nothing weaker.
- **`pageInfo` is handled, and today it never fires.** The reader pages on `offset` while the server
  has sent fewer rows than it counted, capped at `first=5000` and at eight pages. The measured
  server sends every row on the first page, so the loop exits after one request. If CAIDA starts
  honouring `first`, the reader follows the pages rather than silently proposing a prefix of them.
- **A broad query fails loudly.** `name=Telecom` answers `500`, and a `500` is an error here. The
  operator reads a failure rather than a truncated estate.
- **The `endpoint does not answer` badge disappears from the `/sources` modal.** Only *excluded on
  terms* and *no runner ships* render in v1, which is the state before #1519 was filed.
- **The health surface [#1615](https://github.com/winniel123/verge-asm/issues/1615) is unaffected.**
  ADR-0223 §4 is untouched. The Operational record it specifies is the surface that would show an
  operator that this endpoint had failed again, and this ADR gives it two live sources to record.

## Reopening condition

Reopen this ADR when any one of these holds.

1. **A CAIDA artefact carries an `opaqueId` for organisation records, or a second lookup maps an
   `orgId` to one.** §3's last row costs a lookup whenever CAIDA holds the org and publishes no key,
   and `name=Telkom Kenya` shows that is not rare. `/as2org/v1/asns/{asns}` and the bulk
   `as-org2info.jsonl.gz` file both carry `opaqueId` on ASN records, and either could close the gap
   at a cost this ADR did not price.
2. **A wrong proposal reaches an operator often enough to be a complaint.** §2 errs wide on
   ADR-0012's ground that a wrong proposal is declined. A measured rate of wrong proposals would
   re-price that, and an exact-match or a score-threshold rule becomes worth its own measurement.
3. **CAIDA starts applying `first` to `data`.** The reader already pages, but the page size, the
   eight-page cap and the `500` on a broad query were all measured against a server that ignores
   `first`. Each of the three needs re-measuring if that changes.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Swap `caidaBase` and leave `caidaOrgIDs` alone | The exact hazard ADR-0223 refused. `encoding/json` drops the unknown keys, `OpaqueIDs` decodes to `nil`, and a live `200` reads as *no holder matched* |
| Move to `data.caida.org` | Answers `401`. Not keyless, so an ADR-0003 consent change and not a URL change |
| Wire the bulk `as-org2info.jsonl.gz` file instead | 4.6 MB per query per region. It needs a cache policy and a refresh cadence nobody has specified, and the keyless query endpoint needs neither |
| Exact org-name match | Returns nothing for `Safaricom`, which is the research's own worked example. A tight rule loses holders, and a lost holder reads as absence (#50) |
| No name filter at all, and let the delegated-stats join decide | `seacom LLC` is an ARIN company, and a same-RIR near miss under a different name would propose a stranger's prefixes with no filter to stop it. The measured `JBJ Internet` record is that case |
| Filter on `score` instead of on the name | The threshold is a number nobody has grounded. ADR-0223 §4 refused a threshold on the same reasoning, and the measured scores range from 0 to 30,752 with no documented scale |
| Return an empty result when no record carries an `opaqueId` | Turns a gap into *no holder matched*, which is the false absence this whole repair exists to refuse |
| Keep the two proposers barred and record the endpoint only | ADR-0223 §3 wrote its own reopening condition and named this endpoint. Meeting the condition and not acting on it leaves a bar standing on a finding that no longer holds |

## Amendment — [#1634](https://github.com/winniel123/verge-asm/issues/1634), 2026-09-08: `asns/` carries the key an organisation record lacks, so the join reads it there and §3's last row fires only after it

Reopening condition 1 is met and measured. §3's last row stands as written above, and this section
supersedes it in place: the row's error now fires only after a second CAIDA leg has been read. The
original text is not rewritten, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).

### What was measured, 2026-09-08, `api.data.caida.org`

Every figure is a live response, not a schema.

| Request | Answer |
| --- | --- |
| `GET /as2org/v1/search/?name=Telkom%20Kenya` | `200`, 19,656 B, 71 rows, all AFRINIC organisation records. Keys: `changed`, `country`, `date`, `members`, `orgId`, `orgName`, `score`, `source`, `ts`. **No row carries `asn` or `opaqueId`.** `members` is a list of ASN strings: `30994` on 69 rows and `12455` on 48 |
| `GET /as2org/v1/asns/30994` | `200`, 435 B, the same envelope, one ASN record: `asn`, `asnName`, `source: AFRINIC`, `orgId`, `orgName: Kenyan Post & Telecommunications Company / Telkom Kenya Ltd`, **`opaqueId: F367736D_AFRINIC`** |
| `GET /as2org/v1/asns/12455` | `200`, one ASN record, the same `opaqueId: F367736D_AFRINIC` under the same `orgId` |
| `GET /as2org/v1/asns/30994,37061` | `200`, `totalCount: 0`, `data: []`. **A comma list is not a list.** It reads as an unknown ASN |
| `GET /as2org/v1/asns/?asns=30994,37061` | `200`, 146,982 B, `totalCount: 120793`, `hasNextPage: true`. The query form ignores the filter and lists the whole ASN table |
| `GET /as2org/v1/asns/37476` | `200`, `totalCount: 0`, `data: []`. The `Seacom` capture names this ASN on an organisation record, and the ASN table does not hold it |
| `GET /as2org/v1/search/?name=Safaricom` | 280 AFRINIC name-matched rows: 121 keyed, 159 unkeyed. The unkeyed organisation records name 3 distinct members (`328988`, `33771`, `37061`), and all 3 already sit on keyed ASN records in the same response |

The issue guessed that an organisation record carries `asn`. It does not. It carries `members`, and
that is the field the second leg reads. `as-org2info.jsonl.gz` was not fetched: `asns/` closes the
gap, and the bulk file's 4.6 MB per query is already priced and rejected above.

### The decision

**The join reads a second CAIDA leg before §3's last row fires.** After `search/` has been paged,
every organisation record that passes §2's rule and carries no `opaqueId` contributes its `members`.
Each distinct member ASN that no keyed ASN record in the same response already carries is read once
through `/as2org/v1/asns/{asn}`. An ASN record it returns contributes its `opaqueId` under §2's rule
unchanged: the same RIR, and an `orgName` that holds the query. The recovered keys merge into the
one candidate set, and the delegated-stats leg is untouched.

**The cost.** `search/` is one request. The second leg is one request per distinct unkeyed member
ASN, because the endpoint takes one ASN per call and treats a list as no ASN. Measured:
`Telkom Kenya` costs 2, `Seacom` costs 1, `Safaricom` costs 0. The leg is capped at 32 lookups.
A search that names more fails loudly before the first lookup, because a partial second leg would
read as a narrower estate, which is §3's fourth row. The measured need is 0 to 2, so the cap does
not fire today.

**§3's last row now reads:** records match the RIR and the name, none carries an `opaqueId`, and
`asns/` recovers none for the members they name. The error stays, because CAIDA still holds the
organisation under no key and a gap is still not an absence. It is now typed as
`proposer.ErrNoJoinKey`, so a caller can tell the gap from a failure.

### The loud-failure property is kept

A transport failure on the second leg is an error and never an empty result. The tests hold the
property on eight grounds for `asns/`: a transport error, a non-200, the retired `org2ids` envelope,
an HTML body, a non-null `errors` member, fewer rows than `totalCount`, an ASN record with no key,
and an unknown ASN. The delegated-stats file is not fetched after any of them. A search over the
32-lookup cap is a ninth ground.

### What the health surface records

[ADR-0223](./0223-a-bar-is-authored-in-the-release-and-a-health-record-is-per-install-so-the-two-never-share-a-badge.md)
§4's record holds `ok` or `error`, and nothing else. When CAIDA answered correctly on both legs and
the join still found no key, the source did not fail, so the attempt records **`ok`** and the
consecutive-failure count does not accrue. The lookup itself still reports the error to the
operator, because the gap is real. A true transport failure on either leg records `error`, as
before. No third outcome is added: a third value would need a migration, an amendment to ADR-0223
§4 and a label nobody has ruled on, and the two-value record already states the true fact, which is
that the source answered.

### What this amendment does not withdraw

- **§2 stands whole.** The second leg applies the same rule to an ASN record `asns/` returns.
- **§3's first four rows stand whole**, and they now hold on both legs.
- **The Consequences bullet on the health surface stands.** The record is unchanged. This amendment
  rules which outcome one state maps to, and reaches nothing else.
- **Reopening conditions 2 and 3 stand.** Condition 1 is discharged here.
