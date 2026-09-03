package queue

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
)

// This file measures toEdgeFanout, the pure reduction half of ReadEdgeFanout (#1014).
//
// #987 put that reduction on the `/scope` render: renderSeeds assembles a
// custody.Estate per request, and the estate reaches the measurement through
// queue.ReadEdgeFanout. #1018's second census put the same reduction on `/coverage`.
// The dispatcher pays it once per daily tick; the two renders pay it per request, per
// logged-in operator.
//
// #1014 argued the cost from the query and the reduction alone. No profile was taken.
// This benchmark supplied the number, the fix rested on it (#1035), and it stays in the
// tree so the next change to the reduction is measured against the same cells. It TAKES
// NO FIX of its own: nothing here caches, narrows a query, or stores a verdict.
//
// The subject is the pure half on purpose. toEdgeFanout takes the `completed` boolean,
// the row slice the measurement query generates and the material map the second read
// generates, and needs no database, so the number is the parse and the Public Suffix
// List reduction and nothing else. A benchmark through a database would measure the
// database.

// The two SAN counts the benchmark contrasts. They are the two bands ADR-0129 §1's
// threshold sits between: an estate's own edge presents a handful of identities, and a
// shared CDN edge presents hundreds to thousands.
//
// They are ABSOLUTE INTEGERS, in the way internal/custody/corpus/rows.go authors its
// own pair and for a related reason. Written as offsets from
// custody.SharedEdgeThreshold they would follow a threshold move, and the cost this
// benchmark reports would move with it — so two releases' numbers could not be
// compared, which is the whole point of leaving the benchmark in the tree.
// benchCheckFixture pins both against the shipped constant instead, so a session that
// moves the threshold past either count is told by name.
const (
	// benchDedicatedSANs is the TOP of #1014's "1 to 5 dNSName SANs" band. The top is
	// the conservative end: it charges the dedicated shape the most the band allows,
	// so the contrast against the shared shape is not flattered. The band is sampled
	// at this one point and not swept — the row count is the axis that matters, and a
	// sweep of four more counts inside a 5-SAN band would move no number.
	benchDedicatedSANs = 5
	// benchSharedSANs is "several hundred unrelated registrable domains". It sits far
	// above custody.SharedEdgeThreshold, so every such row reduces to `shared` and the
	// reduction walks the whole SAN set.
	benchSharedSANs = 400
)

type benchSANShape struct {
	name string
	// sans is the count of dNSName SANs, each on its own registrable domain, so the
	// fan-out count equals this value exactly.
	sans   int
	shared bool
}

// benchCertShape is how a cell spreads certificates over its rows — the axis #1035
// acted on, and the axis a later change to the reduction is judged against.
type benchCertShape struct {
	name string
	// oneCertificate points every row at a single fingerprint. Otherwise each row
	// names its own.
	oneCertificate bool
}

// benchEdgeFanoutSink holds the reduction's result so the compiler cannot discard the
// call it is timing. It is TYPED, never `any`: boxing custody.EdgeFanout into an
// interface heap-allocates once per iteration, and it would land on allocs/op and
// B/op — the two columns the certificate axis below is read off.
var benchEdgeFanoutSink custody.EdgeFanout

// BenchmarkToEdgeFanout measures the edge-fanout reduction over generated rows, across
// three axes:
//
//   - Row count: 10, 100, 1000 and 5000. The row count is the count of addresses the
//     Scan ever measured, and `edge_fanout_observation` is never pruned (#985), so it
//     grows with the estate and with time.
//   - SAN shape: a dedicated edge and a shared edge. This is what the Public Suffix
//     List reduction walks.
//   - Certificate shape: every row naming the SAME certificate, and every row naming a
//     unique one. `certificate_material` is keyed by fingerprint, so many addresses on
//     one shared CDN edge name — and, before #1035, re-parsed — one certificate many
//     times over.
//
// THE CERTIFICATE AXIS IS THE FIX'S OWN MEASUREMENT. Before #1035 the two cells agreed
// on allocs/op and B/op to within 0.001% at every row count: the reduction keyed on the
// address and never on the fingerprint, so none of the repetition was exploited and all
// of it was headroom. Since #1035 the one-certificate cell parses ONE certificate at
// every row count, so it must now cost far less than the unique-certificates cell at
// the same row count. THE TWO CELLS AGREEING AGAIN IS A REGRESSION, and it is what this
// axis exists to catch.
//
// Read that axis off allocs/op and B/op. Its ns/op now moves the same way, and #1035
// made the columns agree in DIRECTION where they used to disagree — but do not read a
// precise speedup off ns/op. The fixtures' live heaps still differ by orders of
// magnitude (one shared DER against N unique ones), GOGC paces the collector against
// the live heap, and the small-heap cell therefore collects more often than its own
// allocation count implies. The allocation columns carry no such term.
//
// Every row carries the `presented` outcome. A negative outcome names no certificate
// and is not parsed at all, so a mix would only dilute the number this issue asks for.
//
// The `wire-bytes` metric is the TOTAL DER byte count the reads return for that cell,
// not a per-iteration figure. Since #1035 that is one DER per DISTINCT certificate,
// where it was one per row, so the one-certificate cells report a flat figure at every
// row count. It is the wire cost #1014 argued about, and it is exact for generated data.
//
// It runs under -bench alone, so no CI job gets slower.
func BenchmarkToEdgeFanout(b *testing.B) {
	sanShapes := []benchSANShape{
		{name: "dedicated", sans: benchDedicatedSANs, shared: false},
		{name: "shared", sans: benchSharedSANs, shared: true},
	}
	certShapes := []benchCertShape{
		{name: "one-certificate", oneCertificate: true},
		{name: "unique-certificates", oneCertificate: false},
	}
	for _, san := range sanShapes {
		for _, cert := range certShapes {
			for _, rows := range []int{10, 100, 1000, 5000} {
				name := fmt.Sprintf("sans=%s/certs=%s/rows=%d", san.name, cert.name, rows)
				b.Run(name, func(b *testing.B) {
					// The fixture is built INSIDE b.Run, so a filtered run — say
					// -bench '.../rows=10' — generates that cell's certificates alone
					// and not all sixteen cells'. It is UNTIMED wherever it sits:
					// b.Loop's first call resets the timer, so everything above it is
					// setup. A -count run rebuilds it once per repeat, which is the
					// price of the filter and is paid outside the measurement.
					fixture, material := benchEdgeFanoutFixture(b, rows, san, cert)
					wire := 0
					for _, der := range material {
						wire += len(der)
					}
					benchCheckFixture(b, fixture, material, rows, san)

					b.ReportAllocs()
					for b.Loop() {
						benchEdgeFanoutSink = toEdgeFanout(true, fixture, material)
					}
					// AFTER the loop, not before. b.Loop's first call resets the
					// timer, and ResetTimer clears the reported-metric map — a
					// wire-bytes reported ahead of the loop never reaches the output.
					b.ReportMetric(float64(wire), "wire-bytes")
				})
			}
		}
	}
}

