package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// editFreq submits one frequency edit from the folded /verge-core surface, carrying the
// submitting-URL field that surface's forms stamp (ADR-0130 §3, backurl.go). Every
// outcome therefore comes back to /verge-core, which is where the operator was.
func editFreq(t *testing.T, c *http.Client, base, action, port string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/verge-core/frequency", url.Values{
		"action": {action}, "port": {port}, "return": {"/verge-core"},
	})
}

func vergeCoreBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	resp, err := c.Get(base + "/verge-core")
	if err != nil {
		t.Fatal(err)
	}
	return body(t, resp)
}

// The page states the composed set and lists both halves; the sensitive half is
// read-only (no mutating control targets it).
func TestVergeCorePageShowsComposition(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := vergeCoreBody(t, ac, base)
	// verge-core now renders as the Port aperture tab (#313): sensitive tier locked,
	// frequency tier editable. The composition counts still read from real data.
	for _, want := range []string{"136 pairs", "131 TCP", "5 UDP", "Sensitive tier", "locked", "Not editable"} {
		if !strings.Contains(page, want) {
			t.Errorf("aperture page missing %q; body length %d", want, len(page))
		}
	}
	// The sensitive half is offered no edit control — only the frequency half is.
	if strings.Contains(page, `value="add"`) == false {
		t.Errorf("frequency add control missing")
	}
}

// An admin can add and remove a frequency port; the edit is stored as a delta.
func TestVergeCoreAdminEditsFrequency(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := editFreq(t, ac, base, "add", "12345")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("add frequency port: status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()
	if e, ok := f.freqEdits[12345]; !ok || e.action != "add" {
		t.Fatalf("add edit not stored: %+v", f.freqEdits)
	}
	if page := vergeCoreBody(t, ac, base); !strings.Contains(page, "12345/tcp") {
		t.Errorf("added port not listed in the frequency half")
	}

	// Reset drops the delta row.
	editFreq(t, ac, base, "reset", "12345").Body.Close()
	if _, ok := f.freqEdits[12345]; ok {
		t.Errorf("reset did not drop the edit row: %+v", f.freqEdits)
	}
}

// A bad port is rejected with the typed value preserved, and stores nothing. The
// refusal is a post-redirect-get (ADR-0130 §1, ticket #975), so the message and the
// typed port reach the operator on the landing GET rather than as a 400 body at the
// POST URL.
func TestVergeCoreRejectsBadPort(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := editFreq(t, ac, base, "add", "70000")
	if loc := submitLoc(t, resp); loc != "/verge-core" {
		t.Fatalf("rejected port landed at %q, want /verge-core", loc)
	}
	if len(f.freqEdits) != 0 {
		t.Errorf("a rejected edit stored a row: %+v", f.freqEdits)
	}
	got := vergeCoreBody(t, ac, base)
	if !strings.Contains(got, "between 1 and 65535") {
		t.Fatalf("bad port not rejected clearly on the landing; body len=%d", len(got))
	}
	if !strings.Contains(got, `value="70000"`) {
		t.Fatalf("the typed port was not echoed back; body len=%d", len(got))
	}
}

// A viewer reads the composition but is offered and allowed no edit.
func TestVergeCoreViewerReadOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := editFreq(t, vc, base, "add", "8443")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer edit: status=%d, want 403", resp.StatusCode)
	}
	if len(f.freqEdits) != 0 {
		t.Errorf("a viewer edit changed the set: %+v", f.freqEdits)
	}
	page := vergeCoreBody(t, vc, base)
	if !strings.Contains(page, "136 pairs") {
		t.Errorf("viewer cannot read the composition")
	}
	if strings.Contains(page, `action="/verge-core/frequency"`) {
		t.Errorf("an edit control was shown to a viewer")
	}
}
