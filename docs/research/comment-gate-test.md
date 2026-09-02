# Comment policy validation gate — test Go, round 1

SPEC `docs/spec/comment-policy.md` §3.9. Regenerate this sheet with:

```sh
go run ./cmd/commentlint sample --population test --round 1
```

- In-scope Go files read: 221
- Blocks the §3.2 screen admits for deletion: 580
- Blocks drawn into the gate sample: 100
- Blocks drawn into the coverage supplement: 0

Accept a class at 2 or fewer load-bearing blocks. A class that fails three rounds leaves the v1 delete set and stays in the flag set.

## Class coverage

| Class | Admitted | Gate sample | Supplement |
| --- | --- | --- | --- |
| `commented-out-code` | 0 | 0 | 0 |
| `docstring-exported-conventional` | 145 | 28 | 0 |
| `docstring-unexported` | 277 | 46 | 0 |
| `section-divider` | 32 | 9 | 0 |
| `short-label` | 126 | 17 | 0 |

## Verdicts

A reviewer fills the last two columns. A class that admits no block on this population reads `n/a`.

| Class | Read | Load-bearing | Verdict |
| --- | --- | --- | --- |
| `commented-out-code` | 0 | n/a | n/a |
| `docstring-exported-conventional` | 28 | | |
| `docstring-unexported` | 46 | | |
| `section-divider` | 9 | | |
| `short-label` | 17 | | |

## Gate sample

### 1. `cmd/web/adr0130_contract_test.go:29` — `section-divider`

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

### 2. `cmd/web/adr0130_contract_test.go:34-35` — `docstring-unexported`

Load-bearing: [ ]

```go
31 | // contractPkg is cmd/web parsed from disk, minus the tests.
32 | type contractPkg struct {
33 | 	fset *token.FileSet
34 | 	// methods maps a *server method name to its declaration. A handler, a helper and a
35 | 	// renderer are all here; the guards tell them apart by name and by who calls whom.
36 | 	methods map[string]*ast.FuncDecl
37 | 	// funcs maps a package-level FUNCTION name to its declaration. The walk follows these
38 | 	// too, because an answer written through a free function is the same answer: both
```

### 3. `cmd/web/adr0130_contract_test.go:191-193` — `docstring-unexported`

Load-bearing: [ ]

```go
188 | 	})
189 | }
190 | 
191 | // calleesOf lists what fn calls on its live path, in source order: each *server method
192 | // as "s.Name", each package-level function by its bare name, and each answer written
193 | // straight to the ResponseWriter as "http.Error" or "http.Redirect".
194 | func (c *contractPkg) calleesOf(fn *ast.FuncDecl) []string {
195 | 	var out []string
196 | 	inspectLive(fn, func(n ast.Node) bool {
```

### 4. `cmd/web/adr0130_contract_test.go:235-238` — `docstring-unexported`

Load-bearing: [ ]

```go
232 | 	return fn, found
233 | }
234 | 
235 | // reach walks the call graph out of a handler and calls visit on every body the handler
236 | // can reach, itself included. Names are as calleesOf spells them ("s.Method", or a bare
237 | // function name). stop names what the walk does not descend into, so a guard can treat a
238 | // sanctioned answer as terminal.
239 | func (c *contractPkg) reach(start string, stop map[string]bool, visit func(name string, fn *ast.FuncDecl)) {
240 | 	seen := map[string]bool{}
241 | 	var walk func(string)
```

### 5. `cmd/web/adr0130_contract_test.go:302` — `section-divider`

Load-bearing: [ ]

```go
299 | 	return out
300 | }
301 | 
302 | // --- class A: a refusal is a redirect, never a body ------------------------
303 | 
304 | // bodyAnswers are the calls that put a BODY on the response at the URL the form was
305 | // posted to. The eight renders are the page templates; http.Error is the same failure in
```

### 6. `cmd/web/adr0130_contract_test.go:436-438` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
433 | 	}
434 | }
435 | 
436 | // TestContractExemptionsAreLive keeps the three exemption lists honest. An entry naming a
437 | // route the tree no longer serves, or an answer no handler makes any more, is deleted
438 | // rather than left to imply a live exception.
439 | func TestContractExemptionsAreLive(t *testing.T) {
440 | 	c := parseWebPackage(t)
441 | 	for _, m := range []map[string]string{classAExemptRoutes, classEExempt} {
```

### 7. `cmd/web/adr0130_contract_test.go:467` — `section-divider`

Load-bearing: [ ]

```go
464 | 	}
465 | }
466 | 
467 | // --- class E: a mutating act lands on the URL it was submitted from --------
468 | 
469 | // backHelpers are the sanctioned answers to a mutating act: each resolves the submitting
470 | // URL off the posted `return` field (backurl.go resolveBack) and 303s to it, falling back
```

### 8. `cmd/web/adr0130_contract_test.go:594-602` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
591 | 
592 | // --- the carrier: one refusal, shown once ----------------------------------
593 | 
594 | // TestTheSessionFormFlashIsSingleConsume pins the property every §1 landing depends on.
595 | // A refusal is stashed once and read once. The read DELETES it, so the reload the
596 | // operator performs — or the meta-refresh a Scans view performs for them while a scan is
597 | // in flight — finds nothing and re-shows no callout they have already answered.
598 | //
599 | // The surface tests next door assert this end to end on /settings, /scope and /signals.
600 | // This one asserts it at the store, where the property actually lives, so a change to
601 | // the take path fails here with a one-line reason rather than as a puzzling body
602 | // mismatch three files away.
603 | func TestTheSessionFormFlashIsSingleConsume(t *testing.T) {
604 | 	store := newFormFlashStore()
605 | 	now := time.Now()
```

### 9. `cmd/web/annotations_test.go:274-275` — `docstring-unexported`

