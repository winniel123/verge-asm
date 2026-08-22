package main

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Graph screen (#284, canonical `/graph`, ported from
// design-system/examples/console/GraphView.jsx). It draws the estate as a graph:
// Names resolve to Addresses, and a Service rides the Address it is keyed on, so
// the three tiers and their edges are read straight off the open-span corpus the
// Inventory screen already reads (ListAllOpenSpans, ADR-0105) — no fabricated
// topology. Where the corpus holds nothing, the design-system empty-state shows.
//
// The composition is faithful to the example (pannable/zoomable canvas, minimap,
// zoom controls, node drawer, legend), but two of the example's per-node reads are
// NOT wired here because no console read exposes them yet, and neither is faked:
//   - severity halos / the drawer's per-node open-signal list — a node's open
//     signals would need the Signal engine's per-subject verdict joined to the
//     graph; not plumbed (documented as a follow-on on #284). Nodes therefore
//     carry no severity and the drawer states signal status is not yet joined.
//   - a "domain" apex node — the example hand-classifies a registrable apex; the
//     corpus does not, and deriving one by suffix would be a heuristic guess, so
//     every Name renders as a subdomain-style node rather than inventing a root.
//
// Every field that IS shown is real: the node keys, the resolution and service
// edges, an Address's open ports, and each node's earliest open-span instant.

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

// graphNode is one placed node: its key (the drawer's identity), display label,
// tier type (subdomain | ip | service), canvas and minimap coordinates, the
// label's x-offset (past the node's own radius), the Address's open ports where it
// has them, and the earliest instant an open span placed it. Sev is always empty
// — per-node severity is not yet joined to the graph (see file header).
type graphNode struct {
	ID      string
	Label   string
	Type    string
	X, Y    int
	LabelDX int
	Mx, My  int
	Ports   string
	First   string
	Sev     string
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
// real Name/Address/Service topology; an empty corpus renders the empty-state.
func (s *server) graphPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	rows, err := s.store.ListAllOpenSpans(r.Context())
	if err != nil {
		s.serverError(w, "list all open spans", err)
		return
	}
	s.render(w, "graph", map[string]any{
		"Title": "Graph", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "graph",
		"Graph":     buildGraph(rows),
	})
}
