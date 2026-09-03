// Package report holds the on-cadence report machinery: the cadence-to-window
// vocabulary, the cron and preset next-fire evaluator, and the dispatch that cuts
// each due schedule's artifact and stamps its receipt (#502, ADR-0122, ADR-0039).
package report

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func CadenceWindow(cadence string) time.Duration {
	// One source for the mapping, so a manual Run-now and a scheduled run cover the same period.
	c := strings.ToLower(cadence)
	// The artifact period only; the fire instant is a separate concern (ADR-0122).
	switch {
	case strings.Contains(c, "6h"):
		return 6 * time.Hour
	case strings.Contains(c, "daily"):
		return 24 * time.Hour
	case strings.Contains(c, "monthly"):
		return 30 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

func DispatchTick(now time.Time, cadence string) (time.Time, bool) {
	// The preset and cron reading is ADR-0122, which supersedes ADR-0118 §1's no-cron clauses.
	spec, err := cadenceSpec(cadence)
	if err != nil {
		return time.Time{}, false
	}
	// The dispatcher keys on this tick, so two polls between firings must resolve to one value.
	// A missed firing is not backfilled: currency, not history.
	return spec.prevFire(now.UTC())
}

func ValidateCron(expr string) error {
	// The create/edit wizard refuses here rather than coercing to a weekly default (ADR-0122).
	_, err := parseCron(expr)
	return err
}

func cadenceSpec(cadence string) (cronSpec, error) {
	if cron, ok := presetToCron(cadence); ok {
		return parseCron(cron)
	}
	return parseCron(strings.TrimSpace(cadence))
}

var (
	reClock   = regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	reOrdinal = regexp.MustCompile(`(\d{1,2})(?:st|nd|rd|th)`)
	weekdays  = map[string]int{
		"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
	}
)

func presetToCron(cadence string) (string, bool) {
	c := strings.ToLower(strings.TrimSpace(cadence))
	hh, mm, hasClock := parseClock(c)
	switch {
	case strings.Contains(c, "6h"):
		return "0 */6 * * *", true
	case strings.Contains(c, "daily"):
		if !hasClock {
			hh, mm = 0, 0
		}
		return fmt.Sprintf("%d %d * * *", mm, hh), true
	case strings.Contains(c, "weekly"):
		if !hasClock {
			hh, mm = 0, 0
		}
		dow := 1
		for name, n := range weekdays {
			if strings.Contains(c, name) {
				dow = n
				break
			}
		}
		return fmt.Sprintf("%d %d * * %d", mm, hh, dow), true
	case strings.Contains(c, "monthly"):
		if !hasClock {
			hh, mm = 0, 0
		}
		dom := 1
		if m := reOrdinal.FindStringSubmatch(c); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n >= 1 && n <= 31 {
				dom = n
			}
		}
		return fmt.Sprintf("%d %d %d * *", mm, hh, dom), true
	}
	return "", false
}

func parseClock(c string) (hh, mm int, hasClock bool) {
	m := reClock.FindStringSubmatch(c)
	if m == nil {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(m[1])
	n, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil || h > 23 || n > 59 {
		return 0, 0, false
	}
	return h, n, true
}

type cronSpec struct {
	min, hour, dom, month, dow uint64
	domRestricted              bool
	dowRestricted              bool
}

func parseCron(expr string) (cronSpec, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return cronSpec{}, fmt.Errorf("cron: want 5 fields, got %d in %q", len(fields), expr)
	}
	var s cronSpec
	var err error
	if s.min, _, err = parseField(fields[0], 0, 59, nil); err != nil {
		return cronSpec{}, fmt.Errorf("cron minute: %w", err)
	}
	if s.hour, _, err = parseField(fields[1], 0, 23, nil); err != nil {
		return cronSpec{}, fmt.Errorf("cron hour: %w", err)
	}
	if s.dom, s.domRestricted, err = parseField(fields[2], 1, 31, nil); err != nil {
		return cronSpec{}, fmt.Errorf("cron day-of-month: %w", err)
	}
	if s.month, _, err = parseField(fields[3], 1, 12, nil); err != nil {
		return cronSpec{}, fmt.Errorf("cron month: %w", err)
	}
	// Cron spells Sunday as both 0 and 7, so the mask folds 7 onto 0.
	if s.dow, s.dowRestricted, err = parseField(fields[4], 0, 7, normalizeDOW); err != nil {
		return cronSpec{}, fmt.Errorf("cron day-of-week: %w", err)
	}
	return s, nil
}

func normalizeDOW(n int) int {
	if n == 7 {
		return 0
	}
	return n
}

func parseField(field string, min, max int, norm func(int) int) (mask uint64, restricted bool, err error) {
	if field == "*" {
		return rangeMask(min, max), false, nil
	}
	for _, part := range strings.Split(field, ",") {
		lo, hi, step, perr := parsePart(part, min, max)
		if perr != nil {
			return 0, false, perr
		}
		for v := lo; v <= hi; v += step {
			n := v
			if norm != nil {
				n = norm(v)
			}
			mask |= 1 << uint(n)
		}
	}
	return mask, true, nil
}

func parsePart(part string, min, max int) (lo, hi, step int, err error) {
	step = 1
	body := part
	if slash := strings.IndexByte(part, '/'); slash >= 0 {
		body = part[:slash]
		step, err = strconv.Atoi(part[slash+1:])
		if err != nil || step <= 0 {
			return 0, 0, 0, fmt.Errorf("bad step in %q", part)
		}
	}
	switch {
	case body == "*":
		lo, hi = min, max
	case strings.IndexByte(body, '-') >= 0:
		dash := strings.IndexByte(body, '-')
		lo, err = strconv.Atoi(body[:dash])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("bad range start in %q", part)
		}
		hi, err = strconv.Atoi(body[dash+1:])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("bad range end in %q", part)
		}
	default:
		lo, err = strconv.Atoi(body)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("bad value in %q", part)
		}
		if strings.IndexByte(part, '/') >= 0 {
			// Cron reads "a/n" as a-to-max with step n, so the range is open-ended.
			hi = max
		} else {
			hi = lo
		}
	}
	if lo < min || hi > max || lo > hi {
		return 0, 0, 0, fmt.Errorf("value out of range [%d,%d] in %q", min, max, part)
	}
	return lo, hi, step, nil
}

