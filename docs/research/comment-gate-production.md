# Comment policy validation gate — production Go, round 1

SPEC `docs/spec/comment-policy.md` §3.9. Regenerate this sheet with:

```sh
go run ./cmd/commentlint sample --population production --round 1
```

- In-scope Go files read: 255
- Blocks the §3.2 screen admits for deletion: 1978
- Blocks drawn into the gate sample: 100
- Blocks drawn into the coverage supplement: 0

Accept a class at 2 or fewer load-bearing blocks. A class that fails three rounds leaves the v1 delete set and stays in the flag set.

## Class coverage

| Class | Admitted | Gate sample | Supplement |
| --- | --- | --- | --- |
| `commented-out-code` | 0 | 0 | 0 |
| `docstring-exported-conventional` | 827 | 48 | 0 |
| `docstring-unexported` | 1032 | 48 | 0 |
| `section-divider` | 88 | 3 | 0 |
| `short-label` | 31 | 1 | 0 |

## Verdicts

A reviewer fills the last two columns. A class that admits no block on this population reads `n/a`.

| Class | Read | Load-bearing | Verdict |
| --- | --- | --- | --- |
| `commented-out-code` | 0 | n/a | n/a |
| `docstring-exported-conventional` | 48 | | |
| `docstring-unexported` | 48 | | |
| `section-divider` | 3 | | |
| `short-label` | 1 | | |

## Gate sample

### 1. `cmd/web/annotations.go:125-127` — `docstring-unexported`

Load-bearing: [ ]

```go
122 | 	s.redirectBack(w, r, "/signals")
123 | }
124 | 
125 | // knownRule reports whether name is a shipped rule. An Annotation may only name a
126 | // rule that exists — accepting a firing on a signal that can never fire would be
127 | // an acceptance with no reader.
128 | func knownRule(name string) bool {
129 | 	for _, n := range signal.RuleNames() {
130 | 		if n == name {
```

### 2. `cmd/web/api_v1.go:222` — `section-divider`

Load-bearing: [ ]

```go
219 | 	writeAPIJSON(w, out)
220 | }
221 | 
222 | // --- drift ------------------------------------------------------------------
223 | 
224 | type apiDriftResponse struct {
225 | 	Period          string          `json:"period"`
```

### 3. `cmd/web/auth.go:257-259` — `docstring-unexported`

Load-bearing: [ ]

```go
254 | 	Mark string
255 | }
256 | 
257 | // ssoMark derives the login button's mono mark from a provider name: the first letter of up to
258 | // the first two words, uppercased (e.g. "Okta" → "O", "Acme Corp" → "AC"). An empty name yields
259 | // an empty mark, which the tmpl simply renders as a blank chip.
260 | func ssoMark(name string) string {
261 | 	mark := ""
262 | 	for _, field := range strings.Fields(name) {
```

### 4. `cmd/web/auth.go:1343-1348` — `docstring-unexported`

Load-bearing: [ ]

```go
1340 | 	s.render(w, r, "forgot", s.signinData(map[string]any{"Title": "Reset password"}))
1341 | }
1342 | 
1343 | // forgotSubmit mints a single-use reset link for the named account, then always
1344 | // renders the same "if that account exists, a link is on its way" done state — the
1345 | // response is identical whether or not the username exists, so the endpoint reveals
1346 | // nothing about which accounts do. There is no mail on a self-hosted host, so the
1347 | // link is delivered out of band: it is written to the web logs, exactly as the
1348 | // first-boot setup token is, and the operator can also reset directly on the host.
1349 | func (s *server) forgotSubmit(w http.ResponseWriter, r *http.Request) {
1350 | 	username := strings.TrimSpace(r.FormValue("username"))
1351 | 	if acct, err := s.store.GetAccountByUsername(r.Context(), username); err == nil {
```

### 5. `cmd/web/clientip.go:55` — `docstring-unexported`

Load-bearing: [ ]

```go
52 | 	return tp, nil
53 | }
54 | 
55 | // trusts reports whether addr falls inside any configured trusted-proxy range.
56 | func (tp trustedProxies) trusts(addr netip.Addr) bool {
57 | 	addr = addr.Unmap()
58 | 	for _, n := range tp.nets {
```

### 6. `cmd/web/cold.go:575-576` — `docstring-unexported`

Load-bearing: [ ]

```go
572 | 	return out
573 | }
574 | 
575 | // sortCoverageMessages orders the messages deterministically — by kind, then
576 | // subject — so the currency list is stable across loads.
577 | func sortCoverageMessages(msgs []coverageMessageView) {
578 | 	sort.SliceStable(msgs, func(i, j int) bool {
579 | 		if msgs[i].Kind != msgs[j].Kind {
```

### 7. `cmd/web/devfixtures.go:38-43` — `docstring-unexported`

Load-bearing: [ ]

```go
35 | // build keeps the crypto/rand id (newIncidentID, errors.go).
36 | const devFixtureIncidentID = "err_9f3ka72c"
37 | 
38 | // devFixtureAccount is one design fixture login the harness signs in as. states.json
39 | // carries a per-state `session` ("admin" | "viewer"); the dev session mint resolves that
40 | // role to one of these accounts and opens a session before the state's route is captured.
41 | // Dev-only — well-known passwords, seeded only under VERGE_DEV, mirroring fixtures.json →
42 | // accounts (asserted by TestDevFixturesMatchPackage). The password is #nosec G101: not a
43 | // real credential, only ever seeded into a throwaway dev database.
44 | type devFixtureAccount struct {
45 | 	username string
46 | 	role     string
```

### 8. `cmd/web/devfixtures.go:1591-1593` — `docstring-unexported`

Load-bearing: [ ]

```go
1588 | 	return hist
1589 | }
1590 | 
1591 | // signalsRowMap shapes one fixture row for the table (the holes the frozen tmpl's row reads),
1592 | // carrying the per-row descope link (filters preserved) and the withdrawn flag (the withdrawn tab
1593 | // draws the mark).
1594 | func signalsRowMap(row devSignalRow, closeHref string, withdrawn bool) map[string]any {
1595 | 	return map[string]any{
1596 | 		"Severity":    row.Severity,
```

### 9. `cmd/web/devfixtures.go:1812-1814` — `docstring-unexported`

Load-bearing: [ ]

```go
1809 | // devDashUnavailable is fixtures.json dashboard.unavailable: the missed-check vantage the banner names.
1810 | var devDashUnavailable = []string{"ap-south-1"}
1811 | 
1812 | // devDashStat mirrors one fixtures.json dashboard.stat_band cell. liveWhenScanning is the JSON's
1813 | // live_when_scanning flag — the pulse shows only while a scan is running, so Live is set from it AND
1814 | // the active `scanning` variant, never on the resting default.
1815 | type devDashStat struct {
1816 | 	label            string
1817 | 	value            string
```

### 10. `cmd/web/diskstat_other.go:5-7` — `docstring-unexported`

Load-bearing: [ ]

```go
 2 | 
 3 | package main
 4 | 
 5 | // diskUsage has no portable implementation off the unix deployment target (dev on
 6 | // Windows), so it reports ok=false and the instance-health disk figure collapses rather
 7 | // than fabricate one — the same honest degradation the rest of the health tab uses.
 8 | func diskUsage(string) (used, total uint64, ok bool) {
 9 | 	return 0, 0, false
10 | }
```

### 11. `cmd/web/exposure.go:259-261` — `docstring-unexported`

Load-bearing: [ ]

