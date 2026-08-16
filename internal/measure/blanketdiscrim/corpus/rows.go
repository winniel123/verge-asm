package corpus

import (
	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// ControlPorts is the deterministic control-port set the corpus probes with. It
// is three fixed dynamic-range ports rather than the eight crypto/rand draws
// production makes (blanketdiscrim.CryptoPorts): the corpus pins the DECISION and
// the emission, which are the same at any set size, and a fixed set renders
// byte-identically across runs and architectures. The declared count (eight) is
// carried by ParamsDigest, so a change to it still moves the lock.
var ControlPorts = blanketdiscrim.FixedPorts{P: []uint16{50001, 50002, 50003}}

// Step is one run of the composed exchange inside a row: one Batch at one Vantage
// over one scope, against one scripted connector and handshaker.
type Step struct {
	Batch     string
	Scope     co.Scope
	Connect   *scriptConnector
	Handshake scriptHandshaker
}

// Row is one corpus row: the cells it pins, its one-line claim, whether the claim
// is spec-verified rather than measured, the step, and the golden NDJSON file its
// output must equal byte for byte.
type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Step         Step
	Golden       string
}

// AllCells is the enumeration the coverage test counts against: every cell of the
// blanket-discrimination block must be pinned by at least one row.
var AllCells = []string{
	// B1 — a blanket responder gaps every Service and is never handshaked
	"B1/blanket-all-gap", "B1/blanket-no-cert",
	// B2 — an origin (a control port refuses) takes the ordinary connect values
	"B2/origin-reached-value", "B2/origin-not-reached-value",
	// B3 — an incomplete control probe gaps the reach for the same cause
	"B3/incomplete-probe-gap",
	// B4 — each address is discriminated independently within one scope
	"B4/per-address-independence",
}

func profile() co.SafetyProfile { return co.DefaultProfile() }

// scope builds a one-vantage scope over addresses, TCP ports, and declared names.
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
	// ---- B1: a blanket responder ----
	// Every control port answers, so the address is a blanket responder: each of
	// its Services folds to a reachability Gap with the sixth cause, and no
	// certificate handshake runs (there is no reached Service to hand it). The
	// service ports are never even connected — the verdict is the control probe's.
	{
		Cells:        []string{"B1/blanket-all-gap", "B1/blanket-no-cert"},
		Claim:        "an address that answers every control port is a blanket responder; every Service gaps with the blanket-responder cause and none is handshaked",
		SpecVerified: true,
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.40"}, []uint16{80, 443}, []string{"api.example.com"}),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.40:50001": {co.ConnOpen},
				"198.51.100.40:50002": {co.ConnOpen},
				"198.51.100.40:50003": {co.ConnOpen},
			}),
		},
		Golden: "blanket_responder.ndjson",
	},

	// ---- B2: an ordinary origin ----
	// A control port is refused, so the address discriminates by port and is NOT a
	// blanket responder. Its Services take the ordinary connect values — port 80
	// reached (and handshaked), port 443 refused (not-reached, no handshake).
	{
		Cells:        []string{"B2/origin-reached-value", "B2/origin-not-reached-value"},
		Claim:        "an origin refuses the control ports, so its reaches are trustworthy values: an open port is reached and handshaked, a refused port is not-reached",
		SpecVerified: true,
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.41"}, []uint16{80, 443}, []string{"api.example.com"}),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.41:50001": {co.ConnRefused},
				"198.51.100.41:50002": {co.ConnRefused},
				"198.51.100.41:50003": {co.ConnRefused},
				"198.51.100.41:80":    {co.ConnOpen},
				"198.51.100.41:443":   {co.ConnRefused},
			}),
			Handshake: scriptHandshaker{byEndpoint: map[string]co.HandshakeResult{
				"api.example.com@198.51.100.41:80/tcp": {Outcome: co.NoTLS},
			}},
		},
		Golden: "origin_passes.ndjson",
	},

	// ---- B3: an incomplete control probe ----
	// The control ports time out (never scripted), so the probe did not complete:
	// we could not discriminate, and the reach is a Gap for the same sixth cause,
	// with the incomplete reason rather than the blanket one.
	{
		Cells:        []string{"B3/incomplete-probe-gap"},
		Claim:        "a control probe that times out did not complete, so the reach is an undiscriminated Gap with the incomplete reason — never a value",
		SpecVerified: true,
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.42"}, []uint16{443}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				// nothing scripted: the control ports time out -> incomplete -> Gap.
			}),
		},
		Golden: "incomplete_probe.ndjson",
	},

	// ---- B4: per-address independence ----
	// One scope, two addresses: a blanket responder and an origin. Each is
	// discriminated on its own control probe — the blanket one gaps, the origin one
	// keeps its connect value — so blanket-ness is a property of the address, never
	// the batch. The origin's port is refused so the row stays about discrimination
	// and mints no certificate line.
	{
		Cells:        []string{"B4/per-address-independence"},
		Claim:        "two addresses in one scope are discriminated independently: the blanket one gaps its Service, the origin one keeps its not-reached value",
		SpecVerified: true,
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.43", "198.51.100.44"}, []uint16{443}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.43:50001": {co.ConnOpen}, // blanket
				"198.51.100.43:50002": {co.ConnOpen},
				"198.51.100.43:50003": {co.ConnOpen},
				"198.51.100.44:50001": {co.ConnRefused}, // origin
				"198.51.100.44:50002": {co.ConnRefused},
				"198.51.100.44:50003": {co.ConnRefused},
				"198.51.100.44:443":   {co.ConnRefused},
			}),
		},
		Golden: "mixed_addresses.ndjson",
	},
}
