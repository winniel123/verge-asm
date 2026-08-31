package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/transcript"
	"github.com/winniel123/verge-asm/internal/wire"
)

// testKey is a fixed 32-byte XChaCha20-Poly1305 key for sealing in tests.
var testKey = bytes.Repeat([]byte{0x42}, 32)

// TestHeadTail checks the truncation math: a stream within the limit is returned
// whole with no drop; a stream over it keeps the head and tail halves and reports the
// exact dropped middle.
func TestHeadTail(t *testing.T) {
	within := []byte("hello")
	out, dropped := headTail(within, 10)
	if !bytes.Equal(out, within) || dropped != 0 {
		t.Fatalf("within limit: got out=%q dropped=%d, want %q 0", out, dropped, within)
	}

	// 10 bytes truncated to 4: head=2, tail=2, middle 6 dropped.
	over := []byte("0123456789")
	out, dropped = headTail(over, 4)
	if want := []byte("0189"); !bytes.Equal(out, want) {
		t.Errorf("over limit: got out=%q, want %q", out, want)
	}
	if dropped != 6 {
		t.Errorf("over limit: got dropped=%d, want 6", dropped)
	}
	if len(out) != 4 {
		t.Errorf("over limit: kept %d bytes, want 4 (the cap)", len(out))
	}
}

