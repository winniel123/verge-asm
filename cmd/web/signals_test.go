package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// seedZone declares a name-scope Seed and attaches a zone file to it, returning
// nothing — the Signals reads pick it up through ListZoneDeclarations.
func seedZone(t *testing.T, f *fakeStore, admin db.Account, domain, content string) {
	t.Helper()
	s, err := f.CreateNameSeed(t.Context(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.CreateZoneFile(t.Context(), db.CreateZoneFileParams{
		SeedID:     s.ID,
		SuppliedAt: pgtype.Timestamptz{Time: obsClock, Valid: true},
		Content:    content,
		UploadedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSignalsRendersEveryRuleCensus(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	zone := "$ORIGIN example.com.\n" +
		"@ IN SOA ns1 admin 1 2 3 4 5\n" +
		"www IN A 203.0.113.20\n" +
		"api IN A 203.0.113.21\n" +
		"missing IN A 203.0.113.22\n"
	seedZone(t, f, admin, "example.com", zone)

	// lame-delegation: an all-refusing delegation → composed Lame → FIRED.
	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)

	// A plain resolved name → lame-delegation NOT-FIRED, non-global NOT-FIRED.
	f.addClassResolution(t, "good.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.10"]}`)

	// A Shadowed name → lame-delegation NOT-EVALUABLE (distinct from not-fired).
	f.addClassResolution(t, "shadow.example.com", "internet", obsClock, `{"outcome":"Shadowed"}`)

	// An internal address in a public answer → non-global FIRED.
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)

	// A dangling CNAME → cname-target-name-error FIRED.
	f.addClassResolution(t, "alias.example.com", "internet", obsClock, `{"outcome":"NoData"}`)
	f.addDNSRecord(t, "alias.example.com", "CNAME", obsClock, `{"rrs":[{"name":"alias.example.com","type":"CNAME","data":"gone.example.com"}]}`)
	f.addClassResolution(t, "gone.example.com", "internet", obsClock, `{"outcome":"NameError"}`)

	// A declared name our resolver NXDOMAINs → zone-declared-…-name-error FIRED
	// even though it has withdrawn (evidence current, membership irrelevant).
	f.addClassResolution(t, "missing.example.com", "internet", obsClock, `{"outcome":"NameError"}`)

	// A resolving name inside the zone but not declared → absent-from-zone FIRED.
	f.addClassResolution(t, "orphan.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.30"]}`)
	// A declared name that resolves → both zone rules NOT-FIRED.
	f.addClassResolution(t, "www.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.20"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/signals", http.StatusOK)

	// Every rule's census renders.
	for _, rule := range []string{
		"lame-delegation",
		"cname-target-name-error",
		"zone-declared-name-returns-name-error",
		"resolved-name-absent-from-zone",
		"non-globally-reachable-address-resolved-from-internet",
	} {
		if !strings.Contains(page, rule) {
			t.Errorf("Signals page missing rule %q", rule)
		}
	}

	// The version vector composes both leaves it reads.
	if !strings.Contains(page, "resolution-walk/v1") || !strings.Contains(page, "wildcard-discrimination/v1") {
		t.Errorf("version vector not rendered composing both leaves; body: %s", page)
	}

	// Fired members appear and drill to their subject.
	for _, name := range []string{"lame.example.com", "leak.example.com", "orphan.example.com", "missing.example.com", "alias.example.com"} {
		if !strings.Contains(page, `href="/subjects/`+name+`"`) {
			t.Errorf("census member %q not drillable to its subject; body: %s", name, page)
		}
	}

	// The three registers are distinct, labelled members — not-evaluable is not
	// folded into did-not-fire.
	for _, label := range []string{"Fired", "Did not fire", "Not-evaluable"} {
		if !strings.Contains(page, label) {
			t.Errorf("census missing member label %q", label)
		}
	}
	if !strings.Contains(page, "shadow.example.com") {
		t.Errorf("Shadowed name not rendered as a not-evaluable member; body: %s", page)
	}

	// The member header count is locked to list.length — a count element renders.
	if !strings.Contains(page, `class="count"`) {
		t.Errorf("member group renders no locked count; body: %s", page)
	}

	// The census member component is NOT the Subjects row: no search box, no
	// Citation. And a Signal has no severity, and 'finding' is a rejected word.
	if strings.Contains(page, `name="q"`) {
		t.Errorf("Signals page carries a search box; a member list must not")
	}
	if strings.Contains(page, "Citation") {
		t.Errorf("census member carries a Citation; it must not (ADR-0102)")
	}
	low := strings.ToLower(page)
	if strings.Contains(low, "finding") {
		t.Errorf("Signals page uses the rejected word 'finding'")
	}
	if strings.Contains(low, "severity") {
		t.Errorf("Signals page mentions severity; a Signal carries none")
	}
}

func TestSignalsEmptyEstateRendersNoPopulation(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/signals", http.StatusOK)
	// With no subjects, every rule's domain is empty: a no-population panel, never
	// a census of zeroes.
	if !strings.Contains(page, "No population") {
		t.Errorf("empty estate did not render a no-population panel; body: %s", page)
	}
	// Rules still render (the page is the rule set, current state).
	if !strings.Contains(page, "lame-delegation") {
		t.Errorf("rules not rendered on an empty estate; body: %s", page)
	}
}

func TestSignalsRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/signals")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /signals: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
