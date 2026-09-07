package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/queue"
)

type captureDBTX struct {
	args []any
}

func (c *captureDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (c *captureDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (c *captureDBTX) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	c.args = args
	return emptyRow{}
}

type emptyRow struct{}

func (emptyRow) Scan(...any) error { return nil }

func (c *captureDBTX) reservation(t *testing.T, throttle queue.CTThrottle) (source string, intervalSeconds float64) {
	t.Helper()
	c.args = nil
	if _, err := throttle.Reserve(context.Background()); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if len(c.args) != 2 {
		t.Fatalf("ReserveCTSlot got %d args, want 2", len(c.args))
	}
	secs, ok := c.args[0].(float64)
	if !ok {
		t.Fatalf("interval_seconds is %T, want float64", c.args[0])
	}
	slug, ok := c.args[1].(string)
	if !ok {
		t.Fatalf("source is %T, want string", c.args[1])
	}
	return slug, secs
}

func TestSelectCTSourcePacesTheCTLogPathsAtTwelveSeconds(t *testing.T) {
	for _, tc := range []struct {
		name       string
		token      string
		bulkSource string
		bulkSecs   float64
	}{
		{name: "no token", token: "", bulkSource: "crtsh", bulkSecs: 12},
		{name: "certspotter token", token: "a-token", bulkSource: "certspotter", bulkSecs: 360},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &captureDBTX{}
			_, bulk, ctLog, _ := selectCTSource(tc.token, "test", db.New(tx))

			bulkSlug, bulkSecs := tx.reservation(t, bulk)
			if bulkSlug != tc.bulkSource || bulkSecs != tc.bulkSecs {
				t.Errorf("bulk ct reserves %s at %gs, want %s at %gs", bulkSlug, bulkSecs, tc.bulkSource, tc.bulkSecs)
			}

			ctLogSlug, ctLogSecs := tx.reservation(t, ctLog)
			if ctLogSlug != "crtsh" || ctLogSecs != 12 {
				t.Errorf("ct-tail and ct-verify reserve %s at %gs, want crtsh at 12s (#1520)", ctLogSlug, ctLogSecs)
			}
		})
	}
}
