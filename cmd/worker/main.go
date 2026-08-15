// Command worker has no listener. It applies no migrations (web owns
// that), runs the local prober self-test once at startup, and will host
// the Postgres-backed queue dispatch loop a later ticket adds.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/env"
	"github.com/winniel123/verge-asm/internal/pgdb"
	"github.com/winniel123/verge-asm/internal/wire"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check that the running instance is healthy, then exit")
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

	runSelfTest(ctx, env.OrDefault("VERGE_PROBER_PATH", "/app/prober"))

	// Generate SSH keypairs for any newly provisioned vantages, keeping the
	// private half on this worker-only volume and publishing only the public
	// half. No measurement is dispatched over the connection yet (#8, #14).
	provisionVantageKeys(ctx, db.New(pool), env.OrDefault("VERGE_STATE_DIR", "/app/state"))

	log.Print("worker: started, waiting for shutdown")
	<-ctx.Done()
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
