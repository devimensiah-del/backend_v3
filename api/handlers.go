package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/report"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Handler struct {
	submissionSvc   *submission.Service
	enrichmentSvc   *enrichment.Service
	analysisSvc     *analysis.Service
	reportSvc       *report.Service
	asynqClient     *asynq.Client
	db              *sqlx.DB
	redisClient     *redis.Client
	logger          zerolog.Logger
	supabaseURL     string
	supabaseJWTSecret string
}

func NewHandler(
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	analysisSvc *analysis.Service,
	reportSvc *report.Service,
	asynqClient *asynq.Client,
	db *sqlx.DB,
	redisClient *redis.Client,
	logger zerolog.Logger,
	supabaseURL string,
	supabaseJWTSecret string,
) *Handler {
	return &Handler{
		submissionSvc:   submissionSvc,
		enrichmentSvc:   enrichmentSvc,
		analysisSvc:     analysisSvc,
		reportSvc:       reportSvc,
		asynqClient:     asynqClient,
		db:              db,
		redisClient:     redisClient,
		logger:          logger.With().Str("component", "api").Logger(),
		supabaseURL:     supabaseURL,
		supabaseJWTSecret: supabaseJWTSecret,
	}
}

// --- AUTH ENDPOINTS ---

// GetCurrentUser returns the authenticated user's profile
func (h *Handler) GetCurrentUser(c *gin.Context) {
	// Get user ID from AuthMiddleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	// Query user_profiles table
	var profile UserProfile
	query := `
		SELECT id, email, full_name, role, is_active, created_at, updated_at
		FROM user_profiles
		WHERE id = $1
	`

	err := h.db.Get(&profile, query, userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.(string)).Msg("Failed to fetch user profile")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "User profile not found"})
		return
	}

	c.JSON(http.StatusOK, UserProfileResponse{User: profile})
}

// Login handles POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Supabase Auth API endpoint for login
	authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", h.supabaseURL)

	payload := map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Login failed")
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Login failed", Message: "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Signup handles POST /api/v1/auth/signup
func (h *Handler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Supabase Auth API endpoint for signup
	authURL := fmt.Sprintf("%s/auth/v1/signup", h.supabaseURL)

	payload := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
		"data": map[string]string{
			"full_name": req.FullName,
		},
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Signup failed")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Signup failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	// Extract JWT token from Authorization header
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "No token provided"})
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Supabase Auth API endpoint for logout
	authURL := fmt.Sprintf("%s/auth/v1/logout", h.supabaseURL)

	// Call Supabase with the user's token
	req, _ := http.NewRequest("POST", authURL, nil)
	req.Header.Set("apikey", h.supabaseJWTSecret)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Logout request failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Logout failed", Message: err.Error()})
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		h.logger.Error().Int("status", httpResp.StatusCode).Str("body", string(body)).Msg("Logout failed")
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Logged out successfully"})
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Supabase Auth API endpoint for password recovery
	authURL := fmt.Sprintf("%s/auth/v1/recover", h.supabaseURL)

	payload := map[string]string{
		"email": req.Email,
	}

	_, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Password reset request failed")
		// Don't reveal if email exists or not for security
	}

	// Always return success to prevent email enumeration
	c.JSON(http.StatusOK, PasswordResetResponse{
		Message: "If an account exists with this email, you will receive password reset instructions.",
	})
}

// ResetPassword handles POST /api/v1/auth/reset-password
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Supabase Auth API - verify token and update password
	authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=recovery", h.supabaseURL)

	payload := map[string]string{
		"token":    req.Token,
		"password": req.NewPassword,
	}

	resp, err := h.callSupabaseAuth("POST", authURL, payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Password reset failed")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Password reset failed", Message: "Invalid or expired token"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdatePassword handles PUT /api/v1/auth/update-password
func (h *Handler) UpdatePassword(c *gin.Context) {
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Extract JWT token from Authorization header
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "No token provided"})
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Supabase Auth API endpoint for updating user
	authURL := fmt.Sprintf("%s/auth/v1/user", h.supabaseURL)

	payload := map[string]string{
		"password": req.NewPassword,
	}

	payloadBytes, _ := json.Marshal(payload)

	httpReq, _ := http.NewRequest("PUT", authURL, bytes.NewBuffer(payloadBytes))
	httpReq.Header.Set("apikey", h.supabaseJWTSecret)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Update password request failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Update failed", Message: err.Error()})
		return
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode != http.StatusOK {
		h.logger.Error().Int("status", httpResp.StatusCode).Str("body", string(body)).Msg("Update password failed")
		c.JSON(httpResp.StatusCode, ErrorResponse{Error: "Update failed", Message: "Failed to update password"})
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	c.JSON(http.StatusOK, result)
}

