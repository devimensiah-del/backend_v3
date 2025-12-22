package api

import "time"

// ==================== REQUEST TYPES ====================

// ==================== AUTH REQUEST TYPES ====================

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"fullName"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

// SetPasswordRequest - for users who were auto-created without password
type SetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// ==================== SUBMISSION REQUEST TYPES ====================

// CreateSubmissionRequest - simplified submission form
// Required: companyName, contactName, contactEmail, website (unless hasNoWebsite)
// Optional: contactPhone, cnpj
type CreateSubmissionRequest struct {
	// Required fields
	CompanyName  string `json:"companyName" binding:"required"`
	ContactName  string `json:"contactName" binding:"required"`
	ContactEmail string `json:"contactEmail" binding:"required,email"`

	// Optional contact phone (removed from required for simpler landing page)
	ContactPhone *string `json:"contactPhone"`

	// Website: required unless hasNoWebsite is true
	Website      *string `json:"website"`
	HasNoWebsite bool    `json:"hasNoWebsite"`

	// Optional
	CNPJ *string `json:"cnpj"`
}

// ==================== ADMIN REQUEST TYPES ====================

// ==================== COMPANY REQUEST TYPES ====================

// CreateCompanyRequest represents the request body for creating a company directly
type CreateCompanyRequest struct {
	Name         string  `json:"name" binding:"required"`
	Website      *string `json:"website"`
	CNPJ         *string `json:"cnpj"`
	Industry     *string `json:"industry"`
	CompanySize  *string `json:"company_size"`
	Location     *string `json:"location"`
	TargetMarket *string `json:"target_market"`
	FundingStage *string `json:"funding_stage"`
}

// ==================== RESPONSE TYPES ====================

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ==================== AUTH RESPONSE TYPES ====================

type AuthResponse struct {
	User        *AuthUser `json:"user"`
	AccessToken string    `json:"access_token,omitempty"`
	TokenType   string    `json:"token_type,omitempty"`
	ExpiresIn   int       `json:"expires_in,omitempty"`
	ExpiresAt   int64     `json:"expires_at,omitempty"`
}

type AuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

type PasswordResetResponse struct {
	Message string `json:"message"`
}

// SubmissionResponse is the basic response for submission creation
// Returns IDs of created entities (Submission, Company)
// Note: Challenge is no longer created automatically from submission
type SubmissionResponse struct {
	ID        string     `json:"id"`
	CompanyID string     `json:"companyId"` // ID of created/linked company
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// TemporaryAuth contains auth data for auto-created users (1-hour session)
type TemporaryAuth struct {
	AccessToken string   `json:"access_token"`
	ExpiresIn   int      `json:"expires_in"` // 3600 seconds (1 hour)
	User        AuthUser `json:"user"`
}

// CreateSubmissionWithAuthResponse extends SubmissionResponse with optional auth
// Auth is provided when a new user is auto-created during submission
type CreateSubmissionWithAuthResponse struct {
	Submission SubmissionResponse `json:"submission"`
	Auth       *TemporaryAuth     `json:"auth,omitempty"` // nil if user was already authenticated
}

// SubmissionDetailResponse contains the public/admin submission payload used by the UI
type SubmissionDetailResponse struct {
	ID               string   `json:"id"`
	UserID           *string  `json:"userId,omitempty"`
	CompanyName      string   `json:"companyName"`
	CNPJ             *string  `json:"cnpj,omitempty"`
	CompanyWebsite   *string  `json:"companyWebsite,omitempty"`
	CompanyIndustry  *string  `json:"companyIndustry,omitempty"`
	CompanySize      *string  `json:"companySize,omitempty"`
	CompanyLocation  *string  `json:"companyLocation,omitempty"`
	ContactName      string   `json:"contactName"`
	ContactEmail     string   `json:"contactEmail"`
	ContactPhone     *string  `json:"contactPhone,omitempty"`
	ContactPosition  *string  `json:"contactPosition,omitempty"`
	TargetMarket     *string  `json:"targetMarket,omitempty"`
	AnnualRevenueMin *float64 `json:"annualRevenueMin,omitempty"`
	AnnualRevenueMax *float64 `json:"annualRevenueMax,omitempty"`
	FundingStage     *string  `json:"fundingStage,omitempty"`
	AdditionalNotes  *string  `json:"additionalNotes,omitempty"`
	LinkedInURL      *string  `json:"linkedinUrl,omitempty"`
	TwitterHandle    *string  `json:"twitterHandle,omitempty"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`

	// Related entities
	CompanyID   *string `json:"companyId,omitempty"`   // Linked company
	ChallengeID *string `json:"challengeId,omitempty"` // Primary challenge
	AnalysisID  *string `json:"analysisId,omitempty"`  // Latest analysis

	// Challenge data (fetched from related challenge entity)
	ChallengeCategory *string `json:"challengeCategory,omitempty"` // growth, transform, etc.
	ChallengeType     *string `json:"challengeType,omitempty"`     // growth_organic, etc.
	BusinessChallenge *string `json:"businessChallenge,omitempty"` // Free-text challenge description

	// Derived status (from Company enrichment_status + Analysis status)
	Status string `json:"status"` // pending, enriching, analyzing, completed, failed
}

// ... (Keep HealthResponse, MessageResponse, SubmissionListResponse as they were) ...
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

type MessageResponse struct {
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

type SubmissionListResponse struct {
	Data       interface{} `json:"data"` // Changed from "submissions" to match frontend
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"` // Changed from "limit" to match frontend
	Total      int         `json:"total"`
	TotalPages int         `json:"totalPages"` // Added to match frontend
}

// AnalyticsResponse returns admin analytics/metrics
type AnalyticsResponse struct {
	TotalSubmissions     int `json:"totalSubmissions"`
	ActiveSubmissions    int `json:"activeSubmissions"`
	CompletedSubmissions int `json:"completedSubmissions"`
}

// UserProfileResponse returns user profile data
type UserProfileResponse struct {
	User UserProfile `json:"user"`
}

// UserProfile represents a user's profile from user_profiles table
type UserProfile struct {
	ID          string    `json:"id" db:"id"`
	Email       string    `json:"email" db:"email"`
	FullName    *string   `json:"fullName" db:"full_name"`
	Role        string    `json:"role" db:"role"`
	IsActive    bool      `json:"isActive" db:"is_active"`
	PasswordSet bool      `json:"passwordSet" db:"password_set"` // false for auto-created users who need to set password
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// ==================== ANALYSIS RESPONSE TYPES ====================

// AnalysisResponse maps domain Analysis to frontend-expected structure
type AnalysisResponse struct {
	ID              string                 `json:"id"`
	CompanyID       *string                `json:"companyId,omitempty"` // Direct link to company
	ChallengeID     string                 `json:"challengeId"`         // REQUIRED: Link to challenge
	Status          string                 `json:"status"`
	Analysis        map[string]interface{} `json:"analysis"` // Contains all framework data
	IsVisibleToUser bool                   `json:"is_visible_to_user"`
	IsPublic        bool                   `json:"is_public"` // Controls whether access code works without login
	AccessCode      *string                `json:"access_code,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}
