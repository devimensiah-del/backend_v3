package api

import (
	"net/http"

	"backend_v3/domain/framework"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// FrameworkHandlers handles framework-related endpoints
type FrameworkHandlers struct {
	frameworkService *framework.Service
	logger           zerolog.Logger
}

// NewFrameworkHandlers creates a new framework handlers instance
func NewFrameworkHandlers(service *framework.Service, logger zerolog.Logger) *FrameworkHandlers {
	return &FrameworkHandlers{
		frameworkService: service,
		logger:           logger.With().Str("component", "framework_handlers").Logger(),
	}
}

// CreateFrameworkRequest is the request body for creating a framework
type CreateFrameworkRequest struct {
	Code        string   `json:"code" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	NamePT      string   `json:"name_pt" binding:"required"`
	Category    string   `json:"category" binding:"required"`
	Description string   `json:"description"`
	LayerOrder  int      `json:"layer_order" binding:"required,min=1"`
	DependsOn   []string `json:"depends_on"`
	IsActive    bool     `json:"is_active"`
}

// UpdateFrameworkRequest is the request body for updating a framework
type UpdateFrameworkRequest struct {
	Name        string   `json:"name"`
	NamePT      string   `json:"name_pt"`
	Category    string   `json:"category"`
	Description *string  `json:"description"`
	LayerOrder  *int     `json:"layer_order"`
	DependsOn   []string `json:"depends_on"`
	IsActive    *bool    `json:"is_active"`
}

// List handles GET /api/v1/frameworks
// Returns only active frameworks for public consumption
func (h *FrameworkHandlers) List(c *gin.Context) {
	frameworks, err := h.frameworkService.ListActive(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list active frameworks")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve frameworks",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": frameworks,
		"meta": gin.H{
			"total": len(frameworks),
		},
	})
}

// GetByCode handles GET /api/v1/frameworks/:code
// Returns a framework by its unique code
func (h *FrameworkHandlers) GetByCode(c *gin.Context) {
	code := c.Param("code")

	fw, err := h.frameworkService.GetByCode(c.Request.Context(), code)
	if err != nil {
		h.logger.Error().Err(err).Str("code", code).Msg("Failed to get framework by code")
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Framework not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": fw,
	})
}

// AdminList handles GET /api/v1/admin/frameworks
// Returns all frameworks including inactive ones
func (h *FrameworkHandlers) AdminList(c *gin.Context) {
	frameworks, err := h.frameworkService.List(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list all frameworks (admin)")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve frameworks",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": frameworks,
		"meta": gin.H{
			"total": len(frameworks),
		},
	})
}

// AdminCreate handles POST /api/v1/admin/frameworks
// Creates a new framework
func (h *FrameworkHandlers) AdminCreate(c *gin.Context) {
	var req CreateFrameworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid create framework request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}

	fw := &framework.Framework{
		ID:          uuid.New(),
		Code:        req.Code,
		Name:        req.Name,
		NamePT:      req.NamePT,
		Category:    req.Category,
		Description: desc,
		LayerOrder:  req.LayerOrder,
		DependsOn:   req.DependsOn,
		IsActive:    req.IsActive,
	}

	if err := h.frameworkService.Create(c.Request.Context(), fw); err != nil {
		h.logger.Error().Err(err).Str("code", req.Code).Msg("Failed to create framework")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create framework",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("id", fw.ID.String()).
		Str("code", fw.Code).
		Msg("Framework created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"data": fw,
	})
}

// AdminUpdate handles PUT /api/v1/admin/frameworks/:id
// Updates an existing framework
func (h *FrameworkHandlers) AdminUpdate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid framework ID",
			Message: err.Error(),
		})
		return
	}

	var req UpdateFrameworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid update framework request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Fetch existing framework to get current values
	// We need to get by ID - for now we'll construct a full update
	// In production, you'd fetch the existing framework first
	fw := &framework.Framework{
		ID: id,
	}

	// Apply updates (only non-nil/non-empty fields)
	if req.Name != "" {
		fw.Name = req.Name
	}
	if req.NamePT != "" {
		fw.NamePT = req.NamePT
	}
	if req.Category != "" {
		fw.Category = req.Category
	}
	if req.Description != nil {
		fw.Description = req.Description
	}
	if req.LayerOrder != nil {
		fw.LayerOrder = *req.LayerOrder
	}
	if req.DependsOn != nil {
		fw.DependsOn = req.DependsOn
	}
	if req.IsActive != nil {
		fw.IsActive = *req.IsActive
	}

	if err := h.frameworkService.Update(c.Request.Context(), fw); err != nil {
		h.logger.Error().Err(err).Str("id", idStr).Msg("Failed to update framework")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update framework",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("id", id.String()).
		Msg("Framework updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"data": fw,
	})
}

// AdminDeactivate handles DELETE /api/v1/admin/frameworks/:id
// Soft-deletes a framework by setting is_active to false
func (h *FrameworkHandlers) AdminDeactivate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid framework ID",
			Message: err.Error(),
		})
		return
	}

	if err := h.frameworkService.Deactivate(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("id", idStr).Msg("Failed to deactivate framework")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to deactivate framework",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("id", id.String()).
		Msg("Framework deactivated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Framework deactivated successfully",
	})
}
