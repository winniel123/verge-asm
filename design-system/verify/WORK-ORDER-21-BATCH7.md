# Work order — batch 7: Settings, all sections (map #21, package v3.13.0 — SOLO)

Use the consume-design-package skill. Solo batch per WAYFINDER-MAP §Batching: one session, one branch, pinned to v3.13.0. Screens 1–20 are LANDED — do not touch. settings.tmpl replaces templates_settings.go **wholesale** (both template consts); the repo's `scantrigger` define is still called and stays repo-authored. `integrationsEnabled` funcmap gate kept.

## Structure (SPEC-CHANGE #26a)

The spec sidebar (grouped nav, 210px, sticky) replaces the tabs row; navigation stays `?tab=` links. Groups: Scanning (scans, vantages) · Access (sso, team, sessions, audit) · Discovery (sources, aperture) · Instance (instance) · Delivery (channels, integrations, messages, delivery). Defines: "settings" + "forbidden" + one "settings-<tab>" each; holes are documented inline at every define — reshape handler structs to them (field names are the contract; fixtures.settings shows every value).

## Per-section rulings

- **scans (#26b)**: the tab is the queue read — in-flight fan-out (progress meter + jobs table with state dots/badges), recent dispatches, and the relocated cold tier (#21d; POST /settings/cold kept). The mock's "Recurring intent" card (profile/cadence/timeout/cap) and the schedule Calendar are **DEFERRED** — no operator-editable scan-config store exists; fabricate nothing (ADR-0110). They remain in the spec as the target for a future config store; do not build placeholders.
- **vantages (#26c)**: spec VantageCards replace the table (dashed border + no-claims note for unverified). Each prober renders its real lifecycle: awaiting key → install key (inverted code block + hover copy) → pinned (fingerprint chip) + platform + egress callout (links /scope — no declare-egress POST exists). Provision form kept at POST /settings/probers (host/port/username, port defaults 22). **#26d** the preferred-vantage Combobox is DEFERRED (no default-vantage store).
- **sso (#26e)**: OIDC stands (ADR-0113); the mock's SAML metadata drop, claim MappingEditor, SP certificates and require-SSO switch predate it and RETIRE. Provider table with edit disclosure (update/secret/delete routes kept), add form, linked-identities table (remove kept).
- **team (#26i)**: spec table (initial-avatars derived from username — no new datum), kebab menu → the PRG dialogs (?role= ?reenroll= ?remove=), spec dialog panels on scrim links. Remove keeps typed-confirm (JS gates the danger button via #st-typed-name; server still validates confirm_name). Invite keeps the link flow: ?invite=1 → role pick → POST /settings/accounts → shown-once join link (warn callout + copy). Routes kept.
- **sessions**: spec dense table; revoke = icon-link to ?revoke=, revoke-all in the kebab menu to ?revoke-account=; both PRG dialogs spec-styled, revoke-all keeps typed-confirm. Routes kept.
- **audit**: `.AuditRows` nullable; the honest empty (linking delivery record + messages) is the default render.
- **sources**: three tier cards (unencumbered / operator-accepted / barred) with switch-forms POSTing /settings/sources (id + enable; alias or keep the existing enablement route). Enabling an operator-accepted source routes through the PRG consent dialog (?consent={id}): terms list + checkbox-gated "Accept and enable" (JS; the POST carries accept_terms=true). Admin-only callout; viewer renders inert switches.
- **aperture**: sensitive tier as locked chips (+ "not editable, on purpose" callout); frequency tier table (also-sensitive, edit provenance, remove/reset) + add form at POST /verge-core/frequency. Holes kept from the repo shape; `.Sensitive[]` gains `.Service` (display label).
- **instance**: update callout (nullable), three stat cards (Version/Uptime/Queue depth), storage & database (disk meter + postgres line), fleet list. New holes under `.Instance` — build the reads (version/uptime/queue/disk/pg are host facts; nothing fabricated, omit what has no read and the region collapses).
- **channels (#26f)**: class checkboxes and badges render from `.ClassOptions` / per-channel `.ClassStates` / `.Classes` — the store's vocabulary, never hardcoded. Edit disclosure keeps update/delete routes; declare form keeps POST /settings/channels; secret placeholder copy kept.
- **integrations (#26j)**: the spec catalogue maps PRG — ?cat= &q= filter (segmented links + GET search), ?view={id} opens the spec drawer (grants list with write badges, attention callout, installed/last-delivery/classes KV); install/remove/test POST /settings/integrations/* (alias or BUILD). Gated by integrationsEnabled everywhere (nav item, dispatch).
- **messages (#26g)**: the operational store list (cause · class tag · instant · headline · census rows · delivery receipts with the undelivered treatment · jump link · mark-read), spec-styled; /inbox stays the reading surface. Mark-read/read-all keep `return=/settings?tab=messages`.
- **delivery (#26h)**: outcomes table (channel/class/outcome/when) — the mock's error/skeleton/retry states are client-runtime affordances and DROP; the server renders data or the honest empty. Retention dials kept (POST /settings/retention, admin-gated, error re-render).

## Fixtures (fixtures.json → settings.*)

One slice per section: running standard scan (6 jobs, one retrying, 67%) + 3 dispatches + cold scopes (tier on via 203.0.113.0/24); 3 vantage cards + 1 fully-pinned prober (key, fingerprint, platform, egress 203.0.113.5); Okta OIDC provider + 2 bindings; 4 members (ola self) + invite-link fixture; 4 sessions; audit null; 4+4+2 sources with RIPE-family terms; aperture counts 6/8/16 (8443 added); instance health incl. v0.9.3 update; 2 channels with class_states; 8 integration tiles + the PagerDuty attention drawer; 2 store messages (one with a failed delivery); 3 delivery outcomes + zeroed retention dials (updated 2026-08-20).

## Goldens (crop `main`; dialog/drawer states crop `body` · × light/dark @1440)

scans · vantages · sso · team · team-invite (?invite=1) · team-remove (?remove=u3) · sessions · sessions-revoke-all (?revoke-account=u2) · audit · sources · sources-consent (?consent=ripestat) · aperture · instance · channels · integrations · integrations-drawer (?view=pagerduty) · messages · delivery · forbidden (viewer session on an admin act).

## Acceptance

Byte-served from settings.tmpl; no repo-authored markup/CSS remains for /settings (both old template consts deleted); the sidebar navigates all sections with integrations gated; every dialog opens as URL state and its typed/checkbox gates work; sources consent records acceptance; prober lifecycle renders from real reads; deferred regions (#26b/#26d) are absent, not stubbed; G1+G2 green across all 19 states; SPEC-CHANGE gains no silent workarounds.
