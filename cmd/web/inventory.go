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

// An open span is a current member by construction, so no membership is re-derived (ADR-0082).

type inventoryFacet struct {
	Label   string
	Summary string
	IsGap   bool
	Details []spanDetail
	Since   string

	ProxyEdge bool

	facet string
	src   string
	van   pgtype.Int8
}

type inventorySubject struct {
	Kind      string
	Key       string
	Type      string
	Link      string
	Facets    []inventoryFacet
	ProxyEdge bool
}

func (s inventorySubject) HasGap() bool {
	for _, f := range s.Facets {
		if f.IsGap {
			return true
		}
	}
	return false
}

type inventoryGroup struct {
	Kind     string
	Label    string
	Subjects []inventorySubject

	Total       int
	More        int
	ShowAllHref string
}

const inventoryGroupWindow = 25

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

func inventoryRowHref(kind, key string) string {
	if kind == "name" {
		return "/asset/" + url.PathEscape(key)
	}
	// An inventory-local override: changing subjectHref would relink Addresses elsewhere (#243).
	if kind == "address" {
		return ""
	}
	return subjectHref(kind, key)
}

func buildInventory(rows []db.ListAllOpenSpansRow) []inventoryGroup {
	var groups []inventoryGroup
	groupIdx := map[string]int{}
	subjectIdx := map[string]int{}

	// The listing states no denominator: the estate can never honestly have one (ADR-0072).
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
			// Date-only here alone; spanTimeFmt stays the shared format the change views use (#524).
			Since: row.OpenedAt.Time.UTC().Format("2006-01-02"),
			facet: row.Facet,
			src:   row.Source,
			van:   row.VantageID,
		})
	}

	// Stable sorts keep ties in read order, which is what pins an Address's two vantages.
	for gi := range groups {
		for si := range groups[gi].Subjects {
			facets := groups[gi].Subjects[si].Facets
			sort.SliceStable(facets, func(a, b int) bool {
				return inventoryFacetRank(facets[a].facet) < inventoryFacetRank(facets[b].facet)
			})
			// A proxy edge is demoted in place by the existing sort, never by a new key (#762).
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
		groups[gi].Total = len(subs)
		groups[gi].ShowAllHref = "/inventory?all=" + groups[gi].Kind
	}
	propagateProxyEdgeToAddresses(groups)
	sort.SliceStable(groups, func(a, b int) bool {
		return inventoryKindRank(groups[a].Kind) < inventoryKindRank(groups[b].Kind)
	})
	return groups
}

func propagateProxyEdgeToAddresses(groups []inventoryGroup) {
	// No address-kind reach span exists, so without this lift no Address is flagged (ADR-0125, #778).
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

func inventoryServiceAddress(key string) string {
	// Duplicated from internal/queue's serviceAddress so cmd/web imports nothing from the queue.
	i := strings.LastIndex(key, ":")
	if i < 0 {
		return key
	}
	host := key[:i]
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	return host
}

func inventoryProxyEdge(dbFacet string, value []byte, isGap bool) bool {
	// A vendor prefix list is refused as the detector; the measured cause tag decides (ADR-0104).
	if !isGap || dbFacet != "reachability" {
		return false
	}
	var v struct {
		Cause string `json:"cause"`
	}
	if err := json.Unmarshal(value, &v); err != nil {
		return false
	}
	// Incomplete badges too: a fronted address times out more often than it completes (ADR-0125).
	return v.Cause == blanketdiscrim.GapCause
}

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

func lessInventorySubject(a, b inventorySubject) bool {
	af, bf := leadingFacet(a), leadingFacet(b)
	if af.IsGap != bf.IsGap {
		return !af.IsGap
	}
	if af.Since != bf.Since {
		return af.Since > bf.Since
	}
	return a.Key < b.Key
}

func leadingFacet(s inventorySubject) inventoryFacet {
	if len(s.Facets) == 0 {
		return inventoryFacet{}
	}
	return s.Facets[0]
}

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

type invResolutionValue struct {
	RRType    string   `json:"rrtype"`
	Addresses []string `json:"addresses"`
}

type invReachabilityValue struct {
	Outcome string   `json:"outcome"`
	Ports   []string `json:"ports"`
}

// A local shape, because the shared decode carries fields the inventory summary never renders.

type invTLSAcceptanceValue struct {
	Outcome  string   `json:"outcome"`
	Versions []string `json:"versions"`
}

type invCertificateValue struct {
	Chain []struct {
		CN        string `json:"cn"`
		NotAfter  string `json:"not_after"`
		IssuerOrg string `json:"issuer_org"`
	} `json:"chain"`
}

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
			s += " · " + "“" + v.Title + "”"
		}
		if v.RedirectLocation != "" {
			s += " → " + v.RedirectLocation
		}
		return s
	default:
		return ""
	}
}

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

func windowInventoryGroups(groups []inventoryGroup, expand string) {
	// Total stays whole, so the badge and "Show all N" state the group, not the window (#756).
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

var devInventoryGroupTotals = map[string]int{
	"address": 41,
}

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

func (s *server) inventoryPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	rows, err := s.store.ListAllOpenSpans(r.Context())
	if err != nil {
		s.serverError(w, "list all open spans", err)
		return
	}
	groups := buildInventory(rows)
	windowInventoryGroups(groups, r.URL.Query().Get("all"))
	if s.devMode {
		applyInventoryFixtureCounts(groups, r.URL.Query().Get("all"))
	}
	s.render(w, r, "inventory", map[string]any{
		"Title": "Inventory", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":    "inventory",
		"Groups":       groups,
		"DesignTokens": true,
		"HasData":      len(groups) > 0,
	})
}

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

func (s *server) writeInventoryExportCSV(w http.ResponseWriter, groups []inventoryGroup) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="inventory-`+s.now().UTC().Format("2006-01-02")+`.csv"`)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"type", "subject", "facet", "value", "since"})

	for _, g := range groups {
		for _, sub := range g.Subjects {
			for _, f := range sub.Facets {
				// A blank cell reads as a missing export, so a Gap is named (ADR-0072).
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
