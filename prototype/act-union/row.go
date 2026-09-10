package main

// PROTOTYPE — throwaway. Answers #1790 for map #1786. Not production code.

import (
	"encoding/json"
	"fmt"
	"time"
)

// Row is the stored shape. The union rides action TEXT plus subject JSONB, which
// is transcript's variant TEXT plus outcome JSONB one corpus over (ADR-0126).
//
//	CREATE TABLE act (
//	    id          BIGSERIAL PRIMARY KEY,
//	    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
//	    actor_kind  TEXT NOT NULL CHECK (actor_kind IN ('account','grant_holder','system')),
//	    actor       JSONB NOT NULL,
//	    action      TEXT NOT NULL,
//	    subject     JSONB NOT NULL
//	);
//
// No FK to account: the eleven shipped attribution columns restrict, so copying
// them means no admin who has ever acted can be removed (map #1786 Settled #8).
//
// action carries no CHECK. A 62-token constraint needs a migration per act class
// and fails at runtime rather than in CI, so the exhaustive encoder and #1791's
// AST gate hold the set instead. This diverges from transcript's 3-token CHECK
// and the reason is the cardinality.
type Row struct {
	ID        int64
	CreatedAt time.Time
	ActorKind string
	Actor     json.RawMessage
	Action    string
	Subject   json.RawMessage
}

// Encode builds the row. It takes no instant: created_at is DEFAULT now(), the
// idiom all four sibling corpora share, so the caller cannot forge the time
// (#1788). An Act timestamps the recording, never the act.
func Encode(a Act, who Actor) (Row, error) {
	if _, ok := label[a.Class()]; !ok {
		return Row{}, fmt.Errorf("act: class %q has no rendered label", a.Class())
	}
	actor, err := json.Marshal(who.Payload())
	if err != nil {
		return Row{}, fmt.Errorf("act: marshal actor: %w", err)
	}
	subject, err := json.Marshal(a)
	if err != nil {
		return Row{}, fmt.Errorf("act: marshal subject: %w", err)
	}
	return Row{
		ActorKind: who.Kind(),
		Actor:     actor,
		Action:    a.Class(),
		Subject:   subject,
	}, nil
}

// Rendered is the four shipped columns at settings.tmpl:869-876.
type Rendered struct {
	When    string
	Actor   string
	Action  string
	Subject string
}

// Render decodes the row back through the union, so the Subject cell is built by
// the variant and never by the reader guessing at a JSON shape.
func Render(r Row) (Rendered, error) {
	mk, ok := decoders[r.Action]
	if !ok {
		return Rendered{}, fmt.Errorf("act: unknown action %q", r.Action)
	}
	a := mk()
	if err := json.Unmarshal(r.Subject, a); err != nil {
		return Rendered{}, fmt.Errorf("act: unmarshal %s: %w", r.Action, err)
	}
	who, err := decodeActor(r.ActorKind, r.Actor)
	if err != nil {
		return Rendered{}, err
	}
	return Rendered{
		When:    relative(r.CreatedAt),
		Actor:   renderActor(who),
		Action:  label[r.Action],
		Subject: a.(Act).Subject(),
	}, nil
}

func decodeActor(kind string, raw json.RawMessage) (Actor, error) {
	var p struct {
		AccountID int64  `json:"account_id"`
		Username  string `json:"username"`
		Grant     string `json:"grant"`
		InviteID  int64  `json:"invite_id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("act: unmarshal actor: %w", err)
	}
	switch kind {
	case "account":
		return Account{AccountID: p.AccountID, UsernameSnapshot: p.Username}, nil
	case "grant_holder":
		switch p.Grant {
		case "setup-token":
			return GrantHolder{SetupToken{}}, nil
		case "password-reset":
			return GrantHolder{PasswordReset{}}, nil
		case "invite":
			return GrantHolder{Invite{ID: p.InviteID}}, nil
		}
		return nil, fmt.Errorf("act: unknown grant %q", p.Grant)
	case "system":
		return System{}, nil
	}
	return nil, fmt.Errorf("act: unknown actor kind %q", kind)
}

func relative(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
