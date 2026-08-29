package main

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
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

	// ProxyEdge marks a reach Gap the blanket-responder classifier attributed to a
	// provider-fronted / edge-shared address (R4-Q1 #762, ADR-0104, #778): the reach
	// is a Gap because the classifier could not attribute a listener to the origin —
	// either the address answers on every port (a measured blanket responder) or its
	// control probe timed out (undiscriminated) — so the frozen tmpl rides a "proxy
	// edge" badge on the gap. It is set on a reachability Gap whose stored value carries
	// blanketdiscrim's sixth cause (GapCause), for BOTH the blanket and the incomplete
	// reason (#778); a non-reachability facet and a non-Gap are never proxy edges.
	ProxyEdge bool

	// facet is the stored (lower-case) facet tag this row was folded from —
	// resolution, dns-record, reachability, tls-acceptance, certificate,
	// http-identity. It is the sort key the canonical inventory facet order sorts
	// on (inventoryFacetRank), kept alongside src/van so the display Label ("dns-records",
	// "certificate-chain") never has to be reverse-mapped to its facet.
	facet string
	src   string
	van   pgtype.Int8
}

// inventorySubject is one subject and every facet it currently holds. Link is the
// row-click destination — a Name opens the Asset detail (`/asset/{key}`, T1), every
// other kind its own subject drill-down (inventoryRowHref); it is empty for a kind
// with no surface yet, which then renders as plain, non-navigable text.
type inventorySubject struct {
	Kind string
	Key  string
	// Type is the singular domain noun the row's Type cell carries — Name,
	// Service, Endpoint, Address — the subject's kind said in the interface's
	// vocabulary rather than the stored lower-case tag.
	Type   string
	Link   string
	Facets []inventoryFacet
	// ProxyEdge is true when the subject holds at least one proxy-edge reach Gap —
	// the row-level datum the frozen tmpl reads to mark the row data-proxy="1" (the
	// client-side "Hide proxy edge" toggle scopes on it) and to demote it in place via
	// the existing value-before-Gap sort (R4-Q1 #762 — NO new sort key).
	ProxyEdge bool
}

// HasGap reports whether the subject currently holds at least one Gap facet — a
// timeline whose value the system cannot state. The Inventory "Gaps only"
// client-side scope (SPEC-CHANGE #13, package v3.2.4) reads it off each rendered
// row to hide subjects that hold no Gap, without a server round-trip.
func (s inventorySubject) HasGap() bool {
	for _, f := range s.Facets {
		if f.IsGap {
			return true
		}
	}
	return false
}

// inventoryGroup buckets subjects of one kind under a plural heading, in the
// order the kinds first appear in the (kind, key)-ordered read.
type inventoryGroup struct {
	Kind     string
	Label    string
	Subjects []inventorySubject
	// Total is the full count of subjects in this kind — the group-count badge the
	// frozen tmpl renders beside the heading. ListAllOpenSpans returns every open
	// span, so buildInventory sets Total to the true folded subject count; the
	// display window (#756) then caps .Subjects to inventoryGroupWindow rows without
	// touching Total, so the badge always states the whole group.
	Total int
	// More is the number of subjects beyond the shown window (Total − len(Subjects)),
	// 0 when the whole group fits or the group is expanded via ?all=<kind>. The tmpl
	// gates the "Show all N — M more" expander on it.
	More int
	// ShowAllHref lifts the window for this one group — /inventory?all=<kind>.
	ShowAllHref string
}

