package resolutionwalk

import "testing"

func TestDeclaredRetriesUDPBeforeTCP(t *testing.T) {
	var udp, tcp int
	// measurement-offers.md §5.2: two UDP attempts then one TCP,
	// so one lost datagram is not a batch abort (#1660).
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path != PathDeclared || q.Qtype != QtypeA {
			return Msg{}
		}
		if q.Transport == TCP {
			tcp++
			return Msg{}
		}
		udp++
		if udp == 1 {
			return Msg{Unreachable: true}
		}
		return Msg{Reached: true, Rcode: NOERROR, Answer: []RR{{Name: "example.com", Type: QtypeA, Data: "203.0.113.5"}}}
	}}
	got := Resolve(peer, DefaultOffers(), "example.com")
	if got.Unreachable {
		t.Fatalf("one lost UDP datagram marked the result Unreachable: %+v", got)
	}
	if got.Resolution.Outcome != OutcomeResolved {
		t.Fatalf("outcome = %q, want Resolved from the second UDP attempt", got.Resolution.Outcome)
	}
	if udp != 2 || tcp != 0 {
		t.Fatalf("A query made %d UDP and %d TCP attempts, want 2 UDP and 0 TCP", udp, tcp)
	}
}

func TestDeclaredFallsBackToTCPOnUDPExhaustion(t *testing.T) {
	var udp, tcp int
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path != PathDeclared || q.Qtype != QtypeA {
			return Msg{}
		}
		if q.Transport == TCP {
			tcp++
			return Msg{Reached: true, Rcode: NXDOMAIN}
		}
		udp++
		return Msg{Unreachable: true}
	}}
	got := Resolve(peer, DefaultOffers(), "example.com")
	if got.Unreachable {
		t.Fatalf("UDP exhaustion with a TCP answer marked the result Unreachable: %+v", got)
	}
	if udp != 2 || tcp != 1 {
		t.Fatalf("A query made %d UDP and %d TCP attempts, want 2 UDP then 1 TCP", udp, tcp)
	}
}

func TestDeclaredUnreachableOnlyAfterBudgetExhausted(t *testing.T) {
	var udp, tcp int
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path != PathDeclared || q.Qtype != QtypeA {
			return Msg{}
		}
		if q.Transport == TCP {
			tcp++
		} else {
			udp++
		}
		return Msg{Unreachable: true}
	}}
	got := Resolve(peer, DefaultOffers(), "example.com")
	if !got.Unreachable {
		t.Fatalf("an exhausted budget did not mark the result Unreachable: %+v", got)
	}
	if udp != 2 || tcp != 1 {
		t.Fatalf("A query made %d UDP and %d TCP attempts before Unreachable, want 2 UDP then 1 TCP", udp, tcp)
	}
}

func TestWalkAuthorityRetriesUDPPerNameserver(t *testing.T) {
	var udp int
	// The budget is per nameserver (measurement-offers.md §5.2),
	// so the walk's SOA query retries too.
	peer := mapPeer{fn: func(q Query) Msg {
		switch {
		case q.Path == PathDeclared:
			return Msg{Reached: true, Rcode: NXDOMAIN}
		case q.Path == PathWalk && q.Qtype == QtypeNS:
			return Msg{Reached: true, Rcode: NOERROR, Answer: []RR{{Name: "example.com", Type: QtypeNS, Data: "ns1.example.net"}}}
		case q.Path == PathWalk && q.Qtype == QtypeSOA && q.Transport == UDP:
			udp++
			if udp == 1 {
				return Msg{Unreachable: true}
			}
			return Msg{Reached: true, Rcode: NOERROR}
		default:
			return Msg{}
		}
	}}
	got := Resolve(peer, DefaultOffers(), "example.com")
	if udp != 2 {
		t.Fatalf("SOA query made %d UDP attempts, want 2", udp)
	}
	if got.Delegation.Gap || got.Delegation.Lame || len(got.Delegation.Nameservers) != 1 || !got.Delegation.Nameservers[0].Serves {
		t.Fatalf("a nameserver that answered on the second attempt did not serve: %+v", got.Delegation)
	}
}
