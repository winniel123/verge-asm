package main

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// trustedProxies is the set of proxy addresses whose X-Forwarded-For header web
// trusts when deriving a client IP for the rate-limit key ONLY (#738). It is
// parsed from VERGE_TRUSTED_PROXIES, a comma-separated list of IPs and/or CIDRs
// (e.g. "10.0.0.0/8, 192.0.2.7"). Empty (the default) means no proxy is trusted:
// the client IP is always the immediate RemoteAddr and no forwarding header is
// consulted — identical to the pre-#738 behaviour.
//
// This governs the limiter key alone. X-Forwarded-For is NEVER read for identity
// or authorization anywhere in the auth path (v1 spec §4.3, §7); a spoofed header
// can at most spread or concentrate an attacker's OWN failed-attempt budget, never
// grant access.
type trustedProxies struct {
	nets []netip.Prefix
}

// parseTrustedProxies parses the VERGE_TRUSTED_PROXIES spec: a comma-separated
// list where each entry is either a bare IP (treated as a /32 or /128) or a CIDR.
// Blank entries are skipped; an empty spec yields the zero value, which trusts
// nothing. A malformed entry is a hard configuration error so a typo fails the
// deployment loudly rather than silently trusting no proxy.
func parseTrustedProxies(spec string) (trustedProxies, error) {
	var tp trustedProxies
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
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

// trusts reports whether addr falls inside any configured trusted-proxy range.
func (tp trustedProxies) trusts(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, n := range tp.nets {
		if n.Contains(addr) {
			return true
		}
	}
	return false
}

// clientIP derives the client IP the rate-limit key is counted against (#738). It
// is used ONLY for the limiter key, NEVER for identity or authorization.
//
// When no trusted proxy is configured, or the immediate peer (RemoteAddr) is not a
// trusted proxy, the header is not consulted at all and the peer host is returned —
// identical to the pre-#738 behaviour, so a direct-facing deployment is unchanged.
//
// When the peer IS a trusted proxy, the client is the rightmost X-Forwarded-For
// entry that is not itself a trusted proxy: the closest hop the trusted chain
// cannot vouch for, and thus the furthest-right value an attacker could still have
// forged. Walking from the right and stopping at the first untrusted entry means a
// client-supplied prefix of spoofed entries can never be mistaken for the real
// source. If every entry is a trusted proxy (or the header is absent), it falls
// back to the peer host.
func (s *server) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, perr := netip.ParseAddr(host)
	if perr != nil || len(s.trustedProxies.nets) == 0 || !s.trustedProxies.trusts(peer) {
		return host
	}
	// Flatten every X-Forwarded-For header (there may be more than one) into a
	// single left-to-right list of hops.
	var entries []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		for _, part := range strings.Split(v, ",") {
			if p := strings.TrimSpace(part); p != "" {
				entries = append(entries, p)
			}
		}
	}
	for i := len(entries) - 1; i >= 0; i-- {
		a, aerr := netip.ParseAddr(entries[i])
		if aerr != nil {
			// A malformed entry cannot be a trusted proxy, so the chain of trust
			// stops here: this is the closest hop we can no longer vouch for.
			return entries[i]
		}
		if s.trustedProxies.trusts(a) {
			continue
		}
		return a.Unmap().String()
	}
	// Every hop was a trusted proxy (or the header was absent): the peer is the
	// nearest thing to a client we can name.
	return host
}

// loginIPKey is the per-source throttle key a credential attempt is counted
// against (#322, #738): the derived client IP. Behind a configured trusted proxy
// this resolves per-client rather than per-proxy, so the limiter defends rather
// than becoming a whole-instance login-denial lever; unconfigured it is the request
// RemoteAddr, never a proxy-supplied header. It is used for the limiter key only,
// never for identity or authorization.
func (s *server) loginIPKey(r *http.Request) string {
	return "ip:" + s.clientIP(r)
}
