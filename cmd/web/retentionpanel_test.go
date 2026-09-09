package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) CountHeldObservations(_ context.Context, _ int64) (db.CountHeldObservationsRow, error) {
	return db.CountHeldObservationsRow{CountedRows: f.heldObs, EstimatedRows: f.heldEstimate}, nil
}

func (f *fakeStore) ListDerivationBreaks(_ context.Context, _ int64) ([]db.ListDerivationBreaksRow, error) {
	return f.derivBreaks, nil
}

func (f *fakeStore) ListFacetSourceFloors(context.Context) ([]db.ListFacetSourceFloorsRow, error) {
	return f.facetFloors, nil
}

func (f *fakeStore) ListEnabledScans(context.Context) ([]db.Scan, error) {
	if f.listScansErr != nil {
		return nil, f.listScansErr
	}
	out := make([]db.Scan, 0, len(f.scans))
	for _, sc := range f.scans {
		if sc.Enabled {
			out = append(out, sc)
		}
	}
	return out, nil
}

func (f *fakeStore) ListCoveringScanKinds(context.Context) ([]string, error) {
	if f.coveringScanErr != nil {
		return nil, f.coveringScanErr
	}
	return f.coveringScans, nil
}

func seedRetentionPanel(f *fakeStore) {
	f.scans = append(f.scans,
		db.Scan{ID: 501, Kind: "dns", Enabled: true, CadenceSeconds: 86400},
		db.Scan{ID: 502, Kind: "tls-acceptance", Enabled: true, CadenceSeconds: 7 * 86400},
		db.Scan{ID: 503, Kind: "zone", Enabled: true, CadenceSeconds: 30 * 86400},
	)
	// The pair rows below are the same cover, reached per pair, so the two must agree.
	f.coveringScans = []string{"dns", "zone"}
	f.facetFloors = []db.ListFacetSourceFloorsRow{
		{Facet: "dns-record", Source: "resolver", TightestCadence: 86400, ScanKind: "dns", RowsHeld: 400, UncoveredRows: 7},
		{Facet: "dns-record", Source: "zone", TightestCadence: 30 * 86400, ScanKind: "zone", RowsHeld: 120},
		{Facet: "reachability", Source: "resolver", TightestCadence: 0, ScanKind: "", RowsHeld: 9, UncoveredRows: 9},
	}
	f.derivBreaks = []db.ListDerivationBreaksRow{{
		OpenedAt:   pgtype.Timestamptz{Time: time.Now().Add(-72 * time.Hour), Valid: true},
		Previous:   []byte(`[{"leaf":"resolution-walk","version":"3"},{"leaf":"wildcard-discrim","version":"1"}]`),
		Derivation: []byte(`[{"leaf":"resolution-walk","version":"4"},{"leaf":"wildcard-discrim","version":"2"}]`),
	}}
	f.heldObs = 529
	f.retention = db.GetRetentionSettingsRow{}
}

func TestCoverageCarriesTheDialsAndTheClampList(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/coverage", http.StatusOK)

	// The floor is a multiple and a named Scan, never a day count (ADR-0081).
	if !strings.Contains(got, "2 × cadence(dns)") {
		t.Errorf("observation floor is not a multiple of a named Scan; body lacks it")
	}
	if !strings.Contains(got, "2 × cadence(zone)") {
		t.Errorf("dispatch floor is not a multiple of a named Scan; body lacks it")
	}
	if strings.Contains(got, "at least 2 days") {
		t.Errorf("the panel still states the floor as a day count")
	}
	// The unbounded default is a parked terminal stop, never an empty field or a sentinel.
	if !strings.Contains(got, "keep everything") {
		t.Errorf("the terminal stop is missing")
	}
	for _, bad := range []string{"unlimited", "∞"} {
		if strings.Contains(got, bad) {
			t.Errorf("the unbounded default rendered as %q, a sentinel", bad)
		}
	}
	// One row per facet-source pair, each with its own floor and the Scan supplying it.
	for _, want := range []string{"dns-record · resolver", "dns-record · zone", "reachability · resolver"} {
		if !strings.Contains(got, want) {
			t.Errorf("pair row %q missing", want)
		}
	}
	if !strings.Contains(got, "the bound is undefined") {
		t.Errorf("an uncovered pair must say its bound is undefined, not that it is expired")
	}
	// The clamp list, with a Break row per leaf that moved in the transition.
	for _, leaf := range []string{"resolution-walk moved", "wildcard-discrim moved"} {
		if !strings.Contains(got, leaf) {
			t.Errorf("the clamp list does not name %q", leaf)
		}
	}
	// A pair with rows from a disabled Scan says so, rather than reading as fully covered.
	if !strings.Contains(got, "7 of these have no covering Scan") {
		t.Errorf("the uncovered rows inside a covered pair are not stated")
	}
	if !strings.Contains(got, "Every clamp in force") {
		t.Errorf("the clamp list is missing")
	}
	// The parked default is priced beside the dial: a default whose cost is not rendered
	// is a default nobody audits (ADR-0081). The fake declares an address scope, so the
	// denominator-less branch is covered by retention.TestProjectionStatesItHasNoDenominator.
	if !strings.Contains(got, "529 rows") {
		t.Errorf("the projection does not state what is held")
	}
	if !strings.Contains(got, "a year at the enabled cadences is") {
		t.Errorf("the projection does not price the position the handle is parked on")
	}
	// The panel carries no coverage figure, ever — no percentage, no bar, no threshold.
	if strings.Contains(got, "% covered") {
		t.Errorf("the retention panel grew a coverage figure")
	}
	// The discarded group renders in the state where it is always empty, and says why.
	if !strings.Contains(got, "Nothing has been discarded") {
		t.Errorf("the discarded group is missing in its empty state")
	}
}

