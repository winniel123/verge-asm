package connectoutcome

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"syscall"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
)

type ConnResult string

const (
	ConnOpen     ConnResult = "open"
	ConnRefused  ConnResult = "refused"
	ConnTimedOut ConnResult = "timed-out"
	ConnError    ConnResult = "error"
)

// A refusal is an answer, so it is decided at once and never retried (v1 spec §4.1).

func (r ConnResult) decided() bool { return r == ConnOpen || r == ConnRefused }

// The pair is closed: an unscoped pair has no observation, never a third value (CONTEXT.md Reach).

type Outcome string

const (
	Reached    Outcome = "reached"
	NotReached Outcome = "not-reached"
)

// Custody already decided the target set, so nothing opens a port it was not handed (ADR-0019).

type Connector interface {
	Connect(ctx context.Context, target netip.AddrPort) ConnResult
}

func Decide(result ConnResult) Outcome {
	// A caller passes the last result only after Probe spent the retry budget (v1 spec §4.1).
	if result == ConnOpen {
		return Reached
	}
	return NotReached
}

func Probe(ctx context.Context, c Connector, profile SafetyProfile, target netip.AddrPort) (Outcome, ConnResult) {
	// The retry budget is a rate concern: exhausting it never extends the job's deadline (ADR-0021).
	attempts := profile.Retries + 1
	if attempts < 1 {
		attempts = 1
	}
	var last ConnResult
	// Silence decides on a connection-oriented transport, but only once retries are spent (ADR-0083).
	for i := 0; i < attempts; i++ {
		last = c.Connect(ctx, target)
		if last.decided() {
			return Decide(last), last
		}
	}
	return Decide(last), last
}

type NetConnector struct {
	Timeout time.Duration
}

func (n NetConnector) Connect(ctx context.Context, target netip.AddrPort) ConnResult {
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	// Only a pre-validated literal is dialled, and the socket guard below is the backstop (#743).
	if !target.Addr().IsValid() {
		return ConnError
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	d := net.Dialer{Control: custody.EgressGuard("connectoutcome")}
	conn, err := d.DialContext(dialCtx, "tcp", target.String())
	if err == nil {
		_ = conn.Close()
		return ConnOpen
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return ConnTimedOut
	}
	if isRefused(err) {
		return ConnRefused
	}
	return ConnError
}

func isRefused(err error) bool {
	// The text fallback avoids a per-GOOS errno constant, so the binary stays arch-neutral.
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "refused")
}
