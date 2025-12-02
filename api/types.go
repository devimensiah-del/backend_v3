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
// Only companyName is required at the API level - contact fields validated separately from additionalInfo
type CreateSubmissionRequest struct {
	// Required field
	CompanyName string `json:"companyName" binding:"required"`

	// Optional fields that may be empty
	CNPJ                string  `json:"cnpj"`
	Industry            string  `json:"industry"`
	CompanySize         string  `json:"companySize"`
	StrategicGoal       string  `json:"strategicGoal"`
	CurrentChallenges   string  `json:"currentChallenges"`
	CompetitivePosition string  `json:"competitivePosition"`
	Website             *string `json:"website,omitempty"`
	AdditionalInfo      *string `json:"additionalInfo,omitempty"` // JSON string containing contact and other fields
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
	ID          string    `json:"id"`
	CompanyName string    `json:"companyName"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// SubmissionDetailResponse contains the public/admin submission payload used by the UI
type SubmissionDetailResponse struct {
	ID                string   `json:"id"`
	UserID            *string  `json:"userId,omitempty"`
	CompanyName       string   `json:"companyName"`
	CNPJ              *string  `json:"cnpj,omitempty"`
	CompanyWebsite    *string  `json:"companyWebsite,omitempty"`
	CompanyIndustry   *string  `json:"companyIndustry,omitempty"`
	CompanySize       *string  `json:"companySize,omitempty"`
	CompanyLocation   *string  `json:"companyLocation,omitempty"`
	ContactName       string   `json:"contactName"`
	ContactEmail      string   `json:"contactEmail"`
	ContactPhone      *string  `json:"contactPhone,omitempty"`
	ContactPosition   *string  `json:"contactPosition,omitempty"`
	TargetMarket      *string  `json:"targetMarket,omitempty"`
	AnnualRevenueMin  *float64 `json:"annualRevenueMin,omitempty"`
	AnnualRevenueMax  *float64 `json:"annualRevenueMax,omitempty"`
	FundingStage      *string  `json:"fundingStage,omitempty"`
	BusinessChallenge string   `json:"businessChallenge"`
	AdditionalNotes   *string  `json:"additionalNotes,omitempty"`
	LinkedInURL       *string  `json:"linkedinUrl,omitempty"`
	TwitterHandle     *string  `json:"twitterHandle,omitempty"`
	Status            string   `json:"status"`
	CreatedAt         string   `json:"createdAt"`
	UpdatedAt         string   `json:"updatedAt"`
	EnrichmentID      *string  `json:"enrichmentId,omitempty"`
	AnalysisID        *string  `json:"analysisId,omitempty"`
	ReportID          *string  `json:"reportId,omitempty"`
	PDFURL            *string  `json:"pdfUrl,omitempty"`
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
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	FullName  *string   `json:"fullName" db:"full_name"`
	Role      string    `json:"role" db:"role"`
	IsActive  bool      `json:"isActive" db:"is_active"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// ==================== ENRICHMENT RESPONSE TYPES ====================

// EnrichmentResponse maps domain Enrichment to frontend-expected structure
// CRITICAL FIX: Backend domain model uses "enrichedData" but frontend expects "data"
type EnrichmentResponse struct {
	ID           string                 `json:"id"`
	SubmissionID string                 `json:"submissionId"`
	Status       string                 `json:"status"`
	Progress     int                    `json:"progress"`
	CurrentStep  string                 `json:"currentStep"`
	Data         map[string]interface{} `json:"data"` // Renamed from EnrichedData for frontend compatibility
	IsLocked     bool                   `json:"isLocked"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// ==================== ANALYSIS RESPONSE TYPES ====================

// AnalysisResponse maps domain Analysis to frontend-expected structure
type AnalysisResponse struct {
	ID              string                 `json:"id"`
	SubmissionID    string                 `json:"submissionId"`
	Status          string                 `json:"status"`
	Analysis        map[string]interface{} `json:"analysis"` // Contains all framework data
	IsVisibleToUser bool                   `json:"is_visible_to_user"`
	IsBlurred       bool                   `json:"is_blurred"` // Controls premium framework blur overlay
	IsPublic        bool                   `json:"is_public"`  // Controls whether access code works without login
	AccessCode      *string                `json:"access_code,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// ==================== REPORT RESPONSE TYPES ====================

// ReportPreviewResponse returns HTML preview pages for admin review
type ReportPreviewResponse struct {
	Pages map[string]string `json:"pages"` // Map of page name to HTML content
}

// ReportPublishResponse returns status of PDF generation request
type ReportPublishResponse struct {
	ReportID string `json:"reportId"`
	TaskID   string `json:"taskId,omitempty"` // Task ID for async tracking
	Status   string `json:"status"`           // "processing", "completed", "failed"
	Message  string `json:"message,omitempty"`
	PDFURL   string `json:"pdfUrl,omitempty"` // Set when completed synchronously
}
