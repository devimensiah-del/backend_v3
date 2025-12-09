package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend_v3/domain/analysis"
	"backend_v3/domain/challenge"
	"backend_v3/domain/company"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// CompanyHandlers handles company-based re-enrich/re-analyze endpoints
type CompanyHandlers struct {
	CompanyService      *company.Service
	SubmissionService   *submission.Service // Needed for re-analyze flow
	EnrichmentService   *enrichment.Service
	ChallengeService    *challenge.Service // For creating challenges for analysis
	AnalysisService     *analysis.Service  // For fetching latest analysis per challenge
	AsynqClient         *asynq.Client
	Logger              zerolog.Logger
	EnrichmentLimiter   *EnrichmentRateLimiter // Rate limiter for re-enrichment
}

// NewCompanyHandlers creates a new company handler set
func NewCompanyHandlers(
	companySvc *company.Service,
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	challengeSvc *challenge.Service,
	analysisSvc *analysis.Service,
	asynqClient *asynq.Client,
	logger zerolog.Logger,
) *CompanyHandlers {
	return &CompanyHandlers{
		CompanyService:    companySvc,
		SubmissionService: submissionSvc,
		EnrichmentService: enrichmentSvc,
		ChallengeService:  challengeSvc,
		AnalysisService:   analysisSvc,
		AsynqClient:       asynqClient,
		Logger:            logger,
		EnrichmentLimiter: NewEnrichmentRateLimiter(),
	}
}

// checkCompanyAccess verifies user has access to this company (admin or in allowed_users)
func (h *CompanyHandlers) checkCompanyAccess(c *gin.Context, comp *company.Company) bool {
	// Admin roles always allowed
	role, _ := c.Get("userRole")
	if role == "admin" || role == "super_admin" || role == "service_role" {
		return true
	}

	// Check allowed_users array
	userIDStr, exists := c.Get("userID")
	if !exists {
		return false
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return false
	}

	return comp.IsUserAllowed(userID)
}

// ========================================================================
// USER-FACING HANDLERS
// ========================================================================

// CreateCompany handles POST /api/v1/companies
// Creates a company directly (triggers enrichment automatically)
func (h *CompanyHandlers) CreateCompany(c *gin.Context) {
	var req CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal error", Message: "Invalid user ID"})
		return
	}

	// Build input for direct creation (no submission link)
	input := company.CreateFromSubmissionInput{
		CompanyName: req.Name,
		OwnerID:     &userID,
	}

	// Set optional fields (req fields are already pointers)
	input.CNPJ = req.CNPJ
	input.Website = req.Website
	input.Industry = req.Industry
	input.CompanySize = req.CompanySize
	input.Location = req.Location
	input.TargetMarket = req.TargetMarket
	input.FundingStage = req.FundingStage

	// Create company directly (no submission link, no FK constraint issue)
	comp, err := h.CompanyService.CreateDirect(c.Request.Context(), input)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to create company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Creation failed", Message: err.Error()})
		return
	}

	h.Logger.Info().Str("company_id", comp.ID.String()).Str("owner_id", userID.String()).Msg("Company created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"company": comp,
		"message": "Company created successfully. Enrichment will be triggered automatically.",
	})
}

// GetCompany handles GET /api/v1/companies/:id
// Returns company details if user has access
func (h *CompanyHandlers) GetCompany(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve company"})
		return
	}
	if comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access
	if !h.checkCompanyAccess(c, comp) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied to this company"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"company": comp,
	})
}

// GetCompanyAdmin handles GET /api/v1/admin/companies/:id
// Returns company details with full admin view
func (h *CompanyHandlers) GetCompanyAdmin(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve company"})
		return
	}
	if comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Get additional data for admin view
	analysesHistory, _ := h.CompanyService.GetAnalysesHistory(c.Request.Context(), companyUUID)

	c.JSON(http.StatusOK, gin.H{
		"company":          comp,
		"analyses_history": analysesHistory,
	})
}

// ChallengeWithAnalysis extends Challenge with latest analysis info
type ChallengeWithAnalysis struct {
	*challenge.Challenge
	LatestAnalysis *analysis.Analysis `json:"latest_analysis,omitempty"`
	AnalysisCount  int                `json:"analysis_count"`
}

