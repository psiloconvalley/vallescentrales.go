// internal/repo/errors.go
// Shared sentinel errors for the repo layer.
// All repo files use these — never redeclare them.

package repo

import "errors"

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("record not found")

// ErrEmailTaken is returned when a user email already exists.
var ErrEmailTaken = errors.New("email already taken")

// ErrSlugTaken is returned when a listing slug already exists.
var ErrSlugTaken = errors.New("slug already taken")
