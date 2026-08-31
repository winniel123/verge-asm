package queue

import (
	"encoding/json"
	"fmt"
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
// Only the prober variant captures locally (#865). The ct and zone variants land
// with their own tickets (#870, #869); until then their producers return an absent
// transcript, so this switch never sees them. An unexpected variant is a wiring bug
// and errors loudly rather than writing a mislabelled row.
func buildTranscriptParams(jobID int64, capturedAt time.Time, t wire.Transcript, key []byte) (db.InsertTranscriptParams, error) {
	switch v := t.(type) {
	case wire.ProberTranscript:
		return buildProberParams(jobID, capturedAt, v, key)
	default:
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: transcript variant %T not captured yet", t)
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
