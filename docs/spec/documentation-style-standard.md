# Documentation style standard

- **Status:** Accepted — spec content for [#783](https://github.com/winniel123/verge-asm/issues/783)
- **Wayfinder map:** [Documentation style SPEC (#783)](https://github.com/winniel123/verge-asm/issues/783)
- **Ticket:** [#788 Assemble and write the Documentation Style Standard SPEC](https://github.com/winniel123/verge-asm/issues/788)

This document is a standing standard. It defines how two disciplines apply to verge-asm
documentation:

1. **ASD-STE100** — Simplified Technical English. The skill lives in-repo at
   `.claude/skills/asd-ste100/`.
2. **The i-have-adhd format subset** — a static-document subset of the i-have-adhd plugin
   (v0.2.0).

The standard governs new docs. Its first application is the rewrite of the existing documentation
corpus (82,676 lines). That rewrite is a separate downstream effort. This document does not perform
it.

**A rewrite changes form only.** The decision, the rationale, and every technical claim stay
identical. A change of meaning is not a style rewrite. It becomes its own ADR.

This document itself follows STE-flavored mode (see §2).

---

## 1. Scope

### 1.1 In scope

| Path | Family |
| --- | --- |
| `docs/adr/*` | ADRs |
| `docs/spec/*` | specs |
| `docs/agents/*` | agent docs |
| `docs/guides/*` | guides |
| `docs/research/*` | research |
| `CONTEXT.md` | domain model |
| `CLAUDE.md` | agent config |
| `README.md` | project overview |
| `SECURITY.md` | security policy |

**A family row reads to any depth**
([ADR-0156](../adr/0156-a-documentation-family-is-a-directory-tree-so-a-nested-doc-is-in-scope-at-any-depth-and-keeps-its-family.md)).
Read `docs/adr/*` and its four siblings as `docs/adr/**`. A `.md` file in a subdirectory of a family
is in that family, and it keeps that family's §3 verdict. The four root files are exact names.

### 1.2 Out of scope

- `docs/correspondence/` — external comms drafts. Voice is the point.
- `docs/wayfinder/` — session-map archives. Process history, not product docs.
- `CHANGELOG`, generated files, and token files.
- `docs/guides/embed.go` — a Go source file, not a doc.

---

## 2. The two axes

Two axes apply independently. The layering rule sets their order:

> **STE governs the sentence. The ADHD format governs the document and section structure.**

### 2.1 STE mode (whole-doc, chosen by audience)

STE mode is a whole-document property. The audience selects it.

- **Strict** — every STE rule, the hard caps, and one-word-one-meaning discipline. Use it for
  agent-parsed text. A misread has a real cost, because an agent has no human to resolve the
  ambiguity.
- **STE-flavored** — the structural rules in full. The lexical rules are advisory. Use it for
  human prose.

The skill ships no dictionary, so the lexical rules drop from a pass/fail gate to a preference for
plain words. This is why the two modes exist.

### 2.2 The ADHD format subset (per-section, chosen by content type)

The ADHD subset is a per-section property. The content type selects it.

- A **procedural section** gets the subset.
- A **reference section** does not.

**A procedural section is an ordered sequence of actions the reader performs.** Examples: a setup,
a runbook, a troubleshooting list, a report-a-vulnerability procedure, or the steps in an agent
HOWTO. Boundary test:

> Does the reader do these in order?

If yes, the section is procedural. Everything else is reference: definitions, rationale, decision
records, glossary entries, and design discussion.

---

## 3. Per-family table

This table sets the STE mode and the ADHD verdict for each family.

| Family | STE mode | ADHD subset | Special constraint |
| --- | --- | --- | --- |
| guides (`docs/guides/*`) | STE-flavored | Yes — whole doc is procedural | Prime ADHD target. |
| specs (`docs/spec/*`) | STE-flavored | Procedural sections only | Acceptance checklists render as numbered or step lists. |
| ADRs (`docs/adr/*`) | STE-flavored | No | Decision frozen. Titles included, not exempt. Escalate a thesis that STE cannot express. Do not change it silently. |
| research (`docs/research/*`) | STE-flavored | No | Quoted primary-source text is frozen. STE touches the surrounding prose only. |
| `CONTEXT.md` | Strict | No | Glossary genre. The domain-modeling CONTEXT format governs the structure. Strict governs the definitions. |
| `CLAUDE.md` | Strict | No | writing-for-agents. Index and config genre, not a HOWTO. |
| `docs/agents/*` | Strict | Yes | writing-for-agents. Agent HOWTO. The ADHD subset reinforces the agent-doc rules. |
| `README.md` | STE-flavored | Setup and quickstart steps only | Overview and rationale stay reference prose. |
| `SECURITY.md` | STE-flavored | Report-a-vuln procedure only | Disclosure commitments are frozen factual claims. STE reforms the wording, never the commitment. |

---

## 4. STE rules

### 4.1 Mechanical gates (pass/fail)

A reviewer verifies each gate with a fixed rule and no judgment. Strict mode and STE-flavored mode
both enforce all five.

1. **No semicolons.** Search the text for `;`. Any hit fails. (STE Rule 8.1.)
2. **Sentence length cap.** Count words per sentence. Fail above 20 words for an instruction. Fail
   above 25 words for a description.
3. **Noun-cluster cap.** Count nouns stacked as one modifier. Fail at 4 or more. Example that
   fails: "high pressure fuel pump inlet valve assembly".
4. **Simple tenses.** Flag present perfect and other compound or auxiliary verb forms ("have
   received", "has completed"). A compound form may stay when it carries current relevance or a
   hedge. That keep-decision is a judgment residue (see §4.3).
5. **No phrasal verbs.** Flag a verb plus a particle ("take off", "spin up", "kick off"). (STE Rule
   9.3.)

### 4.2 Review prompts (judgment, not a hard gate)

A reviewer applies these by reading. Each keeps a judgment residue, so it is not a pass/fail gate.

- **Active voice.** Detect a passive form (a form of "be" plus a past participle). The allowed
  exception is an actor that is genuinely unknown or irrelevant.
- **One instruction per sentence.** More than one instruction in a sentence fails. Some second
  instructions need a human to see them.
- **No ellipsis.** Each sentence keeps its subject, verb, and article.
- **Lists for sequences.** Three or more steps or conditions in a prose sentence should become a
  numbered or bulleted list.
- **Paragraph limit.** Fail a paragraph above 6 sentences. One topic per paragraph.

### 4.3 Pure judgment (never a mechanical gate)

- **Keep modality.** The reviewer compares the rewrite against the source hedge. "may have failed"
  stays "may have failed". This is the most common failure the skill flags. Never upgrade a hedge
  to a fact.
- **The lexical rules.** One word one meaning, one part of speech per word, verb not noun, and
  domain terms defined once. Each needs the dictionary the skill does not ship. Strict mode keeps
  the discipline. Neither mode cites a lexical rule as a pass/fail gate.

### 4.4 What each mode enforces

- **Strict mode** enforces §4.1, §4.2, and §4.3 (including one-word-one-meaning discipline). Use it
  for `CONTEXT.md`, `CLAUDE.md`, and `docs/agents/*`.
- **STE-flavored mode** enforces §4.1 and §4.2. It treats §4.3 lexical rules as advisory. It keeps
  "keep modality" as a hard rule, because a changed hedge changes meaning. Use it for guides,
  specs, ADRs, research, `README.md`, and `SECURITY.md`.

---

## 5. The ADHD format subset

Eight rules port to a static document. Apply them to a procedural section only (see §2.2 and §3).

1. **Lead with the next action.** A section starts with the action. The command, the path, or the
   snippet goes first.
2. **Number multi-step tasks.** Each step is one bounded action.
3. **End with one concrete next action.** A section ends with one named next step.
4. **Suppress tangents.** Finish the first topic. Move a second topic to a separate section or a
   note.
5. **Matter-of-fact tone for errors.** A troubleshooting entry states the cause and the fix.
6. **Cap lists at 5 items.** This limit applies to any list.
7. **No preamble, no recap, no closing pleasantries.** A section starts with the answer and ends
   when the answer is done.
8. **Pre-send edit pass.** Delete the announcing first sentence. Delete the closing question.
   Delete the sidebar. Delete empty hedges. Delete idioms.

Four rules are session-runtime only. They do not port to a static document. Do not apply them:

- Restate state every turn.
- Give a live time estimate.
- Make completed work visible.
- The persistence and mode-toggle mechanics.

---

## 6. Frozen content

STE reforms the wording. It never changes a frozen fact. If STE and a frozen claim cannot co-exist,
**escalate the claim. Do not change it silently.**

| Frozen content | Family | Rule |
| --- | --- | --- |
| The encoded decision and its rationale | ADRs | The decision stays identical. A meaning change becomes a new ADR. |
| The ADR title | ADRs | The title is not exempt from STE. If a title encodes a thesis STE cannot express, escalate the thesis. Never reword it silently. |
| Disclosure commitments (timelines, scope) | `SECURITY.md` | STE reforms the wording. It never changes a number or a promise. A reworded promise is a changed promise. |
| Quoted primary-source text | research | STE touches the surrounding prose only. The quote and the data tables stay verbatim. |
| A technical claim, number, or scope qualifier | all | Every claim is kept. A rewrite that drops a hedge or a fact has failed. |

---

## 7. Acceptance

Acceptance is a per-family checklist plus a reviewer. No linter sits on the critical path. An
automated lint tool is deferred (see §9).

For each doc, the reviewer runs this checklist:

1. **Family and mode.** Confirm the STE mode and the ADHD verdict against the §3 table.
2. **Mechanical gates.** Apply the five §4.1 gates. Any hit fails.
3. **Review prompts.** Apply the §4.2 prompts by reading.
4. **Frozen content.** Confirm every §6 frozen claim is intact. Confirm no hedge became a fact.
5. **Structure.** For a procedural section, confirm the §5 subset is applied. For a reference
   section, confirm it is not.

---

## 8. Execution tiers

The downstream rewrite runs in three tiers. A tier sets the order of the work, not a different
standard.

| Tier | Content | Rationale |
| --- | --- | --- |
| **T1** | user-facing: guides, specs, `README.md`, `CONTEXT.md`, agent docs (`docs/agents/*`) | Highest read volume. An agent loads `CONTEXT.md` and `CLAUDE.md` every session. |
| **T2** | ADRs (`docs/adr/*`) | Decision records. Frozen decisions raise the review cost per doc. |
| **T3** | research (`docs/research/*`) | Reference. Lower read volume. Heavy frozen quotes. |

---

## 9. Not yet specified

- **An automated STE and ADHD lint tool.** Deferred per §7. It may become its own effort after this
  standard lands. It is a preference-of-travel, not a gate.

---

## 10. Worked examples

One before-to-after example per family. Each uses a real snippet from an existing doc. Each "after"
applies only that family's verdict from §3.

### 10.1 Guides — `docs/guides/authentication.md`

**Verdict: STE-flavored + ADHD (a procedural section).**

**Before**

> ### Enrolling
> 1. On **Profile → Credentials**, start enrollment (`POST /account/totp/enable`).
>    A fresh 160-bit secret is generated and rendered as a **QR code** to scan and
>    as **base32 text** for manual entry. The QR is drawn in-process — the secret
>    never leaves the origin. At rest the secret is **encrypted** with a file-backed
>    key before it touches Postgres; a database dump discloses no usable secret.
> 2. Scan the QR (or type the secret) into your authenticator app.
> 3. Enter the current 6-digit code to confirm (`POST /account/totp/confirm`). A
>    wrong code re-renders the enrollment screen and **does not** enable two-factor.

**After**

> ### Enrolling
> 1. Open **Profile → Credentials**. Start enrollment (`POST /account/totp/enable`).
> 2. Scan the QR code, or type the base32 secret, into your authenticator app.
> 3. Enter the current 6-digit code to confirm (`POST /account/totp/confirm`). A wrong
>    code does not enable two-factor. The screen re-renders for another try.
>
> **How the secret is protected.** The server generates a fresh 160-bit secret. It draws the
> QR code in-process. The secret never leaves the origin. The server encrypts the secret with a
> file-backed key before the secret reaches Postgres. A database dump discloses no usable secret.

**What changed and why:** ADHD *lead with the action* and *suppress tangents* — step 1 held one
action plus three security facts, so the facts move to a note under the list. STE *one idea per
sentence*, *active voice*, *no semicolon*. Every technical claim is kept: 160-bit, in-process QR,
at-rest encryption, "no usable secret", "does not enable".

### 10.2 Specs — `docs/spec/v1-spec.md`

**Verdict: STE-flavored + ADHD (a "how to read" section is procedural).**

**Before**

> **How to read this**: start here for the shape of the whole system, then zoom into
> `CONTEXT.md` for the domain model's full text, the ADRs under `docs/adr/` for the
> reasoning behind any one rule, and the spec files under `docs/spec/` (…) and the research
> notes under `docs/research/` for enumerations that are lists rather than decisions and will
> be revised without a new ADR.

**After**

> **How to read this**
> 1. Start here for the shape of the whole system.
> 2. Read [`CONTEXT.md`](../../CONTEXT.md) for the full domain model.
> 3. Read the ADRs under `docs/adr/` for the reasoning behind any one rule.
> 4. Read the component spec files under `docs/spec/`.
> 5. Read the research notes under `docs/research/` for the enumerations. These are lists, not
>    decisions. They change without a new ADR.

**What changed and why:** one 60-word sentence became five steps (ADHD *number the steps*, *cap at
5*). STE *no ellipsis*. The hedge "lists rather than decisions … revised without a new ADR" is kept
in meaning.

### 10.3 README — `README.md`

**Verdict: STE-flavored, ADHD No (a narrative description, not a procedure).**

**Before**

> verge-asm takes the seed domains and IP ranges you declare, discovers the internet-exposed
> assets that grow from them, actively probes what it is allowed to probe, and makes **exposure
> drift a first-class, queryable object** — a timeline tracked across ports, certificates, DNS
> records and HTTP identity, not subdomains alone.

**After**

> You declare seed domains and IP ranges. verge-asm discovers the internet-exposed assets that
> grow from them. It actively probes what it is allowed to probe. It makes **exposure drift a
> first-class, queryable object** — a timeline across ports, certificates, DNS records, and HTTP
> identity, not subdomains alone.

**What changed and why:** one 45-word sentence split into four (STE *sentence cap*). No numbering
was added. The README verdict gives ADHD to *procedural* sections only. An install or run section
would get it. This narrative does not. The scope qualifier "not subdomains alone" is kept.

### 10.4 SECURITY — `SECURITY.md`

**Verdict: STE-flavored + ADHD on the procedural steps. Disclosure commitments FROZEN.**

**Before**

> This opens a private channel visible only to the maintainers. If Private Vulnerability
> Reporting is unavailable to you, open a minimal public issue that says only "requesting a
> private security contact" — with **no exploit details** — and a maintainer will follow up.

**After**

> This opens a private channel. Only the maintainers can see it. If Private Vulnerability
> Reporting is not available to you, open a minimal public issue. Write only "requesting a
> private security contact". Include **no exploit details**. A maintainer will then respond.

**What changed and why:** one conditional sentence split (STE). The phrasal verb "follow up" became
"respond". The exact quoted string and the "no exploit details" condition are kept. **Freeze
demonstration:** the "Our commitment" list (acknowledge within **3 business days**, assess within
**10 business days**, credit unless anonymous) is a disclosure commitment. It is copied verbatim.
STE never rewords a number or a promise, because a reworded promise is a changed promise.

### 10.5 ADR — `docs/adr/0001-stack-and-runtime.md`

**Verdict: STE-flavored, title included, decision FROZEN, ADHD No.**

**Before (Context sentence)**

> verge-asm is an AGPL-3.0, self-hosted, single-tenant web application, deployed via
> `docker compose`, whose differentiation is **exposure drift as a first-class, queryable
> object with its own lifecycle** across ports, certificates, DNS records and HTTP identity.

**After**

> verge-asm is an AGPL-3.0, self-hosted, single-tenant web application. You deploy it with
> `docker compose`. It makes **exposure drift a first-class, queryable object with its own
> lifecycle**. It tracks that object across ports, certificates, DNS records, and HTTP identity.

**Title check:** `ADR-0001: Stack and runtime for web, worker, and persistence` is already
STE-clean. It is kept verbatim. If a title encoded a thesis STE could not express, the rule is to
escalate the thesis, never reword it silently.

**What changed and why:** one 34-word sentence split into four (STE). The encoded decision — the
differentiator and its four facets — is identical. No ADHD: ADRs are reference.

### 10.6 Research — `docs/research/prior-art-oss-asm.md`

**Verdict: STE-flavored, ADHD No (reference). Data tables stay as data.**

**Before**

> **Method.** Every claim below is sourced to the project's own README, its own documentation
> site, its own source code, or the vendor's own pricing page. No listicles, no "top 10 ASM
> tools" posts. Where a README claim and the source code disagreed, the source code wins and the
> discrepancy is noted.

**After**

> **Method.** Every claim below has a first-party source: the project's own README, its own
> documentation site, its own source code, or the vendor's own pricing page. It uses no listicles
> and no "top 10 ASM tools" posts. Where a README claim and the source code disagreed, the source
> code wins. This document notes each discrepancy.

**What changed and why:** the first sentence became active with a named source list. The last
sentence split (STE *one idea*). The method's rules are kept. **Boundary note:** STE governs prose
sentences only. The comparison-table cells are data, not prose, and stay unchanged.

### 10.7 CONTEXT.md — `CONTEXT.md`

**Verdict: Strict (agent-parsed), ADHD No.**

**Before**

> Self-hosted attack surface management for an operator's own estate. Its subject is not inventory
> but **change**: what is exposed to the internet, and what moved since last time. Change is the
> lens, and the only thing the comparison path reads — yet the **same measured corpus** answers the
> complementary *what do I have right now?* when each open `Span` is read **forward by subject** as
> **inventory** rather than diffed over time.

**After**

> verge-asm is self-hosted attack surface management for an operator's own estate. Its subject is
> not inventory. Its subject is **change**: what is exposed to the internet, and what moved since
> last time. Change is the lens. Change is the only thing the comparison path reads. The **same
> measured corpus** also answers a second question: *what do I have right now?* To answer it, the
> system reads each open `Span` **forward by subject** as **inventory**. It does not diff those
> `Span`s over time.

**What changed and why:** Strict mode. The 60-word compound sentence became short single-idea
sentences. Active voice throughout. The ADR-0105 claim (inventory is a read, not a second thesis)
and the terms "open `Span`" and "forward by subject" are kept exactly.

### 10.8 CLAUDE.md — `CLAUDE.md`

**Verdict: Strict (agent-parsed), ADHD No.**

**Before**

> `design-system/` is the shared UI asset home and the source of truth: `templates/` and `tokens/`
> are embedded and served by the web app (via `design-system/designfs.go`), `tokens/` and
> `components/` are consumed by the docs-site, and all of it may be edited in-repo.

**After**

> `design-system/` is the shared home for UI assets. It is the source of truth. The web app embeds
> and serves `templates/` and `tokens/` through `design-system/designfs.go`. The docs-site reads
> `tokens/` and `components/`. You may edit all of it in the repo.

**What changed and why:** Strict mode. Passive "are embedded and served" and "are consumed by"
became active. The 4-word noun cluster "shared UI asset home" became "shared home for UI assets"
(noun-cluster cap, §4.1). Every path and fact is kept.

### 10.9 docs/agents — `docs/agents/issue-tracker.md`

**Verdict: Strict (agent-parsed) + ADHD (a procedural agent instruction).**

**Before**

> **Blocking**: GitHub's **native issue dependencies** — the canonical, UI-visible representation.
> Add an edge with `gh api --method POST … -F issue_id=<blocker-db-id>`, where `<blocker-db-id>` is
> the blocker's numeric **database id** (`gh api repos/<owner>/<repo>/issues/<n> --jq .id`, _not_ the
> `#number` or `node_id`). Where dependencies aren't available, fall back to a `Blocked by: #<n>`
> line at the top of the child body.

**After**

> **Blocking.** Use GitHub native issue dependencies. This is the canonical, UI-visible
> representation. To add a blocking edge:
> 1. Get the blocker's numeric **database id**. Run `gh api repos/<owner>/<repo>/issues/<n> --jq .id`.
> 2. Do not use the `#number`. Do not use the `node_id`.
> 3. Add the edge. Run `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`.
>
> If dependencies are not available, use a `Blocked by: #<n>, #<n>` line at the top of the child body
> instead. A ticket is unblocked when every blocker is **closed**. The gate is each blocker's
> `state`, never a count.

**What changed and why:** Strict + ADHD. The phrasal verb "fall back" became "use … instead". The
db-id-not-number warning became two flat imperatives an agent cannot misread (ADHD *plain error
tone*, *lead with the action*). The state-not-count rule is kept.
