package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

// The reader is its own group: the recorder needs no read reach (ADR-0149 §3).

type auditStore interface {
	AnyActRecorded(ctx context.Context) (bool, error)
	ListActsInRange(ctx context.Context, arg db.ListActsInRangeParams) ([]db.Act, error)
}

type auditRow struct {
	When         string
	ISO          string
	Actor        string
	ActorIsGrant bool
	Action       string
	Subject      string
}

// A 90d window over a corpus that is never deleted is itself unbounded (ADR-0178 §1).

const auditFeedLimit int32 = 500

// The scope is server-side because the corpus is unbounded (ADR-0158 limb 4, spec §6.2).

func (s *server) resolveAuditWindow(r *http.Request) (token, label string, from, until pgtype.Timestamptz) {
	q := r.URL.Query()
	if tk, lb, f, u, ok := resolveCustomWindow(q); ok {
		return tk, lb, f, u
	}
	p := resolvePeriodPreset(q.Get("period"))
	// A preset carries no upper bound: a skew between this clock and now() would else hide
	// the row the request before this one recorded.
	return p.Token, p.Label, s.presetSince(p), pgtype.Timestamptz{}
}

func (s *server) fillAuditSection(r *http.Request, data map[string]any) error {
	token, label, from, until := s.resolveAuditWindow(r)
	data["Periods"] = periodPresets()
	data["Period"] = token
	data["PeriodLabel"] = label

	rows, err := s.auditStore.ListActsInRange(r.Context(), db.ListActsInRangeParams{
		FromTime: from, UntilTime: until, MaxActs: auditFeedLimit,
	})
	if err != nil {
		return err
	}
	acts, err := s.auditRows(rows)
	if err != nil {
		return err
	}
	data["AuditRows"] = acts
	data["AuditTruncated"] = int32(len(acts)) >= auditFeedLimit // #nosec G115 (len capped at auditFeedLimit=500 via MaxActs)
	data["AuditCount"] = len(acts)
	if len(acts) > 0 {
		return nil
	}
	// E.3 claims the whole record is empty, which a narrowed period cannot tell on its own.
	recorded, err := s.auditStore.AnyActRecorded(r.Context())
	if err != nil {
		return err
	}
	data["AuditCorpusEmpty"] = !recorded
	return nil
}

func (s *server) auditRows(rows []db.Act) ([]auditRow, error) {
	now := s.now()
	out := make([]auditRow, 0, len(rows))
	for _, row := range rows {
		actor, err := act.DecodeActor(row.ActorKind, row.Actor)
		if err != nil {
			// Rendering the rest would hide one act inside an accountability record (ADR-0168 §4).
			return nil, fmt.Errorf("act %d: %w", row.ID, err)
		}
		a, err := act.DecodeSubject(row.Action, row.Subject)
		if err != nil {
			return nil, fmt.Errorf("act %d: %w", row.ID, err)
		}
		out = append(out, auditRow{
			When:         relTime(row.CreatedAt.Time, now),
			ISO:          row.CreatedAt.Time.UTC().Format(time.RFC3339),
			Actor:        act.ActorCell(actor),
			ActorIsGrant: actor.Kind() == act.KindGrantHolder,
			Action:       a.Label(),
			Subject:      act.SubjectCell(a),
		})
	}
	return out, nil
}
