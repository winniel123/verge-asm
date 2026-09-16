package main

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/queue"
)

// A fixture that models an estate no producer can create manufactures a phantom
// bug, and #1985 is the proof of that (ADR-1985 §2).

func fixtureVantageByName(t *testing.T, name string) fixtureVantage {
	t.Helper()
	for _, fv := range inventoryFixtureVantages {
		if fv.name == name {
			return fv
		}
	}
	t.Fatalf("no fixture vantage named %q", name)
	return fixtureVantage{}
}

func fixtureInventoryVantages(t *testing.T) map[int64]inventoryVantage {
	t.Helper()
	covered := fixtureCovered(t)
	out := make(map[int64]inventoryVantage, len(inventoryFixtureVantages))
	for i, fv := range inventoryFixtureVantages {
		out[int64(i+1)] = inventoryVantage{class: deriveFixtureClass(t, fv, covered), name: fv.name}
	}
	return out
}

func fixtureVantageID(t *testing.T, name string) pgtype.Int8 {
	t.Helper()
	for i, fv := range inventoryFixtureVantages {
		if fv.name == name {
			// The seeder's real ids come from the identity column, so only distinctness matters.
			return pgtype.Int8{Int64: int64(i + 1), Valid: true}
		}
	}
	t.Fatalf("no fixture vantage named %q", name)
	return pgtype.Int8{}
}

func TestEveryFixtureSpanNamesASeededVantage(t *testing.T) {
	// 26200 refuses a NULL vantage outside its two exceptions, so a skipped one
	// fails the seeder against the real schema.
	for _, fs := range inventoryFixtureSpans {
		if fs.vantage == "" {
			t.Errorf("fixture span %s/%s/%s carries no vantage", fs.kind, fs.key, fs.facet)
			continue
		}
		fixtureVantageByName(t, fs.vantage)
	}
}

func TestEveryFixtureSpanCarriesTheSourceTheFoldWrites(t *testing.T) {
	for _, fs := range inventoryFixtureSpans {
		want := queue.SourceFor(fs.facet)
		if got := fs.source(); got != want {
			t.Errorf("fixture span %s/%s/%s source = %q, want %q",
				fs.kind, fs.key, fs.facet, got, want)
		}
	}
}

func TestFixtureVantagesPresentOneAddressInScopeAndOneOutside(t *testing.T) {
	covered := fixtureCovered(t)
	internal := fixtureVantageByName(t, fixtureVantageInternal)
	internet := fixtureVantageByName(t, fixtureVantageInternet)

	if got := deriveFixtureClass(t, internal, covered); got != custody.ClassInternal {
		t.Errorf("%s derives %q, want %q", internal.name, got, custody.ClassInternal)
	}
	if got := deriveFixtureClass(t, internet, covered); got != custody.ClassInternet {
		t.Errorf("%s derives %q, want %q", internet.name, got, custody.ClassInternet)
	}
}

func TestSeededAssetPortRendersARealLegPair(t *testing.T) {
	const key = "203.0.113.44:22/tcp"
	// Before #1985 both legs read `never looked` over this measured service.
	addr, _, _ := splitServiceKey(key)

	legs := collapseReachLegs(fixtureReachLegRows(t), fixtureCovered(t))
	ports := buildAssetPorts(fixtureSpanRows(t), legs, map[string]bool{addr: true})

	var port *assetPort
	for i := range ports {
		if ports[i].Port == ":22" {
			port = &ports[i]
		}
	}
	if port == nil {
		t.Fatalf("no :22 row among %d asset ports", len(ports))
	}
	if port.Internal.Label == "never looked" {
		t.Errorf("internal leg reads never looked over a measured service")
	}
	if port.Internet.Label == "never looked" {
		t.Errorf("internet leg reads never looked over a measured service")
	}
	if port.Internal.Label != "reached" {
		t.Errorf("internal leg = %q, want reached", port.Internal.Label)
	}
	if port.Internet.Label != "not reached" {
		t.Errorf("internet leg = %q, want not reached", port.Internet.Label)
	}
}

func fixtureCovered(t *testing.T) func(netip.Addr) bool {
	t.Helper()
	p, err := netip.ParsePrefix(fixtureAddressScope)
	if err != nil {
		t.Fatalf("parse fixture address scope: %v", err)
	}
	return custody.Estate{AddressScopes: []netip.Prefix{p}}.CoversAddressScope
}

func deriveFixtureClass(t *testing.T, fv fixtureVantage, covered func(netip.Addr) bool) custody.VantageClass {
	t.Helper()
	return vantageFactsClass(pgtype.Text{String: fv.dialled, Valid: true}, pgtype.Text{}, covered)
}

