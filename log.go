package smarter

import (
	"context"
	"log/slog"
	"os"
)

// MaskSecret returns a secret, masked for logging: its last four characters, e.g. "****5150", or
// "None" when there is no secret.
func MaskSecret(secret string) string {
	if secret == "" {
		return "None"
	}
	if len(secret) > 8 {
		return "****" + secret[len(secret)-4:]
	}
	return "****"
}

// discardHandler drops every record.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (h discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h discardHandler) WithGroup(string) slog.Handler           { return h }

// newLogger returns the client's logger: the caller's, or one for SMARTER_LOG, or one that discards.
func newLogger(given *slog.Logger, level string) *slog.Logger {
	logger := given
	if logger == nil {
		switch level {
		case "debug":
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
		case "info":
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
		default:
			logger = slog.New(discardHandler{})
		}
	}
	return logger.With("component", "smarter")
}
