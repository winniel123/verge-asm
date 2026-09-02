package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
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
	// #1103 caps each column at graphColumnCap, so 40 names and 40 addresses draw as
	// two capped columns. Twenty rows still reach y=918, past the 640px viewport, so
	// the viewport-basis mapping this test guards against still fails here.
	if len(g.Nodes) != 2*graphColumnCap {
		t.Fatalf("nodes = %d, want %d (a capped name column and a capped address column)", len(g.Nodes), 2*graphColumnCap)
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
		`id="gr-svg"`,                         // the canvas
		`id="gr-minimap"`,                     // the minimap
		`data-gr-zoom="in"`,                   // the pan/zoom controls
		`id="gr-drawer"`,                      // the node drawer
		"var CW =  1200 , CH =  640 ;",        // the minimap's content basis (#1089)
		`transform="translate(0,0) scale(1)"`, // this estate fits, so the fit is the standing origin (#1101)
		"var MINK =  0.5 ,",                   // and the zoom floor is the standing one (#1101)
		"api.example.com",                     // the real Name node
		"203.0.113.5",                         // the real Address node
		":443 tcp",                            // the real Service node label
		`class="sh-pill on" href="/graph"`,    // NavActive wired to graph
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
	if strings.Contains(page, "var MINK") {
		t.Errorf("empty graph emitted the pan/zoom view JS; want none without a canvas")
	}
}

// #1101: the drawing's height is unbounded, so a large estate reaches far past the
// 1200x640 viewport. The fit frames the whole content box inside the viewport and
// centres it, and the zoom floor drops to reach that scale.
func TestGraphFitFramesTheWholeContent(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 40; i++ {
		rows = append(rows, openSpanRow("name", fmt.Sprintf("n%02d.example.com", i), "resolution", "",
			fmt.Sprintf(`{"outcome":"Resolved","addresses":["203.0.113.%d"]}`, i+1), false))
	}

	g := buildGraph(rows)
	if g.ContentH <= g.ViewH {
		t.Fatalf("content height = %d, want past the %dpx viewport for this case to bite", g.ContentH, g.ViewH)
	}
	if g.FitK >= 1 {
		t.Fatalf("fit scale = %v, want below 1 for content taller than the viewport", g.FitK)
	}
	if w := float64(g.ContentW) * g.FitK; w > float64(g.ViewW) {
		t.Errorf("fitted content width = %v, want no wider than the %dpx viewport", w, g.ViewW)
	}
	if h := float64(g.ContentH) * g.FitK; h > float64(g.ViewH) {
		t.Errorf("fitted content height = %v, want no taller than the %dpx viewport", h, g.ViewH)
	}
	if want := (float64(g.ViewW) - float64(g.ContentW)*g.FitK) / 2; math.Abs(g.FitX-want) > 0.1 {
		t.Errorf("fit x = %v, want the centring offset %v", g.FitX, want)
	}
	if want := (float64(g.ViewH) - float64(g.ContentH)*g.FitK) / 2; math.Abs(g.FitY-want) > 0.1 {
		t.Errorf("fit y = %v, want the centring offset %v", g.FitY, want)
	}
	// #1103 caps each column, so a built drawing no longer runs deep enough to push the
	// fit under the standing 0.5 floor. TestGraphFitHoldsAtAnyContentHeight covers the
	// floor against a content box of any depth.
}

// An estate whose content fits the viewport frames exactly as it did before #1101:
// the origin at scale 1, with the zoom floor left at its standing 0.5.
func TestGraphFitIsUnchangedWhenTheContentFits(t *testing.T) {
	g := buildGraph([]db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
		openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false),
	})
	if g.FitK != 1 || g.FitX != 0 || g.FitY != 0 {
		t.Errorf("fit = translate(%v,%v) scale(%v), want the unchanged origin at scale 1", g.FitX, g.FitY, g.FitK)
	}
	if g.MinK != graphZoomFloor {
		t.Errorf("zoom floor = %v, want the standing %v for a graph that fits", g.MinK, graphZoomFloor)
	}
}

