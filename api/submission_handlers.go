package api

import (
	"backend_v3/domain/company"
	"backend_v3/domain/submission"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	AuthHandlers              *AuthHandlers // For auto-creating users on submission
}

// NewSubmissionHandlers creates a new submission.Handlers instance with all dependencies
func NewSubmissionHandlers(
	submissionSvc *submission.Service,
	logger zerolog.Logger,
	submissionResponseBuilder *SubmissionResponseBuilder,
	authHandlers *AuthHandlers,
) *SubmissionHandlers {
	return &SubmissionHandlers{
		SubmissionService:         submissionSvc,
		Logger:                    logger,
		SubmissionResponseBuilder: submissionResponseBuilder,
		AuthHandlers:              authHandlers,
	}
}

func (h *SubmissionHandlers) CreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, h.Logger, http.StatusBadRequest, err, "Requisição inválida: verifique os campos enviados.")
		return
	}

	// Validate website: required unless hasNoWebsite is true
	if !req.HasNoWebsite && (req.Website == nil || strings.TrimSpace(*req.Website) == "") {
		respondError(c, h.Logger, http.StatusBadRequest, fmt.Errorf("website required"), "Website é obrigatório (ou marque 'empresa não possui website').")
		return
	}

	// Normalize website URL: add https:// if missing scheme
	var normalizedWebsite *string
	if req.Website != nil && *req.Website != "" {
		website := strings.TrimSpace(*req.Website)
		if !strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://") {
			website = "https://" + website
		}
		normalizedWebsite = &website
	}

	// Get authenticated user ID if available
	var userID *uuid.UUID
	var tempAuth *TemporaryAuth
	if rawUserID, exists := c.Get("userID"); exists {
		if userIDStr, ok := rawUserID.(string); ok {
			if uid, err := parseUUID(userIDStr); err == nil {
				userID = &uid
			}
		}
	}

	// If user is NOT authenticated, auto-create account or get existing
	if userID == nil && h.AuthHandlers != nil {
		newUserID, err := h.AuthHandlers.CreateUserWithoutPassword(req.ContactEmail, req.ContactName)
		if err != nil {
			// Check if user already exists - get their ID and issue token
			if strings.Contains(err.Error(), "USER_EXISTS") || strings.Contains(err.Error(), "email_exists") {
				// Look up existing user by email
				existingUserID, lookupErr := h.AuthHandlers.GetUserIDByEmail(req.ContactEmail)
				if lookupErr != nil {
					h.Logger.Warn().Err(lookupErr).Str("email", req.ContactEmail).Msg("Failed to lookup existing user")
				} else {
					uid, _ := uuid.Parse(existingUserID)
					userID = &uid

					// Issue temporary token for existing user
					token, tokenErr := h.AuthHandlers.IssueTemporaryToken(existingUserID, "user", req.ContactEmail)
					if tokenErr != nil {
						h.Logger.Error().Err(tokenErr).Msg("Failed to issue temporary token for existing user")
					} else {
						tempAuth = &TemporaryAuth{
							AccessToken: token,
							ExpiresIn:   3600,
							User: AuthUser{
								ID:    existingUserID,
								Email: req.ContactEmail,
								Role:  "user",
							},
						}
					}
					h.Logger.Info().
						Str("user_id", existingUserID).
						Str("email", req.ContactEmail).
						Msg("Issuing token for existing user submission")
				}
			} else {
				// Other error - log and continue without user
				h.Logger.Warn().Err(err).Str("email", req.ContactEmail).Msg("Failed to auto-create user, continuing with anonymous submission")
			}
		} else {
			// User created successfully
			uid, _ := uuid.Parse(newUserID)
			userID = &uid

			// Generate temporary 1-hour token
			token, tokenErr := h.AuthHandlers.IssueTemporaryToken(newUserID, "user", req.ContactEmail)
			if tokenErr != nil {
				h.Logger.Error().Err(tokenErr).Msg("Failed to issue temporary token")
			} else {
				tempAuth = &TemporaryAuth{
					AccessToken: token,
					ExpiresIn:   3600, // 1 hour
					User: AuthUser{
						ID:    newUserID,
						Email: req.ContactEmail,
						Role:  "user",
					},
				}
			}

			h.Logger.Info().
				Str("user_id", newUserID).
				Str("email", req.ContactEmail).
				Msg("Auto-created user for landing page submission")
		}
	}

	// Transform to domain model - simplified fields only
	// ContactPhone is now optional (*string)
	var contactPhone string
	if req.ContactPhone != nil {
		contactPhone = *req.ContactPhone
	}

	submitReq := &submission.SubmitRequest{
		CompanyName:    req.CompanyName,
		CNPJ:           req.CNPJ,
		CompanyWebsite: normalizedWebsite,
		ContactName:    req.ContactName,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   contactPhone,
		UserID:         userID,
	}

	// Use SubmitForm which saves to DB and creates Company
	// Note: Challenge is no longer created automatically
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

	// NOTE: SubmitForm() now handles:
	// 1. Creating Submission
	// 2. Creating Company (triggers async Perplexity enrichment)
	// Challenge creation is now separate (admin/user creates later)

	// Build response with optional auth
	now := time.Now()
	submissionResp := SubmissionResponse{
		ID:        resp.SubmissionID.String(),
		CompanyID: resp.CompanyID.String(),
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	// Return response with auth if user was auto-created
	if tempAuth != nil {
		c.JSON(http.StatusCreated, CreateSubmissionWithAuthResponse{
			Submission: submissionResp,
			Auth:       tempAuth,
		})
	} else {
		// Legacy response format for already-authenticated users or fallback
		c.JSON(http.StatusCreated, gin.H{
			"submission": submissionResp,
		})
	}
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
