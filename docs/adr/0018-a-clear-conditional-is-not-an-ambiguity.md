# ADR-0018: A clear conditional is not an ambiguity, and a grant is not a reading

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#46 May verge-asm index APNIC's bulk file, given its header bars storage in a retrieval system?](https://github.com/winniel123/verge-asm/issues/46)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Amends in effect:** [ADR-0003](./0003-third-party-source-consent-bar.md) — recorded here rather than as a third
  amendment to that file because sibling sessions were editing it concurrently. Reconcile on merge.

## Context

[#38](https://github.com/winniel123/verge-asm/issues/38) measured **657,044** uniquely-named APNIC
PA-equivalent assignments that no APNIC interface reaches by name at any consent tier — the RDAP
entity search returns HTTP 200 with an empty array on wildcard *and* exact `fn=`, and port 43 fails
on a name whose record it returns fine by address. The bulk file `apnic.db.inetnum.gz` is the only
instrument that reaches them.

Its header restricts reproduction and storage *"in a retrieval system"*. Indexing a dump so it can be
searched by name **is** storing it in a retrieval system, so the one path to the largest measured
unreachable population appeared to be closed by the file itself.

Two settled rules each nearly reach the case and neither was written with it in view.
[ADR-0003](./0003-third-party-source-consent-bar.md) limb 1 governs what the *software* inherently
does at runtime. [#27](https://github.com/winniel123/verge-asm/issues/27)'s redistribution rule
governs what *the project* ships. Holding a local copy of a third party's whole corpus and searching
it is neither, in the sense either rule was written for.

[ADR-0003](./0003-third-party-source-consent-bar.md)'s second amendment left the question open in
these words: *"A header that clearly bars us is not an ambiguity, and no toggle cures a plain no."*
That sentence contains the assumption this decision overturns — it treats *clear* and *barring* as
one thing.

## What the terms actually say

**[measured]** 2026-08-13. `https://ftp.apnic.net/apnic/whois/apnic.db.inetnum.gz`, 53,489,652 B,
`Last-Modified: Thu, 13 Aug 2026 15:02:41 GMT`, served over an unauthenticated HTTP `206` range
request. Its header, in full:

```
# Restricted rights.
#
# Except for agreed Internet operational purposes, no part of this
# publication may be reproduced, stored in a retrieval system, or
# transmitted, in any form or by any means, electronic, mechanical,
# recording, or otherwise, without prior permission of APNIC on
# behalf of the copyright holders. Any use of this material to
# target advertising or similar activities are explicitly forbidden
# and will be prosecuted. APNIC requests to be notified of any such
# activities or suspicions thereof.
```

The header does not define *"agreed"*. **The page it is reproduced on does.**
<https://www.apnic.net/manage-ip/using-whois/bulk-access/> carries this header verbatim under the
heading *"Restricted rights to the APNIC Whois Database"*, immediately below:

> *"Bulk access to whois data, including domain objects, is available under an Acceptable Use Policy
> (AUP). This restricts the uses to which whois data may be applied. **Requestors must sign the AUP
> agreement and lodge it with APNIC.** Contact the APNIC Helpdesk for assistance."*
>
> *"To request access to the Whois data for bulk download, please complete and return the APNIC whois
> data acceptable use agreement."*

And the linked copyright page — the `rel: "terms-of-service"` target of every APNIC RDAP and whois
response — glosses the carve-out and says who resolves it:

> *"Users will not be able to download the full contents of the database unless the intended use is
> for 'Internet operational issues'. These words are tightly defined and would include network
> trouble-shooting, abuse reporting, and Internet research and analysis. It would not include
> compiling marketing lists, demographic mapping, or any other commercial application."*
>
> *"Each request would be carefully considered in light of the APNIC Whois Database acceptable use
> agreement."* · *"Any request to pass on information to any other person or organization will be
> evaluated on the statement of purpose outlined by the requestor."*

So the carve-out is not a standing permission a stranger inherits, and it is not a puzzle either. It
is reached by a **named, published, per-requestor act**: sign the AUP, lodge it with APNIC.

## Decision

### 1. ADR-0003 limb 1 governs, reached through ADR-0012. #27's rule does not

The bulk file produces no observations — it yields candidate address scopes — so under
[ADR-0012](./0012-a-proposer-is-not-a-source.md) it is a **proposer**, and `consent` is the one
property that survives onto it. `consent` is ADR-0003's property, so ADR-0003 governs in full.

Limb 1 reaches the indexing directly, because limb 1's own words are *"automated querying, storing
results in a database, retaining them across runs"*. Storage is inside limb 1 by construction; no
stretching is required and no third rule is needed.

[#27](https://github.com/winniel123/verge-asm/issues/27)'s redistribution rule does **not** bite. It
binds on *"anything the project ships — reference data in the image, a bundled corpus, a vendored
dataset"*, and the project ships none of this: the operator's own install fetches the file.
[#34](https://github.com/winniel123/verge-asm/issues/34) already settled that exact boundary for the
consent record — *"the copy is made by the operator's own install, at their own acceptance, and is
never bundled or redistributed by the project."* The same sentence disposes of this. The copyright
page's own redistribution clause (*"may not be passed on in bulk to any other person or organization
unless approved by APNIC"*) is likewise satisfied and stays satisfied only while the product remains
single-tenant and self-hosted.

**What is new is not a rule but a key.** One registry now needs two different `consent` values for
two different instruments — the live registry path and the bulk file. `CONTEXT.md` already permits
this and says so: *"the same service can sit in two different states depending on how it is reached."*
`consent` keys on the instrument, never on the registry.

### 2. The header is clear — and clear in the permitting direction

*"Except for agreed Internet operational purposes"* is a **conditional prohibition with a published
route through it.** Nothing is in tension: the prohibition is plain, the exception is plain, and the
mechanism for reaching the exception is published on the same page as the header.

ADR-0003's ambiguity corollary is therefore **not triggered**, and
[#27](https://github.com/winniel123/verge-asm/issues/27)'s line applies verbatim: *clear text is not
ambiguity, so no email is sent; asking would be a request for an exception rather than a
clarification.* An ask from the project would be a request for a **standing blanket grant** covering
every future install of an AGPL tool — larger than [#23](https://github.com/winniel123/verge-asm/issues/23)
asked for, to the same registry, which did not answer.

**We are not the requestor and cannot become one.** The AUP is signed and lodged *by the party that
will hold the data*. That party is the operator. There is no question the project can usefully put to
APNIC, and no ticket to ask one.

**Technical availability is not permission.** The file is served anonymously, so the copyright page's
*"Users will not be able to download the full contents of the database unless…"* is factually false
of `ftp.apnic.net` today. That is a real tension and it does not create an ambiguity in the terms. It
is the mirror image of the move [#25](https://github.com/winniel123/verge-asm/issues/25) and ADR-0003
already refused — *unreadable is not absent, or any source could earn `unencumbered` by breaking its
own link.* Here: **an unenforced gate is still a gate, or any source could lose its restriction by
leaving the door open.** The terms travel *inside the file*; nobody can download it without receiving
them.

### 3. The tier is `operator-credentialed`, and `Consent` still gains no fourth value

`Consent` answers *whose permission does this run on*, and its three values are the three available
answers: nobody's is needed (`unencumbered`), the operator's own reading of terms the project
declined to read (`operator-accepted`), or **a permission the source actually granted to that
operator** (`operator-credentialed`).

A countersigned AUP is a grant, not a reading. Nothing here is ambiguous for an operator to adjudicate
and bear — APNIC either agreed with that requestor or did not.
[#34](https://github.com/winniel123/verge-asm/issues/34)'s own table already recorded the operative
row: *"is inventory an 'operational purpose approved by APNIC'? — **Yes — approval is per-party**."*
This is the first instrument measured in this project whose residual unknown varies by operator with
**no** constant component left underneath it, which is exactly what #34's axis says routes a case away
from `operator-accepted`.

`operator-credentialed`'s glossary wording is broadened by the smallest span that admits this: a grant
may be a countersigned agreement and not only an API key. The substance of the tier is unchanged — the
operator enters a direct relationship with the source, and the project's reading of the public terms
stops being the governing document.

### 4. The capability does not ship in v1

Ruled on ADR-0003's own rejected alternative, not on cost. The **declared-status bar** was rejected
because it *"invites operators to self-certify into permissions they may not hold"* and *"puts the
project in the position of having designed the loophole."* Every other `operator-credentialed` source
in the spec is structurally honest: without the credential the request fails. **`ftp.apnic.net` checks
nothing**, so an APNIC-bulk toggle would be a tickbox asserting a signed agreement the software can
never see, in front of a download that succeeds either way. That is the rejected alternative rebuilt,
with the project holding the pen.

Deferring is free. [#43](https://github.com/winniel123/verge-asm/issues/43) established that **a
proposer may be added or dropped in any release with no drift consequence whatever** — no `revealed`,
no `Break`, no version bump — so v1.1 may take this up with no rework and no comparability cost. The
land-grab argument is dead map-wide ([ADR-0015](./0015-the-value-space-is-the-commitment.md)), and a
proposer commits no value space.

This generalises to the instrument class: **no third-party bulk registry dump enters v1**, which also
disposes of the RIPE and AFRINIC dump terms that `docs/research/asn-less-operators.md` §13.9 recorded
as unassessed. It does not touch RIPE's `fulltextsearch`, which is a live path.

### 5. The fallback message names a missing capability, no figure and no route

Per [#28](https://github.com/winniel123/verge-asm/issues/28) the propose half of the source axis
carries **no percentage, no bar and no `unknown` in a cost column, ever**, so **657,044 may not be
quoted as a coverage loss** — it is a count of strings in a file, and the estate it would be measured
against is the one thing only the operator can declare.

The message also names **no bulk route**. [#22](https://github.com/winniel123/verge-asm/issues/22)
cut the coverage screen's second axis as an *invitation with a one-click enable*; a route requiring a
signed legal agreement with a registry and a capability the product does not have is not an
invitation, and rendering it as one would be an offer the screen cannot honour. It belongs in
documentation.

## Consequences

- **APNIC keeps exactly one toggle in v1**, the live registry path at `operator-accepted`
  ([#34](https://github.com/winniel123/verge-asm/issues/34)'s table is untouched). The bulk file is
  not a second row on the enablement surface, which is a **convergence for
  [#47](https://github.com/winniel123/verge-asm/issues/47)**: nothing is added to what APNIC's prompt
  renders.
- **[#38](https://github.com/winniel123/verge-asm/issues/38)'s two honest missing-capability
  annotations stand at two**, APNIC and AFRINIC for PA renters, and APNIC's is now final rather than
  pending a licence question.
- **The clear/ambiguous fork has three positions, not two.** Clear-and-barring (a plain no, which no
  toggle cures), clear-and-conditionally-permitting (the condition is somebody's to satisfy — ask
  *whose*), and genuinely in tension (ADR-0003's corollary). A session meeting a carve-out should look
  for a published mechanism before reaching for the ambiguity corollary; here the answer was one page
  away from a document this project had already quoted twice.
- **A fifth quotation defect, and the first that is a truncation.**
  [#21](https://github.com/winniel123/verge-asm/issues/21)'s habit has now paid five times.
  `docs/research/asn-less-operators.md` §13.5 quoted the header accurately for two sentences and
  stopped, dropping the advertising and notification sentences — and, more consequentially, quoted the
  header without the page that defines its only operative term. **Verifying a quote against its source
  is not the same as verifying it against its context**; a clause quoted out of the document that
  qualifies it can be word-perfect and still wrong about what it permits.
- **The two APNIC texts are not identical and both are live.** The file header says *"agreed Internet
  operational purposes"*; the copyright page says *"Internet operational purposes approved by APNIC"*
  and adds a bulk-redistribution sentence the header lacks. Nothing here turns on the difference, but
  a session quoting one should not represent it as the other.
- **§4's reasoning is a property of the tier, not a fact about one instrument** —
  [ADR-0023](./0023-consent-names-the-door.md). *"Without the credential the request fails"* is what makes
  `operator-credentialed` honest anywhere, so an unenforced grant may not carry the value at all, and the
  live registry path can no more hold it than the bulk file can. §3's broadening to a token-less grant
  stands as a definition and has **no v1 instance**.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Read the header as a plain no** — limb 1 fails *"regardless of who the operator is"*, so the instrument is dead | This is ADR-0003's second amendment's own assumption and it is wrong on the text. It reads the prohibition and skips the carve-out, and #34 has already scoped that sentence to **defaults**, not to whether a capability may exist behind an opt-in. It would also delete a capability on a reading APNIC's own published process contradicts |
| **Rule it ambiguous and draft an ask** — the position `docs/research/non-arin-prefix-coverage.md` §8.1 took | §8.1 asked *"must each deployment seek approval individually?"* and APNIC's bulk-access page answers it in published text: yes, by AUP. The finding was formed without that page. Asking anyway would be a request for a blanket exception, which #27 refused as a category, from a registry that has already not answered #23 |
| **#27's redistribution rule governs** | It binds on what the **project** ships and we ship nothing. #34 settled the identical boundary for the consent record: a copy made by the operator's own install is not redistribution |
| **A third rule neither ADR covers** | Tempting, and the ticket invited it. But limb 1 already names *store* and *retain*, and ADR-0012 already routes a proposer's `consent` to ADR-0003. Inventing a rule to cover an act two rules already reach would leave three instruments where two suffice |
| **`operator-accepted`, following #34's table row for the live path** | The value means *the operator takes on a reading the project declined to make*. There is no reading here — the text is clear and the operator's act is to sign a form, not to adjudicate a tension. Filing a grant under a reading is precisely the overloading #34 diagnosed, one value further along |
| **A fourth `Consent` value** for a token-less grant | Refused by #34 on the ground that survives here unchanged: the three values already answer *whose permission does this run on*, and a fourth would be a second property wearing an enum case's clothes |
| **Ship it behind an `operator-credentialed` toggle in v1** | The credential is not machine-checkable and the download is not gated, so the toggle is a self-certification in front of a door that is already open — ADR-0003's rejected declared-status bar, rebuilt by us. #43 prices deferral at zero |
| **Quote 657,044 in the fallback message** | Forbidden by #28, and independently unsafe: §13.9 records that once-only `descr:` strings are a proxy and the count of names an operator would actually type is materially lower and unmeasured |
