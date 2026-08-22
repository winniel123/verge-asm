package main

import (
	"net/http"
	"strings"
	"testing"
)

// The report-artifact view is behind requireLogin — an anonymous GET is bounced to
// sign-in, never served a delivery.
func TestReportDeliveryRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t) // does not follow redirects
	resp, err := c.Get(base + "/reports/delivery")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("anonymous GET /reports/delivery status = %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("anonymous GET /reports/delivery location = %q, want /login", loc)
	}
}

// With no report-delivery backend, the view renders the artifact frame with the
// design-system empty-state — the breadcrumb back to Reports, the Reports nav pill
// active, and the delivered-document empty-state — never a fabricated document.
func TestReportDeliveryRendersEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports/delivery", http.StatusOK)

	if !strings.Contains(page, `class="navpill active" href="/reports"`) {
		t.Errorf("reports nav pill not marked active; body: %s", page)
	}
	for _, want := range []string{
		`href="/reports"`,                  // breadcrumb back to Reports
		"Report delivery",                  // generic heading (no delivery to name)
		"No report has been delivered yet", // design-system empty-state
		"vg-artifact",                      // the self-contained delivered document frame
	} {
		if !strings.Contains(page, want) {
			t.Errorf("report-artifact page missing %q; body: %s", want, page)
		}
	}

	// No fabricated document: no sample org or KPI figures leak in.
	if strings.Contains(page, "acmecorp") {
		t.Errorf("empty delivery must not render sample data; body: %s", page)
	}

	// Domain hygiene: no resolve framing on a signals surface. (The delivered
	// document's own no-severity-ramp guarantee is asserted where it is produced,
	// in internal/message's RenderArtifact test; the shell stylesheet defines the
	// severity tokens on every console page, so it is not checkable here.)
	low := strings.ToLower(page)
	if strings.Contains(low, "time to resolve") {
		t.Errorf("no resolve metric may appear — signals are withdrawn, not resolved; body: %s", page)
	}
}
