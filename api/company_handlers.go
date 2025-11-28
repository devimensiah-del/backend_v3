package api

import (
	"encoding/json"
	"net/http"

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
	CompanyService    *company.Service
	SubmissionService *submission.Service
	EnrichmentService *enrichment.Service
	AsynqClient       *asynq.Client
	Logger            zerolog.Logger
}

// NewCompanyHandlers creates a new company handler set
func NewCompanyHandlers(
	companySvc *company.Service,
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	asynqClient *asynq.Client,
	logger zerolog.Logger,
) *CompanyHandlers {
	return &CompanyHandlers{
		CompanyService:    companySvc,
		SubmissionService: submissionSvc,
		EnrichmentService: enrichmentSvc,
		AsynqClient:       asynqClient,
		Logger:            logger,
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

// ReEnrichCompany handles POST /api/v1/admin/companies/:id/re-enrich
// Creates a new submission from company data and triggers enrichment
// Accepts optional JSON body with new_challenge to override the business challenge
func (h *CompanyHandlers) ReEnrichCompany(c *gin.Context) {
	companyID := c.Param("id")

	// Parse optional request body for challenge override
	var req ChallengeOverrideRequest
	_ = c.ShouldBindJSON(&req) // Ignore error - empty body is valid

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

	// Get last submission for business_challenge (fallback)
	lastSub, err := h.CompanyService.GetLastSubmission(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get last submission")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve last submission"})
		return
	}
	if lastSub == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No previous submission", Message: "No previous submission found for this company"})
		return
	}

	// Determine which challenge to use: new_challenge if provided, else last submission's
	challenge := lastSub.BusinessChallenge
	if req.NewChallenge != "" {
		challenge = req.NewChallenge
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

	// Create submission from company data
	sub, err := h.SubmissionService.CreateFromCompany(c.Request.Context(), submission.CreateFromCompanyInput{
		CompanyID:         companyUUID,
		CompanyName:       comp.Name,
		CNPJ:              comp.CNPJ,
		Website:           comp.Website,
		Industry:          comp.Industry,
		CompanySize:       comp.CompanySize,
		Location:          comp.Location,
		TargetMarket:      comp.TargetMarket,
		FundingStage:      comp.FundingStage,
		AnnualRevenueMin:  comp.AnnualRevenueMin,
		AnnualRevenueMax:  comp.AnnualRevenueMax,
		LinkedInURL:       comp.LinkedInURL,
		TwitterHandle:     comp.TwitterHandle,
		BusinessChallenge: challenge,
		ContactName:       contactName,
		ContactEmail:      contactEmail,
		UserID:            userID,
	})
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to create submission from company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Creation failed", Message: err.Error()})
		return
	}

	// Link submission to company (not primary)
	if err := h.CompanyService.LinkSubmission(c.Request.Context(), companyUUID, sub.ID, false, userID); err != nil {
		h.Logger.Warn().Err(err).Str("company_id", companyID).Msg("Failed to link submission to company")
		// Continue anyway - submission created successfully
	}

	// Create enrichment record
	enrich := enrichment.NewEnrichment(sub.ID)
	if err := h.EnrichmentService.Create(c.Request.Context(), enrich); err != nil {
		h.Logger.Error().Err(err).Str("submission_id", sub.ID.String()).Msg("Failed to create enrichment")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enrichment creation failed", Message: err.Error()})
		return
	}

	// Enqueue enrichment job
	payload := map[string]string{
		"submission_id": sub.ID.String(),
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask("enrichment_job", payloadBytes)
	if _, err := h.AsynqClient.Enqueue(task); err != nil {
		h.Logger.Error().Err(err).Msg("Failed to enqueue enrichment job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: "Failed to start enrichment"})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Str("submission_id", sub.ID.String()).
		Str("enrichment_id", enrich.ID.String()).
		Bool("new_challenge_provided", req.NewChallenge != "").
		Msg("Re-enrichment started from company data")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Re-enrichment started",
		Data: map[string]interface{}{
			"submission_id":          sub.ID.String(),
			"enrichment_id":          enrich.ID.String(),
			"company_id":             companyID,
			"new_challenge_provided": req.NewChallenge != "",
		},
	})
}

