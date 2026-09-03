package corpus

import (
	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

type Step struct {
	Batch      string
	Vantage    string
	Resolver   string
	Names      []string
	SeedScopes []string
	Peer       ScriptPeer
}

// Row is one corpus row: the cells it pins, its one-line claim, whether the claim
// is spec-verified rather than measured (ADR-0021's honesty rider — every W-cell
// is spec-verified today, §8.5), the declared parameters, the steps, and the
// golden NDJSON file its output must equal byte for byte.
type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Params       wd.Params
	Steps        []Step
	Golden       string
}

// AllCells is the enumeration A5 counts against: every cell of the
// wildcard-discrimination block (golden-corpus.md §8) must be pinned by at least
// one row. 19 cells — §8.1's adopted escrow W1–W6 (17) plus §8.2's citation pin
// W7 (2). It is the length of a list, not a target.
var AllCells = []string{
	"W1/NoSynthesis", "W1/Determinate", "W1/Indeterminate",
	"W2/Shadowed", "W2/not-Shadowed",
	// W3 — boundary pins (4 boundaries × 2 = 8)
	"W3a/determinate", "W3a/indeterminate-only",
	"W3b/none-determinate", "W3b/determinate-differing",
	"W3c/no-wildcard", "W3c/incomplete",
	"W3d/discriminated", "W3d/shadowed-all",
	"W4/set-shape", "W4/independent",
	"W5/shared-path",
	"W6/suppression",
	// W7 — the citation pin (1 boundary × 2 = 2)
	"W7.1/cites-nothing", "W7.2/cites",
}

const (
	resolver = "resolver.test"
	parent   = "example.com"
	real     = "real.example.com"
	ghost    = "ghost.example.com"
	gone     = "gone.example.com"
)

func params() wd.Params { return wd.DefaultParams() }

func one(cells []string, claim string, specVerified bool, names []string, peer ScriptPeer, golden string) Row {
	return Row{
		Cells:        cells,
		Claim:        claim,
		SpecVerified: specVerified,
		Params:       params(),
		Steps: []Step{{
			Batch: "b1", Vantage: "v1", Resolver: resolver,
			Names: names, SeedScopes: []string{parent}, Peer: peer,
		}},
		Golden: golden,
	}
}

var Rows = []Row{
	one([]string{"W1/NoSynthesis", "W2/not-Shadowed", "W3c/no-wildcard", "W5/shared-path"},
		"the control probe completes on the declared path and every label answers NXDOMAIN — a determinate NoSynthesis, no wildcard, licensing the name's own Resolved value; a probe on a skewed path would fabricate here",
		true,
		[]string{real},
		ScriptPeer{Rules: []scriptRule{
			{Name: real, Qtype: rw.QtypeA, Reply: noerror(rrA(real, "198.51.100.1"))},
			{Under: parent, Shape: anyShape, Qtype: rw.QtypeA, Reply: nxdomain()},
		}},
		"nowc_license.ndjson"),

	one([]string{"W3c/incomplete"},
		"the control probe under the parent did not complete — every control query was silent — so the name records a Gap, never a value",
		true,
		[]string{real},
		ScriptPeer{Rules: []scriptRule{
			{Name: real, Qtype: rw.QtypeA, Reply: noerror(rrA(real, "198.51.100.1"))},
			// No Under rule: control labels are unmatched and read not-reached.
		}},
		"nowc_gap.ndjson"),

	one([]string{"W1/Determinate", "W2/Shadowed", "W3d/shadowed-all", "W7.1/cites-nothing"},
		"a determinate constant wildcard synthesises the same address for every control label and for a fictional name; the name coincides at the one determinate component and is Shadowed on resolution and on every dns-record discriminator, citing no address",
		true,
		[]string{ghost},
		ScriptPeer{Rules: []scriptRule{
			{Under: parent, Shape: anyShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.1"))},
		}},
		"det_shadowed.ndjson"),

	one([]string{"W3a/determinate", "W3b/determinate-differing", "W3d/discriminated", "W7.2/cites"},
		"under the same determinate wildcard a real name carries its own distinct address, differing at the determinate component; it is discriminated, so no answer is synthesised at any qtype and the resolved set cites",
		true,
		[]string{real},
		ScriptPeer{Rules: []scriptRule{
			{Name: real, Qtype: rw.QtypeA, Reply: noerror(rrA(real, "198.51.100.9"))},
			{Under: parent, Shape: anyShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.1"))},
		}},
		"det_differs.ndjson"),

	one([]string{"W1/Indeterminate", "W3b/none-determinate"},
		"the control labels disagree at A — the random labels answer one address, the structured label another — so the only component is Indeterminate; with no determinate component every name beneath is Shadowed",
		true,
		[]string{ghost},
		ScriptPeer{Rules: []scriptRule{
			{Under: parent, Shape: randomShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.8"))},
			{Under: parent, Shape: structuredShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.7"))},
		}},
		"indet_shadowed.ndjson"),

	one([]string{"W3a/indeterminate-only"},
		"MX is determinate across every control label while A is indeterminate; a name that coincides at the determinate MX and differs only at the indeterminate A is Shadowed — an indeterminate component is never consulted",
		true,
		[]string{ghost},
		ScriptPeer{Rules: []scriptRule{
			{Under: parent, Shape: anyShape, Qtype: rw.QtypeMX, Reply: noerror(rrMX("l."+parent, "mail.example.net"))},
			{Under: parent, Shape: randomShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.8"))},
			{Under: parent, Shape: structuredShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.7"))},
		}},
		"indet_only.ndjson"),

	one([]string{"W6/suppression"},
		"beneath a parent that neither synthesises determinately nor is silent, a name the authority answers NXDOMAIN for is recorded Shadowed rather than NameError — no withdrawal-shaped output (the Name suppression case, spec-verified and unmeasured)",
		true,
		[]string{gone},
		ScriptPeer{Rules: []scriptRule{
			{Name: gone, Qtype: rw.QtypeA, Reply: nxdomain()},
			{Under: parent, Shape: randomShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.8"))},
			{Under: parent, Shape: structuredShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.7"))},
		}},
		"w6_suppress.ndjson"),

	one([]string{"W4/set-shape", "W4/independent"},
		"the control-label set is 9 random plus 1 structured label drawn per batch; the random labels answer NODATA and would read 'no wildcard', but the structured label answers an address, making A indeterminate and withholding the licence — the third door (nip.io), which a random-only set cannot see",
		true,
		[]string{ghost},
		ScriptPeer{Rules: []scriptRule{
			{Under: parent, Shape: randomShape, Qtype: rw.QtypeA, Reply: nodata()},
			{Under: parent, Shape: structuredShape, Qtype: rw.QtypeA, Reply: noerror(rrA("l."+parent, "203.0.113.7"))},
		}},
		"w4_structured.ndjson"),
}
