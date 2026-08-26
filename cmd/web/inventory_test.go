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
	// A Name row opens the Asset detail (T1's /asset/{key}), not the subject
	// drill-down — the row-click destination T15 wires (#310).
	if a.Key != "a.example.com" || a.Link != "/asset/a.example.com" {
		t.Fatalf("subject a = %q link %q, want a.example.com and /asset/a.example.com", a.Key, a.Link)
	}
	if len(a.Facets) != 2 {
		t.Fatalf("subject a facets = %d, want 2 (resolution, dns-record)", len(a.Facets))
	}
	// The inventory resolution summary is `rrtype · <n> addresses` (the pilot shows
	// the shaped value, not the outcome tag), and expands one typed row per address.
	res := a.Facets[0]
	if res.Label != "resolution" || res.Summary != "A · 2 addresses" {
		t.Errorf("resolution facet = %q/%q, want resolution/\"A · 2 addresses\"", res.Label, res.Summary)
	}
	if len(res.Details) != 2 || res.Details[0].Type != "A" || res.Details[0].Data != "203.0.113.1" || res.Details[1].Data != "203.0.113.2" {
		t.Errorf("resolution details = %#v, want the two typed addresses", res.Details)
	}
	// dns-record summarises the ordered-distinct RR types, and the facet label is the
	// plural "dns-records".
	dns := a.Facets[1]
	if dns.Label != "dns-records" || dns.Summary != "A · MX" || len(dns.Details) != 2 || dns.Details[0].Type != "A" {
		t.Errorf("dns-record facet = %q/%q details %#v, want the distinct-types summary and typed rows", dns.Label, dns.Summary, dns.Details)
	}

	// A Gap is carried as a Gap facet — a value the system currently cannot state —
	// not hidden and not expandable; its summary is empty (the template renders the
	// Gap marker off IsGap).
	b := groups[0].Subjects[1]
	if b.Key != "b.example.com" || len(b.Facets) != 1 || !b.Facets[0].IsGap {
		t.Fatalf("subject b = %#v, want one Gap facet", b)
	}
	if b.Facets[0].Summary != "" || b.Facets[0].Details != nil {
		t.Errorf("gap facet = %q / %#v, want an empty summary and no details", b.Facets[0].Summary, b.Facets[0].Details)
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
	// tls-acceptance renders `TLS <versions>` in the inventory pilot and does not
	// expand — its whole value is the summary line.
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
	// http-identity renders `server · status[ · “title”][ → redirect]` and does not
	// expand in the inventory pilot.
	id := ep.Facets[0]
	if id.Summary != "nginx · 200" {
		t.Errorf("http-identity summary = %q, want \"nginx · 200\"", id.Summary)
	}
	if len(id.Details) != 0 {
		t.Errorf("http-identity details = %#v, want none in the inventory pilot", id.Details)
	}
}

// The Inventory page reads the whole estate's open spans once and renders each
// subject's actual current values — the addresses, records, certificate outcome,
// and HTTP identity behind the verdicts — with drill-down links and no denominator.
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

	// The actual observed values — not just counts/verdicts — are on the screen, in
	// the Inventory pilot's shaped vocabulary.
	for _, want := range []string{
		"203.0.113.5",       // resolved address (in the resolution summary)
		"v=spf1 -all",       // the TXT record contents (a dns-record detail)
		"nginx",             // the HTTP Server header
		"leaf api.example.com", // the certificate-chain summary's leaf CN
		"not_after 2026-11-02", // a certificate-chain detail row
		"Names", "Services", "Endpoints", // grouped by kind
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory missing %q; body: %s", want, page)
		}
	}
	// Row → Asset detail (#310, T15): each Name row opens T1's /asset/{key}, wired
	// both as the anchor href and as the whole-row navigable data-href, and the row
	// carries the roving-focus affordances (a focusable, link-role row). The frozen
	// design tmpl marks the navigable row with role="link" + aria-label rather than a
	// class, and the j/k keyboard nav lives in the tmpl's script.
	for _, want := range []string{
		`href="/asset/api.example.com"`,     // the Subject-cell anchor
		`data-href="/asset/api.example.com"`, // whole-row click / Enter destination
		`role="link"`,                       // navigable row (design tmpl markup)
		`aria-label="Open api.example.com"`, // the row's accessible open affordance
		`tabindex="0"`,                      // roving keyboard focus
		`e.key === "j"`,                     // j/k keyboard nav, present in the tmpl's script
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory missing row→asset / keyboard affordance %q; body: %s", want, page)
		}
	}
	// No denominator: inventory states no total, exactly as the Subjects listing.
	if !strings.Contains(page, "no total") {
		t.Errorf("inventory does not refuse a denominator in copy; body: %s", page)
	}

	// The Inventory nav pill is the active one (keyed on NavActive, not Active).
	if !strings.Contains(page, `class="sh-pill on" href="/inventory"`) {
		t.Errorf("inventory nav pill not marked active; body: %s", page)
	}

	// Structural checklist vs 03-console.jpg: the Kind cluster, density control, column
	// picker, a per-row Type in domain vocabulary, and the facet label + value in the
	// Holds cell. (The old title-attr hover peek is not in the frozen design tmpl.)
	for _, want := range []string{
		"All subjects",                      // Kind segmented control
		`aria-label="Row density"`,          // density control
		"Columns",                           // column picker
		`<span class="inv-tag">Name`,        // Type cell — the domain noun, not a wire tag
		`class="inv-facetlabel">resolution`, // the facet label rendered in the Holds cell
		"A · 203.0.113.5",                   // its current value, in the pilot's shaped vocabulary
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory structural element missing %q; body: %s", want, page)
		}
	}

	// Vocabulary guardrail: the table speaks domain nouns, never wire nouns as
	// column headers (no "IP" / "Open ports" / "URL" as modelled things).
	for _, banned := range []string{">IP<", ">Open ports<", ">URL<"} {
		if strings.Contains(page, banned) {
			t.Errorf("inventory rendered a wire-noun column %q; body: %s", banned, page)
		}
	}
}

