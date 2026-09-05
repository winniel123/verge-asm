package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// The export reads a business-table allowlist by rule, so a new table is not swept in (ADR-0124).

var backupTables = []string{ // FK-parent-first, so a naive in-order restore is close to load-safe.
	"account",
	"instance_config",
	"retention_settings",
	"seed",
	"exclusion",
	"cold_scan_scope",
	"zone_file",
	"admitted_name",
	"source_state",
	"vantage",
	"scan",
	"dispatch",
	"batch",
	"observation",
	"span",
	// It carries the batch FK, and dropping it leaves a timeline nothing closes (ADR-0134).
	"seed_withdrawal",
	"verge_core_frequency_edit",
	"signal_instance",
	"annotation",
	"proposer_lookup",
	"proposal",
	"integration_state",
	"channel",
	"delivery",
	"message",
	"message_read",
	"report_schedule",
	"report_delivery",
	"report_notification",
	"personal_token",
	"sso_provider",
	"sso_identity",
}

// The two lists partition the schema, so a new table must be classified (ADR-0161 §1, #1367).
// gosec's credential heuristic fires on the "password" and "token" substrings in this literal.

// #nosec G101 -- keys are database table names (e.g. "password_reset") paired with prose
var backupExcluded = map[string]string{
	"session":        "live login sessions — the ADR-0053 §4.2 'DB leak → live admin sessions' surface; they lapse on restore and are invalid under the regenerated session key",
	"password_reset": "single-use, short-TTL reset-token hashes — expired and meaningless off-host, a needless credential surface with no restore value",
	"recovery_code":  "MFA recovery-code hashes — auth-bypass material with no cross-host restore value; a restored instance's operators re-enroll",
	"invite":         "pending single-use account-creation invite tokens — short-lived, no durable configuration value",
	"heartbeat":      "worker liveness ping — ephemeral runtime state, re-derived immediately after restore",
	"ct_throttle":    "per-source CT fetch rate-limit buckets — ephemeral runtime state",
	"queue_job":      "in-flight scan queue — transient work-in-progress; stale 'running' rows would be phantom after an overwrite restore",
	"transcript":     "raw job output (raw-job-output spec §5.4) — bounded-retention verbatim debug bytes, AEAD ciphertext under a volume key the archive must not carry; excluding it keeps ADR-0124's 'a backup carries data and no credential' invariant and avoids shipping ciphertext a fresh restore cannot decrypt",
}

const (
	backupFormat        = "verge-backup"
	backupFormatVersion = 1
)

// These two are reversible cleartext and the other write-only columns are not (ADR-0160 §1, #1367).

var backupRedactedColumns = map[string][]string{
	"channel":      {"secret"},
	"sso_provider": {"client_secret"},
}

func redactBackupRow(table string, data []byte) ([]byte, error) {
	cols := backupRedactedColumns[table]
	if len(cols) == 0 {
		return data, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	for _, c := range cols {
		if _, ok := obj[c]; ok {
			obj[c] = json.RawMessage("null")
		}
	}
	// The re-marshal may reorder keys, which jsonb_populate_record on restore ignores.
	return json.Marshal(obj)
}

type backupManifest struct {
	Type          string   `json:"type"`
	Format        string   `json:"format"`
	Version       int      `json:"version"`
	SchemaVersion int64    `json:"schema_version"`
	CreatedAt     string   `json:"created_at"`
	Tables        []string `json:"tables"`
}

type backupTableLine struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type backupRowLine struct {
	Type  string          `json:"type"`
	Table string          `json:"table"`
	Data  json.RawMessage `json:"data"`
}

func (s *server) backupDownload(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	if s.pool == nil {
		http.Error(w, "backup is unavailable in this mode", http.StatusServiceUnavailable)
		return
	}

	var schemaVersion int64
	// The archive records its schema version so a restore can refuse a mismatch first (ADR-0124).
	if err := s.pool.QueryRow(ctx, "SELECT COALESCE(max(version_id), 0) FROM goose_db_version").Scan(&schemaVersion); err != nil {
		s.serverError(w, "backup schema version", err)
		return
	}

	filename := "verge-backup-" + s.now().UTC().Format("20060102-150405") + ".ndjson"
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	cw := &countingWriter{w: w}
	if err := s.streamBackup(ctx, cw, schemaVersion, s.now()); err != nil {
		// The status and headers are already sent, so this cannot become an error status.
		log.Printf("web: backup: stream: %v", err)
		return
	}

	if err := s.store.SetLastBackup(ctx, pgtype.Int8{Int64: cw.n, Valid: true}); err != nil {
		log.Printf("web: backup: record last backup: %v", err)
	}
}

func (s *server) streamBackup(ctx context.Context, w io.Writer, schemaVersion int64, now time.Time) error {
	if err := writeBackupManifest(w, schemaVersion, now); err != nil {
		return err
	}
	for _, table := range backupTables {
		if err := writeBackupTableHeader(w, table); err != nil {
			return err
		}
		if err := s.dumpBackupTable(ctx, w, table); err != nil {
			return fmt.Errorf("dump %s: %w", table, err)
		}
	}
	return nil
}

func (s *server) dumpBackupTable(ctx context.Context, w io.Writer, table string) error {
	// The table comes from the backupTables allowlist and never from request input.
	rows, err := s.pool.Query(ctx, `SELECT to_jsonb(t) FROM "`+table+`" t`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}
		redacted, err := redactBackupRow(table, data)
		if err != nil {
			return err
		}
		if err := writeBackupRow(w, table, redacted); err != nil {
			return err
		}
	}
	return rows.Err()
}

func writeBackupManifest(w io.Writer, schemaVersion int64, now time.Time) error {
	return writeJSONLine(w, backupManifest{
		Type:          "manifest",
		Format:        backupFormat,
		Version:       backupFormatVersion,
		SchemaVersion: schemaVersion,
		CreatedAt:     now.UTC().Format(time.RFC3339),
		Tables:        backupTables,
	})
}

func writeBackupTableHeader(w io.Writer, table string) error {
	return writeJSONLine(w, backupTableLine{Type: "table", Name: table})
}

func writeBackupRow(w io.Writer, table string, data json.RawMessage) error {
	return writeJSONLine(w, backupRowLine{Type: "row", Table: table, Data: data})
}

func writeJSONLine(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
