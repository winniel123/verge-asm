package main

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func getBody(t *testing.T, c *http.Client, url string, wantStatus int) string {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s status = %d, want %d (body: %s)", url, resp.StatusCode, wantStatus, got)
	}
	return got
}

func addNameSeed(t *testing.T, f *fakeStore, createdBy int64, domain string) {
	t.Helper()
	if _, err := f.CreateNameSeed(t.Context(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: createdBy,
	}); err != nil {
		t.Fatal(err)
	}
}

// obsClock is the instant the Subjects/Signals fixtures seed their observations
// at. It sits inside the live-tier window of the server's fixedClock (2026-08-15
// 12:00): the derivation reads are gated against that clock (#237), and the daily
// (86400s) fixture Scans give a k=2 live window of 2 days, so a fixture seeded
// here — and its +24h/+48h successors — is read as live, not filtered as stale.
var obsClock = time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)

func TestSubjectsListsCurrentNamesWithSearchAndNoDenominator(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	f.addResolution(t, admin.ID, "www.example.net", "dns", obsClock, `{"outcome":"NoData"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	for _, name := range []string{"api.example.com", "www.example.net"} {
		if !strings.Contains(page, name) {
			t.Errorf("listing missing %q; body: %s", name, page)
		}
		if !strings.Contains(page, `href="/subjects/`+name+`"`) {
			t.Errorf("listing missing drill-down link for %q; body: %s", name, page)
		}
	}
	// No denominator: the screen states it holds no total, and renders no count.
	if !strings.Contains(page, "There is no total") {
		t.Errorf("listing does not refuse a denominator in copy; body: %s", page)
	}
	// No rendered count of the listing anywhere on the screen.
	for _, denom := range []string{"2 names", "2 subjects", "Showing 2", "of 2"} {
		if strings.Contains(page, denom) {
			t.Errorf("listing rendered a count/denominator %q; body: %s", denom, page)
		}
	}

	// Search narrows to the matching Name.
	only := getBody(t, ac, base+"/subjects?q=example.com", http.StatusOK)
	if !strings.Contains(only, "api.example.com") || strings.Contains(only, "www.example.net") {
		t.Errorf("search did not narrow to example.com; body: %s", only)
	}
}

func TestWithdrawnNameNotListedButReachableByKey(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// A live Name and a withdrawn Name (latest resolution is a Name Error).
	f.addResolution(t, admin.ID, "live.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock, `{"outcome":"Resolved"}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"NameError"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	if !strings.Contains(page, "live.example.com") {
		t.Errorf("live name not listed; body: %s", page)
	}
	if strings.Contains(page, "gone.example.com") {
		t.Errorf("withdrawn name appears in listing; body: %s", page)
	}
	// Searching for it does not surface it in the listing either.
	searched := getBody(t, ac, base+"/subjects?q=gone.example.com", http.StatusOK)
	if strings.Contains(searched, `href="/subjects/gone.example.com"`) {
		t.Errorf("withdrawn name surfaced by search listing; body: %s", searched)
	}

	// It is still reachable by its own key, marked as a population of no member.
	drill := getBody(t, ac, base+"/subjects/gone.example.com", http.StatusOK)
	if !strings.Contains(drill, "withdrawn") || !strings.Contains(drill, "no current member") {
		t.Errorf("withdrawn drill-down not marked; body: %s", drill)
	}
}

// A Shadowed Name (wildcard-discrimination's suppression outcome) is suppressed
// from the estate as affirmatively as a NameError one (#192): it is not listed,
// and its drill-down marks it a population of no current member.
func TestShadowedNameSuppressedButReachableByKey(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "real.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	f.addResolution(t, admin.ID, "ghost.example.com", "dns", obsClock, `{"outcome":"Shadowed"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	if !strings.Contains(page, "real.example.com") {
		t.Errorf("discriminated real name not listed; body: %s", page)
	}
	if strings.Contains(page, "ghost.example.com") {
		t.Errorf("Shadowed name appears in listing; body: %s", page)
	}

	drill := getBody(t, ac, base+"/subjects/ghost.example.com", http.StatusOK)
	if !strings.Contains(drill, "withdrawn") || !strings.Contains(drill, "no current member") {
		t.Errorf("Shadowed drill-down not marked as no current member; body: %s", drill)
	}
}

