# A version leaf is a decision, not a binary — and a prober's corpus is authored

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#49 Is the measurement binary a versioned Derivation, and what is its golden corpus?](https://github.com/winniel123/verge-asm/issues/49)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0011](./0011-a-facet-is-six-parts.md) ruled that **any facet value requiring more than one
measurement to establish is decided by the measurement binary inside a single batch, never
assembled afterwards** — because assembling `Shadowed` from this name's answer plus the ~~parent
zone's~~ **parent name's** poison signature (the control probe runs under a name's **parent**, not at
a zone boundary —
[ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md))
puts a cross-observation dependency, with its own currency problem,
inside the comparison path. It named the price and did not pay it: *the measurement binary is now
load-bearing for four facet values, so a prober version is an input to what a value is.*

Under [ADR-0008](./0008-derivation-versions-move-on-content.md) that makes the prober a leaf in
the version vector, and the vector is what decides comparability. Both halves of ADR-0008's answer
looked harder here than anywhere they had been applied before.

- **A prober version moves constantly, for reasons that touch nothing.** A fix to the SSH push
  path or the NDJSON writer has no bearing on whether a name is `Shadowed`. Without a gate, every
  prober release breaks every timeline the binary feeds, which is the whole estate.
- **Its corpus rows are not structured observations.** Every corpus row imagined so far — a
  canonicaliser's, a signal's, the 38-pair list's — is a decoded, structured value. A prober's
  input is bytes off a socket. ADR-0011 confines wire-handling to **decoders**, so every
  *canonicaliser* corpus stays structured and the raw-bytes problem is entirely the prober's.
- **And the question underneath both**: whether a corpus that cannot be hand-written is still a
  corpus a **human** can judge, since the judgement staying human is the only reason ADR-0008's
  gate is not a code hash.

The stated fallback — bump the prober every release and take a `Break` on every timeline in the
estate — is not survivable, and this ADR exists because it may not be left as the answer.

