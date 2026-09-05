package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// Only fired instances paint, as flat rows and never a per-rule census (docs/spec/v1-spec.md §6.5).

// This layer folds facts into the snapshot and never decides a verdict; the engine owns every one.

type dnsRecordValue struct {
	RRs []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Data string `json:"data"`
	} `json:"rrs"`
	Delegation *struct {
		Lame bool `json:"lame"`
		Gap  bool `json:"gap"`
	} `json:"delegation"`
}

type annotationView struct {
	ID      int64
	Subject string
	Href    string
	Signal  string
	Reason  string
	At      string
	Orphan  bool
}

type signalRow struct {
	signalInstanceView
	SevLabel    string
	Annotated   bool
	AnnoReason  string
	AnnoID      int64
	Withdrawn   bool
	ViewKey     string
	DescopeHref string
	Tags        []string
	CVE         string
	Desc        string
	RuleID      string
	RuleVersion string
	DetectedBy  string
	Diff        *sigDiff
	History     []sigHistoryEntry
	idNum       int64
	seenAge     int64
}

type sigDiff struct {
	Title string
	Lines []sigDiffLine
}

type sigDiffLine struct {
	Type string
	Text string
}

type sigHistoryEntry struct {
	Title  string
	Detail string
	Time   string
	Tone   string
	Mono   bool
}

type descopeView struct {
	Asset     string
	CloseHref string
}

type sigSort struct {
	Key, Dir                             string
	SevHref, AssetHref, IDHref, SeenHref string
}

type pageCell struct {
	Num      int
	Href     string
	Active   bool
	Ellipsis bool
}

type signalsForms struct {
	annoError   string
	annoSubject string
	annoSignal  string
	annoReason  string
}

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/signals.tmpl"))

func (s *server) signalsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "signals", s.signalsFixtureData(acct, r))
		return
	}
	forms, _ := takeFormFlash[signalsForms](s, r)
	s.renderSignals(w, r, acct, forms)
}

func (s *server) signalsExport(w http.ResponseWriter, r *http.Request, acct db.Account) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		http.Error(w, "unsupported export format: "+format+" (want csv)", http.StatusBadRequest)
		return
	}

	open, annotated, withdrawn, err := s.buildSignalTabs(r)
	if err != nil {
		s.serverError(w, "signals export: build signal tabs", err)
		return
	}

	tab := r.URL.Query().Get("tab")
	var base []signalRow
	switch tab {
	case "annotated":
		base = annotated
	case "withdrawn":
		base = withdrawn
	default:
		tab = "open"
		base = open
	}

	rows := filterSignalRows(base, strings.TrimSpace(r.URL.Query().Get("q")), r.URL.Query().Get("sev"))
	sortSignalRows(rows, "sev", "asc")
	s.writeSignalsExportCSV(w, tab, rows)
}

