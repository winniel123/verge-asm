package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"net/netip"
	"sort"
	"time"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/exposure"
)

type exposureStore interface {
	ListServiceReachabilitySpansByClass(ctx context.Context) ([]db.ListServiceReachabilitySpansByClassRow, error)
	ListVantages(ctx context.Context) ([]db.ListVantagesRow, error)
}

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/exposure.tmpl"))

type exposureRow struct {
	Asset    string
	Svc      string
	Internal legChip
	Internet legChip
}

// The listing dates a leg to the day; the instant belongs to the detail page (#2034).

const exposureSinceDateFmt = "2006-01-02"

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
	// One binding classifies the vantages, the rows and both delta snapshots (ADR-1945 §1, #2046).
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
		withheld := pageData(acct, "Exposure", "exposure", map[string]any{
			"Withheld": true,
		})
		// A scope over the only internet prober's address withholds the board (ADR-1895 §6).
		s.fillScopeActPanel(ctx, acct, withheld)
		s.render(w, r, "exposure", withheld)
		return
	}

	rows, census, ferr := s.foldExposureUnder(ctx, covered)
	if ferr != nil {
		// An empty board reads as nothing exposed, so this read is loud (ADR-0168 §4, #1424).
		s.serverError(w, "fold exposure", ferr)
		return
	}

	data := pageData(acct, "Exposure", "exposure", map[string]any{
		"Withheld":    false,
		"Rows":        rows,
		"Exposed":     census.Exposed,
		"EdgeOnly":    census.EdgeOnly,
		"Firewalled":  census.Firewalled,
		"Unreachable": census.Unreachable,
		"OneLegged":   census.OneLegged,
	})
	s.fillScopeActPanel(ctx, acct, data)
	if prevAt, ok, err := s.previousBatchInstant(ctx); err != nil {
		log.Printf("web: exposure: previous batch instant: %v", err)
	} else if ok {
		if exposed, dok := s.exposureDeltasFrom(ctx, prevAt, census, covered); dok {
			data["ExposedDelta"] = exposed
			data["HasDeltas"] = true
		}
	}
	s.render(w, r, "exposure", data)
}

type legInfo struct {
	outcome string
	reasons []string
	causes  []string
	since   time.Time
	isGap   bool
	present bool
}

func (s *server) foldExposureUnder(ctx context.Context, covered func(netip.Addr) bool) ([]exposureRow, exposure.Census, error) {
	byClass, err := s.exposureStore.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		return nil, exposure.Census{}, err
	}
	legs := collapseReachLegs(reachRowsFromCurrent(byClass), covered)
	order := make([]string, 0, len(legs))
	for k := range legs {
		order = append(order, k)
	}
	sort.Strings(order)

	var census exposure.Census
	var rows []exposureRow
	for _, svc := range order {
		addr, port, transport := splitServiceKey(svc)
		internalInfo, internetInfo := legs[svc]["internal"], legs[svc]["internet"]
		internal, internet := legFrom(internalInfo), legFrom(internetInfo)

		internalChip := reachLegChip(custody.ClassInternal, internal)
		// A date belongs to the value it sits beside, so each leg carries its own (#2034).
		internalChip.Date = legSinceDate(internalInfo)
		internetChip := reachLegChip(custody.ClassInternet, internet)
		internetChip.Date = legSinceDate(internetInfo)

		rows = append(rows, exposureRow{
			Asset:    addr,
			Svc:      ":" + port + " " + transport,
			Internal: internalChip,
			Internet: internetChip,
		})

		census.Count(internet, internal)
	}
	return rows, census, nil
}

func censusFromLegs(byService map[string]map[string]legInfo) exposure.Census {
	var c exposure.Census
	for _, m := range byService {
		c.Count(legFrom(m["internet"]), legFrom(m["internal"]))
	}
	return c
}

// The change is a difference from the figure the band renders (ADR-1945 §1, #2046).

func (s *server) exposureDeltasFrom(ctx context.Context, prevAt time.Time, cur exposure.Census,
	covered func(netip.Addr) bool) (exposed drift.Delta, ok bool) {
	past, err := s.deltasStore.ListServiceReachabilitySpansByClassAt(ctx, pgtypeTimestamptz(prevAt))
	if err != nil {
		log.Printf("web: exposure delta: list reachability by class at: %v", err)
		return drift.Delta{}, false
	}
	prev := censusFromLegs(collapseReachLegs(reachRowsFromAt(past), covered))
	return drift.Delta{Current: cur.Exposed, Previous: prev.Exposed}, true
}

type legChip struct {
	Tone  string
	Label string
	Date  string
}

func legSince(l legInfo) string { return legSinceIn(l, spanTimeFmt) }

func legSinceDate(l legInfo) string { return legSinceIn(l, exposureSinceDateFmt) }

func legSinceIn(l legInfo, layout string) string {
	// A never-configured leg holds no value, so there is nothing for a date to belong to (#2017).
	if legFrom(l).Status == exposure.LegNeverConfigured || l.since.IsZero() {
		return ""
	}
	return l.since.UTC().Format(layout)
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
