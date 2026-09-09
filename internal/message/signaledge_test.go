package message

import (
	"strings"
	"testing"
)

func TestSignalEdgeFiresDriftAtTheSubjectTheRuleRead(t *testing.T) {
	msg := SignalEdge("endpoint", "admin.example.com@198.51.100.1:443/tcp", "certificate-self-signed", []string{"certificate"}, t0)
	if msg == nil {
		t.Fatal("a not-fired -> fired edge is a message for every rule (ADR-0026 §5)")
	}
	if msg.Cause != CauseDrift || msg.Class != ClassForCause(CauseDrift) {
		t.Errorf("the span the rule reads moved, so the firing is drift class; got cause=%q class=%q", msg.Cause, msg.Class)
	}
	if msg.SubjectKind != "endpoint" || msg.FiredAt != "admin.example.com@198.51.100.1:443/tcp" {
		t.Errorf("fires at the subject the rule read, got %q/%q", msg.SubjectKind, msg.FiredAt)
	}
	if msg.Census != nil {
		t.Errorf("one rule, one edge: nothing is counted, got census %+v", msg.Census)
	}
	for _, want := range []string{"admin.example.com@198.51.100.1:443/tcp", "certificate moved", "certificate-self-signed", "now fired"} {
		if !strings.Contains(msg.Headline, want) {
			t.Errorf("headline %q lacks %q", msg.Headline, want)
		}
	}
	if strings.Contains(msg.Headline, "opened") {
		t.Errorf("an edge is not an opening, so the headline never says opened: %q", msg.Headline)
	}
	if msg.LinkKind() != LinkObject {
		t.Error("the edge links to the subject's own page")
	}
}

func TestSignalEdgeNamesEveryMovedFacet(t *testing.T) {
	msg := SignalEdge("endpoint", "a@198.51.100.1:443/tcp", "plaintext-http-no-https", []string{"certificate", "http-identity"}, t0)
	if !strings.Contains(msg.Headline, "certificate, http-identity moved") {
		t.Errorf("headline names the facets that moved in the fold: %q", msg.Headline)
	}
}

func TestSignalEdgeHeadlineCarriesNoValence(t *testing.T) {
	// Every v1 endpoint and service rule name, so a renamed rule cannot smuggle a valence word in.
	// Two name-rule names carry "resolved", so a name-rule producer must settle that first.
	names := []string{
		"certificate-expired", "certificate-not-yet-valid", "certificate-expiring", "certificate-self-signed",
		"certificate-weak-key-or-signature", "certificate-hostname-san-mismatch", "plaintext-http-no-https",
		"redirect-does-not-upgrade-to-tls", "redirect-to-host-outside-estate", "unauthenticated-request-answered",
		"tls-1.0-accepted", "sensitive-port-reached-from-internet",
	}
	for _, n := range names {
		h := signalEdgeHeadline("admin.example.com@198.51.100.1:443/tcp", n, []string{"certificate", "http-identity"})
		if ContainsValence(h) {
			t.Errorf("headline carries a valence word: %q", h)
		}
	}
}
