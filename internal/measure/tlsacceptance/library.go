package tlsacceptance

import (
	"crypto/tls"
	"errors"
	"io"
	"strings"
)

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

func OfferableCiphers(declared []string) (missing []string) {
	// Exported so the §1.4 offerability gate can name a candidate the library stopped offering.
	for _, name := range declared {
		if _, ok := libraryCiphers[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

func versionID(version string) (uint16, bool) {
	// An undeclared version is refused rather than defaulted, so nothing outside the set is offered.
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

func cipherIDs(names []string) []uint16 {
	out := make([]uint16, 0, len(names))
	for _, n := range names {
		if id, ok := libraryCiphers[n]; ok {
			out = append(out, id)
		}
	}
	return out
}

func cipherName(id uint16, version string) string {
	if version == TLS13 {
		return ""
	}
	return cipherNameByID[id]
}

func spokeTLS(err error) bool {
	var recordErr tls.RecordHeaderError
	if errors.As(err, &recordErr) {
		return false
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
		return true
	}
	// An unclassifiable transport failure asserts no refusal we did not observe.
	return false
}
