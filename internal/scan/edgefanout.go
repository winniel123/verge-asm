// This file builds the `edge-fanout` Scan (CONTEXT.md `Scan`, ADR-0129 §6, ticket
// #983): the seventh Scan, whose no-SNI TLS handshake reads the certificate a candidate
// edge serves to a client that names nothing.
//
// Its scope is the custody-extension candidates — the direct-A targets, and the apex
// `ALIAS`/`ANAME` flattened to A, of in-zone names the extension would reach
// (custody.Estate.ExtensionCandidates). It has NO vantage dimension: a default
// certificate is not a function of vantage, and vantage-varying fan-out is anycast,
// out of v1 (§5). It has no port list either — the edge is measured on 443/tcp alone.
//
// It carries no consent dial of its own. The one handshake is a strict subset of the
// probing the custody extension already authorises, run one step earlier, and it
// reduces total probing; a pure narrowing needs no widening-style control. Like `ct` it
// holds no facet timeline, so it carries no currency bound and withdraws no value.
package scan

import (
	"encoding/json"
	"fmt"
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
	ScanID    int64
	Addresses []string
}

// BuildEdgeFanoutJobs fans an edge-fanout Scan out over the custody-extension
// candidates, in chunks of EdgeFanoutAddressesPerJob. It produces no job for an empty
// candidate set — an instance with no custody extension has nothing to measure, and an
// aperture over an empty scope is a legible state, not an error (CONTEXT.md `Scan`).
//
// Addresses are rendered in their netip form, so the scope a job carries and the
// address a resulting observation names are the same spelling and the recording-side
// scope gate cannot reject a legitimate row over a rendering.
func BuildEdgeFanoutJobs(scanID int64, candidates []netip.Addr) []EdgeFanoutJob {
	if len(candidates) == 0 {
		return nil
	}
	var jobs []EdgeFanoutJob
	for start := 0; start < len(candidates); start += EdgeFanoutAddressesPerJob {
		end := min(start+EdgeFanoutAddressesPerJob, len(candidates))
		addrs := make([]string, 0, end-start)
		for _, a := range candidates[start:end] {
			addrs = append(addrs, a.Unmap().String())
		}
		jobs = append(jobs, EdgeFanoutJob{ScanID: scanID, Addresses: addrs})
	}
	return jobs
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

// AttemptedScope is the by-content record of what the job set out to measure — the
// candidate addresses. It is the Batch's recorded scope on both limbs: a completed
// Batch records what it covered, and a dead-lettered one records the same attempt with
// no observations behind it.
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