func TestTheDialFloorNamesAScanThatBoundsARow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	// ADR-0081's walked case: no zone file, and ct enabled hourly bounds no row.
	f.scans = append(f.scans, db.Scan{ID: 504, Kind: "ct", Enabled: true, CadenceSeconds: 3600})
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/coverage", http.StatusOK)
	if !strings.Contains(got, "2 × cadence(dns)") {
		t.Errorf("the observation floor does not name dns, the tightest Scan that bounds a row")
	}
	if strings.Contains(got, "cadence(ct)") {
		t.Errorf("the observation floor names ct, a Scan that bounds no row — the operator would move the wrong Scan")
	}
	// A floor from an uncovered Scan would put a below-floor stop on the track.
	if strings.Contains(got, `value="1"`) {
		t.Errorf("a stop below the 2-day covering floor is offered on the track")
	}
}

func TestAFailedCoveringScanReadWithholdsThePanel(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	f.coveringScanErr = errors.New("boom")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/coverage", http.StatusOK)
	// An unread cover would draw the dial with no floor, offering the ground as a stop (ADR-0081).
	if strings.Contains(got, `action="/coverage/retention"`) {
		t.Errorf("a failed covering-Scan read still rendered the dial form")
	}
	if !strings.Contains(got, "did not resolve on this load") {
		t.Errorf("the withheld panel does not say why")
	}
}

func TestTheDiscardedGroupDoesNotClaimNothingOnceADialBinds(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	f.retention.ObservationCurrencyDays = 90
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/coverage", http.StatusOK)
	if strings.Contains(got, "Nothing has been discarded") {
		t.Errorf("a bounded dial retires rows on each sweep, so the group may not say nothing was discarded")
	}
	if !strings.Contains(got, "No per-row record of a retirement is kept") {
		t.Errorf("the discarded group does not say why it lists nothing by name")
	}
}

func TestAFailedDialReadWithholdsThePanel(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	f.retentionErr = errors.New("boom")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/coverage", http.StatusOK)
	// A database failure must not render as an editable form parked on keep everything.
	if strings.Contains(got, `action="/coverage/retention"`) {
		t.Errorf("a failed settings read still rendered the dial form")
	}
	if strings.Contains(got, "Nothing has been discarded") || strings.Contains(got, "No clamp is in force yet") {
		t.Errorf("a failed read rendered as a legitimate empty state")
	}
	if !strings.Contains(got, "did not resolve on this load") {
		t.Errorf("the withheld panel does not say why")
	}
}

func TestAWriteWithNoFloorIsRefusedNotPersisted(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	f.retention.ObservationCurrencyDays = 90
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	f.listScansErr = errors.New("boom")
	resp := postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"1"}, "dispatch_cadence_multiple": {"4"},
	})
	resp.Body.Close()
	if resp.StatusCode == http.StatusSeeOther {
		t.Fatalf("a write that could not compute its floor was accepted")
	}
	if f.retention.ObservationCurrencyDays != 90 {
		t.Errorf("a below-floor value was persisted un-raised: %+v", f.retention)
	}
}

func TestABelowFloorDialIsUnreachableRatherThanRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// No stop below the floor is offered, so the control cannot express one.
	body := getBody(t, ac, base+"/coverage", http.StatusOK)
	if strings.Contains(body, `value="1"`) {
		t.Errorf("a stop below the 2-day floor is offered on the track")
	}

	// A hand-made post of a below-floor value is raised, never refused.
	resp := postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"1"}, "dispatch_cadence_multiple": {"1"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("below-floor post: status=%d, want a redirect and no error banner", resp.StatusCode)
	}
	resp.Body.Close()
	if f.retention.ObservationCurrencyDays != 2 {
		t.Errorf("observation dial = %d, want 2 — raised to the floor", f.retention.ObservationCurrencyDays)
	}
	if f.retention.DispatchCadenceMultiple != 2 {
		t.Errorf("dispatch dial = %d, want 2 — raised to the floor", f.retention.DispatchCadenceMultiple)
	}
	if !f.retention.UpdatedBy.Valid {
		t.Errorf("updated_by not attributed")
	}

	// A negative or unreadable post is not the terminal stop: it leaves the dial alone.
	f.retention.ObservationCurrencyDays, f.retention.DispatchCadenceMultiple = 90, 4
	resp = postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"-1"}, "dispatch_cadence_multiple": {"soon"},
	})
	resp.Body.Close()
	if f.retention.ObservationCurrencyDays != 90 || f.retention.DispatchCadenceMultiple != 4 {
		t.Errorf("an unreadable post moved a dial to keep everything: %+v", f.retention)
	}
}

