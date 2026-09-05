// Command worker hosts the Postgres-backed queue and the sweeps that run beside it.
// It has no listener and applies no migrations: the web process owns the schema
// (ADR-0001, ADR-0103).
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
	"github.com/winniel123/verge-asm/internal/scan"
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

	transcriptKeyDir := env.OrDefault("VERGE_TRANSCRIPT_KEY_DIR", "/app/transcript-key")
	// Running on without a key would silently drop the captured corpus (raw-job-output §5.3).
	transcriptKey, err := transcript.LoadOrCreateKey(transcriptKeyDir)
	if err != nil {
		log.Fatalf("worker: transcript key: %v", err)
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)

	// The CT source operator asked that the User-Agent identify this build (passive-discovery §2.2).
	ctVersion := env.OrDefault("VERGE_VERSION", "dev")
	// The CT key is worker-only: the web process never reads it (ct-source-replacement.md §2.4).
	ctFetcher, ctThrottle, ctSource := selectCTSource(env.OrDefault("VERGE_CERTSPOTTER_TOKEN", ""), ctVersion, db.New(pool))

	// The dispatcher's cadence-lag gate and the reaper must never disagree here (#1114).
	staleThreshold := durationOrDefault("VERGE_STALE_JOB_TIMEOUT", queue.DefaultStaleJobThreshold, logger)

	dispatcher := queue.NewDispatcher(pool, time.Now, logger).
		WithCTSource(ctSource.Slug()).
		WithStaleJobThreshold(staleThreshold)
	router := newRemoteProberRouter(
		db.New(pool),
		remoteexec.DirBinaryProvider{Dir: proberDir, Fallback: proberPath},
		stateDir,
		logger,
	)
	// A fixture-only install serves fixtures, never live estate, so it writes no message (ADR-0197 §1).
	devMode := isTruthy(env.OrDefault("VERGE_DEV", ""))
	// A hung prober would block the single-threaded drain loop without this bound (#853).
	probeTimeout := durationOrDefault("VERGE_PROBE_TIMEOUT", queue.DefaultProbeTimeout, logger)
	// The message hook is injected so internal/queue never imports internal/delivery (ADR-0199 §1, #1316).
	worker := queue.NewWorker(pool, queue.ExecProber{Path: proberPath}, time.Now, logger).
		WithCT(ctFetcher, ctThrottle, ctSource).
		WithCTTail(queue.NewHTTPCTFetcher(ctVersion)).
		WithCTVerify(queue.NewHTTPCTFetcher(ctVersion)).
		WithRouter(router).
		WithMessages(delivery.EnqueueForMessage, devMode).
		WithTranscripts(transcriptKey, devMode).
		WithProbeTimeout(probeTimeout)

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

	provisionVantageKeys(ctx, db.New(pool), stateDir)

	measureVantageLatencies(ctx, db.New(pool), sshProber{}, stateDir)

	runSelfTest(ctx, proberPath)

	go func() {
		if err := dispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: dispatcher stopped: %v", err)
		}
	}()

	// Every runner below stays idle until an operator configures it, except the transcript dial.
	reportDispatcher := report.NewDispatcher(pool, time.Now, logger)
	go func() {
		if err := reportDispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: report dispatcher stopped: %v", err)
		}
	}()

	notifyRunner := report.NewNotifyRunner(pool, delivery.NewHTTPDoer(), time.Now, env.OrDefault("VERGE_PUBLIC_URL", ""), logger)
	go func() {
		if err := notifyRunner.Run(ctx, 5*time.Second); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: report notify stopped: %v", err)
		}
	}()

	retirer := retention.NewRetirer(db.New(pool), time.Now, logger)
	go func() {
		if err := retirer.Run(ctx, 24*time.Hour); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: retention stopped: %v", err)
		}
	}()

	obsRetirer := retention.NewObservationRetirer(db.New(pool), time.Now, logger)
	go func() {
		if err := obsRetirer.Run(ctx, 24*time.Hour); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: observation retention stopped: %v", err)
		}
	}()

	// This dial alone ships bounded, at 14 days, so it sweeps on a fresh install (ADR-0126).
	transcriptRetirer := retention.NewTranscriptRetirer(db.New(pool), time.Now, logger)
	go func() {
		if err := transcriptRetirer.Run(ctx, 24*time.Hour); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: transcript retention stopped: %v", err)
		}
	}()

	// No network call until the operator enables it: an air-gapped instance is silent (ADR-0124).
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

	// Warn rather than silently override the operator's value: an inverted pair is theirs to fix.
	if staleThreshold > 0 && probeTimeout > 0 && staleThreshold <= probeTimeout {
		logger.Printf("worker: VERGE_STALE_JOB_TIMEOUT (%s) is at or below VERGE_PROBE_TIMEOUT (%s); "+
			"the reaper may reclaim a job whose probe is still running — set the stale timeout above the probe timeout",
			staleThreshold, probeTimeout)
	}
	reaper := queue.NewReaper(db.New(pool), staleThreshold, time.Now, logger)
	go func() {
		if err := reaper.Run(ctx, 1*time.Minute); err != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("worker: reaper stopped: %v", err)
		}
	}()

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

func selectCTSource(token, version string, q *db.Queries) (queue.CTFetcher, queue.CTThrottle, scan.CTSource) {
	// Cert Spotter's authenticated tier is what clears the consent bar (ADR-0003).
	if token != "" {
		return queue.NewCertSpotterFetcher(version, token), queue.NewCertSpotterThrottle(q), scan.CertSpotterCTSource()
	}
	// Selection is config-time by key presence, so there is no runtime failover (spec §2.3).
	return queue.NewHTTPCTFetcher(version), queue.NewCTThrottle(q), scan.CrtshCTSource()
}

func durationOrDefault(key string, def time.Duration, logger *log.Logger) time.Duration {
	v := strings.TrimSpace(env.OrDefault(key, ""))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		logger.Printf("worker: %s=%q is not a duration (%v); using default %s", key, v, err, def)
		return def
	}
	return d
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

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
