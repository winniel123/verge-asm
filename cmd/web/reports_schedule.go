package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/report"
)

// Report scheduling is live (#290, P0.6/T4), ported from
// design-system/examples/console/Reports.jsx: the "New schedule" wizard declares a
// recurring report and the "Recurring reports" row menu edits, deletes, and runs one
// now. A report_schedule is Declared and carries no timeline — a re-declaration
// through the wizard is a fresh insert, and Edit is a genuine in-place update of what
// was declared, never a recompute (migration 21700). It never touches the comparison
// path and never becomes a Message.
//
// The example's Wizard is a client-side modal with four steps (Scope / Cadence /
// Delivery / Review). The app is server-rendered with no client runtime, so — exactly
// as the onboarding wizard (#307) — the controlled React state becomes a post-back
// form: the accumulated values ride hidden fields, Back/Next re-render the step, and
// the per-step valid gate decides whether Next advances. The markup lives in
// templates_reports.go (the "schedulewizard" template); those components are
// translated to template-local CSS within the existing token vocabulary (restyling,
// not authoring — ADR-0109).
//
// Delivery binds the schedule to a Channel (P0.6c/T7, #508, collision #17 ruled): the
// Destination select offers "Download only" (the default) plus every declared Channel,
// and the chosen channel_id is what the schedule stores (NULL = download-only). The
// bound Channel receives a LINK-ONLY ready-message when a run is cut — the report name,
// its period, and a session-authed link to the in-instance artifact — never the estate
// (ADR-0039 stands). The free-text delivery_target is superseded by the binding: it is
// written empty and no longer read as the destination.

// reportScheduleSection is one selectable report section — the key persisted in the
// sections JSON array and the label the wizard checkbox and the Review list render.
// Ported from Reports.jsx SECTIONS.
type reportScheduleSection struct {
	Key   string
	Label string
}

// The section keys follow the design's fixture vocabulary (fixtures.json →
// reports.wizard.sections): kpis / new-assets / signal-changes / coverage-gaps. The
// wizard checkbox value and the persisted sections JSON both use these keys.
var reportScheduleSections = []reportScheduleSection{
	{"kpis", "Summary KPIs"},
	{"new-assets", "New assets"},
	{"signal-changes", "Signal changes"},
	{"coverage-gaps", "Coverage gaps"},
}

// reportScheduleDefaultSections is the wizard's initial section selection —
// Reports.jsx defaults to the first three (SECTIONS.slice(0, 3)).
func reportScheduleDefaultSections() []string {
	return []string{
		reportScheduleSections[0].Key,
		reportScheduleSections[1].Key,
		reportScheduleSections[2].Key,
	}
}

// reportCadPresets are the CadenceSelect presets, ported verbatim from
// design-system/components/forms/CadenceSelect.jsx. The default is "Weekly · mon
// 09:00" (Reports.jsx's initial cad); "Custom…" reveals a cron field and gates Next
// until it is filled.
var reportCadPresets = []string{"Every 6h", "Daily · 08:00", "Weekly · mon 09:00", "Monthly · 1st", "Custom…"}

const (
	reportDefaultCad = "Weekly · mon 09:00"
	reportCustomCad  = "Custom…"
	// reportScheduleFormat is the delivered form. The Reports.jsx wizard shows the
	// format as a fixed "pdf" in the Review list (there is no format control), so a
	// declared schedule is always a pdf until a format control is added.
	reportScheduleFormat = "pdf"
)

// reportScheduleStepTitles names the four wizard steps in order; reportScheduleLast
// is the index of the Review step, where the flow finishes rather than advances.
var reportScheduleStepTitles = []string{"Scope", "Cadence", "Delivery", "Review"}

const (
	// reportScheduleDeliveryStep is the index of the Delivery step, where the channel
	// destination is chosen (Reports.jsx's "delivery" step between Cadence and Review).
	reportScheduleDeliveryStep = 2
	reportScheduleLast         = 3
)

