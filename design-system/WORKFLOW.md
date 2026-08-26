# WORKFLOW v4 — exact parity by construction

Ruled by the owner 2026-08-24 (form): design authors the served view layer verbatim; hard CI pixel gate; fixtures shipped; threshold 3%; coverage 1440/1024/390 × light/dark × key open states.

## Why v3 still drifted

v3 shipped specs (JSX + screenshots) and the repo *translated* them into Go templates and repo CSS. Translation is authorship — every re-derived margin and re-invented markup node is a drift opportunity that no protocol catches until a human compares. v4 removes the translation step.

## The pact

- **Design owns the view layer.** The package ships the literal artifacts the server serves: Go `html/template` files (`templates/*.tmpl`), the compiled stylesheet (`css/vg.css` + tokens), and the view-behavior JS embedded in those templates (keyboard nav, toggles, palette). The repo embeds them via `//go:embed` and NEVER edits them. A needed change goes through SPEC-CHANGE and comes back in the next package version.
- **Repo owns everything else.** Handlers, data structs (shaped by the templates' fields), routes, migrations, derivations, auth, the fixture loader. Claude Code wires data into design-owned holes; it authors no markup, no CSS, no view JS.
- **Fixtures are the shared retina.** `fixtures/fixtures.json` carries the exact demo corpus the design mocks render (subjects, spans, signals, batches, messages, schedules, accounts, sessions) plus `clock: 2026-08-24T12:00:00Z`. The repo gains `verge seed --fixtures <file>` and an injectable clock (`s.now()` already exists), so a fixture-seeded instance renders the same relative times and rows the goldens show.
- **Two CI gates, both blocking:**
  - **G1 verbatim** — `design-system/templates/**` and `design-system/css/**` in the repo are byte-identical to the landed package (instant; catches local edits).
  - **G2 pixel** — Playwright boots the fixture-seeded server, walks `verify/states.json` (per-screen route + state script: open drawer, open menu, hover row), captures each configured viewport × theme, diffs against `goldens/` with pixelmatch. **Fail = any screen over 3.0% differing pixels.** A viewport/theme with no golden yet is skipped, not failed.

## Determinism — one canonical renderer (no cross-OS diffing)

Pixel output differs across OSes (font rasterization, antialiasing, subpixel policy), so the gate never compares captures from two environments:

- **Both sides of every diff render in the same pinned container**: the Playwright Linux image pinned by digest in `verify/config.json` (browser version pinned with it), `deviceScaleFactor: 1`, `prefers-reduced-motion: reduce`, animations disabled, fixture clock.
- **Goldens are not screenshots of the design workspace.** `verify/render-goldens` composes the design-owned templates + `fixtures.json` statically (no repo code) and captures them **inside the canonical container**; that output is `goldens/`. The repo capture runs in the identical container, so a diff can only come from the repo's rendering, never the OS.
- **Fonts ship in the package** (`css/fonts/*.woff2`) and `vg.css` declares no OS-fallback families behind them — system font substitution is a parity bug, not a fallback.
- **Local runs on Windows/macOS are advisory** (`verify --advisory` prints diffs, exits 0). The gate verdict only ever comes from the canonical container in CI. Developers never regenerate goldens locally; G1 byte-compares `goldens/` to the landed package like every other design-owned artifact.

## Coverage phases (goldens land per phase; the harness supports all from day one)

- **Phase A (now):** 1440 × light + dark × all screens + open states.
- **Phase B:** 1024 — after R1, a design-side responsive audit pass (current specs are 1440-first; most survive via fluid maxWidth, R1 verifies and fixes).
- **Phase C:** 390 mobile — after M1, a design-side mobile layout phase. Mobile layouts DO NOT EXIST yet; gating 390 before M1 would freeze accidents as goldens.

## The loop per change

1. Design edits the spec here → regenerates the affected `.tmpl`/css/goldens/fixtures → bumps VERSION.md → exports.
2. Operator lands the package folder wholesale (it supersedes).
3. Claude Code wires any new data holes (structs/handlers), runs G1+G2 locally, opens the PR.
4. CI blocks on G1/G2. A genuine impossibility = SPEC-CHANGE stop-and-escalate, unchanged from v3.

## Behavior section (required for every new-feature WORK-ORDER)

G1/G2 prove the view; they say nothing about what the feature *does*. Every WORK-ORDER that introduces a new feature (new screen, new state, new user action) MUST carry a `## Behavior` section — Claude Code implements from it instead of guessing. Wiring-only or visual-only orders may state `Behavior: no change`.

Required subsections:

- **Entry points** — how the user reaches the feature (route, nav item, action on another screen).
- **State transitions** — what moves the screen between its `states.json` fixture states (e.g. `signals · default` → `drawer-open`: row click; `drawer-open` → `default`: Esc / scrim click). Every fixture state must be reachable; every transition names its trigger.
- **Data contract** — per template hole: source datum, type, format (matching `fixtures.json`), and whether it can be empty/null. Derivations (counts, relative times, qualifiers) name their inputs.
- **Actions & side effects** — per interactive element: endpoint or handler called, method, payload, success result (what changes on screen), failure result (which error state renders).
- **Edge cases** — empty, loading, error, permission-denied, and any limits (pagination thresholds, truncation rules). Each must map to a fixture state or explicitly rule one out.

Behavior text is spec, not suggestion — a gap found while wiring is a collision (SPEC-CHANGE, AWAITING DESIGN), never an improvisation. The section binds semantics only; the repo still owns implementation shape (handler structure, storage, migrations).

## Migration chart (P4 — new Wayfinder tree)

- P4.0 **Pilot** — Inventory: repo refactors to `//go:embed design-system/templates/inventory.tmpl`, deletes its authored template + screen CSS delta; design ships the tmpl + fixtures + Phase-A goldens for that screen; harness runs it end-to-end. Proves the loop before mass conversion.
- P4.1 **Fixture loader** — `verge seed --fixtures`, injectable clock in dev mode.
- P4.2 **Harness in CI** — `verify/` (Playwright + pixelmatch + states.json + config + render-goldens + the pinned container digest), wired as required checks (G1, G2). CI runs the canonical container only.
- P4.3 **Mass conversion** — remaining screens in Landed-ledger order, one PR per screen: embed design tmpl, delete repo-authored markup/CSS for it, keep handlers.
- P4.4 **Cleanup** — delete all remaining repo-authored screen CSS deltas; `templates_shell.go` chrome converts last (it hosts every screen).
- R1 (design-side) 1024 audit → Phase B goldens. M1 (design-side) mobile layouts → Phase C goldens.

## CLAUDE.md block v2 (replace the §Design decisions block with this)

```
## Design-owned view layer
design-system/templates/**, design-system/css/**, and the JS inside those
templates are DESIGN-OWNED ARTIFACTS, landed verbatim from the design
package (VERSION.md states what landed). Never author, edit, "fix",
reformat, or restyle them — CI gate G1 byte-compares them to the package
and blocks. Data flows only through the holes they already declare.
If a hole is missing, a state is unspecced, or a template seems wrong:
STOP the work item, append the collision to design-system/SPEC-CHANGE.md
(Ruling = "AWAITING DESIGN"), print the DESIGN DECISION NEEDED banner and
the filled hand-off prompt from SPEC-CHANGE.md, and treat the item as
blocked until a new package version lands with the ruling.
Gate G2 (fixtures + Playwright + pixelmatch vs goldens/, threshold 3%)
must pass before merge, and only the canonical container's verdict counts
— local Windows/macOS runs are advisory because pixel output is OS-
specific. A diff is never resolved by editing goldens or thresholds —
only by a new design package.
```
