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
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// The Signals screen (design-system/examples/console/Signals.jsx + SignalData.jsx,
// PARITY-CHART P2.2, #448). It renders a FLAT PER-INSTANCE table: one row per
// currently-fired (rule, subject) pair, carrying its rule's severity, a stable
// minted SIG-#### id, asset, port and last-seen, with a severity filter, a text
// filter, sortable columns, pagination, Open/Annotated/Withdrawn tabs with counts,
// a row Drawer, the AnnotationControl operator dial, and the typed-name descope
// ConfirmDialog. The old per-rule census grouping leaves the screen (ADR-0116: the
// design is normative for look AND functionality; severity is a real datum now,
// P0.1 #442, not a re-skin). The census is still evaluated data-side — it is what
// mints the per-instance rows — but it no longer paints.
//
// The web layer's only fold work is to compose the Derived corpus (resolution /
// dns-record / reachability / certificate / http-identity + the operator's zone
// file) into the per-subject snapshot the `Signal` engine evaluates; the engine
// owns every verdict, and deriveSignalInstances turns the fired census into the
// flat rows this screen draws.

// dnsRecordValue is the JSON payload of a dns-record observation (the shape the
// resolution-walk leaf emits). The handler reads only the CNAME target off the
// CNAME discriminator and the delegation's Lame verdict off the NS discriminator.
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

// annotationView is one declared `Annotation` shaped for rendering: the pair it
// is keyed on, the operator's reason, and the instant declared. There is no
// author cell and no status — every operator dial is unattributed (ADR-0073) and
// an Annotation carries neither a timeline nor an expiry. Orphan is derived on
// read and stored nowhere (ADR-0092): a row whose key is in no current
// population of its rule names a withdrawn or never-measured subject and matches
// nothing right now.
type annotationView struct {
	ID      int64
	Subject string
	Href    string
	Signal  string
	Reason  string
	At      string
	Orphan  bool
}

// signalRow is one rendered row of the flat per-instance Signals table
// (Signals.jsx + SignalData.jsx). It is a signalInstanceView (the P0.1 datum —
// severity, SIG id, asset, port, instants) enriched with the two facts the
// screen's tabs and Drawer add: whether the pair carries an operator Annotation
// (its reason and id), and whether it has withdrawn — its subject has left the
// rule's population, the world's act, never resolved by an operator (ADR-0092).
// ViewKey is the stable token the row Drawer opens on: a fired instance opens on
// its SIG id, an annotation-anchored row (the Annotated / Withdrawn tabs) opens on
// its annotation id, so the Drawer resolves without depending on a live mint.
//
// The trailing fields carry the spec Drawer's rule metadata + join data (#21j):
// the rule tags, an optional CVE, the rule's description prose, the rule id +
// version rendered together, the detecting vantage, the drift diff for the
// subject, and the span-derived history. On the honest live projection these are
// what the corpus can source (rule id from the rule name, the span-derived
// history from the instants); the design's curated fixture (served under devMode)
// carries the full authored set, byte-for-byte with the goldens.
type signalRow struct {
	signalInstanceView
	SevLabel    string // the severity capitalised for the badge label ("Critical")
	Annotated   bool
	AnnoReason  string
	AnnoID      int64
	Withdrawn   bool
	ViewKey     string
	DescopeHref string // per-row descope link (/signals?…&descope={ViewKey}, filters preserved)
	// Drawer (#21j) — rule metadata + join data.
	Tags        []string
	CVE         string
	Desc        string
	RuleID      string
	RuleVersion string
	DetectedBy  string
	Diff        *sigDiff
	History     []sigHistoryEntry
	idNum       int64 // the SIG id as a number, for the Id-column sort
	seenAge     int64 // minutes since last-seen, for the Seen-column sort (unseen sorts last)
}

// sigDiff is the before/after drift join the Drawer shows for a subject (#21j):
// a titled block of typed lines (add | remove | same). Nil where the subject has
// no drift transition to show.
type sigDiff struct {
	Title string
	Lines []sigDiffLine
}

type sigDiffLine struct {
	Type string // add | remove | same
	Text string
}