// The column run has no cap, so the content box can reach any height. The fit stays
// positive and keeps its relative accuracy the whole way down: it never rounds to
// zero (scale(0) draws nothing) and never gives back a useful part of the viewport.
func TestGraphFitHoldsAtAnyContentHeight(t *testing.T) {
	for _, h := range []int{640, 1891, 47102, 5_000_000, 6_400_000, 640_000_000} {
		_, _, k, minK := graphFit(graphViewW, h)
		if k <= 0 {
			t.Errorf("content height %d: fit scale = %v, want above 0", h, k)
		}
		got := float64(h) * k
		if got > graphViewH {
			t.Errorf("content height %d: fitted height = %v, want no taller than the %dpx viewport", h, got, graphViewH)
		}
		if got < float64(graphViewH)*0.998 {
			t.Errorf("content height %d: fitted height = %v, want within 0.2%% of the %dpx viewport", h, got, graphViewH)
		}
		if minK > k {
			t.Errorf("content height %d: zoom floor = %v, want no higher than the fit scale %v", h, minK, k)
		}
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

// #1103 (ADR-0136 §2, §4, §6): each column draws at most graphColumnCap nodes, and
// the drawing states the shortfall rather than dropping subjects silently. The
// selection is the column's existing sorted order, first N — not severity, not
// recency — so an unchanged corpus draws the same set on every reload.
func TestGraphCapsEachColumnAtTheCap(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 30; i++ {
		rows = append(rows, openSpanRow("name", fmt.Sprintf("n%02d.example.com", i), "resolution", "",
			fmt.Sprintf(`{"outcome":"Resolved","addresses":["203.0.113.%d"]}`, i+1), false))
		rows = append(rows, openSpanRow("service", fmt.Sprintf("203.0.113.%d:443/tcp", i+1), "reachability", "",
			`{"outcome":"reached"}`, false))
	}

	g := buildGraph(rows)
	if g.Cap != graphColumnCap {
		t.Errorf("Cap = %d, want graphColumnCap %d", g.Cap, graphColumnCap)
	}

	drawn := map[string]int{}
	for _, n := range g.Nodes {
		drawn[n.Type]++
	}
	if got := drawn["subdomain"] + drawn["domain"]; got != graphColumnCap {
		t.Errorf("name column drew %d nodes, want the cap %d", got, graphColumnCap)
	}
	if drawn["ip"] != graphColumnCap {
		t.Errorf("address column drew %d nodes, want the cap %d", drawn["ip"], graphColumnCap)
	}
	if drawn["service"] != graphColumnCap {
		t.Errorf("service column drew %d nodes, want the cap %d", drawn["service"], graphColumnCap)
	}

	want := []graphColumnCount{
		{Label: "names", Drawn: graphColumnCap, Held: 30},
		{Label: "addresses", Drawn: graphColumnCap, Held: 30},
		{Label: "services", Drawn: graphColumnCap, Held: 30},
	}
	if len(g.Capped) != len(want) {
		t.Fatalf("Capped = %#v, want one entry per capped column %#v", g.Capped, want)
	}
	for i, w := range want {
		if g.Capped[i] != w {
			t.Errorf("Capped[%d] = %#v, want %#v", i, g.Capped[i], w)
		}
	}

	// No node is folded in: the drawing holds only the four Subject kinds it always
	// held, and no prefix or parent rollup stands for the ones the cap left out.
	for _, n := range g.Nodes {
		switch n.Type {
		case "domain", "subdomain", "ip", "service":
		default:
			t.Errorf("node %q has type %q; the cap must never mint a rollup node", n.ID, n.Type)
		}
	}

	// The bounds follow the placed nodes, so the minimap and the PNG export frame the
	// capped drawing rather than the population the cap left out.
	wantH := graphRowTop + (graphColumnCap-1)*graphRowStep + graphRadius("subdomain") + graphPad
	if g.ContentH != wantH {
		t.Errorf("ContentH = %d, want %d — the capped drawing's own bounds, not the 30 it holds", g.ContentH, wantH)
	}
}

// The cap takes the first N of the column's existing sorted order, so the drawn set
// is exactly the head of what an uncapped build would have drawn, in the same order.
func TestGraphCapTakesTheSortedHead(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 25; i++ {
		rows = append(rows, openSpanRow("name", fmt.Sprintf("n%02d.example.com", i), "resolution", "",
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false))
	}

	g := buildGraph(rows)
	var names []string
	for _, n := range g.Nodes {
		if n.Type == "subdomain" {
			names = append(names, n.ID)
		}
	}
	if len(names) != graphColumnCap {
		t.Fatalf("name column drew %d nodes, want the cap %d", len(names), graphColumnCap)
	}
	for i, id := range names {
		if want := fmt.Sprintf("n%02d.example.com", i); id != want {
			t.Errorf("name node %d = %q, want %q (the sorted head, first N)", i, id, want)
		}
	}

	// Every one of the 25 names resolves to the one address, so the five names the cap
	// left out take five edges with them. That deletion is counted, not silent.
	if g.CutEdges != 5 {
		t.Errorf("CutEdges = %d, want 5 (one per name the cap left out)", g.CutEdges)
	}
	if len(g.Edges) != graphColumnCap {
		t.Errorf("edges = %d, want %d (one per drawn name)", len(g.Edges), graphColumnCap)
	}
}

// A column at or under the cap is drawn exactly as it was, and the screen states
// nothing: no shortfall, no cut edge.
func TestGraphUnderTheCapStatesNothing(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < graphColumnCap; i++ {
		rows = append(rows, openSpanRow("name", fmt.Sprintf("n%02d.example.com", i), "resolution", "",
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false))
	}

	g := buildGraph(rows)
	if len(g.Capped) != 0 {
		t.Errorf("Capped = %#v, want none for a column exactly at the cap", g.Capped)
	}
	if g.CutEdges != 0 {
		t.Errorf("CutEdges = %d, want 0 for a drawing the cap did not touch", g.CutEdges)
	}
	if len(g.Nodes) != graphColumnCap+1 {
		t.Errorf("nodes = %d, want %d (every name plus the one address)", len(g.Nodes), graphColumnCap+1)
	}
	if len(g.Edges) != graphColumnCap {
		t.Errorf("edges = %d, want %d (every resolution edge)", len(g.Edges), graphColumnCap)
	}
}

// The screen states the shortfall per column and names a scope selection as the
// remedy, following the Drift feed, which states its 500-event truncation rather
// than dropping rows silently.
func TestGraphPageStatesTheCap(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	if _, err := f.CreateNameSeed(context.Background(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: "example.com", Valid: true}, CreatedBy: admin.ID,
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		f.addResolution(t, admin.ID, fmt.Sprintf("n%02d.example.com", i), "dns", obsClock,
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	for _, want := range []string{
		`class="gr-callout"`,
		"Showing 20 of 25 names.",
		"The graph draws at most 20 nodes per column.",
		"5 edges reach a node it left out and are not drawn.",
		"Pick a scope to bound the drawing to one declared seed.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("graph page missing the cap statement %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, "n19.example.com") || strings.Contains(page, "n20.example.com") {
		t.Errorf("graph page drew the wrong head of the name column; body: %s", page)
	}
}

// A drawing the cap did not touch carries no callout at all.
func TestGraphPageStatesNoCapWhenNoneApplied(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if strings.Contains(page, `class="gr-callout"`) {
		t.Errorf("graph page stated a cap it did not apply; body: %s", page)
	}
}

// #1103: the endpoint fallback answers a nameless or unmeasured endpoint. It must not
// answer a Name node the CAP dropped: re-attributing that firing to the Service leg
// would light a node for a reason that is about the drawing being full, and the
// drawer would name a Name the drawing does not hold. The firing is counted instead,
// exactly as #1102 counts a scope that excludes the name.
func TestJoinSignalsDoesNotFallBackToTheServiceLegForACappedName(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 25; i++ {
		rows = append(rows, openSpanRow("name", fmt.Sprintf("n%02d.example.com", i), "resolution", "",
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false))
	}
	rows = append(rows, openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false))

	g := buildGraph(rows)
	g = joinSignals(g, []signal.Census{
		// n24 is name 25 of 25, so the cap left it out. n00 is drawn.
		{Rule: "plaintext-http-no-https", Fired: []signal.Member{
			{Subject: "n24.example.com@203.0.113.5:443/tcp"},
			{Subject: "n00.example.com@203.0.113.5:443/tcp"},
		}},
	})

	byID := map[string]graphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	if svc := byID["203.0.113.5:443/tcp"]; len(svc.OpenSignals) != 0 {
		t.Errorf("service node carries %d signals; the capped name's firing must not move onto the Service leg: %#v",
			len(svc.OpenSignals), svc.OpenSignals)
	}
	if drawn := byID["n00.example.com"]; len(drawn.OpenSignals) != 1 {
		t.Errorf("drawn name node open signals = %d, want 1 (the cap must not disturb a node it kept)", len(drawn.OpenSignals))
	}
	if g.CutSignals != 1 {
		t.Errorf("CutSignals = %d, want 1 (the firing on the name the cap left out)", g.CutSignals)
	}
}

// A firing whose node the cap dropped is counted, so the screen states the deletion
// rather than losing a severity the operator would otherwise see. A firing whose node
// the corpus never held is NOT counted: the cap did not take it.
func TestJoinSignalsCountsOnlyWhatTheCapDeleted(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 25; i++ {
		rows = append(rows, openSpanRow("name", fmt.Sprintf("n%02d.example.com", i), "resolution", "",
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false))
	}

	g := buildGraph(rows)
	g = joinSignals(g, []signal.Census{
		{Rule: "lame-delegation", Fired: []signal.Member{
			{Subject: "n20.example.com"},   // held by the corpus, dropped by the cap
			{Subject: "n24.example.com"},   // held by the corpus, dropped by the cap
			{Subject: "ghost.example.com"}, // the corpus never held it
			{Subject: "n00.example.com"},   // drawn
		}},
	})

	if g.CutSignals != 2 {
		t.Errorf("CutSignals = %d, want 2 (only the firings the cap deleted)", g.CutSignals)
	}
	var lit int
	for _, n := range g.Nodes {
		lit += len(n.OpenSignals)
	}
	if lit != 1 {
		t.Errorf("lit signals = %d, want 1 (the one firing on a drawn node)", lit)
	}
}

// An estate the cap did not touch counts no deleted firing, so a firing that matches
// no node stays the silent drop it has always been.
func TestJoinSignalsCountsNothingUnderTheCap(t *testing.T) {
	g := buildGraph([]db.ListAllOpenSpansRow{
		openSpanRow("name", "api.example.com", "resolution", "", `{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false),
	})
	g = joinSignals(g, []signal.Census{
		{Rule: "lame-delegation", Fired: []signal.Member{{Subject: "ghost.example.com"}}},
	})
	if g.CutSignals != 0 {
		t.Errorf("CutSignals = %d, want 0 for a drawing the cap did not touch", g.CutSignals)
	}
}
