package api

import (
	"net/http"
	"time"

	"backend_v3/domain/macroeconomics"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// MacroHandlers handles macroeconomics-related admin endpoints
type MacroHandlers struct {
	MacroService *macroeconomics.Service
	Logger       zerolog.Logger
}

// NewMacroHandlers creates a new macro handler set
func NewMacroHandlers(macroSvc *macroeconomics.Service, logger zerolog.Logger) *MacroHandlers {
	return &MacroHandlers{
		MacroService: macroSvc,
		Logger:       logger,
	}
}

// GetLatestSnapshot handles GET /api/v1/admin/macro/latest
// Returns the most recent values for SELIC, IPCA, USD/BRL from the database
func (h *MacroHandlers) GetLatestSnapshot(c *gin.Context) {
	snapshot, err := h.MacroService.GetLatestSnapshot(c.Request.Context())
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to get latest macro snapshot")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve macro data",
			Message: err.Error(),
		})
		return
	}

	if snapshot == nil || !snapshot.HasData() {
		c.JSON(http.StatusOK, gin.H{
			"message":  "No macro data available. Run refresh to populate.",
			"snapshot": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"snapshot": snapshot,
	})
}

// RefreshAll handles POST /api/v1/admin/macro/refresh
// Triggers a refresh of all macro indicators (SELIC, IPCA, USD/BRL)
func (h *MacroHandlers) RefreshAll(c *gin.Context) {
	h.Logger.Info().Msg("Admin triggered macro refresh all")

	err := h.MacroService.RefreshAll(c.Request.Context())
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to refresh macro data")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to refresh macro data",
			Message: err.Error(),
		})
		return
	}

	// Get the updated snapshot to return
	snapshot, _ := h.MacroService.GetLatestSnapshot(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"message":  "Macro data refreshed successfully",
		"snapshot": snapshot,
	})
}

// RefreshIndicator handles POST /api/v1/admin/macro/refresh/:code
// Triggers a refresh of a specific indicator (selic, ipca, usd_brl)
func (h *MacroHandlers) RefreshIndicator(c *gin.Context) {
	code := c.Param("code")

	// Validate indicator code
	validCodes := map[string]bool{
		"selic":   true,
		"ipca":    true,
		"usd_brl": true,
	}

	if !validCodes[code] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid indicator code",
			Message: "Valid codes are: selic, ipca, usd_brl",
		})
		return
	}

	h.Logger.Info().Str("code", code).Msg("Admin triggered macro refresh for indicator")

	err := h.MacroService.FetchIndicator(c.Request.Context(), code)
	if err != nil {
		h.Logger.Error().Err(err).Str("code", code).Msg("Failed to refresh indicator")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to refresh indicator",
			Message: err.Error(),
		})
		return
	}

	// Get the updated value
	snapshot, _ := h.MacroService.GetLatestSnapshot(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"message":   "Indicator refreshed successfully",
		"indicator": code,
		"snapshot":  snapshot,
	})
}

// GetHistory handles GET /api/v1/admin/macro/history/:code
// Returns historical values for a specific indicator
// Query params: from, to (optional, defaults to last 30 days)
func (h *MacroHandlers) GetHistory(c *gin.Context) {
	code := c.Param("code")

	// Validate indicator code
	validCodes := map[string]bool{
		"selic":   true,
		"ipca":    true,
		"usd_brl": true,
	}

	if !validCodes[code] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid indicator code",
			Message: "Valid codes are: selic, ipca, usd_brl",
		})
		return
	}

	// Default to last 30 days
	to := time.Now()
	from := to.AddDate(0, 0, -30)

	// Parse optional from/to query params
	if fromStr := c.Query("from"); fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed
		}
	}

	history, err := h.MacroService.GetHistory(c.Request.Context(), code, from, to)
	if err != nil {
		h.Logger.Error().Err(err).Str("code", code).Msg("Failed to get indicator history")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve history",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"indicator": code,
		"count":     len(history.Values),
		"from":      history.From,
		"to":        history.To,
		"history":   history.Values,
	})
}