Load-bearing: [ ]

```go
271 | 	return prgLanding(t, c, base, resp)
272 | }
273 | 
274 | // annotateFrom declares an Annotation the way the drawer's form does: carrying the
275 | // `return` field that names the exact URL the operator submitted from (backurl.go).
276 | func annotateFrom(t *testing.T, c *http.Client, base, from, subject, signal, reason string) *http.Response {
277 | 	t.Helper()
278 | 	return postForm(t, c, base+"/annotations", url.Values{
```

### 10. `cmd/web/api_auth_test.go:39-40` — `docstring-unexported`

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

### 11. `cmd/web/auth_test.go:88` — `docstring-unexported`

Load-bearing: [ ]

```go
85 | 	return acct
86 | }
87 | 
88 | // login runs a password-only login and returns the authenticated client.
89 | func login(t *testing.T, base, username, password string) *http.Client {
90 | 	t.Helper()
91 | 	c := newClient(t)
```

### 12. `cmd/web/auth_test.go:506` — `section-divider`

Load-bearing: [ ]

```go
503 | 	}
504 | }
505 | 
506 | // --- no forward-auth -------------------------------------------------------
507 | 
508 | func TestNoForwardAuthHeaderTrusted(t *testing.T) {
509 | 	f := newFakeStore()
```

### 13. `cmd/web/backup_test.go:46` — `short-label`

Load-bearing: [ ]

```go
43 | 		case 0:
44 | 			t.Errorf("table %q is neither in the backup allowlist nor the documented exclusions — classify it", tbl)
45 | 		case 1:
46 | 			// good
47 | 		default:
48 | 			t.Errorf("table %q is classified more than once (allowlist and/or exclusions)", tbl)
49 | 		}
```

### 14. `cmd/web/backup_test.go:269` — `short-label`

Load-bearing: [ ]

```go
266 | 	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
267 | 	base := start(t, f, "")
268 | 
269 | 	// Anonymous -> redirect to /login.
270 | 	anon := newClient(t)
271 | 	resp := postForm(t, anon, base+"/settings/backup", url.Values{})
272 | 	resp.Body.Close()
```

### 15. `cmd/web/backup_test.go:277` — `short-label`

Load-bearing: [ ]

```go
274 | 		t.Fatalf("anonymous backup: status=%d loc=%q, want 303 -> /login", resp.StatusCode, resp.Header.Get("Location"))
275 | 	}
276 | 
277 | 	// Viewer -> 403.
278 | 	vc := login(t, base, "viewer", "hunter2hunter2")
279 | 	resp = postForm(t, vc, base+"/settings/backup", url.Values{})
280 | 	resp.Body.Close()
```

### 16. `cmd/web/backurl_test.go:263` — `docstring-unexported`

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

### 17. `cmd/web/backurl_test.go:472-473` — `docstring-unexported`

Load-bearing: [ ]

```go
469 | 	}
470 | }
471 | 
472 | // firstViewKey pulls one row's Drawer key off a rendered Signals list, so a test can
473 | // open a Drawer without hardcoding a minted SIG id.
474 | func firstViewKey(t *testing.T, page string) string {
475 | 	t.Helper()
476 | 	// The row href is html-escaped, so the separator before `view=` reads `&amp;`.
```

### 18. `cmd/web/clientip_test.go:9-11` — `docstring-unexported`

Load-bearing: [ ]

```go
 6 | 	"time"
 7 | )
 8 | 
 9 | // reqWith builds a bare request carrying the given RemoteAddr and, when non-empty,
10 | // a single X-Forwarded-For header — the two inputs clientIP derives the limiter key
11 | // from.
12 | func reqWith(remoteAddr, xff string) *http.Request {
13 | 	r := &http.Request{RemoteAddr: remoteAddr, Header: http.Header{}}
14 | 	if xff != "" {
```

### 19. `cmd/web/credflow_sessions_test.go:47-48` — `docstring-unexported`

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