func TestSubjectDrilldownRendersCitationChainAndPlaceholders(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// An earlier and a later observation; the chain cites the earliest.
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.1"]}`)
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock.Add(48*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.1","203.0.113.2"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/example.com", http.StatusOK)

	// The "why is this here" Citation chain: subject → introducing observation →
	// terminating Seed.
	for _, want := range []string{
		"Citation chain", "resolution-walk", "dns Scan",
		"name scope example.com", "declared by admin",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("citation chain missing %q; body: %s", want, drill)
		}
	}
	// Current resolution value renders.
	if !strings.Contains(drill, "Resolved") || !strings.Contains(drill, "203.0.113.2") {
		t.Errorf("current resolution not rendered; body: %s", drill)
	}
	// The Timelines section is now wired (#190): the two Resolved answers fold to a
	// closed span and a current one, on the resolution timeline. The Rules section
	// remains a placeholder for ticket 22.
	if !strings.Contains(drill, "Current and closed timelines") {
		t.Errorf("timelines section missing; body: %s", drill)
	}
	if !strings.Contains(drill, "ticket 22") {
		t.Errorf("rules placeholder missing; body: %s", drill)
	}
}

// ADR-0107 (#255): a CT-admitted Name that our resolver has since measured shows
// the CT admission as its Citation, not the introducing resolution — the admission
// is why the Name is here; the resolution is only that it is. So the "why is this
// here" chain reads "Admitted by · certificate transparency" and terminates at the
// covering Seed, while the current resolution value still renders.
func TestSubjectDrilldownCitesCTAdmissionOverResolution(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// The Name was admitted by CT, then resolved by our own resolver (wave-1).
	f.addAdmittedName(t, "vpn.example.com", obsClock)
	f.addResolution(t, admin.ID, "vpn.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/vpn.example.com", http.StatusOK)

	// The citation is the admission, terminating at the covering Seed.
	for _, want := range []string{
		"Citation chain", "Admitted by", "certificate transparency", "ct",
		"name scope example.com",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("CT-admitted citation missing %q; body: %s", want, drill)
		}
	}
	// It must NOT read as an introducing resolution — that is the wrong answer to
	// "why is this here" for a CT-admitted Name (ADR-0107).
	if strings.Contains(drill, "Introduced by · observation") {
		t.Errorf("CT-admitted Name cited its resolution, not its admission; body: %s", drill)
	}
	// Membership is still measured: the current resolution value renders.
	if !strings.Contains(drill, "Resolved") || !strings.Contains(drill, "203.0.113.9") {
		t.Errorf("current resolution of a CT-admitted Name not rendered; body: %s", drill)
	}
}

func TestSubjectDrilldownRendersCurrentAndClosedTimelines(t *testing.T) {
	// AC6: re-running the dns Scan with a changed answer produces a new Span and
	// closes the old; the drill-down renders current + closed timelines.
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.1"]}`)
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.2"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/api.example.com", http.StatusOK)

	// A current span (the later answer) and a closed-history row (the earlier one).
	for _, want := range []string{"Current and closed timelines", "Current", "Opened", "Closed"} {
		if !strings.Contains(drill, want) {
			t.Errorf("timeline drill-down missing %q; body: %s", want, drill)
		}
	}
	// The resolution facet timeline is labelled and its closed span is present.
	if !strings.Contains(drill, "resolution") {
		t.Errorf("resolution timeline not labelled; body: %s", drill)
	}
}

