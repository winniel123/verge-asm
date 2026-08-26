package release

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultFeedURL is the release feed the worker checks when VERGE_RELEASE_FEED_URL
// is unset: this repository's GitHub latest-release endpoint. It returns the
// latest published release as JSON carrying tag_name (the version) and body (the
// notes) — the shape HTTPFetcher parses. This reuses the release infrastructure
// the project already publishes to (release.yml / GHCR on a semver tag, ADR-0118)
// rather than standing up a new hosted feed; a fork points the env at its own
// repo's endpoint (or any URL returning the same JSON), and an air-gapped
// deployment leaves update_check_enabled off so the URL is never fetched.
const DefaultFeedURL = "https://api.github.com/repos/winniel123/verge-asm/releases/latest"

// feedTimeout bounds a single check: a short deadline, no retry — a slow or
// unreachable feed fails fast and the tick is a logged no-op (ADR-0124: no retry
// storm, best-effort).
const feedTimeout = 10 * time.Second

// maxFeedBytes caps the response body read so a hostile or misconfigured feed
// cannot exhaust worker memory; a real latest-release payload is well under this.
const maxFeedBytes = 1 << 20 // 1 MiB

// Doer is the outbound HTTP surface, behind an interface so HTTPFetcher is driven
// by a fake in tests and never touches the live network there. *http.Client
// satisfies it.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPFetcher fetches the latest release from a JSON feed URL. It parses the
// GitHub latest-release shape (tag_name + body); a custom feed mirrors those two
// fields.
type HTTPFetcher struct {
	url    string
	client Doer
}

// NewHTTPFetcher builds the production fetcher over url with a short-timeout HTTP
// client (no retry). Pass DefaultFeedURL when the operator has set no override.
func NewHTTPFetcher(url string) *HTTPFetcher {
	return &HTTPFetcher{
		url:    url,
		client: &http.Client{Timeout: feedTimeout},
	}
}

// feedPayload is the subset of a GitHub release object the check needs: the tag
// is the version, the body is the notes.
type feedPayload struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

// Latest fetches and decodes the latest release. Any transport, status, or
// decode failure is returned as an error for the caller to log and swallow — the
// caller treats a returned error as "reached nothing", leaving the cache as-is.
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
