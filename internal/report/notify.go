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

// A report notification is a LINK-ONLY ready-message to a Channel that a scheduled
// report has been cut (P0.6c/T7, #508, ADR-0039). It is NOT a Message and carries NO
// estate: the body names the report, the run's period, and a session-authed link to
// the in-instance artifact — nothing else crosses the wire. The report body never
// leaves the instance; the Channel receives only the notice and the link.
//
// The transport is the delivery package's shared signed-HTTPS path (SendSigned: the
// SSRF guard, the HMAC signing, the redirect-refusing POST) and the shared
// queue.Backoff retry curve. What is deliberately NOT shared is delivery.BuildBody —
// that carries a Message's firing (class, cause, census, headline), and a report run
// is none of those. This is a distinct, minimal body, so no estate field can ride
// along by accident.

// ReadyBody is the entire document POSTed to a Channel: a fixed kind, the report name,
// the run's period bounds, and the console link to the in-instance artifact. There is
// no field for signals, subjects, withdrawals or any estate row — that is the
// ADR-0039 guarantee, enforced by the type having nowhere to put one. encoding/json
// emits the fields in declaration order, so the body is stable across retries.
type ReadyBody struct {
	Kind string `json:"kind"`
	// Report is the schedule's declared name — a label, never estate.
	Report      string    `json:"report"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	URL         string    `json:"url"`
}

const reportReadyPath = "/reports/delivery"

// BuildReadyBody assembles the link-only ready-message from the run's facts and the
// instance base URL. It reads ONLY the report name and the period — never the estate —
// so no row can ride the body. When baseURL is empty the link is the bare path (the
// notice still sends), mirroring the delivery runner's empty-baseURL handling.
func BuildReadyBody(report string, periodStart, periodEnd time.Time, baseURL string) ReadyBody {
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

// RunOnce claims one pending notification and posts it. It returns false when none is
// claimable. A claimed notification always reaches a terminal act — delivered, retried
// or dead-lettered — so it never sticks in 'sending'.
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

// post builds and sends one ready-message, then records its outcome. A 2xx marks the
// notification delivered AND flips the report_delivery receipt to 'delivered' (the run
// left); any other outcome retries while attempts remain and dead-letters past them.
// A failure NEVER touches the receipt: the artifact was generated and stays viewable
// in-instance regardless of whether its ready-message reached the Channel (ADR-0039).
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
	default: // VerdictUndelivered
		failure := notifyError(statusCode, sendErr)
		n.log.Printf("report notify: %d dead-lettered after %d attempts: %s", claim.ID, claim.Attempt, failure)
		// The receipt is deliberately NOT touched: the artifact stays 'generated' and
		// viewable; only the ready-message failed to leave.
		return n.q.MarkReportNotificationUndelivered(ctx, db.MarkReportNotificationUndeliveredParams{
			ID: claim.ID, LastError: pgText(failure),
		})
	}
}

// markDelivered records a successful send: the notification is delivered and, in the
// same transaction, the report_delivery receipt is flipped to 'delivered' with its
// delivered_at stamp — the two facts move together so a viewer never sees a delivered
// receipt without a delivered notification, or the reverse.
func (n *NotifyRunner) markDelivered(ctx context.Context, claim db.ClaimReportNotificationRow) error {
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
