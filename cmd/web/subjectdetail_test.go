package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestServiceDetailRendersV32Composition(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addClassReachability(t, "198.51.100.1:3389/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A3389%2Ftcp", http.StatusOK)

	for _, want := range []string{
		`href="/inventory"`, `class="sh-pill on" href="/inventory"`,
		`<span class="sd-tag">service</span>`,
		`class="as-leg exposed">exposed`,
		"198.51.100.1:3389/tcp",
		"Citation chain", "Reachability", "Current and closed timelines",
		"Rules over this subject", "How it got here", "Signals here",
		"sensitive-port-reached-from-internet", "var(--sev-critical-fill)", "fired",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("service detail missing %q; body: %s", want, drill)
		}
	}
}

func TestEndpointDetailRendersV32Composition(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addHTTPIdentity(t, "api.example.com@198.51.100.1:443/tcp", obsClock,
		`{"outcome":"responded","status":200,"server":"nginx","title":"Example API"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	key := url.QueryEscape("api.example.com@198.51.100.1:443/tcp")
	drill := getBody(t, ac, base+"/subjects/endpoint?key="+key, http.StatusOK)

	for _, want := range []string{
		`href="/inventory"`, `<span class="sd-tag">endpoint</span>`,
		"api.example.com@198.51.100.1:443/tcp",
		"Citation chain", "HTTP identity", "Current and closed timelines",
		"Rules over this subject", "How it got here",
		"nginx", "Example API", "200",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("endpoint detail missing %q; body: %s", want, drill)
		}
	}
	if strings.Contains(drill, "Signals here") {
		t.Errorf("endpoint detail should not carry the Service-only signals-here rail; body: %s", drill)
	}
}

func TestServiceDetailWithdrawn(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "203.0.113.99:5900/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=203.0.113.99%3A5900%2Ftcp", http.StatusOK)

	for _, want := range []string{
		`class="sd-wmark"`,
		"withdrawn",
		"Withdrawn by the world",
		"left the estate", "no current member",
		`<button class="sd-btn" disabled`,
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("withdrawn service detail missing %q; body: %s", want, drill)
		}
	}
	if strings.Contains(drill, "Signals here") {
		t.Errorf("withdrawn service should withhold the signals-here rail; body: %s", drill)
	}
	if strings.Contains(drill, `class="as-leg exposed"`) {
		t.Errorf("withdrawn service should show no ExposureBadge; body: %s", drill)
	}
}

func TestNameByKeyRoutesToAssetDetailNotSubjectDetail(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/api.example.com", http.StatusOK)

	for _, want := range []string{"Open ports", "DNS records", "How it got here", "Drift trail"} {
		if !strings.Contains(drill, want) {
			t.Errorf("Name by-key did not open AssetDetail (missing %q); body: %s", want, drill)
		}
	}
	for _, absent := range []string{"Rules over this subject", "Current and closed timelines"} {
		if strings.Contains(drill, absent) {
			t.Errorf("Name drill-in rendered the SubjectDetail-only card %q; body: %s", absent, drill)
		}
	}
}
