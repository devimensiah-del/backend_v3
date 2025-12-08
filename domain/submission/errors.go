package submission

import (
	"errors"
	"fmt"
)

// =============================================================================
// DOMAIN ERRORS
// =============================================================================

// Sentinel errors for the submission domain.
// These provide stable error types that callers can check with errors.Is().
var (
	ErrNotFound           = errors.New("submission not found")
	ErrValidation         = errors.New("validation error")
	ErrAlreadyExists      = errors.New("submission already exists")
	ErrServiceUnavailable = errors.New("service temporarily unavailable")
	ErrCompanyCreation    = errors.New("failed to create company")
	ErrChallengeCreation  = errors.New("failed to create challenge")
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
// WRAPPED ERRORS (hide internal details)
// =============================================================================

// RepositoryError wraps database errors without exposing internals.
type RepositoryError struct {
	Op  string // Operation that failed (e.g., "create", "get_by_id")
	Err error  // Original error (not exposed in Error())
}

func (e *RepositoryError) Error() string {
	return fmt.Sprintf("submission %s failed", e.Op)
}

func (e *RepositoryError) Unwrap() error {
	// Return sentinel error based on operation, not the raw DB error
	if e.Op == "get_by_id" || e.Op == "get_by_company_id" {
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
// WORKFLOW ERRORS
// =============================================================================

// WorkflowError wraps errors from multi-step workflows.
type WorkflowError struct {
	Step string // Which step failed
	Err  error  // Underlying error
}

func (e *WorkflowError) Error() string {
	return fmt.Sprintf("%s failed", e.Step)
}

func (e *WorkflowError) Unwrap() error {
	return e.Err
}

// Internal returns the original error for logging.
func (e *WorkflowError) Internal() error {
	return e.Err
}

// NewWorkflowError creates a workflow error for a specific step.
func NewWorkflowError(step string, err error) *WorkflowError {
	return &WorkflowError{Step: step, Err: err}
}

// =============================================================================
// ERROR HELPERS
// =============================================================================

// IsNotFound checks if an error indicates a submission was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsValidation checks if an error is a validation error.
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}
