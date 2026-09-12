package main

import (
	"bufio"
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/winniel123/verge-asm/db/migrations"
	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

// serial_number is a column name and X.509 serial is prose, so the type needs its own anchor.

var serialColumnRe = regexp.MustCompile(`(?m)^\s*\w+\s+(?:big|small)?serial\b`)

// A serial id and an identity id differ in pg_attribute, and the restore reads that column.

func serialTablesInMigrations(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		t.Fatalf("read the migrations dir: %v", err)
	}
	out := map[string]bool{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		data, err := migrations.FS.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, stmt := range strings.Split(stripSQLComments(string(data)), ";") {
			name, ok := createdTableName(stmt)
			if ok && serialColumnRe.MatchString(stmt) {
				out[name] = true
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no SERIAL column found in any migration, so this test proves nothing")
	}
	return out
}

func stripSQLComments(sql string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.ToLower(sql), "\n") {
		if body, _, found := strings.Cut(line, "--"); found {
			line = body
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func createdTableName(stmt string) (string, bool) {
	_, after, ok := strings.Cut(stmt, "create table ")
	if !ok {
		return "", false
	}
	after = strings.TrimPrefix(after, "if not exists ")
	name, _, ok := strings.Cut(after, "(")
	if !ok {
		return "", false
	}
	return strings.TrimSpace(name), true
}

var (
	testRestoreActor = actingAccount(db.Account{ID: 7, Username: "admin"})
	testRestoreRef   = act.RestoreRef{
		Archive: "verge-2026-09-08.tar.zst",
		TakenAt: "2026-09-08 14:02 UTC",
	}
)

func testArchiveCursor(t *testing.T, archive []byte) (backupManifest, *bufio.Scanner) {
	t.Helper()
	man, sc, err := openArchive(archive)
	if err != nil {
		t.Fatalf("openArchive: %v", err)
	}
	return man, sc
}

func replayTestArchive(t *testing.T, tx *fakeTx) error {
	t.Helper()
	man, sc := testArchiveCursor(t,
		buildTestArchive(t, 23000, []string{`{"subject_key":"a.example.com","closed_at":null}`}))
	return replayArchive(t.Context(), tx, man, sc, nil, testRestoreActor, testRestoreRef)
}

// Record is the replay's last statement, so the fake's last args are InsertAct's.

func restoreAct(t *testing.T, tx *fakeTx) (act.Actor, act.Act) {
	t.Helper()
	if len(tx.args) != 4 {
		t.Fatalf("the last statement took %d args, want the 4 of InsertAct", len(tx.args))
	}
	kind, _ := tx.args[0].(string)
	actor, _ := tx.args[1].([]byte)
	action, _ := tx.args[2].(string)
	subject, _ := tx.args[3].([]byte)
	return decodeAct(t, db.InsertActParams{
		ActorKind: kind,
		Actor:     actor,
		Action:    action,
		Subject:   subject,
	})
}

func stepOf(trail []string, needle string) int {
	for i, sql := range trail {
		if strings.Contains(sql, needle) {
			return i
		}
	}
	return -1
}

// The apply truncates the corpus it records into, so a row written first is erased (spec §5.3).

func TestRestoreWritesItsActAfterTheTruncation(t *testing.T) {
	tx := &fakeTx{}
	if err := replayTestArchive(t, tx); err != nil {
		t.Fatalf("replayArchive: %v", err)
	}

	truncated := stepOf(tx.trail, "TRUNCATE ")
	recorded := stepOf(tx.trail, "INSERT INTO act")
	if truncated < 0 {
		t.Fatal("the replay truncated nothing, so this test proves no ordering")
	}
	if recorded < 0 {
		t.Fatal("the replay wrote no Act, so the discontinuity goes unrecorded (spec §5.3)")
	}
	if recorded < truncated {
		t.Errorf("the Act ran at step %d and the TRUNCATE at step %d, so the apply erases it",
			recorded, truncated)
	}
}

// A replayed act row carries its own id, so the recorder's insert waits for the resync.

func TestRestoreWritesItsActAfterTheSequenceResync(t *testing.T) {
	tx := &fakeTx{}
	if err := replayTestArchive(t, tx); err != nil {
		t.Fatalf("replayArchive: %v", err)
	}

	resynced := stepOf(tx.trail, "pg_get_serial_sequence")
	recorded := stepOf(tx.trail, "INSERT INTO act")
	if resynced < 0 || recorded < 0 {
		t.Fatalf("the replay resynced at step %d and recorded at step %d; both must run",
			resynced, recorded)
	}
	if recorded < resynced {
		t.Errorf("the Act ran at step %d and the resync at step %d, so its id may collide",
			recorded, resynced)
	}
}

func TestRestoreActNamesTheArchiveAndWhenItWasTaken(t *testing.T) {
	tx := &fakeTx{}
	if err := replayTestArchive(t, tx); err != nil {
		t.Fatalf("replayArchive: %v", err)
	}

	_, a := restoreAct(t, tx)
	applied, ok := a.(act.RestoreApplied)
	if !ok {
		t.Fatalf("the restore recorded a %T, want act.RestoreApplied", a)
	}
	if got, want := applied.Class(), "restore.applied"; got != want {
		t.Errorf("class = %q, want %q", got, want)
	}
	want := "verge-2026-09-08.tar.zst · taken 2026-09-08 14:02 UTC"
	if got := applied.Subject(); got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
}

// The username is a captured value, so the TRUNCATE cannot empty the Actor cell (spec §5.4).

func TestRestoreActNamesTheRestoringAdmin(t *testing.T) {
	tx := &fakeTx{}
	if err := replayTestArchive(t, tx); err != nil {
		t.Fatalf("replayArchive: %v", err)
	}

	actor, _ := restoreAct(t, tx)
	who, ok := actor.(act.Account)
	if !ok {
		t.Fatalf("the restore named a %T as actor, want act.Account", actor)
	}
	if who.AccountID != 7 || who.Name() != "admin" {
		t.Errorf("actor = %d/%q, want 7/%q", who.AccountID, who.Name(), "admin")
	}
}

// The row rides the restore transaction, so a failed insert must take the replay with it.

func TestAFailedDiscontinuityRecordRefusesTheReplay(t *testing.T) {
	want := errors.New("insert act: deadlock detected")
	tx := &fakeTx{failOn: "INSERT INTO act", err: want}

	if err := replayTestArchive(t, tx); !errors.Is(err, want) {
		t.Fatalf("replayArchive returned %v, want %v; the restore would commit unrecorded", err, want)
	}
}

// The replay writes explicit ids, so a sequence the resync skips collides on the next insert.

func TestTheResyncCoversEveryBackedUpGeneratedID(t *testing.T) {
	serial := serialTablesInMigrations(t)
	for _, tbl := range backupTables {
		if !serial[tbl] {
			continue
		}
		if strings.Contains(stripSQLComments(resyncIdentitySequencesSQL), "attidentity") {
			t.Errorf("%s declares a SERIAL id and the resync filters on attidentity, "+
				"which is empty for a serial column, so its sequence is never resynced", tbl)
		}
	}
}

// An archive taken before act joined the allowlist names no act, and its rows must still go.

func TestTheTruncateAlwaysReachesTheActCorpus(t *testing.T) {
	tx := &fakeTx{}
	man, sc := testArchiveCursor(t,
		buildTestArchive(t, 23000, []string{`{"subject_key":"a.example.com","closed_at":null}`}))
	// An archive from before act joined the allowlist, which the schema gate still accepts.
	man.Tables = []string{"account", "span"}

	if err := replayArchive(t.Context(), tx, man, sc, nil, testRestoreActor, testRestoreRef); err != nil {
		t.Fatalf("replayArchive: %v", err)
	}

	at := stepOf(tx.trail, "TRUNCATE ")
	if at < 0 {
		t.Fatal("the replay truncated nothing")
	}
	if !strings.Contains(tx.trail[at], `"act"`) {
		t.Errorf("TRUNCATE ran as %q, which leaves the old corpus beside the new one", tx.trail[at])
	}
}

// TRUNCATE names each table once, and the current manifest already carries act.

func TestTheTruncateNamesTheActCorpusOnce(t *testing.T) {
	man := backupManifest{Tables: []string{"account", "act", "span"}}
	if got := slices.Sorted(slices.Values(truncateTables(man))); !slices.Equal(got,
		[]string{"account", "act", "span"}) {
		t.Errorf("truncateTables = %v, want each table once", got)
	}
}

func TestTheArchiveNameIsBoundedBeforeItReachesTheCorpus(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"a plain name", "verge-2026-09-08.tar.zst", "verge-2026-09-08.tar.zst"},
		{"a posix path", "/home/admin/verge.ndjson", "verge.ndjson"},
		{"a windows path", `C:\Users\admin\verge.ndjson`, "verge.ndjson"},
		{"an empty name", "", "backup.ndjson"},
		{"a control character", "verge\n\t\x00.ndjson", "verge.ndjson"},
		{"only control characters", "\n\x00", "backup.ndjson"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := archiveName(tc.in); got != tc.want {
				t.Errorf("archiveName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	long := archiveName(strings.Repeat("a", 5000) + ".ndjson")
	if len(long) > archiveNameMax {
		t.Errorf("archiveName kept %d bytes, want at most %d", len(long), archiveNameMax)
	}

	// A cut mid-rune would otherwise store bytes the renderer cannot read back.
	cut := archiveName(strings.Repeat("é", 200))
	if !utf8.ValidString(cut) {
		t.Errorf("archiveName returned invalid UTF-8: %q", cut)
	}
}

// An Act holds no secret, so the corpus is ordinary backup data (spec §5.3, ADR-0053).

func TestTheActCorpusRidesTheBackup(t *testing.T) {
	if _, excluded := backupExcluded["act"]; excluded {
		t.Error("act is in backupExcluded, so a restore would erase the whole history (spec §5.3)")
	}
	carried := false
	for _, tbl := range backupTables {
		if tbl == "act" {
			carried = true
		}
	}
	if !carried {
		t.Error("act is not in the backup allowlist, so no archive carries the corpus (spec §5.3)")
	}
}
