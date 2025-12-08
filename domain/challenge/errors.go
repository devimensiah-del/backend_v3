package challenge

import (
	"errors"
	"fmt"
)

// =============================================================================
// DOMAIN ERRORS
// =============================================================================

// Sentinel errors for the challenge domain.
// These provide stable error types that callers can check with errors.Is().
var (
	ErrNotFound           = errors.New("challenge not found")
	ErrValidation         = errors.New("validation error")
	ErrServiceUnavailable = errors.New("service temporarily unavailable")
)

// =============================================================================
// VALIDATION ERRORS
// =============================================================================

// ValidationError provides detailed validation failure information.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

// NewValidationError creates a new validation error for a specific field.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// =============================================================================
// REPOSITORY ERRORS
// =============================================================================

// RepositoryError wraps database errors without exposing internals.
type RepositoryError struct {
	Op  string // Operation that failed (e.g., "create", "get_by_id")
	Err error  // Original error (not exposed in Error())
}

func (e *RepositoryError) Error() string {
	return fmt.Sprintf("challenge %s failed", e.Op)
}

func (e *RepositoryError) Unwrap() error {
	// Return sentinel error based on operation, not the raw DB error
	if e.Op == "get_by_id" || e.Op == "list_by_company" {
		return ErrNotFound
	}
	return ErrServiceUnavailable
}

// Internal returns the original error for logging (not for API responses).
func (e *RepositoryError) Internal() error {
	return e.Err
}

// NewRepositoryError wraps a database error with operation context.
func NewRepositoryError(op string, err error) *RepositoryError {
	return &RepositoryError{Op: op, Err: err}
}

// =============================================================================
// ERROR HELPERS
// =============================================================================

// IsNotFound checks if an error indicates a challenge was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsValidation checks if an error is a validation error.
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}
