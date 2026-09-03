// Package wire is the one shared definition of the job spec and NDJSON observation
// types web, worker and the prober exchange. A schema mismatch here must fail loudly
// rather than misparse a field into false exposure drift (ADR-0001).
package wire

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	MaxProberStdout = 64 << 20 // 64 MiB, matching crtsh's maxCTBody cap on an untrusted read

	MaxObservationLine = 1 << 20 // 1 MiB, replacing bufio.Scanner's implicit 64 KiB default

	MaxObservations = 1 << 20
)

var ErrProberOutputTooLarge = errors.New("wire: prober output exceeds cap")

type LimitedBuffer struct {
	buf   bytes.Buffer
	limit int
	over  bool
}

func NewLimitedBuffer(limit int) *LimitedBuffer { return &LimitedBuffer{limit: limit} }

func (b *LimitedBuffer) Write(p []byte) (int, error) {
	// A compromised prober could otherwise stream unbounded output into the worker (#772).
	if b.over || b.buf.Len()+len(p) > b.limit {
		b.over = true
		// Fail closed: the job errors and retries rather than accepting partial output.
		return 0, ErrProberOutputTooLarge
	}
	return b.buf.Write(p)
}

func (b *LimitedBuffer) Bytes() []byte { return b.buf.Bytes() }

// A tripped buffer holds only the head bytes retained before the trip, never the tail (#865).

func (b *LimitedBuffer) Overflowed() bool { return b.over }

// The scope stays opaque here, so wire never needs to know every measurement kind.

type JobSpec struct {
	Batch string          `json:"batch"`
	Kind  string          `json:"kind"`
	Scope json.RawMessage `json:"scope,omitempty"`
}

// The facet fields are additive: a kind that predates them leaves them empty.

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
	CertMaterial  *CertMaterial   `json:"cert_material,omitempty"`
}

// ADR-0027 fences a facet value to the chain's fingerprints, so this rides beside it (spec §5.3).

type CertMaterial struct {
	Fingerprint string `json:"fingerprint"` // sha256:<hex>, the value the facet's chain[0] carries
	DER         []byte `json:"der"`
	SCTs        []byte `json:"scts,omitempty"`

	// An embedded SCT's leaf hash carries issuer_key_hash = SHA-256(this) (RFC 6962 §3.2, #878).

	IssuerSPKI []byte `json:"issuer_spki,omitempty"`
}

// Embedded SCTs are not here: they ride inside the leaf DER (#878).

type SCTCapture struct {
	TLSExt [][]byte `json:"tls_ext,omitempty"`
	OCSP   []byte   `json:"ocsp,omitempty"`
}

func EncodeSCTCapture(c SCTCapture) []byte {
	// An empty capture stores a NULL scts column, never an empty-object blob.
	if len(c.TLSExt) == 0 && len(c.OCSP) == 0 {
		return nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		panic("wire: marshal SCT capture: " + err.Error())
	}
	return b
}

func DecodeSCTCapture(b []byte) (SCTCapture, error) {
	var c SCTCapture
	if len(b) == 0 {
		return c, nil
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return SCTCapture{}, fmt.Errorf("wire: decode SCT capture: %w", err)
	}
	return c, nil
}

func DecodeJobSpec(r io.Reader) (JobSpec, error) {
	var spec JobSpec
	if err := json.NewDecoder(r).Decode(&spec); err != nil {
		return JobSpec{}, fmt.Errorf("wire: decode job spec: %w", err)
	}
	return spec, nil
}

func EncodeJobSpec(w io.Writer, spec JobSpec) error {
	if err := json.NewEncoder(w).Encode(spec); err != nil {
		return fmt.Errorf("wire: encode job spec: %w", err)
	}
	return nil
}

func EncodeObservation(w io.Writer, obs Observation) error {
	if err := json.NewEncoder(w).Encode(obs); err != nil {
		return fmt.Errorf("wire: encode observation: %w", err)
	}
	return nil
}

type ObservationScanner struct {
	scanner *bufio.Scanner
	obs     Observation
	count   int
	err     error
}

func NewObservationScanner(r io.Reader) *ObservationScanner {
	// The total-byte bound belongs upstream on the sink, where the stream accumulates (#772).
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), MaxObservationLine)
	return &ObservationScanner{scanner: sc}
}

func (s *ObservationScanner) Next() bool {
	if s.count >= MaxObservations {
		s.err = ErrProberOutputTooLarge
		return false
	}
	if !s.scanner.Scan() {
		s.err = s.scanner.Err()
		return false
	}
	s.count++
	// json.Unmarshal overwrites only present fields, so a reused struct would inherit the last line.
	s.obs = Observation{}
	if err := json.Unmarshal(s.scanner.Bytes(), &s.obs); err != nil {
		s.err = fmt.Errorf("wire: decode observation: %w", err)
		return false
	}
	return true
}

func (s *ObservationScanner) Observation() Observation {
	return s.obs
}

func (s *ObservationScanner) Err() error {
	return s.err
}
