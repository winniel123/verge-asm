package message

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
)

// PDF export of a delivered report (#345). RenderArtifactPDF is the second render
// form of the same Artifact that RenderArtifact draws as HTML — the console's
// "Download PDF" control on the report-artifact page serves its bytes. It is a
// pure-Go render (go-pdf/fpdf, no CGO, no external binary) so it runs inside the
// distroless-static web image unchanged; there is no HTML-to-PDF engine in that
// image and adopting a headless browser would mean abandoning it (ADR-0114, #345).
//
// The Artifact struct is the single source of report CONTENT; this file authors a
// second LAYOUT of that content for print, alongside RenderArtifact's markup. To
// keep the two render forms from drifting in what they say, both the drawn PDF and
// the text the valence guard reads are built from one ordered content sequence
// (artifactPDFItems): a role never draws a string the sequence does not carry, and
// the test reads the same sequence. The document grades nothing — no valence word
// in its prose — exactly as the HTML render promises; a delta's tone and a change
// word's drift family select a colour only. The severity ramp (P2.10) is the one
// loud voice: its label is drawn in the severity colour, like the on-screen
// SeverityBadge, and is exempt from the graded-prose view the valence guard reads,
// so the print form keeps the same domain guarantees.
//
// An Artifact with no delivered content (Empty) renders the design-system
// empty-state, never a fabricated document — the same rule the HTML render obeys.

// artifactPDFRole is the visual role of one drawn line: it selects the font,
// weight, size and colour, so the content sequence stays free of styling.
type artifactPDFRole int

const (
	roleTitle       artifactPDFRole = iota // the org, the document's name line
	roleMeta                               // provenance / receipt — muted mono
	roleTag                                // the delivered-format pill
	roleEyebrow                            // micro-label: a stat label or a section title
	roleStatValue                          // a KPI numeral, with an optional toned delta
	roleCaption                            // a stat caption — muted
	roleSeverityBar                        // one "open signals by severity" breakdown row
	roleSignal                             // one "new this week" signal row (severity + text)
	roleChange                             // one change row (drift chip + subject + note)
	roleNote                               // an empty-section statement — muted
	roleEmptyHead                          // the empty-state headline
	roleEmptyBody                          // the empty-state paragraph
)

// artifactPDFItem is one unit of the report's content, tagged with the role that
// will draw it. Text-bearing roles carry text; a stat value also carries its
// signed delta and tone; a change row carries the whole ArtifactChange. Tone is a
// colour selector only and is never emitted as text.
type artifactPDFItem struct {
	role      artifactPDFRole
	text      string
	delta     string
	deltaTone string
	change    ArtifactChange
	signal    ArtifactSignal
	level     string // severity token for a severity-bar row
	count     int    // count for a severity-bar row
}

func artifactPDFItems(a Artifact) []artifactPDFItem {
	var items []artifactPDFItem

	items = append(items, artifactPDFItem{role: roleTitle, text: orDash(a.Org)})
	if prov := artifactProvenance(a); prov != "" {
		items = append(items, artifactPDFItem{role: roleMeta, text: prov})
	}
	if a.Format != "" {
		items = append(items, artifactPDFItem{role: roleTag, text: a.Format})
	}

	if a.Empty() {
		items = append(items,
			artifactPDFItem{role: roleEyebrow, text: artifactEmptyEyebrow},
			artifactPDFItem{role: roleEmptyHead, text: artifactEmptyHeadline},
			artifactPDFItem{role: roleEmptyBody, text: artifactEmptyBody},
		)
	} else {
		// KPI band: label, numeral (+ toned delta), caption.
		for _, s := range a.Stats {
			items = append(items, artifactPDFItem{role: roleEyebrow, text: s.Label})
			items = append(items, artifactPDFItem{role: roleStatValue, text: orDash(s.Value), delta: s.Delta, deltaTone: s.DeltaTone})
			if s.Caption != "" {
				items = append(items, artifactPDFItem{role: roleCaption, text: s.Caption})
			}
		}
		// Open signals by severity: the eyebrow, then a breakdown row per level.
		if len(a.SeverityCounts) > 0 {
			items = append(items, artifactPDFItem{role: roleEyebrow, text: artifactSeverityTitle})
			for _, c := range a.SeverityCounts {
				items = append(items, artifactPDFItem{role: roleSeverityBar, level: normSev(c.Level), count: c.Count})
			}
		}
		// New this week: the eyebrow, then a row per signal (its severity + text).
		if len(a.Signals) > 0 {
			items = append(items, artifactPDFItem{role: roleEyebrow, text: artifactSignalsTitle})
			for _, s := range a.Signals {
				items = append(items, artifactPDFItem{role: roleSignal, signal: s})
			}
		}
		items = append(items, artifactChangeItems(artifactWithdrawnTitle, a.Withdrawn, artifactWithdrawnEmpty)...)
	}

	// Delivery receipt — the host only, never the raw URL (artifactReceipt).
	items = append(items, artifactPDFItem{role: roleMeta, text: artifactReceipt(a)})
	return items
}

