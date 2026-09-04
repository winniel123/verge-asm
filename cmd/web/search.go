package main

import (
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

// The Search results screen (#303, T8, ADR-0110) byte-serves from the design-owned
// design-system/templates/search.tmpl (package v3.12.0, screen 19; the repo-authored
// templates_search.go is retired). The tmpl defines "search" + "hisegs" and reuses
// the landed "sevbadge" (signals.tmpl) — one parse set, so both parse into the shared
// tmpl below and sevbadge resolves at execute time. This file supplies the handler
// data shaped to the tmpl's declared holes and, under VERGE_DEV, the pinned
// fixtures.json search slice for the pixel-parity harness.
//
// The tmpl's holes (search.tmpl header): .Query .Total
//   .Assets[{Href, NameSegs, Type, Severity, SevLabel}]
//   .Signals[{Href, Severity, SevLabel, RuleSegs, SubjectSegs}]
//   .Batches[{Href, Status, LabelSegs}]
//   .Docs[{TitleSegs, SnipSegs}]
//
// SPEC-CHANGE #25 (ruled), wired here:
//   - #25a matched-term highlighting: every matched text field is a segment list
//     [{Text,Hit}] rendered by "hisegs". The handler splits each field on the FIRST
//     case-insensitive occurrence of the query (searchSegs) — text before the hit is
//     one un-hit seg, the matched run is one hit seg, text after is one un-hit seg; a
//     non-matching field is a single un-hit seg.
//   - #25b asset severity: an asset row carries a nullable .Severity/.SevLabel and
//     renders the landed sevbadge after the type tag. Severity is the most urgent
//     (lowest-rank) severity across the signals firing on that subject — the same
//     rollup AssetDetail's header draws (assetHeaderSeverity), read from the live
//     corpus, never fabricated. A subject no signal fires on carries none.
//   - #25c mono input: the leading input icon drops (tmpl-owned); the handler injects
//     no icon and the focus ring is the tmpl's accent treatment.
//
// The sample data is real reads of the tmpl's shape: Assets are the current Name
// subjects (ListCurrentNameSubjects, its server-side Search filtering the query),
// Signals the fired members of the live signal corpus, Batches the recent Dispatches
// off the Operational queue corpus, and Documentation the operator guides embedded
// from docs/guides/. Every navigable row links to the route that already exists. The
// app is server-rendered with no client filter machinery, so the query rides the ?q=
// string and the input is a GET form; the segmentation is computed server-side.

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/search.tmpl"))

type hiSeg struct {
	Text string
	Hit  bool
}

// searchSegs splits text on the FIRST case-insensitive occurrence of q into the
// [{Text,Hit}] segment list "hisegs" renders (#25a): the text before the hit (omitted
// when empty), the matched run (its original case preserved), and the text after
// (omitted when empty). An empty query or a non-match yields a single un-hit seg.
//
// The match is located in the LOWERED text, so the run length comes from the lowered
// query, never len(q): strings.ToLower can change a value's byte length (U+212A KELVIN
// SIGN → "k"; U+023A → the longer U+2C65), and mixing the two offset spaces once
// sliced the original out of range and panicked the whole /search page (#340). The
// clamp falls back to one un-hit seg rather than slice out of bounds.
func searchSegs(text, q string) []hiSeg {
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

// searchAsset is one Name subject in the Assets group: its key (segmented for the
// highlight), the singular type noun, its nullable severity rollup (#25b) and the
// /asset drill-in it links to.
type searchAsset struct {
	NameSegs []hiSeg
	Type     string
	Severity string // the asset's aggregate severity token, or "" when no signal fires
	SevLabel string // the severity capitalised for the sevbadge label ("Critical"), or ""
	Href     string
}

type searchSignal struct {
	RuleSegs    []hiSeg
	SubjectSegs []hiSeg
	Severity    string // the rule's severity token: critical | high | medium | low | info
	SevLabel    string // the severity capitalised for the sevbadge label
	Href        string
}

// searchDoc is one operator guide in the Documentation group: its front-matter title
// and one-line description (both segmented). A guide has no in-console route, so the
// row is non-navigating (no Href), matching the spec's own doc rows.
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

// guideIndex is the operator guides (docs/guides/*.md) parsed once at startup into
// their front-matter title + description — the Documentation store the Search screen
// reads (P2.5). Sorted by filename so the group renders in a stable order.
var guideIndex = loadGuideIndex()

// loadGuideIndex reads every embedded guide, parses its YAML front-matter title and
// description, and returns them in filename order. A guide missing a title is skipped
// rather than rendered blank — the front matter is the index key.
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

// guideFrontMatter extracts the title and description from a guide's leading YAML
// front-matter block (the "---"-fenced key: value header). Values are trimmed, so a
// CRLF checkout yields the same index as an LF one; a colon inside a value is kept
// (only the first ":" splits the key). A file without the opening fence yields "".
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

// searchRenderMap assembles the render map the frozen search.tmpl consumes. Both the
// live read path and the devMode fixture path build the tmpl's holes and hand them
// here, so the two produce the identical shape.
func searchRenderMap(acct db.Account, q string, total int, assets []searchAsset, signals []searchSignal, batches []searchBatch, docs []searchDoc) map[string]any {
	return map[string]any{
		"Title": "Search results", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "",
		"Query":     q,
		"Total":     total,
		"Assets":    assets,
		"Signals":   signals,
		"Batches":   batches,
		"Docs":      docs,
	}
}

// searchPage renders the full-page search. A viewer reads it — it surfaces only data a
// viewer already reads elsewhere, and mutates nothing. Under VERGE_DEV it serves the
// pinned fixtures.json search slice (the pixel-parity harness). Each live read degrades
// to an absent group rather than 500ing: a search that cannot reach one corpus still
// answers over the others.
func (s *server) searchPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "search", s.searchFixtureData(acct, r))
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	ctx := r.Context()

	// Signals — the fired members of the live corpus, matched on rule or subject.
	// Folding the corpus also gives the nav pill its open-signal count AND the
	// per-subject worst severity the asset rows badge (#25b): an asset's severity is
	// the most urgent severity across the signals firing on it (assetHeaderSeverity's
	// rollup). On a corpus failure the group is simply absent.
	var signals []searchSignal
	assetSev := map[string]string{}
	openSignals := 0
	if corpus, err := s.buildSignalCorpus(r); err != nil {
		log.Printf("web: search: build signal corpus: %v", err)
	} else {
		for _, c := range signal.EvaluateCorpus(corpus) {
			openSignals += len(c.Fired)
			sev, _ := signal.SeverityFor(c.Rule)
			for _, m := range c.Fired {
				// Roll the worst (lowest-rank) severity onto the subject for its asset badge.
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

	// Assets — current Name subjects, filtered server-side by the query. The row
	// carries the subject's severity rollup (#25b), nullable when no signal fires.
	var assets []searchAsset
	if rows, err := s.store.ListCurrentNameSubjects(ctx, db.ListCurrentNameSubjectsParams{
		Search: q, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); err != nil {
		log.Printf("web: search: list name subjects: %v", err)
	} else {
		for _, row := range rows {
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

	var batches []searchBatch
	if rows, err := s.store.ListDispatchProgress(ctx, scansHistoryLimit); err != nil {
		log.Printf("web: search: list dispatch progress: %v", err)
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

	// Documentation — the operator guides embedded from docs/guides/, matched on their
	// front-matter title or description (P2.5). A guide has no in-console route, so the
	// rows are non-navigating.
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

	data := searchRenderMap(acct, q, total, assets, signals, batches, docsHits)
	if openSignals > 0 {
		data["SignalCount"] = openSignals
	}
	s.render(w, r, "search", data)
}
