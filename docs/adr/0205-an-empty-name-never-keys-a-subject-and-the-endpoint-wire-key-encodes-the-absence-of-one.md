# ADR-0205: an empty name never keys a subject, and the `Endpoint` wire key encodes the absence of one

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1319 ADR gaps: internal/scan (2/3)](https://github.com/winniel123/verge-asm/issues/1319), gap 6
- **PR that deleted the comment:** [#1318](https://github.com/winniel123/verge-asm/pull/1318)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0011](./0011-a-facet-is-six-parts.md), which makes an `Endpoint`'s `Name` optional and gives a nameless `Endpoint` one leg. It rules the **domain** shape and never the encoding
- **Bounded by:** [ADR-0055](./0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md), which rules that a `Name` key is a label sequence, that an empty text decodes to no label sequence and is **refused**, and that `Endpoint`'s absent `Name` is a **distinguished variant of the key, never an empty name**. This ADR does not touch that rule. It states what the wire may carry underneath it, and §4 answers the Alternatives row that appears to forbid it
- **Bounded by:** [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md), which rules that a composed key holds the subject and never a rendering of it. The `Endpoint` key holds two subjects, and this ADR rules only how their absence is spelled

## Context

[`internal/scan/httpidentity.go:63`](../../internal/scan/httpidentity.go) carried this text until
#1318 deleted it:

```go
			Name:    "", // the nameless Endpoint — the reached Service's own HTTP identity
```

The sweep kept a compressed line at the same site, uncited under §4.7:

```go
			Name:    "", // the distinguished nameless Endpoint: an identity no Name cited
```

### What the domain already rules

`CONTEXT.md`'s `Endpoint` entry:

> The `Name` may be **absent**, meaning *the default response to a client that names nothing*. …
> Absence is a **distinguished variant** of the key and never an empty name. An empty text is refused
> and so is the root alone, and neither may collide with the nameless endpoint.

ADR-0055 states the same rule from the `Name` side and adds the reason: *"the empty text decodes to no
label sequence and is refused, the root label alone is refused, and neither may ever collide with the
nameless `Endpoint`. Two of those are one subject and one is a measurement mode."*

**Neither source rules the wire.** Neither says what byte sequence carries *absent* between the
dispatcher, the prober and the recording side.

### What the wire carries

The empty string, in two producers and one leaf helper.

| Site | Code | What it means |
| --- | --- | --- |
| [`internal/scan/httpidentity.go`](../../internal/scan/httpidentity.go) | `httpexchange.Target{Name: ""}` | The dispatched target for a reached `Service` |
| [`internal/measure/httpexchange/exchange.go`](../../internal/measure/httpexchange/exchange.go) | `EndpointKey(name, serviceKey) = name + "@" + serviceKey` | The `Endpoint` subject key |
| [`internal/measure/connectoutcome/certificate.go`](../../internal/measure/connectoutcome/certificate.go) | `EndpointKey(serverName, target, transport) = serverName + "@" + ServiceKey(...)` | The same key, from the `certificate` handshake |
| [`internal/measure/connectoutcome/certificate.go`](../../internal/measure/connectoutcome/certificate.go) | `Scope.endpointNames()` returns `[]string{""}` when `Names` is empty | The empty string as a **member of a name list** |

`endpointNames` is the sharpest of the four. The empty string is not a zero value that leaked. It is
an explicit list element, and it drives two things at once: `h.Handshake(ctx, target, "")` sends **no
SNI**, which is `measurement-offers.md` §1.6's rule for the nameless endpoint, and the same value
becomes the key's `Name` leg.

**Every `Endpoint` key the product writes today is the leading-`@` form.** No production builder sets
`connectoutcome.Scope.Names` — `internal/scan/hot.go` and `internal/scan/cold.go` build no name list,
and only a corpus fixture does. `BuildHTTPIdentityJobs` writes `Name: ""` unconditionally. So the
leading-`@` variant is not an edge case in this code. It is the whole population.

### What the readers do with it

Three decoders, all splitting at the first `@`, none of them in `internal/measure`:

| Reader | Site | Behaviour on a leading `@` |
| --- | --- | --- |
| The subject page | [`cmd/web/subjects.go`](../../cmd/web/subjects.go), `splitEndpointKey` | Returns `name = ""`, and `endpointPage` then sets `Nameless: name == ""` |
| The signals fold | [`cmd/web/signals.go`](../../cmd/web/signals.go), `splitEndpointName` | Returns `name = ""` and keeps the service leg |
| The #773 re-gate | [`internal/queue/scopegate.go`](../../internal/queue/scopegate.go), `subjectAddrKey` | Drops everything up to the first `@` and normalises the rest as an address |

The round trip is closed at the console: the encoder writes *absent*, and `endpointPage` renders the
nameless mode from it. Nothing anywhere reads the leading `@` as a name that happens to be empty.

### Why this looked like a contradiction

ADR-0055's Alternatives table carries this row:

> **Give `Endpoint`'s absent `Name` an empty-name encoding** — An empty text decodes to no label
> sequence and is refused; the nameless `Endpoint` is a measurement mode. Collapsing them puts a
> refused input and a real mode in one cell.

Read alone, that row appears to forbid exactly what `httpexchange.Target{Name: ""}` does. It does
not, and §4 states why. The row is about the **key**, and the wire field is not the key.

## Decision

> **An empty name never keys a subject. A `Name` subject is a label sequence, and an empty text is
> not one, so it is refused — `CONTEXT.md` and ADR-0055 are unchanged. The `Endpoint` key is not a
> `Name`. It is a composite wire encoding over two legs, and its leading-`@` form is the token for
> *this `Endpoint` has no name*. The encoding is a statement that the name is absent, never a name
> that happens to be empty. Both rules hold, and they govern different things.**

### 1. Two objects a reader must keep apart

- A **`Name` subject key.** ADR-0055 rules it: a label sequence, never a text. An empty text and the
  root alone are refused, and neither is a subject.
- An **`Endpoint` key.** ADR-0011 rules it: `(Name, Service)`. It is a composite, and its `Name` leg
  may be **absent**. Absence is a distinguished variant.

The rules do not conflict, because they are about two objects. The first says what may be a `Name`.
The second says an `Endpoint` need not have one.

### 2. On the wire, `""` is the absence token and never a name

`httpexchange.Target.Name` and `connectoutcome.Scope.endpointNames()` carry the empty string. That
value denotes *absent*. It is never handed to anything that treats it as a `Name`:

- The SNI path reads it as **send no SNI**, which `measurement-offers.md` §1.6 already rules.
- The key function reads it as **the leading `@` variant**, which is what this ADR licenses.
- No probing gate, no `Seed` validation, and no timeline key ever receives it as a `Name`.

An empty string reaching a site that expects a `Name` is a bug at that site, and this rule does not
soften it.

### 3. The soundness test a reader applies

**If the token decodes back to *absent* without ambiguity, the encoding is sound.** Three conditions
carry that, and all three hold here:

1. **The separator is outside both legs.** The `Service` leg is `<addr>:<port>/<transport>` and holds
   no `@`. A `Name` holds no `@` on every path that reaches a key today. `internal/seed`'s `isLDH`
   allows only `a`–`z`, `0`–`9`, `-` and `.` at the operator's declaration, and
   `internal/custody/fanout.go`'s `isLDHDomain` applies the same allowlist to a SAN before the fan-out
   reduction reads it.
2. **A real `Name` is never empty.** ADR-0055 refuses the empty text, so an empty leading segment can
   only mean *absent*. There is no second reading to choose between.
3. **Every decoder splits at the same place.** All three split at the **first** `@` and return the
   prefix as the name. One splitting at the last `@` would decode differently, so the rule binds the
   decoders and not only the encoder.

### 4. This is not the encoding ADR-0055 rejected

ADR-0055's rejected row is about the **key**: giving `Endpoint`'s absent `Name` an *empty-name*
encoding would put the refused empty text and the real measurement mode *"in one cell"* — one value
denoting two things. That is exactly what this code does not do.

The **key** the product writes for a nameless endpoint is `@198.51.100.1:443/tcp`. That is the
distinguished variant ADR-0055 requires, and it collides with nothing, because the refused inputs —
the empty text and the root alone — never produce a key at all. The empty string exists only in a Go
struct field and a JSON value **before** the key function runs. A transport field carrying the absence
token is not a cell in the key space.

The test that separates the two: **can a refused input and the nameless mode ever produce the same
key?** They cannot, because a refused input produces no key.

### 5. What this rule does not reach

- **Whether an `Endpoint` may have a name.** ADR-0011 rules it may. The measurement path that would
  supply one — `connectoutcome.Scope.Names` — exists and is unused, and Consequences records that.
- **The SNI a named handshake sends.** `measurement-offers.md` §1.6 and ADR-0055 rule it: SNI is a
  rendering of the `Name` key.
- **The separator character.** `@` is what both key functions use. This ADR takes it as given and
  rules only what the leading form means.
- **Any other subject's key.** `Service`, `Address` and `Name` are untouched.

## Consequences

- **This ADR changes no Go code.** It also changes no `CONTEXT.md` sentence and no ADR-0055 limb.
  Both keep their present meaning, and this ADR states the third thing that neither said.
- **[ADR-0055](./0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md)
  gains one pointer, recorded in this ticket's manifest.** Its Alternatives row reads, alone and in
  the present tense, as a prohibition on the encoding this code uses. A reader who finds
  `httpexchange.Target{Name: ""}` and that row and nothing else would conclude the code violates it.
  The row's reasoning is unchanged and the pointer only names where the wire question is answered.
- **`CONTEXT.md` gains nothing.** The `Endpoint` entry already carries the domain rule in the exact
  words §1 rests on, and a wire token is not a domain term.
- **Condition 1 of §3 is an assumption for a name the operator never typed, and it ships as its own
  ticket if it is worth closing.** `isLDH` and `isLDHDomain` guard the `Seed` boundary and the custody
  fan-out. A DNS label is an octet string, and `@` is 0x40, so a name from a resolver answer or from a
  CT entry is not refused by anything named above before it could reach a key. No such name reaches a
  key today, because `Scope.Names` is never populated in production. The condition becomes live on the
  day it is.
- **The named branch of the `Endpoint` key is production-dead and corpus-covered.**
  `Scope.endpointNames()` returns a real list only when a builder sets `Names`, and none does. So
  every `certificate` and `http-identity` observation the product writes carries a leading-`@`
  subject. A later session adding named endpoints should read §3 first, because conditions 1 and 3
  start doing work on that day.
- **Nothing enforces §3.** No test asserts that the three decoders agree, and no test round-trips
  `EndpointKey("")` through them. A table test over the two encoders and the three decoders would
  close it and is not written here.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Read `CONTEXT.md` as forbidding the empty wire string, and encode absence as a sentinel — `-@…`, `*@…`, or `(none)@…`** | Puts a text in the `Name` leg that a real name could also hold, which is the collision `CONTEXT.md` forbids in its own sentence — *"neither may collide with the nameless endpoint"*. `-` is legal in a label under `isLDH`. It also breaks the round trip: `splitEndpointKey` would return `"-"` and `endpointPage`'s `Nameless: name == ""` would render a named endpoint whose name is a hyphen |
| **Read the code as violating ADR-0055 and change the key to omit the separator — `198.51.100.1:443/tcp`** | Makes the nameless `Endpoint` key **identical** to its `Service` key. Two different subjects with one key is the arbitration ADR-0011 refuses, and the re-gate's `subjectAddrKey` — which strips everything before the first `@` — could no longer tell an `Endpoint` subject from a `Service` subject |
| **Give the absence its own field — `Target{Name string; Nameless bool}`** | Two fields encoding one fact, with no constraint holding them consistent. `{Name: "www.example.com", Nameless: true}` is representable and means nothing, and every decoder would have to decide which field wins. The single-token encoding has no such state |
| **Change `endpointNames()` to return an empty list rather than `[]string{""}`** | Deletes the measurement. The loop over `endpointNames()` is what performs the no-SNI handshake, so an empty list would produce no `certificate` observation at all for a reached `Service` — and ADR-0163 rules that an absent certificate row is a fan-out of zero rather than a pending one, so the silence would be recorded as a measured value |
| **Amend `CONTEXT.md`'s `Endpoint` entry to mention the wire form** | The glossary entry rules the domain, and it already rules it correctly and completely. Adding a JSON field and a separator character to a ubiquitous-language entry would put an encoding inside the vocabulary, which is what an ADR is for |
| **Rule that the wire may carry an empty name because "the key is not the wire", and stop there** | True and not enough. Without §3's decode test, the next composite key could pick a separator that appears in a leg, or a fourth decoder could split at the last `@`, and the encoding would stop being sound with no rule broken |
