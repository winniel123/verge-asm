package main

import (
	"net/netip"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
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
