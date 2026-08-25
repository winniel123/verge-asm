# Work order — screen 2: ErrorPage (map #2, package v3.5.0)

Use the consume-design-package skill. Inventory (#1) is LANDED — do not touch it.

## Steps

1. Embed `templates/error.tmpl`; delete `templates_error.go`'s template const (keep errors.go handlers). Hole names are unchanged (`Kind/Code/Subject/IncidentID/ActionLabel/ActionHref/.Chrome`), so handlers mostly stand.
2. **Chrome gate**: the spec renders every kind inside the console chrome when a session exists. Wire `notFound` / `forbidden` / the 500 recovery to inject chrome data (Account/IsAdmin → `.Chrome`) when the request carries a valid session; signed-out stays bare. (Cropped goldens are unaffected — this lands the spec rule for #22.)
3. **Deterministic 500**: in fixtures mode, `newIncidentID()` returns `fixtures.json → error.incident_id` so the golden is stable. Real mode keeps crypto/rand.
4. **Dev state routes** (fixtures mode ONLY, refuse otherwise): `/dev/403` → forbidden(), `/dev/panic` → panic("fixture"). 404 needs none (any unknown route); missing-subject/run/settings-forbidden use real routes + fixture data.
5. Seed `fixtures.json → accounts` (admin + viewer, dev passwords) and a dev-only session mint for the harness (`session` field in states.json).
6. Goldens for the six states × light/dark at 1440 (crop `main`); G1 + G2; PR with diff report.

## Acceptance

- Six kinds render byte-served from error.tmpl; copy verbatim (Settings → Team arrows are real →, not entities); icon stroke 1.75; body max-width 440 all kinds; incident copy control appears on hover and swaps to the check for 1.4s.
- 404/403/500 signed-in show chrome; signed-out bare. Contextual kinds keep their ActionHref destinations.
