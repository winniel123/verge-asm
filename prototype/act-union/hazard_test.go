package main

// PROTOTYPE — throwaway. Answers #1790 for map #1786. Not production code.

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestEveryVariantRenders is the falsification test the ticket asks for: a
// variant that renders an empty Subject cell, or that needs a fifth column,
// falsifies the shape before any code exists.
func TestEveryVariantRenders(t *testing.T) {
	for _, e := range catalogue {
		row, err := Encode(e.Act, e.Actor)
		if err != nil {
			t.Fatalf("%s: encode: %v", e.Act.Class(), err)
		}
		row.CreatedAt = time.Now().Add(-90 * time.Minute)
		got, err := Render(row)
		if err != nil {
			t.Fatalf("%s: render: %v", e.Act.Class(), err)
		}
		for name, cell := range map[string]string{
			"When": got.When, "Actor": got.Actor, "Action": got.Action, "Subject": got.Subject,
		} {
			if strings.TrimSpace(cell) == "" {
				t.Errorf("%s: %s cell is empty", e.Act.Class(), name)
			}
		}
	}
}

func TestVariantCount(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range catalogue {
		c := e.Act.Class()
		if seen[c] {
			t.Errorf("duplicate class %q", c)
		}
		seen[c] = true
	}
	// 60 is #1791's corrected count, plus the Q4 split of POST /coverage/retention
	// and the migration class #1805 has not ruled on.
	if len(catalogue) != 62 {
		t.Errorf("catalogue has %d variants, want 62", len(catalogue))
	}
}

func TestLabelCoversEveryClass(t *testing.T) {
	for _, e := range catalogue {
		if label[e.Act.Class()] == "" {
			t.Errorf("class %q has no Action label", e.Act.Class())
		}
	}
	for c := range label {
		if decoders[c] == nil {
			t.Errorf("label %q names no variant", c)
		}
	}
}

// TestHazard1WithdrawnSubject: an act against a Seed since withdrawn, and against
// an account since removed. The row must still render, so the Subject is a value
// and never a live join (map #1786 Settled #8).
func TestHazard1WithdrawnSubject(t *testing.T) {
	rows := []Row{}
	for _, a := range []Act{
		SeedWithdrawn{SeedScope{"10.0.0.0/8"}},
		AccountRemoved{AccountRef{9, "bob"}},
	} {
		row, err := Encode(a, Account{AccountID: 7, UsernameSnapshot: "alice"})
		if err != nil {
			t.Fatal(err)
		}
		row.CreatedAt = time.Now()
		rows = append(rows, row)
	}

	// The store is now empty: the Seed is withdrawn, account 9 is removed, and
	// account 7 has been removed too. Nothing below reads a store.
	want := []string{"10.0.0.0/8", "bob"}
	for i, row := range rows {
		got, err := Render(row)
		if err != nil {
			t.Fatal(err)
		}
		if got.Subject != want[i] {
			t.Errorf("subject = %q, want %q", got.Subject, want[i])
		}
		if got.Actor != "alice" {
			t.Errorf("actor = %q, want alice after the account is removed", got.Actor)
		}
	}
}

// TestHazard2NoFieldForASecret: ADR-0053's split is that a secret being set is
// auditable and its value is not. No variant has a field the value could go in,
// and none has a field loose enough to smuggle one.
func TestHazard2NoFieldForASecret(t *testing.T) {
	banned := []string{"secret", "password", "passphrase", "credential", "ciphertext", "plaintext", "private", "totp"}

	var walk func(t *testing.T, class string, rt reflect.Type)
	walk = func(t *testing.T, class string, rt reflect.Type) {
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				walk(t, class, f.Type)
				continue
			}
			switch f.Type.Kind() {
			case reflect.String, reflect.Int64:
			default:
				// A []byte, a map or an any is a hole a secret fits through.
				t.Errorf("%s.%s is %s; only string and int64 are admitted", class, f.Name, f.Type.Kind())
			}
			for _, b := range banned {
				if strings.Contains(strings.ToLower(f.Name), b) {
					t.Errorf("%s.%s names a secret", class, f.Name)
				}
			}
		}
	}

	for _, e := range catalogue {
		walk(t, e.Act.Class(), reflect.TypeOf(e.Act))
	}

	// The ADR-0053 variant itself: one field, and it is the provider slug.
	rt := reflect.TypeOf(SSOProviderSecretSet{}).Field(0).Type
	if rt.NumField() != 1 || rt.Field(0).Name != "Slug" {
		t.Errorf("sso.provider.secret.set payload = %v, want exactly {Slug string}", rt)
	}
}

