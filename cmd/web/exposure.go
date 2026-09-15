package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"sort"
	"time"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/exposure"
)

type exposureStore interface {
	ListAllOpenSpans(ctx context.Context) ([]db.ListAllOpenSpansRow, error)
	ListServiceReachabilitySpansByClass(ctx context.Context) ([]db.ListServiceReachabilitySpansByClassRow, error)
	ListVantages(ctx context.Context) ([]db.ListVantagesRow, error)
}

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/exposure.tmpl"))

type exposureRow struct {
	Asset    string
	Svc      string
	Internal legChip
	Internet legChip
	Since    string
}

type exposureStats struct {
	exposed      int
	firewalled   int
	notReached   int
	sinceUnknown bool
}

const (
	exposureSinceAbsent  = "—"
	exposureSinceUnknown = "?"
)

func (s *server) exposurePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	if s.devMode {
		s.render(w, r, "exposure", s.exposureFixtureData(acct, r.URL.Query().Get("variant")))
		return
	}

	// With no internet leg no exposure is constructible, so the board is WITHHELD (v1-spec §6.2).
	vantages, err := s.exposureStore.ListVantages(ctx)
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
		s.render(w, r, "exposure", pageData(acct, "Exposure", "exposure", map[string]any{
			"Withheld": true,
		}))
		return
	}

	rows, stats, ferr := s.foldExposure(r)
	if ferr != nil {
		// An empty board reads as nothing exposed, so this read is loud (ADR-0168 §4, #1424).
		s.serverError(w, "fold exposure", ferr)
		return
	}

	data := pageData(acct, "Exposure", "exposure", map[string]any{
		"Withheld":     false,
		"Rows":         rows,
		"Exposed":      stats.exposed,
		"Firewalled":   stats.firewalled,
		"NotReached":   stats.notReached,
		"SinceUnknown": stats.sinceUnknown,
	})
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
	// An existential composition names no vantage, so a gapped leg carries every cause (ADR-0080).
	reasons []string
	causes  []string
	since   time.Time
	isGap   bool
	present bool
}

func (s *server) foldExposure(r *http.Request) ([]exposureRow, exposureStats, error) {
	ctx := r.Context()
	byClass, err := s.exposureStore.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		return nil, exposureStats{}, err
	}
	covered, err := s.addressScopeCovered(ctx)
	if err != nil {
		return nil, exposureStats{}, err
	}
	legs := collapseReachLegs(reachRowsFromCurrent(byClass), covered)
	order := make([]string, 0, len(legs))
	for k := range legs {
		order = append(order, k)
	}
	sort.Strings(order)

	since := map[string]string{}
	var stats exposureStats
	spans, err := s.exposureStore.ListAllOpenSpans(ctx)
	if err != nil {
		// The counts do not read this input, so its failure degrades one column (#1947).
		log.Printf("web: exposure: list all open spans: %v", err)
		stats.sinceUnknown = true
	}
	for _, sp := range spans {
		if sp.SubjectKind != "service" || sp.Facet != "reachability" || !sp.OpenedAt.Valid {
			continue
		}
		d := sp.OpenedAt.Time.UTC().Format("2006-01-02")
		if cur, ok := since[sp.SubjectKey]; !ok || d < cur {
			since[sp.SubjectKey] = d
		}
	}

	var rows []exposureRow
	for _, svc := range order {
		addr, port, transport := splitServiceKey(svc)
		internal := legFrom(legs[svc]["internal"])
		internet := legFrom(legs[svc]["internet"])

		rows = append(rows, exposureRow{
			Asset:    addr,
			Svc:      ":" + port + " " + transport,
			Internal: reachLegChip(custody.ClassInternal, internal),
			Internet: reachLegChip(custody.ClassInternet, internet),
			Since:    sinceDisplay(since[svc], stats.sinceUnknown),
		})

		ev, ok := exposure.Project(internet, internal)
		switch {
		case !ok:
			stats.notReached++
		case ev == exposure.Exposed:
			stats.exposed++
		case ev == exposure.Firewalled:
			stats.firewalled++
		}
	}
	return rows, stats, nil
}

func sinceDisplay(since string, unknown bool) string {
	switch {
	case unknown:
		return exposureSinceUnknown
	case since == "":
		return exposureSinceAbsent
	}
	return since
}

type legChip struct {
	Tone  string
	Label string
	Date  string
}

func legSince(l legInfo) string {
	// A never-configured leg holds no value, so there is nothing for a date to belong to (#2017).
	if legFrom(l).Status == exposure.LegNeverConfigured || l.since.IsZero() {
		return ""
	}
	return l.since.UTC().Format(spanTimeFmt)
}

func reachLegChip(class custody.VantageClass, l exposure.Leg) legChip {
	switch l.Status {
	case exposure.LegValued:
		if l.Value == exposure.Reached {
			if class.IsInternet() {
				// Only the internet leg's reached is the move the product alerts on (ADR-0029).
				return legChip{Tone: "danger", Label: "reached"}
			}
			return legChip{Tone: "neutral", Label: "reached"}
		}
		return legChip{Tone: "neutral", Label: "not reached"}
	case exposure.LegGap:
		// The two absences keep their two statements (ADR-0017 decision 4).
		return legChip{Tone: "warn", Label: "stopped looking"}
	default:
		return legChip{Tone: "absent", Label: "never looked"}
	}
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
