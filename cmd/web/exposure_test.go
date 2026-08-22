package main

import (
	"net/http"
	"testing"
)

// The Exposure analytics folded into the Reports period view (#286, map #275):
// the old /exposure landing GET is now a permanent redirect to /reports, so the
// route reconciliation is proven by the redirect rather than by rendering the
// board here. The board's own rendering is exercised by the Reports screen
// (#285); the reachability-span wiring behind the Reports exposure sections is a
// tracked #285 gap (see reports.go).
func TestExposureRedirectsToReports(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/exposure")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMovedPermanently || resp.Header.Get("Location") != "/reports" {
		t.Fatalf("GET /exposure: status=%d location=%q, want 301 -> /reports",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

// An unauthenticated hit on the deprecated route keeps the login gate it carried
// before the move — it lands on /login, not straight onto the redirect target.
func TestExposureRedirectRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)

	resp, err := c.Get(base + "/exposure")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon GET /exposure: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
