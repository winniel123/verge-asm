package seed

import "testing"

func TestNormalizeExclusionName(t *testing.T) {
	ok := map[string]string{
		"api.example.com":       "api.example.com",
		"  API.Example.COM  ":   "api.example.com",
		"api.example.com.":      "api.example.com", // trailing dot is presentation only
		"example.com":           "example.com",     // a subtree may be as shallow as the scope
		"a.b.c.d.example.co.uk": "a.b.c.d.example.co.uk",
		"xn--80ak6aa92e.com":    "xn--80ak6aa92e.com", // an A-label is a label like any other
	}
	for in, want := range ok {
		got, err := NormalizeExclusionName(in)
		if err != nil {
			t.Errorf("NormalizeExclusionName(%q) errored: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeExclusionName(%q) = %q, want %q", in, got, want)
		}
	}

	bad := []string{
		"",              // empty
		"localhost",     // a single label is not a fully-qualified name
		"*.example.com", // a wildcard denotes a set of names, not a name
		"exam ple.com",  // space
		"http://example.com/",
		"example.com/path",
		"-bad.example.com", // a label may not start with a hyphen
		"bad-.example.com", // …nor end with one
		"foo..example.com", // an empty label
	}
	for _, in := range bad {
		if got, err := NormalizeExclusionName(in); err == nil {
			t.Errorf("NormalizeExclusionName(%q) = %q, want error", in, got)
		}
	}
}

func TestNormalizeExclusionCIDR(t *testing.T) {
	ok := map[string]string{
		"203.0.113.0/24":  "203.0.113.0/24",
		"10.0.0.5/24":     "10.0.0.0/24",    // host bits are cleared
		"203.0.113.5":     "203.0.113.5/32", // a bare address carves a single host
		"2001:db8::1":     "2001:db8::1/128",
		"2001:db8::/48":   "2001:db8::/48",
		"  10.1.2.3/32  ": "10.1.2.3/32",
	}
	for in, want := range ok {
		p, err := NormalizeExclusionCIDR(in)
		if err != nil {
			t.Errorf("NormalizeExclusionCIDR(%q) errored: %v", in, err)
			continue
		}
		if p.String() != want {
			t.Errorf("NormalizeExclusionCIDR(%q) = %q, want %q", in, p.String(), want)
		}
	}

	bad := []string{"", "not-an-address", "10.0.0.0/33", "example.com"}
	for _, in := range bad {
		if p, err := NormalizeExclusionCIDR(in); err == nil {
			t.Errorf("NormalizeExclusionCIDR(%q) = %q, want error", in, p.String())
		}
	}
}
