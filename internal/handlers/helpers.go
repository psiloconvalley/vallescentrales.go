// internal/handlers/helpers.go
// Shared HTTP response helpers for all handlers.
// respond and respondError are the only ways to write JSON.
// render is the only way to write HTML templates (wired in Session 010).

package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// respond writes a JSON response with the given status code and data.
// Every handler uses this. Never write JSON manually.
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode JSON response", "error", err)
		}
	}
}

// respondError writes a consistent JSON error response.
func respondError(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}

// healthResponse is the shape of the /health endpoint response.
type healthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Env       string    `json:"env"`
}
