package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/docs/guides"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
)

type searchStore interface {
	ListCurrentNameSubjects(ctx context.Context, arg db.ListCurrentNameSubjectsParams) ([]db.ListCurrentNameSubjectsRow, error)
	ListDispatchProgress(ctx context.Context, limit int32) ([]db.ListDispatchProgressRow, error)
	ListSpansForSubject(ctx context.Context, arg db.ListSpansForSubjectParams) ([]db.ListSpansForSubjectRow, error)
}

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/search.tmpl"))

type hiSeg struct {
	Text string
	Hit  bool
}

func searchSegs(text, q string) []hiSeg {
	// Lowercasing can change byte length, and mixing the two offset spaces panicked /search (#340).
	if q == "" {
		return []hiSeg{{Text: text}}
	}
	i := strings.Index(strings.ToLower(text), strings.ToLower(q))
	if i < 0 {
		return []hiSeg{{Text: text}}
	}
	end := i + len(strings.ToLower(q))
	if i > len(text) || end > len(text) {
		return []hiSeg{{Text: text}}
	}
	segs := make([]hiSeg, 0, 3)
	if i > 0 {
		segs = append(segs, hiSeg{Text: text[:i]})
	}
	segs = append(segs, hiSeg{Text: text[i:end], Hit: true})
	if end < len(text) {
		segs = append(segs, hiSeg{Text: text[end:]})
	}
	return segs
}

type searchAsset struct {
	NameSegs []hiSeg
	Type     string
	Severity string
	SevLabel string
	Href     string
}

type searchSignal struct {
	RuleSegs    []hiSeg
	SubjectSegs []hiSeg
	Severity    string
	SevLabel    string
	Href        string
}

// A guide has no in-console route, so this row carries no Href.

type searchDoc struct {
	TitleSegs []hiSeg
	SnipSegs  []hiSeg
}

type searchBatch struct {
	Status    string
	LabelSegs []hiSeg
	Href      string
}

type guideDoc struct {
	title string
	desc  string
}

var guideIndex = loadGuideIndex()

func loadGuideIndex() []guideDoc {
	entries, err := guides.FS.ReadDir(".")
	if err != nil {
		log.Printf("web: search: read guides dir: %v", err)
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	out := make([]guideDoc, 0, len(names))
	for _, name := range names {
		b, err := guides.FS.ReadFile(name)
		if err != nil {
			log.Printf("web: search: read guide %s: %v", name, err)
			continue
		}
		title, desc := guideFrontMatter(string(b))
		if title == "" {
			continue
		}
		out = append(out, guideDoc{title: title, desc: desc})
	}
	return out
}

func guideFrontMatter(content string) (title, desc string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "title":
			title = strings.TrimSpace(val)
		case "description":
			desc = strings.TrimSpace(val)
		}
	}
	return title, desc
}

func searchMatch(text, q string) bool {
	return q == "" || strings.Contains(strings.ToLower(text), strings.ToLower(q))
}

type searchFailedReads struct {
	Signals bool
	Assets  bool
	Batches bool
}

func (f searchFailedReads) any() bool { return f.Signals || f.Assets || f.Batches }

func (f searchFailedReads) note() string {
	var groups []string
	if f.Signals {
		groups = append(groups, "signal")
	}
	if f.Assets {
		groups = append(groups, "asset")
	}
	if f.Batches {
		groups = append(groups, "batch")
	}
	switch len(groups) {
	case 0:
		return ""
	case 1:
		return "the " + groups[0] + " read did not resolve"
	}
	last := len(groups) - 1
	return "the " + strings.Join(groups[:last], ", ") + " and " + groups[last] + " reads did not resolve"
}

func searchRenderMap(acct db.Account, q string, total int, failed searchFailedReads, assets []searchAsset, signals []searchSignal, batches []searchBatch, docs []searchDoc) map[string]any {
	rest := map[string]any{
		"Query":         q,
		"Assets":        assets,
		"Signals":       signals,
		"Batches":       batches,
		"Docs":          docs,
		"SignalsFailed": failed.Signals,
		"AssetsFailed":  failed.Assets,
		"BatchesFailed": failed.Batches,
		"TotalWithheld": failed.any(),
		"WithheldNote":  failed.note(),
	}
	if !failed.any() {
		// A total missing an unread group is a figure the read never produced (ADR-0168 §2).
		rest["Total"] = total
	}
	return pageData(acct, "Search results", "", rest)
}

