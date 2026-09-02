package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// buildGraph folds the estate's open spans into a real Name/Address/Service
// topology: a Name's resolution addresses give the Name-to-Address edge, and a
// Service key's Address gives the Address-to-Service edge. No node, edge, or
// severity is invented.
func TestBuildGraphTopologyFromOpenSpans(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	}

	g := buildGraph(rows)
	if g.Empty {
		t.Fatalf("graph reported empty with a name, address and service to plot")
	}
	if len(g.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3 (name, address, service); %#v", len(g.Nodes), g.Nodes)
	}

	byID := map[string]graphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	name, ok := byID["api.example.com"]
	if !ok || name.Type != "subdomain" {
		t.Fatalf("name node = %#v, want a subdomain-type node for api.example.com", name)
	}
	addr, ok := byID["203.0.113.5"]
	if !ok || addr.Type != "ip" {
		t.Fatalf("address node = %#v, want an ip-type node for 203.0.113.5", addr)
	}
	if addr.Ports != ":443" {
		t.Errorf("address open ports = %q, want :443", addr.Ports)
	}
	svc, ok := byID["203.0.113.5:443/tcp"]
	if !ok || svc.Type != "service" {
		t.Fatalf("service node = %#v, want a service-type node", svc)
	}
	if svc.Label != ":443 tcp" {
		t.Errorf("service label = %q, want :443 tcp", svc.Label)
	}
	// buildGraph alone joins no signals — the topology invents no signal state, no
	// severity, no count (the join is a separate fold, joinSignals).
	for _, n := range g.Nodes {
		if len(n.OpenSignals) != 0 {
			t.Errorf("node %q carries %d fabricated signals; buildGraph must invent none", n.ID, len(n.OpenSignals))
		}
	}

	// Two edges: Name -> Address (structural stroke) and Address -> Service (the
	// quieter service stroke).
	if len(g.Edges) != 2 {
		t.Fatalf("edges = %d, want 2 (name->address, address->service); %#v", len(g.Edges), g.Edges)
	}
	var sawToService bool
	for _, e := range g.Edges {
		if e.ToService {
			sawToService = true
		}
	}
	if !sawToService {
		t.Errorf("no address->service edge marked ToService; edges %#v", g.Edges)
	}
}

// joinSignals folds the Signal engine's fired census onto the graph's nodes by the
// honest subject→node mapping: a Name firing lights its Name node, a Service firing
// lights its Service node, and an Endpoint firing lights the Name node it names
// (falling back to its Service node when nameless). Addresses carry none. Each node
// also folds to its worst (most urgent) severity — the real severity its fired rules
// carry (P0.1), never a fabricated level.
// #1089: the three columns run unbounded (y = 44 + idx*46), so a real estate reaches
// far below the 1200x640 viewport. The minimap scaled the VIEWPORT box into its
// 110x59 SVG, so every node past ~row 13 mapped outside the SVG and was clipped —
// the minimap showed a handful of dots and represented almost nothing. The mapping
// now reads the content bounds, so every placed node lands inside the mini box.
func TestGraphMinimapMapsEveryNodeInsideTheMiniBox(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 40; i++ {
		name := fmt.Sprintf("n%02d.example.com", i)
		addr := fmt.Sprintf("203.0.113.%d", i+1)
		rows = append(rows, openSpanRow("name", name, "resolution", "",
			fmt.Sprintf(`{"outcome":"Resolved","addresses":[%q]}`, addr), false))
	}

	g := buildGraph(rows)
	if len(g.Nodes) != 80 {
		t.Fatalf("nodes = %d, want 80 (40 names, 40 addresses)", len(g.Nodes))
	}

	var tallest int
	for _, n := range g.Nodes {
		if n.Y > tallest {
			tallest = n.Y
		}
	}
	if tallest <= g.ViewH {
		t.Fatalf("tallest node y = %d, want past the %dpx viewport for this case to bite", tallest, g.ViewH)
	}
	if g.ContentH <= tallest {
		t.Errorf("ContentH = %d, want past the tallest node at y = %d", g.ContentH, tallest)
	}
	if g.ContentW != g.ViewW {
		t.Errorf("ContentW = %d, want the viewport width %d (the columns are fixed)", g.ContentW, g.ViewW)
	}

	for _, n := range g.Nodes {
		if n.Mx < 0 || n.Mx > float64(g.MiniW) {
			t.Errorf("node %q minimap x = %v, outside the 0..%d mini box", n.ID, n.Mx, g.MiniW)
		}
		if n.My < 0 || n.My > float64(g.MiniH) {
			t.Errorf("node %q minimap y = %v, outside the 0..%d mini box", n.ID, n.My, g.MiniH)
		}
	}
}

