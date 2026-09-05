package queue

import (
	"context"
	"encoding/json"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/estate"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

type membershipInputs struct {
	seeds      []db.ListSeedsRow
	exclusions []db.ListExclusionsRow
}

func readMembershipInputs(ctx context.Context, qtx *db.Queries) (membershipInputs, error) {
	// A partial declared input would fake a departure, so a read error fails the fold (ADR-0001).
	seeds, err := qtx.ListSeeds(ctx)
	if err != nil {
		return membershipInputs{}, err
	}
	exclusions, err := qtx.ListExclusions(ctx)
	if err != nil {
		return membershipInputs{}, err
	}
	return membershipInputs{seeds: seeds, exclusions: exclusions}, nil
}

func foldEstateTransitions(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, obs []wire.Observation, in membershipInputs, deps *[]departure) error {
	// Membership is re-decided only where fresh evidence arrived, never as a background sweep.
	for _, name := range observedResolutionNames(obs) {
		// The value fold runs first, so this Name's current resolution span is already open (ADR-0007).
		open, err := qtx.ListOpenSpansForSubject(ctx, db.ListOpenSpansForSubjectParams{
			SubjectKind: subjectKindName,
			SubjectKey:  name,
		})
		if err != nil {
			return err
		}
		reason, left := decideNameDeparture(open, nameSeedCovered(name, in.seeds), nameExcluded(name, in.exclusions))
		if !left {
			continue
		}
		if err := closeSubjectTimelines(ctx, qtx, open, observedAt, reason, batchID); err != nil {
			return err
		}
		if deps != nil {
			*deps = append(*deps, departure{
				SubjectKind: subjectKindName,
				SubjectKey:  name,
				Reason:      string(reason),
				SourceKey:   coveringExclusionKey(name, reason, in.exclusions),
				Timelines:   len(open),
			})
		}
	}
	return nil
}

func coveringExclusionKey(name string, reason drift.ClosureReason, exclusions []db.ListExclusionsRow) string {
	// An address withdrawal fires one Narrowing per exclusion, so a branch here has no caller (#1032).
	if reason != drift.ReasonDescoped {
		return ""
	}
	name = normalizeDomain(name)
	// This must mirror nameExcluded, so the boundary that removed the Name is the one cited.
	for _, e := range exclusions {
		if !e.Name.Valid {
			continue
		}
		switch e.Kind {
		case "name":
			if name == normalizeDomain(e.Name.String) {
				return normalizeDomain(e.Name.String)
			}
		case "subtree":
			if nameWithinDomain(name, e.Name.String) {
				return normalizeDomain(e.Name.String)
			}
		}
	}
	return ""
}

const (
	subjectKindName     = "name"
	subjectKindAddress  = "address"
	subjectKindService  = "service"
	subjectKindEndpoint = "endpoint"

	exclusionKindAddress = "address"
)

func observedResolutionNames(obs []wire.Observation) []string {
	seen := map[string]bool{}
	var out []string
	for _, o := range obs {
		if o.Facet != resolutionwalk.FacetResolution || o.Subject == "" {
			continue
		}
		if seen[o.Subject] {
			continue
		}
		seen[o.Subject] = true
		out = append(out, o.Subject)
	}
	return out
}

func decideNameDeparture(open []db.ListOpenSpansForSubjectRow, seedCovered, excluded bool) (drift.ClosureReason, bool) {
	if len(open) == 0 {
		return "", false
	}
	if excluded {
		// An exclusion acts on our aperture and claims no absence, so it needs no witness (ADR-0087).
		return drift.ReasonDescoped, true
	}
	witnesses := resolutionWitnesses(open)
	if seedCovered {
		// A Seed admits a Name and holds it only where measurement cannot decide (ADR-0146 §1).
		if estate.DecidedAbsentCrossClass(witnesses) {
			return drift.ReasonMeasuredAbsent, true
		}
		return "", false
	}
	if estate.WithdrawnCrossClass(witnesses) {
		return drift.ReasonMeasuredAbsent, true
	}
	return "", false
}

func resolutionWitnesses(open []db.ListOpenSpansForSubjectRow) []estate.ClassWitness {
	// One class per vantage collapses both unanimity rules into cross-vantage unanimity (ADR-0080).
	byVantage := map[string][]string{}
	order := []string{}
	for _, s := range open {
		if s.Facet != resolutionwalk.FacetResolution {
			continue
		}
		key := vantageClassKey(s.VantageID)
		if _, seen := byVantage[key]; !seen {
			order = append(order, key)
		}
		byVantage[key] = append(byVantage[key], resolutionOutcome(s.Value))
	}
	out := make([]estate.ClassWitness, 0, len(order))
	for _, key := range order {
		out = append(out, estate.ClassWitness{Class: key, Outcomes: byVantage[key]})
	}
	return out
}

func vantageClassKey(v pgtype.Int8) string {
	if !v.Valid {
		return ""
	}
	return "vantage:" + strconv.FormatInt(v.Int64, 10)
}

func resolutionOutcome(value []byte) string {
	var v struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(value, &v)
	return v.Outcome
}

func closeSubjectTimelines(ctx context.Context, qtx *db.Queries, open []db.ListOpenSpansForSubjectRow, at time.Time, reason drift.ClosureReason, batchID int64) error {
	// A withdrawal takes every timeline the subject held: one cause on n objects (ADR-0082).
	ids := make([]int64, 0, len(open))
	for _, s := range open {
		ids = append(ids, s.ID)
	}
	return closeSpansByID(ctx, qtx, ids, at, reason, batchID)
}

func closeSpansByID(ctx context.Context, qtx *db.Queries, ids []int64, at time.Time, reason drift.ClosureReason, batchID int64) error {
	for _, id := range ids {
		if err := qtx.CloseSpan(ctx, db.CloseSpanParams{
			ClosedAt:      tstz(at),
			ClosureReason: pgText(string(reason)),
			ClosedBatchID: pgInt8(batchID),
			ID:            id,
		}); err != nil {
			return err
		}
	}
	return nil
}

func openedByAperture(subjectKind, subjectKey string, in membershipInputs) bool {
	// An exclusion cuts the Seed limb alone, so a still-probed address opens appeared (ADR-0133 §3).
	if subjectKind == subjectKindName {
		return nameSeedCovered(subjectKey, in.seeds) && !nameExcluded(subjectKey, in.exclusions)
	}
	addr, ok := subjectAddress(subjectKind, subjectKey)
	return ok && addressSeedCovered(addr, in.seeds) && coveringAddressExclusion(addr, in.exclusions) == nil
}

func subjectAddress(subjectKind, subjectKey string) (netip.Addr, bool) {
	switch subjectKind {
	case subjectKindAddress:
		addr, err := netip.ParseAddr(subjectKey)
		return addr, err == nil
	case subjectKindService, subjectKindEndpoint:
		addr, err := netip.ParseAddr(serviceAddress(subjectKey))
		return addr, err == nil
	default:
		return netip.Addr{}, false
	}
}

func nameSeedCovered(name string, seeds []db.ListSeedsRow) bool {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	for _, s := range seeds {
		if s.Kind != "name" || !s.NameDomain.Valid {
			continue
		}
		if nameWithinDomain(name, s.NameDomain.String) {
			return true
		}
	}
	return false
}

func nameExcluded(name string, exclusions []db.ListExclusionsRow) bool {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	for _, e := range exclusions {
		if !e.Name.Valid {
			continue
		}
		switch e.Kind {
		case "name":
			if name == normalizeDomain(e.Name.String) {
				return true
			}
		case "subtree":
			if nameWithinDomain(name, e.Name.String) {
				return true
			}
		}
	}
	return false
}

func coveringAddressExclusion(addr netip.Addr, exclusions []db.ListExclusionsRow) *netip.Prefix {
	// An exclusion carries no precedence, so first match is the whole rule (#1032).
	for _, e := range exclusions {
		if e.Kind != exclusionKindAddress || e.AddressCidr == nil {
			continue
		}
		if e.AddressCidr.Contains(addr) {
			return e.AddressCidr
		}
	}
	return nil
}

func addressSeedCovered(addr netip.Addr, seeds []db.ListSeedsRow) bool {
	for _, s := range seeds {
		if s.Kind != "address" || s.AddressCidr == nil {
			continue
		}
		if s.AddressCidr.Contains(addr) {
			return true
		}
	}
	return false
}

func nameWithinDomain(name, domain string) bool {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	domain = normalizeDomain(domain)
	if domain == "" {
		return false
	}
	return name == domain || strings.HasSuffix(name, "."+domain)
}

func normalizeDomain(d string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(d)), ".")
}

func serviceAddress(key string) string {
	if i := strings.LastIndex(key, ":"); i >= 0 {
		host := key[:i]
		host = strings.TrimPrefix(host, "[")
		host = strings.TrimSuffix(host, "]")
		return host
	}
	return key
}
