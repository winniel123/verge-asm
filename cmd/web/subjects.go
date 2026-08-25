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

// The frozen design-owned asset.tmpl (design-system/templates/asset.tmpl, package
// v3.10.0) is the view layer for /asset/{key}: it defines "asset" + "assetexposure"
// and reuses the "sevbadge" define signals.tmpl declares and the "changeglyph" define
// drift.tmpl declares — all parse into the one shared `tmpl` set, so they resolve at
// execute time. It is embedded read-only via the designfs package (auto-globbed
// through `templates/*.tmpl`); the repo authors no markup/CSS/JS for this route.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/asset.tmpl"))

// The Subjects screen (v1 spec §6.6, ADR-0072). At wave-0 only `Name` subjects
// exist — they come from the `resolution-walk` leaf's `resolution` facet (#188).
// The listing is the estate alone: every current Name, searchable, with **no
// denominator** — estate completeness is unmeasurable and refusing to state it
// is the model's closest analogue to honesty. A withdrawn Name is not a row; it
// is reached by its own key on the drill-down, marked as naming a population of
// no current member. Address/Service/Endpoint arrive with later measurement
// tickets and have no surface here yet.

// nameOutcomeNameError and nameOutcomeShadowed are the two resolution outcomes
// that suppress a Name's membership: resolution-walk measuring a Name Error (the
// name does not exist) and wildcard-discrimination reading Shadowed (a
// wildcard-synthesised answer), which suppress a Name as affirmatively as each
// other (#192; ADR-0006, ADR-0086). Kept as local constants so the web binary
// reads the stored value without importing the leaves.
const (
	nameOutcomeNameError = "NameError"
	nameOutcomeShadowed  = "Shadowed"
)

// suppressesNameMembership reports whether a latest resolution outcome takes a
// Name out of the estate — the drill-down renders such a Name as a population of
// no current member (ADR-0072).
func suppressesNameMembership(outcome string) bool {
	return outcome == nameOutcomeNameError || outcome == nameOutcomeShadowed
}

// resolutionValue is the JSON payload of a resolution observation, the shape the
// resolution-walk leaf emits. The web layer reads only the fields it renders.
type resolutionValue struct {
	Outcome   string   `json:"outcome"`
	Addresses []string `json:"addresses"`
}

func decodeResolution(raw []byte) resolutionValue {
	var v resolutionValue
	_ = json.Unmarshal(raw, &v)
	return v
}

// decodeDNSRecord decodes a dns-record observation's value into the shared
// dnsRecordValue shape (declared in signals.go). The Subjects drill-down reads
// each RR's answered type and canonical data off it — the count for the collapsed
// summary, the type+data rows on expand — mirroring decodeResolution so the
// dns-record shape is parsed one way across the package.
func decodeDNSRecord(raw []byte) dnsRecordValue {
	var v dnsRecordValue
	_ = json.Unmarshal(raw, &v)
	return v
}

// citationHop is one link in a subject's "why is this here" chain, rendered
// top-to-bottom from the subject down to the Seed the chain terminates at.
type citationHop struct {
	Label  string // the micro-label: what kind of hop this is
	Value  string // the load-bearing value, rendered mono
	Detail string // optional muted qualifier
}

// servicePageData is the drill-down view for one Service subject.
type servicePageData struct {
	Key       string
	Address   string
	Port      string
	Transport string
	// Withdrawn reports that the Service's Address has left the estate — no
	// current resolution cites it and no Seed covers it — so the Service is a
	// population of no current member, reached only by its own key (ADR-0072).
	Withdrawn bool
	Reach     string
	// ReachGap reports the current reach is a `Gap` — a blanket responder answering
	// on every port, or a control probe that could not complete (ADR-0104) — and
	// ReachGapReason renders its cause in the operator's words. A Gap is the absence
	// of a reach value, not `not-reached`, so the page states *we cannot see your
	// origin from here* rather than *nothing is open*.
	ReachGap       bool
	ReachGapReason string
	// Citation is the "why is this here" chain: Service → Address → the Name whose
	// resolution cites the Address (or the address-scope Seed that covers it),
	// terminating at a Seed.
	Citation           []citationHop
	CitationTerminated bool
	// Timelines are the Service's reachability Span timelines — current and closed
	// — with Breaks and closures derived on read (#195). A Service opening or
	// closing across a re-run of the hot Scan is one Span transition here.
	Timelines []timelineView
	// Header identity + rail data ported from SubjectDetail.jsx (U1, #478): the last
	// observation instant, the covering Seed's declaration date, the reachability
	// rolled up to an exposure state, and the current reachability span's open time.
	Seen         string
	InScopeSince string
	Exposure     string // reachability → exposure state (assetExposure); empty when withdrawn/unmeasured
	Since        string // the current reachability span's OpenedAt
	Provenance   []assetKV
	// Rules are every rule whose predicate domain includes this Service, each with
	// its own versioned verdict (fired / did not fire) and the rule's SeverityBadge.
	Rules []subjectRule
	// Signals are the rules firing on this Service right now (the rail's "Signals
	// here"), each carrying its rule's severity — the same fired census the asset
	// drill-in reads, filtered to this subject's key.
	Signals []assetSignal
}

