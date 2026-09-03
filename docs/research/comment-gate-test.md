# Comment policy validation gate — test Go, round 2

SPEC `docs/spec/comment-policy.md` §3.9. Regenerate this sheet with:

```sh
go run ./cmd/commentlint sample --population test --round 2
```

- In-scope Go files read: 221
- Blocks the §3.2 screen admits for deletion: 404
- Blocks drawn into the gate sample: 100
- Blocks drawn into the coverage supplement: 0

Accept a class at 2 or fewer load-bearing blocks. A class that fails three rounds leaves the v1 delete set and stays in the flag set.

This sheet holds the current round. Every round's verdicts sit in [the round ledger](comment-gate-rounds.md).

## Class coverage

| Class | Admitted | Gate sample | Supplement |
| --- | --- | --- | --- |
| `commented-out-code` | 0 | 0 | 0 |
| `docstring-exported-conventional` | 72 | 18 | 0 |
| `docstring-unexported` | 177 | 40 | 0 |
| `section-divider` | 31 | 13 | 0 |
| `short-label` | 124 | 29 | 0 |

## Verdicts

A reviewer fills the last two columns. A class that admits no block on this population reads `n/a`.

| Class | Read | Load-bearing | Verdict |
| --- | --- | --- | --- |
| `commented-out-code` | 0 | n/a | n/a |
| `docstring-exported-conventional` | 18 | | |
| `docstring-unexported` | 40 | | |
| `section-divider` | 13 | | |
| `short-label` | 29 | | |

## Gate sample

