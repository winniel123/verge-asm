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

// The estate-wide drift feed (#288, ADR-0111). A transition is not stored — it is
// the adjacency between two consecutive spans, derived on read (ADR-0007) — so this
// file turns the store's raw span open/close events (each citing the Batch that
// caused it) into the batch-grouped, classified feed the /drift screen renders: the
// "By batch" transition timeline, the per-kind Movement summary, and the CSV export.
//
// The six change kinds follow #288's own definitions, mapped onto the drift grammar
// (internal/drift):
//
//   - appeared  — the first span on a (subject, facet) timeline: an opening with no
//     predecessor.
//   - changed   — a held value moved within an open timeline: an opening whose
//     predecessor sits under the same Derivation vector and carried no closure
//     reason (an ordinary value move, drift.KindNone), rendered with a before/after
//     diff of the two spans' values.
//   - returned  — a re-open after an absence: an opening whose predecessor was a
//     withdrawal closure (a closure reason is set). Dormant until the
//     withdrawal-persistence path is wired (no production path closes a span with a
//     reason today), but classified end-to-end for when it lands.
//   - withdrawn — a close by withdrawal (closure reason measured-absent / uncited).
//   - descoped  — a close by operator exclusion (closure reason descoped).
//   - revealed  — a widened aperture. It needs an aperture-widened signal the span
//     corpus does not carry (drift.OpeningKind), so it is never emitted here; the
//     legend still names it as vocabulary. Honest rather than guessed.
//
// A transition across a Break (the predecessor sits under a different Derivation
// vector) is not a value move — nothing compares across a Break (ADR-0008) — so it
// is skipped, never narrated as `changed`.

// driftMovement is the per-kind tally the Movement summary renders. Keyed by the
// change word so the template looks each kind up by name.
type driftMovement map[string]int