// ReAnalyzeCompany handles POST /api/v1/admin/companies/:id/re-analyze
// Creates a new submission, copies last enrichment, triggers analysis
// Accepts optional JSON body with new_challenge to override the business challenge
func (h *CompanyHandlers) ReAnalyzeCompany(c *gin.Context) {
	companyID := c.Param("id")

	// Parse optional request body for challenge override
	var req ChallengeOverrideRequest
	_ = c.ShouldBindJSON(&req) // Ignore error - empty body is valid

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

	// Get last submission for business_challenge (fallback)
	lastSub, err := h.CompanyService.GetLastSubmission(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get last submission")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve last submission"})
		return
	}
	if lastSub == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No previous submission", Message: "No previous submission found for this company"})
		return
	}

	// Determine which challenge to use: new_challenge if provided, else last submission's
	challenge := lastSub.BusinessChallenge
	if req.NewChallenge != "" {
		challenge = req.NewChallenge
	}

	// Get last completed enrichment
	lastEnrich, err := h.CompanyService.GetLastCompletedEnrichment(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get last completed enrichment")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve last enrichment"})
		return
	}
	if lastEnrich == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No completed enrichment", Message: "No completed enrichment found for this company"})
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

	// Create submission from company data
	sub, err := h.SubmissionService.CreateFromCompany(c.Request.Context(), submission.CreateFromCompanyInput{
		CompanyID:         companyUUID,
		CompanyName:       comp.Name,
		CNPJ:              comp.CNPJ,
		Website:           comp.Website,
		Industry:          comp.Industry,
		CompanySize:       comp.CompanySize,
		Location:          comp.Location,
		TargetMarket:      comp.TargetMarket,
		FundingStage:      comp.FundingStage,
		AnnualRevenueMin:  comp.AnnualRevenueMin,
		AnnualRevenueMax:  comp.AnnualRevenueMax,
		LinkedInURL:       comp.LinkedInURL,
		TwitterHandle:     comp.TwitterHandle,
		BusinessChallenge: challenge,
		ContactName:       contactName,
		ContactEmail:      contactEmail,
		UserID:            userID,
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

	// Copy enrichment data (creates new enrichment with status=completed)
	newEnrich, err := h.EnrichmentService.CopyEnrichment(c.Request.Context(), lastEnrich.EnrichmentID, sub.ID)
	if err != nil {
		h.Logger.Error().Err(err).Str("source_enrichment_id", lastEnrich.EnrichmentID.String()).Msg("Failed to copy enrichment")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enrichment copy failed", Message: err.Error()})
		return
	}

	// Enqueue analysis job (worker will create the analysis record)
	payload := map[string]string{
		"submission_id": sub.ID.String(),
		"enrichment_id": newEnrich.ID.String(),
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask("analysis_job", payloadBytes)
	if _, err := h.AsynqClient.Enqueue(task); err != nil {
		h.Logger.Error().Err(err).Msg("Failed to enqueue analysis job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: "Failed to start analysis"})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Str("submission_id", sub.ID.String()).
		Str("enrichment_id", newEnrich.ID.String()).
		Bool("new_challenge_provided", req.NewChallenge != "").
		Msg("Re-analysis started from company data (enrichment copied)")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Re-analysis started",
		Data: map[string]interface{}{
			"submission_id":          sub.ID.String(),
			"enrichment_id":          newEnrich.ID.String(),
			"company_id":             companyID,
			"new_challenge_provided": req.NewChallenge != "",
		},
	})
}

// EnrichAndAnalyzeCompany handles POST /api/v1/admin/companies/:id/enrich-and-analyze
// Creates a new submission, runs enrichment with auto_trigger_analysis flag
// Accepts optional JSON body with new_challenge to override the business challenge
func (h *CompanyHandlers) EnrichAndAnalyzeCompany(c *gin.Context) {
	companyID := c.Param("id")

	// Parse optional request body for challenge override
	var req ChallengeOverrideRequest
	_ = c.ShouldBindJSON(&req) // Ignore error - empty body is valid

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

	// Get last submission for business_challenge (fallback)
	lastSub, err := h.CompanyService.GetLastSubmission(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get last submission")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve last submission"})
		return
	}
	if lastSub == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No previous submission", Message: "No previous submission found for this company"})
		return
	}

	// Determine which challenge to use: new_challenge if provided, else last submission's
	challenge := lastSub.BusinessChallenge
	if req.NewChallenge != "" {
		challenge = req.NewChallenge
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

	// Create submission from company data
	sub, err := h.SubmissionService.CreateFromCompany(c.Request.Context(), submission.CreateFromCompanyInput{
		CompanyID:         companyUUID,
		CompanyName:       comp.Name,
		CNPJ:              comp.CNPJ,
		Website:           comp.Website,
		Industry:          comp.Industry,
		CompanySize:       comp.CompanySize,
		Location:          comp.Location,
		TargetMarket:      comp.TargetMarket,
		FundingStage:      comp.FundingStage,
		AnnualRevenueMin:  comp.AnnualRevenueMin,
		AnnualRevenueMax:  comp.AnnualRevenueMax,
		LinkedInURL:       comp.LinkedInURL,
		TwitterHandle:     comp.TwitterHandle,
		BusinessChallenge: challenge,
		ContactName:       contactName,
		ContactEmail:      contactEmail,
		UserID:            userID,
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

	// Create enrichment record with auto_trigger_analysis = true
	enrich, err := h.EnrichmentService.CreateWithAutoAnalyze(c.Request.Context(), sub.ID)
	if err != nil {
		h.Logger.Error().Err(err).Str("submission_id", sub.ID.String()).Msg("Failed to create enrichment with auto-analyze")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enrichment creation failed", Message: err.Error()})
		return
	}

	// Enqueue enrichment job (analysis will auto-trigger after approval)
	payload := map[string]string{
		"submission_id": sub.ID.String(),
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask("enrichment_job", payloadBytes)
	if _, err := h.AsynqClient.Enqueue(task); err != nil {
		h.Logger.Error().Err(err).Msg("Failed to enqueue enrichment job")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Enqueue failed", Message: "Failed to start enrichment"})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Str("submission_id", sub.ID.String()).
		Str("enrichment_id", enrich.ID.String()).
		Bool("auto_trigger_analysis", true).
		Bool("new_challenge_provided", req.NewChallenge != "").
		Msg("Enrich-and-analyze started from company data")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Enrichment started, analysis will auto-trigger after approval",
		Data: map[string]interface{}{
			"submission_id":          sub.ID.String(),
			"enrichment_id":          enrich.ID.String(),
			"company_id":             companyID,
			"new_challenge_provided": req.NewChallenge != "",
		},
	})
}

// ========================================================================
// USER-FACING HANDLERS
// ========================================================================

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

// ChallengeOverrideRequest allows specifying a new business challenge for analysis
// If NewChallenge is empty, falls back to the last submission's challenge
type ChallengeOverrideRequest struct {
	NewChallenge string `json:"new_challenge"`
}

// AddUserRequest represents the request body for adding a user
type AddUserRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// AddUserToCompany handles POST /api/v1/companies/:id/users
// Allows owner to add a user to allowed_users
func (h *CompanyHandlers) AddUserToCompany(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: "user_id is required"})
		return
	}

	newUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user_id", Message: "Invalid user ID format"})
		return
	}

	// Get company
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check if user is owner (or admin)
	role, _ := c.Get("userRole")
	isAdmin := role == "admin" || role == "super_admin" || role == "service_role"

	userIDStr, exists := c.Get("userID")
	if !exists && !isAdmin {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	if !isAdmin {
		userID, _ := uuid.Parse(userIDStr.(string))
		if !comp.CanManageUsers(userID) {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Only the owner can add users"})
			return
		}
	}

	// Add user
	if err := h.CompanyService.AddUserToCompany(c.Request.Context(), companyUUID, newUserID); err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Str("new_user_id", req.UserID).Msg("Failed to add user to company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Update failed", Message: err.Error()})
		return
	}

	h.Logger.Info().Str("company_id", companyID).Str("added_user_id", req.UserID).Msg("User added to company")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "User added to company",
		Data: map[string]interface{}{
			"company_id": companyID,
			"user_id":    req.UserID,
		},
	})
}

