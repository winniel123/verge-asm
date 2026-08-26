package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
)

// The read-only /api/v1 JSON surface (#390, ADR-0123; A3 of #658). Every route here
// is a thin, stable JSON projection of a read the session-authed HTML surface already
// wraps — the same store reads and the same in-process builders the pages call, with
// no new derivation and no store method of its own. The surface is mounted through
// A2's apiBearer spine (api_auth.go): a request reaches a handler below ONLY once the
// instance's api_enabled flag is set, the verb is GET, and a Bearer personal token has
// resolved to a live account. The bearer path shares no machinery with the session
// path — no cookie is read or set, no template is rendered, no redirect-to-signin is
// ever emitted — so a stolen cookie cannot drive the API and a stolen token cannot
// drive the HTML app. When the surface is off every path 404s, byte-indistinguishable
// from a build with no API at all.
//
// Resource list (settled here against the live-tier read gate @as_of/@floor_cadences,
// mirroring the existing session reads — documented on #662):
//
//	GET /api/v1/inventory  — every open span grouped by subject      (buildInventory / ListAllOpenSpans)
//	GET /api/v1/subjects   — the current Name/Service/Endpoint census (ListCurrent*Subjects, live-tier gated)
//	GET /api/v1/drift      — the batch-grouped transition feed (7d)   (buildDriftFeed / ListRecentDriftEvents)
//	GET /api/v1/signals    — the fired-signal census (open/annotated/withdrawn) (buildSignalTabs)
//	GET /api/v1/coverage   — the aperture census meters per scope     (apertureMeters / ListSeeds+ListZoneDeclarations)
//
// A path under /api/v1 that names no resource answers a bare 404 on the JSON surface
// (the trailing catch-all in mountAPIv1), the same body a disabled surface returns, so
// an unknown path is indistinguishable from a disabled one and never reaches the HTML
// home the app's GET / catch-all would otherwise serve.

// mountAPIv1 registers the read-only /api/v1 JSON tree on the shared mux, each route
// wrapped in A2's apiBearer spine. It is deliberately a single call the router (#662)
// adds to handler() as one localized line, so sibling tickets appending their own
// routes to handlers.go in parallel union-merge without touching this block.
//
// Every resource is registered GET-only (ADR-0123 §1: no mutating verb is expressible
// under /api/v1). A GET-only pattern is a strict subset of the app's GET / catch-all,
// so it wins for its path without conflict; a method-less pattern would instead collide
// with GET / (it matches more methods but a narrower path — neither dominates — which
// net/http rejects at registration). A non-GET verb on a resource path matches a GET
// pattern under the wrong method, so the mux itself refuses it 405 before any handler
// runs — the read-only contract holds without a mutating route ever existing. apiBearer
// then owns the surface-off gate: a disabled instance 404s every GET path before it
// authenticates (surface-off beats auth, ADR-0123 §2). The trailing catch-all keeps an
// unknown /api/v1 GET on the JSON surface — a bare 404 — rather than letting it fall
// through to GET / and answer on the HTML/session surface (a redirect to sign-in).
func (s *server) mountAPIv1(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/inventory", s.apiBearer(s.apiInventory))
	mux.HandleFunc("GET /api/v1/subjects", s.apiBearer(s.apiSubjects))
	mux.HandleFunc("GET /api/v1/drift", s.apiBearer(s.apiDrift))
	mux.HandleFunc("GET /api/v1/signals", s.apiBearer(s.apiSignals))
	mux.HandleFunc("GET /api/v1/coverage", s.apiBearer(s.apiCoverage))

	// An unknown /api/v1 subpath 404s on the JSON surface. This subtree is more specific
	// than GET / (so it catches these paths instead of the HTML home) and less specific
	// than the exact resource patterns above (so they still win for their own paths). It
	// reuses apiNotFound — net/http's bare "404 page not found", byte-identical to the
	// body a disabled surface returns — and reads neither cookie nor template.
	mux.HandleFunc("GET /api/v1/", apiNotFound)
}