// sigHistoryEntry is one span-derived Drawer history event (#21j): a title, an
// optional detail line (mono when Mono), a right-aligned time token, and a Tone
// (accent | warn | danger | neutral) that colours the rail dot.
type sigHistoryEntry struct {
	Title  string
	Detail string
	Time   string
	Tone   string
	Mono   bool
}

// descopeView is the spec typed-confirm dialog's data (#21): the exact asset the
// operator must retype to arm the danger button, and the link that closes the
// dialog back to the current tab / filter / drawer.
type descopeView struct {
	Asset     string
	CloseHref string
}

// sigSort carries the sort state and the per-column toggle links for the sortable
// table headers (Severity / Asset / Id / Seen), precomputed so the template only
// renders them. The caret svg renders from Key/Dir directly (#21i — the *Arrow
// string holes retired).
type sigSort struct {
	Key, Dir                             string
	SevHref, AssetHref, IDHref, SeenHref string
}

// pageCell is one slot in the pagination control — a page number and its link, or
// an ellipsis gap. Precomputed windowed like Pagination.jsx so the control never
// changes width while paging.
type pageCell struct {
	Num      int
	Href     string
	Active   bool
	Ellipsis bool
}

// signalsForms carries a declare-form error back onto a re-rendered Signals page
// so a rejected declaration keeps its message and its typed values without a
// redirect.
type signalsForms struct {
	annoError   string
	annoSubject string
	annoSignal  string
	annoReason  string
}

// The frozen design-owned signals.tmpl (design-system/templates/signals.tmpl, package
// v3.9.0) is the view layer: it defines "signals" + the shared "sevbadge" /
// "sevbadge-md" / "withdrawnmark" partials (dashboard.tmpl consumes "sevbadge"). It is
// embedded read-only via the designfs package and parsed into the shared set here; the
// repo authors no markup/CSS/JS for /signals.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/signals.tmpl"))

func (s *server) signalsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// A VERGE_DEV build serves the design's curated fixtures.json signals slice so the
	// screen renders byte-for-byte for the pixel-parity harness (the 47-of-47 open count,
	// the authored rows, the drawer's rule metadata + drift join + span history, the
	// typed-confirm descope). A real deployment renders the honest live projection below.
	if s.devMode {
		s.render(w, "signals", s.signalsFixtureData(acct, r))
		return
	}
	s.renderSignals(w, r, acct, signalsForms{})
}

// signalsExport streams the CURRENT TAB's filtered rows as a downloadable CSV
// (Signals.jsx header; PARITY-CHART collision #6 — the button exports the current
// tab's filtered rows). It reads the same corpus the page evaluates, builds the
// same per-instance rows, applies the same text / severity filters off the query
// string, and emits one row per signal instance. It owns no mutation beyond the
// idempotent mint the read path already runs, and fabricates nothing: an empty tab
// produces a header-only file.
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

// writeSignalsExportCSV emits the flat per-instance rows as one table — one row per
// signal instance, carrying the same cells the screen's table shows plus both
// instants. The severity, id and port cells are controlled tokens; the signal
// (rule) name is controlled too; the asset cell is attacker-influenceable free text
// (a name ingested from CT logs), so it is passed through csvSafe.
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

