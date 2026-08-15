# ADR-0098: A claim is attested by the registrant, never by the registry

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#178 Can an IANA registry service description alone carry a claim, or does the attestation gate need more?](https://github.com/winniel123/verge-asm/issues/178)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

`sensitive-ports.md` §2.2 names exactly three forms that may attest a curated-list claim: the
protocol's **specification** (an RFC or equivalent), the project's or vendor's own **documentation**,
or the project's **shipped default**, as documented. Three of Class B's rows — `512/tcp` rexec,
`513/tcp` rlogin, `514/tcp` rsh — were filed at §43.3's register (and transcribed into
[`curated-table-watch.md`](../spec/curated-table-watch.md) §1.1) as resting, as a group, on "the IANA
Service Name and Transport Protocol Port Number Registry's own service descriptions" — a bare
catalogue entry, never independently checked against any of the three named forms.

§43.6 item 3 named the question and declined to decide it: *"whether a registry description can carry
a claim at all is a §2.2 attestation question this section may not decide."*
[ADR-0048](./0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md) was checked first and
does not settle it. ADR-0048's own rider — *"a registration reaches the limbs only through its
registrant… an IANA row is a record of a registrant's own placement declaration, not an independent
authority"* — is written for §2.4's **determinacy** gate, and ADR-0048's own Rationale 1 explains why
that does not travel automatically: determinacy asks a **conclusion** question, *what listens here*,
answered **at** the wire; attestation (like the claim gate beside it) asks a **normative** question,
*may this statement be made, and by whom*, answered **after** the wire. Different instruments, per
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §6. Landing at the same
verdict for the same party does not mean arriving there by the same rule, and §2.2 owed its own
answer.

The full working is [`sensitive-ports.md`](../research/sensitive-ports.md) §43.10.

## Decision

**A registry entry does not attest a claim on its own word. It attests only where it can be traced to
a registrant, designer, or implementer speaking in their own voice, in a form §2.2 already admits.**

1. **The registry fails §2.2 on genre.** A specification defines protocol behaviour; a registry
   records an assignment, ordinarily a terse description supplied by the registrant at filing time and
   never revisited for currency or accuracy. IANA's own registry-procedures document, RFC 6335, does
   not claim to specify what it catalogues. A registry row sharing a page with a specification is not
   thereby "an RFC or equivalent," and it is neither of §2.2's other two forms either — it is not a
   project's or vendor's own documentation of its own protocol, and it is not a shipped default.
2. **The registry fails §2.2 on ownership, for the reason ADR-0048 already gave the sibling gate.**
   §10.5's *owner* is "the party that designed the protocol, or that authors the reference
   implementation, speaking about the thing it designed or wrote." IANA does neither for a registered
   service — it is a registrar, transcribing a registrant's own declaration and disclaiming, in
   capitals, any further authority over it. The defect is identical to the one ADR-0048's rider found;
   this decision applies it to §2.2 rather than assuming §2.4's ruling already covered it.
3. **The two grounds are independent of volatility.** A registry entry is also a **rung-1** artefact
   under [ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) — a dataset whose
   entries change with no notice a reader would meet — and that is a real, separate hazard the queue
   already tracks. It is not, however, the reason the registry fails §2.2: a hypothetical rung-1
   artefact that genuinely were a registrant's own specification would still attest. Rung explains why
   an admitted registry-adjacent claim would need watching; it does not do the admitting.
4. **A cell resting on the registry alone is rescued only by finding the artefact §2.2 actually asks
   for** — retrieved and traced to the party that designed the protocol or authors the implementation,
   never assumed from a related protocol's specification by family resemblance. Run once, at
   `sensitive-ports.md` §43.10, over the three rows this ticket named: `512/tcp` is rescued by NetBSD's
   own `rexecd(8)`, corroborated by FreeBSD's own `rexec(3)`; `513/tcp` was already carried by RFC
   1282, a genuine specification the corpus had quoted without crediting; `514/tcp` is rescued by
   FreeBSD's own `rshd(8)` and specifically **not** by RFC 1282, which — checked directly — names
   neither `rsh` nor `rshd` anywhere and states its own scope as rlogin alone.

## Rationale

### 1. This is the attestation gate's own missing source rule, not an import from determinacy

