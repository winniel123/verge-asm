package queue

import (
	"context"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/message"
)

const (
	apexName   = "example.com"
	apexAddr   = "203.0.113.7"
	apexSvc    = apexAddr + ":443/tcp"
	subName    = "www." + apexName
	lookalike  = "notexample.com"
	uncitedSvc = "198.51.100.9:443/tcp"
)

func apexFold() []spanChange {
	return []spanChange{
		{SubjectKind: "name", SubjectKey: apexName, Facet: "resolution", Opened: true, Value: resolved(apexAddr)},
		{SubjectKind: "service", SubjectKey: apexSvc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
		{SubjectKind: "endpoint", SubjectKey: apexName + "@" + apexSvc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
		{SubjectKind: "endpoint", SubjectKey: subName + "@" + apexSvc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
		{SubjectKind: "endpoint", SubjectKey: lookalike + "@" + apexSvc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
		{SubjectKind: "endpoint", SubjectKey: subName + "@" + uncitedSvc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
	}
}

func TestMembershipCensusCarriesAnEndpointOnACitedService(t *testing.T) {
	// The fold holds the message and computes no census, so the rule is read here (ADR-1806 §2).
	changes := apexFold()
	census := membershipCensus(changes, changes[0])
	keys := map[string]bool{}
	for _, e := range census.Entries {
		keys[e.Key] = true
	}
	if !keys[apexSvc] || !keys[apexName+"@"+apexSvc] {
		t.Errorf("the cited Service and the apex's own Endpoint are beneath it, got %+v", census.Entries)
	}
	// ADR-0033 §2 names this census the carrier for an Endpoint that entered (#1776).
	if !keys[subName+"@"+apexSvc] {
		t.Errorf("a sub-name's Endpoint entered on the apex's Service, got %+v", census.Entries)
	}
	if !keys[lookalike+"@"+apexSvc] {
		t.Errorf("a foreign Name's Endpoint entered on the apex's Service too, got %+v", census.Entries)
	}
	if keys[subName+"@"+uncitedSvc] {
		t.Errorf("the apex cites no address of that Service, so nothing puts it beneath (#1773), got %+v", census.Entries)
	}
	if census.Len() != 4 {
		t.Errorf("census = %+v, want the Service and the three Endpoints on it", census.Entries)
	}
}

func TestAForeignEndpointKeepsItsOwnMessageBeneathAnApexRoot(t *testing.T) {
	ep := subName + "@" + sensitiveSvc
	changes := []spanChange{
		{SubjectKind: "name", SubjectKey: apexName, Facet: "resolution", Opened: true, Value: resolved("198.51.100.1")},
		{SubjectKind: "endpoint", SubjectKey: ep, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certNoTLS)},
		{SubjectKind: "endpoint", SubjectKey: ep, Facet: "certificate", Opened: true, Value: []byte(certExpired)},
	}
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 31, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	// The Endpoint entered on a Service the apex cites, so the census must reach it at release.
	if c := membershipCensus(changes, changes[0]); c.Len() != 1 || c.Entries[0].Key != ep {
		t.Errorf("the Endpoint is beneath the apex, got %+v", c.Entries)
	}
	var membership, own int
	for _, m := range store.inserted {
		switch {
		case m.SubjectKind == "name" && m.FiredAt == apexName:
			membership++
			basis, err := message.ParseCensusBasis(m.CensusBasis)
			if err != nil {
				t.Fatalf("parse basis: %v", err)
			}
			if basis.RootKey != apexName || !strings.Contains(string(basis.RootValue), "198.51.100.1") {
				t.Errorf("the basis names the apex and the address it cites, got %+v", basis)
			}
		case m.FiredAt == ep:
			own++
		}
	}
	if membership != 1 {
		t.Fatalf("the apex's opening is one membership message, got %+v", store.inserted)
	}
	if own == 0 {
		t.Errorf("a foreign Endpoint's move is no residue of the apex, so it keeps its message; got %+v", store.inserted)
	}
}
