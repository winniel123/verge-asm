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

func addNameSeed(t *testing.T, f *fakeStore, createdBy int64, domain string) int64 {
	t.Helper()
	seed, err := f.CreateNameSeed(t.Context(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: createdBy,
	})
	if err != nil {
		t.Fatal(err)
	}
	return seed.ID
}

// This sits inside the fixed clock's k=2 live window; outside it a fixture is evidential (#237).

var obsClock = time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)

func TestSubjectsListRedirectsToInventory(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/subjects")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMovedPermanently || resp.Header.Get("Location") != "/inventory" {
		t.Fatalf("GET /subjects: status=%d location=%q, want 301 -> /inventory",
			resp.StatusCode, resp.Header.Get("Location"))
	}

	resp, err = ac.Get(base + "/subjects?q=example.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("Location"); resp.StatusCode != http.StatusMovedPermanently || got != "/inventory?q=example.com" {
		t.Fatalf("GET /subjects?q=example.com: status=%d location=%q, want 301 -> /inventory?q=example.com",
			resp.StatusCode, got)
	}
}

func TestWithdrawnNameNotListedButReachableByKey(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "live.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock, `{"outcome":"Resolved"}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"NameError"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/gone.example.com", http.StatusOK)
	if !strings.Contains(drill, "withdrawn") || !strings.Contains(drill, "no current member") {
		t.Errorf("withdrawn drill-down not marked; body: %s", drill)
	}
}

func TestShadowedNameSuppressedButReachableByKey(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "real.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	f.addResolution(t, admin.ID, "ghost.example.com", "dns", obsClock, `{"outcome":"Shadowed"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/ghost.example.com", http.StatusOK)
	if !strings.Contains(drill, "withdrawn") || !strings.Contains(drill, "no current member") {
		t.Errorf("Shadowed drill-down not marked as no current member; body: %s", drill)
	}
}

func TestNameSubjectByKeyOpensAssetDetail(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.1"]}`)
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock.Add(48*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.1","203.0.113.2"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/example.com", http.StatusOK)

	for _, want := range []string{"How it got here", "Drift trail", "Open ports", "203.0.113.2"} {
		if !strings.Contains(drill, want) {
			t.Errorf("Name by-key drill-down did not open the Asset detail (missing %q); body: %s", want, drill)
		}
	}
	if strings.Contains(drill, "ticket 22") {
		t.Errorf("retired Name-drilldown placeholder still present; body: %s", drill)
	}
}

func TestNameByKeyCitesCTAdmissionOverResolution(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addAdmittedName(t, "vpn.example.com", obsClock)
	f.addResolution(t, admin.ID, "vpn.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/vpn.example.com", http.StatusOK)

	for _, want := range []string{"How it got here", "certificate transparency", "203.0.113.9"} {
		if !strings.Contains(drill, want) {
			t.Errorf("CT-admitted Name provenance missing %q; body: %s", want, drill)
		}
	}
	if strings.Contains(drill, "resolution-walk") {
		t.Errorf("CT-admitted Name cited its resolution, not its admission; body: %s", drill)
	}
}

func TestNameByKeyCTAdmissionTerminatesAtAdmittedSeed(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	outer := addNameSeed(t, f, admin.ID, "example.com")
	// A nested scope is what the admitted Seed must be picked over, so the fixture declares one.
	addNameSeed(t, f, admin.ID, "inner.example.com")
	f.addAdmittedNameUnderSeed(t, "host.inner.example.com", outer, obsClock)
	f.addResolution(t, admin.ID, "host.inner.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.9"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/host.inner.example.com", http.StatusOK)

	prov := drill
	if i := strings.Index(prov, "How it got here"); i >= 0 {
		prov = prov[i:]
		if j := strings.Index(prov, "</section>"); j >= 0 {
			prov = prov[:j]
		}
	}
	if !strings.Contains(prov, "example.com") {
		t.Errorf("provenance Seed did not terminate at admitted_name.seed_id (example.com); provenance: %s", prov)
	}
	if strings.Contains(prov, "inner.example.com") {
		t.Errorf("provenance Seed terminated at the longest-suffix Seed, ignoring admitted_name.seed_id (#256); provenance: %s", prov)
	}
}

func TestNameByKeyRendersDriftTrail(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.1"]}`)
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.2"]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/api.example.com", http.StatusOK)

	for _, want := range []string{"Drift trail", "changed", "203.0.113.2"} {
		if !strings.Contains(drill, want) {
			t.Errorf("drift trail missing %q; body: %s", want, drill)
		}
	}
}

