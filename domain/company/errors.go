package company

import (
	"fmt"

	"github.com/google/uuid"
)

// =============================================================================
// DUPLICATE DETECTION ERRORS
// =============================================================================

// DuplicateSubmitterError indicates the same email already submitted for this company.
// This is returned when a user tries to submit for a company they've already submitted for.
type DuplicateSubmitterError struct {
	CompanyID   uuid.UUID
	CompanyName string
	Email       string
}

func (e *DuplicateSubmitterError) Error() string {
	return fmt.Sprintf("email %s already submitted for company %s (%s)",
		e.Email, e.CompanyName, e.CompanyID)
}

// IsDuplicateSubmitterError checks if an error is a DuplicateSubmitterError.
func IsDuplicateSubmitterError(err error) bool {
	_, ok := err.(*DuplicateSubmitterError)
	return ok
}

// CNPJCollisionError indicates a CNPJ discovered during enrichment matches another company.
// This is returned during Step 2 enrichment when we discover a CNPJ that already exists.
type CNPJCollisionError struct {
	CurrentCompanyID    uuid.UUID
	ExistingCompanyID   uuid.UUID
	ExistingCompanyName string
	DiscoveredCNPJ      string
}

func (e *CNPJCollisionError) Error() string {
	return fmt.Sprintf("discovered CNPJ %s belongs to existing company %s (%s)",
		e.DiscoveredCNPJ, e.ExistingCompanyName, e.ExistingCompanyID)
}

// IsCNPJCollisionError checks if an error is a CNPJCollisionError.
func IsCNPJCollisionError(err error) bool {
	_, ok := err.(*CNPJCollisionError)
	return ok
}

// =============================================================================
// MERGE ERRORS
// =============================================================================

// MergeError indicates a failure during company merge operation.
type MergeError struct {
	TargetID uuid.UUID
	SourceID uuid.UUID
	Reason   string
}

func (e *MergeError) Error() string {
	return fmt.Sprintf("failed to merge company %s into %s: %s",
		e.SourceID, e.TargetID, e.Reason)
}
