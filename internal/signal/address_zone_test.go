package signal

import "testing"

func TestAnyNonGloballyReachable(t *testing.T) {
	cases := []struct {
		name  string
		addrs []string
		want  bool
	}{
		{"public v4", []string{"93.184.216.34"}, false},
		{"rfc1918", []string{"10.0.0.1"}, true},
		{"rfc1918 172", []string{"172.16.5.4"}, true},
		{"rfc1918 192.168", []string{"192.168.1.1"}, true},
		{"loopback", []string{"127.0.0.1"}, true},
		{"link-local v4", []string{"169.254.1.1"}, true},
		{"unspecified", []string{"0.0.0.0"}, true},
		{"public v6", []string{"2606:2800:220:1:248:1893:25c8:1946"}, false},
		{"ula v6", []string{"fd00::1"}, true},
		{"link-local v6", []string{"fe80::1"}, true},
		{"v4-mapped private", []string{"::ffff:10.0.0.1"}, true},
		{"mixed one leak", []string{"93.184.216.34", "10.1.2.3"}, true},
		{"unparseable ignored", []string{"not-an-address"}, false},
		{"empty", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := anyNonGloballyReachable(c.addrs); got != c.want {
				t.Fatalf("anyNonGloballyReachable(%v) = %v, want %v", c.addrs, got, c.want)
			}
		})
	}
}

func TestDeclaredNames(t *testing.T) {
	zone := `; example.com zone
$ORIGIN example.com.
$TTL 3600
@       IN SOA ns1.example.com. admin.example.com. ( 1 2 3 4 5 )
        IN NS  ns1.example.com.
www     IN A   93.184.216.34
        IN AAAA 2606:2800:220:1::1
api     IN CNAME www
*.wild  IN A   1.2.3.4
shop.example.com. IN A 5.6.7.8
`
	got := DeclaredNames(zone, "example.com")

	// Owner names, qualified against the origin and canonicalised.
	for _, want := range []string{"example.com", "www.example.com", "api.example.com", "shop.example.com"} {
		if !got[want] {
			t.Fatalf("expected %q declared; got set %v", want, got)
		}
	}
	// The AAAA line continues www's owner (leading whitespace) — no new name.
	// A wildcard owner is not a subject and must not be declared.
	if got["*.wild.example.com"] || got["wild.example.com"] {
		t.Fatalf("wildcard owner must not be a declared name: %v", got)
	}
	// A name the file never mentions is absent.
	if got["absent.example.com"] {
		t.Fatalf("absent name should not be declared")
	}
}

func TestDeclaredNamesFoldsCase(t *testing.T) {
	got := DeclaredNames("WWW IN A 1.2.3.4\n", "Example.COM")
	if !got["www.example.com"] {
		t.Fatalf("owner names must fold ASCII case like every Name key: %v", got)
	}
}