// RemoveUserFromCompany handles DELETE /api/v1/companies/:id/users/:userId
// Allows owner to remove a user from allowed_users
func (h *CompanyHandlers) RemoveUserFromCompany(c *gin.Context) {
	companyID := c.Param("id")
	userToRemoveID := c.Param("userId")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	removeUserID, err := uuid.Parse(userToRemoveID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID", Message: "Invalid user ID format"})
		return
	}

	// Get company
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check if user is owner (or admin)
	role, _ := c.Get("userRole")
	isAdmin := role == "admin" || role == "super_admin" || role == "service_role"

	userIDStr, exists := c.Get("userID")
	if !exists && !isAdmin {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	if !isAdmin {
		userID, _ := uuid.Parse(userIDStr.(string))
		if !comp.CanManageUsers(userID) {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Only the owner can remove users"})
			return
		}
	}

	// Remove user
	if err := h.CompanyService.RemoveUserFromCompany(c.Request.Context(), companyUUID, removeUserID); err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Str("remove_user_id", userToRemoveID).Msg("Failed to remove user from company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Update failed", Message: err.Error()})
		return
	}

	h.Logger.Info().Str("company_id", companyID).Str("removed_user_id", userToRemoveID).Msg("User removed from company")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "User removed from company",
		Data: map[string]interface{}{
			"company_id": companyID,
			"user_id":    userToRemoveID,
		},
	})
}

