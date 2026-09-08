package proposer

import (
	"context"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
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

type errDoer struct{ err error }

func (d *errDoer) Do(*http.Request) (*http.Response, error) { return nil, d.err }

type statusDoer struct{ code int }

func (d *statusDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: d.code, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
}

type pagingDoer struct {
	pages     []string
	delegated string
	searches  []string
}

func (d *pagingDoer) Do(req *http.Request) (*http.Response, error) {
	body := d.delegated
	if strings.Contains(req.URL.String(), "/search/") {
		if len(d.searches) >= len(d.pages) {
			return nil, errors.New("the search was asked for more pages than the server holds")
		}
		body = d.pages[len(d.searches)]
		d.searches = append(d.searches, req.URL.String())
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

type splitDoer struct {
	search string
	rest   Doer
	calls  []string
}

func (d *splitDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls = append(d.calls, req.URL.String())
	if strings.Contains(req.URL.String(), "/search/") {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(d.search)), Header: make(http.Header)}, nil
	}
	return d.rest.Do(req)
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
		"/search/": `{"totalCount":1,"pageInfo":{"first":5000,"offset":0,"hasNextPage":false},"errors":null,` +
			`"data":[{"opaqueId":"ZA-HOLDER-1_AFRINIC","orgName":"Some AFRINIC Org","source":"AFRINIC"}]}`,
		"delegated-afrinic-extended-latest": strings.Join([]string{
			"2.3|afrinic|20240101|3|19830101|20240101|+0000",
			"afrinic|ZA|ipv4|196.1.0.0|512|20010101|allocated|ZA-HOLDER-1",
			"afrinic|ZA|ipv6|2c0f:f000::|32|20060101|allocated|ZA-HOLDER-1",
			"afrinic|NG|ipv4|41.0.0.0|256|20080101|allocated|OTHER-HOLDER",
		}, "\n"),
	}}
	c := NewCAIDA(doer, SlugAFRINIC, "afrinic", "https://api.data.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic")

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

func TestCAIDAProposesFromLiveAS2orgSearchCapture(t *testing.T) {
	// A live api.data.caida.org/as2org/v1/search/?name=Seacom capture, 2026-09-07 (#1616).
	doer := &fakeDoer{routes: map[string]string{
		"/search/":    loadFixture(t, "caida_search_seacom.json"),
		"/asns/37476": loadFixture(t, "caida_asns_unknown.json"),
		"delegated-afrinic-extended-latest": strings.Join([]string{
			"afrinic|MU|ipv4|41.87.96.0|8192|20100816|allocated|F365C741",
			"afrinic|ZA|ipv4|41.78.4.0|1024|20090904|allocated|F3670C40",
			"afrinic|ZA|ipv4|41.216.128.0|4096|20081014|allocated|F36F76E3",
			"afrinic|ZA|ipv6|2c0f:fe20::|32|20100426|allocated|F36F76E3",
			"afrinic|NG|ipv4|102.0.0.0|256|20200101|allocated|a220b89023b5759a8c2bb70cee6da584",
		}, "\n"),
	}}
	c := NewCAIDA(doer, SlugAFRINIC, "afrinic", "https://api.data.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic")

	cands, err := c.Propose(context.Background(), "Seacom")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, cd := range cands {
		got[cd.Scope.String()] = true
	}
	for _, want := range []string{"41.87.96.0/19", "41.78.4.0/22", "41.216.128.0/20", "2c0f:fe20::/32"} {
		if !got[want] {
			t.Errorf("the live capture's opaqueId did not reach %s: %v", want, got)
		}
	}
	if got["102.0.0.0/24"] {
		t.Errorf("an ARIN record's opaqueId joined the AFRINIC file: %v", got)
	}
	for _, call := range doer.calls {
		if strings.Contains(call, "org2ids") {
			t.Errorf("the retired org2ids path was requested: %s", call)
		}
	}
}

