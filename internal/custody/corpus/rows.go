package corpus

// belowThreshold and atThreshold are the two sides of the fan-out boundary, as
// ADR-0129's #955 amendment states it: a SAN set reducing to 99 distinct
// registrable domains derives `not-shared` and the edge is reached; one reducing
// to 100 derives `shared` and the edge is vetoed.
//
// They are ABSOLUTE INTEGERS, deliberately not `custody.SharedEdgeThreshold` and
// `custody.SharedEdgeThreshold - 1`. Written relative to the constant they would
// follow a threshold move and keep straddling it, so the pair would pin the SHAPE
// of the boundary and never its POSITION — the one thing ADR-0008 asks a declared
// parameter's corpus row to hold still. Written absolutely, a move to 99 flips the
// first row's verdict, its golden and the corpus digest at once.
//
// TestFixtureStraddlesTheThreshold pins them against the shipped constant, so a
// session that moves the value is told here, by name, that it owes a
// `custody.Version` bump and a re-bless.
const (
	belowThreshold = 99
	atThreshold    = 100
)

// Resolution is one observed direct A/AAAA record in a row's estate, in the
// spelling a row is authored in: an owner Name holding an address literal.
type Resolution struct {
	Owner   string
	Address string
}

// Step is one run of the derivation inside a row: an estate, the `edge-fanout`
// Scan's disposition, what that Scan observed, and the addresses the row renders.
type Step struct {
	// AddressScopes are the declared address-scope `Seed`s, as CIDR literals.
	AddressScopes []string
	// ExtendedZones are the registrable domains of custody-extended name scopes.
	ExtendedZones []string
	// Resolutions are the observed direct A/AAAA records.
	Resolutions []Resolution
	// ScanInForce is `edge-fanout`'s DISPOSITION — EdgeFanout.Enabled. False is a
	// disabled Scan and a Scan whose row is absent.
	ScanInForce bool
	// ScanBatchCompleted is whether a Batch of the Scan has ever completed —
	// EdgeFanout.BatchCompleted. It is what tells the two unmeasured states apart: a
	// Scan that has not run yet HOLDS its extension candidates, and one that has run
	// and measured none of them is ERRORED on that limb and reaches (#1018).
	ScanBatchCompleted bool
	// Observed maps an address literal to the SAN set the Scan read off its
	// default certificate. An address ABSENT from this map is measurement
	// PENDING, which is the whole of the row that pins the hold.
	Observed map[string][]string
	// Under are the addresses the row renders, one NDJSON line each, in this order.
	Under []string
}

// Row is one corpus row: the cells it pins, its one-line claim, whether the claim
// is spec-verified rather than measured, the step, and the golden NDJSON file its
// rendering must equal byte for byte.
type Row struct {
	Cells        []string
	Claim        string
	SpecVerified bool
	Step         Step
	Golden       string
}

// AllCells is the enumeration the coverage test counts against: every cell of the
// `Custody` block must be pinned by at least one row (golden-corpus.md §10).
var AllCells = []string{
	// C1 — the fan-out boundary, one row on each side (ADR-0129's #955 amendment)
	"C1/below-threshold-reached", "C1/at-threshold-vetoed",
	// C2 — the `Seed` limb is disjoint from the veto (ADR-0129's #956 amendment)
	"C2/seed-covered-at-threshold-operator", "C2/seed-covered-stays-a-candidate",
	// C3 — absence is hold-then-open, and the fall back is the Scan not in force.
	// The errored floor is PER LIMB (#1018): a declaration-limb row does not lift it,
	// and one MEASURED CANDIDATE does.
	"C3/pending-held", "C3/scan-not-in-force-reaches",
	"C3/extension-limb-errored-reaches", "C3/measured-candidate-holds-the-rest",
}

