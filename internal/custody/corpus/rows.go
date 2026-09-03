package corpus

// Relative to the constant the pair would follow a move and pin shape, not position (ADR-0008).

const (
	belowThreshold = 99
	atThreshold    = 100
)

type Resolution struct {
	Owner   string
	Address string
}

type Step struct {
	AddressScopes      []string
	AddressExclusions  []string
	ExtendedZones      []string
	Resolutions        []Resolution
	ScanInForce        bool
	ScanBatchCompleted bool
	Observed           map[string][]string
	Under              []string
}

type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Step         Step
	Golden       string
}

var AllCells = []string{
	"C1/below-threshold-reached", "C1/at-threshold-vetoed",
	"C2/seed-covered-at-threshold-operator", "C2/seed-covered-stays-a-candidate",
	"C3/pending-held", "C3/scan-not-in-force-reaches",
	"C3/extension-limb-errored-reaches", "C3/measured-candidate-holds-the-rest",
	"C4/excluded-inside-a-scope-is-third-party", "C4/the-same-address-without-the-exclusion",
	"C4/excluded-but-extension-reached-is-operator",
}

var Rows = []Row{
	{
		Cells:        []string{"C1/below-threshold-reached"},
		Claim:        "an edge whose SAN set reduces to 99 distinct registrable domains is not-shared: the custody extension reaches it, it derives operator, and the probing gate opens",
		SpecVerified: true,
		Step: Step{
			ExtendedZones:      []string{"example.com"},
			Resolutions:        []Resolution{{Owner: "www.example.com", Address: "93.184.216.34"}},
			ScanInForce:        true,
			ScanBatchCompleted: true,
			Observed:           map[string][]string{"93.184.216.34": SANsBelowThreshold()},
			Under:              []string{"93.184.216.34"},
		},
		Golden: "below_threshold.ndjson",
	},

	{
		Cells:        []string{"C1/at-threshold-vetoed"},
		Claim:        "an edge whose SAN set reduces to 100 distinct registrable domains is shared: the extension declines it, it derives third-party, the gate is shut, and it stays a candidate so a later measurement can lift the veto",
		SpecVerified: true,
		Step: Step{
			ExtendedZones:      []string{"example.com"},
			Resolutions:        []Resolution{{Owner: "www.example.com", Address: "104.16.132.229"}},
			ScanInForce:        true,
			ScanBatchCompleted: true,
			Observed:           map[string][]string{"104.16.132.229": SANsAtThreshold()},
			Under:              []string{"104.16.132.229"},
		},
		Golden: "at_threshold.ndjson",
	},

	// A session that makes the veto global moves this golden, and the gate names the row (ADR-0129).

	// The scope is a /24 on purpose: #956 refuses a specificity test at any prefix width.
	{
		Cells:        []string{"C2/seed-covered-at-threshold-operator", "C2/seed-covered-stays-a-candidate"},
		Claim:        "two shared edges in one estate: the one a declared address scope covers derives operator and is probed at any fan-out count, the one it does not is vetoed — the veto and the declaration are disjoint limbs, never ranked",
		SpecVerified: true,
		Step: Step{
			AddressScopes: []string{"104.16.132.0/24"},
			ExtendedZones: []string{"example.com"},
			Resolutions: []Resolution{
				{Owner: "www.example.com", Address: "104.16.132.10"},
				{Owner: "cdn.example.com", Address: "23.20.0.10"},
			},
			ScanInForce:        true,
			ScanBatchCompleted: true,
			Observed: map[string][]string{
				"104.16.132.10": SANsAtThreshold(),
				"23.20.0.10":    SANsAtThreshold(),
			},
			Under: []string{"104.16.132.10", "23.20.0.10"},
		},
		Golden: "seed_covered_and_veto.ndjson",
	},

	// Every row above carries a measurement, so only this one moves if the hold flips to a reach.
	{
		Cells:        []string{"C3/pending-held"},
		Claim:        "an in-force Scan that has completed no Batch HOLDS the reach: the address is a candidate, derives third-party, and queues no probe until a handshake clears it",
		SpecVerified: true,
		Step: Step{
			ExtendedZones:      []string{"example.com"},
			Resolutions:        []Resolution{{Owner: "www.example.com", Address: "93.184.216.35"}},
			ScanInForce:        true,
			ScanBatchCompleted: false,
			Observed:           nil,
			Under:              []string{"93.184.216.35"},
		},
		Golden: "measurement_pending.ndjson",
	},

	// The declared address is cited out of zone, so no measured candidate lifts the floor (#1018).

	// One bit from measurement_pending above, so a whole-store floor moves this golden.
	{
		Cells:        []string{"C3/extension-limb-errored-reaches"},
		Claim:        "a Scan that completes a Batch and records declaration-limb rows alone is ERRORED on the extension limb: its candidates reach and derive operator, and the declaration limb keeps its own measurement",
		SpecVerified: true,
		Step: Step{
			AddressScopes: []string{"23.20.0.0/24"},
			ExtendedZones: []string{"example.com"},
			Resolutions: []Resolution{
				{Owner: "www.example.com", Address: "93.184.216.36"},
				{Owner: "edge.provider.net", Address: "23.20.0.20"},
			},
			ScanInForce:        true,
			ScanBatchCompleted: true,
			Observed:           map[string][]string{"23.20.0.20": SANsAtThreshold()},
			Under:              []string{"93.184.216.36", "23.20.0.20"},
		},
		Golden: "extension_limb_errored.ndjson",
	},

	// A session reading the floor as any-unmeasured-reaches moves nothing above and moves this golden.
	{
		Cells:        []string{"C3/measured-candidate-holds-the-rest"},
		Claim:        "one measured extension candidate lifts the errored floor and the rest stay HELD: on an install whose Scan has run this limb, an unmeasured candidate is a lag bounded by the cadence, never a failure",
		SpecVerified: true,
		Step: Step{
			ExtendedZones: []string{"example.com"},
			Resolutions: []Resolution{
				{Owner: "www.example.com", Address: "93.184.216.37"},
				{Owner: "api.example.com", Address: "93.184.216.38"},
			},
			ScanInForce:        true,
			ScanBatchCompleted: true,
			Observed:           map[string][]string{"93.184.216.37": SANsBelowThreshold()},
			Under:              []string{"93.184.216.37", "93.184.216.38"},
		},
		Golden: "measured_candidate_holds_the_rest.ndjson",
	},

	// The loud, wasteful direction is the one ADR-0129 §2 accepts; the silent one it refuses.
	{
		Cells:        []string{"C3/scan-not-in-force-reaches"},
		Claim:        "a Scan that is not in force reaches the address even where a shared measurement is on record: case 4 is the pre-ADR-0129 behaviour and the zero value, so an estate assembled without this input probes what it probed before",
		SpecVerified: true,
		Step: Step{
			ExtendedZones:      []string{"example.com"},
			Resolutions:        []Resolution{{Owner: "www.example.com", Address: "104.16.132.230"}},
			ScanInForce:        false,
			ScanBatchCompleted: true,
			Observed:           map[string][]string{"104.16.132.230": SANsAtThreshold()},
			Under:              []string{"104.16.132.230"},
		},
		Golden: "scan_not_in_force.ndjson",
	},

	{
		Cells:        []string{"C4/excluded-inside-a-scope-is-third-party"},
		Claim:        "an address inside a declared address scope AND inside a declared address exclusion is covered by no scope: it derives third-party and the probing gate is shut",
		SpecVerified: true,
		Step: Step{
			AddressScopes:      []string{"93.184.217.0/24"},
			AddressExclusions:  []string{"93.184.217.0/29"},
			Resolutions:        []Resolution{{Owner: "edge.provider.net", Address: "93.184.217.5"}},
			ScanInForce:        true,
			ScanBatchCompleted: true,
			Under:              []string{"93.184.217.5"},
		},
		Golden: "excluded_inside_a_scope.ndjson",
	},

	{
		Cells:        []string{"C4/the-same-address-without-the-exclusion"},
		Claim:        "the same address with the exclusion ABSENT is covered by the declared scope, derives operator and is probed: the refusal above is the exclusion and not the fixture",
		SpecVerified: true,
		Step: Step{
			AddressScopes:      []string{"93.184.217.0/24"},
			Resolutions:        []Resolution{{Owner: "edge.provider.net", Address: "93.184.217.5"}},
			ScanInForce:        true,
			ScanBatchCompleted: true,
			Under:              []string{"93.184.217.5"},
		},
		Golden: "exclusion_removed.ndjson",
	},

	// A session making the exclusion global moves this golden, and the gate names the row (ADR-0133).
	{
		Cells:        []string{"C4/excluded-but-extension-reached-is-operator"},
		Claim:        "an excluded address a custody extension ALSO reaches derives operator and is probed, while its excluded sibling that no extension reaches derives third-party: the exclusion cuts the Seed limb alone",
		SpecVerified: true,
		Step: Step{
			AddressScopes:     []string{"104.16.140.0/24"},
			AddressExclusions: []string{"104.16.140.0/24"},
			ExtendedZones:     []string{"example.com"},
			Resolutions: []Resolution{
				{Owner: "www.example.com", Address: "104.16.140.20"},
				{Owner: "edge.provider.net", Address: "104.16.140.21"},
			},
			ScanInForce:        true,
			ScanBatchCompleted: true,
			Observed:           map[string][]string{"104.16.140.20": SANsBelowThreshold()},
			Under:              []string{"104.16.140.20", "104.16.140.21"},
		},
		Golden: "excluded_but_extension_reached.ndjson",
	},
}
