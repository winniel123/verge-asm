package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/delivery"
	"github.com/winniel123/verge-asm/internal/queue"
)

// No field may carry an estate row; the type is the ADR-0039 guarantee.

type ReadyBody struct {
	Kind        string    `json:"kind"`
	Report      string    `json:"report"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	URL         string    `json:"url"`
}

const reportReadyPath = "/reports/delivery"

func BuildReadyBody(report string, periodStart, periodEnd time.Time, baseURL string) ReadyBody {
	// The firing-shaped delivery body is deliberately not shared, so no estate field rides along.
	return ReadyBody{
		Kind:        "report-ready",
		Report:      report,
		PeriodStart: periodStart.UTC(),
		PeriodEnd:   periodEnd.UTC(),
		URL:         trimSlash(baseURL) + reportReadyPath,
	}
}

func MarshalReadyBody(b ReadyBody) ([]byte, error) { return json.Marshal(b) }

func shouldNotify(channelID pgtype.Int8) bool { return channelID.Valid }

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

type NotifyRunner struct {
	pool     *pgxpool.Pool
	q        *db.Queries
	doer     delivery.Doer
	now      func() time.Time
	baseURL  string
	log      *log.Logger
	resolver delivery.Resolver
}

func NewNotifyRunner(pool *pgxpool.Pool, doer delivery.Doer, now func() time.Time, baseURL string, logger *log.Logger) *NotifyRunner {
	if now == nil {
		now = time.Now
	}
	if doer == nil {
		doer = delivery.NewHTTPDoer()
	}
	return &NotifyRunner{pool: pool, q: db.New(pool), doer: doer, now: now, baseURL: baseURL, log: logger, resolver: net.DefaultResolver}
}

func (n *NotifyRunner) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := n.Drain(ctx); err != nil && !errors.Is(err, context.Canceled) {
			n.log.Printf("report notify: drain: %v", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (n *NotifyRunner) Drain(ctx context.Context) error {
	for {
		ran, err := n.RunOnce(ctx)
		if err != nil {
			return err
		}
		if !ran {
			return nil
		}
	}
}

func (n *NotifyRunner) RunOnce(ctx context.Context) (bool, error) {
	claim, err := n.q.ClaimReportNotification(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("report notify: claim: %w", err)
	}
	if err := n.post(ctx, claim); err != nil {
		return true, fmt.Errorf("report notify: post %d: %w", claim.ID, err)
	}
	return true, nil
}

func (n *NotifyRunner) post(ctx context.Context, claim db.ClaimReportNotificationRow) error {
	ch, err := n.q.GetChannelForDelivery(ctx, claim.ChannelID)
	if err != nil {
		return fmt.Errorf("load channel: %w", err)
	}

	body, err := MarshalReadyBody(BuildReadyBody(claim.Name, claim.PeriodStart.Time, claim.PeriodEnd.Time, n.baseURL))
	if err != nil {
		return fmt.Errorf("marshal ready body: %w", err)
	}
	var secret []byte
	if ch.Secret.Valid {
		secret = []byte(ch.Secret.String)
	}

	statusCode, sendErr := delivery.SendSigned(ctx, n.doer, n.resolver, ch.Url, body, secret, n.now().UTC())
	delivered := sendErr == nil && delivery.Delivered(statusCode)

	switch delivery.Decide(delivered, claim.Attempt, claim.MaxAttempts) {
	case delivery.VerdictDelivered:
		return n.markDelivered(ctx, claim)
	case delivery.VerdictRetry:
		failure := notifyError(statusCode, sendErr)
		n.log.Printf("report notify: %d attempt %d failed, retrying: %s", claim.ID, claim.Attempt, failure)
		return n.q.RetryReportNotification(ctx, db.RetryReportNotificationParams{
			ID:        claim.ID,
			Attempt:   claim.Attempt + 1,
			RunAfter:  tstz(n.now().UTC().Add(queue.Backoff(claim.Attempt + 1))),
			LastError: pgText(failure),
		})
	default:
		failure := notifyError(statusCode, sendErr)
		n.log.Printf("report notify: %d dead-lettered after %d attempts: %s", claim.ID, claim.Attempt, failure)
		// The receipt is untouched on failure: the artifact stays generated and viewable (ADR-0039).
		return n.q.MarkReportNotificationUndelivered(ctx, db.MarkReportNotificationUndeliveredParams{
			ID: claim.ID, LastError: pgText(failure),
		})
	}
}

func (n *NotifyRunner) markDelivered(ctx context.Context, claim db.ClaimReportNotificationRow) error {
	// Both facts move together, or a viewer sees a delivered receipt with an undelivered notice.
	tx, err := n.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := n.q.WithTx(tx)

	if err := qtx.MarkReportNotificationDelivered(ctx, claim.ID); err != nil {
		return fmt.Errorf("mark notification delivered: %w", err)
	}
	if err := qtx.MarkReportDeliveryDelivered(ctx, db.MarkReportDeliveryDeliveredParams{
		ID:          claim.ReportDeliveryID,
		DeliveredAt: tstz(n.now().UTC()),
	}); err != nil {
		return fmt.Errorf("flip receipt delivered: %w", err)
	}
	return tx.Commit(ctx)
}

func notifyError(statusCode int, sendErr error) string {
	if sendErr != nil {
		return sendErr.Error()
	}
	return fmt.Sprintf("HTTP %d", statusCode)
}

func pgText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }
