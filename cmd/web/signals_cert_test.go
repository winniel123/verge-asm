package main

import (
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/signal"
)

// certBoolPtr is a local *bool helper for the v3 chain facts under test.
func certBoolPtr(b bool) *bool { return &b }

// sanMatchesName is the RFC 6125 §6.4.3 rule-2 predicate: an OR over the dNSName SANs
// only, with a single leftmost `*` wildcard covering exactly one non-empty label and
// every other `*`-bearing entry a non-match. It never defaults false to manufacture a
// mismatch — its caller gates it on the leaf actually having been read (P0.10b, #714).
func TestSANMatchesName(t *testing.T) {
	tests := []struct {
		name    string
		sanDNS  []string
		host    string
		want    bool
		comment string
	}{
		{"literal-match", []string{"www.example.com"}, "www.example.com", true, "exact literal SAN"},
		{"literal-nonmatch", []string{"www.example.com"}, "api.example.com", false, "different leftmost label"},
		{"wildcard-leftmost-match", []string{"*.example.com"}, "foo.example.com", true, "leftmost * covers one label"},
		{"wildcard-apex-noncover", []string{"*.example.com"}, "example.com", false, "* does NOT cover the apex"},
		{"wildcard-deeper-noncover", []string{"*.example.com"}, "bar.foo.example.com", false, "* covers exactly one label, not two"},
		{"wildcard-nonleftmost-nonmatch", []string{"bar.*.example.com"}, "bar.foo.example.com", false, "* is not the leftmost label"},
		{"wildcard-partial-nonmatch", []string{"baz*.example.com"}, "bazfoo.example.com", false, "partial-label wildcard never matches"},
		{"wildcard-partial-prefix-nonmatch", []string{"*baz.example.com"}, "foobaz.example.com", false, "partial-label wildcard never matches"},
		{"double-star-nonmatch", []string{"**.example.com"}, "foo.bar.example.com", false, "multiple stars never match"},
		{"ip-only-empty-san-dns", nil, "www.example.com", false, "no dNSName SANs (IP-only leaf) → false, never nil-defaulted true"},
		{"empty-disjunction", []string{}, "www.example.com", false, "no entries → false"},
		{"case-fold", []string{"WWW.Example.COM"}, "www.example.com", true, "ASCII case-insensitive labels"},
		{"case-fold-wildcard", []string{"*.EXAMPLE.com"}, "FOO.example.COM", true, "fold on both wildcard tail and name"},
		{"trailing-dot-san", []string{"www.example.com."}, "www.example.com", true, "trailing dot on SAN ignored"},
		{"trailing-dot-name", []string{"www.example.com"}, "www.example.com.", true, "trailing dot on name ignored"},
		{"or-over-entries", []string{"api.example.com", "*.example.com"}, "www.example.com", true, "any entry matching is a match"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanMatchesName(tc.sanDNS, tc.host); got != tc.want {
				t.Errorf("sanMatchesName(%q, %q) = %v, want %v (%s)", tc.sanDNS, tc.host, got, tc.want, tc.comment)
			}
		})
	}
}

// selfSignedOf is RFC 5280 §3.2 self-signed: self-issued (subject == issuer, exact
// string equality) AND the cert's own signature verifies. Both limbs are required —
// DN-equality alone or a verifying signature alone is not self-signed (T-self #713).
func TestSelfSignedOf(t *testing.T) {
	tests := []struct {
		name            string
		subject, issuer string
		selfSigVerifies bool
		want            bool
	}{
		{"both-limbs", "CN=root", "CN=root", true, true},
		{"dn-eq-only", "CN=root", "CN=root", false, false},
		{"verify-only", "CN=leaf", "CN=ca", true, false},
		{"neither", "CN=leaf", "CN=ca", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := selfSignedOf(tc.subject, tc.issuer, tc.selfSigVerifies); got != tc.want {
				t.Errorf("selfSignedOf(%q,%q,%v) = %v, want %v", tc.subject, tc.issuer, tc.selfSigVerifies, got, tc.want)
			}
		})
	}
}

