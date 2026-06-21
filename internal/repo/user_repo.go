// internal/repo/user_repo.go
// All SQL for the users table.
// Rule 42: repo = SQL only. No business logic. No HTTP.
// Rule 8:  Always parameterized queries. Never string interpolation.
// Rule 9:  Column names verified against live schema 2026-06-20.
//          Updated for migration 000005 (google_id, auth_provider).

package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vallescentrales/internal/models"
)

// UserRepo handles all database operations for the users table.
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

// userColumns is the canonical column list for all user queries.
// Defined once to prevent drift between SELECT and RETURNING clauses.
const userColumns = `id, email, password_hash, full_name, phone, whatsapp,
	role, is_verified, google_id, auth_provider, created_at, updated_at`

// scanUser scans a row into a User struct.
// Every query that returns a user must use this — no duplicate scan logic.
func scanUser(row pgx.Row) (*models.User, error) {
	user := &models.User{}
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Phone,
		&user.WhatsApp,
		&user.Role,
		&user.IsVerified,
		&user.GoogleID,
		&user.AuthProvider,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
}

// Create inserts a new email-registered user and returns the created record.
// password_hash must already be an Argon2id hash — never raw password.
func (r *UserRepo) Create(ctx context.Context, email, passwordHash, fullName string) (*models.User, error) {
	query := fmt.Sprintf(`
		INSERT INTO users (email, password_hash, full_name, auth_provider)
		VALUES ($1, $2, $3, 'email')
		RETURNING %s`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, email, passwordHash, fullName))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("user_repo.Create: %w", err)
	}

	return user, nil
}

// CreateGoogle inserts a new Google-registered user.
// No password is set — auth_provider is 'google'.
// is_verified is true because Google verified the email.
func (r *UserRepo) CreateGoogle(ctx context.Context, email, fullName, googleID string) (*models.User, error) {
	query := fmt.Sprintf(`
		INSERT INTO users (email, full_name, google_id, auth_provider, is_verified)
		VALUES ($1, $2, $3, 'google', TRUE)
		RETURNING %s`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, email, fullName, googleID))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("user_repo.CreateGoogle: %w", err)
	}

	return user, nil
}

// GetByID fetches a user by primary key.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE id = $1`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetByID: %w", err)
	}

	return user, nil
}

// GetByEmail fetches a user by email address.
// Used during login to retrieve the stored password hash.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE email = $1`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetByEmail: %w", err)
	}

	return user, nil
}

// GetByGoogleID fetches a user by their Google account ID.
// Used during Google OAuth callback to find existing users.
func (r *UserRepo) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE google_id = $1`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, googleID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetByGoogleID: %w", err)
	}

	return user, nil
}

// LinkGoogleAccount adds a Google ID to an existing email-registered user.
// Used when a user who registered with email later signs in with Google.
func (r *UserRepo) LinkGoogleAccount(ctx context.Context, userID uuid.UUID, googleID string) (*models.User, error) {
	query := fmt.Sprintf(`
		UPDATE users
		SET google_id = $2, is_verified = TRUE
		WHERE id = $1
		RETURNING %s`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, userID, googleID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.LinkGoogleAccount: %w", err)
	}

	return user, nil
}

// UpdateProfile updates mutable profile fields.
func (r *UserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, fullName string, phone, whatsapp *string) (*models.User, error) {
	query := fmt.Sprintf(`
		UPDATE users
		SET full_name = $2, phone = $3, whatsapp = $4
		WHERE id = $1
		RETURNING %s`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, id, fullName, phone, whatsapp))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.UpdateProfile: %w", err)
	}

	return user, nil
}

// isUniqueViolation returns true if the error is a PostgreSQL
// unique constraint violation (error code 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
