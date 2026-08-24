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

| # | Spec region | Domain fact | Proposed | Ruling |
| --- | --- | --- | --- | --- |
| 1 | Severity across Signals, Dashboard, Graph, Search | CONTEXT.md: "a signal carries no severity" | Presence-only re-skin (shipped, unapproved) | Build the datum — PARITY-CHART P0.1 |
| 2 | Dashboard coverage + severity cards | Denominators live on Coverage; no ramp | Empty-state pointers (shipped, unapproved) | Restore per spec — P2.1 |
| 3 | Reports trend KPIs, MTTR, heatmap | No trend series carried | Honest scalars (shipped, unapproved) | Build the series — P0.3, P2.4 |
| 4 | Exposure stat deltas | Exposure is a current-state census | Drop deltas (shipped, unapproved) | Build vs-last-batch deltas — P0.2, P2.6 |
| 5 | Search Documentation group | No content store (#316) | Drop the group (shipped, unapproved) | Index docs/guides/ — P2.5 |
| 6 | Signals Export CSV | — (was never a collision; the button is in Signals.jsx) | — | Specced: exports the current tab's filtered rows; see Signals.jsx header |
| 7 | Dashboard Vantages card — per-vantage latency in ms (Dashboard.jsx VANTAGES "34ms"/"51ms"; PARITY-CHART P0.4/P2.1) | No latency is measured or stored anywhere: the `vantage` table has no latency column (db/migrations/18700), no observation/facet carries a round-trip, and the worker prober connect is itself unlanded — cmd/worker/vantages.go: "the host key is pinned on the first real connect a later ticket wires." So P0.4's cmd/web-only scope has no datum to read and no write path to populate. | Build the datum in a dedicated measurement ticket: capture round-trip latency on the prober connect and store it on the vantage (nullable), which the probers.go read then surfaces. Not derivable port-side from cmd/web. | AWAITING DESIGN |
| 8 | Dashboard Certs-expiring ≤30d stat cell (Dashboard.jsx) + its vs-last-batch delta (#443, P0.2) | The `certificate` facet value stores only `{outcome, chain[]fingerprints}` (connectoutcome/tls.go, InsecureSkipVerify — fingerprints, never the parsed leaf). No `notAfter` is captured, so `CertDetails.Expiring` is always nil and the `certificate-expiring` rule is structurally not-evaluable (signal/endpoint.go; ListEndpointCertificates comment). A ≤30d count needs each cert's expiry, which the TLS corpus does not hold. | Build the datum — #464 (this PR): captured the leaf's `notAfter` in the tls-handshake leaf (CertVersion v1→v2, certcorpus re-locked); the `certificate` facet value now carries `not_after` (RFC3339), which the P0.2 certs-delta and the P2.1 Dashboard count read. |
| 9 | Reports KPI band, card 2 — "New assets discovered": a per-period discovery COUNT + a daily-discovery BarChart (Reports.jsx `DISCOVERY`, value "12" / "+8" / "8 domains · 4 IPs"; shots 09→10). P2.4/#450. | No per-period asset-discovery count or daily-discovery series is a built datum. P0.3 built signals-over-time, the heatmap ramp and mean-time-to-withdrawal, and P0.2 built a vs-last-batch **assets-watched** census delta (internal/drift.DistinctSubjects over open name/service spans) — but neither carries "assets that first appeared over the range" nor a per-day discovery series. Deriving a daily first-appearance series needs a new span read (MIN(opened_at) per subject / the `appeared` drift-event classification), beyond P2.4's markup + existing-datum-wiring scope. | **Build the datum — #468 (P2.4b).** Derivable from span history (not a design change), so per ADR-0116 the ruling is build, not rename/defer. #450 shipped the honest interim "Assets watched" (P0.2 delta) in the slot meanwhile; #468 builds the daily first-appearance series and swaps card 2 to the spec's "New assets discovered" + BarChart. **SHIPPED (P2.4b, branch parity/p2.4b-assets-discovery-468):** new span read `ListSubjectFirstAppearances` (MIN(opened_at) per name/service subject, `appeared` classification) → `internal/drift.DiscoverySeries`/`DiscoveryCount` fold the per-day series, the per-period total, and the name/service split; card 2 now renders "New assets discovered" (count + vs-previous-period delta + daily BarChart), degrading to the ReportCard empty pattern where the read is unavailable. |
| 10 | Inbox detail actions — the "Mark unread" ghost button (Inbox.jsx:59). Surfaced by the P3 closing gate (#454). | Read-state is a monotonic per-account fact: `markMessageRead` sets a `ReadAt` timestamp and the code states "there is no un-read, since a message is read once the operator has seen it" (cmd/web/messages.go:118-120). There is no `MarkMessageUnread` store method or un-set path. | Restoring the control reverses a stated domain rule and needs a new mutation (clear the read-set row) + its query. Smallest honest alternative: keep the drop until design rules whether read should be reversible; if yes, build the un-read mutation. | AWAITING DESIGN |
| 11 | Report artifact body — the "Open signals by severity" bars (ReportArtifact.jsx:44-53) and the SeverityBadge column in the "New this week" table (ReportArtifact.jsx:58). Surfaced by the P3 closing gate (#454). | internal/message/render.go re-casts both into drift-vocabulary `ArtifactChange` rows and omits the severity bars, citing ADR-0024 ("a signal is a census member, not scored"). P0.1 superseded that reading for the console (internal/signal.SeverityFor is a real ramp, already rendered in Signals/Asset/Search/Graph, and reports.go calls SeverityFor), so the offline artifact model is the lone holdout. The body is additionally always the empty-state at runtime — reportDeliveryPage passes a zero `message.Artifact` (no delivery backend; #285/#290/#291). | Restore severity in the artifact document model: RenderArtifact carries per-signal severity → renders the by-severity bars + a SeverityBadge column, wired when the delivery backend lands. This is an internal/message model change reconciling ADR-0024 vs P0.1 for the offline artifact, not a template restyle — a design call. | AWAITING DESIGN |
| 12 | Scope "Declared name tree" TreeView (Scope.jsx:86-98) — a per-leaf tree (acmecorp.io → www/api/…) with per-leaf severity. Surfaced by the P3 closing gate (#454). | No handler produces a per-leaf declared-name tree with severities for Scope (a grep for NameTree/TreeView across cmd/web finds nothing); it is not a built datum, and it arguably duplicates the Inventory grouped view. | Build the datum (a declared-name tree read + per-leaf max-severity) or defer the region; whether Scope carries a name tree at all vs delegating to Inventory is a design call. | AWAITING DESIGN |
| 13 | Inventory — BulkActionsBar (Rescan/Add tag/Annotate/Remove + row-selection + ConfirmDialog, Inventory.jsx:117,121-131), SavedViews (persisted filters All/Ranges/Critical + save/dirty, Inventory.jsx:100-102), and TagInput filters (`type:`/`sev:` with suggestions, Inventory.jsx:112-114). Surfaced by the P3 closing gate (#454). | None are backed: no bulk-mutation handler over a selection, no saved-view persistence, and no inventory filtering exist in any handler. The spec JSX predates the shipped subject/facet/span domain model (its columns are the older exposure-centric mock). | Build the datums/handlers (selection mutations, saved-view store, filter query) or re-spec Inventory onto the current domain model. A multi-surface design decision, not a port-side fix. | AWAITING DESIGN |
| 14 | Shipped-but-unspecced sanctioned surfaces: the Settings "Sessions" tab (templates_settings.go:37,53,616) and the Profile "Access · single sign-on / Linked identities" section (templates_profile.go:103-130) — both render live, neither appears in Settings.jsx / Profile.jsx. Surfaced by the P3 closing gate (#454). | Not domain collisions but sanctioned feature work — sessions (#406-408, ADR-0117) and SSO (#392, ADR-0112) — that postdates the frozen JSX specs. Acceptance criterion #6 requires a user-facing addition to be reflected in the spec. | Design-package catch-up: add the Sessions tab and the Profile SSO/linked-identities section to Settings.jsx / Profile.jsx (+ screenshots) so the spec matches the shipped, approved surfaces. No code change — the features stay; the spec gains them. | AWAITING DESIGN |
