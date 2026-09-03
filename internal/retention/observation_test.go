package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

const (
	daily   = int64(86400)
	monthly = int64(2592000)
)

func TestObservationBoundIsPerTimeline(t *testing.T) {
	resolver, ok := ObservationBoundSeconds(daily)
	if !ok || resolver != FloorCadences*daily {
		t.Fatalf("resolver bound = (%d,%v), want (%d,true)", resolver, ok, FloorCadences*daily)
	}
	zone, ok := ObservationBoundSeconds(monthly)
	if !ok || zone != FloorCadences*monthly {
		t.Fatalf("zone bound = (%d,%v), want (%d,true)", zone, ok, FloorCadences*monthly)
	}
	if zone <= resolver {
		t.Fatalf("a monthly timeline must outlive a daily one: zone=%d resolver=%d", zone, resolver)
	}
	if _, ok := ObservationBoundSeconds(0); ok {
		t.Error("no covering Scan must yield an undefined bound (ok=false), not a number")
	}
}

func TestObservationFloorIsTheTightestBound(t *testing.T) {
	floor, ok := ObservationFloorDays(daily)
	if !ok || floor != 2 {
		t.Fatalf("floor over a daily Scan = (%d,%v), want (2,true)", floor, ok)
	}
	cases := []struct {
		name    string
		dial    int64
		cadence int64
		below   bool
	}{
		{"zero is unbounded, always allowed", 0, daily, false},
		{"one day is below the 2-day floor", 1, daily, true},
		{"exactly the floor is allowed", 2, daily, false},
		{"above the floor is allowed", 90, daily, false},
		{"no enabled Scan: nothing to floor against", 1, 0, false},
	}
	for _, c := range cases {
		if got := BelowObservationFloor(c.dial, c.cadence); got != c.below {
			t.Errorf("%s: BelowObservationFloor(%d,%d)=%v, want %v", c.name, c.dial, c.cadence, got, c.below)
		}
	}
}

func TestFloorSettingBelowChangesZeroRows(t *testing.T) {
	// ADR-0094's derivation: below the tightest bound every positive dial gives the same answer.
	tightest := FloorCadences * daily
	type row struct {
		age   int64
		bound int64
	}
	rows := []row{
		{age: tightest / 2, bound: FloorCadences * daily},
		{age: tightest * 3, bound: FloorCadences * daily},
		{age: FloorCadences * monthly / 2, bound: FloorCadences * monthly},
		{age: FloorCadences * monthly * 3, bound: FloorCadences * monthly},
	}
	baseline := int64(1)
	for _, dial := range []int64{1, tightest / 2, tightest} {
		for _, r := range rows {
			want := RetainObservation(r.age, r.bound, baseline, true, false)
			got := RetainObservation(r.age, r.bound, dial, true, false)
			if got != want {
				t.Errorf("dial=%d row(age=%d bound=%d): retain=%v, but dead-zone baseline gave %v",
					dial, r.age, r.bound, got, want)
			}
		}
	}
}

func TestRetainNormalTimeline(t *testing.T) {
	bound := FloorCadences * daily
	dial := int64(10) * SecondsPerDay
	cases := []struct {
		name   string
		age    int64
		dial   int64
		retain bool
	}{
		{"live: inside its own bound, kept even with no dial", bound - 1, 0, true},
		{"live: inside its own bound, dial irrelevant", bound - 1, dial, true},
		{"evidential but inside the dial: kept", bound + 1, dial, true},
		{"evidential past both bound and dial: retired", dial + 1, dial, false},
		{"evidential with unbounded dial: kept (corpus grows)", bound * 1000, 0, true},
	}
	for _, c := range cases {
		if got := RetainObservation(c.age, bound, c.dial, true, false); got != c.retain {
			t.Errorf("%s: retain=%v, want %v", c.name, got, c.retain)
		}
	}
}

func TestUndefinedBoundNeverRetired(t *testing.T) {
	dial := int64(5) * SecondsPerDay
	veryOld := dial * 100
	if !RetainObservation(veryOld, 0, dial, false, false) {
		t.Fatal("an undefined-bound row must never be retired, however old and whatever the dial")
	}
	if RetainObservation(veryOld, FloorCadences*daily, dial, true, false) {
		t.Fatal("a defined-bound row of the same age past the dial must be retired — the exception is undefined-only")
	}
}

