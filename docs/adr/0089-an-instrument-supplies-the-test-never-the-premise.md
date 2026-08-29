# ADR-0089: An instrument supplies the test, never the premise — a claim-set failure still takes an owner for the fact it turns on

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#153 `22/tcp`'s exclusion ground is an assertion of ours](https://github.com/winniel123/verge-asm/issues/153)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`sensitive-ports.md`](../research/sensitive-ports.md) §2.2 opens with the bar that makes the table
worth having: **"The claim may not be asserted by us."** §2.1 supplies the instrument the table applies
— three permitted claims, closed by construction at §10.2 — and it justifies the instrument's reach
with a sentence naming seven services the claim set removes *"in a single stroke"*:

> *"Applied honestly it removes SSH, RDP, WinRM, Kibana, Grafana, Jenkins and the Kubernetes API server
> from consideration in a single stroke — because for each of those, a human or a remote client **is**
> the intended audience, and remote administration over an untrusted network is the express purpose the
> protocol was designed to serve."* — §2.1

**Nobody had asked whether that sentence is an attestation.** §22.5
([#87](https://github.com/winniel123/verge-asm/issues/87)) answered in the negative and did not need to:
re-founding `5601/tcp` after withdrawing an assertion of ours, it ruled the surviving ground to be
*"§2.1's Claim 3 failure, **which is this note's own instrument and needs no owner**."* That reading is
correct for `5601`, and it is correct for the reason §22 gave — but it was never tested against a cell
where the instrument's justification **is** the withdrawn clause.

[#133](https://github.com/winniel123/verge-asm/issues/133)'s G5 walk found that cell. §4.6's `22/tcp`
entry read *"Remote administration over an untrusted network is the protocol's express purpose"* —
**uncited**, and word for word the second half of §2.1's justification. §40.3 recorded it as *"an
assertion of ours standing where an owner's sentence should"*, third of the shape, and correctly
declined to rule: *"A gate raises; only a release rules"*
([ADR-0057](./0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)).

**Taking §22.5's wide reading would have discharged the raise for free** — §2.1 names SSH, so the cell
re-founds on the instrument and no retrieval is needed. That is the option this ADR refuses. Under
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as widened by
[#106](https://github.com/winniel123/verge-asm/issues/106) — *a document supersedes itself* — it would
have moved one uncited sentence one level up the document and called the defect discharged.

This ADR moves no row, no class, no footing tier and no coverage figure, and
[ADR-0008](./0008-derivation-versions-move-on-content.md) is not triggered. The full working is
`sensitive-ports.md` **§45**.

## Decision

**An instrument supplies the test. It never supplies the premise the test is applied to.**

Two limbs.

1. **The instrument needs no owner.** A curated table's own claim set — its wording, its closure, and
   the verdict *this subject fails Claim N* — is the project's analytical apparatus. §2.2's first
   sentence does not reach it, and requiring an owner for it would be incoherent: no owner has an
   opinion about our claim set. §22.5 is right about this and is unamended in this limb.

2. **The premise takes an owner.** Every application of the instrument turns on a **fact about the
   subject** — *what this protocol's intended clients are*, *whether this service ships authentication*
   — and that fact is a claim about the world. §2.2's first sentence reaches it, in **both** directions:
   a cell that **excludes** a subject asserts a fact about it exactly as a cell that admits one does.
   A cell citing the instrument's own **justification** as its ground is citing **us**.

**The operational test, and it is one question.** *Strike the instrument's justification out. Does a
sentence an owner wrote still say why this subject fails the claim?* Where the answer is yes, the cell
is grounded and the instrument was only ever doing the classification. Where it is no, the cell rests on
an assertion of ours wearing the instrument's clothes, and the repair is a retrieval scoped to that cell
([ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 2).

**What this does not do.** It does not reopen §10.2's closed claim set, add a claim, or move a row. It
does not make an exclusion harder to reach — it makes the *stated ground* honest about whose sentence it
is. And it does not travel to §2.1 or §2.4, which
[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) confines to
`sensitive-ports.md`. It rides on **§2.2's attestation gate**, which is the half ADR-0032 says does
travel to other curated tables.

## Consequences

**`22/tcp`'s cell is re-founded on the owner's specification and the row does not move.** RFC 4251 §1 —
*"Secure Shell (SSH) is a protocol for secure remote login and other secure network services over an
insecure network"* — is §2.2's **first** form and states in the specifier's voice what the note had been
asserting. Claim 3 requires intended clients inside a boundary the operator controls. The owner says the
protocol exists for the case where they are not. §45.2, §45.6.

**§22.5's sentence is narrowed in place, not deleted.** Its first half is this ADR's limb 1. It is
marked at the site that specifies it, per ADR-0058, and `5601/tcp` is unharmed: §22.2's determinacy
refusal is an owner artefact and §38 measured Elastic's own corpus on the audience question, so that row
is overdetermined twice.

**Two more cells of the shape are named and routed, not swept.** **[measured]** §4.6's `3389/tcp` RDP
entry inherits the withdrawn clause by reference (*"Same category"*), and §17.1 rows 9 and 10 score RDP
and `5985`/`5986/tcp` WinRM on the same words. Each needs its own retrieval — ADR-0037 limb 2 — and
neither is repaired by the pass that found it. §45.7.

**One figure is put in question and deliberately not moved.** §17.2's arithmetic — *14 negatives · 8
un-exposed · 5 searched · 1 exhausted · **0 residue*** — un-exposes rows 9 and 10 on the §10.2 gate.
Limb 1 keeps the gate available. Limb 2 leaves the premise those two rows' application of it turns on
unattested. **Whether an unattested premise still reaches the gate is open**, and answering it inside
#153 would have moved a figure on a retrieval scoped to a different cell — the *machine rules* failure
§39.6 bars and §40.7 declined for the same reason. Ticketed.

**This is the third instance of §17.4's shape and the first found by a gate.** §17.4 (`5672`/`15672`)
and §22 (`5601`) were each found by a ticket that happened to be reading the cell. §40.3 found this one
by walking a population, which is what the gate was built to do — and the two it could strike inline it
struck, while the one needing a retrieval it correctly left. **The instrument and the release did
exactly the jobs ADR-0057 assigned them**, and this ADR is the first evidence of that division working
end to end.

**Thin ground, flagged.** The rule rests on **one** measured instance — the only cell where the
instrument's premise and a withdrawn ground are demonstrably the same sentence — against one
counter-instance (`5601`) where the wide reading was harmless. A rule minted on one instance has been
right once. **What would falsify it:** a cell whose §2.1 naming rests on a premise no owner has stated,
and where requiring an owner would leave the note unable to exclude something it plainly should exclude.
§45.7's two routed cells are the candidates and this ADR does not pre-judge them. §45.9.

## Alternatives considered

| Option | Why it lost |
|---|---|
| **Re-found `22/tcp` on §2.1, as §22 did for `5601`** — no retrieval at all | §2.1's reason for naming Kibana (*"a human is the intended audience"*) is a **different** proposition from the clause §22 withdrew, which is why the move was legitimate there. §2.1's reason for naming **SSH** is the withdrawn clause **verbatim**. The move would relocate the assertion, not retire it — ADR-0058 as widened by #106, precisely |
| **Rule the whole `express purpose` clause inadmissible and strike it from §2.1** | Overreach, and it would break the instrument. §2.1's justification is legitimate *as reasoning about why the claim set excludes those seven*; what is barred is **citing it as a cell's ground**. Limb 1 exists to protect this |
| **Widen §2.2's bar to reach the instrument itself** | Incoherent — no owner has a position on our claim set — and it would make every exclusion unreachable. This is the reading §22.5 was right to refuse |
| **Repair all three cells (`22`, `3389`, `5985`/`5986`) in one pass** | ADR-0037 limb 2: the retrieval that found the shape was scoped to `22/tcp` and answers no gate for the other two. RDP's and WinRM's owner is Microsoft and the unopened classes differ per cell. Recorded and routed |
| **Mint nothing and leave the finding in §45** | The tempting call, since the repair applies only rules that already exist. It loses on ADR-0058's own measured population: a finding that lives in a note beside a **wider** sentence in the same document is a finding the next session reads past. §22.5's *"needs no owner"*, read alone and in the present tense, would cause a competent session to re-derive the wrong answer — which is ADR-0058's test, failed |
