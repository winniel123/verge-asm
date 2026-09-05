package corpus

import (
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

type Step struct {
	Batch   string
	Scope   co.Scope
	Connect *scriptConnector
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
	"C1/reached", "C1/not-reached",
	"C2/open", "C2/refused", "C2/timeout-exhausted",
	"C3/refusal-not-retried", "C3/transient-timeout-recovers",
	"C4/mixed-open-closed", "C4/round-robin-order",
	"C5/udp-recorded-not-probed",
}

func profile() co.SafetyProfile { return co.DefaultProfile() }

func scope(addrs []string, tcp, udp []uint16) co.Scope {
	return co.Scope{
		Vantage:      "v1",
		VantageClass: "internet",
		Addresses:    addrs,
		TCPPorts:     tcp,
		UDPPorts:     udp,
		Profile:      profile(),
	}
}

var Rows = []Row{
	{
		Cells:        []string{"C1/reached", "C2/open"},
		Claim:        "a completed TCP connect to an open port is reached; the connection is opened and closed and the Service records reachable",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.10"}, []uint16{443}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.10:443": {co.ConnOpen},
			}),
		},
		Golden: "reached_open.ndjson",
	},

	{
		Cells:        []string{"C1/not-reached", "C2/refused", "C3/refusal-not-retried"},
		Claim:        "an RST refusal is an answer — the port is shut — so it decides not-reached in one attempt and is never retried",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.11"}, []uint16{3306}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.11:3306": {co.ConnRefused},
			}),
		},
		Golden: "notreached_refused.ndjson",
	},

	{
		Cells:        []string{"C2/timeout-exhausted"},
		Claim:        "silence that outlasts the retry budget decides not-reached: on a connection-oriented transport an unanswered connect is a value, not a Gap (ADR-0083)",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.12"}, []uint16{9200}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.12:9200": {co.ConnTimedOut},
			}),
		},
		Golden: "notreached_timeout.ndjson",
	},

	{
		Cells:        []string{"C3/transient-timeout-recovers"},
		Claim:        "a transient timeout is retried within the budget; a later open wins, so a single dropped SYN does not manufacture a false not-reached",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.13"}, []uint16{8080}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.13:8080": {co.ConnTimedOut, co.ConnOpen},
			}),
		},
		Golden: "recovers_after_timeout.ndjson",
	},

	{
		Cells:        []string{"C4/mixed-open-closed", "C4/round-robin-order"},
		Claim:        "a Service exists for every (Address, port, tcp) in scope, open or closed; two hosts' ports are scheduled round-robin, and both an open and a closed port on each host record a value",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]string{"198.51.100.20", "198.51.100.21"}, []uint16{80, 443}, nil),
			Connect: newScript(map[string][]co.ConnResult{
				"198.51.100.20:80":  {co.ConnOpen},
				"198.51.100.20:443": {co.ConnRefused},
				"198.51.100.21:80":  {co.ConnRefused},
				"198.51.100.21:443": {co.ConnOpen},
			}),
		},
		Golden: "mixed_open_closed.ndjson",
	},

	{
		Cells:        []string{"C5/udp-recorded-not-probed"},
		Claim:        "a scope whose only pairs are UDP produces no reachability observation at all — UDP is recorded in scope and never probed (v1 spec §3.5, ADR-0083)",
		SpecVerified: true,
		Profile:      profile(),
		Step: Step{
			Batch:   "b1",
			Scope:   scope([]string{"198.51.100.30"}, nil, []uint16{161, 623}),
			Connect: newScript(map[string][]co.ConnResult{}),
		},
		Golden: "udp_recorded_not_probed.ndjson",
	},
}
