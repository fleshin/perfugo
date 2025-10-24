package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

type healthResponse struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

// Health is a simple readiness handler suitable for infrastructure probes.
func Health(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status: "ok",
		Time:   time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
