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

const (
	capTranscriptStdout    = 4 << 20
	capTranscriptStderr    = 256 << 10
	capTranscriptSentScope = 64 << 10
)

func buildTranscriptParams(jobID int64, capturedAt time.Time, t wire.Transcript, key []byte) (db.InsertTranscriptParams, error) {
	// A closed union errors on an unknown member, never a mislabelled row (raw-job-output §1.2).
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

func buildZoneParams(jobID int64, capturedAt time.Time, v wire.ZoneTranscript, key []byte) (db.InsertTranscriptParams, error) {
	// Skips ride the stdout column because that is the operator's primary panel (raw-job-output §6.3).
	skips := []byte(strings.Join(v.Skipped, "\n"))
	// The operator's zone-file row already holds the bytes, so none are copied (raw-job-output §1.3).
	stdout, dropped := headTail(skips, capTranscriptStdout)

	markers := map[string]any{}
	if dropped > 0 {
		markers["stdout"] = map[string]any{"kept": len(stdout), "dropped": dropped}
	}
	truncation, err := json.Marshal(markers)
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: marshal truncation marker: %w", err)
	}

	outcome, err := encodeZoneOutcome(v.Outcome, v.Restated)
	if err != nil {
		return db.InsertTranscriptParams{}, err
	}

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
		Stderr:     nil,
		SentScope:  nil,
		Truncation: truncation,
	}, nil
}

func encodeZoneOutcome(o wire.ZoneOutcome, restated int) ([]byte, error) {
	// A variant's own scalar rides its outcome object, so the read surface needs no extra column.
	switch v := o.(type) {
	case wire.ZoneParsed:
		return json.Marshal(map[string]any{"kind": "parsed", "restated": restated})
	case wire.ZoneDecodeError:
		return json.Marshal(map[string]any{"kind": "decode-error", "restated": restated, "text": v.Text})
	default:
		return nil, fmt.Errorf("queue: unknown zone outcome %T", o)
	}
}

func buildCTParams(jobID int64, capturedAt time.Time, v wire.CTTranscript, key []byte) (db.InsertTranscriptParams, error) {
	stdout, dropped := headTail(v.ResponseBody, capTranscriptStdout)

	markers := map[string]any{}
	if dropped > 0 {
		markers["stdout"] = map[string]any{"kept": len(stdout), "dropped": dropped}
	}
	truncation, err := json.Marshal(markers)
	if err != nil {
		return db.InsertTranscriptParams{}, fmt.Errorf("queue: marshal truncation marker: %w", err)
	}

	outcome, err := encodeCTOutcome(v.Outcome, v.RequestURL)
	if err != nil {
		return db.InsertTranscriptParams{}, err
	}

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
		Stderr:     nil,
		SentScope:  nil,
		Truncation: truncation,
	}, nil
}

func encodeCTOutcome(o wire.CTOutcome, requestURL string) ([]byte, error) {
	// Both CT sources carry their credential in a header and never the URL, so this stays plaintext.
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

func buildProberParams(jobID int64, capturedAt time.Time, v wire.ProberTranscript, key []byte) (db.InsertTranscriptParams, error) {
	markers := map[string]any{}

	stdout, stdoutDropped := headTail(v.Stdout, capTranscriptStdout)
	switch {
	case v.StdoutOverflow:
		// A guard trip lost the true tail, so this is not an ordinary cut (raw-job-output §3.2).
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

	truncation, err := json.Marshal(markers)
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

func headTail(b []byte, limit int) (out []byte, dropped int) {
	// A head-only cut loses the crash context an overflowing job is kept for (raw-job-output §3.2).
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

func nonNil(b []byte) []byte {
	// A captured-empty stream must read apart from an absent one (raw-job-output §1.4).
	if b == nil {
		return []byte{}
	}
	return b
}
