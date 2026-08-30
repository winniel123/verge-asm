package transcript

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// newTestKey returns a valid 32-byte key for the seal/open tests.
func newTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, keyLen)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	return key
}

func TestSealOpenRoundTrip(t *testing.T) {
	key := newTestKey(t)

	large := make([]byte, 8192)
	if _, err := rand.Read(large); err != nil {
		t.Fatalf("generate large stream: %v", err)
	}

	cases := map[string][]byte{
		"short":          []byte("connect-outcome running local\n"),
		"binary":         {0x00, 0xff, 0x00, 0x10, 0x7f, 0x80},
		"large":          large,
		"captured-empty": {}, // non-nil, empty: a stream that was captured but held no bytes.
	}
	for name, plaintext := range cases {
		t.Run(name, func(t *testing.T) {
			sealed, err := Seal(key, plaintext)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}
			if sealed == nil {
				t.Fatal("Seal of non-nil plaintext returned nil")
			}
			opened, err := Open(key, sealed)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if opened == nil {
				t.Fatal("Open returned nil for a non-nil sealed stream")
			}
			if !bytes.Equal(opened, plaintext) {
				t.Fatalf("round-trip mismatch: got %v, want %v", opened, plaintext)
			}
		})
	}
}

func TestSealNilPreservesNull(t *testing.T) {
	key := newTestKey(t)

	sealed, err := Seal(key, nil)
	if err != nil {
		t.Fatalf("Seal(nil): %v", err)
	}
	if sealed != nil {
		t.Fatalf("Seal(nil) = %v, want nil (NULL preservation)", sealed)
	}

	opened, err := Open(key, nil)
	if err != nil {
		t.Fatalf("Open(nil): %v", err)
	}
	if opened != nil {
		t.Fatalf("Open(nil) = %v, want nil (NULL preservation)", opened)
	}
}

func TestSealFreshNonce(t *testing.T) {
	key := newTestKey(t)
	plaintext := []byte("the same bytes, sealed twice")

	first, err := Seal(key, plaintext)
	if err != nil {
		t.Fatalf("Seal first: %v", err)
	}
	second, err := Seal(key, plaintext)
	if err != nil {
		t.Fatalf("Seal second: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("two seals of identical plaintext are equal: nonce is not fresh")
	}
}

func TestOpenTamperedFailsClosed(t *testing.T) {
	key := newTestKey(t)
	sealed, err := Seal(key, []byte("stderr: token=hunter2"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	// Flip a bit in the last byte (inside the ciphertext/tag region).
	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 0x01
	if _, err := Open(key, tampered); err == nil {
		t.Fatal("Open of tampered ciphertext succeeded, want fail closed")
	}
}

func TestOpenTruncatedFailsClosed(t *testing.T) {
	key := newTestKey(t)
	sealed, err := Seal(key, []byte("some output"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	// Shorter than the nonce: cannot even split.
	if _, err := Open(key, sealed[:5]); err == nil {
		t.Fatal("Open of a too-short input succeeded, want fail closed")
	}
	// Full nonce but a truncated ciphertext: authentication must fail.
	if _, err := Open(key, sealed[:len(sealed)-4]); err == nil {
		t.Fatal("Open of a truncated ciphertext succeeded, want fail closed")
	}
}

func TestOpenWrongKeyFailsClosed(t *testing.T) {
	sealed, err := Seal(newTestKey(t), []byte("sealed under one key"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := Open(newTestKey(t), sealed); err == nil {
		t.Fatal("Open with a wrong key succeeded, want fail closed")
	}
}

func TestSealRejectsBadKeyLength(t *testing.T) {
	short := make([]byte, keyLen-1)
	if _, err := Seal(short, []byte("x")); err == nil {
		t.Fatal("Seal with an undersized key succeeded, want error")
	}
}
