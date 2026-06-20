package clog

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/onceprgm/cme/internal/store"
)

var (
	fileLogger = slog.New(discardHandler{})
	logger     = slog.New(discardHandler{})
)

func Setup(verbose bool) (func() error, error) {
	dir := store.StateDir()
	if err := store.Ensure(dir); err != nil {
		return nil, err
	}
	f, err := os.Create(filepath.Join(dir, "cme.log"))
	if err != nil {
		return nil, err
	}

	fileHandler := slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug})
	fileLogger = slog.New(fileHandler)

	if verbose {
		stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger = slog.New(fanout{fileHandler, stderrHandler})
	} else {
		logger = fileLogger
	}

	return f.Close, nil
}

func Debug(msg string, args ...any) { logger.Debug(msg, args...) }
func Info(msg string, args ...any)  { logger.Info(msg, args...) }
func Warn(msg string, args ...any)  { logger.Warn(msg, args...) }
func Error(msg string, args ...any) { logger.Error(msg, args...) }

func Mirror(level slog.Level, msg string) {
	fileLogger.Log(context.Background(), level, msg)
}

type fanout []slog.Handler

func (f fanout) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range f {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (f fanout) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range f {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (f fanout) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make(fanout, len(f))
	for i, h := range f {
		out[i] = h.WithAttrs(attrs)
	}
	return out
}

func (f fanout) WithGroup(name string) slog.Handler {
	out := make(fanout, len(f))
	for i, h := range f {
		out[i] = h.WithGroup(name)
	}
	return out
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }
