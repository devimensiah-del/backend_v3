package challenge

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallenge_Validate(t *testing.T) {
	validCompanyID := uuid.New()

	tests := []struct {
		name      string
		challenge *Challenge
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid challenge",
			challenge: &Challenge{
				CompanyID:         validCompanyID,
				ChallengeCategory: CategoryGrowth,
				ChallengeType:     TypeGrowthOrganic,
				BusinessChallenge: "We need to grow revenue by 50%",
			},
			wantErr: false,
		},
		{
			name: "missing company_id",
			challenge: &Challenge{
				CompanyID:         uuid.Nil,
				ChallengeCategory: CategoryGrowth,
				ChallengeType:     TypeGrowthOrganic,
				BusinessChallenge: "Test challenge",
			},
			wantErr: true,
			errMsg:  "company_id is required",
		},
		{
			name: "missing category",
			challenge: &Challenge{
				CompanyID:         validCompanyID,
				ChallengeCategory: "",
				ChallengeType:     TypeGrowthOrganic,
				BusinessChallenge: "Test challenge",
			},
			wantErr: true,
			errMsg:  "challenge_category is required",
		},
		{
			name: "invalid category",
			challenge: &Challenge{
				CompanyID:         validCompanyID,
				ChallengeCategory: "invalid_category",
				ChallengeType:     TypeGrowthOrganic,
				BusinessChallenge: "Test challenge",
			},
			wantErr: true,
			errMsg:  "challenge_category must be one of",
		},
		{
			name: "missing type",
			challenge: &Challenge{
				CompanyID:         validCompanyID,
				ChallengeCategory: CategoryGrowth,
				ChallengeType:     "",
				BusinessChallenge: "Test challenge",
			},
			wantErr: true,
			errMsg:  "challenge_type is required",
		},
		{
			name: "invalid type",
			challenge: &Challenge{
				CompanyID:         validCompanyID,
				ChallengeCategory: CategoryGrowth,
				ChallengeType:     "invalid_type",
				BusinessChallenge: "Test challenge",
			},
			wantErr: true,
			errMsg:  "invalid challenge_type",
		},
		{
			name: "missing business_challenge",
			challenge: &Challenge{
				CompanyID:         validCompanyID,
				ChallengeCategory: CategoryGrowth,
				ChallengeType:     TypeGrowthOrganic,
				BusinessChallenge: "",
			},
			wantErr: true,
			errMsg:  "business_challenge is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.challenge.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewChallenge(t *testing.T) {
	companyID := uuid.New()

	challenge := NewChallenge(
		companyID,
		CategoryTransform,
		TypeTransformDigital,
		"We need digital transformation",
	)

	require.NotNil(t, challenge)
	assert.NotEqual(t, uuid.Nil, challenge.ID)
	assert.Equal(t, companyID, challenge.CompanyID)
	assert.Equal(t, CategoryTransform, challenge.ChallengeCategory)
	assert.Equal(t, TypeTransformDigital, challenge.ChallengeType)
	assert.Equal(t, "We need digital transformation", challenge.BusinessChallenge)
	assert.False(t, challenge.CreatedAt.IsZero())
	assert.False(t, challenge.UpdatedAt.IsZero())
	assert.Nil(t, challenge.DeletedAt)
}

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		category string
		want     bool
	}{
		{"growth", true},
		{"transform", true},
		{"transition", true},
		{"compete", true},
		{"funding", true},
		{"invalid", false},
		{"Growth", false}, // case sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := IsValidCategory(tt.category)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidType(t *testing.T) {
	tests := []struct {
		category string
		typ      string
		want     bool
	}{
		// Growth types
		{"growth", "growth_organic", true},
		{"growth", "growth_geographic", true},
		{"growth", "growth_segment", true},
		{"growth", "growth_product", true},
		{"growth", "growth_channel", true},
		{"growth", "transform_digital", false}, // wrong category

		// Transform types
		{"transform", "transform_digital", true},
		{"transform", "transform_model", true},
		{"transform", "transform_culture", true},
		{"transform", "transform_operational", true},
		{"transform", "growth_organic", false}, // wrong category

		// Transition types
		{"transition", "transition_succession", true},
		{"transition", "transition_exit", true},
		{"transition", "transition_merger", true},
		{"transition", "transition_turnaround", true},

		// Compete types
		{"compete", "compete_differentiate", true},
		{"compete", "compete_defend", true},
		{"compete", "compete_reposition", true},

		// Funding types
		{"funding", "funding_raise", true},
		{"funding", "funding_debt", true},
		{"funding", "funding_ipo", true},

		// Invalid cases
		{"invalid", "growth_organic", false},
		{"growth", "invalid_type", false},
		{"", "growth_organic", false},
	}

	for _, tt := range tests {
		t.Run(tt.category+"_"+tt.typ, func(t *testing.T) {
			got := IsValidType(tt.category, tt.typ)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidTypeAny(t *testing.T) {
	tests := []struct {
		typ  string
		want bool
	}{
		{"growth_organic", true},
		{"transform_digital", true},
		{"transition_exit", true},
		{"compete_defend", true},
		{"funding_ipo", true},
		{"invalid_type", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			got := IsValidTypeAny(tt.typ)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChallenge_IsDeleted(t *testing.T) {
	challenge := &Challenge{}
	assert.False(t, challenge.IsDeleted())

	now := challenge.CreatedAt
	challenge.DeletedAt = &now
	assert.True(t, challenge.IsDeleted())
}

func TestValidCategories(t *testing.T) {
	expected := []ChallengeCategory{
		CategoryGrowth,
		CategoryTransform,
		CategoryTransition,
		CategoryCompete,
		CategoryFunding,
	}

	assert.Equal(t, expected, ValidCategories)
	assert.Len(t, ValidCategories, 5)
}

func TestValidTypesByCategory(t *testing.T) {
	// Ensure all categories are covered
	assert.Len(t, ValidTypesByCategory, 5)

	// Check growth has 5 types
	assert.Len(t, ValidTypesByCategory[CategoryGrowth], 5)

	// Check transform has 4 types
	assert.Len(t, ValidTypesByCategory[CategoryTransform], 4)

	// Check transition has 4 types
	assert.Len(t, ValidTypesByCategory[CategoryTransition], 4)

	// Check compete has 3 types
	assert.Len(t, ValidTypesByCategory[CategoryCompete], 3)

	// Check funding has 3 types
	assert.Len(t, ValidTypesByCategory[CategoryFunding], 3)
}
