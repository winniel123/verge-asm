package main

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

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
