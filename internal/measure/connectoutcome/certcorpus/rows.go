package certcorpus

import (
	"time"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// Step is one run of the reachability exchange inside a row: one Batch at one
// Vantage over one scope, against a scripted connector and a scripted handshaker.
type Step struct {
	Batch     string
	Scope     co.Scope
	Connect   *scriptConnector
	Handshake *scriptHandshaker
}

// Row is one corpus row: the cells it pins, its one-line claim, whether the claim
// is spec-verified, the declared safety profile, the step, and the golden NDJSON
// file its output must equal byte for byte.
type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Profile      co.SafetyProfile
	Step         Step
	Golden       string
}

// AllCells is the enumeration the coverage test counts against: every cell of the
// tls-handshake / certificate block must be pinned by at least one row.
var AllCells = []string{
	// T1 — the value space's three variants (a closed union, ADR-0011)
	"T1/presented", "T1/tls-refused", "T1/no-tls",
	// T2 — the chain is the ordered fingerprints, leaf first
	"T2/chain-leaf-first",
	// T2 — the presented value carries the leaf's not_after (RFC3339); negatives omit it
	"T2/leaf-not-after",
	// T3 — SNI is the Endpoint's name; the nameless endpoint sends none
	"T3/named-sni", "T3/nameless-no-sni",
	// T4 — the handshake rides a REACHED Service only; not-reached emits no cert
	"T4/rides-reached", "T4/not-reached-no-cert",
	// T5 — one certificate per Endpoint when a Service carries several names
	"T5/one-cert-per-endpoint",
	// T6 — the v3 leaf's raw parsed facts the four dark certificate rules derive
	// their verdicts from at read (store-raw / derive-at-read, P0.10b, #704).
	"T6/not-before",                // leaf validity floor (not-yet-valid input)
	"T6/san-dns",                   // leaf carries literal dNSName SANs
	"T6/san-wildcard-leftmost",     // leaf carries a leftmost single-* wildcard SAN
	"T6/san-wildcard-apex-noncover", // wildcard SAN against an apex name (won't cover)
	"T6/san-partial-nonmatch",      // partial-label wildcard SAN (never a match)
	"T6/san-ip-only",               // leaf carries only iPAddress SANs, no dNSName
	"T6/chain-key-params",          // weak RSA (<2048) leaf key params
	"T6/weak-intermediate",         // 1024-bit intermediate in an otherwise-strong chain (chain scope)
	"T6/sig-digest",                // SHA-1-signed CA-issued leaf (sig-digest limb)
	"T6/self-sig-verifies",         // leaf self-signature verifies against its own key
	"T6/self-signed-leaf",          // subject==issuer self-signed leaf
	"T6/self-signed-root-skip",     // self-signed root whose SHA-1 self-sig must NOT fire the sig limb
}

func profile() co.SafetyProfile { return co.DefaultProfile() }

// b returns a pointer to v, for the *bool self_sig_verifies field: a self-signed
// link needs &true; an ordinary CA-signed link needs &false.
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

