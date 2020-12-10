package log

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

type Logger struct {
	outputLevel uint
}

type LogWriter struct {
	Info func(format string, a ...interface{})
}

type contextKey string

var (
	consoleWriter              = LogWriter{Info: info}
	noneWriter                 = LogWriter{Info: func(format string, a ...interface{}) {}}
	logLevelKey     contextKey = "ll"
	defaultLogLevel uint       = 2
)

// FromContext returns logger from context
func FromContext(ctx context.Context) *Logger {
	logLevel := defaultLogLevel
	if l := ctx.Value(logLevelKey); l != nil {
		if v, ok := l.(uint); ok {
			logLevel = v
		}
	}

	return &Logger{outputLevel: logLevel}
}

// WithLogLevel create a context with logging level
func WithLogLevel(ctx context.Context, logLevel uint) context.Context {
	return context.WithValue(ctx, logLevelKey, logLevel)
}

// V returns log writter at log level
func (l *Logger) V(level uint) *LogWriter {
	if level <= l.outputLevel {
		return &consoleWriter
	}
	return &noneWriter
}

func info(format string, a ...interface{}) {
	l := fmt.Sprintf("[%s] %s\n", time.Now().UTC().Format(time.RFC3339), fmt.Sprintf(format, a...))

	failed := color.RedString("FAILED")
	succeeded := color.GreenString("SUCCEEDED")

	l = strings.ReplaceAll(l, "SUCCEEDED", succeeded)
	l = strings.ReplaceAll(l, "FAILED", failed)

	fmt.Print(l)
}