// writeAPIJSON encodes v as the sole body of a 200 application/json response. It is the
// one exit every read handler below takes on success: it never renders a template,
// never reads or writes a cookie, and never redirects, so the JSON surface stays fully
// disjoint from the session/HTML surface (ADR-0123 §3). An encode failure is logged;
// the header is already committed by then, so nothing further can be said on the wire.
func writeAPIJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("web: api: encode response: %v", err)
	}
}

// apiReadError answers a store-read failure on the JSON surface with a 500 whose body
// is JSON, never the HTML error page (which reads the session account). It carries no
// internal detail — the message is a fixed label, the underlying error only logged —
// so a read failure leaks neither the estate's shape nor a stack.
func apiReadError(w http.ResponseWriter, what string, err error) {
	log.Printf("web: api: %s: %v", what, err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
}

// apiInstant renders a nullable persisted instant as an RFC-3339 UTC string, or the
// empty string when the row carries no instant — the same honest "absent" the HTML
// surface shows, never a fabricated zero time.
func apiInstant(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.UTC().Format(time.RFC3339)
}

// --- inventory --------------------------------------------------------------

type apiInventoryResponse struct {
	Groups []apiInventoryGroup `json:"groups"`
}

type apiInventoryGroup struct {
	Kind     string                `json:"kind"`
	Label    string                `json:"label"`
	Subjects []apiInventorySubject `json:"subjects"`
}

type apiInventorySubject struct {
	Kind   string              `json:"kind"`
	Key    string              `json:"key"`
	Type   string              `json:"type"`
	Facets []apiInventoryFacet `json:"facets"`
}

type apiInventoryFacet struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Gap   bool   `json:"gap"`
	Since string `json:"since"`
}

// apiInventory projects the open-span inventory the Inventory screen renders (#243,
// ADR-0105): every open span grouped by subject, read straight off the derived span
// corpus through the exact same buildInventory the page and its CSV export call. A Gap
// facet — a value the system currently cannot state — carries gap=true with an empty
// value, never a zero standing in for a real read.
func (s *server) apiInventory(w http.ResponseWriter, r *http.Request, _ db.Account) {
	rows, err := s.store.ListAllOpenSpans(r.Context())
	if err != nil {
		apiReadError(w, "inventory: list all open spans", err)
		return
	}
	groups := buildInventory(rows)
	out := apiInventoryResponse{Groups: make([]apiInventoryGroup, 0, len(groups))}
	for _, g := range groups {
		grp := apiInventoryGroup{Kind: g.Kind, Label: g.Label, Subjects: make([]apiInventorySubject, 0, len(g.Subjects))}
		for _, sub := range g.Subjects {
			s := apiInventorySubject{Kind: sub.Kind, Key: sub.Key, Type: sub.Type, Facets: make([]apiInventoryFacet, 0, len(sub.Facets))}
			for _, f := range sub.Facets {
				s.Facets = append(s.Facets, apiInventoryFacet{Label: f.Label, Value: f.Summary, Gap: f.IsGap, Since: f.Since})
			}
			grp.Subjects = append(grp.Subjects, s)
		}
		out.Groups = append(out.Groups, grp)
	}
	writeAPIJSON(w, out)
}

// --- subjects ---------------------------------------------------------------

type apiSubjectsResponse struct {
	Names     []apiSubject `json:"names"`
	Services  []apiSubject `json:"services"`
	Endpoints []apiSubject `json:"endpoints"`
}

type apiSubject struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	ObservedAt string `json:"observed_at,omitempty"`
}

