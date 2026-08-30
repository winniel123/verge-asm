package scan

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// readFixture loads one base64 fixture (a real leaf_input / extra_data captured from a
// live CT log) and decodes it to the raw bytes LeafSANs reads.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	raw, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return raw
}

// TestLeafSANs decodes real CT log entries — one x509 entry and one precert entry,
// captured from Google's Argon2026h2 log — and confirms both reach the same SAN. The
// precert path is the one the hand-rolled decoder most has to get right: the leaf holds
// only a TBSCertificate, so the certificate is read from extra_data instead.
func TestLeafSANs(t *testing.T) {
	cases := []struct {
		name  string
		leaf  string
		extra string
	}{
		{"x509_entry", "entry1_leaf.b64", "entry1_extra.b64"},
		{"precert_entry", "entry0_leaf.b64", "entry0_extra.b64"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			leaf := readFixture(t, c.leaf)
			extra := readFixture(t, c.extra)
			names, err := LeafSANs(leaf, extra)
			if err != nil {
				t.Fatalf("LeafSANs: %v", err)
			}
			if len(names) != 1 || names[0] != "flowers-to-the-world.com" {
				t.Fatalf("got SANs %v, want [flowers-to-the-world.com]", names)
			}
		})
	}
}

// TestLeafSANsMalformed confirms a truncated or wrong-version leaf is rejected rather
// than read past its bounds — a decode error the poll skips, never a silent empty read.
func TestLeafSANsMalformed(t *testing.T) {
	if _, err := LeafSANs([]byte{0, 0, 1}, nil); err == nil {
		t.Fatal("want error on a too-short leaf")
	}
	// A valid header but an unknown version.
	bad := make([]byte, ctLeafHeader)
	bad[0] = 9
	if _, err := LeafSANs(bad, nil); err == nil {
		t.Fatal("want error on an unsupported leaf version")
	}
}