// The empty estate keeps a legible "No population" state rather than a blank
// screen, and still refuses a denominator in its intro copy.
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

// Two open reachability timelines on one Address — the same facet reached from two
// vantages — must render as distinguishable rows. In the Inventory pilot the span's
// discriminator carries the vantage qualifier ("vantage 1", "vantage 3"), so the
// facet label ("reachability · vantage 1") is unique by construction and the pilot
// runs no source/vantage disambiguation pass over it.
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

// GET /inventory/export streams a text/csv attachment of the folded inventory: a
// header row plus one row per facet a subject currently holds, mirroring the screen.
// A Gap facet exports as the literal "Gap", never a blank standing in for a real read.
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
		"type,subject,facet,value,since", // header row
		"Name,api.example.com,resolution,A · 203.0.113.5,",
		"Name,gap.example.com,resolution,Gap,", // a Gap exports as "Gap", never blank
	} {
		if !strings.Contains(got, want) {
			t.Errorf("inventory export CSV missing %q; body:\n%s", want, got)
		}
	}
}

// The Export CSV button is gated on data presence exactly as Drift's is: a live link
// to the export once a value is folded, and the disabled button on an empty estate.
func TestInventoryExportButtonGated(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Empty estate: the button is disabled, no export link.
	empty := getBody(t, ac, base+"/inventory", http.StatusOK)
	if strings.Contains(empty, `href="/inventory/export"`) {
		t.Errorf("empty inventory should not offer an export link; body: %s", empty)
	}
	if !strings.Contains(empty, "Nothing to export until a value is folded") {
		t.Errorf("empty inventory lost its disabled-export button; body: %s", empty)
	}

	// With a folded value the button becomes a live link.
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	full := getBody(t, ac, base+"/inventory", http.StatusOK)
	if !strings.Contains(full, `href="/inventory/export"`) {
		t.Errorf("inventory with data did not enable the export link; body: %s", full)
	}
}

// The re-specced Inventory (SPEC-CHANGE #13 / U6, package v3.2.4) carries the
// client-side scope controls that replaced the exposure-era saved views / tag
// filters / bulk-actions bar — a Kind segmented control, a Gaps-only switch, a
// subject filter, and a "no subjects match this scope" state with a Clear-filters
// reset. All scope the already-rendered corpus; nothing hits the server.
func TestInventoryScopeControls(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/inventory", http.StatusOK)

	for _, want := range []string{
		`id="inv-kind"`,                // Kind segmented control (design tmpl markup)
		`data-kind="name"`,             // one Kind option (and section) per rendered group
		`id="inv-gaps"`,                // Gaps-only switch
		"Gaps only",
		`id="inv-q"`,                   // subject filter
		"No subjects match this scope", // no-match empty state
		`id="inv-clear"`,               // its Clear-filters reset
		`data-key="api.example.com"`,   // each row is scopable by its subject key
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inventory scope control missing %q; body: %s", want, page)
		}
	}
	// The retired exposure-era controls (bound to nothing in the subject/facet/span
	// model, SPEC-CHANGE #13) must not have crept back.
	for _, banned := range []string{"Saved view", "Rescan", "Add tag", "Bulk action"} {
		if strings.Contains(page, banned) {
			t.Errorf("inventory rendered a retired exposure-era control %q; body: %s", banned, page)
		}
	}
}

// HasGap reports whether a subject holds at least one Gap facet — the per-row
// datum the "Gaps only" client-side scope reads.
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
