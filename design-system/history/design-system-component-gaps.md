# Component gaps — ASM peer features vs. the Verge kit

**Companion to** `design-system-redesign-brief.md`.
**Purpose:** a build-ahead list of design-system components that other attack-surface-management tools ship and Verge will likely need — filtered through Verge's actual domain model, so we build what fits and skip what fights it.
**Date:** 2026-08-22.

---

## How to read this

Peer ASM tools (Microsoft Defender EASM, Palo Alto Cortex Xpanse, Censys ASM, CyCognito, Tenable, Rapid7 Surface Command, Detectify, Bishop Fox Cosmos, Randori, runZero, Intruder, Shodan Monitor) converge on a recognizable feature set. Most of it maps cleanly onto components Verge doesn't have yet.

But Verge is **not** a generic inventory scanner. Its subject is **change** — "what moved since last time" — and its domain model (`CONTEXT.md`) makes some deliberate exclusions. So this list has two halves:

- **§1–§11 — build these.** Grouped by feature area, with the specific new components and a priority.
- **§12 — deliberately don't build these.** Standard peer features that contradict the model, each with the reason.

Two standing rules from the domain (they shape naming everywhere below):

- **The word is "signal," not "finding."** Also avoid `host`, `target`, `job`, `webhook`, `integration`, `triage state`, `risk acceptance workflow` — the glossary rejects each. A naming cheat-sheet is in §13.
- **Change vocabulary is its own visual language, separate from severity.** `appeared / revealed / withdrawn / descoped / returned / changed` are not severities and must never borrow the red→gray severity ramp.

**Current kit (22 components), for reference:** Button, IconButton, Input, Select, Checkbox, Radio, Switch · Card, Badge, SeverityBadge, Tag, Stat, StatusDot, Table · Dialog, Toast, Tooltip, EmptyState · TopNav, Tabs, Footer, Wordmark.

Priority key: **P0** core to Verge's thesis or unblocks a screen that doesn't exist yet · **P1** needed broadly, most screens want it · **P2** valuable, not urgent.

---

## 0. Reconciliation — extend what exists, don't duplicate it

**Designer: read this section first, and treat it as authoritative wherever a later section says "no component exists" or lists something as new.** Parts of this kit are already built. Several "new" items below are really *extensions* of an existing component or *formalizations* of code currently hand-rolled inside the reference screens (`ui_kits/app/*.jsx`). Build on those; don't reinvent them.

Status legend: **Exists** (use as-is) · **Extend** (add variants/props to a current component) · **Formalize** (promote hand-rolled screen code into a real component) · **New** (genuinely absent).

