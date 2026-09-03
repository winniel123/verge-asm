package delivery

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/message"
)

var causeAt = time.Date(2026, 8, 15, 3, 4, 5, 0, time.UTC)

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
	if b.Link != "https://verge.example/seeds" {
		t.Errorf("aperture link = %q, want the Seeds destination", b.Link)
	}
}

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
	if Sign(secret, []byte(`{"message":43}`), ts) == got {
		t.Error("signature did not change with the body")
	}
	// A replayed body under a fresh timestamp must not reuse the signature.
	if Sign(secret, body, ts.Add(time.Second)) == got {
		t.Error("signature did not change with the timestamp")
	}
}

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

	// With no secret the URL is the only credential, so no header may carry one instead.
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

func TestHTTPDoerRefusesRedirects(t *testing.T) {
	c := NewHTTPDoer()
	if c.CheckRedirect == nil {
		t.Fatal("no CheckRedirect set — redirects would be followed")
	}
	if err := c.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Errorf("CheckRedirect = %v, want ErrUseLastResponse (do not follow)", err)
	}
}

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
	if string(fake.body) != string(body) {
		t.Errorf("posted body differs from signed body:\n got %s\nwant %s", fake.body, body)
	}
	wantSig := sigScheme + Sign(secret, fake.body, ts)
	if fake.sig != wantSig {
		t.Errorf("signature over the received body = %q, want %q", fake.sig, wantSig)
	}
}

func TestDeliveryErrorRedactsCredentialBearingURL(t *testing.T) {
	const secretURL = "https://hooks.example.com/services/T0000/B0000/XXXXsecretpathXXXX"
	// #740: for a no-secret Channel the credential is the URL, and *url.Error embeds it verbatim.
	sendErr := &url.Error{
		Op:  "Post",
		URL: secretURL,
		Err: errors.New("dial tcp 93.184.216.34:443: connect: connection refused"),
	}

	got := deliveryError(0, sendErr)

	// This string is the exact value stored and logged, so one assertion covers both sinks.
	if strings.Contains(got, secretURL) {
		t.Fatalf("deliveryError leaked the full target URL: %q", got)
	}
	for _, secret := range []string{"XXXXsecretpathXXXX", "/services/T0000/B0000/", "hooks.example.com"} {
		if strings.Contains(got, secret) {
			t.Fatalf("deliveryError leaked URL fragment %q in %q", secret, got)
		}
	}
	if !strings.Contains(got, "Post") || !strings.Contains(got, "connection refused") {
		t.Fatalf("deliveryError dropped the useful diagnostic: %q", got)
	}

	wrapped := fmt.Errorf("parse target url: %w", &url.Error{Op: "parse", URL: secretURL, Err: errors.New("invalid control character in URL")})
	if got := deliveryError(0, wrapped); strings.Contains(got, secretURL) {
		t.Fatalf("deliveryError leaked the URL from a wrapped *url.Error: %q", got)
	}

	if got := deliveryError(0, errors.New("boom")); got != "boom" {
		t.Fatalf("deliveryError mangled a non-URL error: %q", got)
	}
	if got := deliveryError(503, nil); got != "HTTP 503" {
		t.Fatalf("deliveryError status fallback = %q, want %q", got, "HTTP 503")
	}
}

type captureDoer struct {
	body   []byte
	sig    string
	called bool
}

func (d *captureDoer) Do(req *http.Request) (*http.Response, error) {
	d.called = true
	if req.Body != nil {
		d.body, _ = io.ReadAll(req.Body)
	}
	d.sig = req.Header.Get(HeaderSignature)
	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
}

type fakeResolver map[string][]netip.Addr

func (f fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	if a, ok := f[host]; ok {
		return a, nil
	}
	return nil, errors.New("no such host: " + host)
}

func TestDeliveryRefusesPrivateResolvedTarget(t *testing.T) {
	// #325: a host resolving into private space is refused and its body is never POSTed.
	fake := &captureDoer{}
	r := &Runner{
		doer: fake,
		now:  func() time.Time { return causeAt },
		resolver: fakeResolver{
			"internal.example": {netip.MustParseAddr("10.0.0.5")},
			"hooks.example":    {netip.MustParseAddr("93.184.216.34")},
		},
	}
	body := []byte("{}")

	if _, err := r.send(context.Background(), "https://internal.example/hook", body, nil); err == nil {
		t.Fatal("send to a private-resolving host returned nil; want refusal")
	}
	if fake.called {
		t.Fatal("doer.Do was called for a private-resolving host — body was POSTed")
	}

	if _, err := r.send(context.Background(), "https://169.254.169.254/", body, nil); err == nil {
		t.Fatal("send to the metadata literal returned nil; want refusal")
	}
	if fake.called {
		t.Fatal("doer.Do was called for the metadata literal")
	}

	if _, err := r.send(context.Background(), "https://hooks.example/hook", body, nil); err != nil {
		t.Fatalf("send to a public-resolving host errored: %v", err)
	}
	if !fake.called {
		t.Fatal("doer.Do was not called for a public-resolving host")
	}
}