func TestSpanDetailsListsRecordsAndAddresses(t *testing.T) {
	// #240 seam: the terminal extraction that turns a span's already-read value
	// JSON into the rows the drill-down lists on expand — RR type+data for a
	// dns-record span, the address list (typeless) for a resolution span.
	tests := []struct {
		name  string
		facet string
		raw   string
		isGap bool
		want  []spanDetail
	}{
		{
			name:  "dns-record RRs list type and data, TXT data as its quoted strings",
			facet: "dns-record",
			raw:   `{"rrs":[{"name":"example.com","type":"A","data":"203.0.113.1"},{"name":"example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`,
			want:  []spanDetail{{Type: "A", Data: "203.0.113.1"}, {Type: "TXT", Data: `"v=spf1 -all"`}},
		},
		{
			name:  "resolution addresses list with no type",
			facet: "resolution",
			raw:   `{"outcome":"Resolved","addresses":["203.0.113.1","203.0.113.2"]}`,
			want:  []spanDetail{{Data: "203.0.113.1"}, {Data: "203.0.113.2"}},
		},
		{
			name:  "resolution with no addresses does not expand",
			facet: "resolution",
			raw:   `{"outcome":"NoData"}`,
			want:  nil,
		},
		{
			name:  "dns-record with no RRs does not expand",
			facet: "dns-record",
			raw:   `{"rrs":[]}`,
			want:  nil,
		},
		{
			name:  "a gap holds no value and expands to nothing",
			facet: "dns-record",
			raw:   `{"rrs":[{"type":"A","data":"203.0.113.1"}]}`,
			isGap: true,
			want:  nil,
		},
		{
			name:  "other facets have no per-item breakdown",
			facet: "reachability",
			raw:   `{"outcome":"reached"}`,
			want:  nil,
		},
		{
			// #243: http-identity expands to its admitted closed set (ADR-0011),
			// each present field one row; the body is never stored so none appears.
			name:  "http-identity lists its admitted fields",
			facet: "http-identity",
			raw:   `{"outcome":"responded","status":200,"server":"nginx","title":"Home","www_authenticate":"Basic realm=x","redirect_location":"https://x/"}`,
			want: []spanDetail{
				{Type: "status", Data: "200"},
				{Type: "server", Data: "nginx"},
				{Type: "title", Data: "Home"},
				{Type: "www-authenticate", Data: "Basic realm=x"},
				{Type: "location", Data: "https://x/"},
			},
		},
		{
			name:  "http-identity that spoke no HTTP lists that negative as its value",
			facet: "http-identity",
			raw:   `{"outcome":"no-http-response"}`,
			want:  []spanDetail{{Type: "outcome", Data: "no HTTP response"}},
		},
		{
			// #243: certificate expands to its presented chain, leaf first.
			name:  "certificate lists its chain, leaf then issuer",
			facet: "certificate",
			raw:   `{"outcome":"valid","chain":["sha256:leaf","sha256:issuer"]}`,
			want:  []spanDetail{{Type: "leaf", Data: "sha256:leaf"}, {Type: "issuer", Data: "sha256:issuer"}},
		},
		{
			// #243: tls-acceptance expands to one row per accepted version — its
			// suites as data, "—" for TLS 1.3 (library-chosen, not measured).
			name:  "tls-acceptance lists accepted versions and their suites",
			facet: "tls-acceptance",
			raw:   `{"outcome":"enumerated","versions":[{"version":"TLS1.3"},{"version":"TLS1.2","ciphers":["ECDHE_RSA_AES_128_GCM","ECDHE_RSA_AES_256_GCM"]}]}`,
			want:  []spanDetail{{Type: "TLS1.3", Data: "—"}, {Type: "TLS1.2", Data: "ECDHE_RSA_AES_128_GCM, ECDHE_RSA_AES_256_GCM"}},
		},
		{
			name:  "tls-acceptance refusal carries no versions and does not expand",
			facet: "tls-acceptance",
			raw:   `{"outcome":"tls-refused"}`,
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := spanDetails(tt.facet, []byte(tt.raw), tt.isGap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("spanDetails(%q) = %#v, want %#v", tt.facet, got, tt.want)
			}
		})
	}
}

