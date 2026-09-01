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

// Restore — the destructive counterpart to B3's backup (cmd/web/backup.go), #391/B4,
// ADR-0124. It has two admin-gated POST entry points, both wired in handlers.go:
//
//   - POST /settings/restore/preflight (multipart, field `archive`) validates an uploaded
//     archive WITHOUT touching anything: it parses B3's manifest, refuses a shape or a
//     schema version this instance cannot restore, and counts the subjects the archive
//     carries. On success it stages the validated archive for this admin and PRG-redirects
//     to the Instance tab, where the Restore card renders `.Preflight`.
//   - POST /settings/restore (confirm=`restore`) applies the staged archive: a single
//     transaction overwrites the estate + config tables from the archive, clears the
//     prober key material so the worker re-provisions (re-pin), and deletes every session
//     row. On commit it regenerates the session signing key so every cookie signed under
//     the old key is dead at once. A failure rolls the whole transaction back and changes
//     nothing (`.RestoreError`).
//
// SECRET REGENERATION IS A NAMED GUARANTEE (ADR-0124, ADR-0053), not an incidental side
// effect: the archive never carried the session key or the prober key (they are not in
// Postgres), so a restore MUST mint fresh ones. The consequence — every session lapses and
// probers re-pin — is the design, the loud-and-recoverable failure ADR-0053 already priced.
//
// The two cleartext-secret columns the backup redacts (#739) — channel.secret (webhook HMAC
// key) and sso_provider.client_secret (OAuth client secret) — are likewise NOT carried
// across a restore. applyRestore re-applies redactBackupRow to every row, so even a
// pre-#739 archive that still holds these values cannot reconstitute them and the redacted
// null in a current archive cannot silently overwrite a live secret: the columns land NULL
// and the operator knowingly re-enters them afterward (Settings → the channel/SSO forms).
// This is a full-replace restore (TRUNCATE + INSERT), so preserving a live secret in place
// is not possible; leaving these two columns NULL is the safe, documented behaviour.
//
// The archive is applied over the raw pgx pool `web` already holds (no sqlc — internal/db
// stays untouched this round), mirroring how backup.go reads it. Every interpolated table
// name is validated against B3's `backupTables` allowlist before it reaches SQL, so the
// manifest cannot smuggle an arbitrary identifier into a statement.

// restoreMaxUpload caps the multipart archive an operator may upload for pre-flight. A
// restore is a rare admin act over a bounded estate; the cap keeps a single upload from
// exhausting memory while comfortably covering a real dump.
const restoreMaxUpload = 1 << 30 // 1 GiB

// restoreStaging is one admin's pre-flighted archive, held in process between the multipart
// pre-flight and the typed-confirm apply. The confirm dialog re-posts only the typed word,
// so the validated bytes and the human-facing summary live here until the apply consumes
// them or the admin navigates away (a plain Instance render leaves the stage untouched).
type restoreStaging struct {
	file     string // the uploaded filename, echoed in the pre-flight callout + dialog
	takenAt  string // manifest CreatedAt, formatted for display
	subjects int    // distinct current subjects the archive carries
	schema   string // manifest schema version, displayed (already validated to match)
	archive  []byte // the whole validated archive, replayed verbatim on apply
}

// restorePreflightView / restoreConfirmView are the settings-form projections the Instance
// tab renders — the Restore card's `.Preflight` warn callout and its `.RestoreConfirm`
// typed-confirm dialog. They mirror the design's data contract (WORK-ORDER-390-391 §#391).
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

// restorePreflight is the parsed, validated result of reading an archive's manifest and
// scanning its rows — the pure product of preflightArchive, independent of any HTTP or
// database state so it is unit-testable against a B3-produced archive.
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

