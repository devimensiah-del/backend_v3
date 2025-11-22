package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- PUBLIC SUBMISSION ENDPOINTS ---

// CreateSubmission starts the workflow
func (h *Handler) CreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	// Parse additionalInfo JSON if provided
	var additionalInfo AdditionalInfoData
	if req.AdditionalInfo != nil && *req.AdditionalInfo != "" {
		if err := json.Unmarshal([]byte(*req.AdditionalInfo), &additionalInfo); err != nil {
			h.logger.Error().Err(err).Msg("Failed to parse additionalInfo")
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid additionalInfo format", Message: err.Error()})
			return
		}
	}

	// Validate required contact fields from additionalInfo
	// Required: contactName, contactEmail
	if additionalInfo.ContactName == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: "Contact name is required (provide in additionalInfo)",
		})
		return
	}
	if additionalInfo.ContactEmail == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: "Contact email is required (provide in additionalInfo)",
		})
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
	if req.CurrentChallenges == "" {
		req.CurrentChallenges = "A definir"
	}
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

		// Strategic information
		BusinessChallenge: fmt.Sprintf("%s | %s | %s", req.StrategicGoal, req.CurrentChallenges, req.CompetitivePosition),
		AdditionalNotes:   stringToPtr(additionalInfo.AdditionalNotes),
		LinkedInURL:       stringToPtr(additionalInfo.LinkedInURL),
		TwitterHandle:     stringToPtr(additionalInfo.TwitterHandle),

		// User association
		UserID: userID,
	}

	// Use SubmitForm which saves to DB AND triggers enrichment workflow
	sub, err := h.submissionSvc.SubmitForm(c.Request.Context(), submitReq)

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create submission")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Submission failed", Message: err.Error()})
		return
	}

	// Frontend expects wrapped response: { submission: {...} }
	c.JSON(http.StatusCreated, gin.H{
		"submission": SubmissionResponse{
			ID:            sub.ID.String(),
			CompanyName:   sub.CompanyName,
			CNPJ:          ptrToString(sub.CNPJ),
			Industry:      ptrToString(sub.CompanyIndustry),
			Status:        string(sub.Status),
			StatusMessage: "AI Agent activated. Analysis started.",
			CreatedAt:     sub.CreatedAt,
			UpdatedAt:     sub.UpdatedAt,
		},
	})
}

// GetSubmission checks status and returns details
// SECURITY: Ensures user can only access their own submission
func (h *Handler) GetSubmission(c *gin.Context) {
	submissionID := c.Param("id")

	// 1. Parse and validate UUID
	subUUID, err := parseUUID(submissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid ID", Message: "Invalid submission ID format"})
		return
	}

	// 2. Fetch Submission
	sub, err := h.submissionSvc.GetByID(c.Request.Context(), subUUID)
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
		h.logger.Error().Msgf("Invalid userID type: %T", requesterID)
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
				h.logger.Warn().Str("sub_id", submissionID).Str("requester", userIDStr).Msg("Unauthorized access attempt")
				c.JSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden", Message: "You do not have permission to view this submission"})
				return
			}
		}
	}

	// 4. Build Response with all fields
	resp := buildSubmissionDetailResponse(sub, h, c.Request.Context(), subUUID, submissionID)
	c.JSON(http.StatusOK, resp)
}

// ListUserSubmissions handles GET /api/v1/submissions (authenticated user's submissions only)
func (h *Handler) ListUserSubmissions(c *gin.Context) {
	// Get authenticated user ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized", Message: "User not authenticated"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		h.logger.Error().Msgf("Invalid userID type: %T", userID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal error", Message: "Authentication error"})
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

	// Parse user ID as UUID
	userUUID, err := parseUUID(userIDStr)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userIDStr).Msg("Invalid user ID format")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal error", Message: "Invalid user ID"})
		return
	}

	// Build list options for this user only
	opts := &submission.ListOptions{
		Limit:   limit,
		Offset:  offset,
		OrderBy: "created_at",
		Order:   "DESC",
		UserID:  &userUUID,  // Filter by user ID
	}

	// Add status filter if provided
	if status != "" {
		s := submission.Status(status)
		opts.Status = &s
	}

	// Query submissions
	submissions, total, err := h.submissionSvc.ListAll(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list user submissions")
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
