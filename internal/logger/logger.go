// Package logger provides structured logging with file and console output.
package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog for structured logging.
type Logger struct {
	zerolog.Logger
}

// New creates a new logger with the specified level and optional file output.
func New(level string, logFile string) (*Logger, error) {
	// parse log level
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	// create writers
	writers := []io.Writer{
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"},
	}

	// add file writer if specified
	if logFile != "" {
		// create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		writers = append(writers, file)
	}

	multi := zerolog.MultiLevelWriter(writers...)

	logger := zerolog.New(multi).
		Level(lvl).
		With().
		Timestamp().
		Caller().
		Logger()

	return &Logger{logger}, nil
}

// Global is the global logger instance for convenience.
var Global *Logger

// Init initializes the global logger.
func Init(level string, logFile string) error {
	l, err := New(level, logFile)
	if err != nil {
		return err
	}
	Global = l
	return nil
}

// Get returns the global logger.
// Returns a no-op logger if not initialized.
func Get() *Logger {
	if Global == nil {
		// return a no-op logger (writes to discard)
		noop := zerolog.Nop()
		return &Logger{noop}
	}
	return Global
}

// Info logs an info message using the global logger.
func Info(msg string) {
	if Global != nil {
		Global.Info().Msg(msg)
	}
}

// Error logs an error message using the global logger.
func Error(msg string, err error) {
	if Global != nil {
		Global.Error().Err(err).Msg(msg)
	}
}

// Debug logs a debug message using the global logger.
func Debug(msg string) {
	if Global != nil {
		Global.Debug().Msg(msg)
	}
}
