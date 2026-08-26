# Work order — batch 8: Shell/chrome (map #22, package v3.14.0 — SOLO, LAST)

Use the consume-design-package skill. Solo batch per WAYFINDER-MAP §Batching: one session, one branch, pinned to v3.14.0. Screens 1–21 are LANDED. shell.tmpl replaces **templates_shell.go's "head" + "chrome" + "foot" defines AND pageCSS wholesale** — this is P4.4 exactly as templates_shell.go's own comments promise: the shell glue, the .DesignTokens gate, the data-design-shell resets, and every legacy page class die with the conversion. Funcmap keeps `integrationsEnabled`, `designTokens`, `signDelta`.

## Rulings (SPEC-CHANGE #27)

- **#27a head** — contract kept: `.Title`, `.Refresh`, and the `localStorage("verge-theme") → data-theme` boot script, byte-equivalent. pageCSS dies; the head inlines the font @import + `designTokens()` (the design token set now serves EVERY page, unconditionally) + the design shell CSS. If tokens/base.css already carries the font @import, drop the head's duplicate — SPEC-CHANGE note either way.
- **#27b chrome** — the spec TopNav renders from the new `.Chrome` struct (holes in the tmpl header; `.Chrome` nullable — auth surfaces pass none and get no chrome, as today). Nav pills + counts from `.Chrome.Nav` (build the slice the current hardcoded links encode; Signals carries the open count). Org: `.Orgs` nullable — a single-org deployment renders the static chip. **The switcher needs an org store: alias or BUILD POST /org/switch; if orgs are not modeled, SPEC-CHANGE escalate and ship the chip** (the org-open golden then defers with it). Bell popover renders the top 4 messages + All messages → /inbox. Account menu: Profile · Settings · palette item · **the existing sign-out form action** (the tmpl says /signout — correct it to the real route at landing if it differs; that is a mechanical alias, not a redesign). Scan status renders when `.ScanRunning`.
- **#27c palette** — the #315 scaffold contract is kept under the spec panel: ids `#cmdk` / `.cmdk-input` / `.cmdk-item` / `.cmdk-group`, the `[data-cmdk-search]` item stays visible through any filter and its href tracks the typed query to `/search?q=…`. Groups are server-rendered from `.Chrome.PaletteGroups` — items are links (Run scan → /scans, Add seed → /scope; the theme item is the one JS action). The Integrations item renders only `{{if integrationsEnabled}}` — thread the gate when building the groups slice, or emit the item and let the handler omit it; either way the gated build must not show it while hidden.
- **#27d toasts** — the spec ToastStack renders `.Chrome.Toasts` from the PRG flash store ("screen tickets fill the toast stack"): tone dot, title/description, dismiss ×, 5s auto-dismiss with the leave animation. Wire the existing flash mechanism to the holes; nothing new is invented.
- **#27e foot** — the spec console footer (AGPL note, Docs/GitHub, version) gated by `.Chrome`; the **scroll-preservation JS is kept byte-for-byte** (behavior contract). The palette/theme/popover/toast JS is design-owned in the tmpl.
- **scantrigger** — the repo-authored define loses pageCSS: restyle its few controls inline within the token vocabulary (repo-owned glue; no design component is authored).

## Fixtures (fixtures.json → shell)

Nav (9 items, Signals ·47, dashboard active), 3 orgs (acmecorp active · 1,284), version v0.9.2, user ola@acmecorp.io (OL), 4 bell messages (2 unread — store classes), palette groups (17 screens incl. gated Integrations + the search-handoff item; 3 actions incl. the theme toggle), toasts variant (ok · "Scan complete · 3 new signals raised."), scanning variant pins `.ScanRunning`.

## Goldens — shell states + the one-time regen (#27f)

Shell states (FULL-PAGE, × light/dark @1440, on the dashboard fixture): default · palette-open (click #sh-cmdk-btn) · bell-open · org-open · acct-open (set the details open) · scan-running (variant) · toasts (variant, delay 400ms — capture before auto-dismiss).

**Golden regen**: with the chrome converted the harness stops cropping to `<main>`. ALL landed screens (1–21) re-render FULL-PAGE goldens in the canonical container at this version — the one-time regen the map §Shell note reserves. Do it in the same PR as the shell landing (the crop change and the chrome change are one atomic diff); every screen's G2 then runs against the new full-page goldens.

## Acceptance

Byte-served from shell.tmpl; templates_shell.go's three defines + pageCSS deleted; every landed screen renders inside the design chrome with no legacy class in the DOM; ⌘K opens/filters/navigates and hands off to /search?q=; bell/org/account popovers open as details and close on outside click + Escape; theme toggles and persists; toasts render from the flash store and auto-dismiss; the footer sticks below short pages; scroll restores across PRG; G1+G2 green across the 7 shell states AND the regenerated full-page goldens for screens 1–21; SPEC-CHANGE gains no silent workarounds.

## After this batch

The map is complete: every console screen and the shell are design-owned. The docs site gets its own D-map items next (per the map's closing note); the v4 loop for the console closes with this landing.
