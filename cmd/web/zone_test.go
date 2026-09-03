package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testZone = `$ORIGIN example.com.
@   IN A 203.0.113.10
www IN CNAME example.com.
`

func uploadZone(t *testing.T, c *http.Client, base string, seedID int64, content string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("seed_id", strconv.FormatInt(seedID, 10)); err != nil {
		t.Fatal(err)
	}
	fw, err := mw.CreateFormFile("zonefile", "example.com.zone")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	mw.Close()
	req, err := http.NewRequest(http.MethodPost, base+"/seeds/zone", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	return resp
}

func TestZoneUploadStoresAtSupplyInstantAndListsIt(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A name scope to attach the file to.
	declare(t, ac, base, "name", "example.com").Body.Close()
	seedID := f.seeds[0].ID

	resp := uploadZone(t, ac, base, seedID, testZone)
	// DF-F2: an accepted upload now fires a "zone files supplied" flash across the PRG, so
	// the redirect carries a toast query — match the path prefix, not the whole URL.
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/scope") {
		t.Fatalf("upload: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	if len(f.zoneFiles) != 1 {
		t.Fatalf("stored %d zone files, want 1", len(f.zoneFiles))
	}
	// The supply instant is the upload act's time (the fixed test clock), not a
	// later read.
	if !f.zoneFiles[0].suppliedAt.Equal(fixedClock()()) {
		t.Errorf("supply instant = %s, want the upload clock %s", f.zoneFiles[0].suppliedAt, fixedClock()())
	}
	if f.zoneFiles[0].content != testZone {
		t.Errorf("stored content does not match the uploaded file")
	}

	// The Scope screen shows the supplied file against its scope (frozen scope.tmpl,
	// #574: the zone row names <domain>.zone; the evidence table relocates with Settings).
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "example.com.zone") {
		t.Errorf("supplied zone file not shown on the scope screen; body: %s", page)
	}
}

func TestZoneFileCardSurfacesStalenessGap(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A current file, supplied at the fixed clock, counts down to the gap under
	// the shipped 30-day cadence.
	declare(t, ac, base, "name", "example.com").Body.Close()
	freshID := f.seeds[0].ID
	uploadZone(t, ac, base, freshID, testZone).Body.Close()

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "example.com.zone") {
		t.Errorf("zone-file card does not name the supplied file; body: %s", page)
	}
	if !strings.Contains(page, "ages into a gap in 30d") {
		t.Errorf("current file did not surface its countdown to a gap; body: %s", page)
	}

	// A second scope whose file was supplied 45 days ago has aged past the
	// 30-day cadence into a coverage gap.
	declare(t, ac, base, "name", "staleco.net").Body.Close()
	var staleID int64
	for _, s := range f.seeds {
		if s.NameDomain.String == "staleco.net" {
			staleID = s.ID
		}
	}
	if staleID == 0 {
		t.Fatalf("second name scope was not declared; seeds: %+v", f.seeds)
	}
	f.zoneFiles = append(f.zoneFiles, fakeZoneFile{
		seedID:     staleID,
		suppliedAt: fixedClock()().Add(-45 * 24 * time.Hour),
		content:    testZone,
		uploadedBy: f.zoneFiles[0].uploadedBy,
	})

	page = seedsBody(t, ac, base)
	if !strings.Contains(page, "aged into a gap") {
		t.Errorf("stale file did not surface the resulting gap; body: %s", page)
	}
}

func TestZoneUploadRequiresName_ScopeAndAdmin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "name", "example.com").Body.Close()
	seedID := f.seeds[0].ID

	// A viewer cannot upload.
	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := uploadZone(t, vc, base, seedID, testZone)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer upload: status=%d, want 403", resp.StatusCode)
	}
	if len(f.zoneFiles) != 0 {
		t.Fatalf("viewer upload stored a file; want none")
	}

	// #21c: the handler infers the scope from the file's apex; a zone whose apex is
	// outside every declared name scope is refused with the reason (never attached to a
	// scope the operator did not name).
	foreignZone := "$ORIGIN notmine.example.\n@   IN A 203.0.113.10\n"
	// The refusal is a post-redirect-get (ADR-0130 §1), so the per-file refusal row is
	// read off the landing GET.
	if got := refusalPage(t, ac, base, uploadZone(t, ac, base, seedID, foreignZone)); !strings.Contains(got, "outside every declared name scope") {
		t.Fatalf("upload of a foreign-apex zone not refused; body=%s", got)
	}
}

func TestZoneIntervalIsConfigurable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The shipped default is monthly; the seeds screen shows 30 days.
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, `value="30"`) {
		t.Errorf("default re-supply interval of 30 days not shown; body: %s", page)
	}

	resp := postForm(t, ac, base+"/seeds/zone/interval", url.Values{"interval_days": {"7"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("set interval: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if f.zoneCadence != 7*86400 {
		t.Errorf("zone cadence = %d, want 7 days in seconds", f.zoneCadence)
	}

	// A non-positive interval is rejected and the typed value preserved.
	resp = postForm(t, ac, base+"/seeds/zone/interval", url.Values{"interval_days": {"0"}})
	if got := refusalPage(t, ac, base, resp); !strings.Contains(got, "at least one day") {
		t.Fatalf("zero interval not rejected; body=%s", got)
	}
}
