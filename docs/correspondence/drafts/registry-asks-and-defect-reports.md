# Drafts: the four registry asks and the two defect reports

**These are drafts. They are evidence of nothing.** Nothing in this file has been sent, and no
surface may read it. Sending means moving the message into `docs/correspondence/` as its own
file with the real recipient and the real date. See [the README](../README.md).

Prepared by [#57](https://github.com/winniel123/verge-asm/issues/57) for
[#59](https://github.com/winniel123/verge-asm/issues/59), which is the dev's to act on. No
session may send any of this.

## Before sending

**Confirm every recipient against the organisation's own current contact page.** Only one
address in this effort is evidenced: `stat@ripe.net`, which
[#19](https://github.com/winniel123/verge-asm/issues/19) took from RIPEstat's own
documentation. The other three were never recorded — that is part of what #57 found — and
each is marked `TO CONFIRM` below. Do not send to an address invented here or anywhere else.

**Send six separate messages, not four.** The two defect reports go on their own, to whatever
address each registry publishes for its website or documentation. Bundling them into the asks
is why both are still unreported.

---

## 1 — RIPE NCC · RIPEstat and the RIPE Database

**To:** `stat@ripe.net` (evidenced — RIPEstat's documented address for above-1000/day
registration). If RIPE publishes a separate contact for RIPE Database terms, copy it.
**Ticket:** [#19](https://github.com/winniel123/verge-asm/issues/19)
**Subject:** Query permissions for an open-source, self-hosted attack-surface tool

> Hello,
>
> I maintain verge-asm, an AGPL-3.0, self-hosted attack-surface monitoring tool. It is not a
> service — each user runs their own copy on their own infrastructure and it inventories only
> their own estate. I would like to check two of your services against their terms before
> deciding whether either may be enabled by default, rather than making that reading on my
> users' behalf.
>
> How it would run, precisely:
>
> - Each installation queries `stat.ripe.net` directly. The project never fetches, aggregates,
>   caches or redistributes RIPEstat data itself.
> - The operator queries only about their own estate — seed domains and address ranges they
>   assert they control.
> - Calls used: `network-info`, `announced-prefixes`, `searchcomplete`. Concurrency capped at
>   8 per your documentation, with a per-run budget aimed at staying under 1,000 requests per
>   day, and an identifying User-Agent naming the project and version.
> - Results are written into the operator's own inventory database and retained across runs so
>   that changes over time can be detected. They are not republished.
> - Operators are typically commercial organisations monitoring their own networks. They are
>   not selling a service built on RIPEstat.
>
> **On RIPEstat**, two things are unclear to me:
>
> 1. Permitted use covers "network analysis, network monitoring and debugging and research",
>    which describes this closely. But the terms also bar re-packaging, compiling or
>    redistributing the service or data without written permission. Does writing announced
>    prefixes into an operator's own inventory database, and keeping them between runs, fall
>    inside "compile"?
> 2. The commercial prohibition names "providing paid services, products or any other
>    derivatives". I read that as barring resale of RIPEstat data rather than barring companies
>    from using it on their own networks, but it does not say so. Is that reading right?
>
> **On the RIPE Database** (the REST search and `rdap.db.ripe.net`), three:
>
> 3. Article 4.1 confines access to the purposes listed in Article 3, and none of them is
>    estate inventory. Is inventorying one's own resources a permitted purpose at all? This is
>    the question that matters most to me.
> 4. Is one organisation's own object set an "insubstantial part" under Article 4.5, and not
>    "a significant part" under the AUP?
> 5. Does storing results across runs constitute "re-use"?
>
> Two plain questions on top of those: may either service ship enabled by default in software
> of this shape, and if not, is enabling it per-operator acceptable, or is written permission
> required for each deployment?
>
> Until I hear back, both ship disabled and stay disabled. I would rather have your answer
> than my own guess.
>
> Thank you,
> Logan Winnie · verge-asm

---

## 2 — APNIC · the live registry path

**To:** `TO CONFIRM` — APNIC's published helpdesk or whois contact.
**Ticket:** [#23](https://github.com/winniel123/verge-asm/issues/23)
**Subject:** Query permissions for an open-source, self-hosted attack-surface tool

> Hello,
>
> I maintain verge-asm, an AGPL-3.0, self-hosted attack-surface monitoring tool. It is not a
> service — each user runs their own copy on their own infrastructure and it inventories only
> their own estate. I would like your answer on two clauses rather than my own reading of
> them.
>
> How it would run: each installation queries APNIC directly, and the project never fetches,
> aggregates or redistributes APNIC data itself. The operator queries only about their own
> estate. Results are written into the operator's own inventory database and retained across
> runs so that changes can be detected. They are not republished. Requests carry an
> identifying User-Agent naming the project and version.
>
> Three questions:
>
> 1. Is inventorying one's own estate an "Internet operational purpose approved by APNIC"?
> 2. Your terms restrict storage "in a retrieval system". Does keeping the returned objects in
>    an operator's own inventory database between runs fall inside or outside that
>    restriction? This is the one I am least able to answer for myself, because it is about
>    what the software inherently does rather than about who is running it — so no individual
>    operator can settle it.
> 3. If approval is required, is it granted per project or per deployment?
>
> Until I hear back, APNIC's live registry path ships disabled and stays disabled.
>
> Separately, and only as something you may want to know: APNIC's RDAP entity search returns
> HTTP 200 with `"entitySearchResults":[]` for every query form I tried, rather than the 501
> that RFC 9082 specifies for an unsupported search. The same handles fetch correctly when
> requested directly. A client that trusts the empty array reads it as "this organisation
> holds nothing", which is a different statement from "this server does not answer that
> question".
>
> Thank you,
> Logan Winnie · verge-asm

---

## 3 — AFRINIC · the live registry path

**To:** `TO CONFIRM` — AFRINIC hostmaster. The address was never recorded, and the terms text
[#20](https://github.com/winniel123/verge-asm/issues/20) read had redacted it.
**Ticket:** [#24](https://github.com/winniel123/verge-asm/issues/24)
**Subject:** Permitted-use question for an open-source, self-hosted attack-surface tool

> Hello,
>
> I maintain verge-asm, an AGPL-3.0, self-hosted attack-surface monitoring tool. It is not a
> service — each user runs their own copy on their own infrastructure and it inventories only
> their own estate.
>
> How it would run: each installation queries AFRINIC directly, and the project never fetches,
> aggregates or redistributes AFRINIC data itself. The operator queries only about their own
> estate. Results are written into the operator's own inventory database and retained across
> runs. They are not republished. Requests carry an identifying User-Agent naming the project
> and version.
>
> AFRINIC publishes a closed permitted-use list, sealed by the statement that no other use
> shall be implied or permitted. Inventorying one's own resources is not named on it. A closed
> list that does not name a use is not an authorisation, and I do not think it is my place to
> read it as one on my users' behalf.
>
> So: does this use fall inside the permitted-use list, and if it does not, can it be permitted
> on request?
>
> Until I hear back, AFRINIC's live registry path ships disabled and stays disabled. I should
> say that your RDAP is the best organisation-to-prefix endpoint of any RIR I looked at — it
> embeds `networks` and `autnums` inline, which is one request where others need two — so this
> is a capability I would like to be able to offer.
>
> Thank you,
> Logan Winnie · verge-asm

---

## 4 — LACNIC · a retrievable statement of terms

**To:** `TO CONFIRM` — LACNIC's published contact.
**Ticket:** [#25](https://github.com/winniel123/verge-asm/issues/25)
**Subject:** Where can the terms your RDAP links to be retrieved?

> Hello,
>
> I maintain verge-asm, an AGPL-3.0, self-hosted attack-surface monitoring tool. It is not a
> service — each user runs their own copy on their own infrastructure and it inventories only
> their own estate.
>
> I am trying to check LACNIC's terms before deciding whether the software may query LACNIC by
> default, and I cannot retrieve them. Your RDAP responses advertise a link with
> `rel: "terms-of-service"`, and that URL returns a 7,014-byte JavaScript shell rather than any
> terms text. Four other documented policy URLs return the byte-identical response.
>
> Two questions:
>
> 1. Where can the terms referenced by that link actually be retrieved?
> 2. Is an operator inventorying their own estate, storing the results across runs, inside
>    them? If there is nothing beyond the banner in the RDAP response, saying so plainly would
>    settle it.
>
> How it would run, for context: each installation queries LACNIC directly, the project never
> aggregates or redistributes LACNIC data itself, the operator queries only about their own
> estate, results go into the operator's own inventory database and are not republished, and
> requests carry an identifying User-Agent naming the project and version.
>
> Until I hear back, LACNIC ships disabled and stays disabled. I would rather ship it disabled
> than assume terms I have not been able to read.
>
> Thank you,
> Logan Winnie · verge-asm

---

## 5 — AFRINIC · defect report, sent separately

**To:** `TO CONFIRM` — whatever AFRINIC publishes for website or documentation problems.
**Ticket:** [#24](https://github.com/winniel123/verge-asm/issues/24)
**Subject:** Broken link: afrinic.net/whois/terms returns 404

> Hello,
>
> A small documentation defect, passed on in case it is useful.
>
> `afrinic.net/whois/terms` returns HTTP 404. It is the URL AFRINIC's own documentation points
> at for the WHOIS terms and conditions. The terms text itself is reachable elsewhere, so this
> is a broken pointer rather than missing content — but anyone following your documentation to
> read the terms before using the service will not find them.
>
> No reply needed.
>
> Thank you,
> Logan Winnie · verge-asm

---

## 6 — LACNIC · defect report, sent separately

**To:** `TO CONFIRM` — whatever LACNIC publishes for website or documentation problems.
**Ticket:** [#25](https://github.com/winniel123/verge-asm/issues/25)
**Subject:** Five documented policy URLs return a script shell instead of content

> Hello,
>
> A defect on your site, passed on in case it is useful.
>
> Five documented URLs return a byte-identical 7,014-byte JavaScript shell rather than the
> documents they name. One of them is the target of the `rel: "terms-of-service"` link that
> LACNIC's own RDAP responses advertise; the other four are documented policy addresses.
>
> The effect is that a client following the link your RDAP publishes cannot retrieve the terms
> it points at. It looks like a client-side rendering path that does not serve content to a
> plain HTTP request, so a browser may well show the pages correctly while anything else does
> not.
>
> No reply needed.
>
> Thank you,
> Logan Winnie · verge-asm
