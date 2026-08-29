# PROTOTYPE — before→after rewrite, one per doc family

> Throwaway artifact for wayfinder ticket #787 (map #783). Not a doc. Do not land on `main`.
> Assumption stated up front: the `prototype` skill offers a LOGIC or a UI branch. Neither
> fits a **documentation-text** prototype, so the artifact is this Markdown file of
> before→after pairs. The question it answers: *does the per-family rule table (Q13) produce
> rewrites that preserve meaning (Q6) and read right for each family?*

Each pair uses a real snippet from an existing doc. Each "after" applies only that family's
verdict from #786. Rules referenced: STE gates #784, ADHD subset #785, per-family table #786.

Legend for the verdict line:
- **STE mode** — Strict (all rules, agent-parsed) or Flavored (structural only, human prose).
- **ADHD** — Yes (procedural section) or No (reference/description).

---

## 1. Guides — `docs/guides/authentication.md`
**Verdict: STE-flavored + ADHD (this is a procedural section).**

### Before
> ### Enrolling
> 1. On **Profile → Credentials**, start enrollment (`POST /account/totp/enable`).
>    A fresh 160-bit secret is generated and rendered as a **QR code** to scan and
>    as **base32 text** for manual entry. The QR is drawn in-process — the secret
>    never leaves the origin. At rest the secret is **encrypted** with a file-backed
>    key before it touches Postgres; a database dump discloses no usable secret.
> 2. Scan the QR (or type the secret) into your authenticator app.
> 3. Enter the current 6-digit code to confirm (`POST /account/totp/confirm`). A
>    wrong code re-renders the enrollment screen and **does not** enable two-factor.

### After
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
sentence*, *active voice* (was "A fresh secret is generated"), *no semicolon*. Every technical
claim is kept: 160-bit, in-process QR, at-rest encryption, "no usable secret", "does not enable".

---

## 2. Specs — `docs/spec/v1-spec.md`
**Verdict: STE-flavored + ADHD (a "how to read" section is procedural).**

### Before
> **How to read this**: start here for the shape of the whole system, then zoom into
> `CONTEXT.md` for the domain model's full text, the ADRs under `docs/adr/` for the
> reasoning behind any one rule, and the spec files under `docs/spec/` (…) and the research
> notes under `docs/research/` for enumerations that are lists rather than decisions and will
> be revised without a new ADR.

### After
> **How to read this**
> 1. Start here for the shape of the whole system.
> 2. Read [`CONTEXT.md`](../../CONTEXT.md) for the full domain model.
> 3. Read the ADRs under `docs/adr/` for the reasoning behind any one rule.
> 4. Read the component spec files under `docs/spec/`.
> 5. Read the research notes under `docs/research/` for the enumerations. These are lists, not
>    decisions. They change without a new ADR.

**What changed and why:** one 60-word sentence became five steps (ADHD *number the steps*, *cap
at 5*). STE *no ellipsis*. The hedge "lists rather than decisions … revised without a new ADR"
is kept verbatim in meaning.

---

## 3. README — `README.md`
**Verdict: STE-flavored, ADHD No (this is a narrative description, not a procedure).**

### Before
> verge-asm takes the seed domains and IP ranges you declare, discovers the internet-exposed
> assets that grow from them, actively probes what it is allowed to probe, and makes **exposure
> drift a first-class, queryable object** — a timeline tracked across ports, certificates, DNS
> records and HTTP identity, not subdomains alone.

### After
> You declare seed domains and IP ranges. verge-asm discovers the internet-exposed assets that
> grow from them. It actively probes what it is allowed to probe. It makes **exposure drift a
> first-class, queryable object** — a timeline across ports, certificates, DNS records, and HTTP
> identity, not subdomains alone.

**What changed and why:** one 45-word sentence split into four (STE *sentence cap*). No numbering
was added: the README verdict gives ADHD to *procedural* sections only (an install/run section
would get it), not to this narrative. The scope qualifier "not subdomains alone" is kept.

---

## 4. SECURITY — `SECURITY.md`
**Verdict: STE-flavored + ADHD on procedural steps. Disclosure commitments FROZEN.**

### Before
> This opens a private channel visible only to the maintainers. If Private Vulnerability
> Reporting is unavailable to you, open a minimal public issue that says only "requesting a
> private security contact" — with **no exploit details** — and a maintainer will follow up.

### After
> This opens a private channel. Only the maintainers can see it. If Private Vulnerability
> Reporting is not available to you, open a minimal public issue. Write only "requesting a
> private security contact". Include **no exploit details**. A maintainer will then respond.

**What changed and why:** one conditional sentence split (STE), and the phrasal verb "follow up"
became "respond". The exact quoted string and the "no exploit details" condition are kept.
**Freeze demonstration:** the "Our commitment" list (acknowledge within **3 business days**,
assess within **10 business days**, credit unless anonymous) is a disclosure commitment. It is
copied verbatim. STE never rewords a number or a promise, because a reworded promise is a
changed promise. Escalate, do not silently change.

---

## 5. ADR — `docs/adr/0001-stack-and-runtime.md`
**Verdict: STE-flavored, title included, decision FROZEN, ADHD No.**