Two constraints arrive with the ticket. [#5](https://github.com/winniel123/verge-asm/issues/5)
chose **one** measurement binary rather than two, under the map's *every seam is a place drift can
be manufactured* rule, so any answer that splits the binary has to answer to it. And
[#31](https://github.com/winniel123/verge-asm/issues/31) could not start Docker Desktop on the
machine that produced `insecure-listener-rules`, so every wire-format claim in that note is
**spec-verified rather than measured** — which means a capture procedure assuming live listeners
carries an unbudgeted prerequisite this project has already failed once.

## Decision

| Concern | Decision |
| --- | --- |
| Is the binary a `Derivation`? | **No.** The binary stays an instrument; **five named decision procedures inside it** are leaves |
| The test for a leaf | A part of the binary is a leaf exactly where **its output can move while the world does not** |
| The leaves | `connect-outcome` · `tls-handshake` · `http-exchange` · `resolution-walk` · `wildcard-discrimination` |
| What is outside every leaf | The job-spec parser, the NDJSON writer, the SSH push, concurrency, rate-limiting and adaptive back-off |
| Where the vector is recorded | On the **`Batch`**, beside its completed scope — a leaf's content is fixed when the measurement happens, not when the fold reads it |
| A corpus row | `(job-spec fragment, authored peer script, expected NDJSON)` plus a **one-line claim in prose** |
| Corpus medium | **Authored** in each protocol's own textual form — never a captured transcript, never a live listener |
| Where the corpus runs | Hermetically, against an **in-process scripted peer**: no network, no containers, no fixture images |
| The gate | **Bidirectional.** Output moved and the version did not → fail. Version moved and nothing justifies it → fail |
| What justifies a bump | A **moved corpus row**, a changed **declared parameter**, or a recorded **uncovered move** naming the input class the corpus cannot reach |
| Declared parameters | Timeouts, retry counts, control-label counts — and a **third-party wire library, but only where it speaks the protocol on our behalf** |
| Strictly-additive relief | [ADR-0011](./0011-a-facet-is-six-parts.md)'s rule applies to prober leaves unchanged, checked on the same corpus |
| The unsurvivable fallback | **Structurally unavailable** — no prober leaf is composed by every timeline, and only `connect-outcome` reaches the exposure board |

## Rationale

### ADR-0011's criterion was the wrong one, and correcting it adds a fifth value

ADR-0011 named four prober-decided values because it was answering a different question: which
values cannot be assembled by a canonicaliser from **one** observation. *Needs two measurements* is
the right test for *where the decision happens*. It is the wrong test for *what needs a version*.

The test that matters is the map's founding fear said out loud: **can this thing's output move
while the world does not?** Apply it and `reachability` is in.
[#4](https://github.com/winniel123/verge-asm/issues/4) specifies a **3 s connect timeout with 2
retries**; a release that halves it moves `connected` → `no-response` on every slow host in the
estate with nothing having changed. That is a single-measurement value, so ADR-0011's list passed
over it, and it is the highest-stakes decision the binary makes: `reachability` is the only prober
output that reaches `Reach`, `Exposure` and the board.

So the four are five, and the fifth was hiding behind a criterion chosen for another purpose. The
converse also holds and is what keeps the answer small: the NDJSON writer, the job-spec parser and
the SSH push cannot move an output with the world unchanged unless they are simply broken, and a
bug is not an output-affecting change — [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) already
rules that history is never re-derived and a fix ships as a version and a `Break`.

Adaptive back-off is the interesting near-miss. [#4](https://github.com/winniel123/verge-asm/issues/4)
makes it mandatory, and it is **outside** every leaf, because it halves the *rate* and never the
per-connection deadline. Had it moved the deadline it would have made `connect-outcome`'s output a
function of how busy the run was, which is a value moving because of us in the most literal
available sense.

### A leaf is named for a decision, and #5's seam rule is about implementations

The reflex objection to several leaves is the map's own: *every seam is a place drift can be
manufactured*, and #5 chose one binary rather than two on exactly that ground. It does not bite,
and saying why is load-bearing because a future session will feel the same reflex.

#5's rule is about **two implementations of one measurement** — two codebases that must agree, and
whose disagreement surfaces as a change in the world. Nothing here splits an implementation. There
is still one binary, one build, one `GOARCH` matrix, one NDJSON contract, pushed to the external
vantage by the instance so that skew is impossible ([#14](https://github.com/winniel123/verge-asm/issues/14)).

> **Rider added 2026-08-15 by [#124](https://github.com/winniel123/verge-asm/issues/124): *one
> `GOARCH` matrix* is a claim this ADR's own gate has to earn, and it earns it only where the corpus
> is run per architecture.** An architecture the corpus is not run on ships these five leaves
> **unverified there**, and the failure is invisible in band precisely because the vector is doing
> its job: two installs on one release hold **equal** derivation vectors, so comparison is licensed,
> and the values were produced by two implementations. So **an architecture is in the matrix exactly
> where this corpus is run on it in CI** — which is `linux/amd64` and `linux/arm64` and nothing else.
> It is not hypothetical: the Go specification permits fused multiply-add and Go's `arm64` backend
> emits it where baseline `amd64` has none, so **`GOAMD64` is pinned at `v1`** and **a declared
> parameter expressed as a fraction is evaluated in exact integer arithmetic** — otherwise
> [#67](https://github.com/winniel123/verge-asm/issues/67)'s cure for `certificate-expiring`'s `N`
> becomes an architecture-dependent output with no leaf to name it. See
> [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §1.
What splits is the **name under which a decision is versioned**, and that is not a seam; it is
ADR-0008's vector doing the job it was built for.

The general rule, because getting it backwards collapses the whole vector: **a version leaf is
named for what it decides, never for the artefact that ships it.** `Exposure`, `availability`,
`currency`, every canonicaliser and every one of the ten signal rules ship in one image today. If
leaves were named for artefacts they would all be one leaf called `verge-asm`, a break would name
nothing, and [#22](https://github.com/winniel123/verge-asm/issues/22)'s *one visual treatment
carrying stated reasons* would have no reason to state.

### The five leaves, and the one that reaches the board

| Leaf | Decides | Feeds | Declared parameters |
| --- | --- | --- | --- |
| `connect-outcome` | `connected │ refused │ no-response` | `reachability` → `Reach` → `Exposure` | Connect timeout, retry count. **No wire library** |
| `tls-handshake` | Handshake completed, chain read off it, or `NoTLS`; per-candidate accept/reject | `certificate`, `tls-acceptance` | Handshake timeout; **the TLS library** |
| `http-exchange` | `Responded(status, Location, WWW-Authenticate, Server, title) │ NotHTTP` | `http-identity` | Request timeout, capped body read; **the HTTP client and parser**; and, added by [#124](https://github.com/winniel123/verge-asm/issues/124) and **both arriving with a value**, the **redirect policy** (valued at **not followed** — [#4](https://github.com/winniel123/verge-asm/issues/4) §4.3's own recommendation) and the **`User-Agent` string** (valued at the identifying string). Both were filed in #4 §9 as knobs the operator must be able to set, and both fail `CONTEXT.md`'s *none is ever operator-configurable*: following a redirect moves the `status` and the `title` this leaf decides, and a WAF that blocks unknown agents returns a different response, so the string moves `http-identity`. §9's rows are struck. The cost of the second is stated rather than hidden — an estate whose WAF blocks us records the WAF's identity, which is a true answer to *what does a client that names nothing meet*, and the alternative is the impersonation #4 §10 refuses |
| `resolution-walk` | `NameError │ NoData │ Lame │ Resolved(unordered address set)`, and per-nameserver `serves │ does-not-serve` | `resolution`, `dns-record` | Query timeout, retries, TCP fallback policy; **the DNS library**; and the **query path** — added by [#116](https://github.com/winniel123/verge-asm/issues/116) / [ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md), **shared with `wildcard-discrimination` and taking one value per `Batch`**, valued at **the `Vantage`'s configured recursive resolver**. It arrives with a value. Two riders it is not safe to omit: **`Resolved`, `NoData` and `NameError` are read on that path**, because this leaf makes *two* queries for one name — a delegation walk and a resolution — and nothing had said which answer is the value (**[measured]** they differ: `s3.amazonaws.com`'s authority answers a synthesised name with a CNAME and no address, a resolver with eight); and **the parameter does not govern the delegation walk**, whose direct queries are `Lame`'s own definition and may not be routed away |
| `wildcard-discrimination` | `Shadowed`, on any qtype | `resolution`, `dns-record` | Control-label count and construction, the match predicate. **The `Name`s the labels are generated under are not here and never were** — that population is the **seventh aperture input**, recorded on the `Batch` ([ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)), because it is a function of the batch's scope and a parameter is authored data. ~~The match predicate has **no value anywhere**~~ — **DISCHARGED** by [#111](https://github.com/winniel123/verge-asm/issues/111) / [ADR-0068](./0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md): it is **set equality on the RDATA set, per `(qtype, RR type)` component, at determinate components only**, and an indeterminate component is never consulted. The **determinacy verdict** is not a parameter and not aperture — it is a measured in-batch fact read off the control probe's own answers. **Control-label count and construction is DISCHARGED** by [#113](https://github.com/winniel123/verge-asm/issues/113) / [ADR-0069](./0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md): the set is ~~**5 random labels + 1 structured label**~~ **9 random labels + 1 structured label** ([#115](https://github.com/winniel123/verge-asm/issues/115), which **raises the count and leaves the construction alone**), each **exactly one label** and each run over the declared qtype set, the structured one being `<a>-<b>-<c>-<d>` over a random RFC 5737 documentation address. `3–5` was a **range**, and this gate diffs values — so the count becoming a value was required whatever the construction, and the value moving `5` → `9` is this gate working rather than an exception to it: it bumps `wildcard-discrimination` and `Break`s `resolution` and `dns-record`, **free while nothing has shipped**. #115's warrant is measured — the instability is **per-label sharding** at `surge.sh` and `appspot.com` and **per-query rotation** at `herokuapp.com` and `vercel.com`, per-time at neither, so a larger count of *distinct* labels is the one lever that buys ([`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §13). ~~**This leaf now has no parameter without a value**~~ — **that sentence is true and its *count* is superseded**: [#116](https://github.com/winniel123/verge-asm/issues/116) / [ADR-0070](./0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md) adds a **fourth**, the **query path**, shared with `resolution-walk` above and taking **one value per `Batch`**, valued at **the `Vantage`'s configured recursive resolver**. It is one parameter held jointly rather than two, and that is load-bearing: **[measured]** a control probe direct to `s3.amazonaws.com`'s authority reads `NoSynthesis` at A — a *determinate* reading — while a resolver answers every candidate beneath with eight addresses, so a skewed pair discriminates **every** fictional label and records it `Resolved`. **A control probe is asked from where the answer it discriminates was asked from.** This leaf still has no parameter without a value |

Two properties of that table are the whole answer to *what happens on the release where we cannot
gate this*, and they are structural rather than procedural.

**No leaf is composed by every timeline.** The worst single bump clamps one or two facet families.
There is no prober change that can `Break` the estate, so the fallback the ticket refused to accept
is not merely discouraged, it is unreachable.

**Only `connect-outcome` reaches the exposure board**, and it is the one leaf with no third-party
wire library in its parameter set. That is not luck. ADR-0008 already singled the `reachability`
canonicaliser out as the *non*-churny counterexample because it "maps a small closed value space";
the same smallness upstream means the kernel decides SYN-ACK, RST or silence and the three-way
mapping is ours. So the flagship value is fed by the one leaf a dependency upgrade cannot move.

The closest call in drawing the table was merging `resolution-walk` and `wildcard-discrimination`.
They break the same two facets, so the split buys nothing in blast radius. It is kept because a
break **names its leaf**, and *the wildcard discriminator changed* is a sentence an operator can
act on where *the resolution machinery changed* is not — ADR-0008's stated reason for flattening
the vector at all, applied one level down.

### The corpus is authored, because a captured transcript has no claim

The ticket assumed the corpus medium had to be captured wire transcripts, and that assumption is
what made the problem look impossible. It is wrong, and the argument that kills it is not about
hermeticity.

**A corpus row's reviewable content is its claim, and a capture has none.** An authored row says,
in the project's own words, *a name whose parent zone wildcards and whose answer matches the
control probe is `Shadowed`*, and the bytes are that sentence's implementation. A captured
transcript arrives as bytes with no claim attached, so somebody has to invent one afterwards — and
the invented claim is what actually gets reviewed. The capture bought nothing and cost
determinism.

Hermeticity is the second argument and it is nearly as strong. A corpus whose fixtures are **live
listeners in containers** is a corpus whose expected output moves when the fixture image moves.
That is ADR-0004's out-of-band reference data arriving in the test harness: the build fails for
reasons that are not ours, on a schedule we do not control, which is the fastest known way to
train a team to ignore a gate. It also cannot express the cases that matter — a half-open socket,
a truncated record, an authority that REFUSEs — because those are not configurations of real
software, they are hostile peers.

And the third argument is the one #31 already paid for. Docker Desktop would not start on the
machine that produced `insecure-listener-rules`, so a capture procedure assuming live listeners has
an **unbudgeted prerequisite** that this project has failed once already. The corpus is CI's gate
on every release; it may not depend on a thing that has already been unavailable.

What makes authoring tractable is that the input medium is not one medium, and treating it as one
is the rest of the mistake:

- **`resolution-walk` and `wildcard-discrimination` take DNS messages**, which have a canonical
  **presentation format**. A row is written the way a zone file or `dig` output is written. This is
  not a wire-transcript problem at all.
- **`http-exchange` takes HTTP/1.1**, whose wire *is* text.
- **`connect-outcome` takes no bytes whatever.** Its stimulus is a socket event and a clock:
  SYN-ACK, RST, or silence for *n* seconds.
- **`tls-handshake` is the only genuinely binary one**, and its value space is small enough to
  author against: a chain the harness generates from a described leaf and test CA, and a handshake
  outcome from a closed alphabet — completed, `protocol_version` alert, `handshake_failure`,
  non-TLS bytes, silence, close.

The peer is **in-process**, over a pipe or a loopback listener the harness owns. No network, no
container, no image. Capture is admitted only as **provenance on a row** — *this shape was seen at
such-and-such* — and never as the row's format.

One honesty rider, inherited rather than introduced. Because every wire-format claim in
`insecure-listener-rules` is spec-verified rather than measured, an authored row encoding one of
those claims **inherits that status and must say so on the row**. A row asserting that PostgreSQL
replies `S` is a claim about the specification until somebody measures it, and #21's habit —
*publish the weak tier rather than smoothing it* — applies to a corpus exactly as it applied to a
port list. This bites in v1.1 rather than v1, the wire-protocol prober being out of scope per
[#41](https://github.com/winniel123/verge-asm/issues/41), but the rule is written now because the
first session to write those rows will not re-derive it.

### Judgeable and hand-writable are two properties, on two sides of a row

The ticket's unasked question — *is a corpus that cannot be hand-written still one a human can
judge?* — contains a conflation, and separating it is most of the answer.

ADR-0008 needs the human to judge **the diff, not the fixture**. What CI renders on a failure is
the row's claim, the old output and the new one: *row 47 — a name under a wildcarded parent whose
answer matches the control probe — was `Shadowed`, is now `Resolved([203.0.113.5])`.* That is
judgeable whether the stimulus was typed by hand or lifted off a wire. **Judgeability is a property
of the output side of a row; hand-writability is a property of the input side.** They are
independent, and only the first is what stops the gate being a code hash.

Hand-writability is not thereby worthless — it is load-bearing for a different thing. A corpus you
cannot author is a corpus you cannot **extend to cover the case you just fixed**, so its coverage
is limited to what happened to occur in the wild against a listener somebody had. So both
properties are required, for two different reasons:

> The **output** side of a corpus row must be human-judgeable, because that is what keeps the gate
> a judgement. The **input** side must be human-authorable, because that is what lets the corpus
> grow to cover what the last bug taught us.

Authored peer scripts satisfy both. Captured transcripts satisfy neither cleanly. Live-listener
fixtures satisfy the first and fail the second, which is why they are the seductive wrong answer.

The corollary is a coverage story a reviewer can actually audit: because every row carries a claim,
the corpus reads as a list of sentences, and a **missing** claim is visible in a way a missing
transcript never is.

### The gate runs both ways, which is what stops a version moving for nothing

ADR-0008's gate is one-directional: if an output moves and the version did not, the build fails.
That is the right gate for a canonicaliser, whose input space is a small closed union. It is not
sufficient for a prober, because the ticket's first worry is the **opposite** failure — a version
moving constantly for reasons that touch nothing.

So the gate is bidirectional, and the second direction is new:

> A leaf's version may move **only** if a corpus row's output moved, a **declared parameter**
> changed, or an **uncovered move** was recorded naming the input class the corpus cannot reach.

All three are mechanically checkable, because the parameter set is declared data and the uncovered
move is a checked-in entry rather than a commit message. A bump nothing justifies fails the build
exactly as an unbumped move does. That converts *"do not bump gratuitously"* from discipline —
which this project refuses everywhere in the comparison path — into a failing test, which it has
never refused.

The parameter route matters more than it looks, because it absorbs the biggest thing a corpus
**cannot** cover. A timeout change moves outputs on hosts the corpus does not contain, by
construction: the corpus's clock is scripted. Under ADR-0008 parameters live inside their
derivation's leaf, so a timeout change bumps `connect-outcome` mechanically, with no judgement
asked and no corpus row expected to move.

### A library is a parameter where it speaks for us

The residual un-coverable case is a dependency upgrade. Go's `crypto/tls` gains or drops a cipher
and the offer changes; `net/http` tightens header parsing and the `NotHTTP` boundary moves; a DNS
library changes its EDNS defaults. None of that is our code and all of it can move a value.

**A third-party wire library is a declared parameter of a leaf exactly where the library speaks the
protocol on our behalf.** `crypto/tls` chooses the ClientHello, so it is a parameter of
`tls-handshake` — which is the same hazard ADR-0011 already caught when it moved negotiated TLS
parameters off `certificate`, arriving here through the dependency manifest instead of the value
space. `net/http`'s parser decides what counts as HTTP, so it is a parameter of `http-exchange`.
The DNS library's retry and EDNS behaviour is a parameter of `resolution-walk` and
`wildcard-discrimination`. The dialler decides nothing — the kernel returns SYN-ACK, RST or
nothing, and the three-way mapping is ours — so `connect-outcome` has **no** library parameter.

The line has to be drawn there and not one step wider. *The toolchain version is a parameter of
every leaf* is the tidier rule and it is ADR-0008's rejected *versions move on releases*
re-entering through `go.mod`: two Go releases a year would clamp the board twice a year for
changes with no bearing on a TCP connect.

Note what a Go upgrade then costs, because it is the worked example of this whole ADR and it is not
free. `tls-handshake` bumps, `Break`ing `certificate` and `tls-acceptance` timelines and clamping
`tls-1.0-accepted`; `http-exchange` bumps, `Break`ing `http-identity`. The board is untouched. And
it fires **two messages of one class**: widening the TLS offer is an aperture change yielding
`revealed` ([ADR-0011](./0011-a-facet-is-six-parts.md) made the candidate set batch scope), while
the leaf bump is a re-baseline — [ADR-0014](./0014-only-revealed-generalises.md) merged those into
one class with two triggers and warned that the payloads must not be levelled. They land on
different objects and compose cleanly, exactly as
[ADR-0009](./0009-verge-core-is-a-union.md) found for a sensitive-list revision.

*Corrected by [#54](https://github.com/winniel123/verge-asm/issues/54) /
[ADR-0025](./0025-an-offer-is-scope-only-where-the-value-enumerates-it.md), which is the ticket the
consequence below opened.* The two prices do **not** land on different objects — both land on the
`tls-acceptance` timeline of the same `Service` — so they would have doubled rather than composed,
and ADR-0009's finding does not transfer. It is moot, because a Go upgrade **cannot widen the TLS
offer**: the candidate set is declared by us and passed to the library explicitly, so the upgrade
bumps `tls-handshake` and fires **one** message. The rule that makes it moot is that **a default is
not a declaration** — a parameter whose value is *whatever the library defaults to* is ADR-0008's
rejected hash-of-the-parameters, and an aperture dimension recorded as a library version cannot
tell a widening from a narrowing. So every offer the binary makes — the ALPN list, the TLS
candidate set, the EDNS options and buffer, the qtypes, the transport fallback — is enumerated in
the job spec and passed down; the library parameter covers only what remains its to decide. The
three offers named in the consequence below are all **parameters**: none carries a per-candidate
negative in any value.

### The release where we cannot tell

There remains a release where an author genuinely cannot say whether output moved, and no
parameter changed. It is the case the ticket demanded an answer to, and the answer has three parts,
none of which is *take the estate-wide break*.

**First, the honest default is to bump**, and bumping is now affordable because it is a leaf. The
cost is one facet family clamped for one cadence, with the census rendering throughout
([ADR-0008](./0008-derivation-versions-move-on-content.md)) and durations rendering as labelled
floors. That is a real cost paid by a real population, and it is a fifth of what it was.

**Second, most of what looks un-coverable is strictly additive**, and ADR-0011's rule applies to
prober leaves without modification: where every corpus row whose output moved previously produced
**no observation at all**, there is no `Break`, only a re-baseline message. That is the mechanism
that carries a new protocol, a new variant, or the wire-protocol prober whenever
[#41](https://github.com/winniel123/verge-asm/issues/41)'s deferral is revisited. CI checks it, so
it is a mechanism and not a judgement.

**Third, where neither applies, the uncovered move is recorded rather than resolved.** The entry
names the input class the corpus cannot reach — *behaviour against servers that close on an
unrecognised extension* — and it is data, so it is reviewable and countable. If uncovered moves
become the common case, the corpus is failing and the count says so out loud, which is #21's
*publish the weak tier* applied to our own instrument.

This is the thinnest part of the ADR and it is marked rather than dressed. Nobody has run a prober
corpus on this project, so how often the third branch fires is unmeasured, and the honest position
is that the mechanism is sound and its frequency is a guess.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) changes in two places.** `Derivation` admits measurement-time
  decisions and states the leaf test, since its current wording — *the procedure that produced a
  Derived value* — excludes every leaf named here while ADR-0011 has already made them
  version-bearing. `Batch` records the leaf versions it ran under, beside its completed scope.
- **`Break`, `Facet` and `Span` are deliberately untouched.** A prober leaf bump is a `Derivation`
  vector change, which is already `Break`'s first cause; a facet is still six parts, because a
  facet fed only by the operator's zone file has no prober leaf at all; and the `Span` already
  carries the vector.
  *(Originally "a facet fed only by `crt.sh`" —
  [ADR-0027](./0027-a-source-may-admit-without-observing.md) found CT feeds no facet at all.
  The point is unchanged and the zone file is the surviving instance.)*
  Nothing here needs a seventh part or a third break cause.
- **[ADR-0011](./0011-a-facet-is-six-parts.md)'s count of four prober-decided values is amended to
  five.** `connected │ refused │ no-response` joins `Shadowed`, `Lame`, `NoTLS` and `NotHTTP`, on a
  different criterion — not *needs two measurements* but *can move with the world unchanged*. The
  old count stands unrewritten in #36 and ADR-0011's own text.
- **[ADR-0008](./0008-derivation-versions-move-on-content.md)'s corpus gate gains a second
  direction, for every derivation and not only the prober's.** *Version moved and nothing justified
  it* fails the build. A canonicaliser has less need of it, but there is no reason to scope it and
  a uniform gate is one rule.
- **Three corpora sit in a chain, on three named interfaces, and this is what makes the prober's
  finite.** The prober leaf's corpus runs scripted peer → NDJSON; the decoder's runs NDJSON →
  value space; the canonicaliser's runs value → canonical form.
  [#5](https://github.com/winniel123/verge-asm/issues/5) fixed that NDJSON contract for unrelated
  reasons — keeping the operator's attack surface off the VPS disk — and it is the seam that lets
  the wire stop at the binary's edge.
- **A `Signal` reading a prober-decided value composes that value's prober leaf**, by
  [ADR-0004](./0004-signals-are-release-coupled-rules.md)'s existing composition rule and with
  nothing new required. Where a rule reads two timelines from two sources it composes the
  **union** of their leaves, so a source with no prober leaf — an operator's zone file — does not
  shield the rule from the other side's bumps.
  *(`crt.sh` was named here too and is removed by
  [ADR-0027](./0027-a-source-may-admit-without-observing.md): it holds no timeline, so no rule can
  read a side of it.)*
- **Retention inherits a break cause it cannot control.** ADR-0008 recorded that a break inside the
  retained window makes `returned` detection unrecoverable. Prober leaves move on a **dependency**
  cadence rather than on our own decisions, which is faster than any canonicaliser, so whoever
  sizes retention is sizing against `go.mod` and not against the project's roadmap.

  > **Discharged by [#121](https://github.com/winniel123/verge-asm/issues/121) ·
  > [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md), and the
  > answer is that **nobody** sizes retention against the break cadence.** Two things close it.
  > `Span`s are never compacted, so the corpus the sizing was for has no window to size; and
  > **retention may never be the tighter clamp** — where a retention rule would truncate earlier
  > than the `Break` clamp already does, it does not truncate at all, because the break clamp is
  > visible and names the leaf while a retention horizon is invisible. What survives is this
  > paragraph's real finding, promoted from a sizing input to a **release obligation**: the leaf on
  > the dependency cadence that reaches membership is `resolution-walk`, so **its golden corpus is
  > what keeps `returned` alive**. ADR-0008's gate bumps a leaf only where a corpus row's output
  > moved, so a DNS-library upgrade that cannot change `Resolved` / `NoData` / `NameError` / `Lame` /
  > `Shadowed` must **provably** not bump it — which needs corpus rows pinning those outcomes
  > specifically. An under-covering corpus no longer costs a spurious `Break`; it costs estate-wide
  > loss of the one transition that tells a returning host from a new one.
- **The corpus is a build-time artefact per leaf**, like every other named derivation's, and it
  runs with no network and no containers — which is a hard requirement rather than a preference,
  and the one place this ADR constrains CI.
- **Enumerating the leaves opened [#54](https://github.com/winniel123/verge-asm/issues/54), which
  blocks [#12](https://github.com/winniel123/verge-asm/issues/12).** ADR-0011's argument for making
  the TLS candidate set batch scope is not about TLS, and nobody has asked whether the ALPN list,
  EDNS options and the DNS transport fallback are aperture inputs on the same grounds. Naming a
  library as a leaf **parameter** here creates a second home competing for the same fact, so the
  two mechanisms have to be reconciled rather than assumed to compose.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| The binary is one `Derivation` leaf | Leaves are named for decisions, not artefacts — and one leaf reinstates the estate-wide `Break` per output-affecting release that this ticket exists to refuse |
| The binary is an instrument and carries no version at all | Its timeouts and its ClientHello decide values; ADR-0011 already priced this and the layer worry is answered by versioning the decisions rather than the instrument |
| A content hash of the binary | ADR-0008 rejected code hashes for bumping on a refactor; here it is worse, since a binary hash moves on every build and dresses the unsurvivable fallback as a mechanism |
| Four leaves, one per value ADR-0011 named | Leaves out `reachability`, the highest-stakes prober decision in the product, because ADR-0011's criterion was chosen for a different question |
| Merging `resolution-walk` and `wildcard-discrimination` | Costs nothing in blast radius and loses the break's ability to name an actionable leaf — the closest call in the table |
| A blast-radius predicate scoping a prober break to affected subjects | ADR-0008 refused it on failure mode: too narrow fails **silently**, emitting our own release as `Transition`s |
| Captured wire transcripts as corpus rows | A capture carries no claim, so the reviewable content is invented afterwards; and it needs the live listeners #31 could not start |
| Live listeners in containers as CI fixtures | The expected output moves when the fixture image moves — ADR-0004's out-of-band reference data inside the test harness, failing the build for reasons that are not ours |
| One corpus medium for the whole binary | Three of the five leaves take a textual protocol or no bytes at all; treating them as one raw-bytes problem is what made this look unsolvable |
| The toolchain version as a parameter of every leaf | ADR-0008's rejected *versions move on releases* arriving through `go.mod`; it would clamp the board twice a year for changes with no bearing on a TCP connect |
| Adaptive back-off inside `connect-outcome` | It halves the rate, never the deadline — and had it moved the deadline, a value would depend on how busy the run was |
| A one-directional gate, as ADR-0008 wrote it | Catches the unbumped move and never the gratuitous bump, which is the ticket's *first* worry |
| Recording the leaf vector only on the `Span` | A leaf's content is fixed when the measurement runs; the `Batch` is already the record of one execution from one vantage |
