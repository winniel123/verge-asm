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
	// ScanInForce is `edge-fanout`'s disposition and health together — EdgeFanout.Enabled.
	ScanInForce bool
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
	// C3 — absence is hold-then-open, and the fall back is the Scan not in force
	"C3/pending-held", "C3/scan-not-in-force-reaches",
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
			ExtendedZones: []string{"example.com"},
			Resolutions:   []Resolution{{Owner: "www.example.com", Address: "93.184.216.34"}},
			ScanInForce:   true,
			Observed:      map[string][]string{"93.184.216.34": SANsBelowThreshold()},
			Under:         []string{"93.184.216.34"},
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
			ExtendedZones: []string{"example.com"},
			Resolutions:   []Resolution{{Owner: "www.example.com", Address: "104.16.132.229"}},
			ScanInForce:   true,
			Observed:      map[string][]string{"104.16.132.229": SANsAtThreshold()},
			Under:         []string{"104.16.132.229"},
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
			ScanInForce: true,
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
	{
		Cells:        []string{"C3/pending-held"},
		Claim:        "an in-force Scan that has not yet measured an edge HOLDS the reach: the address is a candidate, derives third-party, and queues no probe until a handshake clears it",
		SpecVerified: true,
		Step: Step{
			ExtendedZones: []string{"example.com"},
			Resolutions:   []Resolution{{Owner: "www.example.com", Address: "93.184.216.35"}},
			ScanInForce:   true,
			Observed:      nil,
			Under:         []string{"93.184.216.35"},
		},
		Golden: "measurement_pending.ndjson",
	},

	// ---- C3, the fall back ----
	// The Scan is NOT in force — disabled, its row absent, or errored — and a
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
			ExtendedZones: []string{"example.com"},
			Resolutions:   []Resolution{{Owner: "www.example.com", Address: "104.16.132.230"}},
			ScanInForce:   false,
			Observed:      map[string][]string{"104.16.132.230": SANsAtThreshold()},
			Under:         []string{"104.16.132.230"},
		},
		Golden: "scan_not_in_force.ndjson",
	},
}
