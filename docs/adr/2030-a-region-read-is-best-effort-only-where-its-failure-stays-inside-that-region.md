---
number: 2030
title: "A region read is best-effort only where its failure stays inside that region"
slug: a-region-read-is-best-effort-only-where-its-failure-stays-inside-that-region
date: 2026-09-15
status: accepted
source: grilling
ticket: 2030
proof: {ticket: 2030}
relations:
  - {kind: amends, adr: 168, clause: "1"}
---

# ADR-2030: A region read is best-effort only where its failure stays inside that region

## Decision

**A region's read is best-effort only where its failure stays inside that region. Where a
second rendered value derives from a region's read, that read is not a region read, and
ADR-0168 §1's best-effort licence does not reach it.**

The remedy is an honest absence at both ends. The region states that the read did not
resolve. The derived value renders nothing. It is never a 500, and never a substituted
value.

ADR-0168 §4 is unchanged. A subject read, a keyed subject read and the act stay loud.

A reader asks why, because §1 reads as a list of card shapes and the clause after it reads
as decoration.

Rejected: making every leaky read loud, which breaks `dashboardData`, ADR-0168 §4's own
limiting case.

Reversal returns two rules one line apart in one handler, over a header that hides the
verdict a failed read erased.

## 1. Context

Two sessions read [ADR-0168](./0168-a-console-page-degrades-a-failed-region-to-an-honest-absence-and-never-fails-the-page-or-refuses-the-act.md)
two ways and landed opposite answers one line apart in the Name asset handler.

