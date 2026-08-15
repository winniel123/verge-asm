package proposer

import (
	"context"
	"io"
	"math/big"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

// fakeDoer answers requests from an in-memory map keyed by a substring of the
// URL, so no test touches the network. A request whose URL matches no route
// fails loudly, which is what proves the paths never reach a real endpoint.
type fakeDoer struct {
	routes map[string]string // url substring -> body
	calls  []string
}

func (d *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls = append(d.calls, req.URL.String())
	for frag, body := range d.routes {
		if strings.Contains(req.URL.String(), frag) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}
	}
	return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func TestARINProposesDelegationAndReassignment(t *testing.T) {
	doer := &fakeDoer{routes: map[string]string{
		"entities?fn=": `{"entities":[
			{"handle":"ORG-EX","name":"Example Org","networks":[{"cidr":"203.0.113.0/24"}],
			 "customers":[{"name":"Renter LLC","cidr":"198.51.100.8/29"}]}
		]}`,
	}}
	a := NewARIN(doer, "https://whois.arin.net/rest")

	cands, err := a.Propose(context.Background(), "Example")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("candidates = %d, want 2: %+v", len(cands), cands)
	}
	// The org network is an RIR delegation; the SWIP customer block is a
	// compelled reassignment carrying the customer's own name.
	byKind := map[string]Candidate{}
	for _, c := range cands {
		byKind[c.RecordKind] = c
	}
	if d := byKind[RecordRIRDelegation]; d.Scope.String() != "203.0.113.0/24" || d.OrgName != "Example Org" {
		t.Errorf("delegation candidate wrong: %+v", d)
	}
	if r := byKind[RecordCompelledReassignment]; r.Scope.String() != "198.51.100.8/29" || r.OrgName != "Renter LLC" {
		t.Errorf("reassignment candidate wrong: %+v", r)
	}
	for _, c := range cands {
		if c.SourceSlug != SlugARIN {
			t.Errorf("candidate slug = %q, want %q", c.SourceSlug, SlugARIN)
		}
	}
}

func TestCAIDAJoinsOrgIDsToDelegatedStats(t *testing.T) {
	doer := &fakeDoer{routes: map[string]string{
		"org2ids": `{"opaque_ids":["ZA-HOLDER-1"]}`,
		"delegated-afrinic-extended-latest": strings.Join([]string{
			"2.3|afrinic|20240101|3|19830101|20240101|+0000",
			"afrinic|ZA|ipv4|196.1.0.0|512|20010101|allocated|ZA-HOLDER-1",
			"afrinic|ZA|ipv6|2c0f:f000::|32|20060101|allocated|ZA-HOLDER-1",
			"afrinic|NG|ipv4|41.0.0.0|256|20080101|allocated|OTHER-HOLDER",
		}, "\n"),
	}}
	c := NewCAIDA(doer, SlugAFRINIC, "afrinic", "https://api.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic")

	cands, err := c.Propose(context.Background(), "Some AFRINIC Org")
	if err != nil {
		t.Fatal(err)
	}
	// 512 addresses at 196.1.0.0 is exactly a /23; the ipv6 row is a /32; the
	// row under a non-matching opaque id is excluded by the join.
	got := map[string]bool{}
	for _, cd := range cands {
		got[cd.Scope.String()] = true
		if cd.RecordKind != RecordRIRDelegation {
			t.Errorf("delegated-stats row should be an rir-delegation, got %q", cd.RecordKind)
		}
		if cd.SourceSlug != SlugAFRINIC {
			t.Errorf("slug = %q, want %q", cd.SourceSlug, SlugAFRINIC)
		}
	}
	if !got["196.1.0.0/23"] {
		t.Errorf("missing /23 from 512-address ipv4 row: %v", got)
	}
	if !got["2c0f:f000::/32"] {
		t.Errorf("missing ipv6 /32: %v", got)
	}
	if got["41.0.0.0/24"] {
		t.Errorf("row under a non-matching opaque id leaked into candidates: %v", got)
	}
}

func TestRangeToPrefixesDecomposesNonPowerOfTwo(t *testing.T) {
	// 768 addresses from 196.1.0.0 decomposes into the minimal aligned set — a
	// /23 (512) then a /24 (256) — never a single invented prefix.
	start := netip.MustParseAddr("196.1.0.0")
	ps, err := rangeToPrefixes(start, big.NewInt(768))
	if err != nil {
		t.Fatal(err)
	}
	var total uint64
	for _, p := range ps {
		total += 1 << uint(p.Addr().BitLen()-p.Bits())
	}
	if total != 768 {
		t.Fatalf("prefixes cover %d addresses, want 768: %v", total, ps)
	}
	want := []string{"196.1.0.0/23", "196.1.2.0/24"}
	if len(ps) != len(want) {
		t.Fatalf("blocks = %v, want %v", ps, want)
	}
	for i, w := range want {
		if ps[i].String() != w {
			t.Errorf("block %d = %s, want %s", i, ps[i], w)
		}
	}
}

func TestRegistryRunsOnlyEnabledSources(t *testing.T) {
	arinDoer := &fakeDoer{routes: map[string]string{
		"entities?fn=": `{"entities":[{"handle":"O","name":"Org","networks":[{"cidr":"203.0.113.0/24"}]}]}`,
	}}
	caidaDoer := &fakeDoer{routes: map[string]string{
		"org2ids":                           `{"opaque_ids":["X"]}`,
		"delegated-afrinic-extended-latest": "afrinic|ZA|ipv4|196.1.0.0|256|20010101|allocated|X",
	}}
	reg := NewRegistry(
		NewARIN(arinDoer, "https://whois.arin.net/rest"),
		NewCAIDA(caidaDoer, SlugAFRINIC, "afrinic", "https://api.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic"),
	)

	// Only ARIN enabled: AFRINIC must not be queried at all.
	cands, err := reg.Propose(context.Background(), "Org", map[string]bool{SlugARIN: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].SourceSlug != SlugARIN {
		t.Fatalf("expected one ARIN candidate, got %+v", cands)
	}
	if len(caidaDoer.calls) != 0 {
		t.Errorf("disabled AFRINIC source was queried: %v", caidaDoer.calls)
	}

	// Both enabled: both answer.
	cands, err = reg.Propose(context.Background(), "Org", map[string]bool{SlugARIN: true, SlugAFRINIC: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("expected two candidates with both enabled, got %+v", cands)
	}
}
