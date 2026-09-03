// Package connectoutcome is the `connect-outcome` Derivation leaf inside the
// shared measurement binary (v1 spec §3.3). It decides the `reachability` facet
// for a `Service` — an `(Address, port, transport)` triple — by a TCP
// **connect** (never SYN): the leaf opens a full connection and closes it, so it
// runs non-root with `cap_drop: [ALL]` and no added capabilities. It is the
// leaf the daily `hot` Scan dispatches over `verge-core`'s TCP pairs.
//
// The verdict is the closed pair `reached │ not-reached` (CONTEXT.md `Reach`):
// on a connection-oriented transport, silence still decides, so a connect that
// is refused (RST) or times out after its retries is `not-reached`, and only a
// completed connection is `reached`. UDP is off — `connect-outcome` cannot
// decide an honest UDP value (ADR-0083) — so the `hot` Scan records UDP pairs in
// scope and never probes them.
//
// The safety profile (§3.3) that paces the probing lives in `safety.go` and is
// deliberately OUTSIDE the verdict: the adaptive back-off changes the rate and
// never the deadline, and it can neither manufacture nor suppress a reachability
// value (ADR-0021). The verdict here is the golden-corpus-gated part; the pacing
// is not.
package connectoutcome

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Version is the leaf's Derivation version (ADR-0008/ADR-0021). It moves on an
// output-affecting change and only on one, gated bidirectionally by this leaf's
// own golden corpus (§4.4) — separately from the other leaves, so a break names
// its leaf.
const Version = "connect-outcome/v2"

const Kind = "connect-outcome"

// SafetyProfile is the leaf's declared-parameter set: the exact §3.3 safety
// table, recorded on the Batch by content (ADR-0025) so what governed the probe
// is legible and never a library default. None of it is operator-configurable.
//
// The rate/concurrency/ceiling knobs are consumed by the limiter in safety.go;
// they are declared here so a change to any of them is a declared-parameter
// change that moves the leaf's params digest and forces a Version bump.
//
// Scope: every ceiling below is chosen for what one TARGET receives and is
// enforced over what one `Vantage` emits. ADR-0137 discloses that gap rather
// than closing it — a target inside N declared Vantages receives up to N times
// the declared rate, and nothing gates the vantage count.
//
// The per-target half needs no cross-process coordination: `hot` fans out one
// job per `(Vantage, Address)` pair, so two concurrent jobs of one Dispatch
// never share a target from one position, at any worker count. Two overlapping
// Dispatches would, and the cadence-lag gate closes that (#1114). The per-vantage
// half is NOT enforced across processes — the limiter runs inside ONE prober
// process and the worker execs a fresh prober per job (`ExecProber.Probe`), and
// a prober on a remote vantage runs over SSH with no route to Postgres, so no
// reservation can live where the packets leave. The running guide therefore
// still forbids `--scale worker=N` (#1092, #1105, ADR-0137).
type SafetyProfile struct {
	// Technique is fixed: TCP connect, never SYN. Recorded so the Batch states it
	// rather than leaving it to be inferred from the absence of raw sockets.
	Technique string `json:"technique"`
	// HostDiscovery is skipped — the `-Pn` posture. Targets are seeded, not swept
	// for liveness, so "no ports responded" is a diffable observation about a
	// real subject rather than a reason to skip the host.
	HostDiscovery string `json:"host_discovery"`

	// PerHostConnPerSec is the per-host connection rate ceiling — ≤ 50 conn/s from
	// one `Vantage` (see the scope note above). ADR-0005 called this limit
	// intra-job and needing no coordination; the conclusion holds and the unit is
	// wrong. The `hot` and `cold` Scans each build one job per `(Vantage, Address)`
	// pair, so the unit is intra-pair and holds at any worker count within one
	// Dispatch (ADR-0137). #1106 holds the amendment. A host inside two Vantages
	// sits in two jobs and receives twice the rate — the disclosed gap above, not
	// a coordination defect.
	PerHostConnPerSec    int `json:"per_host_conn_per_sec"`
	PerHostConcurrency   int `json:"per_host_concurrency"`
	ConnectTimeoutMillis int `json:"connect_timeout_millis"`
	// Retries is the number of re-attempts after a timeout/error before the
	// verdict is decided — 2. A refusal (RST) is an answer and is never retried.
	Retries int `json:"retries"`

	// PerVantagePacketsPerSec is the 200 pkt/s aggregate ceiling across every
	// target one `Vantage` probes, round-robin by host so adding targets never
	// multiplies load. It is not estate-wide — see the scope note above.
	//
	// It also does not bind under the shipped partitioning. A `hot` job carries
	// one Address (ADR-0005, ADR-0127), so the pacer's per-host map holds one
	// entry. 50 conn/s is a 20 ms interval and 200 pkt/s is a 5 ms one. The
	// per-host interval is therefore always the later, and Pacer.Next always
	// returns the per-host instant. The value is a recorded commitment, not a
	// limit that has governed a connect (#1092).
	PerVantagePacketsPerSec int `json:"per_vantage_packets_per_sec"`
	// RoundRobinByHost records that scheduling cycles hosts, never ports — the
	// canonical way to avoid a dense burst against one destination.
	RoundRobinByHost bool `json:"round_robin_by_host"`

	// AdaptiveBackoff records the back-off policy: halve the rate on a
	// timeout/RST-spike/429/503 signal, never touching the deadline (ADR-0021).
	AdaptiveBackoff BackoffPolicy `json:"adaptive_backoff"`
}

// BackoffPolicy is the adaptive back-off's declared shape. It halves the rate on
// a stress signal and never touches the deadline — the deadline is the leaf's,
// the rate is the limiter's.
type BackoffPolicy struct {
	HalveOnTimeout  bool `json:"halve_on_timeout"`
	HalveOnRSTSpike bool `json:"halve_on_rst_spike"`
	HalveOn429      bool `json:"halve_on_429"`
	HalveOn503      bool `json:"halve_on_503"`
	TouchesDeadline bool `json:"touches_deadline"` // always false — the invariant, recorded
}

// DefaultProfile is the v1 shipped safety profile — the §3.3 table exactly. It
// is authored here, not defaulted by any library, so the job spec carries it to
// the leaf and the Batch records exactly what governed the probe.
func DefaultProfile() SafetyProfile {
	return SafetyProfile{
		Technique:               "tcp-connect",
		HostDiscovery:           "skipped", // -Pn
		PerHostConnPerSec:       50,
		PerHostConcurrency:      20,
		ConnectTimeoutMillis:    3000,
		Retries:                 2,
		PerVantagePacketsPerSec: 200,
		RoundRobinByHost:        true,
		AdaptiveBackoff: BackoffPolicy{
			HalveOnTimeout:  true,
			HalveOnRSTSpike: true,
			HalveOn429:      true,
			HalveOn503:      true,
			TouchesDeadline: false,
		},
	}
}

func (p SafetyProfile) Digest() string {
	b, err := json.Marshal(p)
	if err != nil {
		panic("connectoutcome: marshal safety profile: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
