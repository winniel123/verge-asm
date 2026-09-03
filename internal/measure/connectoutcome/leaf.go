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
	ConnOpen ConnResult = "open"
	// ConnRefused: the host answered with an RST — the port is shut. A decided
	// negative, and an answer, so it is never retried.
	ConnRefused ConnResult = "refused"
	// ConnTimedOut: nothing answered within the connect timeout. On a
	// connection-oriented transport silence still decides (ADR-0083), but only
	// after the retries are exhausted — a transient drop is retried first.
	ConnTimedOut ConnResult = "timed-out"
	// ConnError: a local error prevented the attempt (e.g. no route from the
	// prober). It is our own blindness, retried like a timeout, and if it
	// persists the job fails and the Batch dead-letters rather than recording a
	// value (v1 spec §4.1).
	ConnError ConnResult = "error"
)

func (r ConnResult) decided() bool { return r == ConnOpen || r == ConnRefused }

// Outcome is the reachability verdict for one Service at one Vantage — the
// closed pair `reached │ not-reached` (CONTEXT.md `Reach`). There is no value
// for *we did not look*: a pair outside the recorded scope simply has no
// observation.
type Outcome string

const (
	Reached Outcome = "reached"
	// NotReached: the connect was refused, or timed out after its retries. On a
	// connection-oriented transport silence decides, so this is a value and not a
	// Gap.
	NotReached Outcome = "not-reached"
)

// Connector performs one TCP connect and reports its raw result. The production
// adapter dials the real address with a bounded timeout and closes immediately;
// the golden corpus scripts an in-process Connector, so the leaf's verdict logic
// runs hermetically with no network and no container. A Connector never opens a
// port it was not handed — the Custody gate has already decided the target set
// (ADR-0019), and the leaf connects only to what its scope lists.
type Connector interface {
	Connect(ctx context.Context, target netip.AddrPort) ConnResult
}

// Decide folds a raw connect result to the reachability verdict. It is the pure
// heart of the leaf and the thing the golden corpus pins: an open connects to
// `reached`; a refusal or an (exhausted) timeout to `not-reached`. A ConnError
// that reached this point has exhausted its retries and is our own blindness —
// it decides `not-reached` only where a real timeout would, so callers pass the
// last result after Probe has spent the retry budget.
func Decide(result ConnResult) Outcome {
	if result == ConnOpen {
		return Reached
	}
	return NotReached
}

// Probe runs one Service's connect with the profile's retry budget and returns
// both the reachability verdict and the last raw result (recorded as evidence).
// A decided result — open or refusal — returns at once; a timeout or local error
// is retried up to profile.Retries times before the leaf accepts silence as
// `not-reached`. The retry budget is a rate concern and never a deadline one:
// each attempt carries its own connect timeout, and exhausting retries never
// extends the job's own deadline (ADR-0021).
func Probe(ctx context.Context, c Connector, profile SafetyProfile, target netip.AddrPort) (Outcome, ConnResult) {
	attempts := profile.Retries + 1
	if attempts < 1 {
		attempts = 1
	}
	var last ConnResult
	for i := 0; i < attempts; i++ {
		last = c.Connect(ctx, target)
		if last.decided() {
			return Decide(last), last
		}
	}
	return Decide(last), last
}

// NetConnector is the production Connector: a TCP connect with a bounded timeout
// that closes the moment the handshake completes. It is non-root by construction
// — `connect()` needs no raw socket and no capability — which is the whole
// reason connect is chosen over SYN (§3.3). It is not exercised by the hermetic
// golden corpus.
type NetConnector struct {
	Timeout time.Duration
}

func (n NetConnector) Connect(ctx context.Context, target netip.AddrPort) ConnResult {
	timeout := n.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	// A target must be a valid literal IP: the leaf dials only pre-validated
	// addresses, never a hostname that would re-resolve at connect time with no
	// rebinding backstop. Reject an invalid target at entry, and back it with the
	// socket-level egress guard below so a non-globally-reachable literal fails
	// closed even if this invariant is ever broken upstream (#743).
	if !target.Addr().IsValid() {
		return ConnError
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// custody.EgressGuard is the same Control-hook backstop delivery (NewHTTPDoer)
	// and resolutionwalk (custodyDialer) install: it refuses the socket when the
	// address the kernel is about to connect to is non-globally-reachable (#743).
	d := net.Dialer{Control: custody.EgressGuard("connectoutcome")}
	conn, err := d.DialContext(dialCtx, "tcp", target.String())
	if err == nil {
		_ = conn.Close()
		return ConnOpen
	}
	// A connection refused is an answer (the port is shut); a deadline is
	// silence; anything else is a local error we could not see past.
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return ConnTimedOut
	}
	if isRefused(err) {
		return ConnRefused
	}
	return ConnError
}

// isRefused reports whether a dial error is a connection refusal (the host
// answered with an RST). It reads the OS error by errno where the platform
// exposes one and falls back to the error text — kept portable and free of a
// per-GOOS syscall constant so the binary is arch-neutral (v1 spec §4.1).
func isRefused(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "refused")
}