### 20. `cmd/web/devfixtures_test.go:116-121` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
113 | 	return f
114 | }
115 | 
116 | // TestProfileFixtureMatchesPackage is the byte-exactness gate for the screen-3 seed: every
117 | // value devfixtures.go pins is folded back through the frozen fixtures.json → profile slice
118 | // (and the pinned clock), and the clock-relative / UA-derived renders are reproduced with
119 | // the real formatters (relTime, sessionDeviceFromUA) — so any drift between the seeder and
120 | // the frozen package fails here rather than in a screenshot, exactly as
121 | // TestDevFixturesMatchPackage guards the ErrorPage slice.
122 | func TestProfileFixtureMatchesPackage(t *testing.T) {
123 | 	f := loadFixtureProfilePackage(t)
124 | 	p := f.Profile
```

### 21. `cmd/web/devfixtures_test.go:443-445` — `docstring-unexported`

Load-bearing: [ ]

```go
440 | 	}
441 | }
442 | 
443 | // fixtureExposurePackage mirrors the fixtures.json exposure slice the screen-7 dev fixture pins
444 | // in devfixtures.go (the summary band, the +2 exposed delta, the withheld variant and the six
445 | // board rows).
446 | type fixtureExposurePackage struct {
447 | 	Exposure struct {
448 | 		Exposed         int    `json:"exposed"`
```

### 22. `cmd/web/devfixtures_test.go:464-468` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
461 | 	} `json:"exposure"`
462 | }
463 | 
464 | // TestExposureFixtureMatchesPackage is the byte-exactness gate for the screen-7 conversion: every
465 | // value the dev fixture pins (devfixtures.go, served by exposurePage under devMode) equals the
466 | // frozen fixtures.json exposure slice, in authored order — so a drift between the served candidate
467 | // and the golden (which composes the same fixture statically) fails here rather than in a
468 | // screenshot diff, exactly as TestCoverageFixtureMatchesPackage guards the Coverage slice.
469 | func TestExposureFixtureMatchesPackage(t *testing.T) {
470 | 	raw, err := os.ReadFile("../../design-system/fixtures/fixtures.json")
471 | 	if err != nil {
```

### 23. `cmd/web/devfixtures_test.go:924` — `short-label`

Load-bearing: [ ]

```go
921 | 			r, devScopeRefusalPost, devScopeRefusalInput, devScopeRefusalReason, devScopeRefusalReachable, devScopeRefusalFormError)
922 | 	}
923 | 
924 | 	// Custody.
925 | 	if len(sc.CustodyScopes) != len(devScopeCustody) {
926 | 		t.Fatalf("custody length drift: fixtures.json = %d, pinned = %d", len(sc.CustodyScopes), len(devScopeCustody))
927 | 	}
```

### 24. `cmd/web/devfixtures_test.go:979` — `short-label`

Load-bearing: [ ]

```go
976 | 		}
977 | 	}
978 | 
979 | 	// Proposals.
980 | 	if len(sc.Proposals) != len(devScopeProposals) {
981 | 		t.Fatalf("proposals length drift: fixtures.json = %d, pinned = %d", len(sc.Proposals), len(devScopeProposals))
982 | 	}
```

### 25. `cmd/web/devfixtures_test.go:1137` — `short-label`

Load-bearing: [ ]

```go
1134 | 		t.Errorf("history_rule missing pinned detecting vantage %q: %q", devSignalsDetectedBy, sig.HistoryRule)
1135 | 	}
1136 | 
1137 | 	// Open rows.
1138 | 	if len(sig.Rows) != len(devSignalsOpen) {
1139 | 		t.Fatalf("open rows length drift: fixtures.json = %d, pinned = %d", len(sig.Rows), len(devSignalsOpen))
1140 | 	}
```

### 26. `cmd/web/devfixtures_test.go:1145` — `short-label`

Load-bearing: [ ]

```go
1142 | 		assertSignalRow(t, "open", i, row, devSignalsOpen[i])
1143 | 	}
1144 | 
1145 | 	// Withdrawn rows.
1146 | 	if len(sig.Withdrawn) != len(devSignalsWithdrawn) {
1147 | 		t.Fatalf("withdrawn rows length drift: fixtures.json = %d, pinned = %d", len(sig.Withdrawn), len(devSignalsWithdrawn))
1148 | 	}
```

### 27. `cmd/web/devfixtures_test.go:1373` — `docstring-unexported`

Load-bearing: [ ]

```go
1370 | 	}
1371 | }
1372 | 
1373 | // fixtureAssetPackage is the fixtures.json → asset slice, snake_case as stored.
1374 | type fixtureAssetPackage struct {
1375 | 	Asset struct {
1376 | 		Key          string `json:"key"`
```

### 28. `cmd/web/handlers_test.go:269` — `docstring-unexported`

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

### 29. `cmd/web/handlers_test.go:275-276` — `docstring-unexported`

Load-bearing: [ ]

```go
272 | 	createdBy int64
273 | }
274 | 
275 | // fakeChannel mirrors a channel row, secret included, so tests can assert the
276 | // secret is stored but never surfaced through the render path.
277 | type fakeChannel struct {
278 | 	id                     int64
279 | 	url                    string
```

### 30. `cmd/web/handlers_test.go:403` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
400 | 	return n, nil
401 | }
402 | 
403 | // SetDispatchStatus records a dispatch's operator-ended disposition.
404 | func (f *fakeStore) SetDispatchStatus(_ context.Context, arg db.SetDispatchStatusParams) error {
405 | 	if f.dispatchStatus == nil {
406 | 		f.dispatchStatus = map[int64]string{}
```

### 31. `cmd/web/handlers_test.go:412` — `docstring-exported-conventional`

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

### 32. `cmd/web/handlers_test.go:2347` — `docstring-unexported`

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

### 33. `cmd/web/handlers_test.go:3148-3149` — `docstring-unexported`

Load-bearing: [ ]

```go
3145 | 	return ""
3146 | }
3147 | 
3148 | // usernameForID resolves an account id to its username for the created-by join the
3149 | // SSO list query performs; an unknown id renders empty (a test rarely asserts it).
3150 | func (f *fakeStore) usernameForID(id int64) string {
3151 | 	return f.accounts[id].Username
3152 | }
```

### 34. `cmd/web/integrations_channel_test.go:234` — `short-label`

Load-bearing: [ ]

```go
231 | 	base := start(t, f, "")
232 | 	ac := login(t, base, "admin", "hunter2hunter2")
233 | 
234 | 	// Empty channel unbinds.
235 | 	resp := postForm(t, ac, base+"/settings/integrations/channel", url.Values{"id": {"slack"}, "channel": {""}})
236 | 	resp.Body.Close()
237 | 	if st := f.integrationStates["slack"]; st.ChannelID.Valid {
```

### 35. `cmd/web/inventory_fixture_test.go:54-55` — `docstring-unexported`

Load-bearing: [ ]

```go
51 | 	Data string `json:"data"`
52 | }
53 | 
54 | // fixtureSpanRows builds the exact ListAllOpenSpansRow inputs the loader inserts,
55 | // from the single-source-of-truth inventoryFixtureSpans slice.
56 | func fixtureSpanRows(t *testing.T) []db.ListAllOpenSpansRow {
57 | 	t.Helper()
58 | 	rows := make([]db.ListAllOpenSpansRow, 0, len(inventoryFixtureSpans))
```

### 36. `cmd/web/onboarding_test.go:90` — `short-label`

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

### 37. `cmd/web/probers_test.go:24-25` — `docstring-unexported`

Load-bearing: [ ]

```go
21 | 	})
22 | }
23 | 
24 | // vantagesTab is the section's own URL, and the fallback destination of a provision
25 | // submitted with no `return` field.
26 | const vantagesTab = "/settings?tab=vantages"
27 | 
28 | // vantagesBody reads the Settings → Vantages tab, where prober provisioning + the
```

### 38. `cmd/web/profile_test.go:114-115` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
111 | 
112 | // --- session registry fakes (#405, ADR-0117) -------------------------------
113 | 
114 | // CreateSession opens a session row: a monotonic id, now-stamped created_at and
115 | // last_seen_at, and the caller's token hash / user-agent / ip / expiry.
116 | func (f *fakeStore) CreateSession(_ context.Context, arg db.CreateSessionParams) (db.Session, error) {
117 | 	if f.sessionNextID == 0 {
118 | 		f.sessionNextID = 1
```