// A graph that fits the viewport keeps the viewport as its content box, so its
// minimap mapping is exactly what it was before #1089.
func TestGraphContentBoundsFloorAtTheViewport(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	}

	g := buildGraph(rows)
	if g.ContentW != g.ViewW || g.ContentH != g.ViewH {
		t.Fatalf("content box = %dx%d, want the viewport box %dx%d for a graph that fits",
			g.ContentW, g.ContentH, g.ViewW, g.ViewH)
	}

	byID := map[string]graphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	name := byID["api.example.com"]
	if name.Mx != round1(float64(name.X)*graphMiniW/graphViewW) ||
		name.My != round1(float64(name.Y)*graphMiniH/graphViewH) {
		t.Errorf("name node minimap point = (%v,%v), want the unchanged viewport-basis mapping", name.Mx, name.My)
	}
}

func TestJoinSignalsToGraph(t *testing.T) {
	rows := []db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	}
	g := buildGraph(rows)

	// A fired census on each subject kind, plus one firing that matches no node.
	censuses := []signal.Census{
		{Rule: "lame-delegation", Fired: []signal.Member{{Subject: "api.example.com"}}},                             // name -> name node
		{Rule: "sensitive-port-reached-from-internet", Fired: []signal.Member{{Subject: "203.0.113.5:443/tcp"}}},    // service -> service node
		{Rule: "plaintext-http-no-https", Fired: []signal.Member{{Subject: "api.example.com@203.0.113.5:443/tcp"}}}, // endpoint -> its Name node
		{Rule: "redirect-does-not-upgrade-to-tls", Fired: []signal.Member{{Subject: "@203.0.113.5:443/tcp"}}},       // nameless endpoint -> its Service node
		{Rule: "cname-target-name-error", Fired: []signal.Member{{Subject: "ghost.example.com"}}},                   // matches no node -> dropped
	}

	g = joinSignals(g, censuses)
	byID := map[string]graphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}

	// The Name node carries its own name firing AND the endpoint firing named on it.
	name := byID["api.example.com"]
	if len(name.OpenSignals) != 2 {
		t.Fatalf("name node open signals = %d, want 2 (lame-delegation + the endpoint's plaintext-http); %#v", len(name.OpenSignals), name.OpenSignals)
	}
	var sawEndpoint bool
	for _, s := range name.OpenSignals {
		if s.Rule == "plaintext-http-no-https" {
			sawEndpoint = true
			if s.Subject != "api.example.com@203.0.113.5:443/tcp" {
				t.Errorf("endpoint firing subject = %q, want the endpoint key (the firing's real subject, not the node)", s.Subject)
			}
		}
	}
	if !sawEndpoint {
		t.Errorf("name node missing the endpoint firing attached to its Name leg; %#v", name.OpenSignals)
	}
	// Its worst severity is medium — both lame-delegation and plaintext-http-no-https
	// are medium — the token the Name node's halo tints to.
	if name.Sev != "medium" {
		t.Errorf("name node severity = %q, want medium (worst of its fired rules)", name.Sev)
	}

	// The Service node carries its own service firing AND the nameless endpoint's.
	svc := byID["203.0.113.5:443/tcp"]
	if len(svc.OpenSignals) != 2 {
		t.Fatalf("service node open signals = %d, want 2 (service rule + the nameless endpoint); %#v", len(svc.OpenSignals), svc.OpenSignals)
	}
	// Worst severity is critical — sensitive-port-reached-from-internet outranks the
	// nameless endpoint's low redirect rule — so the service fills to the critical dot.
	if svc.Sev != "critical" {
		t.Errorf("service node severity = %q, want critical (worst of its fired rules)", svc.Sev)
	}
	var sawSev bool
	for _, s := range svc.OpenSignals {
		if s.Rule == "sensitive-port-reached-from-internet" && s.Severity == "critical" {
			sawSev = true
		}
	}
	if !sawSev {
		t.Errorf("service firing missing its real per-rule severity; %#v", svc.OpenSignals)
	}

	// The Address node carries none — no rule censuses an Address, and a Service's
	// firing is not silently rolled up to the Address it rides.
	if addr := byID["203.0.113.5"]; len(addr.OpenSignals) != 0 {
		t.Errorf("address node open signals = %d, want 0 (addresses carry no signals); %#v", len(addr.OpenSignals), addr.OpenSignals)
	}

	// A firing whose subject names no built node is dropped, never invented as a node.
	if _, ok := byID["ghost.example.com"]; ok {
		t.Errorf("a firing on an absent subject invented a node; nodes must come only from the topology")
	}
}

