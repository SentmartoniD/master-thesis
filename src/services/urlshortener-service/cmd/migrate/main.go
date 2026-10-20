package main

import (
	"fmt"
	"os"
	"urlshortener-service/internal/config"
	"urlshortener-service/pkg/logger"
	"urlshortener-service/platform/database/migrations"

	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("urlshortener-service-migrate")
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

	logger.Log.Info("applying migrations",
		zap.String("db_host", cfg.DBHost),
		zap.String("db_name", cfg.DBName),
	)

	if err := migrations.Run(cfg.GetMigrationPostgreSQLConnectionString()); err != nil {
		logger.Log.Error("migrations failed", zap.Error(err))
		return err
	}

	logger.Log.Info("migrations applied")

	return nil
}