// endpointPageData is the drill-down view for one Endpoint subject (#198): the
// (Name, Service) pair the http-exchange leaf's http-identity facet is held on.
type endpointPageData struct {
	Key      string
	Name     string // empty for the nameless endpoint
	Nameless bool
	Service  string
	Address  string
	Port     string
	// Withdrawn reports that the Endpoint's Service has left the estate — its
	// Address is no longer cited by any resolution nor covered by a Seed — so the
	// Endpoint names a population of no current member, reached only by its own key
	// (ADR-0072). An Endpoint closes when either leg withdraws (CONTEXT.md
	// `Endpoint`).
	Withdrawn bool
	// The current HTTP identity, decoded for display — the admitted closed set
	// (ADR-0011): outcome, status, Server, page <title>, WWW-Authenticate challenge,
	// and the recorded redirect Location. No body hash, length, or Content-Type.
	Outcome          string
	Status           string
	Server           string
	Title            string
	WWWAuthenticate  string
	RedirectLocation string
	HasIdentity      bool
	// Citation is the "why is this here" chain: Endpoint → its Name leg → its
	// Service leg → the Address → the Seed the chain terminates at.
	Citation           []citationHop
	CitationTerminated bool
	// Timelines are the Endpoint's http-identity Span timelines — current and
	// closed — with Breaks and closures derived on read (#198).
	Timelines []timelineView
	// Header identity + rail data ported from SubjectDetail.jsx (U1, #478).
	Seen         string
	InScopeSince string
	Provenance   []assetKV
	// Rules are every rule whose predicate domain includes this Endpoint, each with
	// its own versioned verdict and the rule's SeverityBadge.
	Rules []subjectRule
}

// subjectRule is one rule whose predicate domain includes a Service or Endpoint
// subject, as the "Rules over this subject" table renders it (SubjectDetail.jsx):
// the rule slug, its own version, its five-level SeverityBadge (internal/signal
// SeverityFor, a real per-rule datum), and its current verdict — Fired, else "did
// not fire" (a NotEvaluable member is in the domain but the rule could not read
// its evidence, which the operator-facing table folds into "did not fire"). Every
// field is read from the current census, never fabricated.
type subjectRule struct {
	Rule     string
	Version  string // the rule's own version (Census.Version.Rule, e.g. "v1")
	Severity string // the rule's severity token: critical | high | medium | low | info
	Fired    bool
}

// subjectPageData is the drill-down view for one Name.
type subjectPageData struct {
	Name       string
	Withdrawn  bool
	Resolution string
	Addresses  []string
	Citation   []citationHop
	// CitationTerminated reports whether the chain reached a Seed. It always
	// should for a measured Name; a false here is an integrity gap worth showing
	// rather than hiding.
	CitationTerminated bool
	// Timelines are the subject's Span timelines — current and closed — for the
	// resolution and dns-record facets, each with its Breaks and closures derived
	// on read (#190). Empty where no Span has been folded yet.
	Timelines []timelineView
}

// timelineView is one (facet, discriminator, vantage, source) Span timeline
// rendered for the drill-down: its current value if it holds one, its closed
// history, and the Breaks between spans of differing Derivation vectors, all
// derived on read (never stored). A timeline whose last span is closed with no
// successor is a withdrawn/gapped timeline and carries no current span.
type timelineView struct {
	Facet         string
	Discriminator string
	Label         string
	Current       *spanView
	Closed        []spanView
	Breaks        []breakView
}

// spanView is one Span rendered: its value (or a Gap marker), the period it
// spanned, and — where it is a withdrawal's closing side — the ground the closure
// rests on.
type spanView struct {
	Value string
	IsGap bool
	Open  bool
	// Details are the span value's individual records, listed on expand: one row
	// per RR (dns-record) or per address (resolution). The collapsed row keeps its
	// change-first summary (`Value`); the drill-down expands to these so an operator
	// reads a subject's actual records without DB access (#240). Empty for facets
	// with no per-item breakdown, in which case the row does not expand.
	Details  []spanDetail
	OpenedAt string
	ClosedAt string
	Reason   string
}

