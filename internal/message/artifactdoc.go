package message

import (
	"bytes"
	"html/template"
	"io/fs"
	"math"
	"sort"
	"strings"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// ArtifactDoc is the view model for the design-owned "artifactdoc" define
// (design-system/templates/reportartifact.tmpl). SPEC-CHANGE #23g moved the delivered
// document's body OUT of repo-authored markup and INTO that frozen define, so one
// markup renders across the three shells the report reaches: the on-screen
// /reports/delivery page (which passes this struct straight to the page tmpl's `.Doc`
// hole), the delivered email (RenderArtifact executes "artifactdoc" with it), and the
// PDF print form. Its exported fields are the tmpl's holes verbatim, and its JSON tags
// mirror the fixtures.json → reportartifact.doc slice so a fixture can unmarshal into it
// unchanged. BuildArtifactDoc derives one from the domain Artifact.
type ArtifactDoc struct {
	Empty         bool                      `json:"empty"`
	Org           string                    `json:"org"`
	Generated     string                    `json:"generated"`
	Version       string                    `json:"version"`
	Format        string                    `json:"format"`
	Stats         []ArtifactDocStat         `json:"stats"`
	SevBars       []ArtifactDocSevBar       `json:"sev_bars"`
	NewRows       []ArtifactDocNewRow       `json:"new_rows"`
	Withdrawn     []ArtifactDocWithdrawn    `json:"withdrawn"`
	DeliveredAt   string                    `json:"delivered_at"`
	DeliveredTo   string                    `json:"delivered_to"`
	DeliveryState *ArtifactDocDeliveryState `json:"delivery_state"`
}

type ArtifactDocStat struct {
	Label   string           `json:"label"`
	Value   string           `json:"value"`
	Delta   ArtifactDocDelta `json:"delta"`
	Caption string           `json:"caption"`
}

type ArtifactDocDelta struct {
	Has  bool   `json:"has"`
	Text string `json:"text"`
	Dir  string `json:"dir"`
	Tone string `json:"tone"`
}

type ArtifactDocSevBar struct {
	Sev   string `json:"sev"`
	Label string `json:"label"`
	Pct   int    `json:"pct"`
	Count int    `json:"count"`
}

type ArtifactDocNewRow struct {
	Severity string `json:"severity"`
	SevLabel string `json:"sev_label"`
	Signal   string `json:"signal"`
	Asset    string `json:"asset"`
	Seen     string `json:"seen"`
}

type ArtifactDocWithdrawn struct {
	Text   string `json:"text"`
	Reason string `json:"reason"`
}

type ArtifactDocDeliveryState struct {
	Label string `json:"label"`
	Tone  string `json:"tone"`
}

// artifactTmpl parses the design-owned "artifactdoc" define and every partial it calls
// — "deltachip" (reports.tmpl), "sevbadge" (signals.tmpl) and "changeglyph"
// (drift.tmpl) — from the embedded design package, once at init. Executing "artifactdoc"
// against this set resolves those partials exactly as cmd/web's shared template set does
// on screen. The four tmpls use only builtin template actions, so no FuncMap is needed.
var artifactTmpl = template.Must(template.New("artifact").ParseFS(designfs.FS,
	"templates/reportartifact.tmpl",
	"templates/reports.tmpl",
	"templates/signals.tmpl",
	"templates/drift.tmpl",
))

// artifactDocTokens is the design-owned CSS-token vocabulary (design-system/tokens/*.css)
// wrapped in a <style> block, concatenated deterministically (sorted filename order) so a
// standalone shell — the delivered email, a print form — resolves the artifactdoc's inline
// var(--…) styles with no console stylesheet in scope. On the /reports/delivery page the
// chrome already inlines these tokens (DesignTokens), so the page does not use this.
var artifactDocTokens = loadArtifactDocTokens()

func loadArtifactDocTokens() string {
	names, err := fs.Glob(designfs.FS, "tokens/*.css")
	if err != nil {
		return ""
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("<style>")
	for _, name := range names {
		data, err := fs.ReadFile(designfs.FS, name)
		if err != nil {
			continue
		}
		b.Write(data)
		b.WriteString("\n")
	}
	b.WriteString("</style>")
	return b.String()
}

// renderArtifactDoc executes the design-owned "artifactdoc" define with doc, returning
// the delivered document's markup. It is the single markup the on-screen page, the email
// and the PDF-HTML shells all render (SPEC-CHANGE #23g).
func renderArtifactDoc(doc ArtifactDoc) (template.HTML, error) {
	var buf bytes.Buffer
	if err := artifactTmpl.ExecuteTemplate(&buf, "artifactdoc", doc); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil // #nosec G203 -- output of the design-owned "artifactdoc" template, not user input
}

// BuildArtifactDoc derives the design-owned document view model from the domain Artifact
// — the re-point SPEC-CHANGE #23g requires so RenderArtifact and the on-screen page both
// feed the frozen "artifactdoc" define the same recomputed data. Percentages for the
// severity bars scale to the busiest level (mirroring the old on-screen bar math);
// delta direction is read from the pre-signed delta text; the delivery pill stands only
// where the run actually left the instance.
func BuildArtifactDoc(a Artifact) ArtifactDoc {
	doc := ArtifactDoc{
		Empty:       a.Empty(),
		Org:         a.Org,
		Generated:   a.GeneratedAt,
		Version:     a.Version,
		Format:      a.Format,
		DeliveredAt: a.Delivered,
		DeliveredTo: a.ChannelHost,
	}
	for _, s := range a.Stats {
		doc.Stats = append(doc.Stats, ArtifactDocStat{
			Label:   s.Label,
			Value:   s.Value,
			Caption: s.Caption,
			Delta: ArtifactDocDelta{
				Has:  strings.TrimSpace(s.Delta) != "",
				Text: s.Delta,
				Dir:  deltaDir(s.Delta),
				Tone: s.DeltaTone,
			},
		})
	}
	max := 0
	for _, c := range a.SeverityCounts {
		if c.Count > max {
			max = c.Count
		}
	}
	for _, c := range a.SeverityCounts {
		level := normSev(c.Level)
		pct := 0
		if max > 0 {
			pct = int(math.Round(float64(c.Count) / float64(max) * 100))
		}
		doc.SevBars = append(doc.SevBars, ArtifactDocSevBar{
			Sev: level, Label: level, Pct: pct, Count: c.Count,
		})
	}
	for _, sig := range a.Signals {
		level := normSev(sig.Severity)
		doc.NewRows = append(doc.NewRows, ArtifactDocNewRow{
			Severity: level,
			SevLabel: sevTitle(level),
			Signal:   sig.Signal,
			Asset:    sig.Asset,
			Seen:     sig.Raised,
		})
	}
	for _, wd := range a.Withdrawn {
		doc.Withdrawn = append(doc.Withdrawn, ArtifactDocWithdrawn{
			Text:   wd.Subject,
			Reason: wd.Detail,
		})
	}
	if a.Delivered != "" {
		doc.DeliveryState = &ArtifactDocDeliveryState{Label: "delivered", Tone: "ok"}
	}
	return doc
}

// deltaDir reads a pre-signed delta's direction from its leading sign — a leading plus is
// up, a leading ASCII hyphen or a real minus (−, U+2212) is down; anything else has no
// direction, so the deltachip draws no arrow.
func deltaDir(delta string) string {
	d := strings.TrimSpace(delta)
	switch {
	case strings.HasPrefix(d, "+"):
		return "up"
	case strings.HasPrefix(d, "-"), strings.HasPrefix(d, "−"):
		return "down"
	default:
		return ""
	}
}
