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

	for _, want := range []string{"example.com", "www.example.com", "api.example.com", "shop.example.com"} {
		if !got[want] {
			t.Fatalf("expected %q declared; got set %v", want, got)
		}
	}
	if got["*.wild.example.com"] || got["wild.example.com"] {
		t.Fatalf("wildcard owner must not be a declared name: %v", got)
	}
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

func TestDeclaredNamesSkipsHighBitOwners(t *testing.T) {
	got := DeclaredNames("café IN A 1.2.3.4\nwww IN A 1.2.3.5\n", "example.com")
	if got["café.example.com"] {
		t.Fatalf("a high-bit owner typed as text is not a subject (ADR-0055): %v", got)
	}
	if !got["www.example.com"] {
		t.Fatalf("ordinary owner missing: %v", got)
	}
}

func TestDelegatedSubzones(t *testing.T) {
	zone := `$ORIGIN example.com.
@       IN SOA ns1.example.com. admin.example.com. ( 1 2 3 4 5 )
        IN NS  ns1.example.com.
        IN NS  ns2.example.com.
www     IN A   93.184.216.34
sub     IN NS  ns1.other.
        IN ns  ns2.other.
ns1.sub IN A   198.51.100.1
ttl     3600 IN NS ns1.other.
bare    NS ns1.other.
abs.example.com. IN NS ns1.other.
$ORIGIN corp.example.com.
@       IN NS  ns1.corp.
dev     IN NS  ns1.corp.
`
	got := DelegatedSubzones(zone, "example.com")

	for _, want := range []string{
		"sub.example.com", "ttl.example.com", "bare.example.com", "abs.example.com",
		"corp.example.com", "dev.corp.example.com",
	} {
		if !got[want] {
			t.Fatalf("expected %q delegated; got set %v", want, got)
		}
	}
	for _, reject := range []string{"example.com", "www.example.com", "ns1.sub.example.com"} {
		if got[reject] {
			t.Fatalf("%q is not a delegation point: %v", reject, got)
		}
	}
}

func TestDelegatedSubzonesInheritedOwnerAfterOrigin(t *testing.T) {
	zone := "$ORIGIN example.com.\nsub IN A 1.2.3.4\n$ORIGIN other.example.com.\n IN NS ns1.other.\n"
	got := DelegatedSubzones(zone, "example.com")
	if !got["sub.example.com"] || len(got) != 1 {
		t.Fatalf("an inherited owner is the last explicit owner, not the new origin: %v", got)
	}
}

func TestDelegatedSubzonesFoldsCase(t *testing.T) {
	got := DelegatedSubzones("SUB IN NS ns1.other.\n", "Example.COM")
	if !got["sub.example.com"] {
		t.Fatalf("delegation owners must fold ASCII case like every Name key: %v", got)
	}
}
