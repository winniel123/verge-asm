package main

import (
	"net/http"
	"testing"
	"time"
)

// reqWith builds a bare request carrying the given RemoteAddr and, when non-empty,
// a single X-Forwarded-For header — the two inputs clientIP derives the limiter key
// from.
func reqWith(remoteAddr, xff string) *http.Request {
	r := &http.Request{RemoteAddr: remoteAddr, Header: http.Header{}}
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

// TestParseTrustedProxies covers the VERGE_TRUSTED_PROXIES parse: bare IPs, CIDRs,
// blank/whitespace entries, and a malformed entry that must be a hard error.
func TestParseTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
		nets    int
	}{
		{name: "empty", spec: "", nets: 0},
		{name: "blank entries", spec: " , ,\t", nets: 0},
		{name: "single ip", spec: "192.0.2.7", nets: 1},
		{name: "ip and cidr", spec: "10.0.0.0/8, 192.0.2.7", nets: 2},
		{name: "ipv6 cidr", spec: "2001:db8::/32", nets: 1},
		{name: "malformed ip", spec: "not-an-ip", wantErr: true},
		{name: "malformed cidr", spec: "10.0.0.0/99", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tp, err := parseTrustedProxies(tc.spec)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseTrustedProxies(%q) = nil error, want error", tc.spec)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTrustedProxies(%q): %v", tc.spec, err)
			}
			if len(tp.nets) != tc.nets {
				t.Fatalf("parseTrustedProxies(%q) parsed %d nets, want %d", tc.spec, len(tp.nets), tc.nets)
			}
		})
	}
}

// TestClientIP covers key derivation across the untrusted-peer (unchanged) and
// trusted-proxy paths, including the rightmost-untrusted XFF walk and the fallbacks.
func TestClientIP(t *testing.T) {
	trusted, err := parseTrustedProxies("10.0.0.0/8, 192.0.2.7")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		tp         trustedProxies
		remoteAddr string
		xff        string
		want       string
	}{
		{
			// No trusted proxy configured: the header is ignored, RemoteAddr host wins.
			name: "no config ignores xff", tp: trustedProxies{},
			remoteAddr: "10.0.0.9:5555", xff: "203.0.113.1", want: "10.0.0.9",
		},
		{
			// Peer is not a trusted proxy: header ignored even though one is configured.
			name: "untrusted peer ignores xff", tp: trusted,
			remoteAddr: "198.51.100.4:5555", xff: "203.0.113.1", want: "198.51.100.4",
		},
		{
			// Trusted proxy peer, one client hop: that hop is the client.
			name: "trusted peer single hop", tp: trusted,
			remoteAddr: "192.0.2.7:443", xff: "203.0.113.1", want: "203.0.113.1",
		},
		{
			// Trusted proxy peer, a spoofed prefix then the real client then the
			// trusted proxy: the rightmost UNTRUSTED entry (the real client) wins, and
			// the client-supplied spoof on the left is ignored.
			name: "rightmost untrusted", tp: trusted,
			remoteAddr: "10.0.0.9:443", xff: "1.1.1.1, 203.0.113.1, 10.0.0.2", want: "203.0.113.1",
		},
		{
			// Every hop trusted (proxy chain only): fall back to the peer host.
			name: "all trusted falls back to peer", tp: trusted,
			remoteAddr: "10.0.0.9:443", xff: "10.0.0.2, 192.0.2.7", want: "10.0.0.9",
		},
		{
			// Trusted peer but no XFF header: fall back to the peer host.
			name: "trusted peer no header", tp: trusted,
			remoteAddr: "10.0.0.9:443", xff: "", want: "10.0.0.9",
		},
		{
			// A malformed rightmost entry can't be a trusted proxy, so trust stops there.
			name: "malformed rightmost entry", tp: trusted,
			remoteAddr: "10.0.0.9:443", xff: "203.0.113.1, garbage", want: "garbage",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &server{trustedProxies: tc.tp}
			if got := s.clientIP(reqWith(tc.remoteAddr, tc.xff)); got != tc.want {
				t.Fatalf("clientIP(remote=%q xff=%q) = %q, want %q", tc.remoteAddr, tc.xff, got, tc.want)
			}
		})
	}
}

