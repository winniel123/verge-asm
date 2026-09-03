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

// There is no port list: the Scan enumerates open Services, not a port tier (ADR-0028).

type Scope struct {
	Vantage      string          `json:"vantage"`
	VantageClass string          `json:"vantage_class"`
	Services     []ServiceTarget `json:"services"`
	Candidates   CandidateSet    `json:"candidates"`
}

// Transport is not carried per target: TLS rides TCP, so there is nothing to vary.

type ServiceTarget struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
}

func (t ServiceTarget) addrPort() (netip.AddrPort, bool) {
	// A malformed target is our own error, never an acceptance value, so it is skipped.
	addr, err := netip.ParseAddr(t.Address)
	if err != nil {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(addr.Unmap(), t.Port), true
}

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

func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	base := NetEnumerator{Timeout: 3 * time.Second}
	return RunWithEnumerator(context.Background(), base, spec.Batch, scope, w)
}

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
