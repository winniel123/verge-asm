# ADR-0042: A squat is contested where the other convention is live

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#75 Cassandra's shipped config prohibits internet exposure of 7000 and 7001 too — are they rows?](https://github.com/winniel123/verge-asm/issues/75)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) ruled that an evidence
standard attaches to a curated table, and that `sensitive-ports.md` §2.4's **determinacy** gate is
one of the parts that does **not** travel — it applies only where a port stands surrogate for a
service, which is the one surrogate v1 has. So determinacy is local to
`sensitive-port-reached-from-internet`, and it has never been given the treatment the table's other
gates have had. §2.1's claim set was closed by construction in
[#37](https://github.com/winniel123/verge-asm/issues/37). §2.2's attestation acquired three forms,
an owner definition ([ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md)), a
one-way rule and an artefact test ([ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md),
[ADR-0037](./0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)). Determinacy
still reads, in its entirety, that registration cannot be the test because so many well-known ports
are squatted, and that **"uncontested convention has to be"** the test instead.

That sentence has been deciding rows for a year without saying what makes a convention *contested*,
and the table contains two rulings that look identical and came out opposite:

- **~~`9200/tcp` is on the list.~~** Elasticsearch squats on `wap-wsp`, *"WAP connectionless session
  service"*.
- **`9100/tcp` is excluded.** `sensitive-ports.md` §4.6: *"Squats on `hp-pdl-datastr` … One port,
  two completely different services, opposite populations."*

Nothing written down separates them. Read as *a squat is fatal*, ~~`9200`, `9300`,~~ `2181`, `9042`,
`10250`, `10255` and `623/udp` all leave the list. Read as *a squat is never fatal*, `9100` returns
and the gate stops doing work.

> **Both `9200` clauses and the `9300` clause struck at the clause by
> [#133](https://github.com/winniel123/verge-asm/issues/133)'s gate run (G4).** `9200/tcp` and
> `9300/tcp` **are not on the list** — [#114](https://github.com/winniel123/verge-asm/issues/114)
> removed both rows on the **claim** gate ([`sensitive-ports.md`](../research/sensitive-ports.md)
> §38), and they are §4.6 entries. **This ADR's determinacy verdicts on `wap-wsp` and `vrace` are
> unchanged, unread and correct** — determinacy is not what removed them, and the reconstruction
> table below already says so **per row**. What #114 did not do is strike the **premise** three
> hundred lines above the table, which is the clause a session reads first and the one that would
> cause it to rebuild the belief. That is
> [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) exactly:
> the amendment landed at the site that *records* the verdict and not at the site that *specifies*
> the puzzle. **The puzzle is unaffected** — `9100` versus the surviving five is the same puzzle and
> the ADR's answer to it is untouched. The table has been running on an unstated third reading, and an
unstated criterion is precisely what §2 exists to prevent — the defect
[#30](https://github.com/winniel123/verge-asm/issues/30) found in Claim 1 and #37 repaired, one gate
across.

[#75](https://github.com/winniel123/verge-asm/issues/75) is the ticket that could not proceed
without the answer. **[measured]** Apache Cassandra's shipped `conf/cassandra.yaml`, at
`cassandra-5.0.2`, `cassandra-5.0.6` and `cassandra-5.0.9`, carries *"For security reasons, you
should not expose this port to the internet. Firewall it if needed."* immediately above
`storage_port: 7000` and `ssl_storage_port: 7001`. On `7000` the claim passes on both of #37's
ordered Claim 1 steps **and** on Claim 3 with its boundary limb named by the owner, and the
attestation is a first-party prohibition naming the port by number in operative shipped bytes.
Everything the table asks for is present except an answer to: Cassandra squats on
`afs3-fileserver`, and Apple documents `AirPlay · 7000 · TCP` in
[*TCP and UDP ports used by Apple software products*](https://support.apple.com/en-us/103229). Is
that squat contested?

## Decision

**A squat is contested where the other convention is live. It is uncontested where the competing
registration or usage has no deployed population, and liveness is read off the competing service's
own owner, never off a frequency source.**

Three limbs.

1. **The test is on the competing service, not on the candidate.** Determinacy asks whether the
   `(port, transport)` pair determines *a* service. A candidate's own strength — how good its
   attestation is, how clearly its claim fits — is irrelevant to it. An unimpeachable attestation
   does not clear a gate it does not speak to.
2. **Liveness is established by the competing owner's own current documentation** — the vendor or
   project saying, now, that its service listens on that number. Apple's product-port table is the
   paradigm. A registration that its assignee no longer ships, and that no owner currently documents
   as a live listener, does not contest anything.
3. **Deployment share is not admissible, in either direction.** *Which of the two is more often
   found exposed* is a frequency question, and
   [`sensitive-ports.md`](../research/sensitive-ports.md) §1 excludes frequency from this table as a
   matter of framing. A contested convention stays contested however rare the competitor is, and an
   uncontested one is not made contested by a competitor being popular somewhere else.

> **Amendment — [#82](https://github.com/winniel123/verge-asm/issues/82): the criterion stands, the
> source rule arrives, and one phrase below is withdrawn.** This ADR settled *what makes a squat
> contested* and left open *which classes of source may establish a convention at all*, as its own
> Consequences record. That is now
> [ADR-0048](./0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md): a determinacy
> finding is made on **placement statements** and on nothing else, and every other class — IANA rows,
> `nmap-services`, cloud and government port tables, and this project's own frequency half —
> corroborates and never carries. Three things follow for this ADR.
>
> **Limb 3 gains a limb of its own: the unit is the protocol, not the vendor.** Two parties placing
> the **same** protocol on a pair are one convention, not two, with compatibility read off the second
> party's own declaration. The reconstruction table below never met the case; the #82 walk did.
> OpenSearch and the Wazuh indexer are current first-party placements on `9200`, and ScyllaDB is one
> on `9042`. Without this limb both rows would leave the list, wrongly. `sensitive-ports.md` §15.1
> limb 3.
>
> **The table's `9200` cell is withdrawn as a *ground* and stands per the name-and-withdraw
> convention.** *"No — WAP has no deployed population"* is a **population** sentence inside a rule
> that forbids population sentences, and it is the crack through which frequency would return. The
> finding is unchanged and re-founded: **no party currently publishes a placement statement for WSP on
> `9200/tcp`** — the surviving OMA specification is an archived release of a suite nobody ships, and
> the WAP bearer ports are UDP besides. **Liveness is the currency of a declaration, not the size of a
> population**, which is limb 3 of this ADR read strictly rather than a change to it.
> `sensitive-ports.md` §15.3.
>
> **`79/tcp`'s cell is strengthened.** Its defeating artefact is RFC 4146 itself, in force, with
> IANA's annotation corroborating rather than carrying.
>
> **No verdict in the table below moves, and no `(port, transport)` pair moves.** Nine rulings were
> re-run under the source rule — these six plus `7001`, `9090` and the nine convention-resting rows
> individually — and the list stays at 37.

## Rationale

**It is not a new rule. It is the rule the table has been applying.** Every ruling in the note falls
out of it without strain, which is the test a reconstruction has to pass:

| Row | Squat | Live competitor? | Verdict, and it is the existing one |
|---|---|---|---|
| `9200/tcp` Elasticsearch | `wap-wsp` | **No** — WAP has no deployed population | ~~**Listed**~~ — **the ROW is REMOVED by [#114](https://github.com/winniel123/verge-asm/issues/114)** on the **claim** gate (`sensitive-ports.md` §10.3's failure condition, §38). **This ADR's determinacy verdict is unchanged, unread and correct** — determinacy is not what removed it |
| `9300/tcp`, `2181/tcp`, `9042/tcp`, `10250`, `10255`, `623/udp` | `vrace`, `eforward`, unassigned, unassigned, `asf-rmcp` | **No** | **Listed** — except `9300/tcp`, whose **row is REMOVED by [#114](https://github.com/winniel123/verge-asm/issues/114)** on the same claim gate, with its determinacy verdict likewise unchanged |
| `9100/tcp` node_exporter | `hp-pdl-datastr` | **Yes** — HP's own best-practices document says *"9100 Printing should always be enabled"* | **Excluded** |
| `6443/tcp` kube-apiserver | `sun-sr-https` | Contested by Kubernetes itself — upstream notes the API *"serves on port 443"* in typical production | **Excluded** |
| `79/tcp` finger | registered to finger | **Yes** — RFC 4146 records a mail-notification listener on 79 as an additional usage, in IANA's own reference | **Excluded** |
| `7000/tcp` Cassandra | `afs3-fileserver` | **Yes** — Apple documents `AirPlay · 7000 · TCP` | **Excluded** ([#75](https://github.com/winniel123/verge-asm/issues/75)) |

Six existing rulings, one criterion, no exceptions. That is what makes this a **statement** of the
standard rather than a change to it, and it is why no `(port, transport)` pair moves.

**Limb 1 is the load-bearing half, and it is the one a reader will resist.** `7000/tcp` is better
attested than `5432/tcp`, which is on the list. It is better attested than `9042/tcp` was before
[#69](https://github.com/winniel123/verge-asm/issues/69) moved it. And it is refused. That feels
wrong, and the feeling is the same one §4.4 recorded for `6443` — *"the single hardest exclusion and
the one most likely to be challenged"*. The answer is structural: the gates measure different
things. Attestation asks *is exposing this ever correct?* Determinacy asks *if the signal fires,
does the operator know what they are looking at?* A row can be unarguably right on the first and
useless on the second, and trading one against the other converts the table's gates into a score.
`sensitive-ports.md` §13.5 wrote the sentence before the case arrived: **the strongest sentence in
the corpus does not clear a gate it does not speak to.**

**Limb 2 keeps the rule falsifiable.** *Live* is a property of the world, and a criterion that asks
a reviewer to judge whether a protocol is "still a thing" is the ownerless counterfactual §10.1
deleted from Claim 1 and §12.4 refused as a coherence gate. Keying it to the competing owner's own
current documentation makes it a **retrieval**: open the vendor's page, find the port or do not.
That is the same move ADR-0036 made for shipped defaults — replace a judgement with a sentence read
off a document — and it inherits ADR-0037's discipline about extent, so a session that finds no
competing owner records the artefacts it opened.

**Limb 3 is where the rule would leak if it were left out.** The best argument for admitting
`7000/tcp` is that a listener on 7000 *reachable from an internet vantage* is far more likely to be
Cassandra than AirPlay, because AirPlay receivers sit behind NAT. It is a good argument and it is a
frequency argument. §11.5 refused a deployment-share argument by name on `161/udp`, §12.3 refused
another on `snmpd.conf`, and admitting one here would let frequency decide a normative table through
the one gate nobody had fenced. It is also not safely true: `Exposure` fires on `edge-only` as well
as `exposed` (§4.1), and IPv6 estates give every device a globally routable address.

**What the rule costs, stated plainly.** It loses coverage on exactly the ports where a well-behaved
project has been unlucky in its choice of number. Cassandra did nothing wrong. It published a clear
prohibition against a port that Apple later populated at consumer scale, and the operator of a
genuinely exposed Cassandra cluster gets nothing from this signal on `7000`. That is the same loss
`sensitive-ports.md` §11.6 accepted for SNMP and §2.7 accepted for `111/tcp`, and it is accepted for
the same reason: a list whose firings are arguable teaches operators to ignore the product, and that
is a larger loss than a missing row.

## Consequences

- **No `(port, transport)` pair moves.** `sensitive-ports.md` stays at **37 pairs**. §1's count,
  §3's class totals (12 / 7 / 18), §2.2's footing table, §6.1's containment arithmetic and
  [ADR-0009](./0009-verge-core-is-a-union.md)'s union are unchanged, each checked rather than
  asserted (`sensitive-ports.md` §14.7). No rule version bump and no `Break`
  ([ADR-0008](./0008-derivation-versions-move-on-content.md)) — free in the strong sense, not the
  vacuous-before-v1 sense.
- **`7000/tcp` and `7001/tcp` are refused and join §4.6's negative space**, which is now eighteen
  entries. They are the first two rows in the note refused with claim and attestation both
  established. `sensitive-ports.md` §14.
- **The `9200`/`9100` tension is closed.** A reader finding both rulings and no explanation should
  read this ADR rather than concluding one of them is a mistake.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) unblocks on a definite count of 37.**
  The blocker existed because the list #12 assembles would otherwise have been assembled from a
  count about to move. It does not move.
- **It does not generalise to other curated tables**, and that is ADR-0032's ruling rather than a
  choice made here: determinacy applies where a port stands surrogate for a service, and
  `sensitive-port-reached-from-internet` is the only surrogate v1 has. A future table keyed on
  something that is not a surrogate is *outside the domain*, not passed. The weak-key table
  ([#68](https://github.com/winniel123/verge-asm/issues/68)) has no determinacy gate and gains none.
- **The rest of §2.4 still has no evidence standard.** This ADR settles *liveness*. It does not say
  which classes of source may establish a convention at all, the way §2.2 says which sources may
  attest a claim. `sensitive-ports.md` §8 question 10 carries it, routed to
  [#82](https://github.com/winniel123/verge-asm/issues/82). **No row turns on the answer** — §14's
  sources are all first-party about their own products — so it does not block #12.

## Alternatives rejected

**Treat any squat as fatal.** Clean, mechanical, and it deletes `9200`, `9300`, `2181`, `9042`,
`10250`, `10255` and `623/udp` — seven rows including three of the note's strongest. §2.4 already
refused it in terms: *"registration cannot be the determinacy test; uncontested convention has to
be."* Rejected on the measurement, in the shape §10.4.1 used to refuse the symmetric shipped-default
reading.

**Treat a squat as never fatal, and gate on the candidate's attestation strength instead.** This is
the reading that feels fair to Cassandra, and it is the one that destroys the gate: `9100` returns
on node_exporter's Prometheus lineage, `9090` returns on Prometheus's own genuine Claim 3 sentence,
and the note's two most explicit determinacy exclusions both fall. It also converts the gates into a
ranking, which is the severity-shaped reasoning ADR-0004 and §2 exist to keep out.

**Gate on which service is more likely to be internet-exposed.** Refused as limb 3. It is the
laundering `sensitive-ports.md` §1, §2.3, §11.5 and §12.3 refuse elsewhere, arriving through the one
gate that had no fence.

**Leave it unwritten and decide case by case.** The status quo, and it survived a year because no
ruling had turned on it. #75 is the first that does, and an unstated criterion cannot be argued
against — which is the failure §2 was built to prevent and the exact ground on which #37 repaired
Claim 1. Writing it down also has a measured cost of zero: the reconstruction reproduces six
existing verdicts unchanged.
