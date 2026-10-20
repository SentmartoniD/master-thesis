package handlers

import (
	"context"
	"errors"
	"net/http"
	"redirect-service/internal/models"
	"redirect-service/internal/services"
	"redirect-service/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LinkService interface {
	GetLinkByCode(ctx context.Context, code string) (string, error)
}

type ClickPublisher interface {
	Publish(event models.ClickEvent)
}

type RedirectHandler struct {
	linkService LinkService
	publisher   ClickPublisher
}

func NewRedirectHandler(linkService LinkService, publisher ClickPublisher) *RedirectHandler {
	return &RedirectHandler{linkService: linkService, publisher: publisher}
}

func (h *RedirectHandler) Redirect(c *gin.Context) {
	code := c.Param("code")

	targetURL, err := h.linkService.GetLinkByCode(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, services.ErrorLinkNotFound) {
			c.Status(http.StatusNotFound)
			return
		}

		logger.Log.Error("failed to resolve code",
			zap.String("code", code),
			zap.Error(err),
		)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, targetURL)

	// publish message for the analytics-service
	h.publisher.Publish(models.ClickEvent{
		Code:       code,
		OccurredAt: time.Now().UTC(),
		Referrer:   c.Request.Referer(),
		UserAgent:  c.Request.UserAgent(),
	})
}