// scheduleWizardView is the controlled state of the wizard across the post-back
// flow: the step being shown, the schedule id (0 on create, the target on edit), and
// every field's current value. Sections are the checked section keys in canonical
// order.
type scheduleWizardView struct {
	Step     int
	ID       int64
	Name     string
	Sections []string
	Cad      string
	Cron     string
	// ChannelID is the chosen delivery destination: 0 is "Download only" (a NULL
	// channel_id — the run generates in-instance and no ready-message leaves), and any
	// other value is a declared Channel's id (P0.6c/T7).
	ChannelID int64
}

// readScheduleWizardView reconstructs the controlled state from the request form.
// Sections are read from the multi-valued "sections" field and canonicalised (a known
// key kept once, in declared order); an unknown key is dropped. Defaults match the
// example's initial state (weekly cadence).
func readScheduleWizardView(r *http.Request) scheduleWizardView {
	_ = r.ParseForm()

	step := 0
	if n, err := strconv.Atoi(r.FormValue("step")); err == nil {
		step = n
	}
	if step < 0 {
		step = 0
	}
	if step > reportScheduleLast {
		step = reportScheduleLast
	}

	var id int64
	if n, err := strconv.ParseInt(r.FormValue("id"), 10, 64); err == nil {
		id = n
	}

	cad := r.FormValue("cad")
	if cad == "" {
		cad = reportDefaultCad
	}

	var channelID int64
	if n, err := strconv.ParseInt(r.FormValue("channel"), 10, 64); err == nil {
		channelID = n
	}

	return scheduleWizardView{
		Step:      step,
		ID:        id,
		Name:      strings.TrimSpace(r.FormValue("name")),
		Sections:  canonicalSections(r.Form["sections"]),
		Cad:       cad,
		Cron:      strings.TrimSpace(r.FormValue("cron")),
		ChannelID: channelID,
	}
}

// canonicalSections filters a set of submitted section keys to the known set,
// preserving the declared order and dropping duplicates and unknowns — so the stored
// array and the checkbox render are stable regardless of form order.
func canonicalSections(selected []string) []string {
	set := make(map[string]bool, len(selected))
	for _, k := range selected {
		set[k] = true
	}
	out := make([]string, 0, len(reportScheduleSections))
	for _, sec := range reportScheduleSections {
		if set[sec.Key] {
			out = append(out, sec.Key)
		}
	}
	return out
}

// scheduleStepValid is the per-step valid gate, ported from the example's step
// `valid` predicates: Scope needs a name and at least one section; Cadence needs a
// cron when the custom preset is chosen; Review is always valid.
func scheduleStepValid(v scheduleWizardView) bool {
	switch v.Step {
	case 0:
		return strings.TrimSpace(v.Name) != "" && len(v.Sections) > 0
	case 1:
		return v.Cad != reportCustomCad || strings.TrimSpace(v.Cron) != ""
	default:
		return true
	}
}

// scheduleAllValid is the whole-form gate the finish path checks before persisting —
// every step's predicate at once, so a hand-crafted finish POST that skipped a step
// cannot file an incomplete schedule.
func scheduleAllValid(v scheduleWizardView) bool {
	return strings.TrimSpace(v.Name) != "" && len(v.Sections) > 0 &&
		(v.Cad != reportCustomCad || strings.TrimSpace(v.Cron) != "")
}

// reportCadLabel renders the stored cadence label, ported from Reports.jsx's
// `cadLabel`: a custom cadence stores its cron (or "custom" when blank), otherwise
// the lower-cased preset ("weekly · mon 09:00").
func reportCadLabel(cad, cron string) string {
	if cad == reportCustomCad {
		if c := strings.TrimSpace(cron); c != "" {
			return c
		}
		return "custom"
	}
	return strings.ToLower(cad)
}

