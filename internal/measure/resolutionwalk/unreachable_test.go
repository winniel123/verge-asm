package resolutionwalk

import (
	"bytes"
	"testing"
)

func TestResolveMarksUnreachableOnDeclaredTransportFailure(t *testing.T) {
	// A declared-path socket failure is not a value, so the batch aborts (ADR-0108).
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path == PathDeclared {
			return Msg{Unreachable: true}
		}
		return Msg{}
	}}
	got := Resolve(peer, DefaultOffers(), "example.com")
	if !got.Unreachable {
		t.Fatalf("declared-path transport failure did not mark the result Unreachable: %+v", got)
	}
}

func TestRunWithPeerAbortsOnDeclaredResolverUnreachable(t *testing.T) {
	// The resolver is one position for the batch, so one failure aborts it (ADR-0108).
	peer := mapPeer{fn: func(q Query) Msg {
		return Msg{Unreachable: true}
	}}
	scope := Scope{
		Vantage:  "local",
		Resolver: "127.0.0.11:53",
		Names:    []string{"example.com", "www.example.com"},
		Offers:   DefaultOffers(),
	}
	var buf bytes.Buffer
	err := RunWithPeer(peer, "batch-1", scope, &buf)
	if err == nil {
		t.Fatal("RunWithPeer returned nil for an unreachable resolver, want a batch-aborting error")
	}
	if buf.Len() != 0 {
		t.Fatalf("an aborted batch emitted %d bytes of observations, want none:\n%s", buf.Len(), buf.String())
	}
}

func TestWalkUnreachableDoesNotAbortBatch(t *testing.T) {
	// A walk authority's silence is Gap/Lame vocabulary, never our own position (ADR-0108).
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path == PathDeclared {
			return Msg{Reached: true, Rcode: NXDOMAIN}
		}
		return Msg{Unreachable: true}
	}}
	got := Resolve(peer, DefaultOffers(), "example.com")
	if got.Unreachable {
		t.Fatalf("a walk-path failure wrongly aborted the batch: %+v", got)
	}
	if got.Resolution.Outcome != OutcomeNameError {
		t.Fatalf("declared NXDOMAIN did not decide: outcome = %q, want NameError", got.Resolution.Outcome)
	}

	scope := Scope{Vantage: "local", Resolver: "127.0.0.11:53", Names: []string{"example.com"}, Offers: DefaultOffers()}
	var buf bytes.Buffer
	if err := RunWithPeer(peer, "batch-1", scope, &buf); err != nil {
		t.Fatalf("RunWithPeer aborted on a walk-path failure: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("batch emitted no observations though the declared path decided")
	}
}

func TestReachedAnswersAreNeverUnreachable(t *testing.T) {
	// A reached answer is a value or a per-name Gap, never our blindness (ADR-0108).
	for _, rc := range []Rcode{REFUSED, SERVFAIL, NOERROR, NXDOMAIN} {
		peer := mapPeer{fn: func(q Query) Msg {
			if q.Path == PathDeclared {
				return Msg{Reached: true, Rcode: rc}
			}
			return Msg{}
		}}
		if got := Resolve(peer, DefaultOffers(), "example.com"); got.Unreachable {
			t.Errorf("rcode %s wrongly marked the result Unreachable — a reached answer is not our blindness", rc)
		}
	}
}