// TestEncodeProberOutcome pins the JSONB shape each typed outcome stores. A
// ctx-killed prober is context-cancelled, never a fake exited(0).
func TestEncodeProberOutcome(t *testing.T) {
	cases := []struct {
		name    string
		outcome wire.ProberOutcome
		want    map[string]any
	}{
		{"exited", wire.ProberExited{Code: 2}, map[string]any{"kind": "exited", "code": float64(2)}},
		{"signalled", wire.ProberSignalled{Signal: "killed"}, map[string]any{"kind": "signalled", "signal": "killed"}},
		{"context-cancelled", wire.ProberContextCancelled{}, map[string]any{"kind": "context-cancelled"}},
	}
	for _, c := range cases {
		raw, err := encodeProberOutcome(c.outcome)
		if err != nil {
			t.Fatalf("%s: encode: %v", c.name, err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("%s: unmarshal %s: %v", c.name, raw, err)
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
		for k, v := range c.want {
			if got[k] != v {
				t.Errorf("%s: field %q = %v, want %v", c.name, k, got[k], v)
			}
		}
	}

	if _, err := encodeProberOutcome(nil); err == nil {
		t.Error("nil outcome: want error, got nil")
	}
}

// TestBuildProberParamsRoundTrip checks a captured prober transcript maps to the row
// the worker inserts: the frame is stamped, the streams seal and open back verbatim,
// a captured-but-empty stream stays non-NULL (distinct from an absent one), and an
// untruncated capture carries the empty {} marker.
func TestBuildProberParamsRoundTrip(t *testing.T) {
	capturedAt := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	sent := []byte(`{"batch":"b1","kind":"tcp-connect"}` + "\n")
	stdout := []byte(`{"batch":"b1","kind":"tcp-connect","subject":"a"}` + "\n")

	tr := wire.ProberTranscript{
		TranscriptFrame: wire.TranscriptFrame{Kind: "tcp-connect", Duration: 250 * time.Millisecond},
		SentScope:       sent,
		Stdout:          stdout,
		Stderr:          nil, // captured, but the prober wrote nothing to stderr
		Outcome:         wire.ProberExited{Code: 0},
	}

	params, err := buildTranscriptParams(77, capturedAt, tr, testKey)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if params.QueueJobID != 77 {
		t.Errorf("QueueJobID = %d, want 77 (the failed attempt's row)", params.QueueJobID)
	}
	if params.Kind != "tcp-connect" {
		t.Errorf("Kind = %q, want tcp-connect", params.Kind)
	}
	if params.DurationNs != int64(250*time.Millisecond) {
		t.Errorf("DurationNs = %d, want %d", params.DurationNs, int64(250*time.Millisecond))
	}
	if params.Variant != "prober" {
		t.Errorf("Variant = %q, want prober", params.Variant)
	}
	if !params.CapturedAt.Valid || !params.CapturedAt.Time.Equal(capturedAt) {
		t.Errorf("CapturedAt = %v, want %v", params.CapturedAt, capturedAt)
	}
	if string(params.Truncation) != "{}" {
		t.Errorf("Truncation = %s, want {} (nothing truncated)", params.Truncation)
	}

	// The sealed streams open back to the exact captured bytes.
	gotStdout, err := transcript.Open(testKey, params.Stdout)
	if err != nil {
		t.Fatalf("open stdout: %v", err)
	}
	if !bytes.Equal(gotStdout, stdout) {
		t.Errorf("stdout round-trip = %q, want %q", gotStdout, stdout)
	}
	gotSent, err := transcript.Open(testKey, params.SentScope)
	if err != nil {
		t.Fatalf("open sent scope: %v", err)
	}
	if !bytes.Equal(gotSent, sent) {
		t.Errorf("sent-scope round-trip = %q, want %q (verbatim)", gotSent, sent)
	}

	// A captured-but-empty stderr seals to real ciphertext and opens to a non-nil
	// empty slice — NOT NULL. A NULL column is reserved for a stream a variant does
	// not carry; the prober carries all three.
	if params.Stderr == nil {
		t.Fatal("Stderr sealed to NULL, want ciphertext for a captured-but-empty stream")
	}
	gotStderr, err := transcript.Open(testKey, params.Stderr)
	if err != nil {
		t.Fatalf("open stderr: %v", err)
	}
	if gotStderr == nil || len(gotStderr) != 0 {
		t.Errorf("stderr round-trip = %v, want non-nil empty", gotStderr)
	}
}

// TestBuildProberParamsTruncation checks an over-cap stdout is head+tail truncated to
// its store cap and carries an accurate {kept, dropped} marker.
func TestBuildProberParamsTruncation(t *testing.T) {
	big := bytes.Repeat([]byte("x"), capTranscriptStdout+100)
	tr := wire.ProberTranscript{
		TranscriptFrame: wire.TranscriptFrame{Kind: "tcp-connect"},
		SentScope:       []byte("{}"),
		Stdout:          big,
		Stderr:          []byte{},
		Outcome:         wire.ProberExited{Code: 0},
	}

	params, err := buildProberParams(1, time.Now(), tr, testKey)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	var markers map[string]map[string]any
	if err := json.Unmarshal(params.Truncation, &markers); err != nil {
		t.Fatalf("unmarshal truncation %s: %v", params.Truncation, err)
	}
	m, ok := markers["stdout"]
	if !ok {
		t.Fatalf("no stdout marker in %s", params.Truncation)
	}
	if m["kept"] != float64(capTranscriptStdout) {
		t.Errorf("kept = %v, want %d (the cap)", m["kept"], capTranscriptStdout)
	}
	if m["dropped"] != float64(100) {
		t.Errorf("dropped = %v, want 100", m["dropped"])
	}
	if _, tripped := m["memory_guard_tripped"]; tripped {
		t.Error("plain truncation carries a memory_guard_tripped marker, want none")
	}

	got, err := transcript.Open(testKey, params.Stdout)
	if err != nil {
		t.Fatalf("open stdout: %v", err)
	}
	if len(got) != capTranscriptStdout {
		t.Errorf("stored stdout = %d bytes, want %d (the cap)", len(got), capTranscriptStdout)
	}
}

// TestBuildProberParamsMemoryGuard checks that a stdout that tripped the 64 MiB
// memory guard is marked memory-guard-tripped, distinct from plain head+tail
// truncation, even when the retained head is within the store cap.
func TestBuildProberParamsMemoryGuard(t *testing.T) {
	tr := wire.ProberTranscript{
		TranscriptFrame: wire.TranscriptFrame{Kind: "tcp-connect"},
		SentScope:       []byte("{}"),
		Stdout:          []byte("partial head the guard retained"),
		Stderr:          []byte{},
		Outcome:         wire.ProberExited{Code: 0},
		StdoutOverflow:  true,
	}

	params, err := buildProberParams(1, time.Now(), tr, testKey)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	var markers map[string]map[string]any
	if err := json.Unmarshal(params.Truncation, &markers); err != nil {
		t.Fatalf("unmarshal truncation %s: %v", params.Truncation, err)
	}
	m, ok := markers["stdout"]
	if !ok {
		t.Fatalf("no stdout marker in %s", params.Truncation)
	}
	if tripped, _ := m["memory_guard_tripped"].(bool); !tripped {
		t.Errorf("stdout marker = %v, want memory_guard_tripped:true", m)
	}
}

// TestBuildTranscriptParamsUnknownVariant checks that a variant with no local
// capture yet (ct, zone) errors loudly rather than writing a mislabelled row.
func TestBuildTranscriptParamsUnknownVariant(t *testing.T) {
	ct := wire.CTTranscript{
		TranscriptFrame: wire.TranscriptFrame{Kind: "ct"},
		Outcome:         wire.CTHTTP{Status: 200},
	}
	if _, err := buildTranscriptParams(1, time.Now(), ct, testKey); err == nil {
		t.Error("ct variant: want error (not captured yet), got nil")
	}
}

// TestEncodeZoneOutcome pins the JSONB shape the zone outcome stores: the restated
// count rides the object, and decode-error carries its text.
func TestEncodeZoneOutcome(t *testing.T) {
	parsed, err := encodeZoneOutcome(wire.ZoneParsed{}, 42)
	if err != nil {
		t.Fatalf("encode parsed: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(parsed, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", parsed, err)
	}
	if got["kind"] != "parsed" || got["restated"] != float64(42) {
		t.Errorf("parsed = %v, want {kind:parsed, restated:42}", got)
	}

	de, err := encodeZoneOutcome(wire.ZoneDecodeError{Text: "bad rr"}, 3)
	if err != nil {
		t.Fatalf("encode decode-error: %v", err)
	}
	got = nil
	if err := json.Unmarshal(de, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", de, err)
	}
	if got["kind"] != "decode-error" || got["restated"] != float64(3) || got["text"] != "bad rr" {
		t.Errorf("decode-error = %v, want {kind:decode-error, restated:3, text:bad rr}", got)
	}

	if _, err := encodeZoneOutcome(nil, 0); err == nil {
		t.Error("nil zone outcome: want error, got nil")
	}
}

// TestBuildZoneParams checks a captured zone transcript maps to the row the worker
// inserts: variant is zone, the skipped records seal into the stdout role column and
// open back verbatim, the restated count rides the outcome, and stderr/sent-scope stay
// NULL (streams zone does not carry).
func TestBuildZoneParams(t *testing.T) {
	capturedAt := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	tr := wire.ZoneTranscript{
		TranscriptFrame: wire.TranscriptFrame{Kind: "zone", Duration: 12 * time.Millisecond},
		Restated:        7,
		Skipped:         []string{"weird IN FOO whatever", "empty IN TXT"},
		Outcome:         wire.ZoneParsed{},
	}

	params, err := buildTranscriptParams(88, capturedAt, tr, testKey)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if params.Variant != "zone" {
		t.Errorf("Variant = %q, want zone", params.Variant)
	}
	if params.QueueJobID != 88 || params.Kind != "zone" {
		t.Errorf("QueueJobID/Kind = %d/%q, want 88/zone", params.QueueJobID, params.Kind)
	}

	// The skipped records seal into stdout and open back, one per line, verbatim.
	gotStdout, err := transcript.Open(testKey, params.Stdout)
	if err != nil {
		t.Fatalf("open zone stdout: %v", err)
	}
	if want := "weird IN FOO whatever\nempty IN TXT"; string(gotStdout) != want {
		t.Errorf("zone skips round-trip = %q, want %q", gotStdout, want)
	}

	// The restated count rides the outcome object.
	var outcome map[string]any
	if err := json.Unmarshal(params.Outcome, &outcome); err != nil {
		t.Fatalf("unmarshal outcome %s: %v", params.Outcome, err)
	}
	if outcome["kind"] != "parsed" || outcome["restated"] != float64(7) {
		t.Errorf("outcome = %v, want {kind:parsed, restated:7}", outcome)
	}

	// Zone carries no stderr or sent-scope: those columns stay NULL, distinct from the
	// prober's captured-but-empty streams.
	if params.Stderr != nil {
		t.Errorf("Stderr = %v, want NULL (zone carries no stderr)", params.Stderr)
	}
	if params.SentScope != nil {
		t.Errorf("SentScope = %v, want NULL (zone sends nothing)", params.SentScope)
	}
}

// TestClassifyProberOutcome checks the two branches that do not need a real
// ProcessState: a ctx error wins over any exit, and a prober that never started
// (nil ProcessState) reads as exited(-1), an honest "no clean exit".
func TestClassifyProberOutcome(t *testing.T) {
	if got := classifyProberOutcome(nil, context.Canceled); got != (wire.ProberContextCancelled{}) {
		t.Errorf("ctx cancelled: got %#v, want ProberContextCancelled", got)
	}
	got := classifyProberOutcome(nil, nil)
	exited, ok := got.(wire.ProberExited)
	if !ok || exited.Code != -1 {
		t.Errorf("nil ProcessState: got %#v, want ProberExited{-1}", got)
	}
}

// TestPersistTranscriptGate checks the WithTranscripts seam: an unwired worker and a
// devMode worker both report capture off, so they persist nothing (no golden fixture
// moves); an absent transcript is a no-op even when capture is on.
func TestPersistTranscriptGate(t *testing.T) {
	if (&Worker{}).captureOn() {
		t.Error("unwired worker: captureOn=true, want false")
	}
	if (&Worker{captureTranscripts: true, devMode: true}).captureOn() {
		t.Error("devMode worker: captureOn=true, want false")
	}
	if !(&Worker{captureTranscripts: true}).captureOn() {
		t.Error("wired non-dev worker: captureOn=false, want true")
	}

	// An absent transcript with capture on is a no-op — no InsertTranscript, so a nil
	// qtx is never touched. A real failure here would panic on the nil qtx.
	w := &Worker{captureTranscripts: true, transcriptKey: testKey, now: time.Now}
	if err := w.persistTranscript(context.Background(), nil, 1, nil); err != nil {
		t.Errorf("absent transcript: got err %v, want nil no-op", err)
	}
}
