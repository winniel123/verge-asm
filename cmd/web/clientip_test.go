package main

import (
	"net/http"
	"testing"
	"time"
)

func reqWith(remoteAddr, xff string) *http.Request {
	r := &http.Request{RemoteAddr: remoteAddr, Header: http.Header{}}
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

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
			name: "no config ignores xff", tp: trustedProxies{},
			remoteAddr: "10.0.0.9:5555", xff: "203.0.113.1", want: "10.0.0.9",
		},
		{
			name: "untrusted peer ignores xff", tp: trusted,
			remoteAddr: "198.51.100.4:5555", xff: "203.0.113.1", want: "198.51.100.4",
		},
		{
			name: "trusted peer single hop", tp: trusted,
			remoteAddr: "192.0.2.7:443", xff: "203.0.113.1", want: "203.0.113.1",
		},
		{
			name: "rightmost untrusted", tp: trusted,
			remoteAddr: "10.0.0.9:443", xff: "1.1.1.1, 203.0.113.1, 10.0.0.2", want: "203.0.113.1",
		},
		{
			name: "all trusted falls back to peer", tp: trusted,
			remoteAddr: "10.0.0.9:443", xff: "10.0.0.2, 192.0.2.7", want: "10.0.0.9",
		},
		{
			name: "trusted peer no header", tp: trusted,
			remoteAddr: "10.0.0.9:443", xff: "", want: "10.0.0.9",
		},
		{
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

func TestLoginIPKeyPerClientBehindProxy(t *testing.T) {
	proxy, err := parseTrustedProxies("192.0.2.7")
	if err != nil {
		t.Fatal(err)
	}
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	s := &server{trustedProxies: proxy, loginLimiter: newLoginLimiter(c.now)}

	const proxyAddr = "192.0.2.7:443"
	alice := reqWith(proxyAddr, "203.0.113.10")
	bob := reqWith(proxyAddr, "203.0.113.20")

	aliceAcct, aliceIP := loginAccountKey("alice"), s.loginIPKey(alice)
	bobAcct, bobIP := loginAccountKey("bob"), s.loginIPKey(bob)

	if aliceIP == bobIP {
		t.Fatalf("distinct clients behind the proxy share an IP key (%q); the proxy address, not the client, is being keyed", aliceIP)
	}

	for i := 0; i < s.loginLimiter.maxFailures; i++ {
		s.loginLimiter.fail(aliceAcct, aliceIP)
	}
	if !s.loginLimiter.locked(aliceAcct, aliceIP) {
		t.Fatal("alice's keys did not lock after reaching the threshold")
	}
	if s.loginLimiter.locked(bobAcct, bobIP) {
		t.Fatal("bob is rejected by alice's IP lock; the shared proxy address is being keyed instead of the client")
	}

	s.loginLimiter.fail(bobAcct, bobIP)
	s.loginLimiter.reset(bobAcct, bobIP)
	if s.loginLimiter.locked(bobAcct, bobIP) {
		t.Fatal("bob still throttled after a successful-auth reset")
	}
}

func TestAccountLockCeilingBoundsDenial(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newLoginLimiter(c.now)
	acct := loginAccountKey("admin")

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

	c2 := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l2 := newLoginLimiter(c2.now)
	ip := "ip:203.0.113.99"
	for i := 0; i < l2.maxFailures; i++ {
		l2.fail(ip)
	}
	c2.add(l2.acctLockCeiling + time.Minute)
	for i := 0; i < l2.maxFailures; i++ {
		l2.fail(ip)
	}
	if !l2.locked(ip) {
		t.Fatal("per-IP key released like an account key; the attacker host is no longer throttled")
	}
}
