package message

import (
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/signal"
)

func TestClockEdgeStatesThatNoMeasurementMoved(t *testing.T) {
	const ep = "admin.example.com@198.51.100.7:443/tcp"
	nb := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	na := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	at := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		rule string
		want string
	}{
		{"certificate-expiring", ep + " · no measurement moved · certificate-expiring now fired · not_after 2026-09-29 is inside the 30-day horizon"},
		{"certificate-expired", ep + " · no measurement moved · certificate-expired now fired · not_after 2026-09-29 has passed"},
		{"certificate-not-yet-valid", ep + " · no measurement moved · certificate-not-yet-valid now fired · not_before 2026-07-01 has not arrived"},
	}
	for _, c := range cases {
		m := ClockEdge("endpoint", ep, c.rule, signal.CertClock{NotBefore: nb, NotAfter: na}, at)
		if m.Cause != CauseThreshold || m.Class != ClassClock {
			t.Errorf("%s: a clock crossing is the threshold cause, clock class; got %s %s", c.rule, m.Cause, m.Class)
		}
		if m.SubjectKind != "endpoint" || m.FiredAt != ep || !m.Instant.Equal(at) {
			t.Errorf("%s: fires at the endpoint whose span the rule read; got %s %s %v", c.rule, m.SubjectKind, m.FiredAt, m.Instant)
		}
		if m.Headline != c.want {
			t.Errorf("%s headline\n got %q\nwant %q", c.rule, m.Headline, c.want)
		}
		if ContainsValence(m.Headline) {
			t.Errorf("%s: headline carries a valence word: %q", c.rule, m.Headline)
		}
		if m.Census != nil || m.LinkKind() != LinkObject {
			t.Errorf("%s: no census, and the link is the subject's own page", c.rule)
		}
	}
}

func TestClockEdgeHorizonReadsInHoursBelowAWholeDay(t *testing.T) {
	nb := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	na := nb.Add(7 * 24 * time.Hour)
	m := ClockEdge("endpoint", "x", "certificate-expiring", signal.CertClock{NotBefore: nb, NotAfter: na}, na)
	if !strings.Contains(m.Headline, "inside the 84-hour horizon") {
		t.Errorf("a seven-day certificate halves to 84 hours: %q", m.Headline)
	}
}
