package main

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) InDialTx(ctx context.Context, fn func(dialQueries, recorder) error) error {
	f.dialMu.Lock()
	defer f.dialMu.Unlock()
	if f.insideDialLock != nil {
		f.insideDialLock()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	undo := f.snapshotDials()
	// inTx matches the transaction-bound recorder the pool store passes (spec §7.6).
	if err := fn(f, recorder{store: f, inTx: true}); err != nil {
		undo()
		return err
	}
	return nil
}

// A rollback is what makes a failed Act undo the move, so the fake must model one (ADR-1914 §4).

func (f *fakeStore) snapshotDials() func() {
	retentionRow, cfg := f.retention, f.instanceConfig
	zone, dns := f.zoneCadence, f.dnsCadence
	acts, trail := len(f.acts), len(f.actTrail)
	rows, nextID := len(f.actRows), f.actNextID
	return func() {
		f.retention, f.instanceConfig = retentionRow, cfg
		f.zoneCadence, f.dnsCadence = zone, dns
		f.acts, f.actTrail = f.acts[:acts], f.actTrail[:trail]
		f.actRows, f.actNextID = f.actRows[:rows], nextID
	}
}

func (f *fakeStore) LockRetentionSettings(ctx context.Context) (db.LockRetentionSettingsRow, error) {
	row, err := f.GetRetentionSettings(ctx)
	if err != nil {
		return db.LockRetentionSettingsRow{}, err
	}
	return db.LockRetentionSettingsRow{
		ObservationCurrencyDays: row.ObservationCurrencyDays,
		DispatchCadenceMultiple: row.DispatchCadenceMultiple,
		TranscriptCurrencyDays:  row.TranscriptCurrencyDays,
	}, nil
}

func (f *fakeStore) LockInstanceConfig(ctx context.Context) (db.LockInstanceConfigRow, error) {
	cfg, err := f.GetInstanceConfig(ctx)
	if err != nil {
		return db.LockInstanceConfigRow{}, err
	}
	return db.LockInstanceConfigRow{
		ApiEnabled:         cfg.ApiEnabled,
		UpdateCheckEnabled: cfg.UpdateCheckEnabled,
		SeedAddressCap:     cfg.SeedAddressCap,
	}, nil
}

func (f *fakeStore) LockZoneCadenceSeconds(ctx context.Context) (int64, error) {
	return f.GetZoneCadenceSeconds(ctx)
}

func (f *fakeStore) LockDnsCadenceSeconds(ctx context.Context) (int64, error) {
	return f.GetDnsCadenceSeconds(ctx)
}

// One logical move submitted twice writes one Act row. The corpus is append-only, so a
// second row for the same move can never be retracted (ADR-1914).

// The gate holds the first submit inside the section until the second has started, so the
// overlap is the test's own rather than the scheduler's.

func TestTwoConcurrentSubmitsOfOneDialMoveRecordOneRow(t *testing.T) {
	f := newFakeStore()
	f.instanceConfig.ApiEnabled = true
	base, ac := adminSession(t, f)
	f.acts = nil

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var first sync.Once
	f.insideDialLock = func() {
		first.Do(func() {
			entered <- struct{}{}
			<-release
		})
	}

	post := func(errs chan<- error) {
		resp, err := ac.PostForm(base+"/settings/api", url.Values{"enabled": {"false"}})
		if err == nil {
			resp.Body.Close()
		}
		errs <- err
	}

	errs := make(chan error, 2)
	go post(errs)
	// The first submit holds the section, so the second cannot enter before it commits.
	<-entered
	go post(errs)
	close(release)

	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("POST /settings/api: %v", err)
		}
	}

	if got := actClasses(f); len(got) != 1 || got[0] != "api.access.moved" {
		t.Fatalf("two submits of one move recorded %v, want one api.access.moved", got)
	}
	if f.instanceConfig.ApiEnabled {
		t.Error("the move did not land, so the corpus and the estate disagree")
	}
}

// A failed Act rolls its move back, so no move stands that the corpus does not hold.

func TestAFailedRecordRollsTheDialMoveBack(t *testing.T) {
	f := newFakeStore()
	f.instanceConfig.ApiEnabled = true
	base, ac := adminSession(t, f)
	f.acts = nil
	f.actErr = errors.New("insert act: connection reset")

	postForm(t, ac, base+"/settings/api", url.Values{"enabled": {"false"}}).Body.Close()

	if !f.instanceConfig.ApiEnabled {
		t.Error("the move stood after its Act failed, so the estate moved and the corpus did not")
	}
	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a failed insert recorded %v, want nothing", got)
	}
}

// A stale read cannot lose the row: the loser of the race reads the winner's value and
// finds its own submit is a move (ADR-1914 §4).

func TestASubmitBehindAnotherStillRecordsItsOwnMove(t *testing.T) {
	f := newFakeStore()
	f.instanceConfig.ApiEnabled = false
	base, ac := adminSession(t, f)
	f.acts = nil

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var first sync.Once
	f.insideDialLock = func() {
		first.Do(func() {
			entered <- struct{}{}
			<-release
		})
	}

	post := func(enabled string, errs chan<- error) {
		resp, err := ac.PostForm(base+"/settings/api", url.Values{"enabled": {enabled}})
		if err == nil {
			resp.Body.Close()
		}
		errs <- err
	}

	errs := make(chan error, 2)
	go post("true", errs)
	<-entered
	go post("false", errs)
	close(release)

	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("POST /settings/api: %v", err)
		}
	}

	if got := actClasses(f); len(got) != 2 {
		t.Fatalf("two moves recorded %v, want two rows", got)
	}
	if got := actSubjects(t, f, "api.access.moved"); len(got) != 2 || got[1] != "API access · off" {
		t.Fatalf("the trailing submit recorded %v, want it to end at off", got)
	}
}