func fixtureReachLegRows(t *testing.T) []reachLegRow {
	t.Helper()
	var out []reachLegRow
	for i, fs := range inventoryFixtureSpans {
		if fs.kind != "service" || fs.facet != "reachability" {
			continue
		}
		openedAt, err := fs.openedAt()
		if err != nil {
			t.Fatalf("fixture span %s/%s: %v", fs.kind, fs.key, err)
		}
		// The class read joins vantage, so a leg row carries the dialled address.
		out = append(out, reachLegRow{
			subject:  fs.key,
			dialled:  fixtureVantageByName(t, fs.vantage).dialled,
			value:    []byte(fs.value),
			isGap:    fs.isGap,
			openedAt: openedAt,
			id:       int64(i + 1),
		})
	}
	return out
}

func TestSeededTwoVantageServiceNamesBothClasses(t *testing.T) {
	const key = "203.0.113.44:22/tcp"
	var facets []inventoryFacet
	for _, g := range buildInventory(fixtureSpanRows(t), fixtureInventoryVantages(t)) {
		for _, sub := range g.Subjects {
			if sub.Key == key {
				facets = sub.Facets
			}
		}
	}
	if facets == nil {
		t.Fatalf("no fixture subject %q", key)
	}

	var reach []inventoryFacet
	for _, f := range facets {
		if f.facet == "reachability" {
			reach = append(reach, f)
		}
	}
	if len(reach) != 2 {
		t.Fatalf("reachability rows = %d, want 2", len(reach))
	}
	// Before ADR-2027 this card read `reachability` twice over contradictory summaries.
	if reach[0].Label == reach[1].Label {
		t.Fatalf("both reachability rows read %q", reach[0].Label)
	}
	want := map[string]string{
		"reachability · internal": "reached",
		"reachability · internet": "not-reached",
	}
	for _, f := range reach {
		summary, ok := want[f.Label]
		if !ok {
			t.Errorf("reachability label = %q, want one of internal / internet", f.Label)
			continue
		}
		if f.Summary != summary {
			t.Errorf("%s summary = %q, want %q", f.Label, f.Summary, summary)
		}
	}
}

func TestSeededWithinClassTieNamesBothVantages(t *testing.T) {
	// The corpus holds one vantage per class, so the tie is modelled on top of it.
	vantages := fixtureInventoryVantages(t)
	rival := int64(len(inventoryFixtureVantages) + 1)
	vantages[rival] = inventoryVantage{class: custody.ClassInternet, name: "fixture-internet-2"}

	rows := fixtureSpanRows(t)
	for _, row := range rows {
		if row.SubjectKey == "203.0.113.44:22/tcp" && row.Facet == "reachability" &&
			row.VantageID == fixtureVantageID(t, fixtureVantageInternet) {
			row.VantageID = pgtype.Int8{Int64: rival, Valid: true}
			row.Value = []byte(`{"outcome":"reached"}`)
			rows = append(rows, row)
		}
	}

	var got []string
	for _, g := range buildInventory(rows, vantages) {
		for _, sub := range g.Subjects {
			if sub.Key != "203.0.113.44:22/tcp" {
				continue
			}
			for _, f := range sub.Facets {
				if f.facet == "reachability" {
					got = append(got, f.Label)
				}
			}
		}
	}
	want := []string{
		"reachability · internal",
		"reachability · internet · fixture-internet",
		"reachability · internet · fixture-internet-2",
	}
	if len(got) != len(want) {
		t.Fatalf("reachability labels = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("reachability label[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestFixtureCorpusHoldsNoAddressSubject(t *testing.T) {
	// subjectKindFor returns no address kind, so the fold writes no such span (ADR-2027 §5, #2033).
	for _, fs := range inventoryFixtureSpans {
		if fs.kind == "address" {
			t.Errorf("fixture span %s/%s/%s models an unproducible address subject", fs.kind, fs.key, fs.facet)
		}
	}
}

func TestNoFixtureSpanTeachesACustodyDNSToken(t *testing.T) {
	for _, fs := range inventoryFixtureSpans {
		// ADR-0013 §3 rules custody a seed declaration, never a record (#2145).
		row := strings.ToLower(fs.key + " " + fs.discriminator + " " + fs.value)
		if strings.Contains(row, "verge-custody") || strings.Contains(row, "verge_custody") {
			t.Errorf("fixture span %s/%s/%s teaches a custody DNS token: %s", fs.kind, fs.key, fs.facet, fs.value)
		}
	}
}

func TestEveryFixtureDNSRecordNamesAQtype(t *testing.T) {
	// dns-record is the one facet a producer discriminates, and it carries the qtype.
	offers := map[string]bool{}
	for _, qt := range resolutionwalk.DefaultOffers().Qtypes {
		offers[string(qt)] = true
	}
	for _, fs := range inventoryFixtureSpans {
		if fs.facet != "dns-record" {
			continue
		}
		if !offers[fs.discriminator] {
			t.Errorf("fixture span %s/%s dns-record discriminator = %q, want an offered qtype",
				fs.kind, fs.key, fs.discriminator)
		}
	}
}
