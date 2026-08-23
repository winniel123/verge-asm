package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

// Report scheduling persistence (#290). The Reports screen's "New schedule" wizard
// posts here; the handler parses the wizard fields, files one report_schedule, and
// post-redirect-gets back to /reports so a browser refresh does not re-file the
// schedule. It is reached only through requireAdmin — declaring a recurring export
// is an operator config act, gated like channel and seed declaration — so a viewer
// never reaches it. The row-menu's edit/delete and the on-cadence dispatcher stay
// out of scope (#291); this ticket persists the declaration and lists it.

// reportScheduleSections is the closed set of report sections the wizard offers, in
// canonical order. A submitted section is admitted only if it names one of these, so
// a hand-crafted POST cannot smuggle an arbitrary section key into the stored array.
var reportScheduleSections = []string{"summary-kpis", "new-assets", "signal-changes", "coverage-gaps"}

// reportScheduleCadences / reportScheduleFormats are the closed sets the two selects
// offer. An unrecognised value falls back to the wizard's own default rather than
// being stored verbatim, so the table always renders a known token.
var (
	reportScheduleCadences = map[string]bool{"6h": true, "daily": true, "weekly": true, "monthly": true, "custom": true}
	reportScheduleFormats  = map[string]bool{"pdf": true, "csv": true}
)

const (
	defaultReportCadence = "weekly"
	defaultReportFormat  = "pdf"
)

// createReportSchedule files one recurring report from the wizard form. A schedule
// needs a name; an empty name is refused with a plain redirect back (the input also
// carries the HTML `required` attribute, so the empty case is only reachable by a
// hand-crafted POST). Sections, cadence and format are validated against their closed
// sets so only known values are stored, and the schedule is attributed to the admin
// who declared it.
func (s *server) createReportSchedule(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		// Nothing to declare without a name; return to the screen rather than filing an
		// unnamed schedule. The wizard's `required` name input prevents this in-browser.
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}

	// Admit only the sections the wizard offers, in canonical order, so the stored
	// array is stable regardless of checkbox submission order and carries no unknowns.
	chosen := map[string]bool{}
	for _, v := range r.Form["sections"] {
		chosen[v] = true
	}
	sections := make([]string, 0, len(reportScheduleSections))
	for _, key := range reportScheduleSections {
		if chosen[key] {
			sections = append(sections, key)
		}
	}
	sectionsJSON, err := json.Marshal(sections)
	if err != nil {
		s.serverError(w, "marshal report schedule sections", err)
		return
	}

	cadence := r.FormValue("cadence")
	if !reportScheduleCadences[cadence] {
		cadence = defaultReportCadence
	}
	format := r.FormValue("format")
	if !reportScheduleFormats[format] {
		format = defaultReportFormat
	}
	target := strings.TrimSpace(r.FormValue("target"))

	if _, err := s.store.InsertReportSchedule(r.Context(), db.InsertReportScheduleParams{
		Name:           name,
		Sections:       sectionsJSON,
		Cadence:        cadence,
		Format:         format,
		DeliveryTarget: target,
		CreatedBy:      acct.ID,
	}); err != nil {
		s.serverError(w, "insert report schedule", err)
		return
	}

	http.Redirect(w, r, "/reports", http.StatusSeeOther)
}
