package server

import (
	"analytics-service/internal/config"
	"analytics-service/pkg/logger"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Run starts the public and admin servers and blocks until a termination
func Run(ctx context.Context, cfg *config.Config, public, admin http.Handler, drain func()) error {
	publicSrv := New(cfg, public)
	adminSrv := NewAdmin(cfg, admin)

	publicLn, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		return fmt.Errorf("binding %s: %w", cfg.Addr(), err)
	}

	adminLn, err := net.Listen("tcp", cfg.AdminAddr())
	if err != nil {
		publicLn.Close()
		return fmt.Errorf("binding %s: %w", cfg.AdminAddr(), err)
	}

	errCh := make(chan error, 2)

	// Start serving the public http server
	go func() {
		logger.Log.Info("public server listening", zap.String("addr", cfg.Addr()))
		if err := publicSrv.Serve(publicLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("public server: %w", err)
		}
	}()

	// Start serving the admin http server
	go func() {
		logger.Log.Info("admin server listening", zap.String("addr", cfg.AdminAddr()))
		if err := adminSrv.Serve(adminLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("admin server: %w", err)
		}
	}()

	signalCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-signalCtx.Done():
		logger.Log.Info("shutdown signal received")
	}

	return shutdown(cfg, publicSrv, adminSrv, drain)
}

func shutdown(cfg *config.Config, publicSrv, adminSrv *http.Server, drain func()) error {

	drain()
	logger.Log.Info("draining", zap.Duration("for", cfg.ReadinessDrain))
	time.Sleep(cfg.ReadinessDrain)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := publicSrv.Shutdown(ctx); err != nil {
		logger.Log.Error("public server shutdown", zap.Error(err))
	}
	if err := adminSrv.Shutdown(ctx); err != nil {
		logger.Log.Error("admin server shutdown", zap.Error(err))
	}

	logger.Log.Info("stopped")
	return nil
}
