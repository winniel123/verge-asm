package vantage

import "testing"

func TestParseResolverAccepts(t *testing.T) {
	cases := map[string]string{
		" 9.9.9.9:53 ":     "9.9.9.9:53",
		"9.9.9.9":          "9.9.9.9",
		"dns.example.net":  "dns.example.net",
		"[2001:db8::1]:53": "[2001:db8::1]:53",
		"2001:db8::1":      "2001:db8::1",
	}
	for in, want := range cases {
		got, err := ParseResolver(in)
		if err != nil {
			t.Errorf("ParseResolver(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseResolver(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseResolverRejects(t *testing.T) {
	for _, in := range []string{
		"", "   ", "https://dns.example.net", "dns.example.net/query",
		"9.9.9.9 53", "9.9.9.9:0", "9.9.9.9:70000", "9.9.9.9:dns", ":53", "not:an:address",
	} {
		if got, err := ParseResolver(in); err == nil {
			t.Errorf("ParseResolver(%q) = %q, want an error", in, got)
		}
	}
}