// buildDriftFeed folds the store's raw span events into the batch-grouped, classified
// timeline and the per-kind movement tally. rows arrive newest-batch-first, already
// ordered by timeline within a batch (ListRecentDriftEvents), so a single pass groups
// them: a run of consecutive rows sharing a batch id is one group.
func buildDriftFeed(rows []db.ListRecentDriftEventsRow, now time.Time) ([]driftBatch, driftMovement) {
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

// classifyDriftEvent turns one raw span event into a rendered transition, or reports
// ok=false where the event is not a narratable transition (an opening across a Break,
// or a revealed opening the corpus cannot witness).
func classifyDriftEvent(row db.ListRecentDriftEventsRow, now time.Time) (driftEvent, bool) {
	facetLabel := timelineLabel(row.Facet, row.Discriminator)

	if row.Role == "closed" {
		// A reasoned close — withdrawn or descoped. The closing span's value is the one
		// that left; there is no after side, so no diff. The instant is the close.
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

	// An opening. A predecessor exists iff prev_value is non-nil (the span.value column
	// is NOT NULL, so a real predecessor always carries bytes; nil means the LATERAL
	// found none — a first span).
	if row.PrevValue == nil {
		// appeared — the first span on this timeline.
		return driftEvent{
			Change:  "appeared",
			Family:  driftFamily("appeared"),
			Subject: row.SubjectKey,
			Detail:  facetLabel + " · " + valueLabel(row.Facet, row.Value, row.IsGap),
			Time:    relTime(row.OpenedAt.Time, now),
		}, true
	}

	// A predecessor exists. Nothing compares across a Break (ADR-0008): if the vectors
	// differ this opening is a version bump, not a value move, so it is not narrated.
	if !decodeVector(row.Derivation).Equal(decodeVector(row.PrevDerivation)) {
		return driftEvent{}, false
	}

	prevReason := drift.ClosureReason(row.PrevClosureReason.String)
	if prevReason.Valid() {
		// The predecessor was a withdrawal closure, so this opening is a re-entry across
		// an absence. drift.MembershipReturn decides returned vs appeared — a `descoped`
		// prior closure reads `appeared` (a narrowing is not a decommission), the rest
		// read `returned`. witnessBroke is folded into the vector-equality guard above.
		kind := drift.MembershipReturn(&drift.Span{Reason: prevReason}, false)
		change := "returned"
		if kind == drift.KindAppeared {
			change = "appeared"
		}
		return driftEvent{
			Change:  change,
			Family:  driftFamily(change),
			Subject: row.SubjectKey,
			Detail:  facetLabel + " · " + valueLabel(row.Facet, row.Value, row.IsGap),
			Time:    relTime(row.OpenedAt.Time, now),
		}, true
	}

	// An ordinary value move — `changed`, with a before/after diff of the two values.
	return driftEvent{
		Change:  "changed",
		Family:  driftFamily("changed"),
		Subject: row.SubjectKey,
		Detail:  facetLabel,
		Time:    relTime(row.OpenedAt.Time, now),
		Diff:    driftDiff(row),
	}, true
}

// driftDiff renders a changed transition's before/after as a two-line diff: the prior
// value removed, the new value added. Both sides render through the shared valueLabel,
// so the diff reads in the same vocabulary as the Inventory and drill-down timelines.
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

// spanValueIsGap recomputes whether a value is a Gap from the value alone, mirroring
// the fold's isGapValue (internal/queue/spanfold.go) via the same leaf constants so
// the two definitions cannot silently diverge: a resolution OutcomeGap or a
// reachability GapOutcome. Used for a predecessor value, whose is_gap flag the feed
// query does not carry (a LEFT JOIN LATERAL column sqlc cannot type as nullable).
func spanValueIsGap(facet string, raw []byte) bool {
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

// writeDriftExportCSV emits the transition feed as one uniform table — one row per
// transition, in the same newest-first order the screen renders. Times are absolute
// UTC (not the screen's relative "3h ago"), so a directory of exports is unambiguous.
// The same classification the screen uses decides each row, so the file and the
// screen never disagree; a Break-crossing opening or a revealed opening is skipped in
// both.
func (s *server) writeDriftExportCSV(w http.ResponseWriter, period driftPeriod, rows []db.ListRecentDriftEventsRow) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="drift-`+period.Token+`.csv"`)

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

	// The feed is capped (driftFeedLimit); a full result set is stated in a trailing
	// marker row rather than dropping the older tail silently.
	if int32(len(rows)) >= driftFeedLimit {
		_ = cw.Write([]string{"feed capped at " + strconv.Itoa(int(driftFeedLimit)) + " most-recent events; older transitions omitted", "", "", "", "", "", "", "", ""})
	}
}

// csvSafe neutralises spreadsheet formula injection: a cell whose first character is
// one an interpreter may treat as a formula (= + - @) or a control character (tab,
// CR) is prefixed with a single quote, so a subject or measured value ingested from CT
// logs — attacker-influenced free text — cannot execute when the file is opened in
// Excel or Sheets. The prefix is the widely-used mitigation; it leaves the value
// legible and is applied only to the free-text columns.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// driftExportValues resolves the before/after value cells for an export row from the
// change kind: a `changed` carries both sides, an entry (appeared / returned) carries
// only the after value, and an exit (withdrawn / descoped) carries only the before.
func driftExportValues(row db.ListRecentDriftEventsRow, change string) (before, after string) {
	switch change {
	case "changed":
		return valueLabel(row.Facet, row.PrevValue, spanValueIsGap(row.Facet, row.PrevValue)),
			valueLabel(row.Facet, row.Value, row.IsGap)
	case "withdrawn", "descoped":
		return valueLabel(row.Facet, row.Value, row.IsGap), ""
	default: // appeared, returned
		return "", valueLabel(row.Facet, row.Value, row.IsGap)
	}
}

// driftReasonLabel renders a closure ground for the event's " · reason" line.
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

// driftBatchLabel is the group header — the batch kind and how long ago it folded.
func driftBatchLabel(row db.ListRecentDriftEventsRow, now time.Time) string {
	kind := strings.TrimSpace(row.BatchKind)
	if kind == "" {
		kind = "scan"
	}
	return kind + " scan · " + agoLabel(row.BatchAt.Time, now)
}

// agoLabel renders a relative instant as a phrase. relTime returns bare tokens
// (now / 5m / 3h / 2d / 1w); this suffixes " ago" for the past and reads the
// sub-minute case as "just now" rather than the ungrammatical "now ago".
func agoLabel(t, now time.Time) string {
	rel := relTime(t, now)
	if rel == "now" {
		return "just now"
	}
	return rel + " ago"
}

// driftBatchMeta summarises the batch's recorded scope for the group sub-label. The
// scope is recorded by content (ADR-0025) as a JSON object of name/address lists; the
// meta states how much was in scope rather than a library default.
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
