package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Sign/Verify round-trips an arbitrary payload under the key, and a tampered payload,
// a tampered tag, or the wrong key all collapse to ErrInvalidSession (the OIDC login
// transaction rides this signer).
func TestSignVerifyRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	payload := []byte(`{"slug":"okta","state":"abc","nonce":"xyz"}`)

	tok := Sign(key, "sso-tx", payload)
	got, err := Verify(key, "sso-tx", tok)
	if err != nil {
		t.Fatalf("Verify of a freshly signed token: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("round-trip payload = %q, want %q", got, payload)
	}

	// A tampered payload half no longer matches the tag.
	if _, err := Verify(key, "sso-tx", "tampered."+tok[strings_IndexByte(tok, '.')+1:]); err == nil {
		t.Errorf("Verify accepted a tampered payload")
	}
	// The wrong key rejects a validly-formed token.
	if _, err := Verify([]byte("wrongwrongwrongwrongwrongwrongwr"), "sso-tx", tok); err == nil {
		t.Errorf("Verify accepted a token under the wrong key")
	}
	// The wrong domain rejects a validly-signed token — the type tag keeps one signed
	// value from verifying as another under the same key.
	if _, err := Verify(key, "other", tok); err == nil {
		t.Errorf("Verify accepted a token under the wrong domain")
	}
	// A token with no separator is malformed.
	if _, err := Verify(key, "sso-tx", "no-dot-here"); err == nil {
		t.Errorf("Verify accepted a malformed token")
	}
}

// strings_IndexByte avoids importing strings just for the tamper case above.
func strings_IndexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
	if CheckPassword("not a hash", "anything") {
		t.Fatal("malformed hash accepted a password")
	}
}

func TestLoadOrCreateKey(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state") // nonexistent subdir: must be created

	key, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(key) != keyLen {
		t.Fatalf("key length = %d, want %d", len(key), keyLen)
	}

	again, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if string(again) != string(key) {
		t.Fatal("second load returned a different key; it must persist")
	}

	info, err := os.Stat(filepath.Join(dir, keyFile))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("key file mode = %o, want 600", perm)
	}
}

func TestLoadKeyRejectsWrongLength(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, keyFile), []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateKey(dir); err == nil {
		t.Fatal("expected error on a wrong-length key file")
	}
}

func TestSessionSignVerify(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	s := Session{AccountID: 7, Kind: KindSession, ExpiresAt: now.Add(time.Hour)}

	tok, err := SignSession(key, s)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	got, err := VerifySession(key, tok, KindSession, now)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.AccountID != 7 {
		t.Fatalf("account id = %d, want 7", got.AccountID)
	}
}

func TestSessionRejections(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	valid, _ := SignSession(key, Session{AccountID: 1, Kind: KindSession, ExpiresAt: now.Add(time.Hour)})

	t.Run("tampered payload", func(t *testing.T) {
		if _, err := VerifySession(key, "x"+valid, KindSession, now); err != ErrInvalidSession {
			t.Fatalf("err = %v, want ErrInvalidSession", err)
		}
	})
	t.Run("wrong key", func(t *testing.T) {
		other := []byte("ffffffffffffffffffffffffffffffff")
		if _, err := VerifySession(other, valid, KindSession, now); err != ErrInvalidSession {
			t.Fatalf("err = %v, want ErrInvalidSession", err)
		}
	})
	t.Run("expired", func(t *testing.T) {
		if _, err := VerifySession(key, valid, KindSession, now.Add(2*time.Hour)); err != ErrInvalidSession {
			t.Fatalf("err = %v, want ErrInvalidSession", err)
		}
	})
	t.Run("wrong kind", func(t *testing.T) {
		// A password-verified pending cookie must not satisfy a full-session check.
		pending, _ := SignSession(key, Session{AccountID: 1, Kind: KindPending, ExpiresAt: now.Add(time.Hour)})
		if _, err := VerifySession(key, pending, KindSession, now); err != ErrInvalidSession {
			t.Fatalf("pending cookie accepted as a session: err = %v", err)
		}
	})
	t.Run("not a token", func(t *testing.T) {
		if _, err := VerifySession(key, "garbage", KindSession, now); err != ErrInvalidSession {
			t.Fatalf("err = %v, want ErrInvalidSession", err)
		}
	})
}

func TestTOTP(t *testing.T) {
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatalf("secret: %v", err)
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	code, err := TOTPCode(secret, now)
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if len(code) != totpDigits {
		t.Fatalf("code %q length = %d, want %d", code, len(code), totpDigits)
	}
	if !VerifyTOTP(secret, code, now) {
		t.Fatal("current code rejected")
	}
	if !VerifyTOTP(secret, code, now.Add(25*time.Second)) {
		t.Fatal("code rejected inside skew window")
	}
	if VerifyTOTP(secret, code, now.Add(90*time.Second)) {
		t.Fatal("stale code accepted well outside the window")
	}
	if VerifyTOTP(secret, "000000", now) && code != "000000" {
		t.Fatal("wrong code accepted")
	}
}

func TestTOTPKnownVector(t *testing.T) {
	// RFC 6238 SHA-1 test vector: secret "12345678901234567890" (ASCII) at
	// T=59s yields 94287082 -> 6-digit 287082.
	secret := b32.EncodeToString([]byte("12345678901234567890"))
	code, err := TOTPCode(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if code != "287082" {
		t.Fatalf("RFC 6238 vector: got %q, want 287082", code)
	}
}

func TestSetupToken(t *testing.T) {
	a, err := NewSetupToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	b, _ := NewSetupToken()
	if a == b {
		t.Fatal("two setup tokens collided")
	}
	if !TokensEqual(a, a) {
		t.Fatal("token not equal to itself")
	}
	if TokensEqual(a, b) {
		t.Fatal("distinct tokens compared equal")
	}
	if TokensEqual("", "") {
		t.Fatal("empty tokens compared equal; a closed setup must stay closed")
	}
}
