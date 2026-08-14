# ADR-0035: A cryptographic primitive's owner is the body that specifies it — and a floor enforced by relying parties attests nothing about the primitive

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#68 What is `certificate-weak-key-or-signature`'s table, and who owns a claim about the WebPKI?](https://github.com/winniel123/verge-asm/issues/68)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) ruled that an evidence
standard attaches to a **table**, that gate 2 — attestation by the owner — generalises to every
project-authored table asserting about the world, and that
`certificate-weak-key-or-signature`'s thresholds are the third such table in v1 and had never been
written. It also recorded, deliberately unresolved:

> **[#68] will test §10.5's owner definition on a case it was not built for.** *Owner* keys on the
> artefact — the party that designed the protocol or authors the reference implementation. A claim
> about the WebPKI's key and algorithm floor is owned by a body that designed neither X.509 nor any
> implementation of it. That is a live question for #68 and is deliberately not pre-empted here.

[`sensitive-ports.md`](../research/sensitive-ports.md) §10.5 defines **owner** as *the party that
designed the protocol, or that authors the reference implementation, speaking about the thing it
designed or wrote*, adds that *a distributor owns its own shipped configuration and owns nothing
about the protocol*, and states the discriminator that decides this ADR: **the artefact, not the
party, is what the rule keys on.**

#68 set out three routes and flagged the third as the most interesting and most likely to overreach:
(a) the root programmes **are** owners because the WebPKI is a thing they collectively author; (b)
they are corroborators and the rule rests on the algorithm specifiers alone, accepting whatever
coverage that loses; (c) the CA/Browser Forum Baseline Requirements are a **shipped default** under
§2.2's third form, so §10.4's *attests only where it restricts* applies and a floor is unambiguously
a restriction.

The full working is in [`docs/research/weak-key-and-signature.md`](../research/weak-key-and-signature.md).
This ADR records the parts that are general.

## Decision

| Concern | Decision |
|---|---|
| Does §10.5's owner definition survive | **Yes.** It gains one clause naming an artefact class it did not enumerate, and nothing else changes |
| Who owns a claim about a cryptographic primitive | **The body whose standard specifies the primitive**, speaking about the primitive it specified. NIST for SHA-1, SHA-2, ECDSA and the P-curves; the IETF for MD5, and for what a certificate or a TLS handshake may carry |
| Where a relying-party floor stands | **§10.5's distributor limb, unchanged.** A body specifying which primitives a population may **accept** owns its own shipped configuration and owns nothing about the primitive that configuration selects among |
| Route (a) — ownership by collective authorship | **Refused.** It would make *owner* key on **standing** rather than on an artefact, which readmits everything §2.3 excludes. Its premise also fails on measurement |
| Route (c) — the BR as a shipped default | **Refused, on §10.5's own artefact test.** §10.4 answers *whether* a default attests, never *what about*. A restriction over the relying party's acceptance is not an attestation about the primitive |
| Route (b) — the specifiers alone | **Adopted.** Measured cost: **no row lost**; what is lost is **modality** on the key-size rows, disclosed |
| A second table's closed claim set | **Derived, not exempted.** Two claims for this table, closed over the two security properties a certificate's own parameters must deliver. ADR-0032's *derive your own set* method survives its second application |
| Is a one-dimensional table exempt from gate 1 | **No, and dimensionality is not the test.** A table is exempt from gate 1 only where it is not about the world at all |
| Determinacy for this table | **Outside the domain**, as ADR-0032 predicted — and the move that would bring it inside is named and is a plausible future tidy-up |
| Release coupling | **Passes with three orders of magnitude of headroom**, measured. Two near-misses that would have failed it are named and refused |
| Where the §10.5 amendment lands | In `sensitive-ports.md` §10.5, applied by the parent session — #68 could not edit that file, which was concurrently held |

## Rationale

### 1. The protocol owner is silent by design, and its silence points somewhere

RFC 5280 sets no numeric floor anywhere. Its Security Considerations say why:

> "The binding between a key and certificate subject cannot be stronger than the cryptographic module
> implementation and algorithms used to generate the signature. Short key lengths or weak hash
> algorithms will limit the utility of a certificate. CAs are encouraged to note advances in
> cryptology so they can employ strong cryptographic techniques. In addition, CAs SHOULD decline to
> issue certificates to CAs or end entities that generate weak signatures."

This is a stronger object than the *"security issues are not discussed in this memo"* non-statement
`sensitive-ports.md` §2.7 met at RFC 1833. PKIX **names the property the rule is named for**,
declines to supply a number, and refers the number to *"advances in cryptology"*.

**So the search for an owner was never a search for a replacement authority.** It was a search for
the party PKIX pointed at, and that party is whoever specifies the algorithm. The answer was inside
the protocol owner's own text, which is why §10.5 needed a clause rather than a rewrite.

### 2. The clause, and why it admits nobody new

> **A cryptographic primitive's owner is the body whose standard specifies the primitive**, speaking
> about the primitive it specified. NIST owns SHA-1, the SHA-2 family, ECDSA and the P-curves,
> because FIPS 180-4, FIPS 186-5 and SP 800-186 are what those algorithms **are**; the IETF owns MD5
> because RFC 1321 is, and owns what a certificate or a TLS handshake may carry because RFC 5280 and
> RFC 8446 are.
>
> **A body that specifies which primitives a population may accept — the CA/Browser Forum over the
> WebPKI, a root programme over its own trust store — is not thereby an owner of the primitive.** It
> is in §10.5's **distributor** position exactly: it owns its own shipped configuration and owns
> nothing about the artefact that configuration selects among. Its restriction is admissible under
> §2.2's third form **over its own artefact**, and §2.3 governs its prose: corroboration, never sole
> grounds.

§10.5 already said *the IETF owns what its RFCs specify*, so the first paragraph is that sentence
generalised from RFCs to standards. The second paragraph is the distributor limb applied. **Nothing
is admitted that §10.5 did not already admit**; what is added is the artefact class, so the next
session meeting a claim about an algorithm rather than about a protocol does not rediscover it.

### 3. Route (c) is refused by a sentence §10.5 was already carrying

Route (c) is the strongest of the three and every step of it is sound:

1. §2.2's third form admits *the project's shipped default*.
2. §10.4 rules that a shipped default attests **where it restricts**.
3. A floor is unambiguously a restriction.
4. Browsers enforce the floor in code — the most literal possible reading of *a maintainer position
   expressed in code rather than prose*.

**The conclusion does not follow, because §10.4 answers *whether* a default attests and never *what
about*.** PostgreSQL's `listen_addresses = localhost` admits the 5432 row because the artefact
PostgreSQL restricted — exposing PostgreSQL to the network — **is what the row is about**. A
browser's verifier refusing SHA-1 restricts **the browser's own acceptance**; the row is about
**SHA-1's collision resistance**. The artefacts do not match, and §10.5 says it outright: *"The
artefact, not the party, is what the rule keys on."*

This is the second time that sentence has done the deciding work without being cited as the rule —
§9.1 read Debian's `rpcbind.default` while declining Red Hat's Security Guide on it before §10.5
existed. **It is promoted here from a discriminator inside a definition to the general test**: to
place a party, name the artefact its statement is about, and ask whether that artefact is what the
row claims.

### 4. The refusal was principled and the measurement made it easy

Recorded because the gap between the two is the interesting part. Had route (c) been taken, the table
would have rested on BR v2.2.9, and three measured properties would have come with it:

- **The BR has no position on MD5 whatsoever** — the string occurs zero times. MD5 is excluded only
  by omission from a closed permitted list, which is the **absence of an act**, and §10.4's one-way
  rule says an absence attests nothing. Half the signature axis would have had no footing.
- **The BR's SHA-1 prohibition carries a live exception** — §7.1.3.2.1 permits `RSASSA-PKCS1-v1_5
  with SHA-1` under a five-condition carve-out *"Until 2026-09-15"*. A row founded on it would have
  been founded on a conditional expiring four weeks after it was written.
- **The BR's ECDSA restriction is not a weakness claim.** It permits P-256/384/521 and closes the
  list. That excludes P-224 from **issuance**; nowhere does it say P-224 is weak, and NIST — which
  specified the curve — rates `len(n) ≥ 224` acceptable.

Route (a)'s premise fails the same way. *The root programmes collectively author the WebPKI* implies
four bodies with positions: **Chrome and Apple state no algorithm requirement of their own** and
delegate to the BR; **Mozilla still enumerates SHA-1** as a permitted signing algorithm; and
**Microsoft recommends against ECDSA entirely** — *"ECC/ECDSA certificates shouldn't be issued to
subscribers"* — on Windows compatibility grounds, in the same table as its key-size floor. That is
§2.3's measured AWS pattern arriving on the first case.

**The rider from §10.5 binds and is restated so the measurement does not become the ground:** *a
session that finds a consistent distributor and reads the door as open has read the wrong sentence.*
The contradictions are evidence the line is right. They are not the reason for it.

### 5. ADR-0032's *derive your own set* survives its second application, and gains a boundary

ADR-0032 ruled gate 1's three claims a theorem about `Exposure` and required a second table to derive
its own set from what its rule reads. #68 did, and the derivation is **two claims**: a certificate's
own cryptographic parameters must deliver a **work factor** and a **collision resistance**, and
nothing else. Four candidate thirds were tested and refused, and the derivation **changed the
table** — DSA lost a blanket row that FIPS 186-5's withdrawal of approval would otherwise have
carried.

ADR-0032 also asked whether a one-dimensional table might be exempt, and required that any exemption
state its general form. **It is not exempt, and dimensionality is not the test:**

> **A table is exempt from gate 1 only where it is not about the world at all** — ADR-0032's own
> line, `k` and the availability window and the coverage threshold. A table can be one-dimensional in
> its output and still be attested by different sentences in different documents for different
> reasons, and collapsing those into one undifferentiated verdict is how a laundered opinion gets in.

That is the general statement ADR-0032 asked for, and it closes the exemption route rather than
leaving it open for the next table to argue.

### 6. Determinacy stays outside the domain, and the move that would break it is named

The table's key is `(algorithm, parameter)` and the digest OID — fields RFC 5280 defines to carry
exactly these facts. No surrogate, so gate 3 is **outside the domain**, as ADR-0032 predicted.

The check was worth running, because there is a plausible way in that looks like a tidy-up. 2048-bit
RSA and a 224-bit curve order both deliver about 112 bits of security; the integers are not
comparable. A session collapsing the table to a **single bit-count threshold** would be keying on a
surrogate for security strength, and gate 3 would come inside the domain and fail at once.

> **The table's key must be `(algorithm, parameter)` and never a bare bit count.** That is not
> presentation. It is what keeps gate 3 out of the domain.

### 7. A third kind of weak row: scope, distinct from watched and from chased

`sensitive-ports.md` §10.6 and §11 established two kinds of weak row and forbade collapsing them:
one **watched** (a restricting shipped default that could silently flip permissive — the silent
de-attestation ADR-0032 §8 named), one **chased** (a corroborator standing where an owner should — a
retrieval).

This table's weak tier is neither. Its three key rows rest on NIST, which **is** an owner under §2
above and speaks directly about primitives it specified — but NIST scopes its own normative verbs, in
its own definitions section: *Shall* is *"A requirement for Federal Government use"* and *Approval
status* is *"Used to designate usage by the U.S. Federal Government."*

The split runs inside a single sentence. *"The security strength provided by an elliptic-curve-based
signature algorithm is no greater than 1/2 of the length of the domain parameter n"* is arithmetic
and unqualified; *"Therefore, the length of n shall be at least 224 bits to meet the minimum
security-strength requirement of 112 bits for Federal Government use"* is the threshold and is
scoped. **The strength computation is universal; the number 112 is federal.**

> **A scope weakness is a third kind, and it behaves like neither of the other two.** It cannot flip
> silently, so it is not watched. It cannot be chased by finding the right party, because the right
> party has already spoken. It resolves only if a **second** owner speaks unscoped — so its retrieval
> is a search of a corpus rather than a request to a party.

> **Amended by [#73](https://github.com/winniel123/verge-asm/issues/73) — the corpus search was run
> and came back positive.** The prediction above held on its first test and the paragraph stands; two
> things are added. **The second owner was the IETF**, already carrying two rows of this table, and its
> unscoped statements were in **document classes §1 never asked** — RFC 9325 §4.5 (BCP 195, `MUST`,
> a 2048-bit RSA modulus for a server certificate) and RFC 9846 §C.2 (`SHOULD`, *"certification paths
> containing keys or signatures weaker than 2048-bit RSA or 224-bit ECDSA are not appropriate for
> secure applications"*). §1's reading of RFC 5280's silence as *by design* is correct and its
> inference that the number therefore lives outside the IETF is not: **a specification's silence is not
> the owner's silence.** The weak tier goes from three rows to **DSA's `N ≥ 224` limb alone**; no row,
> floor or predicate moves. **§6's requirement that the key be `(algorithm, parameter)` and never a
> bare bit count paid out unexpectedly** — the security-strength level 112 has the thinnest IETF
> pedigree of any number in this area, and no row keys on it.
> See [ADR-0040](./0040-a-specifications-silence-is-not-the-owners-silence.md) and
> [`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §13. **One citation above is
> wrong**: RFC 5280's Security Considerations is **§8**, not §11.

## Consequences

- **[`sensitive-ports.md`](../research/sensitive-ports.md) §10.5 gains one clause**, the text quoted
  in §2 above. #68 could not apply it — that file was concurrently held by
  [#69](https://github.com/winniel123/verge-asm/issues/69) — so the amendment is carried in #68's
  resolution comment for the parent session to land. **Until it lands, this ADR is where the clause
  lives.**
- **`the artefact, not the party` is promoted to the general test for placing a source.** Name the
  artefact a statement is about; ask whether that artefact is what the row claims. It has now decided
  three cases (Debian vs Red Hat, PostgreSQL's bind address, the CA/B BR) and was cited as the rule
  in none of them.
- **[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s third curated table now has content.**
  Five rows: MD5 and SHA-1 on the signature digest; RSA `nlen` < 2048, ECDSA `len(n)` < 224 and DSA
  below (2048, 224) on the key.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) is unblocked** on this side. The
  sixteenth rule has a predicate, a table and a disclosure, and can be assembled.
- **The rule's predicate is written for the first time**, including two things nobody had settled:
  it ranges over **the presented chain** rather than the leaf, and the **signature limb skips a
  self-signed certificate while the key limb does not** — on RFC 8446 §4.2.3's own reason, that such
  signatures *"are not validated, since they begin a certification path"*.
- **An algorithm the table does not name is not weak: the rule is false, never `not-evaluable`.**
  The table can never be the reason a certificate is unevaluable. The alternative would make the
  rule's output a function of our own coverage and would flip an estate to `not-evaluable` the day
  post-quantum certificates ship.
- **A scheduled cryptographic transition is a scheduled edit, never a date in a predicate.** NIST
  expresses several transitions as calendar events; encoding one would move the rule's output at
  midnight with no version bump, defeating [ADR-0008](./0008-derivation-versions-move-on-content.md)
  from the inside.
- **A key-compromise blocklist is permanently out of this table** — refused twice independently, on
  the claim set and on release coupling. It is [#31](https://github.com/winniel123/verge-asm/issues/31)'s
  signature database by definition.
- **The map's curation patch gains a second, cheaper watch.** This table's correct posture is to
  re-read SP 800-131A when a revision goes final and otherwise do nothing — against the port list's
  continuous source drift.
- **`CONTEXT.md` is not edited**, on ADR-0032's precedent and for its reason: concurrent sessions are
  in that file. Nothing here changes a glossary term; the rule's predicate is spec content, not
  vocabulary.

## Alternatives rejected

| Alternative | Why not |
|---|---|
| **Route (a) — the root programmes are owners of the WebPKI** | Makes *owner* key on **standing** rather than on an artefact, which is a market position and not a thing anyone designed; once standing suffices there is no principled way to exclude the sources §2.3 was built on. The premise also fails on measurement: two of the four programmes state no algorithm text at all, and the two that do contradict each other and the specifiers |
| **Route (c) — the BR is a shipped default under §2.2's third form** | §10.4 answers *whether* a default attests, never *what about*. The BR restricts the relying party's **acceptance**; the row claims a property of the **primitive**. Refused on §10.5's artefact sentence — and the retrieval shows it would have cost the MD5 row entirely, imported a conditional expiring 2026-09-15, and set the ECDSA floor on an issuance rule mistaken for a strength claim |
| **Extend §2.1's three claims to this table** | ADR-0032 already ruled they cannot: they are closed over what an **internet vantage** supplies and only one v1 rule reads `Exposure`. Stretching them is #37's open-set failure |
| **Exempt a one-dimensional table from gate 1** | Available and tempting, and wrong. Dimensionality is not the test — exemption is for tables not about the world. This table is one-dimensional in output and two-dimensional in grounds, and collapsing the grounds is how a laundered opinion gets in |
| **Add a second threshold at RSA 3072 / 128-bit** | Severity by another name — *below 2048 = weak*, *2048–3071 = weakening* is a ranked family wearing a threshold's clothes, on a product that refused severity. It would also make the verdict move on the calendar rather than on a release |
| **Carry FIPS 186-5's withdrawal of DSA as a blanket row** | The withdrawal is an **approval decision**, not a work-factor statement, and no claim in the derived set fits it. Carrying it would reopen the set for one row that will never fire, which is exactly the failure closure prevents |
| **Add MD2 and MD4 rows for completeness** | RFC 6149 is Informational and hedges its own result — collision attacks *"are not that damaging"*, best attack 2^63.3 with 2^50 memory — and no owner sentence prohibits MD2 in a certificate. Excluded on §2.7's ground: we could not find anyone entitled to say it |
| **Hold the table out of the release until the key rows are unscoped** | Gate 2 has never been a shipping gate — `161/udp` and `5432/tcp` ship disclosed. Holding would also block #12 on a modality question that no available retrieval resolves quickly |
| **Fold this into ADR-0032** | ADR-0032's subject is *which instrument governs which table*; this ADR's subject is *how to place a source when the artefact is not a protocol*. Folding it would bury a general test — `the artefact, not the party` — inside a walk of sixteen rules |