| Item in this doc | Status | What already exists |
| --- | --- | --- |
| FilterChip / filter token (§2) | **Exists** | `Tag` — has `active`, `onRemove`, `onClick`. This *is* the filter/toggle chip. |
| FilterBar (§2) | **Compose** | Not a primitive — `Inventory.jsx` already composes `Input` (mono search) + removable `Tag`s + `EmptyState`. Formalize that composition. |
| SearchInput, basic (§11) | **Exists** | `Input mono` (used in Inventory). Only *typeahead* is an extension. |
| DataTable (§2) | **Extend `Table`** | `Table` already has mono cells, `align`, custom `render`, `onRowClick`, `selectedKey` (accent-soft fill + inset bar), `dense`. Add: sort, sticky header, **virtualization** (#257), pagination, multi-select/checkbox column. |
| DescriptionList (§2) | **Formalize** | `Findings.jsx` Detail renders a key/value `meta` array — promote that to a component. |
| Drawer / SlideOver (§2/§8) | **New** (precedent) | No overlay drawer exists, but `Findings.jsx` hand-rolls an inline Detail *panel* as a side column — a starting point, not the target. |
| SecretInput (§9) | **Extend `Input`** | `Input` already has `prefix`, `error`, `hint`, `size`. Add a masked/write-only mode. |
| TokenInput / ChipInput (§5) | **Extend** | Compose `Input` + removable `Tag`s for multi-value entry; add validation. |
| ChangeBadge, GapBadge, ExposureBadge, StalenessBadge, AvailabilityBadge | **Extend `Badge` shape, new tones** | `Badge` gives the mono-chip *form* (tones `neutral/accent/ok/warn/danger`, `solid`). Reuse the shape — but these need **their own token palette**, because Badge's current tones are the semantic/severity colors these must *not* borrow. |
| TrendSparkline (§7) | **Formalize** | `Dashboard.jsx` already has a `Sparkline` (inline SVG). |
| Distribution bars / CoverageMeter starting point (§4) | **Formalize** | `Dashboard.jsx` already has `SevBars` (inline SVG severity distribution). |
| Menu / Dropdown (§11) | **Partly exists** | `Select` covers option-picking; a general popover menu is still New. |
| Pagination (§2/§11) | **New** | Inventory filters client-side; no pagination component. |

**One anti-pattern to replace, not restyle:** `Findings.jsx` currently implements a triage-status workflow — `Acknowledge` / `Mark resolved`, a `status` Badge, and a "resolved by admin" line. That is exactly the pattern §3 and §12 rule out (and the "by admin" attribution violates the no-author rule, ADR-0073). The redesign should replace it with the **AnnotationControl** (§3), not carry it forward.

Everything else in §1–§11 not named above is genuinely **New**.

---

## 1. Drift / change surface  — *the biggest gap*

Every serious ASM tool has a "what changed" view; for Verge it's the whole point, and there is currently **no drift screen and no change component**. This is the highest-leverage area.

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **ChangeBadge** | The change vocabulary as chips — `appeared`, `revealed`, `withdrawn`, `descoped`, `returned`, `changed`. **Its own palette, not the severity ramp** (the design-system doc already proposes a `--drift-changed` token). | **Extend `Badge`** (new tones) | **P0** |
| **Timeline** | Vertical time-ordered feed of changes, grouped by `Batch`/run, collapsible per group. | **New** | **P0** |
| **DiffRow / DiffViewer** | Before → after for a single facet value (old value, new value, the transition). | **New** | **P0** |
| **DateRangePicker / time scrubber** | Bound the change view to a window; "since last scan," custom range. | **New** | **P1** |
| **TransitionMarker** | Inline marker on a subject row showing its latest transition without opening the timeline. | **New** | **P1** |

*Domain hooks:* `Span`, `Transition`, `Batch`; closures carry a reason (`descoped` vs. world-moved) — the ChangeBadge must be able to show *why* something left, not just that it did.

---

## 2. Asset inventory — filtering, scale, drill-down

Peers all have heavy inventory tables with faceted filtering, saved segments, and bulk actions. Verge's `Inventory` is a read over the open-`Span` corpus across four subject kinds (`Name`, `Address`, `Service`, `Endpoint`). The kit has only a basic Table. (This area also overlaps closed issues #260/#261 and the open perf issue #257.)

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **DataTable** | Sort, resize, sticky header, **row virtualization** (directly relevant to #257), density toggle. Supersedes the raw Table for large sets. | **Extend `Table`** | **P0** |
| **FilterBar + FacetPanel** | Faceted filtering: subject kind, coverage state, exposure, presence of a Gap, source. | **Compose** (Inventory precedent) | **P0** |
| **FilterChip / token** | Active filters as removable tokens; a filter is a claim you can see and clear. | **Exists — `Tag`** | **P1** |
| **Pagination** | Page/next-prev or "load more" (whichever the redesign favors). | **New** | **P1** |
| **BulkActionBar** | Appears on multi-select; sticky action strip. | **New** | **P1** |
| **SavedView tabs** | Named saved filter sets ("segments"). | **New** (on `Tabs`) | **P2** |
| **ColumnPicker** | Show/hide/reorder columns. | **New** | **P2** |
| **Drawer / SlideOver** | Subject detail without leaving the list — used across inventory, signals, graph. | **New** (panel precedent) | **P0** |
| **DescriptionList** | Key–value detail pairs inside the drawer (mono values per the voice rules). | **Formalize** (Findings `meta`) | **P1** |
| **Breadcrumb** | Drill path: scope → name → address → service → endpoint. | **New** | **P2** |

---

## 3. Signals (not "findings")

Peers ship severity badges (Verge has SeverityBadge) plus triage workflows, evidence panels, and rule references. Verge keeps severity but replaces the triage workflow with a single operator dial — the **Annotation** (accepted risk). See §12 for what *not* to build here.

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **AnnotationControl** | Declare a `(subject, signal)` pair an accepted risk + reason prose. **Not** a status dropdown — it moves a message, never a number, has no state/expiry, and carries no author. | **New** (replaces Findings workflow) | **P0** |
| **EvidencePanel** | The observations/provenance behind a fired signal — what was measured, from which vantage, in which batch. | **New** | **P1** |
| **SignalRuleRef** | Compact reference to the release-coupled rule that fired (its predicate domain / name). | **New** | **P2** |
| **WithdrawnMark** | Marks a signal/subject whose key is in no current population (derived-on-read, no status, no age). | **Extend `Badge`** (new tone) | **P2** |

---

## 4. Coverage & gaps — *Verge-specific, no peer equivalent as strong*

Verge treats "did we even look, and how completely" as first-class: `Coverage` (a census, sometimes with no denominator), `Gap` (absence on a single cell), and staleness / `not-evaluable`. No components exist for any of it, and none of it should reuse severity color.

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **CoverageMeter** | Progress/denominator read for an address scope — **and a distinct "census with no denominator" state** for name scopes and custody extensions. | **New** (`SevBars` a starting point) | **P0** |
| **GapBadge** | Cell-level absence indicator ("absence is a property of a cell, not a row"). Distinct from ChangeBadge and SeverityBadge. | **Extend `Badge`** (new tone) | **P0** |
| **StalenessBadge** | `not-evaluable` / stale-observation / "you stopped telling us" states with the currency bound. | **Extend `Badge`** (new tone) | **P1** |
| **CoverageMessageList** | The coverage-class messages (aperture widened, gained an address, etc.). | **New** | **P2** |

---

## 5. Seeds, proposals & scope onboarding

Peers call this "seeds" / "scope" onboarding. Verge's is richer and opinionated: `Seed` (name or address scope), `Proposal` (registry-suggested, confirm-one-at-a-time / decline-many), exclusions, and the custody-extension toggle. Closed issues #259/#260 (bulk add, pagination) live here — the capability is core even though those tickets are shelved.

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **TokenInput / ChipInput** | Enter many domains / CIDRs with per-token validation (family-agnostic CIDR, the 1,024-address cap). | **Extend** (`Input` + `Tag`) | **P0** |
| **RefusalCallout** | A declaration refusal **names the reachable set and never auto-corrects** — an inline callout distinct from a normal field error. | **New** | **P1** |
| **ProposalReview** | Registry proposals: confirm singularly, decline in bulk — the two acts are deliberately asymmetric, so the UI must be too. | **New** | **P1** |
| **ExclusionEditor** | Manage the three exclusion kinds: exact name, subtree, address scope. | **New** | **P1** |
| **CustodyToggle + extension census** | Toggle custody extension; render the recomputed extension read-only (display, never per-address approval). | **New** (on `Switch`) | **P2** |
| **ImportDialog / FileUpload** | CSV/zone-file import for scopes. | **New** (on `Dialog`) | **P2** |

---

## 6. Scans & runs

Peers have scan scheduling, live progress, and logs. Verge's `Scan` is *configured recurring intent* (six kinds: `dns`, `zone`, `ct`, `tls-acceptance`, two port tiers); the executed record is a `Batch`. Naming matters — **avoid "job."** Live output overlaps closed #262; perf overlaps #257.

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **LogViewer / terminal block** | Live streaming batch output — the inverted-ink panel the system already reserves for logs (was #262). | **New** | **P1** |
| **ProgressBar** | Batch progress; determinate + indeterminate. | **New** | **P1** |
| **CadenceSelect** | Recurring cadence input (daily/weekly/monthly + custom) — currency depends on it. | **Extend `Select`** | **P1** |
| **ScanConfigCard** | Per-scan-kind config (scopes, vantages where applicable, port list where it's a connect). | **New** (on `Card`) | **P2** |
| **BatchStatus** | State of a run (scheduled/running/complete/failed) + its recorded scope. | **Extend `Badge`** | **P2** |

---

## 7. Vantages & probers

Peers with distributed scanning expose "scan engines / collectors." Verge has `Vantage` (a network position, incl. its resolver) with a Derived `Availability` and a `Vantage class` (`internet` / `internal` / `unverified`) re-verified each batch. No components exist.

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **VantageCard** | A vantage with its class, availability, and resolver identity; the `unverified` state is a real, distinct state (no prober → no exposure claims). | **New** (on `Card`) | **P1** |
| **AvailabilityBadge** | The Derived availability state. | **Extend `Badge`** | **P2** |
| **ExposureBadge / ReachBadge** | `exposed` / `firewalled` / `not-reached` / `unverified` — a **state chip, explicitly not a 0–100 score** (see §12). | **Extend `Badge`** (new tones) | **P1** |

---

## 8. Graph / relationship view

The "attack surface graph" is table-stakes for peers; Verge has a GraphView *screen* but no reusable graph components. Edges here are citations/resolutions between the four subject kinds.

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **GraphCanvas** | Interactive node-link canvas with pan/zoom. | **New** | **P1** |
| **GraphLegend** | Subject-kind and edge-kind key. | **New** | **P2** |
| **GraphControls / Minimap** | Zoom, fit, minimap; filter by subject kind (reuse FacetPanel). | **New** | **P2** |
| *(reuse Drawer)* | Node detail on select. | **Reuse** (§2 Drawer) | — |

---

## 9. Notifications, channels & delivery

Peers have alerting + integrations. Verge has `Channel` (one-way https, secret **write-only**, routes **by class** only), `Delivery` (operational record), and `Message` (one firing of one cause). **Avoid "webhook/integration/endpoint."**

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **SecretInput** | Write-only secret field — accepted, never rendered back (also used for API keys). | **Extend `Input`** | **P0** (security-relevant) |
| **ChannelForm** | URL + secret + the class subset it receives; routing is by class and nothing finer. | **Compose** (existing form parts) | **P1** |
| **NotificationCenter / MessageList** | In-app message inbox (per cause/class). | **New** | **P1** |
| **DeliveryTable** | Delivery attempts/outcomes (operational record). | **Extend `Table`** | **P2** |

---

## 10. Reports & export

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **ExportMenu / SplitButton** | Export CSV / JSON / PDF (imperative labels: `Export CSV`). | **New** (needs Menu, §11) | **P1** |
| **ReportCard** | List/preview of generated reports (Reports screen exists; components don't). | **New** (on `Card`) | **P2** |

---

## 11. Shell, settings & universal states

Cross-cutting components every screen leans on; several directly address the open perf issue **#257**.

| Component | What it does | Status | Priority |
| --- | --- | --- | --- |
| **Skeleton** | Loading placeholders for tables/cards — the honest answer to "the UI feels slow" (#257). | **New** | **P0** |
| **Spinner / inline loading** | Determinate/indeterminate inline load state. | **New** | **P1** |
| **ErrorState** | Sibling to EmptyState for failures (fact + retry). | **New** (mirror `EmptyState`) | **P1** |
| **OrgSwitcher** | Switch between managed client orgs — the **MSP audience** is explicit in the brief. | **New** | **P1** |
| **SettingsNav / SectionList** | Left-nav or sectioned settings shell. | **New** | **P1** |
| **Pagination / LoadMore** | (also §2) | **New** | **P1** |
| **CommandPalette** | Keyboard-first quick nav — fits the engineer/dev-tool audience. | **New** | **P2** |
| **SearchInput (typeahead)** | Global search with suggestions. | **Extend `Input`** | **P2** |
| **Menu / Dropdown** | General popover menu primitive (currently only Select exists). | **New** | **P1** |
| **Breadcrumb** | (also §2) | **New** | **P2** |

---

## 12. Deliberately DON'T build these

Standard peer features that contradict Verge's model. Building them creates UI the backend will never honor.

| Peer feature | Why it doesn't fit Verge | Build instead |
| --- | --- | --- |
| **Technology fingerprinting / tech-stack profile** | Explicitly out of scope on drift-integrity grounds (readme + `design-system.md`). No "detected technologies" panel, no per-tech CVE matcher framed as fingerprint. | Nothing — it's a non-goal. |
| **Aggregate risk score / security rating (A–F, 0–100)** | Verge concludes `Exposure` across two `Reach` legs; it is not a rating vendor. A gauge would invent a number the model doesn't produce. | **ExposureBadge / ReachBadge** state chips (§7). |
| **Triage/status workflow** (open→in-progress→resolved, assignee, board) | The glossary rejects `triage state`, `finding state`, `suppression`, `risk acceptance workflow`. The only operator dial is the `Annotation`, which moves a message, not a number, and has no state. | **AnnotationControl** (§3). |
| **Per-user attribution / audit-by-user / RBAC roles** | ADR-0073: an operator act carries **no author**; no act is written down with an actor. Assignee avatars and "changed by X" trails have nothing to bind to. | Unattributed dials; if multi-tenant is needed, that's **OrgSwitcher** (org scope), not per-user identity. |
| **Timed mute / suppression with expiry** | An `Annotation` never lapses — no expiry, no status, no timeline. A returning subject re-mutes itself with no operator act. | The plain **AnnotationControl**; no countdown/expiry UI. |

*If any of these turns out to be wanted after all, it's a domain decision (an ADR), not a component decision — flag it before building.*

---

## 13. Naming cheat-sheet (from the glossary's `_Avoid_` lines)

| Use | Not |
| --- | --- |
| Signal | finding |
| Seed / scope | target, root domain |
| Name | domain, subdomain, hostname, host |
| Address | IP, host, node |
| Service | port, open port, socket |
| Endpoint | URL, site, web asset, vhost |
| Subject | asset*, entity, target |
| Scan | job, scan job |
| Batch | run* (informal ok), scan result |
| Channel | webhook, integration, endpoint, sink |
| Proposal | suggestion, candidate range, pending seed |
| Annotation | status, triage state, suppression, exception |
| Vantage | prober location, scanner, agent |

\* `asset` is allowed only as a UI collective noun; `run` is fine informally but the modeled object is `Batch`.

---

## Suggested build order

1. **Drift set** (§1: ChangeBadge, Timeline, DiffRow) — the missing thesis surface, and the design decision most likely to shape the new palette (the change vocabulary needs its own colors).
2. **Table + states** (§2 DataTable/Drawer, §11 Skeleton/ErrorState) — unblocks every screen and is the concrete answer to #257.
3. **Coverage/Gap set** (§4) — Verge's real differentiator; nothing off-the-shelf covers it.
4. **Seeds/Proposals** (§5) and **Signals/Annotation** (§3) — the two richest interaction areas.
5. Everything else as screens demand it.
