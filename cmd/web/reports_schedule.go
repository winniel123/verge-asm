package main

import (
	"net/http"

	"github.com/winniel123/verge-asm/internal/db"
)

// Report scheduling has no dispatch or delivery backend (#344). A schedule row that
// nothing reads on a cadence and nothing delivers is a declaration the product cannot
// honour, so the Reports UI must not accept one: the "New schedule" wizard is rendered
// disabled alongside its already-disabled sibling controls (Run now / Edit / Delete /
// View last delivery), and this create handler — the last path that could persist a
// row — refuses.
//
// The route stays registered (behind requireAdmin) rather than removed so a
// hand-crafted POST meets a clear, deterministic refusal instead of a bare 405: no
// report_schedule row is ever filed from normal use OR from a crafted request. When the
// on-cadence dispatcher + delivery land (the real "wire report scheduling" feature,
// explicitly out of scope here), this handler regains its wizard and its InsertReport-
// Schedule call; until then it honestly reports the capability as unavailable.
//
// This closes only the defect where the UI over-promised. It builds no dispatcher,
// adds no DB query, and leaves /reports/export (csv/json) and the rest of Reports
// untouched.
func (s *server) createReportSchedule(w http.ResponseWriter, r *http.Request, acct db.Account) {
	http.Error(w, "report scheduling is not available yet", http.StatusNotImplemented)
}
