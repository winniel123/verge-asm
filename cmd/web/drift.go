package main

import (
	"log"
	"net/http"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Drift screen (#283, ADR-0108/ADR-0110) — the canonical `/drift`, nav item 4
// of 7 and the product's thesis: what moved since last time, grouped by batch, in
// change's own language (appeared / revealed / withdrawn / descoped / returned /
// changed) rather than the severity ramp. The composition is ported verbatim from
// design-system/examples/console/Drift.jsx — the change-kind legend, the two-column
// "By batch" transitions timeline beside a "Movement" summary, the per-event diff
// affordance.
//
// The screen is first-class even where its data is thin. No estate-wide,
// batch-grouped transition feed exists in the store yet — a transition is a span
// open/close event, and the web store exposes only per-subject spans
// (ListSpansForSubject) and the current open set (ListAllOpenSpans), never a
// batch-grouped change feed across the estate. So Groups is empty and the timeline
// renders the design-system empty-state; the change vocabulary itself (the legend)
// is definitional, not data, so it always renders. We never fabricate change
// events (the ported example's sample transitions are dropped, not carried). The
// missing plumbing is filed as a follow-on on #283.

// driftFamily is the drift palette a change kind rides — the `.chip` modifier and
// the `--drift-<family>-*` token triple. Change is its own language: never the
// severity ramp. Mirrors ChangeBadge.jsx's FAMILY map exactly.
func driftFamily(change string) string {
	switch change {
	case "appeared", "revealed", "returned":
		return "gain" // violet — a value entered or widened into sight
	case "withdrawn", "descoped":
		return "loss" // slate — a value left sight
	default:
		return "change" // magenta — a held value moved (changed)
	}
}

// driftKind is one entry of the change vocabulary, shaped for the legend row: the
// kind word and the drift family (chip class) it rides.
type driftKind struct {
	Change string
	Family string
}

// driftKinds is the fixed change vocabulary in the example's order. It is the
// language of drift, not a data read — the legend renders it whether or not any
// transition has yet been folded.
func driftKinds() []driftKind {
	kinds := []string{"appeared", "revealed", "withdrawn", "descoped", "returned", "changed"}
	out := make([]driftKind, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, driftKind{Change: k, Family: driftFamily(k)})
	}
	return out
}

// driftDiffLine is one line of a transition's before/after diff — a removal (the
// prior value, danger-red, a true minus), an addition (the new value, ok-green), or
// an unchanged context line. Always mono.
type driftDiffLine struct {
	Type string // "add" | "remove" | "same"
	Text string
}

// driftEvent is one transition in a batch: its change kind (carrying its drift
// family), the subject it moved, a terse detail, a relative time, an optional
// closure/aperture reason, and an optional before/after diff.
type driftEvent struct {
	Change string
	Family string
	Subject string
	Detail string
	Time   string
	Reason string
	Diff   []driftDiffLine
}

// driftBatch is one batch's group of transitions — the unit the timeline groups by.
// Empty until an estate-wide transition feed is plumbed (follow-on on #283).
type driftBatch struct {
	Label  string
	Meta   string
	Events []driftEvent
}

// driftPage renders the Drift screen. It carries the change vocabulary (the legend)
// and the batch-grouped transitions. No transition feed exists yet, so Groups is
// empty and the timeline falls to the empty-state; the handler fabricates no
// change events.
func (s *server) driftPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	var groups []driftBatch // no estate-wide transition feed yet (#283 follow-on)

	// Batch detail entry (#311, T16) — opens the Run detail screen (T2, GET /run/{id};
	// id is a Dispatch id) for the most recent batch. Change and batches are distinct
	// feeds: a batch exists as soon as a scan has been dispatched, well before two
	// batches have folded a transition, so the entry is offered whenever a real
	// dispatch exists and omitted otherwise — never a fabricated id. This mirrors the
	// ported example's `onOpenRun && <Button>Batch detail</Button>`.
	batchID, batchLabel := s.latestBatch(r)

	s.render(w, "drift", map[string]any{
		"Title": "Drift", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":  "drift",
		"Kinds":      driftKinds(),
		"Groups":     groups,
		"BatchID":    batchID,
		"BatchLabel": batchLabel,
	})
}

// latestBatch reads the most recent Dispatch (a batch) so Drift can offer a "Batch
// detail" entry into the Run detail screen. Dispatch is Operational — the read
// records what the system did and never touches the comparison path (ADR-0041). It
// returns a zero id when no scan has been dispatched yet, so the caller offers no
// entry rather than fabricate one; a read error degrades to no entry, never a 500
// on the thesis screen. The list is ordered id DESC, so rows[0] is the latest batch.
func (s *server) latestBatch(r *http.Request) (int64, string) {
	rows, err := s.store.ListDispatchProgress(r.Context(), scansHistoryLimit)
	if err != nil {
		log.Printf("web: drift: latest batch: %v", err)
		return 0, ""
	}
	if len(rows) == 0 {
		return 0, ""
	}
	dv := toDispatchView(rows[0])
	return dv.ID, dv.DispatchedAt
}
