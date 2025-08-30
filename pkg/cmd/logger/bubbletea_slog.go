package logger

import (
	"io"
	"log/slog"
)

type LoggerFormat string

const (
	LoggerFormatJSON LoggerFormat = "json"
	LoggerFormatText LoggerFormat = "text"
)

type StructuredBubbleTeaLogger struct {
	logger slog.Logger
	format LoggerFormat
}

// init a new structured logger that's compatible with bubbletea's
// LogOptionsSetter
func NewStructuredBubbleTeaLogger(loggerFormat LoggerFormat) StructuredBubbleTeaLogger {
	logger := slog.Default()
	slog.SetLogLoggerLevel(slog.LevelDebug)
	return StructuredBubbleTeaLogger{
		logger: *logger,
		format: loggerFormat,
	}
}

func (l StructuredBubbleTeaLogger) SetOutput(w io.Writer) {
	l.logger = *slog.New(
		slog.NewTextHandler(
			w,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			},
		),
	)
	slog.SetDefault(&l.logger)
}

func (l StructuredBubbleTeaLogger) SetPrefix(prefix string) {
	l.logger = *l.logger.With("prefix", prefix)
}
