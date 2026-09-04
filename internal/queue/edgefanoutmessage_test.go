package queue

import (
	"net/netip"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

func declineEstate(edge netip.Addr) custody.Estate {
	return custody.Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []custody.Resolution{{Owner: "shop.example.com", Address: edge}},
	}.WithEdgeFanout(custody.EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{edge: true}})
}

func TestDeclineFiresNoMessage(t *testing.T) {
	// A veto withholds a probe, so it is told by display and never by a message (ADR-0129 #944).
	edge := netip.MustParseAddr("93.184.216.10")
	if got := declineEstate(edge).Derive(edge); got != custody.ThirdParty {
		t.Fatalf("Derive on a declined edge = %q, want %q — the veto did not hold", got, custody.ThirdParty)
	}
	// The last link alone would pass over an empty change set for any reason at all.
	msgs := membershipMessages(time.Unix(0, 0).UTC(), nil, membershipInputs{})
	if len(msgs) != 0 {
		t.Errorf("messages = %+v on a batch that probed a declined edge, want none", msgs)
	}
}

func TestRepointOntoSharedEdgeFiresNoMessage(t *testing.T) {
	closed := []spanChange{{
		SubjectKind: "address",
		SubjectKey:  "203.0.113.77",
		Facet:       resolutionwalk.FacetResolution,
		Opened:      false,
	}}
	if msgs := membershipMessages(time.Unix(0, 0).UTC(), closed, membershipInputs{}); len(msgs) != 0 {
		t.Errorf("messages = %+v on a re-point off a dedicated origin, want none", msgs)
	}
}
