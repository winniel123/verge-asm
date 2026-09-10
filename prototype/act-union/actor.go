package main

// PROTOTYPE — throwaway. Answers #1790 for map #1786. Not production code.

import "fmt"

// Actor is #1795's closed union. It discriminates on how the principal proved
// themselves, at the grain of the kind of proof and never its strength, so a
// password, SSO and TOTP-stepped session are one Account.
type Actor interface {
	Kind() string
	Payload() map[string]any
	isActor()
}

// Account carries the username as it stood at the time, because account.username
// is UNIQUE but reusable after a delete (map #1786 Settled #8).
type Account struct {
	AccountID        int64
	UsernameSnapshot string
}

// GrantHolder is #1795's rename of Bootstrap. A human holds a setup token, so
// the old name asserted a non-human principal that was never there.
type GrantHolder struct{ Grant Grant }

// System asserts that no principal took this act. It stays bare: #1795 barred a
// discriminator on it.
type System struct{}

func (Account) isActor()     {}
func (GrantHolder) isActor() {}
func (System) isActor()      {}

func (a Account) Kind() string   { return "account" }
func (GrantHolder) Kind() string { return "grant_holder" }
func (System) Kind() string      { return "system" }

func (a Account) Payload() map[string]any {
	return map[string]any{"account_id": a.AccountID, "username": a.UsernameSnapshot}
}

func (g GrantHolder) Payload() map[string]any {
	p := map[string]any{"grant": g.Grant.grant()}
	if inv, ok := g.Grant.(Invite); ok {
		// #1795's correlation rule: carry the id exactly when a second Act names
		// the same grant. A consumed reset is deleted by every POST /forgot.
		p["invite_id"] = inv.ID
	}
	return p
}

func (System) Payload() map[string]any { return map[string]any{} }

type Grant interface {
	grant() string
	isGrant()
}

type SetupToken struct{}
type PasswordReset struct{}
type Invite struct{ ID int64 }

func (SetupToken) isGrant()    {}
func (PasswordReset) isGrant() {}
func (Invite) isGrant()        {}

func (SetupToken) grant() string    { return "setup-token" }
func (PasswordReset) grant() string { return "password-reset" }
func (Invite) grant() string        { return "invite" }

// renderActor builds the Actor cell. A closed union we author errors on an
// unknown member, never a mislabelled row (ADR-0209 §1).
func renderActor(a Actor) string {
	switch v := a.(type) {
	case Account:
		return v.UsernameSnapshot
	case GrantHolder:
		switch g := v.Grant.(type) {
		case SetupToken:
			return "setup token"
		case PasswordReset:
			return "password-reset link"
		case Invite:
			return fmt.Sprintf("invite %d", g.ID)
		default:
			panic(fmt.Sprintf("unknown grant %T", g))
		}
	case System:
		return "system"
	default:
		panic(fmt.Sprintf("unknown actor %T", a))
	}
}
