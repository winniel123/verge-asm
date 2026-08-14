# ADR-0040: A specification's silence is not the owner's silence — and a scope weakness is disclosed as a searched corpus, never as a caveat

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#73 Is there an unscoped owner statement of a certificate key-size floor, or is 112 bits permanently NIST's?](https://github.com/winniel123/verge-asm/issues/73)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md) §7 named a **third kind of
weak row** — a *scope* weakness, where the owner has spoken but scopes its own normative verbs — and
predicted its resolution route:

> **A scope weakness is a third kind, and it behaves like neither of the other two.** It cannot flip
> silently, so it is not watched. It cannot be chased by finding the right party, because the right
> party has already spoken. It resolves only if a **second** owner speaks unscoped — so its retrieval
> is a search of a corpus rather than a request to a party.

[`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §9.2 named that corpus search and
recorded the negative it was standing on: RFC 5280, RFC 8446, RFC 5246 and BCP 86 set no certificate
key-size floor, so the three key rows rested on NIST alone, whose §1.2.2 defines *shall* as *"a
requirement for Federal Government use"*.

#73 ran the search. **It came back positive**, and the way it came back positive is what this ADR is
for: the resolving statements were not in some unread corner of the corpus. Two of the three were in
the **IETF**, a body already carrying two rows of the same table, and one of them was in an **appendix
of RFC 8446** — a document §9.2 cited by name as setting no floor.

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §7 requires that a
disclosed weakness name the retrieval that resolves it *"or it is a permanent caveat and decays into
decoration"*, and its #67 amendment added that **a performed retrieval may not leave a row pointing at
itself**. #67 discharged that for a retrieval that came back positive on its whole question. This one
came back positive on two rows and empty on a third, which is the case neither ADR had.

The full working, with the enumerated corpus and every quote checked against retrieved bytes, is
[`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §13.

## Decision

| Concern | Decision |
|---|---|
| Does an unscoped statement of a certificate key-size floor exist | **Yes — three, all IETF, in three different document classes.** RFC 9325 §4.5 (BCP 195, `MUST`), RFC 9846 §C.2 (`SHOULD`, both numbers, *"not appropriate for secure applications"*), RFC 8550/8551 §6 (*"cryptographically insecure"*) |
| Is 112 bits permanently NIST's | **No, and the honest answer is narrower than the question.** The IETF states 112 unscoped in RFC 9325 §4.2 — over **cipher suites**, citing a document that disclaims recommending it. **The numbers this table keys on are 2048 and 224**, and those carry unscoped IETF footing in their own right |
| Why #68's negative failed | **It searched a document *set* where it should have searched a document *class*.** A body's specification, its deployment BCP and its implementation guidance disagree by design, and #68 read only the first |
| The general rule | **A specification's silence is not the owner's silence.** A negative over specifications is a negative about that class, and must be recorded as one |
| What a scope weakness is allowed to be | **A searched corpus with a stated boundary** — never a permanent caveat, never a standing promise. Once run, the disclosure carries the corpus, the result, and the smallest extension that could still move it |
| Is a surviving scope weakness *permanent* | **The word is refused.** A weakness that survives its search is **bounded**, which is falsifiable by naming a document outside the boundary — the property a permanent caveat lacks |
| What moves | **A footing, not a row**, exactly as [#69](https://github.com/winniel123/verge-asm/issues/69) moved one. Five rows, same integers, same predicate. The weak tier goes from **three rows to one limb of one row** |
| Does this break #37's precedent | **No.** The rows move on **RFC 9846** (July 2026) and **RFC 9325**, neither held by #68. That RFC 8446 also carries the sentence is recorded as a **correction of what a document says**, which is not the same act as re-interpreting one |
| Does the rule's output change | **No.** No floor moves, no predicate moves, no version moves. This is a change to the table's *footing* and to its *disclosure* |

## Rationale

### 1. The failure mode has a name, and it is not "we did not look hard enough"

#68 was not lazy. It read the certificate profile (RFC 5280), both TLS specifications (RFC 8446, RFC
5246), the RSA specification (RFC 8017) and the strength-estimation BCP (RFC 3766), and concluded the
IETF sets no certificate key-size floor. Every one of those is a **specification** or a **method**.

A standards body does not put the same content in every document class, and the split is principled
rather than accidental:

| Class | What it does with a number | Instance |
|---|---|---|
| **Specification** | Declines to fix one, because cryptanalysis will move it and the specification will outlive the number | RFC 5280 §8: *"Short key lengths or weak hash algorithms will limit the utility of a certificate. CAs are encouraged to note advances in cryptology"* |
| **Deployment recommendation** | Fixes one, and says so — a BCP is a point-in-time statement by design | RFC 9325 §4.5: *"servers MUST authenticate using certificates with at least a 2048-bit modulus"*, and §1: *"this document provides a floor, not a ceiling"* |
| **Implementation guidance** | Illustrates one, in an appendix, usually as an example | RFC 9846 §C.2: *"For example, certification paths containing keys or signatures weaker than 2048-bit RSA or 224-bit ECDSA are not appropriate for secure applications"* |

RFC 5280's silence is therefore **information about the document class**, not about the IETF. ADR-0035
§1 read that silence correctly as *"silent by design"* and then read one word too much into it — it
concluded the silence *"points somewhere"* else, when the same body says the number two document
classes over.

> **A negative about an owner must enumerate the owner's document classes, not its documents.** Before
> a table records that an owner sets no number, it asks the defining specification, the operational or
> deployment recommendation, and the implementation guidance. A negative over one class is a finding
> about that class and must be written as one.

**This is checkable and it is cheap.** The rule would have cost #68 one additional retrieval — BCP 195
is the IETF's own TLS deployment document and the first thing anyone deploying TLS is pointed at.

### 2. The corpus was never too small; it was the wrong shape

#73's search covered roughly **340 documents**: 63 LAMPS RFCs and 17 active drafts, ~180 IAB
statements, the 247 RFCs carrying an *Also BCP* designation filtered to the security area, and the TLS
and PKIX lines individually. **Exactly one BCP sets a numeric public-key floor**, and it is the one a
session would have guessed third.

The rest of the sweep is worth its cost for a different reason: it establishes the negatives that
remain, and one of them is exemplary. **RFC 1984 (BCP 200) has a section headed `KEY SIZE` and sets
none** — it argues that key sizes must not be legislated, and the string `2048` does not appear
anywhere in it. That is the body one would ask for an internet-wide key-size position having a section
of exactly that name whose content is the opposite of one. It is `sensitive-ports.md` §2.7's citable
non-statement at its strongest, and it is now on the record so that no later session retrieves the
heading and stops reading.

### 3. What a scope weakness is allowed to be

ADR-0032 forbids a permanent caveat. #67 added that a performed retrieval may not leave a row pointing
at itself. Neither covers the case here — a search that came back **positive for two rows and empty for
a third** — and the general statement is owed because #73's charter asked for one.

> **A scope weakness is disclosed as a searched corpus, never as a caveat and never as a promise.**
> Before its search has run, the disclosure names the corpus it will search. **Once run, the
> disclosure carries three things and nothing else:** the corpus actually searched, enumerated; what
> was found and which rows it reached; and the **smallest extension of the corpus that could still
> change the answer**.

> **A weakness that survives that search is not *permanent*. It is *bounded*.** The distinction is not
> a euphemism and it is the whole point: *permanent* is unfalsifiable, which is why ADR-0032 forbids
> it; *bounded* is falsified by anyone who can name one document outside the boundary. A bounded
> weakness invites a specific, cheap act. A permanent one invites nothing, which is what decoration
> means.

Applied to the row that survives — DSA's `N ≥ 224` limb — the boundary is named rather than left to
judgement, and naming it produced a finding of its own. The enumerable candidates are the bodies that
specify DSA domain parameters outside NIST, and **every one of them reproduces NIST's shape in a
different flag**: a recommendation addressed by a body to its own constituency. So:

> **A key-size floor is the kind of statement a body makes to a constituency, and the IETF is unusual
> in having one that is "the internet".** A scope weakness on a cryptographic parameter is therefore
> **structural rather than accidental**, and the set of bodies that can resolve one is small and
> enumerable rather than open. That is why its corpus can be bounded at all, and it is the reason this
> kind of weak row is tractable where a general *somebody may have said this somewhere* is not.

### 4. Moving the row without breaking #37

#37's precedent binds map-wide: **a row moves on retrieval, never on a re-reading of text already
held.** It has now decided #66 (a row came off on retrieval), #67 (a number moved on retrieval), and it
constrains this ticket, whose charter restated it and put NIST's Rev 3 explicitly out of scope for the
same reason.

The rows move on **RFC 9846** — a July 2026 Standards Track publication that obsoletes RFC 8446, RFC
5246 and RFC 8422, which #68 could not have held — and on **RFC 9325**, which #68 never read. Both are
retrievals in the plainest sense.

**The awkward part is recorded rather than smoothed.** RFC 8446 Appendix C.2 carries the same sentence
byte-for-byte and was held. Read at its most literal, #37's precedent would forbid noticing. It does
not, and the distinction is worth fixing:

> **Correcting what a held document *says* is not re-reading what it *means*.** #37's precedent
> protects against a verdict changing because a session re-interpreted evidence it already had. A
> transcription error — a note recording that a document sets no floor when it sets one on a page
> nobody opened — is a **defect**, and #37's own charter was repairing defects. The two are told apart
> by asking whether the new claim is about the document's **content** or about its **import**.

Here the claim is about content: RFC 8446 §C.2 exists, and §2.3.1's *"RFC 8446 and RFC 5246 set none"*
is false. The row's move does not need that correction — RFC 9846 carries it alone — but the false
sentence could not be left standing, and a precedent that would have required leaving it would be the
precedent doing damage.

### 5. What did not move, and the four traps that made it look as though it had

Recorded because the gap between the finding and its near-misses is where the next session will land.

- **DSA's `N` limb has nothing unscoped.** RFC 8550 §6 and RFC 8551 §6 reach `L ≥ 2048` and say nothing
  about `N ≥ 224`. RFC 9846's *"MD5 [SLOTH], SHA-224, and DSA MUST NOT be used"* is a **withdrawal of
  approval** arriving from a second body — the candidate third claim ADR-0035 §5 already refused, and
  it is refused again for the same reason.
- **The right number in the wrong sentence.** RFC 9325 §4.5 says *"Curves of less than 224 bits MUST
  NOT be used"* — this table's ECDSA number exactly — under an antecedent about **ECDH key agreement**,
  justified by a **key-agreement** standard, and filed in the RFC's own changelog as *"ECDH minimal
  curve size is 224"*. A grep for `224` finds it before it finds the sentence that is actually about
  certificates.
- **Curve deprecation on popularity.** RFC 8422 §5.1.1 removes every curve below P-256 and states its
  reason as *"Only three have seen much use."* The effect is a floor; the claim is a deployment
  measurement. Founding a row on it would be *frequency is not a position* failing in the source's own
  words rather than in ours.
- **The 112 that traces to a disclaimer.** RFC 9325 §4.2's unscoped `MUST NOT` below 112 bits of
  security cites RFC 3766, which says *"this doesn't mean that 112 bits is recommended. In fact, 112
  bits is arguably too strong for any practical purpose."* The norm survives its rationale, but the
  number's pedigree is thinner than it looks — and **no row keys on it**, because ADR-0035 §6 already
  required the table's key to be `(algorithm, parameter)` and never a bare bit count. That requirement
  was made for a different reason and paid out here.

## Consequences

- **The weak tier is one limb of one row.** RSA leaves it on four unscoped statements; ECDSA leaves it
  for SHA-1's tier, strong with a **modality** softness that must not be re-filed as a scope one; DSA's
  `N ≥ 224` limb stays, bounded, and ships disclosed exactly as `5432/tcp` does.
- **[ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md) §7 is confirmed and
  narrowed.** The prediction that a scope weakness resolves only through a corpus search held on its
  first test. What is added is that the corpus is enumerated by **document class**, and that the set of
  bodies able to resolve one is small — so the third kind of weak row is tractable rather than open.
- **[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)'s disclosure
  obligation gains a third limb.** Its first: name the retrieval. Its #67 amendment: once performed,
  say what it established. This ADR's: **once performed and partly empty, carry the enumerated corpus
  and the smallest extension that could still move it, and call the residue bounded rather than
  permanent.**
- **Three citations in `weak-key-and-signature.md` and one in ADR-0035 are stale, and one is wrong.**
  RFC 5280's Security Considerations is **§8**; §11 is *References*, and both documents say §11. RFC
  8446 is obsoleted by **RFC 9846**, whose section numbers moved — §4.4.2.4 → §4.5.1.3 and §4.2.3 →
  §4.3.3. The corrections are carried in `weak-key-and-signature.md` §13.7 under the name-and-withdraw
  convention; **no row's grounds change**.
- **The map's curation patch gains a cadence it had not priced.** This table's watch was *re-read SP
  800-131A when a revision goes final and otherwise do nothing*. RFC 9846 landed in July 2026 and
  obsoleted the document two of the five rows cite, four weeks before this was written. The **content**
  cadence is still about one edit per row per decade; the **citation** cadence is faster, and only the
  first was ever measured.
- **`sensitive-ports.md` needs no amendment for this ADR** and was not edited — [#70](https://github.com/winniel123/verge-asm/issues/70)
  holds it. §10.6's two kinds of weak row and ADR-0035's third are unchanged; what changes is what a
  disclosure of the third must contain, which lives here and in the instrument's own document.
- **`CONTEXT.md` is not edited**, on ADR-0032's and ADR-0035's precedent and for their reason. Nothing
  here changes a glossary term.
- **The `certificate` facet's owner set is unchanged.** No new party is admitted. Every statement this
  ADR rests on is the IETF's, and the IETF was already an owner under ADR-0035's clause for *what a
  certificate or a TLS handshake may carry*.

## Alternatives rejected

| Alternative | Why not |
|---|---|
| **Hold all three key rows in the weak tier, because the resolving sentences are `SHOULD` or sit in an appendix** | Confuses two axes. §9's weak tier is about **scope** — a verb addressed to an audience the row is not about — and RFC 9325's and RFC 9846's verbs are addressed to nobody in particular. Modality softness is what the SHA-1 row already ships with, outside the weak tier; holding ECDSA in on a `SHOULD` would apply two standards inside one table |
| **Move the DSA row out too, on RFC 8550/8551's *"RSA and DSA keys of less than 2048 bits … cryptographically insecure"*** | Reaches `L` and not `N`. The row's floor is `(L, N)`, and half a footing is not a footing. Recording it as resolved would be the open-set failure #37 closed, arriving as generosity |
| **Refuse the move entirely on #37's precedent, because RFC 8446 was held** | Would require leaving a demonstrably false sentence — *"RFC 8446 and RFC 5246 set none"* — standing in a research note, to protect a precedent aimed at a different failure. The rows move on RFC 9846 and RFC 9325 regardless, so the refusal would buy nothing and cost a known error |
| **Carry RFC 9325 §4.5's *"Curves of less than 224 bits MUST NOT be used"* as the ECDSA row's footing** | It is about **ECDH key agreement**, on the RFC's own antecedent, its own cited authority and its own changelog. It matches the row's number and not the row's artefact, which is ADR-0035's artefact test failing in the most seductive way available |
| **Carry the CA/B BR now that a second body agrees with it** | Route (c) was refused on the artefact, not on the arithmetic, and agreement changes neither. A body specifying which primitives a population may *accept* is still in the distributor position |
| **Record the residual DSA weakness as permanent and close the question** | ADR-0032 forbids it, and it would be false: the boundary is nameable, so the weakness is bounded. *Permanent* would also be the more comfortable word, which is the reason to distrust it |
| **Open a ticket to chase ANSI X9 or BSI TR-02102 for the DSA `N` limb** | Each reproduces NIST's shape in a different flag — a body addressing its own constituency — so the retrieval is predictably empty, and the row it would move is the one §9 already records as least likely ever to fire. Named as the boundary; not worth a slot |
| **Fold this into ADR-0035** | ADR-0035's subject is *how to place a source when the artefact is not a protocol*. This ADR's is *where to look before recording that an owner is silent, and what a disclosure may say once its search has run*. The second is about the evidence standard's procedure and reaches every table under gate 2, not only this one |
