// This file builds the `zone` Scan (v1 spec §3.4, CONTEXT.md `Scan` and
// `Observation`): monthly-shipped, worker-read, with no port list and no vantage
// choice at all. Its scope is the name scopes holding a supplied zone file, and
// its batches restate the file's observations **at the operator's supply
// instant** — never at the worker's read — because re-parsing unchanged bytes on
// a cadence would otherwise manufacture a current observation of a stale fact and
// make staleness invisible instead of not-evaluable. The file's `dns-record`
// timeline is `zone`'s, one timeline per source (the operator's zone file), held
// distinct from the resolver's own `dns-record` timeline covered by the `dns`
// Scan.
package scan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

// ZoneKind is the scan kind this file dispatches. Kept additive to DNSKind so
// the measurement binary and the queue register the two independently.
const ZoneKind = "zone"

// ZoneSource is the source name every zone observation is attributed to: the
// operator's own zone file, `authority: declared` (CONTEXT.md `Source`,
// `Authority`). It is not the resolver, so its `dns-record` values age on the
// zone Scan's re-supply cadence rather than the resolver's daily one.
const ZoneSource = "zone"

// ZoneFile is one supplied zone file for one name-scope Seed: the operator's
// declared ground truth (v1 spec §3.1), its content and the instant of the
// supply act. SuppliedAt is the operator's supply instant — the moment they
// uploaded — and is the instant every observation restated from Content takes.
type ZoneFile struct {
	SeedID     int64
	Domain     string
	SuppliedAt time.Time
	Content    string
}

// ZoneAging describes how a supplied zone file ages against the operator's
// declared re-supply interval. A zone file is dated at its supply instant, and
// the operator promises to re-export on a cadence; past that cadence the file is
// stale, and its restated observations carry a supply instant older than the
// interval promises. A stale zone file widens a coverage gap — the estate as the
// operator declares it is older than the scan cadence, so what the zone would
// otherwise cover is no longer current and is recorded as a Gap rather than a
// clean current fact (v1 spec §3.4). Before the cadence the file is current, and
// Days counts down to the instant it ages into that gap.
type ZoneAging struct {
	Supplied bool
	// Stale reports whether the file has passed its re-supply interval and so has
	// aged into a coverage gap.
	Stale bool
	Days  int
}

// ZoneAgingAt computes how a zone file supplied at suppliedAt ages, measured at
// now against interval — the operator's declared re-supply cadence. A zero
// supply instant means nothing was supplied (nothing to stale); a zero or
// negative interval means there is no cadence to age against, so the file is
// treated as current with no countdown. The gap instant is suppliedAt+interval:
// at or past it the file is stale and Days counts how long it has been in the
// gap; before it the file is current and Days counts up (ceiling) to the gap so
// a still-current file never reads "in 0d".
func ZoneAgingAt(suppliedAt, now time.Time, interval time.Duration) ZoneAging {
	if suppliedAt.IsZero() {
		return ZoneAging{}
	}
	if interval <= 0 {
		return ZoneAging{Supplied: true}
	}
	const day = 24 * time.Hour
	gapAt := suppliedAt.Add(interval)
	if !now.Before(gapAt) {
		return ZoneAging{Supplied: true, Stale: true, Days: int(now.Sub(gapAt) / day)}
	}
	remaining := gapAt.Sub(now)
	days := int((remaining + day - time.Nanosecond) / day) // ceiling
	return ZoneAging{Supplied: true, Days: days}
}

type ZoneJob struct {
	ScanID     int64
	SeedID     int64
	Domain     string
	SuppliedAt time.Time
	Content    string
}

// BuildZoneJobs fans a zone Scan out into one job per supplied zone file. It
// produces no jobs when no name scope holds a file — an aperture over an empty
// scope is a legible state, not an error (CONTEXT.md `Scan`).
func BuildZoneJobs(scanID int64, files []ZoneFile) []ZoneJob {
	if len(files) == 0 {
		return nil
	}
	jobs := make([]ZoneJob, 0, len(files))
	for _, f := range files {
		jobs = append(jobs, ZoneJob{
			ScanID:     scanID,
			SeedID:     f.SeedID,
			Domain:     f.Domain,
			SuppliedAt: f.SuppliedAt,
			Content:    f.Content,
		})
	}
	return jobs
}

// zoneScope is the wire payload a zone job carries: the supply instant travels
// with the content so the worker restates at the operator's instant rather than
// its own read (v1 spec §3.4).
type zoneScope struct {
	Domain     string    `json:"domain"`
	SuppliedAt time.Time `json:"supplied_at"`
	Content    string    `json:"content"`
}

