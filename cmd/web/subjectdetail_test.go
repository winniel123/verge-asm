package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// The Service drill-in ported to the v3.2 SubjectDetail spec (U1, #478,
// SubjectDetail.jsx / screenshot 22): the shared skeleton — breadcrumb, header with
// the kind tag + an ExposureBadge, the citation chain, the current reachability
// facet, the timelines card, the rules-over-subject table, and the provenance +
// signals-here rail. A sensitive Service reached from the internet fires
// sensitive-port-reached-from-internet (critical), so the rules table carries its
// real per-rule SeverityBadge and a "fired" verdict, and the rail lists it.
func TestServiceDetailRendersV32Composition(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// A Name resolves to the Service's address (the citation ground), and the
	// sensitive port is reached from an internet-class vantage (the firing predicate).
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addClassReachability(t, "198.51.100.1:3389/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A3389%2Ftcp", http.StatusOK)

	for _, want := range []string{
		// Breadcrumb + header + nav
		`href="/inventory"`, `class="navpill active" href="/inventory"`,
		`<span class="sd-tag">service</span>`,
		`class="as-leg exposed">exposed`, // header ExposureBadge rolled up from reachability (assetexposure)
		"198.51.100.1:3389/tcp",          // the key
		// The card composition
		"Citation chain", "Reachability", "Current and closed timelines",
		"Rules over this subject", "How it got here", "Signals here",
		// The firing rule carries its real severity + verdict (design sevbadge), and rides the rail.
		"sensitive-port-reached-from-internet", "var(--sev-critical-fill)", "fired",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("service detail missing %q; body: %s", want, drill)
		}
	}
}

// The Endpoint drill-in ported to the v3.2 SubjectDetail spec (U1, #478,
// SubjectDetail.jsx / screenshot 23): the same skeleton with the HTTP-identity facet
// in place of reachability, and no signals-here rail (a Service-only affordance in
// the spec). The admitted HTTP identity renders — its status, Server header and
// title — as the endpoint's own volunteered identity (ADR-0011), never a
// fingerprint the product inferred.
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
	// The signals-here rail is a Service-only affordance in the spec.
	if strings.Contains(drill, "Signals here") {
		t.Errorf("endpoint detail should not carry the Service-only signals-here rail; body: %s", drill)
	}
}

// The withdrawn Service (screenshot 24): its address has left the estate — no
// resolution cites it and no Seed covers it — so it names a population of no current
// member. The header marks it withdrawn (not an exposure), the withdrawn banner
// renders, Rescan is disabled, and the signals-here rail is withheld.
func TestServiceDetailWithdrawn(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// Reached, but the address is cited by no resolution and covered by no Seed.
	f.addClassReachability(t, "203.0.113.99:5900/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=203.0.113.99%3A5900%2Ftcp", http.StatusOK)

	for _, want := range []string{
		`class="sd-wmark"`, // the header dashed WithdrawnMark (design)
		"withdrawn",
		"Withdrawn by the world",              // the neutral withdrawn Banner heading
		"left the estate", "no current member", // its body prose
		`<button class="sd-btn" disabled`, // Rescan disabled for a withdrawn subject
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("withdrawn service detail missing %q; body: %s", want, drill)
		}
	}
	// A withdrawn Service names no current member, so it shows no ExposureBadge and no
	// signals-here rail.
	if strings.Contains(drill, "Signals here") {
		t.Errorf("withdrawn service should withhold the signals-here rail; body: %s", drill)
	}
	if strings.Contains(drill, `class="as-leg exposed"`) {
		t.Errorf("withdrawn service should show no ExposureBadge; body: %s", drill)
	}
}

// The ruling (SubjectDetail.jsx): a Name subject opens AssetDetail, never
// SubjectDetail. The by-key /subjects/{name} path delegates to the Asset detail —
// so its distinctive composition renders and the Service/Endpoint-only cards do not.
func TestNameByKeyRoutesToAssetDetailNotSubjectDetail(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/api.example.com", http.StatusOK)

	// AssetDetail's own sections render.
	for _, want := range []string{"Open ports", "DNS records", "How it got here", "Drift trail"} {
		if !strings.Contains(drill, want) {
			t.Errorf("Name by-key did not open AssetDetail (missing %q); body: %s", want, drill)
		}
	}
	// The SubjectDetail-only cards (Service/Endpoint) must not render for a Name.
	for _, absent := range []string{"Rules over this subject", "Current and closed timelines"} {
		if strings.Contains(drill, absent) {
			t.Errorf("Name drill-in rendered the SubjectDetail-only card %q; body: %s", absent, drill)
		}
	}
}
