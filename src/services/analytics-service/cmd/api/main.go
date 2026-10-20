package main

import (
	"analytics-service/internal/config"
	"analytics-service/internal/consumers"
	"analytics-service/internal/handlers"
	"analytics-service/internal/metrics"
	"analytics-service/internal/repositories"
	"analytics-service/internal/server"
	"analytics-service/internal/services"
	"analytics-service/pkg/logger"
	"analytics-service/platform/database"
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "analytics-service failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("analytics-service")
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

	clickRepository := repositories.NewClickRepository(database.Pool)
	clickService := services.NewClickService(clickRepository)

	clickConsumer := consumers.New(consumers.Config{
		URL:           cfg.RabbitMQURL,
		Exchange:      cfg.RabbitMQExchange,
		Queue:         cfg.RabbitMQQueue,
		RoutingKey:    cfg.RabbitMQRoutingKey,
		Prefetch:      cfg.ConsumerPrefetch,
		BatchSize:     cfg.BatchSize,
		FlushInterval: cfg.FlushInterval,
	}, clickService)

	clickConsumer.Start()
	defer clickConsumer.Close()

	health := handlers.NewHealthHandler(database.Ping)

	return server.Run(
		ctx,
		cfg,
		handlers.SetupRouter(clickService),
		handlers.SetupAdminRouter(health),
		health.Drain,
	)
}