// JobSpec renders a ZoneJob into the wire JobSpec the worker reads. Unlike the
// dns Scan there is no prober exec: the worker itself parses Scope, which is why
// the whole file content and its supply instant travel in the job rather than a
// vantage and a resolver.
func (j ZoneJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(zoneScope{Domain: j.Domain, SuppliedAt: j.SuppliedAt, Content: j.Content})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal zone scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: ZoneKind, Scope: raw}, nil
}

// AttemptedScope is the by-content record of what the zone job set out to cover:
// the domain and the supply instant it restated. It is the completed Batch's
// recorded scope; a zone read has no network step to fail, so it does not
// dead-letter.
func (j ZoneJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(zoneScopeRecord{Domain: j.Domain, SuppliedAt: j.SuppliedAt})
}

type zoneScopeRecord struct {
	Domain     string    `json:"domain"`
	SuppliedAt time.Time `json:"supplied_at"`
}

// ZoneScopeFromSpec decodes a zone job's wire scope back into a ZoneFile, so the
// worker restates from the same domain/instant/content the dispatcher enqueued.
func ZoneScopeFromSpec(scope []byte) (ZoneFile, error) {
	var zs zoneScope
	if err := json.Unmarshal(scope, &zs); err != nil {
		return ZoneFile{}, fmt.Errorf("scan: decode zone scope: %w", err)
	}
	return ZoneFile{Domain: zs.Domain, SuppliedAt: zs.SuppliedAt, Content: zs.Content}, nil
}

// ZoneRecord is one `dns-record` observation restated from a zone file: one
// RRset for one (owner name, qtype), stamped at the supply instant. The
// discriminator on the observation is Qtype; the subject is Name; the source is
// ZoneSource. ObservedAt is the operator's supply instant, carried here so the
// worker never substitutes its read time.
type ZoneRecord struct {
	Name       string
	Qtype      string
	Data       json.RawMessage
	ObservedAt time.Time
}

// RestateZone parses a supplied zone file into its `dns-record` observations,
// every one stamped at the file's supply instant. Records are grouped into one
// RRset per (owner, qtype) — the shape a `dns-record` timeline holds — in
// first-seen order so the output is deterministic. Lines it cannot parse are
// skipped rather than guessed at: a zone file is the operator's own ground
// truth, and inventing a record it does not clearly state would fabricate an
// observation (v1 spec §4.1).
//
// It also SURFACES the skips (#869, raw-job-output spec §1.3): the second return
// is the verbatim text of every line that looked like a record but was dropped —
// an unknown RR type, an empty rdata, an orphan continuation, or an RRset that
// could not marshal. Blank lines, comments and directives carry no record by
// design and are NOT surfaced. The skips are the zone variant's debug artifact:
// they answer "why is this DNS record missing from the estate?" with "we skipped
// it". Skips are returned in first-seen order.
func RestateZone(zf ZoneFile) (records []ZoneRecord, skipped []string) {
	p := &zoneParser{origin: fqdn(zf.Domain)}
	rrsets := map[string]*zoneRRset{}
	var order []string
	for _, line := range logicalLines(zf.Content) {
		rec, res := p.parse(line)
		switch res {
		case parseSkipped:
			skipped = append(skipped, strings.TrimSpace(line))
		case parseRecord:
			key := rec.name + "\x00" + rec.qtype
			set, seen := rrsets[key]
			if !seen {
				set = &zoneRRset{name: rec.name, qtype: rec.qtype}
				rrsets[key] = set
				order = append(order, key)
			}
			set.rrs = append(set.rrs, rec.rdata)
		}
		// parseNone carries no record and is not a skip: a blank, comment or directive.
	}

	out := make([]ZoneRecord, 0, len(order))
	for _, key := range order {
		set := rrsets[key]
		data, err := json.Marshal(zoneValue{RRs: set.rrs})
		if err != nil {
			// The literal "skipped because it could not marshal" case (spec §1.3).
			// json.Marshal of a []string does not fail in practice, so this is a
			// defensive surface, named by the RRset it dropped.
			skipped = append(skipped, set.name+" "+set.qtype)
			continue
		}
		out = append(out, ZoneRecord{
			Name:       set.name,
			Qtype:      set.qtype,
			Data:       data,
			ObservedAt: zf.SuppliedAt,
		})
	}
	return out, skipped
}

// parseResult is the three-way outcome of parsing one logical zone line: a
// well-formed record, nothing at all (a blank, comment or directive — no record
// by design), or a dropped candidate (a line that looked like a record but could
// not be parsed). Only parseSkipped is surfaced to the operator (#869): a blank
// line is not a missing record.
type parseResult int

const (
	parseNone    parseResult = iota // blank, comment, or directive — carries no record
	parseRecord                     // a well-formed record
	parseSkipped                    // looked like a record but was dropped
)

// zoneValue is the value a zone `dns-record` observation carries: the RRset's
// rdata, as strings. It is deliberately the zone file's own words rather than a
// re-resolution, so the timeline reflects what the operator declared.
type zoneValue struct {
	RRs []string `json:"rrs"`
}

