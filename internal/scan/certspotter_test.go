package scan

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestCertSpotterURL(t *testing.T) {
	// The decoder reads dns_names, so the query must expand them (ct-source-replacement.md §2.6).
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

	next := CertSpotterURL("example.com", "42")
	if got := mustQuery(t, next).Get("after"); got != "42" {
		t.Errorf("after = %q, want 42", got)
	}

	// Percent-encoding the domain stops an injection character smuggling a second parameter (#774).
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
	t.Run("non-array body is an error, never empty", func(t *testing.T) {
		for _, body := range []string{"", "   ", "<html>429</html>", "not json"} {
			if _, err := ParseCertSpotterPage([]byte(body)); err == nil {
				t.Errorf("ParseCertSpotterPage(%q) = nil error, want a parse error", body)
			}
		}
	})
}

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

func TestCertSpotterSourceDisplayName(t *testing.T) {
	// The live stream shows this label to an operator on a non-200, so it is the product name (#780).
	if got := CertSpotterCTSource().DisplayName(); !strings.Contains(got, "Cert Spotter") {
		t.Errorf("DisplayName = %q, want it to name Cert Spotter", got)
	}
}
