package corpus

import (
	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

var ControlPorts = blanketdiscrim.FixedPorts{P: []uint16{50001, 50002, 50003}}

type Step struct {
	Batch     string
	Scope     co.Scope
	Connect   *scriptConnector
	Handshake scriptHandshaker
}

type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Step         Step
	Golden       string
}

var AllCells = []string{
	"B1/blanket-all-gap", "B1/blanket-no-cert",
	"B2/origin-reached-value", "B2/origin-not-reached-value",
	"B3/incomplete-probe-gap",
	"B4/per-address-independence",
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

var Rows = []Row{
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

	{
		Cells:        []string{"B3/incomplete-probe-gap"},
		Claim:        "a control probe that times out did not complete, so the reach is an undiscriminated Gap with the incomplete reason — never a value",
		SpecVerified: true,
		Step: Step{
			Batch:   "b1",
			Scope:   scope([]string{"198.51.100.42"}, []uint16{443}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				// Deliberately empty: the control ports time out, so the probe is incomplete.
			}),
		},
		Golden: "incomplete_probe.ndjson",
	},

	{
		Cells:        []string{"B4/per-address-independence"},
		Claim:        "two addresses in one scope are discriminated independently: the blanket one gaps its Service, the origin one keeps its not-reached value",
		SpecVerified: true,
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.43", "198.51.100.44"}, []uint16{443}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.43:50001": {co.ConnOpen},
				"198.51.100.43:50002": {co.ConnOpen},
				"198.51.100.43:50003": {co.ConnOpen},
				"198.51.100.44:50001": {co.ConnRefused},
				"198.51.100.44:50002": {co.ConnRefused},
				"198.51.100.44:50003": {co.ConnRefused},
				"198.51.100.44:443":   {co.ConnRefused},
			}),
		},
		Golden: "mixed_addresses.ndjson",
	},
}