func TestWithdrawnDialAlone(t *testing.T) {
	bound := FloorCadences * monthly
	dial := int64(3) * SecondsPerDay
	ageInsideBound := bound - 1
	if !RetainObservation(ageInsideBound, bound, dial, true, false) {
		t.Fatal("active subject: a row inside its own bound is live and kept")
	}
	if RetainObservation(ageInsideBound, bound, dial, true, true) {
		t.Fatal("withdrawn subject: no floor, so a row past the dial is retired even inside the old bound")
	}
	if !RetainObservation(bound*100, bound, 0, true, true) {
		t.Fatal("withdrawn subject with an unbounded dial keeps everything")
	}
}

func TestLiveOnlyGatesDerivationReads(t *testing.T) {
	bound := FloorCadences * daily
	rows := []AgedObservation{
		{ID: 1, AgeSeconds: bound - 1, BoundSeconds: bound, HasBound: true},
		{ID: 2, AgeSeconds: bound + 1, BoundSeconds: bound, HasBound: true},
		{ID: 3, AgeSeconds: bound * 50, BoundSeconds: bound, HasBound: true},
		{ID: 4, AgeSeconds: 1, BoundSeconds: 0, HasBound: false},
	}
	live := LiveOnly(rows)
	if len(live) != 1 || live[0].ID != 1 {
		t.Fatalf("LiveOnly returned %+v, want only the live row (id 1)", live)
	}
	evidential := map[int64]bool{2: true, 3: true, 4: true}
	for _, r := range live {
		if evidential[r.ID] {
			t.Fatalf("derivation read reached evidential row %d — the separation is broken", r.ID)
		}
		if TierOf(r.AgeSeconds, r.BoundSeconds, r.HasBound) != Live {
			t.Fatalf("row %d passed the gate but is not live-tier", r.ID)
		}
	}
}

type fakeObsStore struct {
	days         int64
	settingsErr  error
	deleteCalled bool
	deleteArg    db.DeleteExpiredObservationsParams
	deleted      int64
	deleteErr    error
}

func (f *fakeObsStore) GetRetentionSettings(context.Context) (db.GetRetentionSettingsRow, error) {
	return db.GetRetentionSettingsRow{ObservationCurrencyDays: f.days}, f.settingsErr
}

func (f *fakeObsStore) DeleteExpiredObservations(_ context.Context, arg db.DeleteExpiredObservationsParams) (int64, error) {
	f.deleteCalled = true
	f.deleteArg = arg
	return f.deleted, f.deleteErr
}

func TestObservationSweepUnboundedDeletesNothing(t *testing.T) {
	f := &fakeObsStore{days: 0}
	r := NewObservationRetirer(f, time.Now, nil)
	n, err := r.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 0 {
		t.Errorf("retired %d rows, want 0 when the dial is unbounded", n)
	}
	if f.deleteCalled {
		t.Error("delete must not be called when the dial is at 0 (unbounded default)")
	}
}

func TestObservationSweepBoundedDeletesWithOwnBound(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	f := &fakeObsStore{days: 90, deleted: 12}
	r := NewObservationRetirer(f, func() time.Time { return now }, nil)

	n, err := r.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if n != 12 {
		t.Errorf("retired %d rows, want 12", n)
	}
	if !f.deleteCalled {
		t.Fatal("delete must be called when the dial is set")
	}
	if f.deleteArg.DialSeconds != 90*SecondsPerDay {
		t.Errorf("dial seconds = %d, want %d (90 days)", f.deleteArg.DialSeconds, 90*SecondsPerDay)
	}
	if f.deleteArg.FloorCadences != FloorCadences {
		t.Errorf("floor cadences = %d, want k=%d", f.deleteArg.FloorCadences, FloorCadences)
	}
	if !f.deleteArg.AsOf.Time.Equal(now) {
		t.Errorf("as_of = %v, want %v", f.deleteArg.AsOf.Time, now)
	}
}

func TestObservationSweepPropagatesReadError(t *testing.T) {
	sentinel := errors.New("boom")
	f := &fakeObsStore{days: 10, settingsErr: sentinel}
	r := NewObservationRetirer(f, nil, nil)
	if _, err := r.Sweep(context.Background()); !errors.Is(err, sentinel) {
		t.Errorf("Sweep error = %v, want %v", err, sentinel)
	}
	if f.deleteCalled {
		t.Error("delete must not run after a settings read error")
	}
}
