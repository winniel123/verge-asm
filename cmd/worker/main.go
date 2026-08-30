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
	"strings"
	"syscall"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/delivery"
	"github.com/winniel123/verge-asm/internal/env"
	"github.com/winniel123/verge-asm/internal/pgdb"
	"github.com/winniel123/verge-asm/internal/queue"
	"github.com/winniel123/verge-asm/internal/release"
	"github.com/winniel123/verge-asm/internal/remoteexec"
	"github.com/winniel123/verge-asm/internal/report"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/transcript"
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
	proberDir := env.OrDefault("VERGE_PROBER_DIR", "/app/probers")
	stateDir := env.OrDefault("VERGE_STATE_DIR", "/app/state")

	// Provision the instance transcript key on the shared key volume before any
	// job runs, so the writer (#865) can seal captured output at rest. This is the
	// worker's half of the one key web (the reader) also mounts. Fatal if the
	// volume is unwritable — running on without a key would silently drop the
	// corpus (raw-job-output spec §5.3).
	transcriptKeyDir := env.OrDefault("VERGE_TRANSCRIPT_KEY_DIR", "/app/transcript-key")
	if err := transcript.EnsureKey(transcriptKeyDir); err != nil {
		log.Fatalf("worker: transcript key: %v", err)
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	dispatcher := queue.NewDispatcher(pool, time.Now, logger)
	// The off-host measurement router (ADR-0103, #683, P0.8): a provisioned internet
	// Vantage measures from its OWN position — its jobs are pushed to and exec'd on the
	// prober host over SSH, arch-matched by `uname`. The instance ships a prober for each
	// matrix architecture under VERGE_PROBER_DIR (an arm64 instance pushes to an amd64
	// host and vice versa); the own-arch VERGE_PROBER_PATH is the single-binary fallback.
	// A resolver-only vantage's jobs still run on the local prober.
	router := newRemoteProberRouter(
		db.New(pool),
		remoteexec.DirBinaryProvider{Dir: proberDir, Fallback: proberPath},
		stateDir,
		logger,
	)
	// The ct Scan's runner (ADR-0106): a throttled crt.sh fetcher and the
	// instance-wide 5 req/min reservation throttle, wired onto the worker beside
	// the prober. The User-Agent identifies this build, which the source operator
	// asked for (passive-discovery §2.2).
	// The message producer (P0.7): each batch tx folds a flagship/membership
	// transition into a Message and routes it to its bound Channels via
	// delivery.EnqueueForMessage (injected so internal/queue never imports
	// internal/delivery). VERGE_DEV suppresses it entirely — a fixture-only install
	// serves fixtures, never live estate, so it writes no message (AL-25). A default
	// real install binds no Channel, so a Message is written but nothing is POSTed
	// until an admin declares one.
	devMode := isTruthy(env.OrDefault("VERGE_DEV", ""))
	worker := queue.NewWorker(pool, queue.ExecProber{Path: proberPath}, time.Now, logger).
		WithCT(queue.NewHTTPCTFetcher(env.OrDefault("VERGE_VERSION", "dev")), queue.NewCTThrottle(db.New(pool))).
		WithRouter(router).
		WithMessages(delivery.EnqueueForMessage, devMode)

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
	// half. Measurement is dispatched over the connection by the off-host router
	// (#683); this loop only provisions the key material it uses.
	provisionVantageKeys(ctx, db.New(pool), stateDir)

	// Measure the per-vantage connect latency the Dashboard renders (P0.5): for
	// each provisioned prober with a published keypair but no latency yet, dial the
	// prober connect that pins its host key and record the round-trip. Best-effort —
	// an unreachable prober keeps its NULL latency and the Dashboard its em dash.
	measureVantageLatencies(ctx, db.New(pool), sshProber{}, stateDir)

	runSelfTest(ctx, proberPath)

	go func() {
		if err := dispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: dispatcher stopped: %v", err)
		}
	}()

	// On-cadence report dispatch runs beside the queue dispatcher (#502/T3): each
	// tick renders every due schedule's artifact and stamps an in-instance receipt
	// keyed to the tick, idempotent so a second poll in a window is a recorded skip.
	// A schedule bound to a Channel also enqueues one link-only ready-message per won
	// tick (#508/T7); a download-only schedule enqueues nothing. It is a no-op until an
	// admin declares a schedule — no schedule ships, so nothing is cut.
	reportDispatcher := report.NewDispatcher(pool, time.Now, logger)
	go func() {
		if err := reportDispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: report dispatcher stopped: %v", err)
		}
	}()

	// Report notify runs beside the report dispatcher (#508/T7, ADR-0039 stands): it
	// drains pending report notifications and POSTs each bound Channel a LINK-ONLY
	// ready-message — the report name, the run's period, and a session-authed link to
	// the in-instance artifact, never the estate. On a 2xx it flips the receipt to
	// 'delivered'; on dead-letter the receipt stays 'generated' and the artifact stays
	// viewable. It rides the delivery package's shared signed-HTTPS transport and
	// queue.Backoff curve, and is a no-op until a schedule binds a Channel.
	// VERGE_PUBLIC_URL is the absolute base the ready-message's link is built on; empty
	// leaves the link as the bare path.
	notifyRunner := report.NewNotifyRunner(pool, delivery.NewHTTPDoer(), time.Now, env.OrDefault("VERGE_PUBLIC_URL", ""), logger)
	go func() {
		if err := notifyRunner.Run(ctx, 5*time.Second); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: report notify stopped: %v", err)
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

	// Observation retention runs beside it (#208, v1 spec §4.6). It retires only
	// EVIDENTIAL observations — those past their own per-timeline bound AND the
	// operator's dial — and never a live row: the delete evaluates each row's own
	// bound, so a derivation always reads live-tier data. It is a no-op until the
	// operator sets the dial (v1 ships the raw corpus growing without bound).
	obsRetirer := retention.NewObservationRetirer(db.New(pool), time.Now, logger)
	go func() {
		if err := obsRetirer.Run(ctx, 24*time.Hour); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: observation retention stopped: %v", err)
		}
	}()

	// Release check runs beside the retirers (#391, ADR-0124: check + surface +
	// guide, never self-replace). A daily best-effort poll of the upstream release
	// feed, gated on instance_config.update_check_enabled: while the operator has
	// left the check off, it makes NO network call — not on a tick and not on this
	// boot run — so an air-gapped instance stays genuinely silent. When enabled it
	// records a current/newer verdict in the release cache (SetReleaseCache) that
	// the web Version & updates card renders; a failed or unreachable feed is a
	// logged no-op that leaves the cache untouched, never a crash. VERGE_VERSION is
	// the running build compared against the feed's latest; VERGE_RELEASE_FEED_URL
	// overrides the default GitHub latest-release feed (unset ⇒ release.DefaultFeedURL).
	releaseChecker := release.NewChecker(
		db.New(pool),
		release.NewHTTPFetcher(env.OrDefault("VERGE_RELEASE_FEED_URL", release.DefaultFeedURL)),
		env.OrDefault("VERGE_VERSION", "dev"),
		time.Now,
		logger,
	)
	go func() {
		if err := releaseChecker.Run(ctx, 24*time.Hour); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: release check stopped: %v", err)
		}
	}()

	// Channel delivery runs beside the measurement worker (#207, v1 spec §4.5). It
	// drains routed Deliveries off the queue and POSTs each to its Channel, on the
	// same retry/backoff/dead-letter curve the measurement queue uses (queue.Backoff)
	// rather than a second mechanism beside it. It is a no-op on a default install:
	// no Channel ships configured, so nothing is ever routed until an admin declares
	// one. VERGE_PUBLIC_URL is the absolute base each body's link is built on; empty
	// leaves the link off rather than fabricating one.
	deliveryRunner := delivery.NewRunner(pool, delivery.NewHTTPDoer(), time.Now, env.OrDefault("VERGE_PUBLIC_URL", ""), logger)
	go func() {
		if err := deliveryRunner.Run(ctx, 5*time.Second); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: delivery stopped: %v", err)
		}
	}()

	log.Print("worker: started queue dispatch + worker loop")
	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("worker: %v", err)
	}
	log.Print("worker: shutting down")
}

// isTruthy reads the common affirmative spellings of a boolean env value — the
// VERGE_DEV gate the message producer reads (mirrors cmd/web's own).
func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
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