// reportCadPresetFor maps a stored cadence label back to the wizard's Cad/Cron pair
// for the Edit prefill: a label equal to a preset's lower-cased form selects that
// preset, otherwise it is a custom cadence carrying the label as its cron.
func reportCadPresetFor(cadence string) (cad, cron string) {
	for _, p := range reportCadPresets {
		if p != reportCustomCad && strings.ToLower(p) == cadence {
			return p, ""
		}
	}
	return reportCustomCad, cadence
}

// reportsNewWizardPath / reportsEditWizardPath name the wizard's PRG routes (#23f): each
// step POST 303-redirects to a GET at these paths carrying the accumulated values, so the
// flow is bookmarkable and harness-addressable (the wizard goldens hit the GET URLs).
const reportsNewWizardPath = "/reports/schedule/new"

func reportsEditWizardPath(id int64) string {
	return "/reports/schedule/" + strconv.FormatInt(id, 10) + "/edit"
}

// redirectWizardStep 303-redirects the wizard to a GET at base carrying the accumulated
// controlled state as query parameters (#23f) — the post-back PRG shape. The GET handler
// reconstructs the same view and renders the step.
func redirectWizardStep(w http.ResponseWriter, r *http.Request, base string, v scheduleWizardView) {
	q := url.Values{}
	q.Set("step", strconv.Itoa(v.Step))
	q.Set("name", v.Name)
	for _, sec := range v.Sections {
		q.Add("sections", sec)
	}
	q.Set("cad", v.Cad)
	if v.Cron != "" {
		q.Set("cron", v.Cron)
	}
	q.Set("channel", strconv.FormatInt(v.ChannelID, 10))
	http.Redirect(w, r, base+"?"+q.Encode(), http.StatusSeeOther)
}

// newReportScheduleWizard renders the "New schedule" wizard. A fresh GET (no ?step)
// opens at the first step with the example's defaults; a PRG GET (?step=N&…, the
// post-back redirect target #23f) reconstructs the accumulated state and renders that
// step. In a VERGE_DEV build it serves the pinned fixtures.json wizard slice so the
// seeded instance renders byte-for-byte what the golden composes. requireAdmin —
// declaring a schedule is an admin config act.
func (s *server) newReportScheduleWizard(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "schedulewizard", s.reportsWizardFixtureData(r, acct))
		return
	}
	var v scheduleWizardView
	if r.URL.Query().Get("step") == "" {
		v = scheduleWizardView{Sections: reportScheduleDefaultSections(), Cad: reportDefaultCad}
	} else {
		v = readScheduleWizardView(r)
	}
	s.renderScheduleWizard(r.Context(), w, r, acct, v, false)
}

// editReportScheduleWizard renders the wizard prefilled from an existing schedule. A
// fresh GET (no ?step) prefills from the stored row; a PRG GET (?step=N&…) reconstructs
// the accumulated state and renders that step. A stale id (already deleted) redirects
// back to /reports rather than 500ing. Stepping and finishing post to
// /reports/schedule/{id}/edit (editReportSchedule).
func (s *server) editReportScheduleWizard(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}
	// A PRG GET carries the accumulated state; render it without re-reading the row.
	if r.URL.Query().Get("step") != "" {
		v := readScheduleWizardView(r)
		v.ID = id
		s.renderScheduleWizard(r.Context(), w, r, acct, v, true)
		return
	}
	sc, err := s.store.GetReportSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Redirect(w, r, "/reports", http.StatusSeeOther)
			return
		}
		s.serverError(w, "get report schedule", err)
		return
	}
	cad, cron := reportCadPresetFor(sc.Cadence)
	v := scheduleWizardView{
		ID:       sc.ID,
		Name:     sc.Name,
		Sections: parseScheduleSections(sc.Sections),
		Cad:      cad,
		Cron:     cron,
	}
	if sc.ChannelID.Valid {
		v.ChannelID = sc.ChannelID.Int64
	}
	s.renderScheduleWizard(r.Context(), w, r, acct, v, true)
}

