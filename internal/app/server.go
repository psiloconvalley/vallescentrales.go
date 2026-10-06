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

// Server coordinates the HTTP router, middleware, and lifecycle.
type Server struct {
	cfg      *Config
	db       *pgxpool.Pool
	authMW   *middleware.AuthMiddleware
	authH    *handlers.AuthHandler
	listingH *handlers.ListingHandler
	profileH *handlers.ProfileHandler
	passkeyH *handlers.PasskeyHandler
	uploadH  *handlers.UploadHandler
	tmpl     *TemplateRenderer
	router   *chi.Mux
}

// NewServer initializes the application server and routes.
func NewServer(
	cfg *Config,
	db *pgxpool.Pool,
	authMW *middleware.AuthMiddleware,
	authH *handlers.AuthHandler,
	listingH *handlers.ListingHandler,
	profileH *handlers.ProfileHandler,
	passkeyH *handlers.PasskeyHandler,
	uploadH *handlers.UploadHandler,
	tmpl *TemplateRenderer,
) (*Server, error) {
	s := &Server{
		cfg:      cfg,
		db:       db,
		authMW:   authMW,
		authH:    authH,
		listingH: listingH,
		profileH: profileH,
		passkeyH: passkeyH,
		uploadH:  uploadH,
		tmpl:     tmpl,
		router:   chi.NewRouter(),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s, nil
}

// setupMiddleware attaches global middleware to Chi.
func (s *Server) setupMiddleware() {
	s.router.Use(chimiddleware.RequestID)
	s.router.Use(chimiddleware.RealIP)
	s.router.Use(chimiddleware.Recoverer)
	s.router.Use(securityHeaders)
	s.router.Use(s.authMW.LoadUser)
}

// setupRoutes configures subdomain-aware routing and static assets.
func (s *Server) setupRoutes() {
	// Serve static assets across all domains
	fileServer := http.FileServer(http.Dir("static"))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Subdomain Dispatcher
	s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		subdomain := extractSubdomain(r.Host, s.cfg.BaseDomain)

		switch subdomain {
		case "":
			// Apex Domain (vallescentrales.com / localhost:8080) -> Gateway Hub
			s.hubHandler().ServeHTTP(w, r)
		case "bienesraices":
			// Real Estate Subdomain
			s.realtyRouter().ServeHTTP(w, r)
		default:
			// Redirect www.vallescentrales.com to vallescentrales.com
			if subdomain == "www" {
				target := "https://" + s.cfg.BaseDomain + r.URL.Path
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
			http.NotFound(w, r)
		}
	})
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

	// CSRF protection for subdomains HTML-rendering/form submissions
	csrfMiddleware := middleware.CSRFProtect(s.cfg.IsProduction())
	r.Use(csrfMiddleware)

	// Public real estate routes
	r.Get("/", s.listingH.HandleHome)
	r.Get("/propiedades", s.listingH.HandleListListings)
	r.Get("/propiedades/{slug}", s.listingH.HandleGetListing)

	// Authentication routes
	r.Get("/registro", s.authH.HandleRegisterPage)
	r.Post("/registro", s.authH.HandleRegister)
	r.Get("/login", s.authH.HandleLoginPage)
	r.Post("/login", s.authH.HandleLogin)
	r.Post("/logout", s.authH.HandleLogout)
	r.Get("/auth/google", s.authH.HandleGoogleLogin)
	r.Get("/auth/google/callback", s.authH.HandleGoogleCallback)

	// Passkey API
	r.Post("/webauthn/register/begin", s.passkeyH.HandleRegisterBegin)
	r.Post("/webauthn/register/finish", s.passkeyH.HandleRegisterFinish)
	r.Post("/webauthn/login/begin", s.passkeyH.HandleLoginBegin)
	r.Post("/webauthn/login/finish", s.passkeyH.HandleLoginFinish)

	// Public user profile
	r.Get("/usuarios/{id}", s.profileH.HandlePublicProfile)

	// Authenticated route group
	r.Group(func(auth chi.Router) {
		auth.Use(s.authMW.RequireAuth)

		// Listings management panel
		auth.Get("/cuenta", s.listingH.HandleDashboard)
		auth.Get("/publicar", s.listingH.HandleNewListingPage)
		auth.Post("/publicar", s.listingH.HandleCreateListing)
		auth.Get("/propiedades/{slug}/editar", s.listingH.HandleEditListingPage)
		auth.Post("/propiedades/{slug}/editar", s.listingH.HandleEditListing)
		auth.Post("/propiedades/{slug}/publicar", s.listingH.HandlePublishListing)
		auth.Post("/propiedades/{slug}/eliminar", s.listingH.HandleDeleteListing)

		// Profile and Security settings
		auth.Get("/cuenta/perfil", s.profileH.HandleProfileEditPage)
		auth.Post("/cuenta/perfil", s.profileH.HandleProfileSave)
		auth.Get("/cuenta/seguridad", s.profileH.HandleSecurityPage)
		auth.Post("/cuenta/seguridad", s.profileH.HandleChangePassword)
		auth.Post("/webauthn/delete", s.passkeyH.HandleDeletePasskey)

		// Photo and Avatar uploads
		auth.Post("/api/upload/listing-photos", s.uploadH.HandleUploadListingPhotos)
		auth.Post("/api/upload/avatar", s.uploadH.HandleUploadAvatar)
	})

	return r
}

// securityHeaders sets modern security standards following our strict CSP / Zero-Inline rules.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none';")
		next.ServeHTTP(w, r)
	})
}

// extractSubdomain extracts the subdomain from the request Host header.
func extractSubdomain(hostHeader, baseDomain string) string {
	host := hostHeader
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	host = strings.ToLower(host)
	base := strings.ToLower(baseDomain)

	if host == base || host == "www."+base {
		return ""
	}
	if strings.HasSuffix(host, "."+base) {
		return host[:len(host)-len(base)-1]
	}
	if base == "localhost" || base == "127.0.0.1" {
		if host == "localhost" || host == "127.0.0.1" {
			return ""
		}
		if strings.HasSuffix(host, ".localhost") {
			return host[:len(host)-10]
		}
	}
	return ""
}

// Start launches the HTTP server with graceful shutdown.
func (s *Server) Start() error {
	addr := ":" + s.cfg.AppPort
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdownErr := make(chan error, 1)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit

		slog.Info("shutting down server", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		shutdownErr <- httpServer.Shutdown(ctx)
	}()

	slog.Info("starting server", "addr", addr, "domain", s.cfg.BaseDomain)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return <-shutdownErr
}
