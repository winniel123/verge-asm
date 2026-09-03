package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/transcript"
)

func seedTranscript(t *testing.T, f *fakeStore, jobID int64, kind string, stdout, stderr, sent []byte, outcome, truncation string, dur time.Duration, capturedAt time.Time) {
	t.Helper()
	seal := func(b []byte) []byte {
		s, err := transcript.Seal(testTranscriptKey, b)
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		return s
	}
	if f.transcriptsByJob == nil {
		f.transcriptsByJob = map[int64]db.Transcript{}
	}
	f.transcriptsByJob[jobID] = db.Transcript{
		QueueJobID: jobID,
		Kind:       kind,
		DurationNs: dur.Nanoseconds(),
		CapturedAt: pgtype.Timestamptz{Time: capturedAt, Valid: true},
		Variant:    "prober",
		Outcome:    []byte(outcome),
		Stdout:     seal(stdout),
		Stderr:     seal(stderr),
		SentScope:  seal(sent),
		Truncation: []byte(truncation),
	}
}

// An admin opens the dedicated raw view from a captured local job and sees its verbatim
// stdout and stderr (carried to the browser as JSON, built with textContent), the exec-meta
// (exit code, duration, captured-at) and the JobSpec sent (#866 AC1, spec §6).
func TestRawOutputRendersCapture(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	captured := time.Date(2026, 8, 29, 14, 22, 5, 0, time.UTC)
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		52: {{ID: 901, Kind: "connect-outcome", State: "done", Attempt: 1, MaxAttempts: 5,
			VantageName: pgtype.Text{String: "local", Valid: true}}},
	}
	seedTranscript(t, f, 901, "connect-outcome",
		[]byte("open=true rtt=41\nbatch-done observed=1024 open=387"),
		[]byte("probe: worker start pid=2831\nprobe: 1024 targets done"),
		[]byte(`{"kind":"connect-outcome","vantage":"local","ports":[80,443]}`),
		`{"kind":"exited","code":0}`, `{}`,
		1837*time.Millisecond, captured)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/run/52/raw?job=901", http.StatusOK)

	for _, want := range []string{
		"Raw output · job #901",              // header
		"Verbatim · unredacted · admin-only", // the admin/verbatim framing
		"connect-outcome", "local",           // kind · vantage badge
		"How it exited",                     // exec-meta card
		"1.837s",                            // duration
		"2026-08-29T14:22:05Z",              // captured-at
		"batch-done observed=1024 open=387", // verbatim stdout (in window.__RAW__)
		"probe: worker start pid=2831",      // verbatim stderr
		`vantage`,                           // the JobSpec sent carried through
	} {
		if !strings.Contains(page, want) {
			t.Errorf("raw view missing %q; body: %s", want, page)
		}
	}

	// Escape-on-render: the verbatim bytes ride a JSON boot value and are built with
	// textContent — never dumped as HTML. The safe pattern is a data script, not innerHTML.
	if !strings.Contains(page, "window.__RAW__ =") {
		t.Errorf("verbatim streams should ride the __RAW__ boot value; body: %s", page)
	}
	if strings.Contains(page, "innerHTML") {
		t.Errorf("the raw view must never use innerHTML (escape-on-render, §6.4); body: %s", page)
	}
}

// seedZoneTranscript stashes one sealed ZONE transcript in the fake: only the stdout role
// column carries bytes (the skipped records); stderr and sent-scope stay NULL, the streams
// zone does not carry (§1.3). The restated count rides the outcome object.
func seedZoneTranscript(t *testing.T, f *fakeStore, jobID int64, skips []byte, outcome string, capturedAt time.Time) {
	t.Helper()
	sealed, err := transcript.Seal(testTranscriptKey, skips)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if f.transcriptsByJob == nil {
		f.transcriptsByJob = map[int64]db.Transcript{}
	}
	f.transcriptsByJob[jobID] = db.Transcript{
		QueueJobID: jobID,
		Kind:       "zone",
		DurationNs: (5 * time.Millisecond).Nanoseconds(),
		CapturedAt: pgtype.Timestamptz{Time: capturedAt, Valid: true},
		Variant:    "zone",
		Outcome:    []byte(outcome),
		Stdout:     sealed,
		Stderr:     nil, // NULL — zone carries no stderr
		SentScope:  nil, // NULL — zone sends nothing
		Truncation: []byte("{}"),
	}
}

