# ADR-0208: The queue reads a `Service`'s subject and never re-parses its rendering, so a rendered key is an identity token alone

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1320 ADR gaps: internal/queue (#1200, sweep 6/7)](https://github.com/winniel123/verge-asm/issues/1320), gap 3
- **PR that deleted the comment:** [#1324](https://github.com/winniel123/verge-asm/pull/1324)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on, and applies:** [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md), which rules that *"a composed key holds the subject, never its rendering"* and names this exact hazard — *"a decoder that helpfully renders — hands on a string, and the key function is left to re-parse our own output."* This ADR agrees with it and applies it at the queue's read path. **No [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) withdrawal is owed on ADR-0051, and a reader should not go looking for one.** §4 states why
- **Rests on:** [ADR-0007](./0007-drift-is-a-timeline-of-spans.md), which makes a `Span` a timeline identified by its subject and facet. `span.subject_key` is that timeline's identity, and this ADR rules what else it may be used for
- **Sibling of, and not ruled by:** [ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md). That ADR rules the `tls-acceptance` `Scan`'s cadence and aperture — weekly, over the open `Service` population, never a port tier. It is silent on the subject key's wire form, and this ADR adds no cadence or aperture clause
- **Bounds:** [ADR-0207](./0207-an-enumeration-that-assembles-a-probing-target-set-drops-a-row-it-cannot-fully-name-and-never-fabricates-a-target.md). Its drop rule stays exactly as ruled. This ADR removes one of its drop sites' inputs, which is the one site where that drop could fire on every row at once

## Context

`internal/queue/tlsacceptance.go:83` carried this in Go declaration position, until #1324 deleted it:

```go
// parseServiceKey folds a `address:port/tcp` Service subject key back to its
// `(Address, port)`. It is the inverse of the ServiceKey the connect-outcome leaf
// renders. A key without the `/tcp` transport suffix, or one whose address:port does
// not parse, is rejected rather than guessed at.
```

#1324 compressed it to one surviving line at `internal/queue/tlsacceptance.go:66`:

```go
// The inverse of the ServiceKey the connect-outcome leaf renders, so the two move together.
```

The survivor asserts a coupling rule and carries no citation. Nothing under `docs/adr/`,
`docs/spec/`, `docs/guides/` or `CONTEXT.md` states it. That is #1320's gap 3.

### The record says one renderer. There are three

#1320's body says the form *"has exactly one renderer (the `connect-outcome` leaf) and one parser"*.
That is wrong, and two of the three renderers say so themselves in comments #1324 kept.

| Leaf | Renderer | Its own surviving comment |
| --- | --- | --- |
| `connect-outcome` | `ServiceKey` (`internal/measure/connectoutcome/emit.go:12`) | *"The Address renders from its own key, never a restated string, so one host has one spelling."* |
| `tls-acceptance` | `ServiceKey` (`internal/measure/tlsacceptance/emit.go:17`) | *"The connect-outcome leaf renders the same triple, so the two timelines name one Service."* |
| `http-exchange` | `Target.ServiceKey` (`internal/measure/httpexchange/exchange.go:25`) | *"The same triple connect-outcome renders, so the Endpoint's Service leg names one Service."* |

`connectoutcome.ServiceKey` also composes the `Endpoint` key through
`internal/measure/connectoutcome/certificate.go:19`, and `httpexchange.EndpointKey` composes the
other one, so the form reaches two subject kinds rather than one.

**The three are not identical, and the difference is the hazard arriving.**
`httpexchange.Target.ServiceKey` takes an `Address string` and falls back to
`t.Address + ":" + strconv.Itoa(int(t.Port)) + "/tcp"` when the address does not parse as an IP. Its
range is therefore wider than `netip.ParseAddrPort`'s domain. The renderer and the parser are
already not inverses, in the shipped tree, on one of the three sides.

### The parser's real input, measured rather than assumed

`reachedServices` reads `ListReachedServices` (`db/queries/span.sql:165`), which filters
`subject_kind = 'service' AND facet = 'reachability' AND closed_at IS NULL AND is_gap = FALSE AND
(value ->> 'outcome') = 'reached'`.

`connectoutcome.FacetReachability` is the only facet constant with that value, so the rows the
parser reads today were written from `connectoutcome.ServiceKey`. That narrows the **input** and not
the **coupling**: `subjectKindFor` (`internal/queue/pure.go:81`) maps both
`connectoutcome.FacetReachability` and `tlsacceptance.Facet` to `subject_kind = 'service'`, and
`span.subject_key` is free `TEXT` with no constraint on its shape. Any leaf that renders the form
writes rows into the same column, and any future leaf that renders it joins the contract on the day
it ships.

### The round trip, counted

One reached `Service` crosses the render/parse boundary five times in one cycle — three renders and
two parses — for a value the writer already held in its measured form.

| # | Act | Site |
| --- | --- | --- |
| 1 | render `netip.AddrPort` → `"198.51.100.7:443/tcp"` | `connectoutcome.ServiceKey`, at emit |
| 2 | store the string | `foldOne` → `OpenSpan` (`internal/queue/spanfold.go:88`) |
| 3 | **parse** the string → `netip.AddrPort` | `parseServiceKey` (`internal/queue/tlsacceptance.go:65`) |
| 4 | render again → `ReachedService.Address string` | `reachedServices` (`internal/queue/tlsacceptance.go:58`) |
| 5 | **parse** again → `netip.Addr` | `BuildTLSAcceptanceJobs` (`internal/scan/tlsacceptance.go:60`), `BuildHTTPIdentityJobs` (`internal/scan/httpidentity.go:56`) |
| 6 | render again, at the next emit | `tlsacceptance.ServiceKey` / `httpexchange.Target.ServiceKey` |

**The fold already holds the address and throws it away.** `wire.Observation` carries an `Address`
field, `connectoutcome.EmitService` sets it to `target.Addr().String()`, and `foldOne`
(`internal/queue/spanfold.go:36`) never reads it. So step 3 exists to recover a value the writer
had, sent, and the reader discarded one function earlier.

### The failure is silent, and no test would catch it

No file under `internal/queue/*_test.go` names `parseServiceKey`, `reachedServices` or `ServiceKey`.
Nothing pins the round trip in either direction.

A change to the rendered form on any of the three renderers, or to `parseServiceKey`'s expectations,
makes every parse fail. ADR-0207's drop rule then fires correctly on every row, `reachedServices`
returns an empty slice, and `fanOutTLSAcceptance` and `fanOutHTTPIdentity` enqueue zero jobs.
ADR-0206 rules that a zero-job fan-out is a legible success. So the two rules that are individually
right compose, here, into a whole `Scan` silently ceasing to probe, reported as a healthy empty tick.

### `parseServiceKey` is not the only decomposition, and the wider count is the argument

Two functions decompose a composed key — `parseServiceKey` and `serviceAddress` — and they are
reached from seven call sites in six files.

| Site | Act |
| --- | --- |
| `parseServiceKey` (`internal/queue/tlsacceptance.go:65`) | `CutSuffix("/tcp")` then `netip.ParseAddrPort` |
| `serviceAddress` (`internal/queue/membership.go:293`) | `LastIndex(":")`, then trim `[` and `]` |
| `subjectAddress` (`internal/queue/membership.go:209`) | calls `serviceAddress` for `service` and `endpoint` keys |
| `membership.go:205` | address membership of a subject |
| `withdrawal.go:50` | the address-exclusion narrowing fold |
| `seedwithdrawal.go:74` | the address `Seed` withdrawal fold |
| `produce.go:357` and `:359` | a citation lookup, and a `HasPrefix` containment test on the key text |
| `scopegate.go:113` | the scope gate's address for a `service` key |

Every one of them recovers a component of the subject by taking our own rendering apart. This is
ADR-0051's named hazard, arrived at, and it is a package habit rather than one function's slip.

## Decision

> **A `Service`'s rendered subject key is an identity token. It is written by the leaf that measured
> the subject, it is compared for equality, joined on, and displayed — and it is NEVER decomposed to
> recover the subject. A reader that needs a `Service`'s address, port or transport reads the
> subject the writer supplied, from a column that carries it. The queue does not parse
> `span.subject_key`.**

### 1. The `connect-outcome` leaf owns the rendered form, and two more leaves follow it

`connectoutcome.ServiceKey` (`internal/measure/connectoutcome/emit.go:12`) is the definition. It
renders `(Address, port, transport)` as `<addr:port>/<transport>` from the `netip.AddrPort` it
measured, which is what its own comment means by *the Address renders from its own key, never a
restated string*.

`tlsacceptance.ServiceKey` and `httpexchange.Target.ServiceKey` render the same form **so that their
observations land on the same timeline**, and that is their whole reason to exist. They are
followers of the form, not co-owners of it. A change to the form is a change to
`connect-outcome`'s definition, and the other two follow it in the same change.

Three renderers is a fact about the code, and this limb is why it costs nothing: they agree because
they name one `Service`, not because a reader downstream depends on their agreement.

### 2. The rendered form is for identity and display, never for re-derivation

| Use of `span.subject_key` | Permitted |
| --- | --- |
| Equality between two spans of one timeline | **Yes.** It is the timeline's identity (ADR-0007) |
| The `span_open_timeline_idx` unique index | **Yes** |
| A lookup or a join on the whole string | **Yes** |
| Rendering it to an operator | **Yes.** It is a rendering, and that is what renderings are for |
| Recovering the address, the port or the transport | **No** |
| A prefix or suffix test that infers a component | **No** |

The line is *whole token* against *parts of the token*. The moment a reader needs a part, it needs
the subject, and the subject is not in that column.

### 3. The replacement route the reader must take

**A `service` span carries its subject's components in columns, and `reachedServices` reads them.**

The route, named so it is not left as a direction of travel:

1. `wire.Observation` gains the port beside the `Address` it already carries. Under
   [ADR-0151](./0151-a-field-no-emitter-renders-is-cross-leaf-plumbing-and-plumbing-moves-no-leaf-version.md)
   that field is cross-leaf plumbing, so it moves no leaf version and causes no `Break`.
2. `span` gains the subject's components for a `service` subject, and `foldOne` writes them from the
   observation it already receives and currently discards.
3. `ListReachedServices` selects those columns.
4. `reachedServices` maps the rows straight to `scan.ReachedService`, and `parseServiceKey` is
   deleted.

That is the cost, stated plainly: **a column or two on `span`, a field on the wire observation, and
one fold write.** It buys the removal of a parser, a re-render and a second parse from the hot path,
and it ends the coupling between three renderers and one reader permanently rather than for as long
as everyone remembers.

The columns are additive. `subject_key` stays exactly as it is, because it is the timeline's
identity and every index, join and screen that uses it is correct under §2.

### 4. ADR-0051 is applied, not amended, and no withdrawal is owed anywhere

**ADR-0051 needs no ADR-0058 withdrawal, because this ADR agrees with it.** ADR-0058 withdraws a
**superseded** mechanism at the site that specifies it. Nothing in ADR-0051 is superseded here. Its
composed-key rule — *a composed key holds the subject, never its rendering* — is the ground this
Decision stands on, and its policing sentence names the shipped parser before the parser existed. A
later reader who arrives at ADR-0051 looking for a withdrawal clause will find none, and that is
correct.

`CONTEXT.md`'s `Service` entry already states the same thing: *"The `Address` in it is the
**subject**, never a rendering of one … A triple built from a string would put a second normalisation
site in the model, and two spellings back on one host."* No glossary edit is owed either.

The choice this ADR was asked to make was between blessing the parse-back as a persistence boundary
and refusing it. It refuses it, on two grounds.

**Blessing it would overturn a live ADR to legalise the precise failure that ADR was written to
prevent.** ADR-0051 did not merely fail to anticipate the parse-back. It described it, in the
present tense, as the shape to police against. An ADR that blessed it would have to say that the
sentence was wrong, and nothing in the queue's read path is evidence that it was.

**Blessing it would bind three renderers to one parser forever, and the set is open.** A persistence
boundary is a contract, and a contract has parties. The parties here are `connect-outcome`,
`tls-acceptance`, `http-exchange` and every leaf that renders the form later — and the shipped tree
already shows the contract broken on one side, since `httpexchange.Target.ServiceKey`'s fallback
branch emits a form `parseServiceKey` rejects. Reading the subject has one party and no contract.

### 5. What this rule does not reach

- **The rendered form itself.** `<addr:port>/<transport>` is unchanged. This ADR moves no key and
  re-keys nothing, which is exactly what keeps it clear of ADR-0051's expensive re-key migration.
- **`subject_key`'s type and its role.** It stays `TEXT`, it stays the timeline identity, and it
  stays what the open-span unique index runs on.
- **The `Endpoint` key's own components.** An `Endpoint` key embeds a `Service` key, so decomposing
  one to recover an address is the same act and the same ground reaches it. The `Name` leg's own
  form is ADR-0051's stated residue and is not decided here.
- **Drift, `Break`s and comparison.** Nothing in the comparison path changes. No `Derivation` leaf
  moves, and ADR-0151 is why the plumbing field in §3 does not move one either.
- **The narrowing folds' decisions.** ADR-0153 and ADR-0154 rule which mover a fold takes and what it
  does with a candidate it cannot attribute. Both stay exactly as ruled. §Context's seven-site table
  names where those folds obtain an address today, and changing that route changes no outcome they
  rule.
- **A leaf's own use of its own value.** A leaf composing a key from the `netip.AddrPort` it just
  measured is the correct direction and is what §1 describes.
- **An equality comparison between two whole rendered strings.** The recording-side scope gate
  compares the address a job carried against the address an observation names. That is §2's
  permitted use — the whole token, never a part of it — and the rule that the two spellings must
  agree is [#1319](https://github.com/winniel123/verge-asm/issues/1319) gap 2's, not this one's.

## Consequences

- **This ADR changes no Go code and no SQL.** It rules a contract that the code does not yet keep.
- **The shipped parse-back is a defect, and it is not fixed here.** Named precisely:
  - `parseServiceKey` (`internal/queue/tlsacceptance.go:65`) — the parser itself.
  - `reachedServices` (`internal/queue/tlsacceptance.go:41`) — its only caller, and the enumeration
    that depends on it.
  - `ListReachedServices` (`db/queries/span.sql:165`) — the reachability span read that supplies the
    parser its input, and the query that must return the subject's components instead.
  - The three renderers the parse-back couples to: `connectoutcome.ServiceKey`
    (`internal/measure/connectoutcome/emit.go:12`), `tlsacceptance.ServiceKey`
    (`internal/measure/tlsacceptance/emit.go:17`) and `httpexchange.Target.ServiceKey`
    (`internal/measure/httpexchange/exchange.go:25`).

  It ships as its own ticket, taking §3's four steps. **This branch fixes none of it.**
- **The defect is wider than gap 3 recorded, and the wider span is one ticket's second half.** Six
  further sites recover an address by taking a composed key apart: `serviceAddress`
  (`internal/queue/membership.go:293`), `subjectAddress` (`internal/queue/membership.go:209`) and its
  callers at `membership.go:205`, `withdrawal.go:50`, `seedwithdrawal.go:74`, `produce.go:357` and
  `:359`, and `scopegate.go:113`. They are the same act on the same key. Their route changes and
  their outcomes do not, so ADR-0153's and ADR-0154's rulings are untouched by the fix.
- **`httpexchange.Target.ServiceKey`'s fallback branch is a live inconsistency, and it is named
  rather than fixed.** It renders `<address>:<port>/tcp` for an address that is not an IP, and
  `parseServiceKey` rejects that form. Under §2 it stops mattering, because nothing parses the
  output. It is recorded so the fix ticket does not discover it late.
- **The silent-empty-population failure closes with the parser.** §Context measures how ADR-0206 and
  ADR-0207 compose into an undetectable stopped `Scan`. Removing the parse removes the composition,
  and neither of those two rules has to be weakened to achieve it.
- **Nothing pins the round trip, and nothing needs to afterwards.** No test names the parser today. A
  round-trip golden corpus would be the alternative to §3 and is rejected below, so the fix ticket
  ships no such corpus.
- **`CONTEXT.md` gains nothing and ADR-0051 gains no withdrawal.** Both already say what this rules.
  ADR-0051 gains one pointer line so a reader at its policing sentence learns where the rule was
  applied.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Bless the parse-back as a persistence boundary, and bind the renderer and the parser as a contract** — the ticket's other option | It overturns a live ADR in order to legalise the precise failure that ADR was written to prevent: ADR-0051 names *a decoder that helpfully renders, leaving the key function to re-parse our own output* as the shape to police, and the shipped parser is that shape. It also binds **three** renderers plus every future one to a single reader, and the contract is already broken on one side, since `httpexchange.Target.ServiceKey`'s fallback emits a form `parseServiceKey` rejects |
| **Keep the parser and pin it with a round-trip golden corpus** | Pins the defect rather than removing it, and it cannot pin what matters: the corpus would lock one renderer's output against the parser, while the form's agreement across `connect-outcome`, `tls-acceptance` and `http-exchange` is what a future change breaks. It also makes the coupling official, so the next leaf that renders the form inherits a corpus obligation instead of nothing at all |
| **Have the queue import a leaf's `ServiceKey` and compare rendered strings rather than parsing** | Still re-derivation, and worse: it puts an import edge from the dispatcher to three measurement leaves, and it recovers the address by rendering candidates until one matches, which needs the address it is trying to find |
| **Change the rendered form to something with an unambiguous delimiter, so the parse is safe** | Keeps the re-derivation and adds a re-key of every `service` and `endpoint` span in the estate. ADR-0051 describes that migration as deliberately outside every mechanism the model has, and spending it to make a parse we are removing anyway more reliable is the worst available trade |
| **Store the address and port inside a JSON `subject` column beside `subject_key`** | Two representations of one fact, which is the objection ADR-0051 already used to refuse retaining a source's spelling. Columns carry the components once, and the index and the joins keep running on `subject_key` unchanged |
| **Make `parseServiceKey` return an error the fan-out surfaces, instead of dropping the row** | Inverts ADR-0207 for one site and still re-derives. It converts a silent stopped `Scan` into a loud stopped `Scan`, which is better and is not the fix — the parse is the defect, and a louder failure of it is still a failure of it |
| **Fix the parse-back on this ADR's own branch** | It touches a migration, `sqlc`-generated code under `internal/db`, the wire type, the span fold and two packages of callers. That is a production change with its own review and its own migration-number race with `origin/main`, buried under a documentation review |
| **Rule it inside ADR-0028, which owns the `tls-acceptance` `Scan`** | ADR-0028 rules a cadence and an aperture — weekly, over the open `Service` population, never a port tier. The subject key's wire form is neither, the rule reaches `http-identity` and the narrowing folds as well, and filing it there would hide a persistence contract inside a scheduling document |
| **Rule it inside ADR-0051 as a further clause** | ADR-0051 rules how a key is formed and why its normalisation may never move. This rules what a reader may do with a stored rendering, in one package, and it names a defect and a replacement route. Adding it there would also imply ADR-0051 had been wrong, which §4 exists to deny |
