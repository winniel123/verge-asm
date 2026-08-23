package main

import (
	"html"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The Search results screen (#303, T8, ADR-0110) — the full-page search where the
// shell's command palette ("see everything") lands. Ported from
// design-system/examples/console/SearchResults.jsx: a centred 900px column with a
// query header (heading, mono input, result count), then one Card per kind
// (Assets, Signals, Batches, Documentation) whose rows link to the estate's
// existing routes, with the matched term highlighted in each row. When nothing
// matches, the design-system empty-state renders instead. The example's
// components are translated to template-local CSS within the existing token
// vocabulary (restyling, not authoring — ADR-0109); no design-system component is
// authored here.
//
// The sample data is swapped for real reads of the same shape (the ADR-0110
// contract): Assets are the current Name subjects (ListCurrentNameSubjects, its
// server-side Search filtering the query), Signals the fired members of the live
// signal corpus, Batches the recent Dispatches off the Operational queue corpus.
// Each links to the route that already exists — an asset to /asset/{key}, a signal
// to /signals, a batch to /run/{id}. The app is server-rendered with no client
// filter machinery (the T0 shell ships none), so the query rides the ?q= string
// and the input is a GET form; the highlight is computed server-side.
//
// Three domain holds against the mock, on the same footing as the Asset detail and
// Reports screens:
//
//   - Signals and assets carry NO severity. The census is "deliberately not a
//     severity ramp" (signals.go, ADR-0024), so the example's SeverityBadge on the
//     asset and signal rows is dropped rather than fabricated — the signal row
//     leads with the shield-alert glyph, the domain's signal icon. No SeverityBadge
//     renders here; the five-level ramp is never invented.
//   - Signals are withdrawn by the world, never "resolved" by an operator; the copy
//     holds that (the count is "N open", never "N resolved").
//
// The example's fourth group, Documentation, was dropped (#316). It had no content
// store and no /docs route to search over, and none is planned — the first-run docs
// gaps (#242) are closed and the marketing DocsPage is explicitly out of console
// scope (#294) — so the group could never light up. Rather than carry permanently-
// empty dead markup that implies a store waiting to be filled, the group is removed;
// /search now carries only kinds it can actually answer. If an in-console docs
// surface is ever added, the group is re-added deliberately alongside its store.

// searchAsset is one Name subject in the Assets group: its key (the highlighted
// mono label), the singular domain type noun, and the /asset drill-in it links to.
type searchAsset struct {
	Name template.HTML
	Type string
	Href string
}

// searchSignal is one fired signal in the Signals group: the firing rule and the
// subject it fired on, both highlighted, linking to the Signals screen. It carries
// no severity — signals are not a ramp (ADR-0024).
type searchSignal struct {
	Rule    template.HTML
	Subject template.HTML
	Href    string
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
				signals = append(signals, searchSignal{
					Rule:    searchHighlight(c.Rule, q),
					Subject: searchHighlight(m.Subject, q),
					Href:    "/signals",
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

	total := len(assets) + len(signals) + len(batches)

	data := map[string]any{
		"Title": "Search results", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "",
		"Query":     q,
		"Total":     total,
		"Assets":    assets,
		"Signals":   signals,
		"Batches":   batches,
	}
	if openSignals > 0 {
		data["SignalCount"] = openSignals
	}
	s.render(w, "search", data)
}
