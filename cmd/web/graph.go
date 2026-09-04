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

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/graph.tmpl"))

const (
	graphViewW   = 1200
	graphViewH   = 640
	graphColName = 130
	graphColAddr = 560
	graphColSvc  = 1000
	graphRowTop  = 44
	graphRowStep = 46
	graphMiniW   = 110
	graphMiniH   = 59
	graphPad     = 44

	graphRadiusSubdomain = 10

	graphZoomFloor = 0.5

	// These are viewBox user units, not CSS pixels; a pixel basis moves the cap (ADR-0136 §5).

	graphLabelPx    = 11
	graphLabelMinPx = 7
)

// Rounding up costs a sliver of scales but never leaves a label under the floor (ADR-0136 §5).

var graphLabelMinK = math.Ceil(float64(graphLabelMinPx)/float64(graphLabelPx)*1e4) / 1e4

// Past the cap subjects are shed, never folded: a rollup node is no Subject kind (ADR-0136 §2).

const graphColumnCap = (graphViewH*graphLabelPx/graphLabelMinPx-2*graphPad-graphRadiusSubdomain)/graphRowStep + 1

type graphSignal struct {
	Severity string
	SevLabel string
	Rule     string
	Subject  string
}

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

type graphEdge struct {
	X1, Y1, X2, Y2 int
	ToService      bool
}

type graphView struct {
	Nodes                      []graphNode
	Edges                      []graphEdge
	Empty                      bool
	ViewW, ViewH, MiniW, MiniH int
	ContentW, ContentH         int
	FitX, FitY, FitK           float64
	MinK                       float64
	LabelMinK                  float64
	Scopes                     []graphScope
	Scope                      string
	ScopeLabel                 string
	Cap                        int
	Capped                     []graphColumnCount
	CutEdges                   int
	CutSignals                 int
	members                    graphMembers
	held                       map[string]struct{}
}

type graphColumnCount struct {
	Label string
	Drawn int
	Held  int
}

const graphScopeAll = "all"

const graphScopeAllLabel = "Whole estate"

type graphScope struct {
	Token  string
	Label  string
	Domain string
	Prefix netip.Prefix
}

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
		// An invalid prefix bounds nothing, so an entry built from one would draw the whole estate.
		case s.Kind == "address" && s.AddressCidr != nil && s.AddressCidr.IsValid():
			p := s.AddressCidr.Masked()
			out = append(out, graphScope{Token: p.String(), Label: p.String(), Prefix: p})
		}
	}
	return out
}

func resolveGraphScope(scopes []graphScope, token string) graphScope {
	for _, sc := range scopes {
		if sc.Token == token {
			return sc
		}
	}
	return scopes[0]
}

