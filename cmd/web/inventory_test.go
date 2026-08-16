package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func openSpanRow(kind, key, facet, disc, value string, isGap bool) db.ListAllOpenSpansRow {
	return db.ListAllOpenSpansRow{
		SubjectKind: kind, SubjectKey: key, Facet: facet, Discriminator: disc,
		Value: []byte(value), IsGap: isGap,
		OpenedAt: pgtype.Timestamptz{Time: obsClock, Valid: true},
	}
}

// buildInventory groups the estate's open spans by subject, preserving the read's
// (kind, key, facet) order, and renders each facet's actual value inline — the
// same value+details the change views summarise, but shown rather than counted.
func TestBuildInventoryGroupsOpenSpansBySubject(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "a.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.1","203.0.113.2"]}`, false),
		openSpanRow("name", "a.example.com", "dns-record", "", `{"rrs":[{"type":"A","data":"203.0.113.1"},{"type":"MX","data":"10 mail.example.com"}]}`, false),
		openSpanRow("name", "b.example.com", "resolution", "", `{}`, true),
		openSpanRow("service", "203.0.113.1:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
		openSpanRow("service", "203.0.113.1:443/tcp", "tls-acceptance", "", `{"outcome":"enumerated","versions":[{"version":"TLS1.3"},{"version":"TLS1.2","ciphers":["ECDHE_RSA_AES_128_GCM"]}]}`, false),
		openSpanRow("endpoint", "@203.0.113.1:443/tcp", "http-identity", "", `{"outcome":"responded","status":200,"server":"nginx"}`, false),
	}

	groups := buildInventory(rows)

	if len(groups) != 3 {
		t.Fatalf("groups = %d, want 3 (name, service, endpoint); %#v", len(groups), groups)
	}
	if groups[0].Kind != "name" || groups[0].Label != "Names" {
		t.Fatalf("first group = %q/%q, want name/Names", groups[0].Kind, groups[0].Label)
	}
	if len(groups[0].Subjects) != 2 {
		t.Fatalf("name subjects = %d, want 2 (a and b)", len(groups[0].Subjects))
	}

	a := groups[0].Subjects[0]
	if a.Key != "a.example.com" || a.Link != "/subjects/a.example.com" {
		t.Fatalf("subject a = %q link %q, want a.example.com and /subjects/a.example.com", a.Key, a.Link)
	}
	if len(a.Facets) != 2 {
		t.Fatalf("subject a facets = %d, want 2 (resolution, dns-record)", len(a.Facets))
	}
	res := a.Facets[0]
	if res.Label != "resolution" || res.Summary != "Resolved" {
		t.Errorf("resolution facet = %q/%q, want resolution/Resolved", res.Label, res.Summary)
	}
	if len(res.Details) != 2 || res.Details[0].Data != "203.0.113.1" || res.Details[1].Data != "203.0.113.2" {
		t.Errorf("resolution details = %#v, want the two addresses", res.Details)
	}
	dns := a.Facets[1]
	if dns.Summary != "2 records" || len(dns.Details) != 2 || dns.Details[0].Type != "A" {
		t.Errorf("dns-record facet = %q details %#v, want a count summary and typed rows", dns.Summary, dns.Details)
	}

	// A Gap is carried as a Gap facet — a value the system currently cannot state —
	// not hidden and not expandable.
	b := groups[0].Subjects[1]
	if b.Key != "b.example.com" || len(b.Facets) != 1 || !b.Facets[0].IsGap {
		t.Fatalf("subject b = %#v, want one Gap facet", b)
	}
	if b.Facets[0].Details != nil {
		t.Errorf("gap facet expanded to %#v, want no details", b.Facets[0].Details)
	}

	// The Service's link goes through subjectHref, so its key's `:` and `/` are
	// query-escaped exactly as elsewhere in the app (#248), and it holds both its
	// reachability verdict and its tls-acceptance value.
	svc := groups[1].Subjects[0]
	if groups[1].Kind != "service" || svc.Link != "/subjects/service?key=203.0.113.1%3A443%2Ftcp" {
		t.Errorf("service group = %q link %q, want service and escaped ?key= link", groups[1].Kind, svc.Link)
	}
	var tls *inventoryFacet
	for i := range svc.Facets {
		if svc.Facets[i].Label == "tls-acceptance" {
			tls = &svc.Facets[i]
		}
	}
	if tls == nil || tls.Summary != "enumerated" {
		t.Fatalf("service tls-acceptance facet = %#v, want an 'enumerated' summary", tls)
	}
	if len(tls.Details) != 2 || tls.Details[0].Type != "TLS1.3" || tls.Details[0].Data != "—" ||
		tls.Details[1].Type != "TLS1.2" || tls.Details[1].Data != "ECDHE_RSA_AES_128_GCM" {
		t.Errorf("tls-acceptance details = %#v, want a version row each with suites (— for 1.3)", tls.Details)
	}

	ep := groups[2].Subjects[0]
	if ep.Link != "/subjects/endpoint?key=%40203.0.113.1%3A443%2Ftcp" {
		t.Errorf("endpoint link = %q, want escaped ?key= link", ep.Link)
	}
	id := ep.Facets[0]
	if id.Summary != "200 · nginx" {
		t.Errorf("http-identity summary = %q, want 200 · nginx", id.Summary)
	}
	// The identity expands to its admitted closed set — the actual observed values.
	var sawServer bool
	for _, d := range id.Details {
		if d.Type == "server" && d.Data == "nginx" {
			sawServer = true
		}
	}
	if !sawServer {
		t.Errorf("http-identity details = %#v, want a server=nginx row", id.Details)
	}
}

// The Inventory page reads the whole estate's open spans once and renders each
// subject's actual current values — the addresses, records, certificate outcome,
// and HTTP identity behind the verdicts — with drill-down links and no denominator.
func TestInventoryPageRendersEstateValues(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	f.addDNSRecord(t, "api.example.com", "", obsClock, `{"rrs":[{"name":"api.example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`)
	f.addReachability(t, "203.0.113.5:443/tcp", obsClock, `{"outcome":"reached"}`)
	f.addHTTPIdentity(t, "api.example.com@203.0.113.5:443/tcp", obsClock, `{"outcome":"responded","status":200,"server":"nginx"}`)
	f.addCertificate(t, "api.example.com@203.0.113.5:443/tcp", obsClock, `{"outcome":"valid","chain":["sha256:leaf","sha256:issuer"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/inventory", http.StatusOK)

	// The actual observed values — not just counts/verdicts — are on the screen.
	for _, want := range []string{
		"203.0.113.5",       // resolved address
		"v=spf1 -all",       // the TXT record contents
		"nginx",             // the HTTP Server header
		"valid",             // the certificate outcome
		"sha256:leaf",       // a certificate chain link
		"Names", "Services", "Endpoints", // grouped by kind
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory missing %q; body: %s", want, page)
		}
	}
	// Drill-down links back to the change views.
	if !strings.Contains(page, `href="/subjects/api.example.com"`) {
		t.Errorf("inventory missing name drill-down link; body: %s", page)
	}
	// No denominator: inventory states no total, exactly as the Subjects listing.
	if !strings.Contains(page, "no total") {
		t.Errorf("inventory does not refuse a denominator in copy; body: %s", page)
	}
}

// Two open spans of the same facet and discriminator on one subject — the same
// Name resolved from two vantages — must render as distinguishable rows, not two
// identically-labelled ones. ADR-0105's unit of currency is
// (facet, discriminator, vantage, source); the label carries the vantage/source
// exactly where a collision would otherwise erase it.
func TestBuildInventoryDisambiguatesCollidingTimelines(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		{
			SubjectKind: "name", SubjectKey: "multi.example.com", Facet: "resolution", Source: "resolver",
			VantageID: pgtype.Int8{Int64: 1, Valid: true},
			Value:     []byte(`{"outcome":"Resolved","addresses":["203.0.113.1"]}`),
			OpenedAt:  pgtype.Timestamptz{Time: obsClock, Valid: true},
		},
		{
			SubjectKind: "name", SubjectKey: "multi.example.com", Facet: "resolution", Source: "resolver",
			VantageID: pgtype.Int8{Int64: 2, Valid: true},
			Value:     []byte(`{"outcome":"NoData"}`),
			OpenedAt:  pgtype.Timestamptz{Time: obsClock, Valid: true},
		},
	}

	facets := buildInventory(rows)[0].Subjects[0].Facets
	if len(facets) != 2 {
		t.Fatalf("want 2 resolution facets, got %d", len(facets))
	}
	if facets[0].Label == facets[1].Label {
		t.Fatalf("colliding timelines share label %q — not disambiguated", facets[0].Label)
	}
	for _, f := range facets {
		if !strings.Contains(f.Label, "vantage ") {
			t.Errorf("label %q missing its vantage disambiguator", f.Label)
		}
	}
}

func TestInventoryRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/inventory")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /inventory: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
