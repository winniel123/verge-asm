# ADR-0060: A wildcard SAN is a pattern over names and admits none of them

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#99 Does a certificate's wildcard SAN admit a `Name`, and what is the subject if it does?](https://github.com/winniel123/verge-asm/issues/99)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Discharges:** [ADR-0055](./0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md)'s named residue

## Context

[ADR-0027](./0027-a-source-may-admit-without-observing.md) put certificate transparency on the
admission path and nowhere else: it observes no facet, holds no timeline, has no decoder, and is a
`Source` solely because *"`authority: inferred` is exactly the property it exercises over the
`Name`s a SAN carries"*. Everything CT contributes to the model is names read off SAN lists.

[ADR-0055](./0055-a-names-key-is-the-label-sequence-and-we-fold-only-what-the-protocol-folds.md)
then settled what a `Name`'s key is, and found this at the bottom of its own thinness section:

> **Whether a wildcard SAN admits a `Name` is untouched.** `*.example.com.` keys cleanly — `*` is
> octet `0x2A` and the key function does not read it — but whether a certificate's wildcard SAN
> *admits a subject at all* is a question about `authority` and `Citation`, not about
> normalisation, and ruling it here would be by-catch. `Shadowed` decides what happens to names
> **beneath** a wildcard; nothing decides the wildcard name itself.

That last sentence is the gap, and it is exact. The model has ruled twice on the region around this
name and never on the name. [ADR-0006](./0006-subjects-leave-by-measurement.md) rules that
*"admission under a wildcard turns on provenance, not on the answer — a brute-forced label is
discarded … a certificate SAN or a zone-file entry is admitted as shadowed"*, which is about
`www.example.com` sitting **under** a wildcard. `Shadowed` is the value those names take.
[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md) rules that **no `Name` is ever
enumerated** and that a name scope has no `Coverage` denominator. None of the three says what
`*.example.com` itself is.

