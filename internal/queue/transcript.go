package queue

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/transcript"
	"github.com/winniel123/verge-asm/internal/wire"
)

// Per-stream store caps for a captured transcript (raw-job-output spec §3.2),
// distinct from the 64 MiB wire.MaxProberStdout memory guard. A stream past its cap
// is head+tail truncated (headTail) so the tail — the crash/exit context an operator
// most wants — survives the cut.
const (
	capTranscriptStdout    = 4 << 20   // 4 MiB
	capTranscriptStderr    = 256 << 10 // 256 KiB
	capTranscriptSentScope = 64 << 10  // 64 KiB
)

// buildTranscriptParams turns a captured wire.Transcript into the row the worker
// writes inside a job's terminal tx (spec §2.4). The worker stamps jobID (the
// queue_job grain, one per attempt) and capturedAt (w.now()); the producer supplies
// the kind, duration, streams and typed outcome. key seals every captured stream at
// rest (spec §5.3).
//
// The prober variant captures locally (#865); the zone variant captures on its
// completed path (#869); the ct variant captures the crt.sh HTTP exchange on every
// terminal path (#870). An unexpected variant is a wiring bug and errors loudly
// rather than writing a mislabelled row.
func buildTranscriptParams(jobID int64, capturedAt time.Time, t wire.Transcript, key []byte) (db.InsertTranscriptParams, error) {
	switch v := t.(type) {
	case wire.ProberTranscript:
		return buildProberParams(jobID, capturedAt, v, key)
	case wire.ZoneTranscript:
		return buildZoneParams(jobID, capturedAt, v, key)
	case wire.CTTranscript:
		return buildCTParams(jobID, capturedAt, v, key)
	default:
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: transcript variant %T not captured yet", t)
	}
}

// buildZoneParams builds the row for a ZoneTranscript. Zone sends nothing to a
// prober, so it reuses the role columns (migration 23700): the skipped records
// land in the stdout column (the primary panel of the §6 view), one per line,
// head+tail truncated to the stdout cap and sealed at rest. The restated count
// rides the typed outcome object — the zone variant's role-column analog of the
// prober's exit code. The stderr and sent-scope columns stay NULL: zone carries
// neither, and the zone-file bytes are never duplicated here (§1.3).
func buildZoneParams(jobID int64, capturedAt time.Time, v wire.ZoneTranscript, key []byte) (db.InsertTranscriptParams, error) {
	skips := []byte(strings.Join(v.Skipped, "\n"))
	stdout, dropped := headTail(skips, capTranscriptStdout)

	markers := map[string]any{}
	if dropped > 0 {
		markers["stdout"] = map[string]any{"kept": len(stdout), "dropped": dropped}
	}
	truncation, err := json.Marshal(markers) // {} when nothing truncated
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: marshal truncation marker: %w", err)
	}

	outcome, err := encodeZoneOutcome(v.Outcome, v.Restated)
	if err != nil {
		return db.InsertTranscriptParams{}, err
	}

	// A restate with no skips seals a captured-but-empty stdout (non-NULL) — "we
	// skipped nothing", distinct from a stderr the variant does not carry (NULL).
	sealedStdout, err := transcript.Seal(key, nonNil(stdout))
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: seal zone skips: %w", err)
	}

	return db.InsertTranscriptParams{
		QueueJobID: jobID,
		Kind:       v.Kind,
		DurationNs: v.Duration.Nanoseconds(),
		CapturedAt: tstz(capturedAt),
		Variant:    "zone",
		Outcome:    outcome,
		Stdout:     sealedStdout,
		Stderr:     nil, // zone carries no stderr — NULL, not a captured-empty stream
		SentScope:  nil, // zone sends nothing — NULL
		Truncation: truncation,
	}, nil
}

