// Command web is the only listener in the deployment: it serves the
// operator UI, applies database migrations on startup, and is the
// container docker compose's healthcheck watches (ADR-0001, §4.2).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/winniel123/verge-asm/db/migrations"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/env"
	"github.com/winniel123/verge-asm/internal/pgdb"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check that the running instance is healthy, then exit")
	flag.Parse()

	listenAddr := env.OrDefault("VERGE_LISTEN_ADDR", ":8080")

	if *healthcheck {
		if err := checkHealth(listenAddr); err != nil {
			log.Fatalf("web: healthcheck: %v", err)
		}
		return
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

	srv := &http.Server{Addr: listenAddr, Handler: newMux(db.New(pool))}

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

