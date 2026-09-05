// Package auth holds the web listener's identity primitives (ADR-0117). It is
// deliberately database-free, so the one secret that must never reach Postgres — the
// session signing key — is produced and held here on the web-only volume (ADR-0053).
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func CheckPassword(hash, pw string) bool {
	// A bool, never an error, so a comparison failure cannot read as a successful login.
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
