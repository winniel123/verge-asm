# ADR-0131: The console is vanilla server-rendered PRG; the htmx stack is withdrawn

- **Status:** Accepted
- **Date:** 2026-08-31
- **Ticket:** [#942 Resolve the ADR-0001 htmx drift: adopt htmx or ratify vanilla](https://github.com/winniel123/verge-asm/issues/942)
- **Map:** [#937 Wayfinder: actions must not throw the operator to the top of the page](https://github.com/winniel123/verge-asm/issues/937)
- **Withdraws, in part:** [ADR-0001](./0001-stack-and-runtime.md)'s *"htmx"* claim at its Decision-table Frontend row (`:45`), its *"heavy client-side filter state is where htmx starts to hurt"* cost note (`:179`), and its *"With htmx the HTTP surface is HTML fragments and SSE"* sentence (`:237`); and the same claim at [`v1-spec.md`](../spec/v1-spec.md) §4.1. Written at each specifying site per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) as sharpened by its [#106](https://github.com/winniel123/verge-asm/issues/106) amendment.
- **Leaves untouched:** the **server-rendered PRG** decision itself (confirmed, not htmx-dependent); the **SSE** live-feed claim, which is likewise unrealized but is *not* superseded by this pass (it keeps a non-striking pointer only); [ADR-0123](./0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md)'s read-only API reversal.

## Context

ADR-0001's Decision table records the frontend as *"**Server-rendered** `html/template` + **htmx**, SSE for the live drift feed"*. The server-rendered half is real. The **htmx** half was never built.

Map [#937](https://github.com/winniel123/verge-asm/issues/937) root-caused the scroll-jump on console actions ([#938](https://github.com/winniel123/verge-asm/issues/938)) and hardened the existing scroll-restore under one same-URL PRG contract ([#945](https://github.com/winniel123/verge-asm/issues/945) → [ADR-0130](./0130-scroll-restore-is-hardened-by-a-same-url-prg-plus-full-url-key-contract.md)). One mechanism question was left open for class D — the actions that fire inside a modal, drawer, or inner-scroll container: adopt htmx to move those to in-place updates, or ratify the vanilla status quo. [#942](https://github.com/winniel123/verge-asm/issues/942) resolved it.

**[#942] measured the drift.** There are **zero** `hx-*` attributes and **zero** htmx includes anywhere in `design-system/templates/` or `cmd/web`. htmx appears **only** in docs — ADR-0001 `:45`/`:179`/`:237` and `v1-spec.md` §4.1. The real console is server-rendered POST → 303 → GET plus hand-rolled inline vanilla JavaScript. The htmx claim is a two-year-old aspiration that the build never took up, and a reader meeting it in the present tense is told the console runs on a stack that is not present.

## Decision

**The console is vanilla server-rendered PRG plus hand-rolled vanilla JavaScript. No in-place fetch/swap layer — htmx or bespoke — is built. The htmx claim in ADR-0001 and the spec is withdrawn.** Three parts.

### 1. Vanilla is ratified; htmx is withdrawn

Server-rendered PRG plus inline vanilla JavaScript **is** the console's render architecture, not a stopgap before htmx. The htmx claim is withdrawn at every operative-voice site that carries it, each pointing here (Consequences). This is a **withdrawal**, not a deletion: the sites keep their reasoning and gain a dated pointer, per the name-and-withdraw convention ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).

### 2. The class-D in-place layer is out of scope

No in-place layer is built, so the htmx-vs-vanilla mechanism pick is **moot**. ADR-0130's hardening — land back at the exact path+query, restore window scroll — already covers the class-D cases that matter:

- The notable cases are **fire-and-close**. The descope-confirm modal (`signals.tmpl:351-369` → `POST /exclusions` → 303) and the schedule kebab (`reports.tmpl:285-296`, Run-now / Delete → 303) should close after the act, so a full reload landing back in place is the correct result, not a residue.
- Many class-D modals are **URL-addressable** (descope via `DescopeHref`; settings dialogs and the member kebab via query-string PRG), so ADR-0130 reopens them for free — that is class-E/A hardening, not an in-place need.
- The genuine residue is narrow: inner-container **scroll offset** (not open-state) in about two drawers (`settings.tmpl` integration drawer at `:210`; possibly the signal drawer). ADR-0130 restores window scroll, not inner-container scroll. It is low-severity and touches few actions.

### 3. SSE is not touched here

ADR-0001 also claims **SSE** for the live drift feed. SSE is likewise unrealized — the only live-update mechanism is a `fetch`-poll (`rundetail.tmpl:236-247`). This pass supersedes **htmx**, not SSE, so ADR-0058 does not oblige its withdrawal here. The htmx sites carry a **non-striking** pointer noting SSE is unrealized, only so a reader does not silently re-bless it. SSE stays out of this effort's scope.

## Rationale

### The claim is 100% unrealized, so it can only mislead

A present-tense sentence that names a stack the codebase does not contain is exactly the failure ADR-0058 exists to stop: a reader *"believes the mechanism exists and never looks"*. A session reading the Frontend row would reach for `hx-*` patterns that are not there, or would preserve htmx as a live constraint on a change that has none.

### ADR-0130 already covers the class-D cases that pay

The in-place layer's whole value was smoother class-D actions. ADR-0130's PRG hardening delivers the right behaviour for the fire-and-close and URL-addressable cases without a new dependency. The margin left over — inner-container scroll offset in about two drawers — is small.

### The cost lands where ADR-0001 already refuses it

Adding a third-party dependency (htmx) or a bespoke fetch/swap layer into the page that renders the operator's **complete attack-surface inventory** is the same supply-chain / second-toolchain cost ADR-0001's *"Server-rendered over SPA"* section argues against. The narrow residue does not pay for it.

## Consequences

- **ADR-0001 is struck at three sites**, each with an inline pointer here (per ADR-0058 #106, since a blockquote cannot live in a table cell the `:45` row carries a bold in-cell pointer):
  - `:45` Decision-table Frontend row — `~~htmx~~` struck in-cell, bold pointer to this ADR.
  - `:179` — the *"heavy client-side filter state is where htmx starts to hurt"* cost note.
  - `:237` — *"With htmx the HTTP surface is HTML fragments and SSE, so no API falls out for free"*.
- **[`v1-spec.md`](../spec/v1-spec.md) §4.1 is annotated** at the same `Server-rendered … + htmx … SSE` sentence.
- **SSE keeps a non-striking pointer** at those sites. It is not withdrawn here and stays out of scope. A future effort may withdraw it on its own terms.
- **Revisit only as a fresh effort.** If, after ADR-0130 actually lands, the inner-container-scroll residue in the settings-integration or signal drawers still hurts an operator, a future effort MAY reconsider a targeted fix for those specific drawers. That is a fresh effort against a redrawn destination, not a resumption of this map.
- **Progressive enhancement holds.** ADR-0130's hardened forms work with JavaScript off. One pre-existing PE nit is recorded in [#942](https://github.com/winniel123/verge-asm/issues/942) and left for a future UI effort, outside this map's scroll-restore destination: the descope modal's submit button ships `disabled` (`signals.tmpl:364`) and only JavaScript enables it on a typed match.
- [`CONTEXT.md`](../../CONTEXT.md) needs no change. No term is added and none is amended — htmx was never a glossary term.
- **No new build falls out.** This ADR removes an unbuilt aspiration; it commands no code change beyond the doc withdrawals above.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Adopt htmx for class-D in-place updates | Ships a third-party dependency into the attack-surface page for a narrow residue ADR-0130 mostly already covers — the supply-chain cost ADR-0001 argues against. Also decides the mechanism the residue does not force |
| Build a bespoke fetch/swap layer | Same cost as htmx without the ecosystem, and a second render path to maintain forever against the PRG one that works with JavaScript off |
| Withdraw SSE in the same pass | Out of this effort's scope. SSE is unrealized too, but this map's destination is the scroll-jump on actions. ADR-0058 obliges the htmx withdrawal here, not SSE's — that is a fresh effort's call. A non-striking pointer prevents a silent re-blessing without over-reaching |
| Delete the htmx sentences | Takes the reasoning and inbound citations with it, and leaves a later reader unable to tell a withdrawal from an omission — the ground ADR-0058's name-and-withdraw convention already rejects |