// parseScheduleSections reads a schedule's stored sections JSON array back into the
// canonical key list, dropping anything unknown so a hand-edited row cannot surface a
// stray checkbox.
func parseScheduleSections(raw []byte) []string {
	var keys []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &keys)
	}
	return canonicalSections(keys)
}

// createReportSchedule drives the create wizard: Back/Next re-render the step
// (advancing only when the current step's gate passes, mirroring the example's
// disabled Next), and the finishing submit files the schedule and redirects to
// /reports. requireAdmin gates every path, so a viewer is refused before the handler.
func (s *server) createReportSchedule(w http.ResponseWriter, r *http.Request, acct db.Account) {
	v := readScheduleWizardView(r)

	switch r.FormValue("action") {
	case "back":
		if v.Step > 0 {
			v.Step--
		}
		redirectWizardStep(w, r, reportsNewWizardPath, v)
		return
	case "next":
		if v.Step < reportScheduleLast && scheduleStepValid(v) {
			v.Step++
		}
		redirectWizardStep(w, r, reportsNewWizardPath, v)
		return
	}

	// Finish. Redirect back to the first step where the operator can fix an incomplete
	// entry rather than filing a schedule that would render nothing.
	if !scheduleAllValid(v) {
		v.Step = 0
		redirectWizardStep(w, r, reportsNewWizardPath, v)
		return
	}

	sections, err := json.Marshal(v.Sections)
	if err != nil {
		s.serverError(w, "marshal schedule sections", err)
		return
	}
	if _, err := s.store.InsertReportSchedule(r.Context(), db.InsertReportScheduleParams{
		Name:           strings.TrimSpace(v.Name),
		Sections:       sections,
		Cadence:        reportCadLabel(v.Cad, v.Cron),
		Format:         reportScheduleFormat,
		DeliveryTarget: "", // superseded by the channel binding below.
		ChannelID:      channelBinding(v.ChannelID),
		CreatedBy:      acct.ID,
	}); err != nil {
		s.serverError(w, "insert report schedule", err)
		return
	}
	http.Redirect(w, r, "/reports", http.StatusSeeOther)
}

// editReportSchedule drives the edit wizard: the same Back/Next stepping as create,
// but the finishing submit updates the target schedule in place (a genuine update,
// not a recompute — migration 21700) and redirects to /reports. A missing or stale id
// answers /reports rather than 500ing.
func (s *server) editReportSchedule(w http.ResponseWriter, r *http.Request, acct db.Account) {
	v := readScheduleWizardView(r)
	if v.ID == 0 {
		if id, err := strconv.ParseInt(r.PathValue("id"), 10, 64); err == nil {
			v.ID = id
		}
	}
	if v.ID == 0 {
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}

	switch r.FormValue("action") {
	case "back":
		if v.Step > 0 {
			v.Step--
		}
		redirectWizardStep(w, r, reportsEditWizardPath(v.ID), v)
		return
	case "next":
		if v.Step < reportScheduleLast && scheduleStepValid(v) {
			v.Step++
		}
		redirectWizardStep(w, r, reportsEditWizardPath(v.ID), v)
		return
	}

	if !scheduleAllValid(v) {
		v.Step = 0
		redirectWizardStep(w, r, reportsEditWizardPath(v.ID), v)
		return
	}

	sections, err := json.Marshal(v.Sections)
	if err != nil {
		s.serverError(w, "marshal schedule sections", err)
		return
	}
	if _, err := s.store.UpdateReportSchedule(r.Context(), db.UpdateReportScheduleParams{
		ID:             v.ID,
		Name:           strings.TrimSpace(v.Name),
		Sections:       sections,
		Cadence:        reportCadLabel(v.Cad, v.Cron),
		Format:         reportScheduleFormat,
		DeliveryTarget: "",
		ChannelID:      channelBinding(v.ChannelID),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The schedule was deleted between opening the wizard and saving.
			http.Redirect(w, r, "/reports", http.StatusSeeOther)
			return
		}
		s.serverError(w, "update report schedule", err)
		return
	}
	http.Redirect(w, r, "/reports", http.StatusSeeOther)
}