// spanDetail is one row of a span value's expanded contents: an RR (its type and
// data) for a dns-record span, or a single address (typeless) for a resolution
// span.
type spanDetail struct {
	Type string // the answered RR type ("A", "TXT", …); empty for a resolution address
	Data string // the record's canonical data, or the address, as stored
}

// breakView is one Break between two spans, naming the leaf that moved. Derived
// on read from the two spans' vectors; never stored.
type breakView struct {
	MovedLeaves string
	At          string
}

// reachabilityValue is the JSON payload of a reachability observation — the
// verdict and the raw connect result as evidence, plus (on a `Gap`) the
// operator-facing reason a blanket responder's reach carries (ADR-0104). The web
// layer reads the verdict and, where the reach is a Gap, the reason it renders on
// the subject; the stored sixth-cause tag is not read here (the reason is the
// operator-facing rendering), so it is not decoded.
type reachabilityValue struct {
	Outcome string `json:"outcome"`
	Result  string `json:"result"`
	Reason  string `json:"reason"`
}

// reachOutcomeGap is the `outcome` tag a reachability `Gap` carries — a blanket
// responder's undiscriminated reach (ADR-0104). Kept as a local constant so the
// web binary reads the stored value without importing the measurement leaf.
const reachOutcomeGap = "gap"

func decodeReachability(raw []byte) reachabilityValue {
	var v reachabilityValue
	_ = json.Unmarshal(raw, &v)
	return v
}

// httpIdentityValue is the JSON payload of an http-identity observation — the
// shape the http-exchange leaf emits (#198). It is a closed union of admitted
// fields (ADR-0011): the outcome (`responded` | `no-http-response`), the status,
// the Server header, the page <title>, the WWW-Authenticate challenge, and the
// recorded redirect Location. The body itself is never stored — no hash, no
// length, no Content-Type.
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

// httpIdentityLabel renders a short one-line identity for the listing: the status
// code and, where present, the Server header (e.g. `200 · nginx`). A
// no-http-response Endpoint — reached but speaking no HTTP — renders that negative
// as a value rather than an em dash; an empty (unmeasured) value renders nothing.
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

// splitEndpointKey parses a `name@service` Endpoint key into its Name and Service
// legs, splitting at the FIRST `@`. Neither a DNS Name nor a Service key contains
// an `@`, so the split is unambiguous. A key beginning with `@` is the nameless
// endpoint — an empty Name leg, the distinguished nameless variant — never an
// empty name masquerading as a named one.
func splitEndpointKey(key string) (name, service string) {
	if i := strings.Index(key, "@"); i >= 0 {
		return key[:i], key[i+1:]
	}
	return "", key
}

// endpointPage is the drill-down for one Endpoint subject (#198). The Endpoint key
// carries a `/` and an `@`, so it arrives as a `?key=` query parameter rather than
// a path segment. It renders the current HTTP identity, the Citation chain back to
// a Seed through the Endpoint's Name and Service legs, and the http-identity Span
// timelines the hot Scan folds.
func (s *server) endpointPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" {
		s.renderMissingSubject(w, acct, key)
		return
	}
	subject, err := s.store.GetEndpointSubject(r.Context(), db.GetEndpointSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderMissingSubject(w, acct, key)
		return
	}
	if err != nil {
		s.serverError(w, "get endpoint subject", err)
		return
	}

	name, service := splitEndpointKey(subject.SubjectKey)
	addr, port, _ := splitServiceKey(service)
	id := decodeHTTPIdentity(subject.Value)
	data := endpointPageData{
		Key:              subject.SubjectKey,
		Name:             name,
		Nameless:         name == "",
		Service:          service,
		Address:          addr,
		Port:             port,
		Outcome:          id.Outcome,
		Status:           httpIdentityLabel(id),
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

	s.render(w, "endpoint", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory",
		"Endpoint":  data,
	})
}

// buildEndpointCitation assembles the "why is this here" chain for an Endpoint:
// the Endpoint itself, its Name leg (or a nameless marker), its Service leg, and
// then the Address membership limbs the Service rests on — the Name whose current
// resolution cites the Address, or the address-scope Seed that covers it,
// terminating at a Seed. Where neither limb holds, the Address (and with it the
// Service and Endpoint) has left the estate. It reuses the same address-membership
// store reads the Service citation does, so the two chains agree on the ground.
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

