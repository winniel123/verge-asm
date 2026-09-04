package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestOnboardingRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")

	c := newClient(t)
	resp, err := c.Get(base + "/onboarding")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("anon /onboarding: status = %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("anon /onboarding redirect = %q, want /login", loc)
	}
}

func stepFollow(t *testing.T, c *http.Client, base string, form url.Values) string {
	// Stepping mutates nothing, so the step URL stays bookmarkable and viewer-safe.
	t.Helper()
	resp := postForm(t, c, base+"/onboarding", form)
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("stepping POST status = %d, want 303 (PRG redirect to the step GET)", resp.StatusCode)
	}
	if !strings.HasPrefix(loc, "/onboarding?") {
		t.Fatalf("stepping should redirect to the /onboarding step GET; location: %q", loc)
	}
	return getBody(t, c, base+loc, http.StatusOK)
}

func TestOnboardingStepsAdvance(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 2})
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/onboarding", http.StatusOK)
	if !strings.Contains(page, "Set up this workspace") || !strings.Contains(page, `name="seedsadd"`) {
		t.Fatalf("step 0 did not render the seeds step; body: %s", page)
	}

	held := stepFollow(t, ac, base, url.Values{"step": {"0"}, "action": {"next"}})
	if !strings.Contains(held, `name="seedsadd"`) {
		t.Fatalf("empty seeds should hold on step 0; body: %s", held)
	}
	if !strings.Contains(held, `id="ob-next" disabled`) {
		t.Fatalf("empty seeds should render Next disabled server-side (.StepValid floor); body: %s", held)
	}

	cadence := stepFollow(t, ac, base, url.Values{"step": {"0"}, "action": {"next"}, "seedsadd": {"acmecorp.io"}})
	if !strings.Contains(cadence, "Scan profile") || !strings.Contains(cadence, "Passive only") {
		t.Fatalf("did not advance to cadence step; body: %s", cadence)
	}
	if !strings.Contains(cadence, `name="seeds" value="acmecorp.io"`) {
		t.Fatalf("committed seed not carried forward; body: %s", cadence)
	}

	channel := stepFollow(t, ac, base, url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"},
	})
	if !strings.Contains(channel, "Delivery URL (optional)") {
		t.Fatalf("did not advance to channel step; body: %s", channel)
	}

	review := stepFollow(t, ac, base, url.Values{
		"step": {"2"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"},
		"channel": {"https://ops.example/hook"},
	})
	if !strings.Contains(review, "Start first scan") {
		t.Fatalf("did not advance to review step; body: %s", review)
	}
}

func TestOnboardingCustomCadenceGate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 2})
	ac := login(t, base, "admin", "hunter2hunter2")

	held := stepFollow(t, ac, base, url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Custom…"},
	})
	if !strings.Contains(held, "Scan profile") || !strings.Contains(held, `name="cron"`) {
		t.Fatalf("custom cadence without cron should hold on cadence and reveal cron; body: %s", held)
	}

	adv := stepFollow(t, ac, base, url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Custom…"}, "cron": {"0 8 * * 1"},
	})
	if !strings.Contains(adv, "Delivery URL (optional)") {
		t.Fatalf("custom cadence with cron should advance to channel; body: %s", adv)
	}
}

func TestOnboardingReviewReflectsInputs(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 2})
	ac := login(t, base, "admin", "hunter2hunter2")

	review := stepFollow(t, ac, base, url.Values{
		"step": {"2"}, "action": {"next"},
		"seeds": {"acmecorp.io,203.0.113.0/24"}, "profile": {"passive"}, "cad": {"Weekly · mon 09:00"},
		"channel": {"https://ops.example/hook"},
	})
	for _, want := range []string{"acmecorp.io, 203.0.113.0/24", "passive", "weekly · mon 09:00", "https://ops.example/hook"} {
		if !strings.Contains(review, want) {
			t.Errorf("review missing real input %q; body: %s", want, review)
		}
	}
	if !strings.Contains(review, `name="kind" value="dns"`) {
		t.Errorf("passive profile should map the first scan to dns; body: %s", review)
	}
}

func TestOnboardingFinishEnqueuesScan(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 4}
	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/onboarding/finish", url.Values{
		"seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"}, "kind": {"hot"},
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("finish: status = %d, want 303", resp.StatusCode)
	}
	if loc != "/scans" {
		t.Fatalf("finish redirect = %q, want the monitor at /scans", loc)
	}
	if len(trig.calls) != 1 || trig.calls[0] != "hot" {
		t.Fatalf("finish dispatcher calls = %v, want one hot fan-out", trig.calls)
	}
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "hot scan dispatched") || !strings.Contains(page, "4 jobs fanned out") {
		t.Errorf("finish receipt missing; body: %s", page)
	}
}

func TestOnboardingFinishViewerRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 4}
	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, ac, base+"/onboarding/finish", url.Values{"profile": {"standard"}, "kind": {"hot"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer finish: status = %d, want 403", resp.StatusCode)
	}
	if len(trig.calls) != 0 {
		t.Fatalf("a viewer reached the dispatcher: %v", trig.calls)
	}
}
