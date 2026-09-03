package custody

import (
	"fmt"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// TestRegistrableDomainsReduceThroughThePSL: each hostname reduces to its
// eTLD+1, and a multi-label public suffix reduces to the right place — the PSL
// is what decides, never a count of labels.
func TestRegistrableDomainsReduceThroughThePSL(t *testing.T) {
	got := RegistrableDomains([]string{
		"www.example.com",
		"api.staging.example.com",
		"example.com",
		"shop.example.co.uk",
		"a.b.c.d.example.net",
	})
	want := []string{"example.co.uk", "example.com", "example.net"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RegistrableDomains = %v, want %v", got, want)
	}
}

// TestWildcardSANReducesToTheNameBeneathIt: `*.example.com` counts as
// `example.com`, and the wildcard never becomes a domain of its own.
func TestWildcardSANReducesToTheNameBeneathIt(t *testing.T) {
	cases := map[string]string{
		"*.example.com":       "example.com",
		"*.a.b.example.co.uk": "example.co.uk",
		"*.EXAMPLE.NET":       "example.net",
	}
	for san, want := range cases {
		got, ok := registrableDomain(san)
		if !ok {
			t.Errorf("registrableDomain(%q) dropped, want %q", san, want)
			continue
		}
		if got != want {
			t.Errorf("registrableDomain(%q) = %q, want %q", san, got, want)
		}
	}
	// A wildcard SAN and the bare name beneath it are ONE registrable domain,
	// never two.
	if n := FanOut([]string{"*.example.com", "example.com", "www.example.com"}); n != 1 {
		t.Errorf("FanOut over one domain's wildcard and names = %d, want 1", n)
	}
}

// TestSANWithNoRegistrableDomainIsDropped: an address SAN, and a name that
// reduces to nothing, raise the count by zero. The IPv4 case is the sharp one —
// its spelling is LDH, so the PSL's wildcard rule would otherwise hand back the
// nonsense eTLD+1 `2.1`.
func TestSANWithNoRegistrableDomainIsDropped(t *testing.T) {
	dropped := []string{
		"192.0.2.1",             // iPAddress SAN, IPv4
		"2001:db8::1",           // iPAddress SAN, IPv6
		"999.999.999.999",       // dotted-numeric, but no valid address
		"1.2.3",                 // dotted-numeric, three labels
		"3232235777",            // an address as one decimal
		"co.uk",                 // a bare public suffix
		"com",                   // a bare TLD
		"localhost",             // a single label
		"",                      // empty
		"   ",                   // whitespace only
		".",                     // the root
		"*",                     // a bare wildcard
		"foo.*.example.com",     // a wildcard that is not the first label
		"https://example.com/a", // a uniformResourceIdentifier SAN
		"admin@example.com",     // an rfc822Name SAN
		"exa mple.com",          // a space is not LDH
		"exam_ple.com",          // an underscore is not LDH
		"example.com&q=1",       // query-injection characters
		"xn--ÿ.com",             // a non-ASCII byte outside the allowlist
	}
	for _, san := range dropped {
		if reg, ok := registrableDomain(san); ok {
			t.Errorf("registrableDomain(%q) = %q, want dropped", san, reg)
		}
	}
	if n := FanOut(dropped); n != 0 {
		t.Errorf("FanOut over droppable SANs = %d, want 0", n)
	}
	// A dropped SAN beside a good one leaves exactly the good one.
	if n := FanOut([]string{"192.0.2.1", "co.uk", "example.com"}); n != 1 {
		t.Errorf("FanOut = %d, want 1", n)
	}

	// A name whose top label holds a letter still passes, so the numeric-top
	// drop takes no real domain with it.
	for _, san := range []string{"1.2.3.com", "123.example.com", "99designs.com"} {
		if _, ok := registrableDomain(san); !ok {
			t.Errorf("registrableDomain(%q) dropped, want a registrable domain", san)
		}
	}
}

// TestDottedNumericSANsCannotInflateTheCount: a SAN set is third-party wire
// content and Go's x509 parser does not police a dNSName, so a bundle of
// dotted-numeric names must raise the fan-out by zero. Without the numeric-top
// drop the PSL's wildcard rule hands each one back a distinct nonsense eTLD+1
// and the count that gates the veto inflates on demand.
func TestDottedNumericSANsCannotInflateTheCount(t *testing.T) {
	stuffed := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		stuffed = append(stuffed, fmt.Sprintf("10.0.%d.%d", i/256, i%256))
	}
	if n := FanOut(stuffed); n != 0 {
		t.Errorf("FanOut over 200 dotted-numeric SANs = %d, want 0", n)
	}
	if SharedEdge(stuffed) {
		t.Error("SharedEdge over 200 dotted-numeric SANs = true, want false")
	}
}

