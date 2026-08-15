package delivery

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/message"
)

var causeAt = time.Date(2026, 8, 15, 3, 4, 5, 0, time.UTC)

// a flagship-shaped firing: a headline with a count and a census whose entries
// (the facets that opened) must NEVER cross the wire.
func flagshipFiring() Firing {
	census := message.NewCensus(
		message.CensusEntry{Kind: "facet", Key: "reachability"},
		message.CensusEntry{Kind: "facet", Key: "certificate"},
		message.CensusEntry{Kind: "facet", Key: "http-identity"},
	)
	b, _ := census.Marshal()
	return Firing{
		ID:          42,
		Cause:       message.CauseDrift,
		Class:       message.ClassDrift,
		SubjectKind: "service",
		FiredAt:     "203.0.113.7:443/tcp",
		Instant:     causeAt,
		Census:      b,
		Headline:    "203.0.113.7:443/tcp reached from the internet · 3 facets opened beneath it",
	}
}

// AC: the POST body is byte-identical in content to what the in-app Message
// renders — the headline verbatim — and the census is a COUNT, never rows.
func TestBuildBodyCarriesHeadlineVerbatimAndCensusCount(t *testing.T) {
	f := flagshipFiring()
	b := BuildBody(f, "https://verge.example")

	if b.Headline != f.Headline {
		t.Errorf("headline not byte-identical:\n got %q\nwant %q", b.Headline, f.Headline)
	}
	if b.Message != f.ID {
		t.Errorf("message id = %d, want %d (the dedup key)", b.Message, f.ID)
	}
	if b.Census == nil || *b.Census != 3 {
		t.Errorf("census = %v, want a count of 3", b.Census)
	}
	if b.Subject.Key != f.FiredAt || b.Subject.Kind != "service" {
		t.Errorf("subject = %+v, want the fired-at key verbatim", b.Subject)
	}
	if !b.Instant.Equal(causeAt) {
		t.Errorf("instant = %s, want the cause instant %s", b.Instant, causeAt)
	}
	if b.Link != "https://verge.example/subjects/service?key=203.0.113.7%3A443%2Ftcp" {
		t.Errorf("link = %q", b.Link)
	}
}