func (s *server) searchPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "search", s.searchFixtureData(acct, r))
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	ctx := r.Context()

	var signals []searchSignal
	assetSev := map[string]string{}
	openSignals := 0
	var failed searchFailedReads
	if corpus, err := s.buildSignalCorpus(r); err != nil {
		log.Printf("web: search: build signal corpus: %v", err)
		failed.Signals = true
	} else {
		for _, c := range signal.EvaluateCorpus(corpus) {
			openSignals += len(c.Fired)
			sev, _ := signal.SeverityFor(c.Rule)
			for _, m := range c.Fired {
				if cur, ok := assetSev[m.Subject]; !ok || signal.Severity(sev.String()).Rank() < signal.Severity(cur).Rank() {
					assetSev[m.Subject] = sev.String()
				}
				if !searchMatch(c.Rule, q) && !searchMatch(m.Subject, q) {
					continue
				}
				signals = append(signals, searchSignal{
					RuleSegs:    searchSegs(c.Rule, q),
					SubjectSegs: searchSegs(m.Subject, q),
					Severity:    sev.String(),
					SevLabel:    sevLabel(sev.String()),
					Href:        "/signals",
				})
			}
		}
	}

	var assets []searchAsset
	exactHit := false
	if rows, err := s.searchStore.ListCurrentNameSubjects(ctx, db.ListCurrentNameSubjectsParams{
		Search: q, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); err != nil {
		log.Printf("web: search: list name subjects: %v", err)
		failed.Assets = true
	} else {
		for _, row := range rows {
			exactHit = exactHit || row.SubjectKey == q
			sev := assetSev[row.SubjectKey]
			assets = append(assets, searchAsset{
				NameSegs: searchSegs(row.SubjectKey, q),
				Type:     "name",
				Severity: sev,
				SevLabel: sevLabel(sev),
				Href:     "/asset/" + url.PathEscape(row.SubjectKey),
			})
		}
	}
	if q != "" && !exactHit {
		// A withdrawn Name is reached by its exact key alone, never a substring (ADR-0072).
		if rows, err := s.searchStore.ListSpansForSubject(ctx, db.ListSpansForSubjectParams{
			SubjectKind: "name", SubjectKey: q,
		}); err != nil {
			log.Printf("web: search: list spans for %q: %v", q, err)
			failed.Assets = true
		} else if allSpansClosed(rows) {
			sev := assetSev[q]
			assets = append(assets, searchAsset{
				NameSegs: searchSegs(q, q),
				Type:     "withdrawn name",
				Severity: sev,
				SevLabel: sevLabel(sev),
				Href:     "/asset/" + url.PathEscape(q),
			})
		}
	}

	var batches []searchBatch
	if rows, err := s.searchStore.ListDispatchProgress(ctx, scansHistoryLimit); err != nil {
		log.Printf("web: search: list dispatch progress: %v", err)
		failed.Batches = true
	} else {
		for _, row := range rows {
			dv := toDispatchView(row)
			if !searchMatch(dv.DispatchedAt, q) && !searchMatch(dv.ScanKind, q) {
				continue
			}
			status := "complete"
			switch {
			case dv.Active:
				status = "running"
			case dv.Dead > 0:
				status = "failed"
			}
			batches = append(batches, searchBatch{
				Status:    status,
				LabelSegs: searchSegs(dv.DispatchedAt, q),
				Href:      "/run/" + strconv.FormatInt(dv.ID, 10),
			})
		}
	}

	var docsHits []searchDoc
	for _, g := range guideIndex {
		if !searchMatch(g.title, q) && !searchMatch(g.desc, q) {
			continue
		}
		docsHits = append(docsHits, searchDoc{
			TitleSegs: searchSegs(g.title, q),
			SnipSegs:  searchSegs(g.desc, q),
		})
	}

	total := len(assets) + len(signals) + len(batches) + len(docsHits)

	data := searchRenderMap(acct, q, total, failed, assets, signals, batches, docsHits)
	if openSignals > 0 {
		data["SignalCount"] = openSignals
	}
	s.render(w, r, "search", data)
}
