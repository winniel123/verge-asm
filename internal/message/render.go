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

// ContainsValence reports whether s carries any refused valence word, matched on
// word boundaries and case-insensitively. It is the guard behind the model's
// promise that no rendered message copy grades the news.
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

// DeclaredInput returns a Message whose mover is the operator's own declared
// input — a Source they toggled, a zone file they re-supplied. It links to the
// Source the rule reads, and rides the coverage class (we changed what we are
// told). sourceKey is the Source identity the row links to.
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

// flagshipHeadline states the internet leg reaching, with the count of facets
// that opened beneath as its factor. `reached` is a Reach value, not a valence
// word — the sentence names what moved and grades nothing.
func flagshipHeadline(serviceKey string, census Census) string {
	return fmt.Sprintf("%s reached from the internet · %s opened beneath it",
		serviceKey, plural(census.Len(), "facet", "facets"))
}

// membershipHeadline states a root entering the estate, with the count of
// timelines that opened beneath as its factor. `appeared` / `returned` /
// `revealed` are the Transition's own words and carry no valence.
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
// Two domain holds against the reference mock (ADR-0110's "the domain term always
// wins"): the artifact carries NO severity ramp — a signal is a census member,
// not a scored one (ADR-0024) — so the mock's "open signals by severity" bars and
// its severity-scored "new this week" table become domain-honest KPI scalars and
// change rows; and change rides its own drift vocabulary and palette
// (appeared / revealed / withdrawn), never the severity ramp. Signals are
// withdrawn by the world, never resolved by an operator, so the mock's
// mean-time-to-resolve KPI is not modelled.
type Artifact struct {
	// Title is the report's name (e.g. "Weekly exposure summary"); Org is the
	// account the delivery was cut for.
	Title string
	Org   string
	// PeriodStart / PeriodEnd bound the delivery window (ISO dates); DeliveryNo is
	// the delivery's monotonic number within its schedule.
	PeriodStart string
	PeriodEnd   string
	DeliveryNo  int
	// GeneratedAt is the instant the artifact was cut (ISO); Version names the
	// release that cut it; Format is the delivered form ("pdf").
	GeneratedAt string
	Version     string
	Format      string
	// Stats is the KPI band — honest current-state / period scalars. No resolve
	// metric appears: signals are withdrawn by the world, never resolved.
	Stats []ArtifactStat
	// Appeared is what entered view this period ("new this week"); Withdrawn is
	// what the world withdrew. Both ride the drift vocabulary, never severity.
	Appeared  []ArtifactChange
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

// Empty reports whether the artifact carries no delivered content — no KPI band
// and no change rows. A view of a schedule that has never delivered renders the
// design-system empty-state rather than a fabricated document (ADR-0110: a screen
// with no backing data ships an empty-state, never invented data).
func (a Artifact) Empty() bool {
	return len(a.Stats) == 0 && len(a.Appeared) == 0 && len(a.Withdrawn) == 0
}

// artifactTokens is the self-contained slice of the design system the rendered
// artifact needs to stand alone as a PDF / email body, with no console stylesheet
// in scope. The custom-property names and values mirror cmd/web's pageCSS and
// design-system/tokens; dark ships two ways (prefers-color-scheme default and the
// explicit data-theme override) so the delivered document is legible in either.
const artifactTokens = `<style>
.vg-artifact{
  --surface:#ffffff; --sunken:#f2f0ec; --hairline:#e2dfdb; --border-strong:#c7c3be;
  --ink:#231f19; --body:#37322c; --muted:#79746d;
  --ok:#05773b; --ok-soft:#e1fae7; --ok-border:#bfebc9;
  --danger:#ac312c;
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
  --ink:#eae7e4; --body:#d9d5d1; --muted:#898581;
  --ok:#57c07f; --ok-soft:#17281c; --ok-border:#1f4029;
  --danger:#f08c82;
  --drift-gain-bg:#241b33; --drift-gain-border:#423658; --drift-gain-fg:#c0abe9;
  --drift-change-bg:#2f1725; --drift-change-border:#533044; --drift-change-fg:#e2a0c5;
  --drift-loss-bg:#1d2127; --drift-loss-border:#383d47; --drift-loss-fg:#aeb8c9;
  --shadow-sm:0 1px 2px rgba(0,0,0,0.35),0 2px 8px rgba(0,0,0,0.25);
}}
:root[data-theme="dark"] .vg-artifact{
  --surface:#1e1b17; --sunken:#191613; --hairline:#383530; --border-strong:#514c46;
  --ink:#eae7e4; --body:#d9d5d1; --muted:#898581;
  --ok:#57c07f; --ok-soft:#17281c; --ok-border:#1f4029;
  --danger:#f08c82;
  --drift-gain-bg:#241b33; --drift-gain-border:#423658; --drift-gain-fg:#c0abe9;
  --drift-change-bg:#2f1725; --drift-change-border:#533044; --drift-change-fg:#e2a0c5;
  --drift-loss-bg:#1d2127; --drift-loss-border:#383d47; --drift-loss-fg:#aeb8c9;
  --shadow-sm:0 1px 2px rgba(0,0,0,0.35),0 2px 8px rgba(0,0,0,0.25);
}
</style>`

// microLabelStyle is the mono eyebrow the system uses as a section header — the
// signature motif, reused for every h2 in the artifact document.
const microLabelStyle = `margin:0;font:500 11px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)`

// The delivered document's fixed copy — the section titles, the empty-section
// notes stating that nothing moved, and the empty-state block. Both render forms
// read these constants (RenderArtifact draws them as HTML, RenderArtifactPDF as
// print) so the two layouts cannot drift in WHAT the document says, only in how it
// looks (ADR-0114: one canonical content model, two layout forms).
const (
	artifactAppearedTitle  = "New this week"
	artifactWithdrawnTitle = "Withdrawn by the world"
	artifactAppearedEmpty  = "Nothing entered view in this period."
	artifactWithdrawnEmpty = "The world withdrew nothing in this period."
	artifactEmptyEyebrow   = "Nothing delivered"
	artifactEmptyHeadline  = "No report has been delivered yet"
	artifactEmptyBody      = "A delivered report is rendered here once report scheduling lands and a schedule runs. Until then there is no delivery to view."
)

// RenderArtifact renders one delivered report as the document itself — the card
// design-system/examples/console/ReportArtifact.jsx fixes — as self-contained,
// inline-styled HTML that stands alone as a PDF / email body and re-embeds in the
// console view. An artifact with no delivered content renders the design-system
// empty-state rather than fabricating a document.
func RenderArtifact(a Artifact) template.HTML {
	var b strings.Builder
	b.WriteString(artifactTokens)
	b.WriteString(`<section class="vg-artifact" data-artifact style="max-width:800px;width:100%;margin:0 auto;box-sizing:border-box;background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:28px;color:var(--body)">`)
	b.WriteString(`<div style="display:flex;flex-direction:column;gap:24px">`)

	// Identity row: org, provenance, and the delivered format tag.
	b.WriteString(`<div style="display:flex;align-items:baseline;gap:12px;padding-bottom:16px;border-bottom:1px solid var(--hairline);flex-wrap:wrap">`)
	b.WriteString(`<span style="font:600 15px var(--sans);color:var(--ink)">` + esc(orDash(a.Org)) + `</span>`)
	if prov := artifactProvenance(a); prov != "" {
		b.WriteString(`<span style="font:400 11.5px var(--mono);color:var(--muted)">` + esc(prov) + `</span>`)
	}
	if a.Format != "" {
		b.WriteString(`<span style="margin-left:auto">` + artifactTag(a.Format) + `</span>`)
	}
	b.WriteString(`</div>`)

	if a.Empty() {
		b.WriteString(artifactEmptyState())
	} else {
		b.WriteString(artifactStatBand(a.Stats))
		b.WriteString(artifactChangeSection(artifactAppearedTitle, a.Appeared, artifactAppearedEmpty))
		b.WriteString(artifactChangeSection(artifactWithdrawnTitle, a.Withdrawn, artifactWithdrawnEmpty))
	}

	// Delivery receipt — the host only, never the raw URL.
	b.WriteString(`<div style="display:flex;align-items:center;gap:10px;padding-top:16px;border-top:1px solid var(--hairline);flex-wrap:wrap">`)
	b.WriteString(`<span style="font:400 11.5px var(--mono);color:var(--muted)">` + esc(artifactReceipt(a)) + `</span>`)
	if a.Delivered != "" {
		b.WriteString(`<span style="margin-left:auto">` + artifactDeliveredBadge() + `</span>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`</div></section>`)
	return template.HTML(b.String())
}

// artifactStatBand renders the KPI summary band — three honest scalars, each a
// mono numeral over a mono eyebrow, with an optional toned delta and a caption.
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

// artifactProvenance is the mono provenance line beside the org: when the artifact
// was cut and which release cut it, each shown only where present.
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

// artifactReceipt is the footer's delivery line — the instant and the destination
// host where present, or a plain statement that nothing was delivered.
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

// changeFamily maps a change word to its drift-palette family: gain (violet) for
// what entered view, change (magenta) for what moved, loss (slate) for what left.
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

// esc escapes a dynamic string for safe interpolation into the artifact markup.
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

// ArtifactPeriod is the exported delivery-window line for the console header.
func ArtifactPeriod(a Artifact) string { return artifactPeriod(a) }

// group inserts thousands separators into a non-negative integer.
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
