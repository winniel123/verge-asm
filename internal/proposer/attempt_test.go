package proposer

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"
)

type stubSource struct {
	slug  string
	cands []Candidate
	err   error
	calls int
}

func (s *stubSource) Slug() string { return s.slug }

func (s *stubSource) Propose(context.Context, string) ([]Candidate, error) {
	s.calls++
	return s.cands, s.err
}

func TestProposeReturnsOneAttemptPerQueriedSource(t *testing.T) {
	ok := &stubSource{slug: "ok-source", cands: []Candidate{{
		SourceSlug: "ok-source", RecordKind: RecordRIRDelegation,
		Scope: netip.MustParsePrefix("203.0.113.0/24"), OrgName: "Org",
	}}}
	bad := &stubSource{slug: "bad-source", err: errors.New("registry unreachable")}
	reg := NewRegistry(ok, bad)

	cands, attempts, err := reg.Propose(context.Background(), "Org", map[string]bool{
		"ok-source": true, "bad-source": true,
	})
	if len(cands) != 1 {
		t.Fatalf("candidates = %+v, want the one the reachable source returned", cands)
	}
	if err == nil || !strings.Contains(err.Error(), "bad-source") {
		t.Fatalf("err = %v, want the failing slug joined in", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts = %+v, want one per queried source", attempts)
	}
	byslug := map[string]error{}
	for _, a := range attempts {
		byslug[a.SourceSlug] = a.Err
	}
	if e, seen := byslug["ok-source"]; !seen || e != nil {
		t.Errorf("ok-source attempt = %v (seen=%v), want a recorded success", e, seen)
	}
	if e, seen := byslug["bad-source"]; !seen || e == nil {
		t.Errorf("bad-source attempt = %v (seen=%v), want the failure attributed to its own slug", e, seen)
	}
}

func TestProposeRecordsNoAttemptForASourceItDidNotQuery(t *testing.T) {
	off := &stubSource{slug: "off-source"}
	reg := NewRegistry(off)

	cands, attempts, err := reg.Propose(context.Background(), "Org", map[string]bool{"off-source": false})
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 0 {
		t.Fatalf("candidates = %+v, want none", cands)
	}
	if off.calls != 0 {
		t.Fatalf("a source the operator did not enable was queried %d times", off.calls)
	}
	// A source nobody enabled reads never attempted, so it must leave no record (ADR-0223 §4).
	if len(attempts) != 0 {
		t.Errorf("attempts = %+v, want none for a source that was never queried", attempts)
	}
}
