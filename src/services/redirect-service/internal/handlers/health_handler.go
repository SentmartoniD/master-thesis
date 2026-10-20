package handlers

import (
	"context"
	"errors"
	"net/http"
	"redirect-service/pkg/logger"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const readinessPingTimeout = 2 * time.Second

// PingFunc reports whether the database is reachable.
type PingFunc func(ctx context.Context) error

type HealthHandler struct {
	ping PingFunc
	// false at startup, set to true exactly once on SIGTERM.
	draining atomic.Bool
}

func NewHealthHandler(ping PingFunc) *HealthHandler {
	return &HealthHandler{ping: ping}
}

// Drain sets draining true, called when the app is shutdown.
func (h *HealthHandler) Drain() {
	h.draining.Store(true)
}

// Live checks if the service is running.
func (h *HealthHandler) Live(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

// Ready checks is the app is shuting down and is the database working.
func (h *HealthHandler) Ready(c *gin.Context) {
	// check if app is shuting down
	if h.draining.Load() {
		c.String(http.StatusServiceUnavailable, "draining")
		return
	}

	// create a context that cancels after readinessPingTimeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessPingTimeout)
	defer cancel()

	// check if the database is running
	switch err := h.ping(ctx); {
	case err == nil:
		c.String(http.StatusOK, "ready")
	case errors.Is(err, context.DeadlineExceeded):
		logger.Log.Warn("readiness ping timed out, reporting ready anyway")
		c.String(http.StatusOK, "degraded")
	default:
		logger.Log.Error("readiness ping failed", zap.Error(err))
		c.String(http.StatusServiceUnavailable, "database unreachable")
	}
}
