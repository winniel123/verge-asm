# The key and signature floor for `certificate-weak-key-or-signature`

Research ticket [#68](https://github.com/winniel123/verge-asm/issues/68) — wayfinder research for the
verge-asm v1 spec.

**Question.** What is `certificate-weak-key-or-signature`'s table — the key-size floor per algorithm
and the deprecated signature-algorithm set — and **who owns** a claim about the WebPKI's floor, given
that the body which sets it in practice designed neither X.509 nor any implementation of it?

**Why this document exists at all.** `certificate-weak-key-or-signature` has been in the v1 set since
[#16](https://github.com/winniel123/verge-asm/issues/16) and its predicate content had never been
written anywhere in this repo. It appears in
[ADR-0004](../adr/0004-signals-are-release-coupled-rules.md)'s v1 list and in
[ADR-0024](../adr/0024-a-rules-domain-is-the-extension-of-its-name.md)'s domain table, and nowhere
else. [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) found it
while walking all sixteen rules and ruled it the **third** curated table in v1 that asserts something
about the world — after the 37 sensitive `(port, transport)` pairs and `certificate-expiring`'s `N`.

**This is a second instrument with its own document**, which is what ADR-0032 implies: the evidence
standard attaches to a **table**, and a second table owes its own closed claim set, its own per-row
attestation, and its own weak-tier disclosure. It does not inherit
[`sensitive-ports.md`](./sensitive-ports.md) §2.1's three claims — that set is a theorem about
`Exposure` and does not transfer — and it does inherit §2.2's attestation gate with §2.3's
corroborator rule and §10.5's owner definition, which is the thing this ticket puts under test.

**The general parts of this note are recorded as
[ADR-0035](../adr/0035-a-cryptographic-primitives-owner-is-its-specifier.md)**: the owner clause, the
refusal of routes (a) and (c), the boundary on gate-1 exemption, and the third kind of weak row. What
stays here is the table, its footing, and the working.

> **Notation, because the two documents' section numbers overlap.** A bare section reference is to
> [`sensitive-ports.md`](./sensitive-ports.md) wherever it names a part of **the evidence standard**:
> §2.1's three claims, §2.2's attestation routes and its third form, §2.3's corroborator rule, §2.7's
> refusal, §10.4's one-way shipped-default rule, §10.5's owner definition, §10.6's two kinds of weak
> row. **Every other bare reference is to a section of this note.**

Four constraints bind before any evidence is gathered:

1. **No severity.** A weak key and a deprecated signature are **one rule and one fact**, per the
   rule's existing name. This note does not split it into a ranked family and does not introduce a
   middle tier as a second threshold (§6.3).
2. **Frequency is not a position.** *Most CAs stopped issuing SHA-1 in 2017* is frequency. *SHA-1
   must not be used* is a position. Only the second is cited, and §2.6 records what was refused.
3. **The table must stay release-coupled** ([ADR-0004](../adr/0004-signals-are-release-coupled-rules.md)).
   It passes with a decade of margin, and §7 tests it rather than assuming it — including the two
   near-misses that would have failed it.
4. **Editing the table later costs a uniform `Break` on this rule for one cadence**
   ([ADR-0008](../adr/0008-derivation-versions-move-on-content.md)) and nothing else. Cheap, which
   argues for getting it right rather than for getting it minimal.

---

## 1. Summary

| Decision | Answer |
|---|---|
| The table | **Five rows** — two on the signature digest, three on the key. §3 |
| The signature rows | **MD5** and **SHA-1**, keyed on the **digest**, not on the signature OID. §3.1 |
| The key rows | **RSA `nlen` < 2048**, **ECDSA `len(n)` < 224**, **DSA below (2048, 224)**. §3.2 |
| Does Ed25519 need a row | **No, and that is a fact about the algorithm rather than a gap.** EdDSA has no free size parameter for a floor to cut. §3.4 |
| The closed claim set | **Two claims, derived by construction** from the two security properties a certificate's own parameters must deliver: the key's **work factor** and the signature digest's **collision resistance**. §2.2 |
| Does §10.5's owner definition survive | **Yes, and it gains one clause.** *Owner* keys on the **artefact**, which §10.5 already says; the clause names the artefact class this case introduced — a **cryptographic primitive**, owned by the body whose standard specifies it. §2.3 |
| The three routes the ticket set out | **(b) wins.** (a) is refused because it would make *owner* key on standing rather than on an artefact — and its premise of collective authorship fails on measurement, two of the four programmes stating no algorithm text at all; **(c) is refused on §10.5's own artefact test**, and the retrieval shows it would also have imported a dated conditional into a release-coupled table. §8 |
| What (b) costs, measured | **No row is lost.** What is lost is **modality on the three key rows**: their only owner is NIST, whose normative verbs are Federal-scoped by NIST's own definitions section. That is the disclosed weak tier. §9 |
| Determinacy (gate 3) | **Outside the domain** — the key is the fact, not a surrogate for it. The condition that would bring it inside is named and is a real hazard. §2.4 |
| Which certificates in the chain | **Every certificate presented, except that the signature limb skips a self-signed one** — on the IETF's own reason, not ours. §4.1 |
| An algorithm the table does not name | **Not weak. The rule is false, never `not-evaluable`.** §4.2 |
| Release coupling | **Passes, measured** — roughly one edit per row per decade — and the two things that would have broken it are named and refused. §7 |

The headline result is the one that inverts the ticket's own framing:

> **The party the ticket suspected of being the real owner has no position on half the table.** The
> CA/Browser Forum Baseline Requirements v2.2.9 contain **zero occurrences of the string `MD5`**, and
> their SHA-1 prohibition is a closed permitted-list carrying a **live exception that does not sunset
> until 2026-09-15**. The IETF — an owner under §10.5 as already written — says of the same two
> algorithms that an endpoint receiving an MD5-signed certificate **"MUST abort the handshake"**.
> Route (b) is not the conservative option that loses coverage. On the signature axis it is the
> option with the stronger evidence, and route (c) would have put a dated conditional inside a
> release-coupled table.
>
> The same measurement dissolves route (a)'s premise. *The root programmes collectively author the
> WebPKI* implies four bodies with positions; **Chrome and Apple state no algorithm requirement of
> their own at all** and delegate to the BR, Mozilla still lists SHA-1 as a permitted signing
> algorithm, and Microsoft tells CAs not to issue ECDSA certificates on Windows compatibility
> grounds. The collective is one document and two partial restatements of it, disagreeing.

---

## 2. The evidence standard for this table

### 2.1 What this table inherits, and what it does not

Per [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md):

| Gate | Status here |
|---|---|
| **Gate 1 — a named claim from a closed set** | **Does not transfer.** `sensitive-ports.md` §2.1's three claims are closed over what an **internet vantage** supplies, and exactly one v1 rule reads `Exposure`. This table owes its **own** closed set, derived the same way from what **its** rule reads — §2.2 |
| **Gate 2 — attestation by the owner** | **Binds in full.** This table asserts about the world (*this key or algorithm is weak*), so §2.2's attestation route and §2.3's corroborator rule govern every row — worked below at 2.3 and 3.3 |
| **Gate 3 — determinacy** | **Outside the domain.** The surrogate gate has one v1 instance and this is not it — §2.4 |
| **The weak-tier disclosure** | **Binds**, in its stronger form: a disclosed weakness names the retrieval that would resolve it — §9 |

### 2.2 The closed claim set, derived rather than enumerated

ADR-0032 requires a derivation from **what the rule reads**, and offers no exemption. This note does
not claim one, and §2.2.1 records why the tempting exemption argument is wrong.

The rule reads a certificate's own cryptographic parameters: the `subjectPublicKeyInfo` algorithm
with its size or curve, and the `signatureAlgorithm` with its digest. Ask what those parameters are
*for*. A certificate is a signed binding between a name and a key, and its parameters carry exactly
two security properties on which that binding rests:

> **The claim set is closed.** A permitted claim must name a **security property that the
> certificate's own cryptographic parameters must deliver and do not**. A certificate's parameters
> deliver exactly two, and there is nothing else they are for:
>
> | The parameter must deliver | Whose failure it is | Claim |
> |---|---|---|
> | A **work factor** making forgery of the subject's authentication infeasible | the **key** | **Claim 1** |
> | **Collision resistance** making forgery of the issuer's binding infeasible | the signature's **digest** | **Claim 2** |
>
> Two properties, two claims. **The set reopens only if someone names a third security property a
> certificate's own parameters must deliver**, which is a falsifiable condition rather than a failure
> of imagination.

The construction is #37's, applied one artefact across: enumerate what the read thing is *for*,
rather than enumerating the cases that happen to have come up.

**Four candidate thirds were tested and all four fail.** Each refusal removes something a reasonable
session would otherwise have put in the table, which is how the closure earns its keep.

- **Withdrawal of approval.** FIPS 186-5 §4 withdrew DSA outright: *"This standard no longer approves
  the DSA for digital signature generation."* That is an **approval decision, not a work-factor
  statement**, and no claim fits it. So **DSA-2048 gets no row** even though the algorithm's own
  specifier has stopped approving it, and DSA appears in the table only at the point where its
  parameters fall below the strength floor, which is Claim 1. This is the closure doing exactly the
  work §2.1's three claims did when they excluded SSH and RDP.
- **Deprecation by a relying party.** A root programme sunsetting an algorithm, or the BR's closed
  permitted-list, is a **party's acceptance decision** about its own trust store. It names no
  property of the primitive. Refused here and again at §8.2 — a double lock, since it is the same
  refusal arriving through the claim set and through the owner definition.
- **An individual key's compromise.** The Debian OpenSSL weak-key set, ROCA-affected moduli, shared
  moduli from bad entropy. These are properties of a **particular key**, not of an
  `(algorithm, parameter)` pair, so no claim reaches them. They are also what would break release
  coupling — §7.2.
- **Quantum vulnerability.** Every algorithm in this table and every algorithm that would replace it
  today is vulnerable to a large-scale quantum computer. FIPS 186-5 §1 says so in its own words:
  *"Note that the algorithms in this standard are not expected to provide resistance to attacks from
  a large-scale quantum computer."* Admitting it as a claim would fire the rule on **every**
  certificate, which is not a table but a constant, and a rule that fires everywhere reads the world
  not at all.

#### 2.2.1 Why the exemption argument is wrong, since it is available and tempting

The available argument is: *this is a one-dimensional table — is this parameter below the floor? — so
there is nothing for a claim set to discriminate between, and gate 1 is inapplicable the way it is
inapplicable to a table about our own measurement.*

It is wrong, and the general statement it would license is what makes it worth refusing in writing.
**A table is exempt from gate 1 only where it is not about the world at all** — ADR-0032's line, `k`
and the availability window and the coverage threshold. Dimensionality is not the test. This table
is one-dimensional in its *output* and two-dimensional in its *grounds*: RSA-1024 and SHA-1 are both
"below the floor", but one is a work-factor deficit computed from a modulus length and the other is a
demonstrated collision, and **they are attested by different sentences in different documents for
different reasons**. Collapsing them into one undifferentiated *weak* is precisely how a laundered
opinion gets in — it is §2.7's shape, where an authoritative-looking document about magnitude stands
in for a position about correctness.

The proof that the derivation was not ceremonial is that it **changed the table**: DSA lost its
blanket row, and four other candidates were refused above.

### 2.3 Attestation — who owns a claim about a cryptographic primitive

This is the ticket's sharp half, and ADR-0032 flagged it as deliberately unpre-empted.

#### 2.3.1 The protocol owner is silent, and its silence is a citable position

RFC 5280 is PKIX and the IETF owns what its RFCs specify. It sets **no numeric floor anywhere**, and
its Security Considerations say why:

> "The binding between a key and certificate subject cannot be stronger than the cryptographic module
> implementation and algorithms used to generate the signature. Short key lengths or weak hash
> algorithms will limit the utility of a certificate. CAs are encouraged to note advances in
> cryptology so they can employ strong cryptographic techniques. In addition, CAs SHOULD decline to
> issue certificates to CAs or end entities that generate weak signatures."
> — [RFC 5280](https://www.rfc-editor.org/rfc/rfc5280.txt) §11

This is stronger than the *"security issues are not discussed in this memo"* non-statement §2.7 met
at RFC 1833, and it points somewhere. PKIX **names the property the rule is named for** — *short key
lengths*, *weak hash algorithms* — declines to supply a number, and refers the number to *"advances
in cryptology"*. The protocol owner has not forgotten to set a floor. It has said the floor is not
its to set, and has named the discipline whose it is.

That sentence is the hinge of everything below. It means the search for an owner is not a search for
a replacement authority. It is a search for **the party PKIX pointed at**.

#### 2.3.2 The clause §10.5 gains

`sensitive-ports.md` §10.5 defines *owner* as the party that **designed the protocol** or **authors
the reference implementation**, speaking about the thing it designed or wrote — and says outright
that **the artefact, not the party, is what the rule keys on**. Applied here, the artefact is neither
a protocol nor an implementation. It is a **cryptographic primitive**. §10.5's list of artefact kinds
is incomplete rather than wrong, and the repair is one clause:

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

This is a clarification, not a widening. It admits nobody §10.5 did not already admit — the IETF was
already an owner in §10.5's own words, and the distributor limb already says what the second
paragraph says. What it adds is the artefact class, so that the next session meeting a claim about an
algorithm rather than about a protocol does not have to rediscover it.

**The cost is stated rather than smoothed**, and it is not the cost the ticket expected. See §9: no
row is lost, and what is lost is modality on three of the five.

#### 2.3.3 The corroborators, and the fact that they point at the owner

The CA/B BR does not merely fail to be an owner. **It defers to one, in its own text.** BR §6.1.6
sources its RSA public-key checks to *"[Source: Section 5.3.3, NIST SP 800-89]"* and its ECDSA
validation routines to *"[Source: Sections 5.6.2.3.2 and 5.6.2.3.3, respectively, of NIST SP 800-56A:
Revision 2]"*. The body the ticket suspected of being the real owner cites NIST for its cryptographic
content, which is the cleanest possible corroboration of where the content's owner sits.

### 2.4 Determinacy: outside the domain, and the one move that would bring it inside

ADR-0032 ruled gate 3 the **surrogate gate**: it binds a table whose key is not the fact the rule
names, and v1 has exactly one surrogate. This table's key is `(algorithm, parameter)` read from
`subjectPublicKeyInfo`, and the digest read from `signatureAlgorithm`. Both are fields RFC 5280
defines to carry exactly these facts. There is no proxy: a modulus length **is** the modulus length,
and RFC 5280 §4.1.1.2 requires the `signatureAlgorithm` field to match the `signature` field inside
the `tbsCertificate`, so the OID is the certificate's own statement of what signed it rather than an
inference about it. **Outside the domain**, as ADR-0032 predicted.

**But the ticket was right to make this checked rather than assumed, because there is a real way in
and it looks like a simplification.** 2048-bit RSA and a 224-bit curve order both deliver about 112
bits of security. The numbers 2048 and 224 are not comparable and neither are the algorithms they
belong to. A session tidying this table into a **single bit-count threshold** — *keys under N bits
are weak* — would be keying on a **surrogate for security strength**, and gate 3 would come inside
the domain and fail immediately, because the same integer means two incompatible things on a modulus
and on a curve.

> **The table's key must be `(algorithm, parameter)` and never a bare bit count.** That is not
> presentation. It is the thing keeping gate 3 out of the domain, and it is the falsifiable statement
> this section leaves behind.

### 2.5 Release coupling is tested in §7, not assumed

ADR-0004's cadence test is — per [#18](https://github.com/winniel123/verge-asm/issues/18)'s
correction — a **ceiling on how often reference data may change**. The ticket predicted an easy pass
and asked for it to be shown. §7 shows it, and names the two things that would have failed it.

### 2.6 What was excluded from evidence entirely

- **Scan-population statistics on SHA-1 or RSA-1024 certificate share.** Frequency. *How many
  certificates still use SHA-1* is evidence that a problem persists, not evidence that the algorithm
  is weak. Not used anywhere in this note, and note that it is the easiest possible source to obtain
  here — CT logs make it a query.
- **The sunset dates themselves as grounds.** *Chrome stopped accepting SHA-1 certificates in 2017*
  is a fact about a shipped release and it is not a statement that SHA-1 is weak. Where a sunset date
  appears below it is context, never footing. The footing is always a sentence about the primitive.
- **Cryptanalysis papers cited directly.** SHAttered and the Wang MD5 collisions are the *reason*
  RFC 6151, RFC 8446 and SP 800-131A say what they say, and those documents cite them. Citing the
  papers over the standards would put this note in the position of reading cryptanalysis on its own
  authority, which is §2.3's laundering with a more respectable source.
- **RFC 3766 / BCP 86 as a floor.** The IETF's own key-strength document is an **estimation method**
  and a 2004-vintage discussion — *"the security of 1024-bit RSA moduli is doubtful"* — not a floor.
  Recorded because it is the closest the IETF comes to setting one, and it does not.

---

## 3. The table

**Five rows.** Two under Claim 2 on the signature digest, three under Claim 1 on the key. Every row
carries per-row footing, and §9 names which rows rest on the thinnest of it.

### 3.1 Claim 2 — the signature digest is not collision-resistant

The rows are keyed on **the digest**, not on the signature `AlgorithmIdentifier` OID. Three reasons,
and the third is load-bearing for §7:

1. It is how the owner phrases it — RFC 8446 §4.4.2.4 says *"any signature algorithm using an MD5
   hash"*, over the algorithm rather than over an OID list.
2. RSASSA-PSS carries its digest in the algorithm **parameters** rather than in the OID
   (`id-RSASSA-PSS`, 1.2.840.113549.1.1.10), so an OID-keyed table is blind to a PSS-with-SHA-1
   signature by construction.
3. A new `(digest, signature algorithm)` OID over an already-condemned digest is **inside an existing
   row with no table edit**, which is one of the two reasons §7 passes comfortably.

| Digest | Fires on | Footing | Tier |
|---|---|---|---|
| **MD5** | any certificate signature whose digest is MD5 — currently `md5WithRSAEncryption` (1.2.840.113549.1.1.4) | RFC 8446 §4.4.2.4: an endpoint **"MUST abort the handshake"**. RFC 6151 §2: *"MD5 must not be used for digital signatures"* | **Owner, unconditional, two independent owners** |
| **SHA-1** | any certificate signature whose digest is SHA-1 — currently `sha-1WithRSAEncryption` (1.2.840.113549.1.1.5), `id-dsa-with-sha1` (1.2.840.10040.4.3), `ecdsa-with-SHA1` (1.2.840.10045.4.1), and `id-RSASSA-PSS` parameterised with SHA-1 | RFC 8446 §4.2.3 and §4.4.2.4; NIST SP 800-131A Rev 2 Table 8; RFC 9155 §2 with RFC 5246 §7.4.2 for TLS 1.2 | **Owner, and the strongest verb is RECOMMENDED rather than MUST** |

### 3.2 Claim 1 — the key's work factor is below the specifier's floor

| Key algorithm | Floor | Fires when | Footing | Tier |
|---|---|---|---|---|
| **RSA** (`rsaEncryption`, 1.2.840.113549.1.1.1) | modulus `nlen` ≥ 2048 bits | `nlen` < 2048 | FIPS 186-5 §5.1; NIST SP 800-131A Rev 2 §3 and Table 2 | **Owner — NIST, Federal-scoped verb (§9)** |
| **ECDSA** (`id-ecPublicKey`, 1.2.840.10045.2.1) | `len(n)` ≥ 224 bits, `n` the order of the base point | `len(n)` < 224 — e.g. `secp192r1`, `secp224k1` at 224 is at the floor and does not fire | NIST SP 800-131A Rev 2 §3 and Table 2 | **Owner — NIST, Federal-scoped verb (§9)** |
| **DSA** (`id-dsa`, 1.2.840.10040.4.1) | `(L, N)` one of (2048, 224), (2048, 256), (3072, 256) | anything below — `L` < 2048 or `N` < 224 | NIST SP 800-131A Rev 2 Table 2 | **Owner — NIST, Federal-scoped verb (§9)** |

**The DSA row is the narrow one on purpose.** FIPS 186-5 stopped approving DSA for signature
generation at any size, and the table does **not** carry that, because §2.2's claim set admits a
work-factor claim and not a withdrawal of approval. A DSA-2048 certificate does not fire this rule.
That is a deliberate, checkable loss, and it is the price of the closed set.

### 3.3 The quotes behind the rows

Verified against retrieved bytes — the RFC text files from `rfc-editor.org` and the NIST PDFs from
`nvlpubs.nist.gov`, extracted locally. Each quote is given with enough surrounding text that a
conditional or an exception cannot hide in the truncation, per
[#46](https://github.com/winniel123/verge-asm/issues/46)'s finding.

#### MD5 — RFC 8446 §4.4.2.4, and the exception quoted with it

> "Any endpoint receiving any certificate which it would need to validate using any signature
> algorithm using an MD5 hash MUST abort the handshake with a "bad_certificate" alert. SHA-1 is
> deprecated, and it is RECOMMENDED that any endpoint receiving any certificate which it would need
> to validate using any signature algorithm using a SHA-1 hash abort the handshake with a
> "bad_certificate" alert. For clarity, this means that endpoints can accept these algorithms for
> certificates that are self-signed or are trust anchors."
> — [RFC 8446](https://www.rfc-editor.org/rfc/rfc8446.txt) §4.4.2.4

Three things at once, and all three are used. MD5 is a flat `MUST`. SHA-1 is a `RECOMMENDED`
preceded by a bare declarative — *"SHA-1 is deprecated"* — which is a position rather than a
recommendation. And the final sentence is the **owner's own carve-out for self-signed certificates
and trust anchors**, which §4.1 adopts rather than inventing.

#### MD5 — RFC 6151 §2, with the clause that is about something else

> "MD5 is no longer acceptable where collision resistance is required such as digital signatures. It
> is not urgent to stop using MD5 in other ways, such as HMAC-MD5; however, since MD5 must not be
> used for digital signatures, new protocol designs should not employ HMAC-MD5."
> — [RFC 6151](https://www.rfc-editor.org/rfc/rfc6151.txt) §2

The hedge — *"it is not urgent to stop using MD5 in other ways"* — is quoted deliberately, because
quoting *"MD5 must not be used for digital signatures"* alone would be the #46 error even though the
hedge does not touch the signature case. It is about HMAC-MD5 in record protection, which is not what
a certificate signature is.

#### SHA-1 — RFC 8446 §4.2.3, the reason and the sunset that never comes

> "Legacy algorithms: Indicates algorithms which are being deprecated because they use algorithms
> with known weaknesses, specifically SHA-1 which is used in this context with either (1) RSA using
> RSASSA-PKCS1-v1_5 or (2) ECDSA. These values refer solely to signatures which appear in
> certificates (see Section 4.4.2.2) and are not defined for use in signed TLS handshake messages,
> although they MAY appear in "signature_algorithms" and "signature_algorithms_cert" for backward
> compatibility with TLS 1.2. Endpoints SHOULD NOT negotiate these algorithms but are permitted to do
> so solely for backward compatibility. Clients offering these values MUST list them as the lowest
> priority (listed after all other algorithms in SignatureSchemeList). TLS 1.3 servers MUST NOT offer
> a SHA-1 signed certificate unless no valid certificate chain can be produced without it (see
> Section 4.4.2.2)."
> — [RFC 8446](https://www.rfc-editor.org/rfc/rfc8446.txt) §4.2.3

*"known weaknesses"* is the property, stated by the owner of TLS about signatures *"which appear in
certificates"*. The backward-compatibility permission is quoted with it and it is exactly what makes
the SHA-1 row's tier weaker than MD5's — §9.

#### SHA-1 in TLS 1.2 — a two-document chain, with its antecedent discharged

RFC 9155's normative clauses are all about **handshake** messages, and its Introduction points
elsewhere for certificates: *"Note that the CA/Browser Forum (CABF) has also deprecated use of SHA-1
for use in certificate signatures [CABF]."* Citing RFC 9155 for a certificate row directly would be
the out-of-context error. The chain that does reach certificates has two links and both are `MUST`:

> "Clients MUST include the signature_algorithms extension. Clients MUST NOT include MD5 and SHA-1 in
> this extension."
> — [RFC 9155](https://www.rfc-editor.org/rfc/rfc9155.txt) §2

> "If the client provided a "signature_algorithms" extension, then all certificates provided by the
> server MUST be signed by a hash/signature algorithm pair that appears in that extension."
> — [RFC 5246](https://www.rfc-editor.org/rfc/rfc5246.txt) §7.4.2

RFC 5246's requirement is conditional on the client sending the extension. RFC 9155 — which updates
RFC 5246 — **discharges the antecedent** by making it mandatory. The conclusion is the IETF's, not
ours: in TLS 1.2 with a conforming client, no certificate the server provides may carry an MD5 or
SHA-1 signature.

#### SHA-1 — NIST SP 800-131A Rev 2, with its exception

Table 8 gives SHA-1 / *Digital signature generation* / **"Disallowed, except where specifically
allowed by NIST protocol-specific guidance."** The prose beneath it:

> "SHA-1 may only be used for digital signature generation where specifically allowed by NIST
> protocol-specific guidance. For all other applications, SHA-1 is disallowed for digital signature
> generation."
>
> "When used for digital signature verification, SHA-1 is allowed for legacy use."
> — [NIST SP 800-131A Rev. 2](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-131Ar2.pdf) §9

**The verification cell is quoted because it is the trap.** A reader could reason that we are
*verifying* a signature and therefore SHA-1 is *"legacy use"* rather than *"disallowed"*. That is the
wrong cell. The row's claim is about the certificate that was **generated** — a CA applied
cryptographic protection using SHA-1 — and *"Disallowed means that the algorithm or key length is no
longer allowed for applying cryptographic protection."* The exception is inward-facing (NIST's own
protocol-specific guidance) and reaches no WebPKI certificate.

#### The key floors — NIST SP 800-131A Rev 2 §3, and the sentence without the qualifier

> "Private-key lengths providing less than 112 bits of security shall not be used to generate digital
> signatures."
>
> "ECDSA and EdDSA: The security strength provided by an elliptic-curve-based signature algorithm is
> no greater than 1/2 of the length of the domain parameter n. Therefore, the length of n shall be at
> least 224 bits to meet the minimum security-strength requirement of 112 bits for Federal Government
> use."
>
> "RSA: The length of the modulus n shall be 2048 bits or more to meet the minimum security-strength
> requirement of 112 bits for Federal Government use."
> — [NIST SP 800-131A Rev. 2](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-131Ar2.pdf) §3

Table 2's *Digital Signature Generation* row is the enumeration these sentences summarise —
`< 112 bits of security strength`, marked **Disallowed**:

- DSA `(L, N)` ≠ (2048, 224), (2048, 256) or (3072, 256)
- ECDSA `len(n)` < 224
- RSA `len(n)` < 2048

The elliptic-curve sentence is worth pausing on: the first half is **arithmetic** and carries no
qualifier, and the second half is the **threshold** and carries *"for Federal Government use"*. That
split is the whole of §9.

#### RSA — FIPS 186-5 §5.1, without a scope qualifier

> "This standard specifies the use of a modulus whose bit length is an even integer and greater than
> or equal to 2048 bits. Furthermore, this standard specifies that p and q be of the same bit length
> - namely, half the bit length of n."
> — [FIPS 186-5](https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.186-5.pdf) §5.1

This is the standard that **specifies RSA digital signatures**, saying what it specifies. It is the
best-footed sentence behind any key row and it is why the RSA row is the strongest of the three.

#### NIST's own scope, quoted because it is the disclosure

> "**Approval status** — Used to designate usage by the U.S. Federal Government."
>
> "**Shall** — A requirement for Federal Government use. Note that shall may be coupled with not to
> become shall not."
>
> "**Disallowed** means that the algorithm or key length is no longer allowed for applying
> cryptographic protection."
> — [NIST SP 800-131A Rev. 2](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-131Ar2.pdf) §1.2.2

NIST discloses its own scope in its definitions section, and this note reads it rather than around
it. §9 is what follows.

One further clause of §1.2.2 was checked and does **not** reach these rows: *"If a user determines
that the risk is unacceptable, then the algorithm or key length is considered disallowed from the
perspective of that user"* applies, in its own sentence, to the terms **deprecated** and **legacy
use**. Every cell this table relies on reads **Disallowed**.

### 3.4 Ed25519 needs no row, and that is a fact about the algorithm

The ticket asked whether Ed25519 needs a row at all. **No**, and the reason generalises.

A floor row cuts a **free parameter**. RSA has one (the modulus length), ECDSA has one (the curve),
DSA has two. EdDSA has none: the curve, the base point order, the private-key length and the internal
hash are all fixed by the algorithm's own definition — *"For Ed25519, SHA-512 shall be used"* — so
there is no size for a floor to be below. FIPS 186-5 §7.1 records the resulting strength:

> "It is noted that Ed25519 is intended to provide approximately 128-bits of security, and Ed448 is
> intended to provide approximately 224-bits of security."
> — [FIPS 186-5](https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.186-5.pdf) §7.1

Both clear the 112-bit floor by construction, and SP 800-131A Table 2's *"ECDSA or EdDSA: len(n) ≥
224"* cell covers them without a separate line. `id-Ed25519` (1.3.101.112) and `id-Ed448`
(1.3.101.113) are therefore not weak, and a certificate carrying one does not fire this rule.

> **The general statement: an algorithm with no free size parameter cannot have a floor row.** Its
> absence from the table is a property of the algorithm and not a gap in our coverage, and a later
> session should not read the blank as work left undone.

Note in passing that the CA/B BR does not permit Ed25519 in a publicly-trusted TLS certificate at all
— its permitted lists are RSA and ECDSA P-256/P-384/P-521, closed by *"No other algorithms or key
sizes are permitted."* That is a fact about **issuance policy**, it has nothing to do with weakness,
and it is exactly the artefact confusion §8.2 refuses. It appears here as a caution, not as grounds.

---

## 4. What the rule reads

ADR-0024 fixed the rule's **domain**: `certificate` is `Presented`, and `NoTLS` is outside it. What
had never been written is the **predicate**, which is this ticket's charter.

> **`certificate-weak-key-or-signature` fires on a `Presented` chain where any certificate in the
> chain either**
>
> **(a) carries a `subjectPublicKeyInfo` whose `(algorithm, parameter)` is below the floor in §3.2, or**
>
> **(b) is signed with an algorithm whose digest is in §3.1 — excepting a certificate that is
> self-signed.**

### 4.1 The chain, and why the two limbs scope differently

`CONTEXT.md` holds the `certificate` facet as the **ordered chain of fingerprints, leaf first**, so
the predicate ranges over a chain rather than a leaf. Reading the leaf alone would miss a 1024-bit
intermediate, which is the WebPKI's actual historical weakness and not a hypothetical one.

The two limbs scope differently, and the asymmetry is the owner's reasoning rather than ours:

> "The signatures on certificates that are self-signed or certificates that are trust anchors are not
> validated, since they begin a certification path (see [RFC5280], Section 3.2). A certificate that
> begins a certification path MAY use a signature algorithm that is not advertised as being supported
> in the "signature_algorithms" extension."
> — [RFC 8446](https://www.rfc-editor.org/rfc/rfc8446.txt) §4.2.3

A self-signed certificate's signature is verified with its own key and no relying party depends on
it, so a SHA-1 self-signature carries no forgeable binding — **the signature limb skips it**. Its
**key** is depended on, by everyone who trusts it, so **the key limb does not skip it**. Splitting on
the owner's stated reason rather than on the certificate's position in the chain is what makes the
scope checkable.

Two riders. *Self-signed* here means what `certificate-self-signed` reads — issuer ≡ subject with a
verifying self-signature — so the two rules read the same fact and cannot disagree about it. And
because we cannot know which certificate a given relying party treats as its **trust anchor**, the
predicate keys on **self-signed**, which is on the wire, rather than on **trust anchor**, which is
not. RFC 8446 names both. Only one of them is observable from a handshake.

### 4.2 An algorithm the table does not name is not weak

The table is a **deny list**. An `(algorithm, parameter)` pair it does not name is not below a floor
it does not set, so the rule is **false**, not `not-evaluable`.

The alternative — treating an unrecognised algorithm OID as unevaluable — was considered and refused
on the model's own grounds. `not-evaluable` means the evidence is absent. Here the field was read and
the question *is this among the parameters our table names as below a floor* was answered. Worse, it
would make the rule's output a function of **our table's coverage** rather than of the world: the day
post-quantum certificates ship, every one of them would flip an estate to `not-evaluable` with
nothing in the world having changed — the same hazard `CONTEXT.md` names when it keeps the negotiated
version out of the `certificate` facet because it *"would move estate-wide on a library upgrade"*.

An allow list would have the mirror defect and a worse one: it would **fire** on every novel
algorithm, asserting weakness from ignorance.

### 4.3 The routes to `not-evaluable`

1. **Outside the domain** — `NoTLS`, per ADR-0024. Rendered as outside the domain rather than as
   `not-evaluable`, on ADR-0024's three-way treatment.
2. **The chain was not read** — the `certificate` facet rides whichever port tier ran the exchange
   (`CONTEXT.md`), so a `Service` on a slower tier is unmeasured between probes. Inherited from the
   facet, not specific to this table.

There is no third route. In particular there is no route through the table itself, which is §4.2's
whole point: **the table can never be the reason a certificate is unevaluable.**

---

## 5. The negative space — excluded, with reasons

A curated table is judged on what it refuses.

| Excluded | Why |
|---|---|
| **MD2 and MD4 signatures** | The retiring documents are Informational and hedge their own results: RFC 6149 §2 says MD2 *"has been shown to not be collision-free … albeit successful collision attacks for properly implemented MD2 are not that damaging"*, and its Security Considerations put the best collision attack at 2^63.3 operations with 2^50 memory. No owner sentence prohibits MD2 in a certificate, and RFC 8446 §4.4.2.4 names MD5 and SHA-1 and not these. Excluded on §2.7's ground: **we could not find anyone entitled to say it** |
| **RSA public exponent** (e = 3, or e outside 2^16+1 … 2^256−1) | The BR requires a range and cites SP 800-89 for it, but no owner calls a small exponent **weak** — RFC 8017 supports e = 3 and FIPS 186-5 imposes a range on **generation**, which is an issuance requirement rather than a strength claim. No claim in §2.2 fits |
| **RSASSA-PSS parameter defects** (salt length, MGF mismatch) | Encoding conformance, not work factor and not collision resistance. #31's instrument, if any |
| **Known-compromised individual keys** — Debian OpenSSL, ROCA, shared moduli | Not a property of an `(algorithm, parameter)` pair, so no claim reaches them (§2.2), and a growing blocklist is what breaks release coupling (§7.2). This is the single most tempting addition and it is refused twice, independently |
| **Certificate lifetime, key reuse, missing CT** | Different facts, and two of them are other rules. The rule is named for the key and the signature |
| **A "should be 3072 by 2031" second threshold** | Severity smuggled in as a second row. §6.3 |
| **Trust-store membership** | ADR-0032 already ruled that v1 ships no root bundle, and the rule is named `weak`, not `untrusted` |

---

## 6. Three things the constraints forbade, and what happened when they were tried

### 6.1 Frequency, refused where it was easiest to reach

This is the table where a frequency source is nearest to hand: certificate transparency makes *what
share of certificates carry a SHA-1 signature* a query rather than a research problem, and the answer
would be dramatic. It is not used, and it could not have carried a row, because it answers *how many*
and the table asserts *this is weak*. §2.6 records the refusal because the temptation is structural
rather than incidental.

### 6.2 The sunset dates are context, never grounds

Every WebPKI body has a SHA-1 sunset date and they are all citable. Every one of them is a fact about
a **shipped release** — the shape §10.4 governs, over the shipping party's own artefact. They appear
in §8 as corroboration and nowhere in §3.3.

### 6.3 No severity, and the specific door it tried to come through

The threshold that wanted to exist is **RSA 3072 / 128-bit security**. NIST SP 800-57 Part 1 puts the
112-bit level's usable life to 2030, and it is genuinely true that a 2048-bit key is weaker than a
3072-bit one. Adding it would have produced *below 2048 = weak*, *2048–3071 = weakening*, which is a
ranked family wearing a threshold's clothes and is exactly what the ticket forbade.

It is refused, and the refusal is not merely obedience: a second threshold would also have made the
table's verdict move on the **calendar** rather than on a release, which §7.3 shows is independently
disqualifying. The two constraints agree, which is some evidence both are right.

---

## 7. Release coupling, tested

ADR-0004 requires reference data whose change rate a release cadence can absorb. The test is a
ceiling on how often the table may change.

### 7.1 The measured cadence: about one edit per row per decade

| Event | Date |
|---|---|
| Wang et al. publish MD5 collisions | 2004 |
| RFC 6151 records the position for MD5 | 2011 |
| NIST disallows SHA-1 for signature generation | end of 2013 (per RFC 9155 §1) |
| NIST moves the RSA floor to 2048 | 2011–2013 transition |
| SHAttered — the first full SHA-1 collision | 2017 |
| RFC 8446 states the certificate rule for MD5 and SHA-1 | 2018 |
| RFC 9155 closes the TLS 1.2 handshake path | 2021 |
| FIPS 186-5 withdraws DSA and restates the RSA floor | 2023 |
| Next scheduled move — the 112-bit level's end of life | ~2030 |

**Nine changes in twenty-two years across five rows, and the next is scheduled six years out.** A
release cadence measured in weeks has three orders of magnitude of headroom. This passes more easily
than the sensitive-port list, whose sources moved twice during the writing of `sensitive-ports.md`.

Two structural properties make it easier still. Rows are keyed on the **primitive**, so a new OID over
a condemned digest needs no edit (§3.1). And the floors are **thresholds** rather than enumerations,
so a new curve or a new modulus length is inside or outside an existing row without one.

### 7.2 The near-miss that would have failed it, and is refused

A **known-compromised-key blocklist** — Debian OpenSSL's 2008 weak keys, ROCA-affected moduli, shared
moduli harvested from bad entropy — is the natural next thing a reader will want in this table. It
would fail ADR-0004 outright: such a list grows continuously, its value comes from being current, and
a table we would want to push out of band is #31's **signature database** by definition. Refused at
§2.2 on the claim set and again here on cadence, and the two refusals are independent.

### 7.3 The second near-miss: NIST's dated transitions must not be encoded as dates

SP 800-131A and SP 800-57 express several transitions as calendar events — *Disallowed after 2023*,
and the 112-bit level's end of life around 2030. Encoding any of these as a **date comparison in the
predicate** would be a quiet violation: the rule's output would move at midnight on a date with **no
version bump and no release**, making two evaluations non-comparable while
[ADR-0008](../adr/0008-derivation-versions-move-on-content.md) reported nothing had changed. That is
release-coupling defeated from the inside.

> **The table encodes floors as they stand today. A scheduled transition is a scheduled *edit* — one
> `Break`, one rule, one cadence — and never a live date in the predicate.**

---

## 8. The three routes, weighed

The ticket set out three routes and flagged (c) as the most interesting and the most likely to
overreach. The retrieval bears the second half of that out, and for a reason the ticket could not
have anticipated.

### 8.1 Route (a) — the root programmes **are** owners, because they collectively author the WebPKI

**Refused.** The argument is that *owner* should key on the **trust ecosystem** rather than on the
protocol, because the WebPKI is a thing the programmes collectively author and PKIX is not.

It fails on what it would do to the definition. §10.5 keys on an **artefact** and says so. Route (a)
would key on **standing** — on being the party whose acceptance decides whether a certificate works.
That is a market position, not a thing anyone designed, and once *owner* can attach to standing there
is no principled way to keep out the sources §2.3 was built to exclude. AWS's standing over EC2 is
the same shape, and §2.3's measured finding is that AWS's two products **contradict each other about
port 25**.

**The route's own premise was then measured, and it does not hold.** *Collectively author* implies
four bodies with positions. Two of the four have none:

- **Chrome Root Program Policy v1.8** contains no occurrence of `RSA`, `ECDSA`, `key size`,
  `modulus`, `curve`, `SHA-1` or `digest`. It incorporates the BR by reference — *"Chrome Root
  Program Participants … MUST adhere to the latest version of the … Baseline Requirements"* (§1.1.1).
- **Apple's Root Certificate Program** states key sizes only for S/MIME, and otherwise delegates:
  *"TLS CA providers must constantly maintain compliance with the current version of the CA/Browser
  Forum Baseline Requirements"* (§1.3). It never mentions SHA-1.

And the two that do state their own text **contradict each other and the specifiers**, on the very
first case, in §2.3's measured AWS pattern:

- **Mozilla Root Store Policy v3.1** still enumerates `RSASSA-PKCS1-v1_5 with SHA-1` as a permitted
  signature algorithm for root and intermediate RSA signing keys (§5.1.1), with carve-outs in §5.1.3
  permitting SHA-1 over some end-entity and duplicate-intermediate certificates.
- **Microsoft's Root Program Requirements** tell CAs not to use an algorithm NIST specifies and
  SP 800-131A rates acceptable: *"The Microsoft Trusted Root Program recommends that ECC/ECDSA
  certificates shouldn't be issued to subscribers due to this known incompatibility and risk"* —
  where the risk named is that *"Signatures using elliptical curve cryptography (ECC), such as ECDSA,
  aren't supported in Windows and newer Windows security features."* That is a **compatibility**
  position about Microsoft's own platform, arriving in the same table as a key-size floor and
  indistinguishable from it if the programme is read as an owner.

§10.5's rider still governs and is worth restating, because the measurement must not become the
ground: *"a session that finds a consistent distributor and reads the door as open has read the wrong
sentence."* The contradictions are evidence the line is right. They are not the reason for it.

Route (a) also proves too much in a checkable direction. If collective authorship of an ecosystem
confers ownership of the primitives inside it, then the programmes own SHA-256 as much as SHA-1, and
a future programme decision to sunset an algorithm would become an **attestation that it is weak**.
It would not be. It would be a scheduling decision, and §6.2 already refuses those as grounds.

### 8.2 Route (c) — the BR is a shipped default under §2.2's third form

**Refused, and this is the one worth the most words**, because the argument is good and its refusal
is already written in §10.5.

The argument: §2.2's third form admits *the project's shipped default*. §10.4 rules that a shipped
default attests **where it restricts**. A floor is unambiguously a restriction. The BR is enforced in
code by the parties who accept certificates. Therefore the BR admits the rows.

Each step is sound and the conclusion does not follow, because §10.4 answers *whether* a default
attests and never *what about*. PostgreSQL's `listen_addresses = localhost` admits the 5432 row
because the artefact PostgreSQL restricted — exposing PostgreSQL to the network — **is what the row
is about**. A browser's verifier refusing SHA-1 restricts **the browser's own acceptance**, and the
row is about **SHA-1's collision resistance**. The artefacts do not match, and §10.5 says in its own
words: *"The artefact, not the party, is what the rule keys on."* Route (c) is refused by the
sentence §10.5 was already carrying.

**And the retrieval turns a principled refusal into a lucky one.** Had route (c) been taken, the
table would have been founded on the CA/Browser Forum Baseline Requirements v2.2.9 (adopted
2026-07-02, effective 2026-08-06), and three measured properties of that document would have come
with it:

- **The BR has no position on MD5 whatsoever.** The string `MD5` occurs **zero times** in the
  document. MD5 is excluded only by omission from a closed permitted list — which is the *absence of
  an act*, and §10.4's one-way rule says an absence attests nothing. **Half the signature axis would
  have had no footing at all.**
- **The BR's SHA-1 prohibition carries a live exception.** §7.1.3.2.1 permits `RSASSA-PKCS1-v1_5 with
  SHA-1` under a five-condition carve-out *"Until 2026-09-15"*, for re-issued Root CA and
  cross-certificates. A row founded on it would have been founded on a conditional that expires four
  weeks after this note was written — the #46 hazard in its purest form, and unrecoverable, because
  the conditional is the source's and not the quoter's.
- **The BR's ECDSA restriction is not a weakness claim.** §6.1.5 permits only P-256, P-384 and P-521
  and closes with *"No other algorithms or key sizes are permitted."* That excludes P-224 from
  **issuance**. Nowhere does it say P-224 is weak, and NIST — which specified the curve — says the
  opposite, at `len(n) ≥ 224`. Route (c) would have set the ECDSA floor at 256 on a document that
  never made the claim.

So route (c) would have cost the table its MD5 row, put a September sunset inside a release-coupled
artefact, and moved the ECDSA floor on an issuance rule mistaken for a strength claim. **The refusal
is on §10.5's artefact test. The measurement is what turns it from a close call into a clear one.**

### 8.3 Route (b) — the rule rests on the primitives' specifiers, and the coverage loss is measured

**Adopted**, with §2.3.2's clause naming the artefact class. The ticket framed (b) as *"accepting
whatever coverage that loses"*, and the honest measurement is:

**No row is lost.** All five rows are attested by an owner under §10.5 as clarified — the IETF for
the two signature rows, NIST for the three key rows — and §2.3.3 records that the CA/B BR **cites
NIST** for its own cryptographic content.

**What is lost is modality on three rows.** The IETF's certificate statements are unconditional and
unscoped. NIST's key-size statements are scoped, by NIST's own definitions section, to U.S. Federal
Government use. That is a real weakening and it is §9.

**And on the signature axis, route (b) is strictly stronger than (c)**, per §8.2. That result was not
predictable from the ticket and it is the reason this note reports the routes in this order.

---

## 9. The weak tier, disclosed

Per ADR-0032 the disclosure binds every instrument's own document, in its stronger form: a disclosed
weakness names the retrieval that would resolve it.

> **WITHDRAWN as a statement of the current tiering, by
> [#102](https://github.com/winniel123/verge-asm/issues/102) per
> [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).**
> §9.2 named a retrieval; **§13 ([#73](https://github.com/winniel123/verge-asm/issues/73)) ran it and
> it came back positive.** §13.5 supersedes the table below and says so — *"§9 stands unrewritten;
> this is what it should read"* — but that is a pointer at the **superseding** site, and it reaches
> only a reader who already suspects the table is stale. The withdrawal is owed **here**.
>
> **What no longer holds:** the weak tier is **not** three rows. RSA `nlen` < 2048 is **out of the
> weak tier** on four unscoped IETF statements; ECDSA `len(n)` < 224 is out of the *scope*-weak tier
> and into SHA-1's; the weak tier is **one limb of one row** — DSA's `N ≥ 224`. **Read §13.5 for the
> table that is current.** The table below is left standing per the name-and-withdraw convention,
> because §9.1's scope argument is the evidence for why §13's corpus had to be widened.

| Row | Footing | Tier |
|---|---|---|
| **MD5 signature** | RFC 8446 §4.4.2.4 `MUST abort`; RFC 6151 §2 *"must not be used for digital signatures"* | Strongest in the table. Two owners, unconditional, unscoped |
| **SHA-1 signature** | RFC 8446 §4.2.3 *"known weaknesses"* + §4.4.2.4 `RECOMMENDED`; SP 800-131A Table 8 `Disallowed`; RFC 9155 §2 + RFC 5246 §7.4.2 for TLS 1.2 | Strong, with a named softness: the owner's verb for the certificate case is `RECOMMENDED`, and §4.2.3 expressly permits SHA-1 *"solely for backward compatibility"* |
| **RSA `nlen` < 2048** | FIPS 186-5 §5.1 (unqualified); SP 800-131A §3 (Federal-scoped) | **Weak tier** — see below. Strongest of the three key rows, because FIPS 186-5's sentence carries no scope qualifier |
| **ECDSA `len(n)` < 224** | SP 800-131A §3 and Table 2 only | **Weak tier** |
| **DSA below (2048, 224)** | SP 800-131A Table 2 only | **Weak tier**, and additionally the row least likely ever to fire |

### 9.1 The weakness, stated precisely

**The three key rows rest on a single owner, and that owner scopes its own normative verbs.** NIST SP
800-131A Rev 2 §1.2.2 defines *Shall* as *"A requirement for Federal Government use"* and *Approval
status* as *"Used to designate usage by the U.S. Federal Government."* So the sentence that supplies
each key floor is, by its author's own definition, a statement about approval for U.S. federal use —
not a universal statement that the parameter is weak.

The split runs **inside a single sentence**, which is why this is a disclosure rather than a
withdrawal. *"The security strength provided by an elliptic-curve-based signature algorithm is no
greater than 1/2 of the length of the domain parameter n"* is arithmetic and carries no qualifier at
all. *"Therefore, the length of n shall be at least 224 bits to meet the minimum security-strength
requirement of 112 bits for Federal Government use"* is the threshold and carries it. **The strength
computation is universal. The number 112 is federal.** Somebody has to say that 112 bits is too
little, and the only party saying it about these primitives is saying it to federal agencies.

This is `161/udp`'s shape one instrument across — a row whose owner statement does not quite reach
the claim's scope — and it is disclosed on the same terms and for the same reason: the alternative was
a table that condemns SHA-1 and says nothing about a 512-bit RSA key, which no reader would believe.

### 9.2 The retrieval that would resolve it

**Named, and it is a retrieval rather than a re-reading**, per #37's precedent that a row moves on
retrieval and never on re-reading text already held.

> **THE RETRIEVAL HAS BEEN PERFORMED AND IT CAME BACK POSITIVE — §13
> ([#73](https://github.com/winniel123/verge-asm/issues/73)). Limb 1 below is WITHDRAWN as an
> outstanding obligation** by [#102](https://github.com/winniel123/verge-asm/issues/102) per
> [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
> Nobody owes this search. §13.1 enumerates the corpus it is now established over, and §13.2–§13.5
> record what was found: the IETF states an unscoped floor in **three document classes**, one of them
> in an appendix of a document limb 1 cites as setting none. **The criterion below has been met**, for
> two of the three rows, and they have moved — so a session reading this subsection alone would both
> re-run a finished retrieval and ship a weak tier three times too wide. Limb 2 (SP 800-131A Rev. 3)
> **stands**: it is a future publication and §13 did not resolve it.

1. **An IETF or other unscoped statement of a certificate key-size floor.** This note searched for
   one and found none: RFC 5280 §11 declines to set a floor by design (§2.3.1), RFC 8446 and RFC 5246
   set none, and BCP 86 is an estimation method rather than a floor (§2.6). The negative is
   established over those four documents and **not** over the whole IETF corpus — the LAMPS working
   group's current output was not read, and a session with time should read it.
2. **NIST SP 800-131A Rev. 3 when it goes final.** Rev 2 (March 2019) remains the current final
   publication. Rev 3 exists only as an initial public draft whose comment period closed on
   2024-12-04. It will restate these floors and may move the 112-bit level. It will **not** resolve
   the scope question, since it will be a NIST publication addressed the same way — recorded so that
   nobody mistakes it for the resolution.

**The criterion that would change the verdict**, in one line: an unscoped statement by the specifier
of RSA or of the NIST curves that a parameter below the floor is insecure — as opposed to
unapproved — moves these three rows out of the weak tier. Its continued absence does **not** remove
the rows: they ship, disclosed, exactly as `5432/tcp` does, because gate 2 has never been a shipping
gate.

### 9.3 Which kind of weak row this is

§10.6 distinguishes two kinds of weak row and forbids collapsing them: one is **watched** (a shipped
default that could silently flip permissive), the other is **chased** (a corroborator standing where
an owner should).

**These three are neither, and that is a third kind.** No shipped default is involved, so silent
de-attestation cannot reach them. And the owner is not missing — NIST is an owner under §2.3.2 and
speaks directly about the primitives it specified. What is thin is the **modality** of what it says.
That is a **scope** weakness, and it behaves differently from both: it cannot flip silently, and it
cannot be chased by finding the right party, because the right party has already spoken. It resolves
only if a **second** owner speaks unscoped, which is why §9.2's retrieval is a search of a corpus
rather than a request to a party.

> **Read by [#125](https://github.com/winniel123/verge-asm/issues/125), and the consequence for this
> table is a zero.** ADR-0032 §8's watch is no longer a list of weak rows; it is a **gate** over what
> is closed plus a **queue** keyed on the **revision act**
> ([ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)). **This table
> contributes no queue items**, and this section is why: *"no shipped default is involved, so silent
> de-attestation cannot reach them."* Its one queue-shaped item is a **cure-availability** one —
> SP 800-131A Rev. 3 going final (§9.2) — and it sits at **rung 5**, a specification changing only by
> a new document with a new number, which is the bottom of the queue because it is announced. The
> three-pile taxonomy above is untouched: #125 keys on the **act**, not on the **cause**, so *watched*,
> *chased* and *scope* remain the causes this section calls them.

---

## 10. Corroboration — the WebPKI bodies, recorded and not relied on

Under §2.3 these corroborate and can carry no row alone. They are recorded because a reader will
expect them, because §8.2 needs them measured, and because a corroborator's disagreement with an
owner is itself information.

| Body | On the signature axis | On the key axis |
|---|---|---|
| **CA/B Forum BR v2.2.9** (adopted 2026-07-02, effective 2026-08-06) | Permitted list is SHA-256/384/512 only, closed by *"No other encodings are permitted for these fields"* (§7.1.3.2). **No mention of MD5 anywhere.** SHA-1 carve-out live *"Until 2026-09-15"* (§7.1.3.2.1) | RSA modulus *"at least 2048 bits"*; ECDSA restricted to P-256/P-384/P-521 (§6.1.5). Sources its own checks to NIST SP 800-89 and SP 800-56A (§6.1.6) |
| **Mozilla Root Store Policy v3.1** (effective 2026-07-01) | SHA-1 **still enumerated as permitted** for root and intermediate RSA signing keys (§5.1.1), restricted but not removed by §5.1.3. ECDSA paired with SHA-2 only (§5.1.2) | *"RSA keys whose modulus size in bits is divisible by 8, and is at least 2048 bits"*, or ECDSA on P-256/P-384/P-521 (§5.1). EdDSA permitted only with `id-kp-emailProtection`, *"Otherwise, EdDSA keys MUST NOT be included"* |
| **Microsoft Root Program Requirements** | Digest algorithms *"SHA2 (SHA256, SHA384, SHA512)"* (§3.1.20). No SHA-1 clause at all | RSA 2048, or 4096 for code-signing roots; ECC on P-256/P-384/P-521 — **and recommends against ECDSA entirely** on Windows compatibility grounds (§3.1.20 note). *"may not issue new 1024-bit RSA certificates"* (§3.1.9) |
| **Chrome Root Program Policy v1.8** | **Not stated** — no algorithm text of any kind; incorporates the BR (§1.1.1) | **Not stated** |
| **Apple Root Certificate Program** | **Not stated** for TLS; SHA-1 never mentioned | **Not stated** for TLS; RSA 2048 / P-256/384/521 for **S/MIME only** (§2.3). Incorporates the BR (§1.3) |

**Where the corroborators and the owner disagree, the disagreement is recorded and the owner
governs.** Three divergences, and each is an **issuance or compatibility** rule mistaken for a
strength claim if read carelessly:

- **ECDSA curves.** The BR, Mozilla and Microsoft permit three curves. This table's floor is
  `len(n) ≥ 224`, so a P-224 certificate fires no rule here while being unissuable under all three.
  That is correct and deliberate — a certificate outside a permitted list is **non-conforming**,
  which is a different fact from weak and is not this rule's name.
- **SHA-1.** Mozilla still lists it as permitted for some signing. This table condemns it flatly on
  the IETF's and NIST's sentences. **The owner is stricter than the corroborator**, which is the
  direction that ought to reassure.
- **ECDSA at all.** Microsoft recommends against it. NIST specifies it and rates it acceptable. A
  table built on the programmes would have had to arbitrate that, and would have had no principle to
  do it with.

**Two of the five corroborators state nothing to corroborate with**, and that is recorded rather than
elided: Chrome and Apple delegate to the BR, so the WebPKI's "collective" position on algorithms is,
on measurement, one document plus two partial restatements of it.

---

## 11. Open questions

1. **Is `certificate-not-conforming-to-issuance-policy` a rule v1 wants?** §10's divergence is the
   only place this table declines to say something a reader might expect, and the reason is that
   *non-conforming* and *weak* are different facts. Whether the first deserves a rule is a v1.1
   question and would need its own table under gate 2 — a large one, since the BR is a long document.
2. **Does the key limb reading a served root produce a firing the operator cannot act on?** §4.1 keys
   the signature limb on self-signed and the key limb on everything. A server that sends its root in
   the chain and that root carrying a 1024-bit key would fire. The operator **can** act — they
   control what their server sends — but nobody has measured how often this arises.
3. **Who watches these five rows?** The map's standing curation patch. This table's watch is cheaper
   than the port list's: §7.1's cadence means the correct posture is to re-read SP 800-131A when a
   revision goes final and otherwise to do nothing.

---

## 12. Retrieval hazards met, recorded per `sensitive-ports.md` §9.5

- **RFC 9155 does not say what its title suggests, and it says so itself.** Every normative clause is
  about a handshake message — `signature_algorithms`, `CertificateRequest`, `ServerKeyExchange`,
  `CertificateVerify` — and its Introduction hands the certificate case to the CA/Browser Forum. A
  session citing "RFC 9155 deprecates SHA-1" for a certificate row would be quoting a document that
  points away from the row. The path that does reach certificates is the §3.3 chain through RFC 5246
  §7.4.2, and its antecedent has to be discharged explicitly.
- **NIST's SHA-1 cell has an exception clause and the naked sentence circulates without it.** Table 8
  reads *"Disallowed, except where specifically allowed by NIST protocol-specific guidance."* Quoting
  *"SHA-1 is disallowed"* alone is accurate and incomplete in exactly #46's way.
- **The wrong NIST cell is the easier one to reach.** SHA-1 is *Legacy use* for signature
  **verification** and *Disallowed* for signature **generation**, and a scanner naturally thinks of
  itself as verifying. The row is about the CA's act, not ours — §3.3.
- **NIST PDFs do not survive naive text extraction.** `pdftotext` without `-layout` interleaves the
  vertical sidebar (*"This publication is available free of charge from…"*) into the body text and
  fragments the tables. Every NIST quote here was taken from a `-layout` extraction and re-read
  against the surrounding rows.
- **The CA/B BR is too large for a single HTML fetch.** `cabforum.org`'s rendering truncates partway
  through §3.2.2.5. The document was read from the Forum's own `cabforum/servercert` repository copy
  of `docs/BR.md`, whose `main` HEAD at retrieval was the v2.2.9 release commit with no post-release
  commits — a retrieval of the published bytes rather than of a rendering of them.
- **Two root-programme documents are not where the obvious URL points.** `mozilla.github.io/pkipolicy`
  returns 404. The policy is served from `mozilla.org` and mastered at `mozilla/pkipolicy` on GitHub.
  Microsoft's `learn.microsoft.com` program-requirements page carries an explicit superseded notice —
  *"This page has been superceded. The program technical requirements can be found here:
  https://github.com/TrustedRootProgram/Program-Requirements"* — while still rendering a full copy of
  the requirements, which is the exact shape of a stale page that reads as current. Its algorithm
  table matched the live one on this retrieval. That it did is luck, not a reason to cite it.
- **Microsoft's own document disagrees with itself about its version.** The title line reads v1.1 and
  the changelog's latest row reads v1.2, effective 2026-05-20. Recorded so a later session does not
  read one of the two as authoritative without noticing the other.
- **The ECDSA floor invites a units error.** `len(n)` is the bit length of the base point's **order**,
  not of the curve's field prime and not of the encoded public key. They coincide for the P-curves and
  do not in general.

---

## 13. Amendment — [#73](https://github.com/winniel123/verge-asm/issues/73): the corpus search was run, and it came back positive

**Nothing above is rewritten.** §9.2 named a retrieval and this section is that retrieval performed,
recorded per the name-and-withdraw convention. Where a statement above is now stale it is named here
and left standing where it is.

> **The headline.** §9.2 asked whether any **unscoped** statement of a certificate key-size floor
> exists, and answered *not in the four documents we read*. It exists. The IETF states one in **three
> different document classes**, and one of the three is in **an appendix of a document §2.3.1 and §9.2
> both cite as setting no floor**. The number 112 is not permanently NIST's; more precisely, **the
> numbers 2048 and 224 are not NIST's alone**, and the reason §9.2 could not see them is that it
> searched a document *set* where it should have searched a document *class*.

### 13.1 What was retrieved, enumerated

The negative in §9.2 was established over four documents. This is the corpus it is now established
over — enumerated, because a negative is only ever as wide as its corpus and the whole lesson of this
amendment is that §9.2's was too narrow.

| Corpus | Enumerated as | Result |
|---|---|---|
| **The IETF's TLS deployment BCP** | RFC 9325 (BCP 195, November 2022), obsoleting RFC 7525; checked for errata (**none**) and for updates (**RFC 10015** and **RFC 9852**, neither touching §4.5) | **Positive — a MUST-level RSA floor** |
| **The TLS specification, current** | **RFC 9846** (July 2026, Standards Track), which **obsoletes RFC 8446, RFC 5246, RFC 8422, RFC 5077, RFC 6961 and RFC 7627**; no errata | **Positive — RSA and ECDSA floors, at SHOULD** |
| **The TLS specifications, superseded** | RFC 8446 (2018), RFC 5246 (2008) | **Positive, and missed by §9.2** — see 13.3 |
| **LAMPS, published** | **63 RFCs**, enumerated from the Datatracker group API and cross-checked against the working group's document page, all bodies retrieved and grepped | **Positive on RSA and DSA, S/MIME-scoped** — RFC 8550, RFC 8551, RFC 9481 |
| **LAMPS, in flight** | **17 active Internet-Drafts**, including `draft-ietf-lamps-rfc8550bis-00` and `draft-ietf-lamps-rfc8551bis-00` | Floors preserved verbatim; **no new floor** |
| **An RFC 5280 revision** | **Does not exist.** `draft-ietf-lamps-rfc5280bis` returns 404; no `5280bis` appears in the 702-document LAMPS corpus; the charter has no such milestone. RFC 5280 is amended piecemeal (RFC 8399, 9549, 9598, 9608, 9618, 10007), and **none of those touches key size** | **Negative, and structurally so** |
| **Every IETF BCP** | **247 RFCs carrying an *Also BCP* designation**, enumerated from `rfc-index.txt`; full text retrieved for every security, crypto, key, certificate, algorithm and privacy candidate — BCP 61, 72, 86, 106, 107, 132, 173, 174, 179, 182, 188, **195**, 200, 201, 227, 228, 232, 237, 240, 243, 247 and twelve others | **Exactly one positive: BCP 195.** Every other either sets no floor or delegates to BCP 86 |
| **IAB statements** | **~180**, enumerated from the Datatracker `iab` statement stream (`iab.org/documents/correspondence-reports-documents/` now 301s to it); text retrieved for every crypto- or security-relevant one | **Negative, and one of them is a negative on purpose** — see 13.4 |
| **The IETF's other algorithm-requirements documents** | RFC 8247 (IKEv2), RFC 5480 (ECC in PKIX), RFC 8422 (obsoleted by RFC 9846), RFC 8017 (PKCS #1), RFC 7935 (RPKI) | RFC 8247 positive at SHOULD NOT for a different artefact; RFC 7935 positive but an **exact size**, RPKI-scoped; **RFC 5480, RFC 8422 and RFC 8017 negative and instructive** — see 13.4 |

**What the enumeration cost, and why it is written down.** §9.2's negative was over four documents. This
one is over roughly **three hundred and forty** — 63 LAMPS RFCs, 17 LAMPS drafts, ~180 IAB statements,
247 BCP-designated RFCs filtered to the security area, and the TLS and PKIX line individually. **The
positive was in the third one anybody would have guessed**, which is the whole of 13.6's first
statement: the corpus was never large, it was the wrong shape.

### 13.2 The finding — three unscoped statements, in three document classes

Each is quoted from retrieved bytes with enough context that its antecedent is visible.

#### RFC 9325 §4.5 — a MUST, and the strongest of the three

> "When using RSA, servers MUST authenticate using certificates with at least a 2048-bit modulus for
> the public key. In addition, the use of the SHA-256 hash algorithm is RECOMMENDED and SHA-1 or MD5
> MUST NOT be used [RFC9155]…"
> — [RFC 9325](https://www.rfc-editor.org/rfc/rfc9325.txt), BCP 195, §4.5 *Public Key Length*

Three things make this the load-bearing sentence and all three are checkable:

1. **It is unscoped.** RFC 9325 carries no jurisdictional addressee anywhere — a case-insensitive
   search for `federal`, `government`, `jurisdiction` and `United States` matches once, inside the
   title of a cited paper. Its normative verbs are BCP 14's, whose definitions restrict audience not
   at all. This is the exact contrast with SP 800-131A §1.2.2's *"a requirement for Federal Government
   use"*.
2. **It is deliberate.** RFC 7525 §4.3 (2015) read *"servers **SHOULD** authenticate using
   certificates with at least a 2048-bit modulus"*, and RFC 9325's Appendix A lists the change in its
   own words: *"MUST-level requirement for server-side RSA certificates to have a 2048-bit modulus at
   a minimum, replacing a 'SHOULD'."* The IETF raised this verb on purpose, seven years apart.
3. **The document calls itself a floor.** §1: *"this document provides a floor, not a ceiling"*, and
   *"These are minimum recommendations for the use of TLS in the vast majority of implementation and
   deployment scenarios."*

**Two conditionals quoted rather than dropped, per #46.** §4.5 opens *"When using the cipher suites
recommended in this document…"*, and the RSA sentence carries its own *"When using RSA"*. A cautious
reader can ask whether a deployment negotiating something outside §4.2's recommended list escapes the
floor. Three things answer it and the third is decisive:

1. §1 states the recommendations as minima for *"the vast majority of implementation and deployment scenarios"*.
2. the opening clause introduces the paragraph that **counts** the public keys in a handshake, not the paragraph that constrains them.
3. **RFC 9846 §C.2 carries the same floor with no cipher-suite antecedent at all**, so the conditional is not load-bearing for the row either way.

**The scope that is real, and it points the right way.** RFC 9325 is scoped *technically* — to TLS and
DTLS server authentication — even though it is unscoped *jurisdictionally*. That is a genuine limit and
it is not a limit here: [ADR-0024](../adr/0024-a-rules-domain-is-the-extension-of-its-name.md) fixes
this rule's domain at a `Presented` certificate, and §4.1 fixes its population at the chain a TLS
server sent. **RFC 9325's technical scope and this row's population are the same set.** The sentence
must not be carried anywhere wider — it says nothing about an S/MIME key, a code-signing key or a
certificate nobody presented over TLS — and this table never asks it to.

#### RFC 9846 Appendix C.2 — both numbers, and the word *insecure* in everything but name

> "Applications SHOULD also enforce minimum and maximum key sizes. For example, certification paths
> containing keys or signatures weaker than 2048-bit RSA or 224-bit ECDSA are **not appropriate for
> secure applications**."
> — [RFC 9846](https://www.rfc-editor.org/rfc/rfc9846.txt) §C.2 *Certificates and Authentication*

This sentence does four things no NIST sentence in §3.3 does. It is **unscoped**. It names **both**
of the two live floors and **exactly this table's two numbers**. It ranges over **certification
paths** — which is this table's population as §4.1 defines it, the chain rather than the leaf. And
its predicate is **fitness**, not approval: *not appropriate for secure applications* is a statement
about what the key is good for, where *Disallowed* is a statement about what an agency may buy. §9.2
asked for exactly this discrimination — *insecure as opposed to unapproved* — and this is the side of
it §9.2 said did not exist.

**Its softness is named rather than smoothed.** The verb is `SHOULD`, the numbers arrive in a *"For
example"* clause, and Appendix C is *Implementation Notes*, introduced as *"several recommendations
to assist implementors"*. It is weaker than a body MUST. It is not weaker in **scope**, which is the
axis §9 is about.

#### RFC 8550 §6 and RFC 8551 §6 — the word itself, on a document about certificates

> "RSA keys of less than 2048 bits are now considered by many experts to be **cryptographically
> insecure** (due to advances in computing power) and SHOULD no longer be used to sign certificates
> or CRLs."
> — [RFC 8550](https://www.rfc-editor.org/rfc/rfc8550.txt) §6, *S/MIME 4.0 Certificate Handling*

> "**RSA and DSA keys of less than 2048 bits** are now considered by many experts to be
> **cryptographically insecure** (due to advances in computing power) and should no longer be used to
> protect messages."
> — [RFC 8551](https://www.rfc-editor.org/rfc/rfc8551.txt) §6, *S/MIME 4.0 Message Specification*

And the hard verb, in the same family:

> "An S/MIME user agent MUST NOT generate asymmetric keys less than 2048 bits for use with an RSA
> signature algorithm."
> — [RFC 8551](https://www.rfc-editor.org/rfc/rfc8551.txt) §4.1 *Key Pair Generation*

**Quoted with the hedge, per §12's standing rule and #46's.** *"Considered by many experts to be"* is
a real hedge and it is not removed here. The sentence is cited for the word **insecure** and for the
fact that its grammatical subject is **the key**, not a use. **Both bis drafts preserve it verbatim**,
so it is the working group's current position and not a 2019 residue.

**The artefact caveat, stated rather than elided.** RFC 8550 and RFC 8551 are S/MIME documents, and
under §2.3.2's artefact test an S/MIME document's *support matrix* — RFC 8550 §4.3's `key size <=
2047 : SHOULD NOT` table, addressed to *"S/MIME receiving agents"* — is a **relying-party acceptance
rule** and is in the distributor position exactly as the CA/B BR is. §6's sentence is different in
kind: its subject is *RSA keys of less than 2048 bits*, unqualified by S/MIME, and its predicate is a
property of the key. **It corroborates and it is not sole grounds**, and the RSA row does not need it
to be.

### 13.3 The correction §9.2 owes — the sentence was in two documents #68 held

§2.3.1 and §9.2 both state that RFC 8446 and RFC 5246 set no floor, and §1's summary repeats it as
*"TLS constrains signature algorithms, never modulus size."* **That is wrong, and it was wrong when it
was written.** RFC 8446 Appendix C.2 carries the *2048-bit RSA or 224-bit ECDSA* sentence
byte-for-byte as RFC 9846 §C.2 does, and RFC 5246 §D.3 carries its ancestor:

> "TLS supports a range of key sizes and security levels, including some that provide no or minimal
> security. … Applications should also enforce minimum and maximum key sizes. For example, certificate
> chains containing **512-bit RSA keys** or signatures are not appropriate for **high-security**
> applications."
> — [RFC 5246](https://www.rfc-editor.org/rfc/rfc5246.txt) §D.3

The trajectory is the interesting part and it is three documents long: **512 bits and *high-security
applications* (2008) → 2048-bit RSA and 224-bit ECDSA and *secure applications* (2018) → unchanged
(2026)**. The IETF raised the number by two orders of magnitude and **widened the addressee** from a
security-conscious subset to secure applications generally. That is the opposite motion from a scope
qualifier being added.

**How the row moves without breaking #37's precedent.** #37 binds: *a row moves on retrieval, never on
a re-reading of text already held.* Read strictly, RFC 8446 and RFC 5246 were held, and finding C.2 in
them is a re-reading. **The rows below move on RFC 9846** — a July 2026 Standards Track publication
that obsoletes both, which #68 could not have held and which this ticket retrieved — and on **RFC
9325**, which #68 never read at all. RFC 8446 §C.2 and RFC 5246 §D.3 are recorded here as **a
correction of what those documents say**, which is a different act from re-interpreting them: #37
repaired defects of exactly this kind, and leaving a false sentence standing because the precedent
protecting against motivated re-reading also happens to cover it would be the precedent doing damage.

### 13.4 What did **not** move, and the two traps that made it look as though it had

**DSA's `N` limb has nothing unscoped.** RFC 8550 §6 and RFC 8551 §6 both say *RSA and DSA keys of
less than 2048 bits*, which reaches the `L` limb of `(L, N)` and says nothing about `N ≥ 224`. RFC
9846 goes further in a direction that does not help: *"MD5 [SLOTH], SHA-224, and DSA MUST NOT be
used"* is a **prohibition of the algorithm in TLS**, which is §2.2's already-refused *withdrawal of
approval* arriving from a second body. It is not a work-factor statement and no claim in the closed
set fits it. **The DSA row's `N` limb is the whole of what remains in the weak tier.**

**Trap one — the number that matches is in the wrong sentence.** RFC 9325 §4.5 contains the string
*"Curves of less than 224 bits MUST NOT be used"*, which is this table's ECDSA number **exactly**, in
a MUST, in an unscoped BCP. It is the wrong sentence. Its antecedent is two sentences earlier — *"With
regard to **ECDH** keys, implementers are referred to the IANA 'TLS Supported Groups' registry"* — its
justification is *"in line with the latest revision of [NIST.SP.800-56A]"*, which is a **key
agreement** standard rather than a signature one, and RFC 9325's own Appendix A files the change under
*Differences specific to TLS 1.2* as *"ECDH minimal curve size is 224 (vs. 192 previously)."* It is
about the ephemeral key-agreement group, not about the certificate's ECDSA key. **A grep for `224`
finds this sentence before it finds RFC 9846 §C.2, and the two are one word apart in plausibility and
a whole artefact apart in fact.** This is #46's truncated conditional inverted: there, a quote lost its
condition. Here, a quote arrives with a number that matches and a subject that does not.

**Trap two — the IETF's ECC-in-PKIX document sets no floor and reads as though it might.** RFC 5480
§4 asks *"What is the public key size?"*, gives a table of *"comparable minimum bits of security
[SP800-57] for the ECDSA key sizes"*, and then does the opposite of setting a floor: *"To promote
interoperability, the following choices are RECOMMENDED"* — whose **bottom row is 80 bits of security
at `secp192r1`**. The IETF's own ECC certificate-format document had the opportunity and declined it,
and it is a strength **mapping** in RFC 3766's shape rather than a floor. Recorded because a session
that finds RFC 5480 §4 and stops reading has found evidence pointing the other way from the direction
it appears to point.

**RFC 9481 §7 is the same shape and is recorded as corroboration only.** LAMPS's CMP algorithms
document publishes an unscoped bits-of-security table anchoring **112 bits to RSA-2048 and
`secp224r1`**, with *"the implementer MUST choose one offering more bits of security"* — but the
obligation is **relative to the key being managed**, not absolute. It corroborates that the IETF
reckons in the same units and lands on the same two numbers. It is not a floor.

**Trap three — RFC 8422 removes the small curves and does not say they are weak.** §5.1.1 deprecates
`NamedCurve` identifiers 1–22, leaving only P-256 and above, and states its reason: *"RFC 4492 defined
25 different curves in the NamedCurve registry … **Only three have seen much use.** This specification
is deprecating the rest."* The ground is **deployment share**, which is §6.1's *frequency is not a
position* refused in the source's own words rather than in ours. The **effect** is a 256-bit floor and
the **claim** is that nobody used them. A row founded on it would be a row founded on a popularity
measurement. (RFC 8422 is in any case obsoleted by RFC 9846.)

**Trap four — the number 112 has an IETF pedigree and it is not the one it looks like.** RFC 9325 §4.2
is unscoped and normative — *"Implementations MUST NOT negotiate cipher suites offering less than 112
bits of security"* — but its **rationale** reads *"Based on [RFC3766], at least 112 bits of security is
needed"*, and RFC 3766 §4 disclaims exactly that:

> "…we will implicitly assume that the system security requirement is 112 bits; **this doesn't mean
> that 112 bits is recommended. In fact, 112 bits is arguably too strong for any practical purpose.**
> It is used for illustration simply because that is the upper bound on the strength of TripleDES."
> — [RFC 3766 / BCP 86](https://www.rfc-editor.org/rfc/rfc3766.txt) §4

So the answer to *is 112 permanently NIST's* is **narrower than a yes**, and the narrowing is worth
having. **The IETF states the number 112 unscoped and normatively**, so it is not NIST's alone. But it
states it about **negotiated cipher suites**, its stated authority for it is a document that declines
to recommend it, and the two numbers this table actually uses — **2048 and 224** — are the ones that
carry unscoped IETF footing in their own right. **The rows rest on 2048 and 224, not on 112**, and that
is why the finding survives this trap: the security-strength level is the thing with the thin pedigree,
and no row keys on it. §2.4's *the table's key must be `(algorithm, parameter)` and never a bare bit
count* turns out to have protected against this too, which nobody anticipated.

**The IAB's negative is deliberate and is the cleanest non-statement in the corpus.** RFC 1984
(BCP 200, IAB and IESG) has a section headed **`KEY SIZE`**, and it sets none — it argues against the
idea:

> "It is not acceptable to restrict the use or export of cryptosystems based on their key size. Systems
> that are breakable by one country will be breakable by others, possibly unfriendly ones. … as
> computers increase in speed, key sizes that were once out of reach of cryptanalysis will become
> insecure."
> — [RFC 1984 / BCP 200](https://www.rfc-editor.org/rfc/rfc1984.txt), *Technical Analysis — Key Size*

**The string `2048` does not occur in RFC 1984, and neither does any other key size.** This is §2.7's
citable non-statement in its strongest form yet: the body one would ask for an internet-wide key-size
position has a section of that exact name whose content is *do not legislate key sizes*. It reads as a
floor to a search engine and is the opposite of one, and it is recorded so that no later session
retrieves the heading and stops.

**Two more negatives, both confirmations.** RFC 8017 (PKCS #1 v2.2) sets **no modulus floor** — §9.2's
claim is confirmed against the retrieved bytes. Its only size constraints are structural (*"If k <
2hLen + 2, output 'decryption error'"*, *"RSA modulus too short"*), which ask whether the modulus can
hold a padded message and not whether it is strong enough. And **RFC 7935 §3** (RPKI) is the near-miss
worth naming: *"The RSA key pairs used to compute the signatures MUST have a 2048-bit modulus and a
public exponent (e) of 65,537."* That is an **exact size rather than a floor** — a 4096-bit modulus is
non-conformant to it — it is RPKI-scoped, and it reaches this table's population not at all.

### 13.5 The weak tier, restated — one row, on one limb

**Supersedes §9's tier table.** §9 stands unrewritten. This is what it should read.

| Row | Footing after this retrieval | Tier |
|---|---|---|
| **MD5 signature** | RFC 9846 §4.5.1.3 *"MUST abort the handshake"*; RFC 6151 §2 | Unchanged — strongest in the table |
| **SHA-1 signature** | RFC 9846 §4.3.3 and §4.5.1.3; SP 800-131A Table 8; RFC 9155 §2 with RFC 5246 §7.4.2 | Unchanged — strong, with the `RECOMMENDED` softness §9 already named |
| **RSA `nlen` < 2048** | **RFC 9325 §4.5 (MUST, unscoped, BCP 195)**; RFC 9846 §C.2; RFC 8550 §6 and RFC 8551 §6 (*"cryptographically insecure"*); RFC 8551 §4.1 (MUST NOT generate); FIPS 186-5 §5.1 | **Out of the weak tier.** Four unscoped statements from a second owner, one of them a MUST over exactly this artefact |
| **ECDSA `len(n)` < 224** | **RFC 9846 §C.2**, unscoped, naming the number and the population; RFC 9481 §7 corroborating the unit | **Out of the *scope*-weak tier, into SHA-1's tier** — strong with a named softness, and the softness is **modality** (a `SHOULD` in an appendix, the number in an illustrative clause), not scope |
| **DSA below (2048, 224)** | `L ≥ 2048`: RFC 8550 §6 and RFC 8551 §6, unscoped. **`N ≥ 224`: SP 800-131A Table 2 only** | **The weak tier, and it is now one limb of one row** |

**The judgement, flagged as this amendment's thin ground.** Moving the ECDSA row is a call, not a
reading. Against it: RFC 9846 §C.2's number sits in a *"For example"*, and a reader may hold that the
normative content is *enforce some minimum* rather than *enforce 224*. For it: the sentence is
unscoped, from an owner, over this table's exact population, carrying a fitness predicate rather than
an approval one, and carried deliberately through three successive specifications with the number
raised. This table already ships the SHA-1 row outside the weak tier on a `RECOMMENDED`, so
holding ECDSA in on a `SHOULD` would apply two standards to one document. **Ruled: it moves, and the
softness is disclosed as modality.** A session that disagrees is disagreeing about a tier label and
not about a row — no row's membership, floor or predicate changes here.

**Nothing else moves.** The table is still five rows, every floor is the same integer, and the
predicate in §4 is untouched. **What moved is footing, exactly as #69 moved a footing rather than a
row** — and in the same direction, a weak tier shrinking on retrieval.

### 13.6 What a scope weakness is allowed to be — the general statement

§9.3 and [ADR-0035](../adr/0035-a-cryptographic-primitives-owner-is-its-specifier.md) §7 called this
a **third kind** of weak row and predicted its resolution route: *it resolves only if a second owner
speaks unscoped, so its retrieval is a search of a corpus rather than a request to a party.* **The
prediction held and the search succeeded on its first run**, which changes what the disclosure is
allowed to say. [ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md) holds
the ruling. The two general statements are these.

> **A specification's silence is not the owner's silence.** A standards body speaks in more than one
> document class, and the classes disagree by design: its **specification** defines the artefact and
> characteristically declines to fix numbers that cryptanalysis will move; its **deployment
> recommendation** fixes them; its **implementation guidance** illustrates them. A negative
> established over specifications is a negative about **that class**, and must be recorded as one.
> Before a table records that an owner sets no number, it enumerates the owner's document classes,
> not its documents.

> **A scope weakness is disclosed as a searched corpus, never as a caveat and never as a promise.**
> Before its search has run it names the corpus it will search. Once run, the disclosure carries
> **(i)** the corpus actually searched, enumerated; **(ii)** what was found and which rows it reached;
> and **(iii)** the smallest extension of the corpus that could still change the answer. A weakness
> that survives that is not *permanent* — it is **bounded**, and bounded is falsifiable by anyone who
> can name a document outside the boundary, which is the property a permanent caveat lacks and the
> reason [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) forbids
> one.

**Applied to the row that survives.** DSA's `N ≥ 224` limb rests on SP 800-131A Table 2 alone. Its
corpus boundary, per (iii): the smallest extension that could still change it is a body that
**specifies DSA domain parameters** speaking unscoped about `N` — and the enumerable candidates are
ANSI X9.30/X9.62's successors and the national schemes (BSI TR-02102, SOG-IS). **Each of those
reproduces NIST's shape in a different flag**: a recommendation addressed by a body to its own
constituency. That is why the weakness is structural rather than accidental — **a key-size floor is
the kind of statement bodies make to constituencies, and the IETF is unusual in having one that is
"the internet"** — and it is also why the extension is named without being recommended. **The
honest disclosure for the DSA row is that its corpus is bounded, the boundary is named, and the row
ships**, exactly as `5432/tcp` does.

### 13.7 Citation corrections forced by this retrieval

Recorded here rather than edited into §3.3 and the Sources, per the name-and-withdraw convention.
**No row's grounds change. Three citations are stale and one is wrong.**

| Where | Reads | Should read |
|---|---|---|
| §2.3.1, §9.2, ADR-0035 §1, and this note's Sources | **RFC 5280 §11** Security Considerations | **RFC 5280 §8.** §11 is *References*. Verified against retrieved bytes: `8.  Security Considerations` at the document's own section 8, `11.  References` at 11. The quoted text is correct; the section number never was |
| §3.1, §3.3, §9, Sources | **RFC 8446** §4.2.3, §4.4.2.4 | **RFC 9846** (July 2026) §4.3.3 and §4.5.1.3, which obsoletes RFC 8446. Both sentences survive verbatim — *"MUST abort the handshake with a 'bad_certificate' alert"*, and the self-signed reason §4.1 relies on. RFC 8446 remains citable as the text in force 2018–2026 |
| §3.3, §9, Sources | **RFC 5246** §7.4.2, for the TLS 1.2 SHA-1 chain | **Obsoleted by RFC 9846**, along with RFC 8422. The §3.3 chain still holds for TLS 1.2 as historically specified; a reader must not be left thinking RFC 5246 is current |
| §2.6, §9.2 | RFC 8446 and RFC 5246 *"set none"* | **False** — see 13.3. Both set one, in an implementation-notes appendix |

**§11's third open question is answered in passing.** *Who watches these five rows* now has a second
answer: the watch is not only *re-read SP 800-131A when a revision goes final*. **RFC 9846 landed in
July 2026 and obsoleted the document two of the five rows are footed on, four weeks before this was
written.** The IETF half of this table moves on a cadence nobody had priced. §7.1's *one edit per row
per decade* is unaffected — no row's content moved — but the **citation** surface moves faster than
the **content** surface, and only the second was measured.

### 13.8 Retrieval hazards met, recorded per `sensitive-ports.md` §9.5

- **The floor is stated as an example, so normative-language search does not find it.** RFC 9846 §C.2
  and RFC 5246 §D.3 both put the number after *"For example"*. A search for `MUST`, `at least` or
  `minimum … bits` near a key algorithm misses both. **This is the mechanism by which #68's negative
  was wrong**, and it is worth more than the finding: a floor can be stated in an illustrative clause
  and still be the only place its author states it.
- **The right number in the wrong sentence.** RFC 9325's *"Curves of less than 224 bits MUST NOT be
  used"* matches this table's ECDSA floor to the digit and is about ECDH key agreement — 13.4, trap
  one.
- **An appendix is easy to treat as non-normative and this one is not marked so.** RFC 9846's
  Appendix C is part of a Standards Track document and uses BCP 14 keywords. It is weaker than the
  body by convention, not by declaration, and the note says so rather than picking a side.
- **The current TLS RFC is not the one everyone cites.** RFC 8446 is obsoleted by RFC 9846 as of July
  2026. Every secondary source and most tooling still says 8446. The section numbers **moved** —
  §4.4.2.4 → §4.5.1.3, §4.2.3 → §4.3.3 — so a citation copied forward without re-reading points at
  the wrong section of the right document.
- **A working group's negative needs the charter, not just the document list.** The LAMPS result — no
  RFC 5280 revision exists — is established by a 404 on `draft-ietf-lamps-rfc5280bis`, its absence
  from the 702-document group corpus, **and** the charter's milestones, which name no such work. Any
  two of the three would have left *not yet posted* open.
- **`rfc-editor.org/errata/rfcNNNN` redirects to a search form that renders "No matching errata
  found" identically to a failed query.** Both RFC 9325 and RFC 9846 were confirmed errata-free by
  reading that string out of the retrieved HTML. A session that greps for an errata table and finds
  none cannot distinguish *no errata* from *page did not load*.
- **`Updated by` is not `Obsoleted by` and both matter.** RFC 9325 is updated by RFC 10015 and RFC
  9852. RFC 10015 §6 tabulates its changes and **all of them are in §4.1**, so §4.5's RSA sentence
  stands. Checking only for obsolescence would have missed the question. Checking only the count
  would have answered it wrongly.

---

## 14. Amendment — [#79](https://github.com/winniel123/verge-asm/issues/79): the class audit of §5's negatives

[ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md) was measured on this
table's own failure — #68 read five documents, every one a specification or a method, and recorded
that the IETF sets no certificate key-size floor. **§5's negative space was built by the same session,
by the same method, and has never been re-checked against the rule that failure produced.** This
section does that. It **amends §5 by reference**: earlier text is left standing and marked, and where
§14 and §5 disagree, §14 governs. The general rule is
[ADR-0046](../adr/0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md).
The companion audit of the other curated table is
[`sensitive-ports.md`](./sensitive-ports.md) §17.

**Headline result.**

> **No row enters the table and no row leaves. It is still five rows.** Two of §5's seven entries are
> **attestation** negatives and are therefore exposed; the other five are refused on the **claim**, and
> a claim failure has no class list. Both exposed negatives were swept and both survive. **The MD4 half
> of §5's first entry was never measured at all** — #68 cited RFC 6149, which is about MD2 — and RFC
> 6150 has now been retrieved and read end to end.

### 14.1 The population, enumerated

| # | §5 entry | What kind of refusal | Classes searched | Classes not searched | Exposed? |
|---|---|---|---|---|---|
| 1 | **MD2 and MD4 signatures** — *"No owner sentence prohibits MD2 in a certificate"* | **Attestation** — an owner silence | status-change document (RFC 6149, MD2) · specification (RFC 8446 §4.4.2.4) | **deployment** (RFC 9325 / BCP 195) · **implementation guidance** (RFC 9846 Appendix C) · **the MD4 status-change document (RFC 6150)** | **Yes** |
| 2 | **RSA public exponent** — *"no owner calls a small exponent **weak**"* | **Attestation**, with a claim limb behind it | specification (RFC 8017, FIPS 186-5) · distributor (the CA/B BR, refused on §2.3's artefact test) | **deployment** (RFC 9325) | **Yes** |
| 3 | **RSASSA-PSS parameter defects** | **Claim** — encoding conformance is neither work factor nor collision resistance | — | — | No |
| 4 | **Known-compromised individual keys** | **Claim** — not a property of an `(algorithm, parameter)` pair, and §7.2 refuses it independently | — | — | No |
| 5 | **Certificate lifetime, key reuse, missing CT** | **Claim** — different facts, two of them other rules | — | — | No |
| 6 | **A "should be 3072 by 2031" second threshold** | **Constraint** — severity, refused by the ticket and by §7.3 | — | — | No |
| 7 | **Trust-store membership** | **Scope** — ADR-0032 already ruled v1 ships no root bundle | — | — | No |

Five of the seven are refused on the **claim**, and ADR-0046's exposure test says so directly: a claim
failure is not a silence, so it has no document classes to enumerate. Retrieving every deployment BCP
in the IETF would not make a Debian OpenSSL key a property of an `(algorithm, parameter)` pair.

### 14.2 MD2 and MD4 — swept in three classes, and the MD4 half measured for the first time

**Two of the three unsearched classes were already in this repo's hands, which is
[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 1
paying out on a table rather than a file.** RFC 9325, RFC 9846 and RFC 8551 were all retrieved by
[#73](https://github.com/winniel123/verge-asm/issues/73) and all read for **key sizes**. Limb 1 requires
an artefact be read for *"every algorithm or key size"* in the table's domain, and MD2 and MD4 are in
it. Read for those:

> **[measured]** The strings `MD2` and `MD4` occur **zero times** in RFC 9325 (BCP 195), **zero times**
> in RFC 9846, and **zero times** in RFC 8551. RFC 9325 §4.5's certificate paragraph names the digests
> it means and they are the other two: *"the use of the SHA-256 hash algorithm is RECOMMENDED and SHA-1
> or MD5 MUST NOT be used [RFC9155]"*.

That is a negative established in the **deployment** class and in the **implementation-guidance** class
— the two classes whose omission produced #68's failure — and it is the strongest form the negative can
take, because RFC 9325 §4.5 is the exact sentence #73 found for the key rows. **The body that does state
an unscoped certificate floor, in the document where it states it, declines to name MD2 or MD4.**

**The MD4 half was never measured.** §5's entry names *"MD2 and MD4"* and cites **RFC 6149**, which is
*MD2 to Historic Status*. **RFC 6150, `MD4 to Historic Status`, appears nowhere in this note and was
retrieved for the first time by this ticket.** Read end to end, it does not carry a certificate
prohibition, and the two sentences that come closest are instructive:

> "The pre-image attacks on MD4 are practical. It cannot be used as a one-way function. For example, it
> must not be used to hash a cryptographic key of 80 bits or longer."
> — [RFC 6150](https://www.rfc-editor.org/rfc/rfc6150.txt), §6.2, *Pre-Image and Second Pre-Image Resistance*

> "Despite MD4 seeing some deployment on the Internet, this specification obsoletes [MD4] because MD4 is
> not a reasonable candidate for further standardization … MD4 is clearly showing signs of weakness, and
> implementations **should strongly consider** removing support and migrating to another hash algorithm."
> — [RFC 6150](https://www.rfc-editor.org/rfc/rfc6150.txt), §7, *Recommendation*

The first is a `must not` at full strength on **the wrong artefact** — hashing a key, not signing a
certificate — which is §2.3's artefact test and #73's *right number in the wrong sentence* trap arriving
as *right modality in the wrong sentence*. A session grepping RFC 6150 for `must not` finds it first.
The second is hedged in the way §5 already describes for MD2, and it opens with a **frequency clause**.
§5's characterisation of the retiring documents as *"Informational and hedge their own results"* is
therefore **confirmed for MD4 rather than merely extended to it**.

> **Ruling: the MD2/MD4 exclusion stands, and its footing improves from an inference to a
> measurement.** §5's *"we could not find anyone entitled to say it"* is now a **searched corpus**: the
> status-change documents for both algorithms, the current TLS specification, its implementation-guidance
> appendix, the deployment BCP that carries this table's own RSA floor, and the S/MIME message
> specification. **The smallest extension that could still move it:** a LAMPS document profiling
> certificate signature algorithms that names MD2 or MD4 with a prohibition — RFC 3279 and RFC 4055 are
> the candidates, and both are **specifications**, the class ADR-0040 predicts will be silent.

### 14.3 The RSA public exponent — swept in the deployment class, and it is silent

§5's second entry rests on *"no owner calls a small exponent **weak**"* — RFC 8017 supports `e = 3`, and
FIPS 186-5's range is an issuance requirement rather than a strength claim. Both are **specifications**.
The unsearched class is the same one #68 missed.

> **[measured]** `exponent` occurs in RFC 9325 only at §7.4 *Diffie-Hellman Exponent Reuse*, which is
> about reusing DH and ECDH ephemeral exponents across connections. **RFC 9325 states no position on the
> RSA public exponent at all**, and it is the document that does state this table's RSA modulus floor.

The negative survives in the class most likely to have broken it, and the entry's second limb — *"No
claim in §2.2 fits"* — is untouched and is independently sufficient. §5's entry needs no change.

### 14.4 §9's weak tier, and the one distinction this audit adds

§9's DSA `N ≥ 224` residue was already swept by #73 across roughly 340 documents and is already
disclosed as **bounded** with its boundary named (§13.6, ADR-0040 §3). Nothing here touches it.

What this audit adds is a distinction §13's disclosure did not need and the next one will:

> **A weakness of *breadth* is not a weakness of *class*.** Enumerating an owner's document classes
> cures a negative that asked the wrong **kind** of document. It cannot cure a negative that asked the
> right kind and asked only one **party** — because the set of parties is open where the set of classes
> is closed. [ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md)'s obligation
> does not reach a breadth weakness, and treating it as though it did would make its corpus unbounded,
> which is the property ADR-0040 exists to deny.

The live instance is in a sibling note rather than this one:
[`acme-renewal-timing.md`](./acme-renewal-timing.md) §14 discloses that *"No second CA was retrieved
stating a different fraction, and none was retrieved stating the same one."* The class — **issuer
documentation** — was searched. Searching it at a second issuer would test the ruling's **generality**,
not its **completeness**, and §14's disclosure already states it correctly.

### 14.5 What this changed

**Nothing in the table.** Five rows. §1's summary, §3's tables and §4's chain are untouched. No `Break`
([ADR-0008](../adr/0008-derivation-versions-move-on-content.md)), no version bump.

**Two things about §5.** The MD2/MD4 entry's ground is **narrowed and strengthened**: it is no longer
*we could not find anyone entitled to say it* but *the three classes that could say it were searched and
two of them name the other digests instead*. And **RFC 6150 joins the corpus**, closing a gap in which
half of a two-algorithm exclusion rested on a document about the other algorithm.

**One thing about the method.** ADR-0037 limb 1's re-read of held artefacts produced a **confirmation**
here rather than a discovery — three documents, three zero counts. Limb 1's value is usually argued from
#69's near-miss, where the re-read found something. This is the other half of the case, where it found
nothing and thereby converted an inference into a measurement at the cost of three `grep`s.

### 14.6 Thin ground, flagged per the standing rule

**Zero-count negatives over a document are only as good as the search terms.** `MD2` and `MD4` are exact
strings and were searched case-sensitively over the retrieved bytes. A document that wrote *"the MD
family"* or referred to the algorithms only by OID would not match. The OID forms
(`1.2.840.113549.1.1.2`, `md2WithRSAEncryption`) were **not** searched, and that is a real gap in three
documents whose subject is TLS and S/MIME rather than certificate profiling.

**[measured] RFC 6150's own running header says `MD2 to Historic Status` on every page.** The title,
abstract and body are all about MD4. The header is a publication defect in the RFC itself. Two
consequences for anyone re-checking this: a case-insensitive search of RFC 6150 for `MD2` returns hits
that are **all page headers and none content**, and a session searching a local RFC corpus for *"MD4 to
Historic"* by header will not find the document that is exactly that. This is the `LDAPString` trap
(§9.5 of `sensitive-ports.md`) arriving from the publisher rather than from the reader.

**RFC 3279 and RFC 4055 were named as the boundary and not retrieved.** They are the PKIX algorithm
profiles and they are the last place a certificate-scoped MD2 sentence could live. They are
**specifications**, which is the class ADR-0040 gives the lowest prior, and no row moves on the answer
in either direction — MD2 is not in the table and would enter only on a prohibition. Named so the gap is
visible rather than assumed closed.

**§14.1's *exposed?* column depends on §2.2's closed claim set** in the same way `sensitive-ports.md`
§15.9 records. Five of seven entries are called un-exposed because a claim failure has no class list. If
§2.2's two-claim derivation is wrong, some of those five become attestation questions and the sweep is
larger than two.

---

## Sources

Retrieved by [#79](https://github.com/winniel123/verge-asm/issues/79) — §14's class audit
- [RFC 6150, MD4 to Historic Status](https://www.rfc-editor.org/rfc/rfc6150.txt) (March 2011) — §6.2, §7. **The MD4 half of §5's first entry, retrieved for the first time**. It carries no certificate prohibition, and its `must not` is about hashing a cryptographic key. Note that the RFC's own running header reads *"MD2 to Historic Status"*, a publication defect in the document itself
- RFC 9325, RFC 9846 and RFC 8551 **re-read for the rest of this table's domain** per [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 1, over the same bytes #73 retrieved — `MD2` and `MD4` occur **zero times** in each, and `exponent` occurs in RFC 9325 only at §7.4, about Diffie-Hellman exponent **reuse**
- Named as the boundary and **not retrieved** — recorded so the gap is visible: [RFC 3279](https://www.rfc-editor.org/rfc/rfc3279.txt) and [RFC 4055](https://www.rfc-editor.org/rfc/rfc4055.txt), the PKIX algorithm profiles, which are **specifications** and on which no row turns

Retrieved by [#73](https://github.com/winniel123/verge-asm/issues/73) — the unscoped-floor corpus
- [RFC 9325 / BCP 195, Recommendations for Secure Use of TLS and DTLS](https://www.rfc-editor.org/rfc/rfc9325.txt) (November 2022) — §1, §4.2, §4.5, §5, Appendix A. Obsoletes RFC 7525. **The MUST-level RSA floor**
- [RFC 7525 / BCP 195, Recommendations for Secure Use of TLS and DTLS](https://www.rfc-editor.org/rfc/rfc7525.txt) (May 2015) — §4.3, the `SHOULD` that RFC 9325 raised
- [RFC 9846, The Transport Layer Security (TLS) Protocol Version 1.3](https://www.rfc-editor.org/rfc/rfc9846.txt) (July 2026) — §4.3.3, §4.5.1.3, Appendix C.2. **Obsoletes RFC 8446, RFC 5246, RFC 8422, RFC 5077, RFC 6961, RFC 7627**
- [RFC 10015, Deprecating Obsolete Key Exchange Methods in TLS 1.2 and DTLS 1.2](https://www.rfc-editor.org/rfc/rfc10015.txt) — §6, which updates RFC 9325 §4.1 only
- [RFC 8550, S/MIME 4.0 Certificate Handling](https://www.rfc-editor.org/rfc/rfc8550.txt) (April 2019) — §4.3, §6. LAMPS
- [RFC 8551, S/MIME 4.0 Message Specification](https://www.rfc-editor.org/rfc/rfc8551.txt) (April 2019) — §4.1, §6. LAMPS
- [RFC 9481, Certificate Management Protocol (CMP) Algorithms](https://www.rfc-editor.org/rfc/rfc9481.txt) — §7, the bits-of-security table. LAMPS, corroboration only
- [RFC 5480, Elliptic Curve Cryptography Subject Public Key Information](https://www.rfc-editor.org/rfc/rfc5480.txt) (March 2009) — §4, a strength mapping and **not** a floor
- [RFC 8247, Algorithm Implementation Requirements and Usage Guidance for IKEv2](https://www.rfc-editor.org/rfc/rfc8247.txt) — §3.1.1, `SHOULD NOT` below 2048 for a different artefact
- [RFC 1984 / BCP 200, IAB and IESG Statement on Cryptographic Technology and the Internet](https://www.rfc-editor.org/rfc/rfc1984.txt) (August 1996) — *Technical Analysis — Key Size*, a section of that name setting no size
- [RFC 3766 / BCP 86, Determining Strengths For Public Keys](https://www.rfc-editor.org/rfc/rfc3766.txt) (April 2004) — §4, which disclaims recommending 112 bits
- [RFC 7696 / BCP 201, Guidelines for Cryptographic Algorithm Agility](https://www.rfc-editor.org/rfc/rfc7696.txt) — §2.2.2, which requires a size be *specified* and names no value
- [RFC 7935, The Profile for Algorithms and Key Sizes for Use in the RPKI](https://www.rfc-editor.org/rfc/rfc7935.txt) — §3, an exact size rather than a floor
- [RFC 8422, ECC Cipher Suites for TLS 1.2 and Earlier](https://www.rfc-editor.org/rfc/rfc8422.txt) — §5.1.1, curve deprecation on **usage** grounds. Obsoleted by RFC 9846

Specifications — the IETF, as owner of PKIX and TLS
- [RFC 5280, Internet X.509 Public Key Infrastructure Certificate and CRL Profile](https://www.rfc-editor.org/rfc/rfc5280.txt) (May 2008) — §11 Security Considerations, the no-floor position
- [RFC 8446, The Transport Layer Security (TLS) Protocol Version 1.3](https://www.rfc-editor.org/rfc/rfc8446.txt) (August 2018) — §4.2.3, §4.4.2.2, §4.4.2.4
- [RFC 5246, The Transport Layer Security (TLS) Protocol Version 1.2](https://www.rfc-editor.org/rfc/rfc5246.txt) (August 2008) — §7.4.2
- [RFC 9155, Deprecating MD5 and SHA-1 Signature Hashes in TLS 1.2 and DTLS 1.2](https://www.rfc-editor.org/rfc/rfc9155.txt) (December 2021) — §1, §2
- [RFC 6151, Updated Security Considerations for the MD5 Message-Digest and the HMAC-MD5 Algorithms](https://www.rfc-editor.org/rfc/rfc6151.txt) (March 2011) — §2
- [RFC 6149, MD2 to Historic Status](https://www.rfc-editor.org/rfc/rfc6149.txt) (March 2011) — §2, §6. The basis for MD2's exclusion
- [RFC 3279, Algorithms and Identifiers for the Internet X.509 PKI Certificate and CRL Profile](https://www.rfc-editor.org/rfc/rfc3279.txt) (April 2002) — the signature and key OIDs
- [RFC 8410, Algorithm Identifiers for Ed25519, Ed448, X25519, and X448 for Use in the Internet X.509 PKI](https://www.rfc-editor.org/rfc/rfc8410.txt) (August 2018) — `id-Ed25519`, `id-Ed448`
- [RFC 8017, PKCS #1: RSA Cryptography Specifications Version 2.2](https://www.rfc-editor.org/rfc/rfc8017.txt) (November 2016) — sets no modulus floor
- [RFC 3766 / BCP 86, Determining Strengths For Public Keys Used For Exchanging Symmetric Keys](https://www.rfc-editor.org/rfc/rfc3766.txt) (April 2004) — an estimation method, not a floor

Specifications — NIST, as owner of the SHA family, ECDSA and the P-curves
- [NIST SP 800-131A Rev. 2, Transitioning the Use of Cryptographic Algorithms and Key Lengths](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-131Ar2.pdf) (March 2019) — §1.2.2, §3 Table 2, §9 Table 8. Current final publication. [Rev. 3 exists as an initial public draft only](https://csrc.nist.gov/pubs/sp/800/131/a/r3/ipd)
- [FIPS 186-5, Digital Signature Standard (DSS)](https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.186-5.pdf) (February 2023) — §1, §4, §5.1, §7.1

Corroborators — recorded, never sole grounds
- [CA/Browser Forum, Baseline Requirements for the Issuance and Management of Publicly-Trusted TLS Server Certificates, v2.2.9](https://github.com/cabforum/servercert/blob/main/docs/BR.md) (adopted 2026-07-02, effective 2026-08-06) — §6.1.5, §6.1.6, §7.1.3.1, §7.1.3.2. Read from the Forum's own `cabforum/servercert` repository. See §12
- [Mozilla Root Store Policy v3.1](https://www.mozilla.org/en-US/about/governance/policies/security-group/certs/policy/) (effective 2026-07-01) — §5.1, §5.1.1, §5.1.2, §5.1.3
- [Microsoft Root Program Requirements](https://github.com/TrustedRootProgram/Program-Requirements/blob/main/Requirements.md) — §3.1.7, §3.1.9, §3.1.14, §3.1.20, §3.3.2
- [Chrome Root Program Policy v1.8](https://googlechrome.github.io/chromerootprogram/crp/policy/) (2026-02-05) — §1.1.1. States no algorithm or key-size requirement of its own
- [Apple Root Certificate Program](https://www.apple.com/certificateauthority/ca_program.html) — §1.3, §2.3. States no TLS algorithm or key-size requirement of its own
