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

type auditPeriod struct {
	Token  string
	Label  string
	Window time.Duration
}

// The presets drift.tmpl and reports.tmpl already ship, so the third port adds no vocabulary.

func auditPeriods() []auditPeriod {
	return []auditPeriod{
		{Token: "24h", Label: "Last 24h", Window: 24 * time.Hour},
		{Token: "7d", Label: "Last 7d", Window: 7 * 24 * time.Hour},
		{Token: "30d", Label: "Last 30d", Window: 30 * 24 * time.Hour},
		{Token: "90d", Label: "Last 90d", Window: 90 * 24 * time.Hour},
	}
}

const auditDefaultPeriod = "7d"

func resolveAuditPeriod(token string) auditPeriod {
	for _, p := range auditPeriods() {
		if p.Token == token {
			return p
		}
	}
	// The design default is the second preset, not the first, so the fallback names it.
	for _, p := range auditPeriods() {
		if p.Token == auditDefaultPeriod {
			return p
		}
	}
	return auditPeriods()[0]
}

// The scope is server-side because the corpus is unbounded (ADR-0158 limb 4, spec §6.2).

func (s *server) resolveAuditWindow(r *http.Request) (token, label string, from, until pgtype.Timestamptz) {
	q := r.URL.Query()
	if tk, lb, f, u, ok := resolveCustomWindow(q); ok {
		return tk, lb, f, u
	}
	p := resolveAuditPeriod(q.Get("period"))
	// A preset carries no upper bound: a skew between this clock and now() would else hide
	// the row the request before this one recorded.
	return p.Token, p.Label, pgtype.Timestamptz{Time: s.now().UTC().Add(-p.Window), Valid: true},
		pgtype.Timestamptz{}
}

func (s *server) fillAuditSection(r *http.Request, data map[string]any) error {
	token, label, from, until := s.resolveAuditWindow(r)
	data["Periods"] = auditPeriods()
	data["Period"] = token
	data["PeriodLabel"] = label

	rows, err := s.auditStore.ListActsInRange(r.Context(), db.ListActsInRangeParams{
		FromTime: from, UntilTime: until,
	})
	if err != nil {
		return err
	}
	acts, err := s.auditRows(rows)
	if err != nil {
		return err
	}
	data["AuditRows"] = acts
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
