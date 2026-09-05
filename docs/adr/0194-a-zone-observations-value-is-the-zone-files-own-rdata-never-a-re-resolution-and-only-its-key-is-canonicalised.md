# ADR-0194: a `zone` observation's value is the zone file's own rdata, never a re-resolution or a resolver normalisation, and only its key is canonicalised

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1308 ADR gaps: internal/scan (CT and zone Scans)](https://github.com/winniel123/verge-asm/issues/1308), gap 7
- **PR that deleted the comment:** [#1307](https://github.com/winniel123/verge-asm/pull/1307)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0007](./0007-drift-is-a-timeline-of-spans.md), which keys a timeline **per source** and rules that a source conflict is *reported, never resolved*. It is why two timelines exist on `dns-record` at all. It does not say what either of them carries
- **Rests on:** [ADR-0011](./0011-a-facet-is-six-parts.md), whose Rationale names the operator's zone file against our resolver as *the only two-source facet in v1*, and splits the fold into a **decoder per `(facet, source)`** and a **canonicaliser per facet**. It rules the shape of the fold. It does not rule what the zone decoder may put into it, and §4 below shows the shipped decoder does not satisfy it
- **Bounded by:** [ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md), which rules that *"a rule may read which names a zone contains. It may not read what records it holds for them."* It bounds what a **signal rule** may do with these values. It does not say whether the values exist or what they hold
- **Bounded by:** [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md), which fixes DNS presentation format as the corpus's input medium — *"a row is written the way a zone file or `dig` output is written"*. It rules the golden corpus, not this observation

## Context

`internal/scan/zone.go:240` carried this, until #1307 shortened it:

```go
// zoneValue is the value a zone `dns-record` observation carries: the RRset's
// rdata, as strings. It is deliberately the zone file's own words rather than a
// re-resolution, so the timeline reflects what the operator declared.
```

One line survives, at `internal/scan/zone.go:171`:

```go
// The rdata is the file's own words, never a re-resolution, so the timeline is what was declared.
```

### The package cannot re-resolve, and that is measurable rather than asserted

`internal/scan/zone.go` imports `encoding/json`, `fmt`, `strings`, `time` and `internal/wire`.
**No `net`, no `net/http`, no resolver package, no DNS library.** `RestateZone` takes bytes and a
supply instant and returns records. There is no seam through which a resolution could enter.

The Scan around it is the same shape. `internal/queue/zone.go:35` `completeZone` decodes the scope,
calls `RestateZone`, and writes. The scope, `zoneScope`, carries `domain`, `supplied_at` and
`content`. The Scan is worker-read and carries no `Vantage`.

### What the parser does and does not touch

`zoneParser.parse` (`internal/scan/zone.go:202`) walks a logical line and produces three fields.

| Field | Treatment |
| --- | --- |
| **owner** | `resolveName` — trimmed, **lower-cased**, `@` resolved to the origin, a relative name suffixed with `$ORIGIN`, the trailing dot removed |
| **qtype** | `strings.ToUpper`, then checked against `knownTypes`; an unknown type **skips the line** |
| **rdata** | `strings.TrimSpace(strings.Join(fields[1:], " "))` — **the remaining fields, joined, and nothing else** |

Class tokens (`IN`, `CH`, `HS`, `CS`) and TTL tokens are stepped over before the qtype, so they do
not reach the rdata. Comments are stripped at `;` and a parenthesised continuation is flattened to
one line. **After that the rdata is the operator's characters, in the operator's order, with the
operator's spacing collapsed to single spaces.**

`zoneValue` (`internal/scan/zone.go:173`) is `{ RRs []string }`, one string per record in the RRset,
in file order.

### Two sources really do write this facet

`internal/queue/zone.go:23` writes `Facet: "dns-record"`, `SubjectKind: "name"`,
`Discriminator: r.Qtype`, `Source: scan.ZoneSource`, `VantageID: pgtype.Int8{}`, and
`ObservedAt: r.ObservedAt` — the operator's supply instant, not the worker's read.

`internal/measure/resolutionwalk/emit.go:47` writes the same facet, same subject kind, same
discriminator, from `resolution-walk`.

ADR-0011 calls this the only two-source facet in v1, and ADR-0020 confirms it: *"exactly one — the
operator's zone file against our own resolver, on `dns-record`."*

### The two values are not in the same value space

This is the measurement this ADR has to carry, because it decides what §4 rules.

| Source | Value |
| --- | --- |
| `zone` | `{"rrs":["1.2.3.4","5.6.7.8"]}` — a list of **strings** |
| `resolution-walk` | `{"rrs":[{"name":…,"type":…,"data":…}],"delegation":…}` — a list of **objects** |

`cmd/web/signals.go:32` declares one `dnsRecordValue` for both, with `RRs []struct{ Name, Type, Data
string }`. Decoding the zone shape into it **fails**, and both call sites discard the error —
`decodeDNSRecord` (`cmd/web/subjects.go:51`) with `_ = json.Unmarshal`, and the signal fold
(`cmd/web/signals.go:749`) with the same. The measured result of decoding
`{"rrs":["1.2.3.4","5.6.7.8"]}` into it is:

```
err = json: cannot unmarshal string into Go struct field dnsRecordValue.rrs …
len(RRs) = 2
RRs = [{Name: Type: Data:} {Name: Type: Data:}]
```

Two records, every field blank.

**And the two sources compete for the same row.** `ListNameDNSRecords`
(`internal/db/signals.sql.go:109`) keys its cadence CTEs per source and then collapses:

```sql
SELECT DISTINCT ON (o.subject_key, o.discriminator) …
ORDER BY o.subject_key, o.discriminator, o.observed_at DESC, o.id DESC
```

**`source` is not in the `DISTINCT ON` key.** A zone file supplied more recently than the last
resolver walk therefore wins the slot for that `(name, qtype)`, in both consumers of that query:
`cmd/web/subjects.go:1016` renders blank rows on the asset page, and `cmd/web/signals.go:728` fails
to read a `CNAME` target or an `NS` lame flag it would otherwise have read from the resolver.

## Decision

> **A `zone` `dns-record` observation carries the RRset's rdata exactly as the operator's zone file
> spells it. It is never re-resolved and never normalised through the resolver, so the `zone` timeline
> reports what the operator declared rather than what the network answers. The observation's **key** —
> owner name and qtype — is canonicalised, because it must join the resolver's. The **value** is not.
> The value's *shape* is still the facet's value space, per ADR-0011's decoder rule, and the shipped
> `[]string` shape does not satisfy that.**

### 1. The rdata is the file's own words

No query is issued to check it. No provider convention is decoded. No address is reparsed and
re-emitted in RFC 5952 form. What the operator wrote for `MX`, `TXT`, `CAA` or `SRV` is what the
observation carries.

The zone file is the operator's **declaration**, and a declaration's whole evidential value is that
it says what the operator believes. `internal/scan/zone.go:1` already states the sibling rule for
time — the observation is stamped at the **supply instant**, because *"re-parsing unchanged bytes on
a cadence would manufacture a current observation of a stale fact."* Passing the value through the
resolver would manufacture the same falsehood in the other dimension: a `zone` row asserting what
the network answers, dated to when the operator handed us a file.

### 2. The key is canonicalised, and that is not an exception to §1

The owner name is lower-cased and made absolute against `$ORIGIN`. The qtype is upper-cased. Both
are **necessary**, and neither touches the value.

`subject_key` and `discriminator` are join columns. The resolver writes `resolution-walk`'s
observations under `resolutionwalk.CanonicalName`, and `internal/scan/crtsh.go:157` records what
happens when two producers of one key disagree: *"a parallel Unicode fold here breaks the admission
hop's join on subject_key (ADR-0107, #256)."* A zone row keyed `WWW.Example.COM.` would sit on a
different timeline from the resolver's `www.example.com` and the two would never be compared.

**The line is exactly where the fold stops being a key and starts being a claim.** Case-folding a
domain name preserves its denotation — DNS names are case-insensitive. Rewriting an operator's
`MX 10 mail.example.com.` into something a resolver would have returned does not.

### 3. The comparison is the whole point, and normalising one side destroys it

ADR-0007 keys the timeline per source *"so a zone file cannot keep a dead name alive"* and rules a
source conflict **reported, never resolved**. ADR-0011 §Rationale says why the two values must stay
distinct: one canonicaliser over both shapes would mean *"fixing the zone-file parser moves its
version and `Break`s every `dns-record` timeline in the estate — including the ones our own resolver
produced, which nothing touched."*

A `zone` value normalised through the resolver is worse than that. It would not merely couple the two
timelines. **It would make them agree by construction**, so the one conflict pair v1 has would never
report anything. The value of the operator's declaration is precisely the part that can differ.

### 4. The value's shape is the facet's, and the shipped shape is a defect

**§1 is a rule about content, not a licence about shape.** ADR-0011 already rules the shape, and it
is worth quoting because the shipped code is on the wrong side of it:

> A **decoder** per `(facet, source)` turns whatever the source emits into the facet's value space.

and its reason for splitting the fold that way:

> One canonicaliser per `(facet, source)` … costs more than it saves: the two sources then produce
> values in different spaces, and ADR-0007's *source conflict is reported, never resolved* has nothing
> to report with. You cannot report a conflict between values you cannot compare.

`{"rrs":["1.2.3.4"]}` and `{"rrs":[{"name":…,"type":…,"data":…}]}` are two different spaces, so the
condition ADR-0011 built the decoder split to avoid is the condition that shipped.

**The rule stated forward: the zone decoder's job is to put the operator's own rdata into the
`dns-record` value space, unchanged in content.** The two are not in tension. An RR object's `data`
field holds the operator's characters; its `type` is the qtype the parser already read; its `name` is
the owner it already canonicalised. Nothing about §1 requires the flat list.

**This ADR does not change the shape here.** §Consequences names the ticket.

### 5. What this rule does not reach

- **What a signal rule may do with these values.** ADR-0020 rules that: *"a rule may read which names
  a zone contains. It may not read what records it holds for them."* The signal fold honours it today
  — `cmd/web/signals.go:768` reads `ListZoneDeclarations` for the **name set** and never reads a zone
  `dns-record` value. That boundary is untouched.
- **Which lines the parser accepts.** An unknown qtype, a rdata-only line with no prior owner, and a
  line with fewer than two fields are all skipped and reported through `RestateZone`'s second return
  (#869). That is a coverage rule, not a value rule.
- **TTL.** ADR-0011 excludes TTL from the `dns-record` value for every source, deliberately, and
  names the zone file as one reason. The parser steps over it and this ADR does not reopen it.
- **The supply instant.** `internal/scan/zone.go:1` and v1 spec §3.4 rule the timestamp.
- **Provider pseudo-records.** ADR-0020 measured `ALIAS`/`ANAME`, apex CNAME flattening and
  provider-side signing and refused to decode them. The parser skips what it does not know, which is
  that refusal holding.

## Consequences

- **This ADR changes no Go code.** `RestateZone` and `zoneParser` are correct on content.
- **`internal/scan/zone.go:171` gains this ADR's citation** on the surviving line that states the
  rule. Recorded in this issue's manifest.
- **The `zone` decoder emits a value outside the `dns-record` value space. That is a defect against
  [ADR-0011](./0011-a-facet-is-six-parts.md)'s decoder rule, and it ships as its own ticket.**
  `zoneValue` becomes `{ RRs []RR }` carrying the same `name`, `type` and `data` fields
  `resolution-walk` emits, with `data` holding the operator's unchanged rdata. It moves the `zone`
  decoder's version and `Break`s the `zone` `dns-record` timelines, and — by ADR-0011's own argument
  for splitting decoder from canonicaliser — it `Break`s **only** those, never the resolver's.
- **`ListNameDNSRecords` collapses two sources into one row, and that is a second defect this ruling
  exposes.** Its `latest` CTE takes `DISTINCT ON (subject_key, discriminator)` with no `source`, while
  the `cover` and `live` CTEs above it key cadence per source. The newest `observed_at` wins, so a
  freshly supplied zone file silently displaces the resolver's answer for that `(name, qtype)`.
  **It ships as its own ticket**, and the ticket is downstream of the one above: what the read should
  do with two comparable values is ADR-0007's *report, never resolve*, and that is only answerable
  once the values are comparable.
- **Today the collision renders as blank rows and lost facts.** With the shapes as they are, a zone
  row that wins the slot gives `cmd/web/subjects.go:1029` records whose `Type` and `Value` are empty
  strings, and gives `cmd/web/signals.go:755` no `CNAME` target and no `NS` lame flag. The two tickets
  above are the fix; this bullet is what a reader seeing blank DNS rows on an asset page should read
  first.
- **`CONTEXT.md` gains nothing.** `Source`, `Facet` and `dns-record` are already defined there, and
  `Source`'s existing text already carries the zone file's re-supply cadence
  (`internal/scan/zone.go:17` cites it). This ADR adds no term and invalidates no clause.
- **The rule is what makes the one v1 conflict pair worth having.** ADR-0020 counted exactly one
  enumerable pair. If the zone value were normalised through the resolver, that count would be one in
  name and zero in substance.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Re-resolve each declared record and store the answer** | It writes the resolver's answer under `source = zone`, so the one conflict pair ADR-0020 counted would compare the resolver against itself and report nothing, forever. It also dates a live measurement to the operator's supply instant, which is the falsehood `internal/scan/zone.go:1` refuses in the time dimension |
| **Normalise the rdata into the resolver's presentation — RFC 5952 addresses, canonical target names** | It is a decode of provider convention by another name, and ADR-0020 measured that cost: *"a stripper per provider convention, forever"*, which ADR-0004 calls the out-of-band tell. It also makes the two sources agree by construction on exactly the records where a real disagreement would matter |
| **Refuse to store a zone `dns-record` value at all, and keep only the name set** | ADR-0020 bounds what a **rule** may read. It does not say the observation must not exist, and the observation is what gives `dns-record` its second timeline. Dropping it would delete ADR-0007's only v1 conflict pair rather than merely leaving it unused by rules |
| **Store the raw line, owner, class, TTL and all** | The key must be a canonical `(subject_key, discriminator)` pair or the two timelines never meet, per §2 and the ADR-0107 / #256 join. Keeping the class and the TTL inside the value also reintroduces the TTL churn ADR-0011 excluded deliberately for every source |
| **Keep the flat `[]string` shape and give the console a per-source decoder instead** | It moves ADR-0011's decoder from the producer to every consumer, so each new reader of `dns-record` must know which sources exist and how each spells a record. ADR-0011 put the decoder at `(facet, source)` precisely so a reader sees one value space |
| **Canonicalise the owner name only in the console, and store what the file wrote** | The key would then differ between the two sources at rest, so the two timelines would sit on different `subject_key`s and never be compared. `internal/scan/crtsh.go:157` already records this failure once, on the admission hop's join |
| **State it in [`v1-spec.md`](../spec/v1-spec.md) §3.4 or §4.1** | Both are about the `zone` Scan's cadence, supply instant and skip behaviour, and both are settled and correct. What the value carries is a facet question, and this ADR's ground is ADR-0007 and ADR-0011 rather than the Scan's shape |
| **Amend [ADR-0011](./0011-a-facet-is-six-parts.md) with a zone clause** | ADR-0011 rules that a facet is six parts and that the decoder is per `(facet, source)`. That rule is correct and unchanged; the zone decoder is what fails it. Filing this there would put a defect report inside a model ruling and blur which of the two is wrong |
