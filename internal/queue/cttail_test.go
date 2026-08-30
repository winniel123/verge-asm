package queue

import (
	"errors"
	"strings"
	"testing"
)

func TestGetEntriesURL(t *testing.T) {
	got := getEntriesURL("https://ct.example/log/", 100, 355)
	want := "https://ct.example/log/ct/v1/get-entries?start=100&end=355"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEnsureTrailingSlash(t *testing.T) {
	if got := ensureTrailingSlash("https://ct.example/log"); got != "https://ct.example/log/" {
		t.Fatalf("missing slash not added: %q", got)
	}
	if got := ensureTrailingSlash("https://ct.example/log/"); got != "https://ct.example/log/" {
		t.Fatalf("existing slash not preserved: %q", got)
	}
}

func TestCTDriftLabel(t *testing.T) {
	if got := ctDriftLabel(1); got != "1 new certificate for known names" {
		t.Fatalf("singular: %q", got)
	}
	if got := ctDriftLabel(3); got != "3 new certificates for known names" {
		t.Fatalf("plural: %q", got)
	}
}

// TestCTHTTPCause confirms a non-200 is marked safe to surface verbatim (the code and
// endpoint carry no source detail), while a transport error stays redacted to a generic
// phrase — the same discipline the crt.sh path applies (#780).
func TestCTHTTPCause(t *testing.T) {
	safe := ctHTTPCause(nil, 502, "get-sth")
	if got := redactCause(safe); !strings.Contains(got, "502") || !strings.Contains(got, "get-sth") {
		t.Fatalf("non-200 cause not surfaced verbatim: %q", got)
	}
	transport := ctHTTPCause(errors.New("dial tcp: connection refused"), 0, "get-entries")
	if got := redactCause(transport); got != "measurement failed" {
		t.Fatalf("transport error not redacted: %q", got)
	}
}