// TestLoginIPKeyPerClientBehindProxy is the #738 core guarantee: with a fixed
// RemoteAddr (the shared TLS-terminating proxy) but distinct X-Forwarded-For client
// IPs, the per-IP throttle key resolves per-client, so one client tripping the IP
// lock does NOT reject a different client — and a legitimate success clears its key.
func TestLoginIPKeyPerClientBehindProxy(t *testing.T) {
	proxy, err := parseTrustedProxies("192.0.2.7")
	if err != nil {
		t.Fatal(err)
	}
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	s := &server{trustedProxies: proxy, loginLimiter: newLoginLimiter(c.now)}

	const proxyAddr = "192.0.2.7:443"
	// Two operators sharing the proxy but on distinct client IPs, under two usernames.
	alice := reqWith(proxyAddr, "203.0.113.10")
	bob := reqWith(proxyAddr, "203.0.113.20")

	aliceAcct, aliceIP := loginAccountKey("alice"), s.loginIPKey(alice)
	bobAcct, bobIP := loginAccountKey("bob"), s.loginIPKey(bob)

	if aliceIP == bobIP {
		t.Fatalf("distinct clients behind the proxy share an IP key (%q); the proxy address, not the client, is being keyed", aliceIP)
	}

	// Alice hammers bad logins until her IP key locks.
	for i := 0; i < s.loginLimiter.maxFailures; i++ {
		s.loginLimiter.fail(aliceAcct, aliceIP)
	}
	if !s.loginLimiter.locked(aliceAcct, aliceIP) {
		t.Fatal("alice's keys did not lock after reaching the threshold")
	}
	// Bob, behind the same proxy but a different client IP, is unaffected.
	if s.loginLimiter.locked(bobAcct, bobIP) {
		t.Fatal("bob is rejected by alice's IP lock; the shared proxy address is being keyed instead of the client")
	}

	// A legitimate success for bob clears his keys (they were never locked, but the
	// reset must be a no-op-safe clear that leaves him able to authenticate).
	s.loginLimiter.fail(bobAcct, bobIP) // a mistype
	s.loginLimiter.reset(bobAcct, bobIP)
	if s.loginLimiter.locked(bobAcct, bobIP) {
		t.Fatal("bob still throttled after a successful-auth reset")
	}
}

// TestAccountLockCeilingBoundsDenial covers the #738 per-account blast-radius cap:
// an unauthenticated attacker who keeps a known username failing can deny it for at
// most acctLockCeiling; past that, locked() releases the ACCOUNT key so the real
// operator regains access, while the attacker-scoped IP key keeps locking.
func TestAccountLockCeilingBoundsDenial(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newLoginLimiter(c.now)
	acct := loginAccountKey("admin")

	// Sustain the lock well past the ceiling: every time the lock expires, fail again.
	// Advance in baseLockout steps so each iteration re-crosses an expired lock.
	sawReleaseWithinCeiling := false
	for elapsed := time.Duration(0); elapsed <= l.acctLockCeiling+2*l.baseLockout; elapsed += l.baseLockout {
		for i := 0; i < l.maxFailures; i++ {
			l.fail(acct)
		}
		if elapsed >= l.acctLockCeiling {
			if l.locked(acct) {
				t.Fatalf("account still locked at %s, past the %s ceiling", elapsed, l.acctLockCeiling)
			}
			sawReleaseWithinCeiling = true
		}
		c.add(l.baseLockout)
	}
	if !sawReleaseWithinCeiling {
		t.Fatal("test did not advance past the ceiling")
	}

	// The IP key (attacker-scoped) has no ceiling: it stays locked under the same
	// sustained failures, so the guessing host is still throttled.
	c2 := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l2 := newLoginLimiter(c2.now)
	ip := "ip:203.0.113.99"
	for i := 0; i < l2.maxFailures; i++ {
		l2.fail(ip)
	}
	// Advance past the ceiling; the IP lock is escalating and remains in force.
	c2.add(l2.acctLockCeiling + time.Minute)
	for i := 0; i < l2.maxFailures; i++ {
		l2.fail(ip)
	}
	if !l2.locked(ip) {
		t.Fatal("per-IP key released like an account key; the attacker host is no longer throttled")
	}
}
