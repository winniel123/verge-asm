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

// A schedule is Declared and holds no timeline, so an edit updates in place, never recomputes.

type reportScheduleSection struct {
	Key   string
	Label string
}

var reportScheduleSections = []reportScheduleSection{
	{"kpis", "Summary KPIs"},
	{"new-assets", "New assets"},
	{"signal-changes", "Signal changes"},
	{"coverage-gaps", "Coverage gaps"},
}

func reportScheduleDefaultSections() []string {
	return []string{
		reportScheduleSections[0].Key,
		reportScheduleSections[1].Key,
		reportScheduleSections[2].Key,
	}
}

var reportCadPresets = []string{"Every 6h", "Daily · 08:00", "Weekly · mon 09:00", "Monthly · 1st", "Custom…"}

const (
	reportDefaultCad     = "Weekly · mon 09:00"
	reportCustomCad      = "Custom…"
	reportScheduleFormat = "pdf" // The wizard has no format control, so every schedule is a pdf.
)

var reportScheduleStepTitles = []string{"Scope", "Cadence", "Delivery", "Review"}

const (
	reportScheduleDeliveryStep = 2
	reportScheduleLast         = 3
)

type scheduleWizardView struct {
	Step      int
	ID        int64
	Name      string
	Sections  []string
	Cad       string
	Cron      string
	ChannelID int64

	// Held raw: resolveBack is the only guard that decides whether this may become a Location.

	Back string
}

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
		Back:      strings.TrimSpace(r.FormValue(backField)),
	}
}

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

func scheduleCadenceValid(v scheduleWizardView) bool {
	// An uninterpretable cadence is refused at authoring, never coerced to a default (ADR-0122 §6).
	if v.Cad != reportCustomCad {
		return true
	}
	cron := strings.TrimSpace(v.Cron)
	return cron != "" && report.ValidateCron(cron) == nil
}

func scheduleStepValid(v scheduleWizardView) bool {
	switch v.Step {
	case 0:
		return strings.TrimSpace(v.Name) != "" && len(v.Sections) > 0
	case 1:
		return scheduleCadenceValid(v)
	default:
		return true
	}
}

func scheduleAllValid(v scheduleWizardView) bool {
	// A hand-crafted finish POST can skip a step, so every step's gate is re-checked here.
	return strings.TrimSpace(v.Name) != "" && len(v.Sections) > 0 && scheduleCadenceValid(v)
}

func scheduleFirstInvalidStep(v scheduleWizardView) int {
	if strings.TrimSpace(v.Name) == "" || len(v.Sections) == 0 {
		return 0
	}
	if !scheduleCadenceValid(v) {
		return 1
	}
	return 0
}

func reportCadLabel(cad, cron string) string {
	if cad == reportCustomCad {
		if c := strings.TrimSpace(cron); c != "" {
			return c
		}
		return "custom"
	}
	return strings.ToLower(cad)
}

func reportCadPresetFor(cadence string) (cad, cron string) {
	for _, p := range reportCadPresets {
		if p != reportCustomCad && strings.ToLower(p) == cadence {
			return p, ""
		}
	}
	return reportCustomCad, cadence
}

const reportsNewWizardPath = "/reports/schedule/new"

// A stale id is a no-op on this surface, not an error: every act lands back on the list.

const reportsPath = "/reports"

func reportsEditWizardPath(id int64) string {
	return "/reports/schedule/" + strconv.FormatInt(id, 10) + "/edit"
}

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
	if v.Back != "" {
		q.Set(backField, v.Back)
	}
	// The route here is a constant and this is only its query, so no operator value reaches the host.
	http.Redirect(w, r, base+"?"+q.Encode(), http.StatusSeeOther)
}

func (s *server) newReportScheduleWizard(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "schedulewizard", s.reportsWizardFixtureData(r, acct))
		return
	}
	var v scheduleWizardView
	if r.URL.Query().Get("step") == "" {
		v = scheduleWizardView{
			Sections: reportScheduleDefaultSections(),
			Cad:      reportDefaultCad,
			Back:     strings.TrimSpace(r.FormValue(backField)),
		}
	} else {
		v = readScheduleWizardView(r)
	}
	s.renderScheduleWizard(r.Context(), w, r, acct, v, false)
}

func (s *server) editReportScheduleWizard(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.redirectBack(w, r, reportsPath)
		return
	}
	if r.URL.Query().Get("step") != "" {
		v := readScheduleWizardView(r)
		v.ID = id
		s.renderScheduleWizard(r.Context(), w, r, acct, v, true)
		return
	}
	sc, err := s.store.GetReportSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.redirectBack(w, r, reportsPath)
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
		Back:     strings.TrimSpace(r.FormValue(backField)),
	}
	if sc.ChannelID.Valid {
		v.ChannelID = sc.ChannelID.Int64
	}
	s.renderScheduleWizard(r.Context(), w, r, acct, v, true)
}

func parseScheduleSections(raw []byte) []string {
	var keys []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &keys)
	}
	return canonicalSections(keys)
}

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

	if !scheduleAllValid(v) {
		v.Step = scheduleFirstInvalidStep(v)
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
		DeliveryTarget: "",
		ChannelID:      channelBinding(v.ChannelID),
		CreatedBy:      acct.ID,
	}); err != nil {
		s.serverError(w, "insert report schedule", err)
		return
	}
	s.redirectBack(w, r, reportsPath)
}

func (s *server) editReportSchedule(w http.ResponseWriter, r *http.Request, acct db.Account) {
	v := readScheduleWizardView(r)
	if v.ID == 0 {
		if id, err := strconv.ParseInt(r.PathValue("id"), 10, 64); err == nil {
			v.ID = id
		}
	}
	if v.ID == 0 {
		s.redirectBack(w, r, reportsPath)
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
		v.Step = scheduleFirstInvalidStep(v)
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
			s.redirectBack(w, r, reportsPath)
			return
		}
		s.serverError(w, "update report schedule", err)
		return
	}
	s.redirectBack(w, r, reportsPath)
}

func (s *server) runReportScheduleNow(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.redirectBack(w, r, reportsPath)
		return
	}
	sc, err := s.store.GetReportSchedule(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.redirectBack(w, r, reportsPath)
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

	// The result is discarded: this render only confirms the period is cuttable.
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
		DeliveredAt: pgtype.Timestamptz{},
	}); err != nil {
		s.serverError(w, "insert report delivery", err)
		return
	}
	s.redirectBack(w, r, reportsPath)
}

func (s *server) deleteReportSchedule(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.redirectBack(w, r, reportsPath)
		return
	}
	if err := s.store.DeleteReportSchedule(r.Context(), id); err != nil {
		s.serverError(w, "delete report schedule", err)
		return
	}
	s.redirectBack(w, r, reportsPath)
}

func channelBinding(channelID int64) pgtype.Int8 {
	if channelID == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: channelID, Valid: true}
}

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
		"Title":     title,
		"Account":   acct,
		"IsAdmin":   acct.Role == roleAdmin,
		"NavActive": "reports",

		// Chrome injection stamps BackURL only when unset, so setting it here, even empty, matters.
		"BackURL": v.Back,

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