// TestLeafSANsUnknownEntryType confirms an entry type the tail does not recognise yields
// no names and no error — the tail tolerates a future entry type rather than failing the
// whole poll on it.
func TestLeafSANsUnknownEntryType(t *testing.T) {
	leaf := make([]byte, ctLeafHeader)
	// version 0, leaf_type 0, timestamp 0, entry_type 99 (bytes 10:12).
	leaf[10] = 0
	leaf[11] = 99
	names, err := LeafSANs(leaf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if names != nil {
		t.Fatalf("got %v, want no names for an unknown entry type", names)
	}
}

// TestSelectTailLogs checks the embedded log_list.json snapshot yields a non-empty,
// well-formed, deterministic RFC 6962 log-set — every entry carries an id and a url, and
// the set is sorted by log_id.
func TestSelectTailLogs(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	logs, err := SelectTailLogs(now)
	if err != nil {
		t.Fatalf("SelectTailLogs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("want at least one followed log from the embedded snapshot")
	}
	for i, l := range logs {
		if l.LogID == "" || l.URL == "" {
			t.Fatalf("log %d missing id or url: %+v", i, l)
		}
		if i > 0 && logs[i-1].LogID >= l.LogID {
			t.Fatalf("logs not sorted by id at %d: %q >= %q", i, logs[i-1].LogID, l.LogID)
		}
	}
	// Deterministic: a second call returns the same set.
	again, _ := SelectTailLogs(now)
	if len(again) != len(logs) {
		t.Fatalf("non-deterministic selection: %d then %d", len(logs), len(again))
	}
}

func TestTailReadableState(t *testing.T) {
	raw := func(keys ...string) map[string]json.RawMessage {
		m := map[string]json.RawMessage{}
		for _, k := range keys {
			m[k] = json.RawMessage("{}")
		}
		return m
	}
	cases := []struct {
		keys []string
		want bool
	}{
		{[]string{"usable"}, true},
		{[]string{"readonly"}, true},
		{[]string{"retired"}, false},
		{[]string{"qualified"}, false},
		{nil, false},
	}
	for _, c := range cases {
		if got := tailReadableState(raw(c.keys...)); got != c.want {
			t.Errorf("state %v: got %v want %v", c.keys, got, c.want)
		}
	}
}

func TestTailCoversNow(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	mk := func(start, end string) *temporalInterval {
		s, _ := time.Parse(time.RFC3339, start)
		e, _ := time.Parse(time.RFC3339, end)
		return &temporalInterval{StartInclusive: s, EndExclusive: e}
	}
	if !tailCoversNow(nil, now) {
		t.Error("nil interval should always be covered")
	}
	if !tailCoversNow(mk("2026-07-01T00:00:00Z", "2027-01-01T00:00:00Z"), now) {
		t.Error("current shard should be covered")
	}
	if tailCoversNow(mk("2026-01-01T00:00:00Z", "2026-07-01T00:00:00Z"), now) {
		t.Error("an already-ended shard should not be covered")
	}
	if !tailCoversNow(mk("2027-01-01T00:00:00Z", "2027-07-01T00:00:00Z"), now) {
		t.Error("the next shard (within the horizon) should be covered")
	}
	if tailCoversNow(mk("2028-06-01T00:00:00Z", "2029-01-01T00:00:00Z"), now) {
		t.Error("a shard beyond the horizon should not be covered")
	}
}

// TestAdmitCTNames covers the admission decision: scope filter, wildcard refusal, dedup,
// and most-specific-seed attribution.
func TestAdmitCTNames(t *testing.T) {
	seeds := []CTSeed{
		{SeedID: 1, Domain: "example.com"},
		{SeedID: 2, Domain: "sub.example.com"},
	}
	sans := []string{
		"www.example.com",    // in scope, under example.com
		"a.sub.example.com",  // in scope, most-specific is sub.example.com
		"www.example.com",    // duplicate — dropped
		"*.example.com",      // wildcard — refused (ADR-0060)
		"foo.notexample.com", // out of scope — dropped (ADR-0047)
		"other.org",          // out of scope — dropped
	}
	got := AdmitCTNames(sans, seeds)
	want := map[string]int64{
		"www.example.com":   1,
		"a.sub.example.com": 2,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d admissions %+v, want %d", len(got), got, len(want))
	}
	for _, a := range got {
		wantSeed, ok := want[a.Name]
		if !ok {
			t.Errorf("unexpected admission %q", a.Name)
			continue
		}
		if a.SeedID != wantSeed {
			t.Errorf("%q admitted under seed %d, want %d", a.Name, a.SeedID, wantSeed)
		}
	}
}

// TestAdmitCTNamesEmptySeeds confirms that with no declared scope nothing is admitted —
// the tail reads the whole firehose and keeps only what a Seed covers.
func TestAdmitCTNamesEmptySeeds(t *testing.T) {
	got := AdmitCTNames([]string{"www.example.com"}, nil)
	if len(got) != 0 {
		t.Fatalf("got %+v, want no admissions with no seeds", got)
	}
}

// TestCTTailScopeRoundTrip confirms a job's wire scope decodes back to the log it names.
func TestCTTailScopeRoundTrip(t *testing.T) {
	j := CTTailJob{ScanID: 7, Log: CTLog{LogID: "abc=", URL: "https://ct.example/log/", Description: "Example log"}}
	spec, err := j.JobSpec("scan:7:log:abc=")
	if err != nil {
		t.Fatalf("JobSpec: %v", err)
	}
	if spec.Kind != CTTailKind {
		t.Fatalf("kind %q, want %q", spec.Kind, CTTailKind)
	}
	got, err := CTTailScopeFromSpec(spec.Scope)
	if err != nil {
		t.Fatalf("CTTailScopeFromSpec: %v", err)
	}
	if got != j.Log {
		t.Fatalf("round-trip got %+v, want %+v", got, j.Log)
	}
}

// TestParseSTH and TestParseLogEntries cover the RFC 6962 wire parsers' happy and
// malformed paths — a malformed 200 is an error the poll treats as transient, never an
// empty read (ADR-0027 §7).
func TestParseSTH(t *testing.T) {
	sth, err := ParseSTH([]byte(`{"tree_size":42,"timestamp":1,"sha256_root_hash":"x","tree_head_signature":"y"}`))
	if err != nil {
		t.Fatalf("ParseSTH: %v", err)
	}
	if sth.TreeSize != 42 {
		t.Fatalf("tree size %d, want 42", sth.TreeSize)
	}
	if string(sth.Raw) == "" {
		t.Fatal("raw signed head not retained")
	}
	if _, err := ParseSTH([]byte("not json")); err == nil {
		t.Fatal("want error on malformed get-sth")
	}
}

func TestParseLogEntries(t *testing.T) {
	body := `{"entries":[{"leaf_input":"AAA=","extra_data":"AAA="}]}`
	entries, err := ParseLogEntries([]byte(body))
	if err != nil {
		t.Fatalf("ParseLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if _, err := ParseLogEntries([]byte("not json")); err == nil {
		t.Fatal("want error on malformed get-entries")
	}
}
