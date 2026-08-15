package corpus

import (
	"bytes"

	he "github.com/winniel123/verge-asm/internal/measure/httpexchange"
)

// Step is one run of the leaf inside a row: one Batch at one Vantage over one
// scope, against one scripted exchanger.
type Step struct {
	Batch    string
	Scope    he.Scope
	Exchange *scriptExchanger
}

// Row is one corpus row: the cells it pins, its one-line claim, whether the claim
// is spec-verified rather than measured, the declared parameter set, the step, and
// the golden NDJSON file its output must equal byte for byte.
type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Params       he.Params
	Step         Step
	Golden       string
}

// AllCells is the enumeration the coverage test counts against: every cell of the
// http-exchange block must be pinned by at least one row.
var AllCells = []string{
	// H1 — a completed exchange creates the Endpoint and records http-identity,
	// named and nameless.
	"H1/named-200", "H1/nameless-204",
	// H2 — a redirect is recorded (its Location) and NOT followed.
	"H2/redirect-not-followed",
	// H3 — the body read is capped at 64 KB and the overrun is marked truncated.
	"H3/body-capped",
	// H4 — a failed exchange creates no Endpoint and emits no observation.
	"H4/failure-no-endpoint",
	// H5 — one batch of mixed targets: successes emit, failures are omitted.
	"H5/mixed-batch",
}

func params() he.Params { return he.DefaultParams() }

// scope builds a one-vantage scope over the given targets, under the default
// declared parameters.
func scope(targets []he.Target) he.Scope {
	return he.Scope{
		Vantage:      "v1",
		VantageClass: "internet",
		Targets:      targets,
		Params:       params(),
	}
}

// overCapBody is a body one byte longer than the 64 KB cap, so the row exercises
// truncation. The emitted value carries only the digest and byte count, so the
// golden file stays tiny regardless of the body's size.
var overCapBody = bytes.Repeat([]byte("a"), he.DefaultParams().BodyCapBytes+1)

// Rows is the checked-in corpus. Every cell in AllCells appears in some row's
// Cells; the coverage test fails the build (naming the cell) if one does not.
var Rows = []Row{
	// ---- H1/named-200 ----
	{
		Cells:        []string{"H1/named-200"},
		Claim:        "a completed GET / to a named (Name, Service) pair creates the Endpoint and records its http-identity — status, Server header, and a digest over the body",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{{Name: "api.example.com", Address: "198.51.100.10", Port: 443, Scheme: "https"}}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"api.example.com@198.51.100.10:443/tcp": {Status: 200, Server: "nginx", ContentType: "text/html", Body: []byte("<!doctype html>")},
			}),
		},
		Golden: "http_named_200.ndjson",
	},

	// ---- H1/nameless-204 ----
	{
		Cells:        []string{"H1/nameless-204"},
		Claim:        "a Service reached with no citing Name records identity under the nameless Endpoint key (@service) — a distinguished key variant, never an empty name",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{{Name: "", Address: "198.51.100.11", Port: 443, Scheme: "https"}}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"@198.51.100.11:443/tcp": {Status: 204},
			}),
		},
		Golden: "http_nameless_204.ndjson",
	},

	// ---- H2/redirect-not-followed ----
	{
		Cells:        []string{"H2/redirect-not-followed"},
		Claim:        "a 3xx is a completed exchange: the Endpoint exists and the Location is recorded as identity, and the leaf issues exactly one request — redirects are not followed",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{{Name: "www.example.com", Address: "198.51.100.12", Port: 80, Scheme: "http"}}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"www.example.com@198.51.100.12:80/tcp": {Status: 301, Server: "nginx", Location: "https://www.example.com/"},
			}),
		},
		Golden: "http_redirect_not_followed.ndjson",
	},

	// ---- H3/body-capped ----
	{
		Cells:        []string{"H3/body-capped"},
		Claim:        "a body longer than the 64 KB cap is truncated to the cap; body_bytes equals the cap, body_truncated is true, and the digest is over the capped body",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{{Name: "big.example.com", Address: "198.51.100.13", Port: 443, Scheme: "https"}}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"big.example.com@198.51.100.13:443/tcp": {Status: 200, Server: "caddy", Body: overCapBody},
			}),
		},
		Golden: "http_body_capped.ndjson",
	},

	// ---- H4/failure-no-endpoint ----
	{
		Cells:        []string{"H4/failure-no-endpoint"},
		Claim:        "a transport failure is not an identity: no HTTP response completed, so no Endpoint is created and no observation is emitted at all",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{{Name: "down.example.com", Address: "198.51.100.14", Port: 443, Scheme: "https"}}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"down.example.com@198.51.100.14:443/tcp": {Failed: true, Err: "connection reset by peer"},
			}),
		},
		Golden: "http_failure_no_endpoint.ndjson",
	},

	// ---- H5/mixed-batch ----
	{
		Cells:        []string{"H5/mixed-batch"},
		Claim:        "one batch of three targets — a 200, a failure, and a nameless 200 — emits an Endpoint for each successful exchange in target order and omits the failed one",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{
				{Name: "a.example.com", Address: "198.51.100.20", Port: 443, Scheme: "https"},
				{Name: "b.example.com", Address: "198.51.100.21", Port: 443, Scheme: "https"},
				{Name: "", Address: "198.51.100.22", Port: 8080, Scheme: "http"},
			}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"a.example.com@198.51.100.20:443/tcp": {Status: 200, Server: "nginx", Body: []byte("ok")},
				// b.example.com fails — omitted from the output.
				"b.example.com@198.51.100.21:443/tcp": {Failed: true, Err: "i/o timeout"},
				"@198.51.100.22:8080/tcp":             {Status: 200, Server: "gunicorn", Body: []byte("{}")},
			}),
		},
		Golden: "http_mixed_batch.ndjson",
	},
}
