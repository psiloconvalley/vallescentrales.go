// internal/auth/password.go
// Argon2id password hashing and verification.
// ADR-003: Argon2id is the standard. bcrypt is never used.
// Parameters meet OWASP 2024 minimum recommendations.

package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/crypto/argon2"
)

// ErrInvalidPassword is returned when a password does not match the hash.
var ErrInvalidPassword = errors.New("invalid password")

// ErrInvalidHash is returned when a stored hash is malformed.
var ErrInvalidHash = errors.New("invalid password hash format")

// ErrPasswordTooShort is returned when a password is below minimum length.
var ErrPasswordTooShort = errors.New("password must be at least 12 characters")

// ErrPasswordTooLong is returned when a password exceeds maximum length.
// Argon2id with very long inputs is a DoS vector.
var ErrPasswordTooLong = errors.New("password must be 72 characters or fewer")

const (
	minPasswordLength = 12
	maxPasswordLength = 72
)

// argon2Params defines the cost parameters.
// ADR-003: memory=64MB, iterations=3, parallelism=4, keyLen=32.
type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// defaultParams are the OWASP-recommended Argon2id parameters.
var defaultParams = &argon2Params{
	memory:      64 * 1024, // 64MB
	iterations:  3,
	parallelism: 4,
	saltLength:  16,
	keyLength:   32,
}

// validatePasswordLength enforces min/max length before hashing.
func validatePasswordLength(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > maxPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}

// HashPassword hashes a plaintext password using Argon2id.
// Validates length before hashing — never hashes invalid input.
// Returns a formatted string that includes all parameters and salt.
// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
func HashPassword(password string) (string, error) {
	if err := validatePasswordLength(password); err != nil {
		return "", err
	}

	salt := make([]byte, defaultParams.saltLength)
	if _, err := rand.Read(salt); err != nil {
		slog.Error("failed to generate argon2id salt", "error", err)
		return "", fmt.Errorf("auth.HashPassword: failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		defaultParams.iterations,
		defaultParams.memory,
		defaultParams.parallelism,
		defaultParams.keyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		defaultParams.memory,
		defaultParams.iterations,
		defaultParams.parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// VerifyPassword checks a plaintext password against a stored Argon2id hash.
// Returns nil if the password matches, ErrInvalidPassword if it does not.
// Always runs constant-time comparison to prevent timing attacks.
func VerifyPassword(password, encodedHash string) error {
	// Still validate length — reject obviously invalid input fast
	// but do NOT reveal which check failed to the caller
	if len(password) < minPasswordLength || len(password) > maxPasswordLength {
		return ErrInvalidPassword
	}

	p, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		slog.Error("failed to decode password hash", "error", err)
		return ErrInvalidHash
	}

	comparisonHash := argon2.IDKey(
		[]byte(password),
		salt,
		p.iterations,
		p.memory,
		p.parallelism,
		p.keyLength,
	)

	// Constant-time comparison — prevents timing attacks
	if subtle.ConstantTimeCompare(hash, comparisonHash) != 1 {
		return ErrInvalidPassword
	}

	return nil
}

// decodeHash parses an encoded Argon2id hash string into its components.
func decodeHash(encodedHash string) (*argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	// Expected: ["", "argon2id", "v=19", "m=65536,t=3,p=4", "<salt>", "<hash>"]
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}

	p := &argon2Params{}
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	p.keyLength = uint32(len(hash))
	p.saltLength = uint32(len(salt))

	return p, salt, hash, nil
}