// GetCompanyChallenges handles GET /api/v1/companies/:id/challenges
// Returns all challenges for a company with their latest analysis status
func (h *CompanyHandlers) GetCompanyChallenges(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Get company to verify access
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access
	if !h.checkCompanyAccess(c, comp) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied to this company"})
		return
	}

	// Get challenges for this company
	challenges, err := h.ChallengeService.ListByCompany(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to list challenges")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve challenges"})
		return
	}

	// Enrich each challenge with latest analysis
	enrichedChallenges := make([]ChallengeWithAnalysis, 0, len(challenges))
	for _, ch := range challenges {
		enriched := ChallengeWithAnalysis{
			Challenge:     ch,
			AnalysisCount: 0,
		}

		// Try to get the latest analysis for this challenge
		if h.AnalysisService != nil {
			latestAnalysis, err := h.AnalysisService.GetByChallengeID(c.Request.Context(), ch.ID)
			if err == nil && latestAnalysis != nil {
				enriched.LatestAnalysis = latestAnalysis
				enriched.AnalysisCount = 1 // At least one exists
			}
		}

		enrichedChallenges = append(enrichedChallenges, enriched)
	}

	c.JSON(http.StatusOK, gin.H{
		"challenges": enrichedChallenges,
		"count":      len(enrichedChallenges),
	})
}

