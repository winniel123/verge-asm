package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The Signals screen (v1 spec §6.5, ADR-0024/ADR-0102). Every rule's census
// renders as fired / not-fired / not-evaluable, current state only — never a
// delta or trend. The web layer's only job is to fold the Derived corpus
// (resolution / dns-record / membership + the operator's zone file) into the
// per-Name snapshot the `Signal` engine evaluates; the engine owns every verdict.
//
// A census member row is NOT the Subjects row component (ADR-0102): it carries
// no Citation, rides no search, and its header count is exactly list.length. The
// template draws it from a distinct component; this handler never hands it a
// denominator to disagree with.

// dnsRecordValue is the JSON payload of a dns-record observation (the shape the
// resolution-walk leaf emits). The handler reads only the CNAME target off the
// CNAME discriminator and the delegation's Lame verdict off the NS discriminator.
type dnsRecordValue struct {
	RRs []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Data string `json:"data"`
	} `json:"rrs"`
	Delegation *struct {
		Lame bool `json:"lame"`
		Gap  bool `json:"gap"`
	} `json:"delegation"`
}

// signalMember is one census member row: a subject key and nothing else. The
// template links it to /subjects/{Subject} — the drill-down — and renders no
// Citation and no per-row control.
type signalMember struct {
	Subject string
}

// memberGroupView is one census member — a labelled list whose header count is
// exactly len(Members), locked (ADR-0102). Kind is the member's register, used
// only for styling; the three registers are deliberately not a severity ramp.
type memberGroupView struct {
	Label   string
	Kind    string
	Members []signalMember
}

// signalCensusView is one rule's rendered census: three member lists over one
// population, each list's header count locked to its own length. Empty marks a
// rule whose predicate domain holds no subject at all — a no-population panel,
// never a census of zeroes.
type signalCensusView struct {
	Rule    string
	Version string
	Empty   bool
	Groups  []memberGroupView
}

func (s *server) signalsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	facts, err := s.buildNameFacts(r)
	if err != nil {
		s.serverError(w, "build name facts", err)
		return
	}

	views := make([]signalCensusView, 0)
	for _, c := range signal.EvaluateAll(facts) {
		views = append(views, signalCensusView{
			Rule:    c.Rule,
			Version: c.Version.String(),
			Empty:   c.Empty(),
			Groups: []memberGroupView{
				{Label: "Fired", Kind: "fired", Members: members(c.Fired)},
				{Label: "Did not fire", Kind: "not-fired", Members: members(c.NotFired)},
				{Label: "Not-evaluable", Kind: "not-evaluable", Members: members(c.NotEvaluable)},
			},
		})
	}

	s.render(w, "signals", map[string]any{
		"Title": "Signals", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Censuses": views,
	})
}

func members(in []signal.Member) []signalMember {
	out := make([]signalMember, 0, len(in))
	for _, m := range in {
		out = append(out, signalMember{Subject: m.Subject})
	}
	return out
}

