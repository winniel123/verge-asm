package tlsacceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

// Scope is the tls-acceptance-specific payload of a JobSpec. It carries the Vantage
// and its class (recorded), the open Services to enumerate — each already known to
// have a `reached` connect, since the Scan's population is the open `Service` set —
// and the declared candidate set, every offer enumerated so the Batch records what
// went on the wire by content (ADR-0025). There is NO port list: the Services carry
// their own ports, inherited from reachability, because the Scan is an enumeration
// over open Services and not a port tier (ADR-0028).
type Scope struct {
	Vantage      string          `json:"vantage"`
	VantageClass string          `json:"vantage_class"`
	Services     []ServiceTarget `json:"services"`
	Candidates   CandidateSet    `json:"candidates"`
}

// ServiceTarget is one open `(Address, port)` the enumeration runs against. The
// transport is always TCP — TLS rides a connection-oriented transport — so it is
// not carried per target.
type ServiceTarget struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
}

// addrPort folds a target to its netip form, skipping one that does not parse — a
// malformed target is our own error and never an acceptance value.
func (t ServiceTarget) addrPort() (netip.AddrPort, bool) {
	addr, err := netip.ParseAddr(t.Address)
	if err != nil {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(addr.Unmap(), t.Port), true
}

// DecodeScope reads a Scope from a JobSpec's opaque Scope payload.
func DecodeScope(spec wire.JobSpec) (Scope, error) {
	var s Scope
	if len(spec.Scope) == 0 {
		return Scope{}, fmt.Errorf("tlsacceptance: job spec has no scope")
	}
	if err := json.Unmarshal(spec.Scope, &s); err != nil {
		return Scope{}, fmt.Errorf("tlsacceptance: decode scope: %w", err)
	}
	return s, nil
}

// Run executes the leaf against live TLS for one JobSpec, writing NDJSON
// tls-acceptance observations to w. It is the production entrypoint the prober
// dispatches to; the golden corpus calls RunWithEnumerator against a scripted
// Enumerator instead. The network enumerator is best-effort paced to the candidate
// set's per-host handshake ceiling — pacing that never changes which versions or
// suites a listener accepted, only the timing (ADR-0021).
func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	base := NetEnumerator{Timeout: 3 * time.Second}
	return RunWithEnumerator(context.Background(), base, spec.Batch, scope, w)
}

// RunWithEnumerator executes the leaf against an arbitrary Enumerator. Separating
// the enumerator from Run is what lets one code path be driven by the network
// adapter in production and by a scripted enumerator in a hermetic test. It
// produces one `tls-acceptance` observation per Service in scope — enumerated,
// tls-refused or no-tls — since each is a value the enumeration measured against a
// Service already known open.
func RunWithEnumerator(ctx context.Context, e Enumerator, batch string, scope Scope, w io.Writer) error {
	var out []wire.Observation
	for _, svc := range scope.Services {
		target, ok := svc.addrPort()
		if !ok {
			continue
		}
		value := Enumerate(ctx, e, scope.Candidates, target)
		out = append(out, EmitAcceptance(batch, scope.Vantage, target, value))
	}
	return writeNDJSON(w, out)
}

func writeNDJSON(w io.Writer, obs []wire.Observation) error {
	for _, o := range obs {
		if err := wire.EncodeObservation(w, o); err != nil {
			return err
		}
	}
	return nil
}
