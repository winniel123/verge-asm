package signal

import (
	"net/netip"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestSpecialPurposeFiringSet(t *testing.T) {
	cases := []struct {
		addr  string
		fires bool
	}{
		{"100.64.0.1", true},
		{"192.0.2.1", true},
		{"198.18.0.1", true},
		{"240.0.0.1", true},
		{"2001:db8::1", true},
		{"10.1.2.3", true},
		{"172.31.255.254", true},
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"169.254.1.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"192.0.0.8", true},
		{"192.0.0.170", true},
		{"192.0.0.171", true},
		{"192.88.99.2", true},
		{"198.51.100.7", true},
		{"203.0.113.9", true},
		{"::1", true},
		{"::", true},
		{"64:ff9b:1::1", true},
		{"100::1", true},
		{"100:0:0:1::1", true},
		{"2001:100::1", true},
		{"2001:2::1", true},
		{"3fff::1", true},
		{"5f00::1", true},
		{"fd00::1", true},
		{"fe80::1", true},
		{"::ffff:10.0.0.1", true},

		{"8.8.8.8", false},
		{"2606:4700::1111", false},
		{"93.184.216.34", false},
		// Marked reachable by the registry, each inside a wider not-reachable block: longest match.
		{"192.0.0.9", false},
		{"192.0.0.10", false},
		{"2001:1::1", false},
		{"2001:1::2", false},
		{"2001:1::3", false},
		{"192.31.196.1", false},
		{"192.52.193.1", false},
		{"192.175.48.1", false},
		{"64:ff9b::1", false},
		{"2001:3::1", false},
		{"2001:4:112::1", false},
		{"2001:20::1", false},
		{"2001:30::1", false},
		{"2620:4f:8000::1", false},
		// N/A or terminated: the owner supplied no value, so neither do we.
		{"192.88.99.1", false},
		{"192.88.99.3", false},
		{"2001::1234:5678", false},
		{"2001:10::1", false},
		{"2002::1", false},
	}
	for _, c := range cases {
		t.Run(c.addr, func(t *testing.T) {
			addr := netip.MustParseAddr(c.addr).Unmap()
			if got := specialPurposeFires(addr); got != c.fires {
				t.Fatalf("specialPurposeFires(%s) = %v, want %v", c.addr, got, c.fires)
			}
			if got := anyNonGloballyReachable([]string{c.addr}); got != c.fires {
				t.Fatalf("anyNonGloballyReachable(%s) = %v, want %v", c.addr, got, c.fires)
			}
		})
	}
}

var (
	noteBlock    = regexp.MustCompile("`([^`]+)`")
	noteFootnote = regexp.MustCompile(`^\[\d+\]$`)
)

func TestSpecialPurposeTableMatchesResearchNote(t *testing.T) {
	raw, err := os.ReadFile("../../docs/research/special-purpose-address-registry.md")
	if err != nil {
		t.Fatalf("research note unavailable: %v", err)
	}
	want := map[string]bool{}
	rows, firing := 0, 0
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 6 {
			continue
		}
		var blocks []string
		for _, m := range noteBlock.FindAllStringSubmatch(cells[1], -1) {
			if noteFootnote.MatchString(m[1]) {
				continue
			}
			p, err := netip.ParsePrefix(m[1])
			if err != nil {
				blocks = nil
				break
			}
			blocks = append(blocks, p.String())
		}
		cell := strings.Fields(strings.ReplaceAll(cells[3], "*", ""))
		if len(blocks) == 0 || len(cell) == 0 {
			continue
		}
		fires := cell[0] == "False"
		rows++
		if fires {
			firing++
		}
		for _, b := range blocks {
			want[b] = fires
		}
	}
	if rows != 50 || firing != 32 {
		t.Fatalf("note transcribes %d blocks with %d firing, want 50 and 32", rows, firing)
	}

	got := map[string]bool{}
	for _, e := range specialPurposeTable {
		got[e.prefix.String()] = e.fires
	}
	for p, fires := range want {
		g, ok := got[p]
		if !ok {
			t.Errorf("note block %s missing from the code table", p)
			continue
		}
		if g != fires {
			t.Errorf("block %s: code fires=%v, note fires=%v", p, g, fires)
		}
	}
	for p := range got {
		if _, ok := want[p]; !ok {
			t.Errorf("code table block %s is not in the note: a selection, not a transcription", p)
		}
	}
}
