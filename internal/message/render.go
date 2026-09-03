package message

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValenceWords are the words the message vocabulary refuses (v1 spec §5.3,
// ADR-0064): nothing is resolved, fixed, improved, critical or OK, because a
// clear is not always good news and a widening is neither. The list is exported
// so a test can assert every rendered sentence in the product is clear of them,
// and so the store's read path can be checked too. Every value is a lowercase
// whole word; ContainsValence matches on word boundaries, so a value can never
// falsely fire on a substring (`ok` inside `looked`, say).
var ValenceWords = []string{
	"resolved", "resolve", "fixed", "fix", "improved", "improve",
	"critical", "ok", "okay", "good", "bad", "safe", "unsafe",
	"danger", "dangerous", "severe", "severity", "healthy", "unhealthy",
	"clean", "warning", "warn", "urgent", "vulnerable", "vulnerability",
	"success", "failure", "risk", "threat", "worse", "better",
}

var valenceRe = func() *regexp.Regexp {
	quoted := make([]string, len(ValenceWords))
	for i, w := range ValenceWords {
		quoted[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(`(?i)\b(` + strings.Join(quoted, "|") + `)\b`)
}()

func ContainsValence(s string) bool { return valenceRe.MatchString(s) }

// Threshold returns a clock-class Message for a rule whose threshold was crossed
// with no measurement moving — the estate did not move and we did not change how
// we look, only time passed (CONTEXT.md `Message`). It links to the object whose
// span the rule read. Where the same rule instead finds the span it reads has
// moved, the caller fires CauseDrift (a drift-class firing) rather than this —
// class is a property of the firing.
func Threshold(subjectKind, subjectKey, headline string, instant time.Time) *Message {
	return &Message{
		Cause:       CauseThreshold,
		Class:       ClassClock,
		SubjectKind: subjectKind,
		FiredAt:     subjectKey,
		Instant:     instant,
		Headline:    headline,
	}
}

func DeclaredInput(sourceKey, headline string, instant time.Time) *Message {
	return &Message{
		Cause:       CauseDeclaredInput,
		Class:       ClassCoverage,
		SubjectKind: "source",
		FiredAt:     sourceKey,
		Instant:     instant,
		Headline:    headline,
	}
}

func flagshipHeadline(serviceKey string, census Census) string {
	return fmt.Sprintf("%s reached from the internet · %s opened beneath it",
		serviceKey, plural(census.Len(), "facet", "facets"))
}

func membershipHeadline(entry Entry, rootKey string, census Census) string {
	verb := map[Entry]string{
		EntryAppeared: "entered the estate",
		EntryReturned: "returned to the estate",
		EntryRevealed: "came into view",
	}[entry]
	if verb == "" {
		verb = "entered the estate"
	}
	return fmt.Sprintf("%s %s · %s opened beneath it",
		rootKey, verb, plural(census.Len(), "timeline", "timelines"))
}

// narrowingHeadline states a scope narrowing with its two counts as factors, in
// the shape the narrowing-receipt prototype fixed (#167): the scope narrowed,
// the value excluded, the subjects withdrawn, the timelines taken out.
func narrowingHeadline(scope, removed string, subjects, timelines int) string {
	return fmt.Sprintf("%s narrowed · %s excluded · %s withdrawn · %s taken out of the estate",
		scope, removed,
		plural(subjects, "subject", "subjects"),
		plural(timelines, "timeline", "timelines"))
}

// seedWithdrawalHeadline states a Seed withdrawal with the same two counts as
// factors (ADR-0134 §6). It is narrowingHeadline's twin for the act that takes the
// scope itself away.
//
// It needs its own sentence because narrowingHeadline names two objects — a scope
// that narrowed and a value excluded from it — and this act has only one. Rendering
// it through narrowingHeadline would read "10.0.0.0/24 narrowed · 10.0.0.0/24
// excluded", which says EXCLUDED for an act that declared no exclusion and NARROWED
// for a scope that is gone. A Message is written once and never recomputed, so a
// false sentence written here stays false for as long as the corpus is retained.
func seedWithdrawalHeadline(scope string, subjects, timelines int) string {
	return fmt.Sprintf("%s withdrawn · %s withdrawn · %s taken out of the estate",
		scope,
		plural(subjects, "subject", "subjects"),
		plural(timelines, "timeline", "timelines"))
}

// narrowingLoss names what can no longer be told — the one payload element that
// is not a mirror of the widening receipt, because the narrowing cannot be
// corrected afterwards and naming it is the whole of the remedy (ADR-0074).
func narrowingLoss(removed string) string {
	return fmt.Sprintf("A listener answering inside %s after this act is not seen, "+
		"and no later message names it.", removed)
}

// plural renders a count with its noun, thousands-separated so a large factor
// reads (17,920 rather than 17920).
func plural(n int, one, many string) string {
	noun := many
	if n == 1 {
		noun = one
	}
	return fmt.Sprintf("%s %s", group(n), noun)
}

// Artifact is one delivered report, rendered — the data of a report delivery of
// the shape design-system/examples/console/ReportArtifact.jsx fixes, translated
// to the domain. RenderArtifact turns it into the delivered document itself: the
// same markup serves the console "view last delivery" surface and doubles as the
// PDF / email render spec, so it carries its own self-contained token styles
// (both light and dark) rather than leaning on the console stylesheet.
//
// Per-signal severity rides the artifact (P2.10, SPEC-CHANGE.md collision #11):
// the reference mock's "open signals by severity" bars and its severity-scored
// "new this week" table are the spec, and the domain now carries the datum — a
// signal's severity is its rule's severity (P0.1, internal/signal.Severity),
// which reconciles the old "a signal carries no severity" reading of ADR-0024
// (the design is normative for look AND functionality; where the domain lacked
// the datum the fix is to build it). When severity-carrying signals are present
// the artifact draws the by-severity bar breakdown and a SeverityBadge column,
// using the five exact levels and mirroring the on-screen ramp. Two holds still
// stand: change rides its OWN drift vocabulary and palette (appeared / revealed
// / withdrawn), never the severity ramp; and signals are withdrawn by the world,
// never resolved by an operator, so the mock's mean-time-to-resolve KPI is not
// modelled. With no delivery backend the artifact is genuinely Empty and the
// design-system empty-state stands — nothing is fabricated.
type Artifact struct {
	Title       string
	Org         string
	PeriodStart string
	PeriodEnd   string
	DeliveryNo  int
	GeneratedAt string
	Version     string
	Format      string
	// Stats is the KPI band — honest current-state / period scalars. No resolve
	// metric appears: signals are withdrawn by the world, never resolved.
	Stats          []ArtifactStat
	SeverityCounts []ArtifactSeverityCount
	Signals        []ArtifactSignal
	// Withdrawn is what the world withdrew this period ("withdrawn by the world"),
	// riding the drift vocabulary and palette, never the severity ramp.
	Withdrawn []ArtifactChange
	// Delivered is the delivery instant (ISO), empty where nothing was delivered;
	// ChannelHost is the destination host only — never the raw URL, where a token
	// an operator embedded would sit (mirrors the message panel, ADR-0081).
	Delivered   string
	ChannelHost string
}

// ArtifactStat is one KPI in the artifact's summary band: a label, a mono value,
// an optional signed delta with its tone, and a caption. Delta is pre-signed by
// the caller with a true minus; DeltaTone is "good" / "bad" / "neutral" and only
// selects a colour, never rendered as text.
type ArtifactStat struct {
	Label     string
	Value     string
	Delta     string
	DeltaTone string
	Caption   string
}

// ArtifactChange is one change row — a subject that appeared, was revealed, or was
// withdrawn this period, with a short note. Change is a drift-vocabulary word and
// selects the drift-palette family; it is never a severity level.
type ArtifactChange struct {
	Change  string
	Subject string
	Detail  string
}

// ArtifactSeverityCount is one bar in the "open signals by severity" breakdown: a
// severity level and the count of open signals carrying it. Level is one of the
// five exact tokens (critical / high / medium / low / info); any other value folds
// to info rather than manufacturing urgency (mirrors signal.SeverityFor).
type ArtifactSeverityCount struct {
	Level string
	Count int
}

type ArtifactSignal struct {
	Severity string
	Signal   string
	Asset    string
	Raised   string
}

// Empty reports whether the artifact carries no delivered content — no KPI band,
// no severity breakdown, no signals, and no change rows. A view of a schedule that
// has never delivered renders the design-system empty-state rather than a
// fabricated document (ADR-0110: a screen with no backing data ships an
// empty-state, never invented data).
func (a Artifact) Empty() bool {
	return len(a.Stats) == 0 && len(a.SeverityCounts) == 0 &&
		len(a.Signals) == 0 && len(a.Withdrawn) == 0
}

// artifactTokens is the self-contained slice of the design system the rendered
// artifact needs to stand alone as a PDF / email body, with no console stylesheet
// in scope. The custom-property names and values mirror cmd/web's pageCSS and
// design-system/tokens; dark ships two ways (prefers-color-scheme default and the
// explicit data-theme override) so the delivered document is legible in either.
// #nosec G101 ("artifactTokens" is a CSS <style> constant (design-system tokens), not a credential — name-heuristic FP)
const artifactTokens = `<style>
.vg-artifact{
  --surface:#ffffff; --sunken:#f2f0ec; --hairline:#e2dfdb; --border-strong:#c7c3be;
  --ink:#231f19; --body:#37322c; --muted:#79746d; --secondary:#67625c;
  --ok:#05773b; --ok-soft:#e1fae7; --ok-border:#bfebc9;
  --danger:#ac312c;
  --sev-critical-fill:#bf3631; --sev-critical-text:#ffffff; --sev-critical-dot:#bf3631;
  --sev-high-bg:#ffe9d6; --sev-high-border:#ffcdae; --sev-high-fg:#a04400; --sev-high-dot:#e26c00;
  --sev-medium-bg:#ffeecc; --sev-medium-border:#f4d59d; --sev-medium-fg:#8d5600; --sev-medium-dot:#e0a200;
  --sev-low-bg:#d7f7ff; --sev-low-border:#afe3f0; --sev-low-fg:#00728b; --sev-low-dot:#009aba;
  --sev-info-bg:#ebf2f9; --sev-info-border:#d0dae6; --sev-info-fg:#536579; --sev-info-dot:#798898;
  --drift-gain-bg:#f5ebff; --drift-gain-border:#e1d1ff; --drift-gain-fg:#6f4fa1;
  --drift-change-bg:#ffe6f7; --drift-change-border:#fec9e5; --drift-change-fg:#954074;
  --drift-loss-bg:#ecf1fa; --drift-loss-border:#d3dbe9; --drift-loss-fg:#56647a;
  --sans:"Instrument Sans","Helvetica Neue",Arial,sans-serif;
  --mono:"Geist Mono","SFMono-Regular",Consolas,ui-monospace,monospace;
  --r-md:12px; --r-lg:16px; --r-full:999px;
  --shadow-sm:0 1px 2px rgba(35,31,25,0.06),0 2px 8px rgba(35,31,25,0.04);
}
@media (prefers-color-scheme:dark){:root:not([data-theme="light"]) .vg-artifact{
  --surface:#1e1b17; --sunken:#191613; --hairline:#383530; --border-strong:#514c46;
  --ink:#eae7e4; --body:#d9d5d1; --muted:#898581; --secondary:#b1ada8;
  --ok:#57c07f; --ok-soft:#17281c; --ok-border:#1f4029;
  --danger:#f08c82;
  --sev-critical-fill:#c44039; --sev-critical-text:#ffffff; --sev-critical-dot:#e0564e;
  --sev-high-bg:#352414; --sev-high-border:#512f10; --sev-high-fg:#eba57b; --sev-high-dot:#df7a32;
  --sev-medium-bg:#322713; --sev-medium-border:#49360f; --sev-medium-fg:#d7b16a; --sev-medium-dot:#c68d00;
  --sev-low-bg:#192c2e; --sev-low-border:#144048; --sev-low-fg:#62c8df; --sev-low-dot:#00aed1;
  --sev-info-bg:#23272c; --sev-info-border:#3a4149; --sev-info-fg:#a9b3bf; --sev-info-dot:#8b98a6;
  --drift-gain-bg:#241b33; --drift-gain-border:#423658; --drift-gain-fg:#c0abe9;
  --drift-change-bg:#2f1725; --drift-change-border:#533044; --drift-change-fg:#e2a0c5;
  --drift-loss-bg:#1d2127; --drift-loss-border:#383d47; --drift-loss-fg:#aeb8c9;
  --shadow-sm:0 1px 2px rgba(0,0,0,0.35),0 2px 8px rgba(0,0,0,0.25);
}}
:root[data-theme="dark"] .vg-artifact{
  --surface:#1e1b17; --sunken:#191613; --hairline:#383530; --border-strong:#514c46;
  --ink:#eae7e4; --body:#d9d5d1; --muted:#898581; --secondary:#b1ada8;
  --ok:#57c07f; --ok-soft:#17281c; --ok-border:#1f4029;
  --danger:#f08c82;
  --sev-critical-fill:#c44039; --sev-critical-text:#ffffff; --sev-critical-dot:#e0564e;
  --sev-high-bg:#352414; --sev-high-border:#512f10; --sev-high-fg:#eba57b; --sev-high-dot:#df7a32;
  --sev-medium-bg:#322713; --sev-medium-border:#49360f; --sev-medium-fg:#d7b16a; --sev-medium-dot:#c68d00;
  --sev-low-bg:#192c2e; --sev-low-border:#144048; --sev-low-fg:#62c8df; --sev-low-dot:#00aed1;
  --sev-info-bg:#23272c; --sev-info-border:#3a4149; --sev-info-fg:#a9b3bf; --sev-info-dot:#8b98a6;
  --drift-gain-bg:#241b33; --drift-gain-border:#423658; --drift-gain-fg:#c0abe9;
  --drift-change-bg:#2f1725; --drift-change-border:#533044; --drift-change-fg:#e2a0c5;
  --drift-loss-bg:#1d2127; --drift-loss-border:#383d47; --drift-loss-fg:#aeb8c9;
  --shadow-sm:0 1px 2px rgba(0,0,0,0.35),0 2px 8px rgba(0,0,0,0.25);
}
</style>`

const microLabelStyle = `margin:0;font:500 11px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)`

// The delivered document's fixed copy — the section titles, the empty-section
// notes stating that nothing moved, and the empty-state block. Both render forms
// read these constants (RenderArtifact draws them as HTML, RenderArtifactPDF as
// print) so the two layouts cannot drift in WHAT the document says, only in how it
// looks (ADR-0114: one canonical content model, two layout forms).
const (
	artifactSeverityTitle  = "Open signals by severity"
	artifactSignalsTitle   = "New this week"
	artifactWithdrawnTitle = "Withdrawn by the world"
	artifactWithdrawnEmpty = "The world withdrew nothing in this period."
	artifactEmptyEyebrow   = "Nothing delivered"
	artifactEmptyHeadline  = "No report has been delivered yet"
	artifactEmptyBody      = "A delivered report is rendered here once report scheduling lands and a schedule runs. Until then there is no delivery to view."
)

// RenderArtifact renders one delivered report as the document itself by executing the
// design-owned "artifactdoc" define (design-system/templates/reportartifact.tmpl) with the
// recomputed data (SPEC-CHANGE #23g) — the SAME markup the on-screen /reports/delivery page
// and the PDF-HTML shell render, so the three forms cannot drift. The define is fully
// inline-styled; standalone (email / print) it needs only the design token vocabulary, which
// is prepended here so its var(--…) styles resolve with no console stylesheet in scope. An
// artifact with no delivered content renders the design-system empty-state inside the
// document (ADR-0110) rather than fabricating one. A render error (never expected — the
// template and data are internal) degrades to an HTML comment rather than panicking a caller.
func RenderArtifact(a Artifact) template.HTML {
	doc, err := renderArtifactDoc(BuildArtifactDoc(a))
	if err != nil {
		// #nosec G203 -- err text is HTML-escaped via html.EscapeString and wrapped in an HTML comment; no unescaped data reaches output.
		return template.HTML("<!-- artifact render error: " + html.EscapeString(err.Error()) + " -->")
	}
	// #nosec G203 -- artifactDocTokens is a trusted internal <style> constant; doc is html/template output (auto-escaped) built from internal report data, not attacker input.
	return template.HTML(artifactDocTokens) + doc
}

func artifactStatBand(stats []ArtifactStat) string {
	var b strings.Builder
	b.WriteString(`<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:20px">`)
	for _, s := range stats {
		b.WriteString(`<div style="display:flex;flex-direction:column;gap:6px;min-width:0">`)
		b.WriteString(`<span style="` + microLabelStyle + `">` + esc(s.Label) + `</span>`)
		b.WriteString(`<div style="display:flex;align-items:baseline;gap:8px">`)
		b.WriteString(`<span style="font:600 28px var(--mono);line-height:1.1;color:var(--ink)">` + esc(orDash(s.Value)) + `</span>`)
		if s.Delta != "" {
			b.WriteString(`<span style="font:500 12px var(--mono);color:` + deltaColor(s.DeltaTone) + `">` + esc(s.Delta) + `</span>`)
		}
		b.WriteString(`</div>`)
		if s.Caption != "" {
			b.WriteString(`<span style="font:400 11.5px var(--sans);color:var(--muted)">` + esc(s.Caption) + `</span>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// artifactChangeSection renders one titled change list — a mono eyebrow header and
// a row per change, each a drift-palette chip, the subject in mono, and its note.
// An empty section states the fact rather than vanishing (empty states are fact +
// next action; here the fact is that nothing moved).
func artifactChangeSection(title string, changes []ArtifactChange, emptyNote string) string {
	var b strings.Builder
	b.WriteString(`<div style="display:flex;flex-direction:column;gap:10px">`)
	b.WriteString(`<h2 style="` + microLabelStyle + `">` + esc(title) + `</h2>`)
	if len(changes) == 0 {
		b.WriteString(`<span style="font:400 12px var(--sans);color:var(--muted)">` + esc(emptyNote) + `</span>`)
	}
	for _, c := range changes {
		b.WriteString(`<div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap">`)
		b.WriteString(artifactChangeChip(c.Change))
		b.WriteString(`<span style="font:500 12.5px var(--mono);color:var(--ink)">` + esc(c.Subject) + `</span>`)
		if c.Detail != "" {
			b.WriteString(`<span style="font:400 11.5px var(--sans);color:var(--muted)">` + esc(c.Detail) + `</span>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

var artifactSevLevels = []string{"critical", "high", "medium", "low", "info"}

// normSev folds an unknown severity token to info rather than manufacturing
// urgency (mirrors signal.SeverityFor's calm fold and SeverityBadge's default),
// so a stale level never collides with critical.
func normSev(level string) string {
	for _, l := range artifactSevLevels {
		if l == level {
			return l
		}
	}
	return "info"
}

func sevTitle(level string) string {
	l := normSev(level)
	return strings.ToUpper(l[:1]) + l[1:]
}

// artifactSeverityBars renders the "open signals by severity" breakdown — one bar
// per supplied level, the ramp label, a track whose fill scales to the busiest
// level, and the count. Mirrors ReportArtifact.jsx's bar block; the fill takes the
// level's severity-dot token, never a graded colour.
func artifactSeverityBars(counts []ArtifactSeverityCount) string {
	max := 0
	for _, c := range counts {
		if c.Count > max {
			max = c.Count
		}
	}
	var b strings.Builder
	b.WriteString(`<div style="display:flex;flex-direction:column;gap:12px">`)
	b.WriteString(`<h2 data-sev="title" style="` + microLabelStyle + `">` + esc(artifactSeverityTitle) + `</h2>`)
	for _, c := range counts {
		l := normSev(c.Level)
		w := "0"
		if max > 0 {
			w = strconv.FormatFloat(float64(c.Count)/float64(max)*100, 'f', -1, 64)
		}
		b.WriteString(`<div style="display:flex;align-items:center;gap:12px">`)
		b.WriteString(`<span data-sev="` + l + `" style="width:72px;font:500 11px var(--mono);letter-spacing:0.06em;text-transform:uppercase;color:var(--secondary)">` + esc(sevTitle(l)) + `</span>`)
		b.WriteString(`<span style="flex:1;height:8px;border-radius:999px;background:var(--sunken);overflow:hidden">`)
		b.WriteString(`<span style="display:block;height:100%;width:` + w + `%;border-radius:999px;background:var(--sev-` + l + `-dot)"></span>`)
		b.WriteString(`</span>`)
		b.WriteString(`<span style="width:26px;text-align:right;font:500 12.5px var(--mono);color:var(--body)">` + strconv.Itoa(c.Count) + `</span>`)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// artifactSignalsTable renders the "new this week" signals table — a Severity
// column of SeverityBadges, then the signal, the asset (mono), and the raised
// date (mono, right). Mirrors ReportArtifact.jsx's dense table; the delivered
// document is self-contained so the table is inline-styled.
func artifactSignalsTable(sigs []ArtifactSignal) string {
	var b strings.Builder
	b.WriteString(`<div style="display:flex;flex-direction:column;gap:10px">`)
	b.WriteString(`<h2 style="` + microLabelStyle + `">` + esc(artifactSignalsTitle) + `</h2>`)
	b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
	th := `text-align:left;font:600 10px var(--mono);text-transform:uppercase;letter-spacing:0.06em;color:var(--muted);padding:0 12px 8px 0`
	b.WriteString(`<thead><tr>`)
	b.WriteString(`<th data-sev="header" style="` + th + `;width:110px">Severity</th>`)
	b.WriteString(`<th style="` + th + `">Signal</th>`)
	b.WriteString(`<th style="` + th + `;width:220px">Asset</th>`)
	b.WriteString(`<th style="` + th + `;text-align:right;width:70px">Raised</th>`)
	b.WriteString(`</tr></thead><tbody>`)
	td := `padding:8px 12px 8px 0;border-top:1px solid var(--hairline);vertical-align:middle`
	for _, s := range sigs {
		b.WriteString(`<tr>`)
		b.WriteString(`<td style="` + td + `">` + artifactSeverityBadge(s.Severity) + `</td>`)
		b.WriteString(`<td style="` + td + `;font:400 12.5px var(--sans);color:var(--ink)">` + esc(s.Signal) + `</td>`)
		b.WriteString(`<td style="` + td + `;font:400 12px var(--mono);color:var(--body)">` + esc(orDash(s.Asset)) + `</td>`)
		b.WriteString(`<td style="` + td + `;text-align:right;font:400 12px var(--mono);color:var(--muted)">` + esc(orDash(s.Raised)) + `</td>`)
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table>`)
	b.WriteString(`</div>`)
	return b.String()
}

// artifactSeverityBadge renders one severity token as a SeverityBadge (sm) —
// mirroring design-system SeverityBadge.jsx and the on-screen sevbadge: Critical
// is the only solid fill and the only pill-red and carries no dot; High → Info are
// tinted pills with a solid dot, fading to stay ordinal. Marked data-sev so the
// valence guard exempts the ramp label (the one loud voice) as it does the token
// stylesheet.
func artifactSeverityBadge(level string) string {
	l := normSev(level)
	base := `display:inline-flex;align-items:center;gap:5px;height:18px;padding:0 8px;border-radius:999px;font:600 10px var(--mono);letter-spacing:0.05em;text-transform:uppercase;white-space:nowrap`
	if l == "critical" {
		return `<span data-sev="critical" style="` + base + `;background:var(--sev-critical-fill);color:var(--sev-critical-text)">` + esc(sevTitle(l)) + `</span>`
	}
	return `<span data-sev="` + l + `" style="` + base + `;background:var(--sev-` + l + `-bg);border:1px solid var(--sev-` + l + `-border);color:var(--sev-` + l + `-fg)">` +
		`<span style="width:5px;height:5px;border-radius:999px;background:var(--sev-` + l + `-dot);flex:none"></span>` + esc(sevTitle(l)) + `</span>`
}

// artifactEmptyState is the design-system empty-state for a schedule that has
// never delivered: the fact, why, and the next action. No document is fabricated.
func artifactEmptyState() string {
	return `<div style="border:1px dashed var(--border-strong);background:var(--sunken);border-radius:var(--r-md);padding:32px;text-align:center;display:flex;flex-direction:column;gap:8px;align-items:center">` +
		`<span style="` + microLabelStyle + `">` + esc(artifactEmptyEyebrow) + `</span>` +
		`<h2 style="margin:0;font:600 15px var(--sans);color:var(--ink)">` + esc(artifactEmptyHeadline) + `</h2>` +
		`<p style="margin:0;max-width:60ch;font:400 12.5px var(--sans);color:var(--muted)">` + esc(artifactEmptyBody) + `</p>` +
		`</div>`
}

// artifactChangeChip renders a change as a drift-palette chip — the change
// vocabulary's own language (violet gain / magenta change / slate loss), never
// the severity ramp.
func artifactChangeChip(change string) string {
	fam := changeFamily(change)
	return `<span style="display:inline-flex;align-items:center;gap:5px;font:600 10.5px var(--mono);letter-spacing:0.03em;padding:2px 9px;border-radius:var(--r-full);` +
		`border:1px solid var(--drift-` + fam + `-border);background:var(--drift-` + fam + `-bg);color:var(--drift-` + fam + `-fg)">` +
		`<span style="width:5px;height:5px;border-radius:999px;background:currentColor"></span>` + esc(change) + `</span>`
}

// artifactDeliveredBadge is the ok-toned delivery receipt badge. "delivered" is a
// record of what we said, never a grade of the news — it carries no valence.
func artifactDeliveredBadge() string {
	return `<span style="display:inline-flex;align-items:center;gap:5px;font:600 10px var(--mono);text-transform:uppercase;letter-spacing:0.04em;padding:2px 8px;border-radius:var(--r-full);` +
		`border:1px solid var(--ok-border);background:var(--ok-soft);color:var(--ok)">` +
		`<span style="width:5px;height:5px;border-radius:999px;background:var(--ok)"></span>delivered</span>`
}

// artifactTag renders the delivered-format tag (a neutral pill, never a status).
func artifactTag(label string) string {
	return `<span style="display:inline-flex;align-items:center;font:500 11px var(--mono);padding:2px 9px;border-radius:8px;border:1px solid var(--hairline);background:var(--sunken);color:var(--muted)">` + esc(label) + `</span>`
}

func artifactProvenance(a Artifact) string {
	var parts []string
	if a.GeneratedAt != "" {
		parts = append(parts, "generated "+a.GeneratedAt)
	}
	if a.Version != "" {
		parts = append(parts, a.Version)
	}
	return strings.Join(parts, " · ")
}

func artifactReceipt(a Artifact) string {
	if a.Delivered == "" {
		return "not delivered"
	}
	line := "delivered " + a.Delivered
	if a.ChannelHost != "" {
		line += " · " + a.ChannelHost
	}
	return line
}

func changeFamily(change string) string {
	switch change {
	case "appeared", "returned", "revealed":
		return "gain"
	case "withdrawn", "descoped":
		return "loss"
	default: // "changed" and any unmodelled word ride the neutral change family
		return "change"
	}
}

// deltaColor maps a delta tone to its token. The tone selects a colour only; it
// is never rendered as text, so no valence word reaches the page.
func deltaColor(tone string) string {
	switch tone {
	case "good":
		return "var(--ok)"
	case "bad":
		return "var(--danger)"
	default:
		return "var(--muted)"
	}
}

func esc(s string) string { return html.EscapeString(s) }

// orDash renders an em dash where a value is absent, so an empty cell reads as
// deliberately blank rather than collapsing.
func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// artifactPeriod renders the delivery window and its number as one mono line, used
// by the console view around the document. It is exported through the web layer's
// header, not the document body, so it lives beside the other artifact helpers.
func artifactPeriod(a Artifact) string {
	var line string
	switch {
	case a.PeriodStart != "" && a.PeriodEnd != "":
		line = a.PeriodStart + " → " + a.PeriodEnd
	case a.PeriodEnd != "":
		line = a.PeriodEnd
	}
	if a.DeliveryNo > 0 {
		if line != "" {
			line += " · "
		}
		line += "delivery #" + strconv.Itoa(a.DeliveryNo)
	}
	return line
}

func ArtifactPeriod(a Artifact) string { return artifactPeriod(a) }

func group(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}
