package delivery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/queue"
)

// Doer is the outbound HTTP surface, behind an interface so the runner is driven
// by a fake in tests and never touches the live network. It is http.Client's
// shape, so the production doer is a plain *http.Client.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewHTTPDoer builds the production doer: an https client that REFUSES redirects
// (a 3xx is a failure, never followed — it would move our attack surface to a
// host the operator never declared, §4) and bounds every attempt with a timeout.
func NewHTTPDoer() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// Runner is the worker-side loop that drains routed deliveries off the queue and
// POSTs each one, recording the outcome. It reuses the measurement queue's own
// retry/backoff/dead-letter curve (queue.Backoff) rather than minting a second
// schedule beside it.
type Runner struct {
	pool    *pgxpool.Pool
	q       *db.Queries
	doer    Doer
	now     func() time.Time
	baseURL string
	log     *log.Logger
}

// NewRunner builds a Runner over pool driving doer. baseURL is the absolute URL
// into this instance used to build each body's Link; it may be empty, in which
// case bodies carry no link. now is injectable for tests.
func NewRunner(pool *pgxpool.Pool, doer Doer, now func() time.Time, baseURL string, logger *log.Logger) *Runner {
	if now == nil {
		now = time.Now
	}
	if doer == nil {
		doer = NewHTTPDoer()
	}
	return &Runner{pool: pool, q: db.New(pool), doer: doer, now: now, baseURL: baseURL, log: logger}
}

// EnqueueForMessage routes a freshly-written Message to its Channels by class
// alone and enqueues one pending Delivery per Channel that carries the class. It
// is called at the cause, in the same act that writes the Message, so the body a
// delivery later posts is read from the one stored computation and never
// recomputed. Routing consults nothing but each Channel's class subset and
// whether it is enabled (Routes); a disabled Channel is skipped and no Channel
// ships configured, so a default install enqueues nothing.
func EnqueueForMessage(ctx context.Context, q *db.Queries, messageID int64, class message.Class) (int, error) {
	channels, err := q.ListChannels(ctx)
	if err != nil {
		return 0, fmt.Errorf("delivery: list channels: %w", err)
	}
	enqueued := 0
	for _, c := range channels {
		if !c.Enabled || !Routes(c.RouteDrift, c.RouteCoverage, c.RouteClock, class) {
			continue
		}
		if err := q.InsertDelivery(ctx, db.InsertDeliveryParams{MessageID: messageID, ChannelID: c.ID}); err != nil {
			return enqueued, fmt.Errorf("delivery: enqueue message %d to channel %d: %w", messageID, c.ID, err)
		}
		enqueued++
	}
	return enqueued, nil
}

// Run drains routed deliveries, then polls on an interval until ctx is done.
// Delivery has no LISTEN/NOTIFY of its own: a retried delivery is gated by
// run_after (the shared backoff) and a freshly-routed one is picked up on the
// next poll, so a short interval is ample for a corpus of one POST per message
// per subscribed channel.
func (r *Runner) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := r.Drain(ctx); err != nil && !errors.Is(err, context.Canceled) {
			r.log.Printf("delivery: drain: %v", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Drain runs every claimable delivery until none remain.
func (r *Runner) Drain(ctx context.Context) error {
	for {
		ran, err := r.RunOnce(ctx)
		if err != nil {
			return err
		}
		if !ran {
			return nil
		}
	}
}

// RunOnce claims one pending delivery and posts it. It returns false when none is
// claimable. A claimed delivery always reaches a terminal act — delivered,
// retried, or dead-lettered — so it never sticks in 'sending'.
func (r *Runner) RunOnce(ctx context.Context) (bool, error) {
	claim, err := r.q.ClaimDelivery(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("delivery: claim: %w", err)
	}
	if err := r.post(ctx, claim); err != nil {
		return true, fmt.Errorf("delivery: post %d: %w", claim.ID, err)
	}
	return true, nil
}

// post builds, signs and sends one delivery, then records its outcome. A 2xx
// marks the delivery delivered; any other outcome (a non-2xx status, a refused
// redirect, a transport error) is a failure that retries while attempts remain
// and dead-letters past them. Neither branch touches the Message.
func (r *Runner) post(ctx context.Context, claim db.ClaimDeliveryRow) error {
	msg, err := r.q.GetMessageForDelivery(ctx, claim.MessageID)
	if err != nil {
		return fmt.Errorf("load message: %w", err)
	}
	ch, err := r.q.GetChannelForDelivery(ctx, claim.ChannelID)
	if err != nil {
		return fmt.Errorf("load channel: %w", err)
	}

	body, err := MarshalBody(BuildBody(firingFromRow(msg), r.baseURL))
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	var secret []byte
	if ch.Secret.Valid {
		secret = []byte(ch.Secret.String)
	}

	statusCode, sendErr := r.send(ctx, ch.Url, body, secret)
	delivered := sendErr == nil && Delivered(statusCode)

	switch Decide(delivered, claim.Attempt, claim.MaxAttempts) {
	case VerdictDelivered:
		return r.q.MarkDeliveryDelivered(ctx, db.MarkDeliveryDeliveredParams{
			ID: claim.ID, DeliveredAt: tstz(r.now().UTC()),
		})
	case VerdictRetry:
		failure := deliveryError(statusCode, sendErr)
		r.log.Printf("delivery: %d attempt %d failed, retrying: %s", claim.ID, claim.Attempt, failure)
		return r.q.RetryDelivery(ctx, db.RetryDeliveryParams{
			ID:        claim.ID,
			Attempt:   claim.Attempt + 1,
			RunAfter:  tstz(r.now().UTC().Add(queue.Backoff(claim.Attempt + 1))),
			LastError: pgText(failure),
		})
	default: // VerdictUndelivered
		failure := deliveryError(statusCode, sendErr)
		r.log.Printf("delivery: %d dead-lettered after %d attempts: %s", claim.ID, claim.Attempt, failure)
		return r.q.MarkDeliveryUndelivered(ctx, db.MarkDeliveryUndeliveredParams{
			ID: claim.ID, LastError: pgText(failure),
		})
	}
}

// send performs the POST and returns the status code (0 on a transport error).
// The response body is drained and closed so the connection is reusable; its
// contents are never read into a Message.
func (r *Runner) send(ctx context.Context, targetURL string, body, secret []byte) (int, error) {
	req, err := NewRequest(ctx, targetURL, body, secret, r.now().UTC())
	if err != nil {
		return 0, err
	}
	resp, err := r.doer.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func firingFromRow(m db.GetMessageForDeliveryRow) Firing {
	f := Firing{
		ID:          m.ID,
		Cause:       message.Cause(m.Cause),
		Class:       message.Class(m.Class),
		SubjectKind: m.SubjectKind,
		FiredAt:     m.FiredAt,
		Census:      m.Census,
		Headline:    m.Headline,
	}
	if m.Instant.Valid {
		f.Instant = m.Instant.Time
	}
	return f
}

// deliveryError renders the failure string stored on the delivery for the
// channel-surface drill-down. A transport error carries its own message; a
// non-2xx carries its status.
func deliveryError(statusCode int, sendErr error) string {
	if sendErr != nil {
		return sendErr.Error()
	}
	return fmt.Sprintf("HTTP %d", statusCode)
}

func tstz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func pgText(s string) pgtype.Text         { return pgtype.Text{String: s, Valid: true} }
