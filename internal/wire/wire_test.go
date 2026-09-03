package wire

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestSCTCaptureRoundTrip(t *testing.T) {
	want := SCTCapture{
		TLSExt: [][]byte{[]byte("sct-a"), []byte("sct-b")},
		OCSP:   []byte("ocsp"),
	}
	b := EncodeSCTCapture(want)
	if len(b) == 0 {
		t.Fatal("EncodeSCTCapture returned empty for a non-empty capture")
	}
	got, err := DecodeSCTCapture(b)
	if err != nil {
		t.Fatalf("DecodeSCTCapture: %v", err)
	}
	if len(got.TLSExt) != 2 || !bytes.Equal(got.TLSExt[0], want.TLSExt[0]) || !bytes.Equal(got.TLSExt[1], want.TLSExt[1]) {
		t.Errorf("TLSExt round trip: got %v, want %v", got.TLSExt, want.TLSExt)
	}
	if !bytes.Equal(got.OCSP, want.OCSP) {
		t.Errorf("OCSP round trip: got %q, want %q", got.OCSP, want.OCSP)
	}
}

func TestSCTCaptureEmpty(t *testing.T) {
	if b := EncodeSCTCapture(SCTCapture{}); b != nil {
		t.Errorf("EncodeSCTCapture(empty) = %q, want nil", b)
	}
	got, err := DecodeSCTCapture(nil)
	if err != nil {
		t.Fatalf("DecodeSCTCapture(nil): %v", err)
	}
	if len(got.TLSExt) != 0 || len(got.OCSP) != 0 {
		t.Errorf("DecodeSCTCapture(nil) = %+v, want zero", got)
	}
}

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

func TestLimitedBufferFailsClosed(t *testing.T) {
	// #772: the sink errors past the ceiling rather than buffering without bound.
	small := NewLimitedBuffer(1024)
	if _, err := small.Write([]byte("hello")); err != nil {
		t.Fatalf("write under cap: %v", err)
	}
	if got := string(small.Bytes()); got != "hello" {
		t.Fatalf("Bytes = %q, want %q", got, "hello")
	}

	b := NewLimitedBuffer(8)
	if _, err := b.Write([]byte("1234")); err != nil {
		t.Fatalf("first write within cap: %v", err)
	}
	_, err := b.Write([]byte("56789"))
	if !errors.Is(err, ErrProberOutputTooLarge) {
		t.Fatalf("over-cap write err = %v, want ErrProberOutputTooLarge", err)
	}
	if len(b.Bytes()) > 8 {
		t.Fatalf("retained %d bytes, cap is 8", len(b.Bytes()))
	}
	if _, err := b.Write([]byte("x")); !errors.Is(err, ErrProberOutputTooLarge) {
		t.Fatalf("post-overflow write err = %v, want ErrProberOutputTooLarge", err)
	}
}

func TestLimitedBufferBoundsUnboundedStream(t *testing.T) {
	// io.Copy is the shape conn.Run and cmd.Stdout use, so the sink is exercised as it is used.
	limit := 64 * 1024
	sink := NewLimitedBuffer(limit)
	flood := bytes.NewReader(bytes.Repeat([]byte("A"), 4<<20))
	_, err := io.Copy(sink, flood)
	if !errors.Is(err, ErrProberOutputTooLarge) {
		t.Fatalf("io.Copy err = %v, want ErrProberOutputTooLarge", err)
	}
	if len(sink.Bytes()) > limit {
		t.Fatalf("sink retained %d bytes, cap is %d", len(sink.Bytes()), limit)
	}
}

func TestObservationScannerCapsLineLength(t *testing.T) {
	huge := bytes.Repeat([]byte("A"), MaxObservationLine+1)
	sc := NewObservationScanner(bytes.NewReader(huge))
	if sc.Next() {
		t.Fatal("expected Next to reject an over-long line")
	}
	if sc.Err() == nil {
		t.Fatal("expected a scan error for the over-long line, got nil")
	}
}

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
