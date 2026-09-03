package corpus

import "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"

type Step struct {
	Batch    string
	Vantage  string
	Resolver string
	Names    []string
	Peer     ScriptPeer
}

// Row is one corpus row: the cells it pins, its one-line claim, whether the
// claim is spec-verified rather than measured (ADR-0021's honesty rider), the
// offers on the wire, the steps, and the golden NDJSON file its output must
// equal byte for byte.
type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Offers       resolutionwalk.Offers
	Steps        []Step
	Golden       string
}

// AllCells is the enumeration A5 counts against: every cell of the
// resolution-walk block (golden-corpus.md §2) must be pinned by at least one
// row. 27 cells — 5 outcome pins, 9 boundaries × 2, 1 path-provenance pin,
// 3 withdrawal→return. It is the length of a list, not a target: adding a
// boundary adds two cells here and a row for each.
var AllCells = []string{
	// §2.1 Block M1 — outcome pins (5)
	"M1.1", "M1.2", "M1.3", "M1.4", "M1.5",
	// §2.2 Block M2 — boundary pins (9 × 2 = 18)
	"M2.a/NameError", "M2.a/NoData",
	"M2.b/NameError", "M2.b/survives",
	"M2.c/Resolved", "M2.c/NoData",
	"M2.d/Resolved", "M2.d/Gap",
	"M2.e/Lame", "M2.e/Gap",
	"M2.f/Lame", "M2.f/not-Lame",
	"M2.g/set", "M2.g/serialisation",
	"M2.h/folds", "M2.h/distinct",
	"M2.i/NoData", "M2.i/Gap",
	// §2.3 Block M3 — path provenance (1)
	"M3.1",
	// §2.4 Block R — withdrawal→return (3)
	"R.1", "R.2", "R.3",
}

const (
	resolver = "resolver.test"
	nameEx   = "example.com"
	ns1      = "ns1.example.net"
	ns2      = "ns2.example.net"
)

func def() resolutionwalk.Offers { return resolutionwalk.DefaultOffers() }

func one(cells []string, claim string, peer ScriptPeer, golden string) Row {
	return Row{
		Cells:        cells,
		Claim:        claim,
		SpecVerified: true,
		Offers:       def(),
		Steps:        []Step{{Batch: "b1", Vantage: "v1", Resolver: resolver, Names: []string{nameEx}, Peer: peer}},
		Golden:       golden,
	}
}

