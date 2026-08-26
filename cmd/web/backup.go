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

// Backup export — POST `/settings/backup` (#391, ADR-0124, B3). Admin-only, streamed,
// DATA-ONLY. It serves a Go-native logical dump of the estate + config business tables
// over the pgx pool `web` already holds, as a downloadable attachment an operator can
// carry to another host and restore (B4).
//
// SECRET-FREE BY CONSTRUCTION — ADR-0124 / ADR-0053. The archive carries NO secret that
// could turn a read-only leak (a backup, a replica, an export) into live admin sessions:
//
//   - The session signing key (`web-state/session.key`, internal/auth/key.go) and the
//     prober SSH private key (`worker-state`, internal/vantage/keypair.go) live in
//     per-service volumes and are NEVER in Postgres (ADR-0053). This dump reads the
//     database only — over the pool — so it *cannot* contain them. On restore both
//     regenerate; prior sessions lapse and probers re-pin (a feature, not a gap).
//   - The live `session` table — the exact "read-only DB leak → live admin sessions"
//     surface ADR-0053 §4.2 names — is EXCLUDED from the dump (see backupExcluded). Its
//     rows would be invalid under the regenerated key anyway.
//
// The exclusion is an EXPORT INVARIANT, not an accident of where the bytes sit: the dump
// reads ONLY the explicit `backupTables` allowlist. A table absent from the allowlist is
// never read — so a future table (which might hold a new secret) is not swept in by a
// "dump everything reachable" default. This is the same discipline ADR-0053 applies to
// the store: make the violation inexpressible at the site that would write it.

// backupTables is the ordered allowlist of estate + config business tables the archive
// carries. ONLY these are read. The order is FK-parent-first (account and the estate
// roots ahead of their children) so a naive in-order restore is close to load-safe; B4
// owns the authoritative restore ordering.
//
// Every write-only secret that legitimately lives in the database (ADR-0053's accepted
// posture: account.password_hash / totp_secret, personal_token.token_hash,
// sso_provider.client_secret, channel.secret) rides with its row — none is a
// session-minting key, and each is durable operator configuration a restore must
// reconstitute (login, API tokens, SSO, webhook delivery). Redacting them is an
// unspecced behaviour change that would silently break those on restore, so the archive
// keeps them exactly as the live database holds them and carries the same leak posture
// the database already has under ADR-0053 — no more, no less.
var backupTables = []string{
	// Config / identity roots.
	"account",
	"instance_config",
	"retention_settings",
	// Estate — declared inputs.
	"seed",
	"exclusion",
	"cold_scan_scope",
	"zone_file",
	"admitted_name",
	"source_state",
	// Estate — measurement fleet + runs.
	"vantage",
	"scan",
	"dispatch",
	"batch",
	"observation",
	"span",
	"verge_core_frequency_edit",
	// Estate — findings + curation.
	"signal_instance",
	"annotation",
	"proposer_lookup",
	"proposal",
	"integration_state",
	// Config — delivery, messaging, reporting.
	"channel",
	"delivery",
	"message",
	"message_read",
	"report_schedule",
	"report_delivery",
	"report_notification",
	// Config — identity providers + API tokens.
	"personal_token",
	"sso_provider",
	"sso_identity",
}

// backupExcluded is the documented denylist: business schema tables that the archive
// deliberately does NOT carry, each with the reason. Kept beside the allowlist so the two
// partition the known schema (asserted in backup_test.go) — a new table forces a human to
// classify it as estate/config or excluded, rather than being silently dumped or dropped.
//
// `goose_db_version` (goose's migration ledger) is not listed here because it is not a
// business table: restore re-applies migrations and the manifest carries the schema
// version, so the ledger is reconstructed, never carried.
// #nosec G101 -- keys are database table names (e.g. "password_reset") paired with prose
// explaining why each is excluded from the backup; none is a credential value. gosec's
// hardcoded-credential heuristic matches the "password"/"token" substrings in the names.
var backupExcluded = map[string]string{
	"session":        "live login sessions — the ADR-0053 §4.2 'DB leak → live admin sessions' surface; they lapse on restore and are invalid under the regenerated session key",
	"password_reset": "single-use, short-TTL reset-token hashes — expired and meaningless off-host, a needless credential surface with no restore value",
	"recovery_code":  "MFA recovery-code hashes — auth-bypass material with no cross-host restore value; a restored instance's operators re-enroll",
	"invite":         "pending single-use account-creation invite tokens — short-lived, no durable configuration value",
	"heartbeat":      "worker liveness ping — ephemeral runtime state, re-derived immediately after restore",
	"crtsh_throttle": "CT-log fetch rate-limit bucket — ephemeral runtime state",
	"queue_job":      "in-flight scan queue — transient work-in-progress; stale 'running' rows would be phantom after an overwrite restore",
}

const (
	// backupFormat / backupFormatVersion tag the manifest so B4 restore can recognise the
	// archive and refuse an unknown shape.
	backupFormat        = "verge-backup"
	backupFormatVersion = 1
)