// apiSubjects projects the current subject census the Subjects reads back (#189/#195/
// #198): every Name, Service and Endpoint currently in the estate. Each of the three
// reads goes through the live-tier gate (#237, ADR-0041) with the caller's read instant
// (@as_of = s.obsAsOf()) and k (@floor_cadences = retention.FloorCadences) — exactly as
// the dashboard, shell and search reads do — so an evidential row past its per-timeline
// bound is structurally unreadable here, never merely awaiting the Retirer's sweep.
func (s *server) apiSubjects(w http.ResponseWriter, r *http.Request, _ db.Account) {
	ctx := r.Context()
	asOf := s.obsAsOf()

	names, err := s.store.ListCurrentNameSubjects(ctx, db.ListCurrentNameSubjectsParams{
		Search: "", AsOf: asOf, FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		apiReadError(w, "subjects: list name subjects", err)
		return
	}
	services, err := s.store.ListCurrentServiceSubjects(ctx, db.ListCurrentServiceSubjectsParams{
		Search: "", AsOf: asOf, FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		apiReadError(w, "subjects: list service subjects", err)
		return
	}
	endpoints, err := s.store.ListCurrentEndpointSubjects(ctx, db.ListCurrentEndpointSubjectsParams{
		Search: "", AsOf: asOf, FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		apiReadError(w, "subjects: list endpoint subjects", err)
		return
	}

	out := apiSubjectsResponse{
		Names:     make([]apiSubject, 0, len(names)),
		Services:  make([]apiSubject, 0, len(services)),
		Endpoints: make([]apiSubject, 0, len(endpoints)),
	}
	for _, row := range names {
		out.Names = append(out.Names, apiSubject{Key: row.SubjectKey, Value: string(row.Value), ObservedAt: apiInstant(row.ObservedAt)})
	}
	for _, row := range services {
		out.Services = append(out.Services, apiSubject{Key: row.SubjectKey, Value: string(row.Value), ObservedAt: apiInstant(row.ObservedAt)})
	}
	for _, row := range endpoints {
		out.Endpoints = append(out.Endpoints, apiSubject{Key: row.SubjectKey, Value: string(row.Value), ObservedAt: apiInstant(row.ObservedAt)})
	}
	writeAPIJSON(w, out)
}

// --- drift ------------------------------------------------------------------

type apiDriftResponse struct {
	Period          string          `json:"period"`
	TransitionCount int             `json:"transition_count"`
	Movement        map[string]int  `json:"movement"`
	Batches         []apiDriftBatch `json:"batches"`
}

type apiDriftBatch struct {
	Label  string          `json:"label"`
	Meta   string          `json:"meta"`
	Events []apiDriftEvent `json:"events"`
}

type apiDriftEvent struct {
	Change  string `json:"change"`
	Family  string `json:"family"`
	Subject string `json:"subject"`
	Detail  string `json:"detail"`
	Time    string `json:"time"`
	Reason  string `json:"reason,omitempty"`
}

// apiDrift projects the batch-grouped transition feed the Drift screen renders (#288,
// ADR-0111) for the design's default 7d window: every span open/close event a Batch
// caused, classified into one of the six change kinds and grouped by batch through the
// exact same ListRecentDriftEvents read and buildDriftFeed builder the page's live path
// uses, under the same driftFeedLimit cap. It reads span and batch only, never dispatch
// (ADR-0041). The per-kind movement tally rides along as the page's Movement summary.
func (s *server) apiDrift(w http.ResponseWriter, r *http.Request, _ db.Account) {
	period := resolveDriftPeriod(driftDefaultPeriod)
	rows, err := s.store.ListRecentDriftEvents(r.Context(), db.ListRecentDriftEventsParams{
		Since: s.driftSince(period), MaxEvents: driftFeedLimit,
	})
	if err != nil {
		apiReadError(w, "drift: list recent drift events", err)
		return
	}
	groups, movement := buildDriftFeed(rows, s.now())

	out := apiDriftResponse{Period: period.Token, Movement: movement, Batches: make([]apiDriftBatch, 0, len(groups))}
	for _, g := range groups {
		out.TransitionCount += len(g.Events)
		b := apiDriftBatch{Label: g.Label, Meta: g.Meta, Events: make([]apiDriftEvent, 0, len(g.Events))}
		for _, e := range g.Events {
			b.Events = append(b.Events, apiDriftEvent{
				Change: e.Change, Family: e.Family, Subject: e.Subject, Detail: e.Detail, Time: e.Time, Reason: e.Reason,
			})
		}
		out.Batches = append(out.Batches, b)
	}
	if out.Movement == nil {
		out.Movement = map[string]int{}
	}
	writeAPIJSON(w, out)
}

// --- signals ----------------------------------------------------------------

type apiSignalsResponse struct {
	Open      []apiSignal `json:"open"`
	Annotated []apiSignal `json:"annotated"`
	Withdrawn []apiSignal `json:"withdrawn"`
}

type apiSignal struct {
	ID        string `json:"id"`
	Signal    string `json:"signal"`
	Severity  string `json:"severity"`
	Asset     string `json:"asset"`
	IP        string `json:"ip,omitempty"`
	Port      string `json:"port,omitempty"`
	FirstSeen string `json:"first_seen,omitempty"`
	LastSeen  string `json:"last_seen,omitempty"`
}

// apiSignals projects the fired-signal census the Signals screen renders (P0.1, #442):
// the flat per-instance rows of the three tabs — currently-open, annotated and
// withdrawn — built through the exact same buildSignalTabs the page and its CSV export
// call. The idempotent identity mint that read runs (an already-firing pair keeps its
// id and first-seen) is intrinsic to the census read the session surface already wraps,
// not a mutation of the estate; the API mirrors that read verbatim and adds nothing.
func (s *server) apiSignals(w http.ResponseWriter, r *http.Request, _ db.Account) {
	open, annotated, withdrawn, err := s.buildSignalTabs(r)
	if err != nil {
		apiReadError(w, "signals: build signal tabs", err)
		return
	}
	writeAPIJSON(w, apiSignalsResponse{
		Open:      projectSignals(open),
		Annotated: projectSignals(annotated),
		Withdrawn: projectSignals(withdrawn),
	})
}

func projectSignals(rows []signalRow) []apiSignal {
	out := make([]apiSignal, 0, len(rows))
	for _, r := range rows {
		out = append(out, apiSignal{
			ID: r.SigID, Signal: r.Signal, Severity: r.Severity,
			Asset: r.Asset, IP: r.IP, Port: r.Port,
			FirstSeen: r.First, LastSeen: r.Last,
		})
	}
	return out
}

// --- coverage ---------------------------------------------------------------

type apiCoverageResponse struct {
	Meters []apiCoverageMeter `json:"meters"`
}

type apiCoverageMeter struct {
	Label   string  `json:"label"`
	Counted string  `json:"counted"`
	Total   *string `json:"total"`
	Unit    string  `json:"unit"`
	Pct     int     `json:"pct"`
	Detail  string  `json:"detail"`
}

// apiCoverage projects the aperture census the Coverage screen's meter card renders
// (#301, ADR-0110; SPEC-CHANGE #19c): one CoverageMeter per declared scope, built
// through the exact same apertureMeters the page's live path calls over the seed and
// zone reads. Total is null for a name scope (a census bar — it enumerates nothing on
// its own) and a pre-formatted string for an address scope, exactly as the screen shows;
// neither claims a proportion of the estate (ADR-0072). The zone read feeds only a name
// scope's declared-name count, so it is best-effort — a zone failure leaves that meter's
// census at zero rather than failing the whole projection, mirroring the page.
func (s *server) apiCoverage(w http.ResponseWriter, r *http.Request, _ db.Account) {
	ctx := r.Context()
	seeds, err := s.store.ListSeeds(ctx)
	if err != nil {
		apiReadError(w, "coverage: list seeds", err)
		return
	}
	var zones []db.ListZoneDeclarationsRow
	if z, zerr := s.store.ListZoneDeclarations(ctx); zerr == nil {
		zones = z
	} else {
		log.Printf("web: api: coverage: list zone declarations: %v", zerr)
	}
	meters := apertureMeters(seeds, zones)

	out := apiCoverageResponse{Meters: make([]apiCoverageMeter, 0, len(meters))}
	for _, m := range meters {
		out.Meters = append(out.Meters, apiCoverageMeter{
			Label: m.Label, Counted: m.Counted, Total: m.Total, Unit: m.Unit, Pct: m.Pct, Detail: m.Detail,
		})
	}
	writeAPIJSON(w, out)
}
