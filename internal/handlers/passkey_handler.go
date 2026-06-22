// internal/handlers/passkey_handler.go
// WebAuthn passkey registration and login endpoints.
// Rule 42: handlers = HTTP only. No SQL. No business logic.
//
// REGISTRATION FLOW:
//   POST /auth/passkey/register/begin   → returns challenge JSON
//   POST /auth/passkey/register/finish  → validates + stores credential
//
// LOGIN FLOW:
//   POST /auth/passkey/login/begin      → returns challenge JSON
//   POST /auth/passkey/login/finish     → validates + creates session

package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"vallescentrales/internal/auth"
	"vallescentrales/internal/middleware"
	"vallescentrales/internal/repo"
)

// PasskeyHandler handles WebAuthn passkey registration and login.
type PasskeyHandler struct {
	webAuthn   *webauthn.WebAuthn
	passkeys   *repo.PasskeyRepo
	users      *repo.UserRepo
	sessions   *auth.SessionManager
}

// NewPasskeyHandler creates a PasskeyHandler.
func NewPasskeyHandler(
	webAuthn *webauthn.WebAuthn,
	passkeys *repo.PasskeyRepo,
	users *repo.UserRepo,
	sessions *auth.SessionManager,
) *PasskeyHandler {
	return &PasskeyHandler{
		webAuthn: webAuthn,
		passkeys: passkeys,
		users:    users,
		sessions: sessions,
	}
}

