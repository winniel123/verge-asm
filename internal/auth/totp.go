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

// totpMod is 10^totpDigits, the modulus that truncates the HOTP value to
// totpDigits digits. It is derived from the constant so the width and the
// modulus cannot drift apart.
var totpMod = func() uint32 {
	m := uint32(1)
	for i := 0; i < totpDigits; i++ {
		m *= 10
	}
	return m
}()

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
// once, not once per window. It is the stateless check the enrollment-confirm
// path uses; the replay-guarded login path uses VerifyTOTPStep to learn and
// persist which step matched.
func VerifyTOTP(secret, code string, t time.Time) bool {
	_, ok := VerifyTOTPStep(secret, code, t)
	return ok
}

// VerifyTOTPStep reports whether code is valid for secret at time t and, when it
// is, the counter step it matched — the RFC 6238 time-step (unix / period) of the
// accepted window. The step is a monotonic single-use handle: the login path
// records the last step it accepted per account and refuses any code whose step is
// not strictly greater, so a captured valid code cannot be replayed within its
// ~90s validity window (#323, RFC 6238 §5.2). Like VerifyTOTP it accepts the
// immediately-adjacent steps for clock skew and compares in constant time. On a
// non-match the returned step is 0 and must be ignored.
func VerifyTOTPStep(secret, code string, t time.Time) (step int64, ok bool) {
	key, err := decodeSecret(secret)
	if err != nil {
		return 0, false
	}
	code = strings.TrimSpace(code)
	for _, skew := range []time.Duration{-totpPeriod, 0, totpPeriod} {
		at := t.Add(skew)
		want := codeFromKey(key, at)
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return stepFor(at), true
		}
	}
	return 0, false
}

// stepFor is the RFC 6238 time-step covering t: the counter codeFromKey derives
// its HOTP value from. It is exposed to the login path as the per-account replay
// watermark, so it must stay the exact expression codeFromKey uses.
func stepFor(t time.Time) int64 {
	return int64(uint64(t.Unix()) / uint64(totpPeriod.Seconds()))
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
	counter := uint64(stepFor(t))

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
	return fmt.Sprintf("%0*d", totpDigits, bin%totpMod)
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
