package main

import (
	"context"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

// The design-owned TopNav "Scan running" pill is global — it renders on EVERY view
// (R4-D3, #758). Only the dashboard passes its own m["Scanning"]; every other page
// passes none, so injectChrome must read the in-flight flag centrally so the pill
// lights the same on a Signals, Inventory, or Coverage view as on the dashboard, and
// clears when no scan is running.
func TestChromeScanPillLightsOnEveryView(t *testing.T) {
	f := newFakeStore()
	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	srv := newServer(f, testKey, "", fixedClock())

	// A hot dispatch in flight — a view that does NOT set its own "Scanning" (i.e.,
	// anything but the dashboard) still lights the shell pill from the central read.
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0), // active
	}
	data := map[string]any{"IsAdmin": true, "NavActive": "signals"}
	srv.injectChrome(data, nil)
	c, ok := data["Chrome"].(*chromeVM)
	if !ok {
		t.Fatalf("injectChrome set no *chromeVM: %T", data["Chrome"])
	}
	if !c.ScanRunning {
		t.Errorf("a scan is in flight but the shell pill is unlit on a non-dashboard view")
	}

	// No active dispatch (all complete) — the pill clears.
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(9, "dns", tick, 2, 0, 0, 2, 0, 0), // complete
	}
	data = map[string]any{"IsAdmin": true, "NavActive": "signals"}
	srv.injectChrome(data, nil)
	c = data["Chrome"].(*chromeVM)
	if c.ScanRunning {
		t.Errorf("no scan is in flight but the shell pill is still lit")
	}

	// A page that carries its own m["Scanning"] (the dashboard) is authoritative —
	// the central read does not override an explicitly-passed flag.
	f.dispatchProgress = nil // central read would say not-running
	data = map[string]any{"IsAdmin": true, "NavActive": "dashboard", "Scanning": true}
	srv.injectChrome(data, nil)
	c = data["Chrome"].(*chromeVM)
	if !c.ScanRunning {
		t.Errorf("a page that set Scanning=true had its shell pill overridden dark")
	}
}

// chromeScanRunning is the one seam the pill rests on: true when any kind is active,
// false when none, matching activeDispatchKinds.
func TestChromeScanRunning(t *testing.T) {
	f := newFakeStore()
	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	srv := newServer(f, testKey, "", fixedClock())

	if srv.chromeScanRunning(context.Background()) {
		t.Errorf("no dispatches seeded but chromeScanRunning reports running")
	}

	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0), // active
	}
	if !srv.chromeScanRunning(context.Background()) {
		t.Errorf("a hot dispatch is in flight but chromeScanRunning reports idle")
	}

	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(9, "dns", tick, 2, 0, 0, 2, 0, 0), // complete
	}
	if srv.chromeScanRunning(context.Background()) {
		t.Errorf("all dispatches complete but chromeScanRunning still reports running")
	}
}