// renderSignals builds the flat per-instance table view model and renders the
// Signals screen. It is the single render path the GET handler and the declare
// handler's failure case both use, so a rejected declaration re-renders the live
// page with its error banner.
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

	// Filter + sort state, threaded through the query string (the shell ships no
	// client table machinery, T0 seam).
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

	// Pagination — the Open tab only, exactly as Signals.jsx paginates only its
	// working surface. The other tabs list every row.
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

	// The current-view query string, so the Drawer, its scrim and the descope dialog
	// all return to exactly this tab / filter / sort / page.
	state := filterVals()
	state.Set("sort", sortKey)
	state.Set("dir", dir)
	if page > 1 {
		state.Set("page", strconv.Itoa(page))
	}
	closeHref := "/signals?" + state.Encode()

	// Per-column sort toggle links (reset the page) + their arrows.
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

	// Per-row descope link — the header Descope button drops (#21); descope now lives
	// on the row kebab / right-click menu and the drawer. Each visible row carries a
	// link that re-opens the current tab / filter with ?descope=<ViewKey>, which the
	// dialog resolves to the row's asset for the typed-confirm gate.
	for i := range rows {
		rows[i].DescopeHref = closeHref + "&descope=" + url.QueryEscape(rows[i].ViewKey)
	}

	// The row Drawer opens server-side via ?view=<ViewKey> against the active tab.
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

	// The descope typed-confirm dialog (#21) resolves ?descope=<ViewKey> to the row's
	// asset — the exact string the operator must retype — and returns to the current
	// view on cancel/submit. Admin-gated (the tmpl also gates on .IsAdmin).
	var descope *descopeView
	if dk := r.URL.Query().Get("descope"); dk != "" {
		for i := range base {
			if base[i].ViewKey == dk {
				descope = &descopeView{Asset: base[i].Asset, CloseHref: closeHref}
				break
			}
		}
	}

	// Export CSV is gated on the current tab having rows to export (Signals.jsx
	// disables the button when the tab is empty), and carries the current filters.
	hasExport := total > 0
	exportVals := filterVals()
	exportHref := "/signals/export?" + exportVals.Encode()

	s.render(w, "signals", map[string]any{
		"Title": "Signals", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":      "signals",
		"SignalCount":    len(open),
		"Tab":            tab,
		"OpenCount":      len(open),
		"AnnotatedCount": len(annotated),
		"WithdrawnCount": len(withdrawn),
		"Rows":           rows,
		"HasAny":         total > 0,
		"Shown":          shown,
		"Total":          total,
		"Q":              q,
		"Sev":            sevSel,
		"SevOptions":     []string{"All severities", "Critical", "High", "Medium", "Low", "Info"},
		"Sort":           sortState,
		"ClearHref":      "/signals?tab=" + tab,
		"ShowPagination": showPag,
		"Pages":          pages,
		"PrevHref":       prevHref,
		"NextHref":       nextHref,
		"PrevDisabled":   prevDisabled,
		"NextDisabled":   nextDisabled,
		"PageInfo":       pageInfo,
		"CloseHref":      closeHref,
		"ViewPrefix":     closeHref + "&view=",
		"ExportHref":     exportHref,
		"HasExport":      hasExport,
		"SelKey":         selKey,
		"Drawer":         drawer,
		"Descope":        descope,
		"AnnoError":      forms.annoError,
	})
}

