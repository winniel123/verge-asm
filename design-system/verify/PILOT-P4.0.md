# P4.0 — Inventory pilot work order (Wayfinder: single item, run solo)

Package v3.4.0 ships the design-owned artifacts. Prove the WORKFLOW.md loop on one screen end-to-end. STOP + SPEC-CHANGE on any impossibility; never edit templates/, fixtures/, verify/ configs, or goldens.

## Steps

1. **Embed.** Add `//go:embed design-system/templates/*.tmpl` (or per-file) and parse `inventory.tmpl` into the template set. Delete `templates_inventory.go` entirely — the tmpl carries `inventory`, the shared `recordrows`, and `subject-missing`. Handler `inventoryPage` stays; it already produces the exact hole shape (`Groups/Subjects/Facets/Details`, `HasData`).
2. **Since format.** The design renders date-only UTC (`2026-07-14`). Format `inventoryFacet.Since` as `2006-01-02` for this screen (view contract; the drill-downs keep their own formats).
3. **Fixture loader (P4.1, pilot-scoped).** `verge seed --fixtures design-system/fixtures/fixtures.json`: reset dev DB, synthesize open spans such that `buildInventory` renders EXACTLY `fixtures.json → inventory.groups` (keys, facet labels incl. vantage qualifiers, summaries, record details, gaps, since dates), pin the injectable clock to `clock`. The JSON is view-shaped on purpose — the loader owns the span synthesis.
4. **Harness (P4.2, pilot-scoped).** `verify/` gains: `render-goldens` (a small Go cmd that parses the design tmpl with stub `head`/`chrome`/`foot` — stub head links the landed `design-system/tokens/*.css` exactly as the app's head does — executes it with fixtures.json, writes static HTML) and `capture` (Playwright in the pinned container: serve the static HTML for goldens / the fixture-seeded server for candidates, apply `states.json` scripts, screenshot per viewport×theme cropped to `main`, pixelmatch per `config.json`). First run: `--write-goldens` materializes `goldens/inventory/` in-container; commit them; they are design-owned from then on (G1).
5. **CI.** Two required checks: G1 (byte-compare `design-system/{templates,css,fixtures,verify,goldens}` vs the landed package + committed goldens), G2 (capture + diff ≤ 3.0%/screen-state). Local runs outside the container: `--advisory`.
6. **PR** with the G2 report (per-state diff %). Then the operator does the visual confirm (WAYFINDER-MAP step 3) before any other screen starts.

## Acceptance

- `/inventory` on the fixture-seeded instance is byte-served from the design tmpl; no repo-authored markup/CSS/JS remains for this screen.
- All five states × light/dark pass G2 at 1440.
- Kind scope, gaps-only, filter, density, columns, expansion, row nav (click + j/k/Enter), Export CSV link, Add seed link all work as the tmpl wires them.
- `verge seed --fixtures` is idempotent and dev-only (refuse outside dev mode).
- SPEC-CHANGE.md gains no silent workarounds — anything impossible was escalated.
