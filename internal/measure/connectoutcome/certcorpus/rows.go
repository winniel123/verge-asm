package certcorpus

import (
	"time"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

type Step struct {
	Batch     string
	Scope     co.Scope
	Connect   *scriptConnector
	Handshake *scriptHandshaker
}

type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Profile      co.SafetyProfile
	Step         Step
	Golden       string
}

var AllCells = []string{
	// Every cell owes a row, and each row's Claim states what it pins (golden-corpus.md §4).
	"T1/presented", "T1/tls-refused", "T1/no-tls",
	"T2/chain-leaf-first",
	"T2/leaf-not-after",
	"T3/named-sni", "T3/nameless-no-sni",
	"T4/rides-reached", "T4/not-reached-no-cert",
	"T5/one-cert-per-endpoint",
	"T6/not-before",
	"T6/san-dns",
	"T6/san-wildcard-leftmost",
	"T6/san-wildcard-apex-noncover",
	"T6/san-partial-nonmatch",
	"T6/san-ip-only",
	"T6/chain-key-params",
	"T6/weak-intermediate",
	"T6/sig-digest",
	"T6/self-sig-verifies",
	"T6/self-signed-leaf",
	"T6/self-signed-root-skip",
}

func profile() co.SafetyProfile { return co.DefaultProfile() }

func b(v bool) *bool { return &v }

func scope(addrs []string, tcp []uint16, names []string) co.Scope {
	return co.Scope{
		Vantage:      "v1",
		VantageClass: "internet",
		Addresses:    addrs,
		TCPPorts:     tcp,
		Names:        names,
		Profile:      profile(),
	}
}

