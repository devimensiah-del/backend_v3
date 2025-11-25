package api

import (
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog" // Needed for logger
)

// Handlers holds all service dependencies and configuration for submission handlers
type SubmissionHandlers struct {
	SubmissionService         *submission.Service
	EnrichmentService         *enrichment.Service
	AsynqClient               *asynq.Client
	Logger                    zerolog.Logger
	SubmissionResponseBuilder *SubmissionResponseBuilder
}

// NewSubmissionHandlers creates a new submission.Handlers instance with all dependencies
func NewSubmissionHandlers(
	submissionSvc *submission.Service,
	enrichmentSvc *enrichment.Service,
	asynqClient *asynq.Client,
	logger zerolog.Logger,
	submissionResponseBuilder *SubmissionResponseBuilder,
) *SubmissionHandlers {
	return &SubmissionHandlers{
		SubmissionService:         submissionSvc,
		EnrichmentService:         enrichmentSvc,
		AsynqClient:               asynqClient,
		Logger:                    logger,
		SubmissionResponseBuilder: submissionResponseBuilder,
	}
}

func (h *SubmissionHandlers) CreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, h.Logger, http.StatusBadRequest, err, "Requisição inválida: verifique os campos enviados.")
		return
	}

	// Parse additionalInfo JSON if provided
	var additionalInfo AdditionalInfoData
	if req.AdditionalInfo != nil && *req.AdditionalInfo != "" {
		if err := json.Unmarshal([]byte(*req.AdditionalInfo), &additionalInfo); err != nil {
			respondError(c, h.Logger, http.StatusBadRequest, err, "Formato de additionalInfo inválido.")
			return
		}
	}

	// Validate required contact fields from additionalInfo
	// Required: contactName, contactEmail
	if additionalInfo.ContactName == "" {
		respondError(c, h.Logger, http.StatusBadRequest, fmt.Errorf("missing contact name"), "Nome de contato é obrigatório (em additionalInfo).")
		return
	}
	if additionalInfo.ContactEmail == "" {
		respondError(c, h.Logger, http.StatusBadRequest, fmt.Errorf("missing contact email"), "Email de contato é obrigatório (em additionalInfo).")
		return
	}

	// Use defaults for optional fields if not provided
	if req.CNPJ == "" {
		req.CNPJ = "00.000.000/0000-00" // Default CNPJ
	}
	if req.Industry == "" {
		req.Industry = "Não especificado"
	}
	if req.CompanySize == "" {
		req.CompanySize = "Não especificado"
	}
	if req.StrategicGoal == "" {
		req.StrategicGoal = "Em definição"
	}
	// Note: CurrentChallenges deliberately left empty - will be populated during enrichment phase
	if req.CompetitivePosition == "" {
		req.CompetitivePosition = "Em análise"
	}

	// Get authenticated user ID if available
	var userID *uuid.UUID
	if rawUserID, exists := c.Get("userID"); exists {
		if userIDStr, ok := rawUserID.(string); ok {
			if uid, err := parseUUID(userIDStr); err == nil {
				userID = &uid
			}
		}
	}

	// Transform frontend format to domain model and trigger enrichment workflow
	submitReq := &submission.SubmitRequest{
		// Company Information
		CompanyName:     req.CompanyName,
		CNPJ:            stringToPtr(req.CNPJ),
		CompanyIndustry: stringToPtr(req.Industry),
		CompanySize:     stringToPtr(req.CompanySize),
		CompanyWebsite:  req.Website,

		// Contact Information
		ContactName:     additionalInfo.ContactName,
		ContactEmail:    additionalInfo.ContactEmail,
		ContactPhone:    stringToPtr(additionalInfo.ContactPhone),
		ContactPosition: stringToPtr(additionalInfo.ContactPosition),

		// Business Context
		CompanyLocation:  stringToPtr(additionalInfo.CompanyLocation),
		TargetMarket:     stringToPtr(additionalInfo.TargetMarket),
		AnnualRevenueMin: additionalInfo.AnnualRevenueMin,
		AnnualRevenueMax: additionalInfo.AnnualRevenueMax,
		FundingStage:     stringToPtr(additionalInfo.FundingStage),

		// Strategic information - concatenate with " | " separator, skipping empty fields
		BusinessChallenge: buildBusinessChallengeString(req.StrategicGoal, req.CurrentChallenges, req.CompetitivePosition),
		AdditionalNotes:   stringToPtr(additionalInfo.AdditionalNotes),
		LinkedInURL:       stringToPtr(additionalInfo.LinkedInURL),
		TwitterHandle:     stringToPtr(additionalInfo.TwitterHandle),

		// User association
		UserID: userID,
	}

	// Use SubmitForm which saves to DB
	sub, err := h.SubmissionService.SubmitForm(c.Request.Context(), submitReq)

	if err != nil {
		respondError(c, h.Logger, http.StatusInternalServerError, err, "Falha ao criar submissão.")
		return
	}

	ctx := c.Request.Context()

	// -------------------------------------------------------------------------
	// AUTO-TRIGGER ENRICHMENT WORKFLOW WITH IDEMPOTENCY GUARDS
	// -------------------------------------------------------------------------
	existing, err := h.EnrichmentService.GetBySubmissionID(ctx, sub.ID)
	if err == nil {
		// Enrichment already exists - skip duplicate enqueue
		// Valid statuses: pending, completed, approved
		h.Logger.Info().Str("sub_id", sub.ID.String()).Str("status", string(existing.Status)).Msg("Enrichment already exists - skipping duplicate enqueue")
	} else {
		if err != nil && err.Error() != "enrichment not found" {
			h.Logger.Error().Err(err).Str("sub_id", sub.ID.String()).Msg("Failed to lookup enrichment; creating new record anyway")
		}
		// If not found, create a fresh record for the UI and enqueue the worker.
		enrichmentRecord := enrichment.NewEnrichment(sub.ID)
		if createErr := h.EnrichmentService.Create(ctx, enrichmentRecord); createErr != nil {
			h.Logger.Error().Err(createErr).Str("sub_id", sub.ID.String()).Msg("Failed to create enrichment record")
		}

		if err := h.SubmissionService.TriggerEnrichmentProcess(ctx, sub); err != nil {
			h.Logger.Error().Err(err).Str("sub_id", sub.ID.String()).Msg("Failed to enqueue enrichment job")
		} else {
			h.Logger.Info().Str("sub_id", sub.ID.String()).Msg("Enrichment workflow auto-triggered")
		}
	}

	// Frontend expects wrapped response: { submission: {...} }
	c.JSON(http.StatusCreated, gin.H{
		"submission": SubmissionResponse{
			ID:          sub.ID.String(),
			CompanyName: sub.CompanyName,
			Status:      string(sub.Status),
			CreatedAt:   sub.CreatedAt,
			UpdatedAt:   sub.UpdatedAt,
		},
	})
}