// CreateChallenge handles POST /api/v1/challenges
// Creates a new challenge for an existing company
func (h *CompanyHandlers) CreateChallenge(c *gin.Context) {
	var req struct {
		CompanyID         string `json:"company_id" binding:"required"`
		ChallengeCategory string `json:"challenge_category" binding:"required"`
		ChallengeType     string `json:"challenge_type" binding:"required"`
		BusinessChallenge string `json:"business_challenge" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Validate challenge category and type
	if !challenge.IsValidCategory(req.ChallengeCategory) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid category",
			Message: "challenge_category must be one of: growth, transform, transition, compete, funding",
		})
		return
	}
	if !challenge.IsValidType(req.ChallengeCategory, req.ChallengeType) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid type",
			Message: "challenge_type is not valid for the given category",
		})
		return
	}

	// Parse company ID
	companyUUID, err := uuid.Parse(req.CompanyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Get company to verify access
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access
	if !h.checkCompanyAccess(c, comp) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied to this company"})
		return
	}

	// Create the challenge
	newChallenge := challenge.NewChallenge(
		companyUUID,
		challenge.ChallengeCategory(req.ChallengeCategory),
		challenge.ChallengeType(req.ChallengeType),
		req.BusinessChallenge,
	)

	createdChallenge, err := h.ChallengeService.Create(c.Request.Context(), newChallenge)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", req.CompanyID).Msg("Failed to create challenge")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Creation failed", Message: err.Error()})
		return
	}

	h.Logger.Info().
		Str("challenge_id", createdChallenge.ID.String()).
		Str("company_id", req.CompanyID).
		Str("category", req.ChallengeCategory).
		Str("type", req.ChallengeType).
		Msg("Challenge created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"challenge": createdChallenge,
	})
}

// GetMyCompanies handles GET /api/v1/companies
// Returns all companies where the user is owner or in allowed_users
func (h *CompanyHandlers) GetMyCompanies(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user", Message: "Invalid user ID"})
		return
	}

	companies, err := h.CompanyService.GetUserCompanies(c.Request.Context(), userID)
	if err != nil {
		h.Logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get user companies")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve companies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"companies": companies,
		"count":     len(companies),
	})
}

// ListAllCompanies handles GET /api/v1/admin/companies
// Returns all companies with pagination (admin only)
func (h *CompanyHandlers) ListAllCompanies(c *gin.Context) {
	// Parse pagination params
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseIntParam(l, 1, 100); err == nil {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseIntParam(o, 0, 1000000); err == nil {
			offset = parsed
		}
	}

	companies, total, err := h.CompanyService.ListAll(c.Request.Context(), limit, offset)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to list companies")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve companies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"companies": companies,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// NOTE: UpdateCompany and UpdateCompanyUser removed - edit company in Supabase directly

// ========================================================================
// ADMIN HANDLERS - RE-ANALYSIS WORKFLOW
// ========================================================================

// ReAnalyzeRequest contains the challenge info required for a new analysis
type ReAnalyzeRequest struct {
	// Required fields for challenge creation
	ChallengeCategory string `json:"challenge_category"` // growth, transform, transition, compete, funding
	ChallengeType     string `json:"challenge_type"`     // Specific type within category
	BusinessChallenge string `json:"business_challenge"` // Free-text context
}

// ReAnalyzeCompany handles POST /api/v1/admin/companies/:id/re-analyze
// Creates a new challenge and triggers analysis for an existing company
// Requires challenge_category, challenge_type, and business_challenge in request body
func (h *CompanyHandlers) ReAnalyzeCompany(c *gin.Context) {
	companyID := c.Param("id")

	// Parse request body for challenge info
	var req ReAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Request body must contain challenge_category, challenge_type, and business_challenge",
		})
		return
	}

	// Validate required fields
	if req.ChallengeCategory == "" || req.ChallengeType == "" || req.BusinessChallenge == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing fields",
			Message: "challenge_category, challenge_type, and business_challenge are all required",
		})
		return
	}

	// Validate challenge category and type
	if !challenge.IsValidCategory(req.ChallengeCategory) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid category",
			Message: "challenge_category must be one of: growth, transform, transition, compete, funding",
		})
		return
	}
	if !challenge.IsValidType(req.ChallengeCategory, req.ChallengeType) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid type",
			Message: "challenge_type is not valid for the given category",
		})
		return
	}

	// Parse company UUID
	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Get company
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access
	if !h.checkCompanyAccess(c, comp) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied - not admin or authorized user"})
		return
	}

	// Check company enrichment status
	if comp.EnrichmentStatus != "completed" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Enrichment not completed",
			Message: "Company enrichment must be completed before running analysis",
		})
		return
	}

	// Get user info from auth context
	userName, _ := c.Get("userName")
	userEmail, _ := c.Get("userEmail")
	userIDStr, _ := c.Get("userID")

	contactName := "Admin User"
	if userName != nil {
		contactName = userName.(string)
	}
	contactEmail := "admin@imensiah.com"
	if userEmail != nil {
		contactEmail = userEmail.(string)
	}
	var userID *uuid.UUID
	if userIDStr != nil {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			userID = &uid
		}
	}

	// Create new challenge for this company
	newChallenge := challenge.NewChallenge(
		companyUUID,
		challenge.ChallengeCategory(req.ChallengeCategory),
		challenge.ChallengeType(req.ChallengeType),
		req.BusinessChallenge,
	)
	createdChallenge, err := h.ChallengeService.Create(c.Request.Context(), newChallenge)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to create challenge")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Challenge creation failed", Message: err.Error()})
		return
	}

	// Create submission from company data (for tracking/history)
	sub, err := h.SubmissionService.CreateFromCompany(c.Request.Context(), submission.CreateFromCompanyInput{
		CompanyID:        companyUUID,
		CompanyName:      comp.Name,
		CNPJ:             comp.CNPJ,
		Website:          comp.Website,
		Industry:         comp.Industry,
		CompanySize:      comp.CompanySize,
		Location:         comp.Location,
		TargetMarket:     comp.TargetMarket,
		FundingStage:     comp.FundingStage,
		AnnualRevenueMin: comp.AnnualRevenueMin,
		AnnualRevenueMax: comp.AnnualRevenueMax,
		LinkedInURL:      comp.LinkedInURL,
		TwitterHandle:    comp.TwitterHandle,
		ContactName:      contactName,
		ContactEmail:     contactEmail,
		UserID:           userID,
	})
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to create submission from company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Creation failed", Message: err.Error()})
		return
	}

	// Link submission to company (not primary)
	if err := h.CompanyService.LinkSubmission(c.Request.Context(), companyUUID, sub.ID, false, userID); err != nil {
		h.Logger.Warn().Err(err).Str("company_id", companyID).Msg("Failed to link submission to company")
	}

	// Enqueue analysis job with challenge_id
	payload := map[string]string{
		"submission_id": sub.ID.String(),
		"company_id":    companyID,
		"challenge_id":  createdChallenge.ID.String(),
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask("analysis_job", payloadBytes)

	// Enqueue with explicit options for better visibility
	taskInfo, err := h.AsynqClient.Enqueue(task,
		asynq.MaxRetry(3),
		asynq.Queue("default"),
		asynq.Retention(24*time.Hour),
	)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to enqueue analysis job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: "Failed to start analysis"})
		return
	}

	h.Logger.Info().
		Str("task_id", taskInfo.ID).
		Str("queue", taskInfo.Queue).
		Str("submission_id", sub.ID.String()).
		Str("challenge_id", createdChallenge.ID.String()).
		Msg("Analysis job enqueued successfully")

	h.Logger.Info().
		Str("company_id", companyID).
		Str("submission_id", sub.ID.String()).
		Str("challenge_id", createdChallenge.ID.String()).
		Str("challenge_category", req.ChallengeCategory).
		Str("challenge_type", req.ChallengeType).
		Msg("Re-analysis started with new challenge")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Re-analysis started",
		Data: map[string]interface{}{
			"submission_id":      sub.ID.String(),
			"company_id":         companyID,
			"challenge_id":       createdChallenge.ID.String(),
			"challenge_category": req.ChallengeCategory,
			"challenge_type":     req.ChallengeType,
		},
	})
}

// AnalyzeChallenge handles POST /api/v1/admin/challenges/:id/analyze
// Triggers analysis for an existing challenge (no new challenge creation)
func (h *CompanyHandlers) AnalyzeChallenge(c *gin.Context) {
	challengeID := c.Param("id")

	// Parse challenge UUID
	challengeUUID, err := uuid.Parse(challengeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid challenge ID format"})
		return
	}

	// Get the challenge
	existingChallenge, err := h.ChallengeService.GetByID(c.Request.Context(), challengeUUID)
	if err != nil || existingChallenge == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Challenge not found"})
		return
	}

	// Get the company
	comp, err := h.CompanyService.GetByID(c.Request.Context(), existingChallenge.CompanyID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access (admin only for now)
	if !h.checkCompanyAccess(c, comp) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied - not admin or authorized user"})
		return
	}

	// Check company enrichment status
	if comp.EnrichmentStatus != "completed" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Enrichment not completed",
			Message: "Company enrichment must be completed before running analysis",
		})
		return
	}

	// Get user info from auth context
	userName, _ := c.Get("userName")
	userEmail, _ := c.Get("userEmail")
	userIDStr, _ := c.Get("userID")

	contactName := "Admin User"
	if userName != nil {
		contactName = userName.(string)
	}
	contactEmail := "admin@imensiah.com"
	if userEmail != nil {
		contactEmail = userEmail.(string)
	}
	var userID *uuid.UUID
	if userIDStr != nil {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			userID = &uid
		}
	}

	// Create submission from company data (for tracking/history)
	sub, err := h.SubmissionService.CreateFromCompany(c.Request.Context(), submission.CreateFromCompanyInput{
		CompanyID:        comp.ID,
		CompanyName:      comp.Name,
		CNPJ:             comp.CNPJ,
		Website:          comp.Website,
		Industry:         comp.Industry,
		CompanySize:      comp.CompanySize,
		Location:         comp.Location,
		TargetMarket:     comp.TargetMarket,
		FundingStage:     comp.FundingStage,
		AnnualRevenueMin: comp.AnnualRevenueMin,
		AnnualRevenueMax: comp.AnnualRevenueMax,
		LinkedInURL:      comp.LinkedInURL,
		TwitterHandle:    comp.TwitterHandle,
		ContactName:      contactName,
		ContactEmail:     contactEmail,
		UserID:           userID,
	})
	if err != nil {
		h.Logger.Error().Err(err).Str("challenge_id", challengeID).Msg("Failed to create submission for challenge analysis")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Creation failed", Message: err.Error()})
		return
	}

	// Link submission to company (not primary)
	if err := h.CompanyService.LinkSubmission(c.Request.Context(), comp.ID, sub.ID, false, userID); err != nil {
		h.Logger.Warn().Err(err).Str("company_id", comp.ID.String()).Msg("Failed to link submission to company")
	}

	// Enqueue analysis job with existing challenge_id
	payload := map[string]string{
		"submission_id": sub.ID.String(),
		"company_id":    comp.ID.String(),
		"challenge_id":  challengeID,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask("analysis_job", payloadBytes)

	// Generate unique task ID to prevent deduplication with previous runs
	taskID := uuid.New().String()

	h.Logger.Info().
		Str("task_id", taskID).
		Str("task_type", "analysis_job").
		Str("payload", string(payloadBytes)).
		Msg("🔔 ABOUT TO ENQUEUE analysis job - check if worker picks it up")

	// Enqueue with explicit options including unique TaskID
	taskInfo, err := h.AsynqClient.Enqueue(task,
		asynq.TaskID(taskID),
		asynq.MaxRetry(3),
		asynq.Queue("default"),
		asynq.Retention(24*time.Hour),
	)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to enqueue analysis job for challenge")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: "Failed to start analysis"})
		return
	}

	h.Logger.Info().
		Str("task_id", taskInfo.ID).
		Str("queue", taskInfo.Queue).
		Str("submission_id", sub.ID.String()).
		Str("challenge_id", challengeID).
		Str("company_id", comp.ID.String()).
		Msg("Analysis job enqueued for existing challenge")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Analysis started for challenge",
		Data: map[string]interface{}{
			"submission_id":      sub.ID.String(),
			"company_id":         comp.ID.String(),
			"challenge_id":       challengeID,
			"challenge_category": string(existingChallenge.ChallengeCategory),
			"challenge_type":     string(existingChallenge.ChallengeType),
		},
	})
}

// RetryEnrichment handles POST /api/v1/admin/companies/:id/retry-enrichment
// Re-runs enrichment for a company using "fill gaps only" logic
// Only fills in NULL/empty fields - preserves any manually edited values
func (h *CompanyHandlers) RetryEnrichment(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Get company to verify it exists and check access
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve company"})
		return
	}
	if comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access (admin only for retry)
	if !h.checkCompanyAccess(c, comp) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied - admin only"})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Str("company_name", comp.Name).
		Str("current_status", comp.EnrichmentStatus).
		Msg("Admin triggered enrichment retry")

	// Run enrichment asynchronously (fire-and-forget)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.Logger.Error().
					Interface("panic", r).
					Str("company_id", companyID).
					Msg("Enrichment retry goroutine panicked")
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := h.CompanyService.RetryEnrichment(ctx, companyUUID); err != nil {
			h.Logger.Error().
				Err(err).
				Str("company_id", companyID).
				Msg("Enrichment retry failed")
		}
	}()

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Enrichment retry started (fill gaps only)",
		Data: map[string]interface{}{
			"company_id":      companyID,
			"company_name":    comp.Name,
			"previous_status": comp.EnrichmentStatus,
			"mode":            "fill_gaps_only",
		},
	})
}

// ReEnrichCompany handles POST /api/v1/admin/companies/:id/re-enrich
// Re-runs enrichment for a company with rate limiting (max 5 per company per 24 hours)
// Only fills in NULL/empty fields - preserves any manually edited values
func (h *CompanyHandlers) ReEnrichCompany(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Check rate limit first
	if !h.EnrichmentLimiter.CanEnrich(companyUUID) {
		retryAfter := h.EnrichmentLimiter.GetRetryAfter(companyUUID)
		count := h.EnrichmentLimiter.GetEnrichmentCount(companyUUID)

		h.Logger.Warn().
			Str("company_id", companyID).
			Int("enrichment_count_24h", count).
			Dur("retry_after", retryAfter).
			Msg("Rate limit exceeded for re-enrichment")

		c.JSON(http.StatusTooManyRequests, ErrorResponse{
			Error:   "Rate limit exceeded",
			Message: "Maximum 5 re-enrichments per day per company. Please try again later.",
		})
		// Set Retry-After header for client convenience
		c.Header("Retry-After", formatDuration(retryAfter))
		return
	}

	// Get company to verify it exists and check access
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve company"})
		return
	}
	if comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access (admin only for re-enrich)
	if !h.checkCompanyAccess(c, comp) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied - admin only"})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Str("company_name", comp.Name).
		Str("current_status", comp.EnrichmentStatus).
		Msg("Admin triggered re-enrichment (rate limited)")

	// Record the enrichment attempt for rate limiting
	h.EnrichmentLimiter.RecordEnrichment(companyUUID)

	// Run enrichment synchronously to capture results
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Call the enrichment service directly
	if err := h.CompanyService.RetryEnrichment(ctx, companyUUID); err != nil {
		h.Logger.Error().
			Err(err).
			Str("company_id", companyID).
			Msg("Re-enrichment failed")

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Enrichment failed",
			Message: err.Error(),
		})
		return
	}

	// Fetch updated company to get enrichment results
	updatedComp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || updatedComp == nil {
		// Enrichment succeeded but can't fetch results - still return success
		h.Logger.Warn().Err(err).Str("company_id", companyID).Msg("Failed to fetch updated company after enrichment")
		c.JSON(http.StatusOK, MessageResponse{
			Message: "Re-enrichment completed",
			Data: map[string]interface{}{
				"company_id":   companyID,
				"company_name": comp.Name,
			},
		})
		return
	}

	// Count how many fields were enriched (non-null values)
	fieldsUpdated := countEnrichedFields(updatedComp)

	h.Logger.Info().
		Str("company_id", companyID).
		Str("company_name", updatedComp.Name).
		Str("enrichment_status", updatedComp.EnrichmentStatus).
		Int("fields_updated", fieldsUpdated).
		Msg("Re-enrichment completed successfully")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Re-enrichment completed",
		Data: map[string]interface{}{
			"company_id":      companyID,
			"company_name":    updatedComp.Name,
			"status":          updatedComp.EnrichmentStatus,
			"fields_updated":  fieldsUpdated,
			"enrichment_count_24h": h.EnrichmentLimiter.GetEnrichmentCount(companyUUID),
			"remaining_today": 5 - h.EnrichmentLimiter.GetEnrichmentCount(companyUUID),
		},
	})
}

// ========================================================================
// HELPER FUNCTIONS
// ========================================================================

// parseIntParam safely parses an integer parameter with bounds
func parseIntParam(s string, min, max int) (int, error) {
	var val int
	err := json.Unmarshal([]byte(s), &val)
	if err != nil {
		return 0, err
	}
	if val < min {
		return min, nil
	}
	if val > max {
		return max, nil
	}
	return val, nil
}

// formatDuration formats a duration for the Retry-After header
// Returns duration in seconds or human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

// countEnrichedFields counts how many enriched fields have non-null values
func countEnrichedFields(comp *company.Company) int {
	count := 0

	// Basic fields
	if comp.CNPJ != nil && *comp.CNPJ != "" {
		count++
	}
	if comp.Website != nil && *comp.Website != "" {
		count++
	}
	if comp.Industry != nil && *comp.Industry != "" {
		count++
	}
	if comp.CompanySize != nil && *comp.CompanySize != "" {
		count++
	}
	if comp.Location != nil && *comp.Location != "" {
		count++
	}
	if comp.TargetMarket != nil && *comp.TargetMarket != "" {
		count++
	}
	if comp.FundingStage != nil && *comp.FundingStage != "" {
		count++
	}

	// Enriched fields
	if comp.FoundationYear != nil {
		count++
	}
	if comp.LegalName != nil && *comp.LegalName != "" {
		count++
	}
	if comp.Headquarters != nil && *comp.Headquarters != "" {
		count++
	}
	if comp.Sector != nil && *comp.Sector != "" {
		count++
	}
	if comp.TargetAudience != nil && *comp.TargetAudience != "" {
		count++
	}
	if comp.ValueProposition != nil && *comp.ValueProposition != "" {
		count++
	}
	if comp.EmployeesRange != nil && *comp.EmployeesRange != "" {
		count++
	}
	if comp.RevenueEstimate != nil && *comp.RevenueEstimate != "" {
		count++
	}
	if comp.BusinessModel != nil && *comp.BusinessModel != "" {
		count++
	}
	if comp.MarketShareStatus != nil && *comp.MarketShareStatus != "" {
		count++
	}
	if comp.DigitalMaturity != nil {
		count++
	}

	// Arrays
	if comp.Competitors != nil && len(comp.Competitors) > 0 {
		count++
	}
	if comp.Strengths != nil && len(comp.Strengths) > 0 {
		count++
	}
	if comp.Weaknesses != nil && len(comp.Weaknesses) > 0 {
		count++
	}
	if comp.MainProducts != nil && len(comp.MainProducts) > 0 {
		count++
	}
	if comp.RecentNews != nil && len(comp.RecentNews) > 0 {
		count++
	}
	if comp.KeyExecutives != nil && len(comp.KeyExecutives) > 0 {
		count++
	}
	if comp.Opportunities != nil && len(comp.Opportunities) > 0 {
		count++
	}
	if comp.Threats != nil && len(comp.Threats) > 0 {
		count++
	}

	// Strategic fields
	if comp.CompetitiveAdvantage != nil && *comp.CompetitiveAdvantage != "" {
		count++
	}
	if comp.MarketShare != nil && *comp.MarketShare != "" {
		count++
	}
	if comp.TAMEstimate != nil && *comp.TAMEstimate != "" {
		count++
	}
	if comp.SAMEstimate != nil && *comp.SAMEstimate != "" {
		count++
	}
	if comp.SOMEstimate != nil && *comp.SOMEstimate != "" {
		count++
	}

	// Social
	if comp.LinkedInURL != nil && *comp.LinkedInURL != "" {
		count++
	}
	if comp.TwitterHandle != nil && *comp.TwitterHandle != "" {
		count++
	}

	return count
}
