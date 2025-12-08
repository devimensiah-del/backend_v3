package challenge

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Challenge represents a strategic business challenge for a company
// Multiple challenges can exist per company over time
// Each challenge results in one or more analyses
//
// NOTE: Contact info is NOT stored on Challenge. It stays on Submission (the entry point).
// For re-analyze scenarios (no new submission), there is no "requester" to track.
type Challenge struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CompanyID uuid.UUID `json:"company_id" db:"company_id"`

	// Challenge definition (required fields)
	ChallengeCategory ChallengeCategory `json:"challenge_category" db:"challenge_category"`
	ChallengeType     ChallengeType     `json:"challenge_type" db:"challenge_type"`
	BusinessChallenge string            `json:"business_challenge" db:"business_challenge"` // Free-text context

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ChallengeCategory represents the top-level challenge category
type ChallengeCategory string

const (
	CategoryGrowth    ChallengeCategory = "growth"
	CategoryTransform ChallengeCategory = "transform"
	CategoryCompete   ChallengeCategory = "compete"
)

// ChallengeType represents the specific challenge type
type ChallengeType string

// Growth types
const (
	TypeGrowthOrganic    ChallengeType = "growth_organic"
	TypeGrowthGeographic ChallengeType = "growth_geographic"
	TypeGrowthSegment    ChallengeType = "growth_segment"
	TypeGrowthProduct    ChallengeType = "growth_product"
	TypeGrowthChannel    ChallengeType = "growth_channel"
)

// Transform types
const (
	TypeTransformDigital ChallengeType = "transform_digital"
	TypeTransformModel   ChallengeType = "transform_model"
)

// Compete types
const (
	TypeCompeteDifferentiate ChallengeType = "compete_differentiate"
	TypeCompeteDefend        ChallengeType = "compete_defend"
	TypeCompeteReposition    ChallengeType = "compete_reposition"
)

// NewChallenge creates a new challenge with default values
func NewChallenge(
	companyID uuid.UUID,
	category ChallengeCategory,
	challengeType ChallengeType,
	businessChallenge string,
) *Challenge {
	now := time.Now()
	return &Challenge{
		ID:                uuid.New(),
		CompanyID:         companyID,
		ChallengeCategory: category,
		ChallengeType:     challengeType,
		BusinessChallenge: businessChallenge,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// Validate validates the challenge fields
func (c *Challenge) Validate() error {
	if c.CompanyID == uuid.Nil {
		return errors.New("company_id is required")
	}
	if c.ChallengeCategory == "" {
		return errors.New("challenge_category is required")
	}
	if !isValidCategory(c.ChallengeCategory) {
		return errors.New("challenge_category must be one of: growth, transform, compete")
	}
	if c.ChallengeType == "" {
		return errors.New("challenge_type is required")
	}
	if !isValidType(c.ChallengeType) {
		return errors.New("invalid challenge_type for category")
	}
	if c.BusinessChallenge == "" {
		return errors.New("business_challenge is required")
	}
	return nil
}

// IsDeleted returns true if the challenge has been soft deleted
func (c *Challenge) IsDeleted() bool {
	return c.DeletedAt != nil
}

// =============================================================================
// VALIDATION HELPERS (exported for use by other domains)
// =============================================================================

// ValidCategories returns all valid challenge categories
var ValidCategories = []ChallengeCategory{
	CategoryGrowth, CategoryTransform, CategoryCompete,
}

// ValidTypesByCategory maps each category to its valid types
var ValidTypesByCategory = map[ChallengeCategory][]ChallengeType{
	CategoryGrowth:    {TypeGrowthOrganic, TypeGrowthGeographic, TypeGrowthSegment, TypeGrowthProduct, TypeGrowthChannel},
	CategoryTransform: {TypeTransformDigital, TypeTransformModel},
	CategoryCompete:   {TypeCompeteDifferentiate, TypeCompeteDefend, TypeCompeteReposition},
}

// IsValidCategory checks if a category string is valid
func IsValidCategory(category string) bool {
	for _, valid := range ValidCategories {
		if ChallengeCategory(category) == valid {
			return true
		}
	}
	return false
}

// IsValidType checks if a type string is valid for the given category
func IsValidType(category, challengeType string) bool {
	validTypes, ok := ValidTypesByCategory[ChallengeCategory(category)]
	if !ok {
		return false
	}
	for _, valid := range validTypes {
		if ChallengeType(challengeType) == valid {
			return true
		}
	}
	return false
}

// IsValidTypeAny checks if a type string is valid (regardless of category)
func IsValidTypeAny(challengeType string) bool {
	for _, types := range ValidTypesByCategory {
		for _, valid := range types {
			if ChallengeType(challengeType) == valid {
				return true
			}
		}
	}
	return false
}

// internal helpers (keep for model.Validate())
func isValidCategory(category ChallengeCategory) bool {
	return IsValidCategory(string(category))
}

func isValidType(challengeType ChallengeType) bool {
	return IsValidTypeAny(string(challengeType))
}
