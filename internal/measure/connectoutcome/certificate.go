package connectoutcome

import (
	"context"
	"io"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	"github.com/winniel123/verge-asm/internal/wire"
)

// FacetCertificate is the facet the tls-handshake step decides.
const FacetCertificate = "certificate"

// EndpointKey renders the `Endpoint` subject key for one `(Name, Service)` pair:
// `name@address:port/transport`, e.g. `api.example.com@198.51.100.1:443/tcp`. The
// Endpoint is the only key under which a presented certificate chain is
// single-valued — two names on one address and port serve, under SNI, different
// certificates, so keying `certificate` on the `Service` would manufacture drift
// on every virtual host every run (CONTEXT.md `Endpoint`).
//
// The `Name` may be ABSENT — *the default response to a client that names
// nothing* — a distinguished variant of the key and never an empty name. It
// renders with an empty name segment, `@address:port/transport`, and because a
// real name is always non-empty it can never collide with a named endpoint. Both
// legs are subjects, so this key inherits the Service (and its Address) form
// rather than restating it.
func EndpointKey(serverName string, target netip.AddrPort, transport string) string {
	return serverName + "@" + ServiceKey(target, transport)
}

// certificateValue is the JSON payload of a certificate observation: the outcome
// tag, plus — only on a presented chain — the ordered fingerprints, leaf first.
// The value space is a closed union: a presentation carries its chain, and the two
// negatives (`tls-refused`, `no-tls`) carry none, because each is a value in its
// own right and not an absence (CONTEXT.md `Certificate`).
//
// NotAfter is the leaf certificate's expiry as an RFC3339 string, carried only on a
// presented chain (the negatives omit it). It is the exact `not_after` key + format
// cmd/web/deltas.go reads for the Dashboard "Certs expiring ≤30d" stat (SPEC-CHANGE.md
// collision #8, #464).
//
// Issuer and Algorithm are the leaf's parsed identity (its issuer distinguished
// name and signature algorithm), carried only on a presented chain and dropped
// when empty. They are the leaf's own attributes — a read of the certificate the
// handshake already holds, not a new fingerprint — and the Asset detail's
// TLS-certificate card renders them beside the fingerprint and validity
// (SPEC-CHANGE.md collision #22c, #581). A presented value from before this leaf
// parsed them carries neither key, and the card omits each honestly.
type certificateValue struct {
	Outcome   TLSOutcome `json:"outcome"`
	Chain     []string   `json:"chain,omitempty"`
	NotAfter  string     `json:"not_after,omitempty"`
	Issuer    string     `json:"issuer,omitempty"`
	Algorithm string     `json:"algorithm,omitempty"`
	// --- v3 additions: leaf-only scalars ---
	NotBefore string   `json:"not_before,omitempty"` // leaf validity floor, RFC3339
	SANDNS    []string `json:"san_dns,omitempty"`    // leaf dNSName SANs, verbatim (wildcards NOT expanded)
	SANIP     []string `json:"san_ip,omitempty"`     // leaf iPAddress SANs, canonical string form
	// --- v3 addition: per-chain-cert parsed facts, leaf-first, index-aligned with Chain ---
	ChainCerts []chainCert `json:"chain_certs,omitempty"`
}

// chainCert is one presented link's parsed facts, read off the DER at handshake
// time and carried raw so every one of the four dark certificate rules derives its
// verdict AT READ (T-leaf #712 §0: store-raw, derive-at-read). self_sig_verifies is
// the ONE in-leaf-computed datum — a raw crypto fact needing the parsed key bytes,
// unavailable at read. Every field is omitempty so pre-v3 spans stay byte-identical.
type chainCert struct {
	Subject               string `json:"subject,omitempty"`           // this cert's subject DN
	Issuer                string `json:"issuer,omitempty"`            // this cert's issuer DN
	SelfSignatureVerifies *bool  `json:"self_sig_verifies,omitempty"` // signature validates against THIS cert's own key
	KeyAlg                string `json:"key_alg,omitempty"`           // "RSA"|"ECDSA"|"DSA"|"Ed25519"|"Ed448"|raw OID
	KeyBits               int    `json:"key_bits,omitempty"`          // RSA nlen | ECDSA len(n) | DSA L
	KeyParamN             int    `json:"key_n_bits,omitempty"`        // DSA subgroup N (bits); 0 for non-DSA
	SigDigest             string `json:"sig_digest,omitempty"`        // digest name: "MD5"|"SHA-1"|"SHA-256"|…
}

