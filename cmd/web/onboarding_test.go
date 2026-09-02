package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// The onboarding wizard is gated behind login — an anonymous GET is bounced to the
// sign-in page, never rendered (AC: requires-login).
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

// stepFollow posts a wizard step and follows the #25d PRG redirect: it asserts the POST
// 303-redirects to a bookmarkable GET /onboarding?step=… URL and returns that step's rendered
// body. Stepping mutates nothing, so a viewer-safe GET renders the accumulated state.
func stepFollow(t *testing.T, c *http.Client, base string, form url.Values) string {
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

// The four steps render in order and Next advances only past a satisfied valid
// gate: the seeds step blocks Next until a seed is entered, then walks seeds ->
// cadence -> channel -> review (AC: the 4 steps render/advance, controlled).
func TestOnboardingStepsAdvance(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 2})
	ac := login(t, base, "admin", "hunter2hunter2")

	// Step 0 renders the seeds field.
	page := getBody(t, ac, base+"/onboarding", http.StatusOK)
	if !strings.Contains(page, "Set up this workspace") || !strings.Contains(page, `name="seedsadd"`) {
		t.Fatalf("step 0 did not render the seeds step; body: %s", page)
	}

	// Next on an empty seeds step does not advance — the valid gate holds it on step 0, and the
	// server-side floor renders Next disabled (.StepValid false).
	held := stepFollow(t, ac, base, url.Values{"step": {"0"}, "action": {"next"}})
	if !strings.Contains(held, `name="seedsadd"`) {
		t.Fatalf("empty seeds should hold on step 0; body: %s", held)
	}
	if !strings.Contains(held, `id="ob-next" disabled`) {
		t.Fatalf("empty seeds should render Next disabled server-side (.StepValid floor); body: %s", held)
	}

	// Enter a seed and advance to cadence — the typed seed is absorbed as a committed seed.
	cadence := stepFollow(t, ac, base, url.Values{"step": {"0"}, "action": {"next"}, "seedsadd": {"acmecorp.io"}})
	if !strings.Contains(cadence, "Scan profile") || !strings.Contains(cadence, "Passive only") {
		t.Fatalf("did not advance to cadence step; body: %s", cadence)
	}
	// The committed seed rides forward as hidden state.
	if !strings.Contains(cadence, `name="seeds" value="acmecorp.io"`) {
		t.Fatalf("committed seed not carried forward; body: %s", cadence)
	}

	// Advance cadence -> channel.
	channel := stepFollow(t, ac, base, url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"},
	})
	if !strings.Contains(channel, "Delivery URL (optional)") {
		t.Fatalf("did not advance to channel step; body: %s", channel)
	}

	// Advance channel -> review.
	review := stepFollow(t, ac, base, url.Values{
		"step": {"2"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"},
		"channel": {"https://ops.example/hook"},
	})
	if !strings.Contains(review, "Start first scan") {
		t.Fatalf("did not advance to review step; body: %s", review)
	}
}

// The custom cadence gate holds Next until a cron is supplied, mirroring the
// example's valid predicate (cad != Custom || cron non-empty).
func TestOnboardingCustomCadenceGate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 2})
	ac := login(t, base, "admin", "hunter2hunter2")

	// Custom cadence with no cron: Next holds on the cadence step and reveals the cron field.
	held := stepFollow(t, ac, base, url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Custom…"},
	})
	if !strings.Contains(held, "Scan profile") || !strings.Contains(held, `name="cron"`) {
		t.Fatalf("custom cadence without cron should hold on cadence and reveal cron; body: %s", held)
	}

	// With a cron supplied, Next advances to the channel step.
	adv := stepFollow(t, ac, base, url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Custom…"}, "cron": {"0 8 * * 1"},
	})
	if !strings.Contains(adv, "Delivery URL (optional)") {
		t.Fatalf("custom cadence with cron should advance to channel; body: %s", adv)
	}
}

// The review step summarizes the real inputs — the seeds, profile, cadence and
// channel the operator entered, not sample data (AC: review reflects real inputs).
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
	// The finish button carries the passive-profile scan kind (dns discovery).
	if !strings.Contains(review, `name="kind" value="dns"`) {
		t.Errorf("passive profile should map the first scan to dns; body: %s", review)
	}
}

// Completion enqueues a real scan through the existing trigger path: the finish
// button posts the mapped kind to /onboarding/finish, which dispatches the same
// fan-out POST /scans/trigger uses and lands on the monitor (AC: completion
// enqueues a scan, asserted via the scan-trigger path; no fabricated scan).
func TestOnboardingFinishEnqueuesScan(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 4}
	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")

	// A standard profile maps the first scan to the active hot port scan.
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
	// The receipt rides the single-consume flash, fired on the monitor's render.
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "hot scan dispatched") || !strings.Contains(page, "4 jobs fanned out") {
		t.Errorf("finish receipt missing; body: %s", page)
	}
}

// Completion is an admin act — a viewer cannot enqueue the first scan, matching the
// admin-only guardrail on the trigger it reuses (AC: admin-only enqueue).
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
