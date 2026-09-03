package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

// buildTestArchive assembles a B3-format archive with the real backup writers (round-trip
// against cmd/web/backup.go) at a chosen schema version, carrying the given span rows so a
// pre-flight has real subjects to count.
func buildTestArchive(t *testing.T, schemaVersion int64, spanRows []string) []byte {
	t.Helper()
	var buf bytes.Buffer
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	if err := writeBackupManifest(&buf, schemaVersion, now); err != nil {
		t.Fatal(err)
	}
	if err := writeBackupTableHeader(&buf, "span"); err != nil {
		t.Fatal(err)
	}
	for _, r := range spanRows {
		if err := writeBackupRow(&buf, "span", json.RawMessage(r)); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

func TestPreflightArchiveRoundTrip(t *testing.T) {
	archive := buildTestArchive(t, 23000, []string{
		`{"subject_key":"a.example.com","closed_at":null}`,
		`{"subject_key":"b.example.com","closed_at":null}`,
		`{"subject_key":"a.example.com","closed_at":null}`,                   // duplicate open key — one subject
		`{"subject_key":"c.example.com","closed_at":"2026-08-01T00:00:00Z"}`, // closed — not a current subject
	})

	pf, err := preflightArchive(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("preflightArchive: %v", err)
	}
	if pf.SchemaVersion != 23000 {
		t.Errorf("SchemaVersion = %d, want 23000", pf.SchemaVersion)
	}
	if pf.Subjects != 2 {
		t.Errorf("Subjects = %d, want 2 (distinct open subject_keys)", pf.Subjects)
	}
	if len(pf.Tables) != len(backupTables) || pf.Tables[0] != backupTables[0] {
		t.Errorf("Tables did not mirror the backup allowlist")
	}
}

func TestPreflightArchiveRejectsGarbage(t *testing.T) {
	if _, err := preflightArchive(strings.NewReader("this is not a verge archive\n")); err == nil {
		t.Fatal("preflightArchive accepted garbage; want an error")
	}
}

// TestPreflightArchiveRejectsUnknownTable proves a hand-forged manifest naming a table
// outside B3's allowlist is refused, so a crafted archive cannot steer the apply's SQL at an
// arbitrary identifier.
func TestPreflightArchiveRejectsUnknownTable(t *testing.T) {
	manifest := `{"type":"manifest","format":"` + backupFormat + `","version":` +
		strconv.Itoa(backupFormatVersion) + `,"schema_version":23000,"created_at":"2026-08-26T12:00:00Z","tables":["pg_catalog_pg_proc"]}` + "\n"
	if _, err := preflightArchive(strings.NewReader(manifest)); err != errRestoreUnknownTbl {
		t.Fatalf("preflightArchive on unknown table: err = %v, want errRestoreUnknownTbl", err)
	}
}

func TestPreflightArchiveSchemaValueDrivesRefusal(t *testing.T) {
	archive := buildTestArchive(t, 99999, nil)
	pf, err := preflightArchive(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("preflightArchive: %v", err)
	}
	if pf.SchemaVersion != 99999 {
		t.Fatalf("SchemaVersion = %d, want 99999", pf.SchemaVersion)
	}
	const running = int64(23000)
	if pf.SchemaVersion == running {
		t.Fatal("test setup: archive schema unexpectedly matched running")
	}
	if restoreErrorMessage("schema") == "" {
		t.Fatal("schema mismatch has no operator-facing message")
	}
}

// TestRestorePreflightRefusesInFlightScan proves the pre-flight refuses while a scan is in
// flight — before the upload is even parsed — and stages nothing (WORK-ORDER §#391 edge).
func TestRestorePreflightRefusesInFlightScan(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.dispatchProgress = []db.ListDispatchProgressRow{{DispatchID: 1, ScanKind: "standard", Ready: 3}}

	srv := newServer(f, testKey, "", fixedClock())
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	ac := login(t, ts.URL, "admin", "hunter2hunter2")
	resp := postForm(t, ac, ts.URL+"/settings/restore/preflight", url.Values{})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("preflight during scan: status = %d, want 303", resp.StatusCode)
	}
	// The refusal lands on the URL the form was submitted from — here the fallback, since
	// this post carries no `return` field — and spells nothing on it (ADR-0130 §1/§3,
	// ticket #977). The operator-facing line rides the session form flash instead.
	if loc := resp.Header.Get("Location"); loc != "/settings?tab=instance" {
		t.Fatalf("preflight during scan: Location = %q, want /settings?tab=instance", loc)
	}
	if got := pendingSettingsFlash(t, srv).restoreError; got != restoreErrorMessage("inflight") {
		t.Fatalf("flashed restoreError = %q, want the in-flight line", got)
	}
	if srv.stagedRestore(admin.ID) != nil {
		t.Fatal("a refused pre-flight still staged an archive")
	}
}

// pendingSettingsFlash returns the one settingsForms waiting in the session form flash, for
// a test that has just made a single refused settings act. It reads the store directly
// rather than driving the landing GET, so the assertion is about the refusal the handler
// stashed and not about everything else the Settings render happens to need.
func pendingSettingsFlash(t *testing.T, srv *server) settingsForms {
	t.Helper()
	srv.formFlash.mu.Lock()
	defer srv.formFlash.mu.Unlock()
	if n := len(srv.formFlash.m); n != 1 {
		t.Fatalf("form flash holds %d entries, want exactly 1", n)
	}
	for _, e := range srv.formFlash.m {
		f, ok := e.value.(settingsForms)
		if !ok {
			t.Fatalf("form flash holds %T, want settingsForms", e.value)
		}
		return f
	}
	return settingsForms{}
}

// TestRestorePreflightPassesGateWithoutPool proves that with no scan in flight an admin
// passes the gate and reaches the pool guard (503 off a pool), never a 403 — proof the
// refusal above was the in-flight check, not the admin gate.
func TestRestorePreflightPassesGateWithoutPool(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, ac, base+"/settings/restore/preflight", url.Values{})
	resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		t.Fatal("admin pre-flight was refused by requireAdmin (403)")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("admin pre-flight (no pool): status = %d, want 503", resp.StatusCode)
	}
}

