# A `Name`'s key is the label sequence on the wire, and we fold only what the protocol folds

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#94 What is a `Name`'s natural key — case, the trailing dot, and A-label vs U-label](https://github.com/winniel123/verge-asm/issues/94)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends:** [ADR-0011](./0011-a-facet-is-six-parts.md) in one further detail
- **Discharges:** [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)'s named residue

## Context

[ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md) settled
what kind of object a subject key is — **the thing denoted, not the text that named it** — and settled
that its normalisation is **fixed rather than versioned**, because a `Break` is a sentence about two
values on one timeline and a re-key is a sentence about which row a timeline belongs to, which the
model has no object for. It deliberately left `Name`'s **function** undecided, and said so in its own
thinness section: *"`Endpoint`'s `Name` is the residue and it is named in the consequences: nothing
here decides it."*

Three axes were named and none was answered.

- **Case.** DNS matching is case-insensitive and the wire is case-**preserving**
  `[spec]` RFC 1035 §2.3.3, RFC 4343. Is `WWW.Example.Com` one subject with `www.example.com` or two?
- **The trailing dot.** `example.com` and `example.com.` denote one name on almost every surface, and
  a master file is the one artefact where the dot changes what a line means `[spec]` RFC 1035 §5.1.
- **A-label vs U-label.** `xn--bcher-kva.example` and `bücher.example` are one name under IDNA2008
  `[spec]` RFC 5890 §2.3.2.1. Which is stored, and does the answer survive a Unicode release?

