package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/queue"
)

// The tags are a wire contract pinned by TestJobProgressWire and TestDecodeProgress at both ends.

type jobProgress struct {
	Dispatch int64  `json:"dispatch"`
	Job      int64  `json:"job"`
	Level    string `json:"level"`
	Text     string `json:"text"`
}

type progressEvents interface {
	ForDispatch(dispatchID int64) []jobProgress
}

const (
	maxProgressRuns = 256
	maxEventsPerRun = 1024
)

type progressHub struct {
	mu     sync.Mutex
	byRun  map[int64][]jobProgress
	recent []int64
}

func newProgressHub() *progressHub {
	return &progressHub{byRun: map[int64][]jobProgress{}}
}

func (h *progressHub) record(ev jobProgress) {
	if ev.Dispatch == 0 || ev.Job == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	log, ok := h.byRun[ev.Dispatch]
	if !ok {
		h.recent = append(h.recent, ev.Dispatch)
		h.evictLocked()
	}
	// Evicting the front instead would shift every later index and desync the stream cursor (ADR-0182 §2).
	if len(log) >= maxEventsPerRun {
		return
	}
	h.byRun[ev.Dispatch] = append(log, ev)
}

func (h *progressHub) evictLocked() {
	for len(h.recent) > maxProgressRuns {
		oldest := h.recent[0]
		h.recent = h.recent[1:]
		delete(h.byRun, oldest)
	}
}

func (h *progressHub) ForDispatch(dispatchID int64) []jobProgress {
	h.mu.Lock()
	defer h.mu.Unlock()
	log, ok := h.byRun[dispatchID]
	if !ok || len(log) == 0 {
		return nil
	}
	out := make([]jobProgress, len(log))
	copy(out, log)
	return out
}

func decodeProgress(payload []byte) (jobProgress, bool) {
	var ev jobProgress
	if err := json.Unmarshal(payload, &ev); err != nil {
		return jobProgress{}, false
	}
	if ev.Dispatch == 0 || ev.Job == 0 {
		return jobProgress{}, false
	}
	return ev, true
}

func eventStreamLines(events []jobProgress, jobFilter int64, filtered bool) []runStreamLine {
	// The viewer only ever appends, so a reason merged onto an existing line never reaches it (ADR-0182 §1).
	out := make([]runStreamLine, 0, len(events))
	for _, ev := range events {
		if filtered && ev.Job != jobFilter {
			continue
		}
		out = append(out, runStreamLine{
			Tag:   "#" + strconv.FormatInt(ev.Job, 10),
			Level: ev.Level,
			Text:  ev.Text,
		})
	}
	return out
}

const streamCursorBase = 1_000_000

func encodeStreamCursor(events, state int) int {
	// A retry growing the state log must not shift an event's position, so the counters never collide (ADR-0182 §2).
	if state >= streamCursorBase {
		state = streamCursorBase - 1
	}
	return events*streamCursorBase + state
}

func decodeStreamCursor(after int) (events, state int) {
	if after < 0 {
		return 0, 0
	}
	return after / streamCursorBase, after % streamCursorBase
}

func runProgressListener(ctx context.Context, pool *pgxpool.Pool, hub *progressHub, logger *log.Logger) {
	const backoff = 2 * time.Second
	for ctx.Err() == nil {
		if err := listenProgressOnce(ctx, pool, hub); err != nil && ctx.Err() == nil {
			logger.Printf("web: progress listener: %v; retrying in %s", err, backoff)
			select {
			case <-ctx.Done():
			case <-time.After(backoff):
			}
		}
	}
}

func listenProgressOnce(ctx context.Context, pool *pgxpool.Pool, hub *progressHub) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "LISTEN "+queue.ProgressChannel); err != nil {
		return err
	}
	// A missed notification costs one ephemeral line, so there is no catch-up read.
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if ev, ok := decodeProgress([]byte(n.Payload)); ok {
			hub.record(ev)
		}
	}
}
