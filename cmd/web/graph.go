package main

import (
	"html/template"
	"log"
	"math"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The frozen design-owned graph.tmpl (design-system/templates/graph.tmpl, package
// v3.10.0, WORKFLOW v4) is the view layer for /graph: it defines "graph" and reuses
// the landed "sevbadge" define signals.tmpl declares (the drawer's sev-badged signal
// rows) — one parse set. It is embedded read-only via the designfs package
// (auto-globbed through `templates/*.tmpl`); the repo authors no markup/CSS/JS for
// this route (the pan/zoom/minimap/PNG-export engine and the severity-filter listbox
// are the design's own view JS, kept byte-for-byte). Replaces the repo-authored
// templates_graph.go, which is deleted with this conversion (#583).
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/graph.tmpl"))

// The Graph screen (#284, canonical `/graph`, ported from
// design-system/examples/console/GraphView.jsx). It draws the estate as a graph:
// Names resolve to Addresses, and a Service rides the Address it is keyed on, so
// the three tiers and their edges are read straight off the open-span corpus the
// Inventory screen already reads (ListAllOpenSpans, ADR-0105) — no fabricated
// topology. Where the corpus holds nothing, the design-system empty-state shows.
//
// Open signals ARE joined onto the nodes (#289), off the same Signal engine the
// Reports and Signals screens read (buildSignalCorpus → signal.EvaluateCorpus). The
// example draws SEVERITY halos, and every rule now carries a five-level severity
// (P0.1, internal/signal/severity.go), so the graph draws them for real — the old
// presence-only single-accent re-skin is retired (P2.3; the design is normative for
// look AND functionality, ADR-0116). No fabricated level: each node's severity is
// exactly the worst (most urgent) among the rules that actually fired on it.
//   - each node carries the list of rules that FIRED for it (OpenSignals), joined
//     by exact subject-key match: a Name firing lights its Name node, a Service
//     firing lights its Service node, and an Endpoint firing (subject
//     `name@address:port/transport`) lights the Name node it names — falling back
//     to its Service node when the endpoint is nameless — so each firing lands on
//     exactly one node, never double-counted (see joinSignals). Addresses carry no
//     signals: no rule censuses an Address, and a Service's firing is the Service's,
//     not silently rolled up onto the Address it rides.
//   - the halo and minimap dot are tinted to the node's worst severity (Sev, the
//     --sev-<level>-dot token), a service node fills to it, and the drawer lists
//     each fired rule with its own SeverityBadge (or an honest "No open signals");
//     the header Select is the five-level severity filter the spec renders — it
//     lights only the halos of nodes at the chosen level, matching GraphView.jsx.
//
// The "domain" apex node is derived, not faked: classifyNameTypes (below) reads the
// domain|subdomain tiers purely off the observed name set — a name is the apex when it
// parents another observed name and is itself parented by none (#22e) — with no
// public-suffix guess, so an estate whose apex was never measured simply has no domain
// node rather than one being invented.
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
	graphPad     = 44 // content-box margin past the outermost node

	graphZoomFloor = 0.5 // the standing zoom-out limit for a graph that fits the viewport
)

// graphSignal is one open (fired) signal joined to a node: the rule that fired, the
// exact subject it fired on, and the rule's severity (P0.1). Subject may be more
// specific than the node it lights — an Endpoint firing attached to its Name node
// names the endpoint — so the drawer can show the finer subject where it differs.
// Sev is one of the five ramp tokens (critical | high | medium | low | info); the
// drawer keys a SeverityBadge off it.
type graphSignal struct {
	Severity string
	SevLabel string
	Rule     string
	Subject  string
}

// graphNode is one placed node: its key (the drawer's identity), display label,
// tier type (subdomain | ip | service), canvas and minimap coordinates (Mx/My scale
// the canvas point into the minimap box against the CONTENT bounds, not the viewport:
// the column run is unbounded, so past ~14 rows a viewport-basis dot lands outside the
// minimap SVG and is clipped, #1089), the
// label's x-offset (past the node's own radius), the halo radius, the Address's open
// ports where it has them, the earliest instant an open span placed it, the open
// signals joined to it, and Sev — the node's worst (most urgent) severity among
// those signals, one of the five ramp tokens or "" when none fired. Sev tints the
// halo, the minimap dot and (for a service) the node fill, per GraphView.jsx.
type graphNode struct {
	ID          string
	Label       string
	Type        string
	X, Y        int
	LabelDX     int
	HaloA       float64
	HaloB       float64
	Mx, My      float64
	Ports       string
	First       string
	OpenSignals []graphSignal
	Sev         string
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
// the template and its script read back. ContentW/ContentH are the padded bounds of
// everything actually placed. They grow past ViewW/ViewH once a column runs longer
// than the viewport, and the minimap maps BOTH its dots and its viewport rectangle
// against them so the two agree (#1089). They never fall below ViewW/ViewH, so a
// graph that fits the viewport maps exactly as it did before.
// FitX/FitY/FitK frame that content box inside the viewport, and MinK is the zoom
// floor that reaches the fit (#1101).
// Scopes is the scope selector's vocabulary, Scope the selected ?scope token and
// ScopeLabel its label, so the control renders and marks its own selection (#1102).
type graphView struct {
	Nodes                      []graphNode
	Edges                      []graphEdge
	Empty                      bool
	ViewW, ViewH, MiniW, MiniH int
	ContentW, ContentH         int
	FitX, FitY, FitK           float64
	MinK                       float64
	Scopes                     []graphScope
	Scope                      string
	ScopeLabel                 string
	// members is the resolved membership the build narrowed by. joinSignals reads it
	// to tell a Name node the scope dropped from one the corpus never held. Zero
	// means the whole estate, so a graphView assembled by hand narrows nothing.
	members graphMembers
}

// graphScopeAll is the ?scope token for the whole estate — the drawing /graph has
// always rendered. It is the default when the token is absent or names no declared
// Seed (ADR-0136 §3).
const graphScopeAll = "all"

// graphScopeAllLabel is the whole-estate entry's label in the selector.
const graphScopeAllLabel = "Whole estate"

// graphScope is one entry of the graph's scope selector: the ?scope token the URL
// carries, the label the control renders, and the declared Seed the token names.
// A Seed entry sets exactly one of Domain and Prefix. The whole-estate entry sets
// neither, and its zero Domain and invalid Prefix are what mark it.
//
// The token is the Seed's own spelling — the domain, or the masked CIDR — so the
// URL names what the operator declared and a bookmarked link stays readable. The
// spelling SELECTS the Seed. It never decides membership: an address is inside an
// address scope by family-matched prefix comparison, and a name is inside a name
// scope by the label-wise suffix test custody.LabelSuffix owns, so neither answer
// turns on a rendering (CONTEXT.md `Seed`).
type graphScope struct {
	Token  string
	Label  string
	Domain string
	Prefix netip.Prefix
}

// graphScopes reads the declared Seeds into the selector's vocabulary: the whole
// estate first, then one entry per Seed in the order ListSeeds returns. A Seed row
// carrying neither a domain nor a CIDR names no population and is skipped.
func graphScopes(seeds []db.ListSeedsRow) []graphScope {
	out := []graphScope{{Token: graphScopeAll, Label: graphScopeAllLabel}}
	for _, s := range seeds {
		switch {
		case s.Kind == "name" && s.NameDomain.Valid:
			d := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s.NameDomain.String)), ".")
			if d == "" {
				continue
			}
			out = append(out, graphScope{Token: d, Label: d, Domain: d})
		// The validity check is not redundant with the nil check. An invalid prefix
		// spells "invalid Prefix" and bounds nothing, so an entry built from one would
		// label itself a scope and draw the whole estate.
		case s.Kind == "address" && s.AddressCidr != nil && s.AddressCidr.IsValid():
			p := s.AddressCidr.Masked()
			out = append(out, graphScope{Token: p.String(), Label: p.String(), Prefix: p})
		}
	}
	return out
}

