package logging

import (
	"log/slog"
	"os"
)

func Init(mode string, level slog.Level) {
	var handler slog.Handler

	switch mode {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	default:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	slog.SetDefault(slog.New(handler))
	slog.Info("logger initialized", "mode", mode, "level", level.String())
}

// New creates a contextual logger for specific components
func New(component string) *slog.Logger {
	return slog.Default().With("component", component)
}

// Initialize global logger (JSON mode)
// usage
// logging.Init("json", slog.LevelInfo)