// buildNameFacts assembles the current Derived snapshot the engine reads: the
// composed cross-class resolution per Name (folding resolution-walk's outcome,
// the NS delegation's Lame verdict, and wildcard-discrimination's Shadowed), the
// internet-class view, the dns-record CNAME target, and the operator's zone
// declarations. It reads resolution / dns-record / membership only — the five
// rules are Name-only and need no reachability facet.
func (s *server) buildNameFacts(r *http.Request) ([]signal.NameFacts, error) {
	ctx := r.Context()

	resRows, err := s.store.ListNameResolutionsByClass(ctx)
	if err != nil {
		return nil, err
	}
	dnsRows, err := s.store.ListNameDNSRecords(ctx)
	if err != nil {
		return nil, err
	}
	zoneRows, err := s.store.ListZoneDeclarations(ctx)
	if err != nil {
		return nil, err
	}

	// Per Name: the resolution value per Vantage class.
	byClass := map[string]map[string]resolutionValue{}
	for _, row := range resRows {
		m := byClass[row.SubjectKey]
		if m == nil {
			m = map[string]resolutionValue{}
			byClass[row.SubjectKey] = m
		}
		m[row.Class] = decodeResolution(row.Value)
	}

	// Per Name: the CNAME target and the delegation's Lame verdict.
	cnameTarget := map[string]string{}
	nsLame := map[string]bool{}
	for _, row := range dnsRows {
		var v dnsRecordValue
		_ = json.Unmarshal(row.Value, &v)
		switch strings.ToUpper(row.Discriminator) {
		case "CNAME":
			for _, rr := range v.RRs {
				if strings.EqualFold(rr.Type, "CNAME") {
					cnameTarget[row.SubjectKey] = resolutionwalk.CanonicalName(rr.Data)
					break
				}
			}
		case "NS":
			if v.Delegation != nil && v.Delegation.Lame {
				nsLame[row.SubjectKey] = true
			}
		}
	}

	// The operator's zone declarations: the owner-name set (the zone rules'
	// domain) and the declared domains (the containment test for InDeclaredZone).
	declared := map[string]bool{}
	var zoneDomains []string
	for _, row := range zoneRows {
		if !row.NameDomain.Valid {
			continue
		}
		domain := resolutionwalk.CanonicalName(row.NameDomain.String)
		zoneDomains = append(zoneDomains, domain)
		for name := range signal.DeclaredNames(row.Content, domain) {
			declared[name] = true
		}
	}

	// The candidate universe: every Name we have a resolution for, plus every
	// Name a zone file declares (so a declared name that withdrew or was never
	// measured still enters its rule's domain — a signal's lifecycle is its
	// evidence's, not its subject's membership).
	names := map[string]struct{}{}
	for name := range byClass {
		names[name] = struct{}{}
	}
	for name := range declared {
		names[name] = struct{}{}
	}

	// First pass: the composed cross-class resolution per Name, so a CNAME rule
	// can read its target's outcome.
	composed := map[string]composedResolution{}
	for name := range names {
		composed[name] = composeResolution(byClass[name], nsLame[name])
	}

	facts := make([]signal.NameFacts, 0, len(names))
	for name := range names {
		c := composed[name]
		f := signal.NameFacts{
			Name:           name,
			InEstate:       c.inEstate,
			Resolution:     c.outcome,
			Addresses:      c.addresses,
			ZoneDeclared:   declared[name],
			InDeclaredZone: coveredBy(name, zoneDomains),
		}
		if target, ok := cnameTarget[name]; ok {
			f.CNAMETarget = target
			f.TargetResolution = composed[target].outcome // "" when the target was never measured
		}
		if inet, ok := byClass[name]["internet"]; ok {
			f.HasInternetVantage = true
			f.InternetResolution = inet.Outcome
			f.InternetAddresses = inet.Addresses
		}
		facts = append(facts, f)
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Name < facts[j].Name })
	return facts, nil
}

// composedResolution is the one resolution value the four cross-class rules read,
// folded from the per-class observations and the NS delegation.
type composedResolution struct {
	outcome   string
	addresses []string
	inEstate  bool
}

// composeResolution folds the per-class resolution values into one composed
// outcome (CONTEXT.md `Membership`, ADR-0080/ADR-0086). Shadowed wins (it is a
// value about our own sight and cites nothing); then a Resolved answer at any
// class (its address set is the union); then the NS delegation's Lame; then
// NoData; then a withdrawal only where every observed class read NameError; then
// Gap. A Name is in the estate where some class observed it and it is not a
// cross-class NameError.
func composeResolution(classes map[string]resolutionValue, lame bool) composedResolution {
	if len(classes) == 0 {
		return composedResolution{outcome: signal.Gap, inEstate: false}
	}
	anyShadowed, anyResolved, anyNoData := false, false, false
	allNameError := true
	addrs := map[string]struct{}{}
	for _, v := range classes {
		switch v.Outcome {
		case signal.Shadowed:
			anyShadowed = true
		case signal.Resolved:
			anyResolved = true
			for _, a := range v.Addresses {
				addrs[a] = struct{}{}
			}
		case signal.NoData:
			anyNoData = true
		}
		if v.Outcome != signal.NameError {
			allNameError = false
		}
	}

	out := composedResolution{inEstate: !allNameError}
	switch {
	case anyShadowed:
		out.outcome = signal.Shadowed
	case anyResolved:
		out.outcome = signal.Resolved
		out.addresses = sortedKeys(addrs)
	case lame:
		out.outcome = signal.Lame
	case anyNoData:
		out.outcome = signal.NoData
	case allNameError:
		out.outcome = signal.NameError
	default:
		out.outcome = signal.Gap
	}
	return out
}

// coveredBy reports whether a Name falls within any declared zone domain, by the
// same label-wise suffix rule the estate uses everywhere (ADR-0055): the name is
// the domain, or ends with "." + domain.
func coveredBy(name string, domains []string) bool {
	for _, d := range domains {
		if name == d || strings.HasSuffix(name, "."+d) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