// inventoryGroupWindow is the per-group row cap the Inventory windowing (#756)
// applies: a counted group shows at most this many subjects, with a "Show all N"
// expander lifting the cap for that one group (?all=<kind>). The kind segmented
// control and the group-count badge state the whole group regardless.
const inventoryGroupWindow = 25

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
	// An Address has no drill-in surface of its own in the Inventory pilot — the
	// row renders as plain, non-navigable text (fixtures.json carries link="" for
	// every Address). The shared subjectHref would route it to /subjects/{key},
	// which is why this is an inventory-local override rather than a subjectHref
	// change: the graph/search paths still link Addresses through subjectHref.
	if kind == "address" {
		return ""
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
			Label:     inventoryFacetLabel(row.Facet, row.Discriminator),
			Summary:   inventoryValueLabel(row.Facet, row.Value, row.IsGap),
			IsGap:     row.IsGap,
			ProxyEdge: inventoryProxyEdge(row.Facet, row.Value, row.IsGap),
			Details:   inventorySpanDetails(row.Facet, row.Value, row.IsGap),
			// Inventory renders the Since column (and the CSV export) date-only
			// (#524) — the day a subject's currently-held span opened, without the
			// wall-clock time the change/drill-down views carry. spanTimeFmt stays the
			// shared datetime format those other screens depend on; only the inventory
			// facet's Since is the shorter form.
			Since: row.OpenedAt.Time.UTC().Format("2006-01-02"),
			facet: row.Facet,
			src:   row.Source,
			van:   row.VantageID,
		})
	}

	// Order everything by the canonical inventory orderings so the render is
	// deterministic regardless of the open-span read order: facets within a subject
	// by inventoryFacetRank, subjects within a group by their leading facet, and the
	// groups themselves by kind. Every sort is stable so ties keep read order (which
	// is what pins the two vantages of an Address's reachability in insertion order).
	for gi := range groups {
		for si := range groups[gi].Subjects {
			facets := groups[gi].Subjects[si].Facets
			sort.SliceStable(facets, func(a, b int) bool {
				return inventoryFacetRank(facets[a].facet) < inventoryFacetRank(facets[b].facet)
			})
			// A subject is a proxy edge when any facet it holds is a proxy-edge reach
			// Gap; the row-level datum the tmpl marks data-proxy="1" on and the existing
			// value-before-Gap sort demotes in place (R4-Q1 #762, no new sort key).
			for _, f := range groups[gi].Subjects[si].Facets {
				if f.ProxyEdge {
					groups[gi].Subjects[si].ProxyEdge = true
					break
				}
			}
		}
		subs := groups[gi].Subjects
		sort.SliceStable(subs, func(a, b int) bool {
			return lessInventorySubject(subs[a], subs[b])
		})
		// ListAllOpenSpans returns every open span, so the folded subject count IS the
		// true group total (no separate count query needed); the display window (#756)
		// later caps .Subjects without touching Total. ShowAllHref lifts that cap for
		// this one group.
		groups[gi].Total = len(subs)
		groups[gi].ShowAllHref = "/inventory?all=" + groups[gi].Kind
	}
	propagateProxyEdgeToAddresses(groups)
	sort.SliceStable(groups, func(a, b int) bool {
		return inventoryKindRank(groups[a].Kind) < inventoryKindRank(groups[b].Kind)
	})
	return groups
}

// propagateProxyEdgeToAddresses lifts each proxy-edge Service's ProxyEdge onto its
// bare Address subject (#778). A reach Gap folds under the Service subject
// (subjectKindFor("reachability") == "service"), so the blanket-responder verdict
// lands on the Service row (`104.21.61.6:443/tcp`), never the bare Address
// (`104.21.61.6`) the "Hide proxy edge" toggle filters. The pipeline emits no
// address-kind reach span, so without this lift a flagged edge never reaches the
// Address group the toggle scopes. It only sets ProxyEdge, never clears it, so an
// Address already flagged by its own facet (the design fixture's shape) is untouched.
func propagateProxyEdgeToAddresses(groups []inventoryGroup) {
	proxyAddrs := map[string]bool{}
	for gi := range groups {
		if groups[gi].Kind != "service" {
			continue
		}
		for _, sub := range groups[gi].Subjects {
			if sub.ProxyEdge {
				proxyAddrs[inventoryServiceAddress(sub.Key)] = true
			}
		}
	}
	if len(proxyAddrs) == 0 {
		return
	}
	for gi := range groups {
		if groups[gi].Kind != "address" {
			continue
		}
		for si := range groups[gi].Subjects {
			if proxyAddrs[groups[gi].Subjects[si].Key] {
				groups[gi].Subjects[si].ProxyEdge = true
			}
		}
	}
}

