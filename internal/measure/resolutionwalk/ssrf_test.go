package resolutionwalk

import (
	"context"
	"testing"
)

// gatedWalkPeer scripts the initial NS answer of a delegation walk but routes
// every per-authority SOA sub-query through a real NetPeer, so walk() sees
// exactly what a live Vantage would when an in-scope delegation points its NS
// RDATA at a non-globally-reachable address. The initial NS query carries no
// Server (walk() leaves it to the resolver); the per-authority sub-queries carry
// Server = the NS RDATA, and those are the ones the SSRF gate must catch.
type gatedWalkPeer struct {
	ns   []RR
	real NetPeer
}

func (p gatedWalkPeer) Exchange(q Query) Msg {
	if q.Server == "" {
		return Msg{Reached: true, Answer: p.ns}
	}
	return p.real.Exchange(q)
}

// TestWalkRefusesCloudMetadataAuthority is finding #324: an in-scope delegation
// whose NS RDATA is the cloud-metadata address must not be dialed. The walk
// records the authority as unreached (a Gap, our own blindness), never a value.
func TestWalkRefusesCloudMetadataAuthority(t *testing.T) {
	peer := gatedWalkPeer{
		ns: []RR{{Name: "victim.example.com", Type: QtypeNS, Data: "169.254.169.254"}},
		// A short timeout would still expire if the gate failed and we dialed
		// the (unroutable) metadata address; the gate returns before any dial.
		real: NetPeer{Resolver: "127.0.0.1:53"},
	}

	del := walk(peer, DefaultOffers(), "victim.example.com")

	if !del.Gap {
		t.Errorf("walk did not record a Gap for the barred authority: %+v", del)
	}
	if len(del.Nameservers) != 1 {
		t.Fatalf("want exactly one recorded authority, got %+v", del.Nameservers)
	}
	if got := del.Nameservers[0]; got.Server != "169.254.169.254" || got.Serves {
		t.Errorf("barred authority recorded as %+v, want {169.254.169.254 false} (unreached)", got)
	}
}

// TestWalkServerReachableGate pins the gate decision directly: the barred
// literals refuse the dial, ordinary global unicast is allowed through.
func TestWalkServerReachableGate(t *testing.T) {
	ctx := context.Background()
	barred := []string{
		"169.254.169.254:53", // cloud metadata (link-local)
		"127.0.0.1:53",       // loopback
		"10.0.0.5:53",        // RFC1918
		"192.168.1.1:53",     // RFC1918
		"172.16.0.1:53",      // RFC1918
		"[fd00::1]:53",       // ULA
		"[::1]:53",           // IPv6 loopback
		"[::ffff:10.0.0.1]:53", // IPv4-mapped RFC1918
	}
	for _, s := range barred {
		if walkServerReachable(ctx, s) {
			t.Errorf("walkServerReachable(%q) = true, want false (non-globally-reachable)", s)
		}
	}
	allowed := []string{"8.8.8.8:53", "1.1.1.1:53", "[2001:4860:4860::8888]:53"}
	for _, s := range allowed {
		if !walkServerReachable(ctx, s) {
			t.Errorf("walkServerReachable(%q) = false, want true (globally reachable)", s)
		}
	}
}