// enrichSignalDrawer folds the honest live projection of the Drawer's rule metadata
// + join data onto a row about to open in the drawer (#21j). Three fields are REAL
// derived-store joins built here on the production path: the rule id is the rule name
// and the rule version is that rule's own Version().String() vector (ruleVersions,
// keyed by name over the whole shipped corpus); the drift diff is the subject's most
// recent value transition joined out of the Span corpus (signalDrift, the same
// ListSpansForSubject → Breaks read the drill-down timelines and RunDetail's Outcome
// use), nil where the subject has no drift; and the history is span-derived from the
// instants the row carries. Four fields are GENUINELY ABSENT from the corpus — no
// rules registry carries a rule's tags / CVE / description prose, and a fired signal
// is a cross-vantage composition so signal_instance has no single detecting vantage —
// so Tags / CVE / Desc / DetectedBy are omitted rather than invented (the curated
// fixture served under devMode carries the authored set for the pixel goldens).
func (s *server) enrichSignalDrawer(r *http.Request, row *signalRow) {
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

// ruleVersions maps every shipped rule's name to its Version().String() vector, built
// once over the whole corpus (the five Name rules, the ten Endpoint rules, the two
// Service rules — signal.All / AllEndpointRules / AllServiceRules). The drawer's Rule
// cell renders "{RuleID}@{RuleVersion}" off this, so the version is the rule's own
// deterministic vector (rule@vN|leaf/vM|…), not a guess. A name with no shipped rule
// (never expected for a fired instance) maps to "" and renders the id alone.
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

// signalDrift joins the Drawer's before/after drift diff (#21j) out of the subject's
// Span corpus — the same derived-store read the drill-down timelines and RunDetail's
// Outcome perform (ListSpansForSubject → per-(facet,discriminator) timelines with
// Breaks derived on read, ADR-0007/ADR-0008). It surfaces the subject's MOST RECENT
// value transition: across the subject's timelines, the open span whose value differs
// from its immediately-preceding closed span, opened latest. The before/after are the
// shared valueLabel summaries (the same vocabulary driftDiff renders), so a Signals
// drawer diff reads exactly like the Drift feed's. It returns nil — an honest
// not-present, never fabricated — where the subject has no such transition, and
// degrades to nil on any read error (the drawer diff is diagnostic, not load-bearing).
func (s *server) signalDrift(r *http.Request, kind, key string) *sigDiff {
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
		// spanView.OpenedAt is the fixed-width "2006-01-02 15:04 UTC" stamp, so a plain
		// string compare orders the transitions chronologically — the latest opener wins.
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

// buildSignalTabs folds the Derived corpus into the flat per-instance rows and
// partitions them into the screen's three tabs. Open is every currently-fired
// (rule, subject) pair. Annotated is every operator acceptance whose subject is
// still a current member of its rule's population; Withdrawn is every acceptance
// whose subject has left it — orphan on read, the world's act (ADR-0092). The two
// annotation-anchored tabs read each pair's persisted SIG id / first-seen where it
// has one, and its last-seen where it is also currently firing.
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

	// population[rule][subject] — every subject a rule censuses (for orphan
	// detection). fired[(rule,subject)] — the pairs firing right now.
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

	// The flat per-instance datum: one row per currently-fired pair (P0.1, #442).
	instances, err := s.deriveSignalInstances(ctx, censuses)
	if err != nil {
		return nil, nil, nil, err
	}

	// Persisted identity for every minted pair, so an annotation-anchored row (which
	// need not be currently firing) can still show its SIG id and first-seen.
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

// instIdent is a persisted signal_instance identity — the stable id and first-seen
// instant a pure re-derivation cannot reconstruct.
type instIdent struct {
	id      int64
	first   time.Time
	firstOK bool
}

// annotationRow shapes one operator acceptance into a flat per-instance row for the
// Annotated / Withdrawn tabs. Its severity is the rule's (assigned per rule); its
// asset / ip / port fall out of the subject key; its SIG id and first-seen come from
// the persisted identity where the pair has ever fired, and its last-seen is now
// only where it is also firing now (else its last-known instant, never invented).
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
		// The annotation-anchored tabs open the Drawer on the annotation id (a plain
		// number), distinct from the Open tab's `SIG-####` ViewKey — the two id spaces
		// never collide, and a withdrawn pair need not be currently minted to open.
		ViewKey: strconv.FormatInt(av.ID, 10),
	}
}

// filterSignalRows applies the screen's text and severity filters — the same
// predicate Signals.jsx runs: a case-insensitive substring over the row's title,
// asset, id and signal, and an exact severity match. It returns a fresh slice, so
// the caller's base rows are untouched.
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

