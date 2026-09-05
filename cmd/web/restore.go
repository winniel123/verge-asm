package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

const restoreMaxUpload = 1 << 30

// The confirm dialog re-posts only the typed word, so the pre-flighted archive is held here.

type restoreStaging struct {
	file     string
	takenAt  string
	subjects int
	schema   string
	archive  []byte
}

type restorePreflightView struct {
	File     string
	TakenAt  string
	Subjects int
	Schema   string
}

type restoreConfirmView struct {
	File     string
	TakenAt  string
	Subjects int
}

type restorePreflight struct {
	SchemaVersion int64
	Tables        []string
	Subjects      int
}

var (
	errRestoreBadManifest = errors.New("restore: archive has no valid manifest line")
	errRestoreBadFormat   = errors.New("restore: not a verge-backup archive")
	errRestoreUnknownTbl  = errors.New("restore: archive names a table outside the backup allowlist")
	errRestoreSchema      = errors.New("restore: archive schema version does not match this instance")
)

func preflightArchive(r io.Reader) (restorePreflight, error) {
	sc := bufio.NewScanner(r)
	// The 64KiB default token cap on a bufio.Scanner is too small for a wide observation row.
	sc.Buffer(make([]byte, 0, 1<<20), restoreMaxUpload)

	if !sc.Scan() {
		return restorePreflight{}, errRestoreBadManifest
	}
	var man backupManifest
	if err := json.Unmarshal(sc.Bytes(), &man); err != nil {
		return restorePreflight{}, errRestoreBadManifest
	}
	if man.Type != "manifest" || man.Format != backupFormat || man.Version != backupFormatVersion {
		return restorePreflight{}, errRestoreBadFormat
	}
	for _, t := range man.Tables {
		if !backupAllowed(t) {
			return restorePreflight{}, errRestoreUnknownTbl
		}
	}

	subjects := make(map[string]struct{})
	for sc.Scan() {
		line := sc.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var head struct {
			Type  string          `json:"type"`
			Table string          `json:"table"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(line, &head); err != nil {
			return restorePreflight{}, fmt.Errorf("restore: malformed archive line: %w", err)
		}
		if head.Type != "row" || head.Table != "span" {
			continue
		}
		var span struct {
			SubjectKey string           `json:"subject_key"`
			ClosedAt   *json.RawMessage `json:"closed_at"`
		}
		if err := json.Unmarshal(head.Data, &span); err != nil {
			continue
		}
		if span.ClosedAt == nil || string(*span.ClosedAt) == "null" {
			subjects[span.SubjectKey] = struct{}{}
		}
	}
	if err := sc.Err(); err != nil {
		return restorePreflight{}, fmt.Errorf("restore: read archive: %w", err)
	}

	return restorePreflight{
		SchemaVersion: man.SchemaVersion,
		Tables:        man.Tables,
		Subjects:      len(subjects),
	}, nil
}

func backupAllowed(table string) bool {
	for _, t := range backupTables {
		if t == table {
			return true
		}
	}
	return false
}

func (s *server) restorePreflight(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// A restore mid-dispatch would race an in-progress write (docs/guides/backup-and-restore.md).
	if s.scanInFlight(ctx) {
		s.restoreErrorRedirect(w, r, "inflight")
		return
	}

	if s.pool == nil {
		http.Error(w, "restore is unavailable in this mode", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, restoreMaxUpload)
	if err := r.ParseMultipartForm(restoreMaxUpload); err != nil { // #nosec G120 (request body bounded by the MaxBytesReader immediately above)
		s.restoreErrorRedirect(w, r, "unreadable")
		return
	}
	file, header, err := r.FormFile("archive")
	if err != nil {
		s.restoreErrorRedirect(w, r, "unreadable")
		return
	}
	defer file.Close()

	archive, err := io.ReadAll(io.LimitReader(file, restoreMaxUpload))
	if err != nil {
		s.restoreErrorRedirect(w, r, "unreadable")
		return
	}

	pf, err := preflightArchive(bytes.NewReader(archive))
	if err != nil {
		log.Printf("web: restore: preflight: %v", err)
		s.restoreErrorRedirect(w, r, "unreadable")
		return
	}

	running, err := s.runningSchemaVersion(ctx)
	if err != nil {
		s.serverError(w, "restore: read schema version", err)
		return
	}
	// The replay is naive and in-order, so only an archive on this schema lands in the same shape.
	if pf.SchemaVersion != running {
		s.restoreErrorRedirect(w, r, "schema")
		return
	}

	filename := header.Filename
	if filename == "" {
		filename = "backup.ndjson"
	}
	takenAt := formatArchiveTakenAt(archive)

	s.stashRestore(acct.ID, &restoreStaging{
		file:     filename,
		takenAt:  takenAt,
		subjects: pf.Subjects,
		schema:   strconv.FormatInt(pf.SchemaVersion, 10),
		archive:  archive,
	})
	s.backToSection(w, r, "instance")
}

func (s *server) restoreApply(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// The browser's typed-confirm gate is never trusted (docs/guides/backup-and-restore.md).
	if r.FormValue("confirm") != "restore" {
		s.restoreErrorRedirect(w, r, "confirm")
		return
	}

	stg := s.stagedRestore(acct.ID)
	if stg == nil {
		s.restoreErrorRedirect(w, r, "expired")
		return
	}

	if s.scanInFlight(ctx) {
		s.restoreErrorRedirect(w, r, "inflight")
		return
	}

	if s.pool == nil {
		http.Error(w, "restore is unavailable in this mode", http.StatusServiceUnavailable)
		return
	}

	if err := s.applyRestore(ctx, stg.archive); err != nil {
		log.Printf("web: restore: apply: %v", err)
		s.restoreErrorRedirect(w, r, "apply")
		return
	}

	// The archive carries no session or prober key, so a restore mints fresh ones (ADR-0124).
	if err := s.rotateSessionKey(); err != nil {
		log.Printf("web: restore: rotate session key: %v", err)
	}

	s.clearRestore(acct.ID)

	http.Redirect(w, r, "/login?notice=restored", http.StatusSeeOther)
}

func (s *server) applyRestore(ctx context.Context, archive []byte) error {
	sc := bufio.NewScanner(bytes.NewReader(archive))
	sc.Buffer(make([]byte, 0, 1<<20), restoreMaxUpload)
	if !sc.Scan() {
		return errRestoreBadManifest
	}
	var man backupManifest
	if err := json.Unmarshal(sc.Bytes(), &man); err != nil {
		return errRestoreBadManifest
	}
	if man.Type != "manifest" || man.Format != backupFormat || man.Version != backupFormatVersion {
		return errRestoreBadFormat
	}
	for _, t := range man.Tables {
		if !backupAllowed(t) {
			return errRestoreUnknownTbl
		}
	}

	identity, err := s.identityTables(ctx)
	if err != nil {
		return fmt.Errorf("restore: read identity tables: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// CASCADE also clears ephemeral tables the archive never carried, which a restore must drop.
	if len(man.Tables) > 0 {
		var qn []string
		for _, t := range man.Tables {
			qn = append(qn, `"`+t+`"`)
		}
		// TRUNCATE takes CASCADE, so the manifest gate above bounds it too (ADR-0174 §1, #1363).
		if _, err := tx.Exec(ctx, "TRUNCATE "+joinComma(qn)+" RESTART IDENTITY CASCADE"); err != nil {
			return fmt.Errorf("restore: truncate: %w", err)
		}
	}

	// The archive is written FK-parents-first (backup.go), so the replay order is load-bearing.
	for sc.Scan() {
		line := sc.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var row backupRowLine
		if err := json.Unmarshal(line, &row); err != nil {
			return fmt.Errorf("restore: malformed archive line: %w", err)
		}
		if row.Type != "row" {
			continue
		}
		if !backupAllowed(row.Table) {
			return errRestoreUnknownTbl
		}
		// An older archive still holds these cleartext columns, so the redaction re-applies (ADR-0160 §3).
		if data, err := redactBackupRow(row.Table, row.Data); err != nil {
			return fmt.Errorf("restore: redact %s: %w", row.Table, err)
		} else {
			row.Data = data
		}
		// Foreign keys reference the archive's explicit ids, so an identity column must accept them.
		overriding := ""
		if identity[row.Table] {
			overriding = "OVERRIDING SYSTEM VALUE "
		}
		// The table name is interpolated, so backupAllowed is the only bound (ADR-0174 §1, #1363).
		q := `INSERT INTO "` + row.Table + `" ` + overriding +
			`SELECT * FROM jsonb_populate_record(NULL::"` + row.Table + `", $1::jsonb)`
		if _, err := tx.Exec(ctx, q, string(row.Data)); err != nil {
			return fmt.Errorf("restore: insert into %s: %w", row.Table, err)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("restore: read archive: %w", err)
	}

	// No private half for the archived keys exists on this host, so the fleet must re-pin (ADR-0124).
	if _, err := tx.Exec(ctx,
		"UPDATE vantage SET public_key = NULL, host_key = NULL, availability = 'pending', latency_ms = NULL",
	); err != nil {
		return fmt.Errorf("restore: reset prober keys: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM session"); err != nil {
		return fmt.Errorf("restore: lapse sessions: %w", err)
	}

	// The rows carry explicit ids, so a sequence left behind them would collide on the next insert.
	if _, err := tx.Exec(ctx, resyncIdentitySequencesSQL); err != nil {
		return fmt.Errorf("restore: resync sequences: %w", err)
	}

	return tx.Commit(ctx)
}

const resyncIdentitySequencesSQL = `
DO $$
DECLARE
	r     RECORD;
	seq   TEXT;
	maxv  BIGINT;
BEGIN
	FOR r IN
		SELECT c.relname AS tbl, a.attname AS col
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND a.attidentity IN ('a', 'd') AND a.attnum > 0 AND NOT a.attisdropped
	LOOP
		seq := pg_get_serial_sequence(quote_ident(r.tbl), r.col);
		IF seq IS NOT NULL THEN
			EXECUTE format('SELECT COALESCE(max(%I), 0) FROM %I', r.col, r.tbl) INTO maxv;
			PERFORM setval(seq, GREATEST(maxv, 1), maxv > 0);
		END IF;
	END LOOP;
END $$;`

func (s *server) identityTables(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.relname
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND a.attidentity = 'a' AND a.attnum > 0 AND NOT a.attisdropped`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

func (s *server) runningSchemaVersion(ctx context.Context) (int64, error) {
	var v int64
	err := s.pool.QueryRow(ctx, "SELECT COALESCE(max(version_id), 0) FROM goose_db_version").Scan(&v)
	return v, err
}

func (s *server) rotateSessionKey() error {
	var newKey []byte
	var err error
	if s.stateDir != "" {
		newKey, err = auth.RotateKey(s.stateDir)
	} else {
		newKey = make([]byte, 32)
		_, err = rand.Read(newKey)
	}
	if err != nil {
		return err
	}
	totp, _ := auth.DeriveTOTPKey(newKey)
	s.restoreMu.Lock()
	s.key = newKey
	s.totpKey = totp
	s.restoreMu.Unlock()
	return nil
}

func (s *server) scanInFlight(ctx context.Context) bool {
	rows, err := s.store.ListDispatchProgress(ctx, scansHistoryLimit)
	if err != nil {
		log.Printf("web: restore: in-flight check: %v", err)
		return true
	}
	for _, row := range rows {
		if row.Ready+row.Running > 0 {
			return true
		}
	}
	return false
}

func (s *server) stashRestore(accountID int64, stg *restoreStaging) {
	s.restoreMu.Lock()
	defer s.restoreMu.Unlock()
	if s.restoreStage == nil {
		s.restoreStage = make(map[int64]*restoreStaging)
	}
	s.restoreStage[accountID] = stg
}

func (s *server) stagedRestore(accountID int64) *restoreStaging {
	s.restoreMu.Lock()
	defer s.restoreMu.Unlock()
	return s.restoreStage[accountID]
}

func (s *server) clearRestore(accountID int64) {
	s.restoreMu.Lock()
	defer s.restoreMu.Unlock()
	delete(s.restoreStage, accountID)
}

func (s *server) restoreErrorRedirect(w http.ResponseWriter, r *http.Request, code string) {
	s.flashSettings(w, r, settingsForms{
		section:      "instance",
		restoreError: restoreErrorMessage(code),
	})
}

func restoreErrorMessage(code string) string {
	switch code {
	case "inflight":
		return "A scan is in flight — stop the dispatch first, then restore."
	case "unreadable":
		return "That file is not a Verge backup archive, or it is corrupted. Nothing was touched."
	case "schema":
		return "This archive was taken on a different schema version than this instance runs. Restore it on a matching version — nothing was touched."
	case "confirm":
		return "Restore was not confirmed. Type restore to confirm."
	case "expired":
		return "The pre-flighted archive is no longer staged — upload it again to restore."
	case "apply":
		return "Restore failed while applying. The change was rolled back and nothing was overwritten."
	default:
		return ""
	}
}

func formatArchiveTakenAt(archive []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(archive))
	sc.Buffer(make([]byte, 0, 1<<20), restoreMaxUpload)
	if !sc.Scan() {
		return ""
	}
	var man backupManifest
	if err := json.Unmarshal(sc.Bytes(), &man); err != nil {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, man.CreatedAt); err == nil {
		return t.UTC().Format("2006-01-02 15:04 UTC")
	}
	return man.CreatedAt
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
