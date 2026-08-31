// Package wire is the one shared definition of the job spec and NDJSON
// observation types that web, worker and the prober binary pass to each
// other. Nothing outside this package encodes or decodes them — a schema
// mismatch here must fail loudly rather than silently misparse a field into
// false exposure drift (ADR-0001).
package wire

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// A prober's stdout is untrusted input: the threat model is a compromised prober
// host, or a MITM on the vantage channel before the host key is pinned on first
// connect (#772). Left unbounded, such a prober can stream arbitrary volume into
// the worker's in-memory sink and OOM the process that drains the queue for every
// tenant. These ceilings fail closed — the job errors (and so retries / dead-letters)
// rather than truncating into a partial-but-accepted result — mirroring the repo's
// other capped untrusted reads (crtsh's maxCTBody = 64<<20, caida's 32<<20 LimitReader
// + sc.Buffer). The values are generous for any legitimate job while staying finite.
const (
	// MaxProberStdout caps the TOTAL bytes a prober may write to stdout for one
	// job before decoding — the gap the per-line scanner default never closed.
	// 64 MiB matches maxCTBody: comfortably larger than any real NDJSON
	// observation stream, small enough that one hostile job cannot exhaust RAM.
	MaxProberStdout = 64 << 20 // 64 MiB total

	// MaxObservationLine caps a single NDJSON line, making bufio.Scanner's
	// implicit 64 KiB MaxScanTokenSize explicit and tunable. An over-long line
	// surfaces bufio.ErrTooLong via Err() rather than over-allocating.
	MaxObservationLine = 1 << 20 // 1 MiB per line

	// MaxObservations caps how many observation lines one job may yield. Even
	// under MaxProberStdout a flood of tiny lines would grow the decoded slice
	// without bound; this bounds the entry COUNT too.
	MaxObservations = 1 << 20
)

// ErrProberOutputTooLarge is the failure a capped prober-stdout sink or scanner
// returns once a prober exceeds a ceiling above. It is a decode failure, not a
// truncation: the caller errors the job rather than accepting partial output.
var ErrProberOutputTooLarge = errors.New("wire: prober output exceeds cap")

// LimitedBuffer is the fail-closed sink for a prober's stdout. It accumulates up
// to limit bytes; the write that would exceed the ceiling returns
// ErrProberOutputTooLarge and retains nothing further, so the buffer never holds
// more than limit bytes. Passed as the io.Writer stdout sink to conn.Run /
// cmd.Stdout, it bounds memory during the streaming copy — before decoding — so a
// compromised prober cannot OOM the worker by streaming unbounded output (#772).
type LimitedBuffer struct {
	buf   bytes.Buffer
	limit int
	over  bool
}

// NewLimitedBuffer returns a LimitedBuffer that fails closed past limit bytes.
func NewLimitedBuffer(limit int) *LimitedBuffer { return &LimitedBuffer{limit: limit} }

// Write implements io.Writer, failing closed once the ceiling would be exceeded.
func (b *LimitedBuffer) Write(p []byte) (int, error) {
	if b.over || b.buf.Len()+len(p) > b.limit {
		b.over = true
		return 0, ErrProberOutputTooLarge
	}
	return b.buf.Write(p)
}

// Bytes returns the accumulated output, safe to scan once the write completed
// without ErrProberOutputTooLarge.
func (b *LimitedBuffer) Bytes() []byte { return b.buf.Bytes() }

// Overflowed reports whether a write tripped the ceiling. A transcript capture
// (#865) reads this to mark the stored stdout stream memory-guard-tripped rather
// than head+tail truncated (raw-job-output spec §3.2): once it is true the buffer
// holds only the head bytes retained before the trip, never the true tail.
func (b *LimitedBuffer) Overflowed() bool { return b.over }

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
	// CertMaterial rides a `certificate` observation line as additive, optional
	// enrichment — the raw CT inputs a handshake carried (leaf DER + out-of-cert SCTs).
	// It is NEVER part of the facet value in Data: ADR-0027 fences that value to the
	// presented chain's fingerprints, so the material lands in a separate side store
	// (certificate_material) keyed by fingerprint. Present only on a presented handshake
	// whose leaf DER we captured; every other line leaves it nil.
	CertMaterial *CertMaterial `json:"cert_material,omitempty"`
}

// CertMaterial is the raw CT-input capture for one leaf certificate, carried on the
// certificate observation line and landed by the worker in the certificate_material
// side store. The observation still records only the fingerprint in its facet value, so
// ADR-0027's fence stays closed (spec §5.3).
type CertMaterial struct {
	// Fingerprint is the leaf DER's `sha256:<hex>` — the side store's PK and the same
	// value the facet's chain[0] carries, so a chain fingerprint joins here.
	Fingerprint string `json:"fingerprint"`
	// DER is the leaf certificate DER bytes. Embedded SCTs ride INSIDE it.
	DER []byte `json:"der"`
	// SCTs is the out-of-cert SCT material, already serialized by EncodeSCTCapture; nil
	// when the handshake carried none.
	SCTs []byte `json:"scts,omitempty"`
	// IssuerSPKI is the issuer certificate's SubjectPublicKeyInfo DER (chain[1]); nil when the
	// handshake presented no issuer. Verification of an embedded SCT hashes the precertificate,
	// whose leaf hash carries issuer_key_hash = SHA-256(this) (RFC 6962 §3.2, #878). It rides
	// beside the leaf, never inside the facet value; omitempty keeps a golden row with no
	// material byte-identical.
	IssuerSPKI []byte `json:"issuer_spki,omitempty"`
}

// SCTCapture is the out-of-cert SCT material captured at a TLS handshake, before it is
// serialized into the certificate_material.scts column. Embedded SCTs are NOT here —
// they ride inside the leaf DER. Verification (#878) decodes this to check the leaf
// against CT.
type SCTCapture struct {
	// TLSExt is the SCTs delivered in the TLS handshake extension (crypto/tls
	// ConnectionState.SignedCertificateTimestamps), each a serialized SCT.
	TLSExt [][]byte `json:"tls_ext,omitempty"`
	// OCSP is the raw stapled OCSP response (ConnectionState.OCSPResponse), which may
	// carry SCTs in an extension; stored verbatim and parsed by the consumer.
	OCSP []byte `json:"ocsp,omitempty"`
}

// EncodeSCTCapture serializes c for the certificate_material.scts column. It returns nil
// when c carries no material, so an empty capture stores a NULL column rather than an
// empty-object blob.
func EncodeSCTCapture(c SCTCapture) []byte {
	if len(c.TLSExt) == 0 && len(c.OCSP) == 0 {
		return nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		// c holds only [][]byte and []byte, which json.Marshal cannot fail on.
		panic("wire: marshal SCT capture: " + err.Error())
	}
	return b
}

// DecodeSCTCapture reverses EncodeSCTCapture. A nil or empty blob decodes to a zero
// SCTCapture — no material — the state a NULL scts column reads back as.
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
	count   int
	err     error
}

// NewObservationScanner wraps r for line-by-line observation decoding. The
// per-line buffer cap is made explicit (MaxObservationLine) rather than left to
// bufio.Scanner's implicit 64 KiB default, and the line COUNT is bounded in Next
// (MaxObservations) — the total-byte bound belongs on the sink upstream
// (LimitedBuffer), which is where the untrusted stream is actually accumulated.
func NewObservationScanner(r io.Reader) *ObservationScanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), MaxObservationLine)
	return &ObservationScanner{scanner: sc}
}

// Next advances to the next observation, returning false at EOF or on the
// first decode error (retrievable via Err).
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