It is live today, and on the worst path available. The operator's zone file read against this
project's own resolver is the model's **only two-source facet**
([ADR-0020](./0020-a-conflict-needs-two-enumerable-sources.md)), and it is the pair most likely to
disagree on all three axes at once: a master file is the one artefact in the system that
conventionally writes the trailing dot, that may be authored in mixed case, and whose relative
owner names mean nothing without the file's own `$ORIGIN`. Two spellings of one name are two `Name`
subjects, so one name holds two sets of timelines and a re-export or a library upgrade retires one
subject and enters another — a withdrawal and a membership message for an event in which nothing
moved. That is [#6](https://github.com/winniel123/verge-asm/issues/6)'s seam, one subject over from
where ADR-0051 closed it.

One limb of ADR-0051's argument does **not** obviously transfer, and it is the load-bearing one.
Its ruling turned on the observation that *on the measured paths there is no text at all* — an A
record is 32 bits of RDATA, an AAAA record 128, a certificate `iPAddress` SAN an OCTET STRING. A name
looks like the opposite case: it is written, printed, typed, and read aloud. If a name really is text
on the wire, the address argument is unavailable and the construction has to be built rather than
inherited.

## Decision

**A `Name`'s natural key is the label sequence — and we fold exactly the equivalence the protocol
supplies and refuse to supply any of our own.**

| Concern | Decision |
| --- | --- |
| What the key is | The **label sequence**: an ordered sequence of labels, each an **octet string** of 1–63 octets, terminated by the **root label**. Always absolute |
| **Case** | **ASCII-folded.** Octets `0x41`–`0x5A` map to `0x61`–`0x7A`, per octet. **No other octet is touched, ever** `[spec]` RFC 1035 §2.3.3, RFC 4034 §6.1, RFC 4343 |
| **The trailing dot** | **Neither stored nor stripped.** It is the presentation format's marker for *absolute*, consumed by the parse; the **root label it marks is in the key** |
| **A-label vs U-label** | **The A-label, and it is never decoded.** `xn--bcher-kva` is a label like any other and the key function does not know it is one |
| A **U-label presented as text** | **Refused, never interpreted.** It has two denotations — a raw-octet label and an A-label — and separating them is table-driven |
| Comparison | **Label-wise octet equality, in order.** No string comparison anywhere in the model |
| Subtree containment | **Label-wise suffix equality over the key.** Never a suffix test over text |
| Rendering | **One rendering:** labels joined by `.`, **no trailing dot**, computed on read, never stored, never compared. This is also exactly what SNI carries `[spec]` RFC 6066 §3 |
| A **U-label rendering** | **Not in v1.** ToUnicode consults Unicode tables, so it is the one rendering that is not a pure function of the key |
| Does it get an ADR-0011 canonicaliser? | **No.** ADR-0051's rule, unchanged |
| Is it a leaf in the `Derivation` vector? | **No leaf, no version, no `Break`** |
| What it may consult | **The one value being keyed, and nothing else** |
| The key function's **domain** | **A label sequence, never a text.** Turning a wire message, a master file or a typed box into one is **parsing a format** — and that parse is fixed and versionless too |
| The zone file and the resolver | **One key function, two parsers.** The trailing dot and `$ORIGIN` are load-bearing inside a master file and meaningless outside one |
| Names that cannot be keyed | **Not subjects.** A label over 63 octets, a wire form over 255, an empty label, a high-bit octet in a **text** form, and the root alone |
| A high-bit octet **from the wire** | **Keyed as those octets.** A label is an octet string `[spec]` RFC 2181 §11, and nothing there is ambiguous |
| `Endpoint` | **Inherits for free** — confirmed, not assumed. The **absent** `Name` is a distinguished variant of the key, never an empty name |
| Names inside a `dns-record` value | Held **in the key form** — CNAME targets, NS names, MX exchanges. The canonicaliser composes it and does not restate it |
| The received spelling | **Not retained** |
| `Name`'s lifecycle | **Unchanged.** ADR-0006 governs and nothing here touches it |
| Enforcing corpus | **ADR-0051's corpus, extended** — one artefact, one inverted gate, a second input shape |

## Rationale

### A name on the wire is octets too, and the limb transfers with a stronger warrant

This is the question the ticket said might not transfer, and the answer is that it transfers — but
not by analogy, and not because *a name is like an address*. It transfers because the premise that a
name is text on the wire is **false**.

A domain name in a DNS message is a sequence of labels, each a one-octet length field followed by
that many octets, terminated by the zero-length root label `[spec]` RFC 1035 §3.1. A label is an
**octet string**: RFC 2181 §11 is explicit that *any* binary string whatever can be a label, and the
protocol never interprets one as characters. There is no encoding declared anywhere in a DNS message,
because there is nothing to encode.

So the table ADR-0051 built for addresses can be built again, and it comes out the same way:

| Source | What arrives | Where a text would come from |
| --- | --- | --- |
| A resolver answer, owner name | A sequence of length-prefixed labels `[spec]` RFC 1035 §3.1 | Whatever printed it |
| A resolver answer, name-valued RDATA | The same, inside the RDATA `[spec]` RFC 1035 §3.3 | Whatever printed it |
| A certificate `dNSName` SAN | An `IA5String` in A-label form `[spec]` RFC 5280 §4.2.1.6, §7.2 | — |
| A CT log entry, via `crt.sh` | JSON text of the above | The log's own printer |
| The operator's typed `Seed` | Text | The operator |
| The operator's zone file | Text, in presentation format `[spec]` RFC 1035 §5.1 | The operator |

What a printer adds is exactly the three axes the ticket names. It picks a **case** — the wire
preserved whatever the authority stored, and preservation is not a claim. It inserts **separator
dots**, which are not on the wire at all; the wire has length prefixes. And for an internationalised
name it may print a **Unicode rendering**, which is a conversion no DNS message has ever carried.

So the construction is not *the address argument reused*, it is the address argument with a
**stronger** warrant. For addresses we had to observe that no source emits text. Here the
specification says outright that the thing on the wire is octets, and then goes further than IP ever
does: it defines its own **equivalence** over them.

And the denoted thing is unambiguous. A `Name` denotes a **node in the DNS name tree**, and every
label sequence names exactly one node — existing or not, which is a measurement and not a key
question, since [ADR-0006](./0006-subjects-leave-by-measurement.md) already makes existence
something our resolver measures. Two objections were put and both fail on inspection. *Split horizon
makes the denotation vantage-dependent* confuses the subject with its value: what a name resolves to
is `resolution`'s value, keyed per vantage precisely because two vantages measure different facts
(ADR-0020), and the node is the same node from both. *A CNAME makes two names one thing* is likewise
about content: `www.example.com` and its target are two nodes, and
`cname-target-name-error` already reads the second as a separate `Name`.

### Fold where the specification supplies the equivalence; refuse where supplying it would be ours

This is the general rule, and it is what makes two of the three axes fold and the third refuse.

ADR-0051's refusal test was **ambiguity**: `010.1.1.1` is `10.1.1.1` to a strict parser and `8.1.1.1`
to `inet_aton(3)`, so it has two denotations, so it does not denote, so it has no key. A session
reaching for that test here would refuse mixed case and the trailing dot too, on the ground that
they are *variance* and the key should have one form.

That is the wrong test applied to the wrong thing, and the distinction is the whole ruling:

> **Variance is not ambiguity.** `WWW.Example.Com` is not a string with two denotations — DNS
> defines the equivalence, so it denotes exactly one node, the same one `www.example.com` denotes.
> A form is refused where **we** would have to supply the equivalence, and folded where the
> specification already has.

That test is decidable from the documents, and it decides all three axes without a preference being
expressed anywhere:

| Axis | Who supplies the equivalence | Outcome |
| --- | --- | --- |
| Case | The protocol — RFC 1035 §2.3.3, and RFC 4034 §6.1 states the fold as an algorithm | **Fold** |
| The trailing dot | The presentation format — RFC 1035 §5.1 says what it means | **Parse**, and the root label is in the key |
| U-label → A-label | **Us**, via UTS-46's mapping tables and Unicode's properties | **Refuse** |

### Case: the fold is the ASCII fold, and that is exactly why it is versionless

`WWW.Example.Com` and `www.example.com` are **one subject**.

The warrant is not convenience. RFC 1035 §2.3.3 makes every comparison in the official protocol
case-insensitive; RFC 4343 clarifies that the insensitivity covers **the 26 ASCII letters and
nothing else**; and RFC 4034 §6.1 states the fold as an algorithm, because DNSSEC needs a canonical
form and this is it. We are not choosing a normalisation. We are reading one off three documents
that agree.

The other half — the wire is case-**preserving** — is what makes not folding actively dangerous
rather than merely untidy, and it is worth stating in the model's own terms: *the case on the wire
is not a property of the world.* Four producers move it, and none of them is the estate changing.

- An authority echoes whatever the zone's owner name was stored as, so an operator re-typing a
  record in their provider's console re-keys the subject.
- A master file may be authored in any case at all, and commonly is at the apex.
- `crt.sh` hands us its own rendering of a SAN, and its case is the log's, not the certificate's.
- **0x20 encoding** — randomising the case of a query name so the response must echo it, as an
  anti-spoofing measure — makes the case of an answer's owner name *deliberately random*. Under an
  unfolded key every query would mint a new `Name` subject. Our own resolver decides whether to use
  it, which is precisely the point: a key that a measurement-side option can move is
  [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md)'s test failed in the worst
  available place.

The implementation constraint is not incidental and it is a corpus row rather than a comment,
because the obvious code is wrong. The fold is **per octet**, over `0x41`–`0x5A` only. It is not a
string lowercase: Go's `strings.ToLower` decodes UTF-8 and folds far beyond ASCII, so `İ` (U+0130)
and the Kelvin sign fold into things RFC 4343 says are untouched, and a locale-aware fold would
make the key depend on where the binary is running. **The versionlessness is bought by it being
the ASCII fold and not a case fold** — 26 codepoints, fixed since 1987, with a document saying it
extends no further.

### The trailing dot is the parser's, and inside a master file it is load-bearing

`example.com` and `example.com.` are **one subject** — everywhere except the one place they are not,
and getting that exception right is most of the value here.

The dot is not part of a name. On the wire every name is absolute and terminates at the root label;
there is no representation of a relative name in a DNS message at all. The dot exists in the
**presentation format**, where RFC 1035 §5.1 gives it one job: *a domain name ending in a dot is
absolute; otherwise it is relative to the current origin.*

So the rule is neither *store the dot* nor *strip the dot*. It is:

> The dot is **consumed by the parse**, and the **root label it marks** is in the key. The key is
> absolute by construction, which is what makes the glossary's *fully-qualified domain name* a
> statement about the key rather than about a convention.

That reading is what keeps the two-source pair honest. Outside a master file there is no origin, so
a typed `example.com` can only be absolute and the dot is accepted and ignored. **Inside** one there
is, and `www` under `$ORIGIN example.com.` is `www.example.com.` while `www.example.com` — no dot —
is `www.example.com.example.com.`, which is a different node and a real zone-file bug operators
make. A decoder that "helpfully" appended a dot to every owner name would key both of those the
same way, silently, and the failure would surface as
`resolved-name-absent-from-zone` going quiet on a name that genuinely is missing.

Stripping instead is worse and it is the option that looks most reasonable. It reintroduces a
**string** as the compared object, which puts a string comparison back into the probing gate (below),
and it makes a relative owner name look complete when it is not.

### The A-label is the name; the U-label is a conversion the key function may not make

`xn--bcher-kva.example` is **stored, compared and rendered**. `bücher.example` is **refused**.

The first half needs almost no argument. IDNA2008 puts A-labels in the DNS protocol and U-labels in
the user interface `[spec]` RFC 5890 §2.3.2.1; RFC 5280 §7.2 requires the same of a `dNSName` SAN.
No DNS message has ever carried a U-label, so on every measured path the A-label **is** what arrives,
and it arrives as ordinary ASCII octets. The key function does not decode it, does not validate it,
and does not know it is an A-label — `xn--bcher-kva` is 13 octets in the LDH range, and treating it
as anything else would be the key function classifying.

The second half is ADR-0051's own rule applied, not a new one, and it holds on **two independent
grounds**.

**It is ambiguous, in exactly `010.1.1.1`'s way.** The text `bücher.example` has two denotations. Read
strictly as presentation format — which has no concept of Unicode and predates UTF-8 by a decade —
its first label is the five octets `62 C3 BC 63 68 65 72`, a perfectly legal binary label under
RFC 2181 §11. Read as a U-label it is `xn--bcher-kva`. Two parsers, two names, one string. And
choosing between them would be **our** choice, authored and revisable, which is the movable content
the versionless bargain exists to exclude.

**And the conversion is barred outright by ADR-0051's prohibition.** A key normalisation *may consult
only the single value being keyed — never reference data*. UTS-46 is nothing but reference data: an
IDNA mapping table derived from Unicode, NFC normalisation data, Bidi and ContextJ rules read off
Unicode properties. Those tables move on the Unicode Consortium's schedule, not ours, and the
profiles do not even agree with each other today — the four deviation characters (ß, ς, ZWJ, ZWNJ)
key differently under IDNA2003, UTS-46 transitional and UTS-46 non-transitional. A key function that
consulted them would have its content set by somebody else's release, which is the one thing
ADR-0051 makes impossible to price.

The refusal is **total over the class** rather than per-character, and the reason is the same
prohibition read once more: the test that would decide whether a particular U-label is ambiguous is
itself the deviation table. What is left is a test that is a pure function of the value — **does any
octet in this text have its high bit set** — and it is the only shape available.

The asymmetry that falls out is correct and is worth stating, because it looks like an
inconsistency: a high-bit octet arriving **from the wire** is keyed as those octets and makes a
perfectly good subject, while the identical octets arriving **as text** are refused. Nothing is
inconsistent. From the wire the octets *are* the label and there is no second reading; in text there
are two, and the model refuses to pick.

**The cost lands in one place and it is stated.** An operator with an internationalised estate
cannot type `bücher.example` into the `Seed` box or leave a raw U-label in an exported zone file.
The exposure is small — every provider export the project has looked at emits A-labels, resolvers
cannot emit anything else, and `dNSName` cannot hold one — so the surface is the typing box and the
answer is refusal copy naming the A-label as the route, in the shape [#85](https://github.com/winniel123/verge-asm/issues/85)
already established for an over-cap IPv6 declaration. A conversion **is** available as a future
shape, and it is legal only in one form: the box offers the A-label, the operator confirms it, and
what enters the model is a form the operator **declared** — the `Proposal` pattern, with the Unicode
version pinned and shown. That is deliberately not v1, because it is a surface nobody has drawn, and
it is recorded here so nobody later builds the illegal version by mistake.

**One rendering, and no U-label rendering in v1.** ADR-0051's rendering test is that a rendering is
free where it is a **pure function of the key**. Joining labels with dots passes. ToUnicode does not:
Punycode decoding is pure, but IDNA2008's ToUnicode validates against Bidi and ContextJ rules that
are Unicode-derived, so a U-label rendering is the one rendering in the product whose output can move
while the world does not. It also puts a **second spelling in front of the operator** that the
`Seed` box then refuses — which is precisely the failure ADR-0051 named when it rejected retaining a
source's spelling as provenance. So v1 renders the A-label, everywhere, and an internationalised
estate reads as `xn--` throughout. That is a real cost and it is the honest one.

The rendering carries **no trailing dot**, which makes it the same string SNI needs
`[spec]` RFC 6066 §3, the same string the operator types, and the same string every other tool
prints — one rendering in the whole product rather than two. Cost: a rendered name is not
paste-ready into a master file as an absolute name.

### The key function's domain is the label sequence, and the parse carries no version either

This is the structural finding, it follows from ADR-0051 rather than amending it, and without it the
whole ruling leaks.

ADR-0011 makes a **decoder** per `(facet, source)` and versions it separately from the canonicaliser.
The zone-file decoder is where `$ORIGIN` completion, the trailing dot and RFC 1035 §5.1's `\DDD` and
`\.` escapes live — and those decide **which name** a line is about. So if the name-producing half of
a decoder sat inside a versioned leaf, a decoder bump could move a subject's key. A version move
costs a `Break`, a `Break` cannot express a re-key, and ADR-0051's entire ruling rests on that
sentence not being expressible. The hazard is the same one ADR-0051 closed, arriving through the one
door it did not have to shut for addresses, because no address decoder has an `$ORIGIN`.

So the boundary is drawn where it does the work:

> The key function's **domain is a label sequence**, never a text. Turning a wire message, a master
> file or a typed box into a label sequence is **parsing a format into the value** — and it is
> **fixed and versionless on the same terms as the key function itself**: it may consult only what
> the format supplies, it may make **no choice**, and where the format admits two readings it
> **refuses** rather than choosing.

`$ORIGIN` is not reference data under that reading and never had to be excused as an exception: a
relative owner name is an **incomplete value**, in the way `1.1.1` is an incomplete dotted quad, and
the file's own origin directive is part of the value the format delivers. What the parse may not do
is anything ADR-0011's policing rule already forbids — *a decoder that helpfully normalises is an
unversioned canonicaliser wearing a parser's clothes*, whose key-side twin here is a decoder that
helpfully lowercases, helpfully appends a dot, or helpfully runs UTS-46.

The practical consequence: **there is one key function and one implementation, and three parsers
feeding it** — the wire reader, the master-file reader, and the `Seed` box. The parsers differ
because the formats differ; the function does not differ at all.

### The zone file and the resolver key by one function, which is the point of having it

The ticket asks it directly and the answer is **yes, one function** — with a corollary sharp enough to
be worth more than the answer.

ADR-0020's two rules are the model's only two-source pair. `resolved-name-absent-from-zone` asks
whether **any zone-file `dns-record` timeline for this `Name` holds a record**, which is a key
lookup, and `zone-declared-name-returns-name-error` is its mirror. Under two key functions — or one
function fed by a decoder that folds differently — a zone file authored in mixed case makes every
declared name a *different* `Name` from the one the resolver measured, and
`resolved-name-absent-from-zone` fires on the operator's entire estate on the first run. The failure
is loud rather than silent, which is the only good thing about it.

The corollary is the one a session will get wrong: **one key function does not mean the same text
keys the same way from both sources.** `www.example.com` in a master file with an origin is a
different node from `www.example.com` typed into the `Seed` box, and both are keyed correctly by one
function, because the difference was resolved by the parse before the function ever saw it. The rule
that keeps this straight is the one above — parsers differ, the function does not.

Two existing obligations now have a stated referent. ADR-0020's requirement that the zone-file batch
record **the zone rather than the registrable domain** is a `Name`, recorded in the key form, and
*is this name inside the recorded zone* is the label-suffix test below. And ADR-0011's
`(facet, source)` decoder pair for `dns-record` — *our resolver's wire answer against the operator's
zone file* — is exactly the pair this ruling makes comparable.

### Containment runs on the key, and the gate is where it pays

`Address`'s equivalent of this section was CIDR containment, and it was where refusing the mapped
fold would have closed the probing gate on the operator's own machine. `Name` has three containment
tests and they land in the same place.

- A **name-scope `Seed`** bounds the estate by a registrable domain.
- A `Seed`'s **exclusions** are *exact names, subtrees, or address scopes* — two of the three are
  name tests, and an exclusion decides whether a name is queried at all.
- A **`custody extension`**'s transitivity *stops where the resolution chain leaves the declared
  zone*, which is a subtree test whose answer opens or closes the probing gate on every address
  beneath it.

All three are **label-wise suffix equality over the key**: the candidate's label sequence ends with
the scope's label sequence, compared label by label, in the key form. That is the direct analogue of
ADR-0051's *families equal, first n bits equal*, and it is one rule with no branch.

Doing it over text is the classic defect and it fails in both directions at once. A suffix test over
the string `example.com` matches `evilexample.com`, which puts a stranger's name inside the
operator's boundary and opens the gate on it; a prefix test matches `example.com.evil.com`, which
does the same thing from the other end. Neither is possible label-wise, because `evilexample` and
`example` are different labels and no amount of shared text makes them one. This is the safety
property that pays for the whole ruling, exactly as the mapped fold paid for ADR-0051's.

### `Endpoint` inherits, and the absent `Name` is a variant rather than a null

`Endpoint` is `(Name, Service)` and ADR-0051 already rules that **a composed key holds the subject,
never its rendering** — so it should inherit. [#89](https://github.com/winniel123/verge-asm/issues/89)
confirmed rather than assumed this for `Service`, and it is confirmed rather than assumed here.

It inherits, and one limb needed checking that `Service`'s did not. `Endpoint`'s `Name` may be
**absent**, meaning *the default response to a client that names nothing* — the only measurement
mode available on an address-scope `Seed`. Absence is a **distinguished variant of the key**, not a
null and not an empty name, and the distinction is not pedantry: the empty text decodes to no label
sequence and is refused, the root label alone is refused, and neither may ever collide with the
nameless `Endpoint`. Two of those are one subject and one is a measurement mode.

The measurement side follows and is where it becomes visible: the `certificate` handshake sends
**SNI equal to the `Endpoint`'s name** and **no SNI at all** for the nameless one
([`measurement-offers.md`](../spec/measurement-offers.md) §1.6). SNI is therefore a **rendering of
the key**, computed on read like every other, and it is the same rendering — no trailing dot, per
RFC 6066 §3.

`Service` is untouched. It is `(Address, port, transport)` and holds no name.

### The facet consequence, and it is the same shape as ADR-0051's

`resolution`'s value is an address set and holds no names, so it is unaffected. `dns-record` holds
names **inside its values** — a CNAME target, an NS name, an MX exchange — and those are held in the
**key form**, for ADR-0051's reason restated one facet over: a CNAME target is the very thing that
admits the target `Name` that `cname-target-name-error` reads, and a set whose members were not the
names they admitted would be incoherent.

The protocol agrees, which is what keeps this out of authored territory: RFC 4034 §6.2's canonical RR
form replaces the uppercase US-ASCII letters in the DNS names embedded in the RDATA of the record
types it lists — NS, CNAME, SOA, MX, SRV and the rest. Without it, an NS RRset comparison moves on
case alone, and authorities do echo the case they were given.

So the `dns-record` canonicaliser **composes** the key form and does not restate it, a facet gains no
seventh part, and **ADR-0011 is amended in exactly one further detail** beside ADR-0051's — the same
amendment, one facet across.

### The corpus is ADR-0051's, extended, and it gains a second input shape

ADR-0051 created *a corpus of `(input, expected octets, claim in prose)` rows whose gate is
one-directional and inverted: a moved row fails the build, unconditionally.* `Name` **rides that
corpus** rather than getting one of its own, and the ground is that the corpus enforces a **rule**
rather than a datatype — *a subject key is the thing denoted, and its normalisation may never move* —
and there is one rule, one gate and one build failure.

What it gains is a second **input shape**, because a name's input is not always a single value: a
master-file row is `(owner field, current origin)`. The rows below are the v1 set, and each is a claim
in prose plus its expected label sequence, in ADR-0021's form.

| Input | Expected key | The claim |
| --- | --- | --- |
| `WWW.Example.COM.` | `www` `example` `com` root | ASCII case is folded |
| `İ.example` typed (U+0130) | **no key** | and only ASCII case — a Unicode-aware lowercase is the wrong function |
| `www.example.com` typed | `www` `example` `com` root | outside a master file a name is absolute with or without the dot |
| `www` under `$ORIGIN example.com.` | `www` `example` `com` root | a relative owner name is completed by the file's own origin |
| `www.example.com` under `$ORIGIN example.com.` | `www` `example` `com` `example` `com` root | **the dot is load-bearing in a master file and the key function never supplies it** |
| `www\.example.com.` in a master file | `www.example` `com` root | the format's escapes are the format's, and the parse honours them |
| `xn--bcher-kva.example.` | `xn--bcher-kva` `example` root | an A-label is a label like any other; the key never decodes it |
| `bücher.example` typed | **no key** | a name presented in U-label form has two denotations and is refused |
| a wire label containing `0xFF` | that octet, unchanged | a label is an octet string and the key never reads it as characters |
| a 64-octet label | **no key** | RFC 1035 §2.3.4 |
| a 256-octet wire form | **no key** | RFC 1035 §2.3.4 |
| the root alone | **no key** | it renders as the empty string and no source in the model produces one |

An unkeyable name is **not a subject**, exactly as ADR-0051 rules for an address: it is absent from
the `Batch`'s recorded scope, writes no value and no `Gap`, and the count is surfaced rather than
swallowed — a source that starts emitting unkeyable names is a source that has changed shape.

### Where this is thin, stated rather than smoothed

- **The A-label-only rendering is a product decision taken on model grounds.** Nobody has drawn what
  an internationalised estate looks like rendered entirely in `xn--` form, and nobody has measured
  how many operators have one. The argument that a U-label rendering is impure is sound; the
  judgement that the cost is acceptable is a judgement.
- **The SNI residue is real and unmeasured.** Keying folded means we can only ever *send* the folded
  form, so a server doing a case-**sensitive** SNI match against a mixed-case virtual host would not
  match and its chain would read `TLSRefused`. RFC 6066 §3 requires case-insensitive comparison and
  every mainstream server does it, so this is argued from documents rather than exercised — and the
  alternative, retaining the received case to send it, is ADR-0051's rejected provenance-spelling
  under another name.
- **The refusal of U-labels is total because the discriminating test is reference data**, which is
  correct and is also blunter than the hazard: for the overwhelming majority of U-labels every
  profile agrees, and those are refused alongside the four that do not. The bluntness is the price of
  a value-only test, and it is paid on the operator's typing box rather than on any measured path.
- **The root-alone refusal is the least exercised row in the corpus.** It is refused because it
  renders as the empty string and nothing in the model produces one, which is an argument from the
  model's own shape rather than from a document.
- **Whether a wildcard SAN admits a `Name` is untouched.** `*.example.com.` keys cleanly — `*` is
  octet `0x2A` and the key function does not read it — but whether a certificate's wildcard SAN
  *admits a subject at all* is an admission question this ruling deliberately does not answer.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) changes in four places.** `Name` states the key's form, the
  fold, the dot, the A-label, the rendering and the refusals; `Seed` states that a name scope and its
  name exclusions run on the key by label-wise suffix; `Custody extension` states that *leaves the
  declared zone* is that same test; and `Endpoint` states that the absent `Name` is a variant of the
  key rather than a null. **No term is added and no term changes meaning.**
- **ADR-0051's named residue is discharged.** Its rule bound `Name` and its function is now decided.
  ADR-0051 is **confirmed rather than amended** — nothing in it moves, and the one clause this ruling
  adds (the key function's domain, and the parse carrying no version) **follows from** its own
  argument rather than extending it.
- **The `Derivation` vector gains nothing**, and this is worth stating twice because a session will
  look: no `name-key` leaf, no version, no `Break` cause. ADR-0021's five prober leaves stay five.
  The one thing that *is* constrained is existing: the name-producing half of a `dns-record` decoder
  carries no version, even though the decoder does.
- **A facet is still six parts.** ADR-0011 is amended in one further detail — `dns-record`'s
  name-valued RDATA is held in the key form — beside ADR-0051's amendment to `resolution`'s address
  set. Same shape, one facet across.
- **ADR-0020 is confirmed and its two rules become well-defined.** Both read a `Name` key across two
  sources, and both were undefined until this ruling. Its zone-recording obligation now names a
  `Name` in key form, and *inside the recorded zone* is a label-suffix test.
- **The probing gate runs on the key, in three places** — a name-scope `Seed`, a `Seed`'s name and
  subtree exclusions, and the `custody extension`'s zone boundary. None of them may ever compare
  text.
- **ADR-0051's corpus gains twelve rows and a second input shape.** One artefact, one inverted gate;
  a master-file row's input is `(owner field, current origin)`.
- ~~**The `Seeds` screen owes refusal copy for a U-label**, naming the A-label as the route, in the
  shape #85 established for an over-cap IPv6 declaration.~~ **Discharged by
  [ADR-0052](./0052-a-declaration-refusal-names-a-route-and-never-takes-it.md) and
  [#123](https://github.com/winniel123/verge-asm/issues/123)** — the copy is written, and the *shape*
  three ADRs asserted without stating is now stated: a refusal names a route and never takes it. One
  rider that was not obvious from here: the refusal **may not compute the A-label**, because printing
  *did you mean `xn--…`* runs the UTS-46 conversion this ADR bars and renders its output as advice.
  The route named is where to obtain the A-label, never the A-label. The legal future conversion —
  the box offers, the operator declares — is named so nobody builds the illegal one.
- **`Name`'s lifecycle is unchanged and no subject is added.** ADR-0006 governs departure and
  nothing here touches it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Key on the text a source handed us** — the option with no machinery | There is no text: a name on the wire is length-prefixed octet labels (RFC 1035 §3.1, RFC 2181 §11), and the case, the dots and any Unicode form are added by whatever printed it. Two spellings become two `Name` subjects, so the model's only two-source pair disagrees on all three axes at once and `resolved-name-absent-from-zone` fires on the whole estate |
| **A versioned key canonicaliser for `Name`** — the uniform-looking option | ADR-0051's ruling, not reopened: a `Break` is a sentence about two values on one timeline, a re-key is a sentence about which row a timeline belongs to, and the model has no object for the second |
| **Refuse mixed case and the trailing dot, as ADR-0051 refuses `010.1.1.1`** | Applies the refusal test to variance rather than to ambiguity. `WWW.Example.Com` has **one** denotation — DNS supplies the equivalence — so there is nothing for us to choose and nothing to refuse |
| **Fold case with the platform's lowercase** | Go's `strings.ToLower` decodes UTF-8 and folds far past ASCII, so `İ` and the Kelvin sign move; a locale-aware fold makes the key depend on where the binary runs. RFC 4343 says the insensitivity is exactly 26 letters, and the fold's versionlessness is bought by being *that* fold |
| **Strip the trailing dot and compare strings** | The right answer reached with the wrong object. It puts a string comparison back into the probing gate, makes `evilexample.com` a suffix match for `example.com`, and makes a relative master-file owner name look complete when it is not |
| **Append a trailing dot to every owner name in the decoder** | Keys `www` and `www.example.com` identically under `$ORIGIN example.com.`, which are different nodes. The dot is *consumed by the parse*; it is never supplied by us |
| **Store the U-label and render the A-label** — the human-first option | The U-label appears on no measured path, so the key would be produced by a table-driven conversion we author. IDNA2003, UTS-46 transitional and UTS-46 non-transitional disagree on ß, ς, ZWJ and ZWNJ, so a U-label has more than one A-label denotation |
| **Convert U-labels to A-labels at the boundary and store the A-label** — the tempting middle | Barred by ADR-0051's own prohibition: the conversion consults Unicode's mapping, normalisation, Bidi and ContextJ data, so the key function's content would be set by somebody else's release schedule — the one thing versionlessness cannot absorb |
| **Refuse only the ambiguous U-labels** | The test that identifies them **is** the deviation table, so the carve-out reintroduces exactly the reference data it was carving around. What is left that is a pure function of the value is *does any octet have its high bit set* |
| **Refuse high-bit octets from the wire too, for symmetry** | From the wire the octets *are* the label and there is no second reading (RFC 2181 §11). Refusing them would discard real subjects to make a text-boundary rule look tidy |
| **Render the U-label beside the A-label** | ToUnicode validates against Unicode-derived rules, so it is the one rendering whose output can move while the world does not — and it puts a second spelling in front of the operator that the `Seed` box then refuses, which is ADR-0051's rejected provenance-spelling arriving through a screen |
| **Render with a trailing dot** | Correct and it is a second string: SNI must not carry one (RFC 6066 §3) and the operator does not type one, so the product would hold two renderings where one does |
| **Put the trailing dot and `$ORIGIN` inside the versioned decoder** | A decoder bump could then move a subject's key, and a version move costs a `Break`, which cannot express a re-key. The hazard ADR-0051 closed, arriving through the one door addresses did not have |
| **Give `Name` its own key corpus** | The corpus enforces a rule rather than a datatype, and there is one rule, one inverted gate and one build failure. It gains an input shape, not a sibling |
| **Give `Endpoint`'s absent `Name` an empty-name encoding** | An empty text decodes to no label sequence and is refused; the nameless `Endpoint` is a measurement mode. Collapsing them puts a refused input and a real mode in one cell |
