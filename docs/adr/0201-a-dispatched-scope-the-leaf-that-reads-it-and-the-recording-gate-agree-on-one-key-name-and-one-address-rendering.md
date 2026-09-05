# ADR-0201: a dispatched scope, the leaf that reads it, and the recording gate agree on one key name and one address rendering

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1319 ADR gaps: internal/scan (2/3)](https://github.com/winniel123/verge-asm/issues/1319), gap 2
- **PR that deleted the comment:** [#1318](https://github.com/winniel123/verge-asm/pull/1318)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Sibling of, and not ruled by:** [ADR-0150](./0150-a-batch-scope-names-its-dimension-in-the-plural-and-a-one-address-fan-out-ships-a-one-element-list-never-a-scalar.md). That ADR rules the **cardinality** of a scope field — the dimension is named in the plural, and one address ships as a one-element list. This ADR rules the **name** of that field and the **spelling** of its members. The two are neighbours over one struct and neither contains the other
- **Rests on:** [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md), which rules that a subject key is the thing denoted and that its normalisation may never move. It rules the key. It does not rule the scope record that authorises a row about that key
- **Rests on:** [#773](https://github.com/winniel123/verge-asm/issues/773) and its re-gate, which exists because a compromised prober can name any subject. This ADR states the condition under which that gate admits an honest row

## Context

[`internal/scan/edgefanout.go:67`](../../internal/scan/edgefanout.go) and
[`internal/scan/edgefanout.go:34`](../../internal/scan/edgefanout.go) carried these two blocks until
#1318 deleted them:

```go
// edgeFanoutScope is the wire payload an edge-fanout job carries. The field name
// matches the leaf's own Scope (edgefanout.Scope) and the `addresses` key the
// recording-side scope gate reads, so the dispatched scope, the measured scope and the
// authorised denotation are one shape.
```

```go
// Addresses are rendered in their netip form, so the scope a job carries and the
// address a resulting observation names are the same spelling and the recording-side
// scope gate cannot reject a legitimate row over a rendering.
```

The sweep kept two compressed lines at the same sites, both uncited under §4.7:

```go
// The rendering must match what an observation names, or the recording gate drops the row.
```

```go
// The `addresses` key is read by the recording-side scope gate, so it must match the leaf's Scope.
```

**No ADR and no SPEC states the rule.** ADR-0150 rules the shape of one field's value. ADR-0051
rules the subject key. Neither says that three separately-declared Go types must present the same
JSON name, nor that two normalisation sites must be the same function.

### One decoder reads every leaf's recorded scope

[`internal/queue/scopegate.go:28`](../../internal/queue/scopegate.go) declares **one** `scopeShape`
and unmarshals it over the `recorded_scope` of every job kind:

```go
type scopeShape struct {
	Names     []string `json:"names"`
	Addresses []string `json:"addresses"`
	Services  []struct{ Address string `json:"address"` } `json:"services"`
	Targets   []struct{ Address string `json:"address"` } `json:"targets"`
}
```

That union is only safe because the four names have one meaning everywhere they appear. Nothing in
the type system holds them together.

### Three shapes per Scan, declared in three packages

The dispatched scope, the recorded scope and the leaf's own scope are **separate Go types** for four
of the five dispatching Scans. Only `edge-fanout` marshals one type for two of the three roles.

| Scan | Dispatched (`JobSpec.Scope`) | Recorded (`AttemptedScope`) | The leaf's decode target | Key the gate reads |
| --- | --- | --- | --- | --- |
| `dns` | `resolutionwalk.Scope` | `scan.scopeRecord` | `resolutionwalk.Scope` | `names` |
| `hot` | `connectoutcome.Scope` | `scan.hotScopeRecord` | `connectoutcome.Scope` | `addresses` |
| `cold` | `connectoutcome.Scope` | `scan.coldScopeRecord` | `connectoutcome.Scope` | `addresses` |
| `http-identity` | `httpexchange.Scope` | `scan.httpIdentityScopeRecord` | `httpexchange.Scope` | `targets[].address` |
| `tls-acceptance` | `tlsacceptance.Scope` | `scan.tlsAcceptanceScopeRecord` | `tlsacceptance.Scope` | `services[].address` |
| `edge-fanout` | `scan.edgeFanoutScope` | `scan.edgeFanoutScope` | `edgefanout.Scope` | `addresses` |

Ten distinct type declarations sit across three packages. A rename in any one of them compiles.

### The `edge-fanout` arm is the one that fails closed

`authorizedScope.admits` treats an absent set as *"gate nothing"* for every facet, because gating an
undenoted dimension would drop every legitimate line. The facet-less arm is the exception:

```go
case "":
	if o.Kind != edgefanout.Kind {
		return true
	}
	// This arm alone fails closed
	if a.addrs == nil {
		return false
	}
	_, ok := a.addrs[normAddr(o.Address)]
	return ok
```

So for `edge-fanout` a misspelled key is not a weakened gate. `a.addrs` is `nil`, the arm returns
`false`, and **every** observation from **every** `edge-fanout` job is dropped. The dispatch still
completes, the `Batch` still records the full scope, and the custody extension sees a fan-out of
zero. Nothing errors and nothing logs a cause beyond a per-row drop line.

### The rendering is a second, independent agreement

The key name gets the row to the right set. The spelling decides whether it is in it. Four sites
normalise, and they must be one function:

| Site | Code | What it produces |
| --- | --- | --- |
| The builder | [`internal/scan/edgefanout.go`](../../internal/scan/edgefanout.go) | `a.Unmap().String()` into the scope |
| The leaf's target parse | [`internal/measure/edgefanout/run.go`](../../internal/measure/edgefanout/run.go) | `netip.ParseAddr(strings.TrimSpace(addr))`, then `AddrPortFrom(ip.Unmap(), Port)` |
| The leaf's emit | [`internal/measure/edgefanout/emit.go`](../../internal/measure/edgefanout/emit.go) | `target.Addr().String()` on the already-unmapped address |
| The gate | `normAddr` in [`internal/queue/scopegate.go`](../../internal/queue/scopegate.go) | `netip.ParseAddr(strings.TrimSpace(s))`, then `.Unmap().String()` |

`::ffff:198.51.100.1` and `198.51.100.1` are one address and two texts. `Unmap` on one side alone
puts a real observation outside the authorised set. The leaf's own comment records the dependency in
the other direction — *"Trimmed the way internal/queue's scope gate trims, so the two sides agree on
a spelling"* — which is the same rule read from the leaf.

### A fourth reader is outside the measurement path entirely

[`cmd/web/driftfeed.go:240`](../../cmd/web/driftfeed.go) ranges `[]string{"names", "addresses",
"services"}` against the recorded scope to build a batch label. It never sees the producer, it
decodes by name alone, and a mismatch renders no label rather than failing.

## Decision

> **The scope a job dispatches, the scope the leaf decodes, the scope the `Batch` records, and the
> recording-side scope gate name one dimension with one JSON key, and render an address through one
> normalisation. Agreement is by name and by spelling, because no type connects the four sites. A
> disagreement drops a legitimate observation over a text rather than over a denotation, so it is a
> correctness fault and never a cosmetic one.**

### 1. One key name per dimension, across every Scan

`names` for a name list. `addresses` for a bare address list. `services` and `targets` for a list of
objects each carrying `address`. A new Scan that denotes addresses reuses one of those four names. It
does not add a fifth, because `scopeShape` is one union and a fifth name is a field the gate does not
read — which, for a facet-less kind, fails closed and drops everything.

### 2. The recorded scope is the gated one, and it is a different type from the dispatched one

`authorizedScope` is parsed from `job.AttemptedScope`, not from `JobSpec.Scope`. So the agreement
runs in two legs, and both must hold:

- **Dispatched against the leaf**, or the leaf decodes an empty scope and measures nothing.
- **Recorded against the gate**, or the gate drops what the leaf did measure.

The table above shows those are separate declarations in four of the six builders. A builder that
changes one leg changes only one.

### 3. One normalisation, and it is `netip`'s

An address in a scope, an address in an observation, and an address in the gate's set all pass
through `netip.ParseAddr`, `Unmap`, and `String`. The gate normalises both sides for exactly this
reason, so a producer that ships a different-but-equal spelling is still admitted. That tolerance is
the mechanism and not a licence to ship a second spelling: `normAddr` falls back to
`strings.TrimSpace` on an unparseable text, and two unparseable texts agree only by luck.

### 4. The rule binds the leaf's **output** too, not only its input

The gate compares the scope against the address an observation **names**. `edge-fanout` emits
`wire.Observation.Address`. `http-identity` emits an `Endpoint` key whose service leg the gate
recovers with `subjectAddrKey`. Both are renderings the leaf computes. A leaf that echoed its input
text instead of re-rendering it would satisfy this rule by accident and break it on the first
producer that shipped a mapped form.

### 5. What this rule does not reach

- **Which addresses a job may carry.** `Estate.MayProbe` decides that at dispatch (ADR-0019). This
  rule is about the encoding of the answer.
- **The cardinality of the field.** ADR-0150 rules that, and the two rules are independent: a
  correctly-named field can still hold a scalar.
- **The gate's fail-open default for a facet-bearing kind.** That is #773's own ruling and it stands.
- **`Offers`.** An offer is a separate record with its own rule (ADR-0025).

## Consequences

- **This ADR changes no Go code.** All six builders comply today. Three tests pin parts of it —
  [`internal/scan/edgefanout_test.go:113`](../../internal/scan/edgefanout_test.go) decodes the
  dispatched scope through a local `addresses` struct, and
  [`internal/queue/scopegate_test.go`](../../internal/queue/scopegate_test.go) feeds the gate literal
  JSON. No test spans a producer and the gate together.
- **A new leaf's checklist gains one line.** A leaf that denotes a dimension the gate cannot read is
  not a weaker gate. For a facet-less kind it is a total drop, and the failure is silent in every
  derived value.
- **Nothing enforces this.** The union in `scopegate.go` is decoupled from every producer by
  construction, which is what makes it able to read them all. A round-trip test — build a job, run
  the leaf against a scripted handshaker, pass the output through `parseAuthorizedScope(job's
  recorded scope).gate(...)`, and assert nothing is dropped — would enforce it for one Scan. **It is
  not written here and ships as its own ticket.**
- **`CONTEXT.md` gains nothing.** A JSON key name is a wire detail, not a domain term.
- **[ADR-0150](./0150-a-batch-scope-names-its-dimension-in-the-plural-and-a-one-address-fan-out-ships-a-one-element-list-never-a-scalar.md)
  is unchanged.** Its Context already observes that `edge-fanout` names its dimension `addresses`
  and that one decoder serves both leaves. This ADR rules what that observation implies.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Share one Go type across the dispatcher, the leaf and the gate** | The recorded scope is deliberately **not** the dispatched scope. `scan.scopeRecord` drops the resolver and the offers and adds `control_probe_population`. `scan.httpIdentityScopeRecord` drops the vantage class. The record is what licenses a silence, and it carries a different set of fields from the payload a prober needs. One type would force the two apart again through `omitempty` tags, which is the same coupling with the failure moved into a tag |
| **Have the gate read `JobSpec.Scope` instead of the recorded scope** | The gate exists because the prober is untrusted (#773). The dispatched scope is the text handed **to** that prober. The recorded scope is the instance's own statement of what it authorised, and it is the one a later reader audits. Reading the payload would gate the prober against a document the prober was given |
| **Make the gate accept any list-of-strings field it finds** | Turns a name agreement into a shape guess. `tcp_ports` is a list too, `control_probe_population` is a list of names that were never authorised as probe targets, and admitting by shape would let a scope's diagnostic field widen its own authorisation |
| **Compare address texts literally and forbid normalisation on the gate side** | The producer, the leaf and the store each render an address at a different moment. `normAddr` on both sides is what makes an honest row survive a legal re-spelling. Removing it would make the gate reject `::ffff:198.51.100.1` against `198.51.100.1`, which ADR-0051 says are one subject |
| **Let the `edge-fanout` arm fail open like the others** | It fails closed on purpose: an injected `edge-fanout` line feeds the custody veto an answer nothing measured ([#985](https://github.com/winniel123/verge-asm/issues/985)). Opening it to make a key mismatch survivable would trade a loud, total drop for a silent, partial forgery |
| **A `commentlint` or `go vet` check on the JSON tags** | The four names are correct at ten sites and would have to be listed somewhere for a checker to compare them against. That list is this ADR. A checker would restate it and then need updating on every new dimension, which is the ADR's job |