// EmitCertificate renders one Endpoint's presented-certificate value at one
// Vantage into an NDJSON observation. The subject is the `(Name, Service)` pair;
// the facet is `certificate`; the value is the closed union
// `presented(chain) │ tls-refused │ no-tls`. It is emitted ONLY for a Service the
// connect reached — neither negative can be read without knowing the port was
// open, and the value space has no variant meaning *the port was shut*
// (CONTEXT.md `Certificate`).
func EmitCertificate(batch, vantage string, target netip.AddrPort, serverName string, res HandshakeResult) wire.Observation {
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind, // shares the reachability dispatch — the handshake is a step in it
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
		// CertMaterial rides BESIDE Data, never inside it: the facet value stays the
		// fingerprint chain (ADR-0027), and the worker lands the raw CT inputs in the
		// certificate_material side store. Nil on every negative and on a scripted golden
		// row (no leaf DER), so the observation JSON stays byte-identical there (§5.3).
		CertMaterial: presentedCertMaterial(res),
	}
}

// presentedCertMaterial captures the leaf's raw CT inputs — the leaf DER (embedded SCTs
// ride inside it), the TLS-extension SCTs and the stapled OCSP response — for the
// certificate_material side store, carried only on a presented handshake whose leaf DER
// we read. The hermetic golden corpus scripts fingerprints with no DER, so it captures
// none and its goldens stay byte-identical. The fingerprint equals the facet value's
// chain[0], so a chain fingerprint joins the side store (ADR-0027, spec §5.3).
func presentedCertMaterial(res HandshakeResult) *wire.CertMaterial {
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

// presentedNotBefore renders the leaf's validity floor for the presented value,
// mirroring notAfterString: a zero NotBefore — every negative, and any presentation
// whose leaf carried none — renders the empty string, which omitempty drops, so only
// a presented chain carries the key (T-leaf #712).
func presentedNotBefore(res HandshakeResult) string {
	if res.Outcome != TLSPresented || res.NotBefore.IsZero() {
		return ""
	}
	return res.NotBefore.UTC().Format(time.RFC3339)
}

// presentedStrings guards a leaf-derived string slice (the dNSName / iPAddress SANs)
// so it is carried only on a presented chain — every negative outcome renders nil,
// which omitempty drops, exactly as notAfterString guards the expiry (T-leaf #712).
func presentedStrings(res HandshakeResult, v []string) []string {
	if res.Outcome != TLSPresented {
		return nil
	}
	return v
}

// presentedChainCerts projects the parsed per-link chain facts to the leaf's wire
// type, carried only on a presented chain — the two negatives leave it nil, which
// omitempty drops, so a pre-v3 span and every negative stay byte-identical (T-leaf
// #712). The ChainCert values are already parsed at handshake time; this only re-tags
// them to the fold's JSON shape, index-aligned with Chain, leaf-first.
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

// presentedIdentity guards a leaf-identity field (issuer, algorithm) so it is
// carried only on a presented chain — every negative outcome renders the empty
// string, which omitempty drops, exactly as notAfterString guards the expiry.
func presentedIdentity(res HandshakeResult, v string) string {
	if res.Outcome != TLSPresented {
		return ""
	}
	return v
}

// notAfterString renders the leaf's expiry for the presented value: a zero NotAfter
// — every negative, and any presentation whose leaf carried no expiry — renders the
// empty string, which `omitempty` drops, so only a presented chain carries the key.
func notAfterString(res HandshakeResult) string {
	if res.Outcome != TLSPresented || res.NotAfter.IsZero() {
		return ""
	}
	return res.NotAfter.UTC().Format(time.RFC3339)
}

// endpointNames folds a scope's declared server names to the set of Endpoints to
// hand the handshake for a reached Service. An empty set is the single NAMELESS
// endpoint — the only mode available on an address-scope Seed where no name is
// known yet — represented by an empty server name (no SNI). A named scope hands
// one Endpoint per name, each sending its own SNI.
func (s Scope) endpointNames() []string {
	if len(s.Names) == 0 {
		return []string{""} // the nameless endpoint
	}
	return s.Names
}

// RunExchange runs the reachability exchange with the blanket-discrimination
// control probe and the certificate handshake steps composed into it, writing both
// facets' NDJSON to w. It is the production path (Run dispatches here); the
// hermetic connect-outcome corpus drives the plain RunWithConnector, and the
// blanket-discrimination corpus drives this function against a scripted connector
// and a deterministic PortGen.
//
// Before probing an address's service ports it runs the control-port probe
// (blanketdiscrim, ADR-0104): a batch-generated set of dynamic-range ports a
// well-behaved origin refuses. Where the whole set answers, the address is a
// **blanket responder** and every one of its `Service`s folds to a `reachability`
// `Gap` — the sixth gap cause, never a value — and no certificate handshake runs
// (there is no reached Service to hand it). Where the probe did not complete the
// reach is a `Gap` for the same cause. Only a NotBlanket address takes the ordinary
// connect: a `reachability` value per Service, and the TLS handshake — one per
// Endpoint — on each Service the connect REACHED. The control probe rides the same
// paced Connector as the port tiers, so it honours the §3.3 safety budget exactly
// as they do (ADR-0104 Consequences).
//
// A later ticket adds an HTTP step to the same reached-Service branch (#198); this
// function is the composition point the orchestrator reconciles.
func RunExchange(ctx context.Context, c Connector, h Handshaker, gen blanketdiscrim.PortGen, batch string, scope Scope, w io.Writer) error {
	verdicts := discriminateBlanket(ctx, c, gen, scope)

	var out []wire.Observation
	for _, target := range scope.targets() {
		if v, ok := verdicts[target.Addr()]; ok && v.Gaps() {
			// A blanket responder (or an incomplete control probe): the reach is
			// undiscriminated, so the Service folds to a `Gap` with the sixth cause and
			// no certificate handshake runs.
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

// discriminateBlanket runs the blanket-discrimination control probe for every
// address in scope and returns the per-address verdict. The control-port set is
// drawn once per batch and reused across addresses (ADR-0069's per-batch draw); an
// address with no control ports to probe — an empty set — reads as a `Gap`, since
// the discrimination did not run. The control connects use the same Profile retry
// budget as the service connects, so a transient drop is retried before silence is
// read as incomplete.
func discriminateBlanket(ctx context.Context, c Connector, gen blanketdiscrim.PortGen, scope Scope) map[netip.Addr]blanketdiscrim.Verdict {
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

// controlResultOf projects a raw connect result to the blanket decision's closed
// union: an open handshake answers, a refusal is a decided closed port, and a
// timeout or local error (after Probe has spent its retries) is an incomplete
// reading we could not discriminate on.
//
// #778 determination: a provider edge that silently DROPS connects to non-proxied
// ports maps every control connect to ControlIncomplete here. The control ports are
// drawn from the dynamic range 49152-65535 (blanketdiscrim/ports.go), which such an
// edge never proxies, so every control connect times out, no port refuses, and
// blanketdiscrim.Decide returns VerdictGap — never VerdictBlanket. VerdictBlanket
// needs the WHOLE set to complete the handshake (Decide unanimity), which a dropped
// edge cannot reach; a refusing (RST) edge clears instead to NotBlanket. A live
// Cloudflare-fronted address therefore always reaches Inventory as an incomplete
// reach Gap — the shape the "Hide proxy edge" toggle now badges (cmd/web/inventory.go,
// inventoryProxyEdge). The incomplete Gap is the correct verdict for a silent-drop
// edge, so this leaf needs no change. ADR-0125 rules that positive provider-edge
// detection (answers on service ports, drops the control ports) is NOT added: that
// shape does not discriminate a port-selective edge from a plain default-drop origin,
// so a positive verdict off it would fabricate surface from an absence. The incomplete
// Gap stays the whole story.
func controlResultOf(raw ConnResult) blanketdiscrim.ControlResult {
	switch raw {
	case ConnOpen:
		return blanketdiscrim.ControlAnswered
	case ConnRefused:
		return blanketdiscrim.ControlClosed
	default: // ConnTimedOut, ConnError
		return blanketdiscrim.ControlIncomplete
	}
}