// weakKeyOrSignature is the per-link deny-list walk (T-weak #715 §5): ANY link with a
// weak KEY (RSA/DSA<2048, ECDSA<224, DSA N<224) OR a weak SIGNATURE digest ({MD5,SHA-1})
// makes the chain weak. The KEY limb walks every link; the SIGNATURE limb is skipped on
// self-signed links (a root's SHA-1 self-signature is not a forgery risk) — but the same
// root's weak KEY still fires. Strict `<` floors; unnamed algorithms are never weak.
func TestWeakKeyOrSignature(t *testing.T) {
	// A strong leaf whose signature is issued by a CA (not self-signed), reused as the
	// clean baseline the negative cases assert against.
	strongLeaf := chainCert{
		Subject: "CN=leaf", Issuer: "CN=ca",
		SelfSignatureVerifies: certBoolPtr(false),
		KeyAlg:                "RSA", KeyBits: 2048, SigDigest: "SHA-256",
	}

	tests := []struct {
		name  string
		chain []chainCert
		want  bool
	}{
		{"rsa-below-floor-fires", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "RSA", KeyBits: 1024, SigDigest: "SHA-256"}}, true},
		{"rsa-at-floor-clean", []chainCert{strongLeaf}, false},
		{"ecdsa-below-floor-fires", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "ECDSA", KeyBits: 192, SigDigest: "SHA-256"}}, true},
		{"ecdsa-at-floor-clean", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "ECDSA", KeyBits: 224, SigDigest: "SHA-256"}}, false},
		{"dsa-L-below-floor-fires", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "DSA", KeyBits: 1024, KeyParamN: 224, SigDigest: "SHA-256"}}, true},
		{"dsa-N-below-floor-fires", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "DSA", KeyBits: 2048, KeyParamN: 160, SigDigest: "SHA-256"}}, true},
		{"dsa-at-floors-clean", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "DSA", KeyBits: 2048, KeyParamN: 224, SigDigest: "SHA-256"}}, false},
		{"md5-sig-fires", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "MD5"}}, true},
		{"sha1-sig-fires", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-1"}}, true},
		{
			"self-signed-sha1-root-sig-skipped-but-key-clean",
			[]chainCert{{Subject: "CN=root", Issuer: "CN=root", SelfSignatureVerifies: certBoolPtr(true), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-1"}},
			false, // SHA-1 sig limb skipped on the self-signed root; its 2048-bit key is clean
		},
		{
			"self-signed-sha1-root-weak-key-still-fires",
			[]chainCert{{Subject: "CN=root", Issuer: "CN=root", SelfSignatureVerifies: certBoolPtr(true), KeyAlg: "RSA", KeyBits: 1024, SigDigest: "SHA-1"}},
			true, // sig limb skipped, but the weak KEY limb has NO self-signed skip
		},
		{"ed25519-not-weak", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "Ed25519", SigDigest: "Ed25519"}}, false},
		{"unknown-alg-not-weak", []chainCert{{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "UnknownAlg", SigDigest: ""}}, false},
		{
			"weak-intermediate-in-strong-chain-fires",
			[]chainCert{
				strongLeaf,
				{Subject: "CN=int", Issuer: "CN=root", SelfSignatureVerifies: certBoolPtr(false), KeyAlg: "RSA", KeyBits: 1024, SigDigest: "SHA-256"},
				{Subject: "CN=root", Issuer: "CN=root", SelfSignatureVerifies: certBoolPtr(true), KeyAlg: "RSA", KeyBits: 4096, SigDigest: "SHA-256"},
			},
			true, // the 1024-bit intermediate makes the whole chain weak (chain scope)
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := weakKeyOrSignature(tc.chain); got != tc.want {
				t.Errorf("weakKeyOrSignature = %v, want %v", got, tc.want)
			}
		})
	}
}

