package tlsacceptance

import (
	"crypto/tls"
	"errors"
	"io"
	"strings"
)

// This file is the seam between the declared candidate set and the linked
// `crypto/tls` library. The declared set is LITERAL (measurement-offers §1.4) — a
// fixed list of Go constant names — and this file maps those names to the library
// IDs the wire needs, refusing to put an undeclared or unofferable candidate on the
// wire. The offerability MAP is built once from the library's own enumeration, so
// `OfferableCiphers` is the exact check §1.4 requires: the build fails (via the
// offerability test) if any declared candidate is not offerable by the linked
// library, which is what discharges the unmeasured-offerability caveat.

// libraryCiphers is the library's full suite enumeration by constant name → ID,
// secure and insecure alike. It is the union `tls.CipherSuites() ∪
// tls.InsecureCipherSuites()` — read from the library, never hardcoded — so a Go
// upgrade that drops a suite drops it here, and the offerability test then fails on
// the now-unofferable declared candidate rather than letting the wire narrow
// silently (measurement-offers §1.4).
var libraryCiphers = func() map[string]uint16 {
	m := map[string]uint16{}
	for _, s := range tls.CipherSuites() {
		m[s.Name] = s.ID
	}
	for _, s := range tls.InsecureCipherSuites() {
		m[s.Name] = s.ID
	}
	return m
}()

// cipherNameByID is the reverse map, for rendering the negotiated suite back to its
// constant name.
var cipherNameByID = func() map[uint16]string {
	m := map[uint16]string{}
	for name, id := range libraryCiphers {
		m[id] = name
	}
	return m
}()

// OfferableCiphers reports which of the given declared suite names are NOT
// offerable by the linked library — the residue the §1.4 build gate refuses. An
// empty result is the passing state: declared ⊆ offerable. It is exported so the
// offerability test can name any candidate the library stopped offering.
func OfferableCiphers(declared []string) (missing []string) {
	for _, name := range declared {
		if _, ok := libraryCiphers[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

// versionID maps a declared version string to its library constant. An undeclared
// version is refused (ok=false) rather than defaulted, so nothing outside the
// candidate set ever reaches the wire.
func versionID(version string) (uint16, bool) {
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

// cipherIDs maps declared suite names to library IDs for the wire, silently
// dropping any name the library does not offer — the offerability test is what
// keeps that set empty for the declared candidates, so in a passing build this maps
// every offered name.
func cipherIDs(names []string) []uint16 {
	out := make([]uint16, 0, len(names))
	for _, n := range names {
		if id, ok := libraryCiphers[n]; ok {
			out = append(out, id)
		}
	}
	return out
}

// cipherName renders a negotiated suite ID back to its constant name for the value.
// Under TLS 1.3 the suite is the library's choice and not part of our declared
// offer, so it is not recorded (measurement-offers §1.3) and the name is empty.
func cipherName(id uint16, version string) string {
	if version == TLS13 {
		return ""
	}
	return cipherNameByID[id]
}

// spokeTLS reports whether a failed handshake dial nonetheless means the peer spoke
// TLS (a record- or alert-level rejection) rather than never speaking it (a reset,
// EOF or plaintext-looking failure). It is the same best-effort split the
// `certificate` handshake makes; the golden rows pin the accept-fold, not this live
// classification.
func spokeTLS(err error) bool {
	var recordErr tls.RecordHeaderError
	if errors.As(err, &recordErr) {
		return false // a malformed TLS record header — not TLS at all
	}
	if errors.Is(err, io.EOF) {
		return false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "first record does not look like a tls handshake"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "eof"):
		return false
	case strings.Contains(msg, "tls:"),
		strings.Contains(msg, "handshake failure"),
		strings.Contains(msg, "protocol version"),
		strings.Contains(msg, "no cipher suite"):
		return true // it spoke TLS and turned this version/suite set down
	}
	// An unclassifiable transport failure asserts no refusal we did not observe.
	return false
}
