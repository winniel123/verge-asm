package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
)

func openSpanRow(kind, key, facet, disc, value string, isGap bool) db.ListAllOpenSpansRow {
	return db.ListAllOpenSpansRow{
		SubjectKind: kind, SubjectKey: key, Facet: facet, Discriminator: disc,
		Value: []byte(value), IsGap: isGap,
		OpenedAt: pgtype.Timestamptz{Time: obsClock, Valid: true},
	}
}

func TestBuildInventoryGroupsOpenSpansBySubject(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "a.example.com", "resolution", "", `{"rrtype":"A","addresses":["203.0.113.1","203.0.113.2"]}`, false),
		openSpanRow("name", "a.example.com", "dns-record", "", `{"rrs":[{"type":"A","data":"203.0.113.1"},{"type":"MX","data":"10 mail.example.com"}]}`, false),
		openSpanRow("name", "b.example.com", "resolution", "", `{}`, true),
		openSpanRow("service", "203.0.113.1:443/tcp", "reachability", "", `{"outcome":"answers","ports":["443/tcp"]}`, false),
		openSpanRow("service", "203.0.113.1:443/tcp", "tls-acceptance", "", `{"outcome":"enumerated","versions":["1.2","1.3"]}`, false),
		openSpanRow("endpoint", "@203.0.113.1:443/tcp", "http-identity", "", `{"server":"nginx","status":200}`, false),
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
	if a.Key != "a.example.com" || a.Link != "/asset/a.example.com" {
		t.Fatalf("subject a = %q link %q, want a.example.com and /asset/a.example.com", a.Key, a.Link)
	}
	if len(a.Facets) != 2 {
		t.Fatalf("subject a facets = %d, want 2 (resolution, dns-record)", len(a.Facets))
	}
	res := a.Facets[0]
	if res.Label != "resolution" || res.Summary != "A · 2 addresses" {
		t.Errorf("resolution facet = %q/%q, want resolution/\"A · 2 addresses\"", res.Label, res.Summary)
	}
	if len(res.Details) != 2 || res.Details[0].Type != "A" || res.Details[0].Data != "203.0.113.1" || res.Details[1].Data != "203.0.113.2" {
		t.Errorf("resolution details = %#v, want the two typed addresses", res.Details)
	}
	dns := a.Facets[1]
	if dns.Label != "dns-records" || dns.Summary != "A · MX" || len(dns.Details) != 2 || dns.Details[0].Type != "A" {
		t.Errorf("dns-record facet = %q/%q details %#v, want the distinct-types summary and typed rows", dns.Label, dns.Summary, dns.Details)
	}

	b := groups[0].Subjects[1]
	if b.Key != "b.example.com" || len(b.Facets) != 1 || !b.Facets[0].IsGap {
		t.Fatalf("subject b = %#v, want one Gap facet", b)
	}
	if b.Facets[0].Summary != "" || b.Facets[0].Details != nil {
		t.Errorf("gap facet = %q / %#v, want an empty summary and no details", b.Facets[0].Summary, b.Facets[0].Details)
	}

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
	if tls == nil || tls.Summary != "TLS 1.2 · 1.3" {
		t.Fatalf("service tls-acceptance facet = %#v, want a \"TLS 1.2 · 1.3\" summary", tls)
	}
	if len(tls.Details) != 0 {
		t.Errorf("tls-acceptance details = %#v, want none in the inventory pilot", tls.Details)
	}

	ep := groups[2].Subjects[0]
	if ep.Link != "/subjects/endpoint?key=%40203.0.113.1%3A443%2Ftcp" {
		t.Errorf("endpoint link = %q, want escaped ?key= link", ep.Link)
	}
	id := ep.Facets[0]
	if id.Summary != "nginx · 200" {
		t.Errorf("http-identity summary = %q, want \"nginx · 200\"", id.Summary)
	}
	if len(id.Details) != 0 {
		t.Errorf("http-identity details = %#v, want none in the inventory pilot", id.Details)
	}
}