func artifactChangeItems(title string, changes []ArtifactChange, emptyNote string) []artifactPDFItem {
	items := []artifactPDFItem{{role: roleEyebrow, text: title}}
	if len(changes) == 0 {
		return append(items, artifactPDFItem{role: roleNote, text: emptyNote})
	}
	for _, c := range changes {
		items = append(items, artifactPDFItem{role: roleChange, change: c})
	}
	return items
}

// artifactPDFStrings flattens the content sequence to the human-visible text it
// draws, in order — the view the valence guard and the content tests read, built
// from the same items RenderArtifactPDF draws so the two cannot disagree on what
// the document says. A change row flattens to its change word, subject and note; a
// delta is appended to its numeral. Tone is never text.
func artifactPDFStrings(a Artifact) []string {
	items := artifactPDFItems(a)
	out := make([]string, 0, len(items))
	for _, it := range items {
		switch it.role {
		case roleChange:
			line := it.change.Change + " " + it.change.Subject
			if it.change.Detail != "" {
				line += " " + it.change.Detail
			}
			out = append(out, line)
		case roleStatValue:
			line := it.text
			if it.delta != "" {
				line += " " + it.delta
			}
			out = append(out, line)
		case roleSeverityBar:
			// The count only. The level word is the severity ramp — the one loud
			// voice — drawn as colour + label like the badge, and exempt from the
			// valence prose view exactly as a delta's tone is (colour, never text).
			out = append(out, strconv.Itoa(it.count))
		case roleSignal:
			// The signal, its asset and the raised date. The severity is the ramp,
			// drawn as colour + label; it is not part of the graded prose view.
			line := it.signal.Signal
			if it.signal.Asset != "" {
				line += " " + it.signal.Asset
			}
			if it.signal.Raised != "" {
				line += " " + it.signal.Raised
			}
			out = append(out, line)
		default:
			out = append(out, it.text)
		}
	}
	return out
}

// pdfRGB is an sRGB triple drawn from the artifact light-mode token palette. The
// PDF is a print document, so it renders on the light surface; the values mirror
// artifactTokens (:root) and cmd/web's pageCSS.
type pdfRGB struct{ r, g, b int }

var (
	pdfInk    = pdfRGB{35, 31, 25}    // --ink #231f19
	pdfBody   = pdfRGB{55, 50, 44}    // --body #37322c
	pdfMuted  = pdfRGB{121, 116, 109} // --muted #79746d
	pdfOK     = pdfRGB{5, 119, 59}    // --ok #05773b
	pdfDanger = pdfRGB{172, 49, 44}   // --danger #ac312c
	pdfGain   = pdfRGB{111, 79, 161}  // --drift-gain-fg #6f4fa1
	pdfChange = pdfRGB{149, 64, 116}  // --drift-change-fg #954074
	pdfLoss   = pdfRGB{86, 100, 122}  // --drift-loss-fg #56647a
)