// servicePage is the drill-down for one Service subject (#195). The Service key
// carries a `/`, so it arrives as a `?key=` query parameter rather than a path
// segment. It renders the current reachability verdict, the Citation chain back
// to a Seed, and the reachability Span timelines the hot Scan folds.
func (s *server) servicePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" {
		s.renderMissingSubject(w, acct, key)
		return
	}
	subject, err := s.store.GetServiceSubject(r.Context(), db.GetServiceSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderMissingSubject(w, acct, key)
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
		Address:   addr,
		Port:      port,
		Transport: transport,
		Reach:     rv.Outcome,
	}
	if rv.Outcome == reachOutcomeGap {
		// A blanket responder (or an undiscriminated reach): the reach is a Gap, so
		// the subject states the proxy-edge finding in prose rather than a verdict.
		data.ReachGap = true
		data.ReachGapReason = rv.Reason
	}
	var seedScope string
	data.Citation, data.CitationTerminated, data.Withdrawn, seedScope, data.InScopeSince = s.buildServiceCitation(r, addr)
	data.Timelines = s.buildTimelines(r, "service", subject.SubjectKey)
	if subject.ObservedAt.Valid {
		data.Seen = subject.ObservedAt.Time.UTC().Format(spanTimeFmt)
	}
	// The header ExposureBadge rolls the current reachability up to an exposure
	// state (assetExposure), the same read the asset census carries; a withdrawn
	// Service names no current member, so it shows no exposure (the header marks it
	// withdrawn instead).
	if !data.Withdrawn {
		data.Exposure = assetExposure(rv.Outcome, data.ReachGap)
	}
	data.Since = currentReachSince(data.Timelines)
	data.Provenance = subjectProvenance("service", seedScope, firstSeenFromTimelines(data.Timelines))
	data.Rules = s.subjectRules(r, subject.SubjectKey)
	data.Signals = s.assetSignals(r, subject.SubjectKey)

	s.render(w, "service", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory",
		"Service":   data,
	})
}

// splitServiceKey parses an `address:port/transport` Service key into its parts,
// rendering the Address, port and transport for display. It splits the transport
// off the right of the `/` and the port off the right of the last `:`, so an
// IPv6 address (which itself carries `:`) keeps its own colons.
func splitServiceKey(key string) (addr, port, transport string) {
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

// buildServiceCitation assembles the "why is this here" chain for a Service: the
// Service itself, the Address the triple sits on, and the ground the Address's
// membership rests on — the Name whose current resolution cites the Address, or
// the address-scope Seed that covers it, terminating at a Seed. Where neither
// limb holds, the Address has left the estate (the `uncited` / `descoped`
// departure), which the caller renders as a withdrawn Service.
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

	// The covering Seed terminates the chain. An address-scope Seed covers the
	// Address directly; failing that, the Name that cites it traces to a name
	// scope, which we surface via the Address's citing Name.
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

	// The Address is in the estate exactly while a current resolution cites it or
	// a Seed covers it (CONTEXT.md `Address`). Neither limb holding means it has
	// withdrawn — its Services with it.
	withdrawn = !cited && !terminated
	return hops, terminated, withdrawn, seedScope, inScopeSince
}

// subjectPage serves the by-key drill-down at `/subjects/{key}`. The key here
// carries no `/` or `@`, so it is a Name — and a Name opens the Asset detail
// (SubjectDetail.jsx: "A Name subject opens AssetDetail instead; this screen
// covers the other two kinds"). The Service and Endpoint drill-ins have their own
// `?key=` routes (servicePage / endpointPage), so this path is Name-only. It
// delegates to assetPage, which reads the same live-tier-gated Name subject: a
// missing/evidential Name still 404s the subject-missing page, and a withdrawn
// Name still renders reachable by its own key — the semantics this route has
// always carried, now on the AssetDetail surface (#478, U1).
func (s *server) subjectPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.assetPage(w, r, acct)
}

