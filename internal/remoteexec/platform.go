package remoteexec

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Platform is the remote prober's operating system and CPU, read from `uname` on
// the connection. GOOS/GOARCH are the Go identifiers the arch check matches the
// pushed binary against; Label is the accepted-platform chip the VantageCard renders
// (fixtures.json pins the "linux · x86_64" spelling — lowercase OS, a middle-dot
// separator, the raw uname machine — so the live datum reads back in the same shape).
type Platform struct {
	GOOS   string
	GOARCH string
	Label  string
}

// unameToGOOS maps the `uname -s` kernel name to the Go GOOS the prober matrix builds
// for. Only linux is in the matrix today (packaging-and-configuration.md §1.2); an
// unrecognised kernel yields "" so the arch check refuses rather than guessing.
func unameToGOOS(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "linux":
		return "linux"
	default:
		return ""
	}
}

// unameToGOARCH maps the `uname -m` machine name to the Go GOARCH. The two matrix
// architectures (amd64, arm64) each answer to more than one uname spelling; an
// unrecognised machine yields "" so no mismatched binary is ever selected.
func unameToGOARCH(m string) string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return ""
	}
}

// parsePlatform composes a Platform from the raw `uname -s` and `uname -m` outputs.
// It fails when either kernel or machine is unrecognised — the arch check must never
// push a binary for a platform it could not positively identify.
func parsePlatform(unameS, unameM string) (Platform, error) {
	goos := unameToGOOS(unameS)
	goarch := unameToGOARCH(unameM)
	if goos == "" || goarch == "" {
		return Platform{}, fmt.Errorf("remoteexec: unrecognised platform uname -s=%q uname -m=%q", strings.TrimSpace(unameS), strings.TrimSpace(unameM))
	}
	// The chip shows the OS lowercased and the raw uname machine (fixtures.json:
	// "linux · x86_64"), joined by a spaced middle dot.
	label := strings.ToLower(strings.TrimSpace(unameS)) + " · " + strings.TrimSpace(unameM)
	return Platform{GOOS: goos, GOARCH: goarch, Label: label}, nil
}

// parseEgress extracts the client source address from a raw SSH_CLIENT value. sshd
// sets SSH_CLIENT to "<clientip> <clientport> <serverport>" for the session, so the
// egress — the address the probe leaves the instance from, as this outside host sees
// it — is the first field. It returns ("", false) for an empty or malformed value or
// an address that does not parse, so a bad read collapses the chip rather than
// showing a fabricated address.
func parseEgress(sshClient string) (string, bool) {
	fields := strings.Fields(sshClient)
	if len(fields) == 0 {
		return "", false
	}
	addr, err := netip.ParseAddr(fields[0])
	if err != nil {
		return "", false
	}
	return addr.Unmap().String(), true
}

// normalizeDialled extracts the canonical IP string from an SSH transport peer
// address — the observed dialled address the Vantage-class derivation reads (#710).
// The peer address is a "host:port" TCP address (IPv6 as "[::1]:22"), so the port is
// stripped via netip's AddrPort parse; a bare address is accepted as a fallback. It
// returns the family-normalised, `Unmap`ed string (matching how egress is stored), or
// ("", false) for a nil or unparseable address so a bad read leaves the fact zero
// rather than fabricating one.
func normalizeDialled(a net.Addr) (string, bool) {
	if a == nil {
		return "", false
	}
	s := a.String()
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap().String(), true
	}
	if addr, err := netip.ParseAddr(s); err == nil {
		return addr.Unmap().String(), true
	}
	return "", false
}

// Fingerprint renders the SHA256 fingerprint of a pinned host key for the VantageCard
// host-key chip. The pinned value is the known_hosts key field (type + base64) the
// worker stored trust-on-first-use; this parses it and returns ssh.FingerprintSHA256,
// the canonical "SHA256:…" form. It returns "" for an empty or unparseable pin, so an
// un-pinned prober collapses the chip rather than showing a fabricated fingerprint.
func Fingerprint(pinnedHostKey string) string {
	pinnedHostKey = strings.TrimSpace(pinnedHostKey)
	if pinnedHostKey == "" {
		return ""
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pinnedHostKey))
	if err != nil {
		return ""
	}
	return ssh.FingerprintSHA256(key)
}