```go
256 | 	return assetExposure(l.outcome, l.isGap)
257 | }
258 | 
259 | // legFrom lifts a by-class span into the internal/exposure engine's Leg: a decided
260 | // connection outcome is a valued leg, a Gap is a silent (stopped-looking) leg, and
261 | // an unmeasured class was never configured. Only a valued leg feeds Project.
262 | func legFrom(l legInfo) exposure.Leg {
263 | 	if !l.present {
264 | 		return exposure.Leg{Status: exposure.LegNeverConfigured}
```

### 12. `cmd/web/graph.go:225` — `docstring-unexported`

Load-bearing: [ ]

```go
222 | // Seed (ADR-0136 §3).
223 | const graphScopeAll = "all"
224 | 
225 | // graphScopeAllLabel is the whole-estate entry's label in the selector.
226 | const graphScopeAllLabel = "Whole estate"
227 | 
228 | // graphScope is one entry of the graph's scope selector: the ?scope token the URL
```

### 13. `cmd/web/inventory.go:653-659` — `docstring-unexported`

Load-bearing: [ ]

```go
650 | 	}
651 | }
652 | 
653 | // devInventoryGroupTotals pins design-system/fixtures/fixtures.json → inventory
654 | // group totals that differ from the seeded row count, for the VERGE_DEV pixel-parity
655 | // capture. Only the address group differs: the fixture declares 41 (an estate-scale
656 | // count) while the loader seeds 3 rows, so the golden's count badge reads "41" and
657 | // its expander "Show all 41 — 38 more". The other groups' totals equal their seeded
658 | // counts, so they need no override. TestInventoryFixtureCountsMatchPackage folds
659 | // these back through the frozen package — the byte-exactness gate before the pixels.
660 | var devInventoryGroupTotals = map[string]int{
661 | 	"address": 41,
662 | }
```

### 14. `cmd/web/probers.go:47-54` — `docstring-unexported`

Load-bearing: [ ]

```go
44 | 	At                 string
45 | }
46 | 
47 | // provisionProber declares a prober: the operator supplies host, port and a
48 | // non-root username, and the row created here DECLARES "this vantage is on the
49 | // internet" (CONTEXT.md "Vantage class") — there is no network_position field
50 | // and no setup-wizard step. It is reached only through requireAdmin.
51 | //
52 | // No key material is generated here. The worker owns the SSH keypair volume and
53 | // generates the pair out of band, publishing only the public half back to this
54 | // row; web never touches a private key.
55 | func (s *server) provisionProber(w http.ResponseWriter, r *http.Request, acct db.Account) {
56 | 	host := r.FormValue("host")
57 | 	port := r.FormValue("port")
```

### 15. `cmd/web/progress.go:59-61` — `docstring-unexported`

Load-bearing: [ ]

```go
56 | }
57 | 
58 | const (
59 | 	// maxProgressRuns bounds how many dispatches the hub retains events for at once. Events
60 | 	// matter only while a run is live and re-derive to bare state at any time, so a small ring
61 | 	// is ample; the least-recently-started run is evicted past the cap.
62 | 	maxProgressRuns = 256
63 | 	// maxEventsPerRun caps one run's event log. The stream's cursor is an index into this log,
64 | 	// so events are FROZEN at the cap (further events dropped) rather than evicted from the
```

### 16. `cmd/web/reports.go:385-387` — `docstring-unexported`

Load-bearing: [ ]

```go
382 | 	return strconv.Itoa(n) + " assets"
383 | }
384 | 
385 | // reportsDurationDays renders a duration as the console's terse day figure ("2.4d"),
386 | // the form the mean-time-to-withdrawal KPI reads (Reports.jsx). A sub-day mean still
387 | // renders in days ("0.4d") so the KPI keeps one unit.
388 | func reportsDurationDays(d time.Duration) string {
389 | 	return strconv.FormatFloat(d.Hours()/24, 'f', 1, 64) + "d"
390 | }
```

### 17. `cmd/web/reports.go:651-654` — `docstring-unexported`

Load-bearing: [ ]

```go
648 | 	Data  []int  `json:"data"`
649 | }
650 | 
651 | // buildReportsTimeSeries folds the signals-over-time buckets into the chart geometry,
652 | // choosing a nice y-axis step exactly as TimeSeriesChart.jsx does. ok is false where
653 | // the standing level never rises above zero, so the card renders the design's empty
654 | // pattern rather than a flat line on a fabricated axis.
655 | func buildReportsTimeSeries(pts []drift.SignalPoint) (reportsTimeSeries, bool) {
656 | 	n := len(pts)
657 | 	if n < 2 {
```

### 18. `cmd/web/reports_schedule.go:100-103` — `docstring-unexported`

Load-bearing: [ ]

```go
 97 | 	reportScheduleLast         = 3
 98 | )
 99 | 
100 | // scheduleWizardView is the controlled state of the wizard across the post-back
101 | // flow: the step being shown, the schedule id (0 on create, the target on edit), and
102 | // every field's current value. Sections are the checked section keys in canonical
103 | // order.
104 | type scheduleWizardView struct {
105 | 	Step     int
106 | 	ID       int64
```

### 19. `cmd/web/reports_schedule.go:334-338` — `docstring-unexported`

Load-bearing: [ ]

```go
331 | 	s.renderScheduleWizard(r.Context(), w, r, acct, v, false)
332 | }
333 | 
334 | // editReportScheduleWizard renders the wizard prefilled from an existing schedule. A
335 | // fresh GET (no ?step) prefills from the stored row; a PRG GET (?step=N&…) reconstructs
336 | // the accumulated state and renders that step. A stale id (already deleted) redirects
337 | // back to /reports rather than 500ing. Stepping and finishing post to
338 | // /reports/schedule/{id}/edit (editReportSchedule).
339 | func (s *server) editReportScheduleWizard(w http.ResponseWriter, r *http.Request, acct db.Account) {
340 | 	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
341 | 	if err != nil {
```

### 20. `cmd/web/scans.go:467-469` — `docstring-unexported`

Load-bearing: [ ]

```go
464 | 	K, V string
465 | }
466 | 
467 | // runVantage is one vantage's health in this run: the vantage that looked, a
468 | // latency (not stored, so "—"), and a status folded from its jobs (degraded if any
469 | // dead-lettered, else ok). It is a vantage, never a probe/scanner/agent.
470 | type runVantage struct {
471 | 	Name    string
472 | 	Latency string
```

