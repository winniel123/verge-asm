package main

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

func TestGraphScopesVocabularyFromSeeds(t *testing.T) {
	p := netip.MustParsePrefix("162.222.48.0/22")
	seeds := []db.ListSeedsRow{
		{Kind: "name", NameDomain: pgtype.Text{String: "Example.COM.", Valid: true}},
		{Kind: "address", AddressCidr: &p},
		{Kind: "name"},
		{Kind: "address"},
	}

	got := graphScopes(seeds)
	if len(got) != 3 {
		t.Fatalf("scopes = %d, want 3 (whole estate + two declared Seeds); %#v", len(got), got)
	}
	if got[0].Token != graphScopeAll || got[0].Label != graphScopeAllLabel {
		t.Errorf("first entry = %#v, want the whole-estate default", got[0])
	}
	if got[0].Domain != "" || got[0].Prefix.IsValid() {
		t.Errorf("whole-estate entry names a population: %#v", got[0])
	}
	if got[1].Token != "example.com" || got[1].Domain != "example.com" {
		t.Errorf("name scope = %#v, want the case- and dot-folded domain as its token", got[1])
	}
	if got[2].Token != "162.222.48.0/22" || got[2].Prefix.String() != "162.222.48.0/22" {
		t.Errorf("address scope = %#v, want the masked CIDR as its token", got[2])
	}
}

func TestResolveGraphScopeFallsBackToWholeEstate(t *testing.T) {
	scopes := graphScopes([]db.ListSeedsRow{
		{Kind: "name", NameDomain: pgtype.Text{String: "example.com", Valid: true}},
	})

	for _, token := range []string{"", "all", "evil.example.com", "203.0.113.0/24"} {
		if got := resolveGraphScope(scopes, token); got.Token != graphScopeAll {
			t.Errorf("scope for token %q = %q, want the whole-estate default", token, got.Token)
		}
	}
	if got := resolveGraphScope(scopes, "example.com"); got.Domain != "example.com" {
		t.Errorf("scope for a declared token = %#v, want the name Seed it names", got)
	}
}

func TestBuildScopedGraphNameScope(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("name", "notexample.com", "resolution", "", `{"outcome":"Resolved","addresses":["198.51.100.9"]}`, false),
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
		openSpanRow("service", "198.51.100.9:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	}
	scope := graphScope{Token: "example.com", Label: "example.com", Domain: "example.com"}

	g := buildScopedGraph(rows, scope)

	want := map[string]bool{"api.example.com": true, "203.0.113.5": true, "203.0.113.5:443/tcp": true}
	got := map[string]bool{}
	for _, n := range g.Nodes {
		got[n.ID] = true
	}
	if len(got) != len(want) {
		t.Fatalf("nodes = %v, want exactly %v", got, want)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("node %q missing; the scope holds it", id)
		}
	}
	if len(g.Edges) != 2 {
		t.Errorf("edges = %d, want 2 inside the scope; %#v", len(g.Edges), g.Edges)
	}
}

func TestBuildScopedGraphNameScopeKeepsOnlyResolvedAddresses(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("name", "other.test", "resolution", "", `{"outcome":"Resolved","addresses":["198.51.100.9"]}`, false),
	}
	scope := graphScope{Token: "example.com", Domain: "example.com"}

	for _, n := range buildScopedGraph(rows, scope).Nodes {
		if n.ID == "198.51.100.9" {
			t.Fatalf("scoped graph drew an address no in-scope name resolves to")
		}
	}
}

