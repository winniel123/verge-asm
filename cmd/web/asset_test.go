package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAssetDetailRendersSections(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addDNSRecord(t, "api.example.com", "TXT", obsClock, `{"rrs":[{"name":"api.example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	for _, want := range []string{
		"Open ports",
		"DNS records",
		"TLS certificate",
		"How it got here",
		"Signals here",
		"Drift trail",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("asset detail missing section %q; body: %s", want, page)
		}
	}

	for _, want := range []string{
		"198.51.100.1",
		"v=spf1 -all",
		":443",
		"exposed",
		"appeared",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("asset detail missing wired value %q; body: %s", want, page)
		}
	}

	if !strings.Contains(page, "No certificate detail to show") {
		t.Errorf("TLS cert section did not fall to its empty-state; body: %s", page)
	}

	if !strings.Contains(page, `href="/inventory"`) {
		t.Errorf("asset detail breadcrumb missing Inventory root; body: %s", page)
	}
	if !strings.Contains(page, `class="sh-pill on" href="/inventory"`) {
		t.Errorf("asset detail nav pill not marked active; body: %s", page)
	}

	for _, banned := range []string{"nginx", "OpenSSH", "/1.2", "/1.25"} {
		if strings.Contains(page, banned) {
			t.Errorf("asset detail leaked a technology fingerprint %q; body: %s", banned, page)
		}
	}
}

func TestAssetDetailSignalsHereCarrySeverity(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/lame.example.com", http.StatusOK)

	// A rule slug never leaks into the shell palette, so a page-wide match is unambiguous.
	if !strings.Contains(page, "lame-delegation") {
		t.Errorf("Signals here did not list the firing rule; body: %s", page)
	}
	if !strings.Contains(page, "var(--sev-medium-bg)") {
		t.Errorf("Signals here row missing its medium SeverityBadge; body: %s", page)
	}
	if !strings.Contains(page, `href="/signals?view=SIG-`) {
		t.Errorf("Signals here row missing its /signals?view=SIG-#### deep-link; body: %s", page)
	}
}

func TestAssetDetailHeaderAggregateBadges(t *testing.T) {
	// A page-wide badge match would collide with the census and signals-here rows below.
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
		"sha256:leaf01",
		"CN=R11",
		"ECDSA-SHA256",
		"2027-03-01",
		"valid",
		`class="as-badge ok`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("certificate card missing %q; body: %s", want, page)
		}
	}
}

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

	if !strings.Contains(page, "tcp · nginx/1.25.0") {
		t.Errorf("census Service did not join transport with the http-identity Server; body: %s", page)
	}
}

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
