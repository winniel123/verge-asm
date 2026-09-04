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

func (s *server) mountAPIv1(mux *http.ServeMux) {
	// A method-less pattern neither dominates nor is dominated by GET /, which net/http refuses.
	mux.HandleFunc("GET /api/v1/inventory", s.apiBearer(s.apiInventory))
	mux.HandleFunc("GET /api/v1/subjects", s.apiBearer(s.apiSubjects))
	mux.HandleFunc("GET /api/v1/drift", s.apiBearer(s.apiDrift))
	mux.HandleFunc("GET /api/v1/signals", s.apiBearer(s.apiSignals))
	mux.HandleFunc("GET /api/v1/coverage", s.apiBearer(s.apiCoverage))

	// Without this subtree an unknown path falls to GET / and answers on the HTML surface.
	mux.HandleFunc("GET /api/v1/", apiNotFound)
}

func writeAPIJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("web: api: encode response: %v", err)
	}
}

func apiReadError(w http.ResponseWriter, what string, err error) {
	// The HTML error page reads a session account, which a bearer request has none of.
	log.Printf("web: api: %s: %v", what, err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
}

func apiInstant(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.UTC().Format(time.RFC3339)
}

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
	AsOf    string  `json:"as_of"`
	AsOfISO string  `json:"as_of_iso"`
}

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
	var walked []walkedAddr
	if svcs, serr := s.store.ListCurrentServiceSubjects(ctx, db.ListCurrentServiceSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); serr == nil {
		walked = walkedAddresses(svcs)
	} else {
		log.Printf("web: api: coverage: list service subjects: %v", serr)
	}
	// The row's worth is the sentence beside the count, which no JSON field carries (#989).
	meters := apertureMeters(seeds, zones, walked, s.now(), nil)

	out := apiCoverageResponse{Meters: make([]apiCoverageMeter, 0, len(meters))}
	for _, m := range meters {
		out.Meters = append(out.Meters, apiCoverageMeter{
			Label: m.Label, Counted: m.Counted, Total: m.Total, Unit: m.Unit, Pct: m.Pct, Detail: m.Detail,
			AsOf: m.AsOf, AsOfISO: m.AsOfISO,
		})
	}
	writeAPIJSON(w, out)
}