### 39. `cmd/web/profile_test.go:155-157` — `docstring-exported-conventional`

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

### 40. `cmd/web/profile_test.go:204-205` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
201 | 	return nil
202 | }
203 | 
204 | // RevokeAllSessionsForAccount revokes every live session for the account with no
205 | // exception — the reset path (no current session to keep) and admin offboarding.
206 | func (f *fakeStore) RevokeAllSessionsForAccount(_ context.Context, arg db.RevokeAllSessionsForAccountParams) error {
207 | 	for i := range f.sessions {
208 | 		if f.sessions[i].AccountID == arg.AccountID && !f.sessions[i].RevokedAt.Valid {
```

### 41. `cmd/web/profile_test.go:255` — `section-divider`

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

### 42. `cmd/web/proposals_test.go:20-22` — `docstring-unexported`

Load-bearing: [ ]

```go
17 | 	"github.com/winniel123/verge-asm/internal/proposer"
18 | )
19 | 
20 | // fakeProposer stands in for the real registry: it returns canned candidates and
21 | // records the enabled set it was asked to run, so a test can assert that the
22 | // source-enablement state gates which paths run without any network.
23 | type fakeProposer struct {
24 | 	candidates  []proposer.Candidate
25 | 	err         error
```

### 43. `cmd/web/proposals_test.go:63-64` — `docstring-unexported`

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

### 44. `cmd/web/proposals_test.go:409-410` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
406 | 	}
407 | }
408 | 
409 | // TestLookupGenuineMissStillReadsAsAMiss keeps the honest no-match message when
410 | // the registries answered cleanly and simply held nothing.
411 | func TestLookupGenuineMissStillReadsAsAMiss(t *testing.T) {
412 | 	f := newFakeStore()
413 | 	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
```

### 45. `cmd/web/ratelimit_test.go:47-48` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
44 | 	}
45 | }
46 | 
47 | // TestLimiterResetOnSuccess covers the success path: a legitimate operator who
48 | // mistyped a few times, then authenticated, starts clean.
49 | func TestLimiterResetOnSuccess(t *testing.T) {
50 | 	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
51 | 	l := newTestLimiter(c)
```

### 46. `cmd/web/ratelimit_test.go:85-86` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
82 | 	}
83 | }
84 | 
85 | // TestLimiterUnlocksAfterLockout covers recovery: once the lockout span elapses the
86 | // key is usable again, so a locked-out operator is never locked out forever.
87 | func TestLimiterUnlocksAfterLockout(t *testing.T) {
88 | 	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
89 | 	l := newTestLimiter(c)
```

### 47. `cmd/web/reports_test.go:507` — `short-label`

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

### 48. `cmd/web/restore_test.go:17-19` — `docstring-unexported`

Load-bearing: [ ]

```go
14 | 	"github.com/winniel123/verge-asm/internal/db"
15 | )
16 | 
17 | // buildTestArchive assembles a B3-format archive with the real backup writers (round-trip
18 | // against cmd/web/backup.go) at a chosen schema version, carrying the given span rows so a
19 | // pre-flight has real subjects to count.
20 | func buildTestArchive(t *testing.T, schemaVersion int64, spanRows []string) []byte {
21 | 	t.Helper()
22 | 	var buf bytes.Buffer
```

### 49. `cmd/web/restore_test.go:65-66` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
62 | 	}
63 | }
64 | 
65 | // TestPreflightArchiveRejectsGarbage proves an unparseable upload is refused with nothing
66 | // touched (the handler maps this to `.RestoreError`).
67 | func TestPreflightArchiveRejectsGarbage(t *testing.T) {
68 | 	if _, err := preflightArchive(strings.NewReader("this is not a verge archive\n")); err == nil {
69 | 		t.Fatal("preflightArchive accepted garbage; want an error")
```

### 50. `cmd/web/restore_test.go:158-160` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
155 | 	return settingsForms{}
156 | }
157 | 
158 | // TestRestorePreflightPassesGateWithoutPool proves that with no scan in flight an admin
159 | // passes the gate and reaches the pool guard (503 off a pool), never a 403 — proof the
160 | // refusal above was the in-flight check, not the admin gate.
161 | func TestRestorePreflightPassesGateWithoutPool(t *testing.T) {
162 | 	f := newFakeStore()
163 | 	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
```

### 51. `cmd/web/restore_test.go:199-202` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
196 | 	}
197 | }
198 | 
199 | // TestRestoreApplyRequiresConfirmWord proves the apply never applies without the typed word
200 | // `restore` (validated server-side, never trusting the JS gate): a wrong word refuses with
201 | // nothing touched, and a correct word with no staged archive refuses as expired. Neither
202 | // rotates the session key.
203 | func TestRestoreApplyRequiresConfirmWord(t *testing.T) {
204 | 	f := newFakeStore()
205 | 	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
```

### 52. `cmd/web/restore_test.go:286-287` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
283 | 	}
284 | }
285 | 
286 | // TestRestoreErrorMessages proves every refusal code maps to a fixed operator line and an
287 | // unknown code reflects nothing (no arbitrary query text on the page).
288 | func TestRestoreErrorMessages(t *testing.T) {
289 | 	for _, code := range []string{"inflight", "unreadable", "schema", "confirm", "expired", "apply"} {
290 | 		if restoreErrorMessage(code) == "" {
```

### 53. `cmd/web/scans_test.go:383` — `docstring-unexported`

Load-bearing: [ ]

```go
380 | 	}
381 | }
382 | 
383 | // progressRow builds a ListDispatchProgressRow with the given per-state counts.
384 | func progressRow(id int64, kind string, tick time.Time, total, ready, running, done, dead, retried int64) db.ListDispatchProgressRow {
385 | 	return db.ListDispatchProgressRow{
386 | 		DispatchID: id,
```

### 54. `cmd/web/scantrigger_test.go:39-41` — `docstring-unexported`

Load-bearing: [ ]

```go
36 | 	return t.jobs, nil
37 | }
38 | 
39 | // startWithTrigger is start with a scan-trigger seam wired in, the way main.go
40 | // wires the real Dispatcher over the pool. Redirects are not followed so the
41 | // test reads the 303 and its destination.
42 | func startWithTrigger(t *testing.T, f *fakeStore, trig scanTrigger) string {
43 | 	t.Helper()
44 | 	srv := newServer(f, testKey, "", fixedClock())
```

### 55. `cmd/web/scope_bulk_test.go:15-19` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
12 | 	"testing"
13 | )
14 | 
15 | // TestScopeAndOnboardingShareTokenizer proves the paste-split declare (DF-F1) and the
16 | // onboarding seedsadd field tokenize an identical raw string identically — they call
17 | // the ONE parseSeedTokens (cmd/web/onboarding.go), never a fork. The scope side is
18 | // checked end-to-end (every valid token becomes a seed); the onboarding side through
19 | // readOnboardView; and both are pinned to parseSeedTokens' own output.
20 | func TestScopeAndOnboardingShareTokenizer(t *testing.T) {
21 | 	// Commas, spaces, tabs and newlines all split; empty tokens (the double comma) drop.
22 | 	const raw = "a.com, b.com\n c.com\td.com,,e.com"
```

### 56. `cmd/web/scope_bulk_test.go:25` — `short-label`

Load-bearing: [ ]

```go
22 | 	const raw = "a.com, b.com\n c.com\td.com,,e.com"
23 | 	want := []string{"a.com", "b.com", "c.com", "d.com", "e.com"}
24 | 
25 | 	// The shared tokenizer itself.
26 | 	if got := parseSeedTokens(raw); !reflect.DeepEqual(got, want) {
27 | 		t.Fatalf("parseSeedTokens(%q) = %v, want %v", raw, got, want)
28 | 	}
```

### 57. `cmd/web/scope_bulk_test.go:60-64` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
57 | 	}
58 | }
59 | 
60 | // TestDeclareBulkPasteMixedSuccessRefusalDuplicate covers the DF-F1 paste with a mix of
61 | // success, refusal, and a within-paste duplicate: two good names declare, the repeat of
62 | // the first refuses `already declared` (the first wins), and an over-cap block refuses
63 | // with its reachable set named. The response carries BOTH the success flash AND the
64 | // per-token callouts on one render (status 200, since successes committed).
65 | func TestDeclareBulkPasteMixedSuccessRefusalDuplicate(t *testing.T) {
66 | 	f := newFakeStore()
67 | 	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
```

### 58. `cmd/web/scope_bulk_test.go:285` — `section-divider`

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

### 59. `cmd/web/scope_bulk_test.go:327-328` — `docstring-unexported`

Load-bearing: [ ]

```go
324 | 	return strconv.FormatInt(id, 10)
325 | }
326 | 
327 | // followString GETs a URL and returns the body — used to read the destination page a
328 | // PRG toast flash lands on.
329 | func followString(t *testing.T, c *http.Client, url string) string {
330 | 	t.Helper()
331 | 	resp, err := c.Get(url)
```

### 60. `cmd/web/scope_withdrawal_preview_test.go:25` — `docstring-unexported`

Load-bearing: [ ]

```go
22 | 	return db.ListSeedWithdrawalCandidatesRow{ID: id, SubjectKind: kind, SubjectKey: key}
23 | }
24 | 
25 | // previewChip clicks a chip's remove control and returns the landing GET's body.
26 | func previewChip(t *testing.T, c *http.Client, base string, id int64) string {
27 | 	t.Helper()
28 | 	resp := postForm(t, c, base+"/seeds/preview", url.Values{"id": {intStr(id)}})
```

### 61. `cmd/web/session_test.go:149` — `short-label`

Load-bearing: [ ]

```go
146 | 		t.Fatal("both devices should be authed before any revoke")
147 | 	}
148 | 
149 | 	// Device 1 ends its own session.
150 | 	postForm(t, c1, base+"/profile/session/revoke", url.Values{}).Body.Close()
151 | 
152 | 	if authed(t, c1, base) {
```

### 62. `cmd/web/settings_test.go:220` — `short-label`

Load-bearing: [ ]

```go
217 | 		t.Fatalf("blank secret should keep existing; got %q", ch.secret.String)
218 | 	}
219 | 
220 | 	// A typed value replaces it.
221 | 	postForm(t, ac, base+"/settings/channels/update", url.Values{
222 | 		"id": {idStr}, "url": {"https://b.example.com"}, "clock": {"on"}, "secret": {"second"},
223 | 	}).Body.Close()
```

### 63. `cmd/web/settings_test.go:236` — `short-label`

Load-bearing: [ ]

```go
233 | 		t.Fatalf("clear box should null the secret; got valid=%v", f.channels[0].secret.Valid)
234 | 	}
235 | 
236 | 	// Delete removes the row.
237 | 	postForm(t, ac, base+"/settings/channels/delete", url.Values{"id": {idStr}}).Body.Close()
238 | 	if len(f.channels) != 0 {
239 | 		t.Fatalf("channel not deleted; %d remain", len(f.channels))
```

### 64. `cmd/web/settings_test.go:250` — `short-label`

Load-bearing: [ ]

```go
247 | 	base := start(t, f, "")
248 | 	ac := login(t, base, "admin", "hunter2hunter2")
249 | 
250 | 	// Promote the viewer to admin.
251 | 	postForm(t, ac, base+"/settings/accounts/role", url.Values{
252 | 		"id": {itoa(viewer.ID)}, "role": {roleAdmin},
253 | 	}).Body.Close()
```

### 65. `cmd/web/sources_test.go:331` — `section-divider`

Load-bearing: [ ]

```go
328 | 	}
329 | }
330 | 
331 | // --- helpers ----------------------------------------------------------------
332 | 
333 | // sourcesTab is the Sources section's own tab, and the fallback destination of a toggle
334 | // submitted with no `return` field (backToSection).
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

### 67. `cmd/worker/remoterouter_test.go:21` — `docstring-unexported`

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

### 68. `design-system/console_tokens_test.go:27-29` — `docstring-unexported`

Load-bearing: [ ]

```go
24 | 	"--console-pill-warn-fg",
25 | }
26 | 
27 | // Anchored per declaration rather than per file. A file-wide ban would fail the
28 | // day settings.tmpl grows a tooltip or a bulk-actions bar, both of which are
29 | // legitimate --surface-inverted callers.
30 | var consoleDecls = []struct {
31 | 	file   string
32 | 	anchor string
```

### 69. `design-system/designfs_test.go:8-10` — `docstring-exported-conventional`

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

### 70. `internal/custody/corpus/harness_test.go:210-217` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
207 | 	}
208 | }
209 | 
210 | // TestFixtureStraddlesTheThreshold pins the two boundary fixtures against the
211 | // SHIPPED constant. It is the first thing to go red on a threshold move, and its
212 | // message is what tells the session what it owes.
213 | //
214 | // It also guards the fixture itself. The SAN sets reduce through the Public
215 | // Suffix List, so a PSL revision that ever listed `invalid` would silently
216 | // collapse both counts to zero and quietly turn both rows into `not-shared`. That
217 | // failure arrives here, named, rather than as an unexplained digest move.
218 | func TestFixtureStraddlesTheThreshold(t *testing.T) {
219 | 	if atThreshold != custody.SharedEdgeThreshold {
220 | 		t.Errorf("the corpus boundary is authored at %d and custody.SharedEdgeThreshold is %d.\n"+
```

### 71. `internal/custody/fanout_test.go:102-106` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
 99 | 	}
