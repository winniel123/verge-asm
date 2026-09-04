package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestReportDeliveryRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
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

func TestReportDeliveryRendersEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports/delivery", http.StatusOK)

	if !strings.Contains(page, `class="sh-pill on" href="/reports"`) {
		t.Errorf("reports nav pill not marked active; body: %s", page)
	}
	for _, want := range []string{
		`href="/reports"`,
		"Report delivery",
		"No delivery yet",
		"This schedule has not delivered a report",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("report-artifact page missing %q; body: %s", want, page)
		}
	}

	if strings.Contains(page, "acmecorp") {
		t.Errorf("empty delivery must not render sample data; body: %s", page)
	}

	low := strings.ToLower(page)
	if strings.Contains(low, "time to resolve") {
		t.Errorf("no resolve metric may appear — signals are withdrawn, not resolved; body: %s", page)
	}
}

func TestReportDeliveryPageWiresPDFButton(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports/delivery", http.StatusOK)

	if !strings.Contains(page, `href="/reports/delivery/pdf"`) {
		t.Errorf("Download PDF is not wired to the download route; body: %s", page)
	}
	if strings.Contains(page, "Export is not wired yet") {
		t.Errorf("Download PDF must no longer render the disabled not-wired control; body: %s", page)
	}
}

func TestReportDeliveryPDFRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
	resp, err := c.Get(base + "/reports/delivery/pdf")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("anonymous GET /reports/delivery/pdf status = %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("anonymous GET /reports/delivery/pdf location = %q, want /login", loc)
	}
}

func TestReportDeliveryPDFServesDocument(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/reports/delivery/pdf")
	if err != nil {
		t.Fatal(err)
	}
	b := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /reports/delivery/pdf status = %d, want 200 (body: %.80q)", resp.StatusCode, b)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, `attachment; filename="report-delivery.pdf"`) {
		t.Errorf("Content-Disposition = %q, want attachment report-delivery.pdf", cd)
	}
	if !strings.HasPrefix(b, "%PDF-") {
		t.Errorf("body is not a PDF (no %%PDF- header); first bytes: %.8q", b)
	}
}
