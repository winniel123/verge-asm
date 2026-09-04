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
		"Raw output · job #901",
		"Verbatim · unredacted · admin-only",
		"connect-outcome", "local",
		"How it exited",
		"1.837s",
		"2026-08-29T14:22:05Z",
		"batch-done observed=1024 open=387",
		"probe: worker start pid=2831",
		`vantage`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("raw view missing %q; body: %s", want, page)
		}
	}

	if !strings.Contains(page, "window.__RAW__ =") {
		t.Errorf("verbatim streams should ride the __RAW__ boot value; body: %s", page)
	}
	if strings.Contains(page, "innerHTML") {
		t.Errorf("the raw view must never use innerHTML (escape-on-render, §6.4); body: %s", page)
	}
}

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
		Stderr:     nil, // zone carries neither stream (raw-job-output.md §1.3)
		SentScope:  nil,
		Truncation: []byte("{}"),
	}
}

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
		"Raw output · job #902",
		"Restate result",
		"skipped records",
		"weird.example.com IN FOO whatever",
		"parsed",
		"zone sends nothing to a prober",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("zone raw view missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, ">5<") {
		t.Errorf("zone raw view should render the restated count 5; body: %s", page)
	}
	if strings.Contains(page, "How it exited") || strings.Contains(page, "Exit code") {
		t.Errorf("zone raw view must not render prober exec-meta; body: %s", page)
	}
}

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
		Stderr:     nil, // the transport error rides the typed outcome (raw-job-output.md §1.2)
		SentScope:  nil, // a GET request sends no body of its own
		Truncation: []byte("{}"),
	}
}

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
		"Raw output · job #903",
		"CT exchange",
		"response body",
		"a.example.com",
		"HTTP 200",
		"https://crt.sh/?q=example.com",
		"the crt.sh producer sends no stdin",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("ct raw view missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "How it exited") || strings.Contains(page, "Exit code") {
		t.Errorf("ct raw view must not render prober exec-meta; body: %s", page)
	}
}

func TestRawOutputNoCapture(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/run/52/raw?job=404", http.StatusOK)

	if !strings.Contains(page, "No transcript captured") {
		t.Errorf("an uncaptured job should render the legible absence; body: %s", page)
	}
	if strings.Contains(page, "window.__RAW__ =") {
		t.Errorf("an uncaptured job should boot no verbatim streams; body: %s", page)
	}
}

func TestRawOutputAdminOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	captured := time.Date(2026, 8, 29, 14, 22, 5, 0, time.UTC)
	seedTranscript(t, f, 901, "connect-outcome",
		[]byte("secret stdout"), []byte("secret stderr"), []byte("secret spec"),
		`{"kind":"exited","code":0}`, `{}`, time.Second, captured)

	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	page := getBody(t, vc, base+"/run/52/raw?job=901", http.StatusForbidden)
	for _, secret := range []string{"secret stdout", "secret stderr", "secret spec"} {
		if strings.Contains(page, secret) {
			t.Errorf("a refused viewer must not see verbatim content %q; body: %s", secret, page)
		}
	}
}

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
	if !strings.Contains(viewerPage, "job #901") {
		t.Errorf("the viewer should still read the redacted job-filtered log; body: %s", viewerPage)
	}
}
