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

type backupStore interface {
	SetLastBackup(ctx context.Context, lastBackupSize pgtype.Int8) error
}

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
	// It carries the batch FK, and the fan-out verdict reads its newest row per address (#1649).
	"edge_fanout_observation",
	// Immutable CT inputs by fingerprint: a restore with no row cannot verify a chain (#1649).
	"certificate_material",
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
	// An Act holds no secret, and excluding it would erase the history on restore (spec §5.3).
	"act",
}

// The two lists partition the schema, so a new table must be classified (ADR-0161 §1, #1367).

// gosec's credential heuristic fires on the "password" and "token" substrings in this literal.

// #nosec G101 -- keys are database table names (e.g. "password_reset") paired with prose
var backupExcluded = map[string]string{
	"session":               "live login sessions — the ADR-0053 'DB leak → live admin sessions' surface; they lapse on restore and are invalid under the regenerated session key",
	"password_reset":        "single-use, short-TTL reset-token hashes — expired and meaningless off-host, a needless credential surface with no restore value",
	"recovery_code":         "MFA recovery-code hashes — auth-bypass material with no cross-host restore value; a restored instance's operators re-enroll",
	"invite":                "pending single-use account-creation invite tokens — short-lived, no durable configuration value",
	"heartbeat":             "worker liveness ping — ephemeral runtime state, re-derived immediately after restore",
	"ct_throttle":           "per-source CT fetch rate-limit buckets — ephemeral runtime state",
	"ct_log_cursor":         "the ct-tail's per-log forward cursor — runtime state the next poll re-seeds from the log's own head; carrying it would replay a stale window",
	"ct_reliability_sample": "rolling samples of this install's own bulk-CT queries — runtime measurement of our requests, re-accrued after restore",
	"source_health":         "what this install's last attempt to a source did (ADR-0223 §1) — an Operational record of our own requests, re-derived on the next query",
	"queue_job":             "in-flight scan queue — transient work-in-progress; stale 'running' rows would be phantom after an overwrite restore",
	"transcript":            "raw job output (raw-job-output spec §5.4) — bounded-retention verbatim debug bytes, AEAD ciphertext under a volume key the archive must not carry; excluding it keeps ADR-0124's 'a backup carries data and no credential' invariant and avoids shipping ciphertext a fresh restore cannot decrypt",
}

const (
	backupFormat        = "verge-backup"
	backupFormatVersion = 1
)

// Reversible cleartext is redacted and a write-only hash is not (ADR-0160 §1, #1367).

var backupRedactedColumns = map[string]map[string]json.RawMessage{
	// A restore rotates the key that opens the TOTP secret, so the archive drops it (#1419).
	"account": {
		"totp_secret": json.RawMessage("null"),
		// The column is NOT NULL, and a true here strands the login at a factor nothing verifies.
		"totp_enabled": json.RawMessage("false"),
	},
	"channel":      {"secret": json.RawMessage("null")},
	"sso_provider": {"client_secret": json.RawMessage("null")},
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
	for c, v := range cols {
		if _, ok := obj[c]; ok {
			obj[c] = v
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

	if err := s.backupStore.SetLastBackup(ctx, pgtype.Int8{Int64: cw.n, Valid: true}); err != nil {
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
	// The table comes from the allowlist literal, never from request input (ADR-0174, #1363).
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
