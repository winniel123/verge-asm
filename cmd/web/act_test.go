package main

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"reflect"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) InsertAct(ctx context.Context, arg db.InsertActParams) error {
	// A detached context reaches the store live, so the fake refuses a dead one (spec §7.6).
	if err := ctx.Err(); err != nil {
		return err
	}
	f.actTrail = append(f.actTrail, "InsertAct")
	if f.actErr != nil {
		return f.actErr
	}
	f.acts = append(f.acts, arg)
	f.appendActRow(arg)
	return nil
}

// created_at and id are defaulted in the column, so the fake defaults them too (spec §4.1).

func (f *fakeStore) appendActRow(arg db.InsertActParams) db.Act {
	at := f.actNow
	if at.IsZero() {
		at = fixedClock()()
	}
	f.actNextID++
	row := db.Act{
		ID:        f.actNextID,
		CreatedAt: pgtype.Timestamptz{Time: at, Valid: true},
		ActorKind: arg.ActorKind,
		Actor:     arg.Actor,
		Action:    arg.Action,
		Subject:   arg.Subject,
	}
	f.actRows = append(f.actRows, row)
	return row
}

func (f *fakeStore) AnyActRecorded(ctx context.Context) (bool, error) {
	if f.actListErr != nil {
		return false, f.actListErr
	}
	return len(f.actRows) > 0, nil
}

func (f *fakeStore) ListActsInRange(ctx context.Context, arg db.ListActsInRangeParams) ([]db.Act, error) {
	if f.actListErr != nil {
		return nil, f.actListErr
	}
	out := []db.Act{}
	for _, row := range f.actRows {
		if arg.FromTime.Valid && row.CreatedAt.Time.Before(arg.FromTime.Time) {
			continue
		}
		if arg.UntilTime.Valid && !row.CreatedAt.Time.Before(arg.UntilTime.Time) {
			continue
		}
		out = append(out, row)
	}
	// Newest first, the order the query and the render both take.
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].CreatedAt.Time.Equal(out[j].CreatedAt.Time) {
			return out[i].CreatedAt.Time.After(out[j].CreatedAt.Time)
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

func decodeAct(t *testing.T, arg db.InsertActParams) (act.Actor, act.Act) {
	t.Helper()
	actor, err := act.DecodeActor(arg.ActorKind, arg.Actor)
	if err != nil {
		t.Fatalf("decode actor: %v", err)
	}
	a, err := act.DecodeSubject(arg.Action, arg.Subject)
	if err != nil {
		t.Fatalf("decode subject: %v", err)
	}
	return actor, a
}

func TestDeclareSeedRecordsAfterTheMutation(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declareScope(t, ac, base, "example.com").Body.Close()

	// Recorder-before writes an Act with no act, and append-only can never retract it (spec §7.6).
	want := []string{"CreateNameSeed", "InsertAct"}
	if !reflect.DeepEqual(f.actTrail, want) {
		t.Fatalf("call order = %v, want %v", f.actTrail, want)
	}
	if len(f.acts) != 1 {
		t.Fatalf("wrote %d acts, want 1", len(f.acts))
	}
	actor, a := decodeAct(t, f.acts[0])
	if got, want := f.acts[0].Action, "seed.declared"; got != want {
		t.Errorf("action = %q, want %q", got, want)
	}
	if got, want := actor, act.Actor(act.Account{AccountID: admin.ID, UsernameSnapshot: "admin"}); got != want {
		t.Errorf("actor = %#v, want %#v", got, want)
	}
	if got, want := a.Subject(), "example.com"; got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
}

func TestDeclareSeedRecordsOneActPerScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// One Act per subject, never one per request (spec §7.6).
	declareScope(t, ac, base, "good1.com, 10.0.0.0/30, nope..com").Body.Close()

	var subjects []string
	for _, a := range f.acts {
		if a.Action != "seed.declared" {
			t.Fatalf("action = %q, want seed.declared", a.Action)
		}
		_, decoded := decodeAct(t, a)
		subjects = append(subjects, decoded.Subject())
	}
	want := []string{"good1.com", "10.0.0.0/30"}
	if !reflect.DeepEqual(subjects, want) {
		t.Fatalf("recorded subjects = %v, want %v (the refused token writes nothing)", subjects, want)
	}
}

func TestRefusedSeedDeclarationRecordsNothing(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A refused act directed nothing, and failure may not write a row (spec §7.6).
	declareScope(t, ac, base, "nope..com").Body.Close()

	if len(f.acts) != 0 || len(f.actTrail) != 0 {
		t.Fatalf("a refused declaration wrote %v", f.actTrail)
	}
}

func TestDeleteSeedRecordsTheWithdrawal(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declareScope(t, ac, base, "example.com").Body.Close()
	id := f.seeds[0].ID
	f.actTrail = nil
	f.acts = nil

	postForm(t, ac, base+"/seeds/delete", url.Values{"id": {intStr(id)}}).Body.Close()

	want := []string{"WithdrawSeed", "InsertAct"}
	if !reflect.DeepEqual(f.actTrail, want) {
		t.Fatalf("call order = %v, want %v", f.actTrail, want)
	}
	if len(f.acts) != 1 {
		t.Fatalf("wrote %d acts, want 1", len(f.acts))
	}
	actor, a := decodeAct(t, f.acts[0])
	if got, want := f.acts[0].Action, "seed.withdrawn"; got != want {
		t.Errorf("action = %q, want %q", got, want)
	}
	if got, want := actor, act.Actor(act.Account{AccountID: admin.ID, UsernameSnapshot: "admin"}); got != want {
		t.Errorf("actor = %#v, want %#v", got, want)
	}
	if got, want := a.Subject(), "example.com"; got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
}

