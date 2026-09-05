# ADR-0207: An enumeration that assembles a probing target set drops a row it cannot fully name, and never fabricates a target

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1320 ADR gaps: internal/queue (#1200, sweep 6/7)](https://github.com/winniel123/verge-asm/issues/1320), gap 2
- **PR that deleted the comment:** [#1324](https://github.com/winniel123/verge-asm/pull/1324)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md), which rules that an address that cannot be keyed is not a subject, is absent from the `Batch`'s recorded scope, and writes no value and no `Gap`. That is this rule's ground one layer down, at the point a key is formed. This ADR rules the enumeration that reads keys back out
- **Rests on:** [ADR-0019](./0019-the-probing-gate-is-total-over-an-address.md), which makes the probing gate a total function of an address's `Custody`. A gate refusal and a cannot-name drop are different acts and §4 keeps them apart
- **Sibling of, and not ruled by:** [ADR-0154](./0154-a-narrowing-fold-closes-only-what-it-can-attribute-to-a-mover-and-drops-every-other-candidate.md). That ADR rules a **narrowing** act: a fold drops a span it cannot attribute to a declared mover, and the drop leaves a timeline open. This ADR rules a **widening** act: an enumeration drops a row it cannot name, and the drop withholds a probe. Same direction, opposite acts, and neither contains the other
- **Bounded by:** [ADR-0208](./0208-the-queue-reads-a-services-subject-and-never-re-parses-its-rendering-so-a-rendered-key-is-an-identity-token-alone.md). One of the drop sites this ADR names fires on a subject key that does not parse. ADR-0208 rules that the queue must not parse a rendered key at all, which removes that drop's input rather than its rule

## Context

`internal/queue/tlsacceptance.go:56` carried this in Go declaration position, until #1324 deleted it:

```go
// reachedServices reads the open `Service` population from the current reachability
// spans and parses each `address:port/tcp` subject key back to an address and port.
// A span with no vantage row, or a key that does not parse, is skipped — the
// enumeration never fabricates a target it cannot name.
```

The same commit deleted the inline half of it, at the first of the two skips:

```go
continue // a Service with no vantage row — the enumeration is per-vantage
```

#1324 compressed the block to one surviving line at `internal/queue/tlsacceptance.go:42`:

```go
// A target the enumeration cannot name is skipped, never fabricated from a partial row.
```

The survivor carries no citation. [`comment-policy.md`](../spec/comment-policy.md) §4.7 opens a
follow-up wherever a surviving **uncited** reason asserts a decision, and this one does. The same
decision is taken independently at eight further sites, and no ADR, SPEC, guide or `CONTEXT.md`
entry states it as a rule. That is #1320's gap 2.

### Nine drop sites, one direction, and no shared statement

Every read that assembles a probing target set for a fan-out takes the same direction. None of them
cites another, and none of them cites a document.

| Read | Site | Row it drops | What it cannot name |
| --- | --- | --- | --- |
| `reachedServices` | `internal/queue/tlsacceptance.go:50` | a reachability span whose `vantage_id` is NULL | the probe site — the enumeration is per-vantage |
| `reachedServices` | `internal/queue/tlsacceptance.go:53` | a `service` subject key that does not parse | the address and the port |
| `hotEstate` | `internal/queue/hot.go:90` | a name-cited address whose text does not parse | the `Address` itself |
| `coldScope` | `internal/queue/cold.go:47` and `:51` | a `Seed` row whose limb column is NULL | the CIDR, or the domain |
| `nameSeedDomains` | `internal/queue/helpers.go:20` | a name `Seed` row whose domain is NULL | the domain to resolve |
| `mergeResolutionNames` | `internal/queue/helpers.go:42` | an admitted name that canonicalises to empty | the name to resolve |
| `ctSeeds` | `internal/queue/crtsh.go:380` | a name `Seed` row whose domain is NULL | the domain to query CT for |
| `BuildTLSAcceptanceJobs` | `internal/scan/tlsacceptance.go:56` and `:60` | a reached `Service` whose vantage is not in the list, or whose address does not parse | the vantage, or the address |
| `BuildHTTPIdentityJobs` | `internal/scan/httpidentity.go:52` and `:56` | the same two | the same two |

`zoneFiles` (`internal/queue/zone.go:80`) is the one target-set read with no drop, and it is not a
counter-example. Its query filters `WHERE s.kind = 'name'` and `seed_shape` in
`db/migrations/00003_seeds.sql` makes `name_domain` NOT NULL for that kind, so the row it would drop
cannot exist.

### Most of the guards are defence behind a schema that already holds

The measurement worth carrying is that the drops are **not** all the same strength, and the rule has
to be stated so that it binds the weak ones anyway.

- `coldScope`, `nameSeedDomains` and `ctSeeds` guard against a NULL that `seed_shape`'s
  `CHECK ((kind = 'name' AND name_domain IS NOT NULL …) OR (kind = 'address' AND address_cidr IS NOT
  NULL …))` already forbids. The guard can never fire today.
- `hotEstate`'s parse guard reads `NameCitedAddresses`, whose rows were written from addresses the
  model already keyed. It should never fire either.
- `reachedServices`' two guards are the live ones. `span.vantage_id` is nullable by design — the
  shipped resolver position carries no vantage row — and `span.subject_key` is free `TEXT` with no
  constraint on its shape. Both drops can fire on data the schema permits.

So eight of the nine sites are a discipline rather than a working filter, which is exactly why the
rule needed writing down: a reader who measured only the schema would conclude the guards are
redundant and delete them, and the ninth site is the one that would then be alone.

### The drop is invisible, and the fabrication would not be

A dropped row leaves no log line, no count and no row anywhere. `reachedServices` returns a shorter
slice and the fan-out enqueues fewer jobs. Nothing distinguishes *the population is smaller* from
*the population was smaller*.

A fabricated target has the opposite property, and it is the reason the direction is not a matter of
taste. A target that reaches `BuildTLSAcceptanceJobs` or `BuildHTTPIdentityJobs` is enqueued, sent to
a prober, and dialled. A guessed address is a packet to a machine we never had a citation for, and a
guessed port is a connect to a service nobody declared. **The fabrication does not stay inside the
process. It reaches the world**, and it does so under the operator's `Custody` declaration, which is
what makes it the one direction the code may not take.

## Decision

> **A read that assembles a probing target set DROPS a row it cannot fully name, and never
> reconstructs, defaults, or guesses the missing part. Fully named means every component of the
> target is present and keyed: the subject, and the vantage the enumeration probes it from. The rule
> binds every such read, whether the missing part is structurally possible today or only guarded
> against.**

### 1. Fully named is the whole target, not most of it

A target is fully named when the enumeration holds every component the job needs to identify what it
probes and from where. For the reached-`Service` reads that is the address, the port, and the
vantage. For the name reads it is the domain. A row missing any one of them is dropped whole. There
is no partial target: an enumeration never emits a `Service` with an address and a default port, and
never emits a target with no vantage to be probed from.

The ground is ADR-0051's, one layer up from where it was stated. That ADR ruled that an address which
cannot be keyed **is not a subject** — absent from the `Batch`'s recorded scope, writing no value and
no `Gap`. A row this enumeration cannot name is the same object arriving at the reading end: not a
subject, so not a target, so not a probe.

### 2. Reconstruction is refused even where it looks safe

The tempting reconstructions are all local and all refused.

- A `Service` span with a NULL `vantage_id` is not attributed to the first vantage, to every vantage,
  or to the vantage that most recently probed the address.
- A subject key without its `/tcp` suffix is not assumed to be TCP.
- A key whose `address:port` does not parse is not repaired by stripping brackets, retrying a
  different split, or taking the address and defaulting the port.
- A name that canonicalises to empty is not passed through in its raw form.

Each of these would produce a target the data did not contain, and §Context's last paragraph is why
that matters: the reconstruction is dialled. A wrongly attributed vantage also puts a real
measurement on the wrong timeline, because the reachability timeline is keyed per vantage, which
manufactures drift in the sense [#6](https://github.com/winniel123/verge-asm/issues/6) names.

### 3. The drop is silent at the enumeration, and this is a decision rather than an oversight

The observation fold takes the same direction and **names** each drop.
`toEdgeFanoutRows` (`internal/queue/edgefanout.go:71`) builds a `dropped []string` with a reason per
line, and `foldEdgeFanoutObservations` logs each one against the job id. The enumerations do not.

The difference is in what the dropped row is evidence of.

- A fold drops a line **a prober sent us**. The line disagreeing with the model is evidence about an
  instrument outside this process, which the operator may need to act on. That is worth a log line,
  and ADR-0001's *a prober must not deny the queue* is why it is a log line rather than a failure.
- An enumeration drops **our own row**, from a table whose schema already constrains it. A firing
  guard there is a defect in this repo, not a fact about the world, and it is found by the count of
  probed targets moving rather than by a per-row line.

So the rule requires the drop and does not require the reason. A site that already names its drops
keeps doing so.

**The residual hazard is named rather than smoothed.** A silent drop that fires on one row is
invisible and cheap. A silent drop that fires on **every** row empties a whole `Scan`'s population
with no error anywhere, and `reachedServices`' key-parse guard is the one site where that is
reachable — a change to the rendered key form on either side would do it. That is the failure
[ADR-0208](./0208-the-queue-reads-a-services-subject-and-never-re-parses-its-rendering-so-a-rendered-key-is-an-identity-token-alone.md)
closes, by removing the parse rather than by adding a counter to it.

### 4. A gate refusal is a different act, and this rule does not reach it

`BuildTLSAcceptanceJobs` and `BuildHTTPIdentityJobs` also `continue` past a service whose address
fails `estate.MayProbe` (`internal/scan/tlsacceptance.go:64`, `internal/scan/httpidentity.go:59`).
That looks like a drop and is not one.

| | A cannot-name drop | A gate refusal |
| --- | --- | --- |
| What is known | The target cannot be identified | The target is fully identified |
| Ground | ADR-0051 — it is not a subject | ADR-0019 and ADR-0079 — `Custody` refuses it |
| If the row were kept | A fabricated target is probed | A real target is probed without authority |
| Is it recorded? | No. It was never a target | It is a `Custody` outcome the census reads |

Both end in `continue`, and conflating them would let a reader argue that a cannot-name drop is a
`Custody` decision and belongs on a census surface. It is not. `candidateAddrs`' exclusion skip
(`internal/queue/hot.go:144`) is a third thing again — a cost optimisation ahead of a gate that
would refuse anyway — and it is out of this rule too.

### 5. What this rule does not reach

- **An observation fold.** `toEdgeFanoutRows` drops a measured line after the probe ran. Its
  population is what a prober reported, not what we are about to dial, and ADR-0001's
  *log-and-continue* already rules it.
- **A narrowing act.** ADR-0154 rules the three folds that close spans, where the drop leaves a
  timeline open rather than withholding a probe.
- **A malformed entry from an external source.** A CT poll skipping an entry it cannot read is the
  same direction over a third population, and it is [#1323](https://github.com/winniel123/verge-asm/issues/1323)'s
  gap 1. It has its own ADR, because its population is a third party's output and its failure mode is
  a source changing shape rather than our own row being unreadable.
- **Whether the population should have been bigger.** The rule states what an enumeration does with
  an unnameable row. It says nothing about whether the row should have existed, which is a question
  for the write path that produced it.

## Consequences

- **This ADR changes no Go code.** All nine sites already drop. The rule they take was stated only in
  a deleted comment and in one uncited survivor.
- **The survivor at `internal/queue/tlsacceptance.go:42` gains a citation.** It is the one site that
  states the rule in the tree, so it is the one that should point at it.
- **Eight guards that can never fire today are now protected from deletion.** Before this, a reader
  who checked `seed_shape` would find `coldScope`'s, `nameSeedDomains`' and `ctSeeds`' NULL checks
  provably dead and could remove them as noise. They are the rule's expression at those sites, and
  the schema constraint is a second line rather than the only one.
- **A tenth enumeration inherits the direction.** A future fan-out's population read drops what it
  cannot name, and a reviewer has a document rather than a preference.
- **The silent mass drop stays open, and it is ADR-0208's to close.** Nothing here adds a counter, a
  log line or a metric to the enumeration drops. §3 states why, and it names the one site where the
  silence is dangerous.
- **`CONTEXT.md` gains nothing.** `Subject`, `Service` and `Custody` already carry every term this
  rule uses, and none of them changes meaning.
- **A dedup pass has three near neighbours to look at, not one.** This rule, ADR-0154, and #1323's
  CT-entry rule are the same drop-rather-than-guess direction over three populations.
  [`comment-policy.md`](../spec/comment-policy.md) §8.10 owns that pass. The three are kept separate
  here because their grounds differ: ours is a probe that would reach the world, ADR-0154's is a
  closure the operator could not trace, and #1323's is a poll that must not fail on a third party's
  malformed row.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Reconstruct the missing part — default the transport to `tcp`, attribute a vantage-less span to the first vantage** | The reconstruction is enqueued, sent to a prober and dialled. A guessed address is a packet to a machine we hold no citation for, under the operator's `Custody` declaration. A guessed vantage additionally writes a real measurement onto the wrong per-vantage timeline, which manufactures drift rather than merely wasting a probe |
| **Fail the whole fan-out on the first unnameable row** | One malformed `span.subject_key` would stop the `tls-acceptance` and `http-identity` `Scan`s for the entire estate, and `span.subject_key` is free `TEXT` that any writer can put anything into. It also inverts ADR-0206: a tick that could probe nine hundred targets would probe none and report an error, which is a dispatch failure standing in for one bad row |
| **Log every dropped row** | Right for the fold and wrong here. The fold's dropped line is evidence about an instrument outside this process; an enumeration's dropped row is a defect in our own schema use, and eight of the nine guards can never fire. The one case where volume matters is a mass drop, and a per-row line is the worst possible shape for it — it would arrive as one line per `Service` in the estate |
| **Count the drops and surface the count on a coverage screen** | A coverage surface answers *what did we measure*, and a dropped row was never a subject, so it has no place in a coverage denominator. ADR-0051 already ruled the same thing for an unkeyable address: absent from the recorded scope, no value and no `Gap`. Surfacing it would make an unreadable row of ours look like a measurement absence in the world |
| **State the rule inside ADR-0051 as a further clause** | ADR-0051 rules how a key is **formed** from what a source delivered. This rule is about what a reader does with a row it reads back, which is a different act with a different consequence — a probe rather than a subject. Filing it there would put a dispatch rule inside a document about key normalisation |
| **Merge with ADR-0154 into one drop-rather-than-guess ADR** | The two acts have opposite consequences for the same decision. ADR-0154's drop leaves a timeline **open** and its failure mode is a closure the operator cannot trace. This drop withholds a **probe** and its failure mode is a packet nobody authorised. A merged rule would have to state both and would be read for whichever half the reader arrived with |
| **Wait for the SPEC §8.10 dedup pass and write one ADR over all three neighbours** | The three grounds are different, and the dedup pass reads written ADRs. Leaving all three unwritten so that they can be considered together leaves the direction unstated for however long that takes, on a rule whose wrong branch dials a target |
