package main

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/winniel123/verge-asm/internal/db"
)

// graphScopes reads the declared Seeds into the selector's vocabulary — the whole
// estate first, then one entry per Seed — and the token is the Seed's own spelling,
// so a bookmarked /graph?scope= names what the operator declared.
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

// resolveGraphScope follows the Drift feed's ?period: an absent or unrecognised
// token falls back to the named default, so a hand-crafted value never draws a scope
// nobody declared.
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

// A name scope holds the names beneath its domain and the addresses those names
// resolve to. Membership is the label-wise suffix test, so notexample.com is not
// read as inside example.com.
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

// A name scope drops the addresses its names do not resolve to. The drawing states
// nothing about the dropped address.
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

// An address scope holds the addresses its prefix contains and the services riding
// them, and it reads the resolution edge the other way to hold the names that reach
// one. An address outside the prefix is dropped even where an in-scope name resolves
// to both.
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

// An IPv4 address is never read as inside an IPv6 scope: containment is family
// matched, never a comparison of spellings.
func TestBuildScopedGraphAddressScopeIsFamilyMatched(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
	}
	scope := graphScope{Token: "::/0", Prefix: netip.MustParsePrefix("::/0")}

	if g := buildScopedGraph(rows, scope); len(g.Nodes) != 0 || !g.Empty {
		t.Fatalf("an IPv4 address drew inside an IPv6 scope; nodes %#v", g.Nodes)
	}
}

// The whole-estate scope draws exactly what /graph has always drawn — the zero
// graphScope narrows nothing.
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

// A service key naming no address is held by no scope: the graph cannot decide it is
// inside a prefix or beneath a domain, so a scoped drawing drops it rather than
// guessing.
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

// The graph page renders the scope selector off the operator's declared Seeds, marks
// the selection, and falls back to the whole estate for a token nobody declared. The
// control renders on an empty drawing too, so a scope that holds nothing is not a
// dead end.
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
