// internal/app/handlers.go
// Stub handlers — real implementations written in Session 009.
// respond() and respondError() move to internal/handlers/ in Session 009.
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

// handleHealth — returns 200 if server and DB are reachable.
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
	respond(w, http.StatusOK, map[string]string{"page": "home"})
}

func (s *Server) handleListListings(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]any{"listings": []any{}, "total": 0})
}

func (s *Server) handleGetListing(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "listing detail"})
}

// AUTH stubs
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "login"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "register"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

// PROTECTED stubs
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleNewListingPage(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleCreateListing(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleEditListingPage(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleEditListing(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleDeleteListing(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}

func (s *Server) handleUploadPhotos(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented")
}
