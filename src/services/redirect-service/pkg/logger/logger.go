package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log *zap.Logger
)

type Options struct {
	Level   string
	Format  string
	Service string
	Version string
	Commit  string
}

func Init(opts Options) error {
	level, err := zapcore.ParseLevel(strings.ToLower(opts.Level))
	if err != nil {
		return fmt.Errorf("invalid LOG_LEVEL %q: %w", opts.Level, err)
	}

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoding := "json"
	if strings.EqualFold(opts.Format, "console") {
		encoding = "console"
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	zapCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      false,
		Sampling:         nil,
		Encoding:         encoding,
		EncoderConfig:    encoderCfg,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		InitialFields: map[string]any{
			"service": opts.Service,
			"version": opts.Version,
			"commit":  opts.Commit,
		},
	}

	logger, err := zapCfg.Build()
	if err != nil {
		return fmt.Errorf("building logger: %w", err)
	}

	Log = logger

	return nil
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
