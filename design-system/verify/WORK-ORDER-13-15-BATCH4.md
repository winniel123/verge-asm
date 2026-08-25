# Work order — batch 4: AssetDetail + SubjectDetail + Graph (map #13–#15, package v3.10.0)

Use the consume-design-package skill. Fourth parallel batch: up to 3 sessions, one screen each, one branch each, all pinned to v3.10.0, shared files append-only, PRs merge serially with G1+G2 re-runs (WAYFINDER-MAP §Batching). Screens 1–12 are LANDED — do not touch. All three tmpls call the landed `sevbadge` define (signals.tmpl); asset.tmpl also calls `changeglyph` (drift.tmpl) and subjectdetail.tmpl calls `recordrows` (inventory.tmpl) — one parse set, nothing new to land for those.

## Screen 13 — AssetDetail (`templates/asset.tmpl` replaces templates_asset.go)

Defines kept: "asset" + "assetexposure". Holes kept: `.Asset{Key,Type,Severity,Exposure,Seen,InScopeSince,Withdrawn,Ports[],DNS[],Provenance[],Signals[],Drift[]}`. Changed (SPEC-CHANGE #22, ruled):
- `.SevLabel` joins every severity read (header + signals rows; batch-3 precedent).
- `.Signals[]` gains `.SigID` + `.Time`; rows deep-link `/signals?view={id}` (#22b — the mock's VG-#### ids predate the SIG-#### mint; render the real id).
- `.Cert` (nullable `{Name,Issuer,Algorithm,NotAfter,Label,Tone(ok|warn|danger),Fingerprint}`) — **BUILD the parsed-certificate read** off the certificate-chain leaf (#22c): the facet already stores issuer + expiry ("CN=R11 · Let's Encrypt", "exp 2026-11-02"); parse and store identity at fold time (issuer, algorithm, not-after; fingerprint of the leaf). Label/Tone precomputed ("valid · 47d" ok / "expires in Nd" warn ≤30d / "expired Nd ago" danger). Null → the honest empty state stays.
- `.Ports[].Transport` becomes `.Service` — a precomputed display string joining the transport with the http-identity Server/banner where an endpoint on that port holds one (#22d — a read of stored evidence, not a new fingerprint; transport-only where none exists).
- Header gains the spec more-menu (view JS): Annotate → /signals · Descope asset → /scope. "Add tag" is dropped — no tag store exists.
- Drift trail renders the spec collapsible group ("This asset · last 30 days", count pill, rail events); copy affordances get the hover copy control.

## Screen 14 — SubjectDetail (`templates/subjectdetail.tmpl` replaces templates_subjectdetail.go)

Defines kept: "service" + "endpoint" + "subjectstyle"/"subjectbreadcrumb"/"subjecttimelines"/"subjectrules"/"subjectprovenance" (+ new "subjectmenu"/"subjectjs"/"subjectcitation"). Holes kept as shipped (see tmpl header). Changed (#22):
- `.Rules[]` gains `.SevLabel`; rules rows link /signals.
- `.Closed[]` spans gain `.OpenedFull`/`.ClosedFull` (ISO title tooltips — the spec's RelativeTime pattern).
- `.Service`/`.Endpoint` gain `.CopyKey` (the spec's copyable key string, e.g. "203.0.113.7:5900 tcp").
- Service `.Signals[]` gains `.SigID` + `.Time` and deep-links (#22b).
- Withdrawn renders the spec dashed WithdrawnMark chip + the neutral "Withdrawn by the world" Banner (replacing .chip.loss + .notice); the ReachGap notice and Break render as spec banners (Break carries the git-branch icon, accent tone).
- The citation chain renders the spec rail (dots + connector, Seed dot accent); current facets render the spec sunken KeyValueList (Verdict/Since always present on service — "—" when empty; endpoint Redirect shows "— (recorded, not followed)" when none). The details/summary span-record expansion inside timeline values stays (calls landed "recordrows").
- The more-menu is the spec dropdown (view JS): Annotate · Descope address/name.

## Screen 15 — Graph (`templates/graph.tmpl` replaces templates_graph.go)

Define "graph" kept. Holes: `.Graph{Empty,ViewW,ViewH,MiniW,MiniH,Edges[],Nodes[]}`. Changed (#22):
- `.Nodes[].Type` gains the **domain | subdomain** split (#22e): the apex of a name scope → domain (r16, ink stroke sw2, 600 13px ink label); other names → subdomain (r10, neutral-500 stroke). The legend names both (was "name").
- `.HaloR` splits into `.HaloA` (r+7, fill 0.12) + `.HaloB` (r+4.5, ring sw2 0.65) (#22f).
- `.LabelDX` = type radius + 9. `.OpenSignals[]` carries `{Severity,SevLabel,Rule,Subject}` (renders via the landed sevbadge).
- The severity filter is the spec listbox (view JS in tmpl; native select drops); all nodes stay drawn — halos + minimap dots light per level. Node select draws the spec focus ring (class `sel` on the gnode). The drawer renders the spec Drawer (scrim rgba(21,18,15,0.28), 420px slide-in panel, sunken 1-column KV, sev-badged signal rows via the pre-rendered #gr-signal-data block — mechanism kept). Export PNG button keeps the serializer.
- The pan/zoom/minimap engine is the landed port of Graph.jsx — behavior byte-kept (clamp 0.5–2.5, wheel 1.15/0.87, buttons 1.25/0.8, drag threshold 3px).

## Fixtures (fixtures.json → asset / subjectdetail / graph)

- asset: edge-gw-03.acmecorp.io — 3 ports (Service strings joined per #22d), 3 DNS records, cert (valid · 47d, ECDSA-SHA256, R11, pinned fingerprint), 5 provenance keys, 1 signal (SIG-1042 · 4m), 4 drift events.
- subjectdetail: service 203.0.113.7:5900/tcp (chain → Seed, reached since 14:00Z, break at 2026-08-01 on transport, 2 closed spans incl. a Gap, vnc-exposure fired); service_withdrawn 203.0.113.29:8080/tcp (jenkins — closed timeline, ground withdrawn); endpoint edge-gw-03 · :443 https (200 · nginx/1.25.0, one closed span, verbose-server-header fired).
- graph: the 26-node topology (1 domain + 10 subdomains + 8 addresses + 7 services), edges, minimap coords at 110/1200 scale, per-node ports/first/open-signals (4 detailed nodes).

## Goldens (crop `main`; graph node-drawer crops `body` · × light/dark @1440)

asset: default (/asset/edge-gw-03.acmecorp.io). subjectdetail: service · endpoint · service-withdrawn. graph: default · node-drawer (click edge-gw-03) · filtered-critical.

## Acceptance

Byte-served from the three tmpls; no repo-authored markup/CSS remains for these routes; the certificate card renders parsed identity (and the honest empty when a subject holds none); signals-here rows open the signals drawer; drift-trail group collapses; subject menus open; withdrawn states render the spec banner + mark; timeline value expansions still work; graph pans/zooms, filter lights halos per level, node click opens the spec drawer with the fired-rule list, Escape closes; G1+G2 green; SPEC-CHANGE gains no silent workarounds.
