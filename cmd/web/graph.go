package main

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The Graph screen (#284, canonical `/graph`, ported from
// design-system/examples/console/GraphView.jsx). It draws the estate as a graph:
// Names resolve to Addresses, and a Service rides the Address it is keyed on, so
// the three tiers and their edges are read straight off the open-span corpus the
// Inventory screen already reads (ListAllOpenSpans, ADR-0105) — no fabricated
// topology. Where the corpus holds nothing, the design-system empty-state shows.
//
// Open signals ARE now joined onto the nodes (#289), off the same Signal engine
// the Reports and Signals screens read (buildSignalCorpus → signal.EvaluateCorpus).
// The example draws severity halos over a scored model — but signals in this product
// have NO severity (internal/signal: a rule "has no lifecycle of its own and no
// severity"; the census "is deliberately not a severity ramp"). So the example's
// severity affordances are re-skinned to honest signal-PRESENCE, exactly as
// reports.go re-skins its mock severity regions under ADR-0024/ADR-0110 — never a
// fabricated level, badge or gradient:
//   - each node carries the list of rules that FIRED for it (OpenSignals), joined
//     by exact subject-key match: a Name firing lights its Name node, a Service
//     firing lights its Service node, and an Endpoint firing (subject
//     `name@address:port/transport`) lights the Name node it names — falling back
//     to its Service node when the endpoint is nameless — so each firing lands on
//     exactly one node, never double-counted (see joinSignals). Addresses carry no
//     signals: no rule censuses an Address, and a Service's firing is the Service's,
//     not silently rolled up onto the Address it rides.
//   - the halo and minimap dot mark PRESENCE (≥1 open signal), a single --warn
//     token, not a five-level ramp; the drawer lists the fired rules (or an honest
//     "No open signals"); the filter is the presence axis (all / with / without),
//     not the inert severity Select.
//
// One example read is still NOT wired, and is not faked: a "domain" apex node — the
// example hand-classifies a registrable apex; the corpus does not, and deriving one
// by suffix would be a heuristic guess, so every Name renders as a subdomain-style
// node rather than inventing a root.
//
// Every field that IS shown is real: the node keys, the resolution and service
// edges, an Address's open ports, each node's earliest open-span instant, and each
// fired rule's own name and subject.

const (
	graphViewW   = 1200
	graphViewH   = 640
	graphColName = 130  // Names column x
	graphColAddr = 560  // Addresses column x
	graphColSvc  = 1000 // Services column x
	graphRowTop  = 44
	graphRowStep = 46
	graphMiniW   = 110
	graphMiniH   = 59
)

// graphSignal is one open (fired) signal joined to a node: the rule that fired and
// the exact subject it fired on. Subject may be more specific than the node it
// lights — an Endpoint firing attached to its Name node names the endpoint — so the
// drawer can show the finer subject where it differs. It carries NO severity or
// level: a signal in this product has none (see file header).
type graphSignal struct {
	Rule    string
	Subject string
}

// graphNode is one placed node: its key (the drawer's identity), display label,
// tier type (subdomain | ip | service), canvas and minimap coordinates, the
// label's x-offset (past the node's own radius), the presence-halo radius, the
// Address's open ports where it has them, the earliest instant an open span placed
// it, and the open signals joined to it. OpenSignals is a signal-PRESENCE list, not
// a severity — a node with ≥1 entry is marked, one with none is unmarked; there is
// no level, gradient or count-driven ramp (see file header).
type graphNode struct {
	ID          string
	Label       string
	Type        string
	X, Y        int
	LabelDX     int
	HaloR       int
	Mx, My      int
	Ports       string
	First       string
	OpenSignals []graphSignal
}

// graphEdge is one edge with its endpoints pre-resolved to canvas coordinates, so
// the template needs no id lookup. ToService picks the quieter stroke the example
// uses for the Address to Service leg.
type graphEdge struct {
	X1, Y1, X2, Y2 int
	ToService      bool
}

// graphView is the whole rendered graph: the placed nodes and resolved edges, the
// empty marker (no node has an open span to place), and the canvas/minimap dims
// the template and its script read back.
type graphView struct {
	Nodes                      []graphNode
	Edges                      []graphEdge
	Empty                      bool
	ViewW, ViewH, MiniW, MiniH int
}

// graphRadius is a node type's drawn radius, mirroring the example's NODE_R.
func graphRadius(typ string) int {
	switch typ {
	case "ip":
		return 9
	case "service":
		return 6
	default: // subdomain
		return 10
	}
}

// buildGraph folds the estate's open spans into the graph's nodes and edges. It
// reads only relationships the corpus states: a Name's resolution addresses (the
// Name to Address edge) and a Service key's Address (the Address to Service edge).
// Nodes are laid out in three columns — Names, Addresses, Services — each sorted
// so the layout is deterministic. It never invents a node, an edge, or a severity.
func buildGraph(rows []db.ListAllOpenSpansRow) graphView {
	// earliest open-span instant per internal node id.
	first := map[string]time.Time{}
	noteFirst := func(id string, t time.Time) {
		if cur, ok := first[id]; !ok || t.Before(cur) {
			first[id] = t
		}
	}

	names := map[string]struct{}{}
	addrs := map[string]struct{}{}
	services := map[string]struct{}{}
	svcLabel := map[string]string{}               // service key -> ":443 tcp"
	addrPorts := map[string]map[string]struct{}{} // address -> set of ":443"

	type edge struct {
		from, to string
		toSvc    bool
	}
	seen := map[edge]struct{}{}
	var order []edge
	addEdge := func(e edge) {
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		order = append(order, e)
	}

	for _, row := range rows {
		at := row.OpenedAt.Time.UTC()
		switch row.SubjectKind {
		case "name":
			names[row.SubjectKey] = struct{}{}
			noteFirst("name:"+row.SubjectKey, at)
			if row.Facet == "resolution" && !row.IsGap {
				for _, a := range decodeResolution(row.Value).Addresses {
					addrs[a] = struct{}{}
					noteFirst("addr:"+a, at)
					addEdge(edge{from: "name:" + row.SubjectKey, to: "addr:" + a})
				}
			}
		case "service":
			services[row.SubjectKey] = struct{}{}
			noteFirst("svc:"+row.SubjectKey, at)
			if pair, addr, ok := parseServicePair(row.SubjectKey); ok {
				addrs[addr] = struct{}{}
				noteFirst("addr:"+addr, at)
				port := ":" + strconv.Itoa(int(pair.Port))
				if addrPorts[addr] == nil {
					addrPorts[addr] = map[string]struct{}{}
				}
				addrPorts[addr][port] = struct{}{}
				svcLabel[row.SubjectKey] = port + " " + string(pair.Transport)
				addEdge(edge{from: "addr:" + addr, to: "svc:" + row.SubjectKey, toSvc: true})
			} else {
				svcLabel[row.SubjectKey] = row.SubjectKey
			}
		}
	}

	pos := map[string]struct{ x, y int }{}
	var nodes []graphNode

	place := func(internalID, id, label, typ string, col, idx int, ports string) {
		x := col
		y := graphRowTop + idx*graphRowStep
		pos[internalID] = struct{ x, y int }{x, y}
		n := graphNode{
			ID: id, Label: label, Type: typ, X: x, Y: y,
			LabelDX: graphRadius(typ) + 9,
			HaloR:   graphRadius(typ) + 6,
			Mx:      x * graphMiniW / graphViewW,
			My:      y * graphMiniH / graphViewH,
			Ports:   ports,
		}
		if t, ok := first[internalID]; ok {
			n.First = t.Format(spanTimeFmt)
		}
		nodes = append(nodes, n)
	}

	for i, k := range sortedSet(names) {
		place("name:"+k, k, k, "subdomain", graphColName, i, "—")
	}
	for i, a := range sortedSet(addrs) {
		place("addr:"+a, a, a, "ip", graphColAddr, i, joinPorts(addrPorts[a]))
	}
	for i, k := range sortedSet(services) {
		place("svc:"+k, k, svcLabel[k], "service", graphColSvc, i, "—")
	}

	var edges []graphEdge
	for _, e := range order {
		a, ok1 := pos[e.from]
		b, ok2 := pos[e.to]
		if !ok1 || !ok2 {
			continue
		}
		edges = append(edges, graphEdge{X1: a.x, Y1: a.y, X2: b.x, Y2: b.y, ToService: e.toSvc})
	}

	return graphView{
		Nodes: nodes, Edges: edges, Empty: len(nodes) == 0,
		ViewW: graphViewW, ViewH: graphViewH, MiniW: graphMiniW, MiniH: graphMiniH,
	}
}

// joinSignals lights each graph node with the open (fired) signals whose subject
// resolves to it, folding a Census set from the Signal engine onto the built
// topology WITHOUT inventing severity — a node gains a presence list, never a level.
// The subject→node mapping is exact and honest:
//   - a Name-rule firing (subject = the Name) lights the Name node of that key;
//   - a Service-rule firing (subject = `address:port/transport`) lights the Service
//     node of that key;
//   - an Endpoint-rule firing (subject = `name@address:port/transport`) lights the
//     Name node named before the `@` — the endpoint's own DNS identity — falling
//     back to its Service node (after the `@`) when the endpoint is nameless OR its
//     Name node is not in the topology (only the service span is open), so each
//     firing lands on exactly one node it can reach and is never double-counted.
//
// Address nodes gain nothing: no rule censuses an Address subject, and a Service's
// firing is the Service's, not silently aggregated onto the Address it rides
// (rolling it up would assert a signal on a subject the engine never censused). A
// firing whose subject names no built node is dropped — the graph lights only nodes
// the topology already holds, never inventing one from a verdict.
func joinSignals(g graphView, censuses []signal.Census) graphView {
	idx := make(map[string]int, len(g.Nodes))
	for i, n := range g.Nodes {
		idx[n.ID] = i
	}
	attach := func(nodeID string, sig graphSignal) bool {
		if i, ok := idx[nodeID]; ok {
			g.Nodes[i].OpenSignals = append(g.Nodes[i].OpenSignals, sig)
			return true
		}
		return false
	}
	for _, c := range censuses {
		kind := signal.SubjectKindFor(c.Rule)
		for _, m := range c.Fired {
			sig := graphSignal{Rule: c.Rule, Subject: m.Subject}
			switch kind {
			case "name", "service":
				attach(m.Subject, sig)
			case "endpoint":
				// name@service — light the Name leg (the endpoint's DNS identity),
				// falling back to the Service leg for a nameless endpoint or when the
				// Name node is not in the topology (so a real firing never vanishes).
				name, service := splitEndpointName(m.Subject)
				if name == "" || !attach(name, sig) {
					attach(service, sig)
				}
			}
		}
	}
	return g
}

// sortedSet returns a string set's members in ascending order, for a deterministic
// layout.
func sortedSet(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// joinPorts renders an Address's open ports as a space-separated list (":443 :22"),
// or an em dash where it has none.
func joinPorts(set map[string]struct{}) string {
	if len(set) == 0 {
		return "—"
	}
	ports := make([]string, 0, len(set))
	for p := range set {
		ports = append(ports, p)
	}
	sort.Strings(ports)
	out := ports[0]
	for _, p := range ports[1:] {
		out += " " + p
	}
	return out
}

// graphPage renders the Graph screen (#284). It reads the estate's open spans once
// — the same corpus the Inventory screen reads — and folds them into the graph's
// real Name/Address/Service topology, then joins the Signal engine's fired census
// onto the nodes (#289). An empty corpus renders the empty-state.
//
// The topology read is the page's spine; the signal-corpus read is heavier
// (buildSignalCorpus fans out over resolution / reachability / certificate /
// http-identity), so its failure DEGRADES — the graph renders without signal state
// rather than 500ing the topology a viewer depends on, exactly as reports.go
// degrades its KPI on a failed read.
func (s *server) graphPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	rows, err := s.store.ListAllOpenSpans(r.Context())
	if err != nil {
		s.serverError(w, "list all open spans", err)
		return
	}
	g := buildGraph(rows)
	if !g.Empty {
		corpus, err := s.buildSignalCorpus(r)
		if err != nil {
			log.Printf("web: graph: build signal corpus: %v", err)
		} else {
			g = joinSignals(g, signal.EvaluateCorpus(corpus))
		}
	}
	s.render(w, "graph", map[string]any{
		"Title": "Graph", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "graph",
		"Graph":     g,
	})
}