func rangeMask(lo, hi int) uint64 {
	var m uint64
	for v := lo; v <= hi; v++ {
		m |= 1 << uint(v)
	}
	return m
}

func (s cronSpec) dayMatches(t time.Time) bool {
	if s.month&(1<<uint(int(t.Month()))) == 0 {
		return false
	}
	domHit := s.dom&(1<<uint(t.Day())) != 0
	dowHit := s.dow&(1<<uint(int(t.Weekday()))) != 0
	switch {
	case s.domRestricted && s.dowRestricted:
		return domHit || dowHit
	case s.domRestricted:
		return domHit
	case s.dowRestricted:
		return dowHit
	default:
		return true
	}
}

func (s cronSpec) prevFire(now time.Time) (time.Time, bool) {
	// No per-instance timezone is modelled, so every declared clock time is UTC.
	now = now.UTC().Truncate(time.Minute)
	// 400 days covers every matchable calendar pattern, so a spec that never matches terminates.
	for d := 0; d <= 400; d++ {
		day := now.AddDate(0, 0, -d)
		if !s.dayMatches(day) {
			continue
		}
		maxMin := 24*60 - 1
		if d == 0 {
			maxMin = now.Hour()*60 + now.Minute()
		}
		for m := maxMin; m >= 0; m-- {
			hh, mm := m/60, m%60
			if s.hour&(1<<uint(hh)) != 0 && s.min&(1<<uint(mm)) != 0 {
				return time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, time.UTC), true
			}
		}
	}
	return time.Time{}, false
}
