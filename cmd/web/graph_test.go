package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
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
	// No severity is fabricated — per-node signal status is not joined to the graph.
	for _, n := range g.Nodes {
		if n.Sev != "" {
			t.Errorf("node %q carries fabricated severity %q; want none", n.ID, n.Sev)
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
		`id="graph-svg"`,      // the canvas
		`id="graph-minimap"`,  // the minimap
		`id="graph-controls"`, // the pan/zoom controls
		`id="graph-drawer"`,   // the node drawer
		"api.example.com",     // the real Name node
		"203.0.113.5",         // the real Address node
		":443 tcp",            // the real Service node label
		`class="navpill active" href="/graph"`, // NavActive wired to graph
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
	if strings.Contains(page, `id="graph-svg"`) {
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
