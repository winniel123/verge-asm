package vantage

import (
	"fmt"
	"strconv"
	"strings"
)

const DefaultPort = 22

type Endpoint struct {
	Host     string
	Port     int
	Username string
}

func ParseEndpoint(host, port, username string) (Endpoint, error) {
	h := strings.TrimSpace(host)
	if h == "" {
		return Endpoint{}, fmt.Errorf("a host is required")
	}
	if strings.ContainsAny(h, " \t/\\") || strings.Contains(h, "://") {
		return Endpoint{}, fmt.Errorf("%q is not a bare host — enter a hostname or IP, no scheme or path", host)
	}

	p := DefaultPort
	if s := strings.TrimSpace(port); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 65535 {
			return Endpoint{}, fmt.Errorf("%q is not a port between 1 and 65535", port)
		}
		p = n
	}

	u := strings.TrimSpace(username)
	if u == "" {
		return Endpoint{}, fmt.Errorf("a username is required")
	}
	if strings.ContainsAny(u, " \t/\\:") {
		return Endpoint{}, fmt.Errorf("%q is not a bare username", username)
	}
	// A root login would hand a measurement position more of the host than it needs (v1 spec §4.2).
	if u == "root" {
		return Endpoint{}, fmt.Errorf("use a non-root username — the prober runs unprivileged")
	}

	return Endpoint{Host: h, Port: p, Username: u}, nil
}
