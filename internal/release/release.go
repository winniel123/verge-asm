// Package release runs the worker's daily, opt-out, best-effort check for a
// newer Verge release and records it in instance_config (#391, ADR-0124). It
// never self-replaces; the feed defaults to a GitHub endpoint (ADR-0118).
package release

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

type Store interface {
	GetInstanceConfig(ctx context.Context) (db.GetInstanceConfigRow, error)
	SetReleaseCache(ctx context.Context, arg db.SetReleaseCacheParams) error
}

type Feed struct {
	Version string
	Notes   string
}

type Fetcher interface {
	Latest(ctx context.Context) (Feed, error)
}

type Checker struct {
	store   Store
	fetch   Fetcher
	version string
	now     func() time.Time
	log     *log.Logger
}

func NewChecker(store Store, fetch Fetcher, version string, now func() time.Time, logger *log.Logger) *Checker {
	if now == nil {
		now = time.Now
	}
	return &Checker{store: store, fetch: fetch, version: version, now: now, log: logger}
}

func (c *Checker) Check(ctx context.Context) {
	cfg, err := c.store.GetInstanceConfig(ctx)
	if err != nil {
		c.logf("release: read config: %v", err)
		return
	}
	if !cfg.UpdateCheckEnabled {
		// ADR-0124's air-gap rule: no network call while opted out, on a tick or on boot.
		return
	}

	feed, err := c.fetch.Latest(ctx)
	if err != nil {
		// A failed feed leaves the cache exactly as it was, never a guessed verdict (ADR-0124).
		c.logf("release: check upstream: %v", err)
		return
	}

	params := db.SetReleaseCacheParams{ReleaseState: pgtype.Text{String: "current", Valid: true}}
	if isNewer(feed.Version, c.version) {
		params.ReleaseState = pgtype.Text{String: "newer", Valid: true}
		params.ReleaseLatestVersion = pgtype.Text{String: feed.Version, Valid: true}
		params.ReleaseLatestNotes = pgtype.Text{String: feed.Notes, Valid: true}
	}
	if err := c.store.SetReleaseCache(ctx, params); err != nil {
		c.logf("release: write cache: %v", err)
	}
}

func (c *Checker) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	c.Check(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			c.Check(ctx)
		}
	}
}

func (c *Checker) logf(format string, args ...any) {
	if c.log != nil {
		c.log.Printf(format, args...)
	}
}

func isNewer(latest, running string) bool {
	// An unparseable version returns false, so an unstamped dev build never raises a false alarm.
	lv, ok := parseVersion(latest)
	if !ok {
		return false
	}
	rv, ok := parseVersion(running)
	if !ok {
		return false
	}
	for i := 0; i < len(lv) || i < len(rv); i++ {
		var l, r int
		if i < len(lv) {
			l = lv[i]
		}
		if i < len(rv) {
			r = rv[i]
		}
		if l != r {
			return l > r
		}
	}
	return false
}

func parseVersion(s string) ([]int, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return nil, false
	}
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if i := strings.IndexAny(p, "-+"); i >= 0 {
			p = p[:i]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}
