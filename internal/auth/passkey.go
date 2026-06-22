// internal/auth/passkey.go
// WebAuthn/FIDO2 passkey configuration and helper types.
// Uses go-webauthn/webauthn — the standard Go WebAuthn library.

package auth

import (
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
)

// NewWebAuthn creates a configured WebAuthn instance.
// rpID must match the domain exactly — security requirement.
// rpOrigins must include all valid origins (https://domain.com).
func NewWebAuthn(rpID, rpDisplayName string, rpOrigins []string) (*webauthn.WebAuthn, error) {
	cfg := &webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     rpOrigins,

		// Timeout for user interaction (60 seconds)
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    60000,
				TimeoutUVD: 60000,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    60000,
				TimeoutUVD: 60000,
			},
		},
	}

	w, err := webauthn.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("auth.NewWebAuthn: %w", err)
	}

	return w, nil
}

// WebAuthnUser implements the webauthn.User interface.
// Required by the go-webauthn library for credential operations.
type WebAuthnUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte                         { return u.ID }
func (u *WebAuthnUser) WebAuthnName() string                       { return u.Name }
func (u *WebAuthnUser) WebAuthnDisplayName() string                { return u.DisplayName }
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }
func (u *WebAuthnUser) WebAuthnIcon() string                       { return "" }