// resolveGraphScope maps the ?scope token onto the vocabulary, following the Drift
// feed's ?period: an absent or unrecognised token falls back to the named default,
// so a hand-crafted value never draws a scope nobody declared. scopes always holds
// the whole-estate entry first, which is that default.
func resolveGraphScope(scopes []graphScope, token string) graphScope {
	for _, sc := range scopes {
		if sc.Token == token {
			return sc
		}
	}
	return scopes[0]
}

// graphRadius is a node type's drawn radius, mirroring the design's NODE_R. The
// domain apex draws largest (#22e: r16, ink stroke), a subdomain name r10, an
// address r9 and a service r6.
func graphRadius(typ string) int {
	switch typ {
	case "domain":
		return 16
	case "ip":
		return 9
	case "service":
		return 6
	default: // subdomain
		return 10
	}
}

// round1 rounds a float to one decimal place — the minimap coordinate precision the
// design fixture carries (e.g. 90·110/1200 = 8.25 → 8.3).
func round1(v float64) float64 { return math.Round(v*10) / 10 }

// classifyNameTypes splits the Name set into the domain|subdomain tiers (#22e): a
// name is the DOMAIN apex when it is a registrable root of the observed topology —
// it parents at least one other name in the set (some other name is `<label>.<it>`)
// and is itself parented by none. Every other name is a subdomain. The split is read
// purely off the observed name set (no public-suffix guess): an estate whose apex was
// never itself measured simply has no domain node, and each measured leaf stays a
// subdomain. Deterministic for a fixed set.
func classifyNameTypes(names map[string]struct{}) map[string]string {
	isChildOf := func(child, parent string) bool {
		return len(child) > len(parent)+1 && child[len(child)-len(parent)-1] == '.' &&
			child[len(child)-len(parent):] == parent
	}
	out := make(map[string]string, len(names))
	for n := range names {
		hasChild, hasParent := false, false
		for m := range names {
			if m == n {
				continue
			}
			if isChildOf(m, n) {
				hasChild = true
			}
			if isChildOf(n, m) {
				hasParent = true
			}
		}
		if hasChild && !hasParent {
			out[n] = "domain"
		} else {
			out[n] = "subdomain"
		}
	}
	return out
}

