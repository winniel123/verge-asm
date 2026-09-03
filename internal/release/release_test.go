package release

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

type fakeStore struct {
	cfg       db.GetInstanceConfigRow
	cfgErr    error
	setCalled bool
	set       db.SetReleaseCacheParams
	setErr    error
}

func (s *fakeStore) GetInstanceConfig(context.Context) (db.GetInstanceConfigRow, error) {
	return s.cfg, s.cfgErr
}

func (s *fakeStore) SetReleaseCache(_ context.Context, arg db.SetReleaseCacheParams) error {
	s.setCalled = true
	s.set = arg
	return s.setErr
}

type fakeFetcher struct {
	feed  Feed
	err   error
	calls int
}

func (f *fakeFetcher) Latest(context.Context) (Feed, error) {
	f.calls++
	return f.feed, f.err
}

func enabledCfg(enabled bool) db.GetInstanceConfigRow {
	return db.GetInstanceConfigRow{UpdateCheckEnabled: enabled}
}

func TestCheck_Disabled_NeverDispatches(t *testing.T) {
	store := &fakeStore{cfg: enabledCfg(false)}
	fetch := &fakeFetcher{feed: Feed{Version: "v9.9.9"}}
	c := NewChecker(store, fetch, "v1.0.0", nil, nil)

	c.Check(context.Background())

	if fetch.calls != 0 {
		t.Fatalf("fetcher called %d times while disabled; want 0 (no network call ever)", fetch.calls)
	}
	if store.setCalled {
		t.Fatal("release cache written while disabled; want no write")
	}
}

func TestCheck_UpstreamAhead_WritesNewer(t *testing.T) {
	store := &fakeStore{cfg: enabledCfg(true)}
	fetch := &fakeFetcher{feed: Feed{Version: "v3.19.0", Notes: "shiny things"}}
	c := NewChecker(store, fetch, "v3.18.0", nil, nil)

	c.Check(context.Background())

	if !store.setCalled {
		t.Fatal("release cache not written; want a newer verdict written")
	}
	if got := store.set.ReleaseState.String; got != "newer" {
		t.Fatalf("release_state = %q; want %q", got, "newer")
	}
	if got := store.set.ReleaseLatestVersion; got != (pgtype.Text{String: "v3.19.0", Valid: true}) {
		t.Fatalf("release_latest_version = %+v; want v3.19.0", got)
	}
	if got := store.set.ReleaseLatestNotes; got != (pgtype.Text{String: "shiny things", Valid: true}) {
		t.Fatalf("release_latest_notes = %+v; want the feed notes", got)
	}
}

func TestCheck_UpstreamEqual_WritesCurrent(t *testing.T) {
	store := &fakeStore{cfg: enabledCfg(true)}
	fetch := &fakeFetcher{feed: Feed{Version: "v3.18.0", Notes: "same"}}
	c := NewChecker(store, fetch, "v3.18.0", nil, nil)

	c.Check(context.Background())

	if !store.setCalled {
		t.Fatal("release cache not written; want a current verdict written")
	}
	if got := store.set.ReleaseState.String; got != "current" {
		t.Fatalf("release_state = %q; want %q", got, "current")
	}
	if store.set.ReleaseLatestVersion.Valid || store.set.ReleaseLatestNotes.Valid {
		t.Fatalf("latest fields set on a current verdict: %+v / %+v", store.set.ReleaseLatestVersion, store.set.ReleaseLatestNotes)
	}
}

func TestCheck_RunningAhead_WritesCurrent(t *testing.T) {
	store := &fakeStore{cfg: enabledCfg(true)}
	fetch := &fakeFetcher{feed: Feed{Version: "v3.17.0"}}
	c := NewChecker(store, fetch, "v3.18.0", nil, nil)

	c.Check(context.Background())

	if got := store.set.ReleaseState.String; got != "current" {
		t.Fatalf("release_state = %q; want %q", got, "current")
	}
}

func TestCheck_UpstreamUnreachable_NoOp(t *testing.T) {
	store := &fakeStore{cfg: enabledCfg(true)}
	fetch := &fakeFetcher{err: errors.New("dial tcp: no route to host")}
	c := NewChecker(store, fetch, "v3.18.0", nil, nil)

	c.Check(context.Background())

	if store.setCalled {
		t.Fatal("release cache written after an unreachable feed; want the cache untouched")
	}
}

func TestCheck_ConfigReadFails_NoOp(t *testing.T) {
	store := &fakeStore{cfgErr: errors.New("db down")}
	fetch := &fakeFetcher{feed: Feed{Version: "v9.9.9"}}
	c := NewChecker(store, fetch, "v1.0.0", nil, nil)

	c.Check(context.Background())

	if fetch.calls != 0 {
		t.Fatalf("fetcher called %d times after a config-read failure; want 0", fetch.calls)
	}
	if store.setCalled {
		t.Fatal("release cache written after a config-read failure; want no write")
	}
}

func TestCheck_UnstampedDevBuild_WritesCurrent(t *testing.T) {
	store := &fakeStore{cfg: enabledCfg(true)}
	fetch := &fakeFetcher{feed: Feed{Version: "v3.19.0"}}
	c := NewChecker(store, fetch, "dev", nil, nil)

	c.Check(context.Background())

	if got := store.set.ReleaseState.String; got != "current" {
		t.Fatalf("release_state = %q for a dev build; want %q (no false alarm)", got, "current")
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, running string
		want            bool
	}{
		{"v3.19.0", "v3.18.0", true},
		{"3.19.0", "3.18.0", true},
		{"v3.18.1", "v3.18.0", true},
		{"v4.0.0", "v3.18.0", true},
		{"v3.18.0", "v3.18.0", false},
		{"v3.17.0", "v3.18.0", false},
		{"v3.18.0", "v3.18.0-rc1", false},
		{"v3.18", "v3.18.0", false},
		{"v3.19", "v3.18.0", true},
		{"dev", "v3.18.0", false},
		{"v3.19.0", "dev", false},
		{"", "v3.18.0", false},
	}
	for _, tc := range cases {
		if got := isNewer(tc.latest, tc.running); got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v; want %v", tc.latest, tc.running, got, tc.want)
		}
	}
}