func (s *server) writeSignalsExportCSV(w http.ResponseWriter, tab string, rows []signalRow) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="signals-`+tab+`.csv"`)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"severity", "id", "signal", "asset", "port", "first_seen", "last_seen"})
	for _, row := range rows {
		_ = cw.Write([]string{row.Severity, row.SigID, csvSafe(row.Signal), csvSafe(row.Asset), row.Port, row.First, row.Last})
	}
}

func (s *server) renderSignals(w http.ResponseWriter, r *http.Request, acct db.Account, forms signalsForms) {
	open, annotated, withdrawn, err := s.buildSignalTabs(r)
	if err != nil {
		s.serverError(w, "build signal tabs", err)
		return
	}

	tab := r.URL.Query().Get("tab")
	switch tab {
	case "annotated", "withdrawn":
	default:
		tab = "open"
	}

	var base []signalRow
	switch tab {
	case "annotated":
		base = annotated
	case "withdrawn":
		base = withdrawn
	default:
		base = open
	}

	// A form submit discards a client-side scope, so filter and sort ride the query (ADR-0158 §1).
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sevSel := r.URL.Query().Get("sev")
	if sevSel == "" {
		sevSel = "All severities"
	}
	sortKey := r.URL.Query().Get("sort")
	switch sortKey {
	case "sev", "asset", "id", "seen":
	default:
		sortKey = "sev"
	}
	dir := r.URL.Query().Get("dir")
	if dir != "desc" {
		dir = "asc"
	}

	filtered := filterSignalRows(base, q, sevSel)
	sortSignalRows(filtered, sortKey, dir)
	total := len(base)
	shown := len(filtered)

	const pageSize = 10
	page := 1
	if p, e := strconv.Atoi(r.URL.Query().Get("page")); e == nil && p > 1 {
		page = p
	}
	showPag := tab == "open" && shown > 0
	rows := filtered
	var pages []pageCell
	var prevHref, nextHref, pageInfo string
	var prevDisabled, nextDisabled bool

	filterVals := func() url.Values {
		v := url.Values{}
		v.Set("tab", tab)
		if q != "" {
			v.Set("q", q)
		}
		if sevSel != "All severities" {
			v.Set("sev", sevSel)
		}
		return v
	}

	if showPag {
		pageCount := (shown + pageSize - 1) / pageSize
		if page > pageCount {
			page = pageCount
		}
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > shown {
			end = shown
		}
		rows = filtered[start:end]

		pageHref := func(p int) string {
			v := filterVals()
			v.Set("sort", sortKey)
			v.Set("dir", dir)
			if p > 1 {
				v.Set("page", strconv.Itoa(p))
			}
			return "/signals?" + v.Encode()
		}
		for _, p := range pageWindow(page, pageCount) {
			if p < 0 {
				pages = append(pages, pageCell{Ellipsis: true})
				continue
			}
			pages = append(pages, pageCell{Num: p, Href: pageHref(p), Active: p == page})
		}
		prevDisabled = page <= 1
		nextDisabled = page >= pageCount
		prevHref = pageHref(page - 1)
		nextHref = pageHref(page + 1)
		pageInfo = fmt.Sprintf("%d–%d of %d", start+1, end, shown)
	} else {
		page = 1
	}

	state := filterVals()
	state.Set("sort", sortKey)
	state.Set("dir", dir)
	if page > 1 {
		state.Set("page", strconv.Itoa(page))
	}
	closeHref := "/signals?" + state.Encode()

	sortHref := func(col string) string {
		v := filterVals()
		nd := "asc"
		if sortKey == col && dir == "asc" {
			nd = "desc"
		}
		v.Set("sort", col)
		v.Set("dir", nd)
		return "/signals?" + v.Encode()
	}
	sortState := sigSort{
		Key: sortKey, Dir: dir,
		SevHref: sortHref("sev"), AssetHref: sortHref("asset"),
		IDHref: sortHref("id"), SeenHref: sortHref("seen"),
	}

	for i := range rows {
		rows[i].DescopeHref = closeHref + "&descope=" + url.QueryEscape(rows[i].ViewKey)
	}

	selKey := r.URL.Query().Get("view")
	var drawer *signalRow
	if selKey != "" {
		for i := range base {
			if base[i].ViewKey == selKey {
				d := base[i]
				s.enrichSignalDrawer(r, &d)
				drawer = &d
				break
			}
		}
	}

	var descope *descopeView
	// The handler builds this for any role; signals.tmpl is what gates the dialog on .IsAdmin.
	if dk := r.URL.Query().Get("descope"); dk != "" {
		for i := range base {
			if base[i].ViewKey == dk {
				descope = &descopeView{Asset: base[i].Asset, CloseHref: closeHref}
				break
			}
		}
	}

	var annoReasonDraft string
	// A reason echoed into another subject's textarea invites an acceptance against the wrong pair.
	// The stashed signal is not compared: an unknown rule name is the one refusal worth echoing.
	if forms.annoReason != "" && drawer != nil && normalizeSubjectKey(drawer.Asset) == forms.annoSubject {
		annoReasonDraft = forms.annoReason
	}

	hasExport := total > 0
	exportVals := filterVals()
	exportHref := "/signals/export?" + exportVals.Encode()

	s.render(w, r, "signals", map[string]any{
		"Title": "Signals", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":       "signals",
		"SignalCount":     len(open),
		"Tab":             tab,
		"OpenCount":       len(open),
		"AnnotatedCount":  len(annotated),
		"WithdrawnCount":  len(withdrawn),
		"Rows":            rows,
		"HasAny":          total > 0,
		"Shown":           shown,
		"Total":           total,
		"Q":               q,
		"Sev":             sevSel,
		"SevOptions":      []string{"All severities", "Critical", "High", "Medium", "Low", "Info"},
		"Sort":            sortState,
		"ClearHref":       "/signals?tab=" + tab,
		"ShowPagination":  showPag,
		"Pages":           pages,
		"PrevHref":        prevHref,
		"NextHref":        nextHref,
		"PrevDisabled":    prevDisabled,
		"NextDisabled":    nextDisabled,
		"PageInfo":        pageInfo,
		"CloseHref":       closeHref,
		"ViewPrefix":      closeHref + "&view=",
		"ExportHref":      exportHref,
		"HasExport":       hasExport,
		"SelKey":          selKey,
		"Drawer":          drawer,
		"Descope":         descope,
		"AnnoError":       forms.annoError,
		"AnnoReasonDraft": annoReasonDraft,
	})
}

func (s *server) enrichSignalDrawer(r *http.Request, row *signalRow) {
	// No rules registry carries a tag, CVE or description, and a fired signal has no single vantage.
	if row.IP == "" {
		row.IP = "—"
	}
	row.RuleID = row.Signal
	row.RuleVersion = ruleVersions[row.Signal]
	row.Diff = s.signalDrift(r, signal.SubjectKindFor(row.Signal), row.Asset)

	var hist []sigHistoryEntry
	if row.Last != "" {
		title := "Still present"
		tone := "accent"
		if row.Withdrawn {
			title = "Last seen firing"
			tone = "neutral"
		}
		hist = append(hist, sigHistoryEntry{Title: title, Time: row.Seen, Tone: tone})
	}
	if row.First != "" {
		tone := "neutral"
		if row.Severity == "critical" || row.Severity == "high" {
			tone = "danger"
		}
		hist = append(hist, sigHistoryEntry{Title: "Signal raised", Detail: row.SigID, Time: row.First, Tone: tone, Mono: true})
	}
	row.History = hist
}

var ruleVersions = buildRuleVersions()

func buildRuleVersions() map[string]string {
	m := make(map[string]string, len(signal.AllRuleNames()))
	for _, r := range signal.All() {
		m[r.Name()] = r.Version().String()
	}
	for _, r := range signal.AllEndpointRules() {
		m[r.Name()] = r.Version().String()
	}
	for _, r := range signal.AllServiceRules() {
		m[r.Name()] = r.Version().String()
	}
	return m
}

func (s *server) signalDrift(r *http.Request, kind, key string) *sigDiff {
	// The drawer diff is diagnostic rather than load-bearing, so an unreadable corpus degrades to nil.
	if kind == "" || key == "" {
		return nil
	}
	tls := s.buildTimelines(r, kind, key)
	var best *sigDiff
	var bestAt string
	for _, tv := range tls {
		if tv.Current == nil || len(tv.Closed) == 0 {
			continue
		}
		before := tv.Closed[len(tv.Closed)-1].Value
		after := tv.Current.Value
		if before == after {
			continue
		}
		// A fixed-width UTC stamp, so a plain string compare orders the transitions chronologically.
		if best == nil || tv.Current.OpenedAt > bestAt {
			bestAt = tv.Current.OpenedAt
			best = &sigDiff{
				Title: tv.Label + " · drift",
				Lines: []sigDiffLine{{Type: "remove", Text: before}, {Type: "add", Text: after}},
			}
		}
	}
	return best
}

func (s *server) buildSignalTabs(r *http.Request) (open, annotated, withdrawn []signalRow, err error) {
	ctx := r.Context()
	corpus, err := s.buildSignalCorpus(r)
	if err != nil {
		return nil, nil, nil, err
	}
	annos, err := s.store.ListAnnotations(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	censuses := signal.EvaluateCorpus(corpus)

	population := map[string]map[string]bool{}
	fired := map[[2]string]bool{}
	for _, c := range censuses {
		pop := map[string]bool{}
		for _, m := range c.Fired {
			pop[m.Subject] = true
			fired[[2]string{c.Rule, m.Subject}] = true
		}
		for _, m := range c.NotFired {
			pop[m.Subject] = true
		}
		for _, m := range c.NotEvaluable {
			pop[m.Subject] = true
		}
		population[c.Rule] = pop
	}

	instances, err := s.deriveSignalInstances(ctx, censuses)
	if err != nil {
		return nil, nil, nil, err
	}

	identRows, err := s.store.ListSignalInstances(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	ident := make(map[[2]string]instIdent, len(identRows))
	for _, row := range identRows {
		ident[[2]string{row.SignalName, row.SubjectKey}] = instIdent{
			id: row.ID, first: row.FirstSeen.Time, firstOK: row.FirstSeen.Valid,
		}
	}

	annoViews := annotationViews(annos, population)
	annoByPair := make(map[[2]string]annotationView, len(annoViews))
	for _, av := range annoViews {
		annoByPair[[2]string{av.Signal, av.Subject}] = av
	}

	now := s.now().UTC()

	open = make([]signalRow, 0, len(instances))
	for _, inst := range instances {
		row := signalRow{
			signalInstanceView: inst,
			SevLabel:           sevLabel(inst.Severity),
			ViewKey:            inst.SigID,
			idNum:              parseSigNum(inst.SigID),
			seenAge:            seenAgeMinutes(inst.Last, now),
		}
		if av, ok := annoByPair[[2]string{inst.Signal, inst.Asset}]; ok {
			row.Annotated = true
			row.AnnoReason = av.Reason
			row.AnnoID = av.ID
		}
		open = append(open, row)
	}

	for _, av := range annoViews {
		row := annotationRow(av, ident, fired, now)
		// The subject withdrew, not the operator: orphan is derived on read and stored nowhere (ADR-0092).
		if av.Orphan {
			row.Withdrawn = true
			withdrawn = append(withdrawn, row)
		} else {
			annotated = append(annotated, row)
		}
	}
	sortSignalRows(annotated, "sev", "asc")
	sortSignalRows(withdrawn, "sev", "asc")
	return open, annotated, withdrawn, nil
}

type instIdent struct {
	id      int64
	first   time.Time
	firstOK bool
}

func annotationRow(av annotationView, ident map[[2]string]instIdent, fired map[[2]string]bool, now time.Time) signalRow {
	key := [2]string{av.Signal, av.Subject}
	sev, _ := signal.SeverityFor(av.Signal)
	ip, port := subjectAddrPort(av.Subject)
	v := signalInstanceView{
		Signal:   av.Signal,
		Title:    signalTitle(av.Signal),
		Severity: sev.String(),
		SevRank:  sev.Rank(),
		Asset:    av.Subject,
		IP:       ip,
		Port:     port,
		Href:     subjectHref(signal.SubjectKindFor(av.Signal), av.Subject),
	}
	if id, ok := ident[key]; ok {
		v.SigID = formatSigID(id.id)
		if id.firstOK {
			v.First = id.first.UTC().Format(time.RFC3339)
		}
	}
	if fired[key] {
		v.Last = now.Format(time.RFC3339)
		v.Seen = relTime(now, now)
	} else if id, ok := ident[key]; ok && id.firstOK {
		v.Last = id.first.UTC().Format(time.RFC3339)
		v.Seen = relTime(id.first, now)
	}
	return signalRow{
		signalInstanceView: v,
		SevLabel:           sevLabel(v.Severity),
		Annotated:          true,
		AnnoReason:         av.Reason,
		AnnoID:             av.ID,
		idNum:              parseSigNum(v.SigID),
		seenAge:            seenAgeMinutes(v.Last, now),
		// The two id spaces never collide, so a withdrawn pair opens without a live mint.
		ViewKey: strconv.FormatInt(av.ID, 10),
	}
}

func filterSignalRows(rows []signalRow, q, sev string) []signalRow {
	q = strings.ToLower(strings.TrimSpace(q))
	sevWant := ""
	if sev != "" && sev != "All severities" {
		sevWant = strings.ToLower(sev)
	}
	out := make([]signalRow, 0, len(rows))
	for _, r := range rows {
		if sevWant != "" && r.Severity != sevWant {
			continue
		}
		if q != "" {
			hay := strings.ToLower(r.Title + " " + r.Asset + " " + r.SigID + " " + r.Signal)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

func sortSignalRows(rows []signalRow, key, dir string) {
	// A re-render must not reshuffle equal keys, so the sort is stable and the tiebreak total.
	primary := func(a, b signalRow) int {
		switch key {
		case "asset":
			return strings.Compare(a.Asset, b.Asset)
		case "id":
			switch {
			case a.idNum < b.idNum:
				return -1
			case a.idNum > b.idNum:
				return 1
			}
			return 0
		case "seen":
			switch {
			case a.seenAge < b.seenAge:
				return -1
			case a.seenAge > b.seenAge:
				return 1
			}
			return 0
		default:
			return a.SevRank - b.SevRank
		}
	}
	less := func(a, b signalRow) bool {
		if c := primary(a, b); c != 0 {
			return c < 0
		}
		if a.SevRank != b.SevRank {
			return a.SevRank < b.SevRank
		}
		if a.Asset != b.Asset {
			return a.Asset < b.Asset
		}
		return a.idNum < b.idNum
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if dir == "desc" {
			return less(rows[j], rows[i])
		}
		return less(rows[i], rows[j])
	})
}

func pageWindow(page, pageCount int) []int {
	// A constant-width window keeps the pagination control from resizing as the operator pages.
	switch {
	case pageCount <= 7:
		out := make([]int, 0, pageCount)
		for p := 1; p <= pageCount; p++ {
			out = append(out, p)
		}
		return out
	case page <= 4:
		return []int{1, 2, 3, 4, 5, -1, pageCount}
	case page >= pageCount-3:
		return []int{1, -1, pageCount - 4, pageCount - 3, pageCount - 2, pageCount - 1, pageCount}
	default:
		return []int{1, -1, page - 1, page, page + 1, -1, pageCount}
	}
}

func sevLabel(sev string) string {
	if sev == "" {
		return ""
	}
	return strings.ToUpper(sev[:1]) + sev[1:]
}

func parseSigNum(id string) int64 {
	if n, err := strconv.ParseInt(strings.TrimPrefix(id, "SIG-"), 10, 64); err == nil {
		return n
	}
	return 0
}

func seenAgeMinutes(iso string, now time.Time) int64 {
	if iso == "" {
		return 1 << 62
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return 1 << 62
	}
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	return int64(d / time.Minute)
}

func annotationViews(annos []db.Annotation, population map[string]map[string]bool) []annotationView {
	out := make([]annotationView, 0, len(annos))
	for _, a := range annos {
		v := annotationView{
			ID:      a.ID,
			Subject: a.SubjectKey,
			Href:    subjectHref(signal.SubjectKindFor(a.SignalName), a.SubjectKey),
			Signal:  a.SignalName,
			Reason:  a.Reason,
			Orphan:  !population[a.SignalName][a.SubjectKey],
		}
		if a.DeclaredAt.Valid {
			v.At = a.DeclaredAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		out = append(out, v)
	}
	return out
}

func (s *server) buildNameFacts(r *http.Request) ([]signal.NameFacts, error) {
	ctx := r.Context()

	resRows, err := s.store.ListNameResolutionsByClass(ctx, db.ListNameResolutionsByClassParams{
		AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return nil, err
	}
	dnsRows, err := s.store.ListNameDNSRecords(ctx, db.ListNameDNSRecordsParams{
		AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return nil, err
	}
	zoneRows, err := s.store.ListZoneDeclarations(ctx)
	if err != nil {
		return nil, err
	}

	// Vantage class is derived per read from presented-address facts, never a stored column (#709).
	covered, err := s.addressScopeCovered(ctx)
	if err != nil {
		return nil, err
	}
	byClass := collapseNameResolutions(resRows, covered)

	cnameTarget := map[string]string{}
	nsLame := map[string]bool{}
	for _, row := range dnsRows {
		var v dnsRecordValue
		_ = json.Unmarshal(row.Value, &v)
		switch strings.ToUpper(row.Discriminator) {
		case "CNAME":
			for _, rr := range v.RRs {
				if strings.EqualFold(rr.Type, "CNAME") {
					cnameTarget[row.SubjectKey] = resolutionwalk.CanonicalName(rr.Data)
					break
				}
			}
		case "NS":
			if v.Delegation != nil && v.Delegation.Lame {
				nsLame[row.SubjectKey] = true
			}
		}
	}

	declared := map[string]bool{}
	var zoneDomains []string
	for _, row := range zoneRows {
		if !row.NameDomain.Valid {
			continue
		}
		domain := resolutionwalk.CanonicalName(row.NameDomain.String)
		zoneDomains = append(zoneDomains, domain)
		for name := range signal.DeclaredNames(row.Content, domain) {
			declared[name] = true
		}
	}

	// A zone-declared name that never resolved must still enter its rule's domain.
	names := map[string]struct{}{}
	for name := range byClass {
		names[name] = struct{}{}
	}
	for name := range declared {
		names[name] = struct{}{}
	}

	composed := map[string]composedResolution{}
	for name := range names {
		composed[name] = composeResolution(byClass[name], nsLame[name])
	}

	facts := make([]signal.NameFacts, 0, len(names))
	for name := range names {
		c := composed[name]
		f := signal.NameFacts{
			Name:           name,
			InEstate:       c.inEstate,
			Resolution:     c.outcome,
			Addresses:      c.addresses,
			ZoneDeclared:   declared[name],
			InDeclaredZone: custody.WithinAnyZone(name, zoneDomains),
		}
		if target, ok := cnameTarget[name]; ok {
			f.CNAMETarget = target
			f.TargetResolution = composed[target].outcome
		}
		if inet, ok := byClass[name]["internet"]; ok {
			f.HasInternetVantage = true
			f.InternetResolution = inet.Outcome
			f.InternetAddresses = inet.Addresses
		}
		facts = append(facts, f)
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Name < facts[j].Name })
	return facts, nil
}

type composedResolution struct {
	outcome   string
	addresses []string
	inEstate  bool
}

func composeResolution(classes map[string]resolutionValue, lame bool) composedResolution {
	// A false Resolved fabricates addresses while a false Shadowed only withholds one (CONTEXT.md).
	if len(classes) == 0 {
		return composedResolution{outcome: signal.Gap, inEstate: false}
	}
	anyShadowed, anyResolved, anyNoData := false, false, false
	allNameError := true
	addrs := map[string]struct{}{}
	for _, v := range classes {
		switch v.Outcome {
		case signal.Shadowed:
			anyShadowed = true
		case signal.Resolved:
			anyResolved = true
			for _, a := range v.Addresses {
				addrs[a] = struct{}{}
			}
		case signal.NoData:
			anyNoData = true
		}
		if v.Outcome != signal.NameError {
			allNameError = false
		}
	}

	out := composedResolution{inEstate: !allNameError}
	switch {
	case anyShadowed:
		out.outcome = signal.Shadowed
	case anyResolved:
		out.outcome = signal.Resolved
		out.addresses = sortedKeys(addrs)
	case lame:
		out.outcome = signal.Lame
	case anyNoData:
		out.outcome = signal.NoData
	case allNameError:
		out.outcome = signal.NameError
	default:
		out.outcome = signal.Gap
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (s *server) buildSignalCorpus(r *http.Request) (signal.Corpus, error) {
	// Folded by facet name and value, so an unlanded leaf is not-evaluable and never a build break.
	names, err := s.buildNameFacts(r)
	if err != nil {
		return signal.Corpus{}, err
	}
	services, estateAddrs, err := s.buildServiceFacts(r)
	if err != nil {
		return signal.Corpus{}, err
	}
	endpoints, err := s.buildEndpointFacts(r, names, estateAddrs)
	if err != nil {
		return signal.Corpus{}, err
	}
	return signal.Corpus{Names: names, Services: services, Endpoints: endpoints}, nil
}

func (s *server) buildServiceFacts(r *http.Request) ([]signal.ServiceFacts, map[string]bool, error) {
	rows, err := s.store.ListServiceReachabilitySpansByClass(r.Context())
	if err != nil {
		return nil, nil, err
	}
	tlsRows, err := s.store.ListServiceTLSAcceptance(r.Context(), db.ListServiceTLSAcceptanceParams{
		AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return nil, nil, err
	}
	tlsBySubject := make(map[string]tlsAcceptanceValue, len(tlsRows))
	for _, row := range tlsRows {
		tlsBySubject[row.SubjectKey] = decodeTLSAcceptance(row.Value)
	}
	vc := vergecore.Default()

	covered, err := s.addressScopeCovered(r.Context())
	if err != nil {
		return nil, nil, err
	}
	byClass := collapseReachLegs(reachRowsFromCurrent(rows), covered)
	order := make([]string, 0, len(byClass))
	for k := range byClass {
		order = append(order, k)
	}
	sort.Strings(order)

	estateAddrs := map[string]bool{}
	facts := make([]signal.ServiceFacts, 0, len(order))
	for _, sub := range order {
		f := signal.ServiceFacts{Subject: sub}
		if pair, addr, ok := parseServicePair(sub); ok {
			f.OnSensitiveList = pair.Transport == vergecore.TCP && vc.IsSensitive(pair)
			estateAddrs[addr] = true
		}
		// A blanket responder's reach is a Gap, so the rule is damped at the measurement (ADR-0104 §3).
		if l, ok := byClass[sub]["internet"]; ok && !l.isGap && l.outcome != "" {
			f.HasInternetReach = true
			f.InternetReach = l.outcome
		}
		if tls, ok := tlsBySubject[sub]; ok && tls.Outcome == string(tlsacceptance.Enumerated) {
			f.TLSHandshakeCompleted = true
			f.TLSVersionsReadable = len(tls.Versions) > 0
			for _, ver := range tls.Versions {
				if ver.Version == tlsacceptance.TLS10 {
					f.TLS10Accepted = true
					break
				}
			}
		}
		facts = append(facts, f)
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Subject < facts[j].Subject })
	return facts, estateAddrs, nil
}

func (s *server) buildEndpointFacts(r *http.Request, names []signal.NameFacts, estateAddrs map[string]bool) ([]signal.EndpointFacts, error) {
	ctx := r.Context()
	certRows, err := s.store.ListEndpointCertificates(ctx, db.ListEndpointCertificatesParams{
		AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return nil, err
	}
	httpRows, err := s.store.ListCurrentEndpointSubjects(ctx, db.ListCurrentEndpointSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return nil, err
	}

	certVal := map[string]certificateValue{}
	for _, row := range certRows {
		certVal[row.SubjectKey] = decodeCertificate(row.Value)
	}
	httpID := map[string]httpIdentityValue{}
	for _, row := range httpRows {
		httpID[row.SubjectKey] = decodeHTTPIdentity(row.Value)
	}

	nameSet := estateNameSet(names)
	inEstate := func(host string) bool {
		if host == "" {
			return true
		}
		return estateAddrs[host] || nameSet[host]
	}

	subjects := map[string]struct{}{}
	for k := range certVal {
		subjects[k] = struct{}{}
	}
	for k := range httpID {
		subjects[k] = struct{}{}
	}

	facts := make([]signal.EndpointFacts, 0, len(subjects))
	for sub := range subjects {
		name, _ := splitEndpointName(sub)
		f := signal.EndpointFacts{Subject: sub, HasName: name != ""}
		if cv, ok := certVal[sub]; ok {
			f.CertMeasured = true
			f.CertOutcome = cv.Outcome
			f.CertDetails = certDetailsFromValue(cv, s.now(), name)
		}
		if id, ok := httpID[sub]; ok {
			f.HTTPResponded = id.Outcome == httpexchange.OutcomeResponded
			f.HTTPStatus = id.Status
			f.RedirectLocation = id.RedirectLocation
			if f.HTTPResponded && f.HTTPStatus >= 300 && f.HTTPStatus <= 399 && id.RedirectLocation != "" {
				_, host := signal.RedirectTarget(id.RedirectLocation)
				f.RedirectHostInEstate = inEstate(host)
			}
		}
		facts = append(facts, f)
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Subject < facts[j].Subject })
	return facts, nil
}

type certificateValue struct {
	Outcome    string      `json:"outcome"`
	Chain      []string    `json:"chain"`
	NotAfter   string      `json:"not_after"`
	NotBefore  string      `json:"not_before"`
	SANDNS     []string    `json:"san_dns"`
	SANIP      []string    `json:"san_ip"`
	ChainCerts []chainCert `json:"chain_certs"`
}

// These mirror connectoutcome's per-link parsed facts, so a producer edit must land here too.

type chainCert struct {
	Subject               string `json:"subject"`
	Issuer                string `json:"issuer"`
	SelfSignatureVerifies *bool  `json:"self_sig_verifies"`
	KeyAlg                string `json:"key_alg"`
	KeyBits               int    `json:"key_bits"`
	KeyParamN             int    `json:"key_n_bits"`
	SigDigest             string `json:"sig_digest"`
}

func decodeCertificate(raw []byte) certificateValue {
	var v certificateValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func certDetailsFromValue(v certificateValue, now time.Time, serverName string) *signal.CertDetails {
	if v.Outcome != signal.CertPresented {
		return nil
	}
	ref := now.UTC()
	d := &signal.CertDetails{}

	if v.NotAfter != "" {
		if na, err := time.Parse(time.RFC3339, v.NotAfter); err == nil {
			expired := !na.After(ref)
			expiring := na.After(ref) && !na.After(ref.Add(certExpiryWindow))
			d.Expired = &expired
			d.Expiring = &expiring
		}
	}

	if v.NotBefore != "" {
		if nb, err := time.Parse(time.RFC3339, v.NotBefore); err == nil {
			nyv := nb.After(ref)
			d.NotYetValid = &nyv
		}
	}

	// Under omitempty an empty san_dns is unreadable, so chain_certs is the read/unread witness.
	if len(v.ChainCerts) > 0 {
		if serverName != "" {
			m := sanMatchesName(v.SANDNS, serverName)
			d.SANMatchesName = &m
		}
		weak := weakKeyOrSignature(v.ChainCerts)
		d.WeakKeyOrSignature = &weak
		if c0 := v.ChainCerts[0]; c0.SelfSignatureVerifies != nil {
			ss := selfSignedOf(c0.Subject, c0.Issuer, *c0.SelfSignatureVerifies)
			d.SelfSigned = &ss
		}
	}
	return d
}

func selfSignedOf(subject, issuer string, selfSigVerifies bool) bool {
	// Shared so the two rules cannot disagree (docs/research/weak-key-and-signature.md §4.1).
	return subject == issuer && selfSigVerifies
}

func sanMatchesName(sanDNS []string, name string) bool {
	// A wildcard SAN admits no Name yet still matches one here: matching is not admitting (ADR-0060).
	nameLabels := dnsLabels(name)
	if len(nameLabels) == 0 {
		return false
	}
	// iPAddress SANs are ignored entirely, so an IP-keyed endpoint never matches on one.
	for _, entry := range sanDNS {
		if sanEntryMatches(entry, nameLabels) {
			return true
		}
	}
	return false
}

func dnsLabels(name string) []string {
	labels := strings.Split(name, ".")
	if n := len(labels); n > 0 && labels[n-1] == "" {
		labels = labels[:n-1]
	}
	if len(labels) == 1 && labels[0] == "" {
		return nil
	}
	return labels
}

func sanEntryMatches(entry string, nameLabels []string) bool {
	entryLabels := dnsLabels(entry)
	if len(entryLabels) == 0 {
		return false
	}
	stars := 0
	for _, l := range entryLabels {
		stars += strings.Count(l, "*")
	}
	if stars == 0 {
		return labelsEqualFold(entryLabels, nameLabels)
	}
	// The same octets read as a pattern to one client and a literal to the next, so refuse (ADR-0060).
	if stars != 1 || entryLabels[0] != "*" {
		return false
	}
	if len(entryLabels) != len(nameLabels) {
		return false
	}
	if nameLabels[0] == "" {
		return false
	}
	return labelsEqualFold(entryLabels[1:], nameLabels[1:])
}

func labelsEqualFold(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

func weakKeyOrSignature(chain []chainCert) bool {
	// A self-signed link skips the signature limb only (docs/research/weak-key-and-signature.md §4.1).
	weak := false
	for _, c := range chain {
		// An unnamed key algorithm is not weak rather than not-evaluable (weak-key-and-signature.md §4.2).
		switch c.KeyAlg {
		case "RSA":
			if c.KeyBits < 2048 {
				weak = true
			}
		case "ECDSA":
			if c.KeyBits < 224 {
				weak = true
			}
		case "DSA":
			if c.KeyBits < 2048 || c.KeyParamN < 224 {
				weak = true
			}
		}
		selfSig := c.SelfSignatureVerifies != nil && *c.SelfSignatureVerifies
		if !selfSignedOf(c.Subject, c.Issuer, selfSig) {
			if c.SigDigest == "MD5" || c.SigDigest == "SHA-1" {
				weak = true
			}
		}
	}
	return weak
}

func parseServicePair(key string) (pair vergecore.Pair, addr string, ok bool) {
	slash := strings.LastIndex(key, "/")
	if slash < 0 {
		return vergecore.Pair{}, "", false
	}
	hostPort, transport := key[:slash], key[slash+1:]
	ap, err := netip.ParseAddrPort(hostPort)
	if err != nil {
		return vergecore.Pair{}, "", false
	}
	return vergecore.Pair{Port: ap.Port(), Transport: vergecore.Transport(transport)}, ap.Addr().String(), true
}

func splitEndpointName(key string) (name, service string) {
	if at := strings.Index(key, "@"); at >= 0 {
		return key[:at], key[at+1:]
	}
	return "", key
}

func estateNameSet(names []signal.NameFacts) map[string]bool {
	// The redirect host arrives already lowercased, so the zone's own spelling must not decide a match.
	set := make(map[string]bool, len(names))
	for _, n := range names {
		if n.InEstate {
			set[strings.ToLower(n.Name)] = true
		}
	}
	return set
}

type signalInstanceView struct {
	SigID    string
	Signal   string
	Title    string
	Severity string
	SevRank  int
	Asset    string
	IP       string
	Port     string
	First    string
	Last     string
	Seen     string
	Href     string
}

func (s *server) deriveSignalInstances(ctx context.Context, censuses []signal.Census) ([]signalInstanceView, error) {
	type pair struct{ rule, subject string }

	var fired []pair
	var names, subjects []string
	for _, c := range censuses {
		for _, m := range c.Fired {
			fired = append(fired, pair{c.Rule, m.Subject})
			names = append(names, c.Rule)
			subjects = append(subjects, m.Subject)
		}
	}

	// A GET writes here, because a re-derivation cannot reconstruct a stable id or its first-seen.
	if len(fired) > 0 {
		if err := s.store.MintSignalInstances(ctx, db.MintSignalInstancesParams{
			SignalNames: names,
			SubjectKeys: subjects,
		}); err != nil {
			return nil, err
		}
	}

	rows, err := s.store.ListSignalInstances(ctx)
	if err != nil {
		return nil, err
	}
	type identity struct {
		id      int64
		first   time.Time
		firstOK bool
	}
	byPair := make(map[pair]identity, len(rows))
	for _, row := range rows {
		byPair[pair{row.SignalName, row.SubjectKey}] = identity{
			id: row.ID, first: row.FirstSeen.Time, firstOK: row.FirstSeen.Valid,
		}
	}

	now := s.now().UTC()
	out := make([]signalInstanceView, 0, len(fired))
	for _, p := range fired {
		id := byPair[p]
		sev, _ := signal.SeverityFor(p.rule)
		ip, port := subjectAddrPort(p.subject)
		v := signalInstanceView{
			SigID:    formatSigID(id.id),
			Signal:   p.rule,
			Title:    signalTitle(p.rule),
			Severity: sev.String(),
			SevRank:  sev.Rank(),
			Asset:    p.subject,
			IP:       ip,
			Port:     port,
			Last:     now.Format(time.RFC3339),
			Seen:     relTime(now, now),
			Href:     subjectHref(signal.SubjectKindFor(p.rule), p.subject),
		}
		if id.firstOK {
			v.First = id.first.UTC().Format(time.RFC3339)
		}
		out = append(out, v)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].SevRank != out[j].SevRank {
			return out[i].SevRank < out[j].SevRank
		}
		if out[i].Asset != out[j].Asset {
			return out[i].Asset < out[j].Asset
		}
		return out[i].Signal < out[j].Signal
	})
	return out, nil
}

func formatSigID(id int64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("SIG-%04d", id)
}

func subjectAddrPort(subject string) (ip, port string) {
	_, svc := splitEndpointName(subject)
	if p, addr, ok := parseServicePair(svc); ok {
		return addr, ":" + strconv.Itoa(int(p.Port))
	}
	return "", ""
}

func signalTitle(ruleName string) string {
	acronyms := map[string]string{
		"cname": "CNAME", "dns": "DNS", "ns": "NS", "tls": "TLS",
		"http": "HTTP", "https": "HTTPS", "san": "SAN", "ip": "IP", "url": "URL",
	}
	words := strings.Split(ruleName, "-")
	for i, w := range words {
		if a, ok := acronyms[w]; ok {
			words[i] = a
			continue
		}
		if i == 0 && w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