func TestSubjectDrilldownExpandsRecordContents(t *testing.T) {
	// #240: the Name drill-down keeps the collapsed count/outcome summary but
	// expands on click to the span's actual records — RR type+data for
	// dns-record, the address list for resolution — sourced from the already-read
	// span value, no new query. Applies to closed spans too.
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// A resolution timeline that moves (a closed span then a current one), and a
	// dns-record timeline whose contents appear nowhere else on the page.
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.7"]}`)
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.8"]}`)
	f.addDNSRecord(t, "example.com", "TXT", obsClock, `{"rrs":[{"name":"example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/example.com", http.StatusOK)

	// Collapsed summary is unchanged: the dns-record count still renders.
	if !strings.Contains(drill, "1 record") {
		t.Errorf("collapsed dns-record count missing; body: %s", drill)
	}
	// The expand affordance wraps the summary in a disclosure.
	if !strings.Contains(drill, `<details class="spanrecords"`) {
		t.Errorf("expand-on-click affordance missing; body: %s", drill)
	}
	// Expanded contents: the TXT data (present nowhere else) and the current +
	// closed resolution addresses all render.
	for _, want := range []string{`v=spf1 -all`, "203.0.113.7", "203.0.113.8"} {
		if !strings.Contains(drill, want) {
			t.Errorf("expanded record content missing %q; body: %s", want, drill)
		}
	}
}

func TestServiceSubjectsListedAndDrilledDown(t *testing.T) {
	// AC #195: the Subjects page renders Service subjects, and the Service
	// drill-down shows its reachability verdict and citation back to a Seed.
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// A Name resolves to the address the Service sits on — the citation ground.
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The listing carries a Service subjects section with the triple and verdict.
	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	for _, want := range []string{"Service subjects", "198.51.100.1:443/tcp", "reached"} {
		if !strings.Contains(page, want) {
			t.Errorf("subjects listing missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, "/subjects/service?key=") {
		t.Errorf("service drill-down link missing; body: %s", page)
	}

	// The drill-down: verdict, address split out, and citation to the Name and Seed.
	drill := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)
	for _, want := range []string{
		"Observed · Service", "198.51.100.1:443/tcp", "reached",
		"Citation chain", "api.example.com", "443",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("service drill-down missing %q; body: %s", want, drill)
		}
	}
}

// A blanket responder's Service drill-down states the proxy-edge finding in prose
// (ADR-0104 §4): the reach is a Gap, not `reached`, and the page says we cannot see
// the origin from here rather than showing an open/closed verdict.
func TestServiceDrilldownSurfacesBlanketGap(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["104.21.61.6"]}`)
	f.addReachability(t, "104.21.61.6:443/tcp", obsClock,
		`{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=104.21.61.6%3A443%2Ftcp", http.StatusOK)
	for _, want := range []string{
		"proxy edge", "answers on all ports", "Gap", "address scope",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("blanket-responder drill-down missing %q; body: %s", want, drill)
		}
	}
	// It must NOT claim a reached/closed verdict — the reach is undiscriminated.
	if strings.Contains(drill, `<span class="badge">reached</span>`) {
		t.Errorf("a blanketed reach must not render as reached; body: %s", drill)
	}
}

func TestServiceDrilldownRendersOpenCloseTimeline(t *testing.T) {
	// AC #195: re-running the hot Scan with a Service opening produces the correct
	// Span transition — a not-reached span closes and a reached span opens — visible
	// on the Service's own drill-down.
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"not-reached","result":"refused"}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock.Add(24*time.Hour), `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)
	// A current span (reached) and a closed-history row (the earlier not-reached).
	for _, want := range []string{"Current and closed timelines", "reachability", "Current", "Opened", "Closed", "not-reached"} {
		if !strings.Contains(drill, want) {
			t.Errorf("service timeline missing %q; body: %s", want, drill)
		}
	}
}

