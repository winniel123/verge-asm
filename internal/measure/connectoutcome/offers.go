// Package connectoutcome decides the `reachability` and `certificate` facets for a `Service`
// by a full TCP connect and never SYN, so the leaf runs non-root with no added capability
// (v1 spec §3.3). Silence on a connection-oriented transport decides; UDP is refused (ADR-0083).
package connectoutcome

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Moves only on an output-affecting change, gated by this leaf's own golden corpus (ADR-0008).

const Version = "connect-outcome/v2"

const Kind = "connect-outcome"

// Recorded on the Batch by content, so what governed a probe is never a library default (ADR-0025).

type SafetyProfile struct {
	Technique     string `json:"technique"`
	HostDiscovery string `json:"host_discovery"`

	PerHostConnPerSec    int `json:"per_host_conn_per_sec"`
	PerHostConcurrency   int `json:"per_host_concurrency"`
	ConnectTimeoutMillis int `json:"connect_timeout_millis"`
	Retries              int `json:"retries"`

	PerVantagePacketsPerSec int  `json:"per_vantage_packets_per_sec"`
	RoundRobinByHost        bool `json:"round_robin_by_host"`

	AdaptiveBackoff BackoffPolicy `json:"adaptive_backoff"`
}

// The always-false deadline flag is recorded on the Batch, not merely held in code (ADR-0021).

type BackoffPolicy struct {
	HalveOnTimeout  bool `json:"halve_on_timeout"`
	HalveOnRSTSpike bool `json:"halve_on_rst_spike"`
	HalveOn429      bool `json:"halve_on_429"`
	HalveOn503      bool `json:"halve_on_503"`
	TouchesDeadline bool `json:"touches_deadline"`
}

func DefaultProfile() SafetyProfile {
	return SafetyProfile{
		Technique: "tcp-connect",
		// Targets are seeded and never swept for liveness, so no port answering is still an observation.
		HostDiscovery: "skipped",
		// The rate's unit is intra-pair, not intra-job, and holds at any worker count (ADR-0137, #1106).
		PerHostConnPerSec:    50,
		PerHostConcurrency:   20,
		ConnectTimeoutMillis: 3000,
		Retries:              2,
		// Enforced per Vantage, so a target inside N Vantages receives N times the rate (ADR-0137).
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
