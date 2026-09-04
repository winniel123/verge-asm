package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

func activeDispatchStore(t *testing.T, ready, running int64) *fakeStore {
	t.Helper()
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	total := ready + running + 4
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(1408, "standard", tick, total, ready, running, 4, 0, 0),
	}
	return f
}

func TestStopScanAdminCancelsPending(t *testing.T) {
	f := activeDispatchStore(t, 2, 1)
	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/scans/stop", url.Values{"id": {"1408"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != "/settings?tab=scans" {
		t.Fatalf("stop: status=%d loc=%q, want 303 to /settings?tab=scans", resp.StatusCode, loc)
	}
	if got := f.dispatchStatus[1408]; got != "stopped" {
		t.Fatalf("dispatch status = %q, want stopped", got)
	}
	if f.dispatchProgress[0].Ready != 0 {
		t.Errorf("pending jobs not cancelled: ready = %d, want 0", f.dispatchProgress[0].Ready)
	}
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "Dispatch stopped") {
		t.Errorf("stop receipt missing; body: %s", page)
	}
	if !strings.Contains(page, "2 pending jobs cancelled") || !strings.Contains(page, "1 running finishing") {
		t.Errorf("stop receipt figures wrong; body: %s", page)
	}
}

func TestTerminateScanAdminKills(t *testing.T) {
	f := activeDispatchStore(t, 2, 1)
	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/scans/terminate", url.Values{"id": {"1408"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != "/settings?tab=scans" {
		t.Fatalf("terminate: status=%d loc=%q, want 303 to /settings?tab=scans", resp.StatusCode, loc)
	}
	if got := f.dispatchStatus[1408]; got != "terminated" {
		t.Fatalf("dispatch status = %q, want terminated", got)
	}
	if f.dispatchProgress[0].Ready != 0 || f.dispatchProgress[0].Running != 0 {
		t.Errorf("in-flight jobs not cancelled: ready=%d running=%d, want 0/0",
			f.dispatchProgress[0].Ready, f.dispatchProgress[0].Running)
	}
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "Scan terminated") || !strings.Contains(page, "3 jobs stopped") {
		t.Errorf("terminate receipt missing/wrong; body: %s", page)
	}
}

func TestStopTerminateViewerForbidden(t *testing.T) {
	for _, path := range []string{"/scans/stop", "/scans/terminate"} {
		f := newFakeStore()
		seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
		tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
		f.dispatchProgress = []db.ListDispatchProgressRow{
			progressRow(1408, "standard", tick, 6, 2, 1, 3, 0, 0),
		}
		base := startWithTrigger(t, f, &fakeTrigger{})
		ac := login(t, base, "viewer", "hunter2hunter2")

		resp := postForm(t, ac, base+path, url.Values{"id": {"1408"}})
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s viewer: status = %d, want 403", path, resp.StatusCode)
		}
		if len(f.dispatchStatus) != 0 {
			t.Fatalf("%s: a viewer's request recorded a status: %v", path, f.dispatchStatus)
		}
	}
}

func TestStopScanConcludedOrUnknown(t *testing.T) {
	cases := []struct {
		name string
		id   string
		seed func(*fakeStore)
	}{
		{"unknown", "999999", func(*fakeStore) {}},
		{"concluded", "1408", func(f *fakeStore) {
			tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
			f.dispatchProgress = []db.ListDispatchProgressRow{
				progressRow(1408, "standard", tick, 6, 0, 0, 6, 0, 0),
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStore()
			seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
			tc.seed(f)
			base := startWithTrigger(t, f, &fakeTrigger{})
			ac := login(t, base, "admin", "hunter2hunter2")

			resp := postForm(t, ac, base+"/scans/stop", url.Values{"id": {tc.id}})
			loc := resp.Header.Get("Location")
			resp.Body.Close()
			if resp.StatusCode != http.StatusSeeOther || loc != "/settings?tab=scans" {
				t.Fatalf("%s: status=%d loc=%q, want 303 to /settings?tab=scans", tc.name, resp.StatusCode, loc)
			}
			if len(f.dispatchStatus) != 0 {
				t.Errorf("%s: a concluded/unknown dispatch recorded a status: %v", tc.name, f.dispatchStatus)
			}
			page := getBody(t, ac, base+loc, http.StatusOK)
			if !strings.Contains(page, "Dispatch already concluded") {
				t.Errorf("%s: danger flash missing; body: %s", tc.name, page)
			}
		})
	}
}

func TestScansDialogsReachableForAdmin(t *testing.T) {
	f := activeDispatchStore(t, 2, 1)
	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")

	stopPage := getBody(t, ac, base+"/settings?tab=scans&stop=1408", http.StatusOK)
	if !strings.Contains(stopPage, `action="/scans/stop"`) {
		t.Errorf("stop dialog not rendered; body: %s", stopPage)
	}
	termPage := getBody(t, ac, base+"/settings?tab=scans&terminate=1408", http.StatusOK)
	if !strings.Contains(termPage, `action="/scans/terminate"`) {
		t.Errorf("terminate dialog not rendered; body: %s", termPage)
	}
}

func TestScansDialogsHiddenFromViewer(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(1408, "standard", tick, 6, 2, 1, 3, 0, 0),
	}
	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "viewer", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans?stop=1408", http.StatusOK)
	if strings.Contains(page, `action="/scans/stop"`) {
		t.Errorf("a viewer must not see the stop dialog; body: %s", page)
	}
}

func TestScanTriggerFlashSingleConsume(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 3})
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"dns"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("trigger: status = %d, want 303", resp.StatusCode)
	}

	first := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(first, "dns scan dispatched") || !strings.Contains(first, "3 jobs fanned out") {
		t.Errorf("first load missing the trigger toast; body: %s", first)
	}
	second := getBody(t, ac, base+loc, http.StatusOK)
	if strings.Contains(second, "dns scan dispatched") {
		t.Errorf("the trigger toast re-showed on refresh (toast spam); body: %s", second)
	}
}
