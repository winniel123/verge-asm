package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
)

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/asset.tmpl"))

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/subjectdetail.tmpl"))

// The web binary reads the stored outcome value rather than importing the measurement leaf.

const (
	nameOutcomeNameError = "NameError"
	nameOutcomeShadowed  = "Shadowed"
)

func suppressesNameMembership(outcome string) bool {
	// A suppressed Name is a population of no current member, reached only by its own key (ADR-0072).
	return outcome == nameOutcomeNameError || outcome == nameOutcomeShadowed
}

type resolutionValue struct {
	Outcome   string   `json:"outcome"`
	Addresses []string `json:"addresses"`
}

func decodeResolution(raw []byte) resolutionValue {
	var v resolutionValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func decodeDNSRecord(raw []byte) dnsRecordValue {
	var v dnsRecordValue
	_ = json.Unmarshal(raw, &v)
	return v
}

type citationHop struct {
	Label  string
	Value  string
	Detail string
}

type servicePageData struct {
	Key                string
	CopyKey            string
	Address            string
	Port               string
	Transport          string
	Withdrawn          bool
	Reach              string
	ReachGap           bool
	ReachGapReason     string
	Citation           []citationHop
	CitationTerminated bool
	Timelines          []timelineView
	Seen               string
	InScopeSince       string
	Exposure           string
	Since              string
	Provenance         []assetKV
	Rules              []subjectRule
	Signals            []assetSignal
}

type endpointPageData struct {
	Key                string
	CopyKey            string
	Name               string
	Nameless           bool
	Service            string
	Address            string
	Port               string
	Withdrawn          bool
	Outcome            string
	Status             string
	Server             string
	Title              string
	WWWAuthenticate    string
	RedirectLocation   string
	HasIdentity        bool
	Citation           []citationHop
	CitationTerminated bool
	Timelines          []timelineView
	Seen               string
	InScopeSince       string
	Provenance         []assetKV
	Rules              []subjectRule
}

type subjectRule struct {
	Rule     string
	Version  string
	Severity string
	SevLabel string
	Fired    bool
}

type subjectPageData struct {
	Name               string
	Withdrawn          bool
	Resolution         string
	Addresses          []string
	Citation           []citationHop
	CitationTerminated bool
	Timelines          []timelineView
}

type timelineView struct {
	Facet         string
	Discriminator string
	Label         string
	Current       *spanView
	Closed        []spanView
	Breaks        []breakView
}

type spanView struct {
	Value      string
	IsGap      bool
	Open       bool
	Details    []spanDetail
	OpenedAt   string
	ClosedAt   string
	OpenedFull string
	ClosedFull string
	Reason     string
}

type spanDetail struct {
	Type string
	Data string
}

type breakView struct {
	MovedLeaves string
	At          string
}

type reachabilityValue struct {
	Outcome string `json:"outcome"`
	Result  string `json:"result"`
	Reason  string `json:"reason"`
}

const reachOutcomeGap = "gap"

func decodeReachability(raw []byte) reachabilityValue {
	var v reachabilityValue
	_ = json.Unmarshal(raw, &v)
	return v
}

type httpIdentityValue struct {
	Outcome          string `json:"outcome"`
	Status           int    `json:"status"`
	Server           string `json:"server"`
	Title            string `json:"title"`
	WWWAuthenticate  string `json:"www_authenticate"`
	RedirectLocation string `json:"redirect_location"`
}

func decodeHTTPIdentity(raw []byte) httpIdentityValue {
	var v httpIdentityValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func httpIdentityLabel(v httpIdentityValue) string {
	if v.Outcome == httpexchange.OutcomeNoHTTPResponse {
		return "no HTTP response"
	}
	if v.Status == 0 {
		return ""
	}
	label := strconv.Itoa(v.Status)
	if v.Server != "" {
		label += " · " + v.Server
	}
	return label
}

func splitEndpointKey(key string) (name, service string) {
	if i := strings.Index(key, "@"); i >= 0 {
		return key[:i], key[i+1:]
	}
	return "", key
}

func (s *server) endpointPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" {
		s.renderMissingSubject(w, r, acct, key)
		return
	}
	if s.devMode {
		if data, ok := s.endpointFixtureData(acct, key); ok {
			s.render(w, r, "endpoint", data)
			return
		}
	}
	subject, err := s.store.GetEndpointSubject(r.Context(), db.GetEndpointSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderMissingSubject(w, r, acct, key)
		return
	}
	if err != nil {
		s.serverError(w, "get endpoint subject", err)
		return
	}

	name, service := splitEndpointKey(subject.SubjectKey)
	addr, port, transport := splitServiceKey(service)
	id := decodeHTTPIdentity(subject.Value)
	data := endpointPageData{
		Key:              subject.SubjectKey,
		CopyKey:          endpointCopyKey(name, addr, port, transport),
		Name:             name,
		Nameless:         name == "",
		Service:          service,
		Address:          addr,
		Port:             port,
		Outcome:          id.Outcome,
		Status:           endpointStatusLabel(id),
		Server:           id.Server,
		Title:            id.Title,
		WWWAuthenticate:  id.WWWAuthenticate,
		RedirectLocation: id.RedirectLocation,
		HasIdentity:      id.Outcome != "",
	}
	var seedScope string
	data.Citation, data.CitationTerminated, data.Withdrawn, seedScope, data.InScopeSince = s.buildEndpointCitation(r, name, service, addr)
	data.Timelines = s.buildTimelines(r, "endpoint", subject.SubjectKey)
	if subject.ObservedAt.Valid {
		data.Seen = subject.ObservedAt.Time.UTC().Format(spanTimeFmt)
	}
	data.Provenance = subjectProvenance("endpoint", seedScope, firstSeenFromTimelines(data.Timelines))
	data.Rules = s.subjectRules(r, subject.SubjectKey)

	s.render(w, r, "endpoint", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory",
		"Endpoint":  data,
	})
}