// inventoryServiceAddress extracts the bare Address limb of a Service subject key
// (`address:port/transport`, connectoutcome.ServiceKey) — everything before the last
// colon, with a bracketed IPv6 host unwrapped so it equals the Address subject key
// (netip.Addr.String()). It mirrors internal/queue's serviceAddress, kept inventory-
// local so cmd/web imports nothing from the queue package.
func inventoryServiceAddress(key string) string {
	i := strings.LastIndex(key, ":")
	if i < 0 {
		return key
	}
	host := key[:i]
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	return host
}

// inventoryProxyEdge reports whether a facet is a proxy-edge reach Gap — a
// reachability timeline the blanket-responder classifier (internal/measure/
// blanketdiscrim, ADR-0104) gapped because the address is a provider-fronted /
// edge-shared responder. It reuses the leaf's own cause constant rather than an
// address/CIDR list (the project refuses a vendor prefix list as the detector): the
// stored reach-Gap value carries the sixth-cause tag (blanketdiscrim.GapCause).
//
// It marks BOTH reasons the cause carries as a proxy edge (#778): the blanket reason
// (the control set answered on every port — a measured blanket responder) AND the
// incomplete reason (the control probe timed out, so blanket-ness could not be
// decided). A live hot scan of a Cloudflare-fronted address times its control probe
// out far more often than it completes, so the incomplete Gap is the shape the
// operator actually meets; badging it lets the "Hide proxy edge" toggle hide the edge
// address the operator asked to hide, rather than only a positively measured blanket
// responder. Non-reachability facets and non-Gaps are never proxy edges.
func inventoryProxyEdge(dbFacet string, value []byte, isGap bool) bool {
	if !isGap || dbFacet != "reachability" {
		return false
	}
	var v struct {
		Cause string `json:"cause"`
	}
	if err := json.Unmarshal(value, &v); err != nil {
		return false
	}
	return v.Cause == blanketdiscrim.GapCause
}

// inventoryFacetRank is the canonical display order of a subject's facets on the
// Inventory pilot: resolution, dns-record, reachability, tls-acceptance,
// certificate, http-identity — the order the fixture lists them and the order the
// subject sort reads its leading facet from. An unknown facet sorts last so a new
// facet folded ahead of its rank still lists rather than jumping the column.
func inventoryFacetRank(facet string) int {
	switch facet {
	case "resolution":
		return 0
	case "dns-record":
		return 1
	case "reachability":
		return 2
	case "tls-acceptance":
		return 3
	case "certificate":
		return 4
	case "http-identity":
		return 5
	default:
		return 99
	}
}

// inventoryKindRank is the canonical group order: Names, Services, Endpoints,
// Addresses — the estate read top-down from the naming layer to the raw address.
// An unknown kind sorts last, stable, so a new subject kind still groups.
func inventoryKindRank(kind string) int {
	switch kind {
	case "name":
		return 0
	case "service":
		return 1
	case "endpoint":
		return 2
	case "address":
		return 3
	default:
		return 99
	}
}

// lessInventorySubject orders two subjects within a group by their leading facet
// (facet[0] after the canonical facet sort): a subject whose leading facet holds a
// value sorts ahead of one whose leading facet is a Gap; among equals the more
// recently-opened leads (the "since" date compares lexically, later-first); ties
// break on the subject key. This reproduces the fixture's per-group subject order.
func lessInventorySubject(a, b inventorySubject) bool {
	af, bf := leadingFacet(a), leadingFacet(b)
	if af.IsGap != bf.IsGap {
		return !af.IsGap // a value before a Gap
	}
	if af.Since != bf.Since {
		return af.Since > bf.Since // later "since" first
	}
	return a.Key < b.Key
}

