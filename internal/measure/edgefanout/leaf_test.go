package edgefanout

import (
	"bytes"
	"context"
	"encoding/json"
	"net/netip"
	"testing"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/wire"
)

var leafDER = []byte{0x30, 0x82, 0x01, 0x0a, 0xde, 0xad, 0xbe, 0xef}

// scriptHandshaker answers a fixed Result per target, so the fold and the run loop are
// exercised with no network.
type scriptHandshaker struct {
	byTarget map[netip.AddrPort]Result
	calls    []netip.AddrPort
}

func (h *scriptHandshaker) Handshake(_ context.Context, target netip.AddrPort) Result {
	h.calls = append(h.calls, target)
	return h.byTarget[target]
}

func TestFoldOutcomeSpace(t *testing.T) {
	fp := co.Fingerprint(leafDER)
	cases := []struct {
		name string
		in   co.HandshakeResult
		want Result
	}{
		{
			name: "presented carries the fingerprint and the DER",
			in:   co.HandshakeResult{Outcome: co.TLSPresented, Chain: []string{fp}, LeafDER: leafDER},
			want: Result{Outcome: Presented, Fingerprint: fp, LeafDER: leafDER},
		},
		{
			name: "tls-refused carries no fingerprint",
			in:   co.HandshakeResult{Outcome: co.TLSRefused},
			want: Result{Outcome: TLSRefused},
		},
		{
			name: "no-tls carries no fingerprint",
			in:   co.HandshakeResult{Outcome: co.NoTLS},
			want: Result{Outcome: NoTLS},
		},
		{
			name: "an unreached dial is unreachable, never no-tls",
			in:   co.HandshakeResult{Outcome: co.NoTLS, Unreachable: true},
			want: Result{Outcome: Unreachable},
		},
		{
			name: "a presented handshake with no chain is a refusal",
			in:   co.HandshakeResult{Outcome: co.TLSPresented},
			want: Result{Outcome: TLSRefused},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Fold(tc.in)
			if got.Outcome != tc.want.Outcome {
				t.Errorf("Fold outcome = %q, want %q", got.Outcome, tc.want.Outcome)
			}
			if got.Fingerprint != tc.want.Fingerprint {
				t.Errorf("Fold fingerprint = %q, want %q", got.Fingerprint, tc.want.Fingerprint)
			}
			if !bytes.Equal(got.LeafDER, tc.want.LeafDER) {
				t.Errorf("Fold DER = %x, want %x", got.LeafDER, tc.want.LeafDER)
			}
			if got.Outcome != Presented && got.Fingerprint != "" {
				t.Errorf("negative %q carries fingerprint %q, want none", got.Outcome, got.Fingerprint)
			}
		})
	}
}

// TestEmitDeclaresNoFacet pins the shape a membership-deciding leaf takes: the line
// names the Address it measured and declares no facet, no subject, no discriminator and
// no vantage, so it opens no timeline (ADR-0129 §6, #954 amendment).
func TestEmitDeclaresNoFacet(t *testing.T) {
	target := netip.MustParseAddrPort("198.51.100.7:443")
	obs := Emit("batch-1", target, Result{Outcome: Presented, Fingerprint: co.Fingerprint(leafDER), LeafDER: leafDER})

	if obs.Facet != "" {
		t.Errorf("Facet = %q, want empty — this leaf is not a facet", obs.Facet)
	}
	if obs.Subject != "" {
		t.Errorf("Subject = %q, want empty — the vetoed edge has no subject", obs.Subject)
	}
	if obs.Discriminator != "" {
		t.Errorf("Discriminator = %q, want empty", obs.Discriminator)
	}
	if obs.Vantage != "" {
		t.Errorf("Vantage = %q, want empty — the Scan has no vantage dimension", obs.Vantage)
	}
	if obs.Kind != Kind {
		t.Errorf("Kind = %q, want %q", obs.Kind, Kind)
	}
	if obs.Address != "198.51.100.7" {
		t.Errorf("Address = %q, want the measured edge address", obs.Address)
	}
}

