// Package signalfacts derives a rule's facts from a subject's folded facet values.
// The web read path and the worker's census producers share it, so a rule is never
// evaluated on two derivations of one value (ADR-0024, ADR-0033 §3).
package signalfacts

import (
	"encoding/json"
	"net/netip"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

type CertificateValue struct {
	Outcome    string      `json:"outcome"`
	Chain      []string    `json:"chain"`
	NotAfter   string      `json:"not_after"`
	NotBefore  string      `json:"not_before"`
	SANDNS     []string    `json:"san_dns"`
	SANIP      []string    `json:"san_ip"`
	ChainCerts []ChainCert `json:"chain_certs"`
}

// These mirror connectoutcome's per-link parsed facts, so a producer edit must land here too.

type ChainCert struct {
	Subject               string `json:"subject"`
	Issuer                string `json:"issuer"`
	SelfSignatureVerifies *bool  `json:"self_sig_verifies"`
	KeyAlg                string `json:"key_alg"`
	KeyBits               int    `json:"key_bits"`
	KeyParamN             int    `json:"key_n_bits"`
	SigDigest             string `json:"sig_digest"`
}

func DecodeCertificate(raw []byte) CertificateValue {
	var v CertificateValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func CertDetailsFromValue(v CertificateValue, observedAt, now time.Time, serverName string) *signal.CertDetails {
	if v.Outcome != signal.CertPresented {
		return nil
	}
	d := &signal.CertDetails{}

	// A pre-v3 value has no not_before, so its horizon is underdetermined (ADR-0004 #67).
	if nb, nbErr := time.Parse(time.RFC3339, v.NotBefore); nbErr == nil {
		if na, naErr := time.Parse(time.RFC3339, v.NotAfter); naErr == nil {
			d.Clock = &signal.CertClock{NotBefore: nb, NotAfter: na, ObservedAt: observedAt.UTC(), EvaluatedAt: now.UTC()}
		}
	}

	// Under omitempty an empty san_dns is unreadable, so chain_certs is the read/unread witness.
	if len(v.ChainCerts) > 0 {
		if serverName != "" {
			m := SANMatchesName(v.SANDNS, serverName)
			d.SANMatchesName = &m
		}
		weak := WeakKeyOrSignature(v.ChainCerts)
		d.WeakKeyOrSignature = &weak
		if c0 := v.ChainCerts[0]; c0.SelfSignatureVerifies != nil {
			ss := SelfSignedOf(c0.Subject, c0.Issuer, *c0.SelfSignatureVerifies)
			d.SelfSigned = &ss
		}
	}
	return d
}

func SelfSignedOf(subject, issuer string, selfSigVerifies bool) bool {
	// Shared so the two rules cannot disagree.
	// Byte-exact on the presented rendering. RFC 5280 name preparation is refused.
	return subject == issuer && selfSigVerifies
}

func SANMatchesName(sanDNS []string, name string) bool {
	// A wildcard SAN admits no Name yet matches one here: matching is not admitting (ADR-0060).
	nameLabels := dnsLabels(name)
	if len(nameLabels) == 0 {
		return false
	}
	// Only dNSName SANs participate; RFC 6125 puts an iPAddress out of scope (ADR-0175 §3, #1342).
	for _, entry := range sanDNS {
		if sanEntryMatches(entry, nameLabels) {
			return true
		}
	}
	return false
}

func dnsLabels(name string) []string {
	labels := strings.Split(name, ".")
	if n := len(labels); n > 0 && labels[n-1] == "" {
		labels = labels[:n-1]
	}
	if len(labels) == 1 && labels[0] == "" {
		return nil
	}
	return labels
}

func sanEntryMatches(entry string, nameLabels []string) bool {
	entryLabels := dnsLabels(entry)
	if len(entryLabels) == 0 {
		return false
	}
	stars := 0
	for _, l := range entryLabels {
		stars += strings.Count(l, "*")
	}
	if stars == 0 {
		return labelsEqualFold(entryLabels, nameLabels)
	}
	// Same octets read as a pattern to one client and a literal to the next, so refuse (ADR-0060).
	if stars != 1 || entryLabels[0] != "*" {
		return false
	}
	if len(entryLabels) != len(nameLabels) {
		return false
	}
	if nameLabels[0] == "" {
		return false
	}
	return labelsEqualFold(entryLabels[1:], nameLabels[1:])
}

func labelsEqualFold(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

func WeakKeyOrSignature(chain []ChainCert) bool {
	// A self-signed link skips the signature limb only (weak-key-and-signature.md §4.1).
	weak := false
	for _, c := range chain {
		// An unnamed key algorithm is not weak, not unevaluable (weak-key-and-signature.md §4.2).
		switch c.KeyAlg {
		case "RSA":
			if c.KeyBits < 2048 {
				weak = true
			}
		case "ECDSA":
			if c.KeyBits < 224 {
				weak = true
			}
		case "DSA":
			if c.KeyBits < 2048 || c.KeyParamN < 224 {
				weak = true
			}
		}
		selfSig := c.SelfSignatureVerifies != nil && *c.SelfSignatureVerifies
		if !SelfSignedOf(c.Subject, c.Issuer, selfSig) {
			if c.SigDigest == "MD5" || c.SigDigest == "SHA-1" {
				weak = true
			}
		}
	}
	return weak
}

type HTTPIdentityValue struct {
	Outcome          string `json:"outcome"`
	Status           int    `json:"status"`
	Server           string `json:"server"`
	Title            string `json:"title"`
	WWWAuthenticate  string `json:"www_authenticate"`
	RedirectLocation string `json:"redirect_location"`
}

func DecodeHTTPIdentity(raw []byte) HTTPIdentityValue {
	var v HTTPIdentityValue
	_ = json.Unmarshal(raw, &v)
	return v
}

type TLSAcceptanceValue struct {
	Outcome  string `json:"outcome"`
	Versions []struct {
		Version string   `json:"version"`
		Ciphers []string `json:"ciphers"`
	} `json:"versions"`
}

func DecodeTLSAcceptance(raw []byte) TLSAcceptanceValue {
	var v TLSAcceptanceValue
	_ = json.Unmarshal(raw, &v)
	return v
}

func ParseServicePair(key string) (pair vergecore.Pair, addr string, ok bool) {
	slash := strings.LastIndex(key, "/")
	if slash < 0 {
		return vergecore.Pair{}, "", false
	}
	hostPort, transport := key[:slash], key[slash+1:]
	ap, err := netip.ParseAddrPort(hostPort)
	if err != nil {
		return vergecore.Pair{}, "", false
	}
	return vergecore.Pair{Port: ap.Port(), Transport: vergecore.Transport(transport)}, ap.Addr().String(), true
}

func SplitEndpointName(key string) (name, service string) {
	if at := strings.Index(key, "@"); at >= 0 {
		return key[:at], key[at+1:]
	}
	return "", key
}

// A missing facet leaves its rule outside the domain, never fired, so partial evidence is legal.

type ServiceEvidence struct {
	HasInternetReach bool
	InternetReach    string

	HasTLSAcceptance bool
	TLSAcceptance    []byte
}

func ServiceFactsFrom(subject string, ev ServiceEvidence, list vergecore.List) signal.ServiceFacts {
	f := signal.ServiceFacts{Subject: subject}
	if pair, _, ok := ParseServicePair(subject); ok {
		f.OnSensitiveList = pair.Transport == vergecore.TCP && list.IsSensitive(pair)
	}
	f.HasInternetReach = ev.HasInternetReach
	f.InternetReach = ev.InternetReach
	if ev.HasTLSAcceptance {
		if tls := DecodeTLSAcceptance(ev.TLSAcceptance); tls.Outcome == string(tlsacceptance.Enumerated) {
			f.TLSHandshakeCompleted = true
			f.TLSVersionsReadable = len(tls.Versions) > 0
			for _, ver := range tls.Versions {
				if ver.Version == tlsacceptance.TLS10 {
					f.TLS10Accepted = true
					break
				}
			}
		}
	}
	return f
}

type EndpointEvidence struct {
	HasCertificate bool
	Certificate    []byte
	CertObservedAt time.Time

	HasHTTPIdentity bool
	HTTPIdentity    []byte
}

func EndpointFactsFrom(subject string, ev EndpointEvidence, now time.Time, inEstate func(host string) bool) signal.EndpointFacts {
	name, _ := SplitEndpointName(subject)
	f := signal.EndpointFacts{Subject: subject, HasName: name != ""}
	if ev.HasCertificate {
		cv := DecodeCertificate(ev.Certificate)
		f.CertMeasured = true
		f.CertOutcome = cv.Outcome
		f.CertDetails = CertDetailsFromValue(cv, ev.CertObservedAt, now, name)
	}
	if ev.HasHTTPIdentity {
		id := DecodeHTTPIdentity(ev.HTTPIdentity)
		f.HTTPResponded = id.Outcome == httpexchange.OutcomeResponded
		f.HTTPStatus = id.Status
		f.RedirectLocation = id.RedirectLocation
		if inEstate != nil && f.HTTPResponded && f.HTTPStatus >= 300 && f.HTTPStatus <= 399 && id.RedirectLocation != "" {
			_, host := signal.RedirectTarget(id.RedirectLocation)
			f.RedirectHostInEstate = inEstate(host)
		}
	}
	return f
}
