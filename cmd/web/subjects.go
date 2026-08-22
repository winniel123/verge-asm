package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
)

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
		s.renderStatus(w, http.StatusNotFound, "subject-missing", map[string]any{
			"Title": "No such subject", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"Name": key,
		})
		return
	}
	subject, err := s.store.GetEndpointSubject(r.Context(), db.GetEndpointSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderStatus(w, http.StatusNotFound, "subject-missing", map[string]any{
			"Title": "No such subject", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"Name": key,
		})
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
	data.Citation, data.CitationTerminated, data.Withdrawn = s.buildEndpointCitation(r, name, service, addr)
	data.Timelines = s.buildTimelines(r, "endpoint", subject.SubjectKey)

	s.render(w, "endpoint", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Endpoint": data,
	})
}

// buildEndpointCitation assembles the "why is this here" chain for an Endpoint:
// the Endpoint itself, its Name leg (or a nameless marker), its Service leg, and
// then the Address membership limbs the Service rests on — the Name whose current
// resolution cites the Address, or the address-scope Seed that covers it,
// terminating at a Seed. Where neither limb holds, the Address (and with it the
// Service and Endpoint) has left the estate. It reuses the same address-membership
// store reads the Service citation does, so the two chains agree on the ground.
func (s *server) buildEndpointCitation(r *http.Request, name, service, addr string) (hops []citationHop, terminated, withdrawn bool) {
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
			hops = append(hops, citationHop{Label: "Declared · Seed", Value: "address scope " + scope, Detail: detail})
			terminated = true
		}
	}

	withdrawn = !cited && !terminated
	return hops, terminated, withdrawn
}

// servicePage is the drill-down for one Service subject (#195). The Service key
// carries a `/`, so it arrives as a `?key=` query parameter rather than a path
// segment. It renders the current reachability verdict, the Citation chain back
// to a Seed, and the reachability Span timelines the hot Scan folds.
func (s *server) servicePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" {
		s.renderStatus(w, http.StatusNotFound, "subject-missing", map[string]any{
			"Title": "No such subject", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"Name": key,
		})
		return
	}
	subject, err := s.store.GetServiceSubject(r.Context(), db.GetServiceSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderStatus(w, http.StatusNotFound, "subject-missing", map[string]any{
			"Title": "No such subject", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"Name": key,
		})
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
	data.Citation, data.CitationTerminated, data.Withdrawn = s.buildServiceCitation(r, addr)
	data.Timelines = s.buildTimelines(r, "service", subject.SubjectKey)

	s.render(w, "service", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Service": data,
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
func (s *server) buildServiceCitation(r *http.Request, addr string) (hops []citationHop, terminated, withdrawn bool) {
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
			hops = append(hops, citationHop{
				Label:  "Declared · Seed",
				Value:  "address scope " + scope,
				Detail: detail,
			})
			terminated = true
		}
	}

	// The Address is in the estate exactly while a current resolution cites it or
	// a Seed covers it (CONTEXT.md `Address`). Neither limb holding means it has
	// withdrawn — its Services with it.
	withdrawn = !cited && !terminated
	return hops, terminated, withdrawn
}

func (s *server) subjectPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	key := r.PathValue("key")
	subject, err := s.store.GetNameSubject(r.Context(), db.GetNameSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// A Name nothing has ever measured is genuinely not a subject — not a
		// withdrawn one. Refusing it here is not the false absence ADR-0072
		// guards against: that guard is about a Name we measured *gone*, which
		// GetNameSubject still returns.
		s.renderStatus(w, http.StatusNotFound, "subject-missing", map[string]any{
			"Title": "No such subject", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"Name": key,
		})
		return
	}
	if err != nil {
		s.serverError(w, "get name subject", err)
		return
	}

	res := decodeResolution(subject.Value)
	data := subjectPageData{
		Name:       subject.SubjectKey,
		Withdrawn:  suppressesNameMembership(res.Outcome),
		Resolution: res.Outcome,
		Addresses:  res.Addresses,
	}
	data.Citation, data.CitationTerminated = s.buildCitation(r, subject.SubjectKey)
	data.Timelines = s.buildTimelines(r, "name", subject.SubjectKey)

	s.render(w, "subject", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Subject": data,
	})
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
	Ports        []assetPort
	DNS          []assetDNSRow
	Provenance   []assetKV
	Signals      []assetSignal
	Drift        []assetDriftEvent
}

// assetPort is one open Service on this asset's addresses: its port and transport,
// the reachability verdict rendered as an exposure state, and when it first
// opened. It never carries a product/version — no technology fingerprinting.
type assetPort struct {
	Port      string
	Transport string
	Exposure  string
	Since     string
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

// assetSignal is one signal firing on this asset — its rule and the subject it
// fired on. It carries NO severity: the census is deliberately not a severity ramp
// (signals.go / ADR-0024), so a level here would be fabricated.
type assetSignal struct {
	Rule    string
	Subject string
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
	key := r.PathValue("key")
	subject, err := s.store.GetNameSubject(r.Context(), db.GetNameSubjectParams{
		SubjectKey: key, AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderStatus(w, http.StatusNotFound, "subject-missing", map[string]any{
			"Title": "No such subject", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"Name": key,
		})
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
	data.Signals = s.assetSignals(r, key)
	data.Drift = assetDrift(s.buildTimelines(r, "name", key))

	s.render(w, "asset", map[string]any{
		"Title": subject.SubjectKey, "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "inventory",
		"Asset":     data,
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
			Port:      ":" + port,
			Transport: transport,
			Exposure:  assetExposure(decodeReachability(row.Value).Outcome, row.IsGap),
			Since:     row.OpenedAt.Time.UTC().Format(spanTimeFmt),
		})
	}
	return ports
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

// assetSignals folds the full signal corpus and keeps the fired members whose
// subject IS this asset (the Name-rule population is keyed by the Name). It carries
// no severity — the census is deliberately not a severity ramp — so the section
// lists the firing rules honestly rather than inventing a level. Best-effort: a
// corpus-build failure yields no signals (the section empty-states).
func (s *server) assetSignals(r *http.Request, key string) []assetSignal {
	corpus, err := s.buildSignalCorpus(r)
	if err != nil {
		return nil
	}
	var out []assetSignal
	for _, c := range signal.EvaluateCorpus(corpus) {
		for _, m := range c.Fired {
			if m.Subject == key {
				out = append(out, assetSignal{Rule: c.Rule, Subject: m.Subject})
			}
		}
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
