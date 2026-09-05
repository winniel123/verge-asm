# ADR-0143: an `Rcode` is a closed union, and every code the leaf does not discriminate on folds to `OTHER`

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1160 ADR gaps: internal/measure/resolutionwalk](https://github.com/winniel123/verge-asm/issues/1160), gap 1
- **Sweep PR that deleted the comment:** [#1159](https://github.com/winniel123/verge-asm/pull/1159)
- **Not bound by:** [ADR-0015](./0015-the-value-space-is-the-commitment.md), whose commitment is a
  **facet's** value space. `Rcode` is a leaf-internal discriminator that reaches no emitted
  timeline, so widening or narrowing it prices at nothing
- **Rests on:** [ADR-0011](./0011-a-facet-is-six-parts.md) (what a facet is, and therefore what
  `Rcode` is not)

## Context

`internal/measure/resolutionwalk/leaf.go:23-30` declares `Rcode` as a `string` type with five
named constants — `NOERROR`, `NXDOMAIN`, `FORMERR`, `REFUSED` and `SERVFAIL`.

`internal/measure/resolutionwalk/netpeer.go:327-342` fills it. `rcodeName` maps those five wire
codes onto those five constants, and its default branch returns `Rcode(rc.String())`.

**That default returns two shapes, and both are off-model.** `golang.org/x/net@v0.58.0`'s
`dns/dnsmessage/message.go:155-170` holds a name table for the six codes the library defines, and
`RCode.String` falls back to a decimal numeral for everything else. So wire code 4 becomes the Go
identifier `RCodeNotImplemented` and wire code 9 becomes `"9"`. One `Rcode` value space carries
constants, a foreign library's Go identifier, and numerals.

**The domain is exactly sixteen values.** `parseResponse` reads `hdr.RCode` straight from
`dnsmessage.Parser.Start`, which is `RCode(h.bits & 0xF)` (`message.go:493`). No EDNS extended
rcode is assembled. Five of those sixteen map. **Eleven fall through**, and each one of the eleven
mints a distinct `Rcode` value that nothing in the package can act on.

**Nothing reads an `Rcode` except by equality against one of the five.** Every read is at
`leaf.go:125`, `:128`, `:181`, `:206` and `:243`, and every one of them is `==` or `!=` against
`NXDOMAIN`, `NOERROR`, `FORMERR`, `SERVFAIL` or `REFUSED`. The eleven fall-through values are
therefore already equivalent to one another at every site that consults them. The type simply does
not say so.

**`Rcode` never leaves the package.** `Result` carries `Resolution`, `Delegation` and `Records`
(`leaf.go:91-98`). `Rcode` sits on `Msg`, which is the peer's answer to one query, and no field of
it is serialised. That is what makes this a free ruling rather than a `Break`.

The rule was written down once, in a declaration comment on `Rcode`, and the §8 sweep deleted it as
uncited. The rule it stated was true of the intent and **false of the code**: it said the leaf
"never [reads] a numeric default", while `rcodeName`'s default produced exactly that. This ADR
states the rule and moves the code to it.

## Decision

> **`Rcode` is a closed union of six values: the five response codes the leaf discriminates on,
> plus `OTHER`. Every wire code outside the five folds to `OTHER`. No wire number, and no foreign
> library's rendering of one, ever becomes an `Rcode`.**

### 1. The value space is the five discriminands plus one sentinel

`NOERROR`, `NXDOMAIN`, `FORMERR`, `REFUSED`, `SERVFAIL`, `OTHER`. Six values, and the set is
closed. A reader enumerating what an `Rcode` can hold reads the `const` block and is done.

The five are not a list of codes that matter in DNS. They are exactly the codes `leaf.go`
**branches on**. That is the membership rule, and it is why the union is allowed to be this small:
a code the leaf does not branch on is, to this leaf, indistinguishable from any other such code.

### 2. `OTHER` is a domain value, not an error and not an absence

`OTHER` means *the authority answered, with a response code this leaf draws no distinction on*. It
is a real answer from a reached peer. Three things it deliberately is not:

- It is **not unreachable.** `Msg.Unreachable` carries that, and it is a different fact.
- It is **not a `Gap`.** A `Gap` is coverage the system could not obtain
  ([ADR-0066](./0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md)).
  Coverage was obtained here. The answer is simply not one of the five the leaf reads.
- It is **not `Undiscriminated`.** That word is taken, it means an answer that licensed no value,
  and ADR-0066 refused it as a `resolution` variant. Reusing it here would name a fourth thing with
  a term the estate already spends on two.

Under §1's reads, `OTHER` behaves as a non-`NXDOMAIN`, non-`NOERROR`, non-`FORMERR`,
non-`SERVFAIL`, non-`REFUSED` answer at every one of the five sites, which is the behaviour the
eleven fall-through values had. **No verdict moves.**

### 3. The fold happens once, at the wire boundary

`rcodeName` is the only place a `dnsmessage.RCode` becomes an `Rcode`, so it is the only place the
union can be breached. Its default branch returns `OTHER`. After that call, no code in the package
needs to consider a numeral, and no code needs to know that `x/net` has a name table.

### 4. Widening later is cheap, and the condition is named

Should the leaf ever need to branch on a sixth code — `NOTIMP` and `NOTAUTH` are the plausible
candidates — the change is: add the constant, add the `case`, add the branch. **`Rcode` reaches no
emitted timeline, so it is not a facet value space and ADR-0015 does not price the widening.**
There is no `Break`, no `revealed` message and no golden-corpus row.

The rule is: **a constant is added at the moment a branch needs it, and not before.** A constant
with no branch is a value space that promises a distinction the leaf does not make.

## Consequences

- **`leaf.go` gains one constant and `netpeer.go` loses one conversion.** That is the whole code
  change.
- **No golden-corpus row moves and no NDJSON byte changes.** `Rcode` is not emitted, and §2 shows
  every verdict is preserved.
- **The scripted peers are unaffected.** `internal/measure/resolutionwalk/corpus/script.go` and
  `internal/measure/wildcarddiscrim/corpus/script.go` write named constants only, so they were
  already inside the union this ADR closes.
- **A test now pins the default branch.** The old default was untested, which is how it stayed
  wrong while a comment asserted it was right.
- **Diagnosing a rare rcode gets harder.** An operator debugging a `NOTIMP` responder cannot read
  the code out of an `Rcode` any more. This is a real cost and it is small: nothing printed the
  value, `Rcode` is not in `Result`, and the recovery is a packet capture, which such an
  investigation needs anyway.
- **The deleted comment does not come back.** The rule now has a document, so the declaration
  position stays empty under the comment policy.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Leave the type open — accept the library's string as-is**, which is the status quo | It is not one value space but three: five constants, one foreign Go identifier (`RCodeNotImplemented`, from `x/net`'s `rCodeNames` table), and decimal numerals. A reader cannot enumerate what an `Rcode` holds from its declaration, and the eleven fall-through values are distinct from one another while being indistinguishable to every read site. It also makes an upstream data table part of our value space: `x/net` may name a seventh code in any release, and our model would change with it. **The open type buys a distinction no caller makes** |
| **Full RFC 6895 normalisation** — a named constant for every assigned rcode | It mints eleven values that nothing branches on, so it pays the maintenance of a registry mirror to preserve distinctions the leaf discards at every read. It also inherits the registry's problem: 11-15 are unassigned, 16 and above need the EDNS extended rcode this parser never assembles, and IANA can assign a new code, so the space is neither closed nor stable. **A value space is a commitment to make a distinction**, and the leaf makes five |
| **Keep the numeral, as `Rcode(strconv.Itoa(int(rc)))`** | It fixes only the inconsistency between the two shapes, not the openness. The space is still unbounded and still holds values no branch reads. It preserves a debugging affordance nothing consumes |
| **Fold to `SERVFAIL`**, reusing an existing constant | It is a lie about what the authority said, and it is a consequential one: `leaf.go:206` drops an answerless `SERVFAIL` as coverage-not-obtained and `:243` reads it as *does not serve*. An honest `NOTIMP` would be silently converted into a verdict about the nameserver |
| **Return an `error` from `rcodeName` for an unmapped code** | The peer answered. An unmapped rcode is a fact about the answer, not a failure of the exchange, and turning it into an error would abort a walk on a response the leaf is entitled to ignore |
| **A `bool` beside the five**, e.g. `Msg.RcodeKnown` | Two fields that must be read together to know one thing, with an unrepresentable state (`NOERROR` plus not-known) the type cannot forbid. A sixth constant says the same thing in one field |
