package proposer

import (
	"context"
	"io"
	"math/big"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeDoer struct {
	routes map[string]string
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

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func TestARINProposesFromLiveRDAPCapture(t *testing.T) {
	// A live ARIN RDAP capture, 2026-08-26: org, SWIP customer, and POC in one search (#611).
	doer := &fakeDoer{routes: map[string]string{
		"entities?fn=":     loadFixture(t, "hurricane_search.json"),
		"entity/HURRIC-1":  loadFixture(t, "hurricane_entity_HURRIC-1.json"),
		"entity/C01839743": loadFixture(t, "hurricane_entity_C01839743.json"),
		"entity/ZH17-ARIN": loadFixture(t, "hurricane_entity_ZH17-ARIN.json"),
	}}
	a := NewARIN(doer, "https://rdap.arin.net/registry")

	cands, err := a.Propose(context.Background(), "Hurricane Electric")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("candidates = %d, want 2 (POC contributes none): %+v", len(cands), cands)
	}
	byKind := map[string]Candidate{}
	for _, c := range cands {
		byKind[c.RecordKind] = c
		if c.SourceSlug != SlugARIN {
			t.Errorf("candidate slug = %q, want %q", c.SourceSlug, SlugARIN)
		}
		if c.OrgName != "Hurricane Electric" {
			t.Errorf("candidate OrgName = %q, want %q", c.OrgName, "Hurricane Electric")
		}
	}
	if d := byKind[RecordRIRDelegation]; d.Scope.String() != "216.218.130.128/29" {
		t.Errorf("org delegation candidate wrong: %+v", d)
	}
	if r := byKind[RecordCompelledReassignment]; r.Scope.String() != "216.218.130.224/27" {
		t.Errorf("customer reassignment candidate wrong: %+v", r)
	}
	for _, c := range doer.calls {
		if strings.Contains(c, "entity/ZH17-ARIN") {
			t.Errorf("POC entity was fetched but should be skipped: %s", c)
		}
	}
}

func TestARINReportsInterruptionRatherThanPartial(t *testing.T) {
	doer := &fakeDoer{routes: map[string]string{
		"entities?fn=":    loadFixture(t, "hurricane_search.json"),
		"entity/HURRIC-1": loadFixture(t, "hurricane_entity_HURRIC-1.json"),
	}}
	a := NewARIN(doer, "https://rdap.arin.net/registry")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := a.Propose(ctx, "Hurricane Electric"); err == nil {
		t.Fatal("a cancelled walk must report an error, not a silent partial result")
	}
}

func TestARINNoMatchIsNotAnError(t *testing.T) {
	// A 404 is a clean no-match, never a path error the operator sees (#611).
	doer := &fakeDoer{routes: map[string]string{}}
	a := NewARIN(doer, "https://rdap.arin.net/registry")

	cands, err := a.Propose(context.Background(), "No Such Org 12345")
	if err != nil {
		t.Fatalf("a no-match must not error: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("a no-match must yield no candidates, got %+v", cands)
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
		"entities?fn=":    `{"entitySearchResults":[{"handle":"NETORG-1","vcardArray":["vcard",[["version",{},"text","4.0"],["fn",{},"text","Org"]]]}]}`,
		"entity/NETORG-1": `{"handle":"NETORG-1","vcardArray":["vcard",[["fn",{},"text","Org"]]],"networks":[{"cidr0_cidrs":[{"v4prefix":"203.0.113.0","length":24}]}]}`,
	}}
	caidaDoer := &fakeDoer{routes: map[string]string{
		"org2ids":                           `{"opaque_ids":["X"]}`,
		"delegated-afrinic-extended-latest": "afrinic|ZA|ipv4|196.1.0.0|256|20010101|allocated|X",
	}}
	reg := NewRegistry(
		NewARIN(arinDoer, "https://rdap.arin.net/registry"),
		NewCAIDA(caidaDoer, SlugAFRINIC, "afrinic", "https://api.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic"),
	)

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

	cands, err = reg.Propose(context.Background(), "Org", map[string]bool{SlugARIN: true, SlugAFRINIC: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("expected two candidates with both enabled, got %+v", cands)
	}
}
