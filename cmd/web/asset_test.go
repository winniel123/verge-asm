package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func TestAssetDetailRendersSections(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addDNSRecord(t, "api.example.com", "TXT", obsClock, `{"rrs":[{"name":"api.example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`)
	f.addClassReachability(t, "198.51.100.1:443/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)

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
		"Internal leg",
		"Internet leg",
		"reached",
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

	// The header chip ranks the internet leg alone, so the reached class decides its tone.
	headerFor := func(t *testing.T, class string) string {
		t.Helper()
		f := newFakeStore()
		admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
		addNameSeed(t, f, admin.ID, "example.com")
		f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
		f.addClassReachability(t, "198.51.100.1:443/tcp", class, obsClock, `{"outcome":"reached","result":"open"}`)

		base := start(t, f, "")
		ac := login(t, base, "admin", "hunter2hunter2")
		return assetHeader(getBody(t, ac, base+"/asset/api.example.com", http.StatusOK))
	}

	t.Run("internet leg reached", func(t *testing.T) {
		hdr := headerFor(t, "internet")
		if !strings.Contains(hdr, `class="vg-leg danger">reached`) {
			t.Errorf("header missing the danger internet-leg chip; header: %s", hdr)
		}
		if !strings.Contains(hdr, "Internet leg") {
			t.Errorf("header chip is not named as the internet leg; header: %s", hdr)
		}
	})

	t.Run("internal leg only", func(t *testing.T) {
		hdr := headerFor(t, "internal")
		if strings.Contains(hdr, `class="vg-leg danger"`) {
			t.Errorf("an internal-only reach raised a danger header chip; header: %s", hdr)
		}
		if !strings.Contains(hdr, `class="vg-leg absent">never looked`) {
			t.Errorf("header internet-leg chip does not read never looked; header: %s", hdr)
		}
	})
}

func TestAssetDetailInternalReachIsNeverExposed(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addClassReachability(t, "198.51.100.1:443/tcp", "internal", obsClock, `{"outcome":"reached","result":"open"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	// The shell nav links the Exposure board, so only the page body carries the claim.
	main := page[strings.Index(page, `class="as-main"`):]
	for _, banned := range []string{"exposed", "firewalled", "Exposure"} {
		if strings.Contains(main, banned) {
			t.Errorf("asset body named %q for a service never observed from the internet; body: %s", banned, main)
		}
	}
	if !strings.Contains(main, `class="vg-leg absent">never looked`) {
		t.Errorf("internet leg does not read never looked; body: %s", main)
	}
	if !strings.Contains(main, `class="vg-leg neutral">reached`) {
		t.Errorf("internal leg does not read a neutral reached; body: %s", main)
	}
}

func TestAssetPortsCollapseSpansToOneRowPerPort(t *testing.T) {
	// A span keys on vantage, so two probers on one service open two spans (#1962).
	span := func(at time.Time) db.ListAllOpenSpansRow {
		return db.ListAllOpenSpansRow{
			SubjectKind: "service", SubjectKey: "198.51.100.1:443/tcp", Facet: "reachability",
			OpenedAt: pgtype.Timestamptz{Time: at, Valid: true},
		}
	}
	rows := []db.ListAllOpenSpansRow{
		span(time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)),
		span(time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC)),
	}
	legs := map[string]map[string]legInfo{
		"198.51.100.1:443/tcp": {"internal": {outcome: "reached", present: true}},
	}

	ports := buildAssetPorts(rows, legs, map[string]bool{"198.51.100.1": true})

	if len(ports) != 1 {
		t.Fatalf("ports = %d rows, want 1; got %+v", len(ports), ports)
	}
	if ports[0].Since != "2026-06-14 09:00 UTC" {
		t.Errorf("since = %q, want the earliest open span of the two", ports[0].Since)
	}
	if ports[0].Internal.Label != "reached" || ports[0].Internal.Tone != "neutral" {
		t.Errorf("internal leg = %+v, want a neutral reached", ports[0].Internal)
	}
	if ports[0].Internet.Label != "never looked" {
		t.Errorf("internet leg = %+v, want never looked", ports[0].Internet)
	}
}

func TestAssetPortsSeparateTheTwoAbsences(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{{
		SubjectKind: "service", SubjectKey: "198.51.100.1:443/tcp", Facet: "reachability",
		OpenedAt: pgtype.Timestamptz{Time: time.Date(2026, 6, 14, 9, 0, 0, 0, time.UTC), Valid: true},
	}}
	legs := map[string]map[string]legInfo{
		"198.51.100.1:443/tcp": {"internal": {isGap: true, present: true}},
	}

	ports := buildAssetPorts(rows, legs, map[string]bool{"198.51.100.1": true})

	if len(ports) != 1 {
		t.Fatalf("ports = %d rows, want 1", len(ports))
	}
	// The two absences keep their two statements (ADR-0017 decision 4).
	if ports[0].Internal.Label == ports[0].Internet.Label {
		t.Errorf("a Gap and a never-configured leg both read %q", ports[0].Internal.Label)
	}
	if ports[0].Internal.Label != "stopped looking" || ports[0].Internal.Tone != "warn" {
		t.Errorf("gap leg = %+v, want a warn stopped looking", ports[0].Internal)
	}
}

func TestAssetDetailPortlessAssetRendersNoLegChip(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	if !strings.Contains(page, "No open port measured") {
		t.Errorf("ports card did not fall to its empty state; body: %s", page)
	}
	if strings.Contains(page, `class="vg-leg`) {
		t.Errorf("a portless asset rendered a leg chip; body: %s", page)
	}
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

func TestAssetPortsColumnIsSinceNotFirstSeen(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addClassReachability(t, "198.51.100.1:443/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	head, ok := portsTableHead(page)
	if !ok {
		t.Fatalf("asset detail rendered no Open ports table head; body: %s", page)
	}
	if strings.Contains(head, "First seen") {
		t.Errorf("ports column names a first sighting over an open-span minimum; head: %s", head)
	}
	if !strings.Contains(head, "Since") {
		t.Errorf("ports column = %s, want the Since word /exposure runs the same rule under", head)
	}
}

func portsTableHead(page string) (string, bool) {
	i := strings.Index(page, "Open ports")
	if i < 0 {
		return "", false
	}
	rest := page[i:]
	open := strings.Index(rest, "<thead>")
	if open < 0 {
		return "", false
	}
	// The card heading sits outside the ports guard, so an empty state would hand back the DNS head.
	if empty := strings.Index(rest, "as-empty"); empty >= 0 && empty < open {
		return "", false
	}
	rest = rest[open:]
	j := strings.Index(rest, "</thead>")
	if j < 0 {
		return "", false
	}
	return rest[:j], true
}
