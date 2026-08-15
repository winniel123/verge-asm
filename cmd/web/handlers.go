package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

// heartbeater is the slice of *db.Queries the web handlers need, so tests
// can supply a fake instead of a real database connection.
type heartbeater interface {
	RecordHeartbeat(ctx context.Context) (db.Heartbeat, error)
}

func newMux(h heartbeater) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler(h))
	return mux
}

func healthzHandler(h heartbeater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hb, err := h.RecordHeartbeat(r.Context())
		if err != nil {
			log.Printf("web: healthz: record heartbeat: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Status    string    `json:"status"`
			CheckedAt time.Time `json:"checked_at"`
		}{Status: "ok", CheckedAt: hb.CheckedAt.Time})
	}
}