func endpointStatusLabel(v httpIdentityValue) string {
	if v.Outcome == httpexchange.OutcomeNoHTTPResponse {
		return "no HTTP response"
	}
	if v.Status == 0 {
		return ""
	}
	return strconv.Itoa(v.Status)
}

func (s *server) buildEndpointCitation(r *http.Request, name, service, addr string) (hops []citationHop, terminated, withdrawn bool, seedScope, inScopeSince string) {
	hops = []citationHop{{Label: "Subject · Endpoint", Value: r.FormValue("key")}}
	if name != "" {
		hops = append(hops, citationHop{Label: "Named · Name", Value: name})
	} else {
		hops = append(hops, citationHop{Label: "Nameless endpoint", Value: "(no name — reached with no citing Name)"})
	}
	hops = append(hops, citationHop{Label: "On service · Service", Value: service})

	cited := false
	if citing, err := s.store.FindNameCitingAddress(r.Context(), db.FindNameCitingAddressParams{
		Address: addr, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); err == nil {
		detail := ""
		if citing.ObservedAt.Valid {
			detail = "cited since " + citing.ObservedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		hops = append(hops, citationHop{Label: "Cited by · resolution", Value: citing.SubjectKey, Detail: detail})
		cited = true
	}

	if parsed, perr := netip.ParseAddr(addr); perr == nil {
		if seed, err := s.store.FindCoveringAddressSeed(r.Context(), parsed); err == nil {
			scope := ""
			if seed.AddressCidr != nil {
				scope = seed.AddressCidr.String()
			}
			detail := ""
			if seed.CreatedByUsername != "" {
				detail = "declared by " + seed.CreatedByUsername
			}
			seedScope = "address scope " + scope
			if seed.CreatedAt.Valid {
				inScopeSince = seed.CreatedAt.Time.UTC().Format("2006-01-02")
			}
			hops = append(hops, citationHop{Label: "Declared · Seed", Value: seedScope, Detail: detail})
			terminated = true
		}
	}

	withdrawn = !cited && !terminated
	return hops, terminated, withdrawn, seedScope, inScopeSince
}

func (s *server) servicePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" {
		s.renderMissingSubject(w, r, acct, key)
		return
	}
	if s.devMode {
		if data, ok := s.serviceFixtureData(acct, key); ok {
			s.render(w, r, "service", data)
			return
		}
	}
	subject, err := s.store.GetServiceSubject(r.Context(), db.GetServiceSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderMissingSubject(w, r, acct, key)
		return
	}
	if err != nil {
		s.serverError(w, "get service subject", err)
		return
	}

	addr, port, transport := splitServiceKey(subject.SubjectKey)
	rv := decodeReachability(subject.Value)
	data := servicePageData{
		Key:       subject.SubjectKey,
		CopyKey:   serviceCopyKey(addr, port, transport),
		Address:   addr,
		Port:      port,
		Transport: transport,
		Reach:     rv.Outcome,
	}
	if rv.Outcome == reachOutcomeGap {
		// A Gap is absence of reach, so the subject states its cause in the operator's words (ADR-0104).
		data.ReachGap = true
		data.ReachGapReason = rv.Reason
	}
	var seedScope string
	data.Citation, data.CitationTerminated, data.Withdrawn, seedScope, data.InScopeSince = s.buildServiceCitation(r, addr)
	data.Timelines = s.buildTimelines(r, "service", subject.SubjectKey)
	if subject.ObservedAt.Valid {
		data.Seen = subject.ObservedAt.Time.UTC().Format(spanTimeFmt)
	}
	if !data.Withdrawn {
		data.Exposure = assetExposure(rv.Outcome, data.ReachGap)
	}
	data.Since = currentReachSince(data.Timelines)
	data.Provenance = subjectProvenance("service", seedScope, firstSeenFromTimelines(data.Timelines))
	data.Rules = s.subjectRules(r, subject.SubjectKey)
	data.Signals = s.assetSignals(r, subject.SubjectKey)

	s.render(w, r, "service", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory",
		"Service":   data,
	})
}

