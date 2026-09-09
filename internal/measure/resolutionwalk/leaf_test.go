package resolutionwalk

import (
	"encoding/json"
	"net/netip"
	"reflect"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
)

type mapPeer struct {
	fn func(Query) Msg
}

func (p mapPeer) Exchange(q Query) Msg { return p.fn(q) }

func TestResolveCarriesTheTerminalOwnerOutOfTheWalk(t *testing.T) {
	answer := func(rrs ...RR) Msg { return Msg{Reached: true, Rcode: NOERROR, Answer: rrs} }
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path != PathDeclared || q.Qtype != QtypeA {
			return Msg{}
		}
		switch q.Name {
		case "shop.example.com":
			return answer(
				RR{Name: "shop.example.com", Type: QtypeCNAME, Data: "d1.cloudfront.net"},
				RR{Name: "D1.CloudFront.NET.", Type: QtypeA, Data: "13.32.1.1"},
			)
		case "api.example.com":
			return answer(RR{Name: "api.example.com", Type: QtypeA, Data: "52.1.2.3"})
		case "www.example.com":
			return answer(
				RR{Name: "www.example.com", Type: QtypeCNAME, Data: "origin.example.com"},
				RR{Name: "origin.example.com", Type: QtypeA, Data: "52.1.2.4"},
			)
		}
		return Msg{}
	}}

	cases := []struct{ name, addr, owner string }{
		{"shop.example.com", "13.32.1.1", "d1.cloudfront.net"},
		{"api.example.com", "52.1.2.3", "api.example.com"},
		{"www.example.com", "52.1.2.4", "origin.example.com"},
	}
	for _, c := range cases {
		got := Resolve(peer, DefaultOffers(), c.name)
		if got.Resolution.Outcome != OutcomeResolved {
			t.Fatalf("%s: outcome = %q, want Resolved", c.name, got.Resolution.Outcome)
		}
		if owner := got.Resolution.Owners[c.addr]; owner != c.owner {
			t.Errorf("%s: owner of %s = %q, want %q", c.name, c.addr, owner, c.owner)
		}
		for _, o := range Emit("b1", "v1", got) {
			if o.Facet != FacetResolution {
				continue
			}
			var keys map[string]json.RawMessage
			if err := json.Unmarshal(o.Data, &keys); err != nil {
				t.Fatalf("%s: decode resolution value: %v", c.name, err)
			}
			if _, rendered := keys["owners"]; rendered || len(keys) != 2 {
				t.Errorf("%s: resolution value renders %v; the owner is plumbing and the shape stays {outcome, addresses}", c.name, keys)
			}
		}
	}
}

func TestATerminalOwnerOutsideTheExtendedZoneDerivesThirdParty(t *testing.T) {
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path != PathDeclared || q.Qtype != QtypeA {
			return Msg{}
		}
		switch q.Name {
		case "shop.example.com":
			return Msg{Reached: true, Rcode: NOERROR, Answer: []RR{
				{Name: "shop.example.com", Type: QtypeCNAME, Data: "d1.cloudfront.net"},
				{Name: "d1.cloudfront.net", Type: QtypeA, Data: "13.32.1.1"},
			}}
		case "api.example.com":
			return Msg{Reached: true, Rcode: NOERROR, Answer: []RR{{Name: "api.example.com", Type: QtypeA, Data: "52.1.2.3"}}}
		}
		return Msg{}
	}}

	estate := custody.Estate{ExtendedZones: []string{"example.com"}}
	for _, name := range []string{"shop.example.com", "api.example.com"} {
		res := Resolve(peer, DefaultOffers(), name)
		for _, a := range res.Resolution.Addresses {
			estate.Resolutions = append(estate.Resolutions, custody.Resolution{
				Owner:   res.Resolution.Owners[a],
				Address: netip.MustParseAddr(a),
			})
		}
	}

	if got := estate.Derive(netip.MustParseAddr("13.32.1.1")); got != custody.ThirdParty {
		t.Errorf("Derive(13.32.1.1) = %q, want third-party: the A record sits on the foreign CNAME target", got)
	}
	if got := estate.Derive(netip.MustParseAddr("52.1.2.3")); got != custody.Operator {
		t.Errorf("Derive(52.1.2.3) = %q, want operator: a direct A in the extended zone extends", got)
	}
	for _, c := range estate.ExtensionCandidates() {
		if c == netip.MustParseAddr("13.32.1.1") {
			t.Errorf("ExtensionCandidates lists 13.32.1.1; a foreign CNAME target is no candidate")
		}
	}
}