// leadingFacet returns a subject's first facet after the canonical facet sort — the
// facet the subject order keys on — or a zero facet for a subject that (impossibly,
// every inventory subject holds at least one) holds none.
func leadingFacet(s inventorySubject) inventoryFacet {
	if len(s.Facets) == 0 {
		return inventoryFacet{}
	}
	return s.Facets[0]
}

// inventoryFacetLabel renders the facet label the Inventory pilot shows — the
// display facet noun, with the span's discriminator appended where it carries one.
// Two facets are renamed for the surface: dns-record reads "dns-records" and
// certificate reads "certificate-chain"; the other four render their stored tag.
// The discriminator (e.g. "vantage 1") already tells two open timelines of one
// facet apart, so no source/vantage disambiguation runs in the inventory path —
// the label is unique by construction.
func inventoryFacetLabel(dbFacet, discriminator string) string {
	displayFacet := dbFacet
	switch dbFacet {
	case "dns-record":
		displayFacet = "dns-records"
	case "certificate":
		displayFacet = "certificate-chain"
	}
	if discriminator != "" {
		return displayFacet + " · " + discriminator
	}
	return displayFacet
}

// invResolutionValue is the resolution value the Inventory loader stores: the RR
// type answered and the addresses it resolved to. It is inventory-local — the
// shared resolutionValue (subjects.go) carries an `outcome` the change views
// summarise, where the inventory summary is `rrtype · <n> addresses`.
type invResolutionValue struct {
	RRType    string   `json:"rrtype"`
	Addresses []string `json:"addresses"`
}

// invReachabilityValue is the reachability value the Inventory loader stores: the
// outcome and the ports it answered on, rendered `outcome · port · port`.
type invReachabilityValue struct {
	Outcome string   `json:"outcome"`
	Ports   []string `json:"ports"`
}

// invTLSAcceptanceValue is the tls-acceptance value the Inventory loader stores:
// the outcome and the plain version strings ("1.2", "1.3"), rendered `TLS 1.2 · 1.3`
// on an enumeration and the bare outcome ("none · plaintext ssh") otherwise. The
// shared tlsAcceptanceValue carries per-version cipher suites the change views
// expand; the inventory summary needs only the version strings, so it decodes its
// own shape.
type invTLSAcceptanceValue struct {
	Outcome  string   `json:"outcome"`
	Versions []string `json:"versions"`
}

// invCertificateValue is the certificate chain the Inventory loader stores: an
// ordered chain of links, the leaf first, each carrying its CN and either the
// leaf's not_after or an intermediate's issuer_org. The shared certificateValue
// carries opaque fingerprint strings; the inventory chain carries the parsed
// identity the pilot renders, so it decodes its own shape.
type invCertificateValue struct {
	Chain []struct {
		CN        string `json:"cn"`
		NotAfter  string `json:"not_after"`
		IssuerOrg string `json:"issuer_org"`
	} `json:"chain"`
}