// backupManifest is the first NDJSON line of the archive. It names the format and, for
// forward-restorability across a migration bump, records the schema version the archive
// was taken at (the max applied goose migration) so B4 can preflight a mismatch before
// touching anything, plus the ordered table list the stream then follows.
type backupManifest struct {
	Type          string   `json:"type"` // always "manifest"
	Format        string   `json:"format"`
	Version       int      `json:"version"`        // archive format version, not schema
	SchemaVersion int64    `json:"schema_version"` // max applied goose migration
	CreatedAt     string   `json:"created_at"`     // RFC3339 UTC
	Tables        []string `json:"tables"`
}

// backupTableLine marks the start of a table's rows in the stream.
type backupTableLine struct {
	Type string `json:"type"` // always "table"
	Name string `json:"name"`
}

// backupRowLine is one table row: the column object Postgres produced via to_jsonb,
// embedded verbatim (json.RawMessage) so every column type round-trips faithfully.
type backupRowLine struct {
	Type  string          `json:"type"` // always "row"
	Table string          `json:"table"`
	Data  json.RawMessage `json:"data"`
}

// backupDownload streams the data-only logical archive as an attachment (#391, B3).
// Admin-gated (requireAdmin). On a clean stream it records the last-backup instant and
// byte size via SetLastBackup, which the Backup card then surfaces (.Backup.LastAt/
// LastSize). A backup taken during an in-flight restore should 409; the in-flight-restore
// signal is B4's to introduce (restore is not built yet), so that guard is wired when B4
// lands its restore state — this handler carries no such state of its own.
func (s *server) backupDownload(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// The dump reads the raw pool directly (no sqlc — internal/db stays untouched this
	// round). Off a pool (e.g. the pixel/dev harness) there is nothing to dump.
	if s.pool == nil {
		http.Error(w, "backup is unavailable in this mode", http.StatusServiceUnavailable)
		return
	}

	// The schema version the archive is taken at — the max applied goose migration — so
	// B4 restore can refuse a mismatched archive before it overwrites anything.
	var schemaVersion int64
	if err := s.pool.QueryRow(ctx, "SELECT COALESCE(max(version_id), 0) FROM goose_db_version").Scan(&schemaVersion); err != nil {
		s.serverError(w, "backup schema version", err)
		return
	}

	filename := "verge-backup-" + s.now().UTC().Format("20060102-150405") + ".ndjson"
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	// Count what actually reaches the client so the recorded size is the archive's size.
	cw := &countingWriter{w: w}
	if err := s.streamBackup(ctx, cw, schemaVersion, s.now()); err != nil {
		// The stream has already started (status + headers sent), so we cannot switch to
		// an error status here — log it and let the truncated download fail on the client.
		// A partial archive is NOT recorded as a successful backup below.
		log.Printf("web: backup: stream: %v", err)
		return
	}

	// Record the last UI-taken backup only on a clean stream (F's setter). #nosec — size
	// is a byte count, never negative.
	if err := s.store.SetLastBackup(ctx, pgtype.Int8{Int64: cw.n, Valid: true}); err != nil {
		log.Printf("web: backup: record last backup: %v", err)
	}
}

// streamBackup writes the whole archive to w: the manifest line, then, for each
// allowlisted table in order, a table marker line followed by one NDJSON row line per
// database row (the row serialised by Postgres via to_jsonb, so all column types survive).
// It streams table-by-table over the pool — constant memory, big-estate friendly — and
// never reads a table outside backupTables.
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

// dumpBackupTable streams one table's rows as NDJSON. Each row is materialised by Postgres
// as a single jsonb object (to_jsonb(row)), embedded verbatim into the row line so every
// column type — timestamps, numerics, bytea, arrays, jsonb — round-trips without a
// hand-written per-type encoder. The table name comes from the hardcoded backupTables
// allowlist (never request input), so the interpolated identifier is not an injection
// surface; it is still double-quoted defensively.
func (s *server) dumpBackupTable(ctx context.Context, w io.Writer, table string) error {
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
		if err := writeBackupRow(w, table, data); err != nil {
			return err
		}
	}
	return rows.Err()
}

// writeBackupManifest emits the manifest as the archive's first NDJSON line.
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

// writeBackupTableHeader emits a table's start marker.
func writeBackupTableHeader(w io.Writer, table string) error {
	return writeJSONLine(w, backupTableLine{Type: "table", Name: table})
}

// writeBackupRow emits one row line, embedding the Postgres jsonb verbatim.
func writeBackupRow(w io.Writer, table string, data json.RawMessage) error {
	return writeJSONLine(w, backupRowLine{Type: "row", Table: table, Data: data})
}

// writeJSONLine marshals v and writes it as one newline-terminated NDJSON record.
func writeJSONLine(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

// countingWriter tallies the bytes forwarded to the underlying writer so a clean stream
// can record the archive's true size.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
