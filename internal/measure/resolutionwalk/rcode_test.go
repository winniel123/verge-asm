package resolutionwalk

import (
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

func TestRcodeNameMapsTheFiveDiscriminands(t *testing.T) {
	cases := map[dnsmessage.RCode]Rcode{
		dnsmessage.RCodeSuccess:       NOERROR,
		dnsmessage.RCodeNameError:     NXDOMAIN,
		dnsmessage.RCodeFormatError:   FORMERR,
		dnsmessage.RCodeRefused:       REFUSED,
		dnsmessage.RCodeServerFailure: SERVFAIL,
	}
	for rc, want := range cases {
		if got := rcodeName(rc); got != want {
			t.Errorf("rcodeName(%d) = %q, want %q", rc, got, want)
		}
	}
	// x/net names this code alone of the unmapped, so a String fallback leaks a Go identifier.
	if got := rcodeName(dnsmessage.RCodeNotImplemented); got != OTHER {
		t.Errorf("rcodeName(NOTIMP) = %q, want %q", got, OTHER)
	}
}

func TestRcodeNameFoldsEveryOtherWireCodeToOther(t *testing.T) {
	// A header rcode is four bits and no extended rcode is assembled (ADR-0143).
	mapped := map[dnsmessage.RCode]bool{
		dnsmessage.RCodeSuccess:       true,
		dnsmessage.RCodeNameError:     true,
		dnsmessage.RCodeFormatError:   true,
		dnsmessage.RCodeRefused:       true,
		dnsmessage.RCodeServerFailure: true,
	}
	folded := 0
	for i := 0; i < 16; i++ {
		rc := dnsmessage.RCode(i)
		if mapped[rc] {
			continue
		}
		if got := rcodeName(rc); got != OTHER {
			t.Errorf("rcodeName(%d) = %q, want %q: an unmapped wire code left the union", i, got, OTHER)
		}
		folded++
	}
	if folded != 11 {
		t.Fatalf("checked %d unmapped wire codes, want 11", folded)
	}
}
