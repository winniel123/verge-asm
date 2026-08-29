package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/seed"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// The hot Scan (v1 spec §3.4/§3.5) fans out one connect-outcome job per Vantage
// over the addresses Custody admits for that Vantage's class, connecting to the
// verge-core TCP pairs. This file assembles the Custody derivation's inputs from
// the confirmed Seeds and the current resolutions, applies the operator's
// verge-core frequency edits over the shipped default, and enqueues the gated
// jobs. Nothing here probes a target the gate refuses — the gate runs in
// scan.BuildHotJobs before a job is ever enqueued (ADR-0019).

// fanOutHot enqueues one hot job per Vantage over the Custody-admitted addresses.
func (d *Dispatcher) fanOutHot(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	estate, resolved, err := hotEstate(ctx, qtx, d.now())
	if err != nil {
		return 0, err
	}
	// The hot tier probes every declared address scope, so it enumerates all of
	// them (ADR-0047): every declared CIDR is walked daily whether or not a name
	// resolves into it.
	addrs := candidateAddrs(resolved, estate.AddressScopes)
	core, err := hotCore(ctx, qtx)
	if err != nil {
		return 0, err
	}
	vantages, err := vantageList(ctx, qtx)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	for _, j := range scan.BuildHotJobs(scanID, estate, addrs, vantages.scanVantages(), core) {
		if err := enqueueHotJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

// hotEstate builds the Custody derivation's inputs from the confirmed Seeds and
// the current resolutions, and returns the RESOLVED address set (the addresses
// names currently resolve to). Each fan-out then unions this set with the
// addresses its own scopes enumerate via candidateAddrs — the hot tier over every
// declared address scope, the cold tier over only its opted-in scopes — so
// neither tier enumerates a scope it does not probe. The gate then admits or
// refuses each address per Vantage class. asOf is the dispatcher's read instant:
// the current resolutions are read through the live-tier gate (#237, ADR-0041),
// so an Address held only by an evidential resolution — one no derivation may
// still read — is not admitted into the probed estate.
func hotEstate(ctx context.Context, q *db.Queries, asOf time.Time) (custody.Estate, []netip.Addr, error) {
	scopes, err := q.ListAddressScopeCidrs(ctx)
	if err != nil {
		return custody.Estate{}, nil, err
	}
	var prefixes []netip.Prefix
	for _, p := range scopes {
		if p != nil {
			prefixes = append(prefixes, *p)
		}
	}

	zones, err := q.ListExtendedZoneDomains(ctx)
	if err != nil {
		return custody.Estate{}, nil, err
	}
	var extended []string
	for _, z := range zones {
		if z.Valid {
			extended = append(extended, z.String)
		}
	}

	cited, err := q.NameCitedAddresses(ctx, db.NameCitedAddressesParams{
		AsOf:          pgtype.Timestamptz{Time: asOf.UTC(), Valid: true},
		FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return custody.Estate{}, nil, err
	}
	var resolutions []custody.Resolution
	seen := map[netip.Addr]struct{}{}
	var addrs []netip.Addr
	for _, c := range cited {
		addr, perr := netip.ParseAddr(c.Address)
		if perr != nil {
			continue
		}
		addr = addr.Unmap()
		resolutions = append(resolutions, custody.Resolution{Owner: c.SubjectKey, Address: addr})
		if _, ok := seen[addr]; !ok {
			seen[addr] = struct{}{}
			addrs = append(addrs, addr)
		}
	}

	return custody.Estate{
		AddressScopes: prefixes,
		ExtendedZones: extended,
		Resolutions:   resolutions,
	}, addrs, nil
}

// candidateAddrs is a tier's target set BEFORE the Custody gate: the addresses
// names currently resolve to, unioned with every address the given scopes
// enumerate (ADR-0047). Enumerating the scopes here — at the DB boundary, where
// the per-scope address cap (seed.WithinCap at declaration) guarantees each scope
// is bounded — is what turns a declared CIDR into probe targets and dark
// `not-reached` subjects, closing the #779 gap. The caller passes the scopes its
// tier probes: the hot fan-out passes every declared address scope, the cold
// fan-out only its opted-in scopes, so neither enumerates a scope it discards.
// The union is deduplicated and resolved-first, so an address that is both
// resolved and inside a scope is probed once. The Custody gate (scan.BuildHotJobs
// / scan.BuildColdJobs) still runs over the result and remains total (ADR-0019):
// every enumerated candidate is an operator address, but the denotation
// precondition can still bar a non-globally-reachable one from an
// internet-class Vantage.
func candidateAddrs(resolved []netip.Addr, scopes []netip.Prefix) []netip.Addr {
	capHint := len(resolved)
	for _, p := range scopes {
		capHint += seed.EnumCapHint(p)
	}
	seen := make(map[netip.Addr]struct{}, capHint)
	out := make([]netip.Addr, 0, capHint)
	add := func(a netip.Addr) {
		a = a.Unmap()
		if _, ok := seen[a]; ok {
			return
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	for _, a := range resolved {
		add(a)
	}
	for _, p := range scopes {
		for _, a := range seed.EnumerateAddresses(p) {
			add(a)
		}
	}
	return out
}

// hotCore reads the shipped verge-core and applies the operator's frequency-half
// edits over it (v1 spec §3.5). The sensitive half is never read from the
// database — it ships in the release — so no operator edit can reach it.
func hotCore(ctx context.Context, q *db.Queries) (vergecore.List, error) {
	edits, err := q.ListVergeCoreFrequencyEdits(ctx)
	if err != nil {
		return vergecore.List{}, err
	}
	fe := make([]vergecore.FrequencyEdit, 0, len(edits))
	for _, e := range edits {
		fe = append(fe, vergecore.FrequencyEdit{Port: uint16(e.Port), Action: e.Action}) // #nosec G115 (DB port written only via 1..65535-validated edit path)
	}
	return vergecore.Default().WithFrequencyEdits(fe), nil
}

// enqueueHotJob enqueues one connect-outcome job for one Vantage. Its recorded
// scope carries the admitted addresses and the verge-core port sets by content;
// its offers carry the safety profile. It retries like a dns job — a connect is
// a network step that can transiently fail.
func enqueueHotJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.HotJob) error {
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:vantage:%d", scanID, j.VantageID))
	if err != nil {
		return err
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	scopeJSON, err := j.AttemptedScope()
	if err != nil {
		return err
	}
	offersJSON, err := j.OffersJSON()
	if err != nil {
		return err
	}
	_, err = qtx.EnqueueJob(ctx, db.EnqueueJobParams{
		ScanID:         scanID,
		VantageID:      pgInt8(j.VantageID),
		DispatchID:     pgInt8(dispatchID),
		Kind:           j.Kind,
		Spec:           specJSON,
		AttemptedScope: scopeJSON,
		Offers:         offersJSON,
		Attempt:        1,
		MaxAttempts:    5,
		RunAfter:       tstz(time.Now().UTC()),
	})
	return err
}
