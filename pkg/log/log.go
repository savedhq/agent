package log

import (
	"agent/internal/config"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New returns a new zerolog.Logger based on the provided configuration
func New(cfg config.LogConfig) zerolog.Logger {
	var writer io.Writer
	if strings.ToLower(cfg.Path) == "stdout" {
		writer = os.Stdout
	} else {
		writer = &lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
	}

	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	return zerolog.New(writer).With().Timestamp().Logger().Level(level)
}
