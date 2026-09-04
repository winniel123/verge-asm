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
	for _, n := range g.Nodes {
		if len(n.OpenSignals) != 0 {
			t.Errorf("node %q carries %d fabricated signals; buildGraph must invent none", n.ID, len(n.OpenSignals))
		}
	}

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

func TestGraphMinimapMapsEveryNodeInsideTheMiniBox(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 40; i++ {
		name := fmt.Sprintf("n%02d.example.com", i)
		addr := fmt.Sprintf("203.0.113.%d", i+1)
		rows = append(rows, openSpanRow("name", name, "resolution", "",
			fmt.Sprintf(`{"outcome":"Resolved","addresses":[%q]}`, addr), false))
	}

	g := buildGraph(rows)
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

	censuses := []signal.Census{
		{Rule: "lame-delegation", Fired: []signal.Member{{Subject: "api.example.com"}}},
		{Rule: "sensitive-port-reached-from-internet", Fired: []signal.Member{{Subject: "203.0.113.5:443/tcp"}}},
		{Rule: "plaintext-http-no-https", Fired: []signal.Member{{Subject: "api.example.com@203.0.113.5:443/tcp"}}},
		{Rule: "redirect-does-not-upgrade-to-tls", Fired: []signal.Member{{Subject: "@203.0.113.5:443/tcp"}}},
		{Rule: "cname-target-name-error", Fired: []signal.Member{{Subject: "ghost.example.com"}}},
	}

	g = joinSignals(g, censuses)
	byID := map[string]graphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}

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
	if name.Sev != "medium" {
		t.Errorf("name node severity = %q, want medium (worst of its fired rules)", name.Sev)
	}

	svc := byID["203.0.113.5:443/tcp"]
	if len(svc.OpenSignals) != 2 {
		t.Fatalf("service node open signals = %d, want 2 (service rule + the nameless endpoint); %#v", len(svc.OpenSignals), svc.OpenSignals)
	}
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

	if addr := byID["203.0.113.5"]; len(addr.OpenSignals) != 0 {
		t.Errorf("address node open signals = %d, want 0 (addresses carry no signals); %#v", len(addr.OpenSignals), addr.OpenSignals)
	}

	if _, ok := byID["ghost.example.com"]; ok {
		t.Errorf("a firing on an absent subject invented a node; nodes must come only from the topology")
	}
}

func TestJoinSignalsEndpointFallsBackToServiceWhenNameAbsent(t *testing.T) {
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

func TestGraphPageJoinsOpenSignals(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	for _, want := range []string{
		"non-globally-reachable-address-resolved-from-internet",
		`data-for="leak.example.com"`,
		`class="gnode-halo"`,
		"var(--sev-medium-dot)",
		"var(--sev-medium-bg)",
		`data-sev="critical"`,
		"All severities",
		"No open signals on this node.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("graph page missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, `value="with"`) {
		t.Errorf("graph page still renders the presence filter; want the five-level severity filter")
	}
}

func TestGraphPageRendersTopology(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	f.addReachability(t, "203.0.113.5:443/tcp", obsClock, `{"outcome":"reached"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	for _, want := range []string{
		`id="gr-svg"`,
		`id="gr-minimap"`,
		`data-gr-zoom="in"`,
		`id="gr-drawer"`,
		"var CW =  1200 , CH =  640 ;",
		`transform="translate(0,0) scale(1)"`,
		"var MINK =  0.5 ,",
		"api.example.com",
		"203.0.113.5",
		":443 tcp",
		`class="sh-pill on" href="/graph"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("graph page missing %q; body: %s", want, page)
		}
	}
}

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
	// The cap keeps a built drawing above the zoom floor, so no fixture here can reach it (#1103).
}

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

	for _, n := range g.Nodes {
		switch n.Type {
		case "domain", "subdomain", "ip", "service":
		default:
			t.Errorf("node %q has type %q; the cap must never mint a rollup node", n.ID, n.Type)
		}
	}

	wantH := graphRowTop + (graphColumnCap-1)*graphRowStep + graphRadius("subdomain") + graphPad
	if g.ContentH != wantH {
		t.Errorf("ContentH = %d, want %d — the capped drawing's own bounds, not the 30 it holds", g.ContentH, wantH)
	}
}

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

	if g.CutEdges != 5 {
		t.Errorf("CutEdges = %d, want 5 (one per name the cap left out)", g.CutEdges)
	}
	if len(g.Edges) != graphColumnCap {
		t.Errorf("edges = %d, want %d (one per drawn name)", len(g.Edges), graphColumnCap)
	}
}

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

func TestJoinSignalsDoesNotFallBackToTheServiceLegForACappedName(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 25; i++ {
		rows = append(rows, openSpanRow("name", fmt.Sprintf("n%02d.example.com", i), "resolution", "",
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false))
	}
	rows = append(rows, openSpanRow("service", "203.0.113.5:443/tcp", "reachability", "", `{"outcome":"reached"}`, false))

	g := buildGraph(rows)
	g = joinSignals(g, []signal.Census{
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

func TestJoinSignalsCountsOnlyWhatTheCapDeleted(t *testing.T) {
	var rows []db.ListAllOpenSpansRow
	for i := 0; i < 25; i++ {
		rows = append(rows, openSpanRow("name", fmt.Sprintf("n%02d.example.com", i), "resolution", "",
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`, false))
	}

	g := buildGraph(rows)
	g = joinSignals(g, []signal.Census{
		{Rule: "lame-delegation", Fired: []signal.Member{
			{Subject: "n20.example.com"},
			{Subject: "n24.example.com"},
			{Subject: "ghost.example.com"},
			{Subject: "n00.example.com"},
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
