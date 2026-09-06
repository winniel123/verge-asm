package release

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const DefaultFeedURL = "https://api.github.com/repos/winniel123/verge-asm/releases/latest"

const feedTimeout = 10 * time.Second // no retry: a slow feed leaves the tick a no-op (ADR-0124)

const maxFeedBytes = 1 << 20 // a hostile or misconfigured feed cannot exhaust worker memory

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPFetcher struct {
	url    string
	client Doer
}

func NewHTTPDoer() *http.Client {
	return &http.Client{
		Timeout: feedTimeout,
		// A followed 3xx lets the feed's host pick our next destination, e.g. IMDS (ADR-0196 §1).
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func NewHTTPFetcher(url string, doer Doer) *HTTPFetcher {
	if doer == nil {
		doer = NewHTTPDoer()
	}
	return &HTTPFetcher{url: url, client: doer}
}

type feedPayload struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

func (f *HTTPFetcher) Latest(ctx context.Context) (Feed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return Feed{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := f.client.Do(req)
	if err != nil {
		return Feed{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return Feed{}, fmt.Errorf("release feed: status %d", resp.StatusCode)
	}

	var p feedPayload
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxFeedBytes)).Decode(&p); err != nil {
		return Feed{}, fmt.Errorf("release feed: decode: %w", err)
	}
	if p.TagName == "" {
		return Feed{}, fmt.Errorf("release feed: empty tag_name")
	}
	return Feed{Version: p.TagName, Notes: p.Body}, nil
}