// subjectProvenance assembles the rail's "how it got here" facts for a Service or
// Endpoint (SubjectDetail.jsx): the covering Seed the citation terminates at, the
// fixed derivation path the subject class is formed by, and when it was first
// measured. Only real facts render — a Seed or first-seen with no honest source is
// omitted rather than invented (the asset drill-in's "render only what exists"
// pattern, T1), and no per-subject Vantage is fabricated where the domain carries
// only an opaque vantage id.
func subjectProvenance(kind, seedScope, firstSeen string) []assetKV {
	var items []assetKV
	if seedScope != "" {
		items = append(items, assetKV{K: "Seed", V: seedScope})
	}
	// The derivation path is a fixed structural fact about the subject class, not a
	// measured per-subject value: a Service is an address swept from DNS then reached
	// by the hot Scan; an Endpoint is the join of a resolution and a Service.
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

// firstSeenFromTimelines returns the earliest span open instant across a subject's
// timelines — its "first seen". Every OpenedAt is the same fixed-width UTC format,
// so the lexicographically smallest is the chronologically earliest; empty where no
// span has been folded.
func firstSeenFromTimelines(tls []timelineView) string {
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

// currentReachSince returns the open instant of the current reachability span — the
// "Since" the Service's current-facet card carries. Empty where the reachability
// timeline holds no current value (a withdrawn or gapped Service).
func currentReachSince(tls []timelineView) string {
	for _, tl := range tls {
		if tl.Facet == "reachability" && tl.Current != nil {
			return tl.Current.OpenedAt
		}
	}
	return ""
}

// subjectRules lists every rule whose predicate domain includes the given subject
// key — the "Rules over this subject" table (SubjectDetail.jsx). It folds the same
// signal corpus the Signals page reads and keeps the censuses this subject is a
// member of, in EvaluateCorpus order, each carrying its rule version, its verdict
// (Fired, else "did not fire"), and its per-rule Severity (internal/signal
// SeverityFor). The engine is split by subject kind, so a Service key only appears
// in Service-rule censuses and an Endpoint key only in Endpoint-rule censuses — no
// kind filter is needed. Best-effort: a corpus-build failure yields no rules.
func (s *server) subjectRules(r *http.Request, key string) []subjectRule {
	corpus, err := s.buildSignalCorpus(r)
	if err != nil {
		return nil
	}
	var out []subjectRule
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
		out = append(out, subjectRule{Rule: c.Rule, Version: c.Version.Rule, Severity: sev.String(), Fired: fired})
	}
	return out
}

// hopKindAdmission and hopKindObservation are the two ways a Name enters, as
// GetNameCitation's reconciled `hop_kind` reports them (ADR-0107). Named here so
// the Go side reads one contract with the SQL literals rather than repeating the
// string at every comparison and fixture.
const (
	hopKindAdmission   = "admission"
	hopKindObservation = "observation"
)

// nameCitationHop renders the introducing hop of a Name's Citation, reconciled
// per ADR-0107: a CT `admission` hop reads as certificate transparency admitting
// the Name (the CT Batch that introduced it, ADR-0027), while an `observation` hop
// reads as the resolution that first measured it. Membership is measured either
// way; the admission is why the Name is here, the observation is that it is.
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

// nameSeedTerm is the Seed a Name's Citation chain terminates at, reduced to the
// three fields the hop renders. It lets buildCitation treat the two ways the Seed
// is found — by the admitted_name row's id, or by longest-suffix cover — through
// one shape.
type nameSeedTerm struct {
	NameDomain        pgtype.Text
	CreatedAt         pgtype.Timestamptz
	CreatedByUsername string
}

// terminatingNameSeed picks the Seed a Name's Citation chain bottoms out at. A CT
// admission terminates at the Seed the admitted_name row itself carries (ADR-0027,
// #256) — read by id so an overlapping longer-suffix scope cannot displace the Seed
// the admission provenance names. Every other Name — an observation hop, or one
// with no citation at all — terminates at its covering Seed by the longest-suffix
// match. Best-effort: a lookup failure degrades to no Seed hop rather than falling
// back to the suffix match for an admission, which is the very mismatch #256 fixes.
func (s *server) terminatingNameSeed(r *http.Request, key string, cit db.GetNameCitationRow, citErr error) (nameSeedTerm, bool) {
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

// buildCitation assembles the "why is this here" chain for a Name: the subject
// itself, the hop that introduced it (a CT admission or a resolution, ADR-0107),
// and the Seed the chain terminates at. Every hop is best-effort — a missing hop
// degrades to a shorter chain rather than a 500, since the card is diagnostic and
// a partial answer still helps. It reports whether the chain reached a Seed.
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

// buildTimelines reads the subject's Span corpus and assembles one view per
// (facet, discriminator) timeline: its current span if it holds one, its closed
// history, and the Breaks between spans of differing Derivation vectors — the
// last two derived on read, never stored (ADR-0007, ADR-0008). A withdrawn Name's
// timelines are all closed, and the closed corpus is never compacted, so they
// render in full. A best-effort read: a failure degrades to no timelines rather
// than a 500, since the drill-down is diagnostic.
func (s *server) buildTimelines(r *http.Request, kind, key string) []timelineView {
	rows, err := s.store.ListSpansForSubject(r.Context(), db.ListSpansForSubjectParams{
		SubjectKind: kind, SubjectKey: key,
	})
	if err != nil || len(rows) == 0 {
		return nil
	}

	// Group spans into their timelines, preserving the query's (facet,
	// discriminator, vantage, source, opened_at) order.
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
			Value:    valueLabel(facet, row.Value, row.IsGap),
			IsGap:    row.IsGap,
			Open:     !row.ClosedAt.Valid,
			Details:  spanDetails(facet, row.Value, row.IsGap),
			OpenedAt: row.OpenedAt.Time.UTC().Format(spanTimeFmt),
			Reason:   row.ClosureReason.String,
		}
		if row.ClosedAt.Valid {
			sv.ClosedAt = row.ClosedAt.Time.UTC().Format(spanTimeFmt)
			tv.Closed = append(tv.Closed, sv)
		} else {
			cur := sv
			tv.Current = &cur
		}
	}

	// Breaks are derived on read from the spans' vectors and name the moved leaf.
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

// valueLabel renders a span's value as the collapsed one-line summary shared by
// the drill-down timelines and the Inventory axis (#243, ADR-0105): a resolution's
// outcome tag, a dns-record's record count, a reachability/certificate/tls-acceptance
// outcome, an http-identity's status+Server line, and a Gap marker where the span
// holds no value. It is the summary half; spanDetails is the expanded half, and the
// two cover the same facets.
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

// tlsAcceptanceValue is the JSON payload of a tls-acceptance observation — the
// closed union the tls-acceptance leaf emits (ADR-0011): the outcome tag
// (`enumerated` │ `tls-refused` │ `no-tls`) and, only on an enumeration, the
// accepted versions and (for TLS 1.0–1.2) the suites the listener selected. The
// web layer reads only what it renders — the outcome for the summary, the versions
// for the inventory expansion.
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

// spanDetails lists a span value's individual records for the expand-on-click
// affordance shared by the drill-down timelines and the Inventory axis: one entry
// per RR (type + data) for a dns-record span, one per address for a resolution
// span, one per admitted field for an http-identity span, and one per chain link
// for a certificate span. Values come from the span's already-read value JSON —
// the same bytes valueLabel summarises — so no new query is introduced (#240 for
// the first two facets, #243/ADR-0105 for the rest). A Gap holds no value, and a
// facet with no per-item breakdown (e.g. reachability, whose whole value is its
// outcome) expands to nothing and the row does not open.
func spanDetails(facet string, raw []byte, isGap bool) []spanDetail {
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
		// One row per accepted version — its suites (in the listener's own
		// preference order) as the data, or "—" for TLS 1.3 whose three suites are
		// the library's choice, not a measured selection. A refusal or no-tls carries
		// no versions, so its whole value is the outcome summary and it does not open.
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

// httpIdentityDetails lists the admitted closed set of an http-identity value as
// expand rows (ADR-0011): the status, the Server header, the page <title>, the
// WWW-Authenticate challenge, and the recorded redirect Location — each rendered
// only where present. A no-http-response identity — reached but speaking no HTTP —
// lists that outcome as its one row rather than expanding to nothing, so the
// negative is itself inventory. An unmeasured (empty) value expands to nothing.
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

// Asset detail (#296, T1) — the per-asset drill-in ported from
// design-system/examples/console/AssetDetail.jsx (templates_asset.go), reached
// from an Inventory row on the stable `/asset/{key}` route so T15's Inventory can
// link straight here. The "asset" is a Name subject; each section is sourced from
// a Name-scoped read where one honestly exists, and renders the design-system
// empty-state where it does not. No section fabricates a value, none fingerprints
// technology, and the drift trail rides the change palette (never severity).

// assetPageData is the whole asset record, one place: the header identity plus the
// six sections — ports census, DNS, TLS cert (empty-state; parsed cert identity is
// not stored), provenance, signals-here, and the drift trail.
type assetPageData struct {
	Key          string
	Type         string // the domain noun the header tag carries — always "Name"
	Withdrawn    bool
	Seen         string // the latest observation instant for this Name
	InScopeSince string // the covering Seed's declaration date
	Severity     string // header aggregate: the most urgent severity across firing Signals (AssetDetail.jsx:35); empty when none fire
	SevLabel     string // the aggregate severity capitalised for the header badge label ("Critical"); empty when none fire (#22a)
	Exposure     string // header aggregate: the worst reachability across open Ports (AssetDetail.jsx:36); empty when none measured
	Ports        []assetPort
	DNS          []assetDNSRow
	Cert         *assetCert // the TLS certificate's parsed identity off the chain leaf; nil → the honest empty state (#22c)
	Provenance   []assetKV
	Signals      []assetSignal
	Drift        []assetDriftEvent
}

// assetPort is one open Service on this asset's addresses: its port, the reachability
// verdict rendered as an exposure state, and when it first opened. Service is a
// precomputed display string joining the transport with the http-identity Server an
// Endpoint on that port holds, where one exists — a read of stored evidence, not a
// new fingerprint (#22d); transport-only where no Endpoint holds an http-identity.
type assetPort struct {
	Port     string
	Service  string
	Exposure string
	Since    string
}

// assetCert is the TLS certificate's parsed identity for the asset, folded off the
// certificate-chain leaf (#22c). Name is the endpoint's presented name; Fingerprint
// is the leaf's stored fingerprint (chain[0]); NotAfter is the leaf's expiry as a
// date; Issuer and Algorithm are the leaf's parsed identity where the stored value
// carries them (honestly omitted where a pre-parse span does not). Label and Tone
// are precomputed from the days-to-expiry: "valid · Nd" ok, "expires in Nd" warn
// (≤30d), "expired Nd ago" danger.
type assetCert struct {
	Name        string
	Issuer      string
	Algorithm   string
	NotAfter    string
	Label       string
	Tone        string // ok | warn | danger
	Fingerprint string
}

// assetDNSRow is one resolved record: the RR type, its value, and when last seen.
type assetDNSRow struct {
	Type  string
	Value string
	Seen  string
}

// assetKV is one provenance fact ("how it got here"): a micro-label key and a mono
// value.
type assetKV struct {
	K string
	V string
}

// assetSignal is one signal firing on this asset — its rule, the subject it fired
// on, and the rule's severity. Severity is a real per-rule datum (internal/signal
// SeverityFor, P0.1 #442), so the "Signals here" row carries its SeverityBadge
// exactly as the spec renders it (AssetDetail.jsx) — the same ramp Signals, Graph
// and Search read, never fabricated.
type assetSignal struct {
	Rule     string
	Subject  string
	Severity string // the rule's severity token: critical | high | medium | low | info
	SevLabel string // the severity capitalised for the badge label ("Critical") (#22a)
	SigID    string // the stable minted "SIG-####" id the row deep-links to (#22b)
	Time     string // first-raised, rendered relative to now ("4m") (#22b)
}

// assetDriftEvent is one transition on this asset, in change's own language: the
// change kind (carrying its drift family — the chip palette, never severity), the
// facet timeline it moved, a terse detail, and the instant.
type assetDriftEvent struct {
	Change  string
	Family  string
	Subject string
	Detail  string
	Time    string
}

// assetPage renders the per-asset drill-in for one Name (#296). It reads the Name
// subject, then assembles the six sections from Name-scoped reads — each best-effort
// so a thin section falls to its empty-state rather than 500ing the page. The route
// keys on the Name (no `/` or `@`), so a plain `/asset/{key}` path segment resolves.
func (s *server) assetPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// A VERGE_DEV build serves the pinned fixtures.json asset slice — the byte-exact
	// corpus the pixel goldens capture (as the sibling screens do). The seeded key is
	// the fixture's own; any other key still resolves the live read below.
	if s.devMode && r.PathValue("key") == devAssetKey {
		s.render(w, "asset", s.assetFixtureData(acct))
		return
	}
	key := r.PathValue("key")
	subject, err := s.store.GetNameSubject(r.Context(), db.GetNameSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderMissingSubject(w, acct, key)
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

	s.render(w, "asset", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory", "DesignTokens": true,
		"Asset": data,
	})
}

// assetProvenance assembles the "how it got here" facts from the Name's Citation:
// the covering Seed, the hop that introduced it (a CT admission or a resolution
// observation, ADR-0107), the scan source, and when it was first seen. Every fact
// is real — an unmeasured one is simply omitted, never invented — and the covering
// Seed's declaration date doubles as the header's "in scope since".
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

// assetDNS lists the asset's resolved records: the A/AAAA addresses off the current
// resolution, plus every other RR the dns-record facet carries (TXT, MX, CNAME,
// NS). A/AAAA records from the dns-record facet are dropped as duplicates of the
// resolution addresses. Best-effort — a dns-record read failure degrades to just
// the resolution addresses.
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
					continue // already covered by the resolution addresses
				}
				rows = append(rows, assetDNSRow{Type: rr.Type, Value: rr.Data})
			}
		}
	}
	return rows
}

