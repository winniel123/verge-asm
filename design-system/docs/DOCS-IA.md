# Docs IA — the designed left rail (D1 ruling)

Ruled 2026-08-24 in the design workspace. This is the binding section model for docs-site's left rail. It lands as pure frontmatter edits in docs/guides/*.md — no pipeline change (nav-build/T3 already renders it). Spec preview: the `SECTIONS` rail of `design-system/examples/DocsPage.jsx` · ~~shot docs.jpg~~. `55aa367` deleted `design-system/screenshots/docs.jpg` on 2026-08-28 ([#1450](https://github.com/winniel123/verge-asm/issues/1450)). No successor spec capture carries it ([ADR-0115](../../docs/adr/0115-the-docs-site-renders-the-guides-in-place-and-a-version-is-a-git-ref-not-a-copy.md)), so `DocsPage.jsx` is the only preview of this ruling a reader can open. `docs-site/tests/baseline/docs.png` is a different artefact: it captures a built page for the `check:screenshot` pixel diff, not the designed spec.

## What this fixes

Current frontmatter groups 13 of 21 guides into one "Operating" wall, ties all five SSO guides at `order: 9` (nondeterministic rail order), and gives the provider guides titles too long for the 220px rail ("Single sign-on with Microsoft Entra ID"). The rail label comes from frontmatter `title`; the page H1 comes from the markdown body and does NOT change.

## The section model (rail order)

1. **Getting started** — Using verge-asm · First-run mental model · Zone files · Reading your attack surface
2. **Scanning** — Discovery sources · Provisioning a prober
3. **Signals & delivery** — Signals reference · Notification channels · Reports · Integrations
4. **Access** — Accounts, invites & roles · Authentication · Single sign-on (SSO) · SSO with Okta · SSO with Google · SSO with Entra ID · SSO with Keycloak
5. **Operating** — Running verge-asm · Backup & restore · Troubleshooting
6. **Contributing** — Verifying verge-asm

Section names deliberately echo the console's SettingsNav groups (Scanning, Access, Delivery) so the two surfaces speak one vocabulary.

## Frontmatter deltas (apply exactly; omitted fields unchanged)

| Guide | title | section | order |
| --- | --- | --- | --- |
| using.md | — | Getting started | 1 |
| first-run.md | — | Getting started | 2 |
| zone-files.md | — | Getting started | 3 |
| reading-the-estate.md | — | Getting started | 4 |
| sources.md | — | Scanning | 1 |
| prober.md | — | Scanning | 2 |
| signals.md | — | Signals & delivery | 1 |
| notification-channels.md | — | Signals & delivery | 2 |
| reports.md | — | Signals & delivery | 3 |
| integrations.md | — | Signals & delivery | 4 |
| accounts.md | — | Access | 1 |
| authentication.md | — | Access | 2 |
| sso.md | — | Access | 3 |
| sso-okta.md | SSO with Okta | Access | 4 |
| sso-google.md | SSO with Google | Access | 5 |
| sso-entra-id.md | SSO with Entra ID | Access | 6 |
| sso-keycloak.md | SSO with Keycloak | Access | 7 |
| running.md | — | Operating | 1 |
| backup-and-restore.md | — | Operating | 2 |
| troubleshooting.md | — | Operating | 3 |
| verifying.md | — | Contributing | 1 |

"—" = keep the current title. The four SSO provider retitles are rail labels only — each guide's body H1 ("Single sign-on with Microsoft Entra ID", …) stays as written.

## Acceptance

- Left rail renders exactly the six sections above, in this order, with these labels — ~~compare against docs.jpg~~. The capture is deleted — see the preview note above. Compare against the `SECTIONS` list in `design-system/examples/DocsPage.jsx` and the rail it renders.
- No two guides in a section share an `order`.
- Breadcrumb reads Docs › ‹section› › ‹title› (PARITY D5).
