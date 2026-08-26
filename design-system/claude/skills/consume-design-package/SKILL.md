---
name: consume-design-package
description: Land a Verge ASM design-package version and convert or wire screens under the v4 exact-parity workflow. Use whenever a design-system/ package version lands, a WAYFINDER-MAP screen conversion is assigned, a template's data holes need wiring, or gates G1/G2 fail.
---

# Consuming the design package (v4)

You never author UI. The design package is the view layer; you embed it, feed it, and prove it. Authority: design-system/WORKFLOW.md (the pact), WAYFINDER-MAP.md (order + status), VERSION.md (what landed), CLAUDE.md §Design-owned view layer.

## When a new package version lands

1. Read `design-system/VERSION.md` — the log names what changed. Diff `templates/`, `fixtures/`, `verify/`, `goldens/` against the previous landed version.
2. If a template you already embed changed: re-run G1+G2 locally; wire any NEW holes it declares (struct fields, handler reads). Nothing else.
3. If fixtures changed: update nothing in code — the loader is data-driven; re-seed and re-run G2.
4. Never "tidy", reformat, or partially apply a package. It lands wholesale or not at all.

## Converting a screen (a WAYFINDER-MAP item)

1. Confirm the screen is the map's next unstarted stop (or in the current batch). If not, stop — order is operator-controlled.
2. Embed `design-system/templates/<screen>.tmpl` via `//go:embed` into the template set. Delete the repo-authored `templates_<screen>.go` and that screen's CSS/JS deltas entirely. If the old file defined shared partials, verify the tmpl carries them (it should — if not, that is a collision, escalate).
3. Shape the handler to the tmpl's declared holes. Field names in the tmpl are the contract; `fixtures/fixtures.json` shows every value, including formats (dates, qualifiers, summaries). Reuse existing structs where they already match; never rename a tmpl hole to fit a struct.
4. Extend the fixture loader so the seeded instance renders exactly the screen's fixture slice.
5. Add the screen to the harness run; `--write-goldens` in the canonical container if the package introduced new golden inputs; commit.
6. Run G1 + G2. Open the PR with the per-state diff report. The operator visually confirms before the next screen starts — do not begin another stop on your own.

## Implementing a new feature (WORK-ORDER with a `## Behavior` section)

1. Read the order's `## Behavior` section first — it is the functional spec: entry points, state transitions, data contract per hole, actions & side effects, edge cases. Implement from it; never infer behavior from markup alone.
2. Every transition trigger and side effect listed must work as written; every fixture state in `verify/states.json` must be reachable through the listed transitions.
3. Endpoints named in Actions & side effects are contracts — build them to the stated method/payload/result. Naming an unstated endpoint or inventing a transition is authorship: stop, escalate as SPEC-CHANGE.
4. A behavior gap (unlisted state, hole with no data-contract row, action with no stated effect) = collision → AWAITING DESIGN. Never approximate.
5. An order stating `Behavior: no change` means wiring/visual only — touch no handlers' semantics.

## Wiring rules

- A hole with no obvious source datum = collision → SPEC-CHANGE stop-and-escalate (banner + hand-off prompt). Never approximate, empty-state, or drop.
- View formatting decisions (date formats, qualifiers, pluralization) are already made — copy them from fixtures.json, don't re-derive.
- Behavior JS inside a tmpl is design-owned: if it calls an endpoint you lack, build the endpoint to its contract; don't edit the call.

## Gates

- G1: `design-system/{templates,css,fixtures,verify,goldens}` byte-identical to the landed package + committed goldens. A G1 failure means someone edited design-owned files — revert, never merge the edit.
- G2: pixelmatch per verify/config.json in the pinned container only. A G2 failure is YOUR rendering (handler data, seed, format, missing endpoint) unless the tmpl itself is wrong — which is a collision, not a fix.
- Never edit goldens, thresholds, crop, or the container pin to make a gate pass.

## Escalation

Any needed design decision → CLAUDE.md §Design-owned view layer procedure: stop the item, log in SPEC-CHANGE.md as AWAITING DESIGN, print the DESIGN DECISION NEEDED banner + filled prompt, block until a new package version lands.