// assetPorts is the ports census: every open Service reachability span on the
// asset's addresses, read straight off the open-span corpus (the same corpus
// Inventory reads) and filtered by address. It carries the port and transport and
// the reachability verdict as an exposure state — never a product or version, on
// the no-fingerprinting guardrail. Best-effort: a read failure yields no census.
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
	// First pass: the http-identity Server an Endpoint on each of the asset's
	// address:port pairs holds, keyed by address:port. An Endpoint key is
	// `name@address:port/transport`, so its Service leg carries the same address:port
	// a reachability Service key does — the join key the census Service column reads.
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

// assetPortService is the census Service column's precomputed display string (#22d):
// the transport joined with the http-identity Server an Endpoint on the port holds,
// where one exists (`tcp · nginx/1.25.0`); the bare transport where no Endpoint holds
// a Server. It is a read of stored http-identity evidence, never a new fingerprint.
func assetPortService(transport, server string) string {
	if server == "" {
		return transport
	}
	return transport + " · " + server
}

// assetExposure maps a reachability verdict to the exposure state the census chip
// carries. A Gap is undiscriminated reach (ADR-0104), so it reads as unverified —
// never as exposed. A port that answered is exposed; one that did not is
// not-reached (we cannot tell a firewall from silence, so we state the honest
// negative rather than claim "firewalled").
func assetExposure(outcome string, isGap bool) string {
	if isGap {
		return "unverified"
	}
	switch outcome {
	case "reached":
		return "exposed"
	case "not-reached":
		return "not-reached"
	default:
		return "unverified"
	}
}

