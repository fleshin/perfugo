package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	applog "perfugo/internal/log"
)

type healthResponse struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

// Health is a simple readiness handler suitable for infrastructure probes.
func Health(w http.ResponseWriter, r *http.Request) {
	applog.Debug(r.Context(), "health check requested", "method", r.Method)
	resp := healthResponse{
		Status: "ok",
		Time:   time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		applog.Error(r.Context(), "failed to encode health response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	applog.Debug(r.Context(), "health check responded successfully")
}
