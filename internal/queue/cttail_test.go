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

func TestDataTileURL(t *testing.T) {
	full := dataTileURL("https://mon.example/2026h2/", 1234, 256)
	if full != "https://mon.example/2026h2/tile/data/x001/234" {
		t.Fatalf("full tile URL: %q", full)
	}
	partial := dataTileURL("https://mon.example/2026h2/", 1234, 100)
	if partial != "https://mon.example/2026h2/tile/data/x001/234.p/100" {
		t.Fatalf("partial tile URL: %q", partial)
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
