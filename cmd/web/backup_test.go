package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// knownBusinessTables is the full set of business schema tables (every CREATE TABLE in
// db/migrations, excluding goose's own ledger). The backup allowlist and the documented
// denylist must partition it exactly — a new table added to the schema without being
// classified as estate/config (dumped) or excluded (with a reason) fails this test, so a
// future secret table is never silently swept into the archive nor silently dropped.
var knownBusinessTables = []string{
	"account", "admitted_name", "annotation", "batch", "channel", "cold_scan_scope",
	"ct_throttle", "delivery", "dispatch", "exclusion", "heartbeat", "instance_config",
	"integration_state", "invite", "message", "message_read", "observation",
	"password_reset", "personal_token", "proposal", "proposer_lookup", "queue_job",
	"recovery_code", "report_delivery", "report_notification", "report_schedule",
	"retention_settings", "scan", "seed", "seed_withdrawal", "session", "signal_instance", "source_state",
	"span", "sso_identity", "sso_provider", "transcript", "vantage",
	"verge_core_frequency_edit", "zone_file",
}

// TestBackupTablesPartitionSchema proves the allowlist and denylist together cover every
// business table exactly once — no table is both dumped and excluded, and none is
// unclassified.
func TestBackupTablesPartitionSchema(t *testing.T) {
	seen := map[string]int{}
	for _, tbl := range backupTables {
		seen[tbl]++
	}
	for tbl := range backupExcluded {
		seen[tbl]++
	}
	// Every known table is classified exactly once.
	for _, tbl := range knownBusinessTables {
		switch seen[tbl] {
		case 0:
			t.Errorf("table %q is neither in the backup allowlist nor the documented exclusions — classify it", tbl)
		case 1:
			// good
		default:
			t.Errorf("table %q is classified more than once (allowlist and/or exclusions)", tbl)
		}
	}
	// Nothing is classified that is not a known table (catches typos in either list).
	known := map[string]bool{}
	for _, tbl := range knownBusinessTables {
		known[tbl] = true
	}
	for tbl := range seen {
		if !known[tbl] {
			t.Errorf("classified table %q is not a known business table (typo?)", tbl)
		}
	}
}

// TestBackupExcludesSessionAndAuthFlowTables locks in the secret-free contract: the live
// session table and the short-lived one-time auth-flow token tables are never dumped,
// while the durable operator config (accounts, the singleton) that a restore needs IS
// dumped.
func TestBackupExcludesSessionAndAuthFlowTables(t *testing.T) {
	inAllowlist := map[string]bool{}
	for _, tbl := range backupTables {
		inAllowlist[tbl] = true
	}
	// The "read-only DB leak -> live admin sessions" surface and the one-time auth-flow
	// token tables must be excluded.
	for _, tbl := range []string{"session", "password_reset", "recovery_code", "invite"} {
		if inAllowlist[tbl] {
			t.Errorf("secret/session table %q must NOT be in the backup allowlist", tbl)
		}
		if _, ok := backupExcluded[tbl]; !ok {
			t.Errorf("table %q must be in the documented exclusions with a reason", tbl)
		}
	}
	// A usable restore needs the operator roster and the config singleton.
	for _, tbl := range []string{"account", "instance_config"} {
		if !inAllowlist[tbl] {
			t.Errorf("config table %q must be in the backup allowlist (restore needs it)", tbl)
		}
	}
}

// TestBackupArchiveWellFormed builds an archive with the pure NDJSON writers and re-parses
// it, proving the manifest carries the schema version and the ordered table list, the
// per-table markers appear, and row jsonb round-trips verbatim.
func TestBackupArchiveWellFormed(t *testing.T) {
	var buf bytes.Buffer
	const schemaVersion = int64(23000)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

	if err := writeBackupManifest(&buf, schemaVersion, now); err != nil {
		t.Fatal(err)
	}
	if err := writeBackupTableHeader(&buf, "seed"); err != nil {
		t.Fatal(err)
	}
	rowJSON := json.RawMessage(`{"id":1,"kind":"name","value":"example.com"}`)
	if err := writeBackupRow(&buf, "seed", rowJSON); err != nil {
		t.Fatal(err)
	}

	sc := bufio.NewScanner(&buf)

	// Line 1: manifest.
	if !sc.Scan() {
		t.Fatal("no manifest line")
	}
	var man backupManifest
	if err := json.Unmarshal(sc.Bytes(), &man); err != nil {
		t.Fatalf("manifest not valid JSON: %v", err)
	}
	if man.Type != "manifest" || man.Format != backupFormat || man.Version != backupFormatVersion {
		t.Errorf("manifest header wrong: %+v", man)
	}
	if man.SchemaVersion != schemaVersion {
		t.Errorf("manifest schema_version = %d, want %d", man.SchemaVersion, schemaVersion)
	}
	if len(man.Tables) != len(backupTables) || man.Tables[0] != backupTables[0] {
		t.Errorf("manifest tables list does not mirror backupTables")
	}

	// Line 2: table marker.
	if !sc.Scan() {
		t.Fatal("no table line")
	}
	var tl backupTableLine
	if err := json.Unmarshal(sc.Bytes(), &tl); err != nil {
		t.Fatalf("table line not valid JSON: %v", err)
	}
	if tl.Type != "table" || tl.Name != "seed" {
		t.Errorf("table line wrong: %+v", tl)
	}

	// Line 3: row, with the jsonb embedded verbatim.
	if !sc.Scan() {
		t.Fatal("no row line")
	}
	var rl backupRowLine
	if err := json.Unmarshal(sc.Bytes(), &rl); err != nil {
		t.Fatalf("row line not valid JSON: %v", err)
	}
	if rl.Type != "row" || rl.Table != "seed" {
		t.Errorf("row line wrong: type=%q table=%q", rl.Type, rl.Table)
	}
	var got, want map[string]any
	_ = json.Unmarshal(rl.Data, &got)
	_ = json.Unmarshal(rowJSON, &want)
	if got["value"] != want["value"] || got["kind"] != want["kind"] {
		t.Errorf("row data did not round-trip: got %v", got)
	}

	if sc.Scan() {
		t.Errorf("unexpected trailing line: %q", sc.Text())
	}
}