// runReportScheduleNow dispatches one on-demand run of a schedule (the row menu's
// "Run now"). It cuts the artifact for the current period with the canonical
// renderer (internal/message.RenderArtifact) and stamps a report_delivery receipt
// (#291/T2). The wizard declares no recipient, so delivery_target is empty and the
// run generates without delivering — state "generated", no delivered_at (migration
// 22500). The receipt records only the period bounds; the artifact recomputes its
// contents from them at view time, carrying nothing off-instance (ADR-0039). A stale
// id redirects to /reports rather than 500ing.
func (s *server) runReportScheduleNow(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}
	sc, err := s.store.GetReportSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Redirect(w, r, "/reports", http.StatusSeeOther)
			return
		}
		s.serverError(w, "get report schedule", err)
		return
	}

	now := s.now().UTC()
	start := now.Add(-report.CadenceWindow(sc.Cadence))

	no, err := s.store.NextReportDeliveryNo(r.Context(), id)
	if err != nil {
		s.serverError(w, "next report delivery no", err)
		return
	}

	// Cut the artifact for the current period. The delivered document recomputes from
	// the period bounds at render time, so this render confirms the report is cuttable
	// for the window; the receipt snapshots nothing. Content wiring lands with the
	// delivery backend (T5) — here the canonical renderer draws the current period.
	_ = message.RenderArtifact(message.Artifact{
		Title:       sc.Name,
		PeriodStart: start.Format("2006-01-02"),
		PeriodEnd:   now.Format("2006-01-02"),
		DeliveryNo:  int(no),
		GeneratedAt: now.Format("2006-01-02"),
		Format:      sc.Format,
	})

	if _, err := s.store.InsertReportDelivery(r.Context(), db.InsertReportDeliveryParams{
		ScheduleID:  id,
		PeriodStart: pgtype.Timestamptz{Time: start, Valid: true},
		PeriodEnd:   pgtype.Timestamptz{Time: now, Valid: true},
		DeliveryNo:  no,
		State:       "generated",
		DeliveredAt: pgtype.Timestamptz{}, // download-only: generated, not delivered.
	}); err != nil {
		s.serverError(w, "insert report delivery", err)
		return
	}
	http.Redirect(w, r, "/reports", http.StatusSeeOther)
}

// deleteReportSchedule removes a schedule (the row menu's "Delete"). Delete is
// idempotent from the caller's view — a stale id is a no-op, not an error — so it
// always redirects to /reports. requireAdmin gates it.
func (s *server) deleteReportSchedule(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}
	if err := s.store.DeleteReportSchedule(r.Context(), id); err != nil {
		s.serverError(w, "delete report schedule", err)
		return
	}
	http.Redirect(w, r, "/reports", http.StatusSeeOther)
}

// channelBinding maps the wizard's chosen destination to the schedule's channel_id:
// 0 is "Download only" — a NULL binding, so the run generates in-instance and enqueues
// no ready-message — and any other value binds that Channel (P0.6c/T7).
func channelBinding(channelID int64) pgtype.Int8 {
	if channelID == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: channelID, Valid: true}
}

