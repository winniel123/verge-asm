// The `zone` Scan restates a supplied zone file's observations at the operator's
// supply instant, never at the worker's read (v1 spec §3.4): re-parsing unchanged
// bytes on a cadence would manufacture a current observation of a stale fact.
package scan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

const ZoneKind = "zone"

// A zone value ages on the re-supply cadence, not the resolver's daily one (CONTEXT.md Source).

const ZoneSource = "zone"

type ZoneFile struct {
	SeedID     int64
	Domain     string
	SuppliedAt time.Time
	Content    string
}

// A stale file records a Gap: the declared estate is older than the cadence (v1 spec §3.4).

type ZoneAging struct {
	Supplied bool
	Stale    bool
	Days     int
}

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
	// A still-current file must never read "in 0d", so the countdown rounds up.
	remaining := gapAt.Sub(now)
	days := int((remaining + day - time.Nanosecond) / day)
	return ZoneAging{Supplied: true, Days: days}
}

type ZoneJob struct {
	ScanID     int64
	SeedID     int64
	Domain     string
	SuppliedAt time.Time
	Content    string
}

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

type zoneScope struct {
	Domain     string    `json:"domain"`
	SuppliedAt time.Time `json:"supplied_at"`
	Content    string    `json:"content"`
}

func (j ZoneJob) JobSpec(batch string) (wire.JobSpec, error) {
	// A worker-read Scan runs no prober, so its scope rides in the job, not a vantage (v1 spec §3.4).
	raw, err := json.Marshal(zoneScope{Domain: j.Domain, SuppliedAt: j.SuppliedAt, Content: j.Content})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal zone scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: ZoneKind, Scope: raw}, nil
}

func (j ZoneJob) AttemptedScope() ([]byte, error) {
	// A zone read has no network step to fail, so this Batch never dead-letters (ADR-0005).
	return json.Marshal(zoneScopeRecord{Domain: j.Domain, SuppliedAt: j.SuppliedAt})
}

type zoneScopeRecord struct {
	Domain     string    `json:"domain"`
	SuppliedAt time.Time `json:"supplied_at"`
}

func ZoneScopeFromSpec(scope []byte) (ZoneFile, error) {
	var zs zoneScope
	if err := json.Unmarshal(scope, &zs); err != nil {
		return ZoneFile{}, fmt.Errorf("scan: decode zone scope: %w", err)
	}
	return ZoneFile{Domain: zs.Domain, SuppliedAt: zs.SuppliedAt, Content: zs.Content}, nil
}

type ZoneRecord struct {
	Name       string
	Qtype      string
	Data       json.RawMessage
	ObservedAt time.Time
}

// The second return answers "why is this record missing?" for a dropped line (#869, §1.3).

func RestateZone(zf ZoneFile) (records []ZoneRecord, skipped []string) {
	p := &zoneParser{origin: fqdn(zf.Domain)}
	rrsets := map[string]*zoneRRset{}
	var order []string
	// A zone file is ground truth, so an unclear line is skipped and never guessed (v1 spec §4.1).
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
		// A blank, comment or directive is no missing record, so the absent case is deliberate (#869).
	}

	out := make([]ZoneRecord, 0, len(order))
	for _, key := range order {
		set := rrsets[key]
		data, err := json.Marshal(zoneValue{RRs: set.rrs})
		if err != nil {
			// Marshalling a string slice does not fail, so this arm is the spec's defensive surface (§1.3).
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

type parseResult int

const (
	parseNone parseResult = iota
	parseRecord
	parseSkipped
)

// The rdata is the file's own words, never a re-resolution, so the timeline is what was declared.

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

	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "$ORIGIN"):
		if fields := strings.Fields(trimmed); len(fields) >= 2 {
			p.origin = fqdn(fields[1])
		}
		return parsedRecord{}, parseNone
	case strings.HasPrefix(trimmed, "$"):
		return parsedRecord{}, parseNone
	}

	// A line beginning with whitespace inherits the previous owner name (RFC 1035 §5.1).
	ownerInherited := line[0] == ' ' || line[0] == '\t'
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return parsedRecord{}, parseNone
	}

	var owner string
	if ownerInherited {
		if p.lastOwner == "" {
			// The operator wrote rdata the estate never got: a dropped candidate, not a no-op (#869).
			return parsedRecord{}, parseSkipped
		}
		owner = p.lastOwner
	} else {
		owner = p.resolveName(fields[0])
		fields = fields[1:]
	}
	p.lastOwner = owner

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

func logicalLines(content string) []string {
	// A naked ";" inside a quoted TXT string is rare, so full quoting rules are not attempted.
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
