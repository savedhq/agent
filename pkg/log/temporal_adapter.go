package log

import (
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/log"
)

// TemporalAdapter is a Temporal logger adapter for zerolog
type TemporalAdapter struct {
	logger zerolog.Logger
}

// NewTemporalAdapter creates a new TemporalAdapter
func NewTemporalAdapter(logger zerolog.Logger) *TemporalAdapter {
	return &TemporalAdapter{logger: logger}
}

// Debug logs a debug message
func (t *TemporalAdapter) Debug(msg string, keyvals ...interface{}) {
	t.logger.Debug().Fields(keyvals).Msg(msg)
}

// Info logs an info message
func (t *TemporalAdapter) Info(msg string, keyvals ...interface{}) {
	t.logger.Info().Fields(keyvals).Msg(msg)
}

// Warn logs a warning message
func (t *TemporalAdapter) Warn(msg string, keyvals ...interface{}) {
	t.logger.Warn().Fields(keyvals).Msg(msg)
}

// Error logs an error message
func (t *TemporalAdapter) Error(msg string, keyvals ...interface{}) {
	t.logger.Error().Fields(keyvals).Msg(msg)
}

// With returns a new logger with the given keyvals
func (t *TemporalAdapter) With(keyvals ...interface{}) log.Logger {
	return &TemporalAdapter{logger: t.logger.With().Fields(keyvals).Logger()}
}