// encodeZoneOutcome encodes a zone restate's typed outcome as the JSONB object the
// transcript row stores: {"kind":"parsed","restated":N} / {"kind":"decode-error",
// "restated":N,"text":T} (spec §1.2, §1.3). The restated count rides the object so
// the §6 read handler reads it without a separate column, mirroring how the prober
// outcome carries its exit code. The union is closed, so an unknown member is a bug.
func encodeZoneOutcome(o wire.ZoneOutcome, restated int) ([]byte, error) {
	switch v := o.(type) {
	case wire.ZoneParsed:
		return json.Marshal(map[string]any{"kind": "parsed", "restated": restated})
	case wire.ZoneDecodeError:
		return json.Marshal(map[string]any{"kind": "decode-error", "restated": restated, "text": v.Text})
	default:
		return nil, fmt.Errorf("queue: unknown zone outcome %T", o)
	}
}

// buildCTParams builds the row for a CTTranscript. The crt.sh producer sends an HTTP
// request, so the debug artifact is the exchange: the verbatim response body lands in
// the stdout role column (migration 23700) — the primary panel of the §6 view —
// head+tail truncated to the stdout cap and sealed at rest. The request URL and the
// typed outcome (HTTP status or transport-error text) ride the outcome object, the CT
// variant's role-column analog of the prober's exit code. The stderr column stays NULL
// (HTTP has no stderr; the transport-error text is its analog, on the outcome) and the
// sent-scope column stays NULL (a GET sends no request body).
func buildCTParams(jobID int64, capturedAt time.Time, v wire.CTTranscript, key []byte) (db.InsertTranscriptParams, error) {
	stdout, dropped := headTail(v.ResponseBody, capTranscriptStdout)

	markers := map[string]any{}
	if dropped > 0 {
		markers["stdout"] = map[string]any{"kept": len(stdout), "dropped": dropped}
	}
	truncation, err := json.Marshal(markers) // {} when nothing truncated
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: marshal truncation marker: %w", err)
	}

	outcome, err := encodeCTOutcome(v.Outcome, v.RequestURL)
	if err != nil {
		return db.InsertTranscriptParams{}, err
	}

	// A fetch with no body (a transport error or a bodyless status) seals a
	// captured-but-empty response (non-NULL) — "we made the exchange and got no body",
	// distinct from a stderr the variant does not carry (NULL).
	sealedStdout, err := transcript.Seal(key, nonNil(stdout))
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: seal ct response body: %w", err)
	}

	return db.InsertTranscriptParams{
		QueueJobID: jobID,
		Kind:       v.Kind,
		DurationNs: v.Duration.Nanoseconds(),
		CapturedAt: tstz(capturedAt),
		Variant:    "ct",
		Outcome:    outcome,
		Stdout:     sealedStdout,
		Stderr:     nil, // HTTP carries no stderr — NULL; the transport-error text rides the outcome
		SentScope:  nil, // a GET sends no request body — NULL
		Truncation: truncation,
	}, nil
}

// encodeCTOutcome encodes a CT exchange's typed outcome as the JSONB object the
// transcript row stores: {"kind":"http","status":N,"request_url":U} / {"kind":
// "transport-error","text":T,"request_url":U} / {"kind":"context-cancelled",
// "request_url":U} (spec §1.2). The request URL rides every arm so the §6 read handler
// reads it without a separate column, mirroring how the zone outcome carries its
// restated count. The URL is not secret (crt.sh and Cert Spotter carry the credential
// in a header, never the URL), so it stays plaintext on the outcome, not in the sealed
// body column. The union is closed, so an unknown member is a programming error.
func encodeCTOutcome(o wire.CTOutcome, requestURL string) ([]byte, error) {
	switch v := o.(type) {
	case wire.CTHTTP:
		return json.Marshal(map[string]any{"kind": "http", "status": v.Status, "request_url": requestURL})
	case wire.CTTransportError:
		return json.Marshal(map[string]any{"kind": "transport-error", "text": v.Text, "request_url": requestURL})
	case wire.CTContextCancelled:
		return json.Marshal(map[string]any{"kind": "context-cancelled", "request_url": requestURL})
	default:
		return nil, fmt.Errorf("queue: unknown ct outcome %T", o)
	}
}

