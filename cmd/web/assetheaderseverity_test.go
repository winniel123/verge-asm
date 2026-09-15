package main

import (
	"net/http"
	"strings"
	"testing"
)

const severityDidNotResolve = "Severity did not resolve"

func TestAssetHeaderNamesAFailedSignalRead(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/lame.example.com", http.StatusOK)

	hdr := assetHeaderOf(t, page)
	if !strings.Contains(hdr, severityDidNotResolve) {
		t.Errorf("header does not name the failed read; header: %s", hdr)
	}
	if strings.Contains(hdr, "var(--sev-") {
		t.Errorf("a failed read raised a header severity badge; header: %s", hdr)
	}
	for _, banned := range []string{"Critical", "High", "Medium", "Low", "Info"} {
		if strings.Contains(hdr, banned) {
			t.Errorf("the header named severity %q on a read that did not resolve; header: %s", banned, hdr)
		}
	}
	if !strings.Contains(page, signalsDidNotResolve) {
		t.Errorf("the signals card lost its own did-not-resolve state; body: %s", page)
	}
}

func TestWithdrawnAssetHeaderNamesAFailedSignalRead(t *testing.T) {
	f := withdrawnNameStore(t)
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusOK)

	hdr := assetHeaderOf(t, page)
	if !strings.Contains(hdr, severityDidNotResolve) {
		t.Errorf("withdrawn asset header does not name the failed read; header: %s", hdr)
	}
	if strings.Contains(hdr, "var(--sev-") {
		t.Errorf("a failed read raised a header severity badge; header: %s", hdr)
	}
}

func TestAssetHeaderStaysSilentWhenNoRuleFires(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	// A signal-free asset keeps a bare header, which is the distinction the note exists to draw.
	hdr := assetHeaderOf(t, page)
	if strings.Contains(hdr, severityDidNotResolve) {
		t.Errorf("an asset no rule fires on claimed a failed read; header: %s", hdr)
	}
	if strings.Contains(hdr, "var(--sev-") {
		t.Errorf("an asset no rule fires on raised a severity badge; header: %s", hdr)
	}
}

func TestWithdrawnAssetHeaderStaysSilentWhenNoRuleFires(t *testing.T) {
	f := withdrawnNameStore(t)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusOK)

	hdr := assetHeaderOf(t, page)
	if strings.Contains(hdr, severityDidNotResolve) {
		t.Errorf("a withdrawn asset with an empty signal list claimed a failed read; header: %s", hdr)
	}
	if strings.Contains(hdr, "var(--sev-") {
		t.Errorf("a withdrawn asset raised a severity badge; header: %s", hdr)
	}
}

func TestAssetHeaderKeepsItsSeverityBadgeOnAResolvedRead(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/lame.example.com", http.StatusOK)

	hdr := assetHeaderOf(t, page)
	if !strings.Contains(hdr, "var(--sev-") {
		t.Errorf("a resolved read lost its header severity badge; header: %s", hdr)
	}
	if strings.Contains(hdr, severityDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; header: %s", hdr)
	}
}
