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
		"Open ports",      // ports census
		"DNS records",     // resolution
		"TLS certificate", // cert
		"How it got here", // provenance
		"Signals here",    // signals
		"Drift trail",     // history
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

	// Absent any http-identity evidence, the census Service column is transport-only
	// (#22d joins a Server banner ONLY where an Endpoint on the port holds one — none
	// seeded here), so no product/version string appears.
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
	// It leads with the rule's SeverityBadge — the medium ramp rendered through the
	// frozen "sevbadge" define (var(--sev-medium-*)), read never fabricated (#22a).
	if !strings.Contains(page, "var(--sev-medium-bg)") {
		t.Errorf("Signals here row missing its medium SeverityBadge; body: %s", page)
	}
	// The row deep-links to the Signals drawer by the rule's REAL minted SIG-#### id
	// (#22b) — the same id the /signals?view= drawer resolves, never a mock VG id.
	if !strings.Contains(page, `href="/signals?view=SIG-`) {
		t.Errorf("Signals here row missing its /signals?view=SIG-#### deep-link; body: %s", page)
	}
}

// The header identity carries the aggregate SeverityBadge (the most urgent firing
// severity) and ExposureBadge (the worst reachability across the open ports) the
// spec shows (AssetDetail.jsx:35-36). Both are rolled up from the datums the rows
// already carry — nothing is fabricated — and each omits when its datum is absent.
func TestAssetDetailHeaderAggregateBadges(t *testing.T) {
	// assetHeader slices the identity header off the page (its h1 carries the
	// distinctive 21px style) so the badge assertions can't collide with the census
	// or signals-here rows further down.
	assetHeader := func(page string) string {
		from := strings.Index(page, `aria-label="Breadcrumb"`)
		if from < 0 {
			return ""
		}
		hdr := page[from:]
		if end := strings.Index(hdr, "</header>"); end >= 0 {
			hdr = hdr[:end]
		}
		return hdr
	}

	// SeverityBadge: a Name holding a lame delegation fires lame-delegation (medium),
	// so the header rolls it up as the aggregate SeverityBadge — the frozen "sevbadge"
	// define (var(--sev-medium-*)), its label the capitalised token (#22a).
	t.Run("severity", func(t *testing.T) {
		f := newFakeStore()
		seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
		lameName(t, f, "lame.example.com")

		base := start(t, f, "")
		ac := login(t, base, "admin", "hunter2hunter2")
		page := getBody(t, ac, base+"/asset/lame.example.com", http.StatusOK)

		hdr := assetHeader(page)
		if !strings.Contains(hdr, "var(--sev-medium-bg)") {
			t.Errorf("header missing aggregate SeverityBadge; header: %s", hdr)
		}
		if !strings.Contains(hdr, "Medium") {
			t.Errorf("header SeverityBadge missing its capitalised SevLabel; header: %s", hdr)
		}
	})

	// ExposureBadge: a Name with an exposed open port rolls up to an "exposed"
	// aggregate ExposureBadge in the header (the frozen "assetexposure" define).
	t.Run("exposure", func(t *testing.T) {
		f := newFakeStore()
		admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
		addNameSeed(t, f, admin.ID, "example.com")
		f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
		f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)

		base := start(t, f, "")
		ac := login(t, base, "admin", "hunter2hunter2")
		page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

		hdr := assetHeader(page)
		if !strings.Contains(hdr, `class="as-leg exposed"`) {
			t.Errorf("header missing aggregate ExposureBadge; header: %s", hdr)
		}
	})
}

// The TLS certificate card renders the parsed identity folded off the chain leaf
// (#22c): a presented certificate on an Endpoint keyed under the Name surfaces its
// leaf fingerprint, issuer, algorithm and expiry, with a precomputed validity badge —
// never the empty state. The issuer + algorithm are the leaf's own stored attributes
// (a read, not a fingerprint), honestly omitted only when a pre-parse span holds none.
func TestAssetDetailCertificateCard(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addCertificate(t, "api.example.com@198.51.100.1:443/tcp", obsClock,
		`{"outcome":"presented","chain":["sha256:leaf01","sha256:int01"],"not_after":"2027-03-01T12:00:00Z","issuer":"CN=R11, O=Let's Encrypt","algorithm":"ECDSA-SHA256"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	if strings.Contains(page, "No certificate detail to show") {
		t.Errorf("certificate card fell to the empty state despite a presented leaf; body: %s", page)
	}
	for _, want := range []string{
		"sha256:leaf01",      // the leaf fingerprint (chain[0])
		"CN=R11",             // the parsed issuer DN
		"ECDSA-SHA256",       // the parsed signature algorithm
		"2027-03-01",         // the leaf expiry as a date
		"valid",              // the precomputed validity label
		`class="as-badge ok`, // the ok validity tone (well past 30d out)
	} {
		if !strings.Contains(page, want) {
			t.Errorf("certificate card missing %q; body: %s", want, page)
		}
	}
}

// The census Service column joins the transport with the http-identity Server an
// Endpoint on that port holds (#22d) — a read of stored evidence. Where no Endpoint
// holds an http-identity, the column is transport-only.
func TestAssetDetailPortServiceJoinsHTTPIdentity(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)
	f.addHTTPIdentity(t, "api.example.com@198.51.100.1:443/tcp", obsClock, `{"outcome":"answered","status":200,"server":"nginx/1.25.0"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	// The Service cell reads "tcp · nginx/1.25.0" — transport joined with the Server.
	if !strings.Contains(page, "tcp · nginx/1.25.0") {
		t.Errorf("census Service did not join transport with the http-identity Server; body: %s", page)
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
// missing-subject ErrorPage (U3, #480) rather than manufacturing a false record: the
// unmatched key big-mono, the "no subject is keyed under that name" copy, and the way
// back to Inventory, all inside the console chrome.
func TestAssetDetailUnknownName(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/asset/never.measured.example", http.StatusNotFound)
	for _, want := range []string{"No such subject", "No subject is keyed under that name", "never.measured.example", "Back to inventory"} {
		if !strings.Contains(page, want) {
			t.Errorf("missing-subject page missing %q; body: %s", want, page)
		}
	}
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
