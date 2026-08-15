package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// TOTP parameters. These are the values every authenticator app defaults to
// (RFC 6238): SHA-1, 6 digits, a 30-second step. They are fixed, not dials —
// changing them would silently invalidate every enrolled device.
const (
	totpDigits      = 6
	totpPeriod      = 30 * time.Second
	totpSecretBytes = 20 // 160 bits, the RFC 4226 recommendation
)

// b32 is unpadded base32, the encoding authenticator apps expect for the
// shared secret.
var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret returns a fresh random base32 TOTP secret.
func NewTOTPSecret() (string, error) {
	buf := make([]byte, totpSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: generate totp secret: %w", err)
	}
	return b32.EncodeToString(buf), nil
}

// TOTPCode returns the 6-digit code for secret at time t. It is the building
// block VerifyTOTP compares against; it is exported so a caller can render the
// current code in a test-only or diagnostic path.
func TOTPCode(secret string, t time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	return codeFromKey(key, t), nil
}

// VerifyTOTP reports whether code is valid for secret at time t, accepting the
// immediately-adjacent steps as well to tolerate clock skew and a code typed
// as its window closes. The comparison is constant-time. The secret is decoded
// once, not once per window.
func VerifyTOTP(secret, code string, t time.Time) bool {
	key, err := decodeSecret(secret)
	if err != nil {
		return false
	}
	code = strings.TrimSpace(code)
	for _, skew := range []time.Duration{-totpPeriod, 0, totpPeriod} {
		want := codeFromKey(key, t.Add(skew))
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

func decodeSecret(secret string) ([]byte, error) {
	key, err := b32.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return nil, fmt.Errorf("auth: decode totp secret: %w", err)
	}
	return key, nil
}

// codeFromKey computes the HOTP value for the decoded key at the step covering
// t (RFC 4226 dynamic truncation, §5.3).
func codeFromKey(key []byte, t time.Time) string {
	counter := uint64(t.Unix()) / uint64(totpPeriod.Seconds())

	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	return fmt.Sprintf("%0*d", totpDigits, bin%1_000_000)
}

// OtpauthURI builds the otpauth:// URI an authenticator app consumes via QR
// code, naming issuer and account so the entry is identifiable.
func OtpauthURI(secret, account, issuer string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("digits", fmt.Sprintf("%d", totpDigits))
	q.Set("period", fmt.Sprintf("%d", int(totpPeriod.Seconds())))
	return "otpauth://totp/" + label + "?" + q.Encode()
}
