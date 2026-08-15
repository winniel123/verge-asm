package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

const (
	daily   = int64(86400)   // the dns Scan's cadence
	monthly = int64(2592000) // the zone Scan's cadence (30 days)
)

// TestObservationBoundIsPerTimeline proves the live/evidential boundary is k
// cadences of the timeline's OWN covering Scan, never a global number: the
// resolver timeline (daily) and the zone timeline (monthly) get different bounds,
// and a timeline no enabled Scan covers has an undefined bound rather than a loose
// one. (AC: boundary computed per-timeline from its covering Scan's cadence.)
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
	// A timeline no enabled Scan covers: undefined, not loose.
	if _, ok := ObservationBoundSeconds(0); ok {
		t.Error("no covering Scan must yield an undefined bound (ok=false), not a number")
	}
}

// TestObservationFloorIsTheTightestBound proves the dial's floor is the tightest
// bound in force — k cadences of the tightest ENABLED Scan — and that below it any
// positive dial is rejected while 0 (unbounded) is always allowed. With no enabled
// Scan there is no bound to floor against and every non-negative dial is allowed.
// (AC: dial floors at the tightest in-force bound.)
func TestObservationFloorIsTheTightestBound(t *testing.T) {
	// dns daily is the tightest shipped Scan: floor = k*daily = 2 days.
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

// TestFloorSettingBelowChangesZeroRows is the derivation of the floor (ADR-0094):
// below the tightest bound the control changes no row at all. For any non-withdrawn
// row whose own bound is at least the tightest bound, every positive dial value at
// or below that tightest bound yields the SAME retention decision — so a dial set
// anywhere in the dead zone (0, floor] is indistinguishable and there is nothing to
// gain by allowing it.
func TestFloorSettingBelowChangesZeroRows(t *testing.T) {
	tightest := FloorCadences * daily // the tightest bound in force
	// Rows spanning both tiers on two different timelines (daily and monthly bounds).
	type row struct {
		age   int64
		bound int64
	}
	rows := []row{
		{age: tightest / 2, bound: FloorCadences * daily},                  // live daily
		{age: tightest * 3, bound: FloorCadences * daily},                  // evidential daily
		{age: FloorCadences * monthly / 2, bound: FloorCadences * monthly}, // live zone
		{age: FloorCadences * monthly * 3, bound: FloorCadences * monthly}, // evidential zone
	}
	baseline := int64(1) // any positive dial in the dead zone
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

// TestRetainNormalTimeline covers the ordinary rule: a row is retained while its
// age is inside EITHER its own bound OR the dial, whichever is longer.
func TestRetainNormalTimeline(t *testing.T) {
	bound := FloorCadences * daily // 2 days
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

// TestUndefinedBoundNeverRetired proves the first exception: a timeline with no
// covering Scan (an enabled-then-disabled-Scan scenario) has an undefined bound and
// is never retired — even past any dial, and even when the dial would retire an
// ordinary row of the same age. Undefined is not expired. (AC.)
func TestUndefinedBoundNeverRetired(t *testing.T) {
	dial := int64(5) * SecondsPerDay
	veryOld := dial * 100
	if !RetainObservation(veryOld, 0, dial, false /*hasBound*/, false) {
		t.Fatal("an undefined-bound row must never be retired, however old and whatever the dial")
	}
	// Contrast: the same age on a defined (daily) bound IS retired past the dial.
	if RetainObservation(veryOld, FloorCadences*daily, dial, true, false) {
		t.Fatal("a defined-bound row of the same age past the dial must be retired — the exception is undefined-only")
	}
}

// TestWithdrawnDialAlone proves the second exception: a withdrawn subject's
// timelines carry no floor at all, so the dial alone governs — a row that would be
// LIVE on an active subject is still retired once past the dial, and with an
// unbounded dial nothing is retired. (AC.)
func TestWithdrawnDialAlone(t *testing.T) {
	bound := FloorCadences * monthly // a would-be-generous 60-day bound
	dial := int64(3) * SecondsPerDay
	// Age inside the bound but past the dial: an active subject keeps it (live),
	// a withdrawn subject does not — no floor.
	ageInsideBound := bound - 1
	if !RetainObservation(ageInsideBound, bound, dial, true, false) {
		t.Fatal("active subject: a row inside its own bound is live and kept")
	}
	if RetainObservation(ageInsideBound, bound, dial, true, true /*withdrawn*/) {
		t.Fatal("withdrawn subject: no floor, so a row past the dial is retired even inside the old bound")
	}
	// Unbounded dial: a withdrawn subject's rows are all kept (dial alone, dial off).
	if !RetainObservation(bound*100, bound, 0, true, true) {
		t.Fatal("withdrawn subject with an unbounded dial keeps everything")
	}
}

// TestLiveOnlyGatesDerivationReads proves the readability separation (AC): a
// derivation reading through LiveOnly sees only live-tier rows, so an evidential
// row — past its bound, or on an uncovered timeline — is unreadable by any
// derivation path. The "derivation" here is a fold that would re-derive history; we
// assert it never receives an evidential row.
func TestLiveOnlyGatesDerivationReads(t *testing.T) {
	bound := FloorCadences * daily
	rows := []AgedObservation{
		{ID: 1, AgeSeconds: bound - 1, BoundSeconds: bound, HasBound: true},  // live
		{ID: 2, AgeSeconds: bound + 1, BoundSeconds: bound, HasBound: true},  // evidential (aged out)
		{ID: 3, AgeSeconds: bound * 50, BoundSeconds: bound, HasBound: true}, // evidential (long dead)
		{ID: 4, AgeSeconds: 1, BoundSeconds: 0, HasBound: false},             // evidential (no covering Scan)
	}
	live := LiveOnly(rows)
	if len(live) != 1 || live[0].ID != 1 {
		t.Fatalf("LiveOnly returned %+v, want only the live row (id 1)", live)
	}
	// A derivation fold over the gated rows must never observe an evidential id.
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

// --- Retirer ---------------------------------------------------------------

// fakeObsStore is the whole surface the ObservationRetirer can reach: the dial and
// the observation-only delete. It records the params so the sweep's dial-to-seconds
// conversion and k are checkable, and can return an error to prove propagation.
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
