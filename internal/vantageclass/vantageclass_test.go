package vantageclass

import (
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
)

func TestPresentedAddrs(t *testing.T) {
	cases := []struct {
		name    string
		dialled string
		egress  string
		want    []string
	}{
		{"no facts", "", "", nil},
		{"dialled only", "203.0.113.9", "", []string{"203.0.113.9"}},
		{"egress only", "", "198.51.100.7", []string{"198.51.100.7"}},
		{"both, dialled first", "203.0.113.9", "198.51.100.7", []string{"203.0.113.9", "198.51.100.7"}},
		{"unparseable drops out", "10.0.0.1", "garbage", []string{"10.0.0.1"}},
		{"ipv6", "2001:db8::1", "", []string{"2001:db8::1"}},
		{"dual-stack residue: only observed addrs", "203.0.113.9", "2001:db8::9",
			[]string{"203.0.113.9", "2001:db8::9"}},
		{"v4-mapped v6 unmapped", "::ffff:10.0.0.1", "", []string{"10.0.0.1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PresentedAddrs(c.dialled, c.egress)
			if len(got) != len(c.want) {
				t.Fatalf("PresentedAddrs(%q,%q) = %v, want %v", c.dialled, c.egress, got, c.want)
			}
			for i := range got {
				if got[i].String() != c.want[i] {
					t.Errorf("addr[%d] = %s, want %s", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestDerive(t *testing.T) {
	scope := netip.MustParsePrefix("10.0.0.0/8")
	covered := func(a netip.Addr) bool { return scope.Contains(a.Unmap()) }

	cases := []struct {
		name    string
		dialled string
		egress  string
		want    custody.VantageClass
	}{
		{"no presented address is unverified", "", "", custody.ClassUnverified},
		{"every presented covered is internal", "10.1.2.3", "10.4.5.6", custody.ClassInternal},
		{"one uncovered is internet (closed direction)", "10.1.2.3", "52.1.2.3", custody.ClassInternet},
		{"a single public presented is internet", "203.0.113.9", "", custody.ClassInternet},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Derive(c.dialled, c.egress, covered); got != c.want {
				t.Errorf("Derive(%q,%q) = %q, want %q", c.dialled, c.egress, got, c.want)
			}
		})
	}
}
