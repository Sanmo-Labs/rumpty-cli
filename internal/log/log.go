package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const EnvLevel = "RUMPTY_LOG_LEVEL"

// Configure sets the default slog logger. level: error, warn, info, debug.
func Configure(level string, w io.Writer) error {
	lvl, err := ParseLevel(level)
	if err != nil {
		return err
	}
	if w == nil {
		w = os.Stderr
	}
	h := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: lvl == slog.LevelDebug,
	})
	slog.SetDefault(slog.New(h))
	return nil
}

func ParseLevel(level string) (slog.Leveler, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error", "":
		return slog.LevelError, nil
	default:
		return nil, fmt.Errorf("unknown log level %q (use error, warn, info, or debug)", level)
	}
}

func Debug(msg string, args ...any) { slog.Debug(msg, args...) }
func Info(msg string, args ...any)  { slog.Info(msg, args...) }
func Warn(msg string, args ...any)  { slog.Warn(msg, args...) }
func Error(msg string, args ...any) { slog.Error(msg, args...) }