// renderScheduleWizard shapes the controlled state into the "schedulewizard"
// template data: the step progress, the current step's fields, and — on the Review
// step — the KeyValueList summary of the real inputs. editMode switches the form's
// post target and the finish label between the create and edit paths. The Delivery
// step's Destination select is built from the declared Channels (ListChannels);
// a list-read failure degrades to "Download only" alone rather than 500ing the wizard.
func (s *server) renderScheduleWizard(ctx context.Context, w http.ResponseWriter, r *http.Request, acct db.Account, v scheduleWizardView, editMode bool) {
	steps := make([]map[string]any, len(reportScheduleStepTitles))
	for i, title := range reportScheduleStepTitles {
		steps[i] = map[string]any{
			"Num":     i + 1,
			"Title":   title,
			"Done":    i < v.Step,
			"Current": i == v.Step,
		}
	}

	selected := make(map[string]bool, len(v.Sections))
	for _, k := range v.Sections {
		selected[k] = true
	}
	sectionOpts := make([]map[string]any, len(reportScheduleSections))
	labels := make([]string, 0, len(v.Sections))
	for i, sec := range reportScheduleSections {
		on := selected[sec.Key]
		sectionOpts[i] = map[string]any{"Key": sec.Key, "Label": sec.Label, "Checked": on}
		if on {
			labels = append(labels, sec.Label)
		}
	}

	cads := make([]map[string]any, len(reportCadPresets))
	for i, p := range reportCadPresets {
		cads[i] = map[string]any{"Value": p, "Selected": p == v.Cad}
	}

	// The Delivery step's Destination listbox: "Download only" (value 0, the default)
	// plus one option per declared Channel, labelled by its URL (the spec listbox — the
	// trigger shows .ChannelLabel, view JS syncs the hidden `channel` input). A read
	// failure leaves only "Download only" — the wizard still works. channelLabel is the
	// trigger label (and the Review row's value): the bound channel's URL, or the
	// download-only label.
	channelLabel := "Download only"
	deliveryLabel := "download only"
	channelOpts := []map[string]any{
		{"Value": int64(0), "Label": "Download only", "Hint": "artifact stays in Reports", "Selected": v.ChannelID == 0},
	}
	if channels, err := s.store.ListChannels(ctx); err != nil {
		log.Printf("web: reports: list channels for wizard: %v", err)
	} else {
		for _, c := range channels {
			sel := c.ID == v.ChannelID
			if sel {
				deliveryLabel = c.Url
				channelLabel = c.Url
			}
			channelOpts = append(channelOpts, map[string]any{
				"Value": c.ID, "Label": c.Url, "Hint": "signed HTTPS channel", "Selected": sel,
			})
		}
	}

	// Review summary — the real inputs, exactly as the example's KeyValueList maps
	// them: the name (or an em dash), the chosen section labels, the cadence label,
	// the fixed format, and the delivery destination.
	nameSummary := strings.TrimSpace(v.Name)
	if nameSummary == "" {
		nameSummary = "—"
	}
	sectionsSummary := "—"
	if len(labels) > 0 {
		sectionsSummary = strings.Join(labels, ", ")
	}
	review := []map[string]any{
		{"K": "Report", "V": nameSummary},
		{"K": "Sections", "V": sectionsSummary},
		{"K": "Cadence", "V": reportCadLabel(v.Cad, v.Cron)},
		{"K": "Format", "V": reportScheduleFormat},
		{"K": "Delivery", "V": deliveryLabel},
	}

	formAction := reportsNewWizardPath
	finishLabel := "Create schedule"
	title := "New report schedule"
	if editMode {
		formAction = reportsEditWizardPath(v.ID)
		finishLabel = "Save schedule"
		title = "Edit report schedule"
	}

	s.render(w, r, "schedulewizard", map[string]any{
		"Title":       title,
		"Account":     acct,
		"IsAdmin":     acct.Role == roleAdmin,
		"NavActive":   "reports",
		"DesignTokens": true,

		"WizardTitle": title,
		"FormAction":  formAction,
		"FinishLabel": finishLabel,
		"EditMode":    editMode,
		"ID":          v.ID,

		"Step":      v.Step,
		"StepNum":   v.Step + 1,
		"StepTotal": len(reportScheduleStepTitles),
		"Last":      v.Step == reportScheduleLast,
		"Steps":     steps,

		"Name":         v.Name,
		"Sections":     sectionOpts,
		"SectionsKeys": v.Sections,
		"Cads":         cads,
		"Cad":          v.Cad,
		"Cron":         v.Cron,
		"Custom":       v.Cad == reportCustomCad,
		"Channels":     channelOpts,
		"ChannelID":    v.ChannelID,
		"ChannelLabel": channelLabel,

		"Review": review,
	})
}
