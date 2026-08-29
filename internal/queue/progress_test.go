package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// redactCause surfaces a cause verbatim ONLY when it was explicitly marked safe
// (safeProgress) — the CT non-200 the ticket cites. Every other cause, notably a
// prober exec error carrying stderr, is redacted to a generic phrase so no raw
// prober output ever reaches the live stream (ADR-0041 / instance-privacy).
func TestRedactCause(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"safe ct non-200", safeProgress("crt.sh returned HTTP 502"), "crt.sh returned HTTP 502"},
		{"safe wrapped", fmt.Errorf("ct: %w", safeProgress("crt.sh returned HTTP 429")), "crt.sh returned HTTP 429"},
		{
			"prober stderr is redacted",
			fmt.Errorf("queue: exec prober: exit status 1 (stderr: panic: secret internal detail)"),
			"measurement failed",
		},
		{"raw net error is redacted", errors.New("dial tcp 10.0.0.1:443: connect: connection refused"), "measurement failed"},
		{"nil cause", nil, "measurement failed"},
	}
	for _, c := range cases {
		if got := redactCause(c.err); got != c.want {
			t.Errorf("%s: redactCause=%q, want %q", c.name, got, c.want)
		}
	}
}

// The wire payload round-trips through JSON with the tags the web consumer decodes
// (dispatch, job, level, text) — the producer and consumer share this shape by
// convention, not a shared type (they live in different packages).
func TestJobProgressWire(t *testing.T) {
	ev := jobProgress{Dispatch: 42, Job: 701, Level: "warn", Text: "attempt 4 failed · crt.sh returned HTTP 502 · retrying"}
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	// The tags are the contract with cmd/web's decoder; assert them literally.
	for _, want := range []string{`"dispatch":42`, `"job":701`, `"level":"warn"`, `"text":"attempt 4 failed`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("payload %s missing %q", raw, want)
		}
	}
	var back jobProgress
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back != ev {
		t.Errorf("round-trip mismatch: got %+v, want %+v", back, ev)
	}
}

// The label builders produce the exact redacted text each event carries. The retry label is
// the crt.sh-502 example the ticket cites; the completion label is a bare count.
func TestProgressLabels(t *testing.T) {
	if got := retryLabel(4, safeProgress("crt.sh returned HTTP 502")); got != "attempt 4 failed · crt.sh returned HTTP 502 · retrying" {
		t.Errorf("retryLabel=%q", got)
	}
	if got := retryLabel(2, errors.New("queue: exec prober: exit status 1 (stderr: secret)")); got != "attempt 2 failed · measurement failed · retrying" {
		t.Errorf("retryLabel should redact a prober cause: %q", got)
	}
	if got := deadLetterLabel(5, safeProgress("crt.sh returned HTTP 502")); got != "dead-lettered after 5 attempts · crt.sh returned HTTP 502" {
		t.Errorf("deadLetterLabel=%q", got)
	}
	if got := countLabel(1, "name admitted", "names admitted"); got != "1 name admitted" {
		t.Errorf("countLabel singular=%q", got)
	}
	if got := countLabel(12, "observation", "observations"); got != "12 observations" {
		t.Errorf("countLabel plural=%q", got)
	}
}
