package message

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
)

type artifactPDFRole int

const (
	roleTitle artifactPDFRole = iota
	roleMeta
	roleTag
	roleEyebrow
	roleStatValue
	roleCaption
	roleSeverityBar
	roleSignal
	roleChange
	roleNote
	roleEmptyHead
	roleEmptyBody
)

type artifactPDFItem struct {
	role      artifactPDFRole
	text      string
	delta     string
	deltaTone string
	change    ArtifactChange
	signal    ArtifactSignal
	level     string
	count     int
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
		// No resolve metric here: the world withdraws a signal, an operator never resolves it.
		for _, s := range a.Stats {
			items = append(items, artifactPDFItem{role: roleEyebrow, text: s.Label})
			items = append(items, artifactPDFItem{role: roleStatValue, text: orDash(s.Value), delta: s.Delta, deltaTone: s.DeltaTone})
			if s.Caption != "" {
				items = append(items, artifactPDFItem{role: roleCaption, text: s.Caption})
			}
		}
		if len(a.SeverityCounts) > 0 {
			items = append(items, artifactPDFItem{role: roleEyebrow, text: artifactSeverityTitle})
			for _, c := range a.SeverityCounts {
				items = append(items, artifactPDFItem{role: roleSeverityBar, level: normSev(c.Level), count: c.Count})
			}
		}
		if len(a.Signals) > 0 {
			items = append(items, artifactPDFItem{role: roleEyebrow, text: artifactSignalsTitle})
			for _, s := range a.Signals {
				items = append(items, artifactPDFItem{role: roleSignal, signal: s})
			}
		}
		items = append(items, artifactChangeItems(artifactWithdrawnTitle, a.Withdrawn, artifactWithdrawnEmpty)...)
	}

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

func artifactPDFStrings(a Artifact) []string {
	// The drawn document and the guarded text come from one sequence, so the two cannot disagree.
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
			out = append(out, strconv.Itoa(it.count))
		case roleSignal:
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

// A print document renders on the light surface only, so the palette mirrors the light tokens.

type pdfRGB struct{ r, g, b int }

var (
	pdfInk    = pdfRGB{35, 31, 25}
	pdfBody   = pdfRGB{55, 50, 44}
	pdfMuted  = pdfRGB{121, 116, 109}
	pdfOK     = pdfRGB{5, 119, 59}
	pdfDanger = pdfRGB{172, 49, 44}
	pdfGain   = pdfRGB{111, 79, 161}
	pdfChange = pdfRGB{149, 64, 116}
	pdfLoss   = pdfRGB{86, 100, 122}
)

func pdfSevColor(level string) pdfRGB {
	switch normSev(level) {
	case "critical":
		return pdfRGB{191, 54, 49}
	case "high":
		return pdfRGB{160, 68, 0}
	case "medium":
		return pdfRGB{141, 86, 0}
	case "low":
		return pdfRGB{0, 114, 139}
	default:
		return pdfRGB{83, 101, 121}
	}
}

func pdfDeltaColor(tone string) pdfRGB {
	// The tone selects a colour only and is never text, so no valence word reaches the page.
	switch tone {
	case "good":
		return pdfOK
	case "bad":
		return pdfDanger
	default:
		return pdfMuted
	}
}

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

func RenderArtifactPDF(a Artifact) ([]byte, error) {
	// Pure Go, because a headless browser would mean abandoning the distroless-static image (#345).
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	title := a.Title
	if title == "" {
		title = "Report delivery"
	}
	pdf.SetTitle(title, true)
	pdf.SetCreator("verge-asm", true)

	// Standard-font glyphs are Latin-1, and the copy never emits one outside that set.
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
		pdf.SetFont("Courier", "B", 9)
		setColor(pdfSevColor(it.level))
		label := tr(strings.ToUpper(sevTitle(it.level)))
		pdf.Cell(24, 6, label)
		pdf.SetFont("Courier", "B", 10)
		setColor(pdfInk)
		pdf.Cell(0, 6, tr(strconv.Itoa(it.count)))
		pdf.Ln(6)

	case roleSignal:
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
