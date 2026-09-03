# Comment policy validation gate — production Go, round 2

SPEC `docs/spec/comment-policy.md` §3.9. Regenerate this sheet with:

```sh
go run ./cmd/commentlint sample --population production --round 2
```

- In-scope Go files read: 255
- Blocks the §3.2 screen admits for deletion: 1022
- Blocks drawn into the gate sample: 100
- Blocks drawn into the coverage supplement: 0

Accept a class at 2 or fewer load-bearing blocks. A class that fails three rounds leaves the v1 delete set and stays in the flag set.

This sheet holds the current round. Every round's verdicts sit in [the round ledger](comment-gate-rounds.md).

## Class coverage

| Class | Admitted | Gate sample | Supplement |
| --- | --- | --- | --- |
| `commented-out-code` | 0 | 0 | 0 |
| `docstring-exported-conventional` | 462 | 47 | 0 |
| `docstring-unexported` | 444 | 44 | 0 |
| `section-divider` | 88 | 8 | 0 |
| `short-label` | 28 | 1 | 0 |

## Verdicts

A reviewer fills the last two columns. A class that admits no block on this population reads `n/a`.

| Class | Read | Load-bearing | Verdict |
| --- | --- | --- | --- |
| `commented-out-code` | 0 | n/a | n/a |
| `docstring-exported-conventional` | 47 | | |
| `docstring-unexported` | 44 | | |
| `section-divider` | 8 | | |
| `short-label` | 1 | | |

## Gate sample

### 1. `cmd/web/api_v1.go:159` — `section-divider`

Load-bearing: [ ]

```go
156 | 	writeAPIJSON(w, out)
157 | }
158 | 
159 | // --- subjects ---------------------------------------------------------------
160 | 
161 | type apiSubjectsResponse struct {
162 | 	Names     []apiSubject `json:"names"`
```

### 2. `cmd/web/api_v1.go:280` — `section-divider`

Load-bearing: [ ]

```go
277 | 	writeAPIJSON(w, out)
278 | }
279 | 
280 | // --- signals ----------------------------------------------------------------
281 | 
282 | type apiSignalsResponse struct {
283 | 	Open      []apiSignal `json:"open"`
```