// pdfSevColor maps a severity token to its light-mode ramp colour — critical the
// only pill-red (its solid fill), high → info their fg tokens (colors.css :root).
// The print document draws the severity word in this colour, the severity ramp's
// one loud voice; it never grades prose. An unknown token folds to info.
func pdfSevColor(level string) pdfRGB {
	switch normSev(level) {
	case "critical":
		return pdfRGB{191, 54, 49} // --sev-critical-fill #bf3631
	case "high":
		return pdfRGB{160, 68, 0} // --sev-high-fg #a04400
	case "medium":
		return pdfRGB{141, 86, 0} // --sev-medium-fg #8d5600
	case "low":
		return pdfRGB{0, 114, 139} // --sev-low-fg #00728b
	default:
		return pdfRGB{83, 101, 121} // --sev-info-fg #536579
	}
}

// pdfDeltaColor maps a delta tone to its token colour — the same mapping deltaColor
// makes for HTML. The tone selects a colour only; it is never rendered as text.
func pdfDeltaColor(tone string) pdfRGB {
	switch tone {
	case "good":
		return pdfOK
	case "bad":
		return pdfDanger
	default:
		return pdfMuted
	}
}

// pdfChangeColor maps a change word to its drift-palette colour — the same
// families changeFamily selects for HTML (violet gain / magenta change / slate
// loss), never the severity ramp.
func pdfChangeColor(change string) pdfRGB {
	switch changeFamily(change) {
	case "gain":
		return pdfGain
	case "loss":
		return pdfLoss
	default:
		return pdfChange
	}
}

const pdfContentWidth = 210.0 - 18.0 - 18.0

