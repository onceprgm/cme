package ui

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/onceprgm/cme/internal/clog"
)

var colorEnabled = detectColor()

func detectColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return IsTerminal()
}

func IsTerminal() bool {
	info, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

const (
	reset  = "\033[0m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	dim    = "\033[2m"
	bold   = "\033[1m"
)

func paint(code, s string) string {
	if !colorEnabled {
		return s
	}
	return code + s + reset
}

func Green(s string) string  { return paint(green, s) }
func Red(s string) string    { return paint(red, s) }
func Yellow(s string) string { return paint(yellow, s) }
func Dim(s string) string    { return paint(dim, s) }
func Bold(s string) string   { return paint(bold, s) }

func Success(format string, a ...any) {
	fmt.Fprintf(os.Stderr, paint(green, "✓")+" "+format+"\n", a...)
	clog.Mirror(slog.LevelInfo, fmt.Sprintf(format, a...))
}

func Info(format string, a ...any) {
	fmt.Fprintf(os.Stderr, paint(dim, "→")+" "+format+"\n", a...)
	clog.Mirror(slog.LevelInfo, fmt.Sprintf(format, a...))
}

func Warn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, paint(yellow, "!")+" "+format+"\n", a...)
	clog.Mirror(slog.LevelWarn, fmt.Sprintf(format, a...))
}

const barWidth = 24

func Progress(label string, done, total int) {
	if total <= 0 {
		return
	}
	if done == total {
		clog.Mirror(slog.LevelInfo, fmt.Sprintf("%s %d/%d", label, done, total))
	}
	if !colorEnabled {
		if done == total {
			fmt.Fprintf(os.Stderr, "%s %d/%d\n", label, done, total)
		}
		return
	}
	frac := float64(done) / float64(total)
	filled := int(frac * barWidth)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Fprintf(os.Stderr, "\r%-10s %s %3.0f%% (%d/%d)", label, paint(green, bar), frac*100, done, total)
	if done == total {
		fmt.Fprint(os.Stderr, "\n")
	}
}
