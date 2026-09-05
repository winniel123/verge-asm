# ADR-0209: a closed union we author refuses an unknown member and writes no row

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1321 ADR gaps: internal/queue (#1199)](https://github.com/winniel123/verge-asm/issues/1321), gap 1
- **PR that deleted the comment:** [#1327](https://github.com/winniel123/verge-asm/pull/1327)
- **Sibling of, and not ruled by:** [ADR-0143](./0143-an-rcode-is-a-closed-union-and-every-code-the-leaf-does-not-discriminate-on-folds-to-other.md). That ADR rules a closed union over a value a **foreign** party authors — a DNS response code — and folds every member the leaf does not discriminate on to `OTHER`. This ADR rules a closed union over members **we** author. The two share a shape and take opposite defaults, and §2 states the discriminator
- **Rests on:** [ADR-0126](./0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md), which chose the closed-union value shape for the `Transcript` corpus. It rules that `wire.Transcript` **is** a closed union. It does not rule what a reader of one must do with a member it does not know
- **Rests on:** [`raw-job-output.md`](../spec/raw-job-output.md) §1.2, which states the same shape and names the three variants. That document makes no new decision by its own statement, so it cannot carry this one

## Context

`internal/queue/transcript.go:24` carried this in declaration position, until #1327 deleted it:

```go
// buildTranscriptParams turns a captured wire.Transcript into the row the worker
// writes inside a job's terminal tx (spec §2.4). The worker stamps jobID (the
// queue_job grain, one per attempt) and capturedAt (w.now()); the producer supplies
// the kind, duration, streams and typed outcome. key seals every captured stream at
// rest (spec §5.3).
//
// The prober variant captures locally (#865); the zone variant captures on its
// completed path (#869); the ct variant captures the crt.sh HTTP exchange on every
// terminal path (#870). An unexpected variant is a wiring bug and errors loudly
// rather than writing a mislabelled row.
```

The last sentence states a rule. `raw-job-output.md` §1.2 and `CONTEXT.md` both rule that the value
**is** a closed union. Neither rules the default branch. That is #1321's gap 1.

**Four type switches over an internal union exist in the whole non-generated tree, and all four sit
in this one file.**

| Switch | Site | Union | Default |
| --- | --- | --- | --- |
| `buildTranscriptParams` | `internal/queue/transcript.go:22` | `wire.Transcript` | `fmt.Errorf("queue: transcript variant %T not captured yet", t)` |
| `encodeZoneOutcome` | `internal/queue/transcript.go:75` | `wire.ZoneOutcome` | `fmt.Errorf("queue: unknown zone outcome %T", o)` |
| `encodeCTOutcome` | `internal/queue/transcript.go:123` | `wire.CTOutcome` | `fmt.Errorf("queue: unknown ct outcome %T", o)` |
| `encodeProberOutcome` | `internal/queue/transcript.go:195` | `wire.ProberOutcome` | `fmt.Errorf("queue: unknown prober outcome %T", o)` |

The four remaining type switches in the tree are over Go AST nodes, over a template parse node
(`internal/commentlint/surface`), and over `crypto.PublicKey`. All four are over a foreign type. So
the whole population of this rule is four sites, one comment stated it, and the comment is gone.

**The unions are sealed, and that is what makes "a wiring bug" a precise claim rather than a figure
of speech.** Each carries an unexported marker method — `isTranscript()`, `isProberOutcome()`,
`isCTOutcome()`, `isZoneOutcome()` (`internal/wire/transcript.go:9`, `:43`, `:64`, `:87`). Only
`internal/wire` can add a member. The default branch is therefore unreachable from any input, from
any operator act and from any remote party. It is reachable only after an edit inside `internal/wire`
that adds a member and does not add the encoder arm.

**The database closes the union a second time.** Migration `23700_transcript.sql:38` declares
`variant TEXT NOT NULL CHECK (variant IN ('prober', 'ct', 'zone'))`. A fallback that tried to write a
partial row could not name a variant tag that passes the check. The fallback would have to reuse one
of the three and label the row as a variant it is not.

**The naive reading fails, and ADR-0143 is where it fails.** `Rcode` is a closed union whose default
branch deliberately does **not** error. Eleven wire codes fold to `OTHER`. If *closed union* alone
decided the default branch, ADR-0143 and this ADR would contradict each other. It does not, and §2
names what does.

**Refusing is not free, and the price is larger than the transcript.** `persistTranscript`
(`worker.go:144`) is called inside the job's terminal transaction on all three terminal paths:
complete (`worker.go:451`), dead-letter (`:483`) and retry (`:510`). The error returns through
`inTx`, so the rollback takes the observation fold, the membership folds, the message production and
the terminal state marking with it. A job that reaches the default branch does not reach `done`.

## Decision

> **A closed union whose members we author refuses an unknown member. The reader returns an error and
> writes no row. It never writes a partial row, never substitutes a fallback member, and never labels
> the row with a variant tag it is not. The closedness is the contract, so the only way to widen the
> union is to add a variant. A closed union over a value a foreign party authors is a different case
> and folds, under ADR-0143.**

### 1. The default branch errors, and nothing is written

All four sites already do this. The error names the type it could not handle, and the row is not
built, so `InsertTranscript` is never reached. The caller is free to fail. It is not free to write.

### 2. The discriminator is who authored the value, not that the union is closed

This is the whole content of the rule, and it is the half `CONTEXT.md` and the spec leave open.

**A member of a union we author is not a fact about the world.** Every member of `wire.Transcript`
exists because this repo declared it in `internal/wire`. An unknown member is a state the program
cannot be in unless a change shipped half-applied. There is nothing to record, because nothing
happened outside the code.

**A member of a union a foreign party authors is a fact about the world.** ADR-0143's eleven
fall-through response codes are real answers from real authorities. Refusing them would abort a walk
on a response the leaf is entitled to ignore, which is ADR-0143's own reason for folding. The peer
answered, and a fold records that answer at the resolution the leaf makes distinctions at.

So the two rules are one rule read at two origins. **Fold what the world says and you do not
discriminate on. Refuse what only we could have said and did not say.**

### 3. A fallback is never allowed on the write path

The `transcript` row is durable, admin-read, and the only record the corpus keeps of that exchange.
A mislabelled row is worse than an absent one, because a reader cannot tell it is wrong. The `?job=`
view decodes the outcome object by the variant tag, so a row labelled `prober` is read with the
prober decoder whatever it holds.

Widening is the only route. It costs a member in `internal/wire`, an arm in the encoder, a migration
widening the `CHECK`, and a decoder arm in `cmd/web/rawoutput.go`. That is four edits, and the
default branch is what makes the second one impossible to forget.

### 4. The read path is bounded out of this rule

`cmd/web/rawoutput.go`'s three decoders — `rawDecodeOutcome:180`, `rawDecodeZoneOutcome:206`,
`rawDecodeCTOutcome:230` — switch on the JSON `kind` **string** and render `—` for a kind they do not
know. **That is correct, and this ADR does not reach it.** There are two reasons.

The reader decodes a durable row a past version wrote. An unknown `kind` there is a
forward-compatibility fact about the corpus, not a wiring bug in the running binary. The read is of a
fixed row rather than an authoring of a new one.

Failing the page would also deny the operator the streams the row does carry. The stdout panel, the
duration and the truncation markers are all still readable when one scalar is not. Refusing on write
protects the corpus. Refusing on read would spend it.

A later session must not "fix" the reader to match the writer. The asymmetry is the ruling.

### 5. The refusal costs the whole terminal transaction, and it is still right

§Context measures the price. The job does not reach `done`, the reaper reclaims it
(`internal/queue/reaper.go`), and every attempt fails the same way until the encoder arm is added.

That is the loudest failure available, and it is the one wanted. The alternative — commit the job and
drop the capture — leaves a corpus that is silently incomplete exactly when a wiring bug is in
flight. It also makes the drop indistinguishable from `raw-job-output.md` §1.4's **legible absence**,
the no-row state that means *this job captured nothing*.

The bound: the default branch is deterministic and reachable only from an unreleased edit, so it
fails on the first job in any environment that runs one. **No test pins it today**, which is the
defect below.

## Consequences

- **No production behaviour changes.** All four sites already error. The change is that the rule now
  has a document, so the deleted declaration comment does not come back.
- **`internal/queue/transcript.go` gains this ADR's citation** on the survivor comment that already
  states the rule. It is recorded in this issue's manifest rather than edited here.
- **A defect: no test pins any of the four default branches.** ADR-0143 §Consequences found the same
  hole on its own default, and its finding was that an untested default is how a wrong one survives
  behind a comment asserting it is right. Four table-driven cases over a locally declared union
  member would close it. It ships as its own ticket.
- **A defect this ADR exposes rather than creates: a wiring bug becomes an unbounded job-failure
  loop, not a lost transcript.** `raw-job-output.md` §2.4 and ADR-0126 put the capture write inside
  the terminal transaction on purpose, so a transcript cannot outlive a rolled-back attempt. Whether
  a capture-encoding failure specifically should be non-fatal is a separate decision about the
  transaction rather than about the union, and it ships as its own ticket.
- **`CONTEXT.md` gains nothing.** It already states that every value space is a closed union. What a
  reader does at the default branch is a code rule, not a domain term, and a glossary entry would
  file a rule where readers look for a vocabulary.
- **A fifth `Transcript` variant is admitted by the four edits §3 names.** A variant that cannot pay
  the migration is a variant that cannot be written at all, which is the intent.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Fold an unknown variant to a generic row, as ADR-0143 folds an unknown rcode to `OTHER`** | ADR-0143's fold is licensed by the value being the world's answer, which the leaf may record coarsely. A `Transcript` variant is a value only this repo can mint, so a generic row records nothing that happened. It records that we forgot an arm, in a corpus an operator reads as evidence |
| **Write the row with `variant` set to `'unknown'` or NULL** | Migration `23700_transcript.sql:38`'s `CHECK (variant IN ('prober', 'ct', 'zone'))` refuses both, and `variant` is `NOT NULL`. Widening the check to admit a sentinel would put a value in the column that the `?job=` view has no decoder for, so the row would be unreadable as well as unlabelled |
| **Log the unknown variant, skip the capture, commit everything else** | Makes a shipped wiring bug invisible in the one corpus kept for debugging shipped bugs. It is also indistinguishable from `raw-job-output.md` §1.4's legible absence, so an operator reading *no transcript for this job* cannot tell whether the job captured nothing or the encoder dropped it |
| **Panic on the default branch** | Ends the worker process for a condition one job's variant reaches. §5's rollback already fails the job loudly and bounds the damage to that job, and ADR-0141 rules that a worker loop survives a failed unit of work |
| **Reuse the nearest variant — write a `ZoneTranscript` as `prober`** | The mislabelled row §3 refuses, in its most damaging form. It passes the `CHECK`, it renders in the admin view, and it is read with the wrong decoder |
| **Make the switch exhaustive at compile time** | Go has no exhaustiveness check for a type switch. The unexported marker method closes the **set** of members, not the **switch** over them. An `exhaustive` linter is not in CI, and adding one would be a new required check for four call sites |
| **State the rule in [`raw-job-output.md`](../spec/raw-job-output.md) §1.2** | That document states in its own preamble that it makes **no new decision** and folds locked rulings into a buildable spec. A rule nobody has ruled cannot be folded into it |
| **Merge this with the ADR that rules where a variant scalar lands** | Two decisions about one union. One rules the default branch of a reader. The other rules which column a variant's own field earns. Neither implies the other, and merging them would put a schema rule inside a rule about error handling |
