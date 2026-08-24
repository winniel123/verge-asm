// Package auth holds the identity primitives for the web listener: password
// hashing, the file-backed session signing key, HMAC-signed session cookies
// (backed by the server-side session registry, ADR-0117), per-account TOTP,
// and the single-use setup token. It is
// deliberately database-free so the whole of it is unit-testable, and so the
// one auth secret that must never reach Postgres — the session signing key —
// is produced and held here, read from the web-only volume (v1 spec §4.3).
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of pw at the default cost.
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword reports whether pw matches the bcrypt hash. A malformed hash
// or a mismatch both return false; it never errors, so a caller cannot
// accidentally treat a comparison failure as a successful login.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
