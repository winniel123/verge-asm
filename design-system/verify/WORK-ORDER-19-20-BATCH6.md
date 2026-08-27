# Work order — batch 6: SearchResults + Onboarding + FirstRun (map #19–#20, package v3.12.0)

Use the consume-design-package skill. Sixth parallel batch: up to 2 sessions (SearchResults | Onboarding+FirstRun together — #20 is one map item), one branch each, pinned to v3.12.0, shared files append-only, PRs merge serially with G1+G2 re-runs (WAYFINDER-MAP §Batching). Screens 1–18 are LANDED — do not touch. **firstrun.tmpl parses with the landed dashboard.tmpl** (its "home" define wraps "firstrun" when .EmptyEstate); search.tmpl calls landed "sevbadge" (signals.tmpl) — one parse set, nothing new to land for those.

## Screen 19 — SearchResults (`templates/search.tmpl` replaces templates_search.go)

Defines: "search" (kept) + "hisegs" (new). GET /search?q= skeleton kept. Changed (SPEC-CHANGE #25, ruled):
- **#25a** Matched terms render highlighted per the spec: every matched text field becomes a segment list `[{Text,Hit}]` rendered by "hisegs" — the handler splits each field on the FIRST case-insensitive occurrence of the query (`.NameSegs .RuleSegs .SubjectSegs .LabelSegs .TitleSegs .SnipSegs` replace the plain string holes; a non-matching field is one un-hit segment).
- **#25b** `.Assets[]` gains nullable `.Severity`/`.SevLabel` rendering the landed sevbadge after the type tag (the repo rows dropped what the spec shows).
- **#25c** The leading input icon drops — the spec's mono Input carries none; focus ring is the landed accent treatment (border + 3px soft ring), autofocus kept.
- Batch rows keep the BatchStatus chip (running pulses via the local keyframe); doc rows stay non-navigating; the empty state uses the landed accent-softer icon circle.
- Result count copy: `{{.Total}} results for “{{.Query}}”`.

## Screen 20 — Onboarding (`templates/onboarding.tmpl` replaces templates_onboarding.go) + FirstRun (`templates/firstrun.tmpl` replaces templates_firstrun.go)

### onboarding

Define "onboarding" kept; holes kept (see tmpl header) + `.StepValid` (NEW — server-computed validity of the rendered step; the no-JS floor for the Next/Start disable). Changed (#25):
- **#25d** PRG step-state GETs (batch-5 #23f precedent): each step POST 303-redirects to GET `/onboarding?step=N&seeds=…&profile=…&cad=…&cron=…&channel=…` — bookmarkable; the wizard goldens hit these URLs directly.
- The panel renders as the spec dialog-panel (560px, radius 24, shadow-lg) with the spec step rail (done = accent-soft check, current = 1.5px accent ring, lit connectors). Spec components: seed TagInput (chips with × remove — the existing `rm` submit; container focuses the input on click), RadioCards for scan profile (radius 14, 1.5px border, accent-soft when on, 4.5px radio dot), cadence as the spec listbox (view JS syncs the hidden `cad` input; native select drops; cron input appears on Custom…), channel Input with label + hint, Review as the sunken KV grid.
- Next/Start is JS-gated per spec validity (step 0: ≥1 seed chip OR typed seedsadd text — the handler absorbs typed text as a seed on submit; step 1: cron required when Custom…), with `.StepValid` rendering the disabled attribute server-side. Back posts formnovalidate.
- **#25e** NO cancel affordance — the mock dialog's X is a workspace affordance; first-run has nowhere to cancel to. Finish posts `/onboarding/finish` (kept), label "Start first scan".

### firstrun

Bare define "firstrun" — no head/chrome/foot; the landed dashboard.tmpl "home" define wraps it when .EmptyEstate. Holes: `.FirstRunDone` + `.FirstRunSteps[{Num,Done,Title,Detail,HasAction,ActionLabel,ActionHref,ActionPost,Gated,GateTitle}]`.
- **#25f** An action is a link (`.ActionHref` — steps 2/3: /scope, /settings/vantages) or a form post (`.ActionPost` — step 4's Run first batch enqueues; it cannot be a GET); exactly one is set when `.HasAction`. Step 4 renders the secondary disabled button with `.GateTitle` while `.Gated`.
- Layout per the spec: 760px column, 56/32 padding, 24px h1, honest sub copy, "N of 4 complete" micro, card of 4 rows (28px number/check circles — done = ok-soft/ok-border/ok), hairpinning footnote. `firstRunChecklist`'s ported step copy is the fixture copy — keep it verbatim.

## Fixtures (fixtures.json → search / onboarding / firstrun)

- search: q "acme" · 10 results — 4 assets (severities medium/high/critical/none, hrefs to /asset/…), 2 signals (SIG-1042/SIG-1036 drawer deep links), 2 batches (/runs/1407 complete · /runs/1398 failed), 2 docs; all matched fields pre-segmented. Variant `empty` = q "zzz-none", total 0.
- onboarding: seeds ["acmecorp.io"], profile standard, cad "Daily · 08:00", channel https://ops.example/hook, review slice (empty channel renders "none — inbox only").
- firstrun: 1 of 4 done; step 1 done (declared acmecorp.io), steps 2/3 linked, step 4 gated ("Needs an internet vantage first") posting /onboarding/finish.

## Goldens (crop `main` · × light/dark @1440)

search: default (/search?q=acme) · empty (/search?q=zzz-none). onboarding: wizard-1 · wizard-2 · wizard-3 · wizard-4 (the PRG GET URLs in states.json). firstrun: default (/ with the empty-estate variant).

## Acceptance

Byte-served from the three tmpls; no repo-authored markup/CSS remains for these routes (templates_firstrun.go deleted — the define now ships in the package and parses with dashboard.tmpl); search highlights the matched term in every row kind and asset rows carry severity; onboarding walks 4 steps by post-back with JS-gated validity, the cadence listbox works, and finish enqueues the first scan; the first-run checklist renders from real reads with step 4 gated until an internet vantage exists; G1+G2 green; SPEC-CHANGE gains no silent workarounds.
