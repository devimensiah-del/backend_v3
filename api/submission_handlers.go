package api

import (
	"backend_v3/domain/company"
	"backend_v3/domain/submission"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog" // Needed for logger
)

// Handlers holds all service dependencies and configuration for submission handlers
type SubmissionHandlers struct {
	SubmissionService         *submission.Service
	Logger                    zerolog.Logger
	SubmissionResponseBuilder *SubmissionResponseBuilder
}

// NewSubmissionHandlers creates a new submission.Handlers instance with all dependencies
func NewSubmissionHandlers(
	submissionSvc *submission.Service,
	logger zerolog.Logger,
	submissionResponseBuilder *SubmissionResponseBuilder,
) *SubmissionHandlers {
	return &SubmissionHandlers{
		SubmissionService:         submissionSvc,
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
	if additionalInfo.ContactName == "" {
		respondError(c, h.Logger, http.StatusBadRequest, fmt.Errorf("missing contact name"), "Nome de contato é obrigatório (em additionalInfo).")
		return
	}
	if additionalInfo.ContactEmail == "" {
		respondError(c, h.Logger, http.StatusBadRequest, fmt.Errorf("missing contact email"), "Email de contato é obrigatório (em additionalInfo).")
		return
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

	// Transform frontend format to domain model
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

		// Challenge Definition (required - drives the analysis)
		ChallengeCategory: req.ChallengeCategory,
		ChallengeType:     req.ChallengeType,
		BusinessChallenge: req.BusinessChallenge,

		// Additional fields
		AdditionalNotes: stringToPtr(additionalInfo.AdditionalNotes),
		LinkedInURL:     stringToPtr(additionalInfo.LinkedInURL),
		TwitterHandle:   stringToPtr(additionalInfo.TwitterHandle),

		// User association
		UserID: userID,
	}

	// Use SubmitForm which saves to DB and creates Company + Challenge
	// SubmitForm no longer accepts CreateOptions - it's always public workflow
	resp, err := h.SubmissionService.SubmitForm(c.Request.Context(), submitReq)

	if err != nil {
		// Check for duplicate submitter error (same email + CNPJ combo)
		var dupErr *company.DuplicateSubmitterError
		if errors.As(err, &dupErr) {
			h.Logger.Warn().
				Str("company_id", dupErr.CompanyID.String()).
				Str("company_name", dupErr.CompanyName).
				Str("email", dupErr.Email).
				Msg("Duplicate submission attempt detected")

			c.JSON(http.StatusConflict, gin.H{
				"error": "Esta empresa já foi submetida com este email. Por favor, faça login para acompanhar sua análise.",
				"code":  "DUPLICATE_SUBMITTER",
				"details": gin.H{
					"company_id":   dupErr.CompanyID.String(),
					"company_name": dupErr.CompanyName,
					"action":       "login",
				},
			})
			return
		}

		respondError(c, h.Logger, http.StatusInternalServerError, err, "Falha ao criar submissão.")
		return
	}

	// NOTE: SubmitForm() handles:
	// 1. Creating Submission (status: always "received")
	// 2. Creating Company (triggers async Perplexity enrichment)
	// 3. Creating Challenge
	// Analysis is NOT auto-triggered - user must start via wizard or admin can use generate-all

	// Frontend expects wrapped response: { submission: {...} }
	now := time.Now()
	c.JSON(http.StatusCreated, gin.H{
		"submission": SubmissionResponse{
			ID:          resp.SubmissionID.String(),
			CompanyID:   resp.CompanyID.String(),
			ChallengeID: resp.ChallengeID.String(),
			CreatedAt:   &now,
			UpdatedAt:   &now,
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

	// NOTE: Status filter removed - submissions no longer have status field
	// Status is now tracked on related entities (Company.enrichment_status, Analysis.status)
	_ = status // Avoid unused variable error

	// Query submissions
	submissions, total, err := h.SubmissionService.List(c.Request.Context(), opts)
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
