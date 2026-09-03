package connectoutcome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	"github.com/winniel123/verge-asm/internal/wire"
)

// The vantage class is recorded and never re-checked: the Custody gate ran at dispatch (ADR-0019).

type Scope struct {
	Vantage      string   `json:"vantage"`
	VantageClass string   `json:"vantage_class"`
	Addresses    []string `json:"addresses"`
	TCPPorts     []uint16 `json:"tcp_ports"`
	UDPPorts     []uint16 `json:"udp_ports,omitempty"`
	// Empty is the nameless endpoint, the one mode an address-scope Seed has (CONTEXT.md `Endpoint`).

	Names   []string      `json:"names,omitempty"`
	Profile SafetyProfile `json:"profile"`
}

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

func (s Scope) targets() []netip.AddrPort {
	var raw []netip.AddrPort
	// A UDP port is recorded and never probed: a connect decides no honest UDP value (ADR-0083).
	for _, a := range s.Addresses {
		addr, err := netip.ParseAddr(a)
		if err != nil {
			// A malformed target is our own error and never a reachability value.
			continue
		}
		addr = addr.Unmap()
		for _, port := range s.TCPPorts {
			raw = append(raw, netip.AddrPortFrom(addr, port))
		}
	}
	return RoundRobin(raw)
}

func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	base := NetConnector{Timeout: time.Duration(scope.Profile.ConnectTimeoutMillis) * time.Millisecond}
	paced := &pacedConnector{inner: base, pacer: NewPacer(scope.Profile), now: time.Now, sleep: time.Sleep}
	hs := NetHandshaker{Timeout: time.Duration(scope.Profile.ConnectTimeoutMillis) * time.Millisecond}
	return RunExchange(context.Background(), paced, hs, blanketdiscrim.CryptoPorts{}, spec.Batch, scope, w)
}

func RunWithConnector(ctx context.Context, c Connector, batch string, scope Scope, w io.Writer) error {
	var out []wire.Observation
	for _, target := range scope.targets() {
		outcome, raw := Probe(ctx, c, scope.Profile, target)
		out = append(out, EmitService(batch, scope.Vantage, target, outcome, raw))
	}
	return writeNDJSON(w, out)
}

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
	// The back-off moves when an attempt starts and never how long it may run (ADR-0021).
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
