package wire

import (
	"bytes"
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