// certificateLeafValue is the parsed shape of a stored `certificate` facet value
// (internal/measure/connectoutcome): the leaf-first fingerprint chain, the leaf's
// expiry, and — where a leaf that parsed them folded the value (#22c) — the leaf's
// issuer distinguished name and signature algorithm. A pre-parse span carries only
// chain + not_after, so issuer/algorithm read empty and the card omits them.
type certificateLeafValue struct {
	Outcome   string   `json:"outcome"`
	Chain     []string `json:"chain"`
	NotAfter  string   `json:"not_after"`
	Issuer    string   `json:"issuer"`
	Algorithm string   `json:"algorithm"`
}

// assetCertificate folds the asset's TLS certificate card off the certificate-chain
// leaf (#22c). It reads the latest per-Endpoint `certificate` value (the same read
// the certificate rules use) and keeps the presented chain on an Endpoint whose Name
// leg IS this asset — the leaf under which the presented chain is single-valued
// (connectoutcome.EndpointKey). From the leaf it renders the real parsed identity:
// the fingerprint (chain[0]), the expiry as a date, and — where the stored value
// carries them — the issuer and signature algorithm, with the validity Label/Tone
// precomputed from the days-to-expiry. A subject that holds no presented leaf returns
// nil, so the card falls to its honest empty state rather than fabricate one.
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
		// The named Endpoint keyed under this asset is preferred; a nameless Endpoint
		// on one of the asset's addresses is the fallback when no named leaf presented.
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