// sortSignalRows sorts in place by the named column and direction, with a stable
// deterministic tiebreak (severity, then asset, then id) so equal keys never
// reorder between renders. The default column is severity ascending — critical
// first, matching SignalData.jsx's SEV_ORDER.
func sortSignalRows(rows []signalRow, key, dir string) {
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
		default: // sev
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

// pageWindow returns the windowed page-number slots (with -1 marking an ellipsis
// gap), the constant-width windowing Pagination.jsx uses so the control never
// changes width while paging.
func pageWindow(page, pageCount int) []int {
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

// sevLabel capitalises a severity token for the badge label ("critical" ->
// "Critical"); the `.sev` class then upper-cases it in CSS, matching SeverityBadge.
func sevLabel(sev string) string {
	if sev == "" {
		return ""
	}
	return strings.ToUpper(sev[:1]) + sev[1:]
}

// parseSigNum reads the numeric identity out of a `SIG-####` display id for the
// Id-column sort; a row with no minted id (0) sorts first ascending.
func parseSigNum(id string) int64 {
	if n, err := strconv.ParseInt(strings.TrimPrefix(id, "SIG-"), 10, 64); err == nil {
		return n
	}
	return 0
}

// seenAgeMinutes is the row's last-seen age in whole minutes, the Seen-column sort
// key. A row with no last-seen instant sorts last (largest age).
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

// annotationViews shapes the stored annotations for rendering, marking each as
// orphan (naming no current member of its rule) purely on read.
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

// buildNameFacts assembles the current Derived snapshot the engine reads: the
// composed cross-class resolution per Name (folding resolution-walk's outcome,
// the NS delegation's Lame verdict, and wildcard-discrimination's Shadowed), the
// internet-class view, the dns-record CNAME target, and the operator's zone
// declarations. It reads resolution / dns-record / membership only — the five
// rules are Name-only and need no reachability facet.
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

	// Per Name: the resolution value per Vantage class.
	byClass := map[string]map[string]resolutionValue{}
	for _, row := range resRows {
		m := byClass[row.SubjectKey]
		if m == nil {
			m = map[string]resolutionValue{}
			byClass[row.SubjectKey] = m
		}
		m[row.Class] = decodeResolution(row.Value)
	}

	// Per Name: the CNAME target and the delegation's Lame verdict.
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

	// The operator's zone declarations: the owner-name set (the zone rules'
	// domain) and the declared domains (the containment test for InDeclaredZone).
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

	// The candidate universe: every Name we have a resolution for, plus every
	// Name a zone file declares (so a declared name that withdrew or was never
	// measured still enters its rule's domain — a signal's lifecycle is its
	// evidence's, not its subject's membership).
	names := map[string]struct{}{}
	for name := range byClass {
		names[name] = struct{}{}
	}
	for name := range declared {
		names[name] = struct{}{}
	}

	// First pass: the composed cross-class resolution per Name, so a CNAME rule
	// can read its target's outcome.
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
			f.TargetResolution = composed[target].outcome // "" when the target was never measured
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

// composedResolution is the one resolution value the four cross-class rules read,
// folded from the per-class observations and the NS delegation.
type composedResolution struct {
	outcome   string
	addresses []string
	inEstate  bool
}

// composeResolution folds the per-class resolution values into one composed
// outcome (CONTEXT.md `Membership`, ADR-0080/ADR-0086). Shadowed wins (it is a
// value about our own sight and cites nothing); then a Resolved answer at any
// class (its address set is the union); then the NS delegation's Lame; then
// NoData; then a withdrawal only where every observed class read NameError; then
// Gap. A Name is in the estate where some class observed it and it is not a
// cross-class NameError.
func composeResolution(classes map[string]resolutionValue, lame bool) composedResolution {
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

// buildSignalCorpus assembles the full Derived snapshot the seventeen-rule set
// reads: the per-Name facts (the five Name-only rules), the per-Service facts (the
// two Service rules), and the per-Endpoint facts (the ten Endpoint rules). Each is
// folded from the observation corpus generically — by facet name and value — so a
// rule reading a facet whose producing data is not yet present (a certificate's
// parsed leaf, or #199's tls-acceptance) renders `not-evaluable` or a no-population
// panel rather than a compile-time dependency on a leaf that has not landed.
func (s *server) buildSignalCorpus(r *http.Request) (signal.Corpus, error) {
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

// buildServiceFacts folds the internet-class reachability corpus into the
// per-Service snapshot the two Service rules read. It also returns the set of
// estate addresses — the Address leg of every Service subject — which the Endpoint
// redirect rule reads to decide whether a redirect target host is in the estate.
// The `tls-acceptance` facet (tls-1.0-accepted's domain) is not read: its leaf
// (#199) lands concurrently, so every Service is left outside that rule's domain
// (a no-population panel) rather than importing a leaf that may not exist yet.
func (s *server) buildServiceFacts(r *http.Request) ([]signal.ServiceFacts, map[string]bool, error) {
	rows, err := s.store.ListServiceReachabilitySpansByClass(r.Context())
	if err != nil {
		return nil, nil, err
	}
	vc := vergecore.Default()

	// subject -> class -> reach leg. Reading the SPAN (not the latest observation)
	// gives us `is_gap`: a blanket responder's reach is a Gap, and a Gap leg reads
	// as absent (ADR-0104), so it never becomes a value the rule can fire on.
	type leg struct {
		outcome string
		isGap   bool
	}
	byClass := map[string]map[string]leg{}
	order := []string{}
	for _, row := range rows {
		m := byClass[row.SubjectKey]
		if m == nil {
			m = map[string]leg{}
			byClass[row.SubjectKey] = m
			order = append(order, row.SubjectKey)
		}
		m[row.Class] = leg{outcome: decodeReachability(row.Value).Outcome, isGap: row.IsGap}
	}

	estateAddrs := map[string]bool{}
	facts := make([]signal.ServiceFacts, 0, len(order))
	for _, sub := range order {
		f := signal.ServiceFacts{Subject: sub}
		if pair, addr, ok := parseServicePair(sub); ok {
			// A Service exists only for a probed pair (TCP); the sensitive half is
			// always in the probed union, so IsSensitive on the TCP pair is exactly
			// "on the sensitive list AND probed" (the ticket's domain restriction). A
			// blanketed Service is still a subject here — its Address is real and in
			// the estate — so it keeps its census membership; only its reach is absent.
			f.OnSensitiveList = pair.Transport == vergecore.TCP && vc.IsSensitive(pair)
			estateAddrs[addr] = true
		}
		// A blanket responder's internet reach is a Gap: the leg reads as absent, so
		// HasInternetReach stays false and `sensitive-port-reached-from-internet`
		// returns not-evaluable — the signal is damped at the measurement, never in
		// the rule (ADR-0104 Decision §3). A Gap is not a `reachability` value either,
		// so a blanketed port never enters an open-port count.
		if l, ok := byClass[sub]["internet"]; ok && !l.isGap && l.outcome != "" {
			f.HasInternetReach = true
			f.InternetReach = l.outcome
		}
		facts = append(facts, f)
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Subject < facts[j].Subject })
	return facts, estateAddrs, nil
}

// buildEndpointFacts folds the per-Endpoint certificate and http-identity corpus
// into the snapshot the ten Endpoint rules read. The estate membership a redirect
// target is tested against is the union of the current Name subjects and the
// Service addresses — both Derived, so the redirect-to-host rule's version
// composes their leaves. The certificate value carries only its outcome tag and a
// fingerprint chain, so the five certificate-detail rules leave CertDetails nil
// (a presented chain renders `not-evaluable`).
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

	certOutcome := map[string]string{}
	for _, row := range certRows {
		certOutcome[row.SubjectKey] = decodeCertificate(row.Value).Outcome
	}
	httpID := map[string]httpIdentityValue{}
	for _, row := range httpRows {
		httpID[row.SubjectKey] = decodeHTTPIdentity(row.Value)
	}

	// The estate name set the redirect rule reads: a redirect target host is in the
	// estate where it names a current Name or a Service's Address.
	nameSet := estateNameSet(names)
	inEstate := func(host string) bool {
		if host == "" {
			return true // a relative redirect stays on this origin
		}
		return estateAddrs[host] || nameSet[host]
	}

	subjects := map[string]struct{}{}
	for k := range certOutcome {
		subjects[k] = struct{}{}
	}
	for k := range httpID {
		subjects[k] = struct{}{}
	}

	facts := make([]signal.EndpointFacts, 0, len(subjects))
	for sub := range subjects {
		name, _ := splitEndpointName(sub)
		f := signal.EndpointFacts{Subject: sub, HasName: name != ""}
		if o, ok := certOutcome[sub]; ok {
			f.CertMeasured = true
			f.CertOutcome = o
		}
		if id, ok := httpID[sub]; ok {
			// Only a `responded` Endpoint is inside the HTTP rules' domain; a reached
			// Service that returned no-http-response is a value, but outside them.
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

// certificateValue is the JSON payload of a certificate observation — the closed
// union outcome tag and (only on a presentation) the fingerprint chain (#197).
// The engine reads the outcome; the parsed leaf attributes the five detail rules
// need are not stored, so a presented chain renders `not-evaluable`.
type certificateValue struct {
	Outcome string   `json:"outcome"`
	Chain   []string `json:"chain"`
}

func decodeCertificate(raw []byte) certificateValue {
	var v certificateValue
	_ = json.Unmarshal(raw, &v)
	return v
}

// parseServicePair splits a Service key `address:port/transport` into its
// verge-core pair and its Address. A key that does not parse yields ok=false —
// the Service is still a census subject, just never on the sensitive list.
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

// splitEndpointName splits an Endpoint key `name@service` into its Name (empty for
// the nameless endpoint) and Service legs at the first `@` — neither a DNS Name
// nor a Service key contains one.
func splitEndpointName(key string) (name, service string) {
	if at := strings.Index(key, "@"); at >= 0 {
		return key[:at], key[at+1:]
	}
	return "", key
}

// estateNameSet is the set of current Name subjects, keyed lowercased so a
// redirect host (already lowercased by RedirectTarget) matches regardless of the
// zone's spelling.
func estateNameSet(names []signal.NameFacts) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		if n.InEstate {
			set[strings.ToLower(n.Name)] = true
		}
	}
	return set
}

// signalInstanceView is one currently-fired signal instance — a (rule, subject)
// pair the engine placed under `fired` right now — shaped for the flat per-instance
// table SignalData.jsx renders (P0.1, #442).
//
// Every field is real, none fabricated. SigID and First are read from the persisted
// signal_instance identity; Severity/SevRank come from the rule (assigned per rule
// in internal/signal); Last is this derivation instant (the pair is confirmed
// firing now, so last-seen is now); Asset/IP/Port fall out of the subject key.
// CVE, tags and the long remediation description SignalData.jsx also carries are
// genuinely not derivable from the current corpus and are left off rather than
// invented — the spec marks them optional.
type signalInstanceView struct {
	SigID    string // "SIG-####", formatted from the stable minted identity
	Signal   string // the rule name — the named fact (never "finding"/"fingerprint")
	Title    string // the rule name rendered for a human
	Severity string // the rule's severity: critical | high | medium | low | info
	SevRank  int    // index in signal.SevOrder — 0 = critical, the severity sort key
	Asset    string // the subject key (Name, Service or Endpoint)
	IP       string // the subject's address, where the key carries one ("" for a Name)
	Port     string // the subject's port as ":NNNN", where the key carries one
	First    string // first-seen instant, RFC3339 (persisted; "" if never minted)
	Last     string // last-seen instant, RFC3339 — the current derivation instant
	Seen     string // last-seen rendered relative to now ("4m", "now")
	Href     string // route-aware drill-down to the subject
}

// deriveSignalInstances folds the live censuses into the flat per-instance signal
// datum (P0.1, #442): one row per currently-fired (rule, subject) pair. It mints a
// stable identity for every fired pair (idempotent — an already-firing pair keeps
// its id and first-seen), reads the identities back, and shapes each into a
// signalInstanceView with its rule's severity and its instants. Ordered by the
// severity ramp, then subject, then rule, so the exposed table is deterministic and
// already in the design's critical→info order.
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

	// Mint identity for the whole current fired set in one idempotent write. Nothing
	// fires → nothing to mint, and the list below returns the historical rows (none
	// of which match, so no instance renders).
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

// formatSigID renders the stable minted identity as the console's `SIG-####`
// display id (SignalData.jsx). A zero id (a pair with no identity row — not
// expected after a mint) renders empty rather than "SIG-0000".
func formatSigID(id int64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("SIG-%04d", id)
}

// subjectAddrPort pulls the address and port out of a subject key for the
// per-instance table's IP / Port columns. A Service key is `address:port/transport`
// and an Endpoint key is `name@address:port/transport`; a bare Name carries
// neither, so both come back empty. It reuses the same parse the engine's fold
// uses, so the columns agree with the census's own reading of the key.
func subjectAddrPort(subject string) (ip, port string) {
	_, svc := splitEndpointName(subject)
	if p, addr, ok := parseServicePair(svc); ok {
		return addr, ":" + strconv.Itoa(int(p.Port))
	}
	return "", ""
}

// signalTitle renders a rule name as the human title the per-instance row shows
// (SignalData.jsx `title`). The rule name is the source of truth (never
// "finding"/"fingerprint"); this only presents it — hyphens become spaces, known
// acronyms are upper-cased, and the first word is capitalised.
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
