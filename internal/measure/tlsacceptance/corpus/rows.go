package corpus

import (
	ta "github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
)

// Step is one run of the leaf inside a row: one Batch at one Vantage over one scope,
// against one scripted enumerator.
type Step struct {
	Batch     string
	Scope     ta.Scope
	Enumerate *scriptEnumerator
}

// Row is one corpus row: the cells it pins, its one-line claim, whether the claim is
// spec-verified rather than measured, the declared candidate set, the step, and the
// golden NDJSON file its output must equal byte for byte.
type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Candidates   ta.CandidateSet
	Step         Step
	Golden       string
}

// AllCells is the enumeration the coverage test counts against: every cell of the
// tls-acceptance block must be pinned by at least one row.
var AllCells = []string{
	// T1 — a modern listener: TLS 1.2 (with suites) and TLS 1.3 (version-only).
	"T1/modern-1.2-1.3",
	// T2 — a TLS 1.0 accept is recorded (it reads the v1 signal tls-1.0-accepted).
	"T2/tls-1.0-accepted",
	// T3 — a peer that spoke TLS and accepted nothing offered is `tls-refused`.
	"T3/tls-refused",
	// T4 — a port where nothing spoke TLS is `no-tls`, distinct from a refusal.
	"T4/no-tls",
	// T5 — one batch of mixed Services: each emits its own acceptance value.
	"T5/mixed-batch",
}

func candidates() ta.CandidateSet { return ta.DefaultCandidateSet() }

// scope builds a one-vantage scope over the given Services under the default
// declared candidate set.
func scope(services []ta.ServiceTarget) ta.Scope {
	return ta.Scope{
		Vantage:      "v1",
		VantageClass: "internet",
		Services:     services,
		Candidates:   candidates(),
	}
}

// A modern listener: TLS 1.2 with two GCM suites and TLS 1.3.
var modern = listener{spoke: true, accepts: map[string][]string{
	ta.TLS12: {"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
	ta.TLS13: nil,
}}

// A legacy listener still accepting TLS 1.0 (a CBC suite) alongside TLS 1.2.
var legacy10 = listener{spoke: true, accepts: map[string][]string{
	ta.TLS10: {"TLS_RSA_WITH_AES_128_CBC_SHA"},
	ta.TLS12: {"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
}}

// A listener that speaks TLS but accepts no candidate we offered.
var refuses = listener{spoke: true, accepts: map[string][]string{}}

// Rows is the checked-in corpus. Every cell in AllCells appears in some row's Cells;
// the coverage test fails the build (naming the cell) if one does not.
var Rows = []Row{
	// ---- T1/modern-1.2-1.3 ----
	{
		Cells:        []string{"T1/modern-1.2-1.3"},
		Claim:        "a modern listener accepting TLS 1.2 (two GCM suites, in selection order) and TLS 1.3 reads `enumerated` — 1.2 carries its accepted suites, 1.3 records version-only (its suites are the library's choice)",
		SpecVerified: true,
		Candidates:   candidates(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]ta.ServiceTarget{{Address: "198.51.100.10", Port: 443}}),
			Enumerate: newScript(map[string]listener{
				"198.51.100.10:443/tcp": modern,
			}),
		},
		Golden: "tls_modern.ndjson",
	},

	// ---- T2/tls-1.0-accepted ----
	{
		Cells:        []string{"T2/tls-1.0-accepted"},
		Claim:        "a listener accepting TLS 1.0 records version 1.0 in the value — the finding that reads the v1 signal tls-1.0-accepted (measurement-offers §1.2)",
		SpecVerified: true,
		Candidates:   candidates(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]ta.ServiceTarget{{Address: "198.51.100.11", Port: 443}}),
			Enumerate: newScript(map[string]listener{
				"198.51.100.11:443/tcp": legacy10,
			}),
		},
		Golden: "tls_1_0_accepted.ndjson",
	},

	// ---- T3/tls-refused ----
	{
		Cells:        []string{"T3/tls-refused"},
		Claim:        "a peer that spoke TLS and accepted nothing offered is `tls-refused`, carrying no accepted versions — read with the batch's candidate set it is *the peer spoke TLS and refused all of this*",
		SpecVerified: true,
		Candidates:   candidates(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]ta.ServiceTarget{{Address: "198.51.100.12", Port: 443}}),
			Enumerate: newScript(map[string]listener{
				"198.51.100.12:443/tcp": refuses,
			}),
		},
		Golden: "tls_refused.ndjson",
	},

	// ---- T4/no-tls ----
	{
		Cells:        []string{"T4/no-tls"},
		Claim:        "a port where nothing spoke TLS at all is `no-tls`, a value distinct from a refusal — collapsing the two files a plaintext listener under *TLS server*",
		SpecVerified: true,
		Candidates:   candidates(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]ta.ServiceTarget{{Address: "198.51.100.13", Port: 80}}),
			Enumerate: newScript(map[string]listener{
				"198.51.100.13:80/tcp": {spoke: false},
			}),
		},
		Golden: "tls_no_tls.ndjson",
	},

	// ---- T5/mixed-batch ----
	{
		Cells:        []string{"T5/mixed-batch"},
		Claim:        "one batch of four Services — modern, TLS-1.0-accepting, tls-refused, and no-tls — emits one acceptance observation per Service in scope order; every negative is a value the enumeration measured",
		SpecVerified: true,
		Candidates:   candidates(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]ta.ServiceTarget{
				{Address: "198.51.100.20", Port: 443},
				{Address: "198.51.100.21", Port: 443},
				{Address: "198.51.100.22", Port: 443},
				{Address: "198.51.100.23", Port: 8080},
			}),
			Enumerate: newScript(map[string]listener{
				"198.51.100.20:443/tcp":  modern,
				"198.51.100.21:443/tcp":  legacy10,
				"198.51.100.22:443/tcp":  refuses,
				"198.51.100.23:8080/tcp": {spoke: false},
			}),
		},
		Golden: "tls_mixed_batch.ndjson",
	},
}
