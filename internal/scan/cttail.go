// This file builds the `ct-tail` Scan (spec docs/spec/ct-source-replacement.md §4,
// map #854): CT-logs-direct's drift tail. It watches new issuance for names the
// operator already knows by reading the CT logs DIRECTLY, forward-only, rather than
// querying an index like crt.sh. It admits the same way the `ct` Scan does — an
// `admitted_name` row on `authority: inferred` citing its Batch (ADR-0027) — but it
// fans out PER-LOG (not per name-scope Seed) and carries a durable per-log cursor so
// each poll reads only the delta appended since the last one. The design invariant is
// that the tail reads only FORWARD deltas and never backfills history (§4.1).
//
// This file holds the pure half — the log-set selection off the embedded log_list.json,
// the RFC 6962 wire parsers (get-sth, get-entries, the MerkleTreeLeaf that carries each
// certificate), the SAN extraction, and the admission decision that decides what a set
// of log entries admits. The impure half (the forward-delta fetch, the cursor advance,
// the retry/dead-letter, the admission write and the ephemeral drift event) is in
// internal/queue/cttail.go. #874 ships the RFC 6962 client only; the static-ct-api
// (tiled) client for the tiled-only logs is #877 (§4.3).
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

// CTTailKind is the scan kind this file dispatches — the drift tail over CT-logs-direct,
// a SECOND CT Scan beside the bulk `ct` poll (ADR-0106). It is named for what it reads
// (the log tail), not the instrument, the same ADR-0084 move that named `ct`.
const CTTailKind = "ct-tail"

// CTTailSource is the source every tail admission is attributed to: the keyless
// CT-logs-direct tail, `authority: inferred`, `completeness: corroborative`, opt-in
// (DefaultOn: false — §4.4). It matches the source-catalogue slug so the enablement
// state keys line up, exactly as CrtshSource does for `ct`.
const CTTailSource = "ct-tail"

// embeddedLogList is the pinned snapshot of Google's CT log_list.json (v89.34,
// 2026-08-29 — the version §4.3 names). The tail selects which logs it follows from
// this list. It is embedded rather than fetched live so the log-set is deterministic
// and testable and the fan-out reads it with no network step inside its transaction;
// a live-refresh path is deferred to fog. Refreshing it is a snapshot bump, not a
// schema change.
//
// Each log's `key` (its public key) is stripped from the snapshot: this file selects
// and reads logs by id, url, state and temporal_interval, and never verifies a
// signature (the consistency proof is opportunistic and deferred — §4.4), so the key
// is unused weight. Stripping it also keeps the file out of a secret scanner's
// generic-key heuristic. A signature-verifying successor re-embeds the keys.
//
//go:embed log_list.json
var embeddedLogList []byte

// nextShardHorizon is how far past now the tail still follows a not-yet-current log
// shard: the current shard plus the next one (§4.3). CT logs sharded by
// `temporal_interval` accept a certificate into the shard covering its expiry, and a
// CA pre-issues into the next shard before it opens, so following one shard ahead
// catches that issuance. A shard is about six months; a year's horizon covers the
// current shard and the whole of the next with margin.
const nextShardHorizon = 366 * 24 * time.Hour

// CTLog is one CT log the tail may follow: its log_id (the base64 SHA-256 of the log's
// public key, the ct_log_cursor primary key), the base URL the read endpoints hang off,
// a human description for the live event and logs, and which client reads it. Tiled
// selects the static-ct-api (tiled) client (#877, §4.3); false selects the RFC 6962
// client (#874). For an RFC 6962 log URL is the log_list.json `url`; for a tiled log it
// is the `monitoring_url` — the two clients hang different endpoint paths off it, so the
// discriminator travels with the log.
type CTLog struct {
	LogID       string
	URL         string
	Description string
	Tiled       bool
}

// logList is the subset of log_list.json the tail reads. Both log kinds are decoded: the
// RFC 6962 logs under operators[].logs[] (each carrying a `url`, #874) and the
// static-ct-api tiled logs under operators[].tiled_logs[] (each carrying a
// `monitoring_url`, #877 §4.3). The two arrays are the whole basis for the client
// discriminator — a log is tiled iff it comes from tiled_logs[].
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