100 | }
101 | 
102 | // TestDottedNumericSANsCannotInflateTheCount: a SAN set is third-party wire
103 | // content and Go's x509 parser does not police a dNSName, so a bundle of
104 | // dotted-numeric names must raise the fan-out by zero. Without the numeric-top
105 | // drop the PSL's wildcard rule hands each one back a distinct nonsense eTLD+1
106 | // and the count that gates the veto inflates on demand.
107 | func TestDottedNumericSANsCannotInflateTheCount(t *testing.T) {
108 | 	stuffed := make([]string, 0, 200)
109 | 	for i := 0; i < 200; i++ {
```

### 72. `internal/custody/gate_test.go:24-26` — `docstring-unexported`

Load-bearing: [ ]

```go
21 | 	rate    string
22 | }
23 | 
24 | // prober walks a probe plan and gates every connect. connect is the sole egress
25 | // point; recording the attempt there is what makes "zero attempts" a fact about
26 | // the gate rather than about the network.
27 | type prober struct {
28 | 	estate   Estate
29 | 	attempts []probeAttempt
```

### 73. `internal/delivery/delivery_test.go:346` — `docstring-unexported`

Load-bearing: [ ]

```go
343 | 	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
344 | }
345 | 
346 | // fakeResolver places a host in whatever range the test needs without real DNS.
347 | type fakeResolver map[string][]netip.Addr
348 | 
349 | func (f fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
```

### 74. `internal/delivery/delivery_test.go:387` — `short-label`

Load-bearing: [ ]

```go
384 | 		t.Fatal("doer.Do was called for the metadata literal")
385 | 	}
386 | 
387 | 	// A public-resolving host still posts.
388 | 	if _, err := r.send(context.Background(), "https://hooks.example/hook", body, nil); err != nil {
389 | 		t.Fatalf("send to a public-resolving host errored: %v", err)
390 | 	}
```

### 75. `internal/drift/trend_test.go:8-9` — `docstring-unexported`

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

### 76. `internal/exposure/exposure_test.go:10` — `docstring-unexported`

Load-bearing: [ ]

```go
 7 | 	"github.com/winniel123/verge-asm/internal/custody"
 8 | )
 9 | 
