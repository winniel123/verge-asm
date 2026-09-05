# ADR-0173: the API-access tab is the one Settings surface a `viewer` reads, and it carries state without a control

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1334 ADR gaps: cmd/web/auth.go](https://github.com/winniel123/verge-asm/issues/1334), gap 1
- **Sweep PR that deleted the comment:** [#1337](https://github.com/winniel123/verge-asm/pull/1337) — which compressed the block to one uncited line rather than deleting it (`cmd/web/auth.go:104`, blamed to `267c196`)
- **Rests on:** [ADR-0123](./0123-a-token-api-is-read-only-opt-in-and-a-bearer-path-separate-from-sessions.md), which supplies what the tab is about — a read-only `/api/v1`, off by default, behind one instance-wide `api_enabled` flag carrying who and when, with the account's role read live per request
- **Not bound by:** [ADR-0158](./0158-a-read-only-console-screen-may-scope-its-rendered-rows-in-the-client-and-a-screen-that-submits-a-form-carries-its-scope-in-the-query-string.md), which rules how a read-only screen holds filter and sort state over rows the server already rendered. It says nothing about who may reach a screen, and `?tab=` is a destination, not a view scope
- **Withdraws the over-broad statement at its specifying site** ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) discipline): `docs/guides/accounts.md:52-54`, which reads *"The whole **Settings** screen is itself admin-only (`GET /settings` is behind `requireAdmin`)"*

## Context

`GET /settings` is the only route that renders a Settings tab, and the only route in the `/settings`
family that is not `requireAdmin`. `cmd/web/handlers.go:450` mounts it behind
`requireSettingsAdmin`. The other thirty-one are thirty `POST /settings/…` writes, every one
`requireAdmin` (`cmd/web/handlers.go:362-363`, `:431`, `:451-481`), plus `GET /settings/vantages`
(`:366`), which is `requireLogin` over `redirectTo` (`:308-320`), renders nothing, and 303s back to
`/settings?tab=vantages`, where the gate runs again.

`requireSettingsAdmin` (`cmd/web/auth.go:102-110`) refuses a non-admin unless the request names the
`api` tab:

```go
if acct.Role != roleAdmin && validTab(r.URL.Query().Get("tab")) != "api" {
	s.settingsForbidden(w, r, acct)
	return
}
```

The refusal renders the `settings-forbidden` page at 403 (`cmd/web/errors.go:66-77`) — the richer
refusal Settings alone uses, where every other admin route keeps the plain 403.

The rule was written down once, on the line above that test, and no document states it. ADR-0123
rules the API and never reaches the console's role matrix: its Context mentions *"a Settings·Access
enable toggle"* as an operator affordance, and its §4 states only that the **role is read live per
request** — a rule about staleness, not about reach. `docs/guides/accounts.md:52-54` states the
matrix and states it **wrong**, in the direction that would forbid what the code does.

**The carve-out is not decorative, and the code says why.** The viewer-readable Profile page renders,
when and only when the API is off, a link into this tab: `design-system/templates/profile.tmpl:206`
— *"every token is inert until an admin enables it under [Settings · API access](/settings?tab=api)"*
— driven by `APIEnabled` on the Profile view model (`cmd/web/auth.go:1352-1357`, `:1385`). Any
account may mint its own token (`cmd/web/handlers.go:435`), so a viewer holding an inert token is the
ordinary case, and without the carve-out the product hands that viewer a link that 403s.

## Decision

> **Settings is admin-gated as a whole, and the `api` tab is the single carve-out: a `viewer` may
> READ it, and sees the read-only state, its provenance and a note, and no control. Every other tab
> refuses a viewer with the settings-forbidden page. The gate is on the normalized tab identifier,
> never on the rendered widget, and a tab that grows a control loses the carve-out.**

### 1. The gate is on the tab identifier, and it is the identifier the renderer resolves

`settingsTabs` (`cmd/web/settings.go:168-174`) is the closed set of fourteen identifiers: `scans`,
`vantages`, `sso`, `team`, `audit`, `api`, `sessions`, `sources`, `aperture`, `instance`, `channels`,
`integrations`, `messages`, `delivery`. `validTab` (`:176-187`) folds `access` to `team` for a
bookmarked pre-V3 link and folds everything else it does not know — including an absent `?tab=` — to
`scans`.

The gate and the renderer call **the same function on the same query key**: `cmd/web/auth.go:105`
and `cmd/web/settings.go:279` both read `validTab(q.Get("tab"))`, and the `devMode` fixture path does
the same at `cmd/web/settings_fixtures.go:410`. The tab the gate admits is exactly the tab
`renderSettings` fills. There is no looser test — no prefix match, no `Contains`, no raw query value
— so no spelling of `?tab=` clears the gate and renders a different section. `?tab=access`
normalizes to `team` and is refused; a repeated `?tab=api&tab=team` resolves to `api` at both sites,
because both use `Get`.

That is the whole safety of the carve-out: `renderSettings` fills exactly one section
(`cmd/web/settings.go:618-647`), so admitting one identifier admits one section's data.

### 2. What a viewer sees, and what the tab may therefore carry

`fillAPISection` (`cmd/web/settings.go:750-773`) puts three things on the page: `Enabled`, and — only
when enabled — `By`, the username that flipped it, and `At`, the UTC minute. The template renders the
badge, then branches: `design-system/templates/settings.tmpl:1223-1225` puts the toggle form under
`{{if $.IsAdmin}}` and gives a viewer *"You have read access. Enabling or disabling the API is
admin-only."* instead. `IsAdmin` is set once, from the live account row, at `cmd/web/settings.go:610`.