func TestResolveResolvedFoldsAddresses(t *testing.T) {
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path != PathDeclared {
			return Msg{}
		}
		switch q.Qtype {
		case QtypeA:
			return Msg{Reached: true, Rcode: NOERROR, Answer: []RR{{Name: "example.com", Type: QtypeA, Data: "203.0.113.5"}}}
		case QtypeAAAA:
			return Msg{Reached: true, Rcode: NOERROR, Answer: []RR{{Name: "example.com", Type: QtypeAAAA, Data: "::ffff:203.0.113.5"}}}
		default:
			return Msg{}
		}
	}}
	got := Resolve(peer, DefaultOffers(), "Example.COM")
	if got.Name != "example.com" {
		t.Errorf("name key = %q, want example.com", got.Name)
	}
	if got.Resolution.Outcome != OutcomeResolved {
		t.Fatalf("outcome = %q, want Resolved", got.Resolution.Outcome)
	}
	if !reflect.DeepEqual(got.Resolution.Addresses, []string{"203.0.113.5"}) {
		t.Errorf("addresses = %v, want [203.0.113.5] (folded)", got.Resolution.Addresses)
	}
}

func TestResolveNameErrorOnlyWithoutCNAME(t *testing.T) {
	nx := mapPeer{fn: func(q Query) Msg {
		if q.Path == PathDeclared {
			return Msg{Reached: true, Rcode: NXDOMAIN}
		}
		return Msg{}
	}}
	if got := Resolve(nx, DefaultOffers(), "example.com"); got.Resolution.Outcome != OutcomeNameError {
		t.Errorf("plain NXDOMAIN outcome = %q, want NameError", got.Resolution.Outcome)
	}

	withCNAME := mapPeer{fn: func(q Query) Msg {
		if q.Path == PathDeclared && q.Qtype == QtypeA {
			return Msg{Reached: true, Rcode: NXDOMAIN, Answer: []RR{{Name: "example.com", Type: QtypeCNAME, Data: "t.example.net"}}}
		}
		return Msg{}
	}}
	if got := Resolve(withCNAME, DefaultOffers(), "example.com"); got.Resolution.Outcome != OutcomeNoData {
		t.Errorf("NXDOMAIN+CNAME outcome = %q, want NoData", got.Resolution.Outcome)
	}
}

func TestResolveGapOnUnrecoveredTruncation(t *testing.T) {
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path == PathDeclared && q.Qtype == QtypeA {
			return Msg{Reached: true, Rcode: NOERROR, Truncated: true}
		}
		return Msg{}
	}}
	if got := Resolve(peer, DefaultOffers(), "example.com"); got.Resolution.Outcome != OutcomeGap {
		t.Errorf("outcome = %q, want Gap", got.Resolution.Outcome)
	}
}

func TestWalkLameNeedsReachedRefusalNotSilence(t *testing.T) {
	base := func(soa Msg) Peer {
		return mapPeer{fn: func(q Query) Msg {
			switch {
			case q.Path == PathWalk && q.Qtype == QtypeNS:
				return Msg{Reached: true, Rcode: NOERROR, Answer: []RR{{Name: "example.com", Type: QtypeNS, Data: "ns1.example.net"}}}
			case q.Path == PathWalk && q.Qtype == QtypeSOA:
				return soa
			default:
				return Msg{}
			}
		}}
	}
	if got := Resolve(base(Msg{Reached: true, Rcode: REFUSED}), DefaultOffers(), "example.com"); !got.Delegation.Lame {
		t.Errorf("reached+refused should be Lame, got %+v", got.Delegation)
	}
	if got := Resolve(base(Msg{Reached: false}), DefaultOffers(), "example.com"); got.Delegation.Lame || !got.Delegation.Gap {
		t.Errorf("silent authorities should be a Gap and never Lame, got %+v", got.Delegation)
	}
}

func TestOffersDigestStableAndSensitive(t *testing.T) {
	a := DefaultOffers()
	if a.Digest() != DefaultOffers().Digest() {
		t.Error("digest is not stable across calls")
	}
	b := DefaultOffers()
	b.EDNS.DNSSECOK = true
	if a.Digest() == b.Digest() {
		t.Error("digest did not move when a declared offer changed")
	}
}
