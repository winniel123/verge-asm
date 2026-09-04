// Command web is the only listener in the deployment (ADR-0001; v1-spec §4.2).
// It serves the operator UI and applies database migrations on startup.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/winniel123/verge-asm/db/migrations"
	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/env"
	"github.com/winniel123/verge-asm/internal/pgdb"
	"github.com/winniel123/verge-asm/internal/queue"
	"github.com/winniel123/verge-asm/internal/transcript"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check that the running instance is healthy, then exit")
	seedFixtures := flag.String("seed-fixtures", "", "dev-only: reseed the open-span corpus from the given design fixtures file, then exit (requires VERGE_DEV)")
	flag.Parse()

	listenAddr := env.OrDefault("VERGE_LISTEN_ADDR", ":8080")

	if *healthcheck {
		if err := checkHealth(listenAddr); err != nil {
			log.Fatalf("web: healthcheck: %v", err)
		}
		return
	}

	// A reseed deletes the whole span corpus, so it may never fire against a real estate (#525).
	if *seedFixtures != "" && !isTruthy(env.OrDefault("VERGE_DEV", "")) {
		log.Fatalf("web: -seed-fixtures is dev-only; set VERGE_DEV=1 to allow it")
	}

	databaseURL, err := env.Require("DATABASE_URL")
	if err != nil {
		log.Fatalf("web: %v", err)
	}

	if err := migrateUp(databaseURL); err != nil {
		log.Fatalf("web: migrate: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgdb.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("web: connect: %v", err)
	}
	defer pool.Close()

	if *seedFixtures != "" {
		if err := seedInventoryFixtures(ctx, pool, *seedFixtures); err != nil {
			log.Fatalf("web: seed-fixtures: %v", err)
		}
		if err := seedDevOperator(ctx, pool); err != nil {
			log.Fatalf("web: seed-fixtures: %v", err)
		}
		if err := seedDevFixtureAccounts(ctx, pool); err != nil {
			log.Fatalf("web: seed-fixtures: %v", err)
		}
		if err := seedProfileFixtures(ctx, pool); err != nil {
			log.Fatalf("web: seed-fixtures: %v", err)
		}
		if err := seedSigninFixtures(ctx, pool); err != nil {
			log.Fatalf("web: seed-fixtures: %v", err)
		}
		log.Printf("web: seeded inventory fixtures from %s (%d open spans); dev operator %q + %d fixture accounts + Profile + SignIn fixtures ready", *seedFixtures, len(inventoryFixtureSpans), devSeedUsername, len(devFixtureAccounts))
		return
	}

	queries := db.New(pool)

	stateDir := env.OrDefault("VERGE_STATE_DIR", "/app/state")
	key, err := auth.LoadOrCreateKey(stateDir)
	if err != nil {
		log.Fatalf("web: session key: %v", err)
	}

	transcriptKeyDir := env.OrDefault("VERGE_TRANSCRIPT_KEY_DIR", "/app/transcript-key")
	transcriptKey, err := transcript.LoadOrCreateKey(transcriptKeyDir)
	if err != nil {
		log.Fatalf("web: transcript key: %v", err)
	}

	setupToken, err := bootstrapSetupToken(ctx, queries)
	if err != nil {
		log.Fatalf("web: setup token: %v", err)
	}

	// A pinned instant makes relative-time renders deterministic; a deployment never sets it.
	devMode := isTruthy(env.OrDefault("VERGE_DEV", ""))
	clock := time.Now
	if devMode {
		if pinned, perr := devFixtureClockTime(); perr == nil {
			clock = func() time.Time { return pinned }
		} else {
			log.Printf("web: VERGE_DEV: could not pin fixture clock (%v); using wall time", perr)
		}
	}
	web := newServer(queries, key, setupToken, clock)
	web.devMode = devMode
	web.transcriptKey = transcriptKey
	web.stateDir = stateDir
	web.pool = pool
	progressHub := newProgressHub()
	web.progress = progressHub
	go runProgressListener(ctx, pool, progressHub, log.New(os.Stderr, "", log.LstdFlags))
	web.secureCookies = isTruthy(env.OrDefault("VERGE_SECURE_COOKIES", ""))
	trustedProxies, err := parseTrustedProxies(env.OrDefault("VERGE_TRUSTED_PROXIES", ""))
	if err != nil {
		log.Fatalf("web: VERGE_TRUSTED_PROXIES: %v", err)
	}
	web.trustedProxies = trustedProxies
	web.externalURL = env.OrDefault("VERGE_EXTERNAL_URL", "")
	// The stale-job reaper is the worker's, so a manual trigger reads the worker's knob (#1114).
	web.dispatcher = queue.NewDispatcher(pool, time.Now, log.New(os.Stderr, "", log.LstdFlags)).
		WithStaleJobThreshold(durationOrDefault("VERGE_STALE_JOB_TIMEOUT", queue.DefaultStaleJobThreshold))
	// WriteTimeout must clear scans.go's runStreamHold, or the run long poll is cut (gosec G112).
	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           web.handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("web: shutdown: %v", err)
		}
	}()

	log.Printf("web: listening on %s", listenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("web: serve: %v", err)
	}
}

func bootstrapSetupToken(ctx context.Context, q *db.Queries) (string, error) {
	// It must exist before the database has an admin (packaging-and-configuration.md §5.1).
	n, err := q.CountAccounts(ctx)
	if err != nil {
		return "", fmt.Errorf("count accounts: %w", err)
	}
	if n > 0 {
		return "", nil
	}

	token := env.OrDefault("VERGE_SETUP_TOKEN", "")
	if token == "" {
		if token, err = auth.NewSetupToken(); err != nil {
			return "", err
		}
	}
	log.Printf("web: no accounts yet — open /setup with this single-use token: %s", token)
	return token, nil
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func durationOrDefault(key string, def time.Duration) time.Duration {
	// The worker reads the same knob, and one bad value may not configure the two differently.
	v := strings.TrimSpace(env.OrDefault(key, ""))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Printf("web: %s=%q is not a duration (%v); using default %s", key, v, err, def)
		return def
	}
	return d
}

func migrateUp(databaseURL string) error {
	sqlDB, err := pgdb.OpenStdlib(databaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("web: set goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("web: apply migrations: %w", err)
	}
	return nil
}

func checkHealth(listenAddr string) error {
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return fmt.Errorf("parse listen addr %q: %w", listenAddr, err)
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %d", resp.StatusCode)
	}
	return nil
}