// preflightArchive parses an archive stream (B3's NDJSON: a manifest line, then per-table
// markers and one row line each) and returns its schema version, table list and the count
// of current subjects it carries — WITHOUT applying anything. It refuses an unparseable or
// wrong-format archive, and refuses a manifest that names a table outside B3's allowlist
// (so a hand-forged archive cannot steer the apply's SQL at an arbitrary identifier). The
// caller compares SchemaVersion against the running schema; this function does not read the
// database, which is what makes it testable off a live Postgres.
func preflightArchive(r io.Reader) (restorePreflight, error) {
	sc := bufio.NewScanner(r)
	// The default 64KiB token cap is too small for a wide row (a batch of observations),
	// so give the scanner room up to the upload cap.
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

	// Count the current subjects the archive carries — distinct subject_key across its open
	// spans, the same "an open span is a current subject" definition the inventory read uses
	// (ADR-0082). It is read off the archive's own rows, never fabricated.
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
			SubjectKey string `json:"subject_key"`
			ClosedAt   *json.RawMessage `json:"closed_at"`
		}
		if err := json.Unmarshal(head.Data, &span); err != nil {
			continue
		}
		// An open span (no closed_at) marks a subject the estate currently holds.
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

// backupAllowed reports whether a table name is in B3's ordered allowlist. Restore reads
// and overwrites only these — a manifest naming anything else is refused, honouring the
// same allowlist/denylist partition backup writes with.
func backupAllowed(table string) bool {
	for _, t := range backupTables {
		if t == table {
			return true
		}
	}
	return false
}

// restorePreflight (handler) validates an uploaded archive without applying it (#391/B4).
// Admin-gated in handlers.go. It refuses up front while a scan is in flight (an overwrite
// mid-dispatch would strand in-flight work), then parses + schema-checks the upload; any
// failure redirects back to the Instance tab with a fixed `.RestoreError` and NOTHING is
// touched. On success it stages the validated archive for this admin and PRG-redirects so
// the Restore card renders `.Preflight`.
func (s *server) restorePreflight(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// Refuse a restore while a scan is in flight — stop the dispatch first (WORK-ORDER
	// §#391 edge). Checked before the upload is even read, so a busy instance rejects fast.
	if s.scanInFlight(ctx) {
		s.restoreErrorRedirect(w, r, "inflight")
		return
	}

	// Off a pool (the pixel/dev harness) there is nothing to validate a schema against and
	// nothing to restore into, exactly as backup is unavailable there.
	if s.pool == nil {
		http.Error(w, "restore is unavailable in this mode", http.StatusServiceUnavailable)
		return
	}

	// Bound the whole upload before parsing so a large or endless body cannot exhaust
	// memory/disk (gosec G120): the cap covers a real dump with room to spare, and an
	// over-cap upload fails parsing into the same "unreadable" refusal.
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

	// Schema compatibility: the archive must have been taken on the schema version this
	// instance runs, so a naive in-order restore lands in the same shape. A mismatch is
	// refused before anything is touched.
	running, err := s.runningSchemaVersion(ctx)
	if err != nil {
		s.serverError(w, "restore: read schema version", err)
		return
	}
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
	// Back to the URL the pre-flight was submitted from (ADR-0130 §3, ticket #977). A
	// pre-flight reads and hashes the whole archive, so it is one of the slow acts failure
	// class C named: the operator has been waiting, and landing them at the top of the
	// Instance tab rather than at the restore card they submitted from is the miss.
	s.backToSection(w, r, "instance")
}

// restoreApply applies the staged, pre-flighted archive after a typed confirmation
// (#391/B4). Admin-gated. The confirmation word `restore` is validated server-side — never
// trusting the JS gate — and the in-flight-scan refusal is re-checked, since a dispatch may
// have started between pre-flight and apply. The apply itself is a single transaction:
// overwrite the allowlisted tables from the archive, clear prober key material so the
// worker re-provisions, and delete every session row. On commit the session signing key is
// regenerated so the running process stops honouring old cookies at once. Any failure rolls
// the transaction back and changes nothing (`.RestoreError`).
func (s *server) restoreApply(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// Server-side confirm gate: the design's [data-typed-confirm] JS only enables the
	// button; the word is authoritative here. A wrong or missing confirmation applies
	// nothing and leaves the staged archive in place so the operator can retry.
	if r.FormValue("confirm") != "restore" {
		s.restoreErrorRedirect(w, r, "confirm")
		return
	}

	stg := s.stagedRestore(acct.ID)
	if stg == nil {
		// The pre-flight expired (a restart, or the admin already applied/discarded it).
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
		// The transaction rolled back — nothing was overwritten.
		log.Printf("web: restore: apply: %v", err)
		s.restoreErrorRedirect(w, r, "apply")
		return
	}

	// The data is committed and every session row is gone. Regenerate the session signing
	// key so this process stops honouring any cookie signed under the old key immediately,
	// not only after a restart (ADR-0124: every session lapses). A failure here does not
	// un-restore the data; the sessions are already deleted, so it is logged, not fatal.
	if err := s.rotateSessionKey(); err != nil {
		log.Printf("web: restore: rotate session key: %v", err)
	}

	s.clearRestore(acct.ID)

	// The caller's own session is gone with the rest; send them to sign in again.
	http.Redirect(w, r, "/login?notice=restored", http.StatusSeeOther)
}