### 3. `cmd/web/custodycensus.go:64` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
61 | // custodyCensusView is the whole section, shaped for scope.tmpl: the declined rows,
62 | // and how many candidates still wait on their first measurement (#1015).
63 | type custodyCensusView struct {
64 | 	// Rows are the declines, one per (citing name, edge) pair, in resolution order.
65 | 	Rows []custodyCensusRow
66 | 	// Pending is the number of DISTINCT edges the `edge-fanout` Scan holds and has
67 | 	// not measured yet. It is a COUNT and never a row, because a pending candidate
```

### 4. `cmd/web/devfixtures.go:599-600` — `docstring-unexported`

Load-bearing: [ ]

```go
596 | 	devExposureWithheldVariant = "no-internet-vantage"
597 | )
598 | 
599 | // devExposureRow mirrors one fixtures.json exposure.rows entry: a service's address, its
600 | // ":port transport", the internal + internet reach legs (expleg display states), and its since.
601 | type devExposureRow struct {
602 | 	asset    string
603 | 	svc      string
```

### 5. `cmd/web/devfixtures.go:1190-1192` — `docstring-unexported`

Load-bearing: [ ]

```go
1187 | 	devScopeRefusalReachable = "203.0.113.0/22"
1188 | 	devScopeRefusalFormError = "Refused — over the 1,024-address cap."
1189 | 
1190 | 	// The exclusion-preview golden (states.json scope "exclusion-preview") types
1191 | 	// devScopeExclPreviewValue and clicks Preview; the handler renders the firing receipt.
1192 | 	// These mirror fixtures.json scope.exclusion_preview_fixture.
1193 | 	devScopeExclPreviewKind     = "subtree"
1194 | 	devScopeExclPreviewValue    = "staging-4.acmecorp.io"
1195 | 	devScopeExclPreviewFires    = true
```

### 6. `cmd/web/devfixtures.go:1480-1481` — `docstring-unexported`

Load-bearing: [ ]

```go
1477 | 	devSignalsDiscovered = "2026-08-12"
1478 | )
1479 | 
1480 | // devSignalsSevOptions is the severity listbox vocabulary (fixtures.json signals — the filter
1481 | // options), in authored order. "All severities" is the unfiltered default.
1482 | var devSignalsSevOptions = []string{"All severities", "Critical", "High", "Medium", "Low", "Info"}
1483 | 
1484 | // devSignalRow mirrors one fixtures.json signals row (open or withdrawn): the SIG id + view key,
```

### 7. `cmd/web/drift.go:127-129` — `docstring-unexported`

Load-bearing: [ ]

```go
124 | 	Events    []driftEvent
125 | }
126 | 
127 | // driftPeriod is one entry of the period selector: the ?period token, its badge
128 | // label, and the lookback window it maps to. `all` maps to a zero duration, read as
129 | // "since the beginning" (no lower bound).
130 | type driftPeriod struct {
131 | 	Token  string
132 | 	Label  string
```

### 8. `cmd/web/graph.go:225` — `docstring-unexported`

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

### 9. `cmd/web/graph.go:299-300` — `docstring-unexported`

Load-bearing: [ ]

```go
296 | 	}
297 | }
298 | 
299 | // round1 rounds a float to one decimal place — the minimap coordinate precision the
300 | // design fixture carries (e.g. 90·110/1200 = 8.25 → 8.3).
301 | func round1(v float64) float64 { return math.Round(v*10) / 10 }
302 | 
303 | // classifyNameTypes splits the Name set into the domain|subdomain tiers (#22e): a
```

### 10. `cmd/web/inventory.go:410-412` — `docstring-unexported`

Load-bearing: [ ]

```go
407 | 	return a.Key < b.Key
408 | }
409 | 
410 | // leadingFacet returns a subject's first facet after the canonical facet sort — the
411 | // facet the subject order keys on — or a zero facet for a subject that (impossibly,
412 | // every inventory subject holds at least one) holds none.
413 | func leadingFacet(s inventorySubject) inventoryFacet {
414 | 	if len(s.Facets) == 0 {
415 | 		return inventoryFacet{}
```

### 11. `cmd/web/main.go:237` — `docstring-unexported`

Load-bearing: [ ]

```go
234 | 	return token, nil
235 | }
236 | 
237 | // isTruthy reads the common affirmative spellings of a boolean env value.
238 | func isTruthy(v string) bool {
239 | 	switch strings.ToLower(strings.TrimSpace(v)) {
240 | 	case "1", "true", "yes", "on":
```

### 12. `cmd/web/messages.go:39-41` — `docstring-unexported`

Load-bearing: [ ]

```go
36 | 	LastError string
37 | }
38 | 
39 | // messageRow is one message shaped for the panel: the rendered headline, the
40 | // per-mover link, the cause and class as read-only labels, the instant, the
41 | // read-state, and the census rows where the firing carries one.
42 | type messageRow struct {
43 | 	ID       int64
44 | 	Cause    string
```

### 13. `cmd/web/messages.go:55-56` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
52 | 	Instant  string
53 | 	Read     bool
54 | 	Census   []censusRowView
55 | 	// Deliveries is this message's outcome to each routed Channel. A message with
56 | 	// no channel configured carries none; an undelivered one carries a Failed row.
57 | 	Deliveries []deliveryView
58 | 	// AnyUndelivered is true where at least one delivery is undelivered, so the
59 | 	// panel can flag the message without the operator opening every row.
```

### 14. `cmd/web/messages.go:492-496` — `docstring-unexported`

Load-bearing: [ ]

```go
489 | // itself; the menu item only decides whether a delivery exists to open.
490 | const reportDeliveryHref = "/reports/delivery"
491 | 
492 | // reportScheduleRow is one recurring report shaped for the Reports screen's
493 | // "Recurring reports" table (T17, after design-system/examples/console/Reports.jsx).
494 | // Name / Cadence / Format / LastSent are the schedule's facts; the row-action menu
495 | // carries "View last delivery", which opens the delivered artifact when this report
496 | // has a delivery and is disabled where it has none (no fabrication).
497 | type reportScheduleRow struct {
498 | 	// ID keys the row's mutations — the Run now / Edit / Delete row-menu actions post
499 | 	// it back so the handler resolves which schedule to act on (P0.6/T4).
```

### 15. `cmd/web/ratelimit.go:27` — `docstring-unexported`

Load-bearing: [ ]

```go
24 | type loginLimiter struct {
25 | 	now func() time.Time
26 | 
27 | 	// maxFailures is how many failures a key may accrue before it locks.
28 | 	maxFailures int
29 | 	// window is the idle span after which a key's failure count is considered
30 | 	// stale and reset — a reset that costs no goroutine, applied lazily on the
```

### 16. `cmd/web/rawoutput.go:38-41` — `docstring-unexported`

Load-bearing: [ ]

```go
35 | 	SentScope string   `json:"sentScope"`
36 | }
37 | 
38 | // rawOutputView is one job's Transcript shaped for the dedicated admin view. Captured is
39 | // false for a legible absence (no transcript row) — distinct from a captured-but-empty
40 | // stream, which renders as an empty panel with Captured true. The exec-meta fields are
41 | // meaningful only when Captured.
42 | type rawOutputView struct {
43 | 	RunID   int64
44 | 	RunHref string // back to the filtered run page (?job={id})
```

### 17. `cmd/web/rawoutput.go:350-351` — `docstring-unexported`

Load-bearing: [ ]

```go
347 | 	return out
348 | }
349 | 
350 | // rawHumanBytes renders a byte count in binary units (KiB/MiB/…) for the exec-meta and
351 | // truncation notes.
352 | func rawHumanBytes(n int) string {
353 | 	const unit = 1024
354 | 	if n < unit {
```

### 18. `cmd/web/reports.go:376-377` — `docstring-unexported`

Load-bearing: [ ]

```go
373 | 	}
374 | }
375 | 
376 | // pluralAssets renders a discovery count as the console's terse asset figure — the
377 | // bar's hover title. "asset" is the UI collective noun for a watched Name/Service.
378 | func pluralAssets(n int) string {
379 | 	if n == 1 {
380 | 		return "1 asset"
```

### 19. `cmd/web/reports.go:624-627` — `docstring-unexported`

Load-bearing: [ ]

```go
621 | 	X, Y, Text string
622 | }
623 | 
624 | // reportsTimeSeries is the server-rendered form of TimeSeriesChart.jsx for the
625 | // "Open signals over time" card: a fixed viewBox (scaled to width via the SVG),
626 | // nice-stepped y gridlines, sparse x labels, and the two standing series — All open
627 | // (--chart-1) and Critical + high (--chart-2). Every string is paint-ready.
628 | type reportsTimeSeries struct {
629 | 	W, H     int
630 | 	Grid     []reportsGridLine
```

### 20. `cmd/web/reports_schedule.go:100-103` — `docstring-unexported`

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

### 21. `cmd/web/restore.go:176-178` — `docstring-unexported`

Load-bearing: [ ]

```go
173 | 	}, nil
174 | }
175 | 
176 | // backupAllowed reports whether a table name is in B3's ordered allowlist. Restore reads
177 | // and overwrites only these — a manifest naming anything else is refused, honouring the
178 | // same allowlist/denylist partition backup writes with.
179 | func backupAllowed(table string) bool {
180 | 	for _, t := range backupTables {
181 | 		if t == table {
```

### 22. `cmd/web/seeds.go:996-997` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 993 | 	// no cadence to age against.
 994 | 	AgingStale bool
 995 | 	AgingLabel string
 996 | 	// IntervalLabel renders the operator's declared re-supply cadence for the
 997 | 	// file line ("monthly", "weekly", or "every N days").
 998 | 	IntervalLabel string
 999 | }
1000 | 
```

### 23. `cmd/web/settings.go:69-70` — `docstring-unexported`

Load-bearing: [ ]

```go
66 | // checkboxes and badges from this, never from a literal set baked into the markup.
67 | var channelClasses = []string{"drift", "coverage", "clock"}
68 | 
69 | // accountRow is one account in the management list. It carries no password hash
70 | // and no TOTP secret — managing an account needs neither.
71 | type accountRow struct {
72 | 	ID          int64
73 | 	Username    string
```

### 24. `cmd/web/settings.go:187-191` — `docstring-unexported`

Load-bearing: [ ]

```go
184 | 	// success stash) and read on the way out; it never reaches the template.
185 | 	flashTab string
186 | 
187 | 	// team (T18). teamError is an inline error on the members surface; roleError is
188 | 	// the change-role guard's message. inviteLink is a freshly minted join URL,
189 | 	// revealed once by createInvite; inviteOpen re-opens the invite dialog on a
190 | 	// rejected mint and inviteRole echoes its role. removeID/removeError re-open the
191 | 	// remove ConfirmDialog on a typed-name mismatch or a guard refusal.
192 | 	teamError   string
193 | 	roleError   string
194 | 	inviteRole  string
```

### 25. `cmd/web/settings_sso.go:50-53` — `docstring-unexported`

Load-bearing: [ ]

```go
47 | 	LinkedAt     string
48 | }
49 | 
50 | // fillSSOSection reads the configured providers, the current identity bindings, and the
51 | // add-form echo. The section is the honest empty-state when none are configured (the
52 | // SignIn "not configured" state mirrors it); once a provider exists, SignIn renders a
53 | // button for it.
54 | func (s *server) fillSSOSection(r *http.Request, f settingsForms, data map[string]any) error {
55 | 	rows, err := s.store.ListSSOProviders(r.Context())
56 | 	if err != nil {
```

### 26. `cmd/web/sso.go:547` — `docstring-unexported`

Load-bearing: [ ]

```go
544 | 	return tx, true
545 | }
546 | 
547 | // randToken returns a 256-bit URL-safe random token for the state and nonce values.
548 | func randToken() string {
549 | 	b := make([]byte, 32)
550 | 	if _, err := rand.Read(b); err != nil {
```

### 27. `cmd/web/subjects.go:143-145` — `docstring-exported-conventional`

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

### 28. `cmd/web/subjects.go:599-604` — `docstring-unexported`

Load-bearing: [ ]

```go
596 | 	return addr, port, transport
597 | }
598 | 
599 | // buildServiceCitation assembles the "why is this here" chain for a Service: the
600 | // Service itself, the Address the triple sits on, and the ground the Address's
601 | // membership rests on — the Name whose current resolution cites the Address, or
602 | // the address-scope Seed that covers it, terminating at a Seed. Where neither
603 | // limb holds, the Address has left the estate (the `uncited` / `descoped`
604 | // departure), which the caller renders as a withdrawn Service.
605 | func (s *server) buildServiceCitation(r *http.Request, addr string) (hops []citationHop, terminated, withdrawn bool, seedScope, inScopeSince string) {
606 | 	hops = []citationHop{
607 | 		{Label: "Subject · Service", Value: r.FormValue("key")},
```

### 29. `cmd/web/subjects.go:1217` — `docstring-unexported`

Load-bearing: [ ]

```go
1214 | 	Fingerprint string
1215 | }
1216 | 
1217 | // assetDNSRow is one resolved record: the RR type, its value, and when last seen.
1218 | type assetDNSRow struct {
1219 | 	Type  string
1220 | 	Value string
```

### 30. `cmd/web/vantageclass.go:119-121` — `docstring-unexported`

Load-bearing: [ ]

```go
116 | 	return out
117 | }
118 | 
119 | // moreRecent reports whether a per-vantage leg row should replace the one currently
120 | // chosen for its (subject, derived class) bucket — the opened_at DESC, id DESC tiebreak
121 | // the retired SQL DISTINCT ON encoded (a resolution read passes observed_at as openedAt).
122 | func moreRecent(openedAt time.Time, id int64, cur reachLegRow) bool {
123 | 	if openedAt.After(cur.openedAt) {
124 | 		return true
```

### 31. `docs/guides/embed.go:13-16` — `docstring-exported-conventional`

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

### 32. `internal/custody/census.go:42` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
39 | 	// Address is the edge the record points at, Unmap'ed as every address in this
40 | 	// package is.
41 | 	Address netip.Addr
42 | 	// State is why the extension does not reach it.
43 | 	State ExtensionState
44 | 	// Scope is the declared address scope that ALSO covers the edge, and the zero
45 | 	// Prefix where none does. A valid Prefix here is ADR-0129's dual-limb row: the
```

### 33. `internal/custody/corpus/harness.go:202` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
199 | 	UncoveredMoves []UncoveredMove `json:"uncovered_moves"`
200 | }
201 | 
202 | // LoadLock reads corpus.lock.json from dir.
203 | func LoadLock(dir string) (Lock, error) {
204 | 	b, err := os.ReadFile(filepath.Join(dir, "corpus.lock.json")) // #nosec G304 (test corpus loader; filename constant, dir is the fixed test corpus directory ".")
205 | 	if err != nil {
```

### 34. `internal/custody/corpus/rows.go:41` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
38 | 	AddressExclusions []string
39 | 	// ExtendedZones are the registrable domains of custody-extended name scopes.
40 | 	ExtendedZones []string
41 | 	// Resolutions are the observed direct A/AAAA records.
42 | 	Resolutions []Resolution
43 | 	// ScanInForce is `edge-fanout`'s DISPOSITION — EdgeFanout.Enabled. False is a
44 | 	// disabled Scan and a Scan whose row is absent.
```

### 35. `internal/custody/custody.go:67` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
64 | 	// custody extension. A name-scope Seed with the extension off contributes
65 | 	// nothing to the derivation.
66 | 	ExtendedZones []string
67 | 	// Resolutions are the observed direct A/AAAA records of names in the estate.
68 | 	Resolutions []Resolution
69 | 	// edgeFanout is the `edge-fanout` Scan's measured result. It narrows the
70 | 	// custody extension's reach and reaches NOTHING ELSE (ADR-0129 §4).
```

### 36. `internal/delivery/delivery.go:204` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
201 | type Verdict int
202 | 
203 | const (
204 | 	// VerdictDelivered: a 2xx. Nothing more is sent.
205 | 	VerdictDelivered Verdict = iota
206 | 	// VerdictRetry: a failure with attempts remaining. The delivery returns to
207 | 	// pending on the shared backoff.
```

### 37. `internal/delivery/delivery.go:206-207` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
203 | const (
204 | 	// VerdictDelivered: a 2xx. Nothing more is sent.
205 | 	VerdictDelivered Verdict = iota
206 | 	// VerdictRetry: a failure with attempts remaining. The delivery returns to
207 | 	// pending on the shared backoff.
208 | 	VerdictRetry
209 | 	// VerdictUndelivered: a failure with the attempt budget spent. Dead-lettered —
210 | 	// the undelivered mark — leaving the Message untouched.
```

### 38. `internal/drift/delta.go:40-45` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
37 | 	return s.ClosedAt.IsZero() || s.ClosedAt.After(t)
38 | }
39 | 
40 | // OpenAt returns the spans open at instant t — the population one timeline-set
41 | // held then. Evaluated at the previous batch's instant it is the "value a batch
42 | // ago"; the current population is CurrentlyOpen. The caller must supply a span set
43 | // that still carries the spans the most recent batch closed (a read filtered to
44 | // `closed_at IS NULL OR closed_at > prevBatchAt` does), or a span the batch closed
45 | // is missing and the previous count is understated.
46 | func OpenAt(spans []Span, t time.Time) []Span {
47 | 	out := make([]Span, 0, len(spans))
48 | 	for _, s := range spans {
```

### 39. `internal/drift/drift.go:74-76` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
71 | 	return out
72 | }
73 | 
74 | // Equal reports whether two vectors are the same set of (leaf, version) pairs.
75 | // This is the whole comparability precondition: two spans compare only where
76 | // Equal holds, and the boundary between two spans where it does not is a Break.
77 | func (v Vector) Equal(o Vector) bool {
78 | 	if len(v) != len(o) {
79 | 		return false
```

### 40. `internal/drift/trend.go:44` — `section-divider`

Load-bearing: [ ]

```go
41 | 	return idx, true
42 | }
43 | 
44 | // --- Signals over time -------------------------------------------------------
45 | 
46 | // Raise is one signal instance as the trend fold sees it: the instant the
47 | // (rule, subject) pair was FIRST seen firing (signal_instance.first_seen) and
```

### 41. `internal/env/env.go:11` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 8 | 	"os"
 9 | )
10 | 
11 | // OrDefault returns the value of key, or fallback if it is unset.
12 | func OrDefault(key, fallback string) string {
13 | 	if v, ok := os.LookupEnv(key); ok {
14 | 		return v
```

### 42. `internal/measure/blanketdiscrim/corpus/harness.go:79-81` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
76 | 	Date       string `json:"date"`
77 | }
78 | 
79 | // Lock is the checked-in manifest that binds the leaf version to the corpus and
80 | // parameter digests. A lock edit that bumps the version with no digest move and no
81 | // new uncovered move is what CI's version gate refuses.
82 | type Lock struct {
83 | 	LeafVersion    string          `json:"leaf_version"`
84 | 	CorpusDigest   string          `json:"corpus_digest"`
```

### 43. `internal/measure/blanketdiscrim/leaf.go:80` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
77 | 	ControlIncomplete ControlResult = "incomplete"
78 | )
79 | 
80 | // Verdict is the leaf's decision for one `Address`.
81 | type Verdict string
82 | 
83 | const (
```

### 44. `internal/measure/blanketdiscrim/leaf.go:146-147` — `docstring-exported-conventional`

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

### 45. `internal/measure/blanketdiscrim/leaf.go:185-187` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
182 | 	ReasonIncomplete = "the control-port probe did not complete, so this reach could not be discriminated from a blanket responder"
183 | )
184 | 
185 | // ReasonFor renders the operator-facing reason for a gapping Verdict. Only
186 | // VerdictBlanket and VerdictGap gap a reach; VerdictNotBlanket passes the connect
187 | // value through and has no reason.
188 | func ReasonFor(v Verdict) string {
189 | 	switch v {
190 | 	case VerdictBlanket:
```

### 46. `internal/measure/connectoutcome/certcorpus/harness.go:89` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
86 | 	UncoveredMoves []UncoveredMove `json:"uncovered_moves"`
87 | }
88 | 
89 | // LoadLock reads corpus.lock.json from dir.
90 | func LoadLock(dir string) (Lock, error) {
91 | 	b, err := os.ReadFile(filepath.Join(dir, "corpus.lock.json")) // #nosec G304 (test corpus loader; filename constant, dir is the fixed test corpus directory ".")
92 | 	if err != nil {
```

### 47. `internal/measure/connectoutcome/certcorpus/rows.go:9-10` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 6 | 	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
 7 | )
 8 | 
 9 | // Step is one run of the reachability exchange inside a row: one Batch at one
10 | // Vantage over one scope, against a scripted connector and a scripted handshaker.
11 | type Step struct {
12 | 	Batch     string
13 | 	Scope     co.Scope
```

### 48. `internal/measure/connectoutcome/corpus/rows.go:30` — `short-label`

Load-bearing: [ ]

```go
27 | // AllCells is the enumeration the coverage test counts against: every cell of
28 | // the connect-outcome block must be pinned by at least one row.
29 | var AllCells = []string{
30 | 	// C1 — the two verdicts
31 | 	"C1/reached", "C1/not-reached",
32 | 	// C2 — the raw results that decide
33 | 	"C2/open", "C2/refused", "C2/timeout-exhausted",
```

### 49. `internal/measure/connectoutcome/leaf.go:21` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
18 | type ConnResult string
19 | 
20 | const (
21 | 	// ConnOpen: the three-way handshake completed. A decided positive.
22 | 	ConnOpen ConnResult = "open"
23 | 	// ConnRefused: the host answered with an RST — the port is shut. A decided
24 | 	// negative, and an answer, so it is never retried.
```

### 50. `internal/measure/connectoutcome/leaf.go:48-49` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
45 | type Outcome string
46 | 
47 | const (
48 | 	// Reached: the connection completed. The `Service` is reachable from this
49 | 	// Vantage.
50 | 	Reached Outcome = "reached"
51 | 	// NotReached: the connect was refused, or timed out after its retries. On a
52 | 	// connection-oriented transport silence decides, so this is a value and not a
```

### 51. `internal/measure/connectoutcome/offers.go:142-143` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
139 | 	}
140 | }
141 | 
142 | // Digest is a stable content hash of the profile, used by the golden-corpus lock
143 | // to bind a declared-parameter change to a Version bump.
144 | func (p SafetyProfile) Digest() string {
145 | 	b, err := json.Marshal(p)
146 | 	if err != nil {
```

### 52. `internal/measure/connectoutcome/tls.go:54-55` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
51 | type TLSOutcome string
52 | 
53 | const (
54 | 	// TLSPresented: the peer completed a TLS handshake and presented a
55 | 	// certificate chain. The value carries the chain, ordered leaf-first.
56 | 	TLSPresented TLSOutcome = "presented"
57 | 	// TLSRefused: the peer spoke TLS but accepted no candidate we offered — an
58 | 	// SSLv3-only or SNI-required listener that would otherwise be misfiled under
```

### 53. `internal/measure/httpexchange/corpus/harness.go:74-76` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
71 | 	Date       string `json:"date"`
72 | }
73 | 
74 | // Lock is the checked-in manifest that binds the leaf version to the corpus and
75 | // parameter digests. A lock edit that bumps the version with no digest move and no
76 | // new uncovered move is what CI's version gate refuses.
77 | type Lock struct {
78 | 	LeafVersion    string          `json:"leaf_version"`
79 | 	CorpusDigest   string          `json:"corpus_digest"`
```

### 54. `internal/measure/httpexchange/corpus/rows.go:62` — `section-divider`

Load-bearing: [ ]

```go
59 | // Rows is the checked-in corpus. Every cell in AllCells appears in some row's
60 | // Cells; the coverage test fails the build (naming the cell) if one does not.
61 | var Rows = []Row{
62 | 	// ---- H1/named-200 ----
63 | 	{
64 | 		Cells:        []string{"H1/named-200"},
65 | 		Claim:        "a completed GET / to a named (Name, Service) pair creates the Endpoint and records its http-identity — outcome responded, status, Server header, and the page <title> lifted from the body (never the body itself)",
```

### 55. `internal/measure/httpexchange/exchange.go:30` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
27 | 	// Address is the Service's Address (the `A`/`AAAA` the Name resolved to, or a
28 | 	// Seed-covered address).
29 | 	Address string `json:"address"`
30 | 	// Port is the Service's TCP port.
31 | 	Port uint16 `json:"port"`
32 | 	// Scheme is the URL scheme spoken — "http" or "https". It records how the
33 | 	// exchange was framed and never widens what was probed.
```

### 56. `internal/measure/httpexchange/leaf.go:55-56` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
52 | 	Method string `json:"method"`
53 | 	// Path is fixed: `/`. A single request to the root, never a crawl.
54 | 	Path string `json:"path"`
55 | 	// BodyCapBytes bounds the body read — 65536 (64 KB). A response longer than
56 | 	// this is truncated to the cap and the identity records that it was.
57 | 	BodyCapBytes int `json:"body_cap_bytes"`
58 | 	// TimeoutMillis bounds the whole exchange — 10000 (10 s).
59 | 	TimeoutMillis int `json:"timeout_millis"`
```

### 57. `internal/measure/resolutionwalk/corpus/rows.go:216` — `section-divider`

Load-bearing: [ ]

```go
213 | 		}},
214 | 		"m2f_partly.ndjson"),
215 | 
216 | 	// ---- M2.g: set equality <-> serialisation (RR order + 0x20 case) ----
217 | 	{
218 | 		Cells:        []string{"M2.g/set"},
219 | 		Claim:        "an address set in canonical RR order and lower-case qname",
```

### 58. `internal/measure/resolutionwalk/corpus/rows.go:238` — `section-divider`

Load-bearing: [ ]

```go
235 | 		Golden: "m2g_serialisation.ndjson",
236 | 	},
237 | 
238 | 	// ---- M2.h: set equality <-> spelling (IPv4-mapped) ----
239 | 	one([]string{"M2.h/folds"},
240 | 		"an AAAA ::ffff:203.0.113.5 and an A 203.0.113.5 fold to one Address key",
241 | 		ScriptPeer{Rules: []scriptRule{
```

### 59. `internal/measure/tlsacceptance/corpus/harness.go:41-43` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
38 | 	return out, nil
39 | }
40 | 
41 | // CorpusDigest is a stable hash over the rendered corpus, in golden-filename order.
42 | // It moves exactly when a row's expected output moves, which binds an output change
43 | // to a leaf-version bump through the lock.
44 | func CorpusDigest(rendered map[string][]byte) string {
45 | 	names := make([]string, 0, len(rendered))
46 | 	for n := range rendered {
```

### 60. `internal/measure/tlsacceptance/corpus/rows.go:44-45` — `docstring-unexported`

Load-bearing: [ ]

```go
41 | 
42 | func candidates() ta.CandidateSet { return ta.DefaultCandidateSet() }
43 | 
44 | // scope builds a one-vantage scope over the given Services under the default
45 | // declared candidate set.
46 | func scope(services []ta.ServiceTarget) ta.Scope {
47 | 	return ta.Scope{
48 | 		Vantage:      "v1",
```

### 61. `internal/measure/tlsacceptance/corpus/rows.go:67` — `docstring-unexported`

Load-bearing: [ ]

```go
64 | 	ta.TLS12: {"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
65 | }}
66 | 
67 | // A listener that speaks TLS but accepts no candidate we offered.
68 | var refuses = listener{spoke: true, accepts: map[string][]string{}}
69 | 
70 | // Rows is the checked-in corpus. Every cell in AllCells appears in some row's Cells;
```

### 62. `internal/measure/tlsacceptance/corpus/rows.go:70-71` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
67 | // A listener that speaks TLS but accepts no candidate we offered.
68 | var refuses = listener{spoke: true, accepts: map[string][]string{}}
69 | 
70 | // Rows is the checked-in corpus. Every cell in AllCells appears in some row's Cells;
71 | // the coverage test fails the build (naming the cell) if one does not.
72 | var Rows = []Row{
73 | 	// ---- T1/modern-1.2-1.3 ----
74 | 	{
```

### 63. `internal/measure/tlsacceptance/corpus/rows.go:89` — `section-divider`

Load-bearing: [ ]

```go
86 | 		Golden: "tls_modern.ndjson",
87 | 	},
88 | 
89 | 	// ---- T2/tls-1.0-accepted ----
90 | 	{
91 | 		Cells:        []string{"T2/tls-1.0-accepted"},
92 | 		Claim:        "a listener accepting TLS 1.0 records version 1.0 in the value — the finding that reads the v1 signal tls-1.0-accepted (measurement-offers §1.2)",
```

### 64. `internal/measure/tlsacceptance/corpus/rows.go:105` — `section-divider`

Load-bearing: [ ]

```go
102 | 		Golden: "tls_1_0_accepted.ndjson",
103 | 	},
104 | 
105 | 	// ---- T3/tls-refused ----
106 | 	{
107 | 		Cells:        []string{"T3/tls-refused"},
108 | 		Claim:        "a peer that spoke TLS and accepted nothing offered is `tls-refused`, carrying no accepted versions — read with the batch's candidate set it is *the peer spoke TLS and refused all of this*",
```

### 65. `internal/measure/wildcarddiscrim/corpus/harness.go:37` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
34 | 	return buf.Bytes(), nil
35 | }
36 | 
37 | // RenderAll renders every row, keyed by its golden filename.
38 | func RenderAll() (map[string][]byte, error) {
39 | 	out := make(map[string][]byte, len(Rows))
40 | 	for _, r := range Rows {
```

### 66. `internal/measure/wildcarddiscrim/corpus/rows.go:82-83` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
79 | 	}
80 | }
81 | 
82 | // Rows is the checked-in corpus. Every cell in AllCells appears in some row's
83 | // Cells; A5 fails the build (naming the cell) if one does not.
84 | var Rows = []Row{
85 | 	// ---- W3c/no-wildcard + W1/NoSynthesis + W2/not-Shadowed + W5/shared-path ----
86 | 	one([]string{"W1/NoSynthesis", "W2/not-Shadowed", "W3c/no-wildcard", "W5/shared-path"},
```

### 67. `internal/measure/wildcarddiscrim/emit.go:18` — `docstring-unexported`

Load-bearing: [ ]

```go
15 | 	OutcomeGap      = "Gap"
16 | )
17 | 
18 | // resolutionValue is the JSON payload of a composed resolution observation.
19 | type resolutionValue struct {
20 | 	Outcome   string   `json:"outcome"`
21 | 	Addresses []string `json:"addresses,omitempty"`
```

### 68. `internal/message/artifactdoc.go:56-57` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
53 | 	Tone string `json:"tone"`
54 | }
55 | 
56 | // ArtifactDocSevBar is one bar in the "open signals by severity" breakdown: the
57 | // severity token (selects the dot colour), its label, the fill percentage, and count.
58 | type ArtifactDocSevBar struct {
59 | 	Sev   string `json:"sev"`
60 | 	Label string `json:"label"`
```

### 69. `internal/message/census.go:32` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
29 | 	// "signal". It is display-only — the census is a flat enumerable payload, not
30 | 	// a tree to walk.
31 | 	Kind string `json:"kind"`
32 | 	// Key is the facet name, subject key or rule name that opened.
33 | 	Key string `json:"key"`
34 | }
35 | 
```

### 70. `internal/message/flagship.go:27` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
24 | // flagship predicate reads. It carries the leg's class, the from/to values of
25 | // the transition, and whether the leg opened at `reached` rather than moving.
26 | type ReachMove struct {
27 | 	// ServiceKey is the (address:port/transport) key the leg was measured on.
28 | 	ServiceKey string
29 | 	// Class is the Vantage class the leg was measured from.
30 | 	Class VantageClass
```

### 71. `internal/message/message.go:136-137` — `docstring-exported-conventional`

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

### 72. `internal/message/message.go:175-176` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
172 | // LinkKind is the row's link target kind, derived from the cause on read.
173 | func (m Message) LinkKind() LinkKind { return LinkKindForCause(m.Cause) }
174 | 
175 | // CensusLen is the census size, or zero where the message carries none — the
176 | // count a row renders beside a flagship or membership headline.
177 | func (m Message) CensusLen() int {
178 | 	if m.Census == nil {
179 | 		return 0
```

### 73. `internal/message/pdf.go:120-122` — `docstring-unexported`

Load-bearing: [ ]

```go
117 | 	return items
118 | }
119 | 
120 | // artifactChangeItems builds one titled change section — its eyebrow, then a row
121 | // per change, or a single muted note when nothing moved (the same fact-stating
122 | // empty behaviour as the HTML render).
123 | func artifactChangeItems(title string, changes []ArtifactChange, emptyNote string) []artifactPDFItem {
124 | 	items := []artifactPDFItem{{role: roleEyebrow, text: title}}
125 | 	if len(changes) == 0 {
```

### 74. `internal/proposer/arin.go:39-40` — `docstring-unexported`

Load-bearing: [ ]

```go
36 | 
37 | func (a *ARIN) Slug() string { return SlugARIN }
38 | 
39 | // maxARINBody caps a single RDAP body read. A busy org's entity runs to tens of
40 | // kilobytes; this leaves generous headroom while refusing an unbounded read.
41 | const maxARINBody = 8 << 20
42 | 
43 | // The three ARIN object classes an org-name search can match. Only an org and a
```

### 75. `internal/proposer/caida.go:38-39` — `docstring-unexported`

Load-bearing: [ ]

```go
35 | 
36 | func (c *CAIDA) Slug() string { return c.slug }
37 | 
38 | // caidaOrgIDs is CAIDA's org->opaque-id answer: the registry holder ids the
39 | // searched name maps to.
40 | type caidaOrgIDs struct {
41 | 	OpaqueIDs []string `json:"opaque_ids"`
42 | }
```

### 76. `internal/proposer/proposer.go:164` — `docstring-unexported`

Load-bearing: [ ]

```go
161 | 	return new(big.Int).Lsh(big.NewInt(1), tz)
162 | }
163 | 
164 | // largestPow2AtMost returns the largest power of two that is <= n.
165 | func largestPow2AtMost(n *big.Int, bits int) *big.Int {
166 | 	if n.Sign() <= 0 {
167 | 		return big.NewInt(0)
```

### 77. `internal/proposer/proposer.go:185-186` — `docstring-unexported`

Load-bearing: [ ]

```go
182 | 	return new(big.Int).SetBytes(b[:])
183 | }
184 | 
185 | // addrAdd returns addr + delta, staying in addr's family. It reports false on
186 | // overflow past the family's last address.
187 | func addrAdd(addr netip.Addr, delta *big.Int) (netip.Addr, bool) {
188 | 	v := new(big.Int).Add(addrToInt(addr), delta)
189 | 	if addr.Is4() {
```

### 78. `internal/qr/qr.go:46-49` — `docstring-unexported`

Load-bearing: [ ]

```go
43 | // fall back to showing the payload as text.
44 | var ErrTooLong = errors.New("qr: payload too long for a version-10 symbol")
45 | 
46 | // ecBlocks describes, for one version at error-correction level M, how the data
47 | // codewords split into blocks and how many EC codewords each block carries.
48 | // Group 2 blocks (when present) hold one more data codeword than group 1; a
49 | // version with a uniform block layout leaves the group-2 fields zero.
50 | type ecBlocks struct {
51 | 	totalDataCW  int   // data codewords across all blocks
52 | 	ecPerBlock   int   // EC codewords per block (same for every block)
```

### 79. `internal/queue/crtsh.go:31-33` — `docstring-unexported`

Load-bearing: [ ]

```go
28 | // This file holds the throttled fetcher, the throttle, the dispatcher's fan-out
29 | // and the worker's completion path.
30 | 
31 | // maxCTBody bounds a crt.sh response read into memory. The documented 999-row cap
32 | // already bounds the answer, but a defensive ceiling keeps a misbehaving or
33 | // oversized response from exhausting the worker.
34 | const maxCTBody = 64 << 20 // 64 MiB
35 | 
36 | // crtshInterval is the instance-wide spacing between crt.sh requests: 12s, the
```

### 80. `internal/queue/hot.go:243-246` — `docstring-unexported`

Load-bearing: [ ]

```go
240 | 	return vergecore.Default().WithFrequencyEdits(fe), nil
241 | }
242 | 
243 | // enqueueHotJob enqueues one connect-outcome job for one Vantage. Its recorded
244 | // scope carries the admitted addresses and the verge-core port sets by content;
245 | // its offers carry the safety profile. It retries like a dns job — a connect is
246 | // a network step that can transiently fail.
247 | func enqueueHotJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.HotJob) error {
248 | 	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:vantage:%d:addr:%s", scanID, j.VantageID, jobAddr(j.Addresses)))
249 | 	if err != nil {
```

### 81. `internal/queue/membership.go:451-454` — `docstring-unexported`

Load-bearing: [ ]

```go
448 | 	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(d)), ".")
449 | }
450 | 
451 | // serviceAddress extracts the address limb of a Service/Endpoint subject key. A
452 | // Service is keyed "address:port" (an IP and a port); the address is everything
453 | // before the last colon, which leaves a bracketed or bare IPv6 host intact for the
454 | // caller's netip.ParseAddr to accept or reject.
455 | func serviceAddress(key string) string {
456 | 	if i := strings.LastIndex(key, ":"); i >= 0 {
457 | 		host := key[:i]
```

### 82. `internal/queue/nameseedwithdrawal.go:200` — `docstring-unexported`

Load-bearing: [ ]

```go
197 | 	return spanIDs, order, counts
198 | }
199 | 
200 | // pendingWithdrawnDomains is the bound the shared candidate query takes.
201 | func pendingWithdrawnDomains(pending []db.ListPendingNameSeedWithdrawalsRow) []string {
202 | 	out := make([]string, 0, len(pending))
203 | 	for _, w := range pending {
```

### 83. `internal/queue/produce.go:126-128` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
123 | 	// Reason is the drift.ClosureReason the estate fold decided (descoped /
124 | 	// measured-absent), stringified as it is stored on the closed span.
125 | 	Reason string
126 | 	// SourceKey is the declared-input identity the row links to — the covering
127 | 	// Exclusion's declared value for a `descoped` departure, empty for a world
128 | 	// (`measured-absent`) withdrawal that fires no declared-input message.
129 | 	SourceKey string
130 | 	// Timelines is the count of open timelines the withdrawal closed at once
131 | 	// (ADR-0082) — one cause recorded on n objects, the count the headline states.
```

### 84. `internal/queue/progress.go:77-78` — `docstring-unexported`

Load-bearing: [ ]

```go
74 | 	return fmt.Sprintf("attempt %d failed · %s · retrying", failedAttempt, redactCause(cause))
75 | }
76 | 
77 | // deadLetterLabel is the redacted text a dead-letter rides: the attempts spent and the safe
78 | // reason — "dead-lettered after 5 attempts · crt.sh returned HTTP 502".
79 | func deadLetterLabel(attempts int32, cause error) string {
80 | 	return fmt.Sprintf("dead-lettered after %d attempts · %s", attempts, redactCause(cause))
81 | }
```

### 85. `internal/queue/reaper.go:125` — `docstring-unexported`

Load-bearing: [ ]

```go
122 | 	}
123 | }
124 | 
125 | // compile-time proof *db.Queries is a ReaperStore.
126 | var _ ReaperStore = (*db.Queries)(nil)
127 | 
```

### 86. `internal/release/fetcher.go:38-40` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
35 | 	Do(req *http.Request) (*http.Response, error)
36 | }
37 | 
38 | // HTTPFetcher fetches the latest release from a JSON feed URL. It parses the
39 | // GitHub latest-release shape (tag_name + body); a custom feed mirrors those two
40 | // fields.
41 | type HTTPFetcher struct {
42 | 	url    string
43 | 	client Doer
```

### 87. `internal/release/release.go:63` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
60 | 	Latest(ctx context.Context) (Feed, error)
61 | }
62 | 
63 | // Checker runs the daily release check and records its verdict.
64 | type Checker struct {
65 | 	store   Store
66 | 	fetch   Fetcher
```

### 88. `internal/report/notify.go:72` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
69 | 	}
70 | }
71 | 
72 | // MarshalReadyBody renders the ready-message to the exact bytes POSTed and signed.
73 | func MarshalReadyBody(b ReadyBody) ([]byte, error) { return json.Marshal(b) }
74 | 
75 | // shouldNotify is the enqueue predicate: a scheduled run enqueues exactly one
```

### 89. `internal/report/notify.go:233-235` — `docstring-unexported`

Load-bearing: [ ]

```go
230 | 	return tx.Commit(ctx)
231 | }
232 | 
233 | // notifyError renders the failure string stored on the notification for the
234 | // channel-surface drill-down. A transport error carries its own message; a non-2xx
235 | // carries its status.
236 | func notifyError(statusCode int, sendErr error) string {
237 | 	if sendErr != nil {
238 | 		return sendErr.Error()
```

### 90. `internal/scan/cttail.go:335` — `docstring-unexported`

Load-bearing: [ ]

```go
332 | 	// ctLeafHeader is version(1) + leaf_type(1) + timestamp(8) + entry_type(2): the
333 | 	// fixed prefix before the signed_entry (§3.4).
334 | 	ctLeafHeader = 1 + 1 + 8 + 2
335 | 	// ctASN1CertLen is the length prefix width of an ASN.1Cert opaque<1..2^24-1>.
336 | 	ctASN1CertLen = 3
337 | 	// ctIssuerKeyHash is the fixed width of PreCert.issuer_key_hash (a SHA-256).
338 | 	ctIssuerKeyHash = 32
```

### 91. `internal/scan/zone.go:422-424` — `docstring-unexported`

Load-bearing: [ ]

```go
419 | 	return line
420 | }
421 | 
422 | // isTTL reports whether a field is a bare TTL — all digits, or digits with a
423 | // trailing time unit (e.g. 3600, 1h, 2d). It is how the parser tells a TTL
424 | // prefix from the record type.
425 | func isTTL(f string) bool {
426 | 	if f == "" {
427 | 		return false
```

### 92. `internal/signal/rules.go:44-46` — `docstring-unexported`

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

### 93. `internal/signal/severity.go:26-28` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
23 | 	SevInfo     Severity = "info"
24 | )
25 | 
26 | // SevOrder is the ramp from most urgent to least — critical → info — matching
27 | // SignalData.jsx's `SEV_ORDER` exactly. A severity-ordered view sorts by each
28 | // severity's index here.
29 | var SevOrder = []Severity{SevCritical, SevHigh, SevMedium, SevLow, SevInfo}
30 | 
31 | // Rank is the severity's position in SevOrder — 0 for critical (most urgent),
```

### 94. `internal/signal/signal.go:35` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
32 | 	OutsideDomain Outcome = "outside-domain"
33 | 	// Fired: the predicate is true of the subject.
34 | 	Fired Outcome = "fired"
35 | 	// NotFired: the subject is in the domain and the predicate is false.
36 | 	NotFired Outcome = "not-fired"
37 | 	// NotEvaluable: the subject is in the domain but the rule cannot read the
38 | 	// answer — the evidence is a value about our own sight (`Shadowed`) or there
```

### 95. `internal/transcript/key.go:85-86` — `docstring-unexported`

Load-bearing: [ ]

```go
82 | 	return nil
83 | }
84 | 
85 | // validateKey rejects a key file whose length is not keyLen as corruption, rather
86 | // than using truncated key material.
87 | func validateKey(path string, key []byte) ([]byte, error) {
88 | 	if len(key) != keyLen {
89 | 		return nil, fmt.Errorf("transcript: key %s is %d bytes, want %d", path, len(key), keyLen)
```

### 96. `internal/vantage/hostkey.go:16` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
13 | // new key (v1 spec §4.2).
14 | var ErrHostKeyMismatch = errors.New("vantage: host key mismatch — refusing to re-trust")
15 | 
16 | // HostKeyResult is the trust-on-first-use decision for a presented host key.
17 | type HostKeyResult int
18 | 
19 | const (
```

### 97. `internal/vergecore/vergecore.go:152-154` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
149 | 	return ok
150 | }
151 | 
152 | // Union is the whole set, `frequency ∪ sensitive`, deduplicated and sorted. It
153 | // is the recorded scope of the hot Scan — every pair a `Service` subject exists
154 | // for, open or closed.
155 | func (l List) Union() []Pair {
156 | 	seen := map[Pair]struct{}{}
157 | 	for p := range l.frequency {
```

### 98. `internal/wire/transcript.go:162` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
159 | // ZoneParsed is a restate that parsed the zone file.
160 | type ZoneParsed struct{}
161 | 
162 | // ZoneDecodeError is a restate that hit a decode error, carrying the error text.
163 | type ZoneDecodeError struct{ Text string }
164 | 
165 | func (ZoneParsed) isZoneOutcome()      {}
```

### 99. `internal/wire/wire.go:37-39` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
34 | 	// surfaces bufio.ErrTooLong via Err() rather than over-allocating.
35 | 	MaxObservationLine = 1 << 20 // 1 MiB per line
36 | 
37 | 	// MaxObservations caps how many observation lines one job may yield. Even
38 | 	// under MaxProberStdout a flood of tiny lines would grow the decoded slice
39 | 	// without bound; this bounds the entry COUNT too.
40 | 	MaxObservations = 1 << 20
41 | )
42 | 
```

### 100. `internal/wire/wire.go:208-209` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
205 | 	return nil
206 | }
207 | 
208 | // ObservationScanner reads NDJSON observations from a prober's stdout, one
209 | // per call to Next.
210 | type ObservationScanner struct {
211 | 	scanner *bufio.Scanner
212 | 	obs     Observation
```

