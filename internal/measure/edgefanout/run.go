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

// Scope is the edge-fanout-specific payload of a JobSpec: the candidate edge addresses
// to measure.
//
// There is no vantage — the `Scan` has no vantage dimension. There is no port list —
// the edge is measured on 443/tcp alone. There is no name list — the handshake sends no
// server name, so no name selects what the edge serves. An empty address list is a
// legible state: no custody extension is declared, so there is nothing to measure
// (CONTEXT.md `Scan`).
type Scope struct {
	Addresses []string `json:"addresses"`
}

// DecodeScope reads a Scope from a JobSpec's opaque Scope payload.
func DecodeScope(spec wire.JobSpec) (Scope, error) {
	var s Scope
	if len(spec.Scope) == 0 {
		return Scope{}, fmt.Errorf("edgefanout: job spec has no scope")
	}
	if err := json.Unmarshal(spec.Scope, &s); err != nil {
		return Scope{}, fmt.Errorf("edgefanout: decode scope: %w", err)
	}
	return s, nil
}

// Run executes the leaf against live TLS for one JobSpec, writing NDJSON `edge-fanout`
// observations to w. It is the production entrypoint the prober dispatches to; a test
// calls RunWithHandshaker against a scripted Handshaker instead.
func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	return RunWithHandshaker(context.Background(), NetHandshaker{Timeout: 3 * time.Second}, spec.Batch, scope, w)
}

// RunWithHandshaker executes the leaf against an arbitrary Handshaker. Separating the
// handshaker from Run is what lets one code path be driven by the network adapter in
// production and by a scripted handshaker in a hermetic test.
//
// It produces one observation per DISTINCT candidate address, in first-seen order. The
// dedup is what makes the handshake one connect per address: a candidate repeated in
// the scope — two in-zone names flattening to the same edge is the modal case — is
// measured once, and measuring it twice would tell us nothing the first handshake did
// not. An address that does not parse is our own error, never a measured value, so it
// is skipped and carries no row.
func RunWithHandshaker(ctx context.Context, h Handshaker, batch string, scope Scope, w io.Writer) error {
	seen := make(map[netip.AddrPort]struct{}, len(scope.Addresses))
	var out []wire.Observation
	for _, addr := range scope.Addresses {
		target, ok := edgeTarget(addr)
		if !ok {
			continue
		}
		if _, dup := seen[target]; dup {
			continue
		}
		seen[target] = struct{}{}
		out = append(out, Emit(batch, target, h.Handshake(ctx, target)))
	}
	return writeNDJSON(w, out)
}

// edgeTarget folds one candidate address to the measurement point — the address on
// 443/tcp — reporting false for an address that does not parse as a literal IP. The
// leaf dials only literal addresses, never a hostname that would re-resolve at connect
// time with no rebinding backstop (#743).
//
// Surrounding whitespace is trimmed, the way the recording-side scope gate trims
// (internal/queue/scopegate.go normAddr), so a candidate never fails to be measured
// over a spelling difference the two sides disagree about.
func edgeTarget(addr string) (netip.AddrPort, bool) {
	ip, err := netip.ParseAddr(strings.TrimSpace(addr))
	if err != nil {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(ip.Unmap(), Port), true
}

// writeNDJSON writes observations to w as NDJSON, one object per line, in order.
func writeNDJSON(w io.Writer, obs []wire.Observation) error {
	for _, o := range obs {
		if err := wire.EncodeObservation(w, o); err != nil {
			return err
		}
	}
	return nil
}
