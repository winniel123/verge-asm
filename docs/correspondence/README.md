# Correspondence

Outbound messages this project sends to a third party, and the replies they get.

Established by [#57](https://github.com/winniel123/verge-asm/issues/57) and recorded as
[ADR-0003](../adr/0003-third-party-source-consent-bar.md)'s fourth amendment.

## Why this exists

Outbound messages were the only class of project act in this effort with no artefact. Every
measurement has a research note, every decision has a ticket, most have an ADR, every drawn
surface has a prototype directory. Four asks to the regional registries had nothing —
[#19](https://github.com/winniel123/verge-asm/issues/19),
[#23](https://github.com/winniel123/verge-asm/issues/23),
[#24](https://github.com/winniel123/verge-asm/issues/24),
[#25](https://github.com/winniel123/verge-asm/issues/25) recorded no send confirmation, no
recipient for three of the four, and no send date.

The cost was not abstract. Four operator-facing surfaces went on to say *asked, nobody
replied*, and `Coverage` shipped **"Asked 2026-06-02. Nobody replied."** — a fabricated date,
not a stale one, since it precedes this repository's first commit by two months. A surface
needed a precise fact, no artefact existed to supply one, and the prose invented one.

## The rule

**The project may only claim an outbound act it can produce.**

This is [ADR-0005](../adr/0005-scan-execution-model.md)'s rule turned on the project's own
conduct rather than on a `Batch`: *record what completed, never what was attempted.* A
`Batch` that logged its attempted scope would assert absences it never measured; a ticket
that logs an intention to write asserts a question it never put.

**Absence of a file here is evidence of absence.** That is what the directory buys. A surface
may say *no record of an approach exists* today and say something stronger the moment a file
lands, and neither statement is a guess.

## Layout

```
docs/correspondence/
├── README.md
├── drafts/                          ← evidence of nothing
└── YYYY-MM-DD-<recipient>-<slug>.md ← a message that was actually sent
```

**Drafts are not correspondence.** Anything under `drafts/` is unsent text that a session or
the dev prepared; it asserts nothing about whether it was sent, and no surface may read it.
Sending means **moving the file up one level** and filling in the recipient and date. Keeping
the two in one directory would rebuild the exact defect this directory exists to fix.

## What a sent message records

One file per message. Not a note saying a message was sent — the message.

- **Recipient** — the address it actually went to, not the organisation.
- **Sent** — the date it actually went, not the date it was written.
- **Motivating ticket** — the issue that asked for it.
- **Subject** and **body**, verbatim as sent.
- **Replies**, appended verbatim as they arrive, each with its own date. A reply is the only
  thing that reopens a question this project has closed as unanswered — elapsed time is not.

Bounces and delivery failures are recorded in the same file, under the same rule that governs
the consent record: a failure stored in the slot where the document would have gone is a
better record than a silence.

## The mailbox was searched once, and it was empty

[#59](https://github.com/winniel123/verge-asm/issues/59) step 3 — *search your own sent mail* —
ran on **2026-08-14**. Recorded here rather than in the issue alone so that no later session
re-runs it, and so that nobody reads more out of it than it holds.

**Searched:** `logan.m.winnie@gmail.com`, via the Gmail API, over **all folders including sent,
archive, spam, trash and drafts**, with no date bound. Six queries: the four registry domains
as body text; the four domains as `To:`, `From:` and `Cc:` headers; the registry names and
`"regional internet registry"`; `RIPEstat`, `verge-asm`, `RDAP`, `whois`, `"attack surface"`,
`"permitted use"`, `hostmaster`; and the drafts folder on its own.

**Result:** **nothing.** Every registry-scoped query returned zero threads. The one query that
returned anything returned a handful of unrelated threads — matching on the bare word `whois`
and nothing else.

**The search ran.** A control query over sent mail alone returned a large, non-empty result
set, so the empty results are the mailbox's answer and not a broken query.

### What it settles, and the two things it does not

It **closes the *file the evidence* branch**. There is no message to retrieve and file, so the
`drafts/` directory stays the whole of the correspondence record and the surfaces keep the
weaker true sentence.

It does **not** establish that nothing was sent, and no surface may say so. Two limits, both of
which are the reason this section states its extent:

1. **One mailbox of at least three.** The dev's sent mail shows at least two further sender
   addresses as correspondents; neither was searched, and neither is reachable
   from here.
2. **An empty mailbox is not an empty world.** Mail can be sent from an account that keeps no
   copy, or deleted past recovery.

Between them, [ADR-0003](../adr/0003-third-party-source-consent-bar.md)'s fourth amendment still
governs: *we did not ask* is the same defect with the sign flipped, and whether anything was
sent remains a `Gap` whose cause is that the project kept no record. **This search narrowed
where the evidence could have been. It did not convert an absence of evidence into evidence of
absence**, which is the one move the whole directory exists to prevent.

## The four asks and the two defect reports will not be sent

**Decided by the dev on 2026-08-14**, closing
[#59](https://github.com/winniel123/verge-asm/issues/59): *"I'm not going to send the emails."*
This is the second of the two discharges #59 offered — **recorded as deliberately not sent**,
rather than sent and filed.

**This is a record of a decision, not of an act.** Nothing went to RIPE NCC, APNIC, AFRINIC or
LACNIC. The drafts stay under `drafts/` and stay evidence of nothing. The rule above is
unchanged and now binds in the direction it was written for: **absence of a file at the top
level of this directory is evidence of absence** — and that absence is now *settled* rather
than pending.

**What it discharges.** [ADR-0003](../adr/0003-third-party-source-consent-bar.md)'s
ambiguity-corollary debt, which that ADR states is settled *"by sending or by formally
recording that nothing was sent"*. This is the second branch. The debt was against the ADR,
never against the sources.

**What does not move — which is why the decision is cheap.** No source changes ship state.
[#34](https://github.com/winniel123/verge-asm/issues/34)'s five-registry table stands.
Ambiguous terms ship off indefinitely, the identical result whether the question was asked and
ignored or never put at all. The operator-facing wording is
[#57](https://github.com/winniel123/verge-asm/issues/57)'s, and it is **already correct and now
permanently so**: *no reply has ever come, and no record of an approach exists* is true today
and stays true. The ask date stays unrenderable, because there is no ask.

**What it costs, stated rather than glossed.** Two of the four regions have no keyless fallback
at all ([#19](https://github.com/winniel123/verge-asm/issues/19),
[#25](https://github.com/winniel123/verge-asm/issues/25)), and LACNIC's ask was rated *"the
cheapest of the four and the most likely to resolve cleanly"* — one sentence from LACNIC would
have made it `unencumbered` outright. That gain is **forgone, not lost**: it was never held.

**The two defect reports are included in this disposition** — AFRINIC's documented
`whois/terms` URL returning 404, and LACNIC serving a byte-identical 7,014 B JavaScript shell
at the RDAP `terms-of-service` target and four other documented policy URLs. They were
dispositioned alongside the asks because #59 closes as a unit. They remain **separable and
separately reversible**: different recipients, different urgency, only one of them is
ADR-0003's business, and neither blocks anything. Reporting them later needs no ticket and no
permission — write the message, send it, file it here.

**What reopens this: sending.** Nothing else, and specifically not elapsed time. A message sent
later is filed at the top level under the ordinary rule, and the surfaces gain their stronger
sentence the moment it lands.

## One habit worth carrying

**Do not bundle a question the project needs with a courtesy the source needs.**

[#24](https://github.com/winniel123/verge-asm/issues/24) and
[#25](https://github.com/winniel123/verge-asm/issues/25) each folded a defect report into an
ask — AFRINIC's documented terms URL returning 404, LACNIC serving a script shell at five
documented policy addresses. Both defects are still unreported, because they were attached to
a question that is itself unevidenced. They have different recipients and different urgency,
only one of them is ADR-0003's business, and the courtesy has no fallback: it is simply
dropped with the question. Send them separately.
