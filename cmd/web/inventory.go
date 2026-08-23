package main

import (
	"encoding/csv"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Inventory screen (#243, ADR-0105). Where the Subjects listing and its
// drill-down are built around *change* — counts, verdicts, and the timeline of
// Spans — Inventory answers the complementary question "what do I actually have
// right now?". It reads the open-span corpus: the span_open_timeline_idx makes at
// most one open span per (subject, facet, discriminator, vantage, source)
// timeline, so each open span IS the value that timeline currently holds. A
// withdrawal closes a subject's spans (ADR-0082), so an open span is a current
// member by construction — the axis needs no membership re-derivation and, like
// the Subjects listing, states no denominator (ADR-0072). The actual observed
// values — resolved addresses, DNS records, certificate chain, HTTP identity, TLS
// acceptance — are rendered inline via the same spanDetails the drill-down
// expands, so the operator reads what a subject holds without a Postgres session.

// inventoryFacet is one facet a subject currently holds: its human label, the
// collapsed summary (the same value the change views show), and the expanded
// per-item detail rows where the facet has them. A Gap is a facet the system
// currently cannot value, carried as such rather than hidden. src and van carry
// the span's source and vantage so two open timelines of the same facet and
// discriminator — the same Service reached from two vantages, say — can be told
// apart in the label rather than colliding into indistinguishable rows.
type inventoryFacet struct {
	Label   string
	Summary string
	IsGap   bool
	Details []spanDetail
	Since   string

	src string
	van pgtype.Int8
}

// inventorySubject is one subject and every facet it currently holds. Link is the
// row-click destination — a Name opens the Asset detail (`/asset/{key}`, T1), every
// other kind its own subject drill-down (inventoryRowHref); it is empty for a kind
// with no surface yet, which then renders as plain, non-navigable text.
type inventorySubject struct {
	Kind   string
	Key    string
	// Type is the singular domain noun the row's Type cell carries — Name,
	// Service, Endpoint, Address — the subject's kind said in the interface's
	// vocabulary rather than the stored lower-case tag.
	Type   string
	Link   string
	Facets []inventoryFacet
}

// inventoryGroup buckets subjects of one kind under a plural heading, in the
// order the kinds first appear in the (kind, key)-ordered read.
type inventoryGroup struct {
	Kind     string
	Label    string
	Subjects []inventorySubject
}

// inventoryKindLabel renders a subject kind as the plural heading its group
// carries. An unknown kind falls back to the raw kind rather than an empty
// heading, so a facet added ahead of its label still lists.
func inventoryKindLabel(kind string) string {
	switch kind {
	case "name":
		return "Names"
	case "service":
		return "Services"
	case "endpoint":
		return "Endpoints"
	case "address":
		return "Addresses"
	default:
		return kind
	}
}

// inventoryTypeLabel renders a subject kind as the singular domain noun the
// per-row Type cell carries — the four subjects said in the interface's
// vocabulary (Name / Service / Endpoint / Address), never the stored tag. An
// unknown kind falls back to the raw kind so a new facet still labels.
func inventoryTypeLabel(kind string) string {
	switch kind {
	case "name":
		return "Name"
	case "service":
		return "Service"
	case "endpoint":
		return "Endpoint"
	case "address":
		return "Address"
	default:
		return kind
	}
}

// inventoryRowHref is the row-click destination for one inventory subject. A Name
// row opens the Asset detail (#296, T1) on the stable `/asset/{key}` route — the
// per-asset drill-in the Inventory row links to (T15). Every other kind keeps its
// own subject drill-down (subjectHref): Service and Endpoint carry a `/`/`@` and
// arrive as `?key=`, an Address routes to `/subjects/{key}`. The Name key holds no
// `/` or `@`, so a plain `/asset/{key}` path segment resolves.
func inventoryRowHref(kind, key string) string {
	if kind == "name" {
		return "/asset/" + url.PathEscape(key)
	}
	return subjectHref(kind, key)
}

// buildInventory groups the estate's open spans into per-subject inventory,
// preserving the read's (kind, key, facet, discriminator) order so a subject's
// facets list deterministically and the kind groups appear in a stable order. The
// rows are read straight off the derived span corpus, so this is pure rendering:
// no membership re-derivation, no query. Drill-down links go through subjectHref
// so a key's `/`, `@`, or reserved characters are escaped exactly as everywhere
// else in the app (#248).
func buildInventory(rows []db.ListAllOpenSpansRow) []inventoryGroup {
	var groups []inventoryGroup
	groupIdx := map[string]int{}   // kind -> index in groups
	subjectIdx := map[string]int{} // kind\x00key -> index in that group's Subjects

	for _, row := range rows {
		gi, ok := groupIdx[row.SubjectKind]
		if !ok {
			gi = len(groups)
			groupIdx[row.SubjectKind] = gi
			groups = append(groups, inventoryGroup{Kind: row.SubjectKind, Label: inventoryKindLabel(row.SubjectKind)})
		}

		skey := row.SubjectKind + "\x00" + row.SubjectKey
		si, ok := subjectIdx[skey]
		if !ok {
			si = len(groups[gi].Subjects)
			subjectIdx[skey] = si
			groups[gi].Subjects = append(groups[gi].Subjects, inventorySubject{
				Kind: row.SubjectKind,
				Key:  row.SubjectKey,
				Type: inventoryTypeLabel(row.SubjectKind),
				Link: inventoryRowHref(row.SubjectKind, row.SubjectKey),
			})
		}

		s := &groups[gi].Subjects[si]
		s.Facets = append(s.Facets, inventoryFacet{
			Label:   timelineLabel(row.Facet, row.Discriminator),
			Summary: valueLabel(row.Facet, row.Value, row.IsGap),
			IsGap:   row.IsGap,
			Details: spanDetails(row.Facet, row.Value, row.IsGap),
			Since:   row.OpenedAt.Time.UTC().Format(spanTimeFmt),
			src:     row.Source,
			van:     row.VantageID,
		})
	}

	for gi := range groups {
		for si := range groups[gi].Subjects {
			disambiguateFacetLabels(groups[gi].Subjects[si].Facets)
		}
	}
	return groups
}

// disambiguateFacetLabels appends a source/vantage qualifier to any facet rows a
// subject holds that would otherwise share a label — two open timelines of the
// same (facet, discriminator) differing only by vantage or source, which the
// span_open_timeline_idx permits (a Name resolved from two vantage classes, a
// Service reached from two vantages). A subject holding one span per facet is left
// with the plain label, so the common case stays uncluttered.
func disambiguateFacetLabels(facets []inventoryFacet) {
	counts := map[string]int{}
	for _, f := range facets {
		counts[f.Label]++
	}
	for i := range facets {
		if counts[facets[i].Label] <= 1 {
			continue
		}
		if q := facetVantageSource(facets[i].src, facets[i].van); q != "" {
			facets[i].Label += " · " + q
		}
	}
}

// facetVantageSource renders a span's source and vantage as a short qualifier —
// `resolver`, `prober · vantage 3` — used only to tell colliding facet rows apart.
// The vantage is its raw id: its class is re-verified per render from the presented
// address, never a stored label (ADR-0103, `ListReachabilitySpansForExposure`), so
// the id is the honest provisioning detail to disambiguate on.
func facetVantageSource(src string, van pgtype.Int8) string {
	var parts []string
	if src != "" {
		parts = append(parts, src)
	}
	if van.Valid {
		parts = append(parts, "vantage "+strconv.FormatInt(van.Int64, 10))
	}
	return strings.Join(parts, " · ")
}

// inventoryPage is the estate-wide Inventory read (#243). It reads every open span
// once and groups it by subject — the current value of each facet, with the actual
// records/addresses/identity expanded inline.
func (s *server) inventoryPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	rows, err := s.store.ListAllOpenSpans(r.Context())
	if err != nil {
		s.serverError(w, "list all open spans", err)
		return
	}
	groups := buildInventory(rows)
	s.render(w, "inventory", map[string]any{
		"Title": "Inventory", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory",
		"Groups":    groups,
		// Gate the Export CSV button on data presence, exactly as Drift's {{if
		// .HasEvents}} does (#347): an enabled link when a value has been folded, the
		// disabled button otherwise. An estate with no open span has nothing to export.
		"HasData": len(groups) > 0,
	})
}

// inventoryExport serves the folded inventory — every open span, grouped by subject —
// as a downloadable CSV (#347), the same reason the Drift and Reports exports exist:
// pull the current values into a sheet or a pipeline without screenshotting. It reads
// the same open-span corpus the Inventory page renders (read-only, ADR-0007), so the
// file mirrors the screen; it owns no mutation and adds no store method. It fabricates
// nothing: an empty estate produces a header-only file, never invented rows, and a
// facet the system currently cannot value is exported as a Gap rather than a zero.
func (s *server) inventoryExport(w http.ResponseWriter, r *http.Request, acct db.Account) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		http.Error(w, "unsupported export format: "+format+" (want csv)", http.StatusBadRequest)
		return
	}

	rows, err := s.store.ListAllOpenSpans(r.Context())
	if err != nil {
		s.serverError(w, "inventory export: list all open spans", err)
		return
	}
	s.writeInventoryExportCSV(w, buildInventory(rows))
}

// writeInventoryExportCSV emits the inventory as one uniform table — one row per facet
// a subject currently holds, in the same (kind, key, facet) order the screen renders.
// The `type` cell carries the singular domain noun the screen's Type column shows
// (Name / Service / Endpoint / Address), so the file reads in the interface's own
// vocabulary. A Gap facet — a value the system currently cannot state — carries the
// literal "Gap" (f.Summary is already "Gap" for a gap, valueLabel's isGap branch),
// never a blank standing in for a real read. The free-text cells (subject, facet,
// value) are passed through csvSafe so a value ingested from an attacker-influenced
// source cannot execute as a spreadsheet formula.
func (s *server) writeInventoryExportCSV(w http.ResponseWriter, groups []inventoryGroup) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="inventory-`+s.now().UTC().Format("2006-01-02")+`.csv"`)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"type", "subject", "facet", "value", "since"})

	for _, g := range groups {
		for _, sub := range g.Subjects {
			for _, f := range sub.Facets {
				_ = cw.Write([]string{
					csvSafe(sub.Type),
					csvSafe(sub.Key),
					csvSafe(f.Label),
					csvSafe(f.Summary),
					f.Since,
				})
			}
		}
	}
}
