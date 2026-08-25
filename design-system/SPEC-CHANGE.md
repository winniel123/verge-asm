# Spec-change requests — the collision protocol

Written 2026-08-24, alongside PARITY-CHART.md. This file exists to prevent the class of drift the v3 port produced.

## What went wrong

The port ran a local doctrine — "the domain term wins and the visual convention gets re-skinned around it"; "fabricated mock data is re-skinned to honest current-state facts + empty-states" — and applied it as a port-side judgment call. Each individual call was well-reasoned and well-documented, and the sum was a console that no longer matches the design: severity gone from four screens, spec regions replaced by placeholder empty-states, the shell trimmed. Nobody upstream ever saw the collisions to rule on them.

## The protocol

**A domain–spec collision is a design question, never a port-side decision.**

1. When implementing a spec region and the domain lacks a datum it renders (or a domain rule appears to forbid its shape), STOP work on that region. Do not re-skin, reshape, empty-state, drop, or add anything.
2. File the collision here, one entry in the log below: spec file + region, the domain fact in the way (with its source — CONTEXT.md line, ADR number), and the smallest honest alternative you would propose.
3. **Notify the operator and hand off.** Print the banner and the filled hand-off prompt from §Stop and escalate below, then end the work item (park the worktree; other items may continue). The operator pastes the prompt into the design workspace.
4. Design answers one of three ways: **build the datum** (schema/derivation work gets specced and charted), **spec changes** (design updates the .jsx + screenshot; the new spec is binding), or **region deferred** (removed from the spec until the datum exists — also a spec edit, never a silent hole).
5. Until answered, the spec stands and the region waits. An unanswered collision never ships as an improvisation.

Vocabulary is the one standing exception — signal / seed / channel / vantage / annotation, withdrawn never resolved — enforced in copy without a request.

## Stop and escalate (for Claude Code / Wayfinder)

Add this to the repo's CLAUDE.md verbatim so it binds every session:

```
## Design decisions on the web UI
The design package (design-system/) is normative for look AND functionality.
If a work item needs ANY design decision — a domain–spec collision, an
unspecced state, a missing datum, an ambiguity between spec file and
screenshot — do NOT decide, work around, or approximate. Instead:
1. Stop that work item immediately (other items may continue).
2. Append the collision to design-system/SPEC-CHANGE.md §Collision log
   with Ruling = "AWAITING DESIGN".
3. Print, as the final output of the item, the DESIGN DECISION NEEDED
   banner and the filled hand-off prompt from SPEC-CHANGE.md, so the
   operator can paste it into the design workspace.
4. Treat the item as blocked until the operator lands the updated design
   package containing the ruling.
```

Banner + hand-off prompt template (fill every ⟨⟩; keep it under ~200 words so it pastes cleanly):

```
┏━━ DESIGN DECISION NEEDED — work item ⟨id⟩ stopped ━━┓
Paste the prompt below into the Verge ASM design workspace.
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

Design decision needed on ⟨screen · region⟩ (SPEC-CHANGE.md collision #⟨n⟩).
Spec: ⟨examples/console/File.jsx · lines/region⟩, screenshot ⟨NN-console.jpg⟩.
Blocked work item: ⟨PARITY-CHART/WORK-CHART id⟩.
The collision: ⟨one sentence — what the spec renders vs. what the domain
holds, citing the domain source (CONTEXT.md / ADR-nnnn)⟩.
Options we see: (a) build the datum: ⟨what data/derivation⟩; (b) change the
spec: ⟨smallest honest alternative⟩; (c) defer the region.
Constraint(s): ⟨anything binding — migration cost, privacy, perf⟩.
Please rule, update the design + screenshot if the spec changes, and
re-export the package; we resume from the new PARITY/SPEC-CHANGE entry.
```

## Why this direction

The design workspace is where look and functionality are decided, reviewed, and signed off as one artifact. A port that "corrects" the spec locally splits that authority: the code becomes a second source of truth that only its comments know about. The protocol keeps the authority in one place and turns every collision into a recorded decision instead of a buried judgment call.

## Collision log

Merged 2026-08-24T14:52Z: round-1 execution filed #7–14; the four AWAITING DESIGN rows are now ruled (details in PARITY-CHART.md v3.2). The repo's log carries the full narrative for each; this copy is the ruled summary — when replacing design-system/ wholesale, this file supersedes.

