package httpexchange

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

// countingExchanger records how many times Exchange is called per target, so a
// test can assert that a redirect is never followed with a second request. Each
// target answers from a fixed result.
type countingExchanger struct {
	byKey map[string]ExchangeResult
	calls map[string]int
}

func newCounting(rules map[string]ExchangeResult) *countingExchanger {
	return &countingExchanger{byKey: rules, calls: map[string]int{}}
}

func (c *countingExchanger) Exchange(_ context.Context, t Target) ExchangeResult {
	c.calls[t.EndpointKey()]++
	return c.byKey[t.EndpointKey()]
}

func TestDefaultParamsAreTheSafetyTable(t *testing.T) {
	p := DefaultParams()
	if p.Method != "GET" || p.Path != "/" {
		t.Errorf("http-exchange is GET / only, got %s %s", p.Method, p.Path)
	}
	if p.BodyCapBytes != 64*1024 {
		t.Errorf("body cap = %d, want 65536 (64 KB)", p.BodyCapBytes)
	}
	if p.TimeoutMillis != 10000 {
		t.Errorf("timeout = %d ms, want 10000 (10 s)", p.TimeoutMillis)
	}
	if p.PerHostReqPerSec != 10 {
		t.Errorf("per-host rate = %d, want 10 req/s", p.PerHostReqPerSec)
	}
	if p.FollowRedirects {
		t.Error("redirects must NOT be followed by default — the declared invariant")
	}
}

func TestParamsDigestMovesWithADeclaredParameter(t *testing.T) {
	a := DefaultParams()
	b := DefaultParams()
	b.FollowRedirects = true // an operator could never do this; a code change would
	if a.Digest() == b.Digest() {
		t.Error("changing a declared parameter must move the params digest")
	}
	if a.Digest() != DefaultParams().Digest() {
		t.Error("the digest must be stable for the same params")
	}
}

func TestIdentityFoldsACompletedExchange(t *testing.T) {
	id, ok := Identity(ExchangeResult{
		Status: 200, Server: "nginx", ContentType: "text/html", Body: []byte("hello"),
	}, DefaultParams().BodyCapBytes)
	if !ok {
		t.Fatal("a completed exchange must fold to an identity")
	}
	if id.Status != 200 || id.Server != "nginx" || id.ContentType != "text/html" {
		t.Errorf("identity did not carry status/headers: %+v", id)
	}
	if id.BodyBytes != 5 || id.BodyTruncated {
		t.Errorf("a short body is not truncated: %+v", id)
	}
	if !strings.HasPrefix(id.BodySHA256, "sha256:") {
		t.Errorf("body digest not rendered: %q", id.BodySHA256)
	}
}

func TestIdentityFailsForATransportFailure(t *testing.T) {
	if _, ok := Identity(ExchangeResult{Failed: true, Err: "connection reset"}, 64); ok {
		t.Error("a failed exchange must fold to no identity — no Endpoint is created")
	}
}

func TestBodyIsCappedAndMarkedTruncated(t *testing.T) {
	cap := 64
	body := bytes.Repeat([]byte("x"), cap+100)
	id, ok := Identity(ExchangeResult{Status: 200, Body: body}, cap)
	if !ok {
		t.Fatal("completed")
	}
	if id.BodyBytes != cap || !id.BodyTruncated {
		t.Errorf("over-cap body must truncate to the cap and mark truncated: bytes=%d trunc=%v", id.BodyBytes, id.BodyTruncated)
	}
	// The digest is over the CAPPED body, so a longer body with the same first cap
	// bytes yields the same digest — the value is bounded, not the response.
	id2, _ := Identity(ExchangeResult{Status: 200, Body: bytes.Repeat([]byte("x"), cap+9999)}, cap)
	if id.BodySHA256 != id2.BodySHA256 {
		t.Error("the digest must be over the capped body, independent of overrun length")
	}
}

