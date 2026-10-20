package server

import (
	"log"
	"net/http"
	"redirect-service/internal/config"
	"redirect-service/pkg/logger"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates the public http server.
func New(cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr(),
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          stdLog("http"),
	}
}

// New creates the metrics http server.
func NewAdmin(cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.AdminAddr(),
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.AdminWriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          stdLog("admin"),
	}
}

func stdLog(name string) *log.Logger {
	l, err := zap.NewStdLogAt(logger.Log.Named(name), zapcore.WarnLevel)
	if err != nil {
		return nil
	}
	return l
}
