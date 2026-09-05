# ADR-0163: An absent certificate-material row is a fan-out of zero and is reached, and only an absent measurement row is pending

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1389 ADR gaps: db/queries (3/4)](https://github.com/winniel123/verge-asm/issues/1389), gap 2
- **PR that deleted the comment:** [#1388](https://github.com/winniel123/verge-asm/pull/1388)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounds:** [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md)'s #954 amendment, at the absence rule's own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** ADR-0129's #954 amendment, which rules hold-then-open on the custody-extension population, and its #955 amendment, which fixes the threshold at an absolute count of 100

## Context

[`db/queries/certificate_material.sql`](../../db/queries/certificate_material.sql) carried this above
`ListCertificateMaterialDER`, until #1388 deleted it:

```sql
-- A fingerprint with NO captured material returns NO ROW. That absence is a value, not
-- an error: a `presented` handshake whose material never landed yields no names, so its
-- edge reduces to a fan-out of zero and is reached (ADR-0129 §2). The caller must not
-- read a missing row as *measurement pending* — the missing MEASUREMENT row is what
-- means that, and it is a different absence.
```

**The citation does not reach the rule.** ADR-0129 §2 is *"It escapes ADR-0013 §3 because the
failure direction is reversed"*. It rules that shared-edge suppression withholds a probe rather than
opening a gate, so its false positive surfaces loudly as a `Gap`. It states nothing about a zero
fan-out, nothing about an absent capture, and nothing about pending. This is `comment-policy.md`
§8.3's live, on-topic source that rules something else. §4.7 test 4 already records that a `§n`
citation to ADR-0129 is very likely wrong, because that ADR's live rules sit in its issue-named
amendments.

**The rules that are written cover the two neighbours and not this case.**

ADR-0129's #954 amendment states the absence rule for the custody-extension population:

> **enabled but not yet measured** *holds* the reach (census: *measurement pending*), bounded by the
> daily cadence

Its #956 amendment states the other absence rule, for the address-scope population: an unmeasured
declared address is probed normally and carries no row, which is open-then-label. Neither amendment
says what *measured* means when the measurement row exists and the certificate bytes behind it do
not. Read literally, a candidate whose SAN set was never captured has not had its fan-out measured,
so the #954 clause would hold it as pending. The code reaches it. That contradiction is what this
ADR settles.

Nothing under `docs/adr/`, `docs/spec/`, `docs/guides/`, `docs/research/` or `CONTEXT.md` states the
absent-capture rule. [`docs/spec/ct-source-replacement.md`](../spec/ct-source-replacement.md) §5.3
specifies the `certificate_material` side store and rules CT verification's use of it. It never
mentions the fan-out reduction.

**The behaviour is real and the fold makes it in three lines.** In
[`internal/queue/edgefanout.go`](../../internal/queue/edgefanout.go)'s `toEdgeFanout`,
`material[fingerprint]` on a missing key yields `nil`. `edgeFanoutSANs(nil)` returns `nil` at its
`len(der) == 0` guard. `custody.SharedEdge(nil)` counts zero registrable domains against a threshold
of 100 and derives `false`.

**The two absences then part in `internal/custody`, on one map read.**
[`internal/custody/veto.go`](../../internal/custody/veto.go)'s `admits` and
[`internal/custody/census.go`](../../internal/custody/census.go) both read
`shared, measured := f.Shared[addr]`. A key present with `false` is measured and not shared, so the
address is admitted and the census writes no row. A key absent is not measured, so the address is
held and the census writes an `ExtensionPending` row.

The two nearest comments in `edgefanout.go` state the two neighbouring rules and not this one. One
states the missing-measurement case. The other states the negative-outcome case, and cites
`ADR-0129 §2` for it — the same wrong token the deleted block carried.

## Decision

> **A measured address's fan-out verdict is decided by its measurement row, and the absence of a
> `certificate_material` row is a value inside that verdict, never a second kind of absence. A
> `presented` outcome whose certificate material never landed yields no names, reduces to a fan-out
> of zero, derives not-shared, and is reached. Only an absent measurement row means *measurement
> pending*.**

Four limbs.

### 1. Three inputs, and only one of them is pending

| What the fold sees | Key in `Shared` | Derivation | Census |
| --- | --- | --- | --- |
| No measurement row for the address | Absent | Held, not reached | `ExtensionPending` |
| A row with a negative outcome | Present, `false` | Not-shared, reached | No row |
| A row with `presented` and no captured material | Present, `false` | Not-shared, reached | No row |
| A row with `presented` and a captured leaf | Present, `SharedEdge(SANs)` | The measured verdict | A row when shared |

The table is the whole rule. A reader who holds it does not need to know how the map is built.

### 2. `ListCertificateMaterialDER` returning fewer rows than it was asked for is not an error

The query reads over a fingerprint set. It returns one row per distinct certificate that was
captured, and no row for a fingerprint that was not. `readEdgeFanoutMaterial` builds its map from
whatever rows come back and never compares the count to the request.

A caller must not treat a short result as a failed read. A failed read is the error return, which
`ReadEdgeFanout` propagates rather than folding.

### 3. A fan-out of zero is a measurement, not a missing measurement

The reduction counts distinct registrable domains in the leaf's `dNSName` SANs. Zero is a legal count
and it sits below the threshold of 100, so it derives not-shared exactly as a count of one does.

The direction is the one ADR-0129 §2 already accepts for the whole mechanism. Reaching an edge is
the loud direction. Holding one is the quiet direction, and the quiet direction is reserved for the
case where nothing measured the address at all.

### 4. ADR-0129's #954 absence rule is bounded, at its own site

*Not yet measured* means **no measurement row for the address**. It does not mean *no captured
certificate material*. The bounding sentence goes in ADR-0129, at the #954 amendment's absence rule,
not only here. ADR-0058 requires it. A reader who finds that clause and an address the code reaches
must not have to find this ADR first to know which reading is wrong.

Both of ADR-0129's populations keep their existing rules. This limb narrows what counts as
unmeasured. It does not move an address between populations.

## Consequences

- **[ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md)'s #954
  amendment gains one bounding sentence** at its absence rule, naming this ADR.
- **[`internal/queue/edgefanout.go`](../../internal/queue/edgefanout.go) gains this ADR's citation
  in `toEdgeFanout`**, on the line that derives the verdict from the material map.
- **The wrong `ADR-0129 §2` citation on the negative-outcome line in `toEdgeFanout` is repaired to
  this ADR.** That line states limb 1's second row. It sat on the same fold as the deleted block and
  carried the same wrong token, and limb 1 is now the section that rules it.
- **[`db/queries/certificate_material.sql`](../../db/queries/certificate_material.sql) gains
  nothing.** `sqlc` copies a comment above a query into `internal/db` as a doc comment on the
  generated method and on `Querier`. `comment-policy.md` §2.2 keeps Go declaration position empty,
  and #1388 removed the generated copies for that reason.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Its `edge-fanout` paragraph says a candidate is
  held *"until the probe clears or declines it"*. The probe has cleared a `presented` address whose
  fan-out is zero, so that sentence stays true under this ADR and specifies nothing this ADR
  withdraws. ADR-0129's clause is the one that reads wrong, because it says *not yet measured* and
  never says what measured means.
- **No production behaviour changes.** `toEdgeFanout`, `admits` and the census already have the shape
  this ADR states.
- **The three `internal/custody` citations that named ADR-0129's §5 are repaired, and none of the
  three took this ADR's citation.** [#1368](https://github.com/winniel123/verge-asm/issues/1368)
  owned them, and PR [#1440](https://github.com/winniel123/verge-asm/pull/1440) repointed
  `census.go` at ADR-0129's #944 amendment and both `scopecensus.go` lines at its #956 amendment.
  None of the three is about an absence.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Read an absent `certificate_material` row as *measurement pending*** | Holds an address the probe already reached, on the strength of a side store that CT verification owns. A capture that never landed is a gap in `certificate_material`, and turning it into a custody hold puts a storage fault in the reach decision. It also makes the census carry a pending row that no cadence clears, because the measurement row is already there and the `Scan` will not re-measure it |
| **Return an error from `ListCertificateMaterialDER` when a fingerprint has no row** | Turns a legal absence into a failed read, and `ReadEdgeFanout` propagates a failed read rather than folding it. One uncaptured leaf would then withhold the whole fan-out result and widen the reach to everything, which is the direction the read path already refuses |
| **Keep `ADR-0129 §2` as the citation and file no ADR** | §2 rules the failure direction of the suppression against ADR-0013 §3. It states nothing about a count of zero or about an absent capture. §4.7 test 1 and §8.3 both refuse a source that is live and on-topic and rules something else |
| **State the rule as an amendment on [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md)** | ADR-0129 already carries four issue-named amendments, and §4.7 test 4 records that its layering is what makes its citations unreliable. This rule is about the fold in `internal/queue` and the map read in `internal/custody`, which ADR-0129 never ruled on. ADR-0129 takes the bounding sentence its own clause needs and no more |
| **Write the rule in [`docs/spec/ct-source-replacement.md`](../spec/ct-source-replacement.md) §5.3** | That section specifies the side store for CT verification, whose reader is the log check. Filing a custody-reach rule there would put it in front of the one reader who does not need it, and behind the two packages that do |
| **Comment each of the three absence cases at its own line in `toEdgeFanout`** | Three copies of one rule inside one function, which is the shape the sweeps were already flagging. Limb 1's table holds all three, and the fold carries one citation to it |
| **Also mark [`CONTEXT.md`](../../CONTEXT.md)'s `edge-fanout` paragraph under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)** | Its sentence conditions the hold on *"until the probe clears or declines it"*, and the probe has cleared this address. Read alone and in the present tense it specifies nothing that this ADR withdraws, so it passes ADR-0058's reader test |
