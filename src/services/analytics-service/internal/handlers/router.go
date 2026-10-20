package handlers

import (
	"analytics-service/internal/metrics"
	"analytics-service/internal/server/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter builds the public engine: the redirect hot path and nothing else.
func SetupRouter(clickService ClickService) *gin.Engine {
	router := newEngine()
	router.Use(metrics.Instrument())

	analyticsHandler := NewAnalyticsHandler(clickService)

	apiRouter := router.Group("api/v1")

	// analytics router
	analyticsRouter := apiRouter.Group("analytics")
	analyticsRouter.GET(":code", analyticsHandler.GetCodeAnalyticsHandler)

	return router
}

func SetupAdminRouter(hh *HealthHandler) *gin.Engine {
	router := newEngine()

	router.GET("/livez", hh.Live)
	router.GET("/readyz", hh.Ready)

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return router
}

// newEngine returns a gin engine with the middlewares every server shares
func newEngine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(middlewares.Recovery())

	return router
}