// A named endpoint firing whose Name node is NOT in the topology (only the service
// span is open, so no name node was placed) falls back to its Service node rather
// than vanishing — a real open signal must always light the one node it can reach.
func TestJoinSignalsEndpointFallsBackToServiceWhenNameAbsent(t *testing.T) {
	// A service-only estate: the Service node and its Address exist, but there is no
	// open name span, so no Name node for www.example.com is placed.
	g := buildGraph([]db.ListAllOpenSpansRow{
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	})
	censuses := []signal.Census{
		{Rule: "plaintext-http-no-https", Fired: []signal.Member{{Subject: "www.example.com@203.0.113.5:443/tcp"}}},
	}
	g = joinSignals(g, censuses)

	byID := map[string]graphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	if _, ok := byID["www.example.com"]; ok {
		t.Fatalf("no Name node should exist for a service-only estate")
	}
	svc := byID["203.0.113.5:443/tcp"]
	if len(svc.OpenSignals) != 1 || svc.OpenSignals[0].Rule != "plaintext-http-no-https" {
		t.Errorf("named endpoint firing with an absent Name node did not fall back to its Service node; svc signals = %#v", svc.OpenSignals)
	}
}

// The Graph page joins real open signals onto its nodes with their severity (P2.3):
// a fired rule reaches the selected node's drawer with its SeverityBadge, its node
// draws a severity-tinted halo, and the header carries the five-level severity filter.
func TestGraphPageJoinsOpenSignals(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// An internal address leaking into a public answer fires
	// non-globally-reachable-address-resolved-from-internet on the Name.
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	for _, want := range []string{
		"non-globally-reachable-address-resolved-from-internet", // the fired rule reaches the drawer
		`data-for="leak.example.com"`,                           // the per-node drawer signal block
		`class="gnode-halo"`,                                    // a halo is drawn
		"var(--sev-medium-dot)",                                 // tinted to the rule's real severity (medium)
		"var(--sev-medium-bg)",                                  // the landed sevbadge for the firing (medium)
		`data-sev="critical"`,                                   // a five-level severity filter option
		"All severities",                                        // the severity filter's default
		"No open signals on this node.",                         // the honest empty state for unlit nodes
	} {
		if !strings.Contains(page, want) {
			t.Errorf("graph page missing %q; body: %s", want, page)
		}
	}
	// The retired presence filter must not linger — P2.3 replaces it with severity.
	if strings.Contains(page, `value="with"`) {
		t.Errorf("graph page still renders the presence filter; want the five-level severity filter")
	}
}

// The Graph page reads the estate's open spans and renders the real topology into
// a pannable canvas with a minimap, controls, a legend, and a node drawer, keying
// the Graph nav pill active.
func TestGraphPageRendersTopology(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	f.addReachability(t, "203.0.113.5:443/tcp", obsClock, `{"outcome":"reached"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	for _, want := range []string{
		`id="gr-svg"`,                      // the canvas
		`id="gr-minimap"`,                  // the minimap
		`data-gr-zoom="in"`,                // the pan/zoom controls
		`id="gr-drawer"`,                   // the node drawer
		"var CW =  1200 , CH =  640 ;",     // the minimap's content basis (#1089)
		"api.example.com",                  // the real Name node
		"203.0.113.5",                      // the real Address node
		":443 tcp",                         // the real Service node label
		`class="sh-pill on" href="/graph"`, // NavActive wired to graph
	} {
		if !strings.Contains(page, want) {
			t.Errorf("graph page missing %q; body: %s", want, page)
		}
	}
}

// With no subject measured into the estate, the graph shows the design-system
// empty-state rather than an empty or fabricated canvas.
func TestGraphPageEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if !strings.Contains(page, "Nothing to plot yet") {
		t.Errorf("empty graph missing the empty-state; body: %s", page)
	}
	if strings.Contains(page, `id="gr-svg"`) {
		t.Errorf("empty graph rendered a canvas; want only the empty-state")
	}
}

func TestGraphRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/graph")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /graph: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
