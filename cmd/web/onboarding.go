package main

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/onboarding.tmpl"))

var onboardStepTitles = []string{"Seeds", "Cadence", "Channel", "Review"}

const onboardLast = 3

var onboardCadPresets = []string{"Every 6h", "Daily · 08:00", "Weekly · mon 09:00", "Monthly · 1st", "Custom…"}

const (
	onboardDefaultCad = "Daily · 08:00"
	onboardCustomCad  = "Custom…"
)

type onboardView struct {
	Step    int
	Seeds   []string
	Profile string
	Cad     string
	Cron    string
	Channel string
}

func parseSeedTokens(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

func readOnboardView(r *http.Request) onboardView {
	_ = r.ParseForm()

	step := 0
	if n, err := strconv.Atoi(r.FormValue("step")); err == nil {
		step = n
	}
	if step < 0 {
		step = 0
	}
	if step > onboardLast {
		step = onboardLast
	}

	seeds := dedupeStrings(append(parseSeedTokens(r.FormValue("seeds")), parseSeedTokens(r.FormValue("seedsadd"))...))

	profile := r.FormValue("profile")
	if profile != "passive" {
		profile = "standard"
	}
	cad := r.FormValue("cad")
	if cad == "" {
		cad = onboardDefaultCad
	}

	return onboardView{
		Step:    step,
		Seeds:   seeds,
		Profile: profile,
		Cad:     cad,
		Cron:    strings.TrimSpace(r.FormValue("cron")),
		Channel: r.FormValue("channel"),
	}
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func onboardStepValid(v onboardView) bool {
	switch v.Step {
	case 0:
		return len(v.Seeds) > 0
	case 1:
		return v.Cad != onboardCustomCad || strings.TrimSpace(v.Cron) != ""
	default:
		return true
	}
}

func onboardingScanKind(profile string) string {
	if profile == "passive" {
		return scan.DNSKind
	}
	return scan.HotKind
}

func (s *server) onboarding(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderOnboard(w, r, acct, readOnboardView(r))
}

func redirectOnboardStep(w http.ResponseWriter, r *http.Request, v onboardView) {
	q := url.Values{}
	q.Set("step", strconv.Itoa(v.Step))
	q.Set("seeds", strings.Join(v.Seeds, ","))
	q.Set("profile", v.Profile)
	q.Set("cad", v.Cad)
	if v.Cron != "" {
		q.Set("cron", v.Cron)
	}
	q.Set("channel", v.Channel)
	http.Redirect(w, r, "/onboarding?"+q.Encode(), http.StatusSeeOther)
}

func (s *server) onboardingStep(w http.ResponseWriter, r *http.Request, acct db.Account) {
	v := readOnboardView(r)

	if rm := r.FormValue("rm"); rm != "" {
		out := v.Seeds[:0]
		for _, seed := range v.Seeds {
			if seed != rm {
				out = append(out, seed)
			}
		}
		v.Seeds = out
		redirectOnboardStep(w, r, v)
		return
	}

	switch r.FormValue("action") {
	case "back":
		if v.Step > 0 {
			v.Step--
		}
	case "next":
		if v.Step < onboardLast && onboardStepValid(v) {
			v.Step++
		}
	}
	redirectOnboardStep(w, r, v)
}

func (s *server) renderOnboard(w http.ResponseWriter, r *http.Request, acct db.Account, v onboardView) {
	steps := make([]map[string]any, len(onboardStepTitles))
	for i, title := range onboardStepTitles {
		steps[i] = map[string]any{
			"Num":     i + 1,
			"Title":   title,
			"Done":    i < v.Step,
			"Current": i == v.Step,
		}
	}

	cads := make([]map[string]any, len(onboardCadPresets))
	for i, p := range onboardCadPresets {
		cads[i] = map[string]any{"Value": p, "Selected": p == v.Cad}
	}

	seedsSummary := "—"
	if len(v.Seeds) > 0 {
		seedsSummary = strings.Join(v.Seeds, ", ")
	}
	cadence := strings.ToLower(v.Cad)
	if v.Cad == onboardCustomCad {
		cadence = v.Cron
	}
	channelSummary := strings.TrimSpace(v.Channel)
	if channelSummary == "" {
		channelSummary = "none — inbox only"
	}
	review := []map[string]any{
		{"K": "Seeds", "V": seedsSummary},
		{"K": "Profile", "V": v.Profile},
		{"K": "Cadence", "V": cadence},
		{"K": "Channel", "V": channelSummary},
	}

	data := map[string]any{
		"Title":     "Set up this workspace",
		"Account":   acct,
		"IsAdmin":   acct.Role == roleAdmin,
		"NavActive": "",

		"Step":      v.Step,
		"StepNum":   v.Step + 1,
		"StepTotal": len(onboardStepTitles),
		"Last":      v.Step == onboardLast,
		"Steps":     steps,
		// The no-JS floor under the template's own validity gate, so both must agree.
		"StepValid": onboardStepValid(v),

		"Seeds":      v.Seeds,
		"SeedsField": strings.Join(v.Seeds, ","),
		"Profile":    v.Profile,
		"Cads":       cads,
		"Cad":        v.Cad,
		"Cron":       v.Cron,
		"Custom":     v.Cad == onboardCustomCad,
		"Channel":    v.Channel,

		"Review": review,
		"Kind":   onboardingScanKind(v.Profile),
	}
	s.render(w, r, "onboarding", data)
}
