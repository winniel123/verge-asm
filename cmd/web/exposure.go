package main

import (
	"html/template"
	"log"
	"net/http"
	"sort"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/exposure"
)

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/exposure.tmpl"))

type exposureRow struct {
	Asset    string
	Svc      string
	Internal string
	Internet string
	Since    string
}

type exposureStats struct {
	exposed    int
	firewalled int
	notReached int
}

func (s *server) exposurePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	if s.devMode {
		s.render(w, r, "exposure", s.exposureFixtureData(acct, r.URL.Query().Get("variant")))
		return
	}

	// With no internet leg no exposure is constructible, so the board is WITHHELD (v1-spec §6.2).
	vantages, err := s.store.ListVantages(ctx)
	if err != nil {
		s.serverError(w, "list vantages", err)
		return
	}
	covered, err := s.addressScopeCovered(ctx)
	if err != nil {
		s.serverError(w, "address scope coverage", err)
		return
	}
	internetVantage := false
	for _, v := range vantages {
		if vantageFactsClass(v.DialledAddr, v.Egress, covered).IsInternet() {
			internetVantage = true
			break
		}
	}
	if !internetVantage {
		s.render(w, r, "exposure", map[string]any{
			"Title": "Exposure", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"NavActive": "exposure",
			"Withheld":  true,
		})
		return
	}

	rows, stats := s.foldExposure(r)

	data := map[string]any{
		"Title": "Exposure", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":  "exposure",
		"Withheld":   false,
		"Rows":       rows,
		"Exposed":    stats.exposed,
		"Firewalled": stats.firewalled,
		"NotReached": stats.notReached,
	}
	if prevAt, ok, err := s.previousBatchInstant(ctx); err != nil {
		log.Printf("web: exposure: previous batch instant: %v", err)
	} else if ok {
		if exposed, firewalled, notReached, dok := s.exposureCountDeltas(ctx, prevAt); dok {
			data["ExposedDelta"] = exposed
			data["FirewalledDelta"] = firewalled
			data["NotReachedDelta"] = notReached
			data["HasDeltas"] = true
		}
	}
	s.render(w, r, "exposure", data)
}

type legInfo struct {
	outcome string
	isGap   bool
	present bool
}

func (s *server) foldExposure(r *http.Request) ([]exposureRow, exposureStats) {
	ctx := r.Context()
	byClass, err := s.store.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		return nil, exposureStats{}
	}
	covered, err := s.addressScopeCovered(ctx)
	if err != nil {
		return nil, exposureStats{}
	}
	legs := collapseReachLegs(reachRowsFromCurrent(byClass), covered)
	order := make([]string, 0, len(legs))
	for k := range legs {
		order = append(order, k)
	}
	sort.Strings(order)

	since := map[string]string{}
	if spans, err := s.store.ListAllOpenSpans(ctx); err == nil {
		for _, sp := range spans {
			if sp.SubjectKind != "service" || sp.Facet != "reachability" || !sp.OpenedAt.Valid {
				continue
			}
			d := sp.OpenedAt.Time.UTC().Format("2006-01-02")
			if cur, ok := since[sp.SubjectKey]; !ok || d < cur {
				since[sp.SubjectKey] = d
			}
		}
	}

	var rows []exposureRow
	var stats exposureStats
	for _, svc := range order {
		addr, port, transport := splitServiceKey(svc)
		internal := legs[svc]["internal"]
		internet := legs[svc]["internet"]

		sinceStr := since[svc]
		if sinceStr == "" {
			sinceStr = "—"
		}
		rows = append(rows, exposureRow{
			Asset:    addr,
			Svc:      ":" + port + " " + transport,
			Internal: legDisplay(internal),
			Internet: legDisplay(internet),
			Since:    sinceStr,
		})

		ev, ok := exposure.Project(legFrom(internet), legFrom(internal))
		switch {
		case !ok:
			stats.notReached++
		case ev == exposure.Exposed:
			stats.exposed++
		case ev == exposure.Firewalled:
			stats.firewalled++
		}
	}
	return rows, stats
}

func legDisplay(l legInfo) string {
	if !l.present {
		return "unverified"
	}
	return assetExposure(l.outcome, l.isGap)
}

func legFrom(l legInfo) exposure.Leg {
	if !l.present {
		return exposure.Leg{Status: exposure.LegNeverConfigured}
	}
	if l.isGap {
		return exposure.Leg{Status: exposure.LegGap}
	}
	switch l.outcome {
	case string(exposure.Reached):
		return exposure.Leg{Status: exposure.LegValued, Value: exposure.Reached}
	case string(exposure.NotReached):
		return exposure.Leg{Status: exposure.LegValued, Value: exposure.NotReached}
	default:
		return exposure.Leg{Status: exposure.LegNeverConfigured}
	}
}