// TestHazard3Restore: a restore replaces the corpus, then records its own
// discontinuity (map #1786 Settled #13). Its row is written after the truncation
// and its actor's snapshot survives, because Settled #8 captures a value and not
// a join (#1795 rules the variant Account).
func TestHazard3Restore(t *testing.T) {
	corpus := []Row{}
	for _, a := range []Act{SeedDeclared{SeedScope{"10.0.0.0/8"}}, ScanTriggered{ScanKindRef{"hot"}}} {
		row, _ := Encode(a, Account{AccountID: 7, UsernameSnapshot: "alice"})
		corpus = append(corpus, row)
	}
	if len(corpus) != 2 {
		t.Fatal("setup")
	}

	corpus = nil // applyRestore truncates every backup table, this corpus included

	row, err := Encode(
		RestoreApplied{ArchiveRef{"verge-2026-09-08.tar.zst", "2026-09-08T14:02Z"}},
		Account{AccountID: 7, UsernameSnapshot: "alice"},
	)
	if err != nil {
		t.Fatal(err)
	}
	row.CreatedAt = time.Now()
	corpus = append(corpus, row)

	got, err := Render(corpus[0])
	if err != nil {
		t.Fatal(err)
	}
	if got.Actor != "alice" {
		t.Errorf("actor = %q; the snapshot must survive a TRUNCATE of account", got.Actor)
	}
	if len(corpus) != 1 {
		t.Errorf("corpus holds %d rows; the discontinuity row is the whole history", len(corpus))
	}
}

// TestActorCollisionIsReal asserts a defect rather than an invariant. account
// .username is TEXT NOT NULL UNIQUE with no format check and no reserved list
// (00002_accounts.sql:9, validateCredentials at cmd/web/auth.go:1876 caps length
// only), so an account named system renders exactly as the System variant.
func TestActorCollisionIsReal(t *testing.T) {
	impostor := renderActor(Account{AccountID: 9, UsernameSnapshot: "system"})
	instance := renderActor(System{})
	if impostor != instance {
		t.Fatalf("the collision is gone: %q vs %q — reopen the finding", impostor, instance)
	}
	for _, name := range []string{"setup token", "password-reset link", "invite 12"} {
		if renderActor(Account{UsernameSnapshot: name}) != name {
			t.Errorf("%q no longer collides", name)
		}
	}
}

func TestRowCarriesNoAccountForeignKey(t *testing.T) {
	row, _ := Encode(SeedDeclared{SeedScope{"10.0.0.0/8"}}, Account{AccountID: 7, UsernameSnapshot: "alice"})
	var actor map[string]any
	if err := json.Unmarshal(row.Actor, &actor); err != nil {
		t.Fatal(err)
	}
	// The id and the snapshot both ride the payload: username is UNIQUE but
	// reusable after a delete, so neither alone is sufficient (Settled #8).
	if _, ok := actor["account_id"]; !ok {
		t.Error("actor payload drops the account id")
	}
	if actor["username"] != "alice" {
		t.Error("actor payload drops the username snapshot")
	}
	if reflect.TypeOf(Row{}).NumField() != 6 {
		t.Error("Row grew a column; the four rendered cells need six stored fields")
	}
}
