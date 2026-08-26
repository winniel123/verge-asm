package queue

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/wire"
)

// scheduledTick floors now to the Scan's cadence boundary, so two dispatcher
// ticks inside one cadence window resolve to the same (scan, scheduled_time)
// key and the second conflicts rather than fanning out again. Missed ticks are
// not caught up — the boundary is always the current window, never a past one.
func scheduledTick(now time.Time, cadence time.Duration) time.Time {
	if cadence <= 0 {
		return now.UTC().Truncate(time.Second)
	}
	secs := int64(cadence / time.Second)
	if secs <= 0 {
		secs = 1
	}
	floored := (now.UTC().Unix() / secs) * secs
	return time.Unix(floored, 0).UTC()
}

// Backoff exposes the shared retry schedule so a second corpus running on the
// queue's own retry/backoff/dead-letter machinery — Channel delivery (#207) —
// waits on the same curve as a measurement job, rather than minting a second
// schedule beside it (notification-channels.md §4.2). It is the exported face of
// backoff and nothing more.
func Backoff(attempt int32) time.Duration { return backoff(attempt) }

// backoff is the delay before a retried job's new Batch runs: exponential from a
// base, capped, so five attempts span roughly an hour (v1 spec §4.5's retry
// budget, shared machinery — one schedule for measurement jobs and Channel
// deliveries alike, notification-channels.md §4.2). The waits before attempts
// 2..5 are 4m, 8m, 16m, 32m and sum to 60m; a base of 30s summed to only ~15m
// and missed the budget the doc has always claimed.
func backoff(attempt int32) time.Duration {
	base := 2 * time.Minute
	d := base
	for i := int32(1); i < attempt; i++ {
		d *= 2
		if d >= 32*time.Minute {
			return 32 * time.Minute
		}
	}
	return d
}

// toObservationParams maps the prober's NDJSON observations to observation rows
// for one Batch. It attributes every line to the (subject, facet, discriminator,
// vantage, source) timeline the leaf named. Lines with no facet — kinds whose
// leaf a later ticket adds — are skipped rather than written as facet-less rows.
func toObservationParams(batchID int64, vantageID pgtype.Int8, observedAt pgtype.Timestamptz, obs []wire.Observation) []db.InsertObservationParams {
	out := make([]db.InsertObservationParams, 0, len(obs))
	for _, o := range obs {
		if o.Facet == "" {
			continue
		}
		value := []byte(o.Data)
		if len(value) == 0 {
			value = []byte("null")
		}
		out = append(out, db.InsertObservationParams{
			BatchID:       batchID,
			Facet:         o.Facet,
			SubjectKind:   subjectKindFor(o.Facet),
			SubjectKey:    o.Subject,
			Discriminator: o.Discriminator,
			VantageID:     vantageID,
			Source:        sourceFor(o.Facet),
			Value:         value,
			ObservedAt:    observedAt,
		})
	}
	return out
}

// subjectKindFor gives the subject kind for a facet. resolution and dns-record
// are about a Name; reachability is about a Service (an (Address, port,
// transport) triple); http-identity is about an Endpoint (the (Name, Service)
// pair, keyed name@service), whose key the http-exchange leaf renders on the
// subject. The switch grows one additive case per wave-4 facet.
func subjectKindFor(facet string) string {
	switch facet {
	case connectoutcome.FacetReachability, tlsacceptance.Facet:
		// Both are about a `Service` — the (Address, port, transport) triple.
		// `tls-acceptance` keys on the Service, never an Endpoint: SNI is not a
		// candidate, so no name selects the subject (measurement-offers §1.6).
		return "service"
	case connectoutcome.FacetCertificate, httpexchange.FacetHTTPIdentity:
		// The presented chain and HTTP identity are single-valued only under an
		// `(Name, Service)` pair (CONTEXT.md `Endpoint`).
		return "endpoint"
	case resolutionwalk.FacetResolution, resolutionwalk.FacetDNSRecord:
		return "name"
	default:
		return "name"
	}
}

// sourceFor gives the observation source for a facet. Our own resolver sources
// resolution and dns-record; our own prober sources reachability AND http-identity
// — the HTTP exchange rides the reachability exchange, so it shares the prober
// source, a distinct timeline source from the resolver so a prober bound never
// ages against the resolver's cadence (CONTEXT.md `Observation`).
func sourceFor(facet string) string {
	switch facet {
	case connectoutcome.FacetReachability, connectoutcome.FacetCertificate,
		httpexchange.FacetHTTPIdentity, tlsacceptance.Facet:
		// All ride our own prober's exchange — a distinct timeline source from
		// the resolver, so a bound never ages against the resolver's cadence.
		// `tls-acceptance` is our own prober's enumeration on its own weekly Scan.
		return "prober"
	default:
		return "resolver"
	}
}

func pgInt8(v int64) pgtype.Int8 { return pgtype.Int8{Int64: v, Valid: true} }

// pgText wraps a string as a set pgtype.Text; an empty string is a set-but-empty
// value, never SQL NULL. The withdrawal closure only ever passes one of drift's
// three non-empty grounds, so the distinction does not arise here.
func pgText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }
func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
