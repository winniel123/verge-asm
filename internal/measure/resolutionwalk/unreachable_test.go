package resolutionwalk

import (
	"bytes"
	"testing"
)

// A declared-path socket failure is "we could not look", not a value: it sets
// the batch-aborting Unreachable signal, so the whole batch fails and covers
// nothing rather than committing a silent all-Gap measurement (ADR-0108).
func TestResolveMarksUnreachableOnDeclaredTransportFailure(t *testing.T) {
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

// The resolver is one position for the whole batch, so the first declared-path
// transport failure aborts the batch: RunWithPeer returns an error and emits no
// observations at all — the empty-scope dead-letter path takes over (ADR-0108).
func TestRunWithPeerAbortsOnDeclaredResolverUnreachable(t *testing.T) {
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

// The delegation walk dials delegated authorities direct; their silence is the
// Gap/Lame vocabulary, never a statement about our own position. So a walk-path
// transport failure must NOT abort the batch — the declared path decided, and
// the batch commits its values normally (ADR-0108 limb 2).
func TestWalkUnreachableDoesNotAbortBatch(t *testing.T) {
	peer := mapPeer{fn: func(q Query) Msg {
		if q.Path == PathDeclared {
			return Msg{Reached: true, Rcode: NXDOMAIN}
		}
		return Msg{Unreachable: true} // the walk cannot reach any authority
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

// A resolver that was reached and spoke — REFUSED, SERVFAIL, an empty NOERROR,
// NXDOMAIN — is a value or a per-name Gap, never our blindness. None of these
// set Unreachable: the batch commits, it does not fail (ADR-0108 limb 1).
func TestReachedAnswersAreNeverUnreachable(t *testing.T) {
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