func TestCAIDATransportFailureIsNotAnEmptyResult(t *testing.T) {
	// A swapped constant alone would decode a live body to nil and read as absence (#1616).
	base := "https://api.data.caida.org/as2org/v1"
	del := "https://ftp.afrinic.net/stats/afrinic"

	loud := map[string]Doer{
		"transport error":  &errDoer{err: errors.New("dial tcp: no such host")},
		"non-200":          &statusDoer{code: http.StatusInternalServerError},
		"org2ids envelope": &fakeDoer{routes: map[string]string{"/search/": `{"opaque_ids":["ZA-HOLDER-1"]}`}},
		"html body":        &fakeDoer{routes: map[string]string{"/search/": "<html>An Error Occurred</html>"}},
		"reported errors":  &fakeDoer{routes: map[string]string{"/search/": `{"totalCount":0,"pageInfo":{"hasNextPage":false},"errors":["backend down"],"data":[]}`}},
		"short page":       &fakeDoer{routes: map[string]string{"/search/": `{"totalCount":9,"pageInfo":{"hasNextPage":false},"errors":null,"data":[]}`}},
		"no join key": &fakeDoer{routes: map[string]string{
			"/search/": `{"totalCount":1,"pageInfo":{"hasNextPage":false},"errors":null,` +
				`"data":[{"orgName":"Seacom Ltd","source":"AFRINIC","members":["37100"]}]}`,
			"/asns/37100": loadFixture(t, "caida_asns_unknown.json"),
		}},
	}
	for name, doer := range loud {
		t.Run(name, func(t *testing.T) {
			cands, err := NewCAIDA(doer, SlugAFRINIC, "afrinic", base, del).Propose(context.Background(), "Seacom")
			if err == nil {
				t.Fatalf("a failed request read as %d candidates, so it is indistinguishable from no holder (#50)", len(cands))
			}
			if cands != nil {
				t.Errorf("candidates alongside an error: %+v", cands)
			}
		})
	}

	// The live empty envelope is the one case that is an absence of holders, not a failure.
	empty := &fakeDoer{routes: map[string]string{"/search/": loadFixture(t, "caida_search_no_match.json")}}
	cands, err := NewCAIDA(empty, SlugAFRINIC, "afrinic", base, del).Propose(context.Background(), "Airtel Kenya")
	if err != nil {
		t.Fatalf("an empty data array is no holder matched, never an error: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("an empty data array yielded %+v", cands)
	}
	for _, call := range empty.calls {
		if strings.Contains(call, "delegated-") {
			t.Errorf("the delegated-stats file was fetched with no join key: %s", call)
		}
	}
}

func TestCAIDARecoversTheJoinKeyThroughTheASNsLeg(t *testing.T) {
	// Live captures, 2026-09-08: search/?name=Telkom Kenya answers 71 organisation records and no
	// opaqueId, and asns/30994 and asns/12455 each carry F367736D_AFRINIC (#1634).
	doer := &fakeDoer{routes: map[string]string{
		"/search/":    loadFixture(t, "caida_search_telkom_kenya.json"),
		"/asns/30994": loadFixture(t, "caida_asns_30994.json"),
		"/asns/12455": loadFixture(t, "caida_asns_12455.json"),
		"delegated-afrinic-extended-latest": strings.Join([]string{
			"afrinic|KE|ipv4|41.215.128.0|4096|20080512|allocated|F367736D",
			"afrinic|KE|ipv6|2c0f:fe38::|32|20100301|allocated|F367736D",
			"afrinic|KE|ipv4|41.203.208.0|1024|20080101|allocated|F3682104",
		}, "\n"),
	}}
	c := NewCAIDA(doer, SlugAFRINIC, "afrinic", "https://api.data.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic")

	cands, err := c.Propose(context.Background(), "Telkom Kenya")
	if err != nil {
		t.Fatalf("an org CAIDA holds under a member ASN's key still errored: %v", err)
	}
	got := map[string]bool{}
	for _, cd := range cands {
		got[cd.Scope.String()] = true
	}
	for _, want := range []string{"41.215.128.0/20", "2c0f:fe38::/32"} {
		if !got[want] {
			t.Errorf("the key the asns leg recovered did not reach %s: %v", want, got)
		}
	}
	if got["41.203.208.0/22"] {
		t.Errorf("a stranger's opaqueId joined the file: %v", got)
	}
	asns := map[string]int{}
	for _, call := range doer.calls {
		if i := strings.Index(call, "/asns/"); i >= 0 {
			asns[call[i+len("/asns/"):]]++
		}
	}
	if asns["30994"] != 1 || asns["12455"] != 1 || len(asns) != 2 {
		t.Errorf("the asns leg should look each distinct member up once, got %v", asns)
	}
}

func TestCAIDASkipsTheASNsLegForAnASNAKeyedRecordAlreadyCarries(t *testing.T) {
	// name=Safaricom names 3 member ASNs on 115 unkeyed org records, and all 3 sit on keyed ASN
	// records in the same response, so the measured second leg costs nothing there (#1634).
	doer := &fakeDoer{routes: map[string]string{
		"/search/": `{"totalCount":2,"pageInfo":{"hasNextPage":false},"errors":null,"data":[` +
			`{"asn":"37061","asnName":"Safaricom","orgName":"Safaricom Limited","source":"AFRINIC","opaqueId":"F3682104_AFRINIC"},` +
			`{"orgId":"33771","orgName":"Safaricom Limited","source":"AFRINIC","members":["37061"]}]}`,
		"/asns/":                            `{"totalCount":1,"pageInfo":{"hasNextPage":false},"errors":null,"data":[{"asn":"37061","orgName":"Safaricom Limited","source":"AFRINIC","opaqueId":"F3682104_AFRINIC"}]}`,
		"delegated-afrinic-extended-latest": "afrinic|KE|ipv4|41.203.208.0|1024|20080101|allocated|F3682104",
	}}
	c := NewCAIDA(doer, SlugAFRINIC, "afrinic", "https://api.data.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic")
	cands, err := c.Propose(context.Background(), "Safaricom")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 {
		t.Fatalf("want one candidate, got %+v", cands)
	}
	for _, call := range doer.calls {
		if strings.Contains(call, "/asns/") {
			t.Errorf("an ASN a keyed record already carries was looked up again: %s", call)
		}
	}
}

func TestCAIDAASNsLegFailureIsNotAnEmptyResult(t *testing.T) {
	// The recovery leg must not turn a transport failure into no holder matched (#1634, #50).
	base := "https://api.data.caida.org/as2org/v1"
	del := "https://ftp.afrinic.net/stats/afrinic"
	search := loadFixture(t, "caida_search_telkom_kenya.json")
	keyless := `{"totalCount":1,"pageInfo":{"hasNextPage":false},"errors":null,` +
		`"data":[{"asn":"30994","asnName":"Galileo-Kenya","orgName":"Kenyan Post & Telecommunications Company / Telkom Kenya Ltd","source":"AFRINIC"}]}`

	loud := map[string]Doer{
		"transport error":  &errDoer{err: errors.New("dial tcp: connection reset")},
		"non-200":          &statusDoer{code: http.StatusBadGateway},
		"org2ids envelope": &fakeDoer{routes: map[string]string{"/asns/": `{"opaque_ids":["F367736D"]}`}},
		"html body":        &fakeDoer{routes: map[string]string{"/asns/": "<html>An Error Occurred</html>"}},
		"reported errors":  &fakeDoer{routes: map[string]string{"/asns/": `{"totalCount":0,"pageInfo":{"hasNextPage":false},"errors":["backend down"],"data":[]}`}},
		"short page":       &fakeDoer{routes: map[string]string{"/asns/": `{"totalCount":1,"pageInfo":{"hasNextPage":false},"errors":null,"data":[]}`}},
		"still no key":     &fakeDoer{routes: map[string]string{"/asns/": keyless}},
		"unknown asn":      &fakeDoer{routes: map[string]string{"/asns/": loadFixture(t, "caida_asns_unknown.json")}},
	}
	for name, rest := range loud {
		t.Run(name, func(t *testing.T) {
			doer := &splitDoer{search: search, rest: rest}
			cands, err := NewCAIDA(doer, SlugAFRINIC, "afrinic", base, del).Propose(context.Background(), "Telkom Kenya")
			if err == nil {
				t.Fatalf("a failed asns leg read as %d candidates, so it is indistinguishable from no holder (#50)", len(cands))
			}
			if cands != nil {
				t.Errorf("candidates alongside an error: %+v", cands)
			}
			gap := name == "still no key" || name == "unknown asn"
			if errors.Is(err, ErrNoJoinKey) != gap {
				t.Errorf("ErrNoJoinKey = %v for %s, want %v: %v", !gap, name, gap, err)
			}
			for _, call := range doer.calls {
				if strings.Contains(call, "delegated-") {
					t.Errorf("the delegated-stats file was fetched after a failed asns leg: %s", call)
				}
			}
		})
	}

	var members []string
	// A search that names more member ASNs than the leg may look up fails before the first lookup.
	for i := range caidaASNLegMaxLookups + 1 {
		members = append(members, strconv.Itoa(60000+i))
	}
	wide := &fakeDoer{routes: map[string]string{
		"/search/": `{"totalCount":1,"pageInfo":{"hasNextPage":false},"errors":null,` +
			`"data":[{"orgName":"Wide Org","source":"AFRINIC","members":["` + strings.Join(members, `","`) + `"]}]}`,
		"/asns/": `{"totalCount":1,"pageInfo":{"hasNextPage":false},"errors":null,"data":[{"asn":"60000","orgName":"Wide Org","source":"AFRINIC","opaqueId":"WIDE_AFRINIC"}]}`,
	}}
	cands, err := NewCAIDA(wide, SlugAFRINIC, "afrinic", base, del).Propose(context.Background(), "Wide Org")
	if err == nil {
		t.Fatalf("a search over the asns cap read as %d candidates", len(cands))
	}
	for _, call := range wide.calls {
		if strings.Contains(call, "/asns/") {
			t.Errorf("the cap did not stop the first lookup: %s", call)
		}
	}
}

func TestCAIDAPagesUntilTheServerHasSentEveryRowItCounted(t *testing.T) {
	doer := &pagingDoer{pages: []string{
		`{"totalCount":2,"pageInfo":{"hasNextPage":true},"errors":null,` +
			`"data":[{"opaqueId":"ONE_AFRINIC","orgName":"Paged Org","source":"AFRINIC"}]}`,
		`{"totalCount":2,"pageInfo":{"hasNextPage":false},"errors":null,` +
			`"data":[{"opaqueId":"TWO_AFRINIC","orgName":"Paged Org","source":"AFRINIC"}]}`,
	}, delegated: strings.Join([]string{
		"afrinic|ZA|ipv4|196.1.0.0|256|20010101|allocated|ONE",
		"afrinic|ZA|ipv4|196.2.0.0|256|20010101|allocated|TWO",
	}, "\n")}

	c := NewCAIDA(doer, SlugAFRINIC, "afrinic", "https://api.data.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic")
	cands, err := c.Propose(context.Background(), "Paged Org")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("a second page was dropped: %+v", cands)
	}
	if !strings.Contains(doer.searches[1], "offset=1") {
		t.Errorf("the second page did not carry the rows already read: %q", doer.searches[1])
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
		"/search/": `{"totalCount":1,"pageInfo":{"hasNextPage":false},"errors":null,` +
			`"data":[{"opaqueId":"X_AFRINIC","orgName":"Org","source":"AFRINIC"}]}`,
		"delegated-afrinic-extended-latest": "afrinic|ZA|ipv4|196.1.0.0|256|20010101|allocated|X",
	}}
	reg := NewRegistry(
		NewARIN(arinDoer, "https://rdap.arin.net/registry"),
		NewCAIDA(caidaDoer, SlugAFRINIC, "afrinic", "https://api.data.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic"),
	)

	cands, _, err := reg.Propose(context.Background(), "Org", map[string]bool{SlugARIN: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].SourceSlug != SlugARIN {
		t.Fatalf("expected one ARIN candidate, got %+v", cands)
	}
	if len(caidaDoer.calls) != 0 {
		t.Errorf("disabled AFRINIC source was queried: %v", caidaDoer.calls)
	}

	cands, _, err = reg.Propose(context.Background(), "Org", map[string]bool{SlugARIN: true, SlugAFRINIC: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("expected two candidates with both enabled, got %+v", cands)
	}
}

func TestDefaultRegistryCAIDABaseIsThePublishedAS2orgHost(t *testing.T) {
	const live = "https://api.data.caida.org/as2org/v1"
	var seen int
	for _, s := range DefaultRegistry(&fakeDoer{}).sources {
		c, ok := s.(*CAIDA)
		if !ok {
			continue
		}
		seen++
		if c.caidaBase != live {
			t.Errorf("%s caidaBase = %q, want %q — api.caida.org is NXDOMAIN and no host serves org2ids "+
				"(ADR-0227, #1616)", c.Slug(), c.caidaBase, live)
		}
	}
	if seen != 2 {
		t.Fatalf("default registry holds %d CAIDA sources, want 2", seen)
	}
}