func TestSpanDetailsListsRecordsAndAddresses(t *testing.T) {
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
			name:  "certificate lists its chain, leaf then issuer",
			facet: "certificate",
			raw:   `{"outcome":"valid","chain":["sha256:leaf","sha256:issuer"]}`,
			want:  []spanDetail{{Type: "leaf", Data: "sha256:leaf"}, {Type: "issuer", Data: "sha256:issuer"}},
		},
		{
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

func TestNameByKeyRendersDNSRecords(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.7"]}`)
	f.addResolution(t, admin.ID, "example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"Resolved","addresses":["203.0.113.8"]}`)
	f.addDNSRecord(t, "example.com", "TXT", obsClock, `{"rrs":[{"name":"example.com","type":"TXT","data":"\"v=spf1 -all\""}]}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/example.com", http.StatusOK)

	for _, want := range []string{"DNS records", "203.0.113.8", `v=spf1 -all`} {
		if !strings.Contains(drill, want) {
			t.Errorf("Name DNS section missing %q; body: %s", want, drill)
		}
	}
}

func TestServiceSubjectsListedAndDrilledDown(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)
	for _, want := range []string{
		"198.51.100.1:443/tcp", "reached",
		"Citation chain", "api.example.com", "443",
		"Rules over this subject", "How it got here",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("service drill-down missing %q; body: %s", want, drill)
		}
	}
}

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
	if strings.Contains(drill, `<span class="badge">reached</span>`) {
		t.Errorf("a blanketed reach must not render as reached; body: %s", drill)
	}
}

func TestServiceDrilldownRendersOpenCloseTimeline(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"not-reached","result":"refused"}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock.Add(24*time.Hour), `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	drill := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)
	for _, want := range []string{"Current and closed timelines", "reachability", "Current", "Opened", "Closed", "not-reached"} {
		if !strings.Contains(drill, want) {
			t.Errorf("service timeline missing %q; body: %s", want, drill)
		}
	}
}

func TestEndpointSubjectsListedAndDrilledDown(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addHTTPIdentity(t, "api.example.com@198.51.100.1:443/tcp", obsClock,
		`{"outcome":"responded","status":200,"server":"nginx","title":"Example API"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	key := url.QueryEscape("api.example.com@198.51.100.1:443/tcp")
	drill := getBody(t, ac, base+"/subjects/endpoint?key="+key, http.StatusOK)
	for _, want := range []string{
		"api.example.com@198.51.100.1:443/tcp", "HTTP identity",
		"nginx", "Citation chain", "api.example.com", "198.51.100.1:443/tcp",
		"Rules over this subject", "How it got here",
	} {
		if !strings.Contains(drill, want) {
			t.Errorf("endpoint drill-down missing %q; body: %s", want, drill)
		}
	}
}

func TestNamelessEndpointRendersAndRedirectRecorded(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addHTTPIdentity(t, "@198.51.100.2:80/tcp", obsClock,
		`{"outcome":"responded","status":301,"server":"nginx","redirect_location":"https://x.example/"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	key := url.QueryEscape("@198.51.100.2:80/tcp")
	drill := getBody(t, ac, base+"/subjects/endpoint?key="+key, http.StatusOK)
	for _, want := range []string{"(nameless)", "301", "https://x.example/", "not followed"} {
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

func TestDerivationReadGatedAtLiveBoundaryWithoutDelete(t *testing.T) {
	observedAt := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	readInside := observedAt.Add(24 * time.Hour)
	readPast := observedAt.Add(72 * time.Hour)

	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, 1, "api.example.com", "dns", observedAt,
		`{"outcome":"Resolved","addresses":["203.0.113.5"]}`)

	baseInside := startAt(t, f, readInside)
	acInside := login(t, baseInside, "admin", "hunter2hunter2")
	getBody(t, acInside, baseInside+"/subjects/api.example.com", http.StatusOK)

	basePast := startAt(t, f, readPast)
	acPast := login(t, basePast, "admin", "hunter2hunter2")
	drill := getBody(t, acPast, basePast+"/subjects/api.example.com", http.StatusNotFound)
	if !strings.Contains(drill, "No such subject") {
		t.Errorf("evidential Name still reachable by key past its bound; body: %s", drill)
	}

	if len(f.observations) != 1 {
		t.Fatalf("observation corpus changed: got %d rows, want 1 (no delete expected)", len(f.observations))
	}
}

func TestDerivationReadRetainsTimelineWithNoEnabledScan(t *testing.T) {
	observedAt := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, 1, "api.example.com", "dns", observedAt,
		`{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	for i := range f.scans {
		f.scans[i].Enabled = false
	}

	base := startAt(t, f, observedAt.Add(time.Minute))
	ac := login(t, base, "admin", "hunter2hunter2")
	drill := getBody(t, ac, base+"/subjects/api.example.com", http.StatusNotFound)
	if !strings.Contains(drill, "No such subject") {
		t.Errorf("Name on an uncovered timeline was derived (reachable by key); body: %s", drill)
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
