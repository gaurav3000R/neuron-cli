package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

// ANSI color codes
const (
	reset   = "\033[0m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	gray    = "\033[90m"
)

// MultiHandler sends log records to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}

// PrettyHandler is a custom slog.Handler that outputs colorized, human-readable logs.
type PrettyHandler struct {
	w     io.Writer
	opts  slog.HandlerOptions
	attrs []slog.Attr
}

func NewPrettyHandler(w io.Writer, opts slog.HandlerOptions) *PrettyHandler {
	return &PrettyHandler{w: w, opts: opts}
}

func (h *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String()
	levelColor := reset

	// Only apply colors if writing to a terminal (or we can just force it if user wants)
	// For simplicity, we detect if the writer is a file or terminal if needed,
	// but here we just follow the "proper color code formation" request.

	switch r.Level {
	case slog.LevelDebug:
		levelColor = gray
	case slog.LevelInfo:
		levelColor = cyan
	case slog.LevelWarn:
		levelColor = yellow
	case slog.LevelError:
		levelColor = red
	}

	// Format: HH:MM:SS
	timeStr := r.Time.Format("15:04:05")

	// Print Time
	fmt.Fprintf(h.w, "%s%s%s ", gray, timeStr, reset)

	// Print Level
	fmt.Fprintf(h.w, "%s%-5s%s ", levelColor, level, reset)

	// Source info (file:line)
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		source := fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
		fmt.Fprintf(h.w, "%s[%s]%s ", dim, source, reset)
	}

	// Message
	fmt.Fprintf(h.w, "%s ", r.Message)

	// Attributes
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(h.w, " %s%s=%v%s", magenta, a.Key, a.Value.Any(), reset)
		return true
	})

	fmt.Fprintln(h.w)
	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyHandler{w: h.w, opts: h.opts, attrs: append(h.attrs, attrs...)}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h
}

// SetupMulti initializes the global slog logger with multiple outputs (JSON and Pretty).
func SetupMulti(verbose bool, jsonPath string, prettyPath string, console io.Writer) error {
	var handlers []slog.Handler

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	// 1. JSON File
	if jsonPath != "" {
		if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err == nil {
			if f, err := os.OpenFile(jsonPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				handlers = append(handlers, slog.NewJSONHandler(f, &opts))
			}
		}
	}

	// 2. Pretty File (Optional, but good for keeping a colorized history)
	if prettyPath != "" {
		if err := os.MkdirAll(filepath.Dir(prettyPath), 0755); err == nil {
			if f, err := os.OpenFile(prettyPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				handlers = append(handlers, NewPrettyHandler(f, opts))
			}
		}
	}

	// 3. Console (Always Pretty)
	if console != nil {
		handlers = append(handlers, NewPrettyHandler(console, opts))
	}

	logger := slog.New(&MultiHandler{handlers: handlers})
	slog.SetDefault(logger)

	return nil
}