// tiledLogListEntry is a static-ct-api log's log_list.json entry. It is read by
// monitoring_url — the prefix the checkpoint and tile paths hang off — not the
// submission_url (which is write-only, for a CA submitting a certificate). The state and
// temporal filters are identical to an RFC 6962 log's (§4.3).
type tiledLogListEntry struct {
	Description      string                     `json:"description"`
	LogID            string                     `json:"log_id"`
	MonitoringURL    string                     `json:"monitoring_url"`
	State            map[string]json.RawMessage `json:"state"`
	TemporalInterval *temporalInterval          `json:"temporal_interval"`
}

// temporalInterval is a CT log shard's coverage window: it accepts a certificate whose
// expiry falls in [start_inclusive, end_exclusive) (§4.3).
type temporalInterval struct {
	StartInclusive time.Time `json:"start_inclusive"`
	EndExclusive   time.Time `json:"end_exclusive"`
}

// SelectTailLogs is the pure log-set decision (§4.3): every CT log the tail follows at
// instant now, across BOTH client kinds — the RFC 6962 logs (operators[].logs[]) and the
// static-ct-api tiled logs (operators[].tiled_logs[]). A log is followed when BOTH hold —
// its state is `usable` or `readonly` (both readable; `retired` may 404 and placeholders
// are skipped), AND its temporal_interval covers now or the near future (the current
// shard plus the next, nextShardHorizon). A log with no temporal_interval is unsharded
// and always followed. Each result carries the Tiled discriminator so the worker reads it
// with the right client (§4.3). The result is sorted by log_id so the fan-out is
// deterministic across both kinds.
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
				continue // a placeholder entry carries no usable identity
			}
			out = append(out, CTLog{LogID: e.LogID, URL: e.URL, Description: e.Description})
		}
		for _, e := range op.TiledLogs {
			if !tailReadableState(e.State) || !tailCoversNow(e.TemporalInterval, now) {
				continue
			}
			if e.LogID == "" || e.MonitoringURL == "" {
				continue // a placeholder entry carries no usable identity
			}
			out = append(out, CTLog{LogID: e.LogID, URL: e.MonitoringURL, Description: e.Description, Tiled: true})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LogID < out[j].LogID })
	return out, nil
}

// tailReadableState reports whether a log's state makes it readable by the tail:
// `usable` or `readonly`. Every other state — `retired` (may 404), `pending`,
// `qualified`, `rejected`, or an empty/placeholder state — is skipped (§4.3).
func tailReadableState(state map[string]json.RawMessage) bool {
	_, usable := state["usable"]
	_, readonly := state["readonly"]
	return usable || readonly
}

// tailCoversNow reports whether a log's temporal_interval covers now or the next shard.
// A nil interval is unsharded and always covered. Otherwise the interval must not have
// ended (end_exclusive is after now) and must have started, or start, within the next
// shard horizon (§4.3).
func tailCoversNow(ti *temporalInterval, now time.Time) bool {
	if ti == nil {
		return true
	}
	return ti.EndExclusive.After(now) && ti.StartInclusive.Before(now.Add(nextShardHorizon))
}

// CTTailJob is one queue job the tail Scan produces: one log to poll. Unlike the `ct`
// job (one crt.sh query per Seed) the tail fans out per-log — the log is the unit of
// forward-delta work, and the name-scope Seeds it admits under are read on the worker
// side (§4.2). It carries no Vantage: a logged certificate is not a function of where
// we read the log from.
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

// JobSpec renders a CTTailJob into the wire JobSpec the worker reads. Like the `ct` and
// `zone` Scans there is no prober exec — the worker itself polls the log — so the log
// identity travels in the job rather than a vantage and offers.
func (j CTTailJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(ctTailScope{LogID: j.Log.LogID, URL: j.Log.URL, Description: j.Log.Description, Tiled: j.Log.Tiled})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal ct-tail scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: CTTailKind, Scope: raw}, nil
}

// AttemptedScope is the by-content record of what the tail job set out to cover: the
// log it polled. It is the completed Batch's recorded scope on success. A dead-lettered
// tail Batch records EmptyCTTailScope instead — never the log — because a failed poll of
// an append-only, corroborative source asserts no absence (ADR-0005, ADR-0027 §7).
func (j CTTailJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(ctTailScopeRecord{LogID: j.Log.LogID, Description: j.Log.Description})
}

// EmptyCTTailScope is what a dead-lettered tail Batch records — never the polled log,
// which would read as "we covered this log and found no new certificates", the
// false-absence a failed poll must never assert (ADR-0005, ADR-0027 §7).
func EmptyCTTailScope() ([]byte, error) {
	return json.Marshal(ctTailScopeRecord{})
}

