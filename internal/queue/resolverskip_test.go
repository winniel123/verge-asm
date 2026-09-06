package queue

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/scan"
)

func dnsJobsFor(vantages []scan.Vantage, resolvers map[int64]string) []scan.Job {
	jobs := scan.BuildDNSJobs(7, []string{"example.com"}, vantages)
	for i := range jobs {
		jobs[i] = jobs[i].WithResolver(resolvers[jobs[i].VantageID])
	}
	return jobs
}

func TestDispatchDNSJobsSkipsAResolverlessVantageAndDispatchesTheRest(t *testing.T) {
	jobs := dnsJobsFor(
		[]scan.Vantage{{ID: 1, Name: "local"}, {ID: 2, Name: "scanner@probe-1"}, {ID: 3, Name: "scanner@probe-2"}},
		map[int64]string{1: "127.0.0.11:53", 3: "9.9.9.9:53"},
	)

	var logged bytes.Buffer
	var enqueued []string
	n, err := dispatchDNSJobs(jobs, log.New(&logged, "", 0), func(j scan.Job) error {
		if _, err := j.JobSpec("batch"); err != nil {
			return err
		}
		enqueued = append(enqueued, j.Vantage)
		return nil
	})
	if err != nil {
		t.Fatalf("one unconfigured vantage aborted the tick: %v", err)
	}
	if n != 2 {
		t.Errorf("enqueued = %d, want the 2 configured vantages", n)
	}
	if strings.Join(enqueued, ",") != "local,scanner@probe-2" {
		t.Errorf("enqueued %v, want the two vantages that carry a resolver", enqueued)
	}
	if !strings.Contains(logged.String(), "scanner@probe-1") {
		t.Errorf("the skip was not recorded: %q", logged.String())
	}
	if !strings.Contains(logged.String(), "1 vantage(s) with no resolver") {
		t.Errorf("the skip count was not recorded: %q", logged.String())
	}
}

func TestDispatchDNSJobsStillFailsOnARealError(t *testing.T) {
	jobs := dnsJobsFor([]scan.Vantage{{ID: 1, Name: "local"}}, map[int64]string{1: "127.0.0.11:53"})
	boom := errors.New("insert failed")
	if _, err := dispatchDNSJobs(jobs, log.New(&bytes.Buffer{}, "", 0), func(scan.Job) error {
		return boom
	}); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the enqueue failure to travel up", err)
	}
}

func TestDispatchDNSJobsEnqueuesNothingWhenNoVantageHasAResolver(t *testing.T) {
	jobs := dnsJobsFor([]scan.Vantage{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}, nil)
	var logged bytes.Buffer
	n, err := dispatchDNSJobs(jobs, log.New(&logged, "", 0), func(j scan.Job) error {
		_, err := j.JobSpec("batch")
		return err
	})
	if err != nil {
		t.Fatalf("a fully unconfigured fleet aborted the tick: %v", err)
	}
	if n != 0 {
		t.Errorf("enqueued = %d, want 0", n)
	}
	if !strings.Contains(logged.String(), "2 vantage(s) with no resolver") {
		t.Errorf("both skips were not recorded: %q", logged.String())
	}
}