// buildGraph folds the estate's open spans into the graph's nodes and edges. It
// reads only relationships the corpus states: a Name's resolution addresses (the
// Name to Address edge) and a Service key's Address (the Address to Service edge).
// Nodes are laid out in three columns — Names, Addresses, Services — each sorted
// so the layout is deterministic. It never invents a node, an edge, or a severity.
func buildGraph(rows []db.ListAllOpenSpansRow) graphView {
	return buildScopedGraph(rows, graphScope{})
}

// graphMembers is the subject set one declared Seed accounts for, resolved against
// the corpus once. The whole-estate scope holds everything, which is what keeps a
// /graph with no ?scope token drawing exactly what it always drew.
//
// The two Seed kinds reach the drawing's three columns from opposite ends. A name
// scope holds the names beneath its domain, and with them the addresses those names
// resolve to. An address scope holds the addresses its prefix contains, and with
// them the names whose resolution reaches one — the same resolution edge read in the
// other direction. That second direction is a reading the brief did not spell out:
// without it an address scope draws a column of addresses no name points at, and a
// graph whose point is relationships would show none.
//
// A Service is held where the Address it rides is held. It is keyed on that address,
// so it needs no rule of its own.
// The ZERO value holds everything. A graphView assembled by hand — a fixture, a
// test — therefore narrows nothing, and a scope can only ever be applied by a build
// that resolved one.
type graphMembers struct {
	domain string
	prefix netip.Prefix
	names  map[string]struct{}
	addrs  map[netip.Addr]struct{}
}