### 21. `cmd/web/scans.go:506-507` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
503 | 	Vantages    []runVantage
504 | 	// Degraded is nullable (#20): a *runDegraded, nil where no vantage fell short.
505 | 	Degraded *runDegraded
506 | 	// JobFilter is nullable (DF-F3b): set when the request carries ?job={id}; .Log has
507 | 	// already been narrowed to that job's rows server-side by the time it renders.
508 | 	JobFilter *runJobFilter
509 | 	// StreamHref is the per-job stdout long-poll endpoint the frozen rundetail.tmpl
510 | 	// tails while a job is running (R4-D7 #761). It is nullable — empty when no job is
```

### 22. `cmd/web/scans.go:1051-1054` — `docstring-unexported`

Load-bearing: [ ]

```go
1048 | 	return stages
1049 | }
1050 | 
1051 | // runLog turns the dispatch's jobs into the batch log — one line per job, the id
1052 | // as its tag, a level from its state (a dead job errors, a superseded or retrying
1053 | // attempt warns), and the terse kind · state · vantage · batch text. Every line is
1054 | // a real queue event; nothing is invented.
1055 | func runLog(jobs []jobView) []runLogLine {
1056 | 	out := make([]runLogLine, 0, len(jobs))
1057 | 	for _, j := range jobs {
```

### 23. `cmd/web/search.go:60-62` — `docstring-unexported`

Load-bearing: [ ]

```go
57 | // signals.go) plus the shell's "head"/"chrome"/"foot" — all resolve at execute time.
58 | var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/search.tmpl"))
59 | 
60 | // hiSeg is one run of a matched text field: its literal text and whether it is the
61 | // highlighted (query-matched) run. The "hisegs" define wraps a hit seg in the accent
62 | // span and emits an un-hit seg as plain (auto-escaped) text.
63 | type hiSeg struct {
64 | 	Text string
65 | 	Hit  bool
```

### 24. `cmd/web/seedfixtures.go:150-154` — `docstring-unexported`

Load-bearing: [ ]

```go
147 | 	return nil
148 | }
149 | 
150 | // seedDevOperator makes the fixed dev operator exist so the pixel-parity harness can
151 | // log in and reach the authenticated Inventory screen. It is idempotent: it creates
152 | // the operator only when the instance has no account yet, so a re-seed against an
153 | // instance that already has one (the operator, or a real first-run admin) leaves
154 | // accounts untouched. Dev-only — the caller (main.go) has already gated on VERGE_DEV.
155 | func seedDevOperator(ctx context.Context, pool *pgxpool.Pool) error {
156 | 	q := db.New(pool)
157 | 	n, err := q.CountAccounts(ctx)
```

### 25. `cmd/web/seeds.go:60-64` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
57 | 	ID        int64
58 | 	IsAddress bool
59 | 	Scope     string
60 | 	// Anchor is the row's in-page id — the seed-scoped fragment an
61 | 	// aperture-widening message links to so it lands on the exact Seed whose
62 | 	// scope moved, not merely the Seeds list (v1 spec §5.3). Built from Scope by
63 | 	// seedAnchor, which the message renderer uses for the same key so the two
64 | 	// agree.
65 | 	Anchor string
66 | 	By     string
67 | 	At     string
```

### 26. `cmd/web/seeds.go:89-91` — `docstring-unexported`

Load-bearing: [ ]

```go
86 | 	exclError, exclKind, exclValue string
87 | 	custodyError                   string
88 | 	zoneIntervalError              string
89 | 	// zoneErrors are the per-file zone-upload refusals (DF-F2): one row per rejected
90 | 	// file, in upload order. It replaces the single zoneError string — a bulk upload
91 | 	// refuses each file independently.
92 | 	zoneErrors []zoneErrorView
93 | 	// zoneIntervalDays echoes a rejected interval so the admin need not retype
94 | 	// it; empty means render the stored dial.
```

### 27. `cmd/web/seeds.go:114-116` — `docstring-unexported`

Load-bearing: [ ]

```go
111 | 	refusals []refusalView
112 | }
113 | 
114 | // zoneErrorView is one per-file zone-upload refusal (DF-F2): the file's name and the
115 | // reason it was refused (apex outside the name scopes, or not a zone file). It replaces
116 | // the single .ZoneError hole so a bulk upload lists a row per rejected file.
117 | type zoneErrorView struct {
118 | 	File   string
119 | 	Reason string
```

### 28. `cmd/web/seeds.go:547-550` — `docstring-unexported`

Load-bearing: [ ]

```go
544 | 		"leave the estate on the next completed job."
545 | }
546 | 
547 | // seedScopeByID returns the display scope for a declared seed id — the address CIDR for
548 | // an address scope, the domain for a name scope — and whether it is an address scope,
549 | // or "" when no such seed exists. It reuses toSeedViews so the string matches the chip
550 | // the operator clicked.
551 | func (s *server) seedScopeByID(r *http.Request, id int64) (string, bool) {
552 | 	rows, err := s.store.ListSeeds(r.Context())
553 | 	if err != nil {
```

### 29. `cmd/web/settings.go:809-810` — `docstring-unexported`

Load-bearing: [ ]

```go
806 | 	s.backToSection(w, r, "channels")
807 | }
808 | 
809 | // deleteChannel removes a channel. It is idempotent: deleting a row that is
810 | // already gone satisfies the operator's intent either way.
811 | func (s *server) deleteChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
812 | 	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
813 | 	if err != nil {
```

### 30. `cmd/web/settings_fixtures.go:421` — `docstring-unexported`

Load-bearing: [ ]

```go
418 | 	API          sfAPI             `json:"api"`
419 | }
420 | 
421 | // loadSettingsFixture reads and decodes the fixtures.json → settings slice.
422 | func loadSettingsFixture() (settingsFixture, error) {
423 | 	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
424 | 	if err != nil {
```

### 31. `cmd/web/settings_sso.go:25-26` — `docstring-unexported`

Load-bearing: [ ]

```go
22 | // (/login/sso/<slug>), so it is lowercase alphanumeric with internal hyphens.
23 | var ssoSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
24 | 
25 | // ssoProviderView is one configured provider shaped for the Settings table: its
26 | // display fields, whether a secret is set (never the value), and who declared it.
27 | type ssoProviderView struct {
28 | 	ID        int64
29 | 	Slug      string
```

### 32. `cmd/web/signals.go:45-47` — `docstring-unexported`

Load-bearing: [ ]

```go
42 | // owns every verdict, and deriveSignalInstances turns the fired census into the
43 | // flat rows this screen draws.
44 | 
45 | // dnsRecordValue is the JSON payload of a dns-record observation (the shape the
46 | // resolution-walk leaf emits). The handler reads only the CNAME target off the
47 | // CNAME discriminator and the delegation's Lame verdict off the NS discriminator.
48 | type dnsRecordValue struct {
49 | 	RRs []struct {
50 | 		Name string `json:"name"`
```

### 33. `cmd/web/signals.go:1682-1686` — `docstring-unexported`

Load-bearing: [ ]

```go
1679 | 	return fmt.Sprintf("SIG-%04d", id)
1680 | }
1681 | 
1682 | // subjectAddrPort pulls the address and port out of a subject key for the
1683 | // per-instance table's IP / Port columns. A Service key is `address:port/transport`
1684 | // and an Endpoint key is `name@address:port/transport`; a bare Name carries
1685 | // neither, so both come back empty. It reuses the same parse the engine's fold
1686 | // uses, so the columns agree with the census's own reading of the key.
1687 | func subjectAddrPort(subject string) (ip, port string) {
1688 | 	_, svc := splitEndpointName(subject)
1689 | 	if p, addr, ok := parseServicePair(svc); ok {
```

### 34. `cmd/web/sources.go:415-417` — `docstring-unexported`

Load-bearing: [ ]

```go
412 | 	FalseEmptyPass bool
413 | }
414 | 
415 | // ctReliabilityBar is the bar's targets, formatted for the KPI tiles (spec §3). It is
416 | // release-authored, the same for every install, so it is derived from the scan-package
417 | // constants rather than read from Postgres.
418 | type ctReliabilityBar struct {
419 | 	SuccessTarget string // "≥ 99%"
420 | 	LatencyTarget string // "≤ 5 s"
```

### 35. `cmd/web/subjects.go:123-125` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
120 | 	// origin from here* rather than *nothing is open*.
121 | 	ReachGap       bool
122 | 	ReachGapReason string
123 | 	// Citation is the "why is this here" chain: Service → Address → the Name whose
124 | 	// resolution cites the Address (or the address-scope Seed that covers it),
125 | 	// terminating at a Seed.
126 | 	Citation           []citationHop
127 | 	CitationTerminated bool
128 | 	// Timelines are the Service's reachability Span timelines — current and closed
```

### 36. `cmd/web/subjects.go:143-145` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
140 | 	// Rules are every rule whose predicate domain includes this Service, each with
141 | 	// its own versioned verdict (fired / did not fire) and the rule's SeverityBadge.
142 | 	Rules []subjectRule
143 | 	// Signals are the rules firing on this Service right now (the rail's "Signals
144 | 	// here"), each carrying its rule's severity — the same fired census the asset
145 | 	// drill-in reads, filtered to this subject's key.
146 | 	Signals []assetSignal
147 | }
148 | 
```

### 37. `cmd/web/subjects.go:209` — `docstring-unexported`

Load-bearing: [ ]

```go
206 | 	Fired    bool
207 | }
208 | 
209 | // subjectPageData is the drill-down view for one Name.
210 | type subjectPageData struct {
211 | 	Name       string
212 | 	Withdrawn  bool
```

### 38. `cmd/web/subjects.go:721-723` — `docstring-unexported`

Load-bearing: [ ]

```go
718 | 	return best
719 | }
720 | 
721 | // currentReachSince returns the open instant of the current reachability span — the
722 | // "Since" the Service's current-facet card carries. Empty where the reachability
723 | // timeline holds no current value (a withdrawn or gapped Service).
724 | func currentReachSince(tls []timelineView) string {
725 | 	for _, tl := range tls {
726 | 		if tl.Facet == "reachability" && tl.Current != nil {
```

### 39. `docs/guides/embed.go:13-16` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
10 | 
11 | import "embed"
12 | 
13 | // FS holds every operator guide (docs/guides/*.md), front-matter and all. The
14 | // Search index parses each file's YAML front-matter (title + description) to
15 | // build the Documentation group; see cmd/web/search.go.
16 | //
17 | //go:embed *.md
18 | var FS embed.FS
19 | 
```

### 40. `internal/auth/key.go:17-23` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
14 | // keyLen is the signing-key length in bytes: 256 bits, matching HMAC-SHA256.
15 | const keyLen = 32
16 | 
17 | // LoadOrCreateKey returns the session signing key held in dir, creating it on
18 | // first boot. The key is generated by web and written to its own volume; it is
19 | // never persisted to Postgres, so a database dump does not disclose it and a
20 | // database restore does not silently rotate it (v1 spec §4.3).
21 | //
22 | // The file is written 0600. dir is created if absent. A key of the wrong
23 | // length is treated as corruption and reported rather than used.
24 | func LoadOrCreateKey(dir string) ([]byte, error) {
25 | 	path := filepath.Join(dir, keyFile)
26 | 
```

### 41. `internal/auth/session.go:24-25` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
21 | const (
22 | 	// KindSession is a completed login: the bearer is authenticated.
23 | 	KindSession Kind = "session"
24 | 	// KindPending is a half-login: the password was correct and a TOTP code
25 | 	// is still owed. It authorises only the TOTP-completion step.
26 | 	KindPending Kind = "totp"
27 | )
28 | 
```

### 42. `internal/auth/session.go:29-31` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
26 | 	KindPending Kind = "totp"
27 | )
28 | 
29 | // Session is the claim carried by a signed cookie. It holds no role: the role
30 | // is read from the account row on every request, so a role change or an
31 | // account deletion takes effect immediately rather than at cookie expiry.
32 | type Session struct {
33 | 	AccountID int64     `json:"aid"`
34 | 	Kind      Kind      `json:"knd"`
```

### 43. `internal/custody/census.go:36-37` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
33 | // product-chosen number in front of the operator as if it were their business
34 | // (ADR-0129 §5, #987).
35 | type ExtensionCensusEntry struct {
36 | 	// Name is the in-zone Name holding the A record — the CITING name, which is
37 | 	// what the operator recognises. It is never the edge's own reverse name.
38 | 	Name string
39 | 	// Address is the edge the record points at, Unmap'ed as every address in this
40 | 	// package is.
```

### 44. `internal/delivery/delivery.go:110-111` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
107 | 	Link string `json:"link,omitempty"`
108 | }
109 | 
110 | // Subject is the fired-at subject as a bare (kind, key) pair — the key the
111 | // receiver would follow the link on, not a rendered label.
112 | type Subject struct {
113 | 	Kind string `json:"kind"`
114 | 	Key  string `json:"key"`
```

### 45. `internal/delivery/runner.go:73-77` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
70 | 	}
71 | }
72 | 
73 | // Resolver resolves a host to its IP addresses. *net.Resolver satisfies it (so
74 | // net.DefaultResolver is the production value); a test supplies a fake to place a host
75 | // in a private range without real DNS. It is exported so SendSigned — the shared
76 | // signed-POST transport the report notify runner also drives — can be handed the same
77 | // resolver the delivery Runner uses.
78 | type Resolver interface {
79 | 	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
80 | }
```

### 46. `internal/drift/drift.go:171` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
168 | 	Reason   ClosureReason // set only on a withdrawal closure
169 | }
170 | 
171 | // Open reports whether the span is the timeline's current one.
172 | func (s Span) Open() bool { return s.ClosedAt.IsZero() }
173 | 
```

### 47. `internal/drift/transition.go:50-51` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
47 | type Kind string
48 | 
49 | const (
50 | 	// KindNone is an ordinary adjacency — a value moved, and the move is recorded
51 | 	// and unnamed.
52 | 	KindNone Kind = ""
53 | 	// KindAppeared is discovery: a subject entering the estate with no prior
54 | 	// membership span to return from. Membership-only.
```

### 48. `internal/drift/trend.go:46-50` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
43 | 
44 | // --- Signals over time -------------------------------------------------------
45 | 
46 | // Raise is one signal instance as the trend fold sees it: the instant the
47 | // (rule, subject) pair was FIRST seen firing (signal_instance.first_seen) and
48 | // whether the rule's severity is elevated — critical or high, the design's
49 | // "Critical + high" series. Severity is the rule's, resolved by the web layer and
50 | // passed as a bool so this core needs no severity import.
51 | type Raise struct {
52 | 	At       time.Time
53 | 	Elevated bool
```

### 49. `internal/drift/trend.go:70-75` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
67 | 	StandingElevated int
68 | }
69 | 
70 | // SignalsOverTime folds a set of signal raises into a per-bucket series over the
71 | // window ending at `now`, oldest bucket first. Incidence (Count/Elevated) counts the
72 | // raises whose first-seen instant fell in the bucket; standing (Standing/
73 | // StandingElevated) accumulates every raise on or before the bucket's close, so a
74 | // signal raised before the window still lifts the standing level it is still part of.
75 | // A nil/empty raise set yields a series of empty buckets, never a fabricated shape.
76 | func SignalsOverTime(raises []Raise, now time.Time, bucket time.Duration, buckets int) []SignalPoint {
77 | 	start := windowStart(now, bucket, buckets)
78 | 	points := make([]SignalPoint, buckets)
```

### 50. `internal/exposure/exposure.go:234-236` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
231 | 	StoppedLooking OneLeggedReason = "stopped-looking"
232 | )
233 | 
234 | // Board is the populated 2×2: the Service list in each Exposure cell. Every
235 | // member is enumerable in full — the board is a census, never a sampled or ranked
236 | // view — so the cells hold lists, not just counts.
237 | type Board struct {
238 | 	Exposed     []string
239 | 	EdgeOnly    []string
```

### 51. `internal/exposure/exposure.go:294-297` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
291 | 	WhatMoved []string
292 | }
293 | 
294 | // Build assembles the Screen from the per-Service snapshot and the install's
295 | // class presence. It runs one pass, routing each Service to the board, the
296 | // one-legged list, or the broken list, and collecting the flagship moves — which
297 | // are read off the internet leg regardless of where the Service lands.
298 | func Build(services []ServiceInput, internetPresent, internalPresent bool) Screen {
299 | 	s := Screen{
300 | 		InternetPresent: internetPresent,
```

### 52. `internal/measure/blanketdiscrim/corpus/harness.go:63-66` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
60 | 	return "sha256:" + hex.EncodeToString(h.Sum(nil))
61 | }
62 | 
63 | // ParamsDigest is the digest of the leaf's declared-parameter set — the
64 | // control-port count and band — the second thing a version bump may be justified
65 | // by. It reflects the SHIPPED parameters (blanketdiscrim.DefaultParams), not the
66 | // corpus's fixed set, so a change to the production count still moves the lock.
67 | func ParamsDigest() string { return blanketdiscrim.DefaultParams().Digest() }
68 | 
69 | // UncoveredMove is one row of golden-corpus.md §9's register: a version bump
```

### 53. `internal/measure/blanketdiscrim/corpus/rows.go:16-17` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
13 | // carried by ParamsDigest, so a change to it still moves the lock.
14 | var ControlPorts = blanketdiscrim.FixedPorts{P: []uint16{50001, 50002, 50003}}
15 | 
16 | // Step is one run of the composed exchange inside a row: one Batch at one Vantage
17 | // over one scope, against one scripted connector and handshaker.
18 | type Step struct {
19 | 	Batch     string
20 | 	Scope     co.Scope
```

### 54. `internal/measure/blanketdiscrim/leaf.go:146-147` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
143 | 	PortBandHigh     uint16 `json:"port_band_high"`
144 | }
145 | 
146 | // DefaultParams is the v1 shipped parameter set (ports.go): a batch-generated set
147 | // of random high ports drawn from the ephemeral band.
148 | func DefaultParams() Params {
149 | 	return Params{
150 | 		ControlPortCount: ControlPortCount,
```

### 55. `internal/measure/blanketdiscrim/ports.go:36-40` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
33 | 	portBandHigh uint16 = 65535
34 | )
35 | 
36 | // PortGen produces one batch's control-port set. The ports are drawn per batch as
37 | // independent samples; they never appear on any timeline (a control port is an
38 | // input to the decision and a value on nothing), so a deterministic generator
39 | // produces byte-identical observations for the golden corpus while production
40 | // draws from crypto/rand.
41 | type PortGen interface {
42 | 	// Ports returns exactly ControlPortCount distinct ports from the dynamic range,
43 | 	// sorted, so the control probe is order-stable within a batch.
```

### 56. `internal/measure/connectoutcome/certcorpus/rows.go:30-31` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
27 | 	Golden       string
28 | }
29 | 
30 | // AllCells is the enumeration the coverage test counts against: every cell of the
31 | // tls-handshake / certificate block must be pinned by at least one row.
32 | var AllCells = []string{
33 | 	// T1 — the value space's three variants (a closed union, ADR-0011)
34 | 	"T1/presented", "T1/tls-refused", "T1/no-tls",
```

### 57. `internal/measure/connectoutcome/certcorpus/rows.go:104` — `section-divider`

Load-bearing: [ ]

```go
101 | 		Golden: "cert_presented_named.ndjson",
102 | 	},
103 | 
104 | 	// ---- T1/tls-refused ----
105 | 	{
106 | 		Cells: []string{"T1/tls-refused"},
107 | 		Claim: "a reached Service whose peer speaks TLS but accepts no candidate we offered records tls-refused — a value, distinct from no-tls, so an SNI-required or SSLv3-only listener is not misfiled as *not a TLS server*",
```

### 58. `internal/measure/connectoutcome/certcorpus/rows.go:435` — `section-divider`

Load-bearing: [ ]

```go
432 | 		Golden: "cert_v3_sig_digest.ndjson",
433 | 	},
434 | 
435 | 	// ---- T6/self-sig-verifies + T6/self-signed-leaf ----
436 | 	{
437 | 		Cells: []string{"T6/self-sig-verifies", "T6/self-signed-leaf"},
438 | 		Claim: "a self-signed leaf carries subject==issuer AND self_sig_verifies=true, the two raw facts selfSignedOf() folds so certificate-self-signed derives a definite yes at read",
```

### 59. `internal/measure/connectoutcome/run.go:26-30` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
23 | 	Addresses    []string      `json:"addresses"`
24 | 	TCPPorts     []uint16      `json:"tcp_ports"`
25 | 	UDPPorts     []uint16      `json:"udp_ports,omitempty"`
26 | 	// Names are the server names the certificate handshake offers as SNI, one
27 | 	// Endpoint per name, for each Service the connect reaches. Empty is the
28 | 	// nameless endpoint — the only mode available on an address-scope Seed where
29 | 	// no name is known yet (CONTEXT.md `Endpoint`). It never affects the connect
30 | 	// targets, only the handshake step composed onto a reached Service.
31 | 	Names   []string      `json:"names,omitempty"`
32 | 	Profile SafetyProfile `json:"profile"`
33 | }
```

### 60. `internal/measure/httpexchange/leaf.go:58` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
55 | 	// BodyCapBytes bounds the body read — 65536 (64 KB). A response longer than
56 | 	// this is truncated to the cap and the identity records that it was.
57 | 	BodyCapBytes int `json:"body_cap_bytes"`
58 | 	// TimeoutMillis bounds the whole exchange — 10000 (10 s).
59 | 	TimeoutMillis int `json:"timeout_millis"`
60 | 	// PerHostReqPerSec is the per-host request rate ceiling — 10 req/s.
61 | 	PerHostReqPerSec int `json:"per_host_req_per_sec"`
```

### 61. `internal/measure/resolutionwalk/leaf.go:104-105` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
101 | 	Addresses []string `json:"addresses,omitempty"`
102 | }
103 | 
104 | // NSStatus is one authority's serves/does-not-serve verdict on the delegation
105 | // walk. It is what a partly-lame delegation records instead of Lame (M1.5).
106 | type NSStatus struct {
107 | 	Server string `json:"server"`
108 | 	Serves bool   `json:"serves"`
```

### 62. `internal/measure/resolutionwalk/leaf.go:127-129` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
124 | 	RRs   []RR  `json:"rrs"`
125 | }
126 | 
127 | // Result is the leaf's complete decision for one Name at one Vantage. It names
128 | // no transition (golden-corpus.md R.1): the leaf emits outcomes, and whether a
129 | // subject appeared, withdrew or returned is decided downstream from them.
130 | type Result struct {
131 | 	Name       string     `json:"name"`
132 | 	Resolution Resolution `json:"resolution"`
```

### 63. `internal/measure/resolutionwalk/run.go:11` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 8 | 	"github.com/winniel123/verge-asm/internal/wire"
 9 | )
10 | 
11 | // Kind is the JobSpec.Kind that dispatches to this leaf.
12 | const Kind = "resolution-walk"
13 | 
14 | // Scope is the resolution-walk-specific payload of a JobSpec. It carries the
```

### 64. `internal/measure/wildcarddiscrim/corpus/rows.go:47` — `short-label`

Load-bearing: [ ]

```go
44 | 	"W3b/none-determinate", "W3b/determinate-differing",
45 | 	"W3c/no-wildcard", "W3c/incomplete",
46 | 	"W3d/discriminated", "W3d/shadowed-all",
47 | 	// W4 — control-label set (2)
48 | 	"W4/set-shape", "W4/independent",
49 | 	// W5 — the shared path (1)
50 | 	"W5/shared-path",
```

### 65. `internal/measure/wildcarddiscrim/corpus/script.go:52` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
49 | 	Rules []scriptRule
50 | }
51 | 
52 | // Exchange implements resolutionwalk.Peer.
53 | func (s ScriptPeer) Exchange(q rw.Query) rw.Msg {
54 | 	if q.Path != rw.PathDeclared {
55 | 		// The leaf and its candidate resolution use only the declared path here;
```

### 66. `internal/measure/wildcarddiscrim/emit.go:66-69` — `docstring-unexported`

Load-bearing: [ ]

```go
63 | 	return obs
64 | }
65 | 
66 | // resolutionFor composes the resolution value: `Shadowed` cites nothing, `Gap`
67 | // cites nothing, and a not-`Shadowed` verdict passes resolution-walk's own value
68 | // through unchanged — the membership-deciding value is one or the other and the
69 | // leaf that decided it is this one exactly when the value is Shadowed or Gap.
70 | func resolutionFor(res rw.Result, verdict Verdict) resolutionValue {
71 | 	switch verdict {
72 | 	case VerdictShadowed:
```

### 67. `internal/measure/wildcarddiscrim/leaf.go:90` — `docstring-unexported`

Load-bearing: [ ]

```go
87 | // the answer RRs, plus whether every query was reached. Reached is the
88 | // completed/incomplete discriminator — a probe nothing answered records a Gap.
89 | type controlAnswers struct {
90 | 	// perLabel[i][qtype] is label i's answer to that qtype.
91 | 	perLabel []map[rw.Qtype][]rw.RR
92 | 	reached  bool // at least one control query was reached
93 | }
```

### 68. `internal/measure/wildcarddiscrim/leaf.go:95-98` — `docstring-unexported`

Load-bearing: [ ]

```go
 92 | 	reached  bool // at least one control query was reached
 93 | }
 94 | 
 95 | // components folds a control probe's answers into the per-component signature
 96 | // union. A component appears here when some label carried that (asked, answered)
 97 | // RR; a component no label carried is determinately NoSynthesis and is consulted
 98 | // on the candidate side (differsAt) rather than enumerated here.
 99 | func (ca controlAnswers) components() map[compKey]component {
100 | 	// Gather, per component, each label's RDATA set (empty where the label had
101 | 	// no such RR). A component's key set is the union of RR types any label
```

### 69. `internal/message/census.go:28-30` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
25 | // entered beneath a membership root, or a rule that opened at `fired` and has no
26 | // firing edge of its own (a move carries the rule that opens beneath it).
27 | type CensusEntry struct {
28 | 	// Kind is what opened: "facet", "service", "endpoint", "name", "address" or
29 | 	// "signal". It is display-only — the census is a flat enumerable payload, not
30 | 	// a tree to walk.
31 | 	Kind string `json:"kind"`
32 | 	// Key is the facet name, subject key or rule name that opened.
33 | 	Key string `json:"key"`
```

### 70. `internal/message/message.go:136-137` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
133 | // `Message`). It is a frozen value: once a constructor returns one, nothing in
134 | // this package recomputes it.
135 | type Message struct {
136 | 	// ID is the store's identity, zero for a freshly-computed message not yet
137 | 	// written.
138 | 	ID int64
139 | 
140 | 	Cause Cause
```

### 71. `internal/message/pdf.go:179-181` — `docstring-unexported`

Load-bearing: [ ]

```go
176 | 	return out
177 | }
178 | 
179 | // pdfRGB is an sRGB triple drawn from the artifact light-mode token palette. The
180 | // PDF is a print document, so it renders on the light surface; the values mirror
181 | // artifactTokens (:root) and cmd/web's pageCSS.
182 | type pdfRGB struct{ r, g, b int }
183 | 
184 | var (
```

### 72. `internal/message/render.go:132-133` — `docstring-unexported`

Load-bearing: [ ]

```go
129 | 		"and no later message names it.", removed)
130 | }
131 | 
132 | // plural renders a count with its noun, thousands-separated so a large factor
133 | // reads (17,920 rather than 17920).
134 | func plural(n int, one, many string) string {
135 | 	noun := many
136 | 	if n == 1 {
```

### 73. `internal/message/render.go:589-591` — `docstring-unexported`

Load-bearing: [ ]

```go
586 | 	return s
587 | }
588 | 
589 | // artifactPeriod renders the delivery window and its number as one mono line, used
590 | // by the console view around the document. It is exported through the web layer's
591 | // header, not the document body, so it lives beside the other artifact helpers.
592 | func artifactPeriod(a Artifact) string {
593 | 	var line string
594 | 	switch {
```

### 74. `internal/proposer/proposer.go:70-71` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
67 | 	Propose(ctx context.Context, orgName string) ([]Candidate, error)
68 | }
69 | 
70 | // Registry is the set of shipped proposer paths. A lookup runs only the paths
71 | // the operator has left enabled, so a source toggled off proposes nothing.
72 | type Registry struct {
73 | 	sources []Source
74 | }
```

### 75. `internal/qr/qr.go:332` — `docstring-unexported`

Load-bearing: [ ]

```go
329 | 	m.set(8, m.Size-8, true, true)
330 | }
331 | 
332 | // drawFinder draws a 7×7 finder centred at (cx,cy) plus its light separator.
333 | func (m *Matrix) drawFinder(cx, cy int) {
334 | 	for dy := -4; dy <= 4; dy++ {
335 | 		for dx := -4; dx <= 4; dx++ {
```

### 76. `internal/qr/qr.go:629-631` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
626 | 	return forward || backward
627 | }
628 | 
629 | // Encode builds the QR Matrix for data at error-correction level M, choosing
630 | // the smallest fitting version (1..10) and the lowest-penalty mask. It returns
631 | // ErrTooLong when data exceeds a version-10 symbol.
632 | func Encode(data []byte) (*Matrix, error) {
633 | 	version, ec, ok := chooseVersion(len(data))
634 | 	if !ok {
```

### 77. `internal/queue/crtsh.go:47-52` — `docstring-unexported`

Load-bearing: [ ]

```go
44 | // tolerates far more; #879 re-measures this against the reliability bar (spec §3).
45 | const certSpotterInterval = 360 * time.Second
46 | 
47 | // maxCTPages bounds how many pages one CT job fetches for a single name-scope
48 | // domain. crt.sh is single-shot (one page); Cert Spotter paginates by cursor until
49 | // a page comes back empty. The cap is a backstop against a source that never
50 | // returns an empty page (a non-advancing cursor is already caught by the
51 | // next==cursor guard): a legitimate estate's per-domain issuance history sits far
52 | // below it, so reaching it signals an oversized or misbehaving answer.
53 | const maxCTPages = 1000
54 | 
55 | // CTFetcher fetches a crt.sh URL, returning the HTTP status and body. It is an
```

### 78. `internal/queue/produce.go:465-471` — `docstring-unexported`

Load-bearing: [ ]

```go
462 | 	return out
463 | }
464 | 
465 | // flagshipCensus is the census a flagship message carries: every facet timeline that
466 | // opened beneath the newly-reached Service this batch — certificate, http-identity,
467 | // tls-acceptance and the rest — since an opening reaches nobody on its own and rides
468 | // the flagship census instead (CONTEXT.md `Reach`; message.Census). The reachability
469 | // leg that fired is not itself a census facet. Facets nest under the Service by key:
470 | // tls-acceptance keys on the Service, certificate/http-identity on an Endpoint whose
471 | // key carries the Service key.
472 | func flagshipCensus(changes []spanChange, service string) message.Census {
473 | 	seen := map[string]bool{}
474 | 	var entries []message.CensusEntry
```

### 79. `internal/queue/progress.go:56` — `docstring-unexported`

Load-bearing: [ ]

```go
53 | 
54 | func (e safeCause) Error() string { return e.msg }
55 | 
56 | // safeProgress marks a cause message as redaction-safe to surface verbatim in the stream.
57 | func safeProgress(msg string) error { return safeCause{msg} }
58 | 
59 | // redactCause returns a stream-safe summary of a failure cause: the verbatim message only when
```

### 80. `internal/queue/worker.go:722-724` — `docstring-unexported`

Load-bearing: [ ]

```go
719 | 	})
720 | }
721 | 
722 | // retry enqueues a new job (attempt+1) and marks the current one retried, so the
723 | // eventual Batch is a fresh one and no partial batch is ever resumed. The failed
724 | // attempt keeps its own transcript on its own (retired) row.
725 | func (w *Worker) retry(ctx context.Context, job db.ClaimJobRow, t wire.Transcript, cause error) error {
726 | 	w.log.Printf("worker: job %d attempt %d failed, retrying: %v", job.ID, job.Attempt, cause)
727 | 	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
```

### 81. `internal/release/release.go:180-184` — `docstring-unexported`

Load-bearing: [ ]

```go
177 | 	return false
178 | }
179 | 
180 | // parseVersion splits a version string into its dotted numeric components,
181 | // stripping a leading "v" and dropping any pre-release/build suffix on the last
182 | // component (e.g. "v1.4.0-rc1" -> [1 4 0]). It reports ok=false when there is no
183 | // leading numeric component at all, so a non-version string ("dev") is not
184 | // silently treated as 0.0.0.
185 | func parseVersion(s string) ([]int, bool) {
186 | 	s = strings.TrimSpace(s)
187 | 	s = strings.TrimPrefix(s, "v")
```

### 82. `internal/remoteexec/conn.go:68-69` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
65 | 	// ExitExited: the command ran to completion and returned Code. Code is -1 when the
66 | 	// server reported no status (an honest "no clean exit", never a fabricated success).
67 | 	ExitExited ExitKind = iota
68 | 	// ExitSignalled: a signal killed the command. Signal is the SSH signal name, or
69 | 	// empty when the server sent no exit status at all (*ssh.ExitMissingError).
70 | 	ExitSignalled
71 | 	// ExitContextCancelled: the caller's context cancelled the session before the
72 | 	// command finished; the worker killed the remote command.
```

### 83. `internal/remoteexec/platform.go:12-16` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 9 | 	"golang.org/x/crypto/ssh"
10 | )
11 | 
12 | // Platform is the remote prober's operating system and CPU, read from `uname` on
13 | // the connection. GOOS/GOARCH are the Go identifiers the arch check matches the
14 | // pushed binary against; Label is the accepted-platform chip the VantageCard renders
15 | // (fixtures.json pins the "linux · x86_64" spelling — lowercase OS, a middle-dot
16 | // separator, the raw uname machine — so the live datum reads back in the same shape).
17 | type Platform struct {
18 | 	GOOS   string
19 | 	GOARCH string
```

### 84. `internal/remoteexec/platform.go:23-25` — `docstring-unexported`

Load-bearing: [ ]

```go
20 | 	Label  string
21 | }
22 | 
23 | // unameToGOOS maps the `uname -s` kernel name to the Go GOOS the prober matrix builds
24 | // for. Only linux is in the matrix today (packaging-and-configuration.md §1.2); an
25 | // unrecognised kernel yields "" so the arch check refuses rather than guessing.
26 | func unameToGOOS(s string) string {
27 | 	switch strings.ToLower(strings.TrimSpace(s)) {
28 | 	case "linux":
```

### 85. `internal/report/notify.go:75-77` — `docstring-unexported`

Load-bearing: [ ]

```go
72 | // MarshalReadyBody renders the ready-message to the exact bytes POSTed and signed.
73 | func MarshalReadyBody(b ReadyBody) ([]byte, error) { return json.Marshal(b) }
74 | 
75 | // shouldNotify is the enqueue predicate: a scheduled run enqueues exactly one
76 | // ready-message when — and only when — its schedule binds a Channel. A download-only
77 | // schedule (NULL channel_id) binds none and enqueues nothing (P0.6c/T7).
78 | func shouldNotify(channelID pgtype.Int8) bool { return channelID.Valid }
79 | 
80 | func trimSlash(s string) string {
```

### 86. `internal/retention/observation.go:115-129` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
112 | 	return Evidential
113 | }
114 | 
115 | // RetainObservation reports whether an observation row survives a retention sweep.
116 | // It is the row-level rule the deletion query encodes in SQL, stated here as a
117 | // pure function so the two-tier boundary, the tightest-floor collapse and the two
118 | // exception populations are provable without a database.
119 | //
120 | //	ageSeconds   now − observed_at, the row's age.
121 | //	boundSeconds k cadences of the tightest Scan covering the row's timeline.
122 | //	dialSeconds  the operator's dial in seconds; 0 == unbounded (the v1 default).
123 | //	hasBound     false where no enabled Scan covers the timeline (undefined bound).
124 | //	withdrawn    the row's subject has left the estate — its timelines are closed.
125 | //
126 | // A row is retained while its age is inside EITHER its own bound OR the dial,
127 | // whichever is longer. Two populations fall outside that rule and opposite ways:
128 | // an undefined bound is never expired (never retired); a withdrawn subject carries
129 | // no floor at all, so the dial alone governs it.
130 | func RetainObservation(ageSeconds, boundSeconds, dialSeconds int64, hasBound, withdrawn bool) bool {
131 | 	if withdrawn {
132 | 		// No floor: the row's own bound does not protect it, the dial alone governs.
```

### 87. `internal/retention/observation.go:195-199` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
192 | 	DeleteExpiredObservations(ctx context.Context, arg db.DeleteExpiredObservationsParams) (int64, error)
193 | }
194 | 
195 | // ObservationRetirer sweeps evidential observations the operator's dial no longer
196 | // keeps. It never touches a live row: the delete query evaluates each row's own
197 | // per-timeline bound and reaches only rows past BOTH that bound and the dial (or,
198 | // for a withdrawn subject, past the dial alone). It is landed beside the Dispatch
199 | // Retirer, not folded into it.
200 | type ObservationRetirer struct {
201 | 	store ObservationStore
202 | 	now   func() time.Time
```

### 88. `internal/retention/transcript.go:100-103` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 97 | 	return &TranscriptRetirer{store: store, now: now, log: logger}
 98 | }
 99 | 
100 | // Sweep retires every transcript captured before the floored window and returns how
101 | // many it deleted. When the dial is 0 it deletes nothing and returns 0 — the
102 | // explicit operator opt-out. Unlike the other two dials this sweep is ACTIVE on a
103 | // fresh install: the dial ships bounded at 14 days (migration 23700).
104 | func (r *TranscriptRetirer) Sweep(ctx context.Context) (int64, error) {
105 | 	settings, err := r.store.GetRetentionSettings(ctx)
106 | 	if err != nil {
```

### 89. `internal/scan/cttail.go:460-465` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
457 | 	return CTSignedTreeHead{TreeSize: size, Raw: body}, nil
458 | }
459 | 
460 | // DataTilePath renders a tile index into the static-ct-api path encoding: 3-digit,
461 | // zero-padded base-1000 segments, most-significant first, with an `x` prefix on every
462 | // segment but the last. This bounds directory fan-out (a log with billions of entries
463 | // never puts millions of tiles in one directory). Examples: 0 -> "000", 1 -> "001",
464 | // 1234 -> "x001/234", 1234567 -> "x001/x234/567". The caller appends it to
465 | // `<monitoring>/tile/data/` and adds the `.p/<W>` suffix for a partial tile.
466 | func DataTilePath(index int64) string {
467 | 	segs := []string{fmt.Sprintf("%03d", index%1000)}
468 | 	for index /= 1000; index > 0; index /= 1000 {
```

### 90. `internal/scan/ctverify.go:47` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
44 | // (SCTLeafIndex). The digitally-signed signature is read past and discarded: verification
45 | // recomputes inclusion to the head's root and never checks the log's signature (§4.4).
46 | type SCT struct {
47 | 	// Version is the sct_version; only v1 (0) is understood.
48 | 	Version uint8
49 | 	// LogID is the 32-byte SHA-256 of the log's public key — the same value log_list.json
50 | 	// carries base64-encoded, so FindLogByLogID matches on the base64 of this.
```

### 91. `internal/scan/ctverify.go:52` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
49 | 	// LogID is the 32-byte SHA-256 of the log's public key — the same value log_list.json
50 | 	// carries base64-encoded, so FindLogByLogID matches on the base64 of this.
51 | 	LogID [32]byte
52 | 	// Timestamp is the SCT's millisecond timestamp, folded verbatim into the leaf hash.
53 | 	Timestamp uint64
54 | 	// Extensions is the CtExtensions content (already unwrapped from its opaque<0..2^16-1>
55 | 	// length). It rides into the TimestampedEntry.extensions when the leaf hash is built, so
```

### 92. `internal/scan/zone.go:53-54` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
50 | // clean current fact (v1 spec §3.4). Before the cadence the file is current, and
51 | // Days counts down to the instant it ages into that gap.
52 | type ZoneAging struct {
53 | 	// Supplied reports whether there is a dated supply to age at all. A name
54 | 	// scope with no zone file has nothing to stale.
55 | 	Supplied bool
56 | 	// Stale reports whether the file has passed its re-supply interval and so has
57 | 	// aged into a coverage gap.
```

### 93. `internal/signal/endpoint.go:89-93` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
86 | 	HTTPStatus       int
87 | 	RedirectLocation string
88 | 
89 | 	// RedirectHostInEstate reports whether the host the 3xx `Location` names is a
90 | 	// subject in the estate — the pre-folded evidence `redirect-to-host-outside-estate`
91 | 	// reads (the estate membership is a Derived value the web layer folds, like
92 | 	// InDeclaredZone on NameFacts). It is meaningful only where the Endpoint is in
93 | 	// that rule's domain (a 3xx with a Location).
94 | 	RedirectHostInEstate bool
95 | }
96 | 
```

### 94. `internal/signal/endpoint.go:97` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 94 | 	RedirectHostInEstate bool
 95 | }
 96 | 
 97 | // EndpointRule is one `Signal` whose subject is an `Endpoint`.
 98 | type EndpointRule interface {
 99 | 	Name() string
100 | 	Version() Version
```

### 95. `internal/signal/rules.go:44-46` — `docstring-unexported`

Load-bearing: [ ]

```go
41 | // is the one the form and the acceptance guard both read.
42 | func RuleNames() []string { return AllRuleNames() }
43 | 
44 | // hasAnswer reports whether a composed resolution outcome carries an address set
45 | // the world affirmed — the two answer-bearing values. NameError / NoData / Lame
46 | // / Gap carry no answer.
47 | func hasAnswer(outcome string) bool {
48 | 	return outcome == Resolved || outcome == Shadowed
49 | }
```

### 96. `internal/signal/rules.go:99-101` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 96 | func (cnameTargetNameError) Name() string     { return "cname-target-name-error" }
 97 | func (cnameTargetNameError) Version() Version { return Version{Rule: "v1", Composes: leafVersions} }
 98 | 
 99 | // Severity: high — a CNAME pointing at a NameError target is the classic
100 | // subdomain-takeover setup: whoever claims the dangling target serves under the
101 | // operator's name.
102 | func (cnameTargetNameError) Severity() Severity { return SevHigh }
103 | func (cnameTargetNameError) Eval(f NameFacts) Outcome {
104 | 	if f.CNAMETarget == "" {
```

### 97. `internal/signal/signal.go:85-88` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
82 | 	Resolution string
83 | 	Addresses  []string
84 | 
85 | 	// CNAMETarget is the target `Name` of this Name's dns-record CNAME, empty
86 | 	// when the Name holds no CNAME (which puts it outside cname-target-name-error's
87 | 	// domain). TargetResolution is that target's own composed resolution outcome,
88 | 	// empty when the target was never measured.
89 | 	CNAMETarget      string
90 | 	TargetResolution string
91 | 
```

### 98. `internal/transcript/crypto.go:10-20` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 7 | 	"golang.org/x/crypto/chacha20poly1305"
 8 | )
 9 | 
10 | // Seal encrypts a single transcript stream for storage in its bytea column,
11 | // returning a fresh 24-byte random nonce prepended to the XChaCha20-Poly1305
12 | // ciphertext. The result is raw bytes, not base64 — the column is bytea, so no
13 | // text encoding is needed (unlike auth.EncryptTOTPSecret, which targets a text
14 | // column).
15 | //
16 | // A nil plaintext returns nil: a stream the variant does not carry stays SQL NULL,
17 | // preserving the table's NULL-vs-captured-empty distinction (migration 23700). A
18 | // non-nil but empty stream seals to real ciphertext, so a captured-empty stream is
19 | // retained as captured, not collapsed to NULL. Each call draws a fresh random
20 | // nonce, so sealing the same bytes twice yields different ciphertext.
21 | func Seal(key, plaintext []byte) ([]byte, error) {
22 | 	if plaintext == nil {
23 | 		return nil, nil
```

### 99. `internal/vantage/keypair.go:69-70` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
66 | 	return AuthorizedKey(signer.PublicKey()), nil
67 | }
68 | 
69 | // AuthorizedKey renders a public key as a single authorized_keys line with no
70 | // trailing newline, the form stored in the database and rendered on web.
71 | func AuthorizedKey(key ssh.PublicKey) string {
72 | 	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
73 | }
```

### 100. `internal/wire/transcript.go:156` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
153 | 
154 | func (ZoneTranscript) isTranscript() {}
155 | 
156 | // ZoneOutcome is how the zone restate ended: a closed union of two.
157 | type ZoneOutcome interface{ isZoneOutcome() }
158 | 
159 | // ZoneParsed is a restate that parsed the zone file.
```