// Rows is the checked-in corpus. Every cell in AllCells appears in some row's
// Cells; the coverage test fails the build (naming the cell) if one does not.
var Rows = []Row{
	// ---- T1/presented + T2/chain-leaf-first + T3/named-sni + T4/rides-reached ----
	{
		Cells: []string{"T1/presented", "T2/chain-leaf-first", "T2/leaf-not-after", "T3/named-sni", "T4/rides-reached"},
		Claim: "a reached Service whose Endpoint completes a TLS handshake records the presented chain as ordered fingerprints, leaf first, under the (Name, Service) key, with SNI equal to the name, and carries the leaf's not_after (RFC3339); the certificate line rides the same exchange as the reachability line",
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

	// ---- T1/tls-refused ----
	{
		Cells: []string{"T1/tls-refused"},
		Claim: "a reached Service whose peer speaks TLS but accepts no candidate we offered records tls-refused — a value, distinct from no-tls, so an SNI-required or SSLv3-only listener is not misfiled as *not a TLS server*",
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

	// ---- T1/no-tls + T3/nameless-no-sni ----
	{
		Cells: []string{"T1/no-tls", "T3/nameless-no-sni"},
		Claim: "a reached Service on an address-scope Seed is handshaked as the nameless endpoint (no SNI); a port where nothing speaks TLS records no-tls under the @Service key",
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

	// ---- T4/not-reached-no-cert ----
	{
		Cells: []string{"T4/not-reached-no-cert"},
		Claim: "a not-reached Service emits its reachability value and NO certificate line — neither TLS negative can be read without knowing the port was open, and the value space has no variant meaning *the port was shut*",
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

	// ---- T5/one-cert-per-endpoint ----
	{
		Cells: []string{"T5/one-cert-per-endpoint"},
		Claim: "two names on one reached Service are two Endpoints under SNI: each records its own certificate, which is why the chain is keyed on (Name, Service) and never on the Service alone",
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

	// ---- T6/not-before ----
	{
		Cells: []string{"T6/not-before"},
		Claim: "a presented leaf whose validity floor is in the future carries not_before (RFC3339) on the value, the raw datum certificate-not-yet-valid derives its verdict from at read",
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

	// ---- T6/san-dns (literal) ----
	{
		Cells: []string{"T6/san-dns"},
		Claim: "a presented leaf carries its dNSName SANs verbatim in san_dns (wildcards NOT expanded), the raw list certificate-hostname-san-mismatch reads at read",
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

	// ---- T6/san-wildcard-leftmost ----
	{
		Cells: []string{"T6/san-wildcard-leftmost"},
		Claim: "a presented leaf carries a leftmost single-* wildcard SAN verbatim in san_dns; against a single-label subdomain SNI the read-side predicate matches, but the leaf stores the wildcard unexpanded",
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

	// ---- T6/san-wildcard-apex-noncover ----
	{
		Cells: []string{"T6/san-wildcard-apex-noncover"},
		Claim: "a leftmost wildcard SAN is stored verbatim even where the SNI is the apex it does not cover; the leaf never records a match verdict, only the raw wildcard the read-side derives non-match from",
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

	// ---- T6/san-partial-nonmatch ----
	{
		Cells: []string{"T6/san-partial-nonmatch"},
		Claim: "a partial-label wildcard SAN (baz*.example.com) is stored verbatim; the read-side treats any non-leftmost/partial * as a non-match, but the leaf records only the raw string",
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

	// ---- T6/san-ip-only ----
	{
		Cells: []string{"T6/san-ip-only"},
		Claim: "a presented leaf with only iPAddress SANs carries san_ip and no san_dns; the nameless endpoint reads it and the read-side ignores san_ip entirely (RFC 6125 §6.4.3)",
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

	// ---- T6/chain-key-params (weak RSA leaf) ----
	{
		Cells: []string{"T6/chain-key-params"},
		Claim: "a presented leaf's parsed key params (alg + bits) ride chain_certs; a 1024-bit RSA leaf is under the 2048 floor certificate-weak-key-or-signature's key limb reads at read",
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

	// ---- T6/weak-intermediate (chain scope) ----
	{
		Cells: []string{"T6/weak-intermediate"},
		Claim: "the key limb walks EVERY link: a strong leaf over a 1024-bit intermediate and a strong self-signed root records each link's params in chain_certs, so the read-side fires on the intermediate — proving chain, not leaf-only, scope",
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

	// ---- T6/sig-digest (SHA-1 CA-issued leaf) ----
	{
		Cells: []string{"T6/sig-digest"},
		Claim: "a presented leaf carries its signature DIGEST name (SHA-1) in chain_certs; being CA-issued (subject != issuer) it is not skipped, so the sig limb of certificate-weak-key-or-signature fires at read",
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

	// ---- T6/self-sig-verifies + T6/self-signed-leaf ----
	{
		Cells: []string{"T6/self-sig-verifies", "T6/self-signed-leaf"},
		Claim: "a self-signed leaf carries subject==issuer AND self_sig_verifies=true, the two raw facts selfSignedOf() folds so certificate-self-signed derives a definite yes at read",
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

	// ---- T6/self-signed-root-skip ----
	{
		Cells: []string{"T6/self-signed-root-skip"},
		Claim: "a self-signed root (subject==issuer, self_sig_verifies=true) whose signature digest is SHA-1 carries those raw facts; the sig limb skips it because selfSignedOf() holds, so its SHA-1 self-signature does NOT fire weak-key-or-signature",
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
