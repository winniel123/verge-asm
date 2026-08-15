package signal

import "testing"

func TestSensitivePortReachedFromInternet(t *testing.T) {
	r := sensitivePortReachedFromInternet{}
	cases := []struct {
		name string
		f    ServiceFacts
		want Outcome
	}{
		{"off the sensitive list is outside the domain",
			ServiceFacts{OnSensitiveList: false, HasInternetReach: true, InternetReach: Reached}, OutsideDomain},
		{"sensitive and reached from the internet fires",
			ServiceFacts{OnSensitiveList: true, HasInternetReach: true, InternetReach: Reached}, Fired},
		{"sensitive and not reached from the internet does not fire",
			ServiceFacts{OnSensitiveList: true, HasInternetReach: true, InternetReach: NotReached}, NotFired},
		{"sensitive but no internet-class value is not-evaluable (a Gap)",
			ServiceFacts{OnSensitiveList: true, HasInternetReach: false}, NotEvaluable},
	}
	for _, c := range cases {
		if got := r.Eval(c.f); got != c.want {
			t.Errorf("%s: Eval = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSensitivePortVersionComposesConnectOutcome(t *testing.T) {
	got := sensitivePortReachedFromInternet{}.Version().String()
	const want = "rule@v1|connect-outcome/v1"
	if got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
}

func TestTLS10Accepted(t *testing.T) {
	r := tls10Accepted{}
	cases := []struct {
		name string
		f    ServiceFacts
		want Outcome
	}{
		{"no handshake completed is outside the domain (facet absent — #199 not landed)",
			ServiceFacts{TLSHandshakeCompleted: false}, OutsideDomain},
		{"handshake completed, TLS 1.0 accepted fires",
			ServiceFacts{TLSHandshakeCompleted: true, TLSVersionsReadable: true, TLS10Accepted: true}, Fired},
		{"handshake completed, TLS 1.0 not accepted does not fire",
			ServiceFacts{TLSHandshakeCompleted: true, TLSVersionsReadable: true, TLS10Accepted: false}, NotFired},
		{"handshake completed but versions unreadable is not-evaluable",
			ServiceFacts{TLSHandshakeCompleted: true, TLSVersionsReadable: false}, NotEvaluable},
	}
	for _, c := range cases {
		if got := r.Eval(c.f); got != c.want {
			t.Errorf("%s: Eval = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTLS10AcceptedVersionReadsTLSAcceptanceByName(t *testing.T) {
	got := tls10Accepted{}.Version().String()
	const want = "rule@v1|tls-acceptance/v1"
	if got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
}

func TestEvaluateServiceDropsOutsideDomainAndOrders(t *testing.T) {
	services := []ServiceFacts{
		{Subject: "198.51.100.9:22/tcp", OnSensitiveList: true, HasInternetReach: true, InternetReach: Reached},
		{Subject: "198.51.100.1:22/tcp", OnSensitiveList: true, HasInternetReach: true, InternetReach: Reached},
		{Subject: "198.51.100.2:443/tcp", OnSensitiveList: false, HasInternetReach: true, InternetReach: Reached},
		{Subject: "198.51.100.3:3389/tcp", OnSensitiveList: true, HasInternetReach: false},
	}
	c := EvaluateService(sensitivePortReachedFromInternet{}, services)
	// The non-sensitive :443 service is outside the domain — not rendered at all.
	if c.InDomain() != 3 {
		t.Fatalf("InDomain = %d, want 3 (the non-sensitive service excluded)", c.InDomain())
	}
	if len(c.Fired) != 2 || len(c.NotEvaluable) != 1 || len(c.NotFired) != 0 {
		t.Fatalf("members: fired=%d notfired=%d ne=%d", len(c.Fired), len(c.NotFired), len(c.NotEvaluable))
	}
	if c.Fired[0].Subject != "198.51.100.1:22/tcp" || c.Fired[1].Subject != "198.51.100.9:22/tcp" {
		t.Fatalf("fired not ordered by subject: %v", c.Fired)
	}
}

func TestTLS10AcceptedEmptyWhenFacetAbsent(t *testing.T) {
	// The whole population lacks the tls-acceptance facet — no member is in the
	// domain, so the census is a no-population panel, not a compile dependency.
	services := []ServiceFacts{
		{Subject: "198.51.100.1:443/tcp"},
		{Subject: "198.51.100.2:8443/tcp"},
	}
	c := EvaluateService(tls10Accepted{}, services)
	if !c.Empty() {
		t.Fatalf("tls-1.0-accepted with no facet data should be Empty, got InDomain=%d", c.InDomain())
	}
}