func TestEndpointSubjectsListedAndDrilledDown(t *testing.T) {
	// AC #198: the Subjects page renders Endpoint subjects, and the Endpoint
	// drill-down shows its HTTP identity and citation back through its Service and
	// Name legs to a Seed.
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addHTTPIdentity(t, "api.example.com@198.51.100.1:443/tcp", obsClock,
		`{"outcome":"responded","status":200,"server":"nginx","title":"Example API"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The listing carries an Endpoint subjects section with the pair and its identity.
	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	for _, want := range []string{"Endpoint subjects", "api.example.com", "198.51.100.1:443/tcp", "200 · nginx"} {
		if !strings.Contains(page, want) {
			t.Errorf("subjects listing missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, "/subjects/endpoint?key=") {
		t.Errorf("endpoint drill-down link missing; body: %s", page)
	}

	key := url.QueryEscape("api.example.com@198.51.100.1:443/tcp")
	drill := getBody(t, ac, base+"/subjects/endpoint?key="+key, http.StatusOK)
	for _, want := range []string{
		"Observed · Endpoint", "api.example.com@198.51.100.1:443/tcp", "HTTP identity",
		"nginx", "Citation chain", "api.example.com", "198.51.100.1:443/tcp",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("endpoint drill-down missing %q; body: %s", want, drill)
		}
	}
}

func TestNamelessEndpointRendersAndRedirectRecorded(t *testing.T) {
	// AC #198: the nameless endpoint (@service) is a distinguished key variant, and
	// a 3xx records its Location as identity without following it.
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addHTTPIdentity(t, "@198.51.100.2:80/tcp", obsClock,
		`{"outcome":"responded","status":301,"server":"nginx","redirect_location":"https://x.example/"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	if !strings.Contains(page, "(nameless)") {
		t.Errorf("nameless endpoint not marked in listing; body: %s", page)
	}

	key := url.QueryEscape("@198.51.100.2:80/tcp")
	drill := getBody(t, ac, base+"/subjects/endpoint?key="+key, http.StatusOK)
	for _, want := range []string{"nameless endpoint", "301", "https://x.example/", "not followed"} {
		if !strings.Contains(drill, want) {
			t.Errorf("nameless/redirect drill-down missing %q; body: %s", want, drill)
		}
	}
}

func TestEndpointMissingReturns404(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	key := url.QueryEscape("gone.example.com@203.0.113.9:443/tcp")
	got := getBody(t, ac, base+"/subjects/endpoint?key="+key, http.StatusNotFound)
	if !strings.Contains(got, "No such subject") {
		t.Errorf("missing endpoint not reported as 404; body: %s", got)
	}
}

func TestServiceMissingReturns404(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/subjects/service?key=203.0.113.9%3A22%2Ftcp", http.StatusNotFound)
	if !strings.Contains(got, "No such subject") {
		t.Errorf("missing service not reported as 404; body: %s", got)
	}
}

func TestSubjectMissingReturns404(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/subjects/never.measured.example", http.StatusNotFound)
	if !strings.Contains(got, "No such subject") {
		t.Errorf("missing subject not reported as 404 page; body: %s", got)
	}
}

// TestDerivationReadGatedAtLiveBoundaryWithoutDelete is the #237 acceptance
// proof: a derivation read of the observation corpus is bounded by the live-tier
// gate evaluated against the caller's read instant, so an evidential row is
// structurally unreadable the instant it crosses its bound — independent of
// whether the Retirer has swept. One Name is seeded with a single resolution
// observation on a daily-cadence (86400s) dns Scan, so its live window is
// FloorCadences (k=2) cadences = 2 days. The same fakeStore is read by two
// servers whose only difference is the clock: one inside the window, one past it.
// No delete runs between the reads — the observation row is still present after
// the second read — proving the separation is enforced by the read gate, not by
// retirement.
func TestDerivationReadGatedAtLiveBoundaryWithoutDelete(t *testing.T) {
	observedAt := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	readInside := observedAt.Add(24 * time.Hour) // 1 day old — inside the 2-day window
	readPast := observedAt.Add(72 * time.Hour)   // 3 days old — past the window

	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, 1, "api.example.com", "dns", observedAt,
		`{"outcome":"Resolved","addresses":["203.0.113.5"]}`)

	// Inside the window: the derivation lists the Name and reaches its drill-down.
	baseInside := startAt(t, f, readInside)
	acInside := login(t, baseInside, "admin", "hunter2hunter2")
	page := getBody(t, acInside, baseInside+"/subjects", http.StatusOK)
	if !strings.Contains(page, "api.example.com") {
		t.Fatalf("live-tier Name not listed inside its window; body: %s", page)
	}
	getBody(t, acInside, baseInside+"/subjects/api.example.com", http.StatusOK)

	// Past the window, with NO delete: the same row is now evidential, so the
	// derivation neither lists the Name nor reaches it by key.
	basePast := startAt(t, f, readPast)
	acPast := login(t, basePast, "admin", "hunter2hunter2")
	page = getBody(t, acPast, basePast+"/subjects", http.StatusOK)
	if strings.Contains(page, "api.example.com") {
		t.Errorf("evidential Name still listed past its bound; body: %s", page)
	}
	drill := getBody(t, acPast, basePast+"/subjects/api.example.com", http.StatusNotFound)
	if !strings.Contains(drill, "No such subject") {
		t.Errorf("evidential Name still reachable by key past its bound; body: %s", drill)
	}

	// The row was never deleted — the gate, not retirement, made it unreadable.
	if len(f.observations) != 1 {
		t.Fatalf("observation corpus changed: got %d rows, want 1 (no delete expected)", len(f.observations))
	}
}

// TestDerivationReadRetainsTimelineWithNoEnabledScan proves the per-timeline
// bound (#237 AC4): a timeline no enabled Scan covers has an undefined bound and
// yields no live row, so its subject is retained as evidence but never derived.
func TestDerivationReadRetainsTimelineWithNoEnabledScan(t *testing.T) {
	observedAt := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// Seed a resolution on a Scan kind, then disable every Scan so no enabled Scan
	// covers the timeline: its bound is undefined and it must not derive, even read
	// the instant after it was observed.
	f.addResolution(t, 1, "api.example.com", "dns", observedAt,
		`{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	for i := range f.scans {
		f.scans[i].Enabled = false
	}

	base := startAt(t, f, observedAt.Add(time.Minute))
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/subjects", http.StatusOK)
	if strings.Contains(page, "api.example.com") {
		t.Errorf("Name on an uncovered timeline was derived; body: %s", page)
	}
	if len(f.observations) != 1 {
		t.Fatalf("observation corpus changed: got %d rows, want 1 (retained as evidence)", len(f.observations))
	}
}

func TestSubjectsRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/subjects")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /subjects: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