var Rows = []Row{
	one([]string{"M1.1"},
		"declared path Resolved(set); the set is what cites Addresses",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
		}},
		"m1_1_resolved.ndjson"),

	one([]string{"M1.2"},
		"declared path NoData; the name exists, nothing is cited",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: noerror()},
		}},
		"m1_2_nodata.ndjson"),

	one([]string{"M1.3"},
		"declared path NameError — the only outcome that withdraws a Name",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: nxdomain()},
		}},
		"m1_3_nameerror.ndjson"),

	one([]string{"M1.4"},
		"delegation walk Lame — every authority reached and refused; suppresses withdrawal",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: nxdomain()},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeNS, Name: nameEx, Reply: noerror(rrNS(nameEx, ns1), rrNS(nameEx, ns2))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns1, Reply: refused()},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns2, Reply: refused()},
		}},
		"m1_4_lame.ndjson"),

	one([]string{"M1.5"},
		"delegation walk not-Lame with the per-nameserver serves/does-not-serve RRset",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeNS, Name: nameEx, Reply: noerror(rrNS(nameEx, ns1), rrNS(nameEx, ns2))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns1, Reply: noerror()},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns2, Reply: refused()},
		}},
		"m1_5_partly.ndjson"),

	one([]string{"M2.a/NameError"},
		"a name that does not exist answers NXDOMAIN and withdraws",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: nxdomain()},
		}},
		"m2a_nameerror.ndjson"),
	one([]string{"M2.a/NoData"},
		"an empty non-terminal answers NOERROR with an empty ANSWER and does not withdraw",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: noerror()},
		}},
		"m2a_nodata.ndjson"),

	one([]string{"M2.b/NameError"},
		"a plain NXDOMAIN with no CNAME withdraws the Name",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: nxdomain()},
		}},
		"m2b_nameerror.ndjson"),
	one([]string{"M2.b/survives"},
		"an NXDOMAIN carrying a CNAME reflects the final name; the alias survives as NoData (RFC 6604)",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: nxdomain(rrCNAME(nameEx, "target.example.net"))},
		}},
		"m2b_survives.ndjson"),

	one([]string{"M2.c/Resolved"},
		"a CNAME chain resolvable to addresses on the declared path is Resolved",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrCNAME(nameEx, "c.example.net"), rrA("c.example.net", "203.0.113.9"))},
		}},
		"m2c_resolved.ndjson"),
	one([]string{"M2.c/NoData"},
		"a CNAME chain terminating with no address is NoData, not Resolved",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrCNAME(nameEx, "c.example.net"))},
		}},
		"m2c_nodata.ndjson"),

	one([]string{"M2.d/Resolved"},
		"TC=1 with a TCP fallback that recovers the RRset is Resolved",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Transport: resolutionwalk.UDP, Reply: truncated()},
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Transport: resolutionwalk.TCP, Reply: noerror(rrA(nameEx, "203.0.113.7"))},
		}},
		"m2d_resolved.ndjson"),
	one([]string{"M2.d/Gap"},
		"TC=1 with a TCP fallback that also truncates is a Gap, never a partial fold",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: truncated()},
		}},
		"m2d_gap.ndjson"),

	one([]string{"M2.e/Lame"},
		"every delegated authority REFUSEs — reached and does-not-serve — is Lame",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeNS, Name: nameEx, Reply: noerror(rrNS(nameEx, ns1), rrNS(nameEx, ns2))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns1, Reply: refused()},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns2, Reply: refused()},
		}},
		"m2e_lame.ndjson"),
	one([]string{"M2.e/Gap"},
		"every delegated authority silent — not reached — is a Gap, our own blindness",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeNS, Name: nameEx, Reply: noerror(rrNS(nameEx, ns1), rrNS(nameEx, ns2))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns1, Reply: silent()},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns2, Reply: silent()},
		}},
		"m2e_gap.ndjson"),

	one([]string{"M2.f/Lame"},
		"a fully refused delegation is Lame",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeNS, Name: nameEx, Reply: noerror(rrNS(nameEx, ns1), rrNS(nameEx, ns2))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns1, Reply: refused()},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns2, Reply: refused()},
		}},
		"m2f_lame.ndjson"),
	one([]string{"M2.f/not-Lame"},
		"a partly-lame delegation — one authority serves, one does not — is not Lame",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeNS, Name: nameEx, Reply: noerror(rrNS(nameEx, ns1), rrNS(nameEx, ns2))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns1, Reply: noerror()},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns2, Reply: refused()},
		}},
		"m2f_partly.ndjson"),

	{
		Cells:        []string{"M2.g/set"},
		Claim:        "an address set in canonical RR order and lower-case qname",
		SpecVerified: true,
		Offers:       def(),
		Steps: []Step{{Batch: "b1", Vantage: "v1", Resolver: resolver, Names: []string{"example.com"}, Peer: ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.1"), rrA(nameEx, "203.0.113.2"))},
		}}}},
		Golden: "m2g_set.ndjson",
	},
	{
		Cells:        []string{"M2.g/serialisation"},
		Claim:        "the same set in a different RR order under a 0x20-randomised qname folds identically",
		SpecVerified: true,
		Offers:       def(),
		Steps: []Step{{Batch: "b1", Vantage: "v1", Resolver: resolver, Names: []string{"ExAmPlE.CoM"}, Peer: ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.2"), rrA(nameEx, "203.0.113.1"))},
		}}}},
		Golden: "m2g_serialisation.ndjson",
	},

	one([]string{"M2.h/folds"},
		"an AAAA ::ffff:203.0.113.5 and an A 203.0.113.5 fold to one Address key",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeAAAA, Name: nameEx, Reply: noerror(rrAAAA(nameEx, "::ffff:203.0.113.5"))},
		}},
		"m2h_folds.ndjson"),
	one([]string{"M2.h/distinct"},
		"a genuine A and a genuine AAAA are two distinct Address keys",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeAAAA, Name: nameEx, Reply: noerror(rrAAAA(nameEx, "2001:db8::1"))},
		}},
		"m2h_distinct.ndjson"),

	one([]string{"M2.i/NoData"},
		"FORMERR to an OPT query with an EDNS-less retry that succeeds is NoData",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Name: nameEx, EDNS: boolPtr(true), Reply: formerr()},
			{Path: resolutionwalk.PathDeclared, Name: nameEx, EDNS: boolPtr(false), Reply: noerror()},
		}},
		"m2i_nodata.ndjson"),
	one([]string{"M2.i/Gap"},
		"FORMERR to an OPT query whose EDNS-less retry also fails is a Gap",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: formerr()},
		}},
		"m2i_gap.ndjson"),

	one([]string{"M3.1"},
		"Resolved is read from the declared path and Lame from the delegation walk; two disagreeing peers, each field from its own peer",
		ScriptPeer{Rules: []scriptRule{
			{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeNS, Name: nameEx, Reply: noerror(rrNS(nameEx, ns1))},
			{Path: resolutionwalk.PathWalk, Qtype: resolutionwalk.QtypeSOA, Server: ns1, Reply: refused()},
		}},
		"m3_1_provenance.ndjson"),

	{
		Cells:        []string{"R.1"},
		Claim:        "NameError then Resolved under one vector; the leaf names no transition",
		SpecVerified: true,
		Offers:       def(),
		Steps: []Step{
			{Batch: "b1", Vantage: "v1", Resolver: resolver, Names: []string{nameEx}, Peer: ScriptPeer{Rules: []scriptRule{
				{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: nxdomain()},
			}}},
			{Batch: "b2", Vantage: "v1", Resolver: resolver, Names: []string{nameEx}, Peer: ScriptPeer{Rules: []scriptRule{
				{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			}}},
		},
		Golden: "r_1_withdraw_return.ndjson",
	},
	{
		Cells:        []string{"R.2"},
		Claim:        "the same sequence with the leaf version moved between batches; output is byte-identical — the no-op proof",
		SpecVerified: true,
		Offers:       def(),
		Steps: []Step{
			{Batch: "b1", Vantage: "v1", Resolver: resolver, Names: []string{nameEx}, Peer: ScriptPeer{Rules: []scriptRule{
				{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			}}},
			{Batch: "b2", Vantage: "v1", Resolver: resolver, Names: []string{nameEx}, Peer: ScriptPeer{Rules: []scriptRule{
				{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			}}},
		},
		Golden: "r_2_noop.ndjson",
	},
	{
		Cells:        []string{"R.3"},
		Claim:        "NameError at one vantage and Resolved at another in the same batch; two per-vantage outputs and no fold",
		SpecVerified: true,
		Offers:       def(),
		Steps: []Step{
			{Batch: "b1", Vantage: "v-internet", Resolver: resolver, Names: []string{nameEx}, Peer: ScriptPeer{Rules: []scriptRule{
				{Path: resolutionwalk.PathDeclared, Name: nameEx, Reply: nxdomain()},
			}}},
			{Batch: "b1", Vantage: "v-internal", Resolver: resolver, Names: []string{nameEx}, Peer: ScriptPeer{Rules: []scriptRule{
				{Path: resolutionwalk.PathDeclared, Qtype: resolutionwalk.QtypeA, Name: nameEx, Reply: noerror(rrA(nameEx, "203.0.113.5"))},
			}}},
		},
		Golden: "r_3_two_vantages.ndjson",
	},
}