ADR-0048's Context table named the asymmetry directly: §2.2 already has a source rule (three named
forms, four supporting ADRs) while §2.4 had none until ADR-0048 wrote one. That asymmetry cuts the
other way here. §2.2's three forms were never tested against a registry specifically — every prior
attestation ruling (ADR-0036, ADR-0037, ADR-0045, ADR-0067) address the *second* and *third* forms'
seams (a shipped default vs. an example, an artefact's extent, issuance, affirmative naming). None
asks whether a bare catalogue entry can stand in for the *first* form. This decision closes that gap
the way ADR-0048 closed the parallel one for §2.4 — independently, because the two gates ask different
questions of the same document.

### 2. The registrant/registry distinction is not new; only its reach to this gate is

Nothing about *treating a registration as evidence of what its registrant said, never as its own
independent word* is invented here. ADR-0048 already stated it, `sensitive-ports.md` §9.3.4 and §18
already applied it (finger's `79/tcp` is carried by RFC 4146, an in-force specification, with IANA's
own annotation demoted to corroboration), and §22's `esmagent`/`fmtp` walk already followed a
registration to its registrant's contact record rather than reading the registry's stale service name.
What is new is applying the same distinction to §2.2's *claim* gate rather than §2.4's *determinacy*
gate — a different question, answered the same way, for the same underlying reason: IANA disclaims
speaking in its own voice, everywhere it appears in this corpus.

### 3. The rescue check matters as much as the rule, and it is not automatic

A gate that fails a bare registry entry does not, by itself, tell a reader whether a row survives. Two
of the three cells this ticket tested were rescued and one specification already present in the corpus
turned out not to cover a third row it had been informally credited with (§44.5's mis-filing of RFC
1282 against `514/tcp`). The rule and the rescue are stated together in the Decision because a reader
who applies limb 1–3 without running limb 4 would conclude wrongly that the three rows are unattested
and exposed, when in fact each clears the gate cleanly once traced to the right party.

### 4. Why 513/tcp and 514/tcp diverge despite sharing a trust model

RFC 1282 discusses rlogin's `.rhosts`/host-based trust mechanism at length, and rsh shares that same
mechanism in every modern implementation. The temptation to let one protocol's specification carry a
related, unspecified protocol's row — "the same trust model as 513" standing in for a citation — is
the same laundering §10.5 refuses when a distributor's prose is asked to speak for a protocol it did
not design, applied here across sibling protocols rather than across parties. **[measured]**, checked
directly rather than inferred from the family resemblance: RFC 1282 never mentions `rsh`, `rshd` or
`rcmd()`. This decision requires an artefact that names the row's own protocol, not one that names a
neighbour sharing its mechanism.

## Consequences

- **No `(port, transport)` pair moves.** `512/tcp`, `513/tcp` and `514/tcp` stay in Class B, unchanged.
  No footing tier moves — Class B carries no footing cell for any row, on or off this triple.
- **`sensitive-ports.md` §43.3's register loses this triple** (`§43.10.4`). It was carried at rung 1,
  sole-ground, at the head of the queue; it is re-founded at rung 4 (`512`, `514`) and rung 5 (`513`),
  and none of the three cells remains sole-ground. This is a genuine discharge by retrieval, not the
  filter-tightening (#135, #151) the register's prior "loses none" statement described — that
  statement is marked at its own clause and not repealed.
- **`docs/spec/curated-table-watch.md` §1.1's matching row is updated in the same commit**, per
  [ADR-0100](./0100-a-copy-of-a-monotonic-source-is-kept-honest-by-the-act-that-grows-it.md) — the
  Ground column edited in place, no duplicate row appended.
- **`sensitive-ports.md` §44.5's G8-entry table is corrected**: RFC 1282 was filed against `513` and
  `514`; it carries only `513`. `514`'s new carrier and both of `512`'s new carriers are unpinned,
  continuously-rendered vendor pages and join §44.4's G8-**recurring** roster instead (three new
  members; thirty-six over thirty becomes thirty-nine over thirty-two).
- **It generalises.** Any future claim cell that reaches for a bare catalogue entry — IANA's registry
  or a similarly-shaped third-party directory — in place of a specification, an owner's documentation,
  or a shipped default fails §2.2 on the same two grounds, genre and ownership, independent of how
  volatile the catalogue is. A rescue is not assumed to exist and must be retrieved and traced to the
  registrant, designer or implementer before a row may rest on it.
- **Does not generalise to `weak-key-and-signature.md`.** That table's rows all rest on published RFCs
  and no cell there rests on a registry entry; nothing there is walked or moved by this decision.
- **[ADR-0048](./0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md) is untouched.**
  This decision does not amend it, extend its scope to §2.2, or rely on it beyond citing the shared
  reasoning; §2.4's determinacy rule and §2.2's attestation rule remain two separate instruments that
  happen to treat IANA the same way, for related but independently-stated reasons.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Read "the protocol's specification (an RFC or equivalent)" to include a registry entry** | The parenthetical's own examples define behaviour; a registry records an assignment and its own procedures document (RFC 6335) says so. Admitting it would also silently repeal ADR-0048's refusal of "rank sources by authority — IANA above vendors above catalogues" for the sibling gate, through the back door of a different one |
| **Treat the registry as a corroborator under §2.3, never sole grounds** | §21.3's "reading 3" recycled, and it loses for the same two reasons: it is inert wherever it would matter (a corroborator cannot carry a cell with nothing else, which describes exactly the cells in question), and it mislabels the object — the defect is not the wrong party speaking correctly, it is nobody entitled having spoken at all |
| **Treat ADR-0048 as already dispositive and rule by reference** | ADR-0048 says it does not reach this gate, and the two gates ask differently-shaped questions (a conclusion at the wire vs. a normative statement after it). Landing at the same verdict does not mean the same argument applies, and a future reader of §2.2 alone would find no rule there without one written |
| **Let RFC 1282 carry `514/tcp` by family resemblance to `513/tcp`** | Checked directly rather than assumed: RFC 1282 names neither `rsh` nor `rshd` anywhere, and states its own scope as rlogin only. Letting one protocol's specification stand in for a related, unspecified protocol's row is the cross-protocol form of the distributor laundering §10.5 already refuses |
| **Leave the three cells excluded pending a survey of every possible rescuing artefact** | Unbounded and unfalsifiable, the shape §14.9 already declined for `7000/tcp` and ADR-0040 refuses generally. A rescue check is run once, over what is actually retrievable for these specific rows, and stops when an artefact is found or the search is recorded as exhausted |
