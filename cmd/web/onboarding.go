package main

import (
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

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
// button posts to /onboarding/finish, which is wired to the existing on-demand
// trigger handler (triggerScan, scantrigger.go) behind the same requireAdmin gate
// POST /scans/trigger uses. The scan kind is carried as the trigger's `kind`
// field, mapped from the chosen profile — a standard profile runs the active hot
// port scan, a passive profile the dns discovery scan — so the same guardrails
// (disabled-scan refusal, overlap protection) and the same fan-out run, and the
// operator lands on the Scans monitor watching the first scan.

// onboardStepTitles names the four wizard steps in order; onboardLast is the index
// of the review step, where the flow finishes rather than advances.
var onboardStepTitles = []string{"Seeds", "Cadence", "Channel", "Review"}

const onboardLast = 3

// onboardCadPresets are the CadenceSelect presets, ported verbatim from
// design-system/components/forms/CadenceSelect.jsx. The default is the second
// (Daily · 08:00), matching the example's initial cad; the last (Custom…) reveals
// a cron field and gates Next until it is filled.
var onboardCadPresets = []string{"Every 6h", "Daily · 08:00", "Weekly · mon 09:00", "Monthly · 1st", "Custom…"}

const (
	onboardDefaultCad = "Daily · 08:00"
	onboardCustomCad  = "Custom…"
)

// onboardView is the controlled state of the wizard across the post-back flow: the
// step being shown and every field's current value. Seeds are the committed tag
// list (the TagInput's values); Profile / Cad / Cron / Channel are the remaining
// controlled inputs.
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

// readOnboardView reconstructs the controlled state from the request form (or, on
// GET, the query — ParseForm merges both). Seeds accumulate: the committed list in
// the hidden `seeds` field plus any new tokens typed into `seedsadd`, deduped in
// first-seen order. Defaults match the example's initial state (standard profile,
// daily cadence).
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

// onboardStepValid is the per-step valid gate, ported from the example's step
// `valid` predicates: seeds needs at least one seed; cadence needs a cron when the
// custom preset is chosen; channel and review are always valid.
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

// onboarding renders the wizard at its current step. It is a viewer-safe read —
// stepping mutates nothing; only completion (POST /onboarding/finish) enqueues a
// scan, and that is admin-gated.
func (s *server) onboarding(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderOnboard(w, r, acct, readOnboardView(r))
}

// onboardingStep advances or rewinds the controlled flow. A chip's remove button
// (`rm`) drops that seed and re-renders the same step; Back steps back; Next
// advances only when the current step's valid gate passes (mirroring the example's
// disabled Next), otherwise it re-renders the same step so the operator can fix it.
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
		s.renderOnboard(w, r, acct, v)
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
	s.renderOnboard(w, r, acct, v)
}

// renderOnboard shapes the controlled state into the template data: the step
// progress, the current step's fields, and — on the review step — the KeyValueList
// summary of the real inputs plus the scan kind the finish button carries.
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
	s.render(w, "onboarding", data)
}
