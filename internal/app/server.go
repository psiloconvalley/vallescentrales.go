// internal/app/server.go
// Router wiring, security middleware, and graceful shutdown.

package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// maxRequestBodyBytes limits request body size — prevents DoS via large payloads.
	// Photo uploads use multipart with their own limit — this covers everything else.
	maxRequestBodyBytes = 1 << 20 // 1MB
)

type Server struct {
	cfg    *Config
	db     *pgxpool.Pool
	router *chi.Mux
	logger *slog.Logger
}

func NewServer(cfg *Config, db *pgxpool.Pool) *Server {
	var handler slog.Handler
	if cfg.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	s := &Server{
		cfg:    cfg,
		db:     db,
		router: chi.NewRouter(),
		logger: logger,
	}

	s.mountMiddleware()
	s.mountRoutes()

	return s
}

func (s *Server) mountMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.requestLogger)
	s.router.Use(middleware.Timeout(30 * time.Second))
	s.router.Use(s.securityHeaders)
	s.router.Use(s.limitRequestBody)
}

func (s *Server) mountRoutes() {
	// Health check — Railway and load balancers hit this
	s.router.Get("/health", s.handleHealth)

	// Static files
	s.router.Handle("/static/*", http.StripPrefix("/static/",
		http.FileServer(http.Dir("static"))))

	// PUBLIC routes — no auth required (ADR-002)
	s.router.Get("/", s.handleHome)
	s.router.Get("/listings", s.handleListListings)
	s.router.Get("/listings/{slug}", s.handleGetListing)

	// AUTH routes — public (ADR-002)
	s.router.Get("/auth/login", s.handleLoginPage)
	s.router.Post("/auth/login", s.handleLogin)
	s.router.Get("/auth/register", s.handleRegisterPage)
	s.router.Post("/auth/register", s.handleRegister)
	s.router.Post("/auth/logout", s.handleLogout)

	// PROTECTED routes — RequireAuth middleware added in Session 008
	// Grouped so middleware can be applied to the whole subrouter at once
	s.router.Group(func(r chi.Router) {
		// TODO Session 008: r.Use(s.requireAuth)
		r.Get("/dashboard", s.handleDashboard)
		r.Get("/listings/new", s.handleNewListingPage)
		r.Post("/listings/new", s.handleCreateListing)
		r.Get("/listings/{slug}/edit", s.handleEditListingPage)
		r.Post("/listings/{slug}/edit", s.handleEditListing)
		r.Post("/listings/{slug}/delete", s.handleDeleteListing)
		r.Post("/listings/{slug}/photos", s.handleUploadPhotos)
	})
}

// Start runs the server and blocks until a shutdown signal is received.
func (s *Server) Start() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.cfg.AppPort),
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		s.logger.Info("server starting",
			"port", s.cfg.AppPort,
			"env", s.cfg.AppEnv,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.Info("shutdown signal received", "signal", sig)
	}

	// Graceful shutdown — drain in-flight requests up to 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}

	s.logger.Info("server stopped cleanly")
	return nil
}

// requestLogger logs every request with method, path, status, and duration.
func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"ms", time.Since(start).Milliseconds(),
			"id", middleware.GetReqID(r.Context()),
		)
	})
}

// securityHeaders sets defensive HTTP headers on every response.
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// Limit referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Disable browser features we do not use
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// Force HTTPS in production
		if s.cfg.IsProduction() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// limitRequestBody caps the request body size to prevent DoS attacks.
// Photo upload routes handle their own limits via multipart parsing.
func (s *Server) limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}
