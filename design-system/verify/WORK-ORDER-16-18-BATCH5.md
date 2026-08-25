# Work order — batch 5: Reports + ReportArtifact + Inbox (map #16–#18, package v3.11.1)

Use the consume-design-package skill. Fifth parallel batch: up to 3 sessions, one screen each, one branch each, all pinned to v3.11.1 for screen 18 (the v3.11.1 delta is inbox-only; reports/reportartifact bytes are identical to v3.11.0, so the open 16+17 PR stands — no restart), shared files append-only, PRs merge serially with G1+G2 re-runs (WAYFINDER-MAP §Batching). Screens 1–15 are LANDED — do not touch. **reports.tmpl and reportartifact.tmpl land together** (reportartifact calls "deltachip" from reports.tmpl; it also calls landed "sevbadge" (signals.tmpl) + "changeglyph" (drift.tmpl) — one parse set).

## Screen 16 — Reports (`templates/reports.tmpl` replaces templates_reports.go)

Defines kept: "reports" + "schedulewizard" + "deltachip" + "spark" + "barchart". Holes kept (see tmpl header). Changed (SPEC-CHANGE #23, ruled):
- **#23a** The third KPI is **"Mean time to withdrawal"** — the mock's "resolve" predates the withdrawal ontology; caption "critical + high only". Holes .HasMTTW/.MTTW/.MTTWDelta/.MTTWSpark.
- **#23b** The period control is the spec range picker (drift #20b mechanism): preset token links `/reports?period=` + custom ISO pair GET `?start=&end=`. `.Periods[{Token,Label}]`/`.Period`/`.PeriodLabel` join; `.RangeOptions`/`weeks` retire. Export links carry `period=`.
- **#23c** Export is the spec SplitButton: primary = CSV link; menu = JSON + **PDF**. `format=pdf` renders the period document via internal/message.RenderArtifactPDF — BUILD the route wiring (recompute from the period bounds, same machinery as the delivery PDF).
- **#23d** Discovery caption keeps the estate vocabulary: `{{.DiscoveryNames}} names · {{.DiscoveryServices}} services`.
- **#23e** The schedules table gains client-side sort on Report + Last sent (view JS; `.LastMins` is the numeric sort value; caret lights accent per direction). Row menus are the spec kebab dropdowns (view JS; routes kept: /reports/delivery, POST /reports/schedule/run, /reports/schedule/{id}/edit, POST /reports/schedule/delete; "View last delivery" disabled row keeps aria-disabled + title).
- Trend chart: server renders grid/polylines/x-labels from `.SignalSeries` (W1336 H230, PL40/PR8/PT10/PB22, yTop from the tick rule); the hover readout (crosshair + per-series dots + tooltip) is view JS reading `data-n/w/h/labels/series` — same math, no new holes beyond `.N/.LabelsAttr/.SeriesJSON`.
- Sparkline draw-in / bar rise / area+dot fades are CSS-only (component motion); goldens capture the settled state (delay ≥700ms).

### schedulewizard (route kept: /reports/schedule/new + /reports/schedule/{id}/edit)

**#23f** The wizard stays the PRG post-back flow at its routes, rendered as the spec panel (560px, radius 24, shadow-lg) with the spec step rail (done = accent-soft check, current = 1.5px accent ring, lit connectors). The mock's overlay presentation is a workspace affordance — the routed page is correct. Changes:
- Cadence + Destination render as spec listboxes (view JS syncing hidden `cad`/`channel` inputs; native selects drop). Channel options carry `.Hint` ("artifact stays in Reports" / "signed HTTPS channel" / "signed HTTPS channel · paused") — extend the channel option struct.
- Next/Create is JS-gated per the spec's validity rules (step 0: name + ≥1 section; step 1: cron required when Custom…); Back posts formnovalidate; the server still validates.
- **PRG shape**: each step POST 303-redirects to GET `/reports/schedule/new?step=N&name=…&sections=…&cad=…&cron=…&channel=…` carrying accumulated values — bookmarkable and harness-addressable (the wizard goldens hit these URLs directly). `.ChannelLabel` joins (the trigger shows the label, not the id).

## Screen 17 — ReportArtifact (`templates/reportartifact.tmpl` replaces templates_reportartifact.go)

Defines: "reportartifact" (page) + **"artifactdoc"** (the delivered document).
- **#23g — the document body moves into the design-owned "artifactdoc" define.** internal/message.RenderArtifact re-points at it: parse this tmpl once, execute "artifactdoc" with the recomputed data — the on-screen page, /reports/delivery/pdf, and the email form all render THIS markup. The define is fully inline-styled (no dependence on the page <style>) precisely so it ports across those shells. Document holes: `.Empty .Org .Generated .Version .Format .Stats[{Label,Value,Delta{Has,Text,Dir,Tone},Caption}] .SevBars[{Sev,Label,Pct,Count}] .NewRows[{Severity,SevLabel,Signal,Asset,Seen}] .Withdrawn[{Text,Reason}] .DeliveredAt .DeliveredTo .DeliveryState{Label,Tone}`. Never-delivered renders `.Empty` inside the document (ADR-0110) — the PDF is the empty-state document until a delivery lands.
- **#23h** "Edit schedule" links `/reports/schedule/{{.ScheduleID}}/edit` (the route exists now); `.ScheduleID` nullable — a gone schedule renders the disabled treatment with the honest title.
- Page holes: `.Heading .Period .ScheduleID .Doc`. Stats vocabulary follows #23a/#23d (Mean time to withdrawal; names · services).

## Screen 18 — Inbox (`templates/inbox.tmpl` replaces templates_inbox.go)

Define "inbox" kept. PRG skeleton kept: open = `?id` link (marks read), filter = query param, mark-all-read / mark-unread POSTs with `return`. Holes kept (see tmpl header). Changed:
- **#24 (ruled, supersedes the #23i .Body item): the detail card carries NO prose body.** The sample paragraph predates the census store; the ADR-0064 valence-free discipline stands — build no prose grammar, no carve-out. The detail form IS the census + delivery receipts; depth is one jump away on the signal. There is no `.Body` hole.
- (#23i, remainder) The census + delivery regions become design-owned in-tmpl (the repo's global `msgcensus`/`msgdelivery` classes retire for this route): census = sunken rows (kind micro-label + mono key, linked); deliveries = receipt lines — failed renders the danger "undelivered" pill + host + "could not be delivered, not that nothing fired" + dotted `(reason)` tooltip from `.LastError`.
- Map note: the map's "mark-unread menu" golden state predates the mock — mark-unread is a button in the detail card, covered by the message-open golden.

## Fixtures (fixtures.json → reports / reportartifact / inbox)

- reports: period 7d of 4 tokens; KPI 47 (+3 bad, 14-pt spark), 12 (+8 neutral, 8 names · 4 services, 14 bars), MTTW 2.4d (−0.6d good, chart-2 spark); trend series (yTop 60, ticks 0/20/40/60, labels Aug 9–22 sparse ×5, LabelsAttr + SeriesJSON for the hover JS); sev bars 3/11/18/9/6; 84-day heat (levels precomputed); 3 schedules (s1–s3, LastMins 4320/31680/840); wizard slice (4 sections with keys, 5 cadence presets, 3 channels with hints, PRG note).
- reportartifact: delivery #42 of s1 (org acmecorp · generated 2026-08-22T09:00:00Z · verge v0.9.2 · pdf); 3 stats; sev bars; 3 new rows; 2 withdrawn; receipt delivered · ok. Variant `never-delivered` pins .Doc.Empty (schedule s2).
- inbox: 5 messages (m1–m5, 2 unread, ISO instants); selected fixture m1 with census (Signal → /signals?view=SIG-1042 · Asset → /asset/… · Batch → /runs/1407), 2 deliveries (delivered ops hook; failed pager with LastError "TLS handshake failed · 3 attempts"); jump labels per class.

## Goldens (crop `main` · × light/dark @1440)

reports: default · range-open · row-menu-open · wizard-1 · wizard-2 · wizard-3 · wizard-4 (the wizard states hit the PRG GET URLs in states.json; chart/spark states settle — delay ≥700ms). reportartifact: default · never-delivered (variant). inbox: default · message-open (?id=m1) · unread-filter (?filter=unread).

## Acceptance

Byte-served from the three tmpls; no repo-authored markup/CSS remains for these routes; the artifact document renders from "artifactdoc" on screen AND in the PDF/email forms (one markup, three shells); reports period picker + export split + row menus + client sort + chart hover all work; the wizard walks 4 steps by post-back with JS-gated validity and lands the create/update; inbox opens/marks/filters by URL state, failed deliveries render the undelivered treatment; G1+G2 green; SPEC-CHANGE gains no silent workarounds.
