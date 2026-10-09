// internal/app/server.go
// HTTP server lifecycle, middleware stack, subdomain routing, and route registration.

package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"vallescentrales/internal/handlers"
	"vallescentrales/internal/middleware"
)

// Server holds dependencies for the HTTP application.
type Server struct {
	cfg             *Config
	db              *pgxpool.Pool
	router          *chi.Mux
	authMW          *middleware.AuthMiddleware
	authH           *handlers.AuthHandler
	listingH        *handlers.ListingHandler
	profileH        *handlers.ProfileHandler
	passkeyH        *handlers.PasskeyHandler
	uploadH         *handlers.UploadHandler
	wizardAccountH  *handlers.WizardAccountHandler
	wizardPropertyH *handlers.WizardPropertyHandler
	tmpl            *TemplateRenderer
}

// NewServer initializes all middleware and routing layers.
func NewServer(
	cfg *Config,
	db *pgxpool.Pool,
	authMW *middleware.AuthMiddleware,
	authH *handlers.AuthHandler,
	listingH *handlers.ListingHandler,
	profileH *handlers.ProfileHandler,
	passkeyH *handlers.PasskeyHandler,
	uploadH *handlers.UploadHandler,
	wizardAccountH *handlers.WizardAccountHandler,
	wizardPropertyH *handlers.WizardPropertyHandler,
	tmpl *TemplateRenderer,
) (*Server, error) {
	s := &Server{
		cfg:             cfg,
		db:              db,
		router:          chi.NewRouter(),
		authMW:          authMW,
		authH:           authH,
		listingH:        listingH,
		profileH:        profileH,
		passkeyH:        passkeyH,
		uploadH:         uploadH,
		wizardAccountH:  wizardAccountH,
		wizardPropertyH: wizardPropertyH,
		tmpl:            tmpl,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s, nil
}

// setupMiddleware initializes the global HTTP middleware stack.
// Zero in-app compression middleware — edge proxies (Railway/Cloudflare) handle compression natively.
func (s *Server) setupMiddleware() {
	s.router.Use(chimiddleware.RequestID)
	s.router.Use(chimiddleware.RealIP)
	s.router.Use(chimiddleware.Recoverer)
	s.router.Use(securityHeaders)
	s.router.Use(s.authMW.LoadUser)
}

// setupRoutes configures subdomain-aware routing, static assets, and health endpoints.
func (s *Server) setupRoutes() {
	// ─── Health & Readiness Endpoints (unauthenticated, globally accessible) ───
	s.router.Get("/healthz", s.handleLiveness)
	s.router.Get("/readyz", s.handleReadiness)

	// ─── Static Assets (MIME-safe, cached) ───
	s.router.Handle("/static/*", http.StripPrefix("/static/", http.HandlerFunc(staticFileHandler)))

	// ─── Subdomain Dispatcher ───
	s.router.HandleFunc("/*", s.subdomainDispatcher)
}

// subdomainDispatcher routes requests to the correct portal based on the Host header.
func (s *Server) subdomainDispatcher(w http.ResponseWriter, r *http.Request) {
	subdomain := extractSubdomain(r.Host, s.cfg.BaseDomain)

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	switch subdomain {
	case "":
		// Apex domain and www variant (vallescentrales.com, www.vallescentrales.com, or localhost)
		if r.URL.Path != "/" {
			// Redirect non-root apex/www paths to the realty subdomain
			host := "bienesraices." + s.cfg.BaseDomain
			if s.cfg.BaseDomain == "localhost" || s.cfg.BaseDomain == "127.0.0.1" {
				if idx := strings.Index(r.Host, ":"); idx != -1 {
					host = "bienesraices.localhost" + r.Host[idx:]
				} else {
					host = "bienesraices.localhost:8080"
				}
			}
			target := scheme + "://" + host + r.URL.Path
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		// Gateway Hub homepage served directly
		s.hubHandler().ServeHTTP(w, r)

	case "bienesraices":
		// Real Estate portal
		s.realtyRouter().ServeHTTP(w, r)

	default:
		slog.Info("unknown subdomain requested",
			"subdomain", subdomain,
			"host", r.Host,
			"path", r.URL.Path,
			"security.probe", true,
		)
		http.NotFound(w, r)
	}
}

// hubHandler renders the apex Gateway Hub page.
func (s *Server) hubHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realtyURL := fmt.Sprintf("//bienesraices.%s", r.Host)
		if s.cfg.BaseDomain != "localhost" && s.cfg.BaseDomain != "127.0.0.1" {
			realtyURL = fmt.Sprintf("https://bienesraices.%s", s.cfg.BaseDomain)
		}

		data := map[string]any{
			"Title":     "Valles Centrales | Portal Regional",
			"RealtyURL": realtyURL,
			"Global": map[string]any{
				"CurrentYear": time.Now().Year(),
			},
		}

		s.tmpl.Render(w, r, "hub.tmpl", data)
	}
}

