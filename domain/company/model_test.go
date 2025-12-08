package company

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompany_Validate(t *testing.T) {
	tests := []struct {
		name    string
		company *Company
		wantErr error
	}{
		{
			name:    "valid company",
			company: &Company{Name: "Test Corp"},
			wantErr: nil,
		},
		{
			name:    "missing name",
			company: &Company{Name: ""},
			wantErr: ErrNameRequired,
		},
		{
			name:    "name too long",
			company: &Company{Name: string(make([]byte, MaxNameLength+1))},
			wantErr: ErrNameTooLong,
		},
		{
			name: "revenue min greater than max",
			company: &Company{
				Name:             "Test Corp",
				AnnualRevenueMin: ptrFloat64(100000),
				AnnualRevenueMax: ptrFloat64(50000),
			},
			wantErr: ErrRevenueRangeInvalid,
		},
		{
			name: "negative revenue min",
			company: &Company{
				Name:             "Test Corp",
				AnnualRevenueMin: ptrFloat64(-1000),
			},
			wantErr: ErrRevenueNegative,
		},
		{
			name: "negative revenue max",
			company: &Company{
				Name:             "Test Corp",
				AnnualRevenueMax: ptrFloat64(-500),
			},
			wantErr: ErrRevenueNegative,
		},
		{
			name: "website too long",
			company: &Company{
				Name:    "Test Corp",
				Website: ptrString(string(make([]byte, MaxWebsiteLength+1))),
			},
			wantErr: ErrWebsiteTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.company.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompany_IsOwner(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name    string
		company *Company
		userID  uuid.UUID
		want    bool
	}{
		{
			name:    "is owner",
			company: &Company{OwnerID: &ownerID},
			userID:  ownerID,
			want:    true,
		},
		{
			name:    "not owner",
			company: &Company{OwnerID: &ownerID},
			userID:  otherID,
			want:    false,
		},
		{
			name:    "no owner set",
			company: &Company{OwnerID: nil},
			userID:  ownerID,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.company.IsOwner(tt.userID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompany_IsUserAllowed(t *testing.T) {
	ownerID := uuid.New()
	allowedID := uuid.New()
	notAllowedID := uuid.New()

	company := &Company{
		OwnerID:      &ownerID,
		AllowedUsers: UUIDSlice{allowedID},
	}

	assert.True(t, company.IsUserAllowed(ownerID), "owner should be allowed")
	assert.True(t, company.IsUserAllowed(allowedID), "allowed user should be allowed")
	assert.False(t, company.IsUserAllowed(notAllowedID), "random user should not be allowed")
}

func TestNewCompany(t *testing.T) {
	ownerID := uuid.New()
	submissionID := uuid.New()

	input := CreateFromSubmissionInput{
		SubmissionID: submissionID,
		CompanyName:  "Test Corp",
		OwnerID:      &ownerID,
	}

	company := NewCompany(input)

	require.NotNil(t, company)
	assert.NotEqual(t, uuid.Nil, company.ID)
	assert.Equal(t, "Test Corp", company.Name)
	assert.Equal(t, &ownerID, company.OwnerID)
	assert.Equal(t, EnrichmentPending, company.EnrichmentStatus)
	assert.True(t, company.AllowedUsers.Contains(ownerID), "owner should be in allowed_users")
	assert.False(t, company.CreatedAt.IsZero())
	assert.False(t, company.UpdatedAt.IsZero())
}

func TestUUIDSlice_Contains(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	slice := UUIDSlice{id1, id2}

	assert.True(t, slice.Contains(id1))
	assert.True(t, slice.Contains(id2))
	assert.False(t, slice.Contains(id3))
}

func TestUUIDSlice_ValueScan(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	original := UUIDSlice{id1, id2}

	// Test Value
	value, err := original.Value()
	require.NoError(t, err)
	assert.Contains(t, value, id1.String())
	assert.Contains(t, value, id2.String())

	// Test Scan
	var scanned UUIDSlice
	err = scanned.Scan(value)
	require.NoError(t, err)
	assert.Len(t, scanned, 2)
	assert.True(t, scanned.Contains(id1))
	assert.True(t, scanned.Contains(id2))
}

func TestUUIDSlice_ScanNil(t *testing.T) {
	var slice UUIDSlice
	err := slice.Scan(nil)
	require.NoError(t, err)
	assert.Empty(t, slice)
}

func TestUUIDSlice_ScanEmpty(t *testing.T) {
	var slice UUIDSlice
	err := slice.Scan("{}")
	require.NoError(t, err)
	assert.Empty(t, slice)
}

func TestStringSlice_ValueScan(t *testing.T) {
	original := StringSlice{"foo", "bar", "baz"}

	// Test Value
	value, err := original.Value()
	require.NoError(t, err)

	// Test Scan
	var scanned StringSlice
	err = scanned.Scan(value)
	require.NoError(t, err)
	assert.Equal(t, original, scanned)
}

func TestStringSlice_ScanNil(t *testing.T) {
	var slice StringSlice
	err := slice.Scan(nil)
	require.NoError(t, err)
	assert.Empty(t, slice)
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrFloat64(f float64) *float64 {
	return &f
}
