package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger
type Logger struct {
	logger  zerolog.Logger
	logFile *os.File // Keep reference to close on cleanup
}

// New creates a new logger instance
func New(level, format, logFilePath string) *Logger {
	// Set log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// Open log file if path is provided
	var logFile *os.File
	if logFilePath != "" {
		logFile = openLogFile(logFilePath)
	}

	// Create output writer
	output := createOutputWriter(format, logFile)

	// Create logger
	logger := zerolog.New(output).With().Timestamp().Logger()

	return &Logger{
		logger:  logger,
		logFile: logFile,
	}
}

// openLogFile creates log directory and opens the log file
func openLogFile(logFilePath string) *os.File {
	// Create directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If directory creation fails, log to stdout
		tempLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
		tempLogger.Warn().Err(err).Msgf("Failed to create log directory %s, using stdout only", logDir)
		return nil
	}

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// If file opening fails, log to stdout
		tempLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
		tempLogger.Warn().Err(err).Msgf("Failed to open log file %s, using stdout only", logFilePath)
		return nil
	}

	return logFile
}

// createOutputWriter creates the appropriate output writer based on format and log file
func createOutputWriter(format string, logFile *os.File) io.Writer {
	// Determine writers
	var fileWriter, consoleWriter io.Writer

	if logFile != nil {
		fileWriter = logFile
	}

	if format == "text" {
		consoleWriter = zerolog.ConsoleWriter{Out: os.Stdout}
	} else {
		consoleWriter = os.Stdout
	}

	// Combine writers
	if fileWriter != nil {
		return io.MultiWriter(fileWriter, consoleWriter)
	}

	return consoleWriter
}

// Close closes the log file if it was opened
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs an info message with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error) {
	l.logger.Error().Err(err).Msg(msg)
}

// Errorf logs an error message with formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs a debug message with formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs a warning message with formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error) {
	l.logger.Fatal().Err(err).Msg(msg)
}

// With creates a child logger with additional fields
func (l *Logger) With(key string, value interface{}) *Logger {
	return &Logger{
		logger: l.logger.With().Interface(key, value).Logger(),
	}
}
