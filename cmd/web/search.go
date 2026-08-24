package main

import (
	"html"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/winniel123/verge-asm/docs/guides"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The Search results screen (#303, T8, ADR-0110; parity restored P2.5, #451) — the
// full-page search where the shell's command palette ("see everything") lands.
// Ported from design-system/examples/console/SearchResults.jsx: a centred 900px
// column with a query header (heading, mono input, result count), then one Card per
// kind (Assets, Signals, Batches, Documentation) whose rows highlight the matched
// term. When nothing matches, the design-system empty-state renders instead. The
// example's components are translated to template-local CSS within the existing
// token vocabulary (restyling, not authoring — ADR-0109); no design-system component
// is authored here.
//
// The sample data is swapped for real reads of the same shape (the ADR-0110
// contract): Assets are the current Name subjects (ListCurrentNameSubjects, its
// server-side Search filtering the query), Signals the fired members of the live
// signal corpus, Batches the recent Dispatches off the Operational queue corpus, and
// Documentation the operator guides embedded from docs/guides/. Every navigable row
// links to the route that already exists — an asset to /asset/{key}, a signal to
// /signals, a batch to /run/{id}. The app is server-rendered with no client filter
// machinery (the T0 shell ships none), so the query rides the ?q= string and the
// input is a GET form; the highlight is computed server-side.
//
// Two parity restorations land here under the ruling that the design is normative
// for look AND functionality (PARITY-CHART.md §"The ruling"; ADR-0116):
//
//   - SeverityBadge is back on the signal rows. Severity is now a real datum —
//     assigned per rule (internal/signal, P0.1, #442; SeverityFor) — so the ramp is
//     read, never fabricated. The old "signals carry no severity" hold (ADR-0024,
//     SPEC-CHANGE.md collision #1) is superseded. Assets stay pill-free: a Name
//     subject carries no rule and so no severity, exactly as the spec's mail row
//     (sev: null) renders none. Signals are still withdrawn by the world, never
//     "resolved" — the count reads "N open".
//   - The Documentation group is restored (SPEC-CHANGE.md collision #5). Its original
//     drop rationale — "no content store" (#316) — no longer holds: the operator
//     guides under docs/guides/ ARE the store. They are embedded (docs/guides) and
//     indexed on their front-matter title + description; a query matches either, and
//     the group renders per the spec. A guide has no in-console route to open (there
//     is no /docs surface), so — like the spec's own doc rows, which carry no
//     onClick — the rows are non-navigating rather than fabricating a link.

// searchAsset is one Name subject in the Assets group: its key (the highlighted
// mono label), the singular domain type noun, and the /asset drill-in it links to.
type searchAsset struct {
	Name template.HTML
	Type string
	Href string
}

// searchSignal is one fired signal in the Signals group: the firing rule and the
// subject it fired on, both highlighted, linking to the Signals screen. It leads
// with its SeverityBadge — the rule's severity (P0.1), the same ramp the Signals
// screen shows, never fabricated.
type searchSignal struct {
	Rule     template.HTML
	Subject  template.HTML
	Severity string // the rule's severity token: critical | high | medium | low | info
	Href     string
}

// searchDoc is one operator guide in the Documentation group: its front-matter
// title and its one-line description (the spec's snippet), both highlighted. A guide
// has no in-console route to open, so the row is non-navigating (no Href), matching
// the spec's own doc rows which carry no onClick.
type searchDoc struct {
	Title template.HTML
	Snip  template.HTML
}

// guideDoc is one indexed operator guide: its front-matter title and description,
// the two fields a Documentation search matches and highlights on.
type guideDoc struct {
	title string
	desc  string
}

// guideIndex is the operator guides (docs/guides/*.md) parsed once at startup into
// their front-matter title + description — the Documentation store the Search screen
// reads (P2.5). Sorted by filename so the group renders in a stable order.
var guideIndex = loadGuideIndex()

// loadGuideIndex reads every embedded guide, parses its YAML front-matter title and
// description, and returns them in filename order. A guide missing a title is
// skipped rather than rendered blank — the front matter is the index key.
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

// searchBatch is one Dispatch in the Batches group: its BatchStatus state, the
// highlighted dispatched instant (the chip's recorded scope, matching the example's
// scope={b.id}), and the /run drill-in.
type searchBatch struct {
	Status string
	Label  template.HTML
	Href   string
}

// searchMatch reports whether text contains the query, case-insensitively. An empty
// query matches everything — the palette's "see everything" browse lands unfiltered,
// exactly as the example shows all sample data when its query is empty.
func searchMatch(text, q string) bool {
	return q == "" || strings.Contains(strings.ToLower(text), strings.ToLower(q))
}

// searchHighlight escapes text and wraps its first case-insensitive occurrence of q
// in an accent span, mirroring the example's hi() helper. It returns template.HTML,
// so every segment is escaped by hand — the wrapped run is the only markup emitted.
func searchHighlight(text, q string) template.HTML {
	if q == "" {
		return template.HTML(html.EscapeString(text))
	}
	// The match is found in the lowered text, so the span length must come from
	// the lowered query — not len(q). strings.ToLower can change a value's byte
	// length (e.g. U+212A KELVIN SIGN → "k", U+023A → U+2C65), so len(q) would
	// slice out of range and panic (#340). The clamp is a belt-and-braces guard:
	// when the lowered forms shift the trailing offsets past the original text,
	// fall back to the plain escaped original rather than slice out of bounds.
	i := strings.Index(strings.ToLower(text), strings.ToLower(q))
	if i < 0 {
		return template.HTML(html.EscapeString(text))
	}
	end := i + len(strings.ToLower(q))
	if i > len(text) || end > len(text) {
		return template.HTML(html.EscapeString(text))
	}
	var b strings.Builder
	b.WriteString(html.EscapeString(text[:i]))
	b.WriteString(`<span style="color:var(--link);font-weight:600">`)
	b.WriteString(html.EscapeString(text[i:end]))
	b.WriteString(`</span>`)
	b.WriteString(html.EscapeString(text[end:]))
	return template.HTML(b.String())
}

// searchPage renders the full-page search. A viewer reads it — it surfaces only
// data a viewer already reads elsewhere, and mutates nothing. Each read degrades to
// an absent group rather than 500ing the page: a search that cannot reach one
// corpus still answers over the others.
func (s *server) searchPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	ctx := r.Context()

	// Assets — current Name subjects, filtered server-side by the query. A Name
	// carries no severity, so the row leads with the subject glyph, not a ramp.
	var assets []searchAsset
	if rows, err := s.store.ListCurrentNameSubjects(ctx, db.ListCurrentNameSubjectsParams{
		Search: q, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); err != nil {
		log.Printf("web: search: list name subjects: %v", err)
	} else {
		for _, row := range rows {
			assets = append(assets, searchAsset{
				Name: searchHighlight(row.SubjectKey, q),
				Type: "name",
				Href: "/asset/" + url.PathEscape(row.SubjectKey),
			})
		}
	}

	// Signals — the fired members of the live corpus, matched on rule or subject.
	// Building the corpus also gives the nav pill its open-signal count. On a corpus
	// failure the group is simply absent (the page still answers over the rest).
	var signals []searchSignal
	openSignals := 0
	if corpus, err := s.buildSignalCorpus(r); err != nil {
		log.Printf("web: search: build signal corpus: %v", err)
	} else {
		for _, c := range signal.EvaluateCorpus(corpus) {
			openSignals += len(c.Fired)
			for _, m := range c.Fired {
				if !searchMatch(c.Rule, q) && !searchMatch(m.Subject, q) {
					continue
				}
				sev, _ := signal.SeverityFor(c.Rule)
				signals = append(signals, searchSignal{
					Rule:     searchHighlight(c.Rule, q),
					Subject:  searchHighlight(m.Subject, q),
					Severity: sev.String(),
					Href:     "/signals",
				})
			}
		}
	}

	// Batches — recent Dispatches off the Operational queue corpus, matched on the
	// dispatched instant or the scan kind. The state folds exactly as the Scans
	// monitor and Run detail derive it (running / failed / complete).
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
				Status: status,
				Label:  searchHighlight(dv.DispatchedAt, q),
				Href:   "/run/" + strconv.FormatInt(dv.ID, 10),
			})
		}
	}

	// Documentation — the operator guides embedded from docs/guides/, matched on
	// their front-matter title or description. The guides are the content store the
	// group's original drop (#316) said was missing; it exists now (P2.5).
	var docsHits []searchDoc
	for _, g := range guideIndex {
		if !searchMatch(g.title, q) && !searchMatch(g.desc, q) {
			continue
		}
		docsHits = append(docsHits, searchDoc{
			Title: searchHighlight(g.title, q),
			Snip:  searchHighlight(g.desc, q),
		})
	}

	total := len(assets) + len(signals) + len(batches) + len(docsHits)

	data := map[string]any{
		"Title": "Search results", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "",
		"Query":     q,
		"Total":     total,
		"Assets":    assets,
		"Signals":   signals,
		"Batches":   batches,
		"Docs":      docsHits,
	}
	if openSignals > 0 {
		data["SignalCount"] = openSignals
	}
	s.render(w, "search", data)
}