// TestEmitMaterialRidesBesideTheValue pins ADR-0027's fence: the observation value
// carries the fingerprint alone, and the leaf DER rides beside it for the
// fingerprint-keyed side store. Every negative carries neither.
func TestEmitMaterialRidesBesideTheValue(t *testing.T) {
	target := netip.MustParseAddrPort("198.51.100.7:443")
	fp := co.Fingerprint(leafDER)

	obs := Emit("batch-1", target, Result{Outcome: Presented, Fingerprint: fp, LeafDER: leafDER})
	var value struct {
		Outcome     string          `json:"outcome"`
		Fingerprint string          `json:"fingerprint"`
		DER         json.RawMessage `json:"der"`
	}
	if err := json.Unmarshal(obs.Data, &value); err != nil {
		t.Fatalf("decode value: %v", err)
	}
	if value.Outcome != string(Presented) || value.Fingerprint != fp {
		t.Errorf("value = %+v, want presented with fingerprint %q", value, fp)
	}
	if len(value.DER) != 0 {
		t.Error("value carries the DER — it must ride beside the value, never inside it")
	}
	if obs.CertMaterial == nil {
		t.Fatal("CertMaterial = nil, want the leaf DER for the side store")
	}
	if obs.CertMaterial.Fingerprint != fp || !bytes.Equal(obs.CertMaterial.DER, leafDER) {
		t.Errorf("CertMaterial = %+v, want the served leaf keyed by %q", obs.CertMaterial, fp)
	}

	for _, out := range []Outcome{TLSRefused, NoTLS, Unreachable} {
		neg := Emit("batch-1", target, Result{Outcome: out})
		if neg.CertMaterial != nil {
			t.Errorf("%q carries CertMaterial, want none", out)
		}
		if bytes.Contains(neg.Data, []byte("fingerprint")) {
			t.Errorf("%q value = %s, want no fingerprint key", out, neg.Data)
		}
	}
}

func TestRunOneConnectPerAddress(t *testing.T) {
	a := netip.MustParseAddrPort("198.51.100.7:443")
	b := netip.MustParseAddrPort("203.0.113.9:443")
	h := &scriptHandshaker{byTarget: map[netip.AddrPort]Result{
		a: {Outcome: Presented, Fingerprint: co.Fingerprint(leafDER), LeafDER: leafDER},
		b: {Outcome: TLSRefused},
	}}

	var buf bytes.Buffer
	// The repeated candidate is spelled with surrounding whitespace, so the dedup is
	// pinned to survive a spelling difference rather than measuring the edge twice.
	scope := Scope{Addresses: []string{"198.51.100.7", "203.0.113.9", " 198.51.100.7 ", "not-an-address"}}
	if err := RunWithHandshaker(context.Background(), h, "batch-1", scope, &buf); err != nil {
		t.Fatalf("RunWithHandshaker: %v", err)
	}

	if want := []netip.AddrPort{a, b}; len(h.calls) != len(want) {
		t.Fatalf("handshakes = %v, want one per distinct address %v", h.calls, want)
	}
	if h.calls[0] != a || h.calls[1] != b {
		t.Errorf("handshakes = %v, want %v in first-seen order", h.calls, []netip.AddrPort{a, b})
	}

	sc := wire.NewObservationScanner(&buf)
	var got []wire.Observation
	for sc.Next() {
		got = append(got, sc.Observation())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan observations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("observations = %d, want 2", len(got))
	}
	if got[0].Address != "198.51.100.7" || got[1].Address != "203.0.113.9" {
		t.Errorf("addresses = %q/%q, want the two distinct candidates", got[0].Address, got[1].Address)
	}
}

// TestRunEmptyScopeIsLegible pins the empty-scope state: no custody extension is
// declared, so there is nothing to measure and the leaf writes nothing.
func TestRunEmptyScopeIsLegible(t *testing.T) {
	h := &scriptHandshaker{}
	var buf bytes.Buffer
	if err := RunWithHandshaker(context.Background(), h, "batch-1", Scope{}, &buf); err != nil {
		t.Fatalf("RunWithHandshaker: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("output = %q, want nothing", buf.String())
	}
	if len(h.calls) != 0 {
		t.Errorf("handshakes = %v, want none", h.calls)
	}
}

// TestDecodeScopeRejectsAnAbsentScope pins that a job spec with no scope is an error,
// never an empty measurement that would read as *this edge serves nothing*.
func TestDecodeScopeRejectsAnAbsentScope(t *testing.T) {
	if _, err := DecodeScope(wire.JobSpec{Batch: "batch-1", Kind: Kind}); err == nil {
		t.Error("DecodeScope(no scope) = nil error, want a failure")
	}
}
