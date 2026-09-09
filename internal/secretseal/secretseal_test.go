package secretseal

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestSealTextAndOpenText(t *testing.T) {
	key, _ := DeriveKey(parent, LabelChannelSecret)
	if v, err := SealText(key, "   "); err != nil || v.Valid {
		t.Fatalf("SealText(blank) = %+v, %v; want unset, nil", v, err)
	}
	stored, err := SealText(key, " keep-my-spaces ")
	if err != nil || !stored.Valid {
		t.Fatalf("SealText = %+v, %v", stored, err)
	}
	got, err := OpenText(key, stored)
	if err != nil || string(got) != " keep-my-spaces " {
		t.Fatalf("OpenText = %q, %v", got, err)
	}
	if got, err := OpenText(key, pgtype.Text{}); err != nil || got != nil {
		t.Fatalf("OpenText(NULL) = %v, %v; want nil, nil", got, err)
	}
	if _, err := OpenText(key, pgtype.Text{String: "cleartext", Valid: true}); err == nil {
		t.Fatal("OpenText accepted a cleartext row")
	}
}

var parent = []byte("transcriptkey0123456789abcdef012")

func TestDeriveKeyIsDeterministicAndLabelSeparated(t *testing.T) {
	a, err := DeriveKey(parent, LabelChannelSecret)
	if err != nil {
		t.Fatal(err)
	}
	b, err := DeriveKey(parent, LabelChannelSecret)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("the same parent and label derived two different keys")
	}
	c, err := DeriveKey(parent, LabelSSOClientSecret)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) == string(c) {
		t.Fatal("two labels derived the same key")
	}
	if len(a) != 32 {
		t.Fatalf("derived key is %d bytes, want 32", len(a))
	}
}

func TestDeriveKeyRefusesEmptyParent(t *testing.T) {
	if _, err := DeriveKey(nil, LabelChannelSecret); err == nil {
		t.Fatal("a nil parent key derived a sub-key; want an error")
	}
	if _, err := DeriveKey(parent, ""); err == nil {
		t.Fatal("an empty label derived a sub-key; want an error")
	}
}

func TestSealOpenRoundTripHidesTheValue(t *testing.T) {
	key, err := DeriveKey(parent, LabelChannelSecret)
	if err != nil {
		t.Fatal(err)
	}
	const secret = "sign-me-please"
	stored, err := Seal(key, secret)
	if err != nil {
		t.Fatal(err)
	}
	if stored == secret || strings.Contains(stored, secret) {
		t.Fatalf("sealed value contains the secret: %q", stored)
	}
	if _, err := base64.StdEncoding.DecodeString(stored); err != nil {
		t.Fatalf("sealed value is not base64: %v", err)
	}
	again, err := Seal(key, secret)
	if err != nil {
		t.Fatal(err)
	}
	if again == stored {
		t.Fatal("two seals of one value produced one ciphertext; the nonce is not random")
	}
	got, err := Open(key, stored)
	if err != nil {
		t.Fatal(err)
	}
	if got != secret {
		t.Fatalf("Open = %q, want %q", got, secret)
	}
}

func TestSealOpenKeepEmptyEmpty(t *testing.T) {
	key, _ := DeriveKey(parent, LabelChannelSecret)
	stored, err := Seal(key, "")
	if err != nil || stored != "" {
		t.Fatalf("Seal(\"\") = %q, %v; want empty, nil", stored, err)
	}
	got, err := Open(key, "")
	if err != nil || got != "" {
		t.Fatalf("Open(\"\") = %q, %v; want empty, nil", got, err)
	}
}

func TestOpenFailsClosed(t *testing.T) {
	key, _ := DeriveKey(parent, LabelChannelSecret)
	other, _ := DeriveKey(parent, LabelSSOClientSecret)
	stored, err := Seal(key, "sign-me")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		key    []byte
		stored string
	}{
		"legacy cleartext":     {key, "sign-me"},
		"cleartext base64":     {key, base64.StdEncoding.EncodeToString([]byte("sign-me"))},
		"wrong key":            {other, stored},
		"nil key":              {nil, stored},
		"truncated ciphertext": {key, stored[:len(stored)-8]},
	}
	for name, c := range cases {
		got, err := Open(c.key, c.stored)
		if err == nil {
			t.Errorf("%s: Open returned %q with no error; want failure", name, got)
		}
		if got != "" {
			t.Errorf("%s: Open returned a value %q on failure", name, got)
		}
	}
}

func TestSealRefusesNilKey(t *testing.T) {
	if _, err := Seal(nil, "sign-me"); err == nil {
		t.Fatal("Seal under a nil key succeeded; want an error so no cleartext reaches the store")
	}
}
