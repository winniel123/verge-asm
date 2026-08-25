# Work order — batch 3: Scope + Dashboard + Signals (map #10–#12, package v3.9.0)

Use the consume-design-package skill. Third parallel batch: up to 3 sessions, one screen each, one branch each, all pinned to v3.9.0, shared files append-only, PRs merge serially with G1+G2 re-runs (WAYFINDER-MAP §Batching). Screens 1–9 are LANDED — do not touch. **dashboard.tmpl and signals.tmpl land together** (dashboard's recent-signals rows call the "sevbadge" define that signals.tmpl declares; they parse into one set — same mechanism as signin/setup in batch 1).

## Screen 10 — Scope (`templates/scope.tmpl` replaces the "scope" define in templates_scope.go AND the repo-authored "proposals" define — delete both)

Holes kept: .Notice .IsAdmin .AddressCap .FormScope .FormError .Exclusions .ExclError .ExclKind .ExclValue .ExclPreview{Fires,Headline,Loss} .ZoneError .ZoneIntervalDays .ZoneIntervalError .CustodyError. Changed/new holes:
- `.Seeds[]` gains `.ID` (chip remove posts /seeds/delete — add the action if absent; SPEC-CHANGE if impossible).
- `.Refusal{Input,Reason,Reachable}` (nullable) — the spec RefusalCallout; a refused declaration sets it via PRG alongside `.FormError`.
- Seed form: the kind `<select>` drops — the handler infers name/address from the value's shape (#21a). A block wider than `.AddressCap` REFUSES with the reachable /22 set named, never auto-corrects.
- `.CustodyScopes[]` gains `.Census` (int) — the spec toggle + census meter renders once per name scope (#21b); the switch is the POST /seeds/custody submit.
- Zone upload is the spec FileDrop: the seed select drops; the handler infers the scope from the uploaded file's apex and refuses an apex outside every name scope, with the reason (#21c). JS submits on file pick/drop. Interval form stays (compact row, same route).
- `.NameTree[{Label,Count,Sev,Children[{Label,Sev}]}]` — JS-collapsible tree (view JS in tmpl).
- `.CoverageMsgs[]` gains `.Kind .Bound .ISO` (same shape as coverage.tmpl messages).
- Proposals: `.Proposals[{ID,Value,Kind,Source}]` + `.OrgQuery`. Org-name search posts `/proposals/search` (org) — registry org search is spec functionality: alias or BUILD; confirm-one posts `/proposals/confirm` (id); decline-many posts `/proposals/decline` (ids, checkbox form-attribute association). Declines are recorded as exclusions.
- **Relocations (#21d): the cold-tier opt-in and prober provisioning regions leave /scope for /settings** (their design homes — shots 17/18, landing fully at map #21). Move the existing repo-authored regions + routes under /settings now; scope.tmpl does not render them. The zone-files evidence table also leaves /scope (its data re-homes with Settings → Sources at #21).

## Screen 11 — Dashboard (`templates/dashboard.tmpl` replaces templates_dashboard.go; defines "home" + "dashboard" kept, "firstrun" stays repo-authored until #20)

Holes kept: .EmptyEstate .ScanSchedule{…} .Scanning .Unavailable .ProbeDismissed .StatBand[{Label,Value,Live,HasDelta,Change,Tone,Caption}] (signDelta stays) .HasSignals .SevBars[{Sev,Pct,Count}] .Vantages[{Name,Latency,Avail}]. Changed:
- `.ScanDetail` (NEW string, e.g. "214 subjects queued") replaces the inline ActiveScans phrasing (#21e).
- `.CoverageMeters[]` gains nullable `.Total` + `.Pct` — address scopes render counted/total per #19c; name scopes stay census (#21e2).
- `.SilentZone{Bound,Text}` (NEW, nullable) replaces `.SilentVantage` (#21e3).
- `.RecentSignals[]` gains `.SevLabel` + `.ViewKey` — rows deep-link to `/signals?view={key}` (#21f).
- Run scan links /scans; Retry now links /scans; dismiss stays `/?probe=dismissed`; the vantages empty state links /settings/vantages.

## Screen 12 — Signals (`templates/signals.tmpl` replaces templates_signals.go; defines "signals" + "sevbadge" + "sevbadge-md" + "withdrawnmark")

PRG skeleton KEPT: tab/q/sev/sort/page/view/descope stay query-string state on the existing routes; view JS layers only what navigation cannot express (#21g): row kebab menu + right-click menu (items are links/actions), Escape-to-close, annotate-enable, typed-confirm gate. Holes kept: .Tab .OpenCount .AnnotatedCount .WithdrawnCount .Q .Sev .SevOptions .Shown .Total .HasAny .ClearHref .HasExport .ExportHref .IsAdmin .AnnoError .ViewPrefix .SelKey .Sort{Key,Dir,SevHref,AssetHref,IDHref,SeenHref} .ShowPagination .PageInfo .Pages .PrevDisabled .PrevHref .NextDisabled .NextHref .CloseHref. Changed:
- Header Descope button drops — descope lives on row menus + drawer context (#21).
- `.Rows[]` gains `.SevLabel` + `.DescopeHref` (per-row `/signals?…&descope={key}`, filters preserved).
- `.Sort.*Arrow` string holes retire — the caret svg renders from `.Sort.Key`/`.Dir` (#21i). Sort keys: sev/asset/id/seen.
- Severity filter renders as the spec listbox whose options are links carrying the full query (#21h); the search input submits its GET form (tab/sev/sort/dir as hidden fields).
- `.Descope` becomes `{Asset, CloseHref}` — the spec typed-confirm dialog: the operator types the exact asset (JS gates the danger button; the typed value posts as `value` to the existing POST /exclusions kind=subtree; server still validates).
- `.Drawer` gains the spec's real data (#21j — BUILD the reads; the data exists): `.Tags[]` (rule tags), `.CVE` (nullable), `.Desc` (rule description prose), `.RuleID`/`.RuleVersion` (replaces `.Signal`; the annotation form posts `signal={{.RuleID}}` — keep accepting the same field), `.DetectedBy` (vantage), `.Diff{Title,Lines[{Type,Text}]}` (nullable — the drift join for this subject), `.History[{Title,Detail,Time,Tone(accent|warn|danger|neutral),Mono}]` (span-derived: still present / drift detected (when a diff exists) / signal raised / asset discovered — see fixtures.signals.history_rule).
- Annotated drawer state keeps `.AnnoID` + `.AnnoReason`; routes /annotations + /annotations/withdraw unchanged.

## Fixtures (fixtures.json → scope / dashboard / signals)

- scope: 2 seeds; refusal via posting `203.0.113.0/20` (reason "Spans 4,096 addresses — the cap is 1,024 per scope.", reachable /22); custody on for acmecorp.io (census 62); one zone file (2026-07-30 · monthly · "ages into a gap in 7d", interval 30); the 7-leaf name tree; 3 coverage messages; 3 proposals + org-search fixture (adds 203.0.114.0/25, ARIN); 2 exclusions; exclusion-preview fixture (subtree staging-4.acmecorp.io → fires).
- dashboard: schedule 38m/5h 22m; 5-cell stat band with deltas; sev bars 3/11/18/9/6; meters 212/256 (pct 83) + census 1,284; silent zone (9d · internal.acmecorp.io); 3 vantages (ap-south-1 unavailable · 12m); recent = first 6 signal rows. Variant `scanning` pins .Scanning + "214 subjects queued".
- signals: the 10 open rows (of 47 · page 1 of 5 · "1–10 of 47"), 3 withdrawn, annotations on SIG-1027/SIG-1024, diffs on SIG-1042/SIG-1036, rule refs (vnc-exposure@3 / tls-acceptance@2 / dns-mail-policy@1, default svc-exposure@3), detected_by "vantage eu-west-1", default sort sev·asc.

## Goldens (crop `main`; drawer/descope states crop `body` — the overlay escapes main · × light/dark @1440)

scope: default · refusal (post the /20) · exclusion-preview. dashboard: default · scanning (variant) · banner-dismissed (/?probe=dismissed). signals: default · drawer-open (view=SIG-1042) · drawer-annotated (view=SIG-1027) · descope-confirm (descope=SIG-1042) · withdrawn-tab · menu-open (click a kebab).

## Acceptance

Byte-served from the three tmpls; no repo-authored markup/CSS remains for these routes (including the old "proposals" define); seed declaration infers kind and refuses over-cap blocks with the spec callout; zone upload infers apex and auto-submits on pick/drop; proposals confirm one / decline many with a live count; name tree collapses; dashboard rows deep-link into the signals drawer; signals tabs/sort/filter/page/drawer/descope all work as URL state with the spec visuals; kebab + context menus open; descope confirm enables only on the exact typed asset; annotation accept enables only with text; cold tier + probers now render under /settings and are gone from /scope; G1+G2 green; SPEC-CHANGE gains no silent workarounds.
