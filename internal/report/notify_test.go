package report

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/delivery"
)

var periodStart = time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
var periodEnd = time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)

func TestReadyBodyCarriesNoEstate(t *testing.T) {
	// ADR-0039's hard guard: no signal, subject, withdrawal, census, headline or row may appear.
	raw, err := MarshalReadyBody(BuildReadyBody("Weekly exposure summary", periodStart, periodEnd, "https://verge.example"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)

	for _, forbidden := range []string{
		"signal", "subject", "withdraw", "census", "headline", "asset",
		"service", "address", "severity", "entries", "rows", "cause", "class",
	} {
		if strings.Contains(strings.ToLower(doc), forbidden) {
			t.Errorf("ready body leaked an estate term %q: %s", forbidden, doc)
		}
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"kind": true, "report": true, "period_start": true, "period_end": true, "url": true}
	if len(m) != len(want) {
		t.Errorf("body has %d keys, want %d: %s", len(m), len(want), doc)
	}
	for k := range m {
		if !want[k] {
			t.Errorf("body carries an unexpected key %q: %s", k, doc)
		}
	}

	var b ReadyBody
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	if b.Kind != "report-ready" {
		t.Errorf("kind = %q, want report-ready", b.Kind)
	}
	if b.Report != "Weekly exposure summary" {
		t.Errorf("report = %q, want the schedule name verbatim", b.Report)
	}
	if !b.PeriodStart.Equal(periodStart) || !b.PeriodEnd.Equal(periodEnd) {
		t.Errorf("period = [%s, %s], want [%s, %s]", b.PeriodStart, b.PeriodEnd, periodStart, periodEnd)
	}
}

func TestReadyBodyLink(t *testing.T) {
	cases := []struct{ base, want string }{
		{"https://verge.example", "https://verge.example/reports/delivery"},
		{"https://verge.example/", "https://verge.example/reports/delivery"},
		{"", "/reports/delivery"},
	}
	for _, c := range cases {
		got := BuildReadyBody("R", periodStart, periodEnd, c.base).URL
		if got != c.want {
			t.Errorf("url for base %q = %q, want %q", c.base, got, c.want)
		}
	}
}

func TestShouldNotifyOnlyWhenChannelBound(t *testing.T) {
	if !shouldNotify(pgtype.Int8{Int64: 7, Valid: true}) {
		t.Error("a schedule bound to a channel should enqueue a notification")
	}
	if shouldNotify(pgtype.Int8{}) {
		t.Error("a download-only schedule (NULL channel) should enqueue nothing")
	}
}

func TestNotifyPostsSignedReadyBodyDelivered(t *testing.T) {
	// An operator name may legitimately hold an estate word, so the leak check needs a clean one.
	body, err := MarshalReadyBody(BuildReadyBody("Weekly exposure summary", periodStart, periodEnd, "https://verge.example"))
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("k")
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	fake := &captureDoer{status: 200}
	res := fakeResolver{"hooks.example": {netip.MustParseAddr("93.184.216.34")}}

	status, err := delivery.SendSigned(context.Background(), fake, res, "https://hooks.example/hook", body, secret, now)
	if err != nil {
		t.Fatalf("SendSigned: %v", err)
	}
	if !delivery.Delivered(status) {
		t.Errorf("status %d should be delivered", status)
	}
	if string(fake.body) != string(body) {
		t.Errorf("posted body differs from signed body:\n got %s\nwant %s", fake.body, body)
	}
	if strings.Contains(strings.ToLower(string(fake.body)), "signal") {
		t.Errorf("posted body leaked estate: %s", fake.body)
	}
	wantSig := "sha256=" + delivery.Sign(secret, fake.body, now)
	if fake.sig != wantSig {
		t.Errorf("signature over the received body = %q, want %q", fake.sig, wantSig)
	}
}

func TestNotifyOutcomeForkRetriesThenDeadLetters(t *testing.T) {
	const max = int32(5)
	// A notification starts at attempt 0, so with max 5 the sixth failure dead-letters.
	if v := delivery.Decide(delivery.Delivered(200), 0, max); v != delivery.VerdictDelivered {
		t.Errorf("a 2xx: verdict %v, want delivered", v)
	}
	for attempt := int32(0); attempt < max; attempt++ {
		if v := delivery.Decide(delivery.Delivered(500), attempt, max); v != delivery.VerdictRetry {
			t.Errorf("attempt %d of %d failed: verdict %v, want retry", attempt, max, v)
		}
	}
	if v := delivery.Decide(delivery.Delivered(500), max, max); v != delivery.VerdictUndelivered {
		t.Errorf("the spent budget: verdict %v, want undelivered (dead-letter, receipt stays generated)", v)
	}
}

func TestNotifyRefusesPrivateResolvedTarget(t *testing.T) {
	fake := &captureDoer{status: 200}
	res := fakeResolver{"internal.example": {netip.MustParseAddr("10.0.0.5")}}
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)

	if _, err := delivery.SendSigned(context.Background(), fake, res, "https://internal.example/hook", []byte("{}"), nil, now); err == nil {
		t.Fatal("SendSigned to a private-resolving host returned nil; want refusal")
	}
	if fake.called {
		t.Fatal("the ready-message was POSTed to a private-resolving host")
	}
}

type captureDoer struct {
	status int
	body   []byte
	sig    string
	called bool
}

func (d *captureDoer) Do(req *http.Request) (*http.Response, error) {
	d.called = true
	if req.Body != nil {
		d.body, _ = io.ReadAll(req.Body)
	}
	d.sig = req.Header.Get("X-Verge-Signature")
	return &http.Response{StatusCode: d.status, Body: http.NoBody}, nil
}

type fakeResolver map[string][]netip.Addr

func (f fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	if a, ok := f[host]; ok {
		return a, nil
	}
	return nil, errors.New("no such host: " + host)
}
