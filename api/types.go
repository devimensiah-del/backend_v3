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

// ==================== SUBMISSION REQUEST TYPES ====================

// CreateSubmissionRequest matches the frontend SubmissionFormData structure
// Frontend sends: companyName, cnpj, industry, companySize, website, strategicGoal, currentChallenges, competitivePosition, additionalInfo (JSON string)
type CreateSubmissionRequest struct {
	// Required fields from frontend
	CompanyName         string `json:"companyName" binding:"required"`
	CNPJ                string `json:"cnpj" binding:"required"`
	Industry            string `json:"industry" binding:"required"`
	CompanySize         string `json:"companySize" binding:"required"`
	StrategicGoal       string `json:"strategicGoal" binding:"required"`
	CurrentChallenges   string `json:"currentChallenges" binding:"required"`
	CompetitivePosition string `json:"competitivePosition" binding:"required"`

	// Optional fields
	Website        *string `json:"website,omitempty"`
	AdditionalInfo *string `json:"additionalInfo,omitempty"` // JSON string containing contact and other fields
}

// AdditionalInfoData represents the parsed additionalInfo JSON string from frontend
type AdditionalInfoData struct {
	ContactName      string   `json:"contactName"`
	ContactEmail     string   `json:"contactEmail"`
	ContactPhone     string   `json:"contactPhone"`
	ContactPosition  string   `json:"contactPosition"`
	CompanyLocation  string   `json:"companyLocation"`
	TargetMarket     string   `json:"targetMarket"`
	AnnualRevenueMin *float64 `json:"annualRevenueMin"`
	AnnualRevenueMax *float64 `json:"annualRevenueMax"`
	FundingStage     string   `json:"fundingStage"`
	AdditionalNotes  string   `json:"additionalNotes"`
	LinkedInURL      string   `json:"linkedinUrl"`
	TwitterHandle    string   `json:"twitterHandle"`
}

// ==================== ADMIN REQUEST TYPES ====================

type UpdateSubmissionStatusRequest struct {
	Status        string `json:"status" binding:"required"`
	StatusMessage string `json:"statusMessage"`
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
type SubmissionResponse struct {
	ID            string    `json:"id"`
	CompanyName   string    `json:"companyName"`
	CNPJ          string    `json:"cnpj,omitempty"`
	Industry      string    `json:"industry,omitempty"`
	Status        string    `json:"status"`
	StatusMessage string    `json:"status_message,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// SubmissionDetailResponse contains full submission data matching frontend Submission interface
type SubmissionDetailResponse struct {
	// IDs
	ID     string  `json:"id"`
	UserID *string `json:"userId,omitempty"`

	// Company Information
	CompanyName string  `json:"companyName"`
	CNPJ        *string `json:"cnpj,omitempty"`
	Website     *string `json:"website,omitempty"`
	Email       *string `json:"email,omitempty"`
	Industry    *string `json:"industry,omitempty"`
	CompanySize *string `json:"companySize,omitempty"`

	// Strategic Context
	StrategicGoal       *string `json:"strategicGoal,omitempty"`
	CurrentChallenges   *string `json:"currentChallenges,omitempty"`
	CompetitivePosition *string `json:"competitivePosition,omitempty"`
	AdditionalInfo      *string `json:"additionalInfo,omitempty"`

	// Legacy fields (for backward compatibility)
	ContactName      *string  `json:"contactName,omitempty"`
	ContactEmail     *string  `json:"contactEmail,omitempty"`
	ContactPhone     *string  `json:"contactPhone,omitempty"`
	ContactPosition  *string  `json:"contactPosition,omitempty"`
	TargetMarket     *string  `json:"targetMarket,omitempty"`
	AnnualRevenueMin *float64 `json:"annualRevenueMin,omitempty"`
	AnnualRevenueMax *float64 `json:"annualRevenueMax,omitempty"`
	FundingStage     *string  `json:"fundingStage,omitempty"`
	BusinessChallenge *string `json:"businessChallenge,omitempty"`
	AdditionalNotes  *string  `json:"additionalNotes,omitempty"`
	LinkedInURL      *string  `json:"linkedinUrl,omitempty"`
	TwitterHandle    *string  `json:"twitterHandle,omitempty"`
	Location         *string  `json:"location,omitempty"`

	// Metadata
	Status        string  `json:"status"`
	PaymentStatus *string `json:"paymentStatus,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`

	// Relationships
	EnrichmentID *string `json:"enrichmentId,omitempty"`
	AnalysisID   *string `json:"analysisId,omitempty"`
	ReportID     *string `json:"reportId,omitempty"`

	// PDF URL (Only present if status === 'completed')
	PDFURL *string `json:"pdfUrl,omitempty"`
}

// ReportPreviewResponse returns the HTML for the Admin UI
type ReportPreviewResponse struct {
	Pages map[string]string `json:"pages"` // Key: "SWOT", Value: "<html>...</html>"
}

// ReportPublishResponse returns the final PDF link
type ReportPublishResponse struct {
	ReportID string `json:"report_id"`
	PDFURL   string `json:"pdf_url"`
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
	Submissions interface{} `json:"submissions"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
	Total       int         `json:"total"`
}

// AnalyticsResponse returns admin analytics/metrics
type AnalyticsResponse struct {
	TotalSubmissions     int     `json:"totalSubmissions"`
	ActiveSubmissions    int     `json:"activeSubmissions"`
	CompletedSubmissions int     `json:"completedSubmissions"`
	Revenue              float64 `json:"revenue"`
}

// UserProfileResponse returns user profile data
type UserProfileResponse struct {
	User UserProfile `json:"user"`
}

// UserProfile represents a user's profile from user_profiles table
type UserProfile struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	FullName  *string   `json:"fullName" db:"full_name"`
	Role      string    `json:"role" db:"role"`
	IsActive  bool      `json:"isActive" db:"is_active"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}
