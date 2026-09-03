package connectoutcome

import (
	"context"
	"io"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	"github.com/winniel123/verge-asm/internal/wire"
)

const FacetCertificate = "certificate"

// Under SNI two names on one address serve different chains, so a Service key manufactures drift.

func EndpointKey(serverName string, target netip.AddrPort, transport string) string {
	// The nameless endpoint is a distinguished variant; a real name is non-empty and never collides.
	return serverName + "@" + ServiceKey(target, transport)
}

// The not_after key and its RFC3339 form are what the Dashboard certs-expiring stat reads (#464).

type certificateValue struct {
	Outcome    TLSOutcome  `json:"outcome"`
	Chain      []string    `json:"chain,omitempty"`
	NotAfter   string      `json:"not_after,omitempty"`
	Issuer     string      `json:"issuer,omitempty"`
	Algorithm  string      `json:"algorithm,omitempty"`
	NotBefore  string      `json:"not_before,omitempty"`
	SANDNS     []string    `json:"san_dns,omitempty"`
	SANIP      []string    `json:"san_ip,omitempty"`
	ChainCerts []chainCert `json:"chain_certs,omitempty"`
}

type chainCert struct {
	Subject               string `json:"subject,omitempty"`
	Issuer                string `json:"issuer,omitempty"`
	SelfSignatureVerifies *bool  `json:"self_sig_verifies,omitempty"`
	KeyAlg                string `json:"key_alg,omitempty"`
	KeyBits               int    `json:"key_bits,omitempty"`
	KeyParamN             int    `json:"key_n_bits,omitempty"`
	SigDigest             string `json:"sig_digest,omitempty"`
}

func EmitCertificate(batch, vantage string, target netip.AddrPort, serverName string, res HandshakeResult) wire.Observation {
	// Emitted only for a reached Service: no variant of this value space means the port was shut.
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind,
		Facet:   FacetCertificate,
		Subject: EndpointKey(serverName, target, "tcp"),
		Vantage: vantage,
		Address: target.Addr().String(),
		Data: mustJSON(certificateValue{
			Outcome:    res.Outcome,
			Chain:      res.Chain,
			NotAfter:   notAfterString(res),
			Issuer:     presentedIdentity(res, res.Issuer),
			Algorithm:  presentedIdentity(res, res.Algorithm),
			NotBefore:  presentedNotBefore(res),
			SANDNS:     presentedStrings(res, res.SANDNS),
			SANIP:      presentedStrings(res, res.SANIP),
			ChainCerts: presentedChainCerts(res),
		}),
		// Raw material rides beside the value: the facet value stays the chain (ADR-0027).
		CertMaterial: presentedCertMaterial(res),
	}
}

func presentedCertMaterial(res HandshakeResult) *wire.CertMaterial {
	// The leaf fingerprint equals chain[0], so the side store joins a facet value (ADR-0027).
	if res.Outcome != TLSPresented || len(res.LeafDER) == 0 {
		return nil
	}
	return &wire.CertMaterial{
		Fingerprint: Fingerprint(res.LeafDER),
		DER:         res.LeafDER,
		SCTs: wire.EncodeSCTCapture(wire.SCTCapture{
			TLSExt: res.SCTsTLSExt,
			OCSP:   res.OCSPStaple,
		}),
		IssuerSPKI: res.IssuerSPKI,
	}
}

func presentedNotBefore(res HandshakeResult) string {
	if res.Outcome != TLSPresented || res.NotBefore.IsZero() {
		return ""
	}
	return res.NotBefore.UTC().Format(time.RFC3339)
}

func presentedStrings(res HandshakeResult, v []string) []string {
	if res.Outcome != TLSPresented {
		return nil
	}
	return v
}

func presentedChainCerts(res HandshakeResult) []chainCert {
	if res.Outcome != TLSPresented || len(res.ChainCerts) == 0 {
		return nil
	}
	out := make([]chainCert, 0, len(res.ChainCerts))
	for _, c := range res.ChainCerts {
		out = append(out, chainCert{
			Subject:               c.Subject,
			Issuer:                c.Issuer,
			SelfSignatureVerifies: c.SelfSignatureVerifies,
			KeyAlg:                c.KeyAlg,
			KeyBits:               c.KeyBits,
			KeyParamN:             c.KeyParamN,
			SigDigest:             c.SigDigest,
		})
	}
	return out
}

func presentedIdentity(res HandshakeResult, v string) string {
	if res.Outcome != TLSPresented {
		return ""
	}
	return v
}

func notAfterString(res HandshakeResult) string {
	// Every added key is omitempty, so a negative and a pre-parse span stay byte-identical (#712).
	if res.Outcome != TLSPresented || res.NotAfter.IsZero() {
		return ""
	}
	return res.NotAfter.UTC().Format(time.RFC3339)
}

func (s Scope) endpointNames() []string {
	if len(s.Names) == 0 {
		return []string{""}
	}
	return s.Names
}

func RunExchange(ctx context.Context, c Connector, h Handshaker, gen blanketdiscrim.PortGen, batch string, scope Scope, w io.Writer) error {
	// The control probe rides the same paced Connector, so it honours the §3.3 budget (ADR-0104).
	verdicts := discriminateBlanket(ctx, c, gen, scope)

	var out []wire.Observation
	for _, target := range scope.targets() {
		if v, ok := verdicts[target.Addr()]; ok && v.Gaps() {
			out = append(out, EmitServiceGap(batch, scope.Vantage, target,
				blanketdiscrim.GapCause, blanketdiscrim.ReasonFor(v)))
			continue
		}
		outcome, raw := Probe(ctx, c, scope.Profile, target)
		out = append(out, EmitService(batch, scope.Vantage, target, outcome, raw))
		if outcome != Reached {
			continue
		}
		for _, name := range scope.endpointNames() {
			res := h.Handshake(ctx, target, name)
			out = append(out, EmitCertificate(batch, scope.Vantage, target, name, res))
		}
	}
	return writeNDJSON(w, out)
}

func discriminateBlanket(ctx context.Context, c Connector, gen blanketdiscrim.PortGen, scope Scope) map[netip.Addr]blanketdiscrim.Verdict {
	// The control-port set is drawn once per batch and reused across addresses (ADR-0069).
	ports := gen.Ports()
	out := make(map[netip.Addr]blanketdiscrim.Verdict, len(scope.Addresses))
	for _, a := range scope.Addresses {
		addr, err := netip.ParseAddr(a)
		if err != nil {
			continue
		}
		addr = addr.Unmap()
		results := make([]blanketdiscrim.ControlResult, 0, len(ports))
		for _, port := range ports {
			_, raw := Probe(ctx, c, scope.Profile, netip.AddrPortFrom(addr, port))
			results = append(results, controlResultOf(raw))
		}
		out[addr] = blanketdiscrim.Decide(results)
	}
	return out
}

// Positive edge detection is refused: a selective edge and a default drop look alike (ADR-0125).

func controlResultOf(raw ConnResult) blanketdiscrim.ControlResult {
	// A silent-drop edge times out every control port, so Decide gaps and never blankets (#778).
	switch raw {
	case ConnOpen:
		return blanketdiscrim.ControlAnswered
	case ConnRefused:
		return blanketdiscrim.ControlClosed
	default:
		return blanketdiscrim.ControlIncomplete
	}
}