func serviceCopyKey(addr, port, transport string) string {
	key := addr
	if port != "" {
		key += ":" + port
	}
	if transport != "" {
		key += " " + transport
	}
	return key
}

func endpointCopyKey(name, addr, port, transport string) string {
	key := serviceCopyKey(addr, port, transport)
	if name != "" {
		return name + " " + key
	}
	return key
}

func splitServiceKey(key string) (addr, port, transport string) {
	// An IPv6 address carries its own colons, so port and transport split from the right.
	rest := key
	if i := strings.LastIndex(rest, "/"); i >= 0 {
		transport = rest[i+1:]
		rest = rest[:i]
	}
	if i := strings.LastIndex(rest, ":"); i >= 0 {
		port = rest[i+1:]
		addr = rest[:i]
	} else {
		addr = rest
	}
	return addr, port, transport
}

func (s *server) buildServiceCitation(r *http.Request, addr string) (hops []citationHop, terminated, withdrawn bool, seedScope, inScopeSince string) {
	hops = []citationHop{
		{Label: "Subject · Service", Value: r.FormValue("key")},
		{Label: "On address · Address", Value: addr},
	}

	cited := false
	if citing, err := s.store.FindNameCitingAddress(r.Context(), db.FindNameCitingAddressParams{
		Address: addr, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); err == nil {
		detail := ""
		if citing.ObservedAt.Valid {
			detail = "cited since " + citing.ObservedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		hops = append(hops, citationHop{
			Label:  "Cited by · resolution",
			Value:  citing.SubjectKey,
			Detail: detail,
		})
		cited = true
	}

	if parsed, perr := netip.ParseAddr(addr); perr == nil {
		if seed, err := s.store.FindCoveringAddressSeed(r.Context(), parsed); err == nil {
			scope := ""
			if seed.AddressCidr != nil {
				scope = seed.AddressCidr.String()
			}
			detail := ""
			if seed.CreatedByUsername != "" {
				detail = "declared by " + seed.CreatedByUsername
			}
			seedScope = "address scope " + scope
			if seed.CreatedAt.Valid {
				inScopeSince = seed.CreatedAt.Time.UTC().Format("2006-01-02")
			}
			hops = append(hops, citationHop{
				Label:  "Declared · Seed",
				Value:  seedScope,
				Detail: detail,
			})
			terminated = true
		}
	}

	// An Address is in the estate while a resolution cites it or a Seed covers it (CONTEXT.md).
	withdrawn = !cited && !terminated
	return hops, terminated, withdrawn, seedScope, inScopeSince
}

func (s *server) subjectPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.assetPage(w, r, acct)
}

func subjectProvenance(kind, seedScope, firstSeen string) []assetKV {
	var items []assetKV
	if seedScope != "" {
		items = append(items, assetKV{K: "Seed", V: seedScope})
	}
	// The derivation path is structural to the subject class, never a measured per-subject value.
	via := "dns sweep → hot scan"
	if kind == "endpoint" {
		via = "resolution × service join"
	}
	items = append(items, assetKV{K: "Via", V: via})
	if firstSeen != "" {
		items = append(items, assetKV{K: "First seen", V: firstSeen})
	}
	return items
}

func firstSeenFromTimelines(tls []timelineView) string {
	// Every OpenedAt is fixed-width UTC, so the lexicographic minimum is the earliest instant.
	best := ""
	consider := func(t string) {
		if t != "" && (best == "" || t < best) {
			best = t
		}
	}
	for _, tl := range tls {
		if tl.Current != nil {
			consider(tl.Current.OpenedAt)
		}
		for _, sp := range tl.Closed {
			consider(sp.OpenedAt)
		}
	}
	return best
}

func currentReachSince(tls []timelineView) string {
	for _, tl := range tls {
		if tl.Facet == "reachability" && tl.Current != nil {
			return tl.Current.OpenedAt
		}
	}
	return ""
}

func (s *server) subjectRules(r *http.Request, key string) []subjectRule {
	corpus, err := s.buildSignalCorpus(r)
	if err != nil {
		return nil
	}
	var out []subjectRule
	// A rule reads exactly one subject kind, so no kind filter is needed here (ADR-0024).
	for _, c := range signal.EvaluateCorpus(corpus) {
		member, fired := false, false
		for _, m := range c.Fired {
			if m.Subject == key {
				member, fired = true, true
				break
			}
		}
		if !member {
			for _, m := range c.NotFired {
				if m.Subject == key {
					member = true
					break
				}
			}
		}
		if !member {
			for _, m := range c.NotEvaluable {
				if m.Subject == key {
					member = true
					break
				}
			}
		}
		if !member {
			continue
		}
		sev, _ := signal.SeverityFor(c.Rule)
		out = append(out, subjectRule{
			Rule:     c.Rule,
			Version:  strings.TrimPrefix(c.Version.Rule, "v"),
			Severity: sev.String(),
			SevLabel: sevLabel(sev.String()),
			Fired:    fired,
		})
	}
	return out
}

const (
	hopKindAdmission   = "admission"
	hopKindObservation = "observation"
)

func nameCitationHop(cit db.GetNameCitationRow) citationHop {
	batch := " Scan · batch #" + strconv.FormatInt(cit.BatchID, 10)
	if cit.HopKind == hopKindAdmission {
		detail := "source " + cit.Source
		if cit.ObservedAt.Valid {
			detail = "admitted " + cit.ObservedAt.Time.UTC().Format("2006-01-02 15:04 UTC") + " · " + detail
		}
		return citationHop{
			Label:  "Admitted by · certificate transparency",
			Value:  "certificate transparency · " + cit.ScanKind + batch,
			Detail: detail,
		}
	}
	detail := "source " + cit.Source
	if cit.ObservedAt.Valid {
		detail = "first measured " + cit.ObservedAt.Time.UTC().Format("2006-01-02 15:04 UTC") + " · " + detail
	}
	return citationHop{
		Label:  "Introduced by · observation",
		Value:  "resolution-walk · " + cit.ScanKind + batch,
		Detail: detail,
	}
}

type nameSeedTerm struct {
	NameDomain        pgtype.Text
	CreatedAt         pgtype.Timestamptz
	CreatedByUsername string
}

func (s *server) terminatingNameSeed(r *http.Request, key string, cit db.GetNameCitationRow, citErr error) (nameSeedTerm, bool) {
	// An admission's Seed is read by id: a longer-suffix scope must not displace it (ADR-0107, #256).
	if citErr == nil && cit.HopKind == hopKindAdmission && cit.SeedID.Valid {
		seed, err := s.store.FindNameSeedByID(r.Context(), cit.SeedID.Int64)
		if err != nil {
			return nameSeedTerm{}, false
		}
		return nameSeedTerm{NameDomain: seed.NameDomain, CreatedAt: seed.CreatedAt, CreatedByUsername: seed.CreatedByUsername}, true
	}
	seed, err := s.store.FindCoveringNameSeed(r.Context(), key)
	if err != nil {
		return nameSeedTerm{}, false
	}
	return nameSeedTerm{NameDomain: seed.NameDomain, CreatedAt: seed.CreatedAt, CreatedByUsername: seed.CreatedByUsername}, true
}

func (s *server) buildCitation(r *http.Request, key string) ([]citationHop, bool) {
	hops := []citationHop{{
		Label: "Subject · Name", Value: key,
	}}

	terminated := false
	cit, citErr := s.store.GetNameCitation(r.Context(), db.GetNameCitationParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if citErr == nil {
		hops = append(hops, nameCitationHop(cit))
	}

	if seed, ok := s.terminatingNameSeed(r, key, cit, citErr); ok {
		detail := ""
		if seed.CreatedByUsername != "" {
			detail = "declared by " + seed.CreatedByUsername
		}
		if seed.CreatedAt.Valid {
			if detail != "" {
				detail += " · "
			}
			detail += seed.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		hops = append(hops, citationHop{
			Label:  "Declared · Seed",
			Value:  "name scope " + seed.NameDomain.String,
			Detail: detail,
		})
		terminated = true
	}

	return hops, terminated
}

const spanTimeFmt = "2006-01-02 15:04 UTC"

const spanFullFmt = "2006-01-02T15:04Z07:00"

func (s *server) buildTimelines(r *http.Request, kind, key string) []timelineView {
	rows, err := s.store.ListSpansForSubject(r.Context(), db.ListSpansForSubjectParams{
		SubjectKind: kind, SubjectKey: key,
	})
	if err != nil || len(rows) == 0 {
		return nil
	}

	type tlkey struct {
		facet, discriminator string
		vantage              int64
		source               string
	}
	order := []tlkey{}
	byKey := map[tlkey][]db.ListSpansForSubjectRow{}
	for _, row := range rows {
		k := tlkey{facet: row.Facet, discriminator: row.Discriminator, vantage: row.VantageID.Int64, source: row.Source}
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
		}
		byKey[k] = append(byKey[k], row)
	}

	views := make([]timelineView, 0, len(order))
	for _, k := range order {
		views = append(views, buildTimeline(k.facet, k.discriminator, byKey[k]))
	}
	return views
}

func buildTimeline(facet, discriminator string, rows []db.ListSpansForSubjectRow) timelineView {
	tv := timelineView{Facet: facet, Discriminator: discriminator, Label: timelineLabel(facet, discriminator)}

	spans := make([]drift.Span, 0, len(rows))
	for _, row := range rows {
		spans = append(spans, drift.Span{
			Value:    string(row.Value),
			IsGap:    row.IsGap,
			Vector:   decodeVector(row.Derivation),
			OpenedAt: row.OpenedAt.Time,
			ClosedAt: closedTime(row.ClosedAt),
			Reason:   drift.ClosureReason(row.ClosureReason.String),
		})
		sv := spanView{
			Value:      valueLabel(facet, row.Value, row.IsGap),
			IsGap:      row.IsGap,
			Open:       !row.ClosedAt.Valid,
			Details:    spanDetails(facet, row.Value, row.IsGap),
			OpenedAt:   row.OpenedAt.Time.UTC().Format(spanTimeFmt),
			OpenedFull: row.OpenedAt.Time.UTC().Format(spanFullFmt),
			Reason:     row.ClosureReason.String,
		}
		if row.ClosedAt.Valid {
			sv.ClosedAt = row.ClosedAt.Time.UTC().Format(spanTimeFmt)
			sv.ClosedFull = row.ClosedAt.Time.UTC().Format(spanFullFmt)
			tv.Closed = append(tv.Closed, sv)
		} else {
			cur := sv
			tv.Current = &cur
		}
	}

	for _, b := range drift.Breaks(spans) {
		tv.Breaks = append(tv.Breaks, breakView{
			MovedLeaves: strings.Join(b.MovedLeaves, ", "),
			At:          b.After.OpenedAt.UTC().Format(spanTimeFmt),
		})
	}
	return tv
}

func timelineLabel(facet, discriminator string) string {
	if discriminator != "" {
		return facet + " · " + discriminator
	}
	return facet
}

func valueLabel(facet string, raw []byte, isGap bool) string {
	if isGap {
		return "Gap"
	}
	switch facet {
	case "resolution":
		if o := decodeResolution(raw).Outcome; o != "" {
			return o
		}
		return "—"
	case "dns-record":
		rrs := decodeDNSRecord(raw).RRs
		if len(rrs) == 1 {
			return "1 record"
		}
		return strconv.Itoa(len(rrs)) + " records"
	case "reachability":
		if o := decodeReachability(raw).Outcome; o != "" {
			return o
		}
		return "—"
	case "http-identity":
		if l := httpIdentityLabel(decodeHTTPIdentity(raw)); l != "" {
			return l
		}
		return "—"
	case "certificate":
		if o := decodeCertificate(raw).Outcome; o != "" {
			return o
		}
		return "—"
	case "tls-acceptance":
		if o := decodeTLSAcceptance(raw).Outcome; o != "" {
			return o
		}
		return "—"
	default:
		return "—"
	}
}

type tlsAcceptanceValue struct {
	Outcome  string `json:"outcome"`
	Versions []struct {
		Version string   `json:"version"`
		Ciphers []string `json:"ciphers"`
	} `json:"versions"`
}

func decodeTLSAcceptance(raw []byte) tlsAcceptanceValue {
	var v tlsAcceptanceValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func spanDetails(facet string, raw []byte, isGap bool) []spanDetail {
	// An operator reads a subject's actual records here rather than a count alone (#240).
	if isGap {
		return nil
	}
	switch facet {
	case "resolution":
		addrs := decodeResolution(raw).Addresses
		if len(addrs) == 0 {
			return nil
		}
		details := make([]spanDetail, 0, len(addrs))
		for _, a := range addrs {
			details = append(details, spanDetail{Data: a})
		}
		return details
	case "dns-record":
		rrs := decodeDNSRecord(raw).RRs
		if len(rrs) == 0 {
			return nil
		}
		details := make([]spanDetail, 0, len(rrs))
		for _, rr := range rrs {
			details = append(details, spanDetail{Type: rr.Type, Data: rr.Data})
		}
		return details
	case "http-identity":
		return httpIdentityDetails(decodeHTTPIdentity(raw))
	case "certificate":
		chain := decodeCertificate(raw).Chain
		if len(chain) == 0 {
			return nil
		}
		details := make([]spanDetail, 0, len(chain))
		for i, link := range chain {
			label := "leaf"
			if i > 0 {
				label = "issuer"
			}
			details = append(details, spanDetail{Type: label, Data: link})
		}
		return details
	case "tls-acceptance":
		// TLS 1.3's three suites are the library's choice, never a measured selection.
		versions := decodeTLSAcceptance(raw).Versions
		if len(versions) == 0 {
			return nil
		}
		details := make([]spanDetail, 0, len(versions))
		for _, ver := range versions {
			data := "—"
			if len(ver.Ciphers) > 0 {
				data = strings.Join(ver.Ciphers, ", ")
			}
			details = append(details, spanDetail{Type: ver.Version, Data: data})
		}
		return details
	default:
		return nil
	}
}

func httpIdentityDetails(v httpIdentityValue) []spanDetail {
	if v.Outcome == httpexchange.OutcomeNoHTTPResponse {
		return []spanDetail{{Type: "outcome", Data: "no HTTP response"}}
	}
	var details []spanDetail
	if v.Status != 0 {
		details = append(details, spanDetail{Type: "status", Data: strconv.Itoa(v.Status)})
	}
	if v.Server != "" {
		details = append(details, spanDetail{Type: "server", Data: v.Server})
	}
	if v.Title != "" {
		details = append(details, spanDetail{Type: "title", Data: v.Title})
	}
	if v.WWWAuthenticate != "" {
		details = append(details, spanDetail{Type: "www-authenticate", Data: v.WWWAuthenticate})
	}
	if v.RedirectLocation != "" {
		details = append(details, spanDetail{Type: "location", Data: v.RedirectLocation})
	}
	return details
}

type assetPageData struct {
	Key          string
	Type         string
	Withdrawn    bool
	Seen         string
	InScopeSince string
	Severity     string
	SevLabel     string
	Exposure     string
	Ports        []assetPort
	DNS          []assetDNSRow
	Cert         *assetCert
	Provenance   []assetKV
	Signals      []assetSignal
	Drift        []assetDriftEvent
}

type assetPort struct {
	Port     string
	Service  string
	Exposure string
	Since    string
}

type assetCert struct {
	Name        string
	Issuer      string
	Algorithm   string
	NotAfter    string
	Label       string
	Tone        string
	Fingerprint string
}

type assetDNSRow struct {
	Type  string
	Value string
	Seen  string
}

type assetKV struct {
	K string
	V string
}

type assetSignal struct {
	Rule     string
	Subject  string
	Severity string
	SevLabel string
	SigID    string
	Time     string
}

type assetDriftEvent struct {
	Change  string
	Family  string
	Subject string
	Detail  string
	Time    string
}

func (s *server) assetPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode && r.PathValue("key") == devAssetKey {
		s.render(w, r, "asset", s.assetFixtureData(acct))
		return
	}
	key := r.PathValue("key")
	subject, err := s.store.GetNameSubject(r.Context(), db.GetNameSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderMissingSubject(w, r, acct, key)
		return
	}
	if err != nil {
		s.serverError(w, "get name subject", err)
		return
	}

	res := decodeResolution(subject.Value)
	data := assetPageData{
		Key:       subject.SubjectKey,
		Type:      "Name",
		Withdrawn: suppressesNameMembership(res.Outcome),
	}
	if subject.ObservedAt.Valid {
		data.Seen = subject.ObservedAt.Time.UTC().Format(spanTimeFmt)
	}
	data.Provenance, data.InScopeSince = s.assetProvenance(r, key)
	data.DNS = s.assetDNS(r, key, res)
	data.Ports = s.assetPorts(r, res.Addresses)
	data.Cert = s.assetCertificate(r, key, res.Addresses)
	data.Signals = s.assetSignals(r, key)
	data.Severity = assetHeaderSeverity(data.Signals)
	data.SevLabel = sevLabel(data.Severity)
	data.Exposure = assetHeaderExposure(data.Ports)
	data.Drift = assetDrift(s.buildTimelines(r, "name", key))

	s.render(w, r, "asset", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory",
		"Asset":     data,
	})
}

func (s *server) assetProvenance(r *http.Request, key string) (items []assetKV, inScopeSince string) {
	cit, citErr := s.store.GetNameCitation(r.Context(), db.GetNameCitationParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if seed, ok := s.terminatingNameSeed(r, key, cit, citErr); ok {
		if seed.NameDomain.Valid {
			items = append(items, assetKV{K: "Seed", V: seed.NameDomain.String})
		}
		if seed.CreatedAt.Valid {
			inScopeSince = seed.CreatedAt.Time.UTC().Format("2006-01-02")
		}
	}
	if citErr == nil {
		via := "resolution-walk"
		if cit.HopKind == hopKindAdmission {
			via = "certificate transparency"
		}
		if cit.ScanKind != "" {
			via += " · " + cit.ScanKind
		}
		items = append(items, assetKV{K: "Via", V: via})
		if cit.Source != "" {
			items = append(items, assetKV{K: "Source", V: cit.Source})
		}
		if cit.ObservedAt.Valid {
			items = append(items, assetKV{K: "First seen", V: cit.ObservedAt.Time.UTC().Format("2006-01-02")})
		}
	}
	return items, inScopeSince
}

func (s *server) assetDNS(r *http.Request, key string, res resolutionValue) []assetDNSRow {
	var rows []assetDNSRow
	for _, a := range res.Addresses {
		t := "A"
		if ip, err := netip.ParseAddr(a); err == nil && ip.Is6() {
			t = "AAAA"
		}
		rows = append(rows, assetDNSRow{Type: t, Value: a})
	}
	dnsRows, err := s.store.ListNameDNSRecords(r.Context(), db.ListNameDNSRecordsParams{
		AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if err == nil {
		for _, row := range dnsRows {
			if row.SubjectKey != key {
				continue
			}
			for _, rr := range decodeDNSRecord(row.Value).RRs {
				switch strings.ToUpper(rr.Type) {
				case "A", "AAAA":
					continue
				}
				rows = append(rows, assetDNSRow{Type: rr.Type, Value: rr.Data})
			}
		}
	}
	return rows
}

func (s *server) assetPorts(r *http.Request, addresses []string) []assetPort {
	if len(addresses) == 0 {
		return nil
	}
	addrSet := make(map[string]bool, len(addresses))
	for _, a := range addresses {
		addrSet[a] = true
	}
	rows, err := s.store.ListAllOpenSpans(r.Context())
	if err != nil {
		return nil
	}
	servers := map[string]string{}
	for _, row := range rows {
		if row.SubjectKind != "endpoint" || row.Facet != "http-identity" {
			continue
		}
		_, service := splitEndpointKey(row.SubjectKey)
		addr, port, _ := splitServiceKey(service)
		if !addrSet[addr] {
			continue
		}
		if srv := decodeHTTPIdentity(row.Value).Server; srv != "" {
			servers[addr+":"+port] = srv
		}
	}
	var ports []assetPort
	for _, row := range rows {
		if row.SubjectKind != "service" || row.Facet != "reachability" {
			continue
		}
		addr, port, transport := splitServiceKey(row.SubjectKey)
		if !addrSet[addr] {
			continue
		}
		ports = append(ports, assetPort{
			Port:     ":" + port,
			Service:  assetPortService(transport, servers[addr+":"+port]),
			Exposure: assetExposure(decodeReachability(row.Value).Outcome, row.IsGap),
			Since:    row.OpenedAt.Time.UTC().Format(spanTimeFmt),
		})
	}
	return ports
}

func assetPortService(transport, server string) string {
	// The Server is stored http-identity evidence, never a new fingerprint (ADR-0110).
	if server == "" {
		return transport
	}
	return transport + " · " + server
}

func assetExposure(outcome string, isGap bool) string {
	// Undiscriminated reach is a Gap, never an exposure verdict (ADR-0104).
	if isGap {
		return "unverified"
	}
	// A firewall is indistinguishable from silence, so the honest negative is not-reached.
	switch outcome {
	case "reached":
		return "exposed"
	case "not-reached":
		return "not-reached"
	default:
		return "unverified"
	}
}

// A pre-parse span stored only chain and not_after, so issuer and algorithm read empty.

type certificateLeafValue struct {
	Outcome   string   `json:"outcome"`
	Chain     []string `json:"chain"`
	NotAfter  string   `json:"not_after"`
	Issuer    string   `json:"issuer"`
	Algorithm string   `json:"algorithm"`
}

func (s *server) assetCertificate(r *http.Request, key string, addresses []string) *assetCert {
	rows, err := s.store.ListEndpointCertificates(r.Context(), db.ListEndpointCertificatesParams{
		AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return nil
	}
	addrSet := make(map[string]bool, len(addresses))
	for _, a := range addresses {
		addrSet[a] = true
	}
	var chosen *certificateLeafValue
	for _, row := range rows {
		name, service := splitEndpointKey(row.SubjectKey)
		addr, _, _ := splitServiceKey(service)
		// The presented chain is single-valued only under an Endpoint key (CONTEXT.md).
		named := name == key
		if !named && !(name == "" && addrSet[addr]) {
			continue
		}
		var v certificateLeafValue
		if err := json.Unmarshal(row.Value, &v); err != nil {
			continue
		}
		if v.Outcome != "presented" || len(v.Chain) == 0 {
			continue
		}
		vv := v
		if named {
			chosen = &vv
			break
		}
		if chosen == nil {
			chosen = &vv
		}
	}
	if chosen == nil {
		return nil
	}
	cert := &assetCert{
		Name:        key,
		Issuer:      chosen.Issuer,
		Algorithm:   chosen.Algorithm,
		Fingerprint: chosen.Chain[0],
	}
	if chosen.NotAfter != "" {
		if na, perr := time.Parse(time.RFC3339, chosen.NotAfter); perr == nil {
			cert.NotAfter = na.UTC().Format("2006-01-02")
			cert.Label, cert.Tone = certValidity(na, s.now().UTC())
		}
	}
	return cert
}

func certValidity(notAfter, now time.Time) (label, tone string) {
	days := int(notAfter.Sub(now).Hours() / 24)
	switch {
	case days < 0:
		return "expired " + strconv.Itoa(-days) + "d ago", "danger"
	case days <= 30:
		return "expires in " + strconv.Itoa(days) + "d", "warn"
	default:
		return "valid · " + strconv.Itoa(days) + "d", "ok"
	}
}

func assetHeaderSeverity(signals []assetSignal) string {
	best := ""
	bestRank := len(signal.SevOrder)
	for _, sg := range signals {
		if rank := signal.Severity(sg.Severity).Rank(); rank < bestRank {
			bestRank, best = rank, sg.Severity
		}
	}
	return best
}

func assetHeaderExposure(ports []assetPort) string {
	rank := map[string]int{"exposed": 0, "firewalled": 1, "not-reached": 2, "unverified": 3}
	best := ""
	bestRank := len(rank)
	for _, p := range ports {
		r, ok := rank[p.Exposure]
		if ok && r < bestRank {
			bestRank, best = r, p.Exposure
		}
	}
	return best
}

func (s *server) assetSignals(r *http.Request, key string) []assetSignal {
	corpus, err := s.buildSignalCorpus(r)
	if err != nil {
		return nil
	}
	instances, err := s.deriveSignalInstances(r.Context(), signal.EvaluateCorpus(corpus))
	if err != nil {
		return nil
	}
	now := s.now().UTC()
	var out []assetSignal
	for _, inst := range instances {
		if inst.Asset != key {
			continue
		}
		sig := assetSignal{
			Rule:     inst.Signal,
			Subject:  inst.Asset,
			Severity: inst.Severity,
			SevLabel: sevLabel(inst.Severity),
			SigID:    inst.SigID,
		}
		if inst.First != "" {
			if t, perr := time.Parse(time.RFC3339, inst.First); perr == nil {
				sig.Time = relTime(t, now)
			}
		}
		out = append(out, sig)
	}
	return out
}

func assetDrift(timelines []timelineView) []assetDriftEvent {
	var out []assetDriftEvent
	for _, tl := range timelines {
		var change, when, detail string
		switch {
		case tl.Current != nil && len(tl.Closed) == 0:
			change, when, detail = "appeared", tl.Current.OpenedAt, tl.Current.Value
		case tl.Current != nil:
			change, when, detail = "changed", tl.Current.OpenedAt, tl.Current.Value
		case len(tl.Closed) > 0:
			last := tl.Closed[len(tl.Closed)-1]
			change, when, detail = "withdrawn", last.ClosedAt, last.Value
		default:
			continue
		}
		out = append(out, assetDriftEvent{
			Change:  change,
			Family:  driftFamily(change),
			Subject: tl.Label,
			Detail:  detail,
			Time:    when,
		})
	}
	return out
}

func decodeVector(raw []byte) drift.Vector {
	var comps []drift.Component
	_ = json.Unmarshal(raw, &comps)
	return drift.NewVector(comps...)
}

func closedTime(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}