func graphRadius(typ string) int {
	switch typ {
	case "domain":
		return 16
	case "ip":
		return 9
	case "service":
		return 6
	default:
		return graphRadiusSubdomain
	}
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func classifyNameTypes(names map[string]struct{}) map[string]string {
	// A public-suffix guess is refused, so an unmeasured apex simply has no node (ADR-0136 §2).
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

func buildGraph(rows []db.ListAllOpenSpansRow) graphView {
	return buildScopedGraph(rows, graphScope{})
}

type graphMembers struct {
	domain string
	prefix netip.Prefix
	names  map[string]struct{}
	addrs  map[netip.Addr]struct{}
}

func graphScopeMembers(rows []db.ListAllOpenSpansRow, sc graphScope) graphMembers {
	// Resolution reads both ways, or an address scope draws addresses no name points at (#1102).
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

func (m graphMembers) unbounded() bool {
	return m.domain == "" && !m.prefix.IsValid()
}

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

func buildScopedGraph(rows []db.ListAllOpenSpansRow, sc graphScope) graphView {
	// A scoped-out subject is dropped before placement, never dimmed, folded or counted (ADR-0136 §3).
	members := graphScopeMembers(rows, sc)

	first := map[string]time.Time{}
	noteFirst := func(id string, t time.Time) {
		if cur, ok := first[id]; !ok || t.Before(cur) {
			first[id] = t
		}
	}

	names := map[string]struct{}{}
	addrs := map[string]struct{}{}
	services := map[string]struct{}{}
	svcLabel := map[string]string{}
	addrPorts := map[string]map[string]struct{}{}

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
			HaloA:   float64(r) + 7,
			HaloB:   float64(r) + 4.5,
			Ports:   ports,
		}
		if t, ok := first[internalID]; ok {
			n.First = t.Format(spanTimeFmt)
		}
		nodes = append(nodes, n)
	}

	var capped []graphColumnCount
	held := map[string]struct{}{}
	// Severity ordering is refused: it would move the drawing under the operator (ADR-0136 §4).
	capColumn := func(label, prefix string, keys []string) []string {
		for _, k := range keys {
			held[prefix+k] = struct{}{}
		}
		if len(keys) <= graphColumnCap {
			return keys
		}
		capped = append(capped, graphColumnCount{Label: label, Drawn: graphColumnCap, Held: len(keys)})
		return keys[:graphColumnCap]
	}

	// The split reads the uncapped name set, so the cap cannot demote an apex (ADR-0136 §2).
	nameType := classifyNameTypes(names)
	for i, k := range capColumn("names", "name:", sortedSet(names)) {
		place("name:"+k, k, k, nameType[k], graphColName, i, "—")
	}
	for i, a := range capColumn("addresses", "addr:", sortedSet(addrs)) {
		place("addr:"+a, a, a, "ip", graphColAddr, i, joinPorts(addrPorts[a]))
	}
	for i, k := range capColumn("services", "svc:", sortedSet(services)) {
		place("svc:"+k, k, svcLabel[k], "service", graphColSvc, i, "—")
	}

	contentW, contentH := graphContentBounds(nodes)
	// Mapped against the content box: on a viewport basis a dot clips past ~14 rows (#1089).
	for i := range nodes {
		nodes[i].Mx = round1(float64(nodes[i].X) * graphMiniW / float64(contentW))
		nodes[i].My = round1(float64(nodes[i].Y) * graphMiniH / float64(contentH))
	}

	var edges []graphEdge
	cutEdges := 0
	for _, e := range order {
		a, ok1 := pos[e.from]
		b, ok2 := pos[e.to]
		// The cap made this guard reachable, so a cut edge is counted rather than dropped (ADR-0136 §6).
		if !ok1 || !ok2 {
			cutEdges++
			continue
		}
		edges = append(edges, graphEdge{X1: a.x, Y1: a.y, X2: b.x, Y2: b.y, ToService: e.toSvc})
	}

	fitX, fitY, fitK, minK := graphFit(contentW, contentH)

	return graphView{
		Nodes: nodes, Edges: edges, Empty: len(nodes) == 0,
		ViewW: graphViewW, ViewH: graphViewH, MiniW: graphMiniW, MiniH: graphMiniH,
		ContentW: contentW, ContentH: contentH,
		FitX: fitX, FitY: fitY, FitK: fitK, MinK: minK, LabelMinK: graphLabelMinK,
		Cap: graphColumnCap, Capped: capped, CutEdges: cutEdges,
		members: members,
		held:    held,
	}
}

func graphFit(contentW, contentH int) (x, y, k, minK float64) {
	k = math.Min(graphViewW/float64(contentW), graphViewH/float64(contentH))
	// Four significant digits, not a decimal place a tall drawing would truncate to scale(0) (#1101).
	e := math.Pow(10, 4-math.Ceil(math.Log10(k)))
	k = math.Floor(k*e) / e
	x = math.Round((graphViewW-float64(contentW)*k)/2*10) / 10
	y = math.Round((graphViewH-float64(contentH)*k)/2*10) / 10
	return x, y, k, math.Min(graphZoomFloor, k)
}

func graphContentBounds(nodes []graphNode) (int, int) {
	// The viewport floor keeps a small estate mapping as before and the divisor non-zero (#1089).
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

func joinSignals(g graphView, censuses []signal.Census) graphView {
	// Rolling a Service firing up onto its Address asserts a signal the engine never censused (#289).
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
	// A node the corpus never held is no cap deletion, so only a held id counts (ADR-0136 §6).
	cut := func(internalID string) bool {
		_, ok := g.held[internalID]
		return ok
	}
	for _, c := range censuses {
		kind := signal.SubjectKindFor(c.Rule)
		sev, _ := signal.SeverityFor(c.Rule)
		for _, m := range c.Fired {
			sig := graphSignal{Severity: sev.String(), SevLabel: sevLabel(sev.String()), Rule: c.Rule, Subject: m.Subject}
			switch kind {
			case "name", "service":
				prefix := "name:"
				if kind == "service" {
					prefix = "svc:"
				}
				if !attach(m.Subject, sig) && cut(prefix+m.Subject) {
					g.CutSignals++
				}
			case "endpoint":
				name, service := splitEndpointName(m.Subject)
				// Without this the fallback re-attributes a firing to a leg the scope excluded (#1102).
				if name != "" && !g.members.holdsName(name) {
					continue
				}
				if name != "" {
					if attach(name, sig) {
						continue
					}
					// A cap-dropped Name must not fall to the Service leg, so the deletion is counted (#1103).
					if cut("name:" + name) {
						g.CutSignals++
						continue
					}
				}
				if !attach(service, sig) && cut("svc:"+service) {
					g.CutSignals++
				}
			}
		}
	}
	for i := range g.Nodes {
		g.Nodes[i].Sev = worstSeverity(g.Nodes[i].OpenSignals)
	}
	return g
}

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

func sortedSet(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

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

func (s *server) graphPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "graph", s.graphFixtureData(acct))
		return
	}
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
