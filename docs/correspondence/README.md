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

## One habit worth carrying

**Do not bundle a question the project needs with a courtesy the source needs.**

[#24](https://github.com/winniel123/verge-asm/issues/24) and
[#25](https://github.com/winniel123/verge-asm/issues/25) each folded a defect report into an
ask — AFRINIC's documented terms URL returning 404, LACNIC serving a script shell at five
documented policy addresses. Both defects are still unreported, because they were attached to
a question that is itself unevidenced. They have different recipients and different urgency,
only one of them is ADR-0003's business, and the courtesy has no fallback: it is simply
dropped with the question. Send them separately.