It is not a corner. A wildcard certificate is the ordinary shape of a modern estate, `crt.sh` ships
enabled ([#3](https://github.com/winniel123/verge-asm/issues/3)), and its answer for a wildcarded
estate is largely wildcard SANs. A spec that carries CT as an admitting source and cannot say what
one of its commonest rows admits is incomplete on an admission path.

## Decision

**A wildcard SAN denotes a set of names rather than a name. It admits none of them, it admits no
name of its own, and it admits nothing else either — and `*.example.com` is not a `Subject` of any
kind, from any source.**

| Concern | Decision |
| --- | --- |
| Does `*.example.com` in a `dNSName` SAN admit a `Name`? | **No** |
| Does it admit `example.com`? | **No.** `*.example.com` matches `foo.example.com` and **not** `example.com` `[spec]` RFC 6125 §6.4.3 rule 2 — so admitting the apex would invent a subject the evidence expressly excludes |
| Does it admit the names beneath it? | **No.** It enumerates none of them, and ADR-0047's *no `Name` is ever enumerated* is the same refusal one term over |
| What it is, in one sentence | A **matching construct in a presented identifier** — allowed there for backward compatibility `[spec]` RFC 6125 §1.8, §6.4.3, and not a well-formed `dNSName` at all, since RFC 5280 §4.2.1.6 requires *preferred name syntax* and `*` is not in it |
| Is `*.example.com` **ever** a `Name` subject? | **No** — not from a SAN, not from the operator's zone file, not from the wire |
| The operative rule on the SAN path | **No `dNSName` value containing octet `0x2A` in any label admits a `Name`.** Two limbs, both specification-supplied |
| Limb one — leftmost label is exactly `*` | RFC 6125 §6.4.3 rule 2's wildcard. It denotes a **set**, so there is no single thing for a subject key to be |
| Limb two — `0x2A` anywhere else (`baz*.example.net`, `bar.*.example.net`, `**`) | **Two denotations.** RFC 6125 §6.4.3 marks these *SHOULD NOT* and *MAY*, so the same octets are a pattern to one client and a literal label to the next. Refused rather than interpreted — ADR-0051's test verbatim |
| The same octets in a **zone file** or from **the wire** | A **name**, and admitted, wherever RFC 4592 §2.1.2 says they are not an asterisk label — `the*.example.com` and `**.example.com` are ordinary names. Only the leftmost single-`*` label is the wildcard, and that one is not a subject |
| Which field carried it | **Irrelevant.** The rule is about what the string denotes, so it applies to `common_name` as it does to `name_value` |
| ADR-0055's key function | **Untouched and not reopened.** It still does not read `*`; `*.example.com.` still keys. This is admission, and admission is a different step |
| Is the refusal a `Derivation` leaf? | **No leaf, no version, no `Break`.** It is a pure function of the one value being read |
| ADR-0011's differ prohibition | **Untouched.** Nothing here is in a differ, and there is no `certificate` diff involved — CT has no facet |
| `certificate-hostname-san-mismatch` | **Untouched, and vindicated.** It *matches* a presented wildcard under RFC 6125. **Matching is not admitting** |
| `Shadowed` | **Untouched.** It decides names beneath a wildcard, and after this ruling no name is left undecided |
| `wildcard-discrimination` | **Untouched**, and a defect it would otherwise have had is closed by construction |
| What it does to `Coverage` | **No denominator, no count of the estate, no figure.** A prose statement in [#28](https://github.com/winniel123/verge-asm/issues/28)'s fill half |
| Drift under the standing rule | **None.** Nothing enters and nothing withdraws. Admitting would have done both, within one cadence, on the commoner branch |
| A new `Subject` kind or lifecycle | **Neither.** The ticket's constraint is met with room to spare |
| Enforcing corpus | **ADR-0051's corpus, extended again** — same artefact, same inverted gate, rows whose expected output is *not a subject* |
| Cost of the ruling | **Zero.** No wildcard SAN has ever admitted anything, and nothing has shipped |

## Rationale

### A wildcard SAN denotes a set, and a subject key is the thing denoted

This is the whole ruling and the rest is consequence.

[ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)'s rule
is that **a subject key is the thing denoted, never the text that named it**, and its refusal test is
*an identifier with two denotations does not denote, so it has no key*. `010.1.1.1` fails that test
by having two; `bücher.example` typed fails it the same way.

`*.example.com` in a SAN fails it in the purest available form. It does not have two denotations. It
has **no single denotation at all** — it denotes an unbounded set of names, and RFC 6125 §6.4.3 is
explicit about what that set is: *"`*.example.com` would match `foo.example.com` but not
`bar.foo.example.com` or `example.com`"*. One label, exactly, and never the parent.

So there is no *thing* for a key to be the key of. That is not a fact about our machinery; it is
what the string is. A pattern over names is not a name in the way a CIDR is not an address, and the
model already refuses the analogous move: an address scope is a `Seed`, never an `Address`, and it
enumerates precisely because arithmetic closes it. A wildcard's extension has no arithmetic and no
closure, which is
[ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md)'s *no `Name` is ever enumerated*
restated with a certificate in front of it.

The issuance specification agrees, and it draws the line in its own definitions. The CA/Browser
Forum Baseline Requirements define a **Wildcard Domain Name** as *"a string starting with `*.`
(U+002A ASTERISK, U+002E FULL STOP) immediately followed by a Fully-Qualified Domain Name"* —
a string followed by an FQDN, and therefore expressly **not** one. `CONTEXT.md`'s `Name` is *a
fully-qualified domain name*; the BRs put a wildcard in a different category from that, and the
category they put it in is a matching pattern.

PKIX goes further and does not admit the form at all. **RFC 5280 §4.2.1.6 requires a `dNSName` to
be in *preferred name syntax*** `[spec]` RFC 1034 §3.5, RFC 1123 §2.1 — letters, digits and hyphens.
`*` is not in that alphabet. A wildcard SAN is therefore not a malformed name we should repair, and
not a name in an unusual spelling we should fold; it is a construct **RFC 6125 §6.4.3 tolerates for
backward compatibility** in a *presented* identifier, and RFC 6125 §1.8 gives it no place in a
*reference* identifier at all. That asymmetry is the whole of the model's relationship with it, and
it is why the one place we already read a wildcard SAN is a matcher and not an admitter.

### The evidence does not reach the subject, which is a stronger objection than the keying one

Even setting the denotation aside, the admission fails on evidence — and this is the limb worth
carrying, because it is the one that would survive if somebody found a way to key a pattern.

A CA issues `*.example.com` on proof of **control over `example.com`**. The Baseline Requirements
say so mechanically: for a wildcard request the CA *removes the `*.` prefix* and applies the chosen
validation method to the resulting FQDN. Nothing in that process observes, requires or implies the
existence of a `*` owner name in the zone — and wildcard certificates are routinely held by estates
whose zones enumerate every name and contain no wildcard record whatever, because the certificate is
bought for convenience rather than to match a DNS construct.

So a wildcard SAN is evidence about a **control relationship over a zone the operator has already
declared as a `Seed`**, and evidence about **no name**. `authority` governs *whose word is enough to
put a subject in the estate*; here there is no subject the word is about. That is
[ADR-0027](./0027-a-source-may-admit-without-observing.md)'s own discipline read in the direction it
did not have to be read: it sharpened ADR-0012's test to *does it admit subjects*, and a wildcard
SAN is the first row on which the sharpened test returns **no** for a source that otherwise passes
it every day.

It also lands on the rule ADR-0027 already stated for the decoder and generalises cleanly:
**a source's shape is translated, never its facts**. Turning *a certificate may be presented for any
name in this set* into *this name exists* is not a translation. It is manufacturing an observation
nobody made, one step further along than the decoder failure ADR-0027 named.

### Admitting it manufactures drift, and on the commoner branch it manufactures the worst kind

The ticket asks whether admission offends the standing rule that a certificate reissued with the
same wildcard must not enter and withdraw a subject. It does — but the reissue case is not where it
bites hardest, and the branch that does is worth writing down because it is the majority one.

Suppose CT admits `*.example.com` as a `Name`. Our resolver is `enumerable` over a single `Name`
(ADR-0006) and re-queries every known name every cadence, so within one cadence it queries this one.
Two branches, and the model fails on both.

**The zone has no wildcard record** — the ordinary case, per the section above. RFC 4592 §2.3 is
decisive here: *"when a wildcard domain name appears in a message's query section, no special
processing occurs. An asterisk label in a query name only matches a single, corresponding asterisk
label in the existing zone tree."* With no such label in the tree, the query returns a **Name
Error**. Under ADR-0006 a `Name` leaves exactly when our resolver measures one from every available
vantage — so the subject **withdraws in the first cadence after it entered**. A `Name` fires a
membership message on entry ([ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)),
so the operator is told a name appeared and then that it went, once per wildcard certificate, for an
event in which nothing in the world moved. That is the seam the standing rule exists to close,
arriving through the admission step rather than through a key.

**The zone does have one.** Then RFC 4592 §2.3's exact match answers, and the subject stays — which
is worse in a quieter way, because three things beneath it are then wrong at once.

- The answer is the wildcard's own RRset, which is **by construction** the poison signature
  `wildcard-discrimination` measures from random labels under the apex. The match predicate would
  therefore record **`Shadowed`** on the one name in the zone that is not shadowed. Repairing that
  means teaching a **versioned leaf** to read the `*`; refusing the subject means the query is never
  made and there is nothing to repair. The ruling pays for itself here rather than costing.
- The resolution admits an `Address`, the `Address` yields `Service`s, and a `Service` plus this
  `Name` yields an `Endpoint` whose `certificate` handshake sends **SNI equal to the `Endpoint`'s
  name** (`measurement-offers.md` §1.6). RFC 6066 §3's `HostName` is the fully qualified DNS
  hostname of the server being contacted, and no server is contacted at `*.example.com`. The
  measurement would be real and its subject fictional: whatever default virtual host answers a
  nonsense SNI, held on a timeline that moves when the operator reconfigures an unrelated vhost.
- `certificate-hostname-san-mismatch`'s domain is *`certificate` is `Presented` **and** the
  `Endpoint` has a `Name`* ([ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)), so
  that endpoint is **inside** it, and the rule's reference identity would be `*.example.com`. RFC 6125
  §1.8 provides no reading for a wildcard in a reference identifier. A rule with no defined answer on
  a subject in its own domain is exactly what ADR-0024 forbids a domain to contain.

Refusing admission produces none of this. Nothing enters, nothing withdraws, no membership message
fires, and a certificate reissued with the same wildcard — or with a different one, or with none —
changes nothing at all. **The quiet outcome is the correct one**, and it is quiet for the reason
`Completeness` already gives: CT is `corroborative`, so its silence licenses nothing, and a row we
admitted nothing from is not an absence assertion about anything.

### What is lost is what CT already documented itself as missing

The honest question is whether refusing costs discovery, and the answer is measured and already in
the repository. [`passive-discovery-sources.md`](../research/passive-discovery-sources.md) §2.1 lists
CT's **Misses** and names this one outright: *"wildcard certs hide the specific hostnames behind
`*.example.com`."* §2.2 repeats it for `crt.sh`. A wildcard certificate is a **concealment** of an
estate's names, not a disclosure of one, and this ruling ratifies a limitation the research recorded
rather than introducing one.

The practical bite is smaller still, and RFC 6125 §6.4.3 rule 2 is why. Because `*.example.com` does
not match `example.com`, an operator who wants both covered must have both in the certificate — so
the SAN list that would have tempted us into admitting the apex **already carries the apex
explicitly**, and the ordinary path admits it with no help from this ruling. Refusing therefore loses
nothing on the common shape and declines to invent on the uncommon one. And ADR-0047 has already
ruled that the `Seed` does not fill the gap either: *"names arrive from CT, the zone file and
resolution, and the `Seed` decides which of them are inside"* — a name scope filters, it does not
admit, so there was never a route by which the apex arrived from a pattern.

Where names behind a wildcard certificate genuinely need to be known, the model already names the
instrument and it is the strongest one it has: §3.3's *give me your zone*, which *"supersedes CT,
passive DNS, brute force, and web archives combined"*. That is a coverage answer, and it is where
this one goes.

### `Coverage` gets a sentence and never a figure

The ticket asks what this does to `Coverage`, against ADR-0047's rule that a name scope has no
denominator. The answer is that a wildcard is the **paradigm case of why that rule exists**, and it
changes nothing.

There is no denominator because the set the wildcard covers is unbounded — that is what a wildcard
is — so an `of:` would have to be a guess at how large the operator's estate ought to be, which is
precisely [#28](https://github.com/winniel123/verge-asm/issues/28)'s refused completeness score and
[#44](https://github.com/winniel123/verge-asm/issues/44) decision 7's refused *proportion of the
operator's estate*. Nothing may be rendered as a percentage, a shortfall or a count of hidden names,
because there is no such count and inventing one would be the score under a new name.

What is honest is a **statement about our own aperture**, which is the shape #44 decision 7 permits:
*this estate holds a wildcard certificate; certificate transparency can name nothing behind it, and
the zone is the instrument that can*. It sits in #28's **fill** half, where ADR-0027 already put
`crt.sh`, and it is prose. The one number that may back it is a count of **our own reading** — how
many `dNSName` values in the batch we declined to admit — surfaced rather than swallowed for
ADR-0055's reason at the adjacent boundary, that a source whose answer becomes all wildcards is
telling us something we should not lose. It is a count of what a source returned and never of the
estate, and it may not be rendered as a proportion of anything.

### The same octets are a name in one artefact and a pattern in the other, and neither reading is ours

This looks like an inconsistency and is the ruling's tidiest part, because both readings are read off
specifications rather than chosen.

**DNS is precise.** RFC 4592 §2.1.1 defines a wildcard domain name by a bit pattern — the leftmost
label being `0x01 0x2a` — and §2.1.2 shuts the door on everything else in terms: *"no label values
other than that in section 2.1.1 are asterisk labels, hence names beginning with other labels are
never wildcard domain names. Labels such as `the*` and `**` are not asterisk labels."* So on the DNS
paths the question *is this a wildcard* has one answer and the specification supplies it. A zone-file
owner name or a wire label of `the*` or `**` is an ordinary name and is admitted; and the presentation
format even separates the two readings itself, since a literal asterisk label is written `\042` and
ADR-0055's parse already honours the format's escapes.

**PKIX is not precise.** RFC 6125 §6.4.3 says a client *SHOULD NOT* match a wildcard outside the
leftmost label and *MAY* match a partial one such as `baz*.example.net`. Those are two different
clients reaching two different answers about one string, which is `010.1.1.1`'s shape exactly — and
ADR-0051's rule is that where **we** would have to supply the equivalence, the form is **refused
rather than interpreted**.

That is why the operative rule on the SAN path is the blunt one — **no `dNSName` value containing
`0x2A` in any label admits a `Name`** — while the DNS paths keep the specification's precise test.
The bluntness is real and is stated as ADR-0055 stated the identical bluntness for U-labels: a SAN
carrying a literal, non-wildcard asterisk label would be refused alongside the patterns. The exposure
is nil rather than small, because RFC 5280 §4.2.1.6's preferred name syntax excludes the octet
outright, so any `dNSName` carrying one is already outside the format.

And the asymmetry itself is not new. ADR-0055 already ruled that *"a high-bit octet arriving from the
wire is keyed as those octets and makes a perfectly good subject, while the identical octets arriving
as text are refused"*, on the ground that from the wire the octets **are** the label and in text there
are two readings. This is that sentence with one octet substituted.

### The refusal reads one octet, and that is not the classification ADR-0011 forbids

A session will find ADR-0011's sentence and think it has caught a contradiction, so it is answered
here rather than left:

> The moment a **differ** wants to say *the SAN list gained a **wildcard***, it is classifying;
> classification is reference data; and it has become a derivation that must be versioned.

Nothing here is in a differ. A differ is *a pure function of exactly two canonical values*, it renders
a diff of a facet, and **CT has no facet, no decoder and no timeline** (ADR-0027) — so there is no
`certificate` diff in the vicinity of this decision at all. What ADR-0011 forbade was a projection
naming a category to the operator, and it named the correct home for anything richer: *"anything
richer than a diff is a `Signal`, which is versioned already"*. The model **already** reads wildcards
in SANs, in exactly that home — `certificate-hostname-san-mismatch` matches them under RFC 6125, and
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) records it as needing no
curated table.

What this ruling reads is one octet position of the one value being read, in the admission step, and
it consults nothing else. So it takes **no leaf, no version and no `Break`**, on ADR-0021's own test:
its output cannot move while the world does not, because its content is RFC 4592 §2.1.1's bit pattern
and RFC 6125 §6.4.3's two conditionals, and neither is ours to revise. That is the same bargain
ADR-0051 struck for the key function, at the boundary immediately above it — which is also why the
enforcement is the same artefact rather than a sibling.

### The corpus takes it, because the corpus enforces a rule rather than a datatype

ADR-0055 rode ADR-0051's corpus instead of minting one, on the ground that *"the corpus enforces a
**rule** rather than a datatype … and there is one rule, one gate and one build failure."* The same
ground applies and the same artefact takes it, because this **is** that rule — *a subject key is the
thing denoted* — applied to an identifier that denotes no single thing.

The corpus's output column already carries *not a subject*: ADR-0055 rules that a name that cannot be
keyed is not one. The rows below add the same output reached by a different route, and each says
which route in its prose claim, so the two are never confused.

| Input | Expected | The claim |
| --- | --- | --- |
| `*.example.com` as a `dNSName` SAN | **no subject** (the label sequence keys; nothing is admitted) | a wildcard SAN denotes a set of names, so there is no thing for a subject to be |
| `*.example.com` as a `dNSName` SAN, asked for `example.com` | **no subject** | `*.example.com` does not match `example.com` (RFC 6125 §6.4.3 rule 2), so the apex is not even inside the set |
| `baz*.example.net` as a `dNSName` SAN | **no subject** | a partial wildcard is a pattern to one client and a literal label to the next (RFC 6125 §6.4.3 rule 3) |
| `bar.*.example.net` as a `dNSName` SAN | **no subject** | RFC 6125 §6.4.3 rule 1 — and the same two-denotation refusal |
| `*.example.com.` as a zone-file **owner name** | **no subject** | it is the wildcard, by RFC 4592 §2.1.1's bit pattern; the model holds no subject for the rule an authority applies |
| `the*.example.com.` as a zone-file owner name | `the*` `example` `com` root | RFC 4592 §2.1.2 — *`the*` and `**` are not asterisk labels*, so this is an ordinary name |
| `\042.example.com.` in a master file | `*` `example` `com` root | the format's escapes are the format's; an escaped asterisk is a literal label and a real name |

The last two rows are the ones that keep the ruling from over-reaching, and the last one is the reason
the DNS side needs no blunt test: presentation format already separates the wildcard from a literal
asterisk label, and ADR-0055's parse already honours it.

### Where this is thin, stated rather than smoothed

- **The refusal of the zone-file wildcard owner name is by-catch, and it is the one part of this
  ruling that could have been left open.** It is taken because the alternative is one label sequence
  that is a subject from one source and not from another, which puts the nonsense-SNI `Endpoint` and
  the false-`Shadowed` reading back in the model through the door the SAN ruling just shut. It is also
  in direct tension with RFC 4592, which calls a wildcard domain name *an ordinary domain name*: it is
  an ordinary domain **name**, and the claim here is narrower — that it is not a `Subject`, because
  the observations the model makes about a `Name` run through to an `Endpoint` that a client contacts,
  and nothing contacts a wildcard. That is an argument from the model's shape rather than from a
  document, and it is flagged as one.
- **The wildcard's own content becomes invisible as a row, deliberately, and it is not measured how
  much that costs.** *Your wildcard now points somewhere else* is a real fact and after this ruling no
  message carries it: every name beneath stays `Shadowed`, which is exactly the drift suppression
  `Shadowed` was built for, and the repoint rides underneath it. The content is not lost — the poison
  signature measures it every batch — but nothing holds it. That is a facet-shaped want, priced by
  [ADR-0015](./0015-the-value-space-is-the-commitment.md) at `revealed` plus one message with no
  `Break`, and ticketed rather than absorbed.
- **The count backing the `Coverage` sentence has no drawn surface.** #28's fill half is prose today
  and nobody has drawn where a *CT saw only wildcards here* statement sits or how loud it is. The
  claim that it may not be a proportion is firm; the judgement that a bare count of declined values is
  the right backing is a judgement.
- **The majority-branch argument is argued from the validation procedure rather than measured.** That
  wildcard certificates are commonly held by zones with no wildcard record follows from the Baseline
  Requirements validating the pruned FQDN and from what a wildcard certificate is bought for. Nobody
  has counted it, and the ruling would survive the other way round — the second branch fails too, in
  three places — but the *commonest* branch is asserted rather than measured.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) changes in three places.** `Source` records that a wildcard SAN
  carries no `Name` and that a source's admitting property has a boundary; `Name` records that a
  wildcard is a pattern rather than a name and is a subject nowhere; `Shadowed` records that its rule
  decides names **beneath** a wildcard and that the wildcard itself is not a subject, which is what
  makes the sentence complete rather than partial. **No term is added and no term changes meaning.**
- **ADR-0055's named residue is discharged**, and ADR-0055 is **confirmed rather than amended**: its
  key function is untouched, `*.example.com.` still keys, and the ruling sits one step above it in the
  admission step it expressly declined to reach.
- **[ADR-0027](./0027-a-source-may-admit-without-observing.md) gains its first boundary and is
  otherwise unamended.** Its sharpened test — *does it admit subjects* — is the test applied here, and
  `crt.sh` still passes it on every non-wildcard row. What is new is that a source may admit on some
  of a batch's rows and nothing on others, which its `corroborative` `Completeness` already absorbs.
- **[ADR-0006](./0006-subjects-leave-by-measurement.md) is confirmed and now has a stated boundary.**
  *Admission under a wildcard turns on provenance* governs names **beneath** a wildcard and always
  did; it never spoke to the wildcard name, and it does not now.
- **[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)'s
  `certificate-hostname-san-mismatch` row is untouched and vindicated.** It reads a presented
  wildcard SAN and matches it under RFC 6125, which is the one legal thing to do with a pattern.
  **Matching is not admitting**, and after this ruling that sentence is the model's whole account of
  how it treats a wildcard SAN.
- **The `Derivation` vector gains nothing**, and this is worth stating because a session will look:
  no admission leaf, no version, no `Break` cause. ADR-0021's five prober leaves stay five, and
  `wildcard-discrimination` is not touched — the `*` query it would have had to learn to recognise is
  a query the model never makes.
- **ADR-0051's corpus gains seven rows and a third input shape** — a `dNSName` value — under the same
  artefact and the same inverted gate. There is no new build failure and no new artefact.
- **The `Seeds` screen owes refusal copy for a typed `*.example.com`**, naming the **subtree
  exclusion** as the route, in the shape ADR-0055 established for a U-label and
  [#85](https://github.com/winniel123/verge-asm/issues/85) for an over-cap declaration. The two are
  not the same object and must not be silently translated: a subtree exclusion runs on label-wise
  suffix equality over the key and therefore **includes** `example.com`, while `*.example.com`
  expressly excludes it (RFC 6125 §6.4.3 rule 2).
- **`Coverage` gains a sentence in #28's fill half and no figure anywhere.** No denominator, no
  proportion, no count of hidden names. The count that may back it is of `dNSName` values we declined,
  which is ours and not the estate's.
- **The cost is zero and there is no re-baseline.** No wildcard SAN has ever admitted a subject, so
  nothing is withdrawn, no timeline closes, no `Break` fires and no message is owed. This is
  ADR-0027's cheap direction, arrived at for the same reason: the correction removes something that
  could not have produced a value.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) can state what CT admits, in full:** the
  `Name`s a certificate's SAN list carries, excluding every value containing an asterisk label, and
  nothing else — with the `Citation` hopping to the `Batch` per ADR-0027.
- **One residue is ticketed rather than absorbed:** whether the measured wildcard poison signature is
  a value the model **holds**, and on what subject. This ruling makes that question sharper by
  removing its most obvious candidate answer, and ADR-0015 prices a new facet at `revealed` plus one
  message with no deadline, so it does not block #12.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **`*.example.com` is a `Name` subject like any other** — the *losing option*, and the one ADR-0055's clean keying most naturally invites | It loses on drift, twice, on branches that partition the estate. Where the zone has no wildcard record — the ordinary case, since the CA validated the pruned FQDN and never the record — RFC 4592 §2.3 returns a **Name Error** and ADR-0006 withdraws the subject in the first cadence after it entered: a membership message and a withdrawal, per wildcard certificate, for an event in which nothing moved. Where the zone does have one, the exact-match answer **is** the poison signature so `wildcard-discrimination` reads `Shadowed` on the one name that is not shadowed; the `Endpoint` sends an SNI RFC 6066 §3 has no reading for; and `certificate-hostname-san-mismatch` gets a reference identity RFC 6125 §1.8 does not define, inside its own domain |
| **It admits `example.com` and nothing else** — the middle position, and the one that looks generous | RFC 6125 §6.4.3 rule 2: `*.example.com` matches `foo.example.com` and **not** `example.com`. Admitting the apex from a pattern that expressly excludes it is inventing a subject the evidence rules out. It also buys nothing: because the wildcard does not cover the apex, a certificate meant to cover both **carries the apex as its own SAN**, which the ordinary path admits already |
| **It admits the names beneath it** | It enumerates none of them and cannot — the set is unbounded. ADR-0047's *no `Name` is ever enumerated* is this refusal one term over, and §10's refused wordlist brute-force is what admitting a guess from a pattern amounts to |
| **Fold it: strip `*.` and treat the SAN as evidence about the parent zone** | The BRs' own validation does exactly this pruning, which is what makes it tempting — and what it proves is **control**, not existence. The model has a term for a claim about where an estate ends and it is `Seed`, which is Declared and the operator's; a third party's certificate is not one |
| **Admit it, and carve `wildcard-discrimination` out so the wildcard name is not read as `Shadowed`** | Teaches a **versioned leaf** to read the `*` in order to repair a defect the admission created, and leaves the SNI and reference-identity failures untouched. Refusing the subject means the query is never made |
| **Admit it as a subject with no facets — membership only, no timelines** | A `Subject` is *anything an observation can be about*, and one that no observation can be about is a row that never moves. ADR-0027 rejected the fifth `Subject` kind on that exact ground — *a subject whose every span holds the same value forever is a timeline with nothing in it* — and this is that shape without even the value |
| **Let the wildcard SAN feed `Shadowed`, since it is evidence a wildcard exists** | It is not: a wildcard certificate implies no wildcard record, and a wildcard record implies no certificate. And `CONTEXT.md` already forbids the mechanism — `Shadowed` is decided by the **measurement binary inside one batch**, never assembled afterwards from two observations, still less from a third party's log row |
| **Refuse only the leftmost-label wildcard, and admit `baz*.example.net` as an ordinary name** | RFC 6125 §6.4.3 rule 3 says a client **MAY** match a partial wildcard, so the same octets denote a pattern to one client and a label to the next. Two denotations, so ADR-0051 refuses rather than interprets. The precise test exists only on the DNS side, where RFC 4592 §2.1.2 supplies it |
| **Apply the blunt `0x2A` refusal to the DNS paths too, for symmetry** | RFC 4592 §2.1.2 is explicit that `the*` and `**` are **not** asterisk labels, and a literal asterisk label is writable as `\042` in presentation format. Refusing them would discard real subjects to make a text-boundary rule look tidy — ADR-0055's own words for the identical move on high-bit octets |
| **Leave the zone-file wildcard owner name as a `Name` and rule only the SAN** — the narrowest reading of the ticket | One label sequence would then be a subject from one source and not another, which is defensible on `authority` and which reintroduces every failure the SAN ruling closed, through the zone file. The by-catch is flagged rather than hidden |
| **State a `Coverage` figure for what a wildcard hides** | The set is unbounded, so any denominator is a guess at how large the estate ought to be — #28's refused completeness score and #44 decision 7's refused proportion. ADR-0047 already rules a name scope has no denominator, and a wildcard is why |
| **Give this its own corpus** | ADR-0055's answer, unchanged: the corpus enforces a rule and not a datatype, and there is one rule, one inverted gate and one build failure. It gains an input shape, not a sibling |
| **Ship a facet holding the measured wildcard poison signature in v1, so the wildcard is visible** | ADR-0015 prices a new facet at `revealed` plus one message with no `Break`, so it has no deadline — the same ground on which ADR-0027 deferred a CT-fed facet on `Name`. Ticketed, not absorbed |
| **Leave it open and let #12 record both readings** | It is an admission path on the model's flagship keyless source, and a spec that cannot say what one of its commonest rows admits is incomplete where it is least affordable |
