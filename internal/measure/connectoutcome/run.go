package connectoutcome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

// Scope is the connect-outcome-specific payload of a JobSpec. It carries the
// Vantage and its class (recorded — the Custody gate has already run at dispatch,
// ADR-0019), the addresses to probe, the `verge-core` TCP ports to connect to,
// the UDP ports recorded-but-never-probed, and the safety profile — every offer
// enumerated so the Batch records what governed the probe by content (ADR-0025).
type Scope struct {
	Vantage      string        `json:"vantage"`
	VantageClass string        `json:"vantage_class"`
	Addresses    []string      `json:"addresses"`
	TCPPorts     []uint16      `json:"tcp_ports"`
	UDPPorts     []uint16      `json:"udp_ports,omitempty"`
	Profile      SafetyProfile `json:"profile"`
}

// DecodeScope reads a Scope from a JobSpec's opaque Scope payload.
func DecodeScope(spec wire.JobSpec) (Scope, error) {
	var s Scope
	if len(spec.Scope) == 0 {
		return Scope{}, fmt.Errorf("connectoutcome: job spec has no scope")
	}
	if err := json.Unmarshal(spec.Scope, &s); err != nil {
		return Scope{}, fmt.Errorf("connectoutcome: decode scope: %w", err)
	}
	return s, nil
}

// targets folds the scope's addresses and TCP ports into the `(Address, port)`
// set to connect to, in round-robin-by-host order (§3.3). Only TCP is probed;
// the UDP ports are recorded in scope and produce no target. An address that
// does not parse is skipped rather than guessed at — a malformed target is our
// own error and never a reachability value.
func (s Scope) targets() []netip.AddrPort {
	var raw []netip.AddrPort
	for _, a := range s.Addresses {
		addr, err := netip.ParseAddr(a)
		if err != nil {
			continue
		}
		addr = addr.Unmap()
		for _, port := range s.TCPPorts {
			raw = append(raw, netip.AddrPortFrom(addr, port))
		}
	}
	return RoundRobin(raw)
}

// Run executes the leaf against a live network for one JobSpec, writing NDJSON
// reachability observations to w. It is the production entrypoint the prober
// dispatches to; the golden corpus calls RunWithConnector against a scripted
// Connector instead. The production Connector is paced by the safety limiter so
// the connects honour the per-host rate, the global ceiling and the adaptive
// back-off (§3.3) — pacing that never changes a verdict, only its timing.
func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	base := NetConnector{Timeout: time.Duration(scope.Profile.ConnectTimeoutMillis) * time.Millisecond}
	paced := &pacedConnector{inner: base, pacer: NewPacer(scope.Profile), now: time.Now, sleep: time.Sleep}
	return RunWithConnector(context.Background(), paced, spec.Batch, scope, w)
}

// RunWithConnector executes the leaf against an arbitrary Connector. Separating
// the connector from Run is what lets one code path be driven by the paced
// network adapter in production and by a scripted connector in a hermetic test.
// It produces one `reachability` observation per TCP `Service` in scope — open
// or closed — and no observation for a UDP pair, which is recorded in the
// Batch's scope and never probed.
func RunWithConnector(ctx context.Context, c Connector, batch string, scope Scope, w io.Writer) error {
	var out []wire.Observation
	for _, target := range scope.targets() {
		outcome, raw := Probe(ctx, c, scope.Profile, target)
		out = append(out, EmitService(batch, scope.Vantage, target, outcome, raw))
	}
	return writeNDJSON(w, out)
}

// pacedConnector wraps a Connector with the safety limiter: before each connect
// it sleeps until the pacer says the host's next slot is due, and it feeds the
// pacer a stress signal on a timeout so a struggling host is backed off. The
// deadline is untouched — the wrapped connect keeps its own timeout — so the
// back-off changes only when the attempt starts, never how long it may run
// (ADR-0021).
type pacedConnector struct {
	inner Connector
	pacer *Pacer
	now   func() time.Time
	sleep func(time.Duration)
}

func (p *pacedConnector) Connect(ctx context.Context, target netip.AddrPort) ConnResult {
	due := p.pacer.Next(target.Addr(), p.now())
	if wait := due.Sub(p.now()); wait > 0 {
		p.sleep(wait)
	}
	res := p.inner.Connect(ctx, target)
	if res == ConnTimedOut {
		p.pacer.Signal(target.Addr(), StressTimeout)
	}
	return res
}

func writeNDJSON(w io.Writer, obs []wire.Observation) error {
	for _, o := range obs {
		if err := wire.EncodeObservation(w, o); err != nil {
			return err
		}
	}
	return nil
}
