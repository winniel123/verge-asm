package remoteexec

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Platform struct {
	GOOS   string
	GOARCH string
	Label  string
}

func unameToGOOS(s string) string {
	// The prober matrix builds linux alone (packaging-and-configuration.md §1.2).
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "linux":
		return "linux"
	default:
		return ""
	}
}

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

func parsePlatform(unameS, unameM string) (Platform, error) {
	goos := unameToGOOS(unameS)
	goarch := unameToGOARCH(unameM)
	if goos == "" || goarch == "" {
		return Platform{}, fmt.Errorf("remoteexec: unrecognised platform uname -s=%q uname -m=%q", strings.TrimSpace(unameS), strings.TrimSpace(unameM))
	}
	// fixtures.json pins "linux · x86_64", so the live datum must read back in that shape.
	label := strings.ToLower(strings.TrimSpace(unameS)) + " · " + strings.TrimSpace(unameM)
	return Platform{GOOS: goos, GOARCH: goarch, Label: label}, nil
}

func parseEgress(sshClient string) (string, bool) {
	// sshd sets SSH_CLIENT to "<clientip> <clientport> <serverport>", so egress is field one.
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

func normalizeDialled(a net.Addr) (string, bool) {
	if a == nil {
		return "", false
	}
	// A transport peer address is "host:port", IPv6 as "[::1]:22", so the port is stripped.
	s := a.String()
	// Unmap yields the family-normalised form the egress read is already stored in (#710).
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap().String(), true
	}
	if addr, err := netip.ParseAddr(s); err == nil {
		return addr.Unmap().String(), true
	}
	return "", false
}

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
