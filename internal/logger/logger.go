package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds the logger configuration
type Config struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
	LocalTime  bool   `mapstructure:"local_time"`
	Compress   bool   `mapstructure:"compress"`
}

// New creates a new logger instance with the specified log level and format
func New(levelStr string, format string) *logrus.Logger {
	return NewWithConfig(Config{
		Level:  levelStr,
		Format: format,
	})
}

// NewWithConfig creates a new logger instance with extended configuration
func NewWithConfig(config Config) *logrus.Logger {
	log := logrus.New()

	// Parse log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		// Log to stderr before setting up the logger
		log.Errorf("Invalid log level '%s', using info level instead: %v", config.Level, err)
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set output
	var writer io.Writer = os.Stdout
	if config.Output != "" && config.Output != "stdout" {
		if config.MaxSize <= 0 {
			config.MaxSize = 100 // megabytes
		}
		if config.MaxAge <= 0 {
			config.MaxAge = 30 // days
		}
		if config.MaxBackups <= 0 {
			config.MaxBackups = 3
		}

		// Ensure log directory exists
		dir := filepath.Dir(config.Output)
		if err := os.MkdirAll(dir, 0750); err != nil {
			log.Errorf("Failed to create log directory '%s': %v", dir, err)
		}

		writer = &lumberjack.Logger{
			Filename:   config.Output,
			MaxSize:    config.MaxSize,
			MaxAge:     config.MaxAge,
			MaxBackups: config.MaxBackups,
			LocalTime:  config.LocalTime,
			Compress:   config.Compress,
		}
	}
	log.SetOutput(writer)

	// Set formatter
	switch config.Format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	return log
}
