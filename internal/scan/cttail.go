// The `ct-tail` Scan is CT-logs-direct's drift tail (ct-source-replacement.md §4).
// It reads the logs directly, fanning out per-log with a durable cursor, and reads
// only FORWARD deltas, never backfilling history (§4.1).
package scan

import (
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

const CTTailKind = "ct-tail"

const CTTailSource = "ct-tail"

// Embedded rather than fetched live, so the log set is deterministic and needs no network.
// Each log's public key is stripped: no signature is verified here, so it is unused weight.
// Stripping the keys also keeps this file out of a secret scanner's generic-key heuristic.

//go:embed log_list.json
var embeddedLogList []byte

// A CA pre-issues into the next shard before it opens, so following one ahead catches it (§4.3).

const nextShardHorizon = 366 * 24 * time.Hour

// A tiled log's URL is its monitoring_url, so the client discriminator must travel with it (§4.3).

type CTLog struct {
	LogID       string
	URL         string
	Description string
	Tiled       bool
}

type logList struct {
	Version   string `json:"version"`
	Operators []struct {
		Logs      []logListEntry      `json:"logs"`
		TiledLogs []tiledLogListEntry `json:"tiled_logs"`
	} `json:"operators"`
}

type logListEntry struct {
	Description      string                     `json:"description"`
	LogID            string                     `json:"log_id"`
	URL              string                     `json:"url"`
	State            map[string]json.RawMessage `json:"state"`
	TemporalInterval *temporalInterval          `json:"temporal_interval"`
}

// A tiled log's submission_url is write-only for a CA, so the tail reads monitoring_url (§4.3).

type tiledLogListEntry struct {
	Description      string                     `json:"description"`
	LogID            string                     `json:"log_id"`
	MonitoringURL    string                     `json:"monitoring_url"`
	State            map[string]json.RawMessage `json:"state"`
	TemporalInterval *temporalInterval          `json:"temporal_interval"`
}

// A shard accepts a certificate by its expiry, not its issuance instant (§4.3).

type temporalInterval struct {
	StartInclusive time.Time `json:"start_inclusive"`
	EndExclusive   time.Time `json:"end_exclusive"`
}

func SelectTailLogs(now time.Time) ([]CTLog, error) {
	var ll logList
	if err := json.Unmarshal(embeddedLogList, &ll); err != nil {
		return nil, fmt.Errorf("scan: decode ct log list: %w", err)
	}
	var out []CTLog
	for _, op := range ll.Operators {
		for _, e := range op.Logs {
			if !tailReadableState(e.State) || !tailCoversNow(e.TemporalInterval, now) {
				continue
			}
			if e.LogID == "" || e.URL == "" {
				continue
			}
			out = append(out, CTLog{LogID: e.LogID, URL: e.URL, Description: e.Description})
		}
		for _, e := range op.TiledLogs {
			if !tailReadableState(e.State) || !tailCoversNow(e.TemporalInterval, now) {
				continue
			}
			if e.LogID == "" || e.MonitoringURL == "" {
				continue
			}
			out = append(out, CTLog{LogID: e.LogID, URL: e.MonitoringURL, Description: e.Description, Tiled: true})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LogID < out[j].LogID })
	return out, nil
}

func tailReadableState(state map[string]json.RawMessage) bool {
	// A retired log may 404, so only usable and readonly are followed (§4.3).
	_, usable := state["usable"]
	_, readonly := state["readonly"]
	return usable || readonly
}

func tailCoversNow(ti *temporalInterval, now time.Time) bool {
	if ti == nil {
		return true
	}
	return ti.EndExclusive.After(now) && ti.StartInclusive.Before(now.Add(nextShardHorizon))
}

// A logged certificate is not a function of where we read the log, so no Vantage rides (§4.2).

type CTTailJob struct {
	ScanID int64
	Log    CTLog
}

func BuildCTTailJobs(scanID int64, logs []CTLog) []CTTailJob {
	if len(logs) == 0 {
		return nil
	}
	jobs := make([]CTTailJob, 0, len(logs))
	for _, l := range logs {
		jobs = append(jobs, CTTailJob{ScanID: scanID, Log: l})
	}
	return jobs
}

type ctTailScope struct {
	LogID       string `json:"log_id"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Tiled       bool   `json:"tiled,omitempty"`
}

func (j CTTailJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(ctTailScope{LogID: j.Log.LogID, URL: j.Log.URL, Description: j.Log.Description, Tiled: j.Log.Tiled})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal ct-tail scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: CTTailKind, Scope: raw}, nil
}

func (j CTTailJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(ctTailScopeRecord{LogID: j.Log.LogID, Description: j.Log.Description})
}

func EmptyCTTailScope() ([]byte, error) {
	return json.Marshal(ctTailScopeRecord{})
}

type ctTailScopeRecord struct {
	LogID       string `json:"log_id,omitempty"`
	Description string `json:"description,omitempty"`
}

func CTTailScopeFromSpec(scope []byte) (CTLog, error) {
	var s ctTailScope
	if err := json.Unmarshal(scope, &s); err != nil {
		return CTLog{}, fmt.Errorf("scan: decode ct-tail scope: %w", err)
	}
	return CTLog{LogID: s.LogID, URL: s.URL, Description: s.Description, Tiled: s.Tiled}, nil
}

// The raw body is the exact bytes the log signed, which a later consistency proof needs (§4.2).

type CTSignedTreeHead struct {
	TreeSize int64
	Raw      []byte
}

func ParseSTH(body []byte) (CTSignedTreeHead, error) {
	var raw struct {
		TreeSize int64 `json:"tree_size"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return CTSignedTreeHead{}, fmt.Errorf("scan: decode get-sth: %w", err)
	}
	if raw.TreeSize < 0 {
		return CTSignedTreeHead{}, fmt.Errorf("scan: get-sth negative tree size %d", raw.TreeSize)
	}
	return CTSignedTreeHead{TreeSize: raw.TreeSize, Raw: body}, nil
}