func TestRedirectIsRecordedButNotFollowed(t *testing.T) {
	// A 3xx is a completed exchange: the Endpoint exists and the Location is
	// recorded as identity. The leaf issues exactly one request — never the
	// redirect's next hop.
	tgt := Target{Name: "www.example.com", Address: "198.51.100.7", Port: 80, Scheme: "http"}
	ex := newCounting(map[string]ExchangeResult{
		tgt.EndpointKey(): {Status: 301, Location: "https://www.example.com/", Server: "nginx"},
	})
	var buf bytes.Buffer
	if err := RunWithExchanger(context.Background(), ex, "b1", Scope{
		Vantage: "v1", Targets: []Target{tgt}, Params: DefaultParams(),
	}, &buf); err != nil {
		t.Fatal(err)
	}
	if ex.calls[tgt.EndpointKey()] != 1 {
		t.Errorf("redirect followed: %d requests issued, want exactly 1", ex.calls[tgt.EndpointKey()])
	}
	line := buf.String()
	if !strings.Contains(line, `"redirect_location":"https://www.example.com/"`) {
		t.Errorf("3xx Location not recorded as identity: %s", line)
	}
	if !strings.Contains(line, `"http-identity"`) || !strings.Contains(line, `"subject":"www.example.com@198.51.100.7:80/tcp"`) {
		t.Errorf("Endpoint observation not emitted for the 3xx: %s", line)
	}
}

func TestEndpointCreatedPerSuccessfulExchangeNamedAndNameless(t *testing.T) {
	named := Target{Name: "api.example.com", Address: "198.51.100.1", Port: 443, Scheme: "https"}
	nameless := Target{Name: "", Address: "198.51.100.2", Port: 443, Scheme: "https"}
	failed := Target{Name: "down.example.com", Address: "198.51.100.3", Port: 443, Scheme: "https"}
	ex := newCounting(map[string]ExchangeResult{
		named.EndpointKey():    {Status: 200, Server: "nginx", Body: []byte("ok")},
		nameless.EndpointKey(): {Status: 204, Body: nil},
		failed.EndpointKey():   {Failed: true, Err: "timeout"},
	})
	var buf bytes.Buffer
	if err := RunWithExchanger(context.Background(), ex, "b1", Scope{
		Vantage: "v1", Targets: []Target{named, nameless, failed}, Params: DefaultParams(),
	}, &buf); err != nil {
		t.Fatal(err)
	}
	obs := decodeAll(t, buf.Bytes())
	if len(obs) != 2 {
		t.Fatalf("an Endpoint per SUCCESSFUL exchange: got %d observations, want 2 (failed omitted)", len(obs))
	}
	// The named endpoint key is name@service; the nameless is @service — a
	// distinguished variant, never an empty name.
	keys := map[string]bool{}
	for _, o := range obs {
		if o.Facet != FacetHTTPIdentity {
			t.Errorf("facet = %q, want http-identity", o.Facet)
		}
		keys[o.Subject] = true
	}
	if !keys["api.example.com@198.51.100.1:443/tcp"] {
		t.Error("named endpoint key missing")
	}
	if !keys["@198.51.100.2:443/tcp"] {
		t.Error("nameless endpoint key (@service) missing")
	}
	if keys["down.example.com@198.51.100.3:443/tcp"] {
		t.Error("failed exchange must not create an Endpoint")
	}
}

func TestEndpointKeyRendering(t *testing.T) {
	if got := EndpointKey("host.example.com", "198.51.100.1:443/tcp"); got != "host.example.com@198.51.100.1:443/tcp" {
		t.Errorf("named key = %q", got)
	}
	if got := EndpointKey("", "198.51.100.1:443/tcp"); got != "@198.51.100.1:443/tcp" {
		t.Errorf("nameless key = %q, want @service", got)
	}
	// IPv6 Service keys bracket the address; the Endpoint splits at the first @.
	v6 := Target{Name: "", Address: "2001:db8::1", Port: 443}
	if got := v6.ServiceKey(); got != "[2001:db8::1]:443/tcp" {
		t.Errorf("ipv6 service key = %q", got)
	}
}

func TestPacerSpacesPerHostRequests(t *testing.T) {
	p := NewPacer(Params{PerHostReqPerSec: 10}) // 100ms spacing
	base := time.Unix(0, 0)
	first := p.Next("198.51.100.1", base)
	second := p.Next("198.51.100.1", base) // same instant: must be spaced out
	if got := second.Sub(first); got != 100*time.Millisecond {
		t.Errorf("per-host spacing = %v, want 100ms at 10 req/s", got)
	}
	// A different host is not held behind the first host's schedule.
	other := p.Next("198.51.100.2", base)
	if !other.Equal(base) {
		t.Errorf("a fresh host should start at now, got %v", other)
	}
}

func decodeAll(t *testing.T, b []byte) []wire.Observation {
	t.Helper()
	sc := wire.NewObservationScanner(bytes.NewReader(b))
	var out []wire.Observation
	for sc.Next() {
		out = append(out, sc.Observation())
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
