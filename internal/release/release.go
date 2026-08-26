// Package release runs the worker's daily, best-effort check for a newer Verge
// release and records the result in the single instance_config row (#391,
// ADR-0124). It is the write half of the "Version & updates" card: the web
// handler (fillInstanceSection) reads the release cache this package writes.
//
// Three properties, straight out of ADR-0124, shape every path here:
//
//   - Opt-out. The check runs only while instance_config.update_check_enabled is
//     true. While it is false the worker makes NO network call ever — not on a
//     tick and not on boot. The flag is read at the top of every check, so an
//     air-gapped instance that never flips it reaches upstream zero times.
//   - Air-gap-safe / best-effort. A failed or unreachable feed is a no-op: it is
//     logged and the cache is left exactly as it was, never a crash and never a
//     guessed verdict. No path here can fail the worker loop.
//   - Never self-replace. This package only checks and records a version; the
//     image swap is a host action (the literal host steps live on the web side
//     as a release-authored constant, not here).
//
// The feed is an operator-configurable URL (VERGE_RELEASE_FEED_URL) that returns
// the latest release as JSON. It defaults to this repository's GitHub
// latest-release endpoint, reusing the release infrastructure the project already
// publishes to (release.yml / GHCR, ADR-0118) rather than standing up a new
// hosted service; a fork points the env at its own repo's endpoint, and an
// air-gapped deployment simply leaves update_check_enabled off.
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

// Store is the narrow slice of the data layer the Checker needs: the one config
// read that carries the opt-out flag, and the release-cache write. *db.Queries
// satisfies it. It exposes nothing else, so a bug in the checker can touch only
// the release cache.
type Store interface {
	GetInstanceConfig(ctx context.Context) (db.GetInstanceConfigRow, error)
	SetReleaseCache(ctx context.Context, arg db.SetReleaseCacheParams) error
}

// Feed is the latest release as reported by the upstream feed: a version string
// and its human-readable notes. Empty fields are tolerated — a feed that omits
// notes still yields a usable version verdict.
type Feed struct {
	Version string
	Notes   string
}

// Fetcher fetches the latest release from upstream. It is an interface so the
// Checker is driven by a fake in tests and never touches the live network there;
// HTTPFetcher is the production implementation.
type Fetcher interface {
	Latest(ctx context.Context) (Feed, error)
}

// Checker runs the daily release check and records its verdict.
type Checker struct {
	store   Store
	fetch   Fetcher
	version string // the running build version (buildVersion() / VERGE_VERSION)
	now     func() time.Time
	log     *log.Logger
}

// NewChecker builds a Checker. version is the running build's version string
// (the worker reads VERGE_VERSION, the same env the web footer and CT client
// read); it is compared against the feed's latest to decide current vs newer.
// now is injectable for tests.
func NewChecker(store Store, fetch Fetcher, version string, now func() time.Time, logger *log.Logger) *Checker {
	if now == nil {
		now = time.Now
	}
	return &Checker{store: store, fetch: fetch, version: version, now: now, log: logger}
}

// Check performs one best-effort release check. It returns no error: every
// failure is logged and swallowed so a transient upstream or database hiccup is
// retried on the next tick and never fails the worker loop.
//
// The opt-out flag is read first: while update_check_enabled is false the method
// returns before any network call, which is exactly what makes the check
// air-gap-safe on both the boot run and every tick. On a successful fetch it
// writes release_state=newer (with the latest version + notes) when upstream is
// ahead of the running build, else release_state=current with the latest fields
// cleared, stamping release_checked_at=now() either way. A fetch failure leaves
// the cache untouched.
func (c *Checker) Check(ctx context.Context) {
	cfg, err := c.store.GetInstanceConfig(ctx)
	if err != nil {
		c.logf("release: read config: %v", err)
		return
	}
	if !cfg.UpdateCheckEnabled {
		// Opted out — never dispatch a check, not even on boot. No network call.
		return
	}

	feed, err := c.fetch.Latest(ctx)
	if err != nil {
		// Air-gap-safe: an unreachable or failed feed reports nothing and leaves
		// the cache exactly as it was. Never a crash, never a guessed verdict.
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

// Run checks once at start and then every interval until ctx is done. It mirrors
// the other worker runners (retention.Retirer.Run): it only ever returns the
// context's error — a failed check is handled inside Check and never breaks the
// loop. The opt-out flag is honoured inside Check, so the initial run is gated
// too: a disabled instance makes no call on boot.
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

// isNewer reports whether latest is a strictly newer release than running, by a
// dotted-numeric (semver core) comparison that tolerates a leading "v" and any
// pre-release/build suffix (compared on the numeric core only). It is
// deliberately conservative: if either version is not parseable as a dotted
// numeric — the running build is an unstamped "dev", or the feed sent something
// unexpected — it returns false, so the check never raises a false "newer" alarm
// from a version it cannot understand.
func isNewer(latest, running string) bool {
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

// parseVersion splits a version string into its dotted numeric components,
// stripping a leading "v" and dropping any pre-release/build suffix on the last
// component (e.g. "v1.4.0-rc1" -> [1 4 0]). It reports ok=false when there is no
// leading numeric component at all, so a non-version string ("dev") is not
// silently treated as 0.0.0.
func parseVersion(s string) ([]int, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return nil, false
	}
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		// Cut any suffix on the component: "0-rc1" -> "0", "3+build" -> "3".
		if i := strings.IndexAny(p, "-+"); i >= 0 {
			p = p[:i]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			// A component with no numeric head ends the version; accept what we
			// parsed so far, but require at least the first (major) to be numeric.
			break
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}
