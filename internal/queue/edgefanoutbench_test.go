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

// An offset from custody.SharedEdgeThreshold moves with it, so two releases could not be compared.

const (
	benchDedicatedSANs = 5
	benchSharedSANs    = 400
)

type benchSANShape struct {
	name   string
	sans   int
	shared bool
}

type benchCertShape struct {
	name           string
	oneCertificate bool
}

// Boxing into an any would heap-allocate once per iteration and land on allocs/op.

var benchEdgeFanoutSink custody.EdgeFanout

func BenchmarkToEdgeFanout(b *testing.B) {
	// A benchmark through a database would measure the database, so the subject is the pure half.
	sanShapes := []benchSANShape{
		{name: "dedicated", sans: benchDedicatedSANs, shared: false},
		{name: "shared", sans: benchSharedSANs, shared: true},
	}
	// The two cells agreeing again is the regression this axis exists to catch (#1035).
	certShapes := []benchCertShape{
		{name: "one-certificate", oneCertificate: true},
		{name: "unique-certificates", oneCertificate: false},
	}
	for _, san := range sanShapes {
		for _, cert := range certShapes {
			// edge_fanout_observation is never pruned, so the row count grows with the estate (#985).
			for _, rows := range []int{10, 100, 1000, 5000} {
				name := fmt.Sprintf("sans=%s/certs=%s/rows=%d", san.name, cert.name, rows)
				b.Run(name, func(b *testing.B) {
					// Built inside b.Run so a filtered run generates that cell's certificates alone.
					fixture, material := benchEdgeFanoutFixture(b, rows, san, cert)
					// wire-bytes is the total DER the reads return for the cell, never a per-iteration figure.
					wire := 0
					for _, der := range material {
						wire += len(der)
					}
					benchCheckFixture(b, fixture, material, rows, san)

					// GOGC paces the collector against the live heap, so ns/op carries no precise speedup.
					b.ReportAllocs()
					for b.Loop() {
						benchEdgeFanoutSink = toEdgeFanout(true, fixture, material)
					}
					// b.Loop's first call clears the reported-metric map, so a metric reported above never lands.
					b.ReportMetric(float64(wire), "wire-bytes")
				})
			}
		}
	}
}

func benchCheckFixture(b *testing.B, fixture []db.ListEdgeFanoutMeasurementsRow, material map[string][]byte, rows int, san benchSANShape) {
	b.Helper()
	// A degraded fixture measures as a fast one, because the reduction fails silently (#1014).
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
	// A threshold move past these counts must be told by name, not left mislabelling a cell.
	for _, r := range fixture {
		addr := netip.MustParseAddr(r.Address)
		if got.Shared[addr] != san.shared {
			b.Fatalf("a %s edge of %d SANs reduced to shared=%v, want %v "+
				"(custody.SharedEdgeThreshold is %d): move the SAN count, not the threshold",
				san.name, san.sans, got.Shared[addr], san.shared, custody.SharedEdgeThreshold)
		}
	}
}

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

func benchAddress(i int) string {
	return fmt.Sprintf("10.%d.%d.%d", i/65536%256, i/256%256, i%256)
}
