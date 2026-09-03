// This file builds the `edge-fanout` Scan (CONTEXT.md `Scan`, ADR-0129 §6, tickets
// #983 and #988): the seventh Scan, whose no-SNI TLS handshake reads the certificate an
// edge serves to a client that names nothing.
//
// Its scope is custody.Estate.EdgeFanoutPopulation: the custody-extension candidates —
// the direct-A targets, and the apex `ALIAS`/`ANAME` flattened to A, of in-zone names
// the extension would reach — and, since ticket #988, every address a declared address
// scope covers. On the first limb the result decides membership; on the second it labels
// and decides nothing. It has NO vantage dimension: a default certificate is not a
// function of vantage, and vantage-varying fan-out is anycast, out of v1 (§5). It has no
// port list either — the edge is measured on 443/tcp alone.
//
// It carries no consent dial of its own. #983 rested that on two clauses, and #988
// WITHDREW THE SECOND — *it reduces total probing* does not hold on a declared address,
// where the handshake narrows nothing and adds a connect. The first clause carries the
// authority alone: the probing gate is total over an `Address`, so an address a `Seed`
// covers is already connected to on every port, and one further handshake asks for no
// authority the declaration did not already give. Like `ct` it holds no facet timeline,
// so it carries no currency bound and withdraws no value.
package scan

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/wire"
)

// EdgeFanoutKind is the DB Scan kind this file dispatches, and the JobSpec kind that
// selects the leaf. Unlike `http-identity` — whose Scan dispatches a differently named
// leaf — the Scan and the leaf share one name here, as `tls-acceptance` does, so it is
// bound to the leaf's own constant rather than restated.
const EdgeFanoutKind = edgefanout.Kind

// EdgeFanoutAddressesPerJob bounds how many candidate edges one job measures. The leaf
// handshakes its scope serially under a per-handshake timeout, and the worker brackets
// one probe with DefaultProbeTimeout (5 minutes); a scope large enough to outlast that
// bound would be killed mid-measurement and retried whole, measuring the early edges
// again and the late ones never. Chunking keeps one job's worst case well inside the
// bound, and the chunks are independent — one slow edge delays its own job alone.
const EdgeFanoutAddressesPerJob = 50

// EdgeFanoutJob is one queue job the edge-fanout Scan produces: a bounded set of
// candidate edge addresses to handshake. It carries no Vantage (the Scan has no vantage
// dimension) and no offers beyond the empty object — the leaf declares no parameter an
// operator chooses, since the fan-out reduction and the `shared-edge` threshold are
// versioned parameters of the `Custody` derivation and live there (#984).
type EdgeFanoutJob struct {
	ScanID int64
	// Chunk is the job's ordinal within its tick, and it names the Batch. The Scan has
	// no vantage and no seed to key a Batch on, and the fan-out is streamed, so there is
	// no slice index to read one from.
	Chunk     int
	Addresses []string
}

// BuildEdgeFanoutJobs fans an edge-fanout Scan out over the measured population, in
// chunks of EdgeFanoutAddressesPerJob, as a LAZY SEQUENCE. It produces no job for an
// empty population — an instance with neither a custody extension nor an address scope
// has nothing to measure, and an aperture over an empty scope is a legible state, not an
// error (CONTEXT.md `Scan`).
//
// It streams because #988 made this an ADDRESS-SCOPE tier. The chunk bound below is the
// per-JOB load, and until #988 the whole dispatch was bounded by the resolution set, so
// a slice of jobs was safe. It is not any more: ADR-0127 removed the ceiling above the
// operator's address cap, so a declared scope can be an IPv4 `/8`, and ADR-0047 refuses
// a scan-time aperture that would truncate one. So no record holds the whole scope —
// neither the population, nor the jobs, nor the enqueue transaction (queue.streamEnqueue
// commits in chunks).
//
// Chunk numbers a job within its tick, so the Batch label stays legible with no slice to
// index. It restarts at 0 each tick, exactly as the old index did.
//
// Addresses are rendered in their netip form, so the scope a job carries and the
// address a resulting observation names are the same spelling and the recording-side
// scope gate cannot reject a legitimate row over a rendering.
func BuildEdgeFanoutJobs(scanID int64, population iter.Seq[netip.Addr]) iter.Seq[EdgeFanoutJob] {
	return func(yield func(EdgeFanoutJob) bool) {
		chunk := 0
		addrs := make([]string, 0, EdgeFanoutAddressesPerJob)
		for a := range population {
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

// JobSpec renders an EdgeFanoutJob into the wire JobSpec the edge-fanout prober leaf
// reads (edgefanout.DecodeScope): the candidate addresses alone. No vantage, no port
// list and no name list travel with it — the handshake sends no server name, so no name
// selects what the edge serves.
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

// OffersJSON is the empty object: this Scan declares no measurement parameter. The
// fan-out threshold is not an offer here — it is a versioned parameter of the `Custody`
// derivation, applied where that derivation runs (#984), never a knob the probe carries.
func (j EdgeFanoutJob) OffersJSON() ([]byte, error) { return []byte("{}"), nil }

// edgeFanoutScope is the wire payload an edge-fanout job carries. The field name
// matches the leaf's own Scope (edgefanout.Scope) and the `addresses` key the
// recording-side scope gate reads, so the dispatched scope, the measured scope and the
// authorised denotation are one shape.
type edgeFanoutScope struct {
	Addresses []string `json:"addresses"`
}
