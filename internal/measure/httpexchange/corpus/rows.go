package corpus

import (
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
	// H3 — the admitted closed set: <title> lifted from the body and the 401
	// WWW-Authenticate challenge, with NO body hash, length, or Content-Type.
	"H3/admitted-set",
	// H4 — a reached Service that speaks no HTTP records the no-http-response
	// negative and still creates the Endpoint.
	"H4/no-http-response",
	// H5 — one batch of mixed targets: every reached target emits, the non-HTTP one
	// as no-http-response.
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

// Rows is the checked-in corpus. Every cell in AllCells appears in some row's
// Cells; the coverage test fails the build (naming the cell) if one does not.
var Rows = []Row{
	// ---- H1/named-200 ----
	{
		Cells:        []string{"H1/named-200"},
		Claim:        "a completed GET / to a named (Name, Service) pair creates the Endpoint and records its http-identity — outcome responded, status, Server header, and the page <title> lifted from the body (never the body itself)",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{{Name: "api.example.com", Address: "198.51.100.10", Port: 443, Scheme: "https"}}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"api.example.com@198.51.100.10:443/tcp": {Status: 200, Server: "nginx", Body: []byte("<!doctype html><html><head><title>Example API</title></head><body>ok</body></html>")},
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

	// ---- H3/admitted-set ----
	{
		Cells:        []string{"H3/admitted-set"},
		Claim:        "the admitted closed set only: a 401 records outcome responded, status, Server, the <title> lifted from the body, and the WWW-Authenticate challenge — and NO body hash, body length, or Content-Type",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{{Name: "auth.example.com", Address: "198.51.100.13", Port: 443, Scheme: "https"}}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"auth.example.com@198.51.100.13:443/tcp": {
					Status: 401, Server: "caddy",
					WWWAuthenticate: `Basic realm="restricted"`,
					Body:            []byte("<title>Sign in</title>"),
				},
			}),
		},
		Golden: "http_admitted_set.ndjson",
	},

	// ---- H4/no-http-response ----
	{
		Cells:        []string{"H4/no-http-response"},
		Claim:        "a reached Service that returns no valid HTTP response records the no-http-response negative as a VALUE and still creates the Endpoint — never an absence",
		SpecVerified: true,
		Params:       params(),
		Step: Step{
			Batch: "b1",
			Scope: scope([]he.Target{{Name: "down.example.com", Address: "198.51.100.14", Port: 443, Scheme: "https"}}),
			Exchange: newScript(map[string]he.ExchangeResult{
				"down.example.com@198.51.100.14:443/tcp": {Failed: true, Err: "connection reset by peer"},
			}),
		},
		Golden: "http_no_http_response.ndjson",
	},

	// ---- H5/mixed-batch ----
	{
		Cells:        []string{"H5/mixed-batch"},
		Claim:        "one batch of three reached targets — a 200, a non-HTTP one, and a nameless 200 — emits an Endpoint for each in target order, the non-HTTP one as no-http-response",
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
				"a.example.com@198.51.100.20:443/tcp": {Status: 200, Server: "nginx", Body: []byte("<title>A</title>")},
				// b.example.com is reached but speaks no HTTP — recorded no-http-response.
				"b.example.com@198.51.100.21:443/tcp": {Failed: true, Err: "i/o timeout"},
				"@198.51.100.22:8080/tcp":             {Status: 200, Server: "gunicorn", Body: []byte("{}")},
			}),
		},
		Golden: "http_mixed_batch.ndjson",
	},
}
