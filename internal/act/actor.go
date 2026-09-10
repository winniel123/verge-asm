// Package act holds the Act corpus: one recorded act by one principal on this
// instance, the fifth Operational corpus beside Dispatch, Message, Delivery and
// Transcript. See docs/spec/audit-act.md.
//
// Nothing here reads a store. An Act may outlive its subject, so the recorder
// resolves every value before the act and this package renders what it stored.
package act

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

const sep = " · "

// It discriminates on how the principal proved themselves, never on who the row is about (§3.1).

type Actor interface { // a closed union, never a nullable account_id, because null is a shape (§3)
	Kind() string
	Name() string // the @ mark is a renderer's and lands with the reader, never this (§3.5)
	isActor()
}

// 'system' is absent because no migration writes an Act, so the variant is uninhabited (§3.4).

const (
	KindAccount     = "account"
	KindGrantHolder = "grant_holder"
)

// It asserts a signed-in session for this account acted, and no more (§3.2).

type Account struct {
	AccountID        int64  `json:"account_id"`
	UsernameSnapshot string `json:"username_snapshot"` // reusable after a delete (§5.4)
}

func (Account) Kind() string   { return KindAccount }
func (a Account) Name() string { return a.UsernameSnapshot }
func (Account) isActor()       {}

// It asserts A holder acted, never the account's owner, and carries no account (§3.2, §3.3).

type GrantHolder struct {
	Grant Grant `json:"grant"`
}

func (GrantHolder) Kind() string   { return KindGrantHolder }
func (g GrantHolder) Name() string { return g.Grant.Label() }
func (GrantHolder) isActor()       {}

// A variant carries the grant's id exactly when a second Act names the same grant (§3.3).

type Grant interface {
	Label() string // no authored label begins with @, which makes disjointness structural (§3.5)
	tag() string
	isGrant()
}

// No row exists: the token is an in-memory string compared by auth.TokensEqual (§3.3).

type SetupToken struct{}

// Every POST /forgot deletes the consumed row, and POST /forgot is exempt (§3.3).

type PasswordReset struct{}

// POST /settings/accounts is audited too, so the id joins the mint to the acceptance (§3.3).

type Invite struct {
	InviteID int64 `json:"invite_id"`
}

func (SetupToken) Label() string    { return "setup token" }
func (PasswordReset) Label() string { return "password-reset link" }
func (i Invite) Label() string      { return "invite " + strconv.FormatInt(i.InviteID, 10) }

func (SetupToken) tag() string    { return grantSetupToken }
func (PasswordReset) tag() string { return grantPasswordReset }
func (Invite) tag() string        { return grantInvite }

func (SetupToken) isGrant()    {}
func (PasswordReset) isGrant() {}
func (Invite) isGrant()        {}

// The stored tag is distinct from the label, so a copy change rewrites no row (§4.1).

const (
	grantSetupToken    = "setup_token"
	grantPasswordReset = "password_reset"
	grantInvite        = "invite"
)

var grants = map[string]reflect.Type{
	grantSetupToken:    reflect.TypeOf(SetupToken{}),
	grantPasswordReset: reflect.TypeOf(PasswordReset{}),
	grantInvite:        reflect.TypeOf(Invite{}),
}

// The tag rides the payload, so the decoder picks a variant rather than guessing one.

type grantEnvelope struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data,omitempty"`
}

func (g GrantHolder) MarshalJSON() ([]byte, error) {
	if g.Grant == nil {
		return nil, fmt.Errorf("act: a GrantHolder carrying no grant")
	}
	tag := g.Grant.tag()
	if _, known := grants[tag]; !known {
		// An unregistered tag writes a row no decoder can read, and the corpus never deletes one.
		return nil, fmt.Errorf("act: grant %T tags %q, which grants does not register", g.Grant, tag)
	}
	data, err := json.Marshal(g.Grant)
	if err != nil {
		return nil, fmt.Errorf("act: marshal grant %T: %w", g.Grant, err)
	}
	return json.Marshal(grantEnvelope{Kind: tag, Data: data})
}

func (g *GrantHolder) UnmarshalJSON(b []byte) error {
	var env grantEnvelope
	if err := json.Unmarshal(b, &env); err != nil {
		return fmt.Errorf("act: decode grant envelope: %w", err)
	}
	t, ok := grants[env.Kind]
	if !ok {
		// A closed union we author errors on an unknown tag, never a mislabelled row (ADR-0209 §1).
		return fmt.Errorf("act: unknown grant kind %q", env.Kind)
	}
	// A pointer to the concrete type, because json cannot fill a pointer to an interface.
	p := reflect.New(t)
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, p.Interface()); err != nil {
			return fmt.Errorf("act: decode grant %q: %w", env.Kind, err)
		}
	}
	g.Grant = p.Elem().Interface().(Grant)
	return nil
}

func EncodeActor(a Actor) (string, []byte, error) {
	switch v := a.(type) {
	case Account:
		payload, err := json.Marshal(v)
		if err != nil {
			return "", nil, fmt.Errorf("act: marshal account actor: %w", err)
		}
		return v.Kind(), payload, nil
	case GrantHolder:
		payload, err := json.Marshal(v)
		if err != nil {
			return "", nil, fmt.Errorf("act: marshal grant_holder actor: %w", err)
		}
		return v.Kind(), payload, nil
	case *Account:
		// Every method takes a value receiver, so a pointer satisfies Actor and would drop a row.
		if v == nil {
			return "", nil, fmt.Errorf("act: a nil *Account is not an actor")
		}
		return EncodeActor(*v)
	case *GrantHolder:
		if v == nil {
			return "", nil, fmt.Errorf("act: a nil *GrantHolder is not an actor")
		}
		return EncodeActor(*v)
	default:
		// A closed union we author errors on an unknown member (ADR-0209 §1).
		return "", nil, fmt.Errorf("act: actor variant %T is not a member of the union", a)
	}
}

func DecodeActor(kind string, payload []byte) (Actor, error) {
	switch kind {
	case KindAccount:
		var v Account
		if err := json.Unmarshal(payload, &v); err != nil {
			return nil, fmt.Errorf("act: decode account actor: %w", err)
		}
		return v, nil
	case KindGrantHolder:
		var v GrantHolder
		if err := json.Unmarshal(payload, &v); err != nil {
			return nil, fmt.Errorf("act: decode grant_holder actor: %w", err)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("act: unknown actor_kind %q", kind)
	}
}
