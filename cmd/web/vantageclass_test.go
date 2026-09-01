package main

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// presentedAddrs turns a vantage's persisted facts into the 0/1/2 addresses an outside
// observer saw of it (#710): the dialled peer and the SSH_CLIENT egress, unmapped, with
// NULL/unparseable facts dropped and never fabricated.
func TestPresentedAddrs(t *testing.T) {
	txt := func(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
	cases := []struct {
		name    string
		dialled string
		egress  string
		want    []string
	}{
		{"no facts is empty", "", "", nil},
		{"dialled only", "203.0.113.9", "", []string{"203.0.113.9"}},
		{"egress only", "", "198.51.100.7", []string{"198.51.100.7"}},
		{"both (dialled then egress)", "203.0.113.9", "198.51.100.7", []string{"203.0.113.9", "198.51.100.7"}},
		{"unparseable dialled drops out", "not-an-addr", "198.51.100.7", []string{"198.51.100.7"}},
		{"ipv6 is unmapped", "2001:db8::1", "", []string{"2001:db8::1"}},
		{"v4-mapped v6 is unmapped to v4", "::ffff:203.0.113.9", "", []string{"203.0.113.9"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := presentedAddrs(db.Vantage{DialledAddr: txt(c.dialled), Egress: txt(c.egress)})
			if len(got) != len(c.want) {
				t.Fatalf("presentedAddrs = %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i].String() != c.want[i] {
					t.Errorf("addr[%d] = %s, want %s", i, got[i], c.want[i])
				}
			}
		})
	}
}

// deriveVantageClasses classifies each vantage per read from its presented facts (#709):
// no facts -> unverified; any presented address uncovered by an address scope ->
// internet (the closed direction); every presented address covered -> internal.
func TestDeriveVantageClasses(t *testing.T) {
	scope := netip.MustParsePrefix("10.0.0.0/8")
	covered := func(a netip.Addr) bool { return scope.Contains(a.Unmap()) }
	txt := func(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

	vantages := []db.Vantage{
		{ID: 1},                                     // no facts -> unverified
		{ID: 2, DialledAddr: txt("10.0.0.5")},       // covered -> internal
		{ID: 3, DialledAddr: txt("203.0.113.9")},    // uncovered -> internet
		{ID: 4, DialledAddr: txt("10.0.0.5"), Egress: txt("203.0.113.9")}, // one uncovered -> internet
	}
	got := deriveVantageClasses(vantages, covered)
	want := map[int64]custody.VantageClass{
		1: custody.ClassUnverified,
		2: custody.ClassInternal,
		3: custody.ClassInternet,
		4: custody.ClassInternet,
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("vantage %d class = %q, want %q", id, got[id], w)
		}
	}
}

// The two internet-gated rules light ONLY because the detecting vantage DERIVES to
// `internet` from its presented address against the declared scopes (#709) — not from a
// stored column. With the same evidence under an `internal`-deriving vantage, both fall
// outside their internet-scoped domain. This also proves Exposure's internet leg
// composes a real value (HasInternetReach / InternetReach) off the derived class.
func TestDerivedInternetClassLightsFlagshipRules(t *testing.T) {
	const (
		sensitiveSvc = "198.51.100.51:3389/tcp"       // RDP — on the sensitive list
		leakyName    = "leak.example.com"             // resolves to a non-globally-reachable addr
		leakyResol   = `{"outcome":"Resolved","addresses":["10.0.0.5"]}`
	)

	// --- internet-deriving vantage: both rules FIRE ---
	f := newFakeStore()
	f.addClassReachability(t, sensitiveSvc, "internet", obsClock, `{"outcome":"reached"}`)
	f.addClassResolution(t, leakyName, "internet", obsClock, leakyResol)

	srv := newServer(f, testKey, "", fixedClock())
	req := httptest.NewRequest(http.MethodGet, "/signals", nil)

	svcFacts, _, err := srv.buildServiceFacts(req)
	if err != nil {
		t.Fatalf("buildServiceFacts: %v", err)
	}
	sf := serviceFactsByKey(svcFacts)[sensitiveSvc]
	if !sf.HasInternetReach || sf.InternetReach != signal.Reached {
		t.Fatalf("internet leg did not compose off the derived class: %+v", sf)
	}
	if got := serviceRule(t, "sensitive-port-reached-from-internet").Eval(sf); got != signal.Fired {
		t.Errorf("sensitive-port-reached-from-internet verdict = %v, want fired", got)
	}

	nameFacts, err := srv.buildNameFacts(req)
	if err != nil {
		t.Fatalf("buildNameFacts: %v", err)
	}
	nf := nameFactsByKey(nameFacts)[leakyName]
	if !nf.HasInternetVantage {
		t.Fatalf("name %q has no internet-class view despite an internet-deriving vantage: %+v", leakyName, nf)
	}
	if got := nameRule(t, "non-globally-reachable-address-resolved-from-internet").Eval(nf); got != signal.Fired {
		t.Errorf("non-globally-reachable-address-resolved-from-internet verdict = %v, want fired", got)
	}

	// --- control: identical evidence, but an internal-deriving vantage ---
	g := newFakeStore()
	g.addClassReachability(t, sensitiveSvc, "internal", obsClock, `{"outcome":"reached"}`)
	g.addClassResolution(t, leakyName, "internal", obsClock, leakyResol)
	gsrv := newServer(g, testKey, "", fixedClock())
	greq := httptest.NewRequest(http.MethodGet, "/signals", nil)

	gsf := serviceFactsByKey(mustServiceFacts(t, gsrv, greq))[sensitiveSvc]
	if gsf.HasInternetReach {
		t.Errorf("an internal-deriving vantage must NOT compose an internet leg, got %+v", gsf)
	}
	if got := serviceRule(t, "sensitive-port-reached-from-internet").Eval(gsf); got == signal.Fired {
		t.Error("sensitive-port-reached-from-internet fired on an internal-deriving vantage")
	}
	gnf := nameFactsByKey(mustNameFacts(t, gsrv, greq))[leakyName]
	if gnf.HasInternetVantage {
		t.Errorf("an internal-deriving vantage must NOT give a name an internet-class view, got %+v", gnf)
	}
	if got := nameRule(t, "non-globally-reachable-address-resolved-from-internet").Eval(gnf); got == signal.Fired {
		t.Error("non-globally-reachable-address-resolved-from-internet fired on an internal-deriving vantage")
	}
}

func serviceFactsByKey(facts []signal.ServiceFacts) map[string]signal.ServiceFacts {
	m := map[string]signal.ServiceFacts{}
	for _, f := range facts {
		m[f.Subject] = f
	}
	return m
}

func nameFactsByKey(facts []signal.NameFacts) map[string]signal.NameFacts {
	m := map[string]signal.NameFacts{}
	for _, f := range facts {
		m[f.Name] = f
	}
	return m
}

func mustServiceFacts(t *testing.T, s *server, r *http.Request) []signal.ServiceFacts {
	t.Helper()
	facts, _, err := s.buildServiceFacts(r)
	if err != nil {
		t.Fatalf("buildServiceFacts: %v", err)
	}
	return facts
}

func mustNameFacts(t *testing.T, s *server, r *http.Request) []signal.NameFacts {
	t.Helper()
	facts, err := s.buildNameFacts(r)
	if err != nil {
		t.Fatalf("buildNameFacts: %v", err)
	}
	return facts
}

// addressScopeCovered narrows with the derivation (ADR-0133 §4). A declared `address`
// exclusion takes its addresses out of the `covered` predicate the Vantage-class
// derivation binds, and this is the RECLASSIFICATION that follows: a vantage whose
// dialled address sat inside a covered range and now sits inside an excluded one
// derives `internet` rather than `internal`.
//
// The consequence is accepted rather than worked around. #711's invariant is ONE
// binding used identically by batch gating and every render, so there is no second,
// un-narrowed predicate for classification alone.
func TestAddressScopeCoveredNarrowsByAnAddressExclusion(t *testing.T) {
	// The fake's ListAddressScopeCidrs always returns the 10.0.0.0/8 convention scope.
	inside := netip.MustParseAddr("10.200.0.1")
	outside := netip.MustParseAddr("10.0.0.5")

	f := newFakeStore()
	s := newServer(f, testKey, "", fixedClock())

	covered, err := s.addressScopeCovered(t.Context())
	if err != nil {
		t.Fatalf("addressScopeCovered: %v", err)
	}
	if !covered(inside) || !covered(outside) {
		t.Fatal("both fixtures must start covered by the convention scope, or the row below pins nothing")
	}
	classes := deriveVantageClasses([]db.Vantage{{ID: 1, DialledAddr: pgtype.Text{String: inside.String(), Valid: true}}}, covered)
	if classes[1] != custody.ClassInternal {
		t.Fatalf("class = %q, want %q before the exclusion is declared", classes[1], custody.ClassInternal)
	}

	excl := netip.MustParsePrefix("10.200.0.0/24")
	if _, err := f.CreateAddressExclusion(t.Context(), db.CreateAddressExclusionParams{AddressCidr: &excl, CreatedBy: 1}); err != nil {
		t.Fatalf("declare the exclusion: %v", err)
	}

	covered, err = s.addressScopeCovered(t.Context())
	if err != nil {
		t.Fatalf("addressScopeCovered after the exclusion: %v", err)
	}
	if covered(inside) {
		t.Error("an address inside a declared exclusion still reads as address-scope covered: the class predicate did not narrow")
	}
	if !covered(outside) {
		t.Error("an address the exclusion does not cover lost its coverage: the exclusion removed more than it names")
	}
	classes = deriveVantageClasses([]db.Vantage{{ID: 1, DialledAddr: pgtype.Text{String: inside.String(), Valid: true}}}, covered)
	if classes[1] != custody.ClassInternet {
		t.Errorf("class = %q, want %q: a vantage inside a newly excluded range reclassifies (ADR-0133 §4)", classes[1], custody.ClassInternet)
	}
}

func serviceRule(t *testing.T, name string) signal.ServiceRule {
	t.Helper()
	for _, r := range signal.AllServiceRules() {
		if r.Name() == name {
			return r
		}
	}
	t.Fatalf("service rule %q not found", name)
	return nil
}

func nameRule(t *testing.T, name string) signal.Rule {
	t.Helper()
	for _, r := range signal.All() {
		if r.Name() == name {
			return r
		}
	}
	t.Fatalf("name rule %q not found", name)
	return nil
}
