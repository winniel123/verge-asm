package signal

import "testing"

func cb(b bool) *bool { return &b }

func presented(d *CertDetails) EndpointFacts {
	return EndpointFacts{CertMeasured: true, CertOutcome: CertPresented, CertDetails: d, HasName: true}
}

func TestCertificateRulesDomainIsPresented(t *testing.T) {
	rules := []EndpointRule{
		certificateExpired, certificateNotYetValid, certificateExpiring,
		certificateSelfSigned, certificateWeakKeyOrSignature, certificateHostnameSANMismatch{},
	}
	outside := []EndpointFacts{
		{CertMeasured: true, CertOutcome: CertNoTLS, HasName: true},
		{CertMeasured: true, CertOutcome: CertTLSRefused, HasName: true},
		{CertMeasured: false, HasName: true},
	}
	for _, r := range rules {
		for _, f := range outside {
			if got := r.Eval(f); got != OutsideDomain {
				t.Errorf("%s on %+v: Eval = %q, want OutsideDomain", r.Name(), f, got)
			}
		}
	}
}

func TestCertificatePresentedButUnreadableIsNotEvaluable(t *testing.T) {
	rules := []EndpointRule{
		certificateExpired, certificateNotYetValid, certificateExpiring,
		certificateSelfSigned, certificateWeakKeyOrSignature, certificateHostnameSANMismatch{},
	}
	for _, r := range rules {
		if got := r.Eval(presented(nil)); got != NotEvaluable {
			t.Errorf("%s: presented-but-unreadable Eval = %q, want NotEvaluable", r.Name(), got)
		}
	}
}

func TestCertificatePredicates(t *testing.T) {
	cases := []struct {
		rule  EndpointRule
		fired *CertDetails
		clean *CertDetails
	}{
		{certificateExpired, &CertDetails{Expired: cb(true)}, &CertDetails{Expired: cb(false)}},
		{certificateNotYetValid, &CertDetails{NotYetValid: cb(true)}, &CertDetails{NotYetValid: cb(false)}},
		{certificateExpiring, &CertDetails{Expiring: cb(true)}, &CertDetails{Expiring: cb(false)}},
		{certificateSelfSigned, &CertDetails{SelfSigned: cb(true)}, &CertDetails{SelfSigned: cb(false)}},
		{certificateWeakKeyOrSignature, &CertDetails{WeakKeyOrSignature: cb(true)}, &CertDetails{WeakKeyOrSignature: cb(false)}},
		{certificateHostnameSANMismatch{}, &CertDetails{SANMatchesName: cb(false)}, &CertDetails{SANMatchesName: cb(true)}},
	}
	for _, c := range cases {
		if got := c.rule.Eval(presented(c.fired)); got != Fired {
			t.Errorf("%s: predicate-true Eval = %q, want Fired", c.rule.Name(), got)
		}
		if got := c.rule.Eval(presented(c.clean)); got != NotFired {
			t.Errorf("%s: predicate-false Eval = %q, want NotFired", c.rule.Name(), got)
		}
	}
}

func TestHostnameSANMismatchNamelessIsOutside(t *testing.T) {
	f := EndpointFacts{CertMeasured: true, CertOutcome: CertPresented, HasName: false, CertDetails: &CertDetails{SANMatchesName: cb(false)}}
	if got := (certificateHostnameSANMismatch{}).Eval(f); got != OutsideDomain {
		t.Fatalf("nameless endpoint Eval = %q, want OutsideDomain", got)
	}
}

func TestCertDetailPerAttributeNullability(t *testing.T) {
	notEvaluableWhenAttrNil := []struct {
		rule EndpointRule
		d    *CertDetails
	}{
		{certificateNotYetValid, &CertDetails{Expired: cb(false)}},
		{certificateSelfSigned, &CertDetails{Expired: cb(false)}},
		{certificateWeakKeyOrSignature, &CertDetails{Expired: cb(false)}},
		{certificateHostnameSANMismatch{}, &CertDetails{Expired: cb(false)}},
		{certificateHostnameSANMismatch{}, &CertDetails{Expiring: cb(true)}},
	}
	for _, c := range notEvaluableWhenAttrNil {
		if got := c.rule.Eval(presented(c.d)); got != NotEvaluable {
			t.Errorf("%s with its attribute nil: Eval = %q, want NotEvaluable", c.rule.Name(), got)
		}
	}

	if got := certificateExpired.Eval(presented(&CertDetails{Expired: cb(true)})); got != Fired {
		t.Errorf("certificate-expired with Expired=true: Eval = %q, want Fired", got)
	}
	if got := certificateExpiring.Eval(presented(&CertDetails{Expiring: cb(true)})); got != Fired {
		t.Errorf("certificate-expiring with Expiring=true: Eval = %q, want Fired", got)
	}

	if got := (certificateHostnameSANMismatch{}).Eval(presented(&CertDetails{SANMatchesName: nil})); got == Fired {
		t.Fatalf("absent SANs must NEVER fire a mismatch; Eval = %q", got)
	}
}

func TestCertificateVersionsComposeTLSHandshake(t *testing.T) {
	const want = "rule@v1|tls-handshake/v3"
	for _, r := range []EndpointRule{
		certificateExpired, certificateNotYetValid, certificateExpiring,
		certificateSelfSigned, certificateHostnameSANMismatch{},
	} {
		if got := r.Version().String(); got != want {
			t.Errorf("%s version = %q, want %q", r.Name(), got, want)
		}
	}
	const wantWeak = "rule@v1|tls-handshake/v3|weak-key-floor/v1"
	if got := certificateWeakKeyOrSignature.Version().String(); got != wantWeak {
		t.Errorf("certificate-weak-key-or-signature version = %q, want %q", got, wantWeak)
	}
}

