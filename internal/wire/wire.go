// Package wire is the one shared definition of the job spec and NDJSON
// observation types that web, worker and the prober binary pass to each
// other. Nothing outside this package encodes or decodes them — a schema
// mismatch here must fail loudly rather than silently misparse a field into
// false exposure drift (ADR-0001).
package wire

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// JobSpec is written as a single JSON object to the prober's stdin.
type JobSpec struct {
	// Batch identifies the Batch this job belongs to, so the observations
	// it produces can be attributed back to the scope that was attempted.
	Batch string `json:"batch"`
	// Kind names the measurement being requested (e.g. "tcp-connect").
	Kind string `json:"kind"`
	// Scope is the kind-specific payload (addresses, ports, …); left
	// opaque here so this package does not have to know every measurement
	// kind to stay compilable.
	Scope json.RawMessage `json:"scope,omitempty"`
}

// Observation is one NDJSON line the prober writes to stdout per
// measurement it performs.
//
// The Facet/Subject/Discriminator/Vantage fields are additive and optional:
// a leaf that decides a facet value (e.g. resolution-walk) fills them so the
// worker can attribute the line to a (subject, facet, discriminator, vantage)
// timeline without re-deriving it. Kinds that predate them leave them empty.
type Observation struct {
	Batch         string          `json:"batch"`
	Kind          string          `json:"kind"`
	Facet         string          `json:"facet,omitempty"`
	Subject       string          `json:"subject,omitempty"`
	Discriminator string          `json:"discriminator,omitempty"`
	Vantage       string          `json:"vantage,omitempty"`
	Address       string          `json:"address,omitempty"`
	Data          json.RawMessage `json:"data,omitempty"`
	Err           string          `json:"err,omitempty"`
}

// DecodeJobSpec reads a single JobSpec JSON object from r.
func DecodeJobSpec(r io.Reader) (JobSpec, error) {
	var spec JobSpec
	if err := json.NewDecoder(r).Decode(&spec); err != nil {
		return JobSpec{}, fmt.Errorf("wire: decode job spec: %w", err)
	}
	return spec, nil
}

// EncodeJobSpec writes spec to w as a single JSON object.
func EncodeJobSpec(w io.Writer, spec JobSpec) error {
	if err := json.NewEncoder(w).Encode(spec); err != nil {
		return fmt.Errorf("wire: encode job spec: %w", err)
	}
	return nil
}

// EncodeObservation writes obs to w as one NDJSON line.
func EncodeObservation(w io.Writer, obs Observation) error {
	if err := json.NewEncoder(w).Encode(obs); err != nil {
		return fmt.Errorf("wire: encode observation: %w", err)
	}
	return nil
}

// ObservationScanner reads NDJSON observations from a prober's stdout, one
// per call to Next.
type ObservationScanner struct {
	scanner *bufio.Scanner
	obs     Observation
	err     error
}

// NewObservationScanner wraps r for line-by-line observation decoding.
func NewObservationScanner(r io.Reader) *ObservationScanner {
	return &ObservationScanner{scanner: bufio.NewScanner(r)}
}

// Next advances to the next observation, returning false at EOF or on the
// first decode error (retrievable via Err).
func (s *ObservationScanner) Next() bool {
	if !s.scanner.Scan() {
		s.err = s.scanner.Err()
		return false
	}
	// Reset before decoding: json.Unmarshal only overwrites fields present
	// in the line, so a field omitted here (omitempty) would otherwise
	// keep the previous line's value.
	s.obs = Observation{}
	if err := json.Unmarshal(s.scanner.Bytes(), &s.obs); err != nil {
		s.err = fmt.Errorf("wire: decode observation: %w", err)
		return false
	}
	return true
}

// Observation returns the observation decoded by the most recent Next call.
func (s *ObservationScanner) Observation() Observation {
	return s.obs
}

// Err returns the first error encountered while scanning, if any.
func (s *ObservationScanner) Err() error {
	return s.err
}
