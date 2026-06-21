// internal/auth/google.go
// Google OAuth 2.0 flow — redirect, callback, user info.
// Improved from psiloconvalley reference:
//   → Config injected via struct, not global var
//   → slog throughout
//   → SameSite=Strict on state cookie
//   → Validates token expiry
//   → Secure flag from config, not env check

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	oauthStateCookie   = "__vc_oauth_state"
	stateTokenBytes    = 32
	stateCookieMaxAge  = 300
	googleUserInfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo"
	tokenExchangeTimeout = 10 * time.Second
)

// GoogleUser holds the profile data returned by Google.
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// GoogleOAuth manages the Google OAuth 2.0 flow.
type GoogleOAuth struct {
	config *oauth2.Config
	secure bool
}

// NewGoogleOAuth creates a GoogleOAuth handler.
// Returns nil if clientID is empty — Google OAuth is optional.
func NewGoogleOAuth(clientID, clientSecret, redirectURL string, secure bool) *GoogleOAuth {
	if clientID == "" {
		slog.Info("google oauth not configured — feature disabled")
		return nil
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
	}
}

// Enabled returns true if Google OAuth is configured.
func (g *GoogleOAuth) Enabled() bool {
	return g != nil && g.config != nil
}

// RedirectToGoogle generates a state token, sets the CSRF cookie,
// and redirects the user to Google's consent screen.
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

// ProcessCallback validates the state, exchanges the code for a token,
// and fetches the Google user profile.
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

// fetchUser retrieves the Google user's profile information.
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

// setStateCookie writes the CSRF state cookie.
func (g *GoogleOAuth) setStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   stateCookieMaxAge,
		HttpOnly: true,
		Secure:   g.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// verifyStateCookie checks the state parameter matches the cookie.
func (g *GoogleOAuth) verifyStateCookie(r *http.Request, state string) bool {
	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil {
		slog.Warn("oauth state cookie missing")
		return false
	}

	if cookie.Value != state || state == "" {
		expectedPrefix := cookie.Value
		if len(expectedPrefix) > 8 {
			expectedPrefix = expectedPrefix[:8]
		}
		gotPrefix := state
		if len(gotPrefix) > 8 {
			gotPrefix = gotPrefix[:8]
		}

		slog.Warn("oauth state mismatch",
			"expected_prefix", expectedPrefix,
			"got_prefix", gotPrefix,
		)
		return false
	}

	return true
}

// clearStateCookie removes the state cookie after use.
func (g *GoogleOAuth) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// generateStateToken creates a cryptographically secure random string.
func generateStateToken() (string, error) {
	b := make([]byte, stateTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateStateToken: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
