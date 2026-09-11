package main

import (
	"context"
	"log"
	"time"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

type actStore interface {
	InsertAct(ctx context.Context, arg db.InsertActParams) error
}

// The act has already committed, so a slow insert may not hold the response (spec §7.6).

const actRecordBudget = 5 * time.Second

// One method name, two receivers: the pool binds it, the restore a tx (spec §7.1).

type recorder struct {
	store actStore
	inTx  bool
}

func (s *server) recorder() recorder { return recorder{store: s.actStore} }

// The restore's Act commits with the replay it records, spec §7.6's one exception (#1834).

func txRecorder(tx db.DBTX) recorder { return recorder{store: db.New(tx), inTx: true} }

// Only a signed-in session reaches an admin route, so the grant_holder member is the callback's.

func actingAccount(acct db.Account) act.Actor {
	// The name is a captured value, never a join: an Act outlives the account (spec §5.4).
	return act.Account{AccountID: acct.ID, UsernameSnapshot: acct.Username}
}

// The pool caller may discard the error: no durable notice can be built (spec §7.7).

func (rec recorder) Record(ctx context.Context, actor act.Actor, a act.Act) error {
	// A detached tx-bound insert outlives the rollback that tears its own tx down.
	if !rec.inTx {
		var cancel context.CancelFunc
		// On r.Context() the row dies when the operator navigates away (spec §7.6).
		ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), actRecordBudget)
		defer cancel()
	}

	kind, actorJSON, err := act.EncodeActor(actor)
	if err != nil {
		log.Printf("web: act: encode actor: %v", err)
		return err
	}
	action, subject, err := act.EncodeSubject(a)
	if err != nil {
		log.Printf("web: act: encode subject: %v", err)
		return err
	}
	// No retry: a second failure leaves the identical hole (spec §7.6).
	if err := rec.store.InsertAct(ctx, db.InsertActParams{
		ActorKind: kind,
		Actor:     actorJSON,
		Action:    action,
		Subject:   subject,
	}); err != nil {
		log.Printf("web: act: record %q: %v", action, err)
		return err
	}
	return nil
}
