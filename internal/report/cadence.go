// Package report holds the on-cadence report machinery: the cadence→window
// vocabulary a schedule dispatches on, the cron/preset next-fire evaluator that
// decides *when* a schedule fires, and the Dispatcher that renders each due
// schedule's artifact and stamps its in-instance receipt (#502/T3, #639/T?). It is
// the report-side twin of package queue's measurement dispatch, and shares queue's
// idempotency shape — a fire tick keys one run of a window, and a second poll inside
// the window is a recorded skip rather than a second run.
package report

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CadenceWindow is the period a run of a schedule covers, derived from the
// schedule's stored cadence label: the run cuts the artifact for the window the
// cadence implies (6h / daily / weekly / monthly), defaulting to a week for an
// unrecognised label. It is the single source of truth for that mapping — both the
// Run-now handler (cmd/web) and the on-cadence Dispatcher compute the same window
// from this one function, so a scheduled run and a manual run of the same schedule
// cover the same period.
//
// CadenceWindow is the artifact *period* (how much of the estate a run summarises),
// which is deliberately kept SEPARATE from the *fire instant* (when a run happens) —
// that latter is DispatchTick's job. A schedule fires at its declared clock time
// (ADR-0122) but still summarises the coarse window its cadence names; the two
// concerns are orthogonal and Run-now shares only the window mapping.
func CadenceWindow(cadence string) time.Duration {
	c := strings.ToLower(cadence)
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

// Timezone: this build models no per-instance timezone (nothing in the schema or
// config carries one; every stored instant is UTC — pgtype.Timestamptz, time.Now().
// UTC()). A schedule's declared clock time is therefore interpreted in UTC, and
// DispatchTick computes fire instants in UTC. When an instance timezone is ever
// modelled, this is the one place that resolves the location.

// DispatchTick returns the most recent instant at or before now on which the
// schedule's cadence declares a firing — the "fire tick" — together with ok=false
// when the cadence is uninterpretable (neither a known preset nor a parseable cron).
//
// It replaces the old epoch-floor scheduledTick: rather than flooring now to the
// cadence *duration* from the Unix epoch (which fired "daily · 08:00" at ≈00:00 UTC,
// never at 08:00), it honours the operator's declared clock time — presets to the
// minute and Custom as a real 5-field cron expression (ADR-0122, superseding the
// no-cron clauses of ADR-0118 §1). The idempotency shape is unchanged: the fire tick
// is the value the dispatcher keys (schedule_id, scheduled_tick) on, so two poll
// ticks between one firing and the next resolve to the same tick (a recorded skip),
// and the tick advances only when the clock crosses the next declared firing. Missed
// firings are not caught up — DispatchTick returns the single most-recent firing, so
// a worker that was down over one is not backfilled (currency, not history), exactly
// as the epoch floor and the queue dispatcher behave.
func DispatchTick(now time.Time, cadence string) (time.Time, bool) {
	spec, err := cadenceSpec(cadence)
	if err != nil {
		return time.Time{}, false
	}
	return spec.prevFire(now.UTC())
}

// ValidateCron reports whether expr is a well-formed 5-field cron expression. It is
// the guard the schedule create/edit wizard calls to REFUSE a Custom cadence whose
// cron does not parse (ADR-0122) — an uninterpretable cadence is never persisted and
// never silently coerced to a weekly default. nil means valid.
func ValidateCron(expr string) error {
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
		dow := 1 // default Monday
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
		dom := 1 // default 1st
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

// parseCron parses a 5-field cron expression (minute hour day-of-month month
// day-of-week). It supports "*", single values, comma lists, ranges "a-b", and steps
// "*/n" and "a-b/n" (and "a/n" = a-to-max step n). Day-of-week accepts 0–7 with both
// 0 and 7 meaning Sunday. A malformed expression — wrong field count, out-of-range
// value, or unparseable token — is an error; the create/edit wizard rejects it via
// ValidateCron so it never reaches the dispatcher.
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
	// Day-of-week ranges over 0–7 for validation (both 0 and 7 spell Sunday);
	// normalizeDOW folds 7 onto 0 so the mask only ever carries bits 0–6.
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
			hi = max // "a/n" means a..max step n
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

// prevFire returns the most recent minute at or before now (UTC, truncated to the
// minute) whose calendar day and clock time both satisfy the spec, together with
// ok=false when no firing exists within a year's lookback (a spec that never matches,
// e.g. an impossible day-of-month in every month). It walks calendar days backwards
// (cheap: a daily fire is found on day 0, weekly within 7, monthly within ~31), and
// on each matching day scans that day's minutes back from the cutoff.
func (s cronSpec) prevFire(now time.Time) (time.Time, bool) {
	now = now.UTC().Truncate(time.Minute)
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
