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

// The onboarding wizard is now byte-served from the design-owned, frozen onboarding.tmpl
// (package v3.12.0, WORKFLOW v4, map #20), embedded read-only via the designfs package and
// parsed into the shared set here. The repo authors no onboarding markup/CSS/JS:
// templates_onboarding.go is deleted (the "onboarding" define moves to the frozen tmpl). The
// tmpl is self-contained (it calls only the shared head/chrome/foot defines) and auto-embeds
// through designfs's existing templates/*.tmpl glob, so no designfs.go change is needed.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/onboarding.tmpl"))

// The onboarding wizard (#307, T12, ADR-0110) — the first-run setup ported from
// design-system/examples/console/Onboarding.jsx: the seeds → cadence → channel →
// review flow that queues the first scan on completion. The example is a Wizard
// dialog (TagInput, RadioCards, CadenceSelect, Input, KeyValueList); the app is
// server-rendered with no client runtime, so each step is a controlled server
// form — the accumulated state rides hidden fields, Back/Next re-render, and the
// per-step valid gate decides whether Next advances. The markup lives in
// templates_onboarding.go; those components are translated to template-local CSS
// within the existing token vocabulary (restyling, not authoring — ADR-0109). No
// design-system component is authored here.
//
// Completion does not fabricate a scan: the review step's "Start first scan"
// button posts to /onboarding/finish, which runs the existing on-demand dispatch
// (runTrigger, scantrigger.go) behind the same requireAdmin gate POST
// /scans/trigger uses. The scan kind is carried as the trigger's `kind`
// field, mapped from the chosen profile — a standard profile runs the active hot
// port scan, a passive profile the dns discovery scan — so the same guardrails
// (disabled-scan refusal, overlap protection) and the same fan-out run, and the
// operator lands on the Scans monitor watching the first scan.

// onboardStepTitles names the four wizard steps in order; onboardLast is the index
// of the review step, where the flow finishes rather than advances.
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

// parseSeedTokens splits a raw seed entry into individual tokens on commas and
// whitespace, the same commit boundary the TagInput uses (Enter/comma commits).
// Empty tokens are dropped. It never enumerates hosts — a token is a domain or a
// CIDR range (a seed/scope), which discovery expands into subjects.
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

// dedupeStrings returns the input with duplicates removed, preserving first-seen
// order — the TagInput never commits a value it already holds.
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

// onboardingScanKind maps the chosen scan profile to the scan kind the first scan
// dispatches. A standard profile (top TCP ports, active) runs the hot port scan; a
// passive profile (public datasets only, no active probing) runs the dns discovery
// scan. Both ship enabled, so the first scan actually fans out.
func onboardingScanKind(profile string) string {
	if profile == "passive" {
		return scan.DNSKind
	}
	return scan.HotKind
}

func (s *server) onboarding(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderOnboard(w, r, acct, readOnboardView(r))
}

// redirectOnboardStep 303-redirects the wizard to a GET at /onboarding carrying the
// accumulated controlled state as query parameters (#25d, the batch-5 #23f precedent) —
// the post-back PRG shape. The GET handler (onboarding) reconstructs the same view from the
// query and renders the step, so every wizard state is bookmarkable and harness-addressable
// (the wizard goldens hit these GET URLs directly). The seeds ride comma-joined, exactly as
// the hidden `seeds` field carries them; a typed seed absorbed on this submit is already
// folded into v.Seeds, so it rides forward as committed state.
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

// onboardingStep advances or rewinds the controlled flow, then 303-redirects to the GET for
// the resulting step (#25d PRG). A chip's remove button (`rm`) drops that seed and re-renders
// the same step; Back steps back; Next advances only when the current step's valid gate passes
// (mirroring the example's disabled Next), otherwise it redirects back to the same step so the
// operator can fix it.
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

	// Review summary — the real inputs, exactly as the example's KeyValueList maps
	// them: seeds joined (or an em dash), the profile, the cadence (the cron when
	// custom, else the lowercased preset), and the channel (or inbox-only).
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
		// StepValid is the server-computed validity of the rendered step (#25d, NEW): it
		// renders the Next/Start `disabled` attribute server-side as the no-JS floor under
		// the frozen tmpl's JS validity gate (≥1 seed on step 0 — a typed seed is absorbed on
		// submit; a cron required when the cadence is Custom…).
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
