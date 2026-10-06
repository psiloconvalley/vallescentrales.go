// internal/auth/google.go

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	oauthStateCookie     = "__vc_oauth_state"
	stateTokenBytes      = 32
	stateCookieMaxAge    = 300
	googleUserInfoURL    = "https://www.googleapis.com/oauth2/v2/userinfo"
	tokenExchangeTimeout = 10 * time.Second
)

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type GoogleOAuth struct {
	config *oauth2.Config
	secure bool
	domain string
}

func NewGoogleOAuth(clientID, clientSecret, redirectURL string, secure bool, baseDomain string) *GoogleOAuth {
	if clientID == "" {
		slog.Info("google oauth not configured — feature disabled")
		return nil
	}

	cookieDomain := ""
	if baseDomain != "" && baseDomain != "localhost" && baseDomain != "127.0.0.1" {
		if idx := strings.Index(baseDomain, ":"); idx != -1 {
			baseDomain = baseDomain[:idx]
		}
		if !strings.HasPrefix(baseDomain, ".") {
			cookieDomain = "." + baseDomain
		} else {
			cookieDomain = baseDomain
		}
	}

	return &GoogleOAuth{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		secure: secure,
		domain: cookieDomain,
	}
}

func (g *GoogleOAuth) Enabled() bool {
	return g != nil && g.config != nil
}

func (g *GoogleOAuth) RedirectToGoogle(w http.ResponseWriter, r *http.Request) {
	state, err := generateStateToken()
	if err != nil {
		slog.Error("failed to generate oauth state token", "error", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	g.setStateCookie(w, state)

	url := g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (g *GoogleOAuth) ProcessCallback(w http.ResponseWriter, r *http.Request) (*GoogleUser, error) {
	state := r.URL.Query().Get("state")
	if !g.verifyStateCookie(r, state) {
		return nil, fmt.Errorf("invalid oauth state — possible CSRF")
	}
	g.clearStateCookie(w)

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		return nil, fmt.Errorf("oauth denied by user: %s", errParam)
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("no authorization code in callback")
	}

	ctx, cancel := context.WithTimeout(r.Context(), tokenExchangeTimeout)
	defer cancel()

	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	if !token.Valid() {
		return nil, fmt.Errorf("received invalid or expired token from Google")
	}

	return g.fetchUser(ctx, token)
}

func (g *GoogleOAuth) fetchUser(ctx context.Context, token *oauth2.Token) (*GoogleUser, error) {
	client := g.config.Client(ctx, token)

	resp, err := client.Get(googleUserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo API returned status %d", resp.StatusCode)
	}

	var gu GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gu); err != nil {
		return nil, fmt.Errorf("failed to decode google user info: %w", err)
	}

	if gu.Email == "" {
		return nil, fmt.Errorf("no email returned from Google")
	}

	if !gu.VerifiedEmail {
		return nil, fmt.Errorf("google email is not verified")
	}

	return &gu, nil
}

func (g *GoogleOAuth) setStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		Domain:   g.domain,
		MaxAge:   stateCookieMaxAge,
		HttpOnly: true,
		Secure:   g.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (g *GoogleOAuth) verifyStateCookie(r *http.Request, state string) bool {
	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil {
		slog.Warn("oauth state cookie missing")
		return false
	}

	if cookie.Value != state || state == "" {
		slog.Warn("oauth state mismatch")
		return false
	}

	return true
}

func (g *GoogleOAuth) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		Domain:   g.domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   g.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func generateStateToken() (string, error) {
	b := make([]byte, stateTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateStateToken: failed to read random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
