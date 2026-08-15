# Drafts: the four registry asks and the two defect reports

**These are drafts. They are evidence of nothing.** Nothing in this file has been sent, and no
surface may read it. Sending means moving the message into `docs/correspondence/` as its own
file with the real recipient and the real date. See [the README](../README.md).

Prepared by [#57](https://github.com/winniel123/verge-asm/issues/57) for
[#59](https://github.com/winniel123/verge-asm/issues/59), which is the dev's to act on. No
session may send any of this.

> **These will not be sent. Decided by the dev on 2026-08-14, closing #59** — recorded in
> [the README](../README.md#the-four-asks-and-the-two-defect-reports-will-not-be-sent). The text
> below is kept because it is the record of what was drafted, not because it is pending: it
> **asserts nothing about whether anything was sent**, exactly as it did before the decision.
> **The "Before sending" section below is now conditional on a decision that has been made the
> other way** — it stands as the instruction that would apply if the dev ever reverses this,
> and a session must not read it as outstanding work. The two defect reports are separable and
> separately reversible; sending anything later means moving that message up one level under
> the ordinary rule.

## Before sending

**Confirm every recipient against the organisation's own current contact page.** Do not send
to an address invented here or anywhere else.

[#59](https://github.com/winniel123/verge-asm/issues/59) ran that confirmation on 2026-08-14
for the addresses that could be retrieved. **Four of the six now carry a first-party address**
— one retrieved from the organisation's own page, quoted below with the URL it came from — and
**two are still `TO CONFIRM`**, both LACNIC's, because LACNIC's contact page cannot be
retrieved. Fetching `lacnic.net/630/2/lacnic/contact-us` returns the same JavaScript shell that
message 6 exists to report, so the defect blocks confirmation of the address the defect would
be reported to. That is a finding, not an oversight: **open LACNIC's contact page in a browser
and read the address off it.**

Provenance for each retrieved address is stated inline. An address carried by a search
engine's rendering rather than by a direct retrieval is marked as such and is **not**
confirmation — it is a pointer to where to look.

**Send six separate messages, not four.** The two defect reports go on their own, to whatever
address each registry publishes for its website or documentation. Bundling them into the asks
is why both are still unreported.

---

## 1 — RIPE NCC · RIPEstat and the RIPE Database

**To:** `stat@ripe.net` (evidenced — RIPEstat's documented address for above-1000/day
registration, taken by [#19](https://github.com/winniel123/verge-asm/issues/19) from RIPEstat's
own documentation).
**Cc:** `ripe-dbm@ripe.net` — the RIPE Database administration team, which is the separate
contact this draft asked for. Retrieved 2026-08-14 from RIPE's own database documentation,
<https://docs.db.ripe.net/FAQ/>: *"You can get further help by sending an email to
ripe-dbm@ripe.net."* Questions 3–5 below are RIPE Database questions, not RIPEstat ones, so
this address is the one that can actually answer the half of the message that matters most.
**Ticket:** [#19](https://github.com/winniel123/verge-asm/issues/19)
**Subject:** Query permissions for an open-source, self-hosted attack-surface tool

> **Stale as of 2026-08-15, in one respect, and it would be sent as a false statement about the
> software.** [#126](https://github.com/winniel123/verge-asm/issues/126) /
> [ADR-0063](../../adr/0063-a-routing-announcement-names-the-path-not-the-estate.md) rules the BGP leg
> out of v1, so `announced-prefixes` is a call verge-asm will not make and question 1 below asks
> permission for a use that does not exist. Not rewritten — a draft is a record of what was drafted —
> but if the dev ever reverses #59 for this message, **strike `announced-prefixes` from the calls list
> and drop question 1's *"writing announced prefixes into an operator's own inventory database"*
> clause first.** The RIPE Database half of the message (questions 3–5) is untouched.

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

**To:** `helpdesk@apnic.net` — APNIC's published helpdesk. Retrieved 2026-08-14 from APNIC's
own email policy page, <https://www.apnic.net/about-apnic/apnic-email-policy/>. APNIC's
`/get-ip/helpdesk/` now 301s to a Salesforce contact form at `help.apnic.net` that publishes no
address, so the email policy page is the first-party source that still names one.
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

**To:** `membership@afrinic.net` — the address AFRINIC publishes for **Registry Queries**.
Retrieved 2026-08-14 from AFRINIC's own contact page, <https://afrinic.net/contact>, which
lists exactly three addresses: `membership@afrinic.net` (Registry Queries),
`support@afrinic.net` (Technical Support) and `press@afrinic.net` (Media & Press).

This draft originally said *AFRINIC hostmaster*, which #57 could not resolve because the terms
text [#20](https://github.com/winniel123/verge-asm/issues/20) read had redacted the address.
`hostmaster@afrinic.net` does appear across AFRINIC's WHOIS documentation, but it is **not** on
the current contact page and this message is a permitted-use question rather than a WHOIS
object request — so the published Registry Queries address is the right one, and the
substitution is deliberate.
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

**To:** `TO CONFIRM` — **still unconfirmed, and the reason is this message's own subject.**
LACNIC's contact page, <https://www.lacnic.net/630/2/lacnic/contact-us>, returns the same
JavaScript shell to a plain HTTP request that the terms URLs do, so no address can be read off
it directly. A search engine's rendering of that page reports `hostmaster@lacnic.net` as
LACNIC's address for resource requests — **treat that as a pointer, not a confirmation**, and
open the page in a browser before sending.
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

**To:** `support@afrinic.net` — the address AFRINIC publishes for **Technical Support**, which
is the nearest of the three on its contact page to a website defect. Retrieved 2026-08-14 from
<https://afrinic.net/contact>. AFRINIC publishes no separate webmaster address.
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

**To:** `TO CONFIRM` — **unconfirmable for the same reason as message 4**, and here the
circularity is total: LACNIC publishes no retrievable address to report that its pages are
unretrievable. Read it off the contact page in a browser, or send this to whatever address
message 4 goes to.
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
> It is not confined to the policy pages. Your contact page at
> `lacnic.net/630/2/lacnic/contact-us` behaves the same way — a plain HTTP request returns only
> the document title and no contact details. I mention it because it means someone who cannot
> read your pages also cannot find out where to tell you, which is the position I was in
> writing this.
>
> No reply needed.
>
> Thank you,
> Logan Winnie · verge-asm