// inventoryValueLabel renders a facet's collapsed one-line summary for the
// Inventory pilot from the loader-authored structured value. A Gap holds no value,
// so its summary is empty (the template renders the Gap marker off IsGap). Each
// facet composes its own line from the admitted fields — never a raw outcome tag
// where the pilot shows a shaped value.
func inventoryValueLabel(dbFacet string, value []byte, isGap bool) string {
	if isGap {
		return ""
	}
	switch dbFacet {
	case "resolution":
		v := decodeInvResolution(value)
		if len(v.Addresses) == 1 {
			return v.RRType + " · " + v.Addresses[0]
		}
		return v.RRType + " · " + strconv.Itoa(len(v.Addresses)) + " addresses"
	case "dns-record":
		rrs := decodeDNSRecord(value).RRs
		distinct := orderedDistinctRRTypes(rrs)
		if len(distinct) == 1 {
			unit := " records"
			if len(rrs) == 1 {
				unit = " record"
			}
			return distinct[0] + " · " + strconv.Itoa(len(rrs)) + unit
		}
		return strings.Join(distinct, " · ")
	case "reachability":
		v := decodeInvReachability(value)
		if len(v.Ports) > 0 {
			return v.Outcome + " · " + strings.Join(v.Ports, " · ")
		}
		return v.Outcome
	case "tls-acceptance":
		v := decodeInvTLSAcceptance(value)
		if len(v.Versions) > 0 {
			return "TLS " + strings.Join(v.Versions, " · ")
		}
		return v.Outcome
	case "certificate":
		chain := decodeInvCertificate(value).Chain
		if len(chain) == 0 {
			return ""
		}
		return "leaf " + chain[0].CN + " · exp " + chain[0].NotAfter
	case "http-identity":
		v := decodeHTTPIdentity(value)
		s := v.Server + " · " + strconv.Itoa(v.Status)
		if v.Title != "" {
			s += " · " + "“" + v.Title + "”" // curly “ ”
		}
		if v.RedirectLocation != "" {
			s += " → " + v.RedirectLocation // space arrow space
		}
		return s
	default:
		return ""
	}
}

// inventorySpanDetails lists a facet value's expand-on-click rows for the Inventory
// pilot. Only the facets the fixture expands carry detail rows — resolution (one
// row per address, but only where more than one resolved), dns-record (one row per
// RR), and certificate (one row per chain link). reachability, tls-acceptance and
// http-identity render their whole value in the summary, so they expand to nothing.
func inventorySpanDetails(dbFacet string, value []byte, isGap bool) []spanDetail {
	if isGap {
		return nil
	}
	switch dbFacet {
	case "resolution":
		v := decodeInvResolution(value)
		if len(v.Addresses) <= 1 {
			return nil
		}
		details := make([]spanDetail, 0, len(v.Addresses))
		for _, a := range v.Addresses {
			details = append(details, spanDetail{Type: v.RRType, Data: a})
		}
		return details
	case "dns-record":
		rrs := decodeDNSRecord(value).RRs
		if len(rrs) == 0 {
			return nil
		}
		details := make([]spanDetail, 0, len(rrs))
		for _, rr := range rrs {
			details = append(details, spanDetail{Type: rr.Type, Data: rr.Data})
		}
		return details
	case "certificate":
		chain := decodeInvCertificate(value).Chain
		if len(chain) == 0 {
			return nil
		}
		details := make([]spanDetail, 0, len(chain))
		for i, link := range chain {
			if i == 0 {
				details = append(details, spanDetail{Type: "leaf", Data: "CN=" + link.CN + " · not_after " + link.NotAfter})
			} else {
				details = append(details, spanDetail{Type: "int", Data: "CN=" + link.CN + " · " + link.IssuerOrg})
			}
		}
		return details
	default:
		return nil
	}
}

// orderedDistinctRRTypes returns the RR types in a dns-record value, de-duplicated
// but in first-seen order — the ordered set the collapsed summary joins.
func orderedDistinctRRTypes(rrs []struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"`
}) []string {
	seen := map[string]bool{}
	var out []string
	for _, rr := range rrs {
		if seen[rr.Type] {
			continue
		}
		seen[rr.Type] = true
		out = append(out, rr.Type)
	}
	return out
}

