# ADR-0191: an unrecognised CT entry type costs one entry on the framed RFC 6962 path and fails the whole tile on the unframed static-ct-api path

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1308 ADR gaps: internal/scan (CT and zone Scans)](https://github.com/winniel123/verge-asm/issues/1308), gap 4
- **PR that deleted the comments:** [#1307](https://github.com/winniel123/verge-asm/pull/1307)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.3, which rules that **two** client implementations are mandatory — RFC 6962 and static-ct-api. It creates the two paths this ADR rules apart. It does not say how either handles a value it does not know
- **Rests on:** [ADR-0027](./0027-a-source-may-admit-without-observing.md), which rules CT `corroborative` and admitting-without-observing. A source that never asserts an absence is what makes a skipped entry a legible loss rather than a false negative
- **Sibling of, and not ruled by:** [ADR-0143](./0143-an-rcode-is-a-closed-union-and-every-code-the-leaf-does-not-discriminate-on-folds-to-other.md), which rules that an unknown DNS response code folds to a named `OTHER`. It is the same question — *what does a leaf do with a wire value it does not discriminate on* — and it reaches the opposite answer for a reason §3 states. Neither contains the other
- **Bounded by:** [ADR-0213](./0213-a-ct-tail-skips-one-entry-it-cannot-decode-and-the-poll-continues-and-advances-past-it.md). That ADR rules the tail's poll-versus-skip boundary and states that this ADR's *“the tiled path refuses”* binds the framing layer alone. It must not be read as *the tiled path never skips a leaf*

## Context

`internal/scan/cttail.go:334` and `:522` carried these, until #1307 shortened them:

```go
// An entry
// type the tail does not recognise yields no names and no error — the tail tolerates a
// future entry type rather than failing the whole poll on it.
```

```go
// An unknown entry type has an unknown signed_entry length, so the rest of the
// tile cannot be framed. Fail the tile rather than guess an offset.
```

Both survive in short form, at `internal/scan/cttail.go:269`:

```go
default:
	return nil, nil // a future entry type is tolerated, never a failed poll
```

and at `internal/scan/cttail.go:378`:

```go
default:
	// An unknown entry type has an unknown length, so the tile cannot be framed past it.
	return nil, nil, fmt.Errorf("unsupported entry type %d", entryType)
```

### The two entry types, and the two transports

`internal/scan/cttail.go:230` declares the closed set the tail knows:

```go
ctEntryX509    = 0
ctEntryPrecert = 1
```

Both transports carry the same two values in the same two-byte field. **They frame the field
differently, and that is the whole of this ruling.**

**RFC 6962 `get-entries` returns JSON.** `ParseLogEntries` (`internal/scan/cttail.go:203`) decodes
each element's `leaf_input` and `extra_data` from base64 into its own `[]byte`. **The transport
already framed every entry, so an entry's length never depends on its content.** `LeafSANs` reads
one entry's bytes and returns.

**A static-ct-api data tile is one concatenated byte stream.** `ParseDataTile`
(`internal/scan/cttail.go:336`) loops:

```go
der, rest, err := parseTileLeaf(b)
...
b = rest
```

`rest` is the only thing that finds the next leaf. `parseTileLeaf` computes it by walking the
leaf's fields in order: an 8-byte timestamp, a 2-byte `entry_type`, a **type-dependent body**, a
`uint16`-prefixed extensions block, a `pre_certificate` block on the precert arm, and a
`uint16`-prefixed certificate chain. Every one of those has a length that can be read from the wire,
**except the body**:

| `entry_type` | Body the parser must step over |
| --- | --- |
| `0` x509 | one `opaque<1..2^24-1>` certificate |
| `1` precert | a 32-byte `issuer_key_hash`, then one `opaque<1..2^24-1>` TBSCertificate |
| anything else | **unknown, because only the type says what is there** |

So on the tiled path an unrecognised type does not cost the entry. It costs **the offset of every
entry after it**, and there is no way to recover one without guessing.

### The two callers, measured

| | RFC 6962 (`internal/queue/cttail.go:91`) | static-ct-api (`internal/queue/cttail.go:153`) |
| --- | --- | --- |
| Unknown entry type | `LeafSANs` returns `(nil, nil)`. No names, no error, **no log line** | `ParseDataTile` returns an error. `retryOrDeadLetterCT` retries, then dead-letters a Batch |
| Unparseable certificate DER | `LeafSANs` returns an error. `internal/queue/cttail.go:94` logs and `continue`s | `CertSANs` returns an error. `internal/queue/cttail.go:168` logs and `continue`s |
| Cursor after the event | `reached += int64(len(entries))`. **The cursor advances past it** | `reached` is never assigned. **The cursor does not move** |
| Cost of the event | One entry's names | Every entry from that leaf to the end of the tile, plus the poll |

**The tiled path is not globally stricter.** Its second row is identical to the first path's: a leaf
whose certificate will not parse is logged and skipped, exactly as on `get-entries`. The tiled path
refuses **one** thing, and that thing is framing.

### The retry is not the same act on the two paths

`retryOrDeadLetterCT` (`internal/queue/crtsh.go:253`) retries until `MaxAttempts`, then dead-letters.
Every other caller of it on the tail is a **fetch** failure — a non-200 on `get-sth`,
`get-entries`, `checkpoint` or `tile/data`, or a malformed body. Those are transient and a retry is
the right instrument.

**An unknown entry type inside a tile is not transient.** The retry re-fetches the same tile index at
the same width, gets the same bytes, and fails at the same leaf. The retries are spent, the Batch
dead-letters, and the next poll starts from the same unadvanced cursor and does it again.

## Decision

> **A CT log entry whose `entry_type` the tail does not recognise yields no names and no error on the
> RFC 6962 path, because the transport framed that entry independently and the loss is exactly one
> entry. The same value fails the whole fetch inside a static-ct-api data tile, because a tile leaf
> carries no length prefix and an unknown type's body length cannot be read, so no later leaf's offset
> can be found. The two directions are deliberate and opposite. A caller must not treat the tiled
> failure as a retryable fetch failure, because it is deterministic and a retry cannot succeed.**

### 1. The RFC 6962 path tolerates, and the loss is bounded at one entry

`LeafSANs`'s `default` arm returns `(nil, nil)`. Not an error, and not a log line.

The bound is what makes this safe rather than lax. The entry the tail cannot read contributes no
names. Every other entry in the same response is unaffected, because `ParseLogEntries` already
decoded each one separately. The cursor advances by the count the log returned, so the poll makes
forward progress and the tail keeps following the log.

**A skipped entry is a missed admission and never a false absence.** ADR-0027 rules CT
`corroborative` and admitting-without-observing, so nothing the tail fails to read becomes evidence
that a name does not exist. That is why the loss can be absorbed silently: it narrows what is
admitted and asserts nothing.

### 2. The tiled path refuses, and refuses the tile rather than the leaf

`parseTileLeaf`'s `default` arm returns an error, and `ParseDataTile` propagates it, so the whole
tile fetch fails.

**Skipping the leaf is not available.** Skipping requires knowing where the leaf ends, and the leaf's
end is precisely what the unknown type withholds. §Context's table is the measurement: the body is
the one field whose length is not on the wire.

**Guessing is worse than failing.** A guessed offset lands mid-structure and the parser reads
whatever follows as a timestamp and an entry type. It does not reliably error. It can produce leaves
that parse, yield DER that parses, and admit `Name`s that were never in the log. **A silent
fabrication of admissions is a far worse failure than a stalled log**, and it is the failure a
tolerant tiled parser buys.

**Failing the tile rather than truncating it is the same choice again.** Returning the leaves parsed
so far and stopping would advance `reached` past them and leave the rest of the tile unread forever,
with nothing recording that it was skipped. The tail would silently under-read a log and report a
completed poll.

### 3. The two directions are one rule, not an inconsistency

The rule is: **tolerate an unknown value where the transport bounds the damage, and refuse it where
it does not.** RFC 6962's JSON framing bounds the damage at one entry. A tile's concatenation does
not bound it at all.

This is where [ADR-0143](./0143-an-rcode-is-a-closed-union-and-every-code-the-leaf-does-not-discriminate-on-folds-to-other.md)
is the instructive sibling rather than a contradiction. It rules an unknown DNS response code into a
named `OTHER`, and it can, because an rcode is a fixed-width field in a message whose framing does
not depend on its value. A CT tile leaf's `entry_type` **is** its framing. The same tolerance is not
available, and offering it would be the guess §2 refuses.

### 4. A caller must not retry the two the same way

**On the RFC 6962 path there is no unknown-type case to retry.** It never reaches the caller as an
error. Every error the caller does see there is a fetch or a decode failure, and retrying is correct.

**On the tiled path an unknown entry type is deterministic.** The same tile, the same bytes, the same
leaf, every time. Retrying spends `MaxAttempts` to reach the same answer, and the cursor never
advances, so **that log is stalled until a release teaches `parseTileLeaf` the new type**. The retry
is not wrong so much as inert, and a reader must not mistake the dead-letter for a transient log
outage.

**This ADR does not add a discrimination between the two failure kinds**, and §Consequences names why
that is a defect rather than a decision.

### 5. What this rule does not reach

- **The two type values themselves.** `ctEntryX509` and `ctEntryPrecert` are RFC 6962 §3.4 wire
  constants, and `internal/scan/cttail.go:228` already says they are not ours to choose.
- **A leaf whose certificate will not parse.** Both paths log and skip it, and that is one behaviour,
  not two. It is not this rule.
- **Whether the tail should learn a new entry type.** That is a release decision, taken when one
  exists. This ADR rules what happens in the meantime.
- **The `get-entries` short-read rule.** `internal/scan/cttail.go:201` already states that the cursor
  advances by the count returned, and that is §4.4's rule rather than this one.

## Consequences

- **This ADR changes no Go code.** Both arms are correct as they stand.
- **`internal/scan/cttail.go:269` and `:378` each gain this ADR's citation**, on the surviving line
  that states their half of the rule. Recorded in this issue's manifest.
- **A stalled tiled log is invisible as a stall, and that is a defect this ruling exposes.** A
  dead-lettered `ct-tail` Batch carrying `data tile leaf N: unsupported entry type 7` looks exactly
  like a dead-lettered Batch carrying a 502, and the operator has no way to tell a transient log
  outage from a permanent parser gap. **It ships as its own ticket:** discriminate the framing
  failure from a fetch failure and dead-letter it without spending retries, so the message says the
  tail cannot read this log rather than that the fetch failed.
- **The tail keeps following every other log.** The stall is per log, because the tail fans out
  per log and each job carries its own cursor row. One log with an unknown entry type does not stop
  the other 38.
- **Nothing enforces the asymmetry.** A future author who adds tolerance to `parseTileLeaf`'s
  `default` arm gets a green build and a parser that fabricates admissions. This document is what a
  reviewer holds against that change.
- **`CONTEXT.md` gains nothing.** An entry type is a CT wire term and holds no timeline.
- **No upstream entry type beyond `0` and `1` exists today.** The rule is written before it is
  needed, which is the point: the first appearance of a third type must not be the moment anyone
  decides what to do.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Tolerate an unknown type on the tiled path too, by skipping the leaf** | Skipping needs the leaf's length, which is exactly what the unknown type withholds. There is nothing to skip *to* |
| **Guess the offset — assume the unknown body is an `opaque<1..2^24-1>` like both known types** | A guess that is wrong lands mid-structure, and the parser does not reliably error there. It reads arbitrary bytes as a timestamp and a type, and can yield DER that parses and `Name`s that were never logged. Silent fabricated admissions are a worse outcome than a stalled log |
| **Return the leaves parsed before the unknown one and stop** | `reached` would advance past them, so the rest of the tile is skipped permanently with nothing recording it. The tail would under-read a log and report the poll completed |
| **Fail the whole poll on the RFC 6962 path too, for symmetry** | Symmetry is not a property worth buying here. It would stall a log over a loss the transport already bounded at one entry, and it would do it on the 85% of the corpus that is RFC 6962 (ADR-0128) |
| **Fold an unknown type to a named `unknown` value, the way [ADR-0143](./0143-an-rcode-is-a-closed-union-and-every-code-the-leaf-does-not-discriminate-on-folds-to-other.md) folds an rcode to `OTHER`** | An rcode is a fixed-width field in a message whose framing is independent of it. A tile leaf's `entry_type` *is* the framing. Naming the unknown value does not tell the parser where the leaf ends, so the fold buys nothing on the path that needs it |
| **Log the skipped entry on the RFC 6962 path** | It would log once per unknown entry across a firehose that carries about 9,500 entries per minute per shard (§4.4). The first appearance of a third entry type would fill the log and tell the operator nothing they could act on. The tiled path's dead-letter is where the event becomes visible, and the Consequences ticket makes that message accurate |
| **Ship it as one rule on [ADR-0143](./0143-an-rcode-is-a-closed-union-and-every-code-the-leaf-does-not-discriminate-on-folds-to-other.md)** | ADR-0143 rules a DNS leaf's value space, and its answer is a fold this code cannot perform. Filing the opposite answer inside it would make one document say *fold it* and *refuse it* about two unrelated wire formats |
| **State it in [`ct-source-replacement.md`](../spec/ct-source-replacement.md) §4.3 or §4.4** | §4.3 rules which logs are followed and which clients are mandatory. §4.4 rules cadence and the consistency proof. Neither is about what a parser does with a value it does not know, and both are settled |