| # | Spec region | Ruling |
| --- | --- | --- |
| 1 | Severity across Signals, Dashboard, Graph, Search | Build — P0.1. LANDED |
| 2 | Dashboard coverage + severity cards | Restore per spec — P2.1. LANDED |
| 3 | Reports trend KPIs, MTTW, heatmap | Build the series — P0.3, P2.4. LANDED |
| 4 | Exposure stat deltas | Build vs-last-batch deltas — P0.2, P2.6. LANDED |
| 5 | Search Documentation group | Index docs/guides/ — P2.5. LANDED |
| 6 | Signals Export CSV | In spec; exports the current tab's filtered rows. LANDED |
| 7 | Dashboard Vantages per-vantage latency | Build the datum — P0.5: measure RTT at the prober connect, nullable vantage column, probers.go read; spec's pending "—" until first measurement. |
| 8 | Certs-expiring ≤30d count + delta | Build — leaf not_after captured (CertVersion v2), #464. LANDED |
| 9 | Reports "New assets discovered" count + daily BarChart | Build — ListSubjectFirstAppearances → DiscoverySeries, #468 (P2.4b). LANDED |
| 10 | Inbox "Mark unread" | Build — MarkMessageUnread mutation + POST /messages/unread, #473; read is reversible per ADR-0116. LANDED |
| 11 | Report artifact severity bars + SeverityBadge column | Build — P2.10: RenderArtifact carries per-signal severity; renders when the delivery backend lands (empty-state until then is genuinely empty). |
| 12 | Scope declared-name TreeView | Build — declaredNameTree off the signal corpus, #474. LANDED |
| 13 | Inventory BulkActionsBar / SavedViews / TagInput filters | BLOCKED ON DESIGN — U6: the Inventory spec predates the subject/facet/span model and is being re-specced in the design workspace; those controls' fate is decided there. Repo keeps current Inventory; P2.8 is void. |
| 14 | Shipped-but-unspecced: Settings Sessions tab + Profile SSO section | CLOSED — spec catch-up in this package: Profile.jsx realigned to the shipped account model + gains the Linked-identities section (shot 14 recaptured); Settings Sessions already specced (U2, shot 25). No code change. |
| 15 | Reports "Recurring reports" — live scheduling (enabled wizard, editable list) vs shipped 501 (#344: no dispatcher/delivery backend) | Build it live — P0.6 (owner ruling 2026-08-24): dispatch + delivery backend gets built; spec unchanged. Unblocks P2.9, P2.10 rendering, delivery receipts. Tracking #497. |
| 16 | Report delivery transport — spec's email recipients vs ADR-0039 (estate never leaves the instance; SMTP deferred to v1.1) | Notify-with-link (owner ruling 2026-08-24): the recipient message carries only "your report is ready" + a session-authed URL — no estate in the body. Honors ADR-0039; keeps the spec's recipient UI. Needs the minimal mail/webhook transport for the notification only. Charted as P0.6b (map #499). |
| 17 | New-schedule wizard has no delivery destination; report_schedule.delivery_target is unbound free-text | Deliver to a Channel (design ruling 2026-08-24, follows #15+#16): wizard gains a Delivery step — Channel selector over Settings → Channels, default "Download only"; recurring list gains a Delivery column; report_schedule binds to a channel (nullable = download-only). The channel message stays link-only per #16. Spec updated (Reports.jsx + shot 15; DocsPage artifact copy fixed — no "email rendering"). |
| 18 | Profile reconciliation at the v4 conversion (screen 3) | Design ruling 2026-08-25: spec gains a plain token-revoke ConfirmDialog (shipped's typed-name gate relaxes — typed stays reserved for worst acts) and the TOTP-off state (badge + Enable flow); PRG notices/errors render as Callouts and act results ride the shell toast pipeline. Profile.jsx updated both sides; profile.tmpl ships in v3.6.0. |
| 19 | Batch-1 reconciliations (SignIn family, Setup, Coverage) | Design ruling 2026-08-25: (a) username identity everywhere pre-auth (spec catch-up — Email fields, trust-device checkbox, IdP-profile invite copy removed; recovery-code hint + honest reset/forgot copy adopted); (b) enroll = QR+secret+confirm page then recovery page with stored-checkbox gate; done cards stay preview affordances (real flow redirects); (c) address-scope coverage meters render counted/total (denominator = enumerable addresses of the declared range; name scopes stay census — refines ADR-0095, estate-proportion still forbidden); (d) password policy unified at 12+; (e) coverage messages carry the relative-time column. SignIn.jsx/Setup.jsx updated; three tmpls ship in v3.7.0. |
| 20 | Batch-2 reconciliations (Exposure, Drift, RunDetail) | Design ruling 2026-08-25, fully ruled (a–g), no AWAITING items; three tmpls ship in v3.8.0: (a) RunDetail Outcome = `.Transitions` + `.NewSignals` joined from this batch's diff — transitions folded from the diff stage, signals first raised in the batch (2026-08-24 binding ruling); the read joins derived stores, ADR-0041's corpus separation for dispatch execution stands and the comparison path does not move; `.Degraded` becomes nullable `{Vantage,Detail}`; (b) Drift period control is the spec's range picker — presets stay token links inside the popover, a custom ISO pair is added (GET `/drift?start&end` resolves a period window and mints a stable custom `.Period` token for the export link); (c) the repo's "Derived · drift" microlabel above the h1 drops; (d) a missing run retires its own define and routes to error.tmpl's missing-run kind (landed screen 2); (e) RunDetail log levels render as colored text per LogViewer.jsx — the level-pill treatment retires; (f) Exposure renders withheld first-class from `.Withheld`, the exposed delta chip from `.ExposedDelta.Change` via `signDelta`, and the withheld action links `/settings/vantages` (provisioning a prober is a vantage act), not `/scope`; the table-empty state keeps its honest Go-to-Scope action; the design-workspace "Spec state" segmented control is ruled away; (g) Drift kind chips and batch-group collapse are client-side JS shipped in the tmpl (ADR-0105 precedent), so `.Groups` always carries the full period feed and the Movement tally follows the `.Kinds` vocabulary order. |
| 21 | Batch-3 reconciliations (Scope, Dashboard, Signals) | Design ruling 2026-08-25, fully ruled (a–j), no AWAITING items; three tmpls ship in v3.9.0 and carry genuinely new backend reads. Scope: (a) `.Seeds[]` gains `.ID` with a chip-remove `POST /seeds/delete`; the seed form drops its kind select and the handler infers name-vs-address from the value's shape, and a block wider than `.AddressCap` REFUSES via the spec RefusalCallout (`.Refusal{Input,Reason,Reachable}`) naming a reachable /22 set — never auto-corrects; (b) per-name-scope custody toggle plus `.Census` meter (`.CustodyScopes[].Census`) over `POST /seeds/custody`; (c) zone re-supply becomes the spec FileDrop — the handler infers the scope from the uploaded file's apex and refuses an apex outside every name scope, JS auto-submits on pick or drop, the compact interval row stays on the same route; the `.NameTree[{Label,Count,Sev,Children}]` is a JS-collapsible tree; proposals become `.Proposals[{ID,Value,Kind,Source}]` plus `.OrgQuery` with registry org-name search `POST /proposals/search` (alias or BUILD), confirm-one `POST /proposals/confirm`, decline-many `POST /proposals/decline` recorded as exclusions; (d) the cold-tier opt-in and prober-provisioning regions relocate to /settings (design homes shots 17/18, landing fully at Settings map #21) and scope.tmpl stops rendering them, the zone-files evidence table re-homing with them. Dashboard: (e) `.ScanDetail` replaces the inline ActiveScans phrasing; (e2) `.CoverageMeters[]` gains nullable `.Total`+`.Pct` (address scopes counted/total per #19c, name scopes stay census); (e3) `.SilentZone{Bound,Text}` replaces `.SilentVantage`; (f) recent-signals rows gain `.SevLabel`+`.ViewKey` and deep-link to `/signals?view={key}`, Retry links /scans, the vantages empty state links /settings/vantages. Signals: (g) the PRG skeleton is KEPT — tab/q/sev/sort/page/view/descope stay query-string state on the existing routes, and view JS layers only what navigation cannot express (row kebab + right-click menus, Escape-close, annotate-enable, typed-confirm gate); rows gain `.SevLabel`+`.DescopeHref`; (h) the severity filter renders as the spec listbox whose options are links carrying the full query and the search input submits its GET form; (i) the `.Sort.*Arrow` holes retire for a caret rendered from `.Sort.Key`/`.Dir` (keys sev/asset/id/seen); the header Descope button drops (descope moves to row menus + drawer) and `.Descope` becomes `{Asset,CloseHref}` behind a typed-confirm dialog posting to the existing `POST /exclusions kind=subtree`; (j) the drawer gains the spec's real data — BUILD the reads, the data exists: `.Tags[]`, nullable `.CVE`, `.Desc`, `.RuleID`/`.RuleVersion` (the annotation form keeps posting `signal={{.RuleID}}`), `.DetectedBy`, nullable `.Diff{Title,Lines}` (the drift join for the subject), and span-derived `.History[{Title,Detail,Time,Tone,Mono}]`. Fixtures-mode determinism is additive and dev-gated; production seed/coverage/census/signal/exclusion paths are untouched. |
