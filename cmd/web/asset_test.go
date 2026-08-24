package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// The Asset detail screen (#296, T1) — the per-asset drill-in ported from
// design-system/examples/console/AssetDetail.jsx. It renders all six sections for
// a Name subject, wiring real Name-scoped data where it exists (ports census, DNS,
// provenance, drift trail) and the design-system empty-state where it does not
// (the TLS certificate's parsed identity is not stored).
func TestAssetDetailRendersSections(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// A Name that resolves to an address a Service sits on, with a TXT record and an
	// open reachable port — the ground the census, DNS and drift trail read.
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addDNSRecord(t, "api.example.com", "TXT", obsClock, `{"rrs":[{"name":"api.example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	// All six sections render, in the example's vocabulary.
	for _, want := range []string{
		"Open ports",       // ports census
		"DNS records",      // resolution
		"TLS certificate",  // cert
		"How it got here",  // provenance
		"Signals here",     // signals
		"Drift trail",      // history
	} {
		if !strings.Contains(page, want) {
			t.Errorf("asset detail missing section %q; body: %s", want, page)
		}
	}

	// Real data wired: the resolved address and TXT contents (DNS), the open port
	// with its exposure state (census), and the change kind (drift trail).
	for _, want := range []string{
		"198.51.100.1", // resolved A record
		"v=spf1 -all",  // TXT record contents
		":443",         // the open port
		"exposed",      // reachability verdict rendered as an exposure state
		"appeared",     // drift trail change kind (change palette, not severity)
	} {
		if !strings.Contains(page, want) {
			t.Errorf("asset detail missing wired value %q; body: %s", want, page)
		}
	}

	// The TLS certificate section holds no honest parsed value, so it renders the
	// design-system empty-state rather than a fabricated card.
	if !strings.Contains(page, "No certificate detail to show") {
		t.Errorf("TLS cert section did not fall to its empty-state; body: %s", page)
	}

	// The breadcrumb roots at Inventory (the row-click origin) and the Inventory nav
	// pill is the active one — keyed on NavActive.
	if !strings.Contains(page, `href="/inventory"`) {
		t.Errorf("asset detail breadcrumb missing Inventory root; body: %s", page)
	}
	if !strings.Contains(page, `class="navpill active" href="/inventory"`) {
		t.Errorf("asset detail nav pill not marked active; body: %s", page)
	}

	// No technology fingerprinting anywhere in the census: the ports table carries
	// the transport, never a product/version banner.
	for _, banned := range []string{"nginx", "OpenSSH", "/1.2", "/1.25"} {
		if strings.Contains(page, banned) {
			t.Errorf("asset detail leaked a technology fingerprint %q; body: %s", banned, page)
		}
	}
}

// A rule firing on the Name lights its "Signals here" row with the rule's
// SeverityBadge. Severity is a real per-rule datum (internal/signal SeverityFor,
// P0.1 #442), the same ramp Signals, Graph and Search read, so the spec's badge
// (AssetDetail.jsx) renders rather than being held back as "would be fabricated"
// (the P2.7 empty-state-while-datum-exists fix). lame-delegation is rated medium.
func TestAssetDetailSignalsHereCarrySeverity(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/lame.example.com", http.StatusOK)

	// The firing rule is listed in "Signals here" (a rule slug never leaks into the
	// shell palette, so a page-wide match is unambiguous here).
	if !strings.Contains(page, "lame-delegation") {
		t.Errorf("Signals here did not list the firing rule; body: %s", page)
	}
	// It leads with the rule's SeverityBadge — the medium ramp, read never fabricated.
	if !strings.Contains(page, `class="sev sev-medium"`) {
		t.Errorf("Signals here row missing its medium SeverityBadge; body: %s", page)
	}
}

// A Name measured gone renders the withdrawn notice — it is reached by its own key
// and named as a population of no current member, never a false absence.
func TestAssetDetailWithdrawn(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock, `{"outcome":"Resolved"}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"NameError"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusOK)

	if !strings.Contains(page, "withdrawn") {
		t.Errorf("withdrawn asset lost its withdrawal notice; body: %s", page)
	}
}

// A Name nothing has ever measured is not a subject — the drill-down 404s to the
// subject-missing page rather than manufacturing a false record.
func TestAssetDetailUnknownName(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	getBody(t, ac, base+"/asset/never.measured.example", http.StatusNotFound)
}

func TestAssetDetailRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/asset/api.example.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /asset: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
