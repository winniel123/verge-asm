package queue

import (
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
)

func TestCitedResolutionsCarryTheTerminalOwnerNotTheQueriedName(t *testing.T) {
	cited := []db.NameCitedAddressesRow{
		{SubjectKey: "shop.example.com", Address: "13.32.1.1", Owner: "d1.cloudfront.net"},
		{SubjectKey: "api.example.com", Address: "52.1.2.3", Owner: "api.example.com"},
		{SubjectKey: "www.example.com", Address: "::ffff:52.1.2.4", Owner: "origin.example.com"},
		{SubjectKey: "bad.example.com", Address: "not-an-address", Owner: "bad.example.com"},
	}
	estate := custody.Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   CitedResolutions(cited),
	}

	if len(estate.Resolutions) != 3 {
		t.Fatalf("resolutions = %+v, want the three parseable rows", estate.Resolutions)
	}
	if got := estate.Resolutions[0].Owner; got != "d1.cloudfront.net" {
		t.Errorf("owner = %q, want the terminal A record's name d1.cloudfront.net", got)
	}
	if got := estate.Resolutions[2].Address; got != netip.MustParseAddr("52.1.2.4") {
		t.Errorf("address = %s, want 52.1.2.4 unmapped", got)
	}

	if got := estate.Derive(netip.MustParseAddr("13.32.1.1")); got != custody.ThirdParty {
		t.Errorf("Derive(13.32.1.1) = %q, want third-party: the CNAME left the zone (ADR-0013 §3)", got)
	}
	if estate.MayProbe(netip.MustParseAddr("13.32.1.1"), custody.ClassInternet) {
		t.Errorf("MayProbe(13.32.1.1) opened the gate on a foreign CNAME target")
	}
	for _, c := range estate.ExtensionCandidates() {
		if c == netip.MustParseAddr("13.32.1.1") {
			t.Errorf("ExtensionCandidates lists 13.32.1.1; a foreign CNAME target is no candidate")
		}
	}
	for _, a := range []string{"52.1.2.3", "52.1.2.4"} {
		if got := estate.Derive(netip.MustParseAddr(a)); got != custody.Operator {
			t.Errorf("Derive(%s) = %q, want operator: the terminal owner sits inside the extended zone", a, got)
		}
	}
}
