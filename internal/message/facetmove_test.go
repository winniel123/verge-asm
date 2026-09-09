package message

import (
	"strings"
	"testing"
)

func TestFacetMoveFiresDriftAtTheMovedSubject(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: KindRule, Key: "certificate-expired"},
		CensusEntry{Kind: KindRule, Key: "certificate-self-signed"},
	)
	msg := FacetMove("endpoint", "admin.example.com@198.51.100.1:443/tcp", []string{"certificate"}, census, t0)
	if msg == nil {
		t.Fatal("a move that opens a rule at fired is a message (ADR-0033)")
	}
	if msg.Cause != CauseDrift || msg.Class != ClassDrift {
		t.Errorf("the move is drift class, got cause=%q class=%q", msg.Cause, msg.Class)
	}
	if msg.SubjectKind != "endpoint" || msg.FiredAt != "admin.example.com@198.51.100.1:443/tcp" {
		t.Errorf("fires at the moved subject, got %q/%q", msg.SubjectKind, msg.FiredAt)
	}
	if msg.CensusLen() != 2 {
		t.Errorf("carries the census of every rule that opened, got %d", msg.CensusLen())
	}
	for _, want := range []string{"certificate moved", "2 rules opened at fired", "certificate-expired", "certificate-self-signed"} {
		if !strings.Contains(msg.Headline, want) {
			t.Errorf("headline %q lacks %q", msg.Headline, want)
		}
	}
	if msg.LinkKind() != LinkObject {
		t.Error("the move links to the moved subject's page")
	}
}

func TestFacetMoveWithNoRuleEntryIsSilent(t *testing.T) {
	if got := FacetMove("endpoint", "a@198.51.100.1:443/tcp", []string{"certificate"}, NewCensus(CensusEntry{Kind: KindFacet, Key: "certificate"}), t0); got != nil {
		t.Errorf("a move that opens no rule is recorded and never alerted, got %+v", got)
	}
	if got := FacetMove("endpoint", "a@198.51.100.1:443/tcp", []string{"certificate"}, NewCensus(), t0); got != nil {
		t.Errorf("an empty census fires nothing, got %+v", got)
	}
}

func TestFacetMoveHeadlineCarriesNoValence(t *testing.T) {
	// Every v1 endpoint and service rule name, so a renamed rule cannot smuggle a valence word in.
	names := []string{
		"certificate-expired", "certificate-not-yet-valid", "certificate-expiring", "certificate-self-signed",
		"certificate-weak-key-or-signature", "certificate-hostname-san-mismatch", "plaintext-http-no-https",
		"redirect-does-not-upgrade-to-tls", "redirect-to-host-outside-estate", "unauthenticated-request-answered",
		"tls-1.0-accepted", "sensitive-port-reached-from-internet",
	}
	var entries []CensusEntry
	for _, n := range names {
		entries = append(entries, CensusEntry{Kind: KindRule, Key: n})
	}
	h := facetMoveHeadline("admin.example.com@198.51.100.1:443/tcp", []string{"certificate", "http-identity"}, NewCensus(entries...))
	if ContainsValence(h) {
		t.Errorf("headline carries a valence word: %q", h)
	}
}
