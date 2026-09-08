package main

import (
	"bytes"
	"context"
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
		`{"subject_key":"a.example.com","closed_at":null}`,
		`{"subject_key":"c.example.com","closed_at":"2026-08-01T00:00:00Z"}`,
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

func TestPreflightArchiveRejectsUnknownTable(t *testing.T) {
	manifest := `{"type":"manifest","format":"` + backupFormat + `","version":` +
		strconv.Itoa(backupFormatVersion) + `,"schema_version":23000,"created_at":"2026-08-26T12:00:00Z","tables":["pg_catalog_pg_proc"]}` + "\n"
	if _, err := preflightArchive(strings.NewReader(manifest)); err != errRestoreUnknownTbl {
		t.Fatalf("preflightArchive on unknown table: err = %v, want errRestoreUnknownTbl", err)
	}
}

func TestApplyRestoreRefusesForgedManifestTable(t *testing.T) {
	manifest := `{"type":"manifest","format":"` + backupFormat + `","version":` +
		strconv.Itoa(backupFormatVersion) + `,"schema_version":23000,"created_at":"2026-08-26T12:00:00Z","tables":["goose_db_version"]}` + "\n"

	// A dropped gate panics on the nil pool rather than returning (ADR-0174 §6).
	if err := (&server{}).applyRestore(context.Background(), []byte(manifest)); err != errRestoreUnknownTbl {
		t.Fatalf("applyRestore on a forged manifest table: err = %v, want errRestoreUnknownTbl", err)
	}
}

func TestApplyRestoreRefusesForgedRowTable(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildTestArchive(t, 23000, []string{`{"subject_key":"a.example.com","closed_at":null}`}))
	if err := writeBackupRow(&buf, "session", json.RawMessage(`{"id":1}`)); err != nil {
		t.Fatal(err)
	}

	// Preflight reads no non-span row line, so only the row gate refuses this (ADR-0174 §3).
	if _, err := preflightArchive(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("preflightArchive: %v, want the manifest gate to pass this archive", err)
	}
	if err := (&server{}).applyRestore(context.Background(), buf.Bytes()); err != errRestoreUnknownTbl {
		t.Fatalf("applyRestore on a forged row table: err = %v, want errRestoreUnknownTbl", err)
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

func pendingSettingsFlash(t *testing.T, srv *server) settingsForms {
	// Driving the landing GET would make the assertion depend on the whole Settings render.
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

func TestRestoreApplyRequiresConfirmWord(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	srv := newServer(f, testKey, "", fixedClock())
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()
	ac := login(t, ts.URL, "admin", "hunter2hunter2")

	keyBefore := string(srv.key)

	resp := postForm(t, ac, ts.URL+"/settings/restore", url.Values{"confirm": {"yes"}})
	resp.Body.Close()
	if loc := resp.Header.Get("Location"); loc != "/settings?tab=instance" {
		t.Fatalf("wrong confirm: Location = %q, want /settings?tab=instance", loc)
	}
	if got := pendingSettingsFlash(t, srv).restoreError; got != restoreErrorMessage("confirm") {
		t.Fatalf("wrong confirm: flashed restoreError = %q, want the confirm line", got)
	}

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

func TestRotateSessionKeyLapsesSessions(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	srv := newServer(f, testKey, "", fixedClock())
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()
	ac := login(t, ts.URL, "admin", "hunter2hunter2")

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

	resp, err = ac.Get(ts.URL + "/profile")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("after rotate: /profile status=%d loc=%q, want 303 -> /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestLoginAfterRestoreLandsUnenrolled(t *testing.T) {
	f := newFakeStore()
	acct := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	enableTOTP(t, f, acct.ID)

	srv := newServer(f, testKey, "", fixedClock())
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	before := newClient(t)
	resp := postForm(t, before, ts.URL+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}})
	resp.Body.Close()
	if !hasCookie(before, ts.URL, pendingCookie) {
		t.Fatal("test setup: the account was not TOTP-enrolled before the restore")
	}

	f.acctMu.Lock()
	live := f.accounts[acct.ID]
	f.acctMu.Unlock()
	sealed := live.TotpSecret.String
	if sealed == "" {
		t.Fatal("test setup: no sealed TOTP secret to archive")
	}

	rowJSON, err := json.Marshal(live)
	if err != nil {
		t.Fatal(err)
	}
	archived, err := redactBackupRow("account", rowJSON)
	if err != nil {
		t.Fatalf("redactBackupRow(account): %v", err)
	}
	if bytes.Contains(archived, []byte(sealed)) {
		t.Fatalf("the archive carries the TOTP ciphertext: %s", archived)
	}

	if err := srv.rotateSessionKey(); err != nil {
		t.Fatalf("rotateSessionKey: %v", err)
	}
	var replayed db.Account
	if err := json.Unmarshal(archived, &replayed); err != nil {
		t.Fatalf("archived account row does not replay: %v", err)
	}
	f.acctMu.Lock()
	f.accounts[acct.ID] = replayed
	f.acctMu.Unlock()

	// A restore rotates the key that sealed the archived secret, so the login is the test (#1419).
	after := newClient(t)
	resp = postForm(t, after, ts.URL+"/login", url.Values{"username": {"admin"}, "password": {"hunter2hunter2"}})
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusInternalServerError {
		t.Fatal("login after a restore answered 500 — the restored ciphertext does not open under the rotated key")
	}
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login after a restore: status = %d, want 303 (the operator lands unenrolled)", resp.StatusCode)
	}
	if hasCookie(after, ts.URL, pendingCookie) {
		t.Fatal("login after a restore still owes a second factor the instance cannot verify")
	}
	if !hasCookie(after, ts.URL, sessionCookie) {
		t.Fatal("login after a restore minted no session")
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

func TestRestoreStageExpiresAfterTTL(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	srv := newServer(f, testKey, "", func() time.Time { return now })
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()
	ac := login(t, ts.URL, "admin", "hunter2hunter2")

	srv.stashRestore(admin.ID, &restoreStaging{file: "backup.ndjson", archive: []byte("{}")})
	now = now.Add(restoreStageTTL - time.Second)
	if srv.stagedRestore(admin.ID) == nil {
		t.Fatal("a stage inside the TTL was dropped")
	}

	now = now.Add(2 * time.Second)
	resp := postForm(t, ac, ts.URL+"/settings/restore", url.Values{"confirm": {"restore"}})
	resp.Body.Close()
	if got := pendingSettingsFlash(t, srv).restoreError; got != restoreErrorMessage("expired") {
		t.Fatalf("apply after the TTL: flashed restoreError = %q, want the expired line", got)
	}
	if srv.stagedRestore(admin.ID) != nil {
		t.Fatal("an expired stage is still held")
	}
}

func TestRestorePreflightEvictsThePreviousStage(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	srv := newServer(f, testKey, "", fixedClock())
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()
	ac := login(t, ts.URL, "admin", "hunter2hunter2")

	srv.stashRestore(admin.ID, &restoreStaging{file: "old.ndjson", archive: []byte("{}")})
	resp := postForm(t, ac, ts.URL+"/settings/restore/preflight", url.Values{})
	resp.Body.Close()
	if srv.stagedRestore(admin.ID) != nil {
		t.Fatal("a new pre-flight left the previous archive staged")
	}
}
