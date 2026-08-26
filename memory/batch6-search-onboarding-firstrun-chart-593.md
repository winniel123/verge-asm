---
name: batch6-search-onboarding-firstrun-chart-593
description: Batch 6 (screens 19–20) wayfinder chart — SearchResults + Onboarding + FirstRun, package v3.12.0
metadata:
  type: project
---

Batch 6 of the screen-parity conversion effort — **CHARTED not built** 2026-08-25. Wayfinder map [#593](https://github.com/winniel123/verge-asm/issues/593), source work order `design-system/verify/WORK-ORDER-19-20-BATCH6.md`. Filed and STOPPED per [[wayfinders-mean-file-not-implement]]; do not implement/merge without an explicit ask.

**Destination:** `/search`, `/onboarding`, empty-estate `/` home byte-served from frozen `search.tmpl` / `onboarding.tmpl` / `firstrun.tmpl` (v3.12.0); the three `templates_*.go` deleted; SPEC-CHANGE #25(a–f) wired; G1+G2 green @1440 × light/dark.

**Route (4 children, frontier = #594):**
- [#594](https://github.com/winniel123/verge-asm/issues/594) Foundation — land v3.12.0, embed 3 tmpls inert, extend G1 manifest, restore SPEC-CHANGE #20–24 + transcribe #25. No blockers → **takeable now**.
- [#595](https://github.com/winniel123/verge-asm/issues/595) Screen 19 SearchResults — blocked by #594.
- [#596](https://github.com/winniel123/verge-asm/issues/596) Screen 20 Onboarding+FirstRun (ONE branch, both tmpls) — blocked by #594.
- [#597](https://github.com/winniel123/verge-asm/issues/597) Serial-merge + operator sign-off — blocked by #595, #596.

**What differs from [[batch5-reports-reportartifact-inbox-chart-586]]:**
- **No intra-batch template coupling** (unlike batch 5 where s17→s16). Both new tmpls parse against ALREADY-LANDED defines: `search.tmpl`→`sevbadge` (signals.tmpl), `firstrun.tmpl` bare-define wrapped by `dashboard.tmpl` "home" when `.EmptyEstate`, `onboarding.tmpl` self-contained. So #595 and #596 both block only on Foundation → fully parallel.
- **Screen 20 = one ticket, two templates** (#20 is one map item): `onboarding.tmpl` + `firstrun.tmpl` on a single branch.
- **Foundation restores 5 SPEC-CHANGE rows, not 3** — the v3.12.0 export re-truncated the collision log to #19 (removed #20–#24), so #594 restores #20–#24 verbatim from `main` + transcribes new #25. (Package is already dropped in the working tree, uncommitted, at chart time.)

**New backend build this batch (flag, don't stub — escalate a new collision if unsourceable):** search field-segmentation read splitting each matched field on the FIRST case-insensitive query occurrence into `[{Text,Hit}]` for the new "hisegs" define (#25a); onboarding server-computed `.StepValid` no-JS floor + handler absorbing typed seed text on submit (#25d/e); FirstRun `firstRunChecklist` real reads with step-4 gated until an internet vantage exists, `.ActionPost` enqueuing first batch (#25f).

Next after sign-off = screens 21–22 (Settings, Shell) — both SOLO per batching rule. See [[workflow-v4-screen-conversion-playbook]] for reusable gotchas; [[take-my-recommendations-by-default]] (AFK).
