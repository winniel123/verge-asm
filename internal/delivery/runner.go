package delivery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/queue"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewHTTPDoer() *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		// Control runs after resolution, so a rebinding to a private answer is still barred (#325).
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip, err := netip.ParseAddr(host)
			if err != nil {
				return err
			}
			if custody.IsNonGloballyReachable(ip.Unmap()) {
				return fmt.Errorf("delivery: refusing to dial non-globally-reachable address %s", host)
			}
			return nil
		},
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
		// A followed 3xx would move our attack surface to a host the operator never declared (§4).
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

type Runner struct {
	pool     *pgxpool.Pool
	q        *db.Queries
	doer     Doer
	now      func() time.Time
	baseURL  string
	log      *log.Logger
	resolver Resolver
}

func NewRunner(pool *pgxpool.Pool, doer Doer, now func() time.Time, baseURL string, logger *log.Logger) *Runner {
	if now == nil {
		now = time.Now
	}
	if doer == nil {
		doer = NewHTTPDoer()
	}
	return &Runner{pool: pool, q: db.New(pool), doer: doer, now: now, baseURL: baseURL, log: logger, resolver: net.DefaultResolver}
}

func EnqueueForMessage(ctx context.Context, q *db.Queries, messageID int64, class message.Class) (int, error) {
	// A caller enqueues in the same act that writes the Message, so the body is never recomputed.
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

func (r *Runner) Run(ctx context.Context, interval time.Duration) error {
	// No LISTEN/NOTIFY here: run_after gates a retry and a poll picks up a fresh route.
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
		// The measurement queue's curve is reused rather than a second schedule minted (#188).
		return r.q.RetryDelivery(ctx, db.RetryDeliveryParams{
			ID:        claim.ID,
			Attempt:   claim.Attempt + 1,
			RunAfter:  tstz(r.now().UTC().Add(queue.Backoff(claim.Attempt + 1))),
			LastError: pgText(failure),
		})
	default:
		failure := deliveryError(statusCode, sendErr)
		r.log.Printf("delivery: %d dead-lettered after %d attempts: %s", claim.ID, claim.Attempt, failure)
		return r.q.MarkDeliveryUndelivered(ctx, db.MarkDeliveryUndeliveredParams{
			ID: claim.ID, LastError: pgText(failure),
		})
	}
}

func (r *Runner) send(ctx context.Context, targetURL string, body, secret []byte) (int, error) {
	return SendSigned(ctx, r.doer, r.resolver, targetURL, body, secret, r.now().UTC())
}

func SendSigned(ctx context.Context, doer Doer, res Resolver, targetURL string, body, secret []byte, now time.Time) (int, error) {
	// The ONE place the guard and the request build live, so every sender sends by identical rules.
	if err := guardTarget(ctx, res, targetURL); err != nil {
		return 0, err
	}
	req, err := NewRequest(ctx, targetURL, body, secret, now)
	if err != nil {
		return 0, err
	}
	resp, err := doer.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	// The body is drained so net/http can reuse the connection; its contents are never read.
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func guardTarget(ctx context.Context, res Resolver, targetURL string) error {
	// The config-time check sees literals only, so a hostname or a rebinding is barred here (#325).
	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("parse target url: %w", err)
	}
	host := u.Hostname()
	if ip, err := netip.ParseAddr(host); err == nil {
		if custody.IsNonGloballyReachable(ip.Unmap()) {
			return fmt.Errorf("refusing delivery to non-globally-reachable host %s", host)
		}
		return nil
	}
	addrs, err := res.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", host, err)
	}
	for _, a := range addrs {
		if custody.IsNonGloballyReachable(a.Unmap()) {
			return fmt.Errorf("refusing delivery to %q: resolves to non-globally-reachable address %s", host, a)
		}
	}
	return nil
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

func deliveryError(statusCode int, sendErr error) string {
	// Redacting here covers all three sinks: the log line, delivery.last_error and the UI (#740).
	if sendErr != nil {
		return redactTransportError(sendErr)
	}
	return fmt.Sprintf("HTTP %d", statusCode)
}

func redactTransportError(sendErr error) string {
	var urlErr *url.Error
	// For a no-secret Channel the credential is the URL path itself (ADR-0053, #740).
	if errors.As(sendErr, &urlErr) {
		// A *url.Error from http.Client.Do or url.Parse embeds the target URL verbatim.
		if urlErr.Err != nil {
			return fmt.Sprintf("%s: %s", urlErr.Op, urlErr.Err)
		}
		return urlErr.Op
	}
	return sendErr.Error()
}

func tstz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func pgText(s string) pgtype.Text         { return pgtype.Text{String: s, Valid: true} }
