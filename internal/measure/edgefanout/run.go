package edgefanout

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

type Scope struct {
	Addresses []string `json:"addresses"` // the only dimension: no vantage, no port list, no name list
}

func DecodeScope(spec wire.JobSpec) (Scope, error) {
	var s Scope
	// An absent scope is an error, but an empty address list is legible: nothing to measure.
	if len(spec.Scope) == 0 {
		return Scope{}, fmt.Errorf("edgefanout: job spec has no scope")
	}
	if err := json.Unmarshal(spec.Scope, &s); err != nil {
		return Scope{}, fmt.Errorf("edgefanout: decode scope: %w", err)
	}
	return s, nil
}

func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	return RunWithHandshaker(context.Background(), NetHandshaker{Timeout: 3 * time.Second}, spec.Batch, scope, w)
}

func RunWithHandshaker(ctx context.Context, h Handshaker, batch string, scope Scope, w io.Writer) error {
	seen := make(map[netip.AddrPort]struct{}, len(scope.Addresses))
	var out []wire.Observation
	// A skipped candidate carries no row, and Custody reads an absent row as pending (CONTEXT.md).
	for _, addr := range scope.Addresses {
		target, ok := edgeTarget(addr)
		// An unparseable address is our own error, never a measured value.
		if !ok {
			continue
		}
		// Two in-zone names often flatten to one edge, and a second handshake would tell us nothing.
		if _, dup := seen[target]; dup {
			continue
		}
		seen[target] = struct{}{}
		out = append(out, Emit(batch, target, h.Handshake(ctx, target)))
	}
	return writeNDJSON(w, out)
}

func edgeTarget(addr string) (netip.AddrPort, bool) {
	// Trimmed the way internal/queue's scope gate trims, so the two sides agree on a spelling.
	ip, err := netip.ParseAddr(strings.TrimSpace(addr))
	// Never a hostname: it would re-resolve at connect time with no rebinding backstop (#743).
	if err != nil {
		return netip.AddrPort{}, false
	}
	// One handshake per address on 443/tcp, never a port tier (ADR-0028).
	return netip.AddrPortFrom(ip.Unmap(), Port), true
}

func writeNDJSON(w io.Writer, obs []wire.Observation) error {
	for _, o := range obs {
		if err := wire.EncodeObservation(w, o); err != nil {
			return err
		}
	}
	return nil
}
