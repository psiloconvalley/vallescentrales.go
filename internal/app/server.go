// internal/app/server.go
// Router wiring, security middleware, auth-aware route groups,
// CSRF protection, rate limiting, and graceful shutdown.

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
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"vallescentrales/internal/handlers"
	appmiddleware "vallescentrales/internal/middleware"
)

const (
	maxRequestBodyBytes = 1 << 20
)

type Server struct {
	cfg      *Config
	db       *pgxpool.Pool
	router   *chi.Mux
	logger   *slog.Logger
	tmpl     *TemplateRenderer
	authMW   *appmiddleware.AuthMiddleware
	authH    *handlers.AuthHandler
	listingH *handlers.ListingHandler
	profileH *handlers.ProfileHandler
	passkeyH *handlers.PasskeyHandler
}

func NewServer(
	cfg *Config,
	db *pgxpool.Pool,
	authMW *appmiddleware.AuthMiddleware,
	authH *handlers.AuthHandler,
	listingH *handlers.ListingHandler,
	profileH *handlers.ProfileHandler,
	passkeyH *handlers.PasskeyHandler,
	tmpl *TemplateRenderer,
) (*Server, error) {
	if authMW == nil {
		return nil, fmt.Errorf("server: auth middleware is required")
	}
	if authH == nil {
		return nil, fmt.Errorf("server: auth handler is required")
	}
	if listingH == nil {
		return nil, fmt.Errorf("server: listing handler is required")
	}
	if profileH == nil {
		return nil, fmt.Errorf("server: profile handler is required")
	}
	if passkeyH == nil {
		return nil, fmt.Errorf("server: passkey handler is required")
	}
	if tmpl == nil {
		return nil, fmt.Errorf("server: template renderer is required")
	}

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
		cfg:      cfg,
		db:       db,
		router:   chi.NewRouter(),
		logger:   logger,
		tmpl:     tmpl,
		authMW:   authMW,
		authH:    authH,
		listingH: listingH,
		profileH: profileH,
		passkeyH: passkeyH,
	}

	s.mountMiddleware()
	s.mountRoutes()

	return s, nil
}

func (s *Server) mountMiddleware() {
	s.router.Use(chimiddleware.RequestID)
	s.router.Use(chimiddleware.RealIP)
	s.router.Use(chimiddleware.Recoverer)
	s.router.Use(s.requestLogger)
	s.router.Use(chimiddleware.Timeout(30 * time.Second))
	s.router.Use(s.securityHeaders)
	s.router.Use(s.limitRequestBody)
}

func (s *Server) mountRoutes() {
	s.router.Get("/health", s.handleHealth)

	s.router.Handle("/static/*", http.StripPrefix("/static/",
		s.cacheStatic(http.FileServer(http.Dir("static")))))

	authLimiter := appmiddleware.NewRateLimiter(10, time.Minute)

	s.router.Group(func(r chi.Router) {
		r.Use(appmiddleware.CSRFProtect(s.cfg.IsProduction()))
		r.Use(s.authMW.LoadUser)

		// PUBLIC
		r.Get("/", s.listingH.HandleHome)
		r.Get("/listings", s.listingH.HandleListListings)
		r.Get("/listings/{slug}", s.listingH.HandleGetListing)

		// AUTH — rate limited
		r.Group(func(r chi.Router) {
			r.Use(authLimiter.Limit)
			r.Get("/auth/login", s.authH.HandleLoginPage)
			r.Post("/auth/login", s.authH.HandleLogin)
			r.Get("/auth/register", s.authH.HandleRegisterPage)
			r.Post("/auth/register", s.authH.HandleRegister)
			r.Post("/auth/logout", s.authH.HandleLogout)
			r.Get("/auth/google", s.authH.HandleGoogleLogin)
			r.Get("/auth/google/callback", s.authH.HandleGoogleCallback)

			// Passkey login — public (no auth required)
			r.Post("/auth/passkey/login/begin", s.passkeyH.HandleLoginBegin)
			r.Post("/auth/passkey/login/finish", s.passkeyH.HandleLoginFinish)
		})

		// PROTECTED
		r.Group(func(r chi.Router) {
			r.Use(s.authMW.RequireAuth)

			// Dashboard
			r.Get("/dashboard", s.listingH.HandleDashboard)

			// Profile
			r.Get("/dashboard/profile", s.profileH.HandleProfileEditPage)
			r.Post("/dashboard/profile", s.profileH.HandleProfileSave)
			r.Get("/dashboard/security", s.profileH.HandleSecurityPage)
			r.Post("/dashboard/security/password", s.profileH.HandleChangePassword)

			// Passkey management — must be logged in
			r.Post("/auth/passkey/register/begin", s.passkeyH.HandleRegisterBegin)
			r.Post("/auth/passkey/register/finish", s.passkeyH.HandleRegisterFinish)
			r.Delete("/auth/passkey/{id}", s.passkeyH.HandleDeletePasskey)

			// Listings management
			r.Get("/listings/new", s.listingH.HandleNewListingPage)
			r.Post("/listings/new", s.listingH.HandleCreateListing)
			r.Get("/listings/{slug}/edit", s.handleEditListingPage)
			r.Post("/listings/{slug}/edit", s.handleEditListing)
			r.Post("/listings/{slug}/delete", s.listingH.HandleDeleteListing)
			r.Post("/listings/{slug}/photos", s.handleUploadPhotos)
		})
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		s.logger.Error("health check failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"database unavailable"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"status":"ok","env":%q}`, s.cfg.AppEnv)
}

func (s *Server) handleEditListingPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"not implemented"}`))
}

func (s *Server) handleEditListing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"not implemented"}`))
}

func (s *Server) handleUploadPhotos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"not implemented"}`))
}

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}

	s.logger.Info("server stopped cleanly")
	return nil
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"ms", time.Since(start).Milliseconds(),
			"id", chimiddleware.GetReqID(r.Context()),
		)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'")
		if s.cfg.IsProduction() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// cacheStatic sets aggressive cache headers on static assets.
// Assets are versioned via ?v= query string in templates.
// When the version changes (new deploy), browser fetches fresh files.
func (s *Server) cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
