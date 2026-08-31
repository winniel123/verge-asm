package scan

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// The query is the full-domain form the admission mapping needs (research §3.1):
// include_subdomains=true, expand=dns_names and expand=issuer, the domain, and the
// after cursor only once paging has started.
func TestCertSpotterURL(t *testing.T) {
	first := CertSpotterURL("example.com", "")
	u, err := url.Parse(first)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Host != "api.certspotter.com" || u.Path != "/v1/issuances" {
		t.Errorf("endpoint = %q, want api.certspotter.com/v1/issuances", first)
	}
	q := u.Query()
	if q.Get("domain") != "example.com" {
		t.Errorf("domain = %q", q.Get("domain"))
	}
	if q.Get("include_subdomains") != "true" {
		t.Errorf("include_subdomains = %q, want true", q.Get("include_subdomains"))
	}
	if exp := q["expand"]; !reflect.DeepEqual(exp, []string{"dns_names", "issuer"}) {
		t.Errorf("expand = %v, want [dns_names issuer]", exp)
	}
	if _, ok := q["after"]; ok {
		t.Errorf("first page must carry no after cursor: %q", first)
	}

	// Once paging has started the cursor rides as after=<id>.
	next := CertSpotterURL("example.com", "42")
	if got := mustQuery(t, next).Get("after"); got != "42" {
		t.Errorf("after = %q, want 42", got)
	}

	// Belt-and-braces: the domain is percent-encoded, so an injection character
	// cannot smuggle a second parameter (mirrors the crt.sh guard, #774).
	inj := CertSpotterURL("example.com&include_subdomains=false", "")
	if mustQuery(t, inj).Get("include_subdomains") != "true" {
		t.Errorf("injection overrode include_subdomains: %q", inj)
	}
}

func mustQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u.Query()
}

func TestParseCertSpotterPage(t *testing.T) {
	t.Run("valid array with whitespace", func(t *testing.T) {
		body := []byte("  [{\"id\":\"7\",\"dns_names\":[\"a.example.com\",\"b.example.com\"]}]\n")
		got, err := ParseCertSpotterPage(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].ID != "7" || len(got[0].DNSNames) != 2 {
			t.Errorf("issuances = %+v", got)
		}
	})
	t.Run("empty array is valid and carries no names", func(t *testing.T) {
		got, err := ParseCertSpotterPage([]byte("[]"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("issuances = %+v, want empty", got)
		}
	})
	// A malformed 200 is not evidence of anything (ADR-0027 §7): a parse error, so
	// the caller treats it as transient rather than as "no certificates".
	t.Run("non-array body is an error, never empty", func(t *testing.T) {
		for _, body := range []string{"", "   ", "<html>429</html>", "not json"} {
			if _, err := ParseCertSpotterPage([]byte(body)); err == nil {
				t.Errorf("ParseCertSpotterPage(%q) = nil error, want a parse error", body)
			}
		}
	})
}

// The next cursor is the highest id on the page, compared as an integer rather than
// by the page's order, so a page returned in any order still advances the cursor to
// its true maximum. An empty page yields "" — the signal to stop paging.
func TestMaxIssuanceID(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []CertSpotterIssuance
		want string
	}{
		{"empty page stops paging", nil, ""},
		{"ascending", []CertSpotterIssuance{{ID: "3"}, {ID: "8"}, {ID: "20"}}, "20"},
		{"unsorted, longer id is larger", []CertSpotterIssuance{{ID: "20"}, {ID: "9"}, {ID: "100"}}, "100"},
		{"missing id never becomes the cursor", []CertSpotterIssuance{{ID: ""}, {ID: "5"}}, "5"},
	} {
		if got := maxIssuanceID(tc.in); got != tc.want {
			t.Errorf("%s: maxIssuanceID = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// The Cert Spotter source decodes a page into the same filtered admission the ct
// path makes for crt.sh: the shared CTAdmitter refuses wildcards (ADR-0060), drops
// out-of-scope co-tenant SANs (ADR-0047), and dedupes. Fed through DecodePage into
// a CTAdmitter, a mixed page must admit exactly the in-scope, non-wildcard names.
func TestCertSpotterSourceAdmitsThroughSharedFilter(t *testing.T) {
	body := []byte(`[
		{"id":"1","dns_names":["example.com","www.example.com"]},
		{"id":"2","dns_names":["*.example.com","example.com"]},
		{"id":"3","dns_names":["api.example.com","shared.example.net"]},
		{"id":"4","dns_names":["WWW.Example.com."]}
	]`)
	src := CertSpotterCTSource()
	if src.Slug() != CertSpotterSource {
		t.Fatalf("slug = %q, want %q", src.Slug(), CertSpotterSource)
	}
	names, next, err := src.DecodePage(body)
	if err != nil {
		t.Fatalf("DecodePage: %v", err)
	}
	if next != "4" {
		t.Errorf("next cursor = %q, want 4 (the highest id)", next)
	}
	adm := NewCTAdmitter("example.com")
	adm.Add(names)
	got := adm.Names()
	want := []string{"api.example.com", "example.com", "www.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("admitted = %v, want %v (wildcard refused, foreign SAN out of scope, case-folded dup)", got, want)
	}
}

// A single-shot decode reaching an empty page reports next="" so the completion
// loop stops, and a well-formed page with names reports its max id so it continues.
func TestCertSpotterSourceEmptyPageStops(t *testing.T) {
	src := CertSpotterCTSource()
	_, next, err := src.DecodePage([]byte("[]"))
	if err != nil {
		t.Fatalf("DecodePage([]): %v", err)
	}
	if next != "" {
		t.Errorf("empty page next = %q, want \"\" (stop paging)", next)
	}
}

// The DisplayName is the operator-facing label surfaced in the live stream on a
// non-200 (#780); it is the product name, not the slug.
func TestCertSpotterSourceDisplayName(t *testing.T) {
	if got := CertSpotterCTSource().DisplayName(); !strings.Contains(got, "Cert Spotter") {
		t.Errorf("DisplayName = %q, want it to name Cert Spotter", got)
	}
}