// callSupabaseAuth is a helper function to make authenticated requests to Supabase Auth API
func (h *Handler) callSupabaseAuth(method, url string, payload interface{}) (map[string]interface{}, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", h.supabaseJWTSecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		h.logger.Error().Int("status", resp.StatusCode).Str("body", string(body)).Msg("Supabase Auth API error")
		return nil, fmt.Errorf("authentication failed: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// --- PUBLIC ENDPOINTS ---

// CreateSubmission starts the workflow
func (h *Handler) CreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	sub, err := h.submissionSvc.SubmitForm(c.Request.Context(), &submission.SubmitRequest{
		CompanyName:       req.CompanyName,
		CompanyWebsite:    &req.WebsiteURL,
		CompanyIndustry:   &req.IndustryName,
		ContactEmail:      req.Email,
		BusinessChallenge: req.Description,
		// Map other fields...
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create submission")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Submission failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, SubmissionResponse{
		ID:            sub.ID.String(),
		CompanyName:   sub.CompanyName,
		Status:        string(sub.Status),
		StatusMessage: "AI Agent activated. Analysis started.",
		CreatedAt:     sub.CreatedAt,
	})
}

// GetSubmission checks status and returns details
// SECURITY: Ensures user can only access their own submission
func (h *Handler) GetSubmission(c *gin.Context) {
	submissionID := c.Param("id")

	// 1. Fetch Submission
	sub, err := h.submissionSvc.GetByID(c.Request.Context(), parserUUID(submissionID))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// 2. SECURITY CHECK: Ownership
	// We get "userID" and "userRole" from the AuthMiddleware
	requesterID, hasUser := c.Get("userID")
	requesterRole, _ := c.Get("userRole")

	// If the submission belongs to a user (sub.UserID is not nil)
	// AND the requester is not the owner
	// AND the requester is not an admin
	if sub.UserID != nil && hasUser {
		if sub.UserID.String() != requesterID.(string) {
			// Check if Admin bypass
			if requesterRole != "admin" && requesterRole != "super_admin" {
				h.logger.Warn().Str("sub_id", submissionID).Str("requester", requesterID.(string)).Msg("Unauthorized access attempt")
				c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "You do not have permission to view this submission"})
				return
			}
		}
	}

	// 3. Build Response
	resp := SubmissionDetailResponse{
		ID:          sub.ID.String(),
		CompanyName: sub.CompanyName,
		Status:      string(sub.Status),
		CreatedAt:   sub.CreatedAt,
	}

	if enrich, err := h.enrichmentSvc.GetBySubmissionID(c.Request.Context(), parserUUID(submissionID)); err == nil {
		resp.EnrichmentID = enrich.ID.String()
	}
	if anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID); err == nil {
		resp.AnalysisID = anal.ID
	}
	if rep, err := h.reportSvc.GetBySubmissionID(c.Request.Context(), submissionID); err == nil {
		resp.ReportID = rep.ID
		resp.PDFURL = rep.PDFURL
	}

	c.JSON(http.StatusOK, resp)
}

// --- NEW REPORTING ENDPOINTS (Interactive) ---

// PreviewReport generates HTML on-the-fly for the Admin UI
// GET /api/submissions/:id/report/preview
func (h *Handler) PreviewReport(c *gin.Context) {
	submissionID := c.Param("id")

	// Get Analysis ID
	anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready", Message: "Wait for AI to finish."})
		return
	}

	// Generate HTML (In Memory)
	pagesMap, err := h.reportSvc.GeneratePreview(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Preview generation failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Preview failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ReportPreviewResponse{Pages: pagesMap})
}

// PublishReport freezes the report and generates the PDF
// POST /api/submissions/:id/report/publish
func (h *Handler) PublishReport(c *gin.Context) {
	submissionID := c.Param("id")

	anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Analysis not ready"})
		return
	}

	pdfURL, err := h.reportSvc.Publish(c.Request.Context(), submissionID, anal.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Publish failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Publish failed", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ReportPublishResponse{
		ReportID: anal.ID, // Or create a new ID
		PDFURL:   pdfURL,
	})
}

// --- ADMIN ENDPOINTS ---

// ListSubmissions handles GET /api/v1/admin/submissions
func (h *Handler) ListSubmissions(c *gin.Context) {
	// Parse query parameters
	page := 1
	limit := 20
	status := c.Query("status")
	email := c.Query("email")

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build list options
	opts := &submission.ListOptions{
		Limit:   limit,
		Offset:  offset,
		OrderBy: "created_at",
		Order:   "DESC",
	}

	// Add filters if provided
	if status != "" {
		s := submission.Status(status)
		opts.Status = &s
	}
	if email != "" {
		opts.Email = &email
	}

	// Query submissions
	submissions, total, err := h.submissionSvc.ListAll(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list submissions")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve submissions",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SubmissionListResponse{
		Submissions: submissions,
		Page:        page,
		Limit:       limit,
		Total:       total,
	})
}