// applyRestore overwrites the estate + config from the archive in one transaction (#391/B4).
// It truncates the allowlisted tables the manifest carries, replays each row through
// jsonb_populate_record so every column type round-trips, clears the prober key material so
// the worker re-provisions (re-pin), deletes every session row (lapse), and resyncs the
// identity sequences to the restored rows' high-water marks. Any error returns before
// Commit, so the deferred Rollback leaves the database exactly as it was — no partial write.
func (s *server) applyRestore(ctx context.Context, archive []byte) error {
	// Parse the manifest first so we know which tables to truncate before inserting; the
	// row stream that follows is replayed in the same FK-parent-first order B3 wrote it.
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

	// Which of the target tables define a GENERATED ALWAYS AS IDENTITY column: those need
	// OVERRIDING SYSTEM VALUE to accept the archive's explicit ids (the ids FKs reference).
	identity, err := s.identityTables(ctx)
	if err != nil {
		return fmt.Errorf("restore: read identity tables: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	// Truncate every target table (validated ⊆ allowlist) in one statement. CASCADE also
	// clears the excluded ephemeral tables that FK-reference them (session, queue_job,
	// password_reset …) — exactly the stale runtime state a fresh overwrite must not keep.
	if len(man.Tables) > 0 {
		var qn []string
		for _, t := range man.Tables {
			qn = append(qn, `"`+t+`"`)
		}
		if _, err := tx.Exec(ctx, "TRUNCATE "+joinComma(qn)+" RESTART IDENTITY CASCADE"); err != nil {
			return fmt.Errorf("restore: truncate: %w", err)
		}
	}

	// Replay the rows in archive order. jsonb_populate_record rebuilds each row from the
	// stored jsonb against the live table rowtype, so every column — timestamps, arrays,
	// bytea, jsonb — round-trips without a hand-written decoder. The table name is an
	// allowlisted, validated identifier, so the interpolation is not an injection surface.
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
			continue // table markers and the manifest are not row data
		}
		if !backupAllowed(row.Table) {
			return errRestoreUnknownTbl
		}
		// Re-apply the export redaction (#739): the cleartext-secret columns
		// (channel.secret / sso_provider.client_secret) are never written from an archive,
		// new or pre-#739. This is a full-replace restore (TRUNCATE + INSERT), so a live
		// secret cannot be preserved in place; instead these columns land NULL and the
		// operator knowingly re-enters them after the restore — a restore can never
		// reconstitute nor silently overwrite them with an archived value.
		if data, err := redactBackupRow(row.Table, row.Data); err != nil {
			return fmt.Errorf("restore: redact %s: %w", row.Table, err)
		} else {
			row.Data = data
		}
		overriding := ""
		if identity[row.Table] {
			overriding = "OVERRIDING SYSTEM VALUE "
		}
		q := `INSERT INTO "` + row.Table + `" ` + overriding +
			`SELECT * FROM jsonb_populate_record(NULL::"` + row.Table + `", $1::jsonb)`
		if _, err := tx.Exec(ctx, q, string(row.Data)); err != nil {
			return fmt.Errorf("restore: insert into %s: %w", row.Table, err)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("restore: read archive: %w", err)
	}

	// Prober key regeneration (ADR-0124): the archive carried the source instance's public
	// keys, but this host's worker volume holds no matching private halves. Clear the
	// published key material and reset the fleet to pending so the worker's provision sweep
	// mints a fresh keypair per vantage and the prober re-pins — the private half never
	// leaves the worker volume, so web only signals the need here.
	if _, err := tx.Exec(ctx,
		"UPDATE vantage SET public_key = NULL, host_key = NULL, availability = 'pending', latency_ms = NULL",
	); err != nil {
		return fmt.Errorf("restore: reset prober keys: %w", err)
	}

	// Lapse every session (ADR-0124): the archive never carried the session table, and any
	// row that survived the overwrite is invalid under the regenerated key anyway. Deleting
	// them makes the lapse immediate and server-side, independent of the in-memory key swap.
	if _, err := tx.Exec(ctx, "DELETE FROM session"); err != nil {
		return fmt.Errorf("restore: lapse sessions: %w", err)
	}

	// Resync identity sequences to the restored rows' high-water marks. The rows were
	// inserted with explicit ids, so each identity column's sequence must advance past them
	// or the next insert would collide. Empty tables reset to 1 (is_called=false).
	if _, err := tx.Exec(ctx, resyncIdentitySequencesSQL); err != nil {
		return fmt.Errorf("restore: resync sequences: %w", err)
	}

	return tx.Commit(ctx)
}

// resyncIdentitySequencesSQL advances every identity column's sequence to the maximum id
// present after the restore, so a subsequent insert does not collide with a restored row.
// It runs over the public schema's identity columns generically (by-default and always),
// resolving each column's owning sequence via pg_get_serial_sequence.
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

// identityTables returns the set of public tables that define a GENERATED ALWAYS AS
// IDENTITY column — the tables whose id must be inserted with OVERRIDING SYSTEM VALUE so the
// archive's explicit ids (the ids foreign keys reference) are honoured rather than rejected.
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

// runningSchemaVersion is the max applied goose migration — the schema version an archive's
// manifest is checked against before restore. It reads goose's ledger with a raw pool query
// (not sqlc — internal/db stays untouched), the same read backup.go takes to stamp the
// manifest, so a round-tripped archive matches exactly.
func (s *server) runningSchemaVersion(ctx context.Context) (int64, error) {
	var v int64
	err := s.pool.QueryRow(ctx, "SELECT COALESCE(max(version_id), 0) FROM goose_db_version").Scan(&v)
	return v, err
}

// rotateSessionKey mints a fresh session signing key, persists it to web's state volume when
// one is configured, and swaps it (and the derived TOTP sub-key) into the running process so
// every cookie signed under the old key is rejected at once (ADR-0124). Off a configured
// state dir (tests) it swaps an in-memory key only — the lapse still holds for the process.
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

// scanInFlight reports whether any recent dispatch still has ready or running jobs — the
// same "an in-flight dispatch has jobs to run" test the queue monitor uses. Restore refuses
// while one is in flight (an overwrite would strand it). A read error is treated as in
// flight: for a destructive act, refusing on uncertainty is the safe direction.
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

// stashRestore / stagedRestore / clearRestore hold one admin's pre-flighted archive between
// pre-flight and apply, keyed by account id. In-process and best-effort — a restart drops a
// pending pre-flight, the safe direction (nothing was applied).
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

// restoreErrorRedirect PRG-redirects a refused pre-flight or apply back to the URL its form
// was submitted from, carrying the refusal on the session form flash (ADR-0130 §1).
//
// The code used to ride the destination as ?restore-error=<code>. It reflected nothing —
// restoreErrorMessage mapped it to a fixed line — so it was never unsafe; what it did was
// land the operator at a URL their form was not submitted from, which misses the scroll key
// (ADR-0130 §2) on the slowest act in the console. The code is still the internal currency,
// because it is what the call sites name; restoreErrorMessage resolves it here instead of
// on the far side of a query string.
func (s *server) restoreErrorRedirect(w http.ResponseWriter, r *http.Request, code string) {
	s.flashSettings(w, r, settingsForms{
		section:      "instance",
		restoreError: restoreErrorMessage(code),
	})
}

// restoreErrorMessage maps a refusal's code to a fixed `.RestoreError` line. An unknown code
// yields no line, so a caller that names one this table does not hold leaves the landing
// page with no callout rather than with a fabricated one.
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

// formatArchiveTakenAt reads the manifest's CreatedAt from an archive's first line and
// renders it in the Instance card's instant format (matching the Release/Backup cards). A
// manifest that does not parse falls back to the raw stored value, never a fabricated time.
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

// joinComma joins pre-quoted identifiers for a TRUNCATE list without pulling in strings for
// one call site.
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
