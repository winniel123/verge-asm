package custody

import (
	"fmt"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

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
	if n := FanOut([]string{"*.example.com", "example.com", "www.example.com"}); n != 1 {
		t.Errorf("FanOut over one domain's wildcard and names = %d, want 1", n)
	}
}

func TestSANWithNoRegistrableDomainIsDropped(t *testing.T) {
	// An IPv4 spelling is LDH, so the PSL wildcard rule would hand back the nonsense eTLD+1 2.1.
	dropped := []string{
		"192.0.2.1",
		"2001:db8::1",
		"999.999.999.999",
		"1.2.3",
		"3232235777",
		"co.uk",
		"com",
		"localhost",
		"",
		"   ",
		".",
		"*",
		"foo.*.example.com",
		"https://example.com/a",
		"admin@example.com",
		"exa mple.com",
		"exam_ple.com",
		"example.com&q=1",
		"xn--ÿ.com",
	}
	for _, san := range dropped {
		if reg, ok := registrableDomain(san); ok {
			t.Errorf("registrableDomain(%q) = %q, want dropped", san, reg)
		}
	}
	if n := FanOut(dropped); n != 0 {
		t.Errorf("FanOut over droppable SANs = %d, want 0", n)
	}
	if n := FanOut([]string{"192.0.2.1", "co.uk", "example.com"}); n != 1 {
		t.Errorf("FanOut = %d, want 1", n)
	}

	for _, san := range []string{"1.2.3.com", "123.example.com", "99designs.com"} {
		if _, ok := registrableDomain(san); !ok {
			t.Errorf("registrableDomain(%q) dropped, want a registrable domain", san)
		}
	}
}

func TestDottedNumericSANsCannotInflateTheCount(t *testing.T) {
	// A SAN set is third-party wire content, so stuffing must not inflate the count the veto reads.
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

	if !SharedEdge(unrelatedDomains(4000)) {
		t.Error("SharedEdge far above the threshold = false, want true")
	}
	// A zero threshold would read an edge presenting no identity as shared, the unsafe direction.
	if SharedEdge(nil) {
		t.Error("SharedEdge over an empty SAN set = true, want false")
	}
}

func TestCountAppliesNoRelatednessFilter(t *testing.T) {
	// Clustering by brand would be an ownership heuristic, refused as the discriminator (ADR-0129 §1).
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

	tlds := []string{"acme.com", "acme.net", "acme.org", "acme.co.uk", "acme.de"}
	if n := FanOut(tlds); n != len(tlds) {
		t.Errorf("FanOut over one label under %d suffixes = %d, want %d", len(tlds), n, len(tlds))
	}
}

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

	c := DefaultParams()
	c.PublicSuffixList = c.PublicSuffixList + " (a later revision)"
	if a.Digest() == c.Digest() {
		t.Error("digest did not move when the Public Suffix List changed")
	}
	if DefaultParams().PublicSuffixList == "" {
		t.Error("DefaultParams names no Public Suffix List revision")
	}
}

func TestThresholdIsNotOperatorConfigurable(t *testing.T) {
	// The threshold is project-authored, so no operator setting may reach it (ADR-0008, ADR-0129 §3).
	if DefaultParams().SharedEdgeThreshold != SharedEdgeThreshold {
		t.Errorf("DefaultParams threshold = %d, want the constant %d",
			DefaultParams().SharedEdgeThreshold, SharedEdgeThreshold)
	}

	// A SharedEdge method on Params would let a zero value compare count >= 0 and veto every edge.
	if _, found := reflect.TypeOf(Params{}).MethodByName("SharedEdge"); found {
		t.Error("Params has a SharedEdge method — the veto must read the constant, never a caller's value")
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse the custody package: %v", err)
	}
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
