package main

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type trustedProxies struct {
	nets []netip.Prefix
}

func parseTrustedProxies(spec string) (trustedProxies, error) {
	var tp trustedProxies
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// A skipped malformed entry would silently trust no proxy, so a typo must fail the deployment.
		if strings.Contains(part, "/") {
			p, err := netip.ParsePrefix(part)
			if err != nil {
				return trustedProxies{}, fmt.Errorf("trusted proxy CIDR %q: %w", part, err)
			}
			tp.nets = append(tp.nets, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(part)
		if err != nil {
			return trustedProxies{}, fmt.Errorf("trusted proxy IP %q: %w", part, err)
		}
		tp.nets = append(tp.nets, netip.PrefixFrom(a, a.BitLen()))
	}
	return tp, nil
}

func (tp trustedProxies) trusts(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, n := range tp.nets {
		if n.Contains(addr) {
			return true
		}
	}
	return false
}

func (s *server) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, perr := netip.ParseAddr(host)
	// An unnamed proxy is never trusted, so a forgeable header never moves the rate-limit key (ADR-0159 §1).
	if perr != nil || len(s.trustedProxies.nets) == 0 || !s.trustedProxies.trusts(peer) {
		return host
	}
	var entries []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		for _, part := range strings.Split(v, ",") {
			if p := strings.TrimSpace(part); p != "" {
				entries = append(entries, p)
			}
		}
	}
	// Walking from the right stops at the first unvouched hop, so a spoofed prefix never wins.
	for i := len(entries) - 1; i >= 0; i-- {
		a, aerr := netip.ParseAddr(entries[i])
		if aerr != nil {
			// A malformed entry cannot be a trusted proxy, so the chain of trust stops at it.
			return entries[i]
		}
		if s.trustedProxies.trusts(a) {
			continue
		}
		return a.Unmap().String()
	}
	return host
}

func (s *server) loginIPKey(r *http.Request) string {
	// Behind a proxy a per-proxy key makes the limiter a whole-instance login-denial lever.
	return "ip:" + s.clientIP(r)
}
