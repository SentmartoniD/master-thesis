package handlers

import (
	"redirect-service/internal/metrics"
	"redirect-service/internal/repositories"
	"redirect-service/internal/server/middlewares"
	"redirect-service/internal/services"
	"redirect-service/platform/database"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter builds the public engine: the redirect hot path and nothing else.
func SetupRouter(publisher ClickPublisher) *gin.Engine {
	router := newEngine()
	router.Use(metrics.Instrument())

	linkRepository := repositories.NewLinkRepository(database.Pool)
	linkService := services.NewLinkService(linkRepository)
	redirectHandler := NewRedirectHandler(linkService, publisher)

	// redirect router
	router.GET("/:code", redirectHandler.Redirect)

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
	// if panic happens return 500
	router.Use(middlewares.Recovery())

	return router
}