// HandleRegisterBegin starts passkey registration for a logged-in user.
// Returns PublicKeyCredentialCreationOptions JSON for the browser.
func (h *PasskeyHandler) HandleRegisterBegin(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Build WebAuthn user from our user model
	waUser := &auth.WebAuthnUser{
		ID:          user.ID[:],
		Name:        user.Email,
		DisplayName: user.DisplayNameOrFull(),
	}

	// Load existing credentials so we can exclude them
	existing, err := h.passkeys.ListByUserID(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to list existing passkeys", "user_id", user.ID, "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	// Convert existing passkeys to WebAuthn credentials for exclusion list
	for _, pk := range existing {
		waUser.Credentials = append(waUser.Credentials, webauthn.Credential{
			ID: pk.CredentialID,
		})
	}

	// Generate registration options and session data
	options, session, err := h.webAuthn.BeginRegistration(waUser)
	if err != nil {
		slog.Error("failed to begin passkey registration", "user_id", user.ID, "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	// Serialize and store session data in DB
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		slog.Error("failed to marshal passkey session", "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	userID := user.ID
	flowID, err := h.passkeys.SaveChallenge(r.Context(), &userID, "register", sessionJSON)
	if err != nil {
		slog.Error("failed to save passkey challenge", "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	// Return options + flow_id to browser
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"flow_id": flowID,
		"options": options,
	})
}

// HandleRegisterFinish completes passkey registration.
// Validates the credential created by the browser and stores it.
func (h *PasskeyHandler) HandleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Parse request body
	var body struct {
		FlowID     string          `json:"flow_id"`
		DeviceName string          `json:"device_name"`
		Credential json.RawMessage `json:"credential"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Retrieve and validate challenge
	challenge, err := h.passkeys.GetChallenge(r.Context(), body.FlowID)
	if err != nil {
		if errors.Is(err, repo.ErrChallengeNotFound) || errors.Is(err, repo.ErrChallengeExpired) {
			respondError(w, http.StatusBadRequest, "challenge expired — please try again")
			return
		}
		slog.Error("failed to get passkey challenge", "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	// Verify challenge belongs to this user
	if challenge.UserID == nil || *challenge.UserID != user.ID {
		respondError(w, http.StatusForbidden, "challenge user mismatch")
		return
	}

	// Restore session data
	var session webauthn.SessionData
	if err := json.Unmarshal(challenge.SessionData, &session); err != nil {
		slog.Error("failed to unmarshal passkey session", "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	// Parse the credential from the browser
	parsedCredential, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		// Body was already read — parse from raw JSON instead
		parsedCredential, err = protocol.ParseCredentialCreationResponseBytes(body.Credential)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid credential")
			return
		}
	}

	waUser := &auth.WebAuthnUser{
		ID:          user.ID[:],
		Name:        user.Email,
		DisplayName: user.DisplayNameOrFull(),
	}

	// Verify the credential with WebAuthn library
	credential, err := h.webAuthn.CreateCredential(waUser, session, parsedCredential)
	if err != nil {
		slog.Error("failed to verify passkey credential", "user_id", user.ID, "error", err)
		respondError(w, http.StatusBadRequest, "credential verification failed")
		return
	}

	// Store device name
	var deviceName *string
	if body.DeviceName != "" {
		deviceName = &body.DeviceName
	}

	// Save to database
	_, err = h.passkeys.SaveCredential(
		r.Context(),
		user.ID,
		credential.ID,
		credential.PublicKey,
		credential.AttestationType,
		credential.Authenticator.AAGUID,
		transportStrings(credential.Transport),
		deviceName,
	)
	if err != nil {
		slog.Error("failed to save passkey credential", "user_id", user.ID, "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	slog.Info("passkey registered",
		"user_id", user.ID,
		"device", body.DeviceName,
	)

	respond(w, http.StatusOK, map[string]string{
		"message": "passkey registered successfully",
	})
}

// HandleLoginBegin starts a passkey login flow.
// Returns PublicKeyCredentialRequestOptions JSON for the browser.
func (h *PasskeyHandler) HandleLoginBegin(w http.ResponseWriter, r *http.Request) {
	// Discoverable credential flow — no email required
	// Browser will prompt user to select their passkey
	options, session, err := h.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		slog.Error("failed to begin passkey login", "error", err)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		slog.Error("failed to marshal passkey login session", "error", err)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	flowID, err := h.passkeys.SaveChallenge(r.Context(), nil, "login", sessionJSON)
	if err != nil {
		slog.Error("failed to save passkey login challenge", "error", err)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"flow_id": flowID,
		"options": options,
	})
}

// HandleLoginFinish completes the passkey login flow.
// Verifies the assertion and creates a session on success.
func (h *PasskeyHandler) HandleLoginFinish(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FlowID     string          `json:"flow_id"`
		Credential json.RawMessage `json:"credential"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Retrieve and consume the challenge
	challenge, err := h.passkeys.GetChallenge(r.Context(), body.FlowID)
	if err != nil {
		if errors.Is(err, repo.ErrChallengeNotFound) || errors.Is(err, repo.ErrChallengeExpired) {
			respondError(w, http.StatusBadRequest, "challenge expired — please try again")
			return
		}
		slog.Error("failed to get passkey login challenge", "error", err)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(challenge.SessionData, &session); err != nil {
		slog.Error("failed to unmarshal passkey login session", "error", err)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	// Parse the assertion from the browser
	parsedAssertion, err := protocol.ParseCredentialRequestResponseBytes(body.Credential)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid credential assertion")
		return
	}

	// Discoverable login — look up user by credential ID
	var foundUserID uuid.UUID

	credential, err := h.webAuthn.ValidateDiscoverableLogin(
		func(rawID []byte, userHandle []byte) (webauthn.User, error) {
			pk, err := h.passkeys.GetByCredentialID(r.Context(), rawID)
			if err != nil {
				return nil, err
			}

			foundUserID = pk.UserID

			user, err := h.users.GetByID(r.Context(), pk.UserID)
			if err != nil {
				return nil, err
			}

			// Load all credentials for this user
			allKeys, err := h.passkeys.ListByUserID(r.Context(), pk.UserID)
			if err != nil {
				return nil, err
			}

			waUser := &auth.WebAuthnUser{
				ID:          user.ID[:],
				Name:        user.Email,
				DisplayName: user.DisplayNameOrFull(),
			}

			for _, k := range allKeys {
				waUser.Credentials = append(waUser.Credentials, webauthn.Credential{
					ID:        k.CredentialID,
					PublicKey: k.PublicKey,
					Authenticator: webauthn.Authenticator{
						SignCount: uint32(k.SignCount),
					},
				})
			}

			return waUser, nil
		},
		session,
		parsedAssertion,
	)
	if err != nil {
		slog.Warn("passkey login verification failed", "error", err)
		respondError(w, http.StatusUnauthorized, "passkey verification failed")
		return
	}

	// Update sign count — prevents replay attacks
	if err := h.passkeys.UpdateSignCount(
		r.Context(),
		credential.ID,
		int64(credential.Authenticator.SignCount),
	); err != nil {
		slog.Warn("failed to update passkey sign count", "error", err)
	}

	// Create session
	_, err = h.sessions.Create(r.Context(), w, foundUserID)
	if err != nil {
		slog.Error("failed to create session after passkey login",
			"user_id", foundUserID, "error", err,
		)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	slog.Info("user logged in via passkey", "user_id", foundUserID)

	respond(w, http.StatusOK, map[string]string{
		"message":  "authenticated",
		"redirect": "/dashboard",
	})
}

// HandleDeletePasskey removes a registered passkey.
func (h *PasskeyHandler) HandleDeletePasskey(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		PasskeyID string `json:"passkey_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	passkeyID, err := uuid.Parse(body.PasskeyID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid passkey ID")
		return
	}

	if err := h.passkeys.DeleteCredential(r.Context(), passkeyID, user.ID); err != nil {
		slog.Error("failed to delete passkey", "passkey_id", passkeyID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to remove passkey")
		return
	}

	slog.Info("passkey deleted", "user_id", user.ID, "passkey_id", passkeyID)

	respond(w, http.StatusOK, map[string]string{
		"message": "passkey removed",
	})
}

// transportStrings converts WebAuthn transport types to strings.
func transportStrings(transports []protocol.AuthenticatorTransport) []string {
	result := make([]string, len(transports))
	for i, t := range transports {
		result[i] = string(t)
	}
	return result
}