type ctTailScopeRecord struct {
	LogID       string `json:"log_id,omitempty"`
	Description string `json:"description,omitempty"`
}

// CTTailScopeFromSpec decodes a tail job's wire scope back into the log to poll, so the
// worker polls the same log the dispatcher enqueued.
func CTTailScopeFromSpec(scope []byte) (CTLog, error) {
	var s ctTailScope
	if err := json.Unmarshal(scope, &s); err != nil {
		return CTLog{}, fmt.Errorf("scan: decode ct-tail scope: %w", err)
	}
	return CTLog{LogID: s.LogID, URL: s.URL, Description: s.Description, Tiled: s.Tiled}, nil
}

// --- RFC 6962 wire parsers ---------------------------------------------------

// CTSignedTreeHead is a log's current signed tree head (RFC 6962 get-sth): the tree
// size (the count of entries, so the tail reads positions [cursor, tree_size)) and the
// raw signed body kept verbatim as the durable cursor's signed_head. Keeping the raw
// body — rather than re-encoding the parsed fields — preserves the exact bytes the log
// signed, so a later consistency proof reads the head as it was served (§4.2, §4.4).
type CTSignedTreeHead struct {
	TreeSize int64
	Raw      []byte
}

// ParseSTH decodes a get-sth response body. A negative or unparseable tree size is an
// error, not a zero result: a malformed 200 is not evidence the log is empty (ADR-0027
// §7), so the caller treats a parse failure as transient rather than as "nothing new".
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

// ParseLogEntries decodes a get-entries response body into its entries. A body that is
// not the documented shape is an error, not an empty result (ADR-0027 §7). A get-entries
// answer legitimately returns FEWER entries than requested — per-operator batch caps
// (Argon 32/request, others 256/request — §4.4) — so the caller advances by the number
// returned, never by the number asked for.
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

// RFC 6962 §3.4 MerkleTreeLeaf / TimestampedEntry constants.
const (
	ctLeafVersionV1     = 0 // MerkleTreeLeaf.version = v1
	ctLeafTypeTimestamp = 0 // MerkleTreeLeaf.leaf_type = timestamped_entry
	ctEntryX509         = 0 // TimestampedEntry.entry_type = x509_entry
	ctEntryPrecert      = 1 // TimestampedEntry.entry_type = precert_entry

	// ctLeafHeader is version(1) + leaf_type(1) + timestamp(8) + entry_type(2): the
	// fixed prefix before the signed_entry (§3.4).
	ctLeafHeader    = 1 + 1 + 8 + 2
	ctASN1CertLen   = 3
	ctIssuerKeyHash = 32
)

// LeafSANs decodes one CT log entry to the DNS names its certificate carries — the SAN
// dNSName values and the subject common name, unfiltered. It is the tail's read half:
// the worker calls it per entry and hands the names to AdmitCTNames for the scope and
// wildcard rulings (§4.1). For an x509 entry the certificate DER sits in the leaf's
// TimestampedEntry; for a precert entry the leaf carries only the TBSCertificate, so the
// full pre_certificate is read from extra_data instead (RFC 6962 §3.4/§3.1). An entry
// type the tail does not recognise yields no names and no error — the tail tolerates a
// future entry type rather than failing the whole poll on it.
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
		// signed_entry is ASN.1Cert = opaque<1..2^24-1>: a 3-byte length then the DER.
		d, err := readOpaque24(leafInput[ctLeafHeader:])
		if err != nil {
			return nil, fmt.Errorf("scan: ct x509 leaf: %w", err)
		}
		der = d
	case ctEntryPrecert:
		// The precert leaf carries only issuer_key_hash + TBSCertificate, which is not
		// a parseable certificate. The full pre_certificate is the first ASN.1Cert of
		// extra_data's PrecertChainEntry (RFC 6962 §3.4).
		d, err := readOpaque24(extraData)
		if err != nil {
			return nil, fmt.Errorf("scan: ct precert extra_data: %w", err)
		}
		der = d
	default:
		return nil, nil // an unrecognised entry type admits nothing, and is not an error
	}

	return CertSANs(der)
}

