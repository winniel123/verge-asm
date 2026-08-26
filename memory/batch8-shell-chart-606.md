---
name: batch8-shell-chart-606
description: Batch-8 Shell/chrome (screen 22, LAST) wayfinder map — CHARTED not built; frontier + the one structural gotcha.
metadata:
  type: project
---

**Batch 8 · Shell/chrome (screen 22 — the LAST screen, SOLO, package v3.14.0)** charted 2026-08-26 as wayfinder map [#606](https://github.com/winniel123/verge-asm/issues/606). CHARTED, **not built** — filed map + 3 children and STOPPED (see [[wayfinders-mean-file-not-implement]]).

Children (same 3-shape as [[batch7-settings-chart-601]]): Foundation [#607](https://github.com/winniel123/verge-asm/issues/607) → Convert [#608](https://github.com/winniel123/verge-asm/issues/608) → Sign-off [#609](https://github.com/winniel123/verge-asm/issues/609), serially blocked. **Frontier = #607** (unblocked, unclaimed). All `wayfinder:task`, AFK except the sign-off gate. Source: `design-system/verify/WORK-ORDER-22-BATCH8.md`; ruled SPEC-CHANGE **#27** (a–f + scantrigger).

**THE ONE STRUCTURAL DIFFERENCE FROM SETTINGS — shell has no truly-inert intermediate.** The whole app parses into one shared `var tmpl` (`cmd/web/templates_shell.go:576`); `shell.tmpl` **redefines `head`/`chrome`/`foot`** — the exact names templates_shell.go serves live. Adding a live `ParseFS(shell.tmpl)` while the old const defines exist redefines the live shell mid-branch (Go last-parse-wins pre-execution) and breaks G1 for every landed screen. So Foundation lands files + bookkeeping + G1 manifest and only *confirms shell.tmpl parses standalone* — it does NOT wire ParseFS; the embed is atomic with deleting the 3 defines + `pageCSS` in the Convert. This is P4.4: `.DesignTokens` gate + `data-design-shell` resets + pageCSS all die; `designTokens()` then serves every page unconditionally.

Other shell specifics: **#27f one-time FULL-PAGE golden regen** for screens 1–21 rides in the SAME PR as the chrome swap (harness stops cropping to `<main>`). Conditional collision to watch: **org switcher (#27b)** — alias/BUILD `POST /org/switch` if orgs modeled, else SPEC-CHANGE-escalate + ship the static chip (org-open golden defers). v3.14.0 export already sits uncommitted in the working tree; as always it deleted SPEC-CHANGE #20–#26 → Foundation restores them from main + transcribes #27.

Landing #609 **closes the v4 console loop** (every console screen + shell design-owned). Next tree after = docs-site D-map (separate effort).
