# Work order — screen 3: Profile (map #3, package v3.6.0)

Use the consume-design-package skill. Inventory (#1) and ErrorPage (#2) are LANDED — do not touch them.

## Steps

1. Embed `templates/profile.tmpl`; delete `templates_profile.go`. Hole names are unchanged — the PRG interaction model (URL-driven dialogs, form POSTs) is kept, so handlers mostly stand.
2. **Reconciliations (SPEC-CHANGE #18, ruled by design 2026-08-25):**
   - Token revoke is a **plain** ConfirmDialog now (message + detail + danger confirm). Drop the typed-name `confirm_name` requirement from the revoke handler; the typed gate stays reserved for the worst acts (seed descope).
   - Notices/errors render as Callouts (the tmpl does this); **act results** (password changed, session revoked, token revoked, signed out others) ride the shell toast pipeline (P1.7) with the spec's toast copy — retire inline `.Notice` for those paths (hole stays for anything left).
3. **Fixtures:** seed `fixtures.json → profile` — 3 sessions (current = Firefox · macOS), one Okta identity, Google as a linkable provider, tokens t1/t2. Session `LastActive` renders relative ("now" / "2h" / "3d") from the pinned clock; `CreatedISO` / `LinkedAt` / token dates are date-only UTC.
4. **Goldens** (crop `main`, admin session): default · new-token dialog (`?new=1`) · minted (state script submits the form with name "ci-golden") · revoke-token dialog (`?revoke=t1`) · end-session (`?endsession=1`) · sign-out-others (`?signoutothers=1`) — × light/dark at 1440. Minted golden: the handler must render the fixture-deterministic token value `vg_pat_cigolden0example` in fixtures mode so the pixel diff is stable.
5. G1 + G2; PR with diff report.

## Acceptance

- `/profile` byte-served from profile.tmpl; no repo-authored markup/CSS for it remains.
- Cards, badges (pill), spec dialogs (centered, radius 24, scrim fade + pop), hover-reveal copy control on the minted token, spec table styling.
- Revoke flows per #18; toasts fire with spec copy; TOTP off-state renders when the fixture viewer account (no TOTP) signs in.