func TestInventoryPageRendersEstateValues(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"rrtype":"A","addresses":["203.0.113.5"]}`)
	f.addDNSRecord(t, "api.example.com", "", obsClock, `{"rrs":[{"name":"api.example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`)
	f.addReachability(t, "203.0.113.5:443/tcp", obsClock, `{"outcome":"answers","ports":["443/tcp"]}`)
	f.addHTTPIdentity(t, "api.example.com@203.0.113.5:443/tcp", obsClock, `{"server":"nginx","status":200}`)
	f.addCertificate(t, "api.example.com@203.0.113.5:443/tcp", obsClock, `{"chain":[{"cn":"api.example.com","not_after":"2026-11-02"},{"cn":"R11","issuer_org":"Let’s Encrypt"}]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/inventory", http.StatusOK)

	for _, want := range []string{
		"203.0.113.5",
		"v=spf1 -all",
		"nginx",
		"leaf api.example.com",
		"not_after 2026-11-02",
		"Names", "Services", "Endpoints",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory missing %q; body: %s", want, page)
		}
	}
	for _, want := range []string{
		`href="/asset/api.example.com"`,
		`data-href="/asset/api.example.com"`,
		`role="link"`,
		`aria-label="Open api.example.com"`,
		`tabindex="0"`,
		`e.key === "j"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory missing row→asset / keyboard affordance %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, "no total") {
		t.Errorf("inventory does not refuse a denominator in copy; body: %s", page)
	}

	if !strings.Contains(page, `class="sh-pill on" href="/inventory"`) {
		t.Errorf("inventory nav pill not marked active; body: %s", page)
	}

	for _, want := range []string{
		"All subjects",
		`aria-label="Row density"`,
		"Columns",
		`<span class="inv-tag">Name`,
		`class="inv-facetlabel">resolution`,
		"A · 203.0.113.5",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory structural element missing %q; body: %s", want, page)
		}
	}

	for _, banned := range []string{">IP<", ">Open ports<", ">URL<"} {
		if strings.Contains(page, banned) {
			t.Errorf("inventory rendered a wire-noun column %q; body: %s", banned, page)
		}
	}
}

func TestInventoryEmptyStatePreserved(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/inventory", http.StatusOK)
	if !strings.Contains(page, "No population") {
		t.Errorf("empty inventory lost its No-population state; body: %s", page)
	}
	if !strings.Contains(page, "no total") {
		t.Errorf("empty inventory does not refuse a denominator; body: %s", page)
	}
}

func TestBuildInventoryDistinguishesTimelinesByDiscriminator(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("address", "198.51.100.7", "reachability", "vantage 1", `{"outcome":"answers","ports":["443/tcp"]}`, false),
		openSpanRow("address", "198.51.100.7", "reachability", "vantage 3", `{"outcome":"answers","ports":["443/tcp","8443/tcp"]}`, false),
	}

	facets := buildInventory(rows)[0].Subjects[0].Facets
	if len(facets) != 2 {
		t.Fatalf("want 2 reachability facets, got %d", len(facets))
	}
	if facets[0].Label == facets[1].Label {
		t.Fatalf("timelines share label %q — the discriminator did not distinguish them", facets[0].Label)
	}
	if facets[0].Label != "reachability · vantage 1" || facets[1].Label != "reachability · vantage 3" {
		t.Errorf("labels = %q / %q, want \"reachability · vantage 1\" / \"reachability · vantage 3\"", facets[0].Label, facets[1].Label)
	}
}

func TestInventoryExportCSV(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"rrtype":"A","addresses":["203.0.113.5"]}`)
	f.addResolution(t, admin.ID, "gap.example.com", "dns", obsClock, `{"outcome":"Gap"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/inventory/export")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("inventory export status = %d, want 200 (body: %s)", resp.StatusCode, got)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("inventory export Content-Type = %q, want text/csv", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=") || !strings.Contains(cd, ".csv") {
		t.Errorf("inventory export Content-Disposition = %q, want an attachment .csv filename", cd)
	}
	for _, want := range []string{
		"type,subject,facet,value,since",
		"Name,api.example.com,resolution,A · 203.0.113.5,",
		"Name,gap.example.com,resolution,Gap,",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("inventory export CSV missing %q; body:\n%s", want, got)
		}
	}
}

func TestInventoryExportButtonGated(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	empty := getBody(t, ac, base+"/inventory", http.StatusOK)
	if strings.Contains(empty, `href="/inventory/export"`) {
		t.Errorf("empty inventory should not offer an export link; body: %s", empty)
	}
	if !strings.Contains(empty, "Nothing to export until a value is folded") {
		t.Errorf("empty inventory lost its disabled-export button; body: %s", empty)
	}

	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	full := getBody(t, ac, base+"/inventory", http.StatusOK)
	if !strings.Contains(full, `href="/inventory/export"`) {
		t.Errorf("inventory with data did not enable the export link; body: %s", full)
	}
}

