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

func scheduledTick(now time.Time, cadence time.Duration) time.Time {
	// The floor exists so a second tick in one window collides on (scan, scheduled_time).
	if cadence <= 0 {
		return now.UTC().Truncate(time.Second)
	}
	secs := int64(cadence / time.Second)
	if secs <= 0 {
		secs = 1
	}
	// A missed window is never caught up: the boundary is the current one, never a past one.
	floored := (now.UTC().Unix() / secs) * secs
	return time.Unix(floored, 0).UTC()
}

// Channel delivery waits on this curve rather than a second schedule (notification-channels §4.2).

func Backoff(attempt int32) time.Duration { return backoff(attempt) }

// The prober and worker-read paths both fork here, so neither can settle a run early (#753).

func exhaustedRetries(attempt, maxAttempts int32) bool {
	return attempt >= maxAttempts
}

func backoff(attempt int32) time.Duration {
	// A 30s base summed to ~15m and missed the promised hour (notification-channels §4.2).
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

func toObservationParams(batchID int64, vantageID pgtype.Int8, observedAt pgtype.Timestamptz, obs []wire.Observation) []db.InsertObservationParams {
	// A facet-less line names a kind whose leaf has not landed, so it is skipped, never stored.
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

func subjectKindFor(facet string) string {
	switch facet {
	case connectoutcome.FacetReachability, tlsacceptance.Facet:
		// SNI is not a candidate, so no name selects this subject (measurement-offers §1.6).
		return "service"
	case connectoutcome.FacetCertificate, httpexchange.FacetHTTPIdentity:
		// A chain and an HTTP identity are single-valued only per (Name, Service) (CONTEXT.md).
		return "endpoint"
	case resolutionwalk.FacetResolution, resolutionwalk.FacetDNSRecord:
		return "name"
	default:
		return "name"
	}
}

func sourceFor(facet string) string {
	switch facet {
	case connectoutcome.FacetReachability, connectoutcome.FacetCertificate,
		httpexchange.FacetHTTPIdentity, tlsacceptance.Facet:
		// A distinct source keeps a prober bound from ageing against the resolver's cadence (CONTEXT.md).
		return "prober"
	default:
		return "resolver"
	}
}

func pgInt8(v int64) pgtype.Int8 { return pgtype.Int8{Int64: v, Valid: true} }

func pgText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }
func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
