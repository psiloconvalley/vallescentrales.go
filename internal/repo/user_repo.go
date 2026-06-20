// internal/repo/user_repo.go
// All SQL for the users table.
// Rule 42: repo = SQL only. No business logic. No HTTP.
// Rule 8:  Always parameterized queries. Never string interpolation.
// Rule 9:  Column names verified against live schema 2026-06-20.

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

// Create inserts a new user and returns the created record.
// password_hash must already be an Argon2id hash — never raw password.
func (r *UserRepo) Create(ctx context.Context, email, passwordHash, fullName string) (*models.User, error) {
	query := `
		INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, full_name, phone, whatsapp,
		          role, is_verified, created_at, updated_at`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, email, passwordHash, fullName).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Phone,
		&user.WhatsApp,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("user_repo.Create: %w", err)
	}

	return user, nil
}

// GetByID fetches a user by primary key.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, phone, whatsapp,
		       role, is_verified, created_at, updated_at
		FROM users
		WHERE id = $1`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Phone,
		&user.WhatsApp,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
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
	query := `
		SELECT id, email, password_hash, full_name, phone, whatsapp,
		       role, is_verified, created_at, updated_at
		FROM users
		WHERE email = $1`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Phone,
		&user.WhatsApp,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetByEmail: %w", err)
	}

	return user, nil
}

// UpdateProfile updates mutable profile fields.
func (r *UserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, fullName string, phone, whatsapp *string) (*models.User, error) {
	query := `
		UPDATE users
		SET full_name = $2,
		    phone     = $3,
		    whatsapp  = $4
		WHERE id = $1
		RETURNING id, email, password_hash, full_name, phone, whatsapp,
		          role, is_verified, created_at, updated_at`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, id, fullName, phone, whatsapp).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Phone,
		&user.WhatsApp,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
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
