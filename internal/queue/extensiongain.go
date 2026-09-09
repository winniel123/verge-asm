package queue

import (
	"context"
	"encoding/json"
	"net/netip"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/retention"
)

type citationSpan = db.ListNameCitationSpansWithinCurrencyRow

func extensionGainMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, in membershipInputs) ([]*message.Message, error) {
	zones := extendingZones(in.seeds)
	// No live extension, no read: the operator who left it off never sees this (ADR-0013 #55).
	if len(zones) == 0 || !touchesNameCitation(changes) {
		return nil, nil
	}
	rows, err := store.ListNameCitationSpansWithinCurrency(ctx, db.ListNameCitationSpansWithinCurrencyParams{
		At:            tstz(observedAt),
		FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return nil, err
	}
	// Before spans the currency bound, so a re-entry inside it is no gain (ADR-0013 #55).
	before := custody.Estate{ExtendedZones: zones, Resolutions: citationsFrom(rows, func(r citationSpan) bool {
		return r.OpenedAt.Time.Before(observedAt)
	})}
	after := custody.Estate{ExtendedZones: zones, Resolutions: citationsFrom(rows, func(r citationSpan) bool {
		return !r.ClosedAt.Valid
	})}
	reached := map[netip.Addr]bool{}
	for _, a := range before.ExtensionCandidates() {
		reached[a] = true
	}
	type scopeGain struct {
		covered map[netip.Addr]bool
		gained  map[string][]string
	}
	byScope := map[string]*scopeGain{}
	for _, c := range after.ExtensionCitations() {
		scope := extendingScopeOf(c.Owner, zones)
		g := byScope[scope]
		if g == nil {
			g = &scopeGain{covered: map[netip.Addr]bool{}, gained: map[string][]string{}}
			byScope[scope] = g
		}
		g.covered[c.Address] = true
		if !reached[c.Address] {
			g.gained[c.Address.String()] = append(g.gained[c.Address.String()], c.Owner)
		}
	}
	scopes := make([]string, 0, len(byScope))
	for scope := range byScope {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	var msgs []*message.Message
	for _, scope := range scopes {
		g := byScope[scope]
		gains := make([]message.ExtensionGain, 0, len(g.gained))
		for addr, names := range g.gained {
			gains = append(gains, message.ExtensionGain{Address: addr, Names: names})
		}
		if m := message.ExtensionGained(scope, gains, len(g.covered), observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func extendingZones(seeds []db.ListSeedsRow) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range seeds {
		if s.Kind != "name" || !s.CustodyExtension || !s.NameDomain.Valid {
			continue
		}
		zone := normalizeDomain(s.NameDomain.String)
		if zone == "" || seen[zone] {
			continue
		}
		seen[zone] = true
		out = append(out, zone)
	}
	return out
}

func extendingScopeOf(owner string, zones []string) string {
	// The most specific extending scope claims the owner, so nested scopes never both fire.
	best := ""
	for _, zone := range zones {
		if custody.LabelSuffix(owner, zone) && len(zone) > len(best) {
			best = zone
		}
	}
	return best
}

func touchesNameCitation(changes []spanChange) bool {
	for _, c := range changes {
		if c.SubjectKind != subjectKindName {
			continue
		}
		if c.Facet == resolutionwalk.FacetResolution || c.Facet == resolutionwalk.FacetDNSRecord {
			return true
		}
	}
	return false
}

type citationTimeline struct {
	subject string
	vantage pgtype.Int8
}

func citationsFrom(rows []citationSpan, keep func(citationSpan) bool) []custody.Resolution {
	// The owner rides the dns-record facet, as NameCitedAddresses joins it (ADR-0151 §2, #1678).
	owners := map[citationTimeline]map[string]string{}
	for _, r := range rows {
		if r.Facet != resolutionwalk.FacetDNSRecord || !keep(r) {
			continue
		}
		var v struct {
			RRs []resolutionwalk.RR `json:"rrs"`
		}
		if json.Unmarshal(r.Value, &v) != nil {
			continue
		}
		key := citationTimeline{subject: r.SubjectKey, vantage: r.VantageID}
		if owners[key] == nil {
			owners[key] = map[string]string{}
		}
		for _, rr := range v.RRs {
			if rr.Type == "A" || rr.Type == "AAAA" {
				owners[key][rr.Data] = rr.Name
			}
		}
	}
	var out []custody.Resolution
	for _, r := range rows {
		if r.Facet != resolutionwalk.FacetResolution || !keep(r) {
			continue
		}
		var v struct {
			Outcome   string   `json:"outcome"`
			Addresses []string `json:"addresses"`
		}
		if json.Unmarshal(r.Value, &v) != nil || v.Outcome != string(resolutionwalk.OutcomeResolved) {
			continue
		}
		key := citationTimeline{subject: r.SubjectKey, vantage: r.VantageID}
		for _, a := range v.Addresses {
			addr, err := netip.ParseAddr(a)
			if err != nil {
				continue
			}
			owner := r.SubjectKey
			if o, ok := owners[key][a]; ok {
				owner = o
			}
			out = append(out, custody.Resolution{Owner: owner, Address: addr.Unmap()})
		}
	}
	return out
}