// certValidity precomputes the certificate card's validity Label + Tone from the
// leaf's expiry relative to now (#22c): "valid · Nd" ok while more than 30 days
// remain, "expires in Nd" warn within the 30-day window, and "expired Nd ago" danger
// once past. Days are whole days, floored, so "0d" reads on the day of expiry.
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

// assetHeaderSeverity is the header's aggregate SeverityBadge (AssetDetail.jsx:35):
// the most urgent (lowest-rank) severity across the asset's firing signals — the
// same ramp the "Signals here" rows draw, rolled up to one badge. It reads the
// severity already resolved onto each signal, so it invents nothing. Empty when no
// signal fires, in which case the header simply omits the badge (the spec's own
// conditional-omit pattern, as with the seen/scope line).
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

// assetHeaderExposure is the header's aggregate ExposureBadge (AssetDetail.jsx:36):
// the worst reachability across the asset's open ports (exposed ≻ firewalled ≻
// not-reached ≻ unverified) — one port answering from the internet makes the asset
// exposed. It rolls up the states the census already carries, inventing nothing.
// Empty when no port is measured, in which case the header omits the badge.
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

// assetSignals folds the full signal corpus and keeps the fired members whose
// subject IS this asset (the Name-rule population is keyed by the Name), each row
// carrying its rule's severity (internal/signal SeverityFor, P0.1) for the
// SeverityBadge the spec renders. Best-effort: a corpus-build failure yields no
// signals (the section empty-states).
func (s *server) assetSignals(r *http.Request, key string) []assetSignal {
	corpus, err := s.buildSignalCorpus(r)
	if err != nil {
		return nil
	}
	// deriveSignalInstances mints (idempotently) and reads back the stable SIG-####
	// identity + first-seen for every currently-fired (rule, subject) pair — the same
	// id the Signals screen's drawer resolves under /signals?view= (#22b). Keep the
	// members whose subject IS this asset; each carries its rule's severity + label
	// (SeverityFor, never fabricated) and its raised time rendered relative to now.
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

// assetDrift renders the drift trail from the asset's Span timelines, classified in
// change's own language: a lone open span appeared, an open span with prior history
// changed, and a timeline whose last span is closed with no successor was withdrawn
// (by the world, never resolved). One event per facet timeline; the family is the
// chip palette the change rides, never the severity ramp.
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
