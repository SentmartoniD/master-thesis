package handlers

import (
	"urlshortener-service/internal/config"
	"urlshortener-service/internal/metrics"
	"urlshortener-service/internal/repositories"
	"urlshortener-service/internal/server/middlewares"
	"urlshortener-service/internal/services"
	"urlshortener-service/platform/database"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter builds the public gin api engine.
func SetupRouter(cfg *config.Config) *gin.Engine {
	router := newEngine()
	router.Use(metrics.Instrument())

	linkRepository := repositories.NewLinkRepository(database.Pool)
	linkService := services.NewLinkService(linkRepository, cfg)
	linkHandler := NewLinkHandler(linkService)

	apiRouter := router.Group("api/v1")

	// link router
	linkRouter := apiRouter.Group("links")
	linkRouter.POST("", linkHandler.CreateLinkHandler)
	linkRouter.GET(":code", linkHandler.GetLinkHandler)

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
