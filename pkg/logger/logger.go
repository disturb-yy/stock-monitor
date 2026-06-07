package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LogTimeFormat = "2006-01-02 15:04:05.000"
	FormatJSON    = "json"
	FormatText    = "text"
)

var std *slog.Logger

type Config struct {
	Level      string
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	Console    bool
	Format     string
}

type ctxKey struct{}

func Init(cfg Config) {
	level := parseLogLevel(cfg.Level)
	output := newOutputWriter(newFileWriter(cfg), cfg.Console)
	handler := newLogHandler(level, output, cfg.Format)

	std = slog.New(handler)
	slog.SetDefault(std)
}

func WithContext(ctx context.Context, args ...any) context.Context {
	if std == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, std.With(args...))
}

func Info(ctx context.Context, msg string, args ...any) {
	fromCtx(ctx).InfoContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	fromCtx(ctx).ErrorContext(ctx, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	fromCtx(ctx).DebugContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	fromCtx(ctx).WarnContext(ctx, msg, args...)
}

func Fatal(msg string, args ...any) {
	fromCtx(context.Background()).Error("fatal error", append([]any{"msg", msg}, args...)...)
	os.Exit(1)
}

func Sync() {}

func fromCtx(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
			return logger
		}
	}
	if std != nil {
		return std
	}
	return slog.Default()
}

func newFileWriter(cfg Config) io.Writer {
	if cfg.Filename == "" {
		return io.Discard
	}

	if dir := filepath.Dir(cfg.Filename); dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
}

func newOutputWriter(fileWriter io.Writer, console bool) io.Writer {
	if console {
		return io.MultiWriter(fileWriter, os.Stdout)
	}
	return fileWriter
}

func newLogHandler(level slog.Level, writer io.Writer, format string) slog.Handler {
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.String("time", attr.Value.Time().Format(LogTimeFormat))
			}
			return attr
		},
	}

	if strings.EqualFold(format, FormatJSON) {
		return slog.NewJSONHandler(writer, opts)
	}
	return slog.NewTextHandler(writer, opts)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
