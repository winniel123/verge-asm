package resolutionwalk

import (
	"reflect"
	"testing"
)

type mapPeer struct {
	fn func(Query) Msg
}

func (p mapPeer) Exchange(q Query) Msg { return p.fn(q) }

func TestResolveResolvedFoldsAddresses(t *testing.T) {
	// An A and an IPv4-mapped AAAA fold to one Address key (M2.h).
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

	// NXDOMAIN carrying a CNAME does not withdraw: the alias survives as NoData.
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
	// TC=1 on UDP and TCP also truncates -> a Gap, never a partial fold.
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
	// Reached and refused everywhere is Lame; silent everywhere is a Gap (M2.e).
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