// ========================================================================
// ADMIN HANDLERS
// ========================================================================

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
	submissions, _ := h.CompanyService.GetSubmissions(c.Request.Context(), companyUUID)
	history, _ := h.CompanyService.GetHistory(c.Request.Context(), companyUUID)
	enrichmentStatus, _ := h.CompanyService.GetLatestEnrichmentStatus(c.Request.Context(), companyUUID)
	enrichmentsHistory, _ := h.CompanyService.GetEnrichmentsHistory(c.Request.Context(), companyUUID)
	analysesHistory, _ := h.CompanyService.GetAnalysesHistory(c.Request.Context(), companyUUID)

	c.JSON(http.StatusOK, gin.H{
		"company":             comp,
		"submissions":         submissions,
		"history":             history,
		"enrichment_status":   enrichmentStatus,
		"enrichments_history": enrichmentsHistory,
		"analyses_history":    analysesHistory,
	})
}

// UpdateCompany handles PUT /api/v1/admin/companies/:id
// Updates company fields and auto-verifies them (admin only)
func (h *CompanyHandlers) UpdateCompany(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Parse request body
	var req struct {
		Fields map[string]interface{} `json:"fields"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	if len(req.Fields) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: "No fields provided"})
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

	// Update company fields with auto-verification
	updatedCompany, err := h.CompanyService.UpdateFieldsWithAutoVerification(c.Request.Context(), companyUUID, req.Fields, userID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to update company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Update failed", Message: err.Error()})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Int("fields_updated", len(req.Fields)).
		Msg("Company fields updated and verified")

	c.JSON(http.StatusOK, gin.H{
		"company": updatedCompany,
		"message": "Company updated successfully",
	})
}

// VerifyCompany handles POST /api/v1/admin/companies/:id/verify
// Marks a company as verified (locks CNPJ uniqueness)
func (h *CompanyHandlers) VerifyCompany(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	comp, err := h.CompanyService.VerifyCompany(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to verify company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Verification failed", Message: err.Error()})
		return
	}

	h.Logger.Info().Str("company_id", companyID).Msg("Company verified")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Company verified",
		Data: map[string]interface{}{
			"company_id":  companyID,
			"is_verified": comp.IsVerified,
			"cnpj":        comp.CNPJ,
		},
	})
}

// UnverifyCompany handles POST /api/v1/admin/companies/:id/unverify
// Removes verified status from a company
func (h *CompanyHandlers) UnverifyCompany(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	comp, err := h.CompanyService.UnverifyCompany(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to unverify company")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Unverification failed", Message: err.Error()})
		return
	}

	h.Logger.Info().Str("company_id", companyID).Msg("Company unverified")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Company unverified",
		Data: map[string]interface{}{
			"company_id":  companyID,
			"is_verified": comp.IsVerified,
		},
	})
}

// TransferOwnershipRequest represents the request body for transferring ownership
type TransferOwnershipRequest struct {
	NewOwnerID string `json:"new_owner_id" binding:"required"`
}

// TransferOwnership handles POST /api/v1/admin/companies/:id/transfer-ownership
// Transfers company ownership to a new user (admin only)
func (h *CompanyHandlers) TransferOwnership(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	var req TransferOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: "new_owner_id is required"})
		return
	}

	newOwnerID, err := uuid.Parse(req.NewOwnerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid new_owner_id", Message: "Invalid user ID format"})
		return
	}

	comp, err := h.CompanyService.TransferOwnership(c.Request.Context(), companyUUID, newOwnerID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Str("new_owner_id", req.NewOwnerID).Msg("Failed to transfer ownership")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Transfer failed", Message: err.Error()})
		return
	}

	h.Logger.Info().Str("company_id", companyID).Str("new_owner_id", req.NewOwnerID).Msg("Company ownership transferred")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Ownership transferred",
		Data: map[string]interface{}{
			"company_id":   companyID,
			"new_owner_id": req.NewOwnerID,
			"owner_id":     comp.OwnerID,
		},
	})
}

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

// ================================================================================
// FIELD VERIFICATION HANDLERS
// These endpoints manage field-level verification for company data
// Verified fields are protected from being overwritten by re-enrichment
// ================================================================================

// GetFieldVerifications handles GET /api/v1/admin/companies/:id/verifications
// Returns all verified fields for a company
func (h *CompanyHandlers) GetFieldVerifications(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Get company (also validates it exists)
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access
	if !h.checkCompanyAccess(c, comp) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Access denied"})
		return
	}

	// Get field verifications
	verifications, err := h.CompanyService.GetFieldVerifications(c.Request.Context(), companyUUID)
	if err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to get field verifications")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database error", Message: "Failed to retrieve verifications"})
		return
	}

	// Return list of verifiable fields with verification status
	verifiedSet := make(map[string]bool)
	for _, v := range verifications {
		verifiedSet[v.FieldName] = true
	}

	fields := make([]map[string]interface{}, 0, len(company.VerifiableFields))
	for _, fieldName := range company.VerifiableFields {
		field := map[string]interface{}{
			"field_name": fieldName,
			"verified":   verifiedSet[fieldName],
		}
		// Add verification details if verified
		for _, v := range verifications {
			if v.FieldName == fieldName {
				field["verified_at"] = v.VerifiedAt
				field["verified_by"] = v.VerifiedBy
				break
			}
		}
		fields = append(fields, field)
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Field verifications retrieved",
		Data: map[string]interface{}{
			"company_id":       companyID,
			"fields":           fields,
			"total_verifiable": len(company.VerifiableFields),
			"total_verified":   len(verifications),
		},
	})
}

// VerifyFieldRequest is the request body for field verification
type VerifyFieldRequest struct {
	FieldName string `json:"field_name" binding:"required"`
}

// VerifyField handles POST /api/v1/admin/companies/:id/verifications
// Marks a specific field as verified (protected from re-enrichment)
func (h *CompanyHandlers) VerifyField(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Parse request body
	var req VerifyFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: "field_name is required"})
		return
	}

	// Get company (validates exists)
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access (admin only for verification)
	role, _ := c.Get("userRole")
	if role != "admin" && role != "super_admin" && role != "service_role" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Only admins can verify fields"})
		return
	}

	// Get verified_by user ID
	var verifiedBy *uuid.UUID
	if userIDStr, exists := c.Get("userID"); exists {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			verifiedBy = &uid
		}
	}

	// Verify the field
	if err := h.CompanyService.VerifyField(c.Request.Context(), companyUUID, req.FieldName, verifiedBy); err != nil {
		h.Logger.Error().Err(err).
			Str("company_id", companyID).
			Str("field_name", req.FieldName).
			Msg("Failed to verify field")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Verification failed", Message: err.Error()})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Str("field_name", req.FieldName).
		Msg("Field verified successfully")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Field verified",
		Data: map[string]interface{}{
			"company_id": companyID,
			"field_name": req.FieldName,
			"verified":   true,
		},
	})
}

// UnverifyField handles DELETE /api/v1/admin/companies/:id/verifications/:field_name
// Removes verification from a field (allows future re-enrichment)
func (h *CompanyHandlers) UnverifyField(c *gin.Context) {
	companyID := c.Param("id")
	fieldName := c.Param("field_name")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	if fieldName == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: "field_name is required"})
		return
	}

	// Get company (validates exists)
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access (admin only)
	role, _ := c.Get("userRole")
	if role != "admin" && role != "super_admin" && role != "service_role" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Only admins can unverify fields"})
		return
	}

	// Unverify the field
	if err := h.CompanyService.UnverifyField(c.Request.Context(), companyUUID, fieldName); err != nil {
		h.Logger.Error().Err(err).
			Str("company_id", companyID).
			Str("field_name", fieldName).
			Msg("Failed to unverify field")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Unverification failed", Message: err.Error()})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Str("field_name", fieldName).
		Msg("Field unverified successfully")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Field unverified",
		Data: map[string]interface{}{
			"company_id": companyID,
			"field_name": fieldName,
			"verified":   false,
		},
	})
}

// VerifyFieldsRequest is the request body for bulk field verification
type VerifyFieldsRequest struct {
	FieldNames []string `json:"field_names" binding:"required"`
}

// VerifyFields handles POST /api/v1/admin/companies/:id/verifications/bulk
// Marks multiple fields as verified
func (h *CompanyHandlers) VerifyFields(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Parse request body
	var req VerifyFieldsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: "field_names array is required"})
		return
	}

	if len(req.FieldNames) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: "At least one field_name is required"})
		return
	}

	// Get company (validates exists)
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access (admin only)
	role, _ := c.Get("userRole")
	if role != "admin" && role != "super_admin" && role != "service_role" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Only admins can verify fields"})
		return
	}

	// Get verified_by user ID
	var verifiedBy *uuid.UUID
	if userIDStr, exists := c.Get("userID"); exists {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			verifiedBy = &uid
		}
	}

	// Verify multiple fields
	if err := h.CompanyService.VerifyFields(c.Request.Context(), companyUUID, req.FieldNames, verifiedBy); err != nil {
		h.Logger.Error().Err(err).
			Str("company_id", companyID).
			Int("field_count", len(req.FieldNames)).
			Msg("Failed to verify fields")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Verification failed", Message: err.Error()})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Int("field_count", len(req.FieldNames)).
		Msg("Fields verified successfully")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Fields verified",
		Data: map[string]interface{}{
			"company_id":      companyID,
			"field_names":     req.FieldNames,
			"fields_verified": len(req.FieldNames),
		},
	})
}

// VerifyAllFields handles POST /api/v1/admin/companies/:id/verifications/all
// Marks all fields as verified
func (h *CompanyHandlers) VerifyAllFields(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Get company (validates exists)
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access (admin only)
	role, _ := c.Get("userRole")
	if role != "admin" && role != "super_admin" && role != "service_role" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Only admins can verify fields"})
		return
	}

	// Get verified_by user ID
	var verifiedBy *uuid.UUID
	if userIDStr, exists := c.Get("userID"); exists {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			verifiedBy = &uid
		}
	}

	// Verify all fields
	if err := h.CompanyService.VerifyAllFields(c.Request.Context(), companyUUID, verifiedBy); err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to verify all fields")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Verification failed", Message: err.Error()})
		return
	}

	h.Logger.Info().
		Str("company_id", companyID).
		Int("field_count", len(company.VerifiableFields)).
		Msg("All fields verified successfully")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "All fields verified",
		Data: map[string]interface{}{
			"company_id":      companyID,
			"fields_verified": len(company.VerifiableFields),
		},
	})
}

// UnverifyAllFields handles DELETE /api/v1/admin/companies/:id/verifications/all
// Removes verification from all fields
func (h *CompanyHandlers) UnverifyAllFields(c *gin.Context) {
	companyID := c.Param("id")

	companyUUID, err := uuid.Parse(companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid company ID format"})
		return
	}

	// Get company (validates exists)
	comp, err := h.CompanyService.GetByID(c.Request.Context(), companyUUID)
	if err != nil || comp == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Company not found"})
		return
	}

	// Check access (admin only)
	role, _ := c.Get("userRole")
	if role != "admin" && role != "super_admin" && role != "service_role" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "Only admins can unverify fields"})
		return
	}

	// Unverify all fields
	if err := h.CompanyService.UnverifyAllFields(c.Request.Context(), companyUUID); err != nil {
		h.Logger.Error().Err(err).Str("company_id", companyID).Msg("Failed to unverify all fields")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Unverification failed", Message: err.Error()})
		return
	}

	h.Logger.Info().Str("company_id", companyID).Msg("All fields unverified successfully")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "All fields unverified",
		Data: map[string]interface{}{
			"company_id":        companyID,
			"fields_unverified": len(company.VerifiableFields),
		},
	})
}