// TestBackupRedactsChannelAndSSOSecrets proves the data-only archive carries NO cleartext
// webhook/OAuth secret (#739, ADR-0124 / ADR-0053). A channel row and an sso_provider row
// with plaintext secrets are run through the export path (redactBackupRow, the same call
// dumpBackupTable makes) and the emitted row line must (a) contain the redacted column as
// JSON null, (b) not contain the plaintext secret anywhere, while (c) leaving every other
// column intact. A control table (channel-less) round-trips verbatim.
func TestBackupRedactsChannelAndSSOSecrets(t *testing.T) {
	const (
		webhookSecret = "whsec_super_secret_hmac_key_1234567890"
		clientSecret  = "oauth_client_secret_abcdef_confidential"
	)
	cases := []struct {
		table       string
		row         string
		redactedCol string
		plaintext   string
		keepCol     string // a column that must survive verbatim
		keepVal     string
	}{
		{
			table:       "channel",
			row:         `{"id":7,"name":"ops","secret":"` + webhookSecret + `","kind":"webhook","created_by":3}`,
			redactedCol: "secret",
			plaintext:   webhookSecret,
			keepCol:     "name",
			keepVal:     "ops",
		},
		{
			table:       "sso_provider",
			row:         `{"id":2,"client_id":"public-client-id","client_secret":"` + clientSecret + `","issuer":"https://idp.example"}`,
			redactedCol: "client_secret",
			plaintext:   clientSecret,
			keepCol:     "client_id",
			keepVal:     "public-client-id",
		},
	}
	for _, tc := range cases {
		t.Run(tc.table, func(t *testing.T) {
			redacted, err := redactBackupRow(tc.table, []byte(tc.row))
			if err != nil {
				t.Fatalf("redactBackupRow(%s): %v", tc.table, err)
			}
			// The whole emitted NDJSON row line must not contain the plaintext secret.
			var buf bytes.Buffer
			if err := writeBackupRow(&buf, tc.table, redacted); err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(buf.Bytes(), []byte(tc.plaintext)) {
				t.Fatalf("archive row for %s carries the cleartext secret: %s", tc.table, buf.String())
			}
			// The redacted column must be present and JSON null (not a bogus sentinel that a
			// restore would write as if it were the real secret).
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(redacted, &obj); err != nil {
				t.Fatalf("redacted row not valid JSON: %v", err)
			}
			raw, ok := obj[tc.redactedCol]
			if !ok {
				t.Fatalf("redacted column %q missing from row", tc.redactedCol)
			}
			if string(raw) != "null" {
				t.Errorf("column %q = %s, want JSON null", tc.redactedCol, raw)
			}
			// Every other column survives verbatim.
			var kept string
			if err := json.Unmarshal(obj[tc.keepCol], &kept); err != nil {
				t.Fatalf("kept column %q not readable: %v", tc.keepCol, err)
			}
			if kept != tc.keepVal {
				t.Errorf("column %q = %q, want %q (non-secret columns must be intact)", tc.keepCol, kept, tc.keepVal)
			}
		})
	}

	// A table with no redacted columns round-trips byte-for-byte.
	orig := []byte(`{"id":1,"kind":"name","value":"example.com"}`)
	got, err := redactBackupRow("seed", orig)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, orig) {
		t.Errorf("non-secret table was rewritten: got %s, want %s", got, orig)
	}

	// The redaction map covers exactly the two credential-bearing config tables, and every
	// table it names is in the dump allowlist (a redaction that never runs is a silent gap).
	if len(backupRedactedColumns) != 2 {
		t.Errorf("backupRedactedColumns should cover channel + sso_provider only, got %v", backupRedactedColumns)
	}
	for tbl := range backupRedactedColumns {
		if !backupAllowed(tbl) {
			t.Errorf("redacted table %q is not in the backup allowlist — its redaction never runs", tbl)
		}
	}
}

// TestBackupAdminGated proves POST /settings/backup is admin-only: anonymous is bounced to
// login, a viewer is 403, and an admin passes the gate (reaching the pool-less dev guard,
// 503 — never 403). The full stream is a Postgres round-trip covered separately.
func TestBackupAdminGated(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	// Anonymous -> redirect to /login.
	anon := newClient(t)
	resp := postForm(t, anon, base+"/settings/backup", url.Values{})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anonymous backup: status=%d loc=%q, want 303 -> /login", resp.StatusCode, resp.Header.Get("Location"))
	}

	// Viewer -> 403.
	vc := login(t, base, "viewer", "hunter2hunter2")
	resp = postForm(t, vc, base+"/settings/backup", url.Values{})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer backup: status=%d, want 403", resp.StatusCode)
	}
	if f.instanceConfig.LastBackupAt.Valid {
		t.Errorf("a denied backup still recorded a last-backup timestamp")
	}

	// Admin -> passes the gate; with no pool wired in tests the handler answers 503, which
	// proves the admin was NOT refused by requireAdmin (that would be 403).
	ac := login(t, base, "admin", "hunter2hunter2")
	resp = postForm(t, ac, base+"/settings/backup", url.Values{})
	resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("admin backup was refused by requireAdmin (403); admin must pass the gate")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("admin backup (no pool): status=%d, want 503", resp.StatusCode)
	}
}
