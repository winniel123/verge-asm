package main

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

type driftMovement map[string]int

func buildDriftFeed(rows []db.ListRecentDriftEventsRow, now time.Time) ([]driftBatch, driftMovement) {
	// The query already orders by timeline within a batch, so one pass groups the runs.
	movement := driftMovement{}
	var groups []driftBatch
	var cur *driftBatch
	var curID int64

	for _, row := range rows {
		ev, ok := classifyDriftEvent(row, now)
		if !ok {
			continue
		}
		movement[ev.Change]++
		if cur == nil || row.BatchID != curID {
			groups = append(groups, driftBatch{
				Label: driftBatchLabel(row, now),
				Meta:  driftBatchMeta(row),
			})
			cur = &groups[len(groups)-1]
			curID = row.BatchID
		}
		cur.Events = append(cur.Events, ev)
	}
	return groups, movement
}

func classifyDriftEvent(row db.ListRecentDriftEventsRow, now time.Time) (driftEvent, bool) {
	facetLabel := timelineLabel(row.Facet, row.Discriminator)

	if row.Role == "closed" {
		change := "withdrawn"
		if drift.ClosureReason(row.ClosureReason.String) == drift.ReasonDescoped {
			change = "descoped"
		}
		return driftEvent{
			Change:  change,
			Family:  driftFamily(change),
			Subject: row.SubjectKey,
			Detail:  facetLabel + " · " + valueLabel(row.Facet, row.Value, row.IsGap),
			Time:    relTime(row.ClosedAt.Time, now),
			Reason:  driftReasonLabel(drift.ClosureReason(row.ClosureReason.String)),
		}, true
	}

	// The value column is NOT NULL, so a nil predecessor means the LATERAL found none.
	if row.PrevValue == nil {
		change := "appeared"
		if row.OpenedAperture {
			change = "revealed"
		}
		return driftEvent{
			Change:  change,
			Family:  driftFamily(change),
			Subject: row.SubjectKey,
			Detail:  facetLabel + " · " + valueLabel(row.Facet, row.Value, row.IsGap),
			Time:    relTime(row.OpenedAt.Time, now),
		}, true
	}

	if !decodeVector(row.Derivation).Equal(decodeVector(row.PrevDerivation)) {
		return driftEvent{}, false
	}

	prevReason := drift.ClosureReason(row.PrevClosureReason.String)
	if prevReason.Valid() {
		change := "returned"
		switch drift.ReEntryKind(&drift.Span{Reason: prevReason}, false, row.OpenedAperture) {
		case drift.KindAppeared:
			change = "appeared"
		case drift.KindRevealed:
			change = "revealed"
		}
		return driftEvent{
			Change:  change,
			Family:  driftFamily(change),
			Subject: row.SubjectKey,
			Detail:  facetLabel + " · " + valueLabel(row.Facet, row.Value, row.IsGap),
			Time:    relTime(row.OpenedAt.Time, now),
		}, true
	}

	return driftEvent{
		Change:  "changed",
		Family:  driftFamily("changed"),
		Subject: row.SubjectKey,
		Detail:  facetLabel,
		Time:    relTime(row.OpenedAt.Time, now),
		Diff:    driftDiff(row),
	}, true
}

func driftDiff(row db.ListRecentDriftEventsRow) []driftDiffLine {
	before := valueLabel(row.Facet, row.PrevValue, spanValueIsGap(row.Facet, row.PrevValue))
	after := valueLabel(row.Facet, row.Value, row.IsGap)
	if before == after {
		return nil
	}
	return []driftDiffLine{
		{Type: "remove", Text: before},
		{Type: "add", Text: after},
	}
}

func spanValueIsGap(facet string, raw []byte) bool {
	// A LEFT JOIN LATERAL column cannot be typed nullable, so a predecessor's flag is recomputed.
	// The leaf constants are shared with the fold's isGapValue so the two cannot silently diverge.
	if len(raw) == 0 {
		return false
	}
	var v struct {
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	switch facet {
	case resolutionwalk.FacetResolution:
		return v.Outcome == string(resolutionwalk.OutcomeGap)
	case connectoutcome.FacetReachability:
		return v.Outcome == connectoutcome.GapOutcome
	default:
		return false
	}
}

func (s *server) writeDriftExportCSV(w http.ResponseWriter, periodToken string, rows []db.ListRecentDriftEventsRow) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="drift-`+periodToken+`.csv"`)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"batch", "scope", "change", "subject", "detail", "time", "reason", "before", "after"})

	now := s.now()
	for _, row := range rows {
		ev, ok := classifyDriftEvent(row, now)
		if !ok {
			continue
		}
		before, after := driftExportValues(row, ev.Change)
		when := row.OpenedAt.Time
		if row.Role == "closed" {
			when = row.ClosedAt.Time
		}
		_ = cw.Write([]string{
			csvSafe(driftBatchLabel(row, now)),
			csvSafe(driftBatchMeta(row)),
			ev.Change,
			csvSafe(ev.Subject),
			csvSafe(timelineLabel(row.Facet, row.Discriminator)),
			when.UTC().Format(time.RFC3339),
			ev.Reason,
			csvSafe(before),
			csvSafe(after),
		})
	}

	if int32(len(rows)) >= driftFeedLimit { // #nosec G115 (len(rows) under driftFeedLimit=500-row cap)
		_ = cw.Write([]string{"feed capped at " + strconv.Itoa(int(driftFeedLimit)) + " most-recent events; older transitions omitted", "", "", "", "", "", "", "", ""})
	}
}

func csvSafe(s string) string {
	// CT-log text is attacker-influenced, and a leading = + - @ executes when a sheet opens it.
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

func driftExportValues(row db.ListRecentDriftEventsRow, change string) (before, after string) {
	switch change {
	case "changed":
		return valueLabel(row.Facet, row.PrevValue, spanValueIsGap(row.Facet, row.PrevValue)),
			valueLabel(row.Facet, row.Value, row.IsGap)
	case "withdrawn", "descoped":
		return valueLabel(row.Facet, row.Value, row.IsGap), ""
	default:
		return "", valueLabel(row.Facet, row.Value, row.IsGap)
	}
}

func driftReasonLabel(r drift.ClosureReason) string {
	switch r {
	case drift.ReasonMeasuredAbsent:
		return "measured absent"
	case drift.ReasonUncited:
		return "no longer cited"
	case drift.ReasonDescoped:
		return "removed from scope"
	default:
		return ""
	}
}

func driftBatchLabel(row db.ListRecentDriftEventsRow, now time.Time) string {
	kind := strings.TrimSpace(row.BatchKind)
	if kind == "" {
		kind = "scan"
	}
	return kind + " scan · " + agoLabel(row.BatchAt.Time, now)
}

func agoLabel(t, now time.Time) string {
	rel := relTime(t, now)
	if rel == "now" {
		return "just now"
	}
	return rel + " ago"
}

func driftBatchMeta(row db.ListRecentDriftEventsRow) string {
	var scope map[string]json.RawMessage
	if err := json.Unmarshal(row.RecordedScope, &scope); err != nil || len(scope) == 0 {
		return ""
	}
	for _, key := range []string{"names", "addresses", "services"} {
		raw, ok := scope[key]
		if !ok {
			continue
		}
		var list []json.RawMessage
		if json.Unmarshal(raw, &list) == nil && len(list) > 0 {
			noun := key
			if len(list) == 1 {
				noun = strings.TrimSuffix(key, "s")
			}
			return strconv.Itoa(len(list)) + " " + noun
		}
	}
	return ""
}