### Before (Context sentence)
> verge-asm is an AGPL-3.0, self-hosted, single-tenant web application, deployed via
> `docker compose`, whose differentiation is **exposure drift as a first-class, queryable
> object with its own lifecycle** across ports, certificates, DNS records and HTTP identity.

### After
> verge-asm is an AGPL-3.0, self-hosted, single-tenant web application. You deploy it with
> `docker compose`. It makes **exposure drift a first-class, queryable object with its own
> lifecycle**. It tracks that object across ports, certificates, DNS records, and HTTP identity.

**Title check (Q10):** `ADR-0001: Stack and runtime for web, worker, and persistence` is already
STE-clean. It is kept verbatim. If a title encoded a thesis that STE could not express, the rule
is to **escalate the thesis, never reword it silently**.
**What changed and why:** one 34-word sentence split into four (STE). The encoded decision — the
differentiator and its four facets — is identical. No ADHD: ADRs are reference.

---

## 6. Research — `docs/research/prior-art-oss-asm.md`
**Verdict: STE-flavored, ADHD No (reference). Data tables stay as data.**

### Before
> **Method.** Every claim below is sourced to the project's own README, its own documentation
> site, its own source code, or the vendor's own pricing page. No listicles, no "top 10 ASM
> tools" posts. Where a README claim and the source code disagreed, the source code wins and the
> discrepancy is noted.

### After
> **Method.** Every claim below has a first-party source: the project's own README, its own
> documentation site, its own source code, or the vendor's own pricing page. It uses no listicles
> and no "top 10 ASM tools" posts. Where a README claim and the source code disagreed, the source
> code wins. This document notes each discrepancy.

**What changed and why:** the first sentence became active with a named source list. The last
sentence split (STE *one idea*). The method's rules are kept. **Boundary note:** STE governs prose
sentences only. The comparison table cells (dense, quotation-heavy source evidence) are data, not
prose, and are left unchanged.

---

## 7. CONTEXT.md — `CONTEXT.md`
**Verdict: STE-Strict (agent-parsed), ADHD No.**

### Before
> Self-hosted attack surface management for an operator's own estate. Its subject is not inventory
> but **change**: what is exposed to the internet, and what moved since last time. Change is the
> lens, and the only thing the comparison path reads — yet the **same measured corpus** answers the
> complementary *what do I have right now?* when each open `Span` is read **forward by subject** as
> **inventory** rather than diffed over time.

### After
> verge-asm is self-hosted attack surface management for an operator's own estate. Its subject is
> not inventory. Its subject is **change**: what is exposed to the internet, and what moved since
> last time. Change is the lens. Change is the only thing the comparison path reads. The **same
> measured corpus** also answers a second question: *what do I have right now?* To answer it, the
> system reads each open `Span` **forward by subject** as **inventory**. It does not diff those
> `Span`s over time.

**What changed and why:** Strict mode. The 60-word compound sentence became short single-idea
sentences. Active voice throughout. The ADR-0105 claim (inventory is a read, not a second thesis)
and the terms "open `Span`", "forward by subject" are kept exactly.

---

## 8. CLAUDE.md — `CLAUDE.md`
**Verdict: STE-Strict (agent-parsed), ADHD No.**

### Before
> `design-system/` is the shared UI asset home and the source of truth: `templates/` and `tokens/`
> are embedded and served by the web app (via `design-system/designfs.go`), `tokens/` and
> `components/` are consumed by the docs-site, and all of it may be edited in-repo.

### After
> `design-system/` is the shared home for UI assets. It is the source of truth. The web app embeds
> and serves `templates/` and `tokens/` through `design-system/designfs.go`. The docs-site reads
> `tokens/` and `components/`. You may edit all of it in the repo.

**What changed and why:** Strict mode. Passive "are embedded and served" / "are consumed by" became
active ("The web app embeds and serves", "The docs-site reads"). The 4-word noun cluster "shared UI
asset home" became "shared home for UI assets" (noun-cluster cap, #784). Every path and fact is kept.

---

## 9. docs/agents — `docs/agents/issue-tracker.md`
**Verdict: STE-Strict (agent-parsed) + ADHD (procedural agent instruction).**

### Before
> **Blocking**: GitHub's **native issue dependencies** — the canonical, UI-visible representation.
> Add an edge with `gh api --method POST … -F issue_id=<blocker-db-id>`, where `<blocker-db-id>` is
> the blocker's numeric **database id** (`gh api repos/<owner>/<repo>/issues/<n> --jq .id`, _not_ the
> `#number` or `node_id`). Where dependencies aren't available, fall back to a `Blocked by: #<n>`
> line at the top of the child body.

### After
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

---

## Fidelity questions for the human

1. Is the ADHD "suppress tangents → note block" move (example 1) right, or should security facts
   stay inline in the step?
2. Is the README call right — narrative gets STE only, no numbering (example 3)?
3. Is the freeze demonstrated well enough for SECURITY (example 4) and ADR title (example 5)?
4. Any "after" that lost a hedge, condition, or technical claim?
5. Should the SPEC embed exactly these nine, or do any need a different source snippet?