func decodeInvResolution(raw []byte) invResolutionValue {
	var v invResolutionValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func decodeInvReachability(raw []byte) invReachabilityValue {
	var v invReachabilityValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func decodeInvTLSAcceptance(raw []byte) invTLSAcceptanceValue {
	var v invTLSAcceptanceValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func decodeInvCertificate(raw []byte) invCertificateValue {
	var v invCertificateValue
	_ = json.Unmarshal(raw, &v)
	return v
}

// windowInventoryGroups applies the #756 per-group display window: each group shows
// at most inventoryGroupWindow subjects, with .More recording the count beyond the
// window and .Subjects trimmed to it — unless the group's kind is `expand` (the
// ?all=<kind> param), which lifts the cap for that one group so all its rows render.
// .Total is left as buildInventory set it (the whole-group count), so the badge and
// the "Show all N — M more" copy always state the full group.
func windowInventoryGroups(groups []inventoryGroup, expand string) {
	for i := range groups {
		g := &groups[i]
		if g.Kind == expand {
			g.More = 0
			continue
		}
		if len(g.Subjects) > inventoryGroupWindow {
			g.More = g.Total - inventoryGroupWindow
			g.Subjects = g.Subjects[:inventoryGroupWindow]
		}
	}
}

// devInventoryGroupTotals pins design-system/fixtures/fixtures.json → inventory
// group totals that differ from the seeded row count, for the VERGE_DEV pixel-parity
// capture. Only the address group differs: the fixture declares 41 (an estate-scale
// count) while the loader seeds 3 rows, so the golden's count badge reads "41" and
// its expander "Show all 41 — 38 more". The other groups' totals equal their seeded
// counts, so they need no override. TestInventoryFixtureCountsMatchPackage folds
// these back through the frozen package — the byte-exactness gate before the pixels.
var devInventoryGroupTotals = map[string]int{
	"address": 41,
}

// applyInventoryFixtureCounts overrides a dev-seeded group's .Total/.More with the
// pinned fixture total (devInventoryGroupTotals) so the served /inventory renders the
// design fixture's declared estate-scale counts under VERGE_DEV. More is derived as
// total − shown (0 when the group is expanded via ?all=<kind>), so it stays honest
// against however many rows the loader seeded.
func applyInventoryFixtureCounts(groups []inventoryGroup, expand string) {
	for i := range groups {
		g := &groups[i]
		total, ok := devInventoryGroupTotals[g.Kind]
		if !ok {
			continue
		}
		g.Total = total
		if g.Kind == expand {
			g.More = 0
			continue
		}
		g.More = total - len(g.Subjects)
	}
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
	// #756 windowing: cap each counted group to inventoryGroupWindow rows, except the
	// one group the operator expanded via ?all=<kind>. Total already states the whole
	// group; this only trims the rows shown and sets .More for the "Show all" expander.
	windowInventoryGroups(groups, r.URL.Query().Get("all"))
	// The VERGE_DEV pixel-parity capture seeds only a 3-row address group, but the
	// design fixture declares its estate-scale total (41, 38 beyond the window) so the
	// golden shows the count badge + "Show all" expander. Supply that pinned total in
	// dev only; a real deployment renders the honest folded count above. (No effect on
	// the go-test fakeStore path — that never sets devMode.)
	if s.devMode {
		applyInventoryFixtureCounts(groups, r.URL.Query().Get("all"))
	}
	s.render(w, r, "inventory", map[string]any{
		"Title": "Inventory", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory",
		"Groups":    groups,
		// The frozen design tmpl styles against the design-owned CSS-token vocabulary
		// (design-system/tokens/*.css). Opt this page into loading those tokens (the
		// "head" block gates on this datum); no other screen sets it, so their styling
		// is untouched.
		"DesignTokens": true,
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
// literal "Gap" (its inventory summary is empty, so the export substitutes the word),
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
				// A Gap facet holds no value, so its inventory summary is empty (the
				// screen renders the Gap marker off IsGap, not off the summary). The
				// export names the Gap explicitly — the literal "Gap" — rather than a
				// blank cell that reads as a missing export rather than an honest
				// "we currently cannot state this".
				value := f.Summary
				if f.IsGap {
					value = "Gap"
				}
				_ = cw.Write([]string{
					csvSafe(sub.Type),
					csvSafe(sub.Key),
					csvSafe(f.Label),
					csvSafe(value),
					f.Since,
				})
			}
		}
	}
}
