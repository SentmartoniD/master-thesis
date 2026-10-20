package handlers

import (
	"context"
	"errors"
	"net/http"
	"urlshortener-service/internal/models"
	"urlshortener-service/internal/services"
	"urlshortener-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LinkService interface {
	CreateLink(ctx context.Context, req models.CreateLinkRequest) (*models.LinkResponse, error)
	GetLink(ctx context.Context, code string) (*models.LinkResponse, error)
}

type LinkHandler struct {
	linkService LinkService
}

func NewLinkHandler(linkService LinkService) *LinkHandler {
	return &LinkHandler{linkService: linkService}
}

// CreateLinkHandler handles POST /api/v1/links.
func (h *LinkHandler) CreateLinkHandler(c *gin.Context) {
	var req models.CreateLinkRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Debug("invalid create link request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": `request body must contain a valid "url" field`})
		return
	}

	link, err := h.linkService.CreateLink(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrorInvalidTargetURL) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else if errors.Is(err, services.ErrorCodeExhausted) {
			logger.Log.Error("code allocation exhausted", zap.Int("attempts", services.MaxCodeAttempts))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}

		logger.Log.Error("failed to create link", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Header("Location", "/api/v1/links/"+link.Code)
	c.JSON(http.StatusCreated, link)
}

// GetLinkHandler handles GET /api/v1/links/:code.
func (h *LinkHandler) GetLinkHandler(c *gin.Context) {
	code := c.Param("code")

	link, err := h.linkService.GetLink(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, services.ErrorLinkNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		logger.Log.Error("failed to get link",
			zap.String("code", code),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, link)
}