[#1948](https://github.com/winniel123/verge-asm/issues/1948) made a failed open-spans read return a
500 from `cmd/web/subjects.go#server.assetPage`. [#1951](https://github.com/winniel123/verge-asm/issues/1951)
made the neighbouring failed corpus read render an honest absence, and it has since landed. The two
calls sit in consecutive statements of one function, and the two answers are both shipped.

[#2030](https://github.com/winniel123/verge-asm/issues/2030) framed the conflict as ADR-0168 §4's
test disagreeing with §4's three classes. §4's test asks whether *the page's claim about the world
depends on* the read. §4's three classes name the page's own subject, a subject fetched by key, and
the act. The asset page's open-spans read carries a claim and is none of the three classes, and the
ADR gives no tiebreak.

The conflict is resolved earlier, in §1, and it never reaches §4.

## 2. The derivation clause is a containment test

ADR-0168 §1 defines a region as *"a card, a list, a meter row or a callout the page composes beside
others and that no other region derives from."* The clause after the final `and` has never been read
as operative. It is the operative half.

§1's stated reason is availability arithmetic. A page makes many reads, so a loud read makes the
screen's availability the product of them all, and one fault removes regions the failed read does
not feed. That argument holds only while a region's failure stays inside that region. A failure that
escapes into a second rendered value is not the case the arithmetic describes, so the licence the
arithmetic buys does not cover it.

`cmd/web/subjects.go#server.assetPage` is the escape. `cmd/web/subjects.go#server.assetPorts` feeds the ports
table, and `cmd/web/subjects.go#assetHeaderInternetLeg` then derives the header's internet-leg chip
from that same list. An empty ports list is the honest no-ports answer, so the chip a failed read
produces is indistinguishable from the chip a portless asset produces. The region could carry a note
saying the read did not resolve. The chip carries nothing, and it reads as a verdict.

**Two sibling regions fed by one read are not a derivation.** Neither derives from the other, and a
failure empties both where a reader can see it. §1 already governs that case, and this ADR does not
move it.

### 2.1 The half-remedy the asset page ships today

[#1951](https://github.com/winniel123/verge-asm/issues/1951) and
[#2029](https://github.com/winniel123/verge-asm/issues/2029) landed while this ADR was open. Both
read ADR-0168 §1 correctly and both remedied the region end alone, so the asset page now
demonstrates the gap this ADR closes rather than arguing it.

`cmd/web/subjects.go#server.assetSignals` returns its error, and `server.assetPage` carries it into
a `SignalsFailed` flag. `design-system/templates/asset.tmpl` renders *"Signals did not
resolve"* against that flag, in a branch distinct from the branch that states no rule is firing.
That is §1 and §2 satisfied, and it is correct.

The statement after it is unchanged. `assetHeaderSeverity` still derives the header badge from the
same empty list, and the header still guards that badge with `{{if .Severity}}`. So a failed corpus
read on an asset carrying a critical signal renders one page that says both things at once: a card
reporting that the read did not resolve, and a header reporting no severity at all. The second is
the claim an operator reads first.

The covering-seed lookup on the same page took the same half. `server.assetProvenance` returns its
error and the provenance card renders a citation note, while the header's `in scope since` line
derives from the same read and vanishes with nothing said.

The region end is not the whole remedy. That is the rule.

## 3. `dashboardData` is the shipped proof, and `coveragePage` is neutral

`cmd/web/auth.go#server.dashboardData` is a leaky read, and it is already remedied at both ends. Its signal
corpus read feeds four rendered things: the severity-ramp card, the recent-signals card, two tiles
of the stat band, and the global chrome nav badge.

ADR-0168 §4 names `dashboardData` as *"the limiting case the other way: it has no subject, being a
summary of several, so no read on it is loud — consistent with this rule, not an exception."* Make
that read loud and the ADR's own second worked example breaks.

The shipped handling is this ADR's remedy, written before this ADR:

- `design-system/templates/dashboard.tmpl` renders *"The signal census did not resolve on this
  load"* and *"The signal register did not resolve on this load"*. Both sit in an `{{else}}` branch
  distinct from the branch that states no rule is firing.
- `cmd/web/auth.go#statValue` renders an em dash for both signal tiles rather than a zero.
- The nav badge key is never set, so the badge does not render.

So `dashboardData` is not a counterexample to containment. It is the proof of it.

`coveragePage` (`cmd/web/cold.go#server.coveragePage`), ADR-0168's primary worked example, contains no read
of this shape, and this ADR claims no support from it. Its blanketed-reach read feeds two regions,
the Coverage messages card and the Gaps table, and neither derives from the other. What those two
empty states assert on a failed read is an ADR-0168 §2 question about a substituted claim. It is not
a containment question, and this ADR does not reach it.

## 4. Why the remedy is an honest absence, and not a 500

[#1948](https://github.com/winniel123/verge-asm/issues/1948) chose a 500 because it read the
withheld convention as naming a domain cause. On that reading a failed database read could not be a
withheld state, so only a 500 was left.

Two shipped meters contradict the reading. `design-system/templates/coverage.tmpl` and
`design-system/templates/dashboard.tmpl` each carry a `Withheld` branch whose label is *"Not
measured — the read did not resolve"*. The convention already names a failed read, on the two
screens ADR-0168 was written about.

The 500 also costs more than it buys. ADR-0168's first rejected alternative says a loud read hands
the operator the one response carrying no information, and removes every region the failed read does
not feed. Nothing about a leaky read changes that arithmetic. What a leaky read changes is where the
absence must be shown, and the answer is both ends rather than one.

A substituted value is refused for the reason ADR-0168 §2 already gives. A zero, an empty chip or a
hidden badge is a claim about the estate produced by a database fault.

## 5. What this names as defects

ADR-0168 set this precedent itself. It named shipped sites as *"defects, not ratified"* and it fixed
none of them. This ADR does the same. Each site below needs its own ticket.

A survey of `cmd/web` counted **12 call sites across 7 helpers and handlers**, excluding
`dashboardData`. The count is of call sites, not of helpers, so one helper called from two pages
counts two. It was taken after #1951, #2017 and #2029 landed.

| Helper or handler | Sites | Region | What escapes it | Region end |
| --- | --- | --- | --- | --- |
| `cmd/web/subjects.go#server.assetPorts` | 1 | the ports table | the header's internet-leg chip | loud, so neither end |
| `cmd/web/subjects.go#server.serviceReachLegs` | 1 | the Reachability card | the header's internet-leg chip | loud, so neither end |
| `cmd/web/subjects.go#server.assetSignals` | 2 | the signals card | the header's severity badge | remedied |
| `cmd/web/subjects.go#server.assetProvenance` | 2 | the provenance card's Seed row | the header's `in scope since` line | remedied |
| `cmd/web/subjects.go#server.buildEndpointCitation` and `#server.buildServiceCitation` | 2 | the citation hops and the provenance card | the header's `in scope since` line | not remedied |
| `cmd/web/subjects.go#server.buildTimelines` | 2 | the timelines card | the provenance card's first-seen line | not remedied |
| `cmd/web/search.go#server.searchPage` | 1 | the signals results | a per-asset severity, the result total, the nav badge | not remedied |
| `cmd/web/graph.go#server.graphPage` | 1 | the graph | the per-node severity badges | not remedied |

Four sites carry a remedied region end and an unremedied derived end. Those are the half-remedies of
§2.1, and they are the cheapest to finish: the failure already reaches the handler, and only the
derived value still needs to say so.

`assetPorts` and `serviceReachLegs` are loud today. They are the exact mirror of each other, and
both are defects under this ruling rather than models of it.

Three sites this ADR does **not** name, because nothing derives from them:
`cmd/web/subjects.go#server.assetDNS` and `#server.assetCertificate` each feed one panel, and
`server.assetSignals` on the service page feeds a card that `design-system/templates/subjectdetail.tmpl`
omits, over a page whose header carries no severity badge. Each is an ADR-0168 §1 question about
logging a degradation the operator cannot see. None is a containment question.

## 6. Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Read the clause as a taxonomy, so a header chip is not a region and the clause never fires | Returns the question to §4, where the test and the three classes disagree and the ADR gives no tiebreak. That is the state #2030 was filed to end |
| Restore containment by making every leaky read loud | Breaks `dashboardData`, ADR-0168 §4's own limiting case, and sweeps in the search and graph handlers, which have no subject at all |
| Leave the asymmetry and record it as a known divergence | Leaves two rules one line apart in one handler, which is the defect reported |
| Define `region` in `CONTEXT.md` | `CONTEXT.md` is an estate glossary. `region` is console-rendering vocabulary, and adding it starts a second vocabulary inside a file that holds one |

## 7. What reversal costs

Reverting is cheap in code. Every named site keeps its current shape, and the ADR file carries a
`withdrawn` status.

What comes back is the state [#2030](https://github.com/winniel123/verge-asm/issues/2030) reported.
One handler holds two answers to one question, one line apart, and a later session reading the code
finds two rules and no statement of which applies. The operator keeps a header that names a verdict
on an estate nobody measured, and a 500 on the neighbouring read that removes every region of the
page to protect one chip.

That is cheap to reverse mechanically and expensive to reverse in meaning, which is why the rule is
recorded here rather than left beside a helper.
