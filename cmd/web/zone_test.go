package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

const testZone = `$ORIGIN example.com.
@   IN A 203.0.113.10
www IN CNAME example.com.
`

// uploadZone posts a multipart zone-file upload for the given seed id.
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
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/seeds" {
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

	// The Seeds screen shows the supplied file against its scope.
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "Supplied zone files") || !strings.Contains(page, "example.com") {
		t.Errorf("supplied zone file not shown on the seeds screen; body: %s", page)
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

	// An upload against a non-existent scope is rejected clearly.
	resp = uploadZone(t, ac, base, 9999, testZone)
	if got := body(t, resp); resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "no longer exists") {
		t.Fatalf("upload to a missing scope not rejected: status=%d body=%s", resp.StatusCode, got)
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

	// The operator moves the dial.
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
	if got := body(t, resp); resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "at least one day") {
		t.Fatalf("zero interval not rejected: status=%d body=%s", resp.StatusCode, got)
	}
}
