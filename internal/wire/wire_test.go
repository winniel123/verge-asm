package wire

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestJobSpecRoundTrip(t *testing.T) {
	want := JobSpec{Batch: "b1", Kind: "tcp-connect", Scope: []byte(`{"port":443}`)}

	var buf bytes.Buffer
	if err := EncodeJobSpec(&buf, want); err != nil {
		t.Fatalf("EncodeJobSpec: %v", err)
	}

	got, err := DecodeJobSpec(&buf)
	if err != nil {
		t.Fatalf("DecodeJobSpec: %v", err)
	}
	if got.Batch != want.Batch || got.Kind != want.Kind || string(got.Scope) != string(want.Scope) {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestDecodeJobSpecInvalid(t *testing.T) {
	if _, err := DecodeJobSpec(strings.NewReader("not json")); err == nil {
		t.Fatal("expected error decoding invalid job spec, got nil")
	}
}

func TestObservationScannerReadsMultipleLines(t *testing.T) {
	input := strings.NewReader(
		`{"batch":"b1","kind":"tcp-connect","address":"10.0.0.1"}` + "\n" +
			`{"batch":"b1","kind":"tcp-connect","address":"10.0.0.2","err":"timeout"}` + "\n",
	)

	sc := NewObservationScanner(input)

	var got []Observation
	for sc.Next() {
		got = append(got, sc.Observation())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d observations, want 2", len(got))
	}
	if got[0].Address != "10.0.0.1" || got[1].Err != "timeout" {
		t.Fatalf("unexpected observations: %+v", got)
	}
}

func TestObservationScannerDoesNotLeakFieldsBetweenLines(t *testing.T) {
	// The first line sets Address and Err; the second omits both
	// (omitempty). A reused, un-reset struct would let the second
	// observation inherit the first's values.
	input := strings.NewReader(
		`{"batch":"b1","kind":"tcp-connect","address":"10.0.0.1","err":"timeout"}` + "\n" +
			`{"batch":"b1","kind":"tcp-connect"}` + "\n",
	)

	sc := NewObservationScanner(input)

	if !sc.Next() {
		t.Fatalf("expected first observation, scanner error: %v", sc.Err())
	}
	first := sc.Observation()

	if !sc.Next() {
		t.Fatalf("expected second observation, scanner error: %v", sc.Err())
	}
	second := sc.Observation()

	if first.Address != "10.0.0.1" || first.Err != "timeout" {
		t.Fatalf("unexpected first observation: %+v", first)
	}
	if second.Address != "" || second.Err != "" {
		t.Fatalf("second observation leaked fields from the first: %+v", second)
	}
}

func TestObservationScannerStopsOnBadLine(t *testing.T) {
	sc := NewObservationScanner(strings.NewReader("not json\n"))
	if sc.Next() {
		t.Fatal("expected Next to return false on invalid JSON")
	}
	if sc.Err() == nil {
		t.Fatal("expected a decode error, got nil")
	}
}

// TestLimitedBufferFailsClosed proves the prober-stdout sink accepts a
// normal-sized stream but fails closed — errors, retaining no more than the cap —
// once a prober streams past the ceiling, rather than buffering without bound (#772).
func TestLimitedBufferFailsClosed(t *testing.T) {
	// A normal-sized write under the cap succeeds and round-trips.
	small := NewLimitedBuffer(1024)
	if _, err := small.Write([]byte("hello")); err != nil {
		t.Fatalf("write under cap: %v", err)
	}
	if got := string(small.Bytes()); got != "hello" {
		t.Fatalf("Bytes = %q, want %q", got, "hello")
	}

	// A stream exceeding the cap errors and never retains more than the cap.
	b := NewLimitedBuffer(8)
	if _, err := b.Write([]byte("1234")); err != nil {
		t.Fatalf("first write within cap: %v", err)
	}
	_, err := b.Write([]byte("56789")) // would push total to 9 > 8
	if !errors.Is(err, ErrProberOutputTooLarge) {
		t.Fatalf("over-cap write err = %v, want ErrProberOutputTooLarge", err)
	}
	if len(b.Bytes()) > 8 {
		t.Fatalf("retained %d bytes, cap is 8", len(b.Bytes()))
	}
	// Stays failed closed for every subsequent write.
	if _, err := b.Write([]byte("x")); !errors.Is(err, ErrProberOutputTooLarge) {
		t.Fatalf("post-overflow write err = %v, want ErrProberOutputTooLarge", err)
	}
}

// TestLimitedBufferBoundsUnboundedStream feeds the sink far more than its cap via
// io.Copy (the shape conn.Run / cmd.Stdout use) and asserts it errors instead of
// consuming unboundedly — the OOM the finding described.
func TestLimitedBufferBoundsUnboundedStream(t *testing.T) {
	limit := 64 * 1024
	sink := NewLimitedBuffer(limit)
	// A 4 MiB "malicious" stream — well past the cap.
	flood := bytes.NewReader(bytes.Repeat([]byte("A"), 4<<20))
	_, err := io.Copy(sink, flood)
	if !errors.Is(err, ErrProberOutputTooLarge) {
		t.Fatalf("io.Copy err = %v, want ErrProberOutputTooLarge", err)
	}
	if len(sink.Bytes()) > limit {
		t.Fatalf("sink retained %d bytes, cap is %d", len(sink.Bytes()), limit)
	}
}

// TestObservationScannerCapsLineLength proves an over-long single line is rejected
// via the explicit per-line cap rather than over-allocating.
func TestObservationScannerCapsLineLength(t *testing.T) {
	// One line longer than MaxObservationLine, no newline splits it.
	huge := bytes.Repeat([]byte("A"), MaxObservationLine+1)
	sc := NewObservationScanner(bytes.NewReader(huge))
	if sc.Next() {
		t.Fatal("expected Next to reject an over-long line")
	}
	if sc.Err() == nil {
		t.Fatal("expected a scan error for the over-long line, got nil")
	}
}

// TestObservationScannerCapsCount proves a flood of tiny valid lines is bounded by
// MaxObservations, so the decoded slice cannot grow without limit.
func TestObservationScannerCapsCount(t *testing.T) {
	var buf bytes.Buffer
	for i := 0; i < MaxObservations+10; i++ {
		fmt.Fprintf(&buf, "{\"batch\":\"b\",\"kind\":\"k\"}\n")
	}
	sc := NewObservationScanner(&buf)
	n := 0
	for sc.Next() {
		n++
	}
	if n > MaxObservations {
		t.Fatalf("scanned %d observations, cap is %d", n, MaxObservations)
	}
	if !errors.Is(sc.Err(), ErrProberOutputTooLarge) {
		t.Fatalf("Err = %v, want ErrProberOutputTooLarge after count cap", sc.Err())
	}
}
