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

// These fields are the design tmpl's holes verbatim; the tags mirror fixtures.json.

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

// The four tmpls use only builtin actions, so no FuncMap is needed here.

var artifactTmpl = template.Must(template.New("artifact").ParseFS(designfs.FS,
	"templates/reportartifact.tmpl",
	"templates/reports.tmpl",
	"templates/signals.tmpl",
	"templates/drift.tmpl",
))

// The on-screen page's chrome already inlines these tokens, so only a standalone shell reads this.

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

func renderArtifactDoc(doc ArtifactDoc) (template.HTML, error) {
	var buf bytes.Buffer
	if err := artifactTmpl.ExecuteTemplate(&buf, "artifactdoc", doc); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil // #nosec G203 -- output of the design-owned "artifactdoc" template, not user input
}

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
		// A receipt records what we said and never grades the news, so ok is not a valence (ADR-0064).
		doc.DeliveryState = &ArtifactDocDeliveryState{Label: "delivered", Tone: "ok"}
	}
	return doc
}

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