// graphScopeMembers resolves a scope's membership over the corpus. It reads the
// resolution facet alone, because resolution is the only edge that carries one Seed
// kind's population into the other's column.
func graphScopeMembers(rows []db.ListAllOpenSpansRow, sc graphScope) graphMembers {
	if sc.Domain == "" && !sc.Prefix.IsValid() {
		return graphMembers{}
	}
	m := graphMembers{
		domain: sc.Domain,
		prefix: sc.Prefix,
		names:  map[string]struct{}{},
		addrs:  map[netip.Addr]struct{}{},
	}
	for _, row := range rows {
		if row.SubjectKind != "name" || row.Facet != "resolution" || row.IsGap {
			continue
		}
		underDomain := m.domain != "" && custody.LabelSuffix(row.SubjectKey, m.domain)
		for _, a := range decodeResolution(row.Value).Addresses {
			addr, err := netip.ParseAddr(a)
			if err != nil {
				continue
			}
			addr = addr.Unmap()
			if underDomain {
				m.addrs[addr] = struct{}{}
			}
			if m.prefix.IsValid() && m.prefix.Contains(addr) {
				m.names[row.SubjectKey] = struct{}{}
			}
		}
	}
	return m
}

// unbounded reports whether these members are the whole estate — the zero value,
// which names no domain and no prefix and so bounds nothing.
func (m graphMembers) unbounded() bool {
	return m.domain == "" && !m.prefix.IsValid()
}

// holdsName reports whether the scope holds this Name.
func (m graphMembers) holdsName(name string) bool {
	if m.unbounded() {
		return true
	}
	if m.domain != "" {
		return custody.LabelSuffix(name, m.domain)
	}
	_, ok := m.names[name]
	return ok
}

// holdsAddress reports whether the scope holds this Address. The spelling is parsed
// before it is judged, so containment is the family-matched prefix comparison and
// never a comparison of spellings. An address that does not parse is held by no
// scope.
func (m graphMembers) holdsAddress(spelling string) bool {
	if m.unbounded() {
		return true
	}
	addr, err := netip.ParseAddr(spelling)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	if m.domain != "" {
		_, ok := m.addrs[addr]
		return ok
	}
	return m.prefix.Contains(addr)
}

