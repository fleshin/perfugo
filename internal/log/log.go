package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	levelVar = new(slog.LevelVar)
	loggerMu sync.RWMutex
	logger   = newLogger()
)

func init() {
	levelVar.Set(slog.LevelInfo)
}

func newLogger() *slog.Logger {
	return slog.New(newHandler(os.Stdout))
}

func newHandler(w io.Writer) slog.Handler {
	opts := slog.HandlerOptions{
		Level: levelVar,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			switch attr.Key {
			case slog.TimeKey:
				attr.Key = "ts"
				if attr.Value.Kind() == slog.KindTime {
					attr.Value = slog.StringValue(attr.Value.Time().UTC().Format(time.RFC3339Nano))
				}
			case slog.LevelKey:
				attr.Key = "level"
				attr.Value = slog.StringValue(strings.ToLower(attr.Value.String()))
			case slog.MessageKey:
				attr.Key = "msg"
			}
			return attr
		},
	}
	return slog.NewTextHandler(w, &opts)
}

// SetLevel updates the minimum logging level accepted by the global logger.
// Supported levels are "debug", "info", and "error". Values are case-insensitive.
func SetLevel(level string) error {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		levelVar.Set(slog.LevelInfo)
	case "debug":
		levelVar.Set(slog.LevelDebug)
	case "error":
		levelVar.Set(slog.LevelError)
	default:
		return fmt.Errorf("unknown log level: %s", level)
	}
	return nil
}

// Logger returns the underlying slog.Logger instance.
func Logger() *slog.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return logger
}

func setLogger(l *slog.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = l
}

// ReplaceLogger installs a custom slog.Logger.
func ReplaceLogger(l *slog.Logger) {
	if l == nil {
		panic("log: nil logger provided")
	}
	setLogger(l)
}

// Info logs a message at the info level using the global logger.
func Info(ctx context.Context, msg string, args ...any) {
	Logger().InfoContext(withContext(ctx), msg, args...)
}

// Debug logs a message at the debug level using the global logger.
func Debug(ctx context.Context, msg string, args ...any) {
	Logger().DebugContext(withContext(ctx), msg, args...)
}

// Error logs a message at the error level using the global logger.
func Error(ctx context.Context, msg string, args ...any) {
	Logger().ErrorContext(withContext(ctx), msg, args...)
}

func withContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// Sync ensures any buffered log entries are flushed. The default slog text handler
// writes directly to stdout, so Sync is a no-op but is provided for API completeness.
func Sync() error {
	type syncer interface {
		Sync() error
	}
	if s, ok := Logger().Handler().(syncer); ok {
		return s.Sync()
	}
	return nil
}