10 | // valued is a shorthand for a leg holding a Reach value.
11 | func valued(v ReachValue) Leg { return Leg{Status: LegValued, Value: v} }
12 | 
13 | // --- ComposeReach: the class-scoped existential quantifier (ADR-0080) -------
```

### 77. `internal/measure/connectoutcome/dialphase_test.go:14-19` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
11 | 	"testing"
12 | )
13 | 
14 | // TestClassifyDialErrorSplitsThePhase pins the connect/handshake split a caller that
15 | // dials an UNVERIFIED address depends on. One tls.Dialer covers both phases under one
16 | // deadline, so the phase is read off the error's shape (a *net.OpError with Op "dial")
17 | // and never guessed from its text. The outcome column is unchanged from the two-way
18 | // classification the `certificate` facet has always used; only the unreachable column
19 | // is new.
20 | func TestClassifyDialErrorSplitsThePhase(t *testing.T) {
21 | 	dialErr := func(inner error) error {
22 | 		return &net.OpError{Op: "dial", Net: "tcp", Err: inner}
```

### 78. `internal/measure/connectoutcome/egressguard_test.go:39-40` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
36 | 	}
37 | }
38 | 
39 | // TestNetConnectorRejectsInvalidTarget pins the entry assertion: an invalid
40 | // (zero) AddrPort is refused before any dial.
41 | func TestNetConnectorRejectsInvalidTarget(t *testing.T) {
42 | 	c := NetConnector{Timeout: 2 * time.Second}
43 | 	if got := c.Connect(context.Background(), netip.AddrPort{}); got != ConnError {
```

### 79. `internal/measure/edgefanout/leaf_test.go:151-152` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
148 | 	}
149 | }
150 | 
151 | // TestRunOneConnectPerAddress pins the run loop: one observation and one handshake per
152 | // DISTINCT candidate, in first-seen order, with an unparseable address skipped.
153 | func TestRunOneConnectPerAddress(t *testing.T) {
154 | 	a := netip.MustParseAddrPort("198.51.100.7:443")
155 | 	b := netip.MustParseAddrPort("203.0.113.9:443")
```

### 80. `internal/measure/resolutionwalk/leaf_test.go:8-9` — `docstring-unexported`

Load-bearing: [ ]

```go
 5 | 	"testing"
 6 | )
 7 | 
 8 | // mapPeer is a tiny scripted peer for the leaf's own unit tests: it answers by
 9 | // (path, qtype, transport, edns, server), returning a silent Msg when unmatched.
10 | type mapPeer struct {
11 | 	fn func(Query) Msg
12 | }
```

### 81. `internal/measure/resolutionwalk/ssrf_test.go:115-119` — `docstring-unexported`

Load-bearing: [ ]

```go
112 | 	}
113 | }
114 | 
115 | // rebindResolver starts an in-process UDP DNS server whose A answer flips from
116 | // pub (first query) to priv (every later query), and returns a *net.Resolver
117 | // wired to it. It models DNS rebinding: the pre-flight vet resolves the NS name
118 | // and sees the public address, then the dial re-resolves the same name and gets
119 | // the private one. AAAA queries answer empty, so only the A record decides.
120 | func rebindResolver(t *testing.T, pub, priv netip.Addr) *net.Resolver {
121 | 	t.Helper()
122 | 	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
```

### 82. `internal/proposer/proposer_test.go:15-17` — `docstring-unexported`

Load-bearing: [ ]

```go
12 | 	"testing"
13 | )
14 | 
15 | // fakeDoer answers requests from an in-memory map keyed by a substring of the
16 | // URL, so no test touches the network. A request whose URL matches no route
17 | // fails loudly, which is what proves the paths never reach a real endpoint.
18 | type fakeDoer struct {
19 | 	routes map[string]string // url substring -> body
20 | 	calls  []string
```

### 83. `internal/queue/crtsh_test.go:126-129` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
123 | 	}
124 | }
125 | 
126 | // TestCTFetchOutcome pins how a fetch result maps to the CT typed outcome (spec §1.2).
127 | // A ctx-killed fetch reads as CTContextCancelled — never a fake transport error — even
128 | // when the http client wraps context.Canceled in a *url.Error. A transport error before
129 | // any status carries its text; any status the fetch returned rides CTHTTP, non-200 too.
130 | func TestCTFetchOutcome(t *testing.T) {
131 | 	// The http client returns ctx errors wrapped (e.g. *url.Error); ctFetchOutcome must
132 | 	// see through the wrap, so test a wrapped context.Canceled and DeadlineExceeded.
```

### 84. `internal/queue/ctverify_test.go:65-66` — `docstring-unexported`

Load-bearing: [ ]

```go
62 | 	return b.BytesOrPanic()
63 | }
64 | 
65 | // leafIndexExtension builds a static-ct-api leaf_index CtExtensions block (type 0, 5-byte
66 | // big-endian index) for a tiled SCT.
67 | func leafIndexExtension(idx int64) []byte {
68 | 	var b cryptobyte.Builder
69 | 	b.AddUint8(0)
```

### 85. `internal/queue/edgefanoutbench_test.go:66-67` — `docstring-unexported`

Load-bearing: [ ]

```go
63 | 	// sans is the count of dNSName SANs, each on its own registrable domain, so the
64 | 	// fan-out count equals this value exactly.
65 | 	sans int
66 | 	// shared is the verdict the reduction must reach. benchCheckFixture asserts it,
67 | 	// which is what holds the two counts either side of the shipped threshold.
68 | 	shared bool
69 | }
70 | 
```

### 86. `internal/queue/edgefanoutread_test.go:53-54` — `docstring-unexported`

Load-bearing: [ ]

```go
50 | 	return der
51 | }
52 | 
53 | // distinctSANs returns n SANs on n distinct registrable domains, so the fan-out count
54 | // equals n exactly.
55 | func distinctSANs(n int) []string {
56 | 	out := make([]string, 0, n)
57 | 	for i := range n {
```

### 87. `internal/queue/transcript_test.go:299` — `docstring-unexported`

Load-bearing: [ ]

```go
296 | 		t.Errorf("zone skips round-trip = %q, want %q", gotStdout, want)
297 | 	}
298 | 
299 | 	// The restated count rides the outcome object.
300 | 	var outcome map[string]any
301 | 	if err := json.Unmarshal(params.Outcome, &outcome); err != nil {
302 | 		t.Fatalf("unmarshal outcome %s: %v", params.Outcome, err)
```

### 88. `internal/remoteexec/probe_test.go:16-18` — `docstring-unexported`

Load-bearing: [ ]

```go
13 | 	"github.com/winniel123/verge-asm/internal/wire"
14 | )
15 | 
16 | // fakeConn is an in-memory Conn: it answers each command from a table and records
17 | // every command run, so the arch check, the push, the exec and the egress read are all
18 | // exercised with no live SSH server.
19 | type fakeConn struct {
20 | 	// outputs maps an Output/rm command to its stdout.
21 | 	outputs map[string]string
```

### 89. `internal/remoteexec/probe_test.go:34` — `docstring-unexported`

Load-bearing: [ ]

```go
31 | 	execErr error
32 | 	// pushErr, if set, fails the `cat > …` push.
33 | 	pushErr error
34 | 	// remoteAddr is the transport peer address the dialled-address read observes.
35 | 	remoteAddr net.Addr
36 | 
37 | 	ran      []string      // every cmd passed to Run
```

### 90. `internal/retention/observation_test.go:142-145` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
139 | 	}
140 | }
141 | 
142 | // TestWithdrawnDialAlone proves the second exception: a withdrawn subject's
143 | // timelines carry no floor at all, so the dial alone governs — a row that would be
144 | // LIVE on an active subject is still retired once past the dial, and with an
145 | // unbounded dial nothing is retired. (AC.)
146 | func TestWithdrawnDialAlone(t *testing.T) {
147 | 	bound := FloorCadences * monthly // a would-be-generous 60-day bound
148 | 	dial := int64(3) * SecondsPerDay
```

### 91. `internal/retention/observation_test.go:193` — `section-divider`

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

### 92. `internal/retention/transcript_test.go:55-57` — `docstring-unexported`

Load-bearing: [ ]

```go
52 | 	}
53 | }
54 | 
55 | // fakeTranscriptStore is the whole surface the TranscriptRetirer can reach. It has
56 | // no Observation, Dispatch, Span or Batch method — the compiler will not let the
57 | // retirer touch anything but the dial and the transcript delete.
58 | type fakeTranscriptStore struct {
59 | 	dial          int64
60 | 	deleteCalled  bool
```

### 93. `internal/scan/cold_test.go:14-16` — `docstring-unexported`

Load-bearing: [ ]

```go
11 | 	"github.com/winniel123/verge-asm/internal/vergecore"
12 | )
13 | 
14 | // coldJobs collects the streamed cold fan-out into a slice for assertions. The
15 | // builder yields one job per (Vantage, admitted address); collecting an empty
16 | // sequence returns nil, so a nil check still reads as "no jobs".
17 | func coldJobs(scanID int64, estate custody.Estate, addrs []netip.Addr, vantages []Vantage, scope ColdScope) []ColdJob {
18 | 	return slices.Collect(BuildColdJobs(scanID, estate, slices.Values(addrs), vantages, scope))
19 | }
```

### 94. `internal/scan/cttail_test.go:196-197` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
193 | 	}
194 | }
195 | 
196 | // TestAdmitCTNamesEmptySeeds confirms that with no declared scope nothing is admitted —
197 | // the tail reads the whole firehose and keeps only what a Seed covers.
198 | func TestAdmitCTNamesEmptySeeds(t *testing.T) {
199 | 	got := AdmitCTNames([]string{"www.example.com"}, nil)
200 | 	if len(got) != 0 {
```

### 95. `internal/scan/ctverify_test.go:191` — `section-divider`

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

### 96. `internal/scan/ctverify_test.go:193` — `docstring-unexported`

Load-bearing: [ ]

```go
190 | 
191 | // --- SCT parsing -------------------------------------------------------------
192 | 
193 | // buildSCT serializes a minimal SignedCertificateTimestamp for a round-trip test.
194 | func buildSCT(logID [32]byte, timestamp uint64, extensions []byte) []byte {
195 | 	var b cryptobyte.Builder
196 | 	b.AddUint8(0) // v1
```

### 97. `internal/scan/hot_test.go:16-18` — `docstring-unexported`

Load-bearing: [ ]

```go
13 | 
14 | func addr(s string) netip.Addr { return netip.MustParseAddr(s) }
15 | 
16 | // hotJobs collects the streamed hot fan-out into a slice for assertions. The
17 | // builder yields one job per (Vantage, admitted address); collecting an empty
18 | // sequence returns nil, so a nil check still reads as "no jobs".
19 | func hotJobs(scanID int64, estate custody.Estate, addrs []netip.Addr, vantages []Vantage, core vergecore.List) []HotJob {
20 | 	return slices.Collect(BuildHotJobs(scanID, estate, slices.Values(addrs), vantages, core))
21 | }
```

### 98. `internal/signal/endpoint_test.go:5-6` — `docstring-unexported`

Load-bearing: [ ]

```go
2 | 
3 | import "testing"
4 | 
5 | // cb is a *bool literal — a read certificate attribute. A nil attribute means the
6 | // datum was not read (not-evaluable); cb(false)/cb(true) are the two read verdicts.
7 | func cb(b bool) *bool { return &b }
8 | 
9 | // presented builds an EndpointFacts with a presented certificate and the given
```

### 99. `internal/signal/rules_test.go:176-177` — `docstring-exported-conventional`

Load-bearing: [ ]

```go
173 | 	}
174 | }
175 | 
176 | // TestRuleNamesStable pins the five names — a rule is named for the fact it
177 | // reads and its name fixes its domain, so a rename is a domain change.
178 | func TestRuleNamesStable(t *testing.T) {
179 | 	want := []string{
180 | 		"lame-delegation",
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