// An admin opens the raw view of a zone job and sees the restate result: the restated count,
// the parsed outcome, and the records RestateZone skipped — not the prober exec-meta, which
// zone does not carry (#869 AC4, spec §1.3).
func TestRawOutputRendersZoneVariant(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	captured := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		52: {{ID: 902, Kind: "zone", State: "done", Attempt: 1, MaxAttempts: 1}},
	}
	seedZoneTranscript(t, f, 902,
		[]byte("weird.example.com IN FOO whatever\nempty.example.com IN TXT"),
		`{"kind":"parsed","restated":5}`, captured)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/run/52/raw?job=902", http.StatusOK)

	for _, want := range []string{
		"Raw output · job #902",             // header
		"Restate result",                    // the zone restate card
		"skipped records",                   // the primary panel label
		"weird.example.com IN FOO whatever", // a surfaced skip (in window.__RAW__)
		"parsed",                            // the typed zone outcome
		"zone sends nothing to a prober",    // the zone note
	} {
		if !strings.Contains(page, want) {
			t.Errorf("zone raw view missing %q; body: %s", want, page)
		}
	}
	// The restated count renders in the card.
	if !strings.Contains(page, ">5<") {
		t.Errorf("zone raw view should render the restated count 5; body: %s", page)
	}
	// The prober-only exec-meta must not appear for a zone job.
	if strings.Contains(page, "How it exited") || strings.Contains(page, "Exit code") {
		t.Errorf("zone raw view must not render prober exec-meta; body: %s", page)
	}
}

// seedCTTranscript stashes one sealed CT transcript in the fake: only the stdout role column
// carries bytes (the verbatim response body); stderr and sent-scope stay NULL, the streams the
// crt.sh producer does not carry (§1.2). The request URL and HTTP status ride the outcome object.
func seedCTTranscript(t *testing.T, f *fakeStore, jobID int64, body []byte, outcome string, capturedAt time.Time) {
	t.Helper()
	sealed, err := transcript.Seal(testTranscriptKey, body)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if f.transcriptsByJob == nil {
		f.transcriptsByJob = map[int64]db.Transcript{}
	}
	f.transcriptsByJob[jobID] = db.Transcript{
		QueueJobID: jobID,
		Kind:       "ct",
		DurationNs: (800 * time.Millisecond).Nanoseconds(),
		CapturedAt: pgtype.Timestamptz{Time: capturedAt, Valid: true},
		Variant:    "ct",
		Outcome:    []byte(outcome),
		Stdout:     sealed,
		Stderr:     nil, // NULL — HTTP carries no stderr
		SentScope:  nil, // NULL — a GET sends no request body
		Truncation: []byte("{}"),
	}
}

// An admin opens the raw view of a ct job and sees the crt.sh exchange: the request URL, the
// HTTP outcome, and the verbatim response body — not the prober exec-meta, which ct does not
// carry (#870 AC4, spec §1.2).
func TestRawOutputRendersCTVariant(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	captured := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		52: {{ID: 903, Kind: "ct", State: "done", Attempt: 1, MaxAttempts: 5}},
	}
	seedCTTranscript(t, f, 903,
		[]byte(`[{"name_value":"a.example.com"},{"name_value":"b.example.com"}]`),
		`{"kind":"http","status":200,"request_url":"https://crt.sh/?q=example.com&output=json"}`, captured)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/run/52/raw?job=903", http.StatusOK)

	for _, want := range []string{
		"Raw output · job #903",              // header
		"CT exchange",                        // the CT exchange card
		"response body",                      // the primary panel label
		"a.example.com",                      // a name from the verbatim body (in window.__RAW__)
		"HTTP 200",                           // the typed CT outcome
		"https://crt.sh/?q=example.com",      // the request URL
		"the crt.sh producer sends no stdin", // the CT note
	} {
		if !strings.Contains(page, want) {
			t.Errorf("ct raw view missing %q; body: %s", want, page)
		}
	}
	// The prober-only exec-meta must not appear for a ct job.
	if strings.Contains(page, "How it exited") || strings.Contains(page, "Exit code") {
		t.Errorf("ct raw view must not render prober exec-meta; body: %s", page)
	}
}

