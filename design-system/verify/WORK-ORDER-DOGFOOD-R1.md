# Work order — dogfood round 1 fixes (package v3.15.0)

Use the consume-design-package skill. Source: operator dogfood notes 2026-08-26 (uploads/verge-asm-notes). Five templates changed; the rest of the package is byte-identical to v3.14.1. Land wholesale.

**Sequencing:** land WITH or AFTER batch 8 (shell). The #27f full-page golden regen covers every screen, including the five changed here — landing together needs no extra golden pass. If landed separately, regen goldens for: scope, reports, inbox, settings (instance tab), inventory (+ expanded state).

## Behavior: no change
All deltas are visual/view-JS. No new holes, no removed holes, no endpoint or transition changes.

## Template deltas

- **scope.tmpl**
  - `.sc-field form` → `.sc-field > form`: the old descendant selector also hit each seed chip's ×-button form, giving it `flex:1;min-width:80px` — the trailing dead space inside every chip (dogfood ss1).
  - `.sc-zrow` no longer wraps; `.zmeta` truncates with ellipsis so the aging badge sits right-aligned on one line (ss2).
  - Zone-file drop JS now uses `requestSubmit()` (fallback `submit()`): programmatic `submit()` skips the shell's scroll-preservation submit listener, so every zone upload landed at the top of the page (dogfood: "page jumps to top when adding zone files").
- **reports.tmpl** — barchart bars row gains `min-width:0;overflow:hidden`: with ~84 daily bars (12w range) the 3px-min bars + 4px gaps exceeded the KPI card and the baseline drew past the card edge (ss3). Data contract note added: pass ≤31 bars; aggregate to weekly beyond a month.
- **inbox.tmpl** — list empty state padding 28px → 40px, matching the detail empty state so the two empty cards render equal height (ss4).
- **settings.tmpl** — instance tab Fleet · Vantages gains an `{{else}}` empty state ("No vantage provisioned…" linking Scanning · Vantages); `.Instance.Vantages` is now nullable (ss7). No fixture state added — the seeded corpus has vantages; the branch is golden-exempt this round (skip, not fail).
- **inventory.tmpl**
  - `table-layout:fixed` + Subject column 300px + `overflow:hidden` on cells: expanding a records dropdown no longer shifts column widths (dogfood: "width of the columns shifts").
  - Records reveal is animated (grid-rows 0fr→1fr, `--dur-base` ease-out) instead of instant `hidden` toggle; inner `.rin` wrapper added; view JS toggles class `open`. Closed state renders pixel-identical to before.
  - A facet with no records renders muted `none` instead of an empty pill (ss8); `.inv-recdata` wraps with `overflow-wrap:anywhere`.

## Repo-side items (no package change — wire or fix in handlers)

1. **"Scan started" toast spam** (dns trigger): write ONE flash per dispatch, not per fan-out job, and consume the flash store on first read so the in-flight auto-refresh doesn't re-show it. Copy: `dns scan dispatched` / description `N jobs fanned out`.
2. **No toast on seed removal**: `/seeds/delete` writes no flash. Copy: `Scope removed` / description `<scope> — nothing new is admitted under it; existing subjects keep their citations.`
3. **Health tab wiring**: `.Instance.DiskDetail`, `.PgDetail`, and `.QueueDepth` arrive empty (ss7 shows the bar with no numbers and a blank queue count). The holes exist and fixtures.json shows the formats — wire them (DiskDetail like `212 GB of 500 GB · 42%`).
4. **Settings width jump on the Scans tab** (ss5 vs ss6): the repo-authored `scantrigger` define still leans on legacy pageCSS width. Batch 8 already orders its inline restyle — constraint: it must not set any width beyond `100%` of the settings column.
5. **Buttons jump the page to top**: the shell scroll-preservation JS (byte-kept in batch 8) stashes on the `submit` event and restores within 5s. Verify it ships on every console page in the current build; if a specific surface still jumps, report screen + control — a programmatic `.submit()` call in repo-authored JS is the likely cause (same class of bug as the zone drop fix above).
6. **"Scan running" pill on every page**: already fixed by batch 8 — the spec TopNav renders `.Chrome.ScanRunning` and the shell hosts every screen. No extra work; verify after landing.