// GetSubmissionAdmin handles GET /api/v1/admin/submissions/:id
func (h *Handler) GetSubmissionAdmin(c *gin.Context) {
	submissionID := c.Param("id")

	// Get submission (admin can access any submission)
	sub, err := h.submissionSvc.GetByID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// Build detailed response
	resp := SubmissionDetailResponse{
		ID:          sub.ID.String(),
		CompanyName: sub.CompanyName,
		Status:      string(sub.Status),
		CreatedAt:   sub.CreatedAt,
	}

	// Get related entities
	if enrich, err := h.enrichmentSvc.GetBySubmissionID(c.Request.Context(), parserUUID(submissionID)); err == nil {
		resp.EnrichmentID = enrich.ID.String()
	}
	if anal, err := h.analysisSvc.GetBySubmissionID(c.Request.Context(), submissionID); err == nil {
		resp.AnalysisID = anal.ID
	}
	if rep, err := h.reportSvc.GetBySubmissionID(c.Request.Context(), submissionID); err == nil {
		resp.ReportID = rep.ID
		resp.PDFURL = rep.PDFURL
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateSubmissionStatus handles PUT /api/v1/admin/submissions/:id/status
func (h *Handler) UpdateSubmissionStatus(c *gin.Context) {
	submissionID := c.Param("id")

	var req UpdateSubmissionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Parse UUID
	id, err := uuid.Parse(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid submission ID format"})
		return
	}

	// Update status
	newStatus := submission.Status(req.Status)
	if err := h.submissionSvc.UpdateStatus(c.Request.Context(), id, newStatus); err != nil {
		h.logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to update submission status")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Update failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Status updated successfully",
		Data: map[string]interface{}{
			"id":     submissionID,
			"status": req.Status,
		},
	})
}

// RetryEnrichment handles POST /api/v1/admin/submissions/:id/retry-enrichment
func (h *Handler) RetryEnrichment(c *gin.Context) {
	submissionID := c.Param("id")

	// Validate submission exists
	_, err := h.submissionSvc.GetByID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// Create enrichment job payload
	payload := map[string]string{
		"submission_id": submissionID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal enrichment job payload")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: err.Error()})
		return
	}

	// Enqueue enrichment job
	task := asynq.NewTask("enrichment_job", payloadBytes)
	_, err = h.asynqClient.Enqueue(task)
	if err != nil {
		h.logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to enqueue enrichment job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Enqueue failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info().Str("submission_id", submissionID).Msg("Enrichment retry enqueued")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Enrichment retry enqueued successfully",
		Data: map[string]interface{}{
			"id": submissionID,
		},
	})
}

// RetryAnalysis handles POST /api/v1/admin/submissions/:id/retry-analysis
func (h *Handler) RetryAnalysis(c *gin.Context) {
	submissionID := c.Param("id")

	// Validate submission exists
	sub, err := h.submissionSvc.GetByID(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// Get enrichment record
	enrichmentRecord, err := h.enrichmentSvc.GetBySubmissionID(c.Request.Context(), parserUUID(submissionID))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Enrichment required",
			Message: "Cannot retry analysis without enrichment data. Run enrichment first.",
		})
		return
	}

	// Update status to analyzing
	h.submissionSvc.UpdateStatus(c.Request.Context(), sub.ID, "analyzing")

	// Create analysis job payload
	payload := map[string]string{
		"submission_id": submissionID,
		"enrichment_id": enrichmentRecord.ID.String(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal analysis job payload")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: err.Error()})
		return
	}

	// Enqueue analysis job
	task := asynq.NewTask("analysis_job", payloadBytes)
	_, err = h.asynqClient.Enqueue(task)
	if err != nil {
		h.logger.Error().Err(err).Str("submission_id", submissionID).Msg("Failed to enqueue analysis job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Enqueue failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info().Str("submission_id", submissionID).Msg("Analysis retry enqueued")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Analysis retry enqueued successfully",
		Data: map[string]interface{}{
			"id": submissionID,
		},
	})
}

// GetAnalytics handles GET /api/v1/admin/analytics
func (h *Handler) GetAnalytics(c *gin.Context) {
	// Get analytics data from service
	analytics, err := h.submissionSvc.GetAnalytics(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get analytics")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve analytics",
			Message: err.Error(),
		})
		return
	}

	// Map to response type
	response := AnalyticsResponse{
		TotalSubmissions:     analytics.TotalSubmissions,
		ActiveSubmissions:    analytics.ActiveSubmissions,
		CompletedSubmissions: analytics.CompletedSubmissions,
		Revenue:              analytics.Revenue,
	}

	c.JSON(http.StatusOK, response)
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	health := gin.H{
		"status": "healthy",
		"services": gin.H{
			"database": "healthy",
			"redis":    "healthy",
		},
	}

	// Check database
	if err := h.db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["services"].(gin.H)["database"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}

	// Check Redis if available
	if h.redisClient != nil {
		if err := h.redisClient.Ping(c.Request.Context()).Err(); err != nil {
			health["status"] = "unhealthy"
			health["services"].(gin.H)["redis"] = "unhealthy"
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
	}

	c.JSON(http.StatusOK, health)
}

// Helper
func parserUUID(id string) uuid.UUID {
	u, _ := uuid.Parse(id)
	return u
}
