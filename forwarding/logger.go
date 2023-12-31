package forwarding

import (
	"log/slog"
	"os"
	"strings"

	"github.com/powerpuffpenguin/streamf/config"
)

func newLogger(conf *config.Logger) (log *slog.Logger, e error) {
	var level slog.Level
	switch strings.ToLower(conf.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: conf.Source,
	}))
	return
}
