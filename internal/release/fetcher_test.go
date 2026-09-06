package release

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type countingBody struct {
	r    io.Reader
	read int
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	b.read += n
	return n, err
}

func (b *countingBody) Close() error { return nil }

type fakeDoer struct {
	body   *countingBody
	status int
	err    error

	gotReq *http.Request
	calls  int
}

func (d *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls++
	d.gotReq = req
	if d.err != nil {
		return nil, d.err
	}
	return &http.Response{StatusCode: d.status, Body: d.body, Header: make(http.Header)}, nil
}

func doerWith(status int, body string) *fakeDoer {
	return &fakeDoer{status: status, body: &countingBody{r: strings.NewReader(body)}}
}

func TestLatestDecodesAWellFormedFeed(t *testing.T) {
	d := doerWith(http.StatusOK, `{"tag_name":"v1.4.0","body":"the notes"}`)

	feed, err := NewHTTPFetcher("https://feed.example/latest", d).Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if feed.Version != "v1.4.0" {
		t.Errorf("Version = %q, want v1.4.0", feed.Version)
	}
	if feed.Notes != "the notes" {
		t.Errorf("Notes = %q, want %q", feed.Notes, "the notes")
	}
}

func TestLatestRefusesNonOKAndDoesNotDecode(t *testing.T) {
	d := doerWith(http.StatusServiceUnavailable, `{"tag_name":"v9.9.9"}`)

	_, err := NewHTTPFetcher("https://feed.example/latest", d).Latest(context.Background())
	if err == nil {
		t.Fatal("Latest returned no error on a 503")
	}
	if !strings.Contains(err.Error(), "release feed: status 503") {
		t.Errorf("err = %v, want the status refusal", err)
	}
	if d.body.read != 0 {
		t.Errorf("read %d body bytes on a non-200, want 0 (the body is never decoded)", d.body.read)
	}
}

func TestLatestCapsTheBodyAtMaxFeedBytes(t *testing.T) {
	oversized := `{"tag_name":"v1.4.0","body":"` + strings.Repeat("a", 2*maxFeedBytes) + `"}`
	d := doerWith(http.StatusOK, oversized)

	_, err := NewHTTPFetcher("https://feed.example/latest", d).Latest(context.Background())
	if err == nil {
		t.Fatal("an oversized feed decoded cleanly, so the cap did not bite")
	}
	if !strings.Contains(err.Error(), "release feed: decode") {
		t.Errorf("err = %v, want the decode wrap", err)
	}
	if d.body.read > maxFeedBytes {
		t.Errorf("read %d body bytes, want at most maxFeedBytes (%d)", d.body.read, maxFeedBytes)
	}
}

func TestLatestRefusesAnEmptyTagName(t *testing.T) {
	d := doerWith(http.StatusOK, `{"tag_name":"","body":"the notes"}`)

	_, err := NewHTTPFetcher("https://feed.example/latest", d).Latest(context.Background())
	if err == nil {
		t.Fatal("Latest accepted an empty tag_name")
	}
	if !strings.Contains(err.Error(), "empty tag_name") {
		t.Errorf("err = %v, want the empty-tag_name refusal", err)
	}
}

func TestLatestSetsTheGitHubAcceptHeader(t *testing.T) {
	d := doerWith(http.StatusOK, `{"tag_name":"v1.4.0"}`)

	if _, err := NewHTTPFetcher("https://feed.example/latest", d).Latest(context.Background()); err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got := d.gotReq.Header.Get("Accept"); got != "application/vnd.github+json" {
		t.Errorf("Accept = %q, want application/vnd.github+json", got)
	}
	if got := d.gotReq.URL.String(); got != "https://feed.example/latest" {
		t.Errorf("URL = %q, want the configured feed URL", got)
	}
}

func TestLatestPropagatesADoerError(t *testing.T) {
	sentinel := errors.New("dial tcp: i/o timeout")
	d := &fakeDoer{err: sentinel}

	_, err := NewHTTPFetcher("https://feed.example/latest", d).Latest(context.Background())
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want the Doer's own error", err)
	}
}

func TestNewHTTPFetcherFallsBackToTheProductionDoer(t *testing.T) {
	f := NewHTTPFetcher("https://feed.example/latest", nil)
	c, ok := f.client.(*http.Client)
	if !ok {
		t.Fatalf("nil doer left client %T, want *http.Client", f.client)
	}
	if c.Timeout != feedTimeout {
		t.Errorf("Timeout = %v, want feedTimeout (%v)", c.Timeout, feedTimeout)
	}
	if c.CheckRedirect == nil {
		t.Fatal("the fallback client has no CheckRedirect — redirects would be followed")
	}
}

func TestNewHTTPDoerRefusesRedirects(t *testing.T) {
	c := NewHTTPDoer()
	if c.CheckRedirect == nil {
		t.Fatal("no CheckRedirect set — redirects would be followed")
	}
	if err := c.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Errorf("CheckRedirect = %v, want ErrUseLastResponse (do not follow)", err)
	}
}

func TestLatestDoesNotFollowARedirect(t *testing.T) {
	var hopReached bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hopReached = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0","body":"pwned.example.com"}`))
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/latest/meta-data/", http.StatusFound)
	}))
	defer redirector.Close()

	feed, err := NewHTTPFetcher(redirector.URL+"/releases/latest", NewHTTPDoer()).Latest(context.Background())
	if hopReached {
		t.Fatal("the fetcher followed the redirect and reached the next hop: blind SSRF")
	}
	if err == nil {
		t.Fatalf("Latest returned %+v on a 302, want the unfollowed 3xx refused as a non-200", feed)
	}
	if !strings.Contains(err.Error(), "release feed: status 302") {
		t.Errorf("err = %v, want the 302 surfaced as an ordinary non-200", err)
	}
}