// buildProberParams builds the row for a ProberTranscript: each verbatim stream is
// head+tail truncated to its cap, marked, then sealed; the typed outcome is encoded
// as the {"kind": ...} JSONB object. All three streams are captured, so an empty one
// seals to real ciphertext (a captured-but-empty stream, not a NULL absence).
func buildProberParams(jobID int64, capturedAt time.Time, v wire.ProberTranscript, key []byte) (db.InsertTranscriptParams, error) {
	markers := map[string]any{}

	stdout, stdoutDropped := headTail(v.Stdout, capTranscriptStdout)
	switch {
	case v.StdoutOverflow:
		// The 64 MiB guard tripped: stdout holds only the head bytes retained
		// before the trip, and an unknown further amount was never captured. Mark
		// it distinctly so the operator knows the true tail is gone (§3.2).
		markers["stdout"] = map[string]any{"kept": len(stdout), "dropped": stdoutDropped, "memory_guard_tripped": true}
	case stdoutDropped > 0:
		markers["stdout"] = map[string]any{"kept": len(stdout), "dropped": stdoutDropped}
	}

	stderr, stderrDropped := headTail(v.Stderr, capTranscriptStderr)
	if stderrDropped > 0 {
		markers["stderr"] = map[string]any{"kept": len(stderr), "dropped": stderrDropped}
	}

	sentScope, sentDropped := headTail(v.SentScope, capTranscriptSentScope)
	if sentDropped > 0 {
		markers["sent_scope"] = map[string]any{"kept": len(sentScope), "dropped": sentDropped}
	}

	truncation, err := json.Marshal(markers) // {} when nothing truncated
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: marshal truncation marker: %w", err)
	}

	outcome, err := encodeProberOutcome(v.Outcome)
	if err != nil {
		return db.InsertTranscriptParams{}, err
	}

	sealedStdout, err := transcript.Seal(key, nonNil(stdout))
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: seal stdout: %w", err)
	}
	sealedStderr, err := transcript.Seal(key, nonNil(stderr))
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: seal stderr: %w", err)
	}
	sealedSent, err := transcript.Seal(key, nonNil(sentScope))
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: seal sent scope: %w", err)
	}

	return db.InsertTranscriptParams{
		QueueJobID: jobID,
		Kind:       v.Kind,
		DurationNs: v.Duration.Nanoseconds(),
		CapturedAt: tstz(capturedAt),
		Variant:    "prober",
		Outcome:    outcome,
		Stdout:     sealedStdout,
		Stderr:     sealedStderr,
		SentScope:  sealedSent,
		Truncation: truncation,
	}, nil
}

// encodeProberOutcome encodes a prober's typed outcome as the JSONB object the
// transcript row stores: {"kind": "exited", "code": N} / {"kind": "signalled",
// "signal": S} / {"kind": "context-cancelled"} (spec §1.2). A ctx-killed prober is
// context-cancelled, never a fake exited(0). The union is closed, so an unknown
// member is a programming error.
func encodeProberOutcome(o wire.ProberOutcome) ([]byte, error) {
	switch v := o.(type) {
	case wire.ProberExited:
		return json.Marshal(map[string]any{"kind": "exited", "code": v.Code})
	case wire.ProberSignalled:
		return json.Marshal(map[string]any{"kind": "signalled", "signal": v.Signal})
	case wire.ProberContextCancelled:
		return json.Marshal(map[string]any{"kind": "context-cancelled"})
	default:
		return nil, fmt.Errorf("queue: unknown prober outcome %T", o)
	}
}

// headTail truncates b to at most limit bytes, keeping the head and tail halves and
// dropping the middle (spec §3.2). A stream within limit is returned whole with a
// zero drop count. Head+tail (not head-only) because when a job overflows the cap the
// tail holds the crash/exit context; a head-only cut usually loses exactly that.
func headTail(b []byte, limit int) (out []byte, dropped int) {
	if len(b) <= limit {
		return b, 0
	}
	head := limit / 2
	tail := limit - head
	out = make([]byte, 0, limit)
	out = append(out, b[:head]...)
	out = append(out, b[len(b)-tail:]...)
	return out, len(b) - limit
}

// nonNil replaces a nil slice with an empty non-nil one, so a captured-but-empty
// stream seals to real ciphertext (a captured empty stream) instead of NULL (a
// stream this variant does not carry). The prober variant captures all three
// streams, so each must persist as captured even when empty (migration 23700).
func nonNil(b []byte) []byte {
	if b == nil {
		return []byte{}
	}
	return b
}
