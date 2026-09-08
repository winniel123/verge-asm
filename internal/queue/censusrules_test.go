package queue

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

const (
	sensitiveSvc = "198.51.100.1:3306/tcp"
	sensitiveEp  = "admin.example.com@" + sensitiveSvc

	certNoTLS   = `{"outcome":"no-tls"}`
	certRefused = `{"outcome":"tls-refused"}`
	// Expired at produceT0 and self-signed; the SAN matches, the key is 2048-bit RSA.
	certExpired = `{"outcome":"presented","not_before":"2026-01-01T00:00:00Z","not_after":"2026-06-01T00:00:00Z",` +
		`"san_dns":["admin.example.com"],"chain_certs":[{"subject":"CN=admin","issuer":"CN=admin","self_sig_verifies":true,"key_alg":"RSA","key_bits":2048,"sig_digest":"SHA-256"}]}`
	certValid = `{"outcome":"presented","not_before":"2026-01-01T00:00:00Z","not_after":"2027-06-01T00:00:00Z",` +
		`"san_dns":["admin.example.com"],"chain_certs":[{"subject":"CN=admin","issuer":"CN=ca","self_sig_verifies":false,"key_alg":"RSA","key_bits":2048,"sig_digest":"SHA-256"}]}`

	httpNone = `{"outcome":"no-http-response"}`
	http200  = `{"outcome":"responded","status":200}`
)

func ruleKeys(t *testing.T, raw []byte) (rules []string, byKey map[string]message.CensusEntry) {
	t.Helper()
	c, err := message.ParseCensus(raw)
	if err != nil {
		t.Fatalf("parse census: %v", err)
	}
	byKey = map[string]message.CensusEntry{}
	for _, e := range c.Rules() {
		rules = append(rules, e.Key)
		byKey[e.Key] = e
	}
	return rules, byKey
}

func flagshipStore(svc string) *fakeMessageStore {
	return &fakeMessageStore{
		prev:    prevAt(produceT0.Add(-time.Hour)),
		current: []db.ListServiceReachabilitySpansByClassForServicesRow{internetReachRow(svc, "reached")},
		at:      []db.ListServiceReachabilitySpansByClassAtForServicesRow{internetReachAtRow(svc, "not-reached")},
	}
}

func TestFlagshipCensusCarriesTheRulesThatOpenedAtFiredAndNamesThePort(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "service", SubjectKey: sensitiveSvc, Facet: "reachability", Value: reachValue("reached"), Previous: reachValue("not-reached")},
		{SubjectKind: "endpoint", SubjectKey: sensitiveEp, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certNoTLS)},
	}
	store := flagshipStore(sensitiveSvc)
	var log []routed
	if err := produceMessages(context.Background(), store, 20, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	var flagship *db.InsertMessageParams
	for i := range store.inserted {
		if store.inserted[i].SubjectKind == "service" && store.inserted[i].FiredAt == sensitiveSvc {
			flagship = &store.inserted[i]
		}
	}
	if flagship == nil {
		t.Fatalf("no flagship written, got %+v", store.inserted)
	}
	rules, byKey := ruleKeys(t, flagship.Census)
	if strings.Join(rules, ",") != "certificate-expired,certificate-self-signed,sensitive-port-reached-from-internet" {
		t.Errorf("rule entries = %v, want every rule that opened at fired beneath the Service", rules)
	}
	if got := byKey["sensitive-port-reached-from-internet"].Detail; got != "3306/tcp" {
		t.Errorf("the sensitive-port entry names the port, got %q", got)
	}
	if !strings.Contains(flagship.Headline, "sensitive-port-reached-from-internet (3306/tcp)") || !strings.Contains(flagship.Headline, "3 rules opened at fired") {
		t.Errorf("the headline carries the rule entries for the channel body: %q", flagship.Headline)
	}
	c, _ := message.ParseCensus(flagship.Census)
	if len(c.Facets()) != 0 {
		t.Errorf("a moved facet is no opening, so no facet entry rides the census: %+v", c.Entries)
	}
}

func TestFlagshipWithNoFiredRuleCarriesFacetEntriesOnly(t *testing.T) {
	const svc = "198.51.100.1:443/tcp"
	const ep = "www.example.com@" + svc
	changes := []spanChange{
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Value: reachValue("reached"), Previous: reachValue("not-reached")},
		{SubjectKind: "endpoint", SubjectKey: ep, Facet: "certificate", Opened: true, Value: []byte(strings.ReplaceAll(certValid, "admin.example.com", "www.example.com"))},
	}
	store := flagshipStore(svc)
	var log []routed
	if err := produceMessages(context.Background(), store, 21, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("want the flagship alone, got %d", len(store.inserted))
	}
	c, _ := message.ParseCensus(store.inserted[0].Census)
	if c.Len() != 1 || c.Entries[0].Kind != message.KindFacet || c.Entries[0].Key != "certificate" {
		t.Errorf("a flagship with no fired rule carries the same census as before: %+v", c.Entries)
	}
	if h := store.inserted[0].Headline; h != svc+" reached from the internet · 1 facet opened beneath it" {
		t.Errorf("headline unchanged without a rule entry, got %q", h)
	}
}