// benchCheckFixture fails the cell unless the reduction read EVERY row and reached the
// verdict the SAN shape names. It runs once, before the timed loop, so it costs the
// measurement nothing.
//
// toEdgeFanout drops an unparseable address silently, and reduces an absent or
// undecodable DER to a fan-out of zero. Both are the right absence rules for the read
// path — withholding the probe is the safe direction — and both make a DEGRADED fixture
// measure as a FAST one rather than fail. A benchmark that had quietly stopped parsing
// certificates would still report the number #1014 asked for, and would mean nothing by
// it. Since #1035 there is a second way to degrade: a row naming a fingerprint the
// material map does not hold parses nothing at all, and the count check below catches a
// fixture whose two halves have drifted apart.
//
// The verdict half also pins benchSharedSANs and benchDedicatedSANs against the shipped
// custody.SharedEdgeThreshold, as internal/custody/corpus does for its own absolute
// integers. A session moving the threshold past 400 is told here, by name, rather than
// left with a `shared` cell that stopped being shared while its label still said it was.
func benchCheckFixture(b *testing.B, fixture []db.ListEdgeFanoutMeasurementsRow, material map[string][]byte, rows int, san benchSANShape) {
	b.Helper()
	for _, r := range fixture {
		if _, held := material[r.Fingerprint.String]; !held {
			b.Fatalf("address %s names fingerprint %q the material map does not hold: "+
				"this cell would measure a parse it never did", r.Address, r.Fingerprint.String)
		}
	}
	got := toEdgeFanout(true, fixture, material)
	if len(got.Shared) != rows {
		b.Fatalf("the reduction keyed %d of %d rows: the fixture holds an address it cannot read, "+
			"so this cell would measure a parse it never did", len(got.Shared), rows)
	}
	for _, r := range fixture {
		addr := netip.MustParseAddr(r.Address)
		if got.Shared[addr] != san.shared {
			b.Fatalf("a %s edge of %d SANs reduced to shared=%v, want %v "+
				"(custody.SharedEdgeThreshold is %d): move the SAN count, not the threshold",
				san.name, san.sans, got.Shared[addr], san.shared, custody.SharedEdgeThreshold)
		}
	}
}

// benchEdgeFanoutFixture builds one cell in the shape ReadEdgeFanout's two reads
// return: one measurement row per address, naming its certificate by fingerprint, and
// one material entry per DISTINCT certificate (#1035).
//
// A one-certificate cell points EVERY row at one fingerprint and the map holds ONE DER,
// which is what the two reads return when many measured addresses sit behind one shared
// edge. The old fixture repeated that DER once per row, because the old query did.
//
// Both cert shapes carry the SAME SAN set. That is deliberate: the two cells then differ
// in one thing only — whether the parse and the reduction are done once per distinct
// fingerprint or once per address.
func benchEdgeFanoutFixture(tb testing.TB, rows int, san benchSANShape, cert benchCertShape) ([]db.ListEdgeFanoutMeasurementsRow, map[string][]byte) {
	tb.Helper()
	sans := distinctSANs(san.sans)
	var one []byte
	if cert.oneCertificate {
		one = certWithSANs(tb, sans...)
	}
	out := make([]db.ListEdgeFanoutMeasurementsRow, 0, rows)
	material := make(map[string][]byte, rows)
	for i := range rows {
		der := one
		if !cert.oneCertificate {
			der = certWithSANs(tb, sans...)
		}
		fingerprint := co.Fingerprint(der)
		material[fingerprint] = der
		out = append(out, db.ListEdgeFanoutMeasurementsRow{
			Address:     benchAddress(i),
			Outcome:     string(edgefanout.Presented),
			Fingerprint: pgtype.Text{String: fingerprint, Valid: true},
		})
	}
	return out, material
}

// benchAddress renders the i-th distinct measured address. It walks 10.0.0.0/8, which
// holds 16,777,216 addresses and so has room far past this benchmark's 5000-row ceiling.
// toEdgeFanout parses the rendering back, so every value must be one netip.ParseAddr
// accepts, and benchCheckFixture's key count catches a collision if a later row count
// ever outgrows the range.
func benchAddress(i int) string {
	return fmt.Sprintf("10.%d.%d.%d", i/65536%256, i/256%256, i%256)
}