type zoneRRset struct {
	name  string
	qtype string
	rrs   []string
}

type parsedRecord struct {
	name  string
	qtype string
	rdata string
}

// zoneParser carries the master-file parsing state that spans lines: the current
// $ORIGIN and the last owner name (a line beginning with whitespace inherits it,
// RFC 1035 §5.1).
type zoneParser struct {
	origin    string
	lastOwner string
}

var recordClasses = map[string]bool{"IN": true, "CH": true, "HS": true, "CS": true}

var knownTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true, "NS": true, "TXT": true,
	"SOA": true, "PTR": true, "SRV": true, "CAA": true, "DNAME": true, "NAPTR": true,
	"SPF": true, "TLSA": true, "SSHFP": true, "DS": true,
}

func (p *zoneParser) parse(line string) (parsedRecord, parseResult) {
	line = stripComment(line)
	if strings.TrimSpace(line) == "" {
		return parsedRecord{}, parseNone
	}

	// Directives carry no record but set state for the lines that follow.
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "$ORIGIN"):
		if fields := strings.Fields(trimmed); len(fields) >= 2 {
			p.origin = fqdn(fields[1])
		}
		return parsedRecord{}, parseNone
	case strings.HasPrefix(trimmed, "$"):
		return parsedRecord{}, parseNone // $TTL, $INCLUDE and friends carry no record
	}

	// A leading blank means the owner is inherited from the previous record.
	ownerInherited := line[0] == ' ' || line[0] == '\t'
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return parsedRecord{}, parseNone
	}

	var owner string
	if ownerInherited {
		if p.lastOwner == "" {
			// A continuation line with no prior owner: a dropped candidate, not a
			// clean no-op — the operator wrote rdata the estate never got.
			return parsedRecord{}, parseSkipped
		}
		owner = p.lastOwner
	} else {
		owner = p.resolveName(fields[0])
		fields = fields[1:]
	}
	p.lastOwner = owner

	// Skip an optional TTL and an optional class, in either order, before the
	// type. TTL is all-digits (optionally with a unit); class is IN/CH/HS/CS.
	for len(fields) > 0 {
		f := fields[0]
		if recordClasses[strings.ToUpper(f)] {
			fields = fields[1:]
			continue
		}
		if isTTL(f) {
			fields = fields[1:]
			continue
		}
		break
	}
	if len(fields) < 2 {
		return parsedRecord{}, parseSkipped
	}
	qtype := strings.ToUpper(fields[0])
	if !knownTypes[qtype] {
		return parsedRecord{}, parseSkipped
	}
	rdata := strings.TrimSpace(strings.Join(fields[1:], " "))
	if rdata == "" {
		return parsedRecord{}, parseSkipped
	}
	return parsedRecord{name: owner, qtype: qtype, rdata: rdata}, parseRecord
}

func (p *zoneParser) resolveName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "@" {
		return trimDot(p.origin)
	}
	if strings.HasSuffix(raw, ".") {
		return trimDot(strings.ToLower(raw))
	}
	if p.origin == "" {
		return strings.ToLower(raw)
	}
	return trimDot(strings.ToLower(raw) + "." + p.origin)
}

// logicalLines splits content into logical records, joining lines held open by
// an unclosed parenthesis (RFC 1035 §5.1's multi-line form, used by SOA). It
// does not attempt full quoting rules — a zone file with a naked ";" inside a
// quoted TXT string is rare and is skipped rather than mis-split.
func logicalLines(content string) []string {
	raw := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var out []string
	var buf strings.Builder
	depth := 0
	flush := func() {
		joined := strings.ReplaceAll(buf.String(), "(", " ")
		joined = strings.ReplaceAll(joined, ")", " ")
		out = append(out, joined)
		buf.Reset()
	}
	for _, line := range raw {
		noComment := stripComment(line)
		depth += strings.Count(noComment, "(") - strings.Count(noComment, ")")
		if depth < 0 {
			depth = 0
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
			buf.WriteString(strings.TrimSpace(line))
		} else {
			buf.WriteString(line)
		}
		if depth == 0 {
			flush()
		}
	}
	if buf.Len() > 0 {
		flush()
	}
	return out
}

func stripComment(line string) string {
	if i := strings.IndexByte(line, ';'); i >= 0 {
		return line[:i]
	}
	return line
}

func isTTL(f string) bool {
	if f == "" {
		return false
	}
	digits := false
	for i, r := range f {
		switch {
		case r >= '0' && r <= '9':
			digits = true
		case i == len(f)-1 && strings.ContainsRune("smhdwSMHDW", r):
			// a trailing unit is allowed only after at least one digit
		default:
			return false
		}
	}
	return digits
}

func fqdn(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	if strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

func trimDot(name string) string { return strings.TrimSuffix(name, ".") }
