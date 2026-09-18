package migrations

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var migrationFileName = regexp.MustCompile(`^(\d{5})_[a-z0-9_]+\.sql$`)

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err,
			strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

func migrationVersions(t *testing.T) (map[int]string, []string) {
	t.Helper()
	versions := map[int]string{}
	var faults []string
	for _, name := range migrationNames(t) {
		m := migrationFileName.FindStringSubmatch(name)
		if m == nil {
			// upMigrations orders the corpus by filename, so a number of another width
			// concatenates out of migration order and every schema assertion reads it wrong.
			faults = append(faults, name+" does not read as NNNNN_lower_snake_case.sql")
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			faults = append(faults, fmt.Sprintf("%s carries an unreadable version: %v", name, err))
			continue
		}
		if first, taken := versions[n]; taken {
			faults = append(faults, fmt.Sprintf("%s and %s both claim goose version %d",
				first, name, n))
			continue
		}
		versions[n] = name
	}
	return versions, faults
}

func TestNoTwoMigrationsClaimOneGooseVersion(t *testing.T) {
	versions, faults := migrationVersions(t)
	if len(faults) > 0 {
		// goose refuses a duplicate version and the web binary exits on the refusal, so the
		// compose job fails at `wait for a healthy stack` rather than here (#2261).
		t.Errorf("the migration corpus does not number itself: %s", strings.Join(faults, "; "))
	}
	if len(versions) == 0 {
		t.Fatal("no migration carries a readable goose version, so this gate proves nothing")
	}
}

// A test that reads no git cannot tell a branch's own migration from one that merged.

func mergeBase(t *testing.T) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		return "", "git is not on PATH, so the floor this gate measures against cannot be read"
	}
	if _, err := gitOutput("rev-parse", "--git-dir"); err != nil {
		return "", "this tree is no git checkout, so the floor this gate measures against is absent"
	}
	for _, ref := range []string{"refs/remotes/origin/main", "refs/heads/main"} {
		if _, err := gitOutput("rev-parse", "--verify", "--quiet", ref); err != nil {
			continue
		}
		base, err := gitOutput("merge-base", "HEAD", ref)
		if err != nil {
			continue
		}
		return base, ""
	}
	return "", "no main ref is reachable here, and a CI checkout at fetch-depth 1 carries none"
}

func TestANewMigrationNumbersAboveTheMergeBase(t *testing.T) {
	base, absent := mergeBase(t)
	if base == "" {
		// The gate still binds on a developer machine, which is where the number is chosen.
		t.Skipf("%s; run this from a full clone to gate the number", absent)
	}

	listing, err := gitOutput("ls-tree", "--name-only", base, ".")
	if err != nil {
		t.Skipf("cannot read %s, so the floor this gate measures against is unknown: %v", base, err)
	}

	merged := map[int]bool{}
	ceiling := 0
	for _, name := range strings.Split(listing, "\n") {
		m := migrationFileName.FindStringSubmatch(strings.TrimSpace(name))
		if m == nil {
			continue
		}
		n, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			continue
		}
		merged[n] = true
		if n > ceiling {
			ceiling = n
		}
	}
	if ceiling == 0 {
		t.Fatalf("%s carries no numbered migration, so this gate has no floor to measure "+
			"against; the merge base is wrong, not the branch", base)
	}

	var stale []string
	versions, _ := migrationVersions(t)
	for n, name := range versions {
		if !merged[n] && n <= ceiling {
			stale = append(stale, fmt.Sprintf("%s claims %d", name, n))
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("a migration this branch adds numbers at or below %d, the highest already at "+
			"the merge base %s: %s. Renumber above %d. The number was read from a stale fetch, "+
			"and the collision would otherwise surface as a goose duplicate-version refusal that "+
			"fails the compose job. Two branches that each number above this base still collide "+
			"with each other, which no check here can see.",
			ceiling, base[:8], strings.Join(stale, ", "), ceiling)
	}
}