// Rows is the checked-in corpus. Every cell in AllCells appears in some row's
// Cells; the coverage test fails the build (naming the cell) if one does not.
var Rows = []Row{
	// ---- C1, below the boundary ----
	// One in-zone name's direct-A target, measured at 99 distinct registrable
	// domains. 99 is under the threshold, so the edge is NOT shared: the custody
	// extension reaches it, it derives `operator`, and the gate opens.
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

	// ---- C1, at the boundary ----
	// The same estate one domain higher. 100 is AT the threshold and the
	// comparison is `>=`, so the edge is shared: the extension declines the
	// reach, the address derives `third-party`, and the gate is shut. It stays an
	// `edge-fanout` candidate, so a later dedicated certificate lifts the veto.
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

	// ---- C2: the two limbs are disjoint, never ranked ----
	// ONE estate, ONE measurement, TWO addresses. Both are in-zone direct-A
	// targets and both measure shared at 100. A declared address scope covers the
	// first and not the second, and that declaration alone decides the outcome:
	// the covered address derives `operator` and is probed, the other is vetoed.
	//
	// This row is the strongest guard the ADR-0129 map leaves behind. A session
	// that "repairs" the apparent inconsistency by making the veto global moves
	// the first address to `third-party`, which moves this golden and the corpus
	// digest, and the gate names the row.
	//
	// The scope is a /24 and not a /32 ON PURPOSE. ADR-0129's #956 amendment
	// REFUSES a specificity test — the declaration does not win at a /32 and lose
	// inside a wider prefix — so the row declares a wide one and still expects
	// `operator`.
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

	// ---- C3, the hold ----
	// The Scan is in force and has recorded nothing for this address. Absence is
	// hold-then-open (ADR-0129 §4, case 3): the extension neither reaches nor
	// declines, so the address derives `third-party` and no probe is queued until
	// a handshake clears it. Nothing here is a new membership state — a held
	// address is simply not reached.
	//
	// The row exists because the three rows above all carry a measurement, so
	// none of them would move if a session flipped the hold to a reach. That is
	// exactly the silent move the corpus is for.
	//
	// `ScanBatchCompleted` is FALSE and load-bearing (#1018). This is the fresh
	// install: the Scan has not run, so its candidates are genuinely pending and the
	// errored floor must not fire. The row below is the same estate one completed
	// Batch later, and it reaches.
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

	// ---- C3, the errored EXTENSION LIMB ----
	// The Scan is in force, it has completed a Batch, and it recorded a row on the
	// DECLARATION limb alone. No extension candidate was measured, so the extension
	// limb is ERRORED: it reaches every candidate, which is case 4 and the
	// pre-ADR-0129 behaviour (#1018).
	//
	// The estate holds BOTH limbs, because that is the only shape the bug has. The
	// declared address is cited by an OUT-of-zone owner, so it is a declaration-limb
	// address and not an extension candidate — a measured candidate would lift the
	// floor and the row would pin nothing.
	//
	// Read this row against `measurement_pending` above. The two estates differ in
	// ONE bit — whether a Batch has completed — and the extension candidate holds in
	// the first and reaches here. A session that puts the floor back on the whole
	// store moves this golden: the declaration-limb row lifts a whole-store floor,
	// 93.184.216.36 goes back to `third-party`, and the gate names the row.
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

	// ---- C3, the hold on an ESTABLISHED install ----
	// The Scan is in force, it has completed a Batch, and it measured ONE of two
	// extension candidates. That one measured candidate LIFTS the errored floor, so
	// the other is an ordinary unmeasured candidate and is HELD by case 3.
	//
	// This is the row that stops the floor from widening. `measurement_pending` above
	// pins the hold on an install that has completed no Batch, and a session that
	// read the new floor as *any unmeasured candidate reaches* would move nothing
	// there — every candidate in that row is unmeasured and the Scan has not run. It
	// moves HERE: 93.184.216.38 goes to `operator`, its gate opens, and the golden
	// and the digest move with it.
	//
	// It is also the shape a PARTIAL failure takes. The Scan demonstrably measured
	// this limb, so a candidate without a row is a lag bounded by the daily cadence
	// and never an error, and holding it is what the cadence is for.
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

	// ---- C3, the fall back ----
	// The Scan's DISPOSITION is off — disabled, or its row absent — and a
	// shared measurement is on record anyway. Case 4 is the only fall back to
	// reach-everything, so the extension reaches the address and the address
	// derives `operator` DESPITE the shared verdict the row still renders. That is
	// the pre-ADR-0129 behaviour, and it is the loud, wasteful direction §2
	// accepts rather than the silent one.
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
}
