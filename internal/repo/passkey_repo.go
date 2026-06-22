// internal/repo/passkey_repo.go
// All SQL for user_passkeys and passkey_challenges tables.
// Rule 42: repo = SQL only. No business logic. No HTTP.
// Rule 8:  Always parameterized queries.

package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrChallengeExpired is returned when a passkey challenge has expired.
var ErrChallengeExpired = errors.New("passkey challenge expired")

// ErrChallengeNotFound is returned when a challenge flow_id does not exist.
var ErrChallengeNotFound = errors.New("passkey challenge not found")

// Passkey represents a stored WebAuthn credential.
type Passkey struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	AAGUID          []byte
	SignCount       int64
	DeviceName      *string
	Transports      []string
	CreatedAt       time.Time
	LastUsedAt      *time.Time
}

// PasskeyChallenge represents a short-lived WebAuthn challenge.
type PasskeyChallenge struct {
	ID          uuid.UUID
	FlowID      string
	UserID      *uuid.UUID
	FlowType    string
	SessionData []byte
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

// PasskeyRepo handles all database operations for passkey tables.
type PasskeyRepo struct {
	db *pgxpool.Pool
}

// NewPasskeyRepo creates a new PasskeyRepo.
func NewPasskeyRepo(db *pgxpool.Pool) *PasskeyRepo {
	return &PasskeyRepo{db: db}
}

// SaveCredential stores a new passkey credential after successful registration.
func (r *PasskeyRepo) SaveCredential(ctx context.Context,
	userID uuid.UUID,
	credentialID []byte,
	publicKey []byte,
	attestationType string,
	aaguid []byte,
	transports []string,
	deviceName *string,
) (*Passkey, error) {
	query := `
		INSERT INTO user_passkeys
			(user_id, credential_id, public_key, attestation_type, aaguid, transports, device_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, credential_id, public_key, attestation_type,
		          aaguid, sign_count, device_name, transports, created_at, last_used_at`

	pk := &Passkey{}
	err := r.db.QueryRow(ctx, query,
		userID,
		credentialID,
		publicKey,
		attestationType,
		aaguid,
		transports,
		deviceName,
	).Scan(
		&pk.ID,
		&pk.UserID,
		&pk.CredentialID,
		&pk.PublicKey,
		&pk.AttestationType,
		&pk.AAGUID,
		&pk.SignCount,
		&pk.DeviceName,
		&pk.Transports,
		&pk.CreatedAt,
		&pk.LastUsedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("passkey_repo.SaveCredential: %w", err)
	}

	// Mark user as having passkeys enabled
	_, err = r.db.Exec(ctx,
		`UPDATE users SET passkeys_enabled = TRUE WHERE id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("passkey_repo.SaveCredential: enable flag: %w", err)
	}

	return pk, nil
}

// GetByCredentialID fetches a passkey by its credential ID.
// Used during login to find the matching stored credential.
func (r *PasskeyRepo) GetByCredentialID(ctx context.Context, credentialID []byte) (*Passkey, error) {
	query := `
		SELECT id, user_id, credential_id, public_key, attestation_type,
		       aaguid, sign_count, device_name, transports, created_at, last_used_at
		FROM user_passkeys
		WHERE credential_id = $1`

	pk := &Passkey{}
	err := r.db.QueryRow(ctx, query, credentialID).Scan(
		&pk.ID,
		&pk.UserID,
		&pk.CredentialID,
		&pk.PublicKey,
		&pk.AttestationType,
		&pk.AAGUID,
		&pk.SignCount,
		&pk.DeviceName,
		&pk.Transports,
		&pk.CreatedAt,
		&pk.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("passkey_repo.GetByCredentialID: %w", err)
	}

	return pk, nil
}

// ListByUserID returns all passkeys registered for a user.
func (r *PasskeyRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Passkey, error) {
	query := `
		SELECT id, user_id, credential_id, public_key, attestation_type,
		       aaguid, sign_count, device_name, transports, created_at, last_used_at
		FROM user_passkeys
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("passkey_repo.ListByUserID: %w", err)
	}
	defer rows.Close()

	var passkeys []*Passkey
	for rows.Next() {
		pk := &Passkey{}
		err := rows.Scan(
			&pk.ID,
			&pk.UserID,
			&pk.CredentialID,
			&pk.PublicKey,
			&pk.AttestationType,
			&pk.AAGUID,
			&pk.SignCount,
			&pk.DeviceName,
			&pk.Transports,
			&pk.CreatedAt,
			&pk.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("passkey_repo.ListByUserID scan: %w", err)
		}
		passkeys = append(passkeys, pk)
	}

	return passkeys, rows.Err()
}

// UpdateSignCount updates the signature counter after successful authentication.
// Prevents credential cloning/replay attacks.
func (r *PasskeyRepo) UpdateSignCount(ctx context.Context, credentialID []byte, signCount int64) error {
	query := `
		UPDATE user_passkeys
		SET sign_count   = $2,
		    last_used_at = NOW()
		WHERE credential_id = $1`

	result, err := r.db.Exec(ctx, query, credentialID, signCount)
	if err != nil {
		return fmt.Errorf("passkey_repo.UpdateSignCount: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteCredential removes a passkey by ID.
// Disables passkeys_enabled if user has no remaining passkeys.
func (r *PasskeyRepo) DeleteCredential(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_passkeys WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("passkey_repo.DeleteCredential: %w", err)
	}

	// Check if user has any remaining passkeys
	var count int
	err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_passkeys WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("passkey_repo.DeleteCredential: count check: %w", err)
	}

	if count == 0 {
		_, err = r.db.Exec(ctx,
			`UPDATE users SET passkeys_enabled = FALSE WHERE id = $1`, userID)
		if err != nil {
			return fmt.Errorf("passkey_repo.DeleteCredential: disable flag: %w", err)
		}
	}

	return nil
}

// SaveChallenge stores a WebAuthn challenge for a registration or login flow.
func (r *PasskeyRepo) SaveChallenge(ctx context.Context,
	userID *uuid.UUID,
	flowType string,
	sessionData []byte,
) (string, error) {
	flowID, err := generateFlowID()
	if err != nil {
		return "", fmt.Errorf("passkey_repo.SaveChallenge: %w", err)
	}

	query := `
		INSERT INTO passkey_challenges (flow_id, user_id, flow_type, session_data)
		VALUES ($1, $2, $3, $4)`

	_, err = r.db.Exec(ctx, query, flowID, userID, flowType, sessionData)
	if err != nil {
		return "", fmt.Errorf("passkey_repo.SaveChallenge: %w", err)
	}

	return flowID, nil
}

// GetChallenge fetches and deletes a challenge by flow_id.
// Challenges are single-use — deleted on retrieval.
func (r *PasskeyRepo) GetChallenge(ctx context.Context, flowID string) (*PasskeyChallenge, error) {
	query := `
		DELETE FROM passkey_challenges
		WHERE flow_id = $1
		RETURNING id, flow_id, user_id, flow_type, session_data, expires_at, created_at`

	ch := &PasskeyChallenge{}
	err := r.db.QueryRow(ctx, query, flowID).Scan(
		&ch.ID,
		&ch.FlowID,
		&ch.UserID,
		&ch.FlowType,
		&ch.SessionData,
		&ch.ExpiresAt,
		&ch.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("passkey_repo.GetChallenge: %w", err)
	}

	if time.Now().After(ch.ExpiresAt) {
		return nil, ErrChallengeExpired
	}

	return ch, nil
}

// DeleteExpiredChallenges removes expired challenge records.
func (r *PasskeyRepo) DeleteExpiredChallenges(ctx context.Context) (int64, error) {
	result, err := r.db.Exec(ctx,
		`DELETE FROM passkey_challenges WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("passkey_repo.DeleteExpiredChallenges: %w", err)
	}
	return result.RowsAffected(), nil
}

// generateFlowID creates a cryptographically secure random flow identifier.
func generateFlowID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateFlowID: %w", err)
	}
	return hex.EncodeToString(b), nil
}
