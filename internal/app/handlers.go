// internal/app/handlers.go
// Stub handlers — enough to compile and boot.
// Real implementations come in Session 009.
// Rule 42: handlers = HTTP only. No SQL. No business logic.

package app

import (
	"encoding/json"
	"net/http"
	"time"
)

// respond is the single JSON response writer for all handlers.
// Every handler uses this. Never write JSON manually.
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// respondError is the consistent error shape across the entire API.
func respondError(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}

// handleHealth — always returns 200 if DB is reachable.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		s.logger.Error("health check failed", "error", err)
		respondError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"env":       s.cfg.AppEnv,
	})
}

// PUBLIC stubs
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "home — coming in session 010"})
}

func (s *Server) handleListListings(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]any{"listings": []any{}, "total": 0})
}

func (s *Server) handleGetListing(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "listing detail — coming in session 010"})
}

// AUTH stubs
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "login — coming in session 010"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 007")
}

func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "register — coming in session 010"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 007")
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 007")
}

// PROTECTED stubs
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 009")
}

func (s *Server) handleNewListingPage(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 009")
}

func (s *Server) handleCreateListing(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 009")
}

func (s *Server) handleEditListingPage(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 009")
}

func (s *Server) handleEditListing(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 009")
}

func (s *Server) handleDeleteListing(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 009")
}

func (s *Server) handleUploadPhotos(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "coming in session 012")
}