// realtyRouter defines the routes for bienesraices.vallescentrales.com.
func (s *Server) realtyRouter() http.Handler {
	r := chi.NewRouter()

	// CSRF protection for all form submissions on this subdomain
	r.Use(middleware.CSRFProtect(s.cfg.IsProduction()))

	// ─── Public Routes ───
	r.Get("/", s.listingH.HandleHome)
	r.Get("/propiedades", s.listingH.HandleListListings)
	r.Get("/propiedades/{slug}", s.listingH.HandleGetListing)
	r.Get("/usuarios/{id}", s.profileH.HandlePublicProfile)

	// ─── Authentication Routes ───
	r.Get("/registro", s.authH.HandleRegisterPage)
	r.Post("/registro", s.authH.HandleRegister)
	r.Get("/login", s.authH.HandleLoginPage)
	r.Post("/login", s.authH.HandleLogin)
	r.Post("/logout", s.authH.HandleLogout)

	// Google OAuth 2.0
	r.Get("/auth/google", s.authH.HandleGoogleLogin)
	r.Get("/auth/google/callback", s.authH.HandleGoogleCallback)

	// WebAuthn / Passkeys
	r.Post("/webauthn/register/begin", s.passkeyH.HandleRegisterBegin)
	r.Post("/webauthn/register/finish", s.passkeyH.HandleRegisterFinish)
	r.Post("/webauthn/login/begin", s.passkeyH.HandleLoginBegin)
	r.Post("/webauthn/login/finish", s.passkeyH.HandleLoginFinish)

	// ─── Authenticated Routes ───
	r.Group(func(auth chi.Router) {
		auth.Use(s.authMW.RequireAuth)

		// Account Onboarding Wizard
		auth.Get("/bienvenido", s.wizardAccountH.Step)
		auth.Get("/bienvenido/paso/{step}", s.wizardAccountH.Step)
		auth.Post("/bienvenido/paso/{step}", s.wizardAccountH.SaveStep)
		auth.Post("/bienvenido/auto-save", s.wizardAccountH.AutoSave)
		auth.Get("/bienvenido/completar", s.wizardAccountH.Complete)
		auth.Post("/bienvenido/completar", s.wizardAccountH.Complete)

		// Property Publishing Wizard
		auth.Get("/publicar", s.wizardPropertyH.Step)
		auth.Get("/publicar/paso/{step}", s.wizardPropertyH.Step)
		auth.Post("/publicar/paso/{step}", s.wizardPropertyH.SaveStep)
		auth.Post("/publicar/auto-save", s.wizardPropertyH.AutoSave)
		auth.Post("/publicar/foto", s.wizardPropertyH.UploadPhoto)
		auth.Get("/publicar/completar", s.wizardPropertyH.Complete)
		auth.Post("/publicar/completar", s.wizardPropertyH.Complete)

		// Dashboard & Listing Management
		auth.Get("/cuenta", s.listingH.HandleDashboard)
		auth.Get("/propiedades/{slug}/editar", s.listingH.HandleEditListingPage)
		auth.Post("/propiedades/{slug}/editar", s.listingH.HandleEditListing)
		auth.Post("/propiedades/{slug}/publicar", s.listingH.HandlePublishListing)
		auth.Post("/propiedades/{slug}/eliminar", s.listingH.HandleDeleteListing)

		// Profile & Security
		auth.Get("/cuenta/perfil", s.profileH.HandleProfileEditPage)
		auth.Post("/cuenta/perfil", s.profileH.HandleProfileSave)
		auth.Get("/cuenta/seguridad", s.profileH.HandleSecurityPage)
		auth.Post("/cuenta/seguridad", s.profileH.HandleChangePassword)
		auth.Post("/webauthn/delete", s.passkeyH.HandleDeletePasskey)

		// Upload APIs
		auth.Post("/api/upload/listing-photos", s.uploadH.HandleUploadListingPhotos)
		auth.Post("/api/upload/avatar", s.uploadH.HandleUploadAvatar)
	})

	return r
}

// handleLiveness responds with 200 OK if the process is running.
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleReadiness verifies the app is ready to serve traffic.
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		slog.Error("readiness check failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(w, `{"status":"unavailable","database":"unreachable"}`)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","database":"connected"}`))
}

// staticFileHandler serves static files with explicit MIME types and cache headers.
func staticFileHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(path, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		w.Header().Set("Content-Type", "image/jpeg")
	case strings.HasSuffix(path, ".webp"):
		w.Header().Set("Content-Type", "image/webp")
	case strings.HasSuffix(path, ".woff2"):
		w.Header().Set("Content-Type", "font/woff2")
	case strings.HasSuffix(path, ".ico"):
		w.Header().Set("Content-Type", "image/x-icon")
	}

	w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
	http.FileServer(http.Dir("static")).ServeHTTP(w, r)
}

// securityHeaders applies strict security headers across all responses.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self'; "+
				"img-src 'self' data: https:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self';")
		next.ServeHTTP(w, r)
	})
}

// extractSubdomain extracts the subdomain prefix from a Host header.
func extractSubdomain(hostHeader, baseDomain string) string {
	host := hostHeader
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(host)
	base := strings.ToLower(baseDomain)

	// Treat apex and www identically as the root hub
	if host == base || host == "www."+base {
		return ""
	}
	if strings.HasSuffix(host, "."+base) {
		return host[:len(host)-len(base)-1]
	}
	if base == "localhost" || base == "127.0.0.1" {
		if host == "localhost" || host == "127.0.0.1" || host == "www.localhost" {
			return ""
		}
		if strings.HasSuffix(host, ".localhost") {
			return host[:len(host)-10]
		}
	}
	return ""
}

// Start launches the HTTP server with graceful shutdown handling.
func (s *Server) Start() error {
	addr := ":" + s.cfg.AppPort
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	shutdownComplete := make(chan struct{})

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		slog.Info("shutting down server gracefully")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("server forced to shutdown", "error", err)
		}

		close(shutdownComplete)
	}()

	slog.Info("server listening",
		"addr", addr,
		"env", s.cfg.AppEnv,
		"base_domain", s.cfg.BaseDomain,
	)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server listen error: %w", err)
	}

	<-shutdownComplete
	slog.Info("server stopped cleanly")
	return nil
}