// buildScopedGraph is buildGraph bounded to one declared Seed (#1102, ADR-0136 §3).
// It drops the subjects the scope does not hold before any node is placed, so the
// drawing states nothing about them — it does not draw them dimmed, fold them, or
// count them.
func buildScopedGraph(rows []db.ListAllOpenSpansRow, sc graphScope) graphView {
	members := graphScopeMembers(rows, sc)

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
			if !members.holdsName(row.SubjectKey) {
				continue
			}
			names[row.SubjectKey] = struct{}{}
			noteFirst("name:"+row.SubjectKey, at)
			if row.Facet == "resolution" && !row.IsGap {
				for _, a := range decodeResolution(row.Value).Addresses {
					if !members.holdsAddress(a) {
						continue
					}
					addrs[a] = struct{}{}
					noteFirst("addr:"+a, at)
					addEdge(edge{from: "name:" + row.SubjectKey, to: "addr:" + a})
				}
			}
		case "service":
			pair, addr, keyed := parseServicePair(row.SubjectKey)
			// A service key that does not resolve to an address names no address a
			// scope could contain, so only the whole estate holds it.
			if keyed && !members.holdsAddress(addr) {
				continue
			}
			if !keyed && !members.unbounded() {
				continue
			}
			services[row.SubjectKey] = struct{}{}
			noteFirst("svc:"+row.SubjectKey, at)
			if keyed {
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
		r := graphRadius(typ)
		n := graphNode{
			ID: id, Label: label, Type: typ, X: x, Y: y,
			LabelDX: r + 9,
			HaloA:   float64(r) + 7,   // #22f: outer fill halo (r+7, opacity 0.12)
			HaloB:   float64(r) + 4.5, // #22f: inner ring halo (r+4.5, sw2, opacity 0.65)
			Ports:   ports,
		}
		if t, ok := first[internalID]; ok {
			n.First = t.Format(spanTimeFmt)
		}
		nodes = append(nodes, n)
	}

	nameType := classifyNameTypes(names)
	for i, k := range sortedSet(names) {
		place("name:"+k, k, k, nameType[k], graphColName, i, "—")
	}
	for i, a := range sortedSet(addrs) {
		place("addr:"+a, a, a, "ip", graphColAddr, i, joinPorts(addrPorts[a]))
	}
	for i, k := range sortedSet(services) {
		place("svc:"+k, k, svcLabel[k], "service", graphColSvc, i, "—")
	}

	contentW, contentH := graphContentBounds(nodes)
	for i := range nodes {
		nodes[i].Mx = round1(float64(nodes[i].X) * graphMiniW / float64(contentW))
		nodes[i].My = round1(float64(nodes[i].Y) * graphMiniH / float64(contentH))
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

	fitX, fitY, fitK, minK := graphFit(contentW, contentH)

	return graphView{
		Nodes: nodes, Edges: edges, Empty: len(nodes) == 0,
		ViewW: graphViewW, ViewH: graphViewH, MiniW: graphMiniW, MiniH: graphMiniH,
		ContentW: contentW, ContentH: contentH,
		FitX: fitX, FitY: fitY, FitK: fitK, MinK: minK,
		members: members,
	}
}

// graphFit frames the whole content box inside the viewport and centres it, and
// gives the zoom floor that reaches that framing. The content box floors at the
// viewport box, so the scale never exceeds 1 and an estate that fits keeps the
// standing origin at scale 1 and the standing 0.5 floor.
func graphFit(contentW, contentH int) (x, y, k, minK float64) {
	k = math.Min(graphViewW/float64(contentW), graphViewH/float64(contentH))
	// The scale truncates DOWN, so the drawing can only land inside the viewport, never
	// past its edge. The column run has no cap, so the scale must stay meaningful at any
	// height: truncating to four SIGNIFICANT digits holds the same relative accuracy all
	// the way down, where a fixed decimal place would reach 0 — and scale(0) draws
	// nothing — and would cut the scale by a third just above that floor.
	e := math.Pow(10, 4-math.Ceil(math.Log10(k)))
	k = math.Floor(k*e) / e
	x = math.Round((graphViewW-float64(contentW)*k)/2*10) / 10
	y = math.Round((graphViewH-float64(contentH)*k)/2*10) / 10
	return x, y, k, math.Min(graphZoomFloor, k)
}

// graphContentBounds measures the box the placed nodes occupy, padded past the
// outermost node's own radius. The column run is unbounded (y = graphRowTop +
// idx*graphRowStep), so a large estate reaches far below graphViewH and the viewport
// box stops describing the content. The floor is the viewport box itself, which keeps
// a small estate mapping exactly as it did and keeps the divisor non-zero when every
// node shares one row.
func graphContentBounds(nodes []graphNode) (int, int) {
	w, h := graphViewW, graphViewH
	for _, n := range nodes {
		r := graphRadius(n.Type)
		if right := n.X + r + graphPad; right > w {
			w = right
		}
		if bottom := n.Y + r + graphPad; bottom > h {
			h = bottom
		}
	}
	return w, h
}

// joinSignals lights each graph node with the open (fired) signals whose subject
// resolves to it, folding a Census set from the Signal engine onto the built
// topology and carrying each rule's real severity (P0.1) through to the node. A
// node's Sev is the worst (most urgent) severity among the rules that fired on it,
// which tints its halo — no fabricated level, only what actually fired.
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
		sev, _ := signal.SeverityFor(c.Rule)
		for _, m := range c.Fired {
			sig := graphSignal{Severity: sev.String(), SevLabel: sevLabel(sev.String()), Rule: c.Rule, Subject: m.Subject}
			switch kind {
			case "name", "service":
				attach(m.Subject, sig)
			case "endpoint":
				// name@service — light the Name leg (the endpoint's DNS identity),
				// falling back to the Service leg for a nameless endpoint or when the
				// Name node is not in the topology (so a real firing never vanishes).
				name, service := splitEndpointName(m.Subject)
				// A scope that does not hold the name takes the WHOLE firing out
				// (#1102). Without this the fallback would re-attribute it to the
				// Service leg, and the drawing would assert a signal the engine
				// censused against a subject the operator's scope excluded.
				if name != "" && !g.members.holdsName(name) {
					continue
				}
				if name == "" || !attach(name, sig) {
					attach(service, sig)
				}
			}
		}
	}
	// Fold each node's fired signals down to its worst severity — the one the halo,
	// minimap dot and service fill tint to (GraphView.jsx paints a single level per
	// node). A node with no firing keeps Sev == "" and draws no halo.
	for i := range g.Nodes {
		g.Nodes[i].Sev = worstSeverity(g.Nodes[i].OpenSignals)
	}
	return g
}

