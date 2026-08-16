package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// A blanket responder's internet reach is a Gap (ADR-0104), so a sensitive port on
// it reads as ABSENT — HasInternetReach=false — and sensitive-port-reached-from-
// internet returns not-evaluable (never not-fired, and never fired). An ordinary
// reached sensitive port on another address still fires. The damping is at the
// measurement: no rule is narrowed.
func TestBlanketResponderDampsSensitivePortSignal(t *testing.T) {
	f := newFakeStore()
	// A blanket responder: a sensitive internet pair whose reach is a Gap.
	f.addClassReachability(t, "198.51.100.50:3389/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)
	// An ordinary origin: the same sensitive port reached from the internet -> fires.
	f.addClassReachability(t, "198.51.100.51:3389/tcp", "internet", obsClock, `{"outcome":"reached"}`)

	srv := newServer(f, testKey, "", fixedClock())
	req := httptest.NewRequest(http.MethodGet, "/signals", nil)
	facts, _, err := srv.buildServiceFacts(req)
	if err != nil {
		t.Fatalf("buildServiceFacts: %v", err)
	}

	byKey := map[string]signal.ServiceFacts{}
	for _, sf := range facts {
		byKey[sf.Subject] = sf
	}
	blanket, ok := byKey["198.51.100.50:3389/tcp"]
	if !ok {
		t.Fatal("blanket responder service missing from facts — a blanketed Service is still a subject")
	}
	if !blanket.OnSensitiveList {
		t.Error("the blanketed pair is 3389/tcp — it must stay in the rule's domain")
	}
	if blanket.HasInternetReach {
		t.Errorf("a blanketed internet reach must read as absent, got InternetReach=%q", blanket.InternetReach)
	}
	if origin := byKey["198.51.100.51:3389/tcp"]; !origin.HasInternetReach || origin.InternetReach != signal.Reached {
		t.Errorf("an ordinary reached origin must keep its value, got %+v", origin)
	}

	// The rule buckets the blanket responder as not-evaluable (its evidence is a
	// Gap), and the origin as fired — never the reverse.
	var rule signal.ServiceRule
	for _, r := range signal.AllServiceRules() {
		if r.Name() == "sensitive-port-reached-from-internet" {
			rule = r
		}
	}
	if rule == nil {
		t.Fatal("sensitive-port-reached-from-internet rule not found")
	}
	if got := rule.Eval(blanket); got != signal.NotEvaluable {
		t.Errorf("blanket responder verdict = %v, want not-evaluable (not not-fired)", got)
	}
	if got := rule.Eval(byKey["198.51.100.51:3389/tcp"]); got != signal.Fired {
		t.Errorf("ordinary origin verdict = %v, want fired", got)
	}
}

// seedZone declares a name-scope Seed and attaches a zone file to it, returning
// nothing — the Signals reads pick it up through ListZoneDeclarations.
func seedZone(t *testing.T, f *fakeStore, admin db.Account, domain, content string) {
	t.Helper()
	s, err := f.CreateNameSeed(t.Context(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.CreateZoneFile(t.Context(), db.CreateZoneFileParams{
		SeedID:     s.ID,
		SuppliedAt: pgtype.Timestamptz{Time: obsClock, Valid: true},
		Content:    content,
		UploadedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSignalsRendersEveryRuleCensus(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	zone := "$ORIGIN example.com.\n" +
		"@ IN SOA ns1 admin 1 2 3 4 5\n" +
		"www IN A 203.0.113.20\n" +
		"api IN A 203.0.113.21\n" +
		"missing IN A 203.0.113.22\n"
	seedZone(t, f, admin, "example.com", zone)

	// lame-delegation: an all-refusing delegation → composed Lame → FIRED.
	f.addClassResolution(t, "lame.example.com", "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, "lame.example.com", "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)

	// A plain resolved name → lame-delegation NOT-FIRED, non-global NOT-FIRED.
	f.addClassResolution(t, "good.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.10"]}`)

	// A Shadowed name → lame-delegation NOT-EVALUABLE (distinct from not-fired).
	f.addClassResolution(t, "shadow.example.com", "internet", obsClock, `{"outcome":"Shadowed"}`)

	// An internal address in a public answer → non-global FIRED.
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)

	// A dangling CNAME → cname-target-name-error FIRED.
	f.addClassResolution(t, "alias.example.com", "internet", obsClock, `{"outcome":"NoData"}`)
	f.addDNSRecord(t, "alias.example.com", "CNAME", obsClock, `{"rrs":[{"name":"alias.example.com","type":"CNAME","data":"gone.example.com"}]}`)
	f.addClassResolution(t, "gone.example.com", "internet", obsClock, `{"outcome":"NameError"}`)

	// A declared name our resolver NXDOMAINs → zone-declared-…-name-error FIRED
	// even though it has withdrawn (evidence current, membership irrelevant).
	f.addClassResolution(t, "missing.example.com", "internet", obsClock, `{"outcome":"NameError"}`)

	// A resolving name inside the zone but not declared → absent-from-zone FIRED.
	f.addClassResolution(t, "orphan.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.30"]}`)
	// A declared name that resolves → both zone rules NOT-FIRED.
	f.addClassResolution(t, "www.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.20"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/signals", http.StatusOK)

	// Every rule's census renders.
	for _, rule := range []string{
		"lame-delegation",
		"cname-target-name-error",
		"zone-declared-name-returns-name-error",
		"resolved-name-absent-from-zone",
		"non-globally-reachable-address-resolved-from-internet",
	} {
		if !strings.Contains(page, rule) {
			t.Errorf("Signals page missing rule %q", rule)
		}
	}

	// The version vector composes both leaves it reads.
	if !strings.Contains(page, "resolution-walk/v1") || !strings.Contains(page, "wildcard-discrimination/v1") {
		t.Errorf("version vector not rendered composing both leaves; body: %s", page)
	}

	// Fired members appear and drill to their subject.
	for _, name := range []string{"lame.example.com", "leak.example.com", "orphan.example.com", "missing.example.com", "alias.example.com"} {
		if !strings.Contains(page, `href="/subjects/`+name+`"`) {
			t.Errorf("census member %q not drillable to its subject; body: %s", name, page)
		}
	}

	// The three registers are distinct, labelled members — not-evaluable is not
	// folded into did-not-fire.
	for _, label := range []string{"Fired", "Did not fire", "Not-evaluable"} {
		if !strings.Contains(page, label) {
			t.Errorf("census missing member label %q", label)
		}
	}
	if !strings.Contains(page, "shadow.example.com") {
		t.Errorf("Shadowed name not rendered as a not-evaluable member; body: %s", page)
	}

	// The member header count is locked to list.length — a count element renders.
	if !strings.Contains(page, `class="count"`) {
		t.Errorf("member group renders no locked count; body: %s", page)
	}

	// The census member component is NOT the Subjects row: no search box, no
	// Citation. And a Signal has no severity, and 'finding' is a rejected word.
	if strings.Contains(page, `name="q"`) {
		t.Errorf("Signals page carries a search box; a member list must not")
	}
	if strings.Contains(page, "Citation") {
		t.Errorf("census member carries a Citation; it must not (ADR-0102)")
	}
	low := strings.ToLower(page)
	if strings.Contains(low, "finding") {
		t.Errorf("Signals page uses the rejected word 'finding'")
	}
	if strings.Contains(low, "severity") {
		t.Errorf("Signals page mentions severity; a Signal carries none")
	}
}

func TestSignalsRendersServiceAndEndpointRules(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// A name in the estate, so the redirect-to-host rule has an estate to test against.
	f.addClassResolution(t, "good.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	// sensitive-port-reached-from-internet: a sensitive pair (3389/tcp) reached from
	// the internet → FIRED. A non-sensitive pair (443/tcp) is outside the domain.
	f.addClassReachability(t, "198.51.100.1:3389/tcp", "internet", obsClock, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.9:443/tcp", "internet", obsClock, `{"outcome":"reached"}`)
	// A sensitive pair seen only internally → no internet leg → NOT-EVALUABLE.
	f.addClassReachability(t, "198.51.100.2:445/tcp", "internal", obsClock, `{"outcome":"reached"}`)

	// A presented certificate → the five cert-detail rules render it NOT-EVALUABLE
	// (the parsed leaf is not stored), and it is inside hostname-san-mismatch's domain.
	f.addCertificate(t, "secure.example.com@198.51.100.1:443/tcp", obsClock, `{"outcome":"presented","chain":["sha256:abc"]}`)

	// A plaintext endpoint: HTTP responded (200) and the certificate is no-tls →
	// plaintext-http-no-https FIRED, and unauthenticated-request-answered FIRED.
	f.addHTTPIdentity(t, "plain.example.com@198.51.100.5:80/tcp", obsClock, `{"outcome":"responded","status":200}`)
	f.addCertificate(t, "plain.example.com@198.51.100.5:80/tcp", obsClock, `{"outcome":"no-tls"}`)

	// A redirect that does not upgrade and points outside the estate → both redirect
	// rules FIRED.
	f.addHTTPIdentity(t, "redir.example.com@198.51.100.6:80/tcp", obsClock, `{"outcome":"responded","status":301,"redirect_location":"http://outside.test/x"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/signals", http.StatusOK)

	// All seventeen rules render.
	for _, rule := range []string{
		"lame-delegation", "cname-target-name-error", "zone-declared-name-returns-name-error",
		"resolved-name-absent-from-zone", "non-globally-reachable-address-resolved-from-internet",
		"certificate-expired", "certificate-not-yet-valid", "certificate-expiring",
		"certificate-self-signed", "certificate-weak-key-or-signature", "certificate-hostname-san-mismatch",
		"plaintext-http-no-https", "redirect-does-not-upgrade-to-tls", "redirect-to-host-outside-estate",
		"unauthenticated-request-answered", "tls-1.0-accepted", "sensitive-port-reached-from-internet",
	} {
		if !strings.Contains(page, rule) {
			t.Errorf("Signals page missing rule %q", rule)
		}
	}

	// Fired Service and Endpoint members drill to their subjects via the route
	// their kind actually serves (#248): a Service or Endpoint key carries a `/`
	// (an Endpoint also an `@`), so it rides the `?key=` page escaped, never the
	// `/subjects/{key}` path that would 404 on the second segment.
	for _, tc := range []struct{ kind, subject string }{
		{"service", "198.51.100.1:3389/tcp"},                    // sensitive-port fired
		{"endpoint", "plain.example.com@198.51.100.5:80/tcp"},   // plaintext-http fired
		{"endpoint", "redir.example.com@198.51.100.6:80/tcp"},   // redirect rules fired
		{"endpoint", "secure.example.com@198.51.100.1:443/tcp"}, // certificate not-evaluable member
	} {
		want := `href="` + subjectHref(tc.kind, tc.subject) + `"`
		if !strings.Contains(page, want) {
			t.Errorf("census member %q not drillable to its %s drill-down (want %s)", tc.subject, tc.kind, want)
		}
	}

	// The version vectors compose the leaves the rules read.
	for _, ver := range []string{"tls-handshake/v1", "http-exchange/v2", "connect-outcome/v1", "tls-acceptance/v1"} {
		if !strings.Contains(page, ver) {
			t.Errorf("version vector not rendered composing %q", ver)
		}
	}

	// tls-1.0-accepted reads a facet whose leaf (#199) has not landed → its domain
	// is empty and it renders a no-population panel, not a compile dependency.
	if !strings.Contains(page, "No population") {
		t.Errorf("tls-1.0-accepted should render a no-population panel with no tls-acceptance data")
	}
}

func TestSignalsEmptyEstateRendersNoPopulation(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/signals", http.StatusOK)
	// With no subjects, every rule's domain is empty: a no-population panel, never
	// a census of zeroes.
	if !strings.Contains(page, "No population") {
		t.Errorf("empty estate did not render a no-population panel; body: %s", page)
	}
	// Rules still render (the page is the rule set, current state).
	if !strings.Contains(page, "lame-delegation") {
		t.Errorf("rules not rendered on an empty estate; body: %s", page)
	}
}

func TestSignalsRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/signals")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /signals: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