func TestInventoryScopeControls(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/inventory", http.StatusOK)

	// The scope controls filter the already-rendered corpus, so no control reaches the server.
	for _, want := range []string{
		`id="inv-kind"`,
		`data-kind="name"`,
		`id="inv-gaps"`,
		"Gaps only",
		`id="inv-q"`,
		"No subjects match this scope",
		`id="inv-clear"`,
		`data-key="api.example.com"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory scope control missing %q; body: %s", want, page)
		}
	}
	for _, banned := range []string{"Saved view", "Rescan", "Add tag", "Bulk action"} {
		if strings.Contains(page, banned) {
			t.Errorf("inventory rendered a retired exposure-era control %q; body: %s", banned, page)
		}
	}
}

func TestInventorySubjectHasGap(t *testing.T) {
	withGap := inventorySubject{Facets: []inventoryFacet{{Label: "resolution"}, {Label: "dns-record", IsGap: true}}}
	if !withGap.HasGap() {
		t.Errorf("HasGap() = false for a subject holding a Gap facet")
	}
	noGap := inventorySubject{Facets: []inventoryFacet{{Label: "resolution"}}}
	if noGap.HasGap() {
		t.Errorf("HasGap() = true for a subject holding no Gap facet")
	}
}

func TestInventoryProxyEdgeBadgesBlanketAndIncomplete(t *testing.T) {
	reachGap := func(cause, reason string) []byte {
		return []byte(fmt.Sprintf(`{"outcome":"gap","cause":%q,"reason":%q}`, cause, reason))
	}
	cases := []struct {
		name  string
		facet string
		value []byte
		isGap bool
		want  bool
	}{
		{"blanket reason", "reachability", reachGap(blanketdiscrim.GapCause, blanketdiscrim.ReasonBlanket), true, true},
		{"incomplete reason", "reachability", reachGap(blanketdiscrim.GapCause, blanketdiscrim.ReasonIncomplete), true, true},
		{"other cause", "reachability", reachGap("some-other-cause", "unrelated"), true, false},
		{"not a gap", "reachability", []byte(`{"outcome":"answers","ports":["443/tcp"]}`), false, false},
		{"non-reachability gap", "certificate", reachGap(blanketdiscrim.GapCause, blanketdiscrim.ReasonBlanket), true, false},
		{"unparseable value", "reachability", []byte(`{`), true, false},
	}
	for _, c := range cases {
		if got := inventoryProxyEdge(c.facet, c.value, c.isGap); got != c.want {
			t.Errorf("%s: inventoryProxyEdge = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestBuildInventoryPropagatesServiceProxyEdgeToAddress(t *testing.T) {
	gap := fmt.Sprintf(`{"outcome":"gap","cause":%q,"reason":%q}`, blanketdiscrim.GapCause, blanketdiscrim.ReasonIncomplete)
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("service", "104.21.61.6:443/tcp", "reachability", "", gap, true),
		openSpanRow("address", "104.21.61.6", "reachability", "vantage 1", `{"outcome":"answers","ports":["443/tcp"]}`, false),
		openSpanRow("service", "[2606:4700::1]:443/tcp", "reachability", "", gap, true),
		openSpanRow("address", "2606:4700::1", "reachability", "vantage 1", `{"outcome":"answers","ports":["443/tcp"]}`, false),
		openSpanRow("address", "198.51.100.7", "reachability", "vantage 1", `{"outcome":"answers","ports":["443/tcp"]}`, false),
	}

	groups := buildInventory(rows)
	byKind := map[string][]inventorySubject{}
	for _, g := range groups {
		byKind[g.Kind] = g.Subjects
	}

	for _, sub := range byKind["service"] {
		if !sub.ProxyEdge {
			t.Errorf("service %q ProxyEdge = false, want true", sub.Key)
		}
	}
	proxyAddr := map[string]bool{}
	for _, sub := range byKind["address"] {
		proxyAddr[sub.Key] = sub.ProxyEdge
	}
	for _, want := range []string{"104.21.61.6", "2606:4700::1"} {
		if !proxyAddr[want] {
			t.Errorf("address %q ProxyEdge = false, want true (propagated from its Service)", want)
		}
	}
	if proxyAddr["198.51.100.7"] {
		t.Errorf("address 198.51.100.7 ProxyEdge = true, want false (no proxy-edge Service)")
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
