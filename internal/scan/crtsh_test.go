package scan

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

func TestAdmittedNameKeyMatchesResolverKey(t *testing.T) {
	for _, in := range []string{
		"example.com",
		"VPN.Example.COM.",
		"a.b.example.com",
		"MiXeD.Case.Example.Com",
		"trailing.dot.example.com.",
		"İ.EXAMPLE.COM",
		"Ä.example.com",
		"xn--mnchen-3ya.example.com",
	} {
		if got, want := normaliseName(in), resolutionwalk.CanonicalName(in); got != want {
			t.Errorf("normaliseName(%q)=%q but CanonicalName=%q — the admission-citation match (ADR-0107) would silently miss", in, got, want)
		}
	}
	// A crt.sh SAN value may carry surrounding whitespace the resolver never sees (#256).
	for _, ws := range []string{" x.example.com", "x.example.com\t", "  x.example.com  "} {
		if got, want := normaliseName(ws), resolutionwalk.CanonicalName("x.example.com"); got != want {
			t.Errorf("normaliseName(%q)=%q, want %q — surrounding whitespace must fold to the clean resolver key", ws, got, want)
		}
	}
}

func TestAdmittedNamesFiltersAndDedupes(t *testing.T) {
	rows := []CrtshRow{
		{CommonName: "www.example.com", NameValue: "example.com\nwww.example.com"},
		{CommonName: "*.example.com", NameValue: "*.example.com\nexample.com"},
		{CommonName: "", NameValue: "baz*.example.com"},
		{CommonName: "shared.example.net", NameValue: "api.example.com\nshared.example.net"},
		{CommonName: "WWW.Example.com.", NameValue: ""},
	}
	got := AdmittedNames(rows, "example.com")
	want := []string{"api.example.com", "example.com", "www.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AdmittedNames = %v, want %v", got, want)
	}
}

func TestAdmittedNamesWildcardOnlyAdmitsNothing(t *testing.T) {
	rows := []CrtshRow{
		{CommonName: "*.example.com", NameValue: "*.example.com"},
		{CommonName: "*.internal.example.com", NameValue: "*.internal.example.com"},
	}
	if got := AdmittedNames(rows, "example.com"); len(got) != 0 {
		t.Errorf("AdmittedNames = %v, want empty (wildcards admit nothing)", got)
	}
}

func TestAdmittedNamesScopeIsLabelWise(t *testing.T) {
	rows := []CrtshRow{
		{NameValue: "notexample.com\nx.example.com\nexample.com.evil.test"},
	}
	got := AdmittedNames(rows, "example.com")
	want := []string{"x.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AdmittedNames = %v, want %v (notexample.com and example.com.evil.test are outside scope)", got, want)
	}
}

func TestAdmittedNamesAdmitsApex(t *testing.T) {
	rows := []CrtshRow{{CommonName: "example.com", NameValue: "example.com\n*.example.com"}}
	got := AdmittedNames(rows, "example.com")
	want := []string{"example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AdmittedNames = %v, want %v", got, want)
	}
}

func TestAdmittedNamesCapsCardinality(t *testing.T) {
	const over = MaxAdmittedNames + 500
	var b strings.Builder
	for i := 0; i < over; i++ {
		fmt.Fprintf(&b, "h%d.example.com\n", i)
	}
	rows := []CrtshRow{{NameValue: b.String()}}

	got := AdmittedNames(rows, "example.com")
	if len(got) != MaxAdmittedNames {
		t.Fatalf("AdmittedNames admitted %d names from %d in-scope candidates, want capped at %d", len(got), over, MaxAdmittedNames)
	}
	if !sort.StringsAreSorted(got) {
		t.Errorf("capped result is not sorted — AdmittedNames must still return a deterministic sorted set")
	}
	seen := make(map[string]struct{}, len(got))
	for _, n := range got {
		if _, dup := seen[n]; dup {
			t.Fatalf("capped result contains duplicate %q", n)
		}
		seen[n] = struct{}{}
	}
}

func TestParseCrtshRows(t *testing.T) {
	t.Run("valid array with surrounding whitespace", func(t *testing.T) {
		body := []byte("  [{\"common_name\":\"a.example.com\",\"name_value\":\"a.example.com\"}]\n")
		rows, err := ParseCrtshRows(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rows) != 1 || rows[0].CommonName != "a.example.com" {
			t.Errorf("rows = %+v", rows)
		}
	})
	t.Run("empty array is valid and admits nothing", func(t *testing.T) {
		rows, err := ParseCrtshRows([]byte("[]"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rows) != 0 {
			t.Errorf("rows = %+v, want empty", rows)
		}
	})
	t.Run("non-array body is an error, never empty", func(t *testing.T) {
		for _, body := range []string{"", "   ", "<html>404</html>", "not json"} {
			if _, err := ParseCrtshRows([]byte(body)); err == nil {
				t.Errorf("ParseCrtshRows(%q) = nil error, want a parse error", body)
			}
		}
	})
}

func TestCrtshURL(t *testing.T) {
	got := CrtshURL("example.com")
	want := "https://crt.sh/?q=%25.example.com&output=json"
	if got != want {
		t.Errorf("CrtshURL = %q, want %q", got, want)
	}

	// The name validator is the primary guard against query injection; this is depth (#774).
	inj := CrtshURL("example.com&output=text")
	if strings.Contains(inj, "example.com&output=text") {
		t.Errorf("CrtshURL did not encode injection chars: %q", inj)
	}
	if !strings.HasSuffix(inj, "&output=json") {
		t.Errorf("CrtshURL lost its output=json param: %q", inj)
	}
}

func TestCTScopeRoundTrip(t *testing.T) {
	j := CTJob{ScanID: 7, SeedID: 42, Domain: "example.com"}
	spec, err := j.JobSpec("scan:7:seed:42")
	if err != nil {
		t.Fatalf("JobSpec: %v", err)
	}
	if spec.Kind != CTKind {
		t.Errorf("kind = %q, want %q", spec.Kind, CTKind)
	}
	got, err := CTScopeFromSpec(spec.Scope)
	if err != nil {
		t.Fatalf("CTScopeFromSpec: %v", err)
	}
	if got.Domain != "example.com" || got.SeedID != 42 {
		t.Errorf("round-trip = %+v, want {42 example.com}", got)
	}
}

func TestCTAttemptedVsEmptyScope(t *testing.T) {
	attempted, err := (CTJob{Domain: "example.com"}).AttemptedScope()
	if err != nil {
		t.Fatal(err)
	}
	if string(attempted) != `{"domain":"example.com"}` {
		t.Errorf("attempted scope = %s, want the queried domain", attempted)
	}
	empty, err := EmptyCTScope()
	if err != nil {
		t.Fatal(err)
	}
	if string(empty) != `{}` {
		t.Errorf("empty scope = %s, want {} (no domain — asserts no absence)", empty)
	}
}