func TestTheDialsPersistFromCoverage(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	f.retention.TranscriptCurrencyDays = 30
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"365"}, "dispatch_cadence_multiple": {"8"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("save: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if f.retention.ObservationCurrencyDays != 365 || f.retention.DispatchCadenceMultiple != 8 {
		t.Fatalf("dials not persisted: %+v", f.retention)
	}
	// The transcript dial is not on this form and may not be cleared by it (ADR-0126).
	if f.retention.TranscriptCurrencyDays != 30 {
		t.Errorf("transcript dial = %d, want 30 untouched", f.retention.TranscriptCurrencyDays)
	}
}

func TestSettingsLinksToTheDialsAndCarriesNoCopy(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/settings?tab=delivery", http.StatusOK)
	if !strings.Contains(got, `href="/coverage#retention"`) {
		t.Errorf("Settings does not link to the dials on Coverage")
	}
	for _, copy := range []string{"observation_currency_days", "dispatch_cadence_multiple"} {
		if strings.Contains(got, copy) {
			t.Errorf("Settings still carries a copy of the %s dial", copy)
		}
	}
	// The transcript dial stays here: it has a fixed floor and no coverage-style derivation.
	if !strings.Contains(got, "transcript_currency_days") {
		t.Errorf("the transcript dial left Settings")
	}
	if strings.Contains(got, "lands with later work") {
		t.Errorf("the stale floor note survives")
	}
}

func TestTheHeldFigureSaysWhenItIsAnEstimate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	// Above the cap the count stops, so the projection prices the corpus from the estimate.
	f.heldObs = heldCountExactLimit + 1
	f.heldEstimate = 97925120
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/coverage", http.StatusOK)
	if !strings.Contains(got, "97,925,120 rows") {
		t.Errorf("the projection does not price the corpus from the estimate")
	}
	// A price is exact over what the operator typed, so an estimate must say it is one (ADR-0081).
	if !strings.Contains(got, "an estimate") {
		t.Errorf("the held figure is an estimate and the panel does not say so")
	}
	if !strings.Contains(got, "about 97,925,120 rows") {
		t.Errorf("the estimated held figure is not qualified where it is read")
	}
}

func TestACountedHeldFigureIsNotCalledAnEstimate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	f.heldEstimate = 4_000_000
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/coverage", http.StatusOK)
	if !strings.Contains(got, "529 rows") {
		t.Errorf("a count under the cap must be shown, not the estimate beside it")
	}
	if strings.Contains(got, "an estimate") {
		t.Errorf("a counted held figure is called an estimate")
	}
}

func TestACappedHeldCountNeverUnderstatesTheCorpus(t *testing.T) {
	const cap = 100
	for _, tc := range []struct {
		name          string
		row           db.CountHeldObservationsRow
		wantRows      int64
		wantEstimated bool
		wantPriced    bool
	}{
		{"under the cap", db.CountHeldObservationsRow{CountedRows: 99, EstimatedRows: 4}, 99, false, true},
		{"at the cap", db.CountHeldObservationsRow{CountedRows: 100, EstimatedRows: 4}, 100, false, true},
		{"over the cap", db.CountHeldObservationsRow{CountedRows: 101, EstimatedRows: 9000}, 9000, true, true},
		// A capped count is a floor, so drawing it as an estimate of the whole corpus misprices it.
		{"stale statistic", db.CountHeldObservationsRow{CountedRows: 101, EstimatedRows: 50}, 0, false, false},
		{"cold collector", db.CountHeldObservationsRow{CountedRows: 101, EstimatedRows: 0}, 0, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, estimated, priced := heldRows(tc.row, cap)
			if rows != tc.wantRows || estimated != tc.wantEstimated || priced != tc.wantPriced {
				t.Errorf("heldRows = (%d, %v, %v), want (%d, %v, %v)",
					rows, estimated, priced, tc.wantRows, tc.wantEstimated, tc.wantPriced)
			}
		})
	}
}

func TestAColdStatisticWithholdsTheProjection(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedRetentionPanel(f)
	// A table that no vacuum and no analyse has reached reports zero live tuples.
	f.heldObs = heldCountExactLimit + 1
	f.heldEstimate = 0
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := getBody(t, ac, base+"/coverage", http.StatusOK)
	if !strings.Contains(got, "The projection is withheld") {
		t.Errorf("a statistic below the capped count must withhold the projection")
	}
	if strings.Contains(got, "about 100,001 rows") {
		t.Errorf("the panel prices the corpus from the capped count")
	}
	if strings.Contains(got, "a year at the enabled cadences") {
		t.Errorf("the projection rendered from a figure that estimates nothing")
	}
}
