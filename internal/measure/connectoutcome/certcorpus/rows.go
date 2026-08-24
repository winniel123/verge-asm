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
}

func profile() co.SafetyProfile { return co.DefaultProfile() }

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
}
