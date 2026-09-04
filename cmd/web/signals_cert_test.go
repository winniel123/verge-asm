package main

import (
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/signal"
)

func certBoolPtr(b bool) *bool { return &b }

func TestSANMatchesName(t *testing.T) {
	// The wildcard cases are RFC 6125 §6.4.3.
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

func TestSelfSignedOf(t *testing.T) {
	// The two limbs are RFC 5280 §3.2's own definition of self-signed.
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

func TestWeakKeyOrSignature(t *testing.T) {
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
			false,
		},
		{
			"self-signed-sha1-root-weak-key-still-fires",
			[]chainCert{{Subject: "CN=root", Issuer: "CN=root", SelfSignatureVerifies: certBoolPtr(true), KeyAlg: "RSA", KeyBits: 1024, SigDigest: "SHA-1"}},
			true,
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
			true,
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
		v := certificateValue{
			Outcome:  signal.CertPresented,
			Chain:    []string{"sha256:abc"},
			NotAfter: now.Add(90 * 24 * time.Hour).Format(time.RFC3339),
		}
		d := certDetailsFromValue(v, now, "www.example.com")
		if d == nil {
			t.Fatal("a presented span must yield non-nil CertDetails")
		}
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
		v := certificateValue{
			Outcome:   signal.CertPresented,
			Chain:     []string{"sha256:abc"},
			NotAfter:  now.Add(90 * 24 * time.Hour).Format(time.RFC3339),
			NotBefore: now.Add(24 * time.Hour).Format(time.RFC3339),
			SANDNS:    []string{"www.example.com"},
			ChainCerts: []chainCert{{
				Subject:               "CN=www.example.com",
				Issuer:                "CN=www.example.com",
				SelfSignatureVerifies: certBoolPtr(true),
				KeyAlg:                "RSA",
				KeyBits:               1024,
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
