package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) CountHeldObservations(context.Context) (int64, error) {
	return f.heldObs, nil
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

func seedRetentionPanel(f *fakeStore) {
	f.scans = append(f.scans,
		db.Scan{ID: 501, Kind: "dns", Enabled: true, CadenceSeconds: 86400},
		db.Scan{ID: 502, Kind: "tls-acceptance", Enabled: true, CadenceSeconds: 7 * 86400},
		db.Scan{ID: 503, Kind: "zone", Enabled: true, CadenceSeconds: 30 * 86400},
	)
	f.facetFloors = []db.ListFacetSourceFloorsRow{
		{Facet: "dns-record", Source: "resolver", TightestCadence: 86400, ScanKind: "dns", RowsHeld: 400},
		{Facet: "dns-record", Source: "zone", TightestCadence: 30 * 86400, ScanKind: "zone", RowsHeld: 120},
		{Facet: "reachability", Source: "resolver", TightestCadence: 0, ScanKind: "", RowsHeld: 9},
	}
	f.derivBreaks = []db.ListDerivationBreaksRow{{
		OpenedAt:   pgtype.Timestamptz{Time: time.Now().Add(-72 * time.Hour), Valid: true},
		Previous:   []byte(`[{"leaf":"resolution-walk","version":"3"}]`),
		Derivation: []byte(`[{"leaf":"resolution-walk","version":"4"}]`),
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
	// The clamp list, with the Break naming the leaf that moved.
	if !strings.Contains(got, "resolution-walk moved") {
		t.Errorf("the binding clamp does not name the leaf that moved")
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