### 1. `cmd/web/addresscap_test.go:146-148` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
143 | 	}
144 | }
145 | 
146 | // TestAddressCapControlPricesTheCost covers the Variant C readout on the Scans tab: the
147 | // largest scope the cap admits, the per-cadence sweep load on each enabled address-scope
148 | // scan, and the projected evidential disk growth.
149 | func TestAddressCapControlPricesTheCost(t *testing.T) {
150 | 	f := newFakeStore()
151 | 	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
```

### 2. `cmd/web/adr0130_contract_test.go:29` — `section-divider`

Load-bearing: [ ]

```go
26 | // the record of a case ADR-0130 does not cover, with its reason stated at the entry, so
27 | // a reviewer sees the whole set at once and a new entry has to argue for itself.
28 | 
29 | // --- the source and the route table these guards read ----------------------
30 | 
31 | // contractPkg is cmd/web parsed from disk, minus the tests.
32 | type contractPkg struct {
```

### 3. `cmd/web/adr0130_contract_test.go:31` — `docstring-unexported`

Load-bearing: [ ]

```go
28 | 
29 | // --- the source and the route table these guards read ----------------------
30 | 
31 | // contractPkg is cmd/web parsed from disk, minus the tests.
32 | type contractPkg struct {
33 | 	fset *token.FileSet
34 | 	// methods maps a *server method name to its declaration. A handler, a helper and a
```

### 4. `cmd/web/adr0130_contract_test.go:225` — `docstring-unexported`

Load-bearing: [ ]

```go
222 | 	return out
223 | }
224 | 
225 | // decl resolves a callee name from calleesOf back to its declaration, method or function.
226 | func (c *contractPkg) decl(name string) (*ast.FuncDecl, bool) {
227 | 	if m, ok := strings.CutPrefix(name, "s."); ok {
228 | 		fn, found := c.methods[m]
```

### 5. `cmd/web/api_auth_test.go:39-40` — `docstring-unexported`

Load-bearing: [ ]

```go
36 | 	return rec, got
37 | }
38 | 
39 | // seedAPIToken enables the API surface, seeds an account, and mints a token for it whose
40 | // stored hash matches apiTokenPlaintext. It returns the account and the token id.
41 | func seedAPIToken(t *testing.T, f *fakeStore, role string) (db.Account, int64) {
42 | 	t.Helper()
43 | 	f.instanceConfig = db.GetInstanceConfigRow{ApiEnabled: true}
```

### 6. `cmd/web/auth_test.go:186` — `section-divider`

Load-bearing: [ ]

```go
183 | 	}
184 | }
185 | 
186 | // --- login / session -------------------------------------------------------
187 | 
188 | func TestLoginAndSession(t *testing.T) {
189 | 	f := newFakeStore()
```

### 7. `cmd/web/auth_test.go:299` — `section-divider`

Load-bearing: [ ]

```go
296 | 	}
297 | }
298 | 
299 | // --- permission check on the mutating endpoint -----------------------------
300 | 
301 | func TestViewerDeniedMutation(t *testing.T) {
302 | 	f := newFakeStore()
```

### 8. `cmd/web/auth_test.go:336` — `short-label`

Load-bearing: [ ]

```go
333 | 		t.Fatalf("anon mutation: status=%d, want 303 to login", resp.StatusCode)
334 | 	}
335 | 
336 | 	// An admin may perform it.
337 | 	ac := login(t, base, "admin", "hunter2hunter2")
338 | 	// The create is a post-redirect-get (ADR-0130 §3, #974): the confirmation line rides
339 | 	// the session flash to the landing GET, so the 303 itself carries no body.
```

### 9. `cmd/web/auth_test.go:361` — `section-divider`

Load-bearing: [ ]

```go
358 | 	}
359 | }
360 | 
361 | // --- TOTP ------------------------------------------------------------------
362 | 
363 | // secretRE extracts the base32 secret from the frozen enroll screen's copy affordance
364 | // (signin.tmpl: <span class="val">SECRET</span>). Screen 4's design-owned markup replaced the
```

### 10. `cmd/web/backup_test.go:111` — `short-label`

Load-bearing: [ ]

```go
108 | 
109 | 	sc := bufio.NewScanner(&buf)
110 | 
111 | 	// Line 1: manifest.
112 | 	if !sc.Scan() {
113 | 		t.Fatal("no manifest line")
114 | 	}
```

### 11. `cmd/web/backurl_test.go:263` — `docstring-unexported`

Load-bearing: [ ]

```go
260 | 	base := start(t, f, "")
261 | 	ac := login(t, base, "admin", "hunter2hunter2")
262 | 
263 | 	// The declare form, opened on an open row's Drawer under a filtered view.
264 | 	const view = "/signals?tab=open&q=lame"
265 | 	list := getBody(t, ac, base+view, http.StatusOK)
266 | 	key := firstViewKey(t, list)
```

### 12. `cmd/web/backurl_test.go:268` — `docstring-unexported`

Load-bearing: [ ]

```go
265 | 	list := getBody(t, ac, base+view, http.StatusOK)
266 | 	key := firstViewKey(t, list)
267 | 	drawer := getBody(t, ac, base+view+"&view="+url.QueryEscape(key), http.StatusOK)
268 | 	// The query keeps its own order, and html/template escapes the `&` in an attribute.
269 | 	const wantField = `name="return" value="/signals?tab=open&amp;q=lame&amp;view=`
270 | 	if !strings.Contains(drawer, wantField) {
271 | 		t.Errorf("the declare form carries no submitting-URL field starting %q; body: %s", wantField, drawer)
```

### 13. `cmd/web/credflow_sessions_test.go:47-48` — `docstring-unexported`

Load-bearing: [ ]

```go
44 | // (countLiveSessions lives in settings_sessions_test.go — reused here to prove a revoke
45 | // landed in the registry.)
46 | 
47 | // bounced reports whether a client's next authed request is redirected to /login —
48 | // the observable signature of a session whose registry row was revoked.
49 | func bounced(t *testing.T, c *http.Client, base string) bool {
50 | 	t.Helper()
51 | 	resp, err := c.Get(base + "/profile")
```

### 14. `cmd/web/custodycensus_test.go:61-63` — `docstring-unexported`

Load-bearing: [ ]

```go
58 | 	return der
59 | }
60 | 
61 | // censusEstateFixture poses a custody-extended zone whose two in-zone names front two
62 | // separate edges, and puts the `edge-fanout` Scan in force. The measurements are the
63 | // caller's.
64 | func censusEstateFixture(t *testing.T, f *fakeStore) {
65 | 	t.Helper()
66 | 	f.scans = append(f.scans, db.Scan{ID: 99, Kind: scan.EdgeFanoutKind, Enabled: true, CadenceSeconds: 86400})
```

### 15. `cmd/web/devfixtures_test.go:66-67` — `docstring-unexported`

Load-bearing: [ ]

```go
63 | 	}
64 | }
65 | 
66 | // fixtureProfilePackage mirrors the design-owned fixtures.json → profile slice (plus the
67 | // top-level clock) the screen-3 seeder pins in devfixtures.go.
68 | type fixtureProfilePackage struct {
69 | 	Clock   string `json:"clock"`
70 | 	Profile struct {
```

### 16. `cmd/web/devfixtures_test.go:134` — `short-label`

Load-bearing: [ ]

```go
131 | 		t.Fatalf("parse devFixtureClock: %v", err)
132 | 	}
133 | 
134 | 	// Account.
135 | 	if p.Account.Username != devProfileUsername {
136 | 		t.Errorf("account username drift: %q vs %q", p.Account.Username, devProfileUsername)
137 | 	}
```

### 17. `cmd/web/devfixtures_test.go:571` — `short-label`

Load-bearing: [ ]

```go
568 | 	}
569 | 	d := f.Drift
570 | 
571 | 	// Trigger + tally scalars.
572 | 	if d.Period != devDriftPeriod {
573 | 		t.Errorf("period drift: fixtures.json = %q, pinned = %q", d.Period, devDriftPeriod)
574 | 	}
```

### 18. `cmd/web/devfixtures_test.go:632` — `short-label`

Load-bearing: [ ]

```go
629 | 		}
630 | 	}
631 | 
632 | 	// Movement map.
633 | 	if len(d.Movement) != len(devDriftMovement) {
634 | 		t.Fatalf("movement length drift: fixtures.json = %d, pinned = %d", len(d.Movement), len(devDriftMovement))
635 | 	}
```

### 19. `cmd/web/devfixtures_test.go:672-675` — `docstring-unexported`

Load-bearing: [ ]

```go
669 | 	}
670 | }
671 | 
672 | // fixtureRunDetailPackage mirrors the fixtures.json rundetail slice the screen-9 dev fixture pins
673 | // (devfixtures.go, served by runPage under devMode): the run header + Outcome figures, the four
674 | // stages, the seven log lines, the nullable degraded callout, the five params and the three
675 | // vantages. Snake_case JSON → the runView PascalCase the frozen rundetail.tmpl reads.
676 | type fixtureRunDetailPackage struct {
677 | 	RunDetail struct {
678 | 		ID          string `json:"id"`
```

### 20. `cmd/web/devfixtures_test.go:1101-1106` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
1098 | 	}
1099 | }
1100 | 
1101 | // TestSignalsFixtureMatchesPackage is the byte-exactness gate before the pixels: it folds the pinned
1102 | // dev Signals fixture (cmd/web/devfixtures.go) back through the frozen design package
1103 | // (design-system/fixtures/fixtures.json → signals) and fails the build on any divergence — the open
1104 | // scalars, the ten open + three withdrawn rows (with rule metadata), the annotations, the drift
1105 | // diffs, and the two span-history literals the derivation depends on. It guards the same seam
1106 | // TestScopeFixtureMatchesPackage guards for Scope.
1107 | func TestSignalsFixtureMatchesPackage(t *testing.T) {
1108 | 	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
1109 | 	if err != nil {
```

### 21. `cmd/web/devfixtures_test.go:1249-1250` — `docstring-unexported`

Load-bearing: [ ]

```go
1246 | 	} `json:"signals"`
1247 | }
1248 | 
1249 | // dashCountedStr renders a fixtures.json coverage-meter counted RawMessage as the pinned string:
1250 | // a JSON string is unquoted ("1,284"), a JSON number is used verbatim ("212").
1251 | func dashCountedStr(raw json.RawMessage) string {
1252 | 	s := strings.TrimSpace(string(raw))
1253 | 	if len(s) >= 1 && s[0] == '"' {
```

### 22. `cmd/web/devfixtures_test.go:1726` — `short-label`

Load-bearing: [ ]

```go
1723 | 	assertServiceFixture(t, "service", f.SubjectDetail.Service, devServiceData())
1724 | 	assertServiceFixture(t, "service_withdrawn", f.SubjectDetail.ServiceWithdrawn, devServiceWithdrawnData())
1725 | 
1726 | 	// Endpoint.
1727 | 	a, d := f.SubjectDetail.Endpoint, devEndpointData()
1728 | 	if a.Key != d.Key || a.CopyKey != d.CopyKey || a.Nameless != d.Nameless || a.Withdrawn != d.Withdrawn ||
1729 | 		a.Seen != d.Seen || a.InScopeSince != d.InScopeSince || a.CitationTerminated != d.CitationTerminated ||
```

### 23. `cmd/web/devfixtures_test.go:1764` — `docstring-unexported`

Load-bearing: [ ]

```go
1761 | 	}
1762 | }
1763 | 
1764 | // fixtureGraphPackage is the fixtures.json → graph slice, snake_case as stored.
1765 | type fixtureGraphPackage struct {
1766 | 	Graph struct {
1767 | 		Empty bool `json:"empty"`
```

### 24. `cmd/web/error_test.go:12-14` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 9 | 	"testing"
10 | )
11 | 
12 | // TestUnknownPathRendersNotFound proves an unmatched URL lands on the ported 404
13 | // error page (not the scaffold's plain-text NotFound): status 404, an HTML page
14 | // that names the state.
15 | func TestUnknownPathRendersNotFound(t *testing.T) {
16 | 	f := newFakeStore()
17 | 	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
```

### 25. `cmd/web/exclusions_test.go:134` — `short-label`

Load-bearing: [ ]

```go
131 | 
132 | 	vc := login(t, base, "viewer", "hunter2hunter2")
133 | 
134 | 	// The viewer is denied both mutations.
135 | 	resp := exclude(t, vc, base, "name", "nope.example.com")
136 | 	resp.Body.Close()
137 | 	if resp.StatusCode != http.StatusForbidden {
```

### 26. `cmd/web/handlers_test.go:29-30` — `docstring-unexported`

Load-bearing: [ ]

```go
26 | 	"github.com/winniel123/verge-asm/internal/retention"
27 | )
28 | 
29 | // fakeStore is an in-memory store used across the web handler tests, standing
30 | // in for a live Postgres.
31 | type fakeStore struct {
32 | 	hb    db.Heartbeat
33 | 	hbErr error
```

### 27. `cmd/web/handlers_test.go:269` — `docstring-unexported`

Load-bearing: [ ]

```go
266 | 	createdAt   time.Time
267 | }
268 | 
269 | // fakeFreqEdit mirrors a verge-core frequency edit row.
270 | type fakeFreqEdit struct {
271 | 	action    string
272 | 	createdBy int64
```

### 28. `cmd/web/handlers_test.go:368` — `docstring-unexported`

Load-bearing: [ ]

```go
365 | 	return db.Transcript{}, pgx.ErrNoRows
366 | }
367 | 
368 | // dispatchIdx finds the progress row for a dispatch id, or -1.
369 | func (f *fakeStore) dispatchIdx(id int64) int {
370 | 	for i := range f.dispatchProgress {
371 | 		if f.dispatchProgress[i].DispatchID == id {
```

### 29. `cmd/web/handlers_test.go:378-379` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
375 | 	return -1
376 | }
377 | 
378 | // CancelReadyJobsForDispatch cancels the pending (ready) jobs of a dispatch — the stop
379 | // act — moving that count into the cancelled bucket (ready → 0) and returning it.
380 | func (f *fakeStore) CancelReadyJobsForDispatch(_ context.Context, dispatchID pgtype.Int8) (int64, error) {
381 | 	i := f.dispatchIdx(dispatchID.Int64)
382 | 	if i < 0 {
```

### 30. `cmd/web/handlers_test.go:412` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
409 | 	return nil
410 | }
411 | 
412 | // GetInstanceHealth returns the canned instance-health db facts.
413 | func (f *fakeStore) GetInstanceHealth(context.Context) (db.GetInstanceHealthRow, error) {
414 | 	return f.instanceHealth, nil
415 | }
```

### 31. `cmd/web/handlers_test.go:1025` — `short-label`

Load-bearing: [ ]

```go
1022 | 
1023 | func (f *fakeStore) ListAnnotations(context.Context) ([]db.Annotation, error) {
1024 | 	rows := append([]db.Annotation(nil), f.annotations...)
1025 | 	// ORDER BY signal_name, subject_key.
1026 | 	sort.Slice(rows, func(i, j int) bool {
1027 | 		if rows[i].SignalName != rows[j].SignalName {
1028 | 			return rows[i].SignalName < rows[j].SignalName
```

### 32. `cmd/web/handlers_test.go:1100` — `short-label`

Load-bearing: [ ]

```go
1097 | 
1098 | func (f *fakeStore) ListMessages(context.Context) ([]db.Message, error) {
1099 | 	out := make([]db.Message, len(f.messages))
1100 | 	// Newest-first, mirroring ORDER BY id DESC.
1101 | 	for i, m := range f.messages {
1102 | 		out[len(f.messages)-1-i] = m
1103 | 	}
```

### 33. `cmd/web/handlers_test.go:1349-1351` — `docstring-unexported`

Load-bearing: [ ]

```go
1346 | 	return b.ID
1347 | }
1348 | 
1349 | // addResolution records a resolution observation for a Name in a fresh batch of
1350 | // the given scan kind, mirroring what the measurement worker writes. It is the
1351 | // only seam the Subjects tests need to populate the estate.
1352 | func (f *fakeStore) addResolution(t *testing.T, createdBy int64, name, scanKind string, at time.Time, value string) {
1353 | 	t.Helper()
1354 | 	b := f.freshBatch(scanKind, "resolution-walk")
```

### 34. `cmd/web/handlers_test.go:2347` — `docstring-unexported`

Load-bearing: [ ]

```go
2344 | 
2345 | func (f *fakeStore) FindNameCitingAddress(_ context.Context, arg db.FindNameCitingAddressParams) (db.FindNameCitingAddressRow, error) {
2346 | 	address := arg.Address
2347 | 	// The earliest current resolution whose Resolved answer names the address.
2348 | 	var best *db.FindNameCitingAddressRow
2349 | 	for name, o := range f.latestResolutionByName(f.liveObservations(arg.AsOf.Time)) {
2350 | 		if fakeResolutionOutcome(o.Value) != "Resolved" {
```

### 35. `cmd/web/handlers_test.go:2439-2440` — `docstring-unexported`

Load-bearing: [ ]

```go
2436 | 	return db.Vantage{}
2437 | }
2438 | 
2439 | // addClassResolution records a resolution observation at a Vantage of the given
2440 | // class — the Signals reads need the class join the plain Subjects reads do not.
2441 | func (f *fakeStore) addClassResolution(t *testing.T, name, class string, at time.Time, value string) {
2442 | 	t.Helper()
2443 | 	vid := f.vantageForClass(class)
```

### 36. `cmd/web/handlers_test.go:2905` — `short-label`

Load-bearing: [ ]

```go
2902 | 
2903 | func (f *fakeStore) ListReportDeliveries(_ context.Context, scheduleID int64) ([]db.ReportDelivery, error) {
2904 | 	out := []db.ReportDelivery{}
2905 | 	// Newest-first, mirroring ORDER BY id DESC.
2906 | 	for i := len(f.reportDeliveries) - 1; i >= 0; i-- {
2907 | 		if f.reportDeliveries[i].ScheduleID == scheduleID {
2908 | 			out = append(out, f.reportDeliveries[i])
```

### 37. `cmd/web/handlers_test.go:3128-3129` — `docstring-unexported`

Load-bearing: [ ]

```go
3125 | 	return nil
3126 | }
3127 | 
3128 | // ssoSlugForID / ssoNameForID resolve a provider id to its slug/name for the identity
3129 | // join queries; an unknown id renders empty.
3130 | func (f *fakeStore) ssoSlugForID(id int64) string {
3131 | 	for _, p := range f.ssoProviders {
3132 | 		if p.id == id {
```

### 38. `cmd/web/hardening_test.go:142` — `short-label`

Load-bearing: [ ]

```go
139 | 	if respA.StatusCode != http.StatusSeeOther || !hasCookie(cA, base, sessionCookie) {
140 | 		t.Fatalf("first use of a valid code did not complete login: status=%d", respA.StatusCode)
141 | 	}
142 | 	// The replay watermark advanced.
143 | 	if got := f.accounts[acct.ID]; !got.TotpLastStep.Valid || got.TotpLastStep.Int64 == 0 {
144 | 		t.Fatalf("stored totp_last_step did not advance: %+v", got.TotpLastStep)
145 | 	}
```

### 39. `cmd/web/integrations_channel_test.go:24-25` — `docstring-unexported`

Load-bearing: [ ]

```go
21 | // (a reference, not a fold), and a "Send test" that POSTs a real payload through the
22 | // bound Channel's transport. They run only when the surface is live (integrationsEnabled).
23 | 
24 | // fakeChannelSender is the test double for the Send-test egress seam: it records the
25 | // call and returns a scripted status without ever touching the network.
26 | type fakeChannelSender struct {
27 | 	calls      int
28 | 	lastURL    string
```

### 40. `cmd/web/integrations_test.go:250` — `short-label`

Load-bearing: [ ]

```go
247 | 		}
248 | 	}
249 | 
250 | 	// Install persists real state.
251 | 	resp := postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"pagerduty"}})
252 | 	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=integrations" {
253 | 		t.Fatalf("install: status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
```

### 41. `cmd/web/messages_test.go:17-18` — `docstring-unexported`

Load-bearing: [ ]

```go
14 | 	"github.com/winniel123/verge-asm/internal/message"
15 | )
16 | 
17 | // putMessage inserts a computed message straight into the fake store, standing
18 | // in for the cause path that would have written it.
19 | func putMessage(t *testing.T, f *fakeStore, cause message.Cause, subjectKind, firedAt, headline string, census []byte) db.Message {
20 | 	t.Helper()
21 | 	m, err := f.InsertMessage(t.Context(), db.InsertMessageParams{
```

### 42. `cmd/web/onboarding_test.go:82` — `short-label`

Load-bearing: [ ]

```go
79 | 		t.Fatalf("committed seed not carried forward; body: %s", cadence)
80 | 	}
81 | 
82 | 	// Advance cadence -> channel.
83 | 	channel := stepFollow(t, ac, base, url.Values{
84 | 		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"},
85 | 	})
```

### 43. `cmd/web/onboarding_test.go:90` — `short-label`

Load-bearing: [ ]

```go
87 | 		t.Fatalf("did not advance to channel step; body: %s", channel)
88 | 	}
89 | 
90 | 	// Advance channel -> review.
91 | 	review := stepFollow(t, ac, base, url.Values{
92 | 		"step": {"2"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"},
93 | 		"channel": {"https://ops.example/hook"},
```

### 44. `cmd/web/profile_test.go:155-157` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
152 | 	return nil
153 | }
154 | 
155 | // RevokeSession stamps revoked_at on the row scoped to its owner (id AND account_id),
156 | // only while it is still live. Idempotent: an already-revoked or foreign row is
157 | // untouched, mirroring the owner-scoped SQL.
158 | func (f *fakeStore) RevokeSession(_ context.Context, arg db.RevokeSessionParams) error {
159 | 	for i := range f.sessions {
160 | 		if f.sessions[i].ID == arg.ID && f.sessions[i].AccountID == arg.AccountID && !f.sessions[i].RevokedAt.Valid {
```

### 45. `cmd/web/profile_test.go:192-194` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
189 | 	return rows, nil
190 | }
191 | 
192 | // RevokeOtherSessionsForAccount revokes every live session for the account EXCEPT the
193 | // current one (arg.ID) — "sign out other devices" and the password-change invalidation.
194 | // The acting session survives, mirroring the id <> $2 predicate.
195 | func (f *fakeStore) RevokeOtherSessionsForAccount(_ context.Context, arg db.RevokeOtherSessionsForAccountParams) error {
196 | 	for i := range f.sessions {
197 | 		if f.sessions[i].AccountID == arg.AccountID && f.sessions[i].ID != arg.ID && !f.sessions[i].RevokedAt.Valid {
```

### 46. `cmd/web/profile_test.go:243-244` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
240 | 	return rows, nil
241 | }
242 | 
243 | // RevokeSessionByIDForAdmin revokes any one live session by id, NOT owner-scoped — the
244 | // admin single-revoke, gated by requireAdmin at the handler. Idempotent.
245 | func (f *fakeStore) RevokeSessionByIDForAdmin(_ context.Context, arg db.RevokeSessionByIDForAdminParams) error {
246 | 	for i := range f.sessions {
247 | 		if f.sessions[i].ID == arg.ID && !f.sessions[i].RevokedAt.Valid {
```

### 47. `cmd/web/profile_test.go:255` — `section-divider`

Load-bearing: [ ]

```go
252 | 	return nil
253 | }
254 | 
255 | // --- tests -----------------------------------------------------------------
256 | 
257 | func profileBase(t *testing.T) (*fakeStore, string, db.Account) {
258 | 	t.Helper()
```

### 48. `cmd/web/proposals_test.go:63-64` — `docstring-unexported`

Load-bearing: [ ]

```go
60 | 	return resp
61 | }
62 | 
63 | // twoCandidates is a delegation plus a compelled reassignment — the two record
64 | // kinds ARIN returns in one response.
65 | func twoCandidates() []proposer.Candidate {
66 | 	return []proposer.Candidate{
67 | 		{SourceSlug: proposer.SlugARIN, RecordKind: proposer.RecordRIRDelegation,
```

### 49. `cmd/web/proposals_test.go:494` — `short-label`

Load-bearing: [ ]

```go
491 | 			t.Errorf("viewer POST %s: status=%d, want 403", ep.path, resp.StatusCode)
492 | 		}
493 | 	}
494 | 	// Nothing the viewer did changed state.
495 | 	if len(f.seeds) != 0 {
496 | 		t.Errorf("viewer opened the gate: seeds=%d", len(f.seeds))
497 | 	}
```

### 50. `cmd/web/reports_test.go:507` — `short-label`

Load-bearing: [ ]

```go
504 | 		t.Fatalf("viewer's denied delete removed the row: %d left", len(f.reportSchedules))
505 | 	}
506 | 
507 | 	// The admin deletes it.
508 | 	ac := login(t, base, "admin", "hunter2hunter2")
509 | 	resp := postForm(t, ac, base+"/reports/schedule/delete", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
510 | 	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/reports" {
```

### 51. `cmd/web/scans_test.go:459` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
456 | 	}
457 | }
458 | 
459 | // TestDispatchBatchIDs collects the distinct valid batch ids a dispatch's jobs committed.
460 | func TestDispatchBatchIDs(t *testing.T) {
461 | 	rows := []db.ListJobsForDispatchRow{
462 | 		{ID: 1, BatchID: pgtype.Int8{Int64: 1407, Valid: true}},
```

### 52. `cmd/web/scope_bulk_test.go:21` — `docstring-unexported`

Load-bearing: [ ]

```go
18 | // checked end-to-end (every valid token becomes a seed); the onboarding side through
19 | // readOnboardView; and both are pinned to parseSeedTokens' own output.
20 | func TestScopeAndOnboardingShareTokenizer(t *testing.T) {
21 | 	// Commas, spaces, tabs and newlines all split; empty tokens (the double comma) drop.
22 | 	const raw = "a.com, b.com\n c.com\td.com,,e.com"
23 | 	want := []string{"a.com", "b.com", "c.com", "d.com", "e.com"}
24 | 
```

### 53. `cmd/web/scope_bulk_test.go:202-203` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
199 | 	}
200 | }
201 | 
202 | // TestDeleteSeedWritesRemovalFlash: /seeds/delete names the removed scope in a flash
203 | // (WORK-ORDER-DOGFOOD-R1 item 2).
204 | func TestDeleteSeedWritesRemovalFlash(t *testing.T) {
205 | 	f := newFakeStore()
206 | 	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
```

### 54. `cmd/web/scope_bulk_test.go:285` — `section-divider`

Load-bearing: [ ]

```go
282 | 	}
283 | }
284 | 
285 | // --- helpers ---------------------------------------------------------------
286 | 
287 | // declareScope posts a raw (possibly multi-token) scope string to the paste-split
288 | // declare endpoint.
```

### 55. `cmd/web/scope_bulk_test.go:287-288` — `docstring-unexported`

Load-bearing: [ ]

```go
284 | 
285 | // --- helpers ---------------------------------------------------------------
286 | 
287 | // declareScope posts a raw (possibly multi-token) scope string to the paste-split
288 | // declare endpoint.
289 | func declareScope(t *testing.T, c *http.Client, base, raw string) *http.Response {
290 | 	t.Helper()
291 | 	return postForm(t, c, base+"/seeds", url.Values{"scope": {raw}})
```

### 56. `cmd/web/settings_sessions_test.go:10-12` — `docstring-unexported`

Load-bearing: [ ]

```go
 7 | 	"testing"
 8 | )
 9 | 
10 | // liveSessionID returns the id of an account's single live (unrevoked) session in the
11 | // fake registry, failing the test if none is found. A signed-in account holds exactly
12 | // one session per login client.
13 | func liveSessionID(t *testing.T, f *fakeStore, accountID int64) int64 {
14 | 	t.Helper()
15 | 	for _, sess := range f.sessions {
```

### 57. `cmd/web/signals_test.go:560` — `short-label`

Load-bearing: [ ]

```go
557 | 		t.Fatalf("want 2 instances (one per fired pair), got %d", len(insts))
558 | 	}
559 | 
560 | 	// Severity ramp order: critical before medium.
561 | 	crit := insts[0]
562 | 	if crit.Signal != "sensitive-port-reached-from-internet" {
563 | 		t.Fatalf("critical instance should sort first, got %q", crit.Signal)
```

### 58. `cmd/web/signin_test.go:130` — `section-divider`

Load-bearing: [ ]

```go
127 | 	return pgx.ErrNoRows
128 | }
129 | 
130 | // --- test seeding helpers ---------------------------------------------------
131 | 
132 | // serverClock is the instant start()'s fixed clock is pinned to; expiry seeding is
133 | // relative to it so a test can mint a live or a deliberately stale grant.
```

### 59. `cmd/web/signin_test.go:261` — `section-divider`

Load-bearing: [ ]

```go
258 | 	}
259 | }
260 | 
261 | // --- TOTP enrollment + recovery codes ---------------------------------------
262 | 
263 | // recoveryCodeRE matches a recovery code only inside its reveal span, so it counts
264 | // the shown codes rather than any incidental text elsewhere on the page. Post-#338 a
```

### 60. `cmd/web/signin_test.go:429` — `docstring-unexported`

Load-bearing: [ ]

```go
426 | 	}
427 | }
428 | 
429 | // getAnon GETs a URL with no session and asserts the status, returning the body.
430 | func getAnon(t *testing.T, url string, want int) string {
431 | 	t.Helper()
432 | 	resp, err := http.Get(url)
```

### 61. `cmd/web/sources_test.go:17` — `section-divider`

Load-bearing: [ ]

```go
14 | 	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
15 | )
16 | 
17 | // --- fakeStore source-state methods ----------------------------------------
18 | 
19 | func (f *fakeStore) ListSourceStates(context.Context) ([]db.SourceState, error) {
20 | 	rows := make([]db.SourceState, 0, len(f.sourceStates))
```

### 62. `cmd/web/sources_test.go:333-334` — `docstring-unexported`

Load-bearing: [ ]

```go
330 | 
331 | // --- helpers ----------------------------------------------------------------
332 | 
333 | // sourcesTab is the Sources section's own tab, and the fallback destination of a toggle
334 | // submitted with no `return` field (backToSection).
335 | const sourcesTab = "/settings?tab=sources"
336 | 
337 | func toggleSourceReq(t *testing.T, c *http.Client, base, slug, enabled string) *http.Response {
```

### 63. `cmd/web/sources_test.go:366` — `short-label`

Load-bearing: [ ]

```go
363 | 
364 | 	page := sourcesBody(t, ac, base)
365 | 
366 | 	// Every catalogued source appears.
367 | 	for _, c := range sourceCatalog {
368 | 		if !strings.Contains(page, c.Name) {
369 | 			t.Errorf("source %q missing from the modal", c.Name)
```

### 64. `cmd/web/sso_test.go:74` — `docstring-unexported`

Load-bearing: [ ]

```go
71 | 	})
72 | }
73 | 
74 | // stateFromRedirect parses the state parameter out of an IdP redirect Location.
75 | func stateFromRedirect(t *testing.T, loc string) string {
76 | 	t.Helper()
77 | 	u, err := url.Parse(loc)
```

### 65. `cmd/web/sso_test.go:345-347` — `docstring-unexported`

Load-bearing: [ ]

```go
342 | 	}
343 | }
344 | 
345 | // ssoLinkFlow drives an authenticated Profile self-link end to end: the link start
346 | // (303 to the IdP with a minted state) then the link callback echoing that state and a
347 | // code. It returns the callback response for the caller to assert on.
348 | func ssoLinkFlow(t *testing.T, ac *http.Client, base, slug string) *http.Response {
349 | 	t.Helper()
350 | 	r1, err := ac.Get(base + "/profile/sso/" + slug + "/link")
```

### 66. `cmd/web/sso_test.go:399` — `short-label`

Load-bearing: [ ]

```go
396 | 	addSSOProvider(f, 1, "okta", "Okta")
397 | 	base := startWithSSO(t, f, &fakeSSOFlow{sub: "okta-sub-alice", display: "alice@corp"})
398 | 
399 | 	// alice links from her Profile...
400 | 	ac := login(t, base, "alice", "unused-password-x")
401 | 	ssoLinkFlow(t, ac, base, "okta").Body.Close()
402 | 
```

### 67. `cmd/web/vergecore_test.go:70` — `short-label`

Load-bearing: [ ]

```go
67 | 		t.Errorf("added port not listed in the frequency half")
68 | 	}
69 | 
70 | 	// Reset drops the delta row.
71 | 	editFreq(t, ac, base, "reset", "12345").Body.Close()
72 | 	if _, ok := f.freqEdits[12345]; ok {
73 | 		t.Errorf("reset did not drop the edit row: %+v", f.freqEdits)
```

### 68. `cmd/web/zone_test.go:173` — `short-label`

Load-bearing: [ ]

```go
170 | 		t.Errorf("default re-supply interval of 30 days not shown; body: %s", page)
171 | 	}
172 | 
173 | 	// The operator moves the dial.
174 | 	resp := postForm(t, ac, base+"/seeds/zone/interval", url.Values{"interval_days": {"7"}})
175 | 	if resp.StatusCode != http.StatusSeeOther {
176 | 		t.Fatalf("set interval: status=%d", resp.StatusCode)
```

### 69. `cmd/worker/remoterouter_test.go:21` — `docstring-unexported`

Load-bearing: [ ]

```go
18 | 	"github.com/winniel123/verge-asm/internal/wire"
19 | )
20 | 
21 | // fakeVantageGetter is an in-memory remoteVantageStore.
22 | type fakeVantageGetter struct {
23 | 	byID map[int64]db.Vantage
24 | 	err  error
```

### 70. `cmd/worker/remoterouter_test.go:104` — `short-label`

Load-bearing: [ ]

```go
101 | 		stateDir: t.TempDir(),
102 | 	}
103 | 
104 | 	// No vantage id at all.
105 | 	if _, handled, err := rt.ProbeVantage(context.Background(), pgtype.Int8{}, wire.JobSpec{}); handled || err != nil {
106 | 		t.Errorf("no-vantage: handled=%v err=%v, want deferred to local", handled, err)
107 | 	}
```

### 71. `cmd/worker/remoterouter_test.go:108` — `short-label`

Load-bearing: [ ]

```go
105 | 	if _, handled, err := rt.ProbeVantage(context.Background(), pgtype.Int8{}, wire.JobSpec{}); handled || err != nil {
106 | 		t.Errorf("no-vantage: handled=%v err=%v, want deferred to local", handled, err)
107 | 	}
108 | 	// Resolver-only vantage.
109 | 	if _, handled, err := rt.ProbeVantage(context.Background(), pgtype.Int8{Int64: 1, Valid: true}, wire.JobSpec{}); handled || err != nil {
110 | 		t.Errorf("resolver-only: handled=%v err=%v, want deferred to local", handled, err)
111 | 	}
```

### 72. `design-system/designfs_test.go:8-10` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 5 | 	"testing"
 6 | )
 7 | 
 8 | // TestFSExposesDesignArtifacts pins the read-only surface the web app depends
 9 | // on: the inventory template, the full token set, and the fixture corpus must
10 | // all be reachable through FS by their package-relative paths.
11 | func TestFSExposesDesignArtifacts(t *testing.T) {
12 | 	want := []string{
13 | 		"templates/inventory.tmpl",
```

### 73. `internal/auth/auth_test.go:183` — `short-label`

Load-bearing: [ ]

```go
180 | 	if !VerifyTOTP(secret, code, now.Add(25*time.Second)) {
181 | 		t.Fatal("code rejected inside skew window")
182 | 	}
183 | 	// Two steps away, rejected.
184 | 	if VerifyTOTP(secret, code, now.Add(90*time.Second)) {
185 | 		t.Fatal("stale code accepted well outside the window")
186 | 	}
```

### 74. `internal/custody/coverage_test.go:40` — `short-label`

Load-bearing: [ ]

```go
37 | func TestCoversAddressScopeRefusesExtension(t *testing.T) {
38 | 	globallyReachable := netip.MustParseAddr("93.184.216.34")
39 | 	e := Estate{
40 | 		// No address scope covers the address...
41 | 		AddressScopes: nil,
42 | 		// ...but a custody extension does, via a resolution inside the extended zone.
43 | 		ExtendedZones: []string{"example.com"},
```

### 75. `internal/custody/gate_test.go:164-168` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
161 | 	}
162 | }
163 | 
164 | // TestQueryIsNotAConnect: the gate is over an active probe against an Address; it
165 | // says nothing about resolution / dns-record, which run at full aperture on every
166 | // Name regardless of custody. This test documents that boundary by asserting the
167 | // gate is the only thing MayProbe governs — a third-party address returns false
168 | // here, while the DNS facets (not gated by this package) are unaffected.
169 | func TestQueryIsNotAConnect(t *testing.T) {
170 | 	e := Estate{} // custody of nothing
171 | 	if e.MayProbe(addr("52.1.2.3"), ClassInternet) {
```

### 76. `internal/custody/veto_test.go:13-14` — `docstring-unexported`

Load-bearing: [ ]

```go
10 | // from one would pass for the wrong reason.
11 | var edge = netip.MustParseAddr("104.16.132.229")
12 | 
13 | // extended is one custody-extended zone whose in-zone name holds a direct A record on
14 | // the edge — the exact shape the veto narrows.
15 | func extended(f EdgeFanout) Estate {
16 | 	return Estate{
17 | 		ExtendedZones: []string{"example.com"},
```

### 77. `internal/drift/trend_test.go:8-9` — `docstring-unexported`

Load-bearing: [ ]

```go
 5 | 	"time"
 6 | )
 7 | 
 8 | // now anchor for the trend fold — distinct from drift_test.go's t0. A weekly bucket
 9 | // is the Reports range's own week granularity (reportsRangeLabel "last N weeks").
10 | var trendNow = time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
11 | 
12 | const week = 7 * 24 * time.Hour
```

### 78. `internal/estate/withdrawal_test.go:77` — `short-label`

Load-bearing: [ ]

```go
74 | 		t.Error("cross-class disagreement keeps the Name present")
75 | 	}
76 | 
77 | 	// Both classes agree on NameError: withdrawn.
78 | 	est2 := Membership([]Observation{
79 | 		{Name: "dead.example.com", Vantage: "in", Class: "internal", Resolution: ne},
80 | 		{Name: "dead.example.com", Vantage: "out", Class: "internet", Resolution: ne},
```

### 79. `internal/exposure/exposure_test.go:97` — `section-divider`

Load-bearing: [ ]

```go
 94 | 	}
 95 | }
 96 | 
 97 | // --- VerifyClass: re-verified per batch against the presented address -------
 98 | 
 99 | // AC #196: Vantage class is re-verified every Batch against the PRESENTED
100 | // address, not a static config field (CONTEXT.md `Vantage class`). The quantifier
```

### 80. `internal/measure/edgefanout/leaf_test.go:30-31` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
27 | 	return h.byTarget[target]
28 | }
29 | 
30 | // TestFoldOutcomeSpace pins the closed union: a presented chain carries the leaf's
31 | // fingerprint, and each of the three negatives carries its own value and NO fingerprint.
32 | func TestFoldOutcomeSpace(t *testing.T) {
33 | 	fp := co.Fingerprint(leafDER)
34 | 	cases := []struct {
```

### 81. `internal/qr/qr_test.go:62-63` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
59 | 	}
60 | }
61 | 
62 | // TestChooseVersion checks the smallest fitting level-M byte-mode version at a
63 | // few capacity boundaries.
64 | func TestChooseVersion(t *testing.T) {
65 | 	cases := []struct {
66 | 		n, version int
```

### 82. `internal/queue/edgefanoutbench_test.go:59-60` — `docstring-unexported`

Load-bearing: [ ]

```go
56 | 	benchSharedSANs = 400
57 | )
58 | 
59 | // benchSANShape is one of the two edges the benchmark contrasts: how many dNSName SANs
60 | // one certificate carries, and the verdict custody.SharedEdge owes over that SAN set.
61 | type benchSANShape struct {
62 | 	name string
63 | 	// sans is the count of dNSName SANs, each on its own registrable domain, so the
```

### 83. `internal/queue/membership_test.go:112` — `short-label`

Load-bearing: [ ]

```go
109 | 			wantLeft:    false,
110 | 		},
111 | 		{
112 | 			// stays: nothing to close.
113 | 			name:     "no open spans cannot leave",
114 | 			open:     nil,
115 | 			wantLeft: false,
```

### 84. `internal/queue/transcript_test.go:17-19` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
14 | // testKey is a fixed 32-byte XChaCha20-Poly1305 key for sealing in tests.
15 | var testKey = bytes.Repeat([]byte{0x42}, 32)
16 | 
17 | // TestHeadTail checks the truncation math: a stream within the limit is returned
18 | // whole with no drop; a stream over it keeps the head and tail halves and reports the
19 | // exact dropped middle.
20 | func TestHeadTail(t *testing.T) {
21 | 	within := []byte("hello")
22 | 	out, dropped := headTail(within, 10)
```

### 85. `internal/queue/transcript_test.go:149-150` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
146 | 	}
147 | }
148 | 
149 | // TestBuildProberParamsTruncation checks an over-cap stdout is head+tail truncated to
150 | // its store cap and carries an accurate {kept, dropped} marker.
151 | func TestBuildProberParamsTruncation(t *testing.T) {
152 | 	big := bytes.Repeat([]byte("x"), capTranscriptStdout+100)
153 | 	tr := wire.ProberTranscript{
```

### 86. `internal/queue/transcript_test.go:356-359` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
353 | 	}
354 | }
355 | 
356 | // TestBuildCTParams checks a captured CT transcript maps to the row the worker inserts:
357 | // variant is ct, the verbatim response body seals into the stdout role column and opens
358 | // back, the request URL and status ride the outcome, and stderr/sent-scope stay NULL
359 | // (streams the crt.sh producer does not carry).
360 | func TestBuildCTParams(t *testing.T) {
361 | 	capturedAt := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
362 | 	body := []byte(`[{"name_value":"a.example.com"},{"name_value":"b.example.com"}]`)
```

### 87. `internal/remoteexec/probe_test.go:20` — `docstring-unexported`

Load-bearing: [ ]

```go
17 | // every command run, so the arch check, the push, the exec and the egress read are all
18 | // exercised with no live SSH server.
19 | type fakeConn struct {
20 | 	// outputs maps an Output/rm command to its stdout.
21 | 	outputs map[string]string
22 | 	// execStdout is the NDJSON the exec of the pushed path writes back.
23 | 	execStdout string
```

### 88. `internal/remoteexec/probe_test.go:22` — `docstring-unexported`

Load-bearing: [ ]

```go
19 | type fakeConn struct {
20 | 	// outputs maps an Output/rm command to its stdout.
21 | 	outputs map[string]string
22 | 	// execStdout is the NDJSON the exec of the pushed path writes back.
23 | 	execStdout string
24 | 	// execStderr is the stderr the exec writes back (empty unless a crash is simulated).
25 | 	execStderr string
```

### 89. `internal/remoteexec/probe_test.go:84` — `docstring-unexported`

Load-bearing: [ ]

```go
81 | 
82 | func (c *fakeConn) Close() error { return nil }
83 | 
84 | // staticBinaries serves one arch's binary and refuses every other.
85 | type staticBinaries struct {
86 | 	goos, goarch string
87 | 	body         string
```

### 90. `internal/retention/observation_test.go:193` — `section-divider`

Load-bearing: [ ]

```go
190 | 	}
191 | }
192 | 
193 | // --- Retirer ---------------------------------------------------------------
194 | 
195 | // fakeObsStore is the whole surface the ObservationRetirer can reach: the dial and
196 | // the observation-only delete. It records the params so the sweep's dial-to-seconds
```

### 91. `internal/retention/retention_test.go:56-58` — `docstring-unexported`

Load-bearing: [ ]

```go
53 | 	}
54 | }
55 | 
56 | // fakeStore is the whole surface the Retirer can reach. It has no Observation,
57 | // Span or Batch method — the compiler will not let retention code touch measured
58 | // data through it, which is the separation AC proved structurally.
59 | type fakeStore struct {
60 | 	multiple      int64
61 | 	cadence       int64
```

### 92. `internal/scan/cttail_test.go:205` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
202 | 	}
203 | }
204 | 
205 | // TestCTTailScopeRoundTrip confirms a job's wire scope decodes back to the log it names.
206 | func TestCTTailScopeRoundTrip(t *testing.T) {
207 | 	j := CTTailJob{ScanID: 7, Log: CTLog{LogID: "abc=", URL: "https://ct.example/log/", Description: "Example log"}}
208 | 	spec, err := j.JobSpec("scan:7:log:abc=")
```

### 93. `internal/scan/ctverify_test.go:114` — `section-divider`

Load-bearing: [ ]

```go
111 | 	}
112 | }
113 | 
114 | // --- precert TBS reconstruction ----------------------------------------------
115 | 
116 | // TestPrecertTBSRemovesSCTList builds two otherwise-identical certificates — one WITH an
117 | // embedded SCT-list extension, one WITHOUT — and asserts PrecertTBS on the first yields the
```

### 94. `internal/scan/ctverify_test.go:191` — `section-divider`

Load-bearing: [ ]

```go
188 | 	return -1
189 | }
190 | 
191 | // --- SCT parsing -------------------------------------------------------------
192 | 
193 | // buildSCT serializes a minimal SignedCertificateTimestamp for a round-trip test.
194 | func buildSCT(logID [32]byte, timestamp uint64, extensions []byte) []byte {
```

### 95. `internal/scan/ctverify_test.go:249` — `short-label`

Load-bearing: [ ]

```go
246 | 	if !ok || got != idx {
247 | 		t.Fatalf("SCTLeafIndex = %d, %v; want %d, true", got, ok, idx)
248 | 	}
249 | 	// No extensions => no leaf index.
250 | 	if _, ok := SCTLeafIndex(nil); ok {
251 | 		t.Fatal("empty extensions reported a leaf index")
252 | 	}
```

### 96. `internal/scan/ctverify_test.go:262-263` — `docstring-unexported`

Load-bearing: [ ]

```go
259 | 	logID[0] = 0x11
260 | 	sct := buildSCT(logID, 42, nil)
261 | 
262 | 	// SignedCertificateTimestampList: opaque16 list of opaque16 SCTs, wrapped in a DER OCTET
263 | 	// STRING — the shape EmbeddedSCTs unwraps.
264 | 	var list cryptobyte.Builder
265 | 	list.AddUint16LengthPrefixed(func(outer *cryptobyte.Builder) {
266 | 		outer.AddUint16LengthPrefixed(func(one *cryptobyte.Builder) { one.AddBytes(sct) })
```

### 97. `internal/scan/zone_test.go:206` — `short-label`

Load-bearing: [ ]

```go
203 | 	if a := ZoneAgingAt(supply, almost, interval); a.Stale || a.Days != 1 {
204 | 		t.Errorf("half a day from the gap should read 1d and current; got %+v", a)
205 | 	}
206 | 	// No supply: nothing to stale.
207 | 	if a := ZoneAgingAt(time.Time{}, supply, interval); a.Supplied || a.Stale {
208 | 		t.Errorf("an unsupplied scope has nothing to age; got %+v", a)
209 | 	}
```

### 98. `internal/signal/rules_test.go:11` — `docstring-unexported`

Load-bearing: [ ]

```go
 8 | 	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
 9 | )
10 | 
11 | // ruleByName finds a shipped rule by its name for the per-rule tests.
12 | func ruleByName(t *testing.T, name string) Rule {
13 | 	t.Helper()
14 | 	for _, r := range All() {
```

### 99. `internal/signal/severity_test.go:5` — `docstring-unexported`

Load-bearing: [ ]

```go
2 | 
3 | import "testing"
4 | 
5 | // inRamp reports whether a severity is one of the five ramp levels.
6 | func inRamp(s Severity) bool {
7 | 	for _, sv := range SevOrder {
8 | 		if sv == s {
```

### 100. `internal/transcript/crypto_test.go:9` — `docstring-unexported`

Load-bearing: [ ]

```go
 6 | 	"testing"
 7 | )
 8 | 
 9 | // newTestKey returns a valid 32-byte key for the seal/open tests.
10 | func newTestKey(t *testing.T) []byte {
11 | 	t.Helper()
12 | 	key := make([]byte, keyLen)
```

