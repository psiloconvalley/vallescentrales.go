// internal/handlers/helpers.go
// Shared HTTP response helpers and renderer interface.

package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Renderer is the interface for rendering HTML templates.
// app.TemplateRenderer satisfies this interface.
// Defined here to avoid handlers importing the app package.
type Renderer interface {
	Render(w http.ResponseWriter, r *http.Request, name string, data any)
}

// respond writes a JSON response with the given status code and data.
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
