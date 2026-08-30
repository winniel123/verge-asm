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
// public key, the ct_log_cursor primary key), the base URL the RFC 6962 endpoints hang
// off, and a human description for the live event and logs.
type CTLog struct {
	LogID       string
	URL         string
	Description string
}

// logList is the subset of log_list.json the tail reads. Only the RFC 6962 logs
// (operators[].logs[], each carrying a `url`) are decoded here; the static-ct-api
// tiled logs live under operators[].tiled_logs[] and are #877's client (§4.3), so this
// shape does not decode them.
type logList struct {
	Version   string `json:"version"`
	Operators []struct {
		Logs []logListEntry `json:"logs"`
	} `json:"operators"`
}

type logListEntry struct {
	Description      string                     `json:"description"`
	LogID            string                     `json:"log_id"`
	URL              string                     `json:"url"`
	State            map[string]json.RawMessage `json:"state"`
	TemporalInterval *temporalInterval          `json:"temporal_interval"`
}

// temporalInterval is a CT log shard's coverage window: it accepts a certificate whose
// expiry falls in [start_inclusive, end_exclusive) (§4.3).
type temporalInterval struct {
	StartInclusive time.Time `json:"start_inclusive"`
	EndExclusive   time.Time `json:"end_exclusive"`
}

// SelectTailLogs is the pure log-set decision (§4.3): the RFC 6962 logs the tail
// follows at instant now. A log is followed when BOTH hold — its state is `usable` or
// `readonly` (both readable; `retired` may 404 and placeholders are skipped), AND its
// temporal_interval covers now or the near future (the current shard plus the next,
// nextShardHorizon). A log with no temporal_interval is unsharded and always followed.
// The result is sorted by log_id so the fan-out is deterministic.
func SelectTailLogs(now time.Time) ([]CTLog, error) {
	var ll logList
	if err := json.Unmarshal(embeddedLogList, &ll); err != nil {
		return nil, fmt.Errorf("scan: decode ct log list: %w", err)
	}
	var out []CTLog
	for _, op := range ll.Operators {
		for _, e := range op.Logs {
			if !tailReadableState(e.State) {
				continue
			}
			if !tailCoversNow(e.TemporalInterval, now) {
				continue
			}
			if e.LogID == "" || e.URL == "" {
				continue // a placeholder entry carries no usable identity
			}
			out = append(out, CTLog{LogID: e.LogID, URL: e.URL, Description: e.Description})
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

// --- job shape ---------------------------------------------------------------

// CTTailJob is one queue job the tail Scan produces: one log to poll. Unlike the `ct`
// job (one crt.sh query per Seed) the tail fans out per-log — the log is the unit of
// forward-delta work, and the name-scope Seeds it admits under are read on the worker
// side (§4.2). It carries no Vantage: a logged certificate is not a function of where
// we read the log from.
type CTTailJob struct {
	ScanID int64
	Log    CTLog
}

// BuildCTTailJobs fans a tail Scan out into one job per followed log. It produces no
// jobs when the selected log-set is empty — a legible zero-job state, not an error.
// Enablement of the `ct-tail` source is gated by the dispatcher, not here.
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

// ctTailScope is the wire payload a tail job carries: the log to poll. The cursor the
// poll reads and advances lives in ct_log_cursor keyed by LogID, not in the job.
type ctTailScope struct {
	LogID       string `json:"log_id"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// JobSpec renders a CTTailJob into the wire JobSpec the worker reads. Like the `ct` and
// `zone` Scans there is no prober exec — the worker itself polls the log — so the log
// identity travels in the job rather than a vantage and offers.
func (j CTTailJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(ctTailScope{LogID: j.Log.LogID, URL: j.Log.URL, Description: j.Log.Description})
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
	return CTLog{LogID: s.LogID, URL: s.URL, Description: s.Description}, nil
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

// CTLogEntry is one entry of a get-entries answer: the base64-decoded MerkleTreeLeaf
// (leaf_input, which carries the certificate for an x509 entry) and extra_data (the
// certificate chain, and for a precert entry the pre_certificate itself).
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
	ctLeafHeader = 1 + 1 + 8 + 2
	// ctASN1CertLen is the length prefix width of an ASN.1Cert opaque<1..2^24-1>.
	ctASN1CertLen = 3
	// ctIssuerKeyHash is the fixed width of PreCert.issuer_key_hash (a SHA-256).
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

	// crypto/x509 parses a pre_certificate cleanly: the CT poison extension is critical
	// but ParseCertificate records unhandled critical extensions rather than rejecting
	// them (rejection is a Verify-time check, which the tail never runs). So both entry
	// kinds reach the same SAN read.
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

// --- admission ---------------------------------------------------------------

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