// certDetailsFromValue sets each *bool INDEPENDENTLY only when its own input is
// present: a negative outcome is nil (outside every cert rule); a pre-v3 presented span
// leaves the four v3 attributes nil (not-evaluable, never defaulted); a full v3
// presented value sets all six (P0.10a/P0.10b, collision #37 / #704).
func TestCertDetailsFromValueNilDiscipline(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)

	t.Run("negative-outcome-nil", func(t *testing.T) {
		if d := certDetailsFromValue(certificateValue{Outcome: signal.CertNoTLS}, now, "www.example.com"); d != nil {
			t.Errorf("a negative outcome must yield nil CertDetails, got %+v", d)
		}
		if d := certDetailsFromValue(certificateValue{Outcome: signal.CertTLSRefused}, now, "www.example.com"); d != nil {
			t.Errorf("tls-refused must yield nil CertDetails, got %+v", d)
		}
	})

	t.Run("pre-v3-presented-span-v3-attrs-nil", func(t *testing.T) {
		// A pre-v3 span: presented with only a not_after, no not_before / SANs / chain.
		v := certificateValue{
			Outcome:  signal.CertPresented,
			Chain:    []string{"sha256:abc"},
			NotAfter: now.Add(90 * 24 * time.Hour).Format(time.RFC3339),
		}
		d := certDetailsFromValue(v, now, "www.example.com")
		if d == nil {
			t.Fatal("a presented span must yield non-nil CertDetails")
		}
		// Expired/Expiring set off not_after; the four v3 attributes stay nil.
		if d.Expired == nil || d.Expiring == nil {
			t.Errorf("not_after present → Expired/Expiring must be set, got %+v", d)
		}
		if d.NotYetValid != nil {
			t.Errorf("no not_before → NotYetValid must be nil, got %v", *d.NotYetValid)
		}
		if d.SANMatchesName != nil {
			t.Errorf("no chain_certs → SANMatchesName must be nil (never defaulted), got %v", *d.SANMatchesName)
		}
		if d.WeakKeyOrSignature != nil {
			t.Errorf("no chain_certs → WeakKeyOrSignature must be nil, got %v", *d.WeakKeyOrSignature)
		}
		if d.SelfSigned != nil {
			t.Errorf("no chain_certs → SelfSigned must be nil, got %v", *d.SelfSigned)
		}
	})

	t.Run("v3-presented-attrs-set", func(t *testing.T) {
		// A full v3 presented value: not-yet-valid leaf, SAN covers the name, weak key,
		// self-signed leaf.
		v := certificateValue{
			Outcome:   signal.CertPresented,
			Chain:     []string{"sha256:abc"},
			NotAfter:  now.Add(90 * 24 * time.Hour).Format(time.RFC3339),
			NotBefore: now.Add(24 * time.Hour).Format(time.RFC3339), // starts tomorrow → not yet valid
			SANDNS:    []string{"www.example.com"},
			ChainCerts: []chainCert{{
				Subject:               "CN=www.example.com",
				Issuer:                "CN=www.example.com",
				SelfSignatureVerifies: certBoolPtr(true),
				KeyAlg:                "RSA",
				KeyBits:               1024, // weak
				SigDigest:             "SHA-256",
			}},
		}
		d := certDetailsFromValue(v, now, "www.example.com")
		if d == nil {
			t.Fatal("a v3 presented value must yield non-nil CertDetails")
		}
		if d.NotYetValid == nil || !*d.NotYetValid {
			t.Errorf("not_before in the future → NotYetValid must be true, got %v", d.NotYetValid)
		}
		if d.SANMatchesName == nil || !*d.SANMatchesName {
			t.Errorf("SAN covers the name → SANMatchesName must be true, got %v", d.SANMatchesName)
		}
		if d.WeakKeyOrSignature == nil || !*d.WeakKeyOrSignature {
			t.Errorf("1024-bit RSA key → WeakKeyOrSignature must be true, got %v", d.WeakKeyOrSignature)
		}
		if d.SelfSigned == nil || !*d.SelfSigned {
			t.Errorf("subject==issuer & self-sig verifies → SelfSigned must be true, got %v", d.SelfSigned)
		}
	})

	t.Run("v3-presented-empty-server-name-leaves-san-nil", func(t *testing.T) {
		// The nameless endpoint: chain read but no server name → SANMatchesName stays nil
		// (outside certificate-hostname-san-mismatch's domain), while the chain-derived
		// weak-key and self-signed attributes are still set.
		v := certificateValue{
			Outcome: signal.CertPresented,
			Chain:   []string{"sha256:abc"},
			SANDNS:  []string{"www.example.com"},
			ChainCerts: []chainCert{{
				Subject:               "CN=leaf",
				Issuer:                "CN=ca",
				SelfSignatureVerifies: certBoolPtr(false),
				KeyAlg:                "RSA",
				KeyBits:               2048,
				SigDigest:             "SHA-256",
			}},
		}
		d := certDetailsFromValue(v, now, "")
		if d == nil {
			t.Fatal("a v3 presented value must yield non-nil CertDetails")
		}
		if d.SANMatchesName != nil {
			t.Errorf("empty server name → SANMatchesName must be nil, got %v", *d.SANMatchesName)
		}
		if d.WeakKeyOrSignature == nil || *d.WeakKeyOrSignature {
			t.Errorf("2048-bit RSA, SHA-256 → WeakKeyOrSignature must be set false, got %v", d.WeakKeyOrSignature)
		}
		if d.SelfSigned == nil || *d.SelfSigned {
			t.Errorf("CA-issued leaf → SelfSigned must be set false, got %v", d.SelfSigned)
		}
	})
}
