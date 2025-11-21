package api

import "time"

// ==================== REQUEST TYPES ====================

type CreateSubmissionRequest struct {
	CompanyName   string   `json:"company_name" binding:"required"`
	IndustryName  string   `json:"industry_name" binding:"required"`
	WebsiteURL    string   `json:"website_url"`
	AnnualRevenue *float64 `json:"annual_revenue"` // Pointer to allow null/0 distinction
	EmployeeCount *int     `json:"employee_count"`
	Location      string   `json:"location"`
	Description   string   `json:"description"`
	Email         string   `json:"email" binding:"required,email"`
	PhoneNumber   string   `json:"phone_number"`
}

// ==================== RESPONSE TYPES ====================

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type SubmissionResponse struct {
	ID            string    `json:"id"`
	CompanyName   string    `json:"company_name"`
	IndustryName  string    `json:"industry_name"`
	Status        string    `json:"status"`
	StatusMessage string    `json:"status_message"`
	CreatedAt     time.Time `json:"created_at"`
}

type SubmissionDetailResponse struct {
	ID          string    `json:"id"`
	CompanyName string    `json:"company_name"`
	Status      string    `json:"status"` // pending, enriching, enriched, analyzing, ready_for_review, completed
	CreatedAt   time.Time `json:"created_at"`

	// Linking IDs for frontend navigation
	EnrichmentID string `json:"enrichment_id,omitempty"`
	AnalysisID   string `json:"analysis_id,omitempty"`
	ReportID     string `json:"report_id,omitempty"` // Only present if PUBLISHED
	PDFURL       string `json:"pdf_url,omitempty"`   // Only present if PUBLISHED
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