**No token value is on this tab.** Tokens are minted, shown once and revoked on Profile
(`cmd/web/handlers.go:435-436`). The only writer of the flag is `apiToggle`
(`cmd/web/settings.go:1189-1205`), mounted `requireAdmin` at `cmd/web/handlers.go:467`.

The bound is therefore: **the carved-out tab may carry the state, the provenance of that state, and
prose. It may carry no operator dial and no secret.** Provenance is admitted because it is a fact
about the state itself, and its cost is named: a viewer learns one admin's username, which is
otherwise Team-tab data. That price is paid once and does not recur.

### 3. Reading that fact grants nothing

`api_enabled` is one boolean about whether a surface answers. A viewer who reads it can act on
nothing: enabling is admin-only by route, the API is read-only always under ADR-0123 §1, and a
viewer's token reads exactly what a viewer's session reads. Concealing the flag would withhold no
capability. It would only make an inert token indistinguishable from a broken one.

### 4. The carve-out does not grow, and the failure mode is named

- **One tab.** A second requires an ADR that answers §3 for that tab's data.
- **The gate stays on the tab, never on the widget.** Moving the decision into the template — render
  the tab, hide the controls — makes reach a property of markup, and every control later added to
  that section is admitted by default until someone remembers the `{{if $.IsAdmin}}`. The template's
  `IsAdmin` branch is defence in depth over a section with nothing to hide, not the gate.
- **A tab that grows a control loses the carve-out.** If `api` gains a viewer-reachable operator dial
  or a secret, the carve-out is withdrawn from `requireSettingsAdmin` in the same change, and the
  Profile link at `profile.tmpl:206` goes with it.

### 5. Every other tab refuses, and there is no second route

Thirteen identifiers refuse a viewer at `cmd/web/auth.go:105-107`. Three cases are pinned:
`TestSettingsIsAdminOnly` (`cmd/web/settings_test.go:53-60`) pins the no-`?tab=` default,
`cmd/web/settings_sessions_test.go:154` pins `?tab=sessions`, and `TestViewerCannotToggleAPI`
(`cmd/web/settings_api_test.go:61-84`) pins both halves of the carve-out — the viewer's `POST
/settings/api` is 403, and the viewer's `GET /settings?tab=api` is 200 and contains no
`action="/settings/api"`. **The remaining eleven are unpinned**, which is a gap this ADR leaves to a
test rather than to code.

## Consequences

- **`docs/guides/accounts.md:52-54` is wrong twice and is corrected in place**: the gate is
  `requireSettingsAdmin`, and the screen is not admin-only as a whole. Its route table (`:180-186`)
  gains a `/settings?tab=api | GET | viewer` row. The claim the sentence supports — *"a viewer cannot
  even open the Team tab"* — is true and stays.
- **`docs/guides/backup-and-restore.md:305`** carries the same over-broad parenthetical while ruling
  the delivery tab. It narrows to that tab.
- **A viewer on the `api` tab sees a full Settings nav whose other thirteen links 403**
  (`design-system/templates/settings.tmpl:231-256`). The cost is real and small; role-filtering the
  nav would state the matrix a second time, where it can drift from the gate.
- **The line at `cmd/web/auth.go:104` gains a citation** and stays. The declaration position on
  `requireSettingsAdmin` stays empty.
- **A test is owed**, iterating `settingsTabs` and asserting 403 for a viewer on every identifier but
  `api`. Written that way it fails the day a fifteenth tab arrives and nobody decides its role.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Admin-only with no carve-out** — delete the `!= "api"` limb, collapsing `requireSettingsAdmin` into `requireAdmin` | It breaks a link the product renders to viewers itself (`profile.tmpl:206`), and any account can reach the state that renders it by minting a token (`cmd/web/handlers.go:435`). A viewer would get the settings-forbidden page for asking whether their own credential is live, then ask an admin, who reads back one boolean that grants nothing (§3). It also makes the Profile card's warning unverifiable by the only person it addresses |
| **A viewer-visible tab with a disabled control** | It moves reach from the route into the markup: the gate then admits the tab unconditionally and relies on every present and future control in `settings-api` remembering `disabled` — a property no test states once and no reviewer checks by reading the gate. `disabled` is client-side besides, so the control's `POST /settings/api` is still refused at `cmd/web/handlers.go:467`, and its whole contribution is a button that lies about being reachable |
| **A separate viewer-facing page outside Settings**, the pattern `/coverage`, `/scans`, `/sources` and `/verge-core` already follow — `cmd/web/handlers.go:412` states its reason: *"Folding a viewer-readable read into admin Settings would downgrade a viewer's access (#281)"* | It buys a nav destination and a second render path for one boolean. That pattern earns its cost where a screenful of estate data is viewer-readable; here the payload is `Enabled`, `By` and `At`. It also splits the fact from its operator context, and the four guide references to **Settings → API access** (`docs/guides/api.md:36`, `:106`, `:252`, `:260`) would each have to name whichever page matches the reader's role. The half of this worth having is already built: `/profile` carries `APIEnabled` (`cmd/web/auth.go:1385`) as a warning on the token card, not as a second home for the setting |
| **Role-filter the settings nav** so a viewer sees only the tab they may open | It states the role matrix a second time, in a template, with nothing keeping it equal to `cmd/web/auth.go:105`. The two drift silently in the safe direction and as a 403 in the other. The refusal page is where a wrong click is answered |
| **Gate on the raw `Get("tab")` rather than on `validTab`'s output** | The gate would admit strings the renderer folds elsewhere and refuse strings it folds to `api`. Normalizing at both sites is what makes "the tab the gate admitted" and "the section the page filled" one object |
