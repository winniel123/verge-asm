---
number: 1891
title: "An operator act is recorded as a fifth Operational corpus, under a four-limb predicate"
slug: an-operator-act-is-recorded-as-a-fifth-operational-corpus-under-a-four-limb-predicate
date: 2026-09-13
status: accepted
source: grilling
ticket: 1891
map: 1786
proof: {ticket: 1826}
relations:
  - {kind: amends, adr: 73, clause: "1"}
  - {kind: amends, adr: 74}
  - {kind: amends, adr: 87}
  - {kind: amends, adr: 93}
  - {kind: amends, adr: 126}
---

# ADR-1891: An operator act is recorded as a fifth Operational corpus, under a four-limb predicate

## Decision

An operator act is recorded as an `Act`, the fifth Operational corpus beside `Dispatch`, `Message`,
`Delivery` and `Transcript`. An act is auditable when a principal changed the estate's declaration,
changed who may act on this instance, directed the instance to act on the network, or caused the
instance to disclose a value from a corpus the model seals. That is a predicate, never a route list.

This reverses [#127](https://github.com/winniel123/verge-asm/issues/127). The refusal survives over
the Declared layer only: no Declared term acquires an actor, and the act of declaring carries one. No
derivation may read an `Act`.

Rejected: #127 §5's deferral on an absent consumer, with #127 §9's reopening condition as the
trigger. Four ADRs already prescribe an audit write and fifteen attribution columns already shipped,
so the ground is the demand side rather than that condition firing.

Reversal costs a shipped, never-deleted corpus and 42 withdrawn sentences.

## 1. Context

[#127](https://github.com/winniel123/verge-asm/issues/127) ruled an operator-act record out of v1 —
*"named accounts create identity; they do not create a log"*. ADR-0073 §1 then made that refusal
*"total rather than nearly total"*, and four further ADRs inherited it as a ground.

The refusal is now contradicted by the tree it governs, and
[`docs/spec/audit-act.md`](../spec/audit-act.md) measures the contradiction:

- **Four ADRs prescribe an audit write in their own Decision blocks** — ADR-0113, ADR-0053, ADR-0123,
  and `docs/spec/packaging-and-configuration.md` §5.1's test. The SPEC enumerates eighteen such
  sentences.
- **The refusal was already overrun in the store.** Fifteen columns across twelve tables carry a
  `created_by`/`updated_by` foreign key to `account` — the option #127 §3 weighed and rejected by
  name. Six of the fifteen render today.

#127 §9's condition — *"the spec admits a second party who can mutate"* — is supporting, not
load-bearing. A session that waits for it to fire waits for the wrong event.

## 2. The corpus, and the fence that makes it safe

An `Act` is Operational: it records what the system did, never what is true of the estate. The
comparison path may read nothing in it.

That fence is what ADR-0007's *"nothing needs to know a release happened or consult an audit trail"*
now buys. That sentence refuses **consumption**, so it is not withdrawn by this ADR. It is
load-bearing here, and it is the reason no derivation may read an `Act`.

The corpus is append-only and unbounded, with no retention dial. #127 §4 — *"if it is worth writing
it is worth rendering"* — is satisfied rather than withdrawn: the corpus and the admin `audit` tab
ship together.

## 3. The predicate, and why it is not a route list

An act is auditable when a principal

1. changed the estate's declaration, or
2. changed who may act on this instance, or
3. directed the instance to act on the network, or
4. caused the instance to disclose a value from a corpus the model seals.

The four limbs share one shape, and it was never *changed something*: a principal caused an effect
the instance cannot take back. Limb 3 has no local mutation at all. Limb 4 is limb 3 one register
over — disclosure instead of a network call — and once the bytes are on the operator's screen no
later act retrieves them.

A route list goes stale on the next route. That is the failure the predicate exists to avoid.

## 4. The refusal survives over the Declared layer

ADR-0073 §1's outcome stands and its stated ground does not. `Annotation` carries no author: no
account, no name, no initials, no avatar, no *"declared by"* cell, on the object. The **act** of
annotating is recorded as an `Act` with its `Actor`.

So §1's ruling no longer reopens on #127's condition and on no other. It rests instead on the layer
cut — the probing gate reads Declared terms, and operator identity lives in a corpus no derivation
reads.

A per-object refusal stays true everywhere it is written. What is withdrawn is the **generalising**
sentence: *no operator act is written down anywhere*, *the ruling is total*, *the audit facility is a
repo-wide stub*.

## 5. The five targets, and what each relation carries

Each relation is declared in the front matter above. The marker tool writes every marker
(ADR-1644). None is hand-written.

| Target | Scope | What the relation carries |
| --- | --- | --- |
| ADR-0073, clause 1 | *"not stored and not rendered"*; *"total rather than nearly total"*; *"not an exception to #127 … reopens on #127's condition and on no other"* | §4 above |
| ADR-0093 | *"No actor anywhere"*; *"No operator-act record"*; *"Refused, and not by this ADR"* | The record is in scope and the actor column is not left out. What ADR-0093 was right to refuse is a dated Declared history, which #127 §5 refuses on principle and this ADR does not reopen |
| ADR-0087 | *"An actor. #127 ruled the operator-act record out of scope … Refused"* | §6 below. The outcome is confirmed and re-grounded |
| ADR-0074 | *"An operator-act record \| Untouched. #127 stands"* | *"this message names no actor and sits in no Declared term"* is confirmed. *"#127 stands"* is withdrawn |
| ADR-0126 | *"the audit facility is a repo-wide stub"*; *"Reads are unaudited in v1 … Accepted residual risk"* | The alternative ADR-0126 refused is now adopted, selected by ADR-0126's own ground: it is the one corpus Postgres holds a secret for |

ADR-0093's limb 2 is restated rather than withdrawn, and that is a separate decision on a separate
record. This ADR does not make it.

## 6. ADR-0087's refusal is confirmed, and this is the row that bites

A closure carries no actor. The outcome does not move, and its ground is replaced.

ADR-0087 refused the actor because #127 ruled the operator-act record out of scope. That ground is
gone. The refusal now rests on the fence of §2: the `Span` corpus is **Observed** and
derivation-readable, so an actor on a closure puts operator identity one join from the comparison
path. It is exactly the join the Operational fence exists to prevent.

Without this repair a session reads *"#127 is reversed"* and builds the `who`.

The same shape holds at ADR-0074, whose row is an impact-table cell rather than a rule, and at
ADR-0073 §1, whose outcome stands while its scope narrows to the `Annotation` object.

## 7. Rejected alternatives

| Rejected | Why |
| --- | --- |
| #127 §5's deferral — an act log is deferred on the absent consumer | The consumer exists. Four ADRs prescribe the write and the columns already ship |
| #127 §9's reopening condition as the trigger | It is supporting. Waiting for a second mutating party waits for the wrong event, and the SPEC's ground is the demand side |
| A route list instead of a predicate | It goes stale on the next route |
| An actor on a `Span` closure, now that the refusal is lifted | §6. The `Span` corpus is Observed, so that actor is the join the fence prevents |
| Withdrawing ADR-0007's *"consult an audit trail"* sentence | It refuses consumption, not recording. It is now load-bearing |

## 8. Where this is thin, stated rather than smoothed

**One withdrawn sentence carries no marker, and the mechanism cannot reach it.** ADR-0073's
Consequences bullet — *"#127's ruling is now total. No operator act is written down anywhere with an
actor on it holds without exception"* — sits under `## Consequences`, an unnumbered heading.
ADR-1644 makes the unit the numbered clause, and `adr-governance` §4 rules that a `clause` never
names an unnumbered heading.

**This ADR accepts that residual rather than repairing it.** ADR-0073 acquires `amended` status in
`index.json`, and the clause marker sits under `### 1.`, the heading the substantive withdrawal
lands on. A reader who reaches Consequences has passed both.

The price is named: a session that greps the sentence and reads only its bullet finds no marker on
the line. Two repairs were weighed and declined. A legacy in-file amendment reproduces the hand edit
ADR-1644 exists to prevent, whatever `adr-governance` §3 keeps valid at 227 and below. Widening the
marker unit amends ADR-1644 itself, to reach one bullet.

## 9. Proof

`{ticket: 1826}` — the implementation map that builds the corpus,
[#1826](https://github.com/winniel123/verge-asm/issues/1826). No test can prove this today: no
migration, no table, no handler and no copy has landed.