// GetSubmission checks status and returns details
// SECURITY: Ensures user can only access their own submission
func (h *SubmissionHandlers) GetSubmission(c *gin.Context) {
	submissionID := c.Param("id")

	// 1. Parse and validate UUID
	subUUID, err := parseUUID(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid submission ID format"})
		return
	}

	// 2. Fetch Submission
	sub, err := h.SubmissionService.GetByID(c.Request.Context(), subUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found", Message: "Submission not found"})
		return
	}

	// 3. SECURITY CHECK: Ownership with safe type assertion
	requesterID, hasUser := c.Get("userID")
	requesterRole, _ := c.Get("userRole")

	// Safe type assertion for userID
	userIDStr, ok := requesterID.(string)
	if hasUser && !ok {
		h.Logger.Error().Msgf("Invalid userID type: %T", requesterID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal error", Message: "Authentication error"})
		return
	}

	// If the submission belongs to a user (sub.UserID is not nil)
	// AND the requester is not the owner
	// AND the requester is not an admin
	if sub.UserID != nil && hasUser {
		if sub.UserID.String() != userIDStr {
			// Check if Admin bypass
			if requesterRole != "admin" && requesterRole != "super_admin" {
				h.Logger.Warn().Str("sub_id", submissionID).Str("requester", userIDStr).Msg("Unauthorized access attempt")
				c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "You do not have permission to view this submission"})
				return
			}
		}
	}

	// 4. Build Response with all fields
	resp := h.SubmissionResponseBuilder.buildSubmissionDetailResponse(sub, c.Request.Context(), submissionID)
	c.JSON(http.StatusOK, gin.H{"submission": resp})
}

// ListUserSubmissions handles GET /api/v1/submissions (authenticated user's submissions only)
// Filters submissions by matching the user's email with submission contact_email
func (h *SubmissionHandlers) ListUserSubmissions(c *gin.Context) {
	// Get authenticated user email (preferred) or ID
	userEmail, emailExists := c.Get("userEmail")
	userID, idExists := c.Get("userID")

	if !emailExists && !idExists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	// Parse query parameters (frontend sends page and pageSize)
	page := 1
	limit := 20
	status := c.Query("status")

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	// Frontend sends "pageSize" but we also accept "limit"
	if ps := c.Query("pageSize"); ps != "" {
		fmt.Sscanf(ps, "%d", &limit)
	} else if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Build list options - prefer filtering by email for better matching
	opts := &submission.ListOptions{
		Limit:   limit,
		Offset:  offset,
		OrderBy: "created_at",
		Order:   "DESC",
	}

	// Primary filter: by user email (matches submission's contact_email)
	if emailExists {
		if emailStr, ok := userEmail.(string); ok && emailStr != "" {
			opts.Email = &emailStr
			h.Logger.Debug().Str("email", emailStr).Msg("Filtering submissions by user email")
		}
	}

	// Fallback filter: by user ID (if email not available)
	if opts.Email == nil && idExists {
		if userIDStr, ok := userID.(string); ok {
			userUUID, err := parseUUID(userIDStr)
			if err != nil {
				h.Logger.Error().Err(err).Str("user_id", userIDStr).Msg("Invalid user ID format")
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal error", Message: "Invalid user ID"})
				return
			}
			opts.UserID = &userUUID
			h.Logger.Debug().Str("user_id", userIDStr).Msg("Filtering submissions by user ID (email not available)")
		}
	}

	// Add status filter if provided
	if status != "" {
		s := submission.Status(status)
		opts.Status = &s
	}

	// Query submissions
	submissions, total, err := h.SubmissionService.ListAll(c.Request.Context(), opts)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to list user submissions")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve submissions",
			Message: "An error occurred while fetching your submissions",
		})
		return
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit // Ceiling division
	if totalPages < 1 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, SubmissionListResponse{
		Data:       submissions,
		Page:       page,
		PageSize:   limit,
		Total:      total,
		TotalPages: totalPages,
	})
}