// CertSANs decodes one certificate's DER to the DNS names it carries — the SAN dNSName
// values and the subject common name, unfiltered. It is the shared read half of both tail
// clients: the RFC 6962 client (LeafSANs) and the static-ct-api tiled client
// (ParseDataTile) both reduce an entry to a certificate DER, then call this. crypto/x509
// parses a pre_certificate cleanly: the CT poison extension is critical but
// ParseCertificate records unhandled critical extensions rather than rejecting them
// (rejection is a Verify-time check, which the tail never runs).
func CertSANs(der []byte) ([]string, error) {
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

// readOpaque24 reads one TLS opaque<1..2^24-1> value: a 3-byte big-endian length then
// that many bytes. It errors on a truncated length prefix or a length that overruns the
// buffer, so a malformed entry is rejected rather than read past its bounds.
func readOpaque24(b []byte) ([]byte, error) {
	if len(b) < ctASN1CertLen {
		return nil, fmt.Errorf("truncated length prefix")
	}
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	if n == 0 {
		return nil, fmt.Errorf("zero-length value")
	}
	if len(b) < ctASN1CertLen+n {
		return nil, fmt.Errorf("length %d overruns %d-byte buffer", n, len(b)-ctASN1CertLen)
	}
	return b[ctASN1CertLen : ctASN1CertLen+n], nil
}

// A static-ct-api log (C2SP static-ct-api, #877 §4.3) serves its Merkle tree as static
// files under the monitoring_url, not the RFC 6962 dynamic endpoints. The tail reads two
// of them: `checkpoint` (the signed tree head) and `tile/data/<N>` (up to CTTileWidth
// entries each). There is no get-entries and no get-sth on a tiled log.

// CTTileWidth is the entry count of a full static-ct-api data tile: the spec fixes it at
// exactly 256 entries ("Full tiles MUST be exactly 256 hashes wide"). It is the tiled
// batch cap (§4.4) — the tile granularity is the request size, so a tiled poll reads at
// most 256 entries per fetch by construction. A not-yet-full tail tile is a partial tile,
// served under the `.p/<W>` suffix with 1..255 entries.
const CTTileWidth = 256

// ParseCheckpoint decodes a static-ct-api `checkpoint` body into the same
// CTSignedTreeHead the RFC 6962 path uses. The checkpoint is a C2SP signed-note: an
// origin line, the decimal tree size, the base64 root hash, a blank line, then one or
// more `— <keyname> <base64>` signature lines. The tail reads the tree size (line 2) and
// keeps the whole body verbatim as the durable cursor's signed_head, so a later
// consistency proof reads the head with the exact bytes the log signed (§4.2). A body
// with fewer than the three required lines, or a non-numeric size, is an error the poll
// treats as transient — a malformed 200 is never evidence the log is empty (ADR-0027 §7).
func ParseCheckpoint(body []byte) (CTSignedTreeHead, error) {
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

// DataTilePath renders a tile index into the static-ct-api path encoding: 3-digit,
// zero-padded base-1000 segments, most-significant first, with an `x` prefix on every
// segment but the last. This bounds directory fan-out (a log with billions of entries
// never puts millions of tiles in one directory). Examples: 0 -> "000", 1 -> "001",
// 1234 -> "x001/234", 1234567 -> "x001/x234/567". The caller appends it to
// `<monitoring>/tile/data/` and adds the `.p/<W>` suffix for a partial tile.
func DataTilePath(index int64) string {
	segs := []string{fmt.Sprintf("%03d", index%1000)}
	for index /= 1000; index > 0; index /= 1000 {
		segs = append([]string{fmt.Sprintf("%03d", index%1000)}, segs...)
	}
	for i := 0; i < len(segs)-1; i++ {
		segs[i] = "x" + segs[i]
	}
	return strings.Join(segs, "/")
}

// ParseDataTile decodes one static-ct-api data tile into the certificate DER of each
// entry, in tile order. A data tile is a packed sequence of TileLeaf structures with no
// per-entry length prefix, so it is parsed strictly left to right: each leaf is consumed
// exactly, and the next begins where it ends. A trailing partial leaf, or any field that
// overruns the buffer, is an error the poll treats as transient — never a short, silent
// read (ADR-0027 §7). Each TileLeaf is a RFC 6962 TimestampedEntry (no MerkleTreeLeaf
// version/leaf_type header, unlike get-entries), then for a precert entry the full
// pre_certificate, then the issuance chain as SHA-256 fingerprints (skipped — the tail
// needs only the leaf certificate's names). An entry type the tail does not recognise
// ends the tile parse with an error rather than a guess, because an unknown type has an
// unknown length and the next leaf's offset cannot be found.
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

// parseTileLeaf consumes one TileLeaf from the front of b and returns the leaf
// certificate's DER and the remaining bytes. The leaf layout (RFC 6962 §3.4 +
// static-ct-api): timestamp(8) + entry_type(2) + signed_entry + extensions(opaque16),
// then for a precert entry pre_certificate(opaque24), then certificate_chain(opaque16).
// For an x509 entry the signed_entry IS the certificate; for a precert entry the
// signed_entry holds only issuer_key_hash + TBSCertificate, so the parseable certificate
// is the pre_certificate field read after the extensions.
func parseTileLeaf(b []byte) (der, rest []byte, err error) {
	const timestampLen, entryTypeLen = 8, 2
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
		// signed_entry = PreCert: issuer_key_hash[32] + TBSCertificate (not a parseable
		// certificate). The full pre_certificate comes after the extensions.
		if len(b) < ctIssuerKeyHash {
			return nil, nil, fmt.Errorf("truncated precert issuer_key_hash")
		}
		b = b[ctIssuerKeyHash:]
		_, after, e := takeOpaque24(b) // TBSCertificate — skipped
		if e != nil {
			return nil, nil, fmt.Errorf("precert TBSCertificate: %w", e)
		}
		b = after
	default:
		// An unknown entry type has an unknown signed_entry length, so the rest of the
		// tile cannot be framed. Fail the tile rather than guess an offset.
		return nil, nil, fmt.Errorf("unsupported entry type %d", entryType)
	}

	_, after, e := takeOpaque16(b)
	if e != nil {
		return nil, nil, fmt.Errorf("extensions: %w", e)
	}
	b = after

	if entryType == ctEntryPrecert {
		// pre_certificate = ASN.1Cert: the parseable certificate for a precert entry.
		cert, afterPre, pe := takeOpaque24(b)
		if pe != nil {
			return nil, nil, fmt.Errorf("pre_certificate: %w", pe)
		}
		der = cert
		b = afterPre
	}

	// certificate_chain = Fingerprint chain<0..2^16-1> — skipped; the tail needs no chain.
	_, afterChain, ce := takeOpaque16(b)
	if ce != nil {
		return nil, nil, fmt.Errorf("certificate_chain: %w", ce)
	}
	return der, afterChain, nil
}

// takeOpaque24 reads one TLS opaque<1..2^24-1> value (a 3-byte big-endian length then
// that many bytes) from the front of b and returns the value and the remaining bytes. It
// reuses readOpaque24's bounds checks and, unlike it, hands back the tail so a packed
// sequence of values (a data tile) is parsed left to right.
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

// CTAdmission is one Name the tail admits, with the name-scope Seed its Citation chain
// terminates at (ADR-0027). It is the tail's analogue of a crt.sh admitted name, but the
// Seed is resolved per-name (the tail reads the whole log firehose and keeps only the
// names under some declared scope), where crt.sh queries one Seed at a time.
type CTAdmission struct {
	Name   string
	SeedID int64
}

// AdmitCTNames is the pure admission decision for the tail: the set of Names a batch of
// log entries admits, each under the most specific name-scope Seed that covers it. It
// applies the same two rulings the `ct` Scan does — ADR-0060 (no value with an asterisk
// label admits a Name) and ADR-0047 (the Seed decides which names are inside, so a
// certificate's foreign SANs admit nothing) — and dedupes by name. The tail reads the
// full firehose, so the scope filter is the whole thing: a SAN under no declared Seed is
// discarded, exactly as crt.sh discards a co-tenant's name off a shared certificate.
// Admission stops at MaxAdmittedNames distinct Names (#741), the same ceiling the `ct`
// path caps at, so a hostile or oversized delta cannot mint unbounded admitted_name rows.
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
		// ADR-0060: an asterisk anywhere denotes a set (a wildcard) or has two
		// denotations (a partial wildcard); neither admits a Name.
		if strings.Contains(n, "*") {
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seedID, ok := coveringSeed(n, seeds)
		if !ok {
			continue // ADR-0047: no declared scope covers it, so it is not inside
		}
		seen[n] = struct{}{}
		out = append(out, CTAdmission{Name: n, SeedID: seedID})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// coveringSeed returns the id of the most specific name-scope Seed that covers name —
// the longest Seed domain the name falls under (ADR-0047). A name under both
// `sub.example.com` and `example.com` Seeds is attributed to the more specific one, so
// its Citation chain terminates at the scope that most tightly accounts for it. It
// reports false when no Seed covers the name.
func coveringSeed(name string, seeds []CTSeed) (int64, bool) {
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
