# Work order — API token surfaces (#390) + Backup & updates (#391) — package v3.18.0

Use the consume-design-package skill. Templates changed: profile.tmpl, settings.tmpl. fixtures.json + states.json extended. Retires Wayfinder tickets #390/#391 into the design workflow. Land wholesale.

## #390 — read-only API token surfaces

### Behavior
- **Entry points**: Profile · Personal API tokens (existing section); Settings · Access · API access (new tab, ?tab=api).
- **State transitions**: api tab default = disabled; admin POST /settings/api {enabled} → PRG back to ?tab=api (badge, callout, and By/At record flip). Profile inert-note renders while disabled and disappears when enabled.
- **Data contract**:
  - `.API{Enabled,By,At}` (settings) — By/At are the dated act of the CURRENT state (who last flipped it, when); null when never enabled.
  - profile `.Tokens[].Last` becomes nullable — a real last-used timestamp (relative, fixtures format "2h"/"14d"); null renders "never". `.APIEnabled` joins the profile holes.
  - last_used_at: write on each authenticated /api/v1 request, coarsened (upsert at most once per hour per token) — the display is honest without a hot write path.
- **Actions & side effects**:
  - Enable: /api/v1 starts answering GET for valid tokens; flash `API access enabled` / `Personal tokens now answer GET /api/v1/… — read-only, always.`
  - Disable: /api/v1 returns 404 for every path (indistinguishable from absent); tokens inert; flash `API access disabled`.
  - The bearer path NEVER creates a session, reads cookies, or accepts a token on any non-/api/v1 route. No mutating verb exists under /api/v1 — 405 with no body detail.
- **Edge cases**: non-admin POST → 403; viewer sees the read-only view (state + note, no button); token of a removed account → 401; disabled + valid token → 404 (surface off beats auth); Last-used never regresses on restore (it is data, in the backup).
- **Fixtures/states**: fixtures profile.tokens gains never-used `ci-export`; settings.api disabled. States: settings·api (admin), settings·api-viewer, settings·api-enabled (JS submits the toggle — PRG renders the enabled state live).

## #391 — Backup & updates (Settings · Instance)

### Behavior
- **Entry points**: Instance tab — three new cards (Backup, Restore, Version & updates) after Fleet. The old `.Instance.Update` callout is RETIRED — its content moves into the Release card.
- **Data contract**: `.Instance` gains `Backup{InProgress,Streamed,SizeHint,Percent,LastAt,LastSize}` (nullable), `RestoreError`, `Preflight{File,TakenAt,Subjects,Schema}` (nullable), `RestoreConfirm{File,TakenAt,Subjects}` (nullable — ?restore-confirm=1), `Migrations{Pending}`, `Release{State(current|newer|disabled),CheckEnabled,CheckedAt,Latest{Version,Notes},Steps[]}`.
- **Backup**: POST /settings/backup streams the archive as an attachment — DATA ONLY: estate + config; NEVER session key or prober key (they regenerate on restore, old sessions lapse — the card says so as a feature). `.Backup.LastAt/LastSize` record the last UI-taken backup. InProgress renders only if the repo implements async sealing; a plain streamed response may never show it.
- **Restore**: POST /settings/restore/preflight (multipart, admin) validates the archive WITHOUT applying → PRG with `.Preflight` (file, taken-at, subjects, schema). "Continue to restore…" → ?restore-confirm=1 dialog — typed-confirm `restore` (JS-gated via the new generic [data-typed-confirm] wiring in the settings view script; server validates the word too). POST /settings/restore applies: overwrite data, regenerate session + prober keys, every session lapses → sign-in with notice. Unparseable/mismatched archive → `.RestoreError`, nothing touched.
- **Version & updates**: the app NEVER self-replaces. Worker checks the release feed daily when `CheckEnabled` (best-effort, short timeout, no retry storm; failure reports nothing). States: current (ok dot + checked-at) / newer (accent callout + Latest + the LITERAL guided host steps in `.Steps[]`) / disabled (air-gap copy, no network call ever). Toggle: POST /settings/updates/check {enabled}, admin. `Migrations.Pending` from the migration table vs binary (badge: "schema current" / "N migrations pending" warn).
- **Edge cases**: restore during an in-flight scan → refuse preflight with `.RestoreError` ("stop the dispatch first"); backup during restore → 409; update check while disabled → never dispatched (not even on boot); Steps[] are release-authored literals — the UI never composes shell commands.
- **Fixtures/states**: instance fixture carries release.state=newer (richest golden), migrations.pending=0, backup.last_at=null. Preflight/restore-confirm states are golden-exempt this round (need a multipart upload the harness doesn't drive) — the dialog is exercised by the [data-typed-confirm] unit of the view script; capture when a dev state route exists.

## Gates
G1 wholesale. G2: profile default re-captures (never-used row + inert note); settings nav re-captures on every settings state (new Access item); new states api / api-viewer / api-enabled; instance re-captures (three new cards, Update callout gone).
