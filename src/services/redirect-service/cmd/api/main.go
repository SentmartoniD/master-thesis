package main

import (
	"context"
	"fmt"
	"os"
	"redirect-service/internal/config"
	"redirect-service/internal/handlers"
	"redirect-service/internal/metrics"
	"redirect-service/internal/publishers"
	"redirect-service/internal/server"
	"redirect-service/pkg/logger"
	"redirect-service/platform/database"

	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "redirect-service failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("redirect-service")
	if err != nil {
		return err
	}

	if err := logger.Init(logger.Options{
		Level:   cfg.LogLevel,
		Format:  cfg.LogFormat,
		Service: cfg.ServiceName,
		Version: cfg.BuildVersion,
		Commit:  cfg.BuildCommit,
	}); err != nil {
		return err
	}
	defer logger.Sync()

	// Every pod records what it actually loaded, so an anomalous experiment run
	// can be traced back to its exact configuration.
	logger.Log.Info("configuration loaded", zap.Any("config", cfg.GetFields()))

	ctx := context.Background()

	if err := database.Connect(ctx, cfg, &metrics.QueryTracer{}); err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer database.Close()

	logger.Log.Info("connected to database")

	metrics.RegisterDBPoolMetrics()

	var clickPublisher handlers.ClickPublisher = publishers.Noop{}

	// use RabbitMQ
	if cfg.ClickMode == "async" {
		p := publishers.New(publishers.Config{
			URL:        cfg.RabbitMQURL,
			Exchange:   cfg.RabbitMQExchange,
			RoutingKey: cfg.RabbitMQRoutingKey,
			BufferSize: cfg.ClickBufferSize,
		})
		p.Start()
		defer p.Close()

		metrics.RegisterPublisherMetrics(p)
		clickPublisher = p

		logger.Log.Info("click publishing enabled", zap.String("exchange", cfg.RabbitMQExchange))
	}
	// don't use RabbitMQ
	if cfg.ClickMode == "off" {
		logger.Log.Info("click publishing disabled")
	}

	health := handlers.NewHealthHandler(database.Ping)

	return server.Run(
		ctx,
		cfg,
		handlers.SetupRouter(clickPublisher),
		handlers.SetupAdminRouter(health),
		health.Drain,
	)
}
