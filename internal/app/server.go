// internal/app/server.go
// Router wiring and graceful shutdown.
// Handlers are stubs — real implementations come in Session 009.

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
}

func (s *Server) mountRoutes() {
	// Health check — load balancers and Railway hit this
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

	// PROTECTED routes — session required (ADR-002)
	// Middleware enforcement added in Session 008
	s.router.Get("/dashboard", s.handleDashboard)
	s.router.Get("/listings/new", s.handleNewListingPage)
	s.router.Post("/listings/new", s.handleCreateListing)
	s.router.Get("/listings/{slug}/edit", s.handleEditListingPage)
	s.router.Post("/listings/{slug}/edit", s.handleEditListing)
	s.router.Post("/listings/{slug}/delete", s.handleDeleteListing)
	s.router.Post("/listings/{slug}/photos", s.handleUploadPhotos)
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

	// Graceful shutdown — finish in-flight requests up to 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}

	s.logger.Info("server stopped cleanly")
	return nil
}

// requestLogger is structured middleware for every request.
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
