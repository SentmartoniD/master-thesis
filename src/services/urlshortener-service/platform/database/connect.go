package database

import (
	"context"
	"fmt"
	"urlshortener-service/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect(ctx context.Context, cfg *config.Config, tracer pgx.QueryTracer) error {
	poolCfg, err := pgxpool.ParseConfig(cfg.GetPostgreSQLConnectionString())
	if err != nil {
		return fmt.Errorf("parsing database DSN: %w", err)
	}

	poolCfg.MaxConns = cfg.DBMaxConns
	poolCfg.MinConns = cfg.DBMinConns
	poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.DBConnectTimeout
	poolCfg.MaxConnLifetimeJitter = cfg.DBMaxConnLifetime / 10

	poolCfg.ConnConfig.Tracer = tracer

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("creating connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DBConnectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return fmt.Errorf("pinging database: %w", err)
	}

	Pool = pool

	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

func Ping(ctx context.Context) error {
	if Pool == nil {
		return fmt.Errorf("database pool is not initialised")
	}
	return Pool.Ping(ctx)
}

func Stats() *pgxpool.Stat {
	if Pool == nil {
		return nil
	}
	return Pool.Stat()
}
