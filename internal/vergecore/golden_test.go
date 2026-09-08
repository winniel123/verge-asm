package vergecore

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	sensitiveNote = "../../docs/research/sensitive-ports.md"
	probingNote   = "../../docs/research/safe-active-probing.md"
)

var (
	pairCell    = regexp.MustCompile(`^(\d+)/(tcp|udp)$`)
	removedPair = regexp.MustCompile("`(\\d+)/(tcp|udp)` is \\*\\*removed\\*\\*")
	struck      = regexp.MustCompile(`~~[^~]*~~`)
	portOrRange = regexp.MustCompile(`(\d+)(?:[–-](\d+))?`)
)

func readSection(t *testing.T, path, from, to string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	body := string(raw)
	i := strings.Index(body, "\n"+from)
	if i < 0 {
		t.Fatalf("%s: heading %q not found", path, from)
	}
	body = body[i+1:]
	if j := strings.Index(body[len(from):], "\n"+to); j >= 0 {
		body = body[:len(from)+j+1]
	}
	return body
}

func sensitiveListFromNote(t *testing.T) map[Pair]struct{} {
	t.Helper()
	section := readSection(t, sensitiveNote, "## 3. The list", "### 3.4 ")
	set := map[Pair]struct{}{}
	for _, line := range strings.Split(section, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 3 {
			continue
		}
		first := strings.TrimSpace(cells[1])
		m := pairCell.FindStringSubmatch(first)
		if m == nil {
			continue
		}
		set[mustPair(t, m[1], m[2])] = struct{}{}
	}
	for _, m := range removedPair.FindAllStringSubmatch(section, -1) {
		delete(set, mustPair(t, m[1], m[2]))
	}
	return set
}

func frequencyHalfFromNote(t *testing.T) map[Pair]struct{} {
	t.Helper()
	top := readSection(t, probingNote, "### 2.1 ", "### 2.2 ")
	set := map[Pair]struct{}{}
	inFence, found := false, false
	var fence []string
	for _, line := range strings.Split(top, "\n") {
		if strings.HasPrefix(line, "```") {
			if inFence {
				if len(fence) > 0 && found {
					break
				}
				fence = nil
			}
			inFence = !inFence
			continue
		}
		if !inFence {
			continue
		}
		if strings.Trim(line, "0123456789, ") != "" {
			fence = nil
			found = false
			continue
		}
		fence = append(fence, line)
		found = true
	}
	if len(fence) == 0 {
		t.Fatalf("%s §2.1: no fenced block of bare port numbers", probingNote)
	}
	addPorts(t, set, strings.Join(fence, ","))
	if len(set) != 100 {
		t.Fatalf("%s §2.1: top-100 block holds %d ports", probingNote, len(set))
	}

	sel := readSection(t, probingNote, "### 2.3 ", "### 2.4 ")
	if cut := strings.Index(sel, "\n> **Amended"); cut >= 0 {
		sel = sel[:cut]
	}
	dropRe := regexp.MustCompile(`\(drop ([^)]*)\)`)
	m := dropRe.FindStringSubmatch(strings.ReplaceAll(sel, "\n", " "))
	if m == nil {
		t.Fatalf("%s §2.3: no \"(drop ...)\" list", probingNote)
	}
	drop := map[Pair]struct{}{}
	addPorts(t, drop, m[1])
	for p := range drop {
		delete(set, p)
	}

	var items []string
	for _, line := range strings.Split(sel, "\n") {
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)
		if indent >= 2 && strings.HasPrefix(trimmed, "- ") {
			label, list, _ := strings.Cut(trimmed, ":")
			if list == "" {
				t.Fatalf("%s §2.3: supplement bullet %q has no port list", probingNote, label)
			}
			items = append(items, list)
			continue
		}
		if len(items) > 0 && indent >= 4 && trimmed != "" {
			items[len(items)-1] += " " + trimmed
		}
	}
	if len(items) == 0 {
		t.Fatalf("%s §2.3: no supplement bullets", probingNote)
	}
	for _, it := range items {
		addPorts(t, set, struck.ReplaceAllString(it, ""))
	}
	return set
}

func addPorts(t *testing.T, set map[Pair]struct{}, list string) {
	t.Helper()
	for _, m := range portOrRange.FindAllStringSubmatch(list, -1) {
		lo, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatal(err)
		}
		hi := lo
		if m[2] != "" {
			if hi, err = strconv.Atoi(m[2]); err != nil {
				t.Fatal(err)
			}
		}
		for p := lo; p <= hi; p++ {
			set[Pair{Port: uint16(p), Transport: TCP}] = struct{}{}
		}
	}
}

func mustPair(t *testing.T, port, tr string) Pair {
	t.Helper()
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		t.Fatalf("bad port %q: %v", port, err)
	}
	return Pair{Port: uint16(n), Transport: Transport(tr)}
}

func diffSets(want, got map[Pair]struct{}) (missing, extra []Pair) {
	for p := range want {
		if _, ok := got[p]; !ok {
			missing = append(missing, p)
		}
	}
	for p := range got {
		if _, ok := want[p]; !ok {
			extra = append(extra, p)
		}
	}
	return sortedSet(toSet(missing)), sortedSet(toSet(extra))
}

func toSet(ps []Pair) map[Pair]struct{} {
	s := make(map[Pair]struct{}, len(ps))
	for _, p := range ps {
		s[p] = struct{}{}
	}
	return s
}

func TestSensitiveHalfIsResearchNoteSection3(t *testing.T) {
	want := sensitiveListFromNote(t)
	if len(want) != 38 {
		t.Fatalf("note §3 parsed to %d pairs, want 38 (ADR-0067)", len(want))
	}
	got := toSet(Default().SensitivePairs())
	missing, extra := diffSets(want, got)
	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("sensitive half drifted from %s §3\n  listed but absent: %v\n  shipped but unlisted: %v", sensitiveNote, missing, extra)
	}
}

func TestFrequencyHalfIsProbingNoteSection2(t *testing.T) {
	want := frequencyHalfFromNote(t)
	if len(want) != 123 {
		t.Fatalf("note §2.1+§2.3 parsed to %d ports, want 123 (ADR-0009)", len(want))
	}
	got := toSet(Default().FrequencyPairs())
	missing, extra := diffSets(want, got)
	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("frequency half drifted from %s §2.3\n  specified but absent: %v\n  shipped but unspecified: %v", probingNote, missing, extra)
	}
}
