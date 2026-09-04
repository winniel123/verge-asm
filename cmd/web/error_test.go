package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func TestUnknownPathRendersNotFound(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")

	resp, err := c.Get(base + "/no-such-screen")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown path status = %d, want 404 (body: %s)", resp.StatusCode, got)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("404 Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(got, "Page not found") {
		t.Fatalf("404 page missing its title: %s", got)
	}
}

func TestForbiddenRendersAccessDenied(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, vc, base+"/scans/trigger", url.Values{})
	got := body(t, resp)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer POST /scans/trigger: status=%d, want 403 (body: %s)", resp.StatusCode, got)
	}
	if !strings.Contains(got, "Access denied") || !strings.Contains(got, "widen") {
		t.Fatalf("403 page does not explain the denial: %s", got)
	}
}

var incidentRe = regexp.MustCompile(`err_[0-9a-z]{8}`)

func TestRecoveredPanicRendersIncident(t *testing.T) {
	s := newServer(newFakeStore(), testKey, "", fixedClock())
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("kaboom") })
	ts := httptest.NewServer(s.recoverPanics(boom))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/anything")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("recovered panic status = %d, want 500 (body: %s)", resp.StatusCode, got)
	}
	if !strings.Contains(got, "Something broke") {
		t.Fatalf("500 page missing its title: %s", got)
	}
	id := incidentRe.FindString(got)
	if id == "" {
		t.Fatalf("500 page carries no incident id: %s", got)
	}
	if !strings.Contains(got, `data-incident-copy="`+id+`"`) {
		t.Fatalf("incident id %q not wired to a copy control: %s", id, got)
	}
}

func TestNewIncidentIDShape(t *testing.T) {
	a, b := newIncidentID(), newIncidentID()
	if !incidentRe.MatchString(a) {
		t.Fatalf("incident id %q does not match err_[0-9a-z]{8}", a)
	}
	if a == b {
		t.Fatalf("two incident ids collided: %q", a)
	}
}