func TestRestorePreflightAdminGated(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	anon := newClient(t)
	resp := postForm(t, anon, base+"/settings/restore/preflight", url.Values{})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon pre-flight: status=%d loc=%q, want 303 -> /login", resp.StatusCode, resp.Header.Get("Location"))
	}

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp = postForm(t, vc, base+"/settings/restore/preflight", url.Values{})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer pre-flight: status=%d, want 403", resp.StatusCode)
	}
}

// TestRestoreApplyRequiresConfirmWord proves the apply never applies without the typed word
// `restore` (validated server-side, never trusting the JS gate): a wrong word refuses with
// nothing touched, and a correct word with no staged archive refuses as expired. Neither
// rotates the session key.
func TestRestoreApplyRequiresConfirmWord(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	srv := newServer(f, testKey, "", fixedClock())
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()
	ac := login(t, ts.URL, "admin", "hunter2hunter2")

	keyBefore := string(srv.key)

	// Wrong confirmation word — no apply. Both refusals 303 to the same URL a success
	// would, and carry their line on the session flash rather than on the query (#977).
	resp := postForm(t, ac, ts.URL+"/settings/restore", url.Values{"confirm": {"yes"}})
	resp.Body.Close()
	if loc := resp.Header.Get("Location"); loc != "/settings?tab=instance" {
		t.Fatalf("wrong confirm: Location = %q, want /settings?tab=instance", loc)
	}
	if got := pendingSettingsFlash(t, srv).restoreError; got != restoreErrorMessage("confirm") {
		t.Fatalf("wrong confirm: flashed restoreError = %q, want the confirm line", got)
	}

	// Correct word but nothing staged — refused as expired, still no apply. The second
	// refusal replaces the first in the store, so the session never banks two callouts.
	resp = postForm(t, ac, ts.URL+"/settings/restore", url.Values{"confirm": {"restore"}})
	resp.Body.Close()
	if loc := resp.Header.Get("Location"); loc != "/settings?tab=instance" {
		t.Fatalf("no staged archive: Location = %q, want /settings?tab=instance", loc)
	}
	if got := pendingSettingsFlash(t, srv).restoreError; got != restoreErrorMessage("expired") {
		t.Fatalf("no staged archive: flashed restoreError = %q, want the expired line", got)
	}

	if string(srv.key) != keyBefore {
		t.Fatal("a refused apply rotated the session key")
	}
}

// TestRotateSessionKeyLapsesSessions proves regenerating the session key lapses every live
// session in the running process at once: an authenticated request that succeeded before the
// rotation is bounced to /login after it, because its cookie is signed under the dead key.
func TestRotateSessionKeyLapsesSessions(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	srv := newServer(f, testKey, "", fixedClock())
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()
	ac := login(t, ts.URL, "admin", "hunter2hunter2")

	// Before rotation the session resolves — /profile renders.
	resp, err := ac.Get(ts.URL + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("before rotate: /profile status = %d, want 200", resp.StatusCode)
	}

	keyBefore := string(srv.key)
	totpBefore := string(srv.totpKey)
	if err := srv.rotateSessionKey(); err != nil {
		t.Fatalf("rotateSessionKey: %v", err)
	}
	if string(srv.key) == keyBefore {
		t.Fatal("rotateSessionKey did not change the signing key")
	}
	if string(srv.totpKey) == totpBefore {
		t.Fatal("rotateSessionKey did not re-derive the TOTP sub-key")
	}

	// After rotation the same cookie no longer verifies — the session has lapsed.
	resp, err = ac.Get(ts.URL + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("after rotate: /profile status=%d loc=%q, want 303 -> /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestRestoreErrorMessages(t *testing.T) {
	for _, code := range []string{"inflight", "unreadable", "schema", "confirm", "expired", "apply"} {
		if restoreErrorMessage(code) == "" {
			t.Errorf("restoreErrorMessage(%q) = empty, want a fixed line", code)
		}
	}
	if restoreErrorMessage("../../etc/passwd") != "" {
		t.Error("an unknown restore-error code reflected text; want empty")
	}
}