// A job that produced no capture renders a legible "No transcript captured" absence — not a
// 404, and distinct from a captured-but-empty stream.
func TestRawOutputNoCapture(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/run/52/raw?job=404", http.StatusOK)

	if !strings.Contains(page, "No transcript captured") {
		t.Errorf("an uncaptured job should render the legible absence; body: %s", page)
	}
	// No verbatim panel is booted when there is nothing to show.
	if strings.Contains(page, "window.__RAW__ =") {
		t.Errorf("an uncaptured job should boot no verbatim streams; body: %s", page)
	}
}

// The raw view is admin-only (requireAdmin, spec §5.2): a viewer — who can read the redacted
// run/job log — is refused with a 403, an intentional escalation above today's log gate.
func TestRawOutputAdminOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	captured := time.Date(2026, 8, 29, 14, 22, 5, 0, time.UTC)
	seedTranscript(t, f, 901, "connect-outcome",
		[]byte("secret stdout"), []byte("secret stderr"), []byte("secret spec"),
		`{"kind":"exited","code":0}`, `{}`, time.Second, captured)

	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	// The viewer is refused, and none of the sealed content leaks in the 403 body.
	page := getBody(t, vc, base+"/run/52/raw?job=901", http.StatusForbidden)
	for _, secret := range []string{"secret stdout", "secret stderr", "secret spec"} {
		if strings.Contains(page, secret) {
			t.Errorf("a refused viewer must not see verbatim content %q; body: %s", secret, page)
		}
	}
}

// The raw view is login-gated: an unauthenticated request redirects to /login, exactly like
// the run page it hangs off.
func TestRawOutputRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/run/52/raw?job=901")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /run/{id}/raw: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

// The "Raw output (admin)" affordance on the ?job filter chip is admin-only: an admin sees it,
// a viewer sees the same filtered run page without it (the redacted log stays readable to both).
func TestRawOutputAffordanceAdminOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")

	tick := time.Date(2026, 8, 29, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(52, "hot", tick, 1, 0, 0, 1, 0, 0),
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		52: {{ID: 901, Kind: "connect-outcome", State: "done", Attempt: 1, MaxAttempts: 5,
			VantageName: pgtype.Text{String: "local", Valid: true}}},
	}

	base := start(t, f, "")

	adminC := login(t, base, "admin", "hunter2hunter2")
	adminPage := getBody(t, adminC, base+"/run/52?job=901", http.StatusOK)
	if !strings.Contains(adminPage, "Raw output (admin)") {
		t.Errorf("admin should see the raw-output affordance on the chip; body: %s", adminPage)
	}
	if !strings.Contains(adminPage, `href="/run/52/raw?job=901"`) {
		t.Errorf("the affordance should link to the job-scoped raw view; body: %s", adminPage)
	}

	viewerC := login(t, base, "viewer", "hunter2hunter2")
	viewerPage := getBody(t, viewerC, base+"/run/52?job=901", http.StatusOK)
	if strings.Contains(viewerPage, "Raw output (admin)") {
		t.Errorf("a viewer must not see the raw-output affordance; body: %s", viewerPage)
	}
	// The redacted log and its chip stay readable to the viewer.
	if !strings.Contains(viewerPage, "job #901") {
		t.Errorf("the viewer should still read the redacted job-filtered log; body: %s", viewerPage)
	}
}
