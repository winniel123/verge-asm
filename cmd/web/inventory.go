package main

import (
	"net/http"
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
// drill-down href where one exists; it is empty for a kind with no surface yet,
// which then renders as plain text.
type inventorySubject struct {
	Kind   string
	Key    string
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
				Link: subjectHref(row.SubjectKind, row.SubjectKey),
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
		"Groups": groups,
	})
}
