package vantage

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

func ParseResolver(resolver string) (string, error) {
	s := strings.TrimSpace(resolver)
	// A blank resolver is refused, never defaulted, so no undeclared address is dialled (ADR-0202).
	if s == "" {
		return "", fmt.Errorf("a recursive resolver is required — a vantage measures through one")
	}
	if strings.ContainsAny(s, " \t/\\") || strings.Contains(s, "://") {
		return "", fmt.Errorf("%q is not a bare address — enter host or host:port, no scheme or path", resolver)
	}
	if host, port, err := net.SplitHostPort(s); err == nil {
		if strings.TrimSpace(host) == "" {
			return "", fmt.Errorf("a resolver host is required")
		}
		n, cerr := strconv.Atoi(port)
		if cerr != nil || n < 1 || n > 65535 {
			return "", fmt.Errorf("%q is not a port between 1 and 65535", port)
		}
		return s, nil
	}
	if strings.Contains(s, ":") {
		if _, err := netip.ParseAddr(s); err != nil {
			return "", fmt.Errorf("%q is not a host or host:port", resolver)
		}
	}
	return s, nil
}
