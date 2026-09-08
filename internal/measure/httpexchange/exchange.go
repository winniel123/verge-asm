package httpexchange

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure"
)

type Target struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    uint16 `json:"port"`
	Scheme  string `json:"scheme"`
}

func (t Target) ServiceKey() string {
	// The same triple connect-outcome renders, so the Endpoint's Service leg names one Service.
	if addr, err := netip.ParseAddr(t.Address); err == nil {
		return netip.AddrPortFrom(addr.Unmap(), t.Port).String() + "/tcp"
	}
	return t.Address + ":" + strconv.Itoa(int(t.Port)) + "/tcp"
}

func EndpointKey(name, serviceKey string) string {
	// A leading `@` is a distinguished variant, never an empty name; no key part holds one.
	return name + "@" + serviceKey
}

func (t Target) EndpointKey() string { return EndpointKey(t.Name, t.ServiceKey()) }

type ExchangeResult struct {
	Failed          bool
	Err             string
	Status          int
	Server          string
	WWWAuthenticate string
	Location        string
	Body            []byte
}

const (
	OutcomeResponded      = "responded"
	OutcomeNoHTTPResponse = "no-http-response"
)

type HTTPIdentity struct {
	Outcome          string `json:"outcome"`
	Status           int    `json:"status,omitempty"`
	Server           string `json:"server,omitempty"`
	Title            string `json:"title,omitempty"`
	WWWAuthenticate  string `json:"www_authenticate,omitempty"`
	RedirectLocation string `json:"redirect_location,omitempty"`
}

const titleCap = 256

const MaxResponseHeaderBytes = 64 << 10

func Identity(r ExchangeResult, bodyCap int) HTTPIdentity {
	// No valid HTTP response is a determinate negative, never an absence (ADR-0011, ADR-0015).
	if r.Failed {
		return HTTPIdentity{Outcome: OutcomeNoHTTPResponse}
	}
	if bodyCap < 0 {
		bodyCap = 0
	}
	body := r.Body
	if len(body) > bodyCap {
		body = body[:bodyCap]
	}
	// The admitted set is closed; a body hash would diff the Endpoint every run (ADR-0011).
	return HTTPIdentity{
		Outcome:          OutcomeResponded,
		Status:           r.Status,
		Server:           r.Server,
		Title:            extractTitle(body),
		WWWAuthenticate:  r.WWWAuthenticate,
		RedirectLocation: r.Location,
	}
}

func extractTitle(body []byte) string {
	// A deterministic scan, never an HTML parse: the golden corpus pins this output exactly.
	s := string(body)
	open := indexASCIIFold(s, "<title")
	if open < 0 {
		return ""
	}
	gt := strings.IndexByte(s[open:], '>')
	if gt < 0 {
		return ""
	}
	start := open + gt + 1
	end := indexASCIIFold(s[start:], "</title>")
	if end < 0 {
		return ""
	}
	title := strings.Join(strings.Fields(s[start:start+end]), " ")
	if len(title) > titleCap {
		title = title[:titleCap]
	}
	return title
}

func indexASCIIFold(s, sub string) int {
	// strings.ToLower widens a non-UTF-8 byte, so its offsets do not index the raw body (#1648).
	for i := 0; i+len(sub) <= len(s); i++ {
		if strings.EqualFold(s[i:i+len(sub)], sub) {
			return i
		}
	}
	return -1
}

type Exchanger interface {
	Exchange(ctx context.Context, target Target) ExchangeResult
}

type NetExchanger struct {
	Params  Params
	realm   custody.Realm
	control func(network, address string, c syscall.RawConn) error
}

func (n NetExchanger) Exchange(ctx context.Context, target Target) ExchangeResult {
	p := n.Params
	if p.TimeoutMillis <= 0 {
		p.TimeoutMillis = DefaultParams().TimeoutMillis
	}
	if p.BodyCapBytes <= 0 {
		p.BodyCapBytes = DefaultParams().BodyCapBytes
	}
	if len(p.ALPN) == 0 {
		p.ALPN = DefaultParams().ALPN
	}
	// A hostname would be re-resolved at connect time with no rebinding backstop.
	if _, err := netip.ParseAddr(target.Address); err != nil {
		return ExchangeResult{Failed: true, Err: "httpexchange: refusing non-literal address " + target.Address}
	}
	scheme := target.Scheme
	if scheme == "" {
		scheme = "http"
	}
	url := scheme + "://" + net.JoinHostPort(target.Address, strconv.Itoa(int(target.Port))) + "/"

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutMillis)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return ExchangeResult{Failed: true, Err: err.Error()}
	}
	// An identifiable probe lets a target's operator recognise the GET / (spec §3.3).
	req.Header.Set("User-Agent", measure.ProbeUserAgent)
	// Non-nil only in a test that must reach loopback; production installs the guard.
	control := n.control
	if control == nil {
		control = custody.EgressGuard("httpexchange", n.realm)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// The rebinding-proof line: the kernel's own address is refused even when entry passed.
	transport.DialContext = (&net.Dialer{Control: control}).DialContext
	// The transport is dropped after one GET, so a pooled socket idles 90 s per target (#1661).
	transport.DisableKeepAlives = true
	// Three headers ride the observation verbatim, and the wire line caps at 1 MiB (#1662).
	transport.MaxResponseHeaderBytes = MaxResponseHeaderBytes
	// The URL names an IP, so verification fails every real certificate and reads no-HTTP (#1647).
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, // #nosec G402 (accepted: HTTP identity probe — reads the response of an untrusted listener; verifying the chain against an IP literal would drop the measurement. Not a trusted-service client call.)
		// The declared list goes on the wire, so a Go release cannot move the offer (ADR-0025).
		NextProtos: p.ALPN,
	}
	client := &http.Client{
		Transport: transport,
		// Not followed: the 3xx is returned as-is so its Location is identity (ADR-0025).
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return ExchangeResult{Failed: true, Err: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(p.BodyCapBytes)+1))
	return ExchangeResult{
		Status:          resp.StatusCode,
		Server:          resp.Header.Get("Server"),
		WWWAuthenticate: resp.Header.Get("WWW-Authenticate"),
		Location:        resp.Header.Get("Location"),
		Body:            body,
	}
}