func TestWithdrawnScopeReadsTheActsOwnReturning(t *testing.T) {
	p := netip.MustParsePrefix("10.0.0.0/8")
	cases := []struct {
		name string
		row  db.WithdrawSeedRow
		want string
	}{
		{"address", db.WithdrawSeedRow{SeedsRemoved: 1, AddressCidr: &p}, "10.0.0.0/8"},
		{"name", db.WithdrawSeedRow{SeedsRemoved: 1, NameDomain: pgtype.Text{String: "example.com", Valid: true}}, "example.com"},
		{"removed nothing", db.WithdrawSeedRow{}, ""},
	}
	for _, tc := range cases {
		if got := withdrawnScope(tc.row); got != tc.want {
			t.Errorf("%s: withdrawnScope = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestDeleteAddressSeedRecordsTheMaskedPrefix(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declareScope(t, ac, base, "10.0.0.1/24").Body.Close()
	id := f.seeds[0].ID
	f.acts = nil

	postForm(t, ac, base+"/seeds/delete", url.Values{"id": {intStr(id)}}).Body.Close()

	if len(f.acts) != 1 {
		t.Fatalf("wrote %d acts, want 1", len(f.acts))
	}
	_, a := decodeAct(t, f.acts[0])
	if got, want := a.Subject(), "10.0.0.0/24"; got != want {
		t.Errorf("subject = %q, want %q (the masked form the row holds)", got, want)
	}
}

func TestDeleteUnknownSeedRecordsNothing(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	postForm(t, ac, base+"/seeds/delete", url.Values{"id": {"4242"}}).Body.Close()

	if len(f.acts) != 0 {
		t.Fatalf("an unknown id wrote %d acts, want 0", len(f.acts))
	}
}

func TestRecordOutlivesRequestCancellation(t *testing.T) {
	f := newFakeStore()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// The operator navigating away must not drop the row (spec §7.6, ruling 5).
	recorder{store: f}.Record(ctx, actingAccount(db.Account{ID: 7, Username: "admin"}),
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "example.com"}})

	if len(f.acts) != 1 {
		t.Fatalf("a cancelled request wrote %d acts, want 1", len(f.acts))
	}
}

func TestRecordDoesNotRetry(t *testing.T) {
	f := newFakeStore()
	f.actErr = errors.New("insert act: connection reset")

	// One attempt: a second failure leaves the identical hole (spec §7.6).
	recorder{store: f}.Record(t.Context(), actingAccount(db.Account{ID: 7, Username: "admin"}),
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "example.com"}})

	if len(f.actTrail) != 1 {
		t.Fatalf("a failed insert was attempted %d times, want 1", len(f.actTrail))
	}
}

func TestRecordRefusesAnUnregisteredActor(t *testing.T) {
	f := newFakeStore()

	// A row no decoder can read is unreadable forever, because the corpus generates no DELETE.
	recorder{store: f}.Record(t.Context(), act.GrantHolder{},
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "example.com"}})

	if len(f.actTrail) != 0 {
		t.Fatalf("an unencodable actor reached the store: %v", f.actTrail)
	}
}

// The restore binds the same Record to its transaction, spec §7.6's one exception (#1834).

type fakeTx struct {
	sql    string
	args   []any
	ctxErr error
	err    error
}

func (tx *fakeTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.sql = sql
	tx.args = args
	tx.ctxErr = ctx.Err()
	return pgconn.CommandTag{}, tx.err
}

func (tx *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("fakeTx: Query is out of this test's reach")
}

func (tx *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func TestTxRecorderWritesThroughTheTransaction(t *testing.T) {
	tx := &fakeTx{}

	if err := txRecorder(tx).Record(t.Context(), actingAccount(db.Account{ID: 7, Username: "admin"}),
		act.SeedWithdrawn{SeedScope: act.SeedScope{Scope: "example.com"}}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if tx.sql == "" {
		t.Fatal("the tx-bound recorder wrote nothing to its transaction")
	}
	if len(tx.args) != 4 {
		t.Fatalf("InsertAct took %d args, want 4", len(tx.args))
	}
	if got, want := tx.args[2], "seed.withdrawn"; got != want {
		t.Errorf("action = %v, want %q", got, want)
	}
}

func TestTxRecorderReturnsTheInsertError(t *testing.T) {
	want := errors.New("insert act: relation act does not exist")
	tx := &fakeTx{err: want}

	// A failed statement poisons the tx, so the restore must roll back (spec §7.6).
	got := txRecorder(tx).Record(t.Context(), actingAccount(db.Account{ID: 7, Username: "admin"}),
		act.SeedWithdrawn{SeedScope: act.SeedScope{Scope: "example.com"}})

	if !errors.Is(got, want) {
		t.Fatalf("Record returned %v, want %v", got, want)
	}
}

func TestTxRecorderKeepsItsTransactionsCancellation(t *testing.T) {
	tx := &fakeTx{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// Detaching a tx-bound insert outlives the rollback that is tearing its own tx down.
	txRecorder(tx).Record(ctx, actingAccount(db.Account{ID: 7, Username: "admin"}),
		act.SeedWithdrawn{SeedScope: act.SeedScope{Scope: "example.com"}})

	if tx.ctxErr == nil {
		t.Fatal("the tx-bound recorder detached its transaction's context")
	}
}
