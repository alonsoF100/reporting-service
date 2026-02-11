package logger

import (
	"log/slog"
	"os"

	"github.com/alonsoF100/reporting-service/internal/config"
)

func Setup(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	switch cfg.Logger.JSON {
	case true:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: ParseLevel(cfg.Logger.Level)})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: ParseLevel(cfg.Logger.Level)})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
