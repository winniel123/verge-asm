package tlsacceptance

import (
	"crypto/tls"
	"errors"
	"io"
	"strings"

	"github.com/winniel123/verge-asm/internal/measure/tlsoffer"
)

func OfferableCiphers(declared []string) (missing []string) {
	return tlsoffer.Offerable(declared)
}

func versionID(version string) (uint16, bool) {
	return tlsoffer.VersionID(version)
}

func cipherIDs(names []string) []uint16 {
	return tlsoffer.CipherIDs(names)
}

func cipherName(id uint16, version string) string {
	if version == TLS13 {
		return ""
	}
	return tlsoffer.CipherName(id)
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