type CTLogEntry struct {
	LeafInput []byte
	ExtraData []byte
}

// A log may return fewer entries than asked, so the cursor advances by the count returned (§4.4).

func ParseLogEntries(body []byte) ([]CTLogEntry, error) {
	var raw struct {
		Entries []struct {
			LeafInput string `json:"leaf_input"`
			ExtraData string `json:"extra_data"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("scan: decode get-entries: %w", err)
	}
	out := make([]CTLogEntry, 0, len(raw.Entries))
	for i, e := range raw.Entries {
		leaf, err := base64.StdEncoding.DecodeString(e.LeafInput)
		if err != nil {
			return nil, fmt.Errorf("scan: get-entries leaf %d base64: %w", i, err)
		}
		extra, err := base64.StdEncoding.DecodeString(e.ExtraData)
		if err != nil {
			return nil, fmt.Errorf("scan: get-entries extra %d base64: %w", i, err)
		}
		out = append(out, CTLogEntry{LeafInput: leaf, ExtraData: extra})
	}
	return out, nil
}

// These are RFC 6962 §3.4 wire constants, not values of ours to choose.

const (
	ctLeafVersionV1     = 0
	ctLeafTypeTimestamp = 0
	ctEntryX509         = 0
	ctEntryPrecert      = 1

	ctLeafHeader    = 1 + 1 + 8 + 2
	ctASN1CertLen   = 3
	ctIssuerKeyHash = 32
)

func LeafSANs(leafInput, extraData []byte) ([]string, error) {
	if len(leafInput) < ctLeafHeader {
		return nil, fmt.Errorf("scan: ct leaf too short (%d bytes)", len(leafInput))
	}
	if leafInput[0] != ctLeafVersionV1 {
		return nil, fmt.Errorf("scan: ct leaf version %d unsupported", leafInput[0])
	}
	if leafInput[1] != ctLeafTypeTimestamp {
		return nil, fmt.Errorf("scan: ct leaf type %d unsupported", leafInput[1])
	}
	entryType := binary.BigEndian.Uint16(leafInput[10:12])

	var der []byte
	switch entryType {
	case ctEntryX509:
		d, err := readOpaque24(leafInput[ctLeafHeader:])
		if err != nil {
			return nil, fmt.Errorf("scan: ct x509 leaf: %w", err)
		}
		der = d
	case ctEntryPrecert:
		// A precert leaf holds no parseable certificate, so extra_data carries it (RFC 6962 §3.4).
		d, err := readOpaque24(extraData)
		if err != nil {
			return nil, fmt.Errorf("scan: ct precert extra_data: %w", err)
		}
		der = d
	default:
		return nil, nil // a future entry type is tolerated, never a failed poll
	}

	return CertSANs(der)
}

func CertSANs(der []byte) ([]string, error) {
	// The CT poison is a critical extension, but rejection is a Verify-time check we never run.
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("scan: parse ct certificate: %w", err)
	}
	names := make([]string, 0, len(cert.DNSNames)+1)
	names = append(names, cert.DNSNames...)
	if cert.Subject.CommonName != "" {
		names = append(names, cert.Subject.CommonName)
	}
	return names, nil
}

func readOpaque24(b []byte) ([]byte, error) {
	if len(b) < ctASN1CertLen {
		return nil, fmt.Errorf("truncated length prefix")
	}
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	// A TLS opaque<1..2^24-1> has a minimum of one byte, so a zero length is malformed.
	if n == 0 {
		return nil, fmt.Errorf("zero-length value")
	}
	if len(b) < ctASN1CertLen+n {
		return nil, fmt.Errorf("length %d overruns %d-byte buffer", n, len(b)-ctASN1CertLen)
	}
	return b[ctASN1CertLen : ctASN1CertLen+n], nil
}

// A full tile is exactly 256 entries by spec, so the tile is also the request size (§4.4).

const CTTileWidth = 256

func ParseCheckpoint(body []byte) (CTSignedTreeHead, error) {
	// A C2SP signed note is origin, decimal tree size, base64 root hash, then signatures (§4.2).
	lines := strings.Split(string(body), "\n")
	if len(lines) < 3 {
		return CTSignedTreeHead{}, fmt.Errorf("scan: checkpoint has %d lines, want at least 3", len(lines))
	}
	size, err := strconv.ParseInt(strings.TrimSpace(lines[1]), 10, 64)
	if err != nil {
		return CTSignedTreeHead{}, fmt.Errorf("scan: checkpoint tree size %q: %w", lines[1], err)
	}
	if size < 0 {
		return CTSignedTreeHead{}, fmt.Errorf("scan: checkpoint negative tree size %d", size)
	}
	return CTSignedTreeHead{TreeSize: size, Raw: body}, nil
}

func DataTilePath(index int64) string {
	// The x-prefixed base-1000 segments are the spec's encoding, bounding directory fan-out (§4.3).
	segs := []string{fmt.Sprintf("%03d", index%1000)}
	for index /= 1000; index > 0; index /= 1000 {
		segs = append([]string{fmt.Sprintf("%03d", index%1000)}, segs...)
	}
	for i := 0; i < len(segs)-1; i++ {
		segs[i] = "x" + segs[i]
	}
	return strings.Join(segs, "/")
}

func ParseDataTile(body []byte) ([][]byte, error) {
	var out [][]byte
	b := body
	for len(b) > 0 {
		der, rest, err := parseTileLeaf(b)
		if err != nil {
			return nil, fmt.Errorf("scan: data tile leaf %d: %w", len(out), err)
		}
		out = append(out, der)
		b = rest
	}
	return out, nil
}

func parseTileLeaf(b []byte) (der, rest []byte, err error) {
	const timestampLen, entryTypeLen = 8, 2
	// A tile leaf carries no version or leaf_type header, unlike a get-entries leaf (§4.3).
	if len(b) < timestampLen+entryTypeLen {
		return nil, nil, fmt.Errorf("truncated leaf header")
	}
	entryType := binary.BigEndian.Uint16(b[timestampLen : timestampLen+entryTypeLen])
	b = b[timestampLen+entryTypeLen:]

	switch entryType {
	case ctEntryX509:
		cert, after, e := takeOpaque24(b)
		if e != nil {
			return nil, nil, fmt.Errorf("x509 signed_entry: %w", e)
		}
		der = cert
		b = after
	case ctEntryPrecert:
		if len(b) < ctIssuerKeyHash {
			return nil, nil, fmt.Errorf("truncated precert issuer_key_hash")
		}
		b = b[ctIssuerKeyHash:]
		_, after, e := takeOpaque24(b)
		if e != nil {
			return nil, nil, fmt.Errorf("precert TBSCertificate: %w", e)
		}
		b = after
	default:
		// An unknown entry type has an unknown length, so the tile cannot be framed past it.
		return nil, nil, fmt.Errorf("unsupported entry type %d", entryType)
	}

	_, after, e := takeOpaque16(b)
	if e != nil {
		return nil, nil, fmt.Errorf("extensions: %w", e)
	}
	b = after

	if entryType == ctEntryPrecert {
		cert, afterPre, pe := takeOpaque24(b)
		if pe != nil {
			return nil, nil, fmt.Errorf("pre_certificate: %w", pe)
		}
		der = cert
		b = afterPre
	}

	_, afterChain, ce := takeOpaque16(b)
	if ce != nil {
		return nil, nil, fmt.Errorf("certificate_chain: %w", ce)
	}
	return der, afterChain, nil
}

func takeOpaque24(b []byte) (val, rest []byte, err error) {
	v, err := readOpaque24(b)
	if err != nil {
		return nil, nil, err
	}
	return v, b[ctASN1CertLen+len(v):], nil
}

func takeOpaque16(b []byte) (val, rest []byte, err error) {
	if len(b) < 2 {
		return nil, nil, fmt.Errorf("truncated length prefix")
	}
	n := int(b[0])<<8 | int(b[1])
	if len(b) < 2+n {
		return nil, nil, fmt.Errorf("length %d overruns %d-byte buffer", n, len(b)-2)
	}
	return b[2 : 2+n], b[2+n:], nil
}

// The tail reads the whole firehose, so each name's Seed is resolved per name, not per query.

type CTAdmission struct {
	Name   string
	SeedID int64
}

func AdmitCTNames(sans []string, seeds []CTSeed) []CTAdmission {
	seen := map[string]struct{}{}
	out := make([]CTAdmission, 0)
	for _, raw := range sans {
		if len(out) >= MaxAdmittedNames {
			break
		}
		n := normaliseName(raw)
		if n == "" {
			continue
		}
		// An asterisk denotes a set or has two denotations, so it admits no Name (ADR-0060).
		if strings.Contains(n, "*") {
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seedID, ok := coveringSeed(n, seeds)
		if !ok {
			continue // ADR-0047: the Seed decides what is inside, so a foreign SAN admits nothing
		}
		seen[n] = struct{}{}
		out = append(out, CTAdmission{Name: n, SeedID: seedID})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func coveringSeed(name string, seeds []CTSeed) (int64, bool) {
	// The most specific Seed wins, so a Citation names the scope that accounts for it (ADR-0047).
	bestID := int64(0)
	bestLen := -1
	for _, s := range seeds {
		d := normaliseName(s.Domain)
		if !withinScope(name, d) {
			continue
		}
		if len(d) > bestLen {
			bestLen = len(d)
			bestID = s.SeedID
		}
	}
	return bestID, bestLen >= 0
}
