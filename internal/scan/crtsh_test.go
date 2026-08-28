package scan

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// TestAdmittedNameKeyMatchesResolverKey locks the load-bearing invariant behind
// GetNameCitation's admission match (ADR-0107): admitted_name.name is stored via
// normaliseName, but the citation query matches it against the resolution
// subject_key — the ADR-0055 key the resolver assigns via CanonicalName. If those
// two normalisations ever diverge, a CT-admitted Name's admission silently fails
// to match and its citation wrongly falls back to the introducing resolution, the
// exact answer ADR-0107 forbids. normaliseName now routes through CanonicalName
// (#256), so the two fold identically; this guards that they stay so, over the
// non-ASCII spellings the old parallel Unicode fold got wrong, not only ASCII.
func TestAdmittedNameKeyMatchesResolverKey(t *testing.T) {
	// For any DNS-legal spelling — ASCII or not — the admitted key must equal the
	// key the resolver assigns the same name. The non-ASCII rows are the seam #256
	// found: the old Unicode fold lowercased "İ"/"Ä" while the resolver's ASCII-only
	// fold leaves them, so an admission stored under the folded spelling could never
	// match the resolver's subject_key. Both now fold ASCII-only and agree.
	for _, in := range []string{
		"example.com",
		"VPN.Example.COM.",
		"a.b.example.com",
		"MiXeD.Case.Example.Com",
		"trailing.dot.example.com.",
		"İ.EXAMPLE.COM",             // non-ASCII uppercase: ASCII fold leaves it, Unicode fold did not
		"Ä.example.com",             // ditto
		"xn--mnchen-3ya.example.com", // punycode passes through unchanged
	} {
		if got, want := normaliseName(in), resolutionwalk.CanonicalName(in); got != want {
			t.Errorf("normaliseName(%q)=%q but CanonicalName=%q — the admission-citation match (ADR-0107) would silently miss", in, got, want)
		}
	}
	// A crt.sh SAN value may carry surrounding whitespace the resolver never sees.
	// The admitter strips it and still lands on the resolver's key for the clean
	// name, so a whitespace-bearing admission still matches its resolution (#256).
	for _, ws := range []string{" x.example.com", "x.example.com\t", "  x.example.com  "} {
		if got, want := normaliseName(ws), resolutionwalk.CanonicalName("x.example.com"); got != want {
			t.Errorf("normaliseName(%q)=%q, want %q — surrounding whitespace must fold to the clean resolver key", ws, got, want)
		}
	}
}

// AdmittedNames is the whole admission decision, and it enforces two rulings:
// ADR-0060 (no asterisk-label value admits a Name) and ADR-0047 (the Seed decides
// which names are inside). It also splits the newline-separated SAN list, folds in
// the common name, normalises, and dedupes to a deterministic set.
func TestAdmittedNamesFiltersAndDedupes(t *testing.T) {
	rows := []CrtshRow{
		// A leaf cert: SAN list with the apex, a subdomain, and its own CN.
		{CommonName: "www.example.com", NameValue: "example.com\nwww.example.com"},
		// A wildcard cert: the wildcard SAN admits nothing (ADR-0060), but the
		// apex it also carries is admitted.
		{CommonName: "*.example.com", NameValue: "*.example.com\nexample.com"},
		// A partial wildcard: two denotations, refused (ADR-0060).
		{CommonName: "", NameValue: "baz*.example.com"},
		// A multi-SAN cert co-tenanting a foreign estate: the foreign name is
		// outside the queried scope and admits nothing here (ADR-0047).
		{CommonName: "shared.example.net", NameValue: "api.example.com\nshared.example.net"},
		// Case and a trailing dot normalise to the same key as an earlier row.
		{CommonName: "WWW.Example.com.", NameValue: ""},
	}
	got := AdmittedNames(rows, "example.com")
	want := []string{"api.example.com", "example.com", "www.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AdmittedNames = %v, want %v", got, want)
	}
}

// A wildcard-only answer admits nothing — the concealment case (ADR-0060): a
// wildcard hides the names behind it, so a query answered entirely by wildcard
// SANs admits no Name rather than the wildcard or the names beneath it.
func TestAdmittedNamesWildcardOnlyAdmitsNothing(t *testing.T) {
	rows := []CrtshRow{
		{CommonName: "*.example.com", NameValue: "*.example.com"},
		{CommonName: "*.internal.example.com", NameValue: "*.internal.example.com"},
	}
	if got := AdmittedNames(rows, "example.com"); len(got) != 0 {
		t.Errorf("AdmittedNames = %v, want empty (wildcards admit nothing)", got)
	}
}

// The scope filter is label-wise: a name that merely shares a suffix string with
// the domain is not under it (ADR-0047).
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

// The apex itself is admitted when the cert carries it (ADR-0060's note: a cert
// covering both apex and wildcard carries the apex as its own SAN).
func TestAdmittedNamesAdmitsApex(t *testing.T) {
	rows := []CrtshRow{{CommonName: "example.com", NameValue: "example.com\n*.example.com"}}
	got := AdmittedNames(rows, "example.com")
	want := []string{"example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AdmittedNames = %v, want %v", got, want)
	}
}

// AdmittedNames caps how many distinct Names one response may admit (#741). Only
// the 64 MiB body byte-cap otherwise bounds a crt.sh answer, so a compromised or
// MITM'd operator could pack millions of unique in-scope names into a single body
// and mint an admitted_name row for each — DB bloat and mass-resolution
// amplification against the operator's own instance. Fed more than MaxAdmittedNames
// in-scope, deduped names, the admitted set must be capped at the ceiling and stay
// a sorted, deduped set — the count is bounded before any row reaches admitCT.
func TestAdmittedNamesCapsCardinality(t *testing.T) {
	const over = MaxAdmittedNames + 500
	var b strings.Builder
	for i := 0; i < over; i++ {
		fmt.Fprintf(&b, "h%d.example.com\n", i) // all unique, all in scope
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
	// A malformed 200 is not evidence of anything (ADR-0027 §7): a parse error, so
	// the caller treats it as transient rather than as "no certificates".
	t.Run("non-array body is an error, never empty", func(t *testing.T) {
		for _, body := range []string{"", "   ", "<html>404</html>", "not json"} {
			if _, err := ParseCrtshRows([]byte(body)); err == nil {
				t.Errorf("ParseCrtshRows(%q) = nil error, want a parse error", body)
			}
		}
	})
}

// The URL is the wildcard-identity JSON query that includes subdomains
// (passive-discovery §2.2): %25 is the URL-encoded LIKE wildcard.
func TestCrtshURL(t *testing.T) {
	got := CrtshURL("example.com")
	want := "https://crt.sh/?q=%25.example.com&output=json"
	if got != want {
		t.Errorf("CrtshURL = %q, want %q", got, want)
	}
}

// The job's scope round-trips through the wire and back, carrying the domain and
// the Seed the admission chain terminates at (ADR-0027).
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

// The completed Batch records the domain it queried; a dead-lettered Batch records
// an empty scope, never the domain — a failed fetch asserts no absence (ADR-0005).
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