var Rows = []Row{
	{
		Cells:        []string{"T1/presented", "T2/chain-leaf-first", "T2/leaf-not-after", "T3/named-sni", "T4/rides-reached"},
		Claim:        "a reached Service whose Endpoint completes a TLS handshake records the presented chain as ordered fingerprints, leaf first, under the (Name, Service) key, with SNI equal to the name, and carries the leaf's not_after (RFC3339); the certificate line rides the same exchange as the reachability line",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.10"}, []uint16{443}, []string{"api.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.10:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"api.example.com@198.51.100.10:443/tcp": {
					Outcome:  co.TLSPresented,
					Chain:    []string{"sha256:leaf01", "sha256:intermediate01", "sha256:root01"},
					NotAfter: time.Date(2027, 3, 1, 12, 0, 0, 0, time.UTC),
				},
			}),
		},
		Golden: "cert_presented_named.ndjson",
	},

	{
		Cells:        []string{"T1/tls-refused"},
		Claim:        "a reached Service whose peer speaks TLS but accepts no candidate we offered records tls-refused — a value, distinct from no-tls, so an SNI-required or SSLv3-only listener is not misfiled as *not a TLS server*",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.11"}, []uint16{443}, []string{"api.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.11:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"api.example.com@198.51.100.11:443/tcp": {Outcome: co.TLSRefused},
			}),
		},
		Golden: "cert_tls_refused.ndjson",
	},

	{
		Cells:        []string{"T1/no-tls", "T3/nameless-no-sni"},
		Claim:        "a reached Service on an address-scope Seed is handshaked as the nameless endpoint (no SNI); a port where nothing speaks TLS records no-tls under the @Service key",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.12"}, []uint16{8080}, nil),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.12:8080": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"@198.51.100.12:8080/tcp": {Outcome: co.NoTLS},
			}),
		},
		Golden: "cert_no_tls_nameless.ndjson",
	},

	{
		Cells:        []string{"T4/not-reached-no-cert"},
		Claim:        "a not-reached Service emits its reachability value and NO certificate line — neither TLS negative can be read without knowing the port was open, and the value space has no variant meaning *the port was shut*",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.13"}, []uint16{443}, []string{"api.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.13:443": {co.ConnRefused},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				// Nothing scripted: the handshake must never run on a shut port.
			}),
		},
		Golden: "cert_not_reached_no_cert.ndjson",
	},

	{
		Cells:        []string{"T5/one-cert-per-endpoint"},
		Claim:        "two names on one reached Service are two Endpoints under SNI: each records its own certificate, which is why the chain is keyed on (Name, Service) and never on the Service alone",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.14"}, []uint16{443}, []string{"a.example.com", "b.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.14:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"a.example.com@198.51.100.14:443/tcp": {Outcome: co.TLSPresented, Chain: []string{"sha256:leafA"}, NotAfter: time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)},
				"b.example.com@198.51.100.14:443/tcp": {Outcome: co.TLSPresented, Chain: []string{"sha256:leafB"}, NotAfter: time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)},
			}),
		},
		Golden: "cert_one_per_endpoint.ndjson",
	},

	{
		Cells:        []string{"T6/not-before"},
		Claim:        "a presented leaf whose validity floor is in the future carries not_before (RFC3339) on the value, the raw datum certificate-not-yet-valid derives its verdict from at read",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.20"}, []uint16{443}, []string{"nyv.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.20:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"nyv.example.com@198.51.100.20:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafNYV"},
					NotAfter:  time.Date(2099, 3, 1, 12, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"nyv.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=nyv.example.com", Issuer: "CN=Example CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_not_before.ndjson",
	},

	{
		Cells:        []string{"T6/san-dns"},
		Claim:        "a presented leaf carries its dNSName SANs verbatim in san_dns (wildcards NOT expanded), the raw list certificate-hostname-san-mismatch reads at read",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.21"}, []uint16{443}, []string{"api.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.21:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"api.example.com@198.51.100.21:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafSAN"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"api.example.com", "www.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=api.example.com", Issuer: "CN=Example CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_san_dns.ndjson",
	},

	{
		Cells:        []string{"T6/san-wildcard-leftmost"},
		Claim:        "a presented leaf carries a leftmost single-* wildcard SAN verbatim in san_dns; against a single-label subdomain SNI the read-side predicate matches, but the leaf stores the wildcard unexpanded",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.22"}, []uint16{443}, []string{"foo.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.22:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"foo.example.com@198.51.100.22:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafWild"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"*.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=*.example.com", Issuer: "CN=Example CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_san_wildcard_leftmost.ndjson",
	},

	{
		Cells:        []string{"T6/san-wildcard-apex-noncover"},
		Claim:        "a leftmost wildcard SAN is stored verbatim even where the SNI is the apex it does not cover; the leaf never records a match verdict, only the raw wildcard the read-side derives non-match from",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.23"}, []uint16{443}, []string{"example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.23:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"example.com@198.51.100.23:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafApex"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"*.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=*.example.com", Issuer: "CN=Example CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_san_wildcard_apex.ndjson",
	},

	{
		Cells:        []string{"T6/san-partial-nonmatch"},
		Claim:        "a partial-label wildcard SAN (baz*.example.com) is stored verbatim; the read-side treats any non-leftmost/partial * as a non-match, but the leaf records only the raw string",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.24"}, []uint16{443}, []string{"baz1.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.24:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"baz1.example.com@198.51.100.24:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafPartial"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"baz*.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=baz*.example.com", Issuer: "CN=Example CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_san_partial.ndjson",
	},

	{
		Cells:        []string{"T6/san-ip-only"},
		Claim:        "a presented leaf with only iPAddress SANs carries san_ip and no san_dns; the nameless endpoint reads it and the read-side ignores san_ip entirely (RFC 6125 §6.4.3)",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.25"}, []uint16{443}, nil),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.25:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"@198.51.100.25:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafIPonly"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANIP:     []string{"192.0.2.10", "2001:db8::10"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=192.0.2.10", Issuer: "CN=Example CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_san_ip_only.ndjson",
	},

	{
		Cells:        []string{"T6/chain-key-params"},
		Claim:        "a presented leaf's parsed key params (alg + bits) ride chain_certs; a 1024-bit RSA leaf is under the 2048 floor certificate-weak-key-or-signature's key limb reads at read",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.26"}, []uint16{443}, []string{"weak.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.26:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"weak.example.com@198.51.100.26:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafWeakKey"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"weak.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=weak.example.com", Issuer: "CN=Example CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 1024, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_weak_key.ndjson",
	},

	{
		Cells:        []string{"T6/weak-intermediate"},
		Claim:        "the key limb walks EVERY link: a strong leaf over a 1024-bit intermediate and a strong self-signed root records each link's params in chain_certs, so the read-side fires on the intermediate — proving chain, not leaf-only, scope",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.27"}, []uint16{443}, []string{"chainweak.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.27:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"chainweak.example.com@198.51.100.27:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafCW", "sha256:interWeak", "sha256:rootCW"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"chainweak.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=chainweak.example.com", Issuer: "CN=Weak Inter CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
						{Subject: "CN=Weak Inter CA,O=Example", Issuer: "CN=Strong Root,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 1024, SigDigest: "SHA-256"},
						{Subject: "CN=Strong Root,O=Example", Issuer: "CN=Strong Root,O=Example", SelfSignatureVerifies: b(true), KeyAlg: "RSA", KeyBits: 4096, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_weak_intermediate.ndjson",
	},

	{
		Cells:        []string{"T6/sig-digest"},
		Claim:        "a presented leaf carries its signature DIGEST name (SHA-1) in chain_certs; being CA-issued (subject != issuer) it is not skipped, so the sig limb of certificate-weak-key-or-signature fires at read",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.28"}, []uint16{443}, []string{"sha1.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.28:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"sha1.example.com@198.51.100.28:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafSHA1"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"sha1.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=sha1.example.com", Issuer: "CN=Example CA,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-1"},
					},
				},
			}),
		},
		Golden: "cert_v3_sig_digest.ndjson",
	},

	{
		Cells:        []string{"T6/self-sig-verifies", "T6/self-signed-leaf"},
		Claim:        "a self-signed leaf carries subject==issuer AND self_sig_verifies=true, the two raw facts selfSignedOf() folds so certificate-self-signed derives a definite yes at read",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.29"}, []uint16{443}, []string{"selfsigned.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.29:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"selfsigned.example.com@198.51.100.29:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafSelfSigned"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"selfsigned.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=selfsigned.example.com", Issuer: "CN=selfsigned.example.com", SelfSignatureVerifies: b(true), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
					},
				},
			}),
		},
		Golden: "cert_v3_self_signed_leaf.ndjson",
	},

	{
		Cells:        []string{"T6/self-signed-root-skip"},
		Claim:        "a self-signed root (subject==issuer, self_sig_verifies=true) whose signature digest is SHA-1 carries those raw facts; the sig limb skips it because selfSignedOf() holds, so its SHA-1 self-signature does NOT fire weak-key-or-signature",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.30"}, []uint16{443}, []string{"rootskip.example.com"}),
			Connect: newConn(map[string][]co.ConnResult{
				"198.51.100.30:443": {co.ConnOpen},
			}),
			Handshake: newHandshaker(map[string]co.HandshakeResult{
				"rootskip.example.com@198.51.100.30:443/tcp": {
					Outcome:   co.TLSPresented,
					Chain:     []string{"sha256:leafRS", "sha256:rootRS"},
					NotAfter:  time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
					NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					SANDNS:    []string{"rootskip.example.com"},
					ChainCerts: []co.ChainCert{
						{Subject: "CN=rootskip.example.com", Issuer: "CN=Legacy Root,O=Example", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"},
						{Subject: "CN=Legacy Root,O=Example", Issuer: "CN=Legacy Root,O=Example", SelfSignatureVerifies: b(true), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-1"},
					},
				},
			}),
		},
		Golden: "cert_v3_self_signed_root_skip.ndjson",
	},
}