// TestSharedEdgeIsTrueAt100AndFalseAt99: the boundary the #955 amendment fixed,
// and the boundary #986's golden-corpus rows will lock.
func TestSharedEdgeIsTrueAt100AndFalseAt99(t *testing.T) {
	at99 := unrelatedDomains(99)
	if n := FanOut(at99); n != 99 {
		t.Fatalf("fixture reduced to %d domains, want 99", n)
	}
	if SharedEdge(at99) {
		t.Error("SharedEdge at a count of 99 = true, want false")
	}

	at100 := unrelatedDomains(100)
	if n := FanOut(at100); n != 100 {
		t.Fatalf("fixture reduced to %d domains, want 100", n)
	}
	if !SharedEdge(at100) {
		t.Error("SharedEdge at a count of 100 = false, want true")
	}

	// Well above the threshold stays true; an empty SAN set stays false. The
	// empty case is the one a zero-valued threshold would get wrong, and it is
	// the unsafe direction: an edge that presented no identity at all must not
	// read as shared.
	if !SharedEdge(unrelatedDomains(4000)) {
		t.Error("SharedEdge far above the threshold = false, want true")
	}
	if SharedEdge(nil) {
		t.Error("SharedEdge over an empty SAN set = true, want false")
	}
}

// TestCountAppliesNoRelatednessFilter: 100 registrable domains that all "look
// like one brand" count as 100, exactly as 100 unrelated ones do. Clustering
// them would be an ownership heuristic in disguise, and ADR-0129 §1 refuses
// ownership as the discriminator.
func TestCountAppliesNoRelatednessFilter(t *testing.T) {
	oneBrand := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		oneBrand = append(oneBrand, fmt.Sprintf("www.acmecorp%02d.com", i))
	}
	if n := FanOut(oneBrand); n != 100 {
		t.Errorf("FanOut over one brand's 100 domains = %d, want 100", n)
	}
	if !SharedEdge(oneBrand) {
		t.Error("SharedEdge over one brand's 100 domains = false, want true")
	}

	// The same registrable domain under many different TLDs is many domains.
	// The eTLD+1 is the whole of the identity the count reads.
	tlds := []string{"acme.com", "acme.net", "acme.org", "acme.co.uk", "acme.de"}
	if n := FanOut(tlds); n != len(tlds) {
		t.Errorf("FanOut over one label under %d suffixes = %d, want %d", len(tlds), n, len(tlds))
	}
}

// TestParamsDigestStableAndSensitive: the digest is content-addressed, so a
// threshold move breaks #986's A6 lock.
func TestParamsDigestStableAndSensitive(t *testing.T) {
	a := DefaultParams()
	if a.Digest() != DefaultParams().Digest() {
		t.Error("digest is not stable across calls")
	}
	if !strings.HasPrefix(a.Digest(), "sha256:") {
		t.Errorf("digest = %q, want a sha256: prefix", a.Digest())
	}

	b := DefaultParams()
	b.SharedEdgeThreshold = 99
	if a.Digest() == b.Digest() {
		t.Error("digest did not move when the shared-edge threshold changed")
	}

	// A PSL update is a Break in the derivation too (the #954 amendment), so it
	// moves the digest as the threshold does.
	c := DefaultParams()
	c.PublicSuffixList = c.PublicSuffixList + " (a later revision)"
	if a.Digest() == c.Digest() {
		t.Error("digest did not move when the Public Suffix List changed")
	}
	if DefaultParams().PublicSuffixList == "" {
		t.Error("DefaultParams names no Public Suffix List revision")
	}
}

// TestThresholdIsNotOperatorConfigurable: the threshold is project-authored and
// fixed at the release (ADR-0008, ADR-0129 §3). Two proofs. The value the
// shipped params carry IS the constant, so no other source feeds it; and the
// package imports nothing an operator dial could arrive through — no
// environment, no settings, no database. The `var _ [SharedEdgeThreshold]struct{}`
// in fanout.go carries the third proof at compile time.
func TestThresholdIsNotOperatorConfigurable(t *testing.T) {
	if DefaultParams().SharedEdgeThreshold != SharedEdgeThreshold {
		t.Errorf("DefaultParams threshold = %d, want the constant %d",
			DefaultParams().SharedEdgeThreshold, SharedEdgeThreshold)
	}

	// Params RECORDS the threshold; it never supplies it. A method on Params
	// deciding the boolean would let a zero value compare `count >= 0` and veto
	// every edge, so the type must carry no such method.
	if _, found := reflect.TypeOf(Params{}).MethodByName("SharedEdge"); found {
		t.Error("Params has a SharedEdge method — the veto must read the constant, never a caller's value")
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse the custody package: %v", err)
	}
	// Any import through which an operator-set value could reach the
	// derivation. `os` covers the environment variable; the rest cover a
	// settings row and the database that would hold one.
	forbidden := []string{`"os"`, "/internal/env", "/internal/db", "/internal/pgdb", "database/sql"}
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			for _, imp := range file.Imports {
				for _, bad := range forbidden {
					if strings.Contains(imp.Path.Value, bad) {
						t.Errorf("%s imports %s — the threshold must be reachable from no operator setting",
							path, imp.Path.Value)
					}
				}
			}
		}
	}
}

func unrelatedDomains(n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("www.tenant-%04d.example%04d.com", i, i))
	}
	return out
}
