// internal/repo/user_repo.go
// All SQL for the users table.
// Rule 42: repo = SQL only. No business logic. No HTTP.
// Rule 8:  Always parameterized queries. Never string interpolation.
// Rule 9:  Column names verified against live schema 2026-06-21.
//          Updated for migration 000005 (google_id, auth_provider).
//          Updated for migration 000006 (profile fields).

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
// SECURITY: This constant is hardcoded — never contains user input.
// It is safe to use with fmt.Sprintf in SQL queries.
// Order must match scanUser exactly.
const userColumns = `
	id, email, password_hash, full_name, phone, whatsapp,
	role, is_verified, google_id, auth_provider, created_at, updated_at,
	username, display_name, bio, avatar_url, website, location,
	agency_name, languages, show_phone, show_whatsapp, notify_email, preferred_lang`

// scanUser scans a row into a User struct.
// Column order must match userColumns exactly.
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
		&user.Username,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarURL,
		&user.Website,
		&user.Location,
		&user.AgencyName,
		&user.Languages,
		&user.ShowPhone,
		&user.ShowWhatsApp,
		&user.NotifyEmail,
		&user.PreferredLang,
	)
	return user, err
}

// Create inserts a new email-registered user.
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
// Sets avatar_url from Google profile picture if provided.
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

// CreateGoogleWithAvatar inserts a new Google user and stores their profile picture.
func (r *UserRepo) CreateGoogleWithAvatar(ctx context.Context, email, fullName, googleID, avatarURL string) (*models.User, error) {
	query := fmt.Sprintf(`
		INSERT INTO users (email, full_name, google_id, auth_provider, is_verified, avatar_url)
		VALUES ($1, $2, $3, 'google', TRUE, $4)
		RETURNING %s`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, email, fullName, googleID, avatarURL))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("user_repo.CreateGoogleWithAvatar: %w", err)
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

// GetByUsername fetches a user by their public username.
// Used for public profile pages /profile/{username}.
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE username = $1`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, username))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetByUsername: %w", err)
	}

	return user, nil
}

// LinkGoogleAccount adds a Google ID to an existing email user.
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

// UpdateProfileInput holds all editable profile fields.
type UpdateProfileInput struct {
	FullName      string
	DisplayName   *string
	Username      *string
	Bio           *string
	Website       *string
	Location      *string
	AgencyName    *string
	Phone         *string
	WhatsApp      *string
	Languages     []string
	ShowPhone     bool
	ShowWhatsApp  bool
	NotifyEmail   bool
	PreferredLang string
}

// UpdateProfile updates all mutable profile fields in one query.
func (r *UserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (*models.User, error) {
	query := fmt.Sprintf(`
		UPDATE users
		SET full_name      = $2,
		    display_name   = $3,
		    username       = $4,
		    bio            = $5,
		    website        = $6,
		    location       = $7,
		    agency_name    = $8,
		    phone          = $9,
		    whatsapp       = $10,
		    languages      = $11,
		    show_phone     = $12,
		    show_whatsapp  = $13,
		    notify_email   = $14,
		    preferred_lang = $15
		WHERE id = $1
		RETURNING %s`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query,
		id,
		input.FullName,
		input.DisplayName,
		input.Username,
		input.Bio,
		input.Website,
		input.Location,
		input.AgencyName,
		input.Phone,
		input.WhatsApp,
		input.Languages,
		input.ShowPhone,
		input.ShowWhatsApp,
		input.NotifyEmail,
		input.PreferredLang,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("user_repo.UpdateProfile: %w", err)
	}

	return user, nil
}

// UpdateAvatar updates the user's avatar URL.
func (r *UserRepo) UpdateAvatar(ctx context.Context, id uuid.UUID, avatarURL string) (*models.User, error) {
	query := fmt.Sprintf(`
		UPDATE users
		SET avatar_url = $2
		WHERE id = $1
		RETURNING %s`, userColumns)

	user, err := scanUser(r.db.QueryRow(ctx, query, id, avatarURL))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.UpdateAvatar: %w", err)
	}

	return user, nil
}

// isUniqueViolation returns true if the error is a PostgreSQL
// unique constraint violation (error code 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
