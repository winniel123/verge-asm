# ADR-0175: a certificate's identity is compared as presented — a DN byte-exactly, and only `dNSName` SANs participate

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1342 ADR gaps: cmd/web/signals.go](https://github.com/winniel123/verge-asm/issues/1342), gaps 1 and 2
- **Sweep PR that deleted the comment:** [#1341](https://github.com/winniel123/verge-asm/pull/1341)
- **Rests on, by analogy only:** [ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md) and
  [ADR-0060](./0060-a-wildcard-san-is-a-pattern-over-names-and-admits-none-of-them.md), which supply the
  posture — *an identifier a specification leaves two readings of is refused, not interpreted* —
  applied there to a subject key and to a wildcard SAN. **Neither reaches a certificate DN and
  neither reaches an `iPAddress` SAN.** The deleted comment cited them as though they did
- **Not bound by:** [ADR-0163](./0163-an-absent-certificate-material-row-is-a-fan-out-of-zero-and-is-reached-and-only-an-absent-measurement-row-is-pending.md),
  which rules the **absence** of certificate material in the edge-fan-out path; this rules the
  **comparison** of chain material that is present, on the console read path
- **Bounds:** [`weak-key-and-signature.md`](../research/weak-key-and-signature.md) §4.1's *"the two
  rules read the same fact and cannot disagree about it"*, which holds where both evaluate and is
  silent where one abstains (§2)

## Context

`cmd/web/signals.go` carried a block above `selfSignedOf` asserting the DNs are compared *"exact
byte-for-byte string equality — normalisation refused, ADR-0051/0060"*, and one above
`sanMatchesName` asserting the match *"ORs over san_dns only — san_ip is ignored entirely (#714
§3)"*. #1341 deleted both.

**Both citations are wrong, and `comment-policy.md` §4.7 puts a wrong citation below a dead one**
because it survives a file-existence check. ADR-0051 rules that an `Address`'s natural key is the
address rather than the text, that comparison is *"octet equality, family-matched"*, and that a key
normalisation is fixed at v1; it names an `iPAddress` SAN once, in a table of *where a text would
come from* for an address key. ADR-0060 rules that a wildcard SAN denotes a set and admits no
`Name`, and its only normalisation content is the refusal to read octet `0x2A` in a `dNSName` SAN.
Neither reaches a certificate DN. `#714` item 3 is *"Leaf only: SANs are the leaf's identity"*; the
IP question is its item 2, which poses rather than rules.

**What is written covers neighbours only.** `weak-key-and-signature.md` §4.1 states the
shared-predicate half and nothing about byte-exactness; the sweep kept that citation at
`cmd/web/signals.go:1084`, correctly. `docs/spec/golden-corpus.md` §10.3 states that an `iPAddress`
SAN *"raise[s] the count by zero"* — but in the shared-edge reduction, a different path, whose
`internal/queue/edgefanout.go:299-310` returns `cert.DNSNames` alone. A search of `docs/spec/`,
`docs/adr/`, `docs/guides/`, `docs/research/` and `CONTEXT.md` returns nothing for either rule here.

**The code, verified in this tree.** `selfSignedOf` is `cmd/web/signals.go:1083-1086` and its body
is `return subject == issuer && selfSigVerifies`. The two strings are not DER: they are
`pkix.Name.String()` renderings taken at measure time (`internal/measure/connectoutcome/tls.go:203`,
`:204`), carried as `chain_certs[].subject` and `.issuer` and decoded at `cmd/web/signals.go:1029`.
That rendering discards the ASN.1 string type and emits the nine standard attribute types in a fixed
order, so it is neither raw bytes nor RFC 5280 §7.1. `selfSigVerifies` does come from
`CheckSignatureFrom` — `tls.go:201` — captured in-leaf and stored at `:205`. Two callers, both in
this file: `certDetailsFromValue` on `chain_certs[0]` at `:1076`, and `weakKeyOrSignature` per link
at `:1171`. There is no third outside the test.

**`san_ip` is captured and read by nothing.** `tls.go:165-168` renders every `leaf.IPAddresses`
entry, `certificate.go:63` emits it, `cmd/web/signals.go:1022` decodes it — and no expression reads
that field. `certcorpus/rows.go:293-318` pins the shape as `cert_v3_san_ip_only.ndjson`, whose own
claim says the read side *"ignores san_ip entirely"*.

## Decision

> **A certificate's identity is compared as it was presented. Issuer DN and subject DN are compared
> by exact string equality over the recorded rendering, and RFC 5280 §7.1 string preparation is
> refused. The hostname match ORs over `dNSName` SANs alone, so an `iPAddress` SAN never contributes
> a match. Both rules read one predicate, so they cannot disagree about the same certificate.**

### 1. The DN comparison is exact, and normalisation is refused

RFC 5280 §3.2 supplies the definition: *"Self-issued certificates are CA certificates in which the
issuer and subject are the same entity … Self-signed certificates are self-issued certificates where
the digital signature may be verified by the public key bound into the certificate."* Two conjuncts,
and `selfSignedOf` is exactly those two.

§3.2 does not say what *the same entity* means, and §7.1 says two things at once. It requires *"the
LDAP StringPrep profile (including insignificant space handling) … as the basis for comparison"*,
requires `caseIgnoreMatch`, makes support for other equality rules *optional*, and makes RDN
matching order-free within an RDN: *"Two relative distinguished names RDN1 and RDN2 match if they
have the same number of naming attributes and for each naming attribute in RDN1 there is a matching
naming attribute in RDN2."* A floor is fixed and the ceiling is left open, which is two readings of
*the same entity* for any pair that differs only under preparation.

**We take the narrow one.** `subject == issuer` on the presented rendering: no case folding, no
whitespace compression, no Unicode normalisation, no attribute-aware matching. This is ADR-0051's
refusal test applied one object over. It is not ADR-0051's rule, because a DN is not a subject key
here — nothing is keyed on it and no timeline moves when it moves.

### 2. One predicate, and where the two rules part

Both callers pass the same two strings and the same `self_sig_verifies`, so where both evaluate,
`certificate-self-signed` and the signature-limb skip read one answer. They part in one place, which
§4.1's *cannot disagree* does not cover: at `:1075` `certificate-self-signed` **abstains** when
`self_sig_verifies` is absent, leaving `SelfSigned` nil and the rule `not-evaluable`, while `:1170`
folds the same absence to `false` so the signature limb does **not** skip. That asymmetry is
intended and it is the loud direction — an absent datum abstains where it is the whole rule, and
fires where it would otherwise excuse a weak signature.

The populations differ too, and are meant to: `certificate-self-signed` reads the leaf alone, the
signature limb reads every link. The ruling is that they cannot disagree **about the same
certificate**, not that they range over the same set.

### 3. Only `dNSName` SANs participate

RFC 6125 §6.4 is *Matching the DNS Domain Name Portion*, and every rule beneath it — §6.4.3 rule 2's
*"`*.example.com` would match `foo.example.com` but not `bar.foo.example.com` or `example.com`"*
included — is stated over labels of a domain name. §1.7.2 puts the rest out of scope by name:
*"Identifiers other than fully qualified DNS domain names. Some certification authorities issue
server certificates based on IP addresses, but preliminary evidence indicates that such certificates
are a very small percentage (less than 1%) of issued certificates."* The specification supplies a
procedure for one identifier class and declines the other; we implement the class it supplies.

### 4. The nameless endpoint asks no question at all

An `Endpoint` is a `(Name, Service)` pair (`CONTEXT.md` **Endpoint**) whose `Name` leg is a
fully-qualified domain name or the distinguished absent variant. It is never an IP literal. Two
shapes are reachable:

| Endpoint | Certificate | What happens |
| --- | --- | --- |
| Named | Only `iPAddress` SANs | `sanMatchesName` ORs over an empty set, `SANMatchesName` is `false`, `certificate-hostname-san-mismatch` **fires** |
| Nameless | Only `iPAddress` SANs | `HasName` is false (`cmd/web/signals.go:995`), so `Eval` returns `OutsideDomain` (`internal/signal/endpoint.go:156-158`) and the rule never runs |

The first is correct under §6.4: a DNS reference identifier with no DNS-ID to match does not match.
The second is where `san_ip` would have had work, and it is where the rule declines to have a
domain. **An address-keyed endpoint asks no hostname question, so a captured `san_ip` answers none.**

## Consequences

- **A genuinely self-signed certificate can be missed, and the residue is smaller than it looks.**
  Go's rendering already erases the ASN.1 string type and orders the nine standard attributes, so a
  PrintableString-versus-UTF8String issuer is **not** a divergence here. What survives is case,
  insignificant whitespace, Unicode normalisation form, a multi-valued RDN, and an unknown-OID
  attribute, rendered `oid=#<hex of the DER>`. For one certificate both fields are written by one
  tool from one template, and every self-issued corpus row is byte-identical
  (`certcorpus/rows.go:368`, `:422`, `:450`). **Not reached in this corpus** — and an open risk
  rather than a bounded cost, because a cross-signed or hand-built root can produce it and nothing
  detects it when it does.
- **A certificate issued for an IP address always mismatches a named `Endpoint`**, and RFC 6125
  §1.7.2 bounds that population under 1% of issuance. `cmd/web/signals_cert_test.go:30` pins it:
  *"no dNSName SANs (IP-only leaf) → false, never nil-defaulted true"*.
- **`san_ip` is dead weight on the read side and stays.** Deleting the decode moves a golden;
  keeping it preserves the evidence a later address-identity rule would need.
- **The byte-exactness is unpinned by test.** `TestSelfSignedOf`
  (`cmd/web/signals_cert_test.go:47-67`) has four rows and none differs only by case. A row
  asserting `"CN=Root"` against `"CN=root"` reads `false` and belongs there.
- **A defect in the second conjunct is exposed and not fixed here.** `CheckSignatureFrom` is
  stricter than a signature check: Go rejects an MD5 or SHA-1 signature outright, and rejects a
  parent that is not a CA. A real SHA-1 self-signed root therefore records `self_sig_verifies=false`
  and the signature limb does **not** skip it — the exact carve-out §4.1 argues for. The two rules
  still agree; they agree on a wrong answer. Reported for its own ticket.
- **Nothing is superseded**, so ADR-0058's obligation falls due nowhere. §4.1 is bounded rather than
  corrected: §2 adds the case it does not reach.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Full RFC 5280 §7.1 normalisation before comparing** — LDAP StringPrep, `caseIgnoreMatch`, order-free RDN matching | It cannot be built from what we hold. §7.1's algorithm is defined over **attributes** with per-type equality rules, and the comparison runs on renderings (`tls.go:203-204`) that have already flattened multi-valued RDNs and dropped the string type. Doing it correctly means carrying `RawSubject` and `RawIssuer` DER through the wire and the store, which moves the `certificate` value space and re-escrows every golden. It also plants a normaliser inside a rule's predicate whose output moves on a Go or ICU revision with nothing in the world having changed — ADR-0021's test failed in the place ADR-0051 spends its argument protecting |
| **Case-insensitive comparison as a middle path** — `strings.EqualFold(subject, issuer)` | The worst of the three positions. §7.1's floor is StringPrep **and** `caseIgnoreMatch` **and** order-free RDN matching, so folding case alone implements no reading of the specification: stricter than §7.1, looser than the presented bytes, and a reader cannot say which certificates it is right about. `EqualFold` is Unicode simple folding over the rendered string, so it would fold inside an `oid=#hex` blob and compare hex digits case-blind. A rule that is neither available reading is the interpretation ADR-0060 refuses, dressed as a compromise |
| **Admit `iPAddress` SANs for an address-keyed endpoint** — match `san_ip` against the `Service`'s address | There is no reference identifier to match. A nameless `Endpoint` is `CONTEXT.md`'s *default response to a client that names nothing*, and the handshake sends no SNI, so the address we dialled is our routing decision and not an identity the peer was asked to prove. Matching against it invents a claim the exchange never made, and fires `certificate-hostname-san-mismatch` on every correctly-configured address-scope probe of a name-based virtual host. It also needs ADR-0051's key form to compare an address, so the rendered `san_ip` string is the wrong object and a second address normalisation site enters the model — the seam ADR-0051 closed |
| **Fold `san_ip` into `san_dns` at the producer**, so one list is matched | It destroys a distinction the wire carries: RFC 5280 §4.2.1.6 gives `iPAddress` an OCTET STRING and `dNSName` an IA5String, and a leaf presenting `192.0.2.10` as a `dNSName` is a different, worse certificate. It also silently changes the shared-edge reduction, which counts registrable domains over `cert.DNSNames` (`internal/queue/edgefanout.go:310`) and whose fixtures assert an `iPAddress` SAN adds zero (`golden-corpus.md` §10.3) |
| **Drop the DN conjunct and read `self_sig_verifies` alone** | The conjunct looks redundant — a self-signature verifies on a non-self-issued certificate only by accident of key reuse — but it is what makes the predicate implement RFC 5280 §3.2 rather than *the leaf's key signed the leaf*, and it is the half that does not depend on a library call whose strictness has moved and will move again (see Consequences) |
