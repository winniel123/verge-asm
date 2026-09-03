package httpexchange

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

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
	b.FollowRedirects = true // no operator can set this; only a code change (ADR-0025)
	if a.Digest() == b.Digest() {
		t.Error("changing a declared parameter must move the params digest")
	}
	if a.Digest() != DefaultParams().Digest() {
		t.Error("the digest must be stable for the same params")
	}
}

func TestIdentityFoldsACompletedExchange(t *testing.T) {
	id := Identity(ExchangeResult{
		Status: 200, Server: "nginx",
		Body: []byte("<html><head><title>Home Page</title></head></html>"),
	}, DefaultParams().BodyCapBytes)
	if id.Outcome != OutcomeResponded {
		t.Errorf("a completed exchange folds to responded, got %q", id.Outcome)
	}
	if id.Status != 200 || id.Server != "nginx" {
		t.Errorf("identity did not carry status/headers: %+v", id)
	}
	if id.Title != "Home Page" {
		t.Errorf("title not lifted from the body: %q", id.Title)
	}
}

func TestIdentityFoldsANoHTTPResponse(t *testing.T) {
	// The negative is a value, never nothing, and its Endpoint still exists (ADR-0011).
	id := Identity(ExchangeResult{Failed: true, Err: "connection reset"}, 64)
	if id.Outcome != OutcomeNoHTTPResponse {
		t.Errorf("a non-HTTP exchange folds to no-http-response, got %q", id.Outcome)
	}
	if id.Status != 0 || id.Server != "" || id.Title != "" {
		t.Errorf("a no-http-response value carries only its outcome: %+v", id)
	}
}

func TestIdentityRecordsAdmittedFieldsOnly(t *testing.T) {
	id := Identity(ExchangeResult{
		Status: 401, Server: "caddy", WWWAuthenticate: `Basic realm="x"`,
		Body: []byte("<title>Sign in</title>"),
	}, DefaultParams().BodyCapBytes)
	if id.WWWAuthenticate != `Basic realm="x"` || id.Title != "Sign in" {
		t.Errorf("admitted fields not recorded: %+v", id)
	}
	b, _ := json.Marshal(id)
	for _, refused := range []string{"body_sha256", "body_bytes", "body_truncated", "content_type"} {
		if strings.Contains(string(b), refused) {
			t.Errorf("refused field %q present in the value: %s", refused, b)
		}
	}
}

func TestTitleExtraction(t *testing.T) {
	cases := []struct{ body, want string }{
		{"<title>Hello</title>", "Hello"},
		{"<TITLE>Caps</TITLE>", "Caps"},
		{`<title lang="en">Attr</title>`, "Attr"},
		{"<title>  spaced\n  out </title>", "spaced out"},
		{"no title here", ""},
		{"<title>unterminated", ""},
	}
	for _, c := range cases {
		if got := extractTitle([]byte(c.body)); got != c.want {
			t.Errorf("extractTitle(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

func TestRedirectIsRecordedButNotFollowed(t *testing.T) {
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

func TestEndpointCreatedPerReachedExchangeNamedAndNameless(t *testing.T) {
	named := Target{Name: "api.example.com", Address: "198.51.100.1", Port: 443, Scheme: "https"}
	nameless := Target{Name: "", Address: "198.51.100.2", Port: 443, Scheme: "https"}
	noHTTP := Target{Name: "down.example.com", Address: "198.51.100.3", Port: 443, Scheme: "https"}
	ex := newCounting(map[string]ExchangeResult{
		named.EndpointKey():    {Status: 200, Server: "nginx", Body: []byte("<title>ok</title>")},
		nameless.EndpointKey(): {Status: 204, Body: nil},
		noHTTP.EndpointKey():   {Failed: true, Err: "timeout"},
	})
	var buf bytes.Buffer
	if err := RunWithExchanger(context.Background(), ex, "b1", Scope{
		Vantage: "v1", Targets: []Target{named, nameless, noHTTP}, Params: DefaultParams(),
	}, &buf); err != nil {
		t.Fatal(err)
	}
	obs := decodeAll(t, buf.Bytes())
	if len(obs) != 3 {
		t.Fatalf("an Endpoint per REACHED exchange: got %d observations, want 3 (no-http-response included)", len(obs))
	}
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
	if !keys["down.example.com@198.51.100.3:443/tcp"] {
		t.Error("a reached non-HTTP Service still creates an Endpoint (no-http-response)")
	}
}

func TestEndpointKeyRendering(t *testing.T) {
	if got := EndpointKey("host.example.com", "198.51.100.1:443/tcp"); got != "host.example.com@198.51.100.1:443/tcp" {
		t.Errorf("named key = %q", got)
	}
	if got := EndpointKey("", "198.51.100.1:443/tcp"); got != "@198.51.100.1:443/tcp" {
		t.Errorf("nameless key = %q, want @service", got)
	}
	v6 := Target{Name: "", Address: "2001:db8::1", Port: 443}
	if got := v6.ServiceKey(); got != "[2001:db8::1]:443/tcp" {
		t.Errorf("ipv6 service key = %q", got)
	}
}

func TestPacerSpacesPerHostRequests(t *testing.T) {
	p := NewPacer(Params{PerHostReqPerSec: 10})
	base := time.Unix(0, 0)
	first := p.Next("198.51.100.1", base)
	second := p.Next("198.51.100.1", base)
	if got := second.Sub(first); got != 100*time.Millisecond {
		t.Errorf("per-host spacing = %v, want 100ms at 10 req/s", got)
	}
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