func TestPlaintextHTTPNoHTTPS(t *testing.T) {
	r := plaintextHTTPNoHTTPS{}
	cases := []struct {
		name string
		f    EndpointFacts
		want Outcome
	}{
		{"no HTTP response is outside the domain",
			EndpointFacts{HTTPResponded: false}, OutsideDomain},
		{"responded and no-tls fires",
			EndpointFacts{HTTPResponded: true, CertMeasured: true, CertOutcome: CertNoTLS}, Fired},
		{"responded and presented does not fire",
			EndpointFacts{HTTPResponded: true, CertMeasured: true, CertOutcome: CertPresented}, NotFired},
		{"responded but certificate unmeasured is not-evaluable",
			EndpointFacts{HTTPResponded: true, CertMeasured: false}, NotEvaluable},
	}
	for _, c := range cases {
		if got := r.Eval(c.f); got != c.want {
			t.Errorf("%s: Eval = %q, want %q", c.name, got, c.want)
		}
	}
	const wantVer = "rule@v1|http-exchange/v2|tls-handshake/v3"
	if got := r.Version().String(); got != wantVer {
		t.Fatalf("version = %q, want %q", got, wantVer)
	}
}

func TestRedirectDoesNotUpgradeToTLS(t *testing.T) {
	r := redirectDoesNotUpgradeToTLS{}
	cases := []struct {
		name string
		f    EndpointFacts
		want Outcome
	}{
		{"non-3xx is outside the domain",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 200, RedirectLocation: "https://x/"}, OutsideDomain},
		{"3xx with no Location is outside the domain",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 301, RedirectLocation: ""}, OutsideDomain},
		{"redirect to http does not upgrade — fires",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 302, RedirectLocation: "http://example.com/"}, Fired},
		{"redirect to https upgrades — does not fire",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 301, RedirectLocation: "https://example.com/"}, NotFired},
		{"relative redirect keeps plaintext — fires",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 302, RedirectLocation: "/login"}, Fired},
	}
	for _, c := range cases {
		if got := r.Eval(c.f); got != c.want {
			t.Errorf("%s: Eval = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestRedirectToHostOutsideEstate(t *testing.T) {
	r := redirectToHostOutsideEstate{}
	cases := []struct {
		name string
		f    EndpointFacts
		want Outcome
	}{
		{"non-redirect is outside the domain",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 200, RedirectLocation: "http://evil.test/"}, OutsideDomain},
		{"redirect to a host in the estate does not fire",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 301, RedirectLocation: "https://in.example.com/", RedirectHostInEstate: true}, NotFired},
		{"redirect to a host outside the estate fires",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 302, RedirectLocation: "https://evil.test/", RedirectHostInEstate: false}, Fired},
		{"relative redirect stays on this origin — does not fire",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 302, RedirectLocation: "/next", RedirectHostInEstate: false}, NotFired},
	}
	for _, c := range cases {
		if got := r.Eval(c.f); got != c.want {
			t.Errorf("%s: Eval = %q, want %q", c.name, got, c.want)
		}
	}
	const wantVer = "rule@v1|http-exchange/v2|resolution-walk/v1|wildcard-discrimination/v1"
	if got := r.Version().String(); got != wantVer {
		t.Fatalf("version = %q, want %q", got, wantVer)
	}
}

func TestUnauthenticatedRequestAnswered(t *testing.T) {
	r := unauthenticatedRequestAnswered{}
	cases := []struct {
		name string
		f    EndpointFacts
		want Outcome
	}{
		{"no HTTP response is outside the domain",
			EndpointFacts{HTTPResponded: false}, OutsideDomain},
		{"a 3xx is outside the domain (it is the redirect rules')",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 302}, OutsideDomain},
		{"a 500 is outside the domain",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 500}, OutsideDomain},
		{"a 200 answered the unauthenticated request — fires",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 200}, Fired},
		{"a 401 challenged — does not fire",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 401}, NotFired},
		{"a 403 challenged — does not fire",
			EndpointFacts{HTTPResponded: true, HTTPStatus: 403}, NotFired},
	}
	for _, c := range cases {
		if got := r.Eval(c.f); got != c.want {
			t.Errorf("%s: Eval = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestRedirectTargetParsing(t *testing.T) {
	cases := []struct {
		in           string
		scheme, host string
	}{
		{"https://Example.COM/path", "https", "example.com"},
		{"http://evil.test:8080/", "http", "evil.test"},
		{"/relative/path", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		s, h := RedirectTarget(c.in)
		if s != c.scheme || h != c.host {
			t.Errorf("RedirectTarget(%q) = (%q,%q), want (%q,%q)", c.in, s, h, c.scheme, c.host)
		}
	}
}

func TestEvaluateCorpusReturnsSeventeenRules(t *testing.T) {
	got := EvaluateCorpus(Corpus{Names: []NameFacts{{Name: "a", Resolution: Resolved}}})
	if len(got) != 17 {
		t.Fatalf("EvaluateCorpus returned %d censuses, want 17", len(got))
	}
	if len(AllRuleNames()) != 17 {
		t.Fatalf("AllRuleNames has %d, want 17", len(AllRuleNames()))
	}
	seen := map[string]bool{}
	for _, n := range AllRuleNames() {
		if seen[n] {
			t.Fatalf("duplicate rule name %q", n)
		}
		seen[n] = true
	}
}
