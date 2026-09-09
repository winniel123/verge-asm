// Package tlsoffer is the one declared TLS candidate set both TLS exchanges carry: the
// `certificate` handshake and the `tls-acceptance` enumeration (measurement-offers §1.1,
// ADR-0030 §3). Two lists would be two chances to hide a TLS-1.0-only listener (ADR-0025).
package tlsoffer

import "crypto/tls"

const (
	TLS10 = "1.0"
	TLS11 = "1.1"
	TLS12 = "1.2"
	TLS13 = "1.3"
)

func Versions() []string {
	// 1.2 and 1.3 are offered so a correctly-configured listener is never misfiled TLSRefused.
	return []string{TLS10, TLS11, TLS12, TLS13}
}

// The set is literal so a Go upgrade cannot silently widen the offer (measurement-offers §1.4).

// Go ignores Config.CipherSuites under TLS 1.3, so no 1.3 suite may sit in the value (§1.3).

func Ciphers() []string {
	// measurement-offers §1.3's nineteen suites, verbatim.
	return []string{
		// Limb 1 — accepting any of these is itself a finding (measurement-offers §1.3).
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
		"TLS_RSA_WITH_AES_128_CBC_SHA",
		"TLS_RSA_WITH_AES_256_CBC_SHA",
		"TLS_RSA_WITH_AES_128_CBC_SHA256",
		"TLS_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
		// Limb 2 — their absence would make the measurement false: the modal 1.2 set (§1.3).
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
	}
}

var libraryCiphers = func() map[string]uint16 {
	// Read from the library so a Go upgrade that drops a suite fails the offerability test (§1.4).
	m := map[string]uint16{}
	for _, s := range tls.CipherSuites() {
		m[s.Name] = s.ID
	}
	for _, s := range tls.InsecureCipherSuites() {
		m[s.Name] = s.ID
	}
	return m
}()

var cipherNameByID = func() map[uint16]string {
	m := map[uint16]string{}
	for name, id := range libraryCiphers {
		m[id] = name
	}
	return m
}()

func Offerable(declared []string) (missing []string) {
	// Exported so the §1.4 offerability gate can name a candidate the library stopped offering.
	for _, name := range declared {
		if _, ok := libraryCiphers[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

func VersionID(version string) (uint16, bool) {
	// An undeclared version is refused, not defaulted, so nothing outside the set is offered.
	switch version {
	case TLS10:
		return tls.VersionTLS10, true
	case TLS11:
		return tls.VersionTLS11, true
	case TLS12:
		return tls.VersionTLS12, true
	case TLS13:
		return tls.VersionTLS13, true
	default:
		return 0, false
	}
}

func CipherIDs(names []string) []uint16 {
	out := make([]uint16, 0, len(names))
	for _, n := range names {
		if id, ok := libraryCiphers[n]; ok {
			out = append(out, id)
		}
	}
	return out
}

func CipherName(id uint16) string {
	return cipherNameByID[id]
}
