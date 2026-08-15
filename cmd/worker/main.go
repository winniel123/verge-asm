// Command worker has no listener. It applies no migrations (web owns that),
// runs the local prober self-test once at startup, and hosts the
// Postgres-backed queue: the Dispatcher fans each Scan out on its cadence and
// the Worker claims jobs and commits each Batch with its Observations.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/env"
	"github.com/winniel123/verge-asm/internal/pgdb"
	"github.com/winniel123/verge-asm/internal/queue"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/wire"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check that the running instance is healthy, then exit")
	trigger := flag.String("trigger", "", "manually dispatch a Scan by kind (e.g. dns), drain it, then exit")
	flag.Parse()

	databaseURL, err := env.Require("DATABASE_URL")
	if err != nil {
		log.Fatalf("worker: %v", err)
	}

	if *healthcheck {
		if err := checkHealth(databaseURL); err != nil {
			log.Fatalf("worker: healthcheck: %v", err)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgdb.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("worker: connect: %v", err)
	}
	defer pool.Close()

	proberPath := env.OrDefault("VERGE_PROBER_PATH", "/app/prober")
	logger := log.New(os.Stderr, "", log.LstdFlags)
	dispatcher := queue.NewDispatcher(pool, time.Now, logger)
	worker := queue.NewWorker(pool, queue.ExecProber{Path: proberPath}, time.Now, logger)

	// A manual run dispatches an existing Scan, drains it synchronously, and
	// exits — the operator/CI path that produces Observation rows on demand.
	if *trigger != "" {
		n, err := dispatcher.Trigger(ctx, *trigger)
		if err != nil {
			log.Fatalf("worker: trigger %s: %v", *trigger, err)
		}
		log.Printf("worker: triggered %s, %d job(s) enqueued", *trigger, n)
		if err := worker.Drain(ctx); err != nil {
			log.Fatalf("worker: drain: %v", err)
		}
		log.Print("worker: trigger drained")
		return
	}

	// Generate SSH keypairs for any newly provisioned vantages, keeping the
	// private half on this worker-only volume and publishing only the public
	// half. No measurement is dispatched over the connection yet (#8, #14).
	provisionVantageKeys(ctx, db.New(pool), env.OrDefault("VERGE_STATE_DIR", "/app/state"))

	runSelfTest(ctx, proberPath)

	go func() {
		if err := dispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: dispatcher stopped: %v", err)
		}
	}()

	// Dispatch retention runs beside the dispatcher: expired Dispatch rows are
	// the one corpus a wall clock may retire (v1 spec §4.6). It is a structurally
	// separate path that never touches Observation or Span data, and a no-op
	// until the operator sets the dial — v1 ships Dispatch unbounded. A daily
	// sweep is ample for a corpus of one row per firing.
	retirer := retention.NewRetirer(db.New(pool), time.Now, logger)
	go func() {
		if err := retirer.Run(ctx, 24*time.Hour); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: retention stopped: %v", err)
		}
	}()

	log.Print("worker: started queue dispatch + worker loop")
	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("worker: %v", err)
	}
	log.Print("worker: shutting down")
}

// runSelfTest execs the prober once at startup to prove the job-spec-in,
// NDJSON-out contract works inside this deployment. Failure is logged, not
// fatal: the worker has no queue work yet for this to gate.
func runSelfTest(ctx context.Context, proberPath string) {
	obs, err := execProbe(ctx, proberPath, wire.JobSpec{Batch: "startup-selftest", Kind: "noop"})
	if err != nil {
		log.Printf("worker: prober self-test failed: %v", err)
		return
	}
	log.Printf("worker: prober self-test ok: %+v", obs)
}

func checkHealth(databaseURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgdb.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return nil
}