// AC: no row-level data (service lists, address sets) appears in the body — only
// what the in-app message already carries. The census entry KEYS must not appear
// anywhere in the marshalled document.
func TestBuildBodyCarriesNoRows(t *testing.T) {
	raw, err := MarshalBody(BuildBody(flagshipFiring(), "https://verge.example"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	for _, entryKey := range []string{"reachability", "certificate", "http-identity", "entries"} {
		if strings.Contains(doc, entryKey) {
			t.Errorf("body leaked a census row (%q): %s", entryKey, doc)
		}
	}
	// The census field is present and numeric — a count, not an array.
	var probe struct {
		Census json.RawMessage `json:"census"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatal(err)
	}
	if _, err := strconv.Atoi(strings.TrimSpace(string(probe.Census))); err != nil {
		t.Errorf("census is not a bare count: %s", probe.Census)
	}
}

// A narrowing carries only a count in its headline and no census payload (NULL);
// the body omits the census field entirely rather than sending a zero or a list.
func TestBuildBodyOmitsCensusWhenFiringHasNone(t *testing.T) {
	f := Firing{
		ID: 7, Cause: message.CauseAperture, Class: message.ClassCoverage,
		SubjectKind: "seed", FiredAt: "10.0.0.0/8", Instant: causeAt,
		Headline: "10.0.0.0/8 narrowed · 10.0.0.0/24 excluded · 5 subjects withdrawn · 9 timelines taken out of the estate",
	}
	b := BuildBody(f, "https://verge.example")
	if b.Census != nil {
		t.Errorf("census = %v, want nil for a firing with no census", b.Census)
	}
	raw, _ := MarshalBody(b)
	if strings.Contains(string(raw), "census") {
		t.Errorf("census field present for a censusless firing: %s", raw)
	}
	// An aperture firing links to the Seed whose scope moved, never a subject page.
	if b.Link != "https://verge.example/seeds" {
		t.Errorf("aperture link = %q, want the Seeds destination", b.Link)
	}
}

// AC: HMAC-SHA256 over body+timestamp when a secret is configured.
func TestSignIsHMACSHA256OverTimestampAndBody(t *testing.T) {
	secret := []byte("s3cr3t")
	body := []byte(`{"message":42}`)
	ts := causeAt

	got := Sign(secret, body, ts)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(strconv.FormatInt(ts.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Errorf("signature = %s, want %s", got, want)
	}
	// The signature covers the body: a changed byte changes the signature.
	if Sign(secret, []byte(`{"message":43}`), ts) == got {
		t.Error("signature did not change with the body")
	}
	// ...and the timestamp: a replayed body under a new timestamp signs differently.
	if Sign(secret, body, ts.Add(time.Second)) == got {
		t.Error("signature did not change with the timestamp")
	}
}

// AC: a signed request when a secret is set; no signature otherwise; never a
// bearer header, in either case.
func TestNewRequestSignsWithSecretAndNeverCarriesBearer(t *testing.T) {
	body := []byte(`{"message":42}`)
	ts := causeAt
	ctx := context.Background()

	signed, err := NewRequest(ctx, "https://verge.example/hook", body, []byte("k"), ts)
	if err != nil {
		t.Fatal(err)
	}
	if got := signed.Header.Get(HeaderSignature); got != sigScheme+Sign([]byte("k"), body, ts) {
		t.Errorf("signature header = %q", got)
	}
	if got := signed.Header.Get(HeaderTimestamp); got != strconv.FormatInt(ts.Unix(), 10) {
		t.Errorf("timestamp header = %q", got)
	}
	if signed.Header.Get("Authorization") != "" {
		t.Error("a signed request carried an Authorization header — no bearer, ever")
	}
	if signed.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", signed.Method)
	}

	// No secret: the URL is the only credential — no signature header, still no
	// bearer, and the timestamp is still present.
	unsigned, err := NewRequest(ctx, "https://verge.example/hook", body, nil, ts)
	if err != nil {
		t.Fatal(err)
	}
	if unsigned.Header.Get(HeaderSignature) != "" {
		t.Error("an unsigned channel carried a signature header")
	}
	if unsigned.Header.Get("Authorization") != "" {
		t.Error("an unsigned request carried an Authorization header — no bearer, ever")
	}
	if unsigned.Header.Get(HeaderTimestamp) == "" {
		t.Error("the timestamp header is always present")
	}
}

// AC: routing is by class alone — a channel receives a firing exactly when it
// carries that firing's class, and nothing finer.
func TestRoutesByClassAlone(t *testing.T) {
	cases := []struct {
		drift, coverage, clock bool
		class                  message.Class
		want                   bool
	}{
		{true, false, false, message.ClassDrift, true},
		{false, true, false, message.ClassDrift, false},
		{false, true, false, message.ClassCoverage, true},
		{false, false, true, message.ClassClock, true},
		{true, true, false, message.ClassClock, false},
		{true, true, true, message.Class("nonsense"), false},
	}
	for _, c := range cases {
		if got := Routes(c.drift, c.coverage, c.clock, c.class); got != c.want {
			t.Errorf("Routes(d=%v,c=%v,k=%v, %q) = %v, want %v",
				c.drift, c.coverage, c.clock, c.class, got, c.want)
		}
	}
}

// AC: only a 2xx is a delivery; a 3xx, 4xx or 5xx is a failure.
func TestDeliveredOnlyOn2xx(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299} {
		if !Delivered(code) {
			t.Errorf("%d should be delivered", code)
		}
	}
	for _, code := range []int{100, 301, 302, 400, 404, 429, 500, 503} {
		if Delivered(code) {
			t.Errorf("%d should be a failure", code)
		}
	}
}

// AC: retry uses the queue's budget — five attempts, then dead-lettered. Decide
// is the pure fork the runner records, and it reuses queue.Backoff's schedule.
func TestDecideRetriesFiveTimesThenDeadLetters(t *testing.T) {
	if Decide(true, 1, 5) != VerdictDelivered {
		t.Error("a 2xx should be delivered regardless of attempt")
	}
	for attempt := int32(1); attempt < 5; attempt++ {
		if got := Decide(false, attempt, 5); got != VerdictRetry {
			t.Errorf("attempt %d of 5 failed: verdict %v, want retry", attempt, got)
		}
	}
	if got := Decide(false, 5, 5); got != VerdictUndelivered {
		t.Errorf("the fifth failed attempt: verdict %v, want undelivered (dead-letter)", got)
	}
}

// The production doer refuses redirects: a 3xx is returned, never followed, so it
// classifies as a failure rather than chasing an undeclared host.
func TestHTTPDoerRefusesRedirects(t *testing.T) {
	c := NewHTTPDoer()
	if c.CheckRedirect == nil {
		t.Fatal("no CheckRedirect set — redirects would be followed")
	}
	if err := c.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Errorf("CheckRedirect = %v, want ErrUseLastResponse (do not follow)", err)
	}
}

// End-to-end through the request path with a fake doer: the exact body posted is
// the body that was signed, and no live network is touched.
func TestPostedBodyIsTheSignedBody(t *testing.T) {
	f := flagshipFiring()
	body, err := MarshalBody(BuildBody(f, "https://verge.example"))
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("k")
	ts := causeAt
	req, err := NewRequest(context.Background(), "https://verge.example/hook", body, secret, ts)
	if err != nil {
		t.Fatal(err)
	}

	fake := &captureDoer{}
	if _, err := fake.Do(req); err != nil {
		t.Fatal(err)
	}
	// The doer received exactly the bytes we signed.
	if string(fake.body) != string(body) {
		t.Errorf("posted body differs from signed body:\n got %s\nwant %s", fake.body, body)
	}
	wantSig := sigScheme + Sign(secret, fake.body, ts)
	if fake.sig != wantSig {
		t.Errorf("signature over the received body = %q, want %q", fake.sig, wantSig)
	}
}

// captureDoer is the fake HTTP surface: it reads the request body and headers and
// returns a 200 without ever touching the network.
type captureDoer struct {
	body []byte
	sig  string
}

func (d *captureDoer) Do(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		d.body, _ = io.ReadAll(req.Body)
	}
	d.sig = req.Header.Get(HeaderSignature)
	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
}
