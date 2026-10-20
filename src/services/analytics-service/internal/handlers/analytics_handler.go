package handlers

import (
	"analytics-service/internal/models"
	"analytics-service/pkg/logger"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ClickService is what this handler needs from the service layer.
type ClickService interface {
	GetCodeAnalytics(ctx context.Context, code string) (*models.CodeAnalytics, error)
}

type AnalyticsHandler struct {
	clickService ClickService
}

func NewAnalyticsHandler(clickService ClickService) *AnalyticsHandler {
	return &AnalyticsHandler{clickService: clickService}
}

// GetCodeAnalyticsHandler handles GET /api/v1/analytics/:code.
func (h *AnalyticsHandler) GetCodeAnalyticsHandler(c *gin.Context) {
	code := c.Param("code")

	analytics, err := h.clickService.GetCodeAnalytics(c.Request.Context(), code)
	if err != nil {
		logger.Log.Error("failed to get code analytics",
			zap.String("code", code),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}