// RenderArtifactPDF renders one delivered report as a PDF document — the print
// form of the same content RenderArtifact draws as HTML. It is a pure-Go render
// (no CGO, no external binary) so it runs inside the distroless-static web image.
// An Artifact with no delivered content renders the design-system empty-state
// rather than fabricating a document.
func RenderArtifactPDF(a Artifact) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	// Document metadata — the report's name and the delivered instant, so a saved
	// file is identifiable outside the console.
	title := a.Title
	if title == "" {
		title = "Report delivery"
	}
	pdf.SetTitle(title, true)
	pdf.SetCreator("verge-asm", true)

	// Standard-font glyphs are Latin-1; the translator maps the copy's em dash and
	// middle dot into it. artifactPDFItems never emits a glyph outside that set.
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()

	for _, it := range artifactPDFItems(a) {
		drawArtifactPDFItem(pdf, tr, it)
		if pdf.Err() {
			return nil, pdf.Error()
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawArtifactPDFItem draws one content item in its role's type and colour. Each
// role owns its font, size and spacing; text wraps within the printable width so a
// long subject or caption never overflows the page.
func drawArtifactPDFItem(pdf *fpdf.Fpdf, tr func(string) string, it artifactPDFItem) {
	setColor := func(c pdfRGB) { pdf.SetTextColor(c.r, c.g, c.b) }

	switch it.role {
	case roleTitle:
		pdf.Ln(2)
		pdf.SetFont("Helvetica", "B", 16)
		setColor(pdfInk)
		pdf.MultiCell(pdfContentWidth, 8, tr(it.text), "", "L", false)
		pdf.Ln(1)

	case roleTag:
		pdf.SetFont("Courier", "", 9)
		setColor(pdfMuted)
		pdf.MultiCell(pdfContentWidth, 5, tr(it.text), "", "L", false)
		pdf.Ln(1)

	case roleMeta:
		pdf.SetFont("Courier", "", 9)
		setColor(pdfMuted)
		pdf.MultiCell(pdfContentWidth, 5, tr(it.text), "", "L", false)
		pdf.Ln(2)

	case roleEyebrow:
		pdf.Ln(3)
		pdf.SetFont("Courier", "B", 9)
		setColor(pdfMuted)
		pdf.MultiCell(pdfContentWidth, 5, tr(strings.ToUpper(it.text)), "", "L", false)
		pdf.Ln(0.5)

	case roleStatValue:
		pdf.SetFont("Courier", "B", 22)
		setColor(pdfInk)
		// The numeral sits on its own line; a toned delta trails it in a smaller
		// mono, its colour the only thing the tone selects.
		w := pdf.GetStringWidth(tr(it.text))
		pdf.Cell(w+2, 9, tr(it.text))
		if it.delta != "" {
			pdf.SetFont("Courier", "B", 11)
			setColor(pdfDeltaColor(it.deltaTone))
			pdf.Cell(0, 9, tr(it.delta))
		}
		pdf.Ln(9)

	case roleCaption:
		pdf.SetFont("Helvetica", "", 9)
		setColor(pdfMuted)
		pdf.MultiCell(pdfContentWidth, 5, tr(it.text), "", "L", false)

	case roleSeverityBar:
		// The ramp label in its severity colour, then the count in ink — the
		// severity scale, the one loud voice; a colour ramp, never a valence grade.
		pdf.SetFont("Courier", "B", 9)
		setColor(pdfSevColor(it.level))
		label := tr(strings.ToUpper(sevTitle(it.level)))
		pdf.Cell(24, 6, label)
		pdf.SetFont("Courier", "B", 10)
		setColor(pdfInk)
		pdf.Cell(0, 6, tr(strconv.Itoa(it.count)))
		pdf.Ln(6)

	case roleSignal:
		// The severity label in its ramp colour, then the signal in ink, the asset
		// in muted mono, and the raised date — the badge column, ported to print.
		pdf.SetFont("Courier", "B", 9)
		setColor(pdfSevColor(it.signal.Severity))
		sev := tr(strings.ToUpper(sevTitle(it.signal.Severity)))
		pdf.Cell(pdf.GetStringWidth(sev)+3, 6, sev)
		pdf.SetFont("Helvetica", "", 10)
		setColor(pdfInk)
		sig := tr(it.signal.Signal)
		pdf.Cell(pdf.GetStringWidth(sig)+3, 6, sig)
		if it.signal.Asset != "" {
			pdf.SetFont("Courier", "", 9)
			setColor(pdfMuted)
			pdf.Cell(0, 6, tr(it.signal.Asset))
		}
		pdf.Ln(6)

	case roleChange:
		// The change word in its drift-palette colour, then the subject in ink and
		// the note in muted grey — the chip's language, never a severity level.
		pdf.SetFont("Courier", "B", 10)
		setColor(pdfChangeColor(it.change.Change))
		word := tr(it.change.Change)
		pdf.Cell(pdf.GetStringWidth(word)+3, 6, word)
		pdf.SetFont("Courier", "", 10)
		setColor(pdfInk)
		subj := tr(it.change.Subject)
		pdf.Cell(pdf.GetStringWidth(subj)+3, 6, subj)
		if it.change.Detail != "" {
			pdf.SetFont("Helvetica", "", 9)
			setColor(pdfMuted)
			pdf.Cell(0, 6, tr(it.change.Detail))
		}
		pdf.Ln(6)

	case roleNote:
		pdf.SetFont("Helvetica", "", 9)
		setColor(pdfMuted)
		pdf.MultiCell(pdfContentWidth, 5, tr(it.text), "", "L", false)

	case roleEmptyHead:
		pdf.Ln(2)
		pdf.SetFont("Helvetica", "B", 13)
		setColor(pdfInk)
		pdf.MultiCell(pdfContentWidth, 7, tr(it.text), "", "L", false)

	case roleEmptyBody:
		pdf.Ln(1)
		pdf.SetFont("Helvetica", "", 10)
		setColor(pdfBody)
		pdf.MultiCell(pdfContentWidth, 5, tr(it.text), "", "L", false)
		pdf.Ln(2)
	}
}
