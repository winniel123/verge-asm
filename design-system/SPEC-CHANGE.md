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
| 28 | Severity docs vs shipped ramp (drift audit 2026-08-26 item 6) | Docs are the stale side — #1/P0.1 ruling stands; the 5-level ramp is normative. Repo updates signals.md §26-30, v1-spec §488-489, README. No design or code change. |
| 29 | Reports Run-now never notifies (audit: reverse-drift asymmetry) | Deliberate and stands: Run-now is download-only (the operator is present); only the cadence tick delivers via the bound channel. Repo documents it in reports.md; docs also drop the stale "off-instance delivery not built / #17 AWAITING DESIGN" note — #17 is built and landed. `/reports/export?format=pdf` is spec-normative (#23c) — document it, fix the stale handler comment. |
| 30 | Four runnerless RIR proposer tiles (RIPEstat, RIPE DB, APNIC-registry, LACNIC) render consent+toggle but emit nothing | Move them to the existing "Catalogued — not yet executing" bucket (the #241 mechanism, non-toggleable, no consent dialog offered) until a real proposer.Source runner lands. Catalogue-data change in cmd/web/sources.go — the spec sources panel already renders that bucket; no template change. sources.md updated to match. |
| 31 | Schedule cadence presets ("Daily · 08:00") + Custom cron are declared but epoch-floored/ignored | Spec is normative (2026-08-24 binding ruling): BUILD time-honoring for presets and real cron interpretation for Custom. UI unchanged. Until it lands, reports.md states runs fire on epoch-aligned windows. |
| 32 | Drift legend renders all six transition words; four kinds cannot fire (estate wiring unwired) | Wire internal/estate into spanfold closure (the audit's own high-leverage fix) — the legend then reads true and stays. Legend trims ONLY if that wiring is explicitly deferred past the next round; a legend that teaches grammar the engine cannot produce violates the honesty discipline. |
| 33 | Org switcher — carried from batch 8 (#27b nullable-pending-store), renumbered by the repo to avoid the audit's #28 | RETIRE (design ruling 2026-08-26): single-org stands per ADR-0073 — no multi-org store gets built. shell.tmpl drops the OrgSwitcher popover permanently; the static org chip is the only rendering; .Chrome.Orgs and /org/switch never ship; the org-open state and fixture orgs drop. TopNav component updated to match. Ships in v3.17.0. |
| 34 | RunDetail loghead (incl. the DF-F3b job-filter chip) gated inside {{if .Log}} — the specced unknown-job edge (chip over "No log to show") cannot render | Move the loghead outside the conditional: rd-log always renders head (title + chip + live), body OR the 300px-centered empty. rundetail.tmpl updated; no repo change (.JobFilter already set). Ships in v3.17.0. |
| 39 | Settings · Integrations "Send test" (follows #38) — #38 ruled "POST through the worker to the channel URL, handler-only," but an integration tile holds NO channel URL and NO Channel binding (integration_state = slug/state; migration 20600 keeps Channels independent: "an integration is NOT a channel"). SendSigned has no target | Ruling (b), 2026-08-27: **#38's "handler-only / no new backend" premise is void and is hereby lifted.** The drawer's own Callout already states the model — "Channels deliver raw JSON to any URL; integrations add formatting… on top" — so an integration is a formatting layer over a Channel, and the fix is a **nullable integration→channel binding** (a *reference*, not a fold — honoring "an integration is NOT a channel"), the same pattern report schedules use to bind a Channel (#17). (a) minting a per-integration URL+secret is rejected: it duplicates the Channel model and contradicts the on-screen architecture. (c) retire is rejected: the affordance is real once bound. **Design landed HERE (v3.23.0):** the installed drawer gains a "Delivery channel" Select (bound Channel or "Not connected", set post-install); Send test is **disabled with "Connect a channel to test" when unbound** (no fake toast — #37 honesty) and posts the formatted payload through the bound Channel's SendSigned when bound. Applied to Integrations.jsx + ui_kits/console + (repo) settings.tmpl. P0.14 re-scoped: schema (binding column) + drawer selector + gated handler. **No screenshot change** — the ground-truth set (screenshots/README) has no Integrations-drawer shot; slot 21 is Error 404, so #38/#39's "screenshot 21" reference was stale. Round-3 verify V stays open on the re-scoped P0.14. |
| 38 | Settings · Integrations drawer "Send test" — spec ships the button + success toast "Test message sent…" (Integrations.jsx:64/:74, settings.tmpl:1310), reachable in production, but `testIntegration` (integrations.go:298-305) validates the slug and redirects with NO delivery and NO toast — a dead affordance | Ruling (a), 2026-08-27: **build the datum.** Two prior rulings foreclose the alternatives — (c) surfacing the "Test message sent" toast without a send is the #37 honesty violation (a confirmation from evidence not held), and (b) retiring the button drops a specced affordance (Acceptance §5) to match an unwired backend, the inverse of the standing exact-parity doctrine (spec stands; unwired data gets built — P0.7/P0.10a pattern). The constraint confirms no new backend: the delivery worker is integration-agnostic (raw JSON → channel URL). Wire `testIntegration` to POST a real test payload through the existing worker to the channel URL and, on enqueue success, render the spec's exact toast ("Test message sent — Check &lt;name&gt; for the delivery."); on failure, the spec's existing error/degrade path. Charted P0.14. **No design, screenshot, or template change** — the button, toast copy, and the :74 Callout already spec the correct behavior; only the handler is wrong. Closes round-3 verify item V; E13 stays (docs states-table wording is separate). |
| 37 | Signals · certificate rules / CertDetails fold — chart says 6 cert rules light from fields already on the `certificate` observation, but the CertVersion v2 leaf parses only not_after/issuer/algorithm/chain (2 rules); CertDetails is one pointer gating all 6, so a partial construct fires certificate-hostname-san-mismatch off a defaulted-false SANMatchesName | Ruling (b) now + (a) charted, 2026-08-27. **The honesty discipline is dispositive: a rule must never emit a verdict from evidence not held** — SANMatchesName defaulting false to manufacture a mismatch is the exact failure this file exists to prevent. So: **(1)** CertDetails becomes **per-attribute readable** — each field independently nullable, no single pointer gates the set. Each rule evaluates only when its own inputs are present: `certificate-expired` + `certificate-expiring` read `not_after` (in the v2 leaf) → **light now**; `certificate-not-yet-valid` (needs not_before), `certificate-hostname-san-mismatch` (needs leaf SANs), `certificate-weak-key-or-signature` + `certificate-self-signed` (need unspecified determinations) → **not-evaluable** until their datum lands. SANMatchesName MUST NOT default false: absent SANs = not-evaluable, never a mismatch. **(2)** The 4 dormant rules stay in the rule set rendering the existing **not-evaluable** state (AL-6's shipped pattern; Acceptance §7) — spec regions are never replaced (#32 doctrine); no legend/screen edit. **(3)** The full build is charted as the deferred datum item P0.10b — CertVersion **v3** leaf (not_before + leaf SANs) + specified weak-key/signature + self-signed policies, re-gate certcorpus — batched, not piecemeal, because the version bump breaks `certificate` timelines by design (ADR-0082). No design/fixture/screenshot change: Signals already renders fired instances (P2.2) + not-evaluable cert rules; lighting 2 rules adds no new visual state. |
| 36 | Drift · vs-previous TransitionDelta chip — #32's estate wiring already LANDED (#637, merge 51141e8: foldEstateTransitions in the batch tx, all six kinds fire); residual is drift.go:331's hardcoded `""` and unspecced compare | Ruling (a), 2026-08-27: P0.12 re-scopes to **TransitionDelta chip only**; the wiring is LANDED and must not be re-filed. Compare semantics (now normative, matching the spec's existing "vs previous period" caption + suppression rule): **delta = .TransitionCount of the selected period window minus the transition count of the immediately preceding window of equal length ending at the window's start** — same scope/filters as .TransitionCount, all six kinds, presets and custom ISO ranges alike. Nonzero renders signed ("+2"/"−1"); zero renders "0"; **no complete previous window** (install younger than 2× the window, or no batches span it) → empty string → chip suppressed, per the spec's own degrade (P0.2 pattern). No design, fixture, or template change — "+2" and the "vs previous period" caption already encode this. |
| 35 | /runs/1408 identity collision: fixtures made 1408 the Settings active dispatch while states.json maps error·missing-run to /runs/1408 | Active dispatch repoints to 1409 (fixtures id/href + job hrefs; stop/terminate confirm states → 1409); error·missing-run keeps 1408. rundetail gains the running state (/runs/1409) — the repo's parked running-run demo re-enables against 1409 and its golden lands with this version. Ships in v3.17.0. |
