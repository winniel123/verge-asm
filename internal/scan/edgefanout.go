// A no-SNI handshake reads what an edge serves a client that names nothing (ADR-0129 §6).
// It carries no consent dial: the probing gate is already total over a covered `Address` (#988).
// It has no vantage dimension — vantage-varying fan-out is anycast, out of v1 (§5).
package scan

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/wire"
)

const EdgeFanoutKind = edgefanout.Kind

// A scope outlasting the worker's 5-minute probe bracket is killed mid-run and retried whole.

const EdgeFanoutAddressesPerJob = 50

type EdgeFanoutJob struct {
	ScanID    int64
	Chunk     int // names the Batch: this Scan has no vantage or seed to key one on
	Addresses []string
}

func BuildEdgeFanoutJobs(scanID int64, population iter.Seq[netip.Addr]) iter.Seq[EdgeFanoutJob] {
	return func(yield func(EdgeFanoutJob) bool) {
		// A declared scope can be an IPv4 /8, and no aperture may truncate one (ADR-0127, ADR-0047).
		chunk := 0
		addrs := make([]string, 0, EdgeFanoutAddressesPerJob)
		for a := range population {
			// The rendering must match what an observation names, or the recording gate drops the row.
			addrs = append(addrs, a.Unmap().String())
			if len(addrs) < EdgeFanoutAddressesPerJob {
				continue
			}
			if !yield(EdgeFanoutJob{ScanID: scanID, Chunk: chunk, Addresses: addrs}) {
				return
			}
			chunk++
			addrs = make([]string, 0, EdgeFanoutAddressesPerJob)
		}
		if len(addrs) > 0 {
			yield(EdgeFanoutJob{ScanID: scanID, Chunk: chunk, Addresses: addrs})
		}
	}
}

func (j EdgeFanoutJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(edgeFanoutScope{Addresses: j.Addresses})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal edge-fanout scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: EdgeFanoutKind, Scope: raw}, nil
}

func (j EdgeFanoutJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(edgeFanoutScope{Addresses: j.Addresses})
}

// The fan-out threshold is a versioned `Custody` parameter, never a knob the probe carries (#984).

func (j EdgeFanoutJob) OffersJSON() ([]byte, error) { return []byte("{}"), nil }

// The `addresses` key is read by the recording-side scope gate, so it must match the leaf's Scope.

type edgeFanoutScope struct {
	Addresses []string `json:"addresses"`
}
