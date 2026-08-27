# Work order — batch 1: SignIn + Setup + Coverage (map #4–#6, package v3.7.0)

Use the consume-design-package skill. First parallel batch: up to 3 sessions, one screen each, one branch each, all pinned to v3.7.0, shared files append-only, PRs merge serially with G1+G2 re-runs (WAYFINDER-MAP §Batching). Screens 1–3 are LANDED — do not touch.

## Screen 4 — SignIn family (`templates/signin.tmpl` replaces templates_signin.go)

Defines and hole names are kept (login/totp/totp-enroll/totp-recovery/forgot/forgot-sent/reset/reset-invalid/reset-done/invite/invite-invalid + authbrand/authfoot). New holes to wire:
- `.Version` (authfoot) — the real build version the shell already reads.
- `.Username` on the totp step — the mid-login account, for the sub-line.
- login `.SSOProviders[]` gains `.Mark` — a 1–2 letter mono mark derived from the provider name (first letters, upper), e.g. Okta → "O".
Reconciliations (SPEC-CHANGE #19, ruled): password policy hint AND enforcement unify at 12+ (reset currently says 8+); the "locked out?" CLI line lives on the forgot card only (remove from login); enroll drops the otpauth-URI text row (the QR carries it; secret keeps hover-copy); recovery page gains the stored-checkbox gate before Finish; the TOTP code field is the segmented 6-box control (JS in tmpl; posts the same `code` field, auto-submits when complete).

## Screen 5 — Setup (`templates/setup.tmpl` replaces templates_setup.go)

Holes unchanged (.Error .Token). Reuses signin.tmpl's authcss/authbrand/authfoot — land both templates together (they parse into one set; signin.tmpl must be embedded for setup to resolve).

## Screen 6 — Coverage (`templates/coverage.tmpl` replaces templates_coverage.go)

Holes: Meters gain nullable `.Total` + precomputed `.Pct` (0–100 int) and messages gain `.When`/`.ISO` + `.Bound`. Reconciliation (#19c): an ADDRESS scope's meter renders counted/total — denominator = the enumerable addresses of the declared range (a /24's usable size), counted = subjects the batch walked; NAME scopes stay census. Record as an ADR note refining ADR-0095 (estate-proportion stays forbidden; a declared range is not the estate). Messages render the relative time with the ISO instant as the title tooltip. Stale-zone callout is per-zone (`.StaleZones[{Zone,Age}]`, e.g. "2 re-supply intervals").

## Fixtures (fixtures.json → signin / setup / coverage)

- signin: sso_providers [{slug okta, name Okta, mark O}]; version "v0.9.2"; deterministic reset token `fixture-reset-token` (valid) for `/reset`, invite token `fixture-invite-token` (viewer); fixtures mode accepts TOTP code `482913` and pins enrol secret `VG7K-2Q9X-8MRD-P3TL` + recovery codes list (goldens must be stable).
- setup: seed variant `empty` (no accounts; setup token pinned `fixture-setup-token`). The harness runs setup states against that variant.
- coverage: two meters (203.0.113.0/24 → 198/214 subjects with the skip detail; acmecorp.io name scope → census 62 addresses), four messages (gap/stale·9d/silent/not-evaluable with when+iso), three gaps, two unevaluable rules, one stale zone (internal.acmecorp.io · "2 re-supply intervals").

## Goldens (crop: signin/setup full-viewport `body` — they are chrome-less; coverage `main`)

signin: login · login-sso-none (variant without provider) · totp (post fixture creds) · forgot · forgot-sent · reset · reset-invalid · reset-done · invite · invite-invalid · enroll · recovery — × light/dark @1440. setup: default · error (bad token post). coverage: default · empty (variant `empty`).

## Acceptance

Byte-served from the three tmpls; no repo-authored markup/CSS remains for these routes; segmented code input works (type, paste, backspace, auto-submit); recovery Finish gates on the checkbox; 12+ policy enforced; G1+G2 green; SPEC-CHANGE gains no silent workarounds.