// worstSeverity returns the most-urgent severity token among a node's fired signals
// — the lowest rank in signal.SevOrder (critical outranks info) — or "" when none
// fired. It never manufactures a level: the token is one a rule actually carried.
func worstSeverity(sigs []graphSignal) string {
	best := ""
	bestRank := len(signal.SevOrder)
	for _, s := range sigs {
		if r := signal.Severity(s.Severity).Rank(); r < bestRank {
			bestRank = r
			best = s.Severity
		}
	}
	return best
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
// A ?scope token bounds the drawing to one declared Seed (#1102, ADR-0136 §3). It is
// the primary bound on the graph's population, and it is an operator act rather than
// a product rule: it narrows by a declaration the operator made, and it drops nothing
// the operator did not put outside the scope they picked.
//
// The topology read is the page's spine; the signal-corpus read is heavier
// (buildSignalCorpus fans out over resolution / reachability / certificate /
// http-identity), so its failure DEGRADES — the graph renders without signal state
// rather than 500ing the topology a viewer depends on, exactly as reports.go
// degrades its KPI on a failed read.
func (s *server) graphPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// A VERGE_DEV build serves the pinned fixtures.json graph slice — the byte-exact
	// 26-node topology the pixel goldens capture (as the sibling screens do). A real
	// deployment falls through to the honest live reads below.
	if s.devMode {
		s.render(w, r, "graph", s.graphFixtureData(acct))
		return
	}
	// The scope selector's vocabulary is the operator's own declared Seeds. Its read
	// DEGRADES like the signal-corpus read below: a failure costs the operator the
	// control, and an unrecognised token then falls back to the whole estate, rather
	// than 500ing the topology the viewer came for.
	seeds, err := s.store.ListSeeds(r.Context())
	if err != nil {
		log.Printf("web: graph: list seeds: %v", err)
	}
	scopes := graphScopes(seeds)
	selected := resolveGraphScope(scopes, r.URL.Query().Get("scope"))

	rows, err := s.store.ListAllOpenSpans(r.Context())
	if err != nil {
		s.serverError(w, "list all open spans", err)
		return
	}
	g := buildScopedGraph(rows, selected)
	g.Scopes = scopes
	g.Scope = selected.Token
	g.ScopeLabel = selected.Label
	if !g.Empty {
		corpus, err := s.buildSignalCorpus(r)
		if err != nil {
			log.Printf("web: graph: build signal corpus: %v", err)
		} else {
			g = joinSignals(g, signal.EvaluateCorpus(corpus))
		}
	}
	s.render(w, r, "graph", map[string]any{
		"Title": "Graph", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "graph", "DesignTokens": true,
		"Graph": g,
	})
}