func TestBuildScopedGraphAddressScope(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["162.222.48.7","198.51.100.9"]}`, false),
		openSpanRow("name", "away.test", "resolution", "", `{"outcome":"Resolved","addresses":["198.51.100.9"]}`, false),
		openSpanRow("service", "162.222.48.7:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
		openSpanRow("service", "198.51.100.9:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	}
	scope := graphScope{Token: "162.222.48.0/22", Prefix: netip.MustParsePrefix("162.222.48.0/22")}

	g := buildScopedGraph(rows, scope)

	want := map[string]bool{"api.example.com": true, "162.222.48.7": true, "162.222.48.7:443/tcp": true}
	got := map[string]bool{}
	for _, n := range g.Nodes {
		got[n.ID] = true
	}
	if len(got) != len(want) {
		t.Fatalf("nodes = %v, want exactly %v", got, want)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("node %q missing; the scope holds it", id)
		}
	}
}

func TestBuildScopedGraphAddressScopeIsFamilyMatched(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
	}
	scope := graphScope{Token: "::/0", Prefix: netip.MustParsePrefix("::/0")}

	if g := buildScopedGraph(rows, scope); len(g.Nodes) != 0 || !g.Empty {
		t.Fatalf("an IPv4 address drew inside an IPv6 scope; nodes %#v", g.Nodes)
	}
}

func TestBuildScopedGraphWholeEstateMatchesUnscoped(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
		openSpanRow("service", "not-a-service-key", "reachability", "", `{"outcome":"reached"}`, false),
	}

	unscoped := buildGraph(rows)
	scoped := buildScopedGraph(rows, graphScope{})

	if len(unscoped.Nodes) != len(scoped.Nodes) || len(unscoped.Edges) != len(scoped.Edges) {
		t.Fatalf("whole-estate scope narrowed the drawing: %d/%d nodes, %d/%d edges",
			len(scoped.Nodes), len(unscoped.Nodes), len(scoped.Edges), len(unscoped.Edges))
	}
	if len(unscoped.Nodes) != 4 {
		t.Fatalf("nodes = %d, want 4 (name, address, two services); %#v", len(unscoped.Nodes), unscoped.Nodes)
	}
}

func TestBuildScopedGraphDropsUnkeyedService(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("service", "not-a-service-key", "reachability", "", `{"outcome":"reached"}`, false),
	}
	scope := graphScope{Token: "example.com", Domain: "example.com"}

	for _, n := range buildScopedGraph(rows, scope).Nodes {
		if n.ID == "not-a-service-key" {
			t.Fatalf("scoped graph drew a service key naming no address")
		}
	}
}

func TestGraphPageRendersScopeSelector(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	if _, err := f.CreateNameSeed(context.Background(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: "example.com", Valid: true}, CreatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	p := netip.MustParsePrefix("162.222.48.0/22")
	if _, err := f.CreateAddressSeed(context.Background(), db.CreateAddressSeedParams{
		AddressCidr: &p, CreatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/graph?scope=example.com", http.StatusOK)
	for _, want := range []string{`id="gr-scope-btn"`, `href="/graph?scope=all"`, `href="/graph?scope=example.com"`} {
		if !strings.Contains(page, want) {
			t.Errorf("scoped /graph missing %s; body: %s", want, page)
		}
	}
	if !strings.Contains(strings.ToLower(page), "/graph?scope=162.222.48.0%2f22") {
		t.Errorf("scoped /graph missing the address Seed's option; body: %s", page)
	}
	if !strings.Contains(page, `<a class="opt on" href="/graph?scope=example.com"`) {
		t.Errorf("selected scope not marked in the control; body: %s", page)
	}

	fallback := getBody(t, ac, base+"/graph?scope=nobody.declared.this", http.StatusOK)
	if !strings.Contains(fallback, `<a class="opt on" href="/graph?scope=all"`) {
		t.Errorf("unrecognised token did not fall back to the whole estate; body: %s", fallback)
	}
}

func TestJoinSignalsDropsEndpointFiringTheScopeExcluded(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	}
	scope := graphScope{Token: "example.com", Domain: "example.com"}
	censuses := []signal.Census{
		{Rule: "plaintext-http-no-https", Fired: []signal.Member{{Subject: "evil.other.test@203.0.113.5:443/tcp"}}},
	}

	g := joinSignals(buildScopedGraph(rows, scope), censuses)

	for _, n := range g.Nodes {
		if len(n.OpenSignals) != 0 {
			t.Errorf("node %q carries a firing whose Name the scope excluded; %#v", n.ID, n.OpenSignals)
		}
	}

	inScope := []signal.Census{
		{Rule: "plaintext-http-no-https", Fired: []signal.Member{{Subject: "api.example.com@203.0.113.5:443/tcp"}}},
	}
	g = joinSignals(buildScopedGraph(rows, scope), inScope)
	for _, n := range g.Nodes {
		if n.ID == "api.example.com" && len(n.OpenSignals) != 1 {
			t.Errorf("in-scope endpoint firing did not light its Name node; %#v", n.OpenSignals)
		}
	}
}

func TestJoinSignalsKeepsServiceFallbackUnscoped(t *testing.T) {
	g := buildGraph([]db.ListAllOpenSpansRow{
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	})
	censuses := []signal.Census{
		{Rule: "plaintext-http-no-https", Fired: []signal.Member{{Subject: "www.example.com@203.0.113.5:443/tcp"}}},
	}

	g = joinSignals(g, censuses)

	for _, n := range g.Nodes {
		if n.ID == "203.0.113.5:443/tcp" && len(n.OpenSignals) != 1 {
			t.Errorf("unscoped graph lost the Service fallback; %#v", n.OpenSignals)
		}
	}
}

func TestGraphPageEmptyScopeStatesItsOwnEmptiness(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	if _, err := f.CreateNameSeed(context.Background(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: "example.com", Valid: true}, CreatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/graph?scope=example.com", http.StatusOK)
	if !strings.Contains(page, "Nothing to plot in example.com") {
		t.Errorf("empty scope did not name itself; body: %s", page)
	}
	if !strings.Contains(page, `href="/graph?scope=all"`) {
		t.Errorf("empty scope offered no way back to the whole estate; body: %s", page)
	}

	whole := getBody(t, ac, base+"/graph", http.StatusOK)
	if !strings.Contains(whole, "Nothing to plot yet") {
		t.Errorf("unscoped empty graph lost its own empty-state copy; body: %s", whole)
	}
}